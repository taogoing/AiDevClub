package platform

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RecoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				Fail(c, http.StatusInternalServerError, CodeInternalError, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		for _, e := range c.Errors {
			if be, ok := e.Err.(*BizError); ok {
				Fail(c, be.Status, be.Code, be.Message)
				return
			}
		}
		Fail(c, http.StatusInternalServerError, CodeInternalError, "服务器内部错误")
	}
}
