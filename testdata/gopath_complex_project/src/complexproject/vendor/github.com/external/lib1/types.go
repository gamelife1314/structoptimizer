package lib1

// Externallib1Struct 第三方库结构体
type Externallib1Struct struct {
	ID      int64
	Name    string
	Data    []byte
	Enabled bool
	Version uint32
	Config  map[string]string
}

// Externallib1Config 第三方库配置
type Externallib1Config struct {
	Host    string
	Port    int
	Timeout int64
	Retry   int32
}

// Externallib1Client 第三方库客户端
type Externallib1Client struct {
	Connection interface{}
	Timeout    int64
	Enabled    bool
	Retries    int32
	MaxConns   int64
}
