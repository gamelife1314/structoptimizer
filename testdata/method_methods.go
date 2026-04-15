package testdata

// === 指针接收者方法 ===

// PtrMethod 是指针接收者方法
func (p *PtrMethodStruct) PtrMethod() error {
	return nil
}

// PtrMethod2 是另一个指针接收者方法
func (p *PtrMethodStruct) PtrMethod2() ([]byte, error) {
	return nil, nil
}

// === 值接收者方法 ===

// ValueMethod 是值接收者方法
func (v ValueMethodStruct) ValueMethod() {
}

// ValueMethod2 是另一个值接收者方法
func (v ValueMethodStruct) ValueMethod2() int {
	return 0
}

// === 混合接收者方法 ===

// MethodStruct 的指针接收者方法（跨文件定义）
func (m *MethodStruct) Marshal() ([]byte, error) {
	return nil, nil
}

// MethodStruct 的值接收者方法
func (m MethodStruct) Unmarshal(data []byte) error {
	return nil
}
