package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type ResourceCommentRepo struct{ db *gorm.DB }

func NewResourceCommentRepo(db *gorm.DB) *ResourceCommentRepo { return &ResourceCommentRepo{db: db} }

func (r *ResourceCommentRepo) DB() *gorm.DB { return r.db }

func (r *ResourceCommentRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *ResourceCommentRepo) Create(db *gorm.DB, c *model.ResourceComment) error {
	return r.exec(db).Create(c).Error
}

func (r *ResourceCommentRepo) FindByID(db *gorm.DB, id uint) (*model.ResourceComment, error) {
	var c model.ResourceComment
	if err := r.exec(db).Preload("Author").First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ResourceCommentRepo) ListByResource(ctx context.Context, resourceType string, resourceID uint) ([]model.ResourceComment, error) {
	var list []model.ResourceComment
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ? AND hidden = ?", resourceType, resourceID, false).
		Order("created_at asc, id asc").
		Find(&list).Error
	return list, err
}

func (r *ResourceCommentRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.ResourceComment{}, id).Error
}

func (r *ResourceCommentRepo) IncrLikes(db *gorm.DB, id uint, delta int) error {
	return r.exec(db).Model(&model.ResourceComment{}).Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("likes_count + ?", delta)).Error
}

func (r *ResourceCommentRepo) HideChildren(db *gorm.DB, parentID uint) error {
	return r.exec(db).Model(&model.ResourceComment{}).
		Where("parent_id = ?", parentID).
		Update("hidden", true).Error
}
