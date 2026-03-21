"""
Script de nettoyage des anomalies sentinel dans weapon_kills.

Catégorie 1a : weapon_id=0, confidence='none' → weapon_id=NULL
Catégorie 1b : weapon_id=0, confidence='high' → DELETE + reset bit
Catégorie 2  : sentinels {1,2,3} avec confidence/delta_ms/delayed_damage corrompus → normaliser
Catégorie 3  : raw handles (weapon_id >> 32 bits) avec path='none' → DELETE + reset bit
"""

import logging
import sys
from pathlib import Path

# ── Path setup ─────────────────────────────────────────────────────────────
ROOT = Path(__file__).parent.parent
sys.path.insert(0, str(ROOT))

import duckdb  # noqa: E402

logging.basicConfig(level=logging.INFO, format="%(levelname)s  %(message)s")
log = logging.getLogger(__name__)

WEAPON_KILLS_BIT = 1 << 21  # 2_097_152
WEAPON_KILLS_NO_FILM_BIT = 1 << 22  # 4_194_304
BITS_TO_CLEAR = WEAPON_KILLS_BIT | WEAPON_KILLS_NO_FILM_BIT  # 6_291_456

DB_PATH = ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"


def main() -> int:
    if not DB_PATH.exists():
        log.error("DB introuvable : %s", DB_PATH)
        return 1

    log.info("Ouverture en lecture/écriture : %s", DB_PATH)
    conn = duckdb.connect(str(DB_PATH), read_only=False)

    try:
        # ── CAT 1a : weapon_id=0 / confidence='none' → NULL ────────────────
        n1a = conn.execute("""
            SELECT COUNT(*) FROM weapon_kills
            WHERE weapon_id = 0 AND confidence = 'none' AND attribution_path = 'none'
        """).fetchone()[0]
        conn.execute("""
            UPDATE weapon_kills
            SET weapon_id = NULL
            WHERE weapon_id = 0
              AND confidence = 'none'
              AND attribution_path = 'none'
        """)
        log.info("[CAT1a] weapon_id=0 confidence='none' → NULL : %d lignes", n1a)

        # ── CAT 1b : weapon_id=0 / confidence='high' → DELETE + reset bit ──
        bad_zero_ids = [
            r[0]
            for r in conn.execute("""
                SELECT DISTINCT match_id FROM weapon_kills
                WHERE weapon_id = 0 AND confidence = 'high'
            """).fetchall()
        ]
        log.info("[CAT1b] matchs avec weapon_id=0/high : %d", len(bad_zero_ids))

        n1b_rows = conn.execute("""
            SELECT COUNT(*) FROM weapon_kills
            WHERE weapon_id = 0 AND confidence = 'high'
        """).fetchone()[0]
        conn.execute("""
            DELETE FROM weapon_kills
            WHERE weapon_id = 0 AND confidence = 'high'
        """)
        log.info("[CAT1b] weapon_id=0 confidence='high' → DELETE : %d lignes", n1b_rows)

        # ── CAT 2 : sentinels {1,2,3} corrompus → normaliser ────────────────
        n2 = conn.execute("""
            SELECT COUNT(*) FROM weapon_kills
            WHERE weapon_id IN (1, 2, 3)
              AND attribution_path = 'none'
              AND (confidence != 'none' OR delta_ms IS NOT NULL)
        """).fetchone()[0]
        conn.execute("""
            UPDATE weapon_kills
            SET confidence     = 'none',
                delta_ms       = NULL,
                delayed_damage = FALSE,
                swap_detected  = FALSE
            WHERE weapon_id IN (1, 2, 3)
              AND attribution_path = 'none'
              AND (confidence != 'none' OR delta_ms IS NOT NULL)
        """)
        log.info("[CAT2]  sentinels 1/2/3 corrompus → normalisés : %d lignes", n2)

        # ── CAT 3 : raw handles → DELETE + reset bit ────────────────────────
        raw_handle_ids = [
            r[0]
            for r in conn.execute("""
                SELECT DISTINCT match_id FROM weapon_kills
                WHERE attribution_path = 'none'
                  AND weapon_id IS NOT NULL
                  AND weapon_id NOT IN (0, 1, 2, 3)
            """).fetchall()
        ]
        log.info("[CAT3]  matchs avec raw handles : %d", len(raw_handle_ids))

        n3 = conn.execute("""
            SELECT COUNT(*) FROM weapon_kills
            WHERE attribution_path = 'none'
              AND weapon_id IS NOT NULL
              AND weapon_id NOT IN (0, 1, 2, 3)
        """).fetchone()[0]
        conn.execute("""
            DELETE FROM weapon_kills
            WHERE attribution_path = 'none'
              AND weapon_id IS NOT NULL
              AND weapon_id NOT IN (0, 1, 2, 3)
        """)
        log.info(
            "[CAT3]  raw handles → DELETE : %d lignes (depuis %d matchs)", n3, len(raw_handle_ids)
        )

        # ── RESET bits backfill_completed ────────────────────────────────────
        all_to_reset = list(set(bad_zero_ids + raw_handle_ids))
        log.info("Total matchs à resetter : %d", len(all_to_reset))

        if all_to_reset:
            conn.execute("CREATE OR REPLACE TEMP TABLE _reset_ids (match_id VARCHAR)")
            conn.executemany("INSERT INTO _reset_ids VALUES (?)", [(m,) for m in all_to_reset])
            conn.execute(f"""
                UPDATE match_registry
                SET backfill_completed = COALESCE(backfill_completed, 0) & ~{BITS_TO_CLEAR}
                WHERE match_id IN (SELECT match_id FROM _reset_ids)
            """)
            # Vérifier combien de matchs ont été affectés
            n_reset = conn.execute("""
                SELECT COUNT(*) FROM match_registry
                WHERE match_id IN (SELECT match_id FROM _reset_ids)
            """).fetchone()[0]
            log.info("Bits WEAPON_KILLS + NO_FILM effacés : %d matchs", n_reset)
            conn.execute("DROP TABLE _reset_ids")

        conn.commit()
        log.info("Commit OK")

        # ── Vérification finale ──────────────────────────────────────────────
        print()
        log.info("=== Vérification post-fix ===")

        anomalies = conn.execute("""
            SELECT
                CASE
                    WHEN weapon_id IS NULL THEN 'NULL'
                    WHEN weapon_id BETWEEN 0 AND 9 THEN CAST(weapon_id AS VARCHAR)
                    ELSE 'raw_handle'
                END as wtype,
                attribution_path,
                COUNT(*) as n
            FROM weapon_kills
            WHERE weapon_id = 0
               OR (weapon_id IN (1,2,3) AND attribution_path='none'
                   AND (confidence != 'none' OR delta_ms IS NOT NULL))
               OR (attribution_path = 'none'
                   AND weapon_id IS NOT NULL
                   AND weapon_id NOT IN (0,1,2,3))
            GROUP BY 1, 2
            ORDER BY 1
        """).fetchall()

        if anomalies:
            log.warning("Anomalies restantes : %s", anomalies)
        else:
            log.info("✓ Aucune anomalie restante")

        stats = conn.execute("""
            SELECT attribution_path, COUNT(*) as total,
                   COUNT(CASE WHEN weapon_id IS NULL THEN 1 END) as wid_null
            FROM weapon_kills
            GROUP BY 1 ORDER BY total DESC
        """).fetchall()
        for path, total, wid_null in stats:
            log.info("  path=%-15s  total=%6d  wid_null=%d", path, total, wid_null)

    finally:
        conn.close()

    return 0


if __name__ == "__main__":
    sys.exit(main())
