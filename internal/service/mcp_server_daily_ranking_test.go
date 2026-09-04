package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

func newMcpDailyEnv(t *testing.T) (*McpServerService, *ResourceCommentService, *ContentRankingService) {
	t.Helper()
	db, rankSvc, users := newResourceDailyDeps(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	mcpSvc := NewMcpServerService(repo.NewMcpServerRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), cfg, notifSvc, rankSvc)
	resCommentSvc := NewResourceCommentService(repo.NewResourceCommentRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db), repo.NewInteractionRepo(db), users, notifSvc, rankSvc)
	return mcpSvc, resCommentSvc, rankSvc
}

func TestMcpDailyViewLikeCommentArchive(t *testing.T) {
	svc, resCommentSvc, rankSvc := newMcpDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sv := seedMcpServer(t, env.db, env.author.ID, "M", model.ResourceStatusPublished, false)

	if _, err := svc.Get(ctx, env.viewer.ID, sv.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ToggleLike(ctx, env.viewer.ID, sv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := resCommentSvc.Create(ctx, env.viewer.ID, "mcp_server", sv.ID, "hi", nil); err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListMcpServers(ctx, 1, 10)
	if items[0].Score != 6 { // 1 + 2 + 3
		t.Fatalf("score=%d want 6", items[0].Score)
	}

	if _, err := svc.Archive(ctx, env.author.ID, sv.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListMcpServers(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("archive must remove: items=%+v total=%d", items, total)
	}
}

func TestMcpListSortHotWithoutDailyScore(t *testing.T) {
	svc, _, _ := newMcpDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	seedMcpServer(t, env.db, env.author.ID, "M1", model.ResourceStatusPublished, false)
	seedMcpServer(t, env.db, env.author.ID, "M2", model.ResourceStatusPublished, false)

	out, err := svc.List(ctx, McpServerListQuery{Page: 1, PageSize: 10, Sort: "hot"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.List) != 2 {
		t.Fatalf("sort=hot list len=%d want 2", len(out.List))
	}
}
