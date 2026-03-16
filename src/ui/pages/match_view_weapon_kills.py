"""Section armes utilisées — camembert + tableau pour la vue match."""

from __future__ import annotations

import logging

import plotly.graph_objects as go
import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization.theme import apply_halo_plot_style


def _build_weapon_kills_df(db_path: str, match_id: str, xuid: str, lang: str) -> pl.DataFrame:
    """Charge et prépare les kills par arme du joueur pour un match.

    Returns:
        DataFrame (weapon_name, kills) trié par kills DESC, filtré sur kills > 0.
    """
    from src.data.repositories import DuckDBRepository

    try:
        repo = DuckDBRepository(db_path, xuid=xuid, read_only=True)
        # load_weapon_kills_for_player filtre directement par xuid en SQL (v_weapon_kills)
        df = repo.load_weapon_kills_for_player(str(xuid).strip(), match_ids=[match_id])
    except Exception as exc:
        logger.debug("_build_weapon_kills_df match=%s xuid=%s : %s", match_id, xuid, exc)
        return pl.DataFrame(schema={"weapon_name": pl.Utf8, "kills": pl.Int32})

    if df.is_empty():
        return pl.DataFrame(schema={"weapon_name": pl.Utf8, "kills": pl.Int32})

    from src.analysis._weapon_data import (
        EXCLUDED_WEAPON_IDS,
        WEAPON_FUSION_MAP_ID,
        resolve_weapon_display,
    )

    # df a déjà match_id, weapon_id, kills — on filtre uniquement les exclusions et kills > 0
    filtered = (
        df.filter(pl.col("kills") > 0)
        .filter(~pl.col("weapon_id").is_in(list(EXCLUDED_WEAPON_IDS)))
    )
    if filtered.is_empty():
        return pl.DataFrame(schema={"weapon_name": pl.Utf8, "kills": pl.Int32})

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
    fig = go.Figure(
        go.Pie(
            labels=df["weapon_name"].to_list(),
            values=df["kills"].to_list(),
            hole=0.35,
            marker={"colors": palette[: len(df)]},
            textinfo="percent",
            hovertemplate="%{label}<br>%{value} frags (%{percent})<extra></extra>",
            sort=False,
        )
    )
    apply_halo_plot_style(fig, height=260)
    fig.update_layout(
        margin={"t": 10, "b": 10, "l": 10, "r": 10},
        showlegend=False,
    )
    st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)


def _render_weapon_table(df: pl.DataFrame) -> None:
    """Rend le tableau arme / frags."""
    col_w = t("mv_weapon_kills_col_weapon")
    col_f = t("mv_weapon_kills_col_frags")
    rows_html = "".join(
        f"<tr><td style='padding:4px 12px 4px 0'>{row['weapon_name']}</td>"
        f"<td style='text-align:right;font-weight:600'>{row['kills']}</td></tr>"
        for row in df.iter_rows(named=True)
    )
    table_html = (
        f"<table style='width:100%;border-collapse:collapse;font-size:0.9em'>"
        f"<thead><tr>"
        f"<th style='text-align:left;padding:4px 12px 4px 0;color:#888'>{col_w}</th>"
        f"<th style='text-align:right;color:#888'>{col_f}</th>"
        f"</tr></thead><tbody>{rows_html}</tbody></table>"
    )
    st.markdown(table_html, unsafe_allow_html=True)


def _enrich_with_grenade_melee(
    df: pl.DataFrame, db_path: str, match_id: str, xuid: str
) -> pl.DataFrame:
    """Ajoute grenade_kills et melee_kills (API) comme lignes dans le df weapon_name/kills."""
    from src.data.repositories import DuckDBRepository

    try:
        repo = DuckDBRepository(db_path, xuid=xuid, read_only=True)
        grenade, melee = repo.load_grenade_melee_kills(str(xuid).strip(), [match_id])
    except Exception as exc:
        logger.debug("_enrich_with_grenade_melee match=%s xuid=%s : %s", match_id, xuid, exc)
        return df

    extras = []
    if grenade > 0:
        extras.append({"weapon_name": t("col_grenade_kills"), "kills": grenade})
    if melee > 0:
        extras.append({"weapon_name": t("col_melee"), "kills": melee})
    if not extras:
        return df
    return pl.concat([df, pl.DataFrame(extras, schema={"weapon_name": pl.Utf8, "kills": pl.Int32})])


@fragment_if_available
def render_weapon_kills_section(*, db_path: str, match_id: str, xuid: str, colors: dict) -> None:
    """Affiche camembert + tableau des kills par arme du joueur pour un match."""
    from src.ui.i18n import get_lang

    lang = get_lang()
    df = _build_weapon_kills_df(db_path, match_id, xuid, lang)
    df = _enrich_with_grenade_melee(df, db_path, match_id, xuid)

    st.subheader(t("mv_weapon_kills_title"))
    if df.is_empty():
        st.caption(t("mv_weapon_kills_no_data"))
        return

    col_chart, col_table = st.columns([3, 2])
    with col_chart:
        _render_weapon_pie(df, colors)
    with col_table:
        _render_weapon_table(df)
