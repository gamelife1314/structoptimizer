package unexported

// Outer 外层结构体，包含未导出的嵌套结构体
type Outer struct {
	A     bool
	inner innerStruct // 未导出的嵌套结构体
	B     int64
	C     int32
}

// innerStruct 未导出的结构体（小写字母开头）
type innerStruct struct {
	Y int64
	X int32
	Z int32
	N bool
}

// AnotherOuter 另一个测试结构体，包含指针类型的未导出嵌套
type AnotherOuter struct {
	Name  string
	data  *innerStruct // 指针类型的未导出嵌套
	Count int64
}

// MixedOuter 混合测试：导出和未导出嵌套
type MixedOuter struct {
	A        bool
	inner    innerStruct    // 未导出
	Exported ExportedStruct // 导出
	B        int64
	unexport unexportStruct // 另一个未导出
}

// ExportedStruct 导出的结构体
type ExportedStruct struct {
	X int64
	Y int32
	Z int32
}

// unexportStruct 另一个未导出的结构体
type unexportStruct struct {
	M int64
	N int32
	O int32
	P bool
}

// BadUnexportOuter 测试用例：未优化顺序，包含未导出嵌套
type BadUnexportOuter struct {
	A     bool        // 1 字节
	data  innerStruct // 24 字节（未优化前会更大）
	B     int64       // 8 字节
	C     int32       // 4 字节
}

// BadInner 未导出的未优化结构体
type badInner struct {
	A bool   // 1 字节
	B int64  // 8 字节
	C int32  // 4 字节
	D bool   // 1 字节
	E int32  // 4 字节
}

// BadUnexportWithBadInner 包含未优化的未导出嵌套
type BadUnexportWithBadInner struct {
	X     bool     // 1 字节
	inner badInner // 32 字节
	Y     int64    // 8 字节
}
