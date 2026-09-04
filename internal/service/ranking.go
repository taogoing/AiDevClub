package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

const (
	rankKeyArticles = "rank:articles:hot"
	rankKeySkills   = "rank:skills:hot"
	rankKeyMcp      = "rank:mcp_servers:hot"

	rankKeyArticleTitle = "rank:articles:hot:title:%d"
	rankTitleTTL        = 5 * time.Minute
)

type localCacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

type articleCacheEntry struct {
	articles []ArticleSummary
	total    int64
}

type skillCacheEntry struct {
	skills []SkillSummary
	total  int64
}

type mcpCacheEntry struct {
	servers []McpServerSummary
	total   int64
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
	sfGroup     singleflight.Group
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

func (s *RankingService) ListArticleHot(ctx context.Context, page, pageSize int) ([]ArticleSummary, int64, error) {
	cacheKey := fmt.Sprintf("article:%d:%d", page, pageSize)
	if cached, ok := s.cache.get(cacheKey); ok {
		entry := cached.(articleCacheEntry)
		return entry.articles, entry.total, nil
	}

	result, err, _ := s.sfGroup.Do(cacheKey, func() (interface{}, error) {
		if cached, ok := s.cache.get(cacheKey); ok {
			entry := cached.(articleCacheEntry)
			return articleCacheEntry{articles: entry.articles, total: entry.total}, nil
		}

		ids, err := s.GetArticleHotRanking(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return articleCacheEntry{articles: []ArticleSummary{}, total: 0}, nil
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

	total, _ := s.rdb.ZCard(ctx, rankKeyArticles).Result()

	s.cache.set(cacheKey, articleCacheEntry{articles: result, total: total})
		return articleCacheEntry{articles: result, total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	entry := result.(articleCacheEntry)
	return entry.articles, entry.total, nil
}

// --- REST 热榜摘要（本地缓存 + Redis 标题字典，不查 MySQL） ---

type articleBriefCacheEntry struct {
	briefs []HotArticleBrief
	total  int64
}

// GetArticleHotBriefs 返回热榜当前页的 {id, title} 摘要，排名即数组顺序。
// 本地缓存 → singleflight → ZREVRANGE + 标题字典 MGET → MySQL 补全兜底。
// ListArticleHot 保留给 MCP browse_content（需要完整 summary），两者互不影响。
func (s *RankingService) GetArticleHotBriefs(ctx context.Context, page, pageSize int) ([]HotArticleBrief, int64, error) {
	cacheKey := fmt.Sprintf("article-brief:%d:%d", page, pageSize)
	if cached, ok := s.cache.get(cacheKey); ok {
		entry := cached.(articleBriefCacheEntry)
		return entry.briefs, entry.total, nil
	}

	result, err, _ := s.sfGroup.Do(cacheKey, func() (interface{}, error) {
		if cached, ok := s.cache.get(cacheKey); ok {
			entry := cached.(articleBriefCacheEntry)
			return articleBriefCacheEntry{briefs: entry.briefs, total: entry.total}, nil
		}
		briefs, total, err := s.buildArticleHotBriefs(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}
		s.cache.set(cacheKey, articleBriefCacheEntry{briefs: briefs, total: total})
		return articleBriefCacheEntry{briefs: briefs, total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	entry := result.(articleBriefCacheEntry)
	return entry.briefs, entry.total, nil
}

func (s *RankingService) buildArticleHotBriefs(ctx context.Context, page, pageSize int) ([]HotArticleBrief, int64, error) {
	ids, err := s.GetArticleHotRanking(ctx, page, pageSize)
	if err != nil {
		// Redis 不可用：降级 MySQL 热度排序
		return s.articleHotBriefsFromMySQL(ctx, page, pageSize)
	}
	if len(ids) == 0 {
		return []HotArticleBrief{}, 0, nil
	}
	titles, err := s.articleTitles(ctx, ids)
	if err != nil {
		return s.articleHotBriefsFromMySQL(ctx, page, pageSize)
	}
	briefs := make([]HotArticleBrief, 0, len(ids))
	for _, id := range ids {
		if title, ok := titles[id]; ok {
			briefs = append(briefs, HotArticleBrief{ID: id, Title: title})
		}
	}
	total, _ := s.rdb.ZCard(ctx, rankKeyArticles).Result()
	if total < int64(len(briefs)) {
		total = int64(len(briefs))
	}
	return briefs, total, nil
}

// articleTitles 用 MGET 从标题字典批量取标题，miss 的从 MySQL 补全（过滤
// 非 published / hidden）并 best-effort 回写 Redis。
func (s *RankingService) articleTitles(ctx context.Context, ids []uint) (map[uint]string, error) {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf(rankKeyArticleTitle, id)
	}
	values, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	titles := make(map[uint]string, len(ids))
	var missIDs []uint
	for i, v := range values {
		if str, ok := v.(string); ok {
			titles[ids[i]] = str
		} else {
			missIDs = append(missIDs, ids[i])
		}
	}
	if len(missIDs) == 0 {
		return titles, nil
	}
	var articles []model.Article
	if err := s.articleRepo.DB().WithContext(ctx).
		Where("id IN ?", missIDs).
		Where("status = ?", model.ArticleStatusPublished).
		Where("hidden = ?", false).
		Select("id, title").
		Find(&articles).Error; err != nil {
		return nil, err
	}
	pipe := s.rdb.Pipeline()
	for _, a := range articles {
		titles[a.ID] = a.Title
		pipe.Set(ctx, fmt.Sprintf(rankKeyArticleTitle, a.ID), a.Title, rankTitleTTL)
	}
	_, _ = pipe.Exec(ctx)
	return titles, nil
}

func (s *RankingService) articleHotBriefsFromMySQL(ctx context.Context, page, pageSize int) ([]HotArticleBrief, int64, error) {
	list, total, err := s.articleRepo.List(ctx, repo.ArticleQuery{Page: page, PageSize: pageSize, Sort: "hot"})
	if err != nil {
		return nil, 0, err
	}
	briefs := make([]HotArticleBrief, 0, len(list))
	for _, a := range list {
		briefs = append(briefs, HotArticleBrief{ID: a.ID, Title: a.Title})
	}
	return briefs, total, nil
}

func (s *RankingService) ListSkillHot(ctx context.Context, page, pageSize int) ([]SkillSummary, int64, error) {
	cacheKey := fmt.Sprintf("skill:%d:%d", page, pageSize)
	if cached, ok := s.cache.get(cacheKey); ok {
		entry := cached.(skillCacheEntry)
		return entry.skills, entry.total, nil
	}

	result, err, _ := s.sfGroup.Do(cacheKey, func() (interface{}, error) {
		if cached, ok := s.cache.get(cacheKey); ok {
			entry := cached.(skillCacheEntry)
			return skillCacheEntry{skills: entry.skills, total: entry.total}, nil
		}

		ids, err := s.GetSkillHotRanking(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return skillCacheEntry{skills: []SkillSummary{}, total: 0}, nil
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

	total, _ := s.rdb.ZCard(ctx, rankKeySkills).Result()

	s.cache.set(cacheKey, skillCacheEntry{skills: result, total: total})
		return skillCacheEntry{skills: result, total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	entry := result.(skillCacheEntry)
	return entry.skills, entry.total, nil
}

func (s *RankingService) ListMcpServerHot(ctx context.Context, page, pageSize int) ([]McpServerSummary, int64, error) {
	cacheKey := fmt.Sprintf("mcp:%d:%d", page, pageSize)
	if cached, ok := s.cache.get(cacheKey); ok {
		entry := cached.(mcpCacheEntry)
		return entry.servers, entry.total, nil
	}

	result, err, _ := s.sfGroup.Do(cacheKey, func() (interface{}, error) {
		if cached, ok := s.cache.get(cacheKey); ok {
			entry := cached.(mcpCacheEntry)
			return mcpCacheEntry{servers: entry.servers, total: entry.total}, nil
		}

		ids, err := s.GetMcpServerHotRanking(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return mcpCacheEntry{servers: []McpServerSummary{}, total: 0}, nil
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

	total, _ := s.rdb.ZCard(ctx, rankKeyMcp).Result()

	s.cache.set(cacheKey, mcpCacheEntry{servers: result, total: total})
		return mcpCacheEntry{servers: result, total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	entry := result.(mcpCacheEntry)
	return entry.servers, entry.total, nil
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
			pipe.Del(ctx, fmt.Sprintf(rankKeyArticleTitle, id))
			continue
		}
		if a.Status != model.ArticleStatusPublished || a.Hidden {
			pipe.ZRem(ctx, rankKeyArticles, id)
			pipe.Del(ctx, fmt.Sprintf(rankKeyArticleTitle, id))
			continue
		}
		publishedAt := a.PublishedAt
		if publishedAt == nil {
			publishedAt = &a.CreatedAt
		}
		if !s.meetsThreshold(a) {
			pipe.ZRem(ctx, rankKeyArticles, id)
			pipe.Del(ctx, fmt.Sprintf(rankKeyArticleTitle, id))
			continue
		}
		score := CalculateHotScore(a.Views, a.LikesCount, a.FavoritesCount, a.CommentsCount, *publishedAt, s.gravity)
		pipe.ZAdd(ctx, rankKeyArticles, redis.Z{Score: score, Member: id})
		// 预热标题字典（TTL 随每次重算续期）
		pipe.Set(ctx, fmt.Sprintf(rankKeyArticleTitle, a.ID), a.Title, rankTitleTTL)
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
				pipe.Set(ctx, fmt.Sprintf(rankKeyArticleTitle, a.ID), a.Title, rankTitleTTL)
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
