package models

// Level4_ParentOf5 第4层
type Level4_ParentOf5 struct {
	Name       string
	Child      Level5_ParentOf6
	Module     Level5_ModuleConfig
	DataSource Level5_DataSource
	Active     bool
}

// Level4_PluginConfig 第4层-插件配置
type Level4_PluginConfig struct {
	PluginName string
	Version    string
	Enabled    bool
	Config     Level5_ModuleConfig
}

// Level4_MiddlewareConfig 第4层-中间件配置
type Level4_MiddlewareConfig struct {
	Name    string
	Order   int32
	Enabled bool
	Timeout int64
}
