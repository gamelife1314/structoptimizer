package optimizer

import (
	"runtime"
	"testing"
	"time"
)

// TestMemoryGuardBasic 测试内存保护器基本功能
func TestMemoryGuardBasic(t *testing.T) {
	mg := NewMemoryGuard(512, true)
	if mg == nil {
		t.Fatal("NewMemoryGuard 返回 nil")
	}

	// 测试获取内存统计
	stats := mg.GetMemoryStats()
	if stats.AllocMB == 0 && stats.SysMB == 0 {
		t.Error("内存统计信息异常")
	}

	t.Logf("初始内存状态：Alloc=%dMB, Sys=%dMB, NumGC=%d",
		stats.AllocMB, stats.SysMB, stats.NumGC)
}

// TestMemoryGuardThreshold 测试 GC 阈值触发
func TestMemoryGuardThreshold(t *testing.T) {
	// 设置很小的限制，方便测试
	mg := NewMemoryGuard(100, true) // 100MB
	mg.gcThresholdMB = 50
	mg.forceGCThresholdMB = 80

	// 正常情况应该不触发 GC
	err := mg.CheckMemory()
	if err != nil {
		t.Errorf("正常情况下不应该报错：%v", err)
	}
}

// TestMemoryGuardAutoGC 测试自动 GC
func TestMemoryGuardAutoGC(t *testing.T) {
	mg := NewMemoryGuard(100, true)

	// 分配一些内存
	data := make([]byte, 10*1024*1024) // 10MB
	runtime.KeepAlive(data)

	// 检查内存
	err := mg.CheckMemory()
	if err != nil {
		t.Errorf("内存检查失败：%v", err)
	}

	stats := mg.GetMemoryStats()
	t.Logf("分配 10MB 后：Alloc=%dMB", stats.AllocMB)
}

// TestMemoryGuardLogStats 测试日志输出
func TestMemoryGuardLogStats(t *testing.T) {
	mg := NewMemoryGuard(512, true)

	var logOutput string
	mg.LogMemoryStats(func(format string, args ...interface{}) {
		logOutput = format
	})

	if logOutput == "" {
		t.Error("日志输出为空")
	}

	t.Logf("日志输出：%s", logOutput)
}

// TestMemoryGuardSetMaxMemory 测试动态调整最大内存
func TestMemoryGuardSetMaxMemory(t *testing.T) {
	mg := NewMemoryGuard(512, true)

	// 调整最大内存
	mg.SetMaxMemory(256)
	if mg.maxMemoryMB != 256 {
		t.Errorf("最大内存设置失败：期望 256，实际 %d", mg.maxMemoryMB)
	}

	// 阈值应该自动调整
	expectedGCThreshold := 256 * 70 / 100 // 179
	expectedForceGCThreshold := 256 * 85 / 100 // 217

	if mg.gcThresholdMB != expectedGCThreshold {
		t.Errorf("GC 阈值不正确：期望 %d，实际 %d", expectedGCThreshold, mg.gcThresholdMB)
	}
	if mg.forceGCThresholdMB != expectedForceGCThreshold {
		t.Errorf("强制 GC 阈值不正确：期望 %d，实际 %d", expectedForceGCThreshold, mg.forceGCThresholdMB)
	}
}

// TestMemoryGuardEnableAutoGC 测试启用/禁用自动 GC
func TestMemoryGuardEnableAutoGC(t *testing.T) {
	mg := NewMemoryGuard(512, true)
	if !mg.enableAutoGC {
		t.Error("自动 GC 应该默认启用")
	}

	mg.EnableAutoGC(false)
	if mg.enableAutoGC {
		t.Error("自动 GC 应该被禁用")
	}

	mg.EnableAutoGC(true)
	if !mg.enableAutoGC {
		t.Error("自动 GC 应该重新启用")
	}
}

// TestMemoryGuardCheckInterval 测试检查间隔
func TestMemoryGuardCheckInterval(t *testing.T) {
	mg := NewMemoryGuard(512, true)
	mg.checkInterval = 10 * time.Millisecond // 很短的间隔

	// 连续调用应该被节流
	start := time.Now()
	for i := 0; i < 10; i++ {
		mg.CheckMemory()
	}
	elapsed := time.Since(start)

	// 应该很快完成（因为节流）
	if elapsed > 100*time.Millisecond {
		t.Errorf("检查间隔节流失败：耗时 %v", elapsed)
	}
}

// TestMemoryGuardDefaultValues 测试默认值
func TestMemoryGuardDefaultValues(t *testing.T) {
	mg := NewMemoryGuard(0, true) // 使用默认值

	if mg.maxMemoryMB != 512 {
		t.Errorf("默认最大内存不正确：期望 512，实际 %d", mg.maxMemoryMB)
	}

	if mg.gcThresholdMB != 512*70/100 {
		t.Errorf("默认 GC 阈值不正确：期望 %d，实际 %d", 512*70/100, mg.gcThresholdMB)
	}

	if mg.forceGCThresholdMB != 512*85/100 {
		t.Errorf("默认强制 GC 阈值不正确：期望 %d，实际 %d", 512*85/100, mg.forceGCThresholdMB)
	}
}
