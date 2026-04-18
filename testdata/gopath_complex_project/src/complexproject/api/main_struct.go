package api

import (
	"complexproject/models"
	"complexproject/config"
	"complexproject/types"
)

// MainComplexStruct 顶层复杂结构体 - 包含所有测试场景
type MainComplexStruct struct {
	// 基本信息
	Name        string
	Version     string
	Enabled     bool
	Timestamp   int64
	
	// 10层嵌套
	Level0      models.Level0_RootStruct
	
	// 同包未导出类型（通过其他方式引用）
	Config      *config.AppConfig
	
	// 重定义类型
	Status      types.ByteFlag
	Timeout     types.TimeoutMs
	MaxConns    types.Counter64
	AliasName   types.NameString
	
	// 匿名字段
	models.internalBase  // 同包未导出匿名字段
	
	// 复杂类型
	Data        types.ByteSlice
	Metadata    types.StringMap
	Tags        types.StringSlice
}
