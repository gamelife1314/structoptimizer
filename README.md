# StructOptimizer

<div align="center">

[![Test](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml)
[![Release](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gamelife1314/structoptimizer)](https://goreportcard.com/report/github.com/gamelife1314/structoptimizer)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gamelife1314/structoptimizer)](go.mod)

**Optimize Go struct field alignment to reduce memory padding and save memory**

[中文文档](README.zh-CN.md) • [English](README.md)

</div>

---

## 📖 Overview

StructOptimizer is a powerful Go struct memory alignment optimization tool. It reduces memory padding by intelligently rearranging struct field order, helping you save memory in large-scale Go applications.

### Why Memory Optimization Matters

In large Go projects, poorly aligned struct fields can waste significant memory due to padding. Consider this example:

```go
// ❌ Before Optimization (32 bytes, 15 bytes wasted)
type User struct {
    Name    string  // 16 bytes
    Age     uint8   // 1 byte
    Active  bool    // 1 byte
    Balance float64 // 8 bytes
    ID      int64   // 8 bytes
    // 14 bytes padding inserted by compiler
}

// ✅ After Optimization (24 bytes, saved 8 bytes = 25%)
type User struct {
    Balance float64 // 8 bytes
    ID      int64   // 8 bytes
    Name    string  // 16 bytes
    Age     uint8   // 1 byte
    Active  bool    // 1 byte
    // 6 bytes padding only
}
```

**At scale**: If you have 1 million `User` instances, that's **8 MB saved** just from one struct!

---

## ✨ Key Features

### Core Capabilities

| Feature | Description |
|---------|-------------|
| 🔧 **Field Reordering** | Automatically rearrange struct fields for optimal alignment |
| 📦 **Nested Struct Support** | Handle deeply nested struct hierarchies (up to 50 levels) |
| 🔗 **Cross-Package Optimization** | Optimize structs referenced across multiple packages |
| 🎯 **Smart Deduplication** | Each struct optimized only once |
| 📊 **Detailed Reports** | Generate TXT/MD/HTML reports with before/after comparisons |

### Advanced Features

| Feature | Description |
|---------|-------------|
| 💾 **Auto Backup** | Create timestamped backups before modifying source files |
| ⏭️ **Flexible Skipping** | Skip directories, files, or specific structs by name/method |
| 🏗️ **Dual Project Support** | Works with both Go Modules and GOPATH+vendor projects |
| 🛡️ **Safe Optimization** | Only reorder when memory savings are guaranteed |
| 📝 **Reserved Fields** | Keep specific fields (e.g., `reserved`, `padding`) at the end |
| 🔍 **Verbose Logging** | Multiple verbosity levels (-v, -vv, -vvv) for debugging |

---

## 🚀 Quick Start

### Installation

#### Option 1: Universal Installer (Recommended)

```bash
# macOS / Linux - auto-detects package manager or falls back to direct download
curl -fsSL https://raw.githubusercontent.com/gamelife1314/structoptimizer/main/install.sh | bash

# Install a specific version
curl -fsSL https://raw.githubusercontent.com/gamelife1314/structoptimizer/main/install.sh | VERSION=v1.8.0 bash

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/gamelife1314/structoptimizer/main/install.sh | INSTALL_DIR=/usr/bin bash
```

#### Option 2: Homebrew (macOS / Linux)

```bash
brew tap gamelife1314/structoptimizer
brew install structoptimizer
```

#### Option 3: Go Install

```bash
go install github.com/gamelife1314/structoptimizer/cmd/structoptimizer@latest
```

#### Option 4: APT / YUM (Linux)

```bash
# Debian / Ubuntu (APT) - full package manager integration
echo "deb [trusted=yes] https://gamelife1314.github.io/structoptimizer/apt stable main" \
  | sudo tee /etc/apt/sources.list.d/structoptimizer.list
sudo apt update && sudo apt install structoptimizer

# RHEL / Fedora (YUM/DNF) - full package manager integration
sudo tee /etc/yum.repos.d/structoptimizer.repo <<EOF
[structoptimizer]
name=StructOptimizer
baseurl=https://gamelife1314.github.io/structoptimizer/yum
enabled=1
gpgcheck=0
EOF
sudo yum install structoptimizer     # or: sudo dnf install structoptimizer
```

#### Option 5: Manual Download

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
# Extract and add to PATH
```

#### Option 6: Build from Source

```bash
git clone https://github.com/gamelife1314/structoptimizer.git
cd structoptimizer
go build -o structoptimizer ./cmd/structoptimizer
```

### Basic Usage

**Step 1**: Analyze without modifying (generate report only)

```bash
# Optimize a specific struct
structoptimizer -struct=example.com/mypkg.User /path/to/project

# Optimize all structs in a package
structoptimizer -package=example.com/mypkg /path/to/project
```

**Step 2**: Review the generated report (`report.md`)

**Step 3**: Apply optimizations (with automatic backup)

```bash
structoptimizer -package=example.com/mypkg -write -backup /path/to/project
```

---

## 📚 Usage Guide

### Command Line Options

```
Usage: structoptimizer [options] <project_directory>

Options:
  -struct string        Struct name (format: package.path.StructName)
  -package string       Package path (mutually exclusive with -struct)
  -write                Write changes to source files
  -backup               Backup source files before modification (default: true)
  -output string        Report output path
  -format string        Report format: txt, md, html (default: md)
  -skip-dirs string     Skip directories (wildcard support, comma-separated)
  -skip-files string    Skip files (wildcard support, comma-separated)
  -skip-by-methods string  Skip structs with these methods (comma-separated)
  -skip-by-names string    Skip structs with these names (comma-separated)
  -reserved-fields string  Fields to keep at the end (comma-separated)
  -sort-same-size       Reorder fields even when size is the same
  -max-depth int        Maximum recursion depth (default: 50)
  -timeout int          Timeout in seconds (default: 1200)
  -prj-type string      Project type: gomod, gopath (default: gomod)
  -pkg-scope string     Package scope limit (required for GOPATH mode)
  -pkg-limit int        Package concurrency limit (default: 4)
  -gopath string        GOPATH path (optional, uses env if not set)
  -recursive            Recursively scan all sub-packages (-package mode only)
  -lang string          Report language: zh, en (default: zh)
  -allow-external-pkgs  Allow scanning cross-package structs (including vendor, default: false)
  -v, -vv, -vvv         Verbose output levels
  -version              Show version information
```

### Common Scenarios

#### 1. Analyze Single Struct

```bash
# Generate report without modifying source
structoptimizer -struct=github.com/myapp/models.User ./myproject

# Output: report.md
```

#### 2. Optimize Entire Package

```bash
# Optimize all structs in the models package
structoptimizer -package=github.com/myapp/models -write -backup ./myproject
```

#### 2.1. Recursive Package Scanning (NEW)

```bash
# Scan package and ALL its sub-packages recursively
structoptimizer -package=github.com/myapp/pkg -recursive -write -backup ./myproject

# Example output:
# - Scans github.com/myapp/pkg
# - Scans github.com/myapp/pkg/apis
# - Scans github.com/myapp/pkg/models
# - Scans github.com/myapp/pkg/utils
# - Automatically skips vendor and standard library
```

**How it works:**
- Uses BFS (Breadth-First Search) to traverse package dependency graph
- Starts from root package, discovers all imported sub-packages
- Automatically skips standard library and vendor packages
- Only scans sub-packages under the root package path

**Use cases:**
- Large projects with many sub-packages (50+ packages)
- Deeply nested package hierarchies (10+ levels)
- GOPATH+vendor projects
- When you want to optimize entire module at once

#### 2.2. GOPATH Project: `-pkg-scope` Parameter (IMPORTANT)

**For GOPATH mode only**, the `-pkg-scope` parameter is **REQUIRED**:

```bash
# GOPATH project - MUST specify -pkg-scope
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject \
  -recursive -write -backup
```

**What is `-pkg-scope`?**
- Limits the analysis scope to packages under the specified path prefix
- Prevents analyzing unrelated projects in your GOPATH
- Works with `-recursive` to discover all sub-packages within scope

**How to set `-pkg-scope`:**
1. Identify your project's module path (e.g., `github.com/myproject`)
2. Use the root path as scope (e.g., `-pkg-scope=github.com/myproject`)
3. All packages starting with this prefix will be included

**Example:**
```bash
# Project structure:
# $GOPATH/src/github.com/myproject/
# ├── pkg/
# │   ├── apis/
# │   ├── models/
# │   └── utils/
# └── vendor/

# Correct usage:
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject \
  -recursive

# This will scan:
# ✅ github.com/myproject/pkg
# ✅ github.com/myproject/pkg/apis
# ✅ github.com/myproject/pkg/models
# ✅ github.com/myproject/pkg/utils
# ❌ github.com/otherproject/pkg (outside scope)
# ❌ vendor/* (automatically skipped)
```

**Common mistakes:**
```bash
# ❌ Missing -pkg-scope (will fail in GOPATH mode)
structoptimizer -prj-type=gopath -package=github.com/myproject/pkg

# ❌ Too narrow scope (won't find sub-packages)
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject/pkg  # Too specific!

# ✅ Correct: use project root as scope
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject
```

**When to use:**
- Legacy GOPATH projects (pre-Go Modules)
- Projects using vendor directory
- Multiple projects in same GOPATH workspace

#### 2.3. Allow Cross-Package Scanning (`-allow-external-pkgs`) (NEW)

By default, StructOptimizer skips structs outside the `-pkg-scope` range and vendor packages. Use `-allow-external-pkgs` to include them:

```bash
# GOPATH project - include vendor packages in analysis
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/pkg \
  -pkg-scope=github.com/myproject \
  -allow-external-pkgs \
  -recursive

# This will now scan:
# ✅ github.com/myproject/pkg
# ✅ github.com/myproject/pkg/apis
# ✅ vendor/github.com/external/lib  (previously skipped)
# ✅ github.com/otherproject/pkg    (previously skipped, if outside scope)
# ❌ Go standard library (always skipped)
```

**Use cases:**
- Need to optimize structs that reference types in vendor directories
- GOPATH projects where vendor packages contain structs worth optimizing
- When `-pkg-scope` is too restrictive but you still want package isolation

#### 3. Skip Third-Party Code

```bash
# Skip vendor and generated code
structoptimizer -package=github.com/myapp/models \
  -skip-dirs="vendor,generated_*,mocks" \
  -skip-files="*_test.go,*_pb.go" \
  -write -backup ./myproject
```

#### 4. Preserve API Compatibility

```bash
# Keep certain fields at the end for serialization compatibility
structoptimizer -struct=github.com/myapp/models.Config \
  -reserved-fields="XXX_sizecache,XXX_unrecognized,reserved" \
  -write -backup ./myproject
```

#### 5. Skip Structs with Specific Methods

```bash
# Skip structs that have MarshalJSON method (may have custom serialization)
structoptimizer -package=github.com/myapp/models \
  -skip-by-methods="MarshalJSON,UnmarshalJSON,Encode,Decode" \
  -write -backup ./myproject
```

#### 6. GOPATH Project Support

```bash
# For legacy GOPATH projects
structoptimizer -prj-type=gopath \
  -package=github.com/myproject/models \
  -pkg-scope=github.com/myproject \
  -write -backup
```

---

## 📊 Report Example

```markdown
╔════════════════════════════════════════════════════════════════════════════════╗
║                     StructOptimizer Optimization Report                        ║
║  Version v1.7.6                                                                ║
╚════════════════════════════════════════════════════════════════════════════════╝
Generated: 2026-04-18 14:41:15

┌────────────────────────────────────────────────────────────────────────────────┐
│  📊 Summary                                                                    │
├────────────────────────────────────────────────────────────────────────────────┤
│  Total Structs Processed:  156                                                 │
│  ✅ Optimized:              43                                                 │
│  ⏭️  Skipped:               113                                                │
│  💾 Memory Saved:           2,847 bytes                                        │
│  📈 Total Size Before:      45,678 bytes                                       │
│  📉 Total Size After:       42,831 bytes                                       │
│  📊 Overall Optimization:   6.2%                                               │
└────────────────────────────────────────────────────────────────────────────────┘

✏️  Optimized Structs (43)
─────────────────────────────────────────────────────────────────────────────────

📦 github.com/myapp/models.User
   📁 File: models/user.go
   📏 Size: 32 bytes → 24 bytes (saved: 8 bytes, 25.0%)
   
   Field Order Comparison:
   ┌────┬─────────────────────┬─────────────────────┬──────────┬──────────┐
   │ #  │ Before              │ After               │ Size     │ Changed  │
   ├────┼─────────────────────┼─────────────────────┼──────────┼──────────┤
   │ 1  │ Name: string        │ Balance: float64    │ 16 → 8   │ ✓        │
   │ 2  │ Age: uint8          │ ID: int64           │ 1 → 8    │ ✓        │
   │ 3  │ Active: bool        │ Name: string        │ 1 → 16   │ ✓        │
   │ 4  │ Balance: float64    │ Age: uint8          │ 8 → 1    │ ✓        │
   │ 5  │ ID: int64           │ Active: bool        │ 8 → 1    │ ✓        │
   └────┴─────────────────────┴─────────────────────┴──────────┴──────────┘
```

---

## 🏗️ How It Works

### Two-Phase Optimization Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Phase 1: Collection (No Package Loading)                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Parse source files using AST                                             │
│  • Identify all structs and nested relationships                            │
│  • Organize by dependency level (0 = leaf, N = root)                        │
│  • Skip vendor, standard library, third-party packages                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│  Phase 2: Parallel Optimization (On-Demand Package Loading)                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Process from leaf level upward (ensures nested structs optimized first)  │
│  • Load packages only when needed                                           │
│  • Concurrent processing with configurable limits                           │
│  • Automatic garbage collection between levels                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Optimization Algorithm

1. **Field Analysis**: Extract type, size, and alignment for each field
2. **Size Calculation**: Compute original struct size with padding
3. **Optimal Ordering**: Sort fields by size (largest to smallest)
4. **Validation**: Only apply if new size < original size
5. **Source Rewrite**: Update source file with new field order

---

## 🎯 Best Practices

### When to Use

✅ **Recommended:**
- Large structs (>32 bytes) with mixed field types
- High-volume data structures (millions of instances)
- Memory-constrained environments
- Long-running services

⚠️ **Use with Caution:**
- Structs with custom serialization (use `-reserved-fields`)
- Structs shared via FFI or C bindings
- Structs where field order affects external APIs

### Performance Impact

| Scenario | Memory Savings | Performance Change |
|----------|---------------|-------------------|
| Small structs (<16 bytes) | Minimal | Negligible |
| Medium structs (16-64 bytes) | 10-25% | Improved cache locality |
| Large structs (>64 bytes) | 20-40% | Significant improvement |
| Deeply nested structs | Cumulative | Better overall |

---

## 🔧 Troubleshooting

### Common Issues

**Issue**: "GOPATH mode requires -pkg-scope parameter"

**Solution**: Specify your project's package prefix:
```bash
structoptimizer -prj-type=gopath -pkg-scope=github.com/myproject ...
```

---

**Issue**: "Optimization timeout after 1200 seconds"

**Solution**: Increase timeout for large projects:
```bash
structoptimizer -timeout=3600 ...  # 1 hour
```

---

**Issue**: "Multiple packages found"

**Solution**: Ensure you're running from the project root with go.mod:
```bash
cd /path/to/project  # Directory containing go.mod
structoptimizer -package=github.com/myapp/models
```

---

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
git clone https://github.com/gamelife1314/structoptimizer.git
cd structoptimizer
go test ./...  # Run all tests
go build ./cmd/structoptimizer
```

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Inspired by [golang/tools fieldalignment](https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go)
- Built with ❤️ using Go

---

<div align="center">

**Made with Go** | [Report Bug](https://github.com/gamelife1314/structoptimizer/issues) | [Request Feature](https://github.com/gamelife1314/structoptimizer/issues)

</div>
