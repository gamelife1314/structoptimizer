package services

// ServiceInterface 服务接口
type ServiceInterface interface {
	Start() error
	Stop() error
}

// BaseService 基础服务
type BaseService struct {
	Name    string
	Enabled bool
	Timeout int64
}
