# 热榜重构计划

> 日期：2026-09-01
> 状态：待执行

---

## 一、背景与问题

当前热榜实现存在以下问题：

| 问题 | 说明 |
|------|------|
| 全量重建 ZSet | 每 2 分钟 `DEL rank:articles:hot` → 遍历全表 → 逐条 `ZADD`，数据量大时阻塞 Redis |
| 冷文章污染 | 所有文章（无论有无互动）都进入 ZSet，浪费内存 |
| 缓存失效用 KEYS | `KEYS hot:articles:*` 模式扫描，生产环境阻塞 Redis |
| 每次交互都 ZADD | 点赞/收藏触发 goroutine 做 `ZADD`，高并发下 Redis 写压力大 |
| MySQL 降级无时间衰减 | `sort=hot` 的 SQL 只有静态权重，没有时间衰减，和 Redis 排序不一致 |

---

## 二、重构方案（分阶段递进）

### Phase 1：基线方案 — 纯 MySQL 热榜

**目标**：去掉 Redis 依赖，仅用 MySQL 返回热榜数据。

#### 1.1 修改 MySQL 排序公式

在 `internal/repo/article.go` 的 `List` 方法中，将 `sort=hot` 的排序改为带时间衰减：

```sql
ORDER BY (views + 3*likes_count + 5*favorites_count + 2*comments_count + 1)
         / POW(TIMESTAMPDIFF(HOUR, published_at, NOW()) + 2, 1.5) DESC
```

- 权重：浏览 x1、点赞 x3、收藏 x5、评论 x2
- 时间衰减：`POW(小时差 + 2, 1.5)`，gravity=1.5
- 对 Skill、MCP Server 做同样修改

#### 1.2 移除 Redis ZSet 相关代码

- 删除 `RankingService` 中的 `Recalculate*HotRanking`、`Get*HotRanking`、`Update*HotScore` 方法
- 删除 `RankingScheduler`（不再需要 2 分钟全量重建）
- 删除 `ArticleService.updateHotScoreAsync`、`invalidateHotCaches` 中的 Redis ZSet 操作
- 删除 `CommentService.invalidateHotArticles` 中的 Redis 操作
- 简化 `RankingHandler`，直接调用 repo 层的 MySQL 查询

#### 1.3 清理 Redis 缓存键

- 移除 `hot:articles:*`、`hot:tags:*` 等 Redis String 缓存逻辑
- 移除 `KEYS` 模式扫描

#### 1.4 验证

- 由我（AI）审核代码改动和热榜排序逻辑的正确性
- 部署后调用 API 验证热榜返回结果符合预期

---

### Phase 2：Redis ZSet 优化 — 阈值门槛 + 候选集重算

**目标**：引入 Redis ZSet 加速热榜读取，但只让有热度的文章进入 ZSet。

#### 2.1 阈值门槛

文章必须满足以下条件之一才进入 ZSet：

```
likes_count >= 3
OR favorites_count >= 2
OR comments_count >= 2
OR views >= 50
```

阈值可通过配置调整。

#### 2.2 写入链路

用户点赞/收藏/评论时：

```
1. MySQL 更新计数（同步）
2. 重新计算该文章 hot_score
3. 判断是否达到阈值
   - 达到 → ZADD rank:articles:hot（异步 goroutine）
   - 未达到 → 不写入 ZSet
```

#### 2.3 定时任务优化（候选集重算）

每 2 分钟定时任务改为：

```
1. 从 ZSet 取出所有成员（ZRANGE，即候选集）
2. 对候选集中的每篇文章，从 MySQL 读取最新数据，重算 hot_score（含时间衰减）
3. Pipeline 批量 ZADD 更新分数
4. ZREMRANGEBYRANK 裁剪掉分数低于最低阈值的成员
5. 扫描 MySQL 中近期有新互动但不在 ZSet 中的文章，达到阈值的 ZADD 加入
```

不再 `DEL` 全量重建，只处理候选集。

#### 2.4 读取链路

```
Redis ZREVRANGE → 获取 ID 列表 → MySQL WHERE id IN 批量加载详情
```

#### 2.5 Skill / MCP Server 同样处理

#### 2.6 验证

- 单元测试：阈值过滤、候选集重算
- 手动验证：冷文章不在 ZSet 中

---

### Phase 3：本地缓存 — 抗热 key

**目标**：在应用层加本地缓存，减少 Redis 读压力。

#### 3.1 实现

在 `RankingService` 中加入 `sync.Mutex` + 缓存结构：

```go
type localCache struct {
    mu        sync.RWMutex
    data      []uint      // 文章 ID 列表
    expiresAt time.Time
    ttl       time.Duration  // 默认 2 秒
}
```

读链路变为：

```
本地缓存命中 → 直接返回
本地缓存未命中 → Redis ZREVRANGE → 写入本地缓存 → 返回
```

#### 3.2 缓存失效

- 本地缓存 TTL 很短（2~3 秒），自然过期即可
- 不需要主动失效

#### 3.3 验证

- 压测对比：开启本地缓存前后的 QPS 和延迟

---

### Phase 4：快照表 — Redis 故障兜底

**目标**：MySQL 保存热榜快照，Redis 异常时降级返回快照数据。

#### 4.1 新建快照表

```sql
CREATE TABLE rank_snapshots (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    rank_type VARCHAR(32) NOT NULL,    -- 'articles', 'skills', 'mcp_servers'
    item_id BIGINT NOT NULL,
    hot_score DOUBLE NOT NULL,
    rank_no INT NOT NULL,
    snapshot_time DATETIME NOT NULL,
    INDEX idx_type_time (rank_type, snapshot_time)
);
```

#### 4.2 定时保存快照

在定时任务（Phase 2 的候选集重算）完成后，将当前 ZSet Top 200 保存到 `rank_snapshots`。

#### 4.3 降级读取

```
本地缓存 → Redis ZSet → MySQL 快照（最近一次）
```

当 Redis 连接失败时，从 `rank_snapshots` 读取最近一次快照返回。

#### 4.4 快照清理

定时清理超过 24 小时的旧快照数据。

#### 4.5 验证

- 模拟 Redis 宕机，验证降级到快照
- 验证快照数据正确性

---

## 三、性能测试计划

### 3.1 测试环境

- 线上服务器：47.76.151.183
- 站点：https://aidevclub.xyz
- 数据库：MySQL 8 + Redis 7（Docker 部署）

### 3.2 测试数据准备

编写 Go 脚本 `cmd/bench/main.go`，功能：

1. **批量创建测试文章**
   - 创建 **200,000** 篇测试文章（标题带 `bench_` 前缀，方便清理）
   - 每篇文章内容固定为 20 字（如 "这是一篇性能测试文章，仅用于热榜压测。"）
   - 标签从数据库已有标签中随机选取 1~3 个
   - 随机分配 views（0~10000）、likes（0~500）、favorites（0~200）、comments（0~100）
   - 随机分配 published_at（最近 1 秒 ~ 30 天前）
   - 使用已有的测试用户作为 author

2. **模拟交互**
   - 对部分文章批量增加 likes/favorites/comments 计数
   - 模拟真实分布：80% 文章低互动，15% 中等互动，5% 高互动

3. **清理脚本**
   - 删除所有 `bench_` 前缀的测试文章
   - 清理关联的 article_tags、interactions 等数据

### 3.3 测试场景

| 场景 | 说明 | 指标 |
|------|------|------|
| 基线 QPS | 纯 MySQL 热榜查询，无并发 | 响应时间 P50/P95/P99 |
| 并发读 | 200/500/1000 并发查询热榜 | QPS、响应时间、MySQL CPU |
| 写入影响 | 1000 并发查询 + 500 并发点赞 | 响应时间变化 |
| Redis 优化后 | Phase 2 部署后重复上述测试 | 对比提升 |
| 本地缓存 | Phase 3 部署后重复上述测试 | 对比提升 |
| 降级测试 | 停掉 Redis，验证快照兜底 | 降级响应时间 |

### 3.4 测试工具

使用 `wrk` 进行压测：

```bash
wrk -t8 -c200 -d60s "https://aidevclub.xyz/api/v1/articles?sort=hot&page=1&page_size=20"
wrk -t8 -c500 -d60s "https://aidevclub.xyz/api/v1/articles?sort=hot&page=1&page_size=20"
wrk -t8 -c1000 -d60s "https://aidevclub.xyz/api/v1/articles?sort=hot&page=1&page_size=20"
```

并发点赞压测（需要登录态）：

```bash
wrk -t4 -c500 -d60s -s bench/like.lua "https://aidevclub.xyz/api/v1/articles/1/like"
```

### 3.5 测试数据清理

测试完成后执行清理脚本，删除所有测试数据，验证文章数恢复正常。

---

## 四、实施顺序

```
Phase 1（基线重构）
  ↓ 测试通过
Phase 2（Redis ZSet 优化）
  ↓ 测试通过
Phase 3（本地缓存）
  ↓ 测试通过
Phase 4（快照表）
  ↓ 测试通过
性能测试（全阶段对比）
  ↓ 数据记录
清理测试数据
```

---

## 五、涉及文件清单

| 文件 | 改动说明 |
|------|----------|
| `internal/repo/article.go` | 修改 hot 排序 SQL，加时间衰减 |
| `internal/repo/skill.go` | 同上 |
| `internal/repo/mcp_server.go` | 同上 |
| `internal/service/ranking.go` | 重写：阈值门槛、候选集重算、本地缓存、快照降级 |
| `internal/service/article.go` | 移除 updateHotScoreAsync、invalidateHotCaches |
| `internal/service/skill.go` | 同上 |
| `internal/service/mcp_server.go` | 同上 |
| `internal/service/comment.go` | 移除 invalidateHotArticles |
| `internal/scheduler/ranking.go` | 重写：候选集重算 + 快照保存 |
| `internal/handler/ranking.go` | 简化：直接调用 repo 或 RankingService |
| `internal/model/rank_snapshot.go` | 新增：快照表模型 |
| `internal/platform/config.go` | 新增：阈值配置、本地缓存 TTL 配置 |
| `internal/app/services.go` | 调整依赖注入 |
| `cmd/server/main.go` | 调整 scheduler 初始化 |
| `cmd/bench/main.go` | 新增：性能测试脚本 |

---

## 六、简历亮点提炼

- 将热榜从全量重建 Redis ZSet 优化为**阈值门槛 + 候选集增量更新**，Redis 内存占用降低 X%
- 引入**本地缓存（2s TTL）+ Redis ZSet + MySQL 快照**三级读取链路，热榜接口 QPS 提升 X 倍，P99 延迟降低 X%
- 设计**热榜快照表**实现 Redis 故障自动降级，保障首页热榜高可用
- 在**线上环境**使用 **20 万篇**测试数据进行压力测试（最高 1000 并发），输出完整性能对比报告
