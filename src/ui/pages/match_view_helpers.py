"""Helpers génériques pour la page Match View."""

from __future__ import annotations

import html
import logging
import os
import re
from collections.abc import Callable
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any

import polars as pl
import streamlit as st

from src.ui.i18n import t
from src.ui.pages.media_disk_index import index_media_dir

logger = logging.getLogger(__name__)

from src.config import get_repo_root
from src.ui import AppSettings

# =============================================================================
# Utilitaires de conversion date/heure
# =============================================================================


def to_paris_naive_local(dt_value, paris_tz) -> datetime | None:
    """Convertit une date en datetime naïf (sans tzinfo) en heure de Paris."""
    if dt_value is None:
        return None
    try:
        if isinstance(dt_value, datetime):
            ts = dt_value
        elif isinstance(dt_value, str):
            ts = datetime.fromisoformat(dt_value.replace("Z", "+00:00"))
        elif hasattr(dt_value, "to_pydatetime"):
            # pd.Timestamp / np.datetime64 (tolérance migration)
            ts = dt_value.to_pydatetime()
        else:
            ts = datetime.fromisoformat(str(dt_value).replace("Z", "+00:00"))
        if ts.tzinfo is None:
            return ts
        return ts.astimezone(paris_tz).replace(tzinfo=None)
    except Exception:
        return None


def safe_dt(v, paris_tz) -> datetime | None:
    """Alias pour to_paris_naive_local."""
    return to_paris_naive_local(v, paris_tz)


def match_time_window(
    row: dict[str, Any], *, tolerance_minutes: int, paris_tz
) -> tuple[datetime | None, datetime | None, bool]:
    """Calcule la fenêtre temporelle d'un match avec tolérance.

    Returns:
        Tuple (start_window, end_window, duration_known):
        - start_window: début de fenêtre (start_time - tolérance)
        - end_window: fin de fenêtre (end_time + tolérance)
        - duration_known: True si la durée réelle du match a été utilisée
    """
    start = safe_dt(row.get("start_time"), paris_tz)
    if start is None:
        return None, None, False

    dur_s = row.get("time_played_seconds")
    duration_known = False
    try:
        dur = float(dur_s) if dur_s == dur_s else None
    except Exception:
        dur = None

    if dur is not None and dur > 0:
        # Durée réelle du match disponible
        end = start + timedelta(seconds=float(dur))
        duration_known = True
    else:
        # Fallback: durée typique d'un match (~12 min au lieu de 30)
        end = start + timedelta(minutes=12)

    tol = max(0, int(tolerance_minutes))
    return start - timedelta(minutes=tol), end + timedelta(minutes=tol), duration_known


def paris_epoch_seconds_local(dt: datetime | None, paris_tz) -> float | None:
    """Convertit un datetime naïf Paris en epoch seconds."""
    if dt is None:
        return None
    try:
        aware = paris_tz.localize(dt) if dt.tzinfo is None else dt
        return aware.timestamp()
    except Exception:
        return None


def _filter_media_by_gamertag(df: pl.DataFrame, gamertag: str | None) -> pl.DataFrame:
    """Filtre les médias pour ne garder que ceux appartenant au joueur (chemin du fichier)."""
    if df.is_empty() or not gamertag:
        return df
    gt_lower = gamertag.lower()
    return df.filter(pl.col("path").str.to_lowercase().str.contains(gt_lower))


# =============================================================================
# Rendu de la section médias
# =============================================================================


def _show_media_group(
    df_group: pl.DataFrame,
    label: str,
    match_id: str,
) -> None:
    """Affiche un groupe de médias (images en grille + vidéos en selectbox)."""
    imgs = df_group.filter(pl.col("kind") == "image")
    vids = df_group.filter(pl.col("kind") == "video")
    if imgs.is_empty() and vids.is_empty():
        return

    st.caption(label)

    if not imgs.is_empty():
        n_cols = min(len(imgs), 4)
        cols = st.columns(n_cols)
        for i, r in enumerate(imgs.iter_rows(named=True)):
            with cols[i % n_cols]:
                fp = r.get("file_path")
                tp = r.get("thumbnail_path")
                display_path = (tp if tp and Path(tp).exists() else fp) if fp else None
                if display_path and Path(display_path).exists():
                    st.image(display_path)

    if not vids.is_empty():
        paths = [str(r["file_path"]) for r in vids.iter_rows(named=True) if r.get("file_path")]
        valid = [p for p in paths if Path(p).exists()]
        if valid:
            lv = [os.path.basename(p) for p in valid]
            key = f"media_vid_{label}_{match_id}"
            picked = st.selectbox(
                t("ml_video"),
                options=list(range(len(valid))),
                format_func=lambda i, _lv=lv: _lv[i],
                index=0,
                key=key,
                label_visibility="collapsed",
            )
            try:
                st.video(valid[int(picked)])
            except Exception:
                st.write(valid[int(picked)])


def _render_media_from_indexed_db(
    *,
    db_path: str,
    match_id: str,
    current_xuid: str | None,
) -> bool:
    """Tente d'afficher les médias depuis la BDD indexée (media_match_associations).

    Returns:
        True si des médias ont été trouvés et affichés, False sinon.
    """
    try:
        from src.data.media_indexer import MediaIndexer

        media_df = MediaIndexer.load_media_for_match(Path(str(db_path)), match_id, current_xuid)
        if media_df.is_empty():
            return False
    except Exception as e:
        logger.debug("_render_media_from_indexed_db: %s", e)
        return False

    mine = media_df.filter(pl.col("section") == "mine")
    teammates = media_df.filter(pl.col("section") == "teammate")

    st.subheader(t("mv_media_title"))

    if not mine.is_empty():
        _show_media_group(mine, t("mvh_my_captures"), match_id)

    if not teammates.is_empty():
        for gt in sorted(teammates["gamertag"].unique().to_list()):
            if not gt or not str(gt).strip():
                continue
            sub = teammates.filter(pl.col("gamertag") == gt)
            _show_media_group(sub, t("mvh_captures_of", gt=gt), match_id)

    return True


def _render_media_legacy(
    *,
    row: dict[str, Any],
    settings: AppSettings,
    format_datetime_fn: Callable[[datetime | None], str],
    paris_tz,
    gamertag: str | None,
) -> None:
    """Rendu legacy : scan de dossiers par fenêtre temporelle du match."""
    tol = settings.media_tolerance_minutes
    t0, t1, duration_known = match_time_window(row, tolerance_minutes=tol, paris_tz=paris_tz)
    if t0 is None or t1 is None:
        return

    screens_dir = settings.media_screens_dir.strip()
    videos_dir = settings.media_videos_dir.strip()

    if not screens_dir and not videos_dir:
        return

    try:
        t0_epoch = t0.timestamp() if t0 else None
        t1_epoch = t1.timestamp() if t1 else None
    except Exception:
        t0_epoch = t1_epoch = None

    if t0_epoch is None or t1_epoch is None:
        return

    img_hits = _scan_media_in_window(
        screens_dir, ("png", "jpg", "jpeg", "webp"), t0_epoch, t1_epoch, gamertag, 24
    )
    vid_hits = _scan_media_in_window(
        videos_dir, ("mp4", "webm", "mkv", "mov"), t0_epoch, t1_epoch, gamertag, 10
    )

    if img_hits is None and vid_hits is None:
        return

    st.subheader(t("mv_media_title"))
    window_info = t("mvh_window_label", start=format_datetime_fn(t0), end=format_datetime_fn(t1))
    if not duration_known:
        window_info += t("mvh_duration_estimated")
    st.caption(window_info)

    if img_hits is not None:
        st.caption(t("mvh_my_captures"))
        for p in img_hits["path"].to_list():
            try:
                st.image(p, caption=str(p))
            except Exception:
                st.write(str(p))

    if vid_hits is not None:
        _render_legacy_video_selector(vid_hits, row.get("match_id", ""))


def _scan_media_in_window(  # noqa: PLR0913
    directory: str,
    extensions: tuple[str, ...],
    t0_epoch: float,
    t1_epoch: float,
    gamertag: str | None,
    max_items: int,
) -> pl.DataFrame | None:
    """Scanne un dossier et filtre les fichiers dans la fenêtre temporelle."""
    if not directory or not os.path.isdir(directory):
        return None
    df = index_media_dir(directory, extensions)
    if df.is_empty():
        return None
    df = _filter_media_by_gamertag(df, gamertag)
    hits = df.filter((pl.col("mtime") >= t0_epoch) & (pl.col("mtime") <= t1_epoch)).head(max_items)
    return hits if not hits.is_empty() else None


def _render_legacy_video_selector(vid_hits: pl.DataFrame, match_id: str) -> None:
    """Affiche le sélecteur de vidéos legacy."""
    st.caption(t("mv_videos_title"))
    paths = [str(p) for p in vid_hits["path"].to_list() if p]
    if paths:
        labels = [os.path.basename(p) for p in paths]
        picked = st.selectbox(
            t("ml_video"),
            options=list(range(len(paths))),
            format_func=lambda i: labels[i],
            index=0,
            key=f"media_video_pick_{match_id}",
            label_visibility="collapsed",
        )
        p = paths[int(picked)]
        try:
            st.video(p)
            st.caption(str(p))
        except Exception:
            st.write(str(p))


def render_media_section(  # noqa: PLR0913
    *,
    row: dict[str, Any],
    settings: AppSettings,
    format_datetime_fn: Callable[[datetime | None], str],
    paris_tz,
    gamertag: str | None = None,
    db_path: str | None = None,
    current_xuid: str | None = None,
) -> None:
    """Rend la section médias pour un match.

    Utilise en priorité la BDD indexée (media_match_associations) pour afficher
    les captures de tous les joueurs avec indication du propriétaire.
    Si la BDD ne retourne rien, fallback sur le scan de dossiers (legacy).
    """
    if not settings.media_enabled:
        return

    match_id = str(row.get("match_id") or "").strip()

    if (
        db_path
        and match_id
        and _render_media_from_indexed_db(
            db_path=db_path,
            match_id=match_id,
            current_xuid=current_xuid,
        )
    ):
        return

    _render_media_legacy(
        row=row,
        settings=settings,
        format_datetime_fn=format_datetime_fn,
        paris_tz=paris_tz,
        gamertag=gamertag,
    )


# =============================================================================
# Composants UI
# =============================================================================


def _is_valid_css_color(val: str | None) -> bool:
    """Vérifie si une valeur est une couleur CSS valide (hex ou var())."""
    if not val:
        return False
    s = str(val).strip()
    return s.startswith("#") or s.startswith("var(")


def os_card(  # noqa: PLR0913
    title: str,
    kpi: str,
    sub_html: str | None = None,
    *,
    accent: str | None = None,
    kpi_color: str | None = None,
    kpi_is_html: bool = False,
    kpi_extra_style: str | None = None,
    sub_style: str | None = None,
    min_h: int = 112,
    kpi_font_size: str | None = None,
    title_font_size: str | None = None,
    show_title: bool = True,
    center_content: bool = False,
) -> None:
    """Rend une carte KPI avec style OpenSpartan."""
    t = html.escape(str(title or ""))
    k = str(kpi or "-") if kpi_is_html else html.escape(str(kpi or "-"))
    s = "" if not sub_html else str(sub_html)
    style = "min-height:" + str(int(min_h)) + "px; margin-bottom:10px;"
    if center_content:
        style += "display:flex;flex-direction:column;justify-content:center;align-items:center;text-align:center;"
    if accent and _is_valid_css_color(accent):
        # Surcharge la couleur de bordure via la variable CSS lue par ::before
        if str(accent).startswith("#"):
            style += f"--card-border-color:{accent}66;"
        else:
            style += f"--card-border-color:{accent};"
    _kpi_css_parts: list[str] = []
    if _is_valid_css_color(kpi_color):
        _kpi_css_parts.append(f"color:{kpi_color}")
    if kpi_font_size:
        _kpi_css_parts.append(f"font-size:{kpi_font_size}")
    if kpi_extra_style:
        _kpi_css_parts.append(kpi_extra_style)
    kpi_style = (" style='" + ";".join(_kpi_css_parts) + "'") if _kpi_css_parts else ""
    sub_style_attr = (
        "" if not sub_style else ' style="' + html.escape(str(sub_style), quote=True) + '"'
    )
    title_style_attr = (" style='font-size:" + title_font_size + "'") if title_font_size else ""
    title_html = f"<div class='os-card-title'{title_style_attr}>{t}</div>" if show_title else ""
    st.markdown(
        "<div class='os-card' style='"
        + style
        + "'>"
        + title_html
        + f"<div class='os-card-kpi'{kpi_style}>{k}</div>"
        + ("" if not s else f"<div class='os-card-sub'{sub_style_attr}>{s}</div>")
        + "</div>",
        unsafe_allow_html=True,
    )


def map_thumb_path(row: dict[str, Any], map_id: str | None) -> str | None:
    """Trouve le chemin vers la miniature de la carte."""

    def _safe_stem_from_name(name: str | None) -> str:
        s = str(name or "").strip()
        if not s:
            return ""
        s = re.sub(r'[<>:"/\\|?*]', " ", s)
        s = re.sub(r"[\x00-\x1f]", " ", s)
        s = re.sub(r"\s+", " ", s).strip()
        return s

    repo = Path(get_repo_root(__file__))
    base_dirs = [
        repo / "static" / "maps" / "thumbs",
        repo / "static" / "maps",  # Images présentes ici
        repo / "thumbs",
    ]

    candidates: list[str] = []
    mid = str(map_id or "").strip()
    if mid and mid != "-":
        candidates.append(mid)

    safe_name = _safe_stem_from_name(row.get("map_name"))
    if safe_name:
        candidates.append(safe_name)
        candidates.append(safe_name.replace(" ", "_"))

    uniq: list[str] = []
    seen: set[str] = set()
    for c in candidates:
        if c and c not in seen:
            uniq.append(c)
            seen.add(c)

    for base in base_dirs:
        for stem in uniq:
            for ext in (".jpg", ".jpeg", ".png", ".webp"):
                p = base / f"{stem}{ext}"
                if p.exists():
                    return str(p)
    return None


# =============================================================================
# Exports publics
# =============================================================================

__all__ = [
    "to_paris_naive_local",
    "safe_dt",
    "match_time_window",
    "paris_epoch_seconds_local",
    "index_media_dir",
    "render_media_section",
    "os_card",
    "map_thumb_path",
]
