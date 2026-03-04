"""Gestion centralisée des secrets LevelUp.

Permet de charger les secrets depuis Doppler (prioritaire) ou .env.local (fallback).
Les secrets sont injectés dans os.environ, donc tous les modules existants qui
lisent os.environ.get(...) continuent de fonctionner sans modification.

Configuration (app_settings.json) :
    "doppler_enabled": true,
    "doppler_project": "levelup",
    "doppler_config": "dev"

Usage :
    # En général, appelé une seule fois au démarrage de l'app.
    from src.utils.secrets import load_secrets_to_env
    load_secrets_to_env(project="levelup", config="dev")

    # Pour lire un secret spécifique (avec chargement automatique .env.local) :
    from src.utils.secrets import get_secret
    webhook = get_secret("DISCORD_WEBHOOK_URL")

Pré-requis Doppler :
    1. Installer le CLI Doppler : https://docs.doppler.com/docs/install-cli
    2. S'authentifier : `doppler login`
    3. Créer le projet "levelup" avec les secrets correspondants

Fallback :
    Si Doppler est indisponible ou désactivé, .env.local est chargé normalement.
"""

from __future__ import annotations

import logging
import os
import subprocess
from pathlib import Path

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_SECRETS_LOADED = False  # Guard : ne charger qu'une fois par processus


def _load_dotenv_local() -> int:
    """Charge .env.local / .env dans os.environ. Retourne le nombre de variables chargées."""
    n = 0
    for name in (".env.local", ".env"):
        path = _PROJECT_ROOT / name
        if not path.exists():
            continue
        try:
            content = path.read_text(encoding="utf-8")
        except Exception:
            continue
        for raw_line in content.splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" not in line:
                continue
            key, _, val = line.partition("=")
            key = key.strip()
            val = val.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = val
                n += 1
    return n


def load_doppler_secrets_to_env(  # noqa: C901, PLR0912
    project: str = "",
    config: str = "",
    *,
    no_override: bool = True,
) -> bool:
    """Télécharge les secrets Doppler et les injecte dans os.environ.

    Exécute ``doppler secrets download --format env --no-file``.
    Les variables déjà présentes dans os.environ ne sont pas écrasées
    si ``no_override=True`` (comportement par défaut).

    Args:
        project: Projet Doppler (ex: "levelup"). Vide = utilise le projet courant.
        config:  Config Doppler (ex: "dev", "prod"). Vide = utilise la config courante.
        no_override: Si True, n'écrase pas les variables déjà présentes dans os.environ.

    Returns:
        True si le chargement Doppler a réussi, False sinon (fallback .env.local utilisé).
    """
    global _SECRETS_LOADED
    if _SECRETS_LOADED:
        return True

    cmd = ["doppler", "secrets", "download", "--format", "env", "--no-file"]
    if project:
        cmd += ["--project", project]
    if config:
        cmd += ["--config", config]

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=15,
        )
        if result.returncode != 0:
            logger.warning(
                "[Secrets] Doppler CLI erreur (code %d) : %s — fallback .env.local",
                result.returncode,
                result.stderr.strip()[:200],
            )
            n_local = _load_dotenv_local()
            logger.debug("[Secrets] Chargé %d variables depuis .env.local", n_local)
            _SECRETS_LOADED = True
            return False

        # Parser le format env (KEY=VALUE)
        n_injected = 0
        for raw_line in result.stdout.splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" not in line:
                continue
            key, _, val = line.partition("=")
            key = key.strip()
            val = val.strip().strip('"')
            if not key:
                continue
            if no_override and key in os.environ:
                continue
            os.environ[key] = val
            n_injected += 1

        logger.info("[Secrets] %d secrets Doppler chargés en mémoire", n_injected)
        _SECRETS_LOADED = True
        return True

    except FileNotFoundError:
        logger.info("[Secrets] CLI Doppler introuvable — fallback .env.local")
    except subprocess.TimeoutExpired:
        logger.warning("[Secrets] Doppler timeout (>15s) — fallback .env.local")
    except Exception as exc:
        logger.warning("[Secrets] Erreur Doppler inattendue : %s — fallback .env.local", exc)

    n_local = _load_dotenv_local()
    logger.debug("[Secrets] Chargé %d variables depuis .env.local (fallback)", n_local)
    _SECRETS_LOADED = True
    return False


def get_secret(key: str, default: str | None = None) -> str | None:
    """Retourne la valeur d'un secret.

    Charge automatiquement .env.local si non déjà fait.
    Priorise os.environ (injecté par Doppler ou .env.local).

    Args:
        key: Nom de la variable (ex: "DISCORD_WEBHOOK_URL").
        default: Valeur par défaut si le secret n'est pas trouvé.

    Returns:
        Valeur du secret ou default.
    """
    if not _SECRETS_LOADED:
        _load_dotenv_local()
    return os.environ.get(key, default)
