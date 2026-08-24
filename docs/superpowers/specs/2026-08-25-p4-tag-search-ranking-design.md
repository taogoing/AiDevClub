# P4 标签/搜索/排行优化设计文档

## 概述

P4 阶段聚焦三个方向：标签管理优化、全文搜索、热门排行优化。文件上传保持现状（本地存储），不做对象存储集成。

## 目标

1. **标签管理**：管理员可创建/编辑/禁用标签，热门标签缓存优化
2. **全文搜索**：MySQL FULLTEXT + ngram 支持中文搜索，统一搜索接口，结果高亮
3. **热门排行**：定时预计算 + Redis ZSet，时间衰减算法，独立排行接口

## 非目标

- 不做文件上传优化（不引入 S3/MinIO）
- 不做标签合并功能（推迟到 P6 管理后台）
- 不引入 Elasticsearch 或 Meilisearch

## 1. 标签管理优化

### 1.1 数据模型变更

`tags` 表增加 `description` 字段：

```go
type Tag struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"size:64;uniqueIndex;not null"`
    Description string `gorm:"size:255"`  // 新增
    UsageCount  int    `gorm:"not null;default:0"`
    Enabled     bool   `gorm:"not null;default:true"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 1.2 管理员标签接口

```
POST   /api/v1/admin/tags              # 创建标签
PUT    /api/v1/admin/tags/:id          # 更新标签
PATCH  /api/v1/admin/tags/:id/disable  # 禁用标签
PATCH  /api/v1/admin/tags/:id/enable   # 启用标签
GET    /api/v1/admin/tags              # 管理员标签列表
```

**权限**：仅管理员可访问 `/api/v1/admin/*` 接口。

**创建标签请求**：
```json
{
  "name": "Go",
  "description": "Go 编程语言相关"
}
```

**管理员列表查询参数**：
- `keyword` — 名称前缀搜索
- `status` — enabled/disabled/all（默认 all）
- `page`, `page_size` — 分页

### 1.3 用户端标签接口优化

`GET /api/v1/tags` 增强：

```
GET /api/v1/tags?prefix=go&hot=1&limit=20
```

- `prefix` — 前缀匹配（用于输入建议），替代原来的 `keyword`
- `hot` — 按使用次数排序
- `limit` — 返回数量限制
- 仅返回 `enabled = true` 的标签

### 1.4 热门标签缓存

- Redis Key：`hot:tags:{limit}`
- TTL：300 秒（5 分钟）
- 标签 `usage_count` 变更时主动删除缓存

### 1.5 禁用标签规则

- 禁用标签后，已关联的内容仍显示该标签
- 用户创建/编辑内容时不能选择禁用标签
- `ResolveTagSet` 方法已校验标签必须 enabled

## 2. 全文搜索

### 2.1 MySQL FULLTEXT 索引

为三张表添加 FULLTEXT 索引，使用 ngram 分词器支持中文：

```sql
ALTER TABLE articles ADD FULLTEXT INDEX idx_ft_article_search (title, summary, content) WITH PARSER ngram;
ALTER TABLE skills ADD FULLTEXT INDEX idx_ft_skill_search (name, description) WITH PARSER ngram;
ALTER TABLE mcp_servers ADD FULLTEXT INDEX idx_ft_mcp_search (name, description) WITH PARSER ngram;
```

ngram 配置（MySQL 全局变量，需在 docker-compose 或 MySQL 配置中设置）：
```
ngram_token_size=2
```

### 2.2 统一搜索接口

```
GET /api/v1/search?q=关键词&type=article|skill|mcp_server&tag_id=1&category_id=2&page=1&page_size=20
```

**参数说明**：
- `q` — 搜索关键词（必填）
- `type` — 内容类型：article/skill/mcp_server。为空时搜索所有类型
- `tag_id` — 标签筛选（可选）
- `category_id` — 分类筛选（仅 article 类型有效）
- `page`, `page_size` — 分页

**响应格式**：

单类型搜索：
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 1,
        "type": "article",
        "title": "<mark>Go</mark> 语言入门",
        "summary": "学习 <mark>Go</mark> 语言的基础知识...",
        "author": {...},
        "tags": [...],
        "views": 100,
        "likes_count": 10,
        "created_at": "2026-08-25T10:00:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

多类型搜索（type 为空）：
```json
{
  "code": 0,
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "counts": {
      "article": 60,
      "skill": 25,
      "mcp_server": 15
    }
  }
}
```

### 2.3 搜索实现

**FULLTEXT 查询**：
```go
// 文章搜索
db.Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", keyword)

// Skill 搜索
db.Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", keyword)

// MCP Server 搜索
db.Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", keyword)
```

**组合筛选**：
```go
query := db.Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", keyword)
if tagID > 0 {
    query = query.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
                Where("article_tags.tag_id = ?", tagID)
}
if categoryID > 0 {
    query = query.Where("category_id = ?", categoryID)
}
```

**相关性排序**：
```go
query.Order("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE) DESC", keyword)
```

### 2.4 搜索结果高亮

后端实现高亮，使用 `<mark>` 标签包裹匹配关键词：

```go
func highlightText(text, keyword string) string {
    // 简单的字符串替换，不区分大小写
    // 对 HTML 实体做转义处理
}
```

高亮字段：
- 文章：title, summary
- Skill：name, description
- MCP Server：name, description

### 2.5 现有列表接口改造

`GET /api/v1/articles`, `/api/v1/skills`, `/api/v1/mcp-servers` 的 `keyword` 参数改用 FULLTEXT 搜索：

```go
// 原来
d = d.Where("(title LIKE ? OR summary LIKE ?)", kw, kw)

// 改为
d = d.Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", keyword)
```

## 3. 热门排行优化

### 3.1 时间衰减算法

采用 Hacker News 风格的时间衰减公式：

```
hot_score = (views + 3*likes + 5*favorites + 2*comments + 1) / (hours_since_publish + 2)^gravity
```

- `gravity` = 1.5（可配置），控制衰减速度
- `hours_since_publish` = `time.Since(published_at).Hours()`
- 加 2 避免除零，加 1 避免零分

**示例**：
- 新文章（1 小时前发布，100 浏览，10 点赞）：
  - score = (100 + 30 + 50 + 0 + 1) / (1 + 2)^1.5 = 181 / 5.2 ≈ 34.8
- 老文章（30 天前发布，1000 浏览，100 点赞）：
  - score = (1000 + 300 + 500 + 0 + 1) / (720 + 2)^1.5 = 1801 / 19447 ≈ 0.09

### 3.2 定时预计算

**后台定时任务**：
- 每 2 分钟执行一次（可配置）
- 计算所有已发布内容的热度分数
- 写入 Redis ZSet

**实现位置**：`internal/scheduler/ranking.go`

```go
func (s *RankingScheduler) Start() {
    ticker := time.NewTicker(2 * time.Minute)
    for range ticker.C {
        s.recalculateArticleHotRanking()
        s.recalculateSkillHotRanking()
        s.recalculateMcpServerHotRanking()
        s.recalculateDownloadRanking()
    }
}
```

### 3.3 Redis ZSet 设计

**热榜 Key**：
- `rank:articles:hot` — 文章热榜，member=article_id，score=hot_score
- `rank:skills:hot` — Skill 热榜
- `rank:mcp_servers:hot` — MCP Server 热榜

**下载排行 Key**：
- `rank:skills:downloads` — Skill 下载排行，member=skill_id，score=downloads
- `rank:mcp_servers:downloads` — MCP Server 下载排行，member=mcp_server_id，score=downloads

**查询示例**：
```go
// 获取热榜 Top 50
ids, _ := rdb.ZRevRange(ctx, "rank:articles:hot", 0, 49).Result()

// 获取分页热榜
ids, _ := rdb.ZRevRange(ctx, "rank:articles:hot", start, stop).Result()
```

### 3.4 实时性优化

内容互动时立即更新该条内容的分数：

```go
// 文章被点赞后
func (s *ArticleService) updateHotScore(ctx context.Context, articleID uint) {
    article := getArticle(articleID)
    score := calculateHotScore(article)
    s.rdb.ZAdd(ctx, "rank:articles:hot", &redis.Z{Score: score, Member: articleID})
}
```

触发更新的事件：
- 点赞/取消点赞
- 收藏/取消收藏
- 发表评论
- 浏览量增加

### 3.5 排行接口

```
GET /api/v1/articles/ranking?type=hot|downloads&page=&page_size=
GET /api/v1/skills/ranking?type=hot|downloads&page=&page_size=
GET /api/v1/mcp-servers/ranking?type=hot|downloads&page=&page_size=
```

**参数**：
- `type` — hot/downloads（文章不支持 downloads）
- `page`, `page_size` — 分页，默认 page=1, page_size=10

**响应**：
```json
{
  "code": 0,
  "data": {
    "items": [...],
    "total": 1000,
    "page": 1,
    "page_size": 10
  }
}
```

### 3.6 缓存策略汇总

| Key | TTL | 更新策略 |
|-----|-----|----------|
| `rank:articles:hot` | 永久 | 定时任务每 2 分钟全量重算 + 互动时实时更新单条 |
| `rank:skills:hot` | 永久 | 同上 |
| `rank:mcp_servers:hot` | 永久 | 同上 |
| `rank:skills:downloads` | 永久 | 定时任务每 5 分钟重算 |
| `rank:mcp_servers:downloads` | 永久 | 同上 |
| `hot:tags:{limit}` | 300s | usage_count 变更时失效 |

## 4. 数据库迁移

### 4.1 新增字段

```sql
ALTER TABLE tags ADD COLUMN description VARCHAR(255) DEFAULT '' AFTER name;
```

### 4.2 FULLTEXT 索引

```sql
-- 设置 ngram token size（MySQL 全局变量，需在配置文件中设置）
-- ngram_token_size=2

ALTER TABLE articles ADD FULLTEXT INDEX idx_ft_article_search (title, summary, content) WITH PARSER ngram;
ALTER TABLE skills ADD FULLTEXT INDEX idx_ft_skill_search (name, description) WITH PARSER ngram;
ALTER TABLE mcp_servers ADD FULLTEXT INDEX idx_ft_mcp_search (name, description) WITH PARSER ngram;
```

### 4.3 GORM AutoMigrate

在 `internal/platform/database.go` 中确保 Tag 模型更新：

```go
db.AutoMigrate(&model.Tag{})
```

FULLTEXT 索引需要手动创建（GORM 不自动创建 FULLTEXT）。

## 5. 前端变更

### 5.1 统一搜索框

顶部导航栏添加搜索框：
- 输入关键词，回车跳转到搜索结果页
- 搜索结果页：`/search?q=keyword&type=&tag_id=`
- 支持切换类型（全部/文章/Skill/MCP Server）
- 显示各类型数量统计

### 5.2 侧边栏排行组件

`ResourceSidebar` 组件改用排行接口：
- 调用 `/api/v1/skills/ranking?type=hot&page_size=10`
- 调用 `/api/v1/mcp-servers/ranking?type=hot&page_size=10`
- 支持切换"热门/下载量"维度

### 5.3 标签管理页面（管理员）

新增标签管理页面（仅管理员可见）：
- 标签列表（表格形式）
- 创建/编辑标签对话框
- 启用/禁用操作

## 6. 配置项

```yaml
# config.yaml
ranking:
  gravity: 1.5              # 时间衰减系数
  recalc_interval: 2m       # 热榜重算间隔
  download_recalc_interval: 5m  # 下载排行重算间隔

search:
  highlight_tag: "mark"     # 高亮标签
```

## 7. 测试计划

### 7.1 后端测试

- 标签管理：创建/更新/禁用/启用、权限校验
- 全文搜索：FULLTEXT 查询、组合筛选、高亮、分页
- 排行：热度计算、Redis ZSet 操作、定时任务

### 7.2 前端测试

- TypeScript 类型检查：`npm run typecheck`
- 构建：`npm run build`

### 7.3 集成测试

- 启动服务，创建测试数据
- 验证搜索接口返回正确结果
- 验证排行接口返回正确顺序
- 验证标签管理接口功能正常

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| FULLTEXT 索引体积大 | 磁盘占用增加 | 社区站点数据量小，影响可忽略 |
| ngram 分词精度 | 搜索准确性 | ngram_token_size=2 对中文效果较好 |
| 定时任务性能 | 数据量大时计算慢 | 当前规模（<10K 内容）完全够用 |
| Redis ZSet 内存占用 | 内存增加 | 仅存储 ID 和分数，占用极小 |

## 9. 里程碑

1. **标签管理**：管理员接口 + 缓存优化（1-2 天）
2. **全文搜索**：FULLTEXT 索引 + 统一搜索接口 + 高亮（2-3 天）
3. **排行优化**：定时任务 + Redis ZSet + 排行接口（2-3 天）
4. **前端适配**：搜索框 + 侧边栏 + 标签管理页面（2-3 天）
5. **测试与优化**：集成测试 + 性能优化（1-2 天）

总计：8-13 天
