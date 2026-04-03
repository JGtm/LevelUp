"""Gestion des alias XUID -> Gamertag.

Ce module gère les alias XUID->Gamertag depuis shared_matches_v2.duckdb
(table xuid_aliases — source unique de vérité depuis v5.1).

NOTE: Le fichier xuid_aliases.json et les fonctions associées
(load_aliases_file, save_aliases_file) ont été supprimés en v5.2.
Seuls les scripts de migration legacy peuvent encore lire ce JSON.
"""

from __future__ import annotations

import os
from functools import lru_cache

from src.config import XUID_ALIASES_DEFAULT
from src.utils import parse_xuid_input


def _is_duckdb_file(db_path: str) -> bool:
    """Détecte si le fichier est une base DuckDB."""
    return db_path.endswith(".duckdb")


def _safe_mtime(path: str) -> float | None:
    try:
        return os.path.getmtime(path)
    except OSError:
        return None


def load_aliases_from_db(db_path: str) -> dict[str, str]:
    """Charge les alias depuis une DB DuckDB.

    Lit la table xuid_aliases si elle existe.

    Args:
        db_path: Chemin vers la DB (doit être .duckdb).

    Returns:
        Dictionnaire {xuid: gamertag}.
    """
    if not db_path or not os.path.exists(db_path):
        return {}

    # SQLite refusé
    if not _is_duckdb_file(db_path):
        return {}

    mtime = _safe_mtime(db_path)
    return dict(_load_aliases_from_duckdb_cached(db_path, mtime))


def _get_shared_metadata_path(db_path: str) -> str | None:
    """Retourne le chemin vers shared_matches_v2.duckdb depuis un chemin de DB joueur.

    NOTE v5.1 : xuid_aliases est centralisée dans shared_matches_v2.duckdb (13 955 rows).
    metadata.duckdb ne contient PAS cette table.
    """
    from src.utils.paths import get_shared_matches_path_from_player

    shared_path = get_shared_matches_path_from_player(db_path)
    if shared_path and shared_path.exists():
        return str(shared_path)
    return None


@lru_cache(maxsize=16)
def _load_aliases_from_duckdb_cached(db_path: str, mtime: float | None) -> dict[str, str]:
    """Version cachée pour DuckDB — lecture depuis shared_matches_v2.duckdb uniquement.

    NOTE v5.1 : Les aliases locaux (stats.duckdb) sont obsolètes.
    Seule shared_matches_v2.duckdb fait foi.
    """
    from src.utils.db import duckdb_read_only

    result: dict[str, str] = {}

    # V5.1 : Lire UNIQUEMENT depuis shared_matches_v2.duckdb (source de vérité)
    shared_path = _get_shared_metadata_path(db_path)
    if shared_path:
        try:
            with duckdb_read_only(shared_path) as con:
                from src.utils.db import has_table

                if has_table(con, "xuid_aliases"):
                    rows = con.execute(
                        "SELECT xuid, gamertag FROM xuid_aliases WHERE gamertag IS NOT NULL AND gamertag != ''"
                    ).fetchall()
                    result = {str(row[0]).strip(): str(row[1]).strip() for row in rows}
        except Exception:
            pass

    return result


def clear_db_aliases_cache() -> None:
    """Invalide le cache des aliases DB (DuckDB uniquement)."""
    _load_aliases_from_duckdb_cached.cache_clear()


def get_xuid_aliases(db_path: str | None = None) -> dict[str, str]:
    """Retourne les alias (DB > défaut).

    .. deprecated::
        OBSOLÈTE pour la résolution de gamertags dans le contexte d'un match.
        Cette fonction lit shared_matches_v2.duckdb via un cache LRU indirect et ignore
        les données fraîches de match_participants / highlight_events.

        **NE PAS UTILISER** dans les vues de match (render_match_scoreboard,
        render_roster_section, etc.).

        Utiliser à la place :
            load_match_gamertags_fn(db_path, match_id, db_key=db_key)
        qui appelle repo.load_match_player_gamertags() — source de vérité v5.1.

        Utilisation encore acceptable : scripts de migration, export CSV,
        contextes hors-match où aucun match_id n'est disponible.

    L'ordre de priorité est :
    1. Table xuid_aliases de shared_matches_v2.duckdb (si db_path fourni)
    2. Constantes XUID_ALIASES_DEFAULT

    Args:
        db_path: Chemin optionnel vers une DB DuckDB joueur.

    Returns:
        Dictionnaire {xuid: gamertag}.
    """
    merged = dict(XUID_ALIASES_DEFAULT)

    # Les aliases de la DB ont la priorité (plus récents)
    if db_path:
        merged.update(load_aliases_from_db(db_path))

    return merged


def display_name_from_xuid(xuid: str, db_path: str | None = None) -> str:
    """Convertit un XUID en nom d'affichage.

    .. deprecated::
        OBSOLÈTE pour la résolution de gamertags dans le contexte d'un match.
        Passé par get_xuid_aliases() → cache LRU → shared_matches_v2.duckdb et ignore
        les données fraîches de match_participants / highlight_events.

        **NE PAS APPELER** depuis les sections de vue de match.

        Utiliser à la place :
            gt_map = load_match_gamertags_fn(db_path, match_id, db_key=db_key)
            gamertag = gt_map.get(xuid)
        (même approche que render_roster_section, qui fonctionne correctement).

        Utilisation encore acceptable : scripts, contextes hors-match.

    Args:
        xuid: XUID du joueur.
        db_path: Chemin optionnel vers une DB DuckDB.

    Returns:
        Gamertag si un alias existe, sinon le XUID tel quel.
    """
    raw = str(xuid or "").strip()
    # SPNKr/OpenSpartan stockent souvent l'identifiant sous forme "xuid(2533...)".
    # Normaliser ici permet aux alias (clés = XUID numérique) de fonctionner partout.
    normalized = parse_xuid_input(raw) or raw
    return get_xuid_aliases(db_path=db_path).get(normalized, normalized)
