package testdata

// WriteSkipEmbedInner 内层结构体
type WriteSkipEmbedInner struct {
	B int64
	A int8

	C int8
}

// WriteSkipEmbedOuter 包含匿名字段且需要优化
type WriteSkipEmbedOuter struct {
	Flag                bool   // 1 字节
	WriteSkipEmbedInner        // 匿名字段
	Name                string // 16 字节
}
