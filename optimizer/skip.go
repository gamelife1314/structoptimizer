package optimizer

import (
	"go/types"
	"path/filepath"
	"strings"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// shouldSkip 检查是否应该跳过
func (o *Optimizer) shouldSkip(info *StructInfo, st *types.Struct, key string) string {
	// 空结构体
	if len(info.Fields) == 0 {
		return "空结构体"
	}

	// 单字段结构体
	if len(info.Fields) == 1 {
		return "单字段结构体"
	}

	// 检查是否是 vendor 中的第三方包结构体
	if isVendorPackage(info.PkgPath) {
		return "vendor 中的第三方包结构体"
	}

	// 检查是否是项目内部的包
	if !o.isProjectPackage(info.PkgPath) {
		if isStandardLibrary(info.PkgPath) {
			return "Go 标准库结构体"
		}
		return "非项目内部包结构体"
	}

	// 检查是否通过名字指定跳过
	if len(o.config.SkipByNames) > 0 {
		for _, name := range o.config.SkipByNames {
			if o.matchStructName(key, name) {
				return "通过名字指定跳过：" + name
			}
		}
	}

	// 检查是否通过方法指定跳过
	if len(o.config.SkipByMethods) > 0 {
		// 加载包检查方法
		for _, methodName := range o.config.SkipByMethods {
			if o.hasMethodByName(info, methodName) {
				return "通过方法指定跳过：" + methodName
			}
		}
	}

	return ""
}

// hasMethodByName 检查结构体是否有指定方法（使用 MethodIndex，不加载包）
func (o *Optimizer) hasMethodByName(info *StructInfo, methodPattern string) bool {
	// 使用 MethodIndex 查询，无需加载整个包
	return o.methodIndex.HasMethod(info.PkgPath, info.Name, methodPattern)
}

// matchMethod 匹配方法名（支持通配符）
func (o *Optimizer) matchMethod(methodName, pattern string) bool {
	// 完全匹配
	if methodName == pattern {
		return true
	}

	// 通配符匹配
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		if matched, _ := filepath.Match(pattern, methodName); matched {
			return true
		}
	}

	return false
}

// matchStructName 匹配结构体名称（支持通配符）
func (o *Optimizer) matchStructName(key, pattern string) bool {
	// 完全匹配
	if key == pattern {
		return true
	}

	// 简单名称匹配（不包含包路径）
	_, structName := analyzer.ParseStructName(key)
	if structName == pattern {
		return true
	}

	// 通配符匹配
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
