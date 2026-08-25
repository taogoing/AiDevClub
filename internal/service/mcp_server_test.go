package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newMcpServerTestEnv(t *testing.T) (*McpServerService, *model.User) {
	t.Helper()
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	u := &model.User{Email: "mc@t.com", PasswordHash: "x", Nickname: "MC", AvatarURL: "/x.png"}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}
	cfg := &platform.Config{
		DefaultPageSize: 20,
		MaxPageSize:     50,
		HotCacheTTL:     60e9,
	}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	svc := NewMcpServerService(
		repo.NewMcpServerRepo(db),
		repo.NewTagRepo(db),
		repo.NewInteractionRepo(db),
		testutil.NewTestRedis(t),
		cfg,
		notifSvc,
	)
	return svc, u
}

func TestMcpServerCreate(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	sv, err := svc.Create(ctx, u.ID, CreateMcpServerInput{
		Name: "my-mcp", Description: "desc", RepoURL: "https://github.com/x",
		ToolsJSON: `{"tools":[]}`, Readme: "# Readme",
		TagNames: []string{"go", "mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sv.ID == 0 || sv.Status != model.ResourceStatusDraft {
		t.Fatalf("server = %+v", sv)
	}
	db := svc.servers.DB()
	tagRepo := repo.NewTagRepo(db)
	for _, name := range []string{"go", "mcp"} {
		tg, err := tagRepo.FindByName(ctx, name)
		if err != nil {
			t.Fatalf("tag %q not found: %v", name, err)
		}
		if tg.UsageCount != 1 {
			t.Fatalf("tag %q usage_count = %d, want 1", name, tg.UsageCount)
		}
	}
	tagIDs, err := svc.servers.FindMcpServerTags(db, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tagIDs) != 2 {
		t.Fatalf("server tags = %d, want 2", len(tagIDs))
	}
}

func TestMcpServerStatusFlow(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	sv, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "flow"})

	if sv.Status != model.ResourceStatusDraft {
		t.Fatalf("initial status = %s", sv.Status)
	}

	submitted, err := svc.Submit(ctx, u.ID, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if submitted.Status != model.ResourceStatusPendingReview {
		t.Fatalf("after submit status = %s", submitted.Status)
	}

	if _, err := svc.Submit(ctx, u.ID, sv.ID); err == nil {
		t.Fatal("double submit should fail")
	}

	withdrawn, err := svc.Withdraw(ctx, u.ID, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if withdrawn.Status != model.ResourceStatusDraft {
		t.Fatalf("after withdraw status = %s", withdrawn.Status)
	}

	if _, err := svc.Withdraw(ctx, u.ID, sv.ID); err == nil {
		t.Fatal("withdraw from draft should fail")
	}

	if _, err := svc.Archive(ctx, u.ID, sv.ID); err == nil {
		t.Fatal("archive from draft should fail")
	}

	submitted2, _ := svc.Submit(ctx, u.ID, sv.ID)
	if submitted2.Status != model.ResourceStatusPendingReview {
		t.Fatalf("re-submit status = %s", submitted2.Status)
	}
}

func TestMcpServerVisibility(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	pub, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "pub"})
	pub.Status = model.ResourceStatusPublished
	now := pub.CreatedAt
	pub.PublishedAt = &now
	_ = svc.servers.Update(nil, pub)

	draft, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "draft"})

	detail, err := svc.Get(ctx, 0, pub.ID)
	if err != nil {
		t.Fatalf("guest should see published: %v", err)
	}
	if detail.Name != "pub" {
		t.Fatalf("detail name = %q", detail.Name)
	}

	if _, err := svc.Get(ctx, 0, draft.ID); err == nil {
		t.Fatal("guest should not see draft")
	}
	if _, err := svc.Get(ctx, u.ID, draft.ID); err != nil {
		t.Fatalf("author should see own draft: %v", err)
	}

	other := &model.User{Email: "o@t.com", PasswordHash: "x", Nickname: "O", AvatarURL: "/o.png"}
	_ = repo.NewUserRepo(svc.servers.DB()).Create(other)
	if _, err := svc.Get(ctx, other.ID, draft.ID); err == nil {
		t.Fatal("other user should not see draft")
	}
}

func TestMcpServerGetLoadsInteractionState(t *testing.T) {
	svc, user := newMcpServerTestEnv(t)
	ctx := context.Background()
	server, err := svc.Create(ctx, user.ID, CreateMcpServerInput{Name: "interactions"})
	if err != nil {
		t.Fatal(err)
	}
	server.Status = model.ResourceStatusPublished
	if err := svc.servers.Update(nil, server); err != nil {
		t.Fatal(err)
	}
	db := svc.servers.DB()
	if err := db.Create(&model.McpServerLike{McpServerID: server.ID, UserID: user.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.McpServerFavorite{McpServerID: server.ID, UserID: user.ID}).Error; err != nil {
		t.Fatal(err)
	}

	detail, err := svc.Get(ctx, user.ID, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Liked || !detail.Favorited {
		t.Fatalf("Get interaction state = liked %v favorited %v", detail.Liked, detail.Favorited)
	}
}

func TestMcpServerGetRejectsHiddenForNonOwnersWithoutIncrementingViews(t *testing.T) {
	svc, owner := newMcpServerTestEnv(t)
	ctx := context.Background()
	other := &model.User{Email: "mcp-hidden-other@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := repo.NewUserRepo(svc.servers.DB()).Create(other); err != nil {
		t.Fatal(err)
	}
	hidden, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "hidden-get"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	hidden.Views = 7
	if err := svc.servers.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name   string
		userID uint
	}{
		{name: "anonymous", userID: 0},
		{name: "non-owner", userID: other.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Get(ctx, tc.userID, hidden.ID); err != ErrMcpServerNotFound {
				t.Errorf("Get hidden MCP server error = %v, want %v", err, ErrMcpServerNotFound)
			}
		})
	}
	persisted, err := svc.servers.FindByID(nil, hidden.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Views != 7 {
		t.Errorf("views after rejected reads = %d, want 7", persisted.Views)
	}
	if _, err := svc.Get(ctx, owner.ID, hidden.ID); err != nil {
		t.Fatalf("owner cannot get hidden MCP server: %v", err)
	}
}

func TestMcpServerList(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()

	sv1, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "alpha", TagNames: []string{"go"}})
	sv1.Status = model.ResourceStatusPublished
	now := sv1.CreatedAt
	sv1.PublishedAt = &now
	_ = svc.servers.Update(nil, sv1)

	sv2, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "beta", TagNames: []string{"rust"}})
	sv2.Status = model.ResourceStatusPublished
	now2 := sv2.CreatedAt
	sv2.PublishedAt = &now2
	_ = svc.servers.Update(nil, sv2)

	_, _ = svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "draft-mcp"})

	res, err := svc.List(ctx, McpServerListQuery{Page: 1, PageSize: 20, Sort: "latest"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("total = %d, want 2 (only published)", res.Total)
	}

	goTag, _ := repo.NewTagRepo(svc.servers.DB()).FindByName(ctx, "go")
	res2, err := svc.List(ctx, McpServerListQuery{Page: 1, PageSize: 20, TagID: &goTag.ID})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Total != 1 || res2.List[0].Name != "alpha" {
		t.Fatalf("tag filter = %+v", res2)
	}

	res3, err := svc.List(ctx, McpServerListQuery{Page: 1, PageSize: 1, Sort: "latest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res3.List) != 1 || res3.Total != 2 {
		t.Fatalf("pagination = %+v", res3)
	}

	res4, err := svc.List(ctx, McpServerListQuery{Page: 1, PageSize: 20, Keyword: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if res4.Total != 1 {
		t.Fatalf("keyword total = %d", res4.Total)
	}
}

func TestMcpServerPublicListWithAuthorIDHidesHidden(t *testing.T) {
	svc, owner := newMcpServerTestEnv(t)
	ctx := context.Background()
	hidden, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "hidden"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	if err := svc.servers.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}

	authorID := owner.ID
	result, err := svc.List(ctx, McpServerListQuery{AuthorID: &authorID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 || len(result.List) != 0 {
		t.Fatalf("public list exposed hidden MCP server: %+v", result)
	}
}

func TestMcpServerUploadZip(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	sv, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "zip-mcp"})
	sv.Status = model.ResourceStatusPublished
	now := sv.CreatedAt
	sv.PublishedAt = &now
	_ = svc.servers.Update(nil, sv)

	if err := svc.UploadZip(ctx, u.ID, sv.ID, "/static/mcp-servers/abc.zip", "abc.zip", 1024); err != nil {
		t.Fatal(err)
	}
	updated, _ := svc.servers.FindByID(nil, sv.ID)
	if updated.Status != model.ResourceStatusPendingReview {
		t.Fatalf("status = %s, want pending_review", updated.Status)
	}
	if updated.ZipURL != "/static/mcp-servers/abc.zip" {
		t.Fatalf("zip_url = %q", updated.ZipURL)
	}

	if err := svc.UploadZip(ctx, u.ID+999, sv.ID, "/x.zip", "x.zip", 1); err == nil {
		t.Fatal("non-author upload allowed")
	}

	sv2, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "pending"})
	sv2.Status = model.ResourceStatusPendingReview
	_ = svc.servers.Update(nil, sv2)
	if err := svc.UploadZip(ctx, u.ID, sv2.ID, "/x.zip", "x.zip", 1); err == nil {
		t.Fatal("pending_review upload should fail")
	}
}

func TestMcpServerDownload(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	sv, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "dl-mcp"})
	sv.Status = model.ResourceStatusPublished
	sv.ZipURL = "/static/mcp-servers/dl.zip"
	now := sv.CreatedAt
	sv.PublishedAt = &now
	_ = svc.servers.Update(nil, sv)

	url, err := svc.Download(ctx, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if url != "/static/mcp-servers/dl.zip" {
		t.Fatalf("url = %q", url)
	}
	dl, _ := svc.servers.FindByID(nil, sv.ID)
	if dl.Downloads != 1 {
		t.Fatalf("downloads = %d, want 1", dl.Downloads)
	}

	sv2, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "no-zip"})
	sv2.Status = model.ResourceStatusPublished
	now2 := sv2.CreatedAt
	sv2.PublishedAt = &now2
	_ = svc.servers.Update(nil, sv2)
	if _, err := svc.Download(ctx, sv2.ID); err == nil {
		t.Fatal("download with empty zip_url should fail")
	}

	hidden, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "hidden-download"})
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	hidden.ZipURL = "/static/mcp-servers/hidden.zip"
	hidden.Downloads = 7
	if err := svc.servers.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Download(ctx, hidden.ID); err != ErrMcpServerNotFound {
		t.Errorf("hidden download error = %v, want %v", err, ErrMcpServerNotFound)
	}
	persistedHidden, err := svc.servers.FindByID(nil, hidden.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedHidden.Downloads != 7 {
		t.Errorf("hidden downloads = %d, want 7", persistedHidden.Downloads)
	}

	if _, err := svc.Download(ctx, 99999); err == nil {
		t.Fatal("download non-existent should fail")
	}
}

func TestMcpServerToggleLike(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	sv, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "like-mcp"})
	sv.Status = model.ResourceStatusPublished
	now := sv.CreatedAt
	sv.PublishedAt = &now
	_ = svc.servers.Update(nil, sv)

	liked, count, err := svc.ToggleLike(ctx, u.ID, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !liked || count != 1 {
		t.Fatalf("liked=%v count=%d", liked, count)
	}
	liked, count, err = svc.ToggleLike(ctx, u.ID, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if liked || count != 0 {
		t.Fatalf("unliked=%v count=%d", liked, count)
	}

	draft, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "draft-like"})
	if _, _, err := svc.ToggleLike(ctx, u.ID, draft.ID); err == nil {
		t.Fatal("like on draft should fail")
	}
}

func TestMcpServerToggleFavorite(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	sv, _ := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "fav-mcp"})
	sv.Status = model.ResourceStatusPublished
	now := sv.CreatedAt
	sv.PublishedAt = &now
	_ = svc.servers.Update(nil, sv)

	favorited, count, err := svc.ToggleFavorite(ctx, u.ID, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !favorited || count != 1 {
		t.Fatalf("favorited=%v count=%d", favorited, count)
	}
	favorited, count, err = svc.ToggleFavorite(ctx, u.ID, sv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if favorited || count != 0 {
		t.Fatalf("unfavorited=%v count=%d", favorited, count)
	}
}

func TestMcpServerReadDoesNotIncrementViewsOrLoadInteractions(t *testing.T) {
	svc, u := newMcpServerTestEnv(t)
	ctx := context.Background()
	server, err := svc.Create(ctx, u.ID, CreateMcpServerInput{Name: "read-only"})
	if err != nil {
		t.Fatal(err)
	}
	server.Status = model.ResourceStatusPublished
	if err := svc.servers.Update(nil, server); err != nil {
		t.Fatal(err)
	}
	db := svc.servers.DB()
	if err := db.Create(&model.McpServerLike{McpServerID: server.ID, UserID: u.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.McpServerFavorite{McpServerID: server.ID, UserID: u.ID}).Error; err != nil {
		t.Fatal(err)
	}

	beforeViews := server.Views
	beforeDownloads := server.Downloads
	detail, err := svc.Read(ctx, u.ID, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Views != beforeViews || detail.Downloads != beforeDownloads {
		t.Fatalf("detail counts = views %d downloads %d, want views %d downloads %d", detail.Views, detail.Downloads, beforeViews, beforeDownloads)
	}
	if detail.Liked || detail.Favorited {
		t.Fatalf("read returned interaction state: %+v", detail)
	}
	var persisted model.McpServer
	if err := db.First(&persisted, server.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Views != beforeViews || persisted.Downloads != beforeDownloads {
		t.Fatalf("persisted counts = views %d downloads %d, want views %d downloads %d", persisted.Views, persisted.Downloads, beforeViews, beforeDownloads)
	}
}

func TestMcpServerReadRestrictsHiddenAndUnpublishedToOwner(t *testing.T) {
	svc, owner := newMcpServerTestEnv(t)
	ctx := context.Background()
	other := &model.User{Email: "mcp-other@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := repo.NewUserRepo(svc.servers.DB()).Create(other); err != nil {
		t.Fatal(err)
	}
	hidden, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "hidden"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	if err := svc.servers.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}
	draft, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "draft"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Read(ctx, owner.ID, hidden.ID); err != nil {
		t.Fatalf("owner cannot read hidden MCP server: %v", err)
	}
	if _, err := svc.Read(ctx, other.ID, hidden.ID); err == nil {
		t.Fatal("other actor read hidden MCP server")
	}
	if _, err := svc.Read(ctx, owner.ID, draft.ID); err != nil {
		t.Fatalf("owner cannot read draft MCP server: %v", err)
	}
	if _, err := svc.Read(ctx, other.ID, draft.ID); err == nil {
		t.Fatal("other actor read draft MCP server")
	}
}

func TestMcpServerListOwnedIncludesHiddenExcludesDeletedAndValidatesStatus(t *testing.T) {
	svc, owner := newMcpServerTestEnv(t)
	ctx := context.Background()
	other := &model.User{Email: "mcp-list-other@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := repo.NewUserRepo(svc.servers.DB()).Create(other); err != nil {
		t.Fatal(err)
	}
	draft, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "rejected"})
	if err != nil {
		t.Fatal(err)
	}
	rejected.Status = model.ResourceStatusRejected
	if err := svc.servers.Update(nil, rejected); err != nil {
		t.Fatal(err)
	}
	hidden, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "hidden"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	if err := svc.servers.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, other.ID, CreateMcpServerInput{Name: "other"}); err != nil {
		t.Fatal(err)
	}
	deleted, err := svc.Create(ctx, owner.ID, CreateMcpServerInput{Name: "deleted"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.servers.Delete(nil, deleted.ID); err != nil {
		t.Fatal(err)
	}

	got, err := svc.ListOwned(ctx, owner.ID, "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 3 || len(got.List) != 3 {
		t.Fatalf("owned MCP servers = %+v", got)
	}
	seen := map[uint]string{}
	for _, summary := range got.List {
		seen[summary.ID] = summary.Status
	}
	if seen[draft.ID] != string(model.ResourceStatusDraft) || seen[rejected.ID] != string(model.ResourceStatusRejected) || seen[hidden.ID] != string(model.ResourceStatusPublished) {
		t.Fatalf("owned MCP server statuses = %+v", seen)
	}

	rejectedOnly, err := svc.ListOwned(ctx, owner.ID, string(model.ResourceStatusRejected), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if rejectedOnly.Total != 1 || len(rejectedOnly.List) != 1 || rejectedOnly.List[0].ID != rejected.ID {
		t.Fatalf("rejected MCP servers = %+v", rejectedOnly)
	}
	if _, err := svc.ListOwned(ctx, owner.ID, "invalid", 1, 20); err == nil {
		t.Fatal("unknown MCP server status accepted")
	}
}
