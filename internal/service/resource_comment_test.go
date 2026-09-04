package service

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newResCommentTestEnv(t *testing.T) (*ResourceCommentService, *model.User, *model.Skill) {
	t.Helper()
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	u := &model.User{Email: "rc@t.com", PasswordHash: "x", Nickname: "RC", AvatarURL: "/x.png"}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}
	skillRepo := repo.NewSkillRepo(db)
	mcpRepo := repo.NewMcpServerRepo(db)
	sk := &model.Skill{
		AuthorID: u.ID, Name: "test-skill", Status: model.ResourceStatusPublished,
	}
	now := time.Now()
	sk.PublishedAt = &now
	if err := skillRepo.Create(db, sk); err != nil {
		t.Fatal(err)
	}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	svc := NewResourceCommentService(
		repo.NewResourceCommentRepo(db),
		skillRepo,
		mcpRepo,
		repo.NewInteractionRepo(db),
		users,
		notifSvc,
		nil,
	)
	return svc, u, sk
}

func TestResourceCommentCreate(t *testing.T) {
	svc, u, sk := newResCommentTestEnv(t)
	ctx := context.Background()

	c1, err := svc.Create(ctx, u.ID, "skill", sk.ID, "一级评论", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c1.ID == 0 {
		t.Fatal("comment id = 0")
	}
	pid := c1.ID
	c2, err := svc.Create(ctx, u.ID, "skill", sk.ID, "回复", &pid)
	if err != nil {
		t.Fatal(err)
	}
	if c2.ParentID == nil || *c2.ParentID != pid {
		t.Fatalf("parent_id = %v", c2.ParentID)
	}
	if _, err := svc.Create(ctx, u.ID, "skill", sk.ID, "回复的回复", &c2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, u.ID, "skill", 99999, "不存在资源", nil); err == nil {
		t.Fatal("should fail for non-existent resource")
	}
	if _, err := svc.Create(ctx, u.ID, "skill", sk.ID, "", nil); err == nil {
		t.Fatal("should fail for empty content")
	}
}

func TestResourceCommentList(t *testing.T) {
	svc, u, sk := newResCommentTestEnv(t)
	ctx := context.Background()

	c1, _ := svc.Create(ctx, u.ID, "skill", sk.ID, "一级评论", nil)
	pid := c1.ID
	svc.Create(ctx, u.ID, "skill", sk.ID, "回复1", &pid)
	svc.Create(ctx, u.ID, "skill", sk.ID, "回复2", &pid)

	list, err := svc.List(ctx, "skill", sk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("root count = %d, want 1", len(list))
	}
	if len(list[0].Replies) != 2 {
		t.Fatalf("replies = %d, want 2", len(list[0].Replies))
	}
	if list[0].Author.Nickname != "RC" {
		t.Fatalf("author nickname = %q", list[0].Author.Nickname)
	}
}

func TestResourceCommentDelete(t *testing.T) {
	svc, u, sk := newResCommentTestEnv(t)
	ctx := context.Background()

	c, _ := svc.Create(ctx, u.ID, "skill", sk.ID, "要删除的评论", nil)
	if err := svc.Delete(ctx, u.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.List(ctx, "skill", sk.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, u.ID, 99999); err == nil {
		t.Fatal("should fail for non-existent comment")
	}

	other := &model.User{Email: "o@t.com", PasswordHash: "x", Nickname: "O", AvatarURL: "/o.png"}
	_ = repo.NewUserRepo(svc.getDB()).Create(other)
	c2, _ := svc.Create(ctx, other.ID, "skill", sk.ID, "别人的评论", nil)
	if err := svc.Delete(ctx, other.ID+999, c2.ID); err == nil {
		t.Fatal("non-author/non-resource-owner should not delete")
	}
	if err := svc.Delete(ctx, u.ID, c2.ID); err != nil {
		t.Fatalf("resource author should delete: %v", err)
	}
}

func TestResourceCommentToggleLike(t *testing.T) {
	svc, u, sk := newResCommentTestEnv(t)
	ctx := context.Background()

	c, _ := svc.Create(ctx, u.ID, "skill", sk.ID, "点赞评论", nil)
	liked, count, err := svc.ToggleLike(ctx, u.ID, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !liked || count != 1 {
		t.Fatalf("liked=%v count=%d", liked, count)
	}
	liked, count, err = svc.ToggleLike(ctx, u.ID, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if liked || count != 0 {
		t.Fatalf("unliked=%v count=%d", liked, count)
	}
	if _, _, err := svc.ToggleLike(ctx, u.ID, 99999); err == nil {
		t.Fatal("should fail for non-existent comment")
	}
}
