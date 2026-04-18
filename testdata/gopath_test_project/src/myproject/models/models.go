package models

// UnexportedModel 未导出的模型结构体 - 测试同包跨文件引用
type UnexportedModel struct {
	ID        int64
	Name      string
	IsActive  bool
	Data      int32
	CreatedAt int64
	UpdatedAt int64
}

// AnotherModel 另一个模型
type AnotherModel struct {
	Title  string
	Count  int64
	Active bool
}
