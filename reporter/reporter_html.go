package reporter

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// GenerateHTML 生成 HTML 格式报告
func (r *Reporter) GenerateHTML(report *optimizer.Report) (string, error) {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"zh-CN\">\n")
	sb.WriteString("<head>\n")
	sb.WriteString("    <meta charset=\"UTF-8\">\n")
	sb.WriteString("    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("    <title>StructOptimizer 优化报告 v%s</title>\n", Version))
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
	sb.WriteString("        .stats { background: #27ae60; color: white; padding: 15px; border-radius: 8px; display: inline-block; }\n")
	sb.WriteString("        table { border-collapse: collapse; width: 100%%; margin: 15px 0; }\n")
	sb.WriteString("        th, td { border: 1px solid #ddd; padding: 10px; text-align: left; }\n")
	sb.WriteString("        th { background: #3498db; color: white; }\n")
	sb.WriteString("        tr:nth-child(even) { background: #f9f9f9; }\n")
	sb.WriteString("        .changed { background: #d5f5e3 !important; }\n")
	sb.WriteString("        .skipped { background: #fadbd8; }\n")
	sb.WriteString("        .unchanged { background: #d6eaf8; }\n")
	sb.WriteString("        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #7f8c8d; font-size: 12px; }\n")
	sb.WriteString("    </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	sb.WriteString("    <div class=\"container\">\n")

	sb.WriteString(fmt.Sprintf("        <h1>🚀 StructOptimizer 优化报告 <small>v%s</small></h1>\n", Version))
	sb.WriteString(fmt.Sprintf("        <p>生成时间：%s</p>\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 1. 优化总览
	sb.WriteString("        <div class=\"summary\">\n")
	sb.WriteString("            <h2>📊 优化总览</h2>\n")
	sb.WriteString("            <table>\n")
	sb.WriteString(fmt.Sprintf("                <tr><td>🔹 处理结构体总数</td><td><strong>%d</strong></td></tr>\n", report.TotalStructs))
	sb.WriteString(fmt.Sprintf("                <tr><td>✅ 优化的结构体</td><td><strong>%d</strong></td></tr>\n", report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("                <tr><td>⏭️ 跳过的结构体</td><td><strong>%d</strong></td></tr>\n", report.SkippedCount))
	if report.TotalSaved > 0 {
		sb.WriteString(fmt.Sprintf("                <tr><td>💾 节省内存</td><td><strong>%d 字节</strong> (%.2f KB)</td></tr>\n",
			report.TotalSaved, float64(report.TotalSaved)/1024))
	} else {
		sb.WriteString("                <tr><td>💾 节省内存</td><td><strong>0 字节</strong></td></tr>\n")
	}
	if report.RootStruct != "" {
		sb.WriteString(fmt.Sprintf("                <tr><td>🎯 主结构体</td><td><code>%s</code></td></tr>\n", html.EscapeString(report.RootStruct)))
	}
	sb.WriteString("            </table>\n")
	sb.WriteString("        </div>\n\n")

	// 分类结构体
	var optimized, skipped, unchanged []*optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Skipped {
			skipped = append(skipped, sr)
		} else if sr.Saved > 0 {
			optimized = append(optimized, sr)
		} else {
			unchanged = append(unchanged, sr)
		}
	}

	// 2. 调整的结构体（优先显示）
	if len(optimized) > 0 {
		sb.WriteString("        <div class=\"section\">\n")
		sb.WriteString(fmt.Sprintf("            <h2>✏️ 调整的结构体 (%d 个)</h2>\n\n", len(optimized)))

		for _, sr := range optimized {
			sb.WriteString(fmt.Sprintf("            <div class=\"struct-card\">\n"))
			sb.WriteString(fmt.Sprintf("                <h3>📦 %s.%s</h3>\n", html.EscapeString(sr.PkgPath), html.EscapeString(sr.Name)))
			sb.WriteString(fmt.Sprintf("                <p><strong>📁 文件</strong>: <code>%s</code></p>\n", html.EscapeString(sr.File)))

			sb.WriteString("                <div class=\"stats\">\n")
			sb.WriteString(fmt.Sprintf("                    优化前：%d 字节 → 优化后：%d 字节 → 节省：%d 字节 (%.1f%%)\n",
				sr.OrigSize, sr.OptSize, sr.Saved, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("                </div>\n\n")

			// 优化前字段
			sb.WriteString("                <h4>优化前字段顺序:</h4>\n")
			sb.WriteString("                <table>\n")
			sb.WriteString("                    <tr><th>序号</th><th>字段名</th><th>类型</th><th>大小</th></tr>\n")
			for i, field := range sr.OrigFields {
				typeInfo := ""
				sizeInfo := ""
				if sr.FieldTypes != nil {
					if t, ok := sr.FieldTypes[field]; ok {
						typeInfo = t
						sizeInfo = fmt.Sprintf(" [%d 字节]", getFieldSize(typeInfo))
					}
				}
				sb.WriteString(fmt.Sprintf("                    <tr><td>%d</td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>\n",
					i+1, html.EscapeString(field), html.EscapeString(typeInfo), sizeInfo))
			}
			sb.WriteString("                </table>\n\n")

			// 优化后字段
			sb.WriteString("                <h4>优化后字段顺序:</h4>\n")
			sb.WriteString("                <table>\n")
			sb.WriteString("                    <tr><th>序号</th><th>字段名</th><th>类型</th><th>大小</th><th>变化</th></tr>\n")
			for i, field := range sr.OptFields {
				typeInfo := ""
				sizeInfo := ""
				change := ""
				if sr.FieldTypes != nil {
					if t, ok := sr.FieldTypes[field]; ok {
						typeInfo = t
						sizeInfo = fmt.Sprintf(" [%d 字节]", getFieldSize(typeInfo))
					}
				}
				if i < len(sr.OrigFields) && sr.OrigFields[i] != field {
					change = "⬆️"
				}
				className := ""
				if change != "" {
					className = " class=\"changed\""
				}
				sb.WriteString(fmt.Sprintf("                    <tr%s><td>%d</td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td><td>%s</td></tr>\n",
					className, i+1, html.EscapeString(field), html.EscapeString(typeInfo), sizeInfo, change))
			}
			sb.WriteString("                </table>\n")
			sb.WriteString("            </div>\n\n")
		}
		sb.WriteString("        </div>\n\n")
	}

	// 3. 异常跳过的结构体
	if len(skipped) > 0 {
		sb.WriteString("        <div class=\"section\">\n")
		sb.WriteString(fmt.Sprintf("            <h2>⚠️ 异常跳过的结构体 (%d 个)</h2>\n\n", len(skipped)))

		for _, sr := range skipped {
			sb.WriteString(fmt.Sprintf("            <div class=\"struct-card skipped\">\n"))
			sb.WriteString(fmt.Sprintf("                <h3>⏭️ %s.%s</h3>\n", html.EscapeString(sr.PkgPath), html.EscapeString(sr.Name)))
			sb.WriteString(fmt.Sprintf("                <p><strong>原因</strong>: %s</p>\n", html.EscapeString(sr.SkipReason)))
			sb.WriteString("            </div>\n\n")
		}
		sb.WriteString("        </div>\n\n")
	}

	// 4. 未变化的结构体（详细模式下显示）
	if r.level >= ReportLevelFull && len(unchanged) > 0 {
		sb.WriteString("        <div class=\"section\">\n")
		sb.WriteString(fmt.Sprintf("            <h2>✔️ 未变化的结构体 (%d 个)</h2>\n\n", len(unchanged)))

		sb.WriteString("            <table>\n")
		sb.WriteString("                <tr><th>结构体名</th><th>包路径</th><th>大小</th></tr>\n")
		for _, sr := range unchanged {
			sb.WriteString(fmt.Sprintf("                <tr class=\"unchanged\"><td><code>%s</code></td><td><code>%s</code></td><td>%d 字节</td></tr>\n",
				html.EscapeString(sr.Name), html.EscapeString(sr.PkgPath), sr.OrigSize))
		}
		sb.WriteString("            </table>\n")
		sb.WriteString("        </div>\n\n")
	}

	sb.WriteString("        <div class=\"footer\">\n")
	sb.WriteString("            <p>Generated by StructOptimizer</p>\n")
	sb.WriteString("        </div>\n")

	sb.WriteString("    </div>\n")
	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")

	return sb.String(), nil
}
