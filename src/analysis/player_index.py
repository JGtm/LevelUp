"""Détection player_index via méthode acurtis (inv #26).

Identifie le player_index (0-7) de chaque joueur à partir de son XUID
dans les données brutes d'un chunk REPLICATION_DATA.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from bitstring import Bits


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
    from bitstring import Bits as _Bits

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
