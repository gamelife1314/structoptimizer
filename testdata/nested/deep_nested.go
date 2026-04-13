package nested

// Level1 第一层嵌套
type Level1 struct {
	A bool
	B int64
	C int32
	D bool
}

// Level2 第二层嵌套
type Level2 struct {
	X int32
	L1 Level1
	Y int64
	Z bool
}

// Level3 第三层嵌套
type Level3 struct {
	M int64
	L2 Level2
	N int32
	O bool
}

// Level4 第四层嵌套
type Level4 struct {
	P bool
	L3 Level3
	Q int64
	R int32
}

// Level5 第五层嵌套
type Level5 struct {
	S int64
	L4 Level4
	T bool
	U int32
}

// Level6 第六层嵌套（超过 5 层）
type Level6 struct {
	V int32
	L5 Level5
	W int64
	X bool
}

// DeepNested7Level 7 层深度嵌套
type DeepNested7Level struct {
	Name   string
	Level6 Level6
	Count  int64
	Active bool
}
