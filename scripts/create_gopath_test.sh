#!/bin/bash
# create_gopath_test.sh - 创建 GOPATH 测试项目

set -e

# 检查参数
GOPATH_ROOT="${1:-./gopath_test}"
PROJECT_PKG="${2:-mycompany/myproject/typealias}"

echo "创建 GOPATH 测试项目..."
echo "GOPATH: $GOPATH_ROOT"
echo "Package: $PROJECT_PKG"

# 创建目录结构
PKG_DIR="$GOPATH_ROOT/src/$PROJECT_PKG"
mkdir -p "$PKG_DIR"

# 创建测试文件 - 包含重定义类型的结构体
cat > "$PKG_DIR/typealias.go" << 'EOF'
package typealias

// StructWithTypeAlias 包含重定义类型的结构体
type StructWithTypeAlias struct {
	ID   int64
	flag newType
	Name string
}

// newType 重定义的类型，底层是 uint8，大小应该是 1
type newType uint8

// AnotherType 另一个重定义类型
type AnotherType uint16

// StructWithAnother 包含另一种重定义类型的结构体
type StructWithAnother struct {
	X int64
	Y AnotherType
	Z bool
}

// BadStruct 未优化的结构体，包含重定义类型
type BadStruct struct {
	A bool
	B newType
	C int64
	D AnotherType
	E int32
}

// GoodStruct 优化后的结构体
type GoodStruct struct {
	C int64
	D AnotherType
	E int32
	A bool
	B newType
}
EOF

echo "测试项目创建完成: $PKG_DIR"
echo ""
echo "目录结构:"
find "$GOPATH_ROOT" -type f -name "*.go" | head -10
