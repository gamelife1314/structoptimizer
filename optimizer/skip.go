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

	// 检查是否通过方法指定跳过（不常用，性能较差）
	// 注意：-skip-by-methods 需要加载包来检查方法，性能较差
	// 建议优先使用 -skip-by-names
	if len(o.config.SkipByMethods) > 0 && o.config.Verbose >= 3 {
		// 只在详细模式下检查方法，避免性能问题
		for _, methodName := range o.config.SkipByMethods {
			if o.hasMethodSimple(info, methodName) {
				return "通过方法指定跳过：" + methodName
			}
		}
	}

	return ""
}

// hasMethodSimple 简单检查方法名（不加载包，只检查字段类型）
func (o *Optimizer) hasMethodSimple(info *StructInfo, methodName string) bool {
	// 简化实现：只检查方法名是否匹配字段类型名
	// 完整检查需要加载包，性能较差
	for _, field := range info.Fields {
		if field.TypeName == methodName {
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
