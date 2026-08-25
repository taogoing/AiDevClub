# P5 消息通知与举报审核设计文档

**日期：** 2026-08-25  
**阶段：** P5 - 消息通知与举报审核  
**状态：** 待实现

---

## 1. 概述

P5 阶段实现消息通知系统和举报审核机制，对应需求文档 §11/§12。

**核心功能：**
- 消息通知：评论、回复、点赞、审核结果、举报处理、系统公告
- 举报审核：用户可举报违规内容，管理员处理并通知相关方
- 管理员看板：数据概览 + 操作日志

**技术选型：** 同步写入通知（方案 1），后期可根据性能数据优化为异步队列。

---

## 2. 整体架构

复用现有分层架构（model → repo → service → handler）：

```
┌─────────────────────────────────────────────────┐
│                    Handler 层                     │
│  NotificationHandler  ReportHandler  AdminHandler │
├─────────────────────────────────────────────────┤
│                    Service 层                     │
│  NotificationService  ReportService  AdminService │
├─────────────────────────────────────────────────┤
│                     Repo 层                       │
│  NotificationRepo  ReportRepo  AdminLogRepo       │
├─────────────────────────────────────────────────┤
│                    Model 层                       │
│  Notification  Report  AdminLog  Announcement     │
└─────────────────────────────────────────────────┘
```

**设计原则：**
- 通知在各业务操作中**同步触发**
- 通知类型通过常量枚举，方便扩展
- 举报处理和内容操作（隐藏/恢复）在同一事务中完成
- 管理员操作日志通过统一方法记录，各模块调用

---

## 3. 数据模型

### 3.1 新增数据表

#### 通知表 (notifications)

```sql
CREATE TABLE notifications (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,           -- 接收通知的用户
  type VARCHAR(32) NOT NULL,         -- 通知类型
  title VARCHAR(200) NOT NULL,       -- 通知标题
  content TEXT,                      -- 通知内容
  resource_type VARCHAR(32),         -- 关联资源类型（article/skill/mcp_server/comment）
  resource_id BIGINT,                -- 关联资源 ID
  actor_id BIGINT,                   -- 触发通知的用户
  is_read BOOLEAN DEFAULT FALSE,     -- 是否已读
  created_at DATETIME NOT NULL,
  INDEX idx_user_read (user_id, is_read),
  INDEX idx_created_at (created_at)
);
```

**通知类型枚举：**
```go
const (
    NotifTypeCommentArticle      = "comment_article"       // 文章收到评论
    NotifTypeReplyComment        = "reply_comment"         // 评论收到回复
    NotifTypeLikeArticle         = "like_article"          // 文章被点赞
    NotifTypeLikeSkill           = "like_skill"            // Skill 被点赞
    NotifTypeLikeMcpServer       = "like_mcp_server"       // MCP Server 被点赞
    NotifTypeLikeComment         = "like_comment"          // 文章评论被点赞
    NotifTypeLikeResourceComment = "like_resource_comment" // 资源评论被点赞
    NotifTypeResourceApproved    = "resource_approved"     // 资源审核通过
    NotifTypeResourceRejected    = "resource_rejected"     // 资源审核拒绝
    NotifTypeReportResolved      = "report_resolved"       // 举报已处理
    NotifTypeAnnouncement        = "announcement"          // 系统公告
)
```

#### 举报表 (reports)

```sql
CREATE TABLE reports (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  reporter_id BIGINT NOT NULL,       -- 举报人
  target_type VARCHAR(32) NOT NULL,  -- 举报对象类型（article/skill/mcp_server/comment）
  target_id BIGINT NOT NULL,         -- 举报对象 ID
  reason VARCHAR(32) NOT NULL,       -- 举报原因（spam/abuse/copyright/other）
  description TEXT,                  -- 详细说明
  status VARCHAR(16) DEFAULT 'pending', -- pending/resolved/dismissed
  handler_id BIGINT,                 -- 处理管理员
  handle_result TEXT,                -- 处理结果说明
  created_at DATETIME NOT NULL,
  resolved_at DATETIME,              -- 处理时间
  INDEX idx_status (status),
  INDEX idx_target (target_type, target_id)
);
```

**举报原因枚举：**
```go
const (
    ReportReasonSpam     = "spam"      // 垃圾广告
    ReportReasonAbuse    = "abuse"     // 辱骂攻击
    ReportReasonCopyright = "copyright" // 侵权
    ReportReasonOther    = "other"     // 其他
)
```

**举报状态枚举：**
```go
const (
    ReportStatusPending   = "pending"   // 待处理
    ReportStatusResolved  = "resolved"  // 已处理
    ReportStatusDismissed = "dismissed" // 已驳回
)
```

#### 管理员操作日志表 (admin_logs)

```sql
CREATE TABLE admin_logs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  admin_id BIGINT NOT NULL,          -- 操作管理员
  action VARCHAR(32) NOT NULL,       -- 操作类型
  target_type VARCHAR(32),           -- 操作对象类型
  target_id BIGINT,                  -- 操作对象 ID
  detail TEXT,                       -- 操作详情（JSON 格式）
  created_at DATETIME NOT NULL,
  INDEX idx_admin (admin_id),
  INDEX idx_action (action),
  INDEX idx_created_at (created_at)
);
```

**操作类型枚举：**
```go
const (
    AdminLogActionApproveResource = "approve_resource" // 审核通过资源
    AdminLogActionRejectResource  = "reject_resource"  // 审核拒绝资源
    AdminLogActionHideContent     = "hide_content"     // 隐藏内容
    AdminLogActionUnhideContent   = "unhide_content"   // 恢复内容
    AdminLogActionCreateTag       = "create_tag"       // 创建标签
    AdminLogActionUpdateTag       = "update_tag"       // 更新标签
    AdminLogActionCreateAnnouncement = "create_announcement" // 创建公告
    AdminLogActionResolveReport   = "resolve_report"   // 处理举报
)
```

#### 系统公告表 (announcements)

```sql
CREATE TABLE announcements (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  title VARCHAR(200) NOT NULL,
  content TEXT NOT NULL,
  admin_id BIGINT NOT NULL,          -- 发布管理员
  created_at DATETIME NOT NULL,
  INDEX idx_created_at (created_at)
);
```

### 3.2 现有模型修改

#### 用户角色机制

User 模型增加 Role 字段：
```go
type UserRole string

const (
    UserRoleUser  UserRole = "user"
    UserRoleAdmin UserRole = "admin"
)

type User struct {
    // ... 现有字段
    Role UserRole `gorm:"size:16;not null;default:user"` // 新增：用户角色
}
```

**初始管理员：** 通过环境变量 `ADMIN_EMAIL` 配置，启动时自动提升为管理员。

#### 内容隐藏字段

以下模型增加 Hidden 字段：
```go
// Article 模型
type Article struct {
    // ... 现有字段
    Hidden bool `gorm:"not null;default:false"` // 新增：是否隐藏
}

// Comment 模型
type Comment struct {
    // ... 现有字段
    Hidden bool `gorm:"not null;default:false"` // 新增：是否隐藏
}

// Skill 模型
type Skill struct {
    // ... 现有字段
    Hidden bool `gorm:"not null;default:false"` // 新增：是否隐藏
}

// McpServer 模型
type McpServer struct {
    // ... 现有字段
    Hidden bool `gorm:"not null;default:false"` // 新增：是否隐藏
}
```

**隐藏内容的查询规则：**
- 普通用户查询：`Where("hidden = ?", false)`
- 作者查询自己的内容：`Where("author_id = ? OR hidden = ?", userID, false)`
- 隐藏内容不出现在列表、搜索、排行中
- 隐藏内容不出现在热门排行 Redis ZSet 中
- 作者个人主页仍可见

---

## 4. API 设计

### 4.1 用户通知接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/notifications` | 获取通知列表（分页，支持类型筛选） |
| GET | `/api/v1/notifications/unread-count` | 获取未读通知数量 |
| PUT | `/api/v1/notifications/:id/read` | 标记单条通知已读 |
| PUT | `/api/v1/notifications/read` | 标记全部已读 |

**通知列表参数：**
- `type` — 可选，筛选通知类型
- `page`, `page_size` — 分页

**返回结构：**
```json
{
  "list": [
    {
      "id": 1,
      "type": "comment_article",
      "title": "有人评论了你的文章",
      "content": "张三评论了《Go并发编程》",
      "resource_type": "article",
      "resource_id": 123,
      "actor": {"id": 2, "nickname": "张三", "avatar_url": "..."},
      "is_read": false,
      "created_at": "2026-08-25T10:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 20
}
```

### 4.2 举报接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/reports` | 提交举报 |
| GET | `/api/v1/reports` | 查看我的举报列表 |

**提交举报请求体：**
```json
{
  "target_type": "article",
  "target_id": 123,
  "reason": "spam",
  "description": "这是广告内容"
}
```

**举报原因枚举：** `spam`（垃圾广告）、`abuse`（辱骂攻击）、`copyright`（侵权）、`other`（其他）

### 4.3 管理员举报处理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/reports` | 举报列表（分页，状态筛选） |
| PUT | `/api/v1/admin/reports/:id/resolve` | 处理举报 |

**处理举报请求体：**
```json
{
  "action": "hide",
  "result": "已隐藏违规内容"
}
```

**action 枚举：**
- `hide` — 隐藏内容
- `unhide` — 恢复内容
- `dismiss` — 驳回举报（不处理内容）

**举报处理通知规则：**

| action | 通知举报人 | 通知内容作者 |
|--------|-----------|-------------|
| `hide` | ✅ "你的举报已处理，违规内容已被隐藏" | ✅ "你的内容因违规被隐藏" |
| `unhide` | ❌ | ✅ "你的内容已恢复" |
| `dismiss` | ✅ "你的举报已驳回，内容未违规" | ❌ |

### 4.4 管理员内容管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | `/api/v1/admin/articles/:id/hide` | 隐藏文章 |
| PUT | `/api/v1/admin/articles/:id/unhide` | 恢复文章 |
| PUT | `/api/v1/admin/skills/:id/hide` | 隐藏 Skill |
| PUT | `/api/v1/admin/skills/:id/unhide` | 恢复 Skill |
| PUT | `/api/v1/admin/mcp-servers/:id/hide` | 隐藏 MCP Server |
| PUT | `/api/v1/admin/mcp-servers/:id/unhide` | 恢复 MCP Server |

### 4.5 管理员公告接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/admin/announcements` | 发布公告 |
| GET | `/api/v1/admin/announcements` | 公告列表 |

### 4.6 管理员数据看板接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/dashboard` | 数据概览 |
| GET | `/api/v1/admin/logs` | 操作日志列表（分页，操作类型筛选） |

**数据概览返回：**
```json
{
  "user_count": 1000,
  "article_count": 500,
  "skill_count": 50,
  "mcp_server_count": 30,
  "pending_reports": 5,
  "pending_reviews": 3
}
```

---

## 5. 通知触发点设计

通知在各业务操作中**同步触发**，通过 `NotificationService.Create()` 方法写入。

### 5.1 评论通知

| 触发场景 | 通知接收人 | 通知类型 |
|----------|-----------|----------|
| 用户评论文章 | 文章作者 | `comment_article` |
| 用户回复评论 | 被回复评论的作者 | `reply_comment` |

**触发位置：** `CommentService.Create()` 和 `ResourceCommentService.Create()`

**去重规则：** 自己评论自己的文章/回复自己的评论，不发送通知。

### 5.2 点赞通知

| 触发场景 | 通知接收人 | 通知类型 |
|----------|-----------|----------|
| 用户点赞文章 | 文章作者 | `like_article` |
| 用户点赞 Skill | Skill 作者 | `like_skill` |
| 用户点赞 MCP Server | MCP Server 作者 | `like_mcp_server` |
| 用户点赞文章评论 | 评论作者 | `like_comment` |
| 用户点赞资源评论 | 评论作者 | `like_resource_comment` |

**触发位置：** 各 `ToggleLike()` 方法，仅在点赞时触发（取消点赞不通知）。

**去重规则：** 自己点赞自己的内容，不发送通知。

### 5.3 审核结果通知

| 触发场景 | 通知接收人 | 通知类型 |
|----------|-----------|----------|
| 资源审核通过 | 资源作者 | `resource_approved` |
| 资源审核拒绝 | 资源作者 | `resource_rejected` |

**触发位置：** `SkillService.Review()` 和 `McpServerService.Review()`（P3 已有审核逻辑，需补充通知）

### 5.4 举报处理通知

| 触发场景 | 通知接收人 | 通知类型 |
|----------|-----------|----------|
| 举报被处理 | 举报人 + 内容作者（根据 action） | `report_resolved` |

**触发位置：** 管理员处理举报时

**通知规则：**

| action | 通知举报人 | 通知内容作者 |
|--------|-----------|-------------|
| `hide` | ✅ "你的举报已处理，违规内容已被隐藏" | ✅ "你的内容因违规被隐藏" |
| `unhide` | ❌ | ✅ "你的内容已恢复" |
| `dismiss` | ✅ "你的举报已驳回，内容未违规" | ❌ |

### 5.5 系统公告通知

| 触发场景 | 通知接收人 | 通知类型 |
|----------|-----------|----------|
| 管理员发布公告 | 所有用户 | `announcement` |

**触发位置：** 发布公告时，查询所有用户 ID，批量插入通知记录（写时扩散）。

---

## 6. 通知服务设计

```go
type NotificationService struct {
    repo    *repo.NotificationRepo
    users   *repo.UserRepo
}

// Create 创建单条通知
func (s *NotificationService) Create(ctx context.Context, userID uint, notifType string, title, content string, resourceType string, resourceID uint, actorID uint)

// CreateBatch 批量创建通知（用于公告）
func (s *NotificationService) CreateBatch(ctx context.Context, notifType string, title, content string, actorID uint) error

// List 获取用户通知列表
func (s *NotificationService) List(ctx context.Context, userID uint, notifType string, page, pageSize int) (*NotificationListResult, error)

// UnreadCount 获取未读数量
func (s *NotificationService) UnreadCount(ctx context.Context, userID uint) (int64, error)

// MarkRead 标记已读
func (s *NotificationService) MarkRead(ctx context.Context, userID, notifID uint) error

// MarkAllRead 全部标记已读
func (s *NotificationService) MarkAllRead(ctx context.Context, userID uint) error
```

---

## 7. 管理员操作日志服务设计

```go
type AdminLogService struct {
    repo *repo.AdminLogRepo
}

// Log 记录操作日志
func (s *AdminLogService) Log(ctx context.Context, adminID uint, action string, targetType string, targetID uint, detail interface{}) error

// List 获取操作日志列表
func (s *AdminLogService) List(ctx context.Context, action string, page, pageSize int) (*AdminLogListResult, error)
```

---

## 8. 管理员中间件

```go
func AdminMiddleware(cfg *platform.Config) gin.HandlerFunc {
    // 1. 校验 JWT（复用现有 AuthMiddleware）
    // 2. 从 context 获取 userID
    // 3. 查询用户 Role
    // 4. 如果 Role != admin，返回 403
}
```

**初始管理员配置：**
- 环境变量：`ADMIN_EMAIL=admin@example.com`
- 启动时检查该邮箱用户，自动提升为 admin 角色

---

## 9. 前端适配（待后续设计）

- 通知中心页面
- 举报按钮（文章/评论/资源详情页）
- 管理后台：举报处理页面
- 管理后台：数据看板
- 管理后台：操作日志页面

---

## 10. 实现计划（待后续编写）

1. 数据模型迁移
2. 管理员角色机制
3. 通知模块（model → repo → service → handler）
4. 举报模块（model → repo → service → handler）
5. 管理员操作日志模块
6. 管理员内容管理接口
7. 管理员数据看板接口
8. 业务模块集成通知触发点
9. 前端适配
10. 测试与文档

---

## 11. 后续优化方向

- **异步通知：** 引入 Redis Stream 异步队列，提升接口响应速度
- **通知聚合：** 前端展示时按类型+目标聚合（如"张三等 10 人赞了你的文章"）
- **通知偏好：** 用户可配置接收哪些类型的通知
- **举报申诉：** 作者可对隐藏内容发起申诉

---

**文档版本：** v1.0  
**最后更新：** 2026-08-25
