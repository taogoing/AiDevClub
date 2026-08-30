package app

import (
	"context"

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
	Categories       *service.CategoryService
	Tags             *service.TagService
	Notifications    *service.NotificationService
	Reports          *service.ReportService
	Admin            *service.AdminService
	AdminLogs        *service.AdminLogService

	categoryRepo *repo.CategoryRepo
}

func NewServices(infra *Infrastructure, cfg *platform.Config) *Services {
	categories := repo.NewCategoryRepo(infra.DB)
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

	categoryService := service.NewCategoryService(categories)
	tagService := service.NewTagService(tags, infra.Redis)
	articleService := service.NewArticleService(articles, tags, categories, interactions, infra.Redis, cfg, notifications)
	commentService := service.NewCommentService(comments, articles, interactions, users, notifications)

	searchRepo := repo.NewSearchRepo(infra.DB)
	search := service.NewSearchService(searchRepo)

	skills := repo.NewSkillRepo(infra.DB)
	mcpServers := repo.NewMcpServerRepo(infra.DB)
	resourceComments := repo.NewResourceCommentRepo(infra.DB)
	skillService := service.NewSkillService(skills, tags, interactions, infra.Redis, cfg, notifications)
	mcpServerService := service.NewMcpServerService(mcpServers, tags, interactions, infra.Redis, cfg, notifications)
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
	ranking := service.NewRankingService(infra.Redis, articles, skills, mcpServers, 1.5)

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
		Categories:       categoryService,
		Tags:             tagService,
		Notifications:    notifications,
		Reports:          reports,
		Admin:            admin,
		AdminLogs:        adminLogs,
		categoryRepo:     categories,
	}
}

func (s *Services) SeedCategories(ctx context.Context) error {
	return s.categoryRepo.Seed(ctx)
}

func (s *Services) SeedCategoriesForce(ctx context.Context) error {
	return s.categoryRepo.SeedForce(ctx)
}
