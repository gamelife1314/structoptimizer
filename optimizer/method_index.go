package optimizer

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MethodIndex caches method sets for packages.
// Structure: map[package_path]map[struct_name]map[method_name]bool
type MethodIndex struct {
	mu       sync.RWMutex
	cache    map[string]map[string]map[string]bool
	dirCache map[string]string // cached package dir lookups
}

// NewMethodIndex creates a new method indexer
func NewMethodIndex() *MethodIndex {
	return &MethodIndex{
		cache:    make(map[string]map[string]map[string]bool),
		dirCache: make(map[string]string),
	}
}

// HasMethod checks if a struct has the specified method (supports wildcards)
func (mi *MethodIndex) HasMethod(pkgPath, structName, methodPattern string) bool {
	mi.mu.RLock()
	pkgCache, pkgExists := mi.cache[pkgPath]
	mi.mu.RUnlock()

	if !pkgExists {
		if err := mi.indexPkg(pkgPath); err != nil {
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

	for methodName := range structMethods {
		if matchMethod(methodName, methodPattern) {
			return true
		}
	}
	return false
}

// indexPkg scans the package directory to build the index
func (mi *MethodIndex) indexPkg(pkgPath string) error {
	dir, err := mi.getPkgDirCached(pkgPath)
	if err != nil {
		return err
	}
	if dir == "" {
		return fmt.Errorf("cannot get package directory: %s", pkgPath)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}

	mi.mu.Lock()
	if _, ok := mi.cache[pkgPath]; !ok {
		mi.cache[pkgPath] = make(map[string]map[string]bool)
	}
	pkgCache := mi.cache[pkgPath]
	mi.mu.Unlock()

	// Build method map locally, then merge under single lock
	localCache := make(map[string]map[string]bool)
	fset := token.NewFileSet()
	for _, file := range files {
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
			recvType := extractRecvType(funcDecl.Recv.List[0].Type)
			if recvType == "" {
				continue
			}
			methodName := funcDecl.Name.Name
			if _, ok := localCache[recvType]; !ok {
				localCache[recvType] = make(map[string]bool)
			}
			localCache[recvType][methodName] = true
		}
	}

	// Merge local results under a single lock
	mi.mu.Lock()
	for recvType, methods := range localCache {
		if _, ok := pkgCache[recvType]; !ok {
			pkgCache[recvType] = make(map[string]bool)
		}
		for methodName := range methods {
			pkgCache[recvType][methodName] = true
		}
	}
	mi.mu.Unlock()

	return nil
}

// extractRecvType extracts the type name from a receiver AST
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

// getPkgDirCached returns the package directory with caching
func (mi *MethodIndex) getPkgDirCached(pkgPath string) (string, error) {
	mi.mu.RLock()
	if dir, ok := mi.dirCache[pkgPath]; ok {
		mi.mu.RUnlock()
		return dir, nil
	}
	mi.mu.RUnlock()

	dir, err := mi.getPkgDir(pkgPath)
	if err == nil && dir != "" {
		mi.mu.Lock()
		mi.dirCache[pkgPath] = dir
		mi.mu.Unlock()
	}
	return dir, err
}

// getPkgDir uses "go list" to get the package directory
func (mi *MethodIndex) getPkgDir(pkgPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-f", "{{.Dir}}", pkgPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		dir := strings.TrimSpace(string(out))
		if dir != "" {
			if _, err := os.Stat(dir); err == nil {
				return dir, nil
			}
		}
	}

	return mi.getPkgDirFromGOPATH(pkgPath)
}

// getPkgDirFromGOPATH resolves the package path from GOPATH
func (mi *MethodIndex) getPkgDirFromGOPATH(pkgPath string) (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot get user home directory")
		}
		gopath = filepath.Join(home, "go")
	}

	gopaths := filepath.SplitList(gopath)
	for _, gp := range gopaths {
		srcDir := filepath.Join(gp, "src")
		pkgDir := filepath.Join(srcDir, pkgPath)
		if _, err := os.Stat(pkgDir); err == nil {
			return pkgDir, nil
		}
	}

	return "", fmt.Errorf("package not found in GOPATH: %s", pkgPath)
}
