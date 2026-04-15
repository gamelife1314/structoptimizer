package testdata

// TypeGroupTest 测试 type 分组定义
type (
	// GroupA 是分组中的第一个结构体
	// 这是多行注释
	// 第二行注释
	GroupA struct {
		Name string
		ID   int64
	}

	// GroupB 是分组中的第二个结构体
	GroupB struct {
		Value int32
		Flag  bool
	}

	// GroupC 是分组中的第三个结构体，需要优化
	GroupC struct {
		B int64
		A int8

		C int8
	}
)

// NormalDef 普通方式定义的结构体
type NormalDef struct {
	X int64
	Y int8
}
