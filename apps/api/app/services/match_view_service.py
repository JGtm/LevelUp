"""Service Match View — construit la réponse complète `MatchViewResponse`.

Toutes les importations src.* sont lazy pour permettre le mocking en tests.
"""

from __future__ import annotations

import contextlib
import logging
from datetime import datetime
from typing import Any

from apps.api.app._db_helpers import FMT_DATETIME_FR, OUTCOME_LABELS
from apps.api.app._pure_bridge import load_medal_description_map, load_medal_name_maps
from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.filters import FilterContextInput
from apps.api.app.schemas.match_view import (
    AssociatedMediaItem,
    LastMatchResolveRequest,
    LastMatchResolveResponse,
    MatchCitation,
    MatchCitationsTab,
    MatchCombatTab,
    MatchHighlightEvent,
    MatchMedal,
    MatchMediaTab,
    MatchNemesisRow,
    MatchPersonalResult,
    MatchRosterRow,
    MatchScoreboardRow,
    MatchSummaryKpis,
    MatchSummaryTab,
    MatchTeamTab,
    MatchViewHeader,
    MatchViewRank,
    MatchViewResponse,
    MatchWeaponKill,
)

logger = logging.getLogger(__name__)

_OUTCOME_COLORS: dict[int, str] = {
    2: "#22c55e",
    3: "#ef4444",
    1: "#8b5cf6",
    4: "#8b5cf6",
}


# ---------------------------------------------------------------------------
# Point d'entrée principal
# ---------------------------------------------------------------------------


def get_match_view(player: PlayerContext, match_id: str) -> MatchViewResponse:
    """Construit la réponse complète Match View pour un match donné."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    with DuckDBRepository(
        player.db_path,
        xuid=player.xuid,
        shared_db_path=player.shared_db_path,
        metadata_db_path=player.metadata_db_path,
        read_only=True,
    ) as repo:
        row = _load_match_row(repo, match_id, player.xuid)
        had_bot, stored_perf, dominance_flag = repo.load_player_match_enrichment(match_id)
        gamertag_map = _load_gamertag_map(repo, match_id)

        header = _build_header(row, match_id, had_bot, stored_perf, dominance_flag)
        rank = _build_rank(repo, match_id)
        summary = _build_summary_tab(repo, match_id, row, player.xuid, player.metadata_db_path)
        combat = _build_combat_tab(repo, match_id, player.xuid)
        team = _build_team_tab(repo, match_id, player.xuid, gamertag_map)
        media = _build_media_tab(repo, match_id)
        citations = _build_citations_tab(
            repo, match_id, player.xuid, player.db_path, player.metadata_db_path
        )

    return MatchViewResponse(
        header=header,
        rank=rank,
        summary_tab=summary,
        combat_tab=combat,
        team_tab=team,
        media_tab=media,
        citations_tab=citations,
    )


# ---------------------------------------------------------------------------
# Header
# ---------------------------------------------------------------------------


def _load_match_row(repo: Any, match_id: str, xuid: str) -> dict[str, Any]:
    """Charge la ligne de base du match depuis shared.match_registry + match_participants."""
    try:
        conn = repo.conn
        row = conn.execute(
            """
            SELECT
                mr.match_id,
                mr.start_time,
                mr.duration_seconds,
                mr.map_id,
                COALESCE(mr.map_name_ui, mr.map_id) AS map_ui,
                COALESCE(mr.mode_name_ui, mr.category) AS mode_ui,
                COALESCE(mr.playlist_name_ui, '') AS playlist_label,
                mp.outcome AS outcome_code,
                mp.kills,
                mp.deaths,
                mp.assists,
                mp.score,
                mp.rank,
                mp.damage_dealt,
                mp.damage_taken,
                mp.shots_fired,
                mp.shots_hit,
                mp.average_life_seconds
            FROM shared.match_registry mr
            LEFT JOIN shared.match_participants mp
                ON mp.match_id = mr.match_id AND mp.xuid = ?
            WHERE mr.match_id = ?
            """,
            [xuid, match_id],
        ).fetchone()
    except Exception as exc:
        logger.debug("_load_match_row match=%s : %s", match_id, exc)
        return {}

    if not row:
        return {}

    cols = [
        "match_id",
        "start_time",
        "duration_seconds",
        "map_id",
        "map_ui",
        "mode_ui",
        "playlist_label",
        "outcome_code",
        "kills",
        "deaths",
        "assists",
        "score",
        "rank",
        "damage_dealt",
        "damage_taken",
        "shots_fired",
        "shots_hit",
        "average_life_seconds",
    ]
    return dict(zip(cols, row, strict=False))


def _build_header(
    row: dict[str, Any],
    match_id: str,
    had_bot: bool,
    stored_perf: float | None,
    dominance_flag: int | None,
) -> MatchViewHeader:
    """Construit le header de la réponse Match View."""
    start_time = row.get("start_time")
    if isinstance(start_time, str):
        try:
            start_time = datetime.fromisoformat(start_time)
        except ValueError:
            start_time = None

    start_time_label = ""
    if start_time:
        with contextlib.suppress(Exception):
            start_time_label = start_time.strftime(FMT_DATETIME_FR)

    outcome_code = row.get("outcome_code")
    if outcome_code is not None:
        try:
            outcome_code = int(outcome_code)
        except (TypeError, ValueError):
            outcome_code = None

    outcome_label = OUTCOME_LABELS.get(outcome_code, "-") if outcome_code else "-"
    outcome_color = _OUTCOME_COLORS.get(outcome_code, "#94a3b8") if outcome_code else "#94a3b8"

    perf_display = "-"
    perf_color = None
    if stored_perf is not None:
        perf_display = f"{stored_perf:.0f}"
        perf_color = _perf_color(stored_perf)

    return MatchViewHeader(
        match_id=match_id,
        start_time=start_time,
        start_time_label=start_time_label,
        outcome_code=outcome_code,
        outcome_label=outcome_label,
        outcome_color=outcome_color,
        score_label=str(row.get("score") or ""),
        dominance_flag=bool(dominance_flag),
        had_bot_teammate=had_bot,
        map_ui=row.get("map_ui") or "",
        map_id=row.get("map_id"),
        mode_ui=row.get("mode_ui") or "",
        playlist_label=row.get("playlist_label") or "",
        performance_display=perf_display,
        performance_color=perf_color,
    )


def _perf_color(score: float) -> str:
    if score >= 80:
        return "#22c55e"
    if score >= 60:
        return "#06b6d4"
    if score >= 40:
        return "#f59e0b"
    if score >= 20:
        return "#f97316"
    return "#ef4444"


# ---------------------------------------------------------------------------
# Rank
# ---------------------------------------------------------------------------


def _build_rank(repo: Any, match_id: str) -> MatchViewRank:
    """Construit la section rank (CSR ou LUSR) pour ce match."""
    try:
        skill = repo.load_match_skill_data(match_id)
        if skill:
            return MatchViewRank(
                rating_type="LUSR" if skill.get("lusr_rating") else "CSR",
                tier_label=skill.get("tier_label"),
                numeric_value=skill.get("csr") or skill.get("lusr_rating"),
                delta_value=skill.get("csr_delta") or skill.get("lusr_delta"),
            )
    except Exception as exc:
        logger.debug("_build_rank match=%s : %s", match_id, exc)
    return MatchViewRank()


# ---------------------------------------------------------------------------
# Summary tab
# ---------------------------------------------------------------------------


def _build_summary_tab(
    repo: Any,
    match_id: str,
    row: dict[str, Any],
    xuid: str,
    metadata_db_path: str,
) -> MatchSummaryTab:
    kills = _safe_int(row.get("kills"))
    deaths = _safe_int(row.get("deaths"))
    assists = _safe_int(row.get("assists"))

    kda: float | None = None
    if kills is not None and deaths is not None:
        kda = (kills + (assists or 0) * 0.33) / max(deaths, 1)
        kda = round(kda, 2)

    avg_life: str | None = None
    avg_life_sec = row.get("average_life_seconds")
    if avg_life_sec is not None:
        try:
            sec = int(float(avg_life_sec))
            avg_life = f"{sec // 60:02d}:{sec % 60:02d}"
        except (TypeError, ValueError):
            pass

    kpis = MatchSummaryKpis(
        kills=kills,
        deaths=deaths,
        assists=assists,
        kda=kda,
        damage_dealt=_safe_float(row.get("damage_dealt")),
        average_life=avg_life,
    )

    outcome_code = row.get("outcome_code")
    if outcome_code is not None:
        try:
            outcome_code = int(outcome_code)
        except (TypeError, ValueError):
            outcome_code = None
    personal_result = MatchPersonalResult(
        outcome_label=OUTCOME_LABELS.get(outcome_code, "-") if outcome_code else "-",
        outcome_color=_OUTCOME_COLORS.get(outcome_code, "#94a3b8") if outcome_code else "#94a3b8",
        score=_safe_int(row.get("score")),
        rank_in_team=_safe_int(row.get("rank")),
    )

    medals = _load_medals(repo, match_id, metadata_db_path)

    return MatchSummaryTab(
        kpis=kpis,
        personal_result=personal_result,
        medals=medals,
        citations=[],
    )


def _load_medals(repo: Any, match_id: str, metadata_db_path: str) -> list[MatchMedal]:
    """Charge et enrichit les médailles du joueur pour ce match."""
    try:
        raw = repo.load_match_medals(match_id)
    except Exception as exc:
        logger.debug("_load_medals match=%s : %s", match_id, exc)
        return []

    if not raw:
        return []

    name_map: dict[int, str] = {}
    desc_map: dict[int, str] = {}
    try:
        name_map_raw, _ = load_medal_name_maps("fr")
        name_map = {int(k): v for k, v in (name_map_raw or {}).items()}
        desc_raw = load_medal_description_map("fr")
        desc_map = {int(k): v for k, v in (desc_raw or {}).items()}
    except Exception as exc:
        logger.debug("_load_medals : erreur chargement noms %s", exc)

    result: list[MatchMedal] = []
    for item in raw:
        name_id = int(item.get("name_id", 0))
        count = int(item.get("count", 1))
        name = name_map.get(name_id, str(name_id))
        desc = desc_map.get(name_id)
        result.append(MatchMedal(medal_name_id=name_id, name=name, count=count, description=desc))
    return result


# ---------------------------------------------------------------------------
# Combat tab
# ---------------------------------------------------------------------------


def _build_combat_tab(repo: Any, match_id: str, xuid: str) -> MatchCombatTab:
    weapon_kills = _load_weapon_kills(repo, match_id, xuid)
    highlight_events = _load_highlight_events(repo, match_id, xuid)
    return MatchCombatTab(weapon_kills=weapon_kills, highlight_events=highlight_events, charts=[])


def _load_weapon_kills(repo: Any, match_id: str, xuid: str) -> list[MatchWeaponKill]:
    """Charge les kills par arme et résout les labels."""
    try:
        df = repo.load_weapon_kills_for_player(xuid, match_ids=[match_id])
    except Exception as exc:
        logger.debug("_load_weapon_kills match=%s : %s", match_id, exc)
        return []

    if df is None or df.is_empty():
        return []

    label_map: dict[int, str] = {}
    try:
        from src.analysis._weapon_data import resolve_weapon_display

        for wid in df["weapon_id"].to_list():
            label_map[int(wid)] = resolve_weapon_display(int(wid), lang="fr")
    except Exception as exc:
        logger.debug("_load_weapon_kills : erreur résolution labels %s", exc)

    result: list[MatchWeaponKill] = []
    for item in df.to_dicts():
        wid = int(item.get("weapon_id", 0))
        kills = int(item.get("kills", 0))
        if kills <= 0:
            continue
        result.append(
            MatchWeaponKill(
                weapon_id=wid,
                weapon_label=label_map.get(wid, str(wid)),
                kill_count=kills,
            )
        )
    result.sort(key=lambda w: w.kill_count, reverse=True)
    return result


def _load_highlight_events(repo: Any, match_id: str, xuid: str) -> list[MatchHighlightEvent]:
    """Charge les highlight events pour le joueur sur ce match."""
    try:
        raw = repo.load_highlight_events(match_id)
    except Exception as exc:
        logger.debug("_load_highlight_events match=%s : %s", match_id, exc)
        return []

    # Filtrer uniquement les événements du joueur courant
    player_events = [e for e in raw if str(e.get("xuid") or "") == xuid]

    return [
        MatchHighlightEvent(
            event_time_ms=e.get("time_ms"),
            event_type=e.get("event_type") or "",
            actor_xuid=e.get("xuid"),
            target_xuid=None,
        )
        for e in player_events
    ]


# ---------------------------------------------------------------------------
# Team tab
# ---------------------------------------------------------------------------


def _load_gamertag_map(repo: Any, match_id: str) -> dict[str, str]:
    try:
        return repo.load_match_player_gamertags(match_id)
    except Exception:
        return {}


def _build_team_tab(
    repo: Any, match_id: str, xuid: str, gamertag_map: dict[str, str]
) -> MatchTeamTab:
    roster = _build_roster(repo, match_id, xuid)
    scoreboard = _build_scoreboard(repo, match_id, xuid)
    nemesis = _build_nemesis(repo, match_id, xuid, gamertag_map)
    return MatchTeamTab(roster=roster, scoreboard=scoreboard, nemesis=nemesis, encounters=[])


def _build_roster(repo: Any, match_id: str, xuid: str) -> list[MatchRosterRow]:
    try:
        raw = repo.load_match_players_stats(match_id)
    except Exception as exc:
        logger.debug("_build_roster match=%s : %s", match_id, exc)
        return []

    result: list[MatchRosterRow] = []
    for p in raw:
        team_id = p.get("team_id")
        kda: float | None = None
        kills = _safe_int(p.get("kills"))
        deaths = _safe_int(p.get("deaths"))
        assists = _safe_int(p.get("assists"))
        if kills is not None and deaths is not None:
            kda = round((kills + (assists or 0) * 0.33) / max(deaths, 1), 2)
        result.append(
            MatchRosterRow(
                xuid=str(p.get("xuid") or ""),
                gamertag=str(p.get("gamertag") or p.get("xuid") or ""),
                team_side=f"team_{team_id}" if team_id is not None else None,
                is_me=(str(p.get("xuid") or "") == xuid),
                kills=kills,
                deaths=deaths,
                assists=assists,
                kda=kda,
            )
        )
    return result


def _build_scoreboard(repo: Any, match_id: str, xuid: str) -> list[MatchScoreboardRow]:
    try:
        raw = repo.load_match_scoreboard(match_id)
    except Exception as exc:
        logger.debug("_build_scoreboard match=%s : %s", match_id, exc)
        return []

    result: list[MatchScoreboardRow] = []
    for p in raw:
        shots_fired = _safe_int(p.get("shots_fired"))
        shots_hit = _safe_int(p.get("shots_hit"))
        shots_accuracy: float | None = None
        if shots_fired and shots_hit:
            shots_accuracy = round(shots_hit / shots_fired * 100, 1)

        avg_life: str | None = None
        avg_life_sec = p.get("avg_life_seconds")
        if avg_life_sec is not None:
            try:
                sec = int(float(avg_life_sec))
                avg_life = f"{sec // 60:02d}:{sec % 60:02d}"
            except (TypeError, ValueError):
                pass

        outcome_code = _safe_int(p.get("outcome"))
        result.append(
            MatchScoreboardRow(
                xuid=str(p.get("xuid") or ""),
                gamertag=str(p.get("gamertag") or p.get("xuid") or ""),
                team_side=f"team_{p.get('team_id')}" if p.get("team_id") is not None else None,
                is_me=(str(p.get("xuid") or "") == xuid),
                rank=_safe_int(p.get("rank")),
                kills=_safe_int(p.get("kills")),
                deaths=_safe_int(p.get("deaths")),
                assists=_safe_int(p.get("assists")),
                betrayals=_safe_int(p.get("betrayals")),
                suicides=_safe_int(p.get("suicides")),
                shots_fired=shots_fired,
                shots_hit=shots_hit,
                shots_accuracy=shots_accuracy,
                damage_dealt=_safe_float(p.get("damage_dealt")),
                damage_taken=_safe_float(p.get("damage_taken")),
                average_life=avg_life,
                objectives_stolen=_safe_int(p.get("objectives_stolen")),
                headshot_kills=_safe_int(p.get("headshot_kills")),
                max_killing_spree=_safe_int(p.get("max_killing_spree")),
                perfect_kills=_safe_int(p.get("perfect_kills")),
                power_weapon_kills=_safe_int(p.get("power_weapon_kills")),
                melee_kills=_safe_int(p.get("melee_kills")),
                outcome_label=OUTCOME_LABELS.get(outcome_code, "-") if outcome_code else "-",
            )
        )
    return result


def _build_nemesis(
    repo: Any, match_id: str, xuid: str, gamertag_map: dict[str, str]
) -> list[MatchNemesisRow]:
    """Charge les paires killer/victim pour ce match (focus sur le joueur)."""
    try:
        conn = repo.conn
        rows = conn.execute(
            """
            SELECT killer_xuid, victim_xuid, COUNT(*) AS n
            FROM shared.killer_victim_pairs
            WHERE match_id = ?
              AND (killer_xuid = ? OR victim_xuid = ?)
            GROUP BY killer_xuid, victim_xuid
            """,
            [match_id, xuid, xuid],
        ).fetchall()
    except Exception as exc:
        logger.debug("_build_nemesis match=%s : %s", match_id, exc)
        return []

    killed_by: dict[str, int] = {}
    i_killed: dict[str, int] = {}
    for killer, victim, n in rows:
        killer, victim = str(killer or ""), str(victim or "")
        if killer == xuid:
            i_killed[victim] = i_killed.get(victim, 0) + n
        elif victim == xuid:
            killed_by[killer] = killed_by.get(killer, 0) + n

    opponents = set(killed_by) | set(i_killed)
    result: list[MatchNemesisRow] = []
    for opp_xuid in opponents:
        gt = gamertag_map.get(opp_xuid, opp_xuid)
        result.append(
            MatchNemesisRow(
                xuid=opp_xuid,
                gamertag=gt,
                killed_me=killed_by.get(opp_xuid, 0),
                i_killed=i_killed.get(opp_xuid, 0),
            )
        )
    result.sort(key=lambda r: r.killed_me, reverse=True)
    return result[:10]


# ---------------------------------------------------------------------------
# Media tab
# ---------------------------------------------------------------------------


def _build_media_tab(repo: Any, match_id: str) -> MatchMediaTab:
    """Charge les médias associés au match depuis media_match_associations."""
    try:
        conn = repo.conn
        rows = conn.execute(
            """
            SELECT mf.file_path, mf.file_name, mf.thumbnail_path, mf.mtime,
                   mma.match_start_time
            FROM media_match_associations mma
            JOIN media_files mf ON mma.media_path = mf.file_path
            WHERE mma.match_id = ?
            ORDER BY mf.mtime ASC
            """,
            [match_id],
        ).fetchall()
    except Exception as exc:
        logger.debug("_build_media_tab match=%s : %s", match_id, exc)
        return MatchMediaTab(media_items=[])

    items: list[AssociatedMediaItem] = []
    for file_path, file_name, thumbnail_path, _mtime, capture_time in rows:
        capture_dt = None
        if capture_time:
            try:
                if isinstance(capture_time, str):
                    capture_dt = datetime.fromisoformat(capture_time)
                else:
                    capture_dt = capture_time
            except Exception:
                pass

        import hashlib

        file_id = hashlib.md5(str(file_path).encode()).hexdigest()[:12]
        items.append(
            AssociatedMediaItem(
                file_id=file_id,
                file_name=file_name or "",
                file_path=str(file_path or ""),
                thumbnail_url=str(thumbnail_path) if thumbnail_path else None,
                capture_time=capture_dt,
            )
        )

    return MatchMediaTab(media_items=items)


# ---------------------------------------------------------------------------
# Citations tab
# ---------------------------------------------------------------------------


def _build_citations_tab(
    repo: Any, match_id: str, xuid: str, db_path: str, metadata_db_path: str
) -> MatchCitationsTab:
    """Charge les citations (commendations progressées) pour ce match."""
    citations: list[MatchCitation] = []

    try:
        from src.analysis.citations.engine import CitationEngine
        from src.data.citation_definitions import load_citation_definitions

        defs = load_citation_definitions()
        if defs:
            engine = CitationEngine(db_path, xuid)
            delta_map = engine.aggregate_for_display(match_ids=[match_id])
            active = {norm: val for norm, val in delta_map.items() if val > 0}

            for norm, val in active.items():
                defn = defs.get(norm, {}) if isinstance(defs, dict) else {}
                label = defn.get("label") or norm
                color = defn.get("color")
                citations.append(
                    MatchCitation(key=norm, label=str(label), color=color, value=float(val))
                )
    except Exception as exc:
        logger.debug("_build_citations_tab match=%s : %s", match_id, exc)

    medals = _load_medals(repo, match_id, metadata_db_path)

    return MatchCitationsTab(commendations=citations, medals=medals)


# ---------------------------------------------------------------------------
# Last Match — resolve
# ---------------------------------------------------------------------------


def resolve_last_match(
    player: PlayerContext, request: LastMatchResolveRequest
) -> LastMatchResolveResponse:
    """Résout le match courant dans le scope filtré (dernier match ou navigation)."""
    # Charger la liste des match_ids ordonnés par date (du plus récent au plus ancien)
    match_ids = _load_scoped_match_ids(player, request.filters)

    total = len(match_ids)
    if total == 0:
        raise ValueError("no_matches_in_scope")

    # Index courant (par défaut le dernier match = index 0 = plus récent)
    current_index = request.current_index if request.current_index is not None else 0
    current_index = max(0, min(current_index, total - 1))

    current_match_id = match_ids[current_index]
    prev_id = match_ids[current_index - 1] if current_index > 0 else None
    next_id = match_ids[current_index + 1] if current_index < total - 1 else None

    session_key = _make_session_key(request.filters)

    return LastMatchResolveResponse(
        current_match_id=current_match_id,
        total_matches_in_scope=total,
        current_index=current_index,
        previous_match_id=prev_id,
        next_match_id=next_id,
        session_tracking_key=session_key,
    )


def _load_scoped_match_ids(player: PlayerContext, filters: FilterContextInput) -> list[str]:
    """Charge les match_ids correspondant au scope filtré, du plus récent au plus ancien."""
    from apps.api.app.services.match_history_service import _load_matches_polars

    try:
        df = _load_matches_polars(player, filters)
        if df.is_empty():
            return []
        # Tri du plus récent au plus ancien
        import polars as pl

        if "start_time" in df.columns:
            df = df.sort("start_time", descending=True)
        return df["match_id"].cast(pl.Utf8).to_list()
    except Exception as exc:
        logger.debug("_load_scoped_match_ids : %s", exc)
        return []


def _make_session_key(filters: FilterContextInput) -> str:
    """Génère une clé de suivi de session à partir du scope du filtre."""
    import hashlib
    import json

    raw = json.dumps(filters.model_dump(mode="json"), sort_keys=True, default=str)
    return hashlib.md5(raw.encode()).hexdigest()[:12]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _safe_int(v: Any) -> int | None:
    if v is None:
        return None
    try:
        return int(v)
    except (TypeError, ValueError):
        return None


def _safe_float(v: Any) -> float | None:
    if v is None:
        return None
    try:
        return float(v)
    except (TypeError, ValueError):
        return None
