package optimizer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// StructDef 结构体定义（用于文件扫描）
type StructDef struct {
	Name    string
	PkgPath string
	File    string
	Type    *types.Struct
}

// findStructInPackage 在已加载的包中查找结构体（优化阶段使用）
func (o *Optimizer) findStructInPackage(pkg *packages.Package, structName string) (*types.Struct, string, error) {
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		for _, decl := range syntax.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if typeSpec.Name.Name != structName {
					continue
				}

				obj := pkg.TypesInfo.ObjectOf(typeSpec.Name)
				if obj == nil {
					continue
				}

				if named, ok := obj.Type().(*types.Named); ok {
					if st, ok := named.Underlying().(*types.Struct); ok {
						return st, filePath, nil
					}
				}
			}
		}
	}

	return nil, "", fmt.Errorf("struct %s not found in package", structName)
}

// NewOptimizer 创建优化器
func NewOptimizer(cfg *Config, analyzer *analyzer.Analyzer) *Optimizer {
	// 设置默认值
	maxDepth := cfg.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 50 // 默认 50 层
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 1200 // 默认 20 分钟
	}

	pkgWorkerLimit := cfg.PkgWorkerLimit
	if pkgWorkerLimit <= 0 {
		pkgWorkerLimit = 4 // 默认 4 个并发包，防止 OOM
	}

	return &Optimizer{
		config:           cfg,
		analyzer:         analyzer,
		optimized:        make(map[string]*StructInfo),
		processing:       make(map[string]bool),
		collecting:       make(map[string]bool),
		maxDepth:         maxDepth,
		methodIndex:      NewMethodIndex(),
		structQueue:      make([]*StructTask, 0),
		structByLevel:    make(map[int][]*StructTask),
		structByPkgLevel: make(map[int]map[string][]*StructTask),
		workerLimit:      10,  // 结构体并发限制
		pkgWorkerLimit:   pkgWorkerLimit,  // 包并发限制（防止 OOM）
		pkgCache:         make(map[string]*packages.Package),
		pkgFileCache:     NewPackageCache("", true),  // 启用文件缓存
		structCache:      make(map[string]*types.Struct),
		filePathCache:    make(map[string]string),
		memGuard:         NewMemoryGuard(512, true),  // 默认 512MB，启用自动 GC
		report: &Report{
			StructReports: make([]*StructReport, 0),
		},
	}
}

// Optimize 执行优化（入口函数）
// 两阶段处理：
//   阶段 1: 只收集结构体位置信息（不加载包，不分析字段）
//   阶段 2: 并行优化所有收集的结构体（按需加载包）
func (o *Optimizer) Optimize() (*Report, error) {
	o.Log(1, "开始优化...")
	o.Log(1, "配置：最大深度=%d, 超时=%d 秒", o.maxDepth, o.config.Timeout)

	// 设置超时
	done := make(chan struct{})
	var result *Report
	var err error

	go func() {
		defer close(done)
		result, err = o.optimizeInternal()
	}()

	// 等待完成或超时
	select {
	case <-done:
		return result, err
	case <-time.After(time.Duration(o.config.Timeout) * time.Second):
		o.Log(0, "错误：优化超时（%d 秒）", o.config.Timeout)
		return nil, fmt.Errorf("optimization timeout after %d seconds", o.config.Timeout)
	}
}

// optimizeInternal 实际优化逻辑（两阶段）
func (o *Optimizer) optimizeInternal() (*Report, error) {
	// ==================== 阶段 1: 收集结构体 ====================
	o.Log(1, "阶段 1/2: 收集结构体（只解析文件，不加载包）...")
	if o.config.StructName != "" {
		// 优化指定结构体
		pkgPath, structName := analyzer.ParseStructName(o.config.StructName)
		if pkgPath == "" {
			return nil, fmt.Errorf("invalid struct name format: %s", o.config.StructName)
		}

		// 设置主结构体
		o.report.RootStruct = o.config.StructName

		o.Log(1, "收集结构体：%s.%s", pkgPath, structName)
		o.collectStructs(pkgPath, structName, "", 0, 0)
	} else if o.config.Package != "" {
		// 优化包中所有结构体
		o.Log(1, "收集包：%s", o.config.Package)
		
		// GOPATH 模式下，使用文件扫描而不是包加载
		if o.config.ProjectType == "gopath" {
			structs, err := o.findAllStructsByScanning(o.config.Package)
			if err != nil {
				return nil, err
			}
			for _, st := range structs {
				o.collectStructs(st.PkgPath, st.Name, st.File, 0, 0)
			}
		} else {
			// Go Module 模式下，使用原来的方法
			structs, err := o.analyzer.FindAllStructs(o.config.Package)
			if err != nil {
				return nil, err
			}
			for _, st := range structs {
				o.collectStructs(st.PkgPath, st.Name, st.File, 0, 0)
			}
		}
	}

	o.Log(1, "阶段 1 完成：共收集到 %d 个结构体任务", len(o.structQueue))

	// 打印收集到的结构体列表
	o.Log(2, "收集到的结构体列表:")
	for i, task := range o.structQueue {
		o.Log(3, "  [%d] %s.%s (层级:%d, 文件:%s)",
			i+1, task.PkgPath, task.StructName, task.Level, filepath.Base(task.FilePath))
	}

	// ==================== 阶段 2: 并行优化 ====================
	o.Log(1, "阶段 2/2: 并行优化结构体（按需加载包）...")
	o.processStructsParallel()

	// 生成报告
	o.report.TotalStructs = len(o.optimized)
	o.report.OptimizedCount = 0
	o.report.SkippedCount = 0
	o.report.RootStructSize = 0
	o.report.RootStructOptSize = 0

	for _, info := range o.optimized {
		if info.Skipped {
			o.report.SkippedCount++
		} else if info.OrigSize > info.OptSize {
			// 只有真正节省了内存才算优化
			o.report.OptimizedCount++
			o.report.TotalSaved += info.OrigSize - info.OptSize
		}
		
		// 累计所有结构体的大小（用于总览等式）
		o.report.RootStructSize += info.OrigSize
		o.report.RootStructOptSize += info.OptSize
	}

	o.Log(1, "优化完成：共处理 %d 个结构体，优化 %d 个，跳过 %d 个，节省 %d 字节",
		o.report.TotalStructs, o.report.OptimizedCount, o.report.SkippedCount, o.report.TotalSaved)

	return o.report, nil
}

// optimizeStruct 优化单个结构体（递归）
func (o *Optimizer) optimizeStruct(pkgPath, structName, filePath string, depth int) (*StructInfo, error) {
	key := pkgPath + "." + structName

	// 检查递归深度限制
	if depth > o.maxDepth {
		o.Log(2, "跳过结构体（超过最大深度 %d）：%s", o.maxDepth, key)
		info := &StructInfo{
			Name:       structName,
			PkgPath:    pkgPath,
			File:       filePath,
			Skipped:    true,
			SkipReason: fmt.Sprintf("超过最大递归深度 (%d)", o.maxDepth),
		}
		o.optimized[key] = info
		o.addReport(info, info.SkipReason, depth)
		return info, nil
	}

	// 检查是否已优化（加锁保护）
	o.mu.Lock()
	if info, ok := o.optimized[key]; ok {
		o.mu.Unlock()
		o.Log(3, "结构体已处理：%s", key)
		return info, nil
	}

	// 检测循环引用：如果正在处理中，说明有循环引用
	if o.processing[key] {
		o.mu.Unlock()
		o.Log(2, "检测到循环引用，跳过：%s", key)
		info := &StructInfo{
			Name:       structName,
			PkgPath:    pkgPath,
			File:       filePath,
			Skipped:    true,
			SkipReason: "循环引用",
		}
		o.optimized[key] = info
		o.addReport(info, "循环引用", depth)
		return info, nil
	}

	// 标记为正在处理
	o.processing[key] = true
	o.mu.Unlock()

	defer func() {
		// 处理完成后，移除标记
		o.mu.Lock()
		delete(o.processing, key)
		o.mu.Unlock()
	}()

	// 检查是否是 vendor 中的包或第三方包，如果是则跳过
	if isVendorPackage(pkgPath) {
		o.Log(3, "跳过 vendor 中的结构体：%s", key)
		info := &StructInfo{
			Name:       structName,
			PkgPath:    pkgPath,
			File:       filePath,
			Skipped:    true,
			SkipReason: "vendor 中的第三方包结构体",
		}
		o.mu.Lock()
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, "vendor 中的第三方包结构体", depth)
		return info, nil
	}

	// 检查是否是项目内部的包
	if !o.isProjectPackage(pkgPath) {
		o.Log(3, "跳过非项目内部包结构体：%s", key)
		info := &StructInfo{
			Name:       structName,
			PkgPath:    pkgPath,
			File:       filePath,
			Skipped:    true,
			SkipReason: func() string {
				if isStandardLibrary(pkgPath) {
					return "Go 标准库结构体"
				}
				return "非项目内部包结构体"
			}(),
		}
		o.mu.Lock()
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, func() string {
			if isStandardLibrary(pkgPath) {
				return "Go 标准库结构体"
			}
			return "非项目内部包结构体"
		}(), depth)
		return info, nil
	}

	o.Log(2, "[%d] 处理结构体：%s", depth, key)
	if filePath != "" {
		o.Log(3, "    文件路径：%s", filepath.Base(filePath))
	}

	// 优化阶段：优先解析文件，不加载包
	// 只有文件解析失败时才加载包
	var info *StructInfo
	var err error
	var st *types.Struct

	// 尝试只解析文件（快速路径）
	if filePath != "" {
		info, st, err = analyzeStructFromFile(filePath, structName, pkgPath)
		if err != nil {
			o.Log(3, "文件解析失败，加载包：%v", err)
		} else {
			// 快速路径成功，检查是否有未知类型字段
			hasUnknownType := false
			for _, f := range info.Fields {
				// 如果不是基本类型，说明可能是重定义类型或结构体
				if !isBasicType(f.TypeName) {
					hasUnknownType = true
					break
				}
			}
			
			if hasUnknownType {
				// 有未知类型，需要加载包获取准确大小
				// 但使用缓存避免重复加载
				o.Log(3, "检测到重定义类型或结构体字段，加载包获取准确大小（使用缓存）")

				// 加载包（使用缓存）
				pkg, pkgErr := o.loadPackageCached(pkgPath)
				if pkgErr != nil {
					o.Log(3, "加载包失败，使用估算值：%v", pkgErr)
					// 加载失败，继续使用估算值
				} else {
					// 包加载成功，查找结构体并重新分析字段
					st2, _, err := o.findStructInPackage(pkg, structName)
					if err != nil {
						o.Log(3, "查找结构体失败，使用估算值：%v", err)
					} else {
						// 使用完整的类型信息重新分析
						fa := NewFieldAnalyzer(pkg.TypesInfo, pkg.Fset)
						info2 := fa.AnalyzeStruct(st2, structName, pkgPath, filePath)

						// 更新 info 中的字段和大小信息
						info.Fields = info2.Fields
						info.OrigSize = info2.OrigSize
						o.fieldAnalyzer = fa
					}
				}
			}
		}
	}

	// 文件解析失败，加载包（慢速路径）
	if info == nil {
		o.Log(2, "加载包获取类型信息：%s", pkgPath)
		pkg, err := o.loadPackageCached(pkgPath)
		if err != nil {
			o.Log(1, "警告：加载包失败，跳过：%s (%v)", key, err)
			info = &StructInfo{
				Name:       structName,
				PkgPath:    pkgPath,
				File:       filePath,
				Skipped:    true,
				SkipReason: "加载包失败：" + err.Error(),
			}
			o.mu.Lock()
			o.optimized[key] = info
			o.mu.Unlock()
			o.addReport(info, info.SkipReason, depth)
			return info, nil
		}

		// 在包中查找结构体
		st, filePath, err = o.findStructInPackage(pkg, structName)
		if err != nil {
			o.Log(1, "警告：查找结构体失败，跳过：%s (%v)", key, err)
			info = &StructInfo{
				Name:       structName,
				PkgPath:    pkgPath,
				File:       filePath,
				Skipped:    true,
				SkipReason: "查找失败：" + err.Error(),
			}
			o.mu.Lock()
			o.optimized[key] = info
			o.mu.Unlock()
			o.addReport(info, info.SkipReason, depth)
			return info, nil
		}

		// 创建字段分析器（使用完整的类型信息）
		o.fieldAnalyzer = NewFieldAnalyzer(pkg.TypesInfo, pkg.Fset)

		// 分析结构体
		info = o.fieldAnalyzer.AnalyzeStruct(st, structName, pkgPath, filePath)
	}

	// 检查是否应该跳过
	if skipReason := o.shouldSkip(info, st, key); skipReason != "" {
		o.Log(2, "跳过结构体：%s, 原因：%s", key, skipReason)
		info.Skipped = true
		info.SkipReason = skipReason
		info.OptSize = info.OrigSize // 跳过的结构体，优化后大小等于优化前大小
		o.mu.Lock()
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, skipReason, depth)
		return info, nil
	}

	// 重排字段（嵌套结构体已在收集阶段处理）
	// 注意：ReorderFields 内部使用估计大小判断是否重排，可能不准确
	// 我们总是获取排序结果，然后在下面用准确大小判断是否采用
	sortedFields := ReorderFields(info.Fields, o.config.SortSameSize, o.config.ReservedFields)

	// 计算排序后的大小（使用准确的类型信息）
	sortedOptSize := CalcOptimizedSize(sortedFields, o.analyzer.GetTypesInfo())

	// 判断是否采用排序结果：只有能节省内存时才采用
	if sortedOptSize < info.OrigSize {
		info.Fields = sortedFields
		info.OptSize = sortedOptSize
		info.OptOrder = extractFieldNamesFromInfo(sortedFields)
		info.Optimized = true

		o.Log(2, "结构体优化：%s %d -> %d 字节 (节省:%d)",
			key, info.OrigSize, info.OptSize, info.OrigSize-info.OptSize)
	} else {
		// 无法节省内存，不采用新顺序，保持原样
		info.OptSize = info.OrigSize
		info.OptOrder = info.OrigOrder
		// info.Optimized 保持为 false，不会触发文件重写
		o.Log(2, "结构体无需优化：%s", key)
	}

	o.mu.Lock()
	o.optimized[key] = info
	o.mu.Unlock()
	o.addReport(info, "", depth)

	return info, nil
}

// loadPackageCached 惰性加载包，使用缓存避免重复加载
func (o *Optimizer) loadPackageCached(pkgPath string) (*packages.Package, error) {
	// 检查内存缓存
	o.mu.Lock()
	if pkg, ok := o.pkgCache[pkgPath]; ok {
		o.mu.Unlock()
		o.Log(3, "使用内存缓存的包：%s", pkgPath)
		return pkg, nil
	}
	o.mu.Unlock()

	// 检查文件缓存
	if o.pkgFileCache != nil {
		// 获取包的 Go 文件列表
		goFiles := o.getPackageGoFiles(pkgPath)
		if len(goFiles) > 0 {
			cacheEntry, err := o.pkgFileCache.LoadPackageCache(pkgPath, goFiles)
			if err != nil {
				o.Log(3, "加载文件缓存失败：%v", err)
			} else if cacheEntry != nil {
				o.Log(2, "使用文件缓存的包：%s (%d 个结构体)", pkgPath, len(cacheEntry.Structs))
				// TODO: 从缓存恢复包信息
				// 这需要修改 LoadPackage 以支持从缓存创建
			}
		}
	}

	// 加载前内存检查
	if o.memGuard != nil {
		if err := o.memGuard.CheckMemory(); err != nil {
			return nil, fmt.Errorf("加载包前内存检查失败：%s (%v)", pkgPath, err)
		}
	}

	// 加载包
	o.Log(3, "加载包（未缓存）：%s", pkgPath)
	pkg, err := o.analyzer.LoadPackage(pkgPath)
	if err != nil {
		return nil, err
	}

	// 加载后内存检查
	if o.memGuard != nil {
		if err := o.memGuard.CheckMemory(); err != nil {
			o.Log(2, "警告：加载包后内存使用过高：%s (%v)", pkgPath, err)
			// 不返回错误，让程序继续运行
		}
	}

	// 缓存到内存
	o.mu.Lock()
	if o.pkgCache == nil {
		o.pkgCache = make(map[string]*packages.Package)
	}
	o.pkgCache[pkgPath] = pkg
	o.mu.Unlock()

	// 保存到文件缓存（异步）
	if o.pkgFileCache != nil {
		go o.savePackageCache(pkgPath, pkg)
	}

	return pkg, nil
}

// getPackageGoFiles 获取包中的 Go 文件列表
func (o *Optimizer) getPackageGoFiles(pkgPath string) []string {
	pkgDir := o.getPackageDir(pkgPath)
	if pkgDir == "" {
		return nil
	}

	var goFiles []string
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			goFiles = append(goFiles, filepath.Join(pkgDir, entry.Name()))
		}
	}

	return goFiles
}

// savePackageCache 保存包缓存到文件
func (o *Optimizer) savePackageCache(pkgPath string, pkg *packages.Package) {
	if o.pkgFileCache == nil || pkg == nil {
		return
	}

	// 收集结构体信息
	var structs []StructCacheInfo
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()
		
		for _, decl := range syntax.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}

				// 提取字段信息
				var fields []FieldCacheInfo
				for _, f := range st.Fields.List {
					fieldName := ""
					if len(f.Names) > 0 {
						fieldName = f.Names[0].Name
					}
					
					typeName := extractTypeNameFromAST(f.Type)
					size, align := estimateFieldSize(f.Type)
					
					tag := ""
					if f.Tag != nil {
						tag = strings.Trim(f.Tag.Value, "`")
					}

					fields = append(fields, FieldCacheInfo{
						Name:     fieldName,
						TypeName: typeName,
						Size:     size,
						Align:    align,
						IsEmbed:  len(f.Names) == 0,
						Tag:      tag,
					})
				}

				structs = append(structs, StructCacheInfo{
					Name:     ts.Name.Name,
					FilePath: filePath,
					Fields:   fields,
				})
			}
		}
	}

	// 获取 Go 文件列表
	goFiles := o.getPackageGoFiles(pkgPath)

	// 计算哈希
	contentHash, _ := o.pkgFileCache.computePackageHash(goFiles)
	goModHash := o.pkgFileCache.computeGoModHash()

	// 保存缓存
	entry := &CacheEntry{
		Hash:      contentHash,
		PkgPath:   pkgPath,
		Structs:   structs,
		GoModHash: goModHash,
		GoFiles:   goFiles,
		CreatedAt: time.Now().Unix(),
	}

	if err := o.pkgFileCache.SavePackageCache(entry); err != nil {
		o.Log(2, "保存包缓存失败：%s (%v)", pkgPath, err)
	} else {
		o.Log(3, "已保存包缓存：%s (%d 个结构体)", pkgPath, len(structs))
	}
}

// extractTypeNameFromAST 从 AST 表达式中提取类型名称
func extractTypeNameFromAST(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + extractTypeNameFromAST(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.ArrayType:
		if t.Len != nil {
			return "[" + fmt.Sprintf("%v", t.Len) + "]" + extractTypeNameFromAST(t.Elt)
		}
		return "[]" + extractTypeNameFromAST(t.Elt)
	case *ast.MapType:
		return "map[" + extractTypeNameFromAST(t.Key) + "]" + extractTypeNameFromAST(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return ""
	}
}

// createSkippedInfo 创建跳过的结构体信息
func (o *Optimizer) createSkippedInfo(key, filePath, reason string) *StructInfo {
	pkgPath, structName := analyzer.ParseStructName(key)
	return &StructInfo{
		Name:       structName,
		PkgPath:    pkgPath,
		File:       filePath,
		Skipped:    true,
		SkipReason: reason,
	}
}

// addReport 添加报告
func (o *Optimizer) addReport(info *StructInfo, skipReason string, depth int) {
	// 构建字段类型和大小映射
	fieldTypes := make(map[string]string)
	fieldSizes := make(map[string]int64)
	hasEmbed := false
	for _, f := range info.Fields {
		key := f.Name
		if key == "" {
			key = f.TypeName // 匿名字段使用类型名作为 key
		}
		fieldTypes[key] = f.TypeName
		fieldSizes[key] = f.Size

		// 检查是否是匿名字段
		// 判断条件：字段名等于类型名，且类型是结构体类型（非基本类型）
		if f.Name == f.TypeName && !isBasicType(f.TypeName) {
			hasEmbed = true
		}
	}

	report := &StructReport{
		Name:       info.Name,
		PkgPath:    info.PkgPath,
		File:       info.File,
		OrigSize:   info.OrigSize,
		OptSize:    info.OptSize,
		Saved:      info.OrigSize - info.OptSize,
		OrigFields: info.OrigOrder,
		OptFields:  info.OptOrder,
		FieldTypes: fieldTypes,
		FieldSizes: fieldSizes,
		Skipped:    info.Skipped,
		SkipReason: skipReason,
		Depth:      depth,
		HasEmbed:   hasEmbed,
	}

	if info.OptSize == 0 && info.OrigSize == 0 {
		report.Saved = 0
	}

	o.report.StructReports = append(o.report.StructReports, report)
}

// GetOptimized 获取已优化的结构体信息
func (o *Optimizer) GetOptimized() map[string]*StructInfo {
	return o.optimized
}

// GetReport 获取报告
func (o *Optimizer) GetReport() *Report {
	return o.report
}

// Log 日志输出
func (o *Optimizer) Log(level int, format string, args ...interface{}) {
	if level <= o.config.Verbose {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		levelPrefix := ""
		switch level {
		case 1:
			levelPrefix = "[INFO] "
		case 2:
			levelPrefix = "[DEBUG]"
		case 3:
			levelPrefix = "[TRACE]"
		}
		fmt.Printf("%s %s "+format+"\n", append([]interface{}{timestamp, levelPrefix}, args...)...)
	}
}

// isBasicType 判断是否是基本类型
func isBasicType(typeName string) bool {
	basicTypes := map[string]bool{
		"bool": true, "string": true,
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
		"byte": true, "rune": true, "uintptr": true,
	}
	return basicTypes[typeName]
}

// findAllStructsByScanning 通过扫描文件查找包中的所有结构体（GOPATH 模式使用）
func (o *Optimizer) findAllStructsByScanning(pkgPath string) ([]StructDef, error) {
	// 获取包目录
	pkgDir := o.getPackageDir(pkgPath)
	if pkgDir == "" {
		return nil, fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	o.Log(2, "扫描包目录：%s", pkgDir)

	var structs []StructDef

	// 扫描目录中的所有 Go 文件
	err := o.scanDirForStructs(pkgDir, pkgPath, &structs)
	if err != nil {
		return nil, err
	}

	return structs, nil
}

// scanDirForStructs 递归扫描目录查找结构体
func (o *Optimizer) scanDirForStructs(dir, pkgPath string, structs *[]StructDef) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// 跳过 vendor 和.git 等目录
			name := entry.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" {
				continue
			}
			// 递归扫描子目录
			if err := o.scanDirForStructs(filepath.Join(dir, entry.Name()), pkgPath, structs); err != nil {
				return err
			}
			continue
		}

		// 只处理.go 文件
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		filePath := filepath.Join(dir, name)

		// 检查是否应该跳过
		if o.shouldSkipFile(filePath) {
			continue
		}

		// 解析文件查找结构体
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		// 提取结构体
		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if _, ok := ts.Type.(*ast.StructType); ok {
					*structs = append(*structs, StructDef{
						Name:    ts.Name.Name,
						PkgPath: pkgPath,
						File:    filePath,
						Type:    nil, // 文件扫描时无法获取类型信息
					})
				}
			}
		}
	}

	return nil
}

// extractFieldNamesFromInfo 从 FieldInfo 列表提取字段名称
func extractFieldNamesFromInfo(fields []FieldInfo) []string {
	var names []string
	for _, f := range fields {
		if f.Name != "" {
			names = append(names, f.Name)
		} else {
			// 匿名字段使用类型名
			names = append(names, f.TypeName)
		}
	}
	return names
}
