"""Sous-module : graphiques armes/destruction pour la page Séries Temporelles."""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG
from src.visualization.distributions import plot_top_weapons


def _load_grenade_melee_totals(
    db_path: str, xuid: str, match_ids: list[str] | None
) -> tuple[int, int]:
    """Charge les totaux grenade_kills et melee_kills depuis shared.match_participants."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    try:
        repo = DuckDBRepository(db_path, xuid, read_only=True)
        conn = repo._get_connection()
        xuid_str = str(xuid).strip()
        if match_ids:
            placeholders = ", ".join("?" * len(match_ids))
            row = conn.execute(
                f"SELECT COALESCE(SUM(grenade_kills), 0), COALESCE(SUM(melee_kills), 0) "
                f"FROM shared.match_participants "
                f"WHERE xuid = ? AND match_id IN ({placeholders})",
                [xuid_str, *match_ids],
            ).fetchone()
        else:
            row = conn.execute(
                "SELECT COALESCE(SUM(grenade_kills), 0), COALESCE(SUM(melee_kills), 0) "
                "FROM shared.match_participants WHERE xuid = ?",
                [xuid_str],
            ).fetchone()
        if row:
            return int(row[0]), int(row[1])
    except Exception as exc:
        logger.debug("_load_grenade_melee_totals xuid=%s : %s", xuid, exc)
    return 0, 0


def render_weapon_kills_chart(
    dff: pl.DataFrame,
    *,
    db_path: str,
    xuid: str,
    lang: str,
) -> None:
    """Affiche le graphe des kills par arme (barres horizontales) avec grenades/mêlée API."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    match_ids = dff["match_id"].to_list() if "match_id" in dff.columns else None
    try:
        repo = DuckDBRepository(db_path, xuid, read_only=True)
        df_w = repo.load_weapon_kills_aggregated(xuid, match_ids=match_ids)
    except Exception as exc:
        logger.debug("render_weapon_kills_chart xuid=%s : %s", xuid, exc)
        return

    if df_w.is_empty():
        return

    from src.analysis._weapon_data import EXCLUDED_WEAPON_IDS, resolve_weapon_display

    weapons_data = [
        {
            "weapon_name": resolve_weapon_display(row["weapon_id"], lang) or "?",
            "total_kills": row["total_kills"],
            "headshot_rate": 0.0,
            "accuracy": 0.0,
        }
        for row in df_w.iter_rows(named=True)
        if row["weapon_id"] not in EXCLUDED_WEAPON_IDS
    ]

    grenade_total, melee_total = _load_grenade_melee_totals(db_path, xuid, match_ids)
    if grenade_total > 0:
        weapons_data.append(
            {
                "weapon_name": t("col_grenade_kills"),
                "total_kills": grenade_total,
                "headshot_rate": 0.0,
                "accuracy": 0.0,
            }
        )
    if melee_total > 0:
        weapons_data.append(
            {
                "weapon_name": t("col_melee"),
                "total_kills": melee_total,
                "headshot_rate": 0.0,
                "accuracy": 0.0,
            }
        )

    st.divider()
    st.subheader(t("ts_top_weapons_title"))
    with safe_chart_render():
        fig_w = plot_top_weapons(weapons_data, lang=lang)
        st.plotly_chart(fig_w, width="stretch", config=PLOTLY_STATIC_CONFIG)
