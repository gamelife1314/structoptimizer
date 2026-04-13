package methodskip

// WithMethod 具有某些方法的结构体（应该被跳过）
type WithMethod struct {
	A bool
	B int64
	C int32
	D bool
}

// Encode 结构体的方法
func (w *WithMethod) Encode() []byte {
	return nil
}

// Decode 结构体的方法
func (w *WithMethod) Decode(data []byte) error {
	return nil
}

// MarshalJSON JSON 序列化方法
func (w *WithMethod) MarshalJSON() ([]byte, error) {
	return nil, nil
}

// WithoutMethod 没有特殊方法的结构体（应该被优化）
type WithoutMethod struct {
	A bool
	B int64
	C int32
	D bool
}

// AnotherWithMethod 也有方法的结构体（应该被跳过）
type AnotherWithMethod struct {
	X int32
	Y int64
	Z bool
}

// Validate 验证方法
func (a *AnotherWithMethod) Validate() error {
	return nil
}
