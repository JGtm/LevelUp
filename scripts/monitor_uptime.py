"""Surveillance de disponibilité du site LevelUp (Tailscale Funnel).

Ce script est conçu pour être lancé toutes les minutes par le Planificateur
de tâches Windows.  Il fait un ping HTTP sur l'URL Tailscale Funnel et envoie
une notification Discord uniquement lorsque l'état change :

  • OFFLINE → ONLINE  : message « ✅ Site en ligne » avec le lien Tailscale
  • ONLINE  → OFFLINE : message « ❌ Site hors ligne »

Configuration (dans .env.local ou app_settings.json) :
  TAILSCALE_FUNNEL_URL=https://<machine>.<tailnet>.ts.net
  DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/<id>/<token>

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
# Chemins
# ---------------------------------------------------------------------------

_ROOT = Path(__file__).resolve().parents[1]
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


def _get_config() -> tuple[str | None, str | None]:
    """Retourne (tailscale_url, webhook_url) depuis .env.local puis app_settings.json."""
    # Charger .env.local en priorité
    _load_dotenv(_ROOT / ".env.local")
    _load_dotenv(_ROOT / ".env")

    cfg = _load_app_settings()

    tailscale_url = (
        os.environ.get("TAILSCALE_FUNNEL_URL", "").strip()
        or str(cfg.get("tailscale_funnel_url") or "").strip()
    )
    webhook_url = (
        os.environ.get("DISCORD_WEBHOOK_URL", "").strip()
        or str(cfg.get("discord_webhook_url") or "").strip()
    )

    return tailscale_url or None, webhook_url or None


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


def _save_state(status: str, since: datetime) -> None:
    """Persiste le nouvel état."""
    with open(_STATE_FILE, "w", encoding="utf-8") as fh:
        json.dump({"status": status, "since": since.isoformat()}, fh)


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


def _notify_online(webhook_url: str, site_url: str, since: datetime) -> None:
    """Message Discord « site en ligne »."""
    ts = since.astimezone().strftime("%d/%m/%Y à %H:%M:%S")
    payload = {
        "embeds": [
            {
                "title": "✅ LevelUp — Site en ligne",
                "description": (
                    f"Le dashboard est **accessible**.\n\n"
                    f"🔗 **Lien Tailscale Funnel :** {site_url}\n"
                    f"🕐 En ligne depuis : {ts}"
                ),
                "color": 0x57F287,  # vert Discord
                "footer": {"text": "LevelUp Uptime Monitor"},
                "timestamp": since.isoformat(),
            }
        ]
    }
    _send_discord(webhook_url, payload)
    logger.info("[Discord] Notification ONLINE envoyée.")


def _notify_offline(webhook_url: str, since: datetime) -> None:
    """Message Discord « site hors ligne »."""
    ts = since.astimezone().strftime("%d/%m/%Y à %H:%M:%S")
    payload = {
        "embeds": [
            {
                "title": "❌ LevelUp — Site hors ligne",
                "description": (
                    f"Le dashboard est **inaccessible**.\n\n" f"🕐 Hors ligne depuis : {ts}"
                ),
                "color": 0xED4245,  # rouge Discord
                "footer": {"text": "LevelUp Uptime Monitor"},
                "timestamp": since.isoformat(),
            }
        ]
    }
    _send_discord(webhook_url, payload)
    logger.info("[Discord] Notification OFFLINE envoyée.")


# ---------------------------------------------------------------------------
# Point d'entrée principal
# ---------------------------------------------------------------------------


def main() -> None:
    tailscale_url, webhook_url = _get_config()

    if not tailscale_url:
        logger.error(
            "TAILSCALE_FUNNEL_URL non configuré. "
            "Ajoutez-le dans .env.local ou dans app_settings.json (clé tailscale_funnel_url)."
        )
        sys.exit(1)

    if not webhook_url:
        logger.warning(
            "DISCORD_WEBHOOK_URL non configuré — le monitoring tourne mais sans notifications. "
            "Ajoutez-le dans .env.local ou dans app_settings.json (clé discord_webhook_url)."
        )

    previous = _load_state()
    prev_status: str = previous.get("status", "unknown")

    now = datetime.now(timezone.utc)
    is_up = _check_site(tailscale_url)
    current_status = "online" if is_up else "offline"

    logger.info(
        "URL : %s | État précédent : %s | État actuel : %s",
        tailscale_url,
        prev_status,
        current_status,
    )

    if current_status != prev_status:
        _save_state(current_status, now)
        if webhook_url:
            if current_status == "online":
                _notify_online(webhook_url, tailscale_url, now)
            else:
                _notify_offline(webhook_url, now)
    else:
        logger.debug("Pas de changement d'état (%s).", current_status)


if __name__ == "__main__":
    main()
