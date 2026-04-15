package optimizer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
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

	st, fields := extractFieldsFromAST(foundDecl, fset)
	info.Fields = fields
	info.OrigOrder = extractFieldNames(fields)
	info.OrigSize = CalcStructSizeFromFields(fields)

	return info, st, nil
}

// extractFieldsFromAST 从 AST 提取字段信息
func extractFieldsFromAST(ts *ast.TypeSpec, fset *token.FileSet) (*types.Struct, []FieldInfo) {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, nil
	}

	var fields []FieldInfo
	var varFields []*types.Var

	for _, f := range st.Fields.List {
		typeName := extractTypeName(f.Type)
		size, align := estimateFieldSize(f.Type)
		
		// 判断是否是匿名字段
		isEmbed := len(f.Names) == 0

		fi := FieldInfo{
			Name:     getFieldName(f),
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
		varFields = append(varFields, types.NewField(f.Pos(), nil, fi.Name, types.Typ[types.Invalid], false))
	}

	// 创建简化的 types.Struct
	return types.NewStruct(varFields, nil), fields
}

// extractTypeName 从 AST 提取类型名称
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + extractTypeName(t.X)
	case *ast.SelectorExpr:
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
		elemSize, elemAlign := estimateFieldSize(t.Elt)
		return elemSize * 1, elemAlign // 假设长度为 1
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

// extractFieldNames 提取字段名称
func extractFieldNames(fields []FieldInfo) []string {
	var names []string
	for _, f := range fields {
		if f.Name != "" {
			names = append(names, f.Name)
		}
	}
	return names
}
