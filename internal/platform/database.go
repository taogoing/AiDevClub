package platform

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func OpenMySQL(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func CreateFulltextIndexes(db *gorm.DB) {
	db.Exec(`CREATE FULLTEXT INDEX idx_ft_article_search ON articles(title, summary, content) WITH PARSER ngram`)
	db.Exec(`CREATE FULLTEXT INDEX idx_ft_skill_search ON skills(name, description) WITH PARSER ngram`)
	db.Exec(`CREATE FULLTEXT INDEX idx_ft_mcp_search ON mcp_servers(name, description) WITH PARSER ngram`)
}
