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

	// 按层级分组
	for _, task := range o.structQueue {
		o.structByLevel[task.Level] = append(o.structByLevel[task.Level], task)
	}

	// 找出最大层级
	maxLevel := 0
	for level := range o.structByLevel {
		if level > maxLevel {
			maxLevel = level
		}
	}

	o.Log(2, "结构体层级分布：共 %d 层", maxLevel+1)

	// 从叶子节点（最大层级）开始，逐层向上处理
	for level := maxLevel; level >= 0; level-- {
		tasks := o.structByLevel[level]
		o.Log(2, "处理第 %d 层，共 %d 个结构体", level, len(tasks))
		o.processLevelParallel(tasks)
	}
}

// processLevelParallel 并行处理同一层级的结构体
func (o *Optimizer) processLevelParallel(tasks []*StructTask) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, o.workerLimit) // 信号量限制并发数

	for _, task := range tasks {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(t *StructTask) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// panic 恢复
			defer func() {
				if r := recover(); r != nil {
					o.Log(0, "处理结构体时发生 panic：%s.%s: %v", t.PkgPath, t.StructName, r)
				}
			}()

			key := t.PkgPath + "." + t.StructName
			o.Log(3, "优化结构体：%s (层级:%d)", key, t.Level)
			o.optimizeStruct(t.PkgPath, t.StructName, t.FilePath, t.Depth)
		}(task)
	}

	wg.Wait()
}
