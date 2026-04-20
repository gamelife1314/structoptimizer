package reporter

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

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
	s := getStrings(r.lang)
	var sb strings.Builder

	titleLine := fmt.Sprintf("%s %s", s.ReportTitle, fmt.Sprintf("(%s: v%s)", s.VersionLabel, Version))
	sb.WriteString("\n")
	sb.WriteString("╔════════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString(fmt.Sprintf("║  %-78s║\n", titleLine))
	sb.WriteString("╚════════════════════════════════════════════════════════════════════════════════╝\n")
	sb.WriteString(fmt.Sprintf("%s：%s\n\n", s.GeneratedTime, time.Now().Format("2006-01-02 15:04:05")))

	// 1. 优化总览
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│  %-76s│\n", s.OverviewTitle))
	sb.WriteString("├────────────────────────────────────────────────────────────────────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│  %s：%-66d│\n", s.TotalStructs, report.TotalStructs))
	sb.WriteString(fmt.Sprintf("│  %s：%-66d│\n", s.OptimizedStructs, report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("│  %s：%-66d│\n", s.SkippedStructs, report.SkippedCount))
	sb.WriteString(fmt.Sprintf("│  %s：%-60d %s│\n", s.MemorySaved, report.TotalSaved, s.Bytes))
	if report.TotalOrigSize > 0 {
		totalOptRate := float64(report.TotalOrigSize-report.TotalOptSize) / float64(report.TotalOrigSize) * 100
		sb.WriteString(fmt.Sprintf("│  %s：%-58d %s│\n", s.TotalSizeBefore, report.TotalOrigSize, s.Bytes))
		sb.WriteString(fmt.Sprintf("│  %s：%-58d %s│\n", s.TotalSizeAfter, report.TotalOptSize, s.Bytes))
		sb.WriteString(fmt.Sprintf("│  %s：%-60.1f%%│\n", s.TotalOptRate, totalOptRate))
	}
	if report.RootStruct != "" {
		sb.WriteString(fmt.Sprintf("│  %s：%-66s│\n", s.RootStruct, report.RootStruct))
		if report.RootStructSize > 0 {
			optRate := float64(report.RootStructSize-report.RootStructOptSize) / float64(report.RootStructSize) * 100
			sb.WriteString(fmt.Sprintf("│     %s：%-58d %s│\n", s.RootSizeBefore, report.RootStructSize, s.Bytes))
			sb.WriteString(fmt.Sprintf("│     %s：%-58d %s│\n", s.RootSizeAfter, report.RootStructOptSize, s.Bytes))
			sb.WriteString(fmt.Sprintf("│     %s：%-60.1f%%│\n", s.RootOptRate, optRate))
		}
	}
	sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	// 分类结构体
	optimized, skippedNormal, skippedError, unchanged := classifyStructReports(report, s)

	// 2. 调整的结构体（优先显示）
	if len(optimized) > 0 {
		sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
		sb.WriteString(fmt.Sprintf("│  %-76s│\n", fmt.Sprintf("%s (%s)", s.AdjustedTitle, fmt.Sprintf(s.AdjustedSummary, len(optimized)))))
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		for _, sr := range optimized {
			sb.WriteString(fmt.Sprintf("📦 %s.%s\n", sr.PkgPath, sr.Name))
			sb.WriteString(strings.Repeat("─", 120) + "\n")
			sb.WriteString(fmt.Sprintf("   %s：%s\n", s.FileLabel, sr.File))
			sb.WriteString(fmt.Sprintf("   %s：%d %s  →  %s：%d %s  →  %s：%d %s (%.1f%%)\n",
				s.BeforeLabel, sr.OrigSize, s.Bytes, s.AfterLabel, sr.OptSize, s.Bytes, s.SavedLabel, sr.Saved, s.Bytes, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("\n")

			sb.WriteString(fmt.Sprintf("   %s:\n", s.FieldCompareTitle))
			sb.WriteString(fmt.Sprintf("   ┌────┬──────────────────────────┬──────────────────────────┬────────┬──────────────────────────┬──────────────────────────┬────────┬────────┐\n"))
			sb.WriteString(fmt.Sprintf("   │%s│ %-24s │ %-24s │ %-6s │ %-24s │ %-24s │ %-6s │ %-6s │\n",
				s.ColNo, s.ColBeforeName, s.ColBeforeType, s.ColSize, s.ColAfterName, s.ColAfterType, s.ColSize, s.ColChange))
			sb.WriteString("   ├────┼──────────────────────────┼──────────────────────────┼────────┼──────────────────────────┼──────────────────────────┼────────┼────────┤\n")

			maxLen := len(sr.OrigFields)
			if len(sr.OptFields) > maxLen {
				maxLen = len(sr.OptFields)
			}

			for i := 0; i < maxLen; i++ {
				origName, origType, origSize, optName, optType, optSize, change := getFieldCompareData(sr, i)

				sb.WriteString(fmt.Sprintf("   │ %-2d │ %-24s │ %-24s │ %-6s │ %-24s │ %-24s │ %-6s │ %-6s │\n",
					i+1, origName, origType, origSize, optName, optType, optSize, change))
			}
			sb.WriteString("   └────┴──────────────────────────┴──────────────────────────┴────────┴──────────────────────────┴──────────────────────────┴────────┴────────┘\n\n")
		}
	}

	// 3. 正常跳过的结构体（仅详细模式显示）
	if r.level >= ReportLevelFull && len(skippedNormal) > 0 {
		sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
		sb.WriteString(fmt.Sprintf("│  %-76s│\n", fmt.Sprintf("%s (%s)", s.SkippedNormalTitle, fmt.Sprintf(s.SkippedNormalSummary, len(skippedNormal)))))
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		for _, sr := range skippedNormal {
			sb.WriteString(fmt.Sprintf("✓ %s.%s [%d %s]\n", sr.PkgPath, sr.Name, sr.OrigSize, s.Bytes))
			sb.WriteString(fmt.Sprintf("   %s：%s\n\n", s.ReasonLabel, sr.SkipReason))
		}
	}

	// 4. 异常跳过的结构体
	if len(skippedError) > 0 {
		sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
		sb.WriteString(fmt.Sprintf("│  %-76s│\n", fmt.Sprintf("%s (%s)", s.SkippedErrorTitle, fmt.Sprintf(s.SkippedErrorSummary, len(skippedError)))))
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		for _, sr := range skippedError {
			sb.WriteString(fmt.Sprintf("⏭️  %s.%s\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("   %s：%s\n\n", s.ReasonLabel, sr.SkipReason))
		}
	}

	// 5. 未变化的结构体（详细模式下显示）
	if r.level >= ReportLevelFull && len(unchanged) > 0 {
		sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
		sb.WriteString(fmt.Sprintf("│  %-76s│\n", fmt.Sprintf("%s (%s)", s.UnchangedTitle, fmt.Sprintf(s.UnchangedSummary, len(unchanged)))))
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		for _, sr := range unchanged {
			sb.WriteString(fmt.Sprintf("✓ %s.%s [%d %s]\n", sr.PkgPath, sr.Name, sr.OrigSize, s.Bytes))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("╔════════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString(fmt.Sprintf("║  %-78s║\n", s.ReportEnd))
	sb.WriteString("╚════════════════════════════════════════════════════════════════════════════════╝\n")

	return sb.String(), nil
}

// classifyStructReports 分类结构体报告
func classifyStructReports(report *optimizer.Report, s i18n) (optimized, skippedNormal, skippedError, unchanged []*optimizer.StructReport) {
	for _, sr := range report.StructReports {
		if sr.Skipped {
			// 区分正常跳过和异常跳过
			if strings.HasPrefix(sr.SkipReason, s.SkipReasonByMethod) ||
				strings.HasPrefix(sr.SkipReason, s.SkipReasonByName) ||
				sr.SkipReason == s.SkipReasonEmpty ||
				sr.SkipReason == s.SkipReasonSingleField {
				skippedNormal = append(skippedNormal, sr)
			} else {
				skippedError = append(skippedError, sr)
			}
		} else if sr.OrigSize > sr.OptSize {
			optimized = append(optimized, sr)
		} else {
			unchanged = append(unchanged, sr)
		}
	}
	return
}

// getFieldCompareData 获取字段对比数据
func getFieldCompareData(sr *optimizer.StructReport, i int) (origName, origType, origSize, optName, optType, optSize, change string) {
	origName = "-"
	origType = "-"
	origSize = ""
	optName = "-"
	optType = "-"
	optSize = ""
	change = ""

	if i < len(sr.OrigFields) {
		origName = sr.OrigFields[i]
		if sr.FieldTypes != nil {
			if t, ok := sr.FieldTypes[origName]; ok {
				origType = t
			}
		}
		if sr.FieldSizes != nil {
			if size, ok := sr.FieldSizes[origName]; ok {
				origSize = fmt.Sprintf("%d", size)
			}
		}
	}
	if i < len(sr.OptFields) {
		optName = sr.OptFields[i]
		if sr.FieldTypes != nil {
			if t, ok := sr.FieldTypes[optName]; ok {
				optType = t
			}
		}
		if sr.FieldSizes != nil {
			if size, ok := sr.FieldSizes[optName]; ok {
				optSize = fmt.Sprintf("%d", size)
			}
		}
	}

	// 比较变化时使用完整字段信息（包括类型名）
	origKey := origName + ":" + origType
	optKey := optName + ":" + optType
	if origKey != optKey {
		change = "🔄"
	}

	// 显示时匿名字段字段名为空，只显示类型名
	if origName == origType {
		origName = ""
	}
	if optName == optType {
		optName = ""
	}

	return
}
