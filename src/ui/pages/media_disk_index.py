"""Indexation disque des médias pour les fallbacks UI."""

from __future__ import annotations

import os

import polars as pl
import streamlit as st

from src.data.media_helpers import is_generated_thumbnail_path


@st.cache_data(show_spinner=False, ttl=600)
def index_media_dir(dir_path: str, exts: tuple[str, ...]) -> pl.DataFrame:
    """Indexe un répertoire de médias par extension et date de modification."""
    empty = pl.DataFrame(schema={"path": pl.Utf8, "mtime": pl.Float64, "ext": pl.Utf8})
    rows: list[dict[str, object]] = []
    path_str = str(dir_path or "").strip()
    if not path_str or not os.path.isdir(path_str):
        return empty

    wanted = {e.lower().lstrip(".") for e in (exts or ()) if isinstance(e, str) and e.strip()}
    if not wanted:
        return empty

    max_files = 12000
    try:
        for root, dirs, files in os.walk(path_str):
            dirs[:] = [dirname for dirname in dirs if dirname.lower() != "thumbs"]
            for filename in files:
                if len(rows) >= max_files:
                    break
                ext = os.path.splitext(filename)[1].lower().lstrip(".")
                if ext not in wanted:
                    continue
                full_path = os.path.join(root, filename)
                if is_generated_thumbnail_path(full_path):
                    continue
                try:
                    stat_result = os.stat(full_path)
                except Exception:
                    continue
                rows.append({"path": full_path, "mtime": float(stat_result.st_mtime), "ext": ext})
            if len(rows) >= max_files:
                break
    except Exception:
        return empty

    df = pl.DataFrame(rows)
    if df.is_empty():
        return df
    return df.sort("mtime", descending=True)
