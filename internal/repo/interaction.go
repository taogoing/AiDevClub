package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
)

type InteractionRepo struct{ db *gorm.DB }

func NewInteractionRepo(db *gorm.DB) *InteractionRepo { return &InteractionRepo{db: db} }

func (r *InteractionRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *InteractionRepo) ArticleLiked(ctx context.Context, userID, articleID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ArticleLike{}).Where("article_id = ? AND user_id = ?", articleID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) ArticleFavorited(ctx context.Context, userID, articleID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ArticleFavorite{}).Where("article_id = ? AND user_id = ?", articleID, userID).Count(&count).Error
	return count > 0, err
}

func toggleLike(db *gorm.DB, m interface{}, uniqWhere string, uniqArgs ...interface{}) (bool, error) {
	if err := db.Create(m).Error; err == nil {
		return true, nil
	} else if !platform.IsDuplicateEntry(err) {
		return false, err
	}
	if err := db.Where(uniqWhere, uniqArgs...).Delete(m).Error; err != nil {
		return false, err
	}
	return false, nil
}

func (r *InteractionRepo) ToggleArticleLike(db *gorm.DB, userID, articleID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.ArticleLike{UserID: userID, ArticleID: articleID}, "article_id = ? AND user_id = ?", articleID, userID)
}

func (r *InteractionRepo) ToggleArticleFavorite(db *gorm.DB, userID, articleID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.ArticleFavorite{UserID: userID, ArticleID: articleID}, "article_id = ? AND user_id = ?", articleID, userID)
}

func (r *InteractionRepo) ToggleCommentLike(db *gorm.DB, userID, commentID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.CommentLike{UserID: userID, CommentID: commentID}, "comment_id = ? AND user_id = ?", commentID, userID)
}

func (r *InteractionRepo) CommentLiked(db *gorm.DB, userID, commentID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.CommentLike{}).Where("comment_id = ? AND user_id = ?", commentID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) SkillLiked(ctx context.Context, userID, skillID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.SkillLike{}).Where("skill_id = ? AND user_id = ?", skillID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) SkillFavorited(ctx context.Context, userID, skillID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.SkillFavorite{}).Where("skill_id = ? AND user_id = ?", skillID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) ToggleSkillLike(db *gorm.DB, userID, skillID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.SkillLike{UserID: userID, SkillID: skillID}, "skill_id = ? AND user_id = ?", skillID, userID)
}

func (r *InteractionRepo) ToggleSkillFavorite(db *gorm.DB, userID, skillID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.SkillFavorite{UserID: userID, SkillID: skillID}, "skill_id = ? AND user_id = ?", skillID, userID)
}

func (r *InteractionRepo) McpServerLiked(ctx context.Context, userID, serverID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.McpServerLike{}).Where("mcp_server_id = ? AND user_id = ?", serverID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) McpServerFavorited(ctx context.Context, userID, serverID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.McpServerFavorite{}).Where("mcp_server_id = ? AND user_id = ?", serverID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) ToggleMcpServerLike(db *gorm.DB, userID, serverID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.McpServerLike{UserID: userID, McpServerID: serverID}, "mcp_server_id = ? AND user_id = ?", serverID, userID)
}

func (r *InteractionRepo) ToggleMcpServerFavorite(db *gorm.DB, userID, serverID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.McpServerFavorite{UserID: userID, McpServerID: serverID}, "mcp_server_id = ? AND user_id = ?", serverID, userID)
}

func (r *InteractionRepo) ResourceCommentLiked(db *gorm.DB, userID, commentID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.ResourceCommentLike{}).Where("comment_id = ? AND user_id = ?", commentID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) ToggleResourceCommentLike(db *gorm.DB, userID, commentID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.ResourceCommentLike{UserID: userID, CommentID: commentID}, "comment_id = ? AND user_id = ?", commentID, userID)
}
