# StructOptimizer Design Document

## 1. Overview

StructOptimizer is a static analysis tool for optimizing Go struct field alignment. By reordering struct fields, it reduces memory padding and lowers memory consumption.

### 1.1 Problem Statement

In large Go projects, developers may not fully consider struct field alignment, leading to significant memory waste:

```go
// Before: 32 bytes
type BadStruct struct {
    A bool   // 1 byte + 7 bytes padding
    B int64  // 8 bytes
    C int32  // 4 bytes
    D bool   // 1 byte + 3 bytes padding
    E int32  // 4 bytes
    // 4 bytes trailing padding
}

// After: 24 bytes (25% savings)
type GoodStruct struct {
    B int64  // 8 bytes
    C int32  // 4 bytes
    E int32  // 4 bytes
    A bool   // 1 byte
    D bool   // 1 byte
    // 6 bytes trailing padding
}
```

### 1.2 Solution

- Automatically analyze struct field layout
- Intelligently reorder fields to minimize padding
- Support nested struct optimization (depth-first)
- Support cross-package reference optimization
- Support both Go Modules and GOPATH projects
- Generate detailed optimization reports (Markdown, TXT, HTML) with bilingual support (Chinese/English)
- Optionally rewrite source files with optimized field order

## 2. Core Principles

### 2.1 Go Struct Memory Alignment Rules

1. **Field Alignment**: Each field is aligned according to its type size:
   - `bool`, `int8`: 1-byte alignment
   - `int16`: 2-byte alignment
   - `int32`, `float32`: 4-byte alignment
   - `int64`, `float64`: 8-byte alignment

2. **Struct Alignment**: The total size of a struct must be a multiple of its largest field's alignment requirement.

3. **Padding Calculation**:
   ```
   offset = (current_offset + alignment - 1) / alignment * alignment
   total_size = (total_size + max_alignment - 1) / max_alignment * max_alignment
   ```

### 2.2 Optimization Strategy

| Strategy | Description | Implementation |
|----------|-------------|----------------|
| Field Reordering | Sort fields from largest to smallest | `ReorderFields()` |
| Depth-First | Recursively optimize nested structs | `optimizeStruct()` |
| Deduplication | Optimize each struct only once | `optimized` map |
| Skip Rules | Multiple skip conditions supported | `shouldSkip()` |
| Reserved Fields | Designated fields always placed last | `ReservedFields` config |

### 2.3 Skip Rules

Structs are skipped under these conditions:
- Empty structs (no fields)
- Single-field structs (no optimization possible)
- Structs with specified methods (`-skip-by-methods`)
- Structs with specified names (`-skip-by-names`)
- Files matching skip patterns (`-skip-files`, `-skip-dirs`)
- External package structs that cannot be resolved

## 3. System Architecture

### 3.1 Module Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  main.go (CLI Entry)                                                 │
│  - Flag parsing (flag)                                               │
│  - Module coordination                                               │
└─────────────────────────────────────────────────────────────────────┘
         │
         ├──────────────────┬──────────────────┬──────────────────┐
         │                  │                  │                  │
         ▼                  ▼                  ▼                  ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────┐
│   analyzer      │ │   optimizer     │ │   reporter      │ │   writer    │
│   Analyzer      │ │   Optimizer     │ │   Reporter      │ │   Writer    │
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
│   utils         │ │   • field.go    │         │   Test data     │
│   Utilities     │ │   • size.go     │         │                 │
│                 │ │   • optimizer   │         │ • basic/        │
│ • MatchPattern  │ │                 │         │ • nested/       │
│ • FormatSize    │ │                 │         │ • crosspkg/     │
│ • ShouldSkip    │ │                 │         │ • complexpkg/   │
└─────────────────┘ └─────────────────┘         └─────────────────┘
```

### 3.2 Data Flow

```
                    ┌──────────────┐
                    │  User Input   │
                    │  CLI Flags    │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   CLI Parse   │
                    │  (main.go)    │
                    └──────┬───────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌────────────┐ ┌────────────┐ ┌────────────┐
    │  Analyzer  │ │ Optimizer  │ │  Reporter  │
    │  Load Pkg  │ │ Optimize   │ │ Generate   │
    └─────┬──────┘ └─────┬──────┘ └─────┬──────┘
          │              │               │
          ▼              ▼               ▼
    ┌────────────┐ ┌────────────┐ ┌────────────┐
    │ Pkg Info   │ │ Results    │ │ MD/TXT/   │
    │ AST        │ │ Field Order│ │ HTML       │
    │ Types      │ │ Mem Savings│ │ Report     │
    └────────────┘ └────────────┘ └────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │    Writer    │
                    │  Write Files │
                    │  Backup      │
                    └──────────────┘
```

### 3.3 Optimization Flow

```
    ┌─────────┐
    │  Start  │
    └────┬────┘
         │
         ▼
    ┌─────────────┐
    │ Load Pkg    │
    └─────┬───────┘
          │
          ▼
    ┌─────────────┐     ┌──────────┐
    │ Find Struct  │────▶│Already   │───┐
    └─────┬───────┘     │Optimized?│   │
          │             └────┬─────┘   │
          │                  │No       │
          │                  ▼         │
          │            ┌──────────┐    │
          │            │Should    │────┤
          │            │Skip?     │    │
          │            └────┬─────┘    │
          │                 │No        │
          │                 ▼          │
          │           ┌────────────┐   │
          │           │Optimize    │   │
          │           │Nested      │   │
          │           │Structs     │   │
          │           └─────┬──────┘   │
          │                 │          │
          │                 ▼          │
          │           ┌────────────┐   │
          │           │Reorder     │   │
          │           │Fields      │   │
          │           └─────┬──────┘   │
          │                 │          │
          │                 ▼          │
          └────────────▶┌────────┐◀────┘
                        │Record  │
                        │Results │
                        └───┬────┘
                            │
                            ▼
                      ┌─────────┐
                      │  End    │
                      └─────────┘
```

## 4. Module Design

### 4.1 CLI Module (`cmd/structoptimizer/main.go`)

```go
type Config struct {
    Struct         string        // Struct name (pkg.StructName format)
    Package        string        // Package path (mutually exclusive with -struct)
    SourceFile     string        // Source file path
    Write          bool          // Write optimized fields to source files
    Backup         bool          // Backup source files before modification
    SkipDirs       string        // Directories to skip (comma-separated, glob patterns)
    SkipFiles      string        // Files to skip (comma-separated, glob patterns)
    SkipByMethods  string        // Skip structs with these methods (comma-separated)
    SkipByNames    string        // Skip structs with these names (comma-separated)
    Output         string        // Report output path
    Verbose        int           // Verbosity level (0-3)
    SortSameSize   bool          // Reorder fields of same size
    ReportFormat   string        // Report format: md, txt, html
    ProjectType    string        // Project type: gomod or gopath
    GOPATH         string        // GOPATH path (optional, for GOPATH projects)
    TargetDir      string        // Target directory (positional argument)
    MaxDepth       int           // Maximum recursion depth
    Timeout        int           // Timeout in seconds
    PkgScope       string        // Package scope limit (required for GOPATH mode)
    PkgWorkerLimit int           // Package concurrency limit
    ShowVersion    bool          // Show version
    ReservedFields string        // Reserved field names (comma-separated, placed last)
    Recursive      bool          // Recursive scan of sub-packages (-package mode)
    Lang           reporter.Lang // Report language: zh (default) or en
}
```

**Responsibilities**:
- Parse command-line arguments
- Coordinate module workflows
- Error handling and logging

### 4.2 Analyzer Module (`analyzer/analyzer.go`)

```go
type Analyzer struct {
    config     *Config
    fset       *token.FileSet
    info       *types.Info
    pkg        *packages.Package
    pkgMap     map[string]*packages.Package
}
```

**Core Methods**:
- `LoadPackage()`: Load package and its dependencies
- `FindStructByName()`: Find a specific struct by name
- `FindAllStructs()`: Find all structs in a package
- `HasMethod()`: Check if a struct has a specified method

**Dependencies**:
- `golang.org/x/tools/go/packages`
- `go/ast`, `go/types`, `go/token`

### 4.3 Optimizer Module (`optimizer/`)

#### 4.3.1 Field Info (`field.go`)

```go
type FieldInfo struct {
    Name     string      // Field name
    Type     types.Type  // Field type
    Size     int64       // Field size in bytes
    Align    int64       // Alignment requirement
    IsEmbed  bool        // Whether it's an embedded field
    PkgPath  string      // Type package path
    TypeName string      // Type name
}
```

#### 4.3.2 Size Calculation (`size.go`)

```go
func CalcStructSize(st *types.Struct) int64
func CalcFieldSize(typ types.Type) (size, align int64)
func CalcOptimizedSize(fields []FieldInfo) int64
```

Supports all Go types: basic types, pointers, structs, slices, maps, channels, functions, interfaces, arrays, and type aliases.

#### 4.3.3 Core Optimization (`optimizer.go`)

```go
type Optimizer struct {
    config      *Config
    analyzer    *analyzer.Analyzer
    optimized   map[string]*StructInfo
    report      *Report
    methodIndex *MethodIndex
    // ... caching and concurrency fields
}

func (o *Optimizer) Optimize() (*Report, error)
func (o *Optimizer) optimizeStruct(pkgPath, structName string, depth int)
```

**Features**:
- Depth-first recursive optimization of nested structs
- Cross-package struct resolution
- Parallel processing with configurable worker limits
- Package and struct caching for performance

#### 4.3.4 Reordering (`reorder.go`)

```go
func ReorderFields(fields []FieldInfo, sortSameSize bool, reserved []string) []FieldInfo
```

**Algorithm**:
1. Separate embedded fields and named fields
2. Sort named fields by size (descending), then by alignment (if `sortSameSize`)
3. Place reserved fields at the end
4. Merge: embedded fields + sorted named fields + reserved fields

### 4.4 Reporter Module (`reporter/`)

#### 4.4.1 Report Types

```go
type ReportLevel int

const (
    ReportLevelSummary ReportLevel = iota  // Summary only
    ReportLevelChanged                     // Summary + changed structs
    ReportLevelFull                        // All structs
)

type Report struct {
    TotalStructs      int
    OptimizedCount    int
    SkippedCount      int
    TotalSaved        int64
    StructReports     []*StructReport
    RootStruct        string  // Root struct name (-struct mode)
    RootStructSize    int64   // Root struct size before optimization
    RootStructOptSize int64   // Root struct size after optimization
    TotalOrigSize     int64   // Total size before optimization
    TotalOptSize      int64   // Total size after optimization
}

type StructReport struct {
    Name       string
    PkgPath    string
    File       string
    OrigSize   int64
    OptSize    int64
    Saved      int64
    OrigFields []string
    OptFields  []string
    FieldTypes map[string]string  // field name -> type name
    FieldSizes map[string]int64   // field name -> size (bytes)
    Skipped    bool
    SkipReason string
    Depth      int
    HasEmbed   bool
}
```

#### 4.4.2 Internationalization (i18n)

The reporter supports bilingual output through a language system:

```go
type Lang string

const (
    LangZH Lang = "zh"  // Chinese (default)
    LangEN Lang = "en"  // English
)
```

All UI strings are defined in `reporter_i18n.go` with separate structs for each language. The `getStrings()` function returns the appropriate language bundle based on the configured `Lang`.

#### 4.4.3 Supported Formats

| Format | File | Description |
|--------|------|-------------|
| Markdown (default) | `reporter_md.go` | Rich markdown with tables and code blocks |
| TXT | `reporter.go` | Plain text with box-drawing characters |
| HTML | `reporter_html.go` | Styled HTML with responsive tables |

### 4.5 Writer Module (`writer/writer.go`)

```go
type SourceWriter struct {
    config *Config
    fset   *token.FileSet
}

func (w *SourceWriter) BackupFile(filePath string) (string, error)
func (w *SourceWriter) RewriteFile(filePath string, optimized map[string]*StructInfo) error
```

**Features**:
- Creates `.bak` backups before modifying files
- Preserves comments, build tags, and formatting
- Uses `go/printer` for proper Go code formatting

## 5. Algorithm Design

### 5.1 Field Size Calculation

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
        return sizeof(sliceHeader), alignof(sliceHeader)
    case *types.Map:
        return sizeof(mapHeader), alignof(mapHeader)
    case *types.Array:
        return t.Len() * elemSize, elemAlign
    // ... other types
    }
}
```

### 5.2 Struct Size Calculation

```go
func CalcStructSize(st *types.Struct) int64 {
    var offset int64 = 0
    var maxAlign int64 = 1

    for i := 0; i < st.NumFields(); i++ {
        field := st.Field(i)
        size, align := CalcFieldSize(field.Type())

        // Align offset
        if offset % align != 0 {
            offset += align - (offset % align)
        }

        offset += size
        if align > maxAlign {
            maxAlign = align
        }
    }

    // Trailing padding
    if offset % maxAlign != 0 {
        offset += maxAlign - (offset % maxAlign)
    }

    return offset
}
```

### 5.3 Field Reordering

```go
func ReorderFields(fields []FieldInfo, sortSameSize bool, reserved []string) []FieldInfo {
    // 1. Separate embedded, reserved, and named fields
    var embeds, reserved, named []FieldInfo

    // 2. Sort named fields by size (descending), then alignment
    sort.Slice(named, func(i, j int) bool {
        if named[i].Size != named[j].Size {
            return named[i].Size > named[j].Size
        }
        if sortSameSize {
            return named[i].Align > named[j].Align
        }
        return false
    })

    // 3. Merge: embedded + sorted named + reserved
    return append(append(embeds, named...), reserved...)
}
```

### 5.4 Depth-First Optimization

```go
func optimizeStruct(pkgPath, structName string, depth int) {
    // 1. Check if already optimized
    if _, ok := optimized[key]; ok {
        return
    }

    // 2. Check skip conditions
    if shouldSkip() {
        return
    }

    // 3. Recursively optimize nested structs (depth-first)
    for _, field := range fields {
        if isStructType(field.Type) {
            optimizeStruct(field.PkgPath, field.TypeName, depth+1)
        }
    }

    // 4. Reorder current struct fields
    ReorderFields()
}
```

## 6. Interface Design

### 6.1 Command-Line Interface

```bash
# Basic usage
./structoptimizer [flags] [directory]

# Optimize a single struct
./structoptimizer -struct=pkg.Context ./

# Optimize an entire package
./structoptimizer --package pkg/path ./

# Optimize and write changes
./structoptimizer -struct=pkg.Context --write --backup ./

# Skip specific files
./structoptimizer --package pkg/path \
    -skip "*.pb.go" \
    -skip "*_test.go" \
    ./

# Generate English report in HTML format
./structoptimizer -struct=pkg.Context --format html --lang en -o report.html ./

# GOPATH project with recursive scan
./structoptimizer -prj-type=gopath -struct=example.com/pkg.MyStruct \
    -pkg-scope=example.com/pkg -recursive
```

### 6.2 Key Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-struct` | Struct name (`pkg.StructName` format) | - |
| `-package` | Package path (mutually exclusive with `-struct`) | - |
| `-write` | Write optimized fields to source files | `false` |
| `-backup` | Backup source files before modification | `true` |
| `-format` | Report format (`md`/`txt`/`html`) | `md` |
| `-lang` | Report language (`zh`/`en`) | `zh` |
| `-output` | Report output file path | stdout |
| `-skip-dirs` | Directories to skip (glob, comma-separated) | - |
| `-skip-files` | Files to skip (glob, comma-separated) | - |
| `-skip-by-methods` | Skip structs with these methods | - |
| `-skip-by-names` | Skip structs with these names | - |
| `-prj-type` | Project type (`gomod`/`gopath`) | `gomod` |
| `-gopath` | GOPATH path (for GOPATH projects) | - |
| `-pkg-scope` | Package scope limit (GOPATH mode required) | - |
| `-max-depth` | Maximum recursion depth | `50` |
| `-timeout` | Timeout in seconds | `1200` |
| `-pkg-limit` | Package concurrency limit | `4` |
| `-reserved-fields` | Reserved fields placed last (comma-separated) | - |
| `-recursive` | Recursive sub-package scan (`-package` mode) | `false` |
| `-sort-same-size` | Reorder same-size fields | `false` |
| `-v`, `-vv`, `-vvv` | Verbosity levels | - |
| `-version` | Show version | - |

### 6.3 Reporter Interface

```go
type Reporter struct {
    format string      // txt, md, html
    output string      // Output path
    level  ReportLevel // Detail level
    lang   Lang        // zh (default) or en
}

func NewReporter(format, output string, level ReportLevel) *Reporter
func NewReporterWithLang(format, output string, level ReportLevel, lang Lang) *Reporter
func (r *Reporter) Generate(report *optimizer.Report) error
func (r *Reporter) GenerateMD(report *optimizer.Report) (string, error)
func (r *Reporter) GenerateTXT(report *optimizer.Report) (string, error)
func (r *Reporter) GenerateHTML(report *optimizer.Report) (string, error)
```

## 7. Test Design

### 7.1 Test Strategy

| Module | Focus | Coverage Target |
|--------|-------|-----------------|
| utils | Utility function edge cases | >90% |
| optimizer | Size calculation, field reordering | >80% |
| reporter | Report format correctness, i18n | >80% |
| writer | File operation correctness | >70% |
| analyzer | Package loading, struct lookup | >60% |

### 7.2 Test Data Structure

```
testdata/
├── basic/              # Basic test cases
│   └── basic.go        # Simple structs
├── nested/             # Nested test cases
│   ├── nested.go       # 2-3 level nesting
│   └── deep_nested.go  # 5+ level nesting
├── crosspkg/           # Cross-package test cases
│   ├── subpkg1/        # Sub-package 1
│   ├── subpkg2/        # Sub-package 2
│   └── crosspkg.go     # Cross-package references
├── complexpkg/         # Complex test cases
│   └── complex.go      # slice/map/pointer types
├── methodskip/         # Method skip tests
│   └── methodskip.go   # Structs with methods
├── recursive_scan_test/ # Recursive scanning tests
│   └── pkg/            # Multi-package test structure
└── gopath_test_project/ # GOPATH mode tests
    └── ...             # GOPATH-specific scenarios
```

### 7.3 Test Categories

1. **Unit Tests**: Test individual functions/methods
2. **Integration Tests**: Test inter-module cooperation
3. **End-to-End Tests**: Test complete optimization pipeline
4. **Regression Tests**: Protect against known bugs (embedded fields, type aliases, external packages, etc.)

## 8. Performance Considerations

### 8.1 Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| Package loading | O(n) | n = number of files |
| Struct lookup | O(n*m) | n = files, m = declarations |
| Field reordering | O(k log k) | k = number of fields |
| Nested optimization | O(d*k) | d = depth, k = struct count |

### 8.2 Space Complexity

| Data Structure | Complexity | Description |
|----------------|------------|-------------|
| pkgMap | O(p) | p = number of packages |
| optimized | O(s) | s = number of structs |
| AST | O(n) | n = number of nodes |

### 8.3 Optimization Strategies

1. **Package Caching**: Avoid reloading the same package
2. **Struct Deduplication**: Optimize each struct only once
3. **Configurable Concurrency**: Limit parallel workers to prevent OOM (default: 4)
4. **Depth Limiting**: Prevent infinite recursion (default: 50)
5. **Timeout Protection**: Prevent runaway analysis (default: 20 minutes)

## 9. Extensibility

### 9.1 Adding New Report Formats

```go
// Implement the report generation pattern
type JSONReporter struct{}

func (r *JSONReporter) Generate(report *optimizer.Report) error {
    // Implement JSON format report
}
```

### 9.2 Adding New Skip Rules

```go
// Add new rules in shouldSkip()
func (o *Optimizer) shouldSkip() string {
    // ... existing rules

    // New rule: skip protobuf structs
    if hasTag(field, "protobuf") {
        return "protobuf struct"
    }

    return ""
}
```

### 9.3 Adding New Optimization Strategies

```go
// Implement custom reordering strategies
func ReorderFieldsCustom(fields []FieldInfo, strategy Strategy) []FieldInfo {
    switch strategy {
    case SizeDesc:
        // Sort by size descending
    case AlignDesc:
        // Sort by alignment descending
    case Custom:
        // Custom strategy
    }
}
```

### 9.4 Adding New Languages

To add a new language, extend `reporter_i18n.go`:

```go
var jaStrings = i18n{
    ReportTitle:      "🚀 StructOptimizer 最適化レポート",
    GeneratedTime:    "🕐 生成日時",
    // ... all other strings
}
```

Then add the language constant and update `getStrings()`.

## 10. Dependencies

### 10.1 Core Dependencies

```go
require (
    golang.org/x/tools v0.44.0  // packages.Load, go/analysis
)
```

### 10.2 Standard Library Dependencies

- `go/ast`: AST parsing
- `go/types`: Type checking
- `go/token`: Token/position tracking
- `go/parser`: Source code parsing
- `go/printer`: Code formatting
- `golang.org/x/tools/go/packages`: Package loading

## 11. Project Structure

```
structoptimizer/
├── cmd/
│   └── structoptimizer/
│       ├── main.go              # CLI entry point
│       └── main_test.go         # CLI tests
├── analyzer/
│   ├── analyzer.go              # Package and struct analysis
│   ├── analyzer_test.go         # Analyzer tests
│   └── recursive_scan_test.go   # Recursive scan tests
├── optimizer/
│   ├── optimizer.go             # Core optimization logic
│   ├── optimizer_test.go        # Optimization tests
│   ├── field.go                 # Field info types
│   ├── size.go                  # Size calculation
│   ├── reorder.go               # Field reordering
│   ├── skip.go                  # Skip rules
│   ├── types.go                 # Shared types (Report, Config, etc.)
│   ├── helper.go                # Helper functions
│   ├── cache.go                 # Caching utilities
│   ├── collector.go             # Result collection
│   ├── processor.go             # Parallel processing
│   ├── method_index.go          # Method indexing for skip rules
│   ├── file_analyzer.go         # File-level analysis
│   └── *_test.go                # Various test files
├── reporter/
│   ├── reporter.go              # Main reporter + TXT format
│   ├── reporter_md.go           # Markdown format
│   ├── reporter_html.go         # HTML format
│   ├── reporter_types.go        # Reporter types and constructors
│   ├── reporter_utils.go        # Utility functions
│   ├── reporter_i18n.go         # Internationalization (zh/en)
│   └── reporter_test.go         # Reporter tests
├── writer/
│   ├── writer.go                # Source file writer
│   ├── writer_test.go           # Writer tests
│   └── backup_test.go           # Backup tests
├── internal/
│   └── utils/
│       ├── utils.go             # Shared utilities
│       └── utils_test.go        # Utility tests
├── testdata/                    # Test fixtures
├── design.md                    # This document
├── go.mod                       # Module definition
└── go.sum                       # Dependency checksums
```

## 12. Version History

| Version | Date | Changes |
|---------|------|---------|
| v0.1.0 | 2024-01 | Initial release, core functionality |
| v0.2.0 | 2024-01 | Nested struct optimization, cross-package support |
| v0.3.0 | 2024-01 | Report generation, file writing |
| v1.x.x | 2024-2026 | GOPATH support, parallel processing, i18n, reserved fields, recursive scan, bug fixes |

## 13. TODO

- [ ] Support concurrent processing of multiple packages
- [ ] Support JSON format report
- [ ] Support custom optimization strategies
- [ ] Support configuration file (YAML/TOML)
- [ ] Support incremental optimization (skip unchanged files)
- [ ] Support more skip rules (e.g., build tags, protobuf detection)
- [ ] Support additional languages (Japanese, Korean, etc.)
- [ ] Add diff mode (show changes without applying)
- [ ] Add dry-run mode with summary only
