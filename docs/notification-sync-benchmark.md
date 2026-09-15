# 同步通知版本 k6 压测

## 范围与方法

测试在开发机本地 Docker 环境执行，测量“业务事务提交后同步写 `notifications`”的 HTTP 接口行为。每种操作使用一个新建且已发布的测试文章；每个虚拟用户使用专属测试账号和 JWT。点赞与收藏接口是 toggle 操作，20 个用户反复切换状态，只有状态变为点赞/收藏时产生通知；评论每个请求都新建评论。测试不使用热榜压测数据，也不写入或重置已有用户内容。

| 项目 | 配置 |
|---|---|
| 机器 | Windows 11 专业版 10.0.26200，Intel Core i9-12900H，20 个逻辑处理器，16,890,978,304 bytes RAM（约 15.7 GiB） |
| 运行环境 | Docker Engine 29.6.2 / Compose 5.3.1；MySQL `mysql:8`；Redis `redis:7`；RabbitMQ `rabbitmq:4-management`；Go 1.25.0；k6 2.2.0 Windows/amd64 |
| 服务地址 | 测试 HTTP 服务 `http://localhost:18080`；MySQL `localhost:3306/aidevclub`；Redis `localhost:16379`；RabbitMQ `localhost:5672` |
| 鉴权 | 专用测试用户 JWT Bearer Token；每个 VU 一个用户；通过真实 HTTP JWT 中间件 |
| 工具与模型 | k6 `constant-vus`，20 VU，持续 30 秒；每次迭代发一个 HTTP 请求；k6 trend 统计 avg、P50、P90、P95、P99、max |
| 数据前置条件 | Docker Compose 的 MySQL、Redis、RabbitMQ 健康；k6 已安装；工作区 Go 依赖可用；测试用户、文章由 `cmd/notification-bench` 专用助手新增 |

接口均为 `POST`：

| 场景 | 路径 |
|---|---|
| 点赞文章 | `/api/v1/articles/{article_id}/like` |
| 收藏文章 | `/api/v1/articles/{article_id}/favorite` |
| 发表评论 | `/api/v1/articles/{article_id}/comments`，正文为唯一的 k6 评论文本 |

## 复现

在项目根目录的 Windows PowerShell 中运行：

```powershell
docker compose up -d mysql redis rabbitmq
.\scripts\run-notification-benchmark.ps1 -Mode both -VUs 20 -Duration 30s
```

脚本启动同一构建的服务二进制，先设为 `sync` 模式再设为 `outbox` 模式；使用隔离端口 `18080/18081`，不会占用默认的 `8080`。每个场景使用自己的文章与账号。可单独复跑同步部分：

```powershell
.\scripts\run-notification-benchmark.ps1 -Mode sync -VUs 20 -Duration 30s
```

复现前确保已有 Outbox 无待发事件，避免其他本地任务的队列积压干扰测试；本次最终轮开始前确认 RabbitMQ 主队列、三个重试队列、死信队列 ready/unacked 均为 0。脚本不会清理 Docker volume 或删除既有业务数据。

最终轮原始记录：[`20260913-194011-4bb7d47c`](notification-benchmark-runs/README.md#synchronous-run-20260913-194011-4bb7d47c)。[点赞](notification-benchmark-runs/20260913-194011-4bb7d47c/sync-like-k6-summary.json)、[收藏](notification-benchmark-runs/20260913-194011-4bb7d47c/sync-favorite-k6-summary.json)、[评论](notification-benchmark-runs/20260913-194011-4bb7d47c/sync-comment-k6-summary.json)的原始 k6 JSON、完整输出、运行参数及逐次资源采样均在该目录；机器信息见 [`machine.json`](notification-benchmark-runs/20260913-194011-4bb7d47c/machine.json)。

## HTTP 结果

延迟单位为毫秒；错误率是 k6 自定义的非 200 比例。每项共 20 VU、30 秒。所有 k6 状态码检查通过，错误率为 0%。

| 接口 | 请求数 | QPS | P50 | P95 | P99 | 错误率 |
|---|---:|---:|---:|---:|---:|---:|
| 点赞文章 | 1,618 | 53.36 | 347.6 | 589.8 | 687.2 | 0% |
| 收藏文章 | 1,629 | 53.50 | 348.7 | 575.5 | 645.5 | 0% |
| 发表评论 | 2,079 | 68.62 | 290.7 | 320.2 | 332.8 | 0% |

## 资源使用

CPU 百分比来自 Windows 容器的 `docker stats --no-stream` 采样，CPU 的单核百分比可超过 100%；内存为场景采样峰值。应用 CPU 是 Go 进程 CPU 时间折算为整台机器的平均占用。资源采样器每轮还需调用 Docker CLI，因此每场景实际记录 8 个样本；原始时间戳见 JSONL 文件。MySQL/RabbitMQ/Redis 的容器值是同期观测值，包含容器自身后台活动；同步路径不发布 RabbitMQ 消息，RabbitMQ 数字不能解读为同步接口消耗的 MQ 资源。

| 场景 | 应用 CPU / 整机 | 应用峰值内存 | MySQL CPU 平均/峰值；峰值内存 | RabbitMQ CPU 平均/峰值；峰值内存 | Redis CPU 平均/峰值；峰值内存 |
|---|---:|---:|---:|---:|---:|
| 点赞 | 0.50% | 70.8 MiB | 16.6% / 19.5%；952.9 MiB | 30.6% / 122.7%；254.7 MiB | 0.8% / 1.0%；16.4 MiB |
| 收藏 | 0.50% | 71.1 MiB | 16.7% / 20.5%；953.3 MiB | 54.5% / 433.5%；170.5 MiB | 0.8% / 1.0%；16.8 MiB |
| 评论 | 0.65% | 72.1 MiB | 21.8% / 27.3%；953.2 MiB | 48.6% / 386.8%；265.7 MiB | 1.0% / 1.2%；16.4 MiB |

## 解读

该机器上点赞/收藏需要更新同一篇文章的计数行，20 VU 会争用这行；同步通知增加了请求线程的通知 INSERT，但这一负载下点赞和收藏仍约 53 QPS。评论会写评论、更新同一篇文章的评论数并写通知。结果仅代表本机、本次 MySQL Docker 配置和此数据模型，不代表生产容量承诺。k6 测量 HTTP 完成时间；Outbox 版的独立投递和通知落库时间不计入 HTTP 延迟，另由脚本在切换场景前等待队列排空。
