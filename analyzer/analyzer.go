package analyzer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"
)

// Analyzer 分析器（参考 gopls 设计）
type Analyzer struct {
	config     *Config
	fset       *token.FileSet
	info       *types.Info
	pkg        *packages.Package
	pkgMap     map[string]*packages.Package // 已加载的包缓存（线程安全）
	loadedPkgs map[string]bool
	mu         sync.RWMutex                 // 保护包缓存
	structIndex map[string]*StructLocation // 结构体位置索引（包路径。结构体名 -> 文件路径）
}

// StructLocation 结构体位置信息
type StructLocation struct {
	PkgPath  string
	FileName string
	Loaded   bool // 是否已加载包
}

// Config 分析器配置
type Config struct {
	TargetDir     string
	StructName    string // 结构体全名（包路径。结构体名）
	Package       string // 包路径
	SourceFile    string
	SkipDirs      []string
	SkipFiles     []string
	SkipByMethods []string
	SkipByNames   []string
	Verbose       int
	ProjectType   string // 项目类型：gomod 或 gopath
	GOPATH        string // GOPATH 路径（可选，为空则使用环境变量）
}

// NewAnalyzer 创建分析器
func NewAnalyzer(cfg *Config) *Analyzer {
	return &Analyzer{
		config:      cfg,
		fset:        token.NewFileSet(),
		pkgMap:      make(map[string]*packages.Package),
		loadedPkgs:  make(map[string]bool),
		structIndex: make(map[string]*StructLocation),
	}
}

// BuildStructIndex 构建结构体索引（参考 gopls 的文件扫描）
// 快速扫描所有文件，建立结构体位置索引，不加载包
func (a *Analyzer) BuildStructIndex() error {
	a.Log(1, "构建结构体索引...")
	start := time.Now()

	// 确定搜索目录
	searchDir := a.config.TargetDir
	if searchDir == "" {
		searchDir = "."
	}

	// 扫描所有 Go 文件
	err := a.scanDirectory(searchDir)
	if err != nil {
		return err
	}

	a.Log(1, "索引构建完成：共 %d 个结构体，耗时 %v", len(a.structIndex), time.Since(start))
	return nil
}

// scanDirectory 递归扫描目录
func (a *Analyzer) scanDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// 跳过 vendor 和.git 等目录
			name := entry.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" {
				continue
			}
			if err := a.scanDirectory(filepath.Join(dir, name)); err != nil {
				return err
			}
			continue
		}

		// 只处理.go 文件
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		filePath := filepath.Join(dir, name)
		if err := a.scanFile(filePath); err != nil {
			a.Log(2, "扫描文件失败：%v", err)
			continue // 继续处理其他文件
		}
	}

	return nil
}

// scanFile 扫描单个文件，提取结构体定义
func (a *Analyzer) scanFile(filePath string) error {
	// 快速检查：文件是否包含 "type.*struct"
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if !bytes.Contains(data, []byte("type ")) || !bytes.Contains(data, []byte(" struct")) {
		return nil // 文件不包含结构体定义
	}

	// 解析文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// 提取包路径
	pkgPath := a.extractPkgPath(f, filePath)

	// 提取结构体
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

// extractPkgPath 从文件中提取包路径
func (a *Analyzer) extractPkgPath(f *ast.File, filePath string) string {
	// 尝试从 import 推断完整包路径
	if a.config.TargetDir != "" {
		// Go Module 模式：从文件路径推断
		relPath, err := filepath.Rel(a.config.TargetDir, filepath.Dir(filePath))
		if err == nil && relPath != "." {
			modulePath := a.getModulePath()
			if modulePath != "" {
				return modulePath + "/" + filepath.ToSlash(relPath)
			}
		}
	}

	// 使用包名作为后备
	if f.Name != nil {
		return f.Name.Name
	}

	return "unknown"
}

// getModulePath 获取模块路径
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

// LoadPackage 加载包（按需加载，不递归加载依赖包）
func (a *Analyzer) LoadPackage(pkgPath string) (*packages.Package, error) {
	// 检查缓存（读锁）
	a.mu.RLock()
	if pkg, ok := a.pkgMap[pkgPath]; ok {
		a.mu.RUnlock()
		return pkg, nil
	}
	a.mu.RUnlock()

	// 检查是否已加载
	a.mu.RLock()
	if a.loadedPkgs[pkgPath] {
		a.mu.RUnlock()
		return nil, fmt.Errorf("package already loaded: %s", pkgPath)
	}
	a.mu.RUnlock()

	// 按需加载：只解析当前包的文件，不加载依赖包
	return a.loadPackageOnDemand(pkgPath)
}

// loadPackageOnDemand 按需加载包（只解析当前包文件，不递归加载依赖）
func (a *Analyzer) loadPackageOnDemand(pkgPath string) (*packages.Package, error) {
	// 获取包目录
	pkgDir := a.getPackageDir(pkgPath)
	if pkgDir == "" {
		return nil, fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	// 查找所有 Go 文件（排除测试文件）
	goFiles, err := a.findGoFiles(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("查找 Go 文件失败：%v", err)
	}

	if len(goFiles) == 0 {
		a.mu.Lock()
		a.loadedPkgs[pkgPath] = true
		a.mu.Unlock()
		return nil, fmt.Errorf("包中没有 Go 文件：%s", pkgPath)
	}

	a.Log(2, "按需加载包：%s（%d 个文件，不加载依赖）", pkgPath, len(goFiles))

	// 解析所有 Go 文件
	var astFiles []*ast.File
	fset := token.NewFileSet()
	for _, f := range goFiles {
		astFile, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
		if err != nil {
			a.Log(2, "解析文件失败：%s (%v)", f, err)
			continue // 跳过有问题的文件
		}
		astFiles = append(astFiles, astFile)
	}

	if len(astFiles) == 0 {
		a.mu.Lock()
		a.loadedPkgs[pkgPath] = true
		a.mu.Unlock()
		return nil, fmt.Errorf("没有可解析的 Go 文件：%s", pkgPath)
	}

	// 类型检查（只检查当前包，不加载依赖包的完整信息）
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	conf := types.Config{
		Importer: a.newVendorImporter(), // 使用支持 vendor 的 importer
		Sizes:    types.SizesFor("gc", runtime.GOARCH),
		Error: func(err error) {
			// 只记录详细日志，不中断处理
			if a.config.Verbose >= 3 {
				a.Log(3, "类型检查信息：%s (%v)", pkgPath, err)
			}
		},
	}

	typesPkg, err := conf.Check(pkgPath, fset, astFiles, info)
	if err != nil {
		// Check 返回错误但仍然可能创建了部分类型信息
		if typesPkg == nil {
			a.mu.Lock()
			a.loadedPkgs[pkgPath] = true
			a.mu.Unlock()
			return nil, fmt.Errorf("类型检查失败：%v", err)
		}
		// 有部分类型信息，继续处理
		a.Log(2, "类型检查警告：%s (%v)", pkgPath, err)
	}

	// 构建 packages.Package（兼容现有的接口）
	pkg := &packages.Package{
		ID:              pkgPath,
		Name:            typesPkg.Name(),
		PkgPath:         pkgPath,
		GoFiles:         goFiles,
		CompiledGoFiles: goFiles,
		Syntax:          astFiles,
		Types:           typesPkg,
		TypesInfo:       info,
		TypesSizes:      conf.Sizes,
		Fset:            fset,
	}

	// 缓存包
	a.mu.Lock()
	a.pkgMap[pkgPath] = pkg
	a.loadedPkgs[pkgPath] = true
	a.mu.Unlock()

	return pkg, nil
}

// findGoFiles 查找目录中的所有 Go 文件（排除测试文件）
func (a *Analyzer) findGoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 只处理 .go 文件，排除测试文件和生成文件
		if strings.HasSuffix(name, ".go") &&
			!strings.HasSuffix(name, "_test.go") &&
			!strings.HasPrefix(name, "zz_generated") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	return files, nil
}

// FindStructByName 查找指定名称的结构体
// 优化：先尝试快速查找（不加载包），失败后再加载包
func (a *Analyzer) FindStructByName(pkgPath, structName string) (*types.Struct, string, error) {
	// 快速路径：如果包已加载，直接从缓存查找
	if pkg, ok := a.pkgMap[pkgPath]; ok {
		return a.findStructInLoadedPackage(pkg, structName)
	}

	// 快速路径：直接解析文件查找结构体（不加载包）
	st, filePath, err := a.findStructFast(pkgPath, structName)
	if err == nil {
		return st, filePath, nil
	}

	// 慢速路径：加载整个包再查找
	a.Log(3, "快速查找失败，加载包：%s", pkgPath)
	pkg, err := a.LoadPackage(pkgPath)
	if err != nil {
		return nil, "", err
	}

	a.pkg = pkg
	a.info = pkg.TypesInfo

	return a.findStructInLoadedPackage(pkg, structName)
}

// findStructFast 快速查找结构体（不加载包，只解析文件）
func (a *Analyzer) findStructFast(pkgPath, structName string) (*types.Struct, string, error) {
	// 确定搜索目录
	searchDir := a.getPackageDir(pkgPath)
	if searchDir == "" {
		return nil, "", fmt.Errorf("无法确定包目录：%s", pkgPath)
	}

	// 查找包含结构体的文件
	files, err := a.findFilesWithStruct(searchDir, structName)
	if err != nil {
		return nil, "", err
	}

	// 解析找到的文件
	for _, filePath := range files {
		st, err := a.parseStructFromFile(filePath, structName)
		if err == nil && st != nil {
			return st, filePath, nil
		}
	}

	return nil, "", fmt.Errorf("struct %s not found", structName)
}

// findStructInLoadedPackage 在已加载的包中查找结构体
func (a *Analyzer) findStructInLoadedPackage(pkg *packages.Package, structName string) (*types.Struct, string, error) {
	// 遍历包中的所有文件
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		// 在文件中查找结构体
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

				// 查找对应的类型信息
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

// getPackageDir 获取包目录
func (a *Analyzer) getPackageDir(pkgPath string) string {
	// 根据项目类型决定查找方式
	if a.config.ProjectType == "gopath" {
		// GOPATH 模式
		
		// 先检查是否是 vendor 中的包
		// vendor 包路径示例：github.com/xxx/yyy
		// 应该在项目的 vendor 目录中查找
		if a.config.Package != "" && pkgPath != a.config.Package {
			// 不同包，可能是 vendor 中的依赖
			// 查找项目根目录
			projectRoot := a.findProjectRootForPackage()
			if projectRoot != "" {
				vendorPath := filepath.Join(projectRoot, "vendor", pkgPath)
				if info, err := os.Stat(vendorPath); err == nil && info.IsDir() {
					a.Log(3, "找到 vendor 中的包：%s", vendorPath)
					return vendorPath
				}
			}
		}
		
		// 标准 GOPATH 模式：在 $GOPATH/src 中查找
		gopath := a.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			return filepath.Join(gopath, "src", pkgPath)
		}
		return ""
	}

	// Go Module 模式
	if a.config.TargetDir != "" {
		relPath := strings.TrimPrefix(pkgPath, a.getModulePath())
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			return filepath.Join(a.config.TargetDir, relPath)
		}
		return a.config.TargetDir
	}

	return ""
}

// findProjectRootForPackage 查找项目根目录（包含 vendor 的目录）
func (a *Analyzer) findProjectRootForPackage() string {
	// 直接获取主包目录（不触发 vendor 查找）
	gopath := a.config.GOPATH
	if gopath == "" {
		gopath = os.Getenv("GOPATH")
	}
	if gopath == "" || a.config.Package == "" {
		return ""
	}
	
	pkgDir := filepath.Join(gopath, "src", a.config.Package)
	if pkgDir == "" {
		return ""
	}

	// 向上查找，直到 GOPATH/src 或找到 vendor 目录
	dir := pkgDir
	gopathSrc := filepath.Join(gopath, "src")

	for dir != "" && dir != gopathSrc && len(dir) > len(gopathSrc) {
		// 检查是否有 vendor 目录
		vendorDir := filepath.Join(dir, "vendor")
		if info, err := os.Stat(vendorDir); err == nil && info.IsDir() {
			return dir
		}
		
		// 向上移动一级
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// findFilesWithStruct 查找可能包含指定结构体的文件
func (a *Analyzer) findFilesWithStruct(dir, structName string) ([]string, error) {
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
		if a.fileContainsStruct(filePath, structName) {
			result = append(result, filePath)
		}
	}

	return result, nil
}

// fileContainsStruct 快速检查文件是否包含结构体定义（不解析）
func (a *Analyzer) fileContainsStruct(filePath, structName string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// 简单字符串匹配：查找 "type StructName struct"
	pattern := []byte("type " + structName + " struct")
	return bytes.Contains(data, pattern)
}

// parseStructFromFile 从文件中解析结构体
func (a *Analyzer) parseStructFromFile(filePath, structName string) (*types.Struct, error) {
	// 解析文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// 查找结构体定义
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
					// 创建简化的结构体（用于收集依赖）
					result, _ := a.createSimpleStruct(st, fset)
					return result, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("struct %s not found in file %s", structName, filePath)
}

// createSimpleStruct 创建简化的结构体（用于快速收集依赖）
func (a *Analyzer) createSimpleStruct(astStruct *ast.StructType, fset *token.FileSet) (*types.Struct, error) {
	var fields []*types.Var

	for _, field := range astStruct.Fields.List {
		var fieldNames []*ast.Ident
		if field.Names != nil {
			fieldNames = field.Names
		}

		// 创建占位符类型
		fieldType := types.Typ[types.Invalid]

		for _, name := range fieldNames {
			fieldVar := types.NewField(name.Pos(), nil, name.Name, fieldType, false)
			fields = append(fields, fieldVar)
		}

		// 匿名字段
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

// extractTypeName 从 AST 类型表达式中提取类型名称
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

// FindAllStructs 查找包中的所有结构体
type StructDef struct {
	Name    string
	PkgPath string
	File    string
	Type    *types.Struct
}

func (a *Analyzer) FindAllStructs(pkgPath string) ([]StructDef, error) {
	pkg, err := a.LoadPackage(pkgPath)
	if err != nil {
		return nil, err
	}

	a.pkg = pkg
	a.info = pkg.TypesInfo

	var structs []StructDef

	// 遍历包中的所有文件
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		// 如果指定了源文件，只在该文件中查找
		if a.config.SourceFile != "" {
			// 检查文件路径是否匹配（使用 basename 匹配）
			baseName := filepath.Base(filePath)
			if baseName != a.config.SourceFile {
				continue
			}
		}

		// 检查是否应该跳过
		if a.shouldSkipFile(filePath) {
			continue
		}

		// 在文件中查找结构体
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

				// 获取类型信息
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

// HasMethod 检查结构体是否有指定方法
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

// HasAnyMethod 检查结构体是否有任一指定方法
func (a *Analyzer) HasAnyMethod(structType *types.Named, methodNames []string) bool {
	for _, name := range methodNames {
		if a.HasMethod(structType, name) {
			return true
		}
	}
	return false
}

// GetStructMethods 获取结构体的所有方法名
func (a *Analyzer) GetStructMethods(structType *types.Named) []string {
	var methods []string
	for i := 0; i < structType.NumMethods(); i++ {
		methods = append(methods, structType.Method(i).Name())
	}
	return methods
}

// IsExternalPackage 判断是否是外部包（非当前项目）
func (a *Analyzer) IsExternalPackage(pkgPath string) bool {
	// 标准库
	if !strings.Contains(pkgPath, ".") {
		return true
	}

	// 检查是否在项目内
	if a.config.TargetDir != "" {
		// 尝试加载包来判断
		if pkg, ok := a.pkgMap[pkgPath]; ok {
			return !strings.HasPrefix(pkg.GoFiles[0], a.config.TargetDir)
		}
	}

	return false
}

// shouldSkipFile 检查是否应该跳过文件
func (a *Analyzer) shouldSkipFile(filePath string) bool {
	if filePath == "" {
		return false
	}

	name := filepath.Base(filePath)

	// 检查目录跳过（检查路径中的所有目录组件）
	for _, pattern := range a.config.SkipDirs {
		// 遍历文件路径的所有目录组件
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

	// 检查文件跳过
	for _, pattern := range a.config.SkipFiles {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	return false
}

// ParseStructName 解析结构体全名
func ParseStructName(fullName string) (pkgPath, structName string) {
	lastDot := strings.LastIndex(fullName, ".")
	if lastDot == -1 {
		return "", fullName
	}
	return fullName[:lastDot], fullName[lastDot+1:]
}

// LoadAndParseFile 加载并解析单个文件
func (a *Analyzer) LoadAndParseFile(filePath string) (*ast.File, *types.Info, error) {
	// 解析文件
	f, err := parser.ParseFile(a.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	// 类型检查
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

// Log 日志输出
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

// GetTypesInfo 获取类型信息
func (a *Analyzer) GetTypesInfo() *types.Info {
	return a.info
}

// GetFset 获取文件集
func (a *Analyzer) GetFset() *token.FileSet {
	return a.fset
}

// LoadPackages 批量加载包（参考 gopls 的批量加载优化）
func (a *Analyzer) LoadPackages(pkgPaths []string) error {
	if len(pkgPaths) == 0 {
		return nil
	}

	a.Log(1, "批量加载 %d 个包...", len(pkgPaths))
	start := time.Now()

	// 过滤已加载的包
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

	// 根据项目类型构建环境
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

	// 构建配置
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Env:  env,
		Dir:  loadDir,
	}

	// 批量加载
	pkgs, err := packages.Load(cfg, toLoad...)
	if err != nil {
		return err
	}

	// 缓存结果
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

// GetStructIndex 获取结构体索引
func (a *Analyzer) GetStructIndex() map[string]*StructLocation {
	return a.structIndex
}

// FindStructByIndex 通过索引查找结构体（不需要加载包）
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

// vendorImporter 支持 vendor 目录的导入器
// GOPATH 模式下，依赖查找顺序：
// 1. 当前项目 vendor 目录（$GOPATH/src/mycompany/myproject/vendor）
// 2. 父级项目 vendor 目录（向上查找）
// 3. 标准库
// 4. GOPATH 中的其他包
type vendorImporter struct {
	analyzer       *Analyzer
	projectRoot    string              // 项目根目录（用于查找 vendor）
	sourceImporter types.Importer     // 源码导入器
	cache          map[string]*types.Package // 缓存已导入的包
}

// newVendorImporter 创建支持 vendor 目录的导入器
func (a *Analyzer) newVendorImporter() *vendorImporter {
	// 确定项目根目录
	// GOPATH 模式下，项目根目录是 $GOPATH/src/ 下的第一级目录
	projectRoot := a.findProjectRoot()

	return &vendorImporter{
		analyzer:       a,
		projectRoot:    projectRoot,
		sourceImporter: importer.For("source", nil),
		cache:          make(map[string]*types.Package),
	}
}

// findProjectRoot 查找项目根目录（包含 vendor 目录的目录）
func (a *Analyzer) findProjectRoot() string {
	if a.config.ProjectType != "gopath" {
		return a.config.TargetDir
	}

	// GOPATH 模式下，从包目录向上查找包含 vendor 的目录
	pkgDir := a.getPackageDir(a.config.Package)
	if pkgDir == "" {
		return ""
	}

	// 向上查找，直到 GOPATH/src 或找到 vendor 目录
	dir := pkgDir
	gopathSrc := ""
	
	gopath := a.config.GOPATH
	if gopath == "" {
		gopath = os.Getenv("GOPATH")
	}
	if gopath != "" {
		gopathSrc = filepath.Join(gopath, "src")
	}

	for dir != "" && dir != gopathSrc && len(dir) > len(gopathSrc) {
		// 检查是否有 vendor 目录
		vendorDir := filepath.Join(dir, "vendor")
		if info, err := os.Stat(vendorDir); err == nil && info.IsDir() {
			a.Log(3, "找到项目根目录（包含 vendor）：%s", dir)
			return dir
		}
		
		// 向上移动一级
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// 没找到 vendor，返回包目录的父目录
	return filepath.Dir(pkgDir)
}

func (vi *vendorImporter) Import(path string) (*types.Package, error) {
	// 检查缓存
	if pkg, ok := vi.cache[path]; ok {
		return pkg, nil
	}

	// 1. 尝试从 vendor 目录导入
	if vi.projectRoot != "" && path != "" {
		vendorPkgPath := filepath.Join(vi.projectRoot, "vendor", path)
		if info, err := os.Stat(vendorPkgPath); err == nil && info.IsDir() {
			// vendor 目录中有这个包，从源码导入
			if pkg, err := vi.importFromDir(path, vendorPkgPath); err == nil {
				vi.cache[path] = pkg
				return pkg, nil
			}
			vi.analyzer.Log(3, "从 vendor 目录导入失败：%s (%v)", vendorPkgPath, err)
		}
	}

	// 2. 尝试标准库（不包含 "." 的包路径）
	if !strings.Contains(path, ".") || isGoStdLib(path) {
		if pkg, err := vi.sourceImporter.Import(path); err == nil {
			vi.cache[path] = pkg
			return pkg, nil
		}
	}

	// 3. 尝试从 GOPATH 导入（其他项目）
	if vi.analyzer.config.ProjectType == "gopath" {
		gopath := vi.analyzer.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			gopathPkgPath := filepath.Join(gopath, "src", path)
			if info, err := os.Stat(gopathPkgPath); err == nil && info.IsDir() {
				if pkg, err := vi.importFromDir(path, gopathPkgPath); err == nil {
					vi.cache[path] = pkg
					return pkg, nil
				}
			}
		}
	}

	// 4. 都失败了
	return nil, fmt.Errorf("无法导入包 %s", path)
}

// importFromDir 从指定目录导入包
func (vi *vendorImporter) importFromDir(pkgPath, dir string) (*types.Package, error) {
	// 查找目录中的所有 Go 文件
	goFiles, err := vi.analyzer.findGoFiles(dir)
	if err != nil {
		return nil, err
	}

	if len(goFiles) == 0 {
		return nil, fmt.Errorf("目录中没有 Go 文件：%s", dir)
	}

	// 解析文件
	var astFiles []*ast.File
	fset := token.NewFileSet()
	for _, f := range goFiles {
		astFile, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
		if err != nil {
			continue // 跳过有问题的文件
		}
		astFiles = append(astFiles, astFile)
	}

	if len(astFiles) == 0 {
		return nil, fmt.Errorf("没有可解析的 Go 文件：%s", dir)
	}

	// 类型检查
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
	}

	conf := types.Config{
		Importer: vi, // 递归使用当前导入器
		Sizes:    types.SizesFor("gc", runtime.GOARCH),
	}

	pkg, err := conf.Check(pkgPath, fset, astFiles, info)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// fallbackImporter 回退导入器
// 当找不到 .a 文件时，尝试从源码导入
type fallbackImporter struct {
	defaultImporter types.Importer
	sourceImporter  types.Importer
}

func newFallbackImporter() *fallbackImporter {
	return &fallbackImporter{
		defaultImporter: importer.Default(),
		sourceImporter:  importer.For("source", nil),
	}
}

func (fi *fallbackImporter) Import(path string) (*types.Package, error) {
	// 先尝试使用默认 importer（查找 .a 文件）
	pkg, err := fi.defaultImporter.Import(path)
	if err == nil {
		return pkg, nil
	}

	// 默认 importer 失败，尝试从源码导入
	// 只尝试导入标准库，避免递归加载所有依赖
	if !strings.Contains(path, ".") || isGoStdLib(path) {
		// 标准库包（不包含 "." 或者是已知的标准库路径）
		if srcPkg, srcErr := fi.sourceImporter.Import(path); srcErr == nil {
			return srcPkg, nil
		}
	}

	// 都失败了，返回错误但不影响主流程
	// 这样类型检查仍然可以进行，只是缺少部分导入包的信息
	return nil, err
}

// isGoStdLib 判断是否是 Go 标准库
func isGoStdLib(pkgPath string) bool {
	// 标准库包路径不包含 "." 且不包含 "/"
	// 或者以 "github.com/", "golang.org/" 等开头的是第三方库
	if strings.Contains(pkgPath, ".") {
		return false
	}
	// 常见的第三方库前缀
	if strings.HasPrefix(pkgPath, "github.com/") ||
		strings.HasPrefix(pkgPath, "golang.org/") ||
		strings.HasPrefix(pkgPath, "google.golang.org/") ||
		strings.HasPrefix(pkgPath, "gopkg.in/") {
		return false
	}
	return true
}
