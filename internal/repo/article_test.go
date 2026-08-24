package repo

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestArticleRepoCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewArticleRepo(db)
	ctx := context.Background()

	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	if err := NewUserRepo(db).Create(u); err != nil {
		t.Fatal(err)
	}
	cat := &model.Category{Name: "Go", Slug: "go", SortOrder: 1}
	if err := db.Create(cat).Error; err != nil {
		t.Fatal(err)
	}

	a := &model.Article{
		AuthorID: u.ID, CategoryID: cat.ID, Title: "t", Content: "c",
		Status: model.ArticleStatusDraft,
	}
	if err := r.Create(db, a); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByID(db, a.ID)
	if err != nil || got.Title != "t" {
		t.Fatalf("FindByID = %v, %v", got, err)
	}
	if err := r.SetArticleTags(db, a.ID, []uint{10, 11}); err != nil {
		t.Fatal(err)
	}
	ids, err := r.FindArticleTags(db, a.ID)
	if err != nil || len(ids) != 2 {
		t.Fatalf("FindArticleTags = %v, %v", ids, err)
	}
	now := time.Now()
	a.Status = model.ArticleStatusPublished
	a.PublishedAt = &now
	if err := r.Update(db, a); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, a.ID, "views", 1); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, a.ID, "likes_count", 1); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, a.ID)
	if got.Views != 1 || got.LikesCount != 1 {
		t.Fatalf("counts = views %d likes %d", got.Views, got.LikesCount)
	}
	if err := r.Delete(db, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, a.ID); err == nil {
		t.Fatal("soft-deleted article still found")
	}
	_ = ctx
}
