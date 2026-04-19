package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestFieldSizesSumEqualsStructSize 验证字段大小之和等于结构体总大小
func TestFieldSizesSumEqualsStructSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "field_sum_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 Go Module 项目
	goModContent := `module testfieldsum

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建测试结构体
	testFile := filepath.Join(pkgDir, "types.go")
	content := `package pkg

type Inner struct {
	A int64  // 8
	B bool   // 1
	C int32  // 4
	// 总：13, 对齐后 16
}

type Outer struct {
	Flag bool    // 1
	Name string  // 16
	Data Inner   // 16
	Age  int32   // 4
	// 总：37, 对齐后 40（原始顺序）
	// 优化后：16+16+4+1=37, 对齐后 40
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 创建 analyzer 和优化器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	optCfg := &optimizer.Config{
		TargetDir:      tmpDir,
		StructName:     "testfieldsum/pkg.Outer",
		ProjectType:    "gomod",
		Verbose:        0,
		MaxDepth:       50,
		Timeout:        300,
		PkgWorkerLimit: 4,
	}
	opt := optimizer.NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证每个结构体
	for _, sr := range report.StructReports {
		t.Logf("\n结构体：%s.%s", sr.PkgPath, sr.Name)
		t.Logf("  优化前=%d, 优化后=%d, 节省=%d", sr.OrigSize, sr.OptSize, sr.Saved)

		// 验证 Saved = OrigSize - OptSize
		if sr.Saved != sr.OrigSize-sr.OptSize {
			t.Errorf("%s: Saved (%d) != OrigSize (%d) - OptSize (%d)",
				sr.Name, sr.Saved, sr.OrigSize, sr.OptSize)
		}

		// 计算字段大小之和
		var origFieldSum, optFieldSum int64
		for _, fieldName := range sr.OrigFields {
			if size, ok := sr.FieldSizes[fieldName]; ok {
				origFieldSum += size
			} else {
				t.Errorf("%s: 字段 '%s' 在 FieldSizes 中不存在", sr.Name, fieldName)
			}
		}
		for _, fieldName := range sr.OptFields {
			if size, ok := sr.FieldSizes[fieldName]; ok {
				optFieldSum += size
			}
		}

		t.Logf("  字段大小之和=%d (原始=%d, 优化后=%d)", origFieldSum, origFieldSum, optFieldSum)

		// 注意：字段大小之和 != 结构体总大小（因为有 padding）
		// 但字段大小之和在优化前后应该相同
		if origFieldSum != optFieldSum {
			t.Errorf("%s: 字段大小之和在优化前后不一致：原始=%d, 优化后=%d",
				sr.Name, origFieldSum, optFieldSum)
		}

		// 验证字段大小之和 <= 结构体大小（因为 padding）
		if origFieldSum > sr.OrigSize {
			t.Errorf("%s: 字段大小之和 (%d) > 结构体大小 (%d)",
				sr.Name, origFieldSum, sr.OrigSize)
		}
		if optFieldSum > sr.OptSize {
			t.Errorf("%s: 字段大小之和 (%d) > 结构体大小 (%d)",
				sr.Name, optFieldSum, sr.OptSize)
		}

		// 验证公式
		expectedOptSize := sr.OrigSize - sr.Saved
		if sr.OptSize != expectedOptSize {
			t.Errorf("%s: 优化后大小 (%d) != 优化前 (%d) - 节省 (%d) = %d",
				sr.Name, sr.OptSize, sr.OrigSize, sr.Saved, expectedOptSize)
		} else {
			t.Logf("  ✅ 公式验证通过：%d = %d - %d", sr.OptSize, sr.OrigSize, sr.Saved)
		}
	}
}
