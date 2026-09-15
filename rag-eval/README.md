# Local RAG evaluation PoC

This directory contains an isolated evaluation harness. It does not yet add the AI-assistant button or change the Go Pottery post flow.

## What is compared

All four variants use the same Markdown chunks and 50 curated questions:

1. Dense vector retrieval (top 5)
2. Milvus BM25 sparse retrieval (top 5; no MySQL full-text index)
3. Dense + sparse hybrid retrieval with reciprocal-rank fusion (top 5)
4. Hybrid retrieval (30 candidates) + Qwen reranking (top 5)

Each retrieved set is passed to the same grounded Qwen answer prompt. Ragas scores faithfulness, answer relevancy, context precision, context recall, and answer correctness. A `retrieval_hit_at_5` score tracks whether a question's expected article appeared in the top five. Results and per-question traces go into `results/`.

## Markdown chunking

The chunker parses heading paths, preserves fenced code blocks and Markdown tables when they fit, and merges short adjacent sections up to a target of 600 tokens. Only an oversized structural block is split; those pieces use an 80-token overlap and stay under a hard 800-token estimate. `cl100k_base` is used for a repeatable local estimate; Qwen's tokenizer can differ, so treat these as initial tuning values and compare retrieval quality before production.

## Run with Docker Desktop

The existing app database must already be running locally on port 3306 with the Go Pottery schema and seed articles 1–10. The evaluation dataset intentionally selects those ten real seeded posts and excludes local notification/ranking benchmark fixtures.

In PowerShell, from this directory:

```powershell
Copy-Item .env.example .env.local
# Edit .env.local and set DASHSCOPE_API_KEY there; this file is git-ignored.
docker compose --env-file .env.local up -d etcd minio milvus
# Optional: validate Milvus dense, BM25 sparse, and hybrid search without a model key.
docker compose --env-file .env.local run --build --rm rag-eval smoke
docker compose --env-file .env.local run --build --rm rag-eval all --recreate
```

The final command makes paid model calls: it embeds the corpus/questions, generates 200 answers, calls the reranker for the reranked track, then runs Ragas metrics. The configured key is read from the local environment and is not included in this repository. Docker persists Milvus data in named volumes. Use `--recreate` to replace the local evaluation collection.

To use another local MySQL host/port, set `MYSQL_HOST`/`MYSQL_PORT` in Compose or edit the evaluation service environment. On Docker Desktop for Windows, `host.docker.internal` points from the runner container to the host.

## Run tests without API credentials or Docker

```powershell
python -m pip install -r requirements.txt
python -m pytest -q
```

## Dataset caveat

The 50 rows are a hand-curated smoke/relative-comparison set over only ten short seeded articles, five questions per article. They are useful for checking that the retrieval variants run and for finding obvious regressions; they are not a representative benchmark and should not be used to choose production settings without a larger, independently reviewed corpus.

## Production direction after evaluation

If these comparisons support the approach, the next step is to integrate an asynchronous indexing job after article create/update/delete: persist canonical Markdown in MySQL, publish an outbox/job event, chunk and embed idempotently, write chunks to Milvus, track index version/status, and delete or tombstone prior chunks on update or moderation. The AI popup should scope retrieval to the current article or an explicit set of authorized articles and cite source sections. Keep MySQL as the source of truth and validate article visibility/permissions before returning retrieved content.
