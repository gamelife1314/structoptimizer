package optimizer

import (
	"sync"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// SkipCategory classifies why a struct was skipped
type SkipCategory int

const (
	SkipNone        SkipCategory = iota
	SkipEmpty                    // empty struct
	SkipSingleField              // single-field struct
	SkipByMethod                 // skipped by method pattern
	SkipByName                   // skipped by name pattern
	SkipVendor                   // vendor/third-party package
	SkipStdLib                   // standard library
	SkipNonProject               // non-project internal package
	SkipCircular                 // circular reference
	SkipMaxDepth                 // exceeded max recursion depth
	SkipLoadFailed               // failed to load package
	SkipLookupFailed             // failed to find struct in package
	SkipPanic                    // panic during processing
)

// Optimizer is the struct optimizer
type Optimizer struct {
	config     *Config
	analyzer   *analyzer.Analyzer
	optimized  map[string]*StructInfo
	report     *Report
	processing map[string]bool
	maxDepth   int

	// Module path cache (read once from go.mod)
	modulePath    string
	modulePathSet bool

	// Method indexer
	methodIndex *MethodIndex

	// Parallel processing related
	structQueue      []*StructTask
	structByPkgLevel map[int]map[string][]*StructTask // grouped by level and package
	collecting       map[string]bool
	mu               sync.Mutex
	pkgWorkerLimit   int // package-level concurrency limit
}

// StructTask represents a struct processing task
type StructTask struct {
	PkgPath    string
	StructName string
	FilePath   string
	Depth      int
	Level      int
	ParentKey  string // "pkg.StructName" of parent, empty for root
}

// Config holds the optimizer configuration
type Config struct {
	TargetDir      string
	StructName     string
	Package        string
	SourceFile     string
	Write          bool
	Backup         bool
	SkipDirs       []string
	SkipFiles      []string
	SkipByMethods  []string
	SkipByNames    []string
	Verbose        int
	SortSameSize   bool
	Output         string
	ProjectType    string
	GOPATH         string
	MaxDepth       int
	Timeout        int
	PkgScope       string
	PkgWorkerLimit int      // package-level concurrency limit (default 4, prevents OOM)
	ReservedFields   []string // reserved field names (always placed last)
	Recursive        bool     // recursively scan sub-packages (-package mode)
	AllowExternalPkgs bool    // allow scanning cross-package structs (including vendor directory)
}

// StructInfo holds struct information
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

// StructReport represents a struct optimization report
type StructReport struct {
	Name         string
	PkgPath      string
	File         string
	OrigSize     int64
	OptSize      int64
	Saved        int64
	OrigFields   []string
	OptFields    []string
	FieldTypes   map[string]string // field_name -> type_name
	FieldSizes   map[string]int64  // field_name -> size (bytes)
	Skipped      bool
	SkipReason   string
	SkipCategory SkipCategory // enum-based skip classification
	Depth        int
	ParentKey    string // "pkg.StructName" of parent, empty for root
	HasEmbed     bool   // whether it contains embedded fields
}

// Report is the optimization report
type Report struct {
	TotalStructs      int
	OptimizedCount    int
	SkippedCount      int
	TotalSaved        int64
	StructReports     []*StructReport
	RootStruct        string // root struct name (-struct mode)
	RootStructSize    int64  // root struct original size (root struct only)
	RootStructOptSize int64  // root struct optimized size (root struct only)
	TotalOrigSize     int64  // total original size of all structs
	TotalOptSize      int64  // total optimized size of all structs
}
