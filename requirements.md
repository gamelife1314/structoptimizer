### 需求描述

本软件主要是想实现一个用于检测go项目中结构体字段是否对齐的工具，某些大型项目中，在前提开发的时候未充分识别到结构体字段对齐的意义，开发人员随心所以，导致浪费了很多内存，在当下内存价格很高的情况下，这些软件上的内存优化变得尤为珍贵。参考：https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go，
但是这个工具太过简单，没法对嵌套结构体进行优化，没法对结构体中字段跨包引用的结构体进行优化，所以开发一个新的工具用以支持这些功能。

### 实际使用场景

进入到项目目录，执行下面的命令进，相对当前目录找到 `writer/config` 包中的结构体 `Context`，进行字段优化：
> ./structoptimizer -struct=writer/config.Context --write ./

### 需要支持的功能如下

#### 核心功能

对项目中Go语言定义的结构体进行字段对齐优化，特别要支持结构体中跨包引用的字段。例如，下面这个结构体存在字段未对齐的问题：

```go
type BadStruct struct {
    A bool   // 1 字节
    // [填充 7 字节] 以对齐 B (int64 需要 8 字节对齐)
    B int64  // 8 字节
    C int32  // 4 字节
    D bool   // 1 字节
    // [填充 3 字节] 以对齐 E (int32 需要 4 字节对齐)
    E int32  // 4 字节
    // [末尾填充 4 字节] 以使结构体总大小能被 8 整除
}
// 内存计算：1+(7) + 8 + 4 + 1+(3) + 4 + (4) = 32 字节
```

对它进行对齐优化，优化之后为：
```go
type GoodStruct struct {
    B int64  // 8 字节 (偏移量 0)
    C int32  // 4 字节 (偏移量 8)
    E int32  // 4 字节 (偏移量 12)
    A bool   // 1 字节 (偏移量 16)
    D bool   // 1 字节 (偏移量 17)
    // [末尾填充 6 字节] 以使结构体总大小能被 8 整除
}
// 内存计算：8 + 4 + 4 + 1 + 1 + (6) = 24 字节？
// 不对，看这里：8 + 4 + 4 = 16 字节，此时 A 和 D 占用 2 字节。
// 16 + 2 = 18 字节，向上补齐到 8 的倍数，最终为 24 字节。
```

但是核心功能要支持结构体命名字段、匿名字段，跨包引用的场景，相同的结构体只需优化1次。这里的举例只有两层嵌套，实际过程中可能嵌套多层，NestedOuter嵌套SubPkg1，SubPkg1嵌套SubPkg2等等。工具需要按照深度优先的原则对每个结构体进行优化，除非被排除或者已经优化过：

```go
// 主结构体所在包：project/testdata.NestedOuter
type NestedOuter struct {
	Name   string
	Inner  Inner
	Count  int64
	Inner2 Inner2
	subpkg1.SubPkg1
	SubPkg2 subpkg2.SubPkg2
  pkg1s []*subpkg1.SubPkg1
  pkg2s map[uint32]*ubpkg1.SubPkg1
}

// Inner、Inner2和主结构体在相同的包里面：
type Inner struct {
	Y int64
	X int32

	Z int32
}

type Inner2 struct {
	A int64

	C int64
	B int32
}

// SubPkg1 在另外一个包中 project/testdata/subpkg1.SubPkg1
type SubPkg1 struct {
	Y  int64
	N2 bool
	X  int32
	N  bool
	Z  int32
	N1 bool
	Z1 int32
	N3 bool
	Z3 int32
}

// SubPkg2 在另外一个包中 project/testdata/subpkg1.SubPkg2
type SubPkg2 struct {
	A int64
	C int64
	B int32
}
```

#### 支持备份
在修改源码文件时支持对原先的源码文件进行备份，默认：TRUE，启用该功能，参数：--backup。

#### 支持通过目录和结构体报名联合限定要修改入口结构体

限定工具仅在项目的根目录下执行，可以增加参数指定结构体所在的源码文件，例如：`--source-file`，如果没有指定，就在包内进行查找：
1、`./structoptimizer -struct=writer/config.Context  ./` 表示相对项目根目录 `writer/config` 目录中的某个源码文件中存在的结构体 `Context` 及其字段中引用的结构体进行优化，注意这里只指定到了结构体所在的包，并没有指定源码文件；
2、`./structoptimizer -struct=writer/config.Context  worker/` 表示相对项目根目录 `woker/writer/config` 目录中的某个源码文件中存在的结构体 `Context` 及其字段中引用的结构体进行优化，注意这里只指定到了结构体所在的包，并没有指定源码文件;

#### 支持跳过某些目录或者文件，可以通过通配符进行指定
如果结构体中某些字段引用的结构体不需要优化，我们就跳过。例如，可以通过参数：`--skip-dir alpha  --skip-dir generated_*  --skip-file *_test.go --skip *_pb.go` 跳过某些文件

#### 检测结构体是否有某个方法，包括它的指针类型或者值类型，如果有就跳过
如果结构体或者些字段引用的结构体具有某个方法我们也可以对它不优化。例如，可以通过参数：`--skip-by-methods "Encode_By_KKK,Encode_By_KKK1,Encode_By_KKK2"` 指定，这里提供多个命中一个就行。

#### 需要输出一个报告
输出从主结构体开始，对它及其字段引用的结构体做了哪些优化，一共减少了多少字节，优化前后的字段顺序是什么，可以是txt、md或者可交互式的html。通过 `--outpt` 参数指定

#### 需要通过 -vvv 参数显示执行的过程
通过 -v 、-vv 以及 -vvv 显示Debug日志的等级

#### 需要支持就地修改
可以通过 `--write` 参数指定是否直接将优化结果直接写入源文件。如果没有指定，那么就只生成报告。不修改源码文件，只做分析。

#### 需要支持是否在优化前后结构体大小相同时，是否按照字段大小进行重新排序
如果优化前后结构体大小相同，是否需要进行按照字段从大到小的顺序重排。可以通过 `--sort-same-size` 指定。

#### 可以对指定包或者目录下面的所有结构体进行分析
参数 `--package` 和 参数 `--struct` 互斥，表示扫描指定的包。

例如：`./structoptimizer --package writer/config ./` 表示扫描先对项目根路径 `writer/config` 所在的包中的所有结构体，除 `--skip-dir --skip-file --skip-by-methods`过滤掉的结构体。

#### 需要支持go.mod类型的项目，以及GOPATH+vendor形式的项目
早期的Go项目需要放在GOPATH目录下，依赖放在vendor中，结构体引用的字段可能是第三方的，跨库的，对于跨库的结构体不做优化。较新的项目都是gomod形式管理依赖和组织。这两种类型都要支持。


### 实现步骤

1）输出设计文档，以及修改README.md；
2）输出执行计划；
3）实现代码；
4）设计测试案例并进行自我验证；




