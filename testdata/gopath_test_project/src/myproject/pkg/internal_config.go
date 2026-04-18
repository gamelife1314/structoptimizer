package pkg

// internalConfig 未导出的内部配置结构体（同包不同文件）
type internalConfig struct {
	Host        string
	Port        int
	Timeout     int64
	Enabled     bool
	Retries     int32
	MaxConns    int64
	Description string
}
