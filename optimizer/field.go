package optimizer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// FieldInfo 字段信息
type FieldInfo struct {
	Name     string      // 字段名（匿名字段为空）
	Type     types.Type  // 字段类型
	Size     int64       // 字段大小
	Align    int64       // 字段对齐要求
	IsEmbed  bool        // 是否匿名字段
	PkgPath  string      // 字段类型所在包路径
	TypeName string      // 类型名称
	Tag      string      // 字段 tag
}

// StructInfo 结构体信息
type StructInfo struct {
	Name       string      // 结构体名称
	PkgPath    string      // 包路径
	File       string      // 源文件路径
	Fields     []FieldInfo // 字段列表
	OrigSize   int64       // 优化前大小
	OptSize    int64       // 优化后大小
	Optimized  bool        // 是否已优化
	Skipped    bool        // 是否被跳过
	SkipReason string      // 跳过原因
	OrigOrder  []string    // 原始字段顺序
	OptOrder   []string    // 优化后字段顺序
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

// AnalyzeStruct 分析结构体
func (fa *FieldAnalyzer) AnalyzeStruct(structType *types.Struct, structName, pkgPath, filePath string) *StructInfo {
	if structType.NumFields() == 0 {
		return &StructInfo{
			Name:     structName,
			PkgPath:  pkgPath,
			File:     filePath,
			Fields:   []FieldInfo{},
			OrigSize: 0,
			OptSize:  0,
		}
	}

	fields := make([]FieldInfo, 0, structType.NumFields())
	var origOrder []string

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		size, align := CalcFieldSize(field.Type(), fa.info)

		typeName := fa.getTypeName(field.Type())
		pkg := fa.getTypePkg(field.Type())

		fi := FieldInfo{
			Name:     fa.getFieldName(field, i),
			Type:     field.Type(),
			Size:     size,
			Align:    align,
			IsEmbed:  field.Embedded(),
			PkgPath:  pkg,
			TypeName: typeName,
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

	origSize := CalcStructSize(structType, fa.info)

	return &StructInfo{
		Name:      structName,
		PkgPath:   pkgPath,
		File:      filePath,
		Fields:    fields,
		OrigSize:  origSize,
		OrigOrder: origOrder,
	}
}

// getFieldName 获取字段名（处理匿名字段）
func (fa *FieldAnalyzer) getFieldName(field *types.Var, index int) string {
	if field.Embedded() {
		return ""
	}
	return field.Name()
}

// getTypeName 获取类型名称
func (fa *FieldAnalyzer) getTypeName(typ types.Type) string {
	switch t := typ.(type) {
	case *types.Named:
		return t.Obj().Name()
	case *types.Basic:
		return t.Name()
	case *types.Pointer:
		return "*" + fa.getTypeName(t.Elem())
	case *types.Slice:
		return "[]" + fa.getTypeName(t.Elem())
	case *types.Array:
		return "[" + itoa(int(t.Len())) + "]" + fa.getTypeName(t.Elem())
	case *types.Map:
		return "map[" + fa.getTypeName(t.Key()) + "]" + fa.getTypeName(t.Elem())
	case *types.Chan:
		return "chan " + fa.getTypeName(t.Elem())
	case *types.Struct:
		return "struct{}"
	case *types.Interface:
		if t.NumMethods() == 0 {
			return "interface{}"
		}
		return "interface"
	default:
		return typ.String()
	}
}

// getTypePkg 获取类型所在包路径
func (fa *FieldAnalyzer) getTypePkg(typ types.Type) string {
	switch t := typ.(type) {
	case *types.Named:
		if t.Obj().Pkg() != nil {
			return t.Obj().Pkg().Path()
		}
	case *types.Pointer:
		return fa.getTypePkg(t.Elem())
	case *types.Slice:
		return fa.getTypePkg(t.Elem())
	case *types.Array:
		return fa.getTypePkg(t.Elem())
	case *types.Map:
		// Map 的包路径通常不重要
		return ""
	}
	return ""
}

// ReorderFields 重排字段（按大小降序，相同大小按对齐降序）
func ReorderFields(fields []FieldInfo, sortSameSize bool) []FieldInfo {
	if len(fields) <= 1 {
		return fields
	}

	// 分离匿名字段和命名字段
	var embeds []FieldInfo
	var named []FieldInfo

	for _, f := range fields {
		if f.IsEmbed {
			embeds = append(embeds, f)
		} else {
			named = append(named, f)
		}
	}

	// 对命名字段排序
	named = sortFields(named, sortSameSize)

	// 合并：匿名字段在前，命名字段在后
	result := make([]FieldInfo, 0, len(fields))
	result = append(result, embeds...)
	result = append(result, named...)

	return result
}

// sortFields 对字段进行排序
func sortFields(fields []FieldInfo, sortSameSize bool) []FieldInfo {
	// 简单的插入排序（字段数量通常不多）
	result := make([]FieldInfo, len(fields))
	copy(result, fields)

	for i := 1; i < len(result); i++ {
		key := result[i]
		j := i - 1

		for j >= 0 {
			shouldSwap := false

			if result[j].Size < key.Size {
				shouldSwap = true
			} else if result[j].Size == key.Size {
				if result[j].Align < key.Align {
					shouldSwap = true
				} else if result[j].Align == key.Align && sortSameSize {
					// 大小和对齐都相同，按类型名排序
					if strings.Compare(result[j].TypeName, key.TypeName) > 0 {
						shouldSwap = true
					}
				}
			}

			if shouldSwap {
				result[j+1] = result[j]
				j--
			} else {
				break
			}
		}
		result[j+1] = key
	}

	return result
}

// itoa 简单的整数转字符串
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// GetStructTypeFromAst 从 AST 获取结构体类型
func GetStructTypeFromAst(astStruct *ast.StructType, info *types.Info) *types.Struct {
	if astStruct.Fields == nil {
		return types.NewStruct(nil, nil)
	}

	var fields []*types.Var
	var tags []string

	for _, field := range astStruct.Fields.List {
		// 获取字段的类型信息
		typ := info.TypeOf(field.Type)
		if typ == nil {
			continue
		}

		var tag string
		if field.Tag != nil {
			tag = field.Tag.Value
			// 去除反引号
			if len(tag) >= 2 {
				tag = tag[1 : len(tag)-1]
			}
		}

		if len(field.Names) == 0 {
			// 匿名字段
			fields = append(fields, types.NewField(field.Pos(), nil, "", typ, true))
			tags = append(tags, tag)
		} else {
			for _, name := range field.Names {
				fields = append(fields, types.NewField(name.Pos(), nil, name.Name, typ, false))
				tags = append(tags, tag)
			}
		}
	}

	if len(fields) == 0 {
		return types.NewStruct(nil, nil)
	}

	return types.NewStruct(fields, tags)
}
