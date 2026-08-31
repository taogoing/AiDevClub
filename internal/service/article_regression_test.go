package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newRegressionArticleService(t *testing.T) (*ArticleService, *model.User, *redis.Client) {
	t.Helper()
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	u := &model.User{Email: "regression@example.com", PasswordHash: "x", Nickname: "Regression"}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}
	rdb := testutil.NewTestRedis(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50, HotCacheTTL: time.Minute}
	notif := NewNotificationService(repo.NewNotificationRepo(db), users)
	return NewArticleService(repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), rdb, cfg, notif), u, rdb
}

func TestArticleCanBeCreatedWithTagsWithoutCategory(t *testing.T) {
	svc, user, _ := newRegressionArticleService(t)
	a, err := svc.Create(context.Background(), user.ID, CreateArticleInput{
		Title: "无分类文章", Content: "正文", Status: model.ArticleStatusPublished, TagNames: []string{"Go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 {
		t.Fatal("article was not created")
	}
}

func TestArticleMutationInvalidatesHotCaches(t *testing.T) {
	svc, user, rdb := newRegressionArticleService(t)
	ctx := context.Background()
	if err := rdb.Set(ctx, "hot:articles:1:5", "stale", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Set(ctx, "hot:tags:5", "stale", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, user.ID, CreateArticleInput{Title: "缓存测试", Content: "正文", Status: model.ArticleStatusPublished, TagNames: []string{"Go"}}); err != nil {
		t.Fatal(err)
	}
	if n, _ := rdb.Exists(ctx, "hot:articles:1:5", "hot:tags:5").Result(); n != 0 {
		t.Fatalf("hot caches still exist: %d", n)
	}
}
