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
	// 注意：-skip-by-methods 功能已弃用
	// 原因：检查方法需要加载包，性能极差
	// 建议：使用 -skip-by-names 直接跳过结构体名
	// 例如：-skip-by-names='BadStruct,DeprecatedStruct'

	return ""
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
