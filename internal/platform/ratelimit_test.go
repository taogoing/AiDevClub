package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aidevclub/internal/testutil"
)

func TestLoadConfigMCPDefaults(t *testing.T) {
	t.Setenv("AIDEVCLUB_MCP_ADDR", ":9091")
	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, ":9091", cfg.MCPAddr)
	assert.Equal(t, 60, cfg.MCPRateLimitPerMin)
	assert.Equal(t, int64(1<<20), cfg.MCPMaxBodyBytes)
	assert.Equal(t, 30*time.Second, cfg.MCPRequestTimeout)
}

func TestRateLimiterUsesProvidedIdentity(t *testing.T) {
	mr := miniredis.RunT(t)
	limiter := NewRateLimiter(redis.NewClient(&redis.Options{Addr: mr.Addr()}), 1, time.Minute)
	ok, err := limiter.Allow(context.Background(), "mcp:user:7")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = limiter.Allow(context.Background(), "mcp:user:7")
	require.NoError(t, err)
	assert.False(t, ok)
}

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
