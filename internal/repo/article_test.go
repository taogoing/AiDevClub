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

func TestArticleRepoList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewArticleRepo(db)
	ctx := context.Background()

	users := NewUserRepo(db)
	_ = users.Create(&model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A"})
	cats := NewCategoryRepo(db)
	_ = cats.Seed(ctx)
	catList, _ := cats.List(ctx)

	now := time.Now()
	for i := 0; i < 3; i++ {
		a := &model.Article{
			AuthorID: 1, CategoryID: catList[i%len(catList)].ID,
			Title: "x", Content: "c",
			Status: model.ArticleStatusPublished, PublishedAt: &now,
		}
		if err := r.Create(db, a); err != nil {
			t.Fatal(err)
		}
	}
	_ = r.SetArticleTags(db, 1, []uint{1})

	q := ArticleQuery{Page: 1, PageSize: 2, Sort: "latest"}
	list, total, err := r.List(ctx, q)
	if err != nil || total != 3 || len(list) != 2 {
		t.Fatalf("list = %d total, %d len, err %v", total, len(list), err)
	}

	tagID := uint(1)
	q.TagID = &tagID
	list, total, _ = r.List(ctx, q)
	if total != 1 {
		t.Fatalf("tag filter total = %d", total)
	}
}

func TestArticleRepoPublicListWithAuthorIDHidesHidden(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewArticleRepo(db)
	ctx := context.Background()
	owner := &model.User{Email: "hidden-article-list@t.com", PasswordHash: "x", Nickname: "Owner"}
	if err := NewUserRepo(db).Create(owner); err != nil {
		t.Fatal(err)
	}
	category := &model.Category{Name: "Go", Slug: "go", SortOrder: 1}
	if err := db.Create(category).Error; err != nil {
		t.Fatal(err)
	}
	hidden := &model.Article{
		AuthorID: owner.ID, CategoryID: category.ID, Title: "hidden", Content: "content",
		Status: model.ArticleStatusPublished, Hidden: true,
	}
	if err := r.Create(nil, hidden); err != nil {
		t.Fatal(err)
	}

	authorID := owner.ID
	list, total, err := r.List(ctx, ArticleQuery{AuthorID: &authorID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("public list exposed hidden article: total %d, list %+v", total, list)
	}
}

func TestArticleRepoListOwnedScopesAuthorIncludesHiddenAndRejectsUnknownStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewArticleRepo(db)
	ctx := context.Background()
	owner := &model.User{Email: "owned-article@t.com", PasswordHash: "x", Nickname: "Owner"}
	other := &model.User{Email: "other-article@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := NewUserRepo(db).Create(owner); err != nil {
		t.Fatal(err)
	}
	if err := NewUserRepo(db).Create(other); err != nil {
		t.Fatal(err)
	}
	cat := &model.Category{Name: "Go", Slug: "go", SortOrder: 1}
	if err := db.Create(cat).Error; err != nil {
		t.Fatal(err)
	}
	draft := &model.Article{AuthorID: owner.ID, CategoryID: cat.ID, Title: "draft", Content: "content", Status: model.ArticleStatusDraft}
	hidden := &model.Article{AuthorID: owner.ID, CategoryID: cat.ID, Title: "hidden", Content: "content", Status: model.ArticleStatusPublished, Hidden: true}
	foreign := &model.Article{AuthorID: other.ID, CategoryID: cat.ID, Title: "foreign", Content: "content", Status: model.ArticleStatusDraft}
	deleted := &model.Article{AuthorID: owner.ID, CategoryID: cat.ID, Title: "deleted", Content: "content", Status: model.ArticleStatusDraft}
	for _, article := range []*model.Article{draft, hidden, foreign, deleted} {
		if err := r.Create(db, article); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Delete(db, deleted.ID); err != nil {
		t.Fatal(err)
	}

	list, total, err := r.ListOwned(ctx, owner.ID, "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("owned articles = total %d, list %+v", total, list)
	}
	seen := map[uint]bool{}
	for _, article := range list {
		seen[article.ID] = true
	}
	if !seen[draft.ID] || !seen[hidden.ID] || seen[foreign.ID] || seen[deleted.ID] {
		t.Fatalf("owned article IDs = %+v", seen)
	}
	drafts, total, err := r.ListOwned(ctx, owner.ID, string(model.ArticleStatusDraft), 1, 20)
	if err != nil || total != 1 || len(drafts) != 1 || drafts[0].ID != draft.ID {
		t.Fatalf("draft articles = total %d, list %+v, err %v", total, drafts, err)
	}
	if _, _, err := r.ListOwned(ctx, owner.ID, "invalid", 1, 20); err == nil {
		t.Fatal("unknown article status accepted")
	}
}
