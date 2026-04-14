package reporter

// ReportLevel 报告详细程度
type ReportLevel int

const (
	ReportLevelSummary ReportLevel = iota // 只显示优化总览
	ReportLevelChanged                     // 显示优化总览 + 变化的结构体
	ReportLevelFull                        // 显示所有结构体
)

// Reporter 报告生成器
type Reporter struct {
	format  string      // txt, md, html
	output  string      // 输出路径
	level   ReportLevel // 详细程度
}

// NewReporter 创建报告生成器
func NewReporter(format, output string, level ReportLevel) *Reporter {
	// 验证格式，无效则使用默认值
	validFormats := map[string]bool{"txt": true, "md": true, "html": true}
	if format == "" || !validFormats[format] {
		format = "md"
	}
	return &Reporter{
		format: format,
		output: output,
		level:  level,
	}
}
