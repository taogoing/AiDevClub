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

func resCommentRouter(t *testing.T) (*gin.Engine, *repo.UserRepo, *repo.SkillRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	skillRepo := repo.NewSkillRepo(db)
	mcpRepo := repo.NewMcpServerRepo(db)
	notifSvc := service.NewNotificationService(repo.NewNotificationRepo(db), users)
	rcSvc := service.NewResourceCommentService(
		repo.NewResourceCommentRepo(db),
		skillRepo,
		mcpRepo,
		repo.NewInteractionRepo(db),
		users,
		notifSvc,
	)
	h := NewResourceCommentHandler(rcSvc)
	auth := platform.AuthMiddleware("s")
	r := gin.New()

	skillComments := r.Group("/api/v1/skills/:id/comments")
	skillComments.Use(func(c *gin.Context) { c.Set("resource_type", "skill") })
	skillComments.GET("", h.List)
	skillComments.POST("", auth, h.Create)

	mcpComments := r.Group("/api/v1/mcp-servers/:id/comments")
	mcpComments.Use(func(c *gin.Context) { c.Set("resource_type", "mcp_server") })
	mcpComments.GET("", h.List)
	mcpComments.POST("", auth, h.Create)

	resComments := r.Group("/api/v1/resource-comments")
	resComments.DELETE("/:id", auth, h.Delete)
	resComments.POST("/:id/like", auth, h.Like)

	return r, users, skillRepo
}

func TestResourceCommentListEndpoint(t *testing.T) {
	r, users, skillRepo := resCommentRouter(t)
	u := &model.User{Email: "rc@a.com", PasswordHash: "x", Nickname: "RC", AvatarURL: "/x.png"}
	_ = users.Create(u)

	sk := &model.Skill{AuthorID: u.ID, Name: "test-skill", Status: model.ResourceStatusPublished}
	now := time.Now()
	sk.PublishedAt = &now
	_ = skillRepo.Create(nil, sk)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/skills/%d/comments", sk.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data []interface{} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp.Data == nil {
		t.Fatal("data should not be nil")
	}
}

func TestResourceCommentCreateEndpoint(t *testing.T) {
	r, users, skillRepo := resCommentRouter(t)
	u := &model.User{Email: "rc@b.com", PasswordHash: "x", Nickname: "RC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	sk := &model.Skill{AuthorID: u.ID, Name: "test-skill", Status: model.ResourceStatusPublished}
	now := time.Now()
	sk.PublishedAt = &now
	_ = skillRepo.Create(nil, sk)

	body, _ := json.Marshal(map[string]interface{}{"content": "hello"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%d/comments", sk.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ID == 0 {
		t.Fatalf("id = 0")
	}

	body, _ = json.Marshal(map[string]interface{}{"content": "unauth"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%d/comments", sk.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d", w.Code)
	}
}

func TestResourceCommentDeleteAndLikeEndpoint(t *testing.T) {
	r, users, skillRepo := resCommentRouter(t)
	u := &model.User{Email: "rc@c.com", PasswordHash: "x", Nickname: "RC", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	sk := &model.Skill{AuthorID: u.ID, Name: "test-skill", Status: model.ResourceStatusPublished}
	now := time.Now()
	sk.PublishedAt = &now
	_ = skillRepo.Create(nil, sk)

	body, _ := json.Marshal(map[string]interface{}{"content": "hello"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%d/comments", sk.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	var createResp struct{ Data struct{ ID uint } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	commentID := createResp.Data.ID

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/resource-comments/%d/like", commentID), nil)
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
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/resource-comments/%d", commentID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body %s", w.Code, w.Body.String())
	}
}
