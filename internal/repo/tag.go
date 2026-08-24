package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type TagRepo struct{ db *gorm.DB }

func NewTagRepo(db *gorm.DB) *TagRepo { return &TagRepo{db: db} }

func (r *TagRepo) FindByID(ctx context.Context, id uint) (*model.Tag, error) {
	var t model.Tag
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TagRepo) FindByName(ctx context.Context, name string) (*model.Tag, error) {
	var t model.Tag
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TagRepo) Create(ctx context.Context, name string) (*model.Tag, error) {
	t := &model.Tag{Name: name, Enabled: true}
	err := r.db.WithContext(ctx).Create(t).Error
	return t, err
}

func (r *TagRepo) List(ctx context.Context, keyword string, limit int) ([]model.Tag, error) {
	q := r.db.WithContext(ctx).Where("enabled = ?", true)
	if keyword != "" {
		q = q.Where("name LIKE ?", keyword+"%")
	}
	var list []model.Tag
	err := q.Order("name asc").Limit(limit).Find(&list).Error
	return list, err
}

func (r *TagRepo) ListHot(ctx context.Context, limit int) ([]model.Tag, error) {
	var list []model.Tag
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("usage_count desc, id asc").
		Limit(limit).Find(&list).Error
	return list, err
}

func (r *TagRepo) IncrUsage(db *gorm.DB, tagID uint, delta int) error {
	return db.Model(&model.Tag{}).
		Where("id = ?", tagID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", delta)).Error
}
