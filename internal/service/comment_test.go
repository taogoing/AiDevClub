package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

func newCommentTestEnv(t *testing.T) (*CommentService, *ArticleService, *model.User, *model.Category) {
	t.Helper()
	asvc, u, cat := newArticleTestEnv(t)
	db := asvc.articles.DB()
	svc := NewCommentService(
		repo.NewCommentRepo(db),
		asvc.articles,
		asvc.inter,
		repo.NewUserRepo(db),
		asvc.notifSvc,
	)
	return svc, asvc, u, cat
}

func TestCommentCreateListDelete(t *testing.T) {
	svc, asvc, u, cat := newCommentTestEnv(t)
	ctx := context.Background()
	a, _ := asvc.Create(ctx, u.ID, CreateArticleInput{Title: "t", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})

	c1, err := svc.Create(ctx, u.ID, a.ID, "一级评论", nil)
	if err != nil {
		t.Fatal(err)
	}
	pid := c1.ID
	c2, err := svc.Create(ctx, u.ID, a.ID, "回复", &pid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, u.ID, a.ID, "回复的回复", &c2.ID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(ctx, a.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if len(list[0].Replies) != 2 {
		t.Fatalf("replies = %d, want 2", len(list[0].Replies))
	}
	if err := svc.Delete(ctx, u.ID, c2.ID); err != nil {
		t.Fatal(err)
	}
	liked, count, err := svc.ToggleLike(ctx, u.ID, c1.ID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("like = %v, %d, %v", liked, count, err)
	}
	other := &model.User{Email: "b@b.com", PasswordHash: "x", Nickname: "B", AvatarURL: "/x.png"}
	_ = repo.NewUserRepo(asvc.articles.DB()).Create(other)
	if err := svc.Delete(ctx, other.ID, c1.ID); err == nil {
		t.Fatal("other user deleted comment")
	}
}

func TestCommentArticleAuthorDeleteOthersComment(t *testing.T) {
	svc, asvc, u, cat := newCommentTestEnv(t)
	ctx := context.Background()
	a, _ := asvc.Create(ctx, u.ID, CreateArticleInput{Title: "t", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})

	other := &model.User{Email: "b@b.com", PasswordHash: "x", Nickname: "B", AvatarURL: "/x.png"}
	_ = repo.NewUserRepo(asvc.articles.DB()).Create(other)

	c, err := svc.Create(ctx, other.ID, a.ID, "别人的评论", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, u.ID, c.ID); err != nil {
		t.Fatalf("article author should delete others' comment: %v", err)
	}
}

func TestCommentListDraftArticle(t *testing.T) {
	svc, asvc, u, cat := newCommentTestEnv(t)
	ctx := context.Background()
	draft, _ := asvc.Create(ctx, u.ID, CreateArticleInput{Title: "t", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusDraft})
	if _, err := svc.List(ctx, draft.ID); err == nil {
		t.Fatal("should not list comments on draft article")
	}
}

func TestArticleToggleLikeDraft(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	draft, _ := svc.Create(ctx, u.ID, CreateArticleInput{Title: "t", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusDraft})
	if _, _, err := svc.ToggleLike(ctx, u.ID, draft.ID); err == nil {
		t.Fatal("should not like draft article")
	}
}
