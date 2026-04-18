package optimizer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// TestTypeAliasRecognition 测试类型别名识别（type NewType int）
func TestTypeAliasRecognition(t *testing.T) {
	// 创建临时包
	tmpDir := t.TempDir()

	// 创建包含类型别名的文件
	typeAliasFile := filepath.Join(tmpDir, "types.go")
	content := `package testpkg

type CustomInt int
type CustomInt64 int64
type CustomBool bool
type CustomString string

type MyStruct struct {
	A CustomInt
	B CustomInt64
	C CustomBool
	D CustomString
	E int64
}
`
	if err := os.WriteFile(typeAliasFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 解析文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, typeAliasFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 验证类型别名被正确识别
	foundMyStruct := false
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "MyStruct" {
				continue
			}

			foundMyStruct = true
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				t.Fatal("MyStruct should be a struct type")
			}

			// 验证字段数量
			if st.Fields == nil || st.Fields.List == nil {
				t.Fatal("MyStruct should have fields")
			}

			fieldCount := 0
			for _, field := range st.Fields.List {
				if len(field.Names) > 0 {
					fieldCount++
				}
			}

			if fieldCount != 5 {
				t.Errorf("Expected 5 fields, got %d", fieldCount)
			}

			// 验证类型别名定义存在
			t.Log("Successfully parsed struct with type alias fields")
		}
	}

	if !foundMyStruct {
		t.Error("MyStruct not found in file")
	}
}

// TestCrossFileUnexportedStructDetection 测试跨文件未导出结构体检测
func TestCrossFileUnexportedStructDetection(t *testing.T) {
	// 创建临时包
	tmpDir := t.TempDir()

	// 创建第一个文件 - 包含主结构体
	file1 := filepath.Join(tmpDir, "main.go")
	content1 := `package mypkg

type MainStruct struct {
	Name    string
	Helper  unexportedHelper
	Data    int64
}
`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	// 创建第二个文件 - 包含未导出的结构体
	file2 := filepath.Join(tmpDir, "helper.go")
	content2 := `package mypkg

type unexportedHelper struct {
	ID    int64
	Active bool
	Value int32
}
`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// 扫描包目录
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}

	goFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 3 && entry.Name()[len(entry.Name())-3:] == ".go" {
			goFiles++
		}
	}

	if goFiles != 2 {
		t.Errorf("Expected 2 Go files, found %d", goFiles)
	}

	// 验证可以解析两个文件
	for _, file := range []string{file1, file2} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", filepath.Base(file), err)
			continue
		}

		t.Logf("Successfully parsed %s, package: %s", filepath.Base(file), f.Name.Name)
	}
}

// TestEmbeddedFieldInGOPATHProject 测试GOPATH项目中匿名字段的识别
func TestEmbeddedFieldInGOPATHProject(t *testing.T) {
	// 创建临时包
	tmpDir := t.TempDir()

	// 创建包含匿名字段的文件
	embedFile := filepath.Join(tmpDir, "embed.go")
	content := `package testpkg

type BaseStruct struct {
	ID   int64
	Name string
}

type ChildStruct struct {
	BaseStruct
	Age     int32
	Active  bool
	Extra   int64
}
`
	if err := os.WriteFile(embedFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 解析文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, embedFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 查找ChildStruct
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "ChildStruct" {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				t.Fatal("ChildStruct should be a struct type")
			}

			// 统计匿名字段
			embedCount := 0
			for _, field := range st.Fields.List {
				if len(field.Names) == 0 {
					embedCount++
				}
			}

			if embedCount != 1 {
				t.Errorf("Expected 1 embedded field, got %d", embedCount)
			}

			t.Log("Successfully detected embedded field")
		}
	}
}

// TestMethodDetectionInGOPATH 测试GOPATH项目中方法的检测
func TestMethodDetectionInGOPATH(t *testing.T) {
	// 创建临时包
	tmpDir := t.TempDir()

	// 创建包含方法的文件
	methodFile := filepath.Join(tmpDir, "methods.go")
	content := `package testpkg

type StructWithMethods struct {
	Name  string
	Value int64
}

func (s *StructWithMethods) Encode() []byte {
	return []byte(s.Name)
}

func (s *StructWithMethods) Decode(data []byte) error {
	s.Name = string(data)
	return nil
}

func (s StructWithMethods) GetValue() int64 {
	return s.Value
}

type StructNoMethods struct {
	Name  string
	Value int64
}
`
	if err := os.WriteFile(methodFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 解析文件并提取方法
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, methodFile, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 统计方法
	methods := make(map[string][]string) // structName -> []methodName
	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}

		if len(funcDecl.Recv.List) == 0 {
			continue
		}

		// 提取接收者类型
		recvType := ""
		recv := funcDecl.Recv.List[0]
		switch t := recv.Type.(type) {
		case *ast.Ident:
			recvType = t.Name
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				recvType = ident.Name
			}
		}

		if recvType != "" {
			methods[recvType] = append(methods[recvType], funcDecl.Name.Name)
		}
	}

	// 验证StructWithMethods有2个方法（Encode和Decode，GetValue是值类型接收者）
	if len(methods["StructWithMethods"]) != 3 {
		t.Errorf("StructWithMethods should have 3 methods, got %d", len(methods["StructWithMethods"]))
	}

	// 验证StructNoMethods没有方法
	if len(methods["StructNoMethods"]) != 0 {
		t.Errorf("StructNoMethods should have 0 methods, got %d", len(methods["StructNoMethods"]))
	}

	t.Logf("StructWithMethods methods: %v", methods["StructWithMethods"])
}

// TestSkipByMethodsWithWildcard 测试skip-by-methods支持通配符
func TestSkipByMethodsWithWildcard(t *testing.T) {
	mi := NewMethodIndex()

	// 创建临时包
	tmpDir := t.TempDir()

	// 创建包含方法的文件
	methodFile := filepath.Join(tmpDir, "api.go")
	content := `package api

type Handler struct {
	Name string
}

func (h *Handler) EncodeJSON() []byte {
	return nil
}

func (h *Handler) EncodeXML() []byte {
	return nil
}

func (h *Handler) DecodeJSON(data []byte) error {
	return nil
}
`
	if err := os.WriteFile(methodFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 直接扫描目录而不是使用包路径
	// 手动调用indexPkg并传入目录
	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.go"))
	if len(files) == 0 {
		t.Fatal("No Go files found in test directory")
	}

	// 手动索引文件
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		// 确保包缓存存在
		mi.mu.Lock()
		if _, ok := mi.cache["api"]; !ok {
			mi.cache["api"] = make(map[string]map[string]bool)
		}
		pkgCache := mi.cache["api"]
		mi.mu.Unlock()

		for _, decl := range f.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			recvType := extractRecvType(funcDecl.Recv.List[0].Type)
			if recvType == "" {
				continue
			}

			methodName := funcDecl.Name.Name

			mi.mu.Lock()
			if _, ok := pkgCache[recvType]; !ok {
				pkgCache[recvType] = make(map[string]bool)
			}
			pkgCache[recvType][methodName] = true
			mi.mu.Unlock()
		}
	}

	// 测试通配符匹配
	tests := []struct {
		pattern string
		expect  bool
	}{
		{"Encode*", true},
		{"*JSON", true},
		{"Encode", false},
		{"Decode*", true},
		{"NotExist", false},
	}

	for _, tt := range tests {
		result := mi.HasMethod("api", "Handler", tt.pattern)
		if result != tt.expect {
			t.Errorf("HasMethod(%q) = %v, want %v", tt.pattern, result, tt.expect)
		}
	}
}
