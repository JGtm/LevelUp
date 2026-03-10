#!/usr/bin/env python3
"""
Extraction expérimentale de l'arme utilisée lors d'un kill.

Approche v2 (basée sur filmshell / Den Delimarsky + Andy Curtis) :
  1. Charger les kills du joueur depuis shared_matches.duckdb
  2. Exclure les kills melee/grenade via medal_name (raw_json highlight_events)
  3. Dériver le player_index du tueur (order team_id ASC, rank ASC dans match_participants)
  4. Pour chaque kill à temps T, trouver le(s) chunk(s) REPLICATION_DATA couvrant [T-1000ms, T]
  5. Scanner les fire events (lead byte 0x0d, 4-bit nibble offset) dans ces chunks
  6. Filtrer par player_index, garder le dernier fire event avant T
  7. Mapper le weapon_id 8 bytes → nom de l'arme

Structure d'un fire event (dans les chunks REPLICATION_DATA, type 2) :
  Les fire events sont à 4-bit d'offset depuis les byte boundaries :
    Byte 0 : 0x0d  (lead byte)
    Byte 1 : (player_index << 5) | 0x06  [0x26=player1, 0x46=player2 ...]
    Byte 2 : 0x00  (constant)
    Byte 3 : 0x40  (constant, low 2 bits peuvent varier)
    Byte 4 : fire_counter  (incrémente de 4, wraps à 256)
    Byte 5 : weapon_slot   (0x01=primaire, 0x03=secondaire)
    Bytes 6-13 : weapon_id (8 bytes, table Andy Curtis)
    Byte 14 : aim_octant   (0–7)
    Bytes 15-16 : aim_vector (uint16)

Frame markers : A0 7B 42 délimitent les frames dans un chunk REPLICATION_DATA.
  - On compte les frames pour estimer les timestamps relatifs au chunk.

Référence : https://github.com/dend/filmshell
Branche    : experimental/film-weapon-extraction
"""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
import zlib
from pathlib import Path

from bitstring import Bits

# ─── Path setup ──────────────────────────────────────────────────────────────
PROJECT_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

# ─── Weapon IDs (8 bytes, Andy Curtis / filmshell) ───────────────────────────
# Source principale : Andy Curtis (github.com/acurtis166) — liste complète v2
# https://github.com/dend/blog-comments/issues/5#issuecomment-3882279646
# Note : les variantes/skins (ex: S7 Flexfire) partagent l'ID de l'arme de base.
WEAPON_ID_MAP: dict[bytes, str] = {
    # ── Liste confirmée Andy Curtis ──────────────────────────────────────────
    # pragma: allowlist secret
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
    # ── IDs filmshell / Andy Curtis (MAJ 2026-02-28) ─────────────────────────
    # Source : https://github.com/acurtis166/SPNKr (weapon event update)
    # ⚠️  Ces 3 armes ont des suffixes différents de 42c9679f :
    bytes.fromhex(
        "a0955e9e42c9679f"
    ): "Sentinel Beam",  # pragma: allowlist secret  # corrigé (ancien: c24e549e)
    bytes.fromhex(
        "841ac5e5a730e49f"
    ): "Gravity Hammer",  # pragma: allowlist secret  # corrigé (ancien: 8afc0855), suffixe a730e49f
    bytes.fromhex(
        "4ff3937e8978aa7a"
    ): "Energy Sword",  # pragma: allowlist secret  # corrigé (ancien: 1488d0bb), suffixe 8978aa7a
    bytes.fromhex("b6dbead842c9679f"): "Frag Grenade",  # pragma: allowlist secret
    bytes.fromhex("c1e1bab042c9679f"): "Plasma Grenade",  # pragma: allowlist secret
    # ── Nouveaux IDs (acurtis 2026-02-28) ────────────────────────────────────
    bytes.fromhex("1a22fee642c9679f"): "Shock Rifle (Ranked)",  # pragma: allowlist secret
    bytes.fromhex("880fe0bc42c9679f"): "Sandwich",  # pragma: allowlist secret
    bytes.fromhex(
        "b7262ca1c8fb11d0"
    ): "Mythic Sandwich",  # pragma: allowlist secret  # suffixe c8fb11d0
    # ── IDs filmshell alternatifs (possible variante PvE ou mise à jour jeu) ──
    # Ces IDs diffèrent de la liste Andy Curtis pour les mêmes armes.
    # Ils sont peut-être utilisés dans certains modes PvE ou après une MAJ.
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
    # ── IDs identifiés par analyse state blocks (grenades au sol — Inv. #17) ──
    # Ces IDs apparaissent UNIQUEMENT dans les state blocks (fire_event=0), jamais en 0x0D fire event.
    # Ils correspondent aux grenades posées sur la map (ramassables), pas aux grenades en inventaire joueur.
    bytes.fromhex("6683257c42c9679f"): "Spike Grenade (state-block)",  # pragma: allowlist secret
    bytes.fromhex("6d32c7dc42c9679f"): "Dynamo Grenade (state-block)",  # pragma: allowlist secret
}

# ─── Médailles indiquant un kill melee ou grenade (à exclure) ────────────────
MELEE_MEDALS: frozenset[str] = frozenset(
    {
        "Pummel",
        "Assassination",
        "Back Smack",
        "Melee",
        "Quigley",  # melee across map
    }
)

GRENADE_MEDALS: frozenset[str] = frozenset(
    {
        "Sticky Fingers",
        "Grenadier",
        "Boom!",
        "Kong",  # grenade-type medal (mentionné par l'utilisateur)
        "Stick",
        "Grenade Stick",
    }
)

FRAME_MARKER = bytes([0xA0, 0x7B, 0x42])
KILL_WINDOW_MS = 2000  # fenêtre de recherche avant le kill

# Suffixe commun à 22/23 armes connues (Andy Curtis) — sert de filtre rapide
COMMON_WEAPON_SUFFIX = bytes.fromhex("42c9679f")

# ─── Set des weapon IDs connus (entiers uint64) pour validation bitstring ────
WEAPON_IDS_INT: set[int] = set()
WEAPON_INT_TO_NAME: dict[int, str] = {}
for _wid_bytes, _wname in WEAPON_ID_MAP.items():
    _val = int.from_bytes(_wid_bytes, byteorder="big")
    WEAPON_IDS_INT.add(_val)
    WEAPON_INT_TO_NAME[_val] = _wname

# ─── Marqueur bitstring 11 bits (méthode acurtis) ───────────────────────────
# 3 LSBs du lead byte (101) + 8 bits de byte[1] (player_byte)
_WEAPON_BIT_OFFSET = 40  # bits après event_start pour atteindre weapon_id

DEFAULT_MATCH_ID = "e44bfaaa-877b-4893-99c0-aa94271bb8e9"
DEFAULT_GAMERTAG = "JGtm"


# ─── Chargement DB ────────────────────────────────────────────────────────────


def load_player_kills(match_id: str, gamertag: str, data_dir: Path | None = None) -> list[dict]:
    """
    Charge les kills du joueur depuis highlight_events + médailles proches.

    Retourne une liste de dicts :
      {time_ms, gamertag, xuid, medals_nearby: list[str]}
    """
    import duckdb

    base = data_dir or (PROJECT_ROOT / "data")
    db_path = base / "warehouse" / "shared_matches.duckdb"
    conn = duckdb.connect(str(db_path), read_only=True)

    # Kills du joueur
    kill_rows = conn.execute(
        """
        SELECT he.time_ms, he.gamertag, he.xuid
        FROM highlight_events he
        WHERE he.match_id = ?
          AND he.gamertag ILIKE ?
          AND he.event_type = 'kill'
        ORDER BY he.time_ms
        """,
        (match_id, gamertag),
    ).fetchall()

    # Médailles du joueur (pour détection melee/grenade)
    medal_rows = conn.execute(
        """
        SELECT he.time_ms,
               json_extract_string(he.raw_json, '$.medal_name') AS medal_name
        FROM highlight_events he
        WHERE he.match_id = ?
          AND he.gamertag ILIKE ?
          AND he.event_type = 'medal'
          AND json_extract_string(he.raw_json, '$.is_medal') = 'true'
        ORDER BY he.time_ms
        """,
        (match_id, gamertag),
    ).fetchall()

    conn.close()

    # Index medals par proximité temporelle
    medals_by_time: list[tuple[int, str]] = [(t, name) for t, name in medal_rows if name]

    kills = []
    for time_ms, gt, xuid in kill_rows:
        # Médailles dans une fenêtre ±500ms autour du kill
        nearby = [name for (mt, name) in medals_by_time if abs(mt - time_ms) <= 500]
        kills.append(
            {
                "time_ms": time_ms,
                "gamertag": gt,
                "xuid": xuid,
                "medals_nearby": nearby,
                "is_melee": any(m in MELEE_MEDALS for m in nearby),
                "is_grenade": any(m in GRENADE_MEDALS for m in nearby),
            }
        )

    return kills


def get_player_index(match_id: str, gamertag: str, data_dir: Path | None = None) -> int | None:
    """
    Dérive le player_index (0-7) à partir de l'ordre dans match_participants.

    Tri : team_id ASC, rank ASC → row index = player_index film.

    ATTENTION : cette heuristique n'est pas toujours exacte.
    Le film player_index peut différer de l'ordre DB (observé sur match e44bfaaa :
    DB donne idx=2 pour JGtm, mais le film utilise idx=1).
    En cas d'échec, utiliser --scan-all pour identifier le bon index.
    """
    import duckdb

    base = data_dir or (PROJECT_ROOT / "data")
    db_path = base / "warehouse" / "shared_matches.duckdb"
    conn = duckdb.connect(str(db_path), read_only=True)

    # Tous les participants dans l'ordre (sans JOIN pour ne pas filtrer les lignes)
    rows = conn.execute(
        """
        SELECT mp.xuid, mp.team_id, mp.rank
        FROM match_participants mp
        WHERE mp.match_id = ?
        ORDER BY mp.team_id ASC, mp.rank ASC
        """,
        (match_id,),
    ).fetchall()

    if not rows:
        conn.close()
        return None

    xuid_row = conn.execute(
        "SELECT xuid FROM xuid_aliases WHERE gamertag ILIKE ? LIMIT 1",
        (gamertag,),
    ).fetchone()
    conn.close()

    if not xuid_row:
        return None

    target_xuid = str(xuid_row[0])
    for idx, row in enumerate(rows):
        if str(row[0]) == target_xuid:
            return idx

    return None


# ─── Parsing chunks ───────────────────────────────────────────────────────────


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


def _build_frame_estimator(chunk_data: bytes, chunk_start_ms: int, chunk_duration_ms: int):
    """Retourne une fonction estimate_ts(byte_pos) → float basée sur les frame markers."""
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
    """
    Scanne les fire events via bitstring (méthode acurtis, inv #54).

    Recherche exhaustive à TOUTES les positions de bit (pas seulement byte/nibble).
    Validation par weapon_id ∈ WEAPON_IDS_INT (enum) + suffixe commun fallback.

    Structure fire event (relative à event_start = marqueur + 3 bits) :
      [0]:    byte1 = (player_index << 5) | 0x06   (ancre du marqueur)
      [+8b]:  fire_seq        identifiant arme/slot
      [+16b]: b3              0x40-0x43 (bits bas variables)
      [+24b]: fire_counter    0-127, incrémente de 1
      [+40b]: weapon_id       64 bits
      [+104b]: post_byte0     inconnu
      [+112b]: burst_end      bit 1 = dernier tir de rafale (BR, Shock Rifle)
      [+120b]: hit_miss       bit 1 = raté (~90% fiable, acurtis)

    Gains vs double-pass legacy (inv #54, 3 matchs, 79 chunks) :
      +300 events (+6.1%), 0 pertes. Events aux 6 bit-offsets non couverts
      par le double-pass byte+nibble. Toutes armes connues, b3 valide.
    """
    bits = Bits(bytes=chunk_data)
    marker = _build_marker(player_index)
    total_bits = len(bits)
    events: list[dict] = []

    for position in bits.findall(marker, bytealigned=False):
        event_start = position + 3  # Ancre sur byte[1] (0x26)
        weapon_start = event_start + _WEAPON_BIT_OFFSET

        if weapon_start + 64 > total_bits:
            continue

        weapon_int = bits[weapon_start : weapon_start + 64].uint

        # Validation : weapon_id connu (enum) ou suffixe commun
        weapon_bytes = weapon_int.to_bytes(8, byteorder="big")
        if weapon_int not in WEAPON_IDS_INT and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
            continue

        # Extraire les champs contextuels
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

        # Post-weapon bytes (burst_end, hit_miss — acurtis 2026-02-28)
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
    """
    Scanne les fire events pour un player_index dans un chunk REPLICATION_DATA.

    Utilise la méthode bitstring (acurtis, inv #54) : recherche exhaustive
    à toutes les positions de bit. +6.1% d'events vs l'ancien double-pass.
    Retourne une liste de {timestamp_ms, weapon_name, weapon_bytes, fire_seq,
                           fire_counter, bit_offset, byte_pos}.
    """
    estimate_ts = _build_frame_estimator(chunk_data, chunk_start_ms, chunk_duration_ms)
    return _scan_fire_events_bitstring(chunk_data, player_index, estimate_ts)


def scan_all_players(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    n_players: int = 8,
) -> dict[int, list[dict]]:
    """
    Scanne les fire events pour tous les player_index (0..n_players-1).
    Utile pour diagnostiquer lequel correspond au joueur cible.
    Retourne {player_index: [events]}.
    """
    estimate_ts = _build_frame_estimator(chunk_data, chunk_start_ms, chunk_duration_ms)
    return {
        idx: _scan_fire_events_bitstring(chunk_data, idx, estimate_ts) for idx in range(n_players)
    }


# ─── API Halo Infinite ───────────────────────────────────────────────────────


def _load_env_local() -> None:
    """Charge .env.local depuis le repo principal ou le worktree parent."""
    import os

    candidates = [
        PROJECT_ROOT / ".env.local",
        PROJECT_ROOT.parent / "LevelUp" / ".env.local",
    ]
    for p in candidates:
        if p.exists():
            try:
                from dotenv import load_dotenv  # type: ignore

                load_dotenv(p)
            except ImportError:
                with open(p, encoding="utf-8") as fh:
                    for line in fh:
                        line = line.strip()
                        if line and not line.startswith("#") and "=" in line:
                            k, _, v = line.partition("=")
                            os.environ.setdefault(k.strip(), v.strip())
            return


async def download_replication_chunks(
    match_id: str,
    kill_times_ms: list[int],
    cache_dir: Path,
) -> dict[int, tuple[bytes, int, int]]:
    """
    Télécharge les chunks REPLICATION_DATA nécessaires pour couvrir les kill_times.

    Retourne : {chunk_index: (decompressed_bytes, start_ms, duration_ms)}
    Cache les chunks dans cache_dir pour éviter les re-téléchargements.
    """
    _load_env_local()

    from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env

    tokens = await get_tokens_from_env()

    cache_dir.mkdir(parents=True, exist_ok=True)

    # requests_per_second=5 (défaut SPNKr) suffit — les blobs Azure ne sont pas
    # soumis au rate limiter SPNKr (téléchargés via session.get directement).
    async with SPNKrAPIClient(tokens=tokens) as api:
        film_response = await api.client.discovery_ugc.get_film_by_match_id(match_id)
        film = await film_response.parse()
        blob_prefix = film.blob_storage_path_prefix

        chunks_meta = film.custom_data.chunks

        # Identifier les chunks REPLICATION_DATA (type 2) nécessaires
        needed: set[int] = set()
        for kill_t in kill_times_ms:
            window_start = kill_t - KILL_WINDOW_MS
            for ch in chunks_meta:
                # FilmChunkType.REPLICATION_DATA = 2
                if ch.chunk_type.value != 2:
                    continue
                ch_start = ch.chunk_start_time_offset_milliseconds
                ch_end = ch_start + ch.duration_milliseconds
                # Le chunk couvre la fenêtre [window_start, kill_t] ?
                if ch_end >= window_start and ch_start <= kill_t:
                    needed.add(ch.index)

        print(f"  Chunks REPLICATION_DATA nécessaires : {sorted(needed)}")

        # Séparer les chunks déjà en cache de ceux à télécharger
        to_download = []
        result: dict[int, tuple[bytes, int, int]] = {}

        for ch in sorted(chunks_meta, key=lambda c: c.index):
            if ch.index not in needed:
                continue
            cache_path = cache_dir / f"chunk_{ch.index:02d}.bin"
            if cache_path.exists():
                data = cache_path.read_bytes()
                print(f"  [cache] filmChunk{ch.index} ({len(data):,} bytes)")
                result[ch.index] = (
                    data,
                    ch.chunk_start_time_offset_milliseconds,
                    ch.duration_milliseconds,
                )
            else:
                to_download.append(ch)

        # Télécharger les chunks manquants en parallèle (blobs Azure, pas de rate limit)
        if to_download:

            async def _fetch_chunk(ch) -> tuple[int, bytes, int, int]:
                """Télécharge et décompresse un chunk blob Azure."""
                url = blob_prefix + ch.file_relative_path.lstrip("/")
                resp = await api.client._session.get(url)
                resp.raise_for_status()
                raw = await resp.read()
                data = zlib.decompress(raw)
                cache_path = cache_dir / f"chunk_{ch.index:02d}.bin"
                cache_path.write_bytes(data)
                print(
                    f"  [dl]    filmChunk{ch.index} "
                    f"({ch.chunk_size:,} -> {len(data):,} bytes décompressés)"
                )
                return (
                    ch.index,
                    data,
                    ch.chunk_start_time_offset_milliseconds,
                    ch.duration_milliseconds,
                )

            downloaded = await asyncio.gather(*(_fetch_chunk(ch) for ch in to_download))
            for idx, data, start_ms, dur_ms in downloaded:
                result[idx] = (data, start_ms, dur_ms)

    return result


# ─── Corrélation ─────────────────────────────────────────────────────────────


def correlate_kills_to_weapons(
    kills: list[dict],
    fire_events_all: list[dict],
) -> list[dict]:
    """
    Pour chaque kill à T, cherche le dernier fire event dans [T-1000ms, T].
    Exclut les kills melee/grenade.
    """
    results = []

    for kill in kills:
        kill_t = kill["time_ms"]

        if kill["is_melee"]:
            results.append(
                {
                    **kill,
                    "weapon_name": "MELEE (exclu)",
                    "weapon_slot": None,
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
                    "weapon_slot": None,
                    "matched_fire_event": None,
                    "delta_ms": None,
                }
            )
            continue

        # Fenêtre [kill_t - 1000, kill_t]
        candidates = [
            ev
            for ev in fire_events_all
            if (kill_t - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= kill_t
        ]

        best: dict | None = None
        if candidates:
            # Le plus proche précédant le kill (timestamp le plus grand ≤ kill_t)
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


# ─── Affichage ───────────────────────────────────────────────────────────────


def print_results(correlated: list[dict]) -> None:
    print("\n" + "=" * 70)
    print("RESULTATS — ARME PAR KILL (fire events REPLICATION_DATA)")
    print("=" * 70)

    total = len(correlated)
    matched = sum(1 for r in correlated if r["matched_fire_event"] is not None)
    excluded = sum(1 for r in correlated if r.get("is_melee") or r.get("is_grenade"))
    unknown = sum(
        1 for r in correlated if r["matched_fire_event"] and r["weapon_name"].startswith("INCONNU")
    )

    for i, r in enumerate(correlated, 1):
        t_s = (r["time_ms"] or 0) // 1000
        t_min, t_sec = divmod(t_s, 60)
        weapon = r["weapon_name"]
        delta = r.get("delta_ms")
        seq = r.get("fire_seq")
        slot_str = f"[seq {seq}]" if seq is not None else ""
        delta_str = f" (delta {delta}ms)" if delta is not None else ""
        medals_str = f" [{', '.join(r['medals_nearby'])}]" if r.get("medals_nearby") else ""
        print(
            f"  [{i:02d}] {t_min}:{t_sec:02d}  "
            f"{weapon:<30} {slot_str:<10}{delta_str}{medals_str}"
        )

    print()
    print(f"  Total kills      : {total}")
    print(f"  Fire event trouvé: {matched} / {total - excluded}")
    print(f"  Exclus (melee/gr): {excluded}")
    print(f"  IDs inconnus     : {unknown}")

    # Résumé des weapon IDs inconnus
    unknowns: dict[str, int] = {}
    for r in correlated:
        ev = r.get("matched_fire_event")
        if ev and r["weapon_name"].startswith("INCONNU"):
            wid = ev["weapon_bytes"].hex()
            unknowns[wid] = unknowns.get(wid, 0) + 1

    if unknowns:
        print("\n  IDs à identifier (weapon_bytes hex → count) :")
        for wid, cnt in sorted(unknowns.items(), key=lambda x: -x[1]):
            print(f"    {wid} = {cnt} kill(s)")


# ─── Main ─────────────────────────────────────────────────────────────────────


async def main_async(
    match_id: str,
    gamertag: str,
    save_json: bool = False,
    data_dir: Path | None = None,
    player_index_override: int | None = None,
    scan_all: bool = False,
) -> int:
    print("\nExtraction arme-par-kill v2 (fire events / filmshell)")
    print(f"   Match    : {match_id}")
    print(f"   Joueur   : {gamertag}")
    print(f"   Fenetre  : -{KILL_WINDOW_MS}ms avant chaque kill")
    if data_dir:
        print(f"   Data dir : {data_dir}")
    print()

    # 1. Kills + médailles proches
    print("1) Chargement des kills et médailles...")
    kills = load_player_kills(match_id, gamertag, data_dir)
    if not kills:
        print(f"  x Aucun kill trouvé pour {gamertag}.")
        return 1
    n_melee = sum(1 for k in kills if k["is_melee"])
    n_grenade = sum(1 for k in kills if k["is_grenade"])
    print(f"  OK {len(kills)} kills ({n_melee} melee, {n_grenade} grenade)")

    # 2. Player index
    print("2) Détermination du player_index...")
    if player_index_override is not None:
        player_index = player_index_override
        print(f"  OK player_index={player_index}  (forcé via --player-index)")
    else:
        player_index = get_player_index(match_id, gamertag, data_dir)
        if player_index is None:
            print("  x Impossible de déterminer le player_index.")
            return 1
        print(f"  OK player_index={player_index}  (dérivé de match_participants)")
    player_byte1 = (player_index << 5) | 0x06
    print(f"     byte1 fire event = 0x{player_byte1:02x}")

    # 3. Téléchargement des chunks REPLICATION_DATA
    print("3) Téléchargement des chunks REPLICATION_DATA...")
    cache_dir = PROJECT_ROOT / "data" / "investigation" / "chunks" / match_id[:8]
    kill_times = [k["time_ms"] for k in kills]

    try:
        chunks = await download_replication_chunks(match_id, kill_times, cache_dir)
    except Exception as e:
        print(f"  x Erreur téléchargement : {e}")
        return 1
    print(f"  OK {len(chunks)} chunk(s) chargés")

    # 4. Scan fire events dans tous les chunks
    print("4) Scan fire events (0x0d, nibble alignment)...")

    if scan_all:
        # Mode diagnostic : compter les events par player_index pour identifier le bon
        print("   [--scan-all] Comptage pour tous les player_index (0-7) :")
        totals: dict[int, int] = {i: 0 for i in range(8)}
        for _, (chunk_data, start_ms, duration_ms) in sorted(chunks.items()):
            per_player = scan_all_players(chunk_data, start_ms, duration_ms)
            for pidx, evts in per_player.items():
                totals[pidx] += len(evts)
        print(f"   {'idx':>4}  {'byte1':>6}  {'events':>7}")
        for pidx, cnt in totals.items():
            b1 = (pidx << 5) | 0x06
            marker = " <-- dérivé DB" if pidx == player_index else ""
            print(f"   {pidx:>4}  0x{b1:02x}    {cnt:>7}{marker}")
        print()

    all_fire_events: list[dict] = []
    for chunk_idx, (chunk_data, start_ms, duration_ms) in sorted(chunks.items()):
        evts = scan_fire_events(chunk_data, player_index, start_ms, duration_ms)
        print(
            f"  chunk {chunk_idx:02d} [{start_ms//1000}s-{(start_ms+duration_ms)//1000}s] : "
            f"{len(evts)} fire events"
        )
        all_fire_events.extend(evts)

    all_fire_events.sort(key=lambda e: e["timestamp_ms"])
    print(f"  Total fire events : {len(all_fire_events)}")

    if not all_fire_events:
        print("  x Aucun fire event trouvé — player_index probablement incorrect")
        # Diagnostic automatique si pas déjà fait
        if not scan_all:
            print("   Diagnostic automatique (comptage par player_index) :")
            totals2: dict[int, int] = {i: 0 for i in range(8)}
            for chunk_data, start_ms, duration_ms in chunks.values():
                for pidx, evts in scan_all_players(chunk_data, start_ms, duration_ms).items():
                    totals2[pidx] += len(evts)
            for pidx, cnt in totals2.items():
                b1 = (pidx << 5) | 0x06
                marker = " <-- utilisé" if pidx == player_index else ""
                print(f"    idx={pidx} 0x{b1:02x} -> {cnt} events{marker}")
            best_idx = max(totals2, key=lambda k: totals2[k])
            if totals2[best_idx] > 0:
                print(f"   Suggestion : relancer avec --player-index {best_idx}")
        return 1

    # 5. Corrélation kills ↔ fire events
    print("5) Corrélation kills <-> fire events...")
    correlated = correlate_kills_to_weapons(kills, all_fire_events)

    # 6. Affichage
    print_results(correlated)

    # 7. Export JSON optionnel
    if save_json:
        out_path = PROJECT_ROOT / "data" / "investigation" / f"weapon_v2_{match_id[:8]}.json"
        out_path.parent.mkdir(parents=True, exist_ok=True)
        with open(out_path, "w", encoding="utf-8") as f:
            json.dump(correlated, f, indent=2, default=str)
        print(f"\n  Resultats sauvegardes : {out_path}")

    return 0


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Extraire l'arme utilisée lors de chaque kill (v2 - fire events)"
    )
    parser.add_argument(
        "--match-id",
        default=DEFAULT_MATCH_ID,
        help=f"ID du match (défaut: {DEFAULT_MATCH_ID[:8]}...)",
    )
    parser.add_argument(
        "--gamertag",
        default=DEFAULT_GAMERTAG,
        help=f"Gamertag du joueur (défaut: {DEFAULT_GAMERTAG})",
    )
    parser.add_argument(
        "--save-json",
        action="store_true",
        help="Sauvegarder les résultats dans data/investigation/",
    )
    parser.add_argument(
        "--data-dir",
        type=Path,
        default=None,
        help="Chemin vers le dossier data/ (défaut: auto-détecté)",
    )
    parser.add_argument(
        "--player-index",
        type=int,
        default=None,
        help="Forcer le player_index (0-7) au lieu de l'auto-détecter",
    )
    parser.add_argument(
        "--scan-all",
        action="store_true",
        help="Afficher le comptage de fire events pour tous les player_index (diagnostic)",
    )
    args = parser.parse_args()

    # Auto-détection data_dir
    data_dir = args.data_dir
    if data_dir is None:
        for c in [
            PROJECT_ROOT / "data",
            PROJECT_ROOT.parent / "LevelUp" / "data",
        ]:
            if (c / "warehouse" / "shared_matches.duckdb").exists():
                data_dir = c
                break

    return asyncio.run(
        main_async(
            args.match_id,
            args.gamertag,
            args.save_json,
            data_dir,
            player_index_override=args.player_index,
            scan_all=args.scan_all,
        )
    )


if __name__ == "__main__":
    sys.exit(main())
