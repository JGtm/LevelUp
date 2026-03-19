#!/usr/bin/env python3
"""Peuple la table medal_definitions dans metadata.duckdb depuis les JSON.

Lit les fichiers ``static/medals/medals_{lang}.json`` et
``static/medals/medals_descriptions_{lang}.json`` pour insérer ou mettre à
jour les labels et descriptions FR/EN de chaque médaille dans la table
``medal_definitions`` de ``metadata.duckdb``.

Usage:
    python scripts/populate_medal_metadata.py [--dry-run] [--force] [--verbose]
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
from pathlib import Path

logger = logging.getLogger("levelup.medals")

REPO_ROOT = Path(__file__).resolve().parent.parent
MEDALS_DIR = REPO_ROOT / "static" / "medals"
METADATA_DB = REPO_ROOT / "data" / "warehouse" / "metadata.duckdb"

_CUSTOM_MEDAL_THRESHOLD = 9_000_000_000


def _load_json_map(path: Path) -> dict[str, str]:
    """Charge un JSON ``{id_str: label}`` en dict."""
    if not path.exists():
        logger.warning("Fichier introuvable : %s", path)
        return {}
    with open(path, encoding="utf-8") as f:
        raw = json.load(f)
    if not isinstance(raw, dict):
        return {}
    return {str(k): v for k, v in raw.items() if isinstance(v, str) and v.strip()}


def build_medal_rows() -> list[tuple[int, str, str, str, str, bool]]:
    """Construit les lignes à insérer depuis les 4 fichiers JSON.

    Returns:
        Liste de (medal_name_id, name_fr, name_en, description_fr, description_en, is_custom).
    """
    fr_labels = _load_json_map(MEDALS_DIR / "medals_fr.json")
    en_labels = _load_json_map(MEDALS_DIR / "medals_en.json")
    fr_desc = _load_json_map(MEDALS_DIR / "medals_descriptions_fr.json")
    en_desc = _load_json_map(MEDALS_DIR / "medals_descriptions_en.json")

    all_ids = sorted(set(fr_labels) | set(en_labels), key=lambda x: int(x))

    rows: list[tuple[int, str, str, str, str, bool]] = []
    for id_str in all_ids:
        medal_id = int(id_str)
        name_fr = fr_labels.get(id_str, "")
        name_en = en_labels.get(id_str, "")

        if not name_fr and not name_en:
            continue

        if id_str in fr_labels and id_str not in en_labels:
            logger.warning("Médaille %s présente en FR mais absente de EN", id_str)
        elif id_str in en_labels and id_str not in fr_labels:
            logger.warning("Médaille %s présente en EN mais absente de FR", id_str)

        # Fallback croisé pour le name
        if not name_fr:
            name_fr = name_en
        if not name_en:
            name_en = name_fr

        desc_fr = fr_desc.get(id_str, "")
        desc_en = en_desc.get(id_str, "")
        is_custom = medal_id >= _CUSTOM_MEDAL_THRESHOLD

        rows.append((medal_id, name_fr, name_en, desc_fr, desc_en, is_custom))

    return rows


def populate_medal_definitions(
    *,
    dry_run: bool = False,
    force: bool = False,
) -> int:
    """Insère ou met à jour les définitions de médailles dans metadata.duckdb.

    Args:
        dry_run: Si True, ne fait aucune écriture.
        force: Si True, écrase les entrées existantes (INSERT OR REPLACE).

    Returns:
        Nombre de lignes insérées ou mises à jour.
    """
    import duckdb

    from src.data.sync.migrations import ensure_medal_definitions_table

    rows = build_medal_rows()
    if not rows:
        logger.error("Aucune médaille trouvée dans les fichiers JSON")
        return 0

    logger.info("%d médailles trouvées dans les fichiers JSON", len(rows))

    if dry_run:
        custom_count = sum(1 for r in rows if r[5])
        logger.info(
            "[DRY-RUN] %d médailles à insérer (%d custom), aucune écriture",
            len(rows),
            custom_count,
        )
        return 0

    conn = duckdb.connect(str(METADATA_DB), read_only=False)
    try:
        ensure_medal_definitions_table(conn)

        if force:
            sql = (
                "INSERT OR REPLACE INTO medal_definitions "
                "(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom) "
                "VALUES (?, ?, ?, ?, ?, ?)"
            )
        else:
            sql = (
                "INSERT OR IGNORE INTO medal_definitions "
                "(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom) "
                "VALUES (?, ?, ?, ?, ?, ?)"
            )

        conn.executemany(sql, rows)
        count = conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
        conn.commit()
        logger.info(
            "%d médailles insérées/mises à jour (total en DB : %d)",
            len(rows),
            count,
        )
        return len(rows)
    except Exception:
        logger.exception("Erreur lors de la population de medal_definitions")
        return 0
    finally:
        conn.close()


def main() -> int:
    """Point d'entrée CLI."""
    parser = argparse.ArgumentParser(
        description="Peuple medal_definitions dans metadata.duckdb",
    )
    parser.add_argument("--dry-run", action="store_true", help="Mode lecture seule")
    parser.add_argument("--force", action="store_true", help="Écrase les entrées existantes")
    parser.add_argument("--verbose", action="store_true", help="Logs détaillés")
    args = parser.parse_args()

    level = logging.DEBUG if args.verbose else logging.INFO
    logging.basicConfig(level=level, format="%(levelname)s - %(name)s - %(message)s")

    n = populate_medal_definitions(dry_run=args.dry_run, force=args.force)

    if not args.dry_run:
        count = 0
        try:
            import duckdb

            conn = duckdb.connect(str(METADATA_DB), read_only=True)
            try:
                count = conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
            finally:
                conn.close()
        except Exception:
            pass
        if count < 167:
            logger.warning(
                "Validation : seulement %d médailles en DB (attendu >= 167)",
                count,
            )
        else:
            logger.info("Validation : %d médailles en DB ✓", count)

    return 0 if n > 0 or args.dry_run else 1


if __name__ == "__main__":
    sys.exit(main())
