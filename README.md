# AIDevClub

AIDevClub 是一个面向开发者和 AI Agent 的技术内容与 AI 资源分享社区。

## 功能特性

### 三大核心板块

| 板块 | 说明 |
|------|------|
| **技术社区** | 发布和交流技术文章，支持评论、点赞、收藏 |
| **Skills Hub** | 发布和分享 Skill 及其 `SKILL.md` 文档 |
| **MCP Hub** | 发布和分享 MCP Server、安装命令与客户端配置 |

### 其他功能

- 统一标签系统：文章、Skill、MCP Server 共用同一套标签
- 全文搜索：基于 MySQL FULLTEXT + ngram 中文分词
- 热门排行：Redis ZSet 实现，时间衰减算法
- 站内通知：评论、回复、点赞、公告、举报结果等
- 举报审核：管理员处置举报内容
- 管理后台：用户、内容、评论、资源审核、标签、举报、统计、公告、操作日志
- 整站 MCP Server：让 Claude Code、Codex 等 MCP 客户端检索和操作平台内容

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go、Gin、GORM |
| 数据库 | MySQL 8 |
| 缓存 | Redis |
| 前端 | Vue 3、TypeScript、Vite、Element Plus |
| 网关 | Nginx |
| 部署 | Docker、Docker Compose |
| 接口 | REST API、MCP Server |

## 架构

整体采用**模块化单体（Modular Monolith）**架构：

```
cmd/
├── server/          # REST API 服务入口
└── mcp-server/      # MCP Server 入口

internal/
├── app/             # 应用层：基础设施、服务组装、HTTP Server
├── handler/         # HTTP Handler 层
├── service/         # 业务逻辑层
├── repo/            # 数据访问层
├── model/           # 数据模型
├── platform/        # 平台组件：配置、中间件、JWT、限流等
├── mcpserver/       # MCP Server 实现
├── scheduler/       # 定时任务
└── testutil/        # 测试工具

frontend/            # Vue 3 前端
```

**核心设计原则：**

- REST API 与 MCP Server **共用同一套领域服务、权限规则和数据**
- MySQL 持久化业务数据；Redis 负责缓存、限流、Token 状态与热门排行
- 内容与资源采用**软删除**
- Skill 与 MCP Server 需管理员审核（草稿 → 待审核 → 已发布 / 已拒绝 / 已下架）
- 用户认证采用 Access Token + Refresh Token，注册与登录接口限流

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 18+
- Docker & Docker Compose

### 启动基础设施

```bash
docker compose up -d
```

这会启动：
- MySQL 8（端口 3306，数据库 `aidevclub`，用户 `root`，密码 `root`）
- Redis 7（端口 16379）

### 启动后端服务

```bash
# 启动 REST API 服务（默认端口 8080）
go run ./cmd/server

# 启动 MCP Server（默认端口 8081）
go run ./cmd/mcp-server
```

### 配置

通过环境变量配置（前缀 `AIDEVCLUB_`）：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `AIDEVCLUB_HTTP_ADDR` | `:8080` | REST API 监听地址 |
| `AIDEVCLUB_MCP_ADDR` | `:8081` | MCP Server 监听地址 |
| `AIDEVCLUB_MYSQL_DSN` | `root:root@tcp(localhost:3306)/aidevclub?...` | MySQL 连接串 |
| `AIDEVCLUB_REDIS_ADDR` | `localhost:16379` | Redis 地址 |
| `AIDEVCLUB_JWT_SECRET` | `dev-secret-change-me` | JWT 签名密钥（**生产环境必须修改**） |
| `AIDEVCLUB_ADMIN_EMAILS` | - | 管理员邮箱，逗号分隔 |

### 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端开发服务器默认运行在 `http://localhost:5173`。

### 构建

```bash
# 后端
go build ./...

# 前端
cd frontend
npm run build
```

### 测试

```bash
# 后端测试（需要先启动 MySQL 和 Redis）
go test ./...

# 前端类型检查
cd frontend
npm run typecheck
```

## API 概览

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 注册 |
| POST | `/api/v1/auth/login` | 登录 |
| POST | `/api/v1/auth/refresh` | 刷新 Token |
| POST | `/api/v1/auth/logout` | 登出 |

### 用户

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users/me` | 获取当前用户信息 |
| PUT | `/api/v1/users/me` | 更新个人资料 |
| PUT | `/api/v1/users/me/password` | 修改密码 |
| DELETE | `/api/v1/users/me` | 注销账号 |
| POST | `/api/v1/users/me/avatar` | 上传头像 |

### 文章

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/articles` | 文章列表 |
| POST | `/api/v1/articles` | 创建文章 |
| GET | `/api/v1/articles/:id` | 文章详情 |
| PUT | `/api/v1/articles/:id` | 更新文章 |
| DELETE | `/api/v1/articles/:id` | 删除文章 |
| POST | `/api/v1/articles/:id/like` | 点赞/取消点赞 |
| POST | `/api/v1/articles/:id/favorite` | 收藏/取消收藏 |
| POST | `/api/v1/articles/images` | 上传文章图片 |

### Skills

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/skills` | Skill 列表 |
| POST | `/api/v1/skills` | 创建 Skill |
| GET | `/api/v1/skills/:id` | Skill 详情 |
| PUT | `/api/v1/skills/:id` | 更新 Skill |
| DELETE | `/api/v1/skills/:id` | 删除 Skill |
| POST | `/api/v1/skills/:id/submit` | 提交审核 |
| POST | `/api/v1/skills/:id/withdraw` | 撤回审核 |
| POST | `/api/v1/skills/:id/archive` | 下架 Skill |
| POST | `/api/v1/skills/:id/like` | 点赞/取消点赞 |
| POST | `/api/v1/skills/:id/favorite` | 收藏/取消收藏 |

### MCP Servers

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/mcp-servers` | MCP Server 列表 |
| POST | `/api/v1/mcp-servers` | 创建 MCP Server |
| GET | `/api/v1/mcp-servers/:id` | MCP Server 详情 |
| PUT | `/api/v1/mcp-servers/:id` | 更新 MCP Server |
| DELETE | `/api/v1/mcp-servers/:id` | 删除 MCP Server |
| POST | `/api/v1/mcp-servers/:id/submit` | 提交审核 |
| POST | `/api/v1/mcp-servers/:id/withdraw` | 撤回审核 |
| POST | `/api/v1/mcp-servers/:id/archive` | 下架 MCP Server |
| POST | `/api/v1/mcp-servers/:id/like` | 点赞/取消点赞 |
| POST | `/api/v1/mcp-servers/:id/favorite` | 收藏/取消收藏 |

### 搜索与排行

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/search` | 统一搜索 |
| GET | `/api/v1/articles/ranking` | 文章热门排行 |
| GET | `/api/v1/skills/ranking` | Skill 热门排行 |
| GET | `/api/v1/mcp-servers/ranking` | MCP Server 热门排行 |

### 管理后台

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/dashboard` | 数据看板 |
| GET/PUT/DELETE | `/api/v1/admin/users/*` | 用户管理 |
| GET/PUT/DELETE | `/api/v1/admin/articles/*` | 文章管理 |
| GET/PUT/DELETE | `/api/v1/admin/skills/*` | Skill 审核 |
| GET/PUT/DELETE | `/api/v1/admin/mcp-servers/*` | MCP Server 审核 |
| GET/POST | `/api/v1/admin/tags/*` | 标签管理 |
| GET/PUT | `/api/v1/admin/reports/*` | 举报管理 |
| GET/POST | `/api/v1/admin/announcements/*` | 公告管理 |
| GET | `/api/v1/admin/logs` | 操作日志 |

### MCP Server

MCP Server 端点：`/mcp`

支持的工具：
- `search` - 搜索文章、Skill、MCP Server
- `get_article` - 获取文章详情
- `get_skill` - 获取 Skill 详情
- `get_mcp_server` - 获取 MCP Server 详情
- `list_categories` - 获取分类列表
- `list_tags` - 获取标签列表
- `get_ranking` - 获取热门排行
- `get_profile` - 获取用户资料（需认证）
- `get_notifications` - 获取通知列表（需认证）

## 项目路线图

| 阶段 | 名称 | 状态 |
|------|------|------|
| P0 | 基础设施骨架 | ✅ 已完成 |
| P1 | 用户与认证 | ✅ 已完成 |
| P2 | 技术社区 | ✅ 已完成 |
| P3 | AI 资源 | ✅ 已完成 |
| P4 | 标签/搜索/排行优化 | ✅ 已完成 |
| P5 | 消息通知/举报审核 | ✅ 已完成 |
| P6 | 平台 MCP Server/管理后台 | ✅ 已完成 |
| 前端 | 用户端 + 管理端 | ✅ 已完成 |

## 许可证

本项目仅供学习和研究使用。
