package models

// internalBase 未导出的基础匿名字段
type internalBase struct {
	ID        int64
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
	Version   int32
	Active    bool
}

// internalAudit 未导出的审计匿名字段
type internalAudit struct {
	CreatedBy  string
	UpdatedBy  string
	CreatedAt  int64
	UpdatedAt  int64
	IPAddress  string
	UserAgent  string
}
