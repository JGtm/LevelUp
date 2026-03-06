"""Page du wizard de configuration initiale.

Affichée automatiquement au premier lancement si la configuration
est incomplète (pas de credentials Azure ou pas de joueur).

Séparation UI/logique : toute la logique est dans setup_wizard_logic.py.
"""

from __future__ import annotations

import logging

import streamlit as st

from src.ui.i18n import t
from src.ui.pages.setup_wizard_logic import (
    create_player_profile,
    get_setup_status,
    get_sync_command,
    get_token_script_path,
    save_azure_credentials,
    validate_azure_credentials,
    validate_gamertag,
)

logger = logging.getLogger(__name__)


def render_setup_wizard_page() -> None:
    """Rend la page du wizard de configuration initiale."""
    st.title(t("setup_title"))
    st.markdown(t("setup_welcome"))
    st.divider()

    status = get_setup_status()

    if not status.needs_setup:
        st.success(t("setup_already_configured"))
        return

    # ── Étape 1 : Credentials Azure ────────────────────────────────────
    _render_step_azure(status)

    st.divider()

    # ── Étape 2 : Token OAuth ──────────────────────────────────────────
    _render_step_token(status)

    st.divider()

    # ── Étape 3 : Ajout joueur ─────────────────────────────────────────
    _render_step_player(status)


def _render_step_azure(status) -> None:
    """Rend l'étape 1 : credentials Azure."""
    st.subheader(t("setup_step1_title"))

    if status.auth.has_credentials:
        st.success(t("setup_credentials_ok"))
        return

    st.markdown(t("setup_step1_help"))

    with st.form("azure_credentials_form"):
        client_id = st.text_input(
            t("setup_client_id"),
            placeholder="12345678-1234-1234-1234-123456789abc",
        )
        client_secret = st.text_input(
            t("setup_client_secret"),
            type="password",
            placeholder="votre_secret_client",
        )
        redirect_uri = st.text_input(
            t("setup_redirect_uri"),
            value="https://localhost",
        )

        submitted = st.form_submit_button(t("setup_save_credentials"))

    if submitted:
        errors = validate_azure_credentials(client_id, client_secret)
        if errors:
            for err in errors:
                st.error(err)
        else:
            save_azure_credentials(client_id, client_secret, redirect_uri)
            st.success(t("setup_credentials_saved"))
            logger.info("Wizard : credentials Azure sauvegardées")
            st.rerun()


def _render_step_token(status) -> None:
    """Rend l'étape 2 : token OAuth."""
    st.subheader(t("setup_step2_title"))

    if status.auth.has_refresh_token:
        st.success(t("setup_token_ok"))
        return

    if not status.auth.has_credentials:
        st.info("⬆️ " + t("setup_step1_title"))
        return

    st.markdown(t("setup_step2_help"))

    # Afficher la commande à lancer
    script_path = get_token_script_path()
    st.markdown(t("setup_token_command"))
    st.code(f"python {script_path.relative_to(script_path.parent.parent)}", language="bash")

    # Champ pour coller le token
    with st.form("token_form"):
        token = st.text_input(
            t("setup_token_paste"),
            type="password",
        )
        submitted = st.form_submit_button(t("setup_save_token"))

    if submitted:
        token = token.strip()
        if not token:
            st.warning(t("setup_token_empty"))
        else:
            from src.utils.auth import write_env_local

            write_env_local({"SPNKR_OAUTH_REFRESH_TOKEN": token})
            import os

            os.environ["SPNKR_OAUTH_REFRESH_TOKEN"] = token
            st.success(t("setup_token_saved"))
            logger.info("Wizard : token OAuth sauvegardé")
            st.rerun()


def _render_step_player(status) -> None:
    """Rend l'étape 3 : ajout d'un joueur."""
    st.subheader(t("setup_step3_title"))

    if status.has_players:
        st.success(t("setup_profile_created", gamertag=f"{status.player_count} joueur(s)"))
        return

    if not status.auth.has_credentials:
        st.info("⬆️ " + t("setup_step1_title"))
        return

    st.markdown(t("setup_step3_help"))

    with st.form("player_form"):
        gamertag = st.text_input(
            t("setup_gamertag"),
            placeholder="MonGamertag",
        )
        max_matches = st.slider(
            t("setup_max_matches"),
            min_value=50,
            max_value=1000,
            value=200,
            step=50,
        )
        submitted = st.form_submit_button(t("setup_create_profile"))

    if submitted:
        errors = validate_gamertag(gamertag)
        if errors:
            for err in errors:
                st.error(err)
        else:
            profile_key = create_player_profile(gamertag)
            st.success(t("setup_profile_created", gamertag=profile_key))
            logger.info("Wizard : profil joueur créé : %s", profile_key)

            # Afficher la commande de sync
            st.markdown(t("setup_sync_instructions"))
            cmd = get_sync_command(profile_key, max_matches)
            st.code(" ".join(cmd), language="bash")
            st.info(t("setup_sync_done_hint"))
