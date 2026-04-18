package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestNestedStructCrossFile 测试同包不同文件中的嵌套结构体扫描
// 场景：
//
//	文件 A: main.go   - 定义 MainStruct，包含嵌套结构体字段 InternalConfig
//	文件 B: config.go - 定义 InternalConfig 结构体（未导出）
//	期望：能够正确扫描到 InternalConfig 并将其作为嵌套结构体处理
func TestNestedStructCrossFile(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_nested_crossfile_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/nested")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建 main.go（包含 MainStruct，嵌套 InternalConfig）
	mainFile := filepath.Join(pkgDir, "main.go")
	mainContent := `package nested

// MainStruct 主结构体，嵌套 InternalConfig（在同包不同文件中定义）
type MainStruct struct {
	ID       int64
	Name     string
	Config   InternalConfig // 嵌套结构体（在 config.go 中定义）
	Enabled  bool
}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("写入 main.go 失败：%v", err)
	}

	// 创建 config.go（包含 InternalConfig 结构体）
	configFile := filepath.Join(pkgDir, "config.go")
	configContent := `package nested

// InternalConfig 内部配置结构体（未导出）
type InternalConfig struct {
	Timeout  int64
	Retry    int32
	Buffer   uint16
	Flag     bool
}

// PublicConfig 公共配置结构体（导出的，对比用）
type PublicConfig struct {
	Host string
	Port int
}
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入 config.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/nested",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器（GOPATH 模式）
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/nested",
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

	// 验证结果：应该扫描到 MainStruct 和 InternalConfig
	structNames := make(map[string]bool)
	for _, sr := range report.StructReports {
		structNames[sr.Name] = true
		t.Logf("扫描到结构体：%s (%d 字节)", sr.Name, sr.OrigSize)
	}

	// 验证 MainStruct 被扫描到
	if !structNames["MainStruct"] {
		t.Error("未扫描到 MainStruct")
	}

	// 验证 InternalConfig 被扫描到（关键测试：同包不同文件中的嵌套结构体）
	if !structNames["InternalConfig"] {
		t.Error("未扫描到 InternalConfig（同包不同文件中的嵌套结构体）")
	}

	// 验证 PublicConfig 也被扫描到（在同一个文件中）
	if !structNames["PublicConfig"] {
		t.Error("未扫描到 PublicConfig")
	}
}

// TestNestedStructMultipleLevels 测试多层嵌套结构体（跨文件）
func TestNestedStructMultipleLevels(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_nested_multi_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/multilevel")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 文件 1：顶层结构体
	file1 := filepath.Join(pkgDir, "top.go")
	content1 := `package multilevel

// TopStruct 顶层结构体，嵌套 MidStruct（在 mid.go 中定义）
type TopStruct struct {
	ID   int64
	Mid  MidStruct
	Name string
}
`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("写入 top.go 失败：%v", err)
	}

	// 文件 2：中间层结构体
	file2 := filepath.Join(pkgDir, "mid.go")
	content2 := `package multilevel

// MidStruct 中间层结构体，嵌套 BottomStruct（在 bottom.go 中定义）
type MidStruct struct {
	Code   int32
	Bottom BottomStruct
	Flag   bool
}
`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("写入 mid.go 失败：%v", err)
	}

	// 文件 3：底层结构体
	file3 := filepath.Join(pkgDir, "bottom.go")
	content3 := `package multilevel

// BottomStruct 底层结构体
type BottomStruct struct {
	Value int64
	Data  uint8
}
`
	if err := os.WriteFile(file3, []byte(content3), 0644); err != nil {
		t.Fatalf("写入 bottom.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/multilevel",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/multilevel",
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

	// 验证所有层级的结构体都被扫描到
	structNames := make(map[string]bool)
	for _, sr := range report.StructReports {
		structNames[sr.Name] = true
		t.Logf("扫描到结构体：%s (%d 字节)", sr.Name, sr.OrigSize)
	}

	// 验证三层结构体都被扫描到
	expectedStructs := []string{"TopStruct", "MidStruct", "BottomStruct"}
	for _, name := range expectedStructs {
		if !structNames[name] {
			t.Errorf("未扫描到 %s（多层嵌套结构体）", name)
		} else {
			t.Logf("✅ 扫描到 %s", name)
		}
	}
}

// TestNestedStructWithPointer 测试指针类型的嵌套结构体（跨文件）
func TestNestedStructWithPointer(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_nested_ptr_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/ptrnested")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 文件 1：主结构体（使用指针嵌套）
	file1 := filepath.Join(pkgDir, "main.go")
	content1 := `package ptrnested

// MainHandler 主结构体，使用指针嵌套 InternalConfig
type MainHandler struct {
	Name   string
	Config *InternalConfig // 指针类型嵌套（在 config.go 中定义）
	Port   int
}
`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("写入 main.go 失败：%v", err)
	}

	// 文件 2：配置结构体
	file2 := filepath.Join(pkgDir, "config.go")
	content2 := `package ptrnested

// InternalConfig 内部配置（未导出）
type InternalConfig struct {
	Timeout int64
	Retry   int
}
`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("写入 config.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/ptrnested",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/ptrnested",
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
	structNames := make(map[string]bool)
	for _, sr := range report.StructReports {
		structNames[sr.Name] = true
		t.Logf("扫描到结构体：%s (%d 字节)", sr.Name, sr.OrigSize)
	}

	// 验证 MainHandler 被扫描到
	if !structNames["MainHandler"] {
		t.Error("未扫描到 MainHandler")
	}

	// 验证 InternalConfig 被扫描到（指针类型嵌套）
	if !structNames["InternalConfig"] {
		t.Error("未扫描到 InternalConfig（指针类型嵌套结构体）")
	} else {
		t.Logf("✅ 扫描到 InternalConfig（指针类型嵌套）")
	}
}
