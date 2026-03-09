"""Composant UI Streamlit — Connexion Xbox via Device Code Flow (MSAL).

La logique métier est dans ``src/utils/msal_device_flow.py`` et
``src/ui/xbox_oauth.py`` (testables sans Streamlit).
"""

from __future__ import annotations

import logging
import queue
import threading

import streamlit as st

from src.app.session_keys import SK
from src.ui.i18n import t

logger = logging.getLogger(__name__)

# Clés session_state internes (module-private)
_RESULT_KEY = SK.XBOX_OAUTH_RESULT
_DC_FLOW_KEY = SK.DC_FLOW
_DC_APP_KEY = SK.DC_MSAL_APP
_DC_QUEUE_KEY = SK.DC_RESULT_QUEUE
_DC_CLIENT_ID_KEY = SK.DC_CLIENT_ID


def _get_current_db_path() -> str | None:
    """Retourne le db_path du joueur courant depuis session_state."""
    db = st.session_state.get("db_path")
    return str(db).strip() if db and str(db).strip() else None


def _get_current_gamertag() -> str | None:
    """Retourne le gamertag courant depuis session_state."""
    gt = st.session_state.get("waypoint_player") or st.session_state.get("xuid_input")
    return str(gt).strip() if gt and str(gt).strip() else None


def check_dc_queue() -> dict | None:
    """Vérifie la queue du device flow sans bloquer. Retourne le résultat si prêt."""
    q = st.session_state.get(_DC_QUEUE_KEY)
    if q is None:
        return None
    try:
        return q.get_nowait()
    except queue.Empty:
        return None


def reset_device_flow() -> None:
    """Nettoie l'état du device flow en cours."""
    for key in (_DC_FLOW_KEY, _DC_APP_KEY, _DC_QUEUE_KEY, _DC_CLIENT_ID_KEY):
        st.session_state.pop(key, None)


def start_device_flow(client_id: str) -> None:
    """Initie le device flow MSAL et démarre le thread de polling."""
    from src.utils.msal_device_flow import (
        DeviceFlowError,
        acquire_token_blocking,
        initiate_device_flow,
    )

    result, app = initiate_device_flow(client_id)
    q: queue.Queue[dict] = queue.Queue(maxsize=1)

    def _poll() -> None:
        try:
            token = acquire_token_blocking(app, result._flow)
            q.put({"refresh_token": token})
            logger.info("Device flow thread : token obtenu (client_id=%s…)", client_id[:8])
        except DeviceFlowError as exc:
            logger.warning("Device flow thread : erreur=%s — %s", exc.code, exc.detail)
            q.put({"error": exc.code, "detail": exc.detail})
        except Exception as exc:  # noqa: BLE001
            logger.error("Device flow thread : exception inattendue: %s", exc)
            q.put({"error": "unknown", "detail": str(exc)})

    thread = threading.Thread(target=_poll, daemon=True, name="msal-device-flow")
    thread.start()

    st.session_state[_DC_APP_KEY] = app
    st.session_state[_DC_FLOW_KEY] = result
    st.session_state[_DC_QUEUE_KEY] = q
    st.session_state[_DC_CLIENT_ID_KEY] = client_id
    logger.info("Device flow initié (client_id=%s… user_code=%s)", client_id[:8], result.user_code)


def render_xbox_login_section() -> None:
    """Rend la section 'Connexion Xbox' dans la page Paramètres."""
    # ── Afficher le résultat du dernier callback ────────────────────────────────
    oauth_result = st.session_state.pop(_RESULT_KEY, None)
    if oauth_result:
        if "error" in oauth_result:
            st.error(f"{t('xbox_auth_error')} {oauth_result.get('detail', oauth_result['error'])}")
        else:
            gt = oauth_result.get("gamertag", "")
            st.success(t("xbox_auth_success").format(gamertag=gt))

    db_path = _get_current_db_path()
    gamertag = _get_current_gamertag()

    # ── Déjà connecté ─────────────────────────────────────────────────────────────────────
    if db_path:
        from src.ui.xbox_oauth import load_refresh_token

        if load_refresh_token(db_path):
            st.success(f"🎮 {t('xbox_connected_as').format(gamertag=gamertag or '?')}")
            col_a, col_b = st.columns([3, 1])
            col_a.caption(t("xbox_token_stored"))
            if col_b.button(t("xbox_disconnect"), key="xbox_disconnect_btn"):
                _revoke_local_token(db_path)
                reset_device_flow()
                st.rerun()
            return

    # ── Vérifier si un flow en cours vient de terminer ─────────────────────────────────────
    dc_result = check_dc_queue()
    if dc_result is not None:
        reset_device_flow()
        if "error" in dc_result:
            logger.warning("Connexion Xbox Device Code : erreur=%s", dc_result.get("error"))
            st.error(f"{t('xbox_auth_error')} {dc_result.get('detail', dc_result['error'])}")
        else:
            st.success(t("xbox_dc_token_ready"))
            st.session_state["_dc_refresh_token_pending"] = dc_result["refresh_token"]
        st.rerun()
        return

    # ── Flow en cours ──────────────────────────────────────────────────────────────────────────
    dc_flow = st.session_state.get(_DC_FLOW_KEY)
    if dc_flow is not None:
        _render_dc_waiting(dc_flow)
        return

    # ── Pas de flow : afficher la section d'initiation ─────────────────────────────────
    st.caption(t("xbox_auth_intro"))
    _render_dc_start()


def _render_dc_start() -> None:
    """Formulaire de saisie du client_id pour démarrer le device flow."""
    with st.form("dc_start_form"):
        client_id = st.text_input(
            t("xbox_dc_client_id_label"),
            placeholder="12345678-1234-1234-1234-123456789abc",
            help=t("xbox_dc_client_id_help"),
        )
        submitted = st.form_submit_button(t("xbox_dc_start_btn"), type="primary")

    if submitted:
        client_id = client_id.strip()
        if not client_id:
            st.warning(t("xbox_dc_client_id_empty"))
            return
        try:
            start_device_flow(client_id)
        except Exception as exc:
            logger.error("Échec initiation device flow: %s", exc)
            st.error(str(exc))
            return
        st.rerun()


def _render_dc_waiting(dc_flow) -> None:
    """Affiche le code device + bouton pour vérifier."""
    from src.utils.msal_device_flow import DeviceCodeResult

    result: DeviceCodeResult = dc_flow

    st.markdown(
        f"**{t('xbox_dc_code_title')}** " f"[{result.verification_url}]({result.verification_url})"
    )
    st.code(result.user_code, language=None)

    col_verify, col_cancel = st.columns([2, 1])
    if col_verify.button(t("xbox_dc_verify_btn"), type="primary", key="dc_verify"):
        dc_result = check_dc_queue()
        if dc_result is None:
            st.info(t("xbox_dc_waiting"))
        elif "error" in dc_result:
            reset_device_flow()
            st.error(f"{t('xbox_auth_error')} {dc_result.get('detail', dc_result['error'])}")
            st.rerun()
        else:
            st.session_state["_dc_refresh_token_pending"] = dc_result["refresh_token"]
            reset_device_flow()
            st.success(t("xbox_dc_token_ready"))
            st.rerun()

    if col_cancel.button(t("xbox_dc_cancel_btn"), key="dc_cancel"):
        reset_device_flow()
        st.rerun()


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
    """Enregistre les données OAuth dans session_state après un flow réussi.

    Args:
        gamertag: Gamertag Xbox résolu.
        xuid: XUID Xbox Live du joueur.
        db_path: Chemin vers stats.duckdb du joueur.
        refresh_token: Token de rafraîchissement OAuth à stocker.
    """
    from src.ui.xbox_oauth import store_refresh_token

    store_refresh_token(db_path, refresh_token)
    st.session_state[_RESULT_KEY] = {"gamertag": gamertag, "xuid": xuid}
    logger.info("Connexion Xbox Device Code complète pour %s (xuid=%s)", gamertag, xuid)
