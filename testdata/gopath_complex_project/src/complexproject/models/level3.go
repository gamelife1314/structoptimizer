package models

// Level3_ParentOf4 第3层
type Level3_ParentOf4 struct {
	Name     string
	Child    Level4_ParentOf5
	Plugin   Level4_PluginConfig
	Middleware Level4_MiddlewareConfig
	Count    int64
}

// Level3_RouterConfig 第3层-路由配置
type Level3_RouterConfig struct {
	Pattern  string
	Handler  string
	Enabled  bool
	Timeout  int64
	Middleware []string
}

// Level3_ValidatorConfig 第3层-验证器配置
type Level3_ValidatorConfig struct {
	Strict   bool
	Enabled  bool
	Rules    map[string]string
	Timeout  int64
}
