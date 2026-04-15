package writer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// SourceWriter 源码写入器
type SourceWriter struct {
	config *Config
	fset   *token.FileSet
}

// Config 写入器配置
type Config struct {
	Backup bool
	Verbose int
}

// NewSourceWriter 创建源码写入器
func NewSourceWriter(cfg *Config) *SourceWriter {
	return &SourceWriter{
		config: cfg,
		fset:   token.NewFileSet(),
	}
}

// BackupFile 备份源文件
func (w *SourceWriter) BackupFile(filePath string) (string, error) {
	if !w.config.Backup {
		return "", nil
	}

	// 创建备份文件名：xxx.go -> xxx.go.bak
	backupName := filePath + ".bak"

	// 读取原文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// 写入备份文件
	err = os.WriteFile(backupName, content, 0644)
	if err != nil {
		return "", err
	}

	w.log(1, "已备份文件：%s -> %s", filePath, backupName)
	return backupName, nil
}

// WriteStruct 写入优化后的结构体到源文件
func (w *SourceWriter) WriteStruct(filePath string, info *optimizer.StructInfo) error {
	w.log(1, "写入优化后的结构体到文件：%s", filePath)

	// 解析文件
	f, err := parser.ParseFile(w.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("解析文件失败：%v", err)
	}

	// 查找并修改结构体
	modified := false
	ast.Inspect(f, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != info.Name {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// 重排字段
			w.reorderStructFields(structType, info.Fields)
			modified = true
			return false
		}

		return true
	})

	if !modified {
		return fmt.Errorf("未找到结构体：%s", info.Name)
	}

	// 写回文件
	var buf bytes.Buffer
	err = printer.Fprint(&buf, w.fset, f)
	if err != nil {
		return fmt.Errorf("格式化代码失败：%v", err)
	}

	err = os.WriteFile(filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败：%v", err)
	}

	w.log(1, "已写入优化后的结构体：%s", info.Name)
	return nil
}

// reorderStructFields 重排结构体字段
func (w *SourceWriter) reorderStructFields(structType *ast.StructType, fields []optimizer.FieldInfo) {
	if structType.Fields == nil {
		return
	}

	// 创建字段映射
	fieldMap := make(map[string]*ast.Field)
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			// 匿名字段
			typeStr := getTypeString(field.Type)
			fieldMap["embed:"+typeStr] = field
		} else {
			for _, name := range field.Names {
				fieldMap[name.Name] = field
			}
		}
	}

	// 创建新的字段列表
	newFields := make([]*ast.Field, 0, len(fields))
	
	for _, fi := range fields {
		var key string
		if fi.IsEmbed {
			key = "embed:" + fi.TypeName
		} else {
			key = fi.Name
		}

		if field, ok := fieldMap[key]; ok {
			newFields = append(newFields, field)
		}
	}

	structType.Fields.List = newFields
}

// getTypeString 获取类型的字符串表示
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.SelectorExpr:
		return getTypeString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", t)
	}
}

// RewriteFile 重写整个文件（使用优化后的结构体）
func (w *SourceWriter) RewriteFile(filePath string, optimizedStructs map[string]*optimizer.StructInfo) error {
	w.log(1, "重写文件：%s", filePath)

	// 解析文件
	f, err := parser.ParseFile(w.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("解析文件失败：%v", err)
	}

	// 收集该文件中所有需要修改的结构体（按结构体名索引）
	fileStructs := make(map[string]*optimizer.StructInfo)
	for _, info := range optimizedStructs {
		if info.File == filePath && info.Optimized {
			fileStructs[info.Name] = info
		}
	}

	if len(fileStructs) == 0 {
		return nil // 没有需要修改的结构体
	}

	// 修改所有匹配的结构体
	modified := false
	ast.Inspect(f, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			info, exists := fileStructs[typeSpec.Name.Name]
			if !exists {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// 重排字段
			w.reorderStructFields(structType, info.Fields)
			modified = true
		}

		return true
	})

	if !modified {
		return nil
	}

	// 写回文件（使用 go/format 格式化）
	var buf bytes.Buffer
	err = printer.Fprint(&buf, w.fset, f)
	if err != nil {
		return fmt.Errorf("格式化代码失败：%v", err)
	}

	// 使用 go/format 进行标准格式化（删除空行，保留注释）
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("go fmt 格式化失败：%v", err)
	}

	err = os.WriteFile(filePath, formatted, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败：%v", err)
	}

	w.log(1, "文件已重写：%s (已使用 go fmt 格式化)", filePath)
	return nil
}

// WriteFiles 写入多个文件
func (w *SourceWriter) WriteFiles(optimized map[string]*optimizer.StructInfo) error {
	// 按文件分组
	fileStructs := make(map[string][]*optimizer.StructInfo)
	for _, info := range optimized {
		if info.File != "" && info.Optimized {
			fileStructs[info.File] = append(fileStructs[info.File], info)
		}
	}

	// 处理每个文件
	for filePath := range fileStructs {
		// 备份
		if w.config.Backup {
			_, err := w.BackupFile(filePath)
			if err != nil {
				w.log(0, "备份文件失败：%v", err)
				continue
			}
		}

		// 写入
		err := w.RewriteFile(filePath, optimized)
		if err != nil {
			w.log(0, "写入文件失败：%v", err)
			continue
		}
	}

	return nil
}

// log 日志输出
func (w *SourceWriter) log(level int, format string, args ...interface{}) {
	if level <= w.config.Verbose {
		prefix := ""
		if level == 0 {
			prefix = "[ERROR] "
		} else if level == 1 {
			prefix = "[INFO] "
		}
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// GetFileSet 获取文件集
func (w *SourceWriter) GetFileSet() *token.FileSet {
	return w.fset
}

// FormatNode 格式化 AST 节点
func (w *SourceWriter) FormatNode(node ast.Node) (string, error) {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, w.fset, node)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// SortFieldInfos 对字段信息进行排序
func SortFieldInfos(fields []optimizer.FieldInfo, sortSameSize bool) []optimizer.FieldInfo {
	return optimizer.ReorderFields(fields, sortSameSize, nil)
}

// CreateFieldInfo 创建字段信息（用于测试）
func CreateFieldInfo(name string, size, align int64, isEmbed bool, typeName string) optimizer.FieldInfo {
	return optimizer.FieldInfo{
		Name:     name,
		Size:     size,
		Align:    align,
		IsEmbed:  isEmbed,
		TypeName: typeName,
	}
}

// ReadFile 读取文件内容
func ReadFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteFile 写入文件内容
func WriteFile(filePath string, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// BackupAndWrite 备份并写入文件
func BackupAndWrite(filePath, content string, backup bool) error {
	if backup {
		dir := filepath.Dir(filePath)
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		backupName := fmt.Sprintf("%s/%s.backup%s", dir, name, ext)

		origContent, err := os.ReadFile(filePath)
		if err == nil {
			os.WriteFile(backupName, origContent, 0644)
		}
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// GetStructFields 获取结构体的字段列表（用于调试）
func GetStructFields(filePath, structName string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var fields []string
	ast.Inspect(f, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != structName {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		if structType.Fields != nil {
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					// 匿名字段
					fields = append(fields, getTypeString(field.Type))
				} else {
					for _, name := range field.Names {
						fields = append(fields, name.Name)
					}
				}
			}
		}
		return false
	})

	return fields, nil
}

// CompareFields 比较两个字段列表
func CompareFields(orig, new []string) bool {
	if len(orig) != len(new) {
		return false
	}
	for i := range orig {
		if orig[i] != new[i] {
			return false
		}
	}
	return true
}

// FieldsChanged 检查字段是否发生变化
func FieldsChanged(orig, new []optimizer.FieldInfo) bool {
	if len(orig) != len(new) {
		return true
	}
	for i := range orig {
		if orig[i].Name != new[i].Name || orig[i].IsEmbed != new[i].IsEmbed {
			return true
		}
	}
	return false
}

// GenerateStructCode 生成结构体代码
func GenerateStructCode(name string, fields []optimizer.FieldInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", name))

	for _, f := range fields {
		if f.IsEmbed {
			sb.WriteString(fmt.Sprintf("\t%s\n", f.TypeName))
		} else {
			sb.WriteString(fmt.Sprintf("\t%s %s\n", f.Name, f.TypeName))
		}
	}

	sb.WriteString("}")
	return sb.String()
}

// PrintFields 打印字段信息（用于调试）
func PrintFields(fields []optimizer.FieldInfo) {
	for i, f := range fields {
		fmt.Printf("%2d. %-20s size=%3d align=%2d", i+1, f.Name, f.Size, f.Align)
		if f.IsEmbed {
			fmt.Print(" (embedded)")
		}
		fmt.Println()
	}
}

// GroupFieldsBySize 按大小分组字段
func GroupFieldsBySize(fields []optimizer.FieldInfo) map[int64][]optimizer.FieldInfo {
	groups := make(map[int64][]optimizer.FieldInfo)
	for _, f := range fields {
		groups[f.Size] = append(groups[f.Size], f)
	}
	return groups
}

// CalculatePadding 计算填充大小
func CalculatePadding(fields []optimizer.FieldInfo) int64 {
	var offset int64 = 0
	var padding int64 = 0
	var maxAlign int64 = 1

	for _, f := range fields {
		if offset%f.Align != 0 {
			p := f.Align - (offset % f.Align)
			padding += p
			offset += p
		}
		offset += f.Size
		if f.Align > maxAlign {
			maxAlign = f.Align
		}
	}

	// 末尾填充
	if offset%maxAlign != 0 {
		padding += maxAlign - (offset % maxAlign)
	}

	return padding
}
