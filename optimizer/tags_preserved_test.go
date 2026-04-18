package optimizer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
	"github.com/gamelife1314/structoptimizer/writer"
)

// TestStructFieldTagsPreserved 测试结构体字段标签在优化后被保留
func TestStructFieldTagsPreserved(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "structoptimizer-tags-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件，包含带标签的字段
	// Data 结构体：A(1), B(8), C(1), D(8) = 32 字节（有 padding）
	// 优化后：B(8), D(8), A(1), C(1) = 24 字节
	// 故意使用未优化的顺序来触发优化
	testContent := `package test

type Data struct {
	A bool   ` + "`json:\"a\"`" + `
	B int64  ` + "`json:\"b\"`" + `
	C bool   ` + "`json:\"c\"`" + `
	D int64  ` + "`json:\"d\"`" + `
}
`
	testFile := filepath.Join(tmpDir, "config.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 创建 go.mod 文件
	goModContent := `module example.com/test

go 1.21
`
	goModFile := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}

	// 创建分析器
	analyzerCfg := &analyzer.Config{
		StructName:  "example.com/test.Data",
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Verbose:     0, // 关闭日志输出
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	// 创建优化器
	optimizerCfg := &optimizer.Config{
		StructName:  "example.com/test.Data",
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Write:       true,
		Backup:      false,
		Verbose:     0,
		Timeout:     60, // 60 秒超时
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证优化结果
	if report.OptimizedCount == 0 {
		t.Skip("结构体无需优化，跳过标签验证")
	}

	t.Logf("优化统计：处理=%d, 优化=%d, 跳过=%d, 节省=%d 字节",
		report.TotalStructs, report.OptimizedCount, report.SkippedCount, report.TotalSaved)

	// 写入优化后的文件
	writerCfg := &writer.Config{
		Backup:  false,
		Verbose: 0,
	}
	w := writer.NewSourceWriter(writerCfg)

	optimized := opt.GetOptimized()
	if err := w.WriteFiles(optimized); err != nil {
		t.Fatalf("写入文件失败：%v", err)
	}

	// 读取优化后的文件内容
	optimizedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("读取优化后文件失败：%v", err)
	}

	contentStr := string(optimizedContent)
	t.Logf("优化后文件内容:\n%s", contentStr)

	// 验证所有标签都存在
	expectedTags := []string{
		`json:"a"`,
		`json:"b"`,
		`json:"c"`,
		`json:"d"`,
	}

	for _, tag := range expectedTags {
		if !strings.Contains(contentStr, tag) {
			t.Errorf("优化后文件中未找到标签：%s", tag)
		}
	}

	// 验证所有字段名都存在
	expectedFields := []string{"A", "B", "C", "D"}
	for _, field := range expectedFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("优化后文件中未找到字段：%s", field)
		}
	}

	// 验证标签数量没有变化（每个字段一个标签）
	tagCount := strings.Count(contentStr, "`")
	expectedTagCount := 8 // 4 个字段 × 2 个反引号
	if tagCount != expectedTagCount {
		t.Errorf("标签数量不匹配：期望 %d, 实际 %d", expectedTagCount, tagCount)
	}

	t.Log("✅ 所有字段标签在优化后被正确保留")
}
