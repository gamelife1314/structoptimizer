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

// Config 配置
type Config struct {
	Struct        string
	Package       string
	SourceFile    string
	Write         bool
	Backup        bool
	SkipDirs      stringSlice
	SkipFiles     stringSlice
	SkipPatterns  stringSlice
	SkipByMethods stringSlice
	Output        string
	Verbose       int
	SortSameSize  bool
	TargetDir     string
	ReportFormat  string
}

// stringSlice 自定义字符串切片类型
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
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

	// 创建分析器
	analyzerCfg := &analyzer.Config{
		TargetDir:     cfg.TargetDir,
		StructName:    cfg.Struct,
		Package:       cfg.Package,
		SourceFile:    cfg.SourceFile,
		SkipDirs:      cfg.SkipDirs,
		SkipFiles:     cfg.SkipFiles,
		SkipPatterns:  cfg.SkipPatterns,
		SkipByMethods: cfg.SkipByMethods,
		Verbose:       cfg.Verbose,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	// 创建优化器
	optimizerCfg := &optimizer.Config{
		TargetDir:     cfg.TargetDir,
		StructName:    cfg.Struct,
		Package:       cfg.Package,
		SourceFile:    cfg.SourceFile,
		Write:         cfg.Write,
		Backup:        cfg.Backup,
		SkipDirs:      cfg.SkipDirs,
		SkipFiles:     cfg.SkipFiles,
		SkipPatterns:  cfg.SkipPatterns,
		SkipByMethods: cfg.SkipByMethods,
		Verbose:       cfg.Verbose,
		SortSameSize:  cfg.SortSameSize,
		Output:        cfg.Output,
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "优化失败：%v\n", err)
		os.Exit(1)
	}

	// 生成报告
	rep := reporter.NewReporter(cfg.ReportFormat, cfg.Output)
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
	flag.Var(&cfg.SkipDirs, "skip-dir", "跳过的目录（支持通配符，可多次指定）")
	flag.Var(&cfg.SkipFiles, "skip-file", "跳过的文件（支持通配符，可多次指定）")
	flag.Var(&cfg.SkipPatterns, "skip", "跳过的文件模式（支持通配符，可多次指定）")
	flag.Var(&cfg.SkipByMethods, "skip-by-methods", "具有这些方法的结构体跳过（逗号分隔）")
	flag.StringVar(&cfg.Output, "output", "", "报告输出路径")
	flag.StringVar(&cfg.ReportFormat, "format", "md", "报告格式（txt/md/html）")
	flag.BoolVar(&cfg.SortSameSize, "sort-same-size", false, "大小相同时按字段大小重排")

	// 详细程度
	v := flag.Bool("v", false, "显示详细信息")
	vv := flag.Bool("vv", false, "显示调试信息")
	vvv := flag.Bool("vvv", false, "显示跟踪信息")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法：%s [选项] [目录]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Golang 结构体对齐优化工具\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s -struct=writer/config.Context ./\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --package writer/config ./\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -struct=writer/config.Context --write --backup ./\n", os.Args[0])
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

	// 解析 skip-by-methods（逗号分隔）
	var methods []string
	for _, m := range cfg.SkipByMethods {
		parts := strings.Split(m, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				methods = append(methods, p)
			}
		}
	}
	cfg.SkipByMethods = methods

	// 目标目录
	if flag.NArg() > 0 {
		cfg.TargetDir = flag.Arg(0)
	} else {
		cfg.TargetDir = "."
	}

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

	// 验证报告格式
	if cfg.ReportFormat != "" {
		validFormats := map[string]bool{"txt": true, "md": true, "html": true}
		if !validFormats[cfg.ReportFormat] {
			return fmt.Errorf("无效的报告格式：%s（支持：txt, md, html）", cfg.ReportFormat)
		}
	}

	return nil
}
