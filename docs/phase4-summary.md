# P4 阶段总结：标签/搜索/排行优化

## 概述

P4 阶段实现了 AIDevClub 平台的标签管理优化、全文搜索和热门排行优化功能，并完成了前端适配。同时进行了架构简化——去掉文章分类体系，统一使用标签组织内容。

## 完成时间

2026-08-25

## 主要功能

### 1. 标签管理优化

#### 数据模型
- Tag 模型增加 `description` 字段

#### 管理员标签接口
- `POST /api/v1/admin/tags` — 创建标签
- `PUT /api/v1/admin/tags/:id` — 更新标签（部分更新，传什么字段更新什么字段）
- `GET /api/v1/admin/tags` — 管理员标签列表（分页、搜索、状态筛选）

#### 用户端标签优化
- `GET /api/v1/tags` 支持 `prefix` 参数（前缀匹配）
- 热门标签 Redis 缓存（TTL 300 秒）

### 2. 全文搜索

#### MySQL FULLTEXT 索引
- `articles` 表：`idx_ft_article_search (title, summary, content) WITH PARSER ngram`
- `skills` 表：`idx_ft_skill_search (name, description) WITH PARSER ngram`
- `mcp_servers` 表：`idx_ft_mcp_search (name, description) WITH PARSER ngram`
- MySQL 配置 `ngram_token_size=2` 支持中文分词

#### 统一搜索接口
- `GET /api/v1/search?q=keyword&type=article|skill|mcp_server&tag_id=&category_id=&page=&page_size=`
- 支持关键词 + 标签组合筛选
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

#### 定时预计算 + 实时更新
- 后台定时任务每 2 分钟重算所有排行
- 点赞/收藏时异步更新对应内容的热榜分数

#### 排行接口
- `GET /api/v1/articles/ranking?type=hot&page=&page_size=`
- `GET /api/v1/skills/ranking?type=hot|downloads&page=&page_size=`
- `GET /api/v1/mcp-servers/ranking?type=hot|downloads&page=&page_size=`

### 4. 架构简化：去掉分类，统一标签

- 文章不再使用分类体系，统一使用标签组织（与 Skill/MCP Server 一致）
- 首页筛选栏从分类改为热门标签
- 文章卡片和详情页移除分类显示
- 文章编辑页移除分类选择器

### 5. 前端适配

- 导航栏添加搜索框
- 搜索结果页（`/search`）：支持类型切换、高亮显示、分页
- 首页筛选栏改为热门标签
- 侧边栏使用排行接口获取数据
- 管理后台布局（`AdminLayout.vue`）+ 标签管理页面（`/admin/tags`）

### 6. RESTful API 规范化

- 统一使用 GET/POST/PUT/DELETE 四种方法
- 移除 PATCH 方法（用户资料更新改为 PUT）

## 新增文件

**后端：**
- `internal/repo/search.go` — 搜索 repo（FULLTEXT 查询）
- `internal/service/search.go` — 搜索 service（高亮 + 统一搜索）
- `internal/handler/search.go` — 搜索 handler
- `internal/service/ranking.go` — 排行 service（Redis ZSet + 热度计算）
- `internal/handler/ranking.go` — 排行 handler
- `internal/scheduler/ranking.go` — 定时任务（热榜预计算）
- `internal/handler/admin_tag.go` — 管理员标签 handler

**前端：**
- `frontend/src/api/search.ts` — 搜索 API
- `frontend/src/api/ranking.ts` — 排行 API
- `frontend/src/api/adminTag.ts` — 管理员标签 API
- `frontend/src/views/SearchView.vue` — 搜索结果页
- `frontend/src/views/admin/TagManagement.vue` — 标签管理页面
- `frontend/src/components/AdminLayout.vue` — 管理后台布局

## 代码审查修复

- SQL 注入修复（ORDER BY 参数化）
- 缓存 key 生成修复（`string(rune)` → `fmt.Sprintf`）
- FULLTEXT 索引创建错误处理
- 搜索计数错误不再被忽略
- Scheduler goroutine 增加停止机制

## 提交记录

共 20 个提交，涵盖设计文档、后端实现、前端适配、代码审查修复、架构简化。

## 下一步

P5 阶段将实现消息通知和举报审核功能。
