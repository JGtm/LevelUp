"""Filtres V2 pour la page Médias — toolbar horizontale avec groupement."""

from __future__ import annotations

from dataclasses import dataclass

import polars as pl
import streamlit as st

from src.ui.i18n import t

# Clés de tri disponibles et colonnes correspondantes
_SORT_KEYS: list[str] = ["media_sort_date_capture", "media_sort_map", "media_sort_mode"]
_SORT_COLS: dict[str, str] = {
    "media_sort_date_capture": "capture_end_utc",
    "media_sort_map": "map_ui",
    "media_sort_mode": "mode_ui",
}


@dataclass(frozen=True)
class MediaFilterState:
    """État des filtres de la page Médias V2.

    auteur_key : None = tous, "mine" = mes captures, "<gamertag>" = coéquipier.
    group_by   : None = pas de groupement, "date" / "map" / "mode" = regroupement.
    """

    auteur_key: str | None
    carte: str | None
    mode: str | None
    sort_col: str
    sort_desc: bool
    group_by: str | None
    cols_per_row: int = 4


def render_media_filters(media_df: pl.DataFrame) -> MediaFilterState:
    """Rend la toolbar horizontale (Auteur | Carte | Mode | ─ | Grouper par | Tri | Ordre)."""
    map_col = "map_ui" if "map_ui" in media_df.columns else "map_name"
    mode_col = "mode_ui" if "mode_ui" in media_df.columns else "pair_name"
    all_maps = _unique_values(media_df, map_col)
    all_modes = _unique_values(media_df, mode_col)

    tout = t("media_filter_all")
    auteur_display_to_key, auteur_labels = _build_auteur_mapping(media_df, tout)
    sort_labels = [t(k) for k in _SORT_KEYS]
    order_labels = [t("media_sort_desc"), t("media_sort_asc")]

    group_keys = [None, "author", "date", "session", "experience", "map", "mode"]
    group_labels = [
        t("media_group_none"),
        t("media_group_author"),
        t("media_group_date"),
        t("media_group_session"),
        t("media_group_experience"),
        t("media_group_map"),
        t("media_group_mode"),
    ]

    with st.container(border=True):
        c1, c2, c3, sep, c4, c5, c6 = st.columns([1.5, 2, 2, 0.1, 2, 2, 1.2])
        with c1:
            auteur_lbl = st.selectbox(
                t("media_filter_auteur"),
                options=auteur_labels,
                key="mv2_auteur",
            )
        with c2:
            carte_lbl = st.selectbox(
                t("media_filter_map"),
                options=[tout] + all_maps,
                key="mv2_carte",
            )
        with c3:
            mode_lbl = st.selectbox(
                t("media_filter_mode"),
                options=[tout] + all_modes,
                key="mv2_mode",
            )
        sep.markdown(
            "<div style='height:2.5rem;display:flex;align-items:center;color:#666'>│</div>",
            unsafe_allow_html=True,
        )
        with c4:
            group_lbl = st.selectbox(
                t("media_group_by"),
                options=group_labels,
                key="mv2_group",
            )
        with c5:
            sort_lbl = st.selectbox(
                t("media_sort_by"),
                options=sort_labels,
                key="mv2_sort",
            )
        with c6:
            order_lbl = st.selectbox(
                t("media_sort_order"),
                options=order_labels,
                key="mv2_order",
            )

    sort_key = _SORT_KEYS[sort_labels.index(sort_lbl)]
    raw_col = _SORT_COLS[sort_key]
    sort_col = raw_col if raw_col in media_df.columns else "capture_end_utc"
    group_by = group_keys[group_labels.index(group_lbl)]

    return MediaFilterState(
        auteur_key=auteur_display_to_key.get(auteur_lbl),
        carte=None if carte_lbl == tout else carte_lbl,
        mode=None if mode_lbl == tout else mode_lbl,
        sort_col=sort_col,
        sort_desc=order_lbl == t("media_sort_desc"),
        group_by=group_by,
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
        df = df.sort(
            [sort_col, "file_path"],
            descending=[state.sort_desc, False],
            nulls_last=True,
        )
    return df


# ── Helpers privés ────────────────────────────────────────────────────────────


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
