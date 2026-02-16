"""Gestion centralisée des chemins pour le projet OpenSpartan Graph.

Ce module définit tous les chemins utilisés par l'architecture v4 (DuckDB unifiée) :
- data/players/{gamertag}/stats.duckdb : DB joueur
- data/players/{gamertag}/archive/ : Archives Parquet du joueur
- data/warehouse/metadata.duckdb : Référentiels partagés
- data/archive/parquet/ : Cold storage global
"""

from __future__ import annotations

import os
from pathlib import Path

# =============================================================================
# Chemins racine
# =============================================================================


def _find_repo_root() -> Path:
    """Trouve la racine du projet (contient pyproject.toml ou .git)."""
    # Essayer depuis le fichier courant
    current = Path(__file__).resolve()
    for parent in current.parents:
        if (parent / "pyproject.toml").exists() or (parent / ".git").exists():
            return parent

    # Fallback : CWD ou variable d'environnement
    if env_root := os.environ.get("OPENSPARTAN_ROOT"):
        return Path(env_root)

    return Path.cwd()


# Racine du projet OpenSpartan Graph
REPO_ROOT: Path = _find_repo_root()

# Dossier des données
DATA_DIR: Path = REPO_ROOT / "data"

# Dossier des données joueurs (architecture v4)
PLAYERS_DIR: Path = DATA_DIR / "players"

# Dossier des référentiels partagés (metadata.duckdb)
WAREHOUSE_DIR: Path = DATA_DIR / "warehouse"

# Dossier des archives globales (cold storage)
ARCHIVE_DIR: Path = DATA_DIR / "archive"


# =============================================================================
# Constantes de noms de fichiers
# =============================================================================

# Nom du fichier DB DuckDB d'un joueur
PLAYER_DB_FILENAME = "stats.duckdb"

# Nom du fichier DB des référentiels partagés
METADATA_DB_FILENAME = "metadata.duckdb"

# Nom du fichier DB des matchs partagés (v5)
SHARED_MATCHES_DB_FILENAME = "shared_matches.duckdb"

# Nom du fichier d'index des archives
ARCHIVE_INDEX_FILENAME = "archive_index.json"


# =============================================================================
# Fonctions utilitaires
# =============================================================================


def get_player_db_path(gamertag: str) -> Path:
    """Retourne le chemin vers la DB DuckDB d'un joueur.

    Args:
        gamertag: Gamertag du joueur.

    Returns:
        Chemin absolu vers data/players/{gamertag}/stats.duckdb
    """
    return PLAYERS_DIR / gamertag / PLAYER_DB_FILENAME


def get_player_archive_dir(gamertag: str) -> Path:
    """Retourne le chemin vers le dossier d'archives d'un joueur.

    Args:
        gamertag: Gamertag du joueur.

    Returns:
        Chemin absolu vers data/players/{gamertag}/archive/
    """
    return PLAYERS_DIR / gamertag / "archive"


def get_metadata_db_path() -> Path:
    """Retourne le chemin vers la DB des métadonnées (metadata.duckdb).

    Returns:
        Chemin absolu vers data/warehouse/metadata.duckdb
    """
    return WAREHOUSE_DIR / METADATA_DB_FILENAME


def get_shared_matches_path() -> Path:
    """Retourne le chemin vers la DB des matchs partagés (shared_matches.duckdb).

    Returns:
        Chemin absolu vers data/warehouse/shared_matches.duckdb
    """
    return WAREHOUSE_DIR / SHARED_MATCHES_DB_FILENAME


def get_shared_matches_path_from_player(player_db_path: str | Path) -> Path | None:
    """Retourne le chemin vers shared_matches.duckdb depuis un path joueur.

    Args:
        player_db_path: Chemin vers une DB joueur (stats.duckdb).

    Returns:
        Chemin vers shared_matches.duckdb ou None si impossible à déduire.
    """
    db_path = Path(player_db_path)
    # data/players/{gamertag}/stats.duckdb -> data/warehouse/shared_matches.duckdb
    if "players" in db_path.parts:
        idx = db_path.parts.index("players")
        data_root = Path(*db_path.parts[:idx])
        shared_path = data_root / "warehouse" / SHARED_MATCHES_DB_FILENAME
        if shared_path.exists():
            return shared_path
    return None


def list_player_gamertags() -> list[str]:
    """Liste tous les gamertags ayant une DB DuckDB.

    Returns:
        Liste triée des gamertags.
    """
    if not PLAYERS_DIR.exists():
        return []

    gamertags = []
    for player_dir in PLAYERS_DIR.iterdir():
        if player_dir.is_dir():
            db_path = player_dir / PLAYER_DB_FILENAME
            if db_path.exists():
                gamertags.append(player_dir.name)

    return sorted(gamertags)


def player_db_exists(gamertag: str) -> bool:
    """Vérifie si la DB DuckDB d'un joueur existe.

    Args:
        gamertag: Gamertag du joueur.

    Returns:
        True si le fichier stats.duckdb existe.
    """
    return get_player_db_path(gamertag).exists()


def ensure_player_dir(gamertag: str) -> Path:
    """Crée le dossier d'un joueur si nécessaire.

    Args:
        gamertag: Gamertag du joueur.

    Returns:
        Chemin vers le dossier créé.
    """
    player_dir = PLAYERS_DIR / gamertag
    player_dir.mkdir(parents=True, exist_ok=True)
    return player_dir


def ensure_archive_dir(gamertag: str) -> Path:
    """Crée le dossier d'archives d'un joueur si nécessaire.

    Args:
        gamertag: Gamertag du joueur.

    Returns:
        Chemin vers le dossier d'archives créé.
    """
    archive_dir = get_player_archive_dir(gamertag)
    archive_dir.mkdir(parents=True, exist_ok=True)
    return archive_dir
