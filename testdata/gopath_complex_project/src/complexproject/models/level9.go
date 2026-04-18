package models

// Level9_ParentOf10 第9层-包含Level10
type Level9_ParentOf10 struct {
	Name    string
	Level10 Level10_DeepestStruct
	Count   int64
	Enabled bool
}

// Level9_SiblingStruct 第9层-兄弟结构体
type Level9_SiblingStruct struct {
	ID      int64
	Code    string
	Data    []byte
	Another Level10_AnotherStruct
	Active  bool
}

// Level9_ConfigParent 第9层-配置父级
type Level9_ConfigParent struct {
	Name    string
	Config  Level10_ConfigStruct
	Timeout int64
	Retries int32
}
