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
			sb.WriteString(fmt.Sprintf("📦 %s.%s\n", sr.PkgPath, sr.Name))
			sb.WriteString(strings.Repeat("─", 80) + "\n")
			sb.WriteString(fmt.Sprintf("   📁 文件：%s\n", sr.File))
			sb.WriteString(fmt.Sprintf("   📏 优化前：%d 字节  →  📏 优化后：%d 字节  →  💰 节省：%d 字节 (%.1f%%)\n",
				sr.OrigSize, sr.OptSize, sr.Saved, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("\n")

			sb.WriteString("   优化前字段顺序:\n")
			for i, field := range sr.OrigFields {
				typeInfo := ""
				sizeInfo := ""
				if sr.FieldTypes != nil {
					if t, ok := sr.FieldTypes[field]; ok {
						typeInfo = t
					}
					if typeInfo != "" {
						sizeInfo = fmt.Sprintf(" [%d 字节]", getFieldSize(typeInfo))
					}
				}
				sb.WriteString(fmt.Sprintf("      %2d. %-20s %-15s%s\n", i+1, field, typeInfo, sizeInfo))
			}

			sb.WriteString("\n   优化后字段顺序:\n")
			for i, field := range sr.OptFields {
				typeInfo := ""
				sizeInfo := ""
				if sr.FieldTypes != nil {
					if t, ok := sr.FieldTypes[field]; ok {
						typeInfo = t
					}
					if typeInfo != "" {
						sizeInfo = fmt.Sprintf(" [%d 字节]", getFieldSize(typeInfo))
					}
				}
				arrow := ""
				if i < len(sr.OrigFields) && sr.OrigFields[i] != field {
					arrow = " ⬆️"
				}
				sb.WriteString(fmt.Sprintf("      %2d. %-20s %-15s%s%s\n", i+1, field, typeInfo, sizeInfo, arrow))
			}
			sb.WriteString("\n\n")
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
