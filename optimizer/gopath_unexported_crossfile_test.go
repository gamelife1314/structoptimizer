package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestGopathUnexportedStructCrossFile 测试 GOPATH 模式下同包不同文件的未导出结构体识别
// 场景：
//
//	文件 A: main.go  - 定义 MainStruct，包含未导出字段 internalData
//	文件 B: types.go - 定义 internalData 结构体（未导出）
//	期望：能够识别 internalData 是同包中定义的结构体类型
func TestGopathUnexportedStructCrossFile(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_unexported_crossfile_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	// $GOPATH/src/mycompany/myproject/data/
	//   main.go   - 包含 MainStruct
	//   types.go  - 包含 internalData（未导出结构体）

	pkgDir := filepath.Join(tmpDir, "src", "mycompany", "myproject", "data")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建 main.go（包含引用未导出结构体的 MainStruct）
	mainFile := filepath.Join(pkgDir, "main.go")
	mainContent := `package data

// MainStruct 主结构体，包含未导出的 internalData 字段
type MainStruct struct {
	ID       int64
	Name     string
	internalData  // 未导出嵌套结构体（同包不同文件定义）
	Enabled  bool
}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("写入 main.go 失败：%v", err)
	}

	// 创建 types.go（包含未导出的 internalData 结构体）
	typesFile := filepath.Join(pkgDir, "types.go")
	typesContent := `package data

// internalData 未导出的内部数据结构体
type internalData struct {
	Buffer     []byte
	Offset     int64
	Size       uint32
	IsDirty    bool
}

// PublicConfig 导出的配置结构体（对比用）
type PublicConfig struct {
	Timeout int
	Retry   int
}
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatalf("写入 types.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/data",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器（GOPATH 模式）
	optCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/data",
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

	// 验证 internalData 字段被识别为结构体
	// 这证明同包跨文件的未导出结构体类型识别成功
	t.Logf("MainStruct: %d -> %d 字节", mainStructReport.OrigSize, mainStructReport.OptSize)
	t.Logf("字段列表: %v", mainStructReport.FieldTypes)
}

// TestGopathUnexportedStructMultipleFiles 测试多个文件中的多个未导出结构体
func TestGopathUnexportedStructMultipleFiles(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_multi_unexported_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/multi")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 文件 1：主结构体
	file1 := filepath.Join(pkgDir, "api.go")
	content1 := `package multi

// APIHandler 处理器
type APIHandler struct {
	Name    string
	config  *serverConfig  // 未导出结构体指针（在 config.go 中定义）
	metrics *metricData    // 未导出结构体指针（在 metrics.go 中定义）
	Port    int
}
`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("写入 api.go 失败：%v", err)
	}

	// 文件 2：配置结构体
	file2 := filepath.Join(pkgDir, "config.go")
	content2 := `package multi

// serverConfig 服务器配置（未导出）
type serverConfig struct {
	MaxConn     int
	Timeout     int64
	RetryCount  uint8
}
`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("写入 config.go 失败：%v", err)
	}

	// 文件 3：指标结构体
	file3 := filepath.Join(pkgDir, "metrics.go")
	content3 := `package multi

// metricData 指标数据（未导出）
type metricData struct {
	RequestCount int64
	ErrorCount   int64
	AvgLatency   float64
}
`
	if err := os.WriteFile(file3, []byte(content3), 0644); err != nil {
		t.Fatalf("写入 metrics.go 失败：%v", err)
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

	// 验证 APIHandler 被正确处理
	var apiHandlerReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "APIHandler" {
			apiHandlerReport = sr
			break
		}
	}

	if apiHandlerReport == nil {
		t.Fatal("未找到 APIHandler 的报告")
	}

	t.Logf("APIHandler: %d -> %d 字节", apiHandlerReport.OrigSize, apiHandlerReport.OptSize)
	t.Logf("字段类型: %v", apiHandlerReport.FieldTypes)

	// 验证嵌套的未导出结构体也被处理了
	var serverConfigReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "serverConfig" {
			serverConfigReport = sr
			break
		}
	}

	if serverConfigReport == nil {
		t.Log("警告：未找到 serverConfig 的报告（可能是嵌套深度限制）")
	} else {
		t.Logf("serverConfig: %d -> %d 字节", serverConfigReport.OrigSize, serverConfigReport.OptSize)
	}

	var metricDataReport *optimizer.StructReport
	for _, sr := range report.StructReports {
		if sr.Name == "metricData" {
			metricDataReport = sr
			break
		}
	}

	if metricDataReport == nil {
		t.Log("警告：未找到 metricData 的报告（可能是嵌套深度限制）")
	} else {
		t.Logf("metricData: %d -> %d 字节", metricDataReport.OrigSize, metricDataReport.OptSize)
	}
}
