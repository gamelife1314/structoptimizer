# StructOptimizer

[![Test](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml)
[![Release](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gamelife1314/structoptimizer)](https://goreportcard.com/report/github.com/gamelife1314/structoptimizer)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[English](README.md) | **中文文档**

Go 结构体对齐优化工具 - 通过重新排列结构体字段顺序，减少内存填充，降低内存占用。

## 背景

在大型 Go 项目中，开发人员可能没有充分考虑结构体字段对齐问题，导致浪费大量内存。在内存价格昂贵的今天，这种优化显得尤为重要。

参考：[golang/tools fieldalignment](https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go)

但官方工具过于简单，无法处理：
- 嵌套结构体优化
- 跨包引用的结构体优化
- 深度优先的多层嵌套优化

本工具旨在解决这些问题。

## 功能特性

### 核心功能

- ✅ 对 Go 语言定义的结构体进行字段对齐优化
- ✅ 支持结构体中的命名字段和匿名字段
- ✅ 支持跨包引用的结构体优化
- ✅ 深度优先优化嵌套结构体（支持多层嵌套）
- ✅ 相同结构体只优化一次（去重）

### 高级功能

- ✅ 支持备份源文件（`-backup`）
- ✅ 支持通过目录和结构体名联合限定优化目标
- ✅ 支持跳过某些目录或文件（通配符匹配）
- ✅ 支持通过方法名跳过特定结构体（`-skip-by-methods`）
- ✅ 生成优化报告（TXT/MD/HTML 格式）
- ✅ 支持详细日志输出（`-v`, `-vv`, `-vvv`）
- ✅ 支持就地修改源文件（`-write`）
- ✅ 支持大小相同时按字段大小重排（`-sort-same-size`）
- ✅ 支持分析指定包下的所有结构体（`-package`）
- ✅ 支持 go.mod 项目和 GOPATH+vendor 项目
- ✅ 自动跳过 vendor 中的第三方库结构体（不优化）
- ✅ 自动跳过 Go 标准库结构体（不优化）
- ✅ 智能识别项目包，只优化用户自己的代码

## 项目支持

### Go Modules 项目（推荐）

```bash
# 指定项目根目录（包含 go.mod）
./structoptimizer -struct=example.com/myapp/pkg.Context /path/to/project

# 或在项目目录内执行（可省略目录参数）
cd /path/to/project
./structoptimizer -struct=example.com/myapp/pkg.Context
```

### GOPATH + Vendor 项目

对于使用 GOPATH + vendor 的早期项目，需要使用 `-prj-type=gopath` 参数指定：

```bash
# 使用 -prj-type=gopath 指定项目类型
# -pkg-scope 用于限制分析范围，只分析指定包路径前缀的结构体
./structoptimizer -prj-type=gopath -package example.com/myproject/pkg -pkg-scope example.com/myproject

# 可选：指定 GOPATH 路径（否则使用环境变量）
./structoptimizer -prj-type=gopath -gopath=/path/to/gopath -struct=example.com/myproject/pkg.MyStruct -pkg-scope example.com/myproject
```

**注意**：
- GOPATH 项目**不需要指定项目目录**参数
- **`-pkg-scope` 是必填参数**，用于限制分析范围，防止分析到 GOPATH 下其他项目
- `-pkg-scope` 填写你的项目包路径前缀，例如 `example.com/myproject`
- vendor 目录中的第三方库结构体**不会被优化**（符合需求）
- 项目中引用 vendor 结构体的字段会保留原顺序
- 如果尝试直接优化 vendor 中的结构体，会被跳过并显示原因

## 安装

### 方式 1: 下载预编译二进制（推荐）

从 [GitHub Releases](https://github.com/gamelife1314/structoptimizer/releases) 下载适合您平台的预编译二进制文件：

| 平台 | 架构 | 文件名 |
|------|------|--------|
| Linux | amd64 | `structoptimizer-linux-amd64.tar.gz` |
| Linux | arm64 | `structoptimizer-linux-arm64.tar.gz` |
| macOS | amd64 | `structoptimizer-darwin-amd64.tar.gz` |
| macOS | arm64 (Apple Silicon) | `structoptimizer-darwin-arm64.tar.gz` |
| Windows | amd64 | `structoptimizer-windows-amd64.zip` |

下载后解压并添加到 PATH：

```bash
# Linux/macOS
tar -xzf structoptimizer-*.tar.gz
sudo mv structoptimizer-* /usr/local/bin/structoptimizer

# Windows
# 解压后添加到 PATH
```

### 方式 2: 使用 go install

```bash
go install github.com/gamelife1314/structoptimizer/cmd/structoptimizer@latest
```

### 方式 3: 从源码构建

```bash
git clone https://github.com/gamelife1314/structoptimizer.git
cd structoptimizer
go build -o structoptimizer ./cmd/structoptimizer
```

## 快速开始

### 基本用法

优化单个结构体（Go Module 项目）：

```bash
# 指定项目目录
./structoptimizer -struct=pkg.Context /path/to/project
```

### 常用命令

```bash
# 1. 优化单个结构体（生成报告，不修改源码）
./structoptimizer -struct=pkg.Context /path/to/project

# 2. 优化并直接写入源文件（默认备份）
./structoptimizer -struct=pkg.Context -write -backup /path/to/project

# 3. 优化并备份源文件
./structoptimizer -struct=pkg.Context -write -backup /path/to/project

# 4. 优化并写入源文件，不备份
./structoptimizer -struct=pkg.Context -write -backup=false /path/to/project

# 5. 优化指定包中的所有结构体
./structoptimizer -package pkg /path/to/project

# 6. 优化指定包并写入源文件
./structoptimizer -package pkg -write -backup /path/to/project

# 7. 跳过某些目录和文件
./structoptimizer -struct=pkg.Context \
    -skip-dir alpha \
    -skip-dir generated_* \
    -skip-file *_test.go \
    -skip-file *_pb.go \
    /path/to/project

# 7.1 跳过 vendor 目录（松散匹配）
./structoptimizer -package pkg -skip-dir vendor /path/to/project

# 8. 跳过具有特定方法的结构体
./structoptimizer -struct=pkg.Context \
    -skip-by-methods "Encode_By_KKK,Encode_By_KKK1" \
    /path/to/project

# 9. 跳过指定名称的结构体
./structoptimizer -package pkg \
    -skip-by-names "BadStruct,UnusedStruct" \
    /path/to/project

# 10. 生成报告到指定文件
./structoptimizer -struct=pkg.Context \
    -output report.md \
    /path/to/project

# 11. 显示详细执行过程
./structoptimizer -struct=pkg.Context -vvv /path/to/project

# 12. 优化前后大小相同时按字段大小重排
./structoptimizer -struct=pkg.Context -sort-same-size /path/to/project

# 13. GOPATH 项目
./structoptimizer -prj-type=gopath -struct=example.com/pkg.MyStruct

# 14. GOPATH 项目，指定 GOPATH 路径
./structoptimizer -prj-type=gopath -gopath=/path/to/gopath -struct=example.com/pkg.MyStruct
```

### 原地修改和备份

使用 `-write` 参数可以直接将优化结果写入源文件，`-backup` 参数（默认启用）会在修改前备份源文件。

```bash
# 优化并写入源文件，同时创建备份
./structoptimizer -package pkg -write -backup /path/to/project

# 备份文件示例：
#   原文件：pkg/context.go
#   备份文件：pkg/context.go.bak

# 优化并写入源文件，不创建备份
./structoptimizer -package pkg -write -backup=false /path/to/project
```

**注意**：
- 备份文件名为 `原文件名.bak`
- 建议始终启用备份功能（默认），以便在需要时恢复原始代码
- 优化后的代码会使用 `go/printer` 格式化，保持代码风格一致

## 命令行参数

### -skip-dirs 参数说明

`-skip-dirs` 参数用于跳过指定目录中的结构体，支持**双重匹配机制**。

#### 匹配规则

`-skip-dirs` 使用**双重匹配机制**：

1. **basename 匹配**：匹配目录的 basename（最后一部分）
2. **路径包含匹配**：匹配完整路径中是否包含该目录名（作为完整路径组件）

**匹配逻辑**：
```go
func shouldSkipDir(dirPath string) bool {
    baseName := filepath.Base(dirPath)
    normalizedPath := filepath.ToSlash(dirPath)

    for _, pattern := range SkipDirs {
        // 1. basename 匹配
        if matched, _ := filepath.Match(pattern, baseName); matched {
            return true
        }
        // 2. 路径包含匹配（要求完整路径组件）
        if strings.Contains(normalizedPath, "/"+pattern+"/") ||
           strings.Contains(normalizedPath, "/"+pattern) ||
           strings.HasSuffix(normalizedPath, "/"+pattern) {
            return true
        }
    }
    return false
}
```

#### 使用示例

```bash
# 跳过所有 vendor 目录
./structoptimizer -package writer/config -skip-dirs vendor ./

# 以下路径都会被跳过：
# ✓ /project/vendor/lib.go                  # basename 匹配
# ✓ /project/pkg/vendor/lib.go              # basename 匹配
# ✓ /a/b/c/vendor/github.com/lib/lib.go     # 路径包含匹配

# 使用通配符
./structoptimizer -package writer/config -skip-dirs "generated_*" ./

# 以下路径都会被跳过：
# ✓ /project/generated/proto.go             # basename 匹配 generated
# ✓ /project/generated_proto/api.go         # basename 匹配 generated_*
# ✓ /src/generated/proto/api.go             # 路径包含匹配

# 跳过多个目录（逗号分隔）
./structoptimizer -package writer/config -skip-dirs "vendor,generated_*,datas" ./
```

### 完整参数列表

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `<项目目录>` | Go Module 项目的根目录（GOPATH 项目可省略） | - |
| `-struct` | 结构体名称（格式：包路径。结构体名） | - |
| `-package` | 包路径（与 `-struct` 互斥） | - |
| `-source-file` | 源文件路径（限定在指定文件中查找结构体） | - |
| `-write` | 直接写入源文件 | false |
| `-backup` | 修改前备份源文件 | true |
| `-skip-dir` | 跳过的目录（支持通配符，匹配路径中任意目录组件） | - |
| `-skip-file` | 跳过的文件（支持通配符） | - |
| `-skip` | 跳过的文件模式 | - |
| `-skip-by-methods` | 具有这些方法的结构体跳过（支持通配符） | - |
| `-skip-by-names` | 指定名称的结构体跳过（支持通配符） | - |
| `-output` | 报告输出路径 | stdout |
| `-v`, `-vv`, `-vvv` | 详细程度 | 0 |
| `-sort-same-size` | 大小相同时按字段大小重排 | false |
| `-prj-type` | 项目类型（gomod/gopath） | gomod |
| `-gopath` | GOPATH 路径（GOPATH 项目可选） | 环境变量 GOPATH |
| `-help` | 显示帮助 | - |

## 示例

### 优化前

```go
type BadStruct struct {
    A bool   // 1 字节
    // [填充 7 字节]
    B int64  // 8 字节
    C int32  // 4 字节
    D bool   // 1 字节
    // [填充 3 字节]
    E int32  // 4 字节
    // [末尾填充 4 字节]
}
// 总计：32 字节
```

### 优化后

```go
type GoodStruct struct {
    B int64  // 8 字节（偏移量 0）
    C int32  // 4 字节（偏移量 8）
    E int32  // 4 字节（偏移量 12）
    A bool   // 1 字节（偏移量 16）
    D bool   // 1 字节（偏移量 17）
    // [末尾填充 6 字节]
}
// 总计：24 字节（节省 8 字节）
```

### 嵌套结构体优化

```go
// 主结构体：project/testdata.NestedOuter
type NestedOuter struct {
    Name   string
    Inner  Inner
    Count  int64
    Inner2 Inner2
    subpkg1.SubPkg1
    SubPkg2 subpkg2.SubPkg2
    pkg1s  []*subpkg1.SubPkg1
    pkg2s  map[uint32]*subpkg1.SubPkg1
}

// 同包中的结构体
type Inner struct {
    Y int64
    X int32
    Z int32
}

// 跨包结构体（project/testdata/subpkg1.SubPkg1）
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
```

工具会深度优先地优化所有嵌套的结构体。

## 输出报告

### Markdown 格式示例

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
```

## 项目结构

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
├── VERIFICATION_CHECKLIST.md # 修改验证清单
├── design.md                # 设计文档
└── README.md                # 使用说明
```

## 技术原理

### Go 结构体内存对齐规则

1. 每个字段根据其类型大小进行对齐（如 `int64` 需要 8 字节对齐）
2. 结构体总大小必须是其最大字段对齐要求的倍数
3. 不当的字段顺序会导致大量内存填充

### 优化策略

1. **字段重排**：按字段大小从大到小排序
2. **深度优先**：递归优化嵌套结构体
3. **去重优化**：相同结构体只优化一次

## 边界情况处理

- 泛型结构体：跳过不优化
- 外部包结构体：跳过不优化（跨库）
- 循环引用：检测并避免无限递归
- 字段有 tag：保留原有 tag
- 空结构体：跳过
- 只有一个字段：跳过

## CI/CD

本项目使用 GitHub Actions 实现：
- **测试**：推送到 main/master 和 PR 时自动运行
- **发布**：推送标签时自动构建多平台二进制文件

配置见 `.github/workflows/` 目录。

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
