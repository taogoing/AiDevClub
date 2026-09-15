# 站内通知：Transactional Outbox 面试讲解

## 实测性能对比

| 接口 | 同步 QPS / P50 / P95 / P99 / 错误率 | Outbox QPS / P50 / P95 / P99 / 错误率 |
|---|---|---|
| 点赞文章 | 53.36 / 347.6ms / 589.8ms / 687.2ms / 0% | 28.76 / 745.7ms / 1,038.8ms / 1,283.1ms / 0% |
| 收藏文章 | 53.50 / 348.7ms / 575.5ms / 645.5ms / 0% | 29.15 / 735.8ms / 1,026.3ms / 1,248.9ms / 0% |
| 发表评论 | 68.62 / 290.7ms / 320.2ms / 332.8ms / 0% | 19.30 / 1,028.7ms / 1,059.9ms / 1,084.2ms / 0% |

这些数据来自相同本机、20 VU、30 秒的 k6 运行；详细条件和资源采样见[同步版](notification-sync-benchmark.md)与[Outbox 版](notification-rabbitmq-benchmark.md)报告。

## 方案对比

| 维度 | 同步通知 | RabbitMQ + Transactional Outbox |
|---|---|---|
| QPS / P50 / P95 / P99 / 错误率 | 见[同步压测报告](notification-sync-benchmark.md) | 见[RabbitMQ 压测报告](notification-rabbitmq-benchmark.md) |
| 主业务是否等待通知 | 等待通知 INSERT 完成；通知失败只记录日志，互动仍成功 | 只等待业务数据与 Outbox 同事务提交，不等待 broker 或通知表 |
| 通知是否可能丢失 | best-effort：提交后通知写库失败时，互动已成功但通知可能缺失 | 业务与 Outbox 原子提交；Broker 恢复后可补发。超过消费重试上限时进死信队列等待人工处理 |
| 失败重试 | 无自动重试 | Publisher 失败在 MySQL Outbox 退避重试；Consumer 失败按 1s / 5s / 30s 重试，最多 8 次后进入 DLQ |
| 幂等 | 同步调用没有事件身份去重 | `event_id` 唯一索引使重复投递不会重复创建通知 |
| 运维复杂度 | 低，只依赖 MySQL | 较高，需要 RabbitMQ、监控积压与死信、处理重试和故障恢复 |
| 适用场景 | 低流量、可接受少量通知丢失、希望减少组件 | 可靠性和削峰更重要，允许最终一致，且团队能维护 MQ |

## 面试讲解稿

**为什么同步通知会增加接口响应时间？**

点赞、收藏或评论先提交业务事务，随后 HTTP 请求线程继续写 `notifications`。这会增加一次 MySQL 写入、索引维护和连接池占用；当数据库繁忙或通知表写入变慢时，请求必须等到写入成功或失败才返回。同步版本选择 best-effort：通知写入失败会记录日志，但不会把已提交的点赞、收藏或评论报告成失败。

**为什么 goroutine 不等于消息队列？**

goroutine 只是当前进程里的并发执行单元。进程退出、崩溃或被重新部署时，内存中的工作会消失；它本身没有持久化、ACK、可靠重试、死信、跨实例竞争控制或背压机制。消息队列把任务交给有持久化和确认机制的独立服务，Transactional Outbox 则先解决“业务已提交但消息还没可靠进入队列”的缺口。

**为什么保留 `notifications` 表？**

RabbitMQ 负责传递事件，不适合作为用户通知历史查询接口的主存储。通知列表分页、未读数统计、已读状态和长期查询都需要可查询、可更新的业务表。因此 `notifications` 是面向用户读取的持久化投影；增加 MQ 不会取代它。

**Outbox、RabbitMQ、`notifications` 各自负责什么？**

- 业务表记录点赞、收藏、评论等事实。
- `notification_outbox_events` 与对应业务记录在同一个 MySQL 事务提交，保存尚待投递的通知事件、状态、租约、重试次数和错误。
- RabbitMQ 负责把已发布事件可靠路由给消费者，并通过 publisher confirm、消费 ACK、延迟重试队列和死信队列管理传递过程。
- `notifications` 保存最终给用户展示的通知历史、未读和已读状态。

**为什么需要 Transactional Outbox？**

如果先提交业务事务，再直接发布 RabbitMQ 消息，进程可能在两步之间崩溃，造成点赞成功但没有通知；如果先发消息再提交业务事务，则可能通知已经送达但业务回滚。Outbox 把业务变更和事件记录放在同一个本地数据库事务里，只有业务事务提交后 Publisher 才能看到事件，所以恢复后能继续发送。

**如何保证至少一次投递？**

Publisher 使用短期租约和 `FOR UPDATE SKIP LOCKED` 抢占待发事件，发布持久化消息并等待 RabbitMQ publisher confirm；确认成功后才把 Outbox 标为已发布。若确认丢失或状态更新失败，同一事件可能被再次发布，这是至少一次语义。Consumer 成功写入 `notifications` 后才 ACK；未 ACK 的消息会重新投递。Broker 不可用时事件仍留在 MySQL，连接恢复后继续发布。

**如何处理重复消费、失败重试和死信？**

每个事件都有稳定的 `event_id`，`notifications.event_id` 有唯一索引。重复消费通过唯一键冲突幂等结束，不会创建第二条通知。Consumer 写库失败时先将消息发布到带 TTL 的重试队列，等待 1、5 或 30 秒后回到主队列；publisher confirm 成功后才 ACK 原消息。超过 8 次失败后消息进入 `aidevclub.notifications.dead`，同时保留 `event_id` 和错误信息。修复根因后可按 README 的 SQL 将对应已发布 Outbox 事件重新设为 pending，由同一 `event_id` 补发；确认通知已落库后再清理死信。

**为什么选 RabbitMQ，而不是 Kafka？**

这是站内通知任务，吞吐量低到中等，关键需求是 ACK、重试、死信和灵活路由。RabbitMQ 的工作队列与确认语义直接匹配。Kafka 更适合海量日志流、长期保留和大规模回放；本项目当前不需要这些能力，引入 Kafka 会增加超出需求的运维复杂度。

## 压测结论怎么讲

本地 k6 对两种模式运行相同接口、20 个虚拟用户、每项 30 秒、真实 JWT 鉴权和独立测试账号。结果是特定开发机与当前数据库配置下的观测，不应外推为所有部署环境的性能结论。当前单篇文章被 20 个用户共同更新计数，形成热门行锁竞争；Outbox 模式还需要在业务事务中写事件并由 Worker/Consumer 处理，因此这次本地请求延迟没有降低。Outbox 的主要收益是解耦通知持久化等待并提高故障恢复能力，不保证在每种负载下都提高 HTTP QPS。

完整机器参数、原始 k6 JSON、应用和容器采样见两份压测报告及它们链接的原始运行目录。
