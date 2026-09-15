from __future__ import annotations

import csv
import json
from pathlib import Path
from typing import Any

from datasets import Dataset
from langchain_openai import ChatOpenAI, OpenAIEmbeddings
from ragas import evaluate
from ragas.metrics import answer_correctness, answer_relevancy, context_precision, context_recall, faithfulness

from rag_eval.config import Settings


METRICS = [faithfulness, answer_relevancy, context_precision, context_recall, answer_correctness]


def evaluate_variant(
    settings: Settings,
    questions: list[dict[str, Any]],
    runs: list[dict[str, Any]],
    variant: str,
    output_dir: Path,
) -> dict[str, Any]:
    llm = ChatOpenAI(
        model=settings.chat_model,
        api_key=settings.api_key,
        base_url=settings.openai_base_url,
        temperature=0,
        max_retries=3,
        extra_body={"enable_thinking": False},
    )
    embeddings = OpenAIEmbeddings(
        model=settings.embedding_model,
        api_key=settings.api_key,
        base_url=settings.openai_base_url,
        check_embedding_ctx_length=False,
    )
    by_id = {run["id"]: run for run in runs}
    rows = []
    for item in questions:
        run = by_id[item["id"]]
        rows.append({
            "question": item["question"],
            "answer": run["answer"],
            "contexts": [hit["text"] for hit in run["contexts"]],
            "ground_truth": item["reference"],
        })
    result = evaluate(Dataset.from_list(rows), metrics=METRICS, llm=llm, embeddings=embeddings, raise_exceptions=False)
    frame = result.to_pandas()
    output_dir.mkdir(parents=True, exist_ok=True)
    frame.to_csv(output_dir / f"{variant}.csv", index=False, encoding="utf-8-sig")
    summary: dict[str, Any] = {name: float(value) for name, value in frame.mean(numeric_only=True).items()}
    summary["retrieval_hit_at_5"] = sum(
        any(hit["article_id"] in question["article_ids"] for hit in by_id[question["id"]]["contexts"][:5])
        for question in questions
    ) / len(questions)
    (output_dir / f"{variant}-summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    return summary


def write_comparison(rows: list[dict[str, Any]], output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    columns = ["variant", *sorted({key for row in rows for key in row if key != "variant"})]
    with (output_dir / "comparison.csv").open("w", encoding="utf-8-sig", newline="") as file:
        writer = csv.DictWriter(file, fieldnames=columns)
        writer.writeheader()
        writer.writerows(rows)
