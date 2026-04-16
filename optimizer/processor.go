package optimizer

import (
	"runtime"
	"runtime/debug"
	"sync"
)

// processStructsParallel 按层级并行处理结构体优化
// 从叶子节点（最底层）开始，逐层向上处理
func (o *Optimizer) processStructsParallel() {
	if len(o.structQueue) == 0 {
		return
	}

	// 按层级和包分组
	o.structByPkgLevel = make(map[int]map[string][]*StructTask)
	for _, task := range o.structQueue {
		if _, ok := o.structByPkgLevel[task.Level]; !ok {
			o.structByPkgLevel[task.Level] = make(map[string][]*StructTask)
		}
		o.structByPkgLevel[task.Level][task.PkgPath] = append(o.structByPkgLevel[task.Level][task.PkgPath], task)
	}

	// 找出最大层级
	maxLevel := 0
	for level := range o.structByPkgLevel {
		if level > maxLevel {
			maxLevel = level
		}
	}

	o.Log(2, "结构体层级分布：共 %d 层", maxLevel+1)

	// 从叶子节点（最大层级）开始，逐层向上处理
	for level := maxLevel; level >= 0; level-- {
		pkgTasks := o.structByPkgLevel[level]

		// 统计本层级信息
		totalStructs := 0
		for _, tasks := range pkgTasks {
			totalStructs += len(tasks)
		}
		o.Log(2, "处理第 %d 层，共 %d 个包，%d 个结构体", level, len(pkgTasks), totalStructs)

		// 处理前记录内存状态
		if o.memGuard != nil && o.config.Verbose >= 2 {
			o.Log(3, "第 %d 层处理前内存状态：", level)
			o.memGuard.LogMemoryStats(func(format string, args ...interface{}) {
				o.Log(3, "  "+format, args...)
			})
		}

		// 按包并行处理
		o.processByPackageParallel(level, pkgTasks)

		// 每层处理后强制 GC，释放内存
		if level > 0 {
			// 智能 GC：根据内存使用情况决定
			if o.memGuard != nil {
				stats := o.memGuard.GetMemoryStats()
				if stats.AllocMB > 100 { // 超过 100MB 才强制 GC
					o.Log(3, "第 %d 层处理完成，执行 GC（当前内存：%dMB）...", level, stats.AllocMB)
					runtime.GC()
					debug.FreeOSMemory()
					
					// GC 后记录内存状态
					if o.config.Verbose >= 3 {
						o.Log(3, "GC 后内存状态：")
						o.memGuard.LogMemoryStats(func(format string, args ...interface{}) {
							o.Log(3, "  "+format, args...)
						})
					}
				} else {
					o.Log(3, "第 %d 层处理完成（内存：%dMB，不需要 GC）", level, stats.AllocMB)
				}
			} else {
				o.Log(3, "第 %d 层处理完成，执行 GC...", level)
				runtime.GC()
			}
		}
	}

	// 最后再执行一次 GC
	runtime.GC()
	debug.FreeOSMemory()
	if o.memGuard != nil {
		o.Log(2, "所有层级处理完成，执行最终 GC：")
		o.memGuard.LogMemoryStats(func(format string, args ...interface{}) {
			o.Log(2, "  "+format, args...)
		})
	} else {
		o.Log(2, "所有层级处理完成，执行最终 GC")
	}
}

// processByPackageParallel 按包并行处理同一层级的结构体
// 每个包一个 goroutine，避免不同 goroutine 之间的竞争
// 使用信号量限制并发包数量，防止 OOM
func (o *Optimizer) processByPackageParallel(level int, pkgTasks map[string][]*StructTask) {
	var wg sync.WaitGroup
	pkgSem := make(chan struct{}, o.pkgWorkerLimit) // 包级别信号量

	for pkgPath, tasks := range pkgTasks {
		pkgSem <- struct{}{} // 获取信号量
		wg.Add(1)

		go func(pkg string, taskList []*StructTask) {
			defer wg.Done()
			defer func() { <-pkgSem }() // 释放信号量

			// panic 恢复
			defer func() {
				if r := recover(); r != nil {
					o.Log(0, "处理包 %s 时发生 panic: %v", pkg, r)
				}
			}()

			o.Log(3, "处理包：%s (%d 个结构体)", pkg, len(taskList))

			// 串行处理包内的结构体（避免包内竞争）
			for _, task := range taskList {
				// 处理前内存检查
				if o.memGuard != nil {
					if err := o.memGuard.CheckMemory(); err != nil {
						o.Log(1, "警告：内存不足，跳过结构体：%s.%s (%v)", 
							task.PkgPath, task.StructName, err)
						continue
					}
				}

				key := task.PkgPath + "." + task.StructName
				o.Log(3, "优化结构体：%s (层级:%d)", key, task.Level)
				o.optimizeStruct(task.PkgPath, task.StructName, task.FilePath, task.Depth)
				
				// 每处理一个结构体后检查内存
				if o.memGuard != nil && len(taskList) > 10 {
					// 大包每处理 10 个结构体后检查一次
					if idx := indexOfTask(taskList, task); (idx+1)%10 == 0 {
						stats := o.memGuard.GetMemoryStats()
						o.Log(3, "包 %s 处理进度 %d/%d，内存：%dMB", 
							pkg, idx+1, len(taskList), stats.AllocMB)
					}
				}
			}
		}(pkgPath, tasks)
	}

	wg.Wait()
}

// indexOfTask 获取任务在列表中的索引
func indexOfTask(tasks []*StructTask, target *StructTask) int {
	for i, t := range tasks {
		if t == target {
			return i
		}
	}
	return -1
}
