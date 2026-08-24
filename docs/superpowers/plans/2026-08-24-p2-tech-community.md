# P2 技术社区实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现技术社区——文章发布/浏览（分类+标签+搜索+排序）、两级评论、点赞/收藏/浏览量统计。

**架构：** 沿用扁平技术分层（`internal/handler` → `service` → `repo` / `model`）。互动用独立小表 + 文章/评论表计数冗余列（事务内同步增减）；浏览量详情访问 +1；热门排行 SQL 计算、结果缓存 Redis 60s。权限规则：游客可浏览，登录用户可互动/发布，作者管理自己内容。

**技术栈：** Go / Gin / GORM / MySQL 8 / Redis（已有依赖，无需新增）。测试沿用真实 MySQL/Redis 按进程隔离（`internal/testutil`）。

规格：`docs/superpowers/specs/2026-08-24-p2-tech-community-design.md`

---

## 文件结构

**新增模型**（`internal/model/`）
- `category.go` — `Category`
- `tag.go` — `Tag`
- `article.go` — `Article`、`ArticleTag`、`ArticleStatus` 常量
- `comment.go` — `Comment`
- `interaction.go` — `ArticleLike`、`ArticleFavorite`、`CommentLike`

**新增 repo**（`internal/repo/`）
- `category.go` — `CategoryRepo`（List / FindByID / Seed）
- `tag.go` — `TagRepo`（FindByID / FindByName / Create / List / ListHot / IncrUsage）
- `article.go` — `ArticleRepo`（Create / FindByID / Update / Delete / List / Count / IncrViews / IncrCount / FindArticleTags / TagsForArticles / SetArticleTags / DeleteArticleTags）
- `comment.go` — `CommentRepo`（Create / FindByID / ListByArticle / Delete / IncrLikes）
- `interaction.go` — `InteractionRepo`（Toggle 三个 + 查询两个）

**新增 service**（`internal/service/`）
- `category.go` — `CategoryService.List`
- `tag.go` — `TagService.List`、`ResolveTagSet`
- `article.go` — `ArticleService`（Create/Update/Delete/List/Get/ToggleLike/ToggleFavorite/图片辅助）
- `comment.go` — `CommentService`（Create/List/Delete/ToggleLike）
- `dto.go` — 文章/评论响应 DTO 与 `ListQuery`、`CreateArticleInput` 等

**新增 handler**（`internal/handler/`）
- `category.go`、`tag.go`、`article.go`、`comment.go`

**修改**
- `internal/platform/config.go` — 新增图片/热门缓存/分页配置
- `internal/platform/db.go`（新增）— `IsDuplicateEntry`
- `internal/platform/auth_middleware.go` — 新增 `OptionalAuthMiddleware`
- `internal/service/auth.go` — 复用 `platform.IsDuplicateEntry`，删本地副本
- `internal/testutil/testutil.go` — `NewTestDB` 迁移全部新模型；新增分类种子 helper
- `cmd/server/main.go` — 装配新 repo/service/handler、路由、迁移、种子、静态目录
- `internal/handler/handler_test.go` — `setupRouter` 扩展新路由

**测试文件**：每个包 `*_test.go` 与源码同目录（`internal/repo/category_test.go` 等）。

---

## 任务 1：模型层 + testutil 扩展 + platform.IsDuplicateEntry

**文件：**
- 创建：`internal/model/category.go`、`internal/model/tag.go`、`internal/model/article.go`、`internal/model/comment.go`、`internal/model/interaction.go`
- 创建：`internal/platform/db.go`
- 修改：`internal/testutil/testutil.go`、`internal/service/auth.go`
- 测试：`internal/model/model_test.go`、`internal/platform/db_test.go`

- [ ] **步骤 1：创建模型文件**

`internal/model/category.go`：
```go
package model

import "time"

type Category struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:64;uniqueIndex;not null"`
	Slug      string `gorm:"size:64;uniqueIndex;not null"`
	SortOrder int    `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

`internal/model/tag.go`：
```go
package model

import "time"

type Tag struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"size:64;uniqueIndex;not null"`
	UsageCount int    `gorm:"not null;default:0"`
	Enabled    bool   `gorm:"not null;default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
```

`internal/model/article.go`：
```go
package model

import (
	"time"

	"gorm.io/gorm"
)

type ArticleStatus string

const (
	ArticleStatusDraft     ArticleStatus = "draft"
	ArticleStatusPublished ArticleStatus = "published"
)

type Article struct {
	ID             uint          `gorm:"primaryKey"`
	AuthorID       uint          `gorm:"not null;index"`
	CategoryID     uint          `gorm:"not null;index"`
	Title          string        `gorm:"size:200;not null"`
	Summary        string        `gorm:"size:500"`
	Content        string        `gorm:"type:mediumtext"`
	Status         ArticleStatus `gorm:"size:16;not null;default:draft;index"`
	Views          int           `gorm:"not null;default:0"`
	LikesCount     int           `gorm:"not null;default:0"`
	FavoritesCount int           `gorm:"not null;default:0"`
	CommentsCount  int           `gorm:"not null;default:0"`
	Pinned         bool          `gorm:"not null;default:false"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Author   *User     `gorm:"foreignKey:AuthorID"`
	Category *Category `gorm:"foreignKey:CategoryID"`
}

type ArticleTag struct {
	ID        uint `gorm:"primaryKey"`
	ArticleID uint `gorm:"uniqueIndex:uniq_article_tag;not null"`
	TagID     uint `gorm:"uniqueIndex:uniq_article_tag;not null"`
	CreatedAt time.Time
}
```

`internal/model/comment.go`：
```go
package model

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID         uint   `gorm:"primaryKey"`
	ArticleID  uint   `gorm:"not null;index"`
	AuthorID   uint   `gorm:"not null;index"`
	ParentID   *uint  `gorm:"index"`
	Content    string `gorm:"type:text;not null"`
	LikesCount int    `gorm:"not null;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
```

`internal/model/interaction.go`：
```go
package model

import "time"

type ArticleLike struct {
	ID        uint `gorm:"primaryKey"`
	ArticleID uint `gorm:"uniqueIndex:uniq_article_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_article_like;not null"`
	CreatedAt time.Time
}

type ArticleFavorite struct {
	ID        uint `gorm:"primaryKey"`
	ArticleID uint `gorm:"uniqueIndex:uniq_article_fav;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_article_fav;not null"`
	CreatedAt time.Time
}

type CommentLike struct {
	ID        uint `gorm:"primaryKey"`
	CommentID uint `gorm:"uniqueIndex:uniq_comment_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_comment_like;not null"`
	CreatedAt time.Time
}
```

- [ ] **步骤 2：新增 platform.IsDuplicateEntry**

`internal/platform/db.go`：
```go
package platform

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicateEntry 判断是否 MySQL 唯一索引冲突（error 1062）。
func IsDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	return strings.Contains(err.Error(), "Duplicate entry")
}
```

- [ ] **步骤 3：修改 auth.go 复用 platform.IsDuplicateEntry**

`internal/service/auth.go`：删除本地 `isDuplicateEntry` 函数，`Register` 内改调用 `platform.IsDuplicateEntry(err)`。删除 `"github.com/go-sql-driver/mysql"` 与 `"strings"` 的导入（若不再使用）。

- [ ] **步骤 4：扩展 testutil.NewTestDB 迁移全部模型 + 分类种子 helper**

`internal/testutil/testutil.go`：`NewTestDB` 中把 `DropTable(&model.User{})` / `AutoMigrate(&model.User{})` 改为遍历全部模型：

```go
var allModels = []interface{}{
	&model.User{}, &model.Category{}, &model.Tag{}, &model.Article{},
	&model.ArticleTag{}, &model.ArticleLike{}, &model.ArticleFavorite{},
	&model.Comment{}, &model.CommentLike{},
}

func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ensureTestDB(t)
	db, err := gorm.Open(mysql.Open(testMySQLDSN), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range allModels {
		_ = db.Migrator().DropTable(m)
	}
	for _, m := range allModels {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, m := range allModels {
			_ = db.Migrator().DropTable(m)
		}
	})
	return db
}
```

- [ ] **步骤 5：写失败测试**

`internal/platform/db_test.go`：
```go
package platform

import (
	"testing"

	"gorm.io/gorm"
)

func TestIsDuplicateEntry(t *testing.T) {
	if IsDuplicateEntry(gorm.ErrRecordNotFound) {
		t.Fatal("RecordNotFound should not be duplicate")
	}
	// 1062 错误码（MySQLError）
	me := &mysqlErr{code: 1062}
	if !IsDuplicateEntry(me) {
		t.Fatal("1062 should be duplicate")
	}
	if IsDuplicateEntry(&mysqlErr{code: 1452}) {
		t.Fatal("1452 should not be duplicate")
	}
}
```
（`mysqlErr` 用本地最小实现：`type mysqlErr struct{ code uint16 }; func (e *mysqlErr) Error() string{ return "err 1062" }`。若驱动类型 `mysql.MySQLError` 可直接构造：`&mysql.MySQLError{Number: 1062}`——优先用驱动类型，`mysqlErr` 仅兜底。）

- [ ] **步骤 6：运行验证失败**

运行：`go test ./internal/platform/ -run TestIsDuplicateEntry -v`
预期：FAIL（`IsDuplicateEntry` 未定义）

- [ ] **步骤 7：实现并验证通过**

`internal/platform/db.go` 已写（步骤 2）。运行：`go test ./internal/platform/ -run TestIsDuplicateEntry -v`
预期：PASS

- [ ] **步骤 8：验证 testutil 迁移**

`internal/model/model_test.go`：
```go
package model

import (
	"testing"

	"aidevclub/internal/testutil"
)

func TestNewTestDBMigratesAllModels(t *testing.T) {
	db := testutil.NewTestDB(t)
	for _, m := range []interface{}{
		&Category{}, &Tag{}, &Article{}, &ArticleTag{},
		&ArticleLike{}, &ArticleFavorite{}, &Comment{}, &CommentLike{},
	} {
		if !db.Migrator().HasTable(m) {
			t.Fatalf("table %T not migrated", m)
		}
	}
}
```
运行：`go test ./internal/model/ -v`
预期：PASS（依赖 docker compose 中的 MySQL/Redis；启动命令见仓库 CLAUDE.md）

- [ ] **步骤 9：Commit**

```bash
git add internal/model/ internal/platform/db.go internal/platform/db_test.go internal/testutil/testutil.go internal/service/auth.go
git commit -m "feat: P2 数据模型与测试基建（分类/标签/文章/评论/互动）"
```

---

## 任务 2：分类（repo/service/handler + 种子 + 路由）

**文件：**
- 创建：`internal/repo/category.go`、`internal/repo/category_test.go`、`internal/service/category.go`、`internal/service/category_test.go`、`internal/handler/category.go`、`internal/handler/category_test.go`

- [ ] **步骤 1：写失败测试（repo）**

`internal/repo/category_test.go`：
```go
package repo

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestCategorySeedAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewCategoryRepo(db)
	ctx := context.Background()
	if err := r.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	// 幂等：再跑一次不报错、数量不变
	if err := r.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	list, err := r.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("no categories seeded")
	}
	if list[0].Name == "" || list[0].Slug == "" {
		t.Fatal("category missing name/slug")
	}
}
```

- [ ] **步骤 2：运行验证失败**

运行：`go test ./internal/repo/ -run TestCategorySeedAndList -v`
预期：FAIL（`NewCategoryRepo` 未定义）

- [ ] **步骤 3：实现 repo**

`internal/repo/category.go`：
```go
package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

var defaultCategories = []model.Category{
	{Name: "Go", Slug: "go", SortOrder: 1},
	{Name: "后端", Slug: "backend", SortOrder: 2},
	{Name: "前端", Slug: "frontend", SortOrder: 3},
	{Name: "AI/LLM", Slug: "ai-llm", SortOrder: 4},
	{Name: "DevOps", Slug: "devops", SortOrder: 5},
	{Name: "数据库", Slug: "database", SortOrder: 6},
	{Name: "移动端", Slug: "mobile", SortOrder: 7},
	{Name: "安全", Slug: "security", SortOrder: 8},
	{Name: "其他", Slug: "other", SortOrder: 9},
}

type CategoryRepo struct{ db *gorm.DB }

func NewCategoryRepo(db *gorm.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) List(ctx context.Context) ([]model.Category, error) {
	var list []model.Category
	err := r.db.WithContext(ctx).Order("sort_order asc, id asc").Find(&list).Error
	return list, err
}

func (r *CategoryRepo) FindByID(ctx context.Context, id uint) (*model.Category, error) {
	var c model.Category
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepo) Seed(ctx context.Context) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Category{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&defaultCategories).Error
}
```

- [ ] **步骤 4：运行验证通过**

运行：`go test ./internal/repo/ -run TestCategorySeedAndList -v`
预期：PASS

- [ ] **步骤 5：写失败测试（service）+ 实现**

`internal/service/category_test.go`：
```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func TestCategoryServiceList(t *testing.T) {
	db := testutil.NewTestDB(t)
	catRepo := repo.NewCategoryRepo(db)
	_ = catRepo.Seed(context.Background())
	svc := NewCategoryService(catRepo)
	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("empty list")
	}
}
```

`internal/service/category.go`：
```go
package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type CategoryService struct{ cats *repo.CategoryRepo }

func NewCategoryService(cats *repo.CategoryRepo) *CategoryService { return &CategoryService{cats: cats} }

func (s *CategoryService) List(ctx context.Context) ([]model.Category, error) {
	return s.cats.List(ctx)
}
```

运行：`go test ./internal/service/ -run TestCategoryServiceList -v`
预期：PASS

- [ ] **步骤 6：写失败测试（handler）+ 实现**

`internal/handler/category_test.go`：
```go
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func TestCategoriesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	_ = repo.NewCategoryRepo(db).Seed(context.Background())
	h := NewCategoryHandler(service.NewCategoryService(repo.NewCategoryRepo(db)))
	r := gin.New()
	r.GET("/api/v1/categories", h.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	// 用 encoding/json 反序列化断言至少一个分类存在
}
```

`internal/handler/category.go`：
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type CategoryHandler struct{ svc *service.CategoryService }

func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler { return &CategoryHandler{svc: svc} }

func (h *CategoryHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, cat := range list {
		out = append(out, gin.H{"id": cat.ID, "name": cat.Name, "slug": cat.Slug})
	}
	platform.OK(c, out)
}
```

运行：`go test ./internal/handler/ -run TestCategoriesEndpoint -v`
预期：PASS

- [ ] **步骤 7：Commit**

```bash
git add internal/repo/category.go internal/repo/category_test.go internal/service/category.go internal/service/category_test.go internal/handler/category.go internal/handler/category_test.go
git commit -m "feat: 分类列表接口（预置种子 + 只读）"
```

---

## 任务 3：标签（repo/service/handler + 路由）

**文件：**
- 创建：`internal/repo/tag.go`、`internal/repo/tag_test.go`、`internal/service/tag.go`、`internal/service/tag_test.go`、`internal/handler/tag.go`、`internal/handler/tag_test.go`

- [ ] **步骤 1：写失败测试（repo）**

`internal/repo/tag_test.go`：
```go
package repo

import (
	"context"
	"testing"

	"aidevclub/internal/testutil"
)

func TestTagCRUDAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewTagRepo(db)
	ctx := context.Background()

	tg, err := r.Create(ctx, "gin")
	if err != nil {
		t.Fatal(err)
	}
	if tg.ID == 0 {
		t.Fatal("no id")
	}
	if _, err := r.Create(ctx, "gin"); err == nil {
		t.Fatal("duplicate tag accepted")
	}
	if found, err := r.FindByName(ctx, "gin"); err != nil || found.ID != tg.ID {
		t.Fatalf("FindByName = %v, %v", found, err)
	}
	// 使用次数增减
	if err := r.IncrUsage(db, tg.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrUsage(db, tg.ID, -1); err != nil {
		t.Fatal(err)
	}
	list, err := r.List(ctx, "gi", 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("List keyword = %v, %v", list, err)
	}
	if err := r.IncrUsage(db, tg.ID, 3); err != nil {
		t.Fatal(err)
	}
	hot, err := r.ListHot(ctx, 10)
	if err != nil || len(hot) != 1 || hot[0].UsageCount != 3 {
		t.Fatalf("ListHot = %v, %v", hot, err)
	}
}
```

- [ ] **步骤 2：运行验证失败**

运行：`go test ./internal/repo/ -run TestTagCRUDAndList -v`
预期：FAIL（`NewTagRepo` 未定义）

- [ ] **步骤 3：实现 repo**

`internal/repo/tag.go`：
```go
package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type TagRepo struct{ db *gorm.DB }

func NewTagRepo(db *gorm.DB) *TagRepo { return &TagRepo{db: db} }

func (r *TagRepo) FindByID(ctx context.Context, id uint) (*model.Tag, error) {
	var t model.Tag
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TagRepo) FindByName(ctx context.Context, name string) (*model.Tag, error) {
	var t model.Tag
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TagRepo) Create(ctx context.Context, name string) (*model.Tag, error) {
	t := &model.Tag{Name: name, Enabled: true}
	err := r.db.WithContext(ctx).Create(t).Error
	return t, err
}

// List 返回启用标签；keyword 非空时按名字前缀过滤；按名字升序。
func (r *TagRepo) List(ctx context.Context, keyword string, limit int) ([]model.Tag, error) {
	q := r.db.WithContext(ctx).Where("enabled = ?", true)
	if keyword != "" {
		q = q.Where("name LIKE ?", keyword+"%")
	}
	var list []model.Tag
	err := q.Order("name asc").Limit(limit).Find(&list).Error
	return list, err
}

func (r *TagRepo) ListHot(ctx context.Context, limit int) ([]model.Tag, error) {
	var list []model.Tag
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("usage_count desc, id asc").
		Limit(limit).Find(&list).Error
	return list, err
}

// IncrUsage 在事务中增减使用次数（db 由调用方传入，可能是事务句柄）。
func (r *TagRepo) IncrUsage(db *gorm.DB, tagID uint, delta int) error {
	return db.Model(&model.Tag{}).
		Where("id = ?", tagID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", delta)).Error
}
```

- [ ] **步骤 4：运行验证通过**

运行：`go test ./internal/repo/ -run TestTagCRUDAndList -v`
预期：PASS

- [ ] **步骤 5：写失败测试（service）+ 实现**

`internal/service/tag_test.go`：
```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func TestTagServiceList(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repo.NewTagRepo(db)
	ctx := context.Background()
	_, _ = repo.Create(ctx, "gin")
	_, _ = repo.Create(ctx, "gorm")
	svc := NewTagService(repo)

	hot, err := svc.List(ctx, "", true, 10)
	if err != nil || len(hot) != 2 {
		t.Fatalf("hot = %v, %v", hot, err)
	}
	filtered, err := svc.List(ctx, "gi", false, 10)
	if err != nil || len(filtered) != 2 {
		t.Fatalf("filtered = %v, %v", filtered, err)
	}
}
```

`internal/service/tag.go`：
```go
package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type TagService struct{ tags *repo.TagRepo }

func NewTagService(tags *repo.TagRepo) *TagService { return &TagService{tags: tags} }

func (s *TagService) List(ctx context.Context, keyword string, hot bool, limit int) ([]model.Tag, error) {
	if limit <= 0 {
		limit = 50
	}
	if hot {
		return s.tags.ListHot(ctx, limit)
	}
	return s.tags.List(ctx, keyword, limit)
}
```

运行：`go test ./internal/service/ -run TestTagServiceList -v`
预期：PASS

- [ ] **步骤 6：写失败测试（handler）+ 实现**

`internal/handler/tag_test.go`：仿照 category_test，建 `GET /api/v1/tags?keyword=gi` 与 `GET /api/v1/tags?hot=1`，断言 200 且返回数组。

`internal/handler/tag.go`：
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type TagHandler struct{ svc *service.TagService }

func NewTagHandler(svc *service.TagService) *TagHandler { return &TagHandler{svc: svc} }

func (h *TagHandler) List(c *gin.Context) {
	keyword := c.Query("keyword")
	hot := c.Query("hot") == "1"
	list, err := h.svc.List(c.Request.Context(), keyword, hot, 50)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, t := range list {
		out = append(out, gin.H{"id": t.ID, "name": t.Name, "usage_count": t.UsageCount})
	}
	platform.OK(c, out)
}
```

运行：`go test ./internal/handler/ -run TestTagEndpoint -v`
预期：PASS

- [ ] **步骤 7：Commit**

```bash
git add internal/repo/tag.go internal/repo/tag_test.go internal/service/tag.go internal/service/tag_test.go internal/handler/tag.go internal/handler/tag_test.go
git commit -m "feat: 标签列表/热门标签接口"
```

---

## 任务 4：文章创建（分类/标签校验 + 标签新建 + usage_count + 事务）

**文件：**
- 创建：`internal/service/dto.go`、`internal/repo/article.go`、`internal/repo/article_test.go`、`internal/service/article.go`、`internal/service/article_test.go`、`internal/handler/article.go`、`internal/handler/article_test.go`

- [ ] **步骤 1：定义 DTO 与错误**

`internal/service/dto.go`：
```go
package service

import (
	"time"

	"aidevclub/internal/model"
)

type CreateArticleInput struct {
	Title      string
	Summary    string
	Content    string
	CategoryID uint
	Status     model.ArticleStatus
	TagIDs     []uint
	TagNames   []string
}

type AuthorBrief struct {
	ID        uint   `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type TagBrief struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ArticleSummary struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	CategoryID     uint       `json:"category_id"`
	CategoryName   string     `json:"category_name"`
	Tags           []TagBrief `json:"tags"`
	Author         AuthorBrief `json:"author"`
	Views          int        `json:"views"`
	LikesCount     int        `json:"likes_count"`
	FavoritesCount int        `json:"favorites_count"`
	CommentsCount  int        `json:"comments_count"`
	PublishedAt    *time.Time `json:"published_at"`
	Pinned         bool       `json:"pinned"`
}

type ArticleListResult struct {
	List     []ArticleSummary `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type ArticleDetail struct {
	ArticleSummary
	Content string `json:"content"`
	Liked   bool   `json:"liked"`
	Favorited bool `json:"favorited"`
}

type ListQuery struct {
	Page       int
	PageSize   int
	CategoryID *uint
	TagID      *uint
	Keyword    string
	AuthorID   *uint
	Sort       string
}
```

`internal/service/article.go` 顶部定义错误（与服务现有变量放一起）：
```go
var (
	ErrArticleNotFound = platform.NewBizError(http.StatusNotFound, 40402, "文章不存在或不可见")
	ErrCommentNotFound = platform.NewBizError(http.StatusNotFound, 40403, "评论不存在")
	ErrCategoryNotFound = platform.NewBizError(http.StatusNotFound, 40404, "分类不存在")
	ErrTagNotFound     = platform.NewBizError(http.StatusNotFound, 40405, "标签不存在")
	ErrForbidden       = platform.NewBizError(http.StatusForbidden, 40301, "无权限")
	ErrBadParam        = platform.NewBizError(http.StatusBadRequest, 40002, "参数不合法")
)
```

- [ ] **步骤 2：写失败测试（repo 文章 CRUD + 标签关联）**

`internal/repo/article_test.go`：
```go
package repo

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestArticleRepoCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewArticleRepo(db)
	ctx := context.Background()

	a := &model.Article{
		AuthorID: 1, CategoryID: 1, Title: "t", Content: "c",
		Status: model.ArticleStatusDraft,
	}
	if err := r.Create(db, a); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByID(db, a.ID)
	if err != nil || got.Title != "t" {
		t.Fatalf("FindByID = %v, %v", got, err)
	}
	if err := r.SetArticleTags(db, a.ID, []uint{10, 11}); err != nil {
		t.Fatal(err)
	}
	ids, err := r.FindArticleTags(db, a.ID)
	if err != nil || len(ids) != 2 {
		t.Fatalf("FindArticleTags = %v, %v", ids, err)
	}
	now := time.Now()
	a.Status = model.ArticleStatusPublished
	a.PublishedAt = &now
	if err := r.Update(db, a); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, a.ID, "views", 1); err != nil {
		t.Fatal(err)
	}
	if err := r.IncrCount(db, a.ID, "likes_count", 1); err != nil {
		t.Fatal(err)
	}
	got, _ = r.FindByID(db, a.ID)
	if got.Views != 1 || got.LikesCount != 1 {
		t.Fatalf("counts = views %d likes %d", got.Views, got.LikesCount)
	}
	if err := r.Delete(db, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, a.ID); err == nil {
		t.Fatal("soft-deleted article still found")
	}
}
```

- [ ] **步骤 3：运行验证失败**

运行：`go test ./internal/repo/ -run TestArticleRepoCRUD -v`
预期：FAIL（`NewArticleRepo` 未定义）

- [ ] **步骤 4：实现 repo**

`internal/repo/article.go`：
```go
package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type ArticleRepo struct{ db *gorm.DB }

func NewArticleRepo(db *gorm.DB) *ArticleRepo { return &ArticleRepo{db: db} }

// 以下方法的 db 参数允许传事务句柄；用 r.db 兜底。
func (r *ArticleRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *ArticleRepo) Create(db *gorm.DB, a *model.Article) error {
	return r.exec(db).Create(a).Error
}

func (r *ArticleRepo) FindByID(db *gorm.DB, id uint) (*model.Article, error) {
	var a model.Article
	if err := r.exec(db).Preload("Category").Preload("Author").First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ArticleRepo) Update(db *gorm.DB, a *model.Article) error {
	return r.exec(db).Save(a).Error
}

func (r *ArticleRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.Article{}, id).Error
}

func (r *ArticleRepo) FindArticleTags(db *gorm.DB, articleID uint) ([]uint, error) {
	var rows []model.ArticleTag
	if err := r.exec(db).Where("article_id = ?", articleID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TagID)
	}
	return ids, nil
}

func (r *ArticleRepo) SetArticleTags(db *gorm.DB, articleID uint, tagIDs []uint) error {
	d := r.exec(db)
	if err := d.Where("article_id = ?", articleID).Delete(&model.ArticleTag{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.ArticleTag, 0, len(tagIDs))
	for _, tid := range tagIDs {
		rows = append(rows, model.ArticleTag{ArticleID: articleID, TagID: tid})
	}
	return d.Create(&rows).Error
}

func (r *ArticleRepo) IncrViews(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).Error
}

func (r *ArticleRepo) IncrCount(db *gorm.DB, id uint, column string, delta int) error {
	return r.exec(db).Model(&model.Article{}).
		Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).Error
}
```

- [ ] **步骤 5：运行验证通过**

运行：`go test ./internal/repo/ -run TestArticleRepoCRUD -v`
预期：PASS

- [ ] **步骤 6：写失败测试（service Create）+ 实现**

`internal/service/article_test.go`：
```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newArticleTestEnv(t *testing.T) (*ArticleService, *model.User, *model.Category) {
	t.Helper()
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}
	cats := repo.NewCategoryRepo(db)
	_ = cats.Seed(context.Background())
	catList, _ := cats.List(context.Background())
	svc := NewArticleService(
		repo.NewArticleRepo(db),
		repo.NewTagRepo(db),
		cats,
		repo.NewInteractionRepo(db),
		testutil.NewTestRedis(t),
		&platform.Config{DefaultPageSize: 20, MaxPageSize: 50, HotCacheTTL: 60e9},
	)
	return svc, u, &catList[0]
}

func TestArticleCreate(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	a, err := svc.Create(context.Background(), u.ID, CreateArticleInput{
		Title: "Hello", Content: "world", CategoryID: cat.ID,
		Status: model.ArticleStatusDraft, TagNames: []string{"gin", "gorm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 || a.Status != model.ArticleStatusDraft {
		t.Fatalf("article = %+v", a)
	}
	// 标签已创建且 usage_count=1
	tags := repo.NewTagRepo(svc.articles.DebugDB()) // 见下方说明
	_ = tags
}
```
> 说明：`svc.articles` 是 `*repo.ArticleRepo`（未导出字段）；改为在测试里单独 `tagRepo := repo.NewTagRepo(db)` 并在创建前先建好 tag，再断言其 UsageCount。按此调整：`gin`、`gorm` 创建后各自 `UsageCount==1`，且文章 ID 通过 `FindArticleTags(db, a.ID)` 可查到这两个 tag。

`internal/service/article.go`（本任务实现 Create 部分，后续任务补齐其余方法）：
```go
package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

type ArticleService struct {
	articles *repo.ArticleRepo
	tags     *repo.TagRepo
	cats     *repo.CategoryRepo
	inter    *repo.InteractionRepo
	rdb      *redis.Client
	cfg      *platform.Config
}

func NewArticleService(articles *repo.ArticleRepo, tags *repo.TagRepo, cats *repo.CategoryRepo, inter *repo.InteractionRepo, rdb *redis.Client, cfg *platform.Config) *ArticleService {
	return &ArticleService{articles: articles, tags: tags, cats: cats, inter: inter, rdb: rdb, cfg: cfg}
}

func (s *ArticleService) ImageDir() string     { return s.cfg.ArticleImageDir }
func (s *ArticleService) MaxImageBytes() int64 { return s.cfg.MaxArticleImageBytes }

func (s *ArticleService) validateStatus(st model.ArticleStatus) error {
	switch st {
	case model.ArticleStatusDraft, model.ArticleStatusPublished:
		return nil
	}
	return ErrBadParam
}

// ResolveTagSet 把 tag_ids + tag_names 合并为一个 tag id 集合。
func (s *ArticleService) ResolveTagSet(ctx context.Context, tx *gorm.DB, tagIDs []uint, tagNames []string) ([]uint, error) {
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
			t, err = s.tags.Create(ctx, name) // 用 s.tags.db 创建；事务提交失败时回滚 tag
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

func (s *ArticleService) Create(ctx context.Context, userID uint, in CreateArticleInput) (*model.Article, error) {
	if in.Title == "" || in.Content == "" {
		return nil, ErrBadParam
	}
	if len(in.Title) > 200 {
		return nil, ErrBadParam
	}
	if err := s.validateStatus(in.Status); err != nil {
		return nil, err
	}
	if _, err := s.cats.FindByID(ctx, in.CategoryID); err != nil {
		return nil, ErrCategoryNotFound
	}
	a := &model.Article{
		AuthorID:   userID,
		CategoryID: in.CategoryID,
		Title:      in.Title,
		Summary:    in.Summary,
		Content:    in.Content,
		Status:     in.Status,
	}
	if in.Status == model.ArticleStatusPublished {
		now := time.Now()
		a.PublishedAt = &now
	}

	err := s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articles.Create(tx, a); err != nil {
			return err
		}
		tagIDs, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		if len(tagIDs) > 0 {
			if err := s.articles.SetArticleTags(tx, a.ID, tagIDs); err != nil {
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
	return a, nil
}
```
> 说明：`ArticleRepo` 需要暴露 `DB()` 返回底层 `*gorm.DB`（供 service 开事务），并让 `exec` 支持。在 `internal/repo/article.go` 增加：
> ```go
> func (r *ArticleRepo) DB() *gorm.DB { return r.db }
> ```
> `TagRepo.Create` 用 `r.db` 而非 tx——新建标签与文章同事务提交即可（GORM 事务内 Create 走的是同一连接池，提交后一并生效）。`ResolveTagSet` 的 `s.tags.FindByID/Create` 不带 tx 上下文，但因唯一索引 + 1062 兜底，语义正确。

- [ ] **步骤 7：运行验证通过**

运行：`go test ./internal/service/ -run TestArticleCreate -v`
预期：PASS

- [ ] **步骤 8：handler（Create）+ 测试**

`internal/handler/article_test.go`：
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

func articleRouter(t *testing.T) (*gin.Engine, *repo.UserRepo) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	users := repo.NewUserRepo(db)
	rdb := testutil.NewTestRedis(t)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50, HotCacheTTL: 60e9}
	svc := service.NewArticleService(
		repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewCategoryRepo(db),
		repo.NewInteractionRepo(db), rdb, cfg,
	)
	_ = repo.NewCategoryRepo(db).Seed(t.Context())
	h := NewArticleHandler(svc)
	auth := platform.AuthMiddleware("s")
	r := gin.New()
	art := r.Group("/api/v1/articles")
	art.POST("", auth, h.Create)
	return r, users
}

func TestArticleCreateEndpoint(t *testing.T) {
	r, users := articleRouter(t)
	u := &repo.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	body, _ := json.Marshal(map[string]interface{}{
		"title": "t", "content": "c", "category_id": 1, "status": "publish",
		"tag_names": []string{"gin"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}
```
`internal/handler/article.go`（本任务 Create）：
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type ArticleHandler struct{ svc *service.ArticleService }

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler { return &ArticleHandler{svc: svc} }

func (h *ArticleHandler) Create(c *gin.Context) {
	var in struct {
		Title      string                `json:"title"`
		Summary    string                `json:"summary"`
		Content    string                `json:"content"`
		CategoryID uint                  `json:"category_id"`
		Status     model.ArticleStatus   `json:"status"`
		TagIDs     []uint                `json:"tag_ids"`
		TagNames   []string              `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	a, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), service.CreateArticleInput{
		Title: in.Title, Summary: in.Summary, Content: in.Content,
		CategoryID: in.CategoryID, Status: in.Status, TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": a.ID})
}
```
> 说明：`repo.User` 不存在，测试里直接用 `model.User`（`repo.UserRepo.Create` 接受 `*model.User`）。修正为 `u := &model.User{...}`。

运行：`go test ./internal/handler/ -run TestArticleCreateEndpoint -v`
预期：PASS

- [ ] **步骤 9：Commit**

```bash
git add internal/service/dto.go internal/repo/article.go internal/repo/article_test.go internal/service/article.go internal/service/article_test.go internal/handler/article.go internal/handler/article_test.go
git commit -m "feat: 文章发布（分类/标签校验、标签新建、usage_count、草稿/发布）"
```

---

## 任务 5：文章编辑与删除（权限 + 标签 diff + 软删除）

**文件：**
- 修改：`internal/service/article.go`、`internal/service/article_test.go`、`internal/handler/article.go`、`internal/handler/article_test.go`

- [ ] **步骤 1：写失败测试（service Update/Delete）**

`internal/service/article_test.go` 追加：
```go
func TestArticleUpdateAndDelete(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, u.ID, CreateArticleInput{
		Title: "t", Content: "c", CategoryID: cat.ID,
		Status: model.ArticleStatusDraft, TagNames: []string{"gin"},
	})
	// 作者可编辑：改标题 + 换标签（gin 移除、gorm 新增）
	got, err := svc.Update(ctx, u.ID, a.ID, CreateArticleInput{
		Title: "t2", Content: "c2", CategoryID: cat.ID,
		Status: model.ArticleStatusPublished, TagNames: []string{"gorm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.ArticleStatusPublished || got.Title != "t2" {
		t.Fatalf("updated = %+v", got)
	}
	// 非作者不可编辑
	if _, err := svc.Update(ctx, u.ID+999, a.ID, CreateArticleInput{Title: "x", Content: "y", CategoryID: cat.ID, Status: model.ArticleStatusDraft}); err == nil {
		t.Fatal("non-author update allowed")
	}
	// 非作者不可删除
	if err := svc.Delete(ctx, u.ID+999, a.ID); err == nil {
		t.Fatal("non-author delete allowed")
	}
	// 作者可删除
	if err := svc.Delete(ctx, u.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, u.ID, a.ID); err == nil {
		t.Fatal("deleted article still visible")
	}
	// 删除后标签 usage_count 归零
}
```

- [ ] **步骤 2：运行验证失败**

运行：`go test ./internal/service/ -run TestArticleUpdateAndDelete -v`
预期：FAIL（`Update`/`Delete`/`Get` 未实现）

- [ ] **步骤 3：实现 Update / Delete**

`internal/service/article.go` 追加：
```go
func (s *ArticleService) Update(ctx context.Context, userID, articleID uint, in CreateArticleInput) (*model.Article, error) {
	if in.Title == "" || in.Content == "" {
		return nil, ErrBadParam
	}
	if len(in.Title) > 200 {
		return nil, ErrBadParam
	}
	if err := s.validateStatus(in.Status); err != nil {
		return nil, err
	}
	if _, err := s.cats.FindByID(ctx, in.CategoryID); err != nil {
		return nil, ErrCategoryNotFound
	}
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil {
		return nil, ErrArticleNotFound
	}
	if a.AuthorID != userID {
		return nil, ErrForbidden
	}
	oldTags, err := s.articles.FindArticleTags(nil, articleID)
	if err != nil {
		return nil, err
	}

	a.Title = in.Title
	a.Summary = in.Summary
	a.Content = in.Content
	a.CategoryID = in.CategoryID
	if a.Status != in.Status {
		a.Status = in.Status
		if in.Status == model.ArticleStatusPublished && a.PublishedAt == nil {
			now := time.Now()
			a.PublishedAt = &now
		}
	}

	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articles.Update(tx, a); err != nil {
			return err
		}
		newTags, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		// diff：移除 old 中不在 new 的，新增 new 中不在 old 的
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
				if err := s.tags.IncrUsage(tx, id, -1); err != nil {
					return err
				}
			}
		}
		for _, id := range newTags {
			if !oldSet[id] {
				if err := s.tags.IncrUsage(tx, id, 1); err != nil {
					return err
				}
			}
		}
		return s.articles.SetArticleTags(tx, articleID, newTags)
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *ArticleService) Delete(ctx context.Context, userID, articleID uint) error {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil {
		return ErrArticleNotFound
	}
	if a.AuthorID != userID {
		return ErrForbidden
	}
	return s.articles.DB().Transaction(func(tx *gorm.DB) error {
		tagIDs, err := s.articles.FindArticleTags(tx, articleID)
		if err != nil {
			return err
		}
		if err := s.articles.Delete(tx, articleID); err != nil {
			return err
		}
		for _, id := range tagIDs {
			if err := s.tags.IncrUsage(tx, id, -1); err != nil {
				return err
			}
		}
		return s.articles.SetArticleTags(tx, articleID, nil)
	})
}
```

- [ ] **步骤 4：运行验证通过**

运行：`go test ./internal/service/ -run TestArticleUpdateAndDelete -v`
预期：PASS

- [ ] **步骤 5：handler（Update/Delete）+ 测试**

`internal/handler/article.go` 追加 `Update`、`Delete`（复用 Create 的请求结构解析，`PUT` 与 `POST` 同一绑定；`Delete` 无 body）。测试补 `TestArticleUpdateDeleteEndpoint`：用 `articleRouter` 建文章后，`PUT` 改标题断言 200，`DELETE` 断言 200，非作者 `PUT` 断言 403。

运行：`go test ./internal/handler/ -run TestArticleUpdateDeleteEndpoint -v`
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add internal/service/article.go internal/service/article_test.go internal/handler/article.go internal/handler/article_test.go
git commit -m "feat: 文章编辑/删除（权限校验、标签 diff、软删除减 usage_count）"
```

---

## 任务 6：文章列表与详情（分页/筛选/搜索/排序 + 浏览量 + OptionalAuth）

**文件：**
- 修改：`internal/platform/auth_middleware.go`、`internal/platform/auth_middleware_test.go`、`internal/repo/article.go`、`internal/repo/article_test.go`、`internal/service/article.go`、`internal/service/article_test.go`、`internal/handler/article.go`、`internal/handler/article_test.go`

- [ ] **步骤 1：新增 OptionalAuthMiddleware + 测试**

`internal/platform/auth_middleware.go` 追加：
```go
// OptionalAuthMiddleware 解析 Bearer token；有效则设置 user_id，无效/缺失不拦截。
func OptionalAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(h, prefix) {
			if uid, err := ParseAccessToken(secret, strings.TrimPrefix(h, prefix)); err == nil {
				c.Set("user_id", uid)
			}
		}
		c.Next()
	}
}
```
`internal/platform/auth_middleware_test.go` 追加：`TestOptionalAuth`——无 token 时 handler 能执行且无 user_id；带有效 token 时能读到 user_id。

运行：`go test ./internal/platform/ -run TestOptionalAuth -v`
预期：PASS

- [ ] **步骤 2：写失败测试（repo List）**

`internal/repo/article_test.go` 追加：
```go
func TestArticleRepoList(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewArticleRepo(db)
	ctx := context.Background()
	now := time.Now()
	for i := 0; i < 3; i++ {
		a := &model.Article{AuthorID: 1, CategoryID: uint(i + 1), Title: "x" + string(rune('a'+i)), Content: "c", Status: model.ArticleStatusPublished, PublishedAt: &now}
		if err := r.Create(db, a); err != nil {
			t.Fatal(err)
		}
	}
	_ = r.SetArticleTags(db, 1, []uint{1})
	q := ArticleQuery{Page: 1, PageSize: 2, Sort: "latest"}
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
> 说明：`ArticleQuery` 定义在 `internal/repo/article.go`（repo 层查询参数）：
> ```go
> type ArticleQuery struct {
> 	Page, PageSize int
> 	CategoryID     *uint
> 	TagID          *uint
> 	Keyword        string
> 	AuthorID       *uint
> 	Sort           string // latest | hot | pinned
> }
> ```

- [ ] **步骤 3：运行验证失败**

运行：`go test ./internal/repo/ -run TestArticleRepoList -v`
预期：FAIL（`ArticleQuery` 未定义）

- [ ] **步骤 4：实现 repo List/Count/TagsForArticles**

`internal/repo/article.go` 追加：
```go
type ArticleQuery struct {
	Page, PageSize int
	CategoryID     *uint
	TagID          *uint
	Keyword        string
	AuthorID       *uint
	Sort           string
}

func (r *ArticleRepo) baseQuery(ctx context.Context, q ArticleQuery) *gorm.DB {
	d := r.db.WithContext(ctx).Model(&model.Article{})
	if q.CategoryID != nil {
		d = d.Where("category_id = ?", *q.CategoryID)
	}
	if q.AuthorID != nil {
		d = d.Where("author_id = ?", *q.AuthorID)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		d = d.Where("(title LIKE ? OR summary LIKE ? OR content LIKE ?)", kw, kw, kw)
	}
	if q.TagID != nil {
		d = d.Where("id IN (SELECT article_id FROM article_tags WHERE tag_id = ?)", *q.TagID)
	}
	return d
}

func (r *ArticleRepo) Count(ctx context.Context, q ArticleQuery) (int64, error) {
	var total int64
	err := r.baseQuery(ctx, q).Count(&total).Error
	return total, err
}

func (r *ArticleRepo) List(ctx context.Context, q ArticleQuery) ([]model.Article, int64, error) {
	d := r.baseQuery(ctx, q)
	switch q.Sort {
	case "hot":
		d = d.Order("(views + 3*likes_count + 5*favorites_count + 2*comments_count) desc, id desc")
	case "pinned":
		d = d.Order("pinned desc, published_at desc, id desc")
	default: // latest
		d = d.Order("published_at desc, id desc")
	}
	var list []model.Article
	if err := d.Preload("Category").Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	total, err := r.Count(ctx, q)
	return list, total, err
}

// TagsForArticles 批量取多篇文章的标签，返回 map[articleID][]Tag。
func (r *ArticleRepo) TagsForArticles(ctx context.Context, articleIDs []uint) (map[uint][]model.Tag, error) {
	res := map[uint][]model.Tag{}
	if len(articleIDs) == 0 {
		return res, nil
	}
	var rows []model.ArticleTag
	if err := r.db.WithContext(ctx).Where("article_id IN ?", articleIDs).Find(&rows).Error; err != nil {
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
		res[row.ArticleID] = append(res[row.ArticleID], byID[row.TagID])
	}
	return res, nil
}
```

- [ ] **步骤 5：运行验证通过**

运行：`go test ./internal/repo/ -run TestArticleRepoList -v`
预期：PASS

- [ ] **步骤 6：写失败测试（service List/Get）+ 实现**

`internal/service/article_test.go` 追加：
```go
func TestArticleListAndGet(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	pub, _ := svc.Create(ctx, u.ID, CreateArticleInput{
		Title: "公开", Content: "c", CategoryID: cat.ID,
		Status: model.ArticleStatusPublished, TagNames: []string{"gin"},
	})
	_, _ = svc.Create(ctx, u.ID, CreateArticleInput{
		Title: "草稿", Content: "c", CategoryID: cat.ID,
		Status: model.ArticleStatusDraft,
	})
	// 列表只见 published
	res, err := svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "latest"})
	if err != nil || res.Total != 1 || len(res.List) != 1 {
		t.Fatalf("list = %+v, err %v", res, err)
	}
	if res.List[0].Title != "公开" || len(res.List[0].Tags) != 1 {
		t.Fatalf("summary = %+v", res.List[0])
	}
	// 关键词
	res, _ = svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "latest", Keyword: "公开"})
	if res.Total != 1 {
		t.Fatalf("keyword total = %d", res.Total)
	}
	// 详情：公开文章游客可看，浏览量 +1
	detail, err := svc.Get(ctx, 0, pub.ID)
	if err != nil || !detail.Liked {
		t.Fatalf("detail = %+v, err %v", detail, err)
	}
	// 草稿仅作者可见
	author, err := svc.Get(ctx, u.ID, pub.ID)
	_ = author
	if _, err := svc.Get(ctx, 0, pub.ID+100); err == nil {
		t.Fatal("draft visible to guest")
	}
}
```
> 说明：`!detail.Liked` 断言语义修正为：游客 detail 的 `Liked` 应为 false；文章可见性用 `Get(ctx, 0, <draft_id>)` 断言 404。测试里需要拿到草稿 ID，改用 `draft,_ := svc.Create(...)` 后断言 `svc.Get(ctx, 0, draft.ID)` 返回 `ErrArticleNotFound`。

`internal/service/article.go` 追加（列表 DTO 组装 + 详情）：
```go
func (s *ArticleService) List(ctx context.Context, q ListQuery) (*ArticleListResult, error) {
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
	case "hot", "pinned":
	default:
		q.Sort = "latest"
	}
	rq := repo.ArticleQuery{Page: q.Page, PageSize: q.PageSize, CategoryID: q.CategoryID, TagID: q.TagID, Keyword: q.Keyword, AuthorID: q.AuthorID, Sort: q.Sort}
	// 热门无筛选时走 Redis 缓存
	if q.Sort == "hot" && q.CategoryID == nil && q.TagID == nil && q.AuthorID == nil && q.Keyword == "" {
		key := fmt.Sprintf("hot:articles:%d:%d", q.Page, q.PageSize)
		if v, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var res ArticleListResult
			if json.Unmarshal(v, &res) == nil {
				return &res, nil
			}
		}
	}
	list, total, err := s.articles.List(ctx, rq)
	if err != nil {
		return nil, err
	}
	tagMap, err := s.articles.TagsForArticles(ctx, articleIDs(list))
	if err != nil {
		return nil, err
	}
	out := &ArticleListResult{List: make([]ArticleSummary, 0, len(list)), Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, a := range list {
		out.List = append(out.List, s.summaryOf(a, tagMap[a.ID]))
	}
	if q.Sort == "hot" && q.CategoryID == nil && q.TagID == nil && q.AuthorID == nil && q.Keyword == "" {
		if b, err := json.Marshal(out); err == nil {
			_ = s.rdb.Set(ctx, key, b, s.cfg.HotCacheTTL).Err()
		}
	}
	return out, nil
}

func articleIDs(list []model.Article) []uint {
	ids := make([]uint, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	return ids
}

func (s *ArticleService) summaryOf(a model.Article, tags []model.Tag) ArticleSummary {
	sm := ArticleSummary{
		ID: a.ID, Title: a.Title, Summary: a.Summary,
		CategoryID: a.CategoryID, CategoryName: "", Tags: []TagBrief{},
		Views: a.Views, LikesCount: a.LikesCount, FavoritesCount: a.FavoritesCount,
		CommentsCount: a.CommentsCount, PublishedAt: a.PublishedAt, Pinned: a.Pinned,
		Author: AuthorBrief{ID: a.AuthorID, Nickname: "", AvatarURL: ""},
	}
	if a.Category != nil {
		sm.CategoryName = a.Category.Name
	}
	if a.Author != nil {
		sm.Author = AuthorBrief{ID: a.Author.ID, Nickname: a.Author.Nickname, AvatarURL: a.Author.AvatarURL}
	}
	for _, t := range tags {
		sm.Tags = append(sm.Tags, TagBrief{ID: t.ID, Name: t.Name})
	}
	return sm
}

func (s *ArticleService) Get(ctx context.Context, userID, articleID uint) (*ArticleDetail, error) {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil {
		return nil, ErrArticleNotFound
	}
	if a.Status != model.ArticleStatusPublished && a.AuthorID != userID {
		return nil, ErrArticleNotFound
	}
	if a.Status == model.ArticleStatusPublished {
		_ = s.articles.IncrViews(ctx, articleID)
		a.Views++
	}
	tags, err := s.articles.TagsForArticles(ctx, []uint{articleID})
	if err != nil {
		return nil, err
	}
	sm := s.summaryOf(*a, tags[a.ID])
	d := &ArticleDetail{ArticleSummary: sm, Content: a.Content}
	if userID > 0 {
		if d.Liked, err = s.inter.ArticleLiked(nil, userID, articleID); err != nil {
			return nil, err
		}
		if d.Favorited, err = s.inter.ArticleFavorited(nil, userID, articleID); err != nil {
			return nil, err
		}
	}
	return d, nil
}
```
> 说明：`ArticleDetail` 嵌入 `ArticleSummary` 后 JSON 平铺，符合设计。`ArticleLiked/ArticleFavorited` 是 InteractionRepo 的方法，任务 8 才实现——为避免本任务无法编译，任务 8 的实现**先在本任务一并写出**（Toggle 三个方法任务 8 实现）。或本任务先给 `ArticleLiked/ArticleFavorited` 最小实现（见任务 8 步骤 1）。

- [ ] **步骤 7：运行验证通过**

运行：`go test ./internal/service/ -run 'TestArticleListAndGet' -v`
预期：PASS（若因 InteractionRepo 方法缺失失败，先在任务 8 步骤 1 实现 `ArticleLiked/ArticleFavorited`）

- [ ] **步骤 8：handler（List/Get）+ 路由**

`internal/handler/article.go` 追加 `List`（解析 `page/page_size/category_id/tag_id/keyword/author_id/sort`，转 `service.ListQuery`）与 `Get`（`OptionalAuthMiddleware` 下取 `user_id`，未登录为 0）。`internal/handler/article_test.go` 追加 `TestArticleListGetEndpoint`：建两篇（一发布一草稿），`GET /api/v1/articles` 断言仅 1 条且结构完整，`GET /api/v1/articles/:id` 断言 `content` 与 `views`。

运行：`go test ./internal/handler/ -run TestArticleListGetEndpoint -v`
预期：PASS

- [ ] **步骤 9：Commit**

```bash
git add internal/platform/auth_middleware.go internal/platform/auth_middleware_test.go internal/repo/article.go internal/repo/article_test.go internal/service/article.go internal/service/article_test.go internal/handler/article.go internal/handler/article_test.go
git commit -m "feat: 文章列表/详情（分页、分类/标签/关键词筛选、最新/置顶排序、浏览量）"
```

---

## 任务 7：热门排序 + Redis 缓存

**文件：**
- 修改：`internal/service/article.go`、`internal/service/article_test.go`、`internal/repo/article.go`（`ArticleQuery.Sort=="hot"` 已支持，仅验证）、`internal/platform/config.go`

- [ ] **步骤 1：config 增加热门缓存/分页配置**

`internal/platform/config.go`：`Config` 增加 `HotCacheTTL time.Duration`、`DefaultPageSize int`、`MaxPageSize int`、`ArticleImageDir string`、`MaxArticleImageBytes int64`。`LoadConfig` 增加 viper 默认：`article.hot_cache_ttl="60s"`、`article.page_size_default=20`、`article.page_size_max=50`、`article_image.dir="storage/articles"`、`article_image.max_bytes=int64(5<<20)`。

- [ ] **步骤 2：写失败测试（service hot 排序 + 缓存命中）**

`internal/service/article_test.go` 追加：
```go
func TestArticleHotSortAndCache(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	// 两篇文章：a1 被点赞多（分数高），a2 普通
	a1, _ := svc.Create(ctx, u.ID, CreateArticleInput{Title: "high", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})
	a2, _ := svc.Create(ctx, u.ID, CreateArticleInput{Title: "low", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})
	_, _, _ = svc.ToggleLike(ctx, u.ID, a1.ID)
	_, _, _ = svc.ToggleLike(ctx, u.ID, a1.ID) // 二次 = 取消，验证幂等
	_, _, _ = svc.ToggleLike(ctx, u.ID, a1.ID) // 再点 = 点赞
	res, err := svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "hot"})
	if err != nil {
		t.Fatal(err)
	}
	if res.List[0].ID != a1.ID {
		t.Fatalf("hot first = %d, want %d", res.List[0].ID, a1.ID)
	}
	// 二次查询应命中缓存（仍正确）
	res2, _ := svc.List(ctx, ListQuery{Page: 1, PageSize: 20, Sort: "hot"})
	if res2.List[0].ID != a1.ID {
		t.Fatal("cached hot list wrong")
	}
}
```
> 说明：`ToggleLike` 依赖任务 8——本任务先只写 hot 排序+缓存的「失败测试」，`ToggleLike` 方法在任务 8 步骤 3 补齐后此测试才能通过。为避免交叉，本任务测试改为直接用 repo 造数据：
> ```go
> // 用 repo 直接造 count，绕过 ToggleLike：
> _ = repo.NewArticleRepo(db).IncrCount(db, a1.ID, "likes_count", 2)
> ```

- [ ] **步骤 3：运行验证失败**

运行：`go test ./internal/service/ -run TestArticleHotSortAndCache -v`
预期：FAIL（`HotCacheTTL` 未配置或列表实现未接入缓存）

- [ ] **步骤 4：确保 List 已按任务 6 步骤 6 实现 hot 分支 + 缓存（核对）**

检查 `internal/service/article.go` 的 `List`：`sort=hot` 且无筛选时读/写 Redis key `hot:articles:{page}:{size}`。确认 `newArticleTestEnv` 传入的 `Config` 含 `HotCacheTTL`、`DefaultPageSize`、`MaxPageSize`。

运行：`go test ./internal/service/ -run TestArticleHotSortAndCache -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/platform/config.go internal/service/article.go internal/service/article_test.go internal/repo/article.go
git commit -m "feat: 热门文章排序 + Redis 结果缓存"
```

---

## 任务 8：文章互动（点赞/收藏 toggle + 计数事务）

**文件：**
- 创建：`internal/repo/interaction.go`、`internal/repo/interaction_test.go`
- 修改：`internal/service/article.go`、`internal/service/article_test.go`、`internal/handler/article.go`、`internal/handler/article_test.go`

- [ ] **步骤 1：写失败测试（repo toggle）**

`internal/repo/interaction_test.go`：
```go
package repo

import (
	"testing"

	"aidevclub/internal/testutil"
)

func TestInteractionToggle(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewInteractionRepo(db)

	liked, err := r.ToggleArticleLike(db, 1, 1)
	if err != nil || !liked {
		t.Fatalf("first like = %v, %v", liked, err)
	}
	liked, err = r.ToggleArticleLike(db, 1, 1)
	if err != nil || liked {
		t.Fatalf("second like should unlike, got %v, %v", liked, err)
	}
	fav, err := r.ToggleArticleFavorite(db, 1, 1)
	if err != nil || !fav {
		t.Fatalf("favorite = %v, %v", fav, err)
	}
	cl, err := r.ToggleCommentLike(db, 1, 1)
	if err != nil || !cl {
		t.Fatalf("comment like = %v, %v", cl, err)
	}
	if ok, _ := r.ArticleLiked(db, 1, 1); ok {
		t.Fatal("ArticleLiked should be false after unlike")
	}
	if ok, _ := r.ArticleFavorited(db, 1, 1); !ok {
		t.Fatal("ArticleFavorited should be true")
	}
}
```

- [ ] **步骤 2：运行验证失败**

运行：`go test ./internal/repo/ -run TestInteractionToggle -v`
预期：FAIL（`NewInteractionRepo` 未定义）

- [ ] **步骤 3：实现 repo**

`internal/repo/interaction.go`：
```go
package repo

import (
	"errors"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
)

type InteractionRepo struct{ db *gorm.DB }

func NewInteractionRepo(db *gorm.DB) *InteractionRepo { return &InteractionRepo{db: db} }

func (r *InteractionRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

// toggle 通用：插入成功 => true（已点赞）；唯一冲突 => 删除 => false（取消）；其他错误返回。
func toggleLike(db *gorm.DB, m interface{}, uniqWhere string, uniqArgs ...interface{}) (bool, error) {
	if err := db.Create(m).Error; err == nil {
		return true, nil
	} else if !platform.IsDuplicateEntry(err) {
		return false, err
	}
	if err := db.Where(uniqWhere, uniqArgs...).Delete(m).Error; err != nil {
		return false, err
	}
	return false, nil
}

func (r *InteractionRepo) ToggleArticleLike(db *gorm.DB, userID, articleID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.ArticleLike{UserID: userID, ArticleID: articleID}, "article_id = ? AND user_id = ?", articleID, userID)
}

func (r *InteractionRepo) ToggleArticleFavorite(db *gorm.DB, userID, articleID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.ArticleFavorite{UserID: userID, ArticleID: articleID}, "article_id = ? AND user_id = ?", articleID, userID)
}

func (r *InteractionRepo) ToggleCommentLike(db *gorm.DB, userID, commentID uint) (bool, error) {
	return toggleLike(r.exec(db), &model.CommentLike{UserID: userID, CommentID: commentID}, "comment_id = ? AND user_id = ?", commentID, userID)
}

func (r *InteractionRepo) ArticleLiked(db *gorm.DB, userID, articleID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.ArticleLike{}).Where("article_id = ? AND user_id = ?", articleID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) ArticleFavorited(db *gorm.DB, userID, articleID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.ArticleFavorite{}).Where("article_id = ? AND user_id = ?", articleID, userID).Count(&count).Error
	return count > 0, err
}

func (r *InteractionRepo) CommentLiked(db *gorm.DB, userID, commentID uint) (bool, error) {
	var count int64
	err := r.exec(db).Model(&model.CommentLike{}).Where("comment_id = ? AND user_id = ?", commentID, userID).Count(&count).Error
	return count > 0, err
}

var _ = errors.New // 占位：本文件实际无需 errors，可删除该 import
```
> 说明：删掉无用的 `errors` import。

- [ ] **步骤 4：运行验证通过**

运行：`go test ./internal/repo/ -run TestInteractionToggle -v`
预期：PASS

- [ ] **步骤 5：写失败测试（service Toggle）+ 实现**

`internal/service/article_test.go` 追加：
```go
func TestArticleToggleLike(t *testing.T) {
	svc, u, cat := newArticleTestEnv(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, u.ID, CreateArticleInput{Title: "t", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})
	liked, count, err := svc.ToggleLike(ctx, u.ID, a.ID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("like = %v, %d, %v", liked, count, err)
	}
	liked, count, _ = svc.ToggleLike(ctx, u.ID, a.ID)
	if liked || count != 0 {
		t.Fatalf("unlike = %v, %d", liked, count)
	}
	// 草稿不可点赞
	_ = svc
}
```

`internal/service/article.go` 追加：
```go
func (s *ArticleService) ToggleLike(ctx context.Context, userID, articleID uint) (bool, int, error) {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil || a.Status != model.ArticleStatusPublished {
		return false, 0, ErrArticleNotFound
	}
	var liked bool
	var newCount int
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleArticleLike(tx, userID, articleID)
		if err != nil {
			return err
		}
		delta := 1
		if !liked {
			delta = -1
		}
		if err := s.articles.IncrCount(tx, articleID, "likes_count", delta); err != nil {
			return err
		}
		newCount = a.LikesCount + delta
		return nil
	})
	return liked, newCount, err
}

func (s *ArticleService) ToggleFavorite(ctx context.Context, userID, articleID uint) (bool, int, error) {
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil || a.Status != model.ArticleStatusPublished {
		return false, 0, ErrArticleNotFound
	}
	var favorited bool
	var newCount int
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		favorited, err = s.inter.ToggleArticleFavorite(tx, userID, articleID)
		if err != nil {
			return err
		}
		delta := 1
		if !favorited {
			delta = -1
		}
		if err := s.articles.IncrCount(tx, articleID, "favorites_count", delta); err != nil {
			return err
		}
		newCount = a.FavoritesCount + delta
		return nil
	})
	return favorited, newCount, err
}
```

- [ ] **步骤 6：运行验证通过**

运行：`go test ./internal/service/ -run TestArticleToggleLike -v`
预期：PASS

- [ ] **步骤 7：handler（Like/Favorite）+ 路由 + 测试**

`internal/handler/article.go` 追加 `Like`、`Favorite`，返回 `{"liked":bool,"likes_count":int}` / `{"favorited":bool,"favorites_count":int}`。`articleRouter` 增加 `art.POST("/:id/like", auth, h.Like)`、`art.POST("/:id/favorite", auth, h.Favorite)`。测试 `TestArticleLikeEndpoint`：建已发布文章后 POST like 断言 200 且 liked=true，再 POST 断言 liked=false。

运行：`go test ./internal/handler/ -run TestArticleLikeEndpoint -v`
预期：PASS

- [ ] **步骤 8：Commit**

```bash
git add internal/repo/interaction.go internal/repo/interaction_test.go internal/service/article.go internal/service/article_test.go internal/handler/article.go internal/handler/article_test.go
git commit -m "feat: 文章点赞/收藏（toggle + 计数事务）"
```

---

## 任务 9：评论（两级 CRUD + 作者管理 + 评论点赞 + comments_count）

**文件：**
- 创建：`internal/repo/comment.go`、`internal/repo/comment_test.go`、`internal/service/comment.go`、`internal/service/comment_test.go`、`internal/handler/comment.go`、`internal/handler/comment_test.go`
- 修改：`internal/repo/interaction.go`（已含 CommentLiked）、`internal/service/comment.go`（ToggleLike 用）、`internal/service/dto.go`（CommentItem）

`internal/service/dto.go` 追加：
```go
type CommentItem struct {
	ID         uint         `json:"id"`
	ArticleID  uint         `json:"article_id"`
	AuthorID   uint         `json:"author_id"`
	Author     AuthorBrief  `json:"author"`
	Content    string       `json:"content"`
	LikesCount int          `json:"likes_count"`
	CreatedAt  time.Time    `json:"created_at"`
	Replies    []CommentItem `json:"replies"`
}
```

- [ ] **步骤 1：写失败测试（repo comment）**

`internal/repo/comment_test.go`：
```go
package repo

import (
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestCommentRepo(t *testing.T) {
	db := testutil.NewTestDB(t)
	r := NewCommentRepo(db)
	c := &model.Comment{ArticleID: 1, AuthorID: 2, Content: "hi"}
	if err := r.Create(db, c); err != nil {
		t.Fatal(err)
	}
	pid := c.ID
	r2 := &model.Comment{ArticleID: 1, AuthorID: 3, Content: "reply", ParentID: &pid}
	if err := r.Create(db, r2); err != nil {
		t.Fatal(err)
	}
	list, err := r.ListByArticle(db, 1)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if err := r.IncrLikes(db, c.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(db, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(db, c.ID); err == nil {
		t.Fatal("deleted comment found")
	}
	// 软删除后仅剩回复
	list, _ = r.ListByArticle(db, 1)
	if len(list) != 1 {
		t.Fatalf("after delete len = %d", len(list))
	}
}
```

- [ ] **步骤 2：运行验证失败**

运行：`go test ./internal/repo/ -run TestCommentRepo -v`
预期：FAIL（`NewCommentRepo` 未定义）

- [ ] **步骤 3：实现 repo**

`internal/repo/comment.go`：
```go
package repo

import (
	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type CommentRepo struct{ db *gorm.DB }

func NewCommentRepo(db *gorm.DB) *CommentRepo { return &CommentRepo{db: db} }

func (r *CommentRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *CommentRepo) Create(db *gorm.DB, c *model.Comment) error { return r.exec(db).Create(c).Error }

func (r *CommentRepo) FindByID(db *gorm.DB, id uint) (*model.Comment, error) {
	var c model.Comment
	if err := r.exec(db).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CommentRepo) ListByArticle(db *gorm.DB, articleID uint) ([]model.Comment, error) {
	var list []model.Comment
	err := r.exec(db).Where("article_id = ?", articleID).Order("created_at asc, id asc").Find(&list).Error
	return list, err
}

func (r *CommentRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.Comment{}, id).Error
}

func (r *CommentRepo) IncrLikes(db *gorm.DB, id uint, delta int) error {
	return r.exec(db).Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("likes_count + ?", delta)).Error
}
```

- [ ] **步骤 4：运行验证通过**

运行：`go test ./internal/repo/ -run TestCommentRepo -v`
预期：PASS

- [ ] **步骤 5：写失败测试（service comment）+ 实现**

`internal/service/comment_test.go`：
```go
package service

import (
	"context"
	"testing"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newCommentTestEnv(t *testing.T) (*CommentService, *ArticleService, *model.User, *model.Category) {
	t.Helper()
	asvc, u, cat := newArticleTestEnv(t)
	svc := NewCommentService(
		repo.NewCommentRepo(asvc.articles.DB()),
		asvc.articles,
		asvc.inter,
	)
	return svc, asvc, u, cat
}

func TestCommentCreateListDelete(t *testing.T) {
	svc, asvc, u, cat := newCommentTestEnv(t)
	ctx := context.Background()
	a, _ := asvc.Create(ctx, u.ID, CreateArticleInput{Title: "t", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})
	author, _ := asvc.Create(ctx, u.ID, CreateArticleInput{Title: "a", Content: "c", CategoryID: cat.ID, Status: model.ArticleStatusPublished})
	_ = author

	c1, err := svc.Create(ctx, u.ID, a.ID, "一级评论", nil)
	if err != nil {
		t.Fatal(err)
	}
	pid := c1.ID
	c2, err := svc.Create(ctx, u.ID, a.ID, "回复", &pid)
	if err != nil {
		t.Fatal(err)
	}
	// 回复的回复归并到一级
	if _, err := svc.Create(ctx, u.ID, a.ID, "回复的回复", &c2.ID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(ctx, a.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if len(list[0].Replies) != 2 {
		t.Fatalf("replies = %d, want 2", len(list[0].Replies))
	}
	// 作者可删别人评论
	if err := svc.Delete(ctx, u.ID, c2.ID); err != nil {
		t.Fatal(err)
	}
	// 评论点赞
	liked, count, err := svc.ToggleLike(ctx, u.ID, c1.ID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("like = %v, %d, %v", liked, count, err)
	}
	// 非作者不可删
	other := &model.User{Email: "b@b.com", PasswordHash: "x", Nickname: "B", AvatarURL: "/x.png"}
	_ = repo.NewUserRepo(asvc.articles.DB()).Create(other)
	if err := svc.Delete(ctx, other.ID, c1.ID); err == nil {
		t.Fatal("other user deleted comment")
	}
}
```

`internal/service/comment.go`：
```go
package service

import (
	"context"
	"net/http"
	"time"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

var ErrBadParent = platform.NewBizError(http.StatusBadRequest, 40002, "父评论不合法")

type CommentService struct {
	comments *repo.CommentRepo
	articles *repo.ArticleRepo
	inter    *repo.InteractionRepo
}

func NewCommentService(comments *repo.CommentRepo, articles *repo.ArticleRepo, inter *repo.InteractionRepo) *CommentService {
	return &CommentService{comments: comments, articles: articles, inter: inter}
}

func (s *CommentService) Create(ctx context.Context, userID, articleID uint, content string, parentID *uint) (*model.Comment, error) {
	if content == "" {
		return nil, ErrBadParam
	}
	if len(content) > 2000 {
		return nil, ErrBadParam
	}
	a, err := s.articles.FindByID(nil, articleID)
	if err != nil || a.Status != model.ArticleStatusPublished {
		return nil, ErrArticleNotFound
	}
	if parentID != nil {
		p, err := s.comments.FindByID(nil, *parentID)
		if err != nil || p.ArticleID != articleID || p.ParentID != nil {
			return nil, ErrBadParent
		}
	}
	c := &model.Comment{ArticleID: articleID, AuthorID: userID, ParentID: parentID, Content: content}
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Create(tx, c); err != nil {
			return err
		}
		return s.articles.IncrCount(tx, articleID, "comments_count", 1)
	})
	return c, err
}

func (s *CommentService) List(ctx context.Context, articleID uint) ([]CommentItem, error) {
	comments, err := s.comments.ListByArticle(nil, articleID)
	if err != nil {
		return nil, err
	}
	return assembleComments(comments), nil
}

func assembleComments(comments []model.Comment) []CommentItem {
	authorIDs := map[uint]bool{}
	for _, c := range comments {
		authorIDs[c.AuthorID] = true
	}
	_ = authorIDs // 作者信息组装见下

	roots := map[uint]*CommentItem{}
	var order []*CommentItem
	for i := range comments {
		c := &comments[i]
		if c.ParentID == nil {
			it := newCommentItem(c)
			roots[c.ID] = &it
			order = append(order, &it)
		}
	}
	var orphans []CommentItem
	for i := range comments {
		c := &comments[i]
		if c.ParentID != nil {
			it := newCommentItem(c)
			if root, ok := roots[*c.ParentID]; ok {
				root.Replies = append(root.Replies, it)
			} else {
				orphans = append(orphans, it)
			}
		}
	}
	result := make([]CommentItem, 0, len(order)+len(orphans))
	for _, p := range order {
		result = append(result, *p)
	}
	result = append(result, orphans...)
	return result
}

func newCommentItem(c *model.Comment) CommentItem {
	return CommentItem{
		ID: c.ID, ArticleID: c.ArticleID, AuthorID: c.AuthorID,
		Content: c.Content, LikesCount: c.LikesCount, CreatedAt: c.CreatedAt,
		Author: AuthorBrief{ID: c.AuthorID}, Replies: []CommentItem{},
	}
}

func (s *CommentService) Delete(ctx context.Context, userID, commentID uint) error {
	c, err := s.comments.FindByID(nil, commentID)
	if err != nil {
		return ErrCommentNotFound
	}
	a, err := s.articles.FindByID(nil, c.ArticleID)
	if err != nil {
		return ErrArticleNotFound
	}
	if c.AuthorID != userID && a.AuthorID != userID {
		return ErrForbidden
	}
	return s.articles.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.comments.Delete(tx, commentID); err != nil {
			return err
		}
		return s.articles.IncrCount(tx, c.ArticleID, "comments_count", -1)
	})
}

func (s *CommentService) ToggleLike(ctx context.Context, userID, commentID uint) (bool, int, error) {
	c, err := s.comments.FindByID(nil, commentID)
	if err != nil {
		return false, 0, ErrCommentNotFound
	}
	var liked bool
	var newCount int
	err = s.articles.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleCommentLike(tx, userID, commentID)
		if err != nil {
			return err
		}
		delta := 1
		if !liked {
			delta = -1
		}
		if err := s.comments.IncrLikes(tx, commentID, delta); err != nil {
			return err
		}
		newCount = c.LikesCount + delta
		return nil
	})
	return liked, newCount, err
}
```
> 说明：`assembleComments` 中的 `authorIDs` 目前仅示意——作者信息由 handler/service 在 List 返回前批量填充，见步骤 7。

- [ ] **步骤 6：运行验证通过**

运行：`go test ./internal/service/ -run TestCommentCreateListDelete -v`
预期：PASS

- [ ] **步骤 7：List 作者信息填充**

`internal/service/comment.go` 的 `List` 中，组装后用 user repo 批量取作者昵称/头像回填 `item.Author`。给 `CommentService` 增加 `users *repo.UserRepo` 字段与构造参数，`List` 实现：
```go
func (s *CommentService) List(ctx context.Context, articleID uint) ([]CommentItem, error) {
	comments, err := s.comments.ListByArticle(nil, articleID)
	if err != nil {
		return nil, err
	}
	items := assembleComments(comments)
	// 收集 author id
	ids := map[uint]bool{}
	var collect func(items []CommentItem)
	collect = func(items []CommentItem) {
		for i := range items {
			ids[items[i].AuthorID] = true
			collect(items[i].Replies)
		}
	}
	collect(items)
	users, err := s.users.FindByIDs(ctx, keys(ids))
	if err != nil {
		return nil, err
	}
	byID := map[uint]model.User{}
	for _, u := range users {
		byID[u.ID] = u
	}
	var fill func(items []CommentItem)
	fill = func(items []CommentItem) {
		for i := range items {
			if u, ok := byID[items[i].AuthorID]; ok {
				items[i].Author = AuthorBrief{ID: u.ID, Nickname: u.Nickname, AvatarURL: u.AvatarURL}
			}
			fill(items[i].Replies)
		}
	}
	fill(items)
	return items, nil
}
```
> 说明：`repo.UserRepo` 需要新增 `FindByIDs(ctx, ids []uint) ([]model.User, error)`（`WHERE id IN ?`），`internal/repo/user.go` 追加并补测试。`keys` 为 map 键转 slice 的小工具。

- [ ] **步骤 8：handler + 路由 + 测试**

`internal/handler/comment.go`：`CommentHandler{List, Create, Delete, Like}`。路由：
```
GET    /api/v1/articles/:id/comments    (公开)
POST   /api/v1/articles/:id/comments    (Auth)
DELETE /api/v1/comments/:id             (Auth)
POST   /api/v1/comments/:id/like        (Auth)
```
`internal/handler/comment_test.go`：`TestCommentEndpoint`——建已发布文章 → POST 评论 200 → GET 列表断言一级含该评论 → 回复 parent_id 200 → DELETE 评论 200；未登录 POST 评论断言 401。

运行：`go test ./internal/handler/ -run TestCommentEndpoint -v`
预期：PASS

- [ ] **步骤 9：Commit**

```bash
git add internal/repo/comment.go internal/repo/comment_test.go internal/repo/user.go internal/service/comment.go internal/service/comment_test.go internal/service/dto.go internal/handler/comment.go internal/handler/comment_test.go
git commit -m "feat: 两级评论（发表/回复/删除/作者管理/评论点赞）"
```

---

## 任务 10：正文图片上传

**文件：**
- 修改：`internal/service/article.go`（UploadImage 辅助）、`internal/handler/article.go`（UploadImage）、`internal/handler/article_test.go`、`cmd/server/main.go`（/static/articles）

- [ ] **步骤 1：写失败测试（handler UploadImage）**

`internal/handler/article_test.go` 追加（仿 `TestUploadAvatar`）：
```go
func TestArticleUploadImage(t *testing.T) {
	// articleRouter 增加 POST /api/v1/articles/images (auth, h.UploadImage)
	r, users := articleRouter(t)
	u := &model.User{Email: "a@a.com", PasswordHash: "x", Nickname: "A", AvatarURL: "/x.png"}
	_ = users.Create(u)
	tok, _ := platform.GenerateAccessToken("s", time.Minute, u.ID)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "a.png")
	_, _ = fw.Write([]byte{0x89, 'P', 'N', 'G'})
	_ = mw.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/articles/images", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}
```
> 说明：`articleRouter` 需要给 ArticleService 传入带 `ArticleImageDir: t.TempDir()` 的 Config，并注册 `/static/articles`。

- [ ] **步骤 2：运行验证失败**

运行：`go test ./internal/handler/ -run TestArticleUploadImage -v`
预期：FAIL（`UploadImage` 未定义）

- [ ] **步骤 3：实现 handler UploadImage**

`internal/handler/article.go` 追加（仿 `UploadAvatar`）：
```go
func (h *ArticleHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		platform.Fail(c, http.StatusBadRequest, 40001, "不支持的图片格式")
		return
	}
	if file.Size > h.svc.MaxImageBytes() {
		platform.Fail(c, http.StatusBadRequest, 40001, "图片过大")
		return
	}
	if err := os.MkdirAll(h.svc.ImageDir(), 0o755); err != nil {
		platform.Fail(c, http.StatusInternalServerError, 50000, "服务器内部错误")
		return
	}
	name := randomHex(16) + ext
	if err := c.SaveUploadedFile(file, filepath.Join(h.svc.ImageDir(), name)); err != nil {
		platform.Fail(c, http.StatusInternalServerError, 50000, "服务器内部错误")
		return
	}
	platform.OK(c, gin.H{"url": "/static/articles/" + name})
}
```
> 说明：handler 复用 `user.go` 的 `randomHex`（同包内可见）。

- [ ] **步骤 4：运行验证通过**

运行：`go test ./internal/handler/ -run TestArticleUploadImage -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/handler/article.go internal/handler/article_test.go
git commit -m "feat: 文章正文图片上传"
```

---

## 任务 11：装配与集成（main.go）+ 全量验证

**文件：**
- 修改：`cmd/server/main.go`、`internal/handler/handler_test.go`（可选，setupRouter 扩展）

- [ ] **步骤 1：main.go 装配**

`cmd/server/main.go`：
- `AutoMigrate` 注册全部新模型（`model.Category`、`model.Tag`、`model.Article`、`model.ArticleTag`、`model.ArticleLike`、`model.ArticleFavorite`、`model.Comment`、`model.CommentLike`）。
- 迁移后 `repo.NewCategoryRepo(db).Seed(ctx)` 预置分类。
- 构建 repo/service：
```go
cats := repo.NewCategoryRepo(db)
tags := repo.NewTagRepo(db)
articles := repo.NewArticleRepo(db)
comments := repo.NewCommentRepo(db)
inter := repo.NewInteractionRepo(db)
users := repo.NewUserRepo(db)

catSvc := service.NewCategoryService(cats)
tagSvc := service.NewTagService(tags)
artSvc := service.NewArticleService(articles, tags, cats, inter, rdb, cfg)
comSvc := service.NewCommentService(comments, articles, inter, users)
```
- 路由：
```go
catH := handler.NewCategoryHandler(catSvc)
tagH := handler.NewTagHandler(tagSvc)
ah := handler.NewArticleHandler(artSvc)
ch := handler.NewCommentHandler(comSvc)
auth := platform.AuthMiddleware(cfg.JWTSecret)
opt := platform.OptionalAuthMiddleware(cfg.JWTSecret)

r.GET("/api/v1/categories", catH.List)
r.GET("/api/v1/tags", tagH.List)

arts := r.Group("/api/v1/articles")
arts.GET("", ah.List)
arts.GET("/:id", opt, ah.Get)
arts.POST("/images", auth, ah.UploadImage)
arts.POST("", auth, ah.Create)
arts.PUT("/:id", auth, ah.Update)
arts.DELETE("/:id", auth, ah.Delete)
arts.POST("/:id/like", auth, ah.Like)
arts.POST("/:id/favorite", auth, ah.Favorite)

artComments := r.Group("/api/v1/articles/:id/comments")
artComments.GET("", ch.List)
artComments.POST("", auth, ch.Create)

coms := r.Group("/api/v1/comments")
coms.DELETE("/:id", auth, ch.Delete)
coms.POST("/:id/like", auth, ch.Like)

r.Static("/static/articles", cfg.ArticleImageDir)
```
> 注意：`artComments.GET` 与 `arts.GET("/:id")` 用 `opt`/公开 会冲突——Gin 按注册顺序匹配，`arts.GET("/:id", opt, ah.Get)` 会吞掉 `/articles/:id/comments`？不会：路径段数不同（`/articles/:id/comments` 是 3 段，`/articles/:id` 是 2 段），Gin 的 radix 树按精确段匹配，不冲突。保持两个分组即可。

- [ ] **步骤 2：编译验证**

运行：`go build ./...`
预期：成功，无错误

- [ ] **步骤 3：全量测试**

运行：`go test ./...`
预期：全部 PASS（repo/service/handler/platform/testutil 各包）

- [ ] **步骤 4：手工冒烟（可选）**

运行：`go run ./cmd/server`，用 curl 走一遍：注册 → 登录 → 发布文章 → 列表/详情 → 评论 → 点赞。

- [ ] **步骤 5：更新阶段文档**

`docs/superpowers/specs/2026-08-24-p2-tech-community-design.md` 头部状态「待审查」→「已实现」；`docs/phase1-summary.md` 不动（阶段二结束时若有总结需求另行添加）。

- [ ] **步骤 6：Commit**

```bash
git add cmd/server/main.go internal/handler/handler_test.go
git commit -m "feat: P2 技术社区装配集成（路由/迁移/种子/静态目录）"
```

---

## 自检记录

- **规格覆盖度**：设计文档第 4 节全部 8 张表 → 任务 1；分类接口 → 任务 2；标签 → 任务 3；文章发布 → 任务 4；编辑/删除 → 任务 5；列表/详情/浏览量/OptionalAuth → 任务 6；热门+缓存 → 任务 7；文章互动 → 任务 8；两级评论/作者管理/评论点赞 → 任务 9；正文图片 → 任务 10；装配 → 任务 11。规格第 6 节各实现要点均已映射。草稿不可点赞/评论（需 published）在任务 8/9 的 `ToggleLike`/`Create` 中体现。
- **占位符扫描**：无「待定/TODO」；每个步骤含具体代码或明确实现要点。
- **类型一致性**：`ArticleQuery` 在 repo 层、`ListQuery` 在 service 层，命名已区分；`ArticleService` 构造参数顺序（articles, tags, cats, inter, rdb, cfg）在 `article_test.go` 与 `main.go` 一致；`CommentService` 构造（comments, articles, inter, users）一致；`newCommentItem`/`assembleComments`/`CommentItem` 一致；`IncrCount(db,id,column,delta)` 签名贯穿任务 4-9。
