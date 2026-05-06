package optimizer

import (
	"go/ast"
	"go/parser"
	"testing"
)

// TestParseArrayLength 测试数组长度解析（Bug #1 修复）
func TestParseArrayLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "simple integer",
			input:    "10",
			expected: 10,
		},
		{
			name:     "hexadecimal",
			input:    "0x10",
			expected: 16,
		},
		{
			name:     "octal",
			input:    "010",
			expected: 8,
		},
		{
			name:     "parenthesized",
			input:    "(10)",
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 解析表达式
			expr, err := parser.ParseExpr("[" + tt.input + "]int")
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			arrayType, ok := expr.(*ast.ArrayType)
			if !ok {
				t.Fatal("Expected ArrayType")
			}

			result := parseArrayLength(arrayType.Len)
			if result != tt.expected {
				t.Errorf("parseArrayLength() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestEstimateFieldSizeArray 测试数组大小估算（Bug #1 修复）
func TestEstimateFieldSizeArray(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedSize  int64
		expectedAlign int64
	}{
		{
			name:          "slice",
			input:         "[]int64",
			expectedSize:  24,
			expectedAlign: 8,
		},
		{
			name:          "array of 10 int64",
			input:         "[10]int64",
			expectedSize:  80,
			expectedAlign: 8,
		},
		{
			name:          "array of 5 int32",
			input:         "[5]int32",
			expectedSize:  20,
			expectedAlign: 4,
		},
		{
			name:          "array of 3 bool",
			input:         "[3]bool",
			expectedSize:  3,
			expectedAlign: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			size, align := estimateFieldSize(expr)
			if size != tt.expectedSize {
				t.Errorf("estimateFieldSize() size = %v, want %v", size, tt.expectedSize)
			}
			if align != tt.expectedAlign {
				t.Errorf("estimateFieldSize() align = %v, want %v", align, tt.expectedAlign)
			}
		})
	}
}

// TestExtractTypeNameExtended 测试扩展的类型提取（Bug #6 修复）
func TestExtractTypeNameExtended(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  string
		expectedAlias string
	}{
		{
			name:          "array type",
			input:         "[10]int64",
			expectedType:  "[10]int64",
			expectedAlias: "",
		},
		{
			name:          "slice type",
			input:         "[]string",
			expectedType:  "[]string",
			expectedAlias: "",
		},
		{
			name:          "map type",
			input:         "map[string]int",
			expectedType:  "map[string]int",
			expectedAlias: "",
		},
		{
			name:          "chan type",
			input:         "chan int",
			expectedType:  "chan int",
			expectedAlias: "",
		},
		{
			name:          "func type",
			input:         "func()",
			expectedType:  "func",
			expectedAlias: "",
		},
		{
			name:          "interface type",
			input:         "interface{}",
			expectedType:  "interface{}",
			expectedAlias: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			// 创建临时的Optimizer用于测试
			opt := &Optimizer{}
			typeName, pkgAlias := opt.extractTypeNameFromExpr(expr)
			if typeName != tt.expectedType {
				t.Errorf("extractTypeNameFromExpr() typeName = %v, want %v", typeName, tt.expectedType)
			}
			if pkgAlias != tt.expectedAlias {
				t.Errorf("extractTypeNameFromExpr() pkgAlias = %v, want %v", pkgAlias, tt.expectedAlias)
			}
		})
	}
}

// TestAddReportEmbedDetection 测试匿名字段检测（Bug #2 修复）
func TestAddReportEmbedDetection(t *testing.T) {
	// 创建测试用的Optimizer
	opt := &Optimizer{
		optimized: make(map[string]*StructInfo),
		report: &Report{
			StructReports: make([]*StructReport, 0),
		},
	}

	// 创建包含匿名字段的StructInfo
	info := &StructInfo{
		Name:    "TestStruct",
		PkgPath: "test/pkg",
		File:    "test.go",
		Fields: []FieldInfo{
			{Name: "Field1", TypeName: "int64", Size: 8, Align: 8, IsEmbed: false},
			{Name: "", TypeName: "InnerStruct", Size: 16, Align: 8, IsEmbed: true}, // 匿名字段
			{Name: "Field2", TypeName: "bool", Size: 1, Align: 1, IsEmbed: false},
		},
		OrigSize:  32,
		OptSize:   24,
		OrigOrder: []string{"Field1", "InnerStruct", "Field2"},
		OptOrder:  []string{"Field1", "InnerStruct", "Field2"},
	}

	opt.addReport(info, "", 0, "")

	// 验证报告生成
	if len(opt.report.StructReports) != 1 {
		t.Fatalf("Expected 1 report, got %d", len(opt.report.StructReports))
	}

	report := opt.report.StructReports[0]
	if !report.HasEmbed {
		t.Error("Expected HasEmbed to be true for struct with embedded field")
	}
}

// TestCalcStructSizeWithArray 测试包含数组的结构体大小计算（Bug #1 综合测试）
func TestCalcStructSizeWithArray(t *testing.T) {
	// 创建包含数组字段的FieldInfo
	fields := []FieldInfo{
		{Name: "Data", TypeName: "[10]int64", Size: 80, Align: 8, IsEmbed: false},
		{Name: "Flag", TypeName: "bool", Size: 1, Align: 1, IsEmbed: false},
	}

	// 计算大小
	totalSize := CalcStructSizeFromFields(fields)

	// 80 + 1 + 7(padding) = 88
	if totalSize != 88 {
		t.Errorf("CalcStructSizeFromFields() = %v, want 88", totalSize)
	}
}
