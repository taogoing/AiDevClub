# P3 AI 资源（Skills Hub + MCP Hub）实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现 AI 资源——Skill 和 MCP Server 的发布/浏览/下载/互动，ZIP 资源包上传，审核状态机，资源评论，以及错误码常量化重构。

**架构：** 沿用扁平技术分层（`internal/handler` → `service` → `repo` / `model`）。Skill 和 MCP Server 各自独立模型，共用标签体系（`skill_tags` / `mcp_server_tags`）和评论表（`resource_comments`）。互动用独立小表 + 计数冗余列（事务内同步增减）。审核状态机：draft → pending_review → published / rejected → archived。ZIP 存储本地磁盘 + `/static` 静态服务。

**技术栈：** Go / Gin / GORM / MySQL 8 / Redis（已有依赖，无需新增）。测试沿用真实 MySQL/Redis 按进程隔离（`internal/testutil`）。

规格：`docs/superpowers/specs/2026-08-24-p3-ai-resources-design.md`

---

## 文件结构

**新增模型**（`internal/model/`）
- `skill.go` — `Skill`、`SkillTag`、`ResourceStatus` 常量
- `mcp_server.go` — `McpServer`、`McpServerTag`
- `resource_interaction.go` — `SkillLike`、`SkillFavorite`、`McpServerLike`、`McpServerFavorite`、`ResourceComment`、`ResourceCommentLike`

**新增 repo**（`internal/repo/`）
- `skill.go` — `SkillRepo`（Create / FindByID / Update / Delete / List / Count / IncrViews / IncrDownloads / IncrCount / FindSkillTags / SetSkillTags / TagsForSkills）
- `mcp_server.go` — `McpServerRepo`（同 SkillRepo 模式）
- `resource_interaction.go` — `ResourceInteractionRepo`（Toggle 资源点赞/收藏 + 资源评论 CRUD）

**新增 service**（`internal/service/`）
- `skill.go` — `SkillService`（Create/Update/Delete/List/Get/Submit/Withdraw/Archive/Upload/Download/ToggleLike/ToggleFavorite）
- `mcp_server.go` — `McpServerService`（同 SkillService 模式）
- `resource_comment.go` — `ResourceCommentService`（Create/List/Delete/ToggleLike）
- `dto.go` — 追加资源相关 DTO

**新增 handler**（`internal/handler/`）
- `skill.go` — `SkillHandler`
- `mcp_server.go` — `McpServerHandler`
- `resource_comment.go` — `ResourceCommentHandler`

**新增 platform**
- `errors.go` — 错误码常量定义

**修改**
- `internal/platform/config.go` — 新增 ZIP 存储配置
- `internal/testutil/testutil.go` — `NewTestDB` 迁移全部新模型
- `internal/handler/auth.go` — 错误码常量化
- `internal/handler/user.go` — 错误码常量化
- `internal/handler/article.go` — 错误码常量化
- `internal/handler/comment.go` — 错误码常量化
- `internal/handler/category.go` — 错误码常量化
- `internal/handler/tag.go` — 错误码常量化
- `internal/service/*.go` — BizError 改用常量
- `cmd/server/main.go` — 装配新 repo/service/handler、路由、迁移、静态目录

**测试文件**：每个包 `*_test.go` 与源码同目录。

---

## 任务 1：错误码常量化（重构 P1/P2）

**文件：**
- 创建：`internal/platform/errors.go`
- 修改：`internal/handler/auth.go`、`internal/handler/user.go`、`internal/handler/article.go`、`internal/handler/comment.go`、`internal/handler/category.go`、`internal/handler/tag.go`
- 修改：`internal/service/auth.go`、`internal/service/user.go`、`internal/service/article.go`、`internal/service/comment.go`
- 测试：`internal/platform/errors_test.go`

- [ ] **步骤 1：定义错误码常量**

`internal/platform/errors.go`：
```go
package platform

import "net/http"

// 错误码常量
const (
	// 通用错误
	CodeParamError    = 40001 // 参数错误
	CodeBizError      = 40002 // 业务参数校验失败
	CodeStateError    = 40003 // 状态不允许当前操作
	CodeUnauthorized  = 40101 // 未认证
	CodeForbidden     = 40301 // 无权限

	// 资源不存在
	CodeUserNotFound         = 40401 // 用户不存在
	CodeArticleNotFound      = 40402 // 文章不存在
	CodeCommentNotFound      = 40403 // 评论不存在
	CodeCategoryNotFound     = 40404 // 分类不存在
	CodeTagNotFound          = 40405 // 标签不存在
	CodeSkillNotFound        = 40406 // Skill 不存在
	CodeMcpServerNotFound    = 40407 // MCP Server 不存在
	CodeResCommentNotFound   = 40408 // 资源评论不存在

	// 服务器错误
	CodeInternalError = 50000 // 服务器内部错误
)

// 预定义业务错误
var (
	ErrBadRequest   = NewBizError(http.StatusBadRequest, CodeParamError, "参数错误")
	ErrUnauthorized = NewBizError(http.StatusUnauthorized, CodeUnauthorized, "未认证")
	ErrForbidden    = NewBizError(http.StatusForbidden, CodeForbidden, "无权限")
)
```

- [ ] **步骤 2：重构 handler 层硬编码**

逐个修改 handler 文件，将 `40001` 替换为 `platform.CodeParamError`：

`internal/handler/auth.go`：
```go
// 修改前
platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
// 修改后
platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
```

同样修改 `user.go`、`article.go`、`comment.go`、`category.go`、`tag.go`。

- [ ] **步骤 3：重构 service 层 BizError**

`internal/service/article.go`：
```go
// 修改前
ErrArticleNotFound = platform.NewBizError(http.StatusNotFound, 40402, "文章不存在或不可见")
// 修改后
ErrArticleNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeArticleNotFound, "文章不存在或不可见")
```

同样修改其他 service 文件中的 BizError 定义。

- [ ] **步骤 4：运行测试验证重构正确**

运行：`go test ./...`
预期：所有测试 PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/platform/errors.go internal/handler/ internal/service/
git commit -m "refactor: 错误码常量化（P1/P2 handler/service 层）"
```

---

## 任务 2：数据模型 + testutil 扩展 + 配置

**文件：**
- 创建：`internal/model/skill.go`、`internal/model/mcp_server.go`、`internal/model/resource_interaction.go`
- 修改：`internal/platform/config.go`、`internal/testutil/testutil.go`
- 测试：`internal/model/resource_model_test.go`

- [ ] **步骤 1：创建资源模型**

`internal/model/skill.go`：
```go
package model

import (
	"time"

	"gorm.io/gorm"
)

type ResourceStatus string

const (
	ResourceStatusDraft          ResourceStatus = "draft"
	ResourceStatusPendingReview  ResourceStatus = "pending_review"
	ResourceStatusPublished      ResourceStatus = "published"
	ResourceStatusRejected       ResourceStatus = "rejected"
	ResourceStatusArchived       ResourceStatus = "archived"
)

type Skill struct {
	ID             uint           `gorm:"primaryKey"`
	AuthorID       uint           `gorm:"not null;index"`
	Name           string         `gorm:"size:100;not null"`
	Description    string         `gorm:"size:500"`
	RepoURL        string         `gorm:"size:255"`
	ZipURL         string         `gorm:"size:255"`
	ZipFilename    string         `gorm:"size:255"`
	FileSize       int64          `gorm:"not null;default:0"`
	Status         ResourceStatus `gorm:"size:16;not null;default:draft;index"`
	Views          int            `gorm:"not null;default:0"`
	Downloads      int            `gorm:"not null;default:0"`
	LikesCount     int            `gorm:"not null;default:0"`
	FavoritesCount int            `gorm:"not null;default:0"`
	CommentsCount  int            `gorm:"not null;default:0"`
	Pinned         bool           `gorm:"not null;default:false"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Author *User `gorm:"foreignKey:AuthorID"`
}

type SkillTag struct {
	ID        uint `gorm:"primaryKey"`
	SkillID   uint `gorm:"uniqueIndex:uniq_skill_tag;not null"`
	TagID     uint `gorm:"uniqueIndex:uniq_skill_tag;not null"`
	CreatedAt time.Time
}
```

`internal/model/mcp_server.go`：
```go
package model

import (
	"time"

	"gorm.io/gorm"
)

type McpServer struct {
	ID             uint           `gorm:"primaryKey"`
	AuthorID       uint           `gorm:"not null;index"`
	Name           string         `gorm:"size:100;not null"`
	Description    string         `gorm:"size:500"`
	RepoURL        string         `gorm:"size:255"`
	ToolsJSON      string         `gorm:"type:json"`
	Readme         string         `gorm:"type:mediumtext"`
	ZipURL         string         `gorm:"size:255"`
	ZipFilename    string         `gorm:"size:255"`
	FileSize       int64          `gorm:"not null;default:0"`
	Status         ResourceStatus `gorm:"size:16;not null;default:draft;index"`
	Views          int            `gorm:"not null;default:0"`
	Downloads      int            `gorm:"not null;default:0"`
	LikesCount     int            `gorm:"not null;default:0"`
	FavoritesCount int            `gorm:"not null;default:0"`
	CommentsCount  int            `gorm:"not null;default:0"`
	Pinned         bool           `gorm:"not null;default:false"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Author *User `gorm:"foreignKey:AuthorID"`
}

type McpServerTag struct {
	ID          uint `gorm:"primaryKey"`
	McpServerID uint `gorm:"uniqueIndex:uniq_mcp_server_tag;not null"`
	TagID       uint `gorm:"uniqueIndex:uniq_mcp_server_tag;not null"`
	CreatedAt   time.Time
}
```

`internal/model/resource_interaction.go`：
```go
package model

import (
	"time"

	"gorm.io/gorm"
)

type SkillLike struct {
	ID        uint `gorm:"primaryKey"`
	SkillID   uint `gorm:"uniqueIndex:uniq_skill_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_skill_like;not null"`
	CreatedAt time.Time
}

type SkillFavorite struct {
	ID        uint `gorm:"primaryKey"`
	SkillID   uint `gorm:"uniqueIndex:uniq_skill_fav;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_skill_fav;not null"`
	CreatedAt time.Time
}

type McpServerLike struct {
	ID          uint `gorm:"primaryKey"`
	McpServerID uint `gorm:"uniqueIndex:uniq_mcp_server_like;not null"`
	UserID      uint `gorm:"uniqueIndex:uniq_mcp_server_like;not null"`
	CreatedAt   time.Time
}

type McpServerFavorite struct {
	ID          uint `gorm:"primaryKey"`
	McpServerID uint `gorm:"uniqueIndex:uniq_mcp_server_fav;not null"`
	UserID      uint `gorm:"uniqueIndex:uniq_mcp_server_fav;not null"`
	CreatedAt   time.Time
}

type ResourceComment struct {
	ID           uint           `gorm:"primaryKey"`
	ResourceType string         `gorm:"size:16;not null;index:idx_resource"`
	ResourceID   uint           `gorm:"not null;index:idx_resource"`
	AuthorID     uint           `gorm:"not null;index"`
	ParentID     *uint          `gorm:"index"`
	Content      string         `gorm:"type:text;not null"`
	LikesCount   int            `gorm:"not null;default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type ResourceCommentLike struct {
	ID        uint `gorm:"primaryKey"`
	CommentID uint `gorm:"uniqueIndex:uniq_res_comment_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_res_comment_like;not null"`
	CreatedAt time.Time
}
```

- [ ] **步骤 2：扩展配置**

`internal/platform/config.go` 追加：
```go
type Config struct {
	// ... 现有字段
	SkillZipDir         string
	McpServerZipDir     string
	MaxResourceZipBytes int64
}

// LoadConfig 中追加
v.SetDefault("skill_zip.dir", "storage/skills")
v.SetDefault("mcp_server_zip.dir", "storage/mcp_servers")
v.SetDefault("resource_zip.max_bytes", int64(50<<20)) // 50MB

cfg.SkillZipDir = v.GetString("skill_zip.dir")
cfg.McpServerZipDir = v.GetString("mcp_server_zip.dir")
cfg.MaxResourceZipBytes = v.GetInt64("resource_zip.max_bytes")
```

- [ ] **步骤 3：扩展 testutil.NewTestDB**

`internal/testutil/testutil.go`：
```go
var allModels = []interface{}{
	// 现有模型
	&model.User{}, &model.Category{}, &model.Tag{}, &model.Article{},
	&model.ArticleTag{}, &model.ArticleLike{}, &model.ArticleFavorite{},
	&model.Comment{}, &model.CommentLike{},
	// P3 新增
	&model.Skill{}, &model.SkillTag{},
	&model.McpServer{}, &model.McpServerTag{},
	&model.SkillLike{}, &model.SkillFavorite{},
	&model.McpServerLike{}, &model.McpServerFavorite{},
	&model.ResourceComment{}, &model.ResourceCommentLike{},
}
```

- [ ] **步骤 4：写测试验证模型迁移**

`internal/model/resource_model_test.go`：
```go
package model

import (
	"testing"

	"aidevclub/internal/testutil"
)

func TestNewTestDBMigratesResourceModels(t *testing.T) {
	db := testutil.NewTestDB(t)
	for _, m := range []interface{}{
		&Skill{}, &SkillTag{},
		&McpServer{}, &McpServerTag{},
		&SkillLike{}, &SkillFavorite{},
		&McpServerLike{}, &McpServerFavorite{},
		&ResourceComment{}, &ResourceCommentLike{},
	} {
		if !db.Migrator().HasTable(m) {
			t.Fatalf("table %T not migrated", m)
		}
	}
}
```

- [ ] **步骤 5：运行验证**

运行：`go test ./internal/model/ -v`
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add internal/model/skill.go internal/model/mcp_server.go internal/model/resource_interaction.go internal/model/resource_model_test.go internal/platform/config.go internal/testutil/testutil.go
git commit -m "feat: P3 数据模型（Skill/McpServer/资源互动/资源评论）+ 配置扩展"
```

---

## 任务 3：Skill Repo + 测试

**文件：**
- 创建：`internal/repo/skill.go`、`internal/repo/skill_test.go`

- [ ] **步骤 1：写失败测试**

`internal/repo/skill_test.go`：
```go
package repo

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestSkillRepoCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewSkillRepo(db)
	ctx := context.Background()

	s := &model.Skill{
		AuthorID: 1, Name: "test-skill", Description: "desc",
		Status: model.ResourceStatusDraft,
	}
	if err := r.Create(db, s); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByID(db, s.ID)
	if err != nil || got.Name != "test-skill" {
		t.Fatalf("FindByID = %v, %v", got, err)
	}
	// 标签关联
	if err := r.SetSkillTags(db, s.ID, []uint{1, 2}); err != nil {
		t.Fatal(err)
	}
	ids, err := r.FindSkillTags(db, s.ID)
	if err != nil || len(ids) != 2 {
		t.Fatalf("FindSkillTags = %v, %v", ids, err)
	}
	// 更新
	s.Name = "updated"
	if err := r.Update(db, s); err != nil {
		t.Fatal(err)
	}
	// 计数
	if err := r.IncrViews(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, s.ID, "downloads", 1); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, s.ID)
	if got.Views != 1 || got.Downloads != 1 {
		t.Fatalf("counts = views %d downloads %d", got.Views, got.Downloads)
	}
	// 软删除
	if err := r.Delete(db, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, s.ID); err == nil {
		t.Fatal("soft-deleted skill still found")
	}
}

func TestSkillRepoList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewSkillRepo(db)
	ctx := context.Background()
	now := time.Now()

	for i := 0; i < 3; i++ {
		s := &model.Skill{
			AuthorID: 1, Name: "skill", Status: model.ResourceStatusPublished,
			PublishedAt: &now,
		}
		_ = r.Create(db, s)
	}
	_ = r.SetSkillTags(db, 1, []uint{1})

	q := SkillQuery{Page: 1, PageSize: 2, Sort: "latest"}
	list, total, err := r.List(ctx, q)
	if err != nil || total != 3 || len(list) != 2 {
		t.Fatalf("list = %d total, %d len, err %v", total, len(list), err)
	}
	q.TagID = uintPtr(1)
	list, total, _ = r.List(ctx, q)
	if total != 1 {
		t.Fatalf("tag filter total = %d", total)
	}
}

func uintPtr(v uint) *uint { return &v }
```

- [ ] **步骤 2：运行验证失败**

运行：`go test ./internal/repo/ -run TestSkillRepo -v`
预期：FAIL（`NewSkillRepo` 未定义）

- [ ] **步骤 3：实现 SkillRepo**

`internal/repo/skill.go`：
```go
package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type SkillRepo struct{ db *gorm.DB }

func NewSkillRepo(db *gorm.DB) *SkillRepo { return &SkillRepo{db: db} }

func (r *SkillRepo) DB() *gorm.DB { return r.db }

func (r *SkillRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *SkillRepo) Create(db *gorm.DB, s *model.Skill) error {
	return r.exec(db).Create(s).Error
}

func (r *SkillRepo) FindByID(db *gorm.DB, id uint) (*model.Skill, error) {
	var s model.Skill
	if err := r.exec(db).Preload("Author").First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SkillRepo) Update(db *gorm.DB, s *model.Skill) error {
	return r.exec(db).Save(s).Error
}

func (r *SkillRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.Skill{}, id).Error
}

func (r *SkillRepo) FindSkillTags(db *gorm.DB, skillID uint) ([]uint, error) {
	var rows []model.SkillTag
	if err := r.exec(db).Where("skill_id = ?", skillID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TagID)
	}
	return ids, nil
}

func (r *SkillRepo) SetSkillTags(db *gorm.DB, skillID uint, tagIDs []uint) error {
	d := r.exec(db)
	if err := d.Where("skill_id = ?", skillID).Delete(&model.SkillTag{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.SkillTag, 0, len(tagIDs))
	for _, tid := range tagIDs {
		rows = append(rows, model.SkillTag{SkillID: skillID, TagID: tid})
	}
	return d.Create(&rows).Error
}

func (r *SkillRepo) IncrViews(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.Skill{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).Error
}

func (r *SkillRepo) IncrCount(db *gorm.DB, id uint, column string, delta int) error {
	return r.exec(db).Model(&model.Skill{}).
		Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).Error
}

type SkillQuery struct {
	Page, PageSize int
	TagID          *uint
	Keyword        string
	AuthorID       *uint
	Sort           string // latest | hot | downloads
}

func (r *SkillRepo) baseQuery(ctx context.Context, q SkillQuery) *gorm.DB {
	d := r.db.WithContext(ctx).Model(&model.Skill{}).
		Where("status = ?", model.ResourceStatusPublished)
	if q.AuthorID != nil {
		d = d.Where("author_id = ?", *q.AuthorID)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		d = d.Where("(name LIKE ? OR description LIKE ?)", kw, kw)
	}
	if q.TagID != nil {
		d = d.Where("id IN (SELECT skill_id FROM skill_tags WHERE tag_id = ?)", *q.TagID)
	}
	return d
}

func (r *SkillRepo) Count(ctx context.Context, q SkillQuery) (int64, error) {
	var total int64
	err := r.baseQuery(ctx, q).Count(&total).Error
	return total, err
}

func (r *SkillRepo) List(ctx context.Context, q SkillQuery) ([]model.Skill, int64, error) {
	d := r.baseQuery(ctx, q)
	switch q.Sort {
	case "hot":
		d = d.Order("(views + 3*likes_count + 5*favorites_count + 2*comments_count) desc, id desc")
	case "downloads":
		d = d.Order("downloads desc, id desc")
	default:
		d = d.Order("published_at desc, id desc")
	}
	var list []model.Skill
	if err := d.Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	total, err := r.Count(ctx, q)
	return list, total, err
}

func (r *SkillRepo) TagsForSkills(ctx context.Context, skillIDs []uint) (map[uint][]model.Tag, error) {
	res := map[uint][]model.Tag{}
	if len(skillIDs) == 0 {
		return res, nil
	}
	var rows []model.SkillTag
	if err := r.db.WithContext(ctx).Where("skill_id IN ?", skillIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	tagIDs := map[uint]bool{}
	for _, row := range rows {
		tagIDs[row.TagID] = true
	}
	ids := make([]uint, 0, len(tagIDs))
	for id := range tagIDs {
		ids = append(ids, id)
	}
	var tags []model.Tag
	if len(ids) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&tags).Error; err != nil {
			return nil, err
		}
	}
	byID := map[uint]model.Tag{}
	for _, t := range tags {
		byID[t.ID] = t
	}
	for _, row := range rows {
		res[row.SkillID] = append(res[row.SkillID], byID[row.TagID])
	}
	return res, nil
}
```

- [ ] **步骤 4：运行验证通过**

运行：`go test ./internal/repo/ -run TestSkillRepo -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/skill.go internal/repo/skill_test.go
git commit -m "feat: Skill Repo（CRUD/标签关联/列表查询/计数）"
```

---

## 任务 4：Skill Service + 测试（CRUD + 状态机）

**文件：**
- 创建：`internal/service/skill.go`、`internal/service/skill_test.go`
- 修改：`internal/service/dto.go`

- [ ] **步骤 1：定义 DTO**

`internal/service/dto.go` 追加：
```go
// Skill DTO
type SkillSummary struct {
	ID             uint        `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	RepoURL        string      `json:"repo_url"`
	Tags           []TagBrief  `json:"tags"`
	Author         AuthorBrief `json:"author"`
	Views          int         `json:"views"`
	Downloads      int         `json:"downloads"`
	LikesCount     int         `json:"likes_count"`
	FavoritesCount int         `json:"favorites_count"`
	CommentsCount  int         `json:"comments_count"`
	Status         string      `json:"status"`
	PublishedAt    *time.Time  `json:"published_at"`
}

type SkillListResult struct {
	List     []SkillSummary `json:"list"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type SkillDetail struct {
	SkillSummary
	ZipURL      string `json:"zip_url"`
	ZipFilename string `json:"zip_filename"`
	FileSize    int64  `json:"file_size"`
	Liked       bool   `json:"liked"`
	Favorited   bool   `json:"favorited"`
}

type CreateSkillInput struct {
	Name        string
	Description string
	RepoURL     string
	TagIDs      []uint
	TagNames    []string
}

type SkillListQuery struct {
	Page     int
	PageSize int
	TagID    *uint
	Keyword  string
	AuthorID *uint
	Sort     string
}
```

- [ ] **步骤 2：写失败测试**

`internal/service/skill_test.go`：
```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newSkillTestEnv(t *testing.T) (*SkillService, *model.User) {
	t.Helper()
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	u := &model.User{Email: "s@s.com", PasswordHash: "x", Nickname: "S", AvatarURL: "/x.png"}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}
	svc := NewSkillService(
		repo.NewSkillRepo(db),
		repo.NewTagRepo(db),
		repo.NewInteractionRepo(db),
		testutil.NewTestRedis(t),
		&platform.Config{DefaultPageSize: 20, MaxPageSize: 50, HotCacheTTL: 60e9},
	)
	return svc, u
}

func TestSkillCreate(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()

	s, err := svc.Create(ctx, u.ID, CreateSkillInput{
		Name: "test-skill", Description: "desc",
		TagNames: []string{"claude"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == 0 || s.Status != model.ResourceStatusDraft {
		t.Fatalf("skill = %+v", s)
	}
}

func TestSkillStatusFlow(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()

	s, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "s", Description: "d"})

	// draft → pending_review
	if err := svc.Submit(ctx, u.ID, s.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(ctx, u.ID, s.ID)
	if got.Status != string(model.ResourceStatusPendingReview) {
		t.Fatalf("status = %s", got.Status)
	}

	// pending_review → draft (撤回)
	if err := svc.Withdraw(ctx, u.ID, s.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.Get(ctx, u.ID, s.ID)
	if got.Status != string(model.ResourceStatusDraft) {
		t.Fatalf("status = %s", got.Status)
	}

	// 非作者不可操作
	if err := svc.Submit(ctx, u.ID+999, s.ID); err == nil {
		t.Fatal("non-author submit allowed")
	}
}

func TestSkillVisibility(t *testing.T) {
	svc, u := newSkillTestEnv(t)
	ctx := context.Background()

	draft, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "draft", Description: "d"})
	_ = svc.Submit(ctx, u.ID, draft.ID)
	// 手动设置为 published 测试可见性
	svc.skills.DB().Model(&model.Skill{}).Where("id = ?", draft.ID).Update("status", model.ResourceStatusPublished)

	// 游客可见 published
	_, err := svc.Get(ctx, 0, draft.ID)
	if err != nil {
		t.Fatalf("published not visible to guest: %v", err)
	}

	// 草稿仅作者可见
	draftOnly, _ := svc.Create(ctx, u.ID, CreateSkillInput{Name: "draft2", Description: "d"})
	if _, err := svc.Get(ctx, 0, draftOnly.ID); err == nil {
		t.Fatal("draft visible to guest")
	}
	if _, err := svc.Get(ctx, u.ID+999, draftOnly.ID); err == nil {
		t.Fatal("draft visible to other user")
	}
}
```

- [ ] **步骤 3：运行验证失败**

运行：`go test ./internal/service/ -run TestSkill -v`
预期：FAIL（`NewSkillService` 未定义）

- [ ] **步骤 4：实现 SkillService**

`internal/service/skill.go`：
```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var (
	ErrSkillNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeSkillNotFound, "Skill 不存在或不可见")
	ErrStateError    = platform.NewBizError(http.StatusBadRequest, platform.CodeStateError, "当前状态不允许此操作")
)

type SkillService struct {
	skills *repo.SkillRepo
	tags   *repo.TagRepo
	inter  *repo.InteractionRepo
	rdb    *redis.Client
	cfg    *platform.Config
}

func NewSkillService(skills *repo.SkillRepo, tags *repo.TagRepo, inter *repo.InteractionRepo, rdb *redis.Client, cfg *platform.Config) *SkillService {
	return &SkillService{skills: skills, tags: tags, inter: inter, rdb: rdb, cfg: cfg}
}

func (s *SkillService) ZipDir() string        { return s.cfg.SkillZipDir }
func (s *SkillService) MaxZipBytes() int64    { return s.cfg.MaxResourceZipBytes }

func (s *SkillService) Create(ctx context.Context, userID uint, in CreateSkillInput) (*model.Skill, error) {
	if in.Name == "" {
		return nil, ErrBadParam
	}
	if len(in.Name) > 100 {
		return nil, ErrBadParam
	}

	skill := &model.Skill{
		AuthorID:    userID,
		Name:        in.Name,
		Description: in.Description,
		RepoURL:     in.RepoURL,
		Status:      model.ResourceStatusDraft,
	}

	err := s.skills.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.skills.Create(tx, skill); err != nil {
			return err
		}
		tagIDs, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		if len(tagIDs) > 0 {
			if err := s.skills.SetSkillTags(tx, skill.ID, tagIDs); err != nil {
				return err
			}
			for _, id := range tagIDs {
				if err := s.tags.IncrUsage(tx, id, 1); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *SkillService) Update(ctx context.Context, userID, skillID uint, in CreateSkillInput) (*model.Skill, error) {
	if in.Name == "" {
		return nil, ErrBadParam
	}
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	if skill.AuthorID != userID {
		return nil, ErrForbidden
	}
	// pending_review 状态不可编辑
	if skill.Status == model.ResourceStatusPendingReview {
		return nil, ErrStateError
	}

	skill.Name = in.Name
	skill.Description = in.Description
	skill.RepoURL = in.RepoURL

	err = s.skills.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.skills.Update(tx, skill); err != nil {
			return err
		}
		oldTags, _ := s.skills.FindSkillTags(tx, skillID)
		newTags, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		// diff
		oldSet := map[uint]bool{}
		for _, id := range oldTags {
			oldSet[id] = true
		}
		newSet := map[uint]bool{}
		for _, id := range newTags {
			newSet[id] = true
		}
		for _, id := range oldTags {
			if !newSet[id] {
				_ = s.tags.IncrUsage(tx, id, -1)
			}
		}
		for _, id := range newTags {
			if !oldSet[id] {
				_ = s.tags.IncrUsage(tx, id, 1)
			}
		}
		return s.skills.SetSkillTags(tx, skillID, newTags)
	})
	if err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *SkillService) Delete(ctx context.Context, userID, skillID uint) error {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	if skill.AuthorID != userID {
		return ErrForbidden
	}
	return s.skills.DB().Transaction(func(tx *gorm.DB) error {
		tagIDs, _ := s.skills.FindSkillTags(tx, skillID)
		if err := s.skills.Delete(tx, skillID); err != nil {
			return err
		}
		for _, id := range tagIDs {
			_ = s.tags.IncrUsage(tx, id, -1)
		}
		return s.skills.SetSkillTags(tx, skillID, nil)
	})
}

func (s *SkillService) Submit(ctx context.Context, userID, skillID uint) error {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	if skill.AuthorID != userID {
		return ErrForbidden
	}
	// 只有 draft/rejected/archived 可以提交
	switch skill.Status {
	case model.ResourceStatusDraft, model.ResourceStatusRejected, model.ResourceStatusArchived:
	default:
		return ErrStateError
	}
	skill.Status = model.ResourceStatusPendingReview
	return s.skills.Update(nil, skill)
}

func (s *SkillService) Withdraw(ctx context.Context, userID, skillID uint) error {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	if skill.AuthorID != userID {
		return ErrForbidden
	}
	if skill.Status != model.ResourceStatusPendingReview {
		return ErrStateError
	}
	skill.Status = model.ResourceStatusDraft
	return s.skills.Update(nil, skill)
}

func (s *SkillService) Archive(ctx context.Context, userID, skillID uint) error {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	if skill.AuthorID != userID {
		return ErrForbidden
	}
	if skill.Status != model.ResourceStatusPublished {
		return ErrStateError
	}
	skill.Status = model.ResourceStatusArchived
	return s.skills.Update(nil, skill)
}

func (s *SkillService) Get(ctx context.Context, userID, skillID uint) (*SkillDetail, error) {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	// 可见性检查
	if !s.canView(skill, userID) {
		return nil, ErrSkillNotFound
	}
	// 浏览量 +1（仅 published）
	if skill.Status == model.ResourceStatusPublished {
		_ = s.skills.IncrViews(ctx, skillID)
		skill.Views++
	}

	tagMap, _ := s.skills.TagsForSkills(ctx, []uint{skillID})
	sm := s.summaryOf(skill, tagMap[skillID])
	d := &SkillDetail{
		SkillSummary: sm,
		ZipURL:       skill.ZipURL,
		ZipFilename:  skill.ZipFilename,
		FileSize:     skill.FileSize,
	}
	if userID > 0 {
		d.Liked, _ = s.inter.SkillLiked(nil, userID, skillID)
		d.Favorited, _ = s.inter.SkillFavorited(nil, userID, skillID)
	}
	return d, nil
}

func (s *SkillService) List(ctx context.Context, q SkillListQuery) (*SkillListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = s.cfg.DefaultPageSize
	}
	if q.PageSize > s.cfg.MaxPageSize {
		q.PageSize = s.cfg.MaxPageSize
	}
	switch q.Sort {
	case "hot", "downloads":
	default:
		q.Sort = "latest"
	}

	rq := repo.SkillQuery{
		Page: q.Page, PageSize: q.PageSize,
		TagID: q.TagID, Keyword: q.Keyword, AuthorID: q.AuthorID, Sort: q.Sort,
	}

	var key string
	if q.Sort == "hot" && q.TagID == nil && q.AuthorID == nil && q.Keyword == "" {
		key = fmt.Sprintf("hot:skills:%d:%d", q.Page, q.PageSize)
		if v, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var res SkillListResult
			if json.Unmarshal(v, &res) == nil {
				return &res, nil
			}
		}
	}

	list, total, err := s.skills.List(ctx, rq)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(list))
	for _, s := range list {
		ids = append(ids, s.ID)
	}
	tagMap, _ := s.skills.TagsForSkills(ctx, ids)

	out := &SkillListResult{
		List: make([]SkillSummary, 0, len(list)),
		Total: total, Page: q.Page, PageSize: q.PageSize,
	}
	for _, sk := range list {
		out.List = append(out.List, s.summaryOf(&sk, tagMap[sk.ID]))
	}

	if key != "" {
		if b, err := json.Marshal(out); err == nil {
			_ = s.rdb.Set(ctx, key, b, s.cfg.HotCacheTTL).Err()
		}
	}
	return out, nil
}

func (s *SkillService) summaryOf(sk *model.Skill, tags []model.Tag) SkillSummary {
	sm := SkillSummary{
		ID: sk.ID, Name: sk.Name, Description: sk.Description,
		RepoURL: sk.RepoURL, Tags: []TagBrief{},
		Views: sk.Views, Downloads: sk.Downloads,
		LikesCount: sk.LikesCount, FavoritesCount: sk.FavoritesCount,
		CommentsCount: sk.CommentsCount, Status: string(sk.Status),
		PublishedAt: sk.PublishedAt,
		Author: AuthorBrief{ID: sk.AuthorID},
	}
	if sk.Author != nil {
		sm.Author = AuthorBrief{ID: sk.Author.ID, Nickname: sk.Author.Nickname, AvatarURL: sk.Author.AvatarURL}
	}
	for _, t := range tags {
		sm.Tags = append(sm.Tags, TagBrief{ID: t.ID, Name: t.Name})
	}
	return sm
}

func (s *SkillService) canView(skill *model.Skill, userID uint) bool {
	// published 所有人可见
	if skill.Status == model.ResourceStatusPublished {
		return true
	}
	// 作者本人可见
	if skill.AuthorID == userID {
		return true
	}
	// TODO: 管理员可见（P6 实现）
	return false
}

// ResolveTagSet 复用 article service 的逻辑，这里简化实现
func (s *SkillService) ResolveTagSet(ctx context.Context, tx *gorm.DB, tagIDs []uint, tagNames []string) ([]uint, error) {
	set := map[uint]bool{}
	var out []uint
	for _, id := range tagIDs {
		t, err := s.tags.FindByID(ctx, id)
		if err != nil || !t.Enabled {
			return nil, ErrTagNotFound
		}
		if !set[id] {
			set[id] = true
			out = append(out, id)
		}
	}
	for _, name := range tagNames {
		if name == "" {
			continue
		}
		t, err := s.tags.FindByName(ctx, name)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t, err = s.tags.Create(ctx, name)
			if err != nil {
				if platform.IsDuplicateEntry(err) {
					t, err = s.tags.FindByName(ctx, name)
				} else {
					return nil, err
				}
			}
		} else if err != nil {
			return nil, err
		}
		if !t.Enabled {
			continue
		}
		if !set[t.ID] {
			set[t.ID] = true
			out = append(out, t.ID)
		}
	}
	return out, nil
}
```

- [ ] **步骤 5：运行验证通过**

运行：`go test ./internal/service/ -run TestSkill -v`
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add internal/service/skill.go internal/service/skill_test.go internal/service/dto.go
git commit -m "feat: Skill Service（CRUD/状态机/可见性/列表/标签）"
```

---

## 任务 5：Skill Handler + 测试（路由 + 接口）

**文件：**
- 创建：`internal/handler/skill.go`、`internal/handler/skill_test.go`

- [ ] **步骤 1：写失败测试**

`internal/handler/skill_test.go`：
```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func skillRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	rdb := testutil.NewTestRedis(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50, HotCacheTTL: 60e9}
	svc := service.NewSkillService(
		repo.NewSkillRepo(db), repo.NewTagRepo(db),
		repo.NewInteractionRepo(db), rdb, cfg,
	)
	h := NewSkillHandler(svc)
	auth := platform.AuthMiddleware("s")
	opt := platform.OptionalAuthMiddleware("s")

	r := gin.New()
	skills := r.Group("/api/v1/skills")
	skills.GET("", h.List)
	skills.POST("", auth, h.Create)
	skills.GET("/:id", opt, h.Get)
	skills.PUT("/:id", auth, h.Update)
	skills.DELETE("/:id", auth, h.Delete)
	skills.POST("/:id/submit", auth, h.Submit)
	skills.POST("/:id/withdraw", auth, h.Withdraw)
	skills.POST("/:id/archive", auth, h.Archive)
	return r
}

func TestSkillCreateEndpoint(t *testing.T) {
	r := skillRouter(t)
	// 创建用户并生成 token
	// ... 省略 token 生成逻辑，参考 article_test.go

	body, _ := json.Marshal(map[string]interface{}{
		"name": "test-skill", "description": "desc",
		"tag_names": []string{"claude"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}
```

- [ ] **步骤 2：实现 SkillHandler**

`internal/handler/skill.go`：
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type SkillHandler struct{ svc *service.SkillService }

func NewSkillHandler(svc *service.SkillService) *SkillHandler { return &SkillHandler{svc: svc} }

func (h *SkillHandler) Create(c *gin.Context) {
	var in struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		RepoURL     string   `json:"repo_url"`
		TagIDs      []uint   `json:"tag_ids"`
		TagNames    []string `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	s, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), service.CreateSkillInput{
		Name: in.Name, Description: in.Description, RepoURL: in.RepoURL,
		TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": s.ID})
}

func (h *SkillHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		RepoURL     string   `json:"repo_url"`
		TagIDs      []uint   `json:"tag_ids"`
		TagNames    []string `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	s, err := h.svc.Update(c.Request.Context(), c.GetUint("user_id"), id, service.CreateSkillInput{
		Name: in.Name, Description: in.Description, RepoURL: in.RepoURL,
		TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": s.ID})
}

func (h *SkillHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *SkillHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	detail, err := h.svc.Get(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, detail)
}

func (h *SkillHandler) List(c *gin.Context) {
	q := service.SkillListQuery{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
		Keyword:  c.Query("keyword"),
		Sort:     c.Query("sort"),
	}
	if v := c.Query("tag_id"); v != "" {
		id := parseUint(v)
		q.TagID = &id
	}
	if v := c.Query("author_id"); v != "" {
		id := parseUint(v)
		q.AuthorID = &id
	}
	res, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *SkillHandler) Submit(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Submit(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *SkillHandler) Withdraw(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Withdraw(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *SkillHandler) Archive(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Archive(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}
```

- [ ] **步骤 3：运行验证通过**

运行：`go test ./internal/handler/ -run TestSkill -v`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/handler/skill.go internal/handler/skill_test.go
git commit -m "feat: Skill Handler（CRUD/状态机接口）"
```

---

## 任务 6：Skill 上传/下载/点赞/收藏

**文件：**
- 修改：`internal/service/skill.go`、`internal/service/skill_test.go`
- 修改：`internal/handler/skill.go`、`internal/handler/skill_test.go`
- 修改：`internal/repo/interaction.go`（新增 Skill 互动方法）

- [ ] **步骤 1：扩展 InteractionRepo**

`internal/repo/interaction.go` 追加：
```go
// Skill 互动
func (r *InteractionRepo) SkillLiked(db *gorm.DB, userID, skillID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.SkillLike{}).Where("skill_id = ? AND user_id = ?", skillID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) SkillFavorited(db *gorm.DB, userID, skillID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.SkillFavorite{}).Where("skill_id = ? AND user_id = ?", skillID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) ToggleSkillLike(db *gorm.DB, userID, skillID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.SkillLike{UserID: userID, SkillID: skillID}, "skill_id = ? AND user_id = ?", skillID, userID)
}

func (r *InteractionRepo) ToggleSkillFavorite(db *gorm.DB, userID, skillID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.SkillFavorite{UserID: userID, SkillID: skillID}, "skill_id = ? AND user_id = ?", skillID, userID)
}
```

- [ ] **步骤 2：实现 Upload/Download/ToggleLike/ToggleFavorite**

`internal/service/skill.go` 追加：
```go
func (s *SkillService) UploadZip(ctx context.Context, userID, skillID uint, zipURL, zipFilename string, fileSize int64) error {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	if skill.AuthorID != userID {
		return ErrForbidden
	}
	// pending_review 不可上传
	if skill.Status == model.ResourceStatusPendingReview {
		return ErrStateError
	}
	skill.ZipURL = zipURL
	skill.ZipFilename = zipFilename
	skill.FileSize = fileSize
	// published 状态上传 ZIP 自动回退 pending_review
	if skill.Status == model.ResourceStatusPublished {
		skill.Status = model.ResourceStatusPendingReview
	}
	return s.skills.Update(nil, skill)
}

func (s *SkillService) Download(ctx context.Context, skillID uint) (string, error) {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return "", ErrSkillNotFound
	}
	if skill.Status != model.ResourceStatusPublished {
		return "", ErrSkillNotFound
	}
	if skill.ZipURL == "" {
		return "", ErrBadParam
	}
	_ = s.skills.IncrCount(nil, skillID, "downloads", 1)
	return skill.ZipURL, nil
}

func (s *SkillService) ToggleLike(ctx context.Context, userID, skillID uint) (bool, int, error) {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil || skill.Status != model.ResourceStatusPublished {
		return false, 0, ErrSkillNotFound
	}
	var liked bool
	var newCount int
	err = s.skills.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleSkillLike(tx, userID, skillID)
		if err != nil {
			return err
		}
		delta := 1
		if !liked {
			delta = -1
		}
		if err := s.skills.IncrCount(tx, skillID, "likes_count", delta); err != nil {
			return err
		}
		newCount = skill.LikesCount + delta
		return nil
	})
	return liked, newCount, err
}

func (s *SkillService) ToggleFavorite(ctx context.Context, userID, skillID uint) (bool, int, error) {
	skill, err := s.skills.FindByID(nil, skillID)
	if err != nil || skill.Status != model.ResourceStatusPublished {
		return false, 0, ErrSkillNotFound
	}
	var favorited bool
	var newCount int
	err = s.skills.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		favorited, err = s.inter.ToggleSkillFavorite(tx, userID, skillID)
		if err != nil {
			return err
		}
		delta := 1
		if !favorited {
			delta = -1
		}
		if err := s.skills.IncrCount(tx, skillID, "favorites_count", delta); err != nil {
			return err
		}
		newCount = skill.FavoritesCount + delta
		return nil
	})
	return favorited, newCount, err
}
```

- [ ] **步骤 3：实现 Handler 方法**

`internal/handler/skill.go` 追加：
```go
func (h *SkillHandler) Upload(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	// 校验扩展名
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "仅支持 ZIP 文件")
		return
	}
	if file.Size > h.svc.MaxZipBytes() {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "文件过大")
		return
	}
	// 保存文件
	if err := os.MkdirAll(h.svc.ZipDir(), 0o755); err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, "服务器内部错误")
		return
	}
	name := randomHex(16) + ".zip"
	if err := c.SaveUploadedFile(file, filepath.Join(h.svc.ZipDir(), name)); err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, "服务器内部错误")
		return
	}
	zipURL := "/static/skills/" + name
	if err := h.svc.UploadZip(c.Request.Context(), c.GetUint("user_id"), id, zipURL, file.Filename, file.Size); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"url": zipURL})
}

func (h *SkillHandler) Download(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	zipURL, err := h.svc.Download(c.Request.Context(), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"url": zipURL})
}

func (h *SkillHandler) Like(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	liked, count, err := h.svc.ToggleLike(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"liked": liked, "likes_count": count})
}

func (h *SkillHandler) Favorite(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	favorited, count, err := h.svc.ToggleFavorite(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"favorited": favorited, "favorites_count": count})
}
```

- [ ] **步骤 4：运行测试**

运行：`go test ./internal/service/ -run TestSkill -v` 和 `go test ./internal/handler/ -run TestSkill -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/service/skill.go internal/service/skill_test.go internal/handler/skill.go internal/handler/skill_test.go internal/repo/interaction.go
git commit -m "feat: Skill 上传/下载/点赞/收藏"
```

---

## 任务 7：MCP Server 后端（复用 Skill 模式）

**文件：**
- 创建：`internal/repo/mcp_server.go`、`internal/repo/mcp_server_test.go`
- 创建：`internal/service/mcp_server.go`、`internal/service/mcp_server_test.go`
- 创建：`internal/handler/mcp_server.go`、`internal/handler/mcp_server_test.go`
- 修改：`internal/service/dto.go`
- 修改：`internal/repo/interaction.go`

- [ ] **步骤 1：实现 McpServerRepo**

复用 SkillRepo 的模式，创建 `McpServerRepo`，方法签名一致。

- [ ] **步骤 2：定义 MCP Server DTO**

`internal/service/dto.go` 追加：
```go
type McpServerSummary struct {
	ID             uint        `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	RepoURL        string      `json:"repo_url"`
	Tags           []TagBrief  `json:"tags"`
	Author         AuthorBrief `json:"author"`
	Views          int         `json:"views"`
	Downloads      int         `json:"downloads"`
	LikesCount     int         `json:"likes_count"`
	FavoritesCount int         `json:"favorites_count"`
	CommentsCount  int         `json:"comments_count"`
	Status         string      `json:"status"`
	PublishedAt    *time.Time  `json:"published_at"`
}

type McpServerDetail struct {
	McpServerSummary
	ToolsJSON   string `json:"tools_json"`
	Readme      string `json:"readme"`
	ZipURL      string `json:"zip_url"`
	ZipFilename string `json:"zip_filename"`
	FileSize    int64  `json:"file_size"`
	Liked       bool   `json:"liked"`
	Favorited   bool   `json:"favorited"`
}

type CreateMcpServerInput struct {
	Name        string
	Description string
	RepoURL     string
	ToolsJSON   string
	Readme      string
	TagIDs      []uint
	TagNames    []string
}
```

- [ ] **步骤 3：实现 McpServerService**

复用 SkillService 的模式，创建 `McpServerService`。

- [ ] **步骤 4：扩展 InteractionRepo**

追加 `McpServerLiked`、`McpServerFavorited`、`ToggleMcpServerLike`、`ToggleMcpServerFavorite`。

- [ ] **步骤 5：实现 McpServerHandler**

复用 SkillHandler 的模式。

- [ ] **步骤 6：运行测试**

运行：`go test ./internal/repo/ -run TestMcpServer -v`、`go test ./internal/service/ -run TestMcpServer -v`、`go test ./internal/handler/ -run TestMcpServer -v`
预期：PASS

- [ ] **步骤 7：Commit**

```bash
git add internal/repo/mcp_server.go internal/repo/mcp_server_test.go internal/service/mcp_server.go internal/service/mcp_server_test.go internal/handler/mcp_server.go internal/handler/mcp_server_test.go internal/service/dto.go internal/repo/interaction.go
git commit -m "feat: MCP Server 后端（Repo/Service/Handler）"
```

---

## 任务 8：资源评论后端

**文件：**
- 创建：`internal/repo/resource_comment.go`、`internal/repo/resource_comment_test.go`
- 创建：`internal/service/resource_comment.go`、`internal/service/resource_comment_test.go`
- 创建：`internal/handler/resource_comment.go`、`internal/handler/resource_comment_test.go`
- 修改：`internal/service/dto.go`

- [ ] **步骤 1：实现 ResourceCommentRepo**

```go
type ResourceCommentRepo struct{ db *gorm.DB }

func NewResourceCommentRepo(db *gorm.DB) *ResourceCommentRepo {
	return &ResourceCommentRepo{db: db}
}

func (r *ResourceCommentRepo) Create(db *gorm.DB, c *model.ResourceComment) error {
	return r.exec(db).Create(c).Error
}

func (r *ResourceCommentRepo) FindByID(db *gorm.DB, id uint) (*model.ResourceComment, error) {
	var c model.ResourceComment
	if err := r.exec(db).Preload("Author").First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ResourceCommentRepo) ListByResource(ctx context.Context, resourceType string, resourceID uint) ([]model.ResourceComment, error) {
	var list []model.ResourceComment
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Preload("Author").
		Order("created_at asc").
		Find(&list).Error
	return list, err
}

func (r *ResourceCommentRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.ResourceComment{}, id).Error
}

func (r *ResourceCommentRepo) IncrLikes(db *gorm.DB, id uint, delta int) error {
	return r.exec(db).Model(&model.ResourceComment{}).
		Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("likes_count + ?", delta)).Error
}
```

- [ ] **步骤 2：定义资源评论 DTO**

`internal/service/dto.go` 追加：
```go
type ResourceCommentItem struct {
	ID         uint                  `json:"id"`
	ResourceID uint                  `json:"resource_id"`
	AuthorID   uint                  `json:"author_id"`
	Author     AuthorBrief           `json:"author"`
	Content    string                `json:"content"`
	LikesCount int                   `json:"likes_count"`
	CreatedAt  time.Time             `json:"created_at"`
	Replies    []ResourceCommentItem `json:"replies"`
}
```

- [ ] **步骤 3：实现 ResourceCommentService**

两级评论结构，复用文章评论的逻辑模式。

- [ ] **步骤 4：实现 ResourceCommentHandler**

```go
type ResourceCommentHandler struct{ svc *service.ResourceCommentService }

func (h *ResourceCommentHandler) List(c *gin.Context) { ... }
func (h *ResourceCommentHandler) Create(c *gin.Context) { ... }
func (h *ResourceCommentHandler) Delete(c *gin.Context) { ... }
func (h *ResourceCommentHandler) Like(c *gin.Context) { ... }
```

- [ ] **步骤 5：运行测试**

运行：`go test ./internal/repo/ -run TestResourceComment -v` 等
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add internal/repo/resource_comment.go internal/repo/resource_comment_test.go internal/service/resource_comment.go internal/service/resource_comment_test.go internal/handler/resource_comment.go internal/handler/resource_comment_test.go internal/service/dto.go
git commit -m "feat: 资源评论（两级结构/点赞/权限）"
```

---

## 任务 9：路由装配 + 静态目录 + main.go

**文件：**
- 修改：`cmd/server/main.go`

- [ ] **步骤 1：注册新模型迁移**

```go
if err := db.AutoMigrate(
	// 现有模型
	&model.User{}, &model.Category{}, &model.Tag{}, &model.Article{},
	&model.ArticleTag{}, &model.ArticleLike{}, &model.ArticleFavorite{},
	&model.Comment{}, &model.CommentLike{},
	// P3 新增
	&model.Skill{}, &model.SkillTag{},
	&model.McpServer{}, &model.McpServerTag{},
	&model.SkillLike{}, &model.SkillFavorite{},
	&model.McpServerLike{}, &model.McpServerFavorite{},
	&model.ResourceComment{}, &model.ResourceCommentLike{},
); err != nil {
	logger.Error("migrate", "err", err)
	return
}
```

- [ ] **步骤 2：装配新 repo/service/handler**

```go
// P3 repos
skills := repo.NewSkillRepo(db)
mcpServers := repo.NewMcpServerRepo(db)
resComments := repo.NewResourceCommentRepo(db)

// P3 services
skillSvc := service.NewSkillService(skills, tags, inter, rdb, cfg)
mcpSvc := service.NewMcpServerService(mcpServers, tags, inter, rdb, cfg)
resCommentSvc := service.NewResourceCommentService(resComments, skills, mcpServers, inter, users)

// P3 handlers
skillH := handler.NewSkillHandler(skillSvc)
mcpH := handler.NewMcpServerHandler(mcpSvc)
resCommentH := handler.NewResourceCommentHandler(resCommentSvc)
```

- [ ] **步骤 3：注册路由**

```go
// Skills Hub
skillsGroup := r.Group("/api/v1/skills")
skillsGroup.GET("", skillH.List)
skillsGroup.POST("", p2Auth, skillH.Create)
skillsGroup.GET("/:id", opt, skillH.Get)
skillsGroup.PUT("/:id", p2Auth, skillH.Update)
skillsGroup.DELETE("/:id", p2Auth, skillH.Delete)
skillsGroup.POST("/:id/upload", p2Auth, skillH.Upload)
skillsGroup.POST("/:id/submit", p2Auth, skillH.Submit)
skillsGroup.POST("/:id/withdraw", p2Auth, skillH.Withdraw)
skillsGroup.POST("/:id/archive", p2Auth, skillH.Archive)
skillsGroup.POST("/:id/download", skillH.Download)
skillsGroup.POST("/:id/like", p2Auth, skillH.Like)
skillsGroup.POST("/:id/favorite", p2Auth, skillH.Favorite)

// Skill 评论
skillComments := r.Group("/api/v1/skills/:id/comments")
skillComments.GET("", resCommentH.List)
skillComments.POST("", p2Auth, resCommentH.Create)

// MCP Hub
mcpGroup := r.Group("/api/v1/mcp-servers")
mcpGroup.GET("", mcpH.List)
mcpGroup.POST("", p2Auth, mcpH.Create)
mcpGroup.GET("/:id", opt, mcpH.Get)
mcpGroup.PUT("/:id", p2Auth, mcpH.Update)
mcpGroup.DELETE("/:id", p2Auth, mcpH.Delete)
mcpGroup.POST("/:id/upload", p2Auth, mcpH.Upload)
mcpGroup.POST("/:id/submit", p2Auth, mcpH.Submit)
mcpGroup.POST("/:id/withdraw", p2Auth, mcpH.Withdraw)
mcpGroup.POST("/:id/archive", p2Auth, mcpH.Archive)
mcpGroup.POST("/:id/download", mcpH.Download)
mcpGroup.POST("/:id/like", p2Auth, mcpH.Like)
mcpGroup.POST("/:id/favorite", p2Auth, mcpH.Favorite)

// MCP Server 评论
mcpComments := r.Group("/api/v1/mcp-servers/:id/comments")
mcpComments.GET("", resCommentH.List)
mcpComments.POST("", p2Auth, resCommentH.Create)

// 资源评论管理
resComs := r.Group("/api/v1/resource-comments")
resComs.DELETE("/:id", p2Auth, resCommentH.Delete)
resComs.POST("/:id/like", p2Auth, resCommentH.Like)

// 静态目录
r.Static("/static/skills", cfg.SkillZipDir)
r.Static("/static/mcp-servers", cfg.McpServerZipDir)
```

- [ ] **步骤 4：运行全量测试**

运行：`go test ./...`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: P3 路由装配 + 静态目录"
```

---

## 任务 10：前端 API + 类型定义

**文件：**
- 创建：`frontend/src/api/skill.ts`、`frontend/src/api/mcpServer.ts`、`frontend/src/api/resourceComment.ts`
- 修改：`frontend/src/types/index.ts`

- [ ] **步骤 1：定义 TypeScript 类型**

`frontend/src/types/index.ts` 追加：
```typescript
// Skill
export interface SkillSummary {
  id: number
  name: string
  description: string
  repo_url: string
  tags: TagBrief[]
  author: AuthorBrief
  views: number
  downloads: number
  likes_count: number
  favorites_count: number
  comments_count: number
  status: string
  published_at: string | null
}

export interface SkillDetail extends SkillSummary {
  zip_url: string
  zip_filename: string
  file_size: number
  liked: boolean
  favorited: boolean
}

export interface SkillListResult {
  list: SkillSummary[]
  total: number
  page: number
  page_size: number
}

export interface SkillListQuery {
  page?: number
  page_size?: number
  tag_id?: number
  keyword?: string
  sort?: string
}

export interface SkillForm {
  name: string
  description: string
  repo_url?: string
  tag_ids: number[]
  tag_names: string[]
}

// MCP Server
export interface McpServerSummary {
  id: number
  name: string
  description: string
  repo_url: string
  tags: TagBrief[]
  author: AuthorBrief
  views: number
  downloads: number
  likes_count: number
  favorites_count: number
  comments_count: number
  status: string
  published_at: string | null
}

export interface McpServerDetail extends McpServerSummary {
  tools_json: string
  readme: string
  zip_url: string
  zip_filename: string
  file_size: number
  liked: boolean
  favorited: boolean
}

// 资源评论
export interface ResourceCommentItem {
  id: number
  resource_id: number
  author_id: number
  author: AuthorBrief
  content: string
  likes_count: number
  created_at: string
  replies: ResourceCommentItem[]
}
```

- [ ] **步骤 2：实现 API 函数**

`frontend/src/api/skill.ts`：
```typescript
import http from './http'
import type { ApiResponse, SkillListResult, SkillDetail, LikeResult, FavoriteResult } from '@/types'

export function getSkills(params?: Record<string, any>) {
  return http.get<ApiResponse<SkillListResult>>('/api/v1/skills', { params })
}

export function getSkill(id: number) {
  return http.get<ApiResponse<SkillDetail>>(`/api/v1/skills/${id}`)
}

export function createSkill(data: any) {
  return http.post<ApiResponse<{ id: number }>>('/api/v1/skills', data)
}

export function updateSkill(id: number, data: any) {
  return http.put<ApiResponse<{ id: number }>>(`/api/v1/skills/${id}`, data)
}

export function deleteSkill(id: number) {
  return http.delete<ApiResponse<null>>(`/api/v1/skills/${id}`)
}

export function uploadSkillZip(id: number, file: File) {
  const form = new FormData()
  form.append('file', file)
  return http.post<ApiResponse<{ url: string }>>(`/api/v1/skills/${id}/upload`, form)
}

export function submitSkill(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/skills/${id}/submit`)
}

export function withdrawSkill(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/skills/${id}/withdraw`)
}

export function archiveSkill(id: number) {
  return http.post<ApiResponse<null>>(`/api/v1/skills/${id}/archive`)
}

export function downloadSkill(id: number) {
  return http.post<ApiResponse<{ url: string }>>(`/api/v1/skills/${id}/download`)
}

export function likeSkill(id: number) {
  return http.post<ApiResponse<LikeResult>>(`/api/v1/skills/${id}/like`)
}

export function favoriteSkill(id: number) {
  return http.post<ApiResponse<FavoriteResult>>(`/api/v1/skills/${id}/favorite`)
}
```

同样创建 `mcpServer.ts` 和 `resourceComment.ts`。

- [ ] **步骤 3：Commit**

```bash
git add frontend/src/api/ frontend/src/types/
git commit -m "feat: 前端 API + 类型定义（Skill/MCP Server/资源评论）"
```

---

## 任务 11：前端 Skills Hub 页面

**文件：**
- 修改：`frontend/src/views/SkillsView.vue`
- 创建：`frontend/src/components/SkillCard.vue`

- [ ] **步骤 1：实现 SkillsView**

替换占位内容，实现：
- 搜索框
- 排序选择（最新/热门/下载量）
- 标签筛选
- Skill 列表（使用 SkillCard 组件）
- 分页

- [ ] **步骤 2：创建 SkillCard 组件**

展示 Skill 摘要信息：名称、描述、作者、标签、统计数字。

- [ ] **步骤 3：Commit**

```bash
git add frontend/src/views/SkillsView.vue frontend/src/components/SkillCard.vue
git commit -m "feat: 前端 Skills Hub 页面"
```

---

## 任务 12：前端 MCP Hub 页面

**文件：**
- 修改：`frontend/src/views/McpsView.vue`
- 创建：`frontend/src/components/McpServerCard.vue`

- [ ] **步骤 1：实现 McpsView**

替换占位内容，实现：
- 搜索框
- 排序选择
- 标签筛选
- MCP Server 列表
- 分页

- [ ] **步骤 2：创建 McpServerCard 组件**

- [ ] **步骤 3：Commit**

```bash
git add frontend/src/views/McpsView.vue frontend/src/components/McpServerCard.vue
git commit -m "feat: 前端 MCP Hub 页面"
```

---

## 任务 13：前端 Skill/MCP Server 详情页

**文件：**
- 创建：`frontend/src/views/SkillDetailView.vue`、`frontend/src/views/McpServerDetailView.vue`
- 修改：`frontend/src/router/index.ts`

- [ ] **步骤 1：实现 SkillDetailView**

展示 Skill 详情：
- 基本信息（名称、描述、作者、仓库链接）
- 标签
- 统计数字（浏览量、下载量、点赞、收藏、评论）
- 下载按钮
- 点赞/收藏按钮
- 评论区

- [ ] **步骤 2：实现 McpServerDetailView**

展示 MCP Server 详情：
- 基本信息
- Tools 清单（解析 tools_json）
- README 渲染（Markdown）
- 下载按钮（如有 ZIP）
- 点赞/收藏/评论

- [ ] **步骤 3：注册路由**

```typescript
{ path: 'skills/:id', name: 'skill-detail', component: () => import('@/views/SkillDetailView.vue') },
{ path: 'mcps/:id', name: 'mcp-detail', component: () => import('@/views/McpServerDetailView.vue') },
```

- [ ] **步骤 4：Commit**

```bash
git add frontend/src/views/SkillDetailView.vue frontend/src/views/McpServerDetailView.vue frontend/src/router/
git commit -m "feat: 前端 Skill/MCP Server 详情页"
```

---

## 任务 14：前端 Skill/MCP Server 发布/编辑页

**文件：**
- 创建：`frontend/src/views/SkillEditView.vue`、`frontend/src/views/McpServerEditView.vue`
- 修改：`frontend/src/router/index.ts`

- [ ] **步骤 1：实现 SkillEditView**

表单字段：
- 名称、描述、仓库地址
- 标签选择（多选 + 新建）
- ZIP 上传
- 保存草稿 / 发布按钮

- [ ] **步骤 2：实现 McpServerEditView**

表单字段：
- 名称、描述、仓库地址
- Tools 清单（JSON 编辑器或结构化表单）
- README（Markdown 编辑器）
- ZIP 上传（可选）
- 标签选择
- 保存草稿 / 发布按钮

- [ ] **步骤 3：注册路由**

```typescript
{ path: 'skills/new', name: 'skill-new', component: () => import('@/views/SkillEditView.vue'), meta: { requiresAuth: true } },
{ path: 'skills/:id/edit', name: 'skill-edit', component: () => import('@/views/SkillEditView.vue'), meta: { requiresAuth: true } },
{ path: 'mcps/new', name: 'mcp-new', component: () => import('@/views/McpServerEditView.vue'), meta: { requiresAuth: true } },
{ path: 'mcps/:id/edit', name: 'mcp-edit', component: () => import('@/views/McpServerEditView.vue'), meta: { requiresAuth: true } },
```

- [ ] **步骤 4：Commit**

```bash
git add frontend/src/views/SkillEditView.vue frontend/src/views/McpServerEditView.vue frontend/src/router/
git commit -m "feat: 前端 Skill/MCP Server 发布/编辑页"
```

---

## 任务 15：集成测试 + 最终验证

- [ ] **步骤 1：运行后端全量测试**

```bash
go test ./...
```

预期：全部 PASS

- [ ] **步骤 2：运行前端类型检查 + 构建**

```bash
cd frontend
npm run typecheck
npm run build
```

预期：无错误

- [ ] **步骤 3：手动测试**

启动后端 + 前端，测试：
- Skill 创建/编辑/删除
- Skill 提交审核/撤回/下架
- Skill ZIP 上传/下载
- Skill 点赞/收藏/评论
- MCP Server 同上
- 列表筛选/搜索/排序

- [ ] **步骤 4：最终 Commit**

```bash
git add .
git commit -m "feat: P3 AI 资源（Skills Hub + MCP Hub）完成"
```
