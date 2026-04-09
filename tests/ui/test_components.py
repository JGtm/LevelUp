"""Tests des composants UI : checkbox_filter, duckdb_analytics, info_note.

Sprint 7bis – Tâche 7b.7
"""

from __future__ import annotations

from unittest.mock import MagicMock, patch

# ═══════════════════════════════════════════════════════════════════
# render_info_note – composant partagé post-graphe
# ═══════════════════════════════════════════════════════════════════


class TestRenderInfoNote:
    """Tests pour src/ui/components/info_note.render_info_note."""

    def _call(self, text: str, hints: bool = True) -> str:
        """Appelle render_info_note et retourne le HTML injecté via st.markdown."""
        import src.ui.components.info_note as mod

        captured: list[str] = []

        def fake_markdown(html, **_kw):
            captured.append(html)

        with (
            patch.object(mod, "hints_visible", return_value=hints),
            patch("streamlit.markdown", side_effect=fake_markdown),
        ):
            mod.render_info_note(text)

        return captured[0] if captured else ""

    def test_ne_rend_rien_si_hints_invisible(self) -> None:
        html = self._call("Texte ignoré", hints=False)
        assert html == ""

    def test_rend_div_ts_note(self) -> None:
        html = self._call("Simple phrase.")
        assert '<div class="ts-note">' in html

    def test_texte_simple_enveloppe_en_paragraphe(self) -> None:
        html = self._call("Simple phrase.")
        assert "<p>Simple phrase.</p>" in html

    def test_ligne_liste_produit_ul_li(self) -> None:
        html = self._call("- Premier élément")
        assert "<ul>" in html
        assert "<li>Premier élément</li>" in html

    def test_plusieurs_lignes_liste(self) -> None:
        html = self._call("- Item A\n- Item B")
        assert html.count("<li>") == 2
        assert html.count("</ul>") == 1

    def test_gras_converti_en_strong(self) -> None:
        html = self._call("- Un **mot** important")
        assert "<strong>mot</strong>" in html

    def test_gras_dans_paragraphe(self) -> None:
        html = self._call("Phrase avec **accent**.")
        assert "<strong>accent</strong>" in html

    def test_liste_puis_paragraphe_ferme_ul(self) -> None:
        html = self._call("- Item A\nSuite non listée.")
        assert "</ul>" in html
        assert "<p>Suite non listée.</p>" in html

    def test_lignes_vides_ignorees(self) -> None:
        html = self._call("Ligne 1\n\nLigne 2")
        assert html.count("<p>") == 2
        # Pas de <p></p> pour la ligne vide
        assert "<p></p>" not in html

    def test_texte_multilignes_complet(self) -> None:
        text = "- **Surface** large → fort\n- Profils **superposés** → même style"
        html = self._call(text)
        assert "<strong>Surface</strong>" in html
        assert "<strong>superposés</strong>" in html
        assert html.count("<li>") == 2


# ═══════════════════════════════════════════════════════════════════
# checkbox_filter – fonctions pures
# ═══════════════════════════════════════════════════════════════════


class TestInferCategory:
    """Tests pour _infer_category."""

    def test_arena_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Arène : Assassin") == "Assassin"

    def test_btb_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("BTB : CTF") == "BTB"

    def test_super_fiesta(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Super Fiesta : Assassin") == "Fiesta"

    def test_fiesta_in_content(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Communauté : Fiesta Assassin") == "Fiesta"

    def test_husky_raid(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Husky Raid : CDD") == "Fiesta"

    def test_castle_wars(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Castle Wars : Assassin") == "Fiesta"

    def test_ranked_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Ranked : Slayer") == "Ranked"

    def test_classé_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Classé : Slayer") == "Ranked"

    def test_firefight_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Firefight : Heroic") == "Firefight"

    def test_gruntpocalypse(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Gruntpocalypse : Solo") == "Firefight"

    def test_communauté_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Communauté : Oddball") == "Assassin"

    def test_unknown_mode(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Something Random") == "Other"

    def test_tactical_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Tactical : Slayer") == "Assassin"

    def test_event_prefix(self):
        from src.ui.components.checkbox_filter import _infer_category

        assert _infer_category("Event : Special") == "Other"


class TestTranslateCategory:
    """Tests pour _translate_category."""

    def test_known_categories(self):
        from src.ui.components.checkbox_filter import _translate_category

        assert _translate_category("Assassin") == "Assassin"
        assert _translate_category("Fiesta") == "Fiesta"
        assert _translate_category("BTB") == "Grande bataille en équipe"
        assert _translate_category("Ranked") == "Classé"
        assert _translate_category("Firefight") == "Baptême du feu"
        assert _translate_category("Other") == "Autre"

    def test_unknown_returns_input(self):
        from src.ui.components.checkbox_filter import _translate_category

        assert _translate_category("UnknownCat") == "UnknownCat"


class TestExtractModeName:
    """Tests pour _extract_mode_name."""

    def test_with_prefix(self):
        from src.ui.components.checkbox_filter import _extract_mode_name

        assert _extract_mode_name("Arène : Assassin") == "Assassin"

    def test_btb_ctf(self):
        from src.ui.components.checkbox_filter import _extract_mode_name

        assert _extract_mode_name("BTB : Capture du drapeau") == "Capture du drapeau"

    def test_super_husky(self):
        from src.ui.components.checkbox_filter import _extract_mode_name

        assert _extract_mode_name("Super Husky Raid : CDD") == "CDD"

    def test_no_prefix(self):
        from src.ui.components.checkbox_filter import _extract_mode_name

        assert _extract_mode_name("Slayer") == "Slayer"


class TestGetFirefightPlaylists:
    """Tests pour get_firefight_playlists."""

    def test_finds_firefight(self):
        from src.ui.components.checkbox_filter import get_firefight_playlists

        playlists = ["Quick Play", "Ranked Arena", "Firefight: Heroic", "BTB"]
        result = get_firefight_playlists(playlists)
        assert result == {"Firefight: Heroic"}

    def test_finds_firefight_french(self):
        from src.ui.components.checkbox_filter import get_firefight_playlists

        playlists = [
            "Partie rapide",
            "Baptême du feu",
            "Baptême du feu : Roi de la colline héroïque",
        ]
        result = get_firefight_playlists(playlists)
        assert result == {"Baptême du feu", "Baptême du feu : Roi de la colline héroïque"}

    def test_case_insensitive(self):
        from src.ui.components.checkbox_filter import get_firefight_playlists

        playlists = ["FIREFIGHT KOTH", "ranked"]
        result = get_firefight_playlists(playlists)
        assert result == {"FIREFIGHT KOTH"}

    def test_no_firefight(self):
        from src.ui.components.checkbox_filter import get_firefight_playlists

        playlists = ["Quick Play", "Ranked Arena"]
        result = get_firefight_playlists(playlists)
        assert result == set()

    def test_empty_list(self):
        from src.ui.components.checkbox_filter import get_firefight_playlists

        assert get_firefight_playlists([]) == set()


class TestRenderCheckboxFilter:
    """Tests pour render_checkbox_filter avec MockStreamlit."""

    def test_empty_options_returns_empty(self, mock_st):
        from src.ui.components import checkbox_filter as mod

        mock_st(mod)
        result = mod.render_checkbox_filter(
            label="Test",
            options=[],
            session_key="test_empty",
        )
        assert result == set()

    def test_all_selected_by_default(self, mock_st):
        from src.ui.components import checkbox_filter as mod

        ms = mock_st(mod)
        ms.calls["checkbox"] = MagicMock(return_value=True)
        ms._monkeypatch.setattr(mod.st, "checkbox", ms.calls["checkbox"])
        # Buttons return False (no click)
        cols = ms.columns
        for c in cols:
            c.button = MagicMock(return_value=False)

        options = ["Mode A", "Mode B", "Mode C"]
        result = mod.render_checkbox_filter(
            label="Modes",
            options=options,
            session_key="test_all_default",
        )
        assert result == set(options)

    def test_default_unchecked(self, mock_st):
        from src.ui.components import checkbox_filter as mod

        ms = mock_st(mod)
        # Le checkbox mock doit refléter la valeur 'value' passée en kwarg
        # (le checkbox Streamlit reçoit value=checked)
        ms.calls["checkbox"] = MagicMock(side_effect=lambda *_a, value=False, **_kw: value)
        ms._monkeypatch.setattr(mod.st, "checkbox", ms.calls["checkbox"])
        cols = ms.columns
        for c in cols:
            c.button = MagicMock(return_value=False)

        options = ["Quick Play", "Firefight: Heroic", "Ranked"]
        result = mod.render_checkbox_filter(
            label="Playlists",
            options=options,
            session_key="test_unch",
            default_unchecked={"Firefight: Heroic"},
        )
        # La session state initiale exclut Firefight
        assert "Quick Play" in result
        assert "Ranked" in result
        assert "Firefight: Heroic" not in result
