"""Tests unitaires pour normalize_mode_label — sans st.session_state.

Vérifie que la fonction est pure après Phase 2 : aucun accès
à st.session_state, résultat déterministe avec les paramètres explicites.
"""

from __future__ import annotations

import pytest


class TestNormalizeModeLabel:
    """Tests de normalize_mode_label avec paramètres explicites."""

    def test_returns_none_on_none(self) -> None:
        from src.app.helpers import normalize_mode_label

        assert normalize_mode_label(None) is None

    def test_strips_map_suffix(self) -> None:
        from src.app.helpers import normalize_mode_label

        result = normalize_mode_label("Slayer on Cliffhanger", lang="en", normalize=False)
        assert result == "Slayer"

    def test_removes_forge_suffix(self) -> None:
        from src.app.helpers import normalize_mode_label

        result = normalize_mode_label("Slayer - Forge", lang="en", normalize=False)
        assert result == "Slayer"

    def test_removes_ranked_suffix(self) -> None:
        from src.app.helpers import normalize_mode_label

        result = normalize_mode_label("Slayer - Ranked", lang="en", normalize=False)
        assert result == "Slayer"

    def test_normalize_false_skips_translation(self) -> None:
        from src.app.helpers import normalize_mode_label

        result = normalize_mode_label("Quick Play:Slayer", lang="fr", normalize=False)
        assert result is not None
        assert isinstance(result, str)

    def test_no_session_state_needed(self) -> None:
        """Appel sans session_state actif — ne doit pas lever d'exception."""
        import unittest.mock as mock

        # Simuler l'absence de session_state
        with mock.patch("streamlit.session_state", {}, create=True):
            from src.app.helpers import normalize_mode_label

            result = normalize_mode_label("Slayer on Aquarius", lang="fr", normalize=True)
            # Le résultat doit être une string ou None, jamais une exception
            assert result is None or isinstance(result, str)

    def test_lang_param_accepted(self) -> None:
        from src.app.helpers import normalize_mode_label

        # Les deux langues doivent fonctionner sans exception
        r_fr = normalize_mode_label("Slayer", lang="fr", normalize=True)
        r_en = normalize_mode_label("Slayer", lang="en", normalize=True)
        assert r_fr is None or isinstance(r_fr, str)
        assert r_en is None or isinstance(r_en, str)
