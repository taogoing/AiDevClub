package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type ReportService struct {
	repo        *repo.ReportRepo
	adminSvc    *AdminService
	adminLogSvc *AdminLogService
	notifSvc    *NotificationService
}

func NewReportService(
	repo *repo.ReportRepo,
	adminSvc *AdminService,
	adminLogSvc *AdminLogService,
	notifSvc *NotificationService,
) *ReportService {
	return &ReportService{repo: repo, adminSvc: adminSvc, adminLogSvc: adminLogSvc, notifSvc: notifSvc}
}

func (s *ReportService) List(ctx context.Context, status model.ReportStatus, page, pageSize int) (*ReportListResult, error) {
	return &ReportListResult{List: []ReportItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
}

func (s *ReportService) Resolve(ctx context.Context, adminID, reportID uint, action, result string) error {
	return nil
}
