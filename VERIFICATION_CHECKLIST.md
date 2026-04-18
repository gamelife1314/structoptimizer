# StructOptimizer 修改验证 Checklist

> **重要**：每次代码修改后，必须逐项检查本清单，确保所有验证项通过。

---

## 📋 使用说明

### 检查流程

1. **运行完整测试套件**
   ```bash
   go test ./... -v
   ```

2. **逐项验证**
   - 对照本清单的每个检查项
   - 确认测试通过
   - 记录任何失败项

3. **提交前确认**
   - 所有核心项必须 ✅ 通过
   - 如有失败，必须修复后再提交

---

## ✅ 核心功能验证（必须 100% 通过）

### 1. 匿名字段识别

**问题描述**：修复前 `f.Name == f.TypeName` 永远为 false，导致 `HasEmbed` 始终为 false

**验证项**：
- [ ] `TestAddReportEmbedDetection` - 匿名字段在报告中正确标记
- [ ] `TestGOPATHEmbeddedFieldDetection` - GOPATH项目匿名字段识别
- [ ] `TestEmbeddedFieldInGOPATHProject` - 匿名字段AST解析
- [ ] `TestComplexProjectEmbeddedFields` - 复杂项目匿名字段

**运行命令**：
```bash
go test ./optimizer -run "TestAddReportEmbedDetection|TestGOPATHEmbeddedFieldDetection|TestEmbeddedFieldInGOPATHProject" -v
```

**预期结果**：
- ✅ `HasEmbed` 字段在包含匿名字段时为 true
- ✅ 匿名字段（小写开头）被正确收集
- ✅ 报告中正确显示匿名字段信息

---

### 2. 同包不同文件未导出结构体

**问题描述**：同包但不同源文件中定义的未导出类型（小写开头）无法识别

**验证项**：
- [ ] `TestSamePackageUnexportedStructsCrossFiles` - 同包未导出结构体收集
- [ ] `TestUnexportedStructsNaming` - 未导出类型命名验证（小写开头）
- [ ] `TestSamePackageCrossFileDetection` - 跨文件检测
- [ ] `TestGOPATHUnexportedStructCrossFile` - GOPATH跨文件未导出

**运行命令**：
```bash
go test ./optimizer -run "TestSamePackageUnexportedStructsCrossFiles|TestUnexportedStructsNaming|TestGOPATHUnexportedStructCrossFile" -v
```

**预期结果**：
- ✅ 未导出类型（如 `internalConfig`、`localCache`）被正确收集
- ✅ 未被错误跳过
- ✅ 大小计算正确
- ✅ 命名约定验证：小写字母开头 = 未导出

---

### 3. 重定义类型大小识别

**问题描述**：`type CustomInt int64` 等重定义类型应识别为底层类型大小

**验证项**：
- [ ] `TestTypeAliasExactSizeCalculation` - 精确大小验证（9种类型）
- [ ] `TestTypeAliasVsOriginal` - 与原始类型大小对比
- [ ] `TestTypeAliasSizeFix` - 包含重定义类型的结构体
- [ ] `TestGOPATHTypeAliasSize` - GOPATH项目中类型别名

**运行命令**：
```bash
go test ./optimizer -run "TestTypeAliasExactSizeCalculation|TestTypeAliasVsOriginal|TestTypeAliasSizeFix" -v
```

**预期结果**：
- ✅ 1字节类型（`uint8`、`bool`）：1字节
- ✅ 2字节类型（`uint16`、`int16`）：2字节
- ✅ 4字节类型（`uint32`、`int32`、`float32`）：4字节
- ✅ 8字节类型（`uint64`、`int64`、`float64`）：8字节
- ✅ 16字节类型（`string`）：16字节
- ✅ 重定义类型与原始类型大小完全一致

---

### 4. skip-by-methods 功能

**问题描述**：通过方法名跳过结构体优化（支持通配符）

**验证项**：
- [ ] `TestGOPATHSkipByMethods` - 基本功能
- [ ] `TestSkipByMethodsWithWildcard` - 通配符匹配
- [ ] `TestMethodDetectionInGOPATH` - 方法检测
- [ ] `TestComplexProjectSkipByMethods` - 复杂项目

**运行命令**：
```bash
go test ./optimizer -run "TestGOPATHSkipByMethods|TestSkipByMethodsWithWildcard|TestMethodDetectionInGOPATH" -v
```

**预期结果**：
- ✅ 具有指定方法的结构体被跳过
- ✅ 通配符匹配正确（`Encode*`、`*JSON`、`Marshal*`）
- ✅ 无方法的结构体正常优化
- ✅ 跳过原因显示 "通过方法指定跳过：XXX"

---

### 5. vendor 第三方库跳过

**问题描述**：vendor 目录中的第三方库结构体不应被优化

**验证项**：
- [ ] `TestGOPATHVendorPackageSkipped` - vendor包跳过
- [ ] `TestComplexProjectVendorSkipped` - 复杂项目vendor跳过
- [ ] `TestIsVendorPackage` - vendor判断函数

**运行命令**：
```bash
go test ./optimizer -run "TestGOPATHVendorPackageSkipped|TestComplexProjectVendorSkipped|TestIsVendorPackage" -v
```

**预期结果**：
- ✅ vendor 中的结构体不出现在优化报告中
- ✅ 在收集阶段被正确跳过
- ✅ `isVendorPackage()` 函数正确识别 vendor 路径

---

### 6. 数组大小计算

**问题描述**：`[10]int64` 应计算为 80 字节，而非 8 字节

**验证项**：
- [ ] `TestParseArrayLength` - 数组长度解析
- [ ] `TestEstimateFieldSizeArray` - 数组大小估算
- [ ] `TestCalcStructSizeWithArray` - 包含数组的结构体大小

**运行命令**：
```bash
go test ./optimizer -run "TestParseArrayLength|TestEstimateFieldSizeArray|TestCalcStructSizeWithArray" -v
```

**预期结果**：
- ✅ `[10]int64` = 80 字节
- ✅ `[5]int32` = 20 字节
- ✅ `[]int64`（slice）= 24 字节
- ✅ 支持十进制、十六进制、八进制数字

---

### 7. 10层深度嵌套

**问题描述**：支持至少 10 层深度嵌套结构体优化

**验证项**：
- [ ] `TestComplexProjectNestedLevels` - 10层嵌套收集验证

**运行命令**：
```bash
go test ./optimizer -run "TestComplexProjectNestedLevels" -v
```

**预期结果**：
- ✅ Level0 到 Level10 所有层级都被收集
- ✅ 11 个嵌套结构体全部处理
- ✅ 无循环引用或无限递归

---

## 🔧 回归测试（必须通过）

### 8. 完整测试套件

**运行命令**：
```bash
go test ./... -v
```

**预期结果**：
- ✅ 所有包测试通过
- ✅ 无编译错误
- ✅ 无运行时 panic

### 9. 核心模块测试

**运行命令**：
```bash
go test ./optimizer ./analyzer ./writer ./reporter -v
```

**预期结果**：
- ✅ optimizer 包：所有核心测试通过
- ✅ analyzer 包：分析功能测试通过
- ✅ writer 包：写入功能测试通过
- ✅ reporter 包：报告生成测试通过

---

## 🎯 快速验证脚本

### 一键运行所有核心测试

```bash
#!/bin/bash
# 保存为 scripts/verify_checklist.sh

echo "========================================="
echo "StructOptimizer 修改验证 Checklist"
echo "========================================="
echo ""

# 1. 匿名字段识别
echo "1/7. 匿名字段识别..."
go test ./optimizer -run "TestAddReportEmbedDetection|TestGOPATHEmbeddedFieldDetection" -v | grep -E "(PASS|FAIL)"

# 2. 同包不同文件未导出
echo ""
echo "2/7. 同包不同文件未导出结构体..."
go test ./optimizer -run "TestSamePackageUnexportedStructsCrossFiles|TestGOPATHUnexportedStructCrossFile" -v | grep -E "(PASS|FAIL)"

# 3. 重定义类型大小
echo ""
echo "3/7. 重定义类型大小识别..."
go test ./optimizer -run "TestTypeAliasExactSizeCalculation|TestTypeAliasVsOriginal" -v | grep -E "(PASS|FAIL)"

# 4. skip-by-methods
echo ""
echo "4/7. skip-by-methods 功能..."
go test ./optimizer -run "TestGOPATHSkipByMethods|TestSkipByMethodsWithWildcard" -v | grep -E "(PASS|FAIL)"

# 5. vendor跳过
echo ""
echo "5/7. vendor 第三方库跳过..."
go test ./optimizer -run "TestGOPATHVendorPackageSkipped|TestIsVendorPackage" -v | grep -E "(PASS|FAIL)"

# 6. 数组大小计算
echo ""
echo "6/7. 数组大小计算..."
go test ./optimizer -run "TestParseArrayLength|TestEstimateFieldSizeArray" -v | grep -E "(PASS|FAIL)"

# 7. 10层嵌套
echo ""
echo "7/7. 10层深度嵌套..."
go test ./optimizer -run "TestComplexProjectNestedLevels" -v | grep -E "(PASS|FAIL)"

# 完整测试
echo ""
echo "========================================="
echo "运行完整测试套件..."
echo "========================================="
go test ./... 2>&1 | grep -E "^(ok|FAIL)"

echo ""
echo "========================================="
echo "验证完成！请检查上方所有测试结果"
echo "========================================="
```

### 使用方法

```bash
# 赋予执行权限
chmod +x scripts/verify_checklist.sh

# 运行验证
./scripts/verify_checklist.sh
```

---

## 📊 验证记录模板

每次修改后，填写以下记录：

```markdown
### 修改验证记录

**修改日期**：YYYY-MM-DD
**修改内容**：（简要描述）

#### 核心功能验证

| 检查项 | 状态 | 备注 |
|--------|------|------|
| 1. 匿名字段识别 | ✅/❌ | |
| 2. 同包不同文件未导出 | ✅/❌ | |
| 3. 重定义类型大小 | ✅/❌ | |
| 4. skip-by-methods | ✅/❌ | |
| 5. vendor跳过 | ✅/❌ | |
| 6. 数组大小计算 | ✅/❌ | |
| 7. 10层嵌套 | ✅/❌ | |

#### 回归测试

| 测试范围 | 状态 | 备注 |
|---------|------|------|
| 完整测试套件 | ✅/❌ | |
| optimizer包 | ✅/❌ | |
| analyzer包 | ✅/❌ | |
| writer包 | ✅/❌ | |
| reporter包 | ✅/❌ | |

#### 失败项说明

（如有失败，说明原因和修复计划）

**验证人**：（签名）
```

---

## ⚠️ 重要提醒

### 禁止提交的情况

❌ **以下情况禁止提交代码**：

1. 任何核心功能验证项失败
2. 完整测试套件有失败
3. 引入编译错误
4. 导致已有测试用例失败

### 必须额外验证的情况

⚠️ **以下修改需要额外验证**：

| 修改范围 | 需要额外验证的测试 |
|---------|------------------|
| 修改字段大小计算逻辑 | 所有类型别名测试 + 数组大小测试 |
| 修改匿名字段处理 | 所有匿名字段相关测试 |
| 修改包加载逻辑 | GOPATH测试 + vendor跳过测试 |
| 修改方法索引逻辑 | skip-by-methods测试 |
| 修改写入逻辑 | writer包所有测试 |
| 修改收集逻辑 | 同包不同文件测试 + 嵌套测试 |

---

## 📚 相关文档

- `README.md` - 项目使用说明
- `design.md` - 设计文档
- `plan.md` - 执行计划

---

**文档版本**：v1.0  
**创建日期**：2026-04-18  
**维护者**：StructOptimizer Team  
**更新频率**：每次新增核心功能时更新
