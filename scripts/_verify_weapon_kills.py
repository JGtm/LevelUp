"""Vérification rapide de l'état de weapon_kills après nettoyage."""

import sys
from pathlib import Path

ROOT = Path(__file__).parent.parent
sys.path.insert(0, str(ROOT))
import duckdb  # noqa: E402

WEAPON_KILLS_BIT = 1 << 21
WEAPON_KILLS_NO_FILM_BIT = 1 << 22

conn = duckdb.connect(
    str(ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"),
    read_only=True,
)

print("=== Statut backfill match_registry ===")
rows = conn.execute(f"""
    SELECT
        CASE WHEN (COALESCE(backfill_completed,0) & {WEAPON_KILLS_BIT}) = 0
                  AND (COALESCE(backfill_completed,0) & {WEAPON_KILLS_NO_FILM_BIT}) = 0
             THEN 'pending'
             ELSE 'done'
        END as status,
        COUNT(*) as n
    FROM match_registry
    GROUP BY 1
""").fetchall()
for row in rows:
    print(f"  {row[0]}: {row[1]} matchs")

print()
print("=== weapon_kills par attribution_path ===")
rows2 = conn.execute("""
    SELECT attribution_path, COUNT(*) as n,
           COUNT(CASE WHEN weapon_id IS NULL THEN 1 END) as wid_null,
           COUNT(CASE WHEN weapon_id IS NOT NULL AND weapon_id NOT IN (1,2,3) THEN 1 END) as wid_real
    FROM weapon_kills
    GROUP BY 1 ORDER BY n DESC
""").fetchall()
for path, total, null, real in rows2:
    print(f"  {path:<15}  total={total:6d}  wid_null={null:5d}  wid_real={real:5d}")

print()
print("=== Anomalies sentinel restantes ===")
# Sentinels légitimes : weapon_id ∈ {0=grenade, 1=melee, 2=vehicle}
# Anomalie réelle = sentinel avec confidence/delta_ms non-neutres
anomalies = conn.execute("""
    SELECT weapon_id, attribution_path, confidence, delta_ms, COUNT(*) as n
    FROM weapon_kills
    WHERE (
        -- Sentinel avec colonnes de corrélation non neutralisées
        weapon_id IN (0, 1, 2)
        AND attribution_path = 'none'
        AND (confidence != 'none' OR delta_ms IS NOT NULL)
    ) OR (
        -- Raw handle : weapon_id > 3 et non résolu, path='none'
        attribution_path = 'none'
        AND weapon_id IS NOT NULL
        AND weapon_id NOT IN (0, 1, 2)
        AND weapon_id > 1000
    )
    GROUP BY 1, 2, 3, 4
    ORDER BY 1
""").fetchall()
if anomalies:
    for row in anomalies:
        print(f"  {row}")
else:
    print("  ✓ Aucune anomalie")

conn.close()
