package optimizer

import (
	"go/types"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// Optimizer 优化器
type Optimizer struct {
	config        *Config
	analyzer      *analyzer.Analyzer
	optimized     map[string]*StructInfo
	report        *Report
	fieldAnalyzer *FieldAnalyzer
	processing    map[string]bool
	maxDepth      int

	// 并行处理相关
	structQueue      []*StructTask
	structByLevel    map[int][]*StructTask
	structByPkgLevel map[int]map[string][]*StructTask // 按层级和包分组
	collecting       map[string]bool
	mu               sync.Mutex
	workerLimit      int
	pkgWorkerLimit   int // 包并发限制

	// 缓存优化
	pkgCache      map[string]*packages.Package
	structCache   map[string]*types.Struct
	filePathCache map[string]string

	// 警告标记
	methodCheckWarned bool // 是否已显示方法检查警告
}

// StructTask 结构体处理任务
type StructTask struct {
	PkgPath    string
	StructName string
	FilePath   string
	Depth      int
	Level      int
}

// Config 优化器配置
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
	PkgWorkerLimit  int // 包并发限制（默认 4，防止 OOM）
}

// StructInfo 结构体信息
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

// StructReport 结构体报告
type StructReport struct {
	Name        string
	PkgPath     string
	File        string
	OrigSize    int64
	OptSize     int64
	Saved       int64
	OrigFields  []string
	OptFields   []string
	FieldTypes  map[string]string // 字段名 -> 类型名
	Skipped     bool
	SkipReason  string
	Depth       int
}

// Report 优化报告
type Report struct {
	TotalStructs   int
	OptimizedCount int
	SkippedCount   int
	TotalSaved     int64
	StructReports  []*StructReport
	RootStruct     string // 主结构体名称（-struct 模式）
}
