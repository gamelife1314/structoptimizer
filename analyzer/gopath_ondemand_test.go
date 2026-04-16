package analyzer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// TestGopathOnDemandLoad 测试 GOPATH 模式下按需加载包功能
func TestGopathOnDemandLoad(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_ondemand_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 GOPATH 项目结构
	// tmpDir/
	//   src/
	//     mycompany/myproject/mypkg/
	//       types.go      - 定义类型
	//       service.go    - 使用类型的服务

	pkgDir := filepath.Join(tmpDir, "src", "mycompany", "myproject", "mypkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建类型定义文件
	typesFile := filepath.Join(pkgDir, "types.go")
	typesContent := `package mypkg

// Config 配置结构体
type Config struct {
	Name    string
	Timeout int
	Retry   uint8
}

// Status 状态枚举类型
type Status int

const (
	StatusUnknown Status = 0
	StatusActive  Status = 1
	StatusInactive Status = 2
)

// Response 响应结构体
type Response struct {
	Code    int
	Message string
	Data    interface{}
}
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatalf("写入 types.go 失败：%v", err)
	}

	// 创建服务文件
	serviceFile := filepath.Join(pkgDir, "service.go")
	serviceContent := `package mypkg

import (
	"fmt"
	"time"
)

// Service 服务结构体
type Service struct {
	config *Config
	status Status
}

// NewService 创建新服务
func NewService(cfg *Config) *Service {
	return &Service{
		config: cfg,
		status: StatusActive,
	}
}

// GetName 获取名称
func (s *Service) GetName() string {
	return fmt.Sprintf("Service-%s", s.config.Name)
}

// GetTimeout 获取超时
func (s *Service) GetTimeout() time.Duration {
	return time.Duration(s.config.Timeout) * time.Second
}

// Process 处理请求
func (s *Service) Process() *Response {
	return &Response{
		Code:    200,
		Message: "OK",
		Data:    nil,
	}
}
`
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		t.Fatalf("写入 service.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/mypkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试 LoadPackage - 按需加载
	t.Run("LoadPackage", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/mypkg")
		if err != nil {
			t.Fatalf("加载包失败：%v", err)
		}

		if pkg == nil {
			t.Fatal("包为空")
		}

		// 验证包信息
		if pkg.Name != "mypkg" {
			t.Errorf("包名称错误：期望 mypkg，实际 %s", pkg.Name)
		}

		if pkg.PkgPath != "mycompany/myproject/mypkg" {
			t.Errorf("包路径错误：期望 mycompany/myproject/mypkg，实际 %s", pkg.PkgPath)
		}

		// 验证 Go 文件列表（应该有 2 个文件）
		if len(pkg.GoFiles) != 2 {
			t.Errorf("Go 文件数量错误：期望 2，实际 %d", len(pkg.GoFiles))
		}

		// 验证 TypesInfo 不为空
		if pkg.TypesInfo == nil {
			t.Error("TypesInfo 为空")
		}

		// 验证 Syntax 不为空
		if len(pkg.Syntax) != 2 {
			t.Errorf("Syntax 数量错误：期望 2，实际 %d", len(pkg.Syntax))
		}
	})

	// 测试 FindStructByName - 查找结构体
	t.Run("FindStructByName", func(t *testing.T) {
		// 先加载包
		_, err := anlz.LoadPackage("mycompany/myproject/mypkg")
		if err != nil {
			t.Fatalf("加载包失败：%v", err)
		}

		// 查找 Config 结构体
		st, filePath, err := anlz.FindStructByName("mycompany/myproject/mypkg", "Config")
		if err != nil {
			t.Fatalf("查找 Config 结构体失败：%v", err)
		}

		if st == nil {
			t.Fatal("Config 结构体为空")
		}

		// 验证字段数量
		if st.NumFields() != 3 {
			t.Errorf("Config 字段数量错误：期望 3，实际 %d", st.NumFields())
		}

		// 验证文件路径
		if filePath == "" {
			t.Error("文件路径为空")
		}

		if !strings.Contains(filePath, "types.go") {
			t.Errorf("文件路径错误：期望包含 types.go，实际 %s", filePath)
		}

		// 查找 Service 结构体
		st2, filePath2, err := anlz.FindStructByName("mycompany/myproject/mypkg", "Service")
		if err != nil {
			t.Fatalf("查找 Service 结构体失败：%v", err)
		}

		if st2 == nil {
			t.Fatal("Service 结构体为空")
		}

		// 验证字段数量
		if st2.NumFields() != 2 {
			t.Errorf("Service 字段数量错误：期望 2，实际 %d", st2.NumFields())
		}

		// 验证文件路径
		if !strings.Contains(filePath2, "service.go") {
			t.Errorf("文件路径错误：期望包含 service.go，实际 %s", filePath2)
		}
	})

	// 测试跨文件类型引用
	t.Run("CrossFileReference", func(t *testing.T) {
		// 加载包
		pkg, err := anlz.LoadPackage("mycompany/myproject/mypkg")
		if err != nil {
			t.Fatalf("加载包失败：%v", err)
		}

		// 验证 service.go 中引用了 types.go 中的类型
		// Service.config 字段类型是 *Config
		if pkg.TypesInfo == nil {
			t.Fatal("TypesInfo 为空")
		}

		// 查找 Service 结构体
		obj := pkg.Types.Scope().Lookup("Service")
		if obj == nil {
			t.Fatal("未找到 Service 类型")
		}

		// 验证类型信息
		if obj.Type() == nil {
			t.Fatal("Service 类型为空")
		}
	})
}

// TestGopathOnDemandLoadMultiplePackages 测试 GOPATH 模式下加载多个包
func TestGopathOnDemandLoadMultiplePackages(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_multi_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建多个包
	packages := map[string]string{
		"mycompany/myproject/model": `package model

// User 用户模型
type User struct {
	ID       int64
	Name     string
	Email    string
	Age      uint8
	IsActive bool
}

// Product 产品模型
type Product struct {
	ID    int64
	Name  string
	Price float64
}
`,
		"mycompany/myproject/service": `package service

import (
	"mycompany/myproject/model"
)

// UserService 用户服务
type UserService struct {
	cache map[int64]*model.User
}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{
		cache: make(map[int64]*model.User),
	}
}

// ProductService 产品服务
type ProductService struct {
	items []*model.Product
}

// NewProductService 创建产品服务
func NewProductService() *ProductService {
	return &ProductService{
		items: make([]*model.Product, 0),
	}
}
`,
	}

	// 创建包文件
	for pkgPath, content := range packages {
		pkgDir := filepath.Join(tmpDir, "src", pkgPath)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatalf("创建包目录失败 %s：%v", pkgPath, err)
		}

		filePath := filepath.Join(pkgDir, "models.go")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("写入文件失败 %s：%v", pkgPath, err)
		}
	}

	// 创建分析器（GOPATH 模式）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/model",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试加载 model 包
	t.Run("LoadModelPackage", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/model")
		if err != nil {
			t.Fatalf("加载 model 包失败：%v", err)
		}

		if pkg.Name != "model" {
			t.Errorf("包名称错误：期望 model，实际 %s", pkg.Name)
		}

		// 验证 User 结构体
		obj := pkg.Types.Scope().Lookup("User")
		if obj == nil {
			t.Fatal("未找到 User 类型")
		}

		// 验证 Product 结构体
		obj2 := pkg.Types.Scope().Lookup("Product")
		if obj2 == nil {
			t.Fatal("未找到 Product 类型")
		}
	})

	// 测试加载 service 包（引用 model 包）
	t.Run("LoadServicePackage", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/service")
		if err != nil {
			t.Fatalf("加载 service 包失败：%v", err)
		}

		if pkg.Name != "service" {
			t.Errorf("包名称错误：期望 service，实际 %s", pkg.Name)
		}

		// 验证 UserService 结构体
		obj := pkg.Types.Scope().Lookup("UserService")
		if obj == nil {
			t.Fatal("未找到 UserService 类型")
		}

		// 验证 ProductService 结构体
		obj2 := pkg.Types.Scope().Lookup("ProductService")
		if obj2 == nil {
			t.Fatal("未找到 ProductService 类型")
		}
	})
}

// TestGopathOnDemandLoadWithErrors 测试 GOPATH 模式下包加载有错误时的处理
func TestGopathOnDemandLoadWithErrors(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_errors_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建有语法错误的包
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/badpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建有语法错误的文件
	badFile := filepath.Join(pkgDir, "bad.go")
	badContent := `package badpkg

// BadStruct 有语法错误的结构体
type BadStruct struct {
	Name string
	// 缺少闭合大括号
`
	if err := os.WriteFile(badFile, []byte(badContent), 0644); err != nil {
		t.Fatalf("写入 bad.go 失败：%v", err)
	}

	// 创建正常的文件
	goodFile := filepath.Join(pkgDir, "good.go")
	goodContent := `package badpkg

// GoodStruct 正常的结构体
type GoodStruct struct {
	ID   int
	Name string
}
`
	if err := os.WriteFile(goodFile, []byte(goodContent), 0644); err != nil {
		t.Fatalf("写入 good.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/badpkg",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试加载有错误文件的包 - 应该仍然可以加载
	t.Run("LoadPackageWithErrors", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/badpkg")
		
		// 即使有语法错误，也应该能加载（跳过错误文件）
		if err != nil {
			// 如果完全无法加载，应该返回错误
			t.Logf("加载包返回错误（预期）：%v", err)
			return
		}

		// 如果加载成功，验证包信息
		if pkg == nil {
			t.Fatal("包为空")
		}

		// 应该至少有 1 个文件（good.go）被成功解析
		if len(pkg.GoFiles) < 1 {
			t.Errorf("Go 文件数量错误：期望至少 1，实际 %d", len(pkg.GoFiles))
		}

		// 验证 GoodStruct 存在
		obj := pkg.Types.Scope().Lookup("GoodStruct")
		if obj == nil {
			t.Fatal("未找到 GoodStruct 类型")
		}
	})
}
