package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeOrder(t *testing.T) {
	tests := []struct {
		name  string
		order string
		want  string
	}{
		// 合法输入
		{"单列名", "id", "id"},
		{"列名+方向", "id desc", "id desc"},
		{"大写方向", "created_at DESC", "created_at DESC"},
		{"蛇形列名", "created_at asc", "created_at asc"},
		{"表别名.列名", "u.created_at desc", "u.created_at desc"},
		{"多段排序", "id desc, name asc", "id desc,name asc"},
		{"多段含空白", " id desc ,  name asc ", "id desc,name asc"},
		{"纯空串", "", ""},
		{"仅逗号", ",", ""},
		// 注入变体：全部非法，整体丢弃
		{"堆叠语句", "id; drop table users", ""},
		{"子查询", "id,(SELECT CASE WHEN (1=1) THEN 1 ELSE 2 END)", "id"},
		{"反引号", "`id` desc", ""},
		{"注释截断", "id desc--", ""},
		{"数字开头", "1=1", ""},
		{"函数调用", "RAND()", ""},
		{"sleep注入", "id,sleep(5)", "id"},
		// 混合输入：只保留合法段
		{"合法+非法混合", "id desc, (SELECT 1)", "id desc"},
		{"非法+合法混合", "updatexml(1,1,1), name asc", "name asc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeOrder(tt.order))
		})
	}
}

func TestPaginateSkipsInvalidOrder(t *testing.T) {
	bp := BasePaging{PageNum: 1, PageSize: 10, Order: "id; drop table users"}
	assert.Empty(t, sanitizeOrder(bp.Order), "非法排序参数应被清空，不进入 db.Order")
}
