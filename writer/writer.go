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
	"sort"
	"strings"
	"time"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// SourceWriter writes optimized source code
type SourceWriter struct {
	config *Config
	fset   *token.FileSet
}

// Config holds writer configuration
type Config struct {
	Backup  bool
	Verbose int
}

// NewSourceWriter creates a new source writer
func NewSourceWriter(cfg *Config) *SourceWriter {
	return &SourceWriter{
		config: cfg,
		fset:   token.NewFileSet(),
	}
}

// BackupFile creates a timestamped backup of the source file (to avoid overwriting)
func (w *SourceWriter) BackupFile(filePath string) (string, error) {
	if !w.config.Backup {
		return "", nil
	}

	// Create backup filename: xxx.go -> xxx.go.20060102_150405.bak (with timestamp)
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.%s.bak", filePath, timestamp)

	// Read original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Write backup file
	err = os.WriteFile(backupName, content, 0644)
	if err != nil {
		return "", err
	}

	w.log(1, "Backed up file: %s -> %s", filePath, backupName)
	return backupName, nil
}

// WriteStruct writes the optimized struct to the source file
func (w *SourceWriter) WriteStruct(filePath string, info *optimizer.StructInfo) error {
	w.log(1, "Writing optimized struct to file: %s", filePath)

	// Parse the file
	f, err := parser.ParseFile(w.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %v", err)
	}

	// Find and modify the struct
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

			// Reorder fields
			w.reorderStructFields(structType, info.Fields)
			modified = true
			return false
		}

		return true
	})

	if !modified {
		return fmt.Errorf("struct not found: %s", info.Name)
	}

	// Write back to file
	var buf bytes.Buffer
	err = printer.Fprint(&buf, w.fset, f)
	if err != nil {
		return fmt.Errorf("failed to format code: %v", err)
	}

	err = os.WriteFile(filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	w.log(1, "Optimized struct written: %s", info.Name)
	return nil
}

// reorderStructFields reorders struct fields
func (w *SourceWriter) reorderStructFields(structType *ast.StructType, fields []optimizer.FieldInfo) {
	if structType.Fields == nil {
		return
	}

	// Create field mapping
	fieldMap := make(map[string]*ast.Field)
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			// Anonymous (embedded) field
			typeStr := getTypeString(field.Type)
			fieldMap["embed:"+typeStr] = field
		} else {
			for _, name := range field.Names {
				fieldMap[name.Name] = field
			}
		}
	}

	// Create new field list
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

// getTypeString returns the string representation of a type
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

// RewriteFile rewrites the entire file using optimized structs
func (w *SourceWriter) RewriteFile(filePath string, optimizedStructs map[string]*optimizer.StructInfo) error {
	// Canonicalize file path (handles cross-platform path separator differences)
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to canonicalize file path: %v", err)
	}

	w.log(1, "Rewriting file: %s", filePath)

	// Parse the file
	f, err := parser.ParseFile(w.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %v", err)
	}

	// Collect all structs in this file that need modification (indexed by struct name)
	fileStructs := make(map[string]*optimizer.StructInfo)
	for _, info := range optimizedStructs {
		if info.File != "" && info.Optimized {
			// Compare after canonicalization
			absInfoFile, err := filepath.Abs(info.File)
			if err != nil {
				continue // skip paths that cannot be canonicalized
			}
			if absInfoFile == absFilePath {
				fileStructs[info.Name] = info
			}
		}
	}

	if len(fileStructs) == 0 {
		return nil // no structs to modify
	}

	// Modify all matching structs
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

			// Reorder fields
			w.reorderStructFields(structType, info.Fields)
			modified = true
		}

		return true
	})

	if !modified {
		return nil
	}

	// Write back to file (formatted with go/format)
	var buf bytes.Buffer
	err = printer.Fprint(&buf, w.fset, f)
	if err != nil {
		return fmt.Errorf("failed to format code: %v", err)
	}

	// Apply go/format standard formatting (removes blank lines, preserves comments)
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("go fmt format failed: %v", err)
	}

	err = os.WriteFile(filePath, formatted, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	w.log(1, "File rewritten: %s (formatted with go fmt)", filePath)
	return nil
}

// WriteFiles writes multiple files
func (w *SourceWriter) WriteFiles(optimized map[string]*optimizer.StructInfo) error {
	// Group by file
	fileStructs := make(map[string][]*optimizer.StructInfo)
	for _, info := range optimized {
		if info.File != "" && info.Optimized {
			fileStructs[info.File] = append(fileStructs[info.File], info)
		}
	}

	// Process each file
	for filePath := range fileStructs {
		// Backup
		if w.config.Backup {
			backupPath, err := w.BackupFile(filePath)
			if err != nil {
				w.log(0, "Failed to back up file: %v", err)
				continue
			}
			// Clean up old backups (keep the most recent 3)
			if backupPath != "" {
				w.cleanupOldBackups(filePath)
			}
		}

		// Write
		err := w.RewriteFile(filePath, optimized)
		if err != nil {
			w.log(0, "Failed to write file: %v", err)
			continue
		}
	}

	return nil
}

// cleanupOldBackups removes old backup files, keeping the most recent 3
func (w *SourceWriter) cleanupOldBackups(filePath string) {
	// Find all backup files: xxx.go.YYYYMMDD_HHMMSS.bak
	pattern := filePath + ".*.bak"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return // ignore errors
	}

	// If more than 3 backups, delete the oldest
	if len(matches) > 3 {
		// Sort by filename (timestamp is in the filename, earliest first after sorting)
		sort.Strings(matches)
		// Delete oldest files
		for i := 0; i < len(matches)-3; i++ {
			os.Remove(matches[i])
			w.log(2, "Cleaning old backup file: %s", matches[i])
		}
	}
}

// log emits a log line
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

// GetFileSet returns the file set
func (w *SourceWriter) GetFileSet() *token.FileSet {
	return w.fset
}

// FormatNode formats an AST node
func (w *SourceWriter) FormatNode(node ast.Node) (string, error) {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, w.fset, node)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// SortFieldInfos sorts field info entries
func SortFieldInfos(fields []optimizer.FieldInfo, sortSameSize bool) []optimizer.FieldInfo {
	return optimizer.ReorderFields(fields, sortSameSize, nil)
}

// CreateFieldInfo creates a field info entry (for testing)
func CreateFieldInfo(name string, size, align int64, isEmbed bool, typeName string) optimizer.FieldInfo {
	return optimizer.FieldInfo{
		Name:     name,
		Size:     size,
		Align:    align,
		IsEmbed:  isEmbed,
		TypeName: typeName,
	}
}

// ReadFile reads file content
func ReadFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteFile writes content to a file
func WriteFile(filePath string, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// BackupAndWrite creates a backup and writes to a file
func BackupAndWrite(filePath, content string, backup bool) error {
	if backup {
		dir := filepath.Dir(filePath)
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		backupName := fmt.Sprintf("%s/%s.backup%s", dir, name, ext)

		origContent, err := os.ReadFile(filePath)
		if err == nil {
			if err := os.WriteFile(backupName, origContent, 0644); err != nil {
				return fmt.Errorf("failed to back up file: %v", err)
			}
		}
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// GetStructFields returns the field list of a struct (for debugging)
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
					// Anonymous (embedded) field
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

// CompareFields compares two field lists
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

// FieldsChanged checks whether fields have changed
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

// GenerateStructCode generates struct source code
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

// PrintFields prints field info (for debugging)
func PrintFields(fields []optimizer.FieldInfo) {
	for i, f := range fields {
		fmt.Printf("%2d. %-20s size=%3d align=%2d", i+1, f.Name, f.Size, f.Align)
		if f.IsEmbed {
			fmt.Print(" (embedded)")
		}
		fmt.Println()
	}
}

// GroupFieldsBySize groups fields by their size
func GroupFieldsBySize(fields []optimizer.FieldInfo) map[int64][]optimizer.FieldInfo {
	groups := make(map[int64][]optimizer.FieldInfo)
	for _, f := range fields {
		groups[f.Size] = append(groups[f.Size], f)
	}
	return groups
}

// CalculatePadding computes the total padding size
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

	// Trailing padding
	if offset%maxAlign != 0 {
		padding += maxAlign - (offset % maxAlign)
	}

	return padding
}
