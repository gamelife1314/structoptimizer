package complexpkg

import (
	"github.com/gamelife1314/structoptimizer/testdata/crosspkg/subpkg1"
	"github.com/gamelife1314/structoptimizer/testdata/crosspkg/subpkg2"
)

// CrossPkgRef1 跨包引用结构体 1
type CrossPkgRef1 struct {
	A int64
	B subpkg1.SubPkg1
	C int32
	D bool
}

// CrossPkgRef2 跨包引用结构体 2
type CrossPkgRef2 struct {
	X int64
	Y subpkg2.SubPkg2
	Z int32
	W bool
}

// ComplexWithCrossPkg 包含跨包引用的复杂结构体
type ComplexWithCrossPkg struct {
	Name   string
	Pkg1   CrossPkgRef1
	Pkg2   CrossPkgRef2
	Count  int64
	Active bool
}
