package optimizer

import (
	"path/filepath"
	"strings"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// shouldSkip checks if the struct should be skipped.
// Returns the skip reason and category.
func (o *Optimizer) shouldSkip(info *StructInfo, key string) (string, SkipCategory) {
	if len(info.Fields) == 0 {
		return "Empty struct", SkipEmpty
	}
	if len(info.Fields) == 1 {
		return "Single-field struct", SkipSingleField
	}
	if !o.config.AllowExternalPkgs && isVendorPackage(info.PkgPath) {
		return "Third-party package struct in vendor", SkipVendor
	}
	if !o.config.AllowExternalPkgs && !o.isProjectPackage(info.PkgPath) {
		if isStandardLibrary(info.PkgPath) {
			return "Go standard library struct", SkipStdLib
		}
		return "Non-project internal package struct", SkipNonProject
	}

	if len(o.config.SkipByNames) > 0 {
		for _, name := range o.config.SkipByNames {
			if o.matchStructName(key, name) {
				return "Skipped by name: " + name, SkipByName
			}
		}
	}

	if len(o.config.SkipByMethods) > 0 {
		for _, methodName := range o.config.SkipByMethods {
			if o.hasMethodByName(info, methodName) {
				return "Skipped by method: " + methodName, SkipByMethod
			}
		}
	}

	return "", SkipNone
}

// hasMethodByName checks if a struct has the specified method (uses MethodIndex, no package loading)
func (o *Optimizer) hasMethodByName(info *StructInfo, methodPattern string) bool {
	result := o.methodIndex.HasMethod(info.PkgPath, info.Name, methodPattern)
	o.Log(3, "Check method %s.%s.%s = %v", info.PkgPath, info.Name, methodPattern, result)
	return result
}

// matchStructName matches a struct name (supports wildcards)
func (o *Optimizer) matchStructName(key, pattern string) bool {
	if key == pattern {
		return true
	}
	_, structName := analyzer.ParseStructName(key)
	if structName == pattern {
		return true
	}
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
