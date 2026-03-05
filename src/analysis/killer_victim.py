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


def compute_personal_antagonists(  # noqa: C901, PLR0912, PLR0915
    events: Iterable[dict[str, Any]],
    *,
    me_xuid: str,
    tolerance_ms: int = 5,
    official_stats: list[MatchPlayerStats] | None = None,
) -> AntagonistsResult:
    """Calcule Némésis et Souffre-douleur à partir des highlight events.

    Stratégie hybride (A+B) avec validation (Sprint 3.1):
    - Pass 1: on attribue uniquement les duels non ambigus (1 seul candidat).
    - Pass 2: on attribue les cas ambigus via une heuristique déterministe:
        1) privilégie l'adversaire déjà le plus fréquent en "certain" (Pass 1)
        2) NEW: tie-breaker par rang dans le match (meilleur classement = priorité)
        3) sinon, fallback stable: plus petit XUID (numérique si possible)

    Cette approche évite les résultats "aléatoires" tout en conservant
    une transparence: chaque compteur sépare certain vs estimé.

    Args:
        events: highlight events bruts (dicts) du match.
        me_xuid: XUID du joueur (digits recommandés, ou "xuid(...)" accepté).
        tolerance_ms: fenêtre de jointure en millisecondes.
        official_stats: Stats officielles des joueurs du match (pour validation et tie-breaker).

    Returns:
        AntagonistsResult avec flag de validation.
    """

    if tolerance_ms < 0:
        tolerance_ms = 0

    me = _coerce_str(me_xuid) or ""
    if not me:
        return AntagonistsResult(
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

    # Construire un mapping xuid → rang (pour tie-breaker)
    def get_xuid(s) -> str:
        return s.xuid if hasattr(s, "xuid") else s["xuid"]

    def get_rank(s) -> int:
        return s.rank if hasattr(s, "rank") else s.get("rank", 99)

    rank_by_xuid: dict[str, int] = {}
    if official_stats:
        for s in official_stats:
            rank_by_xuid[get_xuid(s)] = get_rank(s)

    kills: list[tuple[int, str, str]] = []
    deaths: list[tuple[int, str, str]] = []
    # Map xuid -> last known gamertag (from events)
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

    kill_times = [t for t, _xu, _gt in kills]
    death_times = [t for t, _xu, _gt in deaths]

    def _xuid_sort_key(xuid_value: str) -> tuple[int, str]:
        s = str(xuid_value or "").strip()
        try:
            return (0, f"{int(s):020d}")
        except Exception:
            return (1, s)

    def _choose_best(candidates: list[str], prefer: dict[str, int]) -> str:
        """Choisit le meilleur candidat parmi les ambigus.

        Heuristiques (dans l'ordre):
        1. Privilégie l'adversaire déjà le plus fréquent en "certain" (Pass 1)
        2. Tie-breaker par rang dans le match (meilleur classement = priorité)
        3. Fallback stable: plus petit XUID numérique

        Sprint 3.1: Ajout du tie-breaker par rang.
        """
        if not candidates:
            return ""

        # heuristique 1: privilégier ceux déjà fréquents en certain
        best_score = None
        best = None
        for c in candidates:
            score = int(prefer.get(c, 0))
            if best_score is None or score > best_score:
                best_score = score
                best = c

        if best is None:
            return ""

        # si plusieurs candidats ont le même score "certain"
        top_score = int(best_score or 0)
        tied = [c for c in candidates if int(prefer.get(c, 0)) == top_score]

        if len(tied) == 1:
            return tied[0]

        # Sprint 3.1: Tie-breaker par rang dans le match
        # Meilleur rang (plus petit numéro) = priorité
        if rank_by_xuid:
            tied.sort(key=lambda x: (rank_by_xuid.get(x, 999), _xuid_sort_key(x)))
        else:
            # Fallback: plus petit xuid (stable)
            tied.sort(key=_xuid_sort_key)

        return tied[0]

    # -----------------
    # Némésis: qui m'a le plus tué (killer -> me)
    # -----------------
    my_deaths = [(t, vx, vgt) for (t, vx, vgt) in deaths if str(vx) == str(me)]
    my_deaths_total = len(my_deaths)
    used_kill_idx: set[int] = set()

    nem_certain: dict[str, int] = {}
    nem_est: dict[str, int] = {}

    pending_deaths: list[tuple[int, str, str, list[int]]] = []

    for t_death, _vx, _vgt in my_deaths:
        lo = bisect_left(kill_times, t_death - tolerance_ms)
        hi = bisect_right(kill_times, t_death + tolerance_ms)
        cand_idx = [i for i in range(lo, hi) if i not in used_kill_idx]
        if len(cand_idx) == 1:
            i = cand_idx[0]
            used_kill_idx.add(i)
            kx = str(kills[i][1] or "").strip()
            if kx:
                nem_certain[kx] = int(nem_certain.get(kx, 0)) + 1
        elif len(cand_idx) > 1:
            pending_deaths.append((t_death, _vx, _vgt, cand_idx))

    # Pass 2: assignation estimée
    for _t_death, _vx, _vgt, cand_idx in pending_deaths:
        cand_idx2 = [i for i in cand_idx if i not in used_kill_idx]
        if not cand_idx2:
            continue
        candidates = [str(kills[i][1] or "").strip() for i in cand_idx2]
        candidates = [c for c in candidates if c]
        if not candidates:
            continue
        chosen = _choose_best(candidates, nem_certain)
        if not chosen:
            continue
        # on consomme un kill event correspondant au chosen (stable)
        chosen_idxs = [i for i in cand_idx2 if str(kills[i][1] or "").strip() == chosen]
        if not chosen_idxs:
            continue
        used_kill_idx.add(min(chosen_idxs))
        nem_est[chosen] = int(nem_est.get(chosen, 0)) + 1

    my_deaths_assigned_certain = sum(nem_certain.values())
    my_deaths_assigned_total = my_deaths_assigned_certain + sum(nem_est.values())

    # -----------------
    # Souffre-douleur: qui j'ai le plus tué (me -> victim)
    # -----------------
    my_kills = [(t, kx, kgt) for (t, kx, kgt) in kills if str(kx) == str(me)]
    my_kills_total = len(my_kills)
    used_death_idx: set[int] = set()

    bully_certain: dict[str, int] = {}
    bully_est: dict[str, int] = {}
    pending_kills: list[tuple[int, str, str, list[int]]] = []

    for t_kill, _kx, _kgt in my_kills:
        lo = bisect_left(death_times, t_kill - tolerance_ms)
        hi = bisect_right(death_times, t_kill + tolerance_ms)
        cand_idx = [i for i in range(lo, hi) if i not in used_death_idx]
        if len(cand_idx) == 1:
            i = cand_idx[0]
            used_death_idx.add(i)
            vx = str(deaths[i][1] or "").strip()
            if vx and vx != str(me):
                bully_certain[vx] = int(bully_certain.get(vx, 0)) + 1
        elif len(cand_idx) > 1:
            pending_kills.append((t_kill, _kx, _kgt, cand_idx))

    for _t_kill, _kx, _kgt, cand_idx in pending_kills:
        cand_idx2 = [i for i in cand_idx if i not in used_death_idx]
        if not cand_idx2:
            continue
        candidates = [str(deaths[i][1] or "").strip() for i in cand_idx2]
        candidates = [c for c in candidates if c and c != str(me)]
        if not candidates:
            continue
        chosen = _choose_best(candidates, bully_certain)
        if not chosen:
            continue
        chosen_idxs = [i for i in cand_idx2 if str(deaths[i][1] or "").strip() == chosen]
        if not chosen_idxs:
            continue
        used_death_idx.add(min(chosen_idxs))
        bully_est[chosen] = int(bully_est.get(chosen, 0)) + 1

    my_kills_assigned_certain = sum(bully_certain.values())
    my_kills_assigned_total = my_kills_assigned_certain + sum(bully_est.values())

    # -----------------
    # Sélection top nemesis / bully + construction du duel (inclut les 2 sens)
    # -----------------
    def _top_xuid(certain_map: dict[str, int], est_map: dict[str, int]) -> str | None:
        keys = set(certain_map.keys()) | set(est_map.keys())
        if not keys:
            return None
        # max sur total/certain; tie-break xuid asc
        best = None
        best_tuple = None
        for x in keys:
            t = (
                int(certain_map.get(x, 0)) + int(est_map.get(x, 0)),
                int(certain_map.get(x, 0)),
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

    nem_xu = _top_xuid(nem_certain, nem_est)
    bully_xu = _top_xuid(bully_certain, bully_est)

    # cross counts (me<->opponent) via the already computed per-direction counters
    def _ec(map_c: dict[str, int], map_e: dict[str, int], key: str | None) -> EstimatedCount:
        if not key:
            return EstimatedCount(0, 0)
        return EstimatedCount(int(map_c.get(key, 0)), int(map_e.get(key, 0)))

    # me killed opponent counters are bully_certain/bully_est (victims)
    def _build_duel(
        op_xu: str | None, *, killed_me_c: dict[str, int], killed_me_e: dict[str, int]
    ) -> OpponentDuel | None:
        if not op_xu:
            return None
        op_gt = gt_by_xuid.get(op_xu, "")
        return OpponentDuel(
            xuid=op_xu,
            gamertag=op_gt,
            opponent_killed_me=_ec(killed_me_c, killed_me_e, op_xu),
            me_killed_opponent=_ec(bully_certain, bully_est, op_xu),
        )

    nemesis = _build_duel(nem_xu, killed_me_c=nem_certain, killed_me_e=nem_est)
    bully = _build_duel(bully_xu, killed_me_c=nem_certain, killed_me_e=nem_est)

    # Sprint 3.1: Validation avec les stats officielles
    is_validated = False
    validation_notes = ""

    def get_xuid(s) -> str:
        return s.xuid if hasattr(s, "xuid") else s["xuid"]

    def get_stat(s, key: str) -> int:
        return getattr(s, key, 0) if hasattr(s, key) else s.get(key, 0)

    if official_stats:
        # Trouver mes stats officielles
        my_official = next((s for s in official_stats if get_xuid(s) == me), None)
        if my_official:
            # Comparer mes kills/deaths reconstitués vs officiels
            off_kills = get_stat(my_official, "kills")
            off_deaths = get_stat(my_official, "deaths")
            kills_diff = my_kills_assigned_total - off_kills
            deaths_diff = my_deaths_assigned_total - off_deaths

            if kills_diff == 0 and deaths_diff == 0:
                is_validated = True
                validation_notes = "Cohérent avec stats officielles"
            else:
                notes = []
                if kills_diff != 0:
                    notes.append(
                        f"kills: {my_kills_assigned_total} vs {off_kills} ({kills_diff:+d})"
                    )
                if deaths_diff != 0:
                    notes.append(
                        f"deaths: {my_deaths_assigned_total} vs {off_deaths} ({deaths_diff:+d})"
                    )
                validation_notes = "Écarts: " + ", ".join(notes)
        else:
            validation_notes = "Stats officielles du joueur non trouvées"
    else:
        validation_notes = "Pas de stats officielles pour validation"

    return AntagonistsResult(
        nemesis=nemesis,
        bully=bully,
        my_deaths_total=int(my_deaths_total),
        my_deaths_assigned_certain=int(my_deaths_assigned_certain),
        my_deaths_assigned_total=int(my_deaths_assigned_total),
        my_kills_total=int(my_kills_total),
        my_kills_assigned_certain=int(my_kills_assigned_certain),
        my_kills_assigned_total=int(my_kills_assigned_total),
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
