package optimizer

import (
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
		
		// 按包并行处理
		o.processByPackageParallel(level, pkgTasks)
	}
}

// processByPackageParallel 按包并行处理同一层级的结构体
// 每个包一个 goroutine，避免不同 goroutine 之间的竞争
func (o *Optimizer) processByPackageParallel(level int, pkgTasks map[string][]*StructTask) {
	var wg sync.WaitGroup
	
	for pkgPath, tasks := range pkgTasks {
		wg.Add(1)
		
		go func(pkg string, taskList []*StructTask) {
			defer wg.Done()
			
			// panic 恢复
			defer func() {
				if r := recover(); r != nil {
					o.Log(0, "处理包 %s 时发生 panic: %v", pkg, r)
				}
			}()
			
			o.Log(3, "处理包：%s (%d 个结构体)", pkg, len(taskList))
			
			// 串行处理包内的结构体（避免包内竞争）
			for _, task := range taskList {
				key := task.PkgPath + "." + task.StructName
				o.Log(3, "优化结构体：%s (层级:%d)", key, task.Level)
				o.optimizeStruct(task.PkgPath, task.StructName, task.FilePath, task.Depth)
			}
		}(pkgPath, tasks)
	}
	
	wg.Wait()
}
