"""Utilitaires de manipulation de chaînes de caractères."""

from __future__ import annotations

import re

_UUID_LIKE_RE = re.compile(
    r"^[a-f0-9]{8}(-[a-f0-9]{4}){0,3}(-[a-f0-9]{1,12})?$",
    re.IGNORECASE,
)


def is_uuid_like(s: str) -> bool:
    """Vérifie si une chaîne ressemble à un UUID (ex: a446725e-b281-414c-a21e)."""
    return bool(_UUID_LIKE_RE.match(s.lower()))
