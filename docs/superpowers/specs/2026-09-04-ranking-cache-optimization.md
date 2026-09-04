# 热榜与热门文章缓存优化设计

> 日期：2026-09-04
> 状态：已批准（待实现）
> 范围：仅文章热榜（P0）。热门文章详情缓存（原 P1）**暂缓**，见第 6 节。

---

## 1. 背景

### 1.1 当前问题

热榜接口 `GET /api/v1/articles/ranking?page=1&page_size=5` 已用 singleflight 解决缓存击穿（QPS 154 → 1,649）。但本地缓存（10s TTL）过期后，重建路径仍要查 MySQL：

```
本地缓存 miss（每 10s 一次，singleflight 合并为 1 个请求）
    ↓
Redis ZREVRANGE → top N 个 ID
    ↓
MySQL SELECT * FROM articles WHERE id IN (...)   ← 含 content 大字段 + Preload Author
    ↓
MySQL SELECT * FROM article_tags WHERE ...       ← 查标签
```

单次重建 ~500ms，且查了大量热榜展示根本不需要的字段（前端只展示排名 + 标题）。

### 1.2 优化目标

1. 热榜接口本地缓存过期后**不查 MySQL**，只走 Redis
2. 热榜响应瘦身：只返回 `{id, title}`（排名即数组顺序）
3. 保持缓存击穿免疫（singleflight，已有模式）

### 1.3 明确不做（本次范围外）

- 热门文章详情缓存（暂缓，原因见第 6 节）
- Skill / MCP 热榜同样优化（后续 follow-up）
- 重算任务的 N+1 查询优化（`FindByID` 逐个查 1000 个候选，后续可改批量 IN 查询）

---

## 2. 三层缓存架构

```
┌─ 本地缓存（进程内存）────────────────────────┐
│ key:  article-brief:{page}:{size}            │
│ 值:   [{id, title}, ...]（按排名排序）+ total │
│ TTL:  30s                                    │
└──────────────────────────────────────────────┘
        ↓ miss（singleflight 合并并发）
┌─ Redis ZSet：rank:articles:hot（已有，不动）──┐
│ member=文章ID, score=热度分                   │
│ 候选集：过阈值（赞≥3/藏≥2/评≥2/浏览≥50 任一） │
│         且分数 top 1000（重算时裁剪）         │
│ 只负责"排名顺序"，不存标题                    │
└──────────────────────────────────────────────┘
        ↓ ZREVRANGE 拿当前页 ID
┌─ Redis 标题字典（新增）──────────────────────┐
│ rank:articles:hot:title:{article_id} → 标题  │
│ TTL:  5 min（兜底；每次重算会续期）           │
│ 覆盖：ZSet 全部候选（~1000），由重算任务预热  │
└──────────────────────────────────────────────┘
        ↓ MGET；miss 的查 MySQL 补全回写（兜底）
```

关键设计认知：**标题字典与排名解耦**。它是"ID → 标题"的映射，key 里不含任何排名信息——排名怎么变，都是拿 ID 去查同一批 key，位次变化永远不会让标题缓存失效。标题缓存唯一会"脏"的场景：作者修改了标题本身。

### 2.1 各层职责与新鲜度

| 层 | 内容 | TTL/频率 | 更新方式 | 脏窗口 |
|----|------|---------|---------|--------|
| 本地缓存 | 当前页快照 `[{id,title}]` + total | 30s | 纯 TTL 过期重建 | 排名+标题 ≤30s |
| Redis ZSet | ID + 热度分（候选 ~1000） | 永久 | 点赞/收藏实时推 + 重算批量刷 | 排名实时（点赞） |
| Redis 标题字典 | `title:{id}` × ~1000 | 5 min 兜底 | 重算预热（每 2 min SET 续期）+ 读路径补全兜底 | 标题 ≤2 min |
| MySQL | 事实数据 | - | 唯一真相源 | - |

Redis 内存：1000 × ~50B ≈ 50KB，可忽略。写放大：每 2 分钟 ~1000 个 pipeline SET，可忽略。

---

## 3. 请求流程

```
GET /api/v1/articles/ranking?page=1&page_size=5
 │
 ├─ 本地缓存命中 → 直接返回 [{id,title}×5, total]
 │
 └─ miss → singleflight（200 并发只有 1 个重建）→
     ├─ ZREVRANGE rank:articles:hot 0 4 → [id1..id5]
     ├─ MGET title keys
     │   ├─ 命中 → 直接用（预热后几乎全命中）
     │   └─ miss → MySQL SELECT id,title WHERE id IN (...)
     │            AND status='published' AND hidden=false
     │            → pipeline 回写 Redis (TTL 5min)
     ├─ ZCard → total
     ├─ 组装 [{id,title}]，写本地缓存 (30s)
     └─ 返回
 │
 └─ 降级：ZREVRANGE / MGET 出错（Redis 故障）
     → 走 MySQL 热度排序查询（repo 已有 Sort:"hot"）
     → 取 id+title 组装，热榜不断服务
```

稳态下每 30s 只有 1 个请求执行 2 次 Redis RTT；MySQL 只在新文章进榜首读时被碰一次。

### 3.1 重算预热（新增，~6 行）

`RecalculateArticleHotRanking`（每 2 分钟）本来就对每个 ZSet 候选执行 `FindByID` 读最新计数来重算分数——标题已经在手上。在现有 pipeline 里顺手：

- 候选通过检查（published、未隐藏、过阈值）→ `pipe.Set title key`（含 TTL，自动续期）
- 候选被剔除（ZRem）→ 顺手 `pipe.Del title key`（保持字典与 ZSet 成员同步，孤儿 key 也靠 TTL 兜底）
- 新候选扫描分支同样补 Set

预热带来两个白赚的好处：

1. 读取路径 MGET 几乎永远全命中（只有进榜后 2 分钟窗口内的新文章会 miss 一次）
2. **改标题不需要任何失效 hook 即可自愈**：作者改完标题，最迟 2 分钟后重算从 MySQL 读到新标题重新 SET

### 3.2 写入/更新时机全景（用户动作 → 各层如何更新）

| 事件 | MySQL | ZSet | 标题字典 | 本地缓存 | 用户看到变化 |
|------|-------|------|---------|---------|-------------|
| 点赞/收藏/评论 | 同步事务更新计数 | 异步 goroutine 回读计数 → ZAdd 新分数（article.go:321 现有逻辑） | 不碰（标题没变） | 不碰，等 30s TTL | 排名 ≤~30s |
| 访问（浏览+1） | 同步 views+1 | **不实时更新**，靠 2min 重算批量重算 | 不碰 | 不碰 | 排名 ≤~2.5min |
| 文章发布 | 同步 | 重算 2min 内拉入候选 | 预热时写入 | 30s | ≤~2.5min 出现 |
| 编辑标题 | 同步 | 不涉及 | 重算 ≤2min 自愈（无失效 hook） | 30s | ≤~2min |
| 删除/下架 | 同步 | 重算 ≤2min 剔除 + Del 标题 key | 同左 | 30s | ≤~2.5min 消失 |

核心心智模型：**数据流单向**——用户动作 → MySQL（事实）→ ZSet（派生排名，点赞实时推/浏览批量重算）；缓存两层永远不被主动更新，只靠 TTL 过期后沿 ZSet → MySQL 重建。点赞/访问路径**零新增代码**。

注：下架/删除文章从热榜消失的时间从原实现的 ≤30s（每次重建都查 MySQL 带 status 过滤）放宽为 ≤~2.5min（ZSet 剔除周期），这是"零失效 hook + 不查 MySQL"的代价，用户已确认接受短期脏数据。

---

## 4. 缓存击穿/穿透/雪崩分析

### 4.1 击穿

两个可能的"过期瞬间"都已被挡住，**无需新增防御代码**：

- **本地缓存 30s 过期瞬间**（真正击穿点）：singleflight（沿用现有 `sfGroup` 模式）——200 并发只有 1 个重建，其余共享结果
- **标题字典 miss 的 MySQL 补全**：发生在 singleflight 保护区**内部**，同一瞬间最多 1 个请求碰 MySQL；且预热让 miss 本身罕见

Redis 整体故障的降级路径（MySQL hot 查询）同样在 singleflight 函数内，等待者共享降级结果。

### 4.2 穿透

不存在——热榜读路径的 ID 全部来自 ZSet 自身，用户无法注入任意 ID。

### 4.3 雪崩

标题 key 由重算任务每 2 分钟统一续期，TTL 5min > 2× 重算周期，稳态下永不过期；TTL 只是重算连续失败时的兜底。本地缓存仅 1-2 个 (page,size) 组合，谈不上雪崩。

---

## 5. 改动清单

### 5.1 后端

1. `internal/service/dto.go`：新增 `HotArticleBrief{ID uint, Title string}`
2. `internal/service/ranking.go`：
   - 新增 `GetArticleHotBriefs(ctx, page, pageSize)`：本地缓存（`article-brief:{page}:{size}`，与 MCP 用的 `article:{page}:{size}` 隔离）→ singleflight → ZREVRANGE → MGET → 补全回写 → ZCard → 组装
   - 标题读取/补全辅助函数：MGET + miss 时 `SELECT id,title` **带 `status='published' AND hidden=false` 过滤** + pipeline 回写（TTL 5min，best-effort 忽略回写错误）
   - 降级：Redis 错误 → `articleRepo.List(Sort:"hot")` 取 id/title
   - `RecalculateArticleHotRanking`：预热 title key（Set/Del，与现有 ZAdd/ZRem 同一 pipeline）
   - **`ListArticleHot` 一行不动**（MCP browse_content 依赖其完整 summary）
3. `internal/handler/ranking.go`：改调 `GetArticleHotBriefs`；响应 `articles` 字段变为 brief 数组（**breaking change**，见 5.3）；对 page/page_size 做下限与上限钳制
4. `internal/platform/config.go`：`ranking.local_cache_ttl` 默认 `10s` → `30s`（该 TTL 也被 skill/mcp 热榜本地缓存共用，同步放宽，可接受且更省）

### 5.2 测试（`internal/service/ranking_test.go`，新建）

沿用 `testutil.NewTestDB/NewTestRedis` 模式（真实 MySQL/Redis）：

1. 排序与字段：按分数排序返回 `[{id,title}]`，total 正确
2. 本地缓存命中：重建后清空 ZSet，第二次调用仍返回缓存结果
3. 标题补全回写：ZSet 有 ID、字典无 key → 从 MySQL 补标题并回写 Redis
4. Redis 标题优先：手动预置错误标题 key → 返回 Redis 值（证明走 Redis 路径）
5. 隐藏/下架过滤：补全查询排除 hidden/非 published 文章
6. 重算预热：跑 `RecalculateArticleHotRanking` 后 title key 存在；被剔除候选的 key 被删
7. Redis 故障降级：关闭 Redis 客户端 → 走 MySQL hot 排序仍返回 brief

### 5.3 前端

运行时已兼容（Sidebar.vue 只用 `article.id` / `article.title`），仅类型层面跟进：

- `frontend/src/types`：新增 `HotArticleBrief`，`getArticleRanking` 返回类型改用它
- `Sidebar.vue`：`hotArticles` ref 类型同步

---

## 6. 暂缓：热门文章详情缓存

原设计（Redis 缓存热榜文章的完整详情 JSON，TTL 60s）发现两个正确性硬伤，且依赖方（`ArticleService.Get`）行为复杂，暂缓实施：

1. **浏览量冻结**：`GET /articles/:id` 每次请求都同步 `IncrViews`（MySQL 写）。缓存命中直接返回会完全跳过这一步——最热的文章浏览量几乎停止增长。需要异步计数或 Redis 聚合回写，设计量不小
2. **per-user 字段串号**：`ArticleDetail` 含 `Liked/Favorited`（按用户查询）。整对象缓存会导致用户 B 命中用户 A 触发回写的缓存，拿到错误的点赞/收藏状态。必须只缓存共享部分（summary+content），命中后仍按用户查 interaction

待热榜优化上线观察后，若详情页 MySQL 压力仍显著再重启此设计。

---

## 7. 性能预期

| 场景 | 修复前 | 优化后 |
|------|--------|--------|
| 本地缓存命中 | ~97ms | ~97ms（不变，瓶颈不在 MySQL） |
| 本地缓存 miss（singleflight 内，1 个请求） | ~500ms（MySQL 含 content 大字段） | ~5ms（2 次 Redis RTT） |
| QPS（200 并发） | 1,649 | 上限受命中路径 ~97ms 支配（约 2,000），以压测实测为准 |

修正原稿：miss 从每 10s 一次变为每 30s 一次，且成本从 ~500ms 降到 ~5ms；但整体 QPS 上限由命中路径延迟决定，"10,000+" 的预期不成立，以压测为准。

---

## 8. 验证清单

1. `go test ./internal/service`：新增 ranking 测试全绿
2. `go build ./...` 通过
3. `cd frontend && npm run typecheck && npm run build` 通过
4. 手动/压测验证：热榜接口 200 并发下成功率 100%，本地 miss 期间无 MySQL 查询（可开慢查询日志观察）
