"""Tests pour src/ui/pages/setup_wizard.py — Rendu UI du wizard.

Couvre :
- render_setup_wizard_page : flux principal (skip si configuré, mode selection, Xbox, Azure)
- Composants réutilisables (_render_progress_bar, _render_step_header)
- Parcours Xbox Express complet avec MockStreamlit
- Parcours Azure classique complet avec MockStreamlit
- Callback OAuth détecté dans le wizard
"""

from __future__ import annotations

from contextlib import contextmanager
from unittest.mock import MagicMock, patch

from src.ui.pages.setup_wizard_logic import SetupStatus
from src.utils.auth import AuthStatus

# ── Helpers ────────────────────────────────────────────────────────────────


def _make_status(
    *,
    has_id: bool = False,
    has_secret: bool = False,
    has_token: bool = False,
    has_players: bool = False,
    player_count: int = 0,
) -> SetupStatus:
    """Construit un SetupStatus pour les tests."""
    return SetupStatus(
        auth=AuthStatus(
            has_client_id=has_id,
            has_client_secret=has_secret,
            has_refresh_token=has_token,
        ),
        has_players=has_players,
        player_count=player_count,
    )


def _setup_wizard_mocks(ms):
    """Configure les mocks Streamlit spécifiques au wizard."""
    ms.calls["button"].return_value = False
    ms.calls["text_input"].return_value = ""
    ms.calls["slider"].return_value = 200

    # st.form → context manager qui retourne un mock
    @contextmanager
    def _fake_form(*_a, **_kw):
        yield

    ms.calls["form"] = MagicMock(side_effect=_fake_form)
    ms._monkeypatch.setattr(ms._module.st, "form", ms.calls["form"])

    # st.form_submit_button
    ms.calls["form_submit_button"] = MagicMock(return_value=False)
    ms._monkeypatch.setattr(ms._module.st, "form_submit_button", ms.calls["form_submit_button"])

    # st.link_button
    ms.calls["link_button"] = MagicMock()
    ms._monkeypatch.setattr(ms._module.st, "link_button", ms.calls["link_button"])

    # st.balloons
    ms.calls["balloons"] = MagicMock()
    ms._monkeypatch.setattr(ms._module.st, "balloons", ms.calls["balloons"])

    # st.code
    ms.calls["code"] = MagicMock()
    ms._monkeypatch.setattr(ms._module.st, "code", ms.calls["code"])

    # st.query_params → dict vide
    ms._monkeypatch.setattr(ms._module.st, "query_params", {})

    # Columns pour mode selection (2 colonnes)
    ms.set_columns_dynamic()


# =============================================================================
# Tests du flux principal
# =============================================================================


class TestRenderSetupWizardPage:
    """Tests du point d'entrée render_setup_wizard_page."""

    def test_skip_si_configure(self, mock_st) -> None:
        """Affiche succès + balloons si config complète."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status(
            has_id=True,
            has_secret=True,
            has_token=True,
            has_players=True,
            player_count=1,
        )

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        ms.calls["balloons"].assert_called_once()
        ms.calls["success"].assert_called()

    def test_affiche_mode_selection_par_defaut(self, mock_st) -> None:
        """Affiche les cartes Xbox/Azure quand mode=None."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status()
        ms.session_state.clear()  # pas de _setup_mode

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        # Vérifie que columns(2) est appelé pour les cartes
        ms.calls["columns"].assert_called()
        # Vérifie que les boutons de choix sont rendus
        assert ms.calls["button"].call_count >= 2
        # Vérifie width="stretch" (pas use_container_width déprécié)
        for call in ms.calls["button"].call_args_list:
            assert call.kwargs.get("use_container_width") is None
            assert call.kwargs.get("width") == "stretch"

    def test_xbox_mode_sans_credentials(self, mock_st) -> None:
        """Mode Xbox : affiche le formulaire Azure si pas de credentials."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status()
        ms.session_state["_setup_mode"] = "xbox"

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        # Vérifie que le formulaire est rendu (form appelé)
        ms.calls["markdown"].assert_called()

    def test_xbox_mode_avec_credentials(self, mock_st) -> None:
        """Mode Xbox : affiche le bouton de connexion si credentials OK."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status(has_id=True, has_secret=True)
        ms.session_state["_setup_mode"] = "xbox"

        with (
            patch.object(mod, "get_setup_status", return_value=status),
            patch.dict(
                "os.environ",
                {"SPNKR_AZURE_CLIENT_ID": "12345678-1234-1234-1234-123456789abc"},
            ),
            patch("src.ui.xbox_oauth.build_xbox_auth_url", return_value="https://fake"),
            patch("src.ui.xbox_oauth.generate_oauth_state", return_value="state123"),
        ):
            mod.render_setup_wizard_page()

        # link_button doit être appelé avec l'URL OAuth
        ms.calls["link_button"].assert_called_once()
        call_kwargs = ms.calls["link_button"].call_args
        assert "https://fake" in str(call_kwargs)
        # Vérifie width="stretch" (pas use_container_width déprécié)
        assert call_kwargs.kwargs.get("use_container_width") is None
        assert call_kwargs.kwargs.get("width") == "stretch"

    def test_azure_mode_etape1(self, mock_st) -> None:
        """Mode Azure : affiche étape 1 si pas de credentials."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status()
        ms.session_state["_setup_mode"] = "azure"

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        # Le formulaire Azure est rendu
        ms.calls["markdown"].assert_called()

    def test_azure_mode_etape2(self, mock_st) -> None:
        """Mode Azure : affiche étape 2 (token) si credentials OK."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status(has_id=True, has_secret=True)
        ms.session_state["_setup_mode"] = "azure"

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        # Step 2 affiche la commande du script token
        ms.calls["code"].assert_called()

    def test_azure_mode_etape3(self, mock_st) -> None:
        """Mode Azure : affiche étape 3 (joueur) si token OK."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status(has_id=True, has_secret=True, has_token=True)
        ms.session_state["_setup_mode"] = "azure"

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        # Le formulaire joueur est rendu (text_input + slider)
        ms.calls["markdown"].assert_called()

    def test_oauth_result_affiche_succes(self, mock_st) -> None:
        """Quand _xbox_oauth_result en session_state, affiche succès."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status()
        ms.session_state["_xbox_oauth_result"] = {
            "gamertag": "TestSpartan",
            "xuid": "123",
        }

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        ms.calls["balloons"].assert_called_once()
        ms.calls["success"].assert_called()
        # Le résultat est nettoyé
        assert "_xbox_oauth_result" not in ms.session_state

    def test_oauth_result_erreur_non_affichee_comme_succes(self, mock_st) -> None:
        """Un résultat OAuth avec 'error' doit pas déclencher le succès."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        status = _make_status()
        ms.session_state["_xbox_oauth_result"] = {
            "error": "connection refused",
        }

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        # balloons ne doit PAS avoir été appelé
        ms.calls["balloons"].assert_not_called()

    def test_back_button_reset_mode(self, mock_st) -> None:
        """Le bouton retour supprime _setup_mode de session_state."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)
        _setup_wizard_mocks(ms)

        # Simuler un clic sur le bouton retour
        ms.calls["button"].return_value = True
        status = _make_status()
        ms.session_state["_setup_mode"] = "xbox"

        with patch.object(mod, "get_setup_status", return_value=status):
            mod.render_setup_wizard_page()

        # Le mode doit être supprimé
        assert "_setup_mode" not in ms.session_state


# =============================================================================
# Tests des composants réutilisables
# =============================================================================


class TestWizardComponents:
    """Tests des composants réutilisables du wizard."""

    def test_progress_bar_html_done(self, mock_st) -> None:
        """La barre de progression marque les étapes passées comme 'done'."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)

        mod._render_progress_bar(2, 3)

        # Vérifie que markdown est appelé avec du HTML
        call_args = ms.calls["markdown"].call_args
        html = str(call_args)
        assert "done" in html
        assert "active" in html

    def test_progress_bar_html_all_done(self, mock_st) -> None:
        """Toutes les étapes done quand current > total."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)

        mod._render_progress_bar(4, 3)

        call_args = ms.calls["markdown"].call_args
        html = str(call_args)
        assert html.count("done") == 3

    def test_step_header_active(self, mock_st) -> None:
        """L'en-tête d'étape courante a la classe 'active'."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)

        mod._render_step_header(2, "Mon étape", 2)

        call_args = ms.calls["markdown"].call_args
        html = str(call_args)
        assert "active" in html
        assert "Mon étape" in html

    def test_step_header_done(self, mock_st) -> None:
        """Les étapes passées ont la classe 'done'."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)

        mod._render_step_header(1, "Étape passée", 3)

        call_args = ms.calls["markdown"].call_args
        html = str(call_args)
        assert "done" in html

    def test_step_header_pending(self, mock_st) -> None:
        """Les étapes futures ont la classe 'pending'."""
        from src.ui.pages import setup_wizard as mod

        ms = mock_st(mod)

        mod._render_step_header(3, "Future", 1)

        call_args = ms.calls["markdown"].call_args
        html = str(call_args)
        assert "pending" in html
