# StructOptimizer 设计文档

## 1. 概述

StructOptimizer 是一个用于优化 Go 项目结构体字段对齐的静态分析工具。通过重新排列结构体字段顺序，减少内存填充（padding），从而降低内存占用。

### 1.1 问题背景

在大型 Go 项目中，开发人员可能没有充分考虑结构体字段对齐问题，导致浪费大量内存：

```go
// 优化前：32 字节
type BadStruct struct {
    A bool   // 1 字节 + 7 字节填充
    B int64  // 8 字节
    C int32  // 4 字节
    D bool   // 1 字节 + 3 字节填充
    E int32  // 4 字节
    // 4 字节末尾填充
}

// 优化后：24 字节（节省 25%）
type GoodStruct struct {
    B int64  // 8 字节
    C int32  // 4 字节
    E int32  // 4 字节
    A bool   // 1 字节
    D bool   // 1 字节
    // 6 字节末尾填充
}
```

### 1.2 解决方案

- 自动分析结构体字段布局
- 智能重排字段顺序
- 支持嵌套结构体优化
- 支持跨包引用优化
- 生成详细优化报告

## 2. 核心原理

### 2.1 Go 结构体内存对齐规则

1. **字段对齐**：每个字段根据其类型大小进行对齐
   - `bool`, `int8`: 1 字节对齐
   - `int16`: 2 字节对齐
   - `int32`, `float32`: 4 字节对齐
   - `int64`, `float64`: 8 字节对齐

2. **结构体对齐**：结构体总大小必须是其最大字段对齐要求的倍数

3. **填充计算**：
   ```
   偏移量 = (当前偏移 + 对齐 - 1) / 对齐 * 对齐
   总大小 = (总大小 + 最大对齐 - 1) / 最大对齐 * 最大对齐
   ```

### 2.2 优化策略

| 策略 | 说明 | 实现 |
|------|------|------|
| 字段重排 | 按字段大小从大到小排序 | `ReorderFields()` |
| 深度优先 | 递归优化嵌套结构体 | `optimizeStruct()` |
| 去重优化 | 相同结构体只优化一次 | `optimized` map |
| 跳过规则 | 支持多种跳过条件 | `shouldSkip()` |

## 3. 系统架构

### 3.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         StructOptimizer                                  │
│                    Go 结构体对齐优化工具                                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────────┐   ┌───────────────────┐   ┌───────────────────┐
│    输入层          │   │    处理层          │   │    输出层          │
│                   │   │                   │   │                   │
│  - 命令行参数      │   │  - 包分析         │   │  - 优化报告       │
│  - 项目目录        │   │  - 结构体查找     │   │  - 源码修改       │
│  - 配置选项        │   │  - 字段优化       │   │  - 文件备份       │
└───────────────────┘   └───────────────────┘   └───────────────────┘
```

### 3.2 模块架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│  main.go (CLI 入口)                                                      │
│  - 参数解析 (flag)                                                       │
│  - 模块协调                                                              │
└─────────────────────────────────────────────────────────────────────────┘
         │
         ├──────────────────┬──────────────────┬──────────────────┐
         │                  │                  │                  │
         ▼                  ▼                  ▼                  ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────┐
│   analyzer      │ │   optimizer     │ │   reporter      │ │   writer    │
│   分析器模块     │ │   优化器模块     │ │   报告模块      │ │   写入模块  │
│                 │ │                 │ │                 │ │             │
│ • LoadPackage   │ │ • Optimize      │ │ • GenerateMD    │ │ • Backup    │
│ • FindStruct    │ │ • ReorderFields │ │ • GenerateTXT   │ │ • Rewrite   │
│ • HasMethod     │ │ • CalcSize      │ │ • GenerateHTML  │ │ • Format    │
└─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────┘
         │                  │
         │                  ├─────────────────────────────┐
         │                  │                             │
         ▼                  ▼                             ▼
┌─────────────────┐ ┌─────────────────┐         ┌─────────────────┐
│   internal/     │ │   optimizer/    │         │   testdata/     │
│   utils         │ │   • field.go    │         │   测试数据       │
│   工具函数       │ │   • size.go     │         │                 │
│                 │ │   • optimizer   │         │ • basic/        │
│ • MatchPattern  │ │                 │         │ • nested/       │
│ • FormatSize    │ │                 │         │ • crosspkg/     │
│ • ShouldSkip    │ │                 │         │ • complexpkg/   │
└─────────────────┘ └─────────────────┘         └─────────────────┘
```

### 3.3 数据流图

```
                    ┌──────────────┐
                    │  用户输入     │
                    │  命令行参数   │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   CLI 解析    │
                    │  (main.go)   │
                    └──────┬───────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌────────────┐ ┌────────────┐ ┌────────────┐
    │  Analyzer  │ │ Optimizer  │ │  Reporter  │
    │  加载包     │ │ 优化结构体  │ │ 生成报告   │
    └─────┬──────┘ └─────┬──────┘ └─────┬──────┘
          │              │               │
          │              │               │
          ▼              ▼               ▼
    ┌────────────┐ ┌────────────┐ ┌────────────┐
    │ 包信息      │ │ 优化结果   │ │ MD/TXT/   │
    │ AST        │ │ 字段顺序   │ │ HTML       │
    │ 类型信息   │ │ 内存节省   │ │            │
    └────────────┘ └────────────┘ └────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │    Writer    │
                    │  写入文件     │
                    │  备份原文件   │
                    └──────────────┘
```

### 3.4 优化流程图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        优化流程                                         │
└─────────────────────────────────────────────────────────────────────────┘

    ┌─────────┐
    │  开始   │
    └────┬────┘
         │
         ▼
    ┌─────────────┐
    │ 加载包信息   │
    └─────┬───────┘
          │
          ▼
    ┌─────────────┐     ┌──────────┐
    │ 查找结构体   │────▶│ 已优化？  │───┐
    └─────┬───────┘     └────┬─────┘   │
          │                  │否       │
          │                  ▼         │
          │            ┌──────────┐    │
          │            │ 应跳过？  │────┤
          │            └────┬─────┘    │
          │                 │否        │
          │                 ▼          │
          │           ┌────────────┐   │
          │           │ 优化嵌套   │   │
          │           │ 字段结构体  │   │
          │           └─────┬──────┘   │
          │                 │          │
          │                 ▼          │
          │           ┌────────────┐   │
          │           │ 重排字段   │   │
          │           └─────┬──────┘   │
          │                 │          │
          │                 ▼          │
          │           ┌────────────┐   │
          │           │ 计算大小   │   │
          │           └─────┬──────┘   │
          │                 │          │
          │                 ▼          │
          └────────────▶┌────────┐◀────┘
                        │ 记录   │
                        │ 结果   │
                        └───┬────┘
                            │
                            ▼
                      ┌─────────┐
                      │  结束   │
                      └─────────┘
```

### 3.5 模块调用关系图

```
┌─────────────────────────────────────────────────────────────────────────┐
│  main.go                                                                 │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────┐         │
│  │  1. parseFlags()    解析命令行参数                          │         │
│  │  2. NewAnalyzer()   创建分析器                              │         │
│  │  3. NewOptimizer()  创建优化器                              │         │
│  │  4. Optimize()      执行优化                                │         │
│  │  5. Generate()      生成报告                                │         │
│  │  6. WriteFiles()    写入文件                                │         │
│  └────────────────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────────────┘
         │
         │ calls
         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  analyzer/analyzer.go                                                    │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────┐         │
│  │  LoadPackage()      加载包及其依赖                          │         │
│  │  FindStructByName() 查找指定结构体                         │         │
│  │  FindAllStructs()   查找所有结构体                          │         │
│  │  HasMethod()        检查结构体方法                          │         │
│  └────────────────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────────────┘
         │
         │ uses
         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  optimizer/optimizer.go                                                  │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────┐         │
│  │  Optimize()         优化入口                               │         │
│  │  optimizeStruct()   优化单个结构体（递归）                  │         │
│  │  shouldSkip()       检查是否跳过                            │         │
│  │  ReorderFields()    重排字段                                │         │
│  │  isStructType()     判断结构体类型                          │         │
│  └────────────────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────────────┘
```

## 4. 模块设计

### 4.1 CLI 模块 (`cmd/structoptimizer/main.go`)

```go
type Config struct {
    Struct          string   // 结构体名称
    Package         string   // 包路径
    SourceFile      string   // 源文件路径
    Write           bool     // 是否写入源文件
    Backup          bool     // 是否备份
    SkipDirs        []string // 跳过的目录
    SkipFiles       []string // 跳过的文件
    SkipByMethods   []string // 跳过的方法
    Output          string   // 报告输出路径
    Verbose         int      // 详细程度
    SortSameSize    bool     // 大小相同重排
    TargetDir       string   // 目标目录
}
```

**职责**：
- 解析命令行参数
- 协调各模块工作
- 错误处理和日志输出

### 4.2 分析器模块 (`analyzer/analyzer.go`)

```go
type Analyzer struct {
    config     *Config
    fset       *token.FileSet
    info       *types.Info
    pkg        *packages.Package
    pkgMap     map[string]*packages.Package
}
```

**核心方法**：
- `LoadPackage()`: 加载包及其依赖
- `FindStructByName()`: 查找指定结构体
- `FindAllStructs()`: 查找包中所有结构体
- `HasMethod()`: 检查结构体是否有指定方法

**依赖**：
- `golang.org/x/tools/go/packages`
- `go/ast`, `go/types`, `go/token`

### 4.3 优化器模块 (`optimizer/`)

#### 4.3.1 字段信息 (`field.go`)

```go
type FieldInfo struct {
    Name     string      // 字段名
    Type     types.Type  // 字段类型
    Size     int64       // 字段大小
    Align    int64       // 对齐要求
    IsEmbed  bool        // 是否匿名
    PkgPath  string      // 类型包路径
    TypeName string      // 类型名称
}
```

#### 4.3.2 大小计算 (`size.go`)

```go
func CalcStructSize(st *types.Struct) int64
func CalcFieldSize(typ types.Type) (size, align int64)
func CalcOptimizedSize(fields []FieldInfo) int64
```

#### 4.3.3 核心优化 (`optimizer.go`)

```go
type Optimizer struct {
    config      *Config
    analyzer    *analyzer.Analyzer
    optimized   map[string]*StructInfo
    report      *Report
}

func (o *Optimizer) Optimize() (*Report, error)
func (o *Optimizer) optimizeStruct(pkgPath, structName string, depth int)
```

### 4.4 报告模块 (`reporter/reporter.go`)

```go
type Report struct {
    TotalStructs   int
    OptimizedCount int
    SkippedCount   int
    TotalSaved     int64
    StructReports  []*StructReport
}
```

**支持的格式**：
- Markdown (默认)
- TXT
- HTML

### 4.5 写入模块 (`writer/writer.go`)

```go
type SourceWriter struct {
    config *Config
    fset   *token.FileSet
}

func (w *SourceWriter) BackupFile(filePath string) (string, error)
func (w *SourceWriter) RewriteFile(filePath string, optimized map[string]*StructInfo) error
```

## 5. 算法设计

### 5.1 字段大小计算算法

```go
func CalcFieldSize(typ types.Type) (size, align int64) {
    switch t := typ.(type) {
    case *types.Basic:
        return basicSize(t.Kind())
    case *types.Pointer:
        return sizeof(uintptr), alignof(uintptr)
    case *types.Struct:
        return CalcStructSize(t), structAlign(t)
    case *types.Slice:
        return sizeof(slice), alignof(slice)
    case *types.Map:
        return sizeof(map), alignof(map)
    // ... 其他类型
    }
}
```

### 5.2 结构体大小计算算法

```go
func CalcStructSize(st *types.Struct) int64 {
    var offset int64 = 0
    var maxAlign int64 = 1
    
    for i := 0; i < st.NumFields(); i++ {
        field := st.Field(i)
        size, align := CalcFieldSize(field.Type())
        
        // 对齐偏移
        if offset % align != 0 {
            offset += align - (offset % align)
        }
        
        offset += size
        if align > maxAlign {
            maxAlign = align
        }
    }
    
    // 末尾填充
    if offset % maxAlign != 0 {
        offset += maxAlign - (offset % maxAlign)
    }
    
    return offset
}
```

### 5.3 字段重排算法

```go
func ReorderFields(fields []FieldInfo, sortSameSize bool) []FieldInfo {
    // 1. 分离匿名字段和命名字段
    var embeds, named []FieldInfo
    
    // 2. 对命名字段排序（按大小降序）
    sort.Slice(named, func(i, j int) bool {
        if named[i].Size != named[j].Size {
            return named[i].Size > named[j].Size
        }
        if sortSameSize {
            return named[i].Align > named[j].Align
        }
        return false
    })
    
    // 3. 合并：匿名 + 命名
    return append(embeds, named...)
}
```

### 5.4 深度优先优化算法

```go
func optimizeStruct(pkgPath, structName string, depth int) {
    // 1. 检查是否已优化
    if _, ok := optimized[key]; ok {
        return
    }
    
    // 2. 检查是否应跳过
    if shouldSkip() {
        return
    }
    
    // 3. 递归优化嵌套结构体（深度优先）
    for _, field := range fields {
        if isStructType(field.Type) {
            optimizeStruct(field.PkgPath, field.TypeName, depth+1)
        }
    }
    
    // 4. 重排当前结构体字段
    ReorderFields()
}
```

## 6. 接口设计

### 6.1 命令行接口

```bash
# 基本用法
./structoptimizer [flags] [directory]

# 优化单个结构体
./structoptimizer -struct=writer/config.Context ./

# 优化整个包
./structoptimizer --package writer/config ./

# 优化并写入
./structoptimizer -struct=writer/config.Context --write --backup ./

# 跳过某些文件
./structoptimizer --package writer/config \
    -skip "*.pb.go" \
    -skip "*_test.go" \
    ./
```

### 6.2 配置接口

```go
type Config interface {
    GetStructName() string
    GetPackage() string
    ShouldWrite() bool
    ShouldBackup() bool
    GetSkipDirs() []string
    GetSkipFiles() []string
    GetVerboseLevel() int
}
```

### 6.3 报告接口

```go
type Reporter interface {
    Generate(report *Report) error
    GenerateMD(report *Report) (string, error)
    GenerateTXT(report *Report) (string, error)
    GenerateHTML(report *Report) (string, error)
}
```

## 7. 测试设计

### 7.1 测试策略

| 模块 | 测试重点 | 覆盖率目标 |
|------|---------|-----------|
| utils | 工具函数边界条件 | >90% |
| optimizer | 大小计算、字段重排 | >80% |
| reporter | 报告格式正确性 | >80% |
| writer | 文件操作正确性 | >70% |
| analyzer | 包加载、结构体查找 | >60% |

### 7.2 测试用例分类

1. **单元测试**：测试单个函数/方法
2. **集成测试**：测试模块间协作
3. **端到端测试**：测试完整流程

### 7.3 测试数据

```
testdata/
├── basic/              # 基础测试用例
│   ├── basic.go        # 简单结构体
│   └── ...
├── nested/             # 嵌套测试用例
│   ├── nested.go       # 2-3 层嵌套
│   └── deep_nested.go  # 5+ 层嵌套
├── crosspkg/           # 跨包测试用例
│   ├── subpkg1/        # 子包 1
│   ├── subpkg2/        # 子包 2
│   └── crosspkg.go     # 跨包引用
├── complexpkg/         # 复杂测试用例
│   └── complex.go      # slice/map/指针
└── methodskip/         # 方法跳过测试
    └── methodskip.go   # 带方法的结构体
```

## 8. 性能考虑

### 8.1 时间复杂度

| 操作 | 复杂度 | 说明 |
|------|--------|------|
| 包加载 | O(n) | n 为文件数 |
| 结构体查找 | O(n*m) | n 为文件数，m 为声明数 |
| 字段重排 | O(k log k) | k 为字段数 |
| 嵌套优化 | O(d*k) | d 为深度，k 为结构体数 |

### 8.2 空间复杂度

| 数据结构 | 复杂度 | 说明 |
|---------|--------|------|
| pkgMap | O(p) | p 为包数 |
| optimized | O(s) | s 为结构体数 |
| AST | O(n) | n 为节点数 |

### 8.3 优化策略

1. **缓存已加载包**：避免重复加载
2. **缓存已优化结构体**：避免重复优化
3. **并发处理**：未来可支持多包并发处理

## 9. 扩展性设计

### 9.1 新增报告格式

```go
// 实现 Reporter 接口
type JSONReporter struct{}

func (r *JSONReporter) Generate(report *Report) error {
    // 实现 JSON 格式报告
}
```

### 9.2 新增跳过规则

```go
// 在 shouldSkip 中添加新规则
func (o *Optimizer) shouldSkip() string {
    // ... 现有规则
    
    // 新增规则
    if hasTag(field, "protobuf") {
        return "protobuf 结构体"
    }
    
    return ""
}
```

### 9.3 新增优化策略

```go
// 实现新的重排策略
func ReorderFieldsCustom(fields []FieldInfo, strategy Strategy) []FieldInfo {
    switch strategy {
    case SizeDesc:
        // 按大小降序
    case AlignDesc:
        // 按对齐降序
    case Custom:
        // 自定义策略
    }
}
```

## 10. 依赖管理

### 10.1 核心依赖

```go
require (
    golang.org/x/tools v0.17.0  // packages.Load
    golang.org/x/mod v0.14.0    // 间接依赖
)
```

### 10.2 标准库依赖

- `go/ast`: AST 解析
- `go/types`: 类型检查
- `go/token`: 词法分析
- `go/parser`: 代码解析
- `go/printer`: 代码格式化

## 11. 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| v0.1.0 | 2024-01 | 初始版本，核心功能实现 |
| v0.2.0 | 2024-01 | 添加嵌套优化、跨包优化 |
| v0.3.0 | 2024-01 | 添加报告生成、文件写入 |

## 12. 待办事项

- [ ] 支持并发处理多个包
- [ ] 支持 JSON 格式报告
- [ ] 支持自定义优化策略
- [ ] 支持配置文件
- [ ] 支持增量优化
- [ ] 支持更多跳过规则
