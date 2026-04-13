package basic

// BadStruct 未优化的结构体示例
type BadStruct struct {
	A bool   // 1 字节
	B int64  // 8 字节
	C int32  // 4 字节
	D bool   // 1 字节
	E int32  // 4 字节
}

// GoodStruct 已优化的结构体示例
type GoodStruct struct {
	B int64  // 8 字节
	C int32  // 4 字节
	E int32  // 4 字节
	A bool   // 1 字节
	D bool   // 1 字节
}

// SmallStruct 小结构体
type SmallStruct struct {
	A bool
	B int8
	C int16
}

// SingleField 单字段结构体（应该被跳过）
type SingleField struct {
	Value int64
}

// EmptyStruct 空结构体（应该被跳过）
type EmptyStruct struct {
}

// WithTag 带 tag 的结构体
type WithTag struct {
	A bool   `json:"a"`
	B int64  `json:"b"`
	C int32  `json:"c"`
}

// WithPointer 包含指针的结构体
type WithPointer struct {
	A bool
	B *int64
	C int32
	D bool
}

// WithSlice 包含切片的结构体
type WithSlice struct {
	A bool
	B []int64
	C int32
	D bool
}

// WithMap 包含 map 的结构体
type WithMap struct {
	A bool
	B map[string]int64
	C int32
	D bool
}
