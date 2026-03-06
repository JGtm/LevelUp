"""Tests de non-régression navigation/état (session_state + query params)."""

from __future__ import annotations

from types import SimpleNamespace

import pytest

from src.app import page_router


@pytest.mark.regression
def test_consume_pending_match_id_trims_and_stores(monkeypatch: pytest.MonkeyPatch) -> None:
    """Le match_id pending doit être trim puis stocké dans l'input."""
    fake_st = SimpleNamespace(session_state={"_pending_match_id": "  abc123  "})
    monkeypatch.setattr(page_router, "st", fake_st)

    page_router.consume_pending_match_id()

    assert fake_st.session_state.get("match_id_input") == "abc123"
    assert "_pending_match_id" not in fake_st.session_state
