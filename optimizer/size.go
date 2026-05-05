package optimizer

import (
	"go/types"
)

// CalcStructSizeFromFields calculates struct size from field information
func CalcStructSizeFromFields(fields []FieldInfo) int64 {
	if len(fields) == 0 {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for _, field := range fields {
		// Alignment
		if offset%field.Align != 0 {
			offset += field.Align - (offset % field.Align)
		}

		offset += field.Size
		if field.Align > maxAlign {
			maxAlign = field.Align
		}
	}

	// Trailing padding
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// CalcStructSize calculates struct size (uses types.Sizes to simulate unsafe.Sizeof)
func CalcStructSize(st *types.Struct, sizes types.Sizes) int64 {
	if st == nil {
		return 0
	}

	// If types.Sizes is available, use it for calculation (more accurate)
	if sizes != nil {
		return sizes.Sizeof(st)
	}

	// Otherwise fall back to manual calculation
	return calcStructSizeManual(st)
}

// calcStructSizeManual manually calculates struct size (fallback)
func calcStructSizeManual(st *types.Struct) int64 {
	if st == nil {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		sz, al := CalcFieldSize(field.Type(), nil)

		// Alignment
		if offset%al != 0 {
			offset += al - (offset % al)
		}

		offset += sz
		if al > maxAlign {
			maxAlign = al
		}
	}

	// Trailing padding
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// CalcFieldSize calculates field size
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
		return CalcStructSize(t, nil), 8

	default:
		return 8, 8
	}
}

// basicSize calculates the size of a basic type
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

// sizeofPtr returns the size of a pointer
func sizeofPtr() int64 {
	return 8
}

// alignofPtr returns the alignment of a pointer
func alignofPtr() int64 {
	return 8
}

// CalcOptimizedSize calculates the size after optimization
func CalcOptimizedSize(fields []FieldInfo, info *types.Info) int64 {
	if len(fields) == 0 {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for _, field := range fields {
		// Alignment
		if offset%field.Align != 0 {
			offset += field.Align - (offset % field.Align)
		}

		offset += field.Size
		if field.Align > maxAlign {
			maxAlign = field.Align
		}
	}

	// Trailing padding
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}
