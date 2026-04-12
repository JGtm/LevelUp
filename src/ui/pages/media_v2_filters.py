"""Filtres V2 pour la page Médias — toolbar horizontale avec groupement."""

from __future__ import annotations

from dataclasses import dataclass

import polars as pl
import streamlit as st

from src.ui.i18n import t

_INLINE_LABEL_CSS = """
<style>
div[data-testid="stSelectbox"] {
    display: flex !important;
    flex-direction: row !important;
    align-items: center !important;
    gap: 8px !important;
    padding-bottom: 0 !important;
}
div[data-testid="stSelectbox"] div[data-testid="stWidgetLabel"] {
    white-space: nowrap !important;
    padding-bottom: 0 !important;
    margin-bottom: 0 !important;
    flex-shrink: 0 !important;
}
div[data-testid="stSelectbox"] [data-baseweb="select"] {
    flex: 1 !important;
    min-width: 0 !important;
}
div[data-testid="stCheckbox"] > div[data-testid="stWidgetLabel"],
div[data-testid="stToggle"] > div[data-testid="stWidgetLabel"] {
    display: none !important;
}
div[data-testid="stCheckbox"],
div[data-testid="stToggle"] {
    margin-top: 0 !important;
}
</style>
"""

_SORT_KEYS: list[str] = ["media_sort_date_capture", "media_sort_map", "media_sort_mode"]
_SORT_COLS: dict[str, str] = {
    "media_sort_date_capture": "capture_end_utc",
    "media_sort_map": "map_ui",
    "media_sort_mode": "mode_ui",
}


@dataclass(frozen=True)
class MediaFilterState:
    """État des filtres de la page Médias V2."""

    auteur_key: str | None
    carte: str | None
    mode: str | None
    sort_col: str
    sort_desc: bool
    group_by: str | None
    cols_per_row: int = 4
    autoplay_videos: bool = True


@dataclass(frozen=True)
class _ToolbarSelection:
    """Sélections UI brutes issues de la toolbar médias."""

    auteur_label: str
    carte_label: str
    mode_label: str
    group_label: str
    sort_label: str
    order_label: str
    autoplay: bool


@dataclass(frozen=True)
class _ToolbarChoices:
    """Jeu d'options injecté dans la toolbar médias."""

    auteur_labels: list[str]
    map_options: list[str]
    mode_options: list[str]
    group_labels: list[str]
    sort_labels: list[str]
    order_labels: list[str]


def render_media_filters(media_df: pl.DataFrame) -> MediaFilterState:
    """Rend la toolbar horizontale compacte (1 ligne : Auteur|Carte|Mode|─|Groupe|Tri|Ordre|▶)."""
    st.markdown(_INLINE_LABEL_CSS, unsafe_allow_html=True)
    map_col = "map_ui" if "map_ui" in media_df.columns else "map_name"
    mode_col = "mode_ui" if "mode_ui" in media_df.columns else "pair_name"
    all_maps = _unique_values(media_df, map_col)
    all_modes = _unique_values(media_df, mode_col)
    tout = t("media_filter_all")
    auteur_display_to_key, auteur_labels = _build_auteur_mapping(media_df, tout)
    sort_labels = [t(k) for k in _SORT_KEYS]
    order_labels = [t("media_sort_desc"), t("media_sort_asc")]
    group_keys, group_labels = _group_options()
    selection = _render_filters_toolbar(
        _ToolbarChoices(
            auteur_labels=auteur_labels,
            map_options=[tout] + all_maps,
            mode_options=[tout] + all_modes,
            group_labels=group_labels,
            sort_labels=sort_labels,
            order_labels=order_labels,
        )
    )
    raw_col = _SORT_COLS[_SORT_KEYS[sort_labels.index(selection.sort_label)]]
    sort_col = raw_col if raw_col in media_df.columns else "capture_end_utc"
    group_by = group_keys[group_labels.index(selection.group_label)]
    return MediaFilterState(
        auteur_key=auteur_display_to_key.get(selection.auteur_label),
        carte=None if selection.carte_label == tout else selection.carte_label,
        mode=None if selection.mode_label == tout else selection.mode_label,
        sort_col=sort_col,
        sort_desc=selection.order_label == t("media_sort_desc"),
        group_by=group_by,
        autoplay_videos=selection.autoplay,
    )


def apply_media_filters(
    df: pl.DataFrame,
    state: MediaFilterState,
    *,
    match_filters: bool = True,
) -> pl.DataFrame:
    """Applique les filtres carte / mode et le tri sur un DataFrame médias."""
    if match_filters:
        map_col = "map_ui" if "map_ui" in df.columns else "map_name"
        mode_col = "mode_ui" if "mode_ui" in df.columns else "pair_name"
        if state.carte and map_col in df.columns:
            df = df.filter(pl.col(map_col) == state.carte)
        if state.mode and mode_col in df.columns:
            df = df.filter(pl.col(mode_col) == state.mode)
    sort_col = state.sort_col if state.sort_col in df.columns else "capture_end_utc"
    if sort_col in df.columns:
        df = df.sort([sort_col, "file_path"], descending=[state.sort_desc, False], nulls_last=True)
    return df


def _group_options() -> tuple[list[str | None], list[str]]:
    """Retourne les options de groupement supportées par la grille médias."""
    keys: list[str | None] = [
        None,
        "author",
        "date",
        "session",
        "experience",
        "map",
        "mode",
        "liked",
    ]
    labels = [
        t("media_group_none"),
        t("media_group_author"),
        t("media_group_date"),
        t("media_group_session"),
        t("media_group_experience"),
        t("media_group_map"),
        t("media_group_mode"),
        t("media_group_liked"),
    ]
    return keys, labels


def _render_filters_toolbar(choices: _ToolbarChoices) -> _ToolbarSelection:
    """Rend la toolbar et retourne les labels bruts sélectionnés."""
    with st.container(border=True):
        c1, c2, c3, sep, c4, c5, c6, c7 = st.columns([1.5, 2, 2.5, 0.05, 1.8, 1.5, 1.1, 1.0])
        with c1:
            auteur_lbl = st.selectbox(
                t("media_filter_auteur"), options=choices.auteur_labels, key="mv2_auteur"
            )
        with c2:
            carte_lbl = st.selectbox(
                t("media_filter_map"), options=choices.map_options, key="mv2_carte"
            )
        with c3:
            mode_lbl = st.selectbox(
                t("media_filter_mode"), options=choices.mode_options, key="mv2_mode"
            )
        sep.markdown(
            "<div style='height:2.5rem;display:flex;align-items:center;color:#666'>│</div>",
            unsafe_allow_html=True,
        )
        with c4:
            group_lbl = st.selectbox(
                t("media_group_by_short"), options=choices.group_labels, key="mv2_group"
            )
        with c5:
            sort_lbl = st.selectbox(
                t("media_sort_by_short"), options=choices.sort_labels, key="mv2_sort"
            )
        with c6:
            order_lbl = st.selectbox(
                t("media_sort_order"), options=choices.order_labels, key="mv2_order"
            )
        with c7:
            autoplay = st.toggle(
                "▶",
                value=st.session_state.get("mv2_autoplay", True),
                key="mv2_autoplay",
                help=t("ml_autoplay_videos") + " — " + t("ml_autoplay_videos_help"),
            )
    return _ToolbarSelection(
        auteur_label=auteur_lbl,
        carte_label=carte_lbl,
        mode_label=mode_lbl,
        group_label=group_lbl,
        sort_label=sort_lbl,
        order_label=order_lbl,
        autoplay=autoplay,
    )


def _unique_values(df: pl.DataFrame, col: str) -> list[str]:
    """Retourne la liste triée des valeurs uniques non-nulles d'une colonne."""
    if col not in df.columns:
        return []
    return sorted({str(v) for v in df[col].drop_nulls().to_list() if v})


def _build_auteur_mapping(
    media_df: pl.DataFrame,
    tout_label: str,
) -> tuple[dict[str, str | None], list[str]]:
    """Construit le mapping label d'affichage → clé interne pour le filtre Auteur."""
    mapping: dict[str, str | None] = {tout_label: None}
    mapping[t("media_owner_mine")] = "mine"
    if "gamertag" in media_df.columns and "section" in media_df.columns:
        teammates = sorted(
            {
                str(g)
                for g in media_df.filter(pl.col("section") == "teammate")["gamertag"]
                .drop_nulls()
                .to_list()
                if g
            }
        )
        for gt in teammates:
            mapping[gt] = gt
    return mapping, list(mapping.keys())


def _unique_values(df: pl.DataFrame, col: str) -> list[str]:
    """Retourne la liste triée des valeurs uniques non-nulles d'une colonne."""
    if col not in df.columns:
        return []
    return sorted({str(v) for v in df[col].drop_nulls().to_list() if v})


def _build_auteur_mapping(
    media_df: pl.DataFrame,
    tout_label: str,
) -> tuple[dict[str, str | None], list[str]]:
    """Construit le mapping label d'affichage → clé interne pour le filtre Auteur."""
    mapping: dict[str, str | None] = {tout_label: None}
    mapping[t("media_owner_mine")] = "mine"
    if "gamertag" in media_df.columns and "section" in media_df.columns:
        teammates = sorted(
            {
                str(g)
                for g in media_df.filter(pl.col("section") == "teammate")["gamertag"]
                .drop_nulls()
                .to_list()
                if g
            }
        )
        for gt in teammates:
            mapping[gt] = gt
    return mapping, list(mapping.keys())
