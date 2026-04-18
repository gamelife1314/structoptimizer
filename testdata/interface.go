package testdata

// Interface 接口定义
type Interface interface {
	DoSomething()
	GetName() string
}

// MultiInterface 多方法接口
type MultiInterface interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

// StructWithInterface 包含接口字段的结构体
type StructWithInterface struct {
	Name   string
	Proc   Interface // 接口字段
	Count  int64
	Writer MultiInterface // 另一个接口字段
	Data   []byte
}

// StructWithEmptyInterface 包含空接口
type StructWithEmptyInterface struct {
	ID    int64
	Value interface{} // 空接口
	Name  string
}

// StructWithPointerInterface 包含接口指针（实际上接口本身就是引用类型）
type StructWithPointerInterface struct {
	First  int64
	Second Interface
	Third  bool
}
