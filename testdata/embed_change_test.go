package testdata

// EmbedA1 内层结构体
type EmbedA1 struct {
	A int8
	B int64
	C int8
}

// EmbedB1 包含匿名字段且需要优化
type EmbedB1 struct {
	Flag bool    // 1 字节 - 应该在最后
	EmbedA1       // 匿名字段 - 应该在前
	Name string  // 16 字节
	// 优化前：1+7+24+16 = 48 字节
	// 优化后：16+24+1+7 = 48 字节（大小不变，顺序变了）
}

// EmbedC1 包含多个匿名字段且需要优化
type EmbedC1 struct {
	Flag bool    // 1 字节
	EmbedA1       // 匿名字段 1
	Count int32  // 4 字节
}

