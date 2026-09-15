package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
)

type testData struct {
	ArticleID uint     `json:"article_id"`
	Tokens    []string `json:"tokens"`
	Label     string   `json:"label"`
}

func main() {
	count := flag.Int("users", 20, "number of dedicated actor users; use the same value for sync and outbox")
	label := flag.String("label", "local", "unique alphanumeric run label")
	flag.Parse()
	if *count < 1 || *count > 500 {
		fatal("users must be between 1 and 500")
	}
	cfg, err := platform.LoadConfig()
	if err != nil {
		fatal("load config: %v", err)
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fatal("open MySQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		fatal("get SQL database: %v", err)
	}
	defer sqlDB.Close()

	owner := model.User{Email: "notif-bench-" + *label + "-owner@local.test", PasswordHash: "benchmark-token-only", Nickname: "通知压测作者", AvatarURL: ""}
	if err := db.Create(&owner).Error; err != nil {
		fatal("create benchmark article author: %v", err)
	}
	actors := make([]model.User, *count)
	for i := range actors {
		actors[i] = model.User{
			Email:        fmt.Sprintf("notif-bench-%s-%03d@local.test", *label, i+1),
			PasswordHash: "benchmark-token-only", Nickname: fmt.Sprintf("压测用户%03d", i+1), AvatarURL: "",
		}
	}
	if err := db.Create(&actors).Error; err != nil {
		fatal("create benchmark actors: %v", err)
	}

	now := time.Now()
	article := model.Article{
		AuthorID: owner.ID, Title: "[NOTIF-BENCH] " + *label,
		Summary: "本地通知写入性能对比专用文章。",
		Content: "用于同步通知和 RabbitMQ Outbox 对比的本地压测数据。",
		Status:  model.ArticleStatusPublished, PublishedAt: &now,
	}
	if err := db.Create(&article).Error; err != nil {
		fatal("create benchmark article: %v", err)
	}

	tokens := make([]string, 0, len(actors))
	for _, actor := range actors {
		token, err := platform.GenerateAccessToken(cfg.JWTSecret, 2*time.Hour, actor.ID)
		if err != nil {
			fatal("create actor token: %v", err)
		}
		tokens = append(tokens, token)
	}
	if err := json.NewEncoder(os.Stdout).Encode(testData{ArticleID: article.ID, Tokens: tokens, Label: *label}); err != nil {
		fatal("write benchmark test data: %v", err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
