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
	if !strings.Contains(pkgPath, "/") || strings.HasPrefix(pkgPath, "go/") {
		return true
	}
	return false
}

// isVendorPackage checks if it is a vendor package or third-party package
func isVendorPackage(pkgPath string) bool {
	// 1. Empty package path (usually a standard library or built-in type)
	if pkgPath == "" {
		return true
	}

	// 2. Check if it contains a vendor directory
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
	// Standard libraries do not contain dots
	if strings.Contains(pkgPath, ".") {
		return false
	}
	// Single-segment package name, check against known standard libraries
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
	}
	return standardLibs[pkgPath]
}

// isProjectPackage checks if it is an internal project package
func (o *Optimizer) isProjectPackage(pkgPath string) bool {
	// Empty package path is not a project package
	if pkgPath == "" {
		return false
	}

	// Packages in vendor are not project packages
	if isVendorPackage(pkgPath) {
		return false
	}

	// Standard libraries are not project packages
	if isStandardLibrary(pkgPath) {
		return false
	}

	// In GOPATH mode, check if it is under the project path
	if o.config.ProjectType == "gopath" {
		gopath := o.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			// Check if the package path starts with GOPATH/src/
			if strings.HasPrefix(pkgPath, "src/") {
				// Extract the project path
				relPath := strings.TrimPrefix(pkgPath, "src/")
				// Check if it contains vendor
				if strings.Contains(relPath, "/vendor/") {
					return false
				}
				return true
			}
		}
		// In GOPATH mode, if not vendor and not standard library, consider it a project package
		return true
	}

	// In Go Module mode, check if it is a package of the current project
	if o.config.ProjectType == "gomod" || o.config.ProjectType == "" {
		// Get the project root directory
		targetDir := o.config.TargetDir
		if targetDir == "" {
			targetDir = "."
		}

		// Try to read go.mod to get the module path
		goModPath := filepath.Join(targetDir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			// Parse the first line of go.mod to get the module path
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					// Check if the package path starts with the module path
					if strings.HasPrefix(pkgPath, modulePath) {
						// Ensure it is a sub-path, not a prefix match
						remaining := strings.TrimPrefix(pkgPath, modulePath)
						if remaining == "" || strings.HasPrefix(remaining, "/") {
							return true
						}
					}
					// Package from another module, not a project package
					return false
				}
			}
		}

		// If go.mod cannot be parsed, conservative: consider it a project package if not vendor/stdlib
		return true
	}

	// Default to considering it a project package
	return true
}

// fieldOrderSame checks if field order is the same
func (o *Optimizer) fieldOrderSame(orig, opt []string) bool {
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
		// Go Module mode
		relPath := strings.TrimPrefix(pkgPath, o.getModulePath())
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			return filepath.Join(o.config.TargetDir, relPath)
		}
		return o.config.TargetDir
	}

	// GOPATH mode
	gopath := o.config.GOPATH
	if gopath == "" {
		gopath = os.Getenv("GOPATH")
	}
	if gopath != "" {
		return filepath.Join(gopath, "src", pkgPath)
	}

	return ""
}

// getModulePath returns the module path (from go.mod)
func (o *Optimizer) getModulePath() string {
	if o.config.TargetDir == "" {
		return ""
	}

	goModPath := filepath.Join(o.config.TargetDir, "go.mod")
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
