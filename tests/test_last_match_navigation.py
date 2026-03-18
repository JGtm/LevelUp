"""Tests de la navigation précédent/suivant — Page Dernier match.

Couvre :
1. ``_resolve_nav_index`` — logique pure (sans Streamlit ni DuckDB)
2. Clés ``SK.LAST_MATCH_NAV_INDEX`` / ``SK.LAST_MATCH_NAV_TOTAL`` dans session_keys
3. Clés i18n ``lm_nav_prev`` / ``lm_nav_next`` présentes et résolues (FR + EN)
4. Import + signature de ``render_last_match_page``
"""

from __future__ import annotations

import pytest

from src.ui.pages.last_match import _resolve_nav_index

# =============================================================================
# _resolve_nav_index — comportement nominal
# =============================================================================


class TestResolveNavIndexNominal:
    """Cas nominaux : premier appel, navigation normale, extrémités."""

    def test_premier_appel_renvoie_dernier_index(self) -> None:
        """Sans historique, l'index est le dernier match (total - 1)."""
        idx, reset = _resolve_nav_index(total=10, stored_index=None, stored_total=None)
        assert idx == 9
        assert reset is True  # total None != 10 → reset

    def test_index_milieu_conserve(self) -> None:
        """Un index valide dans les bornes est renvoyé tel quel."""
        idx, reset = _resolve_nav_index(total=10, stored_index=5, stored_total=10)
        assert idx == 5
        assert reset is False

    def test_index_premier_match(self) -> None:
        """Index 0 (premier match) renvoyé sans modification."""
        idx, reset = _resolve_nav_index(total=10, stored_index=0, stored_total=10)
        assert idx == 0
        assert reset is False

    def test_index_dernier_match(self) -> None:
        """Index total-1 (dernier match) renvoyé sans modification."""
        idx, reset = _resolve_nav_index(total=10, stored_index=9, stored_total=10)
        assert idx == 9
        assert reset is False

    def test_match_unique(self) -> None:
        """Avec un seul match, l'index est toujours 0."""
        idx, reset = _resolve_nav_index(total=1, stored_index=0, stored_total=1)
        assert idx == 0
        assert reset is False


# =============================================================================
# _resolve_nav_index — changement de filtres (reset)
# =============================================================================


class TestResolveNavIndexReset:
    """Réinitialisation quand les filtres changent (total différent)."""

    def test_filtres_reduisent_total(self) -> None:
        """Quand le total diminue, on recale sur le dernier match."""
        idx, reset = _resolve_nav_index(total=5, stored_index=8, stored_total=10)
        assert idx == 4  # dernier match = total - 1
        assert reset is True

    def test_filtres_augmentent_total(self) -> None:
        """Quand le total augmente, on recale aussi sur le dernier match."""
        idx, reset = _resolve_nav_index(total=20, stored_index=5, stored_total=10)
        assert idx == 19
        assert reset is True

    def test_stored_total_none_force_reset(self) -> None:
        """stored_total=None (premier appel) provoque un reset."""
        idx, reset = _resolve_nav_index(total=7, stored_index=None, stored_total=None)
        assert idx == 6
        assert reset is True

    def test_stored_total_zero_force_reset(self) -> None:
        """stored_total=0 different de total courant → reset."""
        idx, reset = _resolve_nav_index(total=3, stored_index=0, stored_total=0)
        assert idx == 2
        assert reset is True


# =============================================================================
# _resolve_nav_index — session_key (bug filtre silencieux + même total)
# =============================================================================


class TestResolveNavIndexSessionKey:
    """Réinitialisation basée sur le changement de session_key.

    Reproduit le bug original : filtre de session silencieusement raté
    → dff reste à 673 matchs → total inchangé → index stale → mauvais match.
    Avec session_key, le changement de label force le reset même si total = même.
    """

    def test_session_changee_meme_total_force_reset(self) -> None:
        """session_key différente → reset même si total identique."""
        idx, reset = _resolve_nav_index(
            total=673,
            stored_index=660,
            stored_total=673,
            session_key="17/03/2026 20:09–20:58 (4)",
            stored_session_key="(toutes)",
        )
        assert reset is True
        assert idx == 672  # dernier match dans dff (le plus récent)

    def test_session_inchangee_pas_de_reset(self) -> None:
        """Même session_key → pas de reset, index préservé."""
        idx, reset = _resolve_nav_index(
            total=4,
            stored_index=2,
            stored_total=4,
            session_key="17/03/2026 20:09–20:58 (4)",
            stored_session_key="17/03/2026 20:09–20:58 (4)",
        )
        assert reset is False
        assert idx == 2

    def test_session_key_none_ignore(self) -> None:
        """session_key=None → comportement legacy (pas de reset par session)."""
        idx, reset = _resolve_nav_index(
            total=10,
            stored_index=5,
            stored_total=10,
            session_key=None,
            stored_session_key="old session",
        )
        assert reset is False
        assert idx == 5

    def test_session_key_prend_precedence_sur_total(self) -> None:
        """session_key change ET total change → reset (les deux conditions)."""
        idx, reset = _resolve_nav_index(
            total=4,
            stored_index=8,
            stored_total=673,
            session_key="session A",
            stored_session_key="(toutes)",
        )
        assert reset is True
        assert idx == 3  # dernier des 4 matchs filtrés


# =============================================================================
# _resolve_nav_index — clamping (index hors bornes sans changement de total)
# =============================================================================


class TestResolveNavIndexClamping:
    """Clamping de l'index quand il sort des bornes avec même total."""

    def test_index_negatif_clampe_a_zero(self) -> None:
        """Un index négatif est clamped à 0."""
        idx, reset = _resolve_nav_index(total=10, stored_index=-1, stored_total=10)
        assert idx == 0
        assert reset is False

    def test_index_trop_grand_clampe(self) -> None:
        """Un index >= total est clamped à total - 1."""
        idx, reset = _resolve_nav_index(total=10, stored_index=15, stored_total=10)
        assert idx == 9
        assert reset is False

    def test_index_exactement_total_clampe(self) -> None:
        """Index == total (hors borne) → clamped à total - 1."""
        idx, reset = _resolve_nav_index(total=10, stored_index=10, stored_total=10)
        assert idx == 9
        assert reset is False


# =============================================================================
# Session keys — présence des nouvelles constantes
# =============================================================================


class TestSessionKeys:
    """Vérifie que les constantes SK pour la navigation sont définies."""

    def test_last_match_nav_index_exists(self) -> None:
        from src.app.session_keys import SK

        assert hasattr(SK, "LAST_MATCH_NAV_INDEX")
        assert isinstance(SK.LAST_MATCH_NAV_INDEX, str)
        assert SK.LAST_MATCH_NAV_INDEX  # non vide

    def test_last_match_nav_total_exists(self) -> None:
        from src.app.session_keys import SK

        assert hasattr(SK, "LAST_MATCH_NAV_TOTAL")
        assert isinstance(SK.LAST_MATCH_NAV_TOTAL, str)
        assert SK.LAST_MATCH_NAV_TOTAL

    def test_last_match_nav_session_key_exists(self) -> None:
        from src.app.session_keys import SK

        assert hasattr(SK, "LAST_MATCH_NAV_SESSION_KEY")
        assert isinstance(SK.LAST_MATCH_NAV_SESSION_KEY, str)
        assert SK.LAST_MATCH_NAV_SESSION_KEY

    def test_nav_keys_distincts(self) -> None:
        """Les trois clés ne doivent pas être identiques (collision session_state)."""
        from src.app.session_keys import SK

        assert SK.LAST_MATCH_NAV_INDEX != SK.LAST_MATCH_NAV_TOTAL
        assert SK.LAST_MATCH_NAV_INDEX != SK.LAST_MATCH_NAV_SESSION_KEY
        assert SK.LAST_MATCH_NAV_TOTAL != SK.LAST_MATCH_NAV_SESSION_KEY


# =============================================================================
# i18n — clés lm_nav_prev / lm_nav_next
# =============================================================================


class TestNavI18nKeys:
    """Vérifie que les clés de navigation sont présentes et résolues."""

    @pytest.mark.parametrize("key", ["lm_nav_prev", "lm_nav_next"])
    def test_key_resolves_fr(self, key: str) -> None:
        from src.ui.i18n import t

        result = t(key, lang="fr")
        assert isinstance(result, str)
        assert result  # non vide
        assert not result.startswith("[")  # pas de clé manquante

    @pytest.mark.parametrize("key", ["lm_nav_prev", "lm_nav_next"])
    def test_key_resolves_en(self, key: str) -> None:
        from src.ui.i18n import t

        result = t(key, lang="en")
        assert isinstance(result, str)
        assert result
        assert not result.startswith("[")

    def test_prev_contains_arrow(self) -> None:
        """Le bouton précédent doit contenir une flèche gauche (◀)."""
        from src.ui.i18n import t

        assert "◀" in t("lm_nav_prev", lang="fr")

    def test_next_contains_arrow(self) -> None:
        """Le bouton suivant doit contenir une flèche droite (▶)."""
        from src.ui.i18n import t

        assert "▶" in t("lm_nav_next", lang="fr")

    def test_prev_different_from_next(self) -> None:
        """Les labels des deux boutons doivent être différents."""
        from src.ui.i18n import t

        assert t("lm_nav_prev", lang="fr") != t("lm_nav_next", lang="fr")


# =============================================================================
# Import et contrat de render_last_match_page
# =============================================================================


class TestRenderLastMatchPageContract:
    """Vérifie l'import et la signature sans exécuter Streamlit."""

    def test_module_importable(self) -> None:
        """Le module s'importe sans erreur."""
        import importlib

        mod = importlib.import_module("src.ui.pages.last_match")
        assert mod is not None

    def test_render_function_callable(self) -> None:
        """render_last_match_page est callable."""
        from src.ui.pages.last_match import render_last_match_page

        assert callable(render_last_match_page)

    def test_resolve_nav_index_callable(self) -> None:
        """_resolve_nav_index est callable (helper interne exporté pour tests)."""
        assert callable(_resolve_nav_index)

    def test_render_function_exported_in_pages_init(self) -> None:
        """render_last_match_page est accessible depuis src.ui.pages."""
        from src.ui import pages

        assert hasattr(pages, "render_last_match_page")
        assert callable(pages.render_last_match_page)
