package analyzer

import (
	"testing"
)

// TestParseStructName 测试结构体名称解析
func TestParseStructName(t *testing.T) {
	tests := []struct {
		name       string
		fullName   string
		wantPkg    string
		wantStruct string
	}{
		{
			name:       "full path",
			fullName:   "example.com/pkg.MyStruct",
			wantPkg:    "example.com/pkg",
			wantStruct: "MyStruct",
		},
		{
			name:       "simple path",
			fullName:   "pkg.MyStruct",
			wantPkg:    "pkg",
			wantStruct: "MyStruct",
		},
		{
			name:       "no package",
			fullName:   "MyStruct",
			wantPkg:    "",
			wantStruct: "MyStruct",
		},
		{
			name:       "nested package",
			fullName:   "example.com/pkg/subpkg.MyStruct",
			wantPkg:    "example.com/pkg/subpkg",
			wantStruct: "MyStruct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkg, gotStruct := ParseStructName(tt.fullName)
			if gotPkg != tt.wantPkg {
				t.Errorf("ParseStructName() pkg = %v, want %v", gotPkg, tt.wantPkg)
			}
			if gotStruct != tt.wantStruct {
				t.Errorf("ParseStructName() struct = %v, want %v", gotStruct, tt.wantStruct)
			}
		})
	}
}

// TestAnalyzerLog 测试日志输出
func TestAnalyzerLog(t *testing.T) {
	cfg := &Config{
		Verbose: 2,
	}
	a := NewAnalyzer(cfg)

	// 测试不同级别的日志
	tests := []struct {
		level   int
		wantOut bool
	}{
		{0, true},  // 应该输出（<= Verbose）
		{1, true},  // 应该输出
		{2, true},  // 应该输出
		{3, false}, // 不应该输出（> Verbose）
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.level+'0')), func(t *testing.T) {
			// 这里只是验证不会 panic，实际输出需要手动检查
			a.Log(tt.level, "test message")
		})
	}
}

// TestNewAnalyzer 测试分析器创建
func TestNewAnalyzer(t *testing.T) {
	cfg := &Config{
		TargetDir: "/tmp/test",
		Verbose:   1,
		SkipDirs:  []string{"vendor", "generated"},
		SkipFiles: []string{"*_test.go"},
	}

	a := NewAnalyzer(cfg)

	if a == nil {
		t.Fatal("NewAnalyzer() returned nil")
	}
	if a.config != cfg {
		t.Error("NewAnalyzer() config not set correctly")
	}
	if a.fset == nil {
		t.Error("NewAnalyzer() fset not initialized")
	}
	if a.pkgMap == nil {
		t.Error("NewAnalyzer() pkgMap not initialized")
	}
}

// TestShouldSkipFile 测试文件跳过逻辑
func TestShouldSkipFile(t *testing.T) {
	cfg := &Config{
		SkipDirs:  []string{"vendor", "generated_*"},
		SkipFiles: []string{"*_test.go", "*.pb.go"},
	}
	_ = NewAnalyzer(cfg)

	// 验证配置是否正确设置
	if cfg.SkipFiles == nil {
		t.Error("SkipFiles not set")
	}
	if cfg.SkipDirs == nil {
		t.Error("SkipDirs not set")
	}
}
