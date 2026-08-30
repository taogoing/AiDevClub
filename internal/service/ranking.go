package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type RankingService struct {
	rdb         *redis.Client
	articleRepo *repo.ArticleRepo
	skillRepo   *repo.SkillRepo
	mcpRepo     *repo.McpServerRepo
	gravity     float64
}

func NewRankingService(rdb *redis.Client, articleRepo *repo.ArticleRepo, skillRepo *repo.SkillRepo, mcpRepo *repo.McpServerRepo, gravity float64) *RankingService {
	return &RankingService{
		rdb:         rdb,
		articleRepo: articleRepo,
		skillRepo:   skillRepo,
		mcpRepo:     mcpRepo,
		gravity:     gravity,
	}
}

func CalculateHotScore(views, likes, favorites, comments int, publishedAt time.Time, gravity float64) float64 {
	score := float64(views + 3*likes + 5*favorites + 2*comments + 1)
	hours := time.Since(publishedAt).Hours()
	return score / math.Pow(hours+2, gravity)
}

func (s *RankingService) RecalculateArticleHotRanking(ctx context.Context) error {
	articles, _, err := s.articleRepo.List(ctx, repo.ArticleQuery{Sort: "latest"})
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, "rank:articles:hot")

	for _, a := range articles {
		publishedAt := a.PublishedAt
		if publishedAt == nil {
			publishedAt = &a.CreatedAt
		}
		score := CalculateHotScore(a.Views, a.LikesCount, a.FavoritesCount, a.CommentsCount, *publishedAt, s.gravity)
		pipe.ZAdd(ctx, "rank:articles:hot", redis.Z{
			Score:  score,
			Member: a.ID,
		})
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RankingService) RecalculateSkillHotRanking(ctx context.Context) error {
	skills, _, err := s.skillRepo.List(ctx, repo.SkillQuery{Sort: "latest"})
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, "rank:skills:hot")

	for _, sk := range skills {
		score := CalculateHotScore(sk.Views, sk.LikesCount, sk.FavoritesCount, sk.CommentsCount, sk.CreatedAt, s.gravity)
		pipe.ZAdd(ctx, "rank:skills:hot", redis.Z{
			Score:  score,
			Member: sk.ID,
		})
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RankingService) RecalculateMcpServerHotRanking(ctx context.Context) error {
	servers, _, err := s.mcpRepo.List(ctx, repo.McpServerQuery{Sort: "latest"})
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, "rank:mcp_servers:hot")

	for _, sv := range servers {
		score := CalculateHotScore(sv.Views, sv.LikesCount, sv.FavoritesCount, sv.CommentsCount, sv.CreatedAt, s.gravity)
		pipe.ZAdd(ctx, "rank:mcp_servers:hot", redis.Z{
			Score:  score,
			Member: sv.ID,
		})
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RankingService) GetArticleHotRanking(ctx context.Context, page, pageSize int) ([]uint, error) {
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	ids, err := s.rdb.ZRevRange(ctx, "rank:articles:hot", start, stop).Result()
	if err != nil {
		return nil, err
	}

	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		var uid uint
		if _, err := fmt.Sscanf(id, "%d", &uid); err == nil {
			result = append(result, uid)
		}
	}
	return result, nil
}

func (s *RankingService) GetSkillHotRanking(ctx context.Context, page, pageSize int) ([]uint, error) {
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	ids, err := s.rdb.ZRevRange(ctx, "rank:skills:hot", start, stop).Result()
	if err != nil {
		return nil, err
	}

	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		var uid uint
		if _, err := fmt.Sscanf(id, "%d", &uid); err == nil {
			result = append(result, uid)
		}
	}
	return result, nil
}

func (s *RankingService) GetMcpServerHotRanking(ctx context.Context, page, pageSize int) ([]uint, error) {
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	ids, err := s.rdb.ZRevRange(ctx, "rank:mcp_servers:hot", start, stop).Result()
	if err != nil {
		return nil, err
	}

	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		var uid uint
		if _, err := fmt.Sscanf(id, "%d", &uid); err == nil {
			result = append(result, uid)
		}
	}
	return result, nil
}

// ListArticleHot loads the current Redis hot page in ZSet order. Stale rank
// members are expected: public visibility and GORM's soft-delete scope are
// reapplied while the page is hydrated.
func (s *RankingService) ListArticleHot(ctx context.Context, page, pageSize int) ([]ArticleSummary, error) {
	ids, err := s.GetArticleHotRanking(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []ArticleSummary{}, nil
	}

	var articles []model.Article
	if err := s.articleRepo.DB().WithContext(ctx).
		Where("id IN ?", ids).
		Where("status = ?", model.ArticleStatusPublished).
		Where("hidden = ?", false).
		Preload("Category").Preload("Author").Find(&articles).Error; err != nil {
		return nil, err
	}
	tagMap, err := s.articleRepo.TagsForArticles(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]ArticleSummary, len(articles))
	for _, article := range articles {
		byID[article.ID] = rankingArticleSummary(article, tagMap[article.ID])
	}
	result := make([]ArticleSummary, 0, len(articles))
	for _, id := range ids {
		if summary, ok := byID[id]; ok {
			result = append(result, summary)
		}
	}
	return result, nil
}

// ListSkillHot loads the current Redis hot page in ZSet order and silently
// omits stale, hidden, unpublished, or soft-deleted members.
func (s *RankingService) ListSkillHot(ctx context.Context, page, pageSize int) ([]SkillSummary, error) {
	ids, err := s.GetSkillHotRanking(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []SkillSummary{}, nil
	}

	var skills []model.Skill
	if err := s.skillRepo.DB().WithContext(ctx).
		Where("id IN ?", ids).
		Where("status = ?", model.ResourceStatusPublished).
		Where("hidden = ?", false).
		Preload("Author").Find(&skills).Error; err != nil {
		return nil, err
	}
	tagMap, err := s.skillRepo.TagsForSkills(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]SkillSummary, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = rankingSkillSummary(skill, tagMap[skill.ID])
	}
	result := make([]SkillSummary, 0, len(skills))
	for _, id := range ids {
		if summary, ok := byID[id]; ok {
			result = append(result, summary)
		}
	}
	return result, nil
}

// ListMcpServerHot loads the current Redis hot page in ZSet order and omits
// rows that are no longer public without mutating their view counts.
func (s *RankingService) ListMcpServerHot(ctx context.Context, page, pageSize int) ([]McpServerSummary, error) {
	ids, err := s.GetMcpServerHotRanking(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []McpServerSummary{}, nil
	}

	var servers []model.McpServer
	if err := s.mcpRepo.DB().WithContext(ctx).
		Where("id IN ?", ids).
		Where("status = ?", model.ResourceStatusPublished).
		Where("hidden = ?", false).
		Preload("Author").Find(&servers).Error; err != nil {
		return nil, err
	}
	tagMap, err := s.mcpRepo.TagsForMcpServers(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]McpServerSummary, len(servers))
	for _, server := range servers {
		byID[server.ID] = rankingMcpServerSummary(server, tagMap[server.ID])
	}
	result := make([]McpServerSummary, 0, len(servers))
	for _, id := range ids {
		if summary, ok := byID[id]; ok {
			result = append(result, summary)
		}
	}
	return result, nil
}

func rankingArticleSummary(article model.Article, tags []model.Tag) ArticleSummary {
	summary := ArticleSummary{
		ID: article.ID, Title: article.Title, Summary: article.Summary,
		CategoryID: article.CategoryID, Tags: rankingTagBriefs(tags),
		Views: article.Views, LikesCount: article.LikesCount,
		FavoritesCount: article.FavoritesCount, CommentsCount: article.CommentsCount,
		Status: string(article.Status), PublishedAt: article.PublishedAt, Pinned: article.Pinned,
		Author: AuthorBrief{ID: article.AuthorID},
	}
	if article.Category != nil {
		summary.CategoryName = article.Category.Name
	}
	if article.Author != nil {
		summary.Author = AuthorBrief{ID: article.Author.ID, Nickname: article.Author.Nickname, AvatarURL: article.Author.AvatarURL}
	}
	return summary
}

func rankingSkillSummary(skill model.Skill, tags []model.Tag) SkillSummary {
	summary := SkillSummary{
		ID: skill.ID, Name: skill.Name, Description: skill.Description,
		RepoURL: skill.RepoURL, Tags: rankingTagBriefs(tags),
		Views:      skill.Views,
		LikesCount: skill.LikesCount, FavoritesCount: skill.FavoritesCount,
		CommentsCount: skill.CommentsCount, Status: string(skill.Status),
		PublishedAt: skill.PublishedAt, Author: AuthorBrief{ID: skill.AuthorID},
	}
	if skill.Author != nil {
		summary.Author = AuthorBrief{ID: skill.Author.ID, Nickname: skill.Author.Nickname, AvatarURL: skill.Author.AvatarURL}
	}
	return summary
}

func rankingMcpServerSummary(server model.McpServer, tags []model.Tag) McpServerSummary {
	summary := McpServerSummary{
		ID: server.ID, Name: server.Name, Description: server.Description,
		RepoURL: server.RepoURL, Tags: rankingTagBriefs(tags),
		Views:      server.Views,
		LikesCount: server.LikesCount, FavoritesCount: server.FavoritesCount,
		CommentsCount: server.CommentsCount, Status: string(server.Status),
		PublishedAt: server.PublishedAt, Author: AuthorBrief{ID: server.AuthorID},
	}
	if server.Author != nil {
		summary.Author = AuthorBrief{ID: server.Author.ID, Nickname: server.Author.Nickname, AvatarURL: server.Author.AvatarURL}
	}
	return summary
}

func rankingTagBriefs(tags []model.Tag) []TagBrief {
	briefs := make([]TagBrief, 0, len(tags))
	for _, tag := range tags {
		briefs = append(briefs, TagBrief{ID: tag.ID, Name: tag.Name})
	}
	return briefs
}

func (s *RankingService) UpdateArticleHotScore(ctx context.Context, article *model.Article) error {
	publishedAt := article.PublishedAt
	if publishedAt == nil {
		publishedAt = &article.CreatedAt
	}
	score := CalculateHotScore(article.Views, article.LikesCount, article.FavoritesCount, article.CommentsCount, *publishedAt, s.gravity)
	return s.rdb.ZAdd(ctx, "rank:articles:hot", redis.Z{
		Score:  score,
		Member: article.ID,
	}).Err()
}

func (s *RankingService) UpdateSkillHotScore(ctx context.Context, skill *model.Skill) error {
	score := CalculateHotScore(skill.Views, skill.LikesCount, skill.FavoritesCount, skill.CommentsCount, skill.CreatedAt, s.gravity)
	return s.rdb.ZAdd(ctx, "rank:skills:hot", redis.Z{
		Score:  score,
		Member: skill.ID,
	}).Err()
}

func (s *RankingService) UpdateMcpServerHotScore(ctx context.Context, server *model.McpServer) error {
	score := CalculateHotScore(server.Views, server.LikesCount, server.FavoritesCount, server.CommentsCount, server.CreatedAt, s.gravity)
	return s.rdb.ZAdd(ctx, "rank:mcp_servers:hot", redis.Z{
		Score:  score,
		Member: server.ID,
	}).Err()
}
