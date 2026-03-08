"""Composant UI Streamlit pour la connexion Xbox via OAuth.

Affiche une section "Connexion Xbox" dans la page Paramètres.
La logique métier est dans ``src/ui/xbox_oauth.py`` (testable sans Streamlit).
"""

from __future__ import annotations

import logging
import os

import streamlit as st

from src.app.session_keys import SK
from src.ui.i18n import t

logger = logging.getLogger(__name__)

# Clé session_state pour le token CSRF anti-replay
_STATE_KEY = SK.XBOX_OAUTH_STATE
# Clé session_state pour afficher le résultat du callback
_RESULT_KEY = SK.XBOX_OAUTH_RESULT


def _get_azure_config() -> tuple[str, str, str]:
    """Lit les variables Azure depuis l'environnement.

    Returns:
        Tuple ``(client_id, client_secret, redirect_uri)``.
        Les valeurs manquantes sont des chaînes vides.
    """
    client_id = str(os.environ.get("SPNKR_AZURE_CLIENT_ID") or "").strip()
    client_secret = str(os.environ.get("SPNKR_AZURE_CLIENT_SECRET") or "").strip()
    redirect_uri = (
        str(os.environ.get("SPNKR_AZURE_REDIRECT_URI") or "").strip() or "http://localhost:8501"
    )
    return client_id, client_secret, redirect_uri


def _get_current_db_path() -> str | None:
    """Retourne le db_path du joueur courant depuis session_state."""
    db = st.session_state.get("db_path")
    return str(db).strip() if db and str(db).strip() else None


def _get_current_gamertag() -> str | None:
    """Retourne le gamertag courant depuis session_state."""
    gt = st.session_state.get("waypoint_player") or st.session_state.get("xuid_input")
    return str(gt).strip() if gt and str(gt).strip() else None


def render_xbox_login_section() -> None:
    """Rend la section 'Connexion Xbox' dans la page Paramètres.

    Affiche selon l'état :
    - Configuration Azure manquante → instructions de setup
    - Joueur déjà connecté (refresh_token en sync_meta) → statut + déconnexion
    - Non connecté → bouton "Se connecter avec Xbox"
    """
    client_id, client_secret, redirect_uri = _get_azure_config()

    # ── 1. Configuration Azure absente ────────────────────────────────────
    if not client_id or not client_secret:
        st.info(t("xbox_auth_missing_config"))
        with st.expander(t("xbox_auth_setup_help_title"), expanded=False):
            st.markdown(t("xbox_auth_setup_help_body"))
        return

    # ── 2. Afficher le résultat du dernier callback OAuth ─────────────────
    oauth_result = st.session_state.pop(_RESULT_KEY, None)
    if oauth_result:
        if "error" in oauth_result:
            st.error(f"{t('xbox_auth_error')} {oauth_result['error']}")
        else:
            gt = oauth_result.get("gamertag", "")
            st.success(t("xbox_auth_success").format(gamertag=gt))

    # ── 3. Statut du joueur courant ────────────────────────────────────────
    db_path = _get_current_db_path()
    gamertag = _get_current_gamertag()
    has_token = False

    if db_path:
        from src.ui.xbox_oauth import load_refresh_token

        token_in_db = load_refresh_token(db_path)
        has_token = bool(token_in_db)

    if has_token and gamertag:
        # Connecté
        st.success(f"🎮 {t('xbox_connected_as').format(gamertag=gamertag)}")
        col_a, col_b = st.columns([3, 1])
        col_a.caption(t("xbox_token_stored"))
        if col_b.button(t("xbox_disconnect"), key="xbox_disconnect_btn"):
            _revoke_local_token(db_path)
            st.rerun()
        return

    # ── 4. Bouton de connexion ─────────────────────────────────────────────
    st.caption(t("xbox_auth_intro"))

    # Générer un state CSRF si absent
    if _STATE_KEY not in st.session_state:
        from src.ui.xbox_oauth import generate_oauth_state

        st.session_state[_STATE_KEY] = generate_oauth_state()

    state = str(st.session_state[_STATE_KEY])

    from src.ui.xbox_oauth import build_xbox_auth_url

    auth_url = build_xbox_auth_url(client_id, redirect_uri, state)

    st.link_button(
        label=t("xbox_connect_btn"),
        url=auth_url,
        help=t("xbox_connect_help"),
    )

    st.caption(t("xbox_redirect_notice").format(redirect_uri=redirect_uri))


def _revoke_local_token(db_path: str | None) -> None:
    """Supprime le refresh_token stocké localement (déconnexion).

    Ne révoque pas le token côté Microsoft — l'utilisateur doit le faire
    depuis son compte Microsoft si nécessaire.

    Args:
        db_path: Chemin vers stats.duckdb du joueur. Si None, ne fait rien.
    """
    if not db_path:
        return

    try:
        from src.utils.db import duckdb_read_write

        with duckdb_read_write(db_path) as conn:
            conn.execute("DELETE FROM sync_meta WHERE key = 'oauth_refresh_token'")
        logger.info("Refresh token OAuth supprimé depuis %s", db_path)
    except Exception as exc:
        logger.warning("Impossible de supprimer le refresh token: %s", exc)


def handle_pending_xbox_result(gamertag: str, xuid: str, db_path: str, refresh_token: str) -> None:
    """Enregistre les données OAuth dans session_state après un callback réussi.

    Appelé depuis ``streamlit_app.py`` après l'échange du code OAuth.

    Args:
        gamertag: Gamertag Xbox résolu.
        xuid: XUID Xbox Live du joueur.
        db_path: Chemin vers stats.duckdb du joueur.
        refresh_token: Token de rafraîchissement OAuth à stocker.
    """
    from src.ui.xbox_oauth import store_refresh_token

    store_refresh_token(db_path, refresh_token)
    st.session_state[_RESULT_KEY] = {"gamertag": gamertag, "xuid": xuid}

    # Nettoyer le state CSRF
    st.session_state.pop(_STATE_KEY, None)

    logger.info("Connexion Xbox OAuth complète pour %s (xuid=%s)", gamertag, xuid)
