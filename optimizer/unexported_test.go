package optimizer

import (
	"testing"
)

// TestUnexportedStructHandling 测试未导出结构体类型的处理
func TestUnexportedStructHandling(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		wantSize int64
		wantAlign int64
	}{
		{
			name:     "导出结构体类型",
			typeName: "ExportedStruct",
			wantSize: 8, // 默认估算值
			wantAlign: 8,
		},
		{
			name:     "未导出结构体类型",
			typeName: "innerStruct",
			wantSize: 8, // 默认估算值
			wantAlign: 8,
		},
		{
			name:     "未导出小写类型",
			typeName: "badInner",
			wantSize: 8, // 默认估算值
			wantAlign: 8,
		},
		{
			name:     "基本类型 bool",
			typeName: "bool",
			wantSize: 1,
			wantAlign: 1,
		},
		{
			name:     "基本类型 int64",
			typeName: "int64",
			wantSize: 8,
			wantAlign: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, align := sizeOfIdent(tt.typeName)
			if size != tt.wantSize {
				t.Errorf("sizeOfIdent(%q) size = %v, want %v", tt.typeName, size, tt.wantSize)
			}
			if align != tt.wantAlign {
				t.Errorf("sizeOfIdent(%q) align = %v, want %v", tt.typeName, align, tt.wantAlign)
			}
		})
	}
}

// TestIsUnexportedStructName 测试未导出类型名称识别
func TestIsUnexportedStructName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"innerStruct", true},    // 未导出
		{"badInner", true},       // 未导出
		{"data", true},           // 未导出
		{"ExportedStruct", false},  // 导出
		{"PublicType", false},    // 导出
		{"bool", false},          // 基本类型
		{"int64", false},         // 基本类型
		{"byte", false},          // 基本类型
		{"string", false},        // 基本类型
		{"", false},              // 空字符串
		{"MyType", false},        // 导出类型
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnexportedStructName(tt.name)
			if got != tt.want {
				t.Errorf("isUnexportedStructName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestEstimateFieldSizeWithUnexported 测试包含未导出类型的字段大小估算
func TestEstimateFieldSizeWithUnexported(t *testing.T) {
	tests := []struct {
		name      string
		typeName  string
		wantSize  int64
		wantAlign int64
	}{
		{
			name:      "未导出结构体字段",
			typeName:  "innerStruct",
			wantSize:  8,
			wantAlign: 8,
		},
		{
			name:      "未导出结构体指针字段",
			typeName:  "*innerStruct",
			wantSize:  8,
			wantAlign: 8,
		},
		{
			name:      "基本类型 bool",
			typeName:  "bool",
			wantSize:  1,
			wantAlign: 1,
		},
		{
			name:      "基本类型 int64",
			typeName:  "int64",
			wantSize:  8,
			wantAlign: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里无法直接测试 estimateFieldSize，因为它需要 AST 表达式
			// 但我们可以测试 sizeOfIdent
			size, align := sizeOfIdent(tt.typeName)
			if size != tt.wantSize {
				t.Errorf("sizeOfIdent(%q) size = %v, want %v", tt.typeName, size, tt.wantSize)
			}
			if align != tt.wantAlign {
				t.Errorf("sizeOfIdent(%q) align = %v, want %v", tt.typeName, align, tt.wantAlign)
			}
		})
	}
}
