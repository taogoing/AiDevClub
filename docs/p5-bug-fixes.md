# P5 Bug 修复记录

## 1. goroutine 使用 HTTP request context 导致通知静默丢失

**严重程度：** 高

**现象：**
通知在某些情况下不会写入数据库，但没有任何错误日志。

**根因分析：**
所有通知发送的 goroutine 都直接使用了 handler 传入的 `ctx`（即 `c.Request.Context()`）。

```go
// 错误代码
go func() {
    _ = s.notifSvc.Create(ctx, a.AuthorID, ...)  // ctx 是 HTTP request context
}()
```

HTTP handler 返回后，Gin 框架会取消该 request 的 context。goroutine 中的数据库写入操作会因为 `context cancelled` 而失败，但错误被 `_ =` 吞掉，导致通知静默丢失。

**涉及位置：**
- `internal/service/comment.go:57` — 评论通知
- `internal/service/comment.go:202` — 评论点赞通知
- `internal/service/article.go:355-357` — 文章点赞通知
- `internal/service/skill.go:384-388` — Skill 点赞通知
- `internal/service/mcp_server.go:394-398` — MCP Server 点赞通知
- `internal/service/resource_comment.go:101` — 资源评论通知
- `internal/service/resource_comment.go:248-250` — 资源评论点赞通知

**修复方案：**
goroutine 内使用 `context.Background()` 而非继承 request context：

```go
// 修复后
go func() {
    _ = s.notifSvc.Create(context.Background(), a.AuthorID, ...)
}()
```

**面试讲解要点：**
- Go 的 context 生命周期管理
- HTTP handler 返回后 context 会被取消
- goroutine 中不应使用短生命周期的 context
- 异步操作应使用独立的 context

---

## 2. ReportService.Resolve 未使用事务

**严重程度：** 高

**现象：**
如果隐藏内容成功但更新举报状态失败，数据会处于不一致状态：内容已隐藏但举报仍为 pending。

**根因分析：**
设计文档 §2 明确要求：「举报处理和内容操作（隐藏/恢复）在同一事务中完成」。

但 `ReportService.Resolve` 方法中，`HideContent`/`UnhideContent` 和 `repo.Update(report)` 是两个独立的数据库操作，没有包裹在事务中。

```go
// 错误代码
if err := s.adminSvc.HideContent(adminTargetType, report.TargetID); err != nil {
    return err
}
// ... 如果这里失败，内容已隐藏但举报状态未更新
if err := s.repo.Update(report); err != nil {
    return err
}
```

**修复方案：**
1. 在 `AdminService` 中添加接受 `*gorm.DB` 参数的 `HideContentTx`/`UnhideContentTx` 方法
2. 在 `ReportService.Resolve` 中使用事务包裹：

```go
err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
    switch action {
    case "hide":
        if err := s.adminSvc.HideContentTx(tx, adminTargetType, report.TargetID); err != nil {
            return err
        }
    case "unhide":
        if err := s.adminSvc.UnhideContentTx(tx, adminTargetType, report.TargetID); err != nil {
            return err
        }
    }
    return tx.Save(report).Error
})
```

**面试讲解要点：**
- 数据库事务的 ACID 特性
- 多步操作需要原子性保证
- 事务传递模式（传入 `*gorm.DB` 而非使用默认连接）
- 部分成功导致的数据不一致问题

---

## 3. ReviewSkill/ReviewMcpServer 用 UpdatedAt 作为 PublishedAt

**严重程度：** 中

**现象：**
资源审核通过后，`PublishedAt` 显示的是最后一次更新（提交审核）的时间，而非审核通过的时间。

**根因分析：**
```go
// 错误代码
now := sk.UpdatedAt  // UpdatedAt 是资源最后一次更新的时间
sk.Status = newStatus
sk.PublishedAt = &now
```

`UpdatedAt` 是资源最后一次更新（提交审核）的时间，不是审核通过的时间。`PublishedAt` 语义上应该是「发布时间」，即审核通过的时刻。

**修复方案：**
使用 `time.Now()` 获取当前时间：

```go
// 修复后
now := time.Now()
sk.Status = newStatus
sk.PublishedAt = &now
```

**面试讲解要点：**
- 时间字段的语义正确性
- `UpdatedAt` vs `CreatedAt` vs 业务时间字段的区别
- 审核流程中的时间戳设计

---

## 4. 测试超时问题（测试基础设施问题）

**严重程度：** 中（不影响生产代码）

**现象：**
运行 `go test ./internal/handler/` 全量测试时，部分测试超时（60s+）。

**根因分析：**
1. **并行测试共享数据库：** Go test 默认并行运行测试函数。每个测试函数调用 `testutil.NewTestDB(t)`，该方法会：
   - 删除所有表
   - AutoMigrate 所有模型（22 个表）
   - 创建 FULLTEXT 索引

2. **竞态条件：** 多个测试并行运行时：
   - Test A 正在删除表
   - Test B 正在 AutoMigrate
   - 导致 `Table doesn't exist` 错误

3. **FULLTEXT 索引创建慢：** 每次测试都要创建 3 个 FULLTEXT 索引，并行时 MySQL 连接竞争严重。

**验证：**
```bash
# 串行运行测试，全部通过
go test ./internal/handler/ -run "TestNotif" -count=1 -v -timeout 60s -p 1
# PASS (32.967s)

# 并行运行测试，超时
go test ./internal/handler/ -count=1 -timeout 60s
# FAIL (timeout)
```

**影响范围：**
- 这是测试基础设施的问题，不是 P5 代码引入的 bug
- P5 新增了 4 个模型，使 AutoMigrate 更慢，加剧了问题

**建议修复方案（未实施）：**
1. 使用 `sync.Once` 确保数据库初始化只执行一次
2. 每个测试函数使用独立的事务，测试结束后回滚
3. 或者使用 `t.Parallel()` 显式控制并行度

**面试讲解要点：**
- 测试隔离性的重要性
- 并行测试的资源竞争问题
- 测试基础设施对开发效率的影响
- `sync.Once` 的使用场景

---

## 5. FULLTEXT 索引在测试数据库中缺失

**严重程度：** 中

**现象：**
运行 `TestArticleListAndGet` 测试时失败：
```
Error 1191 (HY000): Can't find FULLTEXT index matching the column list
```

**根因分析：**
生产环境在 `main.go` 中调用 `platform.CreateFulltextIndexes(db)` 创建 FULLTEXT 索引，但测试环境的 `testutil.NewTestDB` 没有创建这些索引。

当测试代码执行带 `MATCH ... AGAINST` 的查询时，MySQL 报错找不到 FULLTEXT 索引。

**修复方案：**
在 `testutil.NewTestDB` 中添加 FULLTEXT 索引创建：

```go
// 创建 FULLTEXT 索引（忽略已存在的错误）
_ = db.Exec(`CREATE FULLTEXT INDEX idx_ft_article_search ON articles(title, summary, content) WITH PARSER ngram`).Error
_ = db.Exec(`CREATE FULLTEXT INDEX idx_ft_skill_search ON skills(name, description) WITH PARSER ngram`).Error
_ = db.Exec(`CREATE FULLTEXT INDEX idx_ft_mcp_search ON mcp_servers(name, description) WITH PARSER ngram`).Error
```

**提交：** `b299229`

**面试讲解要点：**
- 测试环境与生产环境的一致性
- FULLTEXT 索引的创建方式
- 测试基础设施的完整性检查

---

## 总结

| 问题 | 类型 | 影响 | 状态 |
|------|------|------|------|
| goroutine context | 代码 bug | 通知丢失 | ✅ 已修复 |
| 事务缺失 | 代码 bug | 数据不一致 | ✅ 已修复 |
| PublishedAt 时间 | 代码 bug | 时间错误 | ✅ 已修复 |
| 测试超时 | 测试基础设施 | 开发体验 | ⏸️ 未修复（非阻塞） |
| FULLTEXT 索引缺失 | 测试基础设施 | 测试失败 | ✅ 已修复 |

**面试推荐讲解顺序：**
1. goroutine context（展示对 Go 并发和 context 的理解）
2. 事务缺失（展示对数据库一致性的理解）
3. 测试超时（展示对测试基础设施和并行问题的理解）
