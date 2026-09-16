# Qdrant RAG 与可靠索引 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Milvus with Qdrant native dense/sparse RRF retrieval and make article indexing durable through a RabbitMQ-backed Outbox.

**Architecture:** A Qdrant REST store owns collection provisioning, point writes/deletes, and native hybrid queries. Article service writes document-index events transactionally; a dedicated RabbitMQ publisher/consumer processes them into Qdrant. The Vue assistant renders citations as article links.

**Tech Stack:** Go 1.25, Gin, GORM/MySQL, RabbitMQ, Qdrant REST API, Vue 3, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-09-16-qdrant-rag-design.md`

## Global Constraints

- Use Qdrant as the only vector database; remove all Milvus, etcd, and MinIO code and deployment dependencies.
- Use named `dense` and `lexical` vectors and Qdrant native RRF fusion.
- The lexical encoder must use Chinese bigrams plus complete technical identifiers and FNV-1a 32-bit dimensions.
- Indexing events must be written in the same MySQL transaction as article mutations.
- Preserve unrelated user files and commit only migration work.

---

### Task 1: Qdrant store, lexical encoder, and Markdown chunker

**Files:**
- Create: `internal/service/qdrant_store.go`
- Create: `internal/service/qdrant_store_test.go`
- Create: `internal/service/markdown_chunker.go`
- Create: `internal/service/markdown_chunker_test.go`
- Modify: `internal/platform/config.go`
- Modify: `internal/app/services.go`
- Delete: `internal/service/ai_assistant.go`

**Interfaces:**
- Produces: `NewQdrantStore(QdrantConfig)`, `EnsureCollection`, `UpsertArticle`, `DeleteArticle`, `HybridSearch`, `ChunkMarkdown`, and `EncodeLexical`.
- Consumes: DashScope embedding HTTP response and Qdrant REST `/collections` and `/points/query` endpoints.

- [ ] **Step 1: Write failing lexical and chunker tests**

```go
func TestEncodeLexicalKeepsCodeIdentifiersAndChineseBigrams(t *testing.T) {
    vector := EncodeLexical("使用 singleflight.Group 防止缓存击穿")
    require.NotEmpty(t, vector.Indices)
    require.Equal(t, vector.Indices, sorted(vector.Indices))
}

func TestChunkMarkdownPreservesHeadingAndCodeBlock(t *testing.T) {
    chunks := ChunkMarkdown("# Redis\n\n```go\nvar group singleflight.Group\n```", ChunkOptions{TargetTokens: 450, MaxTokens: 800, OverlapTokens: 80})
    require.Len(t, chunks, 1)
    require.Equal(t, []string{"Redis"}, chunks[0].HeadingPath)
    require.Contains(t, chunks[0].Text, "singleflight.Group")
}
```

- [ ] **Step 2: Run tests and verify they fail because the APIs do not exist**

Run: `go test ./internal/service -run 'TestEncodeLexical|TestChunkMarkdown' -count=1`

- [ ] **Step 3: Implement the minimal lexical encoder and chunker**

```go
type SparseVector struct { Indices []uint32 `json:"indices"`; Values []float32 `json:"values"` }
func EncodeLexical(text string) SparseVector
func ChunkMarkdown(markdown string, options ChunkOptions) []MarkdownChunk
```

- [ ] **Step 4: Run the focused tests and confirm they pass**

Run: `go test ./internal/service -run 'TestEncodeLexical|TestChunkMarkdown' -count=1`

- [ ] **Step 5: Write failing Qdrant request tests**

```go
func TestEnsureCollectionCreatesDenseAndIDFSparseVectors(t *testing.T) { /* assert vectors.dense and sparse_vectors.lexical.modifier == idf */ }
func TestHybridSearchUsesPrefetchAndRRFFusion(t *testing.T) { /* assert two prefetches and query.fusion == rrf */ }
```

- [ ] **Step 6: Implement the Qdrant REST store and configuration**

```go
type QdrantConfig struct { URL, Collection string; VectorDimension int }
func (s *QdrantStore) HybridSearch(ctx context.Context, question string, dense []float64, articleID *uint, limit int) ([]AICitation, error)
```

- [ ] **Step 7: Run store tests and commit the task**

Run: `go test ./internal/service -run 'TestEnsureCollection|TestHybridSearch|TestEncodeLexical|TestChunkMarkdown' -count=1`

Commit: `git add internal/service internal/platform/config.go internal/app/services.go && git commit -m "feat: add qdrant hybrid retrieval store"`

### Task 2: Durable document-index Outbox and RabbitMQ worker

**Files:**
- Create: `internal/model/document_index_outbox.go`
- Create: `internal/repo/document_index_outbox.go`
- Create: `internal/service/document_index_mq.go`
- Create: `internal/service/document_index_mq_test.go`
- Modify: `internal/app/migrations.go`
- Modify: `internal/app/services.go`
- Modify: `internal/service/article.go`
- Modify: `internal/handler/article.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `DocumentIndexOutboxRepo.Enqueue`, `DocumentIndexMQ.Start`, `DocumentIndexMQ.Close`.
- Consumes: article service transaction, `QdrantStore.UpsertArticle`, and `QdrantStore.DeleteArticle`.

- [ ] **Step 1: Write failing transaction and event-version tests**

```go
func TestPublishedArticleCreatesUpsertEventInTheArticleTransaction(t *testing.T) { /* rollback leaves no article or event */ }
func TestDeleteArticleCreatesDeleteEvent(t *testing.T) { /* successful delete writes delete event */ }
func TestConsumerIgnoresOlderArticleVersion(t *testing.T) { /* newer event wins */ }
```

- [ ] **Step 2: Run the tests and verify the expected missing-event failures**

Run: `go test ./internal/service ./internal/repo -run 'TestPublishedArticleCreates|TestDeleteArticleCreates|TestConsumerIgnores' -count=1`

- [ ] **Step 3: Add model/repository/migration and write events inside article transactions**

```go
type DocumentIndexEvent struct { EventID string; Operation string; ArticleID uint; Version time.Time }
func (r *DocumentIndexOutboxRepo) Create(tx *gorm.DB, event *model.DocumentIndexOutboxEvent) error
```

- [ ] **Step 4: Add RabbitMQ topology and worker**

```go
const documentIndexExchange = "aidevclub.document-index"
func NewDocumentIndexMQ(url string, outbox *repo.DocumentIndexOutboxRepo, articles *repo.ArticleRepo, store *service.QdrantStore, interval, lease time.Duration) *DocumentIndexMQ
```

- [ ] **Step 5: Replace handler goroutines and start/stop the worker with the server**

Run: `go test ./internal/service ./internal/repo -run 'TestPublishedArticleCreates|TestDeleteArticleCreates|TestConsumerIgnores' -count=1`

- [ ] **Step 6: Commit the task**

Commit: `git add internal/model internal/repo internal/service internal/app cmd/server internal/handler && git commit -m "feat: index articles through durable outbox"`

### Task 3: Assistant API, frontend citations, and Qdrant deployment

**Files:**
- Modify: `internal/handler/ai_assistant.go`
- Modify: `frontend/src/components/AiAssistantPanel.vue`
- Modify: `frontend/src/api/aiAssistant.ts`
- Modify: `deploy/docker-compose.prod.yml`
- Modify: `deploy/.env.example`
- Modify: `docker-compose.yml`
- Modify: `rag-eval/docker-compose.yml`
- Rename: `rag-eval/rag_eval/milvus_store.py` to `rag-eval/rag_eval/qdrant_store.py`
- Rename: `rag-eval/rag_eval/milvus_schema.py` to `rag-eval/rag_eval/qdrant_schema.py`
- Modify: `rag-eval/requirements.txt`
- Modify: `rag-eval/rag_eval/pipeline.py`
- Modify: `rag-eval/tests/test_milvus_schema.py`
- Modify: `.github/workflows/deploy.yml`

**Interfaces:**
- Consumes: Qdrant API result citations and Vite build-time `VITE_GIT_SHA`.
- Produces: article citation anchors and an immutable-tag deployment verification step.

- [ ] **Step 1: Write failing frontend citation navigation test or component assertion**

```ts
expect(wrapper.get('[data-testid="citation-42"]').attributes('href')).toBe('/articles/42')
```

- [ ] **Step 2: Add an anchor for each citation and retain accessible title/path text**

Run: `cd frontend && npm run typecheck && npm run build`

- [ ] **Step 3: Replace all production and evaluation Compose Milvus stacks with Qdrant**

```yaml
qdrant:
  image: qdrant/qdrant:v1.16.2
  volumes: [qdrant_data:/qdrant/storage]
```

- [ ] **Step 4: Port evaluation store and tests from pymilvus to qdrant-client**

Run: `cd rag-eval && pytest -q`

- [ ] **Step 5: Make deployment immutable and verify frontend version**

```bash
docker compose up -d --force-recreate --remove-orphans
test "$(curl -fsS https://aidevclub.xyz/ | grep -o 'data-build-sha="[^"]*"')" = "data-build-sha=\"${GITHUB_SHA}\""
```

- [ ] **Step 6: Scan for Milvus references and commit the task**

Run: `rg -n -i 'milvus|etcd|minio|pymilvus' --glob '!docs/notification-benchmark-runs/**'`

Commit: `git add frontend deploy docker-compose.yml rag-eval .github && git commit -m "feat: deploy qdrant hybrid rag"`

### Task 4: Full verification, production migration, and release

**Files:**
- Modify: `README.md`
- Modify: `面试讲解/RAG助手搭建.md`

- [ ] **Step 1: Update operational documentation with Qdrant collection recreation and the one-time Milvus volume removal command**

```bash
docker compose stop milvus etcd minio
docker volume rm aidevclub_milvus_data aidevclub_milvus_etcd aidevclub_milvus_minio
```

- [ ] **Step 2: Run all verification**

Run: `go test ./... && go build ./... && cd frontend && npm run typecheck && npm run build && cd ../rag-eval && pytest -q`

- [ ] **Step 3: Inspect the complete diff and commit documentation**

Commit: `git add README.md 面试讲解/RAG助手搭建.md && git commit -m "docs: document qdrant rag operations"`

- [ ] **Step 4: Push master, wait for CI/CD, and verify production**

Run: `git push origin master`

Run: `curl -fsS https://aidevclub.xyz/healthz && curl -fsS https://aidevclub.xyz/`

