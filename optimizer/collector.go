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

// isInterfaceType quickly checks if the type is an interface
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

// isStructType quickly checks if the type is a struct
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

// collectStructs collects all structs that need processing (parses files only, does not load packages)
func (o *Optimizer) collectStructs(pkgPath, structName, filePath string, depth, level int) {
	key := pkgPath + "." + structName

	// Dedup check: check optimized and collecting while holding the lock
	o.mu.Lock()
	if _, ok := o.optimized[key]; ok {
		o.mu.Unlock()
		return
	}
	if o.collecting[key] {
		o.mu.Unlock()
		return
	}
	// Mark as collecting to prevent other goroutines from processing duplicates
	o.collecting[key] = true
	o.mu.Unlock()

	// Check recursion depth
	if depth > o.maxDepth {
		return
	}

	// Check if it is a third-party package (cross-package scan allowed when AllowExternalPkgs=true)
	if !o.config.AllowExternalPkgs {
		if isVendorPackage(pkgPath) || !o.isProjectPackage(pkgPath) {
			return
		}
	}

	// Check if the file path contains directories that should be skipped
	if filePath != "" && o.shouldSkipDir(filePath) {
		o.Log(3, "跳过目录中的结构体：%s (文件：%s)", key, filePath)
		return
	}

	// Parse file only, do not load package
	nestedFields, filePath, err := o.parseStructFromFileOnly(pkgPath, structName, filePath)
	if err != nil {
		// Only log real errors (not basic types)
		if !strings.Contains(err.Error(), "struct ") {
			o.Log(2, "解析文件失败：%s.%s: %v", pkgPath, structName, err)
		}
		return
	}

	// Check if the resolved file path contains directories that should be skipped
	if filePath != "" && o.shouldSkipDir(filePath) {
		o.Log(3, "跳过目录中的结构体：%s (文件：%s)", key, filePath)
		return
	}

	// Add to queue
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

	// Recursively collect nested structs (cross-package analysis)
	for _, field := range nestedFields {
		// Always skip standard library
		if isStandardLibraryPkg(field.PkgPath) {
			continue
		}

		// Skip vendor package only when AllowExternalPkgs=false
		if !o.config.AllowExternalPkgs && isVendorPackage(field.PkgPath) {
			continue
		}

		// Check package scope restriction (skippable when AllowExternalPkgs=true)
		if !o.config.AllowExternalPkgs && o.config.PkgScope != "" && !strings.HasPrefix(field.PkgPath, o.config.PkgScope) {
			o.Log(3, "跳过跨包字段：%s (包：%s, 范围：%s)", field.Name, field.PkgPath, o.config.PkgScope)
			continue
		}

		if o.config.AllowExternalPkgs || o.isProjectPackage(field.PkgPath) {
			// Whether intra-package or cross-package, do not pass filePath.
			// Let parseStructFromFileOnly auto-locate the file containing the struct
			// via findFilesWithStruct. This correctly handles nested structs
			// defined in different files within the same package.
			o.collectStructs(field.PkgPath, field.Name, "", depth+1, level+1)
		}
	}
}

// parseStructFromFileOnly parses only the file to get struct info (does not load package)
func (o *Optimizer) parseStructFromFileOnly(pkgPath, structName, filePath string) ([]nestedField, string, error) {
	// Determine the search directory
	searchDir := o.getPackageDir(pkgPath)
	if searchDir == "" {
		return nil, "", fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	// If no file path specified, find files containing the struct
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

	// Parse file to get struct fields
	return o.parseStructFields(filePath, structName, pkgPath)
}

// parseStructFields parses the struct fields in a file
func (o *Optimizer) parseStructFields(filePath, structName, pkgPath string) ([]nestedField, string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, "", err
	}

	// Parse import mapping
	importMap := o.parseImports(f, pkgPath)

	// Get the package directory (used to find type definitions in other files of the same package)
	pkgDir := o.getPackageDir(pkgPath)

	// Find struct definition (supports both "type xxx struct" and "type ( ... )" forms)
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

			// Extract field info (pass pkgDir to support cross-file type lookup within the same package)
			var nestedFields []nestedField
			for _, field := range st.Fields.List {
				fieldInfo := o.extractFieldInfo(field, importMap, pkgPath, pkgDir)
				if fieldInfo.IsStruct {
					nestedFields = append(nestedFields, fieldInfo)
				}
			}

			return nestedFields, filePath, nil
		}
	}

	return nil, "", fmt.Errorf("struct %s not found in file %s", structName, filePath)
}

// parseImports parses the import mapping of a file
func (o *Optimizer) parseImports(f *ast.File, pkgPath string) map[string]string {
	importMap := make(map[string]string)

	// Add the current package
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
			// Use the last segment of the path as alias
			parts := strings.Split(importPath, "/")
			importMap[parts[len(parts)-1]] = importPath
		}
	}

	return importMap
}

// extractFieldInfo extracts field information
func (o *Optimizer) extractFieldInfo(field *ast.Field, importMap map[string]string, pkgPath string, pkgDir string) nestedField {
	typeName, pkgAlias := o.extractTypeNameFromExpr(field.Type)

	fieldPkg := pkgPath
	if pkgAlias != "" {
		if p, ok := importMap[pkgAlias]; ok {
			fieldPkg = p
		}
	}

	// Check if it is a struct
	isStruct := false

	// 1. First check if it is a basic type
	if !isBasicType(typeName) {
		// 2. For unexported types in the same package, scan package files to find the definition
		if fieldPkg == pkgPath && pkgDir != "" {
			isStruct = o.isStructTypeInPackage(pkgDir, typeName, pkgPath)
		} else if fieldPkg != pkgPath {
			// For cross-package, need to check if it's an interface type.
			// Interface types should not be optimized, so mark as non-struct.
			isStruct = !o.isInterfaceTypeCrossPackage(fieldPkg, typeName)
		}
	}

	return nestedField{
		Name:     typeName,
		PkgPath:  fieldPkg,
		IsStruct: isStruct,
	}
}

// isInterfaceTypeCrossPackage checks whether a cross-package type is an interface
func (o *Optimizer) isInterfaceTypeCrossPackage(pkgPath, typeName string) bool {
	// Get the package directory
	pkgDir := o.getPackageDir(pkgPath)
	if pkgDir == "" {
		// Unable to get package directory, conservatively assume not an interface
		return false
	}

	// Scan Go files in the package
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		if o.isInterfaceTypeInFile(filePath, typeName) {
			return true
		}
	}

	return false
}

// isInterfaceTypeInFile checks whether an interface type is defined in a file
func (o *Optimizer) isInterfaceTypeInFile(filePath, typeName string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return false
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

			// Check if it is an interface
			if _, ok := ts.Type.(*ast.InterfaceType); ok {
				return true
			}

			// Not an interface
			return false
		}
	}

	return false
}

// isStructTypeInPackage checks whether a type is a struct defined in the package
func (o *Optimizer) isStructTypeInPackage(pkgDir, typeName, pkgPath string) bool {
	// Find all Go files in the package
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		if o.isStructTypeInFile(filePath, typeName, pkgPath) {
			return true
		}
	}

	return false
}

// isStructTypeInFile checks whether a specific struct type is defined in a file
func (o *Optimizer) isStructTypeInFile(filePath, typeName, pkgPath string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return false
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

			// Check if it is a struct
			if _, ok := ts.Type.(*ast.StructType); ok {
				return true
			}

			// Not a struct
			return false
		}
	}

	return false
}

// extractTypeNameFromExpr extracts type name and package alias from an AST expression
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
	case *ast.ArrayType:
		// Array or slice: []Type or [10]Type
		elemName, elemAlias := o.extractTypeNameFromExpr(t.Elt)
		if t.Len != nil {
			// Fixed-length array
			return "[" + getArrayLengthString(t.Len) + "]" + elemName, elemAlias
		}
		return "[]" + elemName, elemAlias
	case *ast.MapType:
		// map[Key]Value
		keyName, _ := o.extractTypeNameFromExpr(t.Key)
		valueName, valueAlias := o.extractTypeNameFromExpr(t.Value)
		return "map[" + keyName + "]" + valueName, valueAlias
	case *ast.ChanType:
		// chan Type
		elemName, elemAlias := o.extractTypeNameFromExpr(t.Value)
		return "chan " + elemName, elemAlias
	case *ast.FuncType:
		// Function type
		return "func", ""
	case *ast.InterfaceType:
		// Interface type
		return "interface{}", ""
	case *ast.StructType:
		// Inline struct
		return "struct{}", ""
	default:
		return "", ""
	}
}

// getArrayLengthString returns the string representation of an array length
func getArrayLengthString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	default:
		return "?"
	}
}

// collectNestedFromFields collects nested structs from field info
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

// findFilesWithStruct searches for files that may contain the specified struct (checks skip dirs and files)
func (o *Optimizer) findFilesWithStruct(dir, structName string) ([]string, error) {
	var result []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if this directory should be skipped (pass full path)
			dirPath := filepath.Join(dir, entry.Name())
			if o.shouldSkipDir(dirPath) {
				continue
			}
			// Recursively search subdirectories
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

		// Check if this file should be skipped
		if o.shouldSkipFile(name) {
			continue
		}

		filePath := filepath.Join(dir, name)

		// Quickly check if the file contains the struct name
		if o.fileContainsStruct(filePath, structName) {
			result = append(result, filePath)
		}
	}

	return result, nil
}

// shouldSkipDir checks whether a directory should be skipped (supports path matching)
func (o *Optimizer) shouldSkipDir(dirPath string) bool {
	// Extract the directory basename
	baseName := filepath.Base(dirPath)

	// Normalize path separators (unify Windows and Unix)
	normalizedPath := filepath.ToSlash(dirPath)

	for _, pattern := range o.config.SkipDirs {
		// Match basename (backwards compatibility)
		if matched, _ := filepath.Match(pattern, baseName); matched {
			return true
		}
		// Check if the path contains the directory name as a complete path component.
		// For example: pattern="datas" matches "/do/datas/ele/" or "/do/datas"
		// Use path splitting to ensure we match complete directory names.
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

// shouldSkipFile checks whether a file should be skipped
func (o *Optimizer) shouldSkipFile(fileName string) bool {
	for _, pattern := range o.config.SkipFiles {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
	}
	return false
}

// fileContainsStruct quickly checks if a file contains a struct definition
// (supports both "type xxx struct" and "type ( ... )" forms)
func (o *Optimizer) fileContainsStruct(filePath, structName string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Match "type StructName struct" form
	pattern1 := []byte("type " + structName + " struct")
	if bytes.Contains(data, pattern1) {
		return true
	}

	// Match "type ( ... StructName struct ... )" form.
	// Look for structName followed by the struct keyword (only whitespace in between).
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		// Find structName
		idx := bytes.Index(line, []byte(structName))
		if idx >= 0 {
			// Check if struct keyword follows
			remaining := line[idx+len(structName):]
			// Skip whitespace
			trimmed := bytes.TrimLeft(remaining, " \t\r")
			if bytes.HasPrefix(trimmed, []byte("struct")) {
				return true
			}
		}
	}

	return false
}

// nestedField represents nested field information
type nestedField struct {
	Name     string
	PkgPath  string
	IsStruct bool
}
