"""Tests des interactions locales de la grille médias V2."""

from __future__ import annotations

from unittest.mock import patch


def test_queue_match_navigation_sets_pending_state() -> None:
    """Le callback de navigation prépare la page cible avant le rerun naturel."""
    ss: dict[str, object] = {}

    with patch("src.ui.pages.media_v2_grid.st") as mock_st:
        mock_st.session_state = ss
        from src.ui.pages import media_v2_grid

        media_v2_grid._queue_match_navigation("  match-123  ")

    assert ss["_pending_page"] == "Match"
    assert ss["_pending_match_id"] == "match-123"


def test_open_match_button_keeps_fallback_for_mocked_clicks() -> None:
    """Le helper reste testable même si le mock Streamlit n'exécute pas on_click."""
    ss: dict[str, object] = {}

    with patch("src.ui.pages.media_v2_grid.st") as mock_st:
        mock_st.session_state = ss
        mock_st.button.return_value = True
        from src.ui.pages import media_v2_grid

        media_v2_grid._open_match_button("match-456", "card")

    assert ss["_pending_page"] == "Match"
    assert ss["_pending_match_id"] == "match-456"


def test_queue_lightbox_open_sets_session_state() -> None:
    """L'ouverture de lightbox est posée en session_state avant le rerun du bouton."""
    ss: dict[str, object] = {}
    rows = [{"file_path": "a.png"}, {"file_path": "b.png"}]

    with patch("src.ui.pages.media_v2_grid.st") as mock_st:
        mock_st.session_state = ss
        from src.ui.pages import media_v2_grid

        media_v2_grid._queue_lightbox_open(rows, 1)

    assert ss["_lb_state"] == {"rows": rows, "idx": 1}


def test_set_lightbox_index_clamps_bounds() -> None:
    """La navigation de la lightbox borne toujours l'index dans la plage valide."""
    ss: dict[str, object] = {
        "_lb_state": {"rows": [{"file_path": "a.png"}, {"file_path": "b.png"}], "idx": 0}
    }

    with patch("src.ui.pages.media_v2_grid.st") as mock_st:
        mock_st.session_state = ss
        from src.ui.pages import media_v2_grid

        media_v2_grid._set_lightbox_index(9)
        assert ss["_lb_state"]["idx"] == 1

        media_v2_grid._set_lightbox_index(-5)
        assert ss["_lb_state"]["idx"] == 0


def test_clear_lightbox_state_removes_key() -> None:
    """La fermeture de la lightbox nettoie l'état persistant."""
    ss: dict[str, object] = {"_lb_state": {"rows": [{"file_path": "a.png"}], "idx": 0}}

    with patch("src.ui.pages.media_v2_grid.st") as mock_st:
        mock_st.session_state = ss
        from src.ui.pages import media_v2_grid

        media_v2_grid._clear_lightbox_state()

    assert "_lb_state" not in ss


def test_render_thumb_uses_native_streamlit_image() -> None:
    """La grille V2 affiche désormais les miniatures via st.image natif."""
    row = {
        "file_path": "capture.png",
        "thumbnail_path": "thumb.png",
        "kind": "image",
        "file_name": "capture.png",
    }

    with (
        patch("src.ui.pages.media_v2_grid.Path.exists", return_value=True),
        patch(
            "src.ui.pages.media_v2_grid.load_native_thumbnail_source",
            return_value=b"png-bytes",
        ) as mock_source,
        patch("src.ui.pages.media_v2_grid.st") as mock_st,
    ):
        from src.ui.pages import media_v2_grid

        media_v2_grid._render_thumb(row)

    assert mock_source.call_count == 1
    mock_st.image.assert_called_once_with(b"png-bytes", width="stretch")


def test_sync_like_click_skips_fallback_when_callback_already_ran() -> None:
    """Le clic ne doit pas retrigger un toggle si le callback Streamlit a déjà tourné."""
    from src.ui.pages import media_v2_likes

    file_path = "capture.png"
    ss: dict[str, object] = {media_v2_likes._like_callback_flag_key(file_path): True}

    with (
        patch("src.ui.pages.media_v2_likes.st") as mock_st,
        patch("src.ui.pages.media_v2_likes._toggle_like_callback") as mock_toggle,
    ):
        mock_st.session_state = ss

        media_v2_likes._sync_like_click(
            file_path,
            clicked=True,
            force_full_rerun=False,
        )

    mock_toggle.assert_not_called()
    assert media_v2_likes._like_callback_flag_key(file_path) not in ss


def test_sync_like_click_uses_fallback_for_mocked_button() -> None:
    """Le fallback doit rester actif quand un mock de bouton n'exécute pas le callback."""
    from src.ui.pages import media_v2_likes

    ss: dict[str, object] = {}

    with (
        patch("src.ui.pages.media_v2_likes.st") as mock_st,
        patch("src.ui.pages.media_v2_likes._toggle_like_callback") as mock_toggle,
    ):
        mock_st.session_state = ss

        media_v2_likes._sync_like_click(
            "capture.png",
            clicked=True,
            force_full_rerun=True,
        )

    mock_toggle.assert_called_once_with("capture.png")
    mock_st.rerun.assert_called_once_with()
