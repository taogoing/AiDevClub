from __future__ import annotations

import argparse
import json
from pathlib import Path

from rag_eval.config import Settings


ROOT = Path(__file__).resolve().parent.parent


def main() -> None:
    parser = argparse.ArgumentParser(description="Local Milvus + Qwen RAG evaluation")
    parser.add_argument("command", choices=("smoke", "ingest", "evaluate", "all"))
    parser.add_argument("--questions", type=Path, default=ROOT / "data" / "questions.json")
    parser.add_argument("--output", type=Path, default=ROOT / "results")
    parser.add_argument("--recreate", action="store_true", help="drop and recreate the local evaluation collection")
    args = parser.parse_args()

    settings = Settings.from_env(require_api_key=args.command != "smoke")
    if args.command == "smoke":
        from rag_eval.chunking import ArticleChunk
        from rag_eval.milvus_store import MilvusStore

        smoke_settings = Settings(**{**settings.__dict__, "collection": f"{settings.collection}_smoke"})
        smoke_store = MilvusStore(smoke_settings)
        smoke_store.ensure_collection(recreate=True)
        vector = [1.0] + [0.0] * (settings.embedding_dimension - 1)
        smoke_store.insert_chunks(
            [ArticleChunk(900000, "Milvus smoke test", 0, ["BM25"], "Milvus BM25 sparse hybrid retrieval smoke test.", 8)],
            [vector],
        )
        sparse = smoke_store.search_sparse("BM25 hybrid retrieval", limit=1)
        dense = smoke_store.search_dense(vector, limit=1)
        hybrid = smoke_store.search_hybrid("BM25 hybrid retrieval", vector, limit=1, candidates=2)
        if not sparse or not dense or not hybrid:
            raise RuntimeError("Milvus sparse, dense, or hybrid smoke search returned no result")
        print(json.dumps({"sparse": len(sparse), "dense": len(dense), "hybrid": len(hybrid)}, indent=2))
        smoke_store.client.drop_collection(smoke_settings.collection)
        from pymilvus import connections
        connections.disconnect(smoke_store.alias)
        return
    from rag_eval.milvus_store import MilvusStore
    from rag_eval.pipeline import ingest, run_evaluation
    from rag_eval.qwen import QwenClient

    store = MilvusStore(settings)
    qwen = QwenClient(settings)
    if args.command in ("ingest", "all"):
        count = ingest(settings, qwen, store, args.questions, recreate=args.recreate)
        print(f"Indexed {count} Markdown chunks in Milvus")
    if args.command in ("evaluate", "all"):
        rows = run_evaluation(settings, args.questions, args.output, store, qwen)
        print(json.dumps(rows, ensure_ascii=False, indent=2))
        print(f"Evaluation files written to {args.output.resolve()}")


if __name__ == "__main__":
    main()
