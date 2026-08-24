package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
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
	_ = rdb

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	logger.Info("server starting", "addr", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		logger.Error("server exited", "err", err)
	}
}
