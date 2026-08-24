package repo

import (
	"testing"

	"aidevclub/internal/testutil"
)

func TestInteractionToggle(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewInteractionRepo(db)

	liked, err := r.ToggleArticleLike(db, 1, 1)
	if err != nil || !liked {
		t.Fatalf("first like = %v, %v", liked, err)
	}
	liked, err = r.ToggleArticleLike(db, 1, 1)
	if err != nil || liked {
		t.Fatalf("second like should unlike, got %v, %v", liked, err)
	}
	fav, err := r.ToggleArticleFavorite(db, 1, 1)
	if err != nil || !fav {
		t.Fatalf("favorite = %v, %v", fav, err)
	}
	cl, err := r.ToggleCommentLike(db, 1, 1)
	if err != nil || !cl {
		t.Fatalf("comment like = %v, %v", cl, err)
	}
	if ok, _ := r.ArticleLiked(db, 1, 1); ok {
		t.Fatal("ArticleLiked should be false after unlike")
	}
	if ok, _ := r.ArticleFavorited(db, 1, 1); !ok {
		t.Fatal("ArticleFavorited should be true")
	}
}
