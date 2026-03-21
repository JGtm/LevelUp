"""Fragments SQL partagés pour les requêtes analytiques."""

from src.data.query._sql_fragments import IS_LOSS, IS_WIN, WIN_RATE_EXPR

__all__ = [
    "IS_WIN",
    "IS_LOSS",
    "WIN_RATE_EXPR",
]
