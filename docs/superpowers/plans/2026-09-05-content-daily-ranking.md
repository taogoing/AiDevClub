# 内容日榜重构 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 用基于登录用户当日互动的实时日榜（article / skill / mcp_server 三类内容共用一套服务）替换现有"累计统计 + 时间衰减 + 2 分钟定时重算"热榜，并删除全部旧排行榜代码。

**架构：** 新建 `ContentRankingService`：互动事务提交成功后同步 `ZINCRBY` 当日 ZSet（UTC+8 日界，Key TTL 31 天，扣减后 ≤0 立即 `ZREM`），登录用户浏览用 `SETNX` 去重 Key 计 +1（游客与作者本人不计）；读路径 `ZREVRANGE` + 主键 IN 查询最小展示字段，`page=1&page_size=5` 走 3 秒进程内快照；Redis 故障时写路径记日志不阻断主业务、读路径降级空列表 + total 0；内容不可公开（删除/转草稿/撤回/下架/隐藏/举报隐藏）时 `ZREM` 撤榜。任务 6 删除旧 `RankingService`、scheduler、旧配置与 MCP `RankingReader`。

**技术栈：** Go + Gin + GORM + go-redis v9、MySQL 8、Redis、Vue 3 + TypeScript + Vite

**规格：** `docs/superpowers/specs/2026-09-05-content-daily-ranking-design.md`（已批准）

**测试环境前置：** `docker compose up -d`（MySQL 宿主 3306、Redis 宿主 16379）；单测命令 `go test ./internal/<pkg> -run <TestName> -v`。执行期间每个任务 commit 到本地 master；**全部任务完成并全量验证后**才 `git push origin master`（push 即触发 CI/CD 部署生产，不要中途 push）。

**通用约束（全计划适用）：**

1. 所有日榜 Redis 调用一律同步、best-effort：错误只记 `slog.Error` 日志（带 content_type / content_id / action / delta），**绝不向主业务返回失败**。
2. 所有新增 hook 都必须带 `s.contentRanking != nil` 判空（既有测试以 `nil` 构造服务，不能 panic）。
3. 所有 `List*` 返回的切片必须非 nil（JSON 需渲染成 `[]` 而不是 `null`）。
4. Go 方法不能声明自有类型参数：Top 5 快照的读写用包级泛型函数 `freshTop` / `storeTop`，不能写成 `func (s *ContentRankingService) freshTop[T any](...)`。
5. 既有测试向 `NewSkillService` / `NewMcpServerService` 末位传 `nil`（原 `*RankingService`）：参数类型换成 `*ContentRankingService` 后 `nil` 依旧合法，无需改既有测试。

---

## 文件结构

| 文件 | 动作 | 职责 |
|---|---|---|
| `internal/service/content_ranking.go` | 新建 | ContentRankingService：日界/Key 派生、AddScore/RecordView/Remove、三个 List*、Top 5 快照、读降级 |
| `internal/service/content_ranking_test.go` | 新建 | 服务级 12 项测试 + 共享 seed/env 辅助函数（供任务 2~4 测试复用） |
| `internal/service/dto.go` | 修改 | `HotArticleBrief` 增加 `Score`；新增 `HotSkillBrief`、`HotMcpServerBrief` |
| `internal/repo/article.go`、`skill.go`、`mcp_server.go` | 修改 | 各加一个按 ID 批量查最小展示字段的方法 |
| `internal/service/article_daily_ranking_test.go` | 新建 | 文章/文章评论接入测试 |
| `internal/service/skill_daily_ranking_test.go` | 新建 | Skill 接入测试 |
| `internal/service/mcp_server_daily_ranking_test.go` | 新建 | MCP Server 接入测试 |
| `internal/service/admin_daily_ranking_test.go` | 新建 | 管理端隐藏/举报/隐藏评论测试 |
| `internal/service/article.go`、`comment.go` | 修改 | 浏览计分、点赞/收藏 ±2、评论 ±3、删除/转草稿撤榜 |
| `internal/service/skill.go`、`mcp_server.go`、`resource_comment.go` | 修改 | 同上 + 撤回/下架撤榜 + 删除 List 的旧排行榜委托 |
| `internal/service/admin.go`、`report.go` | 修改 | 隐藏撤榜、举报处理撤榜、隐藏评论 -3 |
| `internal/app/services.go` | 修改 | 构建/注入 ContentRankingService（任务 6 再删 Ranking） |
| `internal/handler/ranking.go` | 重写 | 三个日榜端点 |
| `internal/handler/ranking_test.go` | 新建 | 三个端点测试 |
| `cmd/server/main.go` | 修改 | 注册两个新路由（任务 5）；删 scheduler 与 MCP Ranking 依赖（任务 6） |
| `frontend/src/types/index.ts`、`api/article.ts`、`api/skill.ts`、`api/mcpServer.ts`、`components/ResourceSidebar.vue` | 修改 | 类型/API/侧栏数据源（`Sidebar.vue` 无需改动） |
| `internal/service/ranking.go`、`ranking_test.go`、`internal/scheduler/ranking.go` | 删除 | 旧实现整体移除 |
| `internal/platform/config.go` | 修改 | 删除 7 个旧 ranking 配置 |
| `internal/mcpserver/dependencies.go`、`tool_content.go` | 修改 | 删 `RankingReader`；hot 并入 browseListed |

任务依赖严格线性：1 → 2 → 3 → 4 → 5 → 6，每个任务结束时 `go build ./...` 必须通过（新服务与旧 RankingService 并存到任务 6 才删）。

---

## 任务 1：ContentRankingService 通用日榜服务

**文件：**
- 创建：`internal/service/content_ranking.go`
- 创建：`internal/service/content_ranking_test.go`
- 修改：`internal/service/dto.go:58-61`（HotArticleBrief 加 Score，其后追加两个新 DTO）
- 修改：`internal/repo/article.go`（FindByWithContext 之后加 ListTitlesByIDs）
- 修改：`internal/repo/skill.go`、`internal/repo/mcp_server.go`（同位置加 ListNamesByIDs）

- [ ] **步骤 1：编写失败的测试**

先加 DTO 与 repo 方法（编译前提，属于实现的一部分但先写测试文件引用它们会直接编译失败，符合"先红"）。

`internal/service/content_ranking_test.go` 全文：

```go
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
	sv := &model.McpServer{AuthorID: authorID, Name: name, Status: status, Hidden: hidden}
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
```

- [ ] **步骤 2：运行测试验证失败**

前置：`docker compose up -d` 且 `docker ps` 显示 MySQL/Redis 健康。

运行：`go test ./internal/service -run TestContent -v`
预期：编译失败，报 `undefined: ContentRankingService`、`undefined: NewContentRankingService`、`undefined: RankedContentArticle`、`undefined: cstZone`、`svc.now`（未知字段）等。

- [ ] **步骤 3：编写实现**

3a. `internal/service/dto.go`——把现有 `HotArticleBrief`（58-61 行）替换为以下三个定义：

```go
type HotArticleBrief struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Score int64  `json:"score"`
}

type HotSkillBrief struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Score int64  `json:"score"`
}

type HotMcpServerBrief struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Score int64  `json:"score"`
}
```

3b. `internal/repo/article.go`——在 `FindByIDWithContext` 之后追加：

```go
// ListTitlesByIDs 批量查询已发布且未隐藏文章的最小展示字段（日榜用）。
func (r *ArticleRepo) ListTitlesByIDs(ctx context.Context, ids []uint) ([]model.Article, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.Article
	err := r.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "title").
		Where("id IN ?", ids).
		Where("status = ?", model.ArticleStatusPublished).
		Where("hidden = ?", false).
		Find(&list).Error
	return list, err
}
```

`internal/repo/skill.go` 与 `internal/repo/mcp_server.go` 在 `FindByIDWithContext` 之后各追加（仅模型与字段不同）：

```go
// ListNamesByIDs 批量查询已发布且未隐藏 Skill 的最小展示字段（日榜用）。
func (r *SkillRepo) ListNamesByIDs(ctx context.Context, ids []uint) ([]model.Skill, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.Skill
	err := r.db.WithContext(ctx).Model(&model.Skill{}).
		Select("id", "name").
		Where("id IN ?", ids).
		Where("status = ?", model.ResourceStatusPublished).
		Where("hidden = ?", false).
		Find(&list).Error
	return list, err
}
```

```go
// ListNamesByIDs 批量查询已发布且未隐藏 MCP Server 的最小展示字段（日榜用）。
func (r *McpServerRepo) ListNamesByIDs(ctx context.Context, ids []uint) ([]model.McpServer, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.McpServer
	err := r.db.WithContext(ctx).Model(&model.McpServer{}).
		Select("id", "name").
		Where("id IN ?", ids).
		Where("status = ?", model.ResourceStatusPublished).
		Where("hidden = ?", false).
		Find(&list).Error
	return list, err
}
```

3c. `internal/service/content_ranking.go` 全文：

```go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

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
	dailyTopCacheTTL = 3 * time.Second
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

	mu         sync.Mutex
	articleTop topSnapshot[HotArticleBrief]
	skillTop   topSnapshot[HotSkillBrief]
	mcpTop     topSnapshot[HotMcpServerBrief]
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
		now: time.Now,
	}
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
	newScore, err := s.rdb.ZIncrBy(ctx, key, delta, memberID(contentID)).Result()
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

func (s *ContentRankingService) ListArticles(ctx context.Context, page, pageSize int) ([]HotArticleBrief, int64, error) {
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		items, total, ok := freshTop(s.now, &s.articleTop)
		s.mu.Unlock()
		if ok {
			return items, total, nil
		}
	}
	p, err := s.readPage(ctx, RankedContentArticle, page, pageSize)
	if err != nil {
		s.logRankErr("list", RankedContentArticle, 0, 0, err)
		items, total := []HotArticleBrief{}, int64(0)
		if s.isTop5(page, pageSize) {
			s.mu.Lock()
			storeTop(s.now, &s.articleTop, items, total)
			s.mu.Unlock()
		}
		return items, total, nil
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
			_ = s.Remove(ctx, RankedContentArticle, id) // best-effort 清理不可见成员
			pruned++
			continue
		}
		items = append(items, HotArticleBrief{ID: a.ID, Title: a.Title, Score: p.scores[i]})
	}
	total := p.total - int64(pruned) // 被清理的成员不再计入 total
	if s.isTop5(page, pageSize) {
		s.mu.Lock()
		storeTop(s.now, &s.articleTop, items, total)
		s.mu.Unlock()
	}
	return items, total, nil
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
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/service -run TestContent -v`
预期：全部 PASS（约 10 个测试）。

运行：`go build ./...`
预期：编译通过（dto.go 的 Score 字段对现有 `GetArticleHotBriefs` 无影响）。

- [ ] **步骤 5：Commit**

```bash
git add internal/service/content_ranking.go internal/service/content_ranking_test.go internal/service/dto.go internal/repo/article.go internal/repo/skill.go internal/repo/mcp_server.go
git commit -m "feat: add content daily ranking service"
```

---

## 任务 2：文章与文章评论接入日榜

**文件：**
- 修改：`internal/service/article.go`（struct/ctor/detail/ToggleLike/ToggleFavorite/Delete/Update）
- 修改：`internal/service/comment.go`（struct/ctor/Create/Delete）
- 修改：`internal/app/services.go`（构建 contentRanking 并注入 Article/Comment；**保留** Ranking 给 Skill/MCP，任务 3 再切）
- 测试：`internal/service/article_daily_ranking_test.go`（新建）

- [ ] **步骤 1：编写失败的测试**

`internal/service/article_daily_ranking_test.go` 全文：

```go
package service

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newArticleDailyEnv(t *testing.T) (*ArticleService, *CommentService, *ContentRankingService, *gorm.DB, *model.User, *model.User) {
	t.Helper()
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	users := repo.NewUserRepo(db)
	author := seedUser(t, db, "author@t.com")
	viewer := seedUser(t, db, "viewer@t.com")
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	rankSvc := NewContentRankingService(rdb, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	articleSvc := NewArticleService(repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), cfg, notifSvc, rankSvc)
	commentSvc := NewCommentService(repo.NewCommentRepo(db), repo.NewArticleRepo(db), repo.NewInteractionRepo(db), users, notifSvc, rankSvc)
	return articleSvc, commentSvc, rankSvc, db, author, viewer
}

func seedPublishedArticle(t *testing.T, db *gorm.DB, authorID uint, title string) *model.Article {
	t.Helper()
	return seedArticle(t, db, authorID, title, model.ArticleStatusPublished, false)
}

func TestArticleDailyViewScoresOnce(t *testing.T) {
	svc, _, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	if _, err := svc.Get(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 1 || len(items) != 1 || items[0].ID != a.ID || items[0].Score != 1 {
		t.Fatalf("items=%+v total=%d", items, total)
	}
}

func TestArticleDailyAuthorAndGuestNotScored(t *testing.T) {
	svc, _, rankSvc, db, author, _ := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	if _, err := svc.Get(ctx, author.ID, a.ID); err != nil { // 作者本人
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, 0, a.ID); err != nil { // 游客
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("author/guest must not score: items=%+v total=%d", items, total)
	}
}

func TestArticleDailyToggleLikeAndFavorite(t *testing.T) {
	svc, _, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	if _, _, err := svc.ToggleLike(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListArticles(ctx, 1, 10)
	if items[0].Score != 2 {
		t.Fatalf("after like score=%d want 2", items[0].Score)
	}
	if _, _, err := svc.ToggleLike(ctx, viewer.ID, a.ID); err != nil { // 取消点赞
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("like+unlike must empty the board: items=%+v total=%d", items, total)
	}

	if _, _, err := svc.ToggleFavorite(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, _, _ = rankSvc.ListArticles(ctx, 1, 10)
	if items[0].Score != 2 {
		t.Fatalf("after favorite score=%d want 2", items[0].Score)
	}
	if _, _, err := svc.ToggleFavorite(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ = rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("favorite+unfavorite must empty the board: items=%+v total=%d", items, total)
	}
}

func TestArticleDailyCommentCreateDelete(t *testing.T) {
	_, commentSvc, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")

	c, err := commentSvc.Create(ctx, viewer.ID, a.ID, "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListArticles(ctx, 1, 10)
	if items[0].Score != 3 {
		t.Fatalf("after comment score=%d want 3", items[0].Score)
	}
	if err := commentSvc.Delete(ctx, viewer.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("comment delete must empty: items=%+v total=%d", items, total)
	}
}

func TestArticleDailyDeleteAndDraftRemove(t *testing.T) {
	svc, _, rankSvc, db, author, viewer := newArticleDailyEnv(t)
	ctx := context.Background()
	a := seedPublishedArticle(t, db, author.ID, "A")
	_, _ = svc.Get(ctx, viewer.ID, a.ID)
	if _, _, err := svc.ToggleLike(ctx, viewer.ID, a.ID); err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(ctx, author.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("delete must remove from board: items=%+v total=%d", items, total)
	}

	b := seedPublishedArticle(t, db, author.ID, "B")
	_, _ = svc.Get(ctx, viewer.ID, b.ID)
	if _, err := svc.Update(ctx, author.ID, b.ID, CreateArticleInput{
		Title: "B", Summary: "", Content: "c", Status: model.ArticleStatusDraft,
	}); err != nil {
		t.Fatal(err)
	}
	items, total, _ = rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("publish->draft must remove from board: items=%+v total=%d", items, total)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/service -run TestArticleDaily -v`
预期：编译失败——`NewArticleService` / `NewCommentService` 第 6 个参数仍是 `*RankingService`，传 `*ContentRankingService` 类型不匹配。

- [ ] **步骤 3：编写实现**

3a. `internal/service/article.go`：

struct 字段与 ctor（24-35 行）：

```go
type ArticleService struct {
	articles       *repo.ArticleRepo
	tags           *repo.TagRepo
	inter          *repo.InteractionRepo
	cfg            *platform.Config
	notifSvc       *NotificationService
	contentRanking *ContentRankingService
}

func NewArticleService(articles *repo.ArticleRepo, tags *repo.TagRepo, inter *repo.InteractionRepo, cfg *platform.Config, notifSvc *NotificationService, contentRanking *ContentRankingService) *ArticleService {
	return &ArticleService{articles: articles, tags: tags, inter: inter, cfg: cfg, notifSvc: notifSvc, contentRanking: contentRanking}
}
```

`Update`（139 行起）——在 `if a.Status != in.Status` 之前插入一行，在事务返回后追加撤榜（改 `return a, err` 为显式判断）：

```go
	wasPublished := a.Status == model.ArticleStatusPublished
```

```go
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		// ……原有事务体保持不变……
	})
	if err == nil && wasPublished && a.Status == model.ArticleStatusDraft && s.contentRanking != nil {
		_ = s.contentRanking.Remove(ctx, RankedContentArticle, articleID)
	}
	return a, err
```

`Delete`（207 行起）——`return s.articles.DB().Transaction(...)` 改为捕获 err 并追加撤榜：

```go
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		// ……原有事务体保持不变……
	})
	if err == nil && s.contentRanking != nil {
		_ = s.contentRanking.Remove(ctx, RankedContentArticle, articleID)
	}
	return err
```

`detail`（396-399 行）——浏览埋点追加计分：

```go
	if trackView && a.Status == model.ArticleStatusPublished {
		_ = s.articles.IncrViews(ctx, articleID)
		a.Views++
		if userID > 0 && userID != a.AuthorID && s.contentRanking != nil {
			_, _ = s.contentRanking.RecordView(ctx, RankedContentArticle, articleID, userID)
		}
	}
```

`ToggleLike`（319-327 行）——把 `if s.rankingSvc != nil { go func() {...} }` 块替换为（**保留**其后的点赞通知 goroutine）：

```go
	if err == nil && s.contentRanking != nil {
		delta := int64(2)
		if !liked {
			delta = -2
		}
		_ = s.contentRanking.AddScore(ctx, RankedContentArticle, articleID, delta)
	}
```

`ToggleFavorite`（360-368 行）——同样替换为：

```go
	if err == nil && s.contentRanking != nil {
		delta := int64(2)
		if !favorited {
			delta = -2
		}
		_ = s.contentRanking.AddScore(ctx, RankedContentArticle, articleID, delta)
	}
```

3b. `internal/service/comment.go`——struct 加字段、ctor 加参数（末位）：

```go
type CommentService struct {
	comments       *repo.CommentRepo
	articles       *repo.ArticleRepo
	inter          *repo.InteractionRepo
	users          *repo.UserRepo
	notifSvc       *NotificationService
	contentRanking *ContentRankingService
}

func NewCommentService(comments *repo.CommentRepo, articles *repo.ArticleRepo, inter *repo.InteractionRepo, users *repo.UserRepo, notifSvc *NotificationService, contentRanking *ContentRankingService) *CommentService {
	return &CommentService{comments: comments, articles: articles, inter: inter, users: users, notifSvc: notifSvc, contentRanking: contentRanking}
}
```

`Create`（56-58 行的 `if err == nil` 块）追加同步计分：

```go
	if err == nil {
		if s.contentRanking != nil {
			_ = s.contentRanking.AddScore(ctx, RankedContentArticle, articleID, 3)
		}
		go s.sendCommentNotification(context.Background(), a.AuthorID, userID, articleID, replyToID, content)
	}
```

`Delete`（170-175 行）改为：

```go
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Delete(tx, commentID); err != nil {
			return err
		}
		return s.articles.IncrCount(tx, c.ArticleID, "comments_count", -1)
	})
	if err == nil && s.contentRanking != nil {
		_ = s.contentRanking.AddScore(ctx, RankedContentArticle, c.ArticleID, -3)
	}
	return err
```

3c. `internal/app/services.go`——`Services` struct 加字段 `ContentRanking *service.ContentRankingService`（放在 `Ranking` 之后）；`NewServices` 中在 `ranking := ...` 之后加构建，并改 Article/Comment 两行（Skill/MCP/ResourceComment 仍用 `ranking`，任务 3 切换）：

```go
	ranking := service.NewRankingService(infra.Redis, articles, skills, mcpServers, cfg)
	contentRanking := service.NewContentRankingService(infra.Redis, articles, skills, mcpServers)

	articleService := service.NewArticleService(articles, tags, interactions, cfg, notifications, contentRanking)
	commentService := service.NewCommentService(comments, articles, interactions, users, notifications, contentRanking)
```

返回字面量中加 `ContentRanking: contentRanking,`。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/service -run "TestArticleDaily|TestContent" -v`
预期：全部 PASS。

运行：`go build ./...`
预期：通过（`GetArticleHotBriefs` 等旧代码未动）。

- [ ] **步骤 5：Commit**

```bash
git add internal/service/article.go internal/service/comment.go internal/app/services.go internal/service/article_daily_ranking_test.go
git commit -m "feat: hook article views, likes, favorites and comments into daily ranking"
```

---

## 任务 3：Skill 与 MCP Server 接入日榜（含资源评论、移除列表委托）

**文件：**
- 修改：`internal/service/skill.go`（struct/ctor/detail/ToggleLike/ToggleFavorite/Delete/Withdraw/Archive/List）
- 修改：`internal/service/mcp_server.go`（同上镜像）
- 修改：`internal/service/resource_comment.go`（struct/ctor/Create/Delete）
- 修改：`internal/app/services.go`（Skill/MCP/ResourceComment 切到 contentRanking）
- 测试：`internal/service/skill_daily_ranking_test.go`、`internal/service/mcp_server_daily_ranking_test.go`（新建）

- [ ] **步骤 1：编写失败的测试**

`internal/service/skill_daily_ranking_test.go` 全文：

```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

func newSkillDailyEnv(t *testing.T) (*SkillService, *ResourceCommentService, *ContentRankingService) {
	t.Helper()
	db, rankSvc, users := newResourceDailyDeps(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	skillSvc := NewSkillService(repo.NewSkillRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), cfg, notifSvc, rankSvc)
	resCommentSvc := NewResourceCommentService(repo.NewResourceCommentRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db), repo.NewInteractionRepo(db), users, notifSvc, rankSvc)
	return skillSvc, resCommentSvc, rankSvc
}

func TestSkillDailyViewAuthorGuest(t *testing.T) {
	svc, _, rankSvc := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	author, viewer := env.author, env.viewer

	sk := seedSkill(t, env.db, author.ID, "S", model.ResourceStatusPublished, false)
	if _, err := svc.Get(ctx, viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, author.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, 0, sk.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 1 || len(items) != 1 || items[0].Score != 1 {
		t.Fatalf("items=%+v total=%d", items, total)
	}
}

func TestSkillDailyToggleLikeFavoriteAndResourceComment(t *testing.T) {
	svc, resCommentSvc, rankSvc := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)

	if _, _, err := svc.ToggleLike(ctx, env.viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ToggleFavorite(ctx, env.viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	c, err := resCommentSvc.Create(ctx, env.viewer.ID, "skill", sk.ID, "hi", nil)
	if err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListSkills(ctx, 1, 10)
	if items[0].Score != 7 { // 2 + 2 + 3
		t.Fatalf("score=%d want 7", items[0].Score)
	}

	if _, _, err := svc.ToggleLike(ctx, env.viewer.ID, sk.ID); err != nil { // 取消点赞 -> 5
		t.Fatal(err)
	}
	if err := resCommentSvc.Delete(ctx, env.viewer.ID, c.ID); err != nil { // 删评论 -> 2
		t.Fatal(err)
	}
	items, _, _ = rankSvc.ListSkills(ctx, 1, 10)
	if items[0].Score != 2 {
		t.Fatalf("score=%d want 2", items[0].Score)
	}
}

func TestSkillDailyArchiveRemoves(t *testing.T) {
	svc, _, rankSvc := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)
	if _, err := svc.Get(ctx, env.viewer.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Archive(ctx, env.author.ID, sk.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("archive must remove: items=%+v total=%d", items, total)
	}
}

func TestSkillListSortHotWithoutDailyScore(t *testing.T) {
	// 移除旧委托后：即使日榜为空，sort=hot 也必须走 MySQL 返回已发布内容
	svc, _, _ := newSkillDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	seedSkill(t, env.db, env.author.ID, "S1", model.ResourceStatusPublished, false)
	seedSkill(t, env.db, env.author.ID, "S2", model.ResourceStatusPublished, false)

	out, err := svc.List(ctx, SkillListQuery{Page: 1, PageSize: 10, Sort: "hot"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.List) != 2 {
		t.Fatalf("sort=hot list len=%d want 2", len(out.List))
	}
}
```

`internal/service/mcp_server_daily_ranking_test.go` 全文：

```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

func newMcpDailyEnv(t *testing.T) (*McpServerService, *ResourceCommentService, *ContentRankingService) {
	t.Helper()
	db, rankSvc, users := newResourceDailyDeps(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50}
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	mcpSvc := NewMcpServerService(repo.NewMcpServerRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), cfg, notifSvc, rankSvc)
	resCommentSvc := NewResourceCommentService(repo.NewResourceCommentRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db), repo.NewInteractionRepo(db), users, notifSvc, rankSvc)
	return mcpSvc, resCommentSvc, rankSvc
}

func TestMcpDailyViewLikeCommentArchive(t *testing.T) {
	svc, resCommentSvc, rankSvc := newMcpDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sv := seedMcpServer(t, env.db, env.author.ID, "M", model.ResourceStatusPublished, false)

	if _, err := svc.Get(ctx, env.viewer.ID, sv.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ToggleLike(ctx, env.viewer.ID, sv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := resCommentSvc.Create(ctx, env.viewer.ID, "mcp_server", sv.ID, "hi", nil); err != nil {
		t.Fatal(err)
	}
	items, _, _ := rankSvc.ListMcpServers(ctx, 1, 10)
	if items[0].Score != 6 { // 1 + 2 + 3
		t.Fatalf("score=%d want 6", items[0].Score)
	}

	if _, err := svc.Archive(ctx, env.author.ID, sv.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListMcpServers(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("archive must remove: items=%+v total=%d", items, total)
	}
}

func TestMcpListSortHotWithoutDailyScore(t *testing.T) {
	svc, _, _ := newMcpDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	seedMcpServer(t, env.db, env.author.ID, "M1", model.ResourceStatusPublished, false)
	seedMcpServer(t, env.db, env.author.ID, "M2", model.ResourceStatusPublished, false)

	out, err := svc.List(ctx, McpServerListQuery{Page: 1, PageSize: 10, Sort: "hot"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.List) != 2 {
		t.Fatalf("sort=hot list len=%d want 2", len(out.List))
	}
}
```

两个文件都依赖一个共享 env，追加到 `internal/service/content_ranking_test.go` 末尾：

```go
// --- 任务 3/4 共享 env：resourceDailyEnv 供 Skill/MCP/Admin 接入测试使用 ---

type resourceEnv struct {
	db     *gorm.DB
	author *model.User
	viewer *model.User
}

var resourceDailyEnv resourceEnv

func newResourceDailyDeps(t *testing.T) (*gorm.DB, *ContentRankingService, *repo.UserRepo) {
	t.Helper()
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	users := repo.NewUserRepo(db)
	resourceDailyEnv = resourceEnv{
		db:     db,
		author: seedUser(t, db, "res-author@t.com"),
		viewer: seedUser(t, db, "res-viewer@t.com"),
	}
	rankSvc := NewContentRankingService(rdb, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	return db, rankSvc, users
}
```

注意：包级 `resourceDailyEnv` 在同包测试串行执行下安全（每个测试自己重建）。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/service -run "TestSkillDaily|TestMcpDaily" -v`
预期：编译失败——`NewSkillService` / `NewMcpServerService` / `NewResourceCommentService` 参数类型不匹配。

- [ ] **步骤 3：编写实现**

3a. `internal/service/skill.go`：

struct/ctor（21-32 行）把 `rankingSvc *RankingService` 全部换成 `contentRanking *ContentRankingService`（字段名、参数名、赋值同步替换）。

`Delete`（191-205 行）改为捕获 err + 撤榜（同任务 2 文章 Delete 模式）：

```go
	err = s.skills.DB().Transaction(func(tx *gorm.DB) error {
		// ……原有事务体保持不变……
	})
	if err == nil && s.contentRanking != nil {
		_ = s.contentRanking.Remove(ctx, RankedContentSkill, skillID)
	}
	return err
```

`Withdraw`（228-244 行）在 `s.skills.Update(nil, sk)` 成功后、`return sk, nil` 前加：

```go
	if s.contentRanking != nil {
		_ = s.contentRanking.Remove(ctx, RankedContentSkill, skillID)
	}
```

`Archive`（246-262 行）同样在 `Update` 成功后加：

```go
	if s.contentRanking != nil {
		_ = s.contentRanking.Remove(ctx, RankedContentSkill, skillID)
	}
```

`detail`（306-309 行）浏览埋点改为：

```go
	if trackView && sk.Status == model.ResourceStatusPublished {
		_ = s.skills.IncrViews(ctx, skillID)
		sk.Views++
		if userID > 0 && userID != sk.AuthorID && s.contentRanking != nil {
			_, _ = s.contentRanking.RecordView(ctx, RankedContentSkill, skillID, userID)
		}
	}
```

`ToggleLike`（353-361 行）把 `if s.rankingSvc != nil { go func() {...} }` 替换为（保留点赞通知 goroutine）：

```go
	if err == nil && s.contentRanking != nil {
		delta := int64(2)
		if !liked {
			delta = -2
		}
		_ = s.contentRanking.AddScore(ctx, RankedContentSkill, skillID, delta)
	}
```

`ToggleFavorite`（394-402 行）同样替换（`favorited` 对应 -2）。

`List`（423-434 行）——**删除整个委托块**：

```go
	if q.Sort == "hot" && q.TagID == nil && q.AuthorID == nil && q.Keyword == "" && s.rankingSvc != nil {
		skills, total, err := s.rankingSvc.ListSkillHot(ctx, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		return &SkillListResult{
			List:     skills,
			Total:    total,
			Page:     q.Page,
			PageSize: q.PageSize,
		}, nil
	}
```

删除后 `sort=hot` 自然落入下方 `repo.SkillQuery` 走 MySQL 公式排序。

3b. `internal/service/mcp_server.go`——镜像同样 8 处改动（`RankedContentMcpServer`、`s.servers`、`serverID`）：
- struct/ctor 换 `contentRanking`；
- `Delete`（235-249 行）捕获 err + Remove；
- `Withdraw`（276-292 行）、`Archive`（294-310 行）Update 成功后 Remove；
- `detail`（354-356 行）浏览埋点加 RecordView（作者排除）；
- `ToggleLike`（407-415 行）、`ToggleFavorite`（448-456 行）替换为同步 AddScore ±2；
- `List`（477-488 行）删除委托块。

3c. `internal/service/resource_comment.go`——struct 加 `contentRanking *ContentRankingService` 字段，ctor 末位加参数并赋值。`Create`（100-103 行）改为：

```go
	if err == nil {
		if s.contentRanking != nil {
			if ct, ok := rankedResourceType(resourceType); ok {
				_ = s.contentRanking.AddScore(ctx, ct, resourceID, 3)
			}
		}
		go s.sendResCommentNotification(context.Background(), userID, resourceType, resourceID, replyToID, content)
	}
```

`Delete`（217-222 行）改为：

```go
	err = s.getDB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Delete(tx, commentID); err != nil {
			return err
		}
		return s.incrCommentsCount(c.ResourceType, c.ResourceID, -1)
	})
	if err == nil && s.contentRanking != nil {
		if ct, ok := rankedResourceType(c.ResourceType); ok {
			_ = s.contentRanking.AddScore(ctx, ct, c.ResourceID, -3)
		}
	}
	return err
```

3d. `internal/app/services.go`——三行切换：

```go
	skillService := service.NewSkillService(skills, tags, interactions, cfg, notifications, contentRanking)
	mcpServerService := service.NewMcpServerService(mcpServers, tags, interactions, cfg, notifications, contentRanking)
	resourceCommentService := service.NewResourceCommentService(resourceComments, skills, mcpServers, interactions, users, notifications, contentRanking)
```

3e. 既有测试构造点补参（`NewResourceCommentService` 追加了第 7 个参数 `contentRanking`，既有调用只有 6 参会编译失败）：

- `internal/service/resource_comment_test.go:32` 的 `NewResourceCommentService(...)` 调用末尾补 `, nil`
- `internal/handler/resource_comment_test.go:29` 的 `service.NewResourceCommentService(...)` 调用末尾补 `, nil`

（`go build ./...` / `go vet ./...` 会暴露全部漏改点；如有其他构造点一并补 `nil`。）

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/service -run "TestSkillDaily|TestMcpDaily|TestSkillListSortHot|TestMcpListSortHot" -v`
预期：全部 PASS。

运行：`go test ./internal/service ./internal/handler -v`
预期：既有测试全部 PASS（它们向 ctor 传 `nil`，类型变更后仍编译；委托删除后 sort=hot 测试本就走 repo 路径）。

运行：`go build ./...`
预期：通过。

- [ ] **步骤 5：Commit**

```bash
git add internal/service/skill.go internal/service/mcp_server.go internal/service/resource_comment.go internal/app/services.go internal/service/skill_daily_ranking_test.go internal/service/mcp_server_daily_ranking_test.go internal/service/content_ranking_test.go
git commit -m "feat: hook skill and mcp server interactions into daily ranking"
```

---

## 任务 4：管理端与举报撤榜、隐藏评论扣分

**文件：**
- 修改：`internal/service/admin.go`（struct/ctor/HideArticle/HideSkill/HideMcpServer/HideContent/HideComment/HideResourceComment）
- 修改：`internal/service/report.go`（struct/ctor/Resolve）
- 修改：`internal/app/services.go`（Admin/Reports 注入 contentRanking）
- 测试：`internal/service/admin_daily_ranking_test.go`（新建）

**设计说明（与 spec 的字面差异及理由）：** `HideContentTx` 在 `ReportService.Resolve` 的事务**内部**调用——Redis 撤榜不能放进未提交事务（回滚会误撤）。因此撤榜 hook 放在 `Resolve` 事务**提交成功之后**（report.go:154 的 err 判断后），语义等价于 spec 的"举报路径撤榜"且时序正确。`HideContent`（无事务版本）直接内联撤榜。

- [ ] **步骤 1：编写失败的测试**

`internal/service/admin_daily_ranking_test.go` 全文：

```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

func newAdminDailyEnv(t *testing.T) (*AdminService, *ReportService, *ContentRankingService) {
	t.Helper()
	db, rankSvc, users := newResourceDailyDeps(t)
	notifSvc := NewNotificationService(repo.NewNotificationRepo(db), users)
	adminLogs := NewAdminLogService(repo.NewAdminLogRepo(db), users)
	admin := NewAdminService(
		users, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db),
		repo.NewCommentRepo(db), repo.NewResourceCommentRepo(db), repo.NewReportRepo(db),
		repo.NewAnnouncementRepo(db), adminLogs, notifSvc, rankSvc,
	)
	reports := NewReportService(
		repo.NewReportRepo(db), repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db),
		repo.NewCommentRepo(db), repo.NewResourceCommentRepo(db), admin, adminLogs, notifSvc, rankSvc,
	)
	return admin, reports, rankSvc
}

func TestAdminHideArticleRemovesFromDaily(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	a := seedArticle(t, env.db, env.author.ID, "A", model.ArticleStatusPublished, false)
	_ = rankSvc.AddScore(ctx, RankedContentArticle, a.ID, 4)

	if err := admin.HideArticle(ctx, env.author.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("hide must remove: items=%+v total=%d", items, total)
	}
}

func TestAdminHideContentGenericRemoves(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)
	_ = rankSvc.AddScore(ctx, RankedContentSkill, sk.ID, 4)

	if err := admin.HideContent("skill", sk.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("generic hide must remove: items=%+v total=%d", items, total)
	}
}

func TestAdminHideCommentDeducts(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	a := seedArticle(t, env.db, env.author.ID, "A", model.ArticleStatusPublished, false)
	c := &model.Comment{ArticleID: a.ID, AuthorID: env.viewer.ID, Content: "x"}
	if err := env.db.Create(c).Error; err != nil {
		t.Fatal(err)
	}
	_ = rankSvc.AddScore(ctx, RankedContentArticle, a.ID, 3)

	if err := admin.HideComment(ctx, env.author.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("hide comment must deduct to removal: items=%+v total=%d", items, total)
	}
}

func TestAdminHideResourceCommentDeducts(t *testing.T) {
	admin, _, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	sk := seedSkill(t, env.db, env.author.ID, "S", model.ResourceStatusPublished, false)
	rc := &model.ResourceComment{ResourceType: "skill", ResourceID: sk.ID, AuthorID: env.viewer.ID, Content: "x"}
	if err := env.db.Create(rc).Error; err != nil {
		t.Fatal(err)
	}
	_ = rankSvc.AddScore(ctx, RankedContentSkill, sk.ID, 3)

	if err := admin.HideResourceComment(ctx, env.author.ID, rc.ID); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListSkills(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("hide resource comment must deduct: items=%+v total=%d", items, total)
	}
}

func TestReportResolveHideRemoves(t *testing.T) {
	_, reports, rankSvc := newAdminDailyEnv(t)
	ctx := context.Background()
	env := resourceDailyEnv
	a := seedArticle(t, env.db, env.author.ID, "A", model.ArticleStatusPublished, false)
	_ = rankSvc.AddScore(ctx, RankedContentArticle, a.ID, 4)
	r := &model.Report{ReporterID: env.viewer.ID, TargetType: "article", TargetID: a.ID, Status: model.ReportStatusPending}
	if err := env.db.Create(r).Error; err != nil {
		t.Fatal(err)
	}

	if err := reports.Resolve(ctx, env.author.ID, r.ID, "hide", "违规"); err != nil {
		t.Fatal(err)
	}
	items, total, _ := rankSvc.ListArticles(ctx, 1, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("report hide must remove: items=%+v total=%d", items, total)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/service -run "TestAdmin|TestReportResolve" -v`
预期：编译失败——`NewAdminService` / `NewReportService` 没有 contentRanking 参数。

- [ ] **步骤 3：编写实现**

3a. `internal/service/admin.go`——struct 加字段 `contentRanking *ContentRankingService`（放 `notifSvc` 之后），ctor 末位加参数 `contentRanking *ContentRankingService` 并赋值。

`HideArticle`（715-720 行）改为（HideSkill/HideMcpServer 同模式，换类型与 ID）：

```go
func (s *AdminService) HideArticle(ctx context.Context, adminID, articleID uint) error {
	if err := s.articles.DB().Model(&model.Article{}).Where("id = ?", articleID).Update("hidden", true).Error; err != nil {
		return err
	}
	if s.contentRanking != nil {
		_ = s.contentRanking.Remove(ctx, RankedContentArticle, articleID)
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "article", articleID, nil)
}
```

`HideContent`（639-659 行）三个内容分支各在 Update 成功后加撤榜（该函数无 ctx，用 `context.Background()`；当前无调用方，仅保持入口一致）：

```go
func (s *AdminService) HideContent(targetType string, targetID uint) error {
	switch targetType {
	case "article":
		if err := s.articles.DB().Model(&model.Article{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		if s.contentRanking != nil {
			_ = s.contentRanking.Remove(context.Background(), RankedContentArticle, targetID)
		}
	case "skill":
		if err := s.skills.DB().Model(&model.Skill{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		if s.contentRanking != nil {
			_ = s.contentRanking.Remove(context.Background(), RankedContentSkill, targetID)
		}
	case "mcp_server":
		if err := s.mcpServers.DB().Model(&model.McpServer{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		if s.contentRanking != nil {
			_ = s.contentRanking.Remove(context.Background(), RankedContentMcpServer, targetID)
		}
	case "comment":
		if err := s.comments.DB().Model(&model.Comment{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		return s.comments.DB().Model(&model.Comment{}).Where("parent_id = ?", targetID).Update("hidden", true).Error
	case "resource_comment":
		if err := s.resourceComments.DB().Model(&model.ResourceComment{}).Where("id = ?", targetID).Update("hidden", true).Error; err != nil {
			return err
		}
		return s.resourceComments.DB().Model(&model.ResourceComment{}).Where("parent_id = ?", targetID).Update("hidden", true).Error
	}
	return platform.NewBizError(http.StatusBadRequest, platform.CodeParamError, "不支持的目标类型")
}
```

（comment / resource_comment 分支的隐藏扣分走 `HideComment` / `HideResourceComment`，`HideContent` 的 comment 分支不重复扣分。）

`HideComment`（344-350 行）改为：

```go
func (s *AdminService) HideComment(ctx context.Context, adminID, id uint) error {
	if err := s.comments.DB().Model(&model.Comment{}).Where("id = ?", id).Update("hidden", true).Error; err != nil {
		return err
	}
	_ = s.comments.DB().Model(&model.Comment{}).Where("parent_id = ?", id).Update("hidden", true).Error
	if s.contentRanking != nil {
		if c, err := s.comments.FindByID(nil, id); err == nil {
			_ = s.contentRanking.AddScore(ctx, RankedContentArticle, c.ArticleID, -3)
		}
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "comment", id, nil)
}
```

`HideResourceComment`（427-433 行）改为：

```go
func (s *AdminService) HideResourceComment(ctx context.Context, adminID, id uint) error {
	if err := s.resourceComments.DB().Model(&model.ResourceComment{}).Where("id = ?", id).Update("hidden", true).Error; err != nil {
		return err
	}
	_ = s.resourceComments.DB().Model(&model.ResourceComment{}).Where("parent_id = ?", id).Update("hidden", true).Error
	if s.contentRanking != nil {
		if c, err := s.resourceComments.FindByID(nil, id); err == nil {
			if ct, ok := rankedResourceType(c.ResourceType); ok {
				_ = s.contentRanking.AddScore(ctx, ct, c.ResourceID, -3)
			}
		}
	}
	return s.adminLogSvc.Log(ctx, adminID, model.AdminLogActionHideContent, "resource_comment", id, nil)
}
```

说明：级联隐藏的子回复不逐条扣分（与"取消隐藏不恢复"一致的近似处理）。

3b. `internal/service/report.go`——struct 加 `contentRanking *ContentRankingService`，ctor 末位加参数并赋值。`Resolve` 在事务成功判断后（154-156 行 `if err != nil { return err }` 之后）加：

```go
	if action == "hide" && s.contentRanking != nil {
		switch adminTargetType {
		case "article":
			_ = s.contentRanking.Remove(ctx, RankedContentArticle, report.TargetID)
		case "skill":
			_ = s.contentRanking.Remove(ctx, RankedContentSkill, report.TargetID)
		case "mcp_server":
			_ = s.contentRanking.Remove(ctx, RankedContentMcpServer, report.TargetID)
		}
	}
```

3c. `internal/app/services.go`——Admin 与 Reports 的 ctor 调用末位追加 `contentRanking`：

```go
	admin := service.NewAdminService(
		users, articles, skills, mcpServers, comments, resourceComments, reportsRepo,
		announcementRepo, adminLogs, notifications, contentRanking,
	)
	reports := service.NewReportService(
		reportsRepo, articles, skills, mcpServers, comments, resourceComments, admin, adminLogs, notifications, contentRanking,
	)
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/service -run "TestAdmin|TestReport" -v`
预期：新增 5 个测试 PASS，既有 admin/report 相关测试（如有）PASS。

运行：`go build ./...`
预期：通过。

- [ ] **步骤 5：Commit**

```bash
git add internal/service/admin.go internal/service/report.go internal/app/services.go internal/service/admin_daily_ranking_test.go
git commit -m "feat: remove hidden content from daily ranking via admin and report flows"
```

---

## 任务 5：日榜 HTTP 接口与前端侧栏

**文件：**
- 重写：`internal/handler/ranking.go`
- 创建：`internal/handler/ranking_test.go`
- 修改：`cmd/server/main.go:174-175`（换 handler + 注册两条新路由）
- 修改：`frontend/src/types/index.ts:51-54`、`frontend/src/api/skill.ts`、`frontend/src/api/mcpServer.ts`
- 确认不改：`frontend/src/components/Sidebar.vue`、`frontend/src/api/article.ts`（类型自动带上 score）

- [ ] **步骤 1：编写失败的测试**

`internal/handler/ranking_test.go` 全文：

```go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func rankingRouter(t *testing.T) (*gin.Engine, *service.ContentRankingService, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	svc := service.NewContentRankingService(rdb, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	h := NewRankingHandler(svc)
	r := gin.New()
	r.GET("/api/v1/articles/ranking", h.GetArticleRanking)
	r.GET("/api/v1/skills/ranking", h.GetSkillRanking)
	r.GET("/api/v1/mcp-servers/ranking", h.GetMcpServerRanking)
	return r, svc, db
}

func getRankingJSON(t *testing.T, r *gin.Engine, path string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", path, w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out["data"].(map[string]any)
}

func TestRankingEndpointsEmptyBoard(t *testing.T) {
	r, _, _ := rankingRouter(t)
	for _, tc := range []struct{ path, field string }{
		{"/api/v1/articles/ranking?page=1&page_size=5", "articles"},
		{"/api/v1/skills/ranking?page=1&page_size=5", "skills"},
		{"/api/v1/mcp-servers/ranking?page=1&page_size=5", "mcp_servers"},
	} {
		data := getRankingJSON(t, r, tc.path)
		if data[tc.field] == nil {
			t.Fatalf("%s: %s must be [] not null, data=%v", tc.path, tc.field, data)
		}
		if arr, ok := data[tc.field].([]any); !ok || len(arr) != 0 {
			t.Fatalf("%s: %s want empty array, got %v", tc.path, tc.field, data[tc.field])
		}
		if data["total"].(float64) != 0 {
			t.Fatalf("%s: total=%v want 0", tc.path, data["total"])
		}
	}
}

func TestRankingEndpointReturnsScoredOrder(t *testing.T) {
	r, svc, db := rankingRouter(t)
	ctx := context.Background()
	a1 := &model.Article{AuthorID: 1, Title: "one", Status: model.ArticleStatusPublished}
	a2 := &model.Article{AuthorID: 1, Title: "two", Status: model.ArticleStatusPublished}
	if err := db.Create(a1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(a2).Error; err != nil {
		t.Fatal(err)
	}
	_ = svc.AddScore(ctx, service.RankedContentArticle, a1.ID, 1)
	_ = svc.AddScore(ctx, service.RankedContentArticle, a2.ID, 5)

	data := getRankingJSON(t, r, "/api/v1/articles/ranking?page=1&page_size=5")
	arr := data["articles"].([]any)
	if len(arr) != 2 {
		t.Fatalf("articles len=%d", len(arr))
	}
	first := arr[0].(map[string]any)
	if first["id"].(float64) != float64(a2.ID) || first["title"] != "two" || first["score"].(float64) != 5 {
		t.Fatalf("first=%v want two/5", first)
	}
}
```

再追加两个测试（同文件）：

```go
func TestRankingEndpointPageClamp(t *testing.T) {
	r, _, _ := rankingRouter(t)
	data := getRankingJSON(t, r, "/api/v1/articles/ranking?page=0&page_size=500")
	if data["page"].(float64) != 1 || data["page_size"].(float64) != 50 {
		t.Fatalf("clamp failed: page=%v page_size=%v", data["page"], data["page_size"])
	}
}

func TestRankingEndpointRedisDownDegradesEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	dead := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", DialTimeout: 100 * time.Millisecond, ReadTimeout: 100 * time.Millisecond,
	})
	svc := service.NewContentRankingService(dead, repo.NewArticleRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db))
	h := NewRankingHandler(svc)
	r := gin.New()
	r.GET("/api/v1/articles/ranking", h.GetArticleRanking)
	data := getRankingJSON(t, r, "/api/v1/articles/ranking?page=1&page_size=5")
	if arr, ok := data["articles"].([]any); !ok || len(arr) != 0 {
		t.Fatalf("degraded articles=%v", data["articles"])
	}
	if data["total"].(float64) != 0 {
		t.Fatalf("degraded total=%v", data["total"])
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/handler -run TestRankingEndpoint -v`
预期：编译失败——`NewRankingHandler` 现在要 `*service.RankingService`，且 `GetSkillRanking` / `GetMcpServerRanking` 未定义。

- [ ] **步骤 3：编写实现**

`internal/handler/ranking.go` 重写为：

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type RankingHandler struct {
	rankingSvc *service.ContentRankingService
}

func NewRankingHandler(rankingSvc *service.ContentRankingService) *RankingHandler {
	return &RankingHandler{rankingSvc: rankingSvc}
}

func rankingPageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "5"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 5
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize
}

func (h *RankingHandler) GetArticleRanking(c *gin.Context) {
	page, pageSize := rankingPageParams(c)
	briefs, total, err := h.rankingSvc.ListArticles(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"articles": briefs, "total": total, "page": page, "page_size": pageSize,
	})
}

func (h *RankingHandler) GetSkillRanking(c *gin.Context) {
	page, pageSize := rankingPageParams(c)
	briefs, total, err := h.rankingSvc.ListSkills(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"skills": briefs, "total": total, "page": page, "page_size": pageSize,
	})
}

func (h *RankingHandler) GetMcpServerRanking(c *gin.Context) {
	page, pageSize := rankingPageParams(c)
	briefs, total, err := h.rankingSvc.ListMcpServers(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"mcp_servers": briefs, "total": total, "page": page, "page_size": pageSize,
	})
}
```

`cmd/server/main.go:174-175` 替换为：

```go
	rankingH := handler.NewRankingHandler(services.ContentRanking)
	r.GET("/api/v1/articles/ranking", rankingH.GetArticleRanking)
	r.GET("/api/v1/skills/ranking", rankingH.GetSkillRanking)
	r.GET("/api/v1/mcp-servers/ranking", rankingH.GetMcpServerRanking)
```

（scheduler 与 MCP deps 的 `services.Ranking` 引用**暂不动**，任务 6 处理。）

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/handler -run TestRankingEndpoint -v`
预期：4 个测试全部 PASS。

运行：`go build ./...`
预期：通过。

- [ ] **步骤 5：前端类型与侧栏**

`frontend/src/types/index.ts`——`HotArticleBrief`（51-54 行）加 `score: number;`，其后追加：

```ts
export interface HotSkillBrief {
  id: number
  name: string
  score: number
}

export interface HotMcpServerBrief {
  id: number
  name: string
  score: number
}
```

`frontend/src/api/skill.ts`——import 类型加 `HotSkillBrief`，文件末尾追加：

```ts
export function getSkillRanking(page = 1, pageSize = 5) {
  return http.get<ApiResponse<{ skills: HotSkillBrief[]; total: number }>>('/api/v1/skills/ranking', {
    params: { page, page_size: pageSize },
  })
}
```

`frontend/src/api/mcpServer.ts`——import 类型加 `HotMcpServerBrief`，文件末尾追加：

```ts
export function getMcpServerRanking(page = 1, pageSize = 5) {
  return http.get<ApiResponse<{ mcp_servers: HotMcpServerBrief[]; total: number }>>('/api/v1/mcp-servers/ranking', {
    params: { page, page_size: pageSize },
  })
}
```

`frontend/src/components/ResourceSidebar.vue`——script 部分替换为：

```ts
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { TrendCharts } from '@element-plus/icons-vue'
import { getSkillRanking } from '@/api/skill'
import { getMcpServerRanking } from '@/api/mcpServer'
import type { HotSkillBrief, HotMcpServerBrief } from '@/types'

const props = defineProps<{
  type: 'skill' | 'mcp'
}>()

const router = useRouter()
const hotResources = ref<(HotSkillBrief | HotMcpServerBrief)[]>([])

onMounted(async () => {
  await fetchData()
})

watch(() => props.type, () => {
  fetchData()
})

async function fetchData() {
  try {
    const getRanking = props.type === 'skill' ? getSkillRanking : getMcpServerRanking
    const res = await getRanking(1, 5)
    hotResources.value =
      (props.type === 'skill' ? res.data.data.skills : res.data.data.mcp_servers) ?? []
  } catch {
    hotResources.value = []
  }
}

function handleResourceClick(id: number) {
  if (props.type === 'skill') {
    router.push({ name: 'skill-detail', params: { id } })
  } else {
    router.push({ name: 'mcp-detail', params: { id } })
  }
}
```

（template 不变——`item.id` / `item.name` 两个字段新类型都有。`Sidebar.vue` 与 `api/article.ts` 无需改动。）

- [ ] **步骤 6：前端验证**

运行：`cd frontend; npm run typecheck; npm run build`
预期：两者 exit 0。

- [ ] **步骤 7：Commit**

```bash
git add internal/handler/ranking.go internal/handler/ranking_test.go cmd/server/main.go frontend/src/types/index.ts frontend/src/api/skill.ts frontend/src/api/mcpServer.ts frontend/src/components/ResourceSidebar.vue
git commit -m "feat: expose daily ranking endpoints and switch sidebars to them"
```

---

## 任务 6：删除旧排行榜与旧配置，全量验证

**文件：**
- 删除：`internal/service/ranking.go`、`internal/service/ranking_test.go`、`internal/scheduler/ranking.go`（整包删除，目录清空后移除）
- 修改：`cmd/server/main.go`（删 scheduler 生命周期与 import；MCP deps 删 Ranking）
- 修改：`internal/mcpserver/dependencies.go`（删 RankingReader 与 Ranking 字段）
- 修改：`internal/mcpserver/tool_content.go`（删 browseHot；hot 并入 browseListed）
- 修改：`internal/platform/config.go`（删 7 项旧配置）
- 修改：`internal/app/services.go`（删 Ranking 构建与字段）

- [ ] **步骤 1：删除旧文件**

```bash
git rm internal/service/ranking.go internal/service/ranking_test.go internal/scheduler/ranking.go
```

（`internal/scheduler/` 只含这一个文件，目录随之消失。）

- [ ] **步骤 2：清理引用**

2a. `cmd/server/main.go`：
- import 区删除 `"aidevclub/internal/scheduler"`；
- 删除 179-181 行（`rankingScheduler := ...` / `.Start(ctx)` / `defer .Stop()`）；
- `startMCPServer` 里 `PublicDependencies` 字面量删除 `Ranking: services.Ranking,` 行；
- 顶部若 `"time"` import 仅剩 scheduler 使用则一并删除（检查 `time.Minute` 等其他用法再决定）。

2b. `internal/mcpserver/dependencies.go`：删除 `RankingReader` 接口（33-37 行）与 `PublicDependencies.Ranking` 字段（68 行）。

2c. `internal/mcpserver/tool_content.go`：删除整个 `browseHot` 函数（359 行起）；291-297 行的分派改为：

```go
	if err := browseListed(ctx, deps, publicBaseURL, &output); err != nil {
		return nil, browseContentOutput{}, err
	}
```

（`sort=hot` 经 `browseListed` 调用内容列表服务，落 MySQL 公式排序；`temporarilyUnavailable` 若因此无引用，顺带删除其定义。）

2d. `internal/platform/config.go`：
- struct 删除 `RankingGravity` ~ `RankingLocalCacheTTL` 七个字段（37-43 行）；
- 删除 `v.SetDefault("ranking.gravity", 1.5)` ~ `v.SetDefault("ranking.local_cache_ttl", "30s")` 七行（71-77 行）；
- 删除 `rankingGravity := ...` 与 `rankingLocalCacheTTL, err := ...` 及其 `if err != nil { return nil, err }`（95-99 行）；
- 删除 `Config{}` 字面量里七个 Ranking 赋值（142-148 行）。

2e. `internal/app/services.go`：删除 `Ranking *service.RankingService` 字段、`ranking := service.NewRankingService(...)` 行、返回字面量 `Ranking: ranking,`。保留 `ContentRanking`。

- [ ] **步骤 3：编译与残留扫描**

运行：`go build ./...`
预期：通过——若报错，按编译器指引清理漏改的引用（唯一预期会出现的是 main.go 的 scheduler import 与 MCP deps）。

运行（PowerShell 下用 `rg`）：

```bash
rg -n "RankingService|CalculateHotScore|rankKey|RecalculateArticle|RankingScheduler|GetArticleHotBriefs|ListArticleHot|ListSkillHot|ListMcpServerHot|RankingReader|singleflight" internal cmd
rg -n "ranking\.(gravity|max_candidates|min_likes|min_favorites|min_comments|min_views|local_cache_ttl)" internal cmd frontend/src
```

预期：**零命中**（`repo` 的 `case "hot"` MySQL 排序保留，不属于清理对象）。

- [ ] **步骤 4：全量测试**

前置：`docker compose up -d`。

运行：`go test ./...`
预期：全部 PASS（service 包新增 25+ 个日榜测试；旧的 ranking_test.go 已删除）。

运行：`go vet ./...`
预期：无输出。

运行：`cd frontend; npm run typecheck; npm run build`
预期：exit 0（任务 5 已改，此处为最终门禁）。

- [ ] **步骤 5：Commit**

```bash
git add -A
git commit -m "refactor: remove legacy hot ranking service, scheduler and config"
```

- [ ] **步骤 6：上线与线上验证**

```bash
git push origin master
```

push 触发 GitHub Actions（Deploy to Production + CI）。用 `gh run watch` 等两个工作流 `success`，然后：

```bash
curl -s "https://aidevclub.xyz/api/v1/articles/ranking?page=1&page_size=5"
curl -s "https://aidevclub.xyz/api/v1/skills/ranking?page=1&page_size=5"
curl -s "https://aidevclub.xyz/api/v1/mcp-servers/ranking?page=1&page_size=5"
curl -s "https://aidevclub.xyz/healthz"
```

预期：三个榜单返回 `{"code":0,...,"data":{"articles|skills|mcp_servers":[],"total":0,...}}`（上线初期日榜为空属正常——零点后要有互动才有分）；healthz ok。

生产 Redis 清理旧 Key（一次性，服务器上执行；标题字典 Key 自带 5 分钟 TTL 无需处理）：

```bash
docker compose exec redis redis-cli DEL rank:articles:hot rank:skills:hot rank:mcp_servers:hot
```

---

## 验收对照（对应 spec §11）

| 验收项 | 验证位置 |
|---|---|
| 三类内容独立当日日榜 | 任务 1 `TestContentSkillAndMcpDailyRanking` 等 |
| 只统计登录用户有效互动 / 游客不影响 | 任务 1 `TestContentRecordViewDedupAndGuest`、任务 2/3 |
| 同用户当天浏览只计 1 分 | 任务 1/2/3 view 测试 |
| 点赞/收藏/评论同步更新、撤销扣回 | 任务 2/3 toggle/comment 测试 |
| 不可公开内容立即撤榜（删除/转草稿/撤回/下架/隐藏/举报） | 任务 2/3/4 |
| 隐藏评论扣 3 分 | 任务 4 |
| Redis 故障不影响主业务 | 写路径 best-effort（全部 hook 忽略错误） |
| Redis 读失败返回空列表 + total 0 | 任务 1/5 降级测试 |
| UTC+8 日界 | 任务 1 `TestContentDailyKeyUTC8Boundary` |
| 积分 ≤0 不出现在日榜 | 任务 1 `TestContentAddScoreNonPositiveRemovesMember` |
| 作者本人浏览不计分 | 任务 2/3 author 测试 |
| Top 5 快照 TTL 内不回源 | 任务 1 快照测试 |
| 无 period 参数 / 无周榜 | handler 无该参数（任务 5） |
| 旧热度公式/候选集/重算/本地缓存/标题字典/旧 Key 全删 | 任务 6 rg 扫描零命中 |
| 四条标准命令通过 | 任务 6 步骤 4 |
