package models

// UserWithEmbedded 包含匿名字段的用户结构体
type UserWithEmbedded struct {
	internalBase    // 未导出的匿名字段
	internalAudit   // 未导出的审计匿名字段
	Name       string
	Email      string
	Age        int32
	Enabled    bool
}

// ProductWithEmbedded 包含匿名字段的产品结构体
type ProductWithEmbedded struct {
	internalBase    // 未导出的基础字段
	internalConfig  // 未导出的配置字段
	SKU        string
	Price      float64
	Stock      int64
	Enabled    bool
}

// OrderWithEmbedded 包含匿名字段的订单结构体
type OrderWithEmbedded struct {
	internalBase
	internalAudit
	OrderNumber string
	TotalAmount float64
	Status      string
	Paid        bool
}
