"""Section joueurs pour la page Match View - Némésis et Roster."""

from __future__ import annotations

import contextlib
import html
import logging
import os
import re
from collections.abc import Callable
from typing import Any

import streamlit as st

from src.analysis import compute_personal_antagonists
from src.config import BOT_MAP, TEAM_MAP
from src.ui import display_name_from_xuid
from src.ui.i18n import get_lang, t
from src.ui.pages.match_view_helpers import os_card
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, fragment_if_available
from src.utils import parse_xuid_input
from src.visualization.match_impact_timeline import (
    compute_single_match_impact,
    get_impact_labels,
    plot_all_players_frags_timeline,
    plot_match_kill_death_timeline,
)
from src.visualization.team_dominance_timeline import (
    compute_dominance_buckets,
    detect_streaks,
    plot_dominance_chart,
)

logger = logging.getLogger(__name__)


# =============================================================================
# Helper DuckDB v4
# =============================================================================


def _is_duckdb_v4_path(db_path: str) -> bool:
    """Détecte si le chemin est une DB joueur DuckDB v4."""
    return db_path.endswith(".duckdb") if db_path else False


def _has_table_duckdb(db_path: str, table_name: str) -> bool:
    """Vérifie si une table existe dans une DB DuckDB (locale ou shared)."""
    if not _is_duckdb_v4_path(db_path):
        return False
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        # Vérifier la table locale puis shared (v5.1)
        return repo.has_table(table_name) or repo._has_shared_table(table_name)
    except Exception:
        return False


def _load_match_players_stats(db_path: str, match_id: str) -> list[dict[str, Any]]:
    """Charge les stats des joueurs d'un match."""
    if not _is_duckdb_v4_path(db_path):
        return []
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        return repo.load_match_players_stats(match_id)
    except Exception:
        return []


def _load_match_scoreboard(db_path: str, match_id: str) -> list[dict[str, Any]]:
    """Charge le tableau de bord complet (toutes stats + frags parfaits) d'un match."""
    if not _is_duckdb_v4_path(db_path):
        return []
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        return repo.load_match_scoreboard(match_id)
    except Exception:
        return []


# =============================================================================
# Section Dynamique du match (frise de dominance d'équipe)
# =============================================================================


@fragment_if_available
def render_team_dominance_section(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    is_firefight: bool,
    load_highlight_events_fn: Callable,
) -> None:
    """Rend la frise chronologique de dominance d'équipe (PvP uniquement).

    Affiche deux panneaux liés par l'axe temps :
    - Barres de dominance (tug-of-war) par tranche de 30s.
    - Kill feed individuel avec séries annotées.
    """
    if is_firefight:
        return

    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        return

    st.subheader(t("mv_match_dynamics"))

    with st.spinner(t("mv_dynamics_computing")):
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)

    if not he:
        st.info(t("mv_dynamics_no_data"))
        return

    # Mapping xuid → team_id + gamertag depuis match_participants
    all_players = _load_match_players_stats(db_path, match_id.strip())
    if not all_players:
        st.info(t("mv_dynamics_no_roster"))
        return

    me_xuid = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    xuid_to_team: dict[str, int] = {
        str(p.get("xuid", "")).strip(): int(p["team_id"])
        for p in all_players
        if p.get("team_id") is not None and p.get("xuid")
    }
    xuid_to_gamertag: dict[str, str] = {}
    for _p in all_players:
        _xu = str(_p.get("xuid", "")).strip()
        _gt = str(_p.get("gamertag") or "").strip()
        # Exclure les gamertags non résolus (XUID brut stocké à la place du nom)
        if _xu and _gt and _gt != _xu and not _gt.isdigit() and not _gt.lower().startswith("xuid("):
            xuid_to_gamertag[_xu] = _gt

    # Vérification : au moins 2 équipes distinctes (PvP)
    if len(set(xuid_to_team.values())) < 2:
        return

    my_team_id = xuid_to_team.get(me_xuid)
    if my_team_id is None:
        st.info(t("mv_dynamics_no_team"))
        return

    kill_events = [
        e
        for e in he
        if str(e.get("event_type", "")).lower() == "kill" and e.get("time_ms") is not None
    ]
    if not kill_events:
        st.info(t("mv_dynamics_no_kills"))
        return

    # Durée inférée depuis les events (+ buffer pour la dernière tranche)
    duration_s = max(int(e["time_ms"]) for e in kill_events) / 1000.0 + 20.0

    buckets = compute_dominance_buckets(he, xuid_to_team, my_team_id, duration_s)
    streaks = detect_streaks(he, xuid_to_team, xuid_to_gamertag)

    fig = plot_dominance_chart(
        buckets=buckets,
        streaks=streaks,
        kill_events=kill_events,
        xuid_to_team=xuid_to_team,
        my_team_id=my_team_id,
        duration_s=duration_s,
        height=360,
    )

    if fig is not None:
        st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        st.markdown(t("mv_dominance_legend"))
    else:
        st.info(t("mv_dynamics_no_dominance"))


# =============================================================================
# Section Némésis / Souffre-douleur
# =============================================================================


@fragment_if_available
def render_nemesis_section(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    colors: dict,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Rend la section Némésis / Souffre-douleur."""
    st.subheader(t("mv_antagonists_title"))
    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        st.caption(t("mv_nemesis_unavailable"))
        return

    with st.spinner(t("mv_highlight_loading")):
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)

    match_gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)

    me_xuid = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    # Sprint 3.3: Charger les stats officielles pour validation des antagonistes
    official_stats = _load_match_players_stats(db_path, match_id.strip())

    res = compute_personal_antagonists(
        he, me_xuid=me_xuid, tolerance_ms=5, official_stats=official_stats
    )
    if (res.nemesis is None) and (res.bully is None):
        st.info(t("mv_nemesis_no_data"))
        # On continue pour afficher le graphique Killer-Victim si des données existent

    def _debug_enabled() -> bool:
        env_flag = str(os.environ.get("OPENSPARTAN_DEBUG_ANTAGONISTS") or "").strip().lower()
        if env_flag in {"1", "true", "yes", "y", "on"}:
            return True

        env_flag2 = str(os.environ.get("OPENSPARTAN_DEBUG") or "").strip().lower()
        if env_flag2 in {"1", "true", "yes", "y", "on"}:
            return True

        try:
            if bool(st.session_state.get("ui_debug_antagonists", False)):
                return True
        except Exception:
            pass

        # Query params (compatible Streamlit récent + fallback expérimental)
        try:
            if hasattr(st, "query_params"):
                qp = st.query_params
                v = qp.get("debug_antagonists") or qp.get("debug")
            else:
                qp = st.experimental_get_query_params()
                v = (qp.get("debug_antagonists") or qp.get("debug") or [""])[0]
            if isinstance(v, list | tuple):
                v = v[0] if v else ""
            if str(v or "").strip().lower() in {"1", "true", "yes", "y", "on"}:
                return True
        except Exception:
            pass

        return False

    def _display_name_from_kv(xuid_value, gamertag_value) -> str:
        gt = str(gamertag_value or "").strip()
        xu_raw = str(xuid_value or "").strip()
        xu = parse_xuid_input(xu_raw) or xu_raw

        xu_key = str(xu).strip() if xu is not None else ""
        if xu_key and isinstance(match_gt_map, dict):
            mapped = match_gt_map.get(xu_key)
            if isinstance(mapped, str) and mapped.strip():
                return mapped.strip()

        if (not gt) or gt == "?" or gt.isdigit() or gt.lower().startswith("xuid("):
            if xu:
                return display_name_from_xuid(str(xu).strip())
            return "-"
        return gt

    # Afficher les cartes Némésis/Souffre-douleur uniquement si déterminés
    if (res.nemesis is not None) or (res.bully is not None):
        nemesis_name = "-"
        nemesis_killed_me: int | None = None
        nemesis_killed_me_approx = False
        me_killed_nemesis: int | None = None
        me_killed_nemesis_approx = False
        if res.nemesis is not None:
            nemesis_name = _display_name_from_kv(res.nemesis.xuid, res.nemesis.gamertag)
            nemesis_killed_me = int(res.nemesis.opponent_killed_me.total)
            nemesis_killed_me_approx = bool(res.nemesis.opponent_killed_me.has_estimated)
            me_killed_nemesis = int(res.nemesis.me_killed_opponent.total)
            me_killed_nemesis_approx = bool(res.nemesis.me_killed_opponent.has_estimated)

        bully_name = "-"
        bully_killed_me: int | None = None
        bully_killed_me_approx = False
        me_killed_bully: int | None = None
        me_killed_bully_approx = False
        if res.bully is not None:
            bully_name = _display_name_from_kv(res.bully.xuid, res.bully.gamertag)
            bully_killed_me = int(res.bully.opponent_killed_me.total)
            bully_killed_me_approx = bool(res.bully.opponent_killed_me.has_estimated)
            me_killed_bully = int(res.bully.me_killed_opponent.total)
            me_killed_bully_approx = bool(res.bully.me_killed_opponent.has_estimated)

        def _clean_name(v: str) -> str:
            s = str(v or "")
            s = s.replace("\ufffd", "")
            s = re.sub(r"[\x00-\x1f\x7f]", "", s)
            s = re.sub(r"\s+", " ", s).strip()
            return s or "-"

        nemesis_name = _clean_name(nemesis_name)
        bully_name = _clean_name(bully_name)

        def _cmp_color(deaths_: int | None, kills_: int | None) -> str:
            if deaths_ is None or kills_ is None:
                return colors["slate"]
            if int(deaths_) > int(kills_):
                return colors["red"]
            if int(deaths_) < int(kills_):
                return colors["green"]
            return colors["violet"]

        def _fmt_count(label: str, value: int | None, approx: bool) -> str:
            if value is None:
                return "-"
            prefix = "≈ " if approx else ""
            if label == "deaths":
                return t("mv_deaths_count", prefix=prefix, n=int(value))
            return t("mv_killed_count", prefix=prefix, n=int(value))

        def _fmt_two_lines(
            deaths_: int | None, deaths_approx: bool, kills_: int | None, kills_approx: bool
        ) -> str:
            d = _fmt_count("deaths", deaths_, deaths_approx)
            k = _fmt_count("kills", kills_, kills_approx)
            return html.escape(d) + "<br/>" + html.escape(k)

        c = st.columns(2)
        with c[0]:
            os_card(
                t("lbl_nemesis"),
                nemesis_name,
                _fmt_two_lines(
                    nemesis_killed_me,
                    nemesis_killed_me_approx,
                    me_killed_nemesis,
                    me_killed_nemesis_approx,
                ),
                accent=_cmp_color(nemesis_killed_me, me_killed_nemesis),
                sub_style="color: rgba(245, 248, 255, 0.92); font-weight: 800; font-size: 16px; line-height: 1.15;",
                min_h=110,
            )
        with c[1]:
            os_card(
                t("lbl_victim"),
                bully_name,
                _fmt_two_lines(
                    bully_killed_me, bully_killed_me_approx, me_killed_bully, me_killed_bully_approx
                ),
                accent=_cmp_color(bully_killed_me, me_killed_bully),
                sub_style="color: rgba(245, 248, 255, 0.92); font-weight: 800; font-size: 16px; line-height: 1.15;",
                min_h=110,
            )

    if _debug_enabled():
        deaths_missing = max(0, int(res.my_deaths_total) - int(res.my_deaths_assigned_total))
        deaths_est = max(0, int(res.my_deaths_assigned_total) - int(res.my_deaths_assigned_certain))
        kills_missing = max(0, int(res.my_kills_total) - int(res.my_kills_assigned_total))
        kills_est = max(0, int(res.my_kills_assigned_total) - int(res.my_kills_assigned_certain))

        # Sprint 3.3: Indicateur visuel de confiance
        validation_icon = "✓" if res.is_validated else "⚠"
        validation_label = t("lbl_validated") if res.is_validated else t("lbl_not_validated")

        st.caption(
            f"Debug antagonistes {validation_icon} {validation_label} — "
            f"Morts attribuées {res.my_deaths_assigned_total}/{res.my_deaths_total} "
            f"(certain {res.my_deaths_assigned_certain}, estimé {deaths_est}, manquantes {deaths_missing}) · "
            f"Kills attribués {res.my_kills_assigned_total}/{res.my_kills_total} "
            f"(certain {res.my_kills_assigned_certain}, estimé {kills_est}, manquants {kills_missing})"
        )

        # Sprint 3.3: Afficher validation_notes si présentes
        if res.validation_notes:
            st.caption(f"Validation: {res.validation_notes}")

    # Graphique barres empilées Killer-Victim (antagonist_charts)
    _render_antagonist_chart(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        load_match_gamertags_fn=load_match_gamertags_fn,
        highlight_events=he,
    )


def _display_name_for_chart(
    xuid: str,
    gamertag: str | None,
    gt_map: dict[str, str] | None,
) -> str:
    """Nom d'affichage pour le graphe killer-victime (même logique que le roster)."""
    xu_s = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    if xu_s:
        bot_key = xu_s.strip()
        if bot_key.lower().startswith("bid("):
            bot_name = BOT_MAP.get(bot_key)
            if isinstance(bot_name, str) and bot_name.strip():
                return bot_name.strip()

    if xu_s and isinstance(gt_map, dict):
        mapped = gt_map.get(xu_s)
        if isinstance(mapped, str) and mapped.strip():
            return mapped.strip()

    g = str(gamertag or "").strip()
    if g and g != "?" and (not g.isdigit()) and (not g.lower().startswith("xuid(")):
        return g

    if xu_s:
        return display_name_from_xuid(xu_s)
    return "-"


def _render_antagonist_chart(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    load_match_gamertags_fn: Callable | None = None,
    highlight_events: list | None = None,
) -> None:
    """Affiche le graphique des interactions Killer-Victim du match."""
    if not match_id or not match_id.strip():
        return

    gt_map = None
    if load_match_gamertags_fn is not None:
        try:
            gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)
        except Exception:
            gt_map = None

    pairs_df = None
    if db_path and str(db_path).endswith(".duckdb"):
        try:
            from src.data.repositories.duckdb_repo import DuckDBRepository

            repo = DuckDBRepository(db_path, str(xuid).strip())
            pairs_df = repo.load_killer_victim_pairs_as_polars(match_id=match_id.strip())
        except Exception:
            pairs_df = None

    # Fallback : construire depuis highlight_events
    if (
        pairs_df is None or (hasattr(pairs_df, "is_empty") and pairs_df.is_empty())
    ) and highlight_events:
        try:
            import polars as pl

            from src.analysis import compute_killer_victim_pairs

            kv_pairs = compute_killer_victim_pairs(highlight_events, tolerance_ms=5)
            if kv_pairs:
                pairs_df = pl.DataFrame(
                    {
                        "match_id": [match_id] * len(kv_pairs),
                        "killer_xuid": [p.killer_xuid for p in kv_pairs],
                        "killer_gamertag": [p.killer_gamertag or "?" for p in kv_pairs],
                        "victim_xuid": [p.victim_xuid for p in kv_pairs],
                        "victim_gamertag": [p.victim_gamertag or "?" for p in kv_pairs],
                        "kill_count": [1] * len(kv_pairs),
                        "time_ms": [p.time_ms for p in kv_pairs],
                    }
                )
        except Exception:
            pass

    if pairs_df is not None and not (hasattr(pairs_df, "is_empty") and pairs_df.is_empty()):
        try:
            import polars as pl

            from src.visualization.antagonist_charts import plot_killer_victim_stacked_bars

            # Enrichir les libellés avec la même résolution que le roster (gt_map, BOT_MAP, alias)
            killer_displays = [
                _display_name_for_chart(row[0], row[1], gt_map)
                for row in pairs_df.select("killer_xuid", "killer_gamertag").iter_rows()
            ]
            victim_displays = [
                _display_name_for_chart(row[0], row[1], gt_map)
                for row in pairs_df.select("victim_xuid", "victim_gamertag").iter_rows()
            ]
            pairs_df = pairs_df.with_columns(
                pl.Series("killer_gamertag", killer_displays),
                pl.Series("victim_gamertag", victim_displays),
            )

            # Rangs pour tri des lignes (match_participants après backfill)
            official_stats = _load_match_players_stats(db_path, match_id.strip())
            rank_by_xuid = (
                {s["xuid"]: s["rank"] for s in official_stats} if official_stats else None
            )

            me_xuid = str(
                parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()
            ).strip()
            try:
                fig = plot_killer_victim_stacked_bars(
                    pairs_df,
                    match_id=match_id,
                    me_xuid=me_xuid,
                    rank_by_xuid=rank_by_xuid,
                    title=t("mv_killer_victim_title"),
                    height=400,
                )
                if fig is not None:
                    st.plotly_chart(fig, width="stretch", config={"staticPlot": True})
                else:
                    st.info(t("mv_interactions_no_data"))
            except Exception as e:
                st.warning(t("error_chart", error=e))
        except Exception:
            pass


# =============================================================================
# Section Tableau des scores par équipe
# =============================================================================


def _get_scoreboard_cols() -> list[tuple[str, str]]:
    """Retourne les colonnes du scoreboard traduites."""
    return [
        (t("col_player"), "gamertag"),
        (t("col_rank"), "rank"),
        (t("col_score"), "score"),
        (t("col_kills"), "kills"),
        (t("col_deaths"), "deaths"),
        (t("col_assists_short"), "assists"),
        (t("col_kda"), "kda"),
        (t("col_killing_spree"), "max_killing_spree"),
        (t("col_headshots"), "headshot_kills"),
        (t("col_perfect_kills"), "perfect_kills"),
        (t("col_shots_fired_short"), "shots_fired"),
        (t("col_shots_hit"), "shots_hit"),
        (t("col_accuracy"), "accuracy"),
        (t("col_melee"), "melee_kills"),
        (t("col_power_weapon"), "power_weapon_kills"),
        (t("col_dmg_dealt"), "damage_dealt"),
        (t("col_dmg_taken"), "damage_taken"),
        (t("mv_scoreboard_avg_life"), "avg_life_seconds"),
    ]


# Colonnes non comparables (texte / ordinal) : pas de highlight min/max
_SB_SKIP_HIGHLIGHT: set[str] = {"gamertag", "rank"}

# Colonnes inversées : moins = mieux (vert), plus = pire (rouge)
_SB_INVERTED: set[str] = {"deaths", "damage_taken"}


def _sb_numeric_value(key: str, value: Any) -> float | None:
    """Extrait la valeur numérique comparable d'une cellule du scoreboard."""
    if value is None:
        return None
    try:
        v = float(value)
        # accuracy stockée en fraction 0-1 : normaliser pour comparaison
        if key == "accuracy" and v <= 1.0:
            v *= 100.0
        return v
    except (ValueError, TypeError):
        return None


def _compute_scoreboard_extremes(
    players: list[dict[str, Any]],
) -> dict[str, tuple[float, float]]:
    """Calcule le min et max de chaque colonne numérique sur tous les joueurs.

    Returns:
        Dict {key: (min_val, max_val)} pour les colonnes ayant au moins
        2 valeurs distinctes.
    """
    extremes: dict[str, tuple[float, float]] = {}
    for _, key in _get_scoreboard_cols():
        if key in _SB_SKIP_HIGHLIGHT:
            continue
        vals = [v for p in players if (v := _sb_numeric_value(key, p.get(key))) is not None]
        if len(vals) < 2:
            continue
        mn, mx = min(vals), max(vals)
        if mn == mx:
            continue  # toutes les valeurs identiques → pas de highlight
        extremes[key] = (mn, mx)
    return extremes


def _sb_cell_class(
    key: str,
    value: Any,
    extremes: dict[str, tuple[float, float]],
) -> str:
    """Retourne la classe CSS de highlight pour une cellule du scoreboard."""
    if key in _SB_SKIP_HIGHLIGHT or key not in extremes:
        return ""
    v = _sb_numeric_value(key, value)
    if v is None:
        return ""
    mn, mx = extremes[key]
    inverted = key in _SB_INVERTED
    if v == mx:
        return " os-sb-td--worst" if inverted else " os-sb-td--best"
    if v == mn:
        return " os-sb-td--best" if inverted else " os-sb-td--worst"
    return ""


def _fmt_scoreboard_cell(key: str, value: Any) -> str:
    """Formate une valeur selon son type pour l'affichage dans le scoreboard."""
    if value is None:
        return "—"
    if key == "gamertag":
        s = str(value).strip()
        # Masquer les XUIDs bruts : si tout numérique ou format xuid(…)
        if not s or s.isdigit() or s.lower().startswith("xuid(") or s == "?":
            return "—"
        return s
    if key == "kda":
        return f"{float(value):.2f}"
    if key == "accuracy":
        v = float(value)
        # Si la valeur est déjà en fraction (0–1), convertir en %
        if v <= 1.0:
            v *= 100.0
        return f"{v:.1f}\u202f%"
    if key == "avg_life_seconds":
        secs = int(float(value))
        return f"{secs // 60}:{secs % 60:02d}"
    if key in ("damage_dealt", "damage_taken"):
        return str(int(round(float(value))))
    return str(value)


def render_kd_timeline_section(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Affiche le graphe des frags cumulés de tous les joueurs au fil du match.

    Utilise les highlight_events (kills avec time_ms) pour tracer
    une courbe de frags par joueur. Même palette PLAYER_COLORS que le graphe
    éliminateur-victime.
    """
    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        return

    try:
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)
    except Exception:
        he = None

    if not he:
        return

    me_xu = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    # Charger le mapping gamertag (source fraîche)
    gt_map: dict[str, str] = {}
    with contextlib.suppress(Exception):
        gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key) or {}

    fig = plot_all_players_frags_timeline(
        he,
        me_xu,
        gt_map=gt_map,
        height=380,
    )
    if fig is not None:
        st.subheader(t("mv_kills_over_time"))
        st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)


def render_match_scoreboard(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    load_match_gamertags_fn: Callable,
) -> None:
    """Rend les tableaux de scores par équipe pour un match.

    Affiche un tableau par équipe présente dans le match, avec 18 colonnes
    de statistiques. La ligne du joueur principal est mise en évidence.

    Args:
        match_id: Identifiant du match.
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur principal.
        db_key: Clé de cache (conservée pour cohérence d'API).
        load_match_gamertags_fn: Fonction injectée de résolution gamertags —
            doit être la même que celle passée à render_roster_section.
            Interroge shared.match_participants + highlight_events + xuid_aliases
            (source de vérité v5.1). NE PAS utiliser display_name_from_xuid
            ni get_xuid_aliases (src/ui/aliases.py) qui sont obsolètes pour
            ce contexte et ignorent les données fraîches de la session.
    """
    st.subheader(t("mv_scoreboard"))

    players = _load_match_scoreboard(db_path, match_id.strip())
    if not players:
        st.info(t("mv_scoreboard_no_data"))
        return

    # Normaliser le xuid du joueur principal
    me_xu = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    # Charger le mapping gamertag (même source que render_roster_section)
    # SOURCE DE VÉRITÉ : shared.match_participants → highlight_events → shared.xuid_aliases
    # Ne pas substituer par display_name_from_xuid / get_xuid_aliases (obsolètes ici)
    gt_map: dict[str, str] = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key) or {}

    # Résoudre les gamertags manquants dans les données du scoreboard
    for p in players:
        xu = str(
            parse_xuid_input(str(p.get("xuid") or "").strip()) or str(p.get("xuid") or "").strip()
        ).strip()

        # Priorité 1 : bots (bid(...))
        if xu.lower().startswith("bid("):
            bot_name = BOT_MAP.get(xu)
            if isinstance(bot_name, str) and bot_name.strip():
                p["gamertag"] = bot_name.strip()
                continue

        # Priorité 2 : gt_map issu de load_match_gamertags_fn (source fraîche DuckDB v5)
        if xu and isinstance(gt_map, dict):
            mapped = gt_map.get(xu)
            if isinstance(mapped, str) and mapped.strip():
                p["gamertag"] = mapped.strip()
                continue

        # Priorité 3 : gamertag déjà présent dans les données (COALESCE SQL)
        gt = str(p.get("gamertag") or "").strip()
        if gt and gt != "?" and not gt.isdigit() and not gt.lower().startswith("xuid("):
            continue  # gamertag valide, on garde

        # Aucune résolution possible — afficher le XUID court ou "—"
        p["gamertag"] = None

    # Déterminer l'équipe du joueur principal (après résolution gamertags)
    my_team_id: Any = None
    for p in players:
        p_xu = str(
            parse_xuid_input(str(p.get("xuid") or "").strip()) or str(p.get("xuid") or "").strip()
        ).strip()
        if me_xu and p_xu and p_xu == me_xu:
            my_team_id = p.get("team_id")
            break

    # Grouper par team_id (ordre croissant). Les joueurs sans team_id connu sont regroupés en dernier.
    teams: dict[Any, list[dict[str, Any]]] = {}
    unknown_team: list[dict[str, Any]] = []
    for p in players:
        tid = p.get("team_id")
        if tid is None:
            unknown_team.append(p)
        else:
            teams.setdefault(tid, []).append(p)

    # Ajouter les joueurs sans team_id en dernier si présents
    if unknown_team:
        teams[None] = unknown_team

    # Comptage des équipes réelles (team_id non None)
    n_real_teams = len([t for t in teams if t is not None])

    # Min/max par colonne (toutes équipes confondues) pour highlight vert/rouge
    extremes = _compute_scoreboard_extremes(players)

    # Compter best/worst par joueur pour highlight ligne MVP/LVP
    player_best_count: dict[int, int] = {}  # index dans players → nb best
    player_worst_count: dict[int, int] = {}  # index dans players → nb worst
    for i, p in enumerate(players):
        best = 0
        worst = 0
        for _, key in _get_scoreboard_cols():
            cls = _sb_cell_class(key, p.get(key), extremes)
            if "best" in cls:
                best += 1
            elif "worst" in cls:
                worst += 1
        player_best_count[i] = best
        player_worst_count[i] = worst

    # Le joueur avec le plus de "best" = MVP, le plus de "worst" = LVP
    # Egalité MVP : moins de "worst" gagne, puis meilleur rang (plus bas)
    # Egalité LVP : moins de "best" gagne, puis pire rang (plus haut)
    # (au moins 2 colonnes pour éviter un highlight sur un seul stat)
    mvp_idx = (
        max(
            player_best_count,
            key=lambda i: (
                player_best_count[i],
                -player_worst_count.get(i, 0),
                -(players[i].get("rank") or 999),
            ),
        )
        if player_best_count
        else -1
    )
    lvp_idx = (
        max(
            player_worst_count,
            key=lambda i: (
                player_worst_count[i],
                -player_best_count.get(i, 0),
                (players[i].get("rank") or 0),
            ),
        )
        if player_worst_count
        else -1
    )
    if player_best_count.get(mvp_idx, 0) < 2:
        mvp_idx = -1
    if player_worst_count.get(lvp_idx, 0) < 2:
        lvp_idx = -1
    # Construire un set de xuids MVP/LVP pour lookup rapide
    mvp_xuid = str(players[mvp_idx].get("xuid", "")).strip() if mvp_idx >= 0 else ""
    lvp_xuid = str(players[lvp_idx].get("xuid", "")).strip() if lvp_idx >= 0 else ""

    for tid, team_players in teams.items():
        # Nom de l'équipe via TEAM_MAP
        try:
            raw_name = (
                TEAM_MAP.get(int(tid), t("mv_team_n", n=tid))
                if tid is not None
                else t("mv_team_unknown")
            )
        except (ValueError, TypeError):
            raw_name = t("mv_team_n", n=tid) if tid is not None else t("mv_team_unknown")

        # Détecter si c'est l'équipe du joueur pour la couleur Okabe-Ito
        is_my_team = tid == my_team_id
        team_css_mod = "os-sb-team--mine" if is_my_team else "os-sb-team--enemy"
        team_label = t("mv_team_label", name=html.escape(raw_name))

        # En-têtes colonnes
        sb_cols = _get_scoreboard_cols()
        th_cells = "".join(
            f"<th class='os-sb-th'>{html.escape(label)}</th>" for label, _ in sb_cols
        )
        n_cols = len(sb_cols)
        thead = (
            f"<thead>"
            f"<tr><th class='os-sb-team {team_css_mod}' colspan='{n_cols}'>{team_label}</th></tr>"
            f"<tr>{th_cells}</tr>"
            f"</thead>"
        )

        # Lignes joueurs
        body_rows = []
        for p in team_players:
            p_xu = str(
                parse_xuid_input(str(p.get("xuid") or "").strip())
                or str(p.get("xuid") or "").strip()
            ).strip()
            is_me = bool(me_xu and p_xu and p_xu == me_xu)
            row_class = " os-sb-row--me" if is_me else ""
            # MVP / LVP row highlight (s'applique aussi au joueur principal)
            if p_xu and p_xu == mvp_xuid:
                row_class += " os-sb-row--mvp"
            elif p_xu and p_xu == lvp_xuid:
                row_class += " os-sb-row--lvp"
            cells = "".join(
                f"<td class='os-sb-td{_sb_cell_class(key, p.get(key), extremes)}'>"
                f"{html.escape(_fmt_scoreboard_cell(key, p.get(key)))}</td>"
                for _, key in sb_cols
            )
            body_rows.append(f"<tr class='os-sb-row{row_class}'>{cells}</tr>")

        table_html = (
            "<div class='os-table-wrap os-sb-wrap'>"
            "<table class='os-table os-scoreboard'>"
            f"{thead}"
            "<tbody>" + "".join(body_rows) + "</tbody>"
            "</table>"
            "</div>"
        )
        st.markdown(table_html, unsafe_allow_html=True)

    # Note sur le rang : en mode équipe, le rang est individuel au sein de l'équipe
    if n_real_teams > 1:
        st.caption(t("mv_scoreboard_rank_note"))


# =============================================================================
# Section Roster
# =============================================================================


def render_roster_section(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    load_match_rosters_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Rend la section Joueurs (roster)."""
    st.subheader(t("mv_players_title"))
    rosters = load_match_rosters_fn(db_path, match_id.strip(), xuid.strip(), db_key=db_key)
    if not rosters:
        st.info(t("mv_roster_unavailable"))
        return

    gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)
    me_xu = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    my_team_id = rosters.get("my_team_id")
    my_team_name = rosters.get("my_team_name")
    enemy_team_ids = rosters.get("enemy_team_ids") or []
    enemy_team_names = rosters.get("enemy_team_names") or []

    def _team_label(team_id_value) -> str:
        try:
            tid = int(team_id_value)
        except Exception:
            return "-"
        return TEAM_MAP.get(tid) or f"Team {tid}"

    def _roster_name(xu: str, gt: str | None) -> str:
        xu_s = str(parse_xuid_input(str(xu or "").strip()) or str(xu or "").strip()).strip()

        if xu_s:
            bot_key = xu_s.strip()
            if bot_key.lower().startswith("bid("):
                bot_name = BOT_MAP.get(bot_key)
                if isinstance(bot_name, str) and bot_name.strip():
                    return bot_name.strip()

        if xu_s and isinstance(gt_map, dict):
            mapped = gt_map.get(xu_s)
            if isinstance(mapped, str) and mapped.strip():
                return mapped.strip()

        g = str(gt or "").strip()
        if g and g != "?" and (not g.isdigit()) and (not g.lower().startswith("xuid(")):
            return g

        if xu_s:
            return display_name_from_xuid(xu_s)
        return "-"

    my_rows = rosters.get("my_team") or []
    en_rows = rosters.get("enemy_team") or []

    my_names: list[tuple[str, bool]] = []
    en_names: list[tuple[str, bool]] = []

    for r in my_rows:
        xu = str(r.get("xuid") or "").strip()
        name = str(r.get("display_name") or "").strip() or _roster_name(xu, r.get("gamertag"))
        is_self = bool(me_xu and xu and (str(parse_xuid_input(xu) or xu).strip() == me_xu)) or bool(
            r.get("is_me")
        )
        my_names.append((name, is_self))

    for r in en_rows:
        xu = str(r.get("xuid") or "").strip()
        name = str(r.get("display_name") or "").strip() or _roster_name(xu, r.get("gamertag"))
        en_names.append((name, False))

    rows_n = max(len(my_names), len(en_names), 1)
    my_names += [("", False)] * (rows_n - len(my_names))
    en_names += [("", False)] * (rows_n - len(en_names))

    def _pill_html(name: str, *, side: str, is_self: bool) -> str:
        if not name:
            return "<span class='os-roster-empty'>—</span>"
        safe = html.escape(str(name))
        extra = " os-roster-pill--self" if is_self else ""
        return (
            f"<span class='os-roster-pill os-roster-pill--{side}{extra}'>"
            "<span class='os-roster-pill__dot'></span>"
            f"<span>{safe}</span>"
            "</span>"
        )

    body_rows = []
    for i in range(rows_n):
        n_me, is_self = my_names[i]
        n_en, _ = en_names[i]
        body_rows.append(
            "<tr>"
            f"<td>{_pill_html(n_me, side='me', is_self=is_self)}</td>"
            f"<td>{_pill_html(n_en, side='enemy', is_self=False)}</td>"
            "</tr>"
        )

    _my_team_display = html.escape(str(my_team_name or _team_label(my_team_id)))
    _my_count = len([n for n, _ in my_names if n])
    _enemy_raw = (
        enemy_team_names[0]
        if (
            isinstance(enemy_team_names, list)
            and len(enemy_team_names) == 1
            and enemy_team_names[0]
        )
        else (
            " / ".join([_team_label(t_id) for t_id in enemy_team_ids])
            if enemy_team_ids
            else t("mv_roster_opponents")
        )
    )
    _enemy_display = html.escape(str(_enemy_raw))
    _enemy_count = len([n for n, _ in en_names if n])

    st.markdown(
        "<div class='os-table-wrap os-roster-wrap'>"
        "<table class='os-table os-roster'>"
        "<thead><tr>"
        f"<th class='os-roster-th os-roster-th--me'>{t('mv_roster_my_team', name=_my_team_display, n=_my_count)}</th>"
        f"<th class='os-roster-th os-roster-th--enemy'>{t('mv_roster_enemy_team', name=_enemy_display, n=_enemy_count)}</th>"
        "</tr></thead>"
        "<tbody>" + "".join(body_rows) + "</tbody>"
        "</table>"
        "</div>",
        unsafe_allow_html=True,
    )


# =============================================================================
# Section Impact & Timeline
# =============================================================================


@fragment_if_available
def render_match_impact_section(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    outcome: int | None,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Rend la section Impact & Timeline pour un match unique.

    Affiche un graphe chronologique kills/deaths cumulées du joueur,
    avec annotations des événements d'impact (premier sang, finisseur,
    plus lent, première victime).
    """
    st.subheader(t("mv_impact_title"))

    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        st.caption(t("mv_impact_no_events"))
        return

    with st.spinner(t("mv_impact_computing")):
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)

    if not he:
        st.info(t("mv_impact_no_events_match"))
        return

    me_xuid = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()
    gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)

    # Identifier les événements d'impact
    impact_events = compute_single_match_impact(he, me_xuid, outcome=outcome)

    # Enrichir les gamertags via gt_map
    if gt_map and isinstance(gt_map, dict):
        enriched = []
        for ie in impact_events:
            resolved = gt_map.get(ie.xuid, ie.gamertag)
            if resolved and resolved != ie.gamertag:
                from src.visualization.match_impact_timeline import MatchImpactEvent

                ie = MatchImpactEvent(
                    event_type=ie.event_type,
                    xuid=ie.xuid,
                    gamertag=resolved,
                    time_ms=ie.time_ms,
                    is_me=ie.is_me,
                )
            enriched.append(ie)
        impact_events = enriched

    # Badges d'impact en colonnes
    if impact_events:
        _impact_labels = get_impact_labels(get_lang())
        badge_cols = st.columns(len(impact_events))
        for i, ie in enumerate(impact_events):
            label_info = _impact_labels.get(ie.event_type)
            if not label_info:
                continue
            icon, label_fr = label_info
            display_name = t("lbl_you") if ie.is_me else ie.gamertag
            accent = "#3DFFB5" if ie.is_me else "#FFB703"  # vert si moi, ambre sinon
            with badge_cols[i]:
                os_card(
                    f"{icon} {label_fr}",
                    display_name,
                    _format_time(ie.time_ms),
                    accent=accent,
                    kpi_color=accent,
                    min_h=80,
                )

    # Graphe timeline kills/deaths
    fig = plot_match_kill_death_timeline(
        he,
        me_xuid,
        impact_events,
        height=340,
    )
    if fig is not None:
        st.plotly_chart(fig, width="stretch", config={"staticPlot": True})
    else:
        st.info(t("mv_impact_too_few"))


def _format_time(ms: int) -> str:
    """Formate un timestamp ms en M:SS."""
    total_sec = max(0, ms // 1000)
    minutes = total_sec // 60
    seconds = total_sec % 60
    return f"{minutes}:{seconds:02d}"


# =============================================================================
# Exports publics
# =============================================================================

__all__ = [
    "render_team_dominance_section",
    "render_nemesis_section",
    "render_roster_section",
    "render_match_impact_section",
    "render_kd_timeline_section",
    "render_match_scoreboard",
]
