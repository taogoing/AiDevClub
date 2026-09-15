from __future__ import annotations

import json
from pathlib import Path
from typing import Any


def validate_questions(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    if not isinstance(rows, list) or not rows:
        raise ValueError("question dataset must be a non-empty JSON array")
    seen: set[str] = set()
    for index, row in enumerate(rows):
        if not isinstance(row, dict):
            raise ValueError(f"question at index {index} must be an object")
        question_id = str(row.get("id", "")).strip()
        if not question_id or question_id in seen:
            raise ValueError(f"question ids must be non-empty and unique: {question_id!r}")
        seen.add(question_id)
        if not isinstance(row.get("article_ids"), list) or not row["article_ids"]:
            raise ValueError(f"question {question_id} must include article_ids")
        if not all(isinstance(value, int) and value > 0 for value in row["article_ids"]):
            raise ValueError(f"question {question_id} article_ids must be positive integers")
        for field in ("question", "reference"):
            if not isinstance(row.get(field), str) or not row[field].strip():
                raise ValueError(f"question {question_id} requires a non-empty {field}")
    return rows


def load_questions(path: str | Path) -> list[dict[str, Any]]:
    try:
        rows = json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ValueError(f"could not load evaluation questions from {path}: {exc}") from exc
    return validate_questions(rows)
