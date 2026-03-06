"""Page du wizard de configuration initiale.

Affichée automatiquement au premier lancement si la configuration
est incomplète (pas de credentials Azure ou pas de joueur).

Deux parcours au choix :
- **Xbox Express** : connexion Xbox en 1 clic (OAuth automatique)
- **Azure manuel** : configuration classique étape par étape

Séparation UI/logique : toute la logique est dans setup_wizard_logic.py.
"""

from __future__ import annotations

import logging
import os

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

# ── CSS personnalisé pour le wizard ────────────────────────────────────────

_WIZARD_CSS = """
<style>
/* Cartes de choix Xbox / Azure */
div[data-testid="stHorizontalBlock"] .choice-card {
    border-radius: 12px;
    padding: 1.5rem;
    text-align: center;
    transition: transform 0.15s ease, box-shadow 0.15s ease;
}
div[data-testid="stHorizontalBlock"] .choice-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(0,0,0,0.25);
}
.card-xbox {
    background: linear-gradient(135deg, #107C10 0%, #1A9F1A 100%);
    border: 2px solid #0e6b0e;
}
.card-azure {
    background: linear-gradient(135deg, #0078D4 0%, #1490DF 100%);
    border: 2px solid #005a9e;
}
.card-emoji { font-size: 3rem; margin-bottom: 0.5rem; }
.card-title { font-size: 1.3rem; font-weight: 700; color: white; margin-bottom: 0.3rem; }
.card-desc { font-size: 0.9rem; color: rgba(255,255,255,0.85); line-height: 1.4; }
.card-badge {
    display: inline-block;
    background: rgba(255,255,255,0.2);
    border-radius: 999px;
    padding: 0.15rem 0.6rem;
    font-size: 0.75rem;
    font-weight: 600;
    color: white;
    margin-top: 0.5rem;
}

/* Barre de progression */
.progress-bar {
    display: flex; gap: 0.5rem; margin: 1rem 0 1.5rem;
}
.progress-step {
    flex: 1; height: 6px; border-radius: 3px;
    background: rgba(255,255,255,0.1);
}
.progress-step.done { background: #107C10; }
.progress-step.active {
    background: linear-gradient(90deg, #107C10, #1A9F1A);
    animation: pulse-bar 1.5s ease-in-out infinite;
}
@keyframes pulse-bar {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
}

/* Étapes numérotées */
.step-header {
    display: flex; align-items: center; gap: 0.75rem;
    margin-bottom: 0.75rem;
}
.step-number {
    width: 36px; height: 36px; border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    font-weight: 700; font-size: 1rem; color: white;
    flex-shrink: 0;
}
.step-number.pending { background: rgba(255,255,255,0.15); }
.step-number.active { background: #0078D4; }
.step-number.done { background: #107C10; }
.step-title { font-size: 1.1rem; font-weight: 600; }
</style>
"""


def render_setup_wizard_page() -> None:
    """Rend la page du wizard de configuration initiale."""
    st.markdown(_WIZARD_CSS, unsafe_allow_html=True)

    # ── En-tête ────────────────────────────────────────────────────────────
    st.markdown(
        '<div style="text-align:center; padding: 1rem 0 0.5rem;">'
        '<span style="font-size:3rem;">🎮</span><br>'
        f'<span style="font-size:1.8rem; font-weight:700;">{t("setup_title_clean")}</span>'
        "</div>",
        unsafe_allow_html=True,
    )
    st.caption(
        f'<div style="text-align:center;">{t("setup_subtitle")}</div>',
        unsafe_allow_html=True,
    )

    status = get_setup_status()

    if not status.needs_setup:
        st.balloons()
        st.success(t("setup_already_configured"))
        return

    # ── Vérifier si Xbox OAuth a provisionné un joueur (callback réussi) ──
    oauth_result = st.session_state.get("_xbox_oauth_result")
    if oauth_result and "error" not in oauth_result:
        gt = oauth_result.get("gamertag", "")
        st.balloons()
        st.success(t("setup_xbox_provisioned", gamertag=gt))
        st.info(t("setup_xbox_sync_hint", gamertag=gt))
        cmd = get_sync_command(gt, 200)
        st.code(" ".join(cmd), language="bash")
        st.session_state.pop("_xbox_oauth_result", None)
        return

    # ── Sélection du mode ──────────────────────────────────────────────────
    mode = st.session_state.get("_setup_mode")

    if mode is None:
        _render_mode_selection()
    elif mode == "xbox":
        _render_xbox_flow(status)
    elif mode == "azure":
        _render_azure_flow(status)


# =============================================================================
# Sélection du mode (écran d'accueil)
# =============================================================================


def _render_mode_selection() -> None:
    """Affiche les deux cartes de choix Xbox / Azure."""
    st.markdown("")  # spacer

    col_xbox, col_azure = st.columns(2, gap="large")

    with col_xbox:
        st.markdown(
            '<div class="choice-card card-xbox">'
            '<div class="card-emoji">🎮</div>'
            f'<div class="card-title">{t("setup_xbox_card_title")}</div>'
            f'<div class="card-desc">{t("setup_xbox_card_desc")}</div>'
            f'<div class="card-badge">{t("setup_xbox_card_badge")}</div>'
            "</div>",
            unsafe_allow_html=True,
        )
        if st.button(
            t("setup_xbox_card_btn"),
            key="choose_xbox",
            use_container_width=True,
            type="primary",
        ):
            st.session_state["_setup_mode"] = "xbox"
            st.rerun()

    with col_azure:
        st.markdown(
            '<div class="choice-card card-azure">'
            '<div class="card-emoji">☁️</div>'
            f'<div class="card-title">{t("setup_azure_card_title")}</div>'
            f'<div class="card-desc">{t("setup_azure_card_desc")}</div>'
            f'<div class="card-badge">{t("setup_azure_card_badge")}</div>'
            "</div>",
            unsafe_allow_html=True,
        )
        if st.button(
            t("setup_azure_card_btn"),
            key="choose_azure",
            use_container_width=True,
        ):
            st.session_state["_setup_mode"] = "azure"
            st.rerun()

    # Pied de page
    st.markdown("")
    st.caption(t("setup_footer_note"))


# =============================================================================
# Parcours Xbox Express
# =============================================================================


def _render_xbox_flow(status) -> None:
    """Parcours Xbox en 2 étapes : credentials Azure + connexion Xbox."""
    _render_back_button()

    total_steps = 2
    current = 1 if not status.auth.has_credentials else 2
    _render_progress_bar(current, total_steps)

    # ── Étape 1 : Credentials Azure (toujours nécessaires) ────────────────
    _render_step_header(1, t("setup_xbox_step1_title"), current)

    if status.auth.has_credentials:
        st.success(t("setup_credentials_ok"))
    else:
        st.markdown(t("setup_xbox_step1_help"))
        _render_azure_form(redirect_default="http://localhost:8501")
        return  # bloquer tant que pas rempli

    st.divider()

    # ── Étape 2 : Bouton "Se connecter avec Xbox" ─────────────────────────
    _render_step_header(2, t("setup_xbox_step2_title"), current)

    st.markdown(t("setup_xbox_step2_help"))

    client_id = os.environ.get("SPNKR_AZURE_CLIENT_ID", "").strip()
    redirect_uri = os.environ.get("SPNKR_AZURE_REDIRECT_URI", "").strip() or "http://localhost:8501"

    if client_id:
        from src.ui.xbox_oauth import build_xbox_auth_url, generate_oauth_state

        if "_xbox_oauth_state" not in st.session_state:
            st.session_state["_xbox_oauth_state"] = generate_oauth_state()

        state = st.session_state["_xbox_oauth_state"]
        auth_url = build_xbox_auth_url(client_id, redirect_uri, state)

        st.markdown("")
        st.link_button(
            t("setup_xbox_connect_btn"),
            url=auth_url,
            use_container_width=True,
        )
        st.caption(t("setup_xbox_redirect_note", redirect_uri=redirect_uri))
    else:
        st.warning(t("setup_credentials_missing"))


# =============================================================================
# Parcours Azure classique
# =============================================================================


def _render_azure_flow(status) -> None:
    """Parcours Azure classique en 3 étapes."""
    _render_back_button()

    total_steps = 3
    current = status.current_step or 3
    _render_progress_bar(current, total_steps)

    # ── Étape 1 : Credentials ─────────────────────────────────────────────
    _render_step_header(1, t("setup_step1_title"), current)

    if status.auth.has_credentials:
        st.success(t("setup_credentials_ok"))
    else:
        st.markdown(t("setup_step1_help"))
        _render_azure_form()
        return

    st.divider()

    # ── Étape 2 : Token ───────────────────────────────────────────────────
    _render_step_header(2, t("setup_step2_title"), current)

    if status.auth.has_refresh_token:
        st.success(t("setup_token_ok"))
    elif not status.auth.has_credentials:
        st.info("⬆️ " + t("setup_step1_title"))
        return
    else:
        st.markdown(t("setup_step2_help"))
        script_path = get_token_script_path()
        st.markdown(t("setup_token_command"))
        st.code(
            f"python {script_path.relative_to(script_path.parent.parent)}",
            language="bash",
        )
        _render_token_form()
        return

    st.divider()

    # ── Étape 3 : Joueur ──────────────────────────────────────────────────
    _render_step_header(3, t("setup_step3_title"), current)

    if status.has_players:
        st.success(t("setup_profile_created", gamertag=f"{status.player_count} joueur(s)"))
    elif not status.auth.has_credentials:
        st.info("⬆️ " + t("setup_step1_title"))
    else:
        st.markdown(t("setup_step3_help"))
        _render_player_form()


# =============================================================================
# Composants réutilisables
# =============================================================================


def _render_back_button() -> None:
    """Bouton retour vers l'écran de sélection de mode."""
    if st.button(t("setup_back_btn"), key="setup_back"):
        st.session_state.pop("_setup_mode", None)
        st.rerun()


def _render_progress_bar(current: int, total: int) -> None:
    """Barre de progression stylisée."""
    steps_html = ""
    for i in range(1, total + 1):
        if i < current:
            cls = "done"
        elif i == current:
            cls = "active"
        else:
            cls = ""
        steps_html += f'<div class="progress-step {cls}"></div>'

    st.markdown(f'<div class="progress-bar">{steps_html}</div>', unsafe_allow_html=True)


def _render_step_header(step_num: int, title: str, current: int) -> None:
    """En-tête d'étape avec numéro stylisé."""
    if step_num < current:
        cls = "done"
    elif step_num == current:
        cls = "active"
    else:
        cls = "pending"

    st.markdown(
        f'<div class="step-header">'
        f'<div class="step-number {cls}">{step_num}</div>'
        f'<div class="step-title">{title}</div>'
        f"</div>",
        unsafe_allow_html=True,
    )


def _render_azure_form(redirect_default: str = "https://localhost") -> None:
    """Formulaire de saisie des credentials Azure."""
    with st.form("azure_credentials_form"):
        client_id = st.text_input(
            t("setup_client_id"),
            placeholder="12345678-1234-1234-1234-123456789abc",
        )
        client_secret = st.text_input(
            t("setup_client_secret"),
            type="password",
            placeholder="~xY8Q.secret.value",
        )
        redirect_uri = st.text_input(
            t("setup_redirect_uri"),
            value=redirect_default,
        )
        submitted = st.form_submit_button(t("setup_save_credentials"), type="primary")

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


def _render_token_form() -> None:
    """Formulaire de saisie du refresh token."""
    with st.form("token_form"):
        token = st.text_input(t("setup_token_paste"), type="password")
        submitted = st.form_submit_button(t("setup_save_token"), type="primary")

    if submitted:
        token = token.strip()
        if not token:
            st.warning(t("setup_token_empty"))
        else:
            from src.utils.auth import write_env_local

            write_env_local({"SPNKR_OAUTH_REFRESH_TOKEN": token})
            os.environ["SPNKR_OAUTH_REFRESH_TOKEN"] = token
            st.success(t("setup_token_saved"))
            logger.info("Wizard : token OAuth sauvegardé")
            st.rerun()


def _render_player_form() -> None:
    """Formulaire de création de profil joueur."""
    with st.form("player_form"):
        gamertag = st.text_input(t("setup_gamertag"), placeholder="MonGamertag")
        max_matches = st.slider(
            t("setup_max_matches"),
            min_value=50,
            max_value=1000,
            value=200,
            step=50,
        )
        submitted = st.form_submit_button(t("setup_create_profile"), type="primary")

    if submitted:
        errors = validate_gamertag(gamertag)
        if errors:
            for err in errors:
                st.error(err)
        else:
            profile_key = create_player_profile(gamertag)
            st.success(t("setup_profile_created", gamertag=profile_key))
            logger.info("Wizard : profil joueur créé : %s", profile_key)
            st.markdown(t("setup_sync_instructions"))
            cmd = get_sync_command(profile_key, max_matches)
            st.code(" ".join(cmd), language="bash")
            st.info(t("setup_sync_done_hint"))
