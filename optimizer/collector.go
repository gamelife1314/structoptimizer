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

// collectStructs 收集所有需要处理的结构体（只解析文件，不加载包）
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

	// 检查文件路径是否包含应该跳过的目录
	if filePath != "" && o.shouldSkipDir(filePath) {
		o.Log(3, "跳过目录中的结构体：%s (文件：%s)", key, filePath)
		return
	}

	// 只解析文件，不加载包
	nestedFields, filePath, err := o.parseStructFromFileOnly(pkgPath, structName, filePath)
	if err != nil {
		// 只有真正的错误才记录（不是基本类型）
		if !strings.Contains(err.Error(), "struct ") {
			o.Log(2, "解析文件失败：%s.%s: %v", pkgPath, structName, err)
		}
		return
	}

	// 检查解析后的文件路径是否包含应该跳过的目录
	if filePath != "" && o.shouldSkipDir(filePath) {
		o.Log(3, "跳过目录中的结构体：%s (文件：%s)", key, filePath)
		return
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

	// 递归收集嵌套结构体（跨包分析）
	for _, field := range nestedFields {
		// 跳过标准库和第三方包
		if isStandardLibraryPkg(field.PkgPath) || isVendorPackage(field.PkgPath) {
			continue
		}

		// 检查包范围限制（前缀匹配）
		if o.config.PkgScope != "" && !strings.HasPrefix(field.PkgPath, o.config.PkgScope) {
			continue
		}

		if o.isProjectPackage(field.PkgPath) {
			// 无论是否跨包，都不传递 filePath
			// 因为嵌套结构体可能定义在同包的不同文件中
			o.collectStructs(field.PkgPath, field.Name, "", depth+1, level+1)
		}
	}
}

// parseStructFromFileOnly 只解析文件获取结构体信息（不加载包）
func (o *Optimizer) parseStructFromFileOnly(pkgPath, structName, filePath string) ([]nestedField, string, error) {
	// 确定搜索目录
	searchDir := o.getPackageDir(pkgPath)
	if searchDir == "" {
		return nil, "", fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	o.Log(3, "parseStructFromFileOnly: pkgPath=%s, searchDir=%s, structName=%s, filePath=%s", pkgPath, searchDir, structName, filePath)

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

// parseStructFields 解析文件中的结构体字段
func (o *Optimizer) parseStructFields(filePath, structName, pkgPath string) ([]nestedField, string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, "", err
	}

	// 解析 import 映射
	importMap := o.parseImports(f, pkgPath)

	// 查找结构体（支持 type xxx struct 和 type ( ... ) 两种形式）
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

	// 判断是否是结构体类型：
	// 1. 有包别名的是结构体（如 subpkg.SubStruct）
	// 2. 未导出类型（小写字母开头且不是基本类型）可能是结构体
	// 3. 基本类型（bool, int 等）不是结构体
	// 4. 同包内的导出类型（大写开头）需要在后续验证，这里先假设是结构体
	// 5. 排除已知的基本类型和内置类型
	isStruct := pkgAlias != "" || 
		(isUnexportedStructName(typeName) && !isBasicType(typeName)) ||
		(isExportedStructName(typeName) && !isBasicType(typeName))

	return nestedField{
		Name:     typeName,
		PkgPath:  fieldPkg,
		IsStruct: isStruct,
	}
}

// isExportedStructName 判断是否是导出的结构体类型名称（大写字母开头且不是基本类型）
func isExportedStructName(name string) bool {
	if name == "" {
		return false
	}
	firstChar := name[0]
	// 大写开头，可能是结构体
	if firstChar < 'A' || firstChar > 'Z' {
		return false
	}
	// 排除已知的非结构体类型（如接口、基本类型的别名等）
	// 这里简单处理，只要是大写开头且不是基本类型，就认为是结构体
	return true
}

// extractTypeNameFromExpr 从 AST 表达式中提取类型名称和包别名
// 支持处理指针、Slice、Array 等类型
func (o *Optimizer) extractTypeNameFromExpr(expr ast.Expr) (typeName, pkgAlias string) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, ""
	case *ast.StarExpr:
		return o.extractTypeNameFromExpr(t.X)
	case *ast.ArrayType:
		// 处理 []T 或 [N]T 类型
		return o.extractTypeNameFromExpr(t.Elt)
	case *ast.MapType:
		// 处理 map[K]V 类型，返回 value 的类型
		return o.extractTypeNameFromExpr(t.Value)
	case *ast.ChanType:
		// 处理 chan T 类型
		return o.extractTypeNameFromExpr(t.Value)
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

// findFilesWithStruct 查找可能包含指定结构体的文件（检查跳过目录和文件）
func (o *Optimizer) findFilesWithStruct(dir, structName string) ([]string, error) {
	var result []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// 检查是否应该跳过该目录（传递完整路径）
			dirPath := filepath.Join(dir, entry.Name())
			if o.shouldSkipDir(dirPath) {
				continue
			}
			// 递归搜索子目录
			subFiles, err := o.findFilesWithStruct(dirPath, structName)
			if err == nil {
				result = append(result, subFiles...)
			}
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		// 检查是否应该跳过该文件
		if o.shouldSkipFile(name) {
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

// shouldSkipDir 检查是否应该跳过该目录（支持路径匹配）
func (o *Optimizer) shouldSkipDir(dirPath string) bool {
	// 提取目录的 basename
	baseName := filepath.Base(dirPath)

	// 规范化路径分隔符（Windows 和 Unix 统一）
	normalizedPath := filepath.ToSlash(dirPath)

	for _, pattern := range o.config.SkipDirs {
		// 匹配 basename（向后兼容）
		if matched, _ := filepath.Match(pattern, baseName); matched {
			return true
		}
		// 检查路径中是否包含该目录名（作为完整路径组件）
		// 例如：pattern="datas" 匹配 "/do/datas/ele/" 或 "/do/datas"
		// 使用路径分割来确保匹配完整的目录名
		normalizedPattern := filepath.ToSlash(pattern)
		parts := strings.Split(normalizedPath, "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(normalizedPattern, part); matched {
				return true
			}
		}
	}
	return false
}

// shouldSkipFile 检查是否应该跳过该文件
func (o *Optimizer) shouldSkipFile(fileName string) bool {
	for _, pattern := range o.config.SkipFiles {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
	}
	return false
}

// fileContainsStruct 快速检查文件是否包含结构体定义（支持 type xxx struct 和 type ( ... ) 两种形式）
func (o *Optimizer) fileContainsStruct(filePath, structName string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// 匹配 type StructName struct 形式
	pattern1 := []byte("type " + structName + " struct")
	if bytes.Contains(data, pattern1) {
		return true
	}
	
	// 匹配 type ( ... StructName struct ... ) 形式
	// 查找 structName 后面紧跟 struct 关键字（中间只有空白字符）
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		// 查找 structName
		idx := bytes.Index(line, []byte(structName))
		if idx >= 0 {
			// 检查后面是否有 struct 关键字
			remaining := line[idx+len(structName):]
			// 跳过空白字符
			trimmed := bytes.TrimLeft(remaining, " \t\r")
			if bytes.HasPrefix(trimmed, []byte("struct")) {
				return true
			}
		}
	}
	
	return false
}

// nestedField 嵌套字段信息
type nestedField struct {
	Name     string
	PkgPath  string
	IsStruct bool
}
