package service

import (
	"context"
	"testing"
)

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
