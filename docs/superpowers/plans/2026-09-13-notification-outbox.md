# Notification Outbox Implementation Plan

> **For agentic workers:** Execute this plan task by task with tests before implementation.

**Goal:** Compare synchronous best-effort notifications with RabbitMQ delivery backed by a transactional MySQL outbox, using reproducible local k6 runs.

**Architecture:** A configured notification mode keeps both comparison paths in the same binary. Synchronous mode performs the existing interaction transaction, then waits for best-effort notification persistence. Outbox mode writes event rows in the interaction transaction; a publisher claims and publishes rows to RabbitMQ, and an idempotent consumer stores notifications before ACK. A retry/dead-letter path preserves failed events for operator recovery.

**Tech Stack:** Go 1.25, Gin, GORM, MySQL 8, RabbitMQ, Docker Compose, Vue 3/TypeScript, k6.

**Spec:** User-provided request in this task (two-stage sync → RabbitMQ + transactional outbox, local Docker/k6 comparison).

## Global Constraints

- Preserve unrelated working-tree changes; do not reset, checkout, or overwrite user files.
- Keep `notifications` as the notification-history/read-state table.
- Do not change Redis hot-ranking logic.
- RabbitMQ is the only broker; business success must not depend on RabbitMQ availability in outbox mode.
- Outbox events and their business interaction writes commit in one MySQL transaction.
- Sync notification errors are logged and best-effort after the business transaction commits.
- Compare identical API requests, data, auth, k6 scenarios, concurrency and duration.
- Record only observed benchmark results; never reuse historical hot-ranking results.

---

### Task 1: Establish baseline and comparison harness

**Files:**
- Create: `scripts/notification-benchmark.js`
- Create: `scripts/notification-benchmark.ps1`
- Create: `docs/notification-sync-benchmark.md`
- Modify: `internal/platform/config.go` only if mode configuration is needed by the final implementation.

- [x] Inspect routes, auth, migrations, runtime startup and untracked benchmark files; preserve user-owned files.
- [x] Add a k6 scenario that prepares isolated benchmark identities/content, authenticates, and exercises article like/favorite/comment with repeatable valid state transitions.
- [x] Capture host, Docker, MySQL and application resource snapshots alongside k6 summary output.
- [x] Run and record the synchronous baseline before changing behavior.

### Task 2: Synchronous notification behavior

**Files:**
- Modify: `internal/model/notification.go`
- Modify: `internal/service/article.go`
- Modify: `internal/service/skill.go`
- Modify: `internal/service/mcp_server.go`
- Modify: `internal/service/comment.go`
- Modify: `internal/service/resource_comment.go`
- Modify: `frontend/src/views/NotificationsView.vue`
- Test: corresponding service test files.

- [x] Add failing tests for notification creation on favorites, likes and comments, plus no notification on self-interaction, toggle-off or delete.
- [x] Run the tests to confirm the expected missing behavior.
- [x] Remove notification goroutines; after successful business commits call notification persistence inline, log persistence errors and preserve business success.
- [x] Add favorite notification types and frontend labels/icons.
- [x] Run focused tests, full Go tests and build; rerun the sync benchmark and finalize the measured report.

### Task 3: Outbox model and transaction integration

**Files:**
- Create: `internal/model/notification_outbox.go`
- Create: `internal/repo/notification_outbox.go`
- Modify: `internal/model/notification.go`
- Modify: service and repository files for article, Skill, MCP Server, article comments and resource comments.
- Modify: `internal/testutil/testutil.go`
- Test: outbox repository and service tests.

- [x] Add failing atomicity tests proving interaction and outbox row commit or roll back together.
- [x] Implement typed outbox payload/status model, event-id uniqueness, safe pending-row claims, retry scheduling and dead-letter state.
- [x] Integrate outbox event insertion into each relevant business transaction; switch those services based on notification mode.
- [x] Test RabbitMQ-unavailable behavior by proving the business row and pending outbox row remain committed.

### Task 4: RabbitMQ publisher and notification consumer

**Files:**
- Create: `internal/service/notification_outbox.go`
- Create: `internal/service/notification_mq.go`
- Modify: `internal/model/notification.go`
- Modify: server bootstrap and config files under `internal/app`, `internal/platform`, and `cmd/server`.
- Test: publisher/consumer tests using interfaces/fakes plus RabbitMQ integration where available.

- [x] Add failing tests for publish failure/backoff, eventual republish, duplicate consume, consumer failure and ACK ordering.
- [x] Implement RabbitMQ topology with durable exchange/queue, persistent messages, manual ACK, retry/dead-letter policy and event-id propagation.
- [x] Implement a multi-instance-safe DB claim/lease strategy; only mark published after broker confirmation.
- [x] Implement consumer persistence and unique event-id idempotency; ACK only after successful DB commit.
- [x] Start workers in outbox mode and gracefully stop publishers/consumers during server shutdown.

### Task 5: Local infrastructure and operator documentation

**Files:**
- Modify: `docker-compose.yml`
- Modify: `README.md`
- Create: `docs/notification-rabbitmq-benchmark.md`
- Modify/create: `.env.example` or the project’s documented environment template.

- [x] Add RabbitMQ management image, ports, health check, named volume and configuration.
- [x] Document sync/outbox mode, RabbitMQ setup, retry/dead-letter recovery and local benchmark commands.
- [x] Run tests/build, then execute the same k6 harness in outbox mode and record measured data/resources.

### Task 6: Final verification and comparison

- [x] Compare sync and outbox benchmark inputs and confirm they match.
- [x] Run `go test ./...` and `go build ./...` against the final state.
- [x] Verify Docker Compose health, mode switching, pending-event recovery and report traceability.
- [x] Review the final diff to ensure user-owned untracked files and hot-ranking code are untouched.
- [x] Add the requested comparison table and interview explanation to project documentation.
