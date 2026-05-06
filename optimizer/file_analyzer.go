package optimizer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// analyzeStructFromFile analyzes a struct by parsing only the file (no package loading, fast path)
func analyzeStructFromFile(filePath, structName, pkgPath string) (*StructInfo, *types.Struct, error) {
	// Parse file
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse file failed: %w", err)
	}

	// Find struct definition
	var foundDecl *ast.TypeSpec
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != structName {
				continue
			}

			if _, ok := ts.Type.(*ast.StructType); ok {
				foundDecl = ts
				break
			}
		}
		if foundDecl != nil {
			break
		}
	}

	if foundDecl == nil {
		return nil, nil, fmt.Errorf("struct %s not found in file", structName)
	}

	// Extract field info from AST
	info := &StructInfo{
		Name:    structName,
		PkgPath: pkgPath,
		File:    filePath,
	}

	// Get the package directory (used to find type definitions in the same package)
	pkgDir := ""
	if filePath != "" {
		pkgDir = filepath.Dir(filePath)
	}

	st, fields := extractFieldsFromAST(foundDecl, fset, pkgDir)
	info.Fields = fields
	info.OrigOrder = extractFieldNames(fields)
	// Use CalcStructSizeFromFields to compute size (based on field size and align).
	// Note: Do not use types.Sizes because st is a simplified Struct with field type Invalid.
	info.OrigSize = CalcStructSizeFromFields(fields)

	return info, st, nil
}

// extractFieldsFromAST extracts field info from an AST
func extractFieldsFromAST(ts *ast.TypeSpec, fset *token.FileSet, pkgDir string) (*types.Struct, []FieldInfo) {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, nil
	}

	var fields []FieldInfo
	var varFields []*types.Var

	for _, f := range st.Fields.List {
		typeName := extractTypeName(f.Type)
		size, align := estimateFieldSizeWithLookup(f.Type, pkgDir)

		// Check if it's an embedded field
		isEmbed := len(f.Names) == 0

		// Get field name (for FieldInfo)
		fieldName := getFieldName(f)

		fi := FieldInfo{
			Name:     fieldName,
			Size:     size,
			Align:    align,
			TypeName: typeName,
			IsEmbed:  isEmbed, // correctly set the embedded field flag
		}

		if f.Tag != nil {
			fi.Tag = strings.Trim(f.Tag.Value, "`")
		}

		fields = append(fields, fi)

		// Create types.Var for subsequent processing.
		// Note: embedded fields use the type name in types.Var to avoid "multifields with the same name" errors.
		typesFieldName := fieldName
		if isEmbed {
			typesFieldName = typeName
		}
		varFields = append(varFields, types.NewField(f.Pos(), nil, typesFieldName, types.Typ[types.Invalid], false))
	}

	// Create a simplified types.Struct
	return types.NewStruct(varFields, nil), fields
}

// extractTypeName extracts the type name from an AST expression (keeping the package prefix)
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + extractTypeName(t.X)
	case *ast.SelectorExpr:
		// Keep package name: pkg.TypeName
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.ArrayType:
		return "[]" + extractTypeName(t.Elt)
	case *ast.MapType:
		return "map[" + extractTypeName(t.Key) + "]" + extractTypeName(t.Value)
	case *ast.ChanType:
		return "chan " + extractTypeName(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}

// getFieldName returns the field name
func getFieldName(f *ast.Field) string {
	if len(f.Names) > 0 && f.Names[0] != nil {
		return f.Names[0].Name
	}
	// Embedded field returns an empty string
	return ""
}

// estimateFieldSize estimates field size (platform-aware for pointer and interface)
func estimateFieldSize(expr ast.Expr) (size, align int64) {
	ptrSize := sizeofPtr()
	ptrAlign := alignofPtr()
	switch t := expr.(type) {
	case *ast.Ident:
		return sizeOfIdent(t.Name)
	case *ast.StarExpr:
		return ptrSize, ptrAlign
	case *ast.ArrayType:
		if t.Len == nil {
			return 24, ptrAlign // slice header
		}
		elemSize, elemAlign := estimateFieldSize(t.Elt)
		if length := parseArrayLength(t.Len); length > 0 {
			return elemSize * length, elemAlign
		}
		return elemSize, elemAlign
	case *ast.MapType:
		return ptrSize, ptrAlign
	case *ast.ChanType:
		return ptrSize, ptrAlign
	case *ast.InterfaceType:
		return 16, ptrAlign
	default:
		return ptrSize, ptrAlign
	}
}

// EstimateFieldSizeWithLookup estimates field size (with type lookup) - exported for testing
func EstimateFieldSizeWithLookup(expr ast.Expr, pkgDir string) (size, align int64) {
	return estimateFieldSizeWithLookup(expr, pkgDir)
}

// estimateFieldSizeWithLookup estimates field size (with type lookup)
func estimateFieldSizeWithLookup(expr ast.Expr, pkgDir string) (size, align int64) {
	ptrSize := sizeofPtr()
	ptrAlign := alignofPtr()
	switch t := expr.(type) {
	case *ast.Ident:
		// For identifiers, try to find the type definition within the same package
		if pkgDir != "" {
			underlyingKind := findTypeUnderlyingInPackage(pkgDir, t.Name)
			if underlyingKind != types.Invalid {
				return basicSize(underlyingKind)
			}
			// If not a basic type, try to detect if it's a struct type
			if structSize := findStructSizeInPackage(pkgDir, t.Name); structSize > 0 {
				return structSize, 8
			}
		}
		return sizeOfIdent(t.Name)
	case *ast.SelectorExpr:
		// Handle type with package prefix (e.g. time.Time)
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgName := ident.Name
			typeName := t.Sel.Name
			// Try to find the struct size for standard library or known external packages
			if size := getExternalStructSize(pkgName, typeName, pkgDir); size > 0 {
				return size, 8
			}
		}
		return 8, 8 // unknown external type
	case *ast.StarExpr:
		return ptrSize, ptrAlign
	case *ast.ArrayType:
		if t.Len == nil {
			return 24, ptrAlign // slice
		}
		// Parse fixed-length array
		elemSize, elemAlign := estimateFieldSizeWithLookup(t.Elt, pkgDir)
		if length := parseArrayLength(t.Len); length > 0 {
			return elemSize * length, elemAlign
		}
		return elemSize, elemAlign // fallback when unable to parse
	case *ast.MapType:
		return ptrSize, ptrAlign
	case *ast.ChanType:
		return ptrSize, ptrAlign
	case *ast.InterfaceType:
		return 16, ptrAlign
	case *ast.StructType:
		// Handle inline struct directly (anonymous nested struct)
		size := calcInlineStructSize(t, pkgDir)
		return size, ptrAlign
	default:
		return ptrSize, ptrAlign
	}
}

// getExternalStructSize returns the size of a struct in an external package (standard library / third party)
func getExternalStructSize(pkgName, typeName, localPkgDir string) int64 {
	// Sizes for common standard library types
	if pkgName == "time" {
		switch typeName {
		case "Time":
			return 24 // actual size of time.Time
		case "Duration":
			return 8
		case "Location":
			return 8 // pointer
		}
	}
	if pkgName == "sync" {
		switch typeName {
		case "Mutex":
			return 8
		case "RWMutex":
			return 16
		case "WaitGroup":
			return 16
		case "Cond":
			return 16
		case "Once":
			return 8
		}
	}
	if pkgName == "context" {
		switch typeName {
		case "Context":
			return 16 // interface
		case "CancelFunc":
			return 8 // function pointer
		}
	}
	if pkgName == "bytes" {
		switch typeName {
		case "Buffer":
			return 72
		}
	}
	if pkgName == "strings" {
		switch typeName {
		case "Builder":
			return 24
		}
	}
	if pkgName == "net" {
		switch typeName {
		case "IP":
			return 24 // slice
		case "IPMask":
			return 24 // slice
		}
	}
	if pkgName == "url" {
		switch typeName {
		case "URL":
			return 120
		}
	}
	if pkgName == "http" {
		switch typeName {
		case "Request":
			return 480
		case "Response":
			return 320
		case "Header":
			return 24 // map
		}
	}
	if pkgName == "json" {
		switch typeName {
		case "RawMessage":
			return 24 // slice
		}
	}
	
	// If not standard library, try to find in GOPATH or module cache
	if localPkgDir != "" {
		// Try to find in sibling directory or vendor
		if size := findStructSizeInVendorOrDep(pkgName, typeName, localPkgDir); size > 0 {
			return size
		}
	}
	
	return 0
}

// findStructSizeInVendorOrDep finds struct size in vendor or dependency directories
func findStructSizeInVendorOrDep(pkgName, typeName, localPkgDir string) int64 {
	// Try to find in vendor directory
	vendorDir := filepath.Join(localPkgDir, "..", "..", "vendor", pkgName)
	if size := findStructSizeInPackage(vendorDir, typeName); size > 0 {
		return size
	}
	
	// Try to find in GOPATH/pkg/mod (simplified, only checks common paths).
	// Real projects may need more complex path resolution.
	return 0
}

// findStructSizeInPackage finds the size of a struct type within a package
func findStructSizeInPackage(pkgDir, typeName string) int64 {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		if size := findStructSizeInFile(filePath, typeName); size > 0 {
			return size
		}
	}

	return 0
}

// findStructSizeInFile finds the size of a struct type in a file
func findStructSizeInFile(filePath, typeName string) int64 {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return 0
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != typeName {
				continue
			}

			if st, ok := ts.Type.(*ast.StructType); ok {
				return calcInlineStructSize(st, filepath.Dir(filePath))
			}
		}
	}

	return 0
}

// calcInlineStructSize calculates the size of an inline struct
func calcInlineStructSize(st *ast.StructType, pkgDir string) int64 {
	if st == nil || st.Fields == nil {
		return 0
	}

	var offset int64 = 0
	var maxAlign int64 = 1

	for _, field := range st.Fields.List {
		size, align := estimateFieldSizeWithLookup(field.Type, pkgDir)

		// Alignment
		if offset%align != 0 {
			offset += align - (offset % align)
		}

		offset += size
		if align > maxAlign {
			maxAlign = align
		}
	}

	// Trailing padding
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	return offset
}

// findTypeUnderlyingInPackage finds the underlying type of a type within a package.
// Returns types.BasicKind if it is a basic type, otherwise returns types.Invalid.
func findTypeUnderlyingInPackage(pkgDir, typeName string) types.BasicKind {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return types.Invalid
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		if kind := findTypeUnderlyingInFile(filePath, typeName); kind != types.Invalid {
			return kind
		}
	}

	return types.Invalid
}

// findTypeUnderlyingInFile finds the underlying type of a type within a file
func findTypeUnderlyingInFile(filePath, typeName string) types.BasicKind {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return types.Invalid
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != typeName {
				continue
			}

			// Check underlying type
			return getBasicKindFromExpr(ts.Type)
		}
	}

	return types.Invalid
}

// getBasicKindFromExpr gets the basic type kind from an AST expression
func getBasicKindFromExpr(expr ast.Expr) types.BasicKind {
	switch t := expr.(type) {
	case *ast.Ident:
		return identToBasicKind(t.Name)
	case *ast.ArrayType:
		// Array/slice
		return types.Invalid
	case *ast.StarExpr:
		// Pointer
		return types.Invalid
	default:
		return types.Invalid
	}
}

// identToBasicKind converts an identifier name to a basic type kind
func identToBasicKind(name string) types.BasicKind {
	switch name {
	case "bool":
		return types.Bool
	case "int8":
		return types.Int8
	case "uint8", "byte":
		return types.Uint8
	case "int16":
		return types.Int16
	case "uint16":
		return types.Uint16
	case "int32", "rune":
		return types.Int32
	case "uint32":
		return types.Uint32
	case "int64":
		return types.Int64
	case "uint64":
		return types.Uint64
	case "int":
		return types.Int
	case "uint":
		return types.Uint
	case "float32":
		return types.Float32
	case "float64":
		return types.Float64
	case "string":
		return types.String
	case "uintptr":
		return types.Uintptr
	default:
		return types.Invalid
	}
}

// sizeOfIdent estimates the size from an identifier name
func sizeOfIdent(name string) (int64, int64) {
	switch name {
	case "bool", "byte":
		return 1, 1
	case "int8", "uint8":
		return 1, 1
	case "int16", "uint16":
		return 2, 2
	case "int32", "uint32", "rune":
		return 4, 4
	case "int64", "uint64":
		return 8, 8
	case "int", "uint", "uintptr":
		return 8, 8 // 64-bit system
	case "float32":
		return 4, 4
	case "float64":
		return 8, 8
	case "string":
		return 16, 8
	default:
		return 8, 8 // unknown type
	}
}

// parseArrayLength parses the array length
func parseArrayLength(expr ast.Expr) int64 {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.INT {
			// Remove numeric suffix (e.g. 10u)
			value := strings.TrimRight(e.Value, "uU")
			if n, err := strconv.ParseInt(value, 0, 64); err == nil {
				return n
			}
		}
	case *ast.ParenExpr:
		return parseArrayLength(e.X)
	}
	return 0
}

// extractFieldNames extracts field names
func extractFieldNames(fields []FieldInfo) []string {
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
