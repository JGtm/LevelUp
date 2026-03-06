"""Gestion des tokens d'authentification SPNKr/Halo Waypoint.

Chargement des variables d'environnement et refresh des tokens via Azure.
"""

from __future__ import annotations

import os

from src.utils.async_compat import run_sync as _run_sync_compat
from src.utils.env import load_dotenv_if_present as _load_dotenv_if_present
from src.utils.env import normalize_gamertag_for_env as _normalize_gamertag_for_env


def _is_probable_auth_error(err: Exception) -> bool:
    """Détecte si une erreur est probablement due à un problème d'authentification."""
    msg = str(err or "")
    m = msg.lower()
    # Heuristique: SPNKr/aiohttp remontent souvent 401/unauthorized dans le message.
    return ("401" in m) or ("unauthorized" in m) or ("forbidden" in m and "403" in m)


async def get_tokens(
    session,
    *,
    spartan_token: str | None,
    clearance_token: str | None,
    timeout_seconds: int,
    gamertag: str | None = None,
) -> tuple[str, str]:
    """Récupère ou rafraîchit les tokens SPNKr.

    Si ``gamertag`` est fourni, cherche d'abord le refresh token propre au
    joueur (``SPNKR_OAUTH_REFRESH_TOKEN_<GT_NORMALISÉ>``) avant d'utiliser le
    token global.  Ceci est nécessaire pour les endpoints economy player-gated
    (career rank, customisation privée).

    Args:
        session: Session aiohttp.
        spartan_token: Token Spartan existant (ou None pour en obtenir un nouveau).
        clearance_token: Token Clearance existant (ou None pour en obtenir un nouveau).
        timeout_seconds: Timeout pour les appels réseau.
        gamertag: Gamertag du joueur pour résolution du token per-player (optionnel).

    Returns:
        Tuple (spartan_token, clearance_token).

    Raises:
        RuntimeError: Si les tokens ne peuvent pas être obtenus.
    """
    _load_dotenv_if_present()
    st = str(spartan_token or "").strip()
    ct = str(clearance_token or "").strip()
    if st and ct:
        # Rend les tokens accessibles aux autres helpers (ex: téléchargement d'assets).
        os.environ["SPNKR_SPARTAN_TOKEN"] = st
        os.environ["SPNKR_CLEARANCE_TOKEN"] = ct
        return st, ct

    azure_client_id = str(os.environ.get("SPNKR_AZURE_CLIENT_ID") or "").strip()
    azure_client_secret = str(os.environ.get("SPNKR_AZURE_CLIENT_SECRET") or "").strip()
    azure_redirect_uri = (
        str(os.environ.get("SPNKR_AZURE_REDIRECT_URI") or "").strip() or "https://localhost"
    )

    # Priorité : refresh token propre au joueur si gamertag fourni
    oauth_refresh_token = ""
    if gamertag:
        player_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{_normalize_gamertag_for_env(gamertag)}"
        oauth_refresh_token = str(os.environ.get(player_key) or "").strip()

    # Fallback 2 : refresh token global en env
    if not oauth_refresh_token:
        oauth_refresh_token = str(os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN") or "").strip()

    # Fallback 3 : refresh token stocké en DB (connexion Xbox OAuth via Streamlit UI)
    if not oauth_refresh_token and gamertag:
        try:
            from src.ui.xbox_oauth import load_refresh_token as _load_rt

            _root = _repo_root()
            _player_db = _root / "data" / "players" / gamertag / "stats.duckdb"
            if _player_db.exists():
                _token_from_db = _load_rt(_player_db)
                if _token_from_db:
                    oauth_refresh_token = _token_from_db
        except Exception:
            pass

    if not (azure_client_id and azure_client_secret and oauth_refresh_token):
        raise RuntimeError(
            "Tokens manquants: définis soit SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN, "
            "soit SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET + SPNKR_OAUTH_REFRESH_TOKEN."
        )

    from spnkr import AzureApp, refresh_player_tokens

    async def _refresh_oauth_access_token_v2(refresh_token: str, app: AzureApp) -> str:
        url = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
        data = {
            "client_id": app.client_id,
            "client_secret": app.client_secret,
            "grant_type": "refresh_token",
            "refresh_token": refresh_token,
            "scope": "Xboxlive.signin Xboxlive.offline_access",
        }
        resp = await session.post(url, data=data)
        payload = await resp.json()
        if resp.status >= 400:
            raise RuntimeError(
                "Échec refresh OAuth v2 (consumers). "
                f"status={resp.status} error={payload.get('error')} desc={payload.get('error_description')}"
            )
        access = payload.get("access_token")
        if not isinstance(access, str) or not access.strip():
            raise RuntimeError("OAuth v2: pas de access_token dans la réponse.")
        return access.strip()

    app = AzureApp(azure_client_id, azure_client_secret, azure_redirect_uri)
    try:
        player = await refresh_player_tokens(session, app, oauth_refresh_token)
        st2, ct2 = str(player.spartan_token.token), str(player.clearance_token.token)
        os.environ["SPNKR_SPARTAN_TOKEN"] = st2
        os.environ["SPNKR_CLEARANCE_TOKEN"] = ct2
        return st2, ct2
    except Exception as e:
        msg = str(e)
        if "invalid_client" not in msg or "client_secret" not in msg:
            raise

        # Fallback: endpoint OAuth v2 (consumers) -> chain Xbox/XSTS/Halo
        from spnkr.auth.core import XSTS_V3_HALO_AUDIENCE, XSTS_V3_XBOX_AUDIENCE
        from spnkr.auth.halo import request_clearance_token, request_spartan_token
        from spnkr.auth.xbox import request_user_token, request_xsts_token

        access_token = await _refresh_oauth_access_token_v2(oauth_refresh_token, app)
        user_token = await request_user_token(session, access_token)
        _ = await request_xsts_token(session, user_token.token, XSTS_V3_XBOX_AUDIENCE)
        halo_xsts_token = await request_xsts_token(session, user_token.token, XSTS_V3_HALO_AUDIENCE)
        spartan = await request_spartan_token(session, halo_xsts_token.token)
        clearance = await request_clearance_token(session, spartan.token)
        st3, ct3 = str(spartan.token), str(clearance.token)
        os.environ["SPNKR_SPARTAN_TOKEN"] = st3
        os.environ["SPNKR_CLEARANCE_TOKEN"] = ct3
        return st3, ct3


def ensure_spnkr_tokens(*, timeout_seconds: int = 12) -> tuple[bool, str | None]:
    """Best-effort: s'assure que SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN sont disponibles.

    Utile quand on utilise le cache d'apparence (donc pas d'appel API au run courant),
    mais qu'on veut quand même télécharger des assets /hi/Images/file/ qui exigent une auth.

    Returns:
        Tuple (ok, error_message).
    """

    _run_sync = _run_sync_compat

    async def _run() -> None:
        import aiohttp

        st = str(os.environ.get("SPNKR_SPARTAN_TOKEN") or "").strip() or None
        ct = str(os.environ.get("SPNKR_CLEARANCE_TOKEN") or "").strip() or None

        timeout = aiohttp.ClientTimeout(total=float(timeout_seconds))
        async with aiohttp.ClientSession(timeout=timeout) as session:
            await get_tokens(
                session,
                spartan_token=st,
                clearance_token=ct,
                timeout_seconds=int(timeout_seconds),
            )

    try:
        _run_sync(_run())
        st = str(os.environ.get("SPNKR_SPARTAN_TOKEN") or "").strip()
        ct = str(os.environ.get("SPNKR_CLEARANCE_TOKEN") or "").strip()
        if st and ct:
            return True, None
        return False, "Tokens SPNKr introuvables (env et Azure refresh non configurés)."
    except Exception as e:
        return False, str(e)
