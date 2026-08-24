package repo

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestMcpServerRepoCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewMcpServerRepo(db)
	ctx := context.Background()

	u := &model.User{Email: "m@m.com", PasswordHash: "x", Nickname: "M", AvatarURL: "/x.png"}
	if err := NewUserRepo(db).Create(u); err != nil {
		t.Fatal(err)
	}

	s := &model.McpServer{
		AuthorID: u.ID, Name: "test-mcp", Description: "desc",
		ToolsJSON: "[]", Status: model.ResourceStatusDraft,
	}
	if err := r.Create(db, s); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByID(db, s.ID)
	if err != nil || got.Name != "test-mcp" {
		t.Fatalf("FindByID = %v, %v", got, err)
	}
	if err := r.SetMcpServerTags(db, s.ID, []uint{10, 11}); err != nil {
		t.Fatal(err)
	}
	ids, err := r.FindMcpServerTags(db, s.ID)
	if err != nil || len(ids) != 2 {
		t.Fatalf("FindMcpServerTags = %v, %v", ids, err)
	}
	now := time.Now()
	s.Status = model.ResourceStatusPublished
	s.PublishedAt = &now
	if err := r.Update(db, s); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, s.ID, "views", 1); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, s.ID, "downloads", 3); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, s.ID)
	if got.Views != 1 || got.Downloads != 3 {
		t.Fatalf("counts = views %d downloads %d", got.Views, got.Downloads)
	}
	if err := r.IncrViews(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, s.ID)
	if got.Views != 2 {
		t.Fatalf("IncrViews views = %d", got.Views)
	}
	if err := r.Delete(db, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, s.ID); err == nil {
		t.Fatal("soft-deleted mcp server still found")
	}
}

func TestMcpServerRepoList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewMcpServerRepo(db)
	ctx := context.Background()

	_ = NewUserRepo(db).Create(&model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A"})

	now := time.Now()
	for i := 0; i < 3; i++ {
		s := &model.McpServer{
			AuthorID: 1, Name: "mcp", Description: "d",
			ToolsJSON: "[]", Status: model.ResourceStatusPublished, PublishedAt: &now,
		}
		if err := r.Create(db, s); err != nil {
			t.Fatal(err)
		}
	}
	_ = r.SetMcpServerTags(db, 1, []uint{1})

	q := McpServerQuery{Page: 1, PageSize: 2, Sort: "latest"}
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

	q.Sort = "hot"
	q.TagID = nil
	list, total, _ = r.List(ctx, q)
	if total != 3 || len(list) != 2 {
		t.Fatalf("hot sort = %d total, %d len", total, len(list))
	}

	q.Sort = "downloads"
	list, total, _ = r.List(ctx, q)
	if total != 3 || len(list) != 2 {
		t.Fatalf("downloads sort = %d total, %d len", total, len(list))
	}
}
