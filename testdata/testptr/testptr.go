package testptr

// OuterWithPtr 外层结构体，包含指针类型的未导出嵌套
type OuterWithPtr struct {
	A    bool
	data *innerPtr // 指针类型的未导出结构体
	B    int64
}

// innerPtr 未导出的结构体
type innerPtr struct {
	X bool
	Y int64
	Z int32
}

// OuterWithSlice 外层结构体，包含 Slice 类型的未导出嵌套
type OuterWithSlice struct {
	A     bool
	items []innerItem // Slice 类型的未导出结构体
	B     int64
}

// innerItem 未导出的结构体
type innerItem struct {
	M int64
	N int32
	O bool
}
