package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// GenerateMD generates a Markdown format report
func (r *Reporter) GenerateMD(report *optimizer.Report) (string, error) {
	s := getStrings(r.lang)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", s.ReportTitle))
	sb.WriteString(fmt.Sprintf("> %s：%s  \n", s.GeneratedTime, time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("> %s：v%s\n\n", s.VersionLabel, Version))

	// 1. Optimization overview
	sb.WriteString(fmt.Sprintf("## %s\n\n", s.OverviewTitle))
	sb.WriteString("| 指标 | 数值 |\n")
	sb.WriteString("|------|------|\n")
	sb.WriteString(fmt.Sprintf("| %s | **%d** |\n", s.TotalStructs, report.TotalStructs))
	sb.WriteString(fmt.Sprintf("| %s | **%d** |\n", s.OptimizedStructs, report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("| %s | **%d** |\n", s.SkippedStructs, report.SkippedCount))
	if report.TotalSaved > 0 {
		sb.WriteString(fmt.Sprintf("| %s | **%d %s** (%.2f KB) |\n",
			s.MemorySaved, report.TotalSaved, s.Bytes, float64(report.TotalSaved)/1024))
	} else {
		sb.WriteString(fmt.Sprintf("| %s | 0 %s |\n", s.MemorySaved, s.Bytes))
	}
	// Show total size before/after optimization
	if report.TotalOrigSize > 0 {
		totalOptRate := float64(report.TotalOrigSize-report.TotalOptSize) / float64(report.TotalOrigSize) * 100
		sb.WriteString(fmt.Sprintf("| %s | **%d %s** |\n", s.TotalSizeBefore, report.TotalOrigSize, s.Bytes))
		sb.WriteString(fmt.Sprintf("| %s | **%d %s** |\n", s.TotalSizeAfter, report.TotalOptSize, s.Bytes))
		sb.WriteString(fmt.Sprintf("| %s | **%.1f%%** |\n", s.TotalOptRate, totalOptRate))
	}
	if report.RootStruct != "" {
		sb.WriteString(fmt.Sprintf("| %s | `%s` |\n", s.RootStruct, report.RootStruct))
		if report.RootStructSize > 0 {
			optRate := float64(report.RootStructSize-report.RootStructOptSize) / float64(report.RootStructSize) * 100
			sb.WriteString(fmt.Sprintf("| %s | **%d %s** |\n", s.RootSizeBefore, report.RootStructSize, s.Bytes))
			sb.WriteString(fmt.Sprintf("| %s | **%d %s** |\n", s.RootSizeAfter, report.RootStructOptSize, s.Bytes))
			sb.WriteString(fmt.Sprintf("| %s | **%.1f%%** |\n", s.RootOptRate, optRate))
		}
	}
	sb.WriteString("\n")

	// Classify struct reports
	optimized, skippedNormal, skippedError, unchanged := classifyStructReports(report, s)

	// 2. Adjusted structs (shown first)
	if len(optimized) > 0 {
		sb.WriteString(fmt.Sprintf("## %s\n\n", s.AdjustedTitle))
		sb.WriteString(fmt.Sprintf("**%s**\n\n", fmt.Sprintf(s.AdjustedSummary, len(optimized))))

		for _, sr := range optimized {
			sb.WriteString(fmt.Sprintf("### 📦 %s.%s\n\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("**%s**: `%s`\n\n", s.FileLabel, sr.File))

			// Size comparison
			sb.WriteString(fmt.Sprintf("### %s\n\n", s.MemoryOptTitle))
			sb.WriteString("```\n")
			sb.WriteString(fmt.Sprintf("%s：%6d %s\n", s.BeforeLabel, sr.OrigSize, s.Bytes))
			sb.WriteString("         ↓\n")
			sb.WriteString(fmt.Sprintf("%s：%6d %s\n", s.AfterLabel, sr.OptSize, s.Bytes))
			sb.WriteString("         ↓\n")
			sb.WriteString(fmt.Sprintf("%s：%6d %s (%.1f%%)\n", s.SavedLabel, sr.Saved, s.Bytes, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("```\n\n")

			// Field comparison table
			sb.WriteString(fmt.Sprintf("**%s:**\n\n", s.FieldCompareTitle))
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
				s.ColNo, s.ColBeforeName, s.ColBeforeType, s.ColSize,
				s.ColAfterName, s.ColAfterType, s.ColSize, s.ColChange))
			sb.WriteString("|:----:|----------------|---------------|------|-----------------|---------------|------|------|\n")

			maxLen := len(sr.OrigFields)
			if len(sr.OptFields) > maxLen {
				maxLen = len(sr.OptFields)
			}

			for i := 0; i < maxLen; i++ {
				origName, origType, origSize, optName, optType, optSize, change := getFieldCompareData(sr, i)

				sb.WriteString(fmt.Sprintf("| %d | `%s` | `%s` | %s | `%s` | `%s` | %s | %s |\n",
					i+1, origName, origType, origSize, optName, optType, optSize, change))
			}
			sb.WriteString("\n\n---\n\n")
		}
	}

	// 3. Normal skipped structs (detailed mode only)
	if r.level >= ReportLevelFull && len(skippedNormal) > 0 {
		sb.WriteString(fmt.Sprintf("## %s\n\n", s.SkippedNormalTitle))
		sb.WriteString(fmt.Sprintf("**%s**\n\n", fmt.Sprintf(s.SkippedNormalSummary, len(skippedNormal))))

		for _, sr := range skippedNormal {
			sb.WriteString(fmt.Sprintf("### ✓ %s.%s\n\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", s.ReasonLabel, sr.SkipReason))
		}
	}

	// 4. Error skipped structs
	if len(skippedError) > 0 {
		sb.WriteString(fmt.Sprintf("## %s\n\n", s.SkippedErrorTitle))
		sb.WriteString(fmt.Sprintf("**%s**\n\n", fmt.Sprintf(s.SkippedErrorSummary, len(skippedError))))

		for _, sr := range skippedError {
			sb.WriteString(fmt.Sprintf("### ⏭️ %s.%s\n\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", s.ReasonLabel, sr.SkipReason))
		}
	}

	// 5. Unchanged structs (detailed mode)
	if r.level >= ReportLevelFull && len(unchanged) > 0 {
		sb.WriteString(fmt.Sprintf("## %s\n\n", s.UnchangedTitle))
		sb.WriteString(fmt.Sprintf("**%s**\n\n", fmt.Sprintf(s.UnchangedSummary, len(unchanged))))

		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", s.ColStructName, s.ColPkgPath, s.ColSize))
		sb.WriteString("|----------|--------|------|\n")
		for _, sr := range unchanged {
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %d %s |\n", sr.Name, sr.PkgPath, sr.OrigSize, s.Bytes))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*%s*\n", s.GenBy))

	return sb.String(), nil
}
