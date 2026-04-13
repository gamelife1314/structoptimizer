package analyzer

import (
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
	Verbose       int
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

	// 检测是否是 GOPATH 项目
	isGoMod := false
	if _, err := os.Stat(a.config.TargetDir + "/go.mod"); err == nil {
		isGoMod = true
	} else {
		// 检查父目录是否有 go.mod
		dir := a.config.TargetDir
		for dir != "/" && dir != "." {
			dir = filepath.Dir(dir)
			if _, err := os.Stat(dir + "/go.mod"); err == nil {
				isGoMod = true
				break
			}
		}
	}

	// 构建环境
	env := os.Environ()
	var loadDir string
	if !isGoMod {
		// GOPATH 模式
		env = append(env, "GO111MODULE=off")
		loadDir = "" // GOPATH 模式下，使用 GOPATH 查找包
		a.Log(1, "使用 GOPATH 模式加载包：%s (GOPATH=%s)", pkgPath, os.Getenv("GOPATH"))
	} else {
		loadDir = a.config.TargetDir
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
func (a *Analyzer) FindStructByName(pkgPath, structName string) (*types.Struct, string, error) {
	pkg, err := a.LoadPackage(pkgPath)
	if err != nil {
		return nil, "", err
	}

	a.pkg = pkg
	a.info = pkg.TypesInfo

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

				if typeSpec.Name.Name != structName {
					continue
				}

				_, ok = typeSpec.Type.(*ast.StructType)
				if !ok {
					return nil, "", fmt.Errorf("%s is not a struct", structName)
				}

				// 获取类型信息
				typ := pkg.TypesInfo.TypeOf(typeSpec.Type)
				if typ == nil {
					return nil, "", fmt.Errorf("failed to get type info for %s", structName)
				}

				st, ok := typ.Underlying().(*types.Struct)
				if !ok {
					return nil, "", fmt.Errorf("%s is not a struct", structName)
				}

				return st, filePath, nil
			}
		}
	}

	return nil, "", fmt.Errorf("struct %s not found in package %s", structName, pkgPath)
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
	dir := filepath.Base(filepath.Dir(filePath))

	// 检查目录跳过
	for _, pattern := range a.config.SkipDirs {
		if matched, _ := filepath.Match(pattern, dir); matched {
			return true
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
		prefix := ""
		switch level {
		case 1:
			prefix = "[INFO] "
		case 2:
			prefix = "[DEBUG] "
		case 3:
			prefix = "[TRACE] "
		}
		fmt.Printf(prefix+format+"\n", args...)
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
