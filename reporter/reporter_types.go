package reporter

// ReportLevel represents the verbosity level of the report
type ReportLevel int

const (
	ReportLevelSummary ReportLevel = iota // Show only optimization overview
	ReportLevelChanged                    // Show overview + changed structs
	ReportLevelFull                       // Show all structs
)

// Reporter is the report generator
type Reporter struct {
	format string      // txt, md, html
	output string      // output path
	level  ReportLevel // verbosity level
	lang   Lang        // zh (default) or en
}

// NewReporter creates a report generator
func NewReporter(format, output string, level ReportLevel) *Reporter {
	return NewReporterWithLang(format, output, level, LangZH)
}

// NewReporterWithLang creates a report generator (with language support)
func NewReporterWithLang(format, output string, level ReportLevel, lang Lang) *Reporter {
	// Validate format, use default if invalid
	validFormats := map[string]bool{"txt": true, "md": true, "html": true}
	if format == "" || !validFormats[format] {
		format = "md"
	}
	// Validate language, use default if invalid
	if lang != LangEN {
		lang = LangZH
	}
	return &Reporter{
		format: format,
		output: output,
		level:  level,
		lang:   lang,
	}
}
