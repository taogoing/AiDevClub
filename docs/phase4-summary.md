# P4 阶段总结：标签/搜索/排行优化

## 概述

P4 阶段实现了 AIDevClub 平台的标签管理优化、全文搜索和热门排行优化功能。本阶段完成了后端 API、MySQL FULLTEXT 索引、Redis ZSet 排行缓存等核心功能。

## 完成时间

2026-08-25

## 主要功能

### 1. 标签管理优化

#### 数据模型
- Tag 模型增加 `description` 字段

#### 管理员标签接口
- `POST /api/v1/admin/tags` — 创建标签
- `PUT /api/v1/admin/tags/:id` — 更新标签
- `PATCH /api/v1/admin/tags/:id/disable` — 禁用标签
- `PATCH /api/v1/admin/tags/:id/enable` — 启用标签
- `GET /api/v1/admin/tags` — 管理员标签列表（分页、搜索、状态筛选）

#### 用户端标签优化
- `GET /api/v1/tags` 支持 `prefix` 参数（前缀匹配）
- 热门标签 Redis 缓存（TTL 300 秒）

### 2. 全文搜索

#### MySQL FULLTEXT 索引
- 为 `articles` 表创建 FULLTEXT 索引：`idx_ft_article_search (title, summary, content) WITH PARSER ngram`
- 为 `skills` 表创建 FULLTEXT 索引：`idx_ft_skill_search (name, description) WITH PARSER ngram`
- 为 `mcp_servers` 表创建 FULLTEXT 索引：`idx_ft_mcp_search (name, description) WITH PARSER ngram`
- MySQL 配置 `ngram_token_size=2` 支持中文分词

#### 统一搜索接口
- `GET /api/v1/search?q=keyword&type=article|skill|mcp_server&tag_id=&category_id=&page=&page_size=`
- 支持关键词 + 标签 + 分类组合筛选
- 搜索结果按相关性排序
- 搜索结果高亮（使用 `<mark>` 标签）

#### 现有列表接口改造
- 文章、Skill、MCP Server 列表接口的 `keyword` 参数改用 FULLTEXT 搜索

### 3. 热门排行优化

#### 时间衰减算法
采用 Hacker News 风格的时间衰减公式：
```
hot_score = (views + 3*likes + 5*favorites + 2*comments + 1) / (hours_since_publish + 2)^gravity
```
- `gravity` = 1.5，控制衰减速度
- 新内容更容易上榜，老内容自然沉降

#### Redis ZSet 缓存
- `rank:articles:hot` — 文章热榜
- `rank:skills:hot` — Skill 热榜
- `rank:mcp_servers:hot` — MCP Server 热榜
- `rank:skills:downloads` — Skill 下载排行
- `rank:mcp_servers:downloads` — MCP Server 下载排行

#### 定时预计算
- 后台定时任务每 2 分钟重算所有排行
- 实现位置：`internal/scheduler/ranking.go`

#### 排行接口
- `GET /api/v1/articles/ranking?type=hot&page=&page_size=`
- `GET /api/v1/skills/ranking?type=hot|downloads&page=&page_size=`
- `GET /api/v1/mcp-servers/ranking?type=hot|downloads&page=&page_size=`

### 4. 新增组件

**后端：**
- `internal/repo/search.go` — 搜索 repo（FULLTEXT 查询）
- `internal/service/search.go` — 搜索 service（高亮 + 统一搜索）
- `internal/handler/search.go` — 搜索 handler
- `internal/service/ranking.go` — 排行 service（Redis ZSet + 热度计算）
- `internal/handler/ranking.go` — 排行 handler
- `internal/scheduler/ranking.go` — 定时任务（热榜预计算）
- `internal/handler/admin_tag.go` — 管理员标签 handler

**数据库变更：**
- `tags` 表增加 `description` 字段
- `articles`、`skills`、`mcp_servers` 表增加 FULLTEXT 索引

**配置变更：**
- `docker-compose.yml` 增加 MySQL `--ngram-token-size=2` 参数

## 技术亮点

1. **MySQL FULLTEXT + ngram**：支持中文全文搜索，无需额外搜索引擎
2. **Redis ZSet**：高效实现排行榜，支持分页查询
3. **时间衰减算法**：让新内容更容易上榜，老内容自然沉降
4. **定时预计算**：避免实时计算性能问题
5. **统一搜索接口**：支持跨类型搜索和组合筛选

## 文件统计

- 新增文件：7 个
- 修改文件：10+ 个
- 代码变更：约 +1500 行

## 未完成项

- **任务 13**：互动时实时更新热榜分数（deferred）
  - 需要在 Like/Favorite/Comment 等方法中添加更新热榜的调用
  - 当前依赖定时任务每 2 分钟重算，实时性略有不足
  
- **任务 14-16**：前端适配（deferred）
  - 统一搜索框
  - 侧边栏排行组件改用排行接口
  - 管理员标签管理页面

## 测试

后端代码编译通过，需要 MySQL 和 Redis 环境进行集成测试。

## 下一步

P5 阶段将实现消息通知和举报审核功能。前端适配工作可在 P5 或后续阶段完成。

## 总结

P4 阶段成功实现了标签管理优化、全文搜索和热门排行优化功能。后端 API 完整，核心功能可用。前端适配和实时更新优化可作为后续改进项。
