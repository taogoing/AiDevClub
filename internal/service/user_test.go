package service

import (
	"context"
	"errors"
	"testing"

	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func TestUserServiceGetPropagatesNonNotFoundRepoError(t *testing.T) {
	users := repo.NewUserRepo(testutil.NewTestDB(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewUserService(users, nil, nil).Get(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, must not be ErrUserNotFound", err)
	}
}

func TestUserProfileUpdateChangePasswordDelete(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	_ = svc.Register(ctx, RegisterInput{Email: "u@example.com", Password: "secret123"})
	u, _ := svc.users.FindByEmail("u@example.com")

	userSvc := NewUserService(svc.users, svc.tokens, svc.cfg)

	if err := userSvc.UpdateProfile(ctx, u.ID, UpdateProfileInput{Nickname: "新昵称", Bio: "你好"}); err != nil {
		t.Fatal(err)
	}
	got, _ := userSvc.Get(ctx, u.ID)
	if got.Nickname != "新昵称" {
		t.Fatalf("nickname = %s", got.Nickname)
	}

	if err := userSvc.ChangePassword(ctx, u.ID, "newpass123"); err != nil {
		t.Fatal(err)
	}

	if err := userSvc.Delete(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := userSvc.Get(ctx, u.ID); err == nil {
		t.Fatal("user still visible after delete")
	}
}
