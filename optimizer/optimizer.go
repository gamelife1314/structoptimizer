package optimizer

import (
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// Optimizer 优化器
type Optimizer struct {
	config      *Config
	analyzer    *analyzer.Analyzer
	optimized   map[string]*StructInfo // 已优化的结构体（key: pkgPath.structName）
	report      *Report
	fieldAnalyzer *FieldAnalyzer
	processing  map[string]bool // 正在处理中的结构体（用于检测循环引用）
	maxDepth    int             // 最大递归深度
}

// Config 优化器配置
type Config struct {
	TargetDir     string
	StructName    string
	Package       string
	SourceFile    string
	Write         bool
	Backup        bool
	SkipDirs      []string
	SkipFiles     []string
	SkipPatterns  []string
	SkipByMethods []string
	SkipByNames   []string
	Verbose       int
	SortSameSize  bool
	Output        string
	ProjectType   string // 项目类型：gomod 或 gopath
	GOPATH        string // GOPATH 路径（可选）
}

// Report 优化报告
type Report struct {
	TotalStructs   int
	OptimizedCount int
	SkippedCount   int
	TotalSaved     int64
	StructReports  []*StructReport
}

// StructReport 结构体报告
type StructReport struct {
	Name        string
	PkgPath     string
	File        string
	OrigSize    int64
	OptSize     int64
	Saved       int64
	OrigFields  []string
	OptFields   []string
	Skipped     bool
	SkipReason  string
	Depth       int // 嵌套深度
}

// NewOptimizer 创建优化器
func NewOptimizer(cfg *Config, analyzer *analyzer.Analyzer) *Optimizer {
	return &Optimizer{
		config:   cfg,
		analyzer: analyzer,
		optimized: make(map[string]*StructInfo),
		processing: make(map[string]bool),
		maxDepth: 50, // 最大递归深度 50 层
		report: &Report{
			StructReports: make([]*StructReport, 0),
		},
	}
}

// Optimize 执行优化（入口函数）
func (o *Optimizer) Optimize() (*Report, error) {
	o.Log(1, "开始优化...")

	if o.config.StructName != "" {
		// 优化指定结构体
		pkgPath, structName := analyzer.ParseStructName(o.config.StructName)
		if pkgPath == "" {
			return nil, fmt.Errorf("invalid struct name format: %s", o.config.StructName)
		}

		o.Log(1, "优化结构体：%s.%s", pkgPath, structName)
		_, err := o.optimizeStruct(pkgPath, structName, "", 0)
		if err != nil {
			return nil, err
		}
	} else if o.config.Package != "" {
		// 优化包中所有结构体
		o.Log(1, "优化包：%s", o.config.Package)
		structs, err := o.analyzer.FindAllStructs(o.config.Package)
		if err != nil {
			return nil, err
		}

		for _, st := range structs {
			o.Log(2, "处理结构体：%s", st.Name)
			_, err := o.optimizeStruct(st.PkgPath, st.Name, st.File, 0)
			if err != nil {
				o.Log(1, "优化 %s 失败：%v", st.Name, err)
			}
		}
	}

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

	// 检查是否已优化
	if info, ok := o.optimized[key]; ok {
		o.Log(3, "结构体已处理：%s", key)
		return info, nil
	}

	// 检测循环引用：如果正在处理中，说明有循环引用
	if o.processing[key] {
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
	defer func() {
		// 处理完成后，移除标记
		delete(o.processing, key)
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
		o.optimized[key] = info
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
		o.optimized[key] = info
		o.addReport(info, func() string {
			if isStandardLibrary(pkgPath) {
				return "Go 标准库结构体"
			}
			return "非项目内部包结构体"
		}(), depth)
		return info, nil
	}

	o.Log(2, "[%d] 处理结构体：%s (文件：%s)", depth, key, filePath)

	// 查找结构体
	st, filePath, err := o.analyzer.FindStructByName(pkgPath, structName)

	if err != nil {
		o.Log(2, "查找结构体失败：%v", err)
		return o.createSkippedInfo(key, filePath, "查找失败："+err.Error()), nil
	}

	// 创建字段分析器
	o.fieldAnalyzer = NewFieldAnalyzer(o.analyzer.GetTypesInfo(), o.analyzer.GetFset())

	// 分析结构体
	info := o.fieldAnalyzer.AnalyzeStruct(st, structName, pkgPath, filePath)

	// 检查是否应该跳过
	if skipReason := o.shouldSkip(info, st, key); skipReason != "" {
		o.Log(2, "跳过结构体：%s, 原因：%s", key, skipReason)
		info.Skipped = true
		info.SkipReason = skipReason
		o.optimized[key] = info
		o.addReport(info, skipReason, depth)
		return info, nil
	}

	// 优化嵌套字段（深度优先）- 只优化项目内部的结构体
	for _, field := range info.Fields {
		// 跳过接口类型（大小固定，不需要优化）
		if field.IsInterface {
			o.Log(3, "跳过接口类型字段：%s (大小固定)", field.TypeName)
			continue
		}

		// 跳过标准库类型字段（不需要优化）
		if field.IsStdLib {
			o.Log(3, "跳过标准库字段：%s", field.TypeName)
			continue
		}

		// 跳过第三方包字段（不需要优化）
		if field.IsThirdParty {
			o.Log(3, "跳过第三方包字段：%s", field.TypeName)
			continue
		}

		// 检查是否是结构体类型
		if field.TypeName != "" && isStructType(field.Type) {
			// 获取字段类型的包路径
			fieldPkg := field.PkgPath
			// 如果是同包结构体，使用当前包路径
			if fieldPkg == "" {
				fieldPkg = pkgPath
			}

			// 只优化项目内部的结构体
			if fieldPkg != "" && o.isProjectPackage(fieldPkg) {
				o.Log(3, "优化嵌套结构体：%s.%s (深度:%d)", fieldPkg, field.TypeName, depth+1)
				_, err := o.optimizeStruct(fieldPkg, field.TypeName, "", depth+1)
				if err != nil {
					o.Log(2, "优化嵌套结构体失败：%v", err)
				}
			}
		}
	}

	// 重排字段
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

	o.optimized[key] = info
	o.addReport(info, "", depth)

	return info, nil
}

// shouldSkip 检查是否应该跳过
func (o *Optimizer) shouldSkip(info *StructInfo, st *types.Struct, key string) string {
	// 空结构体
	if len(info.Fields) == 0 {
		return "空结构体"
	}

	// 单字段结构体
	if len(info.Fields) == 1 {
		return "单字段结构体"
	}

	// 检查是否是 vendor 中的第三方包结构体
	if isVendorPackage(info.PkgPath) {
		return "vendor 中的第三方包结构体"
	}

	// 检查是否是项目内部的包
	if !o.isProjectPackage(info.PkgPath) {
		// 判断是标准库还是其他第三方包
		if isStandardLibrary(info.PkgPath) {
			return "Go 标准库结构体"
		}
		return "非项目内部包结构体"
	}

	// 检查是否通过名字指定跳过
	if len(o.config.SkipByNames) > 0 {
		for _, name := range o.config.SkipByNames {
			// 支持全名匹配（包路径。结构体名）和简单名称匹配
			// 支持通配符匹配
			if o.matchStructName(key, name) {
				return "通过名字指定跳过：" + name
			}
		}
	}

	// 检查是否有指定方法（需要具名类型）
	if len(o.config.SkipByMethods) > 0 {
		// 从 StructInfo 中获取具名类型信息
		// 注意：info.Name 可能是完整名称（包含包路径），需要提取结构体名
		structName := info.Name
		// 如果 Name 中包含点号，取最后一部分作为结构体名
		if idx := strings.LastIndex(info.Name, "."); idx != -1 {
			structName = info.Name[idx+1:]
		}
		
		namedType := o.findNamedType(info.PkgPath, structName)
		if namedType != nil {
			for _, method := range o.config.SkipByMethods {
				if o.hasMethod(namedType, method) {
					return "具有方法：" + method
				}
			}
		}
	}

	return ""
}

// matchStructName 匹配结构体名称（支持通配符）
func (o *Optimizer) matchStructName(fullName, pattern string) bool {
	// 完整匹配
	if strings.HasSuffix(fullName, "."+pattern) || fullName == pattern {
		return true
	}
	
	// 通配符匹配
	if matched, err := filepath.Match(pattern, fullName); err == nil && matched {
		return true
	}
	
	// 提取结构体名进行匹配
	structName := fullName
	if idx := strings.LastIndex(fullName, "."); idx != -1 {
		structName = fullName[idx+1:]
	}
	
	if matched, err := filepath.Match(pattern, structName); err == nil && matched {
		return true
	}
	
	return false
}

// matchMethod 匹配方法名（支持通配符）
func (o *Optimizer) matchMethod(methodName, pattern string) bool {
	// 完全匹配
	if methodName == pattern {
		return true
	}
	
	// 通配符匹配
	if matched, err := filepath.Match(pattern, methodName); err == nil && matched {
		return true
	}
	
	return false
}

// isVendorPackage 判断是否是 vendor 中的包或第三方包
func isVendorPackage(pkgPath string) bool {
	// 1. 空包路径（通常是标准库或内置类型）
	if pkgPath == "" {
		return true
	}

	// 2. 检查是否包含 vendor 目录
	if strings.Contains(pkgPath, "/vendor/") || strings.HasPrefix(pkgPath, "vendor/") {
		return true
	}

	// 注意：不再在这里判断 github.com/ 等前缀
	// 因为用户的项目也可能在 github.com 下
	// 具体判断交给 isProjectPackage 函数

	return false
}

// isStandardLibrary 判断是否是 Go 标准库
func isStandardLibrary(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	// 标准库的特点：
	// 1. 不包含点号（没有域名）
	// 2. 是单级包名（如 "fmt", "os"）或者是 Go 官方多级包（如 "go/types", "net/http"）
	// 
	// 项目包的特点：
	// 1. 通常包含域名（如 "github.com/user/project"）
	// 2. 或者是相对路径（如 "analyzer", "./analyzer"）
	//
	// 判断规则：
	// - 如果包含点号，肯定不是标准库
	// - 如果不包含点号，且包含 "/" 但不是 "go/" 开头，可能是项目相对路径
	// - 如果不包含点号，且不包含 "/"，可能是标准库或项目根目录下的包
	if strings.Contains(pkgPath, ".") {
		return false // 包含点号，不是标准库
	}
	// 不包含点号
	if strings.Contains(pkgPath, "/") {
		// 包含斜杠，检查是否是 Go 官方包
		return strings.HasPrefix(pkgPath, "go/") || strings.HasPrefix(pkgPath, "internal/") || strings.HasPrefix(pkgPath, "cmd/")
	}
	// 单级包名，可能是标准库（如 "fmt"）或项目根包（如 "analyzer"）
	// 这里保守判断，只确认是标准库的单级包名
	standardLibs := map[string]bool{
		"fmt": true, "os": true, "io": true, "net": true, "http": true,
		"reflect": true, "errors": true, "bytes": true, "strings": true,
		"bufio": true, "sort": true, "sync": true, "time": true,
		"math": true, "rand": true, "regexp": true, "encoding": true,
		"json": true, "xml": true, "csv": true, "html": true, "url": true,
		"template": true, "text": true, "mime": true, "crypto": true,
		"hash": true, "compress": true, "archive": true, "image": true,
		"draw": true, "color": true, "jpeg": true, "png": true, "gif": true,
		"syscall": true, "runtime": true, "debug": true, "plugin": true,
		"unsafe": true, "atomic": true, "pprof": true, "trace": true,
		"flag": true, "log": true, "testing": true, "testing/iotest": true,
		"iotest": true, "quick": true, "exec": true, "signal": true,
		"path": true, "file": true, "filepath": true,
	}
	return standardLibs[pkgPath]
}

// isProjectPackage 判断是否是项目内部的包
// 需要根据项目类型（gomod/gopath）来判断
func (o *Optimizer) isProjectPackage(pkgPath string) bool {
	// 空包路径不是项目包
	if pkgPath == "" {
		return false
	}

	// vendor 中的不是项目包
	if isVendorPackage(pkgPath) {
		return false
	}

	// 标准库不是项目包
	if isStandardLibrary(pkgPath) {
		return false
	}

	// GOPATH 模式下，需要检查是否在项目路径下
	if o.config.ProjectType == "gopath" {
		gopath := os.Getenv("GOPATH")
		if gopath != "" {
			// 检查包路径是否以 GOPATH/src/ 开头
			if strings.HasPrefix(pkgPath, "src/") {
				// 提取项目路径
				relPath := strings.TrimPrefix(pkgPath, "src/")
				// 检查是否包含 vendor
				if strings.Contains(relPath, "/vendor/") {
					return false
				}
				return true
			}
		}
		// GOPATH 模式下，只要不是 vendor 和标准库，就认为是项目包
		return true
	}

	// Go Module 模式下，需要检查是否是当前项目的包
	if o.config.ProjectType == "gomod" || o.config.ProjectType == "" {
		// 获取项目根目录
		targetDir := o.config.TargetDir
		if targetDir == "" {
			targetDir = "."
		}

		// 尝试读取 go.mod 获取模块路径
		goModPath := filepath.Join(targetDir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			// 解析 go.mod 第一行获取模块路径
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					// 检查包路径是否以模块路径开头
					if strings.HasPrefix(pkgPath, modulePath) {
						// 确保是子路径，不是前缀匹配
						// 例如：modulePath="github.com/a/b", pkgPath="github.com/abc" 应该返回 false
						remaining := strings.TrimPrefix(pkgPath, modulePath)
						if remaining == "" || strings.HasPrefix(remaining, "/") {
							return true
						}
					}
					// 是其他模块的包，不是项目包
					return false
				}
			}
		}

		// 如果无法解析 go.mod，保守判断：只要不是 vendor 和标准库，就认为是项目包
		return true
	}

	// 默认认为是项目包
	return true
}

// findNamedType 查找具名类型
func (o *Optimizer) findNamedType(pkgPath, structName string) *types.Named {
	pkg, err := o.analyzer.LoadPackage(pkgPath)
	if err != nil {
		return nil
	}

	// 在包的作用域中查找类型
	scope := pkg.Types.Scope()
	obj := scope.Lookup(structName)
	if obj == nil {
		return nil
	}

	// 尝试转换为 Named 类型
	if named, ok := obj.Type().(*types.Named); ok {
		return named
	}

	return nil
}

// hasMethod 检查类型是否有指定方法（支持通配符）
func (o *Optimizer) hasMethod(named *types.Named, methodPattern string) bool {
	if named == nil {
		return false
	}

	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if o.matchMethod(method.Name(), methodPattern) {
			return true
		}
	}
	return false
}

// fieldOrderSame 检查字段顺序是否相同
func (o *Optimizer) fieldOrderSame(orig, opt []string) bool {
	if len(orig) != len(opt) {
		return false
	}
	for i := range orig {
		if orig[i] != opt[i] {
			return false
		}
	}
	return true
}

// createSkippedInfo 创建跳过的结构体信息
func (o *Optimizer) createSkippedInfo(key, filePath, reason string) *StructInfo {
	info := &StructInfo{
		Name:       key,
		PkgPath:    filepath.Dir(key),
		File:       filePath,
		Skipped:    true,
		SkipReason: reason,
	}
	o.optimized[key] = info
	o.addReport(info, reason, 0)
	return info
}

// addReport 添加报告
func (o *Optimizer) addReport(info *StructInfo, skipReason string, depth int) {
	report := &StructReport{
		Name:       info.Name,
		PkgPath:    info.PkgPath,
		File:       info.File,
		OrigSize:   info.OrigSize,
		OptSize:    info.OptSize,
		Saved:      info.OrigSize - info.OptSize,
		OrigFields: info.OrigOrder,
		OptFields:  info.OptOrder,
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
		timestamp := time.Now().Format("15:04:05.000")
		levelPrefix := ""
		switch level {
		case 1:
			levelPrefix = "[INFO] "
		case 2:
			levelPrefix = "[DEBUG] "
		case 3:
			levelPrefix = "[TRACE] "
		}
		fmt.Printf("%s%s "+format+"\n", append([]interface{}{timestamp, levelPrefix}, args...)...)
	}
}

// isStructType 检查类型是否是结构体类型（需要优化的类型）
// 接口类型大小固定，不需要优化
func isStructType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch t := typ.(type) {
	case *types.Interface:
		// 接口类型大小固定（16 字节），不需要优化
		return false
	case *types.Struct:
		return true
	case *types.Named:
		return isStructType(t.Underlying())
	case *types.Pointer:
		return isStructType(t.Elem())
	case *types.Slice:
		// 检查 slice 的元素类型
		return isStructType(t.Elem())
	case *types.Map:
		// 检查 map 的键和值类型
		return isStructType(t.Key()) || isStructType(t.Elem())
	default:
		return false
	}
}
