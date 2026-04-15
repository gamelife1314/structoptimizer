package testdata

// MultiEmbedInner1 第一个内层结构体
type MultiEmbedInner1 struct {
	B int64
	A int8
}

// MultiEmbedInner2 第二个内层结构体
type MultiEmbedInner2 struct {
	D int64
	C int32
}

// MultiEmbedOuter 包含多个匿名字段
type MultiEmbedOuter struct {
	// 匿名字段 1
	// 匿名字段 2

	Name string
	MultiEmbedInner1
	MultiEmbedInner2
	Flag bool
}
