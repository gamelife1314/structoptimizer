package complexpkg

import (
	"github.com/gamelife1314/structoptimizer/testdata/crosspkg/subpkg1"
)

// InnerForComplex 用于复杂结构体的内部结构体
type InnerForComplex struct {
	Y int64
	X int32
	Z int32
	A bool
}

// ComplexWithSlice 包含 slice 的复杂结构体
type ComplexWithSlice struct {
	A     bool
	B     []int64
	C     int32
	D     bool
	E     []byte
	F     int64
	G     []InnerForComplex
	H     int32
}

// ComplexWithMap 包含 map 的复杂结构体
type ComplexWithMap struct {
	A bool
	B map[string]int64
	C int32
	D bool
	E map[int32]InnerForComplex
	F int64
	G map[string][]byte
	H int32
}

// ComplexWithSliceAndMap 同时包含 slice 和 map
type ComplexWithSliceAndMap struct {
	Name     string
	IDs      []int64
	Data     map[string]int64
	Inner    InnerForComplex
	Count    int64
	Items    []InnerForComplex
	Config   map[string][]byte
	Active   bool
	Refs     []*subpkg1.SubPkg1
	Cache    map[int32]*subpkg1.SubPkg1
}

// NestedSlice 嵌套 slice
type NestedSlice struct {
	A       bool
	Matrix  [][]int64
	B       int64
	Items   []InnerForComplex
	C       int32
}

// NestedMap 嵌套 map
type NestedMap struct {
	X     bool
	Data  map[string]map[string]int64
	Y     int64
	Cache map[int32]map[string][]byte
	Z     int32
}

// ComplexPointer 包含指针的复杂结构体
type ComplexPointer struct {
	A      bool
	P1     *InnerForComplex
	B      int64
	P2     **InnerForComplex
	C      int32
	Slice  []*InnerForComplex
	D      bool
}

// VeryComplex 非常复杂的结构体
type VeryComplex struct {
	// 基本字段
	Name   string
	ID     int64
	Active bool
	
	// Slice 字段
	IDs      []int64
	Names    []string
	Inners   []InnerForComplex
	
	// Map 字段
	Data       map[string]int64
	Config     map[string][]byte
	ComplexMap map[int32]InnerForComplex
	
	// 跨包引用
	Refs    []*subpkg1.SubPkg1
	Cache   map[int32]*subpkg1.SubPkg1
	
	// 嵌套结构体
	Inner   InnerForComplex
	
	// 指针
	Ptr     *InnerForComplex
	
	// 计数
	Count   int64
	Size    int32
	Flags   bool
}
