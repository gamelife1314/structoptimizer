package optimizer

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
)

// processStructsParallel processes struct optimization in parallel by level.
// Starts from leaf nodes (deepest level) and works upward level by level.
func (o *Optimizer) processStructsParallel() {
	if len(o.structQueue) == 0 {
		return
	}

	// Group by level and package
	o.structByPkgLevel = make(map[int]map[string][]*StructTask)
	for _, task := range o.structQueue {
		if _, ok := o.structByPkgLevel[task.Level]; !ok {
			o.structByPkgLevel[task.Level] = make(map[string][]*StructTask)
		}
		o.structByPkgLevel[task.Level][task.PkgPath] = append(o.structByPkgLevel[task.Level][task.PkgPath], task)
	}

	// Find the maximum level
	maxLevel := 0
	for level := range o.structByPkgLevel {
		if level > maxLevel {
			maxLevel = level
		}
	}

	o.Log(2, "Struct level distribution: %d levels total", maxLevel+1)

	// Start from leaf nodes (max level) and work upward level by level
	for level := maxLevel; level >= 0; level-- {
		pkgTasks := o.structByPkgLevel[level]

		// Collect stats for this level
		totalStructs := 0
		for _, tasks := range pkgTasks {
			totalStructs += len(tasks)
		}
		o.Log(2, "Processing level %d, %d packages, %d structs", level, len(pkgTasks), totalStructs)

		// Process by package in parallel
		o.processByPackageParallel(level, pkgTasks)

		// Force GC after each level to free memory
		if level > 0 {
			o.Log(3, "Level %d processing complete, running GC...", level)
			runtime.GC()
		}
	}

	// Run one final GC
	runtime.GC()
	o.Log(2, "All levels processed, running final GC")
}

// processByPackageParallel processes structs at the same level in parallel by package.
// One goroutine per package, avoiding contention between different goroutines.
// Uses a semaphore to limit concurrent packages, preventing OOM.
func (o *Optimizer) processByPackageParallel(level int, pkgTasks map[string][]*StructTask) {
	var wg sync.WaitGroup
	pkgSem := make(chan struct{}, o.pkgWorkerLimit) // package-level semaphore

	for pkgPath, tasks := range pkgTasks {
		pkgSem <- struct{}{} // acquire semaphore
		wg.Add(1)

		go func(pkg string, taskList []*StructTask) {
			defer wg.Done()
			defer func() { <-pkgSem }() // release semaphore

			// Panic recovery
			defer func() {
				if r := recover(); r != nil {
					// Record the panic message and full stack trace
					stack := debug.Stack()
					o.Log(0, "Panic processing package %s: %v\nStack trace:\n%s", pkg, r, stack)
					// Mark all remaining structs in this package as skipped
					for _, task := range taskList {
						key := task.PkgPath + "." + task.StructName
						o.mu.Lock()
						if _, exists := o.optimized[key]; !exists {
							o.optimized[key] = &StructInfo{
								Name:       task.StructName,
								PkgPath:    task.PkgPath,
								File:       task.FilePath,
								Skipped:    true,
								SkipReason: fmt.Sprintf("Panic during package processing: %v", r),
							}
						}
						o.mu.Unlock()
					}
				}
			}()

			o.Log(3, "Processing package: %s (%d structs)", pkg, len(taskList))

			// Process structs within the package serially (avoid intra-package contention)
			for _, task := range taskList {
				key := task.PkgPath + "." + task.StructName
				o.Log(3, "Optimizing struct: %s (level:%d)", key, task.Level)
				o.optimizeStruct(task.PkgPath, task.StructName, task.FilePath, task.Depth, task.ParentKey)
			}
		}(pkgPath, tasks)
	}

	wg.Wait()
}
