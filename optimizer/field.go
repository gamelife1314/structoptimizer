package optimizer

import (
	"go/token"
	"go/types"
	"strings"
)

// FieldInfo 字段信息
type FieldInfo struct {
	Name         string
	Type         types.Type
	Size         int64
	Align        int64
	IsEmbed      bool
	IsInterface  bool
	IsStdLib     bool
	IsThirdParty bool
	PkgPath      string
	TypeName     string
	Tag          string
}

// FieldAnalyzer 字段分析器
type FieldAnalyzer struct {
	info *types.Info
	fset *token.FileSet
}

// NewFieldAnalyzer 创建字段分析器
func NewFieldAnalyzer(info *types.Info, fset *token.FileSet) *FieldAnalyzer {
	return &FieldAnalyzer{
		info: info,
		fset: fset,
	}
}

// AnalyzeStruct 分析结构体字段
func (fa *FieldAnalyzer) AnalyzeStruct(st *types.Struct, structName, pkgPath, filePath string) *StructInfo {
	info := &StructInfo{
		Name:    structName,
		PkgPath: pkgPath,
		File:    filePath,
	}

	if st == nil {
		return info
	}

	// 分析字段
	fields, origOrder := fa.analyzeFields(st, structName, pkgPath, filePath)
	info.Fields = fields
	info.OrigOrder = origOrder

	// 计算原始大小
	// 注意：types.Info 没有 Sizes 字段，使用手动计算
	info.OrigSize = CalcStructSize(st, nil)

	return info
}

// analyzeFields 分析结构体字段
func (fa *FieldAnalyzer) analyzeFields(structType *types.Struct, structName, pkgPath, filePath string) ([]FieldInfo, []string) {
	var fields []FieldInfo
	var origOrder []string

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		size, align := CalcFieldSize(field.Type(), fa.info)

		typeName := fa.getTypeName(field.Type())
		pkg := fa.getTypePkg(field.Type())

		// 快速判断是否是标准库或第三方包
		isStdLib := isStandardLibraryPkg(pkg)
		isThirdParty := !isStdLib && pkg != "" && !isProjectPkgFast(pkg)

		fi := FieldInfo{
			Name:         fa.getFieldName(field, i),
			Type:         field.Type(),
			Size:         size,
			Align:        align,
			IsEmbed:      field.Embedded(),
			IsInterface:  isInterfaceType(field.Type()),
			IsStdLib:     isStdLib,
			IsThirdParty: isThirdParty,
			PkgPath:      pkg,
			TypeName:     typeName,
		}

		// 获取 tag
		if structType.Tag(i) != "" {
			fi.Tag = structType.Tag(i)
		}

		fields = append(fields, fi)
		if field.Name() != "" {
			origOrder = append(origOrder, field.Name())
		} else {
			origOrder = append(origOrder, typeName)
		}
	}

	return fields, origOrder
}

// getFieldName 获取字段名称
func (fa *FieldAnalyzer) getFieldName(field *types.Var, index int) string {
	// 直接返回字段名，匿名字段返回空字符串
	return field.Name()
}

// getTypePkg 获取类型的包路径
func (fa *FieldAnalyzer) getTypePkg(typ types.Type) string {
	if typ == nil {
		return ""
	}
	switch t := typ.(type) {
	case *types.Named:
		if obj := t.Obj(); obj != nil {
			if pkg := obj.Pkg(); pkg != nil {
				return pkg.Path()
			}
		}
		return ""
	case *types.Pointer:
		return fa.getTypePkg(t.Elem())
	case *types.Slice:
		return fa.getTypePkg(t.Elem())
	case *types.Array:
		return fa.getTypePkg(t.Elem())
	case *types.Map:
		return ""
	default:
		return ""
	}
}

// getTypeName 获取类型名称
func (fa *FieldAnalyzer) getTypeName(typ types.Type) string {
	if typ == nil {
		return ""
	}
	switch t := typ.(type) {
	case *types.Named:
		return t.Obj().Name()
	case *types.Pointer:
		return fa.getTypeName(t.Elem())
	case *types.Slice:
		return fa.getTypeName(t.Elem())
	case *types.Array:
		return fa.getTypeName(t.Elem())
	default:
		return typ.String()
	}
}

// isProjectPkgFast 快速判断是否是项目包
func isProjectPkgFast(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}
	return strings.Contains(pkgPath, "/") && !isStandardLibraryPkg(pkgPath)
}
