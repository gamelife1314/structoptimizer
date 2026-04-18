package models

// Level0_RootStruct 第0层-顶层根结构体
type Level0_RootStruct struct {
	Name     string
	Child    Level1_ParentOf2
	Version  string
	Enabled  bool
	CreatedAt int64
}
