# AIDevClub 阶段二总结（P2：技术社区）

日期：2026-08-24

## 完成内容

后端实现了技术社区全部功能——文章发布/浏览、两级评论、点赞/收藏/浏览量统计。已合并到 `master`。

- **数据模型**：Category、Tag、Article（+ ArticleTag）、Comment、ArticleLike、ArticleFavorite、CommentLike（8 张表 + User 共 9 张），全部软删除。
- **分类**：预置 9 个种子分类（Go/后端/前端/AI-LLM/DevOps/数据库/移动端/安全/其他），只读列表接口。
- **标签**：统一标签体系（文章/Skill/MCP 共用），支持关键词前缀搜索 + 热门标签排行（按 usage_count 排序）。
- **文章**：发布（草稿/发布）、编辑（标签 diff + usage_count 增减）、删除（软删除 + 清理标签关联 + usage_count 归零）、列表（分页/分类筛选/标签筛选/关键词搜索/最新/热门/置顶排序）、详情（浏览量 +1、liked/favorited 状态）。
- **评论**：两级结构（一级 + 回复归并到一级）、发表（仅 published 文章）、删除（评论作者或文章作者可删）、评论点赞。
- **互动**：点赞/收藏 toggle（事务内同步计数列）、浏览量统计。
- **热门排行**：SQL 计算（views + 3×likes + 5×favorites + 2×comments），无筛选时 Redis 缓存 60s。
- **正文图片**：上传（jpg/png/webp/gif，≤5MB），本地磁盘 `storage/articles/`。
- **OptionalAuth**：游客可浏览、登录用户可互动的中间件。
- **测试**：全流程测试，repo/service/handler/platform/testutil 各层覆盖，真实 MySQL/Redis 按进程隔离。

## 技术栈与架构

- 沿用 P1 架构：`internal/handler` → `service` → `repo` / `model`，横切能力在 `internal/platform`。
- 新增 `platform.IsDuplicateEntry`（P1 auth.go 本地副本已删除，统一复用）。
- `platform.OptionalAuthMiddleware`：解析 Bearer token 成功则设 user_id，失败/缺失不拦截。
- 文章/评论的计数列（views/likes_count/favorites_count/comments_count）在事务内同步增减，保证一致性。
- 标签 usage_count 在文章创建/编辑/删除事务内增减。

## 目录结构

```text
cmd/server/main.go              # 装配入口（P1+P2 路由/迁移/种子/静态目录）
internal/model/                  # user, category, tag, article(+tag), comment, interaction(3表)
internal/repo/                   # user, token, category, tag, article, comment, interaction
internal/service/                # auth, user, category, tag, article, comment, dto
internal/handler/                # auth, user, category, tag, article, comment, errors
internal/platform/               # config, database, redis, logger, response, errors, middleware, jwt, ratelimit, db(IsDuplicateEntry)
internal/testutil/               # 测试后端（真实 MySQL/Redis，按进程隔离）
docker-compose.yml               # MySQL 8 + Redis 7（Redis 宿主机端口 16379）
```

## 运行与测试

```bash
docker compose up -d            # 启动 MySQL(3306) + Redis(16379)
go build ./...                  # 编译
go test ./...                   # 全量测试
go run ./cmd/server             # 启动服务（:8080），/healthz 健康检查
```

配置通过环境变量（前缀 `AIDEVCLUB_`）覆盖，见 `internal/platform/config.go`。

## API 端点一览

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| POST | /api/v1/auth/register | 限流 | 注册 |
| POST | /api/v1/auth/login | 限流 | 登录 |
| POST | /api/v1/auth/refresh | - | 刷新 Token |
| POST | /api/v1/auth/logout | - | 登出 |
| GET | /api/v1/users/me | Auth | 个人资料 |
| PATCH | /api/v1/users/me | Auth | 修改资料 |
| PUT | /api/v1/users/me/password | Auth | 改密 |
| DELETE | /api/v1/users/me | Auth | 注销账号 |
| POST | /api/v1/users/me/avatar | Auth | 头像上传 |
| GET | /api/v1/categories | - | 分类列表 |
| GET | /api/v1/tags | - | 标签列表（?keyword= / ?hot=1） |
| GET | /api/v1/articles | - | 文章列表（分页/筛选/搜索/排序） |
| POST | /api/v1/articles | Auth | 发布文章 |
| POST | /api/v1/articles/images | Auth | 上传正文图片 |
| GET | /api/v1/articles/:id | Optional | 文章详情 |
| PUT | /api/v1/articles/:id | Auth | 编辑文章 |
| DELETE | /api/v1/articles/:id | Auth | 删除文章 |
| POST | /api/v1/articles/:id/like | Auth | 点赞/取消 |
| POST | /api/v1/articles/:id/favorite | Auth | 收藏/取消 |
| GET | /api/v1/articles/:id/comments | - | 评论列表 |
| POST | /api/v1/articles/:id/comments | Auth | 发表评论 |
| DELETE | /api/v1/comments/:id | Auth | 删除评论 |
| POST | /api/v1/comments/:id/like | Auth | 评论点赞 |
| GET | /static/avatars/* | - | 头像静态文件 |
| GET | /static/articles/* | - | 文章图片静态文件 |

## 统一响应格式

```json
{ "code": 0, "message": "ok", "data": {...} }
```

错误时 `code` 非 0，HTTP status 对应业务错误码。

## 关键设计决策

- **文章只有 draft/published 两个状态**（无"待审核"——审核机制在 P5 实现）。草稿仅作者可见，列表只展示 published。
- **标签自动创建**：发布文章时传入 `tag_names`，不存在的标签自动创建（事务内 + 1062 兜底）。
- **评论两级结构**：回复的回复自动归并到一级评论下（ParentID 重定向到一级父评论）。
- **热门排序 SQL 计算**：`(views + 3*likes_count + 5*favorites_count + 2*comments_count)`，无筛选时 Redis 缓存。
- **权限规则**：文章编辑/删除仅作者；评论删除=评论作者或文章作者；点赞/收藏/评论仅对 published 文章有效。
- **路由注册顺序**：`POST /images` 在 `GET /:id` 之前注册（Gin radix 树按精确段优先匹配）。

## 遗留跟进项

### 设计/安全决策（需拍板，未阻塞）

1. **ResolveTagSet 事务隔离**：`tx` 参数未实际透传到 `TagRepo` 方法，标签创建游离于文章事务之外。若 `SetArticleTags`/`IncrUsage` 失败回滚，已创建的 Tag 行残留为 `usage_count=0` 孤儿。影响有限（唯一索引兜底 + 重名复用），但与设计意图相悖。
2. **软删除残留**：文章软删后 `article_likes`/`article_favorites`/`comments` 未清理。ToggleLike 经 `FindByID` 拦截无法新增互动，孤儿行无害；评论列表已加可见性校验（草稿/已删除文章的评论不可列出）。
3. **`IncrCount` 列名拼接**：`gorm.Expr(column+" + ?", delta)` 硬编码调用无注入风险，但签名对外开放，建议白名单校验。

### Minor 优化项

1. 省略 `status` 字段时返回 400 而非默认 draft（约定俗成应默认 draft）。
2. 设计文档 6.5 写"parent_id 非一级评论 → 40002"，实现为"重定向到一级父"，实现更合理但文档需同步。
3. `commentRouter` 测试为 `comSvc` 新建独立 `ArticleRepo`/`InteractionRepo`，与生产装配共享实例不一致（功能等价）。
4. P1 遗留项 17 项仍open（见阶段一总结），未在 P2 处理。

## 下一步

前端开发（Vue 3 用户端）：文章浏览/发布/评论/互动界面，对接 P1+P2 全部 REST API。后端 P3（Skills Hub + MCP Hub）待前端完成后继续。
