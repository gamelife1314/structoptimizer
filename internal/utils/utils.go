package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MatchPattern 检查 name 是否匹配 pattern 支持通配符 (* 和 ?)
func MatchPattern(pattern, name string) (bool, error) {
	return filepath.Match(pattern, name)
}

// MatchDirPattern 检查目录是否匹配模式
func MatchDirPattern(pattern, dirName string) bool {
	matched, err := filepath.Match(pattern, dirName)
	return err == nil && matched
}

// FormatSize 格式化字节大小
func FormatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d 字节", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
}

// GetGoModRoot 获取 go.mod 根目录
func GetGoModRoot(dir string) (string, error) {
	// 从给定目录向上查找直到找到 go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// 已经到达根目录，没找到 go.mod
			// 返回原始目录（可能是 GOPATH 项目）
			return dir, nil
		}
		dir = parent
	}
}

// IsGoModProject 判断是否是 go.mod 项目
func IsGoModProject(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// ShouldSkip 检查是否应该跳过某个文件或目录
func ShouldSkip(path string, skipDirs, skipFiles, skipPatterns []string) bool {
	name := filepath.Base(path)
	
	// 检查目录跳过
	for _, pattern := range skipDirs {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	
	// 检查文件跳过
	for _, pattern := range skipFiles {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	
	// 检查通用跳过模式
	for _, pattern := range skipPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	
	return false
}

// SplitStructName 分割结构体名称（包路径。结构体名）
func SplitStructName(fullName string) (pkgPath, structName string) {
	lastDot := strings.LastIndex(fullName, ".")
	if lastDot == -1 {
		return "", fullName
	}
	return fullName[:lastDot], fullName[lastDot+1:]
}

// Ptr 返回值的指针
func Ptr[T any](v T) *T {
	return &v
}
