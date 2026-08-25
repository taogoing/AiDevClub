# P6 阶段总结：平台 MCP Server + 管理后台

## 概述

P6 阶段实现了 AIDevClub 平台的整站 MCP Server 和完整的管理后台前端。MCP Server 提供 9 个只读工具（6 个公开 + 3 个认证），支持 Claude Code 等 MCP 客户端检索和操作平台内容。管理后台前端包含 Dashboard、用户管理、文章管理、评论管理、资源审核、标签管理、举报处理、公告管理和操作日志 9 个页面。

## 完成时间

2026-08-25

## 主要功能

### 1. MCP Server

#### 架构设计
- 独立进程 `cmd/mcp-server`，与 REST API 共享 Service/Repo 层
- 使用官方 Go SDK `github.com/modelcontextprotocol/go-sdk/mcp` v1.7.x
- Streamable HTTP 传输，无状态 JSON 响应
- JWT Bearer Token 认证，支持匿名和认证两种访问模式

#### 公开工具（6 个）
1. `search_content` — 搜索已发布内容（文章/Skill/MCP Server）
2. `browse_content` — 浏览内容（最新/热门/下载量排序）
3. `get_article` — 获取文章详情（不增加浏览量）
4. `get_skill` — 获取 Skill 详情（含 SKILL.md）
5. `get_mcp_server` — 获取 MCP Server 详情（含 Tools JSON 和 README）
6. `list_taxonomy` — 获取分类和标签列表

#### 认证工具（3 个）
1. `get_my_profile` — 获取当前用户资料
2. `list_my_content` — 获取当前用户的内容列表
3. `list_my_notifications` — 获取当前用户的通知列表（不标记已读）

#### 安全特性
- Origin 校验（精确匹配，不支持通配符）
- 请求体限制（默认 1 MiB）
- 请求超时（默认 30 秒）
- 限流（每用户/IP 每分钟 60 次）
- 无效 Token 返回 401，不降级为匿名
- 健康检查端点：`/healthz`（存活）、`/readyz`（就绪）

### 2. 管理后台后端扩展

#### 用户管理
- `GET /api/v1/admin/users` — 用户列表（关键词/角色筛选、分页）
- `PUT /api/v1/admin/users/:id/role` — 修改用户角色
- 管理员不能修改自己的角色
- 角色变更记录操作日志

#### 文章管理
- `GET /api/v1/admin/articles` — 已发布文章列表（关键词/可见性/作者筛选）
- `GET /api/v1/admin/articles/:id` — 文章详情（不增加浏览量）

#### 评论管理
- `GET /api/v1/admin/comments` — 文章评论列表（关键词/可见性筛选）
- `PUT /api/v1/admin/comments/:id/hide` — 隐藏评论（级联隐藏子评论）
- `PUT /api/v1/admin/comments/:id/unhide` — 恢复评论（仅恢复目标）
- `GET /api/v1/admin/resource-comments` — 资源评论列表（关键词/可见性/资源类型筛选）
- `PUT /api/v1/admin/resource-comments/:id/hide` — 隐藏资源评论
- `PUT /api/v1/admin/resource-comments/:id/unhide` — 恢复资源评论

#### 资源审核
- `GET /api/v1/admin/skills` — Skill 列表（默认待审核，支持已发布/已拒绝/已下架）
- `GET /api/v1/admin/skills/:id` — Skill 详情（含 SKILL.md）
- `GET /api/v1/admin/mcp-servers` — MCP Server 列表
- `GET /api/v1/admin/mcp-servers/:id` — MCP Server 详情（含 Tools JSON 和 README）
- 拒绝审核必须提供 1-500 字符的拒绝原因

#### 举报详情
- `GET /api/v1/admin/reports/:id` — 举报详情（含目标内容信息）
- 按需加载目标内容，避免列表 N+1 查询

#### 数据模型扩展
- `Skill` 模型增加 `RejectReason` 字段
- `McpServer` 模型增加 `RejectReason` 字段
- `AdminLogItem.Detail` 类型从 `string` 改为 `any`（支持 JSON 对象）

### 3. 管理后台前端

#### 基础设施
- `frontend/src/api/admin.ts` — 完整的管理员 API 类型定义和调用函数
- `frontend/src/stores/auth.ts` — 添加 `restoreSession` 方法（并发安全）
- `frontend/src/router/index.ts` — 添加所有 admin 路由 + admin guard

#### 页面实现

**DashboardView** — 数据看板
- 显示用户数、文章数、Skill 数、MCP Server 数、待审核数、待处理举报数
- 使用 Element Plus 数字卡片

**UsersView** — 用户管理
- 关键词搜索（邮箱/昵称）、角色筛选
- 显示用户列表，支持升级/降级角色
- 管理员不能修改自己的角色

**ArticlesView** — 文章管理
- 关键词搜索（标题/摘要）、可见性筛选
- 显示已发布文章列表（包括隐藏内容）
- 支持查看详情（抽屉）、隐藏/恢复操作

**CommentsView** — 评论管理
- 双 Tab 设计：文章评论 / 资源评论
- 独立分页，互不影响
- 关键词搜索、可见性筛选
- 资源评论额外支持资源类型筛选

**ResourcesView** — 资源审核
- 双 Tab 设计：Skills / MCP Servers
- 默认显示待审核，支持已发布/已拒绝/已下架筛选
- 查看详情（抽屉）：Skill 显示 SKILL.md，MCP Server 显示 Tools JSON 和 README
- 审核操作：通过（需确认）、拒绝（需填写原因）

**ReportsView** — 举报处理
- 状态筛选（待处理/已处理/已驳回）
- 查看详情（抽屉）：显示举报人、目标内容、处理操作
- 按需加载目标内容，避免列表 N+1 查询
- 处理操作：隐藏内容、恢复内容、驳回举报

**AnnouncementsView** — 公告管理
- 发布公告（需确认会向所有用户发送通知）
- 历史公告列表

**LogsView** — 操作日志
- 操作类型筛选
- 显示管理员、操作、目标、详情、时间
- 详情支持 JSON 对象和字符串

**TagManagement** — 标签管理（已有）
- 创建/编辑/启禁用标签

#### AdminLayout 更新
- 完整菜单（9 个页面）
- 面包屑导航

## 新增文件

**MCP Server：**
- `internal/mcpserver/server.go` — MCP server 创建
- `internal/mcpserver/handler.go` — HTTP handler + 中间件链
- `cmd/mcp-server/main.go` — MCP 进程入口

**管理后台后端：**
- `internal/repo/user.go` — 增加 `ListUsers`、`FindPublicByIDs` 方法
- `internal/repo/article.go` — 增加 `AdminList`、`AdminFindByID` 方法
- `internal/repo/comment.go` — 增加 `AdminList`、`AdminFindByID` 方法
- `internal/repo/resource_comment.go` — 增加 `AdminList`、`AdminFindByID` 方法
- `internal/repo/skill.go` — 增加 `AdminList`、`AdminFindByID` 方法
- `internal/repo/mcp_server.go` — 增加 `AdminList`、`AdminFindByID` 方法
- `internal/service/admin.go` — 增加用户管理、文章管理、评论管理、资源审核方法
- `internal/service/report.go` — 增加 `AdminGet` 方法

**管理后台前端：**
- `frontend/src/api/admin.ts` — 管理员 API 类型定义
- `frontend/src/views/admin/DashboardView.vue` — 数据看板
- `frontend/src/views/admin/UsersView.vue` — 用户管理
- `frontend/src/views/admin/ArticlesView.vue` — 文章管理
- `frontend/src/views/admin/CommentsView.vue` — 评论管理
- `frontend/src/views/admin/ResourcesView.vue` — 资源审核
- `frontend/src/views/admin/ReportsView.vue` — 举报处理
- `frontend/src/views/admin/AnnouncementsView.vue` — 公告管理
- `frontend/src/views/admin/LogsView.vue` — 操作日志

## API 接口汇总

### 新增管理员接口
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/users` | 用户列表 |
| PUT | `/api/v1/admin/users/:id/role` | 修改用户角色 |
| GET | `/api/v1/admin/articles` | 文章列表 |
| GET | `/api/v1/admin/articles/:id` | 文章详情 |
| GET | `/api/v1/admin/comments` | 文章评论列表 |
| PUT | `/api/v1/admin/comments/:id/hide` | 隐藏评论 |
| PUT | `/api/v1/admin/comments/:id/unhide` | 恢复评论 |
| GET | `/api/v1/admin/resource-comments` | 资源评论列表 |
| PUT | `/api/v1/admin/resource-comments/:id/hide` | 隐藏资源评论 |
| PUT | `/api/v1/admin/resource-comments/:id/unhide` | 恢复资源评论 |
| GET | `/api/v1/admin/skills` | Skill 列表 |
| GET | `/api/v1/admin/skills/:id` | Skill 详情 |
| GET | `/api/v1/admin/mcp-servers` | MCP Server 列表 |
| GET | `/api/v1/admin/mcp-servers/:id` | MCP Server 详情 |
| GET | `/api/v1/admin/reports/:id` | 举报详情 |

### MCP Server 工具
| 工具 | 说明 | 认证要求 |
|------|------|----------|
| `search_content` | 搜索内容 | 无 |
| `browse_content` | 浏览内容 | 无 |
| `get_article` | 获取文章 | 无 |
| `get_skill` | 获取 Skill | 无 |
| `get_mcp_server` | 获取 MCP Server | 无 |
| `list_taxonomy` | 获取分类和标签 | 无 |
| `get_my_profile` | 获取当前用户资料 | Bearer Token |
| `list_my_content` | 获取当前用户内容 | Bearer Token |
| `list_my_notifications` | 获取当前用户通知 | Bearer Token |

## 配置

### MCP Server 配置
```
AIDEVCLUB_MCP_ADDR=:8081
AIDEVCLUB_PUBLIC_BASE_URL=http://localhost:5173
AIDEVCLUB_MCP_ALLOWED_ORIGINS=
AIDEVCLUB_MCP_RATE_LIMIT_PER_MINUTE=60
AIDEVCLUB_MCP_REQUEST_TIMEOUT=30s
AIDEVCLUB_MCP_MAX_BODY_BYTES=1048576
```

### MCP 客户端配置示例
```json
{
  "mcpServers": {
    "aidevclub": {
      "url": "http://localhost:8081/mcp",
      "headers": {
        "Authorization": "Bearer <access-token>"
      }
    }
  }
}
```

## 代码审查修复

代码审查发现并修复了 4 个问题：

1. **中** — `report.go` 的 `resolveTarget` 传 `nil` 给 `FindByID`，未传播请求 context
2. **低** — `admin.go` 循环变量 `s` 遮蔽了接收者 `s *AdminService`
3. **低** — `ListArticleComments` 多余的 `FindPublicByIDs` 查询
4. **低** — `Comment` 模型缺少 `Author` 关联字段

## 验证

- ✅ `go build ./...` 通过
- ✅ `go vet ./...` 通过
- ✅ `npm run typecheck` 通过
- ✅ `npm run build` 通过

## 提交记录

共 20+ 个提交，涵盖：
- MCP Server 实现（server/handler/main）
- 管理后台后端扩展（用户/文章/评论/资源/举报）
- 管理后台前端实现（9 个页面 + API + 路由）
- 代码审查修复
- 文档更新（roadmap）

## 下一步

P6 阶段已完成所有计划功能。平台已具备完整的 MCP Server 和管理后台能力，可以进行生产部署。
