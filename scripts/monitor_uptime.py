"""Surveillance de disponibilité du site LevelUp (Tailscale Funnel).

Ce script est conçu pour être lancé toutes les minutes par le Planificateur
de tâches Windows.  Il fait un ping HTTP sur l'URL Tailscale Funnel et envoie
une notification Discord uniquement lorsque l'état change :

  • OFFLINE → ONLINE  : message « ✅ Site en ligne » avec le lien Tailscale
  • ONLINE  → OFFLINE : message « ❌ Site hors ligne »

L'URL Tailscale est récupérée automatiquement via ``tailscale funnel status --json``
(aucune configuration nécessaire).  On peut aussi la forcer via .env.local :

  TAILSCALE_FUNNEL_URL=https://<machine>.<tailnet>.ts.net  # override manuel

Le webhook Discord est configuré dans .env.local ou app_settings.json :

  DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/<id>/<token>

La langue des messages suit ``discord_lang`` dans app_settings.json
(``"fr"`` ou ``"en"``).

Fichier d'état persistant : data/cache/uptime_state.json
  {"status": "online"|"offline", "since": "<ISO datetime>",
   "consecutive_failures": 0}

Anti-flapping : par défaut, le script attend 3 checks consécutifs en échec
avant de notifier OFFLINE (configurable via ``UPTIME_OFFLINE_THRESHOLD``
dans .env.local ou le paramètre ``offline_threshold`` de ``main()``).

Usage :
  .venv\\Scripts\\python.exe scripts\\monitor_uptime.py        # avec fenêtre console
  .venv\\Scripts\\pythonw.exe scripts\\monitor_uptime.py       # sans fenêtre (arrière-plan)
  (le Planificateur de tâches utilise pythonw.exe automatiquement)

Enregistrement de la tâche planifiée (PowerShell admin) :
  .\\scripts\\setup_uptime_task.ps1
"""

from __future__ import annotations

import json
import logging
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

# ---------------------------------------------------------------------------
# Chemin racine → import src.*
# ---------------------------------------------------------------------------

_ROOT = Path(__file__).resolve().parents[1]
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

# ---------------------------------------------------------------------------
# Constantes
# ---------------------------------------------------------------------------

_APP_SETTINGS = _ROOT / "app_settings.json"
_STATE_FILE = _ROOT / "data" / "cache" / "uptime_state.json"
_LOG_FILE = _ROOT / "data" / "cache" / "uptime_monitor.log"

_WEBHOOK_PREFIX = "https://discord.com/api/webhooks/"
_USER_AGENT = "LevelUp-UptimeMonitor/1.0"
_HTTP_TIMEOUT = 10
_DISCORD_RETRY_DELAY = 2  # secondes entre les tentatives Discord
_DISCORD_MAX_RETRIES = 2  # nombre total de tentatives (1 + 1 retry)
_DEFAULT_OFFLINE_THRESHOLD = 3  # checks consécutifs en échec avant notif OFFLINE

# Couleurs Discord
_COLOR_GREEN = 0x57F287
_COLOR_RED = 0xED4245

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Traductions (i18n)
# ---------------------------------------------------------------------------

_STRINGS: dict[str, dict[str, str]] = {
    "fr": {
        "online_title": "✅ LevelUp — Site en ligne",
        "online_body": "Le dashboard est **accessible**.",
        "online_link": "🔗 **Lien Tailscale Funnel :**",
        "online_since": "🕐 En ligne depuis :",
        "offline_title": "❌ LevelUp — Site hors ligne",
        "offline_body": "Le dashboard est **inaccessible**.",
        "offline_since": "🕐 Hors ligne depuis :",
        "footer": "LevelUp Uptime Monitor",
        "log_online": "[Discord] Notification ONLINE envoyée.",
        "log_offline": "[Discord] Notification OFFLINE envoyée.",
    },
    "en": {
        "online_title": "✅ LevelUp — Dashboard online",
        "online_body": "The dashboard is **reachable**.",
        "online_link": "🔗 **Tailscale Funnel link:**",
        "online_since": "🕐 Online since:",
        "offline_title": "❌ LevelUp — Dashboard offline",
        "offline_body": "The dashboard is **unreachable**.",
        "offline_since": "🕐 Offline since:",
        "footer": "LevelUp Uptime Monitor",
        "log_online": "[Discord] ONLINE notification sent.",
        "log_offline": "[Discord] OFFLINE notification sent.",
    },
}


def _get_lang() -> str:
    """Retourne la langue Discord depuis app_settings.json (défaut : 'fr')."""
    return _load_app_settings().get("discord_lang", "fr")


def _t(key: str, lang: str) -> str:
    """Retourne la chaîne traduite ; repli sur 'fr' puis placeholder si clé inconnue."""
    strings = _STRINGS.get(lang, _STRINGS["fr"])
    return strings.get(key, _STRINGS["fr"].get(key, f"[{key}]"))


# ---------------------------------------------------------------------------
# Chargement de la configuration
# (réutilise src.utils.secrets pour .env.local / Doppler — source unique)
# ---------------------------------------------------------------------------


def _load_secrets_once() -> None:
    """Charge les secrets une seule fois via l'infra centralisée du projet."""
    from src.utils.secrets import get_secret  # noqa: F401 — déclenche _load_dotenv_local

    # get_secret charge automatiquement .env.local si pas déjà fait.
    # L'appel suffit à garantir que os.environ est peuplé.
    get_secret("_UPTIME_SECRETS_GUARD")


def _load_app_settings() -> dict[str, Any]:
    """Charge app_settings.json → dict. Retourne {} en cas d'échec."""
    if not _APP_SETTINGS.exists():
        return {}
    try:
        with open(_APP_SETTINGS, encoding="utf-8") as fh:
            return json.load(fh)
    except Exception:
        return {}


def _get_webhook_url() -> str | None:
    """Retourne le webhook Discord si les notifications sont activées.

    Respecte le flag ``discord_notifications_enabled`` d'app_settings.json
    (cohérent avec ``src.utils.discord_notifier``).

    Priorité de l'URL :
    1. Variable d'environnement ``DISCORD_WEBHOOK_URL`` (chargé depuis .env.local)
    2. Clé ``discord_webhook_url`` dans app_settings.json (rétrocompat.)
    """
    cfg = _load_app_settings()

    # Vérifier que les notifications Discord sont activées
    if not cfg.get("discord_notifications_enabled", False):
        logger.debug("[Discord] Notifications désactivées (discord_notifications_enabled=False).")
        return None

    # Charger .env.local explicitement pour garantir DISCORD_WEBHOOK_URL même
    # dans les contextes sans console (pythonw.exe / planificateur de tâches).
    from src.utils.secrets import get_secret

    url = get_secret("DISCORD_WEBHOOK_URL", "").strip()
    if not url:
        url = str(cfg.get("discord_webhook_url") or "").strip()

    if url.startswith(_WEBHOOK_PREFIX):
        return url

    logger.warning(
        "[Discord] Notifications activées mais aucun webhook valide trouvé. "
        "Vérifiez DISCORD_WEBHOOK_URL dans .env.local."
    )
    return None


def _resolve_tailscale_url() -> str | None:
    """Résout l'URL Tailscale Funnel active.

    Priorité :
    1. Variable d'environnement ``TAILSCALE_FUNNEL_URL`` (override manuel)
    2. ``tailscale funnel status --json`` via :func:`src.utils.tailscale.get_funnel_url`
    """
    import os

    override = os.environ.get("TAILSCALE_FUNNEL_URL", "").strip()
    if override:
        logger.debug("[Tailscale] URL configurée manuellement : %s", override)
        return override

    try:
        from src.utils.tailscale import get_funnel_url

        url = get_funnel_url()
        if url:
            logger.debug("[Tailscale] URL détectée via CLI : %s", url)
        return url
    except Exception as exc:
        logger.debug("[Tailscale] Impossible d'appeler get_funnel_url() : %s", exc)
        return None


# ---------------------------------------------------------------------------
# Gestion de l'état persistant
# ---------------------------------------------------------------------------


def _load_state(state_file: Path | None = None) -> dict[str, Any]:
    """Lit l'état sauvegardé. Retourne {"status": "unknown"} si absent."""
    path = state_file or _STATE_FILE
    if not path.exists():
        return {"status": "unknown"}
    try:
        with open(path, encoding="utf-8") as fh:
            return json.load(fh)
    except Exception:
        return {"status": "unknown"}


def _save_state(
    status: str,
    since: datetime,
    *,
    url: str = "",
    previous: dict[str, Any] | None = None,
    consecutive_failures: int = 0,
    state_file: Path | None = None,
) -> None:
    """Persiste l'état courant dans le fichier JSON.

    Args:
        status: ``"online"`` ou ``"offline"``.
        since: Horodatage du changement d'état.
        url: URL Tailscale Funnel connue. Si vide, conserve celle de ``previous``.
        previous: État précédent déjà chargé (évite une relecture fichier).
        consecutive_failures: Nombre de checks consécutifs en échec.
        state_file: Chemin du fichier d'état (pour les tests).
    """
    path = state_file or _STATE_FILE
    data: dict[str, Any] = {
        "status": status,
        "since": since.isoformat(),
        "consecutive_failures": consecutive_failures,
    }
    if url:
        data["url"] = url
    elif previous and previous.get("url"):
        data["url"] = previous["url"]

    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", encoding="utf-8") as fh:
        json.dump(data, fh)


# ---------------------------------------------------------------------------
# Vérification HTTP
# ---------------------------------------------------------------------------


def _check_site(url: str, timeout: int = _HTTP_TIMEOUT) -> bool:
    """Retourne True si le site répond avec un code HTTP < 500."""
    try:
        req = urllib.request.Request(
            url,
            headers={"User-Agent": _USER_AGENT},
            method="GET",
        )
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status < 500
    except urllib.error.HTTPError as exc:
        # 4xx = site up mais erreur applicative → on considère le site vivant
        if exc.code < 500:
            logger.debug("[HTTP] Réponse %d (site considéré vivant).", exc.code)
            return True
        return False
    except Exception:
        return False


# ---------------------------------------------------------------------------
# Notification Discord (réutilise l'infra existante avec retry)
# ---------------------------------------------------------------------------


def _send_discord(webhook_url: str, payload: dict[str, Any]) -> bool:
    """Envoie un embed Discord via l'infra existante, avec retry simple.

    Retourne True si l'envoi a réussi, False sinon.
    """
    from src.utils.discord_notifier import send_discord_notification

    for attempt in range(1, _DISCORD_MAX_RETRIES + 1):
        if send_discord_notification(payload, webhook_url):
            return True
        if attempt < _DISCORD_MAX_RETRIES:
            logger.debug(
                "[Discord] Tentative %d/%d échouée, retry dans %ds…",
                attempt,
                _DISCORD_MAX_RETRIES,
                _DISCORD_RETRY_DELAY,
            )
            time.sleep(_DISCORD_RETRY_DELAY)

    logger.warning("[Discord] Échec d'envoi après %d tentatives.", _DISCORD_MAX_RETRIES)
    return False


def _build_online_payload(site_url: str, since: datetime, lang: str) -> dict[str, Any]:
    """Construit le payload Discord « site en ligne »."""
    ts = since.astimezone().strftime("%d/%m/%Y à %H:%M:%S")
    return {
        "embeds": [
            {
                "title": _t("online_title", lang),
                "description": (
                    f"{_t('online_body', lang)}\n\n"
                    f"{_t('online_link', lang)} {site_url}\n"
                    f"{_t('online_since', lang)} {ts}"
                ),
                "color": _COLOR_GREEN,
                "footer": {"text": _t("footer", lang)},
                "timestamp": since.isoformat(),
            }
        ]
    }


def _build_offline_payload(since: datetime, lang: str) -> dict[str, Any]:
    """Construit le payload Discord « site hors ligne »."""
    ts = since.astimezone().strftime("%d/%m/%Y à %H:%M:%S")
    return {
        "embeds": [
            {
                "title": _t("offline_title", lang),
                "description": (f"{_t('offline_body', lang)}\n\n{_t('offline_since', lang)} {ts}"),
                "color": _COLOR_RED,
                "footer": {"text": _t("footer", lang)},
                "timestamp": since.isoformat(),
            }
        ]
    }


def _notify_online(webhook_url: str, site_url: str, since: datetime, lang: str) -> None:
    """Message Discord « site en ligne »."""
    _send_discord(webhook_url, _build_online_payload(site_url, since, lang))
    logger.info(_t("log_online", lang))


def _notify_offline(webhook_url: str, since: datetime, lang: str) -> None:
    """Message Discord « site hors ligne »."""
    _send_discord(webhook_url, _build_offline_payload(since, lang))
    logger.info(_t("log_offline", lang))


# ---------------------------------------------------------------------------
# Initialisation (appelée une seule fois dans main, pas à l'import)
# ---------------------------------------------------------------------------


def _setup_logging() -> None:
    """Configure le logging vers fichier + stdout. Appelé dans main()."""
    _LOG_FILE.parent.mkdir(parents=True, exist_ok=True)
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
        handlers=[
            logging.FileHandler(_LOG_FILE, encoding="utf-8"),
            logging.StreamHandler(sys.stdout),
        ],
    )


# ---------------------------------------------------------------------------
# Point d'entrée principal
# ---------------------------------------------------------------------------


def main(*, offline_threshold: int | None = None) -> None:
    """Exécute un cycle de surveillance.

    Args:
        offline_threshold: Nombre de checks consécutifs en échec avant de
            notifier OFFLINE. ``None`` = lire ``UPTIME_OFFLINE_THRESHOLD``
            dans l'environnement ou utiliser ``_DEFAULT_OFFLINE_THRESHOLD``.
    """
    _setup_logging()
    _load_secrets_once()

    import os

    if offline_threshold is None:
        offline_threshold = int(
            os.environ.get("UPTIME_OFFLINE_THRESHOLD", str(_DEFAULT_OFFLINE_THRESHOLD))
        )

    lang = _get_lang()
    previous = _load_state()
    prev_status: str = previous.get("status", "unknown")
    cached_url: str = previous.get("url", "")
    prev_failures: int = previous.get("consecutive_failures", 0)

    # 1. Résolution de l'URL Tailscale
    tailscale_url = _resolve_tailscale_url()
    if not tailscale_url:
        if cached_url:
            logger.warning(
                "[Tailscale] Funnel non détecté via CLI — utilisation de l'URL en cache : %s",
                cached_url,
            )
            tailscale_url = cached_url
        else:
            logger.error(
                "[Tailscale] Aucune URL disponible (funnel inactif et aucun cache). "
                "Démarrez le funnel au moins une fois : tailscale funnel --bg 8501"
            )
            # Exit 0 pour ne pas polluer le Planificateur de tâches Windows
            sys.exit(0)

    # 2. Webhook Discord (optionnel — monitoring sans notif si absent)
    webhook_url = _get_webhook_url()
    if not webhook_url:
        logger.warning(
            "[Discord] Aucun webhook configuré ou notifications désactivées — "
            "monitoring actif mais sans notifications. "
            "Activez discord_notifications_enabled et ajoutez DISCORD_WEBHOOK_URL dans .env.local."
        )

    # 3. Ping du site
    now = datetime.now(timezone.utc)
    is_up = _check_site(tailscale_url)
    current_status = "online" if is_up else "offline"

    logger.info(
        "URL : %s | état précédent : %s | état actuel : %s",
        tailscale_url,
        prev_status,
        current_status,
    )

    # 4. Logique de transition avec debounce anti-flapping
    if is_up:
        # Site UP → reset du compteur d'échecs
        if prev_status != "online":
            # Transition vers ONLINE (immédiate, pas de debounce)
            _save_state("online", now, url=tailscale_url, previous=previous, consecutive_failures=0)
            if webhook_url:
                _notify_online(webhook_url, tailscale_url, now, lang)
        else:
            # Toujours online : MAJ URL si changée
            if tailscale_url != cached_url:
                _save_state(
                    "online", now, url=tailscale_url, previous=previous, consecutive_failures=0
                )
            logger.debug("Pas de changement d'état (online).")
    else:
        # Site DOWN → incrémenter le compteur d'échecs
        failures = prev_failures + 1 if prev_status != "offline" else prev_failures

        if prev_status == "offline":
            # Déjà notifié OFFLINE, rien à faire
            logger.debug("Pas de changement d'état (offline, %d échecs).", failures)
        elif failures >= offline_threshold:
            # Seuil atteint → notifier OFFLINE
            logger.info(
                "[Debounce] %d/%d checks en échec → notification OFFLINE.",
                failures,
                offline_threshold,
            )
            _save_state(
                "offline", now, url=tailscale_url, previous=previous, consecutive_failures=failures
            )
            if webhook_url:
                _notify_offline(webhook_url, now, lang)
        else:
            # Pas encore au seuil → incrémenter le compteur sans notifier
            logger.info(
                "[Debounce] Check en échec %d/%d — pas encore de notification.",
                failures,
                offline_threshold,
            )
            _save_state(
                prev_status,
                now,
                url=tailscale_url,
                previous=previous,
                consecutive_failures=failures,
            )


if __name__ == "__main__":
    main()
