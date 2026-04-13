package reporter

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// Reporter 报告生成器
type Reporter struct {
	format string // txt, md, html
	output string // 输出路径
}

// NewReporter 创建报告生成器
func NewReporter(format, output string) *Reporter {
	// 验证格式，无效则使用默认值
	validFormats := map[string]bool{"txt": true, "md": true, "html": true}
	if format == "" || !validFormats[format] {
		format = "md"
	}
	return &Reporter{
		format: format,
		output: output,
	}
}

// Generate 生成报告
func (r *Reporter) Generate(report *optimizer.Report) error {
	var content string
	var err error

	switch r.format {
	case "txt":
		content, err = r.GenerateTXT(report)
	case "md":
		content, err = r.GenerateMD(report)
	case "html":
		content, err = r.GenerateHTML(report)
	default:
		content, err = r.GenerateMD(report)
	}

	if err != nil {
		return err
	}

	// 输出到文件或 stdout
	if r.output != "" {
		return os.WriteFile(r.output, []byte(content), 0644)
	} else {
		fmt.Println(content)
		return nil
	}
}

// GenerateTXT 生成 TXT 格式报告
func (r *Reporter) GenerateTXT(report *optimizer.Report) (string, error) {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔════════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║                     StructOptimizer 优化报告                                   ║\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════════════════════════╝\n")
	sb.WriteString(fmt.Sprintf("生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 摘要
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  📊 摘要                                                                       │\n")
	sb.WriteString("├────────────────────────────────────────────────────────────────────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│  总结构体数：   %-64d│\n", report.TotalStructs))
	sb.WriteString(fmt.Sprintf("│  ✅ 已优化：     %-64d│\n", report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("│  ⏭️  跳过：       %-64d│\n", report.SkippedCount))
	sb.WriteString(fmt.Sprintf("│  💾 节省内存：   %-64d│\n", report.TotalSaved))
	sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	// 优化详情
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  🔧 优化详情                                                                   │\n")
	sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	optimizedCount := 0
	for _, sr := range report.StructReports {
		if sr.Skipped {
			continue
		}
		optimizedCount++

		sb.WriteString(fmt.Sprintf("📦 %s.%s\n", sr.PkgPath, sr.Name))
		sb.WriteString(strings.Repeat("─", 80) + "\n")
		sb.WriteString(fmt.Sprintf("   📁 文件：%s\n", sr.File))
		
		if sr.Saved > 0 {
			sb.WriteString(fmt.Sprintf("   📏 优化前：%d 字节  →  📏 优化后：%d 字节  →  💰 节省：%d 字节 (%.1f%%)\n",
				sr.OrigSize, sr.OptSize, sr.Saved, float64(sr.Saved)/float64(sr.OrigSize)*100))
		} else {
			sb.WriteString(fmt.Sprintf("   📏 大小：%d 字节 (无需优化)\n", sr.OrigSize))
		}
		sb.WriteString("\n")

		sb.WriteString("   优化前字段顺序:\n")
		for i, field := range sr.OrigFields {
			sb.WriteString(fmt.Sprintf("      %2d. %s\n", i+1, field))
		}

		if sr.Saved > 0 {
			sb.WriteString("\n   优化后字段顺序:\n")
			for i, field := range sr.OptFields {
				arrow := ""
				if i < len(sr.OrigFields) && sr.OrigFields[i] != field {
					arrow = " ⬆️"
				}
				sb.WriteString(fmt.Sprintf("      %2d. %s%s\n", i+1, field, arrow))
			}
		}
		sb.WriteString("\n\n")
	}

	if optimizedCount == 0 {
		sb.WriteString("   无优化的结构体\n\n")
	}

	// 跳过的结构体
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  ⏭️  跳过的结构体                                                               │\n")
	sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	hasSkipped := false
	for _, sr := range report.StructReports {
		if sr.Skipped {
			hasSkipped = true
			sb.WriteString(fmt.Sprintf("   ⏭️  %s.%s\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("      原因：%s\n\n", sr.SkipReason))
		}
	}

	if !hasSkipped {
		sb.WriteString("   无跳过的结构体\n\n")
	}

	sb.WriteString("╔════════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║  报告结束                                                                      ║\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════════════════════════╝\n")

	return sb.String(), nil
}

// GenerateMD 生成 Markdown 格式报告
func (r *Reporter) GenerateMD(report *optimizer.Report) (string, error) {
	var sb strings.Builder

	sb.WriteString("# 🚀 StructOptimizer 优化报告\n\n")
	sb.WriteString(fmt.Sprintf("> 🕐 生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 摘要
	sb.WriteString("## 📊 摘要\n\n")
	
	// 使用彩色徽章样式
	sb.WriteString("| 指标 | 数值 |\n")
	sb.WriteString("|------|------|\n")
	sb.WriteString(fmt.Sprintf("| 🔹 总结构体数 | **%d** |\n", report.TotalStructs))
	sb.WriteString(fmt.Sprintf("| ✅ 已优化 | **%d** |\n", report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("| ⏭️ 跳过 | **%d** |\n", report.SkippedCount))
	
	if report.TotalSaved > 0 {
		sb.WriteString(fmt.Sprintf("| 💾 节省内存 | **%d 字节** (%.2f KB) |\n", 
			report.TotalSaved, float64(report.TotalSaved)/1024))
	} else {
		sb.WriteString("| 💾 节省内存 | 0 字节 |\n")
	}
	sb.WriteString("\n")

	// 可视化进度条
	if report.TotalStructs > 0 {
		optPercent := float64(report.OptimizedCount) / float64(report.TotalStructs) * 100
		skipPercent := float64(report.SkippedCount) / float64(report.TotalStructs) * 100
		
		sb.WriteString("**优化覆盖率：**\n\n")
		sb.WriteString(fmt.Sprintf("```\n优化：[%s] %.1f%%\n跳过：[%s] %.1f%%\n```\n\n",
			strings.Repeat("█", int(optPercent/5)), optPercent,
			strings.Repeat("░", int(skipPercent/5)), skipPercent))
	}

	// 优化详情
	sb.WriteString("## 🔧 优化详情\n\n")

	optimizedCount := 0
	for _, sr := range report.StructReports {
		if sr.Skipped {
			continue
		}
		optimizedCount++

		sb.WriteString(fmt.Sprintf("### 📦 %s\n\n", sr.Name))
		sb.WriteString(fmt.Sprintf("**包路径**: `%s`\n\n", sr.PkgPath))
		sb.WriteString(fmt.Sprintf("**文件**: `%s`\n\n", sr.File))
		
		// 大小对比
		if sr.Saved > 0 {
			sb.WriteString("### 📏 内存优化\n\n")
			sb.WriteString("```\n")
			sb.WriteString(fmt.Sprintf("优化前：%6d 字节\n", sr.OrigSize))
			sb.WriteString(fmt.Sprintf("         ↓\n"))
			sb.WriteString(fmt.Sprintf("优化后：%6d 字节\n", sr.OptSize))
			sb.WriteString(fmt.Sprintf("         ↓\n"))
			sb.WriteString(fmt.Sprintf("节省：  %6d 字节 (%.1f%%)\n", sr.Saved, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("```\n\n")
		} else {
			sb.WriteString(fmt.Sprintf("**大小**: %d 字节 (字段顺序已最优)\n\n", sr.OrigSize))
		}

		// 字段对比表格
		sb.WriteString("**字段顺序对比:**\n\n")
		sb.WriteString("| 序号 | 优化前 | 优化后 | 变化 |\n")
		sb.WriteString("|:----:|--------|--------|------|\n")
		
		maxLen := len(sr.OrigFields)
		if len(sr.OptFields) > maxLen {
			maxLen = len(sr.OptFields)
		}
		
		for i := 0; i < maxLen; i++ {
			orig := "-"
			opt := "-"
			change := ""
			
			if i < len(sr.OrigFields) {
				orig = sr.OrigFields[i]
			}
			if i < len(sr.OptFields) {
				opt = sr.OptFields[i]
			}
			
			if orig != "-" && opt != "-" && orig != opt {
				change = "🔄 重排"
			} else if orig != "-" && opt == "-" {
				change = "❌ 删除"
			} else if orig == "-" && opt != "-" {
				change = "✅ 新增"
			} else {
				change = "➖ 不变"
			}
			
			sb.WriteString(fmt.Sprintf("| %d | `%s` | `%s` | %s |\n", i+1, orig, opt, change))
		}
		sb.WriteString("\n---\n\n")
	}

	if optimizedCount == 0 {
		sb.WriteString("> 暂无优化的结构体\n\n")
	}

	// 跳过的结构体
	sb.WriteString("## ⏭️ 跳过的结构体\n\n")

	hasSkipped := false
	for _, sr := range report.StructReports {
		if sr.Skipped {
			hasSkipped = true
			sb.WriteString(fmt.Sprintf("### ⏭️ %s\n\n", sr.Name))
			sb.WriteString(fmt.Sprintf("**包路径**: `%s`\n\n", sr.PkgPath))
			sb.WriteString(fmt.Sprintf("**跳过原因**: %s\n\n", sr.SkipReason))
		}
	}

	if !hasSkipped {
		sb.WriteString("> ✅ 无跳过的结构体\n\n")
	}

	// 页脚
	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by [StructOptimizer](https://github.com/gamelife1314/structoptimizer)*\n")

	return sb.String(), nil
}

// GenerateHTML 生成 HTML 格式报告
func (r *Reporter) GenerateHTML(report *optimizer.Report) (string, error) {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>StructOptimizer 优化报告</title>
    <style>
        :root {
            --primary-color: #2563eb;
            --success-color: #16a34a;
            --warning-color: #d97706;
            --danger-color: #dc2626;
            --bg-color: #f8fafc;
            --card-bg: #ffffff;
            --text-color: #1e293b;
            --border-color: #e2e8f0;
        }
        
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            padding: 40px 20px;
        }
        
        .container { max-width: 1200px; margin: 0 auto; }
        
        .header {
            background: linear-gradient(135deg, var(--primary-color), #7c3aed);
            color: white;
            padding: 40px;
            border-radius: 16px;
            margin-bottom: 30px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }
        
        .header h1 { font-size: 2.5rem; margin-bottom: 10px; }
        .header p { opacity: 0.9; }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .stat-card {
            background: var(--card-bg);
            padding: 24px;
            border-radius: 12px;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
            border: 1px solid var(--border-color);
        }
        
        .stat-card .label { color: #64748b; font-size: 0.875rem; margin-bottom: 8px; }
        .stat-card .value { font-size: 2rem; font-weight: 700; color: var(--primary-color); }
        .stat-card.success .value { color: var(--success-color); }
        .stat-card.warning .value { color: var(--warning-color); }
        
        .section {
            background: var(--card-bg);
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 24px;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
            border: 1px solid var(--border-color);
        }
        
        .section h2 {
            font-size: 1.5rem;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 2px solid var(--border-color);
        }
        
        .struct-card {
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            background: #fafafa;
        }
        
        .struct-card h3 {
            color: var(--primary-color);
            margin-bottom: 12px;
            font-size: 1.125rem;
        }
        
        .struct-meta {
            display: flex;
            gap: 20px;
            flex-wrap: wrap;
            margin-bottom: 16px;
            font-size: 0.875rem;
            color: #64748b;
        }
        
        .struct-meta span {
            display: flex;
            align-items: center;
            gap: 4px;
        }
        
        .size-comparison {
            display: flex;
            align-items: center;
            gap: 16px;
            margin: 16px 0;
            padding: 16px;
            background: white;
            border-radius: 8px;
        }
        
        .size-item {
            text-align: center;
            padding: 12px 24px;
            border-radius: 8px;
            background: var(--bg-color);
        }
        
        .size-item .size-value { font-size: 1.5rem; font-weight: 700; }
        .size-item .size-label { font-size: 0.75rem; color: #64748b; }
        .size-item.saved { background: #dcfce7; }
        .size-item.saved .size-value { color: var(--success-color); }
        
        .arrow { font-size: 1.5rem; color: var(--success-color); }
        
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 16px 0;
            font-size: 0.875rem;
        }
        
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
        }
        
        th {
            background: var(--bg-color);
            font-weight: 600;
        }
        
        tr:hover { background: var(--bg-color); }
        
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 500;
        }
        
        .badge-success { background: #dcfce7; color: var(--success-color); }
        .badge-warning { background: #fef3c7; color: var(--warning-color); }
        .badge-info { background: #dbeafe; color: var(--primary-color); }
        
        .skipped-card {
            border-left: 4px solid var(--warning-color);
            padding: 16px;
            margin-bottom: 16px;
            background: #fffbeb;
            border-radius: 0 8px 8px 0;
        }
        
        .progress-bar {
            height: 8px;
            background: var(--border-color);
            border-radius: 4px;
            overflow: hidden;
            margin: 8px 0;
        }
        
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, var(--success-color), var(--primary-color));
            border-radius: 4px;
            transition: width 0.3s;
        }
        
        .footer {
            text-align: center;
            padding: 20px;
            color: #64748b;
            font-size: 0.875rem;
        }
        
        code {
            background: var(--bg-color);
            padding: 2px 6px;
            border-radius: 4px;
            font-family: "SF Mono", Monaco, Consolas, monospace;
            font-size: 0.875em;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 StructOptimizer 优化报告</h1>
            <p>🕐 生成时间：` + time.Now().Format("2006-01-02 15:04:05") + `</p>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="label">🔹 总结构体数</div>
                <div class="value">` + fmt.Sprintf("%d", report.TotalStructs) + `</div>
            </div>
            <div class="stat-card success">
                <div class="label">✅ 已优化</div>
                <div class="value">` + fmt.Sprintf("%d", report.OptimizedCount) + `</div>
            </div>
            <div class="stat-card warning">
                <div class="label">⏭️ 跳过</div>
                <div class="value">` + fmt.Sprintf("%d", report.SkippedCount) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">💾 节省内存</div>
                <div class="value">` + fmt.Sprintf("%d", report.TotalSaved) + ` <span style="font-size:1rem">字节</span></div>
            </div>
        </div>
`)

	// 优化进度
	if report.TotalStructs > 0 {
		optPercent := float64(report.OptimizedCount) / float64(report.TotalStructs) * 100
		sb.WriteString(fmt.Sprintf(`
        <div class="section">
            <h2>📈 优化进度</h2>
            <div class="progress-bar">
                <div class="progress-fill" style="width: %.1f%%"></div>
            </div>
            <p>已优化 %.1f%% 的结构体 (%d/%d)</p>
        </div>
`, optPercent, optPercent, report.OptimizedCount, report.TotalStructs))
	}

	// 优化详情
	sb.WriteString(`
        <div class="section">
            <h2>🔧 优化详情</h2>
`)

	optimizedCount := 0
	for _, sr := range report.StructReports {
		if sr.Skipped {
			continue
		}
		optimizedCount++

		sb.WriteString(fmt.Sprintf(`
            <div class="struct-card">
                <h3>📦 %s</h3>
                <div class="struct-meta">
                    <span>📁 %s</span>
                    <span>📂 %s</span>
                </div>
`, sr.Name, sr.PkgPath, sr.File))

		if sr.Saved > 0 {
			sb.WriteString(fmt.Sprintf(`
                <div class="size-comparison">
                    <div class="size-item">
                        <div class="size-value">%d</div>
                        <div class="size-label">优化前 (字节)</div>
                    </div>
                    <div class="arrow">→</div>
                    <div class="size-item">
                        <div class="size-value">%d</div>
                        <div class="size-label">优化后 (字节)</div>
                    </div>
                    <div class="arrow">→</div>
                    <div class="size-item saved">
                        <div class="size-value">%d</div>
                        <div class="size-label">节省 (字节)</div>
                    </div>
                </div>
`, sr.OrigSize, sr.OptSize, sr.Saved))
		}

		// 字段表格
		sb.WriteString(`
                <h4>字段顺序对比</h4>
                <table>
                    <thead>
                        <tr>
                            <th>序号</th>
                            <th>优化前</th>
                            <th>优化后</th>
                            <th>变化</th>
                        </tr>
                    </thead>
                    <tbody>
`)

		maxLen := len(sr.OrigFields)
		if len(sr.OptFields) > maxLen {
			maxLen = len(sr.OptFields)
		}

		for i := 0; i < maxLen; i++ {
			orig := "-"
			opt := "-"
			badge := `<span class="badge badge-info">不变</span>`

			if i < len(sr.OrigFields) {
				orig = fmt.Sprintf("<code>%s</code>", sr.OrigFields[i])
			}
			if i < len(sr.OptFields) {
				opt = fmt.Sprintf("<code>%s</code>", sr.OptFields[i])
			}

			if orig != "-" && opt != "-" && orig != opt {
				badge = `<span class="badge badge-warning">🔄 重排</span>`
			}

			sb.WriteString(fmt.Sprintf(`
                        <tr>
                            <td>%d</td>
                            <td>%s</td>
                            <td>%s</td>
                            <td>%s</td>
                        </tr>
`, i+1, orig, opt, badge))
		}

		sb.WriteString(`
                    </tbody>
                </table>
            </div>
`)
	}

	if optimizedCount == 0 {
		sb.WriteString(`<p>暂无优化的结构体</p>`)
	}

	sb.WriteString(`
        </div>
`)

	// 跳过的结构体
	sb.WriteString(`
        <div class="section">
            <h2>⏭️ 跳过的结构体</h2>
`)

	hasSkipped := false
	for _, sr := range report.StructReports {
		if sr.Skipped {
			hasSkipped = true
			sb.WriteString(fmt.Sprintf(`
            <div class="skipped-card">
                <strong>📦 %s</strong>
                <p>包路径：<code>%s</code></p>
                <p>跳过原因：%s</p>
            </div>
`, sr.Name, sr.PkgPath, sr.SkipReason))
		}
	}

	if !hasSkipped {
		sb.WriteString(`<p>✅ 无跳过的结构体</p>`)
	}

	sb.WriteString(`
        </div>
        
        <div class="footer">
            <p>Generated by <strong>StructOptimizer</strong></p>
        </div>
    </div>
</body>
</html>
`)

	return sb.String(), nil
}
