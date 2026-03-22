"""Abstraction de l'authentification OAuth pour SPNKr.

Ce module fournit une interface stable pour l'acquisition de tokens,
indépendante du flow OAuth sous-jacent (Azure AD aujourd'hui, Xbox OAuth demain).

Usage :
    from src.utils.auth import check_credentials, get_auth_status

    status = get_auth_status()
    if not status.has_credentials:
        print("Credentials manquantes")
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass, field
from pathlib import Path

from src.utils.env import load_dotenv_if_present
from src.utils.paths import DATA_DIR, REPO_ROOT

logger = logging.getLogger(__name__)


def _env_local_path() -> Path:
    """Chemin vers .env.local (DATA_DIR en portable, REPO_ROOT en dev)."""
    portable = DATA_DIR / ".env.local"
    if portable.exists():
        return portable
    # En mode dev, DATA_DIR == REPO_ROOT/data, on écrit à la racine
    return REPO_ROOT / ".env.local"


@dataclass(frozen=True)
class AuthStatus:
    """État de l'authentification OAuth."""

    has_env_file: bool = False
    has_client_id: bool = False
    has_refresh_token: bool = False
    missing_keys: list[str] = field(default_factory=list)

    @property
    def has_credentials(self) -> bool:
        """Retourne True si le Client ID Azure est configuré (public client)."""
        return self.has_client_id

    @property
    def is_fully_configured(self) -> bool:
        """Retourne True si un refresh token est aussi présent."""
        return self.has_credentials and self.has_refresh_token


def get_auth_status() -> AuthStatus:
    """Vérifie l'état de la configuration OAuth.

    Charge .env.local si nécessaire, puis inspecte les variables d'environnement.

    Returns:
        AuthStatus avec les flags de présence des credentials.
    """
    load_dotenv_if_present()

    has_env = _env_local_path().exists()
    refresh_token = os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN", "").strip()

    # LEVELUP_CLIENT_ID est toujours intégré dans l'app (src/auth/_constants.py) —
    # l'utilisateur n'a plus besoin de configurer Azure.
    # SPNKR_AZURE_CLIENT_ID reste supporté comme surcharge backend optionnelle.
    has_client_id = True

    missing: list[str] = []
    if not refresh_token:
        missing.append("SPNKR_OAUTH_REFRESH_TOKEN")

    return AuthStatus(
        has_env_file=has_env,
        has_client_id=has_client_id,
        has_refresh_token=bool(refresh_token),
        missing_keys=missing,
    )


def check_credentials() -> bool:
    """Vérifie rapidement si les credentials Azure sont présentes.

    Returns:
        True si Client ID + Client Secret sont configurés.
    """
    return get_auth_status().has_credentials


def write_env_local(values: dict[str, str]) -> None:
    """Écrit ou met à jour des clés dans .env.local.

    Les clés existantes sont mises à jour, les nouvelles sont ajoutées à la fin.
    Les commentaires et lignes vides sont préservés.

    Args:
        values: Dictionnaire clé=valeur à écrire.
    """
    env_path = _env_local_path()
    env_path.parent.mkdir(parents=True, exist_ok=True)
    existing_lines: list[str] = []
    if env_path.exists():
        existing_lines = env_path.read_text(encoding="utf-8").splitlines()

    remaining = dict(values)
    updated_lines: list[str] = []

    for line in existing_lines:
        stripped = line.strip()
        if stripped and not stripped.startswith("#") and "=" in stripped:
            key = stripped.split("=", 1)[0].strip()
            if key in remaining:
                updated_lines.append(f"{key}={remaining.pop(key)}")
                continue
        updated_lines.append(line)

    # Ajouter les clés restantes à la fin
    for key, value in remaining.items():
        updated_lines.append(f"{key}={value}")

    env_path.write_text(
        "\n".join(updated_lines) + "\n",
        encoding="utf-8",
    )
    logger.info("Fichier .env.local mis à jour (%d clé(s))", len(values))
