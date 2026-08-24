# AIDevClub 阶段一总结（P0+P1：基础设施 + 用户认证）

日期：2026-08-24

## 完成内容

后端从零搭建了可运行的服务骨架，并实现了用户认证。已合并到 `master` 并推送到 https://github.com/taogoing/AiDevClub。

- **基础设施**：Go/Gin/GORM 服务、MySQL 8 + Redis 7（Docker Compose）、统一配置/日志/健康检查、统一响应与错误中间件。
- **数据模型**：User（软删除）。
- **认证**：注册 / 登录 / 登出 / Token 刷新（JWT Access + Redis 不透明 Refresh，轮换 + 吊销）。
- **用户资料**：查 / 改 / 改密 / 注销账号、头像上传（本地磁盘）。
- **限流**：注册 / 登录接口 Redis 固定窗口限流。
- **测试**：全流程测试，用真实 MySQL/Redis，按进程隔离（`aidevclub_test_<pid>` 库 + Redis DB `pid%16`）。

## 技术栈与架构

- 后端 Go/Gin/GORM，MySQL 8 持久化，Redis 做缓存/限流/Token 状态。
- 模块化单体，扁平技术分层：`internal/handler` → `service` → `repo` / `model`，横切能力在 `internal/platform`。
- JWT 纯函数下沉到 `platform`（避免循环依赖）；后续平台 MCP Server 复用 `service` 层。

## 目录结构

```text
cmd/server/main.go          # 装配入口
internal/handler/           # auth.go、user.go、errors.go + 测试
internal/service/           # auth.go、user.go + 测试（领域服务）
internal/repo/              # user.go、token.go + 测试
internal/model/             # user.go
internal/platform/          # config、database、redis、logger、response、errors、middleware、jwt、ratelimit
internal/testutil/          # 测试后端（真实 MySQL/Redis，按进程隔离）
docker-compose.yml          # MySQL 8 + Redis 7（Redis 宿主机端口 16379）
```

## 运行与测试

```bash
docker compose up -d        # 启动 MySQL(3306) + Redis(16379)
go build ./...              # 编译
go test ./...               # 全量测试（依赖上面两个容器）
go run ./cmd/server         # 启动服务，/healthz 健康检查
```

配置通过环境变量（前缀 `AIDEVCLUB_`）覆盖，见 `internal/platform/config.go` 的默认值。

## 关键设计决策

- 用户**无状态机**（禁言/封禁已移除），仅登录 / 登出 / 注销（软删除）。
- 注册不验证邮箱；昵称注册时自动生成（`用户_` + 6 位随机）。
- Token：JWT Access（15 分钟）+ Redis 不透明 Refresh（30 天，轮换/吊销）。
- 头像：注册给默认值，P1 即支持上传（本地磁盘 `storage/avatars/`）。
- Redis 端口用 16379（6379 被 Windows 排除端口范围占用）。

## 遗留跟进项

### 设计/安全决策（需拍板，未阻塞）

1. **改密/注销是否加「当前密码复核」**：目前仅凭 access token 即可改密/注销，偷到 token（15 分钟窗口）即可完全接管。标准做法是加 `old_password` 二次验证。
2. **改密/注销原子性 + Refresh 校验用户存在**：改库与吊销 refresh 非同一事务，极端情况（Redis 故障）已注销账号的 refresh 仍能换新 token。

### Minor 优化项

1. `RecoverMiddleware` 的 `c.Errors` 双写风险（现有 handler 不触发）。
2. `Get` 把一切 `FindByID` 错误折叠为 404（掩盖 DB 故障）。
3. 上传成功后 `UpdateProfile` 失败留下孤儿文件。
4. `main.go` 用 `gin.Recovery()` 而非 `platform.RecoverMiddleware`（panic 响应格式不统一）。
5. 限流用 `context.Background()`；`Expire` 错误被忽略（可能致 key 无 TTL 永久 429）。
6. Refresh 并发竞态（Validate/Revoke 之间）+ 双重 Validate。
7. 图片只校验扩展名、无魔数内容校验。
8. handler 的 500 路径 `err.Error()` 泄漏内部错误。
9. `randomHex` 忽略 `rand.Read` 错误。
10. Bearer 前缀大小写敏感（RFC 7235 应大小写不敏感）。
11. 邮箱未归一化、密码无最小长度校验。
12. `UpdateProfile` 用 `!= ""` 判断，无法清空字段。
13. `avatar_url` 可被设为任意 URL。
14. 测试 `TestUploadAvatar` 硬编码 user_id=1。
15. `.env` 未读取（配置提到但未实现 `ReadInConfig`）。
16. docker-compose 无 app 服务（app 未容器化）。
17. JWT keyfunc 未校验 `alg`。

## 下一步

P2 **技术社区**：文章发布/浏览（分类+标签+搜索）、两级评论、点赞/收藏/浏览量统计。对应需求文档 [doc/AIDevClub需求文档.md](../doc/AIDevClub需求文档.md) 第 5 节。
