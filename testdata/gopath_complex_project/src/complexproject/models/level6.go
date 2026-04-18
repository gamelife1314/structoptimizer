package models

// Level6_ParentOf7 第6层
type Level6_ParentOf7 struct {
	Name     string
	Child    Level7_ParentOf8
	Data     Level7_DataParent
	Config   Level7_ServiceConfig
	Enabled  bool
}

// Level6_APIConfig 第6层-API配置
type Level6_APIConfig struct {
	BaseURL     string
	Timeout     int64
	Retries     int32
	RateLimit   int64
	Enabled     bool
}

// Level6_CacheConfig 第6层-缓存配置
type Level6_CacheConfig struct {
	MaxSize   int64
	TTL       int64
	Enabled   bool
	Eviction  string
}
