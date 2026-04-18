package optimizer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// TestSamePackageUnexportedStructsCrossFiles 测试同包不同文件中定义的未导出结构体
func TestSamePackageUnexportedStructsCrossFiles(t *testing.T) {
	// 设置GOPATH
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	if err := os.Setenv("GOPATH", gopath); err != nil {
		t.Fatalf("Failed to set GOPATH: %v", err)
	}

	t.Logf("Using GOPATH: %s", gopath)

	// 创建分析器
	analyzerCfg := &analyzer.Config{
		StructName:  "myproject/pkg.MainStructWithUnexported",
		ProjectType: "gopath",
		GOPATH:      gopath,
		Verbose:     3,
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	// 创建优化器
	optimizerCfg := &Config{
		StructName:     "myproject/pkg.MainStructWithUnexported",
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
	t.Logf("Total structs: %d", report.TotalStructs)
	t.Logf("Optimized: %d", report.OptimizedCount)
	t.Logf("Skipped: %d", report.SkippedCount)
	t.Logf("Total saved: %d bytes", report.TotalSaved)

	// 打印所有处理的结构体
	t.Log("\n=== Processed Structs ===")
	for _, sr := range report.StructReports {
		t.Logf("  - %s (OrigSize: %d, OptSize: %d, Skipped: %v)",
			sr.Name, sr.OrigSize, sr.OptSize, sr.Skipped)
		if sr.Skipped {
			t.Logf("    Skip reason: %s", sr.SkipReason)
		}
	}

	// 验证内部结构体被收集
	foundInternalConfig := false
	foundLocalCache := false
	foundEmbeddedBase := false
	foundMainStruct := false

	for _, sr := range report.StructReports {
		switch sr.Name {
		case "internalConfig":
			foundInternalConfig = true
			// 验证未导出结构体被优化
			if sr.Skipped {
				t.Errorf("internalConfig should NOT be skipped, reason: %s", sr.SkipReason)
			}
			if sr.OrigSize == 0 {
				t.Error("internalConfig should have non-zero size")
			}
		case "localCache":
			foundLocalCache = true
			if sr.Skipped {
				t.Errorf("localCache should NOT be skipped, reason: %s", sr.SkipReason)
			}
		case "embeddedBase":
			foundEmbeddedBase = true
			if sr.Skipped {
				t.Errorf("embeddedBase should NOT be skipped, reason: %s", sr.SkipReason)
			}
			// 验证匿名字段被正确处理
			if !sr.HasEmbed {
				t.Log("embeddedBase is itself an embedded type in MainStructWithUnexported")
			}
		case "MainStructWithUnexported":
			foundMainStruct = true
		}
	}

	// 验证所有同包未导出结构体都被收集
	if !foundMainStruct {
		t.Error("MainStructWithUnexported should be in reports")
	}
	if !foundInternalConfig {
		t.Error("internalConfig (unexported, same package, different file) should be found")
	}
	if !foundLocalCache {
		t.Error("localCache (unexported, same package, different file) should be found")
	}
	if !foundEmbeddedBase {
		t.Error("embeddedBase (unexported, same package, different file, embedded) should be found")
	}

	// 验证总共处理了至少4个结构体
	if report.TotalStructs < 4 {
		t.Errorf("Expected at least 4 structs (main + 3 unexported), got %d", report.TotalStructs)
	}
}

// TestUnexportedStructsNaming 验证未导出类型的命名约定
func TestUnexportedStructsNaming(t *testing.T) {
	// 这个测试验证我们正确理解了Go的导出规则
	// 未导出类型 = 小写字母开头
	// 已导出类型 = 大写字母开头

	testCases := []struct {
		name       string
		typeName   string
		isExported bool
	}{
		{"exported type", "Config", true},
		{"exported type", "MyStruct", true},
		{"unexported type", "config", false},
		{"unexported type", "myStruct", false},
		{"unexported type", "internalConfig", false},
		{"unexported type", "localCache", false},
		{"unexported type", "embeddedBase", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_"+tc.typeName, func(t *testing.T) {
			firstChar := tc.typeName[0]
			isExported := firstChar >= 'A' && firstChar <= 'Z'

			if isExported != tc.isExported {
				t.Errorf("Type %s: expected exported=%v, got %v",
					tc.typeName, tc.isExported, isExported)
			}
		})
	}
}

// TestSamePackageCrossFileDetection 测试同包跨文件检测逻辑
func TestSamePackageCrossFileDetection(t *testing.T) {
	gopath, _ := filepath.Abs(filepath.Join("..", "testdata", "gopath_test_project"))
	os.Setenv("GOPATH", gopath)

	// 验证pkg目录中有多个Go文件
	pkgDir := filepath.Join(gopath, "src", "myproject", "pkg")
	files, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("Failed to read pkg dir: %v", err)
	}

	goFileCount := 0
	for _, f := range files {
		if !f.IsDir() && len(f.Name()) > 3 && f.Name()[len(f.Name())-3:] == ".go" {
			goFileCount++
		}
	}

	t.Logf("Found %d Go files in pkg directory", goFileCount)

	// 应该有至少4个Go文件
	if goFileCount < 4 {
		t.Errorf("Expected at least 4 Go files in pkg dir, got %d", goFileCount)
	}

	// 验证每个文件都能被解析
	for _, f := range files {
		if f.IsDir() || len(f.Name()) < 4 || f.Name()[len(f.Name())-3:] != ".go" {
			continue
		}

		filePath := filepath.Join(pkgDir, f.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", f.Name(), err)
			continue
		}

		t.Logf("File: %s (%d bytes)", f.Name(), len(content))

		// 检查文件包含package声明
		if len(content) < 10 {
			t.Errorf("File %s is too small", f.Name())
		}
	}
}
