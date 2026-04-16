package optimizer

import (
	"os"
	"path/filepath"
	"strings"
)

// isStandardLibraryPkg 快速判断是否是标准库包
func isStandardLibraryPkg(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	// 标准库不包含点号
	if strings.Contains(pkgPath, ".") {
		return false
	}
	// 以 go/ 开头的是标准库
	if strings.HasPrefix(pkgPath, "go/") {
		return true
	}
	// 检查是否是已知的标准库包（单级包名）
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
		"flag": true, "log": true, "testing": true, "path": true,
		"filepath": true, "strconv": true, "unicode": true, "utf8": true,
	}
	return standardLibs[pkgPath]
}

// isVendorPackage 判断是否是 vendor 中的包或第三方包
func isVendorPackage(pkgPath string) bool {
	// 1. 空包路径（通常是标准库或内置类型）
	if pkgPath == "" {
		return true
	}

	// 2. 检查是否包含 vendor 目录
	if strings.Contains(pkgPath, "/vendor/") || strings.HasPrefix(pkgPath, "vendor/") {
		return true
	}

	return false
}

// isStandardLibrary 判断是否是 Go 标准库
func isStandardLibrary(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	// 标准库不包含点号
	if strings.Contains(pkgPath, ".") {
		return false
	}
	// 单级包名，检查是否是已知标准库
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

// isProjectPackage 判断是否是项目内部的包
func (o *Optimizer) isProjectPackage(pkgPath string) bool {
	// 空包路径不是项目包
	if pkgPath == "" {
		return false
	}

	// vendor 中的不是项目包
	if isVendorPackage(pkgPath) {
		return false
	}

	// 标准库不是项目包
	if isStandardLibrary(pkgPath) {
		return false
	}

	// GOPATH 模式下，需要检查是否在项目路径下
	if o.config.ProjectType == "gopath" {
		// vendor 中的不是项目包
		if strings.Contains(pkgPath, "/vendor/") || strings.HasPrefix(pkgPath, "vendor/") {
			return false
		}
		
		// GOPATH 模式下，只要不是 vendor 和标准库，就认为是项目包
		return true
	}

	// Go Module 模式下，需要检查是否是当前项目的包
	if o.config.ProjectType == "gomod" || o.config.ProjectType == "" {
		// 获取项目根目录
		targetDir := o.config.TargetDir
		if targetDir == "" {
			targetDir = "."
		}

		// 尝试读取 go.mod 获取模块路径
		goModPath := filepath.Join(targetDir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			// 解析 go.mod 第一行获取模块路径
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					// 检查包路径是否以模块路径开头
					if strings.HasPrefix(pkgPath, modulePath) {
						// 确保是子路径，不是前缀匹配
						remaining := strings.TrimPrefix(pkgPath, modulePath)
						if remaining == "" || strings.HasPrefix(remaining, "/") {
							return true
						}
					}
					// 是其他模块的包，不是项目包
					return false
				}
			}
		}

		// 如果无法解析 go.mod，保守判断：只要不是 vendor 和标准库，就认为是项目包
		return true
	}

	// 默认认为是项目包
	return true
}

// fieldOrderSame 检查字段顺序是否相同
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

// getPackageDir 获取包所在的目录
func (o *Optimizer) getPackageDir(pkgPath string) string {
	// GOPATH 模式
	if o.config.ProjectType == "gopath" {
		gopath := o.config.GOPATH
		if gopath == "" {
			gopath = os.Getenv("GOPATH")
		}
		if gopath != "" {
			result := filepath.Join(gopath, "src", pkgPath)
			return result
		}
		return ""
	}

	// Go Module 模式
	if o.config.TargetDir != "" {
		relPath := strings.TrimPrefix(pkgPath, o.getModulePath())
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			return filepath.Join(o.config.TargetDir, relPath)
		}
		return o.config.TargetDir
	}

	return ""
}

// getModulePath 获取模块路径（从 go.mod）
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
