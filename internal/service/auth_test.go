package service

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func TestHashAndCheckPassword(t *testing.T) {
	h, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if h == "secret123" {
		t.Fatal("password not hashed")
	}
	if err := checkPassword(h, "secret123"); err != nil {
		t.Fatalf("check = %v, want nil", err)
	}
	if err := checkPassword(h, "wrong"); err == nil {
		t.Fatal("wrong password accepted")
	}
}

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()
	cfg := &platform.Config{
		DefaultAvatarURL: "/static/avatars/default.png",
		JWTSecret:        "s",
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
	}
	return NewAuthService(
		repo.NewUserRepo(testutil.NewTestDB(t)),
		repo.NewTokenRepo(testutil.NewTestRedis(t), time.Hour),
		cfg,
	)
}

func TestRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	if err := svc.Register(ctx, RegisterInput{Email: "a@example.com", Password: "secret123"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Register(ctx, RegisterInput{Email: "a@example.com", Password: "secret123"}); err == nil {
		t.Fatal("duplicate email accepted")
	}

	out, err := svc.Login(ctx, LoginInput{Email: "a@example.com", Password: "secret123"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("empty tokens")
	}
}

func TestRegisterAfterSoftDeleteReturnsErrEmailExists(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)
	const email = "soft-delete@example.com"

	if err := svc.Register(ctx, RegisterInput{Email: email, Password: "secret123"}); err != nil {
		t.Fatal(err)
	}
	u, err := svc.users.FindByEmail(email)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.users.Delete(u.ID); err != nil {
		t.Fatal(err)
	}

	err = svc.Register(ctx, RegisterInput{Email: email, Password: "secret123"})
	if err != ErrEmailExists {
		t.Fatalf("re-register after soft delete = %v, want ErrEmailExists", err)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	if err := svc.Register(ctx, RegisterInput{Email: "a@example.com", Password: "secret123"}); err != nil {
		t.Fatal(err)
	}
	pair, err := svc.Login(ctx, LoginInput{Email: "a@example.com", Password: "secret123"})
	if err != nil {
		t.Fatal(err)
	}

	newPair, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh token not rotated")
	}
	// 旧 refresh 已作废
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("old refresh token still valid")
	}
}

func TestLogoutRevokesRefresh(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	_ = svc.Register(ctx, RegisterInput{Email: "b@example.com", Password: "secret123"})
	pair, _ := svc.Login(ctx, LoginInput{Email: "b@example.com", Password: "secret123"})

	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("refresh token still valid after logout")
	}
}
