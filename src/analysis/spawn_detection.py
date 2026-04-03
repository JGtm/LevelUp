"""Détection filmshell du début de match (spawn detection).

Fonctions pures — aucun accès DB, aucun appel API.
Entrée : chunks REPLICATION_DATA {index: (data, start_ms, dur_ms)}.
Sortie : timestamp estimé du début du match (ms depuis début d'enregistrement).

Algorithme :
1. Scan des frames de position pour chaque player_index dans chaque chunk.
2. Premier frame vu = baseline de spawn (joueurs figés pendant le countdown).
3. Premier frame DIFFÉRENT de la baseline = premier mouvement réel.
4. Estimation = médiane des timestamps des 3 joueurs les plus précoces.

Référentiel : identique à highlight_events.time_ms (t=0 = début enregistrement film).
"""

from __future__ import annotations

import statistics
from typing import TypedDict

from src.analysis.packet_index import build_packet_estimator, index_chunk

# ── Constantes filmshell ───────────────────────────────────────────────────────

FRAME_MARKER: bytes = bytes([0xA0, 0x7B, 0x42])

#: (b6 & 0x1F) valides pour un frame de position — tous formats de stream confondus
_VALID_BASE_TYPES: frozenset[int] = frozenset({0x08, 0x09, 0x0A, 0x0B, 0x28, 0x29})

_MIN_FRAME_LEN: int = 14

#: Retard max par rapport au premier référent : au-delà = suspect AFK
_AFK_THRESHOLD_MS: float = 10_000.0

_BYTE5_HUMAN: int = 0x40
_BYTE9_HUMAN: int = 0x56


# ── Types ──────────────────────────────────────────────────────────────────────


class FirstMovement(TypedDict):
    """Premier mouvement détecté pour un player_index."""

    timestamp_ms: float
    chunk: int
    b5: int
    b9: int
    y_raw: int | None
    x_raw: int | None
    delay_ms: float  # rempli par pick_spawn_references()


# ── Décodage frame de position ────────────────────────────────────────────────


def _is_position_frame(data: bytes, pos: int) -> int | None:
    """Retourne player_index si le marker est un frame de position valide.

    Filtre permissif : seul base_type (b6 & 0x1F) est vérifié, ce qui
    capture tous les joueurs quel que soit le format réseau.

    Returns:
        player_index (0-7) ou None si frame invalide.
    """
    if pos + _MIN_FRAME_LEN > len(data):
        return None
    b6 = data[pos + 6]
    if (b6 & 0x1F) not in _VALID_BASE_TYPES:
        return None
    return b6 >> 5


def _decode_coords(data: bytes, pos: int) -> tuple[int, int] | None:
    """Décode y_raw/x_raw pour les frames format human (b5=0x40, b9=0x56).

    Returns:
        (y_raw, x_raw) ou None si le format ne correspond pas.
    """
    if pos + _MIN_FRAME_LEN > len(data):
        return None
    if data[pos + 5] != _BYTE5_HUMAN or data[pos + 9] != _BYTE9_HUMAN:
        return None
    d0, d1, d2, d3 = data[pos + 10], data[pos + 11], data[pos + 12], data[pos + 13]
    if (d0 >> 4) != 4:
        return None
    return d0 * 256 + d1, (d2 & 0x0F) * 256 + d3


# ── Scan premier mouvement ────────────────────────────────────────────────────


def scan_first_movements(
    chunks: dict[int, tuple[bytes, int, int]],
) -> dict[int, FirstMovement]:
    """Détecte le premier CHANGEMENT de position pour chaque player_index.

    Algorithme :
    - Le premier frame vu par pi = baseline de spawn (position figée pendant le
      countdown, positions initiales dans la partie).
    - Le premier frame DIFFÉRENT de cette baseline = premier mouvement réel.
    - Les chunks sont traités en ordre croissant : chunk_01 fournit la baseline,
      chunk_02+ capturent le premier mouvement si le match démarre plus tard.
    - Accepte tous les formats stream (b5 quelconque, b9 quelconque).

    Args:
        chunks: {chunk_index: (data, start_ms, dur_ms)}.

    Returns:
        {player_index: FirstMovement} — y_raw/x_raw None si format non-human.
    """
    spawn_sig: dict[int, bytes] = {}
    first_change: dict[int, FirstMovement] = {}

    for chunk_idx in sorted(chunks):
        data, start_ms, _ = chunks[chunk_idx]
        try:
            packets = index_chunk(data)
            ts_fn = build_packet_estimator(packets, float(start_ms))
        except Exception:
            _start = float(start_ms)
            ts_fn = lambda _p, _s=_start: _s  # noqa: E731

        pos = 0
        while True:
            idx = data.find(FRAME_MARKER, pos)
            if idx == -1:
                break
            pi = _is_position_frame(data, idx)
            if pi is not None:
                sig = bytes(data[idx + 9 : idx + 16])
                if pi not in spawn_sig:
                    spawn_sig[pi] = sig
                elif sig != spawn_sig[pi] and pi not in first_change:
                    coords = _decode_coords(data, idx)
                    first_change[pi] = FirstMovement(
                        timestamp_ms=ts_fn(idx),
                        chunk=chunk_idx,
                        b5=data[idx + 5],
                        b9=data[idx + 9],
                        y_raw=coords[0] if coords else None,
                        x_raw=coords[1] if coords else None,
                        delay_ms=0.0,
                    )
                    spawn_sig[pi] = sig
            pos = idx + 1

    return first_change


def pick_spawn_references(
    first_movements: dict[int, FirstMovement],
    n: int = 3,
) -> tuple[list[tuple[int, FirstMovement]], list[tuple[int, FirstMovement]]]:
    """Trie les premiers mouvements et retourne (references, afk_suspects).

    Args:
        first_movements: Résultat de scan_first_movements().
        n: Nombre de références à garder (défaut: 3).

    Returns:
        Tuple (references, afk_suspects) — chaque liste est
        [(player_index, FirstMovement)].
        ``references`` : les n joueurs les plus précoces non-AFK.
        ``afk_suspects`` : joueurs avec un retard > _AFK_THRESHOLD_MS par
        rapport au premier référent, ou en surplus des n références.
    """
    if not first_movements:
        return [], []

    sorted_by_ts = sorted(first_movements.items(), key=lambda kv: kv[1]["timestamp_ms"])
    earliest_ms = sorted_by_ts[0][1]["timestamp_ms"]

    references: list[tuple[int, FirstMovement]] = []
    afk_suspects: list[tuple[int, FirstMovement]] = []

    for pi, fm in sorted_by_ts:
        fm["delay_ms"] = fm["timestamp_ms"] - earliest_ms
        if fm["delay_ms"] > _AFK_THRESHOLD_MS:
            afk_suspects.append((pi, fm))
        elif len(references) < n:
            references.append((pi, fm))
        else:
            afk_suspects.append((pi, fm))

    return references, afk_suspects


def estimate_film_match_start_ms(
    chunks: dict[int, tuple[bytes, int, int]],
    min_players: int = 3,
) -> int | None:
    """Calcule l'estimation du timestamp de début de match dans le film.

    Valeur = médiane des timestamps de premier mouvement des ``min_players``
    joueurs les plus précoces (non-AFK).

    Args:
        chunks: {chunk_index: (data, start_ms, dur_ms)}.
        min_players: Nombre minimum de références non-AFK requis.
            Si moins de joueurs sont détectés, retourne quand même
            la médiane de ce qui est disponible (avec 1 joueur minimum).

    Returns:
        Timestamp en ms (entier) ou None si aucun mouvement détecté.
    """
    first_movements = scan_first_movements(chunks)
    if not first_movements:
        return None

    refs, _ = pick_spawn_references(first_movements, n=min_players)
    if not refs:
        # Repli : prendre le premier joueur disponible même si < min_players
        refs, _ = pick_spawn_references(first_movements, n=1)
    if not refs:
        return None

    return int(statistics.median(fm["timestamp_ms"] for _, fm in refs))
