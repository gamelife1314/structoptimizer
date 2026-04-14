package optimizer

import (
	"go/ast"
	"go/token"
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
		// 需要加载包来检查方法
		pkg, err := o.analyzer.LoadPackage(info.PkgPath)
		if err == nil {
			// 在包中查找结构体类型
			for _, syntax := range pkg.Syntax {
				for _, decl := range syntax.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok || genDecl.Tok != token.TYPE {
						continue
					}
					for _, spec := range genDecl.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok || typeSpec.Name.Name != info.Name {
							continue
						}
						if named, ok := pkg.TypesInfo.ObjectOf(typeSpec.Name).(*types.TypeName); ok {
							if t, ok := named.Type().(*types.Named); ok {
								for _, methodName := range o.config.SkipByMethods {
									if o.hasMethodOrMatch(t, methodName) {
										return "通过方法指定跳过：" + methodName
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// hasMethod 检查结构体是否有指定方法（精确匹配）
func (o *Optimizer) hasMethod(named *types.Named, methodName string) bool {
	for i := 0; i < named.NumMethods(); i++ {
		if named.Method(i).Name() == methodName {
			return true
		}
	}
	return false
}

// hasMethodOrMatch 检查结构体是否有指定方法或匹配通配符
func (o *Optimizer) hasMethodOrMatch(named *types.Named, pattern string) bool {
	// 精确匹配
	if o.hasMethod(named, pattern) {
		return true
	}

	// 通配符匹配
	if strings.Contains(pattern, "*") {
		for i := 0; i < named.NumMethods(); i++ {
			methodName := named.Method(i).Name()
			if matched, _ := filepath.Match(pattern, methodName); matched {
				return true
			}
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
