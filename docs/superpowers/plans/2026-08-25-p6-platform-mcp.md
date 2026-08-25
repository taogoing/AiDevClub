# P6 Platform MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a separately deployable, read-only AIDevClub MCP Server that exposes six public content tools and three authenticated account tools by reusing existing domain services.

**Architecture:** Extract the existing application wiring into `internal/app`, then run the REST API and MCP endpoints as separate processes over the same Service/Repo layer. The MCP process uses the official Go SDK with stateless JSON Streamable HTTP, request-bound JWT actors, explicit tool schemas and side-effect-free read methods.

**Tech Stack:** Go 1.25.0, `github.com/modelcontextprotocol/go-sdk/mcp` v1.7.x, net/http, Gin, GORM, MySQL 8, Redis 7, miniredis, httptest.

**Spec:** `docs/superpowers/specs/2026-08-25-p6-mcp-admin-design.md`

## Global Constraints

- Only implement the nine read-only tools named in the spec; do not add likes, favorites, reports, profile mutation or notification mutation.
- Pin the official MCP Go SDK to the latest patch in v1.7.x; do not introduce FastMCP, a second MCP framework or OAuth.
- Keep the repository's current Go directive `go 1.25.0`; do not downgrade it to the task brief's older 1.24 value.
- Use `Stateless: true`, `JSONResponse: true` and `PropagateRequestCancellation: true` for Streamable HTTP.
- Anonymous callers see six public tools; valid Bearer callers see all nine; invalid or expired Bearer credentials return HTTP 401 and never fall back to anonymous.
- MCP reads never increment views/downloads and never query interaction state that is absent from MCP output.
- Do not add a DI framework, BaseService, BaseRepository, generic CRUD layer, custom Tool framework or one-file-per-tool layout.
- Every repository query receives the request context; Tool handlers do not start detached goroutines.
- Run Go commands with a repository-local cache, for example `$env:GOCACHE="$PWD/.gocache"`, because the host Go cache is not writable in this environment.

---

### Task 1: Shared application composition and process lifecycle

**Files:**
- Create: `internal/app/infrastructure.go`
- Create: `internal/app/migrations.go`
- Create: `internal/app/services.go`
- Create: `internal/app/server.go`
- Modify: `internal/platform/config.go`
- Modify: `internal/platform/ratelimit.go`
- Modify: `internal/platform/ratelimit_test.go`
- Modify: `internal/scheduler/ranking.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `app.OpenInfrastructure(*platform.Config) (*app.Infrastructure, error)`, `(*Infrastructure).Ping(context.Context) error`, `(*Infrastructure).Close() error`, `app.Migrate(*gorm.DB) error`, `app.NewServices(*Infrastructure, *platform.Config) *app.Services`, `platform.NewRateLimiter(*redis.Client, int, time.Duration) *platform.RateLimiter`, `(*RateLimiter).Allow(context.Context, string) (bool, error)`.
- Consumes: existing Repo and Service constructors without changing their domain behavior.

- [ ] **Step 1: Write failing configuration, limiter and shutdown tests**

```go
func TestLoadConfigMCPDefaults(t *testing.T) {
    t.Setenv("AIDEVCLUB_MCP_ADDR", ":9091")
    cfg, err := LoadConfig()
    require.NoError(t, err)
    assert.Equal(t, ":9091", cfg.MCPAddr)
    assert.Equal(t, 60, cfg.MCPRateLimitPerMin)
    assert.Equal(t, int64(1<<20), cfg.MCPMaxBodyBytes)
    assert.Equal(t, 30*time.Second, cfg.MCPRequestTimeout)
}

func TestRateLimiterUsesProvidedIdentity(t *testing.T) {
    mr := miniredis.RunT(t)
    limiter := NewRateLimiter(redis.NewClient(&redis.Options{Addr: mr.Addr()}), 1, time.Minute)
    ok, err := limiter.Allow(context.Background(), "mcp:user:7")
    require.NoError(t, err)
    assert.True(t, ok)
    ok, err = limiter.Allow(context.Background(), "mcp:user:7")
    require.NoError(t, err)
    assert.False(t, ok)
}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/platform ./internal/scheduler -run 'TestLoadConfigMCPDefaults|TestRateLimiterUsesProvidedIdentity|TestRankingSchedulerStop' -v
```

Expected: FAIL because the MCP configuration fields, reusable limiter and idempotent scheduler stop do not exist.

- [ ] **Step 3: Implement the minimal shared composition**

```go
type Infrastructure struct {
    DB    *gorm.DB
    Redis *redis.Client
}

type Services struct {
    UserRepo      *repo.UserRepo
    Auth          *service.AuthService
    Users         *service.UserService
    Articles      *service.ArticleService
    Comments      *service.CommentService
    Skills        *service.SkillService
    MCPServers    *service.McpServerService
    ResourceComments *service.ResourceCommentService
    Search        *service.SearchService
    Ranking       *service.RankingService
    Categories    *service.CategoryService
    Tags          *service.TagService
    Notifications *service.NotificationService
    Reports       *service.ReportService
    Admin         *service.AdminService
    AdminLogs     *service.AdminLogService
}

type RateLimiter struct {
    rdb    *redis.Client
    limit  int
    window time.Duration
}

func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error)
```

Move the existing constructor order from `cmd/server/main.go` into `app.NewServices` exactly once. Keep route registration in `cmd/server`; move `AutoMigrate` and FULLTEXT creation into `app.Migrate`; use explicit `http.Server` timeouts and close the scheduler before infrastructure shutdown.

- [ ] **Step 4: Run platform tests and build both current packages**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/platform ./internal/scheduler ./internal/app -v
go build ./cmd/server
```

Expected: PASS; existing REST routes retain their behavior.

- [ ] **Step 5: Commit the composition refactor**

```powershell
git add cmd/server internal/app internal/platform/config.go internal/platform/ratelimit.go internal/platform/ratelimit_test.go internal/scheduler/ranking.go
git commit -m "refactor: share application composition and lifecycle"
```

---

### Task 2: Secure Skill ZIP metadata extraction

**Files:**
- Modify: `internal/model/skill.go`
- Create: `internal/service/skill_zip.go`
- Create: `internal/service/skill_zip_test.go`
- Modify: `internal/service/skill.go`
- Modify: `internal/service/skill_test.go`
- Modify: `internal/app/migrations.go`

**Interfaces:**
- Produces: `Skill.SkillMD string`, `extractSkillMD(io.ReaderAt, int64) (string, error)` and atomic persistence of ZIP metadata plus `SkillMD`.
- Consumes: existing `SkillService.UploadZip` and `platform.ErrInvalidInput` conventions.

- [ ] **Step 1: Write failing ZIP validation tests**

```go
func TestExtractSkillMD(t *testing.T) {
    zipBytes := makeZip(t, map[string]string{"demo/SKILL.md": "# Demo\nUse it."})
    got, err := extractSkillMD(bytes.NewReader(zipBytes), int64(len(zipBytes)))
    require.NoError(t, err)
    assert.Equal(t, "# Demo\nUse it.", got)
}

func TestExtractSkillMDRejectsUnsafeArchive(t *testing.T) {
    cases := []map[string]string{
        {"README.md": "missing"},
        {"../SKILL.md": "escape"},
        {"a/SKILL.md": "one", "b/SKILL.md": "two"},
    }
    for _, files := range cases {
        zipBytes := makeZip(t, files)
        _, err := extractSkillMD(bytes.NewReader(zipBytes), int64(len(zipBytes)))
        assert.ErrorIs(t, err, platform.ErrInvalidInput)
    }
}
```

Also cover encrypted/unsupported entries, symlinks, decompressed total above 10 MiB, more than 256 entries and `SKILL.md` above 1 MiB.

- [ ] **Step 2: Run the focused test and verify it fails**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service -run 'TestExtractSkillMD|TestSkillUpload' -v
```

Expected: FAIL because `extractSkillMD` and `SkillMD` do not exist.

- [ ] **Step 3: Implement streaming ZIP validation and persistence**

```go
const (
    maxArchiveEntries = 256
    maxExpandedBytes   = 10 << 20
    maxSkillMDBytes    = 1 << 20
)

func extractSkillMD(r io.ReaderAt, size int64) (string, error) {
    zr, err := zip.NewReader(r, size)
    if err != nil || len(zr.File) > maxArchiveEntries { return "", platform.ErrInvalidInput }
    // Normalize each path, reject absolute/traversal/symlink entries, count
    // expanded bytes through io.LimitReader, and require exactly one SKILL.md.
}
```

`UploadZip` must validate the saved temporary file before publishing it, then update `zip_url`, `zip_filename`, `file_size` and `skill_md` in one DB update. On failure, remove only the newly uploaded temporary file and leave old metadata unchanged.

- [ ] **Step 4: Run Skill tests**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/repo -run 'TestExtractSkillMD|TestSkill' -v
```

Expected: PASS, including old Skill status-flow tests.

- [ ] **Step 5: Commit ZIP extraction**

```powershell
git add internal/model/skill.go internal/service/skill_zip.go internal/service/skill_zip_test.go internal/service/skill.go internal/service/skill_test.go internal/app/migrations.go
git commit -m "feat: validate skill archives and store SKILL.md"
```

---

### Task 3: Side-effect-free public and owned domain reads

**Files:**
- Modify: `internal/service/dto.go`
- Modify: `internal/service/article.go`
- Modify: `internal/service/article_test.go`
- Modify: `internal/service/skill.go`
- Modify: `internal/service/skill_test.go`
- Modify: `internal/service/mcp_server.go`
- Modify: `internal/service/mcp_server_test.go`
- Modify: `internal/repo/article.go`
- Modify: `internal/repo/skill.go`
- Modify: `internal/repo/mcp_server.go`

**Interfaces:**
- Produces: `Read(ctx context.Context, actorID, id uint)`, `ListOwned(ctx context.Context, actorID uint, status string, page, pageSize int)` on Article/Skill/McpServer services.
- Consumes: existing visibility rules and summary/detail DTO assembly.

- [ ] **Step 1: Write failing no-side-effect and ownership tests**

```go
func TestArticleReadDoesNotIncrementViewsOrLoadInteractions(t *testing.T) {
    before := article.Views
    detail, err := svc.Read(ctx, 0, article.ID)
    require.NoError(t, err)
    assert.Equal(t, before, detail.Views)
    require.NoError(t, db.First(&article, article.ID).Error)
    assert.Equal(t, before, article.Views)
}

func TestSkillListOwnedIncludesDraftRejectedAndOwnHidden(t *testing.T) {
    got, err := svc.ListOwned(ctx, owner.ID, "", 1, 20)
    require.NoError(t, err)
    assert.ElementsMatch(t, []uint{draft.ID, rejected.ID, hidden.ID}, ids(got.List))
}
```

Repeat the visibility assertions for article and MCP Server, and assert another actor cannot read hidden/unpublished data.

- [ ] **Step 2: Run focused service tests and verify they fail**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service -run 'Test(Article|Skill|McpServer)(Read|ListOwned)' -v
```

Expected: FAIL because `Read` and `ListOwned` do not exist.

- [ ] **Step 3: Implement explicit readers without a public options object**

```go
func (s *ArticleService) Get(ctx context.Context, actorID, id uint) (*ArticleDetail, error) {
    return s.detail(ctx, actorID, id, true, true)
}

func (s *ArticleService) Read(ctx context.Context, actorID, id uint) (*ArticleDetail, error) {
    return s.detail(ctx, actorID, id, false, false)
}

func (s *ArticleService) ListOwned(ctx context.Context, actorID uint, status string, page, pageSize int) (*ArticleListResult, error)
func (s *SkillService) ListOwned(ctx context.Context, actorID uint, status string, page, pageSize int) (*SkillListResult, error)
func (s *McpServerService) ListOwned(ctx context.Context, actorID uint, status string, page, pageSize int) (*McpServerListResult, error)
```

Private `detail` methods use booleans only inside the service. Repo owned queries always bind `author_id = actorID`, include own hidden rows, reject unknown statuses and exclude soft-deleted rows.

- [ ] **Step 4: Run service and repo tests**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/repo -run 'Test(Article|Skill|McpServer)' -v
```

Expected: PASS.

- [ ] **Step 5: Commit domain readers**

```powershell
git add internal/service internal/repo/article.go internal/repo/skill.go internal/repo/mcp_server.go
git commit -m "feat: add side-effect-free and owned content readers"
```

---

### Task 4: Search, browse, taxonomy and unread-filter readers

**Files:**
- Modify: `internal/service/search.go`
- Modify: `internal/service/ranking.go`
- Modify: `internal/service/category.go`
- Modify: `internal/service/tag.go`
- Modify: `internal/service/notification.go`
- Modify: `internal/repo/notification.go`
- Create: `internal/service/mcp_read_test.go`

**Interfaces:**
- Produces: search calls with `highlight bool`, ranking ID retrieval/batch hydration, taxonomy filtering, `NotificationService.List(..., unreadOnly bool, ...)`.
- Consumes: existing MySQL FULLTEXT indexes and Redis time-decayed ZSets.

- [ ] **Step 1: Write failing reader tests**

```go
func TestNotificationListUnreadFilterOwnsCountAndPage(t *testing.T) {
    got, err := svc.List(ctx, user.ID, "", true, 1, 10)
    require.NoError(t, err)
    assert.EqualValues(t, 2, got.Total)
    assert.Len(t, got.List, 2)
}

func TestSearchWithoutHighlightReturnsPlainText(t *testing.T) {
    got, err := search.Search(ctx, SearchQuery{Keyword: "Go", Type: "article", Highlight: false})
    require.NoError(t, err)
    assert.NotContains(t, got.Articles[0].Summary, "<em>")
}
```

Add ranking tests that preserve Redis ID order after batch loading and drop hidden/deleted rows.

- [ ] **Step 2: Run and observe the failing tests**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/repo -run 'TestNotificationListUnread|TestSearchWithoutHighlight|TestRankingBatch' -v
```

Expected: FAIL on missing parameters/methods.

- [ ] **Step 3: Implement only the readers required by MCP**

```go
type SearchQuery struct {
    Keyword, ContentType, Sort string
    TagID, CategoryID          *uint
    Page, PageSize             int
    Highlight                  bool
}

func (s *NotificationService) List(ctx context.Context, userID uint, notifType string, unreadOnly bool, page, pageSize int) (*NotificationListResult, error)
func (r *NotificationRepo) List(ctx context.Context, userID uint, notifType string, unreadOnly bool, page, pageSize int) ([]model.Notification, int64, error)
```

Keep three content types as separate result sections for `all`. For hot browse, read ordered IDs from RankingService, batch load summaries once per type, index them by ID and rebuild the result in ZSet order.

- [ ] **Step 4: Run focused and existing notification/search tests**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/repo ./internal/handler -run 'Test(Notification|Search|Ranking|Tag|Category)' -v
```

Expected: PASS; update existing REST handler calls to pass `false` for `unreadOnly`.

- [ ] **Step 5: Commit MCP reader support**

```powershell
git add internal/service internal/repo/notification.go internal/handler/notification.go internal/handler/notification_test.go
git commit -m "feat: add MCP search browse and notification readers"
```

---

### Task 5: Public MCP tools

**Files:**
- Create: `internal/mcpserver/dependencies.go`
- Create: `internal/mcpserver/output.go`
- Create: `internal/mcpserver/errors.go`
- Create: `internal/mcpserver/tool_content.go`
- Create: `internal/mcpserver/tool_resource.go`
- Create: `internal/mcpserver/tool_taxonomy.go`
- Create: `internal/mcpserver/public_tools_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Produces: `RegisterPublicTools(*mcp.Server, PublicDependencies, string)`, the six public tools and stable MCP error codes.
- Consumes: the reader methods from Tasks 3-4.

- [ ] **Step 1: Pin the SDK and write failing fake-reader tool tests**

```powershell
go get github.com/modelcontextprotocol/go-sdk@v1.7.0
```

```go
func TestGetArticleUsesReadAndReturnsWindow(t *testing.T) {
    fake := &fakeArticleReader{detail: &service.ArticleDetail{Content: strings.Repeat("界", 20)}}
    server := newTestServer(PublicDependencies{Articles: fake})
    out := callTool[getArticleOutput](t, server, "get_article", map[string]any{
        "id": 9, "content_offset": 5, "content_limit": 7,
    })
    assert.Equal(t, "界界界界界界界", out.Content)
    assert.True(t, out.HasMore)
    assert.Equal(t, 12, out.NextOffset)
    assert.Equal(t, 1, fake.readCalls)
}
```

Add table tests for invalid content type/sort, `all` grouping, no HTML highlight, category/type compatibility, exact absolute URLs, empty arrays and structured `tools_json`.

- [ ] **Step 2: Run the MCP package tests and verify failure**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/mcpserver -run 'Test(Search|Browse|Get|ListTaxonomy)' -v
```

Expected: FAIL because the package and tools do not exist.

- [ ] **Step 3: Implement explicit schemas and grouped handlers**

```go
type ContentWindowInput struct {
    ContentOffset int `json:"content_offset,omitempty" jsonschema:"minimum=0"`
    ContentLimit  int `json:"content_limit,omitempty" jsonschema:"minimum=1,maximum=50000"`
}

func RegisterPublicTools(server *mcp.Server, deps PublicDependencies, publicBaseURL string) {
    mcp.AddTool(server, &mcp.Tool{Name: "search_content", Description: "Search published AIDevClub content."}, searchContent(deps, publicBaseURL))
    mcp.AddTool(server, &mcp.Tool{Name: "browse_content", Description: "Browse latest or hot AIDevClub content."}, browseContent(deps, publicBaseURL))
    mcp.AddTool(server, &mcp.Tool{Name: "get_article", Description: "Read a published article without changing its view count."}, getArticle(deps.Articles, publicBaseURL))
    mcp.AddTool(server, &mcp.Tool{Name: "get_skill", Description: "Read a published Skill and its SKILL.md."}, getSkill(deps.Skills, publicBaseURL))
    mcp.AddTool(server, &mcp.Tool{Name: "get_mcp_server", Description: "Read a published MCP Server definition."}, getMCPServer(deps.MCPServers, publicBaseURL))
    mcp.AddTool(server, &mcp.Tool{Name: "list_taxonomy", Description: "List enabled tags and article categories."}, listTaxonomy(deps, publicBaseURL))
}
```

Handlers use the official generic signature `func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)`. Keep input/output types next to their domain handler; `output.go` contains only reused `PageInfo`, `AuthorOutput`, `TagOutput` and the Unicode window helper.

- [ ] **Step 4: Run public Tool tests and vet**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/mcpserver -run 'Test(Search|Browse|Get|ListTaxonomy)' -v
go vet ./internal/mcpserver
```

Expected: PASS.

- [ ] **Step 5: Commit public tools**

```powershell
git add go.mod go.sum internal/mcpserver
git commit -m "feat(mcp): add public read-only content tools"
```

---

### Task 6: Authenticated account tools and request-bound tool visibility

**Files:**
- Create: `internal/mcpserver/auth.go`
- Create: `internal/mcpserver/tool_account.go`
- Create: `internal/mcpserver/account_tools_test.go`
- Modify: `internal/mcpserver/dependencies.go`

**Interfaces:**
- Produces: `ActorFromContext(context.Context) Actor`, `RegisterAccountTools(*mcp.Server, AccountDependencies, Actor, string)` and tools `get_my_profile`, `list_my_content`, `list_my_notifications`.
- Consumes: JWT-derived actor, `UserService.Get`, `ListOwned` readers and unread notification reader.

- [ ] **Step 1: Write failing actor-binding tests**

```go
func TestListMyContentNeverAcceptsUserID(t *testing.T) {
    fake := &fakeOwnedReader{}
    actor := Actor{UserID: 42, Authenticated: true}
    out := callAccountTool(t, actor, fake, "list_my_content", map[string]any{
        "content_type": "article", "page": 1, "page_size": 10,
    })
    assert.Equal(t, uint(42), fake.actorID)
    assert.NotNil(t, out)
}

func TestListMyNotificationsDoesNotMarkRead(t *testing.T) {
    fake := &fakeNotificationReader{}
    callAccountTool(t, Actor{UserID: 7, Authenticated: true}, fake, "list_my_notifications", map[string]any{"unread_only": true})
    assert.Equal(t, 0, fake.markReadCalls)
}
```

- [ ] **Step 2: Run account tests and verify failure**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/mcpserver -run 'Test(GetMy|ListMy)' -v
```

Expected: FAIL because account tools do not exist.

- [ ] **Step 3: Implement the three actor-scoped tools**

```go
type Actor struct {
    UserID        uint
    Authenticated bool
}

type ListMyContentInput struct {
    ContentType string `json:"content_type" jsonschema:"required,enum=article,enum=skill,enum=mcp_server"`
    Status      string `json:"status,omitempty"`
    Page        int    `json:"page,omitempty" jsonschema:"minimum=1"`
    PageSize    int    `json:"page_size,omitempty" jsonschema:"minimum=1,maximum=20"`
}
```

Tool inputs contain no `user_id`. Resolve the Actor from the request-created server closure, re-check the user exists through `ProfileReader`, and return `not_authenticated`, `not_found`, `invalid_argument` or `internal_error` without leaking database messages.

- [ ] **Step 4: Run all Tool unit tests**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/mcpserver -v
```

Expected: PASS.

- [ ] **Step 5: Commit account tools**

```powershell
git add internal/mcpserver
git commit -m "feat(mcp): add authenticated account read tools"
```

---

### Task 7: Streamable HTTP server, security middleware and health checks

**Files:**
- Create: `internal/mcpserver/server.go`
- Create: `internal/mcpserver/handler.go`
- Create: `internal/mcpserver/http_test.go`
- Create: `cmd/mcp-server/main.go`
- Modify: `internal/platform/config.go`

**Interfaces:**
- Produces: `mcpserver.NewHandler(Dependencies, *platform.Config, *slog.Logger) http.Handler` and runnable `cmd/mcp-server`.
- Consumes: `app.Infrastructure`, `app.Services`, reusable limiter and all tool registrars.

- [ ] **Step 1: Write failing HTTP behavior tests**

```go
func TestToolsListVariesByBearerActor(t *testing.T) {
    h := NewHandler(testDependencies(t), testConfig(), slog.Default())
    anonymous := listToolsHTTP(t, h, "")
    authenticated := listToolsHTTP(t, h, "Bearer "+validJWT(t, 9))
    assert.Len(t, anonymous, 6)
    assert.Len(t, authenticated, 9)
}

func TestInvalidBearerIsNotAnonymous(t *testing.T) {
    req := newMCPRequest(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
    req.Header.Set("Authorization", "Bearer broken")
    rr := httptest.NewRecorder()
    NewHandler(testDependencies(t), testConfig(), slog.Default()).ServeHTTP(rr, req)
    assert.Equal(t, http.StatusUnauthorized, rr.Code)
    assert.Equal(t, "Bearer", rr.Header().Get("WWW-Authenticate"))
}
```

Also test exact Origin allowlist, no-Origin CLI access, 1 MiB body limit, per-user/IP 429, fail-open Redis errors, request cancellation, `/healthz`, startup schema failure and `/readyz` dependency failure.

- [ ] **Step 2: Run HTTP tests and verify failure**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/mcpserver -run 'Test.*(Bearer|Origin|Body|Rate|Health|Ready|Cancellation)' -v
```

Expected: FAIL because the HTTP handler does not exist.

- [ ] **Step 3: Implement request-scoped server creation and middleware**

```go
func newMCPServer(deps Dependencies, actor Actor, cfg *platform.Config) *mcp.Server {
    server := mcp.NewServer(&mcp.Implementation{Name: "aidevclub", Version: buildVersion}, nil)
    RegisterPublicTools(server, deps.Public, cfg.PublicBaseURL)
    if actor.Authenticated {
        RegisterAccountTools(server, deps.Account, actor, cfg.PublicBaseURL)
    }
    return server
}

sdkHandler := mcp.NewStreamableHTTPHandler(
    func(r *http.Request) *mcp.Server { return newMCPServer(deps, actorFromRequest(r), cfg) },
    &mcp.StreamableHTTPOptions{
        Stateless: true, JSONResponse: true, PropagateRequestCancellation: true,
        MaxRequestBodyBytes: cfg.MCPMaxBodyBytes,
    },
)
```

Wrap in this exact order: request ID → recovery → Origin → Bearer authentication → rate limit → timeout → SDK handler. `/healthz` returns 200 without dependencies; `/readyz` pings MySQL and Redis. `cmd/mcp-server` validates required tables once before ListenAndServe and uses a 10-second graceful shutdown.

- [ ] **Step 4: Run protocol, HTTP and binary tests**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/mcpserver ./internal/app -v
go build ./cmd/mcp-server ./cmd/server
```

Expected: PASS; both binaries compile.

- [ ] **Step 5: Commit the MCP process**

```powershell
git add cmd/mcp-server internal/mcpserver internal/platform/config.go
git commit -m "feat(mcp): serve authenticated stateless Streamable HTTP"
```

---

### Task 8: End-to-end verification and MCP documentation

**Files:**
- Create: `docs/mcp-server.md`
- Modify: `docker-compose.yml`
- Modify: `CLAUDE.md`

**Interfaces:**
- Produces: reproducible local startup and client configuration for anonymous and Bearer access.
- Consumes: completed REST and MCP binaries.

- [ ] **Step 1: Add an SDK protocol integration test**

```go
func TestProtocolListAndCall(t *testing.T) {
    ts := httptest.NewServer(NewHandler(testDependencies(t), testConfig(), slog.Default()))
    defer ts.Close()
    client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
    session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
    require.NoError(t, err)
    defer session.Close()
    tools, err := session.ListTools(context.Background(), nil)
    require.NoError(t, err)
    assert.Len(t, tools.Tools, 6)
}
```

- [ ] **Step 2: Run the complete backend verification**

```powershell
docker compose up -d
$env:GOCACHE="$PWD/.gocache"
go test ./...
go vet ./...
go build ./...
```

Expected: all commands exit 0. If Docker is unavailable, record that infrastructure-dependent tests were not executed; do not claim they passed.

- [ ] **Step 3: Document exact configuration and client examples**

```json
{
  "mcpServers": {
    "aidevclub": {
      "url": "http://localhost:8081/mcp",
      "headers": { "Authorization": "Bearer <access-token>" }
    }
  }
}
```

Document all nine tools, anonymous/authenticated visibility, environment variables, health endpoints, read-only semantics and the fact that file metadata links to platform pages rather than returning filesystem paths.

- [ ] **Step 4: Commit MCP docs and integration coverage**

```powershell
git add internal/mcpserver docs/mcp-server.md docker-compose.yml CLAUDE.md
git commit -m "docs: add MCP server setup and verification"
```
