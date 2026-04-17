package optimizer

import (
	"go/types"
	"testing"
)

// TestNamedTypeSizeCalculation 测试重定义类型大小计算
func TestNamedTypeSizeCalculation(t *testing.T) {
	tests := []struct {
		name      string
		typ       types.Type
		wantSize  int64
		wantAlign int64
	}{
		{
			name:      "uint8 基本类型",
			typ:       types.Typ[types.Uint8],
			wantSize:  1,
			wantAlign: 1,
		},
		{
			name:      "int64 基本类型",
			typ:       types.Typ[types.Int64],
			wantSize:  8,
			wantAlign: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建重定义类型
			named := types.NewNamed(types.NewTypeName(0, nil, "NewType", tt.typ), tt.typ, nil)

			size, align := CalcFieldSize(named, nil)

			if size != tt.wantSize {
				t.Errorf("CalcFieldSize(%s) size = %d, want %d", tt.name, size, tt.wantSize)
			}
			if align != tt.wantAlign {
				t.Errorf("CalcFieldSize(%s) align = %d, want %d", tt.name, align, tt.wantAlign)
			}
		})
	}
}
