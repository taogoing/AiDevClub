package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func commentRouter(t *testing.T) (*gin.Engine, *repo.UserRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	rdb := testutil.NewTestRedis(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50, HotCacheTTL: 60e9}
	artSvc := service.NewArticleService(
		repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewCategoryRepo(db),
		repo.NewInteractionRepo(db), rdb, cfg,
	)
	_ = repo.NewCategoryRepo(db).Seed(t.Context())
	comSvc := service.NewCommentService(
		repo.NewCommentRepo(db), repo.NewArticleRepo(db),
		repo.NewInteractionRepo(db), users,
	)
	auth := platform.AuthMiddleware("s")
	ah := NewArticleHandler(artSvc)
	ch := NewCommentHandler(comSvc)
	r := gin.New()
	arts := r.Group("/api/v1/articles")
	arts.POST("", auth, ah.Create)
	artComments := r.Group("/api/v1/articles/:id/comments")
	artComments.GET("", ch.List)
	artComments.POST("", auth, ch.Create)
	coms := r.Group("/api/v1/comments")
	coms.DELETE("/:id", auth, ch.Delete)
	coms.POST("/:id/like", auth, ch.Like)
	return r, users
}

func TestCommentEndpoint(t *testing.T) {
	r, users := commentRouter(t)
	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{
		"title": "t", "content": "c", "category_id": 1, "status": "published",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create article status = %d", w.Code)
	}
	var art struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &art)

	body, _ = json.Marshal(map[string]interface{}{"content": "hello"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/comments", art.Data.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create comment status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/articles/%d/comments", art.Data.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list comments status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/comments", art.Data.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth comment status = %d", w.Code)
	}
}
