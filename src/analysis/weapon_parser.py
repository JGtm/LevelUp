"""Domaine pur — parsing des armes depuis les films Halo Infinite.

Aucune dépendance IO (pas de DB, pas d'API, pas de réseau).
Seule dépendance externe : ``bitstring``.

Ce module extrait les fonctions de domaine pur depuis
``scripts/experimental/weapon_extraction.py`` pour réutilisation
dans le service d'orchestration et le backfill.
"""

from __future__ import annotations

from collections import Counter

from bitstring import Bits

from src.analysis._weapon_data import (
    GRENADE_MEDALS,  # noqa: F401  (re-export public API)
    MELEE_MEDALS,  # noqa: F401  (re-export public API)
    WEAPON_ID_MAP,
    WEAPON_TIMING,
)

# ══════════════════════════════════════════════════════════════════════════════
# Constantes de parsing
# ══════════════════════════════════════════════════════════════════════════════

FRAME_MARKER = bytes([0xA0, 0x7B, 0x42])
KILL_WINDOW_MS = 5000  # §6a Step 1 — Cindershot/Ravager/Disruptor ont travel_max=5000ms

COMMON_WEAPON_SUFFIX = bytes.fromhex("42c9679f")

# Pattern Section 1 (état snapshot) — Formula A : [20 00 02 pb ... wid:8B]
FORMULA_A_PATTERN = bytes.fromhex("200002")

_WEAPON_BIT_OFFSET = 40  # bits après event_start → weapon_id (64 bits)

# Sets pré-calculés pour validation rapide
WEAPON_IDS_INT: set[int] = set()
WEAPON_INT_TO_NAME: dict[int, str] = {}
for _wid_bytes, _wname in WEAPON_ID_MAP.items():
    _val = int.from_bytes(_wid_bytes, byteorder="big")
    WEAPON_IDS_INT.add(_val)
    WEAPON_INT_TO_NAME[_val] = _wname

# Suffixes 4B distincts (Formula A multi-suffixe — Energy Sword variants, Hammer variants…)
_ALL_FORMULA_A_SUFFIXES: frozenset[bytes] = frozenset(k[4:] for k in WEAPON_ID_MAP if len(k) == 8)

# ══════════════════════════════════════════════════════════════════════════════
# Mapping film_bytes → API weapon_id (entier)
# ══════════════════════════════════════════════════════════════════════════════

# POV player_index (invariant universel — inv #6, #23, #27, #41)
POV_PLAYER_INDEX = 1

# IDs synthétiques (réservés, ne collident pas avec les uint64 film)
MELEE_FILM_ID = 1
GRENADE_FILM_ID = 0
VEHICLE_FILM_ID = 2

# Aliases pour compatibilité (à supprimer après migration complète)
MELEE_API_ID = MELEE_FILM_ID
GRENADE_API_ID = GRENADE_FILM_ID
VEHICLE_API_ID = VEHICLE_FILM_ID


# ══════════════════════════════════════════════════════════════════════════════
# Fonctions de parsing pur
# ══════════════════════════════════════════════════════════════════════════════


def scan_formula_a(data: bytes) -> list[tuple[int, int, bytes]]:
    """Scanne les mises à jour d'état arme Section 1 (Formula A).

    Pattern : ``[20 00 02 pb ... wid:8B]`` où ``pi = pb >> 5``.
    Supporte tous les suffixes 4B de WEAPON_ID_MAP (Energy Sword variants,
    Hammer variants, etc.).
    Utilisé pour l'attribution T1 (coéquipiers non-POV).

    Returns:
        Liste de ``(offset, player_index, weapon_id_bytes)``.
    """
    results: list[tuple[int, int, bytes]] = []
    pos = 0
    while True:
        pos = data.find(FORMULA_A_PATTERN, pos)
        if pos == -1 or pos + 4 > len(data):
            break
        pb = data[pos + 3]
        pi = pb >> 5
        end = min(pos + 68, len(data))
        # Chercher la première occurrence valide de n'importe quel suffixe connu
        best_sx = -1
        for suffix in _ALL_FORMULA_A_SUFFIXES:
            sx_c = data.find(suffix, pos + 4, end)
            if sx_c < 4:
                continue
            ws_c = sx_c - 4
            if ws_c <= pos + 3:
                continue
            # Suffixe non-standard : valider contre WEAPON_ID_MAP (éviter faux positifs)
            if suffix != COMMON_WEAPON_SUFFIX and data[ws_c : ws_c + 8] not in WEAPON_ID_MAP:
                continue
            if best_sx == -1 or sx_c < best_sx:
                best_sx = sx_c
        if best_sx >= 4:
            ws = best_sx - 4
            if ws > pos + 3:
                results.append((pos, pi, data[ws : ws + 8]))
        pos += 4
    return results


def build_weapon_timeline(
    chunks: dict[int, tuple[bytes, int, int]],
) -> tuple[dict[int, dict[int, bytes]], dict[int, set[int]], list[tuple[int, int]]]:
    """Construit la timeline arme par chunk (Formula A, Section 1).

    Args:
        chunks: ``{chunk_idx: (data, start_ms, dur_ms)}``.

    Returns:
        ``(timeline, swap_pis, timing)`` où :
        - ``timeline[chunk_idx][pi]`` = dernière arme vue pour pi dans ce chunk
        - ``swap_pis[chunk_idx]`` = ensemble des pi ayant eu > 1 arme distincte
          dans le chunk (swap intra-chunk → confidence MEDIUM per FINDINGS §6b Step 3)
        - ``timing`` = liste de ``(start_ms, end_ms)`` par chunk_idx ordonnée
    """
    timeline: dict[int, dict[int, bytes]] = {}
    swap_pis: dict[int, set[int]] = {}
    timing: list[tuple[int, int]] = []
    for idx in sorted(chunks):
        data, start_ms, dur_ms = chunks[idx]
        events = scan_formula_a(data)
        chunk_state: dict[int, bytes] = {}
        pi_seen_wids: dict[int, set[bytes]] = {}
        for _, pi, wid in events:
            chunk_state[pi] = wid  # dernière mise à jour = état à la fin du chunk
            pi_seen_wids.setdefault(pi, set()).add(wid)
        timeline[idx] = chunk_state
        swap_pis[idx] = {pi for pi, wids in pi_seen_wids.items() if len(wids) > 1}
        timing.append((start_ms, start_ms + dur_ms))
    return timeline, swap_pis, timing


def find_chunk_at_time(
    chunks_sorted: list[int],
    timing: list[tuple[int, int]],
    t_ms: int,
) -> int:
    """Retourne l'index de chunk couvrant ``t_ms``.

    Fallback : dernier chunk si ``t_ms`` est après la fin.
    """
    for chunk_idx, (start, end) in zip(chunks_sorted, timing, strict=False):
        if start <= t_ms < end:
            return chunk_idx
    return chunks_sorted[-1] if chunks_sorted else 0


def find_frame_positions(data: bytes) -> list[int]:
    """Trouve toutes les positions des frame markers A0 7B 42."""
    positions: list[int] = []
    pos = 0
    while True:
        idx = data.find(FRAME_MARKER, pos)
        if idx == -1:
            break
        positions.append(idx)
        pos = idx + 1
    return positions


def build_frame_estimator(chunk_data: bytes, chunk_start_ms: int, chunk_duration_ms: int):
    """Retourne une fonction estimate_ts(byte_pos) → float."""
    frame_positions = find_frame_positions(chunk_data)
    n_frames = len(frame_positions)
    frame_duration_ms = chunk_duration_ms / n_frames if n_frames > 0 else 16.67

    def estimate_ts(byte_pos: int) -> float:
        lo, hi = 0, n_frames - 1
        frame_idx = 0
        while lo <= hi:
            mid = (lo + hi) // 2
            if frame_positions[mid] <= byte_pos:
                frame_idx = mid
                lo = mid + 1
            else:
                hi = mid - 1
        return chunk_start_ms + frame_idx * frame_duration_ms

    return estimate_ts


def _build_marker(player_index: int) -> Bits:
    """Construit le marqueur 11 bits pour un player_index donné."""
    byte1 = (player_index << 5) | 0x06
    return Bits(bin=f"0b101{byte1:08b}")


def _scan_fire_events_bitstring(
    chunk_data: bytes,
    player_index: int,
    estimate_ts,
) -> list[dict]:
    """Scanne les fire events via recherche exhaustive bitstring.

    Validation par weapon_id ∈ WEAPON_IDS_INT (enum) + suffixe commun.
    """
    bits = Bits(bytes=chunk_data)
    marker = _build_marker(player_index)
    total_bits = len(bits)
    events: list[dict] = []

    for position in bits.findall(marker, bytealigned=False):
        event_start = position + 3
        weapon_start = event_start + _WEAPON_BIT_OFFSET

        if weapon_start + 64 > total_bits:
            continue

        weapon_int = bits[weapon_start : weapon_start + 64].uint

        weapon_bytes = weapon_int.to_bytes(8, byteorder="big")
        if weapon_int not in WEAPON_IDS_INT and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
            continue

        byte_pos = position // 8
        fire_seq = (
            bits[event_start + 8 : event_start + 16].uint if event_start + 16 <= total_bits else 0
        )
        fire_counter = (
            bits[event_start + 24 : event_start + 32].uint if event_start + 32 <= total_bits else 0
        )

        weapon_name = WEAPON_INT_TO_NAME.get(
            weapon_int,
            WEAPON_ID_MAP.get(weapon_bytes, f"INCONNU ({weapon_bytes.hex()})"),
        )

        post_start = weapon_start + 64
        if post_start + 32 <= total_bits:
            post_bytes = bits[post_start : post_start + 32].bytes
        else:
            post_bytes = b"\x00" * 4
        burst_end = bool(post_bytes[1] & 0x01) if len(post_bytes) > 1 else False
        hit_likely = bool((post_bytes[2] & 0x01) == 0) if len(post_bytes) > 2 else None

        events.append(
            {
                "timestamp_ms": estimate_ts(byte_pos),
                "weapon_name": weapon_name,
                "weapon_bytes": weapon_bytes,
                "fire_seq": fire_seq,
                "fire_counter": fire_counter,
                "bit_offset": position % 8,
                "byte_pos": byte_pos,
                "post_bytes": post_bytes,
                "burst_end": burst_end,
                "hit_likely": hit_likely,
            }
        )

    # Dédupliquer par (fire_counter, weapon_bytes)
    deduped: dict[tuple[int, bytes], dict] = {}
    for ev in sorted(events, key=lambda x: x["timestamp_ms"]):
        key = (ev["fire_counter"], ev["weapon_bytes"])
        if key not in deduped:
            deduped[key] = ev
    return sorted(deduped.values(), key=lambda x: x["timestamp_ms"])


def scan_fire_events(
    chunk_data: bytes,
    player_index: int,
    chunk_start_ms: int,
    chunk_duration_ms: int,
) -> list[dict]:
    """Scanne les fire events d'un chunk REPLICATION_DATA pour un player_index."""
    estimate_ts = build_frame_estimator(chunk_data, chunk_start_ms, chunk_duration_ms)
    return _scan_fire_events_bitstring(chunk_data, player_index, estimate_ts)


def scan_all_players(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    n_players: int = 8,
) -> dict[int, list[dict]]:
    """Scanne les fire events pour tous les player_index (diagnostic)."""
    estimate_ts = build_frame_estimator(chunk_data, chunk_start_ms, chunk_duration_ms)
    return {
        idx: _scan_fire_events_bitstring(chunk_data, idx, estimate_ts) for idx in range(n_players)
    }


# ══════════════════════════════════════════════════════════════════════════════
# Corrélation kills ↔ fire events
# ══════════════════════════════════════════════════════════════════════════════


def _get_confidence(weapon_name: str, delta_ms: int | None) -> str:
    """Calcule la confidence d'une attribution POV selon les zones FINDINGS §6a."""
    if delta_ms is None:
        return "none"
    swap_ms, travel_max = WEAPON_TIMING.get(weapon_name, (650, 2000))
    if delta_ms < swap_ms:
        return "high"  # Zone A — swap physiquement impossible
    if delta_ms <= travel_max:
        return "medium"  # Zone B — fenêtre ambiguë
    return "low"  # Zone C — delayed damage


def correlate_kills_to_weapons(
    kills: list[dict],
    fire_events_all: list[dict],
) -> list[dict]:
    """Pour chaque kill à T, cherche le dernier fire event dans [T-KILL_WINDOW, T].

    Calcule delta_ms et confidence selon les zones FINDINGS §6a.
    Exclut les kills melee/grenade de la recherche fire event.
    """
    results = []
    for kill in kills:
        kill_t = kill["time_ms"]
        if kill["is_melee"]:
            results.append(
                {
                    **kill,
                    "weapon_name": "MELEE",
                    "matched_fire_event": None,
                    "delta_ms": None,
                    "confidence": "none",
                    "swap_detected": False,
                    "delayed_damage": False,
                }
            )
            continue
        if kill["is_grenade"]:
            results.append(
                {
                    **kill,
                    "weapon_name": "GRENADE",
                    "matched_fire_event": None,
                    "delta_ms": None,
                    "confidence": "none",
                    "swap_detected": False,
                    "delayed_damage": False,
                }
            )
            continue

        candidates = [
            ev
            for ev in fire_events_all
            if (kill_t - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= kill_t
        ]
        best: dict | None = None
        if candidates:
            best = max(candidates, key=lambda e: e["timestamp_ms"])

        wname = best["weapon_name"] if best else "NON TROUVE"
        delta = int(kill_t - best["timestamp_ms"]) if best else None
        swap_ms, travel_max = WEAPON_TIMING.get(wname, (650, 2000))
        conf = _get_confidence(wname, delta)
        # Zone B W2 check (FINDINGS §6a Step 2) : si MEDIUM, chercher un fire W2
        # dans [kill_t, kill_t + swap_ms] — preuve que W2 était équipé au moment du kill
        if conf == "medium" and best is not None:
            post_swap = [
                ev
                for ev in fire_events_all
                if kill_t < ev["timestamp_ms"] <= kill_t + swap_ms and ev["weapon_name"] != wname
            ]
            if post_swap:
                w2_ev = min(post_swap, key=lambda e: e["timestamp_ms"])
                wname = w2_ev["weapon_name"]
                swap_ms, travel_max = WEAPON_TIMING.get(wname, (650, 2000))
                conf = "high"
        swap_detected = best is not None and delta is not None and delta >= swap_ms
        delayed = best is not None and delta is not None and delta > travel_max
        results.append(
            {
                **kill,
                "weapon_name": wname,
                "fire_seq": best["fire_seq"] if best else None,
                "matched_fire_event": best,
                "delta_ms": delta,
                "confidence": conf,
                "swap_detected": swap_detected,
                "delayed_damage": delayed,
            }
        )
    return results


# ══════════════════════════════════════════════════════════════════════════════
# Agrégation kills → Counter par API weapon_id
# ══════════════════════════════════════════════════════════════════════════════


def count_kills_by_film_weapon(correlated: list[dict]) -> Counter:
    """Agrège les kills corrélés en Counter {film_weapon_uint64: n_kills}.

    Kills melee → MELEE_FILM_ID=1, grenade → GRENADE_FILM_ID=0.
    L'ID stocké est le uint64 big-endian des 8 bytes film (WEAPON_ID_MAP key).
    Kills sans match → ignorés ; armes inconnues → incluses si le uint64 est
    dans WEAPON_IDS_INT (enum validé).
    """
    counts: Counter = Counter()
    for r in correlated:
        if r.get("is_melee"):
            counts[MELEE_FILM_ID] += 1
            continue
        if r.get("is_grenade"):
            counts[GRENADE_FILM_ID] += 1
            continue
        ev = r.get("matched_fire_event")
        if not ev:
            continue
        wbytes = ev.get("weapon_bytes")
        if not wbytes:
            continue
        weapon_uint64 = int.from_bytes(wbytes, byteorder="big")
        if weapon_uint64 in WEAPON_IDS_INT:
            counts[weapon_uint64] += 1
    return counts


# Alias de compatibilité — préférer count_kills_by_film_weapon
count_kills_by_api_weapon = count_kills_by_film_weapon


# ══════════════════════════════════════════════════════════════════════════════
# Détection player_index via méthode acurtis (inv #26)
# ══════════════════════════════════════════════════════════════════════════════


def get_player_index_acurtis(bits: Bits, xuid_int: int) -> int | None:
    """Retourne le player_index d'un joueur depuis son XUID (méthode acurtis).

    Cherche le XUID (little-endian uint64, non-byte-aligned) dans le bitstream
    et lit les 5 bits qui le précèdent.

    Source : acurtis 2026-03-03 (inv #26), confirmé sur 3 matchs JGtm.

    Args:
        bits: Bitstring du chunk REPLICATION_DATA.
        xuid_int: XUID du joueur sous forme d'entier.

    Returns:
        player_index (0-7) ou None si XUID absent du chunk.
    """
    from bitstring import Bits as _Bits  # import local pour éviter dep obligatoire

    term = _Bits(uintle=xuid_int, length=64)
    position = bits.find(term, bytealigned=False)
    if not position:
        return None
    bit_pos = position[0]
    if bit_pos < 5:
        return None
    return bits[bit_pos - 5 : bit_pos].uint


def detect_player_indices(
    chunk_data: bytes,
    xuid_ints: dict[int, str],
) -> dict[int, int]:
    """Mappe player_index → xuid_int pour tous les joueurs du match.

    Utilise la méthode acurtis (inv #26) sur les données brutes d'un chunk.

    Args:
        chunk_data: Données brutes d'un chunk REPLICATION_DATA.
        xuid_ints: {xuid_int: gamertag} pour tous les joueurs du match.

    Returns:
        {player_index: xuid_int} — les joueurs non trouvés sont absents.
    """
    from bitstring import Bits as _Bits

    bits = _Bits(bytes=chunk_data)
    result: dict[int, int] = {}
    seen_indices: set[int] = set()

    for xuid_int, _gamertag in xuid_ints.items():
        pi = get_player_index_acurtis(bits, xuid_int)
        if pi is None:
            continue
        if pi in seen_indices:
            # Collision de player_index (inv #110) — on garde le premier
            continue
        result[pi] = xuid_int
        seen_indices.add(pi)

    return result
