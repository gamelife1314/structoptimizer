package pkg

import (
	"myproject/models"
	"myproject/utils"
	"github.com/external/lib"
)

// MainStruct 主结构体 - 包含多种场景
type MainStruct struct {
	Name       string
	Count      int64
	IsActive   bool
	Model      models.UnexportedModel  // 跨文件未导出结构体
	Helper     utils.HelperUtil        // 跨文件未导出结构体
	External   lib.ExternalStruct      // vendor中的结构体（应跳过）
	Data       []byte
	Config     *Config
	EmbeddedType                       // 匿名字段 - 测试1
	TypeAlias   CustomInt               // 类型别名 - 测试2
	MethodStruct HasMethods             // 用于测试skip-by-methods
	NoMethodStruct NoMethods            // 用于测试无方法结构体
}

// Config 配置结构体
type Config struct {
	Timeout   int64
	Retries   int32
	Enabled   bool
	Name      string
}

// EmbeddedType 匿名字段类型
type EmbeddedType struct {
	ID        int64
	Active    bool
	CreatedAt int64
}

// CustomInt 类型别名 - 测试类型大小识别
type CustomInt int

// HasMethods 用于测试skip-by-methods
type HasMethods struct {
	Name   string
	Value  int64
	Active bool
}

// Encode 测试方法检测
func (h *HasMethods) Encode() []byte {
	return nil
}

// Decode 测试方法检测
func (h *HasMethods) Decode(data []byte) error {
	return nil
}

// NoMethods 没有方法的结构体（应被优化）
type NoMethods struct {
	Flag    bool
	Data    int64
	Name    string
	Count   int32
}
