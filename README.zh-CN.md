# StructOptimizer

<div align="center">

[![Test](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml)
[![Release](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gamelife1314/structoptimizer)](https://goreportcard.com/report/github.com/gamelife1314/structoptimizer)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gamelife1314/structoptimizer)](go.mod)

**优化 Go 结构体字段对齐，减少内存填充，节省内存占用**

[English](README.md) • [中文文档](README.zh-CN.md)

</div>

---

## 📖 简介

StructOptimizer 是一款强大的 Go 结构体内存对齐优化工具。它通过智能重排结构体字段顺序来减少内存填充，帮助你在大型 Go 应用中节省内存。

### 为什么内存优化很重要

在大型 Go 项目中，结构体字段对齐不当会因填充浪费大量内存。看这个例子：

```go
// ❌ 优化前（32 字节，浪费 15 字节）
type User struct {
    Name    string  // 16 字节
    Age     uint8   // 1 字节
    Active  bool    // 1 字节
    Balance float64 // 8 字节
    ID      int64   // 8 字节
    // 编译器插入 14 字节填充
}

// ✅ 优化后（24 字节，节省 8 字节 = 25%）
type User struct {
    Balance float64 // 8 字节
    ID      int64   // 8 字节
    Name    string  // 16 字节
    Age     uint8   // 1 字节
    Active  bool    // 1 字节
    // 仅 6 字节填充
}
```

**规模化效应**：如果你有 100 万个 `User` 实例，仅这一个结构体就能**节省 8 MB 内存**！

---

## ✨ 核心特性

### 基础功能

| 功能 | 说明 |
|------|------|
| 🔧 **字段重排** | 自动重排结构体字段以获得最佳对齐 |
| 📦 **嵌套结构体支持** | 处理深层嵌套的结构体层次（最多 50 层） |
| 🔗 **跨包优化** | 优化跨多个包引用的结构体 |
| 🎯 **智能去重** | 每个结构体只优化一次 |
| 📊 **详细报告** | 生成 TXT/MD/HTML 格式的前后对比报告 |

### 高级功能

| 功能 | 说明 |
|------|------|
| 💾 **自动备份** | 修改源文件前创建带时间戳的备份 |
| ⏭️ **灵活跳过** | 按目录、文件、方法名或结构体名跳过 |
| 🏗️ **双项目支持** | 同时支持 Go Modules 和 GOPATH+vendor 项目 |
| 🛡️ **安全优化** | 仅在确保节省内存时才重排字段 |
| 📝 **预留字段** | 保持特定字段（如 `reserved`、`padding`）在末尾 |
| 🔍 **详细日志** | 多级详细程度（-v、-vv、-vvv）用于调试 |

---

## 🚀 快速开始

### 安装

#### 方式 1：通用安装脚本（推荐）

```bash
# macOS / Linux - 自动检测包管理器，否则直接下载二进制
curl -fsSL https://raw.githubusercontent.com/gamelife1314/structoptimizer/main/install.sh | bash

# 安装指定版本
curl -fsSL https://raw.githubusercontent.com/gamelife1314/structoptimizer/main/install.sh | VERSION=v1.9.1 bash

# 自定义安装目录
curl -fsSL https://raw.githubusercontent.com/gamelife1314/structoptimizer/main/install.sh | INSTALL_DIR=/usr/bin bash
```

#### 方式 2：Homebrew（macOS / Linux）

```bash
brew tap gamelife1314/structoptimizer
brew install structoptimizer
```

#### 方式 3：Go Install

```bash
go install github.com/gamelife1314/structoptimizer/cmd/structoptimizer@latest
```

#### 方式 4：APT / YUM（Linux）

```bash
# Debian / Ubuntu（APT）- 完整包管理器集成
echo "deb [trusted=yes] https://gamelife1314.github.io/structoptimizer/apt stable main" \
  | sudo tee /etc/apt/sources.list.d/structoptimizer.list
sudo apt update && sudo apt install structoptimizer

# RHEL / Fedora（YUM/DNF）- 完整包管理器集成
sudo tee /etc/yum.repos.d/structoptimizer.repo <<EOF
[structoptimizer]
name=StructOptimizer
baseurl=https://gamelife1314.github.io/structoptimizer/yum
enabled=1
gpgcheck=0
EOF
sudo yum install structoptimizer     # 或: sudo dnf install structoptimizer
```

#### 方式 5：手动下载

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/gamelife1314/structoptimizer/releases/latest/download/structoptimizer-darwin-arm64.tar.gz
tar -xzf structoptimizer-darwin-arm64.tar.gz && sudo mv structoptimizer /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/gamelife1314/structoptimizer/releases/latest/download/structoptimizer-darwin-amd64.tar.gz
tar -xzf structoptimizer-darwin-amd64.tar.gz && sudo mv structoptimizer /usr/local/bin/

# Linux (amd64)
curl -LO https://github.com/gamelife1314/structoptimizer/releases/latest/download/structoptimizer-linux-amd64.tar.gz
tar -xzf structoptimizer-linux-amd64.tar.gz && sudo mv structoptimizer /usr/local/bin/

# Linux (arm64)
curl -LO https://github.com/gamelife1314/structoptimizer/releases/latest/download/structoptimizer-linux-arm64.tar.gz
tar -xzf structoptimizer-linux-arm64.tar.gz && sudo mv structoptimizer /usr/local/bin/

# Windows
curl -LO https://github.com/gamelife1314/structoptimizer/releases/latest/download/structoptimizer-windows-amd64.zip
# 解压并添加到 PATH
```

#### 方式 6：从源码构建

```bash
git clone https://github.com/gamelife1314/structoptimizer.git
cd structoptimizer
go build -o structoptimizer ./cmd/structoptimizer
```

### 基本使用

**第 1 步**：分析但不修改（仅生成报告）

```bash
# 优化指定结构体
structoptimizer -struct=example.com/mypkg.User ./myproject

# 优化包中所有结构体
structoptimizer -package=example.com/mypkg ./myproject
```

**第 2 步**：查看生成的报告（`report.md`）

**第 3 步**：应用优化（自动备份）

```bash
structoptimizer -package=example.com/mypkg -write -backup ./myproject
```

---

## 📚 使用指南

### 命令行选项

```
用法：structoptimizer [选项] <项目目录>

选项:
  -struct string        结构体名称（格式：package.path.StructName）
  -package string       包路径（与 -struct 互斥）
  -write                直接写入源文件
  -backup               修改前备份源文件（默认：true）
  -output string        报告输出路径
  -format string        报告格式：txt、md、html（默认：md）
  -skip-dirs string     跳过的目录（支持通配符，逗号分隔）
  -skip-files string    跳过的文件（支持通配符，逗号分隔）
  -skip-by-methods string  跳过具有这些方法的结构体（逗号分隔）
  -skip-by-names string    跳过指定名称的结构体（逗号分隔）
  -reserved-fields string  始终排在最后的字段（逗号分隔）
  -sort-same-size       大小相同时也按字段大小重排
  -max-depth int        最大递归深度（默认 50）
  -timeout int          超时时间（秒，默认 1200）
  -prj-type string      项目类型：gomod、gopath（默认：gomod）
  -pkg-scope string     包范围限制（GOPATH 模式必填，只分析此包内的结构体）
  -pkg-limit int        包并发限制（默认 4，降低可防止 OOM）
  -gopath string        GOPATH 路径（GOPATH 项目可选）
  -recursive            递归扫描子包（仅 -package 模式有效）
  -lang string          报告语言：zh、en（默认：zh）
  -allow-external-pkgs  允许扫描跨包结构体（包括 vendor 目录，默认关闭）
  -v, -vv, -vvv         详细输出级别
  -version              显示版本信息
```

### 常用场景

#### 1. 分析单个结构体

```bash
# 生成报告，不修改源码
structoptimizer -struct=github.com/myapp/models.User ./myproject

# 输出：report.md
```

#### 2. 优化整个包

```bash
# 优化 models 包中的所有结构体
structoptimizer -package=github.com/myapp/models -write -backup ./myproject
```

#### 2.1. 递归包扫描（新增）

```bash
# 递归扫描包及其所有子包
structoptimizer -package=github.com/myapp/pkg -recursive -write -backup ./myproject

# 示例输出：
# - 扫描 github.com/myapp/pkg
# - 扫描 github.com/myapp/pkg/apis
# - 扫描 github.com/myapp/pkg/models
# - 扫描 github.com/myapp/pkg/utils
# - 自动跳过 vendor 和标准库
```

**工作原理：**
- 使用 BFS（广度优先搜索）遍历包依赖图
- 从根包开始，发现所有导入的子包
- 自动跳过标准库和 vendor 包
- 只扫描根包路径下的子包

**使用场景：**
- 大型项目，包含多个子包（50+ 包）
- 深层嵌套的包层次结构（10+ 层）
- GOPATH+vendor 项目
- 想要一次性优化整个模块

#### 2.2. GOPATH 项目：`-pkg-scope` 参数（重要）

**仅 GOPATH 模式**下，`-pkg-scope` 参数是**必填的**：

```bash
# GOPATH 项目 - 必须指定 -pkg-scope
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject \
  -recursive -write -backup
```

**`-pkg-scope` 是什么？**
- 限制分析范围到指定路径前缀下的包
- 防止分析 GOPATH 中不相关的项目
- 与 `-recursive` 配合使用，发现范围内所有子包

**如何设置 `-pkg-scope`：**
1. 确定项目的模块路径（例如：`github.com/myproject`）
2. 使用根路径作为范围（例如：`-pkg-scope=github.com/myproject`）
3. 所有此前缀开头的包都会被包含

**示例：**
```bash
# 项目结构：
# $GOPATH/src/github.com/myproject/
# ├── pkg/
# │   ├── apis/
# │   ├── models/
# │   └── utils/
# └── vendor/

# 正确用法：
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject \
  -recursive

# 这将扫描：
# ✅ github.com/myproject/pkg
# ✅ github.com/myproject/pkg/apis
# ✅ github.com/myproject/pkg/models
# ✅ github.com/myproject/pkg/utils
# ❌ github.com/otherproject/pkg (超出范围)
# ❌ vendor/* (自动跳过)
```

**常见错误：**
```bash
# ❌ 缺少 -pkg-scope（GOPATH 模式下会失败）
structoptimizer -prj-type=gopath -package=github.com/myproject/pkg

# ❌ 范围太窄（找不到子包）
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject/pkg  # 太具体了！

# ✅ 正确：使用项目根路径作为范围
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject
```

**何时使用：**
- 传统 GOPATH 项目（Go Modules 之前）
- 使用 vendor 目录的项目
- 同一 GOPATH 工作区中的多个项目

#### 2.3. 允许跨包扫描（`-allow-external-pkgs`）（新增）

默认情况下，StructOptimizer 会跳过 `-pkg-scope` 范围外的结构体和 vendor 包。使用 `-allow-external-pkgs` 可以将其纳入分析：

```bash
# GOPATH 项目 - 将 vendor 包纳入分析范围
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject \
  -allow-external-pkgs \
  -recursive

# 现在将扫描：
# ✅ github.com/myproject/pkg
# ✅ github.com/myproject/pkg/apis
# ✅ vendor/github.com/external/lib  (之前会被跳过)
# ✅ github.com/otherproject/pkg    (之前超出范围会被跳过)
# ❌ Go 标准库（始终跳过）
```

**使用场景：**
- 需要优化引用了 vendor 目录中类型的结构体
- GOPATH 项目，vendor 包中包含值得优化的结构体
- `-pkg-scope` 限制过严但仍希望保留包隔离

#### 3. 跳过第三方代码

```bash
# 跳过 vendor 和生成的代码
structoptimizer -package=github.com/myapp/models \
  -skip-dirs="vendor,generated_*,mocks" \
  -skip-files="*_test.go,*_pb.go" \
  -write -backup ./myproject
```

#### 4. 保持 API 兼容性

```bash
# 将特定字段保持在末尾以兼容序列化
structoptimizer -struct=github.com/myapp/models.Config \
  -reserved-fields="XXX_sizecache,XXX_unrecognized,reserved" \
  -write -backup ./myproject
```

#### 5. 跳过具有特定方法的结构体

```bash
# 跳过有 MarshalJSON 方法的结构体（可能有自定义序列化）
structoptimizer -package=github.com/myapp/models \
  -skip-by-methods="MarshalJSON,UnmarshalJSON,Encode,Decode" \
  -write -backup ./myproject
```

#### 6. GOPATH 项目支持

```bash
# 对于传统 GOPATH 项目
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/models \
  -pkg-scope=github.com/myproject \
  -write -backup
```

---

## 📊 报告示例

```markdown
╔════════════════════════════════════════════════════════════════════════════════╗
║                     StructOptimizer 优化报告                                   ║
║  版本 v1.7.6                                                                   ║
╚════════════════════════════════════════════════════════════════════════════════╝
生成时间：2026-04-18 14:41:15

┌────────────────────────────────────────────────────────────────────────────────┐
│  📊 优化总览                                                                   │
├────────────────────────────────────────────────────────────────────────────────┤
│  处理结构体总数：156                                                           │
│  ✅ 优化的结构体：43                                                           │
│  ⏭️  跳过的结构体：113                                                         │
│  💾 节省内存：     2,847 字节                                                  │
│  📈 总优化前大小： 45,678 字节                                                 │
│  📉 总优化后大小： 42,831 字节                                                 │
│  📊 总优化率：     6.2%                                                        │
└────────────────────────────────────────────────────────────────────────────────┘

✏️  调整的结构体 (43 个)
─────────────────────────────────────────────────────────────────────────────────

📦 github.com/myapp/models.User
   📁 文件：models/user.go
   📏 大小：32 字节 → 24 字节（节省：8 字节，25.0%）
   
   字段顺序对比:
   ┌────┬─────────────────────┬─────────────────────┬──────────┬──────────┐
   │ #  │ 优化前              │ 优化后              │ 大小     │ 变化     │
   ├────┼─────────────────────┼─────────────────────┼──────────┼──────────┤
   │ 1  │ Name: string        │ Balance: float64    │ 16 → 8   │ ✓        │
   │ 2  │ Age: uint8          │ ID: int64           │ 1 → 8    │ ✓        │
   │ 3  │ Active: bool        │ Name: string        │ 1 → 16   │ ✓        │
   │ 4  │ Balance: float64    │ Age: uint8          │ 8 → 1    │ ✓        │
   │ 5  │ ID: int64           │ Active: bool        │ 8 → 1    │ ✓        │
   └────┴─────────────────────┴─────────────────────┴──────────┴──────────┘
```

---

## 🏗️ 工作原理

### 两阶段优化架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  阶段 1：收集（不加载包）                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  • 使用 AST 解析源文件                                                       │
│  • 识别所有结构体及其嵌套关系                                                │
│  • 按依赖层级组织（0 = 叶子节点，N = 根节点）                                 │
│  • 跳过 vendor、标准库、第三方包                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│  阶段 2：并行优化（按需加载包）                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  • 从叶子节点向上处理（确保嵌套结构体先优化）                                 │
│  • 仅在需要时加载包                                                         │
│  • 可配置限制的并发处理                                                     │
│  • 层级间自动垃圾回收                                                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 优化算法

1. **字段分析**：提取每个字段的类型、大小和对齐要求
2. **大小计算**：计算原始结构体大小（含填充）
3. **最优排序**：按字段大小排序（从大到小）
4. **验证**：仅当新大小 < 原始大小时才应用
5. **源码重写**：用新字段顺序更新源文件

---

## 🎯 最佳实践

### 何时使用

✅ **推荐使用：**
- 大型结构体（>32 字节）且字段类型混合
- 高容量数据结构（数百万实例）
- 内存受限环境
- 长期运行的服务

⚠️ **谨慎使用：**
- 有自定义序列化的结构体（使用 `-reserved-fields`）
- 通过 FFI 或 C 绑定共享的结构体
- 字段顺序影响外部 API 的结构体

### 性能影响

| 场景 | 内存节省 | 性能变化 |
|------|---------|---------|
| 小型结构体（<16 字节） | 微小 | 可忽略 |
| 中型结构体（16-64 字节） | 10-25% | 缓存局部性改善 |
| 大型结构体（>64 字节） | 20-40% | 显著提升 |
| 深层嵌套结构体 | 累积效应 | 整体改善 |

---

## 🔧 故障排除

### 常见问题

**问题**："GOPATH 模式下必须指定 -pkg-scope 参数"

**解决方案**：指定项目的包路径前缀：
```bash
structoptimizer -prj-type=gopath -pkg-scope=github.com/myproject ...
```

---

**问题**："优化超时（1200 秒后）"

**解决方案**：为大型项目增加超时时间：
```bash
structoptimizer -timeout=3600 ...  # 1 小时
```

---

**问题**："找到多个包"

**解决方案**：确保从包含 go.mod 的项目根目录运行：
```bash
cd /path/to/project  # 包含 go.mod 的目录
structoptimizer -package=github.com/myapp/models
```

---

## 🤝 贡献

欢迎贡献！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支（`git checkout -b feature/amazing-feature`）
3. 提交更改（`git commit -m 'Add amazing feature'`）
4. 推送到分支（`git push origin feature/amazing-feature`）
5. 提交 Pull Request

### 开发环境设置

```bash
git clone https://github.com/gamelife1314/structoptimizer.git
cd structoptimizer
go test ./...  # 运行所有测试
go build ./cmd/structoptimizer
```

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢

- 灵感来源于 [golang/tools fieldalignment](https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go)
- 使用 Go 构建 ❤️

---

<div align="center">

**用 Go 构建** | [报告问题](https://github.com/gamelife1314/structoptimizer/issues) | [请求功能](https://github.com/gamelife1314/structoptimizer/issues)

</div>
