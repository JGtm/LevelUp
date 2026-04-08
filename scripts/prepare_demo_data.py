"""Prépare un jeu de données de démo public pour LevelUp.

Extrait les N matchs les plus récents d'un joueur source vers data/demo/,
en anonymisant son gamertag en "DEMO" et en recréant toutes les vues V6.

Utilisation :
    python scripts/prepare_demo_data.py --gamertag JGtm --max-matches 50
    python scripts/prepare_demo_data.py --gamertag JGtm --max-matches 50 --out data/demo
"""

from __future__ import annotations

import argparse
import json
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

        # Anonymiser le gamertag source → "DEMO"
        dst.execute("UPDATE xuid_aliases SET gamertag = 'DEMO' WHERE xuid = ?", (source_xuid,))
        dst.execute(
            "UPDATE match_participants SET gamertag = 'DEMO' WHERE xuid = ?", (source_xuid,)
        )

        # Vues V6 — réutiliser la fonction de migration officielle
        from src.data.sync.migrations import ensure_resolution_views, ensure_weapon_kills_view

        ensure_resolution_views(dst)
        try:
            ensure_weapon_kills_view(dst)
        except Exception as exc:
            print(f"    [warn] v_weapon_kills: {exc}")

    print("  [shared] OK")


def _extract_player(
    src_player: Path,
    out_player: Path,
    match_ids: list[str],
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

        # Vues matérialisées (player) — recréer via la migration officielle
        from src.data.sync.migrations import ensure_player_materialized_views

        try:
            ensure_player_materialized_views(dst)
        except Exception as exc:
            print(f"    [warn] mv_player: {exc}")

    print("  [player] OK")


def _write_configs(out_dir: Path, demo_xuid: str) -> None:
    """Génère db_profiles.json et app_settings.json pour le mode démo."""
    profiles = {
        "version": "2.1",
        "warehouse_path": "data/warehouse",
        "metadata_db": "data/warehouse/metadata.duckdb",
        "profiles": {
            "DEMO": {
                "db_path": "data/players/DEMO/stats.duckdb",
                "xuid": demo_xuid,
                "waypoint_player": "DEMO",
            }
        },
    }
    (out_dir / "db_profiles.json").write_text(
        json.dumps(profiles, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    settings = {
        "lang": "fr",
        "media_enabled": False,
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
    _extract_player(
        src_player_db,
        out_dir / "players" / "DEMO" / "stats.duckdb",
        match_ids,
    )

    # 5. Écrire les fichiers de config
    print("\n[5/5] Génération des fichiers de configuration…")
    _write_configs(out_dir, demo_xuid)

    print(f"\n✅ Données démo prêtes dans {out_dir}")
    print("   Lancer le conteneur : docker compose up -d levelup-demo")


if __name__ == "__main__":
    main()
