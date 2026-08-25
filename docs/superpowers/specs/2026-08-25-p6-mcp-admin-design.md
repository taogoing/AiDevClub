# P6 平台 MCP Server 与管理后台设计

## 1. 背景与目标

AIDevClub 当前已经完成 P0–P5：基础设施、认证、技术社区、AI 资源、搜索排行、通知、举报与管理员领域接口。P6 在此基础上交付两个运行面：

1. 独立运行的平台 MCP Server，使 MCP 客户端能够搜索和读取平台公开内容，并在配置现有 JWT Access Token 后读取本人的资料、内容和通知。
2. 集成在现有 Vue 应用中的管理后台，使管理员能够查询用户并管理角色、管理公开内容与评论、审核 AI 资源、管理标签、处理举报、查看统计和操作日志、发布公告。

本阶段继续采用模块化单体。REST API 和 MCP Server 是同一 Go Module 中的两个独立进程，直接复用同一套 service 和 repo，不通过内部 REST/RPC 相互调用。

## 2. 范围

### 2.1 MCP Server

MCP 首版提供 9 个只读 Tool：

公开 Tool：

- `search_content`
- `browse_content`
- `get_article`
- `get_skill`
- `get_mcp_server`
- `list_taxonomy`

登录后 Tool：

- `get_my_profile`
- `list_my_content`
- `list_my_notifications`

MCP 采用官方 Go SDK、远程 Streamable HTTP、无状态 JSON Response 和现有 JWT Bearer Token。

### 2.2 管理后台

管理后台包含：

- 数据看板
- 用户管理
- 文章和评论管理
- Skill/MCP Server 审核与可见性管理
- 标签管理
- 举报处理
- 公告管理
- 操作日志

用户管理只涉及查询和 `user/admin` 角色，不引入用户状态机。

## 3. 总体架构

```text
                         ┌─────────────────────────┐
Web / Admin ────────────►│ cmd/server              │
                         │ REST + Admin API        │
                         └────────────┬────────────┘
                                      │
                                      ▼
                              Shared Services
                                      │
                                      ▼
                               Repos / MySQL
                                      │
                                      ├──────── Redis
                                      │
                         ┌────────────┴────────────┐
MCP Clients ────────────►│ cmd/mcp-server          │
                         │ Streamable HTTP MCP     │
                         └─────────────────────────┘
```

两个进程共享源码、模型、Repo、Service、配置格式和发布版本，但具有独立的 HTTP Server、连接池、限流、超时和生命周期。MCP 进程不运行迁移、Seed、静态文件服务或排行 Scheduler。

## 4. 共享应用装配

新增 `internal/app`：

```text
internal/app/
├── infrastructure.go
├── services.go
└── migrations.go
```

`Infrastructure` 持有 GORM DB 和 Redis Client，负责打开、Ping 和关闭连接。`Services` 集中构造当前领域 Service，使 `cmd/server` 和 `cmd/mcp-server` 不复制依赖装配。

迁移逻辑提取为 `app.Migrate`，只由 `cmd/server` 调用。分类 Seed、配置管理员提升和 Ranking Scheduler 也只属于 API 进程。

MCP 包不持有完整 `Services` 或数据库连接，而是在消费侧的单个 `dependencies.go` 中按实际调用定义窄接口。接口只覆盖 Tool 已使用的方法，不为每个领域或方法预建独立接口文件，也不引入 DI 框架。

## 5. MCP 进程与传输

新增：

```text
cmd/mcp-server/main.go
internal/mcpserver/
├── server.go
├── handler.go
├── auth.go
├── errors.go
├── output.go
├── dependencies.go
├── tool_content.go
├── tool_resource.go
├── tool_taxonomy.go
└── tool_account.go
```

Tool 按内容、资源、分类和账户四个内聚领域组织，不采用“一个 Tool 一个文件”。`output.go` 只保留至少被两个 Tool 使用的分页、作者摘要和正文窗口类型；每个 Tool 的输入输出继续使用显式结构体，不建立通用 Tool/响应生成器。

依赖锁定到官方 `github.com/modelcontextprotocol/go-sdk/mcp` v1.7.x。使用 `mcp.NewServer`、泛型 `mcp.AddTool` 和 `mcp.NewStreamableHTTPHandler`。Transport 配置为：

```go
&mcp.StreamableHTTPOptions{
    Stateless:    true,
    JSONResponse: true,
}
```

MCP 进程使用标准库 `net/http`，监听配置地址，暴露：

- `/mcp`
- `/healthz`
- `/readyz`

中间件顺序为 Request ID、Panic Recovery、请求体限制、Origin 校验、Bearer 认证、限流、请求超时和 MCP Handler。

## 6. MCP 认证与安全

请求未携带 Authorization 时建立匿名 Actor，只列出 6 个公开 Tool。合法 Bearer Token 通过现有 `platform.ParseAccessToken` 解析，Actor 只从 Token 获得 UserID，并列出全部 9 个 Tool。

请求携带无效或过期 Token 时返回 HTTP 401 和 `WWW-Authenticate: Bearer`，不得降级为匿名。个人 Tool 不接受任意 `user_id` 参数，并在执行时确认用户仍存在。

Origin 规则：无 Origin 的 CLI/桌面请求允许；存在 Origin 时必须精确匹配 `AIDEVCLUB_MCP_ALLOWED_ORIGINS`。不允许通配符，也不根据 Host 自动信任 Origin。

请求体默认最大 1 MiB，Tool 默认超时 30 秒。所有 context 传递到 service、repo、MySQL 和 Redis；Tool 内不创建脱离请求生命周期的 goroutine。

MCP 限流默认每个登录用户或匿名 Remote IP 每分钟 60 次。Redis 计数提取为框架无关的 RateLimiter，Gin 和 net/http 中间件共同使用。Redis 故障时限流降级放行并记录告警。

日志记录 request ID、MCP method、Tool 名、匿名/用户 ID、耗时、状态、错误码和返回条数；不记录 Token、正文、SKILL.md、README、通知正文或邮箱。

## 7. MCP Tool 契约

所有 Tool 使用 snake_case 字段、RFC 3339 时间、绝对页面 URL、空数组而非 null，并同时返回 Structured Content 和简短文本摘要。

### 7.1 `search_content`

输入：

```json
{
  "query": "Go MCP",
  "content_type": "article",
  "tag_id": 12,
  "category_id": 3,
  "sort": "relevance",
  "page": 1,
  "page_size": 10
}
```

`content_type` 为 `all/article/skill/mcp_server`。`query`、`tag_id`、`category_id` 至少提供一个；`category_id` 只允许用于 article；`relevance` 必须提供 query；无关键词时使用现有领域列表查询。单类型默认 10 条、最大 20 条；all 默认每类 5 条、最大每类 10 条。

单类型返回标准分页。all 分别返回 article、skill、mcp_server section，不声称三类 FULLTEXT relevance 可统一比较。MCP 搜索关闭 Web HTML 高亮。

### 7.2 `browse_content`

输入只包含：

```json
{
  "content_type": "all",
  "sort": "hot",
  "page": 1,
  "page_size": 5
}
```

支持 latest 和 hot；downloads 只支持 skill 或 mcp_server，不能与 all/article 组合。all 按类型分组返回。hot 必须使用 Redis RankingService 的时间衰减 ZSet，并批量加载摘要、保持排行 ID 顺序、过滤已隐藏下架删除内容且不增加浏览量。

### 7.3 详情 Tool

`get_article` 返回文章公开元数据和正文；`get_skill` 返回元数据、下载可用性、文件名、大小和提取后的 SKILL.md；`get_mcp_server` 返回元数据、原样结构化 Tools JSON、README 和下载信息。

文章正文、SKILL.md 和 MCP README 使用：

```json
{
  "content_offset": 0,
  "content_limit": 30000
}
```

默认 30,000、最大 50,000 个 Unicode 字符，并返回 has_more 和 next_offset。下载信息只提供文件元数据和平台详情页，不返回静态存储路径，不代理二进制，也不增加下载数。

MCP 详情读取不增加浏览量，也不查询不会返回的点赞/收藏状态。Service 保留现有 Web `Get` 行为，并增加无统计副作用的 `Read`；二者共享私有详情实现，由私有参数明确控制浏览统计和互动状态查询，不对外暴露通用 options 对象。

### 7.4 `list_taxonomy`

按 kind=`all/categories/tags` 返回可用文章分类和已启用统一标签，支持关键词和 limit。匿名调用不能读取禁用标签。

### 7.5 个人 Tool

`get_my_profile` 返回 ID、昵称、头像、简介和角色，不返回邮箱或认证字段。

`list_my_content` 按 content_type、status 和分页查询当前 Actor 自己的文章、Skill、MCP Server。文章状态为 draft/published，Skill 和 MCP Server 状态为 draft/pending_review/published/rejected/archived。Service 新增 `ListOwned`，可返回本人所有有效状态及本人隐藏内容，不接收其他用户 ID，不包含软删除内容。

`list_my_notifications` 按通知类型、unread_only 和分页查询当前 Actor 通知，不改变已读状态。未读筛选在 Repo 查询中完成，保证 total 和分页正确。

### 7.6 错误

业务错误码仅保留：

- `invalid_argument`
- `content_not_found`
- `result_too_large`
- `temporarily_unavailable`
- `internal_error`

认证和限流分别使用 HTTP 401/429。隐藏、未发布和不存在的他人内容统一为 content_not_found。数据库错误、SQL、路径、Redis Key 和堆栈只写服务端日志。

## 8. 内容可见性与浏览统计

现有详情 Service 必须修复 Hidden 可见性：普通用户只能读取 `published && !hidden`；作者可读取自己的内容，包括管理员隐藏内容；管理员通过专用 AdminGet 读取。

公开列表、搜索、排行和 MCP 均过滤隐藏、未发布、下架和删除内容。MCP Read 不增加 views；Web Get 继续增加 views。

## 9. Skill ZIP 与 SKILL.md

`Skill` 新增：

```go
SkillMD string `gorm:"type:mediumtext"`
```

新上传和重新上传 Skill ZIP 时：

- ZIP 条目不超过 1,000；
- 解压后总大小不超过 100 MiB；
- SKILL.md 不超过 1 MiB；
- 拒绝绝对路径、目录逃逸、Windows 盘符路径和符号链接；
- SKILL.md 必须是非空有效 UTF-8；
- 仅允许根目录或唯一顶层目录中的 SKILL.md；
- 多个候选 SKILL.md 时拒绝上传。

校验成功后保存 ZIP 并原样存储 SkillMD；数据库更新失败时清理本次新文件并保留旧记录。已有 Skill 不批量回填，SkillMD 为空时 MCP 返回 `documentation_available=false`。

## 10. 管理后台权限

所有 `/api/v1/admin/*` 路由统一挂载 AuthMiddleware 和 AdminMiddleware。现有 tags 路由移入受保护的 admin group。

`GET /api/v1/users/me` 增加 role。前端 Auth Store 增加并发安全的 `restoreSession`，使用 Refresh Token 恢复 Access Token 和用户资料；路由守卫在进入 requiresAdmin 页面前检查 role。后端仍是最终授权边界。

## 11. 管理后台页面与接口

### 11.1 数据看板

`GET /api/v1/admin/dashboard` 返回未软删除用户、全部文章、两类评论、全部 Skill、全部 MCP Server、资源下载总数、待审核资源和待处理举报。使用专用聚合查询；任一必要查询失败则接口失败，不返回误导性零值。前端使用 Element Plus 数字卡片，不引入图表依赖。

### 11.2 用户管理

```http
GET /api/v1/admin/users
PUT /api/v1/admin/users/:id/role
```

支持邮箱/昵称关键词、role 和分页。角色只允许 user/admin。管理员不能修改自己的角色；`admin.emails` 中的引导管理员不可降级；无实际变化时不写日志。列表返回 role_mutable。角色变化记录 `update_user_role` 操作日志。

### 11.3 文章管理

```http
GET /api/v1/admin/articles
GET /api/v1/admin/articles/:id
PUT /api/v1/admin/articles/:id/hide
PUT /api/v1/admin/articles/:id/unhide
```

只管理已发布文章，包括正常与隐藏内容；支持关键词、可见性、作者和分页。AdminGet 不增加浏览量。

### 11.4 评论管理

```http
GET /api/v1/admin/comments
GET /api/v1/admin/resource-comments
PUT /api/v1/admin/comments/:id/hide
PUT /api/v1/admin/comments/:id/unhide
PUT /api/v1/admin/resource-comments/:id/hide
PUT /api/v1/admin/resource-comments/:id/unhide
```

文章评论和资源评论使用两个 Tab 和两个独立分页接口，不跨表伪造统一分页。支持正文关键词、可见性；资源评论额外支持 resource_type。隐藏父评论级联隐藏直接子评论，恢复只恢复目标评论。操作记录日志。

### 11.5 资源审核

```http
GET /api/v1/admin/skills
GET /api/v1/admin/skills/:id
GET /api/v1/admin/mcp-servers
GET /api/v1/admin/mcp-servers/:id
```

配合现有 review/hide/unhide 接口。默认查询 pending_review，支持 published/rejected/archived，不展示用户 draft。Skill 详情显示 SKILL.md；MCP Server 详情显示格式化 Tools JSON 和 README。拒绝审核必须提供 1–500 个 Unicode 字符的原因。

### 11.6 标签管理

保留现有创建、编辑、启禁用和分页页面，修复路由保护。TagService 接收 adminID 并为创建、修改、启禁用记录实际变化字段的管理员日志。

### 11.7 举报处理

保留现有列表和 resolve 接口，列表批量补充举报人公开信息。新增 `GET /api/v1/admin/reports/:id`，仅在打开处理抽屉时解析一个异构举报目标，返回目标正文/摘要、Hidden、作者和父资源链接，避免列表 N+1 查询。

### 11.8 公告与日志

公告使用现有列表和发布接口；发布前明确确认会向全部用户发送通知。操作日志列表批量补充管理员公开信息，Detail 合法 JSON 时输出对象，旧文本保留字符串，前端类型为 unknown。

## 12. 前端组织

管理端继续位于现有 `/admin`，新增 Dashboard、Users、Articles、Comments、Resources、Reports、Announcements、Logs 页面并保留 Tags 页面。AdminLayout 增加完整菜单和面包屑。

前端增加集中式管理员 API/类型定义。资源审核抽屉属于明确的跨 Skill/MCP Server 重复，可直接复用；页面头、分页和状态展示先在页面内实现，出现至少两处稳定且同构的重复后再提取。不得实现万能表格、万能表单或配置驱动 CRUD 页面。

列表页统一处理：初始加载、筛选后回第一页、关键词 300ms 防抖、loading、空状态、分页、操作确认、重复提交保护和错误提示。

## 13. 生命周期与健康检查

两个进程都使用显式 `http.Server`，设置 ReadHeader、Read、Write、Idle Timeout，监听 SIGINT/SIGTERM，停止接收请求并最多等待 10 秒后关闭 Redis/MySQL。API RankingScheduler 增加幂等 Stop。

MCP 启动时一次性验证 users、articles、skills、mcp_servers、notifications 等必需表存在，失败则直接退出。运行期 `/healthz` 只表示进程存活；`/readyz` 只 Ping MySQL 和 Redis，避免每次探针重复执行 Schema 查询。任一检查失败时返回 503，但不泄露连接或 Schema 详情。

新增配置：

```text
AIDEVCLUB_MCP_ADDR=:8081
AIDEVCLUB_PUBLIC_BASE_URL=http://localhost:5173
AIDEVCLUB_MCP_ALLOWED_ORIGINS=
AIDEVCLUB_MCP_RATE_LIMIT_PER_MINUTE=60
AIDEVCLUB_MCP_REQUEST_TIMEOUT=30s
AIDEVCLUB_MCP_MAX_BODY_BYTES=1048576
```

两个进程使用各自环境中的现有 `AIDEVCLUB_MYSQL_DSN`，因此部署时可给 MCP 进程配置只读 MySQL 账号，无需专用代码字段。

## 14. 实现精简约束

- 沿用现有 Handler → Service → Repo 分层，不新增通用 CRUD 层、BaseService、BaseRepository 或事件总线。
- 领域输入、输出和管理员 DTO 显式定义；仅复用已确定同义且至少出现两次的小类型，不直接返回 GORM Model。
- MCP 注册使用官方 SDK 的泛型能力，不再包装一层自定义 Tool 框架；只集中处理认证、错误映射、正文窗口和公共输出字段。
- 前后端均不创建占位文件、空实现、未使用扩展点或仅转发一次调用的包装函数。
- 重构只服务于 API 与 MCP 的实际共享路径；与 P6 无关的历史代码不顺手改写。

## 15. 测试

领域/Repo 测试覆盖无副作用 Read、Hidden 可见性、ListOwned、管理员查询、Dashboard、角色安全、标签日志、评论分页、举报详情和 ZIP 安全校验。

MCP Tool 使用 fake reader 单测参数、分组、分页、正文窗口、URL、Tools JSON、错误映射和身份绑定。协议测试使用官方 Go SDK内存 Transport 覆盖匿名/登录 tools/list、tools/call、Schema、Structured Content、错误和取消。HTTP 测试使用 httptest 覆盖 Streamable HTTP、无状态 JSON、JWT、Origin、413、429、超时和健康检查。

管理员 Handler 测试覆盖鉴权、列表、隐藏恢复、审核原因、角色规则、举报详情和日志。前端执行现有 typecheck/build，并手工验收管理员路由、会话恢复、菜单和操作流。

最终验证命令：

```powershell
docker compose up -d
go test ./...
go build ./...
cd frontend
npm run typecheck
npm run build
```

另外启动两个进程并使用 MCP Inspector 验证匿名和 Bearer Token 场景下的 9 个 Tool。

## 16. 交付顺序

实现按依赖顺序推进：

1. 共享应用装配和进程生命周期。
2. Skill ZIP 校验和 SkillMD。
3. 可见性、无副作用 Read、排行摘要和本人内容/通知查询。
4. MCP Server、认证安全中间件和 9 个 Tool。
5. 管理员后端查询、权限修复和日志。
6. Vue 管理后台。
7. 全量测试、MCP Inspector 验收和阶段文档。

每项功能遵循测试先行：先写失败测试，再写最小实现，测试通过后重构。
