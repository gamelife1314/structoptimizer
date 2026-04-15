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
		fmt.Printf("[DEBUG] MethodIndex indexPkg 失败 pkg=%s, err=%v\n", pkgPath, err)
		return err
	}

	if dir == "" {
		fmt.Printf("[DEBUG] MethodIndex indexPkg 空目录 pkg=%s\n", pkgPath)
		return fmt.Errorf("无法获取包目录：%s", pkgPath)
	}

	fmt.Printf("[DEBUG] MethodIndex indexPkg pkg=%s dir=%s\n", pkgPath, dir)

	// 扫描文件
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}

	fmt.Printf("[DEBUG] MethodIndex 找到 %d 个文件\n", len(files))

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
			
			fmt.Printf("[DEBUG] MethodIndex 添加方法: %s.%s\n", recvType, methodName)
		}
	}

	fmt.Printf("[DEBUG] MethodIndex 索引完成，包 %s 有 %d 个结构体\n", pkgPath, len(pkgCache))
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
	// 使用 go list 获取目录，支持 Module 和 GOPATH
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", pkgPath)
	
	// 继承当前环境变量，确保 GOPATH 和 GO111MODULE 正确传递
	cmd.Env = os.Environ()
	
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[DEBUG] MethodIndex go list 失败 pkg=%s, err=%v, out=%s\n", pkgPath, err, string(out))
		return "", err
	}
	
	dir := strings.TrimSpace(string(out))
	fmt.Printf("[DEBUG] MethodIndex go list 结果 pkg=%s dir=%s\n", pkgPath, dir)
	
	if dir == "" {
		return "", fmt.Errorf("go list 返回空目录：%s", pkgPath)
	}

	// 验证目录是否存在
	if _, err := os.Stat(dir); err != nil {
		fmt.Printf("[DEBUG] MethodIndex 目录不存在 dir=%s\n", dir)
		return "", err
	}

	return dir, nil
}
