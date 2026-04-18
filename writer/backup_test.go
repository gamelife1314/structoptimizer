package writer

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBackupAndWriteErrorHandling 测试备份错误处理（Bug #3 修复）
func TestBackupAndWriteErrorHandling(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建一个没有写权限的文件（通过创建只读目录）
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Skip("无法创建只读目录，跳过测试")
	}

	readOnlyFile := filepath.Join(readOnlyDir, "test.go")
	if err := os.WriteFile(readOnlyFile, []byte("original"), 0444); err != nil {
		t.Skip("无法创建只读文件，跳过测试")
	}

	// 测试备份失败的情况
	err := BackupAndWrite(readOnlyFile, "new content", true)
	if err == nil {
		t.Error("备份失败时应该返回错误")
	}
}
