package testdata

// WriteEmbedInner 内层结构体
type WriteEmbedInner struct {
	B int64
	A int8

	C int8
}

// WriteEmbedOuter 包含匿名字段且需要优化
type WriteEmbedOuter struct {
	// 1 字节 - 应该在最后
	// 匿名字段 - 应该在前
	Name string
	WriteEmbedInner
	// 16 字节 - 应该在中间
	// 1 字节
	B    int64 // 8 字节
	C    int32
	Flag bool

	A int8

	// 4 字节
}
