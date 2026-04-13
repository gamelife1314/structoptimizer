package optimizer

import (
	"go/types"
	"unsafe"
)

// CalcStructSize 计算结构体的实际大小（包含填充）
func CalcStructSize(st *types.Struct, info *types.Info) int64 {
	if st.NumFields() == 0 {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		size, align := CalcFieldSize(field.Type(), info)

		// 对齐偏移
		if offset%align != 0 {
			offset += align - (offset % align)
		}

		offset += size
		if align > maxAlign {
			maxAlign = align
		}
	}

	// 末尾填充，使结构体大小是最大对齐要求的倍数
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// CalcOptimizedSize 计算优化后的结构体大小（字段按大小降序排列）
func CalcOptimizedSize(fields []FieldInfo, info *types.Info) int64 {
	if len(fields) == 0 {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for _, field := range fields {
		size := field.Size
		align := field.Align

		// 对齐偏移
		if offset%align != 0 {
			offset += align - (offset % align)
		}

		offset += size
		if align > maxAlign {
			maxAlign = align
		}
	}

	// 末尾填充
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// CalcFieldSize 计算字段的大小和对齐要求
func CalcFieldSize(typ types.Type, info *types.Info) (size, align int64) {
	switch t := typ.(type) {
	case *types.Basic:
		return basicSize(t.Kind())

	case *types.Pointer:
		sz := int64(unsafe.Sizeof(uintptr(0)))
		al := int64(unsafe.Alignof(uintptr(0)))
		return sz, al

	case *types.Struct:
		return CalcStructSize(t, info), structAlign(t, info)

	case *types.Array:
		elemSize, elemAlign := CalcFieldSize(t.Elem(), info)
		if t.Len() == 0 {
			return 0, elemAlign
		}
		// 数组大小 = 元素大小 * 元素个数
		// 数组对齐 = 元素对齐
		return elemSize * t.Len(), elemAlign

	case *types.Slice:
		sz := int64(unsafe.Sizeof([]int{}))
		al := int64(unsafe.Alignof([]int{}))
		return sz, al

	case *types.Map:
		sz := int64(unsafe.Sizeof(map[int]int{}))
		al := int64(unsafe.Alignof(map[int]int{}))
		return sz, al

	case *types.Chan:
		sz := int64(unsafe.Sizeof(make(chan int)))
		al := int64(unsafe.Alignof(make(chan int)))
		return sz, al

	case *types.Interface:
		sz := int64(unsafe.Sizeof((*interface{})(nil)))
		al := int64(unsafe.Alignof((*interface{})(nil)))
		return sz, al

	case *types.Named:
		// 具名类型，递归计算其底层类型
		return CalcFieldSize(t.Underlying(), info)

	case *types.Tuple:
		// 函数返回类型等，按结构体处理
		return calcTupleSize(t, info)

	default:
		// 其他类型，返回默认值
		return 8, 8
	}
}

// basicSize 返回基本类型的大小
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
		return int64(unsafe.Sizeof(int(0))), int64(unsafe.Alignof(int(0)))
	case types.Uintptr:
		return int64(unsafe.Sizeof(uintptr(0))), int64(unsafe.Alignof(uintptr(0)))
	case types.String:
		return int64(unsafe.Sizeof("")), int64(unsafe.Alignof(""))
	case types.UnsafePointer:
		return int64(unsafe.Sizeof(unsafe.Pointer(nil))), int64(unsafe.Alignof(unsafe.Pointer(nil)))
	default:
		return 8, 8
	}
}

// structAlign 计算结构体的对齐要求
func structAlign(st *types.Struct, info *types.Info) int64 {
	var maxAlign int64 = 1
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		_, align := CalcFieldSize(field.Type(), info)
		if align > maxAlign {
			maxAlign = align
		}
	}
	return maxAlign
}

// calcTupleSize 计算 tuple 的大小
func calcTupleSize(t *types.Tuple, info *types.Info) (size, align int64) {
	if t == nil || t.Len() == 0 {
		return 0, 1
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for i := 0; i < t.Len(); i++ {
		variable := t.At(i)
		sz, al := CalcFieldSize(variable.Type(), info)

		if offset%al != 0 {
			offset += al - (offset % al)
		}

		offset += sz
		if al > maxAlign {
			maxAlign = al
		}
	}

	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset, maxAlign
}
