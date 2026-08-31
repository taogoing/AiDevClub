package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

type AdminService struct {
	users            *repo.UserRepo
	articles         *repo.ArticleRepo
	skills           *repo.SkillRepo
	mcpServers       *repo.McpServerRepo
	comments         *repo.CommentRepo
	resourceComments *repo.ResourceCommentRepo
	reportRepo       *repo.ReportRepo
	announcementRepo *repo.AnnouncementRepo
	adminLogSvc      *AdminLogService
	notifSvc         *NotificationService
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
	c, err := s.users.Count()
	if err != nil {
		return nil, err
	}
	data.TotalUsers = c

	ac, err := s.articles.Count(ctx, repo.ArticleQuery{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	data.TotalArticles = ac

	sc, err := s.skills.Count(ctx, repo.SkillQuery{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	data.TotalSkills = sc

	mc, err := s.mcpServers.Count(ctx, repo.McpServerQuery{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	data.TotalMcpServers = mc

	ps, err := s.skills.CountByStatus(ctx, model.ResourceStatusPendingReview)
	if err != nil {
		return nil, err
	}
	data.PendingSkills = ps

	pm, err := s.mcpServers.CountByStatus(ctx, model.ResourceStatusPendingReview)
	if err != nil {
		return nil, err
	}
	data.PendingMcpServers = pm

	pr, err := s.reportRepo.CountByStatus(ctx, model.ReportStatusPending)
	if err != nil {
		return nil, err
	}
	data.PendingReports = pr

	return data, nil
}

type AdminUserItem struct {
	ID          uint           `json:"id"`
	Email       string         `json:"email"`
	Nickname    string         `json:"nickname"`
	AvatarURL   string         `json:"avatar_url"`
	Role        model.UserRole `json:"role"`
	RoleMutable bool           `json:"role_mutable"`
	CreatedAt   time.Time      `json:"created_at"`
}

type AdminUserListResult struct {
	List     []AdminUserItem `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

func (s *AdminService) ListUsers(ctx context.Context, adminID uint, keyword string, role model.UserRole, page, pageSize int) (*AdminUserListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := s.users.ListUsers(ctx, repo.UserQuery{
		Keyword:  keyword,
		Role:     role,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	out := &AdminUserListResult{
		List:     make([]AdminUserItem, 0, len(list)),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
	for _, u := range list {
		out.List = append(out.List, AdminUserItem{
			ID:          u.ID,
			Email:       u.Email,
			Nickname:    u.Nickname,
			AvatarURL:   u.AvatarURL,
			Role:        u.Role,
			RoleMutable: u.ID != adminID,
			CreatedAt:   u.CreatedAt,
		})
	}
	return out, nil
}

func (s *AdminService) UpdateUserRole(ctx context.Context, adminID, targetID uint, role model.UserRole) error {
	if adminID == targetID {
		return platform.NewBizError(http.StatusForbidden, platform.CodeForbidden, "不能修改自己的角色")
	}
	target, err := s.users.FindByIDWithContext(ctx, targetID)
	if err != nil {
		return ErrUserNotFound
	}
	if target.Role == role {
		return nil
	}
	if err := s.users.UpdateRole(targetID, role); err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogAction("update_user_role"), "user", targetID, map[string]interface{}{
		"old_role": string(target.Role),
		"new_role": string(role),
	})
}

type AdminArticleQuery struct {
	Keyword    string
	Visibility string
	AuthorID   *uint
	Page       int
	PageSize   int
}

type AdminArticleItem struct {
	ID            uint        `json:"id"`
	Title         string      `json:"title"`
	Summary       string      `json:"summary"`
	Author        AuthorBrief `json:"author"`
	Views         int         `json:"views"`
	LikesCount    int         `json:"likes_count"`
	CommentsCount int         `json:"comments_count"`
	Hidden        bool        `json:"hidden"`
	PublishedAt   *time.Time  `json:"published_at"`
	CreatedAt     time.Time   `json:"created_at"`
}

type AdminArticleListResult struct {
	List     []AdminArticleItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

func (s *AdminService) ListArticles(ctx context.Context, q AdminArticleQuery) (*AdminArticleListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	list, total, err := s.articles.AdminList(ctx, repo.AdminArticleQuery{
		Keyword:    q.Keyword,
		Visibility: q.Visibility,
		AuthorID:   q.AuthorID,
		Page:       q.Page,
		PageSize:   q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	out := &AdminArticleListResult{
		List:     make([]AdminArticleItem, 0, len(list)),
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}
	for _, a := range list {
		out.List = append(out.List, AdminArticleItem{
			ID:            a.ID,
			Title:         a.Title,
			Summary:       a.Summary,
			Author:        AuthorBrief{ID: a.Author.ID, Nickname: a.Author.Nickname, AvatarURL: a.Author.AvatarURL},
			Views:         a.Views,
			LikesCount:    a.LikesCount,
			CommentsCount: a.CommentsCount,
			Hidden:        a.Hidden,
			PublishedAt:   a.PublishedAt,
			CreatedAt:     a.CreatedAt,
		})
	}
	return out, nil
}

type AdminArticleDetail struct {
	ID             uint        `json:"id"`
	Title          string      `json:"title"`
	Summary        string      `json:"summary"`
	Content        string      `json:"content"`
	Author         AuthorBrief `json:"author"`
	Views          int         `json:"views"`
	LikesCount     int         `json:"likes_count"`
	FavoritesCount int         `json:"favorites_count"`
	CommentsCount  int         `json:"comments_count"`
	Hidden         bool        `json:"hidden"`
	PublishedAt    *time.Time  `json:"published_at"`
	CreatedAt      time.Time   `json:"created_at"`
}

func (s *AdminService) GetArticle(ctx context.Context, id uint) (*AdminArticleDetail, error) {
	a, err := s.articles.AdminFindByID(ctx, id)
	if err != nil {
		return nil, ErrArticleNotFound
	}
	return &AdminArticleDetail{
		ID:             a.ID,
		Title:          a.Title,
		Summary:        a.Summary,
		Content:        a.Content,
		Author:         AuthorBrief{ID: a.Author.ID, Nickname: a.Author.Nickname, AvatarURL: a.Author.AvatarURL},
		Views:          a.Views,
		LikesCount:     a.LikesCount,
		FavoritesCount: a.FavoritesCount,
		CommentsCount:  a.CommentsCount,
		Hidden:         a.Hidden,
		PublishedAt:    a.PublishedAt,
		CreatedAt:      a.CreatedAt,
	}, nil
}

type AdminCommentItem struct {
	ID         uint        `json:"id"`
	ArticleID  uint        `json:"article_id"`
	ParentID   uint        `json:"parent_id"`
	Author     AuthorBrief `json:"author"`
	Content    string      `json:"content"`
	LikesCount int         `json:"likes_count"`
	Hidden     bool        `json:"hidden"`
	CreatedAt  time.Time   `json:"created_at"`
}

type AdminCommentListResult struct {
	List     []AdminCommentItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

func (s *AdminService) ListArticleComments(ctx context.Context, q AdminCommentQuery) (*AdminCommentListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	list, total, err := s.comments.AdminList(ctx, repo.AdminCommentQuery{
		Keyword:    q.Keyword,
		Visibility: q.Visibility,
		Page:       q.Page,
		PageSize:   q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	out := &AdminCommentListResult{
		List:     make([]AdminCommentItem, 0, len(list)),
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}
	for _, c := range list {
		var parentID uint
		if c.ParentID != nil {
			parentID = *c.ParentID
		}
		item := AdminCommentItem{
			ID:         c.ID,
			ArticleID:  c.ArticleID,
			ParentID:   parentID,
			Content:    c.Content,
			LikesCount: c.LikesCount,
			Hidden:     c.Hidden,
			CreatedAt:  c.CreatedAt,
		}
		if c.Author != nil {
			item.Author = AuthorBrief{ID: c.Author.ID, Nickname: c.Author.Nickname, AvatarURL: c.Author.AvatarURL}
		}
		out.List = append(out.List, item)
	}
	return out, nil
}

func (s *AdminService) HideComment(ctx context.Context, adminID, id uint) error {
	if err := s.comments.DB().Model(&model.Comment{}).Where("id = ?", id).Update("hidden", true).Error; err != nil {
		return err
	}
	_ = s.comments.DB().Model(&model.Comment{}).Where("parent_id = ?", id).Update("hidden", true).Error
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "comment", id, nil)
}

func (s *AdminService) UnhideComment(ctx context.Context, adminID, id uint) error {
	if err := s.comments.DB().Model(&model.Comment{}).Where("id = ?", id).Update("hidden", false).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionUnhideContent, "comment", id, nil)
}

type AdminResourceCommentItem struct {
	ID           uint        `json:"id"`
	ResourceType string      `json:"resource_type"`
	ResourceID   uint        `json:"resource_id"`
	ParentID     uint        `json:"parent_id"`
	Author       AuthorBrief `json:"author"`
	Content      string      `json:"content"`
	LikesCount   int         `json:"likes_count"`
	Hidden       bool        `json:"hidden"`
	CreatedAt    time.Time   `json:"created_at"`
}

type AdminResourceCommentListResult struct {
	List     []AdminResourceCommentItem `json:"list"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}

func (s *AdminService) ListResourceComments(ctx context.Context, q AdminResourceCommentQuery) (*AdminResourceCommentListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	list, total, err := s.resourceComments.AdminList(ctx, repo.AdminResourceCommentQuery{
		Keyword:      q.Keyword,
		Visibility:   q.Visibility,
		ResourceType: q.ResourceType,
		Page:         q.Page,
		PageSize:     q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	out := &AdminResourceCommentListResult{
		List:     make([]AdminResourceCommentItem, 0, len(list)),
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}
	for _, c := range list {
		var parentID uint
		if c.ParentID != nil {
			parentID = *c.ParentID
		}
		item := AdminResourceCommentItem{
			ID:           c.ID,
			ResourceType: c.ResourceType,
			ResourceID:   c.ResourceID,
			ParentID:     parentID,
			Content:      c.Content,
			LikesCount:   c.LikesCount,
			Hidden:       c.Hidden,
			CreatedAt:    c.CreatedAt,
		}
		if c.Author != nil {
			item.Author = AuthorBrief{ID: c.Author.ID, Nickname: c.Author.Nickname, AvatarURL: c.Author.AvatarURL}
		}
		out.List = append(out.List, item)
	}
	return out, nil
}

func (s *AdminService) HideResourceComment(ctx context.Context, adminID, id uint) error {
	if err := s.resourceComments.DB().Model(&model.ResourceComment{}).Where("id = ?", id).Update("hidden", true).Error; err != nil {
		return err
	}
	_ = s.resourceComments.DB().Model(&model.ResourceComment{}).Where("parent_id = ?", id).Update("hidden", true).Error
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "resource_comment", id, nil)
}

func (s *AdminService) UnhideResourceComment(ctx context.Context, adminID, id uint) error {
	if err := s.resourceComments.DB().Model(&model.ResourceComment{}).Where("id = ?", id).Update("hidden", false).Error; err != nil {
		return err
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionUnhideContent, "resource_comment", id, nil)
}

type AdminCommentQuery struct {
	Keyword    string
	Visibility string
	Page       int
	PageSize   int
}

type AdminResourceCommentQuery struct {
	Keyword      string
	Visibility   string
	ResourceType string
	Page         int
	PageSize     int
}

type AdminResourceItem struct {
	ID          uint        `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Author      AuthorBrief `json:"author"`
	Status      string      `json:"status"`
	Hidden      bool        `json:"hidden"`
	Views       int         `json:"views"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type AdminResourceListResult struct {
	List     []AdminResourceItem `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

func (s *AdminService) ListSkills(ctx context.Context, q AdminResourceQuery) (*AdminResourceListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	list, total, err := s.skills.AdminList(ctx, repo.AdminResourceQuery{
		Keyword:  q.Keyword,
		Status:   q.Status,
		AuthorID: q.AuthorID,
		TagID:    q.TagID,
		Page:     q.Page,
		PageSize: q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	out := &AdminResourceListResult{
		List:     make([]AdminResourceItem, 0, len(list)),
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}
	for _, sk := range list {
		out.List = append(out.List, AdminResourceItem{
			ID:          sk.ID,
			Name:        sk.Name,
			Description: sk.Description,
			Author:      AuthorBrief{ID: sk.Author.ID, Nickname: sk.Author.Nickname, AvatarURL: sk.Author.AvatarURL},
			Status:      string(sk.Status),
			Hidden:      sk.Hidden,
			Views:       sk.Views,
			CreatedAt:   sk.CreatedAt,
			UpdatedAt:   sk.UpdatedAt,
		})
	}
	return out, nil
}

type AdminSkillDetail struct {
	AdminResourceItem
	RepoURL      string `json:"repo_url"`
	SkillMD      string `json:"skill_md"`
	RejectReason string `json:"reject_reason,omitempty"`
}

func (s *AdminService) GetSkill(ctx context.Context, id uint) (*AdminSkillDetail, error) {
	sk, err := s.skills.AdminFindByID(ctx, id)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	return &AdminSkillDetail{
		AdminResourceItem: AdminResourceItem{
			ID:          sk.ID,
			Name:        sk.Name,
			Description: sk.Description,
			Author:      AuthorBrief{ID: sk.Author.ID, Nickname: sk.Author.Nickname, AvatarURL: sk.Author.AvatarURL},
			Status:      string(sk.Status),
			Hidden:      sk.Hidden,
			Views:       sk.Views,
			CreatedAt:   sk.CreatedAt,
			UpdatedAt:   sk.UpdatedAt,
		},
		RepoURL:      sk.RepoURL,
		SkillMD:      sk.SkillMD,
		RejectReason: sk.RejectReason,
	}, nil
}

func (s *AdminService) ListMCPServers(ctx context.Context, q AdminResourceQuery) (*AdminResourceListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	list, total, err := s.mcpServers.AdminList(ctx, repo.AdminResourceQuery{
		Keyword:  q.Keyword,
		Status:   q.Status,
		AuthorID: q.AuthorID,
		TagID:    q.TagID,
		Page:     q.Page,
		PageSize: q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	out := &AdminResourceListResult{
		List:     make([]AdminResourceItem, 0, len(list)),
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}
	for _, m := range list {
		out.List = append(out.List, AdminResourceItem{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Author:      AuthorBrief{ID: m.Author.ID, Nickname: m.Author.Nickname, AvatarURL: m.Author.AvatarURL},
			Status:      string(m.Status),
			Hidden:      m.Hidden,
			Views:       m.Views,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}
	return out, nil
}

type AdminMCPServerDetail struct {
	AdminResourceItem
	RepoURL       string            `json:"repo_url"`
	Installations []McpInstallation `json:"installations"`
	Readme        string            `json:"readme"`
	RejectReason  string            `json:"reject_reason,omitempty"`
}

func (s *AdminService) GetMCPServer(ctx context.Context, id uint) (*AdminMCPServerDetail, error) {
	m, err := s.mcpServers.AdminFindByID(ctx, id)
	if err != nil {
		return nil, ErrMcpServerNotFound
	}
	installations := []McpInstallation{}
	if m.InstallationsJSON != "" {
		if err := json.Unmarshal([]byte(m.InstallationsJSON), &installations); err != nil {
			return nil, err
		}
	}
	return &AdminMCPServerDetail{
		AdminResourceItem: AdminResourceItem{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Author:      AuthorBrief{ID: m.Author.ID, Nickname: m.Author.Nickname, AvatarURL: m.Author.AvatarURL},
			Status:      string(m.Status),
			Hidden:      m.Hidden,
			Views:       m.Views,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		},
		RepoURL:       m.RepoURL,
		Installations: installations,
		Readme:        m.Readme,
		RejectReason:  m.RejectReason,
	}, nil
}

type AdminResourceQuery struct {
	Keyword  string
	Status   string
	AuthorID *uint
	TagID    *uint
	Page     int
	PageSize int
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

func (s *AdminService) HideContentTx(tx *gorm.DB, targetType string, targetID uint) error {
	switch targetType {
	case "article":
		return tx.Model(&model.Article{}).Where("id = ?", targetID).Update("hidden", true).Error
	case "skill":
		return tx.Model(&model.Skill{}).Where("id = ?", targetID).Update("hidden", true).Error
	case "mcp_server":
		return tx.Model(&model.McpServer{}).Where("id = ?", targetID).Update("hidden", true).Error
	case "comment":
		if err := tx.Model(&model.Comment{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		return tx.Model(&model.Comment{}).Where("parent_id = ?", targetID).Update("hidden", true).Error
	case "resource_comment":
		if err := tx.Model(&model.ResourceComment{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		return tx.Model(&model.ResourceComment{}).Where("parent_id = ?", targetID).Update("hidden", true).Error
	}
	return platform.NewBizError(http.StatusBadRequest, platform.CodeParamError, "不支持的目标类型")
}

func (s *AdminService) UnhideContentTx(tx *gorm.DB, targetType string, targetID uint) error {
	switch targetType {
	case "article":
		return tx.Model(&model.Article{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "skill":
		return tx.Model(&model.Skill{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "mcp_server":
		return tx.Model(&model.McpServer{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "comment":
		return tx.Model(&model.Comment{}).Where("id = ?", targetID).Update("hidden", false).Error
	case "resource_comment":
		return tx.Model(&model.ResourceComment{}).Where("id = ?", targetID).Update("hidden", false).Error
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
		now := time.Now()
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
		now := time.Now()
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
