"""Analyse killer → victime à partir des highlight events (film).

Les highlight events (SPNKr) fournissent typiquement des events 'kill' et 'death'
avec un timestamp en ms depuis le début du match, mais sans lien direct
killer→victim. L'approche consiste à joindre:
- chaque kill event (t)
- avec un death event (t')
avec |t - t'| <= tolérance.

Référence: discussions den.dev / SPNKr (jointure kill/death ~ 5ms).

Ce module est un hub de réexport. L'implémentation est répartie dans :
- ``_kv_types.py`` : Dataclasses, TypedDict, helpers de coercition, validation
- ``_killer_victim_polars.py`` : Fonctions d'analyse Polars (antagonistes, timeseries, duels)
"""

from __future__ import annotations

from bisect import bisect_left, bisect_right
from collections.abc import Iterable
from typing import TYPE_CHECKING, Any

from src.analysis._killer_victim_polars import (  # noqa: F401
    AntagonistsResultPolars,
    compute_duel_history_polars,
    compute_kd_timeseries_by_minute_polars,
    compute_personal_antagonists_from_pairs_polars,
    killer_victim_counts_long_polars,
    killer_victim_matrix_polars,
)
from src.analysis._kv_types import (
    AntagonistsResult,
    EstimatedCount,
    KVPair,
    OpponentDuel,
    ValidationResult,
    _coerce_int,
    _coerce_str,
    _infer_event_type,
    get_player_rank,
    validate_and_adjust_pairs,
)

if TYPE_CHECKING:
    from src.analysis._kv_types import MatchPlayerStats


def _xuid_sort_key(xuid_value: str) -> tuple[int, str]:
    """Clé de tri stable pour XUIDs (numérique d'abord, puis alpha)."""
    s = str(xuid_value or "").strip()
    try:
        return (0, f"{int(s):020d}")
    except Exception:
        return (1, s)


def _choose_best(
    candidates: list[str],
    prefer: dict[str, int],
    rank_by_xuid: dict[str, int],
) -> str:
    """Choisit le meilleur candidat parmi les ambigus.

    Heuristiques (dans l'ordre):
    1. Privilégie l'adversaire déjà le plus fréquent en "certain" (Pass 1)
    2. Tie-breaker par rang dans le match (meilleur classement = priorité)
    3. Fallback stable: plus petit XUID numérique
    """
    if not candidates:
        return ""
    best_score = max(int(prefer.get(c, 0)) for c in candidates)
    tied = [c for c in candidates if int(prefer.get(c, 0)) == best_score]
    if len(tied) == 1:
        return tied[0]
    if rank_by_xuid:
        tied.sort(key=lambda x: (rank_by_xuid.get(x, 999), _xuid_sort_key(x)))
    else:
        tied.sort(key=_xuid_sort_key)
    return tied[0]


def _assign_antagonist_counts(  # noqa: PLR0913
    my_events: list[tuple[int, str, str]],
    other_events: list[tuple[int, str, str]],
    other_times: list[int],
    tolerance_ms: int,
    rank_by_xuid: dict[str, int],
    *,
    exclude_self: str = "",
) -> tuple[dict[str, int], dict[str, int], set[int]]:
    """Pass 1 (certain) + Pass 2 (estimé) pour un sens killer→victim.

    Retourne (certain_counts, estimated_counts, used_other_indices).
    """
    used_idx: set[int] = set()
    certain: dict[str, int] = {}
    estimated: dict[str, int] = {}
    pending: list[tuple[int, list[int]]] = []

    for t_evt, _xu, _gt in my_events:
        lo = bisect_left(other_times, t_evt - tolerance_ms)
        hi = bisect_right(other_times, t_evt + tolerance_ms)
        cand_idx = [i for i in range(lo, hi) if i not in used_idx]
        if len(cand_idx) == 1:
            i = cand_idx[0]
            used_idx.add(i)
            xu = str(other_events[i][1] or "").strip()
            if xu and xu != exclude_self:
                certain[xu] = int(certain.get(xu, 0)) + 1
        elif len(cand_idx) > 1:
            pending.append((t_evt, cand_idx))

    for _t_evt, cand_idx in pending:
        cand_idx2 = [i for i in cand_idx if i not in used_idx]
        if not cand_idx2:
            continue
        candidates = [str(other_events[i][1] or "").strip() for i in cand_idx2]
        candidates = [c for c in candidates if c and c != exclude_self]
        if not candidates:
            continue
        chosen = _choose_best(candidates, certain, rank_by_xuid)
        if not chosen:
            continue
        chosen_idxs = [i for i in cand_idx2 if str(other_events[i][1] or "").strip() == chosen]
        if chosen_idxs:
            used_idx.add(min(chosen_idxs))
            estimated[chosen] = int(estimated.get(chosen, 0)) + 1

    return certain, estimated, used_idx


def _select_top_antagonist(
    certain: dict[str, int],
    estimated: dict[str, int],
) -> str | None:
    """Retourne le XUID de l'antagoniste principal (total le plus élevé)."""
    keys = set(certain.keys()) | set(estimated.keys())
    if not keys:
        return None
    best: str | None = None
    best_tuple: tuple[int, int, tuple[int, str]] | None = None
    for x in keys:
        t = (
            int(certain.get(x, 0)) + int(estimated.get(x, 0)),
            int(certain.get(x, 0)),
            _xuid_sort_key(x),
        )
        if (
            best_tuple is None
            or t[0] > best_tuple[0]
            or (
                t[0] == best_tuple[0]
                and (t[1] > best_tuple[1] or (t[1] == best_tuple[1] and t[2] < best_tuple[2]))
            )
        ):
            best_tuple = t
            best = x
    return best


def _build_duel(  # noqa: PLR0913
    op_xu: str | None,
    gt_by_xuid: dict[str, str],
    killed_me_c: dict[str, int],
    killed_me_e: dict[str, int],
    bully_c: dict[str, int],
    bully_e: dict[str, int],
) -> OpponentDuel | None:
    """Construit l'OpponentDuel pour un antagoniste donné."""
    if not op_xu:
        return None

    def _ec(map_c: dict[str, int], map_e: dict[str, int], key: str) -> EstimatedCount:
        return EstimatedCount(int(map_c.get(key, 0)), int(map_e.get(key, 0)))

    return OpponentDuel(
        xuid=op_xu,
        gamertag=gt_by_xuid.get(op_xu, ""),
        opponent_killed_me=_ec(killed_me_c, killed_me_e, op_xu),
        me_killed_opponent=_ec(bully_c, bully_e, op_xu),
    )


def _validate_against_official(
    me: str,
    my_kills_assigned: int,
    my_deaths_assigned: int,
    official_stats: list[MatchPlayerStats] | None,
) -> tuple[bool, str]:
    """Valide les compteurs reconstitués vs les stats officielles."""
    if not official_stats:
        return False, "Pas de stats officielles pour validation"

    def _get_xuid(s: Any) -> str:
        return s.xuid if hasattr(s, "xuid") else s["xuid"]

    def _get_stat(s: Any, key: str) -> int:
        return getattr(s, key, 0) if hasattr(s, key) else s.get(key, 0)

    my_official = next((s for s in official_stats if _get_xuid(s) == me), None)
    if not my_official:
        return False, "Stats officielles du joueur non trouvées"

    off_kills = _get_stat(my_official, "kills")
    off_deaths = _get_stat(my_official, "deaths")
    kills_diff = my_kills_assigned - off_kills
    deaths_diff = my_deaths_assigned - off_deaths

    if kills_diff == 0 and deaths_diff == 0:
        return True, "Cohérent avec stats officielles"

    notes = []
    if kills_diff != 0:
        notes.append(f"kills: {my_kills_assigned} vs {off_kills} ({kills_diff:+d})")
    if deaths_diff != 0:
        notes.append(f"deaths: {my_deaths_assigned} vs {off_deaths} ({deaths_diff:+d})")
    return False, "Écarts: " + ", ".join(notes)


def compute_killer_victim_pairs(  # noqa: C901, PLR0912
    events: Iterable[dict[str, Any]],
    *,
    tolerance_ms: int = 5,
) -> list[KVPair]:
    """Construit les paires killer→victim à partir des highlight events.

    Stratégie:
    - sépare les kills et deaths
    - trie les deaths par time_ms
    - pour chaque kill, cherche les deaths dans [t-tol, t+tol]
      et choisit le death le plus proche (en évitant de réutiliser le même death)

    Args:
        events: liste de dicts (un event par entrée)
        tolerance_ms: fenêtre de jointure en millisecondes

    Returns:
        Liste de KVPair (killer, victim, time_ms).
    """

    if tolerance_ms < 0:
        tolerance_ms = 0

    kills: list[tuple[int, dict[str, Any]]] = []
    deaths: list[tuple[int, dict[str, Any]]] = []

    for e in events:
        if not isinstance(e, dict):
            continue
        et = _infer_event_type(e)
        t = _coerce_int(e.get("time_ms"))
        if t is None:
            continue
        if et == "kill":
            kills.append((t, e))
        elif et == "death":
            deaths.append((t, e))

    if not kills or not deaths:
        return []

    kills.sort(key=lambda x: x[0])
    deaths.sort(key=lambda x: x[0])

    death_times = [t for t, _ in deaths]
    used_death_idx: set[int] = set()

    out: list[KVPair] = []

    for t_kill, kill_event in kills:
        lo = bisect_left(death_times, t_kill - tolerance_ms)
        hi = bisect_right(death_times, t_kill + tolerance_ms)
        if lo >= hi:
            continue

        best_idx: int | None = None
        best_delta: int | None = None
        for idx in range(lo, hi):
            if idx in used_death_idx:
                continue
            delta = abs(death_times[idx] - t_kill)
            if best_delta is None or delta < best_delta:
                best_delta = delta
                best_idx = idx

        if best_idx is None:
            continue

        used_death_idx.add(best_idx)
        victim_event = deaths[best_idx][1]

        killer_xuid = _coerce_str(kill_event.get("xuid")) or ""
        victim_xuid = _coerce_str(victim_event.get("xuid")) or ""
        killer_gt = _coerce_str(kill_event.get("gamertag")) or killer_xuid or "?"
        victim_gt = _coerce_str(victim_event.get("gamertag")) or victim_xuid or "?"

        if not killer_xuid or not victim_xuid:
            # On garde quand même la paire si les gamertags existent.
            pass

        out.append(
            KVPair(
                killer_xuid=killer_xuid,
                killer_gamertag=killer_gt,
                victim_xuid=victim_xuid,
                victim_gamertag=victim_gt,
                time_ms=int(t_kill),
            )
        )

    return out


def _parse_events(
    events: Iterable[dict[str, Any]],
) -> tuple[list[tuple[int, str, str]], list[tuple[int, str, str]], dict[str, str]]:
    """Trie les events en kills et deaths, avec mapping xuid→gamertag."""
    kills: list[tuple[int, str, str]] = []
    deaths: list[tuple[int, str, str]] = []
    gt_by_xuid: dict[str, str] = {}
    for e in events:
        if not isinstance(e, dict):
            continue
        et = _infer_event_type(e)
        t = _coerce_int(e.get("time_ms"))
        if t is None:
            continue
        xu = _coerce_str(e.get("xuid")) or ""
        gt = _coerce_str(e.get("gamertag")) or ""
        if xu and gt:
            gt_by_xuid[xu] = gt
        if et == "kill":
            kills.append((int(t), xu, gt))
        elif et == "death":
            deaths.append((int(t), xu, gt))
    kills.sort(key=lambda x: x[0])
    deaths.sort(key=lambda x: x[0])
    return kills, deaths, gt_by_xuid


def _build_rank_map(official_stats: list[Any] | None) -> dict[str, int]:
    """Construit le mapping xuid→rang depuis les stats officielles."""
    if not official_stats:
        return {}
    rank_by_xuid: dict[str, int] = {}
    for s in official_stats:
        xu = s.xuid if hasattr(s, "xuid") else s["xuid"]
        rk = s.rank if hasattr(s, "rank") else s.get("rank", 99)
        rank_by_xuid[xu] = rk
    return rank_by_xuid


_EMPTY_RESULT = AntagonistsResult(
    nemesis=None,
    bully=None,
    my_deaths_total=0,
    my_deaths_assigned_certain=0,
    my_deaths_assigned_total=0,
    my_kills_total=0,
    my_kills_assigned_certain=0,
    my_kills_assigned_total=0,
    is_validated=False,
    validation_notes="XUID manquant",
)


def compute_personal_antagonists(
    events: Iterable[dict[str, Any]],
    *,
    me_xuid: str,
    tolerance_ms: int = 5,
    official_stats: list[MatchPlayerStats] | None = None,
) -> AntagonistsResult:
    """Calcule Némésis et Souffre-douleur à partir des highlight events.

    Stratégie hybride (A+B) avec validation (Sprint 3.1):
    - Pass 1: on attribue uniquement les duels non ambigus (1 seul candidat).
    - Pass 2: on attribue les cas ambigus via une heuristique déterministe.
    """
    if tolerance_ms < 0:
        tolerance_ms = 0
    me = _coerce_str(me_xuid) or ""
    if not me:
        return _EMPTY_RESULT

    rank_by_xuid = _build_rank_map(official_stats)
    kills, deaths, gt_by_xuid = _parse_events(events)
    kill_times = [t for t, _, _ in kills]
    death_times = [t for t, _, _ in deaths]

    # Némésis: qui m'a le plus tué (killer → me)
    my_deaths = [(t, vx, vgt) for t, vx, vgt in deaths if str(vx) == str(me)]
    nem_certain, nem_est, _ = _assign_antagonist_counts(
        my_deaths,
        kills,
        kill_times,
        tolerance_ms,
        rank_by_xuid,
    )
    # Souffre-douleur: qui j'ai le plus tué (me → victim)
    my_kills = [(t, kx, kgt) for t, kx, kgt in kills if str(kx) == str(me)]
    bully_certain, bully_est, _ = _assign_antagonist_counts(
        my_kills,
        deaths,
        death_times,
        tolerance_ms,
        rank_by_xuid,
        exclude_self=me,
    )

    nem_xu = _select_top_antagonist(nem_certain, nem_est)
    bully_xu = _select_top_antagonist(bully_certain, bully_est)
    nemesis = _build_duel(nem_xu, gt_by_xuid, nem_certain, nem_est, bully_certain, bully_est)
    bully = _build_duel(bully_xu, gt_by_xuid, nem_certain, nem_est, bully_certain, bully_est)

    my_deaths_c = sum(nem_certain.values())
    my_deaths_t = my_deaths_c + sum(nem_est.values())
    my_kills_c = sum(bully_certain.values())
    my_kills_t = my_kills_c + sum(bully_est.values())
    is_validated, validation_notes = _validate_against_official(
        me,
        my_kills_t,
        my_deaths_t,
        official_stats,
    )

    return AntagonistsResult(
        nemesis=nemesis,
        bully=bully,
        my_deaths_total=len(my_deaths),
        my_deaths_assigned_certain=my_deaths_c,
        my_deaths_assigned_total=my_deaths_t,
        my_kills_total=len(my_kills),
        my_kills_assigned_certain=my_kills_c,
        my_kills_assigned_total=my_kills_t,
        is_validated=is_validated,
        validation_notes=validation_notes,
    )


# Réexport explicite pour compatibilité __all__
__all__ = [
    # Types
    "AntagonistsResult",
    "AntagonistsResultPolars",
    "EstimatedCount",
    "KVPair",
    "MatchPlayerStats",
    "OpponentDuel",
    "ValidationResult",
    # Core
    "compute_killer_victim_pairs",
    "compute_personal_antagonists",
    "get_player_rank",
    "validate_and_adjust_pairs",
    # Polars
    "compute_duel_history_polars",
    "compute_kd_timeseries_by_minute_polars",
    "compute_personal_antagonists_from_pairs_polars",
    "killer_victim_counts_long_polars",
    "killer_victim_matrix_polars",
]
