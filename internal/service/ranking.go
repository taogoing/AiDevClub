package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

const (
	rankKeyArticles = "rank:articles:hot"
	rankKeySkills   = "rank:skills:hot"
	rankKeyMcp      = "rank:mcp_servers:hot"
)

type localCacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

type localCache struct {
	mu      sync.RWMutex
	entries map[string]localCacheEntry
	ttl     time.Duration
}

func newLocalCache(ttl time.Duration) *localCache {
	return &localCache{
		entries: make(map[string]localCacheEntry),
		ttl:     ttl,
	}
}

func (c *localCache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *localCache) set(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = localCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}

type RankingService struct {
	rdb         *redis.Client
	articleRepo *repo.ArticleRepo
	skillRepo   *repo.SkillRepo
	mcpRepo     *repo.McpServerRepo
	gravity     float64
	maxCand     int
	minLikes    int
	minFavs     int
	minComments int
	minViews    int
	cache       *localCache
}

func NewRankingService(
	rdb *redis.Client,
	articleRepo *repo.ArticleRepo,
	skillRepo *repo.SkillRepo,
	mcpRepo *repo.McpServerRepo,
	cfg *platform.Config,
) *RankingService {
	return &RankingService{
		rdb:         rdb,
		articleRepo: articleRepo,
		skillRepo:   skillRepo,
		mcpRepo:     mcpRepo,
		gravity:     cfg.RankingGravity,
		maxCand:     cfg.RankingMaxCandidates,
		minLikes:    cfg.RankingMinLikes,
		minFavs:     cfg.RankingMinFavorites,
		minComments: cfg.RankingMinComments,
		minViews:    cfg.RankingMinViews,
		cache:       newLocalCache(cfg.RankingLocalCacheTTL),
	}
}

func CalculateHotScore(views, likes, favorites, comments int, publishedAt time.Time, gravity float64) float64 {
	score := float64(views + 3*likes + 5*favorites + 2*comments + 1)
	hours := time.Since(publishedAt).Hours()
	return score / math.Pow(hours+2, gravity)
}

func (s *RankingService) meetsThreshold(a *model.Article) bool {
	return a.LikesCount >= s.minLikes ||
		a.FavoritesCount >= s.minFavs ||
		a.CommentsCount >= s.minComments ||
		a.Views >= s.minViews
}

func (s *RankingService) meetsSkillThreshold(sk *model.Skill) bool {
	return sk.LikesCount >= s.minLikes ||
		sk.FavoritesCount >= s.minFavs ||
		sk.CommentsCount >= s.minComments ||
		sk.Views >= s.minViews
}

func (s *RankingService) meetsMcpThreshold(sv *model.McpServer) bool {
	return sv.LikesCount >= s.minLikes ||
		sv.FavoritesCount >= s.minFavs ||
		sv.CommentsCount >= s.minComments ||
		sv.Views >= s.minViews
}

// --- Write path: update a single item's score in ZSet ---

func (s *RankingService) UpdateArticleHotScore(ctx context.Context, article *model.Article) error {
	if !s.meetsThreshold(article) {
		// Below threshold: remove from ZSet if present
		return s.rdb.ZRem(ctx, rankKeyArticles, article.ID).Err()
	}
	publishedAt := article.PublishedAt
	if publishedAt == nil {
		publishedAt = &article.CreatedAt
	}
	score := CalculateHotScore(article.Views, article.LikesCount, article.FavoritesCount, article.CommentsCount, *publishedAt, s.gravity)
	return s.rdb.ZAdd(ctx, rankKeyArticles, redis.Z{
		Score:  score,
		Member: article.ID,
	}).Err()
}

func (s *RankingService) UpdateSkillHotScore(ctx context.Context, skill *model.Skill) error {
	if !s.meetsSkillThreshold(skill) {
		return s.rdb.ZRem(ctx, rankKeySkills, skill.ID).Err()
	}
	score := CalculateHotScore(skill.Views, skill.LikesCount, skill.FavoritesCount, skill.CommentsCount, skill.CreatedAt, s.gravity)
	return s.rdb.ZAdd(ctx, rankKeySkills, redis.Z{
		Score:  score,
		Member: skill.ID,
	}).Err()
}

func (s *RankingService) UpdateMcpServerHotScore(ctx context.Context, server *model.McpServer) error {
	if !s.meetsMcpThreshold(server) {
		return s.rdb.ZRem(ctx, rankKeyMcp, server.ID).Err()
	}
	score := CalculateHotScore(server.Views, server.LikesCount, server.FavoritesCount, server.CommentsCount, server.CreatedAt, s.gravity)
	return s.rdb.ZAdd(ctx, rankKeyMcp, redis.Z{
		Score:  score,
		Member: server.ID,
	}).Err()
}

// --- Read path: get top N from ZSet, hydrate from MySQL ---

func (s *RankingService) GetArticleHotRanking(ctx context.Context, page, pageSize int) ([]uint, error) {
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	ids, err := s.rdb.ZRevRange(ctx, rankKeyArticles, start, stop).Result()
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

	ids, err := s.rdb.ZRevRange(ctx, rankKeySkills, start, stop).Result()
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

	ids, err := s.rdb.ZRevRange(ctx, rankKeyMcp, start, stop).Result()
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

// --- Hydrated list (for MCP browse_content tool) ---

func (s *RankingService) ListArticleHot(ctx context.Context, page, pageSize int) ([]ArticleSummary, error) {
	cacheKey := fmt.Sprintf("article:%d:%d", page, pageSize)
	if cached, ok := s.cache.get(cacheKey); ok {
		fmt.Printf("[CACHE HIT] key=%s\n", cacheKey)
		return cached.([]ArticleSummary), nil
	}
	fmt.Printf("[CACHE MISS] key=%s\n", cacheKey)

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
		Preload("Author").Find(&articles).Error; err != nil {
		return nil, err
	}
	tagMap, err := s.articleRepo.TagsForArticles(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]ArticleSummary, len(articles))
	for _, a := range articles {
		byID[a.ID] = rankingArticleSummary(a, tagMap[a.ID])
	}
	result := make([]ArticleSummary, 0, len(ids))
	for _, id := range ids {
		if summary, ok := byID[id]; ok {
			result = append(result, summary)
		}
	}

	s.cache.set(cacheKey, result)
	fmt.Printf("[CACHE SET] key=%s, ttl=%v\n", cacheKey, s.cache.ttl)
	return result, nil
}

func (s *RankingService) ListSkillHot(ctx context.Context, page, pageSize int) ([]SkillSummary, error) {
	cacheKey := fmt.Sprintf("skill:%d:%d", page, pageSize)
	if cached, ok := s.cache.get(cacheKey); ok {
		return cached.([]SkillSummary), nil
	}

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
	for _, sk := range skills {
		byID[sk.ID] = rankingSkillSummary(sk, tagMap[sk.ID])
	}
	result := make([]SkillSummary, 0, len(ids))
	for _, id := range ids {
		if summary, ok := byID[id]; ok {
			result = append(result, summary)
		}
	}

	s.cache.set(cacheKey, result)
	return result, nil
}

func (s *RankingService) ListMcpServerHot(ctx context.Context, page, pageSize int) ([]McpServerSummary, error) {
	cacheKey := fmt.Sprintf("mcp:%d:%d", page, pageSize)
	if cached, ok := s.cache.get(cacheKey); ok {
		return cached.([]McpServerSummary), nil
	}

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
	for _, sv := range servers {
		byID[sv.ID] = rankingMcpServerSummary(sv, tagMap[sv.ID])
	}
	result := make([]McpServerSummary, 0, len(ids))
	for _, id := range ids {
		if summary, ok := byID[id]; ok {
			result = append(result, summary)
		}
	}

	s.cache.set(cacheKey, result)
	return result, nil
}

// --- Scheduler: recalculate candidate set ---

// RecalculateArticleHotRanking recalculates scores for all items currently in the ZSet
// (candidate set) and trims low-scoring items. It also scans for new items that
// recently crossed the threshold.
func (s *RankingService) RecalculateArticleHotRanking(ctx context.Context) error {
	// 1. Get all current candidates from ZSet
	members, err := s.rdb.ZRange(ctx, rankKeyArticles, 0, -1).Result()
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()

	// 2. Recalculate scores for existing candidates
	for _, idStr := range members {
		var id uint
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			continue
		}
		a, err := s.articleRepo.FindByIDWithContext(ctx, id)
		if err != nil {
			// Article may have been deleted; remove from ZSet
			pipe.ZRem(ctx, rankKeyArticles, id)
			continue
		}
		if a.Status != model.ArticleStatusPublished || a.Hidden {
			pipe.ZRem(ctx, rankKeyArticles, id)
			continue
		}
		publishedAt := a.PublishedAt
		if publishedAt == nil {
			publishedAt = &a.CreatedAt
		}
		if !s.meetsThreshold(a) {
			pipe.ZRem(ctx, rankKeyArticles, id)
			continue
		}
		score := CalculateHotScore(a.Views, a.LikesCount, a.FavoritesCount, a.CommentsCount, *publishedAt, s.gravity)
		pipe.ZAdd(ctx, rankKeyArticles, redis.Z{Score: score, Member: id})
	}

	// 3. Scan for new candidates: recently updated articles that meet threshold but aren't in ZSet
	newCandidates, _, err := s.articleRepo.List(ctx, repo.ArticleQuery{
		Page: 1, PageSize: s.maxCand, Sort: "hot",
	})
	if err == nil {
		candidateSet := make(map[uint]bool, len(members))
		for _, idStr := range members {
			var id uint
			if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
				candidateSet[id] = true
			}
		}
		for _, a := range newCandidates {
			if !candidateSet[a.ID] && s.meetsThreshold(&a) {
				publishedAt := a.PublishedAt
				if publishedAt == nil {
					publishedAt = &a.CreatedAt
				}
				score := CalculateHotScore(a.Views, a.LikesCount, a.FavoritesCount, a.CommentsCount, *publishedAt, s.gravity)
				pipe.ZAdd(ctx, rankKeyArticles, redis.Z{Score: score, Member: a.ID})
			}
		}
	}

	// 4. Trim: keep only top N candidates
	pipe.ZRemRangeByRank(ctx, rankKeyArticles, 0, int64(-s.maxCand-1))

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RankingService) RecalculateSkillHotRanking(ctx context.Context) error {
	members, err := s.rdb.ZRange(ctx, rankKeySkills, 0, -1).Result()
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()

	for _, idStr := range members {
		var id uint
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			continue
		}
		sk, err := s.skillRepo.FindByIDWithContext(ctx, id)
		if err != nil {
			pipe.ZRem(ctx, rankKeySkills, id)
			continue
		}
		if sk.Status != model.ResourceStatusPublished || sk.Hidden {
			pipe.ZRem(ctx, rankKeySkills, id)
			continue
		}
		if !s.meetsSkillThreshold(sk) {
			pipe.ZRem(ctx, rankKeySkills, id)
			continue
		}
		score := CalculateHotScore(sk.Views, sk.LikesCount, sk.FavoritesCount, sk.CommentsCount, sk.CreatedAt, s.gravity)
		pipe.ZAdd(ctx, rankKeySkills, redis.Z{Score: score, Member: id})
	}

	newCandidates, _, err := s.skillRepo.List(ctx, repo.SkillQuery{
		Page: 1, PageSize: s.maxCand, Sort: "hot",
	})
	if err == nil {
		candidateSet := make(map[uint]bool, len(members))
		for _, idStr := range members {
			var id uint
			if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
				candidateSet[id] = true
			}
		}
		for _, sk := range newCandidates {
			if !candidateSet[sk.ID] && s.meetsSkillThreshold(&sk) {
				score := CalculateHotScore(sk.Views, sk.LikesCount, sk.FavoritesCount, sk.CommentsCount, sk.CreatedAt, s.gravity)
				pipe.ZAdd(ctx, rankKeySkills, redis.Z{Score: score, Member: sk.ID})
			}
		}
	}

	pipe.ZRemRangeByRank(ctx, rankKeySkills, 0, int64(-s.maxCand-1))

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RankingService) RecalculateMcpServerHotRanking(ctx context.Context) error {
	members, err := s.rdb.ZRange(ctx, rankKeyMcp, 0, -1).Result()
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()

	for _, idStr := range members {
		var id uint
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			continue
		}
		sv, err := s.mcpRepo.FindByIDWithContext(ctx, id)
		if err != nil {
			pipe.ZRem(ctx, rankKeyMcp, id)
			continue
		}
		if sv.Status != model.ResourceStatusPublished || sv.Hidden {
			pipe.ZRem(ctx, rankKeyMcp, id)
			continue
		}
		if !s.meetsMcpThreshold(sv) {
			pipe.ZRem(ctx, rankKeyMcp, id)
			continue
		}
		score := CalculateHotScore(sv.Views, sv.LikesCount, sv.FavoritesCount, sv.CommentsCount, sv.CreatedAt, s.gravity)
		pipe.ZAdd(ctx, rankKeyMcp, redis.Z{Score: score, Member: id})
	}

	newCandidates, _, err := s.mcpRepo.List(ctx, repo.McpServerQuery{
		Page: 1, PageSize: s.maxCand, Sort: "hot",
	})
	if err == nil {
		candidateSet := make(map[uint]bool, len(members))
		for _, idStr := range members {
			var id uint
			if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
				candidateSet[id] = true
			}
		}
		for _, sv := range newCandidates {
			if !candidateSet[sv.ID] && s.meetsMcpThreshold(&sv) {
				score := CalculateHotScore(sv.Views, sv.LikesCount, sv.FavoritesCount, sv.CommentsCount, sv.CreatedAt, s.gravity)
				pipe.ZAdd(ctx, rankKeyMcp, redis.Z{Score: score, Member: sv.ID})
			}
		}
	}

	pipe.ZRemRangeByRank(ctx, rankKeyMcp, 0, int64(-s.maxCand-1))

	_, err = pipe.Exec(ctx)
	return err
}

// --- Summary helpers (kept from Phase 1) ---

func rankingArticleSummary(article model.Article, tags []model.Tag) ArticleSummary {
	summary := ArticleSummary{
		ID: article.ID, Title: article.Title, Summary: article.Summary,
		Tags:  rankingTagBriefs(tags),
		Views: article.Views, LikesCount: article.LikesCount,
		FavoritesCount: article.FavoritesCount, CommentsCount: article.CommentsCount,
		Status: string(article.Status), PublishedAt: article.PublishedAt, Pinned: article.Pinned,
		Author: AuthorBrief{ID: article.AuthorID},
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
