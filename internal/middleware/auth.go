package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mimo-webui/internal/auth"
)

type AuthUser struct {
	UserID   int64
	Username string
	Role     string
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		if cookie, err := c.Cookie("token"); err == nil && cookie != "" {
			tokenStr = cookie
		}
		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			if isHTMX(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			} else {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
			}
			return
		}

		claims, err := auth.ValidateToken(jwtSecret, tokenStr)
		if err != nil {
			if isHTMX(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已过期"})
			} else {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
			}
			return
		}

		c.Set("user", AuthUser{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		})
		c.Next()
	}
}

func GetAuthUser(c *gin.Context) AuthUser {
	user, _ := c.Get("user")
	return user.(AuthUser)
}

func isHTMX(c *gin.Context) bool {
	return c.GetHeader("HX-Request") == "true"
}
