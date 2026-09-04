# 内容日榜重构设计

> 日期：2026-09-05  
> 状态：已批准（待实现）  
> 范围：文章、Skill、MCP Server 的日榜；替换现有基于累计统计、时间衰减和定时重算的排行榜实现。

---

## 1. 目标

平台热门内容改为基于**登录用户当天的有效互动**实时排序。

支持三类内容：

```text
article
skill
mcp_server
```

每类内容仅实现日榜：

```text
当天的热门内容；每天零点切换至新的 Redis Key。
```

目标如下：

1. 有效行为发生后立即更新对应内容的当日日榜，不做定时重算。
2. 游客访问仍增加内容普通浏览量，但不参与日榜积分。
3. 同一登录用户对同一内容当天的浏览只贡献一次热度。
4. 点赞、收藏和评论实时影响排名；撤销或删除时扣回对应积分。
5. Redis 故障不能影响阅读、点赞、收藏和评论等主业务。
6. 删除旧的综合热度、候选集和定时重算方案，不保留两套 Redis 排行榜。

## 2. 非目标

本次不实现：

- 周榜、月榜、历史榜或个性化推荐；
- 游客热度计分；
- 消息队列、事件总线、异步 Worker 或重试任务；
- Redis 标题/名称缓存与 singleflight（Top 5 本地快照缓存除外，见 5.1）；
- 复杂反作弊、限流或质量权重；
- 按标签、作者、关键词过滤后的独立热榜。

## 3. 热度规则

规则对文章、Skill、MCP Server 一致。积分写入被操作的内容 ID，而不是写入用户 ID。

| 用户行为 | 前提 | 当日日榜积分 |
|---|---|---:|
| 浏览内容 | 登录用户（非作者本人）当天首次浏览同一内容 | `+1` |
| 点赞内容 | 点赞事务提交成功 | `+2` |
| 取消点赞 | 取消点赞事务提交成功 | `-2` |
| 收藏内容 | 收藏事务提交成功 | `+2` |
| 取消收藏 | 取消收藏事务提交成功 | `-2` |
| 创建评论 | 评论事务提交成功 | `+3` |
| 删除评论 | 评论删除事务提交成功 | `-3` |
| 内容不可公开 | 隐藏、删除、撤回、下架或转草稿成功 | 从当日日榜移除 |

文章发布、Skill/MCP Server 审核发布不直接获得热度积分；内容必须依靠真实登录用户互动进入日榜。

补充规则：

1. 撤销类操作（取消点赞、取消收藏、删除评论）的扣分一律记入**当日**日榜，不回溯原行为发生日；跨天撤销造成的误差是已接受的近似行为。
2. 任一内容当日积分被扣至 `<= 0` 时，立即从当日日榜移除（`ZREM`），不出现零分或负分成员。
3. 作者本人浏览自己的内容不计分，也不创建浏览去重 Key。
4. 管理员隐藏评论视为该评论不可公开：隐藏时对其所属内容执行 `AddScore(-3)`；取消隐藏不恢复积分。

## 4. Redis 数据设计

日榜的"一天"固定按 **UTC+8（北京时间）** 计算，每天零点切换至新 Key。Go 代码使用 `time.FixedZone("CST", 8*3600)` 构造日界，不依赖系统时区、环境变量或 tzdata。日榜 Key 中的 `YYYYMMDD` 与浏览去重 Key 的过期秒数必须由同一时区派生。

### 4.1 内容日榜 ZSet

```text
content_hot_rank:daily:{contentType}:YYYYMMDD
```

示例：

```text
content_hot_rank:daily:article:20260905
content_hot_rank:daily:skill:20260905
content_hot_rank:daily:mcp_server:20260905
```

类型为 ZSet：

```text
member = contentId
score  = 内容当天累计热度积分
```

写入示例：

```text
ZINCRBY content_hot_rank:daily:article:20260905 2 123
```

读取前五名：

```text
ZREVRANGE content_hot_rank:daily:article:20260905 0 4 WITHSCORES
```

首次写入后设置 TTL 为 31 天。业务接口只读取当天 Key；保留旧日榜仅用于短期排查和未来扩展，不参与当前榜单计算。

### 4.2 登录用户浏览去重 Key

```text
content_hot_view:YYYYMMDD:{contentType}:{contentId}:{userId}
```

示例：

```text
content_hot_view:20260905:article:123:88
content_hot_view:20260905:skill:45:88
content_hot_view:20260905:mcp_server:67:88
```

类型为 String，写入方式：

```text
SET <key> 1 NX EX <到当天结束的秒数（按 UTC+8 日界计算）>
```

只有 `SET NX` 成功时，才执行：

```text
ZINCRBY 当日日榜 +1 contentId
```

`userId == 0` 代表游客：不创建去重 Key，不增加日榜积分。作者本人（`userId == 内容作者 ID`）同样不创建去重 Key、不计分。

### 4.3 不新增标题或名称缓存

排行榜先从 ZSet 取得当前页内容 ID 与积分，再批量查询 MySQL 的最小展示字段：

| 类型 | 查询字段 |
|---|---|
| 文章 | `id, title` |
| Skill | `id, name` |
| MCP Server | `id, name` |

查询必须过滤不可公开内容，并按 ZSet 返回顺序恢复结果。发现已不存在或不可见成员时，应 best-effort 从当前日榜移除该成员。

## 5. 服务设计

新建通用内容榜单服务：

```text
internal/service/content_ranking.go
```

新增内容类型：

```go
type RankedContentType string

const (
    RankedContentArticle   RankedContentType = "article"
    RankedContentSkill     RankedContentType = "skill"
    RankedContentMcpServer RankedContentType = "mcp_server"
)
```

建议服务接口：

```go
func (s *ContentRankingService) AddScore(
    ctx context.Context,
    contentType RankedContentType,
    contentID uint,
    delta int64,
) error

func (s *ContentRankingService) RecordView(
    ctx context.Context,
    contentType RankedContentType,
    contentID, userID uint,
) (counted bool, err error)

func (s *ContentRankingService) Remove(
    ctx context.Context,
    contentType RankedContentType,
    contentID uint,
) error

func (s *ContentRankingService) ListArticles(
    ctx context.Context,
    page, pageSize int,
) ([]HotArticleBrief, int64, error)

func (s *ContentRankingService) ListSkills(
    ctx context.Context,
    page, pageSize int,
) ([]HotSkillBrief, int64, error)

func (s *ContentRankingService) ListMcpServers(
    ctx context.Context,
    page, pageSize int,
) ([]HotMcpServerBrief, int64, error)
```

返回 DTO 必须包含内容 ID、展示名称和积分：

```go
type HotArticleBrief struct {
    ID    uint  `json:"id"`
    Title string `json:"title"`
    Score int64  `json:"score"`
}

type HotSkillBrief struct {
    ID    uint  `json:"id"`
    Name  string `json:"name"`
    Score int64  `json:"score"`
}

type HotMcpServerBrief struct {
    ID    uint  `json:"id"`
    Name  string `json:"name"`
    Score int64  `json:"score"`
}
```

`AddScore` 对当前日榜执行一次 `ZINCRBY`（其返回值为扣减后的新分数），并在 Key 首次写入时设置 TTL；若新分数 `<= 0`，紧接着执行一次 `ZREM` 移除该成员。`Remove` 从当前日榜执行一次 `ZREM`。所有日榜 Key 的日期均按 UTC+8 日界派生。

### 5.1 Top 5 本地快照缓存

`ContentRankingService` 为三类内容各维护一份 `page=1&page_size=5` 的进程内快照（互斥锁 + 过期时间戳），TTL 固定 **3 秒**，代码常量，不进配置。侧栏在站内每个页面都会请求 Top 5，快照把回源频率压到每类内容每 3 秒至多一次。

1. 快照未过期时直接返回，不访问 Redis 与 MySQL；
2. 过期后回源一次并整体替换快照；**降级空结果同样写回快照**，避免 Redis 故障期间每个页面视图都打 Redis 并刷错误日志，恢复后最迟一个 TTL 自愈；
3. `page > 1` 或 `page_size != 5` 的请求不经过快照，直接回源；
4. 不引入 singleflight：回源只是 ZREVRANGE 加一次 `<= 5` 个主键的 IN 查询，毫秒级，偶发并发回源无压力；
5. 不设失效 hook：撤榜与积分变化最迟 3 秒对侧栏可见，与日榜粒度匹配。

## 6. 业务接入

### 6.1 浏览

每个详情读取服务保留原有 MySQL 浏览量增加行为，在现有 `trackView && 内容已发布` 的浏览埋点处追加：

```text
若 userId > 0 且 userId != 作者 ID：
  RecordView(contentType, contentId, userId)
```

覆盖：

```text
ArticleService.Get
SkillService.Get
McpServerService.Get
```

### 6.2 点赞和收藏

对应数据库事务成功后，同步调用 `AddScore`：

```text
点赞成功 / 取消点赞：+2 / -2
收藏成功 / 取消收藏：+2 / -2
```

覆盖：

```text
ArticleService
SkillService
McpServerService
```

### 6.3 评论

文章评论：

```text
CommentService 创建 / 删除评论：+3 / -3
```

资源评论：

```text
ResourceCommentService 创建 / 删除评论：
  resourceType=skill      → Skill +3 / -3
  resourceType=mcp_server → MCP Server +3 / -3
```

管理员隐藏评论：

```text
AdminService.HideComment / HideResourceComment：
  对评论所属内容执行 AddScore(-3)；取消隐藏不恢复
```

### 6.4 内容不可公开

以下操作成功后同步调用 `Remove`：

```text
文章：删除、转草稿、管理员隐藏
Skill：删除、撤回、下架、管理员隐藏
MCP Server：删除、撤回、下架、管理员隐藏
举报处理：AdminService.HideContent / HideContentTx（通用隐藏入口）对三类内容同样撤榜
```

恢复已隐藏内容时不恢复旧积分；后续的新互动会使其重新进入当日日榜。

### 6.5 Redis 失败策略

所有榜单 Redis 调用均为附属操作：

```text
MySQL 主业务事务成功
  ↓
Redis 调用失败
  ↓
记录错误日志（带 content_type、content_id、action）
  ↓
仍向用户返回主业务成功
```

不引入 goroutine、Channel、消息队列或重试任务。

### 6.6 读路径 Redis 失败

三个日榜接口在 Redis 读取失败时降级返回空结果：

```text
articles / skills / mcp_servers: []，total: 0，HTTP 200
```

同时记录错误日志。前端侧栏按空数据渲染，不进入错误态。

### 6.7 已接受的近似行为

- 每天零点后到产生新互动之前，日榜为空，不做昨日榜回退；
- Redis 故障期间丢失的积分不做补偿或对账，日榜为近似值；
- 跨天撤销的扣分误差见第 3 节补充规则。

## 7. HTTP 接口与前端

新增/保留以下接口：

```text
GET /api/v1/articles/ranking?page=1&page_size=5
GET /api/v1/skills/ranking?page=1&page_size=5
GET /api/v1/mcp-servers/ranking?page=1&page_size=5
```

接口统一返回当日日榜；不提供 `period` 参数。

首页侧栏和资源侧栏默认请求各自类型的 Top 5 日榜。前端可以显示排名和名称；第一版不要求显示积分。

## 8. 必须删除的旧排行榜

旧排行榜被整体替换，必须删除下列功能和依赖：

1. 基于累计 `views`、`likes_count`、`favorites_count`、`comments_count` 和发布时间计算的 `CalculateHotScore`。
2. 旧 ZSet Key：

   ```text
   rank:articles:hot
   rank:skills:hot
   rank:mcp_servers:hot
   rank:articles:hot:title:{id}
   ```

3. 候选阈值、候选集扫描、ZSet 裁剪、标题预热、MySQL 热度降级、本地缓存和 singleflight。
4. `internal/scheduler/ranking.go` 中的两分钟定时重算任务，以及 `cmd/server/main.go` 对它的启动和停止代码。
5. `internal/platform/config.go` 中的旧排行榜配置：

   ```text
   ranking.gravity
   ranking.max_candidates
   ranking.min_likes
   ranking.min_favorites
   ranking.min_comments
   ranking.min_views
   ranking.local_cache_ttl
   ```

6. 旧 `RankingService`、`RankingReader`、文章/Skill/MCP Server 的旧排行榜调用和重算测试。

现有内容列表的 MySQL `sort=hot` 不属于新的时间窗口排行榜。本次保留该列表排序，直到后续单独定义“带筛选条件的热门排序”语义；首页与资源侧栏一律使用新的 ranking 接口。

## 9. 文件改动清单

| 文件 | 改动 |
|---|---|
| `internal/service/content_ranking.go` | 新建通用内容日榜服务 |
| `internal/service/content_ranking_test.go` | 新建通用内容日榜服务测试 |
| `internal/service/article.go` | 接入文章浏览、点赞、收藏、删除和状态变化 |
| `internal/service/skill.go` | 接入 Skill 浏览、点赞、收藏、删除和状态变化 |
| `internal/service/mcp_server.go` | 接入 MCP Server 浏览、点赞、收藏、删除和状态变化 |
| `internal/service/comment.go` | 接入文章评论积分 |
| `internal/service/resource_comment.go` | 接入 Skill/MCP Server 评论积分 |
| `internal/service/admin.go` | 隐藏三类内容（含 HideContent/HideContentTx 举报路径）撤榜；隐藏评论 -3 |
| `internal/app/services.go` | 创建并注入 `ContentRankingService` |
| `internal/handler/ranking.go` | 提供文章、Skill、MCP Server 日榜接口 |
| `internal/handler/ranking_test.go` | 覆盖三个日榜接口 |
| `frontend/src/types/index.ts` | 新增三个日榜 brief 类型与 `score` |
| `frontend/src/api/article.ts` | 文章日榜 API 类型 |
| `frontend/src/api/skill.ts` | Skill 日榜 API |
| `frontend/src/api/mcp_server.ts` | MCP Server 日榜 API |
| `frontend/src/components/Sidebar.vue` | 文章侧栏改为调用日榜 |
| `frontend/src/components/ResourceSidebar.vue` | Skill/MCP Server 侧栏改为调用日榜 |
| `internal/service/ranking.go` | 删除旧实现 |
| `internal/service/ranking_test.go` | 删除旧测试 |
| `internal/scheduler/ranking.go` | 删除 |
| `cmd/server/main.go` | 删除旧 Scheduler 生命周期代码；注册两个资源日榜接口 |
| `internal/platform/config.go` | 删除旧排行榜配置 |
| `internal/mcpserver/dependencies.go` | 删除 `RankingReader` |
| `internal/mcpserver/tool_content.go` | 删除对旧排行榜服务的依赖；`sort=hot` 保持调用内容列表服务 |

## 10. 实施步骤与测试

### 步骤 1：通用日榜服务

先创建失败测试，验证三个内容类型均满足：

1. 正负积分会正确累加和扣减；
2. 日榜 Key 格式正确；
3. 日榜 TTL 为 31 天；
4. 分数按降序返回；
5. 登录用户当天首次浏览计分，重复浏览不计分；
6. 游客浏览不计分；
7. 移除内容后，日榜不再包含该内容；
8. 扣分至当日积分 `<= 0` 后，成员从当日日榜移除；
9. 日榜 Key 的日期按 UTC+8 日界生成；
10. Top 5 请求在 TTL 内重复调用时返回快照且不再访问 Redis；
11. 降级空结果同样按 TTL 缓存，过期后重新回源；
12. `page > 1` 或 `page_size != 5` 的请求不经过快照。

确认测试失败后，再实现最小服务代码；实现后确认测试通过。

### 步骤 2：文章接入

为文章浏览、点赞、取消点赞、收藏、取消收藏、评论创建/删除、删除、转草稿和隐藏分别编写失败测试，再接入通用日榜服务。

### 步骤 3：Skill 与 MCP Server 接入

为 Skill 和 MCP Server 的浏览、点赞、收藏、资源评论、撤回/下架/隐藏分别编写失败测试，再接入通用日榜服务。

### 步骤 4：接口与前端

实现三类内容日榜接口，更新前端类型和三个侧栏的数据源。

### 步骤 5：移除旧排行榜

删除第 8 节列出的旧代码、配置、Scheduler、测试和 MCP 依赖。使用 `rg` 确认旧符号、旧 Key 和旧配置没有残留。

## 11. 验收标准

- 文章、Skill、MCP Server 都有独立的当日日榜；
- 每类榜单只统计对应内容的登录用户有效互动；
- 游客不影响任何日榜；
- 同一登录用户当天浏览同一内容只贡献 1 分；
- 点赞、收藏、评论同步更新日榜；
- 取消和删除操作同步扣回积分；
- 隐藏、删除、撤回、下架或转草稿的内容立即从当日日榜撤出；
- Redis 故障不影响内容主业务；
- Redis 读取失败时日榜接口返回空列表与 total 0；
- 日界按 UTC+8 零点切换；
- 当日积分 `<= 0` 的成员不出现在日榜；
- 作者本人浏览不计分；
- 举报处理隐藏的内容立即撤榜；
- 管理员隐藏评论扣回 3 分；
- Top 5 请求在快照 TTL 内不触发新的 Redis/MySQL 查询，其余分页参数直接回源；
- 不存在周榜 Key、周榜聚合逻辑或 `period` 参数；
- 不存在旧热度公式、候选集、定时重算、旧方案的本地缓存、标题字典和旧 Redis 排行 Key；
- `go test ./...`、`go build ./...`、`npm run typecheck` 与 `npm run build` 通过。

## 12. 已接受的取舍

- 每天零点后、新互动发生前日榜为空，不回退昨日榜；
- Redis 故障期间丢失的积分不做补偿，日榜为近似值；
- 跨天撤销的扣分记入当日，存在小额误差；
- Top 5 快照最多滞后 3 秒；
- 上一阶段上线的标题字典、本地缓存与 singleflight 优化（2026-09-04 spec）随旧方案一并删除。
