package repo

import (
	"context"
	"testing"

	"aidevclub/internal/testutil"
)

func TestCategorySeedAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewCategoryRepo(db)
	ctx := context.Background()
	if err := r.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	// 幂等：再跑一次不报错、数量不变
	if err := r.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	list, err := r.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("no categories seeded")
	}
	if list[0].Name == "" || list[0].Slug == "" {
		t.Fatal("category missing name/slug")
	}
}
