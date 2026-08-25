package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func setupReportRouter(t *testing.T) (*gin.Engine, *service.ReportService, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	articles := repo.NewArticleRepo(db)
	skills := repo.NewSkillRepo(db)
	mcpServers := repo.NewMcpServerRepo(db)
	comments := repo.NewCommentRepo(db)
	resourceComments := repo.NewResourceCommentRepo(db)
	reportRepo := repo.NewReportRepo(db)
	adminLogRepo := repo.NewAdminLogRepo(db)
	announcementRepo := repo.NewAnnouncementRepo(db)
	notifRepo := repo.NewNotificationRepo(db)

	notifSvc := service.NewNotificationService(notifRepo, users)
	adminLogSvc := service.NewAdminLogService(adminLogRepo, users)
	adminSvc := service.NewAdminService(users, articles, skills, mcpServers, comments, resourceComments, reportRepo, announcementRepo, adminLogSvc, notifSvc)
	reportSvc := service.NewReportService(reportRepo, articles, skills, mcpServers, comments, resourceComments, adminSvc, adminLogSvc, notifSvc)

	r := gin.New()
	r.Use(platform.RecoverMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	rh := NewReportHandler(reportSvc)
	r.POST("/api/v1/reports", rh.Create)
	r.GET("/api/v1/reports", rh.List)
	return r, reportSvc, db
}

func seedUser(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	u := &model.User{Email: "seed@test.com", PasswordHash: "x", Nickname: "Seed", AvatarURL: "/x.png"}
	if err := db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func seedCategory(t *testing.T, db *gorm.DB) {
	t.Helper()
	_ = repo.NewCategoryRepo(db).Seed(nil)
}

func TestReportCreate(t *testing.T) {
	r, _, db := setupReportRouter(t)
	u := seedUser(t, db)
	seedCategory(t, db)

	_ = db.Create(&model.Article{
		AuthorID: u.ID, CategoryID: 1, Title: "test", Status: model.ArticleStatusPublished,
	})
	var art model.Article
	db.First(&art)

	body, _ := json.Marshal(map[string]interface{}{
		"target_type": "article",
		"target_id":   art.ID,
		"reason":      "spam",
		"description": "this is spam",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ID == 0 {
		t.Fatal("expected report id > 0")
	}
}

func TestReportCreateBadTarget(t *testing.T) {
	r, _, _ := setupReportRouter(t)
	body, _ := json.Marshal(map[string]interface{}{
		"target_type": "article",
		"target_id":   99999,
		"reason":      "spam",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp platform.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code == 0 {
		t.Fatal("expected error for non-existent target")
	}
}

func TestReportList(t *testing.T) {
	r, _, db := setupReportRouter(t)
	u := seedUser(t, db)
	seedCategory(t, db)

	_ = db.Create(&model.Article{
		AuthorID: u.ID, CategoryID: 1, Title: "test", Status: model.ArticleStatusPublished,
	})
	var art model.Article
	db.First(&art)

	body, _ := json.Marshal(map[string]interface{}{
		"target_type": "article",
		"target_id":   art.ID,
		"reason":      "abuse",
		"description": "abusive content",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/reports", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			List  []interface{} `json:"list"`
			Total int64         `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Total != 1 {
		t.Fatalf("total = %d, want 1", resp.Data.Total)
	}
}
