package middleware

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	Name    string
	Order   int32
	Enabled bool
	Timeout int64
}
