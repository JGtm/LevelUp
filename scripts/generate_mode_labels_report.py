"""Génère un fichier CSV de contrôle pour la normalisation des labels de modes.

Compare le label actuel (translate_pair_name) avec le label résolu par
resolve_display_mode(), afin de valider les règles avant intégration UI.

Usage :
    python scripts/check_mode_labels.py [--out labels_check.csv] [--lang fr]

Sortie CSV :
    game_variant_name | mode_category | playlist_name | nb_matchs
    | label_actuel | label_normalise | changed
"""

from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

# ── Racine du projet ──────────────────────────────────────────────────────────
_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_ROOT))


def _load_tables(lang: str) -> dict:
    from src.ui.translations import _load_mode_tables  # noqa: PLC2701

    return _load_mode_tables(lang)


def _current_label(game_variant_name: str, lang: str) -> str:
    from src.ui.translations import translate_pair_name

    return translate_pair_name(game_variant_name, lang=lang) or game_variant_name


def _resolved_label(
    game_variant_name: str,
    mode_category: str,
    lang: str,
    tables: dict,
) -> str:
    from src.analysis.mode_display import resolve_display_mode

    return resolve_display_mode(game_variant_name, mode_category, lang, tables)


def _query_variants(shared_db: Path) -> list[tuple]:
    """Retourne tous les (game_variant_name, mode_category, playlist_name, nb) distincts."""
    from src.utils.db import duckdb_read_only

    sql = """
        SELECT
            COALESCE(game_variant_name, '') AS game_variant_name,
            COALESCE(mode_category,     '') AS mode_category,
            COALESCE(playlist_name,     'Sans playlist') AS playlist_name,
            COUNT(*) AS nb
        FROM match_registry
        GROUP BY game_variant_name, mode_category, playlist_name
        ORDER BY mode_category, game_variant_name, playlist_name
    """
    with duckdb_read_only(str(shared_db)) as conn:
        return conn.execute(sql).fetchall()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", default="labels_check.csv", help="Fichier CSV de sortie")
    parser.add_argument("--lang", default="fr", choices=["fr", "en"], help="Langue (défaut: fr)")
    args = parser.parse_args()

    from src.utils.paths import get_shared_matches_path

    shared_db = get_shared_matches_path()
    if not shared_db.exists():
        print("ERREUR : shared_matches DB introuvable.", file=sys.stderr)
        sys.exit(1)

    print(f"Chargement des tables metadata ({args.lang})…")
    tables = _load_tables(args.lang)

    print(f"Requête des variantes depuis {shared_db.name}…")
    rows = _query_variants(shared_db)
    print(f"{len(rows)} combinaisons (game_variant_name × mode_category × playlist) trouvées.")

    out_path = Path(args.out)
    changed = 0

    with out_path.open("w", newline="", encoding="utf-8") as fh:
        writer = csv.writer(fh, delimiter=";")
        writer.writerow(
            [
                "game_variant_name",
                "mode_category",
                "playlist_name",
                "nb_matchs",
                "label_actuel",
                "label_normalise",
                "changed",
            ]
        )
        for gvn, cat, playlist, nb in rows:
            current = _current_label(gvn, args.lang) if gvn else ""
            resolved = _resolved_label(gvn, cat, args.lang, tables) if gvn else ""
            is_changed = current != resolved
            if is_changed:
                changed += 1
            writer.writerow(
                [gvn, cat, playlist, nb, current, resolved, "OUI" if is_changed else ""]
            )

    print(f"\nFichier écrit : {out_path.resolve()}")
    print(f"Labels modifiés : {changed} / {len(rows)}")
    print("\nOuvrez le CSV dans Excel ou un tableur pour valider les changements.")
    print("Corrections -> ajouter des entrees dans mode_pair_overrides (metadata.duckdb).")


if __name__ == "__main__":
    main()
