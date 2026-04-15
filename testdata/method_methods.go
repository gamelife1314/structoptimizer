package testdata

// Marshal 是 MethodStruct 的方法（在另一个文件中）
func (m *MethodStruct) Marshal() ([]byte, error) {
	return nil, nil
}

// Unmarshal 是 MethodStruct 的方法
func (m *MethodStruct) Unmarshal(data []byte) error {
	return nil
}
