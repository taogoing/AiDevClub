package platform

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(h, prefix) {
			Fail(c, http.StatusUnauthorized, 40101, "未认证")
			c.Abort()
			return
		}
		uid, err := ParseAccessToken(secret, strings.TrimPrefix(h, prefix))
		if err != nil {
			Fail(c, http.StatusUnauthorized, 40101, "未认证")
			c.Abort()
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}

func OptionalAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(h, prefix) {
			if uid, err := ParseAccessToken(secret, strings.TrimPrefix(h, prefix)); err == nil {
				c.Set("user_id", uid)
			}
		}
		c.Next()
	}
}
