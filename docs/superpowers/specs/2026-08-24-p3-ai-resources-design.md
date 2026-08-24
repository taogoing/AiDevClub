# AIDevClub P3 设计：AI 资源（Skills Hub + MCP Hub）

日期：2026-08-24
状态：待审查

## 1. 背景与目标

P0+P1+P2（基础设施骨架 + 用户认证 + 技术社区）已完成并合并到 master。本阶段（P3）实现需求文档 [doc/AIDevClub需求文档.md](../../doc/AIDevClub需求文档.md) 第 6/7/8 节「AI 资源」：

1. Skills Hub：Skill 发布/浏览/下载/互动，ZIP 资源包上传，审核状态机
2. MCP Hub：MCP Server 发布/浏览/下载/互动，Tools 清单 + README 展示，ZIP 资源包上传（可选），审核状态机
3. 资源公共功能：分页浏览、标签筛选、关键词搜索、排序（最新/热门/下载量）、点赞/收藏/评论、浏览量/下载量统计

附带改进：将 P1/P2 handler 层硬编码的错误码数字提取为集中常量。

代码结构、统一响应与错误码约定、真实 MySQL/Redis 按进程隔离测试方式，均沿用已有阶段。

## 2. 范围

### 2.1 包含

- **Skill**：创建（草稿）、编辑、删除（软删除）；上传 ZIP 资源包；提交审核；分页列表（标签筛选/关键词搜索/排序）；详情（浏览量 +1）；下载；点赞/收藏/评论
- **MCP Server**：创建（草稿）、编辑、删除（软删除）；上传 ZIP 资源包（可选）；提交审核；分页列表（标签筛选/关键词搜索/排序）；详情（含 Tools 清单 + README 渲染）；下载；点赞/收藏/评论
- **审核状态机**：draft → pending_review → published / rejected → archived，含状态流转规则与可见性控制
- **资源评论**：统一评论表，两级结构，Skill 和 MCP Server 共用
- **资源互动**：点赞/收藏 toggle + 计数一致性（事务）
- **错误码重构**：提取 P1/P2 handler 层硬编码错误码为集中常量

### 2.2 不包含（非目标，后续阶段）

- 管理员审核接口（approve/reject）—— P6 管理后台阶段
- 管理员角色与权限中间件 —— P6
- 站内通知（审核结果通知等）—— P5
- 举报 —— P5
- 平台 MCP Server —— P6
- 管理后台前端 —— P6
- 浏览量按用户去重、防刷 —— 后续
- 全文搜索（FULLTEXT / 外部引擎）—— 后续

## 3. 关键决策

| 决策项 | 结论 |
|---|---|
| 代码结构 | 沿用扁平技术分层（`internal/handler`/`service`/`repo`/`model`），按领域新增 skill / mcp_server / resource_interaction 文件 |
| 分类 | Skill 和 MCP Server 不设分类，通过统一标签组织 |
| 标签 | 共用已有 `tags` 表，新增 `skill_tags` / `mcp_server_tags` 关联表 |
| MCP Server 信息建模 | 混合方案：核心元数据结构化（name/description/repo_url），Tools 清单用 `tools_json` JSON 字段，安装/配置/环境变量用 `readme` Markdown 字段 |
| 审核状态机 | draft → pending_review → published / rejected → archived；用户点击「发布」进入 pending_review；published 状态下重新上传 ZIP 自动回退 pending_review；编辑文本不触发状态变更 |
| 提交审核 | 用户手动点击「发布」按钮触发 draft → pending_review；pending_review 状态下可撤回至 draft |
| 被拒绝后 | rejected 状态可修改后重新发布 → pending_review |
| 评论 | 统一 `resource_comments` 表（resource_type + resource_id），两级结构，与文章评论独立 |
| 互动 | 独立表（skill_likes / skill_favorites / mcp_server_likes / mcp_server_favorites）+ 计数冗余列（事务内同步增减） |
| 下载 | 仅 published 状态可下载；游客和登录用户均可下载 |
| ZIP 存储 | 本地磁盘：`storage/skills/` 和 `storage/mcp_servers/`，通过 `/static/skills` 和 `/static/mcp_servers` 静态服务 |
| 热门公式 | `score = views + 3×likes + 5×favorites + 2×comments`；`sort=hot` 分页结果缓存 Redis 60s |
| 错误码 | 集中定义在 `internal/platform/errors.go`，重构 P1/P2 handler 层硬编码 |

## 4. 数据模型

沿用 GORM 约定（`created_at` / `updated_at`，软删除用 `gorm.DeletedAt`），与已有模型一致。

### 4.1 skills（Skill，软删除）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint 主键 | |
| author_id | uint，索引 | 作者 |
| name | varchar(100) | Skill 名称 |
| description | varchar(500) | 简短描述 |
| repo_url | varchar(255)，可空 | 开源仓库地址 |
| zip_url | varchar(255)，可空 | ZIP 包存储路径 |
| zip_filename | varchar(255)，可空 | 原始文件名 |
| file_size | bigint，默认 0 | 文件大小（字节） |
| status | varchar(16)，索引 | draft/pending_review/published/rejected/archived |
| views | int，默认 0 | 浏览量 |
| downloads | int，默认 0 | 下载量 |
| likes_count | int，默认 0 | 点赞数 |
| favorites_count | int，默认 0 | 收藏数 |
| comments_count | int，默认 0 | 评论数 |
| pinned | bool，默认 false | 管理员置顶（P6） |
| published_at | timestamp，可空 | 首次发布时间 |
| created_at / updated_at | timestamp | |
| deleted_at | timestamp，索引 | 软删除 |

### 4.2 mcp_servers（MCP Server，软删除）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint 主键 | |
| author_id | uint，索引 | 作者 |
| name | varchar(100) | MCP Server 名称 |
| description | varchar(500) | 简短描述 |
| repo_url | varchar(255)，可空 | 源码仓库 |
| tools_json | json，可空 | Tools 清单：`[{name, description, inputSchema}]` |
| readme | mediumtext，可空 | 安装方式/配置示例/环境变量（Markdown） |
| zip_url | varchar(255)，可空 | ZIP 包路径（可选） |
| zip_filename | varchar(255)，可空 | 原始文件名 |
| file_size | bigint，默认 0 | 文件大小，无 ZIP 时为 0 |
| status | varchar(16)，索引 | 同 skills |
| views / downloads / likes_count / favorites_count / comments_count | int | 同 skills |
| pinned | bool，默认 false | |
| published_at | timestamp，可空 | |
| created_at / updated_at / deleted_at | | 同 skills |

### 4.3 资源标签关联表

- `skill_tags`：`id, skill_id, tag_id, created_at`，唯一索引 `(skill_id, tag_id)`
- `mcp_server_tags`：`id, mcp_server_id, tag_id, created_at`，唯一索引 `(mcp_server_id, tag_id)`

### 4.4 资源互动表

- `skill_likes`：`id, skill_id, user_id, created_at`，唯一索引 `(skill_id, user_id)`
- `skill_favorites`：`id, skill_id, user_id, created_at`，唯一索引 `(skill_id, user_id)`
- `mcp_server_likes`：`id, mcp_server_id, user_id, created_at`，唯一索引 `(mcp_server_id, user_id)`
- `mcp_server_favorites`：`id, mcp_server_id, user_id, created_at`，唯一索引 `(mcp_server_id, user_id)`

### 4.5 resource_comments（资源评论，软删除）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint 主键 | |
| resource_type | varchar(16) | `skill` 或 `mcp_server` |
| resource_id | uint | 资源 ID |
| author_id | uint | 评论作者 |
| parent_id | uint，可空，索引 | null=一级评论；非 null=回复某条一级评论 |
| content | text | |
| likes_count | int，默认 0 | 点赞数 |
| created_at / updated_at | timestamp | |
| deleted_at | timestamp，索引 | 软删除 |

索引：`(resource_type, resource_id)` 联合索引

### 4.6 resource_comment_likes（资源评论点赞）

`id, comment_id, user_id, created_at`，唯一索引 `(comment_id, user_id)`

## 5. 接口设计（REST API，前缀 `/api/v1`）

### 5.1 Skills Hub

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/skills` | 公开 | 列表：`page,page_size(默认20,上限50),tag_id,keyword,sort=latest\|hot\|downloads`（默认 latest） |
| GET | `/skills/:id` | 可选 | 详情（浏览量 +1）；登录时返回 `liked/favorited` |
| POST | `/skills` | 🔒 | 创建：`{name,description,repo_url?,tag_ids[],tag_names[]}`，创建后为 draft |
| PUT | `/skills/:id` | 🔒 作者 | 编辑文本字段（pending_review 状态不可编辑） |
| DELETE | `/skills/:id` | 🔒 作者 | 软删除 |
| POST | `/skills/:id/upload` | 🔒 作者 | 上传 ZIP（multipart `file`）；published 状态上传自动回退 pending_review |
| POST | `/skills/:id/submit` | 🔒 作者 | 提交审核：draft/rejected/archived → pending_review |
| POST | `/skills/:id/withdraw` | 🔒 作者 | 撤回：pending_review → draft |
| POST | `/skills/:id/archive` | 🔒 作者 | 下架：published → archived |
| POST | `/skills/:id/download` | 公开 | 下载 ZIP（仅 published） |
| POST | `/skills/:id/like` | 🔒 | 点赞/取消 toggle，返回 `{liked,likes_count}` |
| POST | `/skills/:id/favorite` | 🔒 | 收藏/取消 toggle，返回 `{favorited,favorites_count}` |

### 5.2 MCP Hub

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/mcp-servers` | 公开 | 列表：同 skills |
| GET | `/mcp-servers/:id` | 可选 | 详情（含 `tools_json` 和 `readme`） |
| POST | `/mcp-servers` | 🔒 | 创建：`{name,description,repo_url?,tools_json,readme,tag_ids[],tag_names[]}` |
| PUT | `/mcp-servers/:id` | 🔒 作者 | 编辑 |
| DELETE | `/mcp-servers/:id` | 🔒 作者 | 软删除 |
| POST | `/mcp-servers/:id/upload` | 🔒 作者 | 上传 ZIP（可选） |
| POST | `/mcp-servers/:id/submit` | 🔒 作者 | 提交审核 |
| POST | `/mcp-servers/:id/withdraw` | 🔒 作者 | 撤回 |
| POST | `/mcp-servers/:id/archive` | 🔒 作者 | 下架：published → archived |
| POST | `/mcp-servers/:id/download` | 公开 | 下载 ZIP（仅 published 且有 ZIP） |
| POST | `/mcp-servers/:id/like` | 🔒 | 点赞/取消 toggle |
| POST | `/mcp-servers/:id/favorite` | 🔒 | 收藏/取消 toggle |

### 5.3 资源评论（统一）

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/skills/:id/comments` | 公开 | 两级评论列表 |
| GET | `/mcp-servers/:id/comments` | 公开 | 两级评论列表 |
| POST | `/skills/:id/comments` | 🔒 | `{content,parent_id?}` |
| POST | `/mcp-servers/:id/comments` | 🔒 | `{content,parent_id?}` |
| DELETE | `/resource-comments/:id` | 🔒 作者或资源作者 | 软删除 |
| POST | `/resource-comments/:id/like` | 🔒 | 点赞/取消 toggle |

### 5.4 响应结构

- 列表：`{list, total, page, page_size}`，`list` 项含 `id,name,description,tags[{id,name}],author{id,nickname,avatar_url},views,downloads,likes_count,favorites_count,comments_count,status,published_at`
- Skill 详情：追加 `repo_url,zip_url,zip_filename,file_size,liked,favorited`
- MCP Server 详情：追加 `repo_url,tools_json,readme,zip_url,zip_filename,file_size,liked,favorited`
- 统一外层沿用 `platform.Response{code,message,data}`

### 5.5 可见性规则

| 状态 | 作者 | 管理员 | 游客/其他用户 |
|---|---|---|---|
| draft | 可见 | 可见 | 不可见 |
| pending_review | 可见 | 可见 | 不可见 |
| published | 可见 | 可见 | 可见 |
| rejected | 可见 | 可见 | 不可见 |
| archived | 可见 | 可见 | 不可见 |

列表接口只返回 `published` 状态的资源（对所有非作者/非管理员角色）。

## 6. 审核状态机

### 6.1 状态流转图

```
创建 → draft
draft → 提交审核 → pending_review
pending_review → 撤回 → draft
pending_review → (管理员审核通过，P6) → published
pending_review → (管理员拒绝，P6) → rejected
published → 编辑文本 → 仍 published
published → 重新上传 ZIP → pending_review
published → 用户下架 → archived
rejected → 修改后重新发布 → pending_review
archived → 编辑文本 → 仍 archived
archived → 重新上传 ZIP → pending_review
archived → 重新提交审核 → pending_review
```

### 6.2 操作限制

| 操作 | 允许的状态 |
|---|---|
| 编辑文本 | draft, rejected, archived, published |
| 上传 ZIP | draft, rejected, archived, published |
| 提交审核 | draft, rejected, archived |
| 撤回 | pending_review |
| 下架 | published |
| 下载 | published |
| 删除（软删除） | 任意状态 |

### 6.3 ZIP 上传与重新审核

- ZIP 上传独立于创建/编辑接口
- 上传后更新 `zip_url/zip_filename/file_size`
- 若当前 status == `published`，自动回退到 `pending_review`
- 编辑文本字段不触发状态变更

## 7. 错误码定义

集中定义在 `internal/platform/errors.go`：

```go
const (
    // 通用
    CodeParamError    = 40001  // 参数错误
    CodeBizError      = 40002  // 业务参数校验失败
    CodeStateError    = 40003  // 状态不允许当前操作
    CodeUnauthorized  = 40101  // 未认证
    CodeForbidden     = 40301  // 无权限

    // 资源不存在
    CodeUserNotFound       = 40401  // 用户不存在
    CodeArticleNotFound    = 40402  // 文章不存在
    CodeCommentNotFound    = 40403  // 评论不存在
    CodeCategoryNotFound   = 40404  // 分类不存在
    CodeTagNotFound        = 40405  // 标签不存在
    CodeSkillNotFound      = 40406  // Skill 不存在
    CodeMcpServerNotFound  = 40407  // MCP Server 不存在
    CodeResCommentNotFound = 40408  // 资源评论不存在

    // 服务器
    CodeInternalError = 50000  // 服务器内部错误
)
```

重构范围：
- `internal/handler/auth.go` - 硬编码 40001 → `platform.CodeParamError`
- `internal/handler/user.go` - 硬编码 40001 → `platform.CodeParamError`
- `internal/handler/article.go` - 硬编码 40001 → `platform.CodeParamError`
- `internal/handler/comment.go` - 硬编码 40001 → `platform.CodeParamError`
- `internal/handler/category.go` - 硬编码 40001 → `platform.CodeParamError`
- `internal/handler/tag.go` - 硬编码 40001 → `platform.CodeParamError`
- `internal/service/*.go` - `platform.NewBizError` 调用改用常量

## 8. 关键实现

### 8.1 互动 toggle + 计数一致性（事务）

- 点赞/收藏：`skill_likes` INSERT + `skills.likes_count+1` 同一 DB 事务；取消：DELETE + `-1`
- 幂等：toggle 语义，已存在即取消；唯一索引兜底并发（1062 → 视为已存在，转为取消）
- 资源评论点赞同理

### 8.2 浏览量与下载量

- 详情访问时 `views = views + 1`（游客也算），无去重
- 下载时 `downloads = downloads + 1`
- 非 published 状态不计浏览量

### 8.3 热门排行 `sort=hot`

- `score = views + 3×likes + 5×favorites + 2×comments`，查询时 SQL 计算
- 分页结果按 `(resource_type,sort,page,page_size)` 为 key 缓存 Redis 60s
- `sort=downloads` 按 `downloads DESC, id DESC` 实时查不缓存

### 8.4 标签 usage_count

- 建立关联 +1、解除 -1；编辑资源对 tag 集合做 diff，与资源更新同一事务
- 请求中 `tag_ids` 与 `tag_names` 合并为一个 tag 集合（复用已有 `ResolveTagSet` 逻辑）

### 8.5 资源评论

- 统一 `resource_comments` 表，通过 `resource_type` 区分 Skill / MCP Server
- 两级结构：回复的 `parent_id` 总指向一级评论
- 发回复校验 `parent_id` 存在且为一级评论
- 列表：一次查出资源全部未删评论，内存按 `parent_id` 组装

### 8.6 ZIP 上传

- 校验扩展名（`.zip`）+ 大小上限（默认 50MB）
- 文件名 `randomHex + .zip`
- 存储路径：`storage/skills/` 和 `storage/mcp_servers/`
- 静态服务：`/static/skills` 和 `/static/mcp_servers`

## 9. 测试策略

沿用真实 MySQL/Redis + 按进程隔离（`internal/testutil`）。

- 扩展 `internal/testutil`：`NewTestDB` 迁移全部新增模型
- **repo 测试**：资源 CRUD、状态字段、ZIP 字段更新、标签关联、互动 toggle、评论层级
- **service 测试**：
  - 创建/编辑/删除权限校验
  - 状态流转规则（draft→pending_review、published+上传ZIP→pending_review 等）
  - 可见性规则（各状态对不同角色的可见性）
  - 标签 diff 与 usage_count
  - 浏览量/下载量统计
  - 点赞/收藏 toggle
- **handler 测试**：路由装配、鉴权 401、错误响应格式、列表/详情响应结构

## 10. 目录与配置变化

### 10.1 配置变化

`internal/platform/config.go` 新增：

```go
SkillZipDir         string        // ZIP 存储目录，默认 "storage/skills"
McpServerZipDir     string        // ZIP 存储目录，默认 "storage/mcp_servers"
MaxResourceZipBytes int64         // ZIP 大小上限，默认 50MB
```

### 10.2 目录变化

```
internal/platform/errors.go        新增（错误码常量）
internal/model/
  新增 skill.go                    Skill 模型 + SkillTag
  新增 mcp_server.go               MCP Server 模型 + McpServerTag
  新增 resource_interaction.go     资源互动表 + 资源评论表
internal/repo/
  新增 skill.go                    Skill Repo
  新增 mcp_server.go               MCP Server Repo
  新增 resource_interaction.go     资源互动 Repo + 资源评论 Repo
internal/service/
  新增 skill.go                    Skill Service
  新增 mcp_server.go               MCP Server Service
  新增 resource_comment.go         资源评论 Service
  修改 dto.go                      新增资源相关 DTO
internal/handler/
  新增 skill.go                    Skill Handler
  新增 mcp_server.go               MCP Server Handler
  新增 resource_comment.go         资源评论 Handler
  修改 auth.go                     错误码常量化
  修改 user.go                     错误码常量化
  修改 article.go                  错误码常量化
  修改 comment.go                  错误码常量化
  修改 category.go                 错误码常量化
  修改 tag.go                      错误码常量化
internal/platform/
  修改 config.go                   新增 ZIP 存储配置
  修改 errors.go                   新增错误码常量 + BizError 改用常量
cmd/server/main.go                 注册新模型迁移、新路由、静态目录
```

docker-compose 无需改动（本地磁盘存储，自动创建）。

## 11. 与需求文档的对应

- 6.1 Skill 信息字段：名称/描述/作者/仓库/ZIP/大小/浏览量/下载量 ✓
- 7.1 MCP Server 信息字段：名称/描述/Tools 清单/安装方式/配置示例/作者/仓库/ZIP/浏览量/下载量 ✓
- 8.1 公共功能：分页/标签筛选/关键词搜索/排序/详情/发布编辑下架/点赞收藏评论/浏览量下载量/查看自己发布收藏 ✓
- 8.2 资源审核：状态机（draft/pending_review/published/rejected/archived）✓；管理员审核接口 → P6
- 10.1 分类标签：Skill/MCP 不设分类，统一标签组织 ✓
- 11 通知、12 举报、13 管理后台 → 后续阶段
