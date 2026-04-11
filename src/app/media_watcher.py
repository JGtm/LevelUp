"""Watcher filesystem inotify pour l'indexation automatique des médias (Linux).

Sur Linux, watchdog utilise inotify (events kernel) — zéro CPU en veille.
Sur Windows, ce module n'est pas chargé (branchement dans media_background.py).

Logique :
  - Observer watchdog sur media_captures_base_dir/ (récursif)
  - Debounce par gamertag pour grouper les events d'une même copie de fichier
  - Scan one-shot au démarrage (rattrapage des fichiers copiés hors app)
"""

from __future__ import annotations

import logging
import threading
from pathlib import Path

from src.data.media_helpers import is_generated_thumbnail_path

logger = logging.getLogger(__name__)

_WATCHED_EXTENSIONS = {
    ".mp4",
    ".mkv",
    ".mov",
    ".avi",
    ".webm",
    ".jpg",
    ".jpeg",
    ".png",
    ".gif",
}


def _gamertag_from_path(file_path: str, base_path: Path) -> str | None:
    """Extrait le gamertag (premier composant sous base_path)."""
    try:
        rel = Path(file_path).relative_to(base_path)
        return rel.parts[0] if rel.parts else None
    except ValueError:
        return None


def _run_initial_scan(base_path: Path, tolerance: int) -> None:
    """Scan one-shot au démarrage pour rattraper les fichiers copiés hors app."""
    from src.app.media_background import _index_all_players

    try:
        _index_all_players(base_path, tolerance)
        logger.info("✅ Watcher: scan initial terminé")
    except Exception as e:
        logger.exception("❌ Watcher: erreur scan initial: %s", e)


class _MediaEventHandler:
    """Handler watchdog : détecte les fichiers media et déclenche l'indexation avec debounce.

    Pas de sous-classage de FileSystemEventHandler au niveau module pour éviter
    l'import de watchdog au chargement du module (import lazy dans start_media_watcher).
    La méthode dispatch() est appelée directement par l'Observer watchdog.
    """

    def __init__(self, base_path: Path, settings, debounce_s: int) -> None:
        self._base_path = base_path
        self._settings = settings
        self._debounce_s = debounce_s
        self._timers: dict[str, threading.Timer] = {}
        self._lock = threading.Lock()

    def dispatch(self, event) -> None:  # watchdog callback universel
        """Filtre les events créations/déplacements de fichiers media."""
        if getattr(event, "is_directory", False):
            return
        src = getattr(event, "dest_path", None) or getattr(event, "src_path", None)
        if not src:
            return
        event_type = getattr(event, "event_type", "")
        if event_type not in ("created", "moved"):
            return
        if is_generated_thumbnail_path(src):
            return
        if Path(src).suffix.lower() not in _WATCHED_EXTENSIONS:
            return
        gamertag = _gamertag_from_path(src, self._base_path)
        if gamertag:
            self._schedule(gamertag)

    def _schedule(self, gamertag: str) -> None:
        """Planifie l'indexation avec debounce (reset si nouvel event avant expiration)."""
        with self._lock:
            existing = self._timers.get(gamertag)
            if existing:
                existing.cancel()
            timer = threading.Timer(self._debounce_s, self._run_index, args=[gamertag])
            self._timers[gamertag] = timer
            timer.start()
        logger.debug("⏱️ Watcher: debounce %ds pour %s", self._debounce_s, gamertag)

    def _run_index(self, gamertag: str) -> None:
        """Indexe un joueur après expiration du debounce."""
        from src.app.media_background import _index_with_retry
        from src.utils.paths import PLAYER_DB_FILENAME, PLAYERS_DIR

        with self._lock:
            self._timers.pop(gamertag, None)

        db_file = PLAYERS_DIR / gamertag / PLAYER_DB_FILENAME
        captures = self._base_path / gamertag
        if not db_file.exists() or not captures.exists():
            logger.debug("Watcher: %s ignoré (db ou captures manquants)", gamertag)
            return

        logger.info("📁 Watcher: nouveaux fichiers détectés → indexation %s", gamertag)
        _index_with_retry(db_file, gamertag, captures, self._settings.media_tolerance_minutes)


def start_media_watcher(base_path: Path, settings) -> None:
    """Démarre l'Observer inotify + lance un scan initial one-shot.

    L'Observer watchdog tourne en daemon thread (arrêt automatique avec l'app).
    Le scan initial tourne dans un thread séparé pour ne pas bloquer le démarrage.
    """
    try:
        from watchdog.observers import Observer
    except ImportError:
        logger.warning("watchdog non installé — watcher inotify désactivé")
        return

    debounce_s = int(getattr(settings, "media_watcher_debounce_seconds", 5))
    handler = _MediaEventHandler(base_path, settings, debounce_s)
    observer = Observer()
    observer.schedule(handler, str(base_path), recursive=True)
    observer.daemon = True  # type: ignore[assignment]
    observer.start()

    logger.info("👁️ Watcher inotify démarré sur %s (debounce=%ds)", base_path, debounce_s)

    threading.Thread(
        target=_run_initial_scan,
        args=[base_path, settings.media_tolerance_minutes],
        daemon=True,
        name="media-watcher-initial-scan",
    ).start()
