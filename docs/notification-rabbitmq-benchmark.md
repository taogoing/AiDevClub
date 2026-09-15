# RabbitMQ + Transactional Outbox 版本 k6 压测

## 范围与方法

本次测量业务写入和 Outbox 事件在同一 MySQL 事务、HTTP 请求返回后再异步发布/消费的版本。除通知模式外，机器、Docker Compose 服务、HTTP 接口、专属 JWT 用户、文章创建方式、20 VU、30 秒和 k6 脚本都与[同步版本](notification-sync-benchmark.md)一致。点赞/收藏为 toggle，每个 VU 反复切换；评论每次都新增一条。每个场景结束后，脚本等待该场景的 Outbox 事件均已发布且对应 `notifications` 行可见，然后才开始下一场景。

本轮开始前没有待投递的上一轮事件；测试前后没有清理数据库卷。本报告采用最终代码（包含 mandatory publish Return 检查）在 `20260913-200150-fd4d4059` 目录中的 Outbox 数据。三场景共有 1,502 个 Outbox 事件发布完成，并生成 1,502 条带 `event_id` 的通知；每个场景的 k6 HTTP 检查全部通过。

## 复现

在 Windows PowerShell 项目根目录运行：

```powershell
docker compose up -d mysql redis rabbitmq
.\scripts\run-notification-benchmark.ps1 -Mode both -VUs 20 -Duration 30s
```

脚本自动构建服务、启动隔离端口服务、为每场景创建测试账号/文章，并分别执行 sync 与 outbox。RabbitMQ UI 是 `http://localhost:15672`；AMQP 为 `localhost:5672`。可单独复跑 Outbox 模式：

```powershell
.\scripts\run-notification-benchmark.ps1 -Mode outbox -VUs 20 -Duration 30s
```

机器、鉴权、接口和测试数据细节见[同步报告的方法部分](notification-sync-benchmark.md#范围与方法)。最终 Outbox 原始记录为[`20260913-200150-fd4d4059`](notification-benchmark-runs/README.md#final-outbox-run-20260913-200150-fd4d4059)：[点赞](notification-benchmark-runs/20260913-200150-fd4d4059/outbox-like-k6-summary.json)、[收藏](notification-benchmark-runs/20260913-200150-fd4d4059/outbox-favorite-k6-summary.json)、[评论](notification-benchmark-runs/20260913-200150-fd4d4059/outbox-comment-k6-summary.json)的原始 k6 JSON、完整输出、应用进程和容器资源采样、运行参数及 [`machine.json`](notification-benchmark-runs/20260913-200150-fd4d4059/machine.json)均可复核。

## HTTP 结果

延迟单位为毫秒；错误率为非 200 比例。

| 接口 | 请求数 | QPS | P50 | P95 | P99 | 错误率 |
|---|---:|---:|---:|---:|---:|---:|
| 点赞文章 | 885 | 28.76 | 745.7 | 1,038.8 | 1,283.1 | 0% |
| 收藏文章 | 895 | 29.15 | 735.8 | 1,026.3 | 1,248.9 | 0% |
| 发表评论 | 599 | 19.30 | 1,028.7 | 1,059.9 | 1,084.2 | 0% |

## 资源使用

容器 CPU 由 `docker stats --no-stream` 采样，内存为场景采样峰值；CPU 单核百分比可能超过 100%。应用 CPU 为 Go 进程 CPU 时间折算为整机平均占用。每项实际记录 8 个带时间戳样本；原始 JSONL 可复核。

| 场景 | 应用 CPU / 整机 | 应用峰值内存 | MySQL CPU 平均/峰值；峰值内存 | RabbitMQ CPU 平均/峰值；峰值内存 | Redis CPU 平均/峰值；峰值内存 |
|---|---:|---:|---:|---:|---:|
| 点赞 | 0.48% | 69.9 MiB | 13.8% / 16.3%；998.4 MiB | 1.6% / 1.9%；171.1 MiB | 0.6% / 0.9%；16.5 MiB |
| 收藏 | 0.48% | 71.0 MiB | 14.3% / 17.0%；998.7 MiB | 10.0% / 70.8%；170.8 MiB | 0.5% / 0.6%；16.5 MiB |
| 评论 | 0.32% | 70.5 MiB | 12.5% / 14.6%；998.7 MiB | 24.2% / 184.9%；260.0 MiB | 0.4% / 0.5%；16.3 MiB |

最终代码的 Outbox 复跑晚于同步压测，使用同一持久化开发数据库，但创建了新的专用账号和文章；之前压测记录没有删除。因此 MySQL 内存和缓存工作集会受累积测试数据影响。接口、每场景用户数、文章数据分布、鉴权、并发、时长和请求脚本保持相同；这些是同机顺序测量，不是隔离数据库的并行 A/B。

## 解读与限制

这次本机测试里 Outbox HTTP 延迟高于同步版本，QPS 也较低。Outbox 把通知表写入移出 HTTP 请求，但业务事务新增了 Outbox INSERT；在单篇文章和 20 个并发用户的场景下，计数更新会争同一行，Publisher 和 Consumer 也使用同一 MySQL。这些是可能的开销和争用来源，本次测试没有将它们逐一隔离，不能据此确定单一因果。异步解耦保证了 RabbitMQ 不可用时互动不会因通知写失败而失败，也可在恢复后补发；它不保证任何机器和数据分布下都能提升单接口吞吐。

此压测是 30 秒、20 VU、每场景单文章的本地观测，不是生产容量测试。k6 指标只统计 HTTP 响应；Outbox 发布及通知最终落库在请求结束后继续执行，脚本等待落库的时间不计入 QPS/延迟。进行正式容量评估时还应覆盖多文章分布、阶梯并发、长时间积压、RabbitMQ 故障注入及恢复吞吐。
