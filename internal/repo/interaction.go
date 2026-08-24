package repo

import (
	"gorm.io/gorm"

	"aidevclub/internal/model"
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
