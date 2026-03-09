"""Acquisition de refresh_token via MSAL Device Code Flow.

Le Device Code Flow simplifie drastiquement la configuration Azure :
- Pas de ``client_secret``
- Pas de ``redirect_uri``
- L'app Azure nécessite uniquement :
    - "Supported account types" → Personal Microsoft accounts only
    - "Allow public client flows" → Yes

L'utilisateur visite ``https://microsoft.com/devicelogin`` et entre un code
court (ex: ``ABCD-1234``). Aucune redirection localhost.

Usage CLI (bloquant) ::

    result, app = initiate_device_flow(client_id)
    print(f"Allez sur {result.verification_url} et entrez : {result.user_code}")
    refresh_token = acquire_token_blocking(app, result._flow)

Usage Streamlit (non-bloquant) ::

    result, app = initiate_device_flow(client_id)
    q = queue.Queue()
    threading.Thread(
        target=lambda: q.put(acquire_token_blocking(app, result._flow)),
        daemon=True,
    ).start()
    # Plus tard, dans un handler Streamlit :
    token = q.get_nowait()  # → str | raises queue.Empty
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from typing import Any

logger = logging.getLogger(__name__)

# Import optionnel — None si msal n'est pas installé (pip install msal)
try:
    import msal
except ImportError:
    msal = None  # type: ignore[assignment]

# Authority Microsoft pour comptes personnels (Xbox Live)
_AUTHORITY = "https://login.microsoftonline.com/consumers"

# Scopes Xbox Live requis
_SCOPES = ["Xboxlive.signin", "Xboxlive.offline_access"]

# Mapping des codes d'erreur MSAL → codes internes
_ERROR_MAP: dict[str, str] = {
    "authorization_declined": "declined",
    "expired_token": "timeout",
    "code_expired": "timeout",
    "authorization_pending": "pending",
    "slow_down": "pending",
}


class DeviceFlowError(Exception):
    """Erreur structurée du Device Code Flow.

    Attributes:
        code: Code machine (``declined``, ``timeout``, ``bad_client_id``,
            ``msal_unavailable``, ``no_refresh_token``, ``unknown``).
        detail: Message lisible pour l'utilisateur.
    """

    def __init__(self, code: str, detail: str) -> None:
        super().__init__(detail)
        self.code = code
        self.detail = detail


@dataclass
class DeviceCodeResult:
    """Données retournées lors de l'initiation du Device Code Flow.

    Attributes:
        user_code: Code court à entrer sur microsoft.com/devicelogin (ex: ``ABCD-1234``).
        verification_url: URL de vérification (``https://microsoft.com/devicelogin``).
        expires_in: Secondes avant expiration du code.
        message: Message MSAL verbatim (localisé par Microsoft).
        _flow: Objet flow opaque MSAL (requis pour ``acquire_token_blocking``).
    """

    user_code: str
    verification_url: str
    expires_in: int
    message: str
    _flow: dict = field(repr=False)


def initiate_device_flow(client_id: str) -> tuple[DeviceCodeResult, Any]:
    """Démarre le Device Code Flow et retourne le code à afficher à l'utilisateur.

    Args:
        client_id: Application (client) ID Azure (sans secret requis).

    Returns:
        Tuple ``(DeviceCodeResult, msal_app)`` — l'app doit être passée à
        ``acquire_token_blocking()``.

    Raises:
        DeviceFlowError: Si client_id vide (code=``bad_client_id``), MSAL absent
            (code=``msal_unavailable``), ou si l'initiation échoue (code=``bad_client_id``).
    """
    if not (client_id or "").strip():
        raise DeviceFlowError("bad_client_id", "client_id vide ou manquant.")

    if msal is None:
        logger.error("MSAL non installé — pip install msal")
        raise DeviceFlowError(
            "msal_unavailable",
            "MSAL non installé. Exécutez : pip install msal",
        )

    app = msal.PublicClientApplication(
        client_id.strip(),
        authority=_AUTHORITY,
    )

    flow = app.initiate_device_flow(scopes=_SCOPES)

    if "error" in flow:
        err = flow.get("error", "unknown")
        desc = flow.get("error_description", "")
        logger.error("Device flow : échec initiation — %s: %s", err, desc)
        raise DeviceFlowError(
            "bad_client_id",
            f"Échec initiation device flow : {err} — {desc}",
        )

    user_code = str(flow.get("user_code", ""))
    expires_in = int(flow.get("expires_in", 900))
    logger.info("Device flow initié — user_code=%s expires_in=%ds", user_code, expires_in)

    result = DeviceCodeResult(
        user_code=user_code,
        verification_url=str(flow.get("verification_uri", "https://microsoft.com/devicelogin")),
        expires_in=expires_in,
        message=str(flow.get("message", "")),
        _flow=flow,
    )
    return result, app


def acquire_token_blocking(msal_app: Any, flow: dict) -> str:
    """Bloque jusqu'à l'obtention du refresh_token ou l'expiration du code.

    Polls internalement à l'intervalle défini par MSAL (typiquement 5 s).
    À appeler dans un thread dédié côté Streamlit ; directement en CLI.

    Args:
        msal_app: Instance ``msal.PublicClientApplication`` retournée par
            ``initiate_device_flow()``.
        flow: Dict opaque MSAL (champ ``_flow`` de ``DeviceCodeResult``).

    Returns:
        Le refresh_token Microsoft.

    Raises:
        DeviceFlowError: Avec code ``timeout``, ``declined``, ``no_refresh_token``
            ou ``unknown``.
    """
    logger.debug("Device flow : début du polling (intervalle=%ds)", flow.get("interval", 5))

    result = msal_app.acquire_token_by_device_flow(flow)

    if "refresh_token" in result:
        token: str = result["refresh_token"]
        logger.info("Device flow : refresh_token obtenu (len=%d)", len(token))
        return token

    # Pas de refresh_token : analyser l'erreur
    error = str(result.get("error", "unknown"))
    desc = str(result.get("error_description", ""))
    internal_code = _ERROR_MAP.get(error, "unknown")

    if internal_code == "declined":
        logger.error("Device flow : refusé par l'utilisateur")
        raise DeviceFlowError("declined", "L'utilisateur a refusé la connexion Xbox.")

    if internal_code == "timeout":
        user_code = str(flow.get("user_code", "?"))
        logger.warning("Device flow : timeout (user_code=%s)", user_code)
        raise DeviceFlowError("timeout", f"Le code a expiré (user_code={user_code}).")

    # Cas access_token présent sans refresh_token (scopes mal configurés)
    if result.get("access_token") and not result.get("refresh_token"):
        logger.error("Device flow : access_token présent mais pas de refresh_token")
        raise DeviceFlowError(
            "no_refresh_token",
            "Réponse Microsoft sans refresh_token. "
            "Vérifiez que l'app Azure autorise 'Xboxlive.offline_access'.",
        )

    logger.error("Device flow : erreur inattendue — %s: %s", error, desc)
    raise DeviceFlowError("unknown", f"Erreur MSAL : {error} — {desc}")
