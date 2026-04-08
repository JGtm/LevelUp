"""Prépare un jeu de données de démo public pour LevelUp.

Extrait les N matchs les plus récents d'un joueur source vers data/demo/,
en anonymisant son gamertag en "DEMO" et en recréant toutes les vues V6.

Utilisation :
    python scripts/prepare_demo_data.py --gamertag JGtm --max-matches 50
    python scripts/prepare_demo_data.py --gamertag JGtm --max-matches 50 --out data/demo
"""

from __future__ import annotations

import argparse
import contextlib
import json
import os
import shutil
import sys
from pathlib import Path

# ── Racine du projet ──────────────────────────────────────────────────────────
_REPO_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_REPO_ROOT))

import duckdb  # noqa: E402


def _copy_metadata(src_meta: Path, out_meta: Path) -> None:
    """Copie intégralement metadata.duckdb (référentiels sans données perso)."""
    out_meta.parent.mkdir(parents=True, exist_ok=True)
    print(f"  [metadata] copie {src_meta} → {out_meta}")
    shutil.copy2(src_meta, out_meta)


def _extract_shared(
    src_shared: Path,
    out_shared: Path,
    match_ids: list[str],
    source_xuid: str,
    demo_xuid: str,
) -> None:
    """Extrait les tables shared_matches_v2 filtrées sur match_ids."""
    out_shared.parent.mkdir(parents=True, exist_ok=True)
    print(f"  [shared] extraction de {len(match_ids)} matchs → {out_shared}")

    ids_literal = ", ".join(f"'{mid}'" for mid in match_ids)

    _shared_tables = [
        ("match_registry", f"match_id IN ({ids_literal})"),
        ("match_participants", f"match_id IN ({ids_literal})"),
        ("medals_earned", f"match_id IN ({ids_literal})"),
        ("highlight_events", f"match_id IN ({ids_literal})"),
        ("weapon_kills", f"match_id IN ({ids_literal})"),
        ("killer_victim_pairs", f"match_id IN ({ids_literal})"),
        (
            "xuid_aliases",
            f"""xuid IN (
                SELECT DISTINCT xuid FROM match_participants
                WHERE match_id IN ({ids_literal})
            )""",
        ),
    ]

    with duckdb.connect(str(out_shared)) as dst:
        dst.execute(f"ATTACH '{src_shared}' AS src (READ_ONLY)")

        for table, where in _shared_tables:
            dst.execute(f"CREATE TABLE {table} AS SELECT * FROM src.{table} WHERE {where}")
            count = dst.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]  # type: ignore[index]
            print(f"    {table}: {count} lignes insérées")

        dst.execute("DETACH src")

        # Le gamertag source est conservé tel quel (données du propriétaire)
        # Seul le xuid est remplacé par une valeur anonyme
        dst.execute("UPDATE xuid_aliases SET xuid = ? WHERE xuid = ?", (demo_xuid, source_xuid))
        dst.execute(
            "UPDATE match_participants SET xuid = ? WHERE xuid = ?", (demo_xuid, source_xuid)
        )

        # Vues V6 — réutiliser la fonction de migration officielle
        from src.data.sync.migrations import (
            ensure_mv_player_matches_view,
            ensure_resolution_views,
            ensure_weapon_kills_reconciled_as,
        )

        ensure_resolution_views(dst)
        try:
            ensure_weapon_kills_reconciled_as(dst)
        except Exception as exc:
            print(f"    [warn] v_weapon_kills: {exc}")

        # Vue mv_player_matches — requise par la plupart des pages analytics
        try:
            ensure_mv_player_matches_view(dst)
            print("    [shared] mv_player_matches: OK")
        except Exception as exc:
            print(f"    [warn] mv_player_matches: {exc}")

    print("  [shared] OK")


def _extract_player(
    src_player: Path,
    out_player: Path,
    match_ids: list[str],
    source_xuid: str = "",
    demo_xuid: str = "",
) -> None:
    """Extrait player_match_enrichment, match_citations, sessions filtrés sur match_ids."""
    out_player.parent.mkdir(parents=True, exist_ok=True)
    print(f"  [player] extraction → {out_player}")

    ids_literal = ", ".join(f"'{mid}'" for mid in match_ids)

    _player_tables = [
        ("player_match_enrichment", f"match_id IN ({ids_literal})"),
        ("match_citations", f"match_id IN ({ids_literal})"),
        ("sessions", "1=1"),
        ("career_progression", "1=1"),
        ("sync_meta", "key NOT IN ('msal_token_cache')"),
        ("match_skill_rank", f"match_id IN ({ids_literal})"),
    ]

    with duckdb.connect(str(out_player)) as dst:
        dst.execute(f"ATTACH '{src_player}' AS src (READ_ONLY)")

        for table, where in _player_tables:
            try:
                dst.execute(f"CREATE TABLE {table} AS SELECT * FROM src.{table} WHERE {where}")
                count = dst.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]  # type: ignore[index]
                print(f"    {table}: {count} lignes insérées")
            except Exception as exc:
                print(f"    [warn] {table}: {exc}")

        dst.execute("DETACH src")

        # Remplacer le xuid réel par le xuid démo dans les tables joueur
        if source_xuid and demo_xuid:
            for tbl in ("player_match_enrichment", "match_skill_rank"):
                with contextlib.suppress(Exception):
                    dst.execute(
                        f"UPDATE {tbl} SET xuid = ? WHERE xuid = ?",  # noqa: S608
                        (demo_xuid, source_xuid),
                    )
            # sync_meta : clé 'xuid' — source canonique lue par _resolve_player_xuid
            with contextlib.suppress(Exception):
                dst.execute(
                    "UPDATE sync_meta SET value = ? WHERE key = 'xuid' AND value = ?",
                    (demo_xuid, source_xuid),
                )

    print("  [player] OK")


# DDL media tables — doit rester synchronisé avec src/data/media_indexer.py
_MEDIA_FILES_DDL = """
CREATE TABLE IF NOT EXISTS media_files (
    file_path VARCHAR PRIMARY KEY,
    file_hash VARCHAR NOT NULL,
    file_name VARCHAR NOT NULL,
    file_size BIGINT NOT NULL,
    file_ext VARCHAR NOT NULL,
    kind VARCHAR NOT NULL,
    mtime DOUBLE NOT NULL,
    mtime_paris_epoch DOUBLE,
    thumbnail_path VARCHAR,
    thumbnail_generated_at TIMESTAMP,
    first_seen_at TIMESTAMP,
    last_scan_at TIMESTAMP,
    scan_version INTEGER,
    capture_start_utc TIMESTAMP,
    capture_end_utc TIMESTAMP,
    duration_seconds DOUBLE,
    title VARCHAR,
    status VARCHAR NOT NULL DEFAULT 'active'
)
"""

_MEDIA_ASSOC_DDL = """
CREATE TABLE IF NOT EXISTS media_match_associations (
    media_path VARCHAR NOT NULL,
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    match_start_time TIMESTAMP NOT NULL,
    association_confidence DOUBLE DEFAULT 1.0,
    associated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    map_id VARCHAR,
    map_name VARCHAR,
    PRIMARY KEY (media_path, match_id, xuid)
)
"""


def _ensure_media_tables(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée les tables média avec contraintes PRIMARY KEY si absentes."""
    conn.execute(_MEDIA_FILES_DDL)
    conn.execute(_MEDIA_ASSOC_DDL)


def _build_demo_map_index(src_shared: Path, match_ids: list[str]) -> dict[str, list[tuple]]:
    """Retourne {map_name: [(match_id, start_time), ...]} pour les matchs démo."""
    ids_lit = ", ".join(f"'{mid}'" for mid in match_ids)
    with duckdb.connect(str(src_shared), read_only=True) as s:
        rows = s.execute(f"""
            SELECT match_id, map_name, start_time FROM match_registry
            WHERE match_id IN ({ids_lit}) AND map_name IS NOT NULL
        """).fetchall()
    index: dict[str, list[tuple]] = {}
    for mid, mname, mtime in rows:
        index.setdefault(mname, []).append((mid, mtime))
    return index


def _collect_media_candidates(
    src: duckdb.DuckDBPyConnection,
    match_ids: list[str],
    demo_map_index: dict[str, list[tuple]],
    max_media: int,
) -> list[tuple[str, str, object, str]]:
    """Collecte jusqu'à max_media médias avec fallback par carte.

    Retourne [(file_path, target_match_id, target_match_start_time, map_name)].
    Priorité : médias directement associés aux matchs démo, puis médias d'autres
    matchs réassignés à un match démo sur la même carte.
    """
    ids_lit = ", ".join(f"'{mid}'" for mid in match_ids)
    seen: set[str] = set()
    result: list[tuple] = []

    direct = src.execute(f"""
        SELECT DISTINCT mf.file_path, mma.match_id, mma.match_start_time, mma.map_name
        FROM media_files mf
        JOIN media_match_associations mma ON mma.media_path = mf.file_path
        WHERE mf.status = 'active' AND mma.match_id IN ({ids_lit})
        ORDER BY mf.mtime DESC LIMIT {max_media}
    """).fetchall()
    for fp, mid, mtime, mname in direct:
        if Path(fp).exists():
            result.append((fp, mid, mtime, mname or ""))
            seen.add(fp)

    if len(result) >= max_media:
        return result[:max_media]

    # Fallback : médias hors démo réassignés à un match demo de même carte
    fallbacks = src.execute(f"""
        SELECT DISTINCT mf.file_path, mma.map_name
        FROM media_files mf
        JOIN media_match_associations mma ON mma.media_path = mf.file_path
        WHERE mf.status = 'active' AND mma.match_id NOT IN ({ids_lit})
          AND mma.map_name IS NOT NULL
        ORDER BY mf.mtime DESC LIMIT 200
    """).fetchall()
    for fp, mname in fallbacks:
        if fp in seen or not Path(fp).exists():
            continue
        demo_matches = demo_map_index.get(mname)
        if not demo_matches:
            continue
        target_mid, target_mtime = demo_matches[0]
        result.append((fp, target_mid, target_mtime, mname))
        seen.add(fp)
        if len(result) >= max_media:
            break

    return result


def _extract_media(
    src_player_db: Path,
    out_player_db: Path,
    out_dir: Path,
    match_ids: list[str],
    src_shared: Path,
    demo_xuid: str = "0000000000000000",
    max_media: int = 5,
) -> int:
    """Extrait jusqu'à max_media fichiers média depuis la DB joueur source.

    Tente d'abord les médias liés aux matchs démo ; si insuffisants, réassigne
    des médias d'autres matchs sur la même carte (fuzzy fallback).
    Note : les fichiers sources doivent être accessibles localement.

    Returns: nombre de fichiers réellement copiés.
    """
    demo_media_dir = out_dir / "players" / "DEMO" / "media"
    levelup_root = Path(os.environ.get("LEVELUP_ROOT", str(_REPO_ROOT)))
    demo_media_root = str(levelup_root / "data" / "players" / "DEMO" / "media")
    extracted = 0
    try:
        with duckdb.connect(str(src_player_db), read_only=True) as src:
            tables = {
                r[0]
                for r in src.execute(
                    "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
                ).fetchall()
            }
            if "media_files" not in tables or "media_match_associations" not in tables:
                print("    [media] tables absentes — skip")
                return 0
            demo_map_index = _build_demo_map_index(src_shared, match_ids)
            candidates = _collect_media_candidates(src, match_ids, demo_map_index, max_media)
            if not candidates:
                print("    [media] aucun fichier média disponible localement — skip")
                return 0
            # Récupérer les lignes media_files pour les candidats
            src.execute("SELECT * FROM media_files LIMIT 0")
            mf_cols = [d[0] for d in src.description]
            fp_idx = mf_cols.index("file_path")
            rows_by_fp: dict[str, tuple] = {}
            for fp, *_ in candidates:
                r = src.execute("SELECT * FROM media_files WHERE file_path = ?", [fp]).fetchone()
                if r:
                    rows_by_fp[fp] = r

        demo_media_dir.mkdir(parents=True, exist_ok=True)
        mf_ph = ", ".join("?" * len(mf_cols))
        with duckdb.connect(str(out_player_db)) as dst:
            _ensure_media_tables(dst)
            for fp, target_mid, target_mtime, map_name in candidates:
                row = rows_by_fp.get(fp)
                if row is None:
                    continue
                filename = Path(fp).name
                new_path = f"{demo_media_root}/{filename}"
                shutil.copy2(fp, demo_media_dir / filename)
                new_row = list(row)
                new_row[fp_idx] = new_path
                dst.execute(
                    f"INSERT INTO media_files ({', '.join(mf_cols)}) VALUES ({mf_ph})"  # noqa: S608
                    " ON CONFLICT (file_path) DO NOTHING",
                    new_row,
                )
                dst.execute(
                    "INSERT INTO media_match_associations"
                    " (media_path, match_id, xuid, match_start_time, map_name)"
                    " VALUES (?, ?, ?, ?, ?)"
                    " ON CONFLICT (media_path, match_id, xuid) DO NOTHING",
                    [new_path, target_mid, demo_xuid, target_mtime, map_name or None],
                )
                label = "direct" if target_mid in [c[1] for c in candidates] else "fuzzy"
                print(f"    [media] {filename} ({label}, {map_name or '?'})")
                extracted += 1
    except Exception as exc:
        print(f"    [warn] extraction media: {exc}")
    return extracted


def _write_configs(
    out_dir: Path,
    demo_xuid: str,
    gamertag: str = "DEMO",
    *,
    media_enabled: bool = False,
    service_tag: str = "",
) -> None:
    """Génère db_profiles.json et app_settings.json pour le mode démo."""
    profiles = {
        "version": "2.1",
        "warehouse_path": "data/warehouse",
        "metadata_db": "data/warehouse/metadata.duckdb",
        "profiles": {
            "DEMO": {
                "db_path": "data/players/DEMO/stats.duckdb",
                "xuid": demo_xuid,
                "waypoint_player": gamertag,
            }
        },
    }
    (out_dir / "db_profiles.json").write_text(
        json.dumps(profiles, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    settings = {
        "lang": "fr",
        "media_enabled": media_enabled,
        "spnkr_refresh_on_start": False,
        "spnkr_refresh_on_manual_refresh": False,
        "spnkr_refresh_max_matches": 0,
        "spnkr_refresh_with_highlight_events": False,
        "spnkr_refresh_with_backfill": False,
        "spnkr_refresh_backfill_medals": False,
        "spnkr_refresh_backfill_events": False,
        "spnkr_refresh_backfill_skill": False,
        "spnkr_refresh_backfill_personal_scores": False,
        "spnkr_refresh_backfill_performance_scores": False,
        "spnkr_refresh_backfill_aliases": False,
        "spnkr_refresh_backfill_lusr": False,
        "spnkr_refresh_backfill_weapons": False,
        "discord_notifications_enabled": False,
        "doppler_enabled": False,
        "tailscale_funnel_enabled": False,
        "profile_assets_download_enabled": False,
        "profile_api_enabled": False,
        "profile_service_tag": service_tag,
        "repository_mode": "duckdb",
        "enable_duckdb_analytics": True,
    }
    (out_dir / "app_settings.json").write_text(
        json.dumps(settings, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    print(f"  [config] db_profiles.json + app_settings.json écrits dans {out_dir}")


def main() -> None:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(description="Prépare les données de démo LevelUp")
    parser.add_argument("--gamertag", required=True, help="Gamertag source (joueur réel)")
    parser.add_argument("--max-matches", type=int, default=50, help="Nombre de matchs à extraire")
    parser.add_argument(
        "--out", default="data/demo", help="Répertoire de sortie (défaut: data/demo)"
    )
    parser.add_argument(
        "--service-tag", default="", help="Spartan ID (ex: SPTA) affiché sous le gamertag"
    )
    args = parser.parse_args()

    gamertag: str = args.gamertag
    max_matches: int = args.max_matches
    out_dir = _REPO_ROOT / args.out

    # Chemins source
    src_profiles_path = _REPO_ROOT / "db_profiles.json"
    profiles = json.loads(src_profiles_path.read_text(encoding="utf-8"))
    profile = profiles.get("profiles", {}).get(gamertag)
    if profile is None:
        print(f"Erreur : joueur '{gamertag}' introuvable dans db_profiles.json")
        sys.exit(1)

    source_xuid: str = str(profile["xuid"])
    src_player_db = _REPO_ROOT / profile["db_path"]
    warehouse_path = _REPO_ROOT / profiles.get("warehouse_path", "data/warehouse")
    src_shared = warehouse_path / "shared_matches_v2.duckdb"
    src_meta = warehouse_path / "metadata.duckdb"

    print("\n=== Préparation données démo ===")
    print(f"  Source     : {gamertag} (xuid={source_xuid})")
    print(f"  Matchs max : {max_matches}")
    print(f"  Sortie     : {out_dir}")

    # 1. Récupérer les match_ids à extraire
    print("\n[1/5] Sélection des matchs…")
    with duckdb.connect(str(src_shared), read_only=True) as conn:
        rows = conn.execute(
            """
            SELECT mr.match_id
            FROM match_registry mr
            JOIN match_participants mp ON mp.match_id = mr.match_id
            WHERE mp.xuid = ?
            ORDER BY mr.start_time DESC
            LIMIT ?
            """,
            (source_xuid, max_matches),
        ).fetchall()
    match_ids = [r[0] for r in rows]
    print(f"  {len(match_ids)} matchs sélectionnés")
    if not match_ids:
        print("Erreur : aucun match trouvé pour ce joueur dans shared_matches_v2")
        sys.exit(1)

    # Utiliser un xuid fictif pour la démo
    demo_xuid = "0000000000000000"

    # 2. Copier metadata.duckdb
    print("\n[2/5] Copie metadata.duckdb…")
    _copy_metadata(src_meta, out_dir / "warehouse" / "metadata.duckdb")

    # 3. Extraire shared_matches_v2.duckdb
    print("\n[3/5] Extraction shared_matches_v2.duckdb…")
    _extract_shared(
        src_shared,
        out_dir / "warehouse" / "shared_matches_v2.duckdb",
        match_ids,
        source_xuid,
        demo_xuid,
    )

    # 4. Extraire stats.duckdb joueur
    print("\n[4/5] Extraction stats.duckdb joueur…")
    out_player_db = out_dir / "players" / "DEMO" / "stats.duckdb"
    _extract_player(src_player_db, out_player_db, match_ids, source_xuid, demo_xuid)

    # 4b. Extraire médias (optionnel, 5 clips max)
    print("\n  [4b] Extraction médias…")
    media_count = _extract_media(
        src_player_db, out_player_db, out_dir, match_ids, src_shared, demo_xuid
    )
    print(f"  {media_count} fichier(s) média extrait(s)")

    # 5. Écrire les fichiers de config
    print("\n[5/5] Génération des fichiers de configuration…")
    _write_configs(
        out_dir,
        demo_xuid,
        gamertag=gamertag,
        media_enabled=media_count > 0,
        service_tag=args.service_tag,
    )

    print(f"\n✅ Données démo prêtes dans {out_dir}")
    print("   Lancer le conteneur : docker compose up -d levelup-demo")


if __name__ == "__main__":
    main()
