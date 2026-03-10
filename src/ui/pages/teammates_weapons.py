"""Tableau des kills par arme pour un coéquipier ou un ensemble de joueurs.

Affiche un tableau HTML (format ``os-table``) agrégé par arme,
avec le nom traduit et le total de kills.
"""

from __future__ import annotations

import html
import logging
from typing import Any

import streamlit as st

from src.ui.i18n import t

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

    # Construire les lignes de données
    rows: list[dict[str, Any]] = []
    for row in df.iter_rows(named=True):
        rows.append(
            {
                "name": row["weapon_name"],
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
