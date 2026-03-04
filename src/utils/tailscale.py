"""Helper Tailscale pour LevelUp.

Permet de démarrer automatiquement `tailscale funnel` au lancement de l'app
et de récupérer l'URL publique pour la partager via Discord.

Pré-requis : Tailscale doit être installé et authentifié sur la machine.

Configuration (app_settings.json) :
    "tailscale_funnel_enabled": true

Usage :
    from src.utils.tailscale import start_funnel, get_funnel_url
    url = start_funnel(port=8501)   # démarre le funnel + retourne l'URL
    url = get_funnel_url()          # récupère l'URL active sans démarrer
"""

from __future__ import annotations

import json
import logging
import subprocess
import threading
from typing import Any

logger = logging.getLogger(__name__)

# Guard niveau processus : le funnel ne démarre qu'une seule fois,
# indépendamment du nombre de sessions WebSocket Streamlit.
_funnel_started: threading.Event = threading.Event()


def get_funnel_url() -> str | None:
    """Retourne l'URL publique active du funnel Tailscale.

    Exécute ``tailscale funnel status --json`` et extrait l'URL.

    Returns:
        URL publique (ex: ``"https://hostname.tailnet.ts.net"``) ou None si
        le funnel n'est pas actif ou si Tailscale n'est pas disponible.
    """
    try:
        result = subprocess.run(
            ["tailscale", "funnel", "status", "--json"],
            capture_output=True,
            text=True,
            timeout=10,
        )
        if result.returncode != 0:
            logger.debug("[Tailscale] `funnel status --json` a échoué : %s", result.stderr)
            return None

        data: Any = json.loads(result.stdout)
        if not isinstance(data, dict):
            return None

        # Format récent (Tailscale ≥1.56) : l'hostname est dans les clés de
        # "AllowFunnel" ou "Web", ex: "jonsbo.tail7d7b97.ts.net:443"
        for section in ("AllowFunnel", "Web"):
            keys = list((data.get(section) or {}).keys())
            if keys:
                host_port = keys[0]  # ex: "hostname.ts.net:443"
                dns_name = host_port.split(":")[0].rstrip(".")
                if dns_name:
                    return f"https://{dns_name}"

        # Champ direct SelfDNS (très anciennes versions)
        dns_name = str(data.get("SelfDNS") or "").rstrip(".")

        # Champ Self > DNSName (versions intermédiaires)
        if not dns_name:
            self_info = data.get("Self") or {}
            dns_name = str(self_info.get("DNSName") or "").rstrip(".")

        if dns_name:
            return f"https://{dns_name}"
        return None

    except FileNotFoundError:
        logger.debug("[Tailscale] CLI `tailscale` introuvable dans le PATH")
        return None
    except json.JSONDecodeError as exc:
        logger.debug("[Tailscale] get_funnel_url() JSON invalide : %s", exc)
        return None
    except Exception as exc:
        logger.debug("[Tailscale] get_funnel_url() erreur : %s", exc)
        return None


def start_funnel(port: int = 8501) -> str | None:
    """Démarre le funnel Tailscale sur le port donné et retourne l'URL.

    Essaie d'abord ``tailscale funnel --bg <port>`` (Tailscale ≥1.56, mode
    background), puis replie sur ``tailscale funnel <port>`` sans ``--bg``
    pour les versions antérieures.

    Args:
        port: Port local à exposer via Tailscale (défaut 8501 = Streamlit).

    Returns:
        URL publique si le funnel est actif, None en cas d'erreur.
    """
    # Tailscale ≥1.56 — le mode foreground (sans --bg) bloque indéfiniment.
    # On essaie d'abord --bg (background, retour immédiat, funnel persistant).
    logger.info("[Tailscale] Démarrage du funnel sur le port %d ...", port)
    for cmd in (
        ["tailscale", "funnel", "--bg", str(port)],
        ["tailscale", "funnel", str(port)],  # fallback anciennes versions
    ):
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=15,
            )
            # Code 0 = succès, 1 = déjà actif — on continue dans les deux cas
            if result.returncode not in (0, 1):
                logger.warning(
                    "[Tailscale] `%s` retour %d : %s",
                    " ".join(cmd),
                    result.returncode,
                    result.stderr.strip() or "(no stderr)",
                )
                # Tenter quand même de récupérer l'URL (funnel peut être déjà actif)
            else:
                # Commande réussie — pas besoin du fallback
                logger.info("[Tailscale] `%s` OK (code %d)", " ".join(cmd), result.returncode)
                break
        except FileNotFoundError:
            logger.warning("[Tailscale] CLI `tailscale` introuvable — funnel non démarré")
            return None
        except subprocess.TimeoutExpired:
            logger.warning(
                "[Tailscale] `%s` timeout (>15s) — commande bloquante, essai suivant ignoré",
                " ".join(cmd),
            )
            # Si le timeout vient du fallback sans --bg, on abandonne
            break
        except Exception as exc:
            logger.warning("[Tailscale] start_funnel() erreur : %s", exc)
            return None

    url = get_funnel_url()
    if url:
        logger.info("[Tailscale] Funnel actif → %s", url)
    else:
        logger.warning("[Tailscale] Funnel démarré mais URL introuvable (get_funnel_url() → None)")
    return url


def is_funnel_started() -> bool:
    """Retourne True si le funnel a déjà été démarré dans ce processus."""
    return _funnel_started.is_set()


def ensure_funnel_started_once(port: int = 8501) -> str | None:
    """Démarre le funnel Tailscale une seule fois par processus Python.

    Appels suivants (depuis d'autres sessions WebSocket Streamlit) sont ignorés.

    Args:
        port: Port local à exposer (défaut 8501).

    Returns:
        URL publique si premier appel et succès, None sinon.
    """
    if _funnel_started.is_set():
        return None  # Déjà démarré dans ce processus
    _funnel_started.set()
    return start_funnel(port=port)
