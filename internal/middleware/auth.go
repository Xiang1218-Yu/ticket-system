package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ticket-system/internal/service"
)

// 上下文键
const (
	ContextUserID = "userID"
	ContextRole   = "role"
)

// Auth 校验 JWT 并把用户信息注入上下文。
// 单一职责：仅做 token 解析与用户加载，不含业务授权（细粒度授权在 service 内）。
func Auth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		tokenStr := normalizeTokenHeader(header)
		claims, err := auth.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已失效"})
			return
		}
		user, err := auth.FindUserByID(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			return
		}
		c.Set(ContextUserID, user.ID)
		c.Set(ContextRole, user.Role)
		c.Set("user", user) // 完整用户对象，供 handler/service 还原 Actor
		c.Next()
	}
}

func normalizeTokenHeader(header string) string {
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

// RequireRole 要求当前用户属于指定角色之一。
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(ContextRole)
		roleStr, _ := role.(string)
		for _, r := range roles {
			if r == roleStr {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
	}
}
