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


def _extract_media(
    src_player_db: Path,
    out_player_db: Path,
    out_dir: Path,
    match_ids: list[str],
    max_media: int = 5,
) -> int:
    """Extrait jusqu'à max_media fichiers média depuis la DB joueur source.

    Returns: nombre de fichiers réellement copiés.
    """
    ids_literal = ", ".join(f"'{mid}'" for mid in match_ids)
    demo_media_dir = out_dir / "players" / "DEMO" / "media"

    # Chemin base dans le container (LEVELUP_ROOT=/app en Docker)
    levelup_root = Path(os.environ.get("LEVELUP_ROOT", str(_REPO_ROOT)))
    demo_media_root = str(levelup_root / "data" / "players" / "DEMO" / "media")

    extracted = 0
    try:
        with duckdb.connect(str(src_player_db), read_only=True) as src:
            tables = {
                r[0]
                for r in src.execute(
                    "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
                ).fetchall()
            }
            if "media_files" not in tables or "media_match_associations" not in tables:
                print("    [media] tables absentes — skip")
                return 0

            rows = src.execute(f"""
                SELECT DISTINCT mf.*
                FROM media_files mf
                JOIN media_match_associations mma ON mma.media_path = mf.file_path
                WHERE mf.status = 'active'
                  AND mma.match_id IN ({ids_literal})
                ORDER BY COALESCE(epoch(mf.capture_start_utc), mf.mtime) DESC
                LIMIT {max_media}
            """).fetchall()
            col_names = [d[0] for d in src.description]

        if not rows:
            print("    [media] aucun média associé aux matchs extraits — skip")
            return 0

        fp_idx = col_names.index("file_path")
        demo_media_dir.mkdir(parents=True, exist_ok=True)
        ph = ", ".join("?" * len(col_names))

        with duckdb.connect(str(out_player_db)) as dst:
            dst.execute(f"ATTACH '{src_player_db}' AS _msrc (READ_ONLY)")
            dst.execute(
                "CREATE TABLE IF NOT EXISTS media_files AS "
                "SELECT * FROM _msrc.media_files WHERE 1=0"
            )
            dst.execute(
                "CREATE TABLE IF NOT EXISTS media_match_associations AS "
                "SELECT * FROM _msrc.media_match_associations WHERE 1=0"
            )
            assoc_cols = [
                d[0]
                for d in dst.execute(
                    "SELECT * FROM _msrc.media_match_associations WHERE 1=0"
                ).description
            ]
            mp_idx = assoc_cols.index("media_path")
            assoc_ph = ", ".join("?" * len(assoc_cols))

            for row in rows:
                src_path = Path(row[fp_idx])
                if not src_path.exists():
                    print(f"    [media] absent: {src_path.name} — skip")
                    continue

                filename = src_path.name
                new_path = f"{demo_media_root}/{filename}"
                shutil.copy2(src_path, demo_media_dir / filename)

                new_row = list(row)
                new_row[fp_idx] = new_path
                dst.execute(
                    f"INSERT INTO media_files ({', '.join(col_names)}) VALUES ({ph})"
                    " ON CONFLICT (file_path) DO NOTHING",
                    new_row,
                )

                assoc_rows = dst.execute(
                    f"SELECT * FROM _msrc.media_match_associations"
                    f" WHERE media_path = ? AND match_id IN ({ids_literal})",
                    [str(src_path)],
                ).fetchall()
                for assoc in assoc_rows:
                    a = list(assoc)
                    a[mp_idx] = new_path
                    dst.execute(
                        f"INSERT INTO media_match_associations"
                        f" ({', '.join(assoc_cols)}) VALUES ({assoc_ph})"
                        " ON CONFLICT (media_path, match_id, xuid) DO NOTHING",
                        a,
                    )

                extracted += 1
                print(f"    [media] {filename} → {new_path}")

            dst.execute("DETACH _msrc")
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
    media_count = _extract_media(src_player_db, out_player_db, out_dir, match_ids)
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
