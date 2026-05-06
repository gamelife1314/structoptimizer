package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
	"github.com/gamelife1314/structoptimizer/reporter"
	"github.com/gamelife1314/structoptimizer/writer"
)

// Version is defined centrally in the reporter package

// Config holds all CLI configuration
type Config struct {
	Struct         string
	Package        string
	SourceFile     string
	Write          bool
	Backup         bool
	SkipDirs       string
	SkipFiles      string
	SkipByMethods  string
	SkipByNames    string
	Output         string
	Verbose        int
	SortSameSize   bool
	ReportFormat   string
	ProjectType    string // gomod or gopath
	GOPATH         string // GOPATH path (optional)
	TargetDir      string // project directory (positional arg)
	MaxDepth       int    // maximum recursion depth
	Timeout        int    // timeout in seconds
	PkgScope       string // package scope limit
	PkgWorkerLimit int    // package concurrency limit
	ShowVersion    bool   // show version info
	ReservedFields     string        // fields to keep at the end (comma-separated)
	Recursive          bool          // recursively scan sub-packages (-package mode only)
	Lang               reporter.Lang // report language
	AllowExternalPkgs  bool          // allow scanning cross-package structs (including vendor)
}

func main() {
	cfg := parseFlags()

	if cfg.ShowVersion {
		fmt.Printf("structoptimizer version %s\n", reporter.Version)
		os.Exit(0)
	}

	if err := validateFlags(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Parse comma-separated lists
	skipDirs := parseCommaList(cfg.SkipDirs)
	skipFiles := parseCommaList(cfg.SkipFiles)
	skipByMethods := parseCommaList(cfg.SkipByMethods)
	skipByNames := parseCommaList(cfg.SkipByNames)
	reservedFields := parseCommaList(cfg.ReservedFields)

	// Create analyzer
	analyzerCfg := &analyzer.Config{
		TargetDir:     cfg.TargetDir,
		StructName:    cfg.Struct,
		Package:       cfg.Package,
		SourceFile:    cfg.SourceFile,
		SkipDirs:      skipDirs,
		SkipFiles:     skipFiles,
		SkipByMethods: skipByMethods,
		SkipByNames:   skipByNames,
		Verbose:       cfg.Verbose,
		ProjectType:   cfg.ProjectType,
		GOPATH:        cfg.GOPATH,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	// Create optimizer
	optimizerCfg := &optimizer.Config{
		TargetDir:      cfg.TargetDir,
		StructName:     cfg.Struct,
		Package:        cfg.Package,
		SourceFile:     cfg.SourceFile,
		Write:          cfg.Write,
		Backup:         cfg.Backup,
		SkipDirs:       skipDirs,
		SkipFiles:      skipFiles,
		SkipByMethods:  skipByMethods,
		SkipByNames:    skipByNames,
		Verbose:        cfg.Verbose,
		SortSameSize:   cfg.SortSameSize,
		Output:         cfg.Output,
		ProjectType:    cfg.ProjectType,
		GOPATH:         cfg.GOPATH,
		MaxDepth:       cfg.MaxDepth,
		Timeout:        cfg.Timeout,
		PkgScope:       cfg.PkgScope,
		PkgWorkerLimit: cfg.PkgWorkerLimit,
		ReservedFields: reservedFields,
		Recursive:      cfg.Recursive,
		AllowExternalPkgs: cfg.AllowExternalPkgs,
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	// Execute optimization
	report, err := opt.Optimize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Optimization failed: %v\n", err)
		os.Exit(1)
	}

	// Generate report
	var reportLevel reporter.ReportLevel
	if cfg.Verbose >= 3 {
		reportLevel = reporter.ReportLevelFull
	} else if cfg.Verbose >= 2 {
		reportLevel = reporter.ReportLevelChanged
	} else {
		reportLevel = reporter.ReportLevelSummary
	}

	rep := reporter.NewReporterWithLang(cfg.ReportFormat, cfg.Output, reportLevel, cfg.Lang)
	if err := rep.Generate(report); err != nil {
		fmt.Fprintf(os.Stderr, "Report generation failed: %v\n", err)
		os.Exit(1)
	}

	// Write to source files
	if cfg.Write {
		writerCfg := &writer.Config{
			Backup:  cfg.Backup,
			Verbose: cfg.Verbose,
		}
		w := writer.NewSourceWriter(writerCfg)

		optimized := opt.GetOptimized()
		if err := w.WriteFiles(optimized); err != nil {
			fmt.Fprintf(os.Stderr, "Write files failed: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}

// parseFlags parses command-line arguments
func parseFlags() *Config {
	cfg := &Config{}

	flag.BoolVar(&cfg.ShowVersion, "version", false, "Show version information")
	flag.StringVar(&cfg.Struct, "struct", "", "Struct name (format: package.path.StructName)")
	flag.StringVar(&cfg.Package, "package", "", "Package path (mutually exclusive with -struct)")
	flag.StringVar(&cfg.SourceFile, "source-file", "", "Source file path")
	flag.BoolVar(&cfg.Write, "write", false, "Write changes to source files")
	flag.BoolVar(&cfg.Backup, "backup", true, "Backup source files before modification")
	flag.StringVar(&cfg.SkipDirs, "skip-dirs", "", "Skip directories (wildcards, comma-separated)")
	flag.StringVar(&cfg.SkipFiles, "skip-files", "", "Skip files (wildcards, comma-separated)")
	flag.StringVar(&cfg.SkipByMethods, "skip-by-methods", "", "Skip structs with these methods (comma-separated)")
	flag.StringVar(&cfg.SkipByNames, "skip-by-names", "", "Skip structs with these names (comma-separated)")
	flag.StringVar(&cfg.Output, "output", "", "Report output path")
	flag.StringVar(&cfg.ReportFormat, "format", "md", "Report format (txt/md/html)")
	flag.BoolVar(&cfg.SortSameSize, "sort-same-size", false, "Reorder fields even when size is the same")
	flag.StringVar(&cfg.ProjectType, "prj-type", "gomod", "Project type: gomod or gopath")
	flag.StringVar(&cfg.GOPATH, "gopath", "", "GOPATH path (optional, uses env if not set)")
	flag.IntVar(&cfg.MaxDepth, "max-depth", 50, "Maximum recursion depth (default: 50)")
	flag.IntVar(&cfg.Timeout, "timeout", 1200, "Timeout in seconds (default: 1200)")
	flag.StringVar(&cfg.PkgScope, "pkg-scope", "", "Package scope limit (required for GOPATH mode)")
	flag.IntVar(&cfg.PkgWorkerLimit, "pkg-limit", 4, "Package concurrency limit (default: 4, lower to prevent OOM)")
	flag.StringVar(&cfg.ReservedFields, "reserved-fields", "", "Fields to keep at the end (comma-separated, e.g. reserved,padding,XXX)")
	flag.BoolVar(&cfg.Recursive, "recursive", false, "Recursively scan all sub-packages (-package mode only)")
	flag.StringVar((*string)(&cfg.Lang), "lang", "zh", "Report language (zh=Chinese, en=English)")
	flag.BoolVar(&cfg.AllowExternalPkgs, "allow-external-pkgs", false, "Allow scanning cross-package structs (including vendor, default: false)")

	// Verbosity levels
	v := flag.Bool("v", false, "Show verbose output")
	vv := flag.Bool("vv", false, "Show debug output")
	vvv := flag.Bool("vvv", false, "Show trace output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <project_dir>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Golang struct alignment optimization tool\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  <project_dir>  Go Module project root (contains go.mod), optional for GOPATH\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Go Module project\n")
		fmt.Fprintf(os.Stderr, "  %s -struct=pkg.Context /path/to/project\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # GOPATH project\n")
		fmt.Fprintf(os.Stderr, "  %s -prj-type=gopath -struct=example.com/pkg.MyStruct\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Custom max depth and timeout\n")
		fmt.Fprintf(os.Stderr, "  %s -struct=pkg.Context -max-depth=100 -timeout=600 /path/to/project\n", os.Args[0])
	}

	flag.Parse()

	// Handle verbosity levels
	if *vvv {
		cfg.Verbose = 3
	} else if *vv {
		cfg.Verbose = 2
	} else if *v {
		cfg.Verbose = 1
	}

	// Get project directory (positional arg)
	if flag.NArg() > 0 {
		cfg.TargetDir = flag.Arg(0)
	}
	// Go Module project uses current dir if not specified
	if cfg.ProjectType == "gomod" && cfg.TargetDir == "" {
		cfg.TargetDir = "."
	}
	// GOPATH project does not require a directory

	return cfg
}

// validateFlags validates command-line arguments
func validateFlags(cfg *Config) error {
	// -struct and -package are mutually exclusive
	if cfg.Struct != "" && cfg.Package != "" {
		return fmt.Errorf("-struct and -package cannot be used together")
	}

	// At least one must be specified
	if cfg.Struct == "" && cfg.Package == "" {
		return fmt.Errorf("must specify -struct or -package")
	}

	// Validate struct name format
	if cfg.Struct != "" {
		if !strings.Contains(cfg.Struct, ".") {
			return fmt.Errorf("invalid struct name format, expected: package.path.StructName")
		}
	}

	// Go Module project requires a target directory
	if cfg.ProjectType == "gomod" && cfg.TargetDir == "" {
		return fmt.Errorf("Go Module project requires a target directory")
	}

	// GOPATH mode requires pkg-scope
	if cfg.ProjectType == "gopath" && cfg.PkgScope == "" {
		return fmt.Errorf("GOPATH mode requires -pkg-scope parameter to limit analysis scope")
	}

	// Validate report format
	if cfg.ReportFormat != "" {
		validFormats := map[string]bool{"txt": true, "md": true, "html": true}
		if !validFormats[cfg.ReportFormat] {
			return fmt.Errorf("invalid report format: %s (supported: txt, md, html)", cfg.ReportFormat)
		}
	}

	// Validate report language
	if cfg.Lang != "" && cfg.Lang != reporter.LangZH && cfg.Lang != reporter.LangEN {
		return fmt.Errorf("invalid report language: %s (supported: zh, en)", cfg.Lang)
	}

	return nil
}

// parseCommaList splits a comma-separated string into a slice
func parseCommaList(s string) []string {
	if s == "" {
		return nil
	}

	var result []string
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
