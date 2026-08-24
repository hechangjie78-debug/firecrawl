package middleware

import (
	"net/http"
	"time"

	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
)

// RateLimiterMiddleware 生产级基于 Redis + Lua 脚本的滑动窗口限流中间件
func RateLimiterMiddleware(redisService *services.RedisService, requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 取客户端 IP 或 Token 作为限流维度
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "global"
		}

		allowed, err := redisService.AllowRateLimit(c.Request.Context(), clientIP, requestsPerMinute, 1*time.Minute)
		if err != nil {
			// 限流器异常时降级日志放行
			c.Next()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Rate limit exceeded. Too many requests in 1 minute.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
