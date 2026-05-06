package optimizer

import (
	"go/token"
	"go/types"
	"strings"
)

// FieldInfo holds field information
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

// FieldAnalyzer analyzes struct fields
type FieldAnalyzer struct {
	info *types.Info
	fset *token.FileSet
}

// NewFieldAnalyzer creates a new field analyzer
func NewFieldAnalyzer(info *types.Info, fset *token.FileSet) *FieldAnalyzer {
	return &FieldAnalyzer{
		info: info,
		fset: fset,
	}
}

// AnalyzeStruct analyzes the fields of a struct
func (fa *FieldAnalyzer) AnalyzeStruct(st *types.Struct, structName, pkgPath, filePath string) *StructInfo {
	info := &StructInfo{
		Name:    structName,
		PkgPath: pkgPath,
		File:    filePath,
	}

	if st == nil {
		return info
	}

	// Analyze fields
	fields, origOrder := fa.analyzeFields(st, structName, pkgPath, filePath)
	info.Fields = fields
	info.OrigOrder = origOrder

	// Calculate original size using types.Sizes (consistent with unsafe.Sizeof)
	info.OrigSize = CalcStructSize(st)

	return info
}

// analyzeFields analyzes the fields of a struct
func (fa *FieldAnalyzer) analyzeFields(structType *types.Struct, structName, pkgPath, filePath string) ([]FieldInfo, []string) {
	var fields []FieldInfo
	var origOrder []string

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		size, align := CalcFieldSize(field.Type())

		typeName := fa.getTypeName(field.Type())
		pkg := fa.getTypePkg(field.Type())

		// Quickly check if it is a standard library or third-party package
		isStdLib := isStandardLibraryPkg(pkg)
		isThirdParty := !isStdLib && pkg != "" && !isProjectPkgFast(pkg)

		fi := FieldInfo{
			Name:         fa.getFieldName(field),
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

		// Get tag
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

// getFieldName returns the field name
func (fa *FieldAnalyzer) getFieldName(field *types.Var) string {
	return field.Name()
}

// getTypePkg returns the package path of a type
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

// getTypeName returns the type name (keeping the package prefix)
func (fa *FieldAnalyzer) getTypeName(typ types.Type) string {
	if typ == nil {
		return ""
	}
	switch t := typ.(type) {
	case *types.Named:
		// Keep package name: pkg.TypeName
		if pkg := t.Obj().Pkg(); pkg != nil {
			return pkg.Path() + "." + t.Obj().Name()
		}
		return t.Obj().Name()
	case *types.Pointer:
		return "*" + fa.getTypeName(t.Elem())
	case *types.Slice:
		return "[]" + fa.getTypeName(t.Elem())
	case *types.Array:
		return "[]" + fa.getTypeName(t.Elem())
	case *types.Map:
		return "map[" + fa.getTypeName(t.Key()) + "]" + fa.getTypeName(t.Elem())
	default:
		return typ.String()
	}
}

// isProjectPkgFast quick check for whether it is a project package
func isProjectPkgFast(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}
	return strings.Contains(pkgPath, "/") && !isStandardLibraryPkg(pkgPath)
}
