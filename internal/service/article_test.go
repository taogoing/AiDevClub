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

func TestArticleListAndGet(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	pub, _ := svc.Create(ctx, u.ID, CreateArticleInput{
		Title: "公开", Content: "c", CategoryID: cat.ID,
		Status: model.ArticleStatusPublished, TagNames: []string{"gin"},
	})
	draft, _ := svc.Create(ctx, u.ID, CreateArticleInput{
		Title: "草稿", Content: "c", CategoryID: cat.ID,
		Status: model.ArticleStatusDraft,
	})

	res, err := svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "latest"})
	if err != nil || res.Total != 1 || len(res.List) != 1 {
		t.Fatalf("list = %+v, err %v", res, err)
	}
	if res.List[0].Title != "公开" || len(res.List[0].Tags) != 1 {
		t.Fatalf("summary = %+v", res.List[0])
	}

	res, _ = svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "latest", Keyword: "公开"})
	if res.Total != 1 {
		t.Fatalf("keyword total = %d", res.Total)
	}

	detail, err := svc.Get(ctx, 0, pub.ID)
	if err != nil || detail.Content != "c" {
		t.Fatalf("detail = %+v, err %v", detail, err)
	}
	if detail.Liked {
		t.Fatal("guest should not have liked")
	}

	if _, err := svc.Get(ctx, 0, draft.ID); err == nil {
		t.Fatal("draft visible to guest")
	}
	if _, err := svc.Get(ctx, u.ID, draft.ID); err != nil {
		t.Fatalf("author can't see draft: %v", err)
	}
}

func TestArticleHotSortAndCache(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	a1, _ := svc.Create(ctx, u.ID, CreateArticleInput{Title: "high", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})
	a2, _ := svc.Create(ctx, u.ID, CreateArticleInput{Title: "low", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})

	db := svc.articles.DB()
	_ = repo.NewArticleRepo(db).IncrCount(db, a1.ID, "likes_count", 2)

	res, err := svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "hot"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.List) < 2 {
		t.Fatalf("list len = %d, want >= 2", len(res.List))
	}
	if res.List[0].ID != a1.ID {
		t.Fatalf("hot first = %d, want %d", res.List[0].ID, a1.ID)
	}
	res2, _ := svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "hot"})
	if res2.List[0].ID != a1.ID {
		t.Fatal("cached hot list wrong")
	}
	_ = a2
}

func TestArticleToggleLike(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, u.ID, CreateArticleInput{Title: "t", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})
	liked, count, err := svc.ToggleLike(ctx, u.ID, a.ID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("like = %v, %d, %v", liked, count, err)
	}
	liked, count, _ = svc.ToggleLike(ctx, u.ID, a.ID)
	if liked || count != 0 {
		t.Fatalf("unlike = %v, %d", liked, count)
	}
}
