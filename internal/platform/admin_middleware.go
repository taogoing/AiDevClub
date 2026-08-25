package platform

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
)

type UserFinder interface {
	FindByID(id uint) (*model.User, error)
}

func AdminMiddleware(users UserFinder) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user_id")
		if !ok {
			Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未认证")
			c.Abort()
			return
		}
		uid, ok := v.(uint)
		if !ok {
			Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未认证")
			c.Abort()
			return
		}
		u, err := users.FindByID(uid)
		if err != nil {
			Fail(c, http.StatusForbidden, CodeForbidden, "无管理员权限")
			c.Abort()
			return
		}
		if u.Role != model.UserRoleAdmin {
			Fail(c, http.StatusForbidden, CodeForbidden, "无管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}
