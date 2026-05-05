package analyzer

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
	"sync"
	"time"

	"golang.org/x/tools/go/packages"
)

// Analyzer (modeled after gopls design)
type Analyzer struct {
	config      *Config
	fset        *token.FileSet
	info        *types.Info
	pkg         *packages.Package
	pkgMap      map[string]*packages.Package // loaded package cache (thread-safe)
	loadedPkgs  map[string]bool
	mu          sync.RWMutex               // protects the package cache
	structIndex map[string]*StructLocation // struct location index (packagePath.structName -> filePath)
}

// StructLocation holds struct location info
type StructLocation struct {
	PkgPath  string
	FileName string
	Loaded   bool // whether the package has been loaded
}

// Config holds analyzer configuration
type Config struct {
	TargetDir     string
	StructName    string // struct full name (packagePath.structName)
	Package       string // package path
	SourceFile    string
	SkipDirs      []string
	SkipFiles     []string
	SkipByMethods []string
	SkipByNames   []string
	Verbose       int
	ProjectType   string // project type: gomod or gopath
	GOPATH        string // GOPATH path (optional, uses env var if empty)
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(cfg *Config) *Analyzer {
	return &Analyzer{
		config:      cfg,
		fset:        token.NewFileSet(),
		pkgMap:      make(map[string]*packages.Package),
		loadedPkgs:  make(map[string]bool),
		structIndex: make(map[string]*StructLocation),
	}
}

// BuildStructIndex builds the struct index (modeled after gopls file scanning)
// Quickly scans all files to build the struct location index without loading packages
func (a *Analyzer) BuildStructIndex() error {
	a.Log(1, "构建结构体索引...")
	start := time.Now()

	// Determine the search directory
	searchDir := a.config.TargetDir
	if searchDir == "" {
		searchDir = "."
	}

	// Scan all Go files
	err := a.scanDirectory(searchDir)
	if err != nil {
		return err
	}

	a.Log(1, "索引构建完成：共 %d 个结构体，耗时 %v", len(a.structIndex), time.Since(start))
	return nil
}

// scanDirectory recursively scans directories
func (a *Analyzer) scanDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip vendor, .git, etc. directories
			name := entry.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" {
				continue
			}
			if err := a.scanDirectory(filepath.Join(dir, name)); err != nil {
				return err
			}
			continue
		}

		// Only process .go files
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		filePath := filepath.Join(dir, name)
		if err := a.scanFile(filePath); err != nil {
			a.Log(2, "扫描文件失败：%v", err)
			continue // continue processing other files
		}
	}

	return nil
}

// scanFile scans a single file and extracts struct definitions
func (a *Analyzer) scanFile(filePath string) error {
	// Quick check: whether the file contains "type.*struct"
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if !bytes.Contains(data, []byte("type ")) || !bytes.Contains(data, []byte(" struct")) {
		return nil // file contains no struct definitions
	}

	// Parse the file
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Extract the package path
	pkgPath := a.extractPkgPath(f, filePath)

	// Extract structs
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

			if _, ok := ts.Type.(*ast.StructType); ok {
				key := pkgPath + "." + ts.Name.Name
				a.structIndex[key] = &StructLocation{
					PkgPath:  pkgPath,
					FileName: filePath,
					Loaded:   false,
				}
			}
		}
	}

	return nil
}

// extractPkgPath extracts the package path from a file
func (a *Analyzer) extractPkgPath(f *ast.File, filePath string) string {
	// Try to infer the full package path from imports
	if a.config.TargetDir != "" {
		// Go Module mode: infer from file path
		relPath, err := filepath.Rel(a.config.TargetDir, filepath.Dir(filePath))
		if err == nil && relPath != "." {
			modulePath := a.getModulePath()
			if modulePath != "" {
				return modulePath + "/" + filepath.ToSlash(relPath)
			}
		}
	}

	// Fallback to using the package name
	if f.Name != nil {
		return f.Name.Name
	}

	return "unknown"
}

// getModulePath retrieves the module path
func (a *Analyzer) getModulePath() string {
	if a.config.TargetDir == "" {
		return ""
	}

	goModPath := filepath.Join(a.config.TargetDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}

	return ""
}

// LoadPackage loads a package (thread-safe, with error handling)
func (a *Analyzer) LoadPackage(pkgPath string) (*packages.Package, error) {
	a.mu.Lock()
	if pkg, ok := a.pkgMap[pkgPath]; ok {
		a.mu.Unlock()
		return pkg, nil
	}
	if a.loadedPkgs[pkgPath] {
		a.mu.Unlock()
		return nil, fmt.Errorf("package already loaded: %s", pkgPath)
	}
	a.loadedPkgs[pkgPath] = true
	a.mu.Unlock()

	// Build the environment based on project type
	isGoMod := a.config.ProjectType != "gopath"

	// Build environment
	env := os.Environ()
	var loadDir string
	if !isGoMod {
		// GOPATH mode
		env = append(env, "GO111MODULE=off")

		// Use configured GOPATH or environment variable
		gopath := a.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			env = append(env, "GOPATH="+gopath)
		}

		loadDir = "" // GOPATH mode, look up packages via GOPATH
		a.Log(1, "使用 GOPATH 模式加载包：%s (GOPATH=%s)", pkgPath, gopath)
	} else {
		// Go Module mode
		loadDir = a.config.TargetDir
		a.Log(1, "使用 Go Module 模式加载包：%s", pkgPath)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  loadDir,
		Fset: a.fset,
		Env:  env,
	}

	// Load the package
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		// Mark as loaded (avoid retry)
		a.mu.Lock()
		a.loadedPkgs[pkgPath] = true
		a.mu.Unlock()
		return nil, fmt.Errorf("load package %s failed: %v", pkgPath, err)
	}

	if len(pkgs) == 0 {
		// Mark as loaded
		a.mu.Lock()
		a.loadedPkgs[pkgPath] = true
		a.mu.Unlock()
		return nil, fmt.Errorf("package not found: %s", pkgPath)
	}

	if len(pkgs) > 1 {
		// Mark as loaded
		a.mu.Lock()
		a.loadedPkgs[pkgPath] = true
		a.mu.Unlock()
		return nil, fmt.Errorf("multiple packages found: %s", pkgPath)
	}

	pkg := pkgs[0]

	// Package has errors, log but still cache (avoid retry)
	if len(pkg.Errors) > 0 {
		a.Log(2, "包 %s 有错误：%v", pkgPath, pkg.Errors)
		a.mu.Lock()
		a.pkgMap[pkgPath] = pkg
		a.loadedPkgs[pkgPath] = true
		a.mu.Unlock()
		return pkg, nil // return the package without error
	}

	// Cache the package (write lock)
	a.mu.Lock()
	a.pkgMap[pkgPath] = pkg
	a.loadedPkgs[pkgPath] = true
	a.mu.Unlock()

	return pkg, nil
}

// FindStructByName finds a struct by name
// Optimization: try fast lookup first (no package load), fall back to loading
func (a *Analyzer) FindStructByName(pkgPath, structName string) (*types.Struct, string, error) {
	// Fast path: if package already loaded, look up from cache
	if pkg, ok := a.pkgMap[pkgPath]; ok {
		return a.findStructInLoadedPackage(pkg, structName)
	}

	// Fast path: parse the file directly without loading the package
	st, filePath, err := a.findStructFast(pkgPath, structName)
	if err == nil {
		return st, filePath, nil
	}

	// Slow path: load the full package, then search
	a.Log(3, "快速查找失败，加载包：%s", pkgPath)
	pkg, err := a.LoadPackage(pkgPath)
	if err != nil {
		return nil, "", err
	}

	a.pkg = pkg
	a.info = pkg.TypesInfo

	return a.findStructInLoadedPackage(pkg, structName)
}

// findStructFast performs a fast struct lookup (no package load, parse file only)
func (a *Analyzer) findStructFast(pkgPath, structName string) (*types.Struct, string, error) {
	// Determine the search directory
	searchDir := a.getPackageDir(pkgPath)
	if searchDir == "" {
		return nil, "", fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	// Find files containing the struct
	files, err := a.findFilesWithStruct(searchDir, structName)
	if err != nil {
		return nil, "", err
	}

	// Parse the found files
	for _, filePath := range files {
		st, err := a.parseStructFromFile(filePath, structName)
		if err == nil && st != nil {
			return st, filePath, nil
		}
	}

	return nil, "", fmt.Errorf("struct %s not found", structName)
}

// findStructInLoadedPackage looks up a struct in a loaded package
func (a *Analyzer) findStructInLoadedPackage(pkg *packages.Package, structName string) (*types.Struct, string, error) {
	// Iterate over all files in the package
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		// Look for the struct in the file
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

				// Look up the corresponding type info
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

// getPackageDir returns the directory for a package
func (a *Analyzer) getPackageDir(pkgPath string) string {
	if a.config.TargetDir != "" {
		// Go Module mode
		relPath := strings.TrimPrefix(pkgPath, a.getModulePath())
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			return filepath.Join(a.config.TargetDir, relPath)
		}
		return a.config.TargetDir
	}

	// GOPATH mode
	gopath := a.config.GOPATH
	if gopath == "" {
		gopath = os.Getenv("GOPATH")
	}
	if gopath != "" {
		return filepath.Join(gopath, "src", pkgPath)
	}

	return ""
}

// findFilesWithStruct finds files that may contain the given struct
func (a *Analyzer) findFilesWithStruct(dir, structName string) ([]string, error) {
	var result []string

	// Read the directory
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

		// Quick check: does the file contain the struct name
		if a.fileContainsStruct(filePath, structName) {
			result = append(result, filePath)
		}
	}

	return result, nil
}

// fileContainsStruct quickly checks if a file contains a struct definition (no parsing)
func (a *Analyzer) fileContainsStruct(filePath, structName string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Simple string match: look for "type StructName struct"
	pattern := []byte("type " + structName + " struct")
	return bytes.Contains(data, pattern)
}

// parseStructFromFile parses a struct from a file
func (a *Analyzer) parseStructFromFile(filePath, structName string) (*types.Struct, error) {
	// Parse the file
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Find the struct definition
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
					// Create a simplified struct (for dependency collection)
					result, _ := a.createSimpleStruct(st, fset)
					return result, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("struct %s not found in file %s", structName, filePath)
}

// createSimpleStruct creates a simplified struct (for fast dependency collection)
func (a *Analyzer) createSimpleStruct(astStruct *ast.StructType, fset *token.FileSet) (*types.Struct, error) {
	var fields []*types.Var

	for _, field := range astStruct.Fields.List {
		var fieldNames []*ast.Ident
		if field.Names != nil {
			fieldNames = field.Names
		}

		// Create placeholder type
		fieldType := types.Typ[types.Invalid]

		for _, name := range fieldNames {
			fieldVar := types.NewField(name.Pos(), nil, name.Name, fieldType, false)
			fields = append(fields, fieldVar)
		}

		// Anonymous (embedded) field
		if len(fieldNames) == 0 {
			typeName := a.extractTypeName(field.Type)
			if typeName != "" {
				fieldVar := types.NewField(field.Pos(), nil, typeName, fieldType, true)
				fields = append(fields, fieldVar)
			}
		}
	}

	return types.NewStruct(fields, nil), nil
}

// extractTypeName extracts a type name from an AST type expression
func (a *Analyzer) extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return a.extractTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

// StructDef holds the definition of a struct found during scanning
type StructDef struct {
	Name    string
	PkgPath string
	File    string
	Type    *types.Struct
}

// FindAllStructs finds all structs in the specified package
func (a *Analyzer) FindAllStructs(pkgPath string) ([]StructDef, error) {
	pkg, err := a.LoadPackage(pkgPath)
	if err != nil {
		return nil, err
	}

	a.pkg = pkg
	a.info = pkg.TypesInfo

	var structs []StructDef

	// Iterate over all files in the package
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		// If a source file is specified, search only in that file
		if a.config.SourceFile != "" {
			// Check if the file path matches (using basename)
			baseName := filepath.Base(filePath)
			if baseName != a.config.SourceFile {
				continue
			}
		}

		// Check whether this file should be skipped
		if a.shouldSkipFile(filePath) {
			continue
		}

		// Look for structs in the file
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

				_, ok = typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				// Get type information
				typ := pkg.TypesInfo.TypeOf(typeSpec.Type)
				if typ == nil {
					continue
				}

				st, ok := typ.Underlying().(*types.Struct)
				if !ok {
					continue
				}

				structs = append(structs, StructDef{
					Name:    typeSpec.Name.Name,
					PkgPath: pkgPath,
					File:    filePath,
					Type:    st,
				})
			}
		}
	}

	return structs, nil
}

// FindAllStructsRecursive recursively finds structs in a package and all sub-packages
func (a *Analyzer) FindAllStructsRecursive(rootPkgPath string) ([]StructDef, error) {
	a.Log(1, "递归扫描包：%s 及其所有子包", rootPkgPath)

	// Collect all sub-package paths
	var allPkgPaths []string
	visited := make(map[string]bool)

	// Use BFS to traverse all imported packages
	queue := []string{rootPkgPath}
	visited[rootPkgPath] = true

	for len(queue) > 0 {
		currentPkg := queue[0]
		queue = queue[1:]

		// Load the current package to get its imports
		pkg, err := a.LoadPackage(currentPkg)
		if err != nil {
			a.Log(2, "加载包 %s 失败：%v", currentPkg, err)
			// Even if loading fails, add to the scan list
			allPkgPaths = append(allPkgPaths, currentPkg)
			continue
		}

		// Add current package to the scan list
		allPkgPaths = append(allPkgPaths, currentPkg)

		// Get all imported packages
		for _, imp := range pkg.Imports {
			impPath := imp.PkgPath
			if !visited[impPath] {
				// Only add project-internal packages (skip stdlib and vendor)
				if a.isProjectPackage(impPath) && !isVendorPackage(impPath) {
					// Check if it is under the root package path
					if strings.HasPrefix(impPath, rootPkgPath+"/") {
						visited[impPath] = true
						queue = append(queue, impPath)
					}
				}
			}
		}
	}

	a.Log(2, "找到 %d 个子包", len(allPkgPaths))

	// Collect structs from all packages
	var allStructs []StructDef
	for _, pkgPath := range allPkgPaths {
		structs, err := a.FindAllStructs(pkgPath)
		if err != nil {
			a.Log(2, "扫描包 %s 失败：%v", pkgPath, err)
			continue
		}
		allStructs = append(allStructs, structs...)
		a.Log(3, "包 %s: 找到 %d 个结构体", pkgPath, len(structs))
	}

	a.Log(1, "递归扫描完成：共找到 %d 个结构体", len(allStructs))
	return allStructs, nil
}

// isProjectPackage checks whether a package is internal to the project
func (a *Analyzer) isProjectPackage(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}
	if isVendorPackage(pkgPath) {
		return false
	}
	if isStandardLibrary(pkgPath) {
		return false
	}
	return true
}

// isVendorPackage checks whether a package is in vendor or is a third-party package
func isVendorPackage(pkgPath string) bool {
	// 1. Empty package path (usually stdlib or built-in types)
	if pkgPath == "" {
		return true
	}

	// 2. Check if the path contains a vendor directory
	if strings.Contains(pkgPath, "/vendor/") || strings.HasPrefix(pkgPath, "vendor/") {
		return true
	}

	return false
}

// isStandardLibrary checks whether a package is part of the Go standard library
func isStandardLibrary(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	// Standard library packages do not contain dots
	if strings.Contains(pkgPath, ".") {
		return false
	}
	return true
}

// HasMethod checks whether a struct has the specified method
func (a *Analyzer) HasMethod(structType *types.Named, methodName string) bool {
	if structType == nil {
		return false
	}

	for i := 0; i < structType.NumMethods(); i++ {
		method := structType.Method(i)
		if method.Name() == methodName {
			return true
		}
	}
	return false
}

// HasAnyMethod checks whether a struct has any of the specified methods
func (a *Analyzer) HasAnyMethod(structType *types.Named, methodNames []string) bool {
	for _, name := range methodNames {
		if a.HasMethod(structType, name) {
			return true
		}
	}
	return false
}

// GetStructMethods returns all method names of a struct
func (a *Analyzer) GetStructMethods(structType *types.Named) []string {
	var methods []string
	for i := 0; i < structType.NumMethods(); i++ {
		methods = append(methods, structType.Method(i).Name())
	}
	return methods
}

// IsExternalPackage checks whether a package is external (not in the current project)
func (a *Analyzer) IsExternalPackage(pkgPath string) bool {
	// Standard library
	if !strings.Contains(pkgPath, ".") {
		return true
	}

	// Check if it is within the project
	if a.config.TargetDir != "" {
		// Try loading the package to determine
		if pkg, ok := a.pkgMap[pkgPath]; ok && len(pkg.GoFiles) > 0 {
			return !strings.HasPrefix(pkg.GoFiles[0], a.config.TargetDir)
		}
	}

	return false
}

// shouldSkipFile checks whether a file should be skipped
func (a *Analyzer) shouldSkipFile(filePath string) bool {
	if filePath == "" {
		return false
	}

	name := filepath.Base(filePath)

	// Check directory skip (check all directory components in the path)
	for _, pattern := range a.config.SkipDirs {
		// Walk all directory components of the file path
		dir := filePath
		for dir != "" && dir != "." {
			base := filepath.Base(dir)
			if matched, _ := filepath.Match(pattern, base); matched {
				return true
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Check file skip
	for _, pattern := range a.config.SkipFiles {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	return false
}

// ParseStructName parses a struct's fully qualified name
func ParseStructName(fullName string) (pkgPath, structName string) {
	lastDot := strings.LastIndex(fullName, ".")
	if lastDot == -1 {
		return "", fullName
	}
	return fullName[:lastDot], fullName[lastDot+1:]
}

// LoadAndParseFile loads and parses a single file
func (a *Analyzer) LoadAndParseFile(filePath string) (*ast.File, *types.Info, error) {
	// Parse the file
	f, err := parser.ParseFile(a.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	// Type check
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	conf := types.Config{Importer: nil}
	pkg, err := conf.Check("", a.fset, []*ast.File{f}, info)
	if err != nil && a.config.Verbose >= 2 {
		fmt.Fprintf(os.Stderr, "type check warning: %v\n", err)
	}

	a.info = info
	_ = pkg

	return f, info, nil
}

// Log emits a log line
func (a *Analyzer) Log(level int, format string, args ...interface{}) {
	if level <= a.config.Verbose {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		levelPrefix := ""
		switch level {
		case 1:
			levelPrefix = "[INFO] "
		case 2:
			levelPrefix = "[DEBUG]"
		case 3:
			levelPrefix = "[TRACE]"
		}
		fmt.Printf("%s %s "+format+"\n", append([]interface{}{timestamp, levelPrefix}, args...)...)
	}
}

// GetTypesInfo returns type information
func (a *Analyzer) GetTypesInfo() *types.Info {
	return a.info
}

// GetFset returns the file set
func (a *Analyzer) GetFset() *token.FileSet {
	return a.fset
}

// LoadPackages loads multiple packages in batch (modeled after gopls batch loading)
func (a *Analyzer) LoadPackages(pkgPaths []string) error {
	if len(pkgPaths) == 0 {
		return nil
	}

	a.Log(1, "批量加载 %d 个包...", len(pkgPaths))
	start := time.Now()

	// Filter already-loaded packages
	var toLoad []string
	for _, pkgPath := range pkgPaths {
		a.mu.RLock()
		_, cached := a.pkgMap[pkgPath]
		loaded := a.loadedPkgs[pkgPath]
		a.mu.RUnlock()

		if !cached && !loaded {
			toLoad = append(toLoad, pkgPath)
		}
	}

	if len(toLoad) == 0 {
		a.Log(2, "所有包已缓存")
		return nil
	}

	// Build the environment based on project type
	isGoMod := a.config.ProjectType != "gopath"
	env := os.Environ()
	var loadDir string
	if !isGoMod {
		env = append(env, "GO111MODULE=off")
		gopath := a.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			env = append(env, "GOPATH="+gopath)
		}
		loadDir = ""
	} else {
		loadDir = a.config.TargetDir
	}

	// Build config
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Env: env,
		Dir: loadDir,
	}

	// Batch load
	pkgs, err := packages.Load(cfg, toLoad...)
	if err != nil {
		return err
	}

	// Cache results
	a.mu.Lock()
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			a.Log(2, "包 %s 有错误：%v", pkg.PkgPath, pkg.Errors)
		}
		a.pkgMap[pkg.PkgPath] = pkg
		a.loadedPkgs[pkg.PkgPath] = true
	}
	a.mu.Unlock()

	a.Log(1, "批量加载完成：加载 %d 个包，耗时 %v", len(pkgs), time.Since(start))
	return nil
}

// GetStructIndex returns the struct index
func (a *Analyzer) GetStructIndex() map[string]*StructLocation {
	return a.structIndex
}

// FindStructByIndex looks up a struct via the index (no package load needed)
func (a *Analyzer) FindStructByIndex(pkgPath, structName string) (*StructLocation, error) {
	key := pkgPath + "." + structName
	a.mu.RLock()
	loc, ok := a.structIndex[key]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("struct %s not found in index", key)
	}

	return loc, nil
}
