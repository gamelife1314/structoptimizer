package complexpkg

import (
	"github.com/gamelife1314/structoptimizer/testdata/crosspkg/subpkg1"
	"github.com/gamelife1314/structoptimizer/testdata/crosspkg/subpkg2"
)

// CrossPkgStruct 跨包引用结构体
type CrossPkgStruct struct {
	Name    string
	Pkg1    subpkg1.SubPkg1
	Count   int64
	Pkg2    subpkg2.SubPkg2
	Pkg1Ptr *subpkg1.SubPkg1
}
