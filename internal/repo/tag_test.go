package repo

import (
	"context"
	"testing"

	"aidevclub/internal/testutil"
)

func TestTagCRUDAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewTagRepo(db)
	ctx := context.Background()

	tg, err := r.Create(ctx, "gin")
	if err != nil {
		t.Fatal(err)
	}
	if tg.ID == 0 {
		t.Fatal("no id")
	}
	if _, err := r.Create(ctx, "gin"); err == nil {
		t.Fatal("duplicate tag accepted")
	}
	if found, err := r.FindByName(ctx, "gin"); err != nil || found.ID != tg.ID {
		t.Fatalf("FindByName = %v, %v", found, err)
	}
	if err := r.IncrUsage(db, tg.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrUsage(db, tg.ID, -1); err != nil {
		t.Fatal(err)
	}
	list, err := r.List(ctx, "gi", 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("List keyword = %v, %v", list, err)
	}
	if err := r.IncrUsage(db, tg.ID, 3); err != nil {
		t.Fatal(err)
	}
	hot, err := r.ListHot(ctx, 10)
	if err != nil || len(hot) != 1 || hot[0].UsageCount != 3 {
		t.Fatalf("ListHot = %v, %v", hot, err)
	}
}
