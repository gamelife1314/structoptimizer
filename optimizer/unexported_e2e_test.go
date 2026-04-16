package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestUnexportedStructEndToEnd 测试未导出结构体的端到端处理
func TestUnexportedStructEndToEnd(t *testing.T) {
	// 获取测试数据目录
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败：%v", err)
	}

	testDataDir := filepath.Join(cwd, "..", "testdata", "unexported")

	// 确保测试数据存在
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Skipf("测试数据目录不存在：%s", testDataDir)
	}

	// 创建分析器
	anlzCfg := &analyzer.Config{
		TargetDir:   filepath.Join(cwd, ".."),
		Package:     "github.com/gamelife1314/structoptimizer/testdata/unexported",
		ProjectType: "gomod",
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   filepath.Join(cwd, ".."),
		Package:     "github.com/gamelife1314/structoptimizer/testdata/unexported",
		ProjectType: "gomod",
		Verbose:     0,
		Timeout:     60, // 60 秒超时
	}
	opt := optimizer.NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证结果
	if report.TotalStructs == 0 {
		t.Error("期望处理至少一个结构体，但结果为 0")
	}

	// 验证未导出结构体被处理
	foundUnexported := false
	for _, sr := range report.StructReports {
		if sr.Name == "badInner" || sr.Name == "innerStruct" || sr.Name == "unexportStruct" {
			foundUnexported = true
			break
		}
	}

	if !foundUnexported {
		t.Error("期望找到未导出结构体（badInner、innerStruct 或 unexportStruct），但未找到")
	}

	// 验证有结构体被优化
	if report.OptimizedCount == 0 {
		t.Error("期望至少有一个结构体被优化")
	}
}

// TestUnexportedStructSizeCalculation 测试未导出结构体大小计算
func TestUnexportedStructSizeCalculation(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败：%v", err)
	}

	testDataDir := filepath.Join(cwd, "..", "testdata", "unexported")

	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Skipf("测试数据目录不存在：%s", testDataDir)
	}

	anlzCfg := &analyzer.Config{
		TargetDir:   filepath.Join(cwd, ".."),
		Package:     "github.com/gamelife1314/structoptimizer/testdata/unexported",
		ProjectType: "gomod",
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	optCfg := &optimizer.Config{
		TargetDir:   filepath.Join(cwd, ".."),
		Package:     "github.com/gamelife1314/structoptimizer/testdata/unexported",
		ProjectType: "gomod",
		Verbose:     0,
		Timeout:     60, // 60 秒超时
	}
	opt := optimizer.NewOptimizer(optCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证 MixedOuter 的大小计算正确（应该包含嵌套结构体的准确大小）
	for _, sr := range report.StructReports {
		if sr.Name == "MixedOuter" {
			// MixedOuter 包含多个嵌套结构体，大小应该大于 40 字节
			if sr.OrigSize <= 40 {
				t.Errorf("MixedOuter 的大小 %d 字节，期望大于 40 字节（嵌套结构体大小未正确计算）", sr.OrigSize)
			}
		}
		if sr.Name == "BadUnexportWithBadInner" {
			// BadUnexportWithBadInner 包含 badInner（24 字节），总大小应该大于 24 字节
			if sr.OrigSize <= 24 {
				t.Errorf("BadUnexportWithBadInner 的大小 %d 字节，期望大于 24 字节", sr.OrigSize)
			}
		}
	}
}
