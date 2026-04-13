package optimizer

import (
	"fmt"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// Optimizer 优化器
type Optimizer struct {
	config      *Config
	analyzer    *analyzer.Analyzer
	optimized   map[string]*StructInfo // 已优化的结构体（key: pkgPath.structName）
	report      *Report
	fieldAnalyzer *FieldAnalyzer
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
	Verbose       int
	SortSameSize  bool
	Output        string
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

	// 检查是否已优化
	if info, ok := o.optimized[key]; ok {
		o.Log(3, "结构体已处理：%s", key)
		return info, nil
	}

	// 检查是否是 vendor 中的包，如果是则跳过
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
	if skipReason := o.shouldSkip(info, st); skipReason != "" {
		o.Log(2, "跳过结构体：%s, 原因：%s", key, skipReason)
		info.Skipped = true
		info.SkipReason = skipReason
		o.optimized[key] = info
		o.addReport(info, skipReason, depth)
		return info, nil
	}

	// 优化嵌套字段（深度优先）- 包括同包和跨包的结构体
	for _, field := range info.Fields {
		// 检查是否是结构体类型
		if field.TypeName != "" && isStructType(field.Type) {
			// 获取字段类型的包路径
			fieldPkg := field.PkgPath
			// 如果是同包结构体，使用当前包路径
			if fieldPkg == "" {
				fieldPkg = pkgPath
			}
			
			// 跳过 vendor 中的第三方包结构体
			if fieldPkg != "" && !isVendorPackage(fieldPkg) {
				o.Log(3, "优化嵌套结构体：%s.%s (深度:%d)", fieldPkg, field.TypeName, depth+1)
				_, err := o.optimizeStruct(fieldPkg, field.TypeName, "", depth+1)
				if err != nil {
					o.Log(2, "优化嵌套结构体失败：%v", err)
				}
			} else if fieldPkg != "" && isVendorPackage(fieldPkg) {
				o.Log(3, "跳过 vendor 中的结构体：%s.%s", fieldPkg, field.TypeName)
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
func (o *Optimizer) shouldSkip(info *StructInfo, st *types.Struct) string {
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

// isVendorPackage 判断是否是 vendor 中的包
func isVendorPackage(pkgPath string) bool {
	// 检查包路径中是否包含 /vendor/ 或 vendor/
	return strings.Contains(pkgPath, "/vendor/") || strings.HasPrefix(pkgPath, "vendor/")
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

// hasMethod 检查类型是否有指定方法
func (o *Optimizer) hasMethod(named *types.Named, methodName string) bool {
	if named == nil {
		return false
	}

	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == methodName {
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
		prefix := ""
		switch level {
		case 1:
			prefix = "[INFO] "
		case 2:
			prefix = "[DEBUG] "
		case 3:
			prefix = "[TRACE] "
		}
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// isStructType 检查类型是否是结构体类型
func isStructType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch t := typ.(type) {
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
