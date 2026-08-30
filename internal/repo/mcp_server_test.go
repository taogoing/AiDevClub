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
		Status: model.ResourceStatusDraft,
	}
	if err := r.Create(db, s); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByID(db, s.ID)
	if err != nil || got.Name != "test-mcp" || got.InstallationsJSON != "[]" {
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
	s.InstallationsJSON = ""
	if err := r.Update(db, s); err != nil {
		t.Fatal(err)
	}
	if s.InstallationsJSON != "[]" {
		t.Fatalf("empty installations were not normalized on update: %q", s.InstallationsJSON)
	}
	if err := r.IncrCount(db, s.ID, "views", 1); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, s.ID)
	if got.Views != 1 {
		t.Fatalf("views = %d", got.Views)
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
			Status: model.ResourceStatusPublished, PublishedAt: &now,
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
}

func TestMcpServerRepoPublicListWithAuthorIDHidesHidden(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewMcpServerRepo(db)
	ctx := context.Background()
	owner := &model.User{Email: "hidden-mcp-list@t.com", PasswordHash: "x", Nickname: "Owner"}
	if err := NewUserRepo(db).Create(owner); err != nil {
		t.Fatal(err)
	}
	hidden := &model.McpServer{
		AuthorID: owner.ID, Name: "hidden", Description: "content",
		Status: model.ResourceStatusPublished, Hidden: true,
	}
	if err := r.Create(nil, hidden); err != nil {
		t.Fatal(err)
	}

	authorID := owner.ID
	list, total, err := r.List(ctx, McpServerQuery{AuthorID: &authorID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("public list exposed hidden MCP server: total %d, list %+v", total, list)
	}
}

func TestMcpServerRepoListOwnedScopesAuthorIncludesHiddenAndRejectsUnknownStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewMcpServerRepo(db)
	ctx := context.Background()
	owner := &model.User{Email: "owned-mcp@t.com", PasswordHash: "x", Nickname: "Owner"}
	other := &model.User{Email: "other-mcp@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := NewUserRepo(db).Create(owner); err != nil {
		t.Fatal(err)
	}
	if err := NewUserRepo(db).Create(other); err != nil {
		t.Fatal(err)
	}
	draft := &model.McpServer{AuthorID: owner.ID, Name: "draft", Status: model.ResourceStatusDraft}
	hidden := &model.McpServer{AuthorID: owner.ID, Name: "hidden", Status: model.ResourceStatusPublished, Hidden: true}
	foreign := &model.McpServer{AuthorID: other.ID, Name: "foreign", Status: model.ResourceStatusRejected}
	deleted := &model.McpServer{AuthorID: owner.ID, Name: "deleted", Status: model.ResourceStatusArchived}
	for _, server := range []*model.McpServer{draft, hidden, foreign, deleted} {
		if err := r.Create(db, server); err != nil {
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
		t.Fatalf("owned MCP servers = total %d, list %+v", total, list)
	}
	seen := map[uint]bool{}
	for _, server := range list {
		seen[server.ID] = true
	}
	if !seen[draft.ID] || !seen[hidden.ID] || seen[foreign.ID] || seen[deleted.ID] {
		t.Fatalf("owned MCP server IDs = %+v", seen)
	}
	drafts, total, err := r.ListOwned(ctx, owner.ID, string(model.ResourceStatusDraft), 1, 20)
	if err != nil || total != 1 || len(drafts) != 1 || drafts[0].ID != draft.ID {
		t.Fatalf("draft MCP servers = total %d, list %+v, err %v", total, drafts, err)
	}
	if _, _, err := r.ListOwned(ctx, owner.ID, "invalid", 1, 20); err == nil {
		t.Fatal("unknown MCP server status accepted")
	}
}
