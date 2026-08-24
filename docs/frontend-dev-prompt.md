# 前端开发续跑提示词（Vue 3 用户端）

> 将此提示词完整粘贴给新会话的 AI 助手，即可开始前端开发。

---

## 项目背景

AIDevClub 是一个面向开发者和 AI Agent 的技术内容与 AI 资源分享社区。后端 P0+P1+P2 已完成（基础设施 + 用户认证 + 技术社区），现在需要开发 Vue 3 用户端前端，对接已有 REST API。

工作目录：`E:\goProject\library\src\GoStudy\AiDevClub`（注意大小写：AiDevClub，不要用 AIDevClub 大写路径，会触发 git 隔离报错）

后端服务运行在 `:8080`，启动命令 `go run ./cmd/server`。数据库 Docker 已配好（MySQL 3306 + Redis 16379）。

## 技术栈

- **框架**：Vue 3（Composition API + `<script setup>`）
- **语言**：TypeScript
- **构建**：Vite
- **UI 库**：自选（推荐 Element Plus 或 Naive UI，也可纯 Tailwind CSS）
- **路由**：Vue Router 4
- **状态**：Pinia
- **HTTP**：Axios（封装拦截器处理 Token 刷新）
- **Markdown**：markdown-it + highlight.js（文章正文渲染）
- **编辑器**：可选用 md-editor-v3 或类似 Markdown 编辑器组件

## 后端 API 完整契约

### 统一响应格式

```json
{ "code": 0, "message": "ok", "data": {...} }
```

- `code=0` 表示成功
- 错误时 `code` 非 0，HTTP status 对应业务错误码
- 认证失败返回 401（code=40101），需要尝试刷新 Token

### 认证机制

- **Access Token**：JWT，15 分钟有效，放在 `Authorization: Bearer <token>` 请求头
- **Refresh Token**：不透明 token，30 天有效，存 localStorage
- **刷新流程**：Access 过期 → 401 → 用 Refresh 调 `/api/v1/auth/refresh` → 获取新 Token 对 → 重发原请求
- **登出**：调 `/api/v1/auth/logout`（body: `{"refresh_token": "..."}`），清 localStorage

### 端点列表

#### 认证

| 方法 | 路径 | 请求体 | 响应 data |
|---|---|---|---|
| POST | /api/v1/auth/register | `{"email":"","password":"","nickname":""}` | - |
| POST | /api/v1/auth/login | `{"email":"","password":""}` | `{"access_token":"","refresh_token":""}` |
| POST | /api/v1/auth/refresh | `{"refresh_token":""}` | `{"access_token":"","refresh_token":""}` |
| POST | /api/v1/auth/logout | `{"refresh_token":""}` | - |

#### 用户

| 方法 | 路径 | 认证 | 响应 data |
|---|---|---|---|
| GET | /api/v1/users/me | Auth | `{"id":0,"email":"","nickname":"","avatar_url":"","bio":""}` |
| PATCH | /api/v1/users/me | Auth | body: `{"nickname":"","avatar_url":"","bio":""}` |
| PUT | /api/v1/users/me/password | Auth | body: `{"password":""}` |
| DELETE | /api/v1/users/me | Auth | 注销账号 |
| POST | /api/v1/users/me/avatar | Auth | multipart/form-data, field="file" → `{"avatar_url":""}` |

#### 分类

| 方法 | 路径 | 响应 data |
|---|---|---|
| GET | /api/v1/categories | `[{"id":0,"name":"","slug":""}]` |

预置分类：Go / 后端 / 前端 / AI-LLM / DevOps / 数据库 / 移动端 / 安全 / 其他

#### 标签

| 方法 | 路径 | 查询参数 | 响应 data |
|---|---|---|---|
| GET | /api/v1/tags | `?keyword=`（前缀搜索）或 `?hot=1`（热门排行） | `[{"id":0,"name":"","usage_count":0}]` |

#### 文章

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | /api/v1/articles | - | 列表，query: `page, page_size, category_id, tag_id, keyword, author_id, sort(latest/hot/pinned)` |
| GET | /api/v1/articles/:id | Optional | 详情（带 token 时返回 liked/favorited） |
| POST | /api/v1/articles | Auth | 发布，body: `{"title":"","summary":"","content":"","category_id":0,"status":"draft|published","tag_ids":[],"tag_names":[]}` |
| PUT | /api/v1/articles/:id | Auth | 编辑，body 同上 |
| DELETE | /api/v1/articles/:id | Auth | 删除（软删除） |
| POST | /api/v1/articles/images | Auth | 上传正文图片，multipart/form-data, field="file" → `{"url":"/static/articles/xxx.png"}` |
| POST | /api/v1/articles/:id/like | Auth | 点赞/取消，返回 `{"liked":bool,"likes_count":0}` |
| POST | /api/v1/articles/:id/favorite | Auth | 收藏/取消，返回 `{"favorited":bool,"favorites_count":0}` |

**文章列表响应**：
```json
{
  "list": [{
    "id": 0, "title": "", "summary": "", "category_id": 0, "category_name": "",
    "tags": [{"id":0,"name":""}],
    "author": {"id":0,"nickname":"","avatar_url":""},
    "views": 0, "likes_count": 0, "favorites_count": 0, "comments_count": 0,
    "published_at": "2026-08-24T12:00:00Z", "pinned": false
  }],
  "total": 0, "page": 1, "page_size": 20
}
```

**文章详情响应**：列表项字段 + `content`（Markdown 原文）+ `liked` + `favorited`

#### 评论

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | /api/v1/articles/:id/comments | - | 评论列表（两级结构） |
| POST | /api/v1/articles/:id/comments | Auth | body: `{"content":"","parent_id":null}` |
| DELETE | /api/v1/comments/:id | Auth | 删除（评论作者或文章作者可删） |
| POST | /api/v1/comments/:id/like | Auth | 评论点赞/取消 |

**评论列表响应**：
```json
[{
  "id": 0, "article_id": 0, "author_id": 0,
  "author": {"id":0,"nickname":"","avatar_url":""},
  "content": "", "likes_count": 0, "created_at": "2026-08-24T12:00:00Z",
  "replies": [{ /* 同结构，嵌套 */ }]
}]
```

#### 静态文件

- 头像：`/static/avatars/xxx.png`
- 文章图片：`/static/articles/xxx.png`

## 需要实现的页面

### 布局

- **顶部导航栏**：Logo、分类下拉、搜索框、登录/注册按钮（或用户头像下拉菜单：个人中心、发布文章、登出）
- **底部**：版权信息

### 页面清单

1. **首页 / 文章列表**（`/`）
   - 文章卡片列表（标题、摘要、作者、分类、标签、统计数据）
   - 分类筛选 tab、排序切换（最新/热门/置顶）
   - 搜索框
   - 分页

2. **文章详情**（`/articles/:id`）
   - Markdown 正文渲染
   - 作者信息、分类、标签
   - 浏览/点赞/收藏/评论统计
   - 点赞/收藏按钮（toggle）
   - 评论区（两级评论列表 + 发表评论 + 回复 + 评论点赞 + 删除）

3. **发布/编辑文章**（`/articles/new`、`/articles/:id/edit`）
   - Markdown 编辑器（支持图片上传，调用 `/api/v1/articles/images`）
   - 标题、摘要、分类选择、标签选择/新建
   - 保存草稿 / 发布按钮

4. **登录**（`/login`）
   - 邮箱 + 密码
   - 注册链接

5. **注册**（`/register`）
   - 邮箱 + 密码 + 昵称（可选）

6. **个人中心**（`/users/me`）
   - 资料（昵称、头像、简介）修改
   - 头像上传
   - 修改密码
   - 注销账号

## 关键实现要点

### Axios 拦截器（Token 刷新）

```typescript
// 请求拦截器：自动加 Authorization 头
// 响应拦截器：401 时用 refresh_token 刷新，重发原请求
// 刷新失败 → 跳转登录页
```

- Access Token 存内存（Pinia state）
- Refresh Token 存 localStorage
- 刷新期间如果有多个请求 401，需要排队等刷新完成

### 前端目录结构建议

```text
frontend/
├── index.html
├── vite.config.ts          # proxy /api → localhost:8080, /static → localhost:8080
├── tsconfig.json
├── package.json
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── router/
│   │   └── index.ts
│   ├── stores/             # Pinia (auth, article, comment)
│   ├── api/                # Axios 实例 + 各模块 API
│   │   ├── http.ts         # 拦截器 + Token 刷新
│   │   ├── auth.ts
│   │   ├── article.ts
│   │   ├── comment.ts
│   │   └── ...
│   ├── components/         # 公共组件
│   │   ├── ArticleCard.vue
│   │   ├── CommentTree.vue
│   │   ├── MarkdownEditor.vue
│   │   ├── TagSelector.vue
│   │   └── ...
│   ├── views/              # 页面
│   │   ├── HomeView.vue
│   │   ├── ArticleDetailView.vue
│   │   ├── ArticleEditView.vue
│   │   ├── LoginView.vue
│   │   ├── RegisterView.vue
│   │   └── ProfileView.vue
│   ├── composables/        # useAuth, useArticle 等
│   ├── types/              # TypeScript 类型定义
│   └── assets/
```

### Vite 代理配置

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
    '/static': 'http://localhost:8080',
  }
}
```

### 开发工作流

项目使用 superpowers-zh 技能框架。收到功能需求时先用 brainstorming skill 做需求分析，然后 writing-plans 写计划，再实现。

设计文档存 `docs/superpowers/specs/`，实现计划存 `docs/superpowers/plans/`。

标准命令：
```bash
cd frontend
npm install
npm run dev          # 开发服务器
npm run build        # 生产构建
npm run typecheck    # 类型检查
npm run lint         # ESLint
```

### 后端启动（前端开发时需要）

```bash
cd E:\goProject\library\src\GoStudy\AiDevClub
docker compose up -d  # MySQL + Redis
go run ./cmd/server   # 后端 :8080
```

## 现有代码

- 后端代码在 `internal/` 下，前端代码将在 `frontend/` 下新建
- 后端路由和响应格式见 `cmd/server/main.go` 和 `internal/platform/response.go`
- 后端 DTO 定义见 `internal/service/dto.go`
- 后端 Handler 见 `internal/handler/*.go`
- 阶段总结见 `docs/phase1-summary.md` 和 `docs/phase2-summary.md`

## 遗留跟进项

- 后端 P2 遗留 3 个 Important + 多个 Minor（见 `docs/phase2-summary.md`），不阻塞前端开发
- 后端 P1 遗留 17 个 Minor（见 `docs/phase1-summary.md`），不阻塞
- 文章列表的 `status` 省略时后端返回 400（应默认 draft），前端需默认传 `"draft"` 或 `"published"`
