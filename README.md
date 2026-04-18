# StructOptimizer

[![Test](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/test.yml)
[![Release](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml/badge.svg)](https://github.com/gamelife1314/structoptimizer/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gamelife1314/structoptimizer)](https://goreportcard.com/report/github.com/gamelife1314/structoptimizer)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**English** | [中文文档](README.zh-CN.md)

Go struct alignment optimizer - reduces memory padding by rearranging struct field order.

## Background

In large Go projects, developers may not fully consider struct field alignment, wasting significant memory. With expensive memory costs, this optimization becomes crucial.

Reference: [golang/tools fieldalignment](https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go)

The official tool is too simple, unable to handle:
- Nested struct optimization
- Cross-package referenced structs
- Deep-first multi-level nested optimization

This tool aims to solve these issues.

## Features

### Core Features

- ✅ Optimize Go struct field alignment
- ✅ Support named and embedded fields
- ✅ Cross-package struct optimization
- ✅ Deep-first nested struct optimization (multi-level)
- ✅ Optimize each struct only once (deduplication)

### Advanced Features

- ✅ Source file backup (`-backup`)
- ✅ Limit targets by directory and struct name
- ✅ Skip directories or files (wildcard matching)
- ✅ Skip structs by method name (`-skip-by-methods`)
- ✅ Generate optimization reports (TXT/MD/HTML)
- ✅ Verbose logging (`-v`, `-vv`, `-vvv`)
- ✅ In-place source modification (`-write`)
- ✅ Sort by field size when optimization savings are equal (`-sort-same-size`)
- ✅ Analyze all structs in a specified package (`-package`)
- ✅ Support go.mod and GOPATH+vendor projects
- ✅ Auto-skip vendor third-party structs
- ✅ Auto-skip Go standard library structs
- ✅ Smart project package detection

## Project Support

### Go Modules (Recommended)

```bash
# Specify project root (contains go.mod)
./structoptimizer -struct=example.com/myapp/pkg.Context /path/to/project

# Or execute in project directory (directory parameter can be omitted)
cd /path/to/project
./structoptimizer -struct=example.com/myapp/pkg.Context
```

### GOPATH + Vendor Projects

For legacy projects using GOPATH + vendor, use `-prj-type=gopath`:

```bash
# Use -prj-type=gopath to specify project type
# -pkg-scope is required to limit analysis scope to your project
./structoptimizer -prj-type=gopath -package example.com/myproject/pkg -pkg-scope example.com/myproject

# Optional: specify GOPATH path (otherwise uses environment variable)
./structoptimizer -prj-type=gopath -gopath=/path/to/gopath -struct=example.com/myproject/pkg.MyStruct -pkg-scope example.com/myproject
```

**Note**:
- GOPATH projects **do not need** to specify project directory
- **`-pkg-scope` is required** to limit analysis scope and prevent analyzing other projects in GOPATH
- `-pkg-scope` should be your project's package path prefix, e.g., `example.com/myproject`
- Third-party structs in vendor **will not be optimized**
- Fields referencing vendor structs retain original order
- Attempting to optimize vendor structs directly will be skipped with reason

## Installation

### Option 1: Download Pre-compiled Binaries (Recommended)

Download pre-compiled binaries from [GitHub Releases](https://github.com/gamelife1314/structoptimizer/releases):

| Platform | Architecture | Filename |
|----------|-------------|----------|
| Linux | amd64 | `structoptimizer-linux-amd64.tar.gz` |
| Linux | arm64 | `structoptimizer-linux-arm64.tar.gz` |
| macOS | amd64 | `structoptimizer-darwin-amd64.tar.gz` |
| macOS | arm64 (Apple Silicon) | `structoptimizer-darwin-arm64.tar.gz` |
| Windows | amd64 | `structoptimizer-windows-amd64.zip` |

Extract and add to PATH:

```bash
# Linux/macOS
tar -xzf structoptimizer-*.tar.gz
sudo mv structoptimizer-* /usr/local/bin/structoptimizer

# Windows
# Extract and add to PATH
```

### Option 2: Install via go install

```bash
go install github.com/gamelife1314/structoptimizer/cmd/structoptimizer@latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/gamelife1314/structoptimizer.git
cd structoptimizer
go build -o structoptimizer ./cmd/structoptimizer
```

## Quick Start

### Basic Usage

Optimize single struct (Go Module):

```bash
# Specify project directory
./structoptimizer -struct=pkg.Context /path/to/project
```

### Common Commands

```bash
# 1. Optimize single struct (generate report, no source modification)
./structoptimizer -struct=pkg.Context /path/to/project

# 2. Optimize and write source (backup enabled by default)
./structoptimizer -struct=pkg.Context -write -backup /path/to/project

# 3. Optimize and backup source
./structoptimizer -struct=pkg.Context -write -backup /path/to/project

# 4. Optimize and write source without backup
./structoptimizer -struct=pkg.Context -write -backup=false /path/to/project

# 5. Optimize all structs in specified package
./structoptimizer -package pkg /path/to/project

# 6. Optimize package and write source
./structoptimizer -package pkg -write -backup /path/to/project

# 7. Skip certain directories and files
./structoptimizer -struct=pkg.Context \
    -skip-dir alpha \
    -skip-dir generated_* \
    -skip-file *_test.go \
    -skip-file *_pb.go \
    /path/to/project

# 7.1 Skip vendor directory (loose match)
./structoptimizer -package pkg -skip-dir vendor /path/to/project

# 8. Skip structs with specific methods
./structoptimizer -struct=pkg.Context \
    -skip-by-methods "Encode_By_KKK,Encode_By_KKK1" \
    /path/to/project

# 9. Skip structs by name
./structoptimizer -package pkg \
    -skip-by-names "BadStruct,UnusedStruct" \
    /path/to/project

# 10. Generate report to file
./structoptimizer -struct=pkg.Context \
    -output report.md \
    /path/to/project

# 11. Show verbose execution
./structoptimizer -struct=pkg.Context -vvv /path/to/project

# 12. Sort by field size when sizes are equal
./structoptimizer -struct=pkg.Context -sort-same-size /path/to/project

# 13. GOPATH project
./structoptimizer -prj-type=gopath -struct=example.com/pkg.MyStruct

# 14. GOPATH project with GOPATH path
./structoptimizer -prj-type=gopath -gopath=/path/to/gopath -struct=example.com/pkg.MyStruct
```

### In-place Modification and Backup

Use `-write` to write optimization results to source files. `-backup` (enabled by default) creates backups before modification.

```bash
# Optimize and write source with backup
./structoptimizer -package pkg -write -backup /path/to/project

# Backup file example:
#   Original: pkg/context.go
#   Backup:   pkg/context.go.bak

# Optimize and write source without backup
./structoptimizer -package pkg -write -backup=false /path/to/project
```

**Note**:
- Backup files are named `original.go.bak`
- Keeping backup (default) is recommended for easy recovery
- Optimized code is formatted with `go/printer` for consistent style

## Command Line Flags

### -skip-dirs Flag

`-skip-dirs` skips directories with struct support for **dual matching**.

#### Matching Rules

`-skip-dirs` uses dual matching:

1. **basename matching**: Match directory basename (last component)
2. **path inclusion matching**: Match if full path contains directory name as complete path component

**Logic**:
```go
func shouldSkipDir(dirPath string) bool {
    baseName := filepath.Base(dirPath)
    normalizedPath := filepath.ToSlash(dirPath)

    for _, pattern := range SkipDirs {
        // 1. basename matching
        if matched, _ := filepath.Match(pattern, baseName); matched {
            return true
        }
        // 2. path inclusion (complete path component)
        if strings.Contains(normalizedPath, "/"+pattern+"/") ||
           strings.Contains(normalizedPath, "/"+pattern) ||
           strings.HasSuffix(normalizedPath, "/"+pattern) {
            return true
        }
    }
    return false
}
```

#### Examples

```bash
# Skip all vendor directories
./structoptimizer -package writer/config -skip-dirs vendor ./

# Following paths will be skipped:
# ✓ /project/vendor/lib.go                  # basename match
# ✓ /project/pkg/vendor/lib.go              # basename match
# ✓ /a/b/c/vendor/github.com/lib/lib.go     # path inclusion match

# With wildcards
./structoptimizer -package writer/config -skip-dirs "generated_*" ./

# Skip multiple directories (comma separated)
./structoptimizer -package writer/config -skip-dirs "vendor,generated_*,datas" ./
```

### Complete Flags Table

| Flag | Description | Default |
|------|-------------|---------|
| `<project_dir>` | Go Module project root (GOPATH can omit) | - |
| `-struct` | Struct name (format: pkg_path.StructName) | - |
| `-package` | Package path (mutually exclusive with `-struct`) | - |
| `-source-file` | Source file path (limit struct search to file) | - |
| `-write` | Write to source file | false |
| `-backup` | Backup source before modification | true |
| `-skip-dir` | Skip directories (wildcard, match any path component) | - |
| `-skip-file` | Skip files (wildcard) | - |
| `-skip` | Skip file pattern | - |
| `-skip-by-methods` | Skip structs with these methods (wildcard) | - |
| `-skip-by-names` | Skip structs by name (wildcard) | - |
| `-output` | Report output path | stdout |
| `-v`, `-vv`, `-vvv` | Verbosity level | 0 |
| `-sort-same-size` | Sort by field size when equal | false |
| `-prj-type` | Project type (gomod/gopath) | gomod |
| `-gopath` | GOPATH path (GOPATH project optional) | GOPATH env var |
| `-help` | Show help | - |

## Example

### Before Optimization

```go
type BadStruct struct {
    A bool   // 1 byte
    // [7 bytes padding]
    B int64  // 8 bytes
    C int32  // 4 bytes
    D bool   // 1 byte
    // [3 bytes padding]
    E int32  // 4 bytes
    // [4 bytes tail padding]
}
// Total: 32 bytes
```

### After Optimization

```go
type GoodStruct struct {
    B int64  // 8 bytes (offset 0)
    C int32  // 4 bytes (offset 8)
    E int32  // 4 bytes (offset 12)
    A bool   // 1 byte (offset 16)
    D bool   // 1 byte (offset 17)
    // [6 bytes tail padding]
}
// Total: 24 bytes (saved 8 bytes)
```

### Nested Struct Optimization

```go
// Main struct: project/testdata.NestedOuter
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

// Structs in same package
type Inner struct {
    Y int64
    X int32
    Z int32
}

// Cross-package struct (project/testdata/subpkg1.SubPkg1)
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

Tool deeply optimizes all nested structs.

## Output Report

### Markdown Format Example

```markdown
# StructOptimizer Report

## Summary
- Total structs: 5
- Optimized: 3
- Skipped: 2
- Memory saved: 128 bytes

## Optimization Details

### writer/config.Context
- File: writer/config/context.go
- Before: 64 bytes
- After: 48 bytes
- Saved: 16 bytes

**Before:**
1. A (bool, 1 byte)
2. B (int64, 8 bytes)
3. C (int32, 4 bytes)
4. D (bool, 1 byte)
5. E (int32, 4 bytes)

**After:**
1. B (int64, 8 bytes)
2. C (int32, 4 bytes)
3. E (int32, 4 bytes)
4. A (bool, 1 byte)
5. D (bool, 1 byte)
```

## Project Structure

```
structoptimizer/
├── cmd/
│   └── structoptimizer/
│       └── main.go          # Entry point
├── analyzer/
│   └── analyzer.go          # Package and type analysis
├── optimizer/
│   ├── optimizer.go         # Core optimization logic
│   ├── field.go             # Field analysis
│   └── size.go              # Size calculation
├── reporter/
│   └── reporter.go          # Report generation
├── writer/
│   └── writer.go            # Source writing
├── internal/
│   └── utils/
│       └── utils.go         # Utility functions
├── testdata/                 # Test data
├── VERIFICATION_CHECKLIST.md # Verification checklist
├── design.md                # Design document
└── README.md                # Usage guide
```

## Technical Details

### Go Struct Memory Alignment Rules

1. Each field aligns to its type size (`int64` requires 8-byte alignment)
2. Struct total size must be multiple of largest field alignment
3. Poor field order causes significant padding

### Optimization Strategy

1. **Field reordering**: Sort by size descending
2. **Deep-first**: Recursively optimize nested structs
3. **Deduplication**: Each struct optimized only once

## Edge Cases

- Generic structs: Skipped
- External package structs: Skipped (cross-library)
- Circular references: Detected and avoided
- Fields with tags: Tags preserved
- Empty structs: Skipped
- Single-field structs: Skipped

## CI/CD

This project uses GitHub Actions for:
- **Test**: Run on push to main/master and PRs
- **Release**: Auto-build multi-platform binaries on tag push

See `.github/workflows/` for configuration.

## License

MIT License

## Contributing

Issues and Pull Requests are welcome!
