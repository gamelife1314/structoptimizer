package utils

// HelperUtil 工具结构体 - 测试同包跨文件引用
type HelperUtil struct {
	Prefix  string
	Timeout int64
	Enabled bool
	Retries int32
}

// Cache 缓存工具
type Cache struct {
	Data    map[string]interface{}
	MaxSize int64
	Enabled bool
}
