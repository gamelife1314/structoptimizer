package testnested

// OuterStruct 外层结构体，包含同包未导出嵌套结构体
type OuterStruct struct {
	A     bool
	data  innerData // 同包未导出结构体
	B     int64
	C     int32
}

// innerData 同包内未导出的结构体（这是需要被优化但可能未被识别的）
type innerData struct {
	X bool   // 1 字节
	Y int64  // 8 字节
	Z int32  // 4 字节
	W bool   // 1 字节
}
