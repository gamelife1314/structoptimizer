package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamelife1314/structoptimizer/analyzer"
)

// TestGopathVendorSupport 测试 GOPATH 模式下 vendor 目录支持
func TestGopathVendorSupport(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_vendor_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 GOPATH 项目结构
	// tmpDir/
	//   src/
	//     mycompany/myproject/
	//       vendor/
	//         github.com/somelib/
	//           utils/
	//             helper.go
	//       app/
	//         main.go  (引用 vendor 中的包)

	// 创建 vendor 目录中的依赖包
	vendorPkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject", "vendor", "github.com", "somelib", "utils")
	if err := os.MkdirAll(vendorPkgDir, 0755); err != nil {
		t.Fatalf("创建 vendor 包目录失败：%v", err)
	}

	// 创建 vendor 中的包文件
	helperFile := filepath.Join(vendorPkgDir, "helper.go")
	helperContent := `package utils

// Helper vendor 中的辅助函数
type Helper struct {
	Name    string
	Version int
}

// NewHelper 创建辅助对象
func NewHelper(name string) *Helper {
	return &Helper{
		Name:    name,
		Version: 1,
	}
}

// GetVersion 获取版本
func (h *Helper) GetVersion() int {
	return h.Version
}
`
	if err := os.WriteFile(helperFile, []byte(helperContent), 0644); err != nil {
		t.Fatalf("写入 helper.go 失败：%v", err)
	}

	// 创建主项目目录
	mainPkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject", "app")
	if err := os.MkdirAll(mainPkgDir, 0755); err != nil {
		t.Fatalf("创建主包目录失败：%v", err)
	}

	// 创建主项目文件（引用 vendor 中的包）
	mainFile := filepath.Join(mainPkgDir, "main.go")
	mainContent := `package app

import (
	"github.com/somelib/utils"
)

// AppConfig 应用配置
type AppConfig struct {
	Name   string
	Port   int
	Helper *utils.Helper
}

// NewAppConfig 创建配置
func NewAppConfig(name string) *AppConfig {
	return &AppConfig{
		Name:   name,
		Port:   8080,
		Helper: utils.NewHelper("default"),
	}
}

// GetVersion 获取版本
func (c *AppConfig) GetVersion() int {
	return c.Helper.GetVersion()
}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("写入 main.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/app",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试加载主包（应该能从 vendor 中找到依赖）
	t.Run("LoadPackageWithVendor", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/app")
		if err != nil {
			t.Fatalf("加载主包失败：%v", err)
		}

		if pkg == nil {
			t.Fatal("包为空")
		}

		if pkg.Name != "app" {
			t.Errorf("包名称错误：期望 app，实际 %s", pkg.Name)
		}

		// 验证 AppConfig 结构体存在
		obj := pkg.Types.Scope().Lookup("AppConfig")
		if obj == nil {
			t.Fatal("未找到 AppConfig 类型")
		}

		t.Logf("成功加载主包，找到 AppConfig 类型")
	})

	// 测试直接加载 vendor 中的包
	t.Run("LoadVendorPackage", func(t *testing.T) {
		vendorPkgPath := "github.com/somelib/utils"
		pkg, err := anlz.LoadPackage(vendorPkgPath)
		if err != nil {
			t.Fatalf("加载 vendor 包失败：%v", err)
		}

		if pkg == nil {
			t.Fatal("vendor 包为空")
		}

		if pkg.Name != "utils" {
			t.Errorf("vendor 包名称错误：期望 utils，实际 %s", pkg.Name)
		}

		// 验证 Helper 类型存在
		obj := pkg.Types.Scope().Lookup("Helper")
		if obj == nil {
			t.Fatal("未找到 Helper 类型")
		}

		t.Logf("成功加载 vendor 包，找到 Helper 类型")
	})
}

// TestGopathVendorMultiple 测试多个 vendor 依赖包
func TestGopathVendorMultiple(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_multi_vendor_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建多个 vendor 依赖包
	vendorPackages := map[string]string{
		"github.com/lib1/models": `package models

// User 用户模型
type User struct {
	ID    int64
	Name  string
	Email string
}

// Product 产品模型
type Product struct {
	ID    int64
	Name  string
	Price float64
}
`,
		"github.com/lib2/services": `package services

// UserService 用户服务
type UserService struct {
	db string
}

// NewUserService 创建用户服务
func NewUserService(db string) *UserService {
	return &UserService{db: db}
}

// ProductService 产品服务
type ProductService struct {
	cache string
}

// NewProductService 创建产品服务
func NewProductService(cache string) *ProductService {
	return &ProductService{cache: cache}
}
`,
	}

	// 创建 vendor 目录
	for pkgPath, content := range vendorPackages {
		pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject", "vendor", pkgPath)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatalf("创建 vendor 包目录失败 %s：%v", pkgPath, err)
		}

		filePath := filepath.Join(pkgDir, "pkg.go")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("写入文件失败 %s：%v", pkgPath, err)
		}
	}

	// 创建主项目
	mainPkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject", "api")
	if err := os.MkdirAll(mainPkgDir, 0755); err != nil {
		t.Fatalf("创建主包目录失败：%v", err)
	}

	// 创建主项目文件（引用多个 vendor 包）
	mainFile := filepath.Join(mainPkgDir, "api.go")
	mainContent := `package api

import (
	"github.com/lib1/models"
	"github.com/lib2/services"
)

// APIHandler API 处理器
type APIHandler struct {
	userSvc    *services.UserService
	productSvc *services.ProductService
}

// NewAPIHandler 创建处理器
func NewAPIHandler(db, cache string) *APIHandler {
	return &APIHandler{
		userSvc:    services.NewUserService(db),
		productSvc: services.NewProductService(cache),
	}
}

// GetUser 获取用户
func (h *APIHandler) GetUser(id int64) *models.User {
	return &models.User{
		ID:   id,
		Name: "test",
	}
}

// GetProduct 获取产品
func (h *APIHandler) GetProduct(id int64) *models.Product {
	return &models.Product{
		ID:    id,
		Name:  "test product",
		Price: 99.99,
	}
}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("写入 api.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/api",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试加载主包
	t.Run("LoadMainPackage", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/api")
		if err != nil {
			t.Fatalf("加载主包失败：%v", err)
		}

		if pkg.Name != "api" {
			t.Errorf("包名称错误：期望 api，实际 %s", pkg.Name)
		}

		// 验证 APIHandler 存在
		obj := pkg.Types.Scope().Lookup("APIHandler")
		if obj == nil {
			t.Fatal("未找到 APIHandler 类型")
		}

		t.Logf("成功加载主包")
	})

	// 测试加载 vendor 包 1
	t.Run("LoadVendorPackage1", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("github.com/lib1/models")
		if err != nil {
			t.Fatalf("加载 models 包失败：%v", err)
		}

		if pkg.Name != "models" {
			t.Errorf("包名称错误：期望 models，实际 %s", pkg.Name)
		}

		// 验证 User 和 Product 存在
		if obj := pkg.Types.Scope().Lookup("User"); obj == nil {
			t.Error("未找到 User 类型")
		}

		if obj := pkg.Types.Scope().Lookup("Product"); obj == nil {
			t.Error("未找到 Product 类型")
		}

		t.Logf("成功加载 models 包")
	})

	// 测试加载 vendor 包 2
	t.Run("LoadVendorPackage2", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("github.com/lib2/services")
		if err != nil {
			t.Fatalf("加载 services 包失败：%v", err)
		}

		if pkg.Name != "services" {
			t.Errorf("包名称错误：期望 services，实际 %s", pkg.Name)
		}

		// 验证 UserService 和 ProductService 存在
		if obj := pkg.Types.Scope().Lookup("UserService"); obj == nil {
			t.Error("未找到 UserService 类型")
		}

		if obj := pkg.Types.Scope().Lookup("ProductService"); obj == nil {
			t.Error("未找到 ProductService 类型")
		}

		t.Logf("成功加载 services 包")
	})
}

// TestGopathVendorNotFound 测试 vendor 目录中没有包的情况
func TestGopathVendorNotFound(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_no_vendor_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目（没有 vendor 目录）
	pkgDir := filepath.Join(tmpDir, "src", "mycompany/myproject/simple")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("创建包目录失败：%v", err)
	}

	// 创建简单文件（只使用标准库）
	simpleFile := filepath.Join(pkgDir, "simple.go")
	simpleContent := `package simple

import "time"

// Task 任务
type Task struct {
	ID       int64
	Name     string
	Duration time.Duration
}

// NewTask 创建任务
func NewTask(id int64, name string) *Task {
	return &Task{
		ID:       id,
		Name:     name,
		Duration: time.Second,
	}
}
`
	if err := os.WriteFile(simpleFile, []byte(simpleContent), 0644); err != nil {
		t.Fatalf("写入 simple.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/simple",
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试加载（应该成功，只依赖标准库）
	t.Run("LoadWithoutVendor", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/simple")
		if err != nil {
			t.Fatalf("加载包失败：%v", err)
		}

		if pkg.Name != "simple" {
			t.Errorf("包名称错误：期望 simple，实际 %s", pkg.Name)
		}

		// 验证 Task 存在
		obj := pkg.Types.Scope().Lookup("Task")
		if obj == nil {
			t.Fatal("未找到 Task 类型")
		}

		t.Logf("成功加载无 vendor 项目的包")
	})
}

// TestGopathSamePackageSubdir 测试同项目子包不从 vendor 查找
func TestGopathSamePackageSubdir(t *testing.T) {
	// 创建临时 GOPATH 目录
	tmpDir, err := os.MkdirTemp("", "gopath_same_pkg_*")
	if err != nil {
		t.Fatalf("创建临时目录失败：%v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建项目结构（带 vendor 目录）
	// tmpDir/
	//   src/
	//     mycompany/myproject/
	//       vendor/                    # vendor 目录存在
	//         github.com/external/lib/ # 但不会用于同项目的包
	//       api/                       # 主包
	//         main.go
	//       api/handlers/              # 子包（同项目）
	//         handlers.go

	// 创建 vendor 目录（但不应该用于同项目的包）
	vendorDir := filepath.Join(tmpDir, "src", "mycompany/myproject", "vendor", "github.com", "external", "lib")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("创建 vendor 目录失败：%v", err)
	}

	// 创建 vendor 中的包
	vendorFile := filepath.Join(vendorDir, "lib.go")
	vendorContent := `package lib

// ExternalLib vendor 中的外部库
type ExternalLib struct {
	Name string
}
`
	if err := os.WriteFile(vendorFile, []byte(vendorContent), 0644); err != nil {
		t.Fatalf("写入 vendor lib.go 失败：%v", err)
	}

	// 创建主包
	apiDir := filepath.Join(tmpDir, "src", "mycompany/myproject", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("创建 api 目录失败：%v", err)
	}

	apiFile := filepath.Join(apiDir, "main.go")
	apiContent := `package api

// APIHandler API 处理器
type APIHandler struct {
	Name string
	Port int
}

// NewAPIHandler 创建处理器
func NewAPIHandler(name string, port int) *APIHandler {
	return &APIHandler{
		Name: name,
		Port: port,
	}
}
`
	if err := os.WriteFile(apiFile, []byte(apiContent), 0644); err != nil {
		t.Fatalf("写入 api/main.go 失败：%v", err)
	}

	// 创建子包（handlers）
	handlersDir := filepath.Join(apiDir, "handlers")
	if err := os.MkdirAll(handlersDir, 0755); err != nil {
		t.Fatalf("创建 handlers 目录失败：%v", err)
	}

	handlersFile := filepath.Join(handlersDir, "handlers.go")
	handlersContent := `package handlers

import "mycompany/myproject/api"

// UserHandler 用户处理器
type UserHandler struct {
	API *api.APIHandler
}

// NewUserHandler 创建用户处理器
func NewUserHandler(api *api.APIHandler) *UserHandler {
	return &UserHandler{API: api}
}
`
	if err := os.WriteFile(handlersFile, []byte(handlersContent), 0644); err != nil {
		t.Fatalf("写入 handlers/handlers.go 失败：%v", err)
	}

	// 创建分析器（GOPATH 模式，指定 -pkg-scope）
	cfg := &analyzer.Config{
		TargetDir:   tmpDir,
		Package:     "mycompany/myproject/api",  // 主包
		ProjectType: "gopath",
		GOPATH:      tmpDir,
		Verbose:     0,
	}

	anlz := analyzer.NewAnalyzer(cfg)

	// 测试加载主包
	t.Run("LoadMainPackage", func(t *testing.T) {
		pkg, err := anlz.LoadPackage("mycompany/myproject/api")
		if err != nil {
			t.Fatalf("加载主包失败：%v", err)
		}

		if pkg.Name != "api" {
			t.Errorf("包名称错误：期望 api，实际 %s", pkg.Name)
		}

		// 验证 APIHandler 存在
		obj := pkg.Types.Scope().Lookup("APIHandler")
		if obj == nil {
			t.Fatal("未找到 APIHandler 类型")
		}

		t.Logf("成功加载主包 mycompany/myproject/api")
	})

	// 测试加载子包（关键测试：不应该从 vendor 查找）
	t.Run("LoadSubPackage", func(t *testing.T) {
		// 这是同项目下的子包，应该从 $GOPATH/src 中查找
		// 而不应该去 vendor 目录查找
		pkg, err := anlz.LoadPackage("mycompany/myproject/api/handlers")
		if err != nil {
			t.Fatalf("加载子包失败：%v", err)
		}

		if pkg.Name != "handlers" {
			t.Errorf("包名称错误：期望 handlers，实际 %s", pkg.Name)
		}

		// 验证 UserHandler 存在
		obj := pkg.Types.Scope().Lookup("UserHandler")
		if obj == nil {
			t.Fatal("未找到 UserHandler 类型")
		}

		t.Logf("成功加载子包 mycompany/myproject/api/handlers（从 GOPATH/src 加载，非 vendor）")
	})
}
