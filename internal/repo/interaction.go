package repo

import (
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

func (r *InteractionRepo) ArticleLiked(db *gorm.DB, userID, articleID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.ArticleLike{}).Where("article_id = ? AND user_id = ?", articleID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) ArticleFavorited(db *gorm.DB, userID, articleID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.ArticleFavorite{}).Where("article_id = ? AND user_id = ?", articleID, userID).Count(&count).Error
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
