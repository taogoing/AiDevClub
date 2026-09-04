# 热榜与热门文章缓存优化设计

> 日期：2026-09-04
> 状态：待实现

---

## 1. 背景

### 1.1 当前问题

热榜接口 `GET /api/v1/articles/ranking?page=1&page_size=5` 在 200 并发压测下：

| 指标 | 修复前 | 修复后（singleflight） |
|------|--------|------------------------|
| QPS | 154 | 1,649 |
| 成功率 | 98.58% | 100% |
| 最大响应 | 1m（超时） | 401ms |

**已修复**：singleflight 解决了缓存击穿问题。

**待优化**：本地缓存（10s TTL）过期后，仍需查询 MySQL：

```
本地缓存 miss
    ↓
Redis ZREVRANGE → 拿 top N 个 ID
    ↓
MySQL SELECT * FROM articles WHERE id IN (...)  ← 查文章详情
    ↓
MySQL SELECT * FROM article_tags WHERE ...      ← 查标签
```

### 1.2 优化目标

1. **热榜列表**：本地缓存过期后不查 MySQL，从 Redis 获取摘要
2. **文章详情**：热榜 top 文章的详情页请求，从 Redis 缓存获取，不查 MySQL

---

## 2. 热榜缓存优化

### 2.1 问题分析

热榜前端只展示：排名 + 文章标题（不展示浏览量、点赞数等）

当前流程每次本地缓存过期都要查 MySQL 获取标题，但标题几乎不变，完全可以缓存到 Redis。

### 2.2 设计方案

#### 本地缓存（10s TTL）

```go
type HotArticleBrief struct {
    Rank  uint   // 排名：1, 2, 3, 4, 5
    Title string // 文章标题
}

// key: "article:1:5"
// value: [{Rank:1, Title:"xxx"}, {Rank:2, Title:"yyy"}, ...]
```

只缓存前端展示需要的字段，不缓存浏览量、点赞数等。

#### Redis 缓存（新增）

```
# 标题缓存
rank:articles:hot:title:{article_id} → "文章标题"
TTL: 300s (5分钟)

# 示例
rank:articles:hot:title:596827 → "Go 并发编程最佳实践"
rank:articles:hot:title:462400 → "Redis 缓存设计模式"
```

### 2.3 请求流程

```
请求: GET /api/v1/articles/ranking?page=1&page_size=5
│
├─ 本地缓存命中？
│   └─ 直接返回 [{rank, title}, ...]
│
├─ 本地缓存 miss（每 10 秒一次）
│   ├─ singleflight 合并并发
│   ├─ Redis ZREVRANGE rank:articles:hot 0 4
│   │   └─ 返回 top 5 个 ID: [596827, 462400, ...]
│   ├─ Redis MGET rank:articles:hot:title:596827 ...
│   │   ├─ 全部命中 → 组装 [{rank, title}]，写入本地缓存
│   │   └─ 部分 miss → 查 MySQL 补全 → 回写 Redis → 写入本地缓存
│   └─ 返回结果
│
└─ 正常情况下完全不查 MySQL
```

### 2.4 写入时机

| 场景 | 操作 |
|------|------|
| 文章发布 | 写入 `rank:articles:hot:title:{id}` |
| 文章编辑（标题变更） | 更新 `rank:articles:hot:title:{id}` |
| 文章删除/下架 | 删除 `rank:articles:hot:title:{id}` |
| 定时任务重算热榜 | 检查标题是否存在，miss 则补全 |

### 2.5 Redis 内存估算

```
热榜候选：1000 篇
每篇标题：~50 bytes
总计：1000 × 50 = 50KB
```

完全可忽略。

---

## 3. 热门文章详情缓存

### 3.1 问题分析

热榜 top 文章被点击率最高，每次访问详情页都要查 MySQL：

```
GET /api/v1/articles/:id
    ↓
MySQL SELECT * FROM articles WHERE id = ?     ← 包含 content 大字段
    ↓
MySQL SELECT * FROM article_tags WHERE ...    ← 查标签
    ↓
MySQL SELECT * FROM users WHERE id = ?        ← 查作者
```

content 字段可能 10KB-100KB，查询开销大。

### 3.2 设计方案

#### 缓存范围

只缓存热榜 ZSet 中的文章（最多 1000 篇，实际 top 50 覆盖 99% 点击）。

#### Redis 缓存结构

```
# 文章详情缓存
rank:articles:detail:{article_id} → JSON {
    "id": 596827,
    "title": "Go 并发编程最佳实践",
    "summary": "...",
    "content": "...",
    "author_id": 1,
    "author_nickname": "张三",
    "author_avatar": "https://...",
    "tags": [{"id": 1, "name": "Go"}],
    "views": 1234,
    "likes": 56,
    "favorites": 23,
    "comments": 12,
    "status": "published",
    "published_at": "2026-09-01T10:00:00Z",
    "created_at": "2026-09-01T10:00:00Z",
    "updated_at": "2026-09-01T10:00:00Z"
}
TTL: 60s
```

#### 内存估算

```
热榜 top 50 篇（覆盖 99% 点击）
每篇详情：~50KB（含 content）
总计：50 × 50KB = 2.5MB
```

Redis 完全承受得起。

### 3.3 请求流程

```
请求: GET /api/v1/articles/:id
│
├─ 文章在热榜 ZSet 中？
│   ├─ Redis GET rank:articles:detail:{id}
│   │   ├─ 命中 → 直接返回详情
│   │   └─ miss → 查 MySQL → 回写 Redis (TTL 60s) → 返回
│   │
│   └─ 不在热榜中
│       └─ 正常查 MySQL（不缓存）
│
└─ 返回结果
```

### 3.4 写入时机

| 场景 | 操作 |
|------|------|
| 文章发布且进入热榜 | 写入 `rank:articles:detail:{id}` |
| 文章编辑 | 删除 `rank:articles:detail:{id}`（下次访问时重建） |
| 文章删除/下架 | 删除 `rank:articles:detail:{id}` |
| 点赞/收藏/评论 | 更新计数字段（或等 TTL 过期自动刷新） |
| 浏览量增加 | 不更新缓存（等 TTL 过期，或异步聚合后更新） |

### 3.5 缓存失效策略

**方案 A：被动失效（推荐）**

- 文章编辑/删除时主动删除缓存
- 其他情况等 TTL 过期（60s）
- 浏览量等计数器允许 60s 延迟

**方案 B：主动更新**

- 点赞/收藏/评论时更新 Redis 中的计数
- 浏览量异步聚合后更新
- 实现复杂，暂不采用

---

## 4. 实现要点

### 4.1 热榜摘要缓存

```go
// internal/service/ranking.go

const (
    rankKeyArticleTitle = "rank:articles:hot:title:%d"
)

// 获取热榜文章标题
func (s *RankingService) GetArticleTitles(ctx context.Context, ids []uint) (map[uint]string, error) {
    keys := make([]string, len(ids))
    for i, id := range ids {
        keys[i] = fmt.Sprintf(rankKeyArticleTitle, id)
    }
    
    values, err := s.rdb.MGet(ctx, keys...).Result()
    if err != nil {
        return nil, err
    }
    
    result := make(map[uint]string)
    var missIDs []uint
    for i, v := range values {
        if v == nil {
            missIDs = append(missIDs, ids[i])
        } else {
            result[ids[i]] = v.(string)
        }
    }
    
    // 补全 miss 的标题
    if len(missIDs) > 0 {
        var articles []model.Article
        s.articleRepo.DB().WithContext(ctx).
            Where("id IN ?", missIDs).
            Select("id, title").
            Find(&articles)
        
        pipe := s.rdb.Pipeline()
        for _, a := range articles {
            result[a.ID] = a.Title
            key := fmt.Sprintf(rankKeyArticleTitle, a.ID)
            pipe.Set(ctx, key, a.Title, 300*time.Second)
        }
        pipe.Exec(ctx)
    }
    
    return result, nil
}
```

### 4.2 文章详情缓存

```go
// internal/service/article.go

const (
    rankKeyArticleDetail = "rank:articles:detail:%d"
)

func (s *ArticleService) GetWithCache(ctx context.Context, id uint) (*ArticleDetail, error) {
    // 检查是否在热榜中
    score, err := s.rdb.ZScore(ctx, rankKeyArticles, id).Result()
    if err == redis.Nil || score == 0 {
        // 不在热榜，正常查 MySQL
        return s.getByID(ctx, id)
    }
    
    // 在热榜，尝试从 Redis 获取
    key := fmt.Sprintf(rankKeyArticleDetail, id)
    cached, err := s.rdb.Get(ctx, key).Result()
    if err == nil {
        var detail ArticleDetail
        json.Unmarshal([]byte(cached), &detail)
        return &detail, nil
    }
    
    // miss，查 MySQL 并回写
    detail, err := s.getByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    data, _ := json.Marshal(detail)
    s.rdb.Set(ctx, key, data, 60*time.Second)
    
    return detail, nil
}
```

---

## 5. 性能预期

### 5.1 热榜接口

| 场景 | 修复前 | 优化后 |
|------|--------|--------|
| 本地缓存命中 | ~97ms | ~97ms |
| 本地缓存 miss | ~500ms（查 MySQL） | ~10ms（只查 Redis） |
| QPS（200 并发） | 1,649 | 预期 10,000+ |

### 5.2 文章详情接口

| 场景 | 当前 | 优化后 |
|------|------|--------|
| 热榜文章（缓存命中） | ~100ms（查 MySQL） | ~10ms（只查 Redis） |
| 非热榜文章 | ~100ms | ~100ms（不变） |

---

## 6. 风险与降级

### 6.1 Redis 故障降级

- 热榜：本地缓存 miss → 直接查 MySQL（当前逻辑）
- 文章详情：Redis miss → 直接查 MySQL（当前逻辑）

### 6.2 数据一致性

- 热榜标题：文章编辑后最多 5 分钟生效（TTL）
- 文章详情：编辑后缓存立即失效，下次访问重建

---

## 7. 实现优先级

| 优先级 | 功能 | 收益 |
|--------|------|------|
| P0 | 热榜标题 Redis 缓存 | 消除本地缓存 miss 时的 MySQL 查询 |
| P1 | 热门文章详情缓存 | 减少热榜文章详情页的 MySQL 查询 |

---

## 8. 测试验证

实现后需要验证：

1. 热榜接口：本地缓存 miss 时不查 MySQL
2. 文章详情：热榜文章访问时从 Redis 获取
3. 文章编辑后缓存正确失效
4. Redis 故障时降级到 MySQL
