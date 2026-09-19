package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func MiddleLog(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
        path := c.Request.URL.Path
        raw := c.Request.URL.RawQuery

        // 执行后续的 handler
        c.Next()

        // handler 执行完后，记录日志
        latency := time.Since(start)
        statusCode := c.Writer.Status()
        clientIP := c.ClientIP()
        method := c.Request.Method

        if raw != "" {
            path = path + "?" + raw
        }

        logger.Info().
            Str("client_ip", clientIP).
            Str("method", method).
            Str("path", path).
            Int("status", statusCode).
            Dur("latency", latency).
            Msg("http_request")
	}
}