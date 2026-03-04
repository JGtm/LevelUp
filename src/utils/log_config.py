"""Configuration centralisée du logging pour LevelUp.

Ce module fournit :
- ``setup_app_logging()`` : configuration Streamlit (console + fichiers rotatifs)
- ``setup_script_logging()`` : configuration CLI (console + optionnel sync.log)
- ``log_duration()`` : context manager pour mesurer les opérations coûteuses
"""

from __future__ import annotations

import logging
import logging.handlers
import threading
import time
from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

LOG_FORMAT = "%(asctime)s [%(levelname)-5s] %(name)-28s | %(message)s"
LOG_DATEFMT = "%Y-%m-%d %H:%M:%S"

APP_LOG_MAX_BYTES = 5 * 1024 * 1024  # 5 Mo
APP_LOG_BACKUP_COUNT = 3
SYNC_LOG_MAX_BYTES = 10 * 1024 * 1024  # 10 Mo
SYNC_LOG_BACKUP_COUNT = 5

_NOISY_LOGGERS: dict[str, int] = {
    "urllib3": logging.WARNING,
    "httpx": logging.WARNING,
    "httpcore": logging.WARNING,
    "aiohttp": logging.WARNING,
    "aiohttp.access": logging.WARNING,
    "streamlit": logging.ERROR,
    "streamlit.runtime.caching": logging.ERROR,
    "streamlit.runtime.caching.cache_data_api": logging.ERROR,
    "watchdog": logging.WARNING,
    "PIL": logging.WARNING,
    "fsevents": logging.WARNING,
}

_logging_configured = threading.Event()


def _get_logs_dir() -> Path:
    from src.utils.paths import DATA_DIR

    logs_dir = DATA_DIR / "logs"
    logs_dir.mkdir(parents=True, exist_ok=True)
    return logs_dir


def setup_app_logging() -> None:
    """Configure le logging pour l'application Streamlit (guard process-level).

    Les logs vont uniquement dans data/logs/ (pas de console — Streamlit).
    """
    if _logging_configured.is_set():
        return
    _logging_configured.set()

    formatter = logging.Formatter(LOG_FORMAT, datefmt=LOG_DATEFMT)
    logs_dir = _get_logs_dir()

    # Fichier app — tous les loggers src/scripts/streamlit_app, niveau DEBUG
    app_file_handler = logging.handlers.RotatingFileHandler(
        logs_dir / "app.log",
        maxBytes=APP_LOG_MAX_BYTES,
        backupCount=APP_LOG_BACKUP_COUNT,
        encoding="utf-8",
    )
    app_file_handler.setLevel(logging.DEBUG)
    app_file_handler.setFormatter(formatter)

    # Fichier sync (séparé pour src.data.sync et scripts.backfill)
    sync_file_handler = logging.handlers.RotatingFileHandler(
        logs_dir / "sync.log",
        maxBytes=SYNC_LOG_MAX_BYTES,
        backupCount=SYNC_LOG_BACKUP_COUNT,
        encoding="utf-8",
    )
    sync_file_handler.setLevel(logging.DEBUG)
    sync_file_handler.setFormatter(formatter)

    # Logger racine app — fichier uniquement (pas de console dans Streamlit)
    for name in ("src", "scripts", "streamlit_app"):
        lg = logging.getLogger(name)
        lg.setLevel(logging.DEBUG)
        lg.addHandler(app_file_handler)

    # Loggers sync/backfill → fichier sync en plus
    for name in ("src.data.sync", "scripts.backfill"):
        logging.getLogger(name).addHandler(sync_file_handler)

    # Silencer les loggers bruyants
    for logger_name, logger_level in _NOISY_LOGGERS.items():
        logging.getLogger(logger_name).setLevel(logger_level)


def setup_script_logging(level: str = "INFO", sync_log: bool = False) -> None:
    """Configure le logging pour un script CLI."""
    if _logging_configured.is_set():
        return
    _logging_configured.set()

    numeric_level = getattr(logging, level.upper(), logging.INFO)
    formatter = logging.Formatter(LOG_FORMAT, datefmt=LOG_DATEFMT)

    console_handler = logging.StreamHandler()
    console_handler.setLevel(numeric_level)
    console_handler.setFormatter(formatter)

    root = logging.getLogger()
    root.setLevel(numeric_level)
    root.addHandler(console_handler)

    if sync_log:
        logs_dir = _get_logs_dir()
        sync_handler = logging.handlers.RotatingFileHandler(
            logs_dir / "sync.log",
            maxBytes=SYNC_LOG_MAX_BYTES,
            backupCount=SYNC_LOG_BACKUP_COUNT,
            encoding="utf-8",
        )
        sync_handler.setLevel(logging.DEBUG)
        sync_handler.setFormatter(formatter)
        root.addHandler(sync_handler)

    for logger_name, logger_level in _NOISY_LOGGERS.items():
        logging.getLogger(logger_name).setLevel(logger_level)


@contextmanager
def log_duration(
    name: str,
    logger: logging.Logger,
    level: int = logging.DEBUG,
    threshold_ms: float = 0.0,
) -> Iterator[None]:
    """Mesure la durée d'une section et la logue si au-dessus du seuil."""
    t0 = time.perf_counter()
    try:
        yield
    finally:
        dt_ms = (time.perf_counter() - t0) * 1000.0
        if dt_ms >= threshold_ms:
            logger.log(level, "%s terminé en %.0f ms", name, dt_ms)
