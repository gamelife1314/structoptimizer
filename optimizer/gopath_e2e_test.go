package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestGopathOptimizerEndToEnd 测试 GOPATH 模式下优化器端到端功能
func TestGopathOptimizerEndToEnd(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_optimizer_e2e_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 GOPATH 项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/api")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建包含需要优化的结构体的文件
	modelsFile := filepath.Join(pkgDir, "models.go")
	modelsContent := `package api

// UnoptimizedRequest 未优化的请求结构体
type UnoptimizedRequest struct {
	Action    string
	Timestamp int64
	Enabled   bool
	UserID    int64
	Retry     uint8
	Data      string
	Priority  int8
}

// UnoptimizedResponse 未优化的响应结构体
type UnoptimizedResponse struct {
	Code      int
	Success   bool
	Message   string
	Data      []byte
	Timestamp int64
	Count     int32
}

// Config 配置结构体（已经优化）
type Config struct {
	Host    string
	Port    int
	Timeout int64
	Enabled bool
}

// Status 状态枚举
type Status int

const (
	StatusOK Status = iota
	StatusError
	StatusTimeout
)
`
	if err := os.WriteFile(modelsFile, []byte(modelsContent), 0644); err != nil {
		t.Fatalf("写入 models.go 失败：%v", err)
	}

	// 创建服务文件（引用 models）
	serviceFile := filepath.Join(pkgDir, "service.go")
	serviceContent := `package api

import "time"

// Handler 处理器
type Handler struct {
	config  *Config
	timeout time.Duration
	status  Status
}

// NewHandler 创建处理器
func NewHandler(cfg *Config) *Handler {
	return &Handler{
		config:  cfg,
		timeout: time.Duration(cfg.Timeout) * time.Second,
		status:  StatusOK,
	}
}

// Process 处理请求
func (h *Handler) Process(req *UnoptimizedRequest) *UnoptimizedResponse {
	return &UnoptimizedResponse{
		Code:      200,
		Success:   true,
		Message:   "OK",
		Data:      nil,
		Timestamp: req.Timestamp,
		Count:     0,
	}
}
`
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		t.Fatalf("写入 service.go 失败：%v", err)
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

	// 创建优化器（GOPATH 模式）
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

	// 验证结果
	if len(report.StructReports) == 0 {
		t.Fatal("期望至少有一个结构体报告")
	}

	// 找到 UnoptimizedRequest 的报告
	var reqReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "UnoptimizedRequest" {
			reqReport = sr
			break
		}
	}

	if reqReport == nil {
		t.Fatal("未找到 UnoptimizedRequest 的报告")
	}

	// 验证优化效果
	if reqReport.OrigSize <= 0 {
		t.Errorf("原始大小应为正数，实际 %d", reqReport.OrigSize)
	}

	if reqReport.OptSize < 0 {
		t.Errorf("优化后大小应为非负数，实际 %d", reqReport.OptSize)
	}

	// 未优化的结构体应该可以优化
	if reqReport.OrigSize == reqReport.OptSize {
		t.Logf("警告：UnoptimizedRequest 没有被优化（原始=%d, 优化后=%d）",
			reqReport.OrigSize, reqReport.OptSize)
	}

	t.Logf("UnoptimizedRequest: %d -> %d 字节 (节省 %d 字节)",
		reqReport.OrigSize, reqReport.OptSize, reqReport.OrigSize-reqReport.OptSize)

	// 验证 Config 结构体（已经优化，不应该再优化）
	var cfgReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "Config" {
			cfgReport = sr
			break
		}
	}

	if cfgReport != nil {
		t.Logf("Config: %d -> %d 字节", cfgReport.OrigSize, cfgReport.OptSize)
	}
}

// TestGopathOptimizerWithNestedStructs 测试 GOPATH 模式下嵌套结构体优化
func TestGopathOptimizerWithNestedStructs(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_nested_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/nested")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建包含嵌套结构体的文件
	nestedFile := filepath.Join(pkgDir, "nested.go")
	nestedContent := `package nested

// Inner 内部结构体
type Inner struct {
	Value int64
	Flag  bool
	Name  string
}

// Outer 外部结构体（包含内部结构体）
type Outer struct {
	ID     int64
	Data   Inner
	Active bool
	Code   int32
}

// Container 容器结构体（包含指针）
type Container struct {
	Items []*Inner
	Count int
	Name  string
}
`
	if err := os.WriteFile(nestedFile, []byte(nestedContent), 0644); err != nil {
		t.Fatalf("写入 nested.go 失败：%v", err)
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

	// 验证结果
	if len(report.StructReports) == 0 {
		t.Fatal("期望至少有一个结构体报告")
	}

	// 打印所有结构体的优化结果
	for _, sr := range report.StructReports {
		t.Logf("%s: %d -> %d 字节 (节省 %d 字节)",
			sr.Name, sr.OrigSize, sr.OptSize, sr.OrigSize-sr.OptSize)
	}

	// 找到 Outer 结构体的报告
	var outerReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "Outer" {
			outerReport = sr
			break
		}
	}

	if outerReport == nil {
		t.Fatal("未找到 Outer 的报告")
	}

	// 验证 Outer 结构体大小
	if outerReport.OrigSize <= 0 {
		t.Errorf("Outer 原始大小应为正数，实际 %d", outerReport.OrigSize)
	}
}

// TestGopathOptimizerSkipsTestFiles 测试 GOPATH 模式下排除测试文件
func TestGopathOptimizerSkipsTestFiles(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_skip_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/skiptest")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建正常的源文件
	mainFile := filepath.Join(pkgDir, "main.go")
	mainContent := `package skiptest

// Data 数据结构
type Data struct {
	ID   int64
	Name string
}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("写入 main.go 失败：%v", err)
	}

	// 创建测试文件（应该被排除）
	testFile := filepath.Join(pkgDir, "main_test.go")
	testContent := `package skiptest

// TestOnlyStruct 只用于测试的结构体
type TestOnlyStruct struct {
	MockData string
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("写入 main_test.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/skiptest",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 加载包
	pkg, err := anlz.LoadPackage("mycompany/myproject/skiptest")
	if err != nil {
		t.Fatalf("加载包失败：%v", err)
	}

	// 验证 Go 文件列表不包含测试文件
	for _, f := range pkg.GoFiles {
		if len(f) > 8 && f[len(f)-8:] == "_test.go" {
			t.Errorf("不应该包含测试文件：%s", f)
		}
	}

	// 验证 TestOnlyStruct 不存在（因为测试文件被排除了）
	obj := pkg.Types.Scope().Lookup("TestOnlyStruct")
	if obj != nil {
		t.Error("TestOnlyStruct 不应该存在（测试文件应该被排除）")
	}

	// 验证 Data 结构存在
	obj = pkg.Types.Scope().Lookup("Data")
	if obj == nil {
		t.Error("Data 结构体应该存在")
	}
}
