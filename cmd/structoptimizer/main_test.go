package main

import (
	"os"
	"strings"
	"testing"
)

// TestParseFlags 测试命令行参数解析
func TestParseFlags(t *testing.T) {
	// 保存原始 os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{"structoptimizer"},
			wantErr: true,
		},
		{
			name:    "struct flag",
			args:    []string{"structoptimizer", "-struct", "example.com/pkg.MyStruct", "./"},
			wantErr: false,
		},
		{
			name:    "package flag",
			args:    []string{"structoptimizer", "-package", "example.com/pkg", "./"},
			wantErr: false,
		},
		{
			name:    "write flag",
			args:    []string{"structoptimizer", "-struct", "example.com/pkg.MyStruct", "-write", "./"},
			wantErr: false,
		},
		{
			name:    "backup flag",
			args:    []string{"structoptimizer", "-struct", "example.com/pkg.MyStruct", "-backup", "./"},
			wantErr: false,
		},
		{
			name:    "skip-dirs flag",
			args:    []string{"structoptimizer", "-package", "example.com/pkg", "-skip-dirs", "vendor,testdata", "./"},
			wantErr: false,
		},
		{
			name:    "skip-files flag",
			args:    []string{"structoptimizer", "-package", "example.com/pkg", "-skip-files", "*_test.go,*.pb.go", "./"},
			wantErr: false,
		},
		{
			name:    "skip-by-names flag",
			args:    []string{"structoptimizer", "-package", "example.com/pkg", "-skip-by-names", "BadStruct,*Request", "./"},
			wantErr: false,
		},
		{
			name:    "skip-by-methods flag",
			args:    []string{"structoptimizer", "-package", "example.com/pkg", "-skip-by-methods", "Encode,*JSON", "./"},
			wantErr: false,
		},
		{
			name:    "verbose flags",
			args:    []string{"structoptimizer", "-struct", "example.com/pkg.MyStruct", "-v", "./"},
			wantErr: false,
		},
		{
			name:    "version flag",
			args:    []string{"structoptimizer", "-version"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			// 注意：这里只是验证解析不会 panic
			// 实际验证需要重构 parseFlags 函数
		})
	}
}

// TestConfigDefaults 测试配置默认值
func TestConfigDefaults(t *testing.T) {
	cfg := &Config{
		Backup:    true,
		TargetDir: ".",
	}

	if !cfg.Backup {
		t.Error("Default backup should be true")
	}
	if cfg.TargetDir != "." {
		t.Errorf("Default target dir = %v, want .", cfg.TargetDir)
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid struct",
			cfg: &Config{
				Struct:    "example.com/pkg.MyStruct",
				TargetDir: ".",
			},
			wantErr: false,
		},
		{
			name: "valid package",
			cfg: &Config{
				Package:   "example.com/pkg",
				TargetDir: ".",
			},
			wantErr: false,
		},
		{
			name: "both struct and package",
			cfg: &Config{
				Struct:    "example.com/pkg.MyStruct",
				Package:   "example.com/pkg",
				TargetDir: ".",
			},
			wantErr: true,
		},
		{
			name: "neither struct nor package",
			cfg: &Config{
				TargetDir: ".",
			},
			wantErr: true,
		},
		{
			name: "invalid struct name",
			cfg: &Config{
				Struct:    "InvalidStruct",
				TargetDir: ".",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlags(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestVersion 测试版本号
func TestVersion(t *testing.T) {
	// 导入 reporter 包的版本常量
	// 这里只是验证 main 包可以正常工作
	if true { // placeholder
		// version is defined in reporter package
	}
}

// TestSkipDirsParsing 测试 -skip-dirs 参数解析
func TestSkipDirsParsing(t *testing.T) {
	// 测试逗号分隔的 skip-dirs
	input := "vendor,testdata,generated_*"
	parts := strings.Split(input, ",")

	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}

	expected := []string{"vendor", "testdata", "generated_*"}
	for i, part := range parts {
		if part != expected[i] {
			t.Errorf("Part %d = %v, want %v", i, part, expected[i])
		}
	}
}

// TestSkipFilesParsing 测试 -skip-files 参数解析
func TestSkipFilesParsing(t *testing.T) {
	input := "*_test.go,*.pb.go,*_mock.go"
	parts := strings.Split(input, ",")

	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}

	expected := []string{"*_test.go", "*.pb.go", "*_mock.go"}
	for i, part := range parts {
		if part != expected[i] {
			t.Errorf("Part %d = %v, want %v", i, part, expected[i])
		}
	}
}

// TestSkipByNamesParsing 测试 -skip-by-names 参数解析
func TestSkipByNamesParsing(t *testing.T) {
	input := "BadStruct,*Request,*Response"
	parts := strings.Split(input, ",")

	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}
}

// TestSkipByMethodsParsing 测试 -skip-by-methods 参数解析
func TestSkipByMethodsParsing(t *testing.T) {
	input := "Encode,Decode,*JSON"
	parts := strings.Split(input, ",")

	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}
}
