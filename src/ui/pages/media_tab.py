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
MEDIA_THUMB_IFRAME_H = MEDIA_THUMB_RATIO_H + 24


def _render_media_grid(
    df: pl.DataFrame,
    *,
    cols_per_row: int = 4,
    thumb_width: int = 200,
    thumb_height: int = 200,
) -> None:
    """Affiche une grille de cartes média (carte + date, thumbnail, bouton match). Ratio 16:9."""
    if df.is_empty():
        return
    thumb_width = max(thumb_width, MEDIA_THUMB_RATIO_W)
    thumb_height = max(thumb_height, MEDIA_THUMB_RATIO_H)
    rows = df.to_dicts()
    for i in range(0, len(rows), cols_per_row):
        chunk = rows[i : i + cols_per_row]
        cols = st.columns(cols_per_row)
        for j, col in enumerate(cols):
            with col:
                if j < len(chunk):
                    row = chunk[j]
                    file_path = row.get("file_path")
                    file_name = row.get("file_name") or ""
                    map_name = row.get("map_ui") or row.get("map_name")
                    capture_end = row.get("capture_end_utc")
                    match_id = row.get("match_id")
                    kind = row.get("kind") or "image"
                    thumbnail_path = row.get("thumbnail_path")
                    # Carte + date au-dessus du thumbnail
                    label = (map_name or "—") + " · " + _format_short_date(capture_end)
                    st.caption(label)
                    # Thumbnail : ratio 16:9
                    # Priorité : thumbnail GIF > fichier original
                    # Fallback sur file_path si le thumbnail n'existe pas (pas encore généré)
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
                            height_iframe=MEDIA_THUMB_IFRAME_H,
                        )
                    else:
                        st.caption(t("media_file_missing", file_name=file_name))
                    # Bouton « Voir en grand » (lightbox en dialog Streamlit, le clic dans l'iframe ne pouvant pas ouvrir en plein écran)
                    if static_path and Path(static_path).exists():
                        lb_key = hashlib.md5(f"lightbox_{file_path}_{i}_{j}".encode()).hexdigest()[
                            :16
                        ]
                        if st.button(
                            t("media_view_full"), key=f"media_lb_{lb_key}", width="stretch"
                        ):
                            st.session_state["_lightbox_media_path"] = file_path
                            st.session_state["_lightbox_media_kind"] = kind
                            st.rerun()
                    # Bouton « Ouvrir le match » (même onglet, joueur actif conservé)
                    if match_id and str(match_id).strip():
                        open_match_button(str(match_id).strip(), unique_suffix=f"{i}_{j}")


# ── Colonnes de df_full à joindre sur media_df pour enrichir les filtres ──────
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


def _build_media_filter_ui(media_df: pl.DataFrame) -> dict:  # noqa: PLR0912
    """Construit l'expander de filtres et retourne les valeurs sélectionnées."""
    from src.ui.i18n import get_lang, get_outcome_map

    lang = get_lang()
    outcome_map = get_outcome_map(lang)

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

    present_codes: set[int] = set()
    if "outcome" in media_df.columns:
        present_codes = {v for v in media_df["outcome"].drop_nulls().to_list() if v is not None}
    outcome_labels: list[str] = [
        label for code, label in outcome_map.items() if code in present_codes
    ]

    sort_labels = [t(k) for k in _SORT_OPTIONS_KEYS]
    section_options = [t(_SECTION_I18N[s]) for s in ["mine", "teammate", "unassigned"]]
    all_label = t("media_filter_all")

    with st.expander(t("media_filters"), expanded=False):
        # Rangée 1 – filtres match
        c1, c2, c3, c4 = st.columns(4)
        with c1:
            section_sel = st.multiselect(
                t("media_filter_owner"), options=section_options, default=[], key="mf_sections"
            )
        with c2:
            map_sel = st.selectbox(
                t("media_filter_map"), options=[all_label] + all_maps, key="mf_map"
            )
        with c3:
            mode_sel = st.selectbox(
                t("media_filter_mode"), options=[all_label] + all_modes, key="mf_mode"
            )
        with c4:
            outcome_sel = st.multiselect(
                t("media_filter_outcome"), options=outcome_labels, default=[], key="mf_outcomes"
            )

        # Rangée 2 – contexte, type, tri, affichage
        c5, c6, c7, c8, c9 = st.columns([1.2, 1.2, 1, 1.5, 1])
        with c5:
            squad_opts = [t("media_squad_all"), t("media_squad_solo"), t("media_squad_squad")]
            squad_sel = st.radio(
                t("media_filter_squad"),
                options=squad_opts,
                index=0,
                horizontal=True,
                key="mf_squad",
            )
        with c6:
            kinds = st.multiselect(
                t("media_type"),
                options=["image", "video"],
                default=["image", "video"],
                key="media_tab_kind",
            )
        with c7:
            name_filter = st.text_input(
                t("media_filename"), value="", placeholder="ex: 2026-01", key="media_tab_name"
            )
        with c8:
            sort_label = st.selectbox(
                t("media_sort_by"), options=sort_labels, index=0, key="mf_sort"
            )
            sort_desc = st.toggle(t("media_sort_desc"), value=True, key="mf_sort_desc")
        with c9:
            cols_per_row = st.slider(
                t("media_columns"), min_value=2, max_value=6, value=4, step=1, key="media_tab_cols"
            )

    section_map_inv = {t(v): k for k, v in _SECTION_I18N.items()}
    selected_sections = [section_map_inv[s] for s in section_sel if s in section_map_inv]
    sort_key = _SORT_OPTIONS_KEYS[sort_labels.index(sort_label)]
    code_by_label = {label: code for code, label in outcome_map.items()}
    return {
        "sections": selected_sections,
        "map": map_sel if map_sel != all_label else None,
        "map_col": map_col,
        "mode": mode_sel if mode_sel != all_label else None,
        "mode_col": mode_col,
        "outcome_codes": [code_by_label[o] for o in outcome_sel if o in code_by_label],
        "squad": squad_sel,
        "squad_solo": t("media_squad_solo"),
        "squad_squad": t("media_squad_squad"),
        "kinds": kinds,
        "name": name_filter,
        "sort_col": _SORT_COL_MAP[sort_key],
        "sort_desc": sort_desc,
        "cols_per_row": cols_per_row,
    }


def _apply_media_filters(
    df: pl.DataFrame,
    filters: dict,
    *,
    apply_match_filters: bool = True,
) -> pl.DataFrame:
    """Applique filtres (kind, nom, match) et tri sur un DataFrame médias."""
    initial_len = len(df)
    if filters["kinds"]:
        df = df.filter(pl.col("kind").is_in(filters["kinds"]))
    if filters["name"].strip():
        df = df.filter(
            pl.col("file_name").str.to_lowercase().str.contains(filters["name"].strip().lower())
        )
    if apply_match_filters:
        if filters["map"] and filters["map_col"] in df.columns:
            df = df.filter(pl.col(filters["map_col"]) == filters["map"])
        if filters["mode"] and filters["mode_col"] in df.columns:
            df = df.filter(pl.col(filters["mode_col"]) == filters["mode"])
        if filters["outcome_codes"] and "outcome" in df.columns:
            df = df.filter(pl.col("outcome").is_in(filters["outcome_codes"]))
        if filters["squad"] == filters["squad_solo"] and "is_with_friends" in df.columns:
            df = df.filter(pl.col("is_with_friends").eq(False))
        elif filters["squad"] == filters["squad_squad"] and "is_with_friends" in df.columns:
            df = df.filter(pl.col("is_with_friends").eq(True))
    sort_col = filters["sort_col"] if filters["sort_col"] in df.columns else "capture_end_utc"
    if sort_col in df.columns:
        df = df.sort(sort_col, descending=filters["sort_desc"], nulls_last=True)
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
    if _path is not None and Path(_path).exists():

        @st.dialog(t("media_dialog_title"), width="large")
        def _lightbox_dialog():
            # CSS pour maximiser la largeur sans débordement (cible le contenu du modal)
            st.markdown(
                "<style>"
                "[data-testid='stModal'] > div, [data-testid='stDialog'] > div {"
                "  max-width: 95vw !important; width: 95vw !important;"
                "}"
                "[data-testid='stModal'] img, [data-testid='stModal'] video, "
                "[data-testid='stDialog'] img, [data-testid='stDialog'] video, "
                "[role='dialog'] img, [role='dialog'] video {"
                "  max-width: 100%; width: 100%; height: auto; object-fit: contain;"
                "  max-height: 85vh;"
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
        mine = _apply_media_filters(mine.unique(subset=["file_path"], keep="first"), filters)
    if not teammate.is_empty():
        teammate = _apply_media_filters(teammate, filters)
    if not unassigned.is_empty():
        # Médias sans match : filtres type+nom seulement (pas de carte/mode/outcome)
        unassigned = _apply_media_filters(
            unassigned.unique(subset=["file_path"], keep="first"),
            filters,
            apply_match_filters=False,
        )

    show_mine = show_all or "mine" in filters["sections"]
    show_teammate = show_all or "teammate" in filters["sections"]
    show_unassigned = show_all or "unassigned" in filters["sections"]

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
        for gamertag in teammate["gamertag"].unique().to_list():
            if not gamertag or (isinstance(gamertag, str) and not gamertag.strip()):
                continue
            st.markdown(f"### {t('media_captures_of', gamertag=gamertag)}")
            sub = teammate.filter(pl.col("gamertag") == gamertag).unique(
                subset=["file_path"], keep="first"
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
