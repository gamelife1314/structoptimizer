package models

// Level7_ParentOf8 第7层
type Level7_ParentOf8 struct {
	Name    string
	Child   Level8_ParentOf9
	Config  Level8_ConfigLevel
	Count   int64
	Enabled bool
}

// Level7_DataParent 第7层-数据父级
type Level7_DataParent struct {
	ID       int64
	Data     Level8_DataStruct
	Metadata map[string]string
	Active   bool
}

// Level7_ServiceConfig 第7层-服务配置
type Level7_ServiceConfig struct {
	ServiceName string
	Config      Level8_ConfigLevel
	Timeout     int64
	Retries     int32
	Enabled     bool
}
