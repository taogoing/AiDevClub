# P4 标签/搜索/排行优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现标签管理优化、MySQL FULLTEXT 全文搜索、Redis ZSet 热门排行（时间衰减算法）

**架构：** 管理员标签接口 + FULLTEXT 索引（ngram 中文分词）+ 定时预计算热度分数写入 Redis ZSet + 统一搜索接口 + 结果高亮

**技术栈：** Go、Gin、GORM、MySQL 8（FULLTEXT + ngram）、Redis 7（ZSet）、Vue 3

---

## 文件结构

### 新增文件

**后端：**
- `internal/handler/admin_tag.go` — 管理员标签 handler（创建/更新/禁用/启用/列表）
- `internal/handler/search.go` — 统一搜索 handler
- `internal/handler/ranking.go` — 排行 handler（文章/Skill/MCP Server）
- `internal/service/search.go` — 搜索 service（FULLTEXT 查询 + 高亮）
- `internal/service/ranking.go` — 排行 service（Redis ZSet 读写 + 热度计算）
- `internal/scheduler/ranking.go` — 定时任务（热榜预计算）
- `internal/repo/search.go` — 搜索 repo（FULLTEXT 查询封装）

**前端：**
- `frontend/src/views/SearchPage.vue` — 搜索结果页
- `frontend/src/components/admin/TagManagement.vue` — 标签管理页面（管理员）
- `frontend/src/api/search.ts` — 搜索 API
- `frontend/src/api/ranking.ts` — 排行 API
- `frontend/src/api/adminTag.ts` — 管理员标签 API

### 修改文件

**后端：**
- `internal/model/tag.go` — Tag 增加 `description` 字段
- `internal/repo/tag.go` — 增加管理员标签 CRUD 方法
- `internal/service/tag.go` — 增加管理员标签 service 方法
- `internal/repo/article.go` — keyword 搜索改用 FULLTEXT
- `internal/repo/skill.go` — keyword 搜索改用 FULLTEXT
- `internal/repo/mcp_server.go` — keyword 搜索改用 FULLTEXT
- `internal/service/article.go` — 点赞/收藏/评论时更新热榜分数
- `internal/service/skill.go` — 点赞/收藏/评论时更新热榜分数
- `internal/service/mcp_server.go` — 点赞/收藏/评论时更新热榜分数
- `internal/platform/database.go` — 创建 FULLTEXT 索引
- `internal/platform/config.go` — 增加 ranking 配置项
- `cmd/server/main.go` — 注册新路由 + 启动定时任务

**前端：**
- `frontend/src/components/Navbar.vue` — 添加搜索框
- `frontend/src/components/ResourceSidebar.vue` — 改用排行接口
- `frontend/src/router/index.ts` — 添加搜索页和标签管理路由
- `docker-compose.yml` — MySQL 增加 `--ngram_token_size=2` 参数

---

## 任务 1：Tag 模型增加 description 字段

**文件：**
- 修改：`internal/model/tag.go`
- 测试：`internal/model/tag_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
// internal/model/tag_test.go
package model

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestTag_HasDescription(t *testing.T) {
    tag := Tag{
        Name:        "Go",
        Description: "Go 编程语言",
    }
    assert.Equal(t, "Go 编程语言", tag.Description)
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/model -run TestTag_HasDescription -v`
预期：FAIL，编译错误 "unknown field Description"

- [ ] **步骤 3：修改 Tag 模型**

```go
// internal/model/tag.go
type Tag struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"size:64;uniqueIndex;not null"`
    Description string `gorm:"size:255"` // 新增
    UsageCount  int    `gorm:"not null;default:0"`
    Enabled     bool   `gorm:"not null;default:true"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/model -run TestTag_HasDescription -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/model/tag.go internal/model/tag_test.go
git commit -m "feat(tag): add description field to Tag model"
```

---

## 任务 2：管理员标签 Repo 层

**文件：**
- 修改：`internal/repo/tag.go`
- 测试：`internal/repo/tag_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
// internal/repo/tag_test.go
func TestTagRepo_AdminCreate(t *testing.T) {
    db := testutil.SetupTestDB(t)
    repo := NewTagRepo(db)
    
    tag, err := repo.AdminCreate(context.Background(), "Go", "Go 编程语言")
    assert.NoError(t, err)
    assert.Equal(t, "Go", tag.Name)
    assert.Equal(t, "Go 编程语言", tag.Description)
    assert.True(t, tag.Enabled)
}

func TestTagRepo_AdminList(t *testing.T) {
    db := testutil.SetupTestDB(t)
    repo := NewTagRepo(db)
    
    // 创建测试数据
    repo.AdminCreate(context.Background(), "Go", "")
    repo.AdminCreate(context.Background(), "Golang", "")
    repo.AdminCreate(context.Background(), "Python", "")
    
    // 测试前缀搜索
    tags, total, err := repo.AdminList(context.Background(), "Go", "all", 1, 10)
    assert.NoError(t, err)
    assert.Equal(t, 2, total)
    assert.Len(t, tags, 2)
    
    // 测试状态筛选
    repo.Disable(context.Background(), 1)
    tags, total, err = repo.AdminList(context.Background(), "", "enabled", 1, 10)
    assert.NoError(t, err)
    assert.Equal(t, 2, total) // Python + Golang
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/repo -run TestTagRepo_Admin -v`
预期：FAIL，方法未定义

- [ ] **步骤 3：实现 Repo 方法**

```go
// internal/repo/tag.go
func (r *TagRepo) AdminCreate(ctx context.Context, name, description string) (*model.Tag, error) {
    tag := &model.Tag{
        Name:        name,
        Description: description,
        Enabled:     true,
    }
    if err := r.db.WithContext(ctx).Create(tag).Error; err != nil {
        return nil, err
    }
    return tag, nil
}

func (r *TagRepo) AdminUpdate(ctx context.Context, id uint, name, description string) error {
    return r.db.WithContext(ctx).
        Model(&model.Tag{}).
        Where("id = ?", id).
        Updates(map[string]interface{}{
            "name":        name,
            "description": description,
        }).Error
}

func (r *TagRepo) Enable(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).
        Model(&model.Tag{}).
        Where("id = ?", id).
        Update("enabled", true).Error
}

func (r *TagRepo) Disable(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).
        Model(&model.Tag{}).
        Where("id = ?", id).
        Update("enabled", false).Error
}

func (r *TagRepo) AdminList(ctx context.Context, keyword, status string, page, pageSize int) ([]model.Tag, int64, error) {
    query := r.db.WithContext(ctx).Model(&model.Tag{})
    
    if keyword != "" {
        query = query.Where("name LIKE ?", keyword+"%")
    }
    
    switch status {
    case "enabled":
        query = query.Where("enabled = ?", true)
    case "disabled":
        query = query.Where("enabled = ?", false)
    }
    
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    var tags []model.Tag
    err := query.Order("id DESC").
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&tags).Error
    
    return tags, total, err
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/repo -run TestTagRepo_Admin -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/tag.go internal/repo/tag_test.go
git commit -m "feat(tag): add admin tag repo methods"
```

---

## 任务 3：管理员标签 Service 和 Handler

**文件：**
- 修改：`internal/service/tag.go`
- 创建：`internal/handler/admin_tag.go`
- 测试：`internal/handler/admin_tag_test.go`

- [ ] **步骤 1：编写 Service 方法**

```go
// internal/service/tag.go
func (s *TagService) AdminCreate(ctx context.Context, name, description string) (*model.Tag, error) {
    if name == "" {
        return nil, errors.New("标签名称不能为空")
    }
    return s.repo.AdminCreate(ctx, name, description)
}

func (s *TagService) AdminUpdate(ctx context.Context, id uint, name, description string) error {
    if name == "" {
        return errors.New("标签名称不能为空")
    }
    return s.repo.AdminUpdate(ctx, id, name, description)
}

func (s *TagService) Enable(ctx context.Context, id uint) error {
    return s.repo.Enable(ctx, id)
}

func (s *TagService) Disable(ctx context.Context, id uint) error {
    return s.repo.Disable(ctx, id)
}

func (s *TagService) AdminList(ctx context.Context, keyword, status string, page, pageSize int) ([]model.Tag, int64, error) {
    return s.repo.AdminList(ctx, keyword, status, page, pageSize)
}
```

- [ ] **步骤 2：编写 Handler**

```go
// internal/handler/admin_tag.go
package handler

import (
    "github.com/gin-gonic/gin"
    "AIDevClub/internal/platform"
    "strconv"
)

type AdminTagHandler struct {
    svc *service.TagService
}

func NewAdminTagHandler(svc *service.TagService) *AdminTagHandler {
    return &AdminTagHandler{svc: svc}
}

type createTagRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
}

func (h *AdminTagHandler) Create(c *gin.Context) {
    var req createTagRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        platform.Fail(c, 400, "参数错误")
        return
    }
    
    tag, err := h.svc.AdminCreate(c.Request.Context(), req.Name, req.Description)
    if err != nil {
        platform.Fail(c, 500, err.Error())
        return
    }
    
    platform.OK(c, tag)
}

type updateTagRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
}

func (h *AdminTagHandler) Update(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        platform.Fail(c, 400, "无效的标签 ID")
        return
    }
    
    var req updateTagRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        platform.Fail(c, 400, "参数错误")
        return
    }
    
    if err := h.svc.AdminUpdate(c.Request.Context(), id, req.Name, req.Description); err != nil {
        platform.Fail(c, 500, err.Error())
        return
    }
    
    platform.OK(c, nil)
}

func (h *AdminTagHandler) Enable(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        platform.Fail(c, 400, "无效的标签 ID")
        return
    }
    
    if err := h.svc.Enable(c.Request.Context(), id); err != nil {
        platform.Fail(c, 500, err.Error())
        return
    }
    
    platform.OK(c, nil)
}

func (h *AdminTagHandler) Disable(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        platform.Fail(c, 400, "无效的标签 ID")
        return
    }
    
    if err := h.svc.Disable(c.Request.Context(), id); err != nil {
        platform.Fail(c, 500, err.Error())
        return
    }
    
    platform.OK(c, nil)
}

func (h *AdminTagHandler) List(c *gin.Context) {
    keyword := c.Query("keyword")
    status := c.DefaultQuery("status", "all")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    
    tags, total, err := h.svc.AdminList(c.Request.Context(), keyword, status, page, pageSize)
    if err != nil {
        platform.Fail(c, 500, err.Error())
        return
    }
    
    platform.OK(c, gin.H{
        "items":     tags,
        "total":     total,
        "page":      page,
        "page_size": pageSize,
    })
}
```

- [ ] **步骤 3：编写 Handler 测试**

```go
// internal/handler/admin_tag_test.go
package handler

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "AIDevClub/internal/testutil"
)

func TestAdminTagHandler_Create(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := testutil.SetupTestDB(t)
    tagRepo := repo.NewTagRepo(db)
    tagSvc := service.NewTagService(tagRepo, nil)
    handler := NewAdminTagHandler(tagSvc)
    
    router := gin.New()
    router.POST("/admin/tags", handler.Create)
    
    body := map[string]string{
        "name":        "Go",
        "description": "Go 编程语言",
    }
    jsonBody, _ := json.Marshal(body)
    
    req := httptest.NewRequest("POST", "/admin/tags", bytes.NewReader(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    
    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, float64(0), resp["code"])
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/handler -run TestAdminTagHandler -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/service/tag.go internal/handler/admin_tag.go internal/handler/admin_tag_test.go
git commit -m "feat(tag): add admin tag service and handler"
```

---

## 任务 4：用户端标签接口优化（前缀搜索 + 缓存）

**文件：**
- 修改：`internal/repo/tag.go`
- 修改：`internal/service/tag.go`
- 修改：`internal/handler/tag.go`

- [ ] **步骤 1：修改 Repo 支持前缀搜索**

```go
// internal/repo/tag.go
func (r *TagRepo) List(ctx context.Context, keyword string, hot bool, limit int) ([]model.Tag, error) {
    var list []model.Tag
    query := r.db.WithContext(ctx).Where("enabled = ?", true)
    
    if keyword != "" {
        query = query.Where("name LIKE ?", keyword+"%") // 前缀匹配
    }
    
    if hot {
        query = query.Order("usage_count DESC, id ASC")
    } else {
        query = query.Order("name ASC")
    }
    
    if limit > 0 {
        query = query.Limit(limit)
    }
    
    err := query.Find(&list).Error
    return list, err
}
```

- [ ] **步骤 2：修改 Service 增加缓存**

```go
// internal/service/tag.go
func (s *TagService) List(ctx context.Context, prefix string, hot bool, limit int) ([]model.Tag, error) {
    if limit == 0 {
        limit = 20
    }
    
    // 热门标签使用缓存
    if hot && prefix == "" {
        key := fmt.Sprintf("hot:tags:%d", limit)
        if v, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
            var tags []model.Tag
            if json.Unmarshal(v, &tags) == nil {
                return tags, nil
            }
        }
        
        tags, err := s.repo.List(ctx, "", true, limit)
        if err != nil {
            return nil, err
        }
        
        if b, err := json.Marshal(tags); err == nil {
            _ = s.rdb.Set(ctx, key, b, 300*time.Second).Err()
        }
        
        return tags, nil
    }
    
    return s.repo.List(ctx, prefix, hot, limit)
}
```

- [ ] **步骤 3：修改 Handler 参数**

```go
// internal/handler/tag.go
func (h *TagHandler) List(c *gin.Context) {
    prefix := c.Query("prefix") // 改为 prefix
    hot := c.Query("hot") == "1"
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    tags, err := h.svc.List(c.Request.Context(), prefix, hot, limit)
    if err != nil {
        platform.Fail(c, 500, err.Error())
        return
    }
    
    platform.OK(c, tags)
}
```

- [ ] **步骤 4：运行测试**

运行：`go test ./internal/... -v`
预期：所有测试 PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/tag.go internal/service/tag.go internal/handler/tag.go
git commit -m "feat(tag): optimize tag list with prefix search and cache"
```

---

## 任务 5：MySQL FULLTEXT 索引创建

**文件：**
- 修改：`internal/platform/database.go`
- 修改：`docker-compose.yml`

- [ ] **步骤 1：修改 docker-compose.yml**

```yaml
# docker-compose.yml
services:
  mysql:
    image: mysql:8.0
    command: --ngram_token_size=2  # 新增
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: aidevclub
    ports:
      - "3306:3306"
```

- [ ] **步骤 2：添加 FULLTEXT 索引创建函数**

```go
// internal/platform/database.go
func createFulltextIndexes(db *gorm.DB) {
    // 文章全文索引
    db.Exec(`
        CREATE FULLTEXT INDEX idx_ft_article_search 
        ON articles(title, summary, content) 
        WITH PARSER ngram
    `)
    
    // Skill 全文索引
    db.Exec(`
        CREATE FULLTEXT INDEX idx_ft_skill_search 
        ON skills(name, description) 
        WITH PARSER ngram
    `)
    
    // MCP Server 全文索引
    db.Exec(`
        CREATE FULLTEXT INDEX idx_ft_mcp_search 
        ON mcp_servers(name, description) 
        WITH PARSER ngram
    `)
}

func InitDB(cfg *Config) (*gorm.DB, error) {
    // ... 现有代码 ...
    
    // AutoMigrate
    db.AutoMigrate(
        &model.User{},
        &model.Category{},
        &model.Article{},
        &model.Tag{}, // 已更新
        &model.Skill{},
        &model.McpServer{},
        // ... 其他模型 ...
    )
    
    // 创建 FULLTEXT 索引
    createFulltextIndexes(db)
    
    return db, nil
}
```

- [ ] **步骤 3：重启 MySQL 验证索引创建**

运行：
```bash
docker compose down
docker compose up -d
go run ./cmd/server
```

验证：
```bash
docker exec -it aidevclub-mysql-1 mysql -uroot -proot aidevclub
SHOW INDEX FROM articles WHERE Key_name = 'idx_ft_article_search';
```

预期：看到 FULLTEXT 索引

- [ ] **步骤 4：Commit**

```bash
git add docker-compose.yml internal/platform/database.go
git commit -m "feat(search): add FULLTEXT indexes with ngram parser"
```

---

## 任务 6：搜索 Repo 层（FULLTEXT 查询）

**文件：**
- 创建：`internal/repo/search.go`
- 测试：`internal/repo/search_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
// internal/repo/search_test.go
func TestSearchRepo_SearchArticles(t *testing.T) {
    db := testutil.SetupTestDB(t)
    repo := NewSearchRepo(db)
    
    // 创建测试文章
    createTestArticle(t, db, "Go 语言入门", "学习 Go 语言的基础知识", "Go 是一门...")
    createTestArticle(t, db, "Python 教程", "Python 编程指南", "Python 是一门...")
    
    // 搜索 "Go"
    results, total, err := repo.SearchArticles(context.Background(), "Go", nil, nil, 1, 10)
    assert.NoError(t, err)
    assert.Equal(t, 1, total)
    assert.Equal(t, "Go 语言入门", results[0].Title)
}

func TestSearchRepo_SearchSkills(t *testing.T) {
    db := testutil.SetupTestDB(t)
    repo := NewSearchRepo(db)
    
    createTestSkill(t, db, "code-reviewer", "代码审查工具")
    createTestSkill(t, db, "test-generator", "测试生成工具")
    
    results, total, err := repo.SearchSkills(context.Background(), "代码", nil, 1, 10)
    assert.NoError(t, err)
    assert.Equal(t, 1, total)
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/repo -run TestSearchRepo -v`
预期：FAIL，SearchRepo 未定义

- [ ] **步骤 3：实现 SearchRepo**

```go
// internal/repo/search.go
package repo

import (
    "context"
    "gorm.io/gorm"
    "AIDevClub/internal/model"
)

type SearchRepo struct {
    db *gorm.DB
}

func NewSearchRepo(db *gorm.DB) *SearchRepo {
    return &SearchRepo{db: db}
}

func (r *SearchRepo) SearchArticles(ctx context.Context, keyword string, tagID, categoryID *uint, page, pageSize int) ([]model.Article, int64, error) {
    query := r.db.WithContext(ctx).
        Model(&model.Article{}).
        Where("status = ?", "published").
        Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", keyword)
    
    if tagID != nil {
        query = query.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
            Where("article_tags.tag_id = ?", *tagID)
    }
    
    if categoryID != nil {
        query = query.Where("category_id = ?", *categoryID)
    }
    
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    var articles []model.Article
    err := query.
        Order("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE) DESC", keyword).
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&articles).Error
    
    return articles, total, err
}

func (r *SearchRepo) SearchSkills(ctx context.Context, keyword string, tagID *uint, page, pageSize int) ([]model.Skill, int64, error) {
    query := r.db.WithContext(ctx).
        Model(&model.Skill{}).
        Where("status = ?", "published").
        Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", keyword)
    
    if tagID != nil {
        query = query.Joins("JOIN skill_tags ON skill_tags.skill_id = skills.id").
            Where("skill_tags.tag_id = ?", *tagID)
    }
    
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    var skills []model.Skill
    err := query.
        Order("MATCH(name, description) AGAINST(? IN BOOLEAN MODE) DESC", keyword).
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&skills).Error
    
    return skills, total, err
}

func (r *SearchRepo) SearchMcpServers(ctx context.Context, keyword string, tagID *uint, page, pageSize int) ([]model.McpServer, int64, error) {
    query := r.db.WithContext(ctx).
        Model(&model.McpServer{}).
        Where("status = ?", "published").
        Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", keyword)
    
    if tagID != nil {
        query = query.Joins("JOIN mcp_server_tags ON mcp_server_tags.mcp_server_id = mcp_servers.id").
            Where("mcp_server_tags.tag_id = ?", *tagID)
    }
    
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    var servers []model.McpServer
    err := query.
        Order("MATCH(name, description) AGAINST(? IN BOOLEAN MODE) DESC", keyword).
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&servers).Error
    
    return servers, total, err
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/repo -run TestSearchRepo -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/search.go internal/repo/search_test.go
git commit -m "feat(search): add search repo with FULLTEXT queries"
```

---

## 任务 7：搜索 Service 层（高亮 + 统一搜索）

**文件：**
- 创建：`internal/service/search.go`
- 测试：`internal/service/search_test.go`

- [ ] **步骤 1：编写高亮函数测试**

```go
// internal/service/search_test.go
func TestHighlightText(t *testing.T) {
    result := highlightText("Go 语言入门", "Go")
    assert.Equal(t, "<mark>Go</mark> 语言入门", result)
    
    result = highlightText("学习 Go 和 Python", "Go")
    assert.Equal(t, "学习 <mark>Go</mark> 和 Python", result)
    
    // 不区分大小写
    result = highlightText("GOLANG 编程", "go")
    assert.Equal(t, "<mark>GOLANG</mark> 编程", result)
}
```

- [ ] **步骤 2：实现高亮函数**

```go
// internal/service/search.go
package service

import (
    "context"
    "regexp"
    "strings"
    "AIDevClub/internal/model"
    "AIDevClub/internal/repo"
)

type SearchService struct {
    searchRepo *repo.SearchRepo
}

func NewSearchService(searchRepo *repo.SearchRepo) *SearchService {
    return &SearchService{searchRepo: searchRepo}
}

func highlightText(text, keyword string) string {
    if keyword == "" {
        return text
    }
    
    // 转义 HTML
    text = strings.ReplaceAll(text, "&", "&amp;")
    text = strings.ReplaceAll(text, "<", "&lt;")
    text = strings.ReplaceAll(text, ">", "&gt;")
    
    // 不区分大小写替换
    pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(keyword))
    return pattern.ReplaceAllStringFunc(text, func(match string) string {
        return "<mark>" + match + "</mark>"
    })
}

type SearchResult struct {
    ID        uint   `json:"id"`
    Type      string `json:"type"`
    Title     string `json:"title"`
    Summary   string `json:"summary"`
    Author    interface{} `json:"author"`
    Tags      []model.Tag `json:"tags"`
    Views     int    `json:"views"`
    LikesCount int   `json:"likes_count"`
    CreatedAt string `json:"created_at"`
}

type SearchResponse struct {
    Items    []SearchResult `json:"items"`
    Total    int64          `json:"total"`
    Page     int            `json:"page"`
    PageSize int            `json:"page_size"`
    Counts   map[string]int64 `json:"counts,omitempty"`
}

func (s *SearchService) Search(ctx context.Context, keyword, searchType string, tagID, categoryID *uint, page, pageSize int) (*SearchResponse, error) {
    if pageSize == 0 {
        pageSize = 20
    }
    
    var items []SearchResult
    var total int64
    counts := make(map[string]int64)
    
    switch searchType {
    case "article":
        articles, count, err := s.searchRepo.SearchArticles(ctx, keyword, tagID, categoryID, page, pageSize)
        if err != nil {
            return nil, err
        }
        total = count
        for _, a := range articles {
            items = append(items, SearchResult{
                ID:      a.ID,
                Type:    "article",
                Title:   highlightText(a.Title, keyword),
                Summary: highlightText(a.Summary, keyword),
                // ... 其他字段
            })
        }
        
    case "skill":
        skills, count, err := s.searchRepo.SearchSkills(ctx, keyword, tagID, page, pageSize)
        if err != nil {
            return nil, err
        }
        total = count
        for _, sk := range skills {
            items = append(items, SearchResult{
                ID:      sk.ID,
                Type:    "skill",
                Title:   highlightText(sk.Name, keyword),
                Summary: highlightText(sk.Description, keyword),
                // ... 其他字段
            })
        }
        
    case "mcp_server":
        servers, count, err := s.searchRepo.SearchMcpServers(ctx, keyword, tagID, page, pageSize)
        if err != nil {
            return nil, err
        }
        total = count
        for _, s := range servers {
            items = append(items, SearchResult{
                ID:      s.ID,
                Type:    "mcp_server",
                Title:   highlightText(s.Name, keyword),
                Summary: highlightText(s.Description, keyword),
                // ... 其他字段
            })
        }
        
    default: // 搜索所有类型
        // 分别查询三种类型，统计数量
        _, articleCount, _ := s.searchRepo.SearchArticles(ctx, keyword, tagID, categoryID, 1, 1)
        _, skillCount, _ := s.searchRepo.SearchSkills(ctx, keyword, tagID, 1, 1)
        _, mcpCount, _ := s.searchRepo.SearchMcpServers(ctx, keyword, tagID, 1, 1)
        
        counts["article"] = articleCount
        counts["skill"] = skillCount
        counts["mcp_server"] = mcpCount
        total = articleCount + skillCount + mcpCount
        
        // 合并结果（简化：先返回文章）
        articles, _, _ := s.searchRepo.SearchArticles(ctx, keyword, tagID, categoryID, page, pageSize)
        for _, a := range articles {
            items = append(items, SearchResult{
                ID:   a.ID,
                Type: "article",
                // ...
            })
        }
    }
    
    return &SearchResponse{
        Items:    items,
        Total:    total,
        Page:     page,
        PageSize: pageSize,
        Counts:   counts,
    }, nil
}
```

- [ ] **步骤 3：运行测试验证通过**

运行：`go test ./internal/service -run TestHighlightText -v`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/service/search.go internal/service/search_test.go
git commit -m "feat(search): add search service with highlighting"
```

---

## 任务 8：搜索 Handler 和路由

**文件：**
- 创建：`internal/handler/search.go`
- 修改：`cmd/server/main.go`

- [ ] **步骤 1：编写 Search Handler**

```go
// internal/handler/search.go
package handler

import (
    "github.com/gin-gonic/gin"
    "strconv"
    "AIDevClub/internal/platform"
    "AIDevClub/internal/service"
)

type SearchHandler struct {
    svc *service.SearchService
}

func NewSearchHandler(svc *service.SearchService) *SearchHandler {
    return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Search(c *gin.Context) {
    keyword := c.Query("q")
    if keyword == "" {
        platform.Fail(c, 400, "搜索关键词不能为空")
        return
    }
    
    searchType := c.Query("type") // article/skill/mcp_server/空
    
    var tagID, categoryID *uint
    if tagIDStr := c.Query("tag_id"); tagIDStr != "" {
        if id, err := strconv.ParseUint(tagIDStr, 10, 64); err == nil {
            id := uint(id)
            tagID = &id
        }
    }
    
    if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
        if id, err := strconv.ParseUint(categoryIDStr, 10, 64); err == nil {
            id := uint(id)
            categoryID = &id
        }
    }
    
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    
    result, err := h.svc.Search(c.Request.Context(), keyword, searchType, tagID, categoryID, page, pageSize)
    if err != nil {
        platform.Fail(c, 500, err.Error())
        return
    }
    
    platform.OK(c, result)
}
```

- [ ] **步骤 2：注册路由**

```go
// cmd/server/main.go
func main() {
    // ... 现有代码 ...
    
    // 搜索接口
    searchRepo := repo.NewSearchRepo(db)
    searchSvc := service.NewSearchService(searchRepo)
    searchHandler := handler.NewSearchHandler(searchSvc)
    
    api := r.Group("/api/v1")
    api.GET("/search", searchHandler.Search)
    
    // ... 其他路由 ...
}
```

- [ ] **步骤 3：运行测试**

运行：`go test ./internal/handler -run TestSearchHandler -v`
预期：PASS（如果有测试）

- [ ] **步骤 4：手动测试**

启动服务：
```bash
go run ./cmd/server
```

测试搜索：
```bash
curl "http://localhost:8080/api/v1/search?q=Go&type=article"
```

预期：返回匹配的文章列表，标题和摘要包含 `<mark>Go</mark>`

- [ ] **步骤 5：Commit**

```bash
git add internal/handler/search.go cmd/server/main.go
git commit -m "feat(search): add unified search API endpoint"
```

---

## 任务 9：现有列表接口改用 FULLTEXT

**文件：**
- 修改：`internal/repo/article.go`
- 修改：`internal/repo/skill.go`
- 修改：`internal/repo/mcp_server.go`

- [ ] **步骤 1：修改 Article Repo**

```go
// internal/repo/article.go
func (r *ArticleRepo) List(ctx context.Context, q ArticleListQuery) (*ArticleListResult, error) {
    d := r.db.WithContext(ctx).Model(&model.Article{}).Where("status = ?", "published")
    
    // 改用 FULLTEXT
    if q.Keyword != "" {
        d = d.Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", q.Keyword)
    }
    
    // ... 其他筛选条件 ...
}
```

- [ ] **步骤 2：修改 Skill Repo**

```go
// internal/repo/skill.go
func (r *SkillRepo) List(ctx context.Context, q SkillListQuery) (*SkillListResult, error) {
    d := r.db.WithContext(ctx).Model(&model.Skill{}).Where("status = ?", "published")
    
    // 改用 FULLTEXT
    if q.Keyword != "" {
        d = d.Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", q.Keyword)
    }
    
    // ... 其他筛选条件 ...
}
```

- [ ] **步骤 3：修改 MCP Server Repo**

```go
// internal/repo/mcp_server.go
func (r *McpServerRepo) List(ctx context.Context, q McpServerListQuery) (*McpServerListResult, error) {
    d := r.db.WithContext(ctx).Model(&model.McpServer{}).Where("status = ?", "published")
    
    // 改用 FULLTEXT
    if q.Keyword != "" {
        d = d.Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", q.Keyword)
    }
    
    // ... 其他筛选条件 ...
}
```

- [ ] **步骤 4：运行测试**

运行：`go test ./internal/repo -v`
预期：所有测试 PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/repo/article.go internal/repo/skill.go internal/repo/mcp_server.go
git commit -m "feat(search): migrate list APIs to FULLTEXT search"
```

---

## 任务 10：排行 Service（Redis ZSet + 热度计算）

**文件：**
- 创建：`internal/service/ranking.go`
- 测试：`internal/service/ranking_test.go`

- [ ] **步骤 1：编写热度计算测试**

```go
// internal/service/ranking_test.go
func TestCalculateHotScore(t *testing.T) {
    now := time.Now()
    
    // 新文章（1 小时前）
    article := &model.Article{
        Views:         100,
        LikesCount:    10,
        FavoritesCount: 5,
        CommentsCount: 3,
        PublishedAt:   now.Add(-1 * time.Hour),
    }
    score := calculateHotScore(article)
    assert.Greater(t, score, 30.0) // 约 34.8
    
    // 老文章（30 天前）
    article.PublishedAt = now.Add(-30 * 24 * time.Hour)
    score = calculateHotScore(article)
    assert.Less(t, score, 1.0) // 约 0.09
}
```

- [ ] **步骤 2：实现排行 Service**

```go
// internal/service/ranking.go
package service

import (
    "context"
    "math"
    "time"
    "github.com/redis/go-redis/v9"
    "AIDevClub/internal/model"
    "AIDevClub/internal/repo"
)

type RankingService struct {
    rdb        *redis.Client
    articleRepo *repo.ArticleRepo
    skillRepo  *repo.SkillRepo
    mcpRepo    *repo.McpServerRepo
    gravity    float64
}

func NewRankingService(rdb *redis.Client, articleRepo *repo.ArticleRepo, skillRepo *repo.SkillRepo, mcpRepo *repo.McpServerRepo, gravity float64) *RankingService {
    return &RankingService{
        rdb:        rdb,
        articleRepo: articleRepo,
        skillRepo:  skillRepo,
        mcpRepo:    mcpRepo,
        gravity:    gravity,
    }
}

func calculateHotScore(article *model.Article) float64 {
    score := float64(article.Views + 3*article.LikesCount + 5*article.FavoritesCount + 2*article.CommentsCount + 1)
    hours := time.Since(article.PublishedAt).Hours()
    return score / math.Pow(hours+2, 1.5)
}

func (s *RankingService) RecalculateArticleHotRanking(ctx context.Context) error {
    articles, err := s.articleRepo.ListAllPublished(ctx)
    if err != nil {
        return err
    }
    
    pipe := s.rdb.Pipeline()
    pipe.Del(ctx, "rank:articles:hot")
    
    for _, a := range articles {
        score := calculateHotScore(&a)
        pipe.ZAdd(ctx, "rank:articles:hot", &redis.Z{
            Score:  score,
            Member: a.ID,
        })
    }
    
    _, err = pipe.Exec(ctx)
    return err
}

func (s *RankingService) GetArticleHotRanking(ctx context.Context, page, pageSize int) ([]uint, error) {
    start := int64((page - 1) * pageSize)
    stop := start + int64(pageSize) - 1
    
    ids, err := s.rdb.ZRevRange(ctx, "rank:articles:hot", start, stop).Result()
    if err != nil {
        return nil, err
    }
    
    result := make([]uint, len(ids))
    for i, id := range ids {
        result[i] = uint(id)
    }
    return result, nil
}

func (s *RankingService) UpdateArticleHotScore(ctx context.Context, articleID uint) error {
    article, err := s.articleRepo.FindByID(ctx, articleID)
    if err != nil {
        return err
    }
    
    score := calculateHotScore(article)
    return s.rdb.ZAdd(ctx, "rank:articles:hot", &redis.Z{
        Score:  score,
        Member: articleID,
    }).Err()
}

// Skill 和 MCP Server 的排行方法类似...
```

- [ ] **步骤 3：运行测试验证通过**

运行：`go test ./internal/service -run TestCalculateHotScore -v`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/service/ranking.go internal/service/ranking_test.go
git commit -m "feat(ranking): add ranking service with hot score calculation"
```

---

## 任务 11：定时任务（热榜预计算）

**文件：**
- 创建：`internal/scheduler/ranking.go`
- 修改：`cmd/server/main.go`

- [ ] **步骤 1：实现定时任务**

```go
// internal/scheduler/ranking.go
package scheduler

import (
    "context"
    "time"
    "log"
    "AIDevClub/internal/service"
)

type RankingScheduler struct {
    rankingSvc *service.RankingService
    interval   time.Duration
}

func NewRankingScheduler(rankingSvc *service.RankingService, interval time.Duration) *RankingScheduler {
    return &RankingScheduler{
        rankingSvc: rankingSvc,
        interval:   interval,
    }
}

func (s *RankingScheduler) Start() {
    go func() {
        ticker := time.NewTicker(s.interval)
        defer ticker.Stop()
        
        for {
            <-ticker.C
            s.recalculate()
        }
    }()
}

func (s *RankingScheduler) recalculate() {
    ctx := context.Background()
    
    if err := s.rankingSvc.RecalculateArticleHotRanking(ctx); err != nil {
        log.Printf("Failed to recalculate article hot ranking: %v", err)
    }
    
    if err := s.rankingSvc.RecalculateSkillHotRanking(ctx); err != nil {
        log.Printf("Failed to recalculate skill hot ranking: %v", err)
    }
    
    if err := s.rankingSvc.RecalculateMcpServerHotRanking(ctx); err != nil {
        log.Printf("Failed to recalculate mcp server hot ranking: %v", err)
    }
    
    log.Println("Ranking recalculation completed")
}
```

- [ ] **步骤 2：在 main.go 中启动定时任务**

```go
// cmd/server/main.go
func main() {
    // ... 初始化代码 ...
    
    // 启动排行定时任务
    rankingSvc := service.NewRankingService(rdb, articleRepo, skillRepo, mcpRepo, cfg.Ranking.Gravity)
    rankingScheduler := scheduler.NewRankingScheduler(rankingSvc, cfg.Ranking.RecalcInterval)
    rankingScheduler.Start()
    
    // ... 启动服务器 ...
}
```

- [ ] **步骤 3：运行测试**

启动服务，等待 2 分钟，查看日志：
```bash
go run ./cmd/server
```

预期日志：
```
Ranking recalculation completed
```

- [ ] **步骤 4：Commit**

```bash
git add internal/scheduler/ranking.go cmd/server/main.go
git commit -m "feat(ranking): add scheduled hot ranking recalculation"
```

---

## 任务 12：排行 Handler 和路由

**文件：**
- 创建：`internal/handler/ranking.go`
- 修改：`cmd/server/main.go`

- [ ] **步骤 1：编写 Ranking Handler**

```go
// internal/handler/ranking.go
package handler

import (
    "github.com/gin-gonic/gin"
    "strconv"
    "AIDevClub/internal/platform"
    "AIDevClub/internal/service"
)

type RankingHandler struct {
    rankingSvc *service.RankingService
    articleSvc *service.ArticleService
    skillSvc   *service.SkillService
    mcpSvc     *service.McpServerService
}

func NewRankingHandler(rankingSvc *service.RankingService, articleSvc *service.ArticleService, skillSvc *service.SkillService, mcpSvc *service.McpServerService) *RankingHandler {
    return &RankingHandler{
        rankingSvc: rankingSvc,
        articleSvc: articleSvc,
        skillSvc:   skillSvc,
        mcpSvc:     mcpSvc,
    }
}

func (h *RankingHandler) GetArticleRanking(c *gin.Context) {
    rankType := c.DefaultQuery("type", "hot")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
    
    switch rankType {
    case "hot":
        ids, err := h.rankingSvc.GetArticleHotRanking(c.Request.Context(), page, pageSize)
        if err != nil {
            platform.Fail(c, 500, err.Error())
            return
        }
        
        articles, err := h.articleSvc.GetByIDs(c.Request.Context(), ids)
        if err != nil {
            platform.Fail(c, 500, err.Error())
            return
        }
        
        platform.OK(c, gin.H{
            "items":     articles,
            "total":     len(articles),
            "page":      page,
            "page_size": pageSize,
        })
        
    default:
        platform.Fail(c, 400, "不支持的排行类型")
    }
}

// GetSkillRanking 和 GetMcpServerRanking 类似...
```

- [ ] **步骤 2：注册路由**

```go
// cmd/server/main.go
rankingHandler := handler.NewRankingHandler(rankingSvc, articleSvc, skillSvc, mcpSvc)

api.GET("/articles/ranking", rankingHandler.GetArticleRanking)
api.GET("/skills/ranking", rankingHandler.GetSkillRanking)
api.GET("/mcp-servers/ranking", rankingHandler.GetMcpServerRanking)
```

- [ ] **步骤 3：手动测试**

```bash
curl "http://localhost:8080/api/v1/articles/ranking?type=hot&page=1&page_size=10"
```

预期：返回热榜文章列表

- [ ] **步骤 4：Commit**

```bash
git add internal/handler/ranking.go cmd/server/main.go
git commit -m "feat(ranking): add ranking API endpoints"
```

---

## 任务 13：互动时实时更新热榜分数

**文件：**
- 修改：`internal/service/article.go`
- 修改：`internal/service/skill.go`
- 修改：`internal/service/mcp_server.go`

- [ ] **步骤 1：修改 Article Service**

```go
// internal/service/article.go
func (s *ArticleService) Like(ctx context.Context, articleID, userID uint) error {
    // ... 现有点赞逻辑 ...
    
    // 更新热榜分数
    go s.rankingSvc.UpdateArticleHotScore(ctx, articleID)
    
    return nil
}

func (s *ArticleService) Favorite(ctx context.Context, articleID, userID uint) error {
    // ... 现有收藏逻辑 ...
    
    // 更新热榜分数
    go s.rankingSvc.UpdateArticleHotScore(ctx, articleID)
    
    return nil
}

func (s *ArticleService) IncrementViews(ctx context.Context, articleID uint) error {
    // ... 现有浏览量逻辑 ...
    
    // 更新热榜分数
    go s.rankingSvc.UpdateArticleHotScore(ctx, articleID)
    
    return nil
}
```

- [ ] **步骤 2：修改 Skill Service 和 MCP Server Service**

类似 Article Service 的修改。

- [ ] **步骤 3：运行测试**

运行：`go test ./internal/service -v`
预期：所有测试 PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/service/article.go internal/service/skill.go internal/service/mcp_server.go
git commit -m "feat(ranking): update hot score on interactions"
```

---

## 任务 14：前端搜索框和搜索结果页

**文件：**
- 创建：`frontend/src/views/SearchPage.vue`
- 创建：`frontend/src/api/search.ts`
- 修改：`frontend/src/components/Navbar.vue`
- 修改：`frontend/src/router/index.ts`

- [ ] **步骤 1：创建搜索 API**

```typescript
// frontend/src/api/search.ts
import request from '@/utils/request'

export interface SearchResult {
  id: number
  type: 'article' | 'skill' | 'mcp_server'
  title: string
  summary: string
  author: any
  tags: any[]
  views: number
  likes_count: number
  created_at: string
}

export interface SearchResponse {
  items: SearchResult[]
  total: number
  page: number
  page_size: number
  counts?: {
    article: number
    skill: number
    mcp_server: number
  }
}

export function search(params: {
  q: string
  type?: string
  tag_id?: number
  category_id?: number
  page?: number
  page_size?: number
}) {
  return request.get<SearchResponse>('/search', { params })
}
```

- [ ] **步骤 2：创建搜索结果页**

```vue
<!-- frontend/src/views/SearchPage.vue -->
<template>
  <div class="search-page">
    <div class="search-header">
      <h1>搜索结果：{{ keyword }}</h1>
      <div class="type-tabs">
        <el-radio-group v-model="searchType" @change="handleSearch">
          <el-radio-button label="">全部 ({{ totalCount }})</el-radio-button>
          <el-radio-button label="article">文章 ({{ counts.article }})</el-radio-button>
          <el-radio-button label="skill">Skill ({{ counts.skill }})</el-radio-button>
          <el-radio-button label="mcp_server">MCP Server ({{ counts.mcp_server }})</el-radio-button>
        </el-radio-group>
      </div>
    </div>
    
    <div class="search-results">
      <div v-for="item in results" :key="`${item.type}-${item.id}`" class="result-item">
        <h3>
          <router-link :to="getItemLink(item)">{{ item.title }}</router-link>
        </h3>
        <p class="summary" v-html="item.summary"></p>
        <div class="meta">
          <span>{{ item.type }}</span>
          <span>{{ item.views }} 浏览</span>
          <span>{{ item.likes_count }} 点赞</span>
        </div>
      </div>
      
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        @current-change="handleSearch"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { search, SearchResult } from '@/api/search'

const route = useRoute()
const router = useRouter()

const keyword = ref('')
const searchType = ref('')
const results = ref<SearchResult[]>([])
const total = ref(0)
const totalCount = ref(0)
const counts = ref({ article: 0, skill: 0, mcp_server: 0 })
const page = ref(1)
const pageSize = 20

const handleSearch = async () => {
  const res = await search({
    q: keyword.value,
    type: searchType.value,
    page: page.value,
    page_size: pageSize,
  })
  
  results.value = res.items
  total.value = res.total
  totalCount.value = res.counts 
    ? res.counts.article + res.counts.skill + res.counts.mcp_server
    : res.total
  
  if (res.counts) {
    counts.value = res.counts
  }
}

const getItemLink = (item: SearchResult) => {
  switch (item.type) {
    case 'article':
      return `/articles/${item.id}`
    case 'skill':
      return `/skills/${item.id}`
    case 'mcp_server':
      return `/mcps/${item.id}`
  }
}

onMounted(() => {
  keyword.value = route.query.q as string || ''
  searchType.value = route.query.type as string || ''
  handleSearch()
})

watch(() => route.query, (newQuery) => {
  keyword.value = newQuery.q as string || ''
  searchType.value = newQuery.type as string || ''
  page.value = 1
  handleSearch()
})
</script>
```

- [ ] **步骤 3：修改 Navbar 添加搜索框**

```vue
<!-- frontend/src/components/Navbar.vue -->
<template>
  <el-header>
    <div class="nav-content">
      <div class="logo">AIDevClub</div>
      <el-menu mode="horizontal">
        <!-- ... 现有菜单 ... -->
      </el-menu>
      
      <div class="search-box">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索..."
          @keyup.enter="handleSearch"
          clearable
        />
      </div>
    </div>
  </el-header>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const searchKeyword = ref('')

const handleSearch = () => {
  if (searchKeyword.value.trim()) {
    router.push({
      path: '/search',
      query: { q: searchKeyword.value }
    })
  }
}
</script>
```

- [ ] **步骤 4：添加路由**

```typescript
// frontend/src/router/index.ts
{
  path: '/search',
  name: 'Search',
  component: () => import('@/views/SearchPage.vue'),
}
```

- [ ] **步骤 5：运行类型检查和构建**

```bash
cd frontend
npm run typecheck
npm run build
```

预期：无错误

- [ ] **步骤 6：Commit**

```bash
git add frontend/src/views/SearchPage.vue frontend/src/api/search.ts frontend/src/components/Navbar.vue frontend/src/router/index.ts
git commit -m "feat(frontend): add search box and search results page"
```

---

## 任务 15：前端侧边栏排行组件改用排行接口

**文件：**
- 修改：`frontend/src/components/ResourceSidebar.vue`
- 创建：`frontend/src/api/ranking.ts`

- [ ] **步骤 1：创建排行 API**

```typescript
// frontend/src/api/ranking.ts
import request from '@/utils/request'

export function getArticleRanking(params: { type?: string; page?: number; page_size?: number }) {
  return request.get('/articles/ranking', { params })
}

export function getSkillRanking(params: { type?: string; page?: number; page_size?: number }) {
  return request.get('/skills/ranking', { params })
}

export function getMcpServerRanking(params: { type?: string; page?: number; page_size?: number }) {
  return request.get('/mcp-servers/ranking', { params })
}
```

- [ ] **步骤 2：修改 ResourceSidebar 组件**

```vue
<!-- frontend/src/components/ResourceSidebar.vue -->
<template>
  <div class="resource-sidebar">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="热门" name="hot">
        <div class="ranking-list">
          <div v-for="item in hotRanking" :key="item.id" class="ranking-item">
            <router-link :to="getItemLink(item)">{{ item.name || item.title }}</router-link>
            <span class="score">{{ item.views }} 浏览</span>
          </div>
        </div>
      </el-tab-pane>
      
      <el-tab-pane label="下载量" name="downloads">
        <div class="ranking-list">
          <div v-for="item in downloadRanking" :key="item.id" class="ranking-item">
            <router-link :to="getItemLink(item)">{{ item.name || item.title }}</router-link>
            <span class="score">{{ item.downloads }} 下载</span>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { getSkillRanking, getMcpServerRanking } from '@/api/ranking'

const props = defineProps<{
  type: 'skill' | 'mcp_server'
}>()

const activeTab = ref('hot')
const hotRanking = ref([])
const downloadRanking = ref([])

const loadRanking = async () => {
  const api = props.type === 'skill' ? getSkillRanking : getMcpServerRanking
  
  const hotRes = await api({ type: 'hot', page_size: 10 })
  hotRanking.value = hotRes.items
  
  const downloadRes = await api({ type: 'downloads', page_size: 10 })
  downloadRanking.value = downloadRes.items
}

watch(activeTab, loadRanking)
loadRanking()
</script>
```

- [ ] **步骤 3：运行类型检查和构建**

```bash
cd frontend
npm run typecheck
npm run build
```

预期：无错误

- [ ] **步骤 4：Commit**

```bash
git add frontend/src/components/ResourceSidebar.vue frontend/src/api/ranking.ts
git commit -m "feat(frontend): update sidebar to use ranking API"
```

---

## 任务 16：前端标签管理页面（管理员）

**文件：**
- 创建：`frontend/src/views/admin/TagManagement.vue`
- 创建：`frontend/src/api/adminTag.ts`
- 修改：`frontend/src/router/index.ts`

- [ ] **步骤 1：创建管理员标签 API**

```typescript
// frontend/src/api/adminTag.ts
import request from '@/utils/request'

export interface Tag {
  id: number
  name: string
  description: string
  usage_count: number
  enabled: boolean
}

export function getAdminTags(params: { keyword?: string; status?: string; page?: number; page_size?: number }) {
  return request.get('/admin/tags', { params })
}

export function createTag(data: { name: string; description: string }) {
  return request.post('/admin/tags', data)
}

export function updateTag(id: number, data: { name: string; description: string }) {
  return request.put(`/admin/tags/${id}`, data)
}

export function enableTag(id: number) {
  return request.patch(`/admin/tags/${id}/enable`)
}

export function disableTag(id: number) {
  return request.patch(`/admin/tags/${id}/disable`)
}
```

- [ ] **步骤 2：创建标签管理页面**

```vue
<!-- frontend/src/views/admin/TagManagement.vue -->
<template>
  <div class="tag-management">
    <div class="header">
      <h1>标签管理</h1>
      <el-button type="primary" @click="showCreateDialog">创建标签</el-button>
    </div>
    
    <div class="filters">
      <el-input v-model="keyword" placeholder="搜索标签" @input="loadTags" clearable />
      <el-select v-model="status" @change="loadTags">
        <el-option label="全部" value="all" />
        <el-option label="已启用" value="enabled" />
        <el-option label="已禁用" value="disabled" />
      </el-select>
    </div>
    
    <el-table :data="tags">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="description" label="描述" />
      <el-table-column prop="usage_count" label="使用次数" />
      <el-table-column label="状态">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'danger'">
            {{ row.enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作">
        <template #default="{ row }">
          <el-button size="small" @click="showEditDialog(row)">编辑</el-button>
          <el-button 
            size="small" 
            :type="row.enabled ? 'danger' : 'success'"
            @click="toggleTag(row)"
          >
            {{ row.enabled ? '禁用' : '启用' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    
    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      @current-change="loadTags"
    />
    
    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editingTag ? '编辑标签' : '创建标签'">
      <el-form :model="form">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAdminTags, createTag, updateTag, enableTag, disableTag, Tag } from '@/api/adminTag'

const tags = ref<Tag[]>([])
const keyword = ref('')
const status = ref('all')
const page = ref(1)
const pageSize = 20
const total = ref(0)

const dialogVisible = ref(false)
const editingTag = ref<Tag | null>(null)
const form = ref({ name: '', description: '' })

const loadTags = async () => {
  const res = await getAdminTags({
    keyword: keyword.value,
    status: status.value,
    page: page.value,
    page_size: pageSize,
  })
  tags.value = res.items
  total.value = res.total
}

const showCreateDialog = () => {
  editingTag.value = null
  form.value = { name: '', description: '' }
  dialogVisible.value = true
}

const showEditDialog = (tag: Tag) => {
  editingTag.value = tag
  form.value = { name: tag.name, description: tag.description }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (editingTag.value) {
    await updateTag(editingTag.value.id, form.value)
  } else {
    await createTag(form.value)
  }
  dialogVisible.value = false
  loadTags()
}

const toggleTag = async (tag: Tag) => {
  if (tag.enabled) {
    await disableTag(tag.id)
  } else {
    await enableTag(tag.id)
  }
  loadTags()
}

onMounted(loadTags)
</script>
```

- [ ] **步骤 3：添加路由**

```typescript
// frontend/src/router/index.ts
{
  path: '/admin/tags',
  name: 'AdminTags',
  component: () => import('@/views/admin/TagManagement.vue'),
  meta: { requiresAdmin: true }
}
```

- [ ] **步骤 4：运行类型检查和构建**

```bash
cd frontend
npm run typecheck
npm run build
```

预期：无错误

- [ ] **步骤 5：Commit**

```bash
git add frontend/src/views/admin/TagManagement.vue frontend/src/api/adminTag.ts frontend/src/router/index.ts
git commit -m "feat(frontend): add admin tag management page"
```

---

## 任务 17：配置项和最终测试

**文件：**
- 修改：`internal/platform/config.go`
- 修改：`config.yaml`（如果存在）

- [ ] **步骤 1：添加 ranking 配置**

```go
// internal/platform/config.go
type RankingConfig struct {
    Gravity    float64       `mapstructure:"gravity"`
    RecalcInterval time.Duration `mapstructure:"recalc_interval"`
}

type Config struct {
    // ... 现有配置 ...
    Ranking RankingConfig `mapstructure:"ranking"`
}

func LoadConfig() (*Config, error) {
    // ... 现有代码 ...
    
    // 默认值
    cfg.Ranking.Gravity = 1.5
    cfg.Ranking.RecalcInterval = 2 * time.Minute
    
    return cfg, nil
}
```

- [ ] **步骤 2：完整测试**

运行所有后端测试：
```bash
go test ./... -v
```

预期：所有测试 PASS

运行前端类型检查和构建：
```bash
cd frontend
npm run typecheck
npm run build
```

预期：无错误

- [ ] **步骤 3：手动集成测试**

启动服务：
```bash
docker compose up -d
go run ./cmd/server
```

测试场景：
1. 创建标签：`POST /api/v1/admin/tags`
2. 搜索文章：`GET /api/v1/search?q=Go&type=article`
3. 获取热榜：`GET /api/v1/articles/ranking?type=hot`
4. 点赞文章后检查热榜更新

- [ ] **步骤 4：Commit**

```bash
git add internal/platform/config.go
git commit -m "feat(config): add ranking configuration"
```

---

## 任务 18：更新文档和路线图

**文件：**
- 修改：`docs/roadmap.md`
- 创建：`docs/phase4-summary.md`

- [ ] **步骤 1：更新路线图**

```markdown
<!-- docs/roadmap.md -->
| P4 | 标签 / 搜索 / 排行优化 | §10 / §14 | ✅ 已完成 |
```

- [ ] **步骤 2：创建 P4 阶段总结**

```markdown
<!-- docs/phase4-summary.md -->
# P4 阶段总结：标签/搜索/排行优化

## 完成时间
2026-08-25

## 主要功能

### 1. 标签管理优化
- 管理员标签 CRUD 接口
- 标签 description 字段
- 热门标签 Redis 缓存
- 前缀搜索支持

### 2. 全文搜索
- MySQL FULLTEXT 索引（ngram 中文分词）
- 统一搜索接口 `/api/v1/search`
- 搜索结果高亮
- 组合筛选（关键词 + 标签 + 分类）

### 3. 热门排行优化
- Hacker News 风格时间衰减算法
- 定时预计算（每 2 分钟）
- Redis ZSet 缓存
- 互动时实时更新分数
- 独立排行接口

### 4. 前端适配
- 统一搜索框
- 搜索结果页
- 侧边栏排行组件改用排行接口
- 管理员标签管理页面

## 技术亮点
1. MySQL FULLTEXT + ngram 支持中文全文搜索
2. Redis ZSet 实现高效排行
3. 时间衰减算法让新内容更容易上榜
4. 定时预计算 + 实时更新保证性能和实时性

## 文件统计
- 新增文件：约 15 个
- 代码变更：约 +3000 行

## 下一步
P5 阶段将实现消息通知和举报审核功能。
```

- [ ] **步骤 3：Commit**

```bash
git add docs/roadmap.md docs/phase4-summary.md
git commit -m "docs: P4 阶段总结，更新路线图"
```

---

## 完成

所有任务完成后，P4 阶段的标签/搜索/排行优化功能将全部实现。

**验证清单：**
- [ ] 所有后端测试通过：`go test ./...`
- [ ] 前端类型检查和构建通过：`npm run typecheck && npm run build`
- [ ] 管理员标签管理功能正常
- [ ] 全文搜索功能正常（支持中文）
- [ ] 热门排行功能正常（时间衰减）
- [ ] 前端搜索框和搜索结果页正常
- [ ] 侧边栏排行组件正常
