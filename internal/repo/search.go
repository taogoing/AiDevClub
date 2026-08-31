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

func (r *SearchRepo) SearchArticles(ctx context.Context, keyword string, tagID *uint, page, pageSize int) ([]model.Article, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Article{}).
		Where("status = ?", "published").
		Where("hidden = ?", false).
		Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", keyword)

	if tagID != nil {
		query = query.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
			Where("article_tags.tag_id = ?", *tagID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var articles []model.Article
	err := query.
		Select("*, MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE) AS relevance", keyword).
		Order("relevance DESC").
		Preload("Author").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&articles).Error

	return articles, total, err
}

func (r *SearchRepo) SearchSkills(ctx context.Context, keyword string, tagID *uint, page, pageSize int) ([]model.Skill, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Skill{}).
		Where("status = ?", "published").
		Where("hidden = ?", false).
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
		Select("*, MATCH(name, description) AGAINST(? IN BOOLEAN MODE) AS relevance", keyword).
		Order("relevance DESC").
		Preload("Author").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&skills).Error

	return skills, total, err
}

func (r *SearchRepo) SearchMcpServers(ctx context.Context, keyword string, tagID *uint, page, pageSize int) ([]model.McpServer, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.McpServer{}).
		Where("status = ?", "published").
		Where("hidden = ?", false).
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
		Select("*, MATCH(name, description) AGAINST(? IN BOOLEAN MODE) AS relevance", keyword).
		Order("relevance DESC").
		Preload("Author").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&servers).Error

	return servers, total, err
}

func (r *SearchRepo) TagsForArticles(ctx context.Context, articleIDs []uint) (map[uint][]model.Tag, error) {
	return NewArticleRepo(r.db).TagsForArticles(ctx, articleIDs)
}

func (r *SearchRepo) TagsForSkills(ctx context.Context, skillIDs []uint) (map[uint][]model.Tag, error) {
	return NewSkillRepo(r.db).TagsForSkills(ctx, skillIDs)
}

func (r *SearchRepo) TagsForMcpServers(ctx context.Context, serverIDs []uint) (map[uint][]model.Tag, error) {
	return NewMcpServerRepo(r.db).TagsForMcpServers(ctx, serverIDs)
}
