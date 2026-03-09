"""Composants UI du Device Code Flow pour le wizard Xbox Express.

Extraits de ``setup_wizard.py`` pour limiter la taille du module principal.
"""

from __future__ import annotations

import logging
import os

import streamlit as st

from src.app.session_keys import SK
from src.ui.i18n import t
from src.ui.pages.setup_wizard_logic import save_dc_credentials, validate_dc_credentials

logger = logging.getLogger(__name__)


def render_dc_client_id_form() -> None:
    """Formulaire de saisie du client_id Azure (public client, sans secret)."""
    with st.form("dc_client_id_form"):
        client_id = st.text_input(
            t("setup_dc_client_id_label"),
            placeholder="12345678-1234-1234-1234-123456789abc",
        )
        submitted = st.form_submit_button(t("setup_save_credentials"), type="primary")
    if submitted:
        errors = validate_dc_credentials(client_id)
        if errors:
            for err in errors:
                st.error(err)
        else:
            save_dc_credentials(client_id)
            st.success(t("setup_credentials_saved"))
            st.rerun()


def render_wizard_dc_flow() -> None:
    """Etape 2 du wizard Xbox : démarrage et vérification du Device Code."""
    from src.ui.xbox_oauth_ui import check_dc_queue, reset_device_flow, start_device_flow

    dc_result = check_dc_queue()
    if dc_result is not None:
        reset_device_flow()
        handle_wizard_dc_result(dc_result)
        return

    dc_flow = st.session_state.get(SK.DC_FLOW)
    if dc_flow is not None:
        render_wizard_dc_waiting(dc_flow)
        return

    client_id = os.environ.get("SPNKR_AZURE_CLIENT_ID", "").strip()
    if st.button(t("setup_dc_start_btn"), type="primary", key="wizard_dc_start"):
        try:
            start_device_flow(client_id)
        except Exception as exc:
            st.error(str(exc))
            return
        st.rerun()


def render_wizard_dc_waiting(dc_flow) -> None:
    """Affiche le code Device Code + bouton Vérifier dans le wizard."""
    from src.ui.xbox_oauth_ui import check_dc_queue, reset_device_flow

    st.markdown(
        f"**{t('xbox_dc_code_title')}** "
        f"[{dc_flow.verification_url}]({dc_flow.verification_url})"
    )
    st.code(dc_flow.user_code, language=None)
    col_v, col_c = st.columns([2, 1])
    if col_v.button(t("xbox_dc_verify_btn"), type="primary", key="wizard_dc_verify"):
        dc_result = check_dc_queue()
        if dc_result is None:
            st.info(t("xbox_dc_waiting"))
        else:
            reset_device_flow()
            handle_wizard_dc_result(dc_result)
    if col_c.button(t("xbox_dc_cancel_btn"), key="wizard_dc_cancel"):
        reset_device_flow()
        st.rerun()


def handle_wizard_dc_result(dc_result: dict) -> None:
    """Traite le résultat du Device Code Flow dans le wizard (token ou erreur)."""
    if "error" in dc_result:
        st.error(f"{t('xbox_auth_error')} {dc_result.get('detail', dc_result['error'])}")
        st.rerun()
        return
    token = dc_result["refresh_token"]
    client_id = os.environ.get("SPNKR_AZURE_CLIENT_ID", "").strip()
    with st.spinner(t("setup_dc_getting_profile")):
        from src.app.player_provisioning import provision_player
        from src.ui.xbox_oauth import complete_device_code_flow, store_refresh_token

        result = complete_device_code_flow(token, client_id)
    if "error" in result:
        st.error(f"{t('xbox_auth_error')} {result['error']}")
        st.rerun()
        return
    gt = result["gamertag"]
    db_path = provision_player(gt, result["xuid"])
    store_refresh_token(str(db_path), token)
    st.session_state["_xbox_oauth_result"] = {"gamertag": gt, "xuid": result["xuid"]}
    logger.info("Wizard Device Code Flow complet pour %s", gt)
    st.rerun()
