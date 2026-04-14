package optimizer

import (
	"path/filepath"
	"testing"
)

// TestMatchMethod 测试方法名通配符匹配
func TestMatchMethod(t *testing.T) {
	o := &Optimizer{}
	
	tests := []struct {
		name     string
		method   string
		pattern  string
		wantMatch bool
	}{
		{"exact match", "Encode", "Encode", true},
		{"exact no match", "Encode", "Decode", false},
		{"prefix wildcard match", "Encode", "Encode*", true},
		{"prefix wildcard match2", "EncodeToJSON", "Encode*", true},
		{"prefix wildcard no match", "Decode", "Encode*", false},
		{"suffix wildcard match", "MarshalJSON", "*JSON", true},
		{"suffix wildcard match2", "UnmarshalJSON", "*JSON", true},
		{"suffix wildcard no match", "Encode", "*JSON", false},
		{"middle wildcard match", "EncodeToJSON", "*JSON", true},
		{"single char wildcard", "Encode", "Encod?", true},
		{"single char wildcard no match", "Encode", "Encod", false},
		{"question mark wildcard", "Validate", "Validat?", true},
		{"question mark wildcard no match", "Valid", "Validat?", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := o.matchMethod(tt.method, tt.pattern); got != tt.wantMatch {
				t.Errorf("matchMethod(%q, %q) = %v, want %v", tt.method, tt.pattern, got, tt.wantMatch)
			}
		})
	}
}

// TestMatchStructName 测试结构体名称通配符匹配
func TestMatchStructName(t *testing.T) {
	o := &Optimizer{}
	
	tests := []struct {
		name     string
		key      string
		pattern  string
		wantMatch bool
	}{
		// 完全匹配
		{"full match", "github.com/pkg.Context", "github.com/pkg.Context", true},
		{"full no match", "github.com/pkg.Context", "github.com/pkg.Config", false},
		
		// 简单名称匹配
		{"simple name match", "github.com/pkg.Context", "Context", true},
		{"simple name no match", "github.com/pkg.Context", "Config", false},
		
		// 通配符匹配 - 完整路径
		{"wildcard full path", "github.com/pkg.Context", "github.com/pkg.*", true},
		{"wildcard full path no match", "github.com/pkg.Context", "github.com/other.*", false},
		
		// 通配符匹配 - 结构体名
		{"wildcard struct name prefix", "github.com/pkg.UserRequest", "*Request", true},
		{"wildcard struct name prefix2", "github.com/pkg.Request", "*Request", true},
		{"wildcard struct name prefix no match", "github.com/pkg.Response", "*Request", false},
		
		{"wildcard struct name suffix", "github.com/pkg.UserRequest", "User*", true},
		{"wildcard struct name suffix2", "github.com/pkg.User", "User*", true},
		{"wildcard struct name suffix no match", "github.com/pkg.Config", "User*", false},
		
		// 问号通配符
		{"question mark wildcard", "github.com/pkg.Context", "Context?", false},
		{"question mark wildcard2", "github.com/pkg.Context", "Context", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := o.matchStructName(tt.key, tt.pattern); got != tt.wantMatch {
				t.Errorf("matchStructName(%q, %q) = %v, want %v", tt.key, tt.pattern, got, tt.wantMatch)
			}
		})
	}
}

// TestShouldSkipDir 测试目录跳过逻辑
func TestShouldSkipDir(t *testing.T) {
	o := &Optimizer{
		config: &Config{
			SkipDirs: []string{"vendor", "generated_*", "datas"},
		},
	}
	
	tests := []struct {
		name     string
		dirPath  string
		wantSkip bool
	}{
		// basename 匹配
		{"basename vendor", "/project/vendor", true},
		{"basename vendor nested", "/project/pkg/vendor", true},
		{"basename generated_proto", "/project/generated_proto", true},
		{"basename generated", "/project/generated", false},
		{"basename datas", "/project/datas", true},
		
		// 路径包含匹配
		{"path contains vendor", "/a/b/c/vendor/github.com/lib", true},
		{"path contains generated_proto", "/src/generated_proto/api.go", true},
		{"path contains generated", "/src/generated/api.go", false},
		{"path contains datas", "/do/datas/ele", true},
		
		// 不匹配的情况
		{"not match database", "/project/database", false},
		{"not match vendor_backup", "/project/vendor_backup", false},
		{"not match normal", "/project/pkg", false},
		{"not match test", "/project/test", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := o.shouldSkipDir(tt.dirPath); got != tt.wantSkip {
				t.Errorf("shouldSkipDir(%q) = %v, want %v", tt.dirPath, got, tt.wantSkip)
			}
		})
	}
}

// TestShouldSkipFile 测试文件跳过逻辑
func TestShouldSkipFile(t *testing.T) {
	o := &Optimizer{
		config: &Config{
			SkipFiles: []string{"*_test.go", "*.pb.go", "*_mock.go"},
		},
	}
	
	tests := []struct {
		name     string
		fileName string
		wantSkip bool
	}{
		{"test file", "api_test.go", true},
		{"pb file", "api.pb.go", true},
		{"mock file", "service_mock.go", true},
		{"normal file", "api.go", false},
		{"go file", "test.go", false},
		{"backup file", "api.go.bak", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := o.shouldSkipFile(tt.fileName); got != tt.wantSkip {
				t.Errorf("shouldSkipFile(%q) = %v, want %v", tt.fileName, got, tt.wantSkip)
			}
		})
	}
}

// TestFilepathMatch 测试 filepath.Match 的行为（作为参考）
func TestFilepathMatch(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		name2    string
		wantMatch bool
	}{
		// 星号通配符
		{"star match all", "*", "anything", true},
		{"star prefix", "test*", "test", true},
		{"star prefix2", "test*", "testing", true},
		{"star prefix3", "test*", "test123", true},
		{"star suffix", "*test", "test", true},
		{"star suffix2", "*test", "mytest", true},
		{"star middle", "test*go", "testing.go", true},
		
		// 问号通配符
		{"question match one", "test?", "test1", true},
		{"question match one2", "test?", "testa", true},
		{"question no match", "test?", "test", false},
		{"question no match2", "test?", "test12", false},
		
		// 字符组
		{"char class", "test[0-9]", "test1", true},
		{"char class no match", "test[0-9]", "testa", false},
		
		// 实际使用场景
		{"generated pattern", "generated_*", "generated_proto", true},
		{"generated pattern no match", "generated_*", "generated", false},
		{"generated pattern no match2", "generated_*", "generate", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := filepath.Match(tt.pattern, tt.name2)
			if err != nil {
				t.Fatalf("filepath.Match error: %v", err)
			}
			if matched != tt.wantMatch {
				t.Errorf("filepath.Match(%q, %q) = %v, want %v", tt.pattern, tt.name2, matched, tt.wantMatch)
			}
		})
	}
}
