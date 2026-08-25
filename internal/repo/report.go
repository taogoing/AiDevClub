package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type ReportRepo struct{ db *gorm.DB }

func NewReportRepo(db *gorm.DB) *ReportRepo { return &ReportRepo{db: db} }

func (r *ReportRepo) Create(report *model.Report) error {
	return r.db.Create(report).Error
}

func (r *ReportRepo) FindByID(id uint) (*model.Report, error) {
	var report model.Report
	if err := r.db.First(&report, id).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepo) List(ctx context.Context, status model.ReportStatus, page, pageSize int) ([]model.Report, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.Report{})
	if status != "" {
		d = d.Where("status = ?", status)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Report
	if err := d.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ReportRepo) ListByReporter(ctx context.Context, reporterID uint, page, pageSize int) ([]model.Report, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.Report{}).Where("reporter_id = ?", reporterID)
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Report
	if err := d.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ReportRepo) Update(report *model.Report) error {
	return r.db.Save(report).Error
}

func (r *ReportRepo) CountByStatus(ctx context.Context, status model.ReportStatus) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.Report{}).Where("status = ?", status).Count(&total).Error
	return total, err
}
