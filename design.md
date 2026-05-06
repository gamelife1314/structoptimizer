# StructOptimizer Design Document

**Version:** 1.9.0  
**Language:** Go  
**Purpose:** A static analysis tool that reorders struct fields to minimize memory padding, improving cache efficiency and reducing memory footprint for Go programs.

---

## 1. Project Overview

StructOptimizer analyzes Go source code to detect structs with suboptimal field ordering. In Go, struct field alignment rules can introduce padding bytes between fields. By reordering fields from largest to smallest (size-descending), the tool eliminates unnecessary padding, reducing the struct's memory footprint without changing behavior.

Key capabilities:
- Targets individual structs (`-struct`) or entire packages (`-package`)
- Supports both Go Modules and GOPATH project layouts
- Recursive discovery of nested/embedded structs across packages
- Produces detailed comparison reports in TXT, Markdown, or HTML
- Optionally rewrites source files in-place with automatic backup

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          main.go                                 │
│                    (CLI Entry Point)                             │
│  parseFlags() → validateFlags() → parseCommaList()              │
│     │              │                  │                          │
│     ▼              ▼                  ▼                          │
│  analyzer.Config  optimizer.Config  writer.Config               │
└─────┬──────────────────┬──────────────────┬─────────────────────┘
      │                  │                  │
      ▼                  ▼                  ▼
┌──────────┐  ┌──────────────────┐  ┌──────────────┐
│ analyzer │  │    optimizer     │  │    writer    │
│          │◄─│  (uses analyzer  │  │              │
│ struct   │  │   for pkg load)  │  │ backup +     │
│ index    │  │                  │  │ rewrite      │
│ pkg load │  │  collect + proc  │  │ source files │
└────┬─────┘  └────────┬─────────┘  └──────┬───────┘
     │                 │                   │
     │        ┌────────┴─────────┐         │
     │        │  size.go         │         │
     │        │  field.go        │         │
     │        │  reorder.go      │         │
     │        │  skip.go         │         │
     │        │  collector.go    │         │
     │        │  processor.go    │         │
     │        │  helper.go       │         │
     │        │  file_analyzer   │         │
     │        │  method_index.go │         │
     │        └──────────────────┘         │
     │                                     │
     ▼                                     ▼
┌──────────┐  ┌─────────────────────┐
│ reporter │  │  internal/utils     │
│          │  │   (MatchPattern,    │
│ txt/md/  │  │    FormatSize,      │
│ html     │  │    GetGoModRoot,    │
│ output   │  │    ShouldSkip, ...) │
└──────────┘  └─────────────────────┘
```

**Component Relationships:**
- `main.go` is the sole entry point; it parses CLI flags and wires all components
- `analyzer` handles package loading (via `golang.org/x/tools/go/packages`) and struct discovery
- `optimizer` depends on `analyzer` for package loading but can also parse files directly
- `optimizer` calls `reporter` for report generation and `writer` for source file modification
- `internal/utils` provides shared utility functions used across packages

---

## 3. Module/Package Design

### 3.1 `cmd/structoptimizer` (Entry Point)

**File:** `cmd/structoptimizer/main.go` (316 lines)

**Purpose:** CLI argument parsing, validation, component wiring, and main orchestration.

**Key Type — `Config`:**

```go
type Config struct {
    Struct         string       // Struct name (format: package.path.StructName)
    Package        string       // Package path (mutually exclusive with -struct)
    SourceFile     string       // Source file path
    Write          bool         // Write changes to source files
    Backup         bool         // Backup source files before modification
    SkipDirs       string       // Skip directories (wildcards, comma-separated)
    SkipFiles      string       // Skip files (wildcards, comma-separated)
    SkipByMethods  string       // Skip structs with these methods (comma-separated)
    SkipByNames    string       // Skip structs with these names (comma-separated)
    Output         string       // Report output path
    Verbose        int          // 0=silent, 1=info, 2=debug, 3=trace
    SortSameSize   bool         // Reorder fields even when size is the same
    ReportFormat   string       // txt/md/html
    ProjectType    string       // gomod or gopath
    GOPATH         string       // GOPATH path
    TargetDir      string       // Project directory (positional argument)
    MaxDepth       int          // Maximum recursion depth
    Timeout        int          // Timeout in seconds
    PkgScope       string       // Package scope limit
    PkgWorkerLimit int          // Package concurrency limit
    ShowVersion    bool         // Show version
    ReservedFields string       // Reserved field names (comma-separated)
    Recursive      bool         // Recursively scan sub-packages (-package mode)
    Lang           reporter.Lang // Report language
    AllowExternalPkgs bool      // Allow scanning cross-package structs (including vendor)
}
```

**Key Functions:**

```go
func parseFlags() *Config
func validateFlags(cfg *Config) error
func parseCommaList(s string) []string
func confirmSkipByMethods() bool
```

**Flow:**
1. `parseFlags()` parses CLI flags into `Config`
2. `validateFlags()` checks mutual exclusion (-struct vs -package), format validation, project type requirements
3. Comma-separated lists (skip-dirs, skip-files, etc.) are parsed
4. All three component configs (`analyzer.Config`, `optimizer.Config`, `writer.Config`) are populated from the CLI config
5. `opt.Optimize()` runs and returns a report
6. `rep.Generate(report)` produces the output report
7. If `-write` is set, `w.WriteFiles(optimized)` rewrites source files

---

### 3.2 `analyzer` — Package Loading & Struct Discovery

**File:** `analyzer/analyzer.go` (975 lines)

**Purpose:** Load Go packages, discover structs, build a struct index, and provide type information.

**Key Type — `Analyzer`:**

```go
type Analyzer struct {
    config      *Config
    fset        *token.FileSet           // Shared file set for position tracking
    info        *types.Info              // Type info from loaded packages
    pkg         *packages.Package        // Currently loaded package
    pkgMap      map[string]*packages.Package  // Loaded package cache (thread-safe)
    loadedPkgs  map[string]bool               // Tracks which packages have been loaded/attempted
    mu          sync.RWMutex                  // Protects pkgMap and loadedPkgs
    structIndex map[string]*StructLocation    // Struct location index (pkgPath.Name -> file path)
}
```

**Key Type — `Config`:**

```go
type Config struct {
    TargetDir     string
    StructName    string
    Package       string
    SourceFile    string
    SkipDirs      []string
    SkipFiles     []string
    SkipByMethods []string
    SkipByNames   []string
    Verbose       int
    ProjectType   string   // gomod or gopath
    GOPATH        string
}
```

**Key Type — `StructDef`:**

```go
type StructDef struct {
    Name    string
    PkgPath string
    File    string
    Type    *types.Struct
}
```

**Key Type — `StructLocation`:**

```go
type StructLocation struct {
    PkgPath  string
    FileName string
    Loaded   bool
}
```

**Key Functions:**

```go
func NewAnalyzer(cfg *Config) *Analyzer
func (a *Analyzer) BuildStructIndex() error                     // Fast scan all files, no package loading
func (a *Analyzer) LoadPackage(pkgPath string) (*packages.Package, error)  // Thread-safe load with caching
func (a *Analyzer) FindStructByName(pkgPath, structName string) (*types.Struct, string, error)
func (a *Analyzer) FindAllStructs(pkgPath string) ([]StructDef, error)
func (a *Analyzer) FindAllStructsRecursive(rootPkgPath string) ([]StructDef, error)  // BFS sub-package discovery
func (a *Analyzer) LoadPackages(pkgPaths []string) error        // Batch load (optimized)
func (a *Analyzer) GetTypesInfo() *types.Info
func (a *Analyzer) HasMethod(structType *types.Named, methodName string) bool
func (a *Analyzer) HasAnyMethod(structType *types.Named, methodNames []string) bool
func ParseStructName(fullName string) (pkgPath, structName string)  // Splits "pkg.StructName"
```

**Internal Data Flow:**
1. `BuildStructIndex()` recursively scans the target directory for `.go` files
2. Each file is checked for `type ... struct` patterns using `bytes.Contains`
3. Files containing structs are parsed and indexed in `structIndex` by `pkgPath.StructName`
4. `LoadPackage()` uses `packages.Load()` with `NeedTypes | NeedSyntax | NeedTypesInfo` mode
5. Packages are cached in `pkgMap` with `sync.RWMutex` protection
6. `LoadPackages()` batches multiple package loads into one `packages.Load()` call

**Project Type Handling:**
- Go Module mode: Sets `Dir` to target directory, uses default module resolution
- GOPATH mode: Sets `GO111MODULE=off`, sets `GOPATH` in environment, omits `Dir`

---

### 3.3 `optimizer` — Core Optimization Logic

The `optimizer` package is the largest, comprising 12 files. It implements the two-phase optimization process.

#### 3.3.1 `optimizer/types.go` — Core Types

**Key Type — `Optimizer`:**

```go
type Optimizer struct {
    config           *Config
    analyzer         *analyzer.Analyzer
    optimized        map[string]*StructInfo        // key: pkgPath.StructName
    report           *Report
    processing       map[string]bool               // Tracks structs currently being processed (deadlock detection)
    maxDepth         int
    methodIndex      *MethodIndex                  // Cached method index for skip-by-methods
    structQueue      []*StructTask                 // Phase 1 results
    structByLevel    map[int][]*StructTask
    structByPkgLevel map[int]map[string][]*StructTask  // Level → (PkgPath → Tasks)
    collecting       map[string]bool                   // Tracks structs being collected (dedup)
    mu               sync.Mutex
    workerLimit      int                           // Struct-level concurrency limit (10)
    pkgWorkerLimit   int                           // Package-level concurrency limit (default 4)
    pkgCache         map[string]*packages.Package
}
```

**Key Type — `StructTask`:**

```go
type StructTask struct {
    PkgPath    string
    StructName string
    FilePath   string
    Depth      int    // Recursion depth
    Level      int    // Nesting level (leaf = deepest)
}
```

**Key Type — `StructInfo`:**

```go
type StructInfo struct {
    Name       string
    PkgPath    string
    File       string
    Fields     []FieldInfo
    OrigSize   int64
    OptSize    int64
    Optimized  bool
    Skipped    bool
    SkipReason string
    OrigOrder  []string
    OptOrder   []string
}
```

**Key Type — `Config`:**

```go
type Config struct {
    TargetDir       string
    StructName      string
    Package         string
    SourceFile      string
    Write           bool
    Backup          bool
    SkipDirs        []string
    SkipFiles       []string
    SkipByMethods   []string
    SkipByNames     []string
    Verbose         int
    SortSameSize    bool
    Output          string
    ProjectType     string
    GOPATH          string
    MaxDepth        int
    Timeout         int
    PkgScope        string
    PkgWorkerLimit  int
    ReservedFields    []string
    Recursive         bool
    AllowExternalPkgs bool
}
```

**Key Type — `StructReport`:**

```go
type StructReport struct {
    Name       string
    PkgPath    string
    File       string
    OrigSize   int64
    OptSize    int64
    Saved      int64
    OrigFields []string
    OptFields  []string
    FieldTypes map[string]string   // FieldName → TypeName
    FieldSizes map[string]int64    // FieldName → Size (bytes)
    Skipped    bool
    SkipReason string
    Depth      int
    HasEmbed   bool               // Whether struct contains embedded fields
}
```

**Key Type — `Report`:**

```go
type Report struct {
    TotalStructs      int
    OptimizedCount    int
    SkippedCount      int
    TotalSaved        int64
    StructReports     []*StructReport
    RootStruct        string
    RootStructSize    int64
    RootStructOptSize int64
    TotalOrigSize     int64
    TotalOptSize      int64
}
```

#### 3.3.2 `optimizer/field.go` — Field Analysis

**Key Type — `FieldInfo`:**

```go
type FieldInfo struct {
    Name         string
    Type         types.Type
    Size         int64
    Align        int64
    IsEmbed      bool
    IsInterface  bool
    IsStdLib     bool
    IsThirdParty bool
    PkgPath      string
    TypeName     string
    Tag          string
}
```

**Key Type — `FieldAnalyzer`:**

```go
type FieldAnalyzer struct {
    info *types.Info
    fset *token.FileSet
}
```

**Key Functions:**

```go
func NewFieldAnalyzer(info *types.Info, fset *token.FileSet) *FieldAnalyzer
func (fa *FieldAnalyzer) AnalyzeStruct(st *types.Struct, structName, pkgPath, filePath string) *StructInfo
func (fa *FieldAnalyzer) getTypePkg(typ types.Type) string          // Extract package path from type
func (fa *FieldAnalyzer) getTypeName(typ types.Type) string         // Preserve package prefix
func (fa *FieldAnalyzer) getFieldName(field *types.Var) string
func isProjectPkgFast(pkgPath string) bool
```

#### 3.3.3 `optimizer/size.go` — Size Calculation

**Key Functions:**

```go
func CalcStructSizeFromFields(fields []FieldInfo) int64
func CalcStructSize(st *types.Struct, sizes types.Sizes) int64
func calcStructSizeManual(st *types.Struct) int64
func CalcFieldSize(typ types.Type, info *types.Info) (size, align int64)
func basicSize(kind types.BasicKind) (size, align int64)
func sizeofPtr() int64          // Returns 8
func alignofPtr() int64         // Returns 8
func CalcOptimizedSize(fields []FieldInfo, info *types.Info) int64
```

#### 3.3.4 `optimizer/reorder.go` — Field Reordering

**Key Functions:**

```go
func ReorderFields(fields []FieldInfo, sortSameSize bool, reservedFields []string) []FieldInfo
func reorderFieldsInternal(fields []FieldInfo, sortSameSize bool) []FieldInfo
func calcSizeFromFields(fields []FieldInfo) int64
```

#### 3.3.5 `optimizer/collector.go` — Phase 1 Struct Collection

**Key Types:**

```go
type nestedField struct {
    Name     string
    PkgPath  string
    IsStruct bool
}
```

**Key Functions:**

```go
func (o *Optimizer) collectStructs(pkgPath, structName, filePath string, depth, level int)
func (o *Optimizer) parseStructFromFileOnly(pkgPath, structName, filePath string) ([]nestedField, string, error)
func (o *Optimizer) parseStructFields(filePath, structName, pkgPath string) ([]nestedField, string, error)
func (o *Optimizer) parseImports(f *ast.File, pkgPath string) map[string]string
func (o *Optimizer) extractFieldInfo(field *ast.Field, importMap map[string]string, pkgPath, pkgDir string) nestedField
func (o *Optimizer) extractTypeNameFromExpr(expr ast.Expr) (typeName, pkgAlias string)
func (o *Optimizer) findFilesWithStruct(dir, structName string) ([]string, error)
func (o *Optimizer) fileContainsStruct(filePath, structName string) bool
func (o *Optimizer) shouldSkipDir(dirPath string) bool
func (o *Optimizer) isStructTypeInPackage(pkgDir, typeName, pkgPath string) bool
func (o *Optimizer) isInterfaceTypeCrossPackage(pkgPath, typeName string) bool
```

#### 3.3.6 `optimizer/processor.go` — Phase 2 Parallel Processing

**Key Functions:**

```go
func (o *Optimizer) processStructsParallel()                         // Entry point for parallel processing
func (o *Optimizer) processByPackageParallel(level int, pkgTasks map[string][]*StructTask)  // Per-package goroutines
```

#### 3.3.7 `optimizer/file_analyzer.go` — File-Only Analysis (Fast Path)

**Key Functions:**

```go
func analyzeStructFromFile(filePath, structName, pkgPath string) (*StructInfo, *types.Struct, error)
func extractFieldsFromAST(ts *ast.TypeSpec, fset *token.FileSet, pkgDir string) (*types.Struct, []FieldInfo)
func estimateFieldSize(expr ast.Expr) (size, align int64)
func estimateFieldSizeWithLookup(expr ast.Expr, pkgDir string) (size, align int64)
func getExternalStructSize(pkgName, typeName, localPkgDir string) int64
func findStructSizeInPackage(pkgDir, typeName string) int64
func calcInlineStructSize(st *ast.StructType, pkgDir string) int64
func findTypeUnderlyingInPackage(pkgDir, typeName string) types.BasicKind
func sizeOfIdent(name string) (int64, int64)
func parseArrayLength(expr ast.Expr) int64
```

This file implements a "fast path" that estimates field sizes without loading packages via `go/packages`. It has a hardcoded knowledge base of standard library struct sizes (e.g., `time.Time` = 24 bytes, `http.Request` = 480 bytes, `sync.Mutex` = 8 bytes).

#### 3.3.8 `optimizer/method_index.go` — Method Index Cache

**Key Type — `MethodIndex`:**

```go
type MethodIndex struct {
    mu    sync.RWMutex
    cache map[string]map[string]map[string]bool
    //      ^pkgPath     ^structName    ^methodName → exists
}
```

**Key Functions:**

```go
func NewMethodIndex() *MethodIndex
func (mi *MethodIndex) HasMethod(pkgPath, structName, methodPattern string) bool
func (mi *MethodIndex) matchMethod(methodName, pattern string) bool
func (mi *MethodIndex) indexPkg(pkgPath string) error
func (mi *MethodIndex) getPkgDir(pkgPath string) (string, error)
func (mi *MethodIndex) getPkgDirFromGOPATH(pkgPath string) (string, error)
func extractRecvType(expr ast.Expr) string
```

#### 3.3.9 `optimizer/helper.go` — Helper Functions

**Key Functions:**

```go
func isStandardLibraryPkg(pkgPath string) bool       // Heuristic: no dots in path
func isVendorPackage(pkgPath string) bool            // Contains /vendor/ in path
func isStandardLibrary(pkgPath string) bool           // Known std lib list
func (o *Optimizer) isProjectPackage(pkgPath string) bool  // go.mod prefix match
func (o *Optimizer) getPackageDir(pkgPath string) string
func (o *Optimizer) getModulePath() string
```

#### 3.3.10 `optimizer/skip.go` — Skip Logic

**Key Functions:**

```go
func (o *Optimizer) shouldSkip(info *StructInfo, key string) string  // Returns reason or ""
func (o *Optimizer) hasMethodByName(info *StructInfo, methodPattern string) bool
func (o *Optimizer) matchStructName(key, pattern string) bool
```

#### 3.3.11 `optimizer/optimizer.go` — Main Orchestration

**Key Functions:**

```go
func NewOptimizer(cfg *Config, analyzer *analyzer.Analyzer) *Optimizer
func (o *Optimizer) Optimize() (*Report, error)                    // Entry point with timeout
func (o *Optimizer) optimizeInternal() (*Report, error)            // Two-phase logic
func (o *Optimizer) optimizeStruct(pkgPath, structName, filePath string, depth int) (*StructInfo, error)
func (o *Optimizer) addReport(info *StructInfo, skipReason string, depth int)
func (o *Optimizer) GetOptimized() map[string]*StructInfo          // Thread-safe copy
func (o *Optimizer) GetReport() *Report
func (o *Optimizer) Log(level int, format string, args ...interface{})
func isBasicType(typeName string) bool
```

---

### 3.4 `reporter` — Report Generation

**Files:** `reporter.go`, `reporter_types.go`, `reporter_md.go`, `reporter_html.go`, `reporter_i18n.go`

**Purpose:** Generate optimization reports in TXT, Markdown, or HTML format with i18n support (Chinese and English).

**Key Type — `Reporter`:**

```go
type Reporter struct {
    format string       // txt, md, html
    output string       // Output path (or "" for stdout)
    level  ReportLevel  // Summary, Changed, Full
    lang   Lang         // zh or en
}
```

**Key Type — `ReportLevel`:**

```go
type ReportLevel int

const (
    ReportLevelSummary ReportLevel = iota  // Overview only
    ReportLevelChanged                     // Overview + changed structs
    ReportLevelFull                        // All structs (incl. skipped, unchanged)
)
```

**Key Functions:**

```go
func NewReporter(format, output string, level ReportLevel) *Reporter
func NewReporterWithLang(format, output string, level ReportLevel, lang Lang) *Reporter
func (r *Reporter) Generate(report *optimizer.Report) error
func (r *Reporter) GenerateTXT(report *optimizer.Report) (string, error)
func (r *Reporter) GenerateMD(report *optimizer.Report) (string, error)
func (r *Reporter) GenerateHTML(report *optimizer.Report) (string, error)
func classifyStructReports(report *optimizer.Report, s i18n) (...)
func getFieldCompareData(sr *optimizer.StructReport, i int) (...)
```

**Report Structure (all formats):**
1. **Overview:** Total structs processed, optimized/skipped counts, total memory saved, root struct stats
2. **Adjusted Structs:** Side-by-side field order comparison tables
3. **Normal Skipped Structs:** Structs with single field, empty structs, skipped by name/method (Full level only)
4. **Error Skipped Structs:** Structs skipped due to errors, vendor packages, etc.
5. **Unchanged Structs:** Structs that don't benefit from reordering (Full level only)

**i18n:** The `i18n` struct holds all translatable strings. Two built-in locales: `zhStrings` (Chinese, default) and `enStrings` (English). The version constant `"1.7.6"` is stored in `reporter_i18n.go`.

---

### 3.5 `writer` — Source File Rewriting

**File:** `writer/writer.go` (532 lines)

**Purpose:** Backup source files, rewrite them with optimized struct field orders, and clean up old backups.

**Key Type — `SourceWriter`:**

```go
type SourceWriter struct {
    config *Config
    fset   *token.FileSet
}
```

**Key Type — `Config`:**

```go
type Config struct {
    Backup  bool
    Verbose int
}
```

**Key Functions:**

```go
func NewSourceWriter(cfg *Config) *SourceWriter
func (w *SourceWriter) BackupFile(filePath string) (string, error)      // Creates xxx.go.YYYYMMDD_HHMMSS.bak
func (w *SourceWriter) WriteStruct(filePath string, info *optimizer.StructInfo) error  // Single struct rewrite
func (w *SourceWriter) RewriteFile(filePath string, optimizedStructs map[string]*optimizer.StructInfo) error  // Multi-struct rewrite
func (w *SourceWriter) WriteFiles(optimized map[string]*optimizer.StructInfo) error   // Batch write with grouping
func (w *SourceWriter) reorderStructFields(structType *ast.StructType, fields []optimizer.FieldInfo)
func (w *SourceWriter) cleanupOldBackups(filePath string)              // Keeps latest 3 backups
```

**Rewrite process:**
1. `WriteFiles()` groups optimized structs by file path
2. For each file, creates a timestamped backup (`file.go.20060102_150405.bak`)
3. Cleans up old backups, keeping the 3 most recent
4. `RewriteFile()` parses the file with `go/parser`, walks the AST to find matching structs, reorders fields using `reorderStructFields()`, formats with `go/printer` and `go/format`, then writes back
5. Field mapping uses a `fieldMap` keyed by field name (or `"embed:"+typeName` for embedded fields) to preserve AST nodes

**Utility Functions (public, used by tests):**

```go
func SortFieldInfos(fields []optimizer.FieldInfo, sortSameSize bool) []optimizer.FieldInfo
func CreateFieldInfo(name string, size, align int64, isEmbed bool, typeName string) optimizer.FieldInfo
func ReadFile(filePath string) (string, error)
func WriteFile(filePath string, content string) error
func BackupAndWrite(filePath, content string, backup bool) error
func GetStructFields(filePath, structName string) ([]string, error)
func CompareFields(orig, new []string) bool
func FieldsChanged(orig, new []optimizer.FieldInfo) bool
func GenerateStructCode(name string, fields []optimizer.FieldInfo) string
func PrintFields(fields []optimizer.FieldInfo)
func GroupFieldsBySize(fields []optimizer.FieldInfo) map[int64][]optimizer.FieldInfo
func CalculatePadding(fields []optimizer.FieldInfo) int64
```

---

### 3.6 `internal/utils` — Shared Utilities

**File:** `internal/utils/utils.go` (95 lines)

**Key Functions:**

```go
func MatchPattern(pattern, name string) (bool, error)       // filepath.Match wrapper
func MatchDirPattern(pattern, dirName string) bool
func FormatSize(bytes int64) string                          // Human-readable: "1.50 MB", "256 字节"
func GetGoModRoot(dir string) (string, error)                // Walks up to find go.mod
func IsGoModProject(dir string) bool
func ShouldSkip(path string, skipDirs, skipFiles, skipPatterns []string) bool
func SplitStructName(fullName string) (pkgPath, structName string)  // Identical to analyzer.ParseStructName
func Ptr[T any](v T) *T                                     // Generic pointer helper
```

---

## 4. Two-Phase Optimization Process

### Phase 1: Collection

```
collectStructs(pkgPath, structName, filePath="", depth=0, level=0)
│
├── Dedup check: skip if already in optimized{} or collecting{}
├── Mark as collecting (collecting[key] = true)
├── Depth check: skip if depth > maxDepth
├── Vendor/stdlib check: skip if vendor or non-project package
├── Skip-dir check: skip if file path matches skip patterns
│
├── parseStructFromFileOnly(pkgPath, structName, filePath)
│   ├── Determine search directory (go.mod relative or GOPATH/src)
│   ├── findFilesWithStruct(dir, structName) — scan .go files for "type Name struct"
│   │   └── fileContainsStruct() — byte-level "type X struct" check
│   ├── Parse file with go/parser
│   ├── Parse imports → importMap
│   ├── Find struct in AST (supports type xxx struct and type ( ... ) forms)
│   └── For each field:
│       ├── extractTypeNameFromExpr() — handles Ident, StarExpr, SelectorExpr, Array, Map, Chan, Func, Interface, Struct
│       ├── Determine if field type is a struct: isStructTypeInPackage() or isInterfaceTypeCrossPackage()
│       └── Return []nestedField
│
├── Append StructTask{...} to structQueue (with mutex)
│
└── For each nestedField where IsStruct=true:
    ├── Skip standard library packages
    ├── Skip vendor packages (unless AllowExternalPkgs)
    ├── Check pkg-scope limit
    └── Recurse: collectStructs(field.PkgPath, field.Name, "", depth+1, level+1)
```

**-package mode entry point:**
```
if -package is set:
    if -recursive:
        analyzer.FindAllStructsRecursive(pkg) → BFS through imports
    else:
        analyzer.FindAllStructs(pkg) → LoadPackage → iterate declarations
    for each struct: collectStructs(...)
```

**Key characteristics:**
- No package loading (`go/packages`) during collection
- Only file parsing (`go/parser`) for AST extraction
- Recursive discovery of nested structs across packages
- `collecting` map prevents duplicate work from concurrent goroutines
- `level` tracks nesting depth (leaf = highest level number)

### Phase 2: Parallel Optimization

```
processStructsParallel()
│
├── Group tasks: structByPkgLevel[level][pkgPath] = []*StructTask
├── Determine maxLevel
│
└── For level = maxLevel down to 0 (bottom-up):
    ├── processByPackageParallel(level, pkgTasks)
    │   ├── Signal-based concurrency limit (pkgSem ← pkgWorkerLimit)
    │   ├── One goroutine per package
    │   │   ├── Serial processing of structs within the package
    │   │   ├── Panic recovery with stack trace
    │   │   └── Calls optimizeStruct() for each task
    │   └── wg.Wait()
    └── runtime.GC() between levels
```

```
optimizeStruct(pkgPath, structName, filePath, depth)
│
├── Depth limit check
├── Already optimized? (optimized map lookup with mutex)
├── Circular reference detection (processing map)
├── Mark as processing
│
├── Fast path: analyzeStructFromFile(filePath, structName, pkgPath)
│   └── Parse file → extract fields → estimate sizes (file_analyzer.go)
│
├── Slow path (fallback): LoadPackage → findStructInPackage → FieldAnalyzer.AnalyzeStruct
│
├── shouldSkip(info, key):
│   ├── Empty structs
│   ├── Single-field structs
│   ├── Vendor/non-project packages
│   ├── Skip-by-names pattern matching
│   └── Skip-by-methods (via MethodIndex)
│
├── ReorderFields(info.Fields, sortSameSize, reservedFields)
│
├── CalcOptimizedSize(sortedFields, typesInfo)
│
├── If sortedOptSize < origSize → adopt new order
│   └── Set info.Optimized = true
│
└── Store in optimized map, generate StructReport
```

---

## 5. Key Algorithms

### 5.1 Field Reordering (Sort by Size Descending)

Implemented in `reorder.go`:

```go
sort.SliceStable(result, func(i, j int) bool {
    if result[i].Size != result[j].Size {
        return result[i].Size > result[j].Size
    }
    if sortSameSize {
        if result[i].Align != result[j].Align {
            return result[i].Align > result[j].Align
        }
    }
    return result[i].Name < result[j].Name
})
```

**Strategy:**
1. Separate reserved fields (specified via `-reserved-fields`) from normal fields
2. Sort normal fields: primary key = size descending, secondary key = (optional) alignment descending, tertiary = name ascending
3. Append reserved fields at the end in their original order

Large fields (e.g., `[256]byte` = 256 bytes) move to the top, small fields (e.g., `bool` = 1 byte) move to the bottom. This minimizes internal padding because each subsequent field's alignment requirement is more likely to be satisfied by the cumulative offset.

### 5.2 Size Calculation (Alignment & Padding)

Implemented in `size.go`:

```go
func CalcOptimizedSize(fields []FieldInfo, info *types.Info) int64 {
    var offset int64 = 0
    var maxAlign int64 = 1
    for _, field := range fields {
        // Align offset to field's alignment requirement
        if offset % field.Align != 0 {
            offset += field.Align - (offset % field.Align)
        }
        offset += field.Size
        if field.Align > maxAlign {
            maxAlign = field.Align
        }
    }
    // Trailing padding to satisfy max alignment
    if offset % maxAlign != 0 {
        offset += maxAlign - (offset % maxAlign)
    }
    return offset
}
```

**Alignment rules (64-bit architecture):**
| Type | Size | Align |
|------|------|-------|
| bool, int8, uint8 | 1 | 1 |
| int16, uint16 | 2 | 2 |
| int32, uint32, float32 | 4 | 4 |
| int64, uint64, float64, complex64 | 8 | 8 |
| int, uint, uintptr | 8 | 8 |
| string | 16 | 8 |
| pointer | 8 | 8 |
| slice | 24 | 8 |
| map | 8 | 8 |
| chan | 8 | 8 |
| interface | 16 | 8 |
| struct | (sum of fields + padding) | 8 |
| array | elemSize × len | elemAlign |

**Two calculation approaches:**
1. **Fast path (file_analyzer.go):** Estimates sizes without `go/packages` — uses `sizeOfIdent()`, hardcoded stdlib sizes, and cross-package type lookup via vendored/dependency directories
2. **Accurate path (field.go + size.go):** Uses `types.Struct`, `types.Sizes` from loaded packages for exact sizes

### 5.3 Recursive Struct Discovery

Implemented in `collector.go`:

The collector traverses a tree of struct types:
1. Start with the target struct (from `-struct` flag or `-package` scan)
2. Parse the file to find field types
3. For each field, determine if the type is a struct using `isStructTypeInPackage()` (same-package) or `type resolution` (cross-package)
4. Recurse into nested structs, incrementing `level` and `depth`
5. Deduplication via `collecting` map prevents infinite loops

**Cross-package struct detection:**
```go
func (o *Optimizer) extractFieldInfo(field *ast.Field, importMap map[string]string, pkgPath string, pkgDir string) nestedField {
    typeName, pkgAlias := extractTypeNameFromExpr(field.Type)
    fieldPkg := pkgPath  // default: same package
    if pkgAlias != "" {
        if p, ok := importMap[pkgAlias]; ok {
            fieldPkg = p  // use import path
        }
    }
    // Check if it's a struct by scanning the target package's files
    if !isBasicType(typeName) {
        if fieldPkg == pkgPath && pkgDir != "" {
            isStruct = isStructTypeInPackage(pkgDir, typeName, pkgPath)
        } else if fieldPkg != pkgPath {
            isStruct = !isInterfaceTypeCrossPackage(fieldPkg, typeName)
        }
    }
    return nestedField{Name: typeName, PkgPath: fieldPkg, IsStruct: isStruct}
}
```

### 5.4 Cross-Package Resolution

When a field type comes from another package:
1. `extractTypeNameFromExpr()` handles `ast.SelectorExpr` (e.g., `time.Time` → typeName="Time", pkgAlias="time")
2. `parseImports()` maps aliases to full import paths
3. `getPackageDir()` resolves the imported package path to a filesystem directory
4. For GOPATH: `$GOPATH/src/<importPath>`
5. For Go Modules: `$(go.mod module path) → relative path → targetDir + relative`
6. To check if the type is a struct: scan all `.go` files in the package directory for `type Name struct` definitions

Interface detection is done to avoid treating interface types as nested structs that need optimization.

---

## 6. Data Structures

All major types are documented in Section 3. Here is a consolidated reference:

| Type | Package | Fields |
|------|---------|--------|
| `StructTask` | optimizer | PkgPath, StructName, FilePath, Depth, Level |
| `StructInfo` | optimizer | Name, PkgPath, File, Fields, OrigSize, OptSize, Optimized, Skipped, SkipReason, OrigOrder, OptOrder |
| `FieldInfo` | optimizer | Name, Type, Size, Align, IsEmbed, IsInterface, IsStdLib, IsThirdParty, PkgPath, TypeName, Tag |
| `StructReport` | optimizer | Name, PkgPath, File, OrigSize, OptSize, Saved, OrigFields, OptFields, FieldTypes, FieldSizes, Skipped, SkipReason, Depth, HasEmbed |
| `Report` | optimizer | TotalStructs, OptimizedCount, SkippedCount, TotalSaved, StructReports, RootStruct, RootStructSize, RootStructOptSize, TotalOrigSize, TotalOptSize |
| `nestedField` | optimizer | Name, PkgPath, IsStruct |
| `StructDef` | analyzer | Name, PkgPath, File, Type |
| `StructLocation` | analyzer | PkgPath, FileName, Loaded |
| `MethodIndex` | optimizer | mu (sync.RWMutex), cache map[string]map[string]map[string]bool |
| `Optimizer` | optimizer | config, analyzer, optimized, report, processing, maxDepth, methodIndex, structQueue, structByLevel, structByPkgLevel, collecting, mu, workerLimit, pkgWorkerLimit, pkgCache |
| `Reporter` | reporter | format, output, level, lang |

---

## 7. Concurrency Model

### Locking Strategy

The `Optimizer` uses a single `sync.Mutex` for all shared state:

```go
type Optimizer struct {
    mu sync.Mutex  // Protects: optimized, processing, collecting, structQueue, report, structByPkgLevel
}
```

`MethodIndex` uses its own `sync.RWMutex`:
```go
type MethodIndex struct {
    mu    sync.RWMutex  // RLock for reads, Lock for writes
}
```

`Analyzer` uses `sync.RWMutex` for package cache:
```go
type Analyzer struct {
    mu          sync.RWMutex  // Protects pkgMap, loadedPkgs
}
```

### Parallel Processing Strategy

1. **Phase 1 (Collection):** Serial execution in `optimizeInternal()`. Structs are collected into `structQueue`.

2. **Phase 2 (Parallel Optimization):** Two-level parallelism:
   - **Package-level parallelism:** One goroutine per package, limited by `pkgWorkerLimit` (default 4) via channel-based semaphore
   - **Within-package serial execution:** Structs in the same package are processed sequentially to avoid intra-package race conditions
   - **Level-based ordering:** Structs are processed level-by-level from deepest (leaf) to shallowest (root), ensuring nested structs are optimized before their containers

```
              ┌─────────────────────────────────────────┐
              │         packageSem (capacity=4)          │
              └─────┬──────┬──────┬──────┬──────────────┘
                    │      │      │      │
              ┌─────▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐
              │ pkg A  │ │ B  │ │ C  │ │ D  │  ← goroutines
              │ s1 s2  │ │ s3 │ │ s4 │ │ s5 │  ← serial within
              └────────┘ └────┘ └────┘ └────┘
```

3. **Garbage Collection:** `runtime.GC()` is called after each level completes and once more after all levels, to release per-package memory (types, AST, etc.).

4. **Panic Recovery:** Each package goroutine has `defer/recover` that marks remaining structs in that package as skipped on panic.

### Worker Limits

| Limit | Default | Purpose |
|-------|---------|---------|
| `workerLimit` | 10 | Maximum concurrent struct processing (currently unused; per-package is the effective limiter) |
| `pkgWorkerLimit` | 4 | Maximum concurrent package loads (prevents OOM from too many loaded packages) |

---

## 8. CLI Interface

### Positional Arguments

| Argument | Description |
|----------|-------------|
| `<project_dir>` | Go Module project root (contains go.mod). Optional for GOPATH. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-struct` | string | `""` | Struct name (format: `package.path.StructName`) |
| `-package` | string | `""` | Package path (mutually exclusive with `-struct`) |
| `-source-file` | string | `""` | Source file path |
| `-write` | bool | `false` | Write changes to source files |
| `-backup` | bool | `true` | Backup source files before modification |
| `-skip-dirs` | string | `""` | Skip directories (wildcards, comma-separated) |
| `-skip-files` | string | `""` | Skip files (wildcards, comma-separated) |
| `-skip-by-methods` | string | `""` | Skip structs with these methods (comma-separated) |
| `-skip-by-names` | string | `""` | Skip structs with these names (comma-separated) |
| `-output` | string | `""` | Report output path (stdout if empty) |
| `-format` | string | `"md"` | Report format (`txt`/`md`/`html`) |
| `-sort-same-size` | bool | `false` | Reorder fields even when size is the same |
| `-prj-type` | string | `"gomod"` | Project type (`gomod` or `gopath`) |
| `-gopath` | string | `""` | GOPATH path (uses env if not set) |
| `-max-depth` | int | `50` | Maximum recursion depth |
| `-timeout` | int | `1200` | Timeout in seconds |
| `-pkg-scope` | string | `""` | Package scope limit (required for GOPATH) |
| `-pkg-limit` | int | `4` | Package concurrency limit |
| `-reserved-fields` | string | `""` | Fields to keep at the end (comma-separated) |
| `-recursive` | bool | `false` | Recursively scan sub-packages (`-package` mode) |
| `-lang` | string | `"zh"` | Report language (`zh`/`en`) |
| `-allow-external-pkgs` | bool | `false` | Allow scanning cross-package structs (including vendor) |
| `-v` | bool | `false` | Verbose output (INFO level) |
| `-vv` | bool | `false` | Debug output (DEBUG level) |
| `-vvv` | bool | `false` | Trace output (TRACE level) |
| `-version` | bool | `false` | Show version information |

### Validation Rules

- `-struct` and `-package` cannot be used together (mutual exclusion)
- At least one of `-struct` or `-package` must be specified
- `-struct` value must contain a dot (`.`)
- Go Module projects require a target directory
- GOPATH mode requires `-pkg-scope`
- Report format must be one of `txt`, `md`, `html`
- Report language must be one of `zh`, `en`

---

## 9. Configuration Flow

```
CLI Args (flag.Parse)
     │
     ▼
cmd.Config
     │
     ├─→ analyzer.Config
     │   ├── TargetDir, StructName, Package, SourceFile
     │   ├── SkipDirs[], SkipFiles[], SkipByMethods[], SkipByNames[]
     │   ├── Verbose, ProjectType, GOPATH
     │   └── (No Write/Backup/output — analyzer is read-only)
     │
     ├─→ optimizer.Config
     │   ├── All analyzer fields +
     │   ├── Write, Backup, SortSameSize, Output
     │   ├── MaxDepth, Timeout, PkgScope, PkgWorkerLimit
     │   ├── ReservedFields[], Recursive, AllowExternalPkgs
     │   └── (No ReportFormat — optimizer doesn't generate output)
     │
     └─→ writer.Config
         ├── Backup, Verbose
         └── (No struct/skip info — writer is stateless)
```

**Note:** `SkipDirs`, `SkipFiles`, `SkipByMethods`, `SkipByNames`, and `ReservedFields` are stored as comma-separated strings in `cmd.Config` but parsed into `[]string` slices before configuring `analyzer.Config` and `optimizer.Config`.

---

## 10. Skip/Limit Mechanisms

### Directory/File Skips

| Mechanism | Configuration | Implementation |
|-----------|--------------|----------------|
| **Skip by directory** | `-skip-dirs` (comma-separated wildcards) | `shouldSkipDir()` checks basename and path components against patterns |
| **Skip by file** | `-skip-files` (comma-separated wildcards) | `shouldSkipFile()` checks filename against patterns |
| **Built-in vendor skip** | Automatic | `isVendorPackage()` checks for `/vendor/` in path |
| **Standard library skip** | Automatic | `isStandardLibraryPkg()` / `isStandardLibrary()` detect std lib packages |

### Struct-Level Skips

| Mechanism | Configuration | Implementation |
|-----------|--------------|----------------|
| **Empty struct** | Automatic | `len(info.Fields) == 0` |
| **Single-field struct** | Automatic | `len(info.Fields) == 1` |
| **Skip by name** | `-skip-by-names` (with wildcards) | `matchStructName()` supports `*` and `?` |
| **Skip by method** | `-skip-by-methods` (with wildcards) | `hasMethodByName()` via `MethodIndex` cache |
| **Non-project packages** | Automatic (unless `-allow-external-pkgs`) | `isProjectPackage()` checks go.mod path prefix |

### Depth/Limit Controls

| Mechanism | Configuration | Implementation |
|-----------|--------------|----------------|
| **Max recursion depth** | `-max-depth` (default: 50) | Checked in both `collectStructs()` and `optimizeStruct()` |
| **Package scope** | `-pkg-scope` (required for GOPATH) | Prefix check on `field.PkgPath` during collection |
| **Timeout** | `-timeout` (default: 1200s) | `select` waiting on done channel vs `time.After()` |

### External Package Control

| Flag | When | Behavior |
|------|------|----------|
| `-allow-external-pkgs=false` (default) | Collection + Optimization | Skips vendor packages and non-project packages |
| `-allow-external-pkgs=true` | Collection + Optimization | Allows cross-package struct scanning including vendor |

---

## 11. Report Generation

### ReportLevel Determination

```go
var reportLevel reporter.ReportLevel
if cfg.Verbose >= 3 {
    reportLevel = reporter.ReportLevelFull      // All structs, all details
} else if cfg.Verbose >= 2 {
    reportLevel = reporter.ReportLevelChanged   // Summary + changed structs
} else {
    reportLevel = reporter.ReportLevelSummary   // Overview only
}
```

### Classification Logic

```go
func classifyStructReports(report *optimizer.Report, s i18n) (optimized, skippedNormal, skippedError, unchanged) {
    for _, sr := range report.StructReports {
        if sr.Skipped {
            if isUserRequestedSkip(sr) {
                skippedNormal ← sr  // Skipped by name/method, empty, single-field
            } else {
                skippedError ← sr   // Package errors, vendor, circular ref, etc.
            }
        } else if sr.OrigSize > sr.OptSize {
            optimized ← sr
        } else {
            unchanged ← sr
        }
    }
}
```

### Three Output Formats

| Format | Generator | Key Features |
|--------|-----------|-------------|
| **TXT** | `GenerateTXT()` | ASCII art borders, aligned tables, emoji indicators |
| **MD** | `GenerateMD()` | GitHub-flavored markdown, code blocks, tables |
| **HTML** | `GenerateHTML()` | Inline CSS, responsive design, color-coded rows, table wrappers for horizontal scroll |

### i18n Support

- `Lang` type: `"zh"` (Chinese) or `"en"` (English)
- All report strings defined in `zhStrings` and `enStrings` structs
- Selected via `-lang` flag
- Defaults to Chinese if unspecified or invalid

---

## 12. Source File Rewriting

### Backup Strategy

1. **Backup file naming:** `original.go` → `original.go.YYYYMMDD_HHMMSS.bak`
2. **Automatic cleanup:** Keeps only the 3 most recent backups; older ones are deleted
3. **Conditional:** Only backs up when `-backup=true` (default) and `-write=true`

### Rewrite Process

```
WriteFiles(optimized map[string]*StructInfo)
│
├── Group by file path
│
└── For each file:
    ├── BackupFile() → write original content to timestamped .bak
    ├── cleanupOldBackups() → keep 3 newest
    │
    └── RewriteFile(filePath, optimized)
        ├── Parse file with go/parser.ParseFile(ParseComments)
        ├── Normalize paths (Abs comparison for cross-platform)
        ├── Find matching structs in this file
        │
        ├── For each matching struct:
        │   └── reorderStructFields(structType, info.Fields)
        │       ├── Build fieldMap: name → *ast.Field (key = "embed:"+typeName for embedded)
        │       └── Reorder structType.Fields.List to match optimized order
        │
        ├── Format: printer.Fprint() → format.Source() (equivalent to go fmt)
        └── Write formatted result back to file
```

### Safety Mechanisms

- Only structs marked `info.Optimized == true` (verified size reduction) are rewritten
- The original file is always backed up before modification
- If a struct cannot be found in the file (e.g., code changed since analysis), it's silently skipped
- Formatting preserves comments via `parser.ParseComments`

---

## 13. Test Strategy

### Test File Inventory

The project has extensive tests across all packages:

| Package | Test File | Focus |
|---------|-----------|-------|
| `cmd/structoptimizer` | `main_test.go` | CLI flag parsing and validation |
| `optimizer` | `optimizer_test.go` | Core optimization logic |
| `optimizer` | `bug_fix_test.go` | Regression tests for fixed bugs |
| `optimizer` | `complex_gopath_test.go` | GOPATH project structure handling |
| `optimizer` | `complex_nested_struct_test.go` | Deeply nested struct optimization |
| `optimizer` | `embedded_field_test.go` | Embedded/anonymous field handling |
| `optimizer` | `external_pkg_struct_test.go` | Cross-package struct detection |
| `optimizer` | `field_size_sum_test.go` | Field size calculation verification |
| `optimizer` | `gopath_nested_struct_test.go` | GOPATH nested struct discovery |
| `optimizer` | `gopath_specific_bugs_test.go` | GOPATH-specific edge cases |
| `optimizer` | `gopath_unexported_crossfile_test.go` | Unexported cross-file types in GOPATH |
| `optimizer` | `gopath_vendor_test.go` | Vendor directory handling in GOPATH |
| `optimizer` | `helper_test.go` | Helper function tests |
| `optimizer` | `method_index_test.go` | Method index and skip-by-methods |
| `optimizer` | `named_type_test.go` | Named type field handling |
| `optimizer` | `nested_struct_crossfile_test.go` | Nested structs across files |
| `optimizer` | `same_pkg_unexported_test.go` | Unexported types in same package |
| `optimizer` | `skip_test.go` | Skip logic validation |
| `optimizer` | `tags_preserved_test.go` | Struct tag preservation |
| `optimizer` | `type_alias_exact_size_test.go` | Type alias exact size calculation |
| `optimizer` | `type_alias_size_test.go` | Type alias size approximation |
| `analyzer` | `analyzer_test.go` | Package loading and struct discovery |
| `analyzer` | `recursive_scan_test.go` | Recursive BFS package scan |
| `reporter` | `reporter_test.go` | Report generation tests |
| `writer` | `writer_test.go` | File rewrite and formatting tests |
| `writer` | `backup_test.go` | Backup creation and cleanup tests |
| `internal/utils` | `utils_test.go` | Utility function tests |

### Testing Approach

1. **Black-box testing** (`package_test` naming convention): Tests use the public API
2. **Table-driven tests** for size calculations, sorting, and pattern matching
3. **Test fixtures** with sample Go source files for parser/rewriter tests
4. **Mock project directories** for GOPATH and Go Module path resolution tests
5. **Concurrency safety** implicitly tested through integration-style optimization runs
6. **Export helpers** (e.g., `writer.CreateFieldInfo`, `optimizer.EstimateFieldSizeWithLookup`) expose internal functions for testing

---

## Appendix: File Map

```
structoptimizer/
├── cmd/
│   └── structoptimizer/
│       ├── main.go              # CLI entry point, flag parsing, orchestration
│       └── main_test.go         # CLI tests
├── optimizer/
│   ├── optimizer.go             # Main optimization orchestration, two-phase process
│   ├── types.go                 # Core types (Optimizer, StructInfo, Report, etc.)
│   ├── collector.go             # Phase 1: struct collection via file parsing
│   ├── processor.go             # Phase 2: parallel processing by level/package
│   ├── field.go                 # FieldInfo, FieldAnalyzer
│   ├── file_analyzer.go         # File-only analysis (fast path, no package loading)
│   ├── size.go                  # Size calculation, alignment, padding
│   ├── reorder.go               # Field reordering algorithm
│   ├── skip.go                  # Skip logic (empty, single, name, method, package)
│   ├── method_index.go          # Method index cache for skip-by-methods
│   ├── helper.go                # Package identification, go.mod parsing
│   └── *_test.go (19 files)     # Extensive test coverage
├── analyzer/
│   ├── analyzer.go              # Package loading, struct index, BFS discovery
│   ├── analyzer_test.go
│   └── recursive_scan_test.go
├── reporter/
│   ├── reporter.go              # Report generation (TXT format)
│   ├── reporter_types.go        # Reporter struct, ReportLevel, constructors
│   ├── reporter_md.go           # Markdown report generation
│   ├── reporter_html.go         # HTML report generation
│   ├── reporter_i18n.go         # i18n strings, version constant (1.7.6)
│   └── reporter_test.go
├── writer/
│   ├── writer.go                # Source file backup, rewrite, formatting
│   ├── writer_test.go
│   └── backup_test.go
└── internal/
    └── utils/
        ├── utils.go             # Shared utilities (pattern matching, go.mod detection)
        └── utils_test.go
```
