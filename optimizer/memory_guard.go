package optimizer

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// MemoryGuard 内存保护器
type MemoryGuard struct {
	mu                sync.Mutex
	maxMemoryMB       int           // 最大内存限制（MB）
	checkInterval     time.Duration // 检查间隔
	lastCheck         time.Time
	gcThresholdMB     int  // 触发 GC 的阈值（MB）
	forceGCThresholdMB int  // 强制 GC 的阈值（MB）
	enableAutoGC      bool // 是否启用自动 GC
}

// MemoryStats 内存统计信息
type MemoryStats struct {
	AllocMB      uint64    // 当前分配的内存（MB）
	TotalAllocMB uint64    // 累计分配的内存（MB）
	SysMB        uint64    // 从系统获取的内存（MB）
	NumGC        uint32    // GC 次数
	GCPercent    float64   // GC 目标百分比
	Timestamp    time.Time // 时间戳
}

// NewMemoryGuard 创建内存保护器
func NewMemoryGuard(maxMemoryMB int, enableAutoGC bool) *MemoryGuard {
	if maxMemoryMB <= 0 {
		maxMemoryMB = 512 // 默认 512MB
	}
	
	return &MemoryGuard{
		maxMemoryMB:       maxMemoryMB,
		checkInterval:     100 * time.Millisecond,
		gcThresholdMB:     maxMemoryMB * 70 / 100,     // 70% 时触发 GC
		forceGCThresholdMB: maxMemoryMB * 85 / 100,    // 85% 时强制 GC
		enableAutoGC:      enableAutoGC,
	}
}

// GetMemoryStats 获取当前内存统计信息
func (mg *MemoryGuard) GetMemoryStats() *MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &MemoryStats{
		AllocMB:      m.Alloc / 1024 / 1024,
		TotalAllocMB: m.TotalAlloc / 1024 / 1024,
		SysMB:        m.Sys / 1024 / 1024,
		NumGC:        m.NumGC,
		GCPercent:    float64(debug.SetGCPercent(-1)),
		Timestamp:    time.Now(),
	}
}

// CheckMemory 检查内存使用情况，必要时触发 GC
func (mg *MemoryGuard) CheckMemory() error {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	
	// 限制检查频率
	now := time.Now()
	if now.Sub(mg.lastCheck) < mg.checkInterval {
		return nil
	}
	mg.lastCheck = now
	
	stats := mg.GetMemoryStats()
	allocMB := stats.AllocMB
	
	// 检查是否超过强制 GC 阈值
	if int(allocMB) >= mg.forceGCThresholdMB {
		return mg.forceGC(stats)
	}
	
	// 检查是否超过普通 GC 阈值
	if mg.enableAutoGC && int(allocMB) >= mg.gcThresholdMB {
		mg.triggerGC(stats)
	}
	
	// 检查是否超过最大限制
	if int(allocMB) >= mg.maxMemoryMB {
		return fmt.Errorf("内存使用超过限制：%dMB >= %dMB", allocMB, mg.maxMemoryMB)
	}
	
	return nil
}

// triggerGC 触发 GC
func (mg *MemoryGuard) triggerGC(stats *MemoryStats) {
	mg.mu.Unlock()
	defer mg.mu.Lock()
	
	runtime.GC()
	newStats := mg.GetMemoryStats()
	
	// 日志记录由调用方处理
	_ = newStats
}

// forceGC 强制 GC 并降低 GC 目标
func (mg *MemoryGuard) forceGC(stats *MemoryStats) error {
	mg.mu.Unlock()
	defer mg.mu.Lock()
	
	// 降低 GC 目标百分比，更积极地回收
	oldGCPercent := debug.SetGCPercent(50)
	defer func() {
		// 恢复原来的 GC 目标
		debug.SetGCPercent(int(oldGCPercent))
	}()
	
	// 强制 GC
	runtime.GC()
	
	// 释放调试内存
	debug.FreeOSMemory()
	
	newStats := mg.GetMemoryStats()
	
	// 如果仍然超过限制，返回错误
	if int(newStats.AllocMB) >= mg.maxMemoryMB {
		return fmt.Errorf("强制 GC 后内存仍然超标：%dMB >= %dMB", newStats.AllocMB, mg.maxMemoryMB)
	}
	
	return nil
}

// SafeLoadPackage 安全地加载包，带内存检查
func (mg *MemoryGuard) SafeLoadPackage(loadFunc func() error, pkgPath string) error {
	// 加载前检查
	if err := mg.CheckMemory(); err != nil {
		return fmt.Errorf("加载包前内存检查失败：%s (%v)", pkgPath, err)
	}
	
	// 加载包
	if err := loadFunc(); err != nil {
		return err
	}
	
	// 加载后检查
	return mg.CheckMemory()
}

// LogMemoryStats 记录内存统计信息（供日志使用）
func (mg *MemoryGuard) LogMemoryStats(logFunc func(format string, args ...interface{})) {
	stats := mg.GetMemoryStats()
	logFunc("内存状态：Alloc=%dMB, Sys=%dMB, TotalAlloc=%dMB, NumGC=%d, GCPercent=%.0f%%",
		stats.AllocMB, stats.SysMB, stats.TotalAllocMB, stats.NumGC, stats.GCPercent)
}

// SetMaxMemory 设置最大内存限制
func (mg *MemoryGuard) SetMaxMemory(maxMB int) {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	
	mg.maxMemoryMB = maxMB
	mg.gcThresholdMB = maxMB * 70 / 100
	mg.forceGCThresholdMB = maxMB * 85 / 100
}

// EnableAutoGC 启用/禁用自动 GC
func (mg *MemoryGuard) EnableAutoGC(enabled bool) {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	mg.enableAutoGC = enabled
}
