package lib2

// Externallib2Struct 第三方库结构体
type Externallib2Struct struct {
	ID      int64
	Name    string
	Data    []byte
	Enabled bool
	Version uint32
	Config  map[string]string
}

// Externallib2Config 第三方库配置
type Externallib2Config struct {
	Host    string
	Port    int
	Timeout int64
	Retry   int32
}

// Externallib2Client 第三方库客户端
type Externallib2Client struct {
	Connection interface{}
	Timeout    int64
	Enabled    bool
	Retries    int32
	MaxConns   int64
}
