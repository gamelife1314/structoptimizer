package testdata

// MethodStruct 结构体定义（在一个文件中）
type MethodStruct struct {
	Name string
	ID   int64
}

// PtrMethodStruct 用于测试指针接收者的结构体
type PtrMethodStruct struct {
	A int8
	B int64
	C int8
}

// ValueMethodStruct 用于测试值接收者的结构体
type ValueMethodStruct struct {
	X int64
	Y int32
}
