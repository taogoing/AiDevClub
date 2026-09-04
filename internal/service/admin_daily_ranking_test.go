package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

func newAdminDailyEnv(t *testing.T) (*AdminService, *ReportService, *ContentRankingService) {
	t.Helper()
	db, rankSvc, users := newResourceDailyDeps(t)
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	adminLogs := NewAdminLogService(repo.NewAdminLogRepo(db), users)
	admin := NewAdminService(
		users, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db),
		repo.NewCommentRepo(db), repo.NewResourceCommentRepo(db), repo.NewReportRepo(db),
		repo.NewAnnouncementRepo(db), adminLogs, notifSvc, rankSvc,
	)
	reports := NewReportService(
		repo.NewReportRepo(db), repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db),
		repo.NewCommentRepo(db), repo.NewResourceCommentRepo(db), admin, adminLogs, notifSvc, rankSvc,
	)
	return admin, reports, rankSvc
}

func TestAdminHideArticleRemovesFromDaily(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	a := seedArticle(t, env.db, env.author.ID, "A", model.ArticleStatusPublished, false)
	_ = rankSvc.AddScore(ctx, RankedContentArticle, a.ID, 4)

	if err := admin.HideArticle(ctx, env.author.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("hide must remove: items=%+v total=%d", items, total)
	}
}

func TestAdminHideContentGenericRemoves(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)
	_ = rankSvc.AddScore(ctx, RankedContentSkill, sk.ID, 4)

	if err := admin.HideContent("skill", sk.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("generic hide must remove: items=%+v total=%d", items, total)
	}
}

func TestAdminHideCommentDeducts(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	a := seedArticle(t, env.db, env.author.ID, "A", model.ArticleStatusPublished, false)
	c := &model.Comment{ArticleID: a.ID, AuthorID: env.viewer.ID, Content: "x"}
	if err := env.db.Create(c).Error; err != nil {
		t.Fatal(err)
	}
	_ = rankSvc.AddScore(ctx, RankedContentArticle, a.ID, 3)

	if err := admin.HideComment(ctx, env.author.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("hide comment must deduct to removal: items=%+v total=%d", items, total)
	}
}

func TestAdminHideResourceCommentDeducts(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)
	rc := &model.ResourceComment{ResourceType: "skill", ResourceID: sk.ID, AuthorID: env.viewer.ID, Content: "x"}
	if err := env.db.Create(rc).Error; err != nil {
		t.Fatal(err)
	}
	_ = rankSvc.AddScore(ctx, RankedContentSkill, sk.ID, 3)

	if err := admin.HideResourceComment(ctx, env.author.ID, rc.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("hide resource comment must deduct: items=%+v total=%d", items, total)
	}
}

func TestReportResolveHideRemoves(t *testing.T) {
	_, reports, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	a := seedArticle(t, env.db, env.author.ID, "A", model.ArticleStatusPublished, false)
	_ = rankSvc.AddScore(ctx, RankedContentArticle, a.ID, 4)
	r := &model.Report{ReporterID: env.viewer.ID, TargetType: "article", TargetID: a.ID, Status: model.ReportStatusPending}
	if err := env.db.Create(r).Error; err != nil {
		t.Fatal(err)
	}

	if err := reports.Resolve(ctx, env.author.ID, r.ID, "hide", "违规"); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("report hide must remove: items=%+v total=%d", items, total)
	}
}
