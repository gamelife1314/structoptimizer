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

// estimateFieldSizeWithLookup 估算字段大小（带类型查找）
func estimateFieldSizeWithLookup(expr ast.Expr, pkgDir string) (size, align int64) {
	switch t := expr.(type) {
	case *ast.Ident:
		// 对于标识符，尝试在同包中查找类型定义
		if pkgDir != "" {
			underlyingKind := findTypeUnderlyingInPackage(pkgDir, t.Name)
			if underlyingKind != types.Invalid {
				return basicSize(underlyingKind)
			}
		}
		return sizeOfIdent(t.Name)
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
	default:
		return 8, 8
	}
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
