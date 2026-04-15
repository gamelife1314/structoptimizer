package reporter

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// 版本信息
const Version = "1.3.0"

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
	sb.WriteString(fmt.Sprintf("║  版本 %-75s║\n", "v"+Version))
	sb.WriteString("╚════════════════════════════════════════════════════════════════════════════════╝\n")
	sb.WriteString(fmt.Sprintf("生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 1. 优化总览
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  📊 优化总览                                                                   │\n")
	sb.WriteString("├────────────────────────────────────────────────────────────────────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│  处理结构体总数：%-61d│\n", report.TotalStructs))
	sb.WriteString(fmt.Sprintf("│  ✅ 优化的结构体：%-61d│\n", report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("│  ⏭️  跳过的结构体：%-61d│\n", report.SkippedCount))
	sb.WriteString(fmt.Sprintf("│  💾 节省内存：     %-61d 字节│\n", report.TotalSaved))
	if report.RootStruct != "" {
		sb.WriteString(fmt.Sprintf("│  🎯 主结构体：     %-61s│\n", report.RootStruct))
		if report.RootStructSize > 0 {
			optRate := float64(report.RootStructSize-report.RootStructOptSize) / float64(report.RootStructSize) * 100
			sb.WriteString(fmt.Sprintf("│     优化前大小：   %-61d 字节│\n", report.RootStructSize))
			sb.WriteString(fmt.Sprintf("│     优化后大小：   %-61d 字节│\n", report.RootStructOptSize))
			sb.WriteString(fmt.Sprintf("│     优化率：       %-61.1f%%│\n", optRate))
		}
	}
	sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

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
		sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
		sb.WriteString(fmt.Sprintf("│  ✏️  调整的结构体 (共 %d 个)                                       │\n", len(optimized)))
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		for _, sr := range optimized {
			sb.WriteString(fmt.Sprintf("📦 %s.%s", sr.PkgPath, sr.Name))
			if sr.HasEmbed {
				sb.WriteString(" ⚠️")
			}
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat("─", 120) + "\n")
			sb.WriteString(fmt.Sprintf("   📁 文件：%s\n", sr.File))
			if sr.HasEmbed {
				sb.WriteString("   ⚠️  警告：包含匿名字段，优化后可能影响兼容性，请手动检查！\n")
				sb.WriteString("   ⚠️  提示：如果使用 -write 参数直接修改源码文件，建议人工审核！\n")
			}
			sb.WriteString(fmt.Sprintf("   📏 优化前：%d 字节  →  📏 优化后：%d 字节  →  💰 节省：%d 字节 (%.1f%%)\n",
				sr.OrigSize, sr.OptSize, sr.Saved, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("\n")

			sb.WriteString("   字段顺序对比:\n")
			sb.WriteString("   ┌────┬──────────────────────────┬──────────────────────────┬────────┬──────────────────────────┬──────────────────────────┬────────┬────────┐\n")
			sb.WriteString("   │序号│ 优化前 - 字段名         │ 优化前 - 类型           │ 大小   │ 优化后 - 字段名         │ 优化后 - 类型           │ 大小   │ 变化   │\n")
			sb.WriteString("   ├────┼──────────────────────────┼──────────────────────────┼────────┼──────────────────────────┼──────────────────────────┼────────┼────────┤\n")
			
			maxLen := len(sr.OrigFields)
			if len(sr.OptFields) > maxLen {
				maxLen = len(sr.OptFields)
			}

			for i := 0; i < maxLen; i++ {
				origName := "-"
				origType := "-"
				origSize := ""
				optName := "-"
				optType := "-"
				optSize := ""
				change := ""

				if i < len(sr.OrigFields) {
					origName = sr.OrigFields[i]
					if sr.FieldTypes != nil {
						if t, ok := sr.FieldTypes[origName]; ok {
							origType = t
							origSize = fmt.Sprintf("%d", getFieldSize(t))
							// 匿名字段：字段名显示为空，只显示类型名
							if origName == origType {
								origName = ""
							}
						}
					}
				}
				if i < len(sr.OptFields) {
					optName = sr.OptFields[i]
					if sr.FieldTypes != nil {
						if t, ok := sr.FieldTypes[optName]; ok {
							optType = t
							optSize = fmt.Sprintf("%d", getFieldSize(t))
							// 匿名字段：字段名显示为空，只显示类型名
							if optName == optType {
								optName = ""
							}
						}
					}
				}

				if origName != "" && optName != "" && origName != optName {
					change = "🔄"
				}

				sb.WriteString(fmt.Sprintf("   │ %-2d │ %-24s │ %-24s │ %-6s │ %-24s │ %-24s │ %-6s │ %-6s │\n",
					i+1, origName, origType, origSize, optName, optType, optSize, change))
			}
			sb.WriteString("   └────┴──────────────────────────┴──────────────────────────┴────────┴──────────────────────────┴──────────────────────────┴────────┴────────┘\n\n")
		}
	}

	// 3. 异常跳过的结构体
	if len(skipped) > 0 {
		sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
		sb.WriteString(fmt.Sprintf("│  ⚠️  异常跳过的结构体 (共 %d 个)                                   │\n", len(skipped)))
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		for _, sr := range skipped {
			sb.WriteString(fmt.Sprintf("⏭️  %s.%s\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("   原因：%s\n\n", sr.SkipReason))
		}
	}

	// 4. 未变化的结构体（详细模式下显示）
	if r.level >= ReportLevelFull && len(unchanged) > 0 {
		sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
		sb.WriteString(fmt.Sprintf("│  ✔️  未变化的结构体 (共 %d 个)                                     │\n", len(unchanged)))
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		for _, sr := range unchanged {
			sb.WriteString(fmt.Sprintf("✓ %s.%s [%d 字节]\n", sr.PkgPath, sr.Name, sr.OrigSize))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("╔════════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║  报告结束                                                                      ║\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════════════════════════╝\n")

	return sb.String(), nil
}
