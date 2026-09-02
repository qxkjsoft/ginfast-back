package cachehelper

import (
	"testing"
	"time"

	"gin-fast/app/global/app"

	"github.com/stretchr/testify/assert"
)

func TestMaskTokenValues(t *testing.T) {
	now := time.Now()
	items := []app.CacheItem{
		{Key: "gin-fast:token:1:abc123", Value: "eyJhbGciOiJIUzI1NiJ9.full.jwt", ExpiresAt: now},
		{Key: "gin-fast:TOKEN:2:xyz", Value: "eyJanother.jwt.string", ExpiresAt: now},
		{Key: "captcha:img:uuid-1", Value: "base64data", ExpiresAt: now},
		{Key: "login:lock:admin", Value: "3", ExpiresAt: now},
	}

	masked := MaskTokenValues(items)

	assert.Equal(t, "<masked>", masked[0].Value, "token 类 key 的值必须脱敏")
	assert.Equal(t, "<masked>", masked[1].Value, "key 大小写不同也应命中脱敏")
	assert.Equal(t, "base64data", masked[2].Value, "普通 key 的值应原样保留")
	assert.Equal(t, "3", masked[3].Value, "非 token 的普通 key 不应被改动")

	// 原切片不被修改
	assert.Equal(t, "eyJhbGciOiJIUzI1NiJ9.full.jwt", items[0].Value, "脱敏应作用于副本，不修改原始数据")

	// key 与过期时间保持不变
	assert.Equal(t, "gin-fast:token:1:abc123", masked[0].Key)
	assert.Equal(t, now, masked[0].ExpiresAt)
}

func TestMaskTokenValuesEmpty(t *testing.T) {
	masked := MaskTokenValues(nil)
	assert.NotNil(t, masked)
	assert.Empty(t, masked)

	masked = MaskTokenValues([]app.CacheItem{})
	assert.Empty(t, masked)
}
