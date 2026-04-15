package testdata

// WriteEmbedTestInner 内层结构体
type WriteEmbedTestInner struct {
	B int64
	A int8

	C int8
}

// WriteEmbedTestOuter 包含匿名字段且需要优化
type WriteEmbedTestOuter struct {
	Flag                bool   // 1 字节 - 应该在最后
	WriteEmbedTestInner        // 匿名字段 - 应该在前
	Name                string // 16 字节 - 应该在中间
}
