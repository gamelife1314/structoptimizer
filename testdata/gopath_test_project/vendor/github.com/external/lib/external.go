package lib

// ExternalStruct vendor中的第三方结构体 - 应被跳过
type ExternalStruct struct {
	ID     int64
	Name   string
	Active bool
	Data   int32
}

// ExternalConfig vendor中的配置结构体
type ExternalConfig struct {
	Host    string
	Port    int
	Enabled bool
}
