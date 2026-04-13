# StructOptimizer 执行计划

## 阶段概览

| 阶段 | 任务 | 预计工作量 |
|------|------|------------|
| 1 | 项目初始化 | 0.5 小时 |
| 2 | 核心模块实现 | 4 小时 |
| 3 | 测试用例编写 | 2 小时 |
| 4 | 验证与优化 | 1.5 小时 |

## 详细执行步骤

### 阶段 1: 项目初始化

#### 1.1 创建项目结构
```bash
mkdir -p cmd/structoptimizer
mkdir -p analyzer
mkdir -p optimizer
mkdir -p reporter
mkdir -p writer
mkdir -p internal/utils
mkdir -p testdata/basic
mkdir -p testdata/nested
mkdir -p testdata/crosspkg/subpkg1
mkdir -p testdata/crosspkg/subpkg2
```

#### 1.2 初始化 Go 模块
```bash
go mod init github.com/structoptimizer/structoptimizer
```

### 阶段 2: 核心模块实现

#### 2.1 工具函数模块 (`internal/utils/utils.go`)
- [ ] `MatchPattern()`: 通配符匹配
- [ ] `FormatSize()`: 格式化字节大小
- [ ] `GetGoModRoot()`: 获取 go.mod 根目录

#### 2.2 分析器模块 (`analyzer/analyzer.go`)
- [ ] `NewAnalyzer()`: 创建分析器
- [ ] `LoadPackage()`: 加载包信息
- [ ] `FindStructByName()`: 查找指定结构体
- [ ] `FindAllStructs()`: 查找包中所有结构体
- [ ] `GetStructFields()`: 获取结构体字段
- [ ] `HasMethod()`: 检查结构体是否有指定方法
- [ ] `IsExternalPackage()`: 判断是否为外部包

#### 2.3 优化器模块

##### 2.3.1 字段分析 (`optimizer/field.go`)
- [ ] `FieldInfo` 结构体定义
- [ ] `StructInfo` 结构体定义
- [ ] `AnalyzeFields()`: 分析字段类型和大小
- [ ] `CalculateFieldSize()`: 计算字段大小

##### 2.3.2 大小计算 (`optimizer/size.go`)
- [ ] `CalcStructSize()`: 计算结构体总大小（含填充）
- [ ] `CalcOptimizedSize()`: 计算优化后大小

##### 2.3.3 核心优化 (`optimizer/optimizer.go`)
- [ ] `NewOptimizer()`: 创建优化器
- [ ] `Optimize()`: 执行优化（入口函数）
- [ ] `OptimizeStruct()`: 优化单个结构体
- [ ] `ReorderFields()`: 重排字段
- [ ] `OptimizeNested()`: 递归优化嵌套结构体
- [ ] `ResolveCrossPackage()`: 解析跨包引用
- [ ] `isOptimized()`: 检查是否已优化

#### 2.4 报告模块 (`reporter/reporter.go`)
- [ ] `Report` 结构体定义
- [ ] `StructReport` 结构体定义
- [ ] `NewReporter()`: 创建报告生成器
- [ ] `Generate()`: 生成报告
- [ ] `GenerateTXT()`: TXT 格式
- [ ] `GenerateMD()`: Markdown 格式
- [ ] `GenerateHTML()`: HTML 格式（可选）

#### 2.5 写入器模块 (`writer/writer.go`)
- [ ] `NewSourceWriter()`: 创建写入器
- [ ] `BackupFile()`: 备份源文件
- [ ] `WriteStruct()`: 写入优化后的结构体
- [ ] `RewriteFile()`: 重写整个文件

#### 2.6 CLI 模块 (`cmd/structoptimizer/main.go`)
- [ ] `Config` 结构体定义
- [ ] 参数解析
- [ ] 主流程控制
- [ ] 日志输出（-v, -vv, -vvv）

### 阶段 3: 测试用例编写

#### 3.1 基础测试 (`optimizer/optimizer_test.go`)
- [ ] 简单结构体优化测试
- [ ] 字段大小计算测试
- [ ] 结构体大小计算测试

#### 3.2 嵌套结构体测试 (`optimizer/nested_test.go`)
- [ ] 两层嵌套测试
- [ ] 多层嵌套测试
- [ ] 循环引用检测测试

#### 3.3 跨包引用测试 (`optimizer/crosspkg_test.go`)
- [ ] 同包结构体引用测试
- [ ] 跨包结构体引用测试
- [ ] 外部包跳过测试

#### 3.4 跳过规则测试 (`analyzer/skip_test.go`)
- [ ] 目录跳过测试
- [ ] 文件跳过测试
- [ ] 方法名跳过测试

#### 3.5 报告生成测试 (`reporter/reporter_test.go`)
- [ ] TXT 格式测试
- [ ] Markdown 格式测试
- [ ] HTML 格式测试

#### 3.6 集成测试 (`integration/integration_test.go`)
- [ ] 完整流程测试
- [ ] 备份功能测试
- [ ] 写入功能测试

### 阶段 4: 验证与优化

#### 4.1 自我验证
- [ ] 使用 testdata 中的测试案例验证
- [ ] 验证内存节省计算准确性
- [ ] 验证字段重排正确性

#### 4.2 性能优化
- [ ] 优化大项目扫描速度
- [ ] 缓存已分析的结构体

#### 4.3 边界情况处理
- [ ] 泛型结构体处理
- [ ] 空结构体处理
- [ ] 单字段结构体处理
- [ ] 带 tag 字段处理

## 实现顺序

```
1. internal/utils/utils.go     (基础工具)
2. optimizer/size.go           (大小计算，其他模块依赖)
3. optimizer/field.go          (字段分析)
4. analyzer/analyzer.go        (包分析)
5. optimizer/optimizer.go      (核心优化逻辑)
6. reporter/reporter.go        (报告生成)
7. writer/writer.go            (源码写入)
8. cmd/structoptimizer/main.go (CLI 入口)
9. 测试用例
10. 集成验证
```

## 关键依赖

```go
import (
    "go/ast"
    "go/parser"
    "go/token"
    "go/types"
    "golang.org/x/tools/go/packages"
)
```

## 风险与应对

| 风险 | 应对措施 |
|------|----------|
| 跨包引用解析复杂 | 使用 `go/packages` 的完整类型信息 |
| 嵌套层级过深 | 设置最大递归深度，检测循环引用 |
| AST 重写丢失格式 | 使用 `go/printer` 保留格式 |
| 泛型处理复杂 | 第一版先跳过泛型结构体 |

## 完成标准

1. ✅ 所有核心功能实现
2. ✅ 所有测试用例通过
3. ✅ 能够成功优化示例中的结构体
4. ✅ 生成准确的优化报告
5. ✅ 代码符合 Go 规范
