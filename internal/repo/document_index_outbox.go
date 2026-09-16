package repo

import (
	"aidevclub/internal/model"
	"context"
	"gorm.io/gorm"
	"time"
)

type DocumentIndexOutboxRepo struct{ db *gorm.DB }

func NewDocumentIndexOutboxRepo(db *gorm.DB) *DocumentIndexOutboxRepo {
	return &DocumentIndexOutboxRepo{db: db}
}
func (r *DocumentIndexOutboxRepo) Create(tx *gorm.DB, e *model.DocumentIndexOutboxEvent) error {
	return tx.Create(e).Error
}
func (r *DocumentIndexOutboxRepo) Pending(ctx context.Context, limit int) ([]model.DocumentIndexOutboxEvent, error) {
	var es []model.DocumentIndexOutboxEvent
	err := r.db.WithContext(ctx).Where("published_at IS NULL").Order("id").Limit(limit).Find(&es).Error
	return es, err
}
func (r *DocumentIndexOutboxRepo) MarkPublished(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.DocumentIndexOutboxEvent{}).Where("id=? AND published_at IS NULL", id).Update("published_at", now).Error
}
