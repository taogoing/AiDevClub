package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

// --- 共享辅助（任务 2~4 的测试复用，勿重名） ---

func newContentRankingEnv(t *testing.T) (*ContentRankingService, *gorm.DB, *redis.Client) {
	t.Helper()
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	seedUser(t, db, "rank-seed@t.com") // ID=1：满足 articles/skills/mcp_servers 的 author 外键
	svc := NewContentRankingService(rdb, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	return svc, db, rdb
}

func deadRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", DialTimeout: 100 * time.Millisecond, ReadTimeout: 100 * time.Millisecond,
	})
}

func seedArticle(t *testing.T, db *gorm.DB, authorID uint, title string, status model.ArticleStatus, hidden bool) *model.Article {
	t.Helper()
	a := &model.Article{AuthorID: authorID, Title: title, Status: status, Hidden: hidden}
	if err := db.Create(a).Error; err != nil {
		t.Fatal(err)
	}
	return a
}

func seedSkill(t *testing.T, db *gorm.DB, authorID uint, name string, status model.ResourceStatus, hidden bool) *model.Skill {
	t.Helper()
	sk := &model.Skill{AuthorID: authorID, Name: name, Status: status, Hidden: hidden}
	if err := db.Create(sk).Error; err != nil {
		t.Fatal(err)
	}
	return sk
}

func seedMcpServer(t *testing.T, db *gorm.DB, authorID uint, name string, status model.ResourceStatus, hidden bool) *model.McpServer {
	t.Helper()
	sv := &model.McpServer{AuthorID: authorID, Name: name, Status: status, Hidden: hidden, InstallationsJSON: "[]"}
	if err := db.Create(sv).Error; err != nil {
		t.Fatal(err)
	}
	return sv
}

func seedUser(t *testing.T, db *gorm.DB, email string) *model.User {
	t.Helper()
	u := &model.User{Email: email, PasswordHash: "x", Nickname: email, AvatarURL: "/x.png"}
	if err := db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

// --- 测试（读取一律用 (1,10) 绕开 Top 5 快照；快照行为单独测） ---

func TestContentAddScoreAccumulateAndDeduct(t *testing.T) {
	svc, db, _ := newContentRankingEnv(t)
	ctx := context.Background()
	a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)

	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 2)
	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 3)
	items, total, err := svc.ListArticles(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != a.ID || items[0].Score != 5 || items[0].Title != "A" {
		t.Fatalf("items=%+v total=%d", items, total)
	}

	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, -1)
	items, _, _ = svc.ListArticles(ctx, 1, 10)
	if items[0].Score != 4 {
		t.Fatalf("score=%d want 4", items[0].Score)
	}
}

func TestContentDailyKeyFormatAndTTL(t *testing.T) {
	svc, db, rdb := newContentRankingEnv(t)
	ctx := context.Background()
	a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)

	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 2)

	key := "content_hot_rank:daily:article:" + time.Now().In(cstZone).Format("20060102")
	if n := rdb.ZCard(ctx, key).Val(); n != 1 {
		t.Fatalf("zcard(%s)=%d want 1", key, n)
	}
	ttl := rdb.TTL(ctx, key).Val()
	if ttl < 30*24*time.Hour || ttl > 31*24*time.Hour {
		t.Fatalf("ttl=%v want ~31d", ttl)
	}
}

func TestContentListDescendingAndFiltersHidden(t *testing.T) {
	svc, db, rdb := newContentRankingEnv(t)
	ctx := context.Background()
	a1 := seedArticle(t, db, 1, "one", model.ArticleStatusPublished, false)
	a2 := seedArticle(t, db, 1, "two", model.ArticleStatusPublished, false)
	a3 := seedArticle(t, db, 1, "hid", model.ArticleStatusPublished, true) // 隐藏
	a4 := seedArticle(t, db, 1, "drf", model.ArticleStatusDraft, false)    // 草稿

	_ = svc.AddScore(ctx, RankedContentArticle, a1.ID, 1)
	_ = svc.AddScore(ctx, RankedContentArticle, a2.ID, 5)
	_ = svc.AddScore(ctx, RankedContentArticle, a3.ID, 9)
	_ = svc.AddScore(ctx, RankedContentArticle, a4.ID, 9)

	items, total, err := svc.ListArticles(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != a2.ID || items[0].Score != 5 || items[1].ID != a1.ID || items[1].Score != 1 {
		t.Fatalf("items=%+v", items)
	}
	if total != 2 {
		t.Fatalf("total=%d want 2", total)
	}
	// best-effort：不可见成员被从 ZSet 移除
	key := "content_hot_rank:daily:article:" + time.Now().In(cstZone).Format("20060102")
	if n := rdb.ZCard(ctx, key).Val(); n != 2 {
		t.Fatalf("zcard=%d want 2 (invisible members pruned)", n)
	}
}

func TestContentRecordViewDedupAndGuest(t *testing.T) {
	svc, db, _ := newContentRankingEnv(t)
	ctx := context.Background()
	a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)

	counted, err := svc.RecordView(ctx, RankedContentArticle, a.ID, 7)
	if err != nil || !counted {
		t.Fatalf("first view counted=%v err=%v", counted, err)
	}
	counted, _ = svc.RecordView(ctx, RankedContentArticle, a.ID, 7)
	if counted {
		t.Fatal("repeat view must not count")
	}
	counted, err = svc.RecordView(ctx, RankedContentArticle, a.ID, 0)
	if err != nil || counted {
		t.Fatalf("guest view counted=%v err=%v", counted, err)
	}
	items, _, _ := svc.ListArticles(ctx, 1, 10)
	if len(items) != 1 || items[0].Score != 1 {
		t.Fatalf("items=%+v want single score 1", items)
	}
}

func TestContentRemoveDropsMember(t *testing.T) {
	svc, db, _ := newContentRankingEnv(t)
	ctx := context.Background()
	a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)
	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 4)

	if err := svc.Remove(ctx, RankedContentArticle, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := svc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("items=%+v total=%d want empty", items, total)
	}
}

func TestContentAddScoreNonPositiveRemovesMember(t *testing.T) {
	svc, db, rdb := newContentRankingEnv(t)
	ctx := context.Background()
	a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)

	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 2)
	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, -2) // 归零 -> 移除
	items, total, _ := svc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("after like+unlike items=%+v total=%d want empty", items, total)
	}

	b := seedArticle(t, db, 1, "B", model.ArticleStatusPublished, false)
	_ = svc.AddScore(ctx, RankedContentArticle, b.ID, -5) // 直接负分 -> 移除
	key := "content_hot_rank:daily:article:" + time.Now().In(cstZone).Format("20060102")
	if n := rdb.ZCard(ctx, key).Val(); n != 0 {
		t.Fatalf("zcard=%d want 0", n)
	}
}

func TestContentDailyKeyUTC8Boundary(t *testing.T) {
	svc, db, rdb := newContentRankingEnv(t)
	ctx := context.Background()
	a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)

	// 2026-09-05 23:59 UTC == 北京 2026-09-06 07:59 -> 当日 Key 应为 20260906
	svc.now = func() time.Time { return time.Date(2026, 9, 5, 23, 59, 0, 0, time.UTC) }
	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 1)
	if n := rdb.ZCard(ctx, "content_hot_rank:daily:article:20260906").Val(); n != 1 {
		t.Fatalf("utc8 day key 20260906 zcard=%d", n)
	}

	// 2026-09-06 16:01 UTC == 北京 2026-09-07 00:01 -> 切到 20260907
	svc.now = func() time.Time { return time.Date(2026, 9, 6, 16, 1, 0, 0, time.UTC) }
	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 1)
	if n := rdb.ZCard(ctx, "content_hot_rank:daily:article:20260907").Val(); n != 1 {
		t.Fatalf("utc8 day key 20260907 zcard=%d", n)
	}
	if n := rdb.ZCard(ctx, "content_hot_rank:daily:article:20260906").Val(); n != 1 {
		t.Fatalf("old day key must be untouched, zcard=%d", n)
	}
}

func TestContentTop5SnapshotServedWithoutRedis(t *testing.T) {
	svc, db, _ := newContentRankingEnv(t)
	ctx := context.Background()
	a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)
	_ = svc.AddScore(ctx, RankedContentArticle, a.ID, 2)

	items1, total1, err := svc.ListArticles(ctx, 1, 5) // 回源并填充快照
	if err != nil || len(items1) != 1 || total1 != 1 {
		t.Fatalf("items1=%+v err=%v", items1, err)
	}

	svc.rdb = deadRedisClient() // Redis 故障

	items2, total2, err := svc.ListArticles(ctx, 1, 5) // TTL 内命中快照
	if err != nil || len(items2) != 1 || total2 != 1 || items2[0].Title != "A" {
		t.Fatalf("snapshot must serve without redis: items=%+v err=%v", items2, err)
	}

	// 推进时钟越过 TTL：降级空结果也写回快照
	base := svc.now
	svc.now = func() time.Time { return base().Add(10 * time.Second) }
	items3, total3, err := svc.ListArticles(ctx, 1, 5)
	if err != nil || len(items3) != 0 || total3 != 0 {
		t.Fatalf("degraded items=%+v total=%d err=%v want empty/0/nil", items3, total3, err)
	}
	items4, total4, err := svc.ListArticles(ctx, 1, 5) // 降级空结果在 TTL 内直接复用
	if err != nil || len(items4) != 0 || total4 != 0 {
		t.Fatalf("degraded snapshot items=%+v err=%v", items4, err)
	}
}

func TestContentNonTop5BypassesSnapshot(t *testing.T) {
	svc, db, _ := newContentRankingEnv(t)
	ctx := context.Background()
	for i := 0; i < 6; i++ {
		a := seedArticle(t, db, 1, "A", model.ArticleStatusPublished, false)
		_ = svc.AddScore(ctx, RankedContentArticle, a.ID, int64(i+1))
	}

	items, total, err := svc.ListArticles(ctx, 1, 10) // pageSize != 5 -> 不经过快照
	if err != nil || len(items) != 6 || total != 6 {
		t.Fatalf("items=%d total=%d err=%v", len(items), total, err)
	}
	page2, _, _ := svc.ListArticles(ctx, 2, 5) // page != 1 -> 不经过快照
	if len(page2) != 1 {
		t.Fatalf("page2 len=%d want 1", len(page2))
	}

	svc.rdb = deadRedisClient()
	items, _, err = svc.ListArticles(ctx, 1, 5) // (1,10)/(2,5) 未填快照 -> 必须降级空
	if err != nil || len(items) != 0 {
		t.Fatalf("non-top5 must not populate snapshot: items=%+v err=%v", items, err)
	}
	items, _, err = svc.ListArticles(ctx, 2, 5) // 直连请求 Redis 故障 -> 降级空
	if err != nil || len(items) != 0 {
		t.Fatalf("direct degrade items=%+v err=%v", items, err)
	}
}

func TestContentSkillAndMcpDailyRanking(t *testing.T) {
	svc, db, _ := newContentRankingEnv(t)
	ctx := context.Background()
	sk := seedSkill(t, db, 1, "S", model.ResourceStatusPublished, false)
	sv := seedMcpServer(t, db, 1, "M", model.ResourceStatusPublished, false)

	_ = svc.AddScore(ctx, RankedContentSkill, sk.ID, 2)
	_ = svc.AddScore(ctx, RankedContentMcpServer, sv.ID, 3)

	skills, total, err := svc.ListSkills(ctx, 1, 10)
	if err != nil || total != 1 || len(skills) != 1 || skills[0].Name != "S" || skills[0].Score != 2 {
		t.Fatalf("skills=%+v err=%v", skills, err)
	}
	servers, total, err := svc.ListMcpServers(ctx, 1, 10)
	if err != nil || total != 1 || len(servers) != 1 || servers[0].Name != "M" || servers[0].Score != 3 {
		t.Fatalf("servers=%+v err=%v", servers, err)
	}
}
