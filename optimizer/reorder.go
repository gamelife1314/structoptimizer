package optimizer

import (
	"sort"
)

// ReorderFields 重排字段以优化内存对齐
// 只有当能节省内存时才调整顺序，否则保持原顺序
// reservedFields 中的字段始终排在最后
func ReorderFields(fields []FieldInfo, sortSameSize bool, reservedFields []string) []FieldInfo {
	if len(fields) <= 1 {
		return fields
	}

	// 分离预留字段和普通字段
	var reserved []FieldInfo
	var normal []FieldInfo
	
	reservedMap := make(map[string]bool)
	for _, name := range reservedFields {
		reservedMap[name] = true
	}
	
	for _, f := range fields {
		if reservedMap[f.Name] {
			reserved = append(reserved, f)
		} else {
			normal = append(normal, f)
		}
	}

	// 对普通字段排序
	sorted := reorderFieldsInternal(normal, sortSameSize)
	
	// 预留字段追加到末尾
	sorted = append(sorted, reserved...)
	
	return sorted
}

// reorderFieldsInternal 内部字段排序逻辑
func reorderFieldsInternal(fields []FieldInfo, sortSameSize bool) []FieldInfo {
	if len(fields) <= 1 {
		return fields
	}

	// 创建字段副本
	result := make([]FieldInfo, len(fields))
	copy(result, fields)

	// 排序：按大小降序，相同大小按对齐降序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Size != result[j].Size {
			return result[i].Size > result[j].Size
		}
		if sortSameSize {
			return result[i].Align > result[j].Align
		}
		return false
	})

	// 计算原始大小
	origSize := calcSizeFromFields(fields)
	// 计算优化后大小
	optSize := calcSizeFromFields(result)

	// 只有能节省内存时才调整顺序
	if optSize < origSize {
		return result
	}

	// 否则保持原顺序
	return fields
}

// calcSizeFromFields 计算字段总大小（含填充）
func calcSizeFromFields(fields []FieldInfo) int64 {
	if len(fields) == 0 {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for _, field := range fields {
		// 对齐
		if offset%field.Align != 0 {
			offset += field.Align - (offset % field.Align)
		}

		offset += field.Size
		if field.Align > maxAlign {
			maxAlign = field.Align
		}
	}

	// 末尾填充
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}
