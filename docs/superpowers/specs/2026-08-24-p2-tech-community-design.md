# AIDevClub P2 设计：技术社区（文章 / 评论 / 互动）

日期：2026-08-24
状态：待审查

## 1. 背景与目标

P0+P1（基础设施骨架 + 用户认证）已完成并合并到 master。本阶段（P2）实现需求文档 [doc/AIDevClub需求文档.md](../../doc/AIDevClub需求文档.md) 第 5 节「技术社区」：

1. 文章发布 / 浏览：分类 + 标签筛选、关键词搜索、最新/热门/置顶排序、内容详情、作者公开信息、统计数字。
2. 两级评论：一级评论 + 回复，作者/内容作者删除，评论点赞。
3. 互动：文章点赞、收藏、浏览量统计。

技术栈、代码结构（扁平技术分层 `internal/handler|service|repo|model`）、统一响应与错误码约定、真实 MySQL/Redis 按进程隔离测试方式，均沿用阶段一。

## 2. 范围

### 2.1 包含

- 文章：发布（草稿 / 正式发布）、编辑、删除（软删除）；分页列表；分类 + 标签筛选；关键词搜索（标题/摘要/正文 LIKE）；最新/热门/置顶排序；详情（含浏览量 +1）；作者公开信息。
- 分类：预置固定列表，只读接口（管理员增删改留管理后台阶段）。
- 标签：统一标签池；发布时从已有选择或新建；记录使用次数并展示热门标签。
- 评论：两级结构（一级评论 + 回复，回复的回复归并到一级评论）；删除自己的评论；内容作者删除自己文章下的评论；评论点赞。
- 互动：文章点赞 / 取消、收藏 / 取消（toggle）；浏览量统计。
- 正文图片上传：`/api/v1/articles/images`，本地磁盘存储 + `/static/articles` 静态服务。

### 2.2 不包含（非目标，后续阶段）

- 举报、站内通知 —— P5。
- 文章审核（草稿→待审核→…）——审核流程仅针对 Skill / MCP Server；文章发布直接生效，无需审核。
- 管理员角色与接口（置顶/隐藏的管理接口、分类/标签管理、评论管理）——管理后台阶段；P2 仅预留 `pinned` 字段 + 排序逻辑。
- 浏览量按用户去重、防刷 —— P2 采用简单口径（每次详情访问 +1），去重留后续。
- 全文搜索（FULLTEXT / 外部引擎）、跨类型搜索 —— 后续搜索阶段，P2 用 MySQL LIKE。
- 前端 —— 后续阶段，本阶段仅 REST API。

## 3. 关键决策

| 决策项 | 结论 |
|---|---|
| 代码结构 | 沿用扁平技术分层（`internal/handler`/`service`/`repo`/`model`），按领域新增 category / tag / article / comment / interaction 文件 |
| 分类管理 | 预置固定列表（Go、后端、前端、AI/LLM、DevOps、数据库、移动端、安全、其他），只读；管理员增改留管理后台 |
| 标签管理 | 用户发布时可从已有选择或新建（新建即生效），`enabled` 禁用/合并留管理后台；查询过滤禁用标签 |
| 互动建模 | 独立互动表（`article_likes` / `article_favorites` / `comment_likes`）+ 计数冗余列（事务内同步增减） |
| 互动语义 | 点赞/收藏为 toggle：已点赞再点 = 取消；重复操作幂等返回当前状态 |
| 浏览量 | 详情访问时 `views + 1`（游客也算），无去重 |
| 热门公式 | `score = views + 3×likes + 5×favorites + 2×comments`；`sort=hot` 分页结果缓存 Redis 60s |
| 文章状态 | `draft` / `published`，无需审核；`draft` 仅作者可见，`published` 公开；软删除 |
| 置顶 | P2 预留 `pinned` 字段 + `sort=pinned` 排序，无管理员接口（数据直接写库） |
| 两级评论 | 回复的 `parent_id` 总指向一级评论；「回复的回复」归并到一级评论 |
| 正文图片 | 复用头像上传模式（本地磁盘 + `/static`），扩展名 + 大小校验 |

## 4. 数据模型

沿用 GORM 约定（`created_at` / `updated_at`，软删除用 `gorm.DeletedAt`），与 `model.User` 一致。

### 4.1 categories（分类，预置只读）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint 主键 | |
| name | varchar(64)，唯一 | 显示名 |
| slug | varchar(64)，唯一 | URL 友好标识 |
| sort_order | int | 排序权重，小在前 |

### 4.2 tags（标签，统一标签池）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint 主键 | |
| name | varchar(64)，唯一 | 标签名 |
| usage_count | int | 被文章关联的次数，关联 +1 / 解除 -1 |
| enabled | bool，默认 true | 禁用留管理后台；查询过滤禁用 |

### 4.3 articles（文章，软删除）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint 主键 | |
| author_id | uint，索引 | 作者 |
| category_id | uint，索引 | 分类 |
| title | varchar(200) | |
| summary | varchar(500)，可空 | 列表摘要 |
| content | mediumtext | Markdown 原文 |
| status | varchar(16) | `draft` / `published` |
| views | int | 浏览量 |
| likes_count / favorites_count / comments_count | int | 计数冗余列 |
| pinned | bool，默认 false | 预留置顶 |
| published_at | timestamp | 发布/更新时间，列表排序依据 |
| deleted_at | timestamp，索引 | 软删除 |

### 4.4 article_tags（多对多）

`id, article_id, tag_id`，唯一索引 `(article_id, tag_id)`。

### 4.5 article_likes / article_favorites（文章互动）

`id, article_id, user_id, created_at`，唯一索引 `(article_id, user_id)`。

### 4.6 comments（评论，软删除）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint 主键 | |
| article_id | uint，索引 | |
| author_id | uint | |
| parent_id | uint，可空，索引 | null=一级评论；非 null=回复某条一级评论 |
| content | text | |
| likes_count | int | 计数冗余列 |
| deleted_at | timestamp，索引 | 软删除 |

### 4.7 comment_likes（评论点赞）

`id, comment_id, user_id, created_at`，唯一索引 `(comment_id, user_id)`。

## 5. 接口设计（REST API，前缀 `/api/v1`）

### 5.1 文章

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/articles` | 公开 | 列表：`page,page_size(默认20,上限50),category_id,tag_id,keyword,author_id,sort=latest\|hot\|pinned`（默认 latest） |
| GET | `/articles/:id` | 可选 | 详情（浏览量 +1）；登录时附带 `liked/favorited` |
| POST | `/articles` | 🔒 | 发布：`{title,summary?,content,category_id,status=draft\|publish,tag_ids[],tag_names[]}` |
| PUT | `/articles/:id` | 🔒 作者 | 编辑（含改状态、标签 diff） |
| DELETE | `/articles/:id` | 🔒 作者 | 软删除 |
| POST | `/articles/:id/like` | 🔒 | 点赞/取消 toggle，返回 `{liked,likes_count}` |
| POST | `/articles/:id/favorite` | 🔒 | 收藏/取消 toggle，返回 `{favorited,favorites_count}` |
| POST | `/articles/images` | 🔒 | 正文图片上传（multipart `file`），返回 `{url}` |

### 5.2 评论

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/articles/:id/comments` | 公开 | 两级评论列表：一级评论数组，每条带 `replies[]` |
| POST | `/articles/:id/comments` | 🔒 | `{content,parent_id?}` 发评论/回复 |
| DELETE | `/comments/:id` | 🔒 作者或文章作者 | 软删除 |
| POST | `/comments/:id/like` | 🔒 | 点赞/取消 toggle，返回 `{liked,likes_count}` |

### 5.3 分类 / 标签

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/categories` | 公开 | 预置分类全量列表 |
| GET | `/tags` | 公开 | 标签：`keyword`（前缀过滤）、`hot`（按 `usage_count` 降序 Top N） |

### 5.4 响应结构

- 列表：`{list, total, page, page_size}`，`list` 项含 `id,title,summary,category{id,name},tags[{id,name}],author{id,nickname,avatar_url},views,likes_count,favorites_count,comments_count,published_at,pinned`。
- 详情：追加 `content`、当前用户 `liked/favorited`。
- 统一外层沿用 `platform.Response{code,message,data}`。

### 5.5 可见性规则

- 列表/详情只返回 `published 且未删除`；`draft` 仅作者本人通过详情接口可见（列表不出现）。
- 删除的评论列表不展示。

## 6. 关键实现

### 6.1 互动 toggle + 计数一致性（事务）

- 点赞/收藏：`article_likes` INSERT + `articles.likes_count+1` 同一 DB 事务；取消：DELETE + `-1`。
- 幂等：toggle 语义，已存在即取消；唯一索引兜底并发（1062 → 视为已存在，转为取消）。
- 评论点赞同理（`comment_likes` + `comments.likes_count`）。

### 6.2 浏览量

详情访问时 `articles.views = views + 1`（游客也算），无去重；不参与事务，直接 UPDATE。`draft` 文章不计浏览量（非公开内容）。

### 6.3 热门排行 `sort=hot`

- `score = views + 3×likes + 5×favorites + 2×comments`，查询时 SQL 计算。
- 分页结果按 `(sort,page,page_size)` 为 key 缓存 Redis 60s；`latest`（`published_at DESC`）与 `pinned`（`pinned DESC, published_at DESC`）实时查不缓存。
- 互动写库后不主动删缓存 key（TTL 60s 足够新鲜，避免失效风暴）。

### 6.4 标签 usage_count

- 建立关联 +1、解除 -1；编辑文章对 tag 集合做 diff，与文章更新同一事务。
- 请求中 `tag_ids` 与 `tag_names` 合并为一个 tag 集合：`tag_ids` 按 id 引用（不存在返回 40405），`tag_names` 走新建（唯一索引 + 1062 兜底拿/建），合并后按 tag id 去重。
- 关联时跳过 `enabled=false` 的标签（新建标签默认启用）。

### 6.5 两级评论

- 发回复校验 `parent_id` 存在且为一级评论（`parent.parent_id IS NULL`），否则 40002。
- 列表：一次查出文章全部未删评论，内存按 `parent_id` 组装，无 N+1。

### 6.6 正文图片上传

复用 P1 头像模式：`storage/articles/` 目录 + `/static/articles` 静态服务；校验扩展名（jpg/jpeg/png/webp/gif）+ 大小上限；文件名 `randomHex`。配置新增 `article_image.dir / max_bytes`。

## 7. 错误码约定

| code | 含义 |
|---|---|
| 40001 | 参数错误（绑定失败，沿用） |
| 40002 | 业务参数校验失败（长度、非法 status、parent_id 层级等） |
| 40101 | 未认证（沿用） |
| 40301 | 无权限（非作者编辑/删除、非作者管理评论） |
| 40401 | 用户不存在（沿用） |
| 40402 | 文章不存在或不可见 |
| 40403 | 评论不存在 |
| 40404 | 分类不存在 |
| 40405 | 标签不存在 |
| 40901 | 冲突（标签重名等，沿用语义） |
| 50000 | 服务器内部错误（沿用） |

权限规则：

- 游客：浏览（列表/详情/评论/分类/标签）、热门排行。
- 登录用户：点赞/收藏/评论/回复/上传图片/发布文章。
- 作者：编辑/删除自己的文章；删除自己的评论。
- 内容作者：删除自己文章下的任意评论。
- `draft` 文章仅作者本人可见（详情），列表对所有人不可见。

## 8. 测试策略

沿用真实 MySQL/Redis + 按进程隔离（`internal/testutil`）。

- 扩展 `internal/testutil`：`NewTestDB` 迁移全部新增模型；分类种子在测试内插入。
- repo 测试：各表 CRUD、唯一索引、事务计数增减、toggle 幂等。
- service 测试：发布/编辑/删除权限、草稿可见性、标签 diff 与 usage_count、评论层级校验、点赞/收藏/浏览量、热门公式与排序、正文图片大小/类型。
- handler 测试：路由装配（复用 `setupRouter` 模式）、鉴权 401、错误响应格式、列表/详情/评论响应结构。

## 9. 目录与配置变化

```
internal/model/   新增 category.go tag.go article.go comment.go interaction.go
internal/repo/    新增 category.go tag.go article.go comment.go interaction.go
internal/service/ 新增 category.go tag.go article.go comment.go interaction.go
internal/handler/ 新增 category.go tag.go article.go comment.go
internal/platform/config.go  新增 article_image.dir / max_bytes、热门缓存 TTL、分页上限等
cmd/server/main.go           注册新模型迁移、预置分类种子、新路由、/static/articles
```

docker-compose 无需改动（本地磁盘存储，`storage/articles/` 自动创建）。

## 10. 与需求文档的对应

- 5.1 文章管理：发布/编辑/删除、草稿与发布、选择分类和标签、上传正文图片 ✓；管理员置顶/隐藏/删除 → 管理后台阶段。
- 5.2 文章浏览：分页、分类标签筛选、关键词搜索、最新/热门/置顶排序、详情、作者信息、统计数字 ✓。
- 5.3 评论与互动：两级评论、删除自己的评论、内容作者管理评论、内容点赞/评论点赞、收藏、浏览量 ✓；举报 → P5。
- 10.1 分类标签：独立分类体系、标签统一管理、用户优先选已有或新建、使用次数与热门标签 ✓；管理员禁用/合并 → 管理后台。
- 10.2 搜索排行：组合筛选、分页、热门排行 + Redis 缓存 ✓；跨类型搜索 → 后续。
- 11 通知、12 举报、13 管理后台 → 后续阶段。
