package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type SearchRepo struct {
	db *gorm.DB
}

func NewSearchRepo(db *gorm.DB) *SearchRepo {
	return &SearchRepo{db: db}
}

func (r *SearchRepo) SearchArticles(ctx context.Context, keyword string, tagID, categoryID *uint, page, pageSize int) ([]model.Article, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Article{}).
		Where("status = ?", "published").
		Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", keyword)

	if tagID != nil {
		query = query.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
			Where("article_tags.tag_id = ?", *tagID)
	}

	if categoryID != nil {
		query = query.Where("category_id = ?", *categoryID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var articles []model.Article
	err := query.
		Order("MATCH(title, summary, content) AGAINST('" + keyword + "' IN BOOLEAN MODE) DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&articles).Error

	return articles, total, err
}

func (r *SearchRepo) SearchSkills(ctx context.Context, keyword string, tagID *uint, page, pageSize int) ([]model.Skill, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Skill{}).
		Where("status = ?", "published").
		Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", keyword)

	if tagID != nil {
		query = query.Joins("JOIN skill_tags ON skill_tags.skill_id = skills.id").
			Where("skill_tags.tag_id = ?", *tagID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var skills []model.Skill
	err := query.
		Order("MATCH(name, description) AGAINST('" + keyword + "' IN BOOLEAN MODE) DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&skills).Error

	return skills, total, err
}

func (r *SearchRepo) SearchMcpServers(ctx context.Context, keyword string, tagID *uint, page, pageSize int) ([]model.McpServer, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.McpServer{}).
		Where("status = ?", "published").
		Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", keyword)

	if tagID != nil {
		query = query.Joins("JOIN mcp_server_tags ON mcp_server_tags.mcp_server_id = mcp_servers.id").
			Where("mcp_server_tags.tag_id = ?", *tagID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var servers []model.McpServer
	err := query.
		Order("MATCH(name, description) AGAINST('" + keyword + "' IN BOOLEAN MODE) DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&servers).Error

	return servers, total, err
}
