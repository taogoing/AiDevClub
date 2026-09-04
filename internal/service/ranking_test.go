package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newRankingTestEnv(t *testing.T) (*RankingService, *gorm.DB, *redis.Client, *model.User) {
	t.Helper()
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	u := &model.User{Email: "rank@t.com", PasswordHash: "x", Nickname: "Ranker"}
	if err := repo.NewUserRepo(db).Create(u); err != nil {
		t.Fatal(err)
	}
	cfg := &platform.Config{
		RankingGravity:       1.5,
		RankingMaxCandidates: 1000,
		RankingMinLikes:      3,
		RankingMinFavorites:  2,
		RankingMinComments:   2,
		RankingMinViews:      50,
		RankingLocalCacheTTL: 10 * time.Second,
	}
	svc := NewRankingService(rdb, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db), cfg)
	return svc, db, rdb, u
}

func seedArticle(t *testing.T, db *gorm.DB, authorID uint, title string, views int) *model.Article {
	t.Helper()
	publishedAt := time.Now().Add(-2 * time.Hour)
	a := &model.Article{
		AuthorID: authorID, Title: title, Summary: "s", Content: "c",
		Status:      model.ArticleStatusPublished,
		Views:       views,
		PublishedAt: &publishedAt,
	}
	if err := db.Create(a).Error; err != nil {
		t.Fatal(err)
	}
	return a
}

func titleKey(id uint) string {
	return fmt.Sprintf(rankKeyArticleTitle, id)
}

func TestGetArticleHotBriefsOrdersByScoreAndReturnsTotal(t *testing.T) {
	svc, db, _, u := newRankingTestEnv(t)
	ctx := context.Background()
	a1 := seedArticle(t, db, u.ID, "first", 100)
	a2 := seedArticle(t, db, u.ID, "second", 300)
	a3 := seedArticle(t, db, u.ID, "third", 200)
	for _, a := range []*model.Article{a1, a2, a3} {
		if err := svc.UpdateArticleHotScore(ctx, a); err != nil {
			t.Fatal(err)
		}
	}

	briefs, total, err := svc.GetArticleHotBriefs(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(briefs) != 2 {
		t.Fatalf("len(briefs) = %d, want 2", len(briefs))
	}
	if briefs[0].ID != a2.ID || briefs[0].Title != "second" {
		t.Fatalf("briefs[0] = %+v, want article %d 'second'", briefs[0], a2.ID)
	}
	if briefs[1].ID != a3.ID || briefs[1].Title != "third" {
		t.Fatalf("briefs[1] = %+v, want article %d 'third'", briefs[1], a3.ID)
	}
}

func TestGetArticleHotBriefsServesLocalCacheUntilTTL(t *testing.T) {
	svc, db, rdb, u := newRankingTestEnv(t)
	ctx := context.Background()
	a1 := seedArticle(t, db, u.ID, "only", 100)
	if err := svc.UpdateArticleHotScore(ctx, a1); err != nil {
		t.Fatal(err)
	}

	briefs, _, err := svc.GetArticleHotBriefs(ctx, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(briefs) != 1 || briefs[0].ID != a1.ID {
		t.Fatalf("first call = %+v", briefs)
	}

	// 清空 ZSet 模拟热榜变化；本地缓存 10s 未过期，应仍返回旧快照
	if err := rdb.Del(ctx, rankKeyArticles).Err(); err != nil {
		t.Fatal(err)
	}
	briefs2, _, err := svc.GetArticleHotBriefs(ctx, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(briefs2) != 1 || briefs2[0].ID != a1.ID {
		t.Fatalf("second call = %+v, want cached snapshot", briefs2)
	}
}

func TestGetArticleHotBriefsBackfillsMissingTitlesToRedis(t *testing.T) {
	svc, db, rdb, u := newRankingTestEnv(t)
	ctx := context.Background()
	a1 := seedArticle(t, db, u.ID, "backfill-me", 100)
	// 直接 ZAdd（未经预热），标题字典 miss → 走 MySQL 补全
	if err := rdb.ZAdd(ctx, rankKeyArticles, redis.Z{Score: 10, Member: a1.ID}).Err(); err != nil {
		t.Fatal(err)
	}

	briefs, _, err := svc.GetArticleHotBriefs(ctx, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(briefs) != 1 || briefs[0].Title != "backfill-me" {
		t.Fatalf("briefs = %+v", briefs)
	}
	cached, err := rdb.Get(ctx, titleKey(a1.ID)).Result()
	if err != nil {
		t.Fatalf("title key not written back: %v", err)
	}
	if cached != "backfill-me" {
		t.Fatalf("cached title = %q, want 'backfill-me'", cached)
	}
}

func TestGetArticleHotBriefsPrefersCachedTitleOverMySQL(t *testing.T) {
	svc, db, rdb, u := newRankingTestEnv(t)
	ctx := context.Background()
	a1 := seedArticle(t, db, u.ID, "real-title", 100)
	if err := rdb.ZAdd(ctx, rankKeyArticles, redis.Z{Score: 10, Member: a1.ID}).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Set(ctx, titleKey(a1.ID), "cached-title", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	briefs, _, err := svc.GetArticleHotBriefs(ctx, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(briefs) != 1 || briefs[0].Title != "cached-title" {
		t.Fatalf("briefs = %+v, want Redis cached title 'cached-title'", briefs)
	}
}

func TestGetArticleHotBriefsSkipsHiddenArticlesOnBackfill(t *testing.T) {
	svc, db, rdb, u := newRankingTestEnv(t)
	ctx := context.Background()
	visible := seedArticle(t, db, u.ID, "visible", 100)
	hidden := seedArticle(t, db, u.ID, "hidden-one", 200)
	hidden.Hidden = true
	if err := db.Save(hidden).Error; err != nil {
		t.Fatal(err)
	}
	// 均无标题 key → 强制走 MySQL 补全路径
	if err := rdb.ZAdd(ctx, rankKeyArticles, redis.Z{Score: 20, Member: hidden.ID}).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.ZAdd(ctx, rankKeyArticles, redis.Z{Score: 10, Member: visible.ID}).Err(); err != nil {
		t.Fatal(err)
	}

	briefs, _, err := svc.GetArticleHotBriefs(ctx, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(briefs) != 1 || briefs[0].ID != visible.ID {
		t.Fatalf("briefs = %+v, want only visible article %d", briefs, visible.ID)
	}
	if rdb.Exists(ctx, titleKey(hidden.ID)).Val() != 0 {
		t.Fatal("hidden article title should not be cached")
	}
}

func TestRecalculateArticleHotRankingPrewarmsAndCleansTitles(t *testing.T) {
	svc, db, rdb, u := newRankingTestEnv(t)
	ctx := context.Background()
	hot := seedArticle(t, db, u.ID, "hot-one", 100)
	cold := seedArticle(t, db, u.ID, "cold-one", 1)
	hidden := seedArticle(t, db, u.ID, "hidden-one", 200)
	hidden.Hidden = true
	if err := db.Save(hidden).Error; err != nil {
		t.Fatal(err)
	}
	// hidden 手动进 ZSet 并预置标题 key，重算应剔除并删除 key
	if err := rdb.ZAdd(ctx, rankKeyArticles, redis.Z{Score: 20, Member: hidden.ID}).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Set(ctx, titleKey(hidden.ID), "stale", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	if err := svc.RecalculateArticleHotRanking(ctx); err != nil {
		t.Fatal(err)
	}

	if err := rdb.ZScore(ctx, rankKeyArticles, strconv.FormatUint(uint64(hot.ID), 10)).Err(); err != nil {
		t.Fatalf("hot article not in ZSet: %v", err)
	}
	cached, err := rdb.Get(ctx, titleKey(hot.ID)).Result()
	if err != nil || cached != "hot-one" {
		t.Fatalf("prewarmed title = %q, err = %v, want 'hot-one'", cached, err)
	}
	if rdb.Exists(ctx, titleKey(cold.ID)).Val() != 0 {
		t.Fatal("below-threshold article should not have title key")
	}
	if rdb.Exists(ctx, titleKey(hidden.ID)).Val() != 0 {
		t.Fatal("removed candidate title key not deleted")
	}
	if err := rdb.ZScore(ctx, rankKeyArticles, strconv.FormatUint(uint64(hidden.ID), 10)).Err(); err == nil {
		t.Fatal("hidden article should be removed from ZSet")
	}
}

func TestGetArticleHotBriefsFallsBackToMySQLWhenRedisUnavailable(t *testing.T) {
	svc, db, rdb, u := newRankingTestEnv(t)
	ctx := context.Background()
	seedArticle(t, db, u.ID, "low", 100)
	a2 := seedArticle(t, db, u.ID, "high", 300)
	a3 := seedArticle(t, db, u.ID, "mid", 200)
	_ = a2
	_ = a3

	// 关闭 Redis → ZREVRANGE 失败 → 降级 MySQL 热度排序
	if err := rdb.Close(); err != nil {
		t.Fatal(err)
	}

	briefs, total, err := svc.GetArticleHotBriefs(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(briefs) != 2 {
		t.Fatalf("len(briefs) = %d, want 2", len(briefs))
	}
	if briefs[0].Title != "high" || briefs[1].Title != "mid" {
		t.Fatalf("fallback briefs = %+v, want ['high', 'mid']", briefs)
	}
}
