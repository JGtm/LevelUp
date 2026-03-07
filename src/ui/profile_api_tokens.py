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


async def get_tokens(  # noqa: C901, PLR0915
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
        oauth_refresh_token = _load_refresh_token_from_db(gamertag) or oauth_refresh_token

    if not (azure_client_id and azure_client_secret and oauth_refresh_token):
        raise RuntimeError(
            "Tokens manquants: définis soit SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN, "
            "soit SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET + SPNKR_OAUTH_REFRESH_TOKEN."
        )

    from src.data.sync._auth import refresh_halo_tokens

    tokens = await refresh_halo_tokens(
        session,
        client_id=azure_client_id,
        client_secret=azure_client_secret,
        redirect_uri=azure_redirect_uri,
        refresh_token=oauth_refresh_token,
    )
    os.environ["SPNKR_SPARTAN_TOKEN"] = tokens.spartan_token
    os.environ["SPNKR_CLEARANCE_TOKEN"] = tokens.clearance_token
    return tokens.spartan_token, tokens.clearance_token


def _load_refresh_token_from_db(gamertag: str) -> str:
    """Charge le refresh token depuis stats.duckdb du joueur (fallback Xbox OAuth)."""
    try:
        from src.ui.xbox_oauth import load_refresh_token as _load_rt
        from src.utils.paths import REPO_ROOT as _repo_root_path

        _player_db = _repo_root_path / "data" / "players" / gamertag / "stats.duckdb"
        if _player_db.exists():
            token = _load_rt(_player_db)
            return token or ""
    except Exception:
        pass
    return ""


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
