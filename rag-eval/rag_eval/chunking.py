from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Iterable

import tiktoken


@dataclass(frozen=True)
class ChunkConfig:
    target_tokens: int = 600
    max_tokens: int = 800
    overlap_tokens: int = 80

    def __post_init__(self) -> None:
        if self.target_tokens < 1 or self.max_tokens < self.target_tokens:
            raise ValueError("max_tokens must be at least target_tokens, both positive")
        if self.overlap_tokens < 0 or self.overlap_tokens >= self.max_tokens:
            raise ValueError("overlap_tokens must be between 0 and max_tokens")


@dataclass(frozen=True)
class ArticleChunk:
    article_id: int
    title: str
    chunk_index: int
    heading_path: list[str]
    text: str
    token_count: int


@dataclass(frozen=True)
class _Block:
    text: str
    heading_path: tuple[str, ...]
    kind: str


_ENCODING = tiktoken.get_encoding("cl100k_base")
_HEADING = re.compile(r"^(#{1,6})\s+(.+?)\s*#*\s*$")
_TABLE_LINE = re.compile(r"^\s*\|.*\|\s*$")
_FENCE = re.compile(r"^\s*(```+|~~~+)")


def _parse_blocks(markdown: str, title: str) -> list[_Block]:
    lines = markdown.replace("\r\n", "\n").replace("\r", "\n").split("\n")
    headings: list[str] = []
    blocks: list[_Block] = []
    paragraph: list[str] = []
    paragraph_path: tuple[str, ...] = (title,)
    fence: str | None = None

    def flush() -> None:
        nonlocal paragraph
        text = "\n".join(paragraph).strip()
        if text:
            blocks.append(_Block(text, paragraph_path, "text"))
        paragraph = []

    i = 0
    while i < len(lines):
        line = lines[i]
        heading = _HEADING.match(line) if fence is None else None
        if heading:
            flush()
            level, label = len(heading.group(1)), heading.group(2).strip()
            headings = headings[: level - 1]
            headings.append(label)
            paragraph_path = tuple(headings) or (title,)
            i += 1
            continue

        start_fence = _FENCE.match(line)
        if start_fence and fence is None:
            flush()
            fence = start_fence.group(1)[:3]
            code = [line]
            i += 1
            while i < len(lines):
                code.append(lines[i])
                if lines[i].lstrip().startswith(fence):
                    i += 1
                    break
                i += 1
            blocks.append(_Block("\n".join(code).strip(), tuple(headings) or (title,), "code"))
            fence = None
            continue

        if not line.strip():
            flush()
            i += 1
            continue

        if _TABLE_LINE.match(line):
            flush()
            table = [line]
            i += 1
            while i < len(lines) and _TABLE_LINE.match(lines[i]):
                table.append(lines[i])
                i += 1
            blocks.append(_Block("\n".join(table), tuple(headings) or (title,), "table"))
            continue

        if not paragraph:
            paragraph_path = tuple(headings) or (title,)
        paragraph.append(line)
        i += 1

    flush()
    return blocks


def _token_count(text: str) -> int:
    return len(_ENCODING.encode(text))


def _split_oversized(text: str, budget: int, overlap: int) -> Iterable[str]:
    tokens = _ENCODING.encode(text)
    if len(tokens) <= budget:
        yield text
        return
    step = max(1, budget - overlap)
    for start in range(0, len(tokens), step):
        piece = _ENCODING.decode(tokens[start : start + budget]).strip()
        if piece:
            yield piece
        if start + budget >= len(tokens):
            break


def chunk_markdown(
    article_id: int,
    title: str,
    markdown: str,
    config: ChunkConfig = ChunkConfig(),
) -> list[ArticleChunk]:
    """Chunk Markdown by structural blocks, merging short adjacent sections.

    cl100k_base is used as a deterministic local token estimator. The target
    embedding API's tokenizer can differ, so the hard limit is kept conservative.
    """
    blocks = _parse_blocks(markdown, title)
    if not any(block.kind != "heading" for block in blocks):
        return []

    chunks: list[ArticleChunk] = []
    current: list[str] = []
    current_path: tuple[str, ...] = (title,)

    def render(path: tuple[str, ...], body: str) -> str:
        prefix = " > ".join(path)
        return f"Article: {title}\nSection: {prefix}\n\n{body}".strip()

    def emit() -> None:
        nonlocal current
        body = "\n\n".join(part for part in current if part.strip()).strip()
        if not body:
            current = []
            return
        rendered = render(current_path, body)
        chunks.append(
            ArticleChunk(
                article_id=article_id,
                title=title,
                chunk_index=len(chunks),
                heading_path=list(current_path),
                text=rendered,
                token_count=_token_count(rendered),
            )
        )
        current = []

    for block in blocks:
        if not current:
            current_path = block.heading_path
        block_text = block.text
        if current and block.heading_path != current_path:
            block_text = f"Section: {' > '.join(block.heading_path)}\n\n{block_text}"
        candidate = "\n\n".join([*current, block_text]).strip()
        if _token_count(render(current_path, candidate)) <= config.target_tokens:
            current.append(block_text)
            continue

        if current:
            emit()
        current_path = block.heading_path
        prefix_cost = _token_count(render(current_path, ""))
        # Leave room for separators and tokenizer boundary changes introduced
        # when the body is joined to its article/section prefix.
        budget = max(1, config.max_tokens - prefix_cost - 8)
        pieces = list(_split_oversized(block.text, budget, config.overlap_tokens))
        if len(pieces) == 1 and _token_count(render(current_path, pieces[0])) <= config.max_tokens:
            current = [pieces[0]]
            continue
        for piece in pieces[:-1]:
            current = [piece]
            emit()
        current = [pieces[-1]] if pieces else []
        if current and _token_count(render(current_path, current[0])) > config.target_tokens:
            emit()

    emit()
    return chunks
