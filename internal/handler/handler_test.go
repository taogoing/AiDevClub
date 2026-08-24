package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func setupRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := &platform.Config{
		DefaultAvatarURL: "/static/avatars/default.png",
		JWTSecret:        "s",
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
		RateLimitPerMin:  1000,
		AvatarDir:        t.TempDir(),
		MaxAvatarBytes:   2 << 20,
	}
	users := repo.NewUserRepo(testutil.NewTestDB(t))
	tokens := repo.NewTokenRepo(testutil.NewTestRedis(t), time.Hour)
	authSvc := service.NewAuthService(users, tokens, cfg)
	userSvc := service.NewUserService(users, tokens, cfg)

	r := gin.New()
	r.Use(platform.RecoverMiddleware())
	ah := NewAuthHandler(authSvc)
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", ah.Register)
	auth.POST("/login", ah.Login)
	auth.POST("/refresh", ah.Refresh)
	auth.POST("/logout", ah.Logout)

	uh := NewUserHandler(userSvc)
	me := r.Group("/api/v1/users", platform.AuthMiddleware(cfg.JWTSecret))
	me.GET("/me", uh.Me)
	me.PATCH("/me", uh.Update)
	me.PUT("/me/password", uh.ChangePassword)
	me.DELETE("/me", uh.Delete)
	return r
}

func TestRegisterLoginMe(t *testing.T) {
	r := setupRouter(t)

	body, _ := json.Marshal(map[string]string{"email": "a@example.com", "password": "secret123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+resp.Data.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me status = %d, body %s", w.Code, w.Body.String())
	}
}
