from __future__ import annotations

import time
from typing import Any

import httpx
from openai import OpenAI

from rag_eval.config import Settings


class QwenClient:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.openai = OpenAI(api_key=settings.api_key, base_url=settings.openai_base_url, timeout=90)
        self.http = httpx.Client(timeout=60)

    def embed(self, texts: list[str]) -> list[list[float]]:
        vectors: list[list[float]] = []
        for offset in range(0, len(texts), 20):
            batch = texts[offset : offset + 20]
            response = self.openai.embeddings.create(model=self.settings.embedding_model, input=batch)
            vectors.extend(item.embedding for item in sorted(response.data, key=lambda value: value.index))
        return vectors

    def answer(self, question: str, contexts: list[dict[str, Any]]) -> str:
        evidence = "\n\n".join(
            f"[chunk_id={item['chunk_id']} | article_id={item['article_id']} | title={item['title']}]\n{item['text']}"
            for item in contexts
        )
        response = self.openai.chat.completions.create(
            model=self.settings.chat_model,
            temperature=0,
            messages=[
                {"role": "system", "content": "你是 Go Pottery 论坛的帖子助手。仅根据给定证据回答；证据不足时明确说帖子中没有足够信息。回答后用 [chunk_id] 标注依据，不要编造。"},
                {"role": "user", "content": f"问题：{question}\n\n证据：\n{evidence}"},
            ],
        )
        return response.choices[0].message.content or ""

    def rerank(self, query: str, candidates: list[dict[str, Any]], top_k: int) -> list[dict[str, Any]]:
        if not candidates:
            return []
        response = self.http.post(
            self.settings.rerank_url,
            headers={"Authorization": f"Bearer {self.settings.api_key}", "Content-Type": "application/json"},
            json={
                "model": self.settings.rerank_model,
                "input": {"query": query, "documents": [row["text"] for row in candidates]},
                "parameters": {"return_documents": False},
            },
        )
        response.raise_for_status()
        payload = response.json()
        output = payload.get("output", {})
        results = output.get("results", [])
        ranked = []
        for result in sorted(results, key=lambda value: value.get("relevance_score", 0), reverse=True)[:top_k]:
            index = int(result["index"])
            ranked.append({**candidates[index], "rerank_score": result.get("relevance_score", 0)})
        return ranked
