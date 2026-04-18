package optimizer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// TestGOPATHProjectWithVendor 测试GOPATH项目+vendor目录的完整场景
func TestGOPATHProjectWithVendor(t *testing.T) {
	// 设置GOPATH - 使用绝对路径（从optimizer目录向上2级到项目根目录）
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	if err := os.Setenv("GOPATH", gopath); err != nil {
		t.Fatalf("Failed to set GOPATH: %v", err)
	}
	t.Logf("Using GOPATH: %s", gopath)

	// 项目根目录（GOPATH模式下可以省略）
	targetDir := ""

	// 创建分析器
	analyzerCfg := &analyzer.Config{
		TargetDir:   targetDir,
		StructName:  "myproject/pkg.MainStruct",
		ProjectType: "gopath",
		GOPATH:      gopath,
		Verbose:     3,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	// 创建优化器
	optimizerCfg := &Config{
		TargetDir:      targetDir,
		StructName:     "myproject/pkg.MainStruct",
		ProjectType:    "gopath",
		GOPATH:         gopath,
		Verbose:        3,
		MaxDepth:       50,
		Timeout:        300,
		PkgWorkerLimit: 4,
	}
	opt := NewOptimizer(optimizerCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 验证结果
	if report == nil {
		t.Fatal("Report should not be nil")
	}

	// 打印报告用于调试
	t.Logf("Total structs: %d", report.TotalStructs)
	t.Logf("Optimized: %d", report.OptimizedCount)
	t.Logf("Skipped: %d", report.SkippedCount)
	t.Logf("Total saved: %d bytes", report.TotalSaved)

	// 验证处理了足够的结构体
	if report.TotalStructs < 5 {
		t.Errorf("Expected at least 5 structs, got %d", report.TotalStructs)
	}
}

// TestGOPATHEmbeddedFieldDetection 测试GOPATH项目中匿名字段识别
func TestGOPATHEmbeddedFieldDetection(t *testing.T) {
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	os.Setenv("GOPATH", gopath)

	analyzerCfg := &analyzer.Config{
		StructName:  "myproject/pkg.MainStruct",
		ProjectType: "gopath",
		GOPATH:      gopath,
		Verbose:     3,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &Config{
		StructName:     "myproject/pkg.MainStruct",
		ProjectType:    "gopath",
		GOPATH:         gopath,
		Verbose:        3,
		MaxDepth:       50,
		Timeout:        300,
		PkgWorkerLimit: 4,
	}
	opt := NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 查找MainStruct的报告
	var mainReport *StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "MainStruct" {
			mainReport = sr
			break
		}
	}

	if mainReport == nil {
		t.Fatal("MainStruct report not found")
	}

	// 验证HasEmbed字段
	if !mainReport.HasEmbed {
		t.Error("MainStruct should have embedded field (HasEmbed should be true)")
	}

	// 验证EmbeddedType在字段列表中
	foundEmbed := false
	for _, field := range mainReport.OrigFields {
		if field == "EmbeddedType" {
			foundEmbed = true
			break
		}
	}
	if !foundEmbed {
		t.Error("EmbeddedType should be in original fields list")
	}
}

// TestGOPATHUnexportedStructCrossFile 测试GOPATH项目中跨文件未导出结构体
func TestGOPATHUnexportedStructCrossFile(t *testing.T) {
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	os.Setenv("GOPATH", gopath)

	analyzerCfg := &analyzer.Config{
		StructName:  "myproject/pkg.MainStruct",
		ProjectType: "gopath",
		GOPATH:      gopath,
		Verbose:     3,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &Config{
		StructName:     "myproject/pkg.MainStruct",
		ProjectType:    "gopath",
		GOPATH:         gopath,
		Verbose:        3,
		MaxDepth:       50,
		Timeout:        300,
		PkgWorkerLimit: 4,
	}
	opt := NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 验证UnexportedModel被正确处理
	foundUnexportedModel := false
	for _, sr := range report.StructReports {
		if sr.Name == "UnexportedModel" {
			foundUnexportedModel = true
			// 验证它被优化了（不是跳过）
			if sr.Skipped {
				t.Errorf("UnexportedModel should not be skipped, reason: %s", sr.SkipReason)
			}
			// 验证大小计算正确
			if sr.OrigSize == 0 {
				t.Error("UnexportedModel should have non-zero size")
			}
			break
		}
	}

	if !foundUnexportedModel {
		t.Error("UnexportedModel should be found in reports (cross-file reference)")
	}
}

// TestGOPATHTypeAliasSize 测试GOPATH项目中类型别名大小识别
func TestGOPATHTypeAliasSize(t *testing.T) {
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	os.Setenv("GOPATH", gopath)

	analyzerCfg := &analyzer.Config{
		StructName:  "myproject/pkg.MainStruct",
		ProjectType: "gopath",
		GOPATH:      gopath,
		Verbose:     3,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &Config{
		StructName:     "myproject/pkg.MainStruct",
		ProjectType:    "gopath",
		GOPATH:         gopath,
		Verbose:        3,
		MaxDepth:       50,
		Timeout:        300,
		PkgWorkerLimit: 4,
	}
	opt := NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 查找MainStruct
	var mainReport *StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "MainStruct" {
			mainReport = sr
			break
		}
	}

	if mainReport == nil {
		t.Fatal("MainStruct report not found")
	}

	// 验证CustomInt类型字段存在
	foundCustomInt := false
	for fieldName, typeName := range mainReport.FieldTypes {
		if fieldName == "TypeAlias" && typeName == "CustomInt" {
			foundCustomInt = true
			break
		}
	}

	if !foundCustomInt {
		t.Error("TypeAlias field with CustomInt type should be in field types")
	}
}

// TestGOPATHSkipByMethods 测试GOPATH项目中skip-by-methods功能
func TestGOPATHSkipByMethods(t *testing.T) {
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	os.Setenv("GOPATH", gopath)

	analyzerCfg := &analyzer.Config{
		StructName:    "myproject/pkg.MainStruct",
		ProjectType:   "gopath",
		GOPATH:        gopath,
		Verbose:       3,
		SkipByMethods: []string{"Encode", "Decode"},
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &Config{
		StructName:      "myproject/pkg.MainStruct",
		ProjectType:     "gopath",
		GOPATH:          gopath,
		Verbose:         3,
		MaxDepth:        50,
		Timeout:         300,
		PkgWorkerLimit:  4,
		SkipByMethods:   []string{"Encode", "Decode"},
	}
	opt := NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 查找HasMethods的报告
	var hasMethodsReport *StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "HasMethods" {
			hasMethodsReport = sr
			break
		}
	}

	if hasMethodsReport == nil {
		t.Fatal("HasMethods report should be found")
	}

	// 验证HasMethods被跳过
	if !hasMethodsReport.Skipped {
		t.Error("HasMethods should be skipped due to having Encode/Decode methods")
	}

	// 验证跳过原因正确
	if hasMethodsReport.SkipReason == "" {
		t.Error("SkipReason should not be empty for skipped struct")
	}

	t.Logf("HasMethods skip reason: %s", hasMethodsReport.SkipReason)
}

// TestGOPATHVendorPackageSkipped 测试vendor中的包被正确跳过
func TestGOPATHVendorPackageSkipped(t *testing.T) {
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	os.Setenv("GOPATH", gopath)

	analyzerCfg := &analyzer.Config{
		StructName:  "myproject/pkg.MainStruct",
		ProjectType: "gopath",
		GOPATH:      gopath,
		Verbose:     3,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &Config{
		StructName:     "myproject/pkg.MainStruct",
		ProjectType:    "gopath",
		GOPATH:         gopath,
		Verbose:        3,
		MaxDepth:       50,
		Timeout:        300,
		PkgWorkerLimit: 4,
	}
	opt := NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 验证ExternalStruct不在报告中（因为vendor中的包应该被跳过）
	for _, sr := range report.StructReports {
		if sr.Name == "ExternalStruct" {
			t.Error("ExternalStruct from vendor should NOT appear in reports (should be skipped during collection)")
			break
		}
	}

	// 验证跳过了vendor包中的结构体
	// 注意：vendor中的类型不会出现在StructReports中，因为它们在收集阶段就被跳过了
	t.Logf("Total structs processed: %d", report.TotalStructs)
	t.Logf("Optimized: %d, Skipped: %d", report.OptimizedCount, report.SkippedCount)
	
	// 验证收集到的结构体数量（应该不包含ExternalStruct）
	if report.TotalStructs > 10 {
		t.Errorf("Expected <=10 structs (excluding vendor), got %d", report.TotalStructs)
	}
}

// TestGOPATHNoMethodsStructOptimized 测试没有方法的结构体被优化
func TestGOPATHNoMethodsStructOptimized(t *testing.T) {
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	os.Setenv("GOPATH", gopath)

	analyzerCfg := &analyzer.Config{
		StructName:    "myproject/pkg.MainStruct",
		ProjectType:   "gopath",
		GOPATH:        gopath,
		Verbose:       3,
		SkipByMethods: []string{"Encode"},
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &Config{
		StructName:      "myproject/pkg.MainStruct",
		ProjectType:     "gopath",
		GOPATH:          gopath,
		Verbose:         3,
		MaxDepth:        50,
		Timeout:         300,
		PkgWorkerLimit:  4,
		SkipByMethods:   []string{"Encode"},
	}
	opt := NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 查找NoMethods的报告
	var noMethodsReport *StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "NoMethods" {
			noMethodsReport = sr
			break
		}
	}

	if noMethodsReport == nil {
		t.Fatal("NoMethods report should be found")
	}

	// 验证NoMethods未被跳过（因为它没有Encode方法）
	if noMethodsReport.Skipped {
		t.Errorf("NoMethods should NOT be skipped, reason: %s", noMethodsReport.SkipReason)
	}
}

// getTestDataDir 获取testdata目录路径
func getTestDataDir() string {
	// 假设测试运行在项目根目录
	return "testdata"
}
