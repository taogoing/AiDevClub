package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type NotificationRepo struct{ db *gorm.DB }

func NewNotificationRepo(db *gorm.DB) *NotificationRepo { return &NotificationRepo{db: db} }

func (r *NotificationRepo) Create(n *model.Notification) error {
	return r.db.Create(n).Error
}

func (r *NotificationRepo) CreateBatch(notifications []*model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	return r.db.Create(notifications).Error
}

func (r *NotificationRepo) List(ctx context.Context, userID uint, notifType string, page, pageSize int) ([]model.Notification, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.Notification{}).Where("user_id = ?", userID)
	if notifType != "" {
		d = d.Where("type = ?", notifType)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Notification
	if err := d.Order("created_at desc, id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, userID uint) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&total).Error
	return total, err
}

func (r *NotificationRepo) MarkRead(ctx context.Context, userID, notifID uint) error {
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Update("is_read", true).Error
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
