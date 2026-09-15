from __future__ import annotations

from typing import Any

import pymysql
from pymysql.cursors import DictCursor

from rag_eval.config import Settings


def load_articles(settings: Settings, article_ids: list[int] | None = None) -> list[dict[str, Any]]:
    """Read published, visible posts; when IDs are supplied, read only that QA corpus."""
    conn = pymysql.connect(
        host=settings.mysql_host,
        port=settings.mysql_port,
        user=settings.mysql_user,
        password=settings.mysql_password,
        database=settings.mysql_database,
        charset="utf8mb4",
        cursorclass=DictCursor,
        connect_timeout=5,
    )
    try:
        sql = "SELECT id, title, summary, content FROM articles WHERE status='published' AND hidden=0 AND deleted_at IS NULL"
        params: tuple[Any, ...] = ()
        if article_ids:
            placeholders = ",".join(["%s"] * len(article_ids))
            sql += f" AND id IN ({placeholders})"
            params = tuple(article_ids)
        sql += " ORDER BY id"
        with conn.cursor() as cursor:
            cursor.execute(sql, params)
            return list(cursor.fetchall())
    finally:
        conn.close()
