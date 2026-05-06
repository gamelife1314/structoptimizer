package optimizer

import (
	"os"
	"path/filepath"
	"strings"
)

// isStandardLibraryPkg quickly checks if it is a standard library package
func isStandardLibraryPkg(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	if strings.Contains(pkgPath, ".") {
		return false
	}
	if strings.HasPrefix(pkgPath, "go/") {
		return true
	}
	if strings.Contains(pkgPath, "/") {
		return false
	}
	return isStandardLibrary(pkgPath)
}

// isVendorPackage checks if it is a vendor package or third-party package
func isVendorPackage(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	if strings.Contains(pkgPath, "/vendor/") || strings.HasPrefix(pkgPath, "vendor/") {
		return true
	}
	return false
}

// isStandardLibrary checks if it is a Go standard library
func isStandardLibrary(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	if strings.Contains(pkgPath, ".") {
		return false
	}
	standardLibs := map[string]bool{
		"fmt": true, "os": true, "io": true, "net": true, "http": true,
		"reflect": true, "errors": true, "bytes": true, "strings": true,
		"bufio": true, "sort": true, "sync": true, "time": true,
		"math": true, "rand": true, "regexp": true, "encoding": true,
		"json": true, "xml": true, "csv": true, "html": true, "url": true,
		"template": true, "text": true, "mime": true, "crypto": true,
		"hash": true, "compress": true, "archive": true, "image": true,
		"draw": true, "color": true, "jpeg": true, "png": true, "gif": true,
		"syscall": true, "runtime": true, "debug": true, "plugin": true,
		"unsafe": true, "atomic": true, "pprof": true, "trace": true,
		"flag": true, "log": true, "testing": true, "testing/iotest": true,
		"iotest": true, "quick": true, "exec": true, "signal": true,
		"path": true, "file": true, "filepath": true,
		// Go 1.21+ packages
		"cmp": true, "maps": true, "slices": true,
		// Go 1.23+ packages
		"iter": true, "unique": true, "structs": true,
		// Go 1.24+ packages
		"weak": true,
	}
	return standardLibs[pkgPath]
}

// isProjectPackage checks if it is an internal project package
func (o *Optimizer) isProjectPackage(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}
	if isVendorPackage(pkgPath) {
		return false
	}
	if isStandardLibrary(pkgPath) {
		return false
	}

	if o.config.ProjectType == "gopath" {
		gopath := o.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			if strings.HasPrefix(pkgPath, "src/") {
				relPath := strings.TrimPrefix(pkgPath, "src/")
				if strings.Contains(relPath, "/vendor/") {
					return false
				}
				return true
			}
		}
		return true
	}

	if o.config.ProjectType == "gomod" || o.config.ProjectType == "" {
		targetDir := o.config.TargetDir
		if targetDir == "" {
			targetDir = "."
		}

		modulePath := o.cachedModulePath(targetDir)
		if modulePath != "" {
			if strings.HasPrefix(pkgPath, modulePath) {
				remaining := strings.TrimPrefix(pkgPath, modulePath)
				if remaining == "" || strings.HasPrefix(remaining, "/") {
					return true
				}
			}
			return false
		}

		return true
	}

	return true
}

// cachedModulePath returns the module path from go.mod, reading only once and caching the result
func (o *Optimizer) cachedModulePath(targetDir string) string {
	if o.modulePathSet {
		return o.modulePath
	}
	o.modulePathSet = true
	o.modulePath = readModulePath(targetDir)
	return o.modulePath
}

// readModulePath reads the module path from go.mod
func readModulePath(targetDir string) string {
	if targetDir == "" {
		return ""
	}
	goModPath := filepath.Join(targetDir, "go.mod")
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

// fieldOrderSame checks if field order is the same (standalone function)
func fieldOrderSame(orig, opt []string) bool {
	if len(orig) != len(opt) {
		return false
	}
	for i := range orig {
		if orig[i] != opt[i] {
			return false
		}
	}
	return true
}

// getPackageDir returns the directory where the package is located
func (o *Optimizer) getPackageDir(pkgPath string) string {
	if o.config.TargetDir != "" {
		modulePath := o.cachedModulePath(o.config.TargetDir)
		relPath := strings.TrimPrefix(pkgPath, modulePath)
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			return filepath.Join(o.config.TargetDir, relPath)
		}
		return o.config.TargetDir
	}

	gopath := o.config.GOPATH
	if gopath == "" {
		gopath = os.Getenv("GOPATH")
	}
	if gopath != "" {
		return filepath.Join(gopath, "src", pkgPath)
	}

	return ""
}

// matchMethod matches a method name (supports wildcards, consolidated package-level function)
func matchMethod(methodName, pattern string) bool {
	if methodName == pattern {
		return true
	}
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		if matched, _ := filepath.Match(pattern, methodName); matched {
			return true
		}
	}
	return false
}
