package optimizer

import (
	"testing"
)

// TestMethodIndexBasic 测试 MethodIndex 基础功能
func TestMethodIndexBasic(t *testing.T) {
	mi := NewMethodIndex()

	// 测试 structoptimizer 自身的 analyzer 包
	pkgPath := "github.com/gamelife1314/structoptimizer/analyzer"
	
	// analyzer.Config 应该没有方法
	if mi.HasMethod(pkgPath, "Config", "LoadPackage") {
		t.Error("analyzer.Config should not have LoadPackage method")
	}

	// analyzer.Analyzer 有 Log 方法
	if !mi.HasMethod(pkgPath, "Analyzer", "Log") {
		t.Error("analyzer.Analyzer should have Log method")
	}

	// 测试通配符
	if !mi.HasMethod(pkgPath, "Analyzer", "L*") {
		t.Error("analyzer.Analyzer should match L* (Log)")
	}

	if !mi.HasMethod(pkgPath, "Analyzer", "*g") {
		t.Error("analyzer.Analyzer should match *g (Log, LoadPackage)")
	}

	// 测试不存在的方法
	if mi.HasMethod(pkgPath, "Analyzer", "NotExist") {
		t.Error("analyzer.Analyzer should not have NotExist method")
	}
}

// TestMethodIndexCaching 测试缓存
func TestMethodIndexCaching(t *testing.T) {
	mi := NewMethodIndex()
	pkgPath := "github.com/gamelife1314/structoptimizer/optimizer"
	
	// 第一次查询
	mi.HasMethod(pkgPath, "Optimizer", "Optimize")
	
	// 检查缓存是否建立
	mi.mu.RLock()
	_, exists := mi.cache[pkgPath]
	mi.mu.RUnlock()
	
	if !exists {
		t.Error("Cache should be populated after first query")
	}
}
