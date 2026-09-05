package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/app"
	"aidevclub/internal/handler"
	"aidevclub/internal/mcpserver"
	"aidevclub/internal/model"
	"aidevclub/internal/platform"
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
	r.GET("/metrics/ranking", func(c *gin.Context) {
		c.JSON(200, services.ContentRanking.RankingMetrics())
	})
	r.GET("/api/v1/metrics/ranking", func(c *gin.Context) {
		c.JSON(200, services.ContentRanking.RankingMetrics())
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

	tagH := handler.NewTagHandler(services.Tags)
	artH := handler.NewArticleHandler(services.Articles)
	comH := handler.NewCommentHandler(services.Comments)
	p2Auth := platform.AuthMiddleware(cfg.JWTSecret)
	opt := platform.OptionalAuthMiddleware(cfg.JWTSecret)

	r.GET("/api/v1/tags", tagH.List)

	adminTagH := handler.NewAdminTagHandler(services.Tags)
	adminTags := r.Group("/api/v1/admin/tags")
	adminTags.POST("", adminTagH.Create)
	adminTags.PUT("/:id", adminTagH.Update)
	adminTags.DELETE("/:id", adminTagH.Delete)
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
	skillsGroup.GET("/mine", p2Auth, skillH.ListMine)
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
	mcpGroup.GET("/mine", p2Auth, mcpH.ListMine)
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

	rankingH := handler.NewRankingHandler(services.ContentRanking)
	r.GET("/api/v1/articles/ranking", rankingH.GetArticleRanking)
	r.GET("/api/v1/skills/ranking", rankingH.GetSkillRanking)
	r.GET("/api/v1/mcp-servers/ranking", rankingH.GetMcpServerRanking)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start MCP server in the same process
	startMCPServer(ctx, cfg, services, infra, logger)

	logger.Info("server starting", "addr", cfg.HTTPAddr)
	if err := app.ServeHTTP(ctx, app.NewHTTPServer(cfg.HTTPAddr, r)); err != nil {
		logger.Error("server exited", "err", err)
	}
}

func startMCPServer(ctx context.Context, cfg *platform.Config, services *app.Services, infra *app.Infrastructure, logger *slog.Logger) {
	mcpDeps := mcpserver.Dependencies{
		Public: mcpserver.PublicDependencies{
			Search:     services.Search,
			Articles:   services.Articles,
			Skills:     services.Skills,
			MCPServers: services.MCPServers,
			Tags:       services.Tags,
		},
		Account: mcpserver.AccountDependencies{
			Profile:       services.Users,
			Articles:      services.Articles,
			Skills:        services.Skills,
			MCPServers:    services.MCPServers,
			Notifications: services.Notifications,
		},
	}

	limiter := platform.NewRateLimiter(infra.Redis, cfg.MCPRateLimitPerMin, 60000000000)
	mcpHandler := mcpserver.NewHandler(mcpDeps, cfg, limiter, infra, logger)

	go func() {
		mcpLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		mcpLogger.Info("mcp server starting", "addr", cfg.MCPAddr)
		if err := app.ServeHTTP(ctx, app.NewHTTPServer(cfg.MCPAddr, mcpHandler)); err != nil {
			mcpLogger.Error("mcp server exited", "err", err)
		}
	}()
}
