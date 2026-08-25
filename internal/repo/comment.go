package repo

import (
	"context"

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
	err := r.exec(db).Where("article_id = ? AND hidden = ?", articleID, false).Order("created_at asc, id asc").Find(&list).Error
	return list, err
}

func (r *CommentRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.Comment{}, id).Error
}

func (r *CommentRepo) IncrLikes(db *gorm.DB, id uint, delta int) error {
	return r.exec(db).Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("likes_count + ?", delta)).Error
}

func (r *CommentRepo) HideChildren(db *gorm.DB, parentID uint) error {
	return r.exec(db).Model(&model.Comment{}).
		Where("parent_id = ?", parentID).
		Update("hidden", true).Error
}

type AdminCommentQuery struct {
	Keyword    string
	Visibility string
	Page       int
	PageSize   int
}

func (r *CommentRepo) AdminList(ctx context.Context, q AdminCommentQuery) ([]model.Comment, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.Comment{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		d = d.Where("content LIKE ?", like)
	}
	if q.Visibility == "visible" {
		d = d.Where("hidden = ?", false)
	} else if q.Visibility == "hidden" {
		d = d.Where("hidden = ?", true)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Comment
	if err := d.Order("created_at desc, id desc").Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *CommentRepo) AdminFindByID(ctx context.Context, id uint) (*model.Comment, error) {
	var c model.Comment
	if err := r.db.WithContext(ctx).Preload("Author").First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}
