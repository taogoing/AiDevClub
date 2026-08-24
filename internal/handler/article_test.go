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

func articleRouter(t *testing.T) (*gin.Engine, *repo.UserRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	rdb := testutil.NewTestRedis(t)
	cfg := &platform.Config{
		DefaultPageSize: 20,
		MaxPageSize:     50,
		HotCacheTTL:     60e9,
	}
	svc := service.NewArticleService(
		repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewCategoryRepo(db),
		repo.NewInteractionRepo(db), rdb, cfg,
	)
	_ = repo.NewCategoryRepo(db).Seed(t.Context())
	h := NewArticleHandler(svc)
	auth := platform.AuthMiddleware("s")
	opt := platform.OptionalAuthMiddleware("s")
	r := gin.New()
	art := r.Group("/api/v1/articles")
	art.POST("", auth, h.Create)
	art.PUT("/:id", auth, h.Update)
	art.DELETE("/:id", auth, h.Delete)
	art.GET("", h.List)
	art.GET("/:id", opt, h.Get)
	art.POST("/:id/like", auth, h.Like)
	art.POST("/:id/favorite", auth, h.Favorite)
	return r, users
}

func TestArticleCreateEndpoint(t *testing.T) {
	r, users := articleRouter(t)
	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{
		"title": "t", "content": "c", "category_id": 1, "status": "draft",
		"tag_names": []string{"gin"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}

func TestArticleUpdateDeleteEndpoint(t *testing.T) {
	r, users := articleRouter(t)
	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{
		"title": "t", "content": "c", "category_id": 1, "status": "draft",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	body, _ = json.Marshal(map[string]interface{}{
		"title": "t2", "content": "c2", "category_id": 1, "status": "published",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/articles/%d", created.Data.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d, body %s", w.Code, w.Body.String())
	}

	u2 := &model.User{Email: "b@b.com", PasswordHash: "x", Nickname: "B", AvatarURL: "/x.png"}
	_ = users.Create(u2)
	tok2, _ := platform.GenerateAccessToken("s", time.Minute, u2.ID)
	body, _ = json.Marshal(map[string]interface{}{
		"title": "x", "content": "y", "category_id": 1, "status": "draft",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/articles/%d", created.Data.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok2)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-author update status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/articles/%d", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body %s", w.Code, w.Body.String())
	}
}

func TestArticleListGetEndpoint(t *testing.T) {
	r, users := articleRouter(t)
	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{
		"title": "公开", "content": "c", "category_id": 1, "status": "published",
		"tag_names": []string{"gin"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d", w.Code)
	}
	var created struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	body, _ = json.Marshal(map[string]interface{}{
		"title": "草稿", "content": "c", "category_id": 1, "status": "draft",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d", w.Code)
	}
	var listResp struct {
		Data struct {
			Total int64         `json:"total"`
			List  []interface{} `json:"list"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp.Data.Total != 1 {
		t.Fatalf("list total = %d, want 1", listResp.Data.Total)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", created.Data.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, body %s", w.Code, w.Body.String())
	}
}

func TestArticleLikeEndpoint(t *testing.T) {
	r, users := articleRouter(t)
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
	var created struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/like", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("like status = %d, body %s", w.Code, w.Body.String())
	}
	var likeResp struct {
		Data struct {
			Liked      bool `json:"liked"`
			LikesCount int  `json:"likes_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &likeResp)
	if !likeResp.Data.Liked || likeResp.Data.LikesCount != 1 {
		t.Fatalf("like resp = %+v", likeResp.Data)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/like", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &likeResp)
	if likeResp.Data.Liked || likeResp.Data.LikesCount != 0 {
		t.Fatalf("unlike resp = %+v", likeResp.Data)
	}
}
