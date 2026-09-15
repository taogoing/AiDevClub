import json
from collections import Counter

import pytest

from rag_eval.dataset import load_questions, validate_questions


def test_checked_in_eval_set_has_50_unique_grounded_questions():
    questions = load_questions("data/questions.json")

    assert len(questions) == 50
    assert len({item["id"] for item in questions}) == 50
    assert all(item["article_ids"] and all(article_id > 0 for article_id in item["article_ids"]) for item in questions)
    assert all(item["question"].strip() and item["reference"].strip() for item in questions)
    assert all(len(item["reference"]) >= 20 for item in questions)
    assert Counter(item["article_ids"][0] for item in questions) == Counter({article_id: 5 for article_id in range(1, 11)})


def test_rejects_duplicate_ids_and_missing_reference():
    sample = {"id": "q1", "article_ids": [1], "question": "Q?", "reference": "A."}
    with pytest.raises(ValueError, match="unique"):
        validate_questions([sample, sample])
    with pytest.raises(ValueError, match="reference"):
        validate_questions([{**sample, "reference": " "}])


def test_loader_rejects_malformed_json(tmp_path):
    path = tmp_path / "bad.json"
    path.write_text(json.dumps([{"id": "q1"}]), encoding="utf-8")

    with pytest.raises(ValueError):
        load_questions(path)
