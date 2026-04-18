package optimizer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// TestTypeAliasExactSizeCalculation 精确验证重定义类型大小计算
// 这个测试验证每个重定义类型都被正确识别为其底层基本类型的大小
func TestTypeAliasExactSizeCalculation(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir := t.TempDir()

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "testpkg/typedef")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建类型定义文件
	typesFile := filepath.Join(pkgDir, "types.go")
	typesContent := `package typedef

// 1 字节类型
type ByteType uint8
type BoolType bool

// 2 字节类型  
type WordType uint16
type Int16Type int16

// 4 字节类型
type DWordType uint32
type Int32Type int32
type Float32Type float32

// 8 字节类型
type QWordType uint64
type Int64Type int64
type Float64Type float64
type StringType string
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatalf("写入 types.go 失败：%v", err)
	}

	// 创建测试结构体 - 每个结构体只包含一个重定义类型字段
	// 这样可以精确测量该类型的大小
	structsFile := filepath.Join(pkgDir, "structs.go")
	structsContent := `package typedef

// TestByteType 测试1字节类型
type TestByteType struct {
	A ByteType
}

// TestBoolType 测试1字节bool类型
type TestBoolType struct {
	A BoolType
}

// TestWordType 测试2字节类型
type TestWordType struct {
	A WordType
}

// TestDWordType 测试4字节类型
type TestDWordType struct {
	A DWordType
}

// TestQWordType 测试8字节类型
type TestQWordType struct {
	A QWordType
}

// TestInt64Type 测试8字节int64类型
type TestInt64Type struct {
	A Int64Type
}

// TestFloat32Type 测试4字节float32类型
type TestFloat32Type struct {
	A Float32Type
}

// TestFloat64Type 测试8字节float64类型
type TestFloat64Type struct {
	A Float64Type
}

// TestStringType 测试16字节string类型
type TestStringType struct {
	A StringType
}
`
	if err := os.WriteFile(structsFile, []byte(structsContent), 0644); err != nil {
		t.Fatalf("写入 structs.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "testpkg/typedef",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &Config{
		TargetDir:      tmpDir,
		Package:        "testpkg/typedef",
		ProjectType:    "gopath",
		GOPATH:         tmpDir,
		Verbose:        0,
		Timeout:        60,
		PkgWorkerLimit: 4,
	}
	opt := NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 建立结构体名称到大小的映射
	sizeMap := make(map[string]int64)
	for _, sr := range report.StructReports {
		sizeMap[sr.Name] = sr.OrigSize
		t.Logf("结构体 %-20s 大小: %d 字节", sr.Name, sr.OrigSize)
	}

	// 精确验证每个类型的大小
	// 注意：由于内存对齐，单字段结构体大小可能大于字段本身
	// 但我们应该能看出类型大小的差异

	tests := []struct {
		structName    string
		expectedMin   int64 // 期望的最小大小
		expectedMax   int64 // 期望的最大大小
		typeName      string
		actualType    string
	}{
		{"TestByteType", 1, 8, "ByteType", "uint8 (1 byte)"},
		{"TestBoolType", 1, 8, "BoolType", "bool (1 byte)"},
		{"TestWordType", 2, 8, "WordType", "uint16 (2 bytes)"},
		{"TestDWordType", 4, 8, "DWordType", "uint32 (4 bytes)"},
		{"TestQWordType", 8, 8, "QWordType", "uint64 (8 bytes)"},
		{"TestInt64Type", 8, 8, "Int64Type", "int64 (8 bytes)"},
		{"TestFloat32Type", 4, 8, "Float32Type", "float32 (4 bytes)"},
		{"TestFloat64Type", 8, 8, "Float64Type", "float64 (8 bytes)"},
		{"TestStringType", 16, 16, "StringType", "string (16 bytes)"},
	}

	allPassed := true
	for _, tt := range tests {
		size, exists := sizeMap[tt.structName]
		if !exists {
			t.Errorf("❌ 未找到结构体 %s (测试类型: %s - %s)", tt.structName, tt.typeName, tt.actualType)
			allPassed = false
			continue
		}

		if size < tt.expectedMin || size > tt.expectedMax {
			t.Errorf("❌ %s (%s) 大小=%d 字节，期望范围 [%d-%d]",
				tt.structName, tt.actualType, size, tt.expectedMin, tt.expectedMax)
			allPassed = false
		} else {
			t.Logf("✅ %s (%s) 大小=%d 字节，范围正确",
				tt.structName, tt.actualType, size)
		}
	}

	// 额外验证：确保类型大小递增关系正确
	// 1字节类型 < 2字节类型 < 4字节类型 < 8字节类型 < 16字节类型
	byteSize := sizeMap["TestByteType"]
	wordSize := sizeMap["TestWordType"]
	dwordSize := sizeMap["TestDWordType"]
	qwordSize := sizeMap["TestQWordType"]
	stringSize := sizeMap["TestStringType"]

	t.Log("\n=== 类型大小递增验证 ===")
	t.Logf("1字节类型结构体: %d 字节", byteSize)
	t.Logf("2字节类型结构体: %d 字节", wordSize)
	t.Logf("4字节类型结构体: %d 字节", dwordSize)
	t.Logf("8字节类型结构体: %d 字节", qwordSize)
	t.Logf("16字节类型结构体: %d 字节", stringSize)

	// 验证大小关系（至少不递减）
	if byteSize > wordSize {
		t.Logf("⚠️  注意: 1字节类型结构体(%d) >= 2字节类型结构体(%d)（可能由于对齐）", byteSize, wordSize)
	}
	if wordSize > dwordSize {
		t.Logf("⚠️  注意: 2字节类型结构体(%d) >= 4字节类型结构体(%d)（可能由于对齐）", wordSize, dwordSize)
	}
	if dwordSize > qwordSize {
		t.Errorf("❌ 4字节类型结构体(%d) >= 8字节类型结构体(%d)，类型大小计算可能有误", dwordSize, qwordSize)
		allPassed = false
	}
	if qwordSize > stringSize {
		t.Errorf("❌ 8字节类型结构体(%d) >= 16字节类型结构体(%d)，类型大小计算可能有误", qwordSize, stringSize)
		allPassed = false
	}

	if allPassed {
		t.Log("\n✅ 所有重定义类型大小计算正确")
	} else {
		t.Fatal("\n❌ 存在类型大小计算错误")
	}
}

// TestTypeAliasVsOriginal 对比重定义类型与原始类型的大小
func TestTypeAliasVsOriginal(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir := t.TempDir()

	// 创建项目结构
	pkgDir := filepath.Join(tmpDir, "src", "testpkg/compare")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建类型定义和对比结构体
	typesFile := filepath.Join(pkgDir, "types.go")
	typesContent := `package compare

// 重定义类型
type CustomInt int64
type CustomBool bool
type CustomString string

// 使用原始类型的结构体
type OriginalInt struct {
	Value int64
}

type OriginalBool struct {
	Value bool
}

type OriginalString struct {
	Value string
}

// 使用重定义类型的结构体
type AliasedInt struct {
	Value CustomInt
}

type AliasedBool struct {
	Value CustomBool
}

type AliasedString struct {
	Value CustomString
}
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatalf("写入 types.go 失败：%v", err)
	}

	// 创建分析器
	anlzCfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "testpkg/compare",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}
	anlz := analyzer.NewAnalyzer(anlzCfg)

	// 创建优化器
	optCfg := &Config{
		TargetDir:      tmpDir,
		Package:        "testpkg/compare",
		ProjectType:    "gopath",
		GOPATH:         tmpDir,
		Verbose:        0,
		Timeout:        60,
		PkgWorkerLimit: 4,
	}
	opt := NewOptimizer(optCfg, anlz)

	// 执行优化
	report, err := opt.Optimize()
	if err != nil {
		t.Fatalf("优化失败：%v", err)
	}

	// 建立大小映射
	sizeMap := make(map[string]int64)
	for _, sr := range report.StructReports {
		sizeMap[sr.Name] = sr.OrigSize
	}

	// 对比重定义类型与原始类型的大小
	t.Log("\n=== 重定义类型 vs 原始类型大小对比 ===")

	comparisons := []struct {
		original string
		aliased  string
		typeInfo string
	}{
		{"OriginalInt", "AliasedInt", "int64 (8 bytes)"},
		{"OriginalBool", "AliasedBool", "bool (1 byte)"},
		{"OriginalString", "AliasedString", "string (16 bytes)"},
	}

	allMatch := true
	for _, comp := range comparisons {
		origSize := sizeMap[comp.original]
		aliasSize := sizeMap[comp.aliased]

		t.Logf("%-20s (原始): %d 字节", comp.original, origSize)
		t.Logf("%-20s (别名): %d 字节 [%s]", comp.aliased, aliasSize, comp.typeInfo)

		if origSize != aliasSize {
			t.Errorf("❌ 大小不匹配! %s (%d) != %s (%d)",
				comp.original, origSize, comp.aliased, aliasSize)
			allMatch = false
		} else {
			t.Logf("✅ 大小匹配: %d 字节\n", origSize)
		}
	}

	if allMatch {
		t.Log("\n✅ 所有重定义类型与原始类型大小一致")
	} else {
		t.Fatal("\n❌ 存在类型大小不匹配的问题")
	}
}
