package pkg

// MainStructWithUnexported 主结构体 - 引用同包不同文件的未导出类型
type MainStructWithUnexported struct {
	Name         string
	Count        int64
	IsActive     bool
	internal     *internalConfig // 同包未导出类型（小写开头）
	cache        localCache      // 同包未导出类型（小写开头）
	Data         []byte
	embeddedBase // 同包未导出匿名字段
}

// internalConfig 同包中的未导出配置结构体（在另一个文件中定义）
// 这个类型在 internal_config.go 中实际定义

// localCache 同包中的未导出缓存结构体（在另一个文件中定义）
// 这个类型在 local_cache.go 中实际定义

// embeddedBase 同包中的未导出匿名字段类型（在另一个文件中定义）
// 这个类型在 embedded_base.go 中实际定义
