"""Tests du flux de navigation Media → Explorer (deep-link match).

Flux complet (3 reruns Streamlit) :
  Rerun 1 (page Media) : bouton "Match" cliqué
    → open_match_button() pose session_state["_pending_page"]="Match"
                              + session_state["_pending_match_id"]=mid
    → st.rerun()
    Note : on utilise session_state (PAS st.query_params) pour éviter que
    la combinaison match_id + gamertag éventuel dans l'URL déclencheswitching
    de DB dans _resolve_db_path().
  Rerun 2 (routing) :
    → _dispatch_pages appelle consume_pending_match_id
      → pop "_pending_match_id" → set "match_id_input"
    → _dispatch_navigation : pending_page="Match" → st.switch_page(explorer) → nouveau rerun
  Rerun 3 (page Explorer) : render_explorer_page
    → _consume_deep_links : pop "match_id_input" (ou "_pending_match_id" si switch_page
      a interrompu avant consume_pending_match_id) → pending_mid
    → show_single_match appelé
"""

from __future__ import annotations

from unittest.mock import patch

# ---------------------------------------------------------------------------
# Helpers de session_state simulée
# ---------------------------------------------------------------------------


def _make_ss(initial: dict | None = None) -> dict:
    """Crée un session_state simulé sous forme de dict ordinaire.
    Streamlit session_state est en réalité dict-like ; un dict Python suffit.
    """
    return dict(initial or {})


# ---------------------------------------------------------------------------
# Tests consume_pending_match_id (page_router)
# ---------------------------------------------------------------------------


class TestConsumePendingMatchId:
    def test_set_match_id_input_from_pending(self):
        """consume_pending_match_id transfère _pending_match_id → match_id_input."""
        ss = _make_ss({"_pending_match_id": "abc-123"})
        with patch("src.app.page_router.st") as mock_st:
            mock_st.session_state = ss
            from src.app import page_router

            page_router.consume_pending_match_id()
        assert ss.get("match_id_input") == "abc-123"
        assert "_pending_match_id" not in ss

    def test_trims_whitespace(self):
        ss = _make_ss({"_pending_match_id": "  abc-123  "})
        with patch("src.app.page_router.st") as mock_st:
            mock_st.session_state = ss
            from src.app import page_router

            page_router.consume_pending_match_id()
        assert ss.get("match_id_input") == "abc-123"

    def test_noop_when_no_pending(self):
        ss = _make_ss({})
        with patch("src.app.page_router.st") as mock_st:
            mock_st.session_state = ss
            from src.app import page_router

            page_router.consume_pending_match_id()
        assert "match_id_input" not in ss

    def test_noop_when_empty_string(self):
        ss = _make_ss({"_pending_match_id": "   "})
        with patch("src.app.page_router.st") as mock_st:
            mock_st.session_state = ss
            from src.app import page_router

            page_router.consume_pending_match_id()
        assert "match_id_input" not in ss


# ---------------------------------------------------------------------------
# Tests _consume_deep_links (explorer)
# ---------------------------------------------------------------------------


class TestConsumeDeepLinks:
    def _call(self, ss_dict: dict) -> tuple[str | None, str]:
        ss = _make_ss(ss_dict)
        with patch("src.ui.pages.explorer.st") as mock_st:
            mock_st.session_state = ss
            from src.ui.pages import explorer

            return explorer._consume_deep_links()

    def test_reads_match_id_input(self):
        """Chemin normal : consume_pending_match_id a écrit match_id_input."""
        gt, mid = self._call({"match_id_input": "match-001"})
        assert mid == "match-001"
        assert gt is None

    def test_reads_pending_match_id_directly(self):
        """Chemin de secours : switch_page a interrompu avant la consommation."""
        gt, mid = self._call({"_pending_match_id": "match-002"})
        assert mid == "match-002"
        assert gt is None

    def test_pending_match_id_has_priority_over_input(self):
        """_pending_match_id direct prioritaire sur match_id_input."""
        gt, mid = self._call(
            {
                "_pending_match_id": "match-new",
                "match_id_input": "match-old",
            }
        )
        assert mid == "match-new"

    def test_pop_clears_match_id_input_from_state(self):
        """Après _consume_deep_links, match_id_input est supprimé."""
        ss = _make_ss({"match_id_input": "match-001"})
        with patch("src.ui.pages.explorer.st") as mock_st:
            mock_st.session_state = ss
            from src.ui.pages import explorer

            explorer._consume_deep_links()
        assert "match_id_input" not in ss

    def test_pop_clears_pending_match_id_from_state(self):
        """Après _consume_deep_links, _pending_match_id est supprimé."""
        ss = _make_ss({"_pending_match_id": "match-001"})
        with patch("src.ui.pages.explorer.st") as mock_st:
            mock_st.session_state = ss
            from src.ui.pages import explorer

            explorer._consume_deep_links()
        assert "_pending_match_id" not in ss

    def test_returns_empty_mid_when_nothing_set(self):
        gt, mid = self._call({})
        assert mid == ""
        assert gt is None

    def test_trims_whitespace(self):
        gt, mid = self._call({"match_id_input": "  match-001  "})
        assert mid == "match-001"

    def test_reads_pending_gamertag(self):
        gt, mid = self._call({"_pending_gamertag": "PlayerX", "match_id_input": ""})
        assert gt == "PlayerX"


# ---------------------------------------------------------------------------
# Test bout en bout : simulation des 2 reruns
# ---------------------------------------------------------------------------


class TestFullNavigationFlow:
    """Simule exactement les 2 reruns du flux Media → Explorer."""

    def test_rerun1_puis_rerun2_match_id_disponible(self):
        """
        Rerun 1 : bouton Media → pose _pending_page + _pending_match_id en session_state.
        Rerun 2 : consume_pending_match_id crée match_id_input.
        Rerun 3 : _consume_deep_links lit match_id_input.
        """
        # --- Rerun 1 : bouton Media cliqué (session_state, pas query_params) ---
        ss = _make_ss({})
        ss["_pending_page"] = "Match"
        ss["_pending_match_id"] = "match-abc"
        assert ss["_pending_match_id"] == "match-abc"

        # --- Rerun 2 : consume_pending_match_id ---
        with patch("src.app.page_router.st") as mock_st:
            mock_st.session_state = ss
            from src.app import page_router

            page_router.consume_pending_match_id()

        # _pending_match_id consommé, match_id_input créé
        assert "_pending_match_id" not in ss
        assert ss.get("match_id_input") == "match-abc"

        # --- Rerun 3 : Explorer _consume_deep_links ---
        with patch("src.ui.pages.explorer.st") as mock_st:
            mock_st.session_state = ss
            from src.ui.pages import explorer

            gt, mid = explorer._consume_deep_links()

        assert mid == "match-abc"
        # Nettoyé après lecture
        assert "match_id_input" not in ss

    def test_rerun2_interrompu_par_switch_page(self):
        """
        Cas : switch_page interrompt consume_pending_match_id AVANT qu'il s'exécute.
        _pending_match_id est encore en session_state quand Explorer s'ouvre.
        """
        # session_state comme si switch_page s'est fait AVANT consume_pending_match_id
        ss = _make_ss({"_pending_match_id": "match-xyz"})
        # Pas de match_id_input (consume_pending_match_id jamais appelé)
        assert "match_id_input" not in ss

        with patch("src.ui.pages.explorer.st") as mock_st:
            mock_st.session_state = ss
            from src.ui.pages import explorer

            gt, mid = explorer._consume_deep_links()

        assert mid == "match-xyz"
        assert "_pending_match_id" not in ss


# ---------------------------------------------------------------------------
# Tests open_match_button (media_library_render)
# ---------------------------------------------------------------------------


class TestOpenMatchButton:
    """Vérifie que le bouton Media pose les clés session_state correctes.

    On utilise session_state (NOT st.query_params) pour éviter que
    la présence conjointe de match_id + gamertag dans l'URL ne déclenche
    le switch de DB dans _resolve_db_path().
    """

    def _call_button_clicked(self, match_id: str, unique_suffix: str | None = None) -> dict:
        """Simule un clic sur le bouton et retourne les clés session_state posées."""
        ss: dict[str, str] = {}
        rerun_called = []

        with patch("src.ui.pages.media_library_render.st") as mock_st:
            mock_st.button.return_value = True  # simule le clic
            mock_st.session_state = ss
            mock_st.rerun = lambda: rerun_called.append(True)
            from src.ui.pages import media_library_render

            media_library_render.open_match_button(match_id, unique_suffix=unique_suffix)

        return {"ss": ss, "rerun_called": bool(rerun_called)}

    def test_sets_pending_page_and_match_id_on_click(self):
        """Un clic pose _pending_page='Match' + _pending_match_id=mid en session_state."""
        result = self._call_button_clicked("match-aaa-111")
        assert result["ss"]["_pending_page"] == "Match"
        assert result["ss"]["_pending_match_id"] == "match-aaa-111"
        assert result["rerun_called"] is True

    def test_trims_match_id(self):
        """L'ID est trimé avant d'être posé."""
        result = self._call_button_clicked("  match-bbb  ")
        assert result["ss"]["_pending_match_id"] == "match-bbb"

    def test_no_click_no_session_state_written(self):
        """Sans clic, aucune clé session_state n'est posée."""
        with patch("src.ui.pages.media_library_render.st") as mock_st:
            mock_st.button.return_value = False
            rerun_called = []
            mock_st.rerun = lambda: rerun_called.append(True)
            from src.ui.pages import media_library_render

            media_library_render.open_match_button("match-ccc")
        assert not rerun_called

    def test_empty_match_id_shows_caption(self):
        """Un match_id vide affiche une caption au lieu d'un bouton."""
        with patch("src.ui.pages.media_library_render.st") as mock_st:
            caption_called = []
            mock_st.caption = lambda *_a, **_kw: caption_called.append(True)
            mock_st.button.return_value = False
            from src.ui.pages import media_library_render

            media_library_render.open_match_button("")
        assert caption_called
        mock_st.button.assert_not_called()
