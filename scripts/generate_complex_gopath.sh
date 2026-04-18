#!/bin/bash

# 生成超复杂的GOPATH测试项目
# 包含：10层嵌套、200+结构体、各种测试场景

GOPATH_DIR="testdata/gopath_complex_project"
SRC_DIR="$GOPATH_DIR/src/complexproject"

# 清理旧目录
rm -rf "$GOPATH_DIR"

echo "生成复杂GOPATH测试项目..."

# 创建目录结构
mkdir -p "$SRC_DIR"/{api,models,services,utils,config,types,middleware,handlers,validators,transformers}
mkdir -p "$SRC_DIR"/vendor/{github.com/external/{lib1,lib2,lib3},google.golang.org/{grpc,protobuf}}
mkdir -p "$GOPATH_DIR/pkg"

# 生成函数
generate_struct() {
    local file=$1
    local struct_name=$2
    local fields=$3
    local pkg=$4
    
    cat >> "$file" << EOF
// $struct_name 结构体描述
type $struct_name struct {
$fields
}
EOF
}

# ==========================================
# 1. 生成类型定义文件（包含重定义类型）
# ==========================================
echo "生成类型定义..."

cat > "$SRC_DIR/types/base_types.go" << 'EOF'
package types

// 1字节重定义类型
type ByteFlag uint8
type BoolFlag bool
type StatusByte uint8

// 2字节重定义类型
type WordSize uint16
type PortNumber uint16
type FlagWord int16

// 4字节重定义类型
type DWordSize uint32
type TimeoutMs uint32
type Counter32 int32
type Float32Value float32

// 8字节重定义类型
type QWordSize uint64
type Timestamp int64
type Counter64 int64
type IDType uint64
type Float64Value float64

// 16字节重定义类型
type NameString string
type URLString string
type KeyString string
type HashString string

// 复杂重定义类型
type ByteSlice []byte
type StringSlice []string
type Int64Slice []int64
type StringMap map[string]string
type InterfaceMap map[string]interface{}
EOF

# ==========================================
# 2. 生成vendor第三方库
# ==========================================
echo "生成vendor第三方库..."

for lib in lib1 lib2 lib3; do
    cat > "$SRC_DIR/vendor/github.com/external/$lib/types.go" << EOF
package $lib

// External${lib}Struct 第三方库结构体
type External${lib}Struct struct {
	ID       int64
	Name     string
	Data     []byte
	Enabled  bool
	Version  uint32
	Config   map[string]string
}

// External${lib}Config 第三方库配置
type External${lib}Config struct {
	Host    string
	Port    int
	Timeout int64
	Retry   int32
}

// External${lib}Client 第三方库客户端
type External${lib}Client struct {
	Connection interface{}
	Timeout    int64
	Enabled    bool
	Retries    int32
	MaxConns   int64
}
EOF
done

# grpc vendor
cat > "$SRC_DIR/vendor/google.golang.org/grpc/types.go" << 'EOF'
package grpc

// GRPCConnection gRPC连接
type GRPCConnection struct {
	Target   string
	Timeout  int64
	Enabled  bool
	Retries  int32
}

// GRPCConfig gRPC配置
type GRPCConfig struct {
	Host          string
	Port          int
	MaxMsgSize    int64
	KeepAliveTime int64
	Enabled       bool
}
EOF

# protobuf vendor
cat > "$SRC_DIR/vendor/google.golang.org/protobuf/types.go" << 'EOF'
package protobuf

// ProtoMessage 协议消息
type ProtoMessage struct {
	Data     []byte
	Size     int64
	Version  int32
	Enabled  bool
}

// ProtoField 协议字段
type ProtoField struct {
	Name     string
	Type     int32
	Number   int32
	Repeated bool
	Optional bool
}
EOF

# ==========================================
# 3. 生成10层嵌套结构体
# ==========================================
echo "生成10层嵌套结构体..."

# Level 10 - 最底层
cat > "$SRC_DIR/models/level10.go" << 'EOF'
package models

// Level10_DeepestStruct 第10层-最深层
type Level10_DeepestStruct struct {
	ID        int64
	Name      string
	Value     float64
	Enabled   bool
	Timestamp int64
}

// Level10_AnotherStruct 第10层-另一个结构体
type Level10_AnotherStruct struct {
	Code    uint32
	Message string
	Data    []byte
	Count   int64
}

// Level10_ConfigStruct 第10层-配置结构体
type Level10_ConfigStruct struct {
	Host        string
	Port        int
	Timeout     int64
	MaxRetries  int32
	Enabled     bool
	Description string
}
EOF

# Level 9
cat > "$SRC_DIR/models/level9.go" << 'EOF'
package models

// Level9_ParentOf10 第9层-包含Level10
type Level9_ParentOf10 struct {
	Name     string
	Level10  Level10_DeepestStruct
	Count    int64
	Enabled  bool
}

// Level9_SiblingStruct 第9层-兄弟结构体
type Level9_SiblingStruct struct {
	ID       int64
	Code     string
	Data     []byte
	Another  Level10_AnotherStruct
	Active   bool
}

// Level9_ConfigParent 第9层-配置父级
type Level9_ConfigParent struct {
	Name      string
	Config    Level10_ConfigStruct
	Timeout   int64
	Retries   int32
}
EOF

# Level 8
cat > "$SRC_DIR/models/level8.go" << 'EOF'
package models

// Level8_ParentOf9 第8层-包含Level9
type Level8_ParentOf9 struct {
	Title    string
	Child1   Level9_ParentOf10
	Child2   Level9_SiblingStruct
	Count    int64
	Active   bool
	TTL      int64
}

// Level8_ConfigLevel 第8层-配置层
type Level8_ConfigLevel struct {
	Name     string
	Config   Level9_ConfigParent
	Enabled  bool
	Timeout  int64
}

// Level8_DataStruct 第8层-数据结构体
type Level8_DataStruct struct {
	ID       int64
	Data     []byte
	Metadata Level10_AnotherStruct
	Size     int64
	Version  uint32
}
EOF

# Level 7
cat > "$SRC_DIR/models/level7.go" << 'EOF'
package models

// Level7_ParentOf8 第7层
type Level7_ParentOf8 struct {
	Name     string
	Child    Level8_ParentOf9
	Config   Level8_ConfigLevel
	Count    int64
	Enabled  bool
}

// Level7_DataParent 第7层-数据父级
type Level7_DataParent struct {
	ID       int64
	Data     Level8_DataStruct
	Metadata map[string]string
	Active   bool
}

// Level7_ServiceConfig 第7层-服务配置
type Level7_ServiceConfig struct {
	ServiceName string
	Config      Level8_ConfigLevel
	Timeout     int64
	Retries     int32
	Enabled     bool
}
EOF

# Level 6
cat > "$SRC_DIR/models/level6.go" << 'EOF'
package models

// Level6_ParentOf7 第6层
type Level6_ParentOf7 struct {
	Name     string
	Child    Level7_ParentOf8
	Data     Level7_DataParent
	Config   Level7_ServiceConfig
	Enabled  bool
}

// Level6_APIConfig 第6层-API配置
type Level6_APIConfig struct {
	BaseURL     string
	Timeout     int64
	Retries     int32
	RateLimit   int64
	Enabled     bool
}

// Level6_CacheConfig 第6层-缓存配置
type Level6_CacheConfig struct {
	MaxSize   int64
	TTL       int64
	Enabled   bool
	Eviction  string
}
EOF

# Level 5
cat > "$SRC_DIR/models/level5.go" << 'EOF'
package models

// Level5_ParentOf6 第5层
type Level5_ParentOf6 struct {
	Name     string
	Child    Level6_ParentOf7
	API      Level6_APIConfig
	Cache    Level6_CacheConfig
	Count    int64
}

// Level5_ModuleConfig 第5层-模块配置
type Level5_ModuleConfig struct {
	ModuleName string
	Enabled    bool
	Config     Level6_APIConfig
	Timeout    int64
}

// Level5_DataSource 第5层-数据源
type Level5_DataSource struct {
	URL      string
	PoolSize int
	Timeout  int64
	Enabled  bool
	Retries  int32
}
EOF

# Level 4
cat > "$SRC_DIR/models/level4.go" << 'EOF'
package models

// Level4_ParentOf5 第4层
type Level4_ParentOf5 struct {
	Name     string
	Child    Level5_ParentOf6
	Module   Level5_ModuleConfig
	DataSource Level5_DataSource
	Active   bool
}

// Level4_PluginConfig 第4层-插件配置
type Level4_PluginConfig struct {
	PluginName string
	Version    string
	Enabled    bool
	Config     Level5_ModuleConfig
}

// Level4_MiddlewareConfig 第4层-中间件配置
type Level4_MiddlewareConfig struct {
	Name     string
	Order    int32
	Enabled  bool
	Timeout  int64
}
EOF

# Level 3
cat > "$SRC_DIR/models/level3.go" << 'EOF'
package models

// Level3_ParentOf4 第3层
type Level3_ParentOf4 struct {
	Name     string
	Child    Level4_ParentOf5
	Plugin   Level4_PluginConfig
	Middleware Level4_MiddlewareConfig
	Count    int64
}

// Level3_RouterConfig 第3层-路由配置
type Level3_RouterConfig struct {
	Pattern  string
	Handler  string
	Enabled  bool
	Timeout  int64
	Middleware []string
}

// Level3_ValidatorConfig 第3层-验证器配置
type Level3_ValidatorConfig struct {
	Strict   bool
	Enabled  bool
	Rules    map[string]string
	Timeout  int64
}
EOF

# Level 2
cat > "$SRC_DIR/models/level2.go" << 'EOF'
package models

// Level2_ParentOf3 第2层
type Level2_ParentOf3 struct {
	Name     string
	Child    Level3_ParentOf4
	Router   Level3_RouterConfig
	Validator Level3_ValidatorConfig
	Enabled  bool
}

// Level2_ServerConfig 第2层-服务器配置
type Level2_ServerConfig struct {
	Host       string
	Port       int
	Enabled    bool
	Timeout    int64
	MaxConns   int64
	TLS        bool
}

// Level2_LoggerConfig 第2层-日志配置
type Level2_LoggerConfig struct {
	Level      string
	Output     string
	Enabled    bool
	MaxSize    int64
	Compress   bool
}
EOF

# Level 1
cat > "$SRC_DIR/models/level1.go" << 'EOF'
package models

// Level1_ParentOf2 第1层
type Level1_ParentOf2 struct {
	Name     string
	Child    Level2_ParentOf3
	Server   Level2_ServerConfig
	Logger   Level2_LoggerConfig
	Enabled  bool
}
EOF

# Level 0 - 顶层Root
cat > "$SRC_DIR/models/level0.go" << 'EOF'
package models

// Level0_RootStruct 第0层-顶层根结构体
type Level0_RootStruct struct {
	Name     string
	Child    Level1_ParentOf2
	Version  string
	Enabled  bool
	CreatedAt int64
}
EOF

echo "✓ 10层嵌套结构体生成完成"

# ==========================================
# 4. 生成同包不同文件的未导出结构体
# ==========================================
echo "生成未导出结构体..."

# 未导出配置结构体
cat > "$SRC_DIR/models/internal_config.go" << 'EOF'
package models

// internalConfig 未导出的内部配置（同包不同文件）
type internalConfig struct {
	Host        string
	Port        int
	Timeout     int64
	Enabled     bool
	Retries     int32
	MaxConns    int64
	Description string
	Debug       bool
	LogLevel    string
}

// internalCache 未导出的缓存配置（同包不同文件）
type internalCache struct {
	MaxSize   int64
	TTL       int64
	Enabled   bool
	Eviction  string
	HitCount  int64
	MissCount int64
}

// internalPool 未导出的连接池配置（同包不同文件）
type internalPool struct {
	MinConns  int32
	MaxConns  int64
	Timeout   int64
	Enabled   bool
	IdleTime  int64
}
EOF

# 未导出匿名字段类型
cat > "$SRC_DIR/models/internal_base.go" << 'EOF'
package models

// internalBase 未导出的基础匿名字段
type internalBase struct {
	ID        int64
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
	Version   int32
	Active    bool
}

// internalAudit 未导出的审计匿名字段
type internalAudit struct {
	CreatedBy  string
	UpdatedBy  string
	CreatedAt  int64
	UpdatedAt  int64
	IPAddress  string
	UserAgent  string
}
EOF

echo "✓ 未导出结构体生成完成"

# ==========================================
# 5. 生成包含方法的结构体（用于skip-by-methods测试）
# ==========================================
echo "生成带方法的结构体..."

cat > "$SRC_DIR/api/handlers.go" << 'EOF'
package api

// HandlerWithEncode 具有Encode方法的处理器（应被skip-by-methods跳过）
type HandlerWithEncode struct {
	Name    string
	Timeout int64
	Enabled bool
	Retries int32
}

// Encode 编码方法
func (h *HandlerWithEncode) Encode() []byte {
	return []byte(h.Name)
}

// Decode 解码方法
func (h *HandlerWithEncode) Decode(data []byte) error {
	h.Name = string(data)
	return nil
}

// HandlerWithMarshal 具有Marshal方法的处理器
type HandlerWithMarshal struct {
	Name    string
	Path    string
	Timeout int64
	Enabled bool
}

// MarshalJSON JSON序列化
func (h *HandlerWithMarshal) MarshalJSON() ([]byte, error) {
	return nil, nil
}

// UnmarshalJSON JSON反序列化
func (h *HandlerWithMarshal) UnmarshalJSON(data []byte) error {
	return nil
}

// HandlerNoMethods 没有方法的处理器（应该被优化）
type HandlerNoMethods struct {
	Name    string
	Path    string
	Timeout int64
	Enabled bool
	Retries int32
	Priority int32
}

// HandlerWithValidate 具有Validate方法的处理器
type HandlerWithValidate struct {
	Name     string
	Strict   bool
	Enabled  bool
	Timeout  int64
}

// Validate 验证方法
func (h *HandlerWithValidate) Validate() error {
	return nil
}
EOF

echo "✓ 带方法的结构体生成完成"

# ==========================================
# 6. 生成大量结构体（达到200+）
# ==========================================
echo "生成大量结构体..."

# 生成50个API模型结构体
cat > "$SRC_DIR/models/batch_models.go" << 'EOF'
package models

EOF

for i in $(seq 1 50); do
    cat >> "$SRC_DIR/models/batch_models.go" << EOF
// BatchModel${i} 批量模型${i}
type BatchModel${i} struct {
	ID       int64
	Name     string
	Enabled  bool
	Data     []byte
	Count    int64
	Version  uint32
}

EOF
done

# 生成50个Service结构体
cat > "$SRC_DIR/services/batch_services.go" << 'EOF'
package services

EOF

for i in $(seq 1 50); do
    cat >> "$SRC_DIR/services/batch_services.go" << EOF
// BatchService${i} 批量服务${i}
type BatchService${i} struct {
	Name     string
	Timeout  int64
	Enabled  bool
	Retries  int32
	PoolSize int
	MaxConns int64
}

EOF
done

# 生成30个Config结构体
cat > "$SRC_DIR/config/batch_configs.go" << 'EOF'
package config

EOF

for i in $(seq 1 30); do
    cat >> "$SRC_DIR/config/batch_configs.go" << EOF
// BatchConfig${i} 批量配置${i}
type BatchConfig${i} struct {
	Key       string
	Value     string
	Enabled   bool
	Timeout   int64
	Priority  int32
}

EOF
done

# 生成30个Middleware结构体
cat > "$SRC_DIR/middleware/batch_middlewares.go" << 'EOF'
package middleware

EOF

for i in $(seq 1 30); do
    cat >> "$SRC_DIR/middleware/batch_middlewares.go" << EOF
// BatchMiddleware${i} 批量中间件${i}
type BatchMiddleware${i} struct {
	Name     string
	Order    int32
	Enabled  bool
	Timeout  int64
	Priority int32
}

EOF
done

# 生成20个Validator结构体
cat > "$SRC_DIR/validators/batch_validators.go" << 'EOF'
package validators

EOF

for i in $(seq 1 20); do
    cat >> "$SRC_DIR/validators/batch_validators.go" << EOF
// BatchValidator${i} 批量验证器${i}
type BatchValidator${i} struct {
	Field    string
	Rule     string
	Strict   bool
	Enabled  bool
	Timeout  int64
}

EOF
done

# 生成20个Transformer结构体
cat > "$SRC_DIR/transformers/batch_transformers.go" << 'EOF'
package transformers

EOF

for i in $(seq 1 20); do
    cat >> "$SRC_DIR/transformers/batch_transformers.go" << EOF
// BatchTransformer${i} 批量转换器${i}
type BatchTransformer${i} struct {
	SourceType   string
	TargetType   string
	Enabled      bool
	Timeout      int64
	MaxRetries   int32
}

EOF
done

echo "✓ 大量结构体生成完成"

# ==========================================
# 7. 生成使用重定义类型的复杂结构体
# ==========================================
echo "生成使用重定义类型的结构体..."

cat > "$SRC_DIR/models/typedef_structs.go" << 'EOF'
package models

import "complexproject/types"

// ComplexTypeStruct 包含多种重定义类型的复杂结构体
type ComplexTypeStruct struct {
	ID         types.IDType
	Status     types.ByteFlag
	Port       types.PortNumber
	Timeout    types.TimeoutMs
	Timestamp  types.Timestamp
	Count      types.Counter64
	Name       types.NameString
	URL        types.URLString
	Data       types.ByteSlice
	Tags       types.StringSlice
	Metadata   types.StringMap
	Enabled    types.BoolFlag
}

// AnotherTypeStruct 另一个包含重定义类型的结构体
type AnotherTypeStruct struct {
	Code       types.WordSize
	Flag       types.FlagWord
	Value      types.Float32Value
	Price      types.Float64Value
	Hash       types.HashString
	Key        types.KeyString
	IDs        types.Int64Slice
	Config     types.InterfaceMap
}
EOF

echo "✓ 重定义类型结构体生成完成"

# ==========================================
# 8. 生成包含匿名字段的复杂结构体
# ==========================================
echo "生成匿名字段结构体..."

cat > "$SRC_DIR/models/embedded_structs.go" << 'EOF'
package models

// UserWithEmbedded 包含匿名字段的用户结构体
type UserWithEmbedded struct {
	internalBase    // 未导出的匿名字段
	internalAudit   // 未导出的审计匿名字段
	Name       string
	Email      string
	Age        int32
	Enabled    bool
}

// ProductWithEmbedded 包含匿名字段的产品结构体
type ProductWithEmbedded struct {
	internalBase    // 未导出的基础字段
	internalConfig  // 未导出的配置字段
	SKU        string
	Price      float64
	Stock      int64
	Enabled    bool
}

// OrderWithEmbedded 包含匿名字段的订单结构体
type OrderWithEmbedded struct {
	internalBase
	internalAudit
	OrderNumber string
	TotalAmount float64
	Status      string
	Paid        bool
}
EOF

echo "✓ 匿名字段结构体生成完成"

# ==========================================
# 9. 生成顶层MainStruct（引用各种场景）
# ==========================================
echo "生成顶层MainStruct..."

cat > "$SRC_DIR/api/main_struct.go" << 'EOF'
package api

import (
	"complexproject/models"
	"complexproject/services"
	"complexproject/config"
	"complexproject/middleware"
	"complexproject/validators"
	"complexproject/transformers"
	"complexproject/types"
	lib1 "github.com/external/lib1"
	lib2 "github.com/external/lib2"
	"google.golang.org/grpc"
)

// MainComplexStruct 顶层复杂结构体 - 包含所有测试场景
type MainComplexStruct struct {
	// 基本信息
	Name        string
	Version     string
	Enabled     bool
	Timestamp   int64
	
	// 10层嵌套
	Level0      models.Level0_RootStruct
	
	// 同包未导出类型（通过其他方式引用）
	Config      *config.AppConfig
	
	// 重定义类型
	Status      types.ByteFlag
	Timeout     types.TimeoutMs
	MaxConns    types.Counter64
	Name        types.NameString
	
	// 匿名字段
	models.internalBase  // 同包未导出匿名字段
	
	// vendor第三方库（应被跳过）
	ExternalLib1  lib1.ExternalLib1Struct
	ExternalLib2  lib2.ExternalLib2Struct
	GRPCConn      grpc.GRPCConnection
	
	// 服务引用
	Services    []services.ServiceInterface
	Handlers    []*HandlerWithEncode  // 带方法的（应被skip-by-methods跳过）
	Middlewares []middleware.MiddlewareConfig
	
	// 验证器和转换器
	Validators  []*validators.ValidatorConfig
	Transformers []transformers.TransformerConfig
	
	// 批量结构体引用
	BatchModels     [10]models.BatchModel1
	BatchServices   [5]services.BatchService1
	
	// 复杂类型
	Data        types.ByteSlice
	Metadata    types.StringMap
	Tags        types.StringSlice
}
EOF

echo "✓ 顶层MainStruct生成完成"

# ==========================================
# 10. 生成必要的辅助文件
# ==========================================
echo "生成辅助文件..."

# AppConfig
cat > "$SRC_DIR/config/app_config.go" << 'EOF'
package config

// AppConfig 应用配置
type AppConfig struct {
	Name        string
	Version     string
	Enabled     bool
	Timeout     int64
	MaxConns    int64
	Debug       bool
	LogLevel    string
}
EOF

# MiddlewareConfig
cat > "$SRC_DIR/middleware/config.go" << 'EOF'
package middleware

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	Name     string
	Order    int32
	Enabled  bool
	Timeout  int64
}
EOF

# ValidatorConfig
cat > "$SRC_DIR/validators/config.go" << 'EOF'
package validators

// ValidatorConfig 验证器配置
type ValidatorConfig struct {
	Field    string
	Rule     string
	Enabled  bool
}
EOF

# TransformerConfig
cat > "$SRC_DIR/transformers/config.go" << 'EOF'
package transformers

// TransformerConfig 转换器配置
type TransformerConfig struct {
	SourceType string
	TargetType string
	Enabled    bool
}
EOF

# ServiceInterface
cat > "$SRC_DIR/services/interface.go" << 'EOF'
package services

// ServiceInterface 服务接口
type ServiceInterface interface {
	Start() error
	Stop() error
}

// BaseService 基础服务
type BaseService struct {
	Name    string
	Enabled bool
	Timeout int64
}
EOF

echo "✓ 辅助文件生成完成"

# ==========================================
# 统计生成的结构体数量
# ==========================================
echo ""
echo "====================================="
echo "统计结构体数量..."
echo "====================================="

count_structs() {
    local dir=$1
    local count=0
    for file in $(find "$dir" -name "*.go" -not -path "*/vendor/*"); do
        n=$(grep -c "^type.*struct {" "$file" 2>/dev/null || echo 0)
        count=$((count + n))
    done
    echo $count
}

# 统计用户代码中的结构体
user_structs=$(count_structs "$SRC_DIR")
echo "用户代码结构体数量: $user_structs"

# 统计vendor中的结构体
vendor_count=0
for file in $(find "$SRC_DIR/vendor" -name "*.go" 2>/dev/null); do
    n=$(grep -c "^type.*struct {" "$file" 2>/dev/null || echo 0)
    vendor_count=$((vendor_count + n))
done
echo "Vendor第三方结构体数量: $vendor_count"

echo "总结构体数量: $((user_structs + vendor_count))"
echo "====================================="
echo "✓ GOPATH复杂测试项目生成完成！"
echo "====================================="
