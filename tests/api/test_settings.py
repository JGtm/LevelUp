"""Tests unitaires — endpoints Settings (Slice 1).

Couvre :
- GET /settings (DEMO_MODE + mode normal)
- PATCH /settings (mise à jour partielle)
- POST /settings/media/reset-index (confirmation + job asynchrone)
"""

from __future__ import annotations

from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def force_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")


@pytest.fixture
def no_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "false")


@pytest.fixture
async def client() -> AsyncClient:
    from apps.api.app.core.config import get_settings
    from apps.api.app.main import create_app

    get_settings.cache_clear()  # type: ignore[attr-defined]
    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


# ===========================================================================
# GET /settings
# ===========================================================================


@pytest.mark.anyio
async def test_get_settings_demo_mode_returns_defaults(client: AsyncClient) -> None:
    """En DEMO_MODE, retourne les valeurs par défaut d'AppSettings."""
    resp = await client.get("/api/v1/settings")
    assert resp.status_code == 200
    data = resp.json()

    # Valeurs par défaut définies dans AppSettings
    assert data["lang"] == "fr"
    assert data["user_timezone"] == "Europe/Paris"
    assert data["normalize_mode_labels"] is True
    assert data["show_records"] is False
    assert data["discord_notifications_enabled"] is False
    assert "discord_webhook_url_present" in data
    assert "discord_webhook_url" not in data  # jamais exposé


@pytest.mark.anyio
async def test_get_settings_schema_complete(client: AsyncClient) -> None:
    """La réponse contient tous les champs attendus du schéma SettingsResponse."""
    resp = await client.get("/api/v1/settings")
    assert resp.status_code == 200
    data = resp.json()

    required_keys = [
        "lang",
        "discord_lang",
        "user_timezone",
        "normalize_mode_labels",
        "show_records",
        "refresh_clears_caches",
        "career_top_exclude_btb",
        "media_captures_base_dir",
        "media_tolerance_minutes",
        "media_watcher_enabled",
        "media_watcher_debounce_seconds",
        "discord_notifications_enabled",
        "discord_webhook_url_present",
        "discord_notify_sync",
        "discord_notify_backfill",
        "discord_notify_new_version",
        "discord_notify_new_media",
        "spnkr_refresh_with_backfill",
        "spnkr_refresh_backfill_medals",
        "spnkr_refresh_backfill_skill",
        "spnkr_refresh_backfill_aliases",
        "spnkr_refresh_backfill_personal_scores",
        "spnkr_refresh_backfill_performance_scores",
        "spnkr_refresh_backfill_lusr",
        "spnkr_refresh_backfill_events",
        "spnkr_refresh_backfill_weapons",
    ]
    for key in required_keys:
        assert key in data, f"Clé manquante dans SettingsResponse : {key}"


@pytest.mark.anyio
async def test_get_settings_no_demo_loads_file(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """En mode normal, charge depuis app_settings.json (mocked)."""
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]

    from apps.api.app.schemas.settings import SettingsResponse

    mock_settings = SettingsResponse(
        lang="en",
        discord_lang="en",
        user_timezone="America/New_York",
        normalize_mode_labels=False,
        show_records=True,
        refresh_clears_caches=False,
        career_top_exclude_btb=True,
        media_captures_base_dir="/media/captures",
        media_tolerance_minutes=5,
        media_watcher_enabled=True,
        media_watcher_debounce_seconds=10,
        discord_notifications_enabled=True,
        discord_webhook_url_present=True,
        discord_notify_sync=True,
        discord_notify_backfill=False,
        discord_notify_new_version=True,
        discord_notify_new_media=False,
        spnkr_refresh_with_backfill=True,
        spnkr_refresh_backfill_medals=True,
        spnkr_refresh_backfill_skill=False,
        spnkr_refresh_backfill_aliases=False,
        spnkr_refresh_backfill_personal_scores=False,
        spnkr_refresh_backfill_performance_scores=True,
        spnkr_refresh_backfill_lusr=True,
        spnkr_refresh_backfill_events=False,
        spnkr_refresh_backfill_weapons=False,
    )

    # load_api_settings est importé lazily dans le router ; patcher à la source.
    with patch(
        "apps.api.app.services.settings_service.load_api_settings",
        return_value=mock_settings,
    ):
        resp = await client.get("/api/v1/settings")

    assert resp.status_code == 200
    data = resp.json()
    assert data["lang"] == "en"
    assert data["user_timezone"] == "America/New_York"
    assert data["show_records"] is True
    assert data["discord_webhook_url_present"] is True
    assert "discord_webhook_url" not in data


# ===========================================================================
# PATCH /settings
# ===========================================================================


@pytest.mark.anyio
async def test_patch_settings_unavailable_in_demo(client: AsyncClient) -> None:
    """En DEMO_MODE, PATCH /settings retourne 422."""
    resp = await client.patch("/api/v1/settings", json={"lang": "en"})
    assert resp.status_code == 422
    assert resp.json()["code"] == "demo_mode_unsupported"


@pytest.mark.anyio
async def test_patch_settings_updates_lang(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """PATCH /settings avec lang='en' persiste et retourne les settings mises à jour."""
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]

    from apps.api.app.schemas.settings import SettingsResponse

    default_settings = SettingsResponse(
        lang="en",
        discord_lang="fr",
        user_timezone="Europe/Paris",
        normalize_mode_labels=True,
        show_records=False,
        refresh_clears_caches=False,
        career_top_exclude_btb=False,
        media_captures_base_dir="",
        media_tolerance_minutes=3,
        media_watcher_enabled=True,
        media_watcher_debounce_seconds=5,
        discord_notifications_enabled=False,
        discord_webhook_url_present=False,
        discord_notify_sync=True,
        discord_notify_backfill=True,
        discord_notify_new_version=True,
        discord_notify_new_media=True,
        spnkr_refresh_with_backfill=False,
        spnkr_refresh_backfill_medals=False,
        spnkr_refresh_backfill_skill=False,
        spnkr_refresh_backfill_aliases=False,
        spnkr_refresh_backfill_personal_scores=False,
        spnkr_refresh_backfill_performance_scores=True,
        spnkr_refresh_backfill_lusr=True,
        spnkr_refresh_backfill_events=False,
        spnkr_refresh_backfill_weapons=False,
    )

    # update_api_settings est importé lazily dans le router ; patcher à la source.
    with patch(
        "apps.api.app.services.settings_service.update_api_settings",
        return_value=default_settings,
    ):
        resp = await client.patch("/api/v1/settings", json={"lang": "en"})

    assert resp.status_code == 200
    data = resp.json()
    assert data["lang"] == "en"
    assert "discord_webhook_url" not in data


@pytest.mark.anyio
async def test_patch_settings_empty_body_is_ok(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """PATCH /settings avec body vide est valide (no-op)."""
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]

    from apps.api.app.schemas.settings import SettingsResponse

    default_settings = SettingsResponse(
        lang="fr",
        discord_lang="fr",
        user_timezone="Europe/Paris",
        normalize_mode_labels=True,
        show_records=False,
        refresh_clears_caches=False,
        career_top_exclude_btb=False,
        media_captures_base_dir="",
        media_tolerance_minutes=3,
        media_watcher_enabled=True,
        media_watcher_debounce_seconds=5,
        discord_notifications_enabled=False,
        discord_webhook_url_present=False,
        discord_notify_sync=True,
        discord_notify_backfill=True,
        discord_notify_new_version=True,
        discord_notify_new_media=True,
        spnkr_refresh_with_backfill=False,
        spnkr_refresh_backfill_medals=False,
        spnkr_refresh_backfill_skill=False,
        spnkr_refresh_backfill_aliases=False,
        spnkr_refresh_backfill_personal_scores=False,
        spnkr_refresh_backfill_performance_scores=True,
        spnkr_refresh_backfill_lusr=True,
        spnkr_refresh_backfill_events=False,
        spnkr_refresh_backfill_weapons=False,
    )

    with patch(
        "apps.api.app.services.settings_service.update_api_settings",
        return_value=default_settings,
    ):
        resp = await client.patch("/api/v1/settings", json={})

    assert resp.status_code == 200


# ===========================================================================
# POST /settings/media/reset-index
# ===========================================================================


@pytest.mark.anyio
async def test_media_reset_without_confirm_returns_400(client: AsyncClient) -> None:
    """Sans confirm_destructive=True, retourne 400."""
    # Note : en DEMO_MODE on teste quand même cette validation
    resp = await client.post(
        "/api/v1/settings/media/reset-index",
        json={"confirm_destructive": False},
    )
    assert resp.status_code == 400
    assert resp.json()["code"] == "confirmation_required"


@pytest.mark.anyio
async def test_media_reset_with_confirm_creates_job(client: AsyncClient) -> None:
    """confirm_destructive=True crée un job asynchrone (202)."""
    # reset_media_index est importé lazily dans le router ; patcher à la source.
    # On ne patch PAS threading.Thread.start pour éviter de bloquer anyio.
    # Le thread démarre avec _reset mocké → se termine en <1ms.
    with patch(
        "apps.api.app.services.settings_service.reset_media_index",
        return_value=None,
    ):
        resp = await client.post(
            "/api/v1/settings/media/reset-index",
            json={"confirm_destructive": True, "reindex_after_reset": False},
        )

    assert resp.status_code == 202
    data = resp.json()
    assert data["job_type"] == "reindex_media"
    # Le thread background peut avoir fini avant la lecture (succeeded) ou pas encore (queued/running)
    assert data["status"] in ("queued", "running", "succeeded")
    assert "job_id" in data
