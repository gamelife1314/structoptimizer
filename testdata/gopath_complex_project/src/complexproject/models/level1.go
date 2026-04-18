package models

// Level1_ParentOf2 第1层
type Level1_ParentOf2 struct {
	Name     string
	Child    Level2_ParentOf3
	Server   Level2_ServerConfig
	Logger   Level2_LoggerConfig
	Enabled  bool
}
