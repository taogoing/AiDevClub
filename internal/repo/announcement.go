package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type AnnouncementRepo struct{ db *gorm.DB }

func NewAnnouncementRepo(db *gorm.DB) *AnnouncementRepo { return &AnnouncementRepo{db: db} }

func (r *AnnouncementRepo) Create(ann *model.Announcement) error {
	return r.db.Create(ann).Error
}

func (r *AnnouncementRepo) List(ctx context.Context, page, pageSize int) ([]model.Announcement, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.Announcement{})
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Announcement
	if err := d.Order("created_at desc, id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
