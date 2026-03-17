"""Domaine pur — parsing des armes depuis les films Halo Infinite.

Parser v2 : claim-and-remove pour tous les joueurs du lobby.
Aucune dépendance IO (pas de DB, pas d'API, pas de réseau).
"""

from __future__ import annotations

import logging
from collections import Counter
from typing import TYPE_CHECKING

from src.analysis._kill_attribution import KillAttribution
from src.analysis._weapon_data import (
    GRENADE_MEDALS,  # noqa: F401 (re-export)
    GRENADE_WEAPON_ID,
    MELEE_MEDALS,  # noqa: F401 (re-export)
    MELEE_WEAPON_ID,
    WEAPON_BYTES_TO_INT,
    WEAPON_ID_MAP,  # noqa: F401 (re-export)
    WEAPON_IDS_INT,  # noqa: F401 (re-export)
    WEAPON_INT_TO_NAME,  # noqa: F401 (re-export)
    WEAPON_TIMING_BY_ID,  # noqa: F401 (re-export)
)
from src.analysis._weapon_scanners import (
    COMMON_WEAPON_SUFFIX,  # noqa: F401 (re-export)
    FORMULA_A_PATTERN,  # noqa: F401 (re-export)
    FRAME_MARKER,  # noqa: F401 (re-export)
    estimate_ts_frames,
    find_frame_positions,  # noqa: F401 (re-export)
    scan_fire_events_bitstring,
    scan_formula_a,  # noqa: F401 (re-export)
    scan_formula_a_ns,  # noqa: F401 (re-export)
)

if TYPE_CHECKING:
    from src.analysis._parser_logging import MatchLogCollector

logger = logging.getLogger(__name__)

# --- Constantes ---

KILL_WINDOW_MS = 5000

POV_PLAYER_INDEX = 1
MELEE_FILM_ID = MELEE_WEAPON_ID
GRENADE_FILM_ID = GRENADE_WEAPON_ID
VEHICLE_FILM_ID = 2
MELEE_API_ID = MELEE_FILM_ID
GRENADE_API_ID = GRENADE_FILM_ID
VEHICLE_API_ID = VEHICLE_FILM_ID

_DEFAULT_TIMING = (650, 2000)

# Aliases internes pour compat
_estimate_ts_frames = estimate_ts_frames
_scan_fire_events_bitstring = scan_fire_events_bitstring
build_frame_estimator = estimate_ts_frames


# ── Fonctions de scan haut-niveau ──


def scan_fire_events(
    chunk_data: bytes,
    player_index: int,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    *,
    packets: list | None = None,
) -> list[dict]:
    """Scanne les fire events d'un chunk REPLICATION_DATA pour un player_index."""
    if packets is not None:
        from src.analysis.packet_index import build_packet_estimator

        ts_fn = build_packet_estimator(packets, chunk_start_ms)
    else:
        ts_fn = estimate_ts_frames(chunk_data, chunk_start_ms, chunk_duration_ms)
    return scan_fire_events_bitstring(chunk_data, player_index, ts_fn)


def scan_fire_events_all(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    *,
    packets: list | None = None,
    n_players: int = 8,
) -> list[dict]:
    """Scanne les fire events pour TOUS les player_index d'un chunk.

    Chaque event est enrichi avec le champ ``player_index``.
    """
    if packets is not None:
        from src.analysis.packet_index import build_packet_estimator

        ts_fn = build_packet_estimator(packets, chunk_start_ms)
    else:
        ts_fn = estimate_ts_frames(chunk_data, chunk_start_ms, chunk_duration_ms)

    all_events: list[dict] = []
    for pi in range(n_players):
        events = scan_fire_events_bitstring(chunk_data, pi, ts_fn)
        for ev in events:
            ev["player_index"] = pi
        all_events.extend(events)
    return sorted(all_events, key=lambda x: x["timestamp_ms"])


def scan_all_players(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    n_players: int = 8,
) -> dict[int, list[dict]]:
    """Scanne les fire events pour tous les player_index (diagnostic)."""
    ts_fn = estimate_ts_frames(chunk_data, chunk_start_ms, chunk_duration_ms)
    return {idx: scan_fire_events_bitstring(chunk_data, idx, ts_fn) for idx in range(n_players)}


# ── Timeline et chunk lookup ──


def build_weapon_timeline(
    chunks: dict[int, tuple[bytes, int, int]],
) -> tuple[dict[int, dict[int, bytes]], dict[int, set[int]], list[tuple[int, int]]]:
    """Construit la timeline arme par chunk (Formula A, Section 1)."""
    timeline: dict[int, dict[int, bytes]] = {}
    swap_pis: dict[int, set[int]] = {}
    timing: list[tuple[int, int]] = []
    for idx in sorted(chunks):
        data, start_ms, dur_ms = chunks[idx]
        events = scan_formula_a(data)
        chunk_state: dict[int, bytes] = {}
        pi_seen_wids: dict[int, set[bytes]] = {}
        for _, pi, wid in events:
            chunk_state[pi] = wid
            pi_seen_wids.setdefault(pi, set()).add(wid)
        timeline[idx] = chunk_state
        swap_pis[idx] = {pi for pi, wids in pi_seen_wids.items() if len(wids) > 1}
        timing.append((start_ms, start_ms + dur_ms))
    return timeline, swap_pis, timing


def build_weapon_timeline_ns(
    chunks: dict[int, tuple[bytes, int, int]],
) -> dict[int, dict[int, bytes]]:
    """Construit la timeline arme (NS layer, TYPE IDs) par chunk."""
    timeline_ns: dict[int, dict[int, bytes]] = {}
    for idx in sorted(chunks):
        data, _, _ = chunks[idx]
        events = scan_formula_a_ns(data)
        chunk_state: dict[int, bytes] = {}
        for _, pi, wid in events:
            chunk_state[pi] = wid
        timeline_ns[idx] = chunk_state
    return timeline_ns


def build_weapon_timelines(
    chunks: dict[int, tuple[bytes, int, int]],
) -> tuple[
    dict[int, dict[int, bytes]],
    dict[int, dict[int, bytes]],
    dict[int, set[int]],
    list[tuple[int, int]],
]:
    """Construit timeline raw + NS en une seule passe sur les chunks.

    Équivalent de build_weapon_timeline() + build_weapon_timeline_ns()
    mais avec un seul tri et une seule itération.

    Returns:
        (timeline, timeline_ns, swap_pis, timing)
    """
    timeline: dict[int, dict[int, bytes]] = {}
    timeline_ns: dict[int, dict[int, bytes]] = {}
    swap_pis: dict[int, set[int]] = {}
    timing: list[tuple[int, int]] = []

    for idx in sorted(chunks):
        data, start_ms, dur_ms = chunks[idx]

        # Raw timeline (Formula A, Section 1) + swap detection
        raw_events = scan_formula_a(data)
        chunk_raw: dict[int, bytes] = {}
        pi_seen_wids: dict[int, set[bytes]] = {}
        for _, pi, wid in raw_events:
            chunk_raw[pi] = wid
            pi_seen_wids.setdefault(pi, set()).add(wid)
        timeline[idx] = chunk_raw
        swap_pis[idx] = {pi for pi, wids in pi_seen_wids.items() if len(wids) > 1}
        timing.append((start_ms, start_ms + dur_ms))

        # NS timeline (TYPE IDs) — même chunk, pas de second sort
        ns_events = scan_formula_a_ns(data)
        chunk_ns: dict[int, bytes] = {}
        for _, pi, wid in ns_events:
            chunk_ns[pi] = wid
        timeline_ns[idx] = chunk_ns

    return timeline, timeline_ns, swap_pis, timing


def find_chunk_at_time(
    chunks_sorted: list[int],
    timing: list[tuple[int, int]],
    t_ms: int,
) -> int:
    """Retourne l'index de chunk couvrant t_ms."""
    for chunk_idx, (start, end) in zip(chunks_sorted, timing, strict=False):
        if start <= t_ms < end:
            return chunk_idx
    return chunks_sorted[-1] if chunks_sorted else 0


# ── Attribution b2_stream → player_index ──


def map_b2_to_player(
    all_fire_events: list[dict],
    timeline: dict[int, dict[int, bytes]],
    timing: list[tuple[int, int]],
    chunks_sorted: list[int],
) -> dict[int, int]:
    """Mappe b2_stream → player_index par cross-référence avec Formula A."""
    b2_votes: dict[int, Counter] = {}
    for ev in all_fire_events:
        b2 = ev["fire_seq"]
        weapon_bytes = ev["weapon_bytes"]
        t_ms = int(ev["timestamp_ms"])
        chunk_idx = find_chunk_at_time(chunks_sorted, timing, t_ms)
        chunk_state = timeline.get(chunk_idx, {})
        for pi, pi_wid in chunk_state.items():
            if pi_wid == weapon_bytes:
                b2_votes.setdefault(b2, Counter())[pi] += 1
                break
    return {b2: ctr.most_common(1)[0][0] for b2, ctr in b2_votes.items() if ctr}


def group_events_by_pi(
    all_fire_events: list[dict],
    b2_to_pi: dict[int, int],
) -> dict[int, list[dict]]:
    """Dispatche les fire events par player_index résolu via b2_to_pi.

    Events dont le b2 n'est pas résolu sont ignorés.
    """
    by_pi: dict[int, list[dict]] = {}
    for ev in all_fire_events:
        pi = b2_to_pi.get(ev["fire_seq"])
        if pi is not None:
            by_pi.setdefault(pi, []).append(ev)
    return by_pi


# ── Confidence ──


def compute_confidence(weapon_id: int | None, delta_ms: int | None) -> str:
    """Calcule la confidence d'une attribution selon les zones FINDINGS §6a."""
    if weapon_id is None or delta_ms is None:
        return "none"
    swap_ms, travel_max = WEAPON_TIMING_BY_ID.get(weapon_id, _DEFAULT_TIMING)
    if delta_ms < swap_ms:
        return "high"
    if delta_ms <= travel_max:
        return "medium"
    return "low"


# Alias legacy
_get_confidence = compute_confidence


# ── Corrélation v2 — claim-and-remove ──


def correlate_kills(  # noqa: PLR0913
    kills: list[dict],
    fire_events_by_pi: dict[int, list[dict]],
    pi_mapping: dict[str, int],
    timeline: dict[int, dict[int, bytes]],
    swap_pis: dict[int, set[int]],
    timing: list[tuple[int, int]],
    chunks_sorted: list[int],
    match_id: str = "",
    *,
    log_collector: MatchLogCollector | None = None,
) -> list[KillAttribution]:
    """Corrélation claim-and-remove : chaque fire event ne sert qu'une fois.

    Pour chaque kill (trié par temps) :
    1. Si melee/grenade → sentinel, path="none"
    2. Sinon → chercher le fire event non-claimé le plus proche dans [t-5s, t]
       pour le player_index du tueur
    3. Si trouvé → claim, path="fire_event"
    4. Sinon → fallback Formula A NS, path="formula_a"
    5. compute_confidence()

    Args:
        kills: Liste de dicts {match_id, xuid, time_ms, is_melee, is_grenade}.
        fire_events_by_pi: {player_index: [events]} triés par timestamp.
        pi_mapping: {xuid: player_index}.
        timeline/swap_pis/timing/chunks_sorted: état Section 1.
        match_id: Pour le logging.
        log_collector: Collecteur de logs structurés (optionnel).
    """
    # deep-copy des listes pour claim-and-remove (on retire les events claimés)
    available: dict[int, list[dict]] = {
        pi: list(events) for pi, events in fire_events_by_pi.items()
    }
    kills_sorted = sorted(kills, key=lambda k: k["time_ms"])
    results: list[KillAttribution] = []

    for kill in kills_sorted:
        attr = _attribute_single_kill(
            kill,
            available,
            pi_mapping,
            timeline,
            swap_pis,
            timing,
            chunks_sorted,
            match_id,
        )
        if log_collector:
            log_collector.kill_decision(
                kill,
                attr,
                {"candidates_count": 0, "fallback_used": attr.attribution_path == "formula_a"},
            )
        results.append(attr)
    return results


def _attribute_single_kill(  # noqa: PLR0913
    kill: dict,
    available: dict[int, list[dict]],
    pi_mapping: dict[str, int],
    timeline: dict[int, dict[int, bytes]],
    swap_pis: dict[int, set[int]],
    timing: list[tuple[int, int]],
    chunks_sorted: list[int],
    match_id: str,
) -> KillAttribution:
    """Attribue un kill unique à une arme (claim-and-remove)."""
    xuid = kill["xuid"]

    # Melee / grenade → sentinel
    if kill.get("is_melee"):
        return _make_sentinel(kill, match_id, MELEE_WEAPON_ID)
    if kill.get("is_grenade"):
        return _make_sentinel(kill, match_id, GRENADE_WEAPON_ID)

    pi = pi_mapping.get(xuid)

    # Essayer fire event (claim-and-remove)
    if pi is not None:
        result = _try_fire_event_claim(kill, pi, available, match_id)
        if result is not None:
            return result

    # Fallback : Formula A (NS timeline)
    return _fallback_formula_a(
        kill,
        pi,
        timeline,
        swap_pis,
        timing,
        chunks_sorted,
        match_id,
    )


def _make_sentinel(kill: dict, match_id: str, weapon_id: int) -> KillAttribution:
    """Crée un KillAttribution pour un kill melee/grenade."""
    return KillAttribution(
        match_id=match_id,
        xuid=kill["xuid"],
        time_ms=kill["time_ms"],
        weapon_id=weapon_id,
        reconciled_as=None,
        delta_ms=None,
        confidence="none",
        attribution_path="none",
        swap_detected=False,
        delayed_damage=False,
        player_index=None,
        source_chunk_idx=None,
    )


def _try_fire_event_claim(
    kill: dict,
    pi: int,
    available: dict[int, list[dict]],
    match_id: str,
) -> KillAttribution | None:
    """Cherche le fire event non-claimé le plus proche, le claim si trouvé."""
    t_ms = kill["time_ms"]
    pi_events = available.get(pi, [])

    # Fenêtre [t - KILL_WINDOW_MS, t]
    candidates = [
        (i, ev)
        for i, ev in enumerate(pi_events)
        if (t_ms - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= t_ms
    ]
    if not candidates:
        return None

    # Prendre le plus proche dans le temps (le dernier avant le kill)
    best_idx, best_ev = max(candidates, key=lambda x: x[1]["timestamp_ms"])

    # Claim : retirer de available
    pi_events.pop(best_idx)

    wid_int = WEAPON_BYTES_TO_INT.get(best_ev["weapon_bytes"])
    if wid_int is None:
        wid_int = int.from_bytes(best_ev["weapon_bytes"], byteorder="big")

    delta = int(t_ms - best_ev["timestamp_ms"])
    conf = compute_confidence(wid_int, delta)
    swap_ms, travel_max = WEAPON_TIMING_BY_ID.get(wid_int, _DEFAULT_TIMING)

    return KillAttribution(
        match_id=match_id,
        xuid=kill["xuid"],
        time_ms=t_ms,
        weapon_id=wid_int,
        reconciled_as=None,
        delta_ms=delta,
        confidence=conf,
        attribution_path="fire_event",
        swap_detected=delta >= swap_ms,
        delayed_damage=delta > travel_max,
        player_index=pi,
        source_chunk_idx=best_ev.get("chunk_idx"),
    )


def _fallback_formula_a(  # noqa: PLR0913
    kill: dict,
    pi: int | None,
    timeline: dict[int, dict[int, bytes]],
    swap_pis: dict[int, set[int]],
    timing: list[tuple[int, int]],
    chunks_sorted: list[int],
    match_id: str,
) -> KillAttribution:
    """Fallback Formula A NS : arme tenue à t_ms selon timeline Section 1."""
    t_ms = kill["time_ms"]
    wid_int = None
    swap = False

    if pi is not None and chunks_sorted:
        chunk_idx = find_chunk_at_time(chunks_sorted, timing, t_ms)
        chunk_state = timeline.get(chunk_idx, {})
        wid_bytes = chunk_state.get(pi)
        if wid_bytes is not None:
            wid_int = WEAPON_BYTES_TO_INT.get(wid_bytes)
            if wid_int is None:
                wid_int = int.from_bytes(wid_bytes, byteorder="big")
        swap = pi in swap_pis.get(chunk_idx, set())

    conf = (
        "medium" if wid_int is not None and not swap else "low" if wid_int is not None else "none"
    )

    return KillAttribution(
        match_id=match_id,
        xuid=kill["xuid"],
        time_ms=t_ms,
        weapon_id=wid_int,
        reconciled_as=None,
        delta_ms=None,
        confidence=conf,
        attribution_path="formula_a",
        swap_detected=swap,
        delayed_damage=False,
        player_index=pi,
        source_chunk_idx=None,
    )


# ── Compat v1 : correlate_kills_to_weapons (POV simple) ──


def correlate_kills_to_weapons(
    kills: list[dict],
    fire_events_all: list[dict],
) -> list[dict]:
    """Compat v1 : corrélation POV simple sans claim-and-remove.

    Conservé pour les tests existants et le service v1.
    Délègue à _weapon_parser_compat.correlate_kills_to_weapons.
    """
    from src.analysis._weapon_parser_compat import (
        correlate_kills_to_weapons as _compat_correlate,
    )

    return _compat_correlate(kills, fire_events_all)


def count_kills_by_film_weapon(correlated: list[dict]) -> Counter:
    """Agrège les kills corrélés en Counter {weapon_id: n_kills}."""
    counts: Counter = Counter()
    for r in correlated:
        wid = r.get("weapon_id")
        if wid is not None:
            counts[wid] += 1
    return counts


count_kills_by_api_weapon = count_kills_by_film_weapon


# ── Re-exports player_index (compat) ──

from src.analysis.player_index import (  # noqa: F401, E402
    detect_player_indices,
    get_player_index_acurtis,
)
