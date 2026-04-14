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
	"time"

	"golang.org/x/tools/go/packages"
)

// Analyzer 分析器
type Analyzer struct {
	config     *Config
	fset       *token.FileSet
	info       *types.Info
	pkg        *packages.Package
	pkgMap     map[string]*packages.Package // 已加载的包缓存
	loadedPkgs map[string]bool
}

// Config 分析器配置
type Config struct {
	TargetDir     string
	StructName    string // 结构体全名（包路径。结构体名）
	Package       string // 包路径
	SourceFile    string
	SkipDirs      []string
	SkipFiles     []string
	SkipPatterns  []string
	SkipByMethods []string
	SkipByNames   []string
	Verbose       int
	ProjectType   string // 项目类型：gomod 或 gopath
	GOPATH        string // GOPATH 路径（可选，为空则使用环境变量）
}

// NewAnalyzer 创建分析器
func NewAnalyzer(cfg *Config) *Analyzer {
	return &Analyzer{
		config:     cfg,
		fset:       token.NewFileSet(),
		pkgMap:     make(map[string]*packages.Package),
		loadedPkgs: make(map[string]bool),
	}
}

// LoadPackage 加载包
func (a *Analyzer) LoadPackage(pkgPath string) (*packages.Package, error) {
	// 检查缓存
	if pkg, ok := a.pkgMap[pkgPath]; ok {
		return pkg, nil
	}

	// 检查是否已加载
	if a.loadedPkgs[pkgPath] {
		return nil, fmt.Errorf("package already loaded: %s", pkgPath)
	}

	// 根据项目类型构建环境
	isGoMod := a.config.ProjectType != "gopath"

	// 构建环境
	env := os.Environ()
	var loadDir string
	if !isGoMod {
		// GOPATH 模式
		env = append(env, "GO111MODULE=off")
		
		// 使用配置的 GOPATH 或环境变量
		gopath := a.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			env = append(env, "GOPATH="+gopath)
		}
		
		loadDir = "" // GOPATH 模式下，使用 GOPATH 查找包
		a.Log(1, "使用 GOPATH 模式加载包：%s (GOPATH=%s)", pkgPath, gopath)
	} else {
		// Go Module 模式
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

	// 加载包
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("package not found: %s", pkgPath)
	}

	if len(pkgs) > 1 {
		return nil, fmt.Errorf("multiple packages found: %s", pkgPath)
	}

	pkg := pkgs[0]

	if len(pkg.Errors) > 0 {
		return pkg, fmt.Errorf("package has errors: %v", pkg.Errors)
	}

	a.pkgMap[pkgPath] = pkg
	a.loadedPkgs[pkgPath] = true

	return pkg, nil
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

// getPackageDir 获取包所在的目录
func (a *Analyzer) getPackageDir(pkgPath string) string {
	if a.config.TargetDir != "" {
		// Go Module 模式
		relPath := strings.TrimPrefix(pkgPath, a.getModulePath())
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			return filepath.Join(a.config.TargetDir, relPath)
		}
		return a.config.TargetDir
	}

	// GOPATH 模式
	gopath := a.config.GOPATH
	if gopath == "" {
		gopath = os.Getenv("GOPATH")
	}
	if gopath != "" {
		return filepath.Join(gopath, "src", pkgPath)
	}

	return ""
}

// getModulePath 获取模块路径（从 go.mod）
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

	// 检查通用跳过模式
	for _, pattern := range a.config.SkipPatterns {
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
