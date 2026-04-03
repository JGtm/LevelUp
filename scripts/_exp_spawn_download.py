"""Script expérimental : téléchargement chunk_01 + scan des premiers mouvements.

Pour chaque match de la session 130, télécharge le(s) premier(s) chunk(s)
REPLICATION_DATA (chunk_01 = 0-20s, chunk_02 = 20-40s) et identifie le
premier position frame de chaque joueur (proxy du spawn initial).

Usage:
    python scripts/_exp_spawn_download.py
    python scripts/_exp_spawn_download.py --chunks 1        # 0-20s seulement
    python scripts/_exp_spawn_download.py --chunks 1 2      # 0-40s complet
    python scripts/_exp_spawn_download.py --dry-run         # vérifier sans DL
    python scripts/_exp_spawn_download.py --match-id f6315f2a-...
"""

from __future__ import annotations

import argparse
import asyncio
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from src.analysis.packet_index import build_packet_estimator, index_chunk
from src.data.services._film_manifest_cache import load_manifest_cache
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


def _decode_position_frame(data: bytes, pos: int) -> tuple[int, int, str] | None:
    """Décode un position frame humain à l'offset donné.

    Returns:
        (player_index, y_raw, x_raw, stream) ou None si frame invalide.
    """
    if pos + MIN_FRAME_LEN > len(data):
        return None

    b5 = data[pos + 5]
    b6 = data[pos + 6]
    b9 = data[pos + 9]

    if b5 != BYTE5_POSITION:
        return None
    if b9 not in (BYTE9_HUMAN, BYTE9_BOT):
        return None

    base_type = b6 & 0x1F
    if base_type not in (0x08, 0x09, 0x28, 0x29):
        return None

    player_index = b6 >> 5
    stream = "human" if b9 == BYTE9_HUMAN else "bot"

    d0, d1, d2, d3 = data[pos + 10], data[pos + 11], data[pos + 12], data[pos + 13]

    if stream == "human":
        if (d0 >> 4) != 4:
            return None
        y_raw = d0 * 256 + d1
        x_raw = (d2 & 0x0F) * 256 + d3
    else:
        if (d1 >> 4) != 0:
            return None
        y_raw = d1 * 256 + d2
        x_raw = (d3 & 0x0F) * 256 + (data[pos + 14] if pos + 14 < len(data) else 0)

    return player_index, y_raw, x_raw, stream


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
) -> dict[int, tuple[bytes, int, int]]:
    """Télécharge les chunks demandés pour ce match si absents du cache.

    Returns:
        {chunk_index: (data, start_ms, duration_ms)}
    """
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
) -> dict[int, dict]:
    """Scanne les chunks pour trouver le premier position frame de chaque joueur.

    Seuls les streams "human" sont retenus.

    Returns:
        {player_index: {"timestamp_ms": float, "chunk": int, "y_raw": int, "x_raw": int}}
    """
    first_seen: dict[int, dict] = {}

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
            decoded = _decode_position_frame(data, idx)
            if decoded is not None:
                pi, y_raw, x_raw, stream = decoded
                if stream == "human" and pi not in first_seen:
                    first_seen[pi] = {
                        "timestamp_ms": ts_fn(idx),
                        "chunk": chunk_idx,
                        "y_raw": y_raw,
                        "x_raw": x_raw,
                    }
            pos = idx + 1

    return first_seen


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

    print(f"\n  {'PI':>3}  {'1er mouvement':>14}  {'Retard':>8}  {'Y_raw':>6}  {'X_raw':>6}  Statut")
    print(f"  {'-'*3}  {'-'*14}  {'-'*8}  {'-'*6}  {'-'*6}  ------")

    for pi, fm in refs:
        ts = _fmt_ms(fm["timestamp_ms"])
        delay = f"+{fm['delay_ms']:.0f}ms" if fm['delay_ms'] > 0 else "REF"
        print(f"  {pi:>3}  {ts:>14}  {delay:>8}  {fm['y_raw']:>6}  {fm['x_raw']:>6}  OK  (chunk_{fm['chunk']:02d})")

    for pi, fm in afk:
        ts = _fmt_ms(fm["timestamp_ms"])
        delay = f"+{fm['delay_ms']/1000:.1f}s"
        print(f"  {pi:>3}  {ts:>14}  {delay:>8}  {fm['y_raw']:>6}  {fm['x_raw']:>6}  AFK? (chunk_{fm['chunk']:02d})")

    if refs:
        ref_ts = refs[0][1]["timestamp_ms"]
        print(f"\n  Reference spawn : {_fmt_ms(ref_ts)}  ({len(refs)} joueur(s) confirmes, {len(afk)} suspect(s) AFK)")
    else:
        print("\n  Aucune reference de spawn trouvee.")


# ── Orchestrateur ──────────────────────────────────────────────────────────────


async def process_match(
    match_id: str,
    api,
    chunk_indices: set[int],
    dry_run: bool,
) -> None:
    """Télécharge chunk(s) si nécessaire, puis scanne les premiers mouvements."""
    print(f"\n{'-'*60}")
    print(f"MATCH {match_id}")
    print(f"{'-'*60}")

    chunks = await download_spawn_chunks(match_id, api, chunk_indices, dry_run)

    if not chunks:
        print("  Pas de données disponibles.")
        return

    first_movements = scan_first_movements(chunks)
    if not first_movements:
        print("  Aucun position frame humain trouve dans les chunks telecharges.")
    else:
        refs, afk = pick_spawn_references(first_movements, n=3)
        print(f"\n  {len(first_movements)} joueur(s) detectes — {len(refs)} refs, {len(afk)} suspect(s) AFK")
        _print_match_report(match_id, first_movements)


async def main(args: argparse.Namespace) -> None:
    match_ids = [args.match_id] if args.match_id else SESSION_MATCH_IDS
    chunk_indices = set(args.chunks)

    print(f"Chunks cibles  : {sorted(chunk_indices)} [0ms, {(chunk_indices.__len__()*20)}s)")
    print(f"Matchs         : {len(match_ids)}")
    if args.dry_run:
        print("Mode           : DRY-RUN (aucun téléchargement)")

    async with create_api_client() as api:
        for mid in match_ids:
            await process_match(mid, api, chunk_indices, args.dry_run)

    print("\n\nTerminé.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Télécharger chunk_01/02 et scanner les spawns initiaux (session 130)"
    )
    parser.add_argument(
        "--match-id",
        metavar="UUID",
        help="Limiter à un seul match",
    )
    parser.add_argument(
        "--chunks",
        nargs="+",
        type=int,
        default=[1, 2],
        metavar="N",
        help="Indices de chunks à télécharger (défaut: 1 2 = 0-40s)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Vérifier sans télécharger (liste ce qui manque)",
    )
    asyncio.run(main(parser.parse_args()))
