package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type RankedContentType string

const (
	RankedContentArticle   RankedContentType = "article"
	RankedContentSkill     RankedContentType = "skill"
	RankedContentMcpServer RankedContentType = "mcp_server"
)

const (
	dailyRankTTL     = 31 * 24 * time.Hour
	dailyTopCacheTTL = 10 * time.Second
	dailyTopPageSize = 5
)

// cstZone 日榜日界固定为 UTC+8，不依赖系统时区、环境变量或 tzdata。
var cstZone = time.FixedZone("CST", 8*3600)

type ContentRankingService struct {
	rdb         *redis.Client
	articleRepo *repo.ArticleRepo
	skillRepo   *repo.SkillRepo
	mcpRepo     *repo.McpServerRepo
	now         func() time.Time

	mu                  sync.Mutex
	articleTop          topSnapshot[HotArticleBrief]
	skillTop            topSnapshot[HotSkillBrief]
	mcpTop              topSnapshot[HotMcpServerBrief]
	topRefresh          singleflight.Group
	topRefreshStateMu   sync.Mutex
	topRefreshActive    bool
	topRefreshHook      func()
	singleflightEnabled bool
	refreshTotal        atomic.Uint64
	refreshShared       atomic.Uint64
	refreshInFlight     atomic.Int64
	refreshDurationNs   atomic.Uint64
	refreshDurationCnt  atomic.Uint64
	refreshBuckets      [7]atomic.Uint64
}

type RankingMetricsSnapshot struct {
	RefreshTotal         uint64  `json:"refresh_total"`
	RefreshShared        uint64  `json:"refresh_shared"`
	RefreshInFlight      int64   `json:"refresh_inflight"`
	RefreshDurationCount uint64  `json:"refresh_duration_count"`
	RefreshAvgMs         float64 `json:"refresh_avg_ms"`
	RefreshDurationP95Ms int     `json:"refresh_duration_p95_ms"`
}

var refreshDurationBounds = [...]time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second, 2 * time.Second}

func (s *ContentRankingService) recordRefreshDuration(d time.Duration) {
	s.refreshDurationNs.Add(uint64(d))
	s.refreshDurationCnt.Add(1)
	idx := len(refreshDurationBounds)
	for i, bound := range refreshDurationBounds {
		if d <= bound {
			idx = i
			break
		}
	}
	s.refreshBuckets[idx].Add(1)
}

func (s *ContentRankingService) RankingMetrics() RankingMetricsSnapshot {
	total := s.refreshDurationCnt.Load()
	avg := float64(0)
	if total > 0 {
		avg = float64(s.refreshDurationNs.Load()) / float64(total) / float64(time.Millisecond)
	}
	p95 := 0
	if total > 0 {
		target := (total*95 + 99) / 100
		var cumulative uint64
		for i := range s.refreshBuckets {
			cumulative += s.refreshBuckets[i].Load()
			if cumulative >= target {
				if i < len(refreshDurationBounds) {
					p95 = int(refreshDurationBounds[i] / time.Millisecond)
				} else {
					p95 = int(2 * time.Second / time.Millisecond)
				}
				break
			}
		}
	}
	return RankingMetricsSnapshot{
		RefreshTotal: s.refreshTotal.Load(), RefreshShared: s.refreshShared.Load(),
		RefreshInFlight: s.refreshInFlight.Load(), RefreshDurationCount: total,
		RefreshAvgMs: avg, RefreshDurationP95Ms: p95,
	}
}

type topSnapshot[T any] struct {
	expiresAt time.Time
	items     []T
	total     int64
}

func NewContentRankingService(
	rdb *redis.Client,
	articleRepo *repo.ArticleRepo,
	skillRepo *repo.SkillRepo,
	mcpRepo *repo.McpServerRepo,
) *ContentRankingService {
	return &ContentRankingService{
		rdb: rdb, articleRepo: articleRepo, skillRepo: skillRepo, mcpRepo: mcpRepo,
		singleflightEnabled: true,
		now:                 time.Now,
	}
}

func (s *ContentRankingService) SetSingleflightEnabled(enabled bool) {
	s.singleflightEnabled = enabled
}

// freshTop/storeTop 为包级泛型函数：Go 方法不能声明自有类型参数。
func freshTop[T any](now func() time.Time, snap *topSnapshot[T]) ([]T, int64, bool) {
	if snap.items != nil && now().Before(snap.expiresAt) {
		return snap.items, snap.total, true
	}
	return nil, 0, false
}

func storeTop[T any](now func() time.Time, snap *topSnapshot[T], items []T, total int64) {
	snap.items = items
	snap.total = total
	snap.expiresAt = now().Add(dailyTopCacheTTL)
}

func (s *ContentRankingService) dailyKey(contentType RankedContentType) string {
	return fmt.Sprintf("content_hot_rank:daily:%s:%s", contentType, s.now().In(cstZone).Format("20060102"))
}

func (s *ContentRankingService) viewKey(contentType RankedContentType, contentID, userID uint) string {
	return fmt.Sprintf("content_hot_view:%s:%s:%d:%d",
		s.now().In(cstZone).Format("20060102"), contentType, contentID, userID)
}

func (s *ContentRankingService) secondsUntilDayEnd() int {
	now := s.now().In(cstZone)
	end := time.Date(now.Year(), now.Month(), now.Day(), 24, 0, 0, 0, cstZone)
	sec := int(end.Sub(now).Seconds()) + 1
	if sec < 1 {
		return 1
	}
	return sec
}

func memberID(id uint) string { return strconv.FormatUint(uint64(id), 10) }

func (s *ContentRankingService) logRankErr(action string, contentType RankedContentType, contentID uint, delta int64, err error) {
	slog.Error("daily ranking redis call failed",
		"action", action, "content_type", contentType, "content_id", contentID, "delta", delta, "err", err)
}

// AddScore 对当日日榜执行一次 ZINCRBY；扣减后新分数 <= 0 时立即 ZREM；Key 首次写入设置 31 天 TTL。
func (s *ContentRankingService) AddScore(ctx context.Context, contentType RankedContentType, contentID uint, delta int64) error {
	key := s.dailyKey(contentType)
	newScore, err := s.rdb.ZIncrBy(ctx, key, float64(delta), memberID(contentID)).Result()
	if err != nil {
		s.logRankErr("add_score", contentType, contentID, delta, err)
		return err
	}
	if newScore <= 0 {
		if err := s.rdb.ZRem(ctx, key, memberID(contentID)).Err(); err != nil {
			s.logRankErr("remove_nonpositive", contentType, contentID, delta, err)
		}
		return nil
	}
	if ttl, err := s.rdb.TTL(ctx, key).Result(); err == nil && ttl == -1 {
		if err := s.rdb.Expire(ctx, key, dailyRankTTL).Err(); err != nil {
			s.logRankErr("expire", contentType, contentID, delta, err)
		}
	}
	return nil
}

// RecordView 登录用户当天首次浏览计 +1（SETNX 去重）；返回是否计分。
func (s *ContentRankingService) RecordView(ctx context.Context, contentType RankedContentType, contentID, userID uint) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	ok, err := s.rdb.SetNX(ctx, s.viewKey(contentType, contentID, userID), 1,
		time.Duration(s.secondsUntilDayEnd())*time.Second).Result()
	if err != nil {
		s.logRankErr("record_view", contentType, contentID, 1, err)
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := s.AddScore(ctx, contentType, contentID, 1); err != nil {
		return false, err
	}
	return true, nil
}

// Remove 从当日日榜移除成员（内容不可公开时调用）。
func (s *ContentRankingService) Remove(ctx context.Context, contentType RankedContentType, contentID uint) error {
	err := s.rdb.ZRem(ctx, s.dailyKey(contentType), memberID(contentID)).Err()
	if err != nil {
		s.logRankErr("remove", contentType, contentID, 0, err)
	}
	return err
}

type zsetPage struct {
	ids    []uint
	scores []int64
	total  int64
}

func (s *ContentRankingService) readPage(ctx context.Context, contentType RankedContentType, page, pageSize int) (*zsetPage, error) {
	key := s.dailyKey(contentType)
	members, err := s.rdb.ZRevRangeWithScores(ctx, key, int64((page-1)*pageSize), int64(page*pageSize-1)).Result()
	if err != nil {
		return nil, err
	}
	total, err := s.rdb.ZCard(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	out := &zsetPage{ids: make([]uint, 0, len(members)), scores: make([]int64, 0, len(members)), total: total}
	for _, m := range members {
		id, err := strconv.ParseUint(m.Member.(string), 10, 64)
		if err != nil {
			continue
		}
		out.ids = append(out.ids, uint(id))
		out.scores = append(out.scores, int64(m.Score))
	}
	return out, nil
}

func (s *ContentRankingService) isTop5(page, pageSize int) bool {
	return page == 1 && pageSize == dailyTopPageSize
}

func (s *ContentRankingService) listArticlesUncached(ctx context.Context, page, pageSize int) ([]HotArticleBrief, int64, error) {
	p, err := s.readPage(ctx, RankedContentArticle, page, pageSize)
	if err != nil {
		s.logRankErr("list", RankedContentArticle, 0, 0, err)
		return []HotArticleBrief{}, 0, nil
	}
	rows, err := s.articleRepo.ListTitlesByIDs(ctx, p.ids)
	if err != nil {
		s.logRankErr("hydrate", RankedContentArticle, 0, 0, err)
		return []HotArticleBrief{}, 0, nil
	}
	byID := make(map[uint]model.Article, len(rows))
	for _, a := range rows {
		byID[a.ID] = a
	}
	items := make([]HotArticleBrief, 0, len(p.ids))
	pruned := 0
	for i, id := range p.ids {
		a, ok := byID[id]
		if !ok {
			_ = s.Remove(ctx, RankedContentArticle, id)
			pruned++
			continue
		}
		items = append(items, HotArticleBrief{ID: a.ID, Title: a.Title, Score: p.scores[i]})
	}
	total := p.total - int64(pruned)
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		storeTop(s.now, &s.articleTop, items, total)
		s.mu.Unlock()
	}
	return items, total, nil
}

func (s *ContentRankingService) ListArticles(ctx context.Context, page, pageSize int) ([]HotArticleBrief, int64, error) {
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		items, total, ok := freshTop(s.now, &s.articleTop)
		s.mu.Unlock()
		if ok {
			return items, total, nil
		}
		refresh := func() (articleTopResult, error) {
			s.mu.Lock()
			items, total, ok := freshTop(s.now, &s.articleTop)
			s.mu.Unlock()
			if ok {
				return articleTopResult{items: items, total: total}, nil
			}
			if s.topRefreshHook != nil {
				s.topRefreshHook()
			}
			s.refreshTotal.Add(1)
			s.refreshInFlight.Add(1)
			started := time.Now()
			defer func() {
				s.refreshInFlight.Add(-1)
				s.recordRefreshDuration(time.Since(started))
			}()
			items, total, err := s.listArticlesUncached(ctx, page, pageSize)
			return articleTopResult{items: items, total: total}, err
		}
		if !s.singleflightEnabled {
			result, err := refresh()
			if err != nil {
				return []HotArticleBrief{}, 0, nil
			}
			return result.items, result.total, nil
		}
		s.topRefreshStateMu.Lock()
		leader := !s.topRefreshActive
		if leader {
			s.topRefreshActive = true
		} else {
			s.refreshShared.Add(1)
		}
		s.topRefreshStateMu.Unlock()
		if leader {
			defer func() {
				s.topRefreshStateMu.Lock()
				s.topRefreshActive = false
				s.topRefreshStateMu.Unlock()
			}()
		}
		value, err, _ := s.topRefresh.Do("article", func() (any, error) { return refresh() })
		if err != nil {
			return []HotArticleBrief{}, 0, nil
		}
		result := value.(articleTopResult)
		return result.items, result.total, nil
	}
	return s.listArticlesUncached(ctx, page, pageSize)
}

type articleTopResult struct {
	items []HotArticleBrief
	total int64
}

func (s *ContentRankingService) ListSkills(ctx context.Context, page, pageSize int) ([]HotSkillBrief, int64, error) {
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		items, total, ok := freshTop(s.now, &s.skillTop)
		s.mu.Unlock()
		if ok {
			return items, total, nil
		}
	}
	p, err := s.readPage(ctx, RankedContentSkill, page, pageSize)
	if err != nil {
		s.logRankErr("list", RankedContentSkill, 0, 0, err)
		items, total := []HotSkillBrief{}, int64(0)
		if s.isTop5(page, pageSize) {
			s.mu.Lock()
			storeTop(s.now, &s.skillTop, items, total)
			s.mu.Unlock()
		}
		return items, total, nil
	}
	rows, err := s.skillRepo.ListNamesByIDs(ctx, p.ids)
	if err != nil {
		s.logRankErr("hydrate", RankedContentSkill, 0, 0, err)
		return []HotSkillBrief{}, 0, nil
	}
	byID := make(map[uint]model.Skill, len(rows))
	for _, sk := range rows {
		byID[sk.ID] = sk
	}
	items := make([]HotSkillBrief, 0, len(p.ids))
	pruned := 0
	for i, id := range p.ids {
		sk, ok := byID[id]
		if !ok {
			_ = s.Remove(ctx, RankedContentSkill, id)
			pruned++
			continue
		}
		items = append(items, HotSkillBrief{ID: sk.ID, Name: sk.Name, Score: p.scores[i]})
	}
	total := p.total - int64(pruned)
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		storeTop(s.now, &s.skillTop, items, total)
		s.mu.Unlock()
	}
	return items, total, nil
}

func (s *ContentRankingService) ListMcpServers(ctx context.Context, page, pageSize int) ([]HotMcpServerBrief, int64, error) {
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		items, total, ok := freshTop(s.now, &s.mcpTop)
		s.mu.Unlock()
		if ok {
			return items, total, nil
		}
	}
	p, err := s.readPage(ctx, RankedContentMcpServer, page, pageSize)
	if err != nil {
		s.logRankErr("list", RankedContentMcpServer, 0, 0, err)
		items, total := []HotMcpServerBrief{}, int64(0)
		if s.isTop5(page, pageSize) {
			s.mu.Lock()
			storeTop(s.now, &s.mcpTop, items, total)
			s.mu.Unlock()
		}
		return items, total, nil
	}
	rows, err := s.mcpRepo.ListNamesByIDs(ctx, p.ids)
	if err != nil {
		s.logRankErr("hydrate", RankedContentMcpServer, 0, 0, err)
		return []HotMcpServerBrief{}, 0, nil
	}
	byID := make(map[uint]model.McpServer, len(rows))
	for _, sv := range rows {
		byID[sv.ID] = sv
	}
	items := make([]HotMcpServerBrief, 0, len(p.ids))
	pruned := 0
	for i, id := range p.ids {
		sv, ok := byID[id]
		if !ok {
			_ = s.Remove(ctx, RankedContentMcpServer, id)
			pruned++
			continue
		}
		items = append(items, HotMcpServerBrief{ID: sv.ID, Name: sv.Name, Score: p.scores[i]})
	}
	total := p.total - int64(pruned)
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		storeTop(s.now, &s.mcpTop, items, total)
		s.mu.Unlock()
	}
	return items, total, nil
}

// rankedResourceType 把资源评论的 resourceType 映射为日榜内容类型。
func rankedResourceType(resourceType string) (RankedContentType, bool) {
	switch resourceType {
	case "skill":
		return RankedContentSkill, true
	case "mcp_server":
		return RankedContentMcpServer, true
	}
	return "", false
}
