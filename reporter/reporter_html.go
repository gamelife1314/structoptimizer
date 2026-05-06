package reporter

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// GenerateHTML generates an HTML format report
func (r *Reporter) GenerateHTML(report *optimizer.Report) (string, error) {
	s := getStrings(r.lang)
	var sb strings.Builder

	langAttr := "zh-CN"
	if r.lang == LangEN {
		langAttr = "en"
	}

	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString(fmt.Sprintf("<html lang=\"%s\">\n", langAttr))
	sb.WriteString("<head>\n")
	sb.WriteString("    <meta charset=\"UTF-8\">\n")
	sb.WriteString("    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("    <title>%s v%s</title>\n", s.ReportTitle, Version))
	sb.WriteString("    <style>\n")
	sb.WriteString("        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; background: #f5f5f5; }\n")
	sb.WriteString("        .container { background: white; border-radius: 8px; padding: 30px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }\n")
	sb.WriteString("        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }\n")
	sb.WriteString("        h2 { color: #34495e; margin-top: 30px; }\n")
	sb.WriteString("        h3 { color: #7f8c8d; }\n")
	sb.WriteString("        .summary { background: #ecf0f1; padding: 20px; border-radius: 8px; margin: 20px 0; }\n")
	sb.WriteString("        .summary table { width: 100%%; }\n")
	sb.WriteString("        .summary td { padding: 8px; }\n")
	sb.WriteString("        .summary td:first-child { font-weight: bold; color: #2c3e50; }\n")
	sb.WriteString("        .section { margin: 30px 0; }\n")
	sb.WriteString("        .struct-card { background: #fafafa; border: 1px solid #e0e0e0; border-radius: 8px; padding: 20px; margin: 20px 0; }\n")
	sb.WriteString("        .struct-card h3 { margin-top: 0; color: #2980b9; }\n")
	sb.WriteString("        .struct-card h4 { color: #7f8c8d; margin-top: 15px; }\n")
	sb.WriteString("        .stats { background: #27ae60; color: white; padding: 15px; border-radius: 8px; display: inline-block; }\n")
	sb.WriteString("        .warning { background: #f39c12; color: white; padding: 10px; border-radius: 5px; margin: 10px 0; }\n")
	sb.WriteString("        table { border-collapse: collapse; width: 100%%; margin: 15px 0; }\n")
	sb.WriteString("        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; font-size: 13px; }\n")
	sb.WriteString("        th { background: #3498db; color: white; white-space: nowrap; }\n")
	sb.WriteString("        td { word-break: break-all; }\n")
	sb.WriteString("        .table-wrapper { overflow-x: auto; margin: 15px 0; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }\n")
	sb.WriteString("        .table-wrapper table { margin: 0; min-width: 900px; }\n")
	sb.WriteString("        tr:nth-child(even) { background: #f9f9f9; }\n")
	sb.WriteString("        .changed { background: #d5f5e3 !important; }\n")
	sb.WriteString("        .skipped { background: #fadbd8; }\n")
	sb.WriteString("        .unchanged { background: #d6eaf8; }\n")
	sb.WriteString("        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #7f8c8d; font-size: 12px; }\n")
	sb.WriteString("    </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	sb.WriteString("    <div class=\"container\">\n")

	sb.WriteString(fmt.Sprintf("        <h1>%s <small>v%s</small></h1>\n", s.ReportTitle, Version))
	sb.WriteString(fmt.Sprintf("        <p>%s：%s</p>\n\n", s.GeneratedTime, time.Now().Format("2006-01-02 15:04:05")))

	// 1. Optimization overview
	sb.WriteString("        <div class=\"summary\">\n")
	sb.WriteString(fmt.Sprintf("            <h2>%s</h2>\n", s.OverviewTitle))
	sb.WriteString("            <table>\n")
	sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d</strong></td></tr>\n", s.TotalStructs, report.TotalStructs))
	sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d</strong></td></tr>\n", s.OptimizedStructs, report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d</strong></td></tr>\n", s.SkippedStructs, report.SkippedCount))
	if report.TotalSaved > 0 {
		sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d %s</strong> (%.2f KB)</td></tr>\n",
			s.MemorySaved, report.TotalSaved, s.Bytes, float64(report.TotalSaved)/1024))
	} else {
		sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>0 %s</strong></td></tr>\n", s.MemorySaved, s.Bytes))
	}
	// Show total size before/after optimization
	if report.TotalOrigSize > 0 {
		totalOptRate := float64(report.TotalOrigSize-report.TotalOptSize) / float64(report.TotalOrigSize) * 100
		sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d %s</strong></td></tr>\n", s.TotalSizeBefore, report.TotalOrigSize, s.Bytes))
		sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d %s</strong></td></tr>\n", s.TotalSizeAfter, report.TotalOptSize, s.Bytes))
		sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%.1f%%</strong></td></tr>\n", s.TotalOptRate, totalOptRate))
	}
	if report.RootStruct != "" {
		sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><code>%s</code></td></tr>\n", s.RootStruct, html.EscapeString(report.RootStruct)))
		if report.RootStructSize > 0 {
			optRate := float64(report.RootStructSize-report.RootStructOptSize) / float64(report.RootStructSize) * 100
			sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d %s</strong></td></tr>\n", s.RootSizeBefore, report.RootStructSize, s.Bytes))
			sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%d %s</strong></td></tr>\n", s.RootSizeAfter, report.RootStructOptSize, s.Bytes))
			sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td><strong>%.1f%%</strong></td></tr>\n", s.RootOptRate, optRate))
		}
	}
	sb.WriteString("            </table>\n")
	sb.WriteString("        </div>\n\n")

	// Struct hierarchy tree (struct mode only)
	addHierarchySectionHTML(&sb, report)

	// Classify struct reports
	optimized, skippedNormal, skippedError, unchanged := classifyStructReports(report, s)

	// 2. Adjusted structs (shown first)
	if len(optimized) > 0 {
		sb.WriteString("        <div class=\"section\">\n")
		sb.WriteString(fmt.Sprintf("            <h2>%s (%s)</h2>\n\n", s.AdjustedTitle, fmt.Sprintf(s.AdjustedSummary, len(optimized))))

		for _, sr := range optimized {
			sb.WriteString(fmt.Sprintf("            <div class=\"struct-card\">\n"))
			sb.WriteString(fmt.Sprintf("                <h3>📦 %s.%s</h3>\n", html.EscapeString(sr.PkgPath), html.EscapeString(sr.Name)))
			sb.WriteString(fmt.Sprintf("                <p><strong>%s</strong>: <code>%s</code></p>\n", s.FileLabel, html.EscapeString(sr.File)))

			sb.WriteString("                <div class=\"stats\">\n")
			sb.WriteString(fmt.Sprintf("                    %s：%d %s → %s：%d %s → %s：%d %s (%.1f%%)\n",
				s.BeforeLabel, sr.OrigSize, s.Bytes, s.AfterLabel, sr.OptSize, s.Bytes, s.SavedLabel, sr.Saved, s.Bytes, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("                </div>\n\n")

			// Field comparison table
			sb.WriteString(fmt.Sprintf("                <h4>%s:</h4>\n", s.FieldCompareTitle))
			sb.WriteString("                <div class=\"table-wrapper\">\n")
			sb.WriteString("                <table>\n")
			sb.WriteString(fmt.Sprintf("                    <tr><th>%s</th><th>%s</th><th>%s</th><th>%s</th><th>%s</th><th>%s</th><th>%s</th><th>%s</th></tr>\n",
				s.ColNo, s.ColBeforeName, s.ColBeforeType, s.ColSize, s.ColAfterName, s.ColAfterType, s.ColSize, s.ColChange))

			maxLen := len(sr.OrigFields)
			if len(sr.OptFields) > maxLen {
				maxLen = len(sr.OptFields)
			}

			for i := 0; i < maxLen; i++ {
				origName, origType, origSize, optName, optType, optSize, change := getFieldCompareData(sr, i)

				className := ""
				if change != "" {
					className = " class=\"changed\""
				}
				sb.WriteString(fmt.Sprintf("                    <tr%s><td>%d</td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td><td>%s</td></tr>\n",
					className, i+1, html.EscapeString(origName), html.EscapeString(origType), origSize,
					html.EscapeString(optName), html.EscapeString(optType), optSize, change))
			}
			sb.WriteString("                </table>\n")
			sb.WriteString("                </div>\n")
			sb.WriteString("            </div>\n\n")
		}
		sb.WriteString("        </div>\n\n")
	}

	// 3. Normal skipped structs (detailed mode only)
	if r.level >= ReportLevelFull && len(skippedNormal) > 0 {
		sb.WriteString("        <div class=\"section\">\n")
		sb.WriteString(fmt.Sprintf("            <h2>%s (%s)</h2>\n\n", s.SkippedNormalTitle, fmt.Sprintf(s.SkippedNormalSummary, len(skippedNormal))))

		for _, sr := range skippedNormal {
			sb.WriteString(fmt.Sprintf("            <div class=\"struct-card\">\n"))
			sb.WriteString(fmt.Sprintf("                <h3>✓ %s.%s</h3>\n", html.EscapeString(sr.PkgPath), html.EscapeString(sr.Name)))
			sb.WriteString(fmt.Sprintf("                <p><strong>%s</strong>: %s</p>\n", s.ReasonLabel, html.EscapeString(sr.SkipReason)))
			sb.WriteString("            </div>\n\n")
		}
		sb.WriteString("        </div>\n\n")
	}

	// 4. Error skipped structs
	if len(skippedError) > 0 {
		sb.WriteString("        <div class=\"section\">\n")
		sb.WriteString(fmt.Sprintf("            <h2>%s (%s)</h2>\n\n", s.SkippedErrorTitle, fmt.Sprintf(s.SkippedErrorSummary, len(skippedError))))

		for _, sr := range skippedError {
			sb.WriteString(fmt.Sprintf("            <div class=\"struct-card skipped\">\n"))
			sb.WriteString(fmt.Sprintf("                <h3>⏭️ %s.%s</h3>\n", html.EscapeString(sr.PkgPath), html.EscapeString(sr.Name)))
			sb.WriteString(fmt.Sprintf("                <p><strong>%s</strong>: %s</p>\n", s.ReasonLabel, html.EscapeString(sr.SkipReason)))
			sb.WriteString("            </div>\n\n")
		}
		sb.WriteString("        </div>\n\n")
	}

	// 5. Unchanged structs (detailed mode)
	if r.level >= ReportLevelFull && len(unchanged) > 0 {
		sb.WriteString("        <div class=\"section\">\n")
		sb.WriteString(fmt.Sprintf("            <h2>%s (%s)</h2>\n\n", s.UnchangedTitle, fmt.Sprintf(s.UnchangedSummary, len(unchanged))))

		sb.WriteString("            <div class=\"table-wrapper\">\n")
		sb.WriteString("            <table>\n")
		sb.WriteString(fmt.Sprintf("                <tr><th>%s</th><th>%s</th><th>%s</th></tr>\n", s.ColStructName, s.ColPkgPath, s.ColSize))
		for _, sr := range unchanged {
			sb.WriteString(fmt.Sprintf("                <tr class=\"unchanged\"><td><code>%s</code></td><td><code>%s</code></td><td>%d %s</td></tr>\n",
				html.EscapeString(sr.Name), html.EscapeString(sr.PkgPath), sr.OrigSize, s.Bytes))
		}
		sb.WriteString("            </table>\n")
		sb.WriteString("            </div>\n")
		sb.WriteString("        </div>\n\n")
	}

	sb.WriteString("        <div class=\"footer\">\n")
	sb.WriteString(fmt.Sprintf("            <p>%s</p>\n", s.GenBy))
	sb.WriteString("        </div>\n")

	sb.WriteString("    </div>\n")
	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")

	return sb.String(), nil
}
