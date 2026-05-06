package optimizer

import (
	"sort"
)

// ReorderFields reorders fields to optimize memory alignment.
// Returns the sorted order; the caller decides whether to adopt it (only if it saves memory).
// Fields in reservedFields are always placed last.
func ReorderFields(fields []FieldInfo, sortSameSize bool, reservedFields []string) []FieldInfo {
	if len(fields) <= 1 {
		return fields
	}

	// Separate reserved fields and normal fields
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

	// Sort normal fields
	sorted := reorderFieldsInternal(normal, sortSameSize)

	// Append reserved fields to the end
	sorted = append(sorted, reserved...)

	return sorted
}

// reorderFieldsInternal implements the internal field sorting logic.
// Note: this function always returns the sorted result; the caller decides whether to adopt it.
func reorderFieldsInternal(fields []FieldInfo, sortSameSize bool) []FieldInfo {
	if len(fields) <= 1 {
		return fields
	}

	// Create a copy of the fields
	result := make([]FieldInfo, len(fields))
	copy(result, fields)

	// Sort: descending by size, then by alignment for equal sizes
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Size != result[j].Size {
			return result[i].Size > result[j].Size
		}
		if sortSameSize {
			if result[i].Align != result[j].Align {
				return result[i].Align > result[j].Align
			}
		}
		return result[i].Name < result[j].Name
	})

	// Always return the sorted result; the caller decides whether to adopt it
	return result
}

