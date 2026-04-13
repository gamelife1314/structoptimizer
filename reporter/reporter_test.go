package reporter

import (
	"strings"
	"testing"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

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

	r := NewReporter("md", "")
	content, err := r.GenerateMD(report)
	if err != nil {
		t.Fatalf("GenerateMD() error = %v", err)
	}

	// 验证报告内容
	if !strings.Contains(content, "# 🚀 StructOptimizer 优化报告") {
		t.Error("GenerateMD() missing header")
	}
	if !strings.Contains(content, "📊 摘要") {
		t.Error("GenerateMD() missing summary section")
	}
	if !strings.Contains(content, "🔧 优化详情") {
		t.Error("GenerateMD() missing details section")
	}
	if !strings.Contains(content, "⏭️ 跳过的结构体") {
		t.Error("GenerateMD() missing skipped section")
	}
	if !strings.Contains(content, "TestStruct") {
		t.Error("GenerateMD() missing struct name")
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

	r := NewReporter("txt", "")
	content, err := r.GenerateTXT(report)
	if err != nil {
		t.Fatalf("GenerateTXT() error = %v", err)
	}

	// 验证报告内容
	if !strings.Contains(content, "StructOptimizer 优化报告") {
		t.Error("GenerateTXT() missing header")
	}
	if !strings.Contains(content, "摘要") {
		t.Error("GenerateTXT() missing summary")
	}
	if !strings.Contains(content, "优化详情") {
		t.Error("GenerateTXT() missing details")
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

	r := NewReporter("html", "")
	content, err := r.GenerateHTML(report)
	if err != nil {
		t.Fatalf("GenerateHTML() error = %v", err)
	}

	// 验证 HTML 结构
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("GenerateHTML() missing DOCTYPE")
	}
	if !strings.Contains(content, "<html") {
		t.Error("GenerateHTML() missing html tag")
	}
	if !strings.Contains(content, "<style>") {
		t.Error("GenerateHTML() missing styles")
	}
	if !strings.Contains(content, "StructOptimizer") {
		t.Error("GenerateHTML() missing title")
	}
}

// TestNewReporter 测试报告生成器创建
func TestNewReporter(t *testing.T) {
	tests := []struct {
		format     string
		wantFormat string
	}{
		{"md", "md"},
		{"txt", "txt"},
		{"html", "html"},
		{"", "md"},      // 默认格式
		{"json", "md"},  // 无效格式使用默认
		{"xml", "md"},   // 无效格式使用默认
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			r := NewReporter(tt.format, "/tmp/report")
			if r.format != tt.wantFormat {
				t.Errorf("NewReporter() format = %v, want %v", r.format, tt.wantFormat)
			}
			if r.output != "/tmp/report" {
				t.Errorf("NewReporter() output = %v, want /tmp/report", r.output)
			}
		})
	}
}

// TestReportGeneration 测试完整报告生成流程
func TestReportGeneration(t *testing.T) {
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
			r := NewReporter(format, "")
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
	r := NewReporter("md", "")
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
