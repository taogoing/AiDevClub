from __future__ import annotations

import json
from typing import Any

from pymilvus import (
    AnnSearchRequest,
    Collection,
    CollectionSchema,
    DataType,
    FieldSchema,
    Function,
    FunctionType,
    MilvusClient,
    RRFRanker,
    connections,
)

from rag_eval.chunking import ArticleChunk
from rag_eval.config import Settings
from rag_eval.milvus_schema import chinese_analyzer_params


class MilvusStore:
    alias = "rag-eval"

    def __init__(self, settings: Settings):
        self.settings = settings
        connections.connect(alias=self.alias, uri=settings.milvus_uri)
        self.client = MilvusClient(uri=settings.milvus_uri)
        self.collection: Collection | None = None

    def ensure_collection(self, recreate: bool = False) -> None:
        name = self.settings.collection
        if self.client.has_collection(name) and recreate:
            self.client.drop_collection(name)
        if not self.client.has_collection(name):
            schema = CollectionSchema(
                fields=[
                    FieldSchema("chunk_id", DataType.VARCHAR, is_primary=True, max_length=80),
                    FieldSchema("article_id", DataType.INT64),
                    FieldSchema("chunk_index", DataType.INT64),
                    FieldSchema("title", DataType.VARCHAR, max_length=1024),
                    FieldSchema("heading_path", DataType.VARCHAR, max_length=8192),
                    FieldSchema("text", DataType.VARCHAR, max_length=65535, enable_analyzer=True, analyzer_params=chinese_analyzer_params()),
                    FieldSchema("dense", DataType.FLOAT_VECTOR, dim=self.settings.embedding_dimension),
                    FieldSchema("sparse", DataType.SPARSE_FLOAT_VECTOR),
                ],
                description="Markdown article chunks with dense and BM25 sparse vectors",
            )
            schema.add_function(
                Function(
                    name="text_bm25",
                    input_field_names=["text"],
                    output_field_names=["sparse"],
                    function_type=FunctionType.BM25,
                )
            )
            self.client.create_collection(collection_name=name, schema=schema)
            collection = Collection(name, using=self.alias)
            collection.create_index("dense", {"index_type": "HNSW", "metric_type": "COSINE", "params": {"M": 16, "efConstruction": 200}})
            collection.create_index("sparse", {"index_type": "SPARSE_INVERTED_INDEX", "metric_type": "BM25", "params": {"inverted_index_algo": "DAAT_MAXSCORE"}})
            collection.create_index("article_id", {"index_type": "INVERTED"})
        self.collection = Collection(name, using=self.alias)
        self.collection.load()

    def insert_chunks(self, chunks: list[ArticleChunk], vectors: list[list[float]]) -> None:
        if not chunks:
            return
        if len(chunks) != len(vectors):
            raise ValueError("chunk and embedding counts do not match")
        rows = []
        for chunk, vector in zip(chunks, vectors, strict=True):
            rows.append({
                "chunk_id": f"a{chunk.article_id}-c{chunk.chunk_index}",
                "article_id": chunk.article_id,
                "chunk_index": chunk.chunk_index,
                "title": chunk.title,
                "heading_path": json.dumps(chunk.heading_path, ensure_ascii=False),
                "text": chunk.text,
                "dense": vector,
            })
        assert self.collection is not None
        self.collection.insert(rows)
        self.collection.flush()

    def delete_articles(self, article_ids: list[int]) -> None:
        if not article_ids:
            return
        assert self.collection is not None
        expression = f"article_id in [{','.join(str(value) for value in article_ids)}]"
        self.collection.delete(expression)
        self.collection.flush()

    def search_dense(self, vector: list[float], limit: int = 5) -> list[dict[str, Any]]:
        return self._search(vector, "dense", {"metric_type": "COSINE", "params": {"ef": 64}}, limit)

    def search_sparse(self, query: str, limit: int = 5) -> list[dict[str, Any]]:
        return self._search(query, "sparse", {"metric_type": "BM25", "params": {}}, limit)

    def search_hybrid(self, query: str, vector: list[float], limit: int = 5, candidates: int = 20) -> list[dict[str, Any]]:
        assert self.collection is not None
        requests = [
            AnnSearchRequest([vector], "dense", {"metric_type": "COSINE", "params": {"ef": 64}}, limit=candidates),
            AnnSearchRequest([query], "sparse", {"metric_type": "BM25", "params": {}}, limit=candidates),
        ]
        results = self.collection.hybrid_search(
            reqs=requests,
            rerank=RRFRanker(k=60),
            limit=limit,
            output_fields=["article_id", "chunk_index", "title", "heading_path", "text"],
        )[0]
        return [self._format(hit) for hit in results]

    def _search(self, data: Any, field: str, params: dict[str, Any], limit: int) -> list[dict[str, Any]]:
        assert self.collection is not None
        results = self.collection.search(
            data=[data],
            anns_field=field,
            param=params,
            limit=limit,
            output_fields=["article_id", "chunk_index", "title", "heading_path", "text"],
        )[0]
        return [self._format(hit) for hit in results]

    @staticmethod
    def _format(hit: Any) -> dict[str, Any]:
        entity = hit.entity
        return {
            "chunk_id": hit.id,
            "score": float(hit.score),
            "article_id": entity.get("article_id"),
            "chunk_index": entity.get("chunk_index"),
            "title": entity.get("title"),
            "heading_path": entity.get("heading_path"),
            "text": entity.get("text"),
        }
