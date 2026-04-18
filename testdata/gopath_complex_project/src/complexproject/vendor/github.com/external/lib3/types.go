package lib3

// Externallib3Struct 第三方库结构体
type Externallib3Struct struct {
	ID       int64
	Name     string
	Data     []byte
	Enabled  bool
	Version  uint32
	Config   map[string]string
}

// Externallib3Config 第三方库配置
type Externallib3Config struct {
	Host    string
	Port    int
	Timeout int64
	Retry   int32
}

// Externallib3Client 第三方库客户端
type Externallib3Client struct {
	Connection interface{}
	Timeout    int64
	Enabled    bool
	Retries    int32
	MaxConns   int64
}
