package service

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newArticleDailyEnv(t *testing.T) (*ArticleService, *CommentService, *ContentRankingService, *gorm.DB, *model.User, *model.User) {
	t.Helper()
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	users := repo.NewUserRepo(db)
	author := seedUser(t, db, "author@t.com")
	viewer := seedUser(t, db, "viewer@t.com")
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	rankSvc := NewContentRankingService(rdb, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	articleSvc := NewArticleService(repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), cfg, notifSvc, rankSvc)
	commentSvc := NewCommentService(repo.NewCommentRepo(db), repo.NewArticleRepo(db), repo.NewInteractionRepo(db), users, notifSvc, rankSvc)
	return articleSvc, commentSvc, rankSvc, db, author, viewer
}

func seedPublishedArticle(t *testing.T, db *gorm.DB, authorID uint, title string) *model.Article {
	t.Helper()
	return seedArticle(t, db, authorID, title, model.ArticleStatusPublished, false)
}

func TestArticleDailyViewScoresOnce(t *testing.T) {
	svc, _, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	if _, err := svc.Get(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 1 || len(items) != 1 || items[0].ID != a.ID || items[0].Score != 1 {
		t.Fatalf("items=%+v total=%d", items, total)
	}
}

func TestArticleDailyAuthorAndGuestNotScored(t *testing.T) {
	svc, _, rankSvc, db, author, _ := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	if _, err := svc.Get(ctx, author.ID, a.ID); err != nil { // 作者本人
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, 0, a.ID); err != nil { // 游客
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("author/guest must not score: items=%+v total=%d", items, total)
	}
}

func TestArticleDailyToggleLikeAndFavorite(t *testing.T) {
	svc, _, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	if _, _, err := svc.ToggleLike(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListArticles(ctx, 1, 10)
	if items[0].Score != 2 {
		t.Fatalf("after like score=%d want 2", items[0].Score)
	}
	if _, _, err := svc.ToggleLike(ctx, viewer.ID, a.ID); err != nil { // 取消点赞
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("like+unlike must empty the board: items=%+v total=%d", items, total)
	}

	if _, _, err := svc.ToggleFavorite(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, _, _ = rankSvc.ListArticles(ctx, 1, 10)
	if items[0].Score != 2 {
		t.Fatalf("after favorite score=%d want 2", items[0].Score)
	}
	if _, _, err := svc.ToggleFavorite(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ = rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("favorite+unfavorite must empty the board: items=%+v total=%d", items, total)
	}
}

func TestArticleDailyCommentCreateDelete(t *testing.T) {
	_, commentSvc, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	c, err := commentSvc.Create(ctx, viewer.ID, a.ID, "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListArticles(ctx, 1, 10)
	if items[0].Score != 3 {
		t.Fatalf("after comment score=%d want 3", items[0].Score)
	}
	if err := commentSvc.Delete(ctx, viewer.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("comment delete must empty: items=%+v total=%d", items, total)
	}
}

func TestArticleDailyDeleteAndDraftRemove(t *testing.T) {
	svc, _, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")
	_, _ = svc.Get(ctx, viewer.ID, a.ID)
	if _, _, err := svc.ToggleLike(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(ctx, author.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("delete must remove from board: items=%+v total=%d", items, total)
	}

	b := seedPublishedArticle(t, db, author.ID, "B")
	_, _ = svc.Get(ctx, viewer.ID, b.ID)
	if _, err := svc.Update(ctx, author.ID, b.ID, CreateArticleInput{
		Title: "B", Summary: "", Content: "c", Status: model.ArticleStatusDraft,
	}); err != nil {
		t.Fatal(err)
	}
	items, total, _ = rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("publish->draft must remove from board: items=%+v total=%d", items, total)
	}
}
