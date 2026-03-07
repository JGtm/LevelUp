"""Chargement de fichiers .env et normalisation de gamertags pour les variables d'env.

Centralise ``_load_dotenv_if_present`` et ``_normalize_gamertag_for_env``
utilisés dans ``api_client.py`` et ``profile_api_tokens.py``.
"""

from __future__ import annotations

import os
import re

from src.utils.paths import DATA_DIR, REPO_ROOT


def load_dotenv_if_present() -> None:
    """Charge ``.env.local`` puis ``.env`` depuis DATA_DIR et REPO_ROOT.

    Règles :
    - Lignes ``KEY=VALUE`` (commentaires ``#`` ignorés).
    - Ne remplace pas une variable déjà définie dans l'environnement.
    - Cherche d'abord dans DATA_DIR (portable: %APPDATA%/LevelUp/), puis REPO_ROOT.
    """
    search_roots = [DATA_DIR, REPO_ROOT] if DATA_DIR != REPO_ROOT / "data" else [REPO_ROOT]
    for root in search_roots:
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
