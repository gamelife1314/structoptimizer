package testdata

// NestedChild 嵌套子结构体（有方法）
type NestedChild struct {
	Name string
	ID   int64
}

// Marshal 是 NestedChild 的指针接收者方法
func (n *NestedChild) Marshal() ([]byte, error) {
	return nil, nil
}

// NestedParent 0 层结构体（包含嵌套子结构体）
type NestedParent struct {
	Child NestedChild // 嵌套子结构体
	Value int64
	Flag  bool
}
