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

func (s *RankingService) RecalculateDownloadRanking(ctx context.Context) error {
	skills, _, err := s.skillRepo.List(ctx, repo.SkillQuery{Sort: "downloads"})
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, "rank:skills:downloads")

	for _, sk := range skills {
		pipe.ZAdd(ctx, "rank:skills:downloads", redis.Z{
			Score:  float64(sk.Downloads),
			Member: sk.ID,
		})
	}

	servers, _, err := s.mcpRepo.List(ctx, repo.McpServerQuery{Sort: "downloads"})
	if err != nil {
		return err
	}

	pipe.Del(ctx, "rank:mcp_servers:downloads")
	for _, sv := range servers {
		pipe.ZAdd(ctx, "rank:mcp_servers:downloads", redis.Z{
			Score:  float64(sv.Downloads),
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

func (s *RankingService) GetSkillDownloadRanking(ctx context.Context, page, pageSize int) ([]uint, error) {
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	ids, err := s.rdb.ZRevRange(ctx, "rank:skills:downloads", start, stop).Result()
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

func (s *RankingService) GetMcpServerDownloadRanking(ctx context.Context, page, pageSize int) ([]uint, error) {
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	ids, err := s.rdb.ZRevRange(ctx, "rank:mcp_servers:downloads", start, stop).Result()
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
