package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		SkillZipDir:     t.TempDir(),
	}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	svc := NewSkillService(
		repo.NewSkillRepo(db),
		repo.NewTagRepo(db),
		repo.NewInteractionRepo(db),
		testutil.NewTestRedis(t),
		cfg,
		notifSvc,
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

func TestSkillGetLoadsInteractionState(t *testing.T) {
	svc, user := newSkillTestEnv(t)
	ctx := context.Background()
	skill, err := svc.Create(ctx, user.ID, CreateSkillInput{Name: "interactions"})
	if err != nil {
		t.Fatal(err)
	}
	skill.Status = model.ResourceStatusPublished
	if err := svc.skills.Update(nil, skill); err != nil {
		t.Fatal(err)
	}
	db := svc.skills.DB()
	if err := db.Create(&model.SkillLike{SkillID: skill.ID, UserID: user.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SkillFavorite{SkillID: skill.ID, UserID: user.ID}).Error; err != nil {
		t.Fatal(err)
	}

	detail, err := svc.Get(ctx, user.ID, skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Liked || !detail.Favorited {
		t.Fatalf("Get interaction state = liked %v favorited %v", detail.Liked, detail.Favorited)
	}
}

func TestSkillGetRejectsHiddenForNonOwnersWithoutIncrementingViews(t *testing.T) {
	svc, owner := newSkillTestEnv(t)
	ctx := context.Background()
	other := &model.User{Email: "skill-hidden-other@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := repo.NewUserRepo(svc.skills.DB()).Create(other); err != nil {
		t.Fatal(err)
	}
	hidden, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "hidden-get"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	hidden.Views = 7
	if err := svc.skills.Update(nil, hidden); err != nil {
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
			if _, err := svc.Get(ctx, tc.userID, hidden.ID); err != ErrSkillNotFound {
				t.Errorf("Get hidden skill error = %v, want %v", err, ErrSkillNotFound)
			}
		})
	}
	persisted, err := svc.skills.FindByID(nil, hidden.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Views != 7 {
		t.Errorf("views after rejected reads = %d, want 7", persisted.Views)
	}
	if _, err := svc.Get(ctx, owner.ID, hidden.ID); err != nil {
		t.Fatalf("owner cannot get hidden skill: %v", err)
	}
}

func TestSkillUploadZip(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "zip-skill"})
	sk.Status = model.ResourceStatusPublished
	now := sk.CreatedAt
	sk.PublishedAt = &now
	_ = svc.skills.Update(nil, sk)

	zipData := makeSkillZip(t, zipFixture{name: "SKILL.md", content: "# ABC"})
	zipPath := filepath.Join(svc.ZipDir(), "abc.zip")
	if err := os.WriteFile(zipPath, zipData, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.UploadZip(ctx, u.ID, sk.ID, "/static/skills/abc.zip", "abc.zip", int64(len(zipData))); err != nil {
		t.Fatal(err)
	}
	updated, _ := svc.skills.FindByID(nil, sk.ID)
	if updated.Status != model.ResourceStatusPendingReview {
		t.Fatalf("status = %s, want pending_review", updated.Status)
	}
	if updated.ZipURL != "/static/skills/abc.zip" {
		t.Fatalf("zip_url = %q", updated.ZipURL)
	}
	if updated.SkillMD != "# ABC" {
		t.Fatalf("skill_md = %q", updated.SkillMD)
	}

	if err := svc.UploadZip(ctx, u.ID+999, sk.ID, "/x.zip", "x.zip", 1); err == nil {
		t.Fatal("non-author upload allowed")
	}

	sk2, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "pending"})
	sk2.Status = model.ResourceStatusPendingReview
	_ = svc.skills.Update(nil, sk2)
	if err := svc.UploadZip(ctx, u.ID, sk2.ID, "/x.zip", "x.zip", 1); err == nil {
		t.Fatal("pending_review upload should fail")
	}
}

func TestSkillUploadZipRejectsInvalidReplacementWithoutChangingStoredMetadata(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, err := svc.Create(ctx, u.ID, CreateSkillInput{Name: "zip-replacement"})
	if err != nil {
		t.Fatal(err)
	}
	sk.ZipURL = "/static/skills/old.zip"
	sk.ZipFilename = "old.zip"
	sk.FileSize = 10
	sk.SkillMD = "# Old"
	if err := svc.skills.Update(nil, sk); err != nil {
		t.Fatal(err)
	}

	newPath := filepath.Join(svc.ZipDir(), "invalid.zip")
	if err := os.WriteFile(newPath, makeSkillZip(t, zipFixture{name: "README.md", content: "missing"}), 0o600); err != nil {
		t.Fatal(err)
	}

	err = svc.UploadZip(ctx, u.ID, sk.ID, "/static/skills/invalid.zip", "invalid.zip", 1)
	if !errors.Is(err, platform.ErrInvalidInput) {
		t.Fatalf("error = %v, want platform.ErrInvalidInput", err)
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf("invalid replacement still exists: %v", err)
	}
	updated, err := svc.skills.FindByID(nil, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ZipURL != "/static/skills/old.zip" || updated.ZipFilename != "old.zip" || updated.FileSize != 10 || updated.SkillMD != "# Old" {
		t.Fatalf("metadata changed after invalid replacement: %+v", updated)
	}
}

func TestSkillUploadZipRemovesNewFileWhenMetadataUpdateFails(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, err := svc.Create(ctx, u.ID, CreateSkillInput{Name: "zip-update-failure"})
	if err != nil {
		t.Fatal(err)
	}
	oldZip := makeSkillZip(t, zipFixture{name: "SKILL.md", content: "# Old"})
	oldPath := filepath.Join(svc.ZipDir(), "old.zip")
	if err := os.WriteFile(oldPath, oldZip, 0o600); err != nil {
		t.Fatal(err)
	}
	sk.ZipURL = "/static/skills/old.zip"
	sk.ZipFilename = "old.zip"
	sk.FileSize = int64(len(oldZip))
	sk.SkillMD = "# Old"
	if err := svc.skills.Update(nil, sk); err != nil {
		t.Fatal(err)
	}

	newZip := makeSkillZip(t, zipFixture{name: "SKILL.md", content: "# New"})
	newPath := filepath.Join(svc.ZipDir(), "new.zip")
	if err := os.WriteFile(newPath, newZip, 0o600); err != nil {
		t.Fatal(err)
	}

	err = svc.UploadZip(ctx, u.ID, sk.ID, "/static/skills/new.zip", strings.Repeat("a", 256), int64(len(newZip)))
	if err == nil {
		t.Fatal("upload with an oversized filename should fail database update")
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf("new file still exists after failed update: %v", err)
	}
	if got, err := os.ReadFile(oldPath); err != nil || string(got) != string(oldZip) {
		t.Fatalf("old file = %q, err = %v, want original contents", got, err)
	}
	updated, err := svc.skills.FindByID(nil, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ZipURL != "/static/skills/old.zip" || updated.ZipFilename != "old.zip" || updated.FileSize != int64(len(oldZip)) || updated.SkillMD != "# Old" {
		t.Fatalf("metadata changed after failed update: %+v", updated)
	}
}

func TestSkillDownload(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "dl-skill"})
	sk.Status = model.ResourceStatusPublished
	sk.ZipURL = "/static/skills/dl.zip"
	now := sk.CreatedAt
	sk.PublishedAt = &now
	_ = svc.skills.Update(nil, sk)

	url, err := svc.Download(ctx, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if url != "/static/skills/dl.zip" {
		t.Fatalf("url = %q", url)
	}
	dl, _ := svc.skills.FindByID(nil, sk.ID)
	if dl.Downloads != 1 {
		t.Fatalf("downloads = %d, want 1", dl.Downloads)
	}

	sk2, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "no-zip"})
	sk2.Status = model.ResourceStatusPublished
	now2 := sk2.CreatedAt
	sk2.PublishedAt = &now2
	_ = svc.skills.Update(nil, sk2)
	if _, err := svc.Download(ctx, sk2.ID); err == nil {
		t.Fatal("download with empty zip_url should fail")
	}

	hidden, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "hidden-download"})
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	hidden.ZipURL = "/static/skills/hidden.zip"
	hidden.Downloads = 7
	if err := svc.skills.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Download(ctx, hidden.ID); err != ErrSkillNotFound {
		t.Errorf("hidden download error = %v, want %v", err, ErrSkillNotFound)
	}
	persistedHidden, err := svc.skills.FindByID(nil, hidden.ID)
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

func TestSkillToggleLike(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "like-skill"})
	sk.Status = model.ResourceStatusPublished
	now := sk.CreatedAt
	sk.PublishedAt = &now
	_ = svc.skills.Update(nil, sk)

	liked, count, err := svc.ToggleLike(ctx, u.ID, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !liked || count != 1 {
		t.Fatalf("liked=%v count=%d", liked, count)
	}
	liked, count, err = svc.ToggleLike(ctx, u.ID, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if liked || count != 0 {
		t.Fatalf("unliked=%v count=%d", liked, count)
	}

	draft, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "draft-like"})
	if _, _, err := svc.ToggleLike(ctx, u.ID, draft.ID); err == nil {
		t.Fatal("like on draft should fail")
	}
}

func TestSkillToggleFavorite(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	sk, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "fav-skill"})
	sk.Status = model.ResourceStatusPublished
	now := sk.CreatedAt
	sk.PublishedAt = &now
	_ = svc.skills.Update(nil, sk)

	favorited, count, err := svc.ToggleFavorite(ctx, u.ID, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !favorited || count != 1 {
		t.Fatalf("favorited=%v count=%d", favorited, count)
	}
	favorited, count, err = svc.ToggleFavorite(ctx, u.ID, sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if favorited || count != 0 {
		t.Fatalf("unfavorited=%v count=%d", favorited, count)
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

func TestSkillPublicListWithAuthorIDHidesHidden(t *testing.T) {
	svc, owner := newSkillTestEnv(t)
	ctx := context.Background()
	hidden, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "hidden"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	if err := svc.skills.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}

	authorID := owner.ID
	result, err := svc.List(ctx, SkillListQuery{AuthorID: &authorID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 || len(result.List) != 0 {
		t.Fatalf("public list exposed hidden skill: %+v", result)
	}
}

func TestSkillReadDoesNotIncrementViewsOrLoadInteractions(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()
	skill, err := svc.Create(ctx, u.ID, CreateSkillInput{Name: "read-only"})
	if err != nil {
		t.Fatal(err)
	}
	skill.Status = model.ResourceStatusPublished
	if err := svc.skills.Update(nil, skill); err != nil {
		t.Fatal(err)
	}
	db := svc.skills.DB()
	if err := db.Create(&model.SkillLike{SkillID: skill.ID, UserID: u.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SkillFavorite{SkillID: skill.ID, UserID: u.ID}).Error; err != nil {
		t.Fatal(err)
	}

	beforeViews := skill.Views
	beforeDownloads := skill.Downloads
	detail, err := svc.Read(ctx, u.ID, skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Views != beforeViews || detail.Downloads != beforeDownloads {
		t.Fatalf("detail counts = views %d downloads %d, want views %d downloads %d", detail.Views, detail.Downloads, beforeViews, beforeDownloads)
	}
	if detail.Liked || detail.Favorited {
		t.Fatalf("read returned interaction state: %+v", detail)
	}
	var persisted model.Skill
	if err := db.First(&persisted, skill.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Views != beforeViews || persisted.Downloads != beforeDownloads {
		t.Fatalf("persisted counts = views %d downloads %d, want views %d downloads %d", persisted.Views, persisted.Downloads, beforeViews, beforeDownloads)
	}
}

func TestSkillReadRestrictsHiddenAndUnpublishedToOwner(t *testing.T) {
	svc, owner := newSkillTestEnv(t)
	ctx := context.Background()
	other := &model.User{Email: "skill-other@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := repo.NewUserRepo(svc.skills.DB()).Create(other); err != nil {
		t.Fatal(err)
	}
	hidden, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "hidden"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	if err := svc.skills.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}
	draft, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "draft"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Read(ctx, owner.ID, hidden.ID); err != nil {
		t.Fatalf("owner cannot read hidden skill: %v", err)
	}
	if _, err := svc.Read(ctx, other.ID, hidden.ID); err == nil {
		t.Fatal("other actor read hidden skill")
	}
	if _, err := svc.Read(ctx, owner.ID, draft.ID); err != nil {
		t.Fatalf("owner cannot read draft skill: %v", err)
	}
	if _, err := svc.Read(ctx, other.ID, draft.ID); err == nil {
		t.Fatal("other actor read draft skill")
	}
}

func TestSkillListOwnedIncludesHiddenExcludesDeletedAndValidatesStatus(t *testing.T) {
	svc, owner := newSkillTestEnv(t)
	ctx := context.Background()
	other := &model.User{Email: "skill-list-other@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := repo.NewUserRepo(svc.skills.DB()).Create(other); err != nil {
		t.Fatal(err)
	}
	draft, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "rejected"})
	if err != nil {
		t.Fatal(err)
	}
	rejected.Status = model.ResourceStatusRejected
	if err := svc.skills.Update(nil, rejected); err != nil {
		t.Fatal(err)
	}
	hidden, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "hidden"})
	if err != nil {
		t.Fatal(err)
	}
	hidden.Status = model.ResourceStatusPublished
	hidden.Hidden = true
	if err := svc.skills.Update(nil, hidden); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, other.ID, CreateSkillInput{Name: "other"}); err != nil {
		t.Fatal(err)
	}
	deleted, err := svc.Create(ctx, owner.ID, CreateSkillInput{Name: "deleted"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.skills.Delete(nil, deleted.ID); err != nil {
		t.Fatal(err)
	}

	got, err := svc.ListOwned(ctx, owner.ID, "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 3 || len(got.List) != 3 {
		t.Fatalf("owned skills = %+v", got)
	}
	seen := map[uint]string{}
	for _, summary := range got.List {
		seen[summary.ID] = summary.Status
	}
	if seen[draft.ID] != string(model.ResourceStatusDraft) || seen[rejected.ID] != string(model.ResourceStatusRejected) || seen[hidden.ID] != string(model.ResourceStatusPublished) {
		t.Fatalf("owned skill statuses = %+v", seen)
	}

	rejectedOnly, err := svc.ListOwned(ctx, owner.ID, string(model.ResourceStatusRejected), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if rejectedOnly.Total != 1 || len(rejectedOnly.List) != 1 || rejectedOnly.List[0].ID != rejected.ID {
		t.Fatalf("rejected skills = %+v", rejectedOnly)
	}
	if _, err := svc.ListOwned(ctx, owner.ID, "invalid", 1, 20); err == nil {
		t.Fatal("unknown skill status accepted")
	}
}
