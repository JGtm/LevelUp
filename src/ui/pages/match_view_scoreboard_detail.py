"""Détails d'un joueur dans le scoreboard — panneau d'expansion.

Charge les données étendues (médailles, armes, enrichissement joueur)
et rend un panneau accordéon par joueur après le tableau des scores.
"""

from __future__ import annotations

import logging
from typing import Any

import streamlit as st

from src.config import TEAM_MAP
from src.ui.i18n import get_lang, t
from src.ui.medals import render_medals_grid
from src.utils import parse_xuid_input
from src.utils.paths import get_player_db_path

logger = logging.getLogger(__name__)

# Nombre max d'awards affichés dans le panneau de détail
_MAX_AWARDS_DISPLAY = 8
# Nombre max de citations affichées
_MAX_CITATIONS_DISPLAY = 6
# Nombre de colonnes de la grille de médailles
_MEDALS_COLS_PER_ROW = 8


# =============================================================================
# Chargement de données
# =============================================================================


def _load_player_medals(main_db_path: str, match_id: str, xuid: str) -> list[dict[str, int]]:
    """Charge les médailles d'un joueur pour un match depuis la DB partagée."""
    if not main_db_path or not xuid:
        return []
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(main_db_path, xuid=xuid, read_only=True)
        return repo.load_match_medals(match_id)
    except Exception:
        logger.debug("Médailles indisponibles pour xuid=%s match=%s", xuid, match_id)
        return []


def _load_player_top_weapons(
    main_db_path: str, match_id: str, xuid: str
) -> list[tuple[int, int]]:
    """Charge les armes les plus utilisées par un joueur dans un match.

    Returns:
        Liste de (weapon_id, kills) triée par kills décroissant, max 5.
    """
    if not main_db_path or not xuid:
        return []
    try:
        import polars as pl

        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(main_db_path, xuid=xuid, read_only=True)
        df = repo.load_weapon_kills_for_player(xuid=xuid, match_ids=[match_id])
        if df.is_empty():
            return []
        top_df = (
            df.filter(pl.col("match_id") == match_id)
            .sort("kills", descending=True)
            .head(5)
        )
        wids = top_df["weapon_id"].to_list()
        kills = top_df["kills"].to_list()
        return list(zip(wids, kills))
    except Exception:
        logger.debug("Armes indisponibles pour xuid=%s match=%s", xuid, match_id)
        return []


def _load_player_db_enrichment(
    match_id: str, gamertag: str, xuid: str
) -> dict[str, Any]:
    """Charge l'enrichissement depuis la DB propre du joueur si disponible.

    Returns:
        Dict avec clés :
        - ``has_db``: bool
        - ``performance_score``: float | None
        - ``session_label``: str | None
        - ``awards``: list[dict]
        - ``citations``: list[dict]
    """
    result: dict[str, Any] = {
        "has_db": False,
        "performance_score": None,
        "session_label": None,
        "awards": [],
        "citations": [],
    }

    if not gamertag:
        return result

    player_db = get_player_db_path(gamertag)
    if not player_db.exists():
        return result

    result["has_db"] = True

    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(player_db) as conn:
            # Performance score + session
            try:
                row = conn.execute(
                    "SELECT performance_score, session_label"
                    " FROM player_match_enrichment WHERE match_id = ? LIMIT 1",
                    [match_id],
                ).fetchone()
                if row:
                    result["performance_score"] = row[0]
                    result["session_label"] = row[1]
            except Exception:
                pass

            # PersonalScoreAwards
            try:
                rows_awards = conn.execute(
                    "SELECT award_name, award_category, award_count, award_score"
                    " FROM personal_score_awards WHERE match_id = ?"
                    " ORDER BY award_score DESC",
                    [match_id],
                ).fetchall()
                result["awards"] = [
                    {
                        "award_name": r[0],
                        "award_category": r[1],
                        "award_count": r[2],
                        "award_score": r[3],
                    }
                    for r in rows_awards
                ]
            except Exception:
                pass

            # Citations
            try:
                rows_cit = conn.execute(
                    "SELECT citation_name_norm, value FROM match_citations WHERE match_id = ?",
                    [match_id],
                ).fetchall()
                result["citations"] = [{"name": r[0], "value": r[1]} for r in rows_cit]
            except Exception:
                pass

    except Exception:
        logger.debug("Enrichissement DB indisponible gamertag=%s", gamertag)

    return result


# =============================================================================
# Formatage
# =============================================================================


def _fmt_stat(key: str, value: Any) -> str:
    """Formate une valeur de stat pour l'affichage dans le panneau."""
    if value is None:
        return "—"
    if key == "kda":
        try:
            return f"{float(value):.2f}"
        except (ValueError, TypeError):
            return str(value)
    if key == "accuracy":
        try:
            v = float(value)
            if v <= 1.0:
                v *= 100.0
            return f"{v:.1f}\u202f%"
        except (ValueError, TypeError):
            return str(value)
    if key == "avg_life_seconds":
        try:
            secs = int(float(value))
            return f"{secs // 60}:{secs % 60:02d}"
        except (ValueError, TypeError):
            return str(value)
    if key in ("damage_dealt", "damage_taken"):
        try:
            return f"{int(round(float(value))):,}"
        except (ValueError, TypeError):
            return str(value)
    return str(value)


# =============================================================================
# Rendu du panneau de détail
# =============================================================================


def _render_stats_grid(player: dict[str, Any]) -> None:
    """Affiche les stats du joueur en grille de métriques."""
    col1, col2, col3, col4 = st.columns(4)

    with col1:
        st.metric(t("col_score"), _fmt_stat("score", player.get("score")))
        st.metric(t("col_rank"), _fmt_stat("rank", player.get("rank")))
        st.metric(t("col_kda"), _fmt_stat("kda", player.get("kda")))

    with col2:
        st.metric(t("col_kills"), _fmt_stat("kills", player.get("kills")))
        st.metric(t("col_deaths"), _fmt_stat("deaths", player.get("deaths")))
        st.metric(t("col_assists_short"), _fmt_stat("assists", player.get("assists")))

    with col3:
        st.metric(t("col_headshots"), _fmt_stat("headshots", player.get("headshot_kills")))
        st.metric(t("col_melee"), _fmt_stat("melee", player.get("melee_kills")))
        st.metric(t("col_power_weapon"), _fmt_stat("power", player.get("power_weapon_kills")))

    with col4:
        st.metric(
            t("col_accuracy"), _fmt_stat("accuracy", player.get("accuracy"))
        )
        st.metric(
            t("mv_scoreboard_avg_life"),
            _fmt_stat("avg_life_seconds", player.get("avg_life_seconds")),
        )
        st.metric(
            t("col_killing_spree"),
            _fmt_stat("spree", player.get("max_killing_spree")),
        )


def _render_damage_row(player: dict[str, Any]) -> None:
    """Affiche les stats de dégâts et de tirs."""
    c1, c2, c3 = st.columns(3)
    with c1:
        st.metric(
            t("col_dmg_dealt"), _fmt_stat("damage_dealt", player.get("damage_dealt"))
        )
    with c2:
        st.metric(
            t("col_dmg_taken"), _fmt_stat("damage_taken", player.get("damage_taken"))
        )
    with c3:
        shots_fired = player.get("shots_fired")
        shots_hit = player.get("shots_hit")
        if shots_fired and shots_hit:
            st.metric(
                t("col_shots_fired_short"),
                f"{_fmt_stat('shots_fired', shots_fired)} / {_fmt_stat('shots_hit', shots_hit)}",
            )
        else:
            st.metric(t("col_shots_fired_short"), "—")


def _render_weapons_section(
    main_db_path: str, match_id: str, xuid: str
) -> None:
    """Charge et affiche le top des armes utilisées."""
    weapons = _load_player_top_weapons(main_db_path, match_id, xuid)
    if not weapons:
        return

    lang = get_lang()
    st.caption(t("sb_detail_top_weapons"))
    parts = []
    for wid, kills in weapons:
        try:
            from src.analysis._weapon_data import resolve_weapon_display

            name = resolve_weapon_display(int(wid), lang=lang) or f"#{wid}"
        except Exception:
            name = f"#{wid}"
        parts.append(f"**{name}** ({kills})")
    st.write(" · ".join(parts))


def _render_medals_section(
    main_db_path: str, match_id: str, xuid: str
) -> None:
    """Charge et affiche la grille de médailles."""
    medals = _load_player_medals(main_db_path, match_id, xuid)
    if not medals:
        return

    st.caption(t("sb_detail_medals"))
    lang = get_lang()
    render_medals_grid(medals, cols_per_row=_MEDALS_COLS_PER_ROW, lang=lang)


def _render_player_db_section(enrichment: dict[str, Any]) -> None:
    """Affiche les données provenant de la DB propre du joueur."""
    if not enrichment.get("has_db"):
        return

    st.divider()
    st.caption(f"📊 {t('sb_detail_player_db')}")

    perf = enrichment.get("performance_score")
    session = enrichment.get("session_label")
    awards = enrichment.get("awards") or []
    citations = enrichment.get("citations") or []

    has_any = perf is not None or session or awards or citations
    if not has_any:
        st.caption(t("sb_detail_no_enrichment"))
        return

    c1, c2 = st.columns(2)
    with c1:
        if perf is not None:
            try:
                st.metric(t("col_performance"), f"{float(perf):.0f}")
            except (ValueError, TypeError):
                st.metric(t("col_performance"), str(perf))
    with c2:
        if session:
            st.metric(t("sb_detail_session"), str(session))

    # PersonalScoreAwards
    if awards:
        st.caption(t("sb_detail_awards"))
        award_parts = []
        for aw in sorted(awards, key=lambda a: a.get("award_score", 0), reverse=True)[:_MAX_AWARDS_DISPLAY]:
            name = str(aw.get("award_name") or aw.get("award_category") or "?")
            count = aw.get("award_count", 0)
            award_parts.append(f"**{name}** ×{count}")
        st.write(" · ".join(award_parts))

    # Citations
    if citations:
        st.caption(t("sb_detail_citations"))
        citation_parts = []
        for cit in citations[:_MAX_CITATIONS_DISPLAY]:
            name = str(cit.get("name") or "?").replace("_", " ").title()
            val = cit.get("value", 0)
            citation_parts.append(f"**{name}** ={val}")
        st.write(" · ".join(citation_parts))


# =============================================================================
# Expanders par joueur
# =============================================================================


def _team_label_for_player(tid: Any) -> str:
    """Retourne le label court de l'équipe d'un joueur."""
    if tid is None:
        return ""
    try:
        return TEAM_MAP.get(int(tid), f"Équipe {tid}")
    except (ValueError, TypeError):
        return str(tid)


def render_scoreboard_detail_expanders(
    *,
    players: list[dict[str, Any]],
    main_db_path: str,
    match_id: str,
    me_xuid: str,
) -> None:
    """Rend les accordéons de détail joueur sous le tableau des scores.

    Chaque accordéon porte le nom du joueur en titre. En l'ouvrant,
    l'utilisateur accède aux stats complètes, médailles et armes.
    Si le joueur possède sa propre DB, des données supplémentaires
    (performance, session, awards, citations) sont également affichées.

    Args:
        players: Liste des dicts joueur (tels que retournés par load_match_scoreboard).
        main_db_path: Chemin vers la DB du joueur courant (accès shared).
        match_id: Identifiant du match.
        me_xuid: XUID du joueur courant (mis en avant dans le titre).
    """
    if not players:
        return

    st.markdown(f"#### {t('sb_detail_section_title')}")

    for p in players:
        p_xu = str(
            parse_xuid_input(str(p.get("xuid") or "").strip())
            or str(p.get("xuid") or "").strip()
        ).strip()

        gamertag = str(p.get("gamertag") or "").strip() or "?"
        kills = p.get("kills") or 0
        deaths = p.get("deaths") or 0
        assists = p.get("assists") or 0
        team_label = _team_label_for_player(p.get("team_id"))

        is_me = bool(me_xuid and p_xu and p_xu == me_xuid)
        me_tag = " 🎮" if is_me else ""

        # Titre de l'accordéon : Gamertag — Équipe | K/D/A
        expander_label = (
            f"{gamertag}{me_tag}"
            f"{'  —  ' + team_label if team_label else ''}"
            f"  ·  {kills} / {deaths} / {assists}"
        )

        with st.expander(expander_label, expanded=False):
            _render_stats_grid(p)
            _render_damage_row(p)

            if p_xu and main_db_path:
                _render_weapons_section(main_db_path, match_id, p_xu)
                _render_medals_section(main_db_path, match_id, p_xu)

            # Enrichissement depuis la DB propre du joueur (si disponible)
            if gamertag and gamertag != "?":
                enrichment = _load_player_db_enrichment(match_id, gamertag, p_xu)
                _render_player_db_section(enrichment)
