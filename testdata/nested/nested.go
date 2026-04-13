package nested

// Inner 内部结构体
type Inner struct {
	Y int64
	X int32
	Z int32
}

// Inner2 另一个内部结构体
type Inner2 struct {
	A int64
	C int64
	B int32
}

// NestedOuter 嵌套结构体
type NestedOuter struct {
	Name   string
	Inner  Inner
	Count  int64
	Inner2 Inner2
}

// DeepNested 深度嵌套
type DeepNested struct {
	A     bool
	Outer NestedOuter
	B     int64
	C     int32
}
