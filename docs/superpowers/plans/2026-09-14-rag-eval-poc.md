# Local RAG Evaluation PoC Plan

## Goal

Build an isolated, local Docker-based evaluation harness for the Go Pottery Markdown article corpus. Compare dense retrieval, Milvus BM25 sparse retrieval, Milvus hybrid retrieval, and hybrid retrieval with Qwen reranking. Evaluate 50 curated question/reference examples with Ragas. This is an evaluation PoC, not production UI integration.

## Decisions

- Milvus Standalone for vector and BM25 sparse retrieval; no MySQL full-text search.
- DashScope-compatible Qwen embedding, chat, and rerank APIs; credentials come only from the user's local environment and are never committed.
- Markdown-aware fixed chunking: target 600 tokens, hard maximum 800 tokens, 80-token overlap only when a single section must be split; preserve headings, code fences, tables, and article metadata.
- Read published, visible Markdown posts from the existing MySQL database; exclude synthetic benchmark fixtures by default through an explicit corpus filter.
- Docker Compose for Milvus dependencies and the evaluation runner. The app's existing database remains host-managed and is reached through `host.docker.internal`.
- Four retrieval variants are evaluated against the same 50 examples and same answer model; record Ragas answer relevancy, faithfulness, context precision/recall, answer correctness, and retrieval hit rate.

## Implementation Tasks (TDD)

1. Add unit tests for heading-aware Markdown chunking, code/table preservation, overlap, and maximum size; then implement the chunker.
2. Add validation tests for the 50-row QA dataset format and article/chunk attribution; then implement loading and corpus export.
3. Implement Qwen API adapters, Milvus schema/index creation, ingestion, four retrieval modes, and Ragas evaluation output.
4. Add Docker Compose, environment example, operator README, and a small no-credentials smoke path.
5. Run Python unit tests, existing Go tests, compose/config validation where Docker is available, and report whether live Ragas evaluation ran (requires a locally configured API key).

## Validation

- Pure unit tests require no API key, MySQL, or Milvus.
- Integration/evaluation runs require Docker services, the existing local MySQL corpus, and `DASHSCOPE_API_KEY` supplied locally.
- Results are informational because the local corpus is small and synthetic benchmark posts are excluded.
