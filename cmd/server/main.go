package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/app"
	"aidevclub/internal/handler"
	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/scheduler"
)

func main() {
	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger := platform.NewLogger()

	infra, err := app.OpenInfrastructure(cfg)
	if err != nil {
		logger.Error("open infrastructure", "err", err)
		return
	}
	defer func() {
		if err := infra.Close(); err != nil {
			logger.Error("close infrastructure", "err", err)
		}
	}()
	if err := app.Migrate(infra.DB); err != nil {
		logger.Error("migrate", "err", err)
		return
	}
	services := app.NewServices(infra, cfg)

	// 支持强制重新初始化分类（通过环境变量 AIDEVCLUB_SEED_CATEGORIES_FORCE=true）
	if os.Getenv("AIDEVCLUB_SEED_CATEGORIES_FORCE") == "true" {
		logger.Info("force seeding categories")
		if err := services.SeedCategoriesForce(context.Background()); err != nil {
			logger.Error("seed categories force", "err", err)
			return
		}
	} else if err := services.SeedCategories(context.Background()); err != nil {
		logger.Error("seed categories", "err", err)
		return
	}

	for _, email := range cfg.AdminEmails {
		u, err := services.UserRepo.FindByEmail(email)
		if err == nil && u.Role != model.UserRoleAdmin {
			_ = services.UserRepo.UpdateRole(u.ID, model.UserRoleAdmin)
		}
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	ah := handler.NewAuthHandler(services.Auth)
	rl := platform.RateLimitMiddleware(infra.Redis, cfg.RateLimitPerMin, time.Minute)
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", rl, ah.Register)
	auth.POST("/login", rl, ah.Login)
	auth.POST("/refresh", ah.Refresh)
	auth.POST("/logout", ah.Logout)

	uh := handler.NewUserHandler(services.Users)
	me := r.Group("/api/v1/users", platform.AuthMiddleware(cfg.JWTSecret))
	me.GET("/me", uh.Me)
	me.PUT("/me", uh.Update)
	me.PUT("/me/password", uh.ChangePassword)
	me.DELETE("/me", uh.Delete)
	me.POST("/me/avatar", uh.UploadAvatar)
	r.Static("/static/avatars", cfg.AvatarDir)

	catH := handler.NewCategoryHandler(services.Categories)
	tagH := handler.NewTagHandler(services.Tags)
	artH := handler.NewArticleHandler(services.Articles)
	comH := handler.NewCommentHandler(services.Comments)
	p2Auth := platform.AuthMiddleware(cfg.JWTSecret)
	opt := platform.OptionalAuthMiddleware(cfg.JWTSecret)

	r.GET("/api/v1/categories", catH.List)
	r.GET("/api/v1/tags", tagH.List)

	adminTagH := handler.NewAdminTagHandler(services.Tags)
	adminTags := r.Group("/api/v1/admin/tags")
	adminTags.POST("", adminTagH.Create)
	adminTags.PUT("/:id", adminTagH.Update)
	adminTags.GET("", adminTagH.List)

	searchH := handler.NewSearchHandler(services.Search)
	r.GET("/api/v1/search", searchH.Search)

	arts := r.Group("/api/v1/articles")
	arts.GET("", artH.List)
	arts.GET("/mine", p2Auth, artH.ListMine)
	arts.POST("/images", p2Auth, artH.UploadImage)
	arts.POST("", p2Auth, artH.Create)
	arts.GET("/:id", opt, artH.Get)
	arts.PUT("/:id", p2Auth, artH.Update)
	arts.DELETE("/:id", p2Auth, artH.Delete)
	arts.POST("/:id/like", p2Auth, artH.Like)
	arts.POST("/:id/favorite", p2Auth, artH.Favorite)

	artComments := r.Group("/api/v1/articles/:id/comments")
	artComments.GET("", comH.List)
	artComments.POST("", p2Auth, comH.Create)

	coms := r.Group("/api/v1/comments")
	coms.DELETE("/:id", p2Auth, comH.Delete)
	coms.POST("/:id/like", p2Auth, comH.Like)

	r.Static("/static/articles", cfg.ArticleImageDir)

	skillH := handler.NewSkillHandler(services.Skills)
	mcpH := handler.NewMcpServerHandler(services.MCPServers)
	resCommentH := handler.NewResourceCommentHandler(services.ResourceComments)

	skillsGroup := r.Group("/api/v1/skills")
	skillsGroup.GET("", skillH.List)
	skillsGroup.POST("", p2Auth, skillH.Create)
	skillsGroup.GET("/:id", opt, skillH.Get)
	skillsGroup.PUT("/:id", p2Auth, skillH.Update)
	skillsGroup.DELETE("/:id", p2Auth, skillH.Delete)
	skillsGroup.POST("/:id/submit", p2Auth, skillH.Submit)
	skillsGroup.POST("/:id/withdraw", p2Auth, skillH.Withdraw)
	skillsGroup.POST("/:id/archive", p2Auth, skillH.Archive)
	skillsGroup.POST("/:id/like", p2Auth, skillH.Like)
	skillsGroup.POST("/:id/favorite", p2Auth, skillH.Favorite)

	skillComments := r.Group("/api/v1/skills/:id/comments")
	skillComments.Use(func(c *gin.Context) { c.Set("resource_type", "skill") })
	skillComments.GET("", resCommentH.List)
	skillComments.POST("", p2Auth, resCommentH.Create)

	mcpGroup := r.Group("/api/v1/mcp-servers")
	mcpGroup.GET("", mcpH.List)
	mcpGroup.POST("", p2Auth, mcpH.Create)
	mcpGroup.GET("/:id", opt, mcpH.Get)
	mcpGroup.PUT("/:id", p2Auth, mcpH.Update)
	mcpGroup.DELETE("/:id", p2Auth, mcpH.Delete)
	mcpGroup.POST("/:id/submit", p2Auth, mcpH.Submit)
	mcpGroup.POST("/:id/withdraw", p2Auth, mcpH.Withdraw)
	mcpGroup.POST("/:id/archive", p2Auth, mcpH.Archive)
	mcpGroup.POST("/:id/like", p2Auth, mcpH.Like)
	mcpGroup.POST("/:id/favorite", p2Auth, mcpH.Favorite)

	mcpComments := r.Group("/api/v1/mcp-servers/:id/comments")
	mcpComments.Use(func(c *gin.Context) { c.Set("resource_type", "mcp_server") })
	mcpComments.GET("", resCommentH.List)
	mcpComments.POST("", p2Auth, resCommentH.Create)

	resComs := r.Group("/api/v1/resource-comments")
	resComs.DELETE("/:id", p2Auth, resCommentH.Delete)
	resComs.POST("/:id/like", p2Auth, resCommentH.Like)

	nh := handler.NewNotificationHandler(services.Notifications)
	notifs := r.Group("/api/v1/notifications", p2Auth)
	notifs.GET("", nh.List)
	notifs.GET("/unread-count", nh.UnreadCount)
	notifs.PUT("/:id/read", nh.MarkRead)
	notifs.PUT("/read", nh.MarkAllRead)

	rh := handler.NewReportHandler(services.Reports)
	reports := r.Group("/api/v1/reports", p2Auth)
	reports.POST("", rh.Create)
	reports.GET("", rh.List)

	adminAuth := r.Group("/api/v1/admin", p2Auth, platform.AdminMiddleware(services.UserRepo))
	adminH := handler.NewAdminHandler(services.Admin, services.Reports, services.AdminLogs)
	adminH.RegisterRoutes(adminAuth)

	rankingH := handler.NewRankingHandler(services.Ranking, services.Articles, services.Skills, services.MCPServers)
	r.GET("/api/v1/articles/ranking", rankingH.GetArticleRanking)
	r.GET("/api/v1/skills/ranking", rankingH.GetSkillRanking)
	r.GET("/api/v1/mcp-servers/ranking", rankingH.GetMcpServerRanking)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rankingScheduler := scheduler.NewRankingScheduler(services.Ranking, 2*time.Minute)
	rankingScheduler.Start(ctx)
	defer rankingScheduler.Stop()

	logger.Info("server starting", "addr", cfg.HTTPAddr)
	if err := app.ServeHTTP(ctx, app.NewHTTPServer(cfg.HTTPAddr, r)); err != nil {
		logger.Error("server exited", "err", err)
	}
}
