"""Chargement de fichiers .env et normalisation de gamertags pour les variables d'env.

Centralise ``_load_dotenv_if_present`` et ``_normalize_gamertag_for_env``
utilisés dans ``api_client.py`` et ``profile_api_tokens.py``.
"""

from __future__ import annotations

import os
import re
from pathlib import Path


def _repo_root() -> Path:
    """Retourne la racine du repository (3 niveaux au-dessus de utils/)."""
    return Path(__file__).resolve().parents[2]


def load_dotenv_if_present() -> None:
    """Charge ``.env.local`` puis ``.env`` à la racine du repo si présents.

    Règles :
    - Lignes ``KEY=VALUE`` (commentaires ``#`` ignorés).
    - Ne remplace pas une variable déjà définie dans l'environnement.
    """
    root = _repo_root()
    for name in (".env.local", ".env"):
        dotenv_path = root / name
        if not dotenv_path.exists():
            continue
        try:
            content = dotenv_path.read_text(encoding="utf-8")
        except Exception:  # noqa: BLE001
            continue

        for raw_line in content.splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" not in line:
                continue
            key, value = line.split("=", 1)
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            if not key:
                continue
            if os.environ.get(key) is None:
                os.environ[key] = value


def normalize_gamertag_for_env(gamertag: str) -> str:
    """Normalise un gamertag en clé d'env valide.

    Transforme en majuscules et remplace tout caractère non alphanumérique
    par un underscore.

    Exemples::

        "SpartanC"    → "SPARTANC"
        "Mon GT 2"    → "MON_GT_2"
        "Spartan#42"  → "SPARTAN_42"
    """
    return re.sub(r"[^A-Za-z0-9]", "_", gamertag.strip()).upper()


def get_player_token_env_key(gamertag: str) -> str:
    """Retourne le nom de la variable d'env du refresh token propre au joueur.

    Args:
        gamertag: Gamertag du joueur.

    Returns:
        Nom de la variable ``SPNKR_OAUTH_REFRESH_TOKEN_<GT_NORMALISÉ>``.
    """
    return f"SPNKR_OAUTH_REFRESH_TOKEN_{normalize_gamertag_for_env(gamertag)}"
