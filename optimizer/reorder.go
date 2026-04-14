package optimizer

import (
	"sort"
)

// ReorderFields 重排字段以优化内存对齐
func ReorderFields(fields []FieldInfo, sortSameSize bool) []FieldInfo {
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

	return result
}
