from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    api_key: str
    chat_model: str = "qwen3.8-flash"
    embedding_model: str = "qwen3.7-text-embedding-flash"
    rerank_model: str = "qwen3.7-text-rerank"
    openai_base_url: str = "https://dashscope.aliyuncs.com/compatible-mode/v1"
    rerank_url: str = "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank"
    mysql_host: str = "127.0.0.1"
    mysql_port: int = 3306
    mysql_user: str = "root"
    mysql_password: str = "root"
    mysql_database: str = "aidevclub"
    milvus_uri: str = "http://127.0.0.1:19530"
    collection: str = "article_chunks"
    embedding_dimension: int = 1024

    @classmethod
    def from_env(cls, require_api_key: bool = True) -> "Settings":
        key = os.getenv("DASHSCOPE_API_KEY", "").strip()
        if require_api_key and not key:
            raise ValueError("DASHSCOPE_API_KEY must be configured in the local environment")
        return cls(
            api_key=key,
            chat_model=os.getenv("QWEN_CHAT_MODEL", cls.chat_model),
            embedding_model=os.getenv("QWEN_EMBEDDING_MODEL", cls.embedding_model),
            rerank_model=os.getenv("QWEN_RERANK_MODEL", cls.rerank_model),
            openai_base_url=os.getenv("DASHSCOPE_OPENAI_BASE_URL", cls.openai_base_url),
            rerank_url=os.getenv("DASHSCOPE_RERANK_URL", cls.rerank_url),
            mysql_host=os.getenv("MYSQL_HOST", cls.mysql_host),
            mysql_port=int(os.getenv("MYSQL_PORT", str(cls.mysql_port))),
            mysql_user=os.getenv("MYSQL_USER", cls.mysql_user),
            mysql_password=os.getenv("MYSQL_PASSWORD", cls.mysql_password),
            mysql_database=os.getenv("MYSQL_DATABASE", cls.mysql_database),
            milvus_uri=os.getenv("MILVUS_URI", cls.milvus_uri),
            collection=os.getenv("MILVUS_COLLECTION", cls.collection),
            embedding_dimension=int(os.getenv("EMBEDDING_DIMENSION", str(cls.embedding_dimension))),
        )
