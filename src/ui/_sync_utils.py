"""Fonctions utilitaires pour la synchronisation.

Contient les helpers de détection de DB, nettoyage de fichiers temporaires,
résolution de chemins et métadonnées de sync.
"""

from __future__ import annotations

import logging
import time as time_module
from pathlib import Path
from typing import TYPE_CHECKING

import streamlit as st

from src.utils.paths import REPO_ROOT

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


def pick_latest_spnkr_db_if_any(repo_root: Path | None = None) -> str:
    """Sélectionne la base SPNKr la plus récente dans data/.

    Args:
        repo_root: Racine du repo (déduit automatiquement si None).

    Returns:
        Chemin de la DB ou chaîne vide si aucune trouvée.
    """
    try:
        if repo_root is None:
            repo_root = REPO_ROOT
        data_dir = repo_root / "data"
        if not data_dir.exists():
            return ""
        candidates = [p for p in data_dir.glob("spnkr*.db") if p.is_file()]
        if not candidates:
            return ""
        # On évite de sélectionner une DB vide (0 octet), ce qui bloque l'app (aucune table).
        non_empty = [p for p in candidates if p.exists() and p.stat().st_size > 0]
        non_empty.sort(key=lambda p: p.stat().st_mtime if p.exists() else 0.0, reverse=True)
        if non_empty:
            return str(non_empty[0])
        # Fallback: si tout est vide, retourne quand même la plus récente pour debug.
        candidates.sort(key=lambda p: p.stat().st_mtime if p.exists() else 0.0, reverse=True)
        return str(candidates[0])
    except Exception:
        return ""


def is_spnkr_db_path(db_path: str) -> bool:
    """Vérifie si un chemin correspond à une base DuckDB valide.

    Seul DuckDB (.duckdb) est supporté. SQLite (.db) est refusé.
    """
    try:
        p = Path(db_path)
        suffix = p.suffix.lower()
        # SQLite (.db) refusé — DuckDB uniquement
        return suffix == ".duckdb"
    except Exception:
        return False


def cleanup_orphan_tmp_dbs(repo_root: Path | None = None) -> None:  # noqa: PLR0912
    """Nettoie les fichiers .tmp.*.db orphelins dans le dossier data/.

    Ces fichiers peuvent rester si un import SPNKr a été interrompu
    (crash, timeout, fermeture de l'app). On supprime ceux de plus de 1h.

    Args:
        repo_root: Racine du repo (déduit automatiquement si None).
    """
    if st.session_state.get("_tmp_db_cleanup_done"):
        return
    st.session_state["_tmp_db_cleanup_done"] = True

    try:
        if repo_root is None:
            repo_root = REPO_ROOT
        data_dir = repo_root / "data"
        if not data_dir.exists():
            return

        now = time_module.time()
        one_hour_ago = now - 3600  # 1 heure

        # Pattern: *.tmp.*.db (ex: spnkr_gt_SpartanA.db.tmp.1234567890.12345.db)
        for tmp_file in data_dir.glob("*.tmp.*.db"):
            try:
                if tmp_file.stat().st_mtime < one_hour_ago:
                    tmp_file.unlink()
            except Exception:
                pass

        # Pattern alternatif: *.db.tmp.* sans extension finale
        for tmp_file in data_dir.glob("*.db.tmp.*"):
            try:
                if tmp_file.stat().st_mtime < one_hour_ago:
                    tmp_file.unlink()
            except Exception:
                pass
    except Exception:
        pass


def _get_sync_metadata_smart(db_path: str, xuid: str | None = None) -> dict:
    """Récupère les métadonnées de sync selon le type de base."""
    # DuckDB v4 : utiliser le repository
    if db_path.endswith(".duckdb"):
        try:
            from src.data.repositories.duckdb_repo import DuckDBRepository

            repo = DuckDBRepository(db_path, str(xuid or "").strip() or "unknown")
            return repo.get_sync_metadata()
        except Exception:
            return {"last_sync_at": None, "total_matches": 0}

    # SQLite (.db) interdit - retourner métadonnées vides
    return {"last_sync_at": None, "total_matches": 0, "last_match_time": None, "player_xuid": None}


def _shared_path(player_db: Path) -> Path:
    """Résout le chemin vers shared_matches.duckdb depuis le chemin joueur."""
    from src.utils.paths import get_shared_matches_path_from_player

    return (
        get_shared_matches_path_from_player(str(player_db))
        or player_db.parent.parent.parent / "warehouse" / "shared_matches.duckdb"
    )


def get_player_duckdb_path(gamertag: str, repo_root: Path | None = None) -> Path | None:
    """Retourne le chemin vers stats.duckdb d'un joueur si existant.

    Args:
        gamertag: Gamertag du joueur.
        repo_root: Racine du repo (déduit automatiquement si None).

    Returns:
        Path vers stats.duckdb ou None si non trouvé.
    """
    if repo_root is None:
        repo_root = REPO_ROOT

    stats_db = repo_root / "data" / "players" / gamertag / "stats.duckdb"

    if stats_db.exists():
        return stats_db
    return None


def is_duckdb_player(gamertag: str, repo_root: Path | None = None) -> bool:
    """Vérifie si un joueur utilise l'architecture DuckDB v4.

    Args:
        gamertag: Gamertag du joueur.
        repo_root: Racine du repo.

    Returns:
        True si le joueur a une DB DuckDB.
    """
    return get_player_duckdb_path(gamertag, repo_root) is not None


def _summarize_sync_results(results: list[tuple[str, bool, str]]) -> tuple[bool, str]:
    """Résume les résultats de sync multi-joueurs.

    Args:
        results: Liste de (label, succès, message) par joueur.

    Returns:
        Tuple (succès_global, message_résumé).
    """
    if not results:
        return False, "Aucun joueur à synchroniser."

    success_count = sum(1 for _, ok, _ in results if ok)
    total = len(results)

    if success_count == total:
        return (
            True,
            f"✅ {total} joueur{'s' if total > 1 else ''} synchronisé{'s' if total > 1 else ''}.",
        )
    elif success_count > 0:
        failed = [label for label, ok, _ in results if not ok]
        return True, f"⚠️ {success_count}/{total} OK. Échec: {', '.join(failed)}"
    else:
        errors = [f"{label}: {msg}" for label, ok, msg in results if not ok]
        return False, "❌ Échec pour tous les joueurs.\n" + "\n".join(errors[:3])
