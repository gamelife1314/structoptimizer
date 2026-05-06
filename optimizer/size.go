package optimizer

import (
	"runtime"

	"go/types"
)

// platformPtrSize returns the pointer size for the current platform
var platformPtrSize = func() int64 {
	sizes := types.SizesFor("gc", runtime.GOARCH)
	if sizes != nil {
		return sizes.Sizeof(types.NewPointer(types.Typ[types.Int]))
	}
	return 8
}()

// platformPtrAlign returns the pointer alignment for the current platform
var platformPtrAlign = func() int64 {
	sizes := types.SizesFor("gc", runtime.GOARCH)
	if sizes != nil {
		return sizes.Alignof(types.NewPointer(types.Typ[types.Int]))
	}
	return 8
}()

// alignAndComputeSize aligns fields and computes the total struct size.
// Single canonical implementation used by all size calculation paths.
func alignAndComputeSize(fields []sizeableField) int64 {
	if len(fields) == 0 {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for _, f := range fields {
		align := f.getAlign()
		size := f.getSize()

		if offset%align != 0 {
			offset += align - (offset % align)
		}

		offset += size
		if align > maxAlign {
			maxAlign = align
		}
	}

	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// sizeableField is an interface for anything that can be laid out in a struct
type sizeableField interface {
	getSize() int64
	getAlign() int64
}

type fieldInfoAdapter struct {
	FieldInfo
}

func (f fieldInfoAdapter) getSize() int64  { return f.Size }
func (f fieldInfoAdapter) getAlign() int64 { return f.Align }

// CalcStructSizeFromFields calculates struct size from field information
func CalcStructSizeFromFields(fields []FieldInfo) int64 {
	adapters := make([]sizeableField, len(fields))
	for i := range fields {
		adapters[i] = fieldInfoAdapter{fields[i]}
	}
	return alignAndComputeSize(adapters)
}

// CalcStructSize calculates struct size (uses types.Sizes to simulate unsafe.Sizeof)
func CalcStructSize(st *types.Struct) int64 {
	if st == nil {
		return 0
	}

	sizes := types.SizesFor("gc", runtime.GOARCH)
	if sizes != nil {
		return sizes.Sizeof(st)
	}

	return calcStructSizeManual(st)
}

// calcStructSizeManual manually calculates struct size (fallback when types.Sizes unavailable)
func calcStructSizeManual(st *types.Struct) int64 {
	if st == nil {
		return 0
	}

	var fields []sizeableField
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		sz, al := CalcFieldSize(field.Type())
		fields = append(fields, &varFieldAdapter{sz: sz, algn: al})
	}
	return alignAndComputeSize(fields)
}

type varFieldAdapter struct {
	sz   int64
	algn int64
}

func (v *varFieldAdapter) getSize() int64  { return v.sz }
func (v *varFieldAdapter) getAlign() int64 { return v.algn }

// CalcFieldSize calculates field size using platform-aware sizes
func CalcFieldSize(typ types.Type) (size, align int64) {
	sizes := types.SizesFor("gc", runtime.GOARCH)
	if sizes != nil {
		return sizes.Sizeof(typ), sizes.Alignof(typ)
	}
	return calcFieldSizeFallback(typ)
}

// calcFieldSizeFallback manually calculates field size (fallback)
func calcFieldSizeFallback(typ types.Type) (size, align int64) {
	if typ == nil {
		return 0, 1
	}

	switch t := typ.(type) {
	case *types.Basic:
		return basicSize(t.Kind())
	case *types.Pointer:
		return sizeofPtr(), alignofPtr()
	case *types.Array:
		elemSize, elemAlign := calcFieldSizeFallback(t.Elem())
		if t.Len() == 0 {
			return 0, elemAlign
		}
		return elemSize * t.Len(), elemAlign
	case *types.Slice:
		return 24, alignofPtr()
	case *types.Map:
		return alignofPtr(), alignofPtr()
	case *types.Chan:
		return alignofPtr(), alignofPtr()
	case *types.Interface:
		return 16, alignofPtr()
	case *types.Named:
		return calcFieldSizeFallback(t.Underlying())
	case *types.Struct:
		return CalcStructSize(t), alignofPtr()
	default:
		return alignofPtr(), alignofPtr()
	}
}

// basicSize calculates the size of a basic type (platform-aware for int/uint)
func basicSize(kind types.BasicKind) (size, align int64) {
	sizes := types.SizesFor("gc", runtime.GOARCH)
	if sizes != nil {
		switch kind {
		case types.Bool:
			return sizes.Sizeof(types.Typ[types.Bool]), sizes.Alignof(types.Typ[types.Bool])
		case types.Int8, types.Uint8:
			return 1, 1
		case types.Int16, types.Uint16:
			return 2, 2
		case types.Int32, types.Uint32, types.Float32:
			return 4, 4
		case types.Int64, types.Uint64, types.Float64:
			return 8, 8
		case types.Int, types.Uint:
			return sizes.Sizeof(types.Typ[kind]), sizes.Alignof(types.Typ[kind])
		case types.Uintptr:
			return sizes.Sizeof(types.Typ[types.Uintptr]), sizes.Alignof(types.Typ[types.Uintptr])
		case types.String:
			return sizes.Sizeof(types.Typ[types.String]), sizes.Alignof(types.Typ[types.String])
		case types.UnsafePointer:
			return sizes.Sizeof(types.Typ[types.UnsafePointer]), sizes.Alignof(types.Typ[types.UnsafePointer])
		default:
			return sizes.Sizeof(types.Typ[kind]), sizes.Alignof(types.Typ[kind])
		}
	}

	// Fallback to hardcoded 64-bit values
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

// sizeofPtr returns the size of a pointer for the current platform
func sizeofPtr() int64 {
	return platformPtrSize
}

// alignofPtr returns the alignment of a pointer for the current platform
func alignofPtr() int64 {
	return platformPtrAlign
}

// CalcOptimizedSize calculates the size after optimization (uses FieldInfo sizes)
func CalcOptimizedSize(fields []FieldInfo) int64 {
	return CalcStructSizeFromFields(fields)
}
