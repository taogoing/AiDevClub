package service

import (
	"time"

	"aidevclub/internal/model"
)

type CreateArticleInput struct {
	Title    string
	Summary  string
	Content  string
	Status   model.ArticleStatus
	TagIDs   []uint
	TagNames []string
}

type AuthorBrief struct {
	ID        uint   `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type TagBrief struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ArticleSummary struct {
	ID             uint        `json:"id"`
	Title          string      `json:"title"`
	Summary        string      `json:"summary"`
	Tags           []TagBrief  `json:"tags"`
	Author         AuthorBrief `json:"author"`
	Views          int         `json:"views"`
	LikesCount     int         `json:"likes_count"`
	FavoritesCount int         `json:"favorites_count"`
	CommentsCount  int         `json:"comments_count"`
	Status         string      `json:"status"`
	PublishedAt    *time.Time  `json:"published_at"`
	Pinned         bool        `json:"pinned"`
}

type ArticleListResult struct {
	List     []ArticleSummary `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type ArticleDetail struct {
	ArticleSummary
	Content   string `json:"content"`
	Liked     bool   `json:"liked"`
	Favorited bool   `json:"favorited"`
}

type HotArticleBrief struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Score int64  `json:"score"`
}

type HotSkillBrief struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Score int64  `json:"score"`
}

type HotMcpServerBrief struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Score int64  `json:"score"`
}

type ListQuery struct {
	Page     int
	PageSize int
	TagID    *uint
	Keyword  string
	AuthorID *uint
	Sort     string
}

type CommentItem struct {
	ID         uint          `json:"id"`
	ArticleID  uint          `json:"article_id"`
	AuthorID   uint          `json:"author_id"`
	Author     AuthorBrief   `json:"author"`
	Content    string        `json:"content"`
	LikesCount int           `json:"likes_count"`
	CreatedAt  time.Time     `json:"created_at"`
	Replies    []CommentItem `json:"replies"`
}

type SkillSummary struct {
	ID             uint        `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	RepoURL        string      `json:"repo_url"`
	Tags           []TagBrief  `json:"tags"`
	Author         AuthorBrief `json:"author"`
	Views          int         `json:"views"`
	LikesCount     int         `json:"likes_count"`
	FavoritesCount int         `json:"favorites_count"`
	CommentsCount  int         `json:"comments_count"`
	Status         string      `json:"status"`
	PublishedAt    *time.Time  `json:"published_at"`
}

type SkillListResult struct {
	List     []SkillSummary `json:"list"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type SkillDetail struct {
	SkillSummary
	SkillMD   string `json:"skill_md"`
	Liked     bool   `json:"liked"`
	Favorited bool   `json:"favorited"`
}

type CreateSkillInput struct {
	Name        string
	Description string
	RepoURL     string
	SkillMD     string
	TagIDs      []uint
	TagNames    []string
}

type SkillListQuery struct {
	Page     int
	PageSize int
	TagID    *uint
	Keyword  string
	AuthorID *uint
	Sort     string
}

type McpServerSummary struct {
	ID             uint        `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	RepoURL        string      `json:"repo_url"`
	Tags           []TagBrief  `json:"tags"`
	Author         AuthorBrief `json:"author"`
	Views          int         `json:"views"`
	LikesCount     int         `json:"likes_count"`
	FavoritesCount int         `json:"favorites_count"`
	CommentsCount  int         `json:"comments_count"`
	Status         string      `json:"status"`
	PublishedAt    *time.Time  `json:"published_at"`
}

type McpServerDetail struct {
	McpServerSummary
	Installations []McpInstallation `json:"installations"`
	Readme        string            `json:"readme"`
	Liked         bool              `json:"liked"`
	Favorited     bool              `json:"favorited"`
}

type McpServerListResult struct {
	List     []McpServerSummary `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type CreateMcpServerInput struct {
	Name          string
	Description   string
	RepoURL       string
	Installations []McpInstallation
	Readme        string
	TagIDs        []uint
	TagNames      []string
}

type McpInstallation struct {
	Client        string         `json:"client"`
	Command       string         `json:"command,omitempty"`
	Config        map[string]any `json:"config,omitempty"`
	WindowsConfig map[string]any `json:"windows_config,omitempty"`
}

type McpServerListQuery struct {
	Page     int
	PageSize int
	TagID    *uint
	Keyword  string
	AuthorID *uint
	Sort     string
}

type ResourceCommentItem struct {
	ID         uint                  `json:"id"`
	ResourceID uint                  `json:"resource_id"`
	AuthorID   uint                  `json:"author_id"`
	Author     AuthorBrief           `json:"author"`
	Content    string                `json:"content"`
	LikesCount int                   `json:"likes_count"`
	CreatedAt  time.Time             `json:"created_at"`
	Replies    []ResourceCommentItem `json:"replies"`
}

type NotificationItem struct {
	ID           uint        `json:"id"`
	Type         string      `json:"type"`
	Title        string      `json:"title"`
	Content      string      `json:"content"`
	ResourceType string      `json:"resource_type"`
	ResourceID   uint        `json:"resource_id"`
	Actor        AuthorBrief `json:"actor"`
	IsRead       bool        `json:"is_read"`
	CreatedAt    time.Time   `json:"created_at"`
}

type NotificationListResult struct {
	List     []NotificationItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type AdminLogItem struct {
	ID         uint        `json:"id"`
	AdminID    uint        `json:"admin_id"`
	Admin      AuthorBrief `json:"admin"`
	Action     string      `json:"action"`
	TargetType string      `json:"target_type"`
	TargetID   uint        `json:"target_id"`
	Detail     any         `json:"detail"`
	CreatedAt  time.Time   `json:"created_at"`
}

type AdminLogListResult struct {
	List     []AdminLogItem `json:"list"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type DashboardData struct {
	TotalUsers        int64 `json:"total_users"`
	TotalArticles     int64 `json:"total_articles"`
	TotalSkills       int64 `json:"total_skills"`
	TotalMcpServers   int64 `json:"total_mcp_servers"`
	PendingSkills     int64 `json:"pending_skills"`
	PendingMcpServers int64 `json:"pending_mcp_servers"`
	PendingReports    int64 `json:"pending_reports"`
}

type AnnouncementItem struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AdminID   uint      `json:"admin_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AnnouncementListResult struct {
	List     []AnnouncementItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type ReportItem struct {
	ID           uint       `json:"id"`
	ReporterID   uint       `json:"reporter_id"`
	TargetType   string     `json:"target_type"`
	TargetID     uint       `json:"target_id"`
	Reason       string     `json:"reason"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	HandlerID    uint       `json:"handler_id"`
	HandleResult string     `json:"handle_result"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
}

type ReportListResult struct {
	List     []ReportItem `json:"list"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}
