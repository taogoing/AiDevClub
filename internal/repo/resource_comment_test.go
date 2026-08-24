package repo

import (
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestResourceCommentRepoCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewResourceCommentRepo(db)

	users := NewUserRepo(db)
	_ = users.Create(&model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A"})

	c := &model.ResourceComment{ResourceType: "skill", ResourceID: 1, AuthorID: 1, Content: "hi"}
	if err := r.Create(db, c); err != nil {
		t.Fatal(err)
	}
	pid := c.ID
	r2 := &model.ResourceComment{ResourceType: "skill", ResourceID: 1, AuthorID: 1, Content: "reply", ParentID: &pid}
	if err := r.Create(db, r2); err != nil {
		t.Fatal(err)
	}
	list, err := r.ListByResource(t.Context(), "skill", 1)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if err := r.IncrLikes(db, c.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(db, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, c.ID); err == nil {
		t.Fatal("deleted comment found")
	}
	list, _ = r.ListByResource(t.Context(), "skill", 1)
	if len(list) != 1 {
		t.Fatalf("after delete len = %d", len(list))
	}
}
