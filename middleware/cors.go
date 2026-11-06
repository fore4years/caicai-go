package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件，对齐 Java CorsInterceptor（SC-006）。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Headers", "*")

		// OPTIONS 预检直接返回 204，不进入后续鉴权。
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
