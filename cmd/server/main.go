package main

import (
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
	if err := db.AutoMigrate(&model.User{}); err != nil {
		logger.Error("migrate", "err", err)
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

	logger.Info("server starting", "addr", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		logger.Error("server exited", "err", err)
	}
}
