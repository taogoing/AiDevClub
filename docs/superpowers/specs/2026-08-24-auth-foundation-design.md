# AIDevClub P0+P1 设计：基础设施骨架 + 用户与认证

日期：2026-08-24
状态：待审查

## 1. 背景与目标

AIDevClub 是一个面向开发者和 AI Agent 的技术内容与 AI 资源分享社区（技术文章 + Skills Hub + MCP Hub）。本阶段（P0 基础设施骨架 + P1 用户与认证）的目标是：

1. 建立一个可运行的后端服务骨架，把技术栈（Go/Gin/GORM/MySQL/Redis）真正跑起来。
2. 实现用户注册、登录、登出、Token 刷新、资料管理、注销账号。

本阶段是后续文章、Skill/MCP 资源、平台 MCP Server 等所有阶段的地基——后续每个阶段都以一致的方式扩展本项目。

## 2. 范围

### 2.1 包含

- 可运行的 Gin 服务骨架 + Docker Compose（MySQL 8、Redis、应用）。
- 健康检查、统一配置、日志、错误处理、统一响应格式。
- 用户注册 / 登录 / 登出 / 刷新 Token / 查改资料 / 改密码 / 注销账号。
- 头像：注册时分配默认头像，支持上传修改。
- 注册与登录接口限流。

### 2.2 不包含（非目标）

- 用户状态机（禁言 / 禁止发布 / 封禁等）——已明确移除，管理员不封禁或禁言用户。
- 邮箱验证——注册不发送验证邮件，邮箱仅作唯一登录标识。
- 文章、评论、Skill、MCP Server 等业务——属后续阶段。
- 头像以外的文件上传（正文图片、ZIP 资源包）——属后续文件管理阶段。

## 3. 关键决策

| 决策项 | 结论 |
|---|---|
| 代码结构 | 扁平技术分层（`internal/handler` / `service` / `repo` / `model`），领域边界保留在 service 层内部 |
| Token 策略 | JWT Access Token（HS256，约 15 分钟）+ Redis 存储的不透明 Refresh Token（30 天，可吊销、可轮换） |
| 邮箱验证 | 暂不验证 |
| 用户状态 | 无状态机；仅有登录、登出、注销账号（软删除） |
| 昵称 | 注册时可不填，自动生成默认值，后续可修改 |
| 头像 | 注册给默认头像；P1 即支持上传修改（本地磁盘存储） |

## 4. 数据模型

### 4.1 users 表（GORM 模型）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint，自增主键 | |
| email | varchar(191)，唯一索引，非空 | 登录标识 |
| password_hash | varchar，非空 | bcrypt 哈希 |
| nickname | varchar，非空 | 注册时不填则自动生成 `用户` + 6 位随机字符 |
| avatar_url | varchar，非空 | 注册时填默认头像 URL |
| bio | text，可空 | 个人简介 |
| created_at / updated_at | timestamp | |
| deleted_at | timestamp，可空 | 软删除，用于「注销账号」 |

### 4.2 Refresh Token（Redis）

- Key：`refresh:{token}` → 值 `user_id`
- TTL：30 天
- 登出、改密码、注销账号时删除对应 key，实现即时吊销。

## 5. API 接口（前缀 `/api/v1`）

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| GET | /healthz | 无 | 健康检查 |
| POST | /auth/register | 无 | 注册（email、password，nickname 可选） |
| POST | /auth/login | 无 | 登录 → access + refresh |
| POST | /auth/refresh | 无 | 刷新 → 新 access（refresh 轮换） |
| POST | /auth/logout | Bearer | 登出，吊销 refresh |
| GET | /users/me | Bearer | 查当前用户资料 |
| PATCH | /users/me | Bearer | 改昵称 / 简介 / 头像 URL |
| PUT | /users/me/password | Bearer | 改密码，吊销所有 refresh |
| POST | /users/me/avatar | Bearer | 头像上传（multipart） |
| DELETE | /users/me | Bearer | 注销账号（软删除 + 吊销） |

## 6. 鉴权流程

- **Access Token**：JWT（HS256），过期约 15 分钟，claims 含 `user_id`、`exp`、`iat`。
- **Refresh Token**：32 字节随机不透明串（base64url），存 Redis。
- 注册 / 登录返回 `access_token` + `refresh_token`。
- **刷新**：校验 refresh 在 Redis 中存在 → 签发新 access；同时轮换 refresh（旧 refresh 作废，签发新 refresh）。
- **登出 / 改密码 / 注销**：删除 Redis 中的 refresh，使旧 refresh 立即失效。
- **中间件**：解析 `Authorization: Bearer <access>`，验签后把 `user_id` 注入请求 context；验签失败返回 401。

## 7. 头像上传

- 存储：本地磁盘 `storage/avatars/`，静态服务由 Nginx 提供（开发期由 Gin 静态路由提供）。
- 限制：文件类型 `jpg / png / webp / gif`，大小 ≤ 2MB。
- 文件名：`{user_id}_{随机串}.{ext}`，避免冲突。
- 上传成功后返回可访问 URL 并写入 `users.avatar_url`。

## 8. 限流

- 用 Redis 固定窗口（INCR + EXPIRE）实现。
- 仅作用于注册、登录接口。
- 默认每 IP 每分钟 10 次，可配置。

## 9. 技术选型

- 配置：环境变量 + 可选 `.env`（`github.com/spf13/viper`）
- 日志：标准库 `log/slog`
- 数据库迁移：GORM AutoMigrate（开发期够用，后续可换显式 SQL 迁移）
- JWT：`github.com/golang-jwt/jwt/v5`
- 密码哈希：`golang.org/x/crypto/bcrypt`
- Redis：`github.com/redis/go-redis/v9`
- ORM：`gorm.io/gorm` + `gorm.io/driver/mysql`
- HTTP 框架：`github.com/gin-gonic/gin`

## 10. 错误处理与响应格式

- 统一响应：`{"code": 0, "message": "ok", "data": ...}`，非零 `code` 表示业务错误。
- 全局错误处理中间件把内部错误映射为 HTTP 状态码 + 业务错误码。
- 业务错误码示例：参数错误、未认证（401）、邮箱已存在（冲突）、限流、资源不存在等。

## 11. 测试策略

所有测试使用真实 MySQL 与 Redis（由 Docker Compose 提供，需已启动）：

- **service 层单元测试**：密码哈希、token、注册/登录/刷新/登出逻辑，直接用真实仓库（UserRepo/TokenRepo）打到测试库。
- **handler 层集成测试**：`httptest` + 真实 MySQL/Redis。
- 测试隔离：MySQL 用独立 `aidevclub_test` 库（`CREATE DATABASE IF NOT EXISTS` 自动创建，测试前后 drop `users` 表），Redis 用 DB 15（测试前后 `FlushDB`）。

## 12. 项目结构

```text
cmd/server/main.go          # 装配入口：config → db/redis → service → handler → 路由
internal/handler/           # auth.go、user.go、health.go
internal/service/           # auth.go、user.go（领域服务，后续 MCP Server 复用）
internal/repo/              # user.go、token.go
internal/model/             # user.go
internal/platform/          # config.go、database.go、redis.go、logger.go、middleware.go、response.go
```

说明：采用扁平技术分层（方案 B），领域边界保留在 service 层内部（按 `auth` / `user` 分文件），后续文章、资源等阶段在 service/repo/handler 各层内新增对应文件，避免按领域分包导致的后期重构。
