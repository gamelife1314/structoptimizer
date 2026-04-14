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

// collectStructs 收集所有需要处理的结构体（不执行优化，只收集依赖）
func (o *Optimizer) collectStructs(pkgPath, structName, filePath string, depth, level int) {
	key := pkgPath + "." + structName

	// 快速去重：使用 map 代替 slice 遍历
	o.mu.Lock()
	// 检查是否已收集或正在收集
	if _, ok := o.optimized[key]; ok {
		o.mu.Unlock()
		return
	}
	if o.collecting[key] {
		o.mu.Unlock()
		return
	}
	// 标记为正在收集
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

	// 快速路径：通过文件扫描查找结构体（不加载包）
	st, filePath, err := o.findStructFast(pkgPath, structName)
	if err != nil {
		o.Log(3, "快速查找失败，加载包：%s.%s: %v", pkgPath, structName, err)
		// 慢速路径：加载包查找
		st, filePath, err = o.findStructInPackageSlow(pkgPath, structName)
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

	// 分析字段，收集嵌套结构体
	o.collectNestedStructs(st, structName, pkgPath, filePath, depth, level)
}

// findStructFast 快速查找结构体（只解析文件，不加载包）
func (o *Optimizer) findStructFast(pkgPath, structName string) (*types.Struct, string, error) {
	// 确定搜索目录
	searchDir := o.getPackageDir(pkgPath)
	if searchDir == "" {
		return nil, "", fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	// 查找包含结构体的文件
	files, err := o.findFilesWithStruct(searchDir, structName)
	if err != nil {
		return nil, "", err
	}

	// 解析找到的文件
	for _, filePath := range files {
		st, err := o.parseStructFromFile(filePath, structName)
		if err == nil && st != nil {
			return st, filePath, nil
		}
	}

	return nil, "", fmt.Errorf("struct %s not found", structName)
}

// findStructInPackageSlow 慢速路径：加载包查找结构体
func (o *Optimizer) findStructInPackageSlow(pkgPath, structName string) (*types.Struct, string, error) {
	pkg, err := o.analyzer.LoadPackage(pkgPath)
	if err != nil {
		return nil, "", err
	}

	return o.findStructInPackage(pkg, structName)
}

// findStructInPackage 在已加载的包中查找结构体
func (o *Optimizer) findStructInPackage(pkg *packages.Package, structName string) (*types.Struct, string, error) {
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		// 在文件中查找结构体定义
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

				// 获取类型信息
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

// collectNestedStructs 收集嵌套的结构体依赖
func (o *Optimizer) collectNestedStructs(st *types.Struct, structName, pkgPath, filePath string, depth, level int) {
	if st == nil {
		return
	}

	// 遍历字段
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		fieldType := field.Type()

		// 跳过接口、标准库、第三方包
		if isInterfaceType(fieldType) {
			continue
		}

		pkg := o.getTypePkg(fieldType)
		if isStandardLibraryPkg(pkg) || isVendorPackage(pkg) || !o.isProjectPackage(pkg) {
			continue
		}

		// 检查是否是结构体类型
		if isStructType(fieldType) {
			typeName := o.getTypeName(fieldType)
			fieldPkg := pkg
			if fieldPkg == "" {
				fieldPkg = pkgPath
			}

			if fieldPkg != "" && o.isProjectPackage(fieldPkg) {
				// 递归收集，层级 +1
				o.collectStructs(fieldPkg, typeName, filePath, depth+1, level+1)
			}
		}
	}
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
		// Map 的包路径通常不重要
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

// findFilesWithStruct 查找可能包含指定结构体的文件
func (o *Optimizer) findFilesWithStruct(dir, structName string) ([]string, error) {
	var result []string

	// 读取目录
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

// fileContainsStruct 快速检查文件是否包含结构体定义（不解析）
func (o *Optimizer) fileContainsStruct(filePath, structName string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// 简单字符串匹配：查找 "type StructName struct"
	pattern := []byte("type " + structName + " struct")
	return bytes.Contains(data, pattern)
}

// parseStructFromFile 从文件中解析结构体（简化版）
func (o *Optimizer) parseStructFromFile(filePath, structName string) (*types.Struct, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if ts.Name.Name == structName {
				if st, ok := ts.Type.(*ast.StructType); ok {
					return o.createSimpleStruct(st, fset), nil
				}
			}
		}
	}

	return nil, fmt.Errorf("struct %s not found in file %s", structName, filePath)
}

// createSimpleStruct 创建简化的结构体
func (o *Optimizer) createSimpleStruct(astStruct *ast.StructType, fset *token.FileSet) *types.Struct {
	var fields []*types.Var

	for _, field := range astStruct.Fields.List {
		var fieldNames []*ast.Ident
		if field.Names != nil {
			fieldNames = field.Names
		}

		fieldType := types.Typ[types.Invalid]

		for _, name := range fieldNames {
			fieldVar := types.NewField(name.Pos(), nil, name.Name, fieldType, false)
			fields = append(fields, fieldVar)
		}

		if len(fieldNames) == 0 {
			typeName := o.extractTypeName(field.Type)
			if typeName != "" {
				fieldVar := types.NewField(field.Pos(), nil, typeName, fieldType, true)
				fields = append(fields, fieldVar)
			}
		}
	}

	return types.NewStruct(fields, nil)
}

// extractTypeName 从 AST 提取类型名称
func (o *Optimizer) extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return o.extractTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}
