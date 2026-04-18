package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// TestFindAllStructsRecursive 测试递归扫描包功能
func TestFindAllStructsRecursive(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "recursive_scan_test")

	// 检查测试目录是否存在
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("测试目录不存在，跳过测试")
	}

	// 创建分析器
	cfg := &analyzer.Config{
		Package:     "example.com/recursivetest/pkg",
		TargetDir:   testDir,
		ProjectType: "gomod",
		Verbose:     2,
	}
	anlz := analyzer.NewAnalyzer(cfg)

	// 测试递归扫描
	structs, err := anlz.FindAllStructsRecursive("example.com/recursivetest/pkg")
	if err != nil {
		t.Fatalf("递归扫描失败：%v", err)
	}

	t.Logf("递归扫描找到 %d 个结构体", len(structs))

	// 验证找到的结构体
	expectedStructs := map[string]bool{
		"RootConfig":       false,
		"ConfigWithNested": false,
		"UserModel":        false,
		"ProfileModel":     false,
		"Handler":          false,
		"UserService":      false,
		"Helper":           false,
	}

	for _, st := range structs {
		if _, ok := expectedStructs[st.Name]; ok {
			expectedStructs[st.Name] = true
			t.Logf("✅ 找到结构体：%s.%s (文件：%s)", st.PkgPath, st.Name, filepath.Base(st.File))
		} else {
			t.Logf("找到额外结构体：%s.%s", st.PkgPath, st.Name)
		}
	}

	// 验证所有预期的结构体都找到了
	for name, found := range expectedStructs {
		if !found {
			t.Errorf("未找到预期的结构体：%s", name)
		}
	}

	// 验证包路径
	pkgPaths := make(map[string]bool)
	for _, st := range structs {
		pkgPaths[st.PkgPath] = true
	}

	expectedPkgs := []string{
		"example.com/recursivetest/pkg",
		"example.com/recursivetest/pkg/models",
		"example.com/recursivetest/pkg/api",
		"example.com/recursivetest/pkg/utils",
	}

	for _, pkg := range expectedPkgs {
		if !pkgPaths[pkg] {
			t.Errorf("未扫描到预期的包：%s", pkg)
		}
	}

	if len(structs) < 7 {
		t.Errorf("期望至少找到 7 个结构体，实际找到 %d 个", len(structs))
	}

	t.Log("✅ 递归扫描测试通过")
}

// TestFindAllStructsNonRecursive 测试非递归扫描（只扫描当前包）
func TestFindAllStructsNonRecursive(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "recursive_scan_test")

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("测试目录不存在，跳过测试")
	}

	cfg := &analyzer.Config{
		Package:     "example.com/recursivetest/pkg",
		TargetDir:   testDir,
		ProjectType: "gomod",
		Verbose:     2,
	}
	anlz := analyzer.NewAnalyzer(cfg)

	// 测试非递归扫描
	structs, err := anlz.FindAllStructs("example.com/recursivetest/pkg")
	if err != nil {
		t.Fatalf("扫描失败：%v", err)
	}

	t.Logf("非递归扫描找到 %d 个结构体", len(structs))

	// 非递归扫描应该只找到 pkg 包中的结构体
	if len(structs) < 2 {
		t.Errorf("非递归扫描应该至少找到 2 个结构体，实际找到 %d 个", len(structs))
	}

	// 验证找到的是 pkg 包的结构体
	for _, st := range structs {
		if st.PkgPath != "example.com/recursivetest/pkg" {
			t.Errorf("非递归扫描找到其他包的结构体：%s", st.PkgPath)
		}
	}

	t.Log("✅ 非递归扫描测试通过")
}
