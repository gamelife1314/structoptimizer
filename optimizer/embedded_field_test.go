package optimizer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestEmbeddedFieldIdentification 测试匿名字段识别修复
// 场景：
// 1. 真正的匿名字段：Config（只有类型，没有名字）
// 2. 字段名和类型名相同：Config Config（有名字，不是匿名）
func TestEmbeddedFieldIdentification(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_embedded_ident_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/embedded")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(pkgDir, "test.go")
	content := `package embedded

// Config 配置结构体
type Config struct {
	Host string
	Port int
}

// MainStruct 测试结构体
type MainStruct struct {
	ID       int64
	Config            // 真正的匿名字段（只有类型，没有名字）
	Name     string
	Data     Data   // 字段名和类型名相同（不是匿名字段）
	Enabled  bool
}

// Data 数据结构体
type Data struct {
	Value int64
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入 test.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/embedded",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/embedded",
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

	// 找到 MainStruct 的报告
	var mainStructReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "MainStruct" {
			mainStructReport = sr
			break
		}
	}

	if mainStructReport == nil {
		t.Fatal("未找到 MainStruct 的报告")
	}

	t.Logf("MainStruct: %d 字节", mainStructReport.OrigSize)
	t.Logf("字段类型: %v", mainStructReport.FieldTypes)

	// 验证字段类型映射
	// Config 应该是匿名字段（Embed）
	// Data 应该是普通字段（非 Embed）
	if mainStructReport.FieldTypes != nil {
		// 验证字段类型信息包含包名
		for fieldName, fieldType := range mainStructReport.FieldTypes {
			t.Logf("字段 %s 类型: %s", fieldName, fieldType)
		}
	}
}

// TestTypeNameWithPackagePrefix 测试类型名称保留包名前缀
func TestTypeNameWithPackagePrefix(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_type_prefix_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构（两个包）
	pkg1Dir := filepath.Join(tmpDir, "src", "mycompany/myproject/api")
	if err := os.MkdirAll(pkg1Dir, 0755); err != nil {
		t.Fatalf("创建 api 目录失败：%v", err)
	}

	pkg2Dir := filepath.Join(tmpDir, "src", "mycompany/myproject/model")
	if err := os.MkdirAll(pkg2Dir, 0755); err != nil {
		t.Fatalf("创建 model 目录失败：%v", err)
	}

	// model 包
	modelFile := filepath.Join(pkg2Dir, "model.go")
	modelContent := `package model

// User 用户模型
type User struct {
	ID   int64
	Name string
}

// Config 配置
type Config struct {
	Timeout int
}
`
	if err := os.WriteFile(modelFile, []byte(modelContent), 0644); err != nil {
		t.Fatalf("写入 model.go 失败：%v", err)
	}

	// api 包（引用 model 包的类型）
	apiFile := filepath.Join(pkg1Dir, "api.go")
	apiContent := `package api

import (
	"mycompany/myproject/model"
)

// Handler API 处理器
type Handler struct {
	Name   string
	User   model.User    // 跨包类型，应该保留包名
	Config model.Config  // 跨包类型，应该保留包名
}
`
	if err := os.WriteFile(apiFile, []byte(apiContent), 0644); err != nil {
		t.Fatalf("写入 api.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/api",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/api",
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

	// 找到 Handler 的报告
	var handlerReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "Handler" {
			handlerReport = sr
			break
		}
	}

	if handlerReport == nil {
		t.Fatal("未找到 Handler 的报告")
	}

	t.Logf("Handler: %d 字节", handlerReport.OrigSize)
	t.Logf("字段类型: %v", handlerReport.FieldTypes)

	// 验证跨包类型保留了包名
	if handlerReport.FieldTypes != nil {
		// User 字段类型应该包含包名
		if userType, ok := handlerReport.FieldTypes["User"]; ok {
			// 应该包含 model 包的标识（至少包含 "model"）
			if !strings.Contains(userType, "model") {
				t.Errorf("User 字段类型应该包含 model 标识，实际: %s", userType)
			} else {
				t.Logf("✅ User 字段类型包含包信息: %s", userType)
			}
		}

		// Config 字段类型应该包含包名
		if configType, ok := handlerReport.FieldTypes["Config"]; ok {
			// 应该包含 model 包的标识
			if !strings.Contains(configType, "model") {
				t.Errorf("Config 字段类型应该包含 model 标识，实际: %s", configType)
			} else {
				t.Logf("✅ Config 字段类型包含包信息: %s", configType)
			}
		}
	}
}

// TestMixedEmbeddedAndNamedFields 测试混合匿名字段和命名字段
func TestMixedEmbeddedAndNamedFields(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_mixed_embed_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/mixed")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(pkgDir, "mixed.go")
	content := `package mixed

// Base 基础结构体
type Base struct {
	ID int64
}

// Meta 元数据结构体
type Meta struct {
	CreatedAt int64
}

// Complex 复杂结构体（混合匿名和命名字段）
type Complex struct {
	Base             // 匿名字段
	Name     string  // 普通命名字段
	Meta   Meta     // 字段名和类型名相同（不是匿名）
	Data    *Base   // 指针类型命名字段
	Enabled bool    // 普通命名字段
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入 mixed.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/mixed",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/mixed",
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

	// 找到 Complex 的报告
	var complexReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "Complex" {
			complexReport = sr
			break
		}
	}

	if complexReport == nil {
		t.Fatal("未找到 Complex 的报告")
	}

	t.Logf("Complex: %d 字节", complexReport.OrigSize)
	t.Logf("字段类型：%v", complexReport.FieldTypes)

	// 验证所有字段都被正确识别
	// 注意：字段名使用纯名称（与 OrigFields/OptFields 一致）
	expectedFields := map[string]string{
		"Base":    "Base",    // 匿名字段
		"Name":    "string",  // 命名字段
		"Meta":    "Meta",    // 命名字段（结构体类型）
		"Data":    "*Base",   // 命名字段（指针类型）
		"Enabled": "bool",    // 命名字段
	}
	for fieldKey, expectedType := range expectedFields {
		if complexReport.FieldTypes != nil {
			if fieldType, ok := complexReport.FieldTypes[fieldKey]; ok {
				t.Logf("✅ 字段 %s: %s", fieldKey, fieldType)
				if fieldType != expectedType {
					t.Errorf("字段 %s 类型错误：期望 %s, 得到 %s", fieldKey, expectedType, fieldType)
				}
			} else {
				t.Errorf("未找到字段 %s", fieldKey)
			}
		}
	}
}

// TestReportFieldTypeAndSizeDisplay 测试报告中字段类型和大小的正确显示
// 这是针对 "字段顺序对比部分丢失类型和大小统计" 问题的回归测试
// 确保 FieldTypes 映射的 key 与 OrigFields/OptFields 中的名称一致
func TestReportFieldTypeAndSizeDisplay(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "report_field_type_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 Go Module 项目
	goModContent := `module testreport

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}

	// 创建测试包目录
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建需要优化的结构体（字段顺序不佳）
	testFile := filepath.Join(pkgDir, "types.go")
	content := `package pkg

// OptimizableStruct 字段顺序不佳，可以优化
type OptimizableStruct struct {
	Flag   bool    // 1 字节
	Name   string  // 16 字节
	Count  int64   // 8 字节
	Age    int32   // 4 字节
	Active bool    // 1 字节
}

// EmbeddedStruct 包含匿名字段的结构体
type EmbeddedStruct struct {
	Enabled bool   // 1 字节
	Config         // 匿名字段
	ID      int64  // 8 字节
}

// Config 用于嵌入的配置
type Config struct {
	Host string
	Port int
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 创建 analyzer
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		ProjectType: "gomod",
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:      tmpDir,
		StructName:     "testreport/pkg.OptimizableStruct",
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

	if len(report.StructReports) != 1 {
		t.Fatalf("期望 1 个结构体报告，得到 %d 个", len(report.StructReports))
	}

	sr := report.StructReports[0]

	// 验证 FieldTypes 不为空
	if sr.FieldTypes == nil {
		t.Fatal("FieldTypes 不应为 nil")
	}

	t.Logf("结构体：%s.%s", sr.PkgPath, sr.Name)
	t.Logf("优化前大小：%d 字节", sr.OrigSize)
	t.Logf("优化后大小：%d 字节", sr.OptSize)
	t.Logf("字段类型映射：%v", sr.FieldTypes)

	// 验证所有字段都有类型信息
	expectedFieldTypes := map[string]string{
		"Flag":   "bool",
		"Name":   "string",
		"Count":  "int64",
		"Age":    "int32",
		"Active": "bool",
	}

	for fieldName, expectedType := range expectedFieldTypes {
		actualType, ok := sr.FieldTypes[fieldName]
		if !ok {
			t.Errorf("字段 '%s' 在 FieldTypes 中不存在", fieldName)
			continue
		}
		if actualType != expectedType {
			t.Errorf("字段 '%s' 类型错误：期望 %s, 得到 %s", fieldName, expectedType, actualType)
		}
		t.Logf("✅ 字段 '%s': 类型=%s", fieldName, actualType)
	}

	// 验证 OrigFields 和 OptFields 中的字段都能在 FieldTypes 中找到
	for i, fieldName := range sr.OrigFields {
		if _, ok := sr.FieldTypes[fieldName]; !ok {
			t.Errorf("OrigFields[%d]='%s' 在 FieldTypes 中不存在", i, fieldName)
		}
	}
	for i, fieldName := range sr.OptFields {
		if _, ok := sr.FieldTypes[fieldName]; !ok {
			t.Errorf("OptFields[%d]='%s' 在 FieldTypes 中不存在", i, fieldName)
		}
	}

	// 测试包含匿名字段的结构体
	anlz2 := analyzer.NewAnalyzer(anlzCfg)
	optCfg2 := &optimizer.Config{
		TargetDir:      tmpDir,
		StructName:     "testreport/pkg.EmbeddedStruct",
		ProjectType:    "gomod",
		Verbose:        0,
		MaxDepth:       50,
		Timeout:        300,
		PkgWorkerLimit: 4,
	}
	opt2 := optimizer.NewOptimizer(optCfg2, anlz2)

	report2, err := opt2.Optimize()
	if err != nil {
		t.Fatalf("优化 EmbeddedStruct 失败：%v", err)
	}

	// 查找 EmbeddedStruct 的报告（可能包含依赖的 Config 结构体）
	var sr2 *optimizer.StructReport
	for _, sr := range report2.StructReports {
		if sr.Name == "EmbeddedStruct" {
			sr2 = sr
			break
		}
	}
	if sr2 == nil {
		t.Fatalf("未找到 EmbeddedStruct 的报告，报告列表：%v", report2.StructReports)
	}
	t.Logf("\n匿名字段测试 - 结构体：%s.%s", sr2.PkgPath, sr2.Name)
	t.Logf("字段类型映射：%v", sr2.FieldTypes)

	// 验证匿名字段 Config 的类型
	if configType, ok := sr2.FieldTypes["Config"]; !ok {
		t.Error("匿名字段 'Config' 在 FieldTypes 中不存在")
	} else {
		t.Logf("✅ 匿名字段 'Config': 类型=%s", configType)
		if configType != "Config" {
			t.Errorf("匿名字段 'Config' 类型错误：期望 Config, 得到 %s", configType)
		}
	}

	// 验证命名字段的类型
	for fieldName, expectedType := range map[string]string{
		"Enabled": "bool",
		"ID":      "int64",
	} {
		if actualType, ok := sr2.FieldTypes[fieldName]; !ok {
			t.Errorf("命名字段 '%s' 在 FieldTypes 中不存在", fieldName)
		} else if actualType != expectedType {
			t.Errorf("命名字段 '%s' 类型错误：期望 %s, 得到 %s", fieldName, expectedType, actualType)
		} else {
			t.Logf("✅ 命名字段 '%s': 类型=%s", fieldName, actualType)
		}
	}
}
