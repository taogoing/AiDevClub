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

func mcpServerRouter(t *testing.T) (*gin.Engine, *repo.UserRepo, *repo.McpServerRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	rdb := testutil.NewTestRedis(t)
	cfg := &platform.Config{
		DefaultPageSize:     20,
		MaxPageSize:         50,
		HotCacheTTL:         60e9,
		McpServerZipDir:     t.TempDir(),
		MaxResourceZipBytes: 10 << 20,
	}
	mcpRepo := repo.NewMcpServerRepo(db)
	notifSvc := service.NewNotificationService(repo.NewNotificationRepo(db), users)
	svc := service.NewMcpServerService(
		mcpRepo, repo.NewTagRepo(db),
		repo.NewInteractionRepo(db), rdb, cfg, notifSvc,
	)
	h := NewMcpServerHandler(svc)
	auth := platform.AuthMiddleware("s")
	opt := platform.OptionalAuthMiddleware("s")
	r := gin.New()
	g := r.Group("/api/v1/mcp-servers")
	g.POST("", auth, h.Create)
	g.PUT("/:id", auth, h.Update)
	g.DELETE("/:id", auth, h.Delete)
	g.GET("", h.List)
	g.GET("/:id", opt, h.Get)
	g.POST("/:id/submit", auth, h.Submit)
	g.POST("/:id/withdraw", auth, h.Withdraw)
	g.POST("/:id/archive", auth, h.Archive)
	g.POST("/:id/upload", auth, h.Upload)
	g.POST("/:id/download", h.Download)
	g.POST("/:id/like", auth, h.Like)
	g.POST("/:id/favorite", auth, h.Favorite)
	return r, users, mcpRepo
}

func TestMcpServerCreateEndpoint(t *testing.T) {
	r, users, _ := mcpServerRouter(t)
	u := &model.User{Email: "mc@a.com", PasswordHash: "x", Nickname: "MC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "my-mcp", "description": "desc", "repo_url": "https://github.com/x",
		"tools_json": `{"tools":[]}`, "readme": "# Readme",
		"tag_names": []string{"go"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct{ ID uint } `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ID == 0 {
		t.Fatalf("id = 0")
	}
}

func TestMcpServerListEndpoint(t *testing.T) {
	r, users, _ := mcpServerRouter(t)
	u := &model.User{Email: "mc@b.com", PasswordHash: "x", Nickname: "MC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "s1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/mcp-servers?page=1&page_size=10&sort=latest", nil)
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

func TestMcpServerListEndpointAuthorIDHidesHidden(t *testing.T) {
	r, users, servers := mcpServerRouter(t)
	owner := &model.User{Email: "hidden-mcp-handler@t.com", PasswordHash: "x", Nickname: "Owner"}
	if err := users.Create(owner); err != nil {
		t.Fatal(err)
	}
	hidden := &model.McpServer{
		AuthorID: owner.ID, Name: "hidden", Description: "content", ToolsJSON: "[]",
		Status: model.ResourceStatusPublished, Hidden: true,
	}
	if err := servers.Create(nil, hidden); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/mcp-servers?author_id=%d", owner.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			Total int64 `json:"total"`
			List  []any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 0 || len(response.Data.List) != 0 {
		t.Fatalf("public endpoint exposed hidden MCP server: %+v", response.Data)
	}
}

func TestMcpServerGetEndpoint(t *testing.T) {
	r, users, _ := mcpServerRouter(t)
	u := &model.User{Email: "mc@c.com", PasswordHash: "x", Nickname: "MC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "get-mcp"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	var created struct {
		Data struct{ ID uint } `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/mcp-servers/%d", created.Data.ID), nil)
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
	if detail.Data.Name != "get-mcp" {
		t.Fatalf("name = %q", detail.Data.Name)
	}
}

func TestMcpServerSubmitWithdrawEndpoint(t *testing.T) {
	r, users, _ := mcpServerRouter(t)
	u := &model.User{Email: "mc@d.com", PasswordHash: "x", Nickname: "MC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "flow-mcp"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	var created struct {
		Data struct{ ID uint } `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/submit", created.Data.ID), nil)
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
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/withdraw", created.Data.ID), nil)
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
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/submit", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("re-submit status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/submit", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("double submit status = %d, body %s", w.Code, w.Body.String())
	}
}

func TestMcpServerLikeEndpoint(t *testing.T) {
	r, users, mcpRepo := mcpServerRouter(t)
	u := &model.User{Email: "mc@e.com", PasswordHash: "x", Nickname: "MC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "like-mcp"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	var created struct {
		Data struct{ ID uint } `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	sv, _ := mcpRepo.FindByID(nil, created.Data.ID)
	sv.Status = model.ResourceStatusPublished
	now := sv.CreatedAt
	sv.PublishedAt = &now
	_ = mcpRepo.Update(nil, sv)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/like", created.Data.ID), nil)
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
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/like", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &likeResp)
	if likeResp.Data.Liked || likeResp.Data.LikesCount != 0 {
		t.Fatalf("unlike resp = %+v", likeResp.Data)
	}
}

func TestMcpServerFavoriteEndpoint(t *testing.T) {
	r, users, mcpRepo := mcpServerRouter(t)
	u := &model.User{Email: "mc@f.com", PasswordHash: "x", Nickname: "MC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{"name": "fav-mcp"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	var created struct {
		Data struct{ ID uint } `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	sv, _ := mcpRepo.FindByID(nil, created.Data.ID)
	sv.Status = model.ResourceStatusPublished
	now := sv.CreatedAt
	sv.PublishedAt = &now
	_ = mcpRepo.Update(nil, sv)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/favorite", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("favorite status = %d, body %s", w.Code, w.Body.String())
	}
	var favResp struct {
		Data struct {
			Favorited      bool `json:"favorited"`
			FavoritesCount int  `json:"favorites_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &favResp)
	if !favResp.Data.Favorited || favResp.Data.FavoritesCount != 1 {
		t.Fatalf("fav resp = %+v", favResp.Data)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/mcp-servers/%d/favorite", created.Data.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &favResp)
	if favResp.Data.Favorited || favResp.Data.FavoritesCount != 0 {
		t.Fatalf("unfav resp = %+v", favResp.Data)
	}
}
