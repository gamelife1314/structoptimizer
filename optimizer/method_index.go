package optimizer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// MethodIndex caches method sets for packages.
// Structure: map[package_path]map[struct_name]map[method_name]bool
type MethodIndex struct {
	mu    sync.RWMutex
	cache map[string]map[string]map[string]bool
}

// NewMethodIndex creates a new method indexer
func NewMethodIndex() *MethodIndex {
	return &MethodIndex{
		cache: make(map[string]map[string]map[string]bool),
	}
}

// HasMethod checks if a struct has the specified method (supports wildcards)
func (mi *MethodIndex) HasMethod(pkgPath, structName, methodPattern string) bool {
	mi.mu.RLock()
	pkgCache, pkgExists := mi.cache[pkgPath]
	mi.mu.RUnlock()

	if !pkgExists {
		if err := mi.indexPkg(pkgPath); err != nil {
			// Indexing failed, conservatively return false
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

	// Check method match
	for methodName := range structMethods {
		if mi.matchMethod(methodName, methodPattern) {
			return true
		}
	}
	return false
}

// matchMethod matches a method name (supports wildcards)
func (mi *MethodIndex) matchMethod(methodName, pattern string) bool {
	// Exact match
	if methodName == pattern {
		return true
	}

	// Wildcard match
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		matched, _ := filepath.Match(pattern, methodName)
		if matched {
			return true
		}
	}

	return false
}

// indexPkg scans the package directory to build the index
func (mi *MethodIndex) indexPkg(pkgPath string) error {
	// Get the package directory
	dir, err := mi.getPkgDir(pkgPath)
	if err != nil {
		return err
	}

	if dir == "" {
		return fmt.Errorf("cannot get package directory: %s", pkgPath)
	}

	// Scan files
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}

	// Ensure package cache exists
	mi.mu.Lock()
	if _, ok := mi.cache[pkgPath]; !ok {
		mi.cache[pkgPath] = make(map[string]map[string]bool)
	}
	pkgCache := mi.cache[pkgPath]
	mi.mu.Unlock()

	fset := token.NewFileSet()
	for _, file := range files {
		// Skip test files
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

			// Extract receiver type
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

// getPkgDir uses "go list" to get the package directory
func (mi *MethodIndex) getPkgDir(pkgPath string) (string, error) {
	// Attempt 1: use "go list" (Go Modules mode)
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", pkgPath)
	// Do not modify the environment; use the current project's module mode
	out, err := cmd.CombinedOutput()
	if err == nil {
		dir := strings.TrimSpace(string(out))
		if dir != "" {
			// Verify the directory exists
			if _, err := os.Stat(dir); err == nil {
				return dir, nil
			}
		}
	}

	// Attempt 2: manually resolve via GOPATH
	return mi.getPkgDirFromGOPATH(pkgPath)
}

// getPkgDirFromGOPATH resolves the package path from GOPATH
func (mi *MethodIndex) getPkgDirFromGOPATH(pkgPath string) (string, error) {
	// Get GOPATH
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Default GOPATH
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot get user home directory")
		}
		gopath = filepath.Join(home, "go")
	}

	// GOPATH may have multiple paths, split by separator
	gopaths := filepath.SplitList(gopath)

	// Search under each GOPATH's src directory
	for _, gp := range gopaths {
		srcDir := filepath.Join(gp, "src")
		pkgDir := filepath.Join(srcDir, pkgPath)

		// Verify the directory exists
		if _, err := os.Stat(pkgDir); err == nil {
			return pkgDir, nil
		}
	}

	return "", fmt.Errorf("package not found in GOPATH: %s", pkgPath)
}
