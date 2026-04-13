"""Logging structuré avec structlog — output JSON en prod, coloré en dev."""

from __future__ import annotations

import logging
import sys

import structlog

from apps.api.app.core.config import get_settings


def configure_logging() -> None:
    """Configure structlog + stdlib logging selon l'environnement."""
    settings = get_settings()

    # Niveau de log stdlib partagé
    level = getattr(logging, settings.log_level.upper(), logging.INFO)
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=level,
    )

    shared_processors: list[structlog.types.Processor] = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
    ]

    if settings.log_json:
        # Production : JSON sur stdout (ingestible par Loki, Datadog, etc.)
        processors: list[structlog.types.Processor] = [
            *shared_processors,
            structlog.processors.dict_tracebacks,
            structlog.processors.JSONRenderer(),
        ]
    else:
        # Dev : sortie colorée lisible
        processors = [
            *shared_processors,
            structlog.dev.ConsoleRenderer(),
        ]

    structlog.configure(
        processors=processors,
        wrapper_class=structlog.make_filtering_bound_logger(level),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )


def get_logger(name: str) -> structlog.BoundLogger:
    """Retourne un logger structlog typé."""
    return structlog.get_logger(name)
