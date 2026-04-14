package optimizer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// isInterfaceType 快速判断是否是接口类型
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

// isStructType 快速判断是否是结构体类型
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
	default:
		return false
	}
}

// collectStructs 收集所有需要处理的结构体（两阶段：快速扫描 + 按需加载）
func (o *Optimizer) collectStructs(pkgPath, structName, filePath string, depth, level int) {
	key := pkgPath + "." + structName

	// 快速去重
	o.mu.Lock()
	if _, ok := o.optimized[key]; ok {
		o.mu.Unlock()
		return
	}
	if o.collecting[key] {
		o.mu.Unlock()
		return
	}
	o.collecting[key] = true
	o.mu.Unlock()

	// 检查递归深度
	if depth > o.maxDepth {
		return
	}

	// 检查是否是第三方包
	if isVendorPackage(pkgPath) || !o.isProjectPackage(pkgPath) {
		return
	}

	// 阶段 1: 快速扫描文件查找结构体（不加载包）
	nestedFields, filePath, err := o.scanStructFields(pkgPath, structName, filePath)
	if err != nil {
		// 阶段 2: 加载包查找（慢速但准确）
		nestedFields, filePath, err = o.collectFromPackage(pkgPath, structName)
		if err != nil {
			o.Log(3, "查找结构体失败：%s.%s: %v", pkgPath, structName, err)
			return
		}
	}

	// 添加到队列
	task := &StructTask{
		PkgPath:    pkgPath,
		StructName: structName,
		FilePath:   filePath,
		Depth:      depth,
		Level:      level,
	}

	o.mu.Lock()
	o.structQueue = append(o.structQueue, task)
	o.mu.Unlock()

	// 递归收集嵌套结构体
	o.collectNestedFromFields(nestedFields, pkgPath, filePath, depth, level)
}

// scanStructFields 快速扫描文件中的结构体字段（不加载包）
func (o *Optimizer) scanStructFields(pkgPath, structName, filePath string) ([]nestedField, string, error) {
	// 确定搜索目录
	searchDir := o.getPackageDir(pkgPath)
	if searchDir == "" {
		return nil, "", fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	// 如果没有指定文件路径，查找包含结构体的文件
	if filePath == "" {
		files, err := o.findFilesWithStruct(searchDir, structName)
		if err != nil {
			return nil, "", err
		}
		if len(files) == 0 {
			return nil, "", fmt.Errorf("struct %s not found", structName)
		}
		filePath = files[0]
	}

	// 解析文件获取结构体字段
	return o.parseStructFields(filePath, structName, pkgPath)
}

// collectFromPackage 从包中收集结构体信息（加载包，慢速但准确）
func (o *Optimizer) collectFromPackage(pkgPath, structName string) ([]nestedField, string, error) {
	pkg, err := o.analyzer.LoadPackage(pkgPath)
	if err != nil {
		return nil, "", err
	}

	st, filePath, err := o.findStructInPackage(pkg, structName)
	if err != nil {
		return nil, "", err
	}

	// 提取嵌套字段信息
	var nestedFields []nestedField
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		fieldType := field.Type()

		if isInterfaceType(fieldType) {
			continue
		}

		pkg := o.getTypePkg(fieldType)
		typeName := o.getTypeName(fieldType)

		if isStructType(fieldType) && !isStandardLibraryPkg(pkg) && !isVendorPackage(pkg) {
			fieldPkg := pkg
			if fieldPkg == "" {
				fieldPkg = pkgPath
			}
			nestedFields = append(nestedFields, nestedField{
				Name:     typeName,
				PkgPath:  fieldPkg,
				IsStruct: true,
			})
		}
	}

	return nestedFields, filePath, nil
}

// nestedField 嵌套字段信息
type nestedField struct {
	Name     string
	PkgPath  string
	IsStruct bool
}

// parseStructFields 解析文件中的结构体字段
func (o *Optimizer) parseStructFields(filePath, structName, pkgPath string) ([]nestedField, string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, "", err
	}

	// 解析 import 映射
	importMap := o.parseImports(f, pkgPath)

	// 查找结构体
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

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return nil, "", fmt.Errorf("%s is not a struct", structName)
			}

			// 提取字段信息
			var nestedFields []nestedField
			for _, field := range st.Fields.List {
				fieldInfo := o.extractFieldInfo(field, importMap, pkgPath)
				if fieldInfo.IsStruct {
					nestedFields = append(nestedFields, fieldInfo)
				}
			}

			return nestedFields, filePath, nil
		}
	}

	return nil, "", fmt.Errorf("struct %s not found in file %s", structName, filePath)
}

// parseImports 解析文件的 import 映射
func (o *Optimizer) parseImports(f *ast.File, pkgPath string) map[string]string {
	importMap := make(map[string]string)

	// 添加当前包
	importMap[""] = pkgPath

	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}

		if alias != "" && alias != "_" && alias != "." {
			importMap[alias] = importPath
		} else {
			// 使用路径的最后一部分作为别名
			parts := strings.Split(importPath, "/")
			importMap[parts[len(parts)-1]] = importPath
		}
	}

	return importMap
}

// extractFieldInfo 提取字段信息
func (o *Optimizer) extractFieldInfo(field *ast.Field, importMap map[string]string, pkgPath string) nestedField {
	typeName, pkgAlias := o.extractTypeNameFromExpr(field.Type)

	fieldPkg := pkgPath
	if pkgAlias != "" {
		if p, ok := importMap[pkgAlias]; ok {
			fieldPkg = p
		}
	}

	return nestedField{
		Name:     typeName,
		PkgPath:  fieldPkg,
		IsStruct: true, // 假设是结构体，后续会验证
	}
}

// extractTypeNameFromExpr 从 AST 表达式中提取类型名称和包别名
func (o *Optimizer) extractTypeNameFromExpr(expr ast.Expr) (typeName, pkgAlias string) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, ""
	case *ast.StarExpr:
		return o.extractTypeNameFromExpr(t.X)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name, ident.Name
		}
		return t.Sel.Name, ""
	default:
		return "", ""
	}
}

// collectNestedFromFields 从字段信息收集中嵌套结构体
func (o *Optimizer) collectNestedFromFields(fields []nestedField, pkgPath, filePath string, depth, level int) {
	for _, field := range fields {
		if !field.IsStruct || isStandardLibraryPkg(field.PkgPath) || isVendorPackage(field.PkgPath) {
			continue
		}

		if o.isProjectPackage(field.PkgPath) {
			o.collectStructs(field.PkgPath, field.Name, filePath, depth+1, level+1)
		}
	}
}

// findFilesWithStruct 查找可能包含指定结构体的文件
func (o *Optimizer) findFilesWithStruct(dir, structName string) ([]string, error) {
	var result []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		filePath := filepath.Join(dir, name)

		// 快速检查文件是否包含结构体名称
		if o.fileContainsStruct(filePath, structName) {
			result = append(result, filePath)
		}
	}

	return result, nil
}

// fileContainsStruct 快速检查文件是否包含结构体定义
func (o *Optimizer) fileContainsStruct(filePath, structName string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	pattern := []byte("type " + structName + " struct")
	return bytes.Contains(data, pattern)
}

// findStructInPackage 在已加载的包中查找结构体
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

// getTypePkg 获取类型的包路径
func (o *Optimizer) getTypePkg(typ types.Type) string {
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
		return o.getTypePkg(t.Elem())
	case *types.Slice:
		return o.getTypePkg(t.Elem())
	case *types.Array:
		return o.getTypePkg(t.Elem())
	case *types.Map:
		return ""
	default:
		return ""
	}
}

// getTypeName 获取类型名称
func (o *Optimizer) getTypeName(typ types.Type) string {
	if typ == nil {
		return ""
	}
	switch t := typ.(type) {
	case *types.Named:
		return t.Obj().Name()
	case *types.Pointer:
		return o.getTypeName(t.Elem())
	case *types.Slice:
		return o.getTypeName(t.Elem())
	case *types.Array:
		return o.getTypeName(t.Elem())
	default:
		return typ.String()
	}
}
