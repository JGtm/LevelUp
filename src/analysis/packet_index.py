"""Index structurel des paquets REPLICATION_DATA (acurtis 2026-03, inv #130).

Chaque chunk REPLICATION_DATA est une séquence de paquets avec un header 16 octets :
  - type       (uint16le)  — PacketType enum
  - byte2      (uint8)     — usage inconnu
  - byte3      (uint8)     — usage inconnu
  - size       (uint32le)  — taille du payload en octets
  - timestamp  (uint64le)  — timestamp en microsecondes

Fournit :
  - ``index_chunk()`` — index structurel de tous les paquets
  - ``build_packet_estimator()`` — timestamps µs précis (remplace frame markers)
  - ``extract_metadata_payload()`` — extrait le payload PLAYER_METADATA
  - ``detect_pi_from_metadata()`` — détection rapide des player_index (~25KB vs ~700KB)
"""

from __future__ import annotations

import struct
from collections.abc import Callable
from dataclasses import dataclass
from enum import IntEnum

# ══════════════════════════════════════════════════════════════════════════════
# Structures
# ══════════════════════════════════════════════════════════════════════════════

HEADER_STRUCT = struct.Struct("<HBBIQ")  # 16 bytes


class PacketType(IntEnum):
    """Types de paquets REPLICATION_DATA (inv #130)."""

    FRAME = 0  # ~258B, données de jeu (~60fps)
    START_CHUNK = 1  # Début de chunk
    INIT_STATE = 2  # ~155KB, état initial
    CHUNK_INIT = 6  # 4B
    END_CHUNK = 7  # Fin de chunk
    PLAYER_METADATA = 8  # ~25KB, contient mapping pi→xuid
    HEARTBEAT = 10  # 10B, sync
    INIT_MARKER = 12  # 4B zeros


@dataclass(slots=True)
class Packet:
    """Un paquet REPLICATION_DATA avec offset du payload."""

    type: int
    size: int
    microseconds: int
    offset: int  # byte offset du payload dans le chunk (juste après le header)


# ══════════════════════════════════════════════════════════════════════════════
# Indexation
# ══════════════════════════════════════════════════════════════════════════════


def index_chunk(data: bytes) -> list[Packet]:
    """Indexe tous les paquets d'un chunk REPLICATION_DATA.

    Parcourt les headers 16 bytes séquentiellement.
    Le dernier paquet est END_CHUNK (type 7).
    """
    packets: list[Packet] = []
    pos = 0
    unpack = HEADER_STRUCT.unpack_from
    end_type = PacketType.END_CHUNK
    data_len = len(data)
    while pos + 16 <= data_len:
        pkt_type, _b2, _b3, size, microseconds = unpack(data, pos)
        payload_offset = pos + 16
        packets.append(
            Packet(
                type=pkt_type,
                size=size,
                microseconds=microseconds,
                offset=payload_offset,
            )
        )
        if pkt_type == end_type:
            break
        pos = payload_offset + size
        if pos > data_len:
            break
    return packets


# ══════════════════════════════════════════════════════════════════════════════
# Timestamps précis (remplacement de build_frame_estimator)
# ══════════════════════════════════════════════════════════════════════════════


def build_packet_estimator(
    packets: list[Packet],
    chunk_start_ms: float,
) -> Callable[[int], float]:
    """Construit un estimateur ``byte_pos → timestamp_ms`` depuis l'index.

    Utilise les timestamps µs réels de chaque paquet FRAME au lieu de
    l'interpolation linéaire par marqueurs ``A0 7B 42`` (build_frame_estimator).

    Chaque fire event reçoit le timestamp de son paquet FRAME englobant.
    Précision : ~16ms (1 frame) au lieu de l'interpolation approximative.
    """
    frame_type = PacketType.FRAME
    frames = [
        (p.offset, p.offset + p.size, p.microseconds) for p in packets if p.type == frame_type
    ]
    if not frames:
        return lambda _: chunk_start_ms  # noqa: ARG005

    base_us = frames[0][2]

    def estimate_ts(byte_pos: int) -> float:
        """Recherche binaire du FRAME contenant byte_pos."""
        lo, hi = 0, len(frames) - 1
        frame_idx = 0
        while lo <= hi:
            mid = (lo + hi) // 2
            if frames[mid][0] <= byte_pos:
                frame_idx = mid
                lo = mid + 1
            else:
                hi = mid - 1
        return chunk_start_ms + (frames[frame_idx][2] - base_us) / 1000

    return estimate_ts


# ══════════════════════════════════════════════════════════════════════════════
# PLAYER_METADATA — extraction payload + détection pi
# ══════════════════════════════════════════════════════════════════════════════


def extract_metadata_payload(
    chunk_data: bytes,
    packets: list[Packet],
) -> bytes | None:
    """Extrait le payload PLAYER_METADATA (~25KB) d'un chunk indexé."""
    meta_type = PacketType.PLAYER_METADATA
    for pkt in packets:
        if pkt.type == meta_type:
            return chunk_data[pkt.offset : pkt.offset + pkt.size]
    return None


def detect_pi_from_metadata(
    metadata_payload: bytes,
    xuid_ints: dict[int, str],
) -> dict[int, int]:
    """Détecte les player_index depuis le PLAYER_METADATA payload.

    Plus rapide (~25KB vs ~700KB) et plus fiable que la recherche plein chunk.
    Cherche toutes les occurrences de chaque XUID et sélectionne celle avec pi ≠ 0.
    Le POV player (pi=0 uniquement) n'est pas retourné — correct car le POV est
    toujours pi=1 dans l'espace Section 2.

    Returns:
        ``{player_index: xuid_int}`` — même format que ``detect_player_indices()``.
    """
    from bitstring import Bits

    bits = Bits(bytes=metadata_payload)
    result: dict[int, int] = {}
    seen_pis: set[int] = set()

    for xuid_int in xuid_ints:
        term = Bits(uintle=xuid_int, length=64)
        for pos in bits.findall(term, bytealigned=False):
            if pos < 5:
                continue
            pi = bits[pos - 5 : pos].uint
            if pi != 0 and pi not in seen_pis:
                result[pi] = xuid_int
                seen_pis.add(pi)
                break

    return result
