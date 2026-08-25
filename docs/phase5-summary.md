# P5 阶段总结：消息通知 + 举报审核

## 概述

P5 阶段实现了 AIDevClub 平台的站内消息通知系统、举报审核机制和管理员后台功能。用户可以收到评论、回复、点赞、公告等通知，管理员可以处理举报、隐藏/恢复内容、审核资源和发布公告。

## 完成时间

2026-08-25

## 主要功能

### 1. 数据模型扩展

- **User 模型**：增加 `Role` 字段（`user` / `admin`）
- **Article / Skill / McpServer 模型**：增加 `Hidden` 字段（管理员隐藏标记）
- **Comment / ResourceComment 模型**：增加 `Hidden`、`ReplyToID` 字段
- **新增 Notification 模型**：通知类型枚举（comment_article / reply_comment / like_article / like_comment / like_skill / like_mcp_server / like_resource_comment / announcement / report_resolved）
- **新增 Report 模型**：举报目标类型（article / comment / skill / mcp_server / resource_comment）、举报状态（pending / resolved / dismissed）
- **新增 AdminLog 模型**：管理员操作日志（hide / unhide / review / announce / resolve_report）
- **新增 Announcement 模型**：系统公告

### 2. 通知系统

#### 通知触发点
- 评论文章 → 通知文章作者（`comment_article`）
- 回复评论 → 通知被回复者（`reply_comment`）
- 点赞文章 / 评论 / Skill / MCP Server / ResourceComment → 通知作者（`like_*`）
- 发布公告 → 写时扩散通知所有用户（`announcement`）
- 处理举报 → 通知举报人和/或作者（`report_resolved`）
- 自己操作不触发通知（自动跳过）

#### 通知接口（用户端）
- `GET /api/v1/notifications` — 通知列表（分页、类型筛选）
- `GET /api/v1/notifications/unread-count` — 未读计数
- `PUT /api/v1/notifications/:id/read` — 标记单条已读
- `PUT /api/v1/notifications/read` — 全部标记已读

### 3. 举报系统

#### 举报接口（用户端）
- `POST /api/v1/reports` — 提交举报（目标类型 + 原因 + 描述）
- `GET /api/v1/reports` — 查看我的举报列表

#### 管理员处理举报
- `GET /api/v1/admin/reports` — 举报列表（分页、状态筛选）
- `PUT /api/v1/admin/reports/:id/resolve` — 处理举报（hide / unhide / dismiss）

### 4. 管理员后台

#### 数据看板
- `GET /api/v1/admin/dashboard` — 统计数据（用户数、文章数、评论数、待审核数、待处理举报数等）

#### 内容管理（隐藏/恢复）
- `PUT /api/v1/admin/articles/:id/hide` — 隐藏文章
- `PUT /api/v1/admin/articles/:id/unhide` — 恢复文章
- `PUT /api/v1/admin/skills/:id/hide` — 隐藏 Skill
- `PUT /api/v1/admin/skills/:id/unhide` — 恢复 Skill
- `PUT /api/v1/admin/mcp-servers/:id/hide` — 隐藏 MCP Server
- `PUT /api/v1/admin/mcp-servers/:id/unhide` — 恢复 MCP Server

#### 审核管理
- `PUT /api/v1/admin/skills/:id/review` — 审核 Skill（通过/拒绝）
- `PUT /api/v1/admin/mcp-servers/:id/review` — 审核 MCP Server（通过/拒绝）

#### 公告管理
- `POST /api/v1/admin/announcements` — 发布公告（写时扩散通知所有用户）
- `GET /api/v1/admin/announcements` — 公告列表

#### 操作日志
- `GET /api/v1/admin/logs` — 管理员操作日志列表（分页、操作类型筛选）

### 5. Hidden 过滤机制

- 文章 / Skill / MCP Server 列表查询：作者本人可见自己的隐藏内容，其他用户不可见
- 评论列表：过滤隐藏评论
- 搜索结果：过滤隐藏内容
- 隐藏评论时级联隐藏所有子评论
- 恢复评论时仅恢复指定评论，不级联恢复子评论

### 6. 管理员权限控制

- `AdminMiddleware`：验证用户角色为 `admin`，否则返回 403
- 初始管理员通过配置文件 `admin.emails` 指定，启动时自动提升

## 新增文件

**模型层：**
- `internal/model/notification.go` — 通知模型、类型枚举
- `internal/model/report.go` — 举报模型、原因/状态枚举
- `internal/model/admin_log.go` — 管理员操作日志模型、操作类型枚举
- `internal/model/announcement.go` — 系统公告模型

**数据访问层：**
- `internal/repo/notification.go` — 通知数据访问
- `internal/repo/report.go` — 举报数据访问
- `internal/repo/admin_log.go` — 管理员日志数据访问
- `internal/repo/announcement.go` — 公告数据访问

**业务逻辑层：**
- `internal/service/notification.go` — 通知业务逻辑
- `internal/service/report.go` — 举报业务逻辑
- `internal/service/admin.go` — 管理员业务逻辑（看板、审核、内容管理、公告）
- `internal/service/admin_log.go` — 管理员日志业务逻辑

**接口层：**
- `internal/handler/notification.go` — 通知接口
- `internal/handler/report.go` — 举报接口
- `internal/handler/admin.go` — 管理员接口

**平台层：**
- `internal/platform/admin_middleware.go` — 管理员权限中间件

**测试文件：**
- `internal/handler/notification_test.go` — 通知接口测试
- `internal/handler/report_test.go` — 举报接口测试
- `internal/handler/admin_test.go` — 管理员接口测试

## API 接口汇总

### 用户端通知接口
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/notifications` | 通知列表 |
| GET | `/api/v1/notifications/unread-count` | 未读计数 |
| PUT | `/api/v1/notifications/:id/read` | 标记单条已读 |
| PUT | `/api/v1/notifications/read` | 全部标记已读 |

### 用户端举报接口
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/reports` | 提交举报 |
| GET | `/api/v1/reports` | 我的举报列表 |

### 管理员接口
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/dashboard` | 数据看板 |
| PUT | `/api/v1/admin/articles/:id/hide` | 隐藏文章 |
| PUT | `/api/v1/admin/articles/:id/unhide` | 恢复文章 |
| PUT | `/api/v1/admin/skills/:id/hide` | 隐藏 Skill |
| PUT | `/api/v1/admin/skills/:id/unhide` | 恢复 Skill |
| PUT | `/api/v1/admin/skills/:id/review` | 审核 Skill |
| PUT | `/api/v1/admin/mcp-servers/:id/hide` | 隐藏 MCP Server |
| PUT | `/api/v1/admin/mcp-servers/:id/unhide` | 恢复 MCP Server |
| PUT | `/api/v1/admin/mcp-servers/:id/review` | 审核 MCP Server |
| POST | `/api/v1/admin/announcements` | 发布公告 |
| GET | `/api/v1/admin/announcements` | 公告列表 |
| GET | `/api/v1/admin/reports` | 举报列表 |
| PUT | `/api/v1/admin/reports/:id/resolve` | 处理举报 |
| GET | `/api/v1/admin/logs` | 操作日志 |

## 提交记录

共 16 个提交，涵盖设计文档、后端实现（模型 → Repo → Service → Handler）、集成测试、文档归档。

## 测试

- 通知接口测试：列表、未读计数、标记已读、全部已读
- 举报接口测试：提交举报、错误目标、举报列表
- 管理员接口测试：数据看板、隐藏/恢复文章、处理举报、发布公告、操作日志
- 全部 P5 测试通过

## 下一步

P6 阶段将实现整站 MCP Server 和管理后台前端。
