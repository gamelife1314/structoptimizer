package optimizer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// isInterfaceType 快速判断是否是接口类型
func isInterfaceType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch t := typ.(type) {
	case *types.Interface:
		return true
	case *types.Named:
		return isInterfaceType(t.Underlying())
	case *types.Pointer:
		return isInterfaceType(t.Elem())
	default:
		return false
	}
}

// isStructType 快速判断是否是结构体类型
func isStructType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch t := typ.(type) {
	case *types.Struct:
		return true
	case *types.Named:
		return isStructType(t.Underlying())
	case *types.Pointer:
		return isStructType(t.Elem())
	default:
		return false
	}
}

// collectStructs 收集所有需要处理的结构体（不执行优化，只收集依赖）
func (o *Optimizer) collectStructs(pkgPath, structName, filePath string, depth, level int) {
	key := pkgPath + "." + structName

	// 快速去重：使用 map 代替 slice 遍历
	o.mu.Lock()
	// 检查是否已收集或正在收集
	if _, ok := o.optimized[key]; ok {
		o.mu.Unlock()
		return
	}
	if o.collecting[key] {
		o.mu.Unlock()
		return
	}
	// 标记为正在收集
	o.collecting[key] = true
	o.mu.Unlock()

	// 检查递归深度
	if depth > o.maxDepth {
		return
	}

	// 检查是否是第三方包
	if isVendorPackage(pkgPath) || !o.isProjectPackage(pkgPath) {
		return
	}

	// 加载包并查找结构体（需要完整类型信息来收集嵌套依赖）
	pkg, err := o.analyzer.LoadPackage(pkgPath)
	if err != nil {
		o.Log(3, "加载包失败：%s: %v", pkgPath, err)
		return
	}

	// 在包中查找结构体
	st, filePath, err := o.findStructInPackage(pkg, structName)
	if err != nil {
		o.Log(3, "查找结构体失败：%s.%s: %v", pkgPath, structName, err)
		return
	}

	// 添加到队列
	task := &StructTask{
		PkgPath:    pkgPath,
		StructName: structName,
		FilePath:   filePath,
		Depth:      depth,
		Level:      level,
	}

	o.mu.Lock()
	o.structQueue = append(o.structQueue, task)
	o.mu.Unlock()

	// 分析字段，收集嵌套结构体
	o.collectNestedStructs(st, structName, pkgPath, filePath, depth, level)
}

// findStructInPackage 在已加载的包中查找结构体
func (o *Optimizer) findStructInPackage(pkg *packages.Package, structName string) (*types.Struct, string, error) {
	for _, syntax := range pkg.Syntax {
		filePath := pkg.Fset.File(syntax.Pos()).Name()

		// 在文件中查找结构体定义
		for _, decl := range syntax.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if typeSpec.Name.Name != structName {
					continue
				}

				// 获取类型信息
				obj := pkg.TypesInfo.ObjectOf(typeSpec.Name)
				if obj == nil {
					continue
				}

				if named, ok := obj.Type().(*types.Named); ok {
					if st, ok := named.Underlying().(*types.Struct); ok {
						return st, filePath, nil
					}
				}
			}
		}
	}

	return nil, "", fmt.Errorf("struct %s not found in package", structName)
}

// collectNestedStructs 收集嵌套的结构体依赖
func (o *Optimizer) collectNestedStructs(st *types.Struct, structName, pkgPath, filePath string, depth, level int) {
	if st == nil {
		return
	}

	// 遍历字段
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		fieldType := field.Type()

		// 跳过接口、标准库、第三方包
		if isInterfaceType(fieldType) {
			continue
		}

		pkg := o.getTypePkg(fieldType)
		if isStandardLibraryPkg(pkg) || isVendorPackage(pkg) || !o.isProjectPackage(pkg) {
			continue
		}

		// 检查是否是结构体类型
		if isStructType(fieldType) {
			typeName := o.getTypeName(fieldType)
			fieldPkg := pkg
			if fieldPkg == "" {
				fieldPkg = pkgPath
			}

			if fieldPkg != "" && o.isProjectPackage(fieldPkg) {
				// 递归收集，层级 +1
				o.collectStructs(fieldPkg, typeName, filePath, depth+1, level+1)
			}
		}
	}
}

// getTypePkg 获取类型的包路径
func (o *Optimizer) getTypePkg(typ types.Type) string {
	if typ == nil {
		return ""
	}
	switch t := typ.(type) {
	case *types.Named:
		if obj := t.Obj(); obj != nil {
			if pkg := obj.Pkg(); pkg != nil {
				return pkg.Path()
			}
		}
		return ""
	case *types.Pointer:
		return o.getTypePkg(t.Elem())
	case *types.Slice:
		return o.getTypePkg(t.Elem())
	case *types.Array:
		return o.getTypePkg(t.Elem())
	case *types.Map:
		// Map 的包路径通常不重要
		return ""
	default:
		return ""
	}
}

// getTypeName 获取类型名称
func (o *Optimizer) getTypeName(typ types.Type) string {
	if typ == nil {
		return ""
	}
	switch t := typ.(type) {
	case *types.Named:
		return t.Obj().Name()
	case *types.Pointer:
		return o.getTypeName(t.Elem())
	case *types.Slice:
		return o.getTypeName(t.Elem())
	case *types.Array:
		return o.getTypeName(t.Elem())
	default:
		return typ.String()
	}
}
