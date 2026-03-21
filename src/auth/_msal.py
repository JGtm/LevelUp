"""Couche MSAL — cache SerializableTokenCache ↔ DuckDB + Device Code Flow primitives.

Le cache MSAL est sérialisé en JSON et stocké dans la table ``sync_meta``
de la base stats.duckdb du joueur (clé : ``msal_token_cache``).

À chaque obtention silencieuse de token, MSAL tourne automatiquement le
refresh_token interne. ``save_msal_cache_if_changed()`` détecte ce changement
via ``has_state_changed`` et persiste le nouveau cache, garantissant une
durée de vie indefinie (pas d'expiration ~90j).

Usage typique :
    cache = load_msal_cache(db_path)
    app = build_msal_app(cache)
    access_token = acquire_token_silent(app)
    if access_token is None:
        result, flow = initiate_device_flow(app)
        access_token = acquire_token_by_device_flow(app, flow)
    save_msal_cache_if_changed(db_path, cache)
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from src.auth._constants import LEVELUP_CLIENT_ID, MSAL_AUTHORITY, MSAL_CACHE_DB_KEY, XBOX_SCOPES

logger = logging.getLogger(__name__)

# Import optionnel — None si msal n'est pas installé
try:
    import msal as _msal_module
except ImportError:
    _msal_module = None  # type: ignore[assignment]


# =============================================================================
# Structures de données
# =============================================================================


class MsalUnavailableError(Exception):
    """Levée si le package ``msal`` n'est pas installé."""


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
class DeviceCodeInfo:
    """Informations à afficher à l'utilisateur pour le Device Code Flow.

    Attributes:
        user_code: Code court (ex: ``ABCD-1234``) à entrer sur la page de vérification.
        verification_url: URL où entrer le code (``https://microsoft.com/devicelogin``).
        expires_in: Secondes avant expiration du code.
        message: Message MSAL verbatim (localisé par Microsoft).
        _flow: Dict opaque MSAL requis pour ``acquire_token_by_device_flow()``.
    """

    user_code: str
    verification_url: str
    expires_in: int
    message: str
    _flow: dict = field(repr=False)


# =============================================================================
# Cache MSAL ↔ DuckDB
# =============================================================================


_SYNC_META_DDL = """
CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
"""


def load_msal_cache(db_path: Path | str) -> Any:
    """Charge le cache MSAL depuis sync_meta.

    Si la base n'existe pas ou si aucun cache n'est stocké, retourne un cache
    vide prêt à l'emploi (premier lancement).

    Args:
        db_path: Chemin vers stats.duckdb du joueur.

    Returns:
        Instance ``msal.SerializableTokenCache`` (vide ou chargée).

    Raises:
        MsalUnavailableError: Si MSAL n'est pas installé.
    """
    if _msal_module is None:
        raise MsalUnavailableError("MSAL non installé. Exécutez : pip install msal")

    cache = _msal_module.SerializableTokenCache()
    path = Path(db_path)

    if not path.exists():
        logger.debug("load_msal_cache: DB absente (%s) — cache vide", path)
        return cache

    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(path) as conn:
            row = conn.execute(
                "SELECT value FROM sync_meta WHERE key = ?",
                (MSAL_CACHE_DB_KEY,),
            ).fetchone()
            if row and row[0]:
                cache.deserialize(str(row[0]))
                logger.debug("load_msal_cache: cache MSAL chargé depuis %s", path.name)
            else:
                logger.debug("load_msal_cache: aucun cache stocké dans %s", path.name)
    except Exception as exc:
        logger.warning("load_msal_cache: impossible de lire sync_meta (%s) — cache vide", exc)

    return cache


def save_msal_cache_if_changed(db_path: Path | str, cache: Any) -> None:
    """Persiste le cache MSAL dans sync_meta si MSAL l'a modifié.

    Microsoft tourne le refresh_token à chaque utilisation. Si ``has_state_changed``
    est True, le nouveau cache contenant le nouveau refresh_token est sérialisé
    et écrit dans DuckDB.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        cache: Instance ``msal.SerializableTokenCache`` retournée par ``load_msal_cache()``.
    """
    if not cache.has_state_changed:
        return

    try:
        from src.utils.db import duckdb_read_write

        serialized = cache.serialize()
        with duckdb_read_write(db_path) as conn:
            conn.execute(_SYNC_META_DDL)
            conn.execute(
                """INSERT OR REPLACE INTO sync_meta (key, value, updated_at)
                   VALUES (?, ?, CURRENT_TIMESTAMP)""",
                (MSAL_CACHE_DB_KEY, serialized),
            )
        logger.debug("save_msal_cache_if_changed: cache MSAL persisté dans %s", Path(db_path).name)
    except Exception as exc:
        logger.warning("save_msal_cache_if_changed: échec écriture cache MSAL: %s", exc)


# =============================================================================
# App MSAL
# =============================================================================


def build_msal_app(cache: Any) -> Any:
    """Construit l'application MSAL publique LevelUp avec le cache fourni.

    Args:
        cache: Instance ``msal.SerializableTokenCache`` (chargée depuis DuckDB).

    Returns:
        Instance ``msal.PublicClientApplication`` prête à l'emploi.

    Raises:
        MsalUnavailableError: Si MSAL n'est pas installé.
    """
    if _msal_module is None:
        raise MsalUnavailableError("MSAL non installé. Exécutez : pip install msal")

    return _msal_module.PublicClientApplication(
        LEVELUP_CLIENT_ID,
        authority=MSAL_AUTHORITY,
        token_cache=cache,
    )


# =============================================================================
# Acquisition silencieuse (sans UI)
# =============================================================================


def acquire_token_silent(app: Any) -> str | None:
    """Tente l'obtention silencieuse d'un access_token depuis le cache MSAL.

    Ne demande rien à l'utilisateur. Retourne None si aucun compte valide
    ou si le refresh_token a expiré/été révoqué.

    Args:
        app: Instance ``msal.PublicClientApplication``.

    Returns:
        ``access_token`` (str) en cas de succès, ``None`` sinon.
    """
    accounts = app.get_accounts()
    if not accounts:
        logger.debug("acquire_token_silent: aucun compte dans le cache MSAL")
        return None

    result = app.acquire_token_silent(XBOX_SCOPES, account=accounts[0])
    if result and "access_token" in result:
        logger.debug("acquire_token_silent: token obtenu silencieusement")
        return str(result["access_token"])

    error = result.get("error", "unknown") if result else "no_result"
    logger.debug("acquire_token_silent: échec (%s) — Device Code requis", error)
    return None


# =============================================================================
# Device Code Flow
# =============================================================================


def initiate_device_flow(app: Any) -> DeviceCodeInfo:
    """Démarre le Device Code Flow.

    À appeler côté CLI ou Streamlit pour obtenir le code à afficher
    à l'utilisateur.

    Args:
        app: Instance ``msal.PublicClientApplication``.

    Returns:
        ``DeviceCodeInfo`` avec les informations à afficher.

    Raises:
        DeviceFlowError: Si l'initiation échoue (app Azure mal configurée, réseau…).
    """
    flow = app.initiate_device_flow(scopes=XBOX_SCOPES)

    if "error" in flow:
        err = flow.get("error", "unknown")
        desc = flow.get("error_description", "")
        logger.error("initiate_device_flow: échec — %s: %s", err, desc)
        raise DeviceFlowError(
            "bad_client_id",
            f"Échec initiation Device Code Flow : {err} — {desc}",
        )

    user_code = str(flow.get("user_code", ""))
    expires_in = int(flow.get("expires_in", 900))
    logger.info("Device Code Flow initié — code=%s expires_in=%ds", user_code, expires_in)

    return DeviceCodeInfo(
        user_code=user_code,
        verification_url=str(flow.get("verification_uri", "https://microsoft.com/devicelogin")),
        expires_in=expires_in,
        message=str(flow.get("message", "")),
        _flow=flow,
    )


_FLOW_ERROR_MAP: dict[str, str] = {
    "authorization_declined": "declined",
    "expired_token": "timeout",
    "code_expired": "timeout",
    "authorization_pending": "pending",
    "slow_down": "pending",
}


def acquire_token_by_device_flow(app: Any, flow: dict) -> str:
    """Bloque jusqu'à l'obtention du access_token ou l'expiration du code.

    MSAL gère le polling interne (typiquement toutes les 5s).
    À appeler dans un thread dédié côté Streamlit ; directement en CLI.

    Args:
        app: Instance ``msal.PublicClientApplication``.
        flow: Dict opaque MSAL (champ ``_flow`` de ``DeviceCodeInfo``).

    Returns:
        ``access_token`` Microsoft.

    Raises:
        DeviceFlowError: Avec code ``timeout``, ``declined``, ``unknown``.
    """
    logger.debug(
        "acquire_token_by_device_flow: début polling (interval=%ds)", flow.get("interval", 5)
    )
    result = app.acquire_token_by_device_flow(flow)

    if result and "access_token" in result:
        token: str = result["access_token"]
        logger.info("Device Code Flow: access_token obtenu (len=%d)", len(token))
        return token

    error = str(result.get("error", "unknown") if result else "no_result")
    desc = str(result.get("error_description", "") if result else "")
    code = _FLOW_ERROR_MAP.get(error, "unknown")

    if code == "declined":
        logger.error("Device Code Flow: refusé par l'utilisateur")
        raise DeviceFlowError("declined", "L'utilisateur a refusé la connexion Xbox.")

    if code == "timeout":
        user_code = str(flow.get("user_code", "?"))
        logger.warning("Device Code Flow: timeout (user_code=%s)", user_code)
        raise DeviceFlowError("timeout", f"Le code a expiré (user_code={user_code}).")

    logger.error("Device Code Flow: erreur inattendue — %s: %s", error, desc)
    raise DeviceFlowError("unknown", f"Erreur MSAL : {error} — {desc}")
