package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newSkillTestEnv(t *testing.T) (*SkillService, *model.User) {
	t.Helper()
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	u := &model.User{Email: "sk@t.com", PasswordHash: "x", Nickname: "SK", AvatarURL: "/x.png"}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}
	cfg := &platform.Config{
		DefaultPageSize: 20,
		MaxPageSize:     50,
		HotCacheTTL:     60e9,
	}
	svc := NewSkillService(
		repo.NewSkillRepo(db),
		repo.NewTagRepo(db),
		repo.NewInteractionRepo(db),
		testutil.NewTestRedis(t),
		cfg,
	)
	return svc, u
}

func TestSkillCreate(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, err := svc.Create(ctx, u.ID, CreateSkillInput{
		Name: "my-skill", Description: "desc", RepoURL: "https://github.com/x",
		TagNames: []string{"go", "mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sk.ID == 0 || sk.Status != model.ResourceStatusDraft {
		t.Fatalf("skill = %+v", sk)
	}
	db := svc.skills.DB()
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
	tagIDs, err := svc.skills.FindSkillTags(db, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tagIDs) != 2 {
		t.Fatalf("skill tags = %d, want 2", len(tagIDs))
	}
}

func TestSkillUpdate(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, _ := svc.Create(ctx, u.ID, CreateSkillInput{
		Name: "s1", TagNames: []string{"go"},
	})

	got, err := svc.Update(ctx, u.ID, sk.ID, CreateSkillInput{
		Name: "s1-updated", Description: "new desc", TagNames: []string{"rust"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "s1-updated" {
		t.Fatalf("name = %q", got.Name)
	}
	db := svc.skills.DB()
	tagRepo := repo.NewTagRepo(db)
	goTag, _ := tagRepo.FindByName(ctx, "go")
	if goTag.UsageCount != 0 {
		t.Fatalf("go tag usage = %d, want 0", goTag.UsageCount)
	}
	rustTag, _ := tagRepo.FindByName(ctx, "rust")
	if rustTag.UsageCount != 1 {
		t.Fatalf("rust tag usage = %d, want 1", rustTag.UsageCount)
	}

	if _, err := svc.Update(ctx, u.ID+999, sk.ID, CreateSkillInput{Name: "x"}); err == nil {
		t.Fatal("non-author update allowed")
	}

	sk2, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "s2"})
	sk2.Status = model.ResourceStatusPendingReview
	_ = svc.skills.Update(nil, sk2)
	if _, err := svc.Update(ctx, u.ID, sk2.ID, CreateSkillInput{Name: "s2-edit"}); err == nil {
		t.Fatal("pending_review skill should not be editable")
	}
}

func TestSkillDelete(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, _ := svc.Create(ctx, u.ID, CreateSkillInput{
		Name: "del-me", TagNames: []string{"tmp"},
	})
	db := svc.skills.DB()
	tagRepo := repo.NewTagRepo(db)
	tmpTag, _ := tagRepo.FindByName(ctx, "tmp")
	if tmpTag.UsageCount != 1 {
		t.Fatalf("before delete usage = %d", tmpTag.UsageCount)
	}

	if err := svc.Delete(ctx, u.ID+999, sk.ID); err == nil {
		t.Fatal("non-author delete allowed")
	}
	if err := svc.Delete(ctx, u.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	tmpTag2, _ := tagRepo.FindByName(ctx, "tmp")
	if tmpTag2.UsageCount != 0 {
		t.Fatalf("after delete usage = %d", tmpTag2.UsageCount)
	}
	if _, err := svc.skills.FindByID(nil, sk.ID); err == nil {
		t.Fatal("deleted skill still found")
	}
}

func TestSkillStatusFlow(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "flow"})

	if sk.Status != model.ResourceStatusDraft {
		t.Fatalf("initial status = %s", sk.Status)
	}

	submitted, err := svc.Submit(ctx, u.ID, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if submitted.Status != model.ResourceStatusPendingReview {
		t.Fatalf("after submit status = %s", submitted.Status)
	}

	if _, err := svc.Submit(ctx, u.ID, sk.ID); err == nil {
		t.Fatal("double submit should fail")
	}

	withdrawn, err := svc.Withdraw(ctx, u.ID, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if withdrawn.Status != model.ResourceStatusDraft {
		t.Fatalf("after withdraw status = %s", withdrawn.Status)
	}

	if _, err := svc.Withdraw(ctx, u.ID, sk.ID); err == nil {
		t.Fatal("withdraw from draft should fail")
	}

	if _, err := svc.Archive(ctx, u.ID, sk.ID); err == nil {
		t.Fatal("archive from draft should fail")
	}

	submitted2, _ := svc.Submit(ctx, u.ID, sk.ID)
	if submitted2.Status != model.ResourceStatusPendingReview {
		t.Fatalf("re-submit status = %s", submitted2.Status)
	}
}

func TestSkillVisibility(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	pub, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "pub"})
	pub.Status = model.ResourceStatusPublished
	now := pub.CreatedAt
	pub.PublishedAt = &now
	_ = svc.skills.Update(nil, pub)

	draft, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "draft"})

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
	_ = repo.NewUserRepo(svc.skills.DB()).Create(other)
	if _, err := svc.Get(ctx, other.ID, draft.ID); err == nil {
		t.Fatal("other user should not see draft")
	}
}

func TestSkillList(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()

	sk1, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "alpha", TagNames: []string{"go"}})
	sk1.Status = model.ResourceStatusPublished
	now := sk1.CreatedAt
	sk1.PublishedAt = &now
	_ = svc.skills.Update(nil, sk1)

	sk2, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "beta", TagNames: []string{"rust"}})
	sk2.Status = model.ResourceStatusPublished
	now2 := sk2.CreatedAt
	sk2.PublishedAt = &now2
	_ = svc.skills.Update(nil, sk2)

	_, _ = svc.Create(ctx, u.ID, CreateSkillInput{Name: "draft-skill"})

	res, err := svc.List(ctx, SkillListQuery{Page: 1, PageSize: 20, Sort: "latest"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("total = %d, want 2 (only published)", res.Total)
	}

	goTag, _ := repo.NewTagRepo(svc.skills.DB()).FindByName(ctx, "go")
	res2, err := svc.List(ctx, SkillListQuery{Page: 1, PageSize: 20, TagID: &goTag.ID})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Total != 1 || res2.List[0].Name != "alpha" {
		t.Fatalf("tag filter = %+v", res2)
	}

	res3, err := svc.List(ctx, SkillListQuery{Page: 1, PageSize: 1, Sort: "latest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res3.List) != 1 || res3.Total != 2 {
		t.Fatalf("pagination = %+v", res3)
	}

	res4, err := svc.List(ctx, SkillListQuery{Page: 1, PageSize: 20, Keyword: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if res4.Total != 1 {
		t.Fatalf("keyword total = %d", res4.Total)
	}
}
