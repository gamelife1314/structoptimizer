package models

// Level8_ParentOf9 第8层-包含Level9
type Level8_ParentOf9 struct {
	Title  string
	Child1 Level9_ParentOf10
	Child2 Level9_SiblingStruct
	Count  int64
	Active bool
	TTL    int64
}

// Level8_ConfigLevel 第8层-配置层
type Level8_ConfigLevel struct {
	Name    string
	Config  Level9_ConfigParent
	Enabled bool
	Timeout int64
}

// Level8_DataStruct 第8层-数据结构体
type Level8_DataStruct struct {
	ID       int64
	Data     []byte
	Metadata Level10_AnotherStruct
	Size     int64
	Version  uint32
}
