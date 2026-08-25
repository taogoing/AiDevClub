package service

import (
	"context"
	"net/http"
	"time"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var ErrReportNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeReportNotFound, "举报不存在")

type ReportService struct {
	repo             *repo.ReportRepo
	articles         *repo.ArticleRepo
	skills           *repo.SkillRepo
	mcpServers       *repo.McpServerRepo
	comments         *repo.CommentRepo
	resourceComments *repo.ResourceCommentRepo
	adminSvc         *AdminService
	adminLogSvc      *AdminLogService
	notifSvc         *NotificationService
}

func NewReportService(
	reportRepo *repo.ReportRepo,
	articles *repo.ArticleRepo,
	skills *repo.SkillRepo,
	mcpServers *repo.McpServerRepo,
	comments *repo.CommentRepo,
	resourceComments *repo.ResourceCommentRepo,
	adminSvc *AdminService,
	adminLogSvc *AdminLogService,
	notifSvc *NotificationService,
) *ReportService {
	return &ReportService{
		repo: reportRepo, articles: articles, skills: skills, mcpServers: mcpServers,
		comments: comments, resourceComments: resourceComments,
		adminSvc: adminSvc, adminLogSvc: adminLogSvc, notifSvc: notifSvc,
	}
}

func (s *ReportService) Create(ctx context.Context, userID uint, targetType string, targetID uint, reason model.ReportReason, description string) (*model.Report, error) {
	if err := s.validateTarget(targetType, targetID); err != nil {
		return nil, err
	}
	report := &model.Report{
		ReporterID:  userID,
		TargetType:  targetType,
		TargetID:    targetID,
		Reason:      reason,
		Description: description,
		Status:      model.ReportStatusPending,
	}
	if err := s.repo.Create(report); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *ReportService) List(ctx context.Context, status model.ReportStatus, page, pageSize int) (*ReportListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := s.repo.List(ctx, status, page, pageSize)
	if err != nil {
		return nil, err
	}
	out := &ReportListResult{
		List: make([]ReportItem, 0, len(list)),
		Total: total, Page: page, PageSize: pageSize,
	}
	for _, r := range list {
		out.List = append(out.List, toReportItem(r))
	}
	return out, nil
}

func (s *ReportService) ListByReporter(ctx context.Context, reporterID uint, page, pageSize int) (*ReportListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := s.repo.ListByReporter(ctx, reporterID, page, pageSize)
	if err != nil {
		return nil, err
	}
	out := &ReportListResult{
		List: make([]ReportItem, 0, len(list)),
		Total: total, Page: page, PageSize: pageSize,
	}
	for _, r := range list {
		out.List = append(out.List, toReportItem(r))
	}
	return out, nil
}

func (s *ReportService) Resolve(ctx context.Context, adminID, reportID uint, action, result string) error {
	report, err := s.repo.FindByID(reportID)
	if err != nil {
		return ErrReportNotFound
	}
	if report.Status != model.ReportStatusPending {
		return platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "该举报已处理")
	}

	adminTargetType := mapReportTargetType(report.TargetType)

	now := time.Now()
	report.HandlerID = adminID
	report.HandleResult = result
	report.ResolvedAt = &now

	switch action {
	case "hide":
		report.Status = model.ReportStatusResolved
	case "unhide":
		report.Status = model.ReportStatusResolved
	case "dismiss":
		report.Status = model.ReportStatusDismissed
	default:
		return platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "不支持的处理操作")
	}

	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		switch action {
		case "hide":
			if err := s.adminSvc.HideContentTx(tx, adminTargetType, report.TargetID); err != nil {
				return err
			}
		case "unhide":
			if err := s.adminSvc.UnhideContentTx(tx, adminTargetType, report.TargetID); err != nil {
				return err
			}
		}
		return tx.Save(report).Error
	})
	if err != nil {
		return err
	}

	s.sendResolveNotifications(ctx, report, adminID, action)

	_ = s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionResolveReport, report.TargetType, report.ID, map[string]interface{}{
		"action": action, "result": result,
	})

	return nil
}

func (s *ReportService) validateTarget(targetType string, targetID uint) error {
	switch targetType {
	case "article":
		if _, err := s.articles.FindByID(nil, targetID); err != nil {
			return ErrArticleNotFound
		}
	case "skill":
		if _, err := s.skills.FindByID(nil, targetID); err != nil {
			return ErrSkillNotFound
		}
	case "mcp_server":
		if _, err := s.mcpServers.FindByID(nil, targetID); err != nil {
			return ErrMcpServerNotFound
		}
	case "article_comment":
		if _, err := s.comments.FindByID(nil, targetID); err != nil {
			return ErrCommentNotFound
		}
	case "resource_comment":
		if _, err := s.resourceComments.FindByID(nil, targetID); err != nil {
			return ErrResCommentNotFound
		}
	default:
		return ErrBadParam
	}
	return nil
}

func (s *ReportService) getTargetAuthorID(targetType string, targetID uint) (uint, error) {
	switch targetType {
	case "article":
		a, err := s.articles.FindByID(nil, targetID)
		if err != nil {
			return 0, err
		}
		return a.AuthorID, nil
	case "skill":
		sk, err := s.skills.FindByID(nil, targetID)
		if err != nil {
			return 0, err
		}
		return sk.AuthorID, nil
	case "mcp_server":
		ms, err := s.mcpServers.FindByID(nil, targetID)
		if err != nil {
			return 0, err
		}
		return ms.AuthorID, nil
	case "article_comment":
		c, err := s.comments.FindByID(nil, targetID)
		if err != nil {
			return 0, err
		}
		return c.AuthorID, nil
	case "resource_comment":
		rc, err := s.resourceComments.FindByID(nil, targetID)
		if err != nil {
			return 0, err
		}
		return rc.AuthorID, nil
	}
	return 0, nil
}

func (s *ReportService) sendResolveNotifications(ctx context.Context, report *model.Report, adminID uint, action string) {
	authorID, _ := s.getTargetAuthorID(report.TargetType, report.TargetID)

	switch action {
	case "hide":
		_ = s.notifSvc.Create(ctx, report.ReporterID, model.NotifTypeReportResolved,
			"举报已处理", "你的举报已处理，违规内容已被隐藏",
			report.TargetType, report.TargetID, adminID)
		if authorID > 0 {
			_ = s.notifSvc.Create(ctx, authorID, model.NotifTypeReportResolved,
				"内容被隐藏", "你的内容因违规被隐藏",
				report.TargetType, report.TargetID, adminID)
		}
	case "unhide":
		if authorID > 0 {
			_ = s.notifSvc.Create(ctx, authorID, model.NotifTypeReportResolved,
				"内容已恢复", "你的内容已恢复",
				report.TargetType, report.TargetID, adminID)
		}
	case "dismiss":
		_ = s.notifSvc.Create(ctx, report.ReporterID, model.NotifTypeReportResolved,
			"举报已驳回", "你的举报已驳回，内容未违规",
			report.TargetType, report.TargetID, adminID)
	}
}

func mapReportTargetType(reportTargetType string) string {
	if reportTargetType == "article_comment" {
		return "comment"
	}
	return reportTargetType
}

func toReportItem(r model.Report) ReportItem {
	return ReportItem{
		ID: r.ID, ReporterID: r.ReporterID,
		TargetType: r.TargetType, TargetID: r.TargetID,
		Reason: string(r.Reason), Description: r.Description,
		Status: string(r.Status), HandlerID: r.HandlerID,
		HandleResult: r.HandleResult, CreatedAt: r.CreatedAt,
		ResolvedAt: r.ResolvedAt,
	}
}
