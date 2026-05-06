package optimizer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// findStructInPackage finds a struct in a loaded package (used in optimization phase)
func (o *Optimizer) findStructInPackage(pkg *packages.Package, structName string) (*types.Struct, string, error) {
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		for _, decl := range syntax.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if typeSpec.Name.Name != structName {
					continue
				}

				obj := pkg.TypesInfo.ObjectOf(typeSpec.Name)
				if obj == nil {
					continue
				}

				if named, ok := obj.Type().(*types.Named); ok {
					if st, ok := named.Underlying().(*types.Struct); ok {
						return st, filePath, nil
					}
				}
			}
		}
	}

	return nil, "", fmt.Errorf("struct %s not found in package", structName)
}

// NewOptimizer creates a new optimizer
func NewOptimizer(cfg *Config, analyzer *analyzer.Analyzer) *Optimizer {
	// Set defaults
	maxDepth := cfg.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 50 // default 50 levels
	}

	pkgWorkerLimit := cfg.PkgWorkerLimit
	if pkgWorkerLimit <= 0 {
		pkgWorkerLimit = 4 // default 4 concurrent packages to prevent OOM
	}

	return &Optimizer{
		config:           cfg,
		analyzer:         analyzer,
		optimized:        make(map[string]*StructInfo),
		processing:       make(map[string]bool),
		collecting:       make(map[string]bool),
		maxDepth:         maxDepth,
		methodIndex:      NewMethodIndex(),
		structQueue:      make([]*StructTask, 0),
		structByLevel:    make(map[int][]*StructTask),
		structByPkgLevel: make(map[int]map[string][]*StructTask),
		workerLimit:      10,             // struct-level concurrency limit
		pkgWorkerLimit:   pkgWorkerLimit, // package-level concurrency limit (prevent OOM)
		pkgCache:         make(map[string]*packages.Package),
		report: &Report{
			StructReports: make([]*StructReport, 0),
		},
	}
}

// Optimize runs the optimization (entry point).
// Two-phase processing:
//
//	Phase 1: Collect struct location info only (no package loading, no field analysis)
//	Phase 2: Optimize all collected structs in parallel (load packages on demand)
func (o *Optimizer) Optimize() (*Report, error) {
	o.Log(1, "Starting optimization...")
	o.Log(1, "Config: max depth=%d, timeout=%d seconds", o.maxDepth, o.config.Timeout)

	// Set up timeout
	done := make(chan struct{})
	var result *Report
	var err error

	go func() {
		defer close(done)
		result, err = o.optimizeInternal()
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		return result, err
	case <-time.After(time.Duration(o.config.Timeout) * time.Second):
		o.Log(0, "Error: optimization timed out (%d seconds)", o.config.Timeout)
		// Note: the goroutine will continue until completion, but the result will be discarded.
		// This is acceptable because the user already received a response after the timeout.
		return nil, fmt.Errorf("optimization timeout after %d seconds", o.config.Timeout)
	}
}

// optimizeInternal performs the actual optimization logic (two-phase)
func (o *Optimizer) optimizeInternal() (*Report, error) {
	// ==================== Phase 1: Collect structs ====================
	o.Log(1, "Phase 1/2: Collecting structs (file parsing only, no package loading)...")
	if o.config.StructName != "" {
		// Optimize a specific struct
		pkgPath, structName := analyzer.ParseStructName(o.config.StructName)
		if pkgPath == "" {
			return nil, fmt.Errorf("invalid struct name format: %s", o.config.StructName)
		}

		// Set the root struct
		o.report.RootStruct = o.config.StructName

		o.Log(1, "Collecting struct: %s.%s", pkgPath, structName)
		o.collectStructs(pkgPath, structName, "", 0, 0)
	} else if o.config.Package != "" {
		// Optimize all structs in a package
		o.Log(1, "Collecting package: %s", o.config.Package)
		
		var structs []analyzer.StructDef
		var err error
		
		if o.config.Recursive {
			// Recursively scan all sub-packages
			o.Log(1, "Recursive mode: scanning %s and all sub-packages", o.config.Package)
			structs, err = o.analyzer.FindAllStructsRecursive(o.config.Package)
		} else {
			// Scan only the current package
			structs, err = o.analyzer.FindAllStructs(o.config.Package)
		}
		
		if err != nil {
			return nil, err
		}

		for _, st := range structs {
			o.collectStructs(st.PkgPath, st.Name, st.File, 0, 0)
		}
	}

	o.Log(1, "Phase 1 complete: collected %d struct tasks", len(o.structQueue))

	// Print the list of collected structs
	o.Log(2, "Collected struct list:")
	for i, task := range o.structQueue {
		o.Log(3, "  [%d] %s.%s (level:%d, file:%s)",
			i+1, task.PkgPath, task.StructName, task.Level, filepath.Base(task.FilePath))
	}

	// ==================== Phase 2: Parallel optimization ====================
	o.Log(1, "Phase 2/2: Optimizing structs in parallel (loading packages on demand)...")
	o.processStructsParallel()

	// Generate report
	o.report.TotalStructs = len(o.optimized)
	o.report.OptimizedCount = 0
	o.report.SkippedCount = 0
	o.report.TotalOrigSize = 0
	o.report.TotalOptSize = 0

	for _, info := range o.optimized {
		if info.Skipped {
			o.report.SkippedCount++
		} else if info.OrigSize > info.OptSize {
			o.report.OptimizedCount++
			o.report.TotalSaved += info.OrigSize - info.OptSize
		}
	}

	isStructMode := o.report.RootStruct != ""

	for _, sr := range o.report.StructReports {
		// In -struct mode, count all depths: when allocating the root struct,
		// all nested child structs are also created in memory.
		// In -package mode, only count depth-0 (the package's own structs).
		if isStructMode || sr.Depth == 0 {
			o.report.TotalOrigSize += sr.OrigSize
			o.report.TotalOptSize += sr.OptSize
		}

		if isStructMode && sr.PkgPath+"."+sr.Name == o.report.RootStruct {
			o.report.RootStructSize = sr.OrigSize
			o.report.RootStructOptSize = sr.OptSize
		}
	}

	o.Log(1, "Optimization complete: %d structs processed, %d optimized, %d skipped, %d bytes saved",
		o.report.TotalStructs, o.report.OptimizedCount, o.report.SkippedCount, o.report.TotalSaved)

	return o.report, nil
}

// optimizeStruct optimizes a single struct (recursive)
func (o *Optimizer) optimizeStruct(pkgPath, structName, filePath string, depth int) (*StructInfo, error) {
	key := pkgPath + "." + structName

	// Check recursion depth limit
	if depth > o.maxDepth {
		o.Log(2, "Skipping struct (exceeded max depth %d): %s", o.maxDepth, key)
		info := &StructInfo{
			Name:       structName,
			PkgPath:    pkgPath,
			File:       filePath,
			Skipped:    true,
			SkipReason: fmt.Sprintf("Exceeded max recursion depth (%d)", o.maxDepth),
		}
		o.optimized[key] = info
		o.addReport(info, info.SkipReason, depth)
		return info, nil
	}

	// Check if already optimized (lock-protected)
	o.mu.Lock()
	if info, ok := o.optimized[key]; ok {
		o.mu.Unlock()
		o.Log(3, "Struct already processed: %s", key)
		return info, nil
	}

	// Detect circular reference: if already processing, there's a circular reference
	if o.processing[key] {
		o.Log(2, "Detected circular reference, skipping: %s", key)
		info := &StructInfo{
			Name:       structName,
			PkgPath:    pkgPath,
			File:       filePath,
			Skipped:    true,
			SkipReason: "Circular reference",
		}
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, "Circular reference", depth)
		return info, nil
	}

	// Mark as processing
	o.processing[key] = true
	o.mu.Unlock()

	defer func() {
		// After processing, remove the mark
		o.mu.Lock()
		delete(o.processing, key)
		o.mu.Unlock()
	}()

	// Check if it's a vendor package or third-party package (scan allowed when AllowExternalPkgs=true)
	if !o.config.AllowExternalPkgs && isVendorPackage(pkgPath) {
		o.Log(3, "Skipping vendor struct: %s", key)
		info := &StructInfo{
			Name:       structName,
			PkgPath:    pkgPath,
			File:       filePath,
			Skipped:    true,
			SkipReason: "Third-party package struct in vendor",
		}
		o.mu.Lock()
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, "Third-party package struct in vendor", depth)
		return info, nil
	}

	// Check if it's an internal project package (cross-package scan allowed when AllowExternalPkgs=true)
	if !o.config.AllowExternalPkgs && !o.isProjectPackage(pkgPath) {
		o.Log(3, "Skipping non-project internal package struct: %s", key)
		info := &StructInfo{
			Name:    structName,
			PkgPath: pkgPath,
			File:    filePath,
			Skipped: true,
			SkipReason: func() string {
				if isStandardLibrary(pkgPath) {
					return "Go standard library struct"
				}
				return "Non-project internal package struct"
			}(),
		}
		o.mu.Lock()
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, func() string {
			if isStandardLibrary(pkgPath) {
				return "Go standard library struct"
			}
			return "Non-project internal package struct"
		}(), depth)
		return info, nil
	}

	o.Log(2, "[%d] Processing struct: %s", depth, key)
	if filePath != "" {
		o.Log(3, "    File path: %s", filepath.Base(filePath))
	}

	// Optimization phase: prefer parsing files, do not load packages.
	// Only load packages when file parsing fails.
	var info *StructInfo
	var err error
	var st *types.Struct

	// Try parsing only the file (fast path)
	if filePath != "" {
		info, st, err = analyzeStructFromFile(filePath, structName, pkgPath)
		if err != nil {
			o.Log(3, "File parsing failed, loading package: %v", err)
		}
	}

	// File parsing failed, load the package (slow path)
	if info == nil {
		o.Log(2, "Loading package for type info: %s", pkgPath)
		pkg, err := o.analyzer.LoadPackage(pkgPath)
		if err != nil {
			o.Log(1, "Warning: failed to load package, skipping: %s (%v)", key, err)
			info = &StructInfo{
				Name:       structName,
				PkgPath:    pkgPath,
				File:       filePath,
				Skipped:    true,
				SkipReason: "Failed to load package: " + err.Error(),
			}
			o.mu.Lock()
			o.optimized[key] = info
			o.mu.Unlock()
			o.addReport(info, info.SkipReason, depth)
			return info, nil
		}

		// Find the struct in the package
		st, filePath, err = o.findStructInPackage(pkg, structName)
		if err != nil {
			o.Log(1, "Warning: failed to find struct, skipping: %s (%v)", key, err)
			info = &StructInfo{
				Name:       structName,
				PkgPath:    pkgPath,
				File:       filePath,
				Skipped:    true,
				SkipReason: "Lookup failed: " + err.Error(),
			}
			o.mu.Lock()
			o.optimized[key] = info
			o.mu.Unlock()
			o.addReport(info, info.SkipReason, depth)
			return info, nil
		}

		fieldAnalyzer := NewFieldAnalyzer(pkg.TypesInfo, pkg.Fset)

		info = fieldAnalyzer.AnalyzeStruct(st, structName, pkgPath, filePath)
		
		// Recalculate size using types.Sizes (consistent with unsafe.Sizeof)
		sizes := types.SizesFor("gc", "amd64")
		if sizes != nil {
			info.OrigSize = sizes.Sizeof(st)
		}
	}

	// Check if it should be skipped
	if skipReason := o.shouldSkip(info, key); skipReason != "" {
		o.Log(2, "Skipping struct: %s, reason: %s", key, skipReason)
		info.Skipped = true
		info.SkipReason = skipReason
		info.OptSize = info.OrigSize // skipped structs: optimized size equals original size
		o.mu.Lock()
		o.optimized[key] = info
		o.mu.Unlock()
		o.addReport(info, skipReason, depth)
		return info, nil
	}

	// Reorder fields (nested structs already handled in collection phase).
	// Note: ReorderFields uses estimated sizes internally to decide whether to reorder,
	// which may be inaccurate. We always get the sorted result, then use accurate sizes
	// below to decide whether to adopt it.
	sortedFields := ReorderFields(info.Fields, o.config.SortSameSize, o.config.ReservedFields)

	// Calculate the size after sorting (using accurate type info)
	sortedOptSize := CalcOptimizedSize(sortedFields, o.analyzer.GetTypesInfo())

	// Decide whether to adopt the sorted result: only if it saves memory
	if sortedOptSize < info.OrigSize {
		info.Fields = sortedFields
		info.OptSize = sortedOptSize
		info.OptOrder = extractFieldNamesFromInfo(sortedFields)
		info.Optimized = true

		o.Log(2, "Struct optimized: %s %d -> %d bytes (saved:%d)",
			key, info.OrigSize, info.OptSize, info.OrigSize-info.OptSize)
	} else {
		// Cannot save memory, keep original order, do not adopt new order.
		info.OptSize = info.OrigSize
		info.OptOrder = info.OrigOrder
		// info.Optimized remains false, will not trigger file rewrite
		o.Log(2, "Struct optimization not needed: %s", key)
	}

	o.mu.Lock()
	o.optimized[key] = info
	o.mu.Unlock()
	o.addReport(info, "", depth)

	return info, nil
}

// createSkippedInfo creates a skipped struct info entry
func (o *Optimizer) createSkippedInfo(key, filePath, reason string) *StructInfo {
	pkgPath, structName := analyzer.ParseStructName(key)
	return &StructInfo{
		Name:       structName,
		PkgPath:    pkgPath,
		File:       filePath,
		Skipped:    true,
		SkipReason: reason,
	}
}

// addReport adds a report entry
func (o *Optimizer) addReport(info *StructInfo, skipReason string, depth int) {
	// Build field type map and field size map.
	// Note: key format is consistent with OrigOrder/OptOrder (plain field name, embedded fields use type name).
	fieldTypes := make(map[string]string)
	fieldSizes := make(map[string]int64)
	hasEmbed := false
	for _, f := range info.Fields {
		// key is consistent with extractFieldNames
		var key string
		if f.Name != "" {
			// Named field: use the field name
			key = f.Name
		} else {
			// Embedded field: use the type name
			key = f.TypeName
		}
		fieldTypes[key] = f.TypeName
		fieldSizes[key] = f.Size

		// Check if it is an embedded field.
		// Condition: IsEmbed is true, and the type is a struct type (not a basic type).
		if f.IsEmbed && !isBasicType(f.TypeName) {
			hasEmbed = true
		}
	}

	report := &StructReport{
		Name:       info.Name,
		PkgPath:    info.PkgPath,
		File:       info.File,
		OrigSize:   info.OrigSize,
		OptSize:    info.OptSize,
		Saved:      info.OrigSize - info.OptSize,
		OrigFields: info.OrigOrder,
		OptFields:  info.OptOrder,
		FieldTypes: fieldTypes,
		FieldSizes: fieldSizes,
		Skipped:    info.Skipped,
		SkipReason: skipReason,
		Depth:      depth,
		HasEmbed:   hasEmbed,
	}

	if info.OptSize == 0 && info.OrigSize == 0 {
		report.Saved = 0
	}

	// Lock-protected concurrent write to report
	o.mu.Lock()
	o.report.StructReports = append(o.report.StructReports, report)
	o.mu.Unlock()
}

// GetOptimized returns the optimized struct info map (thread-safe)
func (o *Optimizer) GetOptimized() map[string]*StructInfo {
	o.mu.Lock()
	result := make(map[string]*StructInfo, len(o.optimized))
	for k, v := range o.optimized {
		result[k] = v
	}
	o.mu.Unlock()
	return result
}

// GetReport returns the report (thread-safe)
func (o *Optimizer) GetReport() *Report {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.report
}

// Log outputs a log message
func (o *Optimizer) Log(level int, format string, args ...interface{}) {
	if level <= o.config.Verbose {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		levelPrefix := ""
		switch level {
		case 1:
			levelPrefix = "[INFO] "
		case 2:
			levelPrefix = "[DEBUG]"
		case 3:
			levelPrefix = "[TRACE]"
		}
		fmt.Printf("%s %s "+format+"\n", append([]interface{}{timestamp, levelPrefix}, args...)...)
	}
}

// isBasicType checks if the type name is a basic type
func isBasicType(typeName string) bool {
	basicTypes := map[string]bool{
		"bool": true, "string": true,
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
		"byte": true, "rune": true, "uintptr": true,
	}
	return basicTypes[typeName]
}

// extractFieldNamesFromInfo extracts field names from a FieldInfo slice
func extractFieldNamesFromInfo(fields []FieldInfo) []string {
	var names []string
	for _, f := range fields {
		if f.Name != "" {
			names = append(names, f.Name)
		} else {
			// Embedded field uses type name
			names = append(names, f.TypeName)
		}
	}
	return names
}
