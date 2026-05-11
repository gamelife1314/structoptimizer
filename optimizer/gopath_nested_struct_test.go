package optimizer_test

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestGOPATHNestedStructSizeCalculation 测试 GOPATH 模式下嵌套结构体大小计算
// 验证 OrigSize = OptSize + Saved 公式在 GOPATH 项目中成立
func TestGOPATHNestedStructSizeCalculation(t *testing.T) {
	// 创建临时 GOPATH 目录
	gopath, err := os.MkdirTemp("", "gopath_nested_*")
	if err != nil {
		t.Fatalf("创建临时 GOPATH 失败：%v", err)
	}
	defer os.RemoveAll(gopath)

	// 创建 GOPATH 项目结构
	pkgDir := filepath.Join(gopath, "src", "example.com/myproject/pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建复杂的嵌套结构体（模拟真实 GOPATH 项目）
	testFile := filepath.Join(pkgDir, "models.go")
	content := `package pkg

// Config 配置结构体
type Config struct {
	Host string
	Port int32
	Timeout int64
	Enabled bool
}

// Database 数据库配置（嵌套 Config）
type Database struct {
	Driver   string  // 16
	DSN      string  // 16
	Config   Config  // 嵌套结构体
	PoolSize int32   // 4
	Active   bool    // 1
}

// Cache 缓存配置（嵌套 Config）
type Cache struct {
	Type     string  // 16
	Address  string  // 16
	Config   Config  // 嵌套结构体
	TTL      int64   // 8
	Enabled  bool    // 1
}

// Server 服务器配置（多层嵌套）
type Server struct {
	Name     string   // 16
	Host     string   // 16
	Port     int32    // 4
	Database Database // 嵌套
	Cache    Cache    // 嵌套
	Timeout  int64    // 8
	Debug    bool     // 1
}

// App 应用配置（最外层，多层嵌套）
type App struct {
	ID       int64    // 8
	Name     string   // 16
	Version  string   // 16
	Server   Server   // 嵌套（最复杂）
	Config   Config   // 嵌套
	Active   bool     // 1
	Count    int32    // 4
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}

	// 使用 unsafe.Sizeof 计算实际大小（在本地编译验证）
	type Config struct {
		Host    string
		Port    int32
		Timeout int64
		Enabled bool
	}

	type Database struct {
		Driver   string
		DSN      string
		Config   Config
		PoolSize int32
		Active   bool
	}

	type Cache struct {
		Type    string
		Address string
		Config  Config
		TTL     int64
		Enabled bool
	}

	type Server struct {
		Name     string
		Host     string
		Port     int32
		Database Database
		Cache    Cache
		Timeout  int64
		Debug    bool
	}

	type App struct {
		ID      int64
		Name    string
		Version string
		Server  Server
		Config  Config
		Active  bool
		Count   int32
	}

	expectedConfigSize := int64(unsafe.Sizeof(Config{}))
	expectedDatabaseSize := int64(unsafe.Sizeof(Database{}))
	expectedCacheSize := int64(unsafe.Sizeof(Cache{}))
	expectedServerSize := int64(unsafe.Sizeof(Server{}))
	expectedAppSize := int64(unsafe.Sizeof(App{}))

	t.Logf("unsafe.Sizeof 计算结果:")
	t.Logf("  Config:   %d 字节", expectedConfigSize)
	t.Logf("  Database: %d 字节", expectedDatabaseSize)
	t.Logf("  Cache:    %d 字节", expectedCacheSize)
	t.Logf("  Server:   %d 字节", expectedServerSize)
	t.Logf("  App:      %d 字节", expectedAppSize)

	// 创建 analyzer 和优化器（GOPATH 模式）
	// 注意：GOPATH 模式下需要使用 Package 参数，TargetDir 应该是 GOPATH 根目录
	anlzCfg := &analyzer.Config{
		TargetDir:   gopath,
		Package:     "example.com/myproject/pkg",
		ProjectType: "gopath",
		GOPATH:      gopath,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	optCfg := &optimizer.Config{
		TargetDir:      gopath,
		Package:        "example.com/myproject/pkg",
		ProjectType:    "gopath",
		GOPATH:         gopath,
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

	// 收集所有报告
	reports := make(map[string]*optimizer.StructReport)
	for _, sr := range report.StructReports {
		reports[sr.Name] = sr
		t.Logf("结构体 %s: 优化前=%d, 优化后=%d, 节省=%d, 跳过=%v, 原因=%s",
			sr.Name, sr.OrigSize, sr.OptSize, sr.Saved, sr.Skipped, sr.SkipReason)
	}

	t.Logf("共收集到 %d 个结构体报告", len(report.StructReports))
	t.Logf("总结构体数：%d, 优化数：%d, 跳过数：%d",
		report.TotalStructs, report.OptimizedCount, report.SkippedCount)

	// 验证每个结构体
	validateStruct := func(name string, expectedSize int64) {
		sr, ok := reports[name]
		if !ok {
			t.Errorf("未找到结构体 %s 的报告", name)
			return
		}

		// 验证与 unsafe.Sizeof 一致
		if sr.OrigSize != expectedSize {
			t.Errorf("%s 大小错误：期望 %d (unsafe.Sizeof), 得到 %d (optimizer)",
				name, expectedSize, sr.OrigSize)
		} else {
			t.Logf("✅ %s 大小正确：%d 字节 (与 unsafe.Sizeof 一致)", name, sr.OrigSize)
		}

		// 验证公式：OrigSize = OptSize + Saved
		if sr.OrigSize != sr.OptSize+sr.Saved {
			t.Errorf("%s 大小公式错误：优化前 (%d) != 优化后 (%d) + 节省 (%d)",
				name, sr.OrigSize, sr.OptSize, sr.Saved)
		} else {
			t.Logf("✅ %s 大小公式正确：%d = %d + %d",
				name, sr.OrigSize, sr.OptSize, sr.Saved)
		}
	}

	validateStruct("Config", expectedConfigSize)
	validateStruct("Database", expectedDatabaseSize)
	validateStruct("Cache", expectedCacheSize)
	validateStruct("Server", expectedServerSize)
	validateStruct("App", expectedAppSize)

	// 特别验证 App 结构体的嵌套字段大小
	if appReport, ok := reports["App"]; ok {
		t.Logf("\nApp 结构体字段大小:")
		expectedAppFields := map[string]int64{
			"ID":      8,
			"Name":    16,
			"Version": 16,
			"Server":  expectedServerSize,
			"Config":  expectedConfigSize,
			"Active":  1,
			"Count":   4,
		}

		for fieldName, expectedSize := range expectedAppFields {
			if actualSize, ok := appReport.FieldSizes[fieldName]; !ok {
				t.Errorf("App.%s 在 FieldSizes 中不存在", fieldName)
			} else if actualSize != expectedSize {
				t.Errorf("App.%s 大小错误：期望 %d, 得到 %d",
					fieldName, expectedSize, actualSize)
			} else {
				t.Logf("✅ App.%s 大小正确：%d 字节", fieldName, actualSize)
			}
		}
	}

	// 验证 Server 结构体的嵌套字段大小
	if serverReport, ok := reports["Server"]; ok {
		t.Logf("\nServer 结构体字段大小:")
		expectedServerFields := map[string]int64{
			"Name":     16,
			"Host":     16,
			"Port":     4,
			"Database": expectedDatabaseSize,
			"Cache":    expectedCacheSize,
			"Timeout":  8,
			"Debug":    1,
		}

		for fieldName, expectedSize := range expectedServerFields {
			if actualSize, ok := serverReport.FieldSizes[fieldName]; !ok {
				t.Errorf("Server.%s 在 FieldSizes 中不存在", fieldName)
			} else if actualSize != expectedSize {
				t.Errorf("Server.%s 大小错误：期望 %d, 得到 %d",
					fieldName, expectedSize, actualSize)
			} else {
				t.Logf("✅ Server.%s 大小正确：%d 字节", fieldName, actualSize)
			}
		}
	}
}
