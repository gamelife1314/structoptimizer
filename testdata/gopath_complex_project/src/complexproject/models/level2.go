package models

// Level2_ParentOf3 第2层
type Level2_ParentOf3 struct {
	Name     string
	Child    Level3_ParentOf4
	Router   Level3_RouterConfig
	Validator Level3_ValidatorConfig
	Enabled  bool
}

// Level2_ServerConfig 第2层-服务器配置
type Level2_ServerConfig struct {
	Host       string
	Port       int
	Enabled    bool
	Timeout    int64
	MaxConns   int64
	TLS        bool
}

// Level2_LoggerConfig 第2层-日志配置
type Level2_LoggerConfig struct {
	Level      string
	Output     string
	Enabled    bool
	MaxSize    int64
	Compress   bool
}
