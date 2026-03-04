"""Provisionnement automatique d'un profil joueur LevelUp.

Utilisé lors de la connexion Xbox OAuth pour créer automatiquement le
dossier et la base de données d'un nouveau joueur.

Séquence :
  1. ``create_player_db(gamertag)`` → crée ``data/players/{gamertag}/stats.duckdb``
     avec la table ``sync_meta`` initialisée.
  2. ``register_player_profile(gamertag, xuid, db_path)`` → ajoute l'entrée
     dans ``db_profiles.json``.

Ces deux opérations sont idempotentes : appelables plusieurs fois sans erreur.
"""

from __future__ import annotations

import logging
from pathlib import Path

logger = logging.getLogger(__name__)

# DDL minimal pour bootstrapper une DB joueur fraîchement créée.
# Le schéma complet est créé lors du premier sync (SYNC_SCHEMA_DDL dans engine.py).
_BOOTSTRAP_DDL = """
CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
"""


def create_player_db(gamertag: str, *, base_dir: str | Path | None = None) -> Path:
    """Crée le dossier + la DB stats.duckdb pour un nouveau joueur.

    L'opération est idempotente : si le dossier/DB existent déjà, retourne
    simplement le chemin sans rien modifier.

    Args:
        gamertag: Gamertag Xbox du joueur (ex: ``JGtm``).
        base_dir: Répertoire ``data/players/``. Si ``None``, déduit depuis
                  la racine du repo (``Path(__file__).parents[3] / "data" / "players"``).

    Returns:
        Chemin absolu vers ``data/players/{gamertag}/stats.duckdb``.
    """
    if base_dir is None:
        repo_root = Path(__file__).resolve().parents[2]
        base_dir = repo_root / "data" / "players"

    player_dir = Path(base_dir) / gamertag
    player_dir.mkdir(parents=True, exist_ok=True)

    db_path = player_dir / "stats.duckdb"

    if not db_path.exists():
        import duckdb

        conn = duckdb.connect(str(db_path))
        try:
            conn.execute(_BOOTSTRAP_DDL)
            conn.execute(
                "INSERT OR REPLACE INTO sync_meta (key, value, updated_at) "
                "VALUES ('gamertag', ?, CURRENT_TIMESTAMP)",
                (gamertag,),
            )
            logger.info("DB joueur créée : %s", db_path)
        finally:
            conn.close()
    else:
        logger.debug("DB joueur déjà existante : %s", db_path)

    return db_path


def register_player_profile(gamertag: str, xuid: str, db_path: str | Path) -> bool:
    """Enregistre le joueur dans ``db_profiles.json``.

    L'opération est idempotente : si le profil existe déjà, il est mis à jour
    avec les nouvelles valeurs (xuid, db_path).

    Args:
        gamertag: Gamertag Xbox du joueur.
        xuid: XUID Xbox Live du joueur (identifiant numérique).
        db_path: Chemin vers ``stats.duckdb`` du joueur.

    Returns:
        ``True`` si la sauvegarde a réussi, ``False`` sinon.
    """
    from src.utils.profiles import load_profiles, save_profiles

    db_str = str(db_path)

    # Normaliser en chemin relatif si possible (meilleure portabilité)
    try:
        repo_root = Path(__file__).resolve().parents[2]
        rel = Path(db_path).relative_to(repo_root)
        db_str = str(rel).replace("\\", "/")
    except ValueError:
        pass  # Chemin hors du repo → garder absolu

    profiles = load_profiles()
    profiles[gamertag] = {
        "db_path": db_str,
        "xuid": str(xuid),
        "waypoint_player": gamertag,
    }

    ok, err = save_profiles(profiles)
    if ok:
        logger.info("Profil joueur enregistré : %s (xuid=%s)", gamertag, xuid)
    else:
        logger.error("Impossible d'enregistrer le profil %s : %s", gamertag, err)

    return ok


def provision_player(gamertag: str, xuid: str, *, base_dir: str | Path | None = None) -> Path:
    """Crée DB + enregistre profil en une seule opération.

    Fonction principale à appeler après un flux OAuth Xbox réussi.

    Args:
        gamertag: Gamertag Xbox du joueur.
        xuid: XUID Xbox Live du joueur.
        base_dir: Répertoire ``data/players/`` (optionnel, déduit du repo).

    Returns:
        Chemin vers ``data/players/{gamertag}/stats.duckdb``.

    Raises:
        RuntimeError: Si la création de la DB ou l'enregistrement du profil échoue.
    """
    db_path = create_player_db(gamertag, base_dir=base_dir)
    ok = register_player_profile(gamertag, xuid, db_path)

    if not ok:
        raise RuntimeError(
            f"Provisionnement joueur '{gamertag}' : "
            f"DB créée ({db_path}) mais enregistrement du profil échoué."
        )

    return db_path
