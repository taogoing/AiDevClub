package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

func newSkillDailyEnv(t *testing.T) (*SkillService, *ResourceCommentService, *ContentRankingService) {
	t.Helper()
	db, rankSvc, users := newResourceDailyDeps(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	skillSvc := NewSkillService(repo.NewSkillRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), cfg, notifSvc, rankSvc)
	resCommentSvc := NewResourceCommentService(repo.NewResourceCommentRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db), repo.NewInteractionRepo(db), users, notifSvc, rankSvc)
	return skillSvc, resCommentSvc, rankSvc
}

func TestSkillDailyViewAuthorGuest(t *testing.T) {
	svc, _, rankSvc := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	author, viewer := env.author, env.viewer

	sk := seedSkill(t, env.db, author.ID, "S", model.ResourceStatusPublished, false)
	if _, err := svc.Get(ctx, viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, author.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, 0, sk.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 1 || len(items) != 1 || items[0].Score != 1 {
		t.Fatalf("items=%+v total=%d", items, total)
	}
}

func TestSkillDailyToggleLikeFavoriteAndResourceComment(t *testing.T) {
	svc, resCommentSvc, rankSvc := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)

	if _, _, err := svc.ToggleLike(ctx, env.viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ToggleFavorite(ctx, env.viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	c, err := resCommentSvc.Create(ctx, env.viewer.ID, "skill", sk.ID, "hi", nil)
	if err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListSkills(ctx, 1, 10)
	if items[0].Score != 7 { // 2 + 2 + 3
		t.Fatalf("score=%d want 7", items[0].Score)
	}

	if _, _, err := svc.ToggleLike(ctx, env.viewer.ID, sk.ID); err != nil { // 取消点赞 -> 5
		t.Fatal(err)
	}
	if err := resCommentSvc.Delete(ctx, env.viewer.ID, c.ID); err != nil { // 删评论 -> 2
		t.Fatal(err)
	}
	items, _, _ = rankSvc.ListSkills(ctx, 1, 10)
	if items[0].Score != 2 {
		t.Fatalf("score=%d want 2", items[0].Score)
	}
}

func TestSkillDailyArchiveRemoves(t *testing.T) {
	svc, _, rankSvc := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)
	if _, err := svc.Get(ctx, env.viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Archive(ctx, env.author.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("archive must remove: items=%+v total=%d", items, total)
	}
}

func TestSkillListSortHotWithoutDailyScore(t *testing.T) {
	// 移除旧委托后：即使日榜为空，sort=hot 也必须走 MySQL 返回已发布内容
	svc, _, _ := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	seedSkill(t, env.db, env.author.ID, "S1", model.ResourceStatusPublished, false)
	seedSkill(t, env.db, env.author.ID, "S2", model.ResourceStatusPublished, false)

	out, err := svc.List(ctx, SkillListQuery{Page: 1, PageSize: 10, Sort: "hot"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.List) != 2 {
		t.Fatalf("sort=hot list len=%d want 2", len(out.List))
	}
}
