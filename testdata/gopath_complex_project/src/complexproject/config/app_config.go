package config

// AppConfig 应用配置
type AppConfig struct {
	Name     string
	Version  string
	Enabled  bool
	Timeout  int64
	MaxConns int64
	Debug    bool
	LogLevel string
}
