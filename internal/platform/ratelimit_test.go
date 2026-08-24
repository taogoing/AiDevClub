package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/testutil"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := testutil.NewTestRedis(t)
	r := gin.New()
	r.Use(RateLimitMiddleware(rdb, 2, time.Minute))
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	do := func() int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		return w.Code
	}
	if do() != 200 || do() != 200 {
		t.Fatal("first two requests should pass")
	}
	if do() != http.StatusTooManyRequests {
		t.Fatal("third request should be rate limited")
	}
}
