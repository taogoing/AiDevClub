package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func rankingRouter(t *testing.T) (*gin.Engine, *service.ContentRankingService, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	svc := service.NewContentRankingService(rdb, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	h := NewRankingHandler(svc)
	r := gin.New()
	r.GET("/api/v1/articles/ranking", h.GetArticleRanking)
	r.GET("/api/v1/skills/ranking", h.GetSkillRanking)
	r.GET("/api/v1/mcp-servers/ranking", h.GetMcpServerRanking)
	return r, svc, db
}

func getRankingJSON(t *testing.T, r *gin.Engine, path string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", path, w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out["data"].(map[string]any)
}

func TestRankingEndpointsEmptyBoard(t *testing.T) {
	r, _, _ := rankingRouter(t)
	for _, tc := range []struct{ path, field string }{
		{"/api/v1/articles/ranking?page=1&page_size=5", "articles"},
		{"/api/v1/skills/ranking?page=1&page_size=5", "skills"},
		{"/api/v1/mcp-servers/ranking?page=1&page_size=5", "mcp_servers"},
	} {
		data := getRankingJSON(t, r, tc.path)
		if data[tc.field] == nil {
			t.Fatalf("%s: %s must be [] not null, data=%v", tc.path, tc.field, data)
		}
		if arr, ok := data[tc.field].([]any); !ok || len(arr) != 0 {
			t.Fatalf("%s: %s want empty array, got %v", tc.path, tc.field, data[tc.field])
		}
		if data["total"].(float64) != 0 {
			t.Fatalf("%s: total=%v want 0", tc.path, data["total"])
		}
	}
}

func TestRankingEndpointReturnsScoredOrder(t *testing.T) {
	r, svc, db := rankingRouter(t)
	ctx := context.Background()
	author := &model.User{Email: "rank-handler@t.com", PasswordHash: "x", Nickname: "Rank", AvatarURL: "/x.png"}
	if err := db.Create(author).Error; err != nil {
		t.Fatal(err)
	}
	a1 := &model.Article{AuthorID: author.ID, Title: "one", Status: model.ArticleStatusPublished}
	a2 := &model.Article{AuthorID: author.ID, Title: "two", Status: model.ArticleStatusPublished}
	if err := db.Create(a1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(a2).Error; err != nil {
		t.Fatal(err)
	}
	_ = svc.AddScore(ctx, service.RankedContentArticle, a1.ID, 1)
	_ = svc.AddScore(ctx, service.RankedContentArticle, a2.ID, 5)

	data := getRankingJSON(t, r, "/api/v1/articles/ranking?page=1&page_size=5")
	arr := data["articles"].([]any)
	if len(arr) != 2 {
		t.Fatalf("articles len=%d", len(arr))
	}
	first := arr[0].(map[string]any)
	if first["id"].(float64) != float64(a2.ID) || first["title"] != "two" || first["score"].(float64) != 5 {
		t.Fatalf("first=%v want two/5", first)
	}
}

func TestRankingEndpointPageClamp(t *testing.T) {
	r, _, _ := rankingRouter(t)
	data := getRankingJSON(t, r, "/api/v1/articles/ranking?page=0&page_size=500")
	if data["page"].(float64) != 1 || data["page_size"].(float64) != 50 {
		t.Fatalf("clamp failed: page=%v page_size=%v", data["page"], data["page_size"])
	}
}

func TestRankingEndpointRedisDownDegradesEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	dead := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", DialTimeout: 100 * time.Millisecond, ReadTimeout: 100 * time.Millisecond,
	})
	svc := service.NewContentRankingService(dead, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	h := NewRankingHandler(svc)
	r := gin.New()
	r.GET("/api/v1/articles/ranking", h.GetArticleRanking)
	data := getRankingJSON(t, r, "/api/v1/articles/ranking?page=1&page_size=5")
	if arr, ok := data["articles"].([]any); !ok || len(arr) != 0 {
		t.Fatalf("degraded articles=%v", data["articles"])
	}
	if data["total"].(float64) != 0 {
		t.Fatalf("degraded total=%v", data["total"])
	}
}
