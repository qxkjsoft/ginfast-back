package middleware

import (
	"net/http"

	"gin-fast/app/global/app"

	"github.com/gin-gonic/gin"
)

// CorsNext 跨域中间件：从配置 httpserver.corsorigins 读取白名单，仅回显命中的 Origin
func CorsNext() gin.HandlerFunc {
	return corsWithOrigins(app.ConfigYml.GetStringSlice("httpserver.corsorigins"))
}

// corsWithOrigins 仅回显白名单中的 Origin；未命中不设置跨域响应头，由浏览器拦截。
// 白名单含 "*" 时放行任意来源，此时不发送 Allow-Credentials（规范禁止 * 与凭证组合）
func corsWithOrigins(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowAll := false
		matched := ""
		for _, allowed := range allowedOrigins {
			if allowed == "*" {
				allowAll = true
			} else if allowed == origin && origin != "" {
				matched = allowed
			}
		}

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if matched != "" {
			c.Header("Access-Control-Allow-Origin", matched)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Headers,Authorization,User-Agent, Keep-Alive, Content-Type, X-Requested-With,X-CSRF-Token,AccessToken,Token")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, PATCH, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")

		// 放行所有OPTIONS方法
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		// 处理请求
		c.Next()
	}
}
