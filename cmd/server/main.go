package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/handler"
	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/scheduler"
	"aidevclub/internal/service"
)

func main() {
	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger := platform.NewLogger()

	db, err := platform.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		logger.Error("open mysql", "err", err)
		return
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Category{}, &model.Tag{}, &model.Article{},
		&model.ArticleTag{}, &model.ArticleLike{}, &model.ArticleFavorite{},
		&model.Comment{}, &model.CommentLike{},
		&model.Skill{}, &model.SkillTag{},
		&model.McpServer{}, &model.McpServerTag{},
		&model.SkillLike{}, &model.SkillFavorite{},
		&model.McpServerLike{}, &model.McpServerFavorite{},
		&model.ResourceComment{}, &model.ResourceCommentLike{},
	); err != nil {
		logger.Error("migrate", "err", err)
		return
	}
	if err := platform.CreateFulltextIndexes(db); err != nil {
		logger.Warn("fulltext indexes", "err", err)
	}
	cats := repo.NewCategoryRepo(db)
	if err := cats.Seed(context.Background()); err != nil {
		logger.Error("seed categories", "err", err)
		return
	}
	rdb := platform.OpenRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)

	users := repo.NewUserRepo(db)
	tokens := repo.NewTokenRepo(rdb, cfg.RefreshTokenTTL)
	authSvc := service.NewAuthService(users, tokens, cfg)
	userSvc := service.NewUserService(users, tokens, cfg)

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	ah := handler.NewAuthHandler(authSvc)
	rl := platform.RateLimitMiddleware(rdb, cfg.RateLimitPerMin, time.Minute)
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", rl, ah.Register)
	auth.POST("/login", rl, ah.Login)
	auth.POST("/refresh", ah.Refresh)
	auth.POST("/logout", ah.Logout)

	uh := handler.NewUserHandler(userSvc)
	me := r.Group("/api/v1/users", platform.AuthMiddleware(cfg.JWTSecret))
	me.GET("/me", uh.Me)
	me.PATCH("/me", uh.Update)
	me.PUT("/me/password", uh.ChangePassword)
	me.DELETE("/me", uh.Delete)
	me.POST("/me/avatar", uh.UploadAvatar)
	r.Static("/static/avatars", cfg.AvatarDir)

	tags := repo.NewTagRepo(db)
	articles := repo.NewArticleRepo(db)
	comments := repo.NewCommentRepo(db)
	inter := repo.NewInteractionRepo(db)

	catSvc := service.NewCategoryService(cats)
	tagSvc := service.NewTagService(tags, rdb)
	artSvc := service.NewArticleService(articles, tags, cats, inter, rdb, cfg)
	comSvc := service.NewCommentService(comments, articles, inter, users)

	catH := handler.NewCategoryHandler(catSvc)
	tagH := handler.NewTagHandler(tagSvc)
	artH := handler.NewArticleHandler(artSvc)
	comH := handler.NewCommentHandler(comSvc)
	p2Auth := platform.AuthMiddleware(cfg.JWTSecret)
	opt := platform.OptionalAuthMiddleware(cfg.JWTSecret)

	r.GET("/api/v1/categories", catH.List)
	r.GET("/api/v1/tags", tagH.List)

	adminTagH := handler.NewAdminTagHandler(tagSvc)
	adminTags := r.Group("/api/v1/admin/tags")
	adminTags.POST("", adminTagH.Create)
	adminTags.PUT("/:id", adminTagH.Update)
	adminTags.PATCH("/:id/enable", adminTagH.Enable)
	adminTags.PATCH("/:id/disable", adminTagH.Disable)
	adminTags.GET("", adminTagH.List)

	searchRepo := repo.NewSearchRepo(db)
	searchSvc := service.NewSearchService(searchRepo)
	searchH := handler.NewSearchHandler(searchSvc)
	r.GET("/api/v1/search", searchH.Search)

	arts := r.Group("/api/v1/articles")
	arts.GET("", artH.List)
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

	skills := repo.NewSkillRepo(db)
	mcpServers := repo.NewMcpServerRepo(db)
	resComments := repo.NewResourceCommentRepo(db)

	skillSvc := service.NewSkillService(skills, tags, inter, rdb, cfg)
	mcpSvc := service.NewMcpServerService(mcpServers, tags, inter, rdb, cfg)
	resCommentSvc := service.NewResourceCommentService(resComments, skills, mcpServers, inter, users)

	skillH := handler.NewSkillHandler(skillSvc)
	mcpH := handler.NewMcpServerHandler(mcpSvc)
	resCommentH := handler.NewResourceCommentHandler(resCommentSvc)

	skillsGroup := r.Group("/api/v1/skills")
	skillsGroup.GET("", skillH.List)
	skillsGroup.POST("", p2Auth, skillH.Create)
	skillsGroup.GET("/:id", opt, skillH.Get)
	skillsGroup.PUT("/:id", p2Auth, skillH.Update)
	skillsGroup.DELETE("/:id", p2Auth, skillH.Delete)
	skillsGroup.POST("/:id/upload", p2Auth, skillH.Upload)
	skillsGroup.POST("/:id/submit", p2Auth, skillH.Submit)
	skillsGroup.POST("/:id/withdraw", p2Auth, skillH.Withdraw)
	skillsGroup.POST("/:id/archive", p2Auth, skillH.Archive)
	skillsGroup.POST("/:id/download", skillH.Download)
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
	mcpGroup.POST("/:id/upload", p2Auth, mcpH.Upload)
	mcpGroup.POST("/:id/submit", p2Auth, mcpH.Submit)
	mcpGroup.POST("/:id/withdraw", p2Auth, mcpH.Withdraw)
	mcpGroup.POST("/:id/archive", p2Auth, mcpH.Archive)
	mcpGroup.POST("/:id/download", mcpH.Download)
	mcpGroup.POST("/:id/like", p2Auth, mcpH.Like)
	mcpGroup.POST("/:id/favorite", p2Auth, mcpH.Favorite)

	mcpComments := r.Group("/api/v1/mcp-servers/:id/comments")
	mcpComments.Use(func(c *gin.Context) { c.Set("resource_type", "mcp_server") })
	mcpComments.GET("", resCommentH.List)
	mcpComments.POST("", p2Auth, resCommentH.Create)

	resComs := r.Group("/api/v1/resource-comments")
	resComs.DELETE("/:id", p2Auth, resCommentH.Delete)
	resComs.POST("/:id/like", p2Auth, resCommentH.Like)

	r.Static("/static/skills", cfg.SkillZipDir)
	r.Static("/static/mcp-servers", cfg.McpServerZipDir)

	rankingSvc := service.NewRankingService(rdb, articles, repo.NewSkillRepo(db), repo.NewMcpServerRepo(db), 1.5)
	rankingH := handler.NewRankingHandler(rankingSvc, artSvc, skillSvc, mcpSvc)
	r.GET("/api/v1/articles/ranking", rankingH.GetArticleRanking)
	r.GET("/api/v1/skills/ranking", rankingH.GetSkillRanking)
	r.GET("/api/v1/mcp-servers/ranking", rankingH.GetMcpServerRanking)

	rankingScheduler := scheduler.NewRankingScheduler(rankingSvc, 2*time.Minute)
	rankingScheduler.Start()

	logger.Info("server starting", "addr", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		logger.Error("server exited", "err", err)
	}
}
