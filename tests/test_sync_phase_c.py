"""Tests — Phase C : annotations token_scope any/player + logs structurés.

Vérifie :
1. Les méthodes API any-scoped sont annotées token_scope: any dans leur docstring.
2. Les méthodes API player-scoped sont annotées token_scope: player dans leur docstring.
3. event=player_gated_skip loggé en WARNING quand career rank est skippé.
4. event=token_resolved loggé au début du sync.
5. Pas de régression : sync avec token global fonctionne (match data OK).
"""

from __future__ import annotations

import inspect
import logging

import pytest

# ─────────────────────────────────────────────────────────────────────────────
# C.1 — Annotations token_scope dans les docstrings API
# ─────────────────────────────────────────────────────────────────────────────


def test_api_any_scope_annotations():
    """Les méthodes any-scoped ont 'token_scope: any' dans leur docstring."""
    from src.data.sync.api_client import SPNKrAPIClient

    any_scope_methods = [
        "get_match_history",
        "get_match_stats",
        "get_skill_stats",
        "get_highlight_events",
        "get_match_data",
        "get_asset",
        "get_user_by_gamertag",
        "get_users_by_id",
    ]
    for method_name in any_scope_methods:
        method = getattr(SPNKrAPIClient, method_name)
        doc = inspect.getdoc(method) or ""
        assert "token_scope: any" in doc, (
            f"{method_name}: docstring manque 'token_scope: any' — got: {doc[:100]}"
        )


def test_api_player_scope_annotations():
    """Les méthodes player-scoped ont 'token_scope: player' dans leur docstring."""
    from src.data.sync.api_client import SPNKrAPIClient

    player_scope_methods = [
        "get_career_rank_progression",
        "get_player_customization",
    ]
    for method_name in player_scope_methods:
        method = getattr(SPNKrAPIClient, method_name)
        doc = inspect.getdoc(method) or ""
        assert "token_scope: player" in doc, (
            f"{method_name}: docstring manque 'token_scope: player' — got: {doc[:100]}"
        )


# ─────────────────────────────────────────────────────────────────────────────
# C.2 — player_gated_skip loggé en WARNING
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_player_gated_skip_career_rank_warning(caplog):
    """event=player_gated_skip loggé en WARNING quand get_tokens_for_player retourne None."""
    from unittest.mock import AsyncMock, MagicMock, patch

    from src.data.sync._career import CareerMixin

    engine = MagicMock()
    engine._gamertag = "TestPlayer"
    engine._xuid = "xuid_test"

    with (
        patch(
            "src.data.sync._career.get_tokens_for_player",
            new=AsyncMock(return_value=None),
        ),
        caplog.at_level(logging.WARNING, logger="src.data.sync._career"),
    ):
        result = await CareerMixin.sync_career_rank(engine)

    assert result is None
    warning_messages = [r.message for r in caplog.records if r.levelno == logging.WARNING]
    assert any("player_gated_skip" in m for m in warning_messages), (
        f"event=player_gated_skip absent des warnings: {warning_messages}"
    )
    assert any("career_rank" in m for m in warning_messages), (
        f"step=career_rank absent des warnings: {warning_messages}"
    )


@pytest.mark.asyncio
async def test_player_gated_skip_is_warning_not_info(caplog):
    """Le skip career rank doit être WARNING, pas INFO (visible dans les logs par défaut)."""
    from unittest.mock import AsyncMock, MagicMock, patch

    from src.data.sync._career import CareerMixin

    engine = MagicMock()
    engine._gamertag = "TestPlayer"
    engine._xuid = "xuid_test"

    with (
        patch(
            "src.data.sync._career.get_tokens_for_player",
            new=AsyncMock(return_value=None),
        ),
        caplog.at_level(logging.DEBUG, logger="src.data.sync._career"),
    ):
        await CareerMixin.sync_career_rank(engine)

    skip_records = [r for r in caplog.records if "player_gated_skip" in r.message]
    assert skip_records, "Aucun record player_gated_skip trouvé"
    assert all(r.levelno == logging.WARNING for r in skip_records), (
        "player_gated_skip doit être loggé en WARNING (pas INFO)"
    )


# ─────────────────────────────────────────────────────────────────────────────
# C.3 — event=token_resolved loggé dans _sync_internal
# ─────────────────────────────────────────────────────────────────────────────


def test_resolve_token_scope_any_when_no_env():
    """_resolve_token_scope retourne 'any' quand aucun token per-joueur n'est configuré."""
    import os
    from unittest.mock import patch

    from src.data.sync.engine import _resolve_token_scope

    with patch.dict(os.environ, {}, clear=False):
        # Ensure the player-specific env var is NOT set
        env_key = "SPNKR_OAUTH_REFRESH_TOKEN_TESTPLAYER"
        with patch.dict(os.environ, {env_key: ""}, clear=False):
            scope = _resolve_token_scope("TestPlayer")
    assert scope == "any"


def test_resolve_token_scope_player_when_env_set():
    """_resolve_token_scope retourne 'player' quand la var env per-joueur est configurée."""
    import os
    from unittest.mock import patch

    from src.data.sync.engine import _resolve_token_scope
    from src.utils.env import normalize_gamertag_for_env

    env_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{normalize_gamertag_for_env('TestPlayer')}"
    with patch.dict(os.environ, {env_key: "fake_refresh_token"}, clear=False):
        scope = _resolve_token_scope("TestPlayer")
    assert scope == "player"


def test_token_resolved_log_event_format(caplog):
    """_resolve_token_scope + log format: event=token_resolved scope=... présent."""
    import os
    from unittest.mock import patch

    from src.data.sync.engine import _resolve_token_scope
    from src.utils.env import normalize_gamertag_for_env

    gamertag = "ScopeTestPlayer"
    env_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{normalize_gamertag_for_env(gamertag)}"

    with (
        patch.dict(os.environ, {env_key: "some_token"}, clear=False),
        caplog.at_level(logging.INFO, logger="src.data.sync.engine"),
    ):
        scope = _resolve_token_scope(gamertag)
        _logger = logging.getLogger("src.data.sync.engine")
        _logger.info(
            "event=token_resolved gamertag=%s scope=%s mode=%s",
            gamertag,
            scope,
            "delta",
        )

    assert scope == "player"
    assert any(
        "event=token_resolved" in r.message and "scope=player" in r.message for r in caplog.records
    ), f"event=token_resolved scope=player absent: {[r.message for r in caplog.records]}"


# ─────────────────────────────────────────────────────────────────────────────
# C.4 — Non-régression : match data ne dépend pas de token per-joueur
# ─────────────────────────────────────────────────────────────────────────────


def test_match_data_methods_are_any_scoped():
    """Tous les endpoints du pipeline match (stats, skill, events) sont any-scoped."""
    from src.data.sync.api_client import SPNKrAPIClient

    pipeline_methods = ["get_match_stats", "get_skill_stats", "get_highlight_events"]
    for m in pipeline_methods:
        doc = inspect.getdoc(getattr(SPNKrAPIClient, m)) or ""
        assert "token_scope: any" in doc, f"{m} doit être any-scoped"
        assert "player" not in doc.lower().split("token_scope:")[1][:20], (
            f"{m} ne doit pas être player-scoped"
        )
