package grpc

// GRPCConnection gRPC连接
type GRPCConnection struct {
	Target   string
	Timeout  int64
	Enabled  bool
	Retries  int32
}

// GRPCConfig gRPC配置
type GRPCConfig struct {
	Host          string
	Port          int
	MaxMsgSize    int64
	KeepAliveTime int64
	Enabled       bool
}
