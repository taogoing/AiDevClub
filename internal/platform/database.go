package platform

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func OpenMySQL(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)

	return db, nil
}

func CreateFulltextIndexes(db *gorm.DB) error {
	if err := db.Exec(`CREATE FULLTEXT INDEX idx_ft_article_search ON articles(title, summary, content) WITH PARSER ngram`).Error; err != nil {
		return fmt.Errorf("create article fulltext index: %w", err)
	}
	if err := db.Exec(`CREATE FULLTEXT INDEX idx_ft_skill_search ON skills(name, description) WITH PARSER ngram`).Error; err != nil {
		return fmt.Errorf("create skill fulltext index: %w", err)
	}
	if err := db.Exec(`CREATE FULLTEXT INDEX idx_ft_mcp_search ON mcp_servers(name, description) WITH PARSER ngram`).Error; err != nil {
		return fmt.Errorf("create mcp_server fulltext index: %w", err)
	}
	return nil
}
