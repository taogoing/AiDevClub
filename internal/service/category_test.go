package service

import (
	"context"
	"testing"

	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func TestCategoryServiceList(t *testing.T) {
	db := testutil.NewTestDB(t)
	catRepo := repo.NewCategoryRepo(db)
	_ = catRepo.Seed(context.Background())
	svc := NewCategoryService(catRepo)
	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("empty list")
	}
}
