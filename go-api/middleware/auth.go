package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/firecrawl/go-api/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const (
	// AuthContextKey 是在 Gin.Context 中存放团队鉴权信息的 Key
	AuthContextKey = "team_auth"
)

// AuthMiddleware 校验 Authorization Header 中的 Bearer API Key 令牌
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取环境变量中的配置
		useAuth := os.Getenv("USE_DB_AUTHENTICATION") == "true"
		testAPIKey := os.Getenv("TEST_API_KEY")

		// 2. 从 HTTP Header 中获取 Authorization
		authHeader := c.GetHeader("Authorization")

		// 如果关闭了身份验证，或在本地开发环境，默认赋予测试团队身份
		if !useAuth && authHeader == "" {
			c.Set(AuthContextKey, &models.TeamAuth{
				TeamID:   "default-team-id",
				APIKeyID: "dev-key",
				Plan:     "free",
			})
			c.Next()
			return
		}

		// 3. 校验 Authorization 格式 (Authorization: Bearer <token>)
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn().Str("path", c.Request.URL.Path).Msg("缺少或非法的 Authorization 请求头")
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ScrapeResponse{
				Success: false,
				Error:   "Unauthorized: Missing or invalid Authorization header",
				Code:    "UNAUTHORIZED",
			})
			return
		}

		apiKey := strings.TrimPrefix(authHeader, "Bearer ")

		// 4. 简化的 Key 校验逻辑（如果设置了 TEST_API_KEY，匹配则通过）
		if testAPIKey != "" && apiKey != testAPIKey {
			log.Warn().Str("apiKey", apiKey).Msg("API Key 不匹配")
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ScrapeResponse{
				Success: false,
				Error:   "Unauthorized: Invalid API Key",
				Code:    "UNAUTHORIZED",
			})
			return
		}

		// 5. 鉴权通过，构建团队上下文信息并继续传递
		authInfo := &models.TeamAuth{
			TeamID:   "team_" + apiKey[:min(len(apiKey), 8)], // 提取 key 的前缀作为 team_id 标识
			APIKeyID: apiKey,
			Plan:     "pro",
		}
		c.Set(AuthContextKey, authInfo)

		c.Next()
	}
}

// GetTeamAuth 辅助函数：从 Gin 上下文中安全提取当前请求对应的团队鉴权对象
func GetTeamAuth(c *gin.Context) (*models.TeamAuth, bool) {
	val, exists := c.Get(AuthContextKey)
	if !exists {
		return nil, false
	}
	auth, ok := val.(*models.TeamAuth)
	return auth, ok
}

// 辅助函数：求小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
