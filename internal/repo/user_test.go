package repo

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestUserRepoFindByIDWithContextHonorsCancellation(t *testing.T) {
	r := NewUserRepo(testutil.NewTestDB(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.FindByIDWithContext(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestUserRepoCreateAndFindByEmail(t *testing.T) {
	r := NewUserRepo(testutil.NewTestDB(t))

	u := &model.User{Email: "a@example.com", PasswordHash: "x", Nickname: "用户_abc", AvatarURL: "/static/avatars/default.png"}
	if err := r.Create(u); err != nil {
		t.Fatal(err)
	}

	got, err := r.FindByEmail("a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatalf("id = %d, want %d", got.ID, u.ID)
	}
}

func TestUserRepoSoftDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewUserRepo(db)

	u := &model.User{Email: "b@example.com", PasswordHash: "x", Nickname: "用户_abc", AvatarURL: "/static/avatars/default.png"}
	if err := r.Create(u); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByEmail("b@example.com"); err != gorm.ErrRecordNotFound {
		t.Fatalf("err = %v, want ErrRecordNotFound", err)
	}

	// 证明是软删除：Unscoped 仍能查到原记录（硬删除则会 ErrRecordNotFound）。
	var raw model.User
	if err := db.Unscoped().First(&raw, u.ID).Error; err != nil {
		t.Fatalf("软删除后 Unscoped 应仍能查到记录: %v", err)
	}
	if !raw.DeletedAt.Valid {
		t.Fatalf("DeletedAt.Valid = false, 期望软删除已写入删除时间戳")
	}
}
