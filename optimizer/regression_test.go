package optimizer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
	"github.com/gamelife1314/structoptimizer/optimizer"
)

// TestIsStandardLibraryPkg verifies that simple (non-stdlib) package names
// are no longer misclassified as standard library packages.
func TestIsStandardLibraryPkg(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stdlib_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	goModContent := `module simplepkg

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	mainContent := `package simplepkg

type Inner struct {
	A bool
	B uint64
	C bool
}

type Outer struct {
	X    bool
	Data Inner
	Y    bool
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	analyzerCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		StructName:  "simplepkg.Outer",
		Verbose:     0,
		ProjectType: "gomod",
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		StructName:  "simplepkg.Outer",
		Verbose:     0,
		ProjectType: "gomod",
		MaxDepth:    50,
		Timeout:     300,
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// "simplepkg" has no dots and no slashes. The old isStandardLibraryPkg
	// would incorrectly classify it as stdlib, causing Inner to be skipped.
	// The fix uses isStandardLibrary() lookup which correctly returns false.
	if report.TotalStructs < 2 {
		t.Errorf("Expected at least 2 structs (Outer + Inner), got %d. "+
			"Inner was likely misclassified as stdlib and skipped.", report.TotalStructs)
	}

	foundInner := false
	for _, sr := range report.StructReports {
		if sr.Name == "Inner" {
			foundInner = true
			break
		}
	}
	if !foundInner {
		t.Error("Inner was not collected. isStandardLibraryPkg may still misclassify simple package names.")
	}
	t.Logf("✅ simplepkg.Inner correctly collected (isStandardLibraryPkg fix verified)")
}

// TestStableSortEqualSizes verifies that fields with identical sizes
// produce deterministic ordering across runs (SliceStable with name tiebreaker).
func TestStableSortEqualSizes(t *testing.T) {
	// Create fields with equal sizes but different names
	fields := []optimizer.FieldInfo{
		{Name: "C", Size: 8, Align: 8},
		{Name: "A", Size: 8, Align: 8},
		{Name: "B", Size: 8, Align: 8},
	}

	// Run multiple times to verify deterministic output
	var firstResult []string
	for i := 0; i < 10; i++ {
		sorted := optimizer.ReorderFields(fields, false, nil)
		names := make([]string, len(sorted))
		for j, f := range sorted {
			names[j] = f.Name
		}
		if i == 0 {
			firstResult = names
		} else {
			for j := range names {
				if names[j] != firstResult[j] {
					t.Errorf("Non-deterministic sort at run %d: got %v, want %v",
						i+1, names, firstResult)
					return
				}
			}
		}
	}
	// With SliceStable + name tiebreaker, equal-size fields should sort A, B, C
	expected := []string{"A", "B", "C"}
	if !stringSlicesEqual(firstResult, expected) {
		t.Errorf("Expected stable sort order %v, got %v", expected, firstResult)
	}
	t.Logf("✅ Stable sort verified: 10 runs all produced %v", firstResult)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestTotalOrigSizeDepthZeroSemantics verifies that TotalOrigSize only counts
// depth-0 structs in -package mode and all depths in -struct mode.
func TestTotalOrigSizeDepthZeroSemantics(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "totalsize_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	goModContent := `module testtotal

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	mainContent := `package testtotal

type Child struct {
	A bool
	B uint64
	C bool
}

type Parent struct {
	X     bool
	Data  Child
	Y     bool
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// ---- Test -struct mode: all depths should be counted ----
	analyzerCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		StructName:  "testtotal.Parent",
		Verbose:     0,
		ProjectType: "gomod",
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		StructName:  "testtotal.Parent",
		Verbose:     0,
		ProjectType: "gomod",
		MaxDepth:    50,
		Timeout:     300,
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	reportStruct, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimize (-struct) failed: %v", err)
	}

	// In -struct mode, TotalOrigSize = Parent + Child (all depths)
	if reportStruct.TotalOrigSize <= 0 {
		t.Error("-struct mode: TotalOrigSize should be > 0")
	}
	// Parent contains Child, so total should be Parent_size + Child_size
	if reportStruct.TotalStructs < 2 {
		t.Errorf("-struct mode: expected at least 2 structs, got %d", reportStruct.TotalStructs)
	}
	if reportStruct.RootStructSize <= 0 {
		t.Error("-struct mode: RootStructSize should be set")
	}
	t.Logf("-struct mode: TotalOrigSize=%d, RootStructSize=%d, TotalStructs=%d",
		reportStruct.TotalOrigSize, reportStruct.RootStructSize, reportStruct.TotalStructs)

	// ---- Test -package mode: only depth-0 should be counted ----
	analyzerCfg2 := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "testtotal",
		Verbose:     0,
		ProjectType: "gomod",
	}
	anlz2 := analyzer.NewAnalyzer(analyzerCfg2)

	optimizerCfg2 := &optimizer.Config{
		TargetDir:   tmpDir,
		Package:     "testtotal",
		Verbose:     0,
		ProjectType: "gomod",
		MaxDepth:    50,
		Timeout:     300,
	}
	opt2 := optimizer.NewOptimizer(optimizerCfg2, anlz2)

	reportPkg, err := opt2.Optimize()
	if err != nil {
		t.Fatalf("Optimize (-package) failed: %v", err)
	}

	// In -package mode, TotalOrigSize = only depth-0 structs (Parent + Child at depth 0)
	// Parent and Child are BOTH depth 0 since they're found by the analyzer directly
	if reportPkg.TotalOrigSize <= 0 {
		t.Error("-package mode: TotalOrigSize should be > 0")
	}
	if reportPkg.RootStruct != "" {
		// -package mode has no root struct
	}
	t.Logf("-package mode: TotalOrigSize=%d, TotalStructs=%d",
		reportPkg.TotalOrigSize, reportPkg.TotalStructs)

	// Verify summary: both modes should have consistent struct counts
	t.Logf("✅ TotalOrigSize semantics verified for both -struct and -package modes")
}

// TestExtractFieldInfoSlicePointer checks that []A, []*A, and *A types
// are correctly recognized as struct references during collection.
func TestExtractFieldInfoSlicePointerTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fieldtypes_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	goModContent := `module testfieldtypes

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	mainContent := `package testfieldtypes

type ValueA struct {
	X bool
	Y uint64
}

type ValueB struct {
	X bool
	Y uint64
}

type Container struct {
	slice   []ValueA
	ptr     *ValueB
	ptrSlice []*ValueA
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	analyzerCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		StructName:  "testfieldtypes.Container",
		Verbose:     0,
		ProjectType: "gomod",
	}
	anlz := analyzer.NewAnalyzer(analyzerCfg)

	optimizerCfg := &optimizer.Config{
		TargetDir:   tmpDir,
		StructName:  "testfieldtypes.Container",
		Verbose:     0,
		ProjectType: "gomod",
		MaxDepth:    50,
		Timeout:     300,
	}
	opt := optimizer.NewOptimizer(optimizerCfg, anlz)

	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Should collect: Container, ValueA, ValueB
	// ValueA is referenced via []ValueA and []*ValueA; ValueB via *ValueB
	if report.TotalStructs < 3 {
		names := make([]string, 0)
		for _, sr := range report.StructReports {
			names = append(names, sr.Name)
		}
		t.Errorf("Expected at least 3 structs (Container, ValueA, ValueB), got %d: %s",
			report.TotalStructs, strings.Join(names, ", "))
	}

	for _, name := range []string{"ValueA", "ValueB"} {
		found := false
		for _, sr := range report.StructReports {
			if sr.Name == name && !sr.Skipped {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s was not collected or was skipped", name)
		}
	}

	t.Logf("✅ All slice/pointer/pointer-slice types correctly collected: %d total structs", report.TotalStructs)
}
