import tiktoken

from rag_eval.chunking import ChunkConfig, chunk_markdown


def token_count(text: str) -> int:
    return len(tiktoken.get_encoding("cl100k_base").encode(text))


def test_retains_heading_path_and_article_metadata():
    chunks = chunk_markdown(
        article_id=42,
        title="RAG 实践",
        markdown="# RAG 实践\n\n## 召回\n\n向量召回用于发现语义相关段落。",
        config=ChunkConfig(target_tokens=100, max_tokens=120, overlap_tokens=10),
    )

    assert len(chunks) == 1
    assert chunks[0].article_id == 42
    assert chunks[0].title == "RAG 实践"
    assert chunks[0].heading_path == ["RAG 实践", "召回"]
    assert "向量召回" in chunks[0].text


def test_splits_oversized_section_under_hard_limit_with_overlap():
    paragraph = " ".join(f"retrieval passage {i:03d} contains useful evidence." for i in range(180))
    chunks = chunk_markdown(
        article_id=7,
        title="Dense retrieval",
        markdown=f"# Dense retrieval\n\n{paragraph}",
        config=ChunkConfig(target_tokens=90, max_tokens=110, overlap_tokens=12),
    )

    assert len(chunks) > 1
    assert all(token_count(chunk.text) <= 110 for chunk in chunks)
    assert chunks[0].text[-60:] in chunks[1].text


def test_keeps_fenced_code_and_markdown_table_together_when_they_fit():
    markdown = """# Example

```go
func answer() string {
    return "grounded"
}
```

| mode | purpose |
| --- | --- |
| dense | semantic match |
| BM25 | exact terms |
"""
    chunks = chunk_markdown(
        article_id=3,
        title="Example",
        markdown=markdown,
        config=ChunkConfig(target_tokens=200, max_tokens=220, overlap_tokens=20),
    )

    assert len(chunks) == 1
    assert "```go\nfunc answer() string" in chunks[0].text
    assert "| dense | semantic match |" in chunks[0].text


def test_does_not_emit_empty_chunks_for_heading_only_markdown():
    chunks = chunk_markdown(
        article_id=1,
        title="Empty sections",
        markdown="# Empty sections\n\n## First\n\n## Second",
        config=ChunkConfig(target_tokens=100, max_tokens=120, overlap_tokens=10),
    )

    assert chunks == []
