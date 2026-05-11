package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestComplexNestedStructWithUnsafeVerification 测试复杂嵌套结构体大小计算
// 使用 unsafe.Sizeof() 验证大小计算的准确性
func TestComplexNestedStructWithUnsafeVerification(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "complex_nested_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 Go Module 项目
	goModContent := `module testcomplex

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建复杂的嵌套结构体定义
	testFile := filepath.Join(pkgDir, "types.go")
	content := `package pkg

// Level3 最深层结构体
type Level3 struct {
	A int64   // 8 字节
	B bool    // 1 字节
	C int32   // 4 字节
	// 原始：8+1+4=13, 对齐后 16 字节
}

// Level2 中间层结构体
type Level2 struct {
	X    float64 // 8 字节
	Y    Level3  // 16 字节（嵌套）
	Z    bool    // 1 字节
	Name string  // 16 字节
	// 原始：8+16+1+16=41, 考虑对齐后 48 字节
}

// Level1 外层结构体
type Level1 struct {
	ID       int64   // 8 字节
	Data     Level2  // 48 字节（嵌套）
	Flag     bool    // 1 字节
	Tags     []string // 24 字节（slice）
	Metadata string  // 16 字节
	Count    int32   // 4 字节
	Active   bool    // 1 字节
	// 原始顺序：8+48+1+24+16+4+1=102, 考虑对齐后 104 字节
	// 优化后应该能减少 padding
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 使用 unsafe.Sizeof 计算实际大小
	// 注意：这些计算基于当前 Go 编译器的实现
	type Level3 struct {
		A int64
		B bool
		C int32
	}

	type Level2 struct {
		X    float64
		Y    Level3
		Z    bool
		Name string
	}

	type Level1 struct {
		ID       int64
		Data     Level2
		Flag     bool
		Tags     []string
		Metadata string
		Count    int32
		Active   bool
	}

	expectedLevel3Size := int64(unsafe.Sizeof(Level3{}))
	expectedLevel2Size := int64(unsafe.Sizeof(Level2{}))
	expectedLevel1Size := int64(unsafe.Sizeof(Level1{}))

	t.Logf("unsafe.Sizeof 计算结果:")
	t.Logf("  Level3: %d 字节", expectedLevel3Size)
	t.Logf("  Level2: %d 字节", expectedLevel2Size)
	t.Logf("  Level1: %d 字节", expectedLevel1Size)

	// 创建 analyzer 和优化器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	optCfg := &optimizer.Config{
		TargetDir:      tmpDir,
		StructName:     "testcomplex/pkg.Level1",
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

	// 查找各层结构体的报告
	reports := make(map[string]*optimizer.StructReport)
	for _, sr := range report.StructReports {
		reports[sr.Name] = sr
		t.Logf("结构体 %s: 优化前=%d, 优化后=%d, 节省=%d",
			sr.Name, sr.OrigSize, sr.OptSize, sr.Saved)
	}

	// 验证 Level3 大小
	if level3Report, ok := reports["Level3"]; ok {
		if level3Report.OrigSize != expectedLevel3Size {
			t.Errorf("Level3 大小错误：期望 %d (unsafe.Sizeof), 得到 %d (optimizer)",
				expectedLevel3Size, level3Report.OrigSize)
		} else {
			t.Logf("✅ Level3 大小正确：%d 字节 (与 unsafe.Sizeof 一致)", level3Report.OrigSize)
		}
	}

	// 验证 Level2 大小
	if level2Report, ok := reports["Level2"]; ok {
		if level2Report.OrigSize != expectedLevel2Size {
			t.Errorf("Level2 大小错误：期望 %d (unsafe.Sizeof), 得到 %d (optimizer)",
				expectedLevel2Size, level2Report.OrigSize)
		} else {
			t.Logf("✅ Level2 大小正确：%d 字节 (与 unsafe.Sizeof 一致)", level2Report.OrigSize)
		}

		// 验证嵌套的 Level3 字段大小
		if ySize, ok := level2Report.FieldSizes["Y"]; ok {
			if ySize != expectedLevel3Size {
				t.Errorf("Level2.Y (Level3) 字段大小错误：期望 %d, 得到 %d",
					expectedLevel3Size, ySize)
			} else {
				t.Logf("✅ Level2.Y (Level3) 字段大小正确：%d 字节", ySize)
			}
		}
	}

	// 验证 Level1 大小
	if level1Report, ok := reports["Level1"]; ok {
		if level1Report.OrigSize != expectedLevel1Size {
			t.Errorf("Level1 大小错误：期望 %d (unsafe.Sizeof), 得到 %d (optimizer)",
				expectedLevel1Size, level1Report.OrigSize)
		} else {
			t.Logf("✅ Level1 大小正确：%d 字节 (与 unsafe.Sizeof 一致)", level1Report.OrigSize)
		}

		// 验证大小公式
		if level1Report.OrigSize != level1Report.OptSize+level1Report.Saved {
			t.Errorf("Level1 大小公式错误：优化前 (%d) != 优化后 (%d) + 节省 (%d)",
				level1Report.OrigSize, level1Report.OptSize, level1Report.Saved)
		} else {
			t.Logf("✅ Level1 大小公式正确：%d = %d + %d",
				level1Report.OrigSize, level1Report.OptSize, level1Report.Saved)
		}

		// 验证嵌套的 Level2 字段大小
		if dataSize, ok := level1Report.FieldSizes["Data"]; ok {
			if dataSize != expectedLevel2Size {
				t.Errorf("Level1.Data (Level2) 字段大小错误：期望 %d, 得到 %d",
					expectedLevel2Size, dataSize)
			} else {
				t.Logf("✅ Level1.Data (Level2) 字段大小正确：%d 字节", dataSize)
			}
		}

		// 验证所有字段大小
		expectedFieldSizes := map[string]int64{
			"ID":       8,                  // int64
			"Data":     expectedLevel2Size, // Level2
			"Flag":     1,                  // bool
			"Tags":     24,                 // []string (slice header)
			"Metadata": 16,                 // string
			"Count":    4,                  // int32
			"Active":   1,                  // bool
		}

		for fieldName, expectedSize := range expectedFieldSizes {
			if actualSize, ok := level1Report.FieldSizes[fieldName]; !ok {
				t.Errorf("字段 '%s' 在 FieldSizes 中不存在", fieldName)
			} else if actualSize != expectedSize {
				t.Errorf("字段 '%s' 大小错误：期望 %d 字节，得到 %d 字节",
					fieldName, expectedSize, actualSize)
			} else {
				t.Logf("✅ 字段 '%s' 大小正确：%d 字节", fieldName, actualSize)
			}
		}
	}
}

// TestComplexStructWithVariousTypes 测试包含各种复杂类型的结构体
// 包括指针、数组、map、interface、chan 等
func TestComplexStructWithVariousTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "complex_types_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 Go Module 项目
	goModContent := `module testvarioustypes

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建包含各种类型的结构体
	testFile := filepath.Join(pkgDir, "types.go")
	content := `package pkg

type Inner struct {
	A int64
	B bool
}

type ComplexStruct struct {
	// 基本类型
	BoolVal   bool    // 1 字节
	Int8Val   int8    // 1 字节
	Int16Val  int16   // 2 字节
	Int32Val  int32   // 4 字节
	Int64Val  int64   // 8 字节
	UintVal   uint    // 8 字节
	FloatVal  float64 // 8 字节
	
	// 复合类型
	Str       string     // 16 字节
	Slice     []int      // 24 字节
	Map       map[string]int // 8 字节
	Ptr       *Inner     // 8 字节 (指针)
	Array     [3]int64   // 24 字节 (3*8)
	
	// 嵌套结构体
	Nested    Inner      // 16 字节
	
	// 其他类型
	Chan      chan int   // 8 字节
	Interface interface{} // 16 字节
	
	// 更多字段增加复杂度
	Extra1    int32      // 4 字节
	Extra2    bool       // 1 字节
	Extra3    int64      // 8 字节
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 使用 unsafe.Sizeof 验证
	type Inner struct {
		A int64
		B bool
	}

	type ComplexStruct struct {
		BoolVal   bool
		Int8Val   int8
		Int16Val  int16
		Int32Val  int32
		Int64Val  int64
		UintVal   uint
		FloatVal  float64
		Str       string
		Slice     []int
		Map       map[string]int
		Ptr       *Inner
		Array     [3]int64
		Nested    Inner
		Chan      chan int
		Interface interface{}
		Extra1    int32
		Extra2    bool
		Extra3    int64
	}

	expectedSize := int64(unsafe.Sizeof(ComplexStruct{}))
	t.Logf("unsafe.Sizeof(ComplexStruct) = %d 字节", expectedSize)

	// 创建 analyzer 和优化器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	optCfg := &optimizer.Config{
		TargetDir:      tmpDir,
		StructName:     "testvarioustypes/pkg.ComplexStruct",
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

	// 查找报告
	var complexReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "ComplexStruct" {
			complexReport = sr
			break
		}
	}

	if complexReport == nil {
		t.Fatal("未找到 ComplexStruct 的报告")
	}

	t.Logf("ComplexStruct: 优化前=%d, 优化后=%d, 节省=%d",
		complexReport.OrigSize, complexReport.OptSize, complexReport.Saved)

	// 验证总大小与 unsafe.Sizeof 一致
	if complexReport.OrigSize != expectedSize {
		t.Errorf("ComplexStruct 大小错误：期望 %d (unsafe.Sizeof), 得到 %d (optimizer)",
			expectedSize, complexReport.OrigSize)
	} else {
		t.Logf("✅ ComplexStruct 大小正确：%d 字节 (与 unsafe.Sizeof 一致)", complexReport.OrigSize)
	}

	// 验证大小公式
	if complexReport.OrigSize != complexReport.OptSize+complexReport.Saved {
		t.Errorf("大小公式错误：优化前 (%d) != 优化后 (%d) + 节省 (%d)",
			complexReport.OrigSize, complexReport.OptSize, complexReport.Saved)
	} else {
		t.Logf("✅ 大小公式正确：%d = %d + %d",
			complexReport.OrigSize, complexReport.OptSize, complexReport.Saved)
	}

	// 验证关键字段大小
	expectedFieldSizes := map[string]int64{
		"BoolVal":   1,
		"Int64Val":  8,
		"FloatVal":  8,
		"Str":       16,
		"Slice":     24,
		"Map":       8,
		"Ptr":       8,
		"Array":     24,
		"Nested":    16,
		"Chan":      8,
		"Interface": 16,
	}

	allCorrect := true
	for fieldName, expectedSize := range expectedFieldSizes {
		if actualSize, ok := complexReport.FieldSizes[fieldName]; !ok {
			t.Errorf("字段 '%s' 在 FieldSizes 中不存在", fieldName)
			allCorrect = false
		} else if actualSize != expectedSize {
			t.Errorf("字段 '%s' 大小错误：期望 %d 字节，得到 %d 字节",
				fieldName, expectedSize, actualSize)
			allCorrect = false
		} else {
			t.Logf("✅ 字段 '%s' 大小正确：%d 字节", fieldName, actualSize)
		}
	}

	if allCorrect {
		t.Log("✅ 所有字段大小验证通过")
	}
}

// TestSliceOfStructRecursiveOptimization verifies that structs referenced
// via slices/pointers ([]A, []*A) are recursively collected and optimized.
func TestSliceOfStructRecursiveOptimization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slice_struct_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	goModContent := `module testslice

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	mainContent := `package testslice

type A struct {
	Field1 bool
	Field2 uint64
	Field3 bool
}

type B struct {
	Field1 bool
	Field2 uint64
	Field3 bool
}

type XXX struct {
	isOK         bool
	information  []A
	information1 []*B
	isSuccess    bool
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	analyzerCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		StructName:  "testslice.XXX",
		Verbose:     0,
		ProjectType: "gomod",
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		StructName:  "testslice.XXX",
		Verbose:     0,
		ProjectType: "gomod",
		MaxDepth:    50,
		Timeout:     300,
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Verify A (via []A) and B (via []*B) were both collected and optimized
	foundA, foundB := false, false
	for _, sr := range report.StructReports {
		if sr.Name == "A" {
			foundA = true
			if sr.OptSize >= sr.OrigSize {
				t.Errorf("A should be optimized: OptSize(%d) >= OrigSize(%d)", sr.OptSize, sr.OrigSize)
			}
		}
		if sr.Name == "B" {
			foundB = true
			if sr.OptSize >= sr.OrigSize {
				t.Errorf("B should be optimized: OptSize(%d) >= OrigSize(%d)", sr.OptSize, sr.OrigSize)
			}
		}
	}

	if !foundA {
		t.Errorf("A was not collected as a nested struct of XXX via []A field")
	}
	if !foundB {
		t.Errorf("B was not collected as a nested struct of XXX via []*B field")
	}

	t.Logf("✅ Total structs collected: %d", report.TotalStructs)
	if foundA {
		t.Logf("✅ A successfully collected and optimized via []A slice field")
	}
	if foundB {
		t.Logf("✅ B successfully collected and optimized via []*B pointer-slice field")
	}
}
