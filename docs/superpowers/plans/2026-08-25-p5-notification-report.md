# P5 消息通知与举报审核实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现消息通知系统、举报审核机制和管理员后台功能。

**架构：** 复用现有分层架构（model → repo → service → handler），通知同步写入，管理员操作记录日志。

**技术栈：** Go 1.24、Gin、GORM、MySQL 8、Redis 7

**设计文档：** `docs/superpowers/specs/2026-08-25-p5-notification-report-design.md`

**执行顺序注意：** 由于服务间存在依赖关系，实际执行顺序应为：
1. 任务 1-5：基础模块（模型、配置、中间件、通知）
2. 任务 7：AdminLogService（无依赖）
3. 任务 8：AdminService（依赖 AdminLogService）
4. 任务 6：ReportService（依赖 AdminService）
5. 任务 9-13：Handler、集成、测试、文档

---

## 文件结构

### 新增文件

| 文件路径 | 职责 |
|----------|------|
| `internal/model/notification.go` | 通知模型、类型枚举 |
| `internal/model/report.go` | 举报模型、原因/状态枚举 |
| `internal/model/admin_log.go` | 管理员操作日志模型、操作类型枚举 |
| `internal/model/announcement.go` | 系统公告模型 |
| `internal/repo/notification.go` | 通知数据访问 |
| `internal/repo/report.go` | 举报数据访问 |
| `internal/repo/admin_log.go` | 管理员日志数据访问 |
| `internal/repo/announcement.go` | 公告数据访问 |
| `internal/service/notification.go` | 通知业务逻辑 |
| `internal/service/report.go` | 举报业务逻辑 |
| `internal/service/admin.go` | 管理员业务逻辑（看板、审核、内容管理、公告） |
| `internal/service/admin_log.go` | 管理员日志业务逻辑 |
| `internal/handler/notification.go` | 通知接口 |
| `internal/handler/report.go` | 举报接口 |
| `internal/handler/admin.go` | 管理员接口（看板、日志、举报处理、内容管理、审核、公告） |
| `internal/platform/admin_middleware.go` | 管理员权限中间件 |

### 修改文件

| 文件路径 | 修改内容 |
|----------|----------|
| `internal/model/user.go` | 增加 Role 字段、UserRole 枚举 |
| `internal/model/article.go` | 增加 Hidden 字段 |
| `internal/model/comment.go` | 增加 Hidden、ReplyToID 字段 |
| `internal/model/resource_interaction.go` | ResourceComment 增加 Hidden、ReplyToID 字段 |
| `internal/model/skill.go` | 增加 Hidden 字段 |
| `internal/model/mcp_server.go` | 增加 Hidden 字段 |
| `internal/repo/user.go` | 增加 AllUserIDs、UpdateRole、Count 方法 |
| `internal/repo/article.go` | 查询增加 hidden 过滤（作者本人除外） |
| `internal/repo/comment.go` | 增加 hidden 过滤、HideChildren 方法 |
| `internal/repo/resource_comment.go` | 增加 hidden 过滤、HideChildren 方法 |
| `internal/repo/skill.go` | 查询增加 hidden 过滤（作者本人除外）、增加 CountByStatus |
| `internal/repo/mcp_server.go` | 查询增加 hidden 过滤（作者本人除外）、增加 CountByStatus |
| `internal/repo/search.go` | 搜索增加 hidden 过滤 |
| `internal/service/comment.go` | 集成通知触发、修改 Create 保留原始 parentID 用于通知 |
| `internal/service/resource_comment.go` | 集成通知触发、修改 Create 保留原始 parentID 用于通知 |
| `internal/service/article.go` | 集成点赞通知 |
| `internal/service/skill.go` | 集成点赞通知 |
| `internal/service/mcp_server.go` | 集成点赞通知 |
| `internal/platform/config.go` | 增加 AdminEmails 配置 |
| `internal/platform/errors.go` | 增加新错误码 |
| `internal/service/dto.go` | 增加新 DTO |
| `cmd/server/main.go` | 注册新路由、初始化新服务、自动迁移新模型、初始管理员设置 |

### 测试文件

| 文件路径 | 职责 |
|----------|------|
| `internal/service/notification_test.go` | 通知服务测试 |
| `internal/service/report_test.go` | 举报服务测试 |
| `internal/handler/notification_test.go` | 通知接口测试 |
| `internal/handler/report_test.go` | 举报接口测试 |
| `internal/handler/admin_test.go` | 管理员接口测试 |

---

## 任务 1：数据模型迁移

**文件：**
- 修改：`internal/model/user.go` — 增加 UserRole 类型和 Role 字段
- 修改：`internal/model/article.go` — 增加 Hidden 字段
- 修改：`internal/model/comment.go` — 增加 Hidden、ReplyToID 字段
- 修改：`internal/model/resource_interaction.go` — ResourceComment 增加 Hidden、ReplyToID 字段
- 修改：`internal/model/skill.go` — 增加 Hidden 字段
- 修改：`internal/model/mcp_server.go` — 增加 Hidden 字段
- 创建：`internal/model/notification.go`
- 创建：`internal/model/report.go`
- 创建：`internal/model/admin_log.go`
- 创建：`internal/model/announcement.go`

- [ ] **步骤 1：修改 User 模型**

在 `internal/model/user.go` 中增加 `UserRole` 类型和 `Role` 字段：

```go
type UserRole string

const (
    UserRoleUser  UserRole = "user"
    UserRoleAdmin UserRole = "admin"
)
```

在 User struct 中 Bio 字段后增加：
```go
Role         UserRole       `gorm:"size:16;not null;default:user"`
```

- [ ] **步骤 2：修改 Article 模型**

在 `internal/model/article.go` 的 Article struct 中 Pinned 字段后增加：
```go
Hidden         bool          `gorm:"not null;default:false"`
```

- [ ] **步骤 3：修改 Comment 模型**

替换 `internal/model/comment.go` 整个文件内容：

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

type Comment struct {
    ID         uint           `gorm:"primaryKey"`
    ArticleID  uint           `gorm:"not null;index"`
    AuthorID   uint           `gorm:"not null;index"`
    ParentID   *uint          `gorm:"index"`
    ReplyToID  *uint          `gorm:"index"`
    Content    string         `gorm:"type:text;not null"`
    LikesCount int            `gorm:"not null;default:0"`
    Hidden     bool           `gorm:"not null;default:false"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  gorm.DeletedAt `gorm:"index"`
}
```

- [ ] **步骤 4：修改 ResourceComment 模型**

在 `internal/model/resource_interaction.go` 的 ResourceComment struct 中：
- 在 `ParentID` 字段后增加 `ReplyToID *uint \`gorm:"index"\``
- 在 `LikesCount` 字段后增加 `Hidden bool \`gorm:"not null;default:false"\``

- [ ] **步骤 5：修改 Skill 模型**

在 `internal/model/skill.go` 的 Skill struct 中 Pinned 字段后增加：
```go
Hidden         bool           `gorm:"not null;default:false"`
```

- [ ] **步骤 6：修改 McpServer 模型**

在 `internal/model/mcp_server.go` 的 McpServer struct 中 Pinned 字段后增加：
```go
Hidden         bool           `gorm:"not null;default:false"`
```

- [ ] **步骤 7：创建通知模型**

创建 `internal/model/notification.go`，内容参照设计文档 §3.1 通知表。

- [ ] **步骤 8：创建举报模型**

创建 `internal/model/report.go`，内容参照设计文档 §3.1 举报表。

- [ ] **步骤 9：创建管理员操作日志模型**

创建 `internal/model/admin_log.go`，内容参照设计文档 §3.1 管理员操作日志表。

- [ ] **步骤 10：创建系统公告模型**

创建 `internal/model/announcement.go`，内容参照设计文档 §3.1 系统公告表。

- [ ] **步骤 11：运行编译验证**

运行：`go build ./...`
预期：编译通过

- [ ] **步骤 12：Commit**

```bash
git add internal/model/
git commit -m "feat(model): add notification, report, admin_log, announcement models and update existing models"
```

---

## 任务 2：新增错误码和配置

**文件：**
- 修改：`internal/platform/errors.go`
- 修改：`internal/platform/config.go`

- [ ] **步骤 1：增加新错误码**

在 `internal/platform/errors.go` 中增加：
```go
CodeReportNotFound       = 40409
CodeNotifNotFound        = 40410
CodeAnnouncementNotFound = 40411
```

- [ ] **步骤 2：增加 AdminEmails 配置**

在 Config struct 中增加 `AdminEmails []string` 字段。
在 `LoadConfig()` 中读取 `admin.emails` 配置（逗号分隔），解析为 `[]string`。
设置默认值 `v.SetDefault("admin.emails", "")`。

- [ ] **步骤 3：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 4：Commit**

```bash
git add internal/platform/errors.go internal/platform/config.go
git commit -m "feat(platform): add error codes and admin emails config"
```

---

## 任务 3：管理员中间件和 UserRepo 扩展

**文件：**
- 创建：`internal/platform/admin_middleware.go`
- 修改：`internal/repo/user.go`

- [ ] **步骤 1：UserRepo 增加方法**

在 `internal/repo/user.go` 中增加：
```go
func (r *UserRepo) UpdateRole(id uint, role model.UserRole) error
func (r *UserRepo) AllUserIDs() ([]uint, error)
func (r *UserRepo) Count() (int64, error)
```

- [ ] **步骤 2：创建管理员中间件**

创建 `internal/platform/admin_middleware.go`：
- 复用 AuthMiddleware 设置的 `user_id`
- 查询用户 Role，非 admin 返回 403

```go
func AdminMiddleware(users *repo.UserRepo) gin.HandlerFunc
```

- [ ] **步骤 3：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 4：Commit**

```bash
git add internal/platform/admin_middleware.go internal/repo/user.go
git commit -m "feat(platform): add admin middleware and user repo methods"
```

---

## 任务 4：通知模块（Repo + Service + DTO）

**文件：**
- 创建：`internal/repo/notification.go`
- 创建：`internal/service/notification.go`
- 修改：`internal/service/dto.go`

- [ ] **步骤 1：创建 NotificationRepo**

参照设计文档 §6 实现：
- `Create(n *model.Notification) error`
- `CreateBatch(notifications []*model.Notification) error`
- `List(ctx, userID, notifType, page, pageSize) ([]model.Notification, int64, error)`
- `UnreadCount(ctx, userID) (int64, error)`
- `MarkRead(ctx, userID, notifID) error`
- `MarkAllRead(ctx, userID) error`

- [ ] **步骤 2：增加通知 DTO**

在 `internal/service/dto.go` 中增加 `NotificationItem` 和 `NotificationListResult`。

- [ ] **步骤 3：创建 NotificationService**

参照设计文档 §6 实现：
- `Create(ctx, userID, notifType, title, content, resourceType, resourceID, actorID)` — 自动跳过 userID == actorID
- `CreateBatchForAllUsers(ctx, notifType, title, content, actorID)` — 查询所有用户 ID 批量插入
- `List(ctx, userID, notifType, page, pageSize) (*NotificationListResult, error)`
- `UnreadCount(ctx, userID) (int64, error)`
- `MarkRead(ctx, userID, notifID) error`
- `MarkAllRead(ctx, userID) error`

- [ ] **步骤 4：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/notification.go internal/service/notification.go internal/service/dto.go
git commit -m "feat(notification): add notification repo, service and DTO"
```

---

## 任务 5：通知模块（Handler + 测试）

**文件：**
- 创建：`internal/handler/notification.go`
- 创建：`internal/handler/notification_test.go`

- [ ] **步骤 1：创建 NotificationHandler**

实现 4 个接口方法：
- `List` — GET `/api/v1/notifications`
- `UnreadCount` — GET `/api/v1/notifications/unread-count`
- `MarkRead` — PUT `/api/v1/notifications/:id/read`
- `MarkAllRead` — PUT `/api/v1/notifications/read`

遵循现有 handler 模式（parseUintParam, queryInt, platform.OK/Fail）。

- [ ] **步骤 2：编写测试**

测试通知列表、未读计数、标记已读。

- [ ] **步骤 3：运行测试**

运行：`go test ./internal/handler/ -run TestNotif -v`

- [ ] **步骤 4：Commit**

```bash
git add internal/handler/notification.go internal/handler/notification_test.go
git commit -m "feat(notification): add notification handler and tests"
```

---

## 任务 6：举报模块（Repo + Service）

**文件：**
- 创建：`internal/repo/report.go`
- 创建：`internal/service/report.go`

**注意：** 本任务依赖任务 7（AdminLogService）和任务 8（AdminService），需要先完成它们。

- [ ] **步骤 1：创建 ReportRepo**

实现：
- `Create(report *model.Report) error`
- `FindByID(id uint) (*model.Report, error)`
- `List(ctx, status, page, pageSize) ([]model.Report, int64, error)`
- `ListByReporter(ctx, reporterID, page, pageSize) ([]model.Report, int64, error)`
- `Update(report *model.Report) error`
- `CountByStatus(ctx, status) (int64, error)`

- [ ] **步骤 2：增加举报 DTO**

在 `internal/service/dto.go` 中增加 `ReportItem`、`ReportListResult`。

- [ ] **步骤 3：创建 ReportService**

**依赖注入：**
```go
type ReportService struct {
    repo       *repo.ReportRepo
    adminSvc   *AdminService      // 用于隐藏/恢复内容
    adminLogSvc *AdminLogService  // 用于记录日志
    notifSvc   *NotificationService // 用于发送通知
}

func NewReportService(
    repo *repo.ReportRepo,
    adminSvc *AdminService,
    adminLogSvc *AdminLogService,
    notifSvc *NotificationService,
) *ReportService
```

实现：
- `Create(ctx, userID, targetType, targetID, reason, description)` — 验证目标存在
- `List(ctx, status, page, pageSize)` — 管理员列表
- `ListByReporter(ctx, reporterID, page, pageSize)` — 用户查看自己的举报
- `Resolve(ctx, adminID, reportID, action, result)` — 处理举报

Resolve 处理逻辑：
1. 验证举报存在且状态为 pending
2. 根据 action 调用 AdminService 方法：
   - `hide` → 调用 `adminSvc.HideContent(targetType, targetID)`（级联隐藏子评论）
   - `unhide` → 调用 `adminSvc.UnhideContent(targetType, targetID)`
   - `dismiss` → 不操作内容
3. 更新举报状态
4. 发送通知（根据 action 规则通知举报人和/或作者）
5. 记录管理员日志

- [ ] **步骤 4：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/report.go internal/service/report.go internal/service/dto.go
git commit -m "feat(report): add report repo and service"
```

---

## 任务 7：管理员操作日志模块

**文件：**
- 创建：`internal/repo/admin_log.go`
- 创建：`internal/service/admin_log.go`

- [ ] **步骤 1：创建 AdminLogRepo**

实现：
- `Create(log *model.AdminLog) error`
- `List(ctx, action, page, pageSize) ([]model.AdminLog, int64, error)`

- [ ] **步骤 2：创建 AdminLogService**

实现：
- `Log(ctx, adminID, action, targetType, targetID, detail)` — detail 序列化为 JSON
- `List(ctx, action, page, pageSize) (*AdminLogListResult, error)`

- [ ] **步骤 3：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 4：Commit**

```bash
git add internal/repo/admin_log.go internal/service/admin_log.go
git commit -m "feat(admin): add admin log repo and service"
```

---

## 任务 8：公告 Repo + 管理员 Service + Handler

**文件：**
- 创建：`internal/repo/announcement.go`
- 创建：`internal/service/admin.go`
- 创建：`internal/handler/admin.go`

- [ ] **步骤 1：创建 AnnouncementRepo**

实现：
- `Create(ann *model.Announcement) error`
- `List(ctx, page, pageSize) ([]model.Announcement, int64, error)`

- [ ] **步骤 2：创建 AdminService**

实现：
- `Dashboard(ctx) (*DashboardData, error)` — 统计数据
- `HideContent(targetType, targetID)` — 统一隐藏内容接口（供 ReportService 调用）
  - 文章/Skill/MCP Server：设置 hidden=true
  - 评论：设置 hidden=true，**级联隐藏所有子评论**
- `UnhideContent(targetType, targetID)` — 统一恢复内容接口（供 ReportService 调用）
  - 仅恢复指定内容，**不级联恢复子评论**
- `HideArticle/UnhideArticle(ctx, adminID, articleID)` — 隐藏/恢复文章（记录日志）
- `HideSkill/UnhideSkill(ctx, adminID, skillID)` — 隐藏/恢复 Skill（记录日志）
- `HideMcpServer/UnhideMcpServer(ctx, adminID, mcpServerID)` — 隐藏/恢复 MCP Server（记录日志）
- `ReviewSkill(ctx, adminID, skillID, approved, reason)` — 审核 Skill
- `ReviewMcpServer(ctx, adminID, mcpServerID, approved, reason)` — 审核 MCP Server
- `CreateAnnouncement(ctx, adminID, title, content)` — 发布公告（写时扩散通知）
- `ListAnnouncements(ctx, page, pageSize)` — 公告列表

所有写操作（除 HideContent/UnhideContent 外）记录管理员日志。

- [ ] **步骤 3：创建 AdminHandler**

实现接口方法：
- `Dashboard` — GET `/api/v1/admin/dashboard`
- `HideArticle/UnhideArticle` — PUT `/api/v1/admin/articles/:id/hide|unhide`
- `HideSkill/UnhideSkill` — PUT `/api/v1/admin/skills/:id/hide|unhide`
- `HideMcpServer/UnhideMcpServer` — PUT `/api/v1/admin/mcp-servers/:id/hide|unhide`
- `ReviewSkill` — PUT `/api/v1/admin/skills/:id/review`
- `ReviewMcpServer` — PUT `/api/v1/admin/mcp-servers/:id/review`
- `CreateAnnouncement` — POST `/api/v1/admin/announcements`
- `ListAnnouncements` — GET `/api/v1/admin/announcements`
- `ListReports` — GET `/api/v1/admin/reports`
- `ResolveReport` — PUT `/api/v1/admin/reports/:id/resolve`
- `ListLogs` — GET `/api/v1/admin/logs`

实现 `RegisterRoutes(r *gin.RouterGroup)` 方法，逐个注册上述路由：

```go
func (h *AdminHandler) RegisterRoutes(r *gin.RouterGroup) {
    r.GET("/dashboard", h.Dashboard)
    r.PUT("/articles/:id/hide", h.HideArticle)
    r.PUT("/articles/:id/unhide", h.UnhideArticle)
    r.PUT("/skills/:id/hide", h.HideSkill)
    r.PUT("/skills/:id/unhide", h.UnhideSkill)
    r.PUT("/skills/:id/review", h.ReviewSkill)
    r.PUT("/mcp-servers/:id/hide", h.HideMcpServer)
    r.PUT("/mcp-servers/:id/unhide", h.UnhideMcpServer)
    r.PUT("/mcp-servers/:id/review", h.ReviewMcpServer)
    r.POST("/announcements", h.CreateAnnouncement)
    r.GET("/announcements", h.ListAnnouncements)
    r.GET("/reports", h.ListReports)
    r.PUT("/reports/:id/resolve", h.ResolveReport)
    r.GET("/logs", h.ListLogs)
}
```

- [ ] **步骤 4：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/announcement.go internal/service/admin.go internal/handler/admin.go
git commit -m "feat(admin): add admin service, handler, announcement repo"
```

---

## 任务 9：举报接口（用户端）

**文件：**
- 创建：`internal/handler/report.go`

- [ ] **步骤 1：创建 ReportHandler**

实现：
- `Create` — POST `/api/v1/reports`
- `List` — GET `/api/v1/reports`（查看我的举报）

- [ ] **步骤 2：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 3：Commit**

```bash
git add internal/handler/report.go
git commit -m "feat(report): add report handler"
```

---

## 任务 10：Repo 层 hidden 过滤、CountByStatus 和搜索过滤

**文件：**
- 修改：`internal/repo/article.go`
- 修改：`internal/repo/skill.go`
- 修改：`internal/repo/mcp_server.go`
- 修改：`internal/repo/comment.go`
- 修改：`internal/repo/resource_comment.go`
- 修改：`internal/repo/search.go`

- [ ] **步骤 1：ArticleRepo 增加 hidden 过滤（作者本人除外）**

在 `baseQuery` 中增加 hidden 过滤，但当查询作者自己的内容时不过滤：

```go
func (r *ArticleRepo) baseQuery(ctx context.Context, q ArticleQuery) *gorm.DB {
    d := r.db.WithContext(ctx).Model(&model.Article{}).Where("status = ?", model.ArticleStatusPublished)
    // 作者查看自己的内容时，不过滤 hidden
    if q.AuthorID == nil {
        d = d.Where("hidden = ?", false)
    }
    // ... 其他条件
}
```

- [ ] **步骤 2：SkillRepo 增加 hidden 过滤（作者本人除外）和 CountByStatus**

同 ArticleRepo 逻辑。增加 `CountByStatus` 方法：

```go
func (r *SkillRepo) CountByStatus(ctx context.Context, status model.ResourceStatus) (int64, error) {
    var total int64
    err := r.db.WithContext(ctx).Model(&model.Skill{}).Where("status = ?", status).Count(&total).Error
    return total, err
}
```

- [ ] **步骤 3：McpServerRepo 增加 hidden 过滤（作者本人除外）和 CountByStatus**

同 SkillRepo。

- [ ] **步骤 4：CommentRepo 增加 hidden 过滤和 HideChildren**

在 `ListByArticle` 中增加 `Where("hidden = ?", false)`：

```go
func (r *CommentRepo) ListByArticle(db *gorm.DB, articleID uint) ([]model.Comment, error) {
    var list []model.Comment
    err := r.exec(db).Where("article_id = ? AND hidden = ?", articleID, false).
        Order("created_at asc, id asc").Find(&list).Error
    return list, err
}
```

增加 `HideChildren` 方法（级联隐藏子评论）：

```go
func (r *CommentRepo) HideChildren(db *gorm.DB, parentID uint) error {
    return r.exec(db).Model(&model.Comment{}).
        Where("parent_id = ?", parentID).
        Update("hidden", true).Error
}
```

- [ ] **步骤 5：ResourceCommentRepo 增加 hidden 过滤和 HideChildren**

同 CommentRepo。

- [ ] **步骤 6：SearchRepo 增加 hidden 过滤**

在 `SearchArticles`、`SearchSkills`、`SearchMcpServers` 中增加 `Where("hidden = ?", false)`：

```go
func (r *SearchRepo) SearchArticles(ctx context.Context, keyword string, tagID, categoryID *uint, page, pageSize int) ([]model.Article, int64, error) {
    query := r.db.WithContext(ctx).
        Model(&model.Article{}).
        Where("status = ?", "published").
        Where("hidden = ?", false).  // 新增
        Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", keyword)
    // ...
}
```

同样修改 `SearchSkills` 和 `SearchMcpServers`。

- [ ] **步骤 7：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 8：Commit**

```bash
git add internal/repo/
git commit -m "feat(repo): add hidden filtering, CountByStatus, and search hidden filter"
```

---

## 任务 11：集成通知触发点 + ReplyToID + 路由注册 + 初始管理员

**重要：** 本任务将通知集成、ReplyToID、路由注册和 main.go 更新合并为一个任务，避免编译断裂。

**文件：**
- 修改：`internal/service/comment.go` — 集成通知 + ReplyToID
- 修改：`internal/service/resource_comment.go` — 集成通知 + ReplyToID
- 修改：`internal/service/article.go` — 集成点赞通知
- 修改：`internal/service/skill.go` — 集成点赞通知
- 修改：`internal/service/mcp_server.go` — 集成点赞通知
- 修改：`cmd/server/main.go` — 更新构造函数、注册路由、初始管理员

- [ ] **步骤 1：修改 CommentService 集成通知 + ReplyToID**

- 注入 `NotificationService` 到 struct
- 修改 `NewCommentService` 增加 `notifSvc *NotificationService` 参数
- `Create()` 中保留原始 parentID 作为 ReplyToID：

```go
replyToID := parentID // 保存原始值用于通知和 ReplyToID
if parentID != nil {
    p, err := s.comments.FindByID(nil, *parentID)
    // ...
    if p.ParentID != nil {
        parentID = p.ParentID // 修正为根评论
    }
}
c := &model.Comment{
    ArticleID: articleID, AuthorID: userID,
    ParentID: parentID, ReplyToID: replyToID,
    Content: content,
}
```

- 评论成功后发送通知：
  - 评论文章 → 通知文章作者（`comment_article`）
  - 回复评论 → 查询 replyToID 对应的用户，通知该用户（`reply_comment`）
  - 自己评论/回复自己不发通知
- `ToggleLike()` 中点赞时通知评论作者（`like_comment`）

- [ ] **步骤 2：修改 ResourceCommentService 集成通知 + ReplyToID**

同 CommentService 逻辑。通知类型为 `like_resource_comment`。

- [ ] **步骤 3：修改 ArticleService 集成通知**

- 注入 `NotificationService`
- 修改 `NewArticleService` 增加 `notifSvc *NotificationService` 参数
- `ToggleLike()` 中点赞时通知文章作者（`like_article`）
- 自己点赞不发通知

- [ ] **步骤 4：修改 SkillService 集成通知**

- 注入 `NotificationService`
- 修改 `NewSkillService` 增加 `notifSvc *NotificationService` 参数
- `ToggleLike()` 中点赞时通知 Skill 作者（`like_skill`）

- [ ] **步骤 5：修改 McpServerService 集成通知**

- 注入 `NotificationService`
- 修改 `NewMcpServerService` 增加 `notifSvc *NotificationService` 参数
- `ToggleLike()` 中点赞时通知 MCP Server 作者（`like_mcp_server`）

- [ ] **步骤 6：更新 main.go 构造函数调用**

由于步骤 1-5 修改了构造函数签名，需要同步更新 main.go 中的调用：

```go
// 先初始化通知服务
notifSvc := service.NewNotificationService(notifRepo, users)

// 更新现有 Service 构造函数调用
comSvc := service.NewCommentService(comments, articles, inter, users, notifSvc)
artSvc := service.NewArticleService(articles, tags, cats, inter, rdb, cfg, notifSvc)
skillSvc := service.NewSkillService(skills, tags, inter, rdb, cfg, notifSvc)
mcpSvc := service.NewMcpServerService(mcpServers, tags, inter, rdb, cfg, notifSvc)
resCommentSvc := service.NewResourceCommentService(resComments, skills, mcpServers, inter, users, notifSvc)
```

- [ ] **步骤 7：自动迁移新模型**

在 `db.AutoMigrate()` 中增加：
```go
&model.Notification{}, &model.Report{}, &model.AdminLog{}, &model.Announcement{}
```

- [ ] **步骤 8：初始化新 Service 和注册路由**

```go
// 初始化新 Repo
adminLogRepo := repo.NewAdminLogRepo(db)
announcementRepo := repo.NewAnnouncementRepo(db)
reportRepo := repo.NewReportRepo(db)

// 初始化新 Service（注意依赖顺序）
adminLogSvc := service.NewAdminLogService(adminLogRepo)
adminSvc := service.NewAdminService(users, articles, skills, mcpServers, reportRepo, announcementRepo, adminLogSvc, notifSvc)
reportSvc := service.NewReportService(reportRepo, adminSvc, adminLogSvc, notifSvc)

// 用户通知路由
nh := handler.NewNotificationHandler(notifSvc)
notifs := r.Group("/api/v1/notifications", p2Auth)
notifs.GET("", nh.List)
notifs.GET("/unread-count", nh.UnreadCount)
notifs.PUT("/:id/read", nh.MarkRead)
notifs.PUT("/read", nh.MarkAllRead)

// 用户举报路由
rh := handler.NewReportHandler(reportSvc)
reports := r.Group("/api/v1/reports", p2Auth)
reports.POST("", rh.Create)
reports.GET("", rh.List)

// 管理员路由（逐个注册，不使用 RegisterRoutes）
adminAuth := r.Group("/api/v1/admin", p2Auth, platform.AdminMiddleware(users))
adminH := handler.NewAdminHandler(adminSvc, reportSvc, adminLogSvc)
adminH.RegisterRoutes(adminAuth)  // 在 AdminHandler 中实现此方法，逐个注册路由
```

- [ ] **步骤 9：初始管理员设置**

在启动时（AutoMigrate 之后）检查 `cfg.AdminEmails`，将匹配的用户提升为 admin：
```go
for _, email := range cfg.AdminEmails {
    u, err := users.FindByEmail(email)
    if err == nil && u.Role != model.UserRoleAdmin {
        _ = users.UpdateRole(u.ID, model.UserRoleAdmin)
    }
}
```

- [ ] **步骤 10：运行编译验证**

运行：`go build ./...`

- [ ] **步骤 11：Commit**

```bash
git add internal/service/ cmd/server/main.go
git commit -m "feat: integrate notifications, ReplyToID, register P5 routes and seed admin users"
```

---

## 任务 12：集成测试

**文件：**
- 创建：`internal/handler/report_test.go`
- 创建：`internal/handler/admin_test.go`

- [ ] **步骤 1：编写举报接口测试**

测试提交举报、查看举报列表。

- [ ] **步骤 2：编写管理员接口测试**

测试数据看板、隐藏/恢复内容、处理举报、发布公告。

- [ ] **步骤 3：运行全部测试**

运行：`go test ./...`
预期：全部通过

- [ ] **步骤 4：Commit**

```bash
git add internal/handler/
git commit -m "test: add report and admin handler tests"
```

---

## 任务 13：文档归档

**文件：**
- 创建：`docs/phase5-summary.md`
- 修改：`docs/roadmap.md`
- 修改：`CLAUDE.md`

- [ ] **步骤 1：编写 P5 阶段总结**

总结完成的功能、新增文件、API 列表。

- [ ] **步骤 2：更新路线图**

标记 P5 为已完成。

- [ ] **步骤 3：更新 CLAUDE.md**

更新现状描述。

- [ ] **步骤 4：Commit**

```bash
git add docs/ CLAUDE.md
git commit -m "docs: P5 阶段归档，更新路线图和 CLAUDE.md"
```
