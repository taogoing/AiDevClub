package app

import (
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
)

type Services struct {
	UserRepo         *repo.UserRepo
	Auth             *service.AuthService
	Users            *service.UserService
	Articles         *service.ArticleService
	Comments         *service.CommentService
	Skills           *service.SkillService
	MCPServers       *service.McpServerService
	ResourceComments *service.ResourceCommentService
	Search           *service.SearchService
	Ranking          *service.RankingService
	Tags             *service.TagService
	Notifications    *service.NotificationService
	Reports          *service.ReportService
	Admin            *service.AdminService
	AdminLogs        *service.AdminLogService
}

func NewServices(infra *Infrastructure, cfg *platform.Config) *Services {
	users := repo.NewUserRepo(infra.DB)
	tokens := repo.NewTokenRepo(infra.Redis, cfg.RefreshTokenTTL)
	auth := service.NewAuthService(users, tokens, cfg)
	userService := service.NewUserService(users, tokens, cfg)

	tags := repo.NewTagRepo(infra.DB)
	articles := repo.NewArticleRepo(infra.DB)
	comments := repo.NewCommentRepo(infra.DB)
	interactions := repo.NewInteractionRepo(infra.DB)
	notificationRepo := repo.NewNotificationRepo(infra.DB)
	notifications := service.NewNotificationService(notificationRepo, users)

	tagService := service.NewTagService(tags)
	articleService := service.NewArticleService(articles, tags, interactions, cfg, notifications)
	commentService := service.NewCommentService(comments, articles, interactions, users, notifications)

	searchRepo := repo.NewSearchRepo(infra.DB)
	search := service.NewSearchService(searchRepo)

	skills := repo.NewSkillRepo(infra.DB)
	mcpServers := repo.NewMcpServerRepo(infra.DB)
	resourceComments := repo.NewResourceCommentRepo(infra.DB)
	skillService := service.NewSkillService(skills, tags, interactions, cfg, notifications)
	mcpServerService := service.NewMcpServerService(mcpServers, tags, interactions, cfg, notifications)
	resourceCommentService := service.NewResourceCommentService(resourceComments, skills, mcpServers, interactions, users, notifications)

	adminLogRepo := repo.NewAdminLogRepo(infra.DB)
	announcementRepo := repo.NewAnnouncementRepo(infra.DB)
	reportsRepo := repo.NewReportRepo(infra.DB)
	adminLogs := service.NewAdminLogService(adminLogRepo, users)
	admin := service.NewAdminService(
		users, articles, skills, mcpServers, comments, resourceComments, reportsRepo,
		announcementRepo, adminLogs, notifications,
	)
	reports := service.NewReportService(
		reportsRepo, articles, skills, mcpServers, comments, resourceComments, admin, adminLogs, notifications,
	)
	ranking := service.NewRankingService(articles, skills, mcpServers)

	return &Services{
		UserRepo:         users,
		Auth:             auth,
		Users:            userService,
		Articles:         articleService,
		Comments:         commentService,
		Skills:           skillService,
		MCPServers:       mcpServerService,
		ResourceComments: resourceCommentService,
		Search:           search,
		Ranking:          ranking,
		Tags:             tagService,
		Notifications:    notifications,
		Reports:          reports,
		Admin:            admin,
		AdminLogs:        adminLogs,
	}
}
