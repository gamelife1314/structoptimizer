package optimizer

import (
	"go/types"
	"testing"
	"unsafe"
)

// TestCalcFieldSize 测试字段大小计算
func TestCalcFieldSize(t *testing.T) {
	tests := []struct {
		name      string
		typ       types.Type
		wantSize  int64
		wantAlign int64
	}{
		{
			name:      "bool",
			typ:       types.Typ[types.Bool],
			wantSize:  1,
			wantAlign: 1,
		},
		{
			name:      "int8",
			typ:       types.Typ[types.Int8],
			wantSize:  1,
			wantAlign: 1,
		},
		{
			name:      "int16",
			typ:       types.Typ[types.Int16],
			wantSize:  2,
			wantAlign: 2,
		},
		{
			name:      "int32",
			typ:       types.Typ[types.Int32],
			wantSize:  4,
			wantAlign: 4,
		},
		{
			name:      "int64",
			typ:       types.Typ[types.Int64],
			wantSize:  8,
			wantAlign: 8,
		},
		{
			name:      "pointer",
			typ:       types.NewPointer(types.Typ[types.Int32]),
			wantSize:  int64(unsafe.Sizeof(uintptr(0))),
			wantAlign: int64(unsafe.Alignof(uintptr(0))),
		},
		{
			name:      "slice",
			typ:       types.NewSlice(types.Typ[types.Int32]),
			wantSize:  int64(unsafe.Sizeof([]int{})),
			wantAlign: int64(unsafe.Alignof([]int{})),
		},
		{
			name:      "map",
			typ:       types.NewMap(types.Typ[types.String], types.Typ[types.Int32]),
			wantSize:  int64(unsafe.Sizeof(map[string]int{})),
			wantAlign: int64(unsafe.Alignof(map[string]int{})),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, align := CalcFieldSize(tt.typ, nil)
			if size != tt.wantSize {
				t.Errorf("CalcFieldSize() size = %v, want %v", size, tt.wantSize)
			}
			if align != tt.wantAlign {
				t.Errorf("CalcFieldSize() align = %v, want %v", align, tt.wantAlign)
			}
		})
	}
}

// TestCalcStructSize 测试结构体大小计算
func TestCalcStructSize(t *testing.T) {
	// 创建测试结构体：BadStruct
	// type BadStruct struct {
	//     A bool   // 1 字节
	//     B int64  // 8 字节
	//     C int32  // 4 字节
	//     D bool   // 1 字节
	//     E int32  // 4 字节
	// }
	fields := []*types.Var{
		types.NewField(0, nil, "A", types.Typ[types.Bool], false),
		types.NewField(0, nil, "B", types.Typ[types.Int64], false),
		types.NewField(0, nil, "C", types.Typ[types.Int32], false),
		types.NewField(0, nil, "D", types.Typ[types.Bool], false),
		types.NewField(0, nil, "E", types.Typ[types.Int32], false),
	}
	badStruct := types.NewStruct(fields, nil)

	size := CalcStructSize(badStruct)
	// 计算：1+(7 填充) + 8 + 4 + 1+(3 填充) + 4 + (4 末尾填充) = 32 字节
	if size != 32 {
		t.Errorf("CalcStructSize(BadStruct) = %v, want 32", size)
	}

	// 创建优化后的结构体：GoodStruct
	// type GoodStruct struct {
	//     B int64  // 8 字节
	//     C int32  // 4 字节
	//     E int32  // 4 字节
	//     A bool   // 1 字节
	//     D bool   // 1 字节
	// }
	optFields := []*types.Var{
		types.NewField(0, nil, "B", types.Typ[types.Int64], false),
		types.NewField(0, nil, "C", types.Typ[types.Int32], false),
		types.NewField(0, nil, "E", types.Typ[types.Int32], false),
		types.NewField(0, nil, "A", types.Typ[types.Bool], false),
		types.NewField(0, nil, "D", types.Typ[types.Bool], false),
	}
	goodStruct := types.NewStruct(optFields, nil)

	size = CalcStructSize(goodStruct)
	// 计算：8 + 4 + 4 + 1 + 1 + (6 末尾填充) = 24 字节
	if size != 24 {
		t.Errorf("CalcStructSize(GoodStruct) = %v, want 24", size)
	}
}

// TestReorderFields 测试字段重排
func TestReorderFields(t *testing.T) {
	fields := []FieldInfo{
		{Name: "A", Size: 1, Align: 1, TypeName: "bool"},
		{Name: "B", Size: 8, Align: 8, TypeName: "int64"},
		{Name: "C", Size: 4, Align: 4, TypeName: "int32"},
		{Name: "D", Size: 1, Align: 1, TypeName: "bool"},
		{Name: "E", Size: 4, Align: 4, TypeName: "int32"},
	}

	result := ReorderFields(fields, false)

	// 期望顺序：B(8), C(4), E(4), A(1), D(1)
	expected := []string{"B", "C", "E", "A", "D"}

	if len(result) != len(expected) {
		t.Fatalf("ReorderFields() len = %v, want %v", len(result), len(expected))
	}

	for i, name := range expected {
		if result[i].Name != name {
			t.Errorf("ReorderFields()[%d] = %v, want %v", i, result[i].Name, name)
		}
	}
}

// TestSortFields 测试字段排序
func TestSortFields(t *testing.T) {
	fields := []FieldInfo{
		{Name: "A", Size: 1, Align: 1, TypeName: "bool"},
		{Name: "B", Size: 8, Align: 8, TypeName: "int64"},
		{Name: "C", Size: 4, Align: 4, TypeName: "int32"},
	}

	result := ReorderFields(fields, false)

	if result[0].Name != "B" {
		t.Errorf("ReorderFields()[0] = %v, want B", result[0].Name)
	}
	if result[1].Name != "C" {
		t.Errorf("ReorderFields()[1] = %v, want C", result[1].Name)
	}
	if result[2].Name != "A" {
		t.Errorf("ReorderFields()[2] = %v, want A", result[2].Name)
	}
}

// TestCalcOptimizedSize 测试优化后大小计算
func TestCalcOptimizedSize(t *testing.T) {
	fields := []FieldInfo{
		{Name: "B", Size: 8, Align: 8},
		{Name: "C", Size: 4, Align: 4},
		{Name: "E", Size: 4, Align: 4},
		{Name: "A", Size: 1, Align: 1},
		{Name: "D", Size: 1, Align: 1},
	}

	size := CalcOptimizedSize(fields, nil)
	// 8 + 4 + 4 + 1 + 1 + (6 填充) = 24 字节
	if size != 24 {
		t.Errorf("CalcOptimizedSize() = %v, want 24", size)
	}
}

// TestIsStructType 测试结构体类型判断
func TestIsStructType(t *testing.T) {
	// 创建具名结构体类型
	structType := types.NewStruct(nil, nil)
	namedType := types.NewTypeName(0, nil, "MyStruct", structType)
	wrappedNamed := types.NewNamed(namedType, structType, nil)

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{"struct", structType, true},
		{"named struct", wrappedNamed, true},
		{"pointer to struct", types.NewPointer(structType), true},
		{"slice of struct", types.NewSlice(structType), false}, // Slice 本身不是结构体
		{"int", types.Typ[types.Int32], false},
		{"string", types.Typ[types.String], false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStructType(tt.typ); got != tt.want {
				t.Errorf("isStructType(%v) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
