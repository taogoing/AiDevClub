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

func (r *TagRepo) AdminCreate(ctx context.Context, name, description string) (*model.Tag, error) {
	t := &model.Tag{Name: name, Description: description, Enabled: true}
	err := r.db.WithContext(ctx).Create(t).Error
	return t, err
}

func (r *TagRepo) AdminUpdate(ctx context.Context, id uint, name, description string) error {
	return r.db.WithContext(ctx).
		Model(&model.Tag{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"name":        name,
			"description": description,
		}).Error
}

func (r *TagRepo) AdminPatch(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&model.Tag{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *TagRepo) AdminList(ctx context.Context, keyword, status string, page, pageSize int) ([]model.Tag, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Tag{})

	if keyword != "" {
		query = query.Where("name LIKE ?", keyword+"%")
	}

	switch status {
	case "enabled":
		query = query.Where("enabled = ?", true)
	case "disabled":
		query = query.Where("enabled = ?", false)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tags []model.Tag
	err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tags).Error

	return tags, total, err
}
