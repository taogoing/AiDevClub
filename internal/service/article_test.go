package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newArticleTestEnv(t *testing.T) (*ArticleService, *model.User, *model.Category) {
	t.Helper()
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}
	cats := repo.NewCategoryRepo(db)
	_ = cats.Seed(context.Background())
	catList, _ := cats.List(context.Background())
	cfg := &platform.Config{
		DefaultPageSize: 20,
		MaxPageSize:     50,
		HotCacheTTL:     60e9,
	}
	svc := NewArticleService(
		repo.NewArticleRepo(db),
		repo.NewTagRepo(db),
		cats,
		repo.NewInteractionRepo(db),
		testutil.NewTestRedis(t),
		cfg,
	)
	return svc, u, &catList[0]
}

func TestArticleCreate(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	db := svc.articles.DB()
	a, err := svc.Create(context.Background(), u.ID, CreateArticleInput{
		Title: "Hello", Content: "world", CategoryID: cat.ID,
		Status: model.ArticleStatusDraft, TagNames: []string{"gin", "gorm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 || a.Status != model.ArticleStatusDraft {
		t.Fatalf("article = %+v", a)
	}
	tagRepo := repo.NewTagRepo(db)
	for _, name := range []string{"gin", "gorm"} {
		tg, err := tagRepo.FindByName(context.Background(), name)
		if err != nil {
			t.Fatalf("tag %q not found: %v", name, err)
		}
		if tg.UsageCount != 1 {
			t.Fatalf("tag %q usage_count = %d, want 1", name, tg.UsageCount)
		}
	}
	artRepo := repo.NewArticleRepo(db)
	tagIDs, err := artRepo.FindArticleTags(db, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tagIDs) != 2 {
		t.Fatalf("article tags = %d, want 2", len(tagIDs))
	}
}

func TestArticleUpdateAndDelete(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, u.ID, CreateArticleInput{
		Title: "t", Content: "c", CategoryID: cat.ID,
		Status: model.ArticleStatusDraft, TagNames: []string{"gin"},
	})
	got, err := svc.Update(ctx, u.ID, a.ID, CreateArticleInput{
		Title: "t2", Content: "c2", CategoryID: cat.ID,
		Status: model.ArticleStatusPublished, TagNames: []string{"gorm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.ArticleStatusPublished || got.Title != "t2" {
		t.Fatalf("updated = %+v", got)
	}
	if _, err := svc.Update(ctx, u.ID+999, a.ID, CreateArticleInput{Title: "x", Content: "y", CategoryID: cat.ID, Status: model.ArticleStatusDraft}); err == nil {
		t.Fatal("non-author update allowed")
	}
	if err := svc.Delete(ctx, u.ID+999, a.ID); err == nil {
		t.Fatal("non-author delete allowed")
	}
	if err := svc.Delete(ctx, u.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	db := svc.articles.DB()
	artRepo := repo.NewArticleRepo(db)
	if _, err := artRepo.FindByID(db, a.ID); err == nil {
		t.Fatal("deleted article still found")
	}
}
