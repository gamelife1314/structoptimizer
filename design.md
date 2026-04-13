# StructOptimizer 设计文档

## 1. 概述

StructOptimizer 是一个用于优化 Go 项目结构体字段对齐的静态分析工具。通过重新排列结构体字段顺序，减少内存填充（padding），从而降低内存占用。

## 2. 核心原理

### 2.1 Go 结构体内存对齐规则

- 每个字段根据其类型大小进行对齐（如 `int64` 需要 8 字节对齐）
- 结构体总大小必须是其最大字段对齐要求的倍数
- 不当的字段顺序会导致大量内存填充

### 2.2 优化策略

1. **字段重排**：按字段大小从大到小排序
2. **深度优先**：递归优化嵌套结构体
3. **去重优化**：相同结构体只优化一次

## 3. 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI 层 (cmd)                           │
│  - 参数解析 (flag)                                          │
│  - 命令分发                                                 │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    分析器层 (analyzer)                       │
│  - 包解析 (packages.Load)                                   │
│  - AST 遍历                                                  │
│  - 类型信息收集 (types.Info)                                │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   优化器层 (optimizer)                       │
│  - 字段分析 (FieldAnalyzer)                                 │
│  - 字段重排 (FieldReorderer)                                │
│  - 嵌套结构体处理 (NestedStructHandler)                     │
│  - 跨包引用处理 (CrossPackageResolver)                      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    报告层 (reporter)                         │
│  - 报告生成 (TXT/MD/HTML)                                   │
│  - 内存节省统计                                             │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   写入器层 (writer)                          │
│  - 源码修改                                                 │
│  - 文件备份                                                 │
│  - AST 重写                                                  │
└─────────────────────────────────────────────────────────────┘
```

## 4. 模块设计

### 4.1 CLI 模块 (`cmd/`)

```go
type Config struct {
    Struct          string   // 结构体名称 (包路径。结构体名)
    Package         string   // 包路径
    SourceFile      string   // 源文件路径
    Write           bool     // 是否写入源文件
    Backup          bool     // 是否备份
    SkipDirs        []string // 跳过的目录
    SkipFiles       []string // 跳过的文件
    SkipByMethods   []string // 具有这些方法的结构体跳过
    Output          string   // 报告输出路径
    Verbose         int      // 详细程度 (0-3)
    SortSameSize    bool     // 大小相同时是否重排
    TargetDir       string   // 目标目录
}
```

### 4.2 分析器模块 (`analyzer/`)

```go
type Analyzer struct {
    config *Config
    fset   *token.FileSet
    info   *types.Info
    pkg    *packages.Package
}

// 主要功能：
// - LoadPackage: 加载包及其依赖
// - FindStructs: 查找结构体定义
// - GetStructFields: 获取结构体字段信息
// - HasMethod: 检查结构体是否有指定方法
```

### 4.3 优化器模块 (`optimizer/`)

```go
type FieldInfo struct {
    Name     string
    Type     types.Type
    Size     int64
    Align    int64
    Offset   int64
    IsEmbed  bool     // 是否匿名字段
    PkgPath  string   // 字段类型所在包
}

type StructInfo struct {
    Name       string
    PkgPath    string
    File       string
    Fields     []FieldInfo
    OrigSize   int64
    OptSize    int64
    Optimized  bool
}

type Optimizer struct {
    config      *Config
    optimized   map[string]*StructInfo  // 已优化的结构体
    report      *Report
}

// 主要功能：
// - AnalyzeStruct: 分析结构体字段
// - CalculateSize: 计算结构体大小
// - ReorderFields: 重排字段
// - OptimizeNested: 优化嵌套结构体
// - ResolveCrossPackage: 解析跨包引用
```

### 4.4 报告模块 (`reporter/`)

```go
type Report struct {
    TotalStructs   int
    OptimizedCount int
    TotalSaved     int64
    StructReports  []*StructReport
}

type StructReport struct {
    Name        string
    File        string
    OrigSize    int64
    OptSize     int64
    Saved       int64
    OrigFields  []string
    OptFields   []string
    Skipped     bool
    SkipReason  string
}

type Reporter struct {
    format string  // txt, md, html
}
```

### 4.5 写入器模块 (`writer/`)

```go
type SourceWriter struct {
    config *Config
    fset   *token.FileSet
}

// 主要功能：
// - BackupFile: 备份源文件
// - WriteStruct: 写入优化后的结构体
// - RewriteFile: 重写整个文件
```

## 5. 核心算法

### 5.1 字段大小计算

```go
func calculateFieldSize(typ types.Type) (size, align int64) {
    switch t := typ.(type) {
    case *types.Basic:
        return basicSize(t.Kind())
    case *types.Pointer:
        return unsafe.Sizeof(uintptr(0)), unsafe.Alignof(uintptr(0))
    case *types.Struct:
        return calcStructSize(t)
    case *types.Array:
        elemSize, elemAlign := calculateFieldSize(t.Elem())
        return elemSize * t.Len(), elemAlign
    case *types.Slice:
        return unsafe.Sizeof([]int{}), unsafe.Alignof([]int{})
    case *types.Map:
        return unsafe.Sizeof(map[int]int{}), unsafe.Alignof(map[int]int{})
    case *types.Chan:
        return unsafe.Sizeof(make(chan int)), unsafe.Alignof(make(chan int))
    case *types.Interface:
        return unsafe.Sizeof((*interface{})(nil)), unsafe.Alignof((*interface{})(nil))
    case *types.Named:
        return calculateFieldSize(t.Underlying())
    // ... 处理其他类型
    }
}
```

### 5.2 结构体大小计算（考虑填充）

```go
func calcStructSize(st *types.Struct) (totalSize int64) {
    var offset int64 = 0
    var maxAlign int64 = 1
    
    for i := 0; i < st.NumFields(); i++ {
        field := st.Field(i)
        size, align := calculateFieldSize(field.Type())
        
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
func reorderFields(fields []FieldInfo, sortSameSize bool) []FieldInfo {
    // 1. 分离匿名字段和命名字段
    // 2. 按大小降序排序（大小相同时，按对齐要求降序）
    // 3. 如果 sortSameSize=true，大小相同的字段按类型名排序
    // 4. 匿名字段保持在前面
    return sorted
}
```

### 5.4 深度优先优化流程

```
1. 从入口结构体开始
2. 检查是否已优化过 → 是则跳过
3. 检查是否被排除 → 是则跳过
4. 检查是否有指定方法 → 有则跳过
5. 分析当前结构体字段
6. 对每个字段类型：
   - 如果是结构体类型，递归优化
7. 重排当前结构体字段
8. 记录优化结果
```

## 6. 命令行参数设计

```bash
./structoptimizer [flags] [directory]

Flags:
  -struct string       结构体名称 (格式：包路径。结构体名)
  -package string      包路径（与 -struct 互斥）
  -source-file string  源文件路径
  -write               直接写入源文件（默认只生成报告）
  -backup              修改前备份源文件（默认 true）
  -skip-dir strings    跳过的目录（支持通配符）
  -skip-file strings   跳过的文件（支持通配符）
  -skip strings        跳过的文件模式
  -skip-by-methods strings  具有这些方法的结构体跳过
  -output string       报告输出路径
  -v, -vv, -vvv        详细程度
  -sort-same-size      大小相同时按字段大小重排
  -help                显示帮助
```

## 7. 项目结构

```
structoptimizer/
├── cmd/
│   └── structoptimizer/
│       └── main.go          # 入口程序
├── analyzer/
│   └── analyzer.go          # 包和类型分析
├── optimizer/
│   ├── optimizer.go         # 核心优化逻辑
│   ├── field.go             # 字段分析
│   └── size.go              # 大小计算
├── reporter/
│   └── reporter.go          # 报告生成
├── writer/
│   └── writer.go            # 源码写入
├── internal/
│   └── utils/
│       └── utils.go         # 工具函数
├── testdata/                 # 测试数据
│   ├── basic/
│   ├── nested/
│   └── crosspkg/
├── design.md                # 设计文档
├── README.md                # 使用说明
└── go.mod
```

## 8. 输出报告格式示例

### Markdown 格式

```markdown
# StructOptimizer Report

## 摘要
- 总结构体数：5
- 已优化：3
- 跳过：2
- 节省内存：128 字节

## 优化详情

### writer/config.Context
- 文件：writer/config/context.go
- 优化前大小：64 字节
- 优化后大小：48 字节
- 节省：16 字节

**优化前字段顺序：**
1. A (bool, 1 字节)
2. B (int64, 8 字节)
3. C (int32, 4 字节)
4. D (bool, 1 字节)
5. E (int32, 4 字节)

**优化后字段顺序：**
1. B (int64, 8 字节)
2. C (int32, 4 字节)
3. E (int32, 4 字节)
4. A (bool, 1 字节)
5. D (bool, 1 字节)

### 跳过的结构体

#### writer/config.SkippedStruct
- 原因：具有方法 Encode_By_KKK
```

## 9. 技术选型

| 组件 | 技术 |
|------|------|
| 参数解析 | `flag` 标准库 |
| Go 代码解析 | `go/packages` + `go/ast` + `go/types` |
| AST 重写 | `go/ast` + `go/printer` |
| 通配符匹配 | `path/filepath.Match` |
| 单元测试 | `testing` |

## 10. 边界情况处理

1. **泛型结构体**：跳过不优化
2. **外部包结构体**：跳过不优化（跨库）
3. **循环引用**：检测并避免无限递归
4. **字段有 tag**：保留原有 tag
5. **空结构体**：跳过
6. **只有一个字段**：跳过
