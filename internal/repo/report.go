package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type ReportRepo struct{ db *gorm.DB }

func NewReportRepo(db *gorm.DB) *ReportRepo { return &ReportRepo{db: db} }

func (r *ReportRepo) CountByStatus(ctx context.Context, status model.ReportStatus) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.Report{}).Where("status = ?", status).Count(&total).Error
	return total, err
}
