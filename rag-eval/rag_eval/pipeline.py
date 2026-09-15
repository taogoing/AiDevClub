from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from rag_eval.chunking import ChunkConfig, chunk_markdown
from rag_eval.config import Settings
from rag_eval.dataset import load_questions
from rag_eval.evaluate import evaluate_variant, write_comparison
from rag_eval.milvus_store import MilvusStore
from rag_eval.qwen import QwenClient
from rag_eval.source import load_articles


VARIANTS = ("dense_only", "sparse_only", "hybrid", "hybrid_rerank")


def ingest(settings: Settings, qwen: QwenClient, store: MilvusStore, questions_path: Path, recreate: bool = False) -> int:
    questions = load_questions(questions_path)
    article_ids = sorted({article_id for question in questions for article_id in question["article_ids"]})
    store.ensure_collection(recreate=recreate)
    articles = load_articles(settings, article_ids)
    missing = set(article_ids) - {int(article["id"]) for article in articles}
    if missing:
        raise ValueError(f"published, visible seed articles are missing from MySQL: {sorted(missing)}")
    store.delete_articles(article_ids)
    all_chunks = []
    for article in articles:
        markdown = f"# {article['title']}\n\n## 摘要\n\n{article['summary']}\n\n{article['content']}"
        all_chunks.extend(chunk_markdown(int(article["id"]), article["title"], markdown, ChunkConfig()))
    vectors = qwen.embed([chunk.text for chunk in all_chunks])
    store.insert_chunks(all_chunks, vectors)
    return len(all_chunks)


def run_evaluation(
    settings: Settings,
    questions_path: Path,
    output_dir: Path,
    store: MilvusStore,
    qwen: QwenClient,
) -> list[dict[str, Any]]:
    output_dir.mkdir(parents=True, exist_ok=True)
    questions = load_questions(questions_path)
    store.ensure_collection()
    queries = [item["question"] for item in questions]
    query_vectors = qwen.embed(queries)
    vector_by_id = {item["id"]: vector for item, vector in zip(questions, query_vectors, strict=True)}
    comparisons = []
    for variant in VARIANTS:
        runs = []
        for item in questions:
            question, vector = item["question"], vector_by_id[item["id"]]
            if variant == "dense_only":
                contexts = store.search_dense(vector, limit=5)
            elif variant == "sparse_only":
                contexts = store.search_sparse(question, limit=5)
            else:
                contexts = store.search_hybrid(question, vector, limit=20 if variant == "hybrid_rerank" else 5, candidates=30)
                if variant == "hybrid_rerank":
                    contexts = qwen.rerank(question, contexts, top_k=5)
            runs.append({"id": item["id"], "answer": qwen.answer(question, contexts), "contexts": contexts})
        (output_dir / f"{variant}-runs.json").write_text(json.dumps(runs, ensure_ascii=False, indent=2), encoding="utf-8")
        summary = evaluate_variant(settings, questions, runs, variant, output_dir)
        comparisons.append({"variant": variant, **summary})
    write_comparison(comparisons, output_dir)
    return comparisons
