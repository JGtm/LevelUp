"""Grille et lightbox V2 pour la page Médias."""

from __future__ import annotations

import hashlib
import logging
from pathlib import Path

import polars as pl
import streamlit as st

from src.ui.components.media_thumbnail import render_media_thumbnail
from src.ui.i18n import t
from src.ui.pages.media_v2_filters import MediaFilterState

logger = logging.getLogger(__name__)

# Dimensions de référence pour les thumbnails (ratio 16:9)
_THUMB_W: int = 320
_THUMB_H: int = 180
# Largeur estimée de la zone principale Streamlit (hors sidebar)
_MAIN_W: int = 960


# ── Lightbox ─────────────────────────────────────────────────────────────────


def render_lightbox_if_pending() -> None:
    """Affiche le dialog lightbox si un état est en attente en session_state.

    Doit être appelée AVANT tout autre rendu st.* dans la page.
    État attendu : st.session_state["_lb_state"] = {"rows": list[dict], "idx": int}
    """
    lb: dict | None = st.session_state.pop("_lb_state", None)
    if not lb:
        return

    rows: list[dict] = lb["rows"]
    idx: int = max(0, min(lb["idx"], len(rows) - 1))
    n = len(rows)
    row = rows[idx]

    kind = row.get("kind") or "image"
    file_path = row.get("file_path") or ""
    thumb = row.get("thumbnail_path") or ""
    display_path = _resolve_display_path(file_path, thumb, kind)

    map_name = row.get("map_ui") or row.get("map_name") or "—"
    mode_name = row.get("mode_ui") or row.get("pair_name") or "—"
    meta = f"{map_name} · {mode_name}"
    date_long = _fmt_date_long(row.get("capture_end_utc"))

    @st.dialog(" ", width="large")
    def _show() -> None:
        st.markdown(
            "<style>"
            "[data-testid='stDialog'] h2, [data-testid='stDialog'] [data-testid='stHeadingWithActionElements']"
            " { display:none !important; }"
            "[data-testid='stDialog'] [data-testid='stVerticalBlockBorderWrapper']"
            " { max-height:82vh; overflow:hidden; }"
            "[data-testid='stDialog'] img, [data-testid='stDialog'] video"
            " { max-width:100%; width:100%; height:auto; max-height:72vh; object-fit:contain; }"
            ".lb-nav button { height:100% !important; min-height:180px; }"
            "</style>",
            unsafe_allow_html=True,
        )
        has_nav = n > 1
        # En-tête : label + compteur
        counter = f"{idx + 1} / {n}" if has_nav else ""
        header = f"{meta} · {date_long}  {counter}"
        st.caption(header)

        if has_nav:
            prev_c, media_c, next_c = st.columns([1, 14, 1])
        else:
            media_c = st.container()

        with media_c:
            if display_path:
                if kind == "video":
                    st.video(str(display_path))
                else:
                    st.image(str(display_path), width="stretch")
            else:
                st.warning(t("media_file_missing", file_name=row.get("file_name", "?")))

        if has_nav:
            with prev_c:
                st.markdown("<div class='lb-nav'>", unsafe_allow_html=True)
                if st.button("◀", disabled=(idx == 0), key="lb_prev", use_container_width=True):
                    st.session_state["_lb_state"] = {**lb, "idx": idx - 1}
                    st.rerun()
                st.markdown("</div>", unsafe_allow_html=True)
            with next_c:
                st.markdown("<div class='lb-nav'>", unsafe_allow_html=True)
                if st.button("▶", disabled=(idx == n - 1), key="lb_next", use_container_width=True):
                    st.session_state["_lb_state"] = {**lb, "idx": idx + 1}
                    st.rerun()
                st.markdown("</div>", unsafe_allow_html=True)

    _show()


# ── Grille ───────────────────────────────────────────────────────────────────


def render_media_grid(df: pl.DataFrame, state: MediaFilterState) -> None:
    """Affiche la grille de cartes média (label + thumbnail + boutons)."""
    if df.is_empty():
        return
    col_w = _MAIN_W / state.cols_per_row  # noqa: F841  (conservé pour référence future)
    iframe_h = _THUMB_H + 24  # marge suffisante pour le container iframe Streamlit
    rows = df.to_dicts()

    st.markdown(
        "<style>div[data-testid='stButton'] p { font-size:12px !important; }</style>",
        unsafe_allow_html=True,
    )

    for i in range(0, len(rows), state.cols_per_row):
        chunk = rows[i : i + state.cols_per_row]
        n = len(chunk)
        # On crée exactement n colonnes pour que les thumbnails gardent la même taille
        # que dans une rangée pleine (col_w et iframe_h calculés sur cols_per_row)
        th_cols = st.columns(n)
        for j, col in enumerate(th_cols):
            with col:
                row = chunk[j]
                st.caption(f"{_type_badge(row)}  {_format_label(row)}")
                _render_thumb(row, iframe_h)

        # Rangée B : [Agrandir] [Match] — 2 boutons par item, alignés sur les n colonnes
        btn_cols = st.columns([1] * (n * 2))
        for j in range(n):
            row = chunk[j]
            file_path = row.get("file_path")
            thumb = row.get("thumbnail_path") or ""
            kind = row.get("kind") or "image"
            match_id = row.get("match_id")

            with btn_cols[j * 2]:
                if _resolve_display_path(file_path, thumb, kind):
                    lb_key = hashlib.md5(f"lb_{file_path}_{i}_{j}".encode()).hexdigest()[:12]
                    if st.button(t("media_view_full"), key=f"mv2_lb_{lb_key}", width="stretch"):
                        st.session_state["_lb_state"] = {"rows": rows, "idx": i + j}
                        st.rerun()

            with btn_cols[j * 2 + 1]:
                if match_id and str(match_id).strip():
                    match_key = hashlib.md5(f"match_{file_path}_{i}_{j}".encode()).hexdigest()[:12]
                    _open_match_button(str(match_id).strip(), match_key)


# ── Helpers privés ────────────────────────────────────────────────────────────


def _fmt_session_label(raw: str) -> str:
    """Reformate 'DD/MM/YYYY HH:MM–HH:MM (N)' → 'DD/MM/YY · HH:MM–HH:MM · N match(s)'."""
    import re

    m = re.match(r"(\d{2})/(\d{2})/(\d{4})\s+(.+?)\s+\((\d+)\)", raw.strip())
    if not m:
        return raw
    d, mo, y, times, n = m.groups()
    word = "match" if n == "1" else "matchs"
    return f"{d}/{mo}/{y[2:]} · {times} · {n} {word}"


def _fmt_date(ts: object) -> str:
    """Formate un timestamp en JJ/MM/AA."""
    if ts is None:
        return "—"
    try:
        if hasattr(ts, "strftime"):
            return ts.strftime("%d/%m/%y")
        s = str(ts)[:10]
        parts = s.split("-")
        if len(parts) == 3:
            return f"{parts[2]}/{parts[1]}/{parts[0][2:]}"
        return s
    except Exception:
        return str(ts)[:10]


_JOURS_FR = ["lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi", "dimanche"]
_MOIS_FR = [
    "",
    "janvier",
    "février",
    "mars",
    "avril",
    "mai",
    "juin",
    "juillet",
    "août",
    "septembre",
    "octobre",
    "novembre",
    "décembre",
]


def _fmt_date_long(ts: object) -> str:
    """Formate un timestamp en date longue : vendredi 12 mars 2026."""
    if ts is None:
        return "—"
    try:
        import datetime

        if hasattr(ts, "year"):
            dt = ts
        else:
            s = str(ts)[:19].replace("T", " ")
            dt = datetime.datetime.strptime(s, "%Y-%m-%d %H:%M:%S")
        return f"{_JOURS_FR[dt.weekday()]} {dt.day} {_MOIS_FR[dt.month]} {dt.year}"
    except Exception:
        return _fmt_date(ts)


def _format_label(row: dict) -> str:
    """Construit le label normalisé : Carte · Mode · JJ/MM/AA (format court pour la grille)."""
    carte = row.get("map_ui") or row.get("map_name") or "—"
    mode = row.get("mode_ui") or row.get("pair_name") or "—"
    date = _fmt_date(row.get("capture_end_utc"))
    return f"{carte} · {mode} · {date}"


def _type_badge(row: dict) -> str:
    """Retourne le badge de type : 📷 ou 🎬 + durée."""
    if row.get("kind") == "video":
        dur = row.get("duration_seconds")
        if dur:
            mins, secs = divmod(int(dur), 60)
            return f"🎬 {mins}:{secs:02d}"
        return "🎬"
    return "📷"


def _render_thumb(row: dict, iframe_h: int) -> None:
    """Affiche le thumbnail d'une carte (statique + survol GIF pour les vidéos)."""
    file_path = row.get("file_path")
    thumbnail_path = row.get("thumbnail_path") or ""
    kind = row.get("kind") or "image"
    file_name = row.get("file_name") or ""

    thumb_ok = bool(thumbnail_path) and Path(thumbnail_path).exists()
    static_path = thumbnail_path if thumb_ok else file_path

    if static_path and Path(static_path).exists():
        hover_path = None
        if kind == "video" and thumb_ok and str(thumbnail_path).lower().endswith(".gif"):
            hover_path = thumbnail_path
        render_media_thumbnail(
            static_path=Path(static_path),
            hover_path=Path(hover_path) if hover_path else None,
            full_media_path=Path(file_path) if file_path else None,
            kind=kind,
            width=_THUMB_W,
            height=_THUMB_H,
            media_id=file_path,
            height_iframe=iframe_h,
        )
    else:
        st.caption(t("media_file_missing", file_name=file_name))


def _resolve_display_path(
    file_path: str | None,
    thumbnail_path: str,
    kind: str,  # noqa: ARG001
) -> str | None:
    """Retourne le chemin à afficher : fichier original > thumbnail > None."""
    if file_path and Path(file_path).exists():
        return file_path
    if thumbnail_path and Path(thumbnail_path).exists():
        return thumbnail_path
    return None


def _open_match_button(match_id: str, suffix: str) -> None:
    """Bouton de navigation vers la page Match."""
    if st.button(t("ml_open_match"), key=f"mv2_match_{match_id}_{suffix}", width="stretch"):
        st.session_state["_pending_page"] = "Match"
        st.session_state["_pending_match_id"] = match_id
        st.rerun()
