"""Tests pour le module _refdata_outcomes et le re-export i18n.

Valide que get_outcome_map() retourne les labels corrects en FR et EN,
et que le dead code (OUTCOME_TO_FR, get_outcome_name_fr) a bien été supprimé.
"""

from src.data.domain._refdata_outcomes import get_outcome_map
from src.ui.i18n import get_outcome_map as get_outcome_map_i18n


class TestGetOutcomeMap:
    """Tests pour get_outcome_map()."""

    def test_get_outcome_map_fr_all_codes(self):
        """get_outcome_map('fr') contient {1, 2, 3, 4}."""
        result = get_outcome_map("fr")
        assert set(result.keys()) == {1, 2, 3, 4}

    def test_get_outcome_map_en_all_codes(self):
        """get_outcome_map('en') contient {1, 2, 3, 4}."""
        result = get_outcome_map("en")
        assert set(result.keys()) == {1, 2, 3, 4}

    def test_get_outcome_map_default_lang(self):
        """Sans argument → labels FR."""
        result = get_outcome_map()
        assert result[2] == "Victoire"
        assert result[3] == "Défaite"

    def test_get_outcome_map_win_fr(self):
        """get_outcome_map('fr')[2] == 'Victoire'."""
        assert get_outcome_map("fr")[2] == "Victoire"

    def test_get_outcome_map_win_en(self):
        """get_outcome_map('en')[2] == 'Win'."""
        assert get_outcome_map("en")[2] == "Win"

    def test_get_outcome_map_loss_fr(self):
        """get_outcome_map('fr')[3] == 'Défaite'."""
        assert get_outcome_map("fr")[3] == "Défaite"

    def test_get_outcome_map_tie_fr(self):
        """get_outcome_map('fr')[1] == 'Égalité'."""
        assert get_outcome_map("fr")[1] == "Égalité"

    def test_get_outcome_map_dnf_fr(self):
        """get_outcome_map('fr')[4] == 'Non terminé'."""
        assert get_outcome_map("fr")[4] == "Non terminé"

    def test_get_outcome_map_unknown_lang_fallback(self):
        """Langue inconnue → fallback vers FR."""
        result = get_outcome_map("zz")
        assert result[2] == "Victoire"

    def test_outcome_map_keys_are_expected(self):
        """Les clés du map sont exactement {1, 2, 3, 4}."""
        assert set(get_outcome_map().keys()) == {1, 2, 3, 4}

    def test_dead_code_removed(self):
        """refdata n'exporte plus get_outcome_name_fr ni OUTCOME_TO_FR."""
        import src.data.domain.refdata as refdata

        assert not hasattr(refdata, "get_outcome_name_fr"), (
            "get_outcome_name_fr doit être supprimé de refdata"
        )
        assert not hasattr(refdata, "OUTCOME_TO_FR"), "OUTCOME_TO_FR doit être supprimé de refdata"


class TestI18nReExport:
    """Vérifie que le re-export i18n renvoie la même implémentation."""

    def test_i18n_reexport_fr(self):
        """get_outcome_map via i18n retourne les mêmes labels FR."""
        direct = get_outcome_map("fr")
        via_i18n = get_outcome_map_i18n("fr")
        assert direct == via_i18n

    def test_i18n_reexport_en(self):
        """get_outcome_map via i18n retourne les mêmes labels EN."""
        direct = get_outcome_map("en")
        via_i18n = get_outcome_map_i18n("en")
        assert direct == via_i18n
