package models

// Level10_DeepestStruct 第10层-最深层
type Level10_DeepestStruct struct {
	ID        int64
	Name      string
	Value     float64
	Enabled   bool
	Timestamp int64
}

// Level10_AnotherStruct 第10层-另一个结构体
type Level10_AnotherStruct struct {
	Code    uint32
	Message string
	Data    []byte
	Count   int64
}

// Level10_ConfigStruct 第10层-配置结构体
type Level10_ConfigStruct struct {
	Host        string
	Port        int
	Timeout     int64
	MaxRetries  int32
	Enabled     bool
	Description string
}
