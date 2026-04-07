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

# IDs sentinel pour grenades/mêlée (définis dans EXCLUDED_WEAPON_IDS, ici réutilisés)
_GRENADE_WEAPON_ID = 0
_MELEE_WEAPON_ID = 1
# weapon_ids à exclure du film (incomplet) avant réinjection API
_FILM_EXCLUDED_IDS = {_GRENADE_WEAPON_ID, _MELEE_WEAPON_ID}


def _resolve_weapon_name(wid: int, lang: str) -> str:
    """Résout l'ID arme en nom d'affichage, avec gestion des IDs synthétiques."""
    from src.analysis._weapon_data import resolve_weapon_display

    if wid == _GRENADE_WEAPON_ID:
        return t("col_grenade_kills", lang=lang)
    if wid == _MELEE_WEAPON_ID:
        return t("col_melee", lang=lang)
    return resolve_weapon_display(wid, lang=lang) or "?"


def _append_grenade_melee(
    conn: Any,
    xuid: str,
    match_ids: list[str],
    df: pl.DataFrame,
) -> pl.DataFrame:
    """Réinjecte les kills grenade/mêlée depuis shared.match_participants.

    Args:
        conn: Connexion DuckDB (schéma shared doit contenir match_participants).
        xuid: XUID du joueur.
        match_ids: Matchs à considérer (vide = tous les matchs du joueur).
        df: DataFrame existant (weapon_id: UInt64, total_kills: Int32).

    Returns:
        DataFrame enrichi avec les lignes grenade/mêlée si leurs totaux > 0.
    """
    try:
        if match_ids:
            placeholders = ", ".join("?" * len(match_ids))
            row = conn.execute(
                f"SELECT COALESCE(SUM(grenade_kills), 0), COALESCE(SUM(melee_kills), 0) "
                f"FROM shared.match_participants "
                f"WHERE xuid = ? AND match_id IN ({placeholders})",
                [xuid, *match_ids],
            ).fetchone()
        else:
            row = conn.execute(
                "SELECT COALESCE(SUM(grenade_kills), 0), COALESCE(SUM(melee_kills), 0) "
                "FROM shared.match_participants WHERE xuid = ?",
                [xuid],
            ).fetchone()
        if not row:
            return df
        grenade, melee = int(row[0]), int(row[1])
        extras = []
        if grenade > 0:
            extras.append({"weapon_id": _GRENADE_WEAPON_ID, "total_kills": grenade})
        if melee > 0:
            extras.append({"weapon_id": _MELEE_WEAPON_ID, "total_kills": melee})
        if extras:
            return pl.concat(
                [
                    df,
                    pl.DataFrame(extras, schema={"weapon_id": pl.UInt64, "total_kills": pl.Int32}),
                ]
            )
    except Exception as exc:
        logger.debug("_append_grenade_melee xuid=%s : %s", xuid, exc)
    return df


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

    # Retirer les entrées film pour grenade/mêlée, réinjecter depuis l'API.
    # Limité au remainder (api_total - film_kills) pour éviter le double-comptage
    # des melee kills filmés sous le weapon_id de l'arme tenue.
    df = df.filter(~pl.col("weapon_id").is_in(list(_FILM_EXCLUDED_IDS)))
    try:
        grenade, melee = repo.load_grenade_melee_kills(xuid, match_ids)
        film_kills = int(df["total_kills"].sum()) if not df.is_empty() else 0
        api_total = repo.load_total_kills_for_player(xuid, match_ids)
        remainder = max(0, api_total - film_kills)
        melee_net = min(melee, remainder)
        grenade_net = min(grenade, max(0, remainder - melee_net))
        extras = []
        if grenade_net > 0:
            extras.append({"weapon_id": _GRENADE_WEAPON_ID, "total_kills": grenade_net})
        if melee_net > 0:
            extras.append({"weapon_id": _MELEE_WEAPON_ID, "total_kills": melee_net})
        if extras:
            df = pl.concat(
                [df, pl.DataFrame(extras, schema={"weapon_id": pl.UInt64, "total_kills": pl.Int32})]
            )
    except Exception as exc:
        logger.debug(
            "render_weapon_kills_table: grenade/melee enrich failed xuid=%s : %s", xuid, exc
        )

    header = title or t("section_weapon_stats")
    st.markdown(f"##### {html.escape(header)}")

    from src.ui.i18n import get_lang

    lang = get_lang()
    st.markdown(_build_weapon_table_html(df, lang), unsafe_allow_html=True)


def _build_weapon_table_html(df: pl.DataFrame, lang: str) -> str:
    """Construit le HTML du tableau d'armes trié par kills décroissants."""
    rows: list[dict[str, Any]] = []
    for row in df.iter_rows(named=True):
        rows.append(
            {
                "name": _resolve_weapon_name(int(row["weapon_id"]), lang=lang),
                "faction": "—",
                "kills": int(row["total_kills"]),
            }
        )
    rows.sort(key=lambda r: r["kills"], reverse=True)

    col_weapon = html.escape(t("col_weapon_name"))
    col_faction = html.escape(t("col_weapon_kills"))
    th = f"<th>{col_weapon}</th><th>Faction</th><th>{col_faction}</th>"

    body = []
    for r in rows:
        name_esc = html.escape(r["name"])
        faction_esc = html.escape(r["faction"])
        body.append(f"<tr><td>{name_esc}</td><td>{faction_esc}</td><td>{r['kills']}</td></tr>")
    return (
        "<div class='os-table-wrap'>"
        "<table class='os-table'>"
        f"<thead><tr>{th}</tr></thead>"
        f"<tbody>{''.join(body)}</tbody>"
        "</table></div>"
    )


_TOP_N_WEAPONS = 12


def load_weapon_kills_data(
    player_infos: list[tuple[str, str, list[str]]],
    db_path: str,
) -> list[tuple[str, pl.DataFrame]]:
    """Charge les données de kills par arme depuis la DB, avec grenades/mêlée API.

    Args:
        player_infos: Liste de (nom, xuid, match_ids).
        db_path: Chemin DB (shared_matches accessible).

    Returns:
        Liste de (nom, DataFrame top-N armes + grenades/mêlée) pour les joueurs avec données.
    """
    from src.data.repositories import DuckDBRepository

    data: list[tuple[str, pl.DataFrame]] = []
    try:
        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        for name, xuid, match_ids in player_infos:
            df = repo.load_weapon_kills_aggregated(xuid, match_ids or [])
            # Retirer les entrées film (incomplètes) pour grenade/mêlée
            df = df.filter(~pl.col("weapon_id").is_in(list(_FILM_EXCLUDED_IDS)))
            df = df.head(_TOP_N_WEAPONS)
            # Réinjecter depuis l'API, limité au remainder pour éviter le double-comptage
            # des melee kills filmés sous le weapon_id de l'arme tenue.
            grenade, melee = repo.load_grenade_melee_kills(xuid, match_ids or [])
            film_kills = int(df["total_kills"].sum()) if not df.is_empty() else 0
            api_total = repo.load_total_kills_for_player(xuid, match_ids or [])
            remainder = max(0, api_total - film_kills)
            melee_net = min(melee, remainder)
            grenade_net = min(grenade, max(0, remainder - melee_net))
            extras = []
            if grenade_net > 0:
                extras.append({"weapon_id": _GRENADE_WEAPON_ID, "total_kills": grenade_net})
            if melee_net > 0:
                extras.append({"weapon_id": _MELEE_WEAPON_ID, "total_kills": melee_net})
            if extras:
                df = pl.concat(
                    [
                        df,
                        pl.DataFrame(
                            extras, schema={"weapon_id": pl.UInt64, "total_kills": pl.Int32}
                        ),
                    ]
                )
            if not df.is_empty():
                data.append((name, df))
    except Exception as exc:
        logger.warning("load_weapon_kills_data: %s", exc)
    return data


def _build_zebra_shapes(n: int) -> list[dict]:
    """Génère les bandes de fond alternées pour délimiter visuellement chaque arme."""
    return [
        {
            "type": "rect",
            "xref": "paper",
            "yref": "y",
            "x0": 0,
            "x1": 1,
            "y0": i - 0.5,
            "y1": i + 0.5,
            "fillcolor": "rgba(255,255,255,0.15)",
            "line_width": 0,
            "layer": "below",
        }
        for i in range(n)
        if i % 2 == 0
    ]


def _add_weapon_bar_traces(
    fig: go.Figure,
    data: list[tuple[str, pl.DataFrame]],
    weapons_sorted_ids: list[int],
    weapons_sorted: list[str],
    colors_by_name: dict[str, str],
) -> None:
    """Ajoute une trace Bar par joueur sur la figure."""
    for idx, (name, df) in enumerate(data):
        kills_map = {int(r["weapon_id"]): r["total_kills"] for r in df.iter_rows(named=True)}
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

    from src.ui.i18n import get_lang

    lang = get_lang()
    all_weapons: dict[int, int] = {}
    for _, df in data:
        for row in df.iter_rows(named=True):
            wid = int(row["weapon_id"])
            all_weapons[wid] = all_weapons.get(wid, 0) + row["total_kills"]
    weapons_sorted_ids = sorted(all_weapons.keys(), key=lambda w: all_weapons[w])
    weapons_sorted = [_resolve_weapon_name(wid, lang=lang) for wid in weapons_sorted_ids]

    fig = go.Figure()
    _add_weapon_bar_traces(fig, data, weapons_sorted_ids, weapons_sorted, colors_by_name)

    fig.update_layout(
        title=t("tm_weapon_kills_chart"),
        height=max(350, len(weapons_sorted) * 38),
        margin={"l": 20, "r": 80, "t": 60, "b": 20},
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        barmode="group",
        bargap=0.35,
        bargroupgap=0.08,
        shapes=_build_zebra_shapes(len(weapons_sorted)),
        showlegend=False,
    )
    fig.update_xaxes(showgrid=False, visible=False)
    st.plotly_chart(
        fig,
        width="stretch",
        key=f"weapon_kills_bar_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
    )
