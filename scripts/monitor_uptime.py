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
  {"status": "online"|"offline", "since": "<ISO datetime>"}

Usage :
  .venv\\Scripts\\python.exe scripts\\monitor_uptime.py
  (ou via Planificateur de tâches, toutes les 1 minute)
"""

from __future__ import annotations

import json
import logging
import os
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

# ---------------------------------------------------------------------------
# Chemin racine → import src.*
# ---------------------------------------------------------------------------

_ROOT = Path(__file__).resolve().parents[1]
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

_APP_SETTINGS = _ROOT / "app_settings.json"
_STATE_FILE = _ROOT / "data" / "cache" / "uptime_state.json"
_LOG_FILE = _ROOT / "data" / "cache" / "uptime_monitor.log"

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

_STATE_FILE.parent.mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.FileHandler(_LOG_FILE, encoding="utf-8"),
        logging.StreamHandler(sys.stdout),
    ],
)
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
    try:
        with open(_APP_SETTINGS, encoding="utf-8") as fh:
            return json.load(fh).get("discord_lang", "fr")
    except Exception:
        return "fr"


def _t(key: str, lang: str) -> str:
    """Retourne la chaîne traduite ; repli sur 'fr' si lang inconnue."""
    return _STRINGS.get(lang, _STRINGS["fr"])[key]


# ---------------------------------------------------------------------------
# Chargement de la configuration
# ---------------------------------------------------------------------------


def _load_dotenv(path: Path) -> None:
    """Charge un fichier .env dans os.environ (parser minimal)."""
    if not path.exists():
        return
    with open(path, encoding="utf-8") as fh:
        for raw in fh:
            line = raw.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            # Retire les guillemets éventuels autour de la valeur
            value = value.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = value


def _load_app_settings() -> dict:
    if not _APP_SETTINGS.exists():
        return {}
    try:
        with open(_APP_SETTINGS, encoding="utf-8") as fh:
            return json.load(fh)
    except Exception:
        return {}


def _get_webhook_url() -> str | None:
    """Retourne le webhook Discord configuré (.env.local → app_settings.json)."""
    _load_dotenv(_ROOT / ".env.local")
    _load_dotenv(_ROOT / ".env")
    cfg = _load_app_settings()
    url = (
        os.environ.get("DISCORD_WEBHOOK_URL", "").strip()
        or str(cfg.get("discord_webhook_url") or "").strip()
    )
    return url if url.startswith("https://discord.com/api/webhooks/") else None


def _resolve_tailscale_url() -> str | None:
    """Résout l'URL Tailscale Funnel active.

    Priorité :
    1. Variable d'environnement ``TAILSCALE_FUNNEL_URL`` (override manuel)
    2. ``tailscale funnel status --json`` via :func:`src.utils.tailscale.get_funnel_url`
    """
    _load_dotenv(_ROOT / ".env.local")
    _load_dotenv(_ROOT / ".env")

    # 1. Override manuel
    override = os.environ.get("TAILSCALE_FUNNEL_URL", "").strip()
    if override:
        logger.debug("[Tailscale] URL configurée manuellement : %s", override)
        return override

    # 2. Interrogation live de Tailscale
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


def _load_state() -> dict:
    """Lit l'état sauvegardé. Retourne {"status": "unknown"} si absent."""
    if not _STATE_FILE.exists():
        return {"status": "unknown"}
    try:
        with open(_STATE_FILE, encoding="utf-8") as fh:
            return json.load(fh)
    except Exception:
        return {"status": "unknown"}


def _save_state(status: str, since: datetime, url: str = "") -> None:
    """Persiste l'état courant (et l'URL connue si fournie)."""
    data: dict = {"status": status, "since": since.isoformat()}
    if url:
        data["url"] = url
    else:
        # Conserver l'URL précédente si on ne la connaît plus
        prev = _load_state()
        if prev.get("url"):
            data["url"] = prev["url"]
    with open(_STATE_FILE, "w", encoding="utf-8") as fh:
        json.dump(data, fh)


# ---------------------------------------------------------------------------
# Vérification HTTP
# ---------------------------------------------------------------------------


def _check_site(url: str, timeout: int = 10) -> bool:
    """Retourne True si le site répond avec un code HTTP < 500."""
    try:
        req = urllib.request.Request(
            url,
            headers={"User-Agent": "LevelUp-UptimeMonitor/1.0"},
            method="GET",
        )
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status < 500
    except urllib.error.HTTPError as exc:
        # 4xx = site up mais erreur applicative → on considère le site vivant
        return exc.code < 500
    except Exception:
        return False


# ---------------------------------------------------------------------------
# Notification Discord
# ---------------------------------------------------------------------------


def _send_discord(webhook_url: str, payload: dict) -> None:
    """Envoie un embed Discord via webhook (failsafe)."""
    try:
        body = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(
            webhook_url,
            data=body,
            headers={
                "Content-Type": "application/json",
                "User-Agent": "LevelUp-UptimeMonitor/1.0",
            },
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            if resp.status not in (200, 204):
                logger.warning("[Discord] Réponse inattendue : %s", resp.status)
    except Exception as exc:
        logger.warning("[Discord] Échec d'envoi : %s", exc)


def _notify_online(webhook_url: str, site_url: str, since: datetime, lang: str) -> None:
    """Message Discord « site en ligne »."""
    ts = since.astimezone().strftime("%d/%m/%Y à %H:%M:%S")
    payload = {
        "embeds": [
            {
                "title": _t("online_title", lang),
                "description": (
                    f"{_t('online_body', lang)}\n\n"
                    f"{_t('online_link', lang)} {site_url}\n"
                    f"{_t('online_since', lang)} {ts}"
                ),
                "color": 0x57F287,  # vert Discord
                "footer": {"text": _t("footer", lang)},
                "timestamp": since.isoformat(),
            }
        ]
    }
    _send_discord(webhook_url, payload)
    logger.info(_t("log_online", lang))


def _notify_offline(webhook_url: str, since: datetime, lang: str) -> None:
    """Message Discord « site hors ligne »."""
    ts = since.astimezone().strftime("%d/%m/%Y à %H:%M:%S")
    payload = {
        "embeds": [
            {
                "title": _t("offline_title", lang),
                "description": (
                    f"{_t('offline_body', lang)}\n\n" f"{_t('offline_since', lang)} {ts}"
                ),
                "color": 0xED4245,  # rouge Discord
                "footer": {"text": _t("footer", lang)},
                "timestamp": since.isoformat(),
            }
        ]
    }
    _send_discord(webhook_url, payload)
    logger.info(_t("log_offline", lang))


# ---------------------------------------------------------------------------
# Point d'entrée principal
# ---------------------------------------------------------------------------


def main() -> None:
    lang = _get_lang()
    previous = _load_state()
    prev_status: str = previous.get("status", "unknown")
    cached_url: str = previous.get("url", "")

    # 1. Résolution de l'URL Tailscale
    #    - Détection live (funnel status CLI) ou override .env.local en priorité
    #    - Si introuvable, repli sur l'URL mémorisée lors du dernier ONLINE
    #      → le ping échouera naturellement si le funnel est éteint
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
            sys.exit(1)

    # 2. Webhook Discord (optionnel — monitoring sans notif si absent)
    webhook_url = _get_webhook_url()
    if not webhook_url:
        logger.warning(
            "[Discord] Aucun webhook configuré — monitoring actif mais sans notifications. "
            "Ajoutez DISCORD_WEBHOOK_URL dans .env.local."
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

    # 4. Notification uniquement en cas de changement d'état
    if current_status != prev_status:
        _save_state(current_status, now, tailscale_url)
        if webhook_url:
            if current_status == "online":
                _notify_online(webhook_url, tailscale_url, now, lang)
            else:
                _notify_offline(webhook_url, now, lang)
    else:
        # Pas de changement : mettre à jour l'URL si elle a changé
        if tailscale_url != cached_url:
            _save_state(current_status, now, tailscale_url)
        logger.debug("Pas de changement d'état (%s).", current_status)


if __name__ == "__main__":
    main()
