"""Détection filmshell du début de match (spawn detection).

Fonctions pures — aucun accès DB, aucun appel API.
Entrée : chunks REPLICATION_DATA {index: (data, start_ms, dur_ms)}.
Sortie : timestamp estimé du début du match (ms depuis début d'enregistrement).

Algorithme :
1. Scan des frames de position pour chaque player_index dans chaque chunk.
2. Détection du début de mouvement soutenu par joueur (find_motion_onset).
   En lobby, les joueurs font 1-2 changements de caméra puis restent statiques.
   Au spawn réel, ils commencent à courir = changements continus (N+ dans 2s).
3. Densest cluster : fenêtre [t, t+2s] où le maximum de joueurs ont leur onset.
4. Estimation = médiane des timestamps du cluster.

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

#: Fenêtre temporelle pour détecter le cluster de spawn (ms).
#: Au début réel du match, tous les joueurs bougent dans cette fenêtre.
#: En lobby, les mouvements de caméra sont éparpillés → cluster moins dense.
_SPAWN_CLUSTER_WINDOW_MS: float = 2_000.0

#: Si l'estimation filmshell est en avance de plus de cette durée (ms)
#: sur le premier event API, un second passage avec ignore_before_ms est déclenché.
#: Le premier kill/death est une contrainte dure : le match a forcément démarré avant.
_LOBBY_CORRECTION_THRESHOLD_MS: float = 15_000.0

#: Marge de sécurité avant le premier event lors du second passage (ms).
_LOBBY_CORRECTION_BUFFER_MS: float = 10_000.0

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
    ignore_before_ms: float = 0.0,
) -> dict[int, FirstMovement]:
    """Détecte le premier CHANGEMENT de position pour chaque player_index.

    Algorithme :
    - Le premier frame vu par pi = baseline de spawn (position figée pendant le
      countdown, positions initiales dans la partie).
    - Le premier frame DIFFÉRENT de cette baseline = premier mouvement réel.
    - Les chunks sont traités en ordre croissant : chunk_01 fournit la baseline,
      chunk_02+ capturent le premier mouvement si le match démarre plus tard.
    - ``ignore_before_ms`` : si > 0, les changements détectés avant ce timestamp
      sont ignorés (mais la signature est quand même mise à jour). Cela permet
      au second passage de détecter le premier VRAI mouvement après le lobby.
    - Accepte tous les formats stream (b5 quelconque, b9 quelconque).

    Args:
        chunks: {chunk_index: (data, start_ms, dur_ms)}.
        ignore_before_ms: Ignorer les changements avant ce timestamp (ms).

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
                    ts = ts_fn(idx)
                    if ts >= ignore_before_ms:
                        coords = _decode_coords(data, idx)
                        first_change[pi] = FirstMovement(
                            timestamp_ms=ts,
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


# ── Pic d'activité simultanée ─────────────────────────────────────────────────


def find_peak_activity_window(
    chunks: dict[int, tuple[bytes, int, int]],
    window_ms: float = _SPAWN_CLUSTER_WINDOW_MS,
) -> float | None:
    """Trouve la fenêtre où le plus de joueurs DISTINCTS sont actifs simultanément.

    Scanne TOUS les changements de signature (pas seulement le premier par joueur).
    Au spawn réel, TOUS les joueurs se déplacent en même temps → pic maximal.
    En lobby, les sous-groupes bougent à des moments différents → pic partiel.

    En cas d'égalité de densité : la fenêtre la plus tardive est préférée
    (lobby précède toujours le spawn réel).

    Args:
        chunks: {chunk_index: (data, start_ms, dur_ms)}.
        window_ms: Durée de la fenêtre glissante (défaut: 2000ms).

    Returns:
        Timestamp (ms) du début de la fenêtre avec le plus de joueurs actifs,
        ou None si aucun changement détecté.
    """
    events: list[tuple[float, int]] = []  # (timestamp_ms, player_index)
    current_sig: dict[int, bytes] = {}

    for chunk_idx in sorted(chunks):
        data, start_ms, _ = chunks[chunk_idx]
        try:
            packets = index_chunk(data)
            ts_fn = build_packet_estimator(packets, float(start_ms))
        except Exception:
            _s = float(start_ms)
            ts_fn = lambda _p, _start=_s: _start  # noqa: E731

        pos = 0
        while True:
            idx = data.find(FRAME_MARKER, pos)
            if idx == -1:
                break
            pi = _is_position_frame(data, idx)
            if pi is not None:
                sig = bytes(data[idx + 9 : idx + 16])
                if pi not in current_sig:
                    current_sig[pi] = sig
                elif sig != current_sig[pi]:
                    current_sig[pi] = sig
                    events.append((ts_fn(idx), pi))
            pos = idx + 1

    if not events:
        return None

    events.sort()
    best_ts: float = events[0][0]
    best_count: int = 0

    for t_start, _ in events:
        t_end = t_start + window_ms
        unique = {pi for t, pi in events if t_start <= t <= t_end}
        if len(unique) > best_count or (len(unique) == best_count and t_start > best_ts):
            best_count = len(unique)
            best_ts = t_start

    return best_ts



def find_densest_spawn_cluster(
    first_movements: dict[int, FirstMovement],
    window_ms: float = _SPAWN_CLUSTER_WINDOW_MS,
) -> tuple[list[tuple[int, FirstMovement]], list[tuple[int, FirstMovement]]]:
    """Détecte le cluster de spawn le plus dense (vrai début de match).

    Fenêtre glissante sur les premiers mouvements de chaque joueur.
    La fenêtre [t, t + window_ms] avec le plus grand nombre de joueurs est
    retenue — au spawn réel, tous les joueurs bougent quasi-simultanément.
    En lobby, les mouvements de caméra sont éparpillés (1-2 joueurs).
    En cas d'égalité de densité : la fenêtre la plus tardive est préférée
    (les mouvements de lobby précèdent toujours le vrai spawn).

    Args:
        first_movements: {player_index: FirstMovement}.
        window_ms: Durée de la fenêtre glissante en ms (défaut: 2000ms).

    Returns:
        (cluster, outside) — chaque liste est [(player_index, FirstMovement)].
        ``cluster`` : joueurs dans la fenêtre la plus dense.
        ``outside`` : joueurs hors du cluster (mouvements isolés ou de lobby).
    """
    if not first_movements:
        return [], []

    sorted_items = sorted(first_movements.items(), key=lambda kv: kv[1]["timestamp_ms"])
    timestamps = [fm["timestamp_ms"] for _, fm in sorted_items]

    best_start_ts: float = timestamps[0]
    best_count: int = 0

    for t_start in timestamps:
        count = sum(1 for t in timestamps if t_start <= t <= t_start + window_ms)
        # Égalité → préférer la fenêtre la plus tardive (lobby < spawn réel)
        if count > best_count or (count == best_count and t_start > best_start_ts):
            best_count = count
            best_start_ts = t_start

    t_end = best_start_ts + window_ms
    cluster = [(pi, fm) for pi, fm in sorted_items if best_start_ts <= fm["timestamp_ms"] <= t_end]
    outside = [(pi, fm) for pi, fm in sorted_items if not (best_start_ts <= fm["timestamp_ms"] <= t_end)]

    earliest = cluster[0][1]["timestamp_ms"] if cluster else 0.0
    for _, fm in cluster:
        fm["delay_ms"] = fm["timestamp_ms"] - earliest
    for _, fm in outside:
        fm["delay_ms"] = fm["timestamp_ms"] - earliest

    return cluster, outside


def estimate_film_match_start_ms(
    chunks: dict[int, tuple[bytes, int, int]],
    min_players: int = 3,
    api_first_event_ms: float | None = None,
) -> int | None:
    """Estime le timestamp (ms) du début de match dans le film.

    Algorithme en deux étapes :
    1. ``find_peak_activity_window`` scanne TOUS les changements de signature
       et retient la fenêtre [t, t+2s] où le plus de joueurs distincts sont
       actifs simultanément. Plus robuste que l'ancien premier-mouvement seul.
    2. Si ``api_first_event_ms`` est fourni et que le gap dépasse
       ``_LOBBY_CORRECTION_THRESHOLD_MS``, un second scan est déclenché avec
       ``ignore_before_ms`` positionné avant le premier event connu.
       La contrainte dure : aucun kill/death ne peut précéder le début du match.

    Args:
        chunks: {chunk_index: (data, start_ms, dur_ms)}.
        min_players: Nombre minimum de références pour le second passage.
        api_first_event_ms: Timestamp du premier kill/death (même référentiel
            que le film). Contrainte dure utilisée pour valider et corriger.

    Returns:
        Timestamp en ms (entier) ou None si aucun changement détecté.
    """
    # Étape 1 : estimation initiale via pic d'activité simultanée
    peak_ts = find_peak_activity_window(chunks)
    if peak_ts is None:
        return None
    estimate = int(peak_ts)

    # Étape 2 : correction si l'estimate est trop précoce vs premier event API
    if api_first_event_ms is not None:
        gap_ms = api_first_event_ms - estimate
        if gap_ms > _LOBBY_CORRECTION_THRESHOLD_MS:
            ignore_before = estimate + gap_ms - _LOBBY_CORRECTION_BUFFER_MS
            first_movements_2 = scan_first_movements(chunks, ignore_before_ms=ignore_before)
            refs_2, _ = pick_spawn_references(first_movements_2, n=min_players)
            if len(refs_2) >= min_players:
                estimate_2 = int(statistics.median(fm["timestamp_ms"] for _, fm in refs_2))
                if estimate_2 > estimate:
                    estimate = estimate_2

    return estimate
