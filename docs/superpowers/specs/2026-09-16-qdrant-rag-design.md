# Qdrant RAG 与可靠索引设计

## 目标

移除项目中的 Milvus、etcd 和 MinIO，改用单节点 Qdrant 实现文章 RAG 的 dense + sparse 混合检索；将文章索引从进程内 goroutine 改为 MySQL Outbox 和 RabbitMQ 驱动的可靠异步任务。

## 范围与约束

- Qdrant 是唯一的向量数据库；生产 Compose 仅运行一个 `qdrant/qdrant` 服务和 `qdrant_data` 持久卷。
- 不保留任何 Milvus 运行依赖、配置键、Python 客户端依赖或以 `milvus` 命名的代码文件。
- 向量维度通过 `AIDEVCLUB_AI_VECTOR_DIMENSION` 配置，默认 `1024`，并在写入前验证 embedding 维度。
- 内容数据库仍是 MySQL；Qdrant 索引可由已发布文章重建。
- 继续使用 DashScope 生成 dense embedding；不新增 Python 或模型服务。

## 切片

索引 worker 使用 Markdown 标题感知切片：按标题章节聚合，并在段落、句子边界递归拆分超过 800 tokens 的章节。目标大小 450–650 tokens，普通文本相邻块重叠 80 tokens；代码块和表格保持原子性，仅在单块超过硬限制时按行拆分。每个块携带标题、`heading_path`、`chunk_index` 和文章版本。

## Qdrant 集合与混合检索

集合含两个 named vectors：

- `dense`：1024 维 Cosine 向量。
- `lexical`：稀疏向量，Qdrant `idf` modifier 开启。

稀疏编码器不维护额外词表：将中文连续文本切成 Unicode 二元词，英文、数字、下划线和连字符组成的技术标识符保持完整；token 以稳定的 FNV-1a 32 位哈希作为稀疏维度，值为 `1 + ln(tf)`。Qdrant 在查询时应用集合 IDF，因此词频和文档频率不需要由应用重建。

查询同时 prefetch dense 和 lexical 候选（各 30），由 Qdrant 原生 RRF 融合，返回前 8 个；可选 rerank 再重排结果。文章范围查询使用 Qdrant payload filter。

## 可靠索引任务

新增 `document_index_outbox_events` 表。文章创建、更新、删除必须和 upsert/delete 索引事件处于同一 MySQL 事务。事件载荷包含事件 ID、操作、文章 ID 和 `updated_at` 版本；消费者对重复或乱序事件使用版本过滤，Qdrant point ID 固定为 `article_id:version:chunk_index`，并按文章 ID 过滤删除旧点。任务使用独立 RabbitMQ exchange、主队列、重试队列和死信队列。

已发布且未隐藏文章写 upsert；删除、隐藏、或转为草稿写 delete。消费者只在消息成功执行后 ACK；失败按既有通知队列策略重试并进入死信队列。

## 部署与前端

- 生产 Compose 用 Qdrant 替代 Milvus/etcd/MinIO，并将后端 `AIDEVCLUB_AI_QDRANT_URL` 配为容器地址。
- CI 对镜像标注 Git SHA，部署运行 `docker compose up -d --force-recreate --remove-orphans`。
- 部署验证除 `/healthz` 外，检查首页返回的 JS 入口含该 SHA；该 SHA 在前端构建时注入页面。
- AI 助手的 citation 改为 `/articles/{article_id}` 路由链接。

## 验收

Go 单测覆盖切片、稀疏编码、Qdrant 请求、Outbox 事务事件和消费幂等性；前端 typecheck/build 通过；`go test ./...` 和 `go build ./...` 通过。生产环境首页版本与推送提交一致，容器列表无 Milvus、etcd 或 MinIO。
