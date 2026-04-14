package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
	"github.com/gamelife1314/structoptimizer/reporter"
	"github.com/gamelife1314/structoptimizer/writer"
)

// Config 配置
type Config struct {
	Struct         string
	Package        string
	SourceFile     string
	Write          bool
	Backup        bool
	SkipDirs      string
	SkipFiles     string
	SkipByMethods string
	SkipByNames   string
	Output        string
	Verbose       int
	SortSameSize  bool
	ReportFormat  string
	ProjectType   string // 项目类型：gomod 或 gopath
	GOPATH        string // GOPATH 路径（可选）
	TargetDir     string // 项目目录（位置参数）
	MaxDepth      int    // 最大递归深度
	Timeout       int    // 超时时间（秒）
	PkgScope      string // 包范围限制
	PkgWorkerLimit int   // 包并发限制
}

func main() {
	// 解析命令行参数
	cfg := parseFlags()

	// 验证参数
	if err := validateFlags(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "错误：%v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// 解析逗号分隔的参数
	skipDirs := parseCommaList(cfg.SkipDirs)
	skipFiles := parseCommaList(cfg.SkipFiles)
	skipByMethods := parseCommaList(cfg.SkipByMethods)
	skipByNames := parseCommaList(cfg.SkipByNames)

	// 如果使用了 -skip-by-methods，需要用户确认
	if len(skipByMethods) > 0 {
		if !confirmSkipByMethods() {
			fmt.Println("已取消执行")
			os.Exit(0)
		}
	}

	// 创建分析器
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

	// 创建优化器
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
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "优化失败：%v\n", err)
		os.Exit(1)
	}

	// 生成报告
	var reportLevel reporter.ReportLevel
	if cfg.Verbose >= 3 {
		reportLevel = reporter.ReportLevelFull
	} else if cfg.Verbose >= 2 {
		reportLevel = reporter.ReportLevelChanged
	} else {
		reportLevel = reporter.ReportLevelSummary
	}

	rep := reporter.NewReporter(cfg.ReportFormat, cfg.Output, reportLevel)
	if err := rep.Generate(report); err != nil {
		fmt.Fprintf(os.Stderr, "生成报告失败：%v\n", err)
		os.Exit(1)
	}

	// 写入源文件
	if cfg.Write {
		writerCfg := &writer.Config{
			Backup:  cfg.Backup,
			Verbose: cfg.Verbose,
		}
		w := writer.NewSourceWriter(writerCfg)

		optimized := opt.GetOptimized()
		if err := w.WriteFiles(optimized); err != nil {
			fmt.Fprintf(os.Stderr, "写入文件失败：%v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}

// parseFlags 解析命令行参数
func parseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Struct, "struct", "", "结构体名称（格式：包路径。结构体名）")
	flag.StringVar(&cfg.Package, "package", "", "包路径（与 -struct 互斥）")
	flag.StringVar(&cfg.SourceFile, "source-file", "", "源文件路径")
	flag.BoolVar(&cfg.Write, "write", false, "直接写入源文件")
	flag.BoolVar(&cfg.Backup, "backup", true, "修改前备份源文件")
	flag.StringVar(&cfg.SkipDirs, "skip-dirs", "", "跳过的目录（支持通配符，逗号分隔）")
	flag.StringVar(&cfg.SkipFiles, "skip-files", "", "跳过的文件（支持通配符，逗号分隔）")
	flag.StringVar(&cfg.SkipByMethods, "skip-by-methods", "", "具有这些方法的结构体跳过（逗号分隔）")
	flag.StringVar(&cfg.SkipByNames, "skip-by-names", "", "指定名称的结构体跳过（逗号分隔）")
	flag.StringVar(&cfg.Output, "output", "", "报告输出路径")
	flag.StringVar(&cfg.ReportFormat, "format", "md", "报告格式（txt/md/html）")
	flag.BoolVar(&cfg.SortSameSize, "sort-same-size", false, "大小相同时按字段大小重排")
	flag.StringVar(&cfg.ProjectType, "prj-type", "gomod", "项目类型（gomod/gopath）")
	flag.StringVar(&cfg.GOPATH, "gopath", "", "GOPATH 路径（GOPATH 项目可选）")
	flag.IntVar(&cfg.MaxDepth, "max-depth", 50, "最大递归深度（默认 50）")
	flag.IntVar(&cfg.Timeout, "timeout", 1200, "超时时间（秒，默认 20 分钟）")
	flag.StringVar(&cfg.PkgScope, "pkg-scope", "", "包范围限制（GOPATH 模式必填，只分析此包内的结构体）")
	flag.IntVar(&cfg.PkgWorkerLimit, "pkg-limit", 4, "包并发限制（默认 4，降低可防止 OOM）")

	// 详细程度
	v := flag.Bool("v", false, "显示详细信息")
	vv := flag.Bool("vv", false, "显示调试信息")
	vvv := flag.Bool("vvv", false, "显示跟踪信息")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法：%s [选项] <项目目录>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Golang 结构体对齐优化工具\n\n")
		fmt.Fprintf(os.Stderr, "参数:\n")
		fmt.Fprintf(os.Stderr, "  <项目目录>  Go Module 项目的根目录（包含 go.mod），GOPATH 项目可省略\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  # Go Module 项目\n")
		fmt.Fprintf(os.Stderr, "  %s -struct=pkg.Context /path/to/project\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # GOPATH 项目\n")
		fmt.Fprintf(os.Stderr, "  %s -prj-type=gopath -struct=example.com/pkg.MyStruct\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 自定义最大深度和超时时间\n")
		fmt.Fprintf(os.Stderr, "  %s -struct=pkg.Context -max-depth=100 -timeout=600 /path/to/project\n", os.Args[0])
	}

	flag.Parse()

	// 处理详细程度
	if *vvv {
		cfg.Verbose = 3
	} else if *vv {
		cfg.Verbose = 2
	} else if *v {
		cfg.Verbose = 1
	}

	// 获取项目目录（位置参数）
	if flag.NArg() > 0 {
		cfg.TargetDir = flag.Arg(0)
	}
	// Go Module 项目如果未指定目录，使用当前目录
	if cfg.ProjectType == "gomod" && cfg.TargetDir == "" {
		cfg.TargetDir = "."
	}
	// GOPATH 项目不需要指定目录

	return cfg
}

// validateFlags 验证命令行参数
func validateFlags(cfg *Config) error {
	// -struct 和 -package 互斥
	if cfg.Struct != "" && cfg.Package != "" {
		return fmt.Errorf("-struct 和 -package 不能同时使用")
	}

	// 至少需要一个
	if cfg.Struct == "" && cfg.Package == "" {
		return fmt.Errorf("必须指定 -struct 或 -package")
	}

	// 验证结构体名称格式
	if cfg.Struct != "" {
		if !strings.Contains(cfg.Struct, ".") {
			return fmt.Errorf("结构体名称格式错误，应为：包路径。结构体名")
		}
	}

	// Go Module 项目需要指定目录
	if cfg.ProjectType == "gomod" && cfg.TargetDir == "" {
		return fmt.Errorf("Go Module 项目需要指定项目目录")
	}

	// GOPATH 模式下必须指定包范围
	if cfg.ProjectType == "gopath" && cfg.PkgScope == "" {
		return fmt.Errorf("GOPATH 模式下必须指定 -pkg-scope 参数，用于限制分析范围")
	}

	// 验证报告格式
	if cfg.ReportFormat != "" {
		validFormats := map[string]bool{"txt": true, "md": true, "html": true}
		if !validFormats[cfg.ReportFormat] {
			return fmt.Errorf("无效的报告格式：%s（支持：txt, md, html）", cfg.ReportFormat)
		}
	}

	return nil
}

// parseCommaList 解析逗号分隔的列表
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

// confirmSkipByMethods 确认是否使用 -skip-by-methods
func confirmSkipByMethods() bool {
	fmt.Println("⚠️  警告：-skip-by-methods 需要加载包并检查每个结构体的方法")
	fmt.Println("   这会导致运行速度显著变慢，尤其是大型项目或嵌套层次深的结构体")
	fmt.Println()
	fmt.Println("   建议：")
	fmt.Println("   - 小型项目（<100 个结构体）可以使用")
	fmt.Println("   - 大型项目建议使用 -skip-by-names 代替（极快）")
	fmt.Println()
	fmt.Print("是否继续执行？[y/N]: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
