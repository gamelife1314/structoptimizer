package pkg

// embeddedBase 未导出的基础匿名字段类型（同包不同文件）
type embeddedBase struct {
	ID        int64
	CreatedAt int64
	UpdatedAt int64
	Active    bool
	Version   int32
}
