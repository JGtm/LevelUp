"""Section armes utilisées — camembert + tableau pour la vue match."""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import plotly.graph_objects as go
import polars as pl
import streamlit as st

if TYPE_CHECKING:
    from src.data.repositories import DuckDBRepository

logger = logging.getLogger(__name__)

from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, fragment_if_available
from src.visualization.theme import apply_halo_plot_style


def _build_weapon_kills_df(
    repo: DuckDBRepository, match_id: str, xuid: str, lang: str
) -> pl.DataFrame:
    """Charge et prépare les kills par arme du joueur pour un match.

    Returns:
        DataFrame (weapon_name, kills) trié par kills DESC, filtré sur kills > 0.
    """
    empty = pl.DataFrame(schema={"weapon_name": pl.Utf8, "kills": pl.Int32})
    try:
        # load_weapon_kills_for_player filtre directement par xuid en SQL (v_weapon_kills)
        df = repo.load_weapon_kills_for_player(str(xuid).strip(), match_ids=[match_id])
    except Exception as exc:
        logger.debug("_build_weapon_kills_df match=%s xuid=%s : %s", match_id, xuid, exc)
        return empty

    if df.is_empty():
        return empty

    from src.analysis._weapon_data import (
        EXCLUDED_WEAPON_IDS,
        WEAPON_FUSION_MAP_ID,
        resolve_weapon_display,
    )

    # df a déjà match_id, weapon_id, kills — on filtre uniquement les exclusions et kills > 0
    filtered = df.filter(pl.col("kills") > 0).filter(
        ~pl.col("weapon_id").is_in(list(EXCLUDED_WEAPON_IDS))
    )
    if filtered.is_empty():
        return empty

    # Fusion variantes → canonical weapon_id
    fused = (
        filtered.with_columns(pl.col("weapon_id").replace(WEAPON_FUSION_MAP_ID).alias("weapon_id"))
        .group_by("weapon_id")
        .agg(pl.col("kills").sum())
    )
    # Résolution weapon_id → nom d'affichage
    resolved = fused.with_columns(
        pl.col("weapon_id")
        .map_elements(lambda wid: resolve_weapon_display(wid, lang) or "?", return_dtype=pl.Utf8)
        .alias("weapon_name")
    ).select("weapon_name", "kills")
    return resolved.sort("kills", descending=True)


def _render_weapon_pie(df: pl.DataFrame, colors: dict) -> None:
    """Rend le camembert des kills par arme."""
    palette = [
        colors.get("highlight", "#29B6F6"),
        colors.get("accent", "#FF7043"),
        "#66BB6A",
        "#FFA726",
        "#AB47BC",
        "#26C6DA",
        "#EC407A",
        "#8D6E63",
    ]
    slice_colors = (palette * ((len(df) // len(palette)) + 1))[: len(df)]
    fig = go.Figure(
        go.Pie(
            labels=df["weapon_name"].to_list(),
            values=df["kills"].to_list(),
            hole=0.35,
            marker={"colors": slice_colors},
            textinfo="percent",
            textposition="inside",
            insidetextorientation="horizontal",
            hovertemplate="%{label}<br>%{value} frags (%{percent})<extra></extra>",
            sort=False,
        )
    )
    apply_halo_plot_style(fig, height=300)
    fig.update_layout(
        margin={"t": 10, "b": 10, "l": 10, "r": 10},
        showlegend=True,
        legend={
            "orientation": "v",
            "x": 1.02,
            "y": 0.5,
            "xanchor": "left",
            "yanchor": "middle",
            "font": {"color": "#dce8ff", "size": 11},
            "bgcolor": "rgba(0,0,0,0)",
            "borderwidth": 0,
        },
    )
    st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)


def _render_weapon_table(df: pl.DataFrame) -> None:
    """Rend le tableau arme / frags avec le style scoreboard (os-sb-*)."""
    col_w = t("mv_weapon_kills_col_weapon")
    col_f = t("mv_weapon_kills_col_frags")
    col_headers = "".join(
        f"<th class='os-sb-th'>{h}</th>"
        for h in [col_w, col_f]
    )
    rows_html = "".join(
        f"<tr class='os-sb-row'>"
        f"<td class='os-sb-td'>{row['weapon_name']}</td>"
        f"<td class='os-sb-td' style='text-align:right;font-weight:600'>{row['kills']}</td>"
        f"</tr>"
        for row in df.iter_rows(named=True)
    )
    table_html = (
        "<div style='display:flex;align-items:center;min-height:300px'>"
        "<div class='os-table-wrap' style='width:100%'>"
        "<table class='os-table os-scoreboard' style='min-width:0;width:100%'>"
        f"<thead><tr>{col_headers}</tr></thead>"
        f"<tbody>{rows_html}</tbody>"
        "</table>"
        "</div>"
        "</div>"
    )
    st.markdown(table_html, unsafe_allow_html=True)


def _enrich_with_grenade_melee(
    df: pl.DataFrame, repo: DuckDBRepository, match_id: str, xuid: str
) -> pl.DataFrame:
    """Ajoute grenade_kills et melee_kills (API) comme lignes dans le df weapon_name/kills.

    Limite les kills ajoutés au remainder (api_total - film_kills) pour éviter le
    double-comptage des melee kills filmés sous le weapon_id de l'arme tenue.
    """
    try:
        grenade, melee = repo.load_grenade_melee_kills(str(xuid).strip(), [match_id])
    except Exception as exc:
        logger.debug("_enrich_with_grenade_melee match=%s xuid=%s : %s", match_id, xuid, exc)
        return df

    if grenade == 0 and melee == 0:
        return df

    film_kills = int(df["kills"].sum()) if not df.is_empty() else 0
    api_total = repo.load_total_kills_for_player(str(xuid).strip(), [match_id])
    remainder = max(0, api_total - film_kills)

    melee_net = min(melee, remainder)
    grenade_net = min(grenade, max(0, remainder - melee_net))

    extras = []
    if grenade_net > 0:
        extras.append({"weapon_name": t("col_grenade_kills"), "kills": grenade_net})
    if melee_net > 0:
        extras.append({"weapon_name": t("col_melee"), "kills": melee_net})
    if not extras:
        return df
    return pl.concat([df, pl.DataFrame(extras, schema={"weapon_name": pl.Utf8, "kills": pl.Int32})])


@fragment_if_available
def render_weapon_kills_section(*, db_path: str, match_id: str, xuid: str, colors: dict) -> None:
    """Affiche camembert + tableau des kills par arme du joueur pour un match."""
    from src.data.repositories import DuckDBRepository
    from src.ui.i18n import get_lang

    lang = get_lang()
    try:
        repo = DuckDBRepository(db_path, xuid=xuid, read_only=True)
    except Exception as exc:
        logger.debug(
            "render_weapon_kills_section: DB unavailable match=%s xuid=%s: %s", match_id, xuid, exc
        )
        st.subheader(t("mv_weapon_kills_title"))
        st.caption(t("mv_weapon_kills_no_data"))
        return

    df = _build_weapon_kills_df(repo, match_id, xuid, lang)

    st.subheader(t("mv_weapon_kills_title"))
    if df.is_empty():
        st.caption(t("mv_weapon_kills_no_data"))
        return

    df = _enrich_with_grenade_melee(df, repo, match_id, xuid)

    col_chart, col_table = st.columns([3, 2])
    with col_chart:
        _render_weapon_pie(df, colors)
    with col_table:
        _render_weapon_table(df)
