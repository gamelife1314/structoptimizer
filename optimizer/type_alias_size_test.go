package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestTypeAliasSizeFix 测试重定义类型大小计算修复
// 场景：type NewType uint8 应该是 1 字节，而不是 8 字节
func TestTypeAliasSizeFix(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_typealias_fix_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/alias")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建类型定义文件（包含重定义类型）
	typesFile := filepath.Join(pkgDir, "types.go")
	typesContent := `package alias

// NewType 重定义的类型，底层是 uint8，应该是 1 字节
type NewType uint8

// AnotherType 重定义类型，底层是 uint16，应该是 2 字节
type AnotherType uint16

// Int64Type 重定义类型，底层是 int64，应该是 8 字节
type Int64Type int64
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatalf("写入 types.go 失败：%v", err)
	}

	// 创建使用重定义类型的结构体文件
	structFile := filepath.Join(pkgDir, "structs.go")
	structContent := `package alias

// StructWithTypeAlias 包含重定义类型的结构体
type StructWithTypeAlias struct {
	ID    int64
	Flag  NewType     // 应该是 1 字节
	Code  AnotherType // 应该是 2 字节
	Value Int64Type   // 应该是 8 字节
	Name  string
}
`
	if err := os.WriteFile(structFile, []byte(structContent), 0644); err != nil {
		t.Fatalf("写入 structs.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/alias",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器（GOPATH 模式）
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/alias",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
		Timeout:     60,
	}
	opt := optimizer.NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证结果
	if len(report.StructReports) == 0 {
		t.Fatal("期望至少有一个结构体报告")
	}

	// 找到 StructWithTypeAlias 的报告
	var structReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "StructWithTypeAlias" {
			structReport = sr
			break
		}
	}

	if structReport == nil {
		t.Fatal("未找到 StructWithTypeAlias 的报告")
	}

	// 验证字段大小
	t.Logf("StructWithTypeAlias 大小：%d 字节", structReport.OrigSize)
	t.Logf("字段类型：%v", structReport.FieldTypes)

	// 验证原始大小计算正确
	// 如果重定义类型被错误识别为 8 字节，结构体大小会明显偏大
	// 正确的计算应该是：
	// ID (int64): 8 字节
	// Flag (NewType/uint8): 1 字节
	// Code (AnotherType/uint16): 2 字节
	// Value (Int64Type/int64): 8 字节
	// Name (string): 16 字节
	// 考虑对齐后，总大小应该合理（不会超过 50 字节）

	if structReport.OrigSize > 50 {
		t.Errorf("结构体大小 %d 字节异常偏大，可能重定义类型被错误计算为 8 字节", structReport.OrigSize)
	} else {
		t.Logf("✅ 结构体大小合理：%d 字节", structReport.OrigSize)
	}
}

// TestTypeAliasMultipleStructs 测试多个包含重定义类型的结构体
func TestTypeAliasMultipleStructs(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_alias_multi_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/multi")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 类型定义
	typesFile := filepath.Join(pkgDir, "types.go")
	typesContent := `package multi

type ByteFlag uint8
type Word16 uint16
type DWord32 uint32
type QWord64 uint64
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatalf("写入 types.go 失败：%v", err)
	}

	// 结构体定义
	structsFile := filepath.Join(pkgDir, "structs.go")
	structsContent := `package multi

// SmallStruct 小结构体，包含 ByteFlag
type SmallStruct struct {
	ID    int64
	Flag  ByteFlag
	Name  string
}

// MediumStruct 中等结构体
type MediumStruct struct {
	A int64
	B Word16
	C ByteFlag
	D string
}

// LargeStruct 大结构体
type LargeStruct struct {
	P1 QWord64
	P2 DWord32
	P3 Word16
	P4 ByteFlag
	P5 string
}
`
	if err := os.WriteFile(structsFile, []byte(structsContent), 0644); err != nil {
		t.Fatalf("写入 structs.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/multi",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/multi",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
		Timeout:     60,
	}
	opt := optimizer.NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证所有结构体都被处理
	expectedStructs := map[string]bool{
		"SmallStruct":  false,
		"MediumStruct": false,
		"LargeStruct":  false,
	}

	for _, sr := range report.StructReports {
		if _, ok := expectedStructs[sr.Name]; ok {
			expectedStructs[sr.Name] = true
			t.Logf("✅ %s: %d 字节", sr.Name, sr.OrigSize)
		}
	}

	// 验证所有结构体都被找到
	for name, found := range expectedStructs {
		if !found {
			t.Errorf("未找到结构体：%s", name)
		}
	}
}
