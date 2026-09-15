def chinese_analyzer_params() -> dict[str, str]:
    """Milvus analyzer config: the tokenizer name belongs under `tokenizer`."""
    return {"tokenizer": "jieba"}
