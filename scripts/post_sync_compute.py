"""Pré-calcul post-sync des agrégats pour accélérer le chargement UI.

Ce script est appelé automatiquement à la fin de chaque synchronisation
réussie (via ``SyncEngine.refresh_aggregates``). Il pré-calcule et
persiste des tables d'agrégats dans la DB joueur (``stats.duckdb``)
afin que la première visite de chaque page Streamlit soit quasi-instantanée.

Tables créées/rafraîchies :
    - ``precomputed_sessions``     : Mapping match → session_id / session_label
    - ``precomputed_kda_trend``    : Moyennes mobiles KDA/kills/accuracy (fenêtre 20)
    - ``precomputed_global_stats`` : Agrégats globaux (total matchs, KDA moyen, etc.)

Usage standalone :
    python scripts/post_sync_compute.py --gamertag MonGT

Sprint 8ter.4 — Pré-calcul post-sync.
"""

from __future__ import annotations

import argparse
import logging
import sys
import time
from pathlib import Path

# Ajouter la racine du projet au path
_ROOT = Path(__file__).resolve().parent.parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

import duckdb

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Constantes
# ---------------------------------------------------------------------------

_DEFAULT_GAP_MINUTES = 60
_KDA_WINDOW = 20

# ---------------------------------------------------------------------------
# Fonctions de pré-calcul
# ---------------------------------------------------------------------------


def _precompute_sessions(
    conn: duckdb.DuckDBPyConnection,
    gap_minutes: int = _DEFAULT_GAP_MINUTES,
) -> int:
    """Pré-calcule les sessions et les persiste.

    Utilise la logique de gap temporel pour découper les matchs en sessions.

    Returns:
        Nombre de lignes insérées.
    """
    conn.execute(f"""
        CREATE OR REPLACE TABLE precomputed_sessions AS
        WITH ordered AS (
            SELECT
                match_id,
                start_time,
                teammates_signature,
                ROW_NUMBER() OVER (ORDER BY start_time) AS rn,
                COALESCE(
                    EPOCH(start_time - LAG(start_time) OVER (ORDER BY start_time)) / 60,
                    {gap_minutes + 1}
                ) AS gap_min
            FROM match_stats
            WHERE start_time IS NOT NULL
        ),
        with_session AS (
            SELECT
                *,
                SUM(CASE WHEN gap_min > {gap_minutes} THEN 1 ELSE 0 END)
                    OVER (ORDER BY start_time) AS session_num
            FROM ordered
        )
        SELECT
            match_id,
            start_time,
            'S' || LPAD(CAST(session_num + 1 AS VARCHAR), 4, '0') AS session_id,
            'Session ' || CAST(session_num + 1 AS VARCHAR) AS session_label
        FROM with_session
        ORDER BY start_time
    """)

    result = conn.execute("SELECT COUNT(*) FROM precomputed_sessions").fetchone()
    return result[0] if result else 0


def _precompute_kda_trend(
    conn: duckdb.DuckDBPyConnection,
    window: int = _KDA_WINDOW,
) -> int:
    """Pré-calcule les moyennes mobiles KDA et précision.

    Returns:
        Nombre de lignes insérées.
    """
    conn.execute(f"""
        CREATE OR REPLACE TABLE precomputed_kda_trend AS
        SELECT
            match_id,
            start_time,
            kda,
            kills,
            deaths,
            assists,
            accuracy,
            AVG(kda) OVER w AS kda_ma{window},
            AVG(kills) OVER w AS kills_ma{window},
            AVG(deaths) OVER w AS deaths_ma{window},
            AVG(accuracy) OVER w AS acc_ma{window}
        FROM match_stats
        WHERE start_time IS NOT NULL
        WINDOW w AS (ORDER BY start_time ROWS BETWEEN {window - 1} PRECEDING AND CURRENT ROW)
        ORDER BY start_time
    """)

    result = conn.execute("SELECT COUNT(*) FROM precomputed_kda_trend").fetchone()
    return result[0] if result else 0


def _precompute_global_stats(conn: duckdb.DuckDBPyConnection) -> int:
    """Pré-calcule les statistiques globales du joueur.

    Returns:
        1 si réussi, 0 sinon.
    """
    conn.execute("""
        CREATE OR REPLACE TABLE precomputed_global_stats AS
        SELECT
            COUNT(*)                                    AS total_matches,
            COALESCE(SUM(kills), 0)                     AS total_kills,
            COALESCE(SUM(deaths), 0)                    AS total_deaths,
            COALESCE(SUM(assists), 0)                   AS total_assists,
            COALESCE(SUM(time_played_seconds) / 3600.0, 0) AS total_time_hours,
            COALESCE(AVG(kda), 0)                       AS avg_kda,
            COALESCE(AVG(accuracy), 0)                  AS avg_accuracy,
            SUM(CASE WHEN outcome = 'Win' THEN 1 ELSE 0 END)  AS wins,
            SUM(CASE WHEN outcome = 'Loss' THEN 1 ELSE 0 END) AS losses,
            CASE
                WHEN COUNT(*) > 0
                THEN SUM(CASE WHEN outcome = 'Win' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)
                ELSE 0
            END                                         AS win_rate,
            CASE
                WHEN COUNT(*) > 0
                THEN SUM(CASE WHEN outcome = 'Loss' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)
                ELSE 0
            END                                         AS loss_rate
        FROM match_stats
    """)

    return 1


# ---------------------------------------------------------------------------
# Orchestrateur principal
# ---------------------------------------------------------------------------


def post_sync_compute(
    db_path: str,
    *,
    gap_minutes: int = _DEFAULT_GAP_MINUTES,
    kda_window: int = _KDA_WINDOW,
) -> dict[str, int]:
    """Exécute tous les pré-calculs post-sync.

    Args:
        db_path: Chemin vers la DB joueur (``stats.duckdb``).
        gap_minutes: Seuil de gap pour le découpage en sessions.
        kda_window: Taille de la fenêtre pour les moyennes mobiles.

    Returns:
        Dict table → nombre de lignes.
    """
    t0 = time.perf_counter()
    results: dict[str, int] = {}

    conn: duckdb.DuckDBPyConnection | None = None
    try:
        conn = duckdb.connect(str(db_path), read_only=False)

        # Vérifier que match_stats existe
        tables = [
            r[0]
            for r in conn.execute(
                "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
            ).fetchall()
        ]
        if "match_stats" not in tables:
            logger.info("Table match_stats absente — pré-calcul ignoré.")
            return results

        results["precomputed_sessions"] = _precompute_sessions(conn, gap_minutes)
        results["precomputed_kda_trend"] = _precompute_kda_trend(conn, kda_window)
        results["precomputed_global_stats"] = _precompute_global_stats(conn)

        conn.commit()

        elapsed = time.perf_counter() - t0
        logger.info(
            "Post-sync pré-calcul terminé en %.1fms : %s",
            elapsed * 1000,
            results,
        )

    except Exception as e:
        logger.warning("Erreur post_sync_compute : %s", e)
    finally:
        if conn is not None:
            conn.close()

    return results


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def main() -> None:
    """Point d'entrée CLI."""
    parser = argparse.ArgumentParser(description="Pré-calcul post-sync des agrégats.")
    parser.add_argument("--gamertag", required=True, help="Gamertag du joueur")
    parser.add_argument("--gap-minutes", type=int, default=_DEFAULT_GAP_MINUTES)
    parser.add_argument("--kda-window", type=int, default=_KDA_WINDOW)
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")

    db_path = _ROOT / "data" / "players" / args.gamertag / "stats.duckdb"
    if not db_path.exists():
        print(f"[ERREUR] DB joueur introuvable : {db_path}")
        sys.exit(1)

    results = post_sync_compute(
        str(db_path),
        gap_minutes=args.gap_minutes,
        kda_window=args.kda_window,
    )
    for table, count in results.items():
        print(f"  {table}: {count} lignes")
    print("Terminé.")


if __name__ == "__main__":
    main()
