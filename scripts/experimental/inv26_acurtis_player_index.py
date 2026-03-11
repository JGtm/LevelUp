#!/usr/bin/env python3
"""
Investigation #26 — Méthode acurtis : player_index non-byte-aligned via bitstring

════════════════════════════════════════════════════════════════════════════════
CONTEXTE (acurtis 2026-03-03, issue GitHub SPNKr)
════════════════════════════════════════════════════════════════════════════════

acurtis localise le player_index de chaque joueur en cherchant son XUID dans
le bitstream raw du chunk (non-byte-aligned), puis en lisant les 5 bits qui le
précèdent :

    from bitstring import Bits

    def get_player_index(bits: Bits, player_id: str) -> int:
        term = Bits(uintle=int(player_id[5:-1]), length=64)   # XUID en LE
        position = bits.find(term, bytealigned=False)          # non-aligné
        if not position:
            raise ValueError(f"Player {player_id} not found")
        return bits[position[0] - 5 : position[0]].uint        # 5 bits avant

Il précise également :
  - Sauter les joueurs déjà trouvés (index constant cross-chunks)
  - Sauter les joueurs pas encore rejoints à la fin du chunk (join_time)
  - La séquence 00→07 pour 8 joueurs dans ses chunks (Team Slayer classique)

DIFFÉRENCE VS NOTRE INV #25 (byte-aligned)
  - Inv #25 : data.find(xuid_bytes) → lit data[idx-1] (1 byte entier)
  - Inv #26 : Bits.find(xuid_bits, bytealigned=False) → lit bits[pos-5:pos].uint

OBJECTIF
  1. Télécharger TOUS les chunks REPLICATION_DATA des 3 matchs récents JGtm
  2. Appliquer la méthode acurtis chunk par chunk
  3. Comparer byte-aligned vs non-byte-aligned
  4. Vérifier si la séquence 00→07 est reproductible sur nos films

Matchs cibles (JGtm, 2026-03-01) :
  d9329229  Team Slayer Arena — Streets
  00162144  Slayer Arena       — Smallhalla
  63d6f727  Team Slayer Arena  — Shiro
════════════════════════════════════════════════════════════════════════════════
"""

from __future__ import annotations

import argparse
import asyncio
import struct
import sys
import zlib
from pathlib import Path

try:
    from bitstring import Bits
except ImportError:
    print("ERREUR : bitstring non installé. Lancer : pip install bitstring")
    sys.exit(1)

# ── Chemins ──────────────────────────────────────────────────────────────────
FILM_ROOT = Path(__file__).resolve().parents[2]  # LevelUp-film-weapons
LEVELUP_ROOT = FILM_ROOT.parent / "LevelUp"  # repo sibling
sys.path.insert(0, str(LEVELUP_ROOT))
sys.path.insert(0, str(LEVELUP_ROOT / "src"))

CACHE_DIR = FILM_ROOT / "data" / "investigation" / "chunks"

TARGET_MATCHES = [
    "d9329229-693f-429c-a84c-3fd2784bcf4a",  # Team Slayer — Streets   (4v4)
    "00162144-fd65-4e5a-ba7a-8a121c53dafa",  # Slayer Arena — Smallhalla
    "63d6f727-d6a6-42ff-a8ba-821d4bc12060",  # Team Slayer — Shiro     (4v4)
]
GAMERTAG = "JGtm"

# ─────────────────────────────────────────────────────────────────────────────
# DB helpers
# ─────────────────────────────────────────────────────────────────────────────


def load_match_info(match_id: str) -> dict:
    """Charge les infos du match ET la liste des XUIDs+gamertags depuis shared_matches."""
    import duckdb

    db_path = LEVELUP_ROOT / "data" / "warehouse" / "shared_matches.duckdb"
    if not db_path.exists():
        print(f"  [warn] DB introuvable : {db_path}")
        return {}

    con = duckdb.connect(str(db_path), read_only=True)
    try:
        # UUID complet
        row = con.execute(
            "SELECT match_id, mode_category, game_variant_name, map_name, player_count "
            "FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()
        if not row:
            # essai recherche par préfixe
            row = con.execute(
                "SELECT match_id, mode_category, game_variant_name, map_name, player_count "
                "FROM match_registry WHERE REPLACE(match_id,'-','') LIKE ?",
                [f"{match_id.replace('-','')[:8]}%"],
            ).fetchone()
        if not row:
            print(f"  [warn] Match {match_id[:8]} introuvable en DB")
            return {}

        full_uuid, mode, variant, map_name, player_count = row

        # XUIDs + gamertags + team_id
        players = con.execute(
            """
            SELECT mp.xuid, xa.gamertag, mp.team_id
            FROM match_participants mp
            LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
            WHERE mp.match_id = ?
            ORDER BY mp.team_id, mp.rank
            """,
            [full_uuid],
        ).fetchall()

        return {
            "match_id": full_uuid,
            "mode": mode,
            "variant": variant,
            "map": map_name,
            "player_count": player_count,
            "players": [
                {"xuid": int(xuid), "gamertag": gt or f"xuid:{xuid}", "team": team_id}
                for xuid, gt, team_id in players
                if xuid is not None
            ],
        }
    finally:
        con.close()


# ─────────────────────────────────────────────────────────────────────────────
# Téléchargement de TOUS les chunks REPLICATION_DATA
# ─────────────────────────────────────────────────────────────────────────────


def _load_env_local() -> None:
    """Charge .env.local si présent (tokens SPNKr)."""
    env_path = LEVELUP_ROOT / ".env.local"
    if not env_path.exists():
        env_path = FILM_ROOT / ".env.local"
    if env_path.exists():
        for line in env_path.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, _, v = line.partition("=")
            import os

            os.environ.setdefault(k.strip(), v.strip())


async def download_all_replication_chunks(
    match_id: str,
    cache_dir: Path,
    force: bool = False,
) -> dict[int, tuple[bytes, int, int]]:
    """
    Télécharge TOUS les chunks REPLICATION_DATA (type=2) du match.
    Retourne {chunk_index: (decompressed_bytes, start_ms, duration_ms)}.
    Cache sur disque.
    """
    _load_env_local()

    from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env

    tokens = await get_tokens_from_env()
    cache_dir.mkdir(parents=True, exist_ok=True)

    async with SPNKrAPIClient(tokens=tokens) as api:
        film_response = await api.client.discovery_ugc.get_film_by_match_id(match_id)
        film = await film_response.parse()
        blob_prefix = film.blob_storage_path_prefix
        chunks_meta = film.custom_data.chunks

        # Tous les chunks REPLICATION_DATA (type 2)
        repl_chunks = [ch for ch in chunks_meta if ch.chunk_type.value == 2]
        print(f"  Chunks REPLICATION_DATA disponibles : {len(repl_chunks)}")

        to_download = []
        result: dict[int, tuple[bytes, int, int]] = {}

        for ch in sorted(repl_chunks, key=lambda c: c.index):
            cache_path = cache_dir / f"chunk_{ch.index:02d}.bin"
            if not force and cache_path.exists():
                data = cache_path.read_bytes()
                print(f"  [cache] chunk_{ch.index:02d}  {len(data):>10,} bytes")
                result[ch.index] = (
                    data,
                    ch.chunk_start_time_offset_milliseconds,
                    ch.duration_milliseconds,
                )
            else:
                to_download.append(ch)

        if to_download:

            async def _fetch(ch) -> tuple[int, bytes, int, int]:
                url = blob_prefix + ch.file_relative_path.lstrip("/")
                resp = await api.client._session.get(url)
                resp.raise_for_status()
                raw = await resp.read()
                data = zlib.decompress(raw)
                cache_path = cache_dir / f"chunk_{ch.index:02d}.bin"
                cache_path.write_bytes(data)
                print(
                    f"  [dl]    chunk_{ch.index:02d} "
                    f"{ch.chunk_size:>8,} → {len(data):>10,} bytes"
                )
                return (
                    ch.index,
                    data,
                    ch.chunk_start_time_offset_milliseconds,
                    ch.duration_milliseconds,
                )

            downloaded = await asyncio.gather(*(_fetch(ch) for ch in to_download))
            for idx, data, start_ms, dur_ms in downloaded:
                result[idx] = (data, start_ms, dur_ms)

    return result


def load_cached_chunks(cache_dir: Path) -> dict[int, tuple[bytes, int, int]]:
    """Charge les chunks déjà en cache (sans téléchargement)."""
    result: dict[int, tuple[bytes, int, int]] = {}
    for path in sorted(cache_dir.glob("chunk_*.bin")):
        try:
            idx = int(path.stem.split("_")[1])
        except (ValueError, IndexError):
            continue
        data = path.read_bytes()
        # On n'a pas les métadonnées start_ms/dur_ms en cache pur → 0
        result[idx] = (data, 0, 0)
    return result


# ─────────────────────────────────────────────────────────────────────────────
# Méthode acurtis — non-byte-aligned bitstring
# ─────────────────────────────────────────────────────────────────────────────


def get_player_index_acurtis(bits: Bits, xuid: int) -> int | None:
    """
    Implémentation directe de la méthode acurtis.

    Cherche le XUID (little-endian uint64) non-aligné sur byte dans le bitstream,
    puis lit les 5 bits précédant la première occurrence.

    Retourne None si le XUID n'est pas trouvé.
    """
    term = Bits(uintle=xuid, length=64)
    position = bits.find(term, bytealigned=False)
    if not position:
        return None
    bit_pos = position[0]
    if bit_pos < 5:
        return None
    return bits[bit_pos - 5 : bit_pos].uint


def get_player_index_acurtis_all(bits: Bits, xuid: int) -> list[tuple[int, int]]:
    """
    Retourne [(bit_position, player_index)] pour TOUTES les occurrences du XUID.
    Permet de voir si l'index est constant à travers le bitstream.
    """
    term = Bits(uintle=xuid, length=64)
    results = []
    start = 0
    while True:
        pos = bits.find(term, start=start, bytealigned=False)
        if not pos:
            break
        bit_pos = pos[0]
        pidx = bits[bit_pos - 5 : bit_pos].uint if bit_pos >= 5 else -1
        results.append((bit_pos, pidx))
        start = bit_pos + 1
    return results


# ─────────────────────────────────────────────────────────────────────────────
# Méthode byte-aligned (inv #25) — pour comparaison
# ─────────────────────────────────────────────────────────────────────────────


def get_player_index_byte_aligned(data: bytes, xuid: int) -> int | None:
    """
    Méthode inv #25 : cherche XUID aligné sur byte, retourne data[idx-1].
    Retourne None si XUID absent des données.
    """
    needle = struct.pack("<Q", xuid)
    idx = data.find(needle)
    if idx == -1:
        return None
    return data[idx - 1] if idx > 0 else None


def get_all_byte_aligned(data: bytes, xuid: int) -> list[tuple[int, int]]:
    """Retourne [(offset, prec_byte)] pour toutes les occurrences byte-aligned."""
    needle = struct.pack("<Q", xuid)
    results = []
    start = 0
    while True:
        idx = data.find(needle, start)
        if idx == -1:
            break
        prec = data[idx - 1] if idx > 0 else -1
        results.append((idx, prec))
        start = idx + 1
    return results


# ─────────────────────────────────────────────────────────────────────────────
# Analyse d'un match
# ─────────────────────────────────────────────────────────────────────────────


def analyze_match(
    match_id: str,
    chunks: dict[int, tuple[bytes, int, int]],
    players: list[dict],
    verbose: bool = False,
) -> None:
    """
    Pour chaque chunk, applique les deux méthodes sur chaque joueur.
    Affiche un tableau comparatif.
    """
    print(f"\n{'═'*80}")
    print(f"Match  : {match_id}")
    print(f"Chunks : {len(chunks)} REPLICATION_DATA")
    print(f"Joueurs: {len(players)}")
    print(f"{'═'*80}")

    if not chunks:
        print("  aucun chunk disponible — passer --download")
        return

    # Concat all chunks pour méthode byte-aligned globale
    all_data = b"".join(data for _, (data, _, _) in sorted(chunks.items()))
    all_bits = Bits(all_data)

    print(f"Taille concat : {len(all_data):,} bytes ({len(all_bits):,} bits)\n")

    # ── Tableau résultat acurtis (global) ─────────────────────────────────────
    print(
        f"{'Gamertag':<22} {'XUID':<20} {'acurtis_idx':>12} {'byte_prec':>10}  Toutes occurrences (bit_pos → idx)"
    )
    print("─" * 100)

    idx_seen: dict[int, int] = {}  # xuid → premier index acurtis trouvé

    for p in sorted(players, key=lambda x: x["xuid"]):
        xuid = p["xuid"]
        gt = p["gamertag"]

        # Méthode acurtis (global concat)
        all_occurrences = get_player_index_acurtis_all(all_bits, xuid)
        first_acurtis = all_occurrences[0][1] if all_occurrences else None
        acurtis_str = f"{first_acurtis}" if first_acurtis is not None else "—"

        # Méthode byte-aligned
        byte_occs = get_all_byte_aligned(all_data, xuid)
        prec_vals = sorted({p for _, p in byte_occs})
        byte_str = "/".join(f"0x{v:02x}" for v in prec_vals[:5]) if prec_vals else "—"

        # Résumé occurrences acurtis (premières 6)
        if all_occurrences:
            occ_summary = ", ".join(
                f"{bit_pos//8:,}B→{pidx}" for bit_pos, pidx in all_occurrences[:6]
            )
            if len(all_occurrences) > 6:
                occ_summary += f" (+{len(all_occurrences)-6})"
        else:
            occ_summary = "—"

        pov_flag = "★" if first_acurtis == 1 else " "
        print(f"{pov_flag}{gt:<21} {xuid:<20} {acurtis_str:>12} {byte_str:>10}  {occ_summary}")

        if first_acurtis is not None:
            idx_seen[xuid] = first_acurtis

    # ── Séquence détectée ────────────────────────────────────────────────────
    print()
    if idx_seen:
        sorted_by_idx = sorted(idx_seen.items(), key=lambda kv: kv[1])
        print("Séquence acurtis détectée :")
        for xuid, pidx in sorted_by_idx:
            gt = next((p["gamertag"] for p in players if p["xuid"] == xuid), str(xuid))
            print(f"  idx={pidx:>3}  {gt}")

        indices = [v for _, v in sorted_by_idx]
        expected = list(range(len(indices)))
        seq_ok = indices == expected
        seq_max = len(indices) - 1
        print(f"\n  Séquence 0→{seq_max} continue : {'✓ OUI' if seq_ok else '✗ NON'}")
    else:
        print("  Aucun player_index trouvé.")

    if not verbose:
        return

    # ── Mode verbose : chunk par chunk ───────────────────────────────────────
    print(f"\n{'─'*80}")
    print("Détail par chunk (verbose) :")
    print(f"{'─'*80}")

    for chunk_idx, (chunk_data, start_ms, dur_ms) in sorted(chunks.items()):
        chunk_bits = Bits(chunk_data)
        print(
            f"\n  chunk_{chunk_idx:02d}  [{start_ms//1000}s – {(start_ms+dur_ms)//1000}s]  {len(chunk_data):,} bytes"
        )
        print(f"  {'Gamertag':<22} {'acurtis':>8} {'byte':>8}")
        for p in sorted(players, key=lambda x: x["xuid"]):
            xuid = p["xuid"]
            a_idx = get_player_index_acurtis(chunk_bits, xuid)
            b_idx = get_player_index_byte_aligned(chunk_data, xuid)
            a_str = f"{a_idx}" if a_idx is not None else "—"
            b_str = f"0x{b_idx:02x}" if b_idx is not None else "—"
            if a_idx is not None or b_idx is not None:
                print(f"    {p['gamertag']:<22} {a_str:>8} {b_str:>8}")


# ─────────────────────────────────────────────────────────────────────────────
# Main async
# ─────────────────────────────────────────────────────────────────────────────


async def main_async(
    match_ids: list[str],
    download: bool,
    force: bool,
    verbose: bool,
    cache_only: bool,
) -> int:
    for match_id in match_ids:
        short = match_id.replace("-", "")[:8]
        cache_dir = CACHE_DIR / short

        print(f"\n{'━'*80}")
        print(f"Match : {match_id}")

        # Infos DB
        info = load_match_info(match_id)
        if info:
            players = info["players"]
            print(
                f"Mode  : {info['mode']} — {info['variant']} — {info['map']}"
                f"  ({len(players)} joueurs)"
            )
        else:
            players = []

        # Téléchargement ou cache
        if cache_only:
            chunks = load_cached_chunks(cache_dir)
            print(f"Cache : {len(chunks)} chunk(s) chargés depuis {cache_dir.name}/")
        elif download:
            print("Téléchargement...")
            try:
                chunks = await download_all_replication_chunks(match_id, cache_dir, force=force)
            except Exception as exc:
                print(f"  ✗ Erreur téléchargement : {exc}")
                # Fallback cache
                chunks = load_cached_chunks(cache_dir)
                if chunks:
                    print(f"  Fallback cache : {len(chunks)} chunk(s)")
                else:
                    print("  Aucun chunk disponible.")
                    continue
        else:
            chunks = load_cached_chunks(cache_dir)
            if not chunks:
                print(f"  Aucun cache trouvé dans {cache_dir}")
                print("  → Relancer avec --download pour télécharger")
                continue
            print(f"Cache : {len(chunks)} chunk(s)")

        analyze_match(match_id, chunks, players, verbose=verbose)

    return 0


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Inv #26 — Méthode acurtis : player_index non-byte-aligned"
    )
    parser.add_argument(
        "--match-ids",
        nargs="+",
        default=TARGET_MATCHES,
        metavar="UUID",
        help="Match IDs à analyser (défaut : 3 matchs récents JGtm)",
    )
    parser.add_argument(
        "--download",
        action="store_true",
        help="Télécharger tous les chunks REPLICATION_DATA (nécessite .env.local)",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Forcer le re-téléchargement même si en cache",
    )
    parser.add_argument(
        "--cache-only",
        action="store_true",
        help="Utiliser uniquement le cache local (pas de connexion API)",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Afficher le détail chunk par chunk",
    )
    args = parser.parse_args()

    return asyncio.run(
        main_async(
            match_ids=args.match_ids,
            download=args.download,
            force=args.force,
            verbose=args.verbose,
            cache_only=args.cache_only,
        )
    )


if __name__ == "__main__":
    sys.exit(main())
