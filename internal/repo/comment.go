package repo

import (
	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type CommentRepo struct{ db *gorm.DB }

func NewCommentRepo(db *gorm.DB) *CommentRepo { return &CommentRepo{db: db} }

func (r *CommentRepo) DB() *gorm.DB { return r.db }

func (r *CommentRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *CommentRepo) Create(db *gorm.DB, c *model.Comment) error {
	return r.exec(db).Create(c).Error
}

func (r *CommentRepo) FindByID(db *gorm.DB, id uint) (*model.Comment, error) {
	var c model.Comment
	if err := r.exec(db).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CommentRepo) ListByArticle(db *gorm.DB, articleID uint) ([]model.Comment, error) {
	var list []model.Comment
	err := r.exec(db).Where("article_id = ?", articleID).Order("created_at asc, id asc").Find(&list).Error
	return list, err
}

func (r *CommentRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.Comment{}, id).Error
}

func (r *CommentRepo) IncrLikes(db *gorm.DB, id uint, delta int) error {
	return r.exec(db).Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("likes_count + ?", delta)).Error
}
