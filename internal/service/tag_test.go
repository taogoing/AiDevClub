package service

import (
	"context"
	"testing"

	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func TestTagServiceList(t *testing.T) {
	db := testutil.NewTestDB(t)
	tagRepo := repo.NewTagRepo(db)
	ctx := context.Background()
	_, _ = tagRepo.Create(ctx, "gin")
	_, _ = tagRepo.Create(ctx, "gorm")
	svc := NewTagService(tagRepo)

	hot, err := svc.List(ctx, "", true, 10)
	if err != nil || len(hot) != 2 {
		t.Fatalf("hot = %v, %v", hot, err)
	}
	filtered, err := svc.List(ctx, "gi", false, 10)
	if err != nil || len(filtered) != 1 {
		t.Fatalf("filtered = %v, %v", filtered, err)
	}
}
