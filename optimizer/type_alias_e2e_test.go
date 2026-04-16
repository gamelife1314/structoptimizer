package optimizer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestTypeAliasSizeCalculation 测试重定义类型大小计算（GOPATH 模式）
func TestTypeAliasSizeCalculation(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_typealias_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany", "myproject", "typealias")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 写入测试文件
	testFile := filepath.Join(pkgDir, "typealias.go")
	testContent := `package typealias

// BadStruct 未优化的结构体，包含重定义类型
type BadStruct struct {
	A bool
	B newType      // uint8 重定义，应该是 1 字节
	C int64
	D AnotherType  // uint16 重定义，应该是 2 字节
	E int32
}

// newType 重定义的类型，底层是 uint8
type newType uint8

// AnotherType 重定义类型，底层是 uint16
type AnotherType uint16
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 创建分析器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/typealias",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/typealias",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		StructName:  "mycompany/myproject/typealias.BadStruct",
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

	// 找到 BadStruct 的报告
	var badStructReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "BadStruct" {
			badStructReport = sr
			break
		}
	}

	if badStructReport == nil {
		t.Fatal("未找到 BadStruct 的报告")
	}

	// 验证字段大小
	// newType 应该是 1 字节
	if size, ok := badStructReport.FieldSizes["B"]; ok {
		if size != 1 {
			t.Errorf("字段 B (newType) 的大小 = %d, 期望 1 字节", size)
		}
	} else {
		t.Error("未找到字段 B 的大小信息")
	}

	// AnotherType 应该是 2 字节
	if size, ok := badStructReport.FieldSizes["D"]; ok {
		if size != 2 {
			t.Errorf("字段 D (AnotherType) 的大小 = %d, 期望 2 字节", size)
		}
	} else {
		t.Error("未找到字段 D 的大小信息")
	}

	// 验证优化后的大小是正确的
	if badStructReport.OptSize > 0 && badStructReport.OrigSize > 0 {
		if badStructReport.OptSize >= badStructReport.OrigSize {
			t.Errorf("优化后大小 (%d) 不应大于等于优化前大小 (%d)", 
				badStructReport.OptSize, badStructReport.OrigSize)
		}
	}
}

// TestTypeAliasFieldSizesInReport 测试报告中字段大小的准确性
func TestTypeAliasFieldSizesInReport(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_report_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试项目
	pkgDir := filepath.Join(tmpDir, "src", "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 写入测试文件
	testFile := filepath.Join(pkgDir, "test.go")
	testContent := `package testpkg

// TestStruct 测试结构体
type TestStruct struct {
	A bool
	B myUint8
	C int64
	D myUint16
	E int32
}

type myUint8 uint8
type myUint16 uint16
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 创建分析器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "testpkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "testpkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		StructName:  "testpkg.TestStruct",
		Verbose:     0,
		Timeout:     60,
	}
	opt := optimizer.NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证报告
	if len(report.StructReports) == 0 {
		t.Fatal("期望至少有一个结构体报告")
	}

	sr := report.StructReports[0]
	if sr.Name != "TestStruct" {
		t.Fatalf("期望结构体名称为 TestStruct，实际为：%s", sr.Name)
	}

	// 验证 FieldSizes 映射存在
	if sr.FieldSizes == nil {
		t.Fatal("FieldSizes 映射为空")
	}

	// 验证各字段大小
	expectedSizes := map[string]int64{
		"A": 1, // bool
		"B": 1, // myUint8 (uint8)
		"C": 8, // int64
		"D": 2, // myUint16 (uint16)
		"E": 4, // int32
	}

	for fieldName, expectedSize := range expectedSizes {
		if actualSize, ok := sr.FieldSizes[fieldName]; ok {
			if actualSize != expectedSize {
				t.Errorf("字段 %s 的大小 = %d, 期望 %d", fieldName, actualSize, expectedSize)
			}
		} else {
			t.Errorf("未找到字段 %s 的大小信息", fieldName)
		}
	}
}

// TestTypeAliasOptimization 测试重定义类型的优化效果
func TestTypeAliasOptimization(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_opt_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试项目
	pkgDir := filepath.Join(tmpDir, "src", "optpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 写入测试文件
	testFile := filepath.Join(pkgDir, "opt.go")
	testContent := `package optpkg

// Unoptimized 未优化的结构体
type Unoptimized struct {
	A bool
	B myByte
	C int64
	D myWord
	E int32
}

type myByte uint8
type myWord uint16
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 创建分析器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "optpkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "optpkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		StructName:  "optpkg.Unoptimized",
		Verbose:     0,
		Timeout:     60,
	}
	opt := optimizer.NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 验证优化效果
	if len(report.StructReports) == 0 {
		t.Fatal("期望至少有一个结构体报告")
	}

	sr := report.StructReports[0]
	
	// 检查是否被优化
	if sr.OptSize <= 0 {
		t.Fatal("优化后大小应为正数")
	}

	// 优化后的大小应该小于等于优化前的大小
	if sr.OptSize > sr.OrigSize {
		t.Errorf("优化后大小 (%d) 不应大于优化前大小 (%d)", sr.OptSize, sr.OrigSize)
	}

	// 验证节省的字节数
	saved := sr.OrigSize - sr.OptSize
	if saved < 0 {
		t.Errorf("节省的字节数不应为负数：%d", saved)
	}

	// 检查字段类型映射
	if sr.FieldTypes == nil {
		t.Fatal("FieldTypes 映射为空")
	}

	// 验证字段类型是正确的
	expectedTypes := map[string]string{
		"A": "bool",
		"B": "myByte",
		"C": "int64",
		"D": "myWord",
		"E": "int32",
	}

	for fieldName, expectedType := range expectedTypes {
		if actualType, ok := sr.FieldTypes[fieldName]; ok {
			if actualType != expectedType {
				t.Errorf("字段 %s 的类型 = %s, 期望 %s", fieldName, actualType, expectedType)
			}
		} else {
			t.Errorf("未找到字段 %s 的类型信息", fieldName)
		}
	}
}

// TestTypeAliasCrossFile 测试跨文件定义的重定义类型
func TestTypeAliasCrossFile(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_crossfile_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试项目
	pkgDir := filepath.Join(tmpDir, "src", "crosspkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 写入主结构体文件
	mainFile := filepath.Join(pkgDir, "main.go")
	mainContent := `package crosspkg

// MainStruct 主结构体，引用其他文件中定义的重定义类型
type MainStruct struct {
	ID   int64
	flag myFlag
	Name string
}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("写入主文件失败：%v", err)
	}

	// 写入类型定义文件
	typeFile := filepath.Join(pkgDir, "types.go")
	typeContent := `package crosspkg

// myFlag 重定义类型
type myFlag uint8

// myPriority 另一个重定义类型
type myPriority uint16
`
	if err := os.WriteFile(typeFile, []byte(typeContent), 0644); err != nil {
		t.Fatalf("写入类型文件失败：%v", err)
	}

	// 创建分析器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "crosspkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "crosspkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		StructName:  "crosspkg.MainStruct",
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

	sr := report.StructReports[0]
	if sr.Name != "MainStruct" {
		t.Fatalf("期望结构体名称为 MainStruct，实际为：%s", sr.Name)
	}

	// 验证 myFlag 字段大小是 1 字节
	if size, ok := sr.FieldSizes["flag"]; ok {
		if size != 1 {
			t.Errorf("字段 flag (myFlag) 的大小 = %d, 期望 1 字节", size)
		}
	} else {
		t.Error("未找到字段 flag 的大小信息")
	}

	// 验证字段类型映射包含重定义类型
	if sr.FieldTypes != nil {
		if flagType, ok := sr.FieldTypes["flag"]; ok {
			if !strings.Contains(flagType, "myFlag") {
				t.Errorf("字段 flag 的类型 = %s, 期望包含 myFlag", flagType)
			}
		}
	}
}
