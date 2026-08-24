package repo

import (
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestCommentRepo(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewCommentRepo(db)

	users := NewUserRepo(db)
	_ = users.Create(&model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A"})
	cats := NewCategoryRepo(db)
	_ = cats.Seed(t.Context())
	catList, _ := cats.List(t.Context())
	artRepo := NewArticleRepo(db)
	art := &model.Article{AuthorID: 1, CategoryID: catList[0].ID, Title: "t", Content: "c", Status: model.ArticleStatusPublished}
	_ = artRepo.Create(db, art)

	c := &model.Comment{ArticleID: art.ID, AuthorID: 1, Content: "hi"}
	if err := r.Create(db, c); err != nil {
		t.Fatal(err)
	}
	pid := c.ID
	r2 := &model.Comment{ArticleID: art.ID, AuthorID: 1, Content: "reply", ParentID: &pid}
	if err := r.Create(db, r2); err != nil {
		t.Fatal(err)
	}
	list, err := r.ListByArticle(db, art.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if err := r.IncrLikes(db, c.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(db, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, c.ID); err == nil {
		t.Fatal("deleted comment found")
	}
	list, _ = r.ListByArticle(db, art.ID)
	if len(list) != 1 {
		t.Fatalf("after delete len = %d", len(list))
	}
}
