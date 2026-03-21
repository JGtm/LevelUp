"""Migration des citation_name_norm et award_name vers des IDs techniques.

Ce script met à jour :
1. metadata.duckdb : citation_mappings.citation_name_norm + award_name + composite_children
2. Per-player stats.duckdb : match_citations.citation_name_norm
3. Per-player stats.duckdb : personal_score_awards.award_name

Usage:
    python scripts/migration/migrate_to_technical_ids.py --dry-run
    python scripts/migration/migrate_to_technical_ids.py
"""

from __future__ import annotations

import argparse
import json
import shutil
from pathlib import Path

import duckdb

# ── Mapping citation_name_norm : ancien (FR normalisé) → nouveau (snake_case EN) ──
CITATION_NORM_MAP: dict[str, str] = {
    # PVP
    "a la charge": "charge",
    "annexion forcee": "forced_annexation",
    "defenseur du drapeau": "flag_defender",
    "je te tiens !": "got_you",
    "partie prenante": "stakeholder",
    "sus au porteur du drapeau": "flag_carrier_hunter",
    "victoire au drapeau": "flag_victory",
    "victoire en assassin": "slayer_victory",
    "victoire en bases": "strongholds_victory",
    "ecrasement": "splatter",
    "pilote": "driver",
    "assassin": "assassin",
    "carnage de spartans": "spartan_carnage",
    "combat rapproche": "close_combat",
    "combattant opportuniste": "opportunist",
    "multifrag": "multikill",
    "pugilat": "melee_fighter",
    "tir a la tete": "headshot",
    "tueur de spartans": "spartan_killer",
    "oeil de lynx": "eagle_eye",
    "\u0153il de lynx": "eagle_eye",  # variante avec ligature \u0153
    "flag 'em down": "flag_em_down",
    "grand theft": "grand_theft",
    "helping hand": "helping_hand",
    "i'm just perfect": "im_just_perfect",
    "lawnmower": "lawnmower",
    "look ma no pin": "look_ma_no_pin",
    "lucky": "lucky",
    "no hard feelings": "no_hard_feelings",
    "positive contribution": "positive_contribution",
    "power play": "power_play",
    "road trip": "road_trip",
    "sting like a bee": "sting_like_a_bee",
    "the reaper": "the_reaper",
    "too fast for you": "too_fast_for_you",
    "vandalisme": "vandalism",
    "destructeur d'apparitions": "wraith_destroyer",
    "destructeur de banshees": "banshee_destroyer",
    "destructeur de ghosts": "ghost_destroyer",
    "destructeur de mongooses": "mongoose_destroyer",
    "destructeur de scorpions": "scorpion_destroyer",
    "destructeur de warthogs": "warthog_destroyer",
    "destructeur de wasps": "wasp_destroyer",
    # PVE
    "tueur de grognards": "grunt_slayer",
    "tueur d'elites": "elite_slayer",
    "tueur de rapaces": "jackal_slayer",
    "tueur de chasseurs": "hunter_slayer",
    "tueur de sentinelles": "sentinel_slayer",
    "like a boss": "like_a_boss",
    "player vs everything": "player_vs_everything",
    "tueur de brutes": "brute_slayer",
    "tueur de skimmers": "skimmer_slayer",
    "tueur de marines": "marine_slayer",
    # Composite
    "destructeur de covenants": "covenant_destroyer",
}

# ── Mapping award_name : ancien (FR/EN display) → nouveau (technique) ─────
AWARD_NAME_MAP: dict[str, str] = {
    # ── Combat ──
    "Joueur tué": "killed_player",
    "Trahison": "betrayed_player",
    "Auto-destruction": "self_destruction",
    "Joueur éliminé": "eliminated_player",
    "Joueur réanimé": "revived_player",
    "Réanimation empêchée": "revive_denied",
    # ── Assistances ──
    "Assistance kill": "kill_assist",
    "Assistance marquage": "mark_assist",
    "Assistance capteur": "sensor_assist",
    "Assistance EMP": "emp_assist",
    "Assistance conducteur": "driver_assist",
    # ── CTF ──
    "Drapeau capturé": "flag_captured",
    "Drapeau volé": "flag_stolen",
    "Drapeau ramené": "flag_returned",
    "Drapeau pris": "flag_taken",
    "Assistance capture": "flag_capture_assist",
    "Porteur arrêté": "runner_stopped",
    # ── Oddball ──
    "Contrôle balle": "ball_control",
    "Balle prise": "ball_taken",
    # ── KOTH ──
    "Contrôle colline": "hill_control",
    "Points colline": "hill_scored",
    # ── Zones ──
    "Zone capturée": "zone_captured",
    "Zone Capture": "zone_captured",
    "Zone sécurisée": "zone_secured",
    # ── Stockpile ──
    "Graine sécurisée": "power_seed_secured",
    "Graine volée": "power_seed_stolen",
    "Porteur tué": "carrier_killed",
    "Stockpile marqué": "stockpile_scored",
    # ── Extraction ──
    "Extraction initiée": "extraction_initiated",
    "Extraction convertie": "extraction_converted",
    "Extraction complète": "extraction_completed",
    "Extraction refusée": "extraction_denied",
    # ── Infection ──
    "Conversion empêchée": "conversion_denied",
    # ── LSS ──
    "XP bonus collecté": "collected_bonus_xp",
    # ── Hacking ──
    "Terminal piraté": "hacked_terminal",
    # ── Custom ──
    "Personnalisé": "custom",
    # ── Score générique ──
    "Score": "score",
    # ── Objectifs EN legacy ──
    "Zone Captured": "zone_captured",
    "Carrier Killed": "carrier_killed",
    "Flag Returned": "flag_returned",
    "Zone Secured": "zone_secured",
    "Runner Stopped": "runner_stopped",
    "Flag Carrier Kill": "runner_stopped",
    "Flag Carrier Killed": "runner_stopped",
    # ── Véhicules détruits (stockés en MAJUSCULES depuis l'enum API) ──
    "DESTROYED_BANSHEE": "destroyed_banshee",
    "DESTROYED_GHOST": "destroyed_ghost",
    "DESTROYED_MONGOOSE": "destroyed_mongoose",
    "DESTROYED_SCORPION": "destroyed_scorpion",
    "DESTROYED_WARTHOG": "destroyed_warthog",
    "DESTROYED_WASP": "destroyed_wasp",
    "DESTROYED_WRAITH": "destroyed_wraith",
    "DESTROYED_CHOPPER": "destroyed_chopper",
    "DESTROYED_GUNGOOSE": "destroyed_gungoose",
    "DESTROYED_ROCKET_WARTHOG": "destroyed_rocket_warthog",
    "DESTROYED_RAZORBACK": "destroyed_razorback",
    # ── Hijacks ──
    "HIJACKED_BANSHEE": "hijacked_banshee",
    "HIJACKED_GHOST": "hijacked_ghost",
    "HIJACKED_MONGOOSE": "hijacked_mongoose",
    "HIJACKED_WARTHOG": "hijacked_warthog",
    "HIJACKED_WASP": "hijacked_wasp",
    "HIJACKED_WRAITH": "hijacked_wraith",
    "HIJACKED_CHOPPER": "hijacked_chopper",
    "HIJACKED_SCORPION": "hijacked_scorpion",
    "HIJACKED_GUNGOOSE": "hijacked_gungoose",
    "HIJACKED_ROCKET_WARTHOG": "hijacked_rocket_warthog",
    "HIJACKED_RAZORBACK": "hijacked_razorback",
    # ── Legacy anglais (display names d'anciennes syncs) ──
    "Wraith Destroyed": "destroyed_wraith",
    "Mongoose Destroyed": "destroyed_mongoose",
    "Warthog Destroyed": "destroyed_warthog",
    "Rocket Warthog Destroyed": "destroyed_rocket_warthog",
    "Apparition Destroyed": "destroyed_wraith",
    "Ghost Destroyed": "destroyed_ghost",
    "Banshee Destroyed": "destroyed_banshee",
    "Scorpion Destroyed": "destroyed_scorpion",
    "Wasp Destroyed": "destroyed_wasp",
}


def backup_db(db_path: Path, backup_dir: Path) -> None:
    """Sauvegarde une base DuckDB avant migration."""
    backup_dir.mkdir(parents=True, exist_ok=True)
    dest = backup_dir / db_path.name
    if not dest.exists():
        shutil.copy2(db_path, dest)
        print(f"  Backup: {dest}")


def migrate_citation_norms(
    con: duckdb.DuckDBPyConnection, table: str, dry_run: bool
) -> int:
    """Met à jour les citation_name_norm dans une table."""
    updated = 0
    for old, new in CITATION_NORM_MAP.items():
        if old == new:
            continue
        count = con.execute(
            f"SELECT COUNT(*) FROM {table} WHERE citation_name_norm = ?", [old]
        ).fetchone()[0]
        if count > 0:
            if dry_run:
                print(f"    [DRY-RUN] {table}: '{old}' -> '{new}' ({count} rows)")
            else:
                con.execute(
                    f"UPDATE {table} SET citation_name_norm = ? "
                    f"WHERE citation_name_norm = ?",
                    [new, old],
                )
                print(f"    OK {table}: '{old}' -> '{new}' ({count} rows)")
            updated += count
    return updated


def migrate_award_names(
    con: duckdb.DuckDBPyConnection, table: str, dry_run: bool
) -> int:
    """Met à jour les award_name dans une table."""
    updated = 0
    for old, new in AWARD_NAME_MAP.items():
        count = con.execute(
            f"SELECT COUNT(*) FROM {table} WHERE award_name = ?", [old]
        ).fetchone()[0]
        if count > 0:
            if dry_run:
                print(f"    [DRY-RUN] {table}: '{old}' -> '{new}' ({count} rows)")
            else:
                con.execute(
                    f"UPDATE {table} SET award_name = ? WHERE award_name = ?",
                    [new, old],
                )
                print(f"    OK {table}: '{old}' -> '{new}' ({count} rows)")
            updated += count
    return updated


def migrate_composite_children(
    con: duckdb.DuckDBPyConnection, dry_run: bool
) -> int:
    """Met à jour les composite_children dans citation_mappings."""
    rows = con.execute(
        "SELECT citation_name_norm, composite_children FROM citation_mappings "
        "WHERE composite_children IS NOT NULL AND composite_children != ''"
    ).fetchall()
    updated = 0
    for norm, children_str in rows:
        if not children_str:
            continue
        # Parse JSON array properly (format: '["a", "b", ...]')
        try:
            children = json.loads(children_str)
        except (json.JSONDecodeError, TypeError):
            # Fallback: comma-separated plain text
            children = [c.strip() for c in children_str.split(",")]
        new_children = [CITATION_NORM_MAP.get(c, c) for c in children]
        new_str = json.dumps(new_children, ensure_ascii=False)
        if new_str != children_str:
            if dry_run:
                print(
                    f"    [DRY-RUN] composite_children: "
                    f"'{children_str}' -> '{new_str}'"
                )
            else:
                con.execute(
                    "UPDATE citation_mappings SET composite_children = ? "
                    "WHERE citation_name_norm = ?",
                    [new_str, norm],
                )
                print(
                    f"    OK composite_children: '{children_str}' -> '{new_str}'"
                )
            updated += 1
    return updated


def table_exists(con: duckdb.DuckDBPyConnection, table: str) -> bool:
    """Vérifie si une table existe."""
    result = con.execute(
        "SELECT COUNT(*) FROM information_schema.tables "
        "WHERE table_name = ?",
        [table],
    ).fetchone()
    return result[0] > 0


def has_column(con: duckdb.DuckDBPyConnection, table: str, column: str) -> bool:
    """Vérifie si une colonne existe dans une table."""
    result = con.execute(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_name = ? AND column_name = ?",
        [table, column],
    ).fetchone()
    return result[0] > 0


def main() -> None:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(
        description="Migration citation_name_norm et award_name vers IDs techniques"
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Afficher les changements sans modifier les données",
    )
    args = parser.parse_args()

    root = Path(__file__).resolve().parent.parent.parent
    backup_dir = root / "backups" / "pre_technical_ids"

    total_updated = 0

    # ── 1. metadata.duckdb ─────────────────────────────────────────────
    meta_path = root / "data" / "warehouse" / "metadata.duckdb"
    if meta_path.exists():
        print(f"\n{'='*60}")
        print(f"[DB] {meta_path.name}")
        backup_db(meta_path, backup_dir)
        con = duckdb.connect(str(meta_path))
        try:
            if table_exists(con, "citation_mappings"):
                total_updated += migrate_citation_norms(
                    con, "citation_mappings", args.dry_run
                )
                total_updated += migrate_composite_children(con, args.dry_run)
                if has_column(con, "citation_mappings", "award_name"):
                    total_updated += migrate_award_names(
                        con, "citation_mappings", args.dry_run
                    )
        finally:
            con.close()

    # ── 2. Per-player stats.duckdb ─────────────────────────────────────
    players_dir = root / "data" / "players"
    if players_dir.exists():
        for player_dir in sorted(players_dir.iterdir()):
            if not player_dir.is_dir():
                continue
            stats_path = player_dir / "stats.duckdb"
            if not stats_path.exists():
                continue

            print(f"\n{'='*60}")
            print(f"[DB] {player_dir.name}/stats.duckdb")
            backup_db(stats_path, backup_dir / player_dir.name)

            con = duckdb.connect(str(stats_path))
            try:
                # match_citations
                if table_exists(con, "match_citations"):
                    total_updated += migrate_citation_norms(
                        con, "match_citations", args.dry_run
                    )

                # personal_score_awards
                if table_exists(con, "personal_score_awards"):
                    total_updated += migrate_award_names(
                        con, "personal_score_awards", args.dry_run
                    )

                # citation_mappings (copie locale éventuelle)
                if table_exists(con, "citation_mappings"):
                    total_updated += migrate_citation_norms(
                        con, "citation_mappings", args.dry_run
                    )
            finally:
                con.close()

    # ── Résumé ─────────────────────────────────────────────────────────
    print(f"\n{'='*60}")
    mode = "[DRY-RUN] " if args.dry_run else ""
    print(f"{mode}Total: {total_updated} rows à migrer")
    if args.dry_run:
        print("Relancer sans --dry-run pour appliquer.")


if __name__ == "__main__":
    main()
