package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOKWritesUnifiedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ok", func(c *gin.Context) { OK(c, gin.H{"id": 1}) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != `{"code":0,"message":"ok","data":{"id":1}}` {
		t.Fatalf("body = %s", got)
	}
}

func TestErrorMiddlewareMapsBizError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RecoverMiddleware())
	r.GET("/err", func(c *gin.Context) {
		_ = c.Error(NewBizError(http.StatusConflict, 40901, "邮箱已存在"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/err", nil))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if got := w.Body.String(); got != `{"code":40901,"message":"邮箱已存在"}` {
		t.Fatalf("body = %s", got)
	}
}
