package cachehelper

import (
	"strings"

	"gin-fast/app/global/app"
)

// maskedValue 敏感缓存项的脱敏占位值
const maskedValue = "<masked>"

// MaskTokenValues 返回脱敏后的缓存项副本：key 含 "token" 的项（如在线用户的完整 JWT）
// 值替换为 <masked>，其余原样保留，避免调试端点泄露可用凭证。
func MaskTokenValues(items []app.CacheItem) []app.CacheItem {
	masked := make([]app.CacheItem, len(items))
	for i, item := range items {
		if strings.Contains(strings.ToLower(item.Key), "token") {
			item.Value = maskedValue
		}
		masked[i] = item
	}
	return masked
}
