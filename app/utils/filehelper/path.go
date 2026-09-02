package filehelper

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrUnsafePath 相对路径试图逃逸目标根目录或为绝对路径
var ErrUnsafePath = errors.New("不安全的文件路径：禁止逃逸目标目录或使用绝对路径")

// SafeJoinPath 将 rel 安全拼接到 root 下，用于解压等场景。
// rel 必须是相对路径，且清洗后不能以 ".." 逃逸 root（同时兼容 Windows 反斜杠写法）。
func SafeJoinPath(root, rel string) (string, error) {
	if rel == "" {
		return "", ErrUnsafePath
	}

	// 统一按分隔符处理，避免 Windows 下 "\" 绕过检查
	normalized := strings.ReplaceAll(rel, "\\", "/")

	// 绝对路径（unix 风格）与 Windows 盘符一律拒绝
	if strings.HasPrefix(normalized, "/") || filepath.IsAbs(rel) {
		return "", ErrUnsafePath
	}
	if len(normalized) >= 2 && normalized[1] == ':' {
		return "", ErrUnsafePath
	}

	cleaned := filepath.Clean(normalized)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", ErrUnsafePath
	}
	if cleaned == "." {
		return "", ErrUnsafePath
	}

	return filepath.Join(root, filepath.FromSlash(cleaned)), nil
}
