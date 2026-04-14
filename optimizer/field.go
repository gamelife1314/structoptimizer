package optimizer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// FieldInfo 字段信息
type FieldInfo struct {
	Name        string      // 字段名（匿名字段为空）
	Type        types.Type  // 字段类型
	Size        int64       // 字段大小
	Align       int64       // 字段对齐要求
	IsEmbed     bool        // 是否匿名字段
	IsInterface bool        // 是否是接口类型（接口大小固定，不需要优化）
	IsStdLib    bool        // 是否是标准库类型（不需要深入分析）
	IsThirdParty bool       // 是否是第三方包类型（不需要深入分析）
	PkgPath     string      // 字段类型所在包路径
	TypeName    string      // 类型名称
	Tag         string      // 字段 tag
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
		
		// 获取字段类型信息
		fieldType := field.Type()
		typeName := fa.getTypeName(fieldType)
		pkg := fa.getTypePkg(fieldType)
		
		// 快速判断是否是标准库或第三方包
		isStdLib := isStandardLibraryPkg(pkg)
		isThirdParty := !isStdLib && pkg != "" && !isProjectPkg(pkg)
		
		// 对于标准库和第三方包，使用快速大小计算，不深入递归
		var size, align int64
		if isStdLib || isThirdParty {
			size, align = calcFieldSizeFast(fieldType)
		} else {
			size, align = CalcFieldSize(fieldType, fa.info)
		}

		fi := FieldInfo{
			Name:        fa.getFieldName(field, i),
			Type:        fieldType,
			Size:        size,
			Align:       align,
			IsEmbed:     field.Embedded(),
			IsInterface: isInterfaceType(fieldType),
			IsStdLib:    isStdLib,
			IsThirdParty: isThirdParty,
			PkgPath:     pkg,
			TypeName:    typeName,
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

// isInterfaceType 检查类型是否是接口类型
func isInterfaceType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch t := typ.(type) {
	case *types.Interface:
		return true
	case *types.Named:
		return isInterfaceType(t.Underlying())
	case *types.Pointer:
		return isInterfaceType(t.Elem())
	default:
		return false
	}
}

// isStandardLibraryPkg 快速判断是否是标准库包
func isStandardLibraryPkg(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	// 标准库不包含点号
	if strings.Contains(pkgPath, ".") {
		return false
	}
	// 单级包名或 go/ 开头的多级包名
	if !strings.Contains(pkgPath, "/") || strings.HasPrefix(pkgPath, "go/") {
		return true
	}
	return false
}

// isProjectPkg 快速判断是否是项目包（基于常见项目路径前缀）
// 注意：这是一个快速判断，真正的判断需要在优化器中进行
func isProjectPkg(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}
	// 如果包含常见的代码托管平台域名，可能是项目包
	// 这里只做快速判断，详细判断交给优化器
	return strings.Contains(pkgPath, "/") && !isStandardLibraryPkg(pkgPath)
}

// calcFieldSizeFast 快速计算字段大小，不深入递归分析
// 用于标准库和第三方包类型
func calcFieldSizeFast(typ types.Type) (size, align int64) {
	if typ == nil {
		return 0, 1
	}
	
	switch t := typ.(type) {
	case *types.Basic:
		return basicSizeFast(t.Kind())
	
	case *types.Pointer:
		// 指针大小固定
		return sizeofPtr(), alignofPtr()
	
	case *types.Array:
		// 数组：元素大小 * 元素个数
		elemSize, elemAlign := calcFieldSizeFast(t.Elem())
		if t.Len() == 0 {
			return 0, elemAlign
		}
		return elemSize * t.Len(), elemAlign
	
	case *types.Slice:
		// slice 头大小固定（24 字节：data 指针 + len + cap）
		return 24, 8
	
	case *types.Map:
		// map 头大小固定（8 字节指针）
		return 8, 8
	
	case *types.Chan:
		// chan 大小固定（8 字节指针）
		return 8, 8
	
	case *types.Interface:
		// 接口大小固定（16 字节：data 指针 + type 指针）
		return 16, 8
	
	case *types.Named:
		// 具名类型，使用底层类型的快速计算
		return calcFieldSizeFast(t.Underlying())
	
	case *types.Struct:
		// 对于匿名结构体，使用简化计算
		return calcStructSizeFast(t)
	
	default:
		// 其他类型，返回默认值
		return 8, 8
	}
}

// basicSizeFast 快速计算基本类型大小
func basicSizeFast(kind types.BasicKind) (size, align int64) {
	switch kind {
	case types.Bool, types.Uint8, types.Int8:
		return 1, 1
	case types.Uint16, types.Int16:
		return 2, 2
	case types.Uint32, types.Int32, types.Float32:
		return 4, 4
	case types.Uint64, types.Int64, types.Float64:
		return 8, 8
	case types.Uint, types.Int:
		return 8, 8 // 假设 64 位系统
	case types.Uintptr:
		return 8, 8
	case types.String:
		return 16, 8 // string 头（data 指针 + len）
	case types.UnsafePointer:
		return 8, 8
	default:
		return 8, 8
	}
}

// calcStructSizeFast 快速计算结构体大小
func calcStructSizeFast(st *types.Struct) (size, align int64) {
	if st.NumFields() == 0 {
		return 0, 1
	}
	
	var offset int64 = 0
	var maxAlign int64 = 1
	
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		sz, al := calcFieldSizeFast(field.Type())
		
		// 对齐
		if offset%al != 0 {
			offset += al - (offset % al)
		}
		
		offset += sz
		if al > maxAlign {
			maxAlign = al
		}
	}
	
	// 末尾填充
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}
	
	return offset, maxAlign
}

// sizeofPtr 返回指针大小
func sizeofPtr() int64 {
	return 8 // 64 位系统
}

// alignofPtr 返回指针对齐
func alignofPtr() int64 {
	return 8
}
