package optimizer

import (
	"go/types"
)

// CalcStructSize 计算结构体大小
func CalcStructSize(st *types.Struct) int64 {
	if st == nil {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		sz, al := CalcFieldSize(field.Type(), nil)

		// 对齐
		if offset%al != 0 {
			offset += al - (offset % al)
		}

		offset += sz
		if al > maxAlign {
			maxAlign = al
		}
	}

	// 末尾填充
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// CalcFieldSize 计算字段大小
func CalcFieldSize(typ types.Type, info *types.Info) (size, align int64) {
	if typ == nil {
		return 0, 1
	}

	switch t := typ.(type) {
	case *types.Basic:
		return basicSize(t.Kind())

	case *types.Pointer:
		return sizeofPtr(), alignofPtr()

	case *types.Array:
		elemSize, elemAlign := CalcFieldSize(t.Elem(), info)
		if t.Len() == 0 {
			return 0, elemAlign
		}
		return elemSize * t.Len(), elemAlign

	case *types.Slice:
		return 24, 8

	case *types.Map:
		return 8, 8

	case *types.Chan:
		return 8, 8

	case *types.Interface:
		return 16, 8

	case *types.Named:
		return CalcFieldSize(t.Underlying(), info)

	case *types.Struct:
		return CalcStructSize(t), 8

	default:
		return 8, 8
	}
}

// basicSize 计算基本类型大小
func basicSize(kind types.BasicKind) (size, align int64) {
	switch kind {
	case types.Bool, types.Uint8, types.Int8:
		return 1, 1
	case types.Uint16, types.Int16:
		return 2, 2
	case types.Uint32, types.Int32, types.Float32:
		return 4, 4
	case types.Uint64, types.Int64, types.Float64:
		return 8, 8
	case types.Uint, types.Int:
		return 8, 8
	case types.Uintptr:
		return 8, 8
	case types.String:
		return 16, 8
	case types.UnsafePointer:
		return 8, 8
	default:
		return 8, 8
	}
}

// sizeofPtr 返回指针大小
func sizeofPtr() int64 {
	return 8
}

// alignofPtr 返回指针对齐
func alignofPtr() int64 {
	return 8
}

// CalcOptimizedSize 计算优化后的大小
func CalcOptimizedSize(fields []FieldInfo, info *types.Info) int64 {
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
