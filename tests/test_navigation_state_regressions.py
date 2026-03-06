"""Tests de non-régression navigation/état (session_state + query params)."""

from __future__ import annotations

from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest

from src.app import page_router


@pytest.mark.regression
def test_consume_pending_page_applies_valid_and_pops(monkeypatch: pytest.MonkeyPatch) -> None:
    """Une page en attente valide doit être appliquée et consommée."""
    fake_st = SimpleNamespace(session_state={"_pending_page": "media"})
    monkeypatch.setattr(page_router, "st", fake_st)

    page_router.consume_pending_page()

    assert fake_st.session_state.get("page") == "media"
    assert "_pending_page" not in fake_st.session_state


@pytest.mark.regression
def test_consume_pending_page_defaults_when_missing(monkeypatch: pytest.MonkeyPatch) -> None:
    """Sans page active ni pending valide, la page par défaut est injectée."""
    fake_st = SimpleNamespace(session_state={})
    monkeypatch.setattr(page_router, "st", fake_st)

    page_router.consume_pending_page()

    assert fake_st.session_state["page"] == "timeseries"


@pytest.mark.regression
def test_consume_pending_page_ignores_invalid_keeps_existing(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Une pending page invalide ne doit pas écraser la page courante."""
    fake_st = SimpleNamespace(session_state={"_pending_page": "Inconnue", "page": "citations"})
    monkeypatch.setattr(page_router, "st", fake_st)

    page_router.consume_pending_page()

    assert fake_st.session_state["page"] == "citations"
    assert "_pending_page" not in fake_st.session_state


@pytest.mark.regression
def test_consume_pending_match_id_trims_and_stores(monkeypatch: pytest.MonkeyPatch) -> None:
    """Le match_id pending doit être trim puis stocké dans l'input."""
    fake_st = SimpleNamespace(session_state={"_pending_match_id": "  abc123  "})
    monkeypatch.setattr(page_router, "st", fake_st)

    page_router.consume_pending_match_id()

    assert fake_st.session_state.get("match_id_input") == "abc123"
    assert "_pending_match_id" not in fake_st.session_state


@pytest.mark.regression
def test_render_page_selector_uses_canonical_pages(monkeypatch: pytest.MonkeyPatch) -> None:
    """Le sélecteur retourne un slug correspondant au label sélectionné."""
    # En contexte test, t() retourne le français par défaut,
    # donc get_page_labels() retourne les labels FR.
    from src.app.page_router import get_page_labels

    labels = get_page_labels()
    # Simuler la sélection du dernier label (correspond au slug "settings")
    segmented_control = MagicMock(return_value=labels[-1])
    fake_st = SimpleNamespace(
        segmented_control=segmented_control,
        session_state={},
    )
    monkeypatch.setattr(page_router, "st", fake_st)

    selected = page_router.render_page_selector()

    assert selected == "settings"
    segmented_control.assert_called_once()
    _, kwargs = segmented_control.call_args
    assert kwargs["options"] == labels
