package app

import (
	"log/slog"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{}, &model.Category{}, &model.Tag{}, &model.Article{},
		&model.ArticleTag{}, &model.ArticleLike{}, &model.ArticleFavorite{},
		&model.Comment{}, &model.CommentLike{},
		&model.Skill{}, &model.SkillTag{},
		&model.McpServer{}, &model.McpServerTag{},
		&model.SkillLike{}, &model.SkillFavorite{},
		&model.McpServerLike{}, &model.McpServerFavorite{},
		&model.ResourceComment{}, &model.ResourceCommentLike{},
		&model.Notification{}, &model.Report{}, &model.AdminLog{}, &model.Announcement{},
	); err != nil {
		return err
	}
	if err := platform.CreateFulltextIndexes(db); err != nil {
		slog.Warn("fulltext indexes", "err", err)
	}
	return nil
}
