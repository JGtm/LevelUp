"""Tests pour le module de traduction des rangs Halo Infinite."""

from src.ui.i18n.ranks import (
    CAREER_RANK_NAMES_FR,
    CSR_TIER_NAMES_FR,
    RANK_NAMES_FR,
    translate_rank,
)


class TestTranslateRank:
    """Traduction EN→FR des rangs."""

    def test_career_rank_known(self):
        assert translate_rank("Recruit") == "Recrue"
        assert translate_rank("General") == "Général"
        assert translate_rank("Hero") == "Héros"

    def test_csr_tier_known(self):
        assert translate_rank("Silver") == "Argent"
        assert translate_rank("Gold") == "Or"
        assert translate_rank("Diamond") == "Diamant"

    def test_unknown_rank_fallback(self):
        assert translate_rank("Mythic") == "Mythic"

    def test_empty_string(self):
        assert translate_rank("") == ""

    def test_combined_dict_has_all_entries(self):
        expected = len(CAREER_RANK_NAMES_FR) + len(CSR_TIER_NAMES_FR)
        assert len(RANK_NAMES_FR) == expected

    def test_no_duplicate_keys(self):
        career_keys = set(CAREER_RANK_NAMES_FR.keys())
        csr_keys = set(CSR_TIER_NAMES_FR.keys())
        assert career_keys.isdisjoint(csr_keys), "Clés dupliquées entre career et CSR"
