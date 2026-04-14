package optimizer

import (
	"testing"
)

// TestIsVendorPackage 测试 vendor 包判断
func TestIsVendorPackage(t *testing.T) {
	tests := []struct {
		name   string
		pkgPath string
		want   bool
	}{
		{"empty", "", true},
		{"vendor with slash", "vendor/github.com/pkg", true},
		{"vendor in path", "github.com/pkg/vendor/lib", true},
		{"normal package", "github.com/pkg/lib", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVendorPackage(tt.pkgPath); got != tt.want {
				t.Errorf("isVendorPackage(%q) = %v, want %v", tt.pkgPath, got, tt.want)
			}
		})
	}
}

// TestIsStandardLibraryPkg 测试标准库判断
func TestIsStandardLibraryPkg(t *testing.T) {
	tests := []struct {
		name   string
		pkgPath string
		want   bool
	}{
		{"empty", "", true},
		{"fmt", "fmt", true},
		{"go/types", "go/types", true},
		{"net/http", "net/http", false}, // 多级包名不包含 go/ 前缀
		{"github.com/pkg", "github.com/pkg", false},
		{"example.com/pkg", "example.com/pkg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStandardLibraryPkg(tt.pkgPath); got != tt.want {
				t.Errorf("isStandardLibraryPkg(%q) = %v, want %v", tt.pkgPath, got, tt.want)
			}
		})
	}
}

// TestIsStandardLibrary 测试标准库判断（完整）
func TestIsStandardLibrary(t *testing.T) {
	tests := []struct {
		name   string
		pkgPath string
		want   bool
	}{
		{"empty", "", true},
		{"fmt", "fmt", true},
		{"errors", "errors", true},
		{"github.com/pkg", "github.com/pkg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStandardLibrary(tt.pkgPath); got != tt.want {
				t.Errorf("isStandardLibrary(%q) = %v, want %v", tt.pkgPath, got, tt.want)
			}
		})
	}
}

// TestFieldOrderSame 测试字段顺序比较
func TestFieldOrderSame(t *testing.T) {
	o := &Optimizer{}

	tests := []struct {
		name string
		orig []string
		opt  []string
		want bool
	}{
		{"empty", []string{}, []string{}, true},
		{"same", []string{"A", "B"}, []string{"A", "B"}, true},
		{"different", []string{"A", "B"}, []string{"B", "A"}, false},
		{"different length", []string{"A"}, []string{"A", "B"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := o.fieldOrderSame(tt.orig, tt.opt); got != tt.want {
				t.Errorf("fieldOrderSame(%v, %v) = %v, want %v", tt.orig, tt.opt, got, tt.want)
			}
		})
	}
}
