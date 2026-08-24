package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func TestTagEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tagRepo := repo.NewTagRepo(db)
	ctx := context.Background()
	_, _ = tagRepo.Create(ctx, "gin")
	_, _ = tagRepo.Create(ctx, "gorm")
	h := NewTagHandler(service.NewTagService(tagRepo))
	r := gin.New()
	r.GET("/api/v1/tags", h.List)

	// keyword filter
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tags?keyword=gi", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 tag with keyword=gi, got %d", len(resp.Data))
	}

	// hot list
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/tags?hot=1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hot status = %d, body %s", w.Code, w.Body.String())
	}
}
