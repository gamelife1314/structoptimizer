package api

// HandlerWithEncode 具有Encode方法的处理器（应被skip-by-methods跳过）
type HandlerWithEncode struct {
	Name    string
	Timeout int64
	Enabled bool
	Retries int32
}

// Encode 编码方法
func (h *HandlerWithEncode) Encode() []byte {
	return []byte(h.Name)
}

// Decode 解码方法
func (h *HandlerWithEncode) Decode(data []byte) error {
	h.Name = string(data)
	return nil
}

// HandlerWithMarshal 具有Marshal方法的处理器
type HandlerWithMarshal struct {
	Name    string
	Path    string
	Timeout int64
	Enabled bool
}

// MarshalJSON JSON序列化
func (h *HandlerWithMarshal) MarshalJSON() ([]byte, error) {
	return nil, nil
}

// UnmarshalJSON JSON反序列化
func (h *HandlerWithMarshal) UnmarshalJSON(data []byte) error {
	return nil
}

// HandlerNoMethods 没有方法的处理器（应该被优化）
type HandlerNoMethods struct {
	Name     string
	Path     string
	Timeout  int64
	Enabled  bool
	Retries  int32
	Priority int32
}

// HandlerWithValidate 具有Validate方法的处理器
type HandlerWithValidate struct {
	Name    string
	Strict  bool
	Enabled bool
	Timeout int64
}

// Validate 验证方法
func (h *HandlerWithValidate) Validate() error {
	return nil
}
