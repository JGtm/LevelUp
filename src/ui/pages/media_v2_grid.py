"""Grille et lightbox V2 pour la page Médias."""

from __future__ import annotations

import hashlib
import logging
from pathlib import Path

import polars as pl
import streamlit as st
import streamlit.components.v1 as components

from src.ui.components.media_thumbnail import load_native_thumbnail_source
from src.ui.i18n import t
from src.ui.pages.media_v2_filters import MediaFilterState
from src.ui.pages.media_v2_likes import HEART_BUTTON_BASE_CSS, render_media_like_button

logger = logging.getLogger(__name__)


# ── Lightbox ─────────────────────────────────────────────────────────────────

_NEXT_BTN = "\u25b6"  # ▶ — doit correspondre exactement au label du bouton lb_next
_LIGHTBOX_STATE_KEY = "_lb_state"


def _build_header_meta(row: dict) -> tuple[str, str]:
    """Retourne (meta, date_long) pour l'en-tête de la lightbox."""
    map_name = row.get("map_ui") or row.get("map_name") or "—"
    mode_name = row.get("mode_ui") or row.get("pair_name") or "—"
    return f"{map_name} · {mode_name}", _fmt_date_long(row.get("capture_end_utc"))


def _inject_autoadvance_js() -> None:
    """Injecte du JS pour avancer automatiquement au média suivant en fin de vidéo."""
    components.html(
        f"""<script>
(function waitForVideo() {{
    var doc = window.parent.document;
    var dlg = doc.querySelector('[data-testid="stDialog"]');
    if (!dlg) {{ setTimeout(waitForVideo, 250); return; }}
    var vid = dlg.querySelector('video');
    if (!vid) {{ setTimeout(waitForVideo, 250); return; }}
    vid.addEventListener('ended', function onEnded() {{
        vid.removeEventListener('ended', onEnded);
        var btns = dlg.querySelectorAll('button');
        for (var i = 0; i < btns.length; i++) {{
            if (btns[i].textContent.trim() === '{_NEXT_BTN}') {{ btns[i].click(); return; }}
        }}
    }});
}})();
</script>""",
        height=0,
    )


def _queue_lightbox_open(rows: list[dict], idx: int) -> None:
    """Enregistre l'état lightbox avant le rerun naturel du clic."""
    st.session_state[_LIGHTBOX_STATE_KEY] = {"rows": rows, "idx": max(0, int(idx))}


def _set_lightbox_index(idx: int) -> None:
    """Met à jour l'index courant de la lightbox en le bornant."""
    lb = st.session_state.get(_LIGHTBOX_STATE_KEY)
    if not isinstance(lb, dict):
        return
    rows = lb.get("rows") or []
    if not rows:
        st.session_state.pop(_LIGHTBOX_STATE_KEY, None)
        return
    bounded_idx = max(0, min(int(idx), len(rows) - 1))
    st.session_state[_LIGHTBOX_STATE_KEY] = {**lb, "idx": bounded_idx}


def _clear_lightbox_state() -> None:
    """Ferme la lightbox en supprimant son état persistant."""
    st.session_state.pop(_LIGHTBOX_STATE_KEY, None)


def _queue_match_navigation(match_id: str) -> None:
    """Prépare la navigation vers Explorer avant le rerun naturel du clic."""
    mid = str(match_id or "").strip()
    if not mid:
        return
    st.session_state["_pending_page"] = "Match"
    st.session_state["_pending_match_id"] = mid


def _get_lightbox_state() -> tuple[list[dict], int] | None:
    """Retourne l'état courant de la lightbox, déjà borné."""
    lb = st.session_state.get(_LIGHTBOX_STATE_KEY)
    if not isinstance(lb, dict):
        return None
    rows: list[dict] = lb.get("rows") or []
    if not rows:
        _clear_lightbox_state()
        return None
    idx = max(0, min(int(lb.get("idx", 0) or 0), len(rows) - 1))
    return rows, idx


def _render_lightbox_nav_button(
    *,
    symbol: str,
    disabled: bool,
    key: str,
    target_idx: int,
    current_idx: int,
) -> None:
    """Rend un bouton prev/next de la lightbox avec fallback pour les tests."""
    clicked = st.button(
        symbol,
        disabled=disabled,
        key=key,
        width="stretch",
        on_click=_set_lightbox_index,
        args=(target_idx,),
    )
    if clicked and (st.session_state.get(_LIGHTBOX_STATE_KEY) or {}).get("idx") == current_idx:
        _set_lightbox_index(target_idx)


def _render_lightbox_dialog_body() -> None:
    """Rend le contenu du dialog lightbox à partir du session_state courant."""
    state = _get_lightbox_state()
    if state is None:
        return
    rows, idx = state
    n = len(rows)
    row = rows[idx]
    kind = row.get("kind") or "image"
    file_path = row.get("file_path") or ""
    thumb = row.get("thumbnail_path") or ""
    display_path = _resolve_display_path(file_path, thumb, kind)
    meta, date_long = _build_header_meta(row)
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
    counter = f"{idx + 1} / {n}" if has_nav else ""
    st.caption(f"{meta} · {date_long}  {counter}")

    if has_nav:
        prev_c, media_c, next_c = st.columns([1, 14, 1])
    else:
        media_c = st.container()

    autoadvance: bool = st.session_state.get("mv2_autoplay", True)
    with media_c:
        if display_path:
            if kind == "video":
                st.video(str(display_path), autoplay=True)
                if autoadvance and idx < n - 1:
                    _inject_autoadvance_js()
            else:
                st.image(str(display_path), width="stretch")
        else:
            st.warning(t("media_file_missing", file_name=row.get("file_name", "?")))

    if not has_nav:
        return

    with prev_c:
        st.markdown("<div class='lb-nav'>", unsafe_allow_html=True)
        _render_lightbox_nav_button(
            symbol="◀",
            disabled=(idx == 0),
            key="lb_prev",
            target_idx=idx - 1,
            current_idx=idx,
        )
        st.markdown("</div>", unsafe_allow_html=True)
    with next_c:
        st.markdown("<div class='lb-nav'>", unsafe_allow_html=True)
        _render_lightbox_nav_button(
            symbol="▶",
            disabled=(idx == n - 1),
            key="lb_next",
            target_idx=idx + 1,
            current_idx=idx,
        )
        st.markdown("</div>", unsafe_allow_html=True)


def render_lightbox_if_pending() -> None:
    """Affiche le dialog lightbox si un état est en attente en session_state.

    Doit être appelée AVANT tout autre rendu st.* dans la page.
    État attendu : st.session_state["_lb_state"] = {"rows": list[dict], "idx": int}
    """
    if _get_lightbox_state() is None:
        return

    @st.dialog(" ", width="large", on_dismiss=_clear_lightbox_state)
    def _show() -> None:
        _render_lightbox_dialog_body()

    _show()


# ── Grille ───────────────────────────────────────────────────────────────────

_HEADER_LABEL_CSS = (
    "display:flex;align-items:center;min-height:42px;"
    "white-space:nowrap;overflow:hidden;text-overflow:ellipsis;"
    "font-size:0.95rem;line-height:1;color:#98a2b3;"
)


def render_media_grid(df: pl.DataFrame, state: MediaFilterState) -> None:
    """Affiche la grille de cartes média (label + thumbnail + boutons)."""
    if df.is_empty():
        return
    rows = df.to_dicts()

    st.markdown(HEART_BUTTON_BASE_CSS, unsafe_allow_html=True)
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
                file_path = row.get("file_path") or ""
                lbl_col, heart_col = st.columns(
                    [14, 2.8],
                    gap="small",
                    vertical_alignment="center",
                )
                lbl_col.markdown(
                    f"<div style='{_HEADER_LABEL_CSS}'>{_type_badge(row)}&nbsp;&nbsp;{_format_label(row)}</div>",
                    unsafe_allow_html=True,
                )
                with heart_col:
                    render_media_like_button(
                        file_path,
                        force_full_rerun=state.group_by == "liked",
                    )
                _render_thumb(row)

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
                    lb_clicked = st.button(
                        t("media_view_full"),
                        key=f"mv2_lb_{lb_key}",
                        width="stretch",
                        on_click=_queue_lightbox_open,
                        args=(rows, i + j),
                    )
                    if (
                        lb_clicked
                        and (st.session_state.get(_LIGHTBOX_STATE_KEY) or {}).get("idx") != i + j
                    ):
                        _queue_lightbox_open(rows, i + j)

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


def _render_thumb(row: dict) -> None:
    """Affiche le thumbnail d'une carte via un rendu natif Streamlit sans iframe."""
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
        native_source = load_native_thumbnail_source(
            Path(static_path),
            hover_path=Path(hover_path) if hover_path else None,
        )
        if native_source is None:
            st.caption(t("media_file_missing", file_name=file_name))
            return
        try:
            st.image(native_source, width="stretch")
        except Exception:
            logger.debug("Thumbnail natif impossible pour %s", file_name, exc_info=True)
            st.caption(t("media_file_missing", file_name=file_name))
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
    mid = str(match_id or "").strip()
    if not mid:
        return
    clicked = st.button(
        t("ml_open_match"),
        key=f"mv2_match_{mid}_{suffix}",
        width="stretch",
        on_click=_queue_match_navigation,
        args=(mid,),
    )
    if clicked and (
        st.session_state.get("_pending_page") != "Match"
        or st.session_state.get("_pending_match_id") != mid
    ):
        _queue_match_navigation(mid)
