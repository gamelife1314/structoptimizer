package models

// Level5_ParentOf6 第5层
type Level5_ParentOf6 struct {
	Name     string
	Child    Level6_ParentOf7
	API      Level6_APIConfig
	Cache    Level6_CacheConfig
	Count    int64
}

// Level5_ModuleConfig 第5层-模块配置
type Level5_ModuleConfig struct {
	ModuleName string
	Enabled    bool
	Config     Level6_APIConfig
	Timeout    int64
}

// Level5_DataSource 第5层-数据源
type Level5_DataSource struct {
	URL      string
	PoolSize int
	Timeout  int64
	Enabled  bool
	Retries  int32
}
