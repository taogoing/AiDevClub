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

func skillRouter(t *testing.T) (*gin.Engine, *repo.UserRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	rdb := testutil.NewTestRedis(t)
	cfg := &platform.Config{
		DefaultPageSize:     20,
		MaxPageSize:         50,
		HotCacheTTL:         60e9,
		SkillZipDir:         t.TempDir(),
		MaxResourceZipBytes: 10 << 20,
	}
	svc := service.NewSkillService(
		repo.NewSkillRepo(db), repo.NewTagRepo(db),
		repo.NewInteractionRepo(db), rdb, cfg,
	)
	h := NewSkillHandler(svc)
	auth := platform.AuthMiddleware("s")
	opt := platform.OptionalAuthMiddleware("s")
	r := gin.New()
	g := r.Group("/api/v1/skills")
	g.POST("", auth, h.Create)
	g.PUT("/:id", auth, h.Update)
	g.DELETE("/:id", auth, h.Delete)
	g.GET("", h.List)
	g.GET("/:id", opt, h.Get)
	g.POST("/:id/submit", auth, h.Submit)
	g.POST("/:id/withdraw", auth, h.Withdraw)
	g.POST("/:id/archive", auth, h.Archive)
	return r, users
}

func TestSkillCreateEndpoint(t *testing.T) {
	r, users := skillRouter(t)
	u := &model.User{Email: "sk@a.com", PasswordHash: "x", Nickname: "SK", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "my-skill", "description": "desc", "repo_url": "https://github.com/x",
		"tag_names": []string{"go"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ID == 0 {
		t.Fatalf("id = 0")
	}
}

func TestSkillListEndpoint(t *testing.T) {
	r, users := skillRouter(t)
	u := &model.User{Email: "sk@b.com", PasswordHash: "x", Nickname: "SK", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "s1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/skills?page=1&page_size=10&sort=latest", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data struct {
			Total int64         `json:"total"`
			List  []interface{} `json:"list"`
			Page  int           `json:"page"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp.Data.Page != 1 {
		t.Fatalf("page = %d, want 1", listResp.Data.Page)
	}
}

func TestSkillGetEndpoint(t *testing.T) {
	r, users := skillRouter(t)
	u := &model.User{Email: "sk@c.com", PasswordHash: "x", Nickname: "SK", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "get-skill"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	var created struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/skills/%d", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, body %s", w.Code, w.Body.String())
	}
	var detail struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if detail.Data.Name != "get-skill" {
		t.Fatalf("name = %q", detail.Data.Name)
	}
}

func TestSkillSubmitWithdrawEndpoint(t *testing.T) {
	r, users := skillRouter(t)
	u := &model.User{Email: "sk@d.com", PasswordHash: "x", Nickname: "SK", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "flow-skill"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	var created struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%d/submit", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("submit status = %d, body %s", w.Code, w.Body.String())
	}
	var submitResp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &submitResp)
	if submitResp.Data.Status != "pending_review" {
		t.Fatalf("submit status = %q", submitResp.Data.Status)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%d/withdraw", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("withdraw status = %d, body %s", w.Code, w.Body.String())
	}
	var withdrawResp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &withdrawResp)
	if withdrawResp.Data.Status != "draft" {
		t.Fatalf("withdraw status = %q", withdrawResp.Data.Status)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%d/submit", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("re-submit status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%d/submit", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("double submit status = %d, body %s", w.Code, w.Body.String())
	}
}
