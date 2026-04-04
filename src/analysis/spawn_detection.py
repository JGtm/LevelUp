"""Détection filmshell du début de match (spawn detection).

Fonctions pures — aucun accès DB, aucun appel API.
Entrée : chunks REPLICATION_DATA {index: (data, start_ms, dur_ms)}.
Sortie : timestamp estimé du début du match (ms depuis début d'enregistrement).

Algorithme actuel :
1. Scan des changements de signature de bytes[9:16] par player_index.
2. Fenêtre glissante [t, t+2s] où le plus de joueurs sont actifs simultanément.
3. Estimation = timestamp du début de cette fenêtre.
4. Correction-cap API : si l'estimate dépasse le premier kill/death connu,
   on recule à api_first_event_ms - 5s.

Référentiel : identique à highlight_events.time_ms (t=0 = début enregistrement film).

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ÉTAT DE LA RECHERCHE — Dernière mise à jour : 2026-04-04
Performance actuelle : 55% à ±5s, 60% à ±10s, 91% à ±30s (198 matchs)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## Ce qui a été testé et pourquoi ça n'a pas fonctionné

### 1. Approche par discontinuité de coordonnées (spawn = téléportation)
Hypothèse : à la mort/spawn, le joueur se téléporte → grand saut de coordonnées.
Test : delta Y/X entre frames consécutifs, seuil > 4000 (valeur filmshell DISCONTINUITY_THRESHOLD).
Résultat : ÉCHEC. Le champ X est encodé sur 12 bits (range 0-4095).
Un mouvement normal peut traverser 0→4095 (wraparound) générant un delta de 4096
sans téléportation réelle. Les deltas "Y" oscillent entre 0 et ±1 sur Fortress lobby.
La correction de wraparound ramène tous les deltas sous 2048 → impossible de distinguer.

### 2. Filtre strict sur format human (b5=0x40, b7=0x00, b9=0x56, d0hnib=4)
Hypothèse : seul ce format code de vraies positions spatiales (confirmé filmshell docs).
Test : exclure tous les frames d'autres formats, ne garder que le format "human".
Résultat : ÉCHEC partiel.
 - Sur Fortress : élimine correctement les faux positifs de lobby à 12.3s
   (causés par des frames b5=0x80 avec b9 variable → faux sig-changes).
 - Sur d'autres maps (ex: 5aa360c3) : le format b5=0x80, base=0x0B est le format
   DOMINANT pendant tout le match, pas uniquement en lobby. Le filtre strict le supprime
   et rate les 14-37 premières secondes de match réel.
 - Performance globale : 31% à ±5s vs 47% baseline → régression de -16 points.

### 3. Correction API via second scan (ignore_before_ms)
Hypothèse : si l'estimate est trop précoce vs premier kill/death, relancer un scan
depuis ignore_before = estimate + gap - 10s pour trouver le vrai spawn.
Test : appliqué à Fortress (41b61fb9) avec api_first_event_ms=35s → donne 34.1s (correct).
Résultat : ÉCHEC sur les grandes maps.
 - Sur une grande map avec travel time 30-45s : le premier kill arrive à ~60s,
   l'estimate correct (spawn à 15s) génère gap=45s → second scan depuis 50s
   → détecte une activité en cours de match à 55s → écrase le bon résultat.
 - La prémisse « grand gap = lobby indétectable » est fausse : le gap peut être
   grand simplement parce que les joueurs n'ont pas encore croisé d'ennemi.
 - Stratégie remplacée par une simple contrainte dure (cap) : si estimate > premier kill,
   on recule uniquement. Un grand gap est désormais ignoré.

### 4. Approche par déplacement vectoriel (delta Y²+X² entre frames consécutifs)
Hypothèse : en lobby, mouvements "courts et restreints" → faibles deltas par frame.
          Au spawn réel, les joueurs courent → grands deltas → pic dense simultané.
Test : seuils 5, 10, 20, 30, 50 unités filmshell testés sur 200 matchs.
Résultat : ÉCHEC. La prémisse est fausse.
 - Dans la staging area Halo, les joueurs se déplacent à vitesse normale.
 - Le déplacement par frame entre deux positions consécutives est identique en lobby
   et en match (même format de frames, même vitesse de marche/course).
 - Ce qui diffère réellement : la SURFACE TOTALE de trajectoire (bounding box)
   sur 5-10s. En lobby : zone restreinte → bounding box petite.
   En match : progression linéaire vers objectif → bounding box expansive.
 - Pour exploiter ça, il faudrait calculer l'expansion de la bounding box par joueur
   sur une fenêtre glissante de 5-10s. Mais les frames b5=0x40 décodables ne couvrent
   que 23% des matchs dans les chunks 1-2 (les autres maps utilisent b5=0x80).
 - Performance sur les 23% couverts : 34% à ±5s vs 61% avec la signature → régression.
 - Infrastructure présente (find_peak_displacement_window, _collect_displacement_events,
   _displacement) mais NON UTILISÉE dans estimate_film_match_start_ms.

## Explication du résultat actuel (55% à ±5s)

L'algorithme signature confond lobby et spawn sur les matchs où :
 A) Un sous-groupe de joueurs (3+) fait des mouvements de caméra groupés en lobby
    (rare mais arrive sur BTB en attente de joueurs → cluster dense mais précoce).
 B) Le match commence DANS le chunk_01 (< 20s depuis recording start) sans lobby
    filmshell visible → signature correcte mais on mesure le premier mouvement
    absolu plutôt que le spawn.
 C) AFK ou chargement lent d'un joueur → cluster < min_players dans la fenêtre spawn.

## Piste la plus prometteuse non testée

Analyse de la bounding box expansive par joueur sur 5-10s glissants :
 - Filtrer uniquement les frames b5=0x40 (représente 23% des matchs actuellement).
 - Pour chaque joueur : calculer la bounding box de ses positions sur la fenêtre [t, t+5s].
 - Lobby : bounding box < 50 unités (zone de staging restreinte).
 - Match réel : bounding box > 200 unités (progression vers objectif).
 - Utiliser le moment où la médiane des bounding boxes d'au moins 3 joueurs dépasse
   le seuil → timestamp du début de match.
 - Limite : nécessite une meilleure couverture de frames b5=0x40 en début de match.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"""

from __future__ import annotations

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

#: Déplacement minimum (unités filmshell) entre deux frames consécutifs d'un joueur
#: pour que le frame soit compté comme un mouvement réel.
#: En lobby : petits déplacements non-soutenus (walking restreint).
#: Au spawn : course vers objectif → grands deltas consécutifs sur tous les joueurs.
#: Valeur empirique — calibrer si nécessaire selon le scale world-units de la map.
_MIN_DISPLACEMENT: float = 5.0

#: Marge de sécurité (ms) appliquée si l'estimate dépasse le premier kill connu.
_API_CAP_BUFFER_MS: float = 5_000.0

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

    Filtre permissif avec exclusion des frames game-state (b5=0x00).
    Les frames b5=0x00, base=0x0A sont des frames d'état de jeu (timer, score
    objectif) répétés avec b9 variable — ils créent de faux sig-changes en
    lobby sur certains modes (Fortress, etc.) et ne codent pas une position.
    Tous les autres formats (b5=0x40, b5=0x80...) sont acceptés.

    Returns:
        player_index (0-7) ou None si frame invalide.
    """
    if pos + _MIN_FRAME_LEN > len(data):
        return None
    if data[pos + 5] == 0x00:  # frames game-state : pas de position physique
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



def _displacement(prev: tuple[int, int], cur: tuple[int, int]) -> float:
    """Distance euclidienne entre deux positions (Y 16-bit, X 12-bit).

    Corrige le wraparound du champ X 12-bit (range 0-4095).
    """
    dy = cur[0] - prev[0]
    dx = cur[1] - prev[1]
    if dx > 2048:
        dx -= 4096
    elif dx < -2048:
        dx += 4096
    return (dy * dy + dx * dx) ** 0.5


def _collect_displacement_events(
    chunks: dict[int, tuple[bytes, int, int]],
    min_displacement: float,
) -> list[tuple[float, int]]:
    """Retourne la liste (timestamp_ms, player_index) des frames où un joueur
    s'est déplacé de plus de ``min_displacement`` unités depuis son frame précédent.
    Limites décodables : frames format human (b5=0x40, b9=0x56) uniquement.
    """
    events: list[tuple[float, int]] = []
    last_pos: dict[int, tuple[int, int]] = {}

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
            coords = _decode_coords(data, idx)
            if coords is not None:
                b6 = data[idx + 6]
                if (b6 & 0x1F) in _VALID_BASE_TYPES:
                    pi = b6 >> 5
                    if pi in last_pos and _displacement(last_pos[pi], coords) >= min_displacement:
                        events.append((ts_fn(idx), pi))
                    last_pos[pi] = coords
            pos = idx + 1

    return events


def find_peak_displacement_window(
    chunks: dict[int, tuple[bytes, int, int]],
    window_ms: float = _SPAWN_CLUSTER_WINDOW_MS,
    min_displacement: float = _MIN_DISPLACEMENT,
) -> float | None:
    """Variante de find_peak_activity_window basée sur le déplacement vectoriel réel.

    Decode les coordonnées (Y 16-bit, X 12-bit) des frames format human
    (b5=0x40, b9=0x56). N'émet un événement de mouvement que si le déplacement
    entre deux frames consécutifs d'un même joueur dépasse ``min_displacement``.

    Lobby : petits déplacements ponctuels → peu de joueurs simultanément actifs.
    Spawn réel : tous les joueurs courent → grands deltas simultanés → pic maximal.

    En cas d'égalité de densité : fenêtre la plus tardive préférée (lobby < spawn).

    Returns:
        Timestamp (ms) du début de la fenêtre la plus dense, ou None si données
        insuffisantes (maps sans frames human décodables → fallback signature).
    """
    events = _collect_displacement_events(chunks, min_displacement)

    if not events:
        return None

    events.sort()
    # Nombre minimum de joueurs distincts pour valider le pic → évite les
    # faux pics à 1-2 joueurs sur des maps avec peu de frames human.
    _MIN_PEAK = 2
    best_ts: float = events[0][0]
    best_count: int = 0

    for t_start, _ in events:
        t_end = t_start + window_ms
        unique = {pi for t, pi in events if t_start <= t <= t_end}
        if len(unique) > best_count or (len(unique) == best_count and t_start > best_ts):
            best_count = len(unique)
            best_ts = t_start

    if best_count < _MIN_PEAK:
        return None

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

    Algorithme :
    1. ``find_peak_activity_window`` scanne TOUS les changements de signature
       (toutes les maps, tous les formats de frames) et retient la fenêtre
       [t, t+2s] où le plus de joueurs distincts sont actifs simultanément.
       En lobby, les mouvements de caméra/staging sont éparpillés → pic partiel.
       Au spawn, tous les joueurs bougent en même temps → pic maximal.
    2. Contrainte dure API : si l'estimate dépasse le premier kill/death connu
       (impossible physiquement), on recule à ``api_first_event_ms - 5s``.
       Un grand gap estimate→premier_kill est NORMAL sur les grandes maps
       (travel time 30-45s avant le premier contact) → pas de second scan.

    Note : ``find_peak_displacement_window`` (déplacement vectoriel réel)
    est disponible dans ce module pour usage futur. Elle nécessite que les
    frames format human (b5=0x40) couvrent le début de match AND que les
    mouvements lobby soient distinguables via leur amplitude (non garanti en
    staging Halo, où les joueurs se déplacent à vitesse normale dans une zone
    restreinte → per-frame delta similaire au sprint en match réel).

    Args:
        chunks: {chunk_index: (data, start_ms, dur_ms)}.
        min_players: Inutilisé (conservé pour compatibilité API).
        api_first_event_ms: Timestamp du premier kill/death (mm référentiel
            que le film). Contrainte dure uniquement (cap).

    Returns:
        Timestamp en ms (entier) ou None si aucun changement détecté.
    """
    # Étape 1 : pic d'activité simultanée (toutes maps, tous formats de frames).
    peak_ts = find_peak_activity_window(chunks)
    if peak_ts is None:
        return None
    estimate = int(peak_ts)

    # Étape 2 : contrainte dure API — le premier kill/death ne peut pas
    # précéder le début du match. Si l'estimate la dépasse (rare), on recule.
    # On ne déclenche PAS de second scan : un grand gap estimate→premier_kill
    # est NORMAL sur les grandes maps (travel time 30-45s).
    if api_first_event_ms is not None and estimate > api_first_event_ms:
        estimate = max(0, int(api_first_event_ms) - int(_API_CAP_BUFFER_MS))

    return estimate
