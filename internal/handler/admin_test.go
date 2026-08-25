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

func setupAdminRouter(t *testing.T) (*gin.Engine, *service.AdminService, *service.ReportService, *service.AdminLogService, *gorm.DB) {
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
	adminLogSvc := service.NewAdminLogService(adminLogRepo)
	adminSvc := service.NewAdminService(users, articles, skills, mcpServers, comments, resourceComments, reportRepo, announcementRepo, adminLogSvc, notifSvc)
	reportSvc := service.NewReportService(reportRepo, articles, skills, mcpServers, comments, resourceComments, adminSvc, adminLogSvc, notifSvc)

	r := gin.New()
	r.Use(platform.RecoverMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	adminH := NewAdminHandler(adminSvc, reportSvc, adminLogSvc)
	adminH.RegisterRoutes(r.Group("/api/v1/admin"))
	return r, adminSvc, reportSvc, adminLogSvc, db
}

func seedAdminUser(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	u := &model.User{Email: "admin@test.com", PasswordHash: "x", Nickname: "Admin", AvatarURL: "/x.png"}
	if err := db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func seedAdminCategory(t *testing.T, db *gorm.DB) {
	t.Helper()
	_ = repo.NewCategoryRepo(db).Seed(nil)
}

func TestAdminDashboard(t *testing.T) {
	r, _, _, _, _ := setupAdminRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			TotalUsers int64 `json:"total_users"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.TotalUsers != 0 {
		t.Fatalf("total_users = %d, want 0", resp.Data.TotalUsers)
	}
}

func TestAdminHideUnhideArticle(t *testing.T) {
	r, _, _, _, db := setupAdminRouter(t)
	u := seedAdminUser(t, db)
	seedAdminCategory(t, db)

	_ = db.Create(&model.Article{
		AuthorID: u.ID, CategoryID: 1, Title: "hide-me", Status: model.ArticleStatusPublished,
	})
	var art model.Article
	db.First(&art)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/articles/"+itoa(art.ID)+"/hide", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hide status = %d, body %s", w.Code, w.Body.String())
	}

	var artAfter model.Article
	db.First(&artAfter, art.ID)
	if !artAfter.Hidden {
		t.Fatal("expected article to be hidden")
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/admin/articles/"+itoa(art.ID)+"/unhide", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unhide status = %d, body %s", w.Code, w.Body.String())
	}

	db.First(&artAfter, art.ID)
	if artAfter.Hidden {
		t.Fatal("expected article to be unhidden")
	}
}

func TestAdminResolveReport(t *testing.T) {
	r, _, reportSvc, _, db := setupAdminRouter(t)
	u := seedAdminUser(t, db)
	seedAdminCategory(t, db)

	_ = db.Create(&model.Article{
		AuthorID: u.ID, CategoryID: 1, Title: "report-me", Status: model.ArticleStatusPublished,
	})
	var art model.Article
	db.First(&art)

	_, _ = reportSvc.Create(nil, 1, "article", art.ID, model.ReportReasonSpam, "spam content")

	body, _ := json.Marshal(map[string]interface{}{
		"action": "dismiss",
		"result": "not a violation",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/reports/1/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, body %s", w.Code, w.Body.String())
	}

	var rpt model.Report
	db.First(&rpt, 1)
	if rpt.Status != model.ReportStatusDismissed {
		t.Fatalf("status = %s, want dismissed", rpt.Status)
	}
}

func TestAdminCreateListAnnouncement(t *testing.T) {
	r, _, _, _, _ := setupAdminRouter(t)

	body, _ := json.Marshal(map[string]string{
		"title":   "System Update",
		"content": "We will have maintenance tonight",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/announcements", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body %s", w.Code, w.Body.String())
	}
	var createResp struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	if createResp.Data.ID == 0 {
		t.Fatal("expected announcement id > 0")
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/admin/announcements", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data struct {
			List  []interface{} `json:"list"`
			Total int64         `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp.Data.Total != 1 {
		t.Fatalf("total = %d, want 1", listResp.Data.Total)
	}
}

func TestAdminListLogs(t *testing.T) {
	r, _, _, _, _ := setupAdminRouter(t)

	body, _ := json.Marshal(map[string]string{
		"title":   "Log Test",
		"content": "This should generate a log entry",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/announcements", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create announcement status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/admin/logs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logs status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Total < 1 {
		t.Fatalf("total = %d, want >= 1", resp.Data.Total)
	}
}
