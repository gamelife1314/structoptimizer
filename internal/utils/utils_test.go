package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMatchPattern 测试通配符匹配
func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		file    string
		want    bool
	}{
		{"exact match", "*.go", "test.go", true},
		{"prefix match", "test_*", "test_file.go", true},
		{"suffix match", "*.pb.go", "api.pb.go", true},
		{"no match", "*.go", "test.txt", false},
		{"single char", "test?.go", "test1.go", true},
		{"single char no match", "test?.go", "test12.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchPattern(tt.pattern, tt.file)
			if err != nil {
				t.Fatalf("MatchPattern() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("MatchPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMatchDirPattern 测试目录匹配
func TestMatchDirPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		dirName string
		want    bool
	}{
		{"vendor", "vendor", "vendor", true},
		{"generated prefix", "generated_*", "generated_proto", true},
		{"no match", "vendor", "src", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchDirPattern(tt.pattern, tt.dirName); got != tt.want {
				t.Errorf("MatchDirPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatSize 测试文件大小格式化
func TestFormatSize(t *testing.T) {
	tests := []struct {
		name string
		bytes int64
		want  string
	}{
		{"bytes", 100, "100 字节"},
		{"KB", 1024, "1.00 KB"},
		{"KB fraction", 1536, "1.50 KB"},
		{"MB", 1048576, "1.00 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSize(tt.bytes); got != tt.want {
				t.Errorf("FormatSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetGoModRoot 测试 go.mod 根目录查找
func TestGetGoModRoot(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()
	goModDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(goModDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	
	// 创建 go.mod 文件
	goModPath := filepath.Join(goModDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// 测试在子目录中查找
	subDir := filepath.Join(goModDir, "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	root, err := GetGoModRoot(subDir)
	if err != nil {
		t.Fatalf("GetGoModRoot() error = %v", err)
	}
	if root != goModDir {
		t.Errorf("GetGoModRoot() = %v, want %v", root, goModDir)
	}
}

// TestIsGoModProject 测试是否为 go.mod 项目
func TestIsGoModProject(t *testing.T) {
	tmpDir := t.TempDir()
	
	// 没有 go.mod
	if IsGoModProject(tmpDir) {
		t.Error("IsGoModProject() should return false without go.mod")
	}
	
	// 创建 go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
	
	if !IsGoModProject(tmpDir) {
		t.Error("IsGoModProject() should return true with go.mod")
	}
}

// TestShouldSkip 测试跳过逻辑
func TestShouldSkip(t *testing.T) {
	skipDirs := []string{"vendor", "generated_*"}
	skipFiles := []string{"*_test.go", "*.pb.go"}
	skipPatterns := []string{"*_mock.go"}

	tests := []struct {
		name     string
		path     string
		wantSkip bool
	}{
		{"normal file", "file.go", false},
		{"vendor dir", "vendor", true},
		{"generated dir", "generated_proto", true},
		{"test file", "file_test.go", true},
		{"pb file", "api.pb.go", true},
		{"mock file", "service_mock.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldSkip(tt.path, skipDirs, skipFiles, skipPatterns); got != tt.wantSkip {
				t.Errorf("ShouldSkip(%s) = %v, want %v", tt.path, got, tt.wantSkip)
			}
		})
	}
}

// TestSplitStructName 测试结构体名称分割
func TestSplitStructName(t *testing.T) {
	tests := []struct {
		name      string
		fullName  string
		wantPkg   string
		wantStruct string
	}{
		{"full path", "example.com/pkg.MyStruct", "example.com/pkg", "MyStruct"},
		{"simple", "pkg.MyStruct", "pkg", "MyStruct"},
		{"no package", "MyStruct", "", "MyStruct"},
		{"nested", "a.b.c/pkg/subpkg.Struct", "a.b.c/pkg/subpkg", "Struct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkg, gotStruct := SplitStructName(tt.fullName)
			if gotPkg != tt.wantPkg {
				t.Errorf("SplitStructName() pkg = %v, want %v", gotPkg, tt.wantPkg)
			}
			if gotStruct != tt.wantStruct {
				t.Errorf("SplitStructName() struct = %v, want %v", gotStruct, tt.wantStruct)
			}
		})
	}
}

// TestPtr 测试 Ptr 辅助函数
func TestPtr(t *testing.T) {
	val := 42
	ptr := Ptr(val)
	
	if ptr == nil {
		t.Fatal("Ptr() returned nil")
	}
	if *ptr != val {
		t.Errorf("Ptr() value = %v, want %v", *ptr, val)
	}
}
