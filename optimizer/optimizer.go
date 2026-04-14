package optimizer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

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
		timeout = 300 // 默认 300 秒
	}

	return &Optimizer{
		config:           cfg,
		analyzer:         analyzer,
		optimized:        make(map[string]*StructInfo),
		processing:       make(map[string]bool),
		collecting:       make(map[string]bool),
		maxDepth:         maxDepth,
		structQueue:      make([]*StructTask, 0),
		structByLevel:    make(map[int][]*StructTask),
		structByPkgLevel: make(map[int]map[string][]*StructTask),
		workerLimit:      10, // 最多 10 个并发协程
		pkgCache:         make(map[string]*packages.Package),
		structCache:      make(map[string]*types.Struct),
		filePathCache:    make(map[string]string),
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

		o.Log(1, "收集结构体：%s.%s", pkgPath, structName)
		o.collectStructs(pkgPath, structName, "", 0, 0)
	} else if o.config.Package != "" {
		// 优化包中所有结构体
		o.Log(1, "收集包：%s", o.config.Package)
		structs, err := o.analyzer.FindAllStructs(o.config.Package)
		if err != nil {
			return nil, err
		}

		for _, st := range structs {
			o.collectStructs(st.PkgPath, st.Name, st.File, 0, 0)
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

	for _, info := range o.optimized {
		if info.Skipped {
			o.report.SkippedCount++
		} else if info.Optimized {
			o.report.OptimizedCount++
			o.report.TotalSaved += info.OrigSize - info.OptSize
		}
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

	// 加载包获取完整类型信息（用于优化阶段）
	pkg, err := o.analyzer.LoadPackage(pkgPath)
	if err != nil {
		o.Log(1, "警告：加载包失败，跳过：%s (%v)", key, err)
		info := &StructInfo{
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
	st, filePath, err := o.findStructInPackage(pkg, structName)
	if err != nil {
		o.Log(1, "警告：查找结构体失败，跳过：%s (%v)", key, err)
		info := &StructInfo{
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
	info := o.fieldAnalyzer.AnalyzeStruct(st, structName, pkgPath, filePath)

	// 检查是否应该跳过
	if skipReason := o.shouldSkip(info, st, key); skipReason != "" {
		o.Log(2, "跳过结构体：%s, 原因：%s", key, skipReason)
		info.Skipped = true
		info.SkipReason = skipReason
		o.mu.Lock()
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, skipReason, depth)
		return info, nil
	}

	// 重排字段（嵌套结构体已在收集阶段处理）
	optimizedFields := ReorderFields(info.Fields, o.config.SortSameSize)
	info.Fields = optimizedFields

	// 计算优化后大小
	info.OptSize = CalcOptimizedSize(optimizedFields, o.analyzer.GetTypesInfo())

	// 生成优化后的字段顺序
	var optOrder []string
	for _, f := range optimizedFields {
		if f.Name != "" {
			optOrder = append(optOrder, f.Name)
		} else {
			optOrder = append(optOrder, f.TypeName)
		}
	}
	info.OptOrder = optOrder

	// 检查是否真正优化了
	if info.OrigSize != info.OptSize || !o.fieldOrderSame(info.OrigOrder, info.OptOrder) {
		info.Optimized = true
		o.Log(2, "结构体优化：%s %d -> %d 字节 (节省:%d)",
			key, info.OrigSize, info.OptSize, info.OrigSize-info.OptSize)
	} else {
		o.Log(2, "结构体无需优化：%s", key)
	}

	o.mu.Lock()
	o.optimized[key] = info
	o.mu.Unlock()
	o.addReport(info, "", depth)

	return info, nil
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
	// 构建字段类型映射
	fieldTypes := make(map[string]string)
	for _, f := range info.Fields {
		key := f.Name
		if key == "" {
			key = f.TypeName // 匿名字段使用类型名
		}
		fieldTypes[key] = f.TypeName
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
		Skipped:    info.Skipped,
		SkipReason: skipReason,
		Depth:      depth,
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
