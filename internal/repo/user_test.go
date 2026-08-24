package repo

import (
	"testing"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestUserRepoCreateAndFindByEmail(t *testing.T) {
	r := NewUserRepo(testutil.NewTestDB(t))

	u := &model.User{Email: "a@example.com", PasswordHash: "x", Nickname: "用户_abc", AvatarURL: "/static/avatars/default.png"}
	if err := r.Create(u); err != nil {
		t.Fatal(err)
	}

	got, err := r.FindByEmail("a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatalf("id = %d, want %d", got.ID, u.ID)
	}
}

func TestUserRepoSoftDelete(t *testing.T) {
	r := NewUserRepo(testutil.NewTestDB(t))

	u := &model.User{Email: "b@example.com", PasswordHash: "x", Nickname: "用户_abc", AvatarURL: "/static/avatars/default.png"}
	if err := r.Create(u); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByEmail("b@example.com"); err != gorm.ErrRecordNotFound {
		t.Fatalf("err = %v, want ErrRecordNotFound", err)
	}
}
