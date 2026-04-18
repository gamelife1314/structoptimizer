package optimizer

import (
	"fmt"
	"go/ast"
	"os"
	"os/exec"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"sync"
)

// MethodIndex 缓存包的方法集
// 结构: map[包路径]map[结构体名]map[方法名]bool
type MethodIndex struct {
	mu    sync.RWMutex
	cache map[string]map[string]map[string]bool
}

// NewMethodIndex 创建方法索引器
func NewMethodIndex() *MethodIndex {
	return &MethodIndex{
		cache: make(map[string]map[string]map[string]bool),
	}
}

// HasMethod 检查结构体是否有指定方法（支持通配符）
func (mi *MethodIndex) HasMethod(pkgPath, structName, methodPattern string) bool {
	mi.mu.RLock()
	pkgCache, pkgExists := mi.cache[pkgPath]
	mi.mu.RUnlock()

	if !pkgExists {
		if err := mi.indexPkg(pkgPath); err != nil {
			// 索引失败，保守起见返回 false
			return false
		}
		mi.mu.RLock()
		pkgCache = mi.cache[pkgPath]
		mi.mu.RUnlock()
	}

	if pkgCache == nil {
		return false
	}

	structMethods, structExists := pkgCache[structName]
	if !structExists {
		return false
	}

	// 检查方法匹配
	for methodName := range structMethods {
		if mi.matchMethod(methodName, methodPattern) {
			return true
		}
	}
	return false
}

// matchMethod 匹配方法名（支持通配符）
func (mi *MethodIndex) matchMethod(methodName, pattern string) bool {
	// 完全匹配
	if methodName == pattern {
		return true
	}
	
	// 通配符匹配
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		matched, _ := filepath.Match(pattern, methodName)
		if matched {
			return true
		}
	}
	
	return false
}

// indexPkg 扫描包目录构建索引
func (mi *MethodIndex) indexPkg(pkgPath string) error {
	// 获取包目录
	dir, err := mi.getPkgDir(pkgPath)
	if err != nil {
		return err
	}

	if dir == "" {
		return fmt.Errorf("无法获取包目录：%s", pkgPath)
	}

	// 扫描文件
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}

	// 确保包缓存存在
	mi.mu.Lock()
	if _, ok := mi.cache[pkgPath]; !ok {
		mi.cache[pkgPath] = make(map[string]map[string]bool)
	}
	pkgCache := mi.cache[pkgPath]
	mi.mu.Unlock()

	fset := token.NewFileSet()
	for _, file := range files {
		// 忽略测试文件
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		for _, decl := range f.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			// 提取接收者类型
			recvType := extractRecvType(funcDecl.Recv.List[0].Type)
			if recvType == "" {
				continue
			}

			methodName := funcDecl.Name.Name

			mi.mu.Lock()
			if _, ok := pkgCache[recvType]; !ok {
				pkgCache[recvType] = make(map[string]bool)
			}
			pkgCache[recvType][methodName] = true
			mi.mu.Unlock()
		}
	}

	return nil
}

// extractRecvType 从接收者 AST 中提取类型名称
func extractRecvType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// getPkgDir 使用 go list 获取包目录
func (mi *MethodIndex) getPkgDir(pkgPath string) (string, error) {
	// 尝试 1: 使用 go list（Go Modules 模式）
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", pkgPath)
	// 不要修改环境，使用当前项目的模块模式
	out, err := cmd.CombinedOutput()
	if err == nil {
		dir := strings.TrimSpace(string(out))
		if dir != "" {
			// 验证目录是否存在
			if _, err := os.Stat(dir); err == nil {
				return dir, nil
			}
		}
	}

	// 尝试 2: GOPATH 模式手动解析
	return mi.getPkgDirFromGOPATH(pkgPath)
}

// getPkgDirFromGOPATH 从 GOPATH 解析包路径
func (mi *MethodIndex) getPkgDirFromGOPATH(pkgPath string) (string, error) {
	// 获取 GOPATH
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// 默认 GOPATH
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法获取用户主目录")
		}
		gopath = filepath.Join(home, "go")
	}
	
	// GOPATH 可能有多个路径，用分隔符分割
	gopaths := filepath.SplitList(gopath)
	
	// 在每个 GOPATH 的 src 目录下查找
	for _, gp := range gopaths {
		srcDir := filepath.Join(gp, "src")
		pkgDir := filepath.Join(srcDir, pkgPath)
		
		// 验证目录是否存在
		if _, err := os.Stat(pkgDir); err == nil {
			return pkgDir, nil
		}
	}
	
	return "", fmt.Errorf("在 GOPATH 中未找到包：%s", pkgPath)
}
