"""Script de création du corpus de test pour les tests de parité.

Extrait ~500 matchs (et les données associées) depuis la DB de production
vers ``tests/fixtures/ref_player/`` sous forme de fichiers DuckDB légers.

Usage :
    python scripts/create_test_corpus.py --gamertag MonGamertag [--limit 500]

Le script crée :
    tests/fixtures/ref_player/
        stats.duckdb            ← données player_match_enrichment + scoped matches
        shared_matches_v2.duckdb   ← sous-ensemble shared_matches_v2 pour ce joueur
        metadata.duckdb         ← copie complète metadata.duckdb (référentiels)
        xuid.txt                ← xuid du joueur de référence
        README.md               ← mis à jour avec les stats d'extraction
"""

from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path

# Ajouter la racine du projet au sys.path
sys.path.insert(0, str(Path(__file__).parent.parent))

import duckdb

from src.utils.paths import get_shared_matches_path_from_player

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO, format="%(levelname)s  %(message)s")

FIXTURES_DIR = Path(__file__).parent.parent / "tests" / "fixtures" / "ref_player"
DEFAULT_LIMIT = 500


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _resolve_player_db(gamertag: str, profiles_path: Path | None = None) -> Path:
    """Retrouve le chemin stats.duckdb pour un gamertag donné.

    Supporte les deux formats db_profiles.json :
    - Format liste   : [{"gamertag": "...", "db_path": "..."}, ...]
    - Format dict v2 : {"profiles": {"Gamertag": {"db_path": "..."}}, ...}
    """
    import json

    if profiles_path is None:
        profiles_path = Path(__file__).parent.parent / "db_profiles.json"
    if not profiles_path.exists():
        raise FileNotFoundError(f"db_profiles.json introuvable : {profiles_path}")

    repo_root = profiles_path.parent
    with open(profiles_path, encoding="utf-8") as f:
        raw = json.load(f)

    # Normalisation : produire un itérable de (gamertag_key, db_path_str)
    if isinstance(raw, list):
        # Format legacy liste
        items = [(p.get("gamertag", ""), p.get("db_path", "")) for p in raw]
    elif isinstance(raw, dict) and "profiles" in raw:
        # Format dict v2
        items = [(gt, info.get("db_path", "")) for gt, info in raw["profiles"].items()]
    else:
        raise ValueError(f"Format db_profiles.json non reconnu dans {profiles_path}")

    for gt, db_path_str in items:
        if gt.lower() == gamertag.lower():
            db_path = Path(db_path_str)
            if not db_path.is_absolute():
                db_path = repo_root / db_path
            return db_path

    available = [gt for gt, _ in items]
    raise ValueError(
        f"Gamertag '{gamertag}' non trouvé dans db_profiles.json. "
        f"Gamertags disponibles : {available}"
    )


def _resolve_xuid(conn: duckdb.DuckDBPyConnection) -> str:
    """Extrait le xuid depuis sync_meta."""
    try:
        row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
        return str(row[0]).strip() if row else ""
    except Exception:
        return ""


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------


def _extract_player_corpus(
    player_db: Path,
    shared_db: Path,
    metadata_db: Path,
    out_dir: Path,
    limit: int,
) -> dict[str, int]:
    """Extrait les données dans out_dir et retourne les compteurs."""
    out_dir.mkdir(parents=True, exist_ok=True)

    # ----- 1. Lire la DB joueur pour obtenir les match_ids -----
    logger.info("Lecture de la DB joueur : %s", player_db)
    with duckdb.connect(str(player_db), read_only=True) as src_conn:
        xuid = _resolve_xuid(src_conn)
        if not xuid:
            raise ValueError("Impossible de résoudre le xuid depuis sync_meta")
        logger.info("xuid résolu : %s", xuid)

        # Charger shared pour obtenir les matchs
        if shared_db.exists():
            try:
                src_conn.execute(f"ATTACH '{shared_db}' AS shared (READ_ONLY)")
            except Exception as e:
                logger.warning("Impossible d'attacher shared : %s", e)

        # Récupérer les match_ids les plus récents (via mv_player_matches ou join direct)
        try:
            rows = src_conn.execute(
                """
                SELECT match_id FROM shared.mv_player_matches
                WHERE xuid = ?
                ORDER BY start_time DESC
                LIMIT ?
                """,
                [xuid, limit],
            ).fetchall()
        except Exception:
            logger.info("mv_player_matches indisponible, fallback join direct")
            rows = src_conn.execute(
                """
                SELECT r.match_id
                FROM shared.match_registry r
                JOIN shared.match_participants p ON r.match_id = p.match_id
                WHERE p.xuid = ?
                ORDER BY r.start_time DESC
                LIMIT ?
                """,
                [xuid, limit],
            ).fetchall()

    match_ids = [str(r[0]) for r in rows]
    if not match_ids:
        raise ValueError("Aucun match trouvé pour ce joueur")
    logger.info("%d matchs sélectionnés", len(match_ids))

    # Persistance du xuid
    (out_dir / "xuid.txt").write_text(xuid, encoding="utf-8")

    counts: dict[str, int] = {}

    # ----- 2. Créer shared_matches_v2.duckdb (sous-ensemble) -----
    out_shared = out_dir / "shared_matches_v2.duckdb"
    out_shared.unlink(missing_ok=True)
    logger.info("Création de %s ...", out_shared)

    placeholders = ", ".join("?" * len(match_ids))
    with (
        duckdb.connect(str(shared_db), read_only=True) as shared_src,
        duckdb.connect(str(out_shared)) as shared_dst,
    ):
        for table in (
            "match_registry",
            "match_participants",
            "highlight_events",
            "medals_earned",
            "weapon_kills",
        ):
            try:
                df = shared_src.execute(
                    f"SELECT * FROM {table} WHERE match_id IN ({placeholders})",
                    match_ids,
                ).to_arrow_table()
                shared_dst.register("_tmp", df)
                shared_dst.execute(f"CREATE TABLE {table} AS SELECT * FROM _tmp")
                cnt = shared_dst.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                counts[f"shared.{table}"] = cnt
                logger.info("  shared.%s : %d lignes", table, cnt)
            except Exception as e:
                logger.warning("  shared.%s ignorée : %s", table, e)

        # Tables sans match_id : copie complète
        for table in ("xuid_aliases",):
            try:
                df = shared_src.execute(f"SELECT * FROM {table}").to_arrow_table()
                shared_dst.register("_tmp", df)
                shared_dst.execute(f"CREATE TABLE {table} AS SELECT * FROM _tmp")
                cnt = shared_dst.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                counts[f"shared.{table}"] = cnt
                logger.info("  shared.%s : %d lignes", table, cnt)
            except Exception as e:
                logger.warning("  shared.%s ignorée : %s", table, e)

        # Vues v6 essentielles: xuid_aliases + mv_player_matches
        try:
            _create_shared_views(shared_dst)
        except Exception as e:
            logger.warning("Vues shared non créées : %s", e)

    # ----- 3. Créer stats.duckdb (enrichissements joueur) -----
    out_player = out_dir / "stats.duckdb"
    out_player.unlink(missing_ok=True)
    logger.info("Création de %s ...", out_player)

    with (
        duckdb.connect(str(player_db), read_only=True) as player_src,
        duckdb.connect(str(out_player)) as player_dst,
    ):
        for table in (
            "player_match_enrichment",
            "personal_score_awards",
            "match_citations",
        ):
            try:
                df = player_src.execute(
                    f"SELECT * FROM {table} WHERE match_id IN ({placeholders})",
                    match_ids,
                ).to_arrow_table()
                player_dst.register("_tmp", df)
                player_dst.execute(f"CREATE TABLE {table} AS SELECT * FROM _tmp")
                cnt = player_dst.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                counts[f"player.{table}"] = cnt
                logger.info("  player.%s : %d lignes", table, cnt)
            except Exception as e:
                logger.warning("  player.%s ignorée : %s", table, e)

        # Tables sans match_id : copie complète
        for table in ("career_progression", "sessions", "sync_meta"):
            try:
                df = player_src.execute(f"SELECT * FROM {table}").to_arrow_table()
                player_dst.register("_tmp", df)
                player_dst.execute(f"CREATE TABLE {table} AS SELECT * FROM _tmp")
                cnt = player_dst.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                counts[f"player.{table}"] = cnt
                logger.info("  player.%s : %d lignes", table, cnt)
            except Exception as e:
                logger.warning("  player.%s ignorée : %s", table, e)

    # ----- 4. Copier metadata.duckdb (référentiels complets) -----
    out_meta = out_dir / "metadata.duckdb"
    out_meta.unlink(missing_ok=True)
    logger.info("Copie de metadata.duckdb ...")
    import shutil

    shutil.copy2(str(metadata_db), str(out_meta))
    counts["metadata"] = 1

    counts["match_ids"] = len(match_ids)
    return counts


def _create_shared_views(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée les vues v6 dans le shared.duckdb de test."""
    conn.execute("""
        CREATE VIEW IF NOT EXISTS v_gamertag_lookup AS
        SELECT
            xa.xuid,
            COALESCE(xa.gamertag, mp.gamertag) AS gamertag
        FROM xuid_aliases xa
        FULL OUTER JOIN (
            SELECT DISTINCT xuid, gamertag FROM match_participants
        ) mp ON xa.xuid = mp.xuid
    """)
    # mv_player_matches est une vue matérialisée — la recréer comme une vue simple
    conn.execute("""
        CREATE VIEW IF NOT EXISTS mv_player_matches AS
        SELECT
            p.xuid,
            r.match_id,
            r.start_time,
            r.map_id,
            r.map_name,
            NULL AS map_name_fr,
            r.playlist_id,
            r.playlist_name,
            NULL AS playlist_name_fr,
            r.pair_id,
            r.pair_name,
            NULL AS pair_name_fr,
            r.game_variant_id,
            r.game_variant_name,
            NULL AS game_variant_name_fr,
            p.outcome,
            p.team_id,
            p.kda,
            p.kills,
            p.deaths,
            p.assists,
            p.accuracy,
            p.personal_score,
            p.time_played_seconds,
            CASE WHEN p.team_id = 0 THEN r.team_0_score ELSE r.team_1_score END AS my_team_score,
            CASE WHEN p.team_id = 0 THEN r.team_1_score ELSE r.team_0_score END AS enemy_team_score,
            CASE WHEN p.team_id = 0 THEN r.team_0_ps_score ELSE r.team_1_ps_score END AS my_team_ps_score,
            CASE WHEN p.team_id = 0 THEN r.team_1_ps_score ELSE r.team_0_ps_score END AS enemy_team_ps_score,
            COALESCE(r.is_firefight, FALSE) AS is_firefight,
            COALESCE(r.is_ranked, FALSE) AS is_ranked
        FROM match_registry r
        JOIN match_participants p ON r.match_id = p.match_id
    """)


# ---------------------------------------------------------------------------
# Golden values
# ---------------------------------------------------------------------------

GOLDEN_DIR = Path(__file__).parent.parent / "tests" / "fixtures" / "golden_values"


def _generate_golden_values(
    gamertag: str,
    xuid: str,
    stats_db: Path,
    shared_db: Path,
    out_dir: Path | None = None,
) -> None:
    """Génère career.json et match_history_full.json depuis le corpus."""
    from datetime import UTC, datetime

    golden_dir = out_dir if out_dir is not None else GOLDEN_DIR
    golden_dir.mkdir(parents=True, exist_ok=True)

    # -- Career golden values --
    career_gv: dict = {
        "_comment": "Golden values Carrière — générés par create_test_corpus.py.",
        "gamertag": gamertag,
        "generated_at": datetime.now(UTC).isoformat(),
    }
    try:
        with duckdb.connect(str(stats_db), read_only=True) as con:
            row = con.execute(
                "SELECT rank, xp_total FROM career_progression ORDER BY recorded_at DESC LIMIT 1"
            ).fetchone()
            career_gv["rank_number"] = int(row[0]) if row else None
            career_gv["xp_total"] = int(row[1]) if row and row[1] is not None else 0
    except Exception as exc:
        logger.warning("career golden values inaccessibles : %s", exc)
        career_gv["rank_number"] = None
        career_gv["xp_total"] = 0

    career_gv["lusr_rating"] = None
    try:
        with duckdb.connect(str(stats_db), read_only=True) as con:
            row = con.execute(
                "SELECT rating_value FROM match_skill_rank ORDER BY match_id DESC LIMIT 1"
            ).fetchone()
            if row:
                career_gv["lusr_rating"] = float(row[0])
    except Exception:
        pass

    # -- Match history golden values --
    mh_gv: dict = {
        "_comment": "Golden values Match History (scope complet) — générés par create_test_corpus.py.",
        "gamertag": gamertag,
        "generated_at": datetime.now(UTC).isoformat(),
        "total_matches": 0,
        "first_match_id": None,
    }
    if xuid:
        try:
            with duckdb.connect(str(shared_db), read_only=True) as con:
                row = con.execute(
                    "SELECT COUNT(*) FROM match_participants WHERE xuid = ?",
                    [xuid],
                ).fetchone()
                mh_gv["total_matches"] = int(row[0]) if row else 0

                row2 = con.execute(
                    """
                    SELECT mr.match_id
                    FROM match_registry mr
                    JOIN match_participants mp ON mp.match_id = mr.match_id
                    WHERE mp.xuid = ?
                    ORDER BY mr.start_time DESC LIMIT 1
                    """,
                    [xuid],
                ).fetchone()
                mh_gv["first_match_id"] = row2[0] if row2 else None
        except Exception as exc:
            logger.warning("match_history golden values inaccessibles : %s", exc)

    (golden_dir / "career.json").write_text(
        __import__("json").dumps(career_gv, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    (golden_dir / "match_history_full.json").write_text(
        __import__("json").dumps(mh_gv, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    logger.info(
        "Golden values : rang=%s xp=%s matchs=%s",
        career_gv.get("rank_number"),
        career_gv.get("xp_total"),
        mh_gv.get("total_matches"),
    )


# ---------------------------------------------------------------------------
# Point d'entrée
# ---------------------------------------------------------------------------


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Crée le corpus de test pour les tests de parité API."
    )
    parser.add_argument("--gamertag", required=True, help="Gamertag du joueur de référence")
    parser.add_argument(
        "--limit",
        type=int,
        default=DEFAULT_LIMIT,
        help=f"Nombre max de matchs à extraire (défaut: {DEFAULT_LIMIT})",
    )
    parser.add_argument(
        "--out-dir",
        type=Path,
        default=FIXTURES_DIR,
        help=f"Répertoire de sortie (défaut: {FIXTURES_DIR})",
    )
    parser.add_argument(
        "--profiles",
        type=Path,
        default=None,
        help="Chemin vers db_profiles.json (défaut: db_profiles.json à la racine du repo). "
        "Utile pour pointer sur un repo externe.",
    )
    args = parser.parse_args()

    player_db = _resolve_player_db(args.gamertag, args.profiles)
    shared_db = get_shared_matches_path_from_player(str(player_db))
    if shared_db:
        shared_db = Path(shared_db)
    else:
        # Fallback : même répertoire que data/warehouse/
        shared_db = player_db.parent.parent.parent / "warehouse" / "shared_matches_v2.duckdb"

    metadata_db = player_db.parent.parent.parent / "warehouse" / "metadata.duckdb"

    logger.info("=== Extraction du corpus de test ===")
    logger.info("Joueur  : %s", args.gamertag)
    logger.info("Source  : %s", player_db)
    logger.info("Shared  : %s", shared_db)
    logger.info("Meta    : %s", metadata_db)
    logger.info("Sortie  : %s", args.out_dir.resolve())
    logger.info("Limite  : %d matchs", args.limit)

    if not player_db.exists():
        logger.error("DB joueur introuvable : %s", player_db)
        sys.exit(1)
    if not shared_db.exists():
        logger.error("DB partagée introuvable : %s", shared_db)
        sys.exit(1)
    if not metadata_db.exists():
        logger.error("DB metadata introuvable : %s", metadata_db)
        sys.exit(1)

    counts = _extract_player_corpus(player_db, shared_db, metadata_db, args.out_dir, args.limit)

    logger.info("=== Extraction terminée ===")
    for k, v in counts.items():
        logger.info("  %-30s %d", k, v)

    # Génération des golden values
    xuid_file = args.out_dir / "xuid.txt"
    xuid = xuid_file.read_text(encoding="utf-8").strip() if xuid_file.exists() else ""
    _generate_golden_values(
        args.gamertag,
        xuid,
        args.out_dir / "stats.duckdb",
        args.out_dir / "shared_matches_v2.duckdb",
    )

    # Mise à jour du README
    _update_readme(args.out_dir, args.gamertag, args.limit, counts)
    logger.info("README mis à jour dans %s", args.out_dir / "README.md")
    logger.info("Golden values mis à jour dans %s", args.out_dir.parent.parent / "golden_values")


def _update_readme(out_dir: Path, gamertag: str, limit: int, counts: dict[str, int]) -> None:
    """Met à jour le README.md du corpus avec les stats d'extraction."""
    from datetime import date

    lines = [
        "# Corpus de test — Joueur de référence",
        "",
        f"Généré le {date.today()} depuis le gamertag `{gamertag}` (limit={limit}).",
        "",
        "## Contenu",
        "",
        "| Fichier | Description |",
        "|---------|-------------|",
        "| `stats.duckdb` | Enrichissements joueur (`player_match_enrichment`, etc.) |",
        "| `shared_matches_v2.duckdb` | Sous-ensemble shared matches pour ce joueur |",
        "| `metadata.duckdb` | Référentiels complets |",
        "| `xuid.txt` | XUID du joueur |",
        "",
        "## Statistiques d'extraction",
        "",
        "| Table | Lignes |",
        "|-------|--------|",
    ]
    for k, v in counts.items():
        lines.append(f"| `{k}` | {v} |")

    lines += [
        "",
        "## Usage",
        "",
        "Ces fichiers sont utilisés par :",
        "- `tests/api/test_filters.py` (tests schéma filtre avec DB réelle)",
        "- `tests/parity/` (tests de parité API vs Streamlit)",
        "",
        "Pour régénérer :",
        "```bash",
        f"python scripts/create_test_corpus.py --gamertag {gamertag} --limit {limit}",
        "```",
    ]

    (out_dir / "README.md").write_text("\n".join(lines), encoding="utf-8")


if __name__ == "__main__":
    main()
