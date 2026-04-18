package models

// internalConfig 未导出的内部配置（同包不同文件）
type internalConfig struct {
	Host        string
	Port        int
	Timeout     int64
	Enabled     bool
	Retries     int32
	MaxConns    int64
	Description string
	Debug       bool
	LogLevel    string
}

// internalCache 未导出的缓存配置（同包不同文件）
type internalCache struct {
	MaxSize   int64
	TTL       int64
	Enabled   bool
	Eviction  string
	HitCount  int64
	MissCount int64
}

// internalPool 未导出的连接池配置（同包不同文件）
type internalPool struct {
	MinConns  int32
	MaxConns  int64
	Timeout   int64
	Enabled   bool
	IdleTime  int64
}
