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
	t.Setenv("AIDEVCLUB_MCP_ADDR", "")
	t.Setenv("AIDEVCLUB_PUBLIC_BASE_URL", "")
	t.Setenv("AIDEVCLUB_MCP_ALLOWED_ORIGINS", "")
	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, ":8081", cfg.MCPAddr)
	assert.Equal(t, "http://localhost:5173", cfg.PublicBaseURL)
	assert.Empty(t, cfg.MCPAllowedOrigins)
	assert.Equal(t, 60, cfg.MCPRateLimitPerMin)
	assert.Equal(t, int64(1<<20), cfg.MCPMaxBodyBytes)
	assert.Equal(t, 30*time.Second, cfg.MCPRequestTimeout)
}

func TestLoadConfigMCPEnvironment(t *testing.T) {
	t.Setenv("AIDEVCLUB_MCP_ADDR", ":9091")
	t.Setenv("AIDEVCLUB_PUBLIC_BASE_URL", "https://aidevclub.example")
	t.Setenv("AIDEVCLUB_MCP_ALLOWED_ORIGINS", " https://admin.example , http://localhost:5173 ")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, ":9091", cfg.MCPAddr)
	assert.Equal(t, "https://aidevclub.example", cfg.PublicBaseURL)
	assert.Equal(t, []string{"https://admin.example", "http://localhost:5173"}, cfg.MCPAllowedOrigins)
}

func TestLoadConfigRejectsInvalidPublicBaseURL(t *testing.T) {
	for _, value := range []string{"/local", "ftp://aidevclub.example"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("AIDEVCLUB_PUBLIC_BASE_URL", value)
			t.Setenv("AIDEVCLUB_MCP_ALLOWED_ORIGINS", "")
			_, err := LoadConfig()
			require.Error(t, err)
		})
	}
}

func TestLoadConfigRejectsInvalidMCPAllowedOrigin(t *testing.T) {
	for _, value := range []string{
		"*",
		"https://*.example.com",
		"example.com",
		"https://admin.example/path",
		"https://admin.example?",
		"https://admin.example#",
	} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("AIDEVCLUB_PUBLIC_BASE_URL", "http://localhost:5173")
			t.Setenv("AIDEVCLUB_MCP_ALLOWED_ORIGINS", value)
			_, err := LoadConfig()
			require.Error(t, err)
		})
	}
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

func TestRateLimiterRepairsCounterMissingTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	ctx := context.Background()
	const key = "mcp:user:8"
	require.NoError(t, rdb.Set(ctx, key, 1, 0).Err())

	limiter := NewRateLimiter(rdb, 2, time.Minute)
	ok, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Greater(t, mr.TTL(key), time.Duration(0))
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
