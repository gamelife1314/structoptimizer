package optimizer

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PackageCache 包缓存管理器
type PackageCache struct {
	mu       sync.Mutex
	cacheDir string
	enabled  bool
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Hash        string            `json:"hash"`        // 包内容哈希
	PkgPath     string            `json:"pkg_path"`    // 包路径
	Structs     []StructCacheInfo `json:"structs"`     // 结构体信息
	GoModHash   string            `json:"go_mod_hash"` // go.mod 哈希
	GoFiles     []string          `json:"go_files"`    // Go 文件列表
	CreatedAt   int64             `json:"created_at"`  // 创建时间戳
}

// StructCacheInfo 结构体缓存信息
type StructCacheInfo struct {
	Name      string           `json:"name"`
	FilePath  string           `json:"file_path"`
	Fields    []FieldCacheInfo `json:"fields"`
	OrigSize  int64            `json:"orig_size"`
	OptSize   int64            `json:"opt_size"`
}

// FieldCacheInfo 字段缓存信息
type FieldCacheInfo struct {
	Name     string `json:"name"`
	TypeName string `json:"type_name"`
	Size     int64  `json:"size"`
	Align    int64  `json:"align"`
	IsEmbed  bool   `json:"is_embed"`
	Tag      string `json:"tag"`
}

// NewPackageCache 创建包缓存管理器
func NewPackageCache(cacheDir string, enabled bool) *PackageCache {
	return &PackageCache{
		cacheDir: cacheDir,
		enabled:  enabled,
	}
}

// GetCacheDir 获取缓存目录
func (pc *PackageCache) GetCacheDir() string {
	if pc.cacheDir == "" {
		// 默认缓存目录：用户缓存目录下的 structoptimizer 子目录
		cacheBase, err := os.UserCacheDir()
		if err != nil {
			cacheBase = os.TempDir()
		}
		pc.cacheDir = filepath.Join(cacheBase, "structoptimizer")
	}
	return pc.cacheDir
}

// LoadPackageCache 从文件加载包缓存
func (pc *PackageCache) LoadPackageCache(pkgPath string, goFiles []string) (*CacheEntry, error) {
	if !pc.enabled {
		return nil, nil
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 计算包内容哈希
	contentHash, err := pc.computePackageHash(goFiles)
	if err != nil {
		return nil, fmt.Errorf("计算包哈希失败：%v", err)
	}

	// 计算 go.mod 哈希
	goModHash := pc.computeGoModHash()

	// 构建缓存文件路径
	cacheFile := pc.buildCacheFilePath(pkgPath)

	// 检查缓存文件是否存在
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil, nil // 缓存不存在
	}

	// 读取缓存文件
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("读取缓存文件失败：%v", err)
	}

	// 解析缓存
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("解析缓存失败：%v", err)
	}

	// 验证哈希是否匹配
	if entry.Hash != contentHash || entry.GoModHash != goModHash {
		return nil, nil // 缓存已过期
	}

	return &entry, nil
}

// SavePackageCache 保存包缓存到文件
func (pc *PackageCache) SavePackageCache(entry *CacheEntry) error {
	if !pc.enabled {
		return nil
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 创建缓存目录
	cacheDir := pc.GetCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败：%v", err)
	}

	// 序列化缓存
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存失败：%v", err)
	}

	// 写入缓存文件
	cacheFile := pc.buildCacheFilePath(entry.PkgPath)
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("写入缓存文件失败：%v", err)
	}

	return nil
}

// computePackageHash 计算包内容哈希
func (pc *PackageCache) computePackageHash(goFiles []string) (string, error) {
	h := sha256.New()

	// 按顺序处理每个文件
	for _, file := range goFiles {
		// 添加文件路径
		h.Write([]byte(file))
		h.Write([]byte{0})

		// 添加文件内容
		f, err := os.Open(file)
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()

		h.Write([]byte{0})
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// computeGoModHash 计算 go.mod 哈希
func (pc *PackageCache) computeGoModHash() string {
	// 尝试在多个位置查找 go.mod
	possiblePaths := []string{
		"go.mod",
		filepath.Join(pc.cacheDir, "..", "..", "go.mod"),
	}

	for _, path := range possiblePaths {
		if data, err := os.ReadFile(path); err == nil {
			h := sha256.Sum256(data)
			return fmt.Sprintf("%x", h)
		}
	}

	return ""
}

// buildCacheFilePath 构建缓存文件路径
func (pc *PackageCache) buildCacheFilePath(pkgPath string) string {
	// 将包路径转换为安全的文件名
	// code.XXX.com/XX1/XX/dd/sss -> code.XXX.com_XX1_XX_dd_sss.hash
	safeName := strings.ReplaceAll(pkgPath, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	
	cacheDir := pc.GetCacheDir()
	return filepath.Join(cacheDir, safeName+".cache.json")
}

// GobEncoder 用于 gob 编码（如果需要）
type GobEncoder struct{}

// Encode 编码缓存条目
func (e *GobEncoder) Encode(entry *CacheEntry) ([]byte, error) {
	var buf strings.Builder
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(entry); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// Decode 解码缓存条目
func (e *GobEncoder) Decode(data []byte) (*CacheEntry, error) {
	buf := strings.NewReader(string(data))
	dec := gob.NewDecoder(buf)
	var entry CacheEntry
	if err := dec.Decode(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}
