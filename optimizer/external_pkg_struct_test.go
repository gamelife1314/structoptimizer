package optimizer_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestExternalPackageStructSize 测试包含外部包类型的结构体大小计算
func TestExternalPackageStructSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "external_pkg_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 Go Module 项目
	goModContent := `module testexternal

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建包含标准库类型的结构体
	testFile := filepath.Join(pkgDir, "types.go")
	content := `package pkg

import "time"

type Event struct {
	ID        int64     // 8
	Name      string    // 16
	Timestamp time.Time // 48 (标准库类型)
	Active    bool      // 1
	Data      int32     // 4
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 使用 unsafe.Sizeof 验证
	type Event struct {
		ID        int64
		Name      string
		Timestamp time.Time
		Active    bool
		Data      int32
	}

	expectedSize := int64(unsafe.Sizeof(Event{}))
	t.Logf("unsafe.Sizeof(Event) = %d 字节", expectedSize)

	// 创建 analyzer 和优化器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	optCfg := &optimizer.Config{
		TargetDir:      tmpDir,
		StructName:     "testexternal/pkg.Event",
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
	var eventReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "Event" {
			eventReport = sr
			break
		}
	}

	if eventReport == nil {
		t.Fatal("未找到 Event 结构体的报告")
	}

	t.Logf("Event: 优化前=%d, 优化后=%d, 节省=%d",
		eventReport.OrigSize, eventReport.OptSize, eventReport.Saved)

	// 调试：打印所有字段信息
	t.Log("字段信息:")
	for _, fieldName := range eventReport.OrigFields {
		if size, ok := eventReport.FieldSizes[fieldName]; ok {
			t.Logf("  %s: size=%d", fieldName, size)
		}
	}

	// 验证总大小与 unsafe.Sizeof 一致
	if eventReport.OrigSize != expectedSize {
		t.Errorf("Event 大小错误：期望 %d (unsafe.Sizeof), 得到 %d (optimizer)",
			expectedSize, eventReport.OrigSize)
	} else {
		t.Logf("✅ Event 大小正确：%d 字节 (与 unsafe.Sizeof 一致)", eventReport.OrigSize)
	}

	// 验证公式
	if eventReport.OrigSize != eventReport.OptSize+eventReport.Saved {
		t.Errorf("大小公式错误：优化前 (%d) != 优化后 (%d) + 节省 (%d)",
			eventReport.OrigSize, eventReport.OptSize, eventReport.Saved)
	} else {
		t.Logf("✅ 大小公式正确：%d = %d + %d",
			eventReport.OrigSize, eventReport.OptSize, eventReport.Saved)
	}

	// 验证 time.Time 字段大小
	if tsSize, ok := eventReport.FieldSizes["Timestamp"]; !ok {
		t.Error("字段 'Timestamp' 在 FieldSizes 中不存在")
	} else if tsSize != 24 {
		t.Errorf("time.Time 字段 'Timestamp' 大小错误：期望 24 字节，得到 %d 字节", tsSize)
	} else {
		t.Logf("✅ time.Time 字段 'Timestamp' 大小正确：%d 字节", tsSize)
	}

	// 验证所有字段大小
	expectedFieldSizes := map[string]int64{
		"ID":        8,
		"Name":      16,
		"Timestamp": 24,
		"Active":    1,
		"Data":      4,
	}

	allCorrect := true
	for fieldName, expectedSize := range expectedFieldSizes {
		if actualSize, ok := eventReport.FieldSizes[fieldName]; !ok {
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

// TestMultipleExternalPackages 测试包含多个外部包类型的结构体
func TestMultipleExternalPackages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "multi_external_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 Go Module 项目
	goModContent := `module testmultiext

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建包含多个标准库类型的结构体
	testFile := filepath.Join(pkgDir, "types.go")
	content := `package pkg

import (
	"sync"
	"time"
)

type Service struct {
	Name      string      // 16
	CreatedAt time.Time   // 48
	Mutex     sync.Mutex  // 40
	Timeout   int64       // 8
	Active    bool        // 1
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 使用 unsafe.Sizeof 验证
	type Service struct {
		Name      string
		CreatedAt time.Time
		Mutex     sync.Mutex
		Timeout   int64
		Active    bool
	}

	expectedSize := int64(unsafe.Sizeof(Service{}))
	t.Logf("unsafe.Sizeof(Service) = %d 字节", expectedSize)

	// 创建 analyzer 和优化器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	optCfg := &optimizer.Config{
		TargetDir:      tmpDir,
		StructName:     "testmultiext/pkg.Service",
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
	var serviceReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "Service" {
			serviceReport = sr
			break
		}
	}

	if serviceReport == nil {
		t.Fatal("未找到 Service 结构体的报告")
	}

	t.Logf("Service: 优化前=%d, 优化后=%d, 节省=%d",
		serviceReport.OrigSize, serviceReport.OptSize, serviceReport.Saved)

	// 验证总大小与 unsafe.Sizeof 一致
	if serviceReport.OrigSize != expectedSize {
		t.Errorf("Service 大小错误：期望 %d (unsafe.Sizeof), 得到 %d (optimizer)",
			expectedSize, serviceReport.OrigSize)
	} else {
		t.Logf("✅ Service 大小正确：%d 字节 (与 unsafe.Sizeof 一致)", serviceReport.OrigSize)
	}

	// 验证公式
	if serviceReport.OrigSize != serviceReport.OptSize+serviceReport.Saved {
		t.Errorf("大小公式错误：优化前 (%d) != 优化后 (%d) + 节省 (%d)",
			serviceReport.OrigSize, serviceReport.OptSize, serviceReport.Saved)
	} else {
		t.Logf("✅ 大小公式正确：%d = %d + %d",
			serviceReport.OrigSize, serviceReport.OptSize, serviceReport.Saved)
	}

	// 验证外部包类型字段大小
	expectedExternalSizes := map[string]int64{
		"CreatedAt": 24, // time.Time
		"Mutex":     8,  // sync.Mutex
	}

	for fieldName, expectedSize := range expectedExternalSizes {
		if actualSize, ok := serviceReport.FieldSizes[fieldName]; !ok {
			t.Errorf("字段 '%s' 在 FieldSizes 中不存在", fieldName)
		} else if actualSize != expectedSize {
			t.Errorf("字段 '%s' 大小错误：期望 %d 字节，得到 %d 字节",
				fieldName, expectedSize, actualSize)
		} else {
			t.Logf("✅ 字段 '%s' 大小正确：%d 字节", fieldName, actualSize)
		}
	}
}
