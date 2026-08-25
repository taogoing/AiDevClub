package repo

import (
	"context"
	"errors"
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
	if ok, _ := r.ArticleLiked(context.Background(), 1, 1); ok {
		t.Fatal("ArticleLiked should be false after unlike")
	}
	if ok, _ := r.ArticleFavorited(context.Background(), 1, 1); !ok {
		t.Fatal("ArticleFavorited should be true")
	}
}

func TestInteractionReadMethodsRespectContext(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewInteractionRepo(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reads := []struct {
		name  string
		query func() (bool, error)
	}{
		{name: "article liked", query: func() (bool, error) { return r.ArticleLiked(ctx, 1, 1) }},
		{name: "article favorited", query: func() (bool, error) { return r.ArticleFavorited(ctx, 1, 1) }},
		{name: "skill liked", query: func() (bool, error) { return r.SkillLiked(ctx, 1, 1) }},
		{name: "skill favorited", query: func() (bool, error) { return r.SkillFavorited(ctx, 1, 1) }},
		{name: "MCP server liked", query: func() (bool, error) { return r.McpServerLiked(ctx, 1, 1) }},
		{name: "MCP server favorited", query: func() (bool, error) { return r.McpServerFavorited(ctx, 1, 1) }},
	}
	for _, read := range reads {
		t.Run(read.name, func(t *testing.T) {
			if _, err := read.query(); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
}
