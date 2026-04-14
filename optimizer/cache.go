package optimizer

import (
	"go/types"
)

// findStructWithCache 带缓存的结构体查找
func (o *Optimizer) findStructWithCache(pkgPath, structName string) (*types.Struct, string, error) {
	key := pkgPath + "." + structName

	// 检查结构体缓存
	o.mu.Lock()
	if st, ok := o.structCache[key]; ok {
		filePath := o.filePathCache[key]
		o.mu.Unlock()
		return st, filePath, nil
	}
	o.mu.Unlock()

	// 查找结构体
	st, filePath, err := o.analyzer.FindStructByName(pkgPath, structName)
	if err != nil {
		return nil, "", err
	}

	// 缓存结果
	o.mu.Lock()
	o.structCache[key] = st
	o.filePathCache[key] = filePath
	o.mu.Unlock()

	return st, filePath, nil
}
