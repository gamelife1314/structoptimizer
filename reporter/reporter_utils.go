package reporter

// getFieldSize 根据类型名称估算字段大小
func getFieldSize(typeName string) int64 {
	switch typeName {
	case "bool", "byte", "int8", "uint8":
		return 1
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "rune", "float32":
		return 4
	case "int64", "uint64", "float64", "int", "uint", "uintptr":
		return 8
	case "string":
		return 16
	default:
		if typeName == "" {
			return 8
		}
		if typeName[0] == '[' && len(typeName) > 1 && typeName[1] == ']' {
			return 24 // slice 头
		}
		if len(typeName) >= 4 && typeName[:4] == "map[" {
			return 8 // map 头
		}
		if typeName[0] == '*' {
			return 8 // 指针
		}
		return 8 // 默认
	}
}

// classifyReports 将结构体报告分类
func classifyReports(reports []interface{}) (optimized, skipped, unchanged []interface{}) {
	for _, sr := range reports {
		// 使用反射或类型断言来分类
		// 这里简化处理，实际使用时需要根据具体类型调整
		optimized = append(optimized, sr)
	}
	return
}
