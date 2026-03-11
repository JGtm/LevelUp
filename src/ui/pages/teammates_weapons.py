"""Tableau et graphes des kills par arme pour un coéquipier ou un ensemble de joueurs."""

from __future__ import annotations

import html
import logging
from typing import Any

import plotly.graph_objects as go
import polars as pl
import streamlit as st

from src.config import OKABE_ITO_PALETTE
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG

logger = logging.getLogger(__name__)


def render_weapon_kills_table(
    db_path: str,
    xuid: str,
    match_ids: list[str],
    *,
    title: str | None = None,
) -> None:
    """Affiche un tableau des kills par arme agrégé sur un ensemble de matchs.

    Args:
        db_path: Chemin vers la base de données joueur.
        xuid: XUID du joueur.
        match_ids: Matchs à considérer.
        title: Titre optionnel (default: t("section_weapon_stats")).
    """
    if not match_ids:
        return

    from src.data.repositories import DuckDBRepository

    try:
        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        df = repo.load_weapon_kills_aggregated(xuid, match_ids)
    except Exception as exc:
        logger.debug("weapon_kills_table : %s", exc)
        return

    if df.is_empty():
        return

    header = title or t("section_weapon_stats")
    st.markdown(f"##### {html.escape(header)}")

    from src.analysis._weapon_data import resolve_weapon_display
    from src.ui.i18n import get_lang

    lang = get_lang()
    # Construire les lignes de données
    rows: list[dict[str, Any]] = []
    for row in df.iter_rows(named=True):
        rows.append(
            {
                "name": resolve_weapon_display(row["weapon_id"], lang=lang) or "?",
                "faction": "—",
                "kills": int(row["total_kills"]),
            }
        )

    # Tri par kills décroissant
    rows.sort(key=lambda r: r["kills"], reverse=True)

    # En-tête HTML
    col_weapon = html.escape(t("col_weapon_name"))
    col_faction = html.escape(t("col_weapon_kills"))
    th = f"<th>{col_weapon}</th><th>Faction</th><th>{col_faction}</th>"

    # Lignes
    body = []
    for r in rows:
        name_esc = html.escape(r["name"])
        faction_esc = html.escape(r["faction"])
        body.append(
            f"<tr><td>{name_esc}</td>" f"<td>{faction_esc}</td>" f"<td>{r['kills']}</td></tr>"
        )

    table_html = (
        "<div class='os-table-wrap'>"
        "<table class='os-table'>"
        f"<thead><tr>{th}</tr></thead>"
        f"<tbody>{''.join(body)}</tbody>"
        "</table></div>"
    )
    st.markdown(table_html, unsafe_allow_html=True)


_TOP_N_WEAPONS = 12


def load_weapon_kills_data(
    player_infos: list[tuple[str, str, list[str]]],
    db_path: str,
) -> list[tuple[str, pl.DataFrame]]:
    """Charge les données de kills par arme depuis la DB.

    Args:
        player_infos: Liste de (nom, xuid, match_ids).
        db_path: Chemin DB (shared_matches accessible).

    Returns:
        Liste de (nom, DataFrame top-N armes) pour les joueurs avec données.
    """
    from src.data.repositories import DuckDBRepository

    data: list[tuple[str, pl.DataFrame]] = []
    try:
        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        for name, xuid, match_ids in player_infos:
            df = repo.load_weapon_kills_aggregated(xuid, match_ids or [])
            if not df.is_empty():
                data.append((name, df.head(_TOP_N_WEAPONS)))
    except Exception as exc:
        logger.warning("load_weapon_kills_data: %s", exc)
    return data


def render_weapon_kills_bar_chart(
    player_infos: list[tuple[str, str, list[str]]],
    colors_by_name: dict[str, str],
    db_path: str,
    key_suffix: str,
) -> None:
    """Graphe horizontal groupé des armes de kill — tous les joueurs sur un seul graphe.

    Args:
        player_infos: Liste de (nom, xuid, match_ids).
        colors_by_name: Mapping nom → couleur hex.
        db_path: Chemin DB (shared_matches accessible).
        key_suffix: Suffixe clé Streamlit unique.
    """
    data = load_weapon_kills_data(player_infos, db_path)

    if not data:
        st.caption("ℹ Aucune donnée d'armes pour ces matchs")
        return

    from src.analysis._weapon_data import resolve_weapon_display
    from src.ui.i18n import get_lang

    lang = get_lang()
    # Union de toutes les armes (top N par joueur) triées par total global
    all_weapons: dict[int, int] = {}
    for _, df in data:
        for row in df.iter_rows(named=True):
            wid = row["weapon_id"]
            all_weapons[wid] = all_weapons.get(wid, 0) + row["total_kills"]
    weapons_sorted_ids = sorted(all_weapons.keys(), key=lambda w: all_weapons[w])
    weapons_sorted = [resolve_weapon_display(wid, lang=lang) or "?" for wid in weapons_sorted_ids]

    fig = go.Figure()
    for idx, (name, df) in enumerate(data):
        kills_map = {r["weapon_id"]: r["total_kills"] for r in df.iter_rows(named=True)}
        kills = [kills_map.get(wid, 0) for wid in weapons_sorted_ids]
        color = colors_by_name.get(name, OKABE_ITO_PALETTE[idx % len(OKABE_ITO_PALETTE)])
        fig.add_trace(
            go.Bar(
                orientation="h",
                x=kills,
                y=weapons_sorted,
                name=name,
                marker_color=color,
                text=[str(k) if k > 0 else "" for k in kills],
                textposition="outside",
                textfont={"color": "white", "size": 11, "family": "Arial Black"},
            ),
        )

    fig.update_layout(
        title=t("tm_weapon_kills_chart"),
        height=max(350, len(weapons_sorted) * 30),
        margin={"l": 20, "r": 80, "t": 60, "b": 20},
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        barmode="group",
        legend={"orientation": "h", "yanchor": "top", "y": -0.05, "xanchor": "center", "x": 0.5},
    )
    fig.update_xaxes(showgrid=False, visible=False)
    st.plotly_chart(
        fig,
        width="stretch",
        key=f"weapon_kills_bar_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
    )
