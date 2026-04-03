"""Script expérimental : détection adaptative du début de match via filmshell.

Pour chaque match de la session 130, télécharge les chunks REPLICATION_DATA
un par un jusqu'à identifier le premier mouvement réel de min_players joueurs
(= fin du countdown = début effectif du match).

Robustesse :
- Matchs à chargement rapide (<5s) : s'arrête à chunk_02 (20-40s couverts).
- Matchs à chargement lent (<60s)  : descend jusqu'à chunk_04 ou chunk_05.
- chunk_01 sert toujours de baseline spawn (positions figées du countdown).
- Le timestamp de début de match est calculé dynamiquement — jamais hardcodé.

Usage:
    python scripts/_exp_spawn_download.py
    python scripts/_exp_spawn_download.py --min-players 2   # moins strict
    python scripts/_exp_spawn_download.py --max-chunks 8    # matchs très lents
    python scripts/_exp_spawn_download.py --dry-run         # cache existant seulement
    python scripts/_exp_spawn_download.py --match-id f6315f2a-...
"""

from __future__ import annotations

import argparse
import asyncio
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

import statistics

import duckdb

from src.analysis.packet_index import build_packet_estimator, index_chunk
from src.data.services._film_manifest_cache import load_manifest_cache, write_manifest_cache
from src.data.sync.api_factory import create_api_client

# ── Constantes filmshell ───────────────────────────────────────────────────────

FRAME_MARKER = bytes([0xA0, 0x7B, 0x42])
BYTE5_POSITION = 0x40
BYTE9_HUMAN = 0x56
BYTE9_BOT = 0x35
MIN_FRAME_LEN = 14

CHUNK_DIR = ROOT / "data/cache/film_chunks"
MANIFEST_DIR = ROOT / "data/cache/film_manifests"

# Matchs de la session 130 (2026-03-31)
SESSION_MATCH_IDS = [
    "f6315f2a-e54b-4b89-8274-bc07869d7689",
    "e61dbcd9-2ae7-4d9b-9fe9-602bbeccdb4c",
    "04d05635-a0c6-4121-93bd-cf8e6afefaa3",
    "8faf5c41-0af2-4102-b687-60b297afc1c7",
    "e37f4519-e909-4826-ae69-fde1e1c605d7",
    "0727867d-ca7a-43dc-9845-bef7093a88e5",
    "0c0fafd8-1405-447d-bdf8-190edc7dbc1f",
    "41b61fb9-3d71-40b7-bde7-45682fba6d57",
    "ade271ff-7a52-4bea-97e6-d0658707e782",
    "70a1c6c6-6cf6-48dd-9b34-39df2b111a2a",
    "c139818f-9378-4b24-8d9a-43f767ca8656",
]


# ── Décodage position frame ────────────────────────────────────────────────────

# Formats de base_type valides pour un frame de position
# (b6 & 0x1F) — communs à tous les streams observés
_VALID_BASE_TYPES = frozenset({0x08, 0x09, 0x0A, 0x0B, 0x28, 0x29})


def _is_position_frame(data: bytes, pos: int) -> int | None:
    """Retourne player_index si le marker est un frame de position valide.

    Filtre permissif : seul base_type (b6 & 0x1F) est vérifié.
    Ni b5 (format stream), ni b9 (stream selector), ni d0h ne sont requis —
    ce qui permet de capturer tous les joueurs quel que soit le format réseau
    (human/b9=0x56, enemy/b9=0x11, distant/b5=0x80, b5=0x00, etc.).

    Returns:
        player_index (0-7) ou None si frame invalide.
    """
    if pos + MIN_FRAME_LEN > len(data):
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
    if pos + MIN_FRAME_LEN > len(data):
        return None
    if data[pos + 5] != BYTE5_POSITION or data[pos + 9] != BYTE9_HUMAN:
        return None
    d0, d1, d2, d3 = data[pos + 10], data[pos + 11], data[pos + 12], data[pos + 13]
    if (d0 >> 4) != 4:
        return None
    return d0 * 256 + d1, (d2 & 0x0F) * 256 + d3


def _fmt_ms(ms: float) -> str:
    """Formate ms en mm:ss.mmm."""
    s = ms / 1000
    m = int(s) // 60
    return f"{m:02d}:{s - m * 60:06.3f}"


# ── Téléchargement ─────────────────────────────────────────────────────────────


async def download_spawn_chunks(
    match_id: str,
    api,
    chunk_indices: set[int],
    dry_run: bool = False,
    cached_only: bool = False,
) -> dict[int, tuple[bytes, int, int]]:
    """Télécharge les chunks demandés pour ce match si absents du cache.

    Args:
        cached_only: Si True, ne fait aucun appel API (ni manifest ni blob).
                     Retourne {} si le manifest n'est pas en cache local.

    Returns:
        {chunk_index: (data, start_ms, duration_ms)}
    """
    cached = load_manifest_cache(MANIFEST_DIR, match_id)
    if cached is None:
        if cached_only:
            return {}  # manifest absent et mode cache-only : skip silencieux
        # Manifest absent du cache : appel API + sauvegarde
        film = await api.get_film_by_match_id(match_id)
        if film is None:
            print(f"  [{match_id[:8]}] SKIP : film introuvable via API")
            return {}
        MANIFEST_DIR.mkdir(parents=True, exist_ok=True)
        write_manifest_cache(
            MANIFEST_DIR,
            match_id,
            film.blob_storage_path_prefix,
            film.custom_data.chunks,
        )
        cached = load_manifest_cache(MANIFEST_DIR, match_id)
    if cached is None:
        print(f"  [{match_id[:8]}] SKIP : manifest non trouvé")
        return {}

    blob_prefix, chunks_meta = cached
    match_cache = CHUNK_DIR / match_id[:8]
    match_cache.mkdir(parents=True, exist_ok=True)

    result: dict[int, tuple[bytes, int, int]] = {}

    for ch in sorted(chunks_meta, key=lambda c: c.index):
        if ch.chunk_type.value != 2:  # REPLICATION_DATA uniquement
            continue
        if ch.index not in chunk_indices:
            continue

        cache_path = match_cache / f"chunk_{ch.index:02d}.bin"
        start_ms = ch.chunk_start_time_offset_milliseconds
        dur_ms = ch.duration_milliseconds

        if cache_path.exists():
            data = cache_path.read_bytes()
            print(f"  [{match_id[:8]}] CACHE chunk_{ch.index:02d}.bin  ({len(data)//1024}KB)")
            result[ch.index] = (data, start_ms, dur_ms)
            continue

        url = blob_prefix + ch.file_relative_path.lstrip("/")
        size_hint = f"start={start_ms}ms dur={dur_ms}ms"
        print(f"  [{match_id[:8]}] DL    chunk_{ch.index:02d}  {size_hint}")

        if dry_run:
            print(f"             -> DRY-RUN, skip")
            continue

        data = await api.download_film_chunk(url)
        if data is not None:
            cache_path.write_bytes(data)
            print(f"             -> {len(data)//1024}KB ok")
            result[ch.index] = (data, start_ms, dur_ms)
        else:
            print(f"             -> ECHEC")

    return result


# ── Scan premier mouvement ─────────────────────────────────────────────────────


def scan_first_movements(
    chunks: dict[int, tuple[bytes, int, int]],
    ignore_before_ms: float = 0.0,
) -> dict[int, dict]:
    """Détecte le premier CHANGEMENT de position pour chaque player_index.

    Algorithme :
    - Le premier frame vu par pi = baseline de spawn (position figée pendant le countdown).
    - Le premier frame DIFFÉRENT de cette baseline = premier mouvement réel.
    - Les chunks sont traités en ordre croissant : chunk_01 fournit la baseline,
      chunk_02+ capturent le premier mouvement.
    - ``ignore_before_ms`` : ignorer les changements détectés avant ce timestamp,
      pour éliminer les mouvements de lobby (rotation caméra pre-match).
    - Accepte tous les formats stream (b5 quelconque, b9 quelconque).

    Returns:
        {player_index: {"timestamp_ms", "chunk", "b5", "b9", "y_raw", "x_raw"}}
        y_raw/x_raw sont None si le format n'est pas h/b5=0x40/b9=0x56.
    """
    spawn_sig: dict[int, bytes] = {}   # pi -> bytes signature du frame de spawn
    first_change: dict[int, dict] = {}

    for chunk_idx in sorted(chunks):
        data, start_ms, _ = chunks[chunk_idx]
        try:
            packets = index_chunk(data)
            ts_fn = build_packet_estimator(packets, float(start_ms))
        except Exception:
            ts_fn = lambda pos: float(start_ms)  # noqa: E731

        pos = 0
        while True:
            idx = data.find(FRAME_MARKER, pos)
            if idx == -1:
                break
            pi = _is_position_frame(data, idx)
            if pi is not None:
                sig = bytes(data[idx + 9 : idx + 16])
                if pi not in spawn_sig:
                    spawn_sig[pi] = sig  # premier frame = baseline de spawn
                elif sig != spawn_sig[pi] and pi not in first_change:
                    ts = ts_fn(idx)
                    if ts >= ignore_before_ms:
                        coords = _decode_coords(data, idx)
                        first_change[pi] = {
                            "timestamp_ms": ts,
                            "chunk": chunk_idx,
                            "b5": data[idx + 5],
                            "b9": data[idx + 9],
                            "y_raw": coords[0] if coords else None,
                            "x_raw": coords[1] if coords else None,
                        }
                    spawn_sig[pi] = sig  # update baseline
            pos = idx + 1

    return first_change


# ── Références de spawn ───────────────────────────────────────────────────────

# Au-delà de ce seuil par rapport à la référence la plus précoce,
# un joueur est considéré comme suspect AFK (>10s de retard)
_AFK_THRESHOLD_MS = 10_000.0


def pick_spawn_references(
    first_movements: dict[int, dict],
    n: int = 3,
) -> tuple[list[tuple[int, dict]], list[tuple[int, dict]]]:
    """Trie les premiers mouvements et retourne (references, afk_suspects).

    Args:
        first_movements: Résultat de scan_first_movements().
        n: Nombre de références à garder (défaut: 3).

    Returns:
        Tuple (references, afk_suspects) — chaque liste est [(player_index, fm_dict)].
        ``references`` contient les n joueurs les plus précoces non-AFK.
        ``afk_suspects`` contient les joueurs avec un retard > _AFK_THRESHOLD_MS
        par rapport au premier référent.
    """
    if not first_movements:
        return [], []

    sorted_by_ts = sorted(first_movements.items(), key=lambda kv: kv[1]["timestamp_ms"])
    earliest_ms = sorted_by_ts[0][1]["timestamp_ms"]

    references: list[tuple[int, dict]] = []
    afk_suspects: list[tuple[int, dict]] = []

    for pi, fm in sorted_by_ts:
        delay = fm["timestamp_ms"] - earliest_ms
        fm["delay_ms"] = delay
        if delay > _AFK_THRESHOLD_MS:
            afk_suspects.append((pi, fm))
        elif len(references) < n:
            references.append((pi, fm))
        else:
            afk_suspects.append((pi, fm))  # au-delà des n références voulues

    return references, afk_suspects


# ── Rapport ────────────────────────────────────────────────────────────────────


def _print_match_report(match_id: str, first_movements: dict[int, dict]) -> None:
    refs, afk = pick_spawn_references(first_movements, n=3)

    def _coord(v: int | None) -> str:
        return f"{v:>6}" if v is not None else "     ?"

    print(f"\n  {'PI':>3}  {'1er mouvement':>14}  {'Retard':>8}  {'Y_raw':>6}  {'X_raw':>6}  b5  b9   Statut")
    print(f"  {'-'*3}  {'-'*14}  {'-'*8}  {'-'*6}  {'-'*6}  --  --   ------")

    for pi, fm in refs:
        ts = _fmt_ms(fm["timestamp_ms"])
        delay = f"+{fm['delay_ms']:.0f}ms" if fm['delay_ms'] > 0 else "REF"
        print(f"  {pi:>3}  {ts:>14}  {delay:>8}  {_coord(fm['y_raw'])}  {_coord(fm['x_raw'])}  {fm['b5']:02X}  {fm['b9']:02X}   OK  (chunk_{fm['chunk']:02d})")

    for pi, fm in afk:
        ts = _fmt_ms(fm["timestamp_ms"])
        delay = f"+{fm['delay_ms']/1000:.1f}s"
        print(f"  {pi:>3}  {ts:>14}  {delay:>8}  {_coord(fm['y_raw'])}  {_coord(fm['x_raw'])}  {fm['b5']:02X}  {fm['b9']:02X}   AFK? (chunk_{fm['chunk']:02d})")

    if refs:
        ref_ts = refs[0][1]["timestamp_ms"]
        print(f"\n  Reference spawn : {_fmt_ms(ref_ts)}  ({len(refs)} joueur(s) confirmes, {len(afk)} suspect(s) AFK)")
    else:
        print("\n  Aucune reference de spawn trouvee.")


# ── Corrélation API highlight_events ─────────────────────────────────────────

# DB par défaut — shared_matches_v2
_SHARED_DB_PATH = ROOT / "data/warehouse/shared_matches_v2.duckdb"

# Écart maximum toléré entre l'estimation filmshell et le premier event API
# avant de signaler un verdict "suspect" (en ms).
_CORRELATION_WARN_GAP_MS = 60_000  # 60s


def correlate_with_api(
    filmshell_estimate_ms: float,
    match_id: str,
    n_events: int = 30,
    db_path: Path = _SHARED_DB_PATH,
) -> dict:
    """Corrèle l'estimation filmshell du début de match avec N highlight_events.

    Au lieu de se fier à un seul event (ex: premier death d'un joueur),
    utilise la distribution des N premiers kill/death de tous les joueurs.

    Légende des référentiels :
    - ``highlight_events.time_ms`` utilise le même référentiel que le film
      (t=0 = début de l'enregistrement, countdown inclus).
    - ``filmshell_estimate_ms`` est le timestamp détecté dans le film au moment
      où les joueurs commencent à bouger (= fin du countdown ≈ match start).
    - Les gaps = time_ms[event] - filmshell_estimate devraient donc tous être
      positifs (les events surviennent après le début du match) et raisonnables
      (<durée du match).

    Args:
        filmshell_estimate_ms: Timestamp filmshell du début de match (median des refs).
        match_id: UUID du match.
        n_events: Nombre de kill/death events à récupérer (défaut: 30).
        db_path: Chemin vers shared_matches*.duckdb.

    Returns:
        dict avec clés :
          events    : list[int] des time_ms des N premiers events
          gaps      : list[int] des (time_ms - filmshell_estimate)
          gap_min   : int  (ms) — gap du premier event
          gap_max   : int  (ms) — gap du dernier event parmi les N
          gap_median: float (ms)
          n_negative: int  — nb d'events AVANT l'estimation (anomalies)
          verdict   : "ok" | "suspect" | "no_data" | "unavailable"
    """
    if not db_path.exists():
        return {"verdict": "unavailable", "events": [], "gaps": []}

    try:
        with duckdb.connect(str(db_path), read_only=True) as con:
            rows = con.execute(
                """
                SELECT time_ms
                FROM highlight_events
                WHERE match_id = ?
                  AND event_type IN ('kill', 'death')
                ORDER BY time_ms
                LIMIT ?
                """,
                [match_id, n_events],
            ).fetchall()
    except Exception as exc:  # noqa: BLE001
        return {"verdict": "unavailable", "events": [], "gaps": [], "error": str(exc)}

    if not rows:
        return {"verdict": "no_data", "events": [], "gaps": []}

    events = [r[0] for r in rows]
    gaps = [t - filmshell_estimate_ms for t in events]
    n_negative = sum(1 for g in gaps if g < 0)
    gap_min = int(min(gaps))
    gap_max = int(max(gaps))
    gap_median = statistics.median(gaps)

    # Verdict :
    # - "ok" : tous les events après l'estimation ET premier event plausible (<60s après)
    # - "suspect" : n_negative > 0 (event avant estimate) OU gap_min > seuil
    if n_negative > 0 or gap_min > _CORRELATION_WARN_GAP_MS:
        verdict = "suspect"
    else:
        verdict = "ok"

    return {
        "events": events,
        "gaps": gaps,
        "gap_min": gap_min,
        "gap_max": gap_max,
        "gap_median": gap_median,
        "n_negative": n_negative,
        "n_events": len(events),
        "verdict": verdict,
    }


def _print_correlation_report(corr: dict, estimate_ms: float) -> None:
    """Affiche le rapport de corrélation API."""
    verdict = corr["verdict"]

    if verdict == "unavailable":
        print(f"  [CORR] DB non disponible ({corr.get('error', '?')})")
        return
    if verdict == "no_data":
        print("  [CORR] Aucun kill/death trouvé dans highlight_events pour ce match.")
        return

    n = corr["n_events"]
    gap_min = corr["gap_min"]
    gap_median = corr["gap_median"]
    gap_max = corr["gap_max"]
    n_neg = corr["n_negative"]
    sign = "OK" if verdict == "ok" else "SUSPECT"

    print(
        f"  [CORR {sign}] {n} events — "
        f"gap min={gap_min/1000:+.2f}s  med={gap_median/1000:+.2f}s  max={gap_max/1000:+.2f}s"
        + (f"  ({n_neg} event(s) AVANT l'estimation!)" if n_neg else "")
    )
    if verdict == "suspect" and gap_min > _CORRELATION_WARN_GAP_MS:
        print(
            f"  [CORR] Premier event a +{gap_min/1000:.0f}s apres l'estimate "
            f"({estimate_ms/1000:.1f}s) — le match a peut-etre demarre plus tot."
        )


def write_film_start_to_db(
    match_id: str,
    film_match_start_ms: int,
    db_path: Path = _SHARED_DB_PATH,
) -> None:
    """Ecrit film_match_start_ms dans match_registry pour ce match.

    Idempotente : un deuxième appel sur le même match_id écrase la valeur.
    """
    with duckdb.connect(str(db_path)) as con:
        con.execute(
            "UPDATE match_registry SET film_match_start_ms = ? WHERE match_id = ?",
            [film_match_start_ms, match_id],
        )
        updated = con.execute(
            "SELECT film_match_start_ms FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()
    if updated and updated[0] == film_match_start_ms:
        print(f"  [DB] film_match_start_ms = {film_match_start_ms}ms ecrit.")
    else:
        print(f"  [DB] ERREUR: match_id introuvable dans match_registry ({match_id[:8]})")


# ── Téléchargement adaptatif ─────────────────────────────────────────────────


async def process_match_adaptive(
    match_id: str,
    api,
    *,
    min_players: int = 3,
    max_chunks: int = 6,
    corr_events: int = 30,
    write_db: bool = False,
    cached_only: bool = False,
    dry_run: bool = False,
) -> None:
    """Télécharger chunk par chunk jusqu'à ce que min_players soient détectés.

    Robustesse aux matchs à chargement long (<60s) ou court (<10s) :
    - chunk_01 sert toujours de baseline de spawn (positions figées du countdown)
    - chunk_02, 03… sont ajoutés un à un jusqu'à ce que min_players premiers
      mouvements soient détectés, ou jusqu'à max_chunks (limite de sécurité).
    - La valeur "début de match" (timestamp_ms du premier mouvement détecté)
      est calculée dynamiquement — jamais hardcodée.
    - La corrélation finale avec highlight_events utilise corr_events kill/death
      (pas un seul joueur/event) pour une validation robuste.

    Args:
        min_players:  Nombre de joueurs non-AFK à détecter avant d'arrêter (défaut: 3).
        max_chunks:   Limite de chunks à télécharger, quel que soit le résultat (défaut: 6 = 0-120s).
        corr_events:  Nombre de kill/death events API utilisés pour la corrélation (défaut: 30).
        write_db:     Ecrire film_match_start_ms dans match_registry si la corrélation est ok.
        cached_only:  Si True, ne télécharge que depuis le cache local (ni manifest ni blob API).
        dry_run:      Ne pas télécharger, juste vérifier le cache.
    """
    print(f"\n{'-'*60}")
    print(f"MATCH {match_id}")
    print(f"{'-'*60}")

    all_chunks: dict[int, tuple[bytes, int, int]] = {}
    next_chunk_idx = 1  # toujours commencer par chunk_01 (baseline)

    try:
      while next_chunk_idx <= max_chunks:
        new = await download_spawn_chunks(
            match_id, api, {next_chunk_idx}, dry_run, cached_only=cached_only
        )

        # Si chunk_01 est introuvable et qu'on n'est pas en mode cached-only,
        # le film n'existe pas pour ce match (404 API) → inutile d'essayer les
        # chunks suivants, on sortirait avec le même 404 à chaque itération.
        if not new and next_chunk_idx == 1 and not cached_only and not dry_run:
            print(f"  [{match_id[:8]}] Pas de film disponible, skip.")
            return

        if new:
            all_chunks.update(new)

        first_movements = scan_first_movements(all_chunks)
        refs, afk = pick_spawn_references(first_movements, n=min_players)

        # Condition d'arrêt : assez de références ou plafond atteint
        enough = len(refs) >= min_players
        at_limit = next_chunk_idx >= max_chunks

        if not first_movements and at_limit:
            print("  Aucun position frame humain trouve apres", max_chunks, "chunks.")
            return

        if enough or at_limit:
            reason = f"{len(refs)} refs ok" if enough else f"limite {max_chunks} chunks atteinte"
            chunks_span_s = next_chunk_idx * 20  # 20s par chunk REPLICATION_DATA
            print(
                f"  Stop a chunk_{next_chunk_idx:02d}"
                f" ([0-{chunks_span_s}s]) — {reason}"
                f" — {len(first_movements)} joueur(s) total"
            )
            break

        # Pas assez → ajouter le chunk suivant
        print(
            f"  chunk_{next_chunk_idx:02d} traite — {len(refs)}/{min_players} refs,"
            f" telechargement chunk_{next_chunk_idx + 1:02d}..."
        )
        next_chunk_idx += 1

    except Exception as exc:  # noqa: BLE001
        print(f"  [{match_id[:8]}] ERREUR inattendue: {exc}")
        return

    if first_movements:
        _print_match_report(match_id, first_movements)
        # Estimation consensus : médiane des timestamps des refs filmshell
        refs, _ = pick_spawn_references(first_movements, n=min_players)
        if refs:
            estimate_ms = statistics.median(fm["timestamp_ms"] for _, fm in refs)
            corr = correlate_with_api(estimate_ms, match_id, n_events=corr_events)
            _print_correlation_report(corr, estimate_ms)

            # Second passage : correction des mouvements de lobby
            # Si le premier event API est >15s après notre estimation → l'estimation
            # est trop précoce (mouvement de caméra pendant le countdown).
            # On re-scanne en ignorant les mouvements avant (gap_min - 10s).
            if (
                corr.get("verdict") != "unavailable"
                and corr.get("n_negative", 1) == 0
                and corr.get("gap_min", 0) > 15_000
            ):
                ignore_before_ms = estimate_ms + corr["gap_min"] - 10_000
                print(
                    f"  [CORR] Lobby détecté — re-scan depuis {ignore_before_ms / 1000:.1f}s"
                    f" (gap_min={corr['gap_min'] / 1000:.1f}s)"
                )
                # Charger les chunks manquants jusqu'à couvrir ignore_before_ms + 20s
                needed_coverage_ms = ignore_before_ms + 20_000
                needed_chunk = int(needed_coverage_ms // 20_000) + 1
                needed_chunk = min(needed_chunk, max_chunks)
                for ci in range(next_chunk_idx + 1, needed_chunk + 1):
                    if ci not in all_chunks:
                        extra = await download_spawn_chunks(
                            match_id, api, {ci}, dry_run, cached_only=cached_only
                        )
                        if extra:
                            all_chunks.update(extra)

                fm2 = scan_first_movements(all_chunks, ignore_before_ms=ignore_before_ms)
                refs2, _ = pick_spawn_references(fm2, n=min_players)
                if len(refs2) >= min_players:
                    estimate2 = int(statistics.median(fm["timestamp_ms"] for _, fm in refs2))
                    # Guard : accepter seulement si l'estimate est bien après ignore_before.
                    # Faux positif : joueurs déjà actifs → détection quasi-immédiate.
                    if estimate2 - ignore_before_ms > 5_000:
                        estimate_ms = estimate2
                        _print_match_report(match_id, fm2)
                        corr = correlate_with_api(estimate_ms, match_id, n_events=corr_events)
                        _print_correlation_report(corr, estimate_ms)
                        print(f"  [CORR] Estimation corrigee -> {estimate_ms / 1000:.1f}s")
                    else:
                        print(
                            f"  [CORR] Correction rejetee (faux positif) :"
                            f" estimate2={estimate2 / 1000:.1f}s trop proche ignore_before"
                            f" ({ignore_before_ms / 1000:.1f}s) — match sans mouvement lobby"
                        )
                else:
                    print(
                        f"  [CORR] Correction échouée : {len(refs2)}/{min_players} refs"
                        f" — chunks supplémentaires nécessaires (essayer --max-chunks plus élevé)"
                    )

            if write_db and not dry_run:
                write_film_start_to_db(match_id, int(estimate_ms))
    else:
        print("  Pas de donnees disponibles.")


async def main(args: argparse.Namespace) -> None:
    if args.all_matches:
        with duckdb.connect(str(_SHARED_DB_PATH), read_only=True) as con:
            q = "SELECT match_id FROM match_registry"
            if args.skip_done:
                q += " WHERE film_match_start_ms IS NULL"
            q += " ORDER BY start_time"
            if args.limit:
                q += f" LIMIT {args.limit}"
            match_ids = [r[0] for r in con.execute(q).fetchall()]
        if args.cached_only:
            available = {p.stem for p in MANIFEST_DIR.glob("*.json")}
            before = len(match_ids)
            match_ids = [m for m in match_ids if m[:8] in available]
            print(f"[cached-only] {before} -> {len(match_ids)} matchs (manifest en cache)")
    else:
        match_ids = [args.match_id] if args.match_id else SESSION_MATCH_IDS

    print(f"Matchs         : {len(match_ids)}")
    print(f"Min joueurs    : {args.min_players}  (refs non-AFK avant arret)")
    print(f"Max chunks     : {args.max_chunks}  (=> couverture max {args.max_chunks * 20}s du film)")
    print(f"Corr events    : {args.corr_events}  (kill/death API pour validation)")
    print(f"Write DB       : {args.write_db}")
    if args.dry_run:
        print("Mode           : DRY-RUN (aucun téléchargement)")

    async with create_api_client() as api:
        for mid in match_ids:
            await process_match_adaptive(
                mid,
                api,
                min_players=args.min_players,
                max_chunks=args.max_chunks,
                corr_events=args.corr_events,
                write_db=args.write_db,
                cached_only=args.cached_only,
                dry_run=args.dry_run,
            )

    print("\n\nTerminé.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description=(
            "Téléchargement adaptatif chunk-par-chunk pour détecter le début réel "
            "de chaque match (session 130). S'adapte automatiquement aux matchs "
            "dont le chargement est long ou court."
        )
    )
    parser.add_argument(
        "--cached-only",
        action="store_true",
        help=(
            "(Avec --all-matches) Ne traiter que les matchs dont le manifest est"
            " deja en cache local (evite les appels API, les 404 sur vieux matchs)."
        ),
    )
    parser.add_argument(
        "--all-matches",
        action="store_true",
        help="Traiter tous les matchs de match_registry (au lieu de la liste session 130).",
    )
    parser.add_argument(
        "--skip-done",
        action="store_true",
        default=True,
        help="(Avec --all-matches) Ignorer les matchs qui ont déjà film_match_start_ms (défaut: True).",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        metavar="N",
        help="(Avec --all-matches) Limiter à N matchs par run — utile pour lancer par batches.",
    )
    parser.add_argument(
        "--match-id",
        metavar="UUID",
        help="Limiter à un seul match",
    )
    parser.add_argument(
        "--min-players",
        type=int,
        default=3,
        metavar="N",
        help=(
            "Nombre de joueurs non-AFK à détecter avant d'arrêter le téléchargement "
            "(défaut: 3). Plus la valeur est haute, plus la détection est fiable "
            "mais plus de chunks peuvent être téléchargés."
        ),
    )
    parser.add_argument(
        "--max-chunks",
        type=int,
        default=6,
        metavar="N",
        help=(
            "Limite de chunks à télécharger par match (défaut: 6 = 0-120s). "
            "Augmenter si le chargement du match prend plus de 80-90s."
        ),
    )
    parser.add_argument(
        "--corr-events",
        type=int,
        default=30,
        metavar="N",
        help=(
            "Nombre de kill/death events API utilisés pour valider l'estimation filmshell "
            "(défaut: 30). Un N plus élevé est plus robuste aux matchs où les premiers "
            "frags ou triple-kills arrivent tardivement."
        ),
    )
    parser.add_argument(
        "--write-db",
        action="store_true",
        help=(
            "Ecrire film_match_start_ms dans match_registry pour chaque match traite. "
            "Sans ce flag, le script affiche uniquement les resultats (mode lecture seule)."
        ),
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Vérifier sans télécharger (utilise uniquement le cache existant)",
    )
    asyncio.run(main(parser.parse_args()))
