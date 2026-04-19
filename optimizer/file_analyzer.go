package optimizer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// analyzeStructFromFile 只解析文件分析结构体（不加载包，快速路径）
func analyzeStructFromFile(filePath, structName, pkgPath string) (*StructInfo, *types.Struct, error) {
	// 解析文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse file failed: %w", err)
	}

	// 查找结构体定义
	var foundDecl *ast.TypeSpec
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != structName {
				continue
			}

			if _, ok := ts.Type.(*ast.StructType); ok {
				foundDecl = ts
				break
			}
		}
		if foundDecl != nil {
			break
		}
	}

	if foundDecl == nil {
		return nil, nil, fmt.Errorf("struct %s not found in file", structName)
	}

	// 从 AST 提取字段信息
	info := &StructInfo{
		Name:    structName,
		PkgPath: pkgPath,
		File:    filePath,
	}

	// 获取包目录（用于查找同包中的类型定义）
	pkgDir := ""
	if filePath != "" {
		pkgDir = filepath.Dir(filePath)
	}

	st, fields := extractFieldsFromAST(foundDecl, fset, pkgDir)
	info.Fields = fields
	info.OrigOrder = extractFieldNames(fields)
	// 使用 CalcStructSizeFromFields 计算大小（基于字段的 size 和 align）
	// 注意：不使用 types.Sizes，因为 st 是简化的 Struct，字段类型是 Invalid
	info.OrigSize = CalcStructSizeFromFields(fields)

	return info, st, nil
}

// extractFieldsFromAST 从 AST 提取字段信息
func extractFieldsFromAST(ts *ast.TypeSpec, fset *token.FileSet, pkgDir string) (*types.Struct, []FieldInfo) {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, nil
	}

	var fields []FieldInfo
	var varFields []*types.Var

	for _, f := range st.Fields.List {
		typeName := extractTypeName(f.Type)
		size, align := estimateFieldSizeWithLookup(f.Type, pkgDir)

		// 判断是否是匿名字段
		isEmbed := len(f.Names) == 0

		// 获取字段名（用于 FieldInfo）
		fieldName := getFieldName(f)

		fi := FieldInfo{
			Name:     fieldName,
			Size:     size,
			Align:    align,
			TypeName: typeName,
			IsEmbed:  isEmbed, // 正确设置匿名字段标记
		}

		if f.Tag != nil {
			fi.Tag = strings.Trim(f.Tag.Value, "`")
		}

		fields = append(fields, fi)

		// 创建 types.Var 用于后续处理
		// 注意：匿名字段在 types.Var 中使用类型名，避免 "multifields with the same name" 错误
		typesFieldName := fieldName
		if isEmbed {
			typesFieldName = typeName
		}
		varFields = append(varFields, types.NewField(f.Pos(), nil, typesFieldName, types.Typ[types.Invalid], false))
	}

	// 创建简化的 types.Struct
	return types.NewStruct(varFields, nil), fields
}

// extractTypeName 从 AST 提取类型名称（保留包名前缀）
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + extractTypeName(t.X)
	case *ast.SelectorExpr:
		// 保留包名：pkg.TypeName
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.ArrayType:
		return "[]" + extractTypeName(t.Elt)
	case *ast.MapType:
		return "map[" + extractTypeName(t.Key) + "]" + extractTypeName(t.Value)
	case *ast.ChanType:
		return "chan " + extractTypeName(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}

// getFieldName 获取字段名称
func getFieldName(f *ast.Field) string {
	if len(f.Names) > 0 && f.Names[0] != nil {
		return f.Names[0].Name
	}
	// 匿名字段返回空字符串
	return ""
}

// estimateFieldSize 估算字段大小
func estimateFieldSize(expr ast.Expr) (size, align int64) {
	switch t := expr.(type) {
	case *ast.Ident:
		return sizeOfIdent(t.Name)
	case *ast.StarExpr:
		return 8, 8 // 指针
	case *ast.ArrayType:
		if t.Len == nil {
			return 24, 8 // slice
		}
		// 解析固定长度数组
		elemSize, elemAlign := estimateFieldSize(t.Elt)
		if length := parseArrayLength(t.Len); length > 0 {
			return elemSize * length, elemAlign
		}
		return elemSize, elemAlign // 无法解析时回退
	case *ast.MapType:
		return 8, 8 // map
	case *ast.ChanType:
		return 8, 8 // chan
	case *ast.InterfaceType:
		return 16, 8 // interface
	default:
		return 8, 8
	}
}

// EstimateFieldSizeWithLookup 估算字段大小（带类型查找）- 导出用于测试
func EstimateFieldSizeWithLookup(expr ast.Expr, pkgDir string) (size, align int64) {
	return estimateFieldSizeWithLookup(expr, pkgDir)
}

// estimateFieldSizeWithLookupInternal 估算字段大小（带类型查找）- 内部实现
func estimateFieldSizeWithLookup(expr ast.Expr, pkgDir string) (size, align int64) {
	switch t := expr.(type) {
	case *ast.Ident:
		// 对于标识符，尝试在同包中查找类型定义
		if pkgDir != "" {
			underlyingKind := findTypeUnderlyingInPackage(pkgDir, t.Name)
			if underlyingKind != types.Invalid {
				return basicSize(underlyingKind)
			}
			// 如果不是基本类型，尝试查找是否是结构体类型
			if structSize := findStructSizeInPackage(pkgDir, t.Name); structSize > 0 {
				return structSize, 8
			}
		}
		return sizeOfIdent(t.Name)
	case *ast.SelectorExpr:
		// 处理带包前缀的类型（如 time.Time）
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgName := ident.Name
			typeName := t.Sel.Name
			// 尝试查找标准库或已知外部包的结构体大小
			if size := getExternalStructSize(pkgName, typeName, pkgDir); size > 0 {
				return size, 8
			}
		}
		return 8, 8 // 未知外部类型
	case *ast.StarExpr:
		return 8, 8 // 指针
	case *ast.ArrayType:
		if t.Len == nil {
			return 24, 8 // slice
		}
		// 解析固定长度数组
		elemSize, elemAlign := estimateFieldSizeWithLookup(t.Elt, pkgDir)
		if length := parseArrayLength(t.Len); length > 0 {
			return elemSize * length, elemAlign
		}
		return elemSize, elemAlign // 无法解析时回退
	case *ast.MapType:
		return 8, 8 // map
	case *ast.ChanType:
		return 8, 8 // chan
	case *ast.InterfaceType:
		return 16, 8 // interface
	case *ast.StructType:
		// 直接处理内联结构体（匿名嵌套结构体）
		size := calcInlineStructSize(t, pkgDir)
		return size, 8
	default:
		return 8, 8
	}
}

// getExternalStructSize 获取外部包（标准库/第三方库）中结构体的大小
func getExternalStructSize(pkgName, typeName, localPkgDir string) int64 {
	// 标准库常见类型的大小
	if pkgName == "time" {
		switch typeName {
		case "Time":
			return 24 // time.Time 的实际大小
		case "Duration":
			return 8
		case "Location":
			return 8 // 指针
		}
	}
	if pkgName == "sync" {
		switch typeName {
		case "Mutex":
			return 8
		case "RWMutex":
			return 16
		case "WaitGroup":
			return 16
		case "Cond":
			return 16
		case "Once":
			return 8
		}
	}
	if pkgName == "context" {
		switch typeName {
		case "Context":
			return 16 // interface
		case "CancelFunc":
			return 8 // 函数指针
		}
	}
	if pkgName == "bytes" {
		switch typeName {
		case "Buffer":
			return 72
		}
	}
	if pkgName == "strings" {
		switch typeName {
		case "Builder":
			return 24
		}
	}
	if pkgName == "net" {
		switch typeName {
		case "IP":
			return 24 // slice
		case "IPMask":
			return 24 // slice
		}
	}
	if pkgName == "url" {
		switch typeName {
		case "URL":
			return 120
		}
	}
	if pkgName == "http" {
		switch typeName {
		case "Request":
			return 480
		case "Response":
			return 320
		case "Header":
			return 24 // map
		}
	}
	if pkgName == "json" {
		switch typeName {
		case "RawMessage":
			return 24 // slice
		}
	}
	
	// 如果不是标准库，尝试在 GOPATH 或 module cache 中查找
	if localPkgDir != "" {
		// 尝试在同级目录或 vendor 中查找
		if size := findStructSizeInVendorOrDep(pkgName, typeName, localPkgDir); size > 0 {
			return size
		}
	}
	
	return 0
}

// findStructSizeInVendorOrDep 在 vendor 或依赖目录中查找结构体大小
func findStructSizeInVendorOrDep(pkgName, typeName, localPkgDir string) int64 {
	// 尝试在 vendor 目录查找
	vendorDir := filepath.Join(localPkgDir, "..", "..", "vendor", pkgName)
	if size := findStructSizeInPackage(vendorDir, typeName); size > 0 {
		return size
	}
	
	// 尝试在 GOPATH/pkg/mod 中查找（简化处理，只检查常见路径）
	// 实际项目中可能需要更复杂的路径解析
	return 0
}

// findStructSizeInPackage 在包中查找结构体类型的大小
func findStructSizeInPackage(pkgDir, typeName string) int64 {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		if size := findStructSizeInFile(filePath, typeName); size > 0 {
			return size
		}
	}

	return 0
}

// findStructSizeInFile 在文件中查找结构体类型的大小
func findStructSizeInFile(filePath, typeName string) int64 {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return 0
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != typeName {
				continue
			}

			if st, ok := ts.Type.(*ast.StructType); ok {
				return calcInlineStructSize(st, filepath.Dir(filePath))
			}
		}
	}

	return 0
}

// calcInlineStructSize 计算内联结构体的大小
func calcInlineStructSize(st *ast.StructType, pkgDir string) int64 {
	if st == nil || st.Fields == nil {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for _, field := range st.Fields.List {
		size, align := estimateFieldSizeWithLookup(field.Type, pkgDir)

		// 对齐
		if offset%align != 0 {
			offset += align - (offset % align)
		}

		offset += size
		if align > maxAlign {
			maxAlign = align
		}
	}

	// 末尾填充
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// findTypeUnderlyingInPackage 在包中查找类型的底层类型
// 返回 types.BasicKind 如果是基本类型，否则返回 types.Invalid
func findTypeUnderlyingInPackage(pkgDir, typeName string) types.BasicKind {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return types.Invalid
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		if kind := findTypeUnderlyingInFile(filePath, typeName); kind != types.Invalid {
			return kind
		}
	}

	return types.Invalid
}

// findTypeUnderlyingInFile 在文件中查找类型的底层类型
func findTypeUnderlyingInFile(filePath, typeName string) types.BasicKind {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return types.Invalid
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != typeName {
				continue
			}

			// 检查底层类型
			return getBasicKindFromExpr(ts.Type)
		}
	}

	return types.Invalid
}

// getBasicKindFromExpr 从 AST 表达式获取基本类型种类
func getBasicKindFromExpr(expr ast.Expr) types.BasicKind {
	switch t := expr.(type) {
	case *ast.Ident:
		return identToBasicKind(t.Name)
	case *ast.ArrayType:
		// 数组/切片
		return types.Invalid
	case *ast.StarExpr:
		// 指针
		return types.Invalid
	default:
		return types.Invalid
	}
}

// identToBasicKind 将标识符名称转换为基本类型种类
func identToBasicKind(name string) types.BasicKind {
	switch name {
	case "bool":
		return types.Bool
	case "int8":
		return types.Int8
	case "uint8", "byte":
		return types.Uint8
	case "int16":
		return types.Int16
	case "uint16":
		return types.Uint16
	case "int32", "rune":
		return types.Int32
	case "uint32":
		return types.Uint32
	case "int64":
		return types.Int64
	case "uint64":
		return types.Uint64
	case "int":
		return types.Int
	case "uint":
		return types.Uint
	case "float32":
		return types.Float32
	case "float64":
		return types.Float64
	case "string":
		return types.String
	case "uintptr":
		return types.Uintptr
	default:
		return types.Invalid
	}
}

// sizeOfIdent 根据标识符名称估算大小
func sizeOfIdent(name string) (int64, int64) {
	switch name {
	case "bool", "byte":
		return 1, 1
	case "int8", "uint8":
		return 1, 1
	case "int16", "uint16":
		return 2, 2
	case "int32", "uint32", "rune":
		return 4, 4
	case "int64", "uint64":
		return 8, 8
	case "int", "uint", "uintptr":
		return 8, 8 // 64 位系统
	case "float32":
		return 4, 4
	case "float64":
		return 8, 8
	case "string":
		return 16, 8
	default:
		return 8, 8 // 未知类型
	}
}

// parseArrayLength 解析数组长度
func parseArrayLength(expr ast.Expr) int64 {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.INT {
			// 移除数字后缀（如 10u）
			value := strings.TrimRight(e.Value, "uU")
			if n, err := strconv.ParseInt(value, 0, 64); err == nil {
				return n
			}
		}
	case *ast.ParenExpr:
		return parseArrayLength(e.X)
	}
	return 0
}

// extractFieldNames 提取字段名称
func extractFieldNames(fields []FieldInfo) []string {
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
