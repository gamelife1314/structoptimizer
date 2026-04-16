package optimizer

import (
	"go/types"
	"testing"
)

// TestCalcFieldSizeWithNamedType 测试重定义类型的大小计算
func TestCalcFieldSizeWithNamedType(t *testing.T) {
	tests := []struct {
		name         string
		basicKind    types.BasicKind
		wantSize     int64
		wantAlign    int64
	}{
		{
			name:      "uint8 重定义类型",
			basicKind: types.Uint8,
			wantSize:  1,
			wantAlign: 1,
		},
		{
			name:      "int8 重定义类型",
			basicKind: types.Int8,
			wantSize:  1,
			wantAlign: 1,
		},
		{
			name:      "bool 重定义类型",
			basicKind: types.Bool,
			wantSize:  1,
			wantAlign: 1,
		},
		{
			name:      "uint16 重定义类型",
			basicKind: types.Uint16,
			wantSize:  2,
			wantAlign: 2,
		},
		{
			name:      "int16 重定义类型",
			basicKind: types.Int16,
			wantSize:  2,
			wantAlign: 2,
		},
		{
			name:      "uint32 重定义类型",
			basicKind: types.Uint32,
			wantSize:  4,
			wantAlign: 4,
		},
		{
			name:      "int32 重定义类型",
			basicKind: types.Int32,
			wantSize:  4,
			wantAlign: 4,
		},
		{
			name:      "uint64 重定义类型",
			basicKind: types.Uint64,
			wantSize:  8,
			wantAlign: 8,
		},
		{
			name:      "int64 重定义类型",
			basicKind: types.Int64,
			wantSize:  8,
			wantAlign: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建基本类型
			basicType := types.Typ[tt.basicKind]
			
			// 创建重定义类型（Named type）
			namedType := types.NewNamed(types.NewTypeName(0, nil, "testType", basicType), basicType, nil)
			
			// 计算大小
			size, align := CalcFieldSize(namedType, nil)
			
			if size != tt.wantSize {
				t.Errorf("CalcFieldSize(%s) size = %d, want %d", tt.name, size, tt.wantSize)
			}
			if align != tt.wantAlign {
				t.Errorf("CalcFieldSize(%s) align = %d, want %d", tt.name, align, tt.wantAlign)
			}
		})
	}
}

// TestBasicSize 测试基本类型大小计算
func TestBasicSize(t *testing.T) {
	tests := []struct {
		name      string
		kind      types.BasicKind
		wantSize  int64
		wantAlign int64
	}{
		{"bool", types.Bool, 1, 1},
		{"int8", types.Int8, 1, 1},
		{"uint8", types.Uint8, 1, 1},
		{"int16", types.Int16, 2, 2},
		{"uint16", types.Uint16, 2, 2},
		{"int32", types.Int32, 4, 4},
		{"uint32", types.Uint32, 4, 4},
		{"int64", types.Int64, 8, 8},
		{"uint64", types.Uint64, 8, 8},
		{"int", types.Int, 8, 8},
		{"uint", types.Uint, 8, 8},
		{"float32", types.Float32, 4, 4},
		{"float64", types.Float64, 8, 8},
		{"string", types.String, 16, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, align := basicSize(tt.kind)
			if size != tt.wantSize {
				t.Errorf("basicSize(%s) size = %d, want %d", tt.name, size, tt.wantSize)
			}
			if align != tt.wantAlign {
				t.Errorf("basicSize(%s) align = %d, want %d", tt.name, align, tt.wantAlign)
			}
		})
	}
}
