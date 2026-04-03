"""Script expérimental : scan des position frames filmshell pour détecter spawns/respawns.

Théorie testée :
  - Les chunks REPLICATION_DATA contiennent des frames de position marqués A0 7B 42.
  - byte5=0x40  → position frame
  - byte9=0x56  → stream humain / byte9=0x35 → stream bot
  - byte6 low-5-bits = type (0x09 standard), bits 7-5 = player_index
  - Les coordonnées accumulées (deltas cumulatifs) signalent un respawn quand
    |delta| > WRAPAROUND_THRESHOLD (discontinuité physique).
  - build_packet_estimator() date chaque frame à ±16ms.

Usage :
    python scripts/_exp_spawn_scan.py
    python scripts/_exp_spawn_scan.py --match-id f6315f2a-e54b-4b89-8274-bc07869d7689
    python scripts/_exp_spawn_scan.py --verbose
"""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass, field
from pathlib import Path
from types import SimpleNamespace

# Assurer que src/ est dans le path
ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from src.analysis.packet_index import PacketType, index_chunk, build_packet_estimator

# ── Constantes filmshell (docs/motion-extraction.md) ──────────────────────────

FRAME_MARKER = bytes([0xA0, 0x7B, 0x42])

# Position frames humains
BYTE5_POSITION = 0x40       # prefix type "position frame"
BYTE9_HUMAN = 0x56          # stream humain
BYTE9_BOT = 0x35            # stream bot
BASE_TYPE_POSITION = 0x09   # low 5 bits de byte6 = type standard

# Coordonnées 16-bit (Y) et 12-bit (X)
# Y: wraparound à ±32768, discontinuité > 32000 = respawn
# X: wraparound à ±2048,  discontinuité > 4000 = respawn
Y_WRAP = 65536
Y_HALF = Y_WRAP // 2
X_WRAP = 4096
X_HALF = X_WRAP // 2
# Seuils de respawn calibrés empiriquement :
# Y_raw range ~109 unités, delta normal max ~37 → seuil=60 discrimine bien
# X_raw range ~2000 unités, delta normal max ~100 → seuil=300
RESPAWN_THRESHOLD_Y = 60   # delta brut Y > 60 = discontinuité physique (mort/tp)
RESPAWN_THRESHOLD_X = 300  # delta brut X > 300 = discontinuité physique

# Frame minimal valide : au moins 14 bytes après le marker
MIN_FRAME_LEN = 14

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

# ── Structures ────────────────────────────────────────────────────────────────


@dataclass
class PositionFrame:
    """Résultat du décodage d'un position frame."""
    byte_pos: int
    timestamp_ms: float
    player_index: int
    stream: str          # "human" | "bot"
    y_raw: int
    x_raw: int


@dataclass
class PlayerTrack:
    """Track cumulatif des coordonnées d'un joueur."""
    player_index: int
    stream: str
    acc_y: int = 0
    acc_x: int = 0
    prev_y_raw: int | None = None
    prev_x_raw: int | None = None
    n_frames: int = 0
    respawn_events: list[dict] = field(default_factory=list)
    # Premier frame observé (relatif au début des chunks disponibles)
    first_frame_ms: float | None = None
    first_frame_chunk: int | None = None


@dataclass
class ChunkScanResult:
    """Résultat du scan d'un chunk."""
    chunk_index: int
    start_ms: int
    duration_ms: int
    n_position_frames: int
    n_packets: int
    player_indices_seen: set[int]


@dataclass
class MatchScanResult:
    """Résultat global pour un match."""
    match_id: str
    chunks_scanned: list[ChunkScanResult]
    player_tracks: dict[str, PlayerTrack]      # clé = "pi_{idx}_{stream}"
    total_position_frames: int
    errors: list[str]


# ── Parseur de position frames ────────────────────────────────────────────────


def _decode_position_frame(data: bytes, pos: int) -> PositionFrame | None:
    """Décode un position frame à partir de son offset dans le chunk.

    Layout (doc filmshell motion-extraction.md) :
      [0-2]  A0 7B 42  marker
      [3-4]  inconnu (timestamp interne ? séquence ?)
      [5]    0x40 = prefix type "position frame"
      [6]    type (low 5 bits) + player_index (bits 7-5)
      [7]    stream selector : 0x00 = human, 0x40 = bot
      [8]    subtype (varies)
      [9]    0x56 human / 0x35 bot
      [10]   d0 — Y high (nibble haute = 4 pour human, 0 pour bot)
      [11]   d1 — Y low
      [12]   d2 — X high nibble (low nibble = bits 11-8 de X)
      [13]   d3 — X low byte

    Returns:
        PositionFrame ou None si les conditions ne sont pas remplies.
    """
    if pos + MIN_FRAME_LEN > len(data):
        return None

    b5 = data[pos + 5]
    b6 = data[pos + 6]
    b9 = data[pos + 9]

    # Filtrer sur le type prefix et le marker stream
    if b5 != BYTE5_POSITION:
        return None
    if b9 not in (BYTE9_HUMAN, BYTE9_BOT):
        return None

    base_type = b6 & 0x1F   # low 5 bits
    player_index = b6 >> 5  # top 3 bits

    # Type standard = 0x09, variantes connues : 0x08, 0x29, 0x28
    if base_type not in (0x08, 0x09, 0x28, 0x29):
        return None

    stream = "human" if b9 == BYTE9_HUMAN else "bot"

    d0 = data[pos + 10]
    d1 = data[pos + 11]
    d2 = data[pos + 12]
    d3 = data[pos + 13]

    # Vérification du nibble haute de d0 :
    # human : d0 high nibble doit être 4 (0x40-0x4F)
    # bot   : décalé d'un byte → utiliser pos+11 à la place, high nibble = 0
    if stream == "human" and (d0 >> 4) != 4:
        return None
    if stream == "bot" and (d1 >> 4) != 0:
        return None  # bot offset : bytes aux positions 11-14

    if stream == "human":
        y_raw = d0 * 256 + d1       # 16-bit
        x_raw = (d2 & 0x0F) * 256 + d3  # 12-bit
    else:
        # Bot : décalé d'un byte (pos+11 = d0_bot, pos+12 = d1_bot, ...)
        d0b = data[pos + 11]
        d1b = data[pos + 12]
        d2b = data[pos + 13]
        d3b = data[pos + 14] if pos + 14 < len(data) else 0
        y_raw = d0b * 256 + d1b
        x_raw = (d2b & 0x0F) * 256 + d3b

    return PositionFrame(
        byte_pos=pos,
        timestamp_ms=0.0,  # sera défini par le caller
        player_index=player_index,
        stream=stream,
        y_raw=y_raw,
        x_raw=x_raw,
    )


def _update_track(track: PlayerTrack, frame: PositionFrame) -> bool:
    """Met à jour le track cumulatif, retourne True si respawn détecté."""
    is_respawn = False

    if track.prev_y_raw is not None:
        # Delta Y (16-bit wraparound)
        dy = frame.y_raw - track.prev_y_raw
        if dy > Y_HALF:
            dy -= Y_WRAP
        elif dy < -Y_HALF:
            dy += Y_WRAP

        # Delta X (12-bit wraparound)
        dx = frame.x_raw - track.prev_x_raw  # type: ignore[operator]
        if dx > X_HALF:
            dx -= X_WRAP
        elif dx < -X_HALF:
            dx += X_WRAP

        if abs(dy) > RESPAWN_THRESHOLD_Y or abs(dx) > RESPAWN_THRESHOLD_X:
            # Discontinuité = respawn
            track.respawn_events.append({
                "timestamp_ms": frame.timestamp_ms,
                "delta_y": dy,
                "delta_x": dx,
                "y_raw_before": track.prev_y_raw,
                "x_raw_before": track.prev_x_raw,
                "y_raw_after": frame.y_raw,
                "x_raw_after": frame.x_raw,
            })
            is_respawn = True
            # Réinitialiser acc après discontinuité
            track.acc_y = frame.y_raw
            track.acc_x = frame.x_raw
        else:
            track.acc_y += dy
            track.acc_x += dx
    else:
        # Premier frame pour ce joueur
        track.acc_y = frame.y_raw
        track.acc_x = frame.x_raw

    track.prev_y_raw = frame.y_raw
    track.prev_x_raw = frame.x_raw
    track.n_frames += 1
    return is_respawn


# ── Chargement chunks ─────────────────────────────────────────────────────────


def _load_manifest(manifest_dir: Path, match_id: str) -> list[SimpleNamespace] | None:
    """Charge les métadonnées des chunks depuis le cache manifest."""
    path = manifest_dir / f"{match_id[:8]}.json"
    if not path.exists():
        return None
    data = json.loads(path.read_text(encoding="utf-8"))
    return [
        SimpleNamespace(
            index=d["index"],
            chunk_type=d["chunk_type"],
            start_ms=d["start_ms"],
            duration_ms=d["duration_ms"],
        )
        for d in data["chunks"]
    ]


def _iter_cached_chunks(
    chunk_dir: Path,
    manifest: list[SimpleNamespace],
) -> list[tuple[int, bytes, int, int]]:
    """Retourne [(chunk_index, data, start_ms, duration_ms)] pour les chunks présents."""
    results = []
    meta_by_index = {m.index: m for m in manifest}
    for f in sorted(chunk_dir.iterdir()):
        if not f.suffix == ".bin":
            continue
        # Nommage : chunk_03.bin → index 3
        try:
            idx = int(f.stem.replace("chunk_", ""))
        except ValueError:
            continue
        if idx not in meta_by_index:
            continue
        m = meta_by_index[idx]
        if m.chunk_type != 2:  # chunk_type 2 = REPLICATION_DATA
            continue
        results.append((idx, f.read_bytes(), m.start_ms, m.duration_ms))
    return results


# ── Scan principal ─────────────────────────────────────────────────────────────


def scan_match(
    match_id: str,
    chunk_dir: Path,
    manifest_dir: Path,
    verbose: bool = False,
) -> MatchScanResult:
    """Scanne les position frames de tous les chunks disponibles d'un match."""
    result = MatchScanResult(
        match_id=match_id,
        chunks_scanned=[],
        player_tracks={},
        total_position_frames=0,
        errors=[],
    )

    manifest = _load_manifest(manifest_dir, match_id)
    if manifest is None:
        result.errors.append("manifest non trouvé")
        return result

    chunks = _iter_cached_chunks(chunk_dir, manifest)
    if not chunks:
        result.errors.append("aucun chunk REPLICATION_DATA en cache")
        return result

    for chunk_idx, data, start_ms, duration_ms in chunks:
        # Indexer les paquets FRAME pour timestamps précis
        try:
            packets = index_chunk(data)
            ts_fn = build_packet_estimator(packets, float(start_ms))
        except Exception as exc:
            result.errors.append(f"chunk {chunk_idx}: index_chunk failed: {exc}")
            ts_fn = lambda pos: float(start_ms)  # noqa: E731
            packets = []

        # Trouver tous les markers A0 7B 42
        frames_in_chunk: list[PositionFrame] = []
        pos = 0
        while True:
            idx = data.find(FRAME_MARKER, pos)
            if idx == -1:
                break
            frame = _decode_position_frame(data, idx)
            if frame is not None:
                frame.timestamp_ms = ts_fn(idx)
                frames_in_chunk.append(frame)
                result.total_position_frames += 1
            pos = idx + 1

        # Trier par timestamp
        frames_in_chunk.sort(key=lambda f: f.timestamp_ms)

        # Mettre à jour les tracks par joueur
        pis_seen: set[int] = set()
        for frame in frames_in_chunk:
            key = f"pi_{frame.player_index}_{frame.stream}"
            if key not in result.player_tracks:
                result.player_tracks[key] = PlayerTrack(
                    player_index=frame.player_index,
                    stream=frame.stream,
                )
            track = result.player_tracks[key]

            if track.first_frame_ms is None:
                track.first_frame_ms = frame.timestamp_ms
                track.first_frame_chunk = chunk_idx

            _update_track(track, frame)
            pis_seen.add(frame.player_index)

        result.chunks_scanned.append(
            ChunkScanResult(
                chunk_index=chunk_idx,
                start_ms=start_ms,
                duration_ms=duration_ms,
                n_position_frames=len(frames_in_chunk),
                n_packets=len(packets),
                player_indices_seen=pis_seen,
            )
        )

        if verbose:
            print(
                f"  chunk {chunk_idx:02d} [{start_ms/1000:.0f}s-{(start_ms+duration_ms)/1000:.0f}s]"
                f"  {len(frames_in_chunk):4d} pos-frames"
                f"  pi={sorted(pis_seen)}"
                f"  paquets={len(packets)}"
            )

    return result


# ── Rapport ───────────────────────────────────────────────────────────────────


def _fmt_ms(ms: float) -> str:
    """Formate un timestamp ms en mm:ss.mmm."""
    total_s = ms / 1000
    m = int(total_s) // 60
    s = total_s - m * 60
    return f"{m:02d}:{s:06.3f}"


def print_report(result: MatchScanResult, verbose: bool = False) -> None:
    """Affiche le rapport de scan d'un match."""
    mid = result.match_id
    print(f"\n{'='*70}")
    print(f"MATCH {mid[:8]}...  |  chunks scannés={len(result.chunks_scanned)}"
          f"  |  total pos-frames={result.total_position_frames}")
    if result.errors:
        for e in result.errors:
            print(f"  ⚠ {e}")
        if result.total_position_frames == 0:
            return

    # Résumé par joueur
    print(f"\n  {'PI':>3}  {'Stream':6}  {'Frames':>7}  {'Premier frame':>14}  "
          f"{'1er chunk':>9}  {'Respawns':>8}")
    print(f"  {'-'*3}  {'-'*6}  {'-'*7}  {'-'*14}  {'-'*9}  {'-'*8}")

    for key in sorted(result.player_tracks):
        t = result.player_tracks[key]
        first = _fmt_ms(t.first_frame_ms) if t.first_frame_ms is not None else "       -"
        print(
            f"  {t.player_index:>3}  {t.stream:6}  {t.n_frames:>7}  "
            f"  {first}  chunk_{t.first_frame_chunk or '-':02}  {len(t.respawn_events):>8}"
        )
        if verbose and t.respawn_events:
            for ev in t.respawn_events[:5]:
                print(
                    f"         respawn @ {_fmt_ms(ev['timestamp_ms'])}"
                    f"  ΔY={ev['delta_y']:+6d}  ΔX={ev['delta_x']:+5d}"
                    f"  →  Y={ev['y_raw_after']}  X={ev['x_raw_after']}"
                )
            if len(t.respawn_events) > 5:
                print(f"         ... +{len(t.respawn_events)-5} autres respawns")


def print_summary(results: list[MatchScanResult]) -> None:
    """Résumé global sur tous les matchs."""
    total_frames = sum(r.total_position_frames for r in results)
    total_respawns = sum(
        len(t.respawn_events)
        for r in results
        for t in r.player_tracks.values()
    )
    matches_with_data = sum(1 for r in results if r.total_position_frames > 0)
    matches_with_respawns = sum(
        1 for r in results
        if any(t.respawn_events for t in r.player_tracks.values())
    )

    print(f"\n{'='*70}")
    print("RÉSUMÉ GLOBAL")
    print(f"  Matchs analysés      : {len(results)}")
    print(f"  Matchs avec données  : {matches_with_data}/{len(results)}")
    print(f"  Total pos-frames     : {total_frames:,}")
    print(f"  Matchs avec respawns : {matches_with_respawns}/{len(results)}")
    print(f"  Total respawns       : {total_respawns}")

    if total_frames > 0:
        print("\n  → THÉORIE CONFIRMÉE : les chunks REPLICATION_DATA contiennent")
        print("    bien des frames de position A0 7B 42 datables par build_packet_estimator.")
        if total_respawns > 0:
            print("  → Les discontinuités (|delta| > seuil) détectent les respawns.")
    else:
        print("\n  → Aucun position frame détecté — vérifier les seuils de filtrage.")


# ── CLI ───────────────────────────────────────────────────────────────────────


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Scan expérimental des position frames filmshell."
    )
    parser.add_argument(
        "--match-id",
        help="Scanner un seul match (par ID complet ou préfixe 8 chars).",
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Affiche le détail chunk par chunk et les événements respawn.",
    )
    parser.add_argument(
        "--chunk-dir",
        default="data/cache/film_chunks",
        help="Répertoire des chunks en cache (défaut: data/cache/film_chunks).",
    )
    parser.add_argument(
        "--manifest-dir",
        default="data/cache/film_manifests",
        help="Répertoire des manifests (défaut: data/cache/film_manifests).",
    )
    args = parser.parse_args()

    chunk_root = Path(args.chunk_dir)
    manifest_dir = Path(args.manifest_dir)

    if args.match_id:
        # Résoudre ID partiel
        prefix = args.match_id[:8]
        matching = [m for m in SESSION_MATCH_IDS if m.startswith(prefix)]
        if not matching:
            # Tenter dans le cache directement
            candidates = [d.name for d in chunk_root.iterdir() if d.name.startswith(prefix)]
            if candidates:
                matching = [candidates[0] + "-xxxx-xxxx-xxxx-xxxxxxxxxxxx"]
        match_ids = matching if matching else [args.match_id]
    else:
        match_ids = SESSION_MATCH_IDS

    results: list[MatchScanResult] = []
    for mid in match_ids:
        chunk_dir = chunk_root / mid[:8]
        if not chunk_dir.exists():
            print(f"[SKIP] {mid[:8]} — pas de cache local")
            continue

        print(f"\n[SCAN] {mid[:8]}...")
        if args.verbose:
            print("  Chunks REPLICATION_DATA :")

        r = scan_match(mid, chunk_dir, manifest_dir, verbose=args.verbose)
        results.append(r)
        print_report(r, verbose=args.verbose)

    if results:
        print_summary(results)


if __name__ == "__main__":
    main()
