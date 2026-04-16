package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// TestGopathSubPackageImport 测试用户场景：-pkg-scope 下的子包导入
// 模拟场景：
//   -struct github.com/gamelife1314/anylyzer/writer.Config
//   -pkg-scope github.com/gamelife1314/anylyzer
func TestGopathSubPackageImport(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_subpkg_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构
	// $GOPATH/src/github.com/gamelife1314/anylyzer/
	//   writer/
	//     config.go
	//   parser/
	//     parser.go

	projectDir := filepath.Join(tmpDir, "src", "github.com", "gamelife1314", "anylyzer")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("创建项目目录失败：%v", err)
	}

	// 创建主包文件（可选，有些项目主包为空）
	docFile := filepath.Join(projectDir, "doc.go")
	docContent := `// Package anylyzer 提供代码分析功能
package anylyzer
`
	if err := os.WriteFile(docFile, []byte(docContent), 0644); err != nil {
		t.Fatalf("写入 doc.go 失败：%v", err)
	}

	// 创建 writer 包
	writerDir := filepath.Join(projectDir, "writer")
	if err := os.MkdirAll(writerDir, 0755); err != nil {
		t.Fatalf("创建 writer 目录失败：%v", err)
	}

	configFile := filepath.Join(writerDir, "config.go")
	configContent := `package writer

// Config 写入器配置
type Config struct {
	BufferSize int
	Flush      bool
	Output     string
}

// NewConfig 创建配置
func NewConfig() *Config {
	return &Config{
		BufferSize: 4096,
		Flush:      true,
		Output:     "stdout",
	}
}
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入 config.go 失败：%v", err)
	}

	// 创建 parser 包（引用 writer 包）
	parserDir := filepath.Join(projectDir, "parser")
	if err := os.MkdirAll(parserDir, 0755); err != nil {
		t.Fatalf("创建 parser 目录失败：%v", err)
	}

	parserFile := filepath.Join(parserDir, "parser.go")
	parserContent := `package parser

import "github.com/gamelife1314/anylyzer/writer"

// Parser 解析器
type Parser struct {
	config *writer.Config
}

// NewParser 创建解析器
func NewParser(cfg *writer.Config) *Parser {
	return &Parser{config: cfg}
}

// GetConfig 获取配置
func (p *Parser) GetConfig() *writer.Config {
	return p.config
}
`
	if err := os.WriteFile(parserFile, []byte(parserContent), 0644); err != nil {
		t.Fatalf("写入 parser.go 失败：%v", err)
	}

	// 创建分析器（模拟用户的配置）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "github.com/gamelife1314/anylyzer", // -pkg-scope
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     3, // 启用详细日志
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试 1: 加载主包
	t.Run("LoadRootPackage", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("github.com/gamelife1314/anylyzer")
		if err != nil {
			t.Fatalf("加载主包失败：%v", err)
		}

		t.Logf("成功加载主包：%s", pkg.PkgPath)
	})

	// 测试 2: 加载 writer 子包（关键测试）
	t.Run("LoadWriterSubPackage", func(t *testing.T) {
		// 这是 -pkg-scope 下的子包，应该从 GOPATH/src 加载
		pkg, err := anlz.LoadPackage("github.com/gamelife1314/anylyzer/writer")
		if err != nil {
			t.Fatalf("加载 writer 子包失败：%v", err)
		}

		if pkg.Name != "writer" {
			t.Errorf("包名称错误：期望 writer，实际 %s", pkg.Name)
		}

		// 验证 Config 类型存在
		obj := pkg.Types.Scope().Lookup("Config")
		if obj == nil {
			t.Fatal("未找到 Config 类型")
		}

		t.Logf("成功加载 writer 子包，找到 Config 类型")
	})

	// 测试 3: 加载 parser 子包（引用 writer 子包）
	t.Run("LoadParserSubPackage", func(t *testing.T) {
		// parser 引用 writer，这是同项目下的跨包引用
		pkg, err := anlz.LoadPackage("github.com/gamelife1314/anylyzer/parser")
		if err != nil {
			t.Fatalf("加载 parser 子包失败：%v", err)
		}

		if pkg.Name != "parser" {
			t.Errorf("包名称错误：期望 parser，实际 %s", pkg.Name)
		}

		// 验证 Parser 类型存在
		obj := pkg.Types.Scope().Lookup("Parser")
		if obj == nil {
			t.Fatal("未找到 Parser 类型")
		}

		t.Logf("成功加载 parser 子包，找到 Parser 类型")
	})
}
