package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// GenerateMD 生成 Markdown 格式报告
func (r *Reporter) GenerateMD(report *optimizer.Report) (string, error) {
	var sb strings.Builder

	sb.WriteString("# 🚀 StructOptimizer 优化报告\n\n")
	sb.WriteString(fmt.Sprintf("> 🕐 生成时间：%s  \n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("> 📦 版本：v%s\n\n", Version))

	// 1. 优化总览
	sb.WriteString("## 📊 优化总览\n\n")
	sb.WriteString("| 指标 | 数值 |\n")
	sb.WriteString("|------|------|\n")
	sb.WriteString(fmt.Sprintf("| 🔹 处理结构体总数 | **%d** |\n", report.TotalStructs))
	sb.WriteString(fmt.Sprintf("| ✅ 优化的结构体 | **%d** |\n", report.OptimizedCount))
	sb.WriteString(fmt.Sprintf("| ⏭️ 跳过的结构体 | **%d** |\n", report.SkippedCount))
	if report.TotalSaved > 0 {
		sb.WriteString(fmt.Sprintf("| 💾 节省内存 | **%d 字节** (%.2f KB) |\n",
			report.TotalSaved, float64(report.TotalSaved)/1024))
	} else {
		sb.WriteString("| 💾 节省内存 | 0 字节 |\n")
	}
	if report.RootStruct != "" {
		sb.WriteString(fmt.Sprintf("| 🎯 主结构体 | `%s` |\n", report.RootStruct))
		if report.RootStructSize > 0 {
			optRate := float64(report.RootStructSize-report.RootStructOptSize) / float64(report.RootStructSize) * 100
			sb.WriteString(fmt.Sprintf("| 📏 优化前大小 | **%d 字节** |\n", report.RootStructSize))
			sb.WriteString(fmt.Sprintf("| 📏 优化后大小 | **%d 字节** |\n", report.RootStructOptSize))
			sb.WriteString(fmt.Sprintf("| 📈 优化率 | **%.1f%%** |\n", optRate))
		}
	}
	sb.WriteString("\n")

	// 分类结构体
	var optimized, skippedNormal, skippedError, unchanged []*optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Skipped {
			// 区分正常跳过和异常跳过
			if strings.HasPrefix(sr.SkipReason, "通过方法指定跳过") ||
				strings.HasPrefix(sr.SkipReason, "通过名字指定跳过") ||
				sr.SkipReason == "空结构体" ||
				sr.SkipReason == "单字段结构体" {
				skippedNormal = append(skippedNormal, sr)
			} else {
				skippedError = append(skippedError, sr)
			}
		} else if sr.OrigSize > sr.OptSize {
			// 与 OptimizedCount 统计标准一致
			optimized = append(optimized, sr)
		} else {
			unchanged = append(unchanged, sr)
		}
	}

	// 2. 调整的结构体（优先显示）
	if len(optimized) > 0 {
		sb.WriteString("## ✏️ 调整的结构体\n\n")
		sb.WriteString(fmt.Sprintf("**共 %d 个结构体被优化**\n\n", len(optimized)))

		for _, sr := range optimized {
			warning := ""
			if sr.HasEmbed {
				warning = " ⚠️"
			}
			sb.WriteString(fmt.Sprintf("### 📦 %s.%s%s\n\n", sr.PkgPath, sr.Name, warning))
			sb.WriteString(fmt.Sprintf("**📁 文件**: `%s`\n\n", sr.File))
			if sr.HasEmbed {
				sb.WriteString("> ⚠️  **警告：包含匿名字段，优化后可能影响兼容性，请手动检查！**\n")
				sb.WriteString("> ⚠️  **提示：如果使用 -write 参数直接修改源码文件，建议人工审核！**\n\n")
			}

			// 大小对比
			sb.WriteString("### 📏 内存优化\n\n")
			sb.WriteString("```\n")
			sb.WriteString(fmt.Sprintf("优化前：%6d 字节\n", sr.OrigSize))
			sb.WriteString(fmt.Sprintf("         ↓\n"))
			sb.WriteString(fmt.Sprintf("优化后：%6d 字节\n", sr.OptSize))
			sb.WriteString(fmt.Sprintf("         ↓\n"))
			sb.WriteString(fmt.Sprintf("节省：  %6d 字节 (%.1f%%)\n", sr.Saved, float64(sr.Saved)/float64(sr.OrigSize)*100))
			sb.WriteString("```\n\n")

			// 字段对比表格
			sb.WriteString("**字段顺序对比:**\n\n")
			sb.WriteString("| 序号 | 优化前 - 字段名 | 优化前 - 类型 | 大小 | 优化后 - 字段名 | 优化后 - 类型 | 大小 | 变化 |\n")
			sb.WriteString("|:----:|----------------|---------------|------|-----------------|---------------|------|------|\n")

			maxLen := len(sr.OrigFields)
			if len(sr.OptFields) > maxLen {
				maxLen = len(sr.OptFields)
			}

			for i := 0; i < maxLen; i++ {
				origName := ""
				origType := "-"
				origSize := ""
				optName := ""
				optType := "-"
				optSize := ""
				change := ""

				if i < len(sr.OrigFields) {
					origName = sr.OrigFields[i]
					if sr.FieldTypes != nil {
						if t, ok := sr.FieldTypes[origName]; ok {
							origType = t
							// 使用准确的字段大小，如果有的话
							if sr.FieldSizes != nil {
								if s, ok := sr.FieldSizes[origName]; ok {
									origSize = fmt.Sprintf("%d", s)
								} else {
									origSize = fmt.Sprintf("%d", getFieldSize(t))
								}
							} else {
								origSize = fmt.Sprintf("%d", getFieldSize(t))
							}
						}
					}
				}
				if i < len(sr.OptFields) {
					optName = sr.OptFields[i]
					if sr.FieldTypes != nil {
						if t, ok := sr.FieldTypes[optName]; ok {
							optType = t
							// 使用准确的字段大小，如果有的话
							if sr.FieldSizes != nil {
								if s, ok := sr.FieldSizes[optName]; ok {
									optSize = fmt.Sprintf("%d", s)
								} else {
									optSize = fmt.Sprintf("%d", getFieldSize(t))
								}
							} else {
								optSize = fmt.Sprintf("%d", getFieldSize(t))
							}
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

				sb.WriteString(fmt.Sprintf("| %d | `%s` | `%s` | %s | `%s` | `%s` | %s | %s |\n",
					i+1, origName, origType, origSize, optName, optType, optSize, change))
			}
			sb.WriteString("\n\n---\n\n")
		}
	}

	// 3. 正常跳过的结构体（仅详细模式显示）
	if r.level >= ReportLevelFull && len(skippedNormal) > 0 {
		sb.WriteString("## ⏭️ 正常跳过的结构体\n\n")
		sb.WriteString(fmt.Sprintf("**共 %d 个结构体被跳过**\n\n", len(skippedNormal)))

		for _, sr := range skippedNormal {
			sb.WriteString(fmt.Sprintf("### ✓ %s.%s\n\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("**原因**: %s\n\n", sr.SkipReason))
		}
	}

	// 4. 异常跳过的结构体
	if len(skippedError) > 0 {
		sb.WriteString("## ⚠️ 异常跳过的结构体\n\n")
		sb.WriteString(fmt.Sprintf("**共 %d 个结构体因异常被跳过**\n\n", len(skippedError)))

		for _, sr := range skippedError {
			sb.WriteString(fmt.Sprintf("### ⏭️ %s.%s\n\n", sr.PkgPath, sr.Name))
			sb.WriteString(fmt.Sprintf("**原因**: %s\n\n", sr.SkipReason))
		}
	}

	// 5. 未变化的结构体（详细模式下显示）
	if r.level >= ReportLevelFull && len(unchanged) > 0 {
		sb.WriteString("## ✔️ 未变化的结构体\n\n")
		sb.WriteString(fmt.Sprintf("**共 %d 个结构体无需优化**\n\n", len(unchanged)))

		sb.WriteString("| 结构体名 | 包路径 | 大小 |\n")
		sb.WriteString("|----------|--------|------|\n")
		for _, sr := range unchanged {
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %d 字节 |\n", sr.Name, sr.PkgPath, sr.OrigSize))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n*Generated by StructOptimizer*\n")

	return sb.String(), nil
}
