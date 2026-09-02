package filehelper

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeJoinPath(t *testing.T) {
	root := "uploads"

	t.Run("正常相对路径", func(t *testing.T) {
		p, err := SafeJoinPath(root, "tmp/abc/chunk_0")
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(root, "tmp/abc/chunk_0"), p)
	})

	t.Run("带前导./的相对路径", func(t *testing.T) {
		p, err := SafeJoinPath(root, "./tmp/a.txt")
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(root, "tmp/a.txt"), p)
	})

	t.Run("清理中间的.与冗余分隔符", func(t *testing.T) {
		p, err := SafeJoinPath(root, "a//b/./c.txt")
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(root, "a/b/c.txt"), p)
	})

	unsafeCases := []struct {
		name string
		rel  string
	}{
		{"直接上级逃逸", "../escape.txt"},
		{"深层逃逸", "a/../../b.txt"},
		{"仅两点", ".."},
		{"清洗后回到根", "a/.."},
		{"unix绝对路径", "/etc/passwd"},
		{"Windows反斜杠逃逸", "..\\..\\x.txt"},
		{"Windows反斜杠绝对路径", "\\Windows\\system32\\x"},
		{"Windows盘符", "C:\\Windows\\x.txt"},
		{"Windows盘符相对写法", "C:x.txt"},
		{"空串", ""},
	}
	for _, tc := range unsafeCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SafeJoinPath(root, tc.rel)
			assert.ErrorIs(t, err, ErrUnsafePath)
		})
	}
}
