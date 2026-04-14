package reporter

import (
	"strings"
	"testing"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestNewReporter 测试报告生成器创建
func TestNewReporter(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		output     string
		level      ReportLevel
		wantFormat string
	}{
		{"md format", "md", "/tmp/report.md", ReportLevelFull, "md"},
		{"txt format", "txt", "/tmp/report.txt", ReportLevelSummary, "txt"},
		{"html format", "html", "/tmp/report.html", ReportLevelChanged, "html"},
		{"empty format defaults to md", "", "/tmp/report.md", ReportLevelFull, "md"},
		{"invalid format defaults to md", "json", "/tmp/report.md", ReportLevelFull, "md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReporter(tt.format, tt.output, tt.level)
			if r == nil {
				t.Fatal("NewReporter() returned nil")
			}
			if r.format != tt.wantFormat {
				t.Errorf("NewReporter() format = %v, want %v", r.format, tt.wantFormat)
			}
			if r.output != tt.output {
				t.Errorf("NewReporter() output = %v, want %v", r.output, tt.output)
			}
			if r.level != tt.level {
				t.Errorf("NewReporter() level = %v, want %v", r.level, tt.level)
			}
		})
	}
}

// TestReportLevels 测试报告级别
func TestReportLevels(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   3,
		OptimizedCount: 2,
		SkippedCount:   1,
		TotalSaved:     16,
	}

	levels := []ReportLevel{
		ReportLevelSummary,
		ReportLevelChanged,
		ReportLevelFull,
	}

	for _, level := range levels {
		t.Run(strings.ToLower([]string{"Summary", "Changed", "Full"}[level]), func(t *testing.T) {
			r := NewReporter("md", "", level)
			err := r.Generate(report)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
		})
	}
}

// TestGenerateMD 测试 Markdown 报告生成
func TestGenerateMD(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   3,
		OptimizedCount: 2,
		SkippedCount:   1,
		TotalSaved:     16,
		StructReports: []*optimizer.StructReport{
			{
				Name:       "TestStruct",
				PkgPath:    "example.com/pkg",
				File:       "/path/to/file.go",
				OrigSize:   32,
				OptSize:    24,
				Saved:      8,
				OrigFields: []string{"A", "B", "C"},
				OptFields:  []string{"B", "C", "A"},
				Skipped:    false,
			},
			{
				Name:       "SkippedStruct",
				PkgPath:    "example.com/pkg",
				File:       "/path/to/file.go",
				Skipped:    true,
				SkipReason: "单字段结构体",
			},
		},
	}

	r := NewReporter("md", "", ReportLevelFull)
	content, err := r.GenerateMD(report)
	if err != nil {
		t.Fatalf("GenerateMD() error = %v", err)
	}

	// 验证报告内容
	requiredStrings := []string{
		"# 🚀 StructOptimizer 优化报告",
		"## 📊 优化总览",
		"🔹 处理结构体总数",
		"✅ 优化的结构体",
		"⏭️ 跳过的结构体",
		"## ✏️ 调整的结构体",
		"TestStruct",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(content, required) {
			t.Errorf("GenerateMD() missing %q in output", required)
		}
	}
}

// TestGenerateTXT 测试 TXT 报告生成
func TestGenerateTXT(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   2,
		OptimizedCount: 1,
		SkippedCount:   1,
		TotalSaved:     8,
		StructReports: []*optimizer.StructReport{
			{
				Name:       "TestStruct",
				PkgPath:    "example.com/pkg",
				OrigSize:   32,
				OptSize:    24,
				Saved:      8,
				OrigFields: []string{"A", "B"},
				OptFields:  []string{"B", "A"},
				Skipped:    false,
			},
		},
	}

	r := NewReporter("txt", "", ReportLevelFull)
	content, err := r.GenerateTXT(report)
	if err != nil {
		t.Fatalf("GenerateTXT() error = %v", err)
	}

	// 验证报告内容
	requiredStrings := []string{
		"StructOptimizer 优化报告",
		"优化总览",
		"TestStruct",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(content, required) {
			t.Errorf("GenerateTXT() missing %q in output", required)
		}
	}
}

// TestGenerateHTML 测试 HTML 报告生成
func TestGenerateHTML(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   1,
		OptimizedCount: 1,
		TotalSaved:     8,
		StructReports: []*optimizer.StructReport{
			{
				Name:       "TestStruct",
				PkgPath:    "example.com/pkg",
				OrigSize:   32,
				OptSize:    24,
				Saved:      8,
				Skipped:    false,
			},
		},
	}

	r := NewReporter("html", "", ReportLevelFull)
	content, err := r.GenerateHTML(report)
	if err != nil {
		t.Fatalf("GenerateHTML() error = %v", err)
	}

	// 验证 HTML 结构
	requiredStrings := []string{
		"<!DOCTYPE html>",
		"<html",
		"<style>",
		"StructOptimizer",
		"优化报告",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(content, required) {
			t.Errorf("GenerateHTML() missing %q in output", required)
		}
	}
}

// TestGenerateReport 测试完整报告生成流程
func TestGenerateReport(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   5,
		OptimizedCount: 3,
		SkippedCount:   2,
		TotalSaved:     24,
		StructReports: []*optimizer.StructReport{
			{
				Name:       "OptimizedStruct1",
				PkgPath:    "example.com/pkg",
				OrigSize:   32,
				OptSize:    24,
				Saved:      8,
				Skipped:    false,
			},
			{
				Name:       "OptimizedStruct2",
				PkgPath:    "example.com/pkg",
				OrigSize:   48,
				OptSize:    40,
				Saved:      8,
				Skipped:    false,
			},
			{
				Name:       "OptimizedStruct3",
				PkgPath:    "example.com/pkg",
				OrigSize:   24,
				OptSize:    16,
				Saved:      8,
				Skipped:    false,
			},
			{
				Name:       "SkippedStruct1",
				PkgPath:    "example.com/pkg",
				Skipped:    true,
				SkipReason: "单字段结构体",
			},
			{
				Name:       "SkippedStruct2",
				PkgPath:    "example.com/pkg",
				Skipped:    true,
				SkipReason: "空结构体",
			},
		},
	}

	formats := []string{"md", "txt", "html"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			r := NewReporter(format, "", ReportLevelFull)
			err := r.Generate(report)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", format, err)
			}
		})
	}
}

// TestReportTime 测试报告时间戳
func TestReportTime(t *testing.T) {
	report := &optimizer.Report{}
	r := NewReporter("md", "", ReportLevelFull)
	content, err := r.GenerateMD(report)
	if err != nil {
		t.Fatalf("GenerateMD() error = %v", err)
	}

	// 验证时间戳格式
	if !strings.Contains(content, "生成时间：") {
		t.Error("GenerateMD() missing timestamp")
	}

	// 验证时间格式 (YYYY-MM-DD HH:MM:SS)
	now := time.Now().Format("2006-01-02 15:04:05")
	if !strings.Contains(content, now[:10]) { // 只检查日期部分
		t.Error("GenerateMD() timestamp format incorrect")
	}
}

// TestReportLevelSummary 测试摘要级别报告
func TestReportLevelSummary(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   10,
		OptimizedCount: 5,
		SkippedCount:   5,
		TotalSaved:     50,
	}

	r := NewReporter("md", "", ReportLevelSummary)
	content, err := r.GenerateMD(report)
	if err != nil {
		t.Fatalf("GenerateMD() error = %v", err)
	}

	// 摘要级别应该包含统计信息
	if !strings.Contains(content, "处理结构体总数") {
		t.Error("Summary level missing total structs count")
	}
	if !strings.Contains(content, "优化的结构体") {
		t.Error("Summary level missing optimized count")
	}
	if !strings.Contains(content, "跳过的结构体") {
		t.Error("Summary level missing skipped count")
	}
	if !strings.Contains(content, "节省内存") {
		t.Error("Summary level missing saved bytes")
	}
}

// TestReportLevelChanged 测试变更级别报告
func TestReportLevelChanged(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   10,
		OptimizedCount: 5,
		SkippedCount:   5,
		TotalSaved:     50,
		StructReports: []*optimizer.StructReport{
			{
				Name:       "ChangedStruct",
				PkgPath:    "example.com/pkg",
				OrigSize:   32,
				OptSize:    24,
				Saved:      8,
				Skipped:    false,
			},
			{
				Name:       "UnchangedStruct",
				PkgPath:    "example.com/pkg",
				OrigSize:   24,
				OptSize:    24,
				Saved:      0,
				Skipped:    false,
			},
		},
	}

	r := NewReporter("md", "", ReportLevelChanged)
	content, err := r.GenerateMD(report)
	if err != nil {
		t.Fatalf("GenerateMD() error = %v", err)
	}

	// 变更级别应该包含有变化的结构体
	if !strings.Contains(content, "ChangedStruct") {
		t.Error("Changed level missing changed struct")
	}
}

// TestEmptyReport 测试空报告
func TestEmptyReport(t *testing.T) {
	report := &optimizer.Report{}

	r := NewReporter("md", "", ReportLevelFull)
	content, err := r.GenerateMD(report)
	if err != nil {
		t.Fatalf("GenerateMD() error = %v", err)
	}

	// 空报告应该仍然有效
	if content == "" {
		t.Error("Empty report generated empty content")
	}
}

// TestReportWithZeroSavings 测试零节省报告
func TestReportWithZeroSavings(t *testing.T) {
	report := &optimizer.Report{
		TotalStructs:   2,
		OptimizedCount: 2,
		SkippedCount:   0,
		TotalSaved:     0,
		StructReports: []*optimizer.StructReport{
			{
				Name:       "NoChange1",
				PkgPath:    "example.com/pkg",
				OrigSize:   24,
				OptSize:    24,
				Saved:      0,
				Skipped:    false,
			},
			{
				Name:       "NoChange2",
				PkgPath:    "example.com/pkg",
				OrigSize:   32,
				OptSize:    32,
				Saved:      0,
				Skipped:    false,
			},
		},
	}

	r := NewReporter("md", "", ReportLevelFull)
	content, err := r.GenerateMD(report)
	if err != nil {
		t.Fatalf("GenerateMD() error = %v", err)
	}

	// 验证零节省情况
	if !strings.Contains(content, "节省内存") {
		t.Error("Report missing saved bytes section")
	}
}
