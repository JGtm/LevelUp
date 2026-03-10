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

# ══════════════════════════════════════════════════════════════════════════════
# Constantes : Weapon IDs (8 bytes, Andy Curtis / filmshell)
# ══════════════════════════════════════════════════════════════════════════════

WEAPON_ID_MAP: dict[bytes, str] = {
    # ── Liste confirmée Andy Curtis ──────────────────────────────────────
    bytes.fromhex("6acdc44d42c9679f"): "Bandit Evo",  # pragma: allowlist secret
    bytes.fromhex("2b1824d542c9679f"): "BR75",  # pragma: allowlist secret
    bytes.fromhex("230447b142c9679f"): "Cindershot",  # pragma: allowlist secret
    bytes.fromhex("b619d84a42c9679f"): "CQS48 Bulldog",  # pragma: allowlist secret
    bytes.fromhex("84bd29ed42c9679f"): "Disruptor",  # pragma: allowlist secret
    bytes.fromhex("9d6aaed242c9679f"): "Fuel Rod SPNKr",  # pragma: allowlist secret
    bytes.fromhex("2ac9c2ff42c9679f"): "Heatwave",  # pragma: allowlist secret
    bytes.fromhex("71ab0a2c42c9679f"): "M41 SPNKr",  # pragma: allowlist secret
    bytes.fromhex("2fb21c8742c9679f"): "M392 Bandit",  # pragma: allowlist secret
    bytes.fromhex("48c19d2d42c9679f"): "MA40 AR",  # pragma: allowlist secret
    bytes.fromhex("f5c335dfe7232c0b"): "MA5K Avenger",  # pragma: allowlist secret
    bytes.fromhex("80977ba542c9679f"): "Mangler",  # pragma: allowlist secret
    bytes.fromhex("767db96d42c9679f"): "MLRS-2 Hydra",  # pragma: allowlist secret
    bytes.fromhex("f408190f42c9679f"): "Mk51 Sidekick",  # pragma: allowlist secret
    bytes.fromhex("d791556542c9679f"): "Mutilator",  # pragma: allowlist secret
    bytes.fromhex("b533957e42c9679f"): "Needler",  # pragma: allowlist secret
    bytes.fromhex("c354294642c9679f"): "Plasma Pistol",  # pragma: allowlist secret
    bytes.fromhex("30484ea642c9679f"): "Pulse Carbine",  # pragma: allowlist secret
    bytes.fromhex("c30d87c742c9679f"): "Ravager",  # pragma: allowlist secret
    bytes.fromhex("0a1992bc42c9679f"): "S7 Sniper",  # pragma: allowlist secret
    bytes.fromhex("9387a8b942c9679f"): "Shock Rifle",  # pragma: allowlist secret
    bytes.fromhex("0d20c46942c9679f"): "Skewer",  # pragma: allowlist secret
    bytes.fromhex("daf193c742c9679f"): "Stalker Rifle",  # pragma: allowlist secret
    bytes.fromhex("fd98554c42c9679f"): "VK78 Commando",  # pragma: allowlist secret
    bytes.fromhex("3e07021742c9679f"): "Vestige Carbine",  # pragma: allowlist secret
    # ── IDs filmshell / Andy Curtis (MAJ 2026-02-28) ─────────────────────
    bytes.fromhex("a0955e9e42c9679f"): "Sentinel Beam",  # pragma: allowlist secret
    bytes.fromhex("841ac5e5a730e49f"): "Gravity Hammer",  # pragma: allowlist secret
    bytes.fromhex("4ff3937e8978aa7a"): "Energy Sword",  # pragma: allowlist secret
    bytes.fromhex("b6dbead842c9679f"): "Frag Grenade",  # pragma: allowlist secret
    bytes.fromhex("c1e1bab042c9679f"): "Plasma Grenade",  # pragma: allowlist secret
    # ── Nouveaux IDs (acurtis 2026-02-28) ────────────────────────────────
    bytes.fromhex("1a22fee642c9679f"): "Shock Rifle (Ranked)",  # pragma: allowlist secret
    bytes.fromhex("880fe0bc42c9679f"): "Sandwich",  # pragma: allowlist secret
    bytes.fromhex("b7262ca1c8fb11d0"): "Mythic Sandwich",  # pragma: allowlist secret
    # ── IDs filmshell alternatifs (possible variante PvE ou MAJ jeu) ─────
    bytes.fromhex("7e53b3c642c9679f"): "Pulse Carbine (alt)",  # pragma: allowlist secret
    bytes.fromhex("04e7f00b42c9679f"): "Plasma Pistol (alt)",  # pragma: allowlist secret
    bytes.fromhex("3d34488542c9679f"): "Heatwave (alt)",  # pragma: allowlist secret
    bytes.fromhex("f5ef3bdb42c9679f"): "Stalker Rifle (alt)",  # pragma: allowlist secret
    bytes.fromhex("fcc6aa7642c9679f"): "Shock Rifle (alt)",  # pragma: allowlist secret
    bytes.fromhex("7deb133f42c9679f"): "Mangler (alt)",  # pragma: allowlist secret
    bytes.fromhex("cb30ec5e42c9679f"): "Disruptor (alt)",  # pragma: allowlist secret
    bytes.fromhex("2b1d61e442c9679f"): "Ravager (alt)",  # pragma: allowlist secret
    bytes.fromhex("7a11aeef42c9679f"): "Skewer (alt)",  # pragma: allowlist secret
    bytes.fromhex("c2a6d5e042c9679f"): "Cindershot (alt)",  # pragma: allowlist secret
    bytes.fromhex("1f6ae65542c9679f"): "MLRS-2 Hydra (alt)",  # pragma: allowlist secret
    # ── State blocks (grenades au sol, pas en fire event) ────────────────
    bytes.fromhex("6683257c42c9679f"): "Spike Grenade (state-block)",  # pragma: allowlist secret
    bytes.fromhex("6d32c7dc42c9679f"): "Dynamo Grenade (state-block)",  # pragma: allowlist secret
}

# ══════════════════════════════════════════════════════════════════════════════
# Médailles indiquant un kill melee ou grenade (à exclure)
# ══════════════════════════════════════════════════════════════════════════════

MELEE_MEDALS: frozenset[str] = frozenset(
    {"Pummel", "Assassination", "Back Smack", "Melee", "Quigley"}
)

GRENADE_MEDALS: frozenset[str] = frozenset(
    {"Sticky Fingers", "Grenadier", "Boom!", "Kong", "Stick", "Grenade Stick"}
)

# ══════════════════════════════════════════════════════════════════════════════
# Constantes de parsing
# ══════════════════════════════════════════════════════════════════════════════

FRAME_MARKER = bytes([0xA0, 0x7B, 0x42])
KILL_WINDOW_MS = 2000

COMMON_WEAPON_SUFFIX = bytes.fromhex("42c9679f")

_WEAPON_BIT_OFFSET = 40  # bits après event_start → weapon_id (64 bits)

# Sets pré-calculés pour validation rapide
WEAPON_IDS_INT: set[int] = set()
WEAPON_INT_TO_NAME: dict[int, str] = {}
for _wid_bytes, _wname in WEAPON_ID_MAP.items():
    _val = int.from_bytes(_wid_bytes, byteorder="big")
    WEAPON_IDS_INT.add(_val)
    WEAPON_INT_TO_NAME[_val] = _wname

# ══════════════════════════════════════════════════════════════════════════════
# Mapping film_bytes → API weapon_id (entier)
# ══════════════════════════════════════════════════════════════════════════════

FILM_NAME_TO_API_ID: dict[str, int] = {
    # UNSC
    "Bandit Evo": 10090,
    "BR75": 41533,
    "CQS48 Bulldog": 36844,
    "M41 SPNKr": 75491,
    "MA40 AR": 73886,
    "MLRS-2 Hydra": 4447,
    "Mk51 Sidekick": 19954,
    "S7 Sniper": 79993,
    "VK78 Commando": 94689,
    # Covenant
    "Energy Sword": 95667,
    "Needler": 76498,
    "Plasma Pistol": 117,
    "Pulse Carbine": 44817,
    "Stalker Rifle": 59527,
    # Banished
    "Gravity Hammer": 6943,
    "Mangler": 14717,
    "Ravager": 7717,
    "Skewer": 103477,
    # Forerunner
    "Cindershot": 38834,
    "Disruptor": 77817,
    "Heatwave": 22223,
    "Sentinel Beam": 69099,
    "Shock Rifle": 58947,
    "Shock Rifle (Ranked)": 58947,
    # Grenades → API ID 0 (générique)
    "Frag Grenade": 0,
    "Plasma Grenade": 0,
}

FILM_BYTES_TO_API_ID: dict[bytes, int | None] = {}
for _wbytes, _wname in WEAPON_ID_MAP.items():
    _canon = _wname.split(" (")[0]  # retire "(alt)", "(state-block)", etc.
    if _canon in FILM_NAME_TO_API_ID:
        FILM_BYTES_TO_API_ID[_wbytes] = FILM_NAME_TO_API_ID[_canon]
    elif _wname in FILM_NAME_TO_API_ID:
        FILM_BYTES_TO_API_ID[_wbytes] = FILM_NAME_TO_API_ID[_wname]
    else:
        FILM_BYTES_TO_API_ID[_wbytes] = None

# POV player_index (invariant universel — inv #6, #23, #27, #41)
POV_PLAYER_INDEX = 1

# API IDs synthétiques pour melee/grenade/véhicule
MELEE_API_ID = 1
GRENADE_API_ID = 0
VEHICLE_API_ID = 2


# ══════════════════════════════════════════════════════════════════════════════
# Fonctions de parsing pur
# ══════════════════════════════════════════════════════════════════════════════


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


def correlate_kills_to_weapons(
    kills: list[dict],
    fire_events_all: list[dict],
) -> list[dict]:
    """Pour chaque kill à T, cherche le dernier fire event dans [T-KILL_WINDOW, T].

    Exclut les kills melee/grenade de la recherche fire event.
    """
    results = []
    for kill in kills:
        kill_t = kill["time_ms"]
        if kill["is_melee"]:
            results.append(
                {
                    **kill,
                    "weapon_name": "MELEE (exclu)",
                    "matched_fire_event": None,
                    "delta_ms": None,
                }
            )
            continue
        if kill["is_grenade"]:
            results.append(
                {
                    **kill,
                    "weapon_name": "GRENADE (exclu)",
                    "matched_fire_event": None,
                    "delta_ms": None,
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
        results.append(
            {
                **kill,
                "weapon_name": best["weapon_name"] if best else "NON TROUVE",
                "fire_seq": best["fire_seq"] if best else None,
                "matched_fire_event": best,
                "delta_ms": int(kill_t - best["timestamp_ms"]) if best else None,
            }
        )
    return results


# ══════════════════════════════════════════════════════════════════════════════
# Agrégation kills → Counter par API weapon_id
# ══════════════════════════════════════════════════════════════════════════════


def count_kills_by_api_weapon(correlated: list[dict]) -> Counter:
    """Agrège les kills corrélés en Counter {api_weapon_id: n_kills}.

    Kills melee → weapon_id=1, grenade → weapon_id=0.
    Kills sans match / armes non mappées → ignorés.
    """
    counts: Counter = Counter()
    for r in correlated:
        if r.get("is_melee"):
            counts[MELEE_API_ID] += 1
            continue
        if r.get("is_grenade"):
            counts[GRENADE_API_ID] += 1
            continue
        ev = r.get("matched_fire_event")
        if not ev:
            continue
        wbytes = ev.get("weapon_bytes")
        if not wbytes:
            continue
        api_id = FILM_BYTES_TO_API_ID.get(wbytes)
        if api_id is None:
            continue
        counts[api_id] += 1
    return counts
