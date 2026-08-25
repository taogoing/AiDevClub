package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type AdminLogRepo struct{ db *gorm.DB }

func NewAdminLogRepo(db *gorm.DB) *AdminLogRepo { return &AdminLogRepo{db: db} }

func (r *AdminLogRepo) Create(log *model.AdminLog) error {
	return r.db.Create(log).Error
}

func (r *AdminLogRepo) List(ctx context.Context, action model.AdminLogAction, page, pageSize int) ([]model.AdminLog, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.AdminLog{})
	if action != "" {
		d = d.Where("action = ?", action)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AdminLog
	if err := d.Order("created_at desc, id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
