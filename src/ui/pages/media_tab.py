"""Page Médias – grille par sections, carte + date, bouton Ouvrir le match (Sprint 5)."""

from __future__ import annotations

import hashlib
import logging
from pathlib import Path

import polars as pl
import streamlit as st

from src.data.media_indexer import MediaIndexer
from src.ui.components.media_thumbnail import render_media_thumbnail
from src.ui.i18n import t
from src.ui.pages.media_library_render import open_match_button
from src.ui.settings import AppSettings

logger = logging.getLogger(__name__)


def _format_short_date(ts) -> str:
    """Formate un timestamp en date courte (ex. 07/02/26)."""
    if ts is None:
        return ""
    try:
        if hasattr(ts, "strftime"):
            return ts.strftime("%d/%m/%y")
        s = str(ts)
        if " " in s:
            return s.split(" ")[0].replace("-", "/")[-8:]  # yy/mm/dd → dd/mm/yy
        return s[:10] if len(s) >= 10 else s
    except Exception:
        return str(ts)[:10]


# Ratio 16:9 pour toutes les cartes (grille alignée)
MEDIA_THUMB_RATIO_W = 320
MEDIA_THUMB_RATIO_H = 180  # 320 * 9 / 16
MEDIA_THUMB_IFRAME_H = MEDIA_THUMB_RATIO_H + 4


# Largeur estimée de la zone principale Streamlit (hors sidebar) en pixels
_STREAMLIT_MAIN_W = 960


def _render_media_grid(
    df: pl.DataFrame,
    *,
    cols_per_row: int = 4,
    thumb_width: int = 200,
    thumb_height: int = 200,
) -> None:
    """Affiche une grille de cartes média (carte + date, thumbnail, boutons). Ratio 16:9."""
    if df.is_empty():
        return
    thumb_width = max(thumb_width, MEDIA_THUMB_RATIO_W)
    thumb_height = max(thumb_height, MEDIA_THUMB_RATIO_H)
    col_w = _STREAMLIT_MAIN_W / cols_per_row
    iframe_h = int(col_w * 9 / 16) + 12
    rows = df.to_dicts()
    for i in range(0, len(rows), cols_per_row):
        chunk = rows[i : i + cols_per_row]

        # ── Rangée A : labels + thumbnails ────────────────────────────────
        th_cols = st.columns(cols_per_row)
        for j, col in enumerate(th_cols):
            with col:
                if j >= len(chunk):
                    continue
                row = chunk[j]
                file_path = row.get("file_path")
                file_name = row.get("file_name") or ""
                map_name = row.get("map_ui") or row.get("map_name")
                capture_end = row.get("capture_end_utc")
                kind = row.get("kind") or "image"
                thumbnail_path = row.get("thumbnail_path")
                label = (map_name or "—") + " · " + _format_short_date(capture_end)
                st.caption(label)
                thumb_ok = thumbnail_path and Path(thumbnail_path).exists()
                static_path = thumbnail_path if thumb_ok else file_path
                if static_path and Path(static_path).exists():
                    hover_path = None
                    if (
                        kind == "video"
                        and thumb_ok
                        and str(thumbnail_path).lower().endswith(".gif")
                    ):
                        hover_path = thumbnail_path
                    render_media_thumbnail(
                        static_path=Path(static_path),
                        hover_path=Path(hover_path) if hover_path else None,
                        full_media_path=Path(file_path) if file_path else None,
                        kind=kind,
                        width=thumb_width,
                        height=thumb_height,
                        media_id=file_path,
                        height_iframe=iframe_h,
                    )
                else:
                    st.caption(t("media_file_missing", file_name=file_name))

        # ── Rangée B : boutons côte à côte (colonnes directes, pas de nesting) ──
        # 2 colonnes par carte : [Agrandir][Ouvrir le match]
        btn_cols = st.columns([1] * (cols_per_row * 2))
        for j in range(len(chunk)):
            row = chunk[j]
            file_path = row.get("file_path")
            match_id = row.get("match_id")
            kind = row.get("kind") or "image"
            th = row.get("thumbnail_path")
            static_path = th if (th and Path(th).exists()) else file_path

            with btn_cols[j * 2]:
                lightbox_path, lightbox_kind = _resolve_lightbox_path(file_path, th, kind)
                if lightbox_path:
                    lb_key = hashlib.md5(f"lightbox_{file_path}_{i}_{j}".encode()).hexdigest()[:16]
                    if st.button(
                        t("media_view_full"),
                        key=f"media_lb_{lb_key}",
                        width="stretch",
                    ):
                        st.session_state["_lightbox_media_path"] = lightbox_path
                        st.session_state["_lightbox_media_kind"] = lightbox_kind
                        st.rerun()

            with btn_cols[j * 2 + 1]:
                if match_id and str(match_id).strip():
                    open_match_button(str(match_id).strip(), unique_suffix=f"{i}_{j}")


def _resolve_lightbox_path(
    file_path: str | None, thumbnail_path: str | None, kind: str
) -> tuple[str | None, str]:
    """Retourne (chemin_lightbox, kind_lightbox) : vidéo originale > thumbnail > None."""
    if file_path and Path(file_path).exists():
        return file_path, kind
    if thumbnail_path and Path(thumbnail_path).exists():
        return thumbnail_path, "image"
    return None, kind


_ENRICH_COLS = ["match_id", "outcome", "pair_name", "mode_ui", "map_ui", "is_with_friends"]

# Clés i18n des options de tri → colonne polars correspondante
_SORT_OPTIONS_KEYS = [
    "media_sort_date_capture",
    "media_sort_map",
    "media_sort_mode",
    "media_sort_outcome",
    "media_sort_owner",
]
_SORT_COL_MAP: dict[str, str] = {
    "media_sort_date_capture": "capture_end_utc",
    "media_sort_map": "map_ui",
    "media_sort_mode": "mode_ui",
    "media_sort_outcome": "outcome",
    "media_sort_owner": "section",
}

# Mapping section interne → clé i18n du label
_SECTION_I18N: dict[str, str] = {
    "mine": "media_owner_mine",
    "teammate": "media_owner_teammate",
    "unassigned": "media_owner_unassigned",
}


def _enrich_media_with_match_data(
    media_df: pl.DataFrame,
    df_full: pl.DataFrame | None,
) -> pl.DataFrame:
    """Enrichit media_df avec outcome/mode/carte/is_with_friends depuis df_full."""
    if df_full is None or df_full.is_empty():
        logger.debug(
            "_enrich_media_with_match_data: df_full absent ou vide — enrichissement ignoré"
        )
        return media_df
    if "match_id" not in media_df.columns or "match_id" not in df_full.columns:
        logger.debug(
            "_enrich_media_with_match_data: colonne match_id manquante — enrichissement ignoré"
        )
        return media_df
    available = [c for c in _ENRICH_COLS if c in df_full.columns]
    if len(available) <= 1:
        logger.debug(
            "_enrich_media_with_match_data: aucune colonne enrichissable disponible dans df_full"
        )
        return media_df
    # Exclure les colonnes déjà présentes dans media_df (sauf match_id = clé de jointure)
    existing = set(media_df.columns) - {"match_id"}
    cols = ["match_id"] + [c for c in available[1:] if c not in existing]
    df_enrich = df_full.select(cols).unique(subset=["match_id"])
    result = media_df.join(df_enrich, on="match_id", how="left")
    logger.debug(
        "_enrich_media_with_match_data: %d médias enrichis avec colonnes %s",
        len(result),
        cols[1:],
    )
    return result


def _build_media_filter_ui(media_df: pl.DataFrame) -> dict:
    """Barre de filtres plate (1 ligne) — filtres gauche │ tri droite."""
    # Options dynamiques depuis les données
    map_col = "map_ui" if "map_ui" in media_df.columns else "map_name"
    all_maps: list[str] = (
        sorted({str(m) for m in media_df[map_col].drop_nulls().to_list() if m})
        if map_col in media_df.columns
        else []
    )

    mode_col = "mode_ui" if "mode_ui" in media_df.columns else "pair_name"
    all_modes: list[str] = (
        sorted({str(m) for m in media_df[mode_col].drop_nulls().to_list() if m})
        if mode_col in media_df.columns
        else []
    )

    sort_labels = [t(k) for k in _SORT_OPTIONS_KEYS]
    section_options = [t(_SECTION_I18N[s]) for s in ["mine", "teammate", "unassigned"]]
    all_label = t("media_filter_all")

    with st.container(border=True):
        st.markdown(
            f"<div class='v7-context-toolbar-label'>{t('exp_filters')}</div>",
            unsafe_allow_html=True,
        )

        # ── Barre unique ──────────────────────────────────────────────────────
        # [Proprio][Carte][Mode][Contexte] │ [Trier][Ordre]
        c1, c2, c3, c4, _s1, c6, c7 = st.columns([1.3, 1.7, 1.7, 1.1, 0.07, 1.7, 1.1])

        with c1:
            section_sel = st.selectbox(
                t("media_filter_owner"),
                options=[all_label] + section_options,
                index=0,
                key="mf_sections",
                label_visibility="collapsed",
            )
        with c2:
            map_sel = st.selectbox(
                t("media_filter_map"),
                options=[all_label] + all_maps,
                key="mf_map",
                label_visibility="collapsed",
            )
        with c3:
            mode_sel = st.selectbox(
                t("media_filter_mode"),
                options=[all_label] + all_modes,
                key="mf_mode",
                label_visibility="collapsed",
            )
        with c4:
            squad_opts = [t("media_squad_all"), t("media_squad_solo"), t("media_squad_squad")]
            squad_sel = st.selectbox(
                t("media_filter_squad"),
                options=squad_opts,
                index=0,
                key="mf_squad",
                label_visibility="collapsed",
            )
        _s1.markdown("<div class='v7-toolbar-divider'></div>", unsafe_allow_html=True)

        with c6:
            sort_label = st.selectbox(
                t("media_sort_by"),
                options=sort_labels,
                index=0,
                key="mf_sort",
                label_visibility="collapsed",
            )
        with c7:
            sort_order = st.selectbox(
                t("media_sort_order"),
                options=[t("media_sort_desc"), t("media_sort_asc")],
                index=0,
                key="mf_sort_order",
                label_visibility="collapsed",
            )
    cols_per_row = 5

    section_map_inv = {t(v): k for k, v in _SECTION_I18N.items()}
    selected_sections = (
        [section_map_inv[section_sel]]
        if section_sel != all_label and section_sel in section_map_inv
        else []
    )
    sort_key = _SORT_OPTIONS_KEYS[sort_labels.index(sort_label)]
    return {
        "sections": selected_sections,
        "map": map_sel if map_sel != all_label else None,
        "map_col": map_col,
        "mode": mode_sel if mode_sel != all_label else None,
        "mode_col": mode_col,
        "squad": squad_sel,
        "squad_solo": t("media_squad_solo"),
        "squad_squad": t("media_squad_squad"),
        "sort_col": _SORT_COL_MAP[sort_key],
        "sort_desc": sort_order == t("media_sort_desc"),
        "cols_per_row": cols_per_row,
    }


def _apply_media_filters(
    df: pl.DataFrame,
    filters: dict,
    *,
    apply_match_filters: bool = True,
) -> pl.DataFrame:
    """Applique filtres (kind, match) et tri sur un DataFrame médias."""
    initial_len = len(df)
    if apply_match_filters:
        if filters["map"] and filters["map_col"] in df.columns:
            df = df.filter(pl.col(filters["map_col"]) == filters["map"])
        if filters["mode"] and filters["mode_col"] in df.columns:
            df = df.filter(pl.col(filters["mode_col"]) == filters["mode"])
        if filters["squad"] == filters["squad_solo"] and "is_with_friends" in df.columns:
            df = df.filter(pl.col("is_with_friends").eq(False))
        elif filters["squad"] == filters["squad_squad"] and "is_with_friends" in df.columns:
            df = df.filter(pl.col("is_with_friends").eq(True))
    sort_col = filters["sort_col"] if filters["sort_col"] in df.columns else "capture_end_utc"
    if sort_col in df.columns:
        # Tri secondaire stable sur file_path pour tie-breaking déterministe
        df = df.sort(
            [sort_col, "file_path"], descending=[filters["sort_desc"], False], nulls_last=True
        )
    logger.debug(
        "_apply_media_filters: %d→%d médias (match_filters=%s, tri='%s' desc=%s)",
        initial_len,
        len(df),
        apply_match_filters,
        sort_col,
        filters["sort_desc"],
    )
    return df


def render_media_tab(  # noqa: C901, PLR0912, PLR0915
    *,
    df_full: pl.DataFrame | None = None,
    settings: AppSettings | None = None,
) -> None:
    """Rend la page Médias : sections Mes captures, Captures de XXX, Sans correspondance."""
    st.subheader(t("mv_media_title"))

    # Lightbox « Voir en grand » : traiter en premier pour que le dialog s'ouvre bien après rerun
    _path = st.session_state.pop("_lightbox_media_path", None)
    _kind = st.session_state.pop("_lightbox_media_kind", "image")
    if _path is not None:

        @st.dialog(t("media_dialog_title"), width="large")
        def _lightbox_dialog():
            # CSS : dialog contenu dans le viewport, sans scroll page
            st.markdown(
                "<style>"
                "[data-testid='stModal'] [data-testid='stVerticalBlockBorderWrapper'],"
                "[data-testid='stDialog'] [data-testid='stVerticalBlockBorderWrapper'] {"
                "  max-height: 80vh; overflow: hidden;"
                "}"
                "[data-testid='stModal'] > div, [data-testid='stDialog'] > div {"
                "  max-width: 92vw !important; width: 92vw !important;"
                "  padding: 0.5rem !important;"
                "  margin-left: auto !important; margin-right: auto !important;"
                "}"
                "[data-testid='stModal'] img, [data-testid='stModal'] video, "
                "[data-testid='stDialog'] img, [data-testid='stDialog'] video, "
                "[role='dialog'] img, [role='dialog'] video {"
                "  max-width: 100%; width: 100%; height: auto; object-fit: contain;"
                "  max-height: 75vh;"
                "}"
                "</style>",
                unsafe_allow_html=True,
            )
            if _kind == "video":
                st.video(str(_path))
            else:
                st.image(str(_path), width="stretch")

        _lightbox_dialog()

    if not settings:
        settings = AppSettings()
    if not getattr(settings, "media_enabled", True):
        st.info(t("media_disabled"))
        return

    db_path = st.session_state.get("db_path", "")
    xuid_input = st.session_state.get("xuid_input", "")

    try:
        from src.app.profile import get_identity_from_secrets, resolve_xuid
    except ImportError:
        resolve_xuid = None
        get_identity_from_secrets = None

    current_xuid = None
    if get_identity_from_secrets and resolve_xuid:
        identity = get_identity_from_secrets()
        current_xuid = (
            resolve_xuid(xuid_input or identity.gamertag or "", db_path, identity) or identity.xuid
        )

    if not db_path or not str(db_path).endswith(".duckdb"):
        st.info(t("no_data_filter"))
        return

    media_df = MediaIndexer.load_media_for_ui(Path(db_path), current_xuid)
    if media_df.is_empty():
        st.info(t("media_no_indexed"))
        return

    # Enrichir avec les données de match (outcome, mode, carte, is_with_friends)
    media_df = _enrich_media_with_match_data(media_df, df_full)
    filters = _build_media_filter_ui(media_df)
    cols_per_row = filters["cols_per_row"]

    # Splitter par section, appliquer filtres+tri, contrôler la visibilité
    show_all = not filters["sections"]
    mine = media_df.filter(pl.col("section") == "mine")
    teammate = media_df.filter(pl.col("section") == "teammate")
    unassigned = media_df.filter(pl.col("section") == "unassigned")

    if not mine.is_empty():
        mine = _apply_media_filters(
            mine.unique(subset=["file_path"], keep="first", maintain_order=True), filters
        )
    if not teammate.is_empty():
        teammate = _apply_media_filters(
            teammate.unique(subset=["file_path"], keep="first", maintain_order=True), filters
        )
    if not unassigned.is_empty():
        # Médias sans match : filtres type+nom seulement (pas de carte/mode/outcome)
        unassigned = _apply_media_filters(
            unassigned.unique(subset=["file_path"], keep="first", maintain_order=True),
            filters,
            apply_match_filters=False,
        )

    show_mine = show_all or "mine" in filters["sections"]
    show_teammate = show_all or "teammate" in filters["sections"]
    show_unassigned = show_all or "unassigned" in filters["sections"]

    # CSS : police des boutons à 12px (le texte Streamlit est dans un <p> à l'intérieur du btn)
    st.markdown(
        "<style>div[data-testid='stButton'] p { font-size: 12px !important; }</style>",
        unsafe_allow_html=True,
    )

    # Section « Mes captures » (ratio 16:9)
    if not mine.is_empty() and show_mine:
        st.markdown(f"### {t('media_my_captures')}")
        _render_media_grid(
            mine,
            cols_per_row=cols_per_row,
            thumb_width=MEDIA_THUMB_RATIO_W,
            thumb_height=MEDIA_THUMB_RATIO_H,
        )

    # Section « Captures de XXX » par gamertag
    if not teammate.is_empty() and show_teammate:
        for gamertag in teammate["gamertag"].unique(maintain_order=True).to_list():
            if not gamertag or (isinstance(gamertag, str) and not gamertag.strip()):
                continue
            st.markdown(f"### {t('media_captures_of', gamertag=gamertag)}")
            sub = teammate.filter(pl.col("gamertag") == gamertag).unique(
                subset=["file_path"], keep="first", maintain_order=True
            )
            _render_media_grid(
                sub,
                cols_per_row=cols_per_row,
                thumb_width=MEDIA_THUMB_RATIO_W,
                thumb_height=MEDIA_THUMB_RATIO_H,
            )

    # Section « Sans correspondance » (masquée si vide)
    if not unassigned.is_empty() and show_unassigned:
        st.markdown(f"### {t('media_unmatched')}")
        _render_media_grid(
            unassigned,
            cols_per_row=cols_per_row,
            thumb_width=MEDIA_THUMB_RATIO_W,
            thumb_height=MEDIA_THUMB_RATIO_H,
        )
