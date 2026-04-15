package testdata

// WriteEmbedInner2 内层结构体
type WriteEmbedInner2 struct {
	B int64
	A int8

	C int8
}

// WriteEmbedOuter2 包含匿名字段且需要优化（非最优顺序）
type WriteEmbedOuter2 struct {
	// 1 字节
	// 1 字节
	// 匿名字段
	Name string
	WriteEmbedInner2
	A    int8
	Flag bool

	// 16 字节
}
