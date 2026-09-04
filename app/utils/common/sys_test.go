package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeepLettersPathAndUnderscoreLower(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// 保留下划线（与 KeepLettersAndPathLower 的关键差异）
		{"下划线保留", "sys_demo", "sys_demo"},
		{"路径+下划线", "Test/Admin_Foo", "test/admin_foo"},
		// 路径规范化行为与 KeepLettersAndPathLower 一致
		{"纯路径大小写", "Test/Admin", "test/admin"},
		{"合并连续斜杠", "//a//b//", "a/b"},
		{"仅斜杠", "/", ""},
		// 非法字符剥离
		{"剥离非法字符", "sys-demo 01!", "sysdemo"},
		{"空串", "", ""},
		{"仅非法字符", "-!*@", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, KeepLettersPathAndUnderscoreLower(tt.in))
		})
	}
}
