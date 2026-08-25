package repo

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestSkillRepoCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewSkillRepo(db)
	ctx := context.Background()

	u := &model.User{Email: "s@s.com", PasswordHash: "x", Nickname: "S", AvatarURL: "/x.png"}
	if err := NewUserRepo(db).Create(u); err != nil {
		t.Fatal(err)
	}

	s := &model.Skill{
		AuthorID: u.ID, Name: "test-skill", Description: "desc",
		Status: model.ResourceStatusDraft,
	}
	if err := r.Create(db, s); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByID(db, s.ID)
	if err != nil || got.Name != "test-skill" {
		t.Fatalf("FindByID = %v, %v", got, err)
	}
	if err := r.SetSkillTags(db, s.ID, []uint{10, 11}); err != nil {
		t.Fatal(err)
	}
	ids, err := r.FindSkillTags(db, s.ID)
	if err != nil || len(ids) != 2 {
		t.Fatalf("FindSkillTags = %v, %v", ids, err)
	}
	now := time.Now()
	s.Status = model.ResourceStatusPublished
	s.PublishedAt = &now
	if err := r.Update(db, s); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, s.ID, "views", 1); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, s.ID, "downloads", 3); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, s.ID)
	if got.Views != 1 || got.Downloads != 3 {
		t.Fatalf("counts = views %d downloads %d", got.Views, got.Downloads)
	}
	if err := r.IncrViews(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, s.ID)
	if got.Views != 2 {
		t.Fatalf("IncrViews views = %d", got.Views)
	}
	if err := r.Delete(db, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, s.ID); err == nil {
		t.Fatal("soft-deleted skill still found")
	}
}

func TestSkillRepoList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewSkillRepo(db)
	ctx := context.Background()

	_ = NewUserRepo(db).Create(&model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A"})

	now := time.Now()
	for i := 0; i < 3; i++ {
		s := &model.Skill{
			AuthorID: 1, Name: "skill", Description: "d",
			Status: model.ResourceStatusPublished, PublishedAt: &now,
		}
		if err := r.Create(db, s); err != nil {
			t.Fatal(err)
		}
	}
	_ = r.SetSkillTags(db, 1, []uint{1})

	q := SkillQuery{Page: 1, PageSize: 2, Sort: "latest"}
	list, total, err := r.List(ctx, q)
	if err != nil || total != 3 || len(list) != 2 {
		t.Fatalf("list = %d total, %d len, err %v", total, len(list), err)
	}

	tagID := uint(1)
	q.TagID = &tagID
	list, total, _ = r.List(ctx, q)
	if total != 1 {
		t.Fatalf("tag filter total = %d", total)
	}

	q.Sort = "hot"
	q.TagID = nil
	list, total, _ = r.List(ctx, q)
	if total != 3 || len(list) != 2 {
		t.Fatalf("hot sort = %d total, %d len", total, len(list))
	}

	q.Sort = "downloads"
	list, total, _ = r.List(ctx, q)
	if total != 3 || len(list) != 2 {
		t.Fatalf("downloads sort = %d total, %d len", total, len(list))
	}
}

func TestSkillRepoListOwnedScopesAuthorIncludesHiddenAndRejectsUnknownStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewSkillRepo(db)
	ctx := context.Background()
	owner := &model.User{Email: "owned-skill@t.com", PasswordHash: "x", Nickname: "Owner"}
	other := &model.User{Email: "other-skill@t.com", PasswordHash: "x", Nickname: "Other"}
	if err := NewUserRepo(db).Create(owner); err != nil {
		t.Fatal(err)
	}
	if err := NewUserRepo(db).Create(other); err != nil {
		t.Fatal(err)
	}
	draft := &model.Skill{AuthorID: owner.ID, Name: "draft", Status: model.ResourceStatusDraft}
	hidden := &model.Skill{AuthorID: owner.ID, Name: "hidden", Status: model.ResourceStatusPublished, Hidden: true}
	foreign := &model.Skill{AuthorID: other.ID, Name: "foreign", Status: model.ResourceStatusRejected}
	deleted := &model.Skill{AuthorID: owner.ID, Name: "deleted", Status: model.ResourceStatusArchived}
	for _, skill := range []*model.Skill{draft, hidden, foreign, deleted} {
		if err := r.Create(db, skill); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Delete(db, deleted.ID); err != nil {
		t.Fatal(err)
	}

	list, total, err := r.ListOwned(ctx, owner.ID, "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("owned skills = total %d, list %+v", total, list)
	}
	seen := map[uint]bool{}
	for _, skill := range list {
		seen[skill.ID] = true
	}
	if !seen[draft.ID] || !seen[hidden.ID] || seen[foreign.ID] || seen[deleted.ID] {
		t.Fatalf("owned skill IDs = %+v", seen)
	}
	drafts, total, err := r.ListOwned(ctx, owner.ID, string(model.ResourceStatusDraft), 1, 20)
	if err != nil || total != 1 || len(drafts) != 1 || drafts[0].ID != draft.ID {
		t.Fatalf("draft skills = total %d, list %+v, err %v", total, drafts, err)
	}
	if _, _, err := r.ListOwned(ctx, owner.ID, "invalid", 1, 20); err == nil {
		t.Fatal("unknown skill status accepted")
	}
}
