package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MatchPattern checks whether name matches pattern, supporting wildcards (* and ?)
func MatchPattern(pattern, name string) (bool, error) {
	return filepath.Match(pattern, name)
}

// MatchDirPattern checks whether a directory name matches the pattern
func MatchDirPattern(pattern, dirName string) bool {
	matched, err := filepath.Match(pattern, dirName)
	return err == nil && matched
}

// FormatSize formats a byte size for display
func FormatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d 字节", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
}

// GetGoModRoot finds the go.mod root directory
func GetGoModRoot(dir string) (string, error) {
	// Walk upward from the given directory until go.mod is found
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root, no go.mod found
			// Return the original directory (may be a GOPATH project)
			return dir, nil
		}
		dir = parent
	}
}

// IsGoModProject checks whether a directory is a go.mod project
func IsGoModProject(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// ShouldSkip checks whether a file or directory should be skipped
func ShouldSkip(path string, skipDirs, skipFiles, skipPatterns []string) bool {
	name := filepath.Base(path)

	// Check directory skip
	for _, pattern := range skipDirs {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Check file skip
	for _, pattern := range skipFiles {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	// Check generic skip patterns
	for _, pattern := range skipPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}

	return false
}

// SplitStructName splits a struct's fully qualified name (packagePath.structName)
func SplitStructName(fullName string) (pkgPath, structName string) {
	lastDot := strings.LastIndex(fullName, ".")
	if lastDot == -1 {
		return "", fullName
	}
	return fullName[:lastDot], fullName[lastDot+1:]
}

// Ptr returns a pointer to the given value
func Ptr[T any](v T) *T {
	return &v
}
