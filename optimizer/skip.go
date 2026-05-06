package optimizer

import (
	"path/filepath"
	"strings"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// shouldSkip checks if the struct should be skipped
func (o *Optimizer) shouldSkip(info *StructInfo, key string) string {
	// Empty struct
	if len(info.Fields) == 0 {
		return "Empty struct"
	}

	// Single-field struct
	if len(info.Fields) == 1 {
		return "Single-field struct"
	}

	// Check if it's a third-party struct in vendor (scan allowed when AllowExternalPkgs=true)
	if !o.config.AllowExternalPkgs && isVendorPackage(info.PkgPath) {
		return "Third-party package struct in vendor"
	}

	// Check if it's an internal project package (cross-package scan allowed when AllowExternalPkgs=true)
	if !o.config.AllowExternalPkgs && !o.isProjectPackage(info.PkgPath) {
		if isStandardLibrary(info.PkgPath) {
			return "Go standard library struct"
		}
		return "Non-project internal package struct"
	}

	// Check if it should be skipped by name
	if len(o.config.SkipByNames) > 0 {
		for _, name := range o.config.SkipByNames {
			if o.matchStructName(key, name) {
				return "Skipped by name: " + name
			}
		}
	}

	// Check if it should be skipped by method
	if len(o.config.SkipByMethods) > 0 {
		// Load the package to check methods
		for _, methodName := range o.config.SkipByMethods {
			if o.hasMethodByName(info, methodName) {
				return "Skipped by method: " + methodName
			}
		}
	}

	return ""
}

// hasMethodByName checks if a struct has the specified method (uses MethodIndex, no package loading)
func (o *Optimizer) hasMethodByName(info *StructInfo, methodPattern string) bool {
	// Query using MethodIndex, no need to load the entire package
	result := o.methodIndex.HasMethod(info.PkgPath, info.Name, methodPattern)
	o.Log(3, "Check method %s.%s.%s = %v", info.PkgPath, info.Name, methodPattern, result)
	return result
}

// matchMethod matches a method name (supports wildcards)
func (o *Optimizer) matchMethod(methodName, pattern string) bool {
	// Exact match
	if methodName == pattern {
		return true
	}

	// Wildcard match
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		if matched, _ := filepath.Match(pattern, methodName); matched {
			return true
		}
	}

	return false
}

// matchStructName matches a struct name (supports wildcards)
func (o *Optimizer) matchStructName(key, pattern string) bool {
	// Exact match
	if key == pattern {
		return true
	}

	// Simple name match (without package path)
	_, structName := analyzer.ParseStructName(key)
	if structName == pattern {
		return true
	}

	// Wildcard match
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, key)
		if matched {
			return true
		}
		matched, _ = filepath.Match(pattern, structName)
		return matched
	}

	return false
}
