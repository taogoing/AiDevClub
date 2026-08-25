# P6 Admin Console Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the missing administrator APIs and ship a Vue 3 management console for users, published content, comments, resource review, tags, reports, dashboard statistics, announcements and audit logs.

**Architecture:** Extend the existing Admin Handler → Service → Repo flow with explicit query DTOs and separate article/resource-comment pagination. The Vue app keeps `/admin` inside the existing SPA, restores the current user before route authorization, and implements focused business pages backed by one typed admin API module.

**Tech Stack:** Go 1.25.0, Gin, GORM, MySQL 8, Redis 7, Vue 3, TypeScript, Vite, Element Plus, Pinia, Vue Router.

**Spec:** `docs/superpowers/specs/2026-08-25-p6-mcp-admin-design.md`

## Global Constraints

- User roles remain exactly `user` and `admin`; do not add normal/muted/restricted/banned states or related controls.
- Admin article management contains published rows only, including visible and hidden; it never exposes user drafts.
- Article comments and resource comments use separate Tabs and independent pagination; never merge tables into fake global pagination.
- Admin detail reads never increment views or downloads.
- An administrator cannot change their own role; configured `admin.emails` bootstrap administrators cannot be demoted; unchanged role requests do not write logs.
- Rejecting a Skill/MCP Server requires a 1-500 Unicode-character reason; draft resources never appear in admin review lists.
- Existing tag writes must be protected by admin authentication and must record actual changed fields.
- Do not add BaseRepository/BaseService, generic CRUD configuration, universal table/form components, placeholder pages or unused extension points.
- Frontend API and DTO types are explicit and must not expose GORM models directly.
- Run Go commands with `$env:GOCACHE="$PWD/.gocache"`; final frontend verification is `npm run typecheck` followed by `npm run build`.

---

### Task 1: Administrator users, dashboard accuracy and audit enrichment

**Files:**
- Modify: `internal/service/dto.go`
- Modify: `internal/repo/user.go`
- Modify: `internal/repo/article.go`
- Modify: `internal/repo/skill.go`
- Modify: `internal/repo/mcp_server.go`
- Modify: `internal/repo/admin_log.go`
- Modify: `internal/service/admin.go`
- Modify: `internal/service/admin_log.go`
- Modify: `internal/handler/admin.go`
- Modify: `internal/handler/admin_test.go`

**Interfaces:**
- Produces: `AdminService.ListUsers`, `AdminService.UpdateUserRole`, accurate `Dashboard`, enriched `AdminLogItem.Admin` and routes `GET /users`, `PUT /users/:id/role`.
- Consumes: `Config.AdminEmails`, existing `AdminLogService.Log`, user roles and admin middleware.

- [ ] **Step 1: Write failing service/handler tests**

```go
func TestAdminUpdateUserRoleSafety(t *testing.T) {
    err := svc.UpdateUserRole(ctx, admin.ID, admin.ID, model.UserRoleUser)
    assert.ErrorIs(t, err, platform.ErrForbidden)
    err = svc.UpdateUserRole(ctx, admin.ID, bootstrap.ID, model.UserRoleUser)
    assert.ErrorIs(t, err, platform.ErrForbidden)
    require.NoError(t, svc.UpdateUserRole(ctx, admin.ID, member.ID, model.UserRoleAdmin))
    assert.Equal(t, model.UserRoleAdmin, reloadUser(t, db, member.ID).Role)
}

func TestDashboardReturnsErrorsInsteadOfPartialZeroes(t *testing.T) {
    svc := newAdminServiceWithBrokenArticleCounter(t)
    _, err := svc.Dashboard(context.Background())
    assert.Error(t, err)
}
```

Handler tests must cover keyword/role pagination, `role_mutable`, self-demotion 403, configured-admin demotion 403, unchanged role no-op and exactly one `update_user_role` log for a real change.

- [ ] **Step 2: Run focused tests and verify failure**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdmin(UpdateUserRole|ListUsers|Dashboard)' -v
```

Expected: FAIL because the user-management methods/routes and accurate error propagation do not exist.

- [ ] **Step 3: Implement explicit query/result types and role mutation**

```go
type AdminUserItem struct {
    ID          uint           `json:"id"`
    Email       string         `json:"email"`
    Nickname    string         `json:"nickname"`
    AvatarURL   string         `json:"avatar_url"`
    Role        model.UserRole `json:"role"`
    RoleMutable bool           `json:"role_mutable"`
    CreatedAt   time.Time      `json:"created_at"`
}

func (s *AdminService) ListUsers(ctx context.Context, keyword string, role model.UserRole, page, pageSize int) (*AdminUserListResult, error)
func (s *AdminService) UpdateUserRole(ctx context.Context, adminID, targetID uint, role model.UserRole) error
```

`Dashboard` must return an error immediately when any required count fails. Batch load admin public information for logs with one `FindPublicByIDs` call; parse valid JSON detail into `any`, otherwise preserve the legacy string.

- [ ] **Step 4: Register and verify endpoints**

```go
r.GET("/users", h.ListUsers)
r.PUT("/users/:id/role", h.UpdateUserRole)
```

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdmin(UpdateUserRole|ListUsers|Dashboard|ListLogs)' -v
```

Expected: PASS.

- [ ] **Step 5: Commit user/dashboard APIs**

```powershell
git add internal/service internal/repo internal/handler/admin.go internal/handler/admin_test.go
git commit -m "feat(admin): add user management and accurate dashboard"
```

---

### Task 2: Published article management and hidden-detail visibility

**Files:**
- Modify: `internal/service/dto.go`
- Modify: `internal/repo/article.go`
- Modify: `internal/service/article.go`
- Modify: `internal/service/admin.go`
- Modify: `internal/handler/admin.go`
- Modify: `internal/handler/admin_test.go`

**Interfaces:**
- Produces: `AdminService.ListArticles`, `AdminService.GetArticle`, routes `GET /articles`, `GET /articles/:id`, existing hide/unhide routes with correct visibility.
- Consumes: side-effect-free article `Read` path from the MCP plan.

- [ ] **Step 1: Write failing article-admin tests**

```go
func TestAdminArticleListNeverReturnsDrafts(t *testing.T) {
    result, err := svc.ListArticles(ctx, AdminArticleQuery{Page: 1, PageSize: 20})
    require.NoError(t, err)
    assert.NotContains(t, articleStatuses(result.List), model.ArticleStatusDraft)
    assert.ElementsMatch(t, []uint{published.ID, hidden.ID}, articleIDs(result.List))
}

func TestAdminCanReadHiddenArticleWithoutViewIncrement(t *testing.T) {
    before := hidden.Views
    got, err := svc.GetArticle(ctx, hidden.ID)
    require.NoError(t, err)
    assert.Equal(t, before, got.Views)
}
```

- [ ] **Step 2: Run and observe failure**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdminArticle' -v
```

Expected: FAIL on missing list/detail methods.

- [ ] **Step 3: Implement the published-only query**

```go
type AdminArticleQuery struct {
    Keyword   string
    Visibility string
    AuthorID  *uint
    Page      int
    PageSize  int
}

func (s *AdminService) ListArticles(ctx context.Context, q AdminArticleQuery) (*AdminArticleListResult, error)
func (s *AdminService) GetArticle(ctx context.Context, id uint) (*AdminArticleDetail, error)
```

The Repo query always applies `status = published`, applies `hidden` only when visibility is `visible` or `hidden`, supports title/summary keyword and author filters, and excludes soft-deleted rows. The admin DTO explicitly includes `hidden`; do not reuse the public DTO if doing so would omit moderation state.

- [ ] **Step 4: Register endpoints and run tests**

```go
r.GET("/articles", h.ListArticles)
r.GET("/articles/:id", h.GetArticle)
```

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdminArticle|TestAdminHideUnhideArticle' -v
```

Expected: PASS.

- [ ] **Step 5: Commit article management**

```powershell
git add internal/repo/article.go internal/service/article.go internal/service/admin.go internal/service/dto.go internal/handler/admin.go internal/handler/admin_test.go
git commit -m "feat(admin): add published article management"
```

---

### Task 3: Separate article and resource comment management

**Files:**
- Modify: `internal/service/dto.go`
- Modify: `internal/repo/comment.go`
- Modify: `internal/repo/resource_comment.go`
- Modify: `internal/service/admin.go`
- Modify: `internal/handler/admin.go`
- Modify: `internal/handler/admin_test.go`

**Interfaces:**
- Produces: independent list/hide/unhide methods and `/comments` plus `/resource-comments` routes.
- Consumes: existing hidden fields and direct-child cascade semantics.

- [ ] **Step 1: Write failing independent-pagination tests**

```go
func TestAdminCommentKindsHaveIndependentTotals(t *testing.T) {
    articles, err := svc.ListArticleComments(ctx, AdminCommentQuery{Page: 1, PageSize: 2})
    require.NoError(t, err)
    resources, err := svc.ListResourceComments(ctx, AdminResourceCommentQuery{ResourceType: "skill", Page: 1, PageSize: 2})
    require.NoError(t, err)
    assert.EqualValues(t, 3, articles.Total)
    assert.EqualValues(t, 5, resources.Total)
}

func TestAdminRestoreCommentOnlyRestoresTarget(t *testing.T) {
    require.NoError(t, svc.UnhideComment(ctx, admin.ID, parent.ID))
    assert.False(t, reloadComment(t, db, parent.ID).Hidden)
    assert.True(t, reloadComment(t, db, child.ID).Hidden)
}
```

- [ ] **Step 2: Run and verify failures**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdmin(Comment|ResourceComment)' -v
```

Expected: FAIL because list and moderation routes do not exist.

- [ ] **Step 3: Implement explicit, separate query paths**

```go
func (s *AdminService) ListArticleComments(ctx context.Context, q AdminCommentQuery) (*AdminCommentListResult, error)
func (s *AdminService) ListResourceComments(ctx context.Context, q AdminResourceCommentQuery) (*AdminResourceCommentListResult, error)
func (s *AdminService) HideComment(ctx context.Context, adminID, id uint) error
func (s *AdminService) UnhideComment(ctx context.Context, adminID, id uint) error
func (s *AdminService) HideResourceComment(ctx context.Context, adminID, id uint) error
func (s *AdminService) UnhideResourceComment(ctx context.Context, adminID, id uint) error
```

Both list methods support body keyword and visibility; resource comments additionally validate `resource_type` as `skill` or `mcp_server`. Hide target plus direct children in one transaction; unhide only the target. Each actual change writes one audit record.

- [ ] **Step 4: Register routes and run tests**

```go
r.GET("/comments", h.ListArticleComments)
r.PUT("/comments/:id/hide", h.HideComment)
r.PUT("/comments/:id/unhide", h.UnhideComment)
r.GET("/resource-comments", h.ListResourceComments)
r.PUT("/resource-comments/:id/hide", h.HideResourceComment)
r.PUT("/resource-comments/:id/unhide", h.UnhideResourceComment)
```

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdmin(Comment|ResourceComment)' -v
```

Expected: PASS.

- [ ] **Step 5: Commit comment management**

```powershell
git add internal/repo/comment.go internal/repo/resource_comment.go internal/service/admin.go internal/service/dto.go internal/handler/admin.go internal/handler/admin_test.go
git commit -m "feat(admin): add separate comment moderation APIs"
```

---

### Task 4: Skill and MCP Server review lists/details

**Files:**
- Modify: `internal/service/dto.go`
- Modify: `internal/repo/skill.go`
- Modify: `internal/repo/mcp_server.go`
- Modify: `internal/service/admin.go`
- Modify: `internal/handler/admin.go`
- Modify: `internal/handler/admin_test.go`

**Interfaces:**
- Produces: admin list/detail methods and `GET /skills`, `GET /skills/:id`, `GET /mcp-servers`, `GET /mcp-servers/:id`.
- Consumes: `Skill.SkillMD`, MCP Tools JSON/README and existing review/hide/unhide actions.

- [ ] **Step 1: Write failing review-list and rejection tests**

```go
func TestAdminResourceListDefaultsPendingAndExcludesDraft(t *testing.T) {
    got, err := svc.ListSkills(ctx, AdminResourceQuery{Page: 1, PageSize: 20})
    require.NoError(t, err)
    assert.Equal(t, []uint{pending.ID}, skillIDs(got.List))
}

func TestRejectResourceRequiresUnicodeReason(t *testing.T) {
    err := svc.ReviewSkill(ctx, admin.ID, pending.ID, false, "")
    assert.ErrorIs(t, err, platform.ErrInvalidInput)
    err = svc.ReviewSkill(ctx, admin.ID, pending.ID, false, strings.Repeat("界", 501))
    assert.ErrorIs(t, err, platform.ErrInvalidInput)
}
```

- [ ] **Step 2: Run the focused tests and verify failure**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdmin(Resource|Skill|McpServer)|TestRejectResource' -v
```

Expected: FAIL on missing query/detail methods and reason validation.

- [ ] **Step 3: Implement resource review readers and strict reason checks**

```go
type AdminResourceQuery struct {
    Keyword, Status string
    AuthorID        *uint
    TagID           *uint
    Page, PageSize  int
}

func (s *AdminService) ListSkills(ctx context.Context, q AdminResourceQuery) (*AdminSkillListResult, error)
func (s *AdminService) GetSkill(ctx context.Context, id uint) (*AdminSkillDetail, error)
func (s *AdminService) ListMCPServers(ctx context.Context, q AdminResourceQuery) (*AdminMCPServerListResult, error)
func (s *AdminService) GetMCPServer(ctx context.Context, id uint) (*AdminMCPServerDetail, error)
```

Default status is `pending_review`; accepted filters are `pending_review`, `published`, `rejected`, `archived`; reject `draft`. Detail DTOs include hidden/status/rejection reason/file metadata; Skill adds `skill_md`; MCP Server adds parsed `tools` plus raw-safe README. Use `utf8.RuneCountInString` for the 1-500 rule.

- [ ] **Step 4: Register routes and run tests**

```go
r.GET("/skills", h.ListSkills)
r.GET("/skills/:id", h.GetSkill)
r.GET("/mcp-servers", h.ListMCPServers)
r.GET("/mcp-servers/:id", h.GetMCPServer)
```

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdmin(Resource|Skill|McpServer)|TestRejectResource' -v
```

Expected: PASS.

- [ ] **Step 5: Commit resource review APIs**

```powershell
git add internal/repo/skill.go internal/repo/mcp_server.go internal/service/admin.go internal/service/dto.go internal/handler/admin.go internal/handler/admin_test.go
git commit -m "feat(admin): add resource review queries and details"
```

---

### Task 5: Report details, tag authorization/logs and admin route completion

**Files:**
- Modify: `internal/service/dto.go`
- Modify: `internal/repo/report.go`
- Modify: `internal/service/report.go`
- Modify: `internal/service/tag.go`
- Modify: `internal/handler/admin.go`
- Modify: `internal/handler/admin_tag.go`
- Modify: `internal/handler/admin_test.go`
- Modify: `internal/handler/tag_test.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `ReportService.AdminGet`, actor-enriched report lists, audited Tag writes and protected admin tag routes.
- Consumes: target-specific repos and `AdminLogService`.

- [ ] **Step 1: Write failing report/tag security tests**

```go
func TestAdminReportDetailLoadsOneTarget(t *testing.T) {
    got, err := reportSvc.AdminGet(ctx, report.ID)
    require.NoError(t, err)
    assert.Equal(t, report.TargetID, got.Target.ID)
    assert.NotEmpty(t, got.Target.ParentURL)
}

func TestAdminTagRoutesRequireAdminAndLogChanges(t *testing.T) {
    rr := performJSON(router, http.MethodPut, "/api/v1/admin/tags/1", `{"enabled":false}`, memberToken)
    assert.Equal(t, http.StatusForbidden, rr.Code)
    rr = performJSON(router, http.MethodPut, "/api/v1/admin/tags/1", `{"enabled":false}`, adminToken)
    assert.Equal(t, http.StatusOK, rr.Code)
    assert.Equal(t, 1, countLogs(t, db, model.AdminLogActionUpdateTag))
}
```

- [ ] **Step 2: Run focused tests and verify failure**

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/service ./internal/handler -run 'TestAdminReport|TestAdminTag' -v
```

Expected: FAIL due to missing report detail and unsecured/unlogged tag routes.

- [ ] **Step 3: Implement on-demand report detail and audited tags**

```go
func (s *ReportService) AdminGet(ctx context.Context, reportID uint) (*AdminReportDetail, error)
func (s *TagService) AdminCreate(ctx context.Context, adminID uint, name, description string) (*model.Tag, error)
func (s *TagService) AdminUpdate(ctx context.Context, adminID, id uint, updates map[string]any) error
```

Report list batches reporter public data only. `AdminGet` resolves exactly one heterogeneous target and returns target text/summary, hidden state, author and parent resource URL. Tag logs include only fields whose values actually changed; no-op updates do not log.

- [ ] **Step 4: Register routes behind the existing admin group and test**

```go
adminAuth := r.Group("/api/v1/admin", p2Auth, platform.AdminMiddleware(services.UserRepo))
adminH.RegisterRoutes(adminAuth)
adminTagH.RegisterRoutes(adminAuth)
```

```powershell
$env:GOCACHE="$PWD/.gocache"
go test ./internal/handler ./internal/service -run 'TestAdminReport|TestAdminTag|TestAdminListLogs|TestAdminAnnouncement' -v
```

Expected: PASS; no `/api/v1/admin/tags` route exists outside `adminAuth`.

- [ ] **Step 5: Commit report/tag completion**

```powershell
git add cmd/server/main.go internal/repo/report.go internal/service/report.go internal/service/tag.go internal/service/dto.go internal/handler
git commit -m "feat(admin): complete reports and secure tag management"
```

---

### Task 6: Frontend admin API, session restoration and layout

**Files:**
- Create: `frontend/src/api/admin.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/stores/auth.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/api/adminTag.ts`

**Interfaces:**
- Produces: typed functions for every admin endpoint, `auth.restoreSession(): Promise<void>` and a working `requiresAdmin` guard on the existing admin route.
- Consumes: existing Axios response interceptor, Pinia user/token state and Element Plus.

- [ ] **Step 1: Add explicit frontend DTOs and API signatures**

```ts
export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface AdminUser {
  id: number
  email: string
  nickname: string
  avatar_url: string
  role: 'user' | 'admin'
  role_mutable: boolean
  created_at: string
}

export const getAdminUsers = (params: AdminUserQuery) =>
  http.get<ApiResponse<PageResult<AdminUser>>>('/api/v1/admin/users', { params })
export const updateAdminUserRole = (id: number, role: 'user' | 'admin') =>
  http.put<ApiResponse<null>>(`/api/v1/admin/users/${id}/role`, { role })
```

Define explicit DTOs for dashboard, articles, both comment kinds, Skill/MCP review detail, reports, announcements and logs. Do not type API payloads as `any`; only `AdminLogItem.detail` is `unknown`.

- [ ] **Step 2: Implement idempotent session restoration before authorization**

```ts
let restorePromise: Promise<void> | null = null
async function restoreSession() {
  if (!accessToken.value || user.value) return
  restorePromise ??= fetchMe()
    .then(({ data }) => { user.value = data.data })
    .catch(() => clearSession())
    .finally(() => { restorePromise = null })
  return restorePromise
}

router.beforeEach(async (to) => {
  await auth.restoreSession()
  if (to.meta.requiresAdmin && auth.user?.role !== 'admin') return '/'
})
```

Preserve the intended destination across login. Do not trust a role stored separately from `/users/me`.

- [ ] **Step 3: Protect the existing admin route without creating placeholder pages**

```ts
{
  path: '/admin', component: AdminLayout, meta: { requiresAdmin: true },
  children: [
    { path: '', redirect: '/admin/tags' },
    { path: 'tags', component: () => import('@/views/admin/TagManagement.vue') },
  ],
}
```

The complete route table and menu are registered in Task 10 after every target page exists.

- [ ] **Step 4: Run frontend static verification**

```powershell
Set-Location frontend
npm run typecheck
npm run build
```

Expected: PASS using only the existing Tag page; no empty or temporary view is created.

- [ ] **Step 5: Commit frontend foundation**

```powershell
git add frontend/src/api frontend/src/types/index.ts frontend/src/stores/auth.ts frontend/src/router/index.ts
git commit -m "feat(admin-ui): add typed API and session guard"
```

---

### Task 7: Dashboard, users and article pages

**Files:**
- Create: `frontend/src/views/admin/DashboardView.vue`
- Create: `frontend/src/views/admin/UsersView.vue`
- Create: `frontend/src/views/admin/ArticlesView.vue`

**Interfaces:**
- Produces: complete dashboard cards, role management and published article moderation workflows.
- Consumes: Task 6 admin API functions and DTOs.

- [ ] **Step 1: Implement dashboard with explicit loading/error states**

```ts
const data = ref<AdminDashboard | null>(null)
const loading = ref(false)
const error = ref('')
async function loadDashboard() {
  loading.value = true
  error.value = ''
  try { data.value = (await getAdminDashboard()).data.data }
  catch (e) { error.value = apiErrorMessage(e) }
  finally { loading.value = false }
}
```

Render user/article/comment/Skill/MCP/pending-review/pending-report totals. Never display failed fields as zero.

- [ ] **Step 2: Implement user search, paging and guarded role changes**

```ts
async function changeRole(row: AdminUser, role: 'user' | 'admin') {
  if (!row.role_mutable || role === row.role) return
  await ElMessageBox.confirm(`确认将 ${row.nickname} 的角色改为 ${role}？`, '修改角色')
  submitting.value.add(row.id)
  try { await updateAdminUserRole(row.id, role); await loadUsers() }
  finally { submitting.value.delete(row.id) }
}
```

Use a 300 ms keyword debounce, reset to page 1 on filters and disable the row while submitting.

- [ ] **Step 3: Implement published article detail/hide/restore flow**

```ts
async function toggleHidden(row: AdminArticle) {
  const action = row.hidden ? unhideAdminArticle : hideAdminArticle
  await ElMessageBox.confirm(row.hidden ? '确认恢复这篇文章？' : '确认隐藏这篇文章？', '内容管理')
  await action(row.id)
  await loadArticles()
}
```

Article filters are keyword, visibility and author ID. Open detail in a drawer fetched on demand; render Markdown/text safely and do not issue public detail requests.

- [ ] **Step 4: Typecheck/build and commit**

```powershell
Set-Location frontend
npm run typecheck
npm run build
Set-Location ..
git add frontend/src/views/admin/DashboardView.vue frontend/src/views/admin/UsersView.vue frontend/src/views/admin/ArticlesView.vue
git commit -m "feat(admin-ui): add dashboard users and articles"
```

Expected: both commands exit 0.

---

### Task 8: Comment and resource review pages

**Files:**
- Create: `frontend/src/views/admin/CommentsView.vue`
- Create: `frontend/src/views/admin/ResourcesView.vue`
- Create: `frontend/src/components/admin/ResourceReviewDrawer.vue`

**Interfaces:**
- Produces: two independently paged comment tabs and a shared Skill/MCP review drawer.
- Consumes: Task 6 comment/resource API methods and DTOs.

- [ ] **Step 1: Implement independent comment tab state**

```ts
const articleState = reactive({ page: 1, pageSize: 20, total: 0, list: [] as AdminComment[] })
const resourceState = reactive({ page: 1, pageSize: 20, total: 0, list: [] as AdminResourceComment[] })

watch(activeTab, (tab) => {
  if (tab === 'article') loadArticleComments()
  else loadResourceComments()
})
```

Each tab owns keyword, visibility, page and loading. Resource tab additionally owns `resource_type`. Hide/restore confirmations reload only the active tab.

- [ ] **Step 2: Implement the genuinely shared resource review drawer**

```ts
type ReviewResource = AdminSkillDetail | AdminMCPServerDetail
const props = defineProps<{ modelValue: boolean; kind: 'skill' | 'mcp_server'; resource: ReviewResource | null }>()
const emit = defineEmits<{ reviewed: []; 'update:modelValue': [boolean] }>()
```

Display SKILL.md for Skills; pretty-print parsed Tools JSON and show README for MCP Server. Approve needs confirmation; reject requires a reason whose Unicode length is 1-500 and disables repeat submission.

- [ ] **Step 3: Implement review queues and moderation actions**

```ts
async function submitReview(approved: boolean) {
  const reason = rejectReason.value.trim()
  if (!approved && [...reason].length === 0) return ElMessage.error('请填写拒绝原因')
  if ([...reason].length > 500) return ElMessage.error('拒绝原因不能超过 500 个字符')
  await reviewResource(kind.value, selected.value!.id, { approved, reason })
  emit('reviewed')
}
```

Use separate Skill/MCP tabs and fetch detail only when opening the drawer. Draft must not appear as a selectable status.

- [ ] **Step 4: Typecheck/build and commit**

```powershell
Set-Location frontend
npm run typecheck
npm run build
Set-Location ..
git add frontend/src/views/admin/CommentsView.vue frontend/src/views/admin/ResourcesView.vue frontend/src/components/admin/ResourceReviewDrawer.vue
git commit -m "feat(admin-ui): add comments and resource review"
```

Expected: PASS.

---

### Task 9: Tags and report processing pages

**Files:**
- Modify: `frontend/src/views/admin/TagManagement.vue`
- Create: `frontend/src/views/admin/ReportsView.vue`

**Interfaces:**
- Produces: authenticated tag CRUD UX and on-demand report target resolution.
- Consumes: existing tag UI patterns and Task 6 report API.

- [ ] **Step 1: Align tag page with protected API and actual-change behavior**

```ts
async function saveTag() {
  const payload = { name: form.name.trim(), description: form.description.trim(), enabled: form.enabled }
  if (editing.value && isSameTag(payload, original.value)) return closeEditor()
  await (editing.value ? updateTag(editing.value.id, payload) : createTag(payload))
  await loadTags()
}
```

Keep existing page structure; add no generic form wrapper.

- [ ] **Step 2: Implement report list and detail-on-open drawer**

```ts
async function openReport(row: AdminReport) {
  selected.value = null
  drawerOpen.value = true
  selected.value = (await getAdminReport(row.id)).data.data
}

async function resolve(action: 'hide' | 'unhide' | 'dismiss') {
  await resolveAdminReport(selected.value!.id, { action, result: result.value.trim() })
  drawerOpen.value = false
  await loadReports()
}
```

List filters by pending/resolved/dismissed status. Drawer shows reporter, reason/description, resolved target, parent link and moderation result. Do not resolve every list target during list loading.

- [ ] **Step 3: Typecheck/build and commit**

```powershell
Set-Location frontend
npm run typecheck
npm run build
Set-Location ..
git add frontend/src/views/admin/TagManagement.vue frontend/src/views/admin/ReportsView.vue
git commit -m "feat(admin-ui): add tag and report workflows"
```

Expected: PASS.

---

### Task 10: Announcements and audit-log pages

**Files:**
- Create: `frontend/src/views/admin/AnnouncementsView.vue`
- Create: `frontend/src/views/admin/LogsView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/AdminLayout.vue`

**Interfaces:**
- Produces: announcement publishing/history and readable audit log inspection.
- Consumes: Task 6 announcement/log API and `detail: unknown` typing.

- [ ] **Step 1: Implement announcement history and publish confirmation**

```ts
async function publish() {
  await ElMessageBox.confirm('发布后会向全部用户发送站内通知，确认继续？', '发布公告')
  submitting.value = true
  try {
    await createAdminAnnouncement({ title: form.title.trim(), content: form.content.trim() })
    Object.assign(form, { title: '', content: '' })
    await loadAnnouncements()
  } finally { submitting.value = false }
}
```

Validate title/content before confirmation and prevent duplicate submission.

- [ ] **Step 2: Implement audit-log filters and safe detail display**

```ts
function formatDetail(detail: unknown): string {
  if (typeof detail === 'string') return detail
  try { return JSON.stringify(detail, null, 2) }
  catch { return String(detail) }
}
```

Filter by action and paginate. Show administrator public info, action, target, timestamp and formatted detail; never render detail as raw HTML.

- [ ] **Step 3: Typecheck/build and commit**

Register Dashboard, Users, Articles, Comments, Resources, Tags, Reports, Announcements and Logs under `/admin`; change the index redirect to `/admin/dashboard`; add the matching Element Plus menu entries and breadcrumb labels only after all page modules exist.

```ts
const adminChildren: RouteRecordRaw[] = [
  { path: '', redirect: '/admin/dashboard' },
  { path: 'dashboard', name: 'admin-dashboard', component: () => import('@/views/admin/DashboardView.vue') },
  { path: 'users', name: 'admin-users', component: () => import('@/views/admin/UsersView.vue') },
  { path: 'articles', name: 'admin-articles', component: () => import('@/views/admin/ArticlesView.vue') },
  { path: 'comments', name: 'admin-comments', component: () => import('@/views/admin/CommentsView.vue') },
  { path: 'resources', name: 'admin-resources', component: () => import('@/views/admin/ResourcesView.vue') },
  { path: 'tags', name: 'admin-tags', component: () => import('@/views/admin/TagManagement.vue') },
  { path: 'reports', name: 'admin-reports', component: () => import('@/views/admin/ReportsView.vue') },
  { path: 'announcements', name: 'admin-announcements', component: () => import('@/views/admin/AnnouncementsView.vue') },
  { path: 'logs', name: 'admin-logs', component: () => import('@/views/admin/LogsView.vue') },
]
```

```powershell
Set-Location frontend
npm run typecheck
npm run build
Set-Location ..
git add frontend/src/views/admin/AnnouncementsView.vue frontend/src/views/admin/LogsView.vue frontend/src/router/index.ts frontend/src/components/AdminLayout.vue
git commit -m "feat(admin-ui): add announcements and audit logs"
```

Expected: PASS.

---

### Task 11: Full admin verification and phase documentation

**Files:**
- Modify: `docs/roadmap.md`
- Create: `docs/phase6-summary.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- Produces: tested P6 admin deliverable and accurate project documentation.
- Consumes: all prior admin tasks and MCP plan completion status.

- [ ] **Step 1: Run all backend tests and static checks**

```powershell
docker compose up -d
$env:GOCACHE="$PWD/.gocache"
go test ./...
go vet ./...
go build ./...
```

Expected: every command exits 0. Record any unavailable external dependency honestly rather than treating skipped execution as success.

- [ ] **Step 2: Run frontend typecheck and production build**

```powershell
Set-Location frontend
npm run typecheck
npm run build
```

Expected: both commands exit 0 with no TypeScript errors.

- [ ] **Step 3: Perform a focused manual smoke test**

```text
1. Refresh /admin/dashboard with a persisted admin access token: session restores and page remains accessible.
2. Refresh the same URL as a normal user: redirect to / without an admin API call.
3. Search users and modify a mutable role; self/configured admin role controls are disabled.
4. Hide and restore one published article; drafts never appear.
5. Verify article/resource comment tabs keep independent page state.
6. Approve one resource and reject one with a reason; draft is absent.
7. Open one report and verify only then its target loads; resolve it.
8. Create/edit/disable a tag, publish an announcement and inspect matching audit logs.
```

- [ ] **Step 4: Document only verified behavior**

```markdown
## P6 verification

- Backend: `go test ./...`, `go vet ./...`, `go build ./...`
- Frontend: `npm run typecheck`, `npm run build`
- Admin scope: roles only; no user status machine
- MCP scope: nine read-only tools
```

Update roadmap only after both P6 plans pass their verification tasks.

- [ ] **Step 5: Commit phase documentation**

```powershell
git add docs/roadmap.md docs/phase6-summary.md CLAUDE.md
git commit -m "docs: complete P6 MCP and admin phase"
```
