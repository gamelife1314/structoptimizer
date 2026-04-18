package pkg

// localCache 未导出的本地缓存结构体（同包不同文件）
type localCache struct {
	Data      map[string]interface{}
	MaxSize   int64
	TTL       int64
	Enabled   bool
	HitCount  int32
}
