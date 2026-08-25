package service

import (
	"context"
	"net/http"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

type AdminService struct {
	users          *repo.UserRepo
	articles       *repo.ArticleRepo
	skills         *repo.SkillRepo
	mcpServers     *repo.McpServerRepo
	comments       *repo.CommentRepo
	resourceComments *repo.ResourceCommentRepo
	reportRepo     *repo.ReportRepo
	announcementRepo *repo.AnnouncementRepo
	adminLogSvc    *AdminLogService
	notifSvc       *NotificationService
}

func NewAdminService(
	users *repo.UserRepo,
	articles *repo.ArticleRepo,
	skills *repo.SkillRepo,
	mcpServers *repo.McpServerRepo,
	comments *repo.CommentRepo,
	resourceComments *repo.ResourceCommentRepo,
	reportRepo *repo.ReportRepo,
	announcementRepo *repo.AnnouncementRepo,
	adminLogSvc *AdminLogService,
	notifSvc *NotificationService,
) *AdminService {
	return &AdminService{
		users: users, articles: articles, skills: skills, mcpServers: mcpServers,
		comments: comments, resourceComments: resourceComments,
		reportRepo: reportRepo, announcementRepo: announcementRepo,
		adminLogSvc: adminLogSvc, notifSvc: notifSvc,
	}
}

func (s *AdminService) Dashboard(ctx context.Context) (*DashboardData, error) {
	data := &DashboardData{}
	if c, err := s.users.Count(); err == nil {
		data.TotalUsers = c
	}
	if c, err := s.articles.Count(ctx, repo.ArticleQuery{Page: 1, PageSize: 1}); err == nil {
		data.TotalArticles = c
	}
	if c, err := s.skills.Count(ctx, repo.SkillQuery{Page: 1, PageSize: 1}); err == nil {
		data.TotalSkills = c
	}
	if c, err := s.mcpServers.Count(ctx, repo.McpServerQuery{Page: 1, PageSize: 1}); err == nil {
		data.TotalMcpServers = c
	}
	if c, err := s.skills.CountByStatus(ctx, model.ResourceStatusPendingReview); err == nil {
		data.PendingSkills = c
	}
	if c, err := s.mcpServers.CountByStatus(ctx, model.ResourceStatusPendingReview); err == nil {
		data.PendingMcpServers = c
	}
	if c, err := s.reportRepo.CountByStatus(ctx, model.ReportStatusPending); err == nil {
		data.PendingReports = c
	}
	return data, nil
}

func (s *AdminService) HideContent(targetType string, targetID uint) error {
	switch targetType {
	case "article":
		return s.articles.DB().Model(&model.Article{}).Where("id = ?", targetID).Update("hidden", true).Error
	case "skill":
		return s.skills.DB().Model(&model.Skill{}).Where("id = ?", targetID).Update("hidden", true).Error
	case "mcp_server":
		return s.mcpServers.DB().Model(&model.McpServer{}).Where("id = ?", targetID).Update("hidden", true).Error
	case "comment":
		if err := s.comments.DB().Model(&model.Comment{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		return s.comments.DB().Model(&model.Comment{}).Where("parent_id = ?", targetID).Update("hidden", true).Error
	case "resource_comment":
		if err := s.resourceComments.DB().Model(&model.ResourceComment{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		return s.resourceComments.DB().Model(&model.ResourceComment{}).Where("parent_id = ?", targetID).Update("hidden", true).Error
	}
	return platform.NewBizError(http.StatusBadRequest, platform.CodeParamError, "不支持的目标类型")
}

func (s *AdminService) UnhideContent(targetType string, targetID uint) error {
	switch targetType {
	case "article":
		return s.articles.DB().Model(&model.Article{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "skill":
		return s.skills.DB().Model(&model.Skill{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "mcp_server":
		return s.mcpServers.DB().Model(&model.McpServer{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "comment":
		return s.comments.DB().Model(&model.Comment{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "resource_comment":
		return s.resourceComments.DB().Model(&model.ResourceComment{}).Where("id = ?", targetID).Update("hidden", false).Error
	}
	return platform.NewBizError(http.StatusBadRequest, platform.CodeParamError, "不支持的目标类型")
}

func (s *AdminService) HideArticle(ctx context.Context, adminID, articleID uint) error {
	if err := s.articles.DB().Model(&model.Article{}).Where("id = ?", articleID).Update("hidden", true).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "article", articleID, nil)
}

func (s *AdminService) UnhideArticle(ctx context.Context, adminID, articleID uint) error {
	if err := s.articles.DB().Model(&model.Article{}).Where("id = ?", articleID).Update("hidden", false).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionUnhideContent, "article", articleID, nil)
}

func (s *AdminService) HideSkill(ctx context.Context, adminID, skillID uint) error {
	if err := s.skills.DB().Model(&model.Skill{}).Where("id = ?", skillID).Update("hidden", true).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "skill", skillID, nil)
}

func (s *AdminService) UnhideSkill(ctx context.Context, adminID, skillID uint) error {
	if err := s.skills.DB().Model(&model.Skill{}).Where("id = ?", skillID).Update("hidden", false).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionUnhideContent, "skill", skillID, nil)
}

func (s *AdminService) HideMcpServer(ctx context.Context, adminID, mcpServerID uint) error {
	if err := s.mcpServers.DB().Model(&model.McpServer{}).Where("id = ?", mcpServerID).Update("hidden", true).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "mcp_server", mcpServerID, nil)
}

func (s *AdminService) UnhideMcpServer(ctx context.Context, adminID, mcpServerID uint) error {
	if err := s.mcpServers.DB().Model(&model.McpServer{}).Where("id = ?", mcpServerID).Update("hidden", false).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionUnhideContent, "mcp_server", mcpServerID, nil)
}

func (s *AdminService) ReviewSkill(ctx context.Context, adminID, skillID uint, approved bool, reason string) error {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	if sk.Status != model.ResourceStatusPendingReview {
		return platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "当前状态不可审核")
	}
	var newStatus model.ResourceStatus
	var action model.AdminLogAction
	var notifType model.NotifType
	if approved {
		newStatus = model.ResourceStatusPublished
		action = model.AdminLogActionApproveResource
		notifType = model.NotifTypeResourceApproved
		now := sk.UpdatedAt
		sk.Status = newStatus
		sk.PublishedAt = &now
	} else {
		newStatus = model.ResourceStatusRejected
		action = model.AdminLogActionRejectResource
		notifType = model.NotifTypeResourceRejected
		sk.Status = newStatus
	}
	if err := s.skills.Update(nil, sk); err != nil {
		return err
	}
	_ = s.adminLogSvc.Log(ctx, adminID, action, "skill", skillID, map[string]interface{}{"reason": reason})
	_ = s.notifSvc.Create(ctx, sk.AuthorID, notifType, "资源审核结果", "您的 Skill「"+sk.Name+"」"+reason, "skill", skillID, adminID)
	return nil
}

func (s *AdminService) ReviewMcpServer(ctx context.Context, adminID, mcpServerID uint, approved bool, reason string) error {
	ms, err := s.mcpServers.FindByID(nil, mcpServerID)
	if err != nil {
		return ErrMcpServerNotFound
	}
	if ms.Status != model.ResourceStatusPendingReview {
		return platform.NewBizError(http.StatusBadRequest, platform.CodeBizError, "当前状态不可审核")
	}
	var newStatus model.ResourceStatus
	var action model.AdminLogAction
	var notifType model.NotifType
	if approved {
		newStatus = model.ResourceStatusPublished
		action = model.AdminLogActionApproveResource
		notifType = model.NotifTypeResourceApproved
		now := ms.UpdatedAt
		ms.Status = newStatus
		ms.PublishedAt = &now
	} else {
		newStatus = model.ResourceStatusRejected
		action = model.AdminLogActionRejectResource
		notifType = model.NotifTypeResourceRejected
		ms.Status = newStatus
	}
	if err := s.mcpServers.Update(nil, ms); err != nil {
		return err
	}
	_ = s.adminLogSvc.Log(ctx, adminID, action, "mcp_server", mcpServerID, map[string]interface{}{"reason": reason})
	_ = s.notifSvc.Create(ctx, ms.AuthorID, notifType, "资源审核结果", "您的 MCP Server「"+ms.Name+"」"+reason, "mcp_server", mcpServerID, adminID)
	return nil
}

func (s *AdminService) CreateAnnouncement(ctx context.Context, adminID uint, title, content string) (*model.Announcement, error) {
	if title == "" || content == "" {
		return nil, ErrBadParam
	}
	ann := &model.Announcement{
		Title:   title,
		Content: content,
		AdminID: adminID,
	}
	if err := s.announcementRepo.Create(ann); err != nil {
		return nil, err
	}
	_ = s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionCreateAnnouncement, "announcement", ann.ID, nil)
	_ = s.notifSvc.CreateBatchForAllUsers(ctx, model.NotifTypeAnnouncement, title, content, adminID)
	return ann, nil
}

func (s *AdminService) ListAnnouncements(ctx context.Context, page, pageSize int) (*AnnouncementListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := s.announcementRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	out := &AnnouncementListResult{
		List:     make([]AnnouncementItem, 0, len(list)),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
	for _, a := range list {
		out.List = append(out.List, AnnouncementItem{
			ID: a.ID, Title: a.Title, Content: a.Content,
			AdminID: a.AdminID, CreatedAt: a.CreatedAt,
		})
	}
	return out, nil
}
