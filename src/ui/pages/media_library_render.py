"""Rendu grille et affichage groupé de la bibliothèque médias."""

from __future__ import annotations

import hashlib
import html
import os

import polars as pl
import streamlit as st

from src.ui.formatting import format_datetime_fr_hm
from src.ui.i18n import t
from src.visualization._compat import DataFrameLike, ensure_polars


def _build_app_url(page: str, **params: str) -> str:
    """Construit une URL intra-app avec query params."""
    import urllib.parse

    qp: dict[str, str] = {"page": str(page)}
    for k, v in params.items():
        s = str(v or "").strip()
        if s:
            qp[str(k)] = s
    return "?" + urllib.parse.urlencode(qp)


def open_match_button(match_id: str, *, unique_suffix: str | None = None) -> None:
    """Affiche un bouton pour ouvrir la page Match.

    Args:
        match_id: ID du match à ouvrir.
        unique_suffix: Suffixe optionnel pour rendre la clé unique.
    """
    mid = str(match_id or "").strip()
    if not mid:
        st.caption(t("media_unknown_match"))
        return

    button_key = f"open_match_{mid}_{unique_suffix}" if unique_suffix else f"open_match_{mid}"

    if st.button(t("ml_open_match"), key=button_key, width="stretch"):
        st.session_state["_pending_page"] = "Match"
        st.session_state["_pending_match_id"] = mid
        st.rerun()


def _placeholder_html(base: str, hint: str | None = None) -> str:
    """HTML du placeholder vidéo (sans charger la miniature)."""
    if hint is None:
        hint = t("ml_click_thumbnail")
    return (
        "<div style='padding:18px;border-radius:12px;"
        "border:1px solid rgba(255,255,255,0.12);'>"
        "<div style='font-size:34px;line-height:1'>🎬</div>"
        "<div style='opacity:0.85;margin-top:6px'>" + html.escape(base) + "</div>"
        "<div style='font-size:11px;opacity:0.6;margin-top:4px'>" + html.escape(hint) + "</div>"
        "</div>"
    )


def render_media_grid(
    items: DataFrameLike,
    *,
    cols_per_row: int,
    render_context: str = "default",
) -> None:
    """Affiche une grille de médias Streamlit."""
    if items is None:
        st.info(t("media_no_filter_result"))
        return
    items = ensure_polars(items)
    if items.is_empty():
        st.info(t("media_no_filter_result"))
        return

    cols_per_row = max(2, min(8, int(cols_per_row)))

    # Identifiant stable pour les clés session_state
    items = items.with_row_index("_stable_id")

    rows = items.to_dicts()
    for i in range(0, len(rows), cols_per_row):
        chunk = rows[i : i + cols_per_row]
        cols = st.columns(cols_per_row)
        for col_idx in range(cols_per_row):
            with cols[col_idx]:
                if col_idx < len(chunk):
                    _render_single_media(chunk[col_idx], render_context)


def _render_single_media(rec: dict, render_context: str) -> None:
    """Rend un seul élément média dans une cellule de grille."""
    path = str(rec.get("path") or "").strip()
    kind = str(rec.get("kind") or "")
    base = str(rec.get("basename") or os.path.basename(path))
    mid = rec.get("match_id")

    if kind == "image" and path:
        try:
            st.image(path, width="stretch")
        except Exception:
            st.caption(base)
    else:
        _render_video_cell(rec, path, base, mid, render_context)

    st.caption(base)

    # Bouton "Ouvrir le match" sauf dans un contexte de groupe
    if isinstance(mid, str) and mid.strip() and not render_context.startswith("match_"):
        stable_id = rec.get("_stable_id", 0)
        open_match_button(mid, unique_suffix=str(stable_id))
    elif not (isinstance(mid, str) and mid.strip()):
        st.caption(t("media_unassociated_match"))


def _render_video_cell(
    rec: dict, path: str, base: str, mid: str | None, render_context: str
) -> None:
    """Rend une cellule vidéo avec thumbnail et preview."""
    thumb_path = str(rec.get("thumbnail_path") or "").strip()
    path_hash = hashlib.md5(path.encode()).hexdigest()
    match_id_part = str(mid).strip() if isinstance(mid, str) and mid.strip() else "no_match"
    stable_id = rec.get("_stable_id", 0)
    thumb_key = f"thumb_show::{path_hash}::{match_id_part}" f"::{render_context}::{stable_id}"
    show_thumb = st.session_state.get(thumb_key, False)

    if show_thumb and thumb_path and os.path.exists(thumb_path):
        try:
            st.image(thumb_path, width="stretch")
        except Exception:
            st.markdown(_placeholder_html(base), unsafe_allow_html=True)
        if st.button(t("ml_hide_thumbnail"), key=thumb_key + "::btn"):
            st.session_state[thumb_key] = False
            st.rerun()
    else:
        st.markdown(_placeholder_html(base), unsafe_allow_html=True)
        if thumb_path and os.path.exists(thumb_path):
            if st.button(t("ml_show_thumbnail"), key=thumb_key + "::btn"):
                st.session_state[thumb_key] = True
                st.rerun()
        else:
            st.caption(t("media_no_thumbnail"))

    if path:
        preview_key = (
            f"media_preview::{path_hash}::{match_id_part}" f"::{render_context}::{stable_id}"
        )
        if st.button(t("ml_preview"), key=preview_key, width="stretch"):
            st.session_state[preview_key + "::open"] = True
        if st.session_state.get(preview_key + "::open"):
            try:
                st.video(path)
            except Exception:
                st.caption(path)


def render_grouped_media(
    assigned: pl.DataFrame,
    unassigned: pl.DataFrame,
    *,
    group_by_match: bool,
    show_unassigned: bool,
    cols_per_row: int,
) -> None:
    """Rend les médias groupés par match ou en grille plate."""
    if not group_by_match:
        # Combiner assigned + unassigned pour affichage plat
        parts = []
        if not assigned.is_empty():
            parts.append(assigned)
        if not unassigned.is_empty():
            parts.append(unassigned)
        if parts:
            all_media = pl.concat(parts, how="diagonal_relaxed")
            render_media_grid(all_media, cols_per_row=cols_per_row, render_context="all")
        return

    # Affichage groupé par match
    if not assigned.is_empty():
        assigned = assigned.with_columns(
            pl.col("match_start_time").cast(pl.Datetime, strict=False).alias("_match_sort")
        )
        assigned = assigned.sort(["_match_sort", "mtime"], descending=[True, False])

        for match_id, g in assigned.group_by("match_id", maintain_order=True):
            match_id_val = match_id[0] if isinstance(match_id, tuple) else match_id
            title_dt = None
            try:
                dt0 = g["match_start_time"][0]
                title_dt = format_datetime_fr_hm(dt0) if dt0 is not None else None
            except Exception:
                title_dt = None

            label = f"Match {match_id_val}" + (" — " + str(title_dt) if title_dt else "")
            with st.expander(label, expanded=False):
                open_match_button(str(match_id_val))
                g2 = g.sort("mtime", descending=False)
                g2 = g2.unique(subset=["path"], keep="first")
                render_media_grid(
                    g2,
                    cols_per_row=cols_per_row,
                    render_context=f"match_{match_id_val}",
                )

    if show_unassigned and not unassigned.is_empty():
        st.divider()
        st.subheader(t("media_unassociated"))
        render_media_grid(unassigned, cols_per_row=cols_per_row, render_context="unassigned")
