package testutil

import (
	"testing"

	"aidevclub/internal/model"
)

func TestNewTestDBMigratesAllModels(t *testing.T) {
	db := NewTestDB(t)
	for _, m := range []interface{}{
		&model.Category{}, &model.Tag{}, &model.Article{}, &model.ArticleTag{},
		&model.ArticleLike{}, &model.ArticleFavorite{}, &model.Comment{}, &model.CommentLike{},
	} {
		if !db.Migrator().HasTable(m) {
			t.Fatalf("table %T not migrated", m)
		}
	}
}
