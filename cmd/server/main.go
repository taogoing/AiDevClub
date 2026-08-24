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
	); err != nil {
		logger.Error("migrate", "err", err)
		return
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
	tagSvc := service.NewTagService(tags)
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

	logger.Info("server starting", "addr", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		logger.Error("server exited", "err", err)
	}
}
