package repo

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/testutil"
)

func TestTokenRepoIssueValidateRevoke(t *testing.T) {
	ctx := context.Background()
	r := NewTokenRepo(testutil.NewTestRedis(t), time.Hour)

	tok, err := r.Issue(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}

	uid, err := r.Validate(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 7 {
		t.Fatalf("uid = %d, want 7", uid)
	}

	if err := r.Revoke(ctx, tok); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Validate(ctx, tok); err == nil {
		t.Fatal("token still valid after revoke")
	}
}

func TestTokenRepoRevokeAllForUser(t *testing.T) {
	ctx := context.Background()
	r := NewTokenRepo(testutil.NewTestRedis(t), time.Hour)

	t1, _ := r.Issue(ctx, 9)
	t2, _ := r.Issue(ctx, 9)
	if err := r.RevokeAllForUser(ctx, 9); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Validate(ctx, t1); err == nil {
		t.Fatal("t1 still valid")
	}
	if _, err := r.Validate(ctx, t2); err == nil {
		t.Fatal("t2 still valid")
	}
}
