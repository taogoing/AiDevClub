package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"aidevclub/internal/model"
)

func main() {
	total := flag.Int("total", 200000, "插入文章总数")
	batchSize := flag.Int("batch", 500, "每批插入数量")
	clean := flag.Bool("clean", false, "清除所有 [BENCH] 开头的测试数据")
	dsn := flag.String("dsn", "", "MySQL DSN")
	flag.Parse()

	if *dsn == "" {
		*dsn = "root:Cht20020924.@tcp(47.76.151.183:3306)/aidevclub?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(*dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("连接数据库失败：%v", err)
	}

	if *clean {
		cleanTestData(db)
		return
	}

	log.Printf("开始插入 %d 条测试文章（每批 %d 条）...", *total, *batchSize)

	var users []model.User
	if err := db.Where("role = ?", model.UserRoleUser).Find(&users).Error; err != nil || len(users) == 0 {
		log.Fatal("需要至少一个普通用户")
	}
	log.Printf("找到 %d 个用户作为作者", len(users))

	var tags []model.Tag
	if err := db.Where("enabled = ?", true).Find(&tags).Error; err != nil || len(tags) == 0 {
		log.Fatal("需要至少一个启用的标签")
	}
	log.Printf("找到 %d 个可用标签", len(tags))

	now := time.Now()
	oneMonthAgo := now.AddDate(0, -1, 0)
	rng := rand.New(rand.NewSource(now.UnixNano()))

	var totalInserted int

	for offset := 0; offset < *total; offset += *batchSize {
		remaining := *total - offset
		currentBatch := *batchSize
		if remaining < currentBatch {
			currentBatch = remaining
		}

		articles := make([]model.Article, 0, currentBatch)
		articleTags := make([]model.ArticleTag, 0, currentBatch*2)

		for i := 0; i < currentBatch; i++ {
			publishedAt := randomTime(rng, oneMonthAgo, now)
			views := rng.Intn(10001)
			likesCount := rng.Intn(501)
			favoritesCount := rng.Intn(201)
			commentsCount := rng.Intn(101)

			article := model.Article{
				AuthorID:       users[rng.Intn(len(users))].ID,
				Title:          fmt.Sprintf("[BENCH] 性能测试文章 %d", offset+i),
				Summary:        "这是一篇性能测试文章，仅用于热榜压测。",
				Content:        "这是一篇性能测试文章，仅用于热榜压测。内容不重要，重点是测试热榜排序性能。",
				Status:         model.ArticleStatusPublished,
				Views:          views,
				LikesCount:     likesCount,
				FavoritesCount: favoritesCount,
				CommentsCount:  commentsCount,
				PublishedAt:    &publishedAt,
			}
			articles = append(articles, article)
		}

		if err := db.Create(&articles).Error; err != nil {
			log.Fatalf("插入文章批次失败：%v", err)
		}

		for _, a := range articles {
			tagCount := 1 + rng.Intn(3)
			if tagCount > len(tags) {
				tagCount = len(tags)
			}
			selected := rng.Perm(len(tags))[:tagCount]
			for _, idx := range selected {
				articleTags = append(articleTags, model.ArticleTag{
					ArticleID: a.ID,
					TagID:     tags[idx].ID,
				})
			}
		}

		if err := db.Create(&articleTags).Error; err != nil {
			log.Fatalf("插入文章标签关联失败：%v", err)
		}

		for _, t := range tags {
			count := 0
			for _, at := range articleTags {
				if at.TagID == t.ID {
					count++
				}
			}
			if count > 0 {
				db.Model(&model.Tag{}).Where("id = ?", t.ID).
					UpdateColumn("usage_count", gorm.Expr("usage_count + ?", count))
			}
		}

		totalInserted += currentBatch
		log.Printf("进度：%d/%d (%.1f%%)", totalInserted, *total, float64(totalInserted)/float64(*total)*100)
	}

	log.Printf("插入完成！共插入 %d 篇测试文章", totalInserted)
}

func cleanTestData(db *gorm.DB) {
	log.Println("开始清除测试数据...")

	var testArticleIDs []uint
	db.Model(&model.Article{}).
		Where("title LIKE ?", "[BENCH] %").
		Pluck("id", &testArticleIDs)

	if len(testArticleIDs) == 0 {
		log.Println("没有找到测试数据")
		return
	}

	log.Printf("找到 %d 篇测试文章，开始清除...", len(testArticleIDs))

	db.Where("article_id IN ?", testArticleIDs).Delete(&model.ArticleTag{})
	db.Where("id IN ?", testArticleIDs).Delete(&model.Article{})

	log.Println("测试数据清除完成")
}

func randomTime(rng *rand.Rand, min, max time.Time) time.Time {
	delta := max.Sub(min).Nanoseconds()
	if delta <= 0 {
		return min
	}
	return min.Add(time.Duration(rng.Int63n(delta)))
}
