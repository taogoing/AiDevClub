package app

import (
	"log/slog"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
)

func Migrate(db *gorm.DB) error {
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if db.Migrator().HasColumn(&model.Article{}, "category_id") {
		db.Exec("ALTER TABLE articles DROP FOREIGN KEY fk_articles_category")
		if err := db.Migrator().DropColumn(&model.Article{}, "category_id"); err != nil {
			return err
		}
	}
	if db.Migrator().HasTable("categories") {
		if err := db.Migrator().DropTable("categories"); err != nil {
			return err
		}
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	if err := db.AutoMigrate(
		&model.User{}, &model.Tag{}, &model.Article{},
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
