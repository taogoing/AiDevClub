package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func setupNotifRouter(t *testing.T) (*gin.Engine, *service.NotificationService) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	_ = db.AutoMigrate(&model.Notification{})
	users := repo.NewUserRepo(db)
	notifRepo := repo.NewNotificationRepo(db)
	notifSvc := service.NewNotificationService(notifRepo, users)

	r := gin.New()
	r.Use(platform.RecoverMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	nh := NewNotificationHandler(notifSvc)
	r.GET("/api/v1/notifications", nh.List)
	r.GET("/api/v1/notifications/unread-count", nh.UnreadCount)
	r.PUT("/api/v1/notifications/:id/read", nh.MarkRead)
	r.PUT("/api/v1/notifications/read", nh.MarkAllRead)
	return r, notifSvc
}

func TestNotifListEmpty(t *testing.T) {
	r, _ := setupNotifRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp platform.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Fatalf("code = %d, msg %s", resp.Code, resp.Message)
	}
}

func TestNotifUnreadCount(t *testing.T) {
	r, svc := setupNotifRouter(t)
	_ = svc.Create(nil, 1, model.NotifTypeLikeArticle, "test", "content", "article", 1, 2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Count int64 `json:"count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Count != 1 {
		t.Fatalf("count = %d, want 1", resp.Data.Count)
	}
}

func TestNotifMarkRead(t *testing.T) {
	r, svc := setupNotifRouter(t)
	_ = svc.Create(nil, 1, model.NotifTypeLikeArticle, "test", "content", "article", 1, 2)

	res, _ := svc.List(nil, 1, "", 1, 20)
	if len(res.List) == 0 {
		t.Fatal("no notifications")
	}
	notifID := res.List[0].ID

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/"+itoa(notifID)+"/read", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	r.ServeHTTP(w, req)
	var resp struct {
		Data struct {
			Count int64 `json:"count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Count != 0 {
		t.Fatalf("count = %d, want 0", resp.Data.Count)
	}
}

func TestNotifMarkAllRead(t *testing.T) {
	r, svc := setupNotifRouter(t)
	_ = svc.Create(nil, 1, model.NotifTypeLikeArticle, "t1", "c1", "article", 1, 2)
	_ = svc.Create(nil, 1, model.NotifTypeLikeSkill, "t2", "c2", "skill", 2, 2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/read", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	r.ServeHTTP(w, req)
	var resp struct {
		Data struct {
			Count int64 `json:"count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Count != 0 {
		t.Fatalf("count = %d, want 0", resp.Data.Count)
	}
}

func itoa(v uint) string {
	return fmt.Sprintf("%d", v)
}
