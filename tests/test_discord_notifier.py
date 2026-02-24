"""Tests unitaires pour src/utils/discord_notifier.py.

Couvre :
- Dataclasses (LastMatchInfo, DiscordPlayerResult)
- Formatage de durée (_format_duration)
- Formatage des champs joueur (_format_player_field)
- Construction du payload embed (build_embed_payload)
- Envoi HTTP (send_discord_notification) — User-Agent, statuts, exceptions
- Point d'entrée public (notify_operation_done) — failsafe, disabled, no webhook
- Lecture de la configuration (_get_webhook_url, _load_app_settings)
"""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from datetime import datetime, timedelta, timezone
from io import BytesIO
from pathlib import Path
from typing import Any
from unittest.mock import MagicMock, patch

from src.utils.discord_notifier import (
    DiscordPlayerResult,
    LastMatchInfo,
    _format_duration,
    _format_player_field,
    _load_app_settings,
    build_embed_payload,
    count_matches_missing_data,
    notify_operation_done,
    send_discord_notification,
)

# =============================================================================
# Fixtures
# =============================================================================

_NOW = datetime(2026, 2, 24, 12, 0, 0, tzinfo=timezone.utc)
_WEBHOOK = "https://discord.com/api/webhooks/123456789/abcdefghij_token"


def _player(
    gamertag: str = "JGtm",
    matches_synced: int = 5,
    missing: int = 0,
    error: str | None = None,
    last_match: LastMatchInfo | None = None,
) -> DiscordPlayerResult:
    return DiscordPlayerResult(
        gamertag=gamertag,
        xuid="2533274823110022",
        matches_synced=matches_synced,
        missing_data_count=missing,
        last_match=last_match,
        error=error,
    )


def _last_match(outcome: int = 2, is_ranked: bool = False) -> LastMatchInfo:
    return LastMatchInfo(
        match_id="abc123",
        map_name="Bazaar",
        playlist_name="Quick Play",
        game_variant_name="Slayer",
        is_ranked=is_ranked,
        start_time=_NOW - timedelta(hours=1),
        kills=10,
        deaths=4,
        assists=2,
        outcome=outcome,
        score=1200,
    )


def _mock_urlopen_ok(status: int = 204) -> MagicMock:
    """Simule une réponse HTTP positive de Discord."""
    resp = MagicMock()
    resp.status = status
    resp.__enter__ = lambda s: s
    resp.__exit__ = MagicMock(return_value=False)
    return resp


# =============================================================================
# Tests — LastMatchInfo
# =============================================================================


class TestLastMatchInfo:
    def test_default_values(self):
        lm = LastMatchInfo()
        assert lm.map_name == "—"
        assert lm.kills == 0
        assert lm.outcome == 0
        assert lm.is_ranked is False

    def test_custom_values(self):
        lm = _last_match(outcome=2, is_ranked=True)
        assert lm.map_name == "Bazaar"
        assert lm.kills == 10
        assert lm.outcome == 2
        assert lm.is_ranked is True


# =============================================================================
# Tests — DiscordPlayerResult
# =============================================================================


class TestDiscordPlayerResult:
    def test_defaults(self):
        p = DiscordPlayerResult(gamertag="Test")
        assert p.matches_synced == 0
        assert p.missing_data_count == 0
        assert p.last_match is None
        assert p.error is None

    def test_with_error(self):
        p = _player(error="Connexion API échouée")
        assert p.error == "Connexion API échouée"


# =============================================================================
# Tests — _format_duration
# =============================================================================


class TestFormatDuration:
    def test_seconds_only(self):
        t0 = _NOW
        t1 = _NOW + timedelta(seconds=42)
        assert _format_duration(t0, t1) == "42s"

    def test_minutes_and_seconds(self):
        t0 = _NOW
        t1 = _NOW + timedelta(minutes=3, seconds=7)
        assert _format_duration(t0, t1) == "3m07s"

    def test_minutes_zero_seconds(self):
        t0 = _NOW
        t1 = _NOW + timedelta(minutes=5)
        assert _format_duration(t0, t1) == "5m00s"

    def test_hours(self):
        t0 = _NOW
        t1 = _NOW + timedelta(hours=1, minutes=23, seconds=45)
        assert _format_duration(t0, t1) == "1h23m45s"

    def test_negative_duration_returns_zero(self):
        # finished_at < started_at → 0s
        t0 = _NOW + timedelta(seconds=10)
        t1 = _NOW
        assert _format_duration(t0, t1) == "0s"

    def test_exactly_one_minute(self):
        t0 = _NOW
        t1 = _NOW + timedelta(seconds=60)
        assert _format_duration(t0, t1) == "1m00s"


# =============================================================================
# Tests — _format_player_field
# =============================================================================


class TestFormatPlayerField:
    def test_sync_label_starts_with_plus(self):
        p = _player(matches_synced=12)
        name, value = _format_player_field(p, "sync_delta")
        assert "+12" in value
        assert "synchronisé" in value

    def test_backfill_label_no_plus(self):
        p = _player(matches_synced=8)
        name, value = _format_player_field(p, "backfill")
        assert "8" in value
        assert "traité" in value
        assert "+8" not in value

    def test_complete_data_shows_checkmark(self):
        p = _player(missing=0)
        _, value = _format_player_field(p, "sync_delta")
        assert "✅" in value

    def test_missing_data_shows_warning(self):
        p = _player(missing=15)
        _, value = _format_player_field(p, "sync_delta")
        assert "⚠️" in value
        assert "15" in value

    def test_error_shows_stop_sign(self):
        p = _player(error="Échec API")
        _, value = _format_player_field(p, "sync_delta")
        assert "⛔" in value
        assert "Échec API" in value

    def test_last_match_win_shows_trophy(self):
        p = _player(last_match=_last_match(outcome=2))
        _, value = _format_player_field(p, "sync_delta")
        assert "Bazaar" in value
        assert "🏆" in value
        assert "10K / 4D / 2A" in value

    def test_last_match_loss_shows_skull(self):
        p = _player(last_match=_last_match(outcome=3))
        _, value = _format_player_field(p, "sync_delta")
        assert "💀" in value

    def test_last_match_ranked_shows_medal(self):
        p = _player(last_match=_last_match(is_ranked=True))
        _, value = _format_player_field(p, "sync_delta")
        assert "🏅" in value

    def test_last_match_not_ranked_no_medal(self):
        p = _player(last_match=_last_match(is_ranked=False))
        _, value = _format_player_field(p, "sync_delta")
        assert "🏅" not in value

    def test_no_last_match_omits_match_section(self):
        p = _player(last_match=None)
        _, value = _format_player_field(p, "sync_delta")
        assert "Dernier match" not in value

    def test_name_contains_gamertag(self):
        p = _player(gamertag="Chocoboflor")
        name, _ = _format_player_field(p, "sync_delta")
        assert "Chocoboflor" in name

    def test_name_max_256_chars(self):
        long_tag = "A" * 300
        p = _player(gamertag=long_tag)
        name, _ = _format_player_field(p, "sync_delta")
        assert len(name) <= 256

    def test_value_max_1024_chars(self):
        long_error = "E" * 2000
        p = _player(error=long_error)
        _, value = _format_player_field(p, "sync_delta")
        assert len(value) <= 1024

    def test_unknown_outcome_shows_question_mark(self):
        lm = _last_match(outcome=99)
        p = _player(last_match=lm)
        _, value = _format_player_field(p, "sync_delta")
        assert "❓" in value


# =============================================================================
# Tests — build_embed_payload
# =============================================================================


class TestBuildEmbedPayload:
    def _build(self, players, success=True, operation="sync_delta"):
        t0 = _NOW
        t1 = _NOW + timedelta(minutes=2, seconds=30)
        return build_embed_payload(operation, t0, t1, players, success)

    def test_structure_has_embeds_key(self):
        payload = self._build([_player()])
        assert "embeds" in payload
        assert len(payload["embeds"]) == 1

    def test_embed_has_required_fields(self):
        embed = self._build([_player()])["embeds"][0]
        assert "title" in embed
        assert "description" in embed
        assert "color" in embed
        assert "fields" in embed
        assert "footer" in embed
        assert "timestamp" in embed

    def test_color_green_when_success_and_complete(self):
        embed = self._build([_player(missing=0)], success=True)["embeds"][0]
        assert embed["color"] == 0x57F287

    def test_color_yellow_when_missing_data(self):
        embed = self._build([_player(missing=5)], success=True)["embeds"][0]
        assert embed["color"] == 0xFEE75C

    def test_color_red_when_failure(self):
        embed = self._build([_player()], success=False)["embeds"][0]
        assert embed["color"] == 0xED4245

    def test_color_red_when_error_in_player(self):
        embed = self._build([_player(error="crash")], success=True)["embeds"][0]
        assert embed["color"] == 0xED4245

    def test_description_contains_duration(self):
        embed = self._build([_player()])["embeds"][0]
        assert "2m30s" in embed["description"]

    def test_description_contains_player_count(self):
        embed = self._build([_player(), _player(gamertag="Other")])["embeds"][0]
        assert "2 joueur" in embed["description"]

    def test_description_contains_total_synced(self):
        players = [_player(matches_synced=5), _player(gamertag="B", matches_synced=3)]
        embed = self._build(players)["embeds"][0]
        assert "8 match" in embed["description"]

    def test_fields_one_per_player(self):
        players = [_player(), _player(gamertag="B"), _player(gamertag="C")]
        embed = self._build(players)["embeds"][0]
        assert len(embed["fields"]) == 3

    def test_fields_capped_at_25(self):
        players = [_player(gamertag=f"Player{i}") for i in range(30)]
        embed = self._build(players)["embeds"][0]
        assert len(embed["fields"]) == 25

    def test_title_contains_op_label(self):
        embed = self._build([_player()], operation="sync_delta")["embeds"][0]
        assert "Sync delta" in embed["title"]

    def test_title_backfill(self):
        embed = self._build([_player()], operation="backfill")["embeds"][0]
        assert "Backfill" in embed["title"]

    def test_footer_text(self):
        embed = self._build([_player()])["embeds"][0]
        assert "LevelUp" in embed["footer"]["text"]

    def test_timestamp_is_iso_format(self):
        embed = self._build([_player()])["embeds"][0]
        # Doit être parseable comme datetime ISO
        ts = embed["timestamp"]
        parsed = datetime.fromisoformat(ts)
        assert parsed is not None

    def test_fields_inline_false(self):
        embed = self._build([_player()])["embeds"][0]
        for field in embed["fields"]:
            assert field["inline"] is False

    def test_empty_players_list_no_fields(self):
        t0, t1 = _NOW, _NOW + timedelta(seconds=5)
        payload = build_embed_payload("sync_delta", t0, t1, [], True)
        embed = payload["embeds"][0]
        assert embed["fields"] == []

    def test_payload_serializable_to_json(self):
        payload = self._build([_player(last_match=_last_match())])
        # Ne doit pas lever d'exception
        serialized = json.dumps(payload, ensure_ascii=False)
        assert len(serialized) > 0


# =============================================================================
# Tests — send_discord_notification
# =============================================================================


class TestSendDiscordNotification:
    """Tests critiques pour l'envoi HTTP — dont le User-Agent qui causa la 403."""

    SIMPLE_PAYLOAD = {"embeds": [{"title": "Test", "description": "OK"}]}

    def test_sends_post_request(self):
        """Vérifie qu'un POST est bien émis (pas GET ni autre)."""
        with patch("urllib.request.urlopen") as mock_open:
            mock_open.return_value.__enter__ = lambda _: _mock_urlopen_ok(204)
            mock_open.return_value.__exit__ = MagicMock(return_value=False)
            captured_req: list[urllib.request.Request] = []

            def capture(req, timeout=None):
                captured_req.append(req)
                return _mock_urlopen_ok(204)

            mock_open.side_effect = capture
            send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)

        assert len(captured_req) == 1
        assert captured_req[0].method == "POST"

    def test_user_agent_is_not_python_urllib(self):
        """Régression : User-Agent Python générique → 403 Cloudflare.

        Le User-Agent doit être un agent custom, pas 'Python-urllib/...'
        qui est bloqué par Cloudflare devant l'API Discord.
        """
        captured_req: list[urllib.request.Request] = []

        def capture(req, timeout=None):
            captured_req.append(req)
            return _mock_urlopen_ok(204)

        with patch("urllib.request.urlopen", side_effect=capture):
            send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)

        assert len(captured_req) == 1
        ua = captured_req[0].get_header("User-agent")
        assert ua is not None, "User-Agent manquant dans la requête"
        assert "python-urllib" not in ua.lower(), (
            f"User-Agent Python générique détecté : '{ua}'. "
            "Cloudflare bloque ces requêtes avec 403."
        )
        assert (
            "LevelUp" in ua or "levelup" in ua.lower() or "bot" in ua.lower()
        ), f"User-Agent doit identifier l'application, obtenu : '{ua}'"

    def test_content_type_is_json(self):
        """Content-Type doit être application/json."""
        captured_req: list[urllib.request.Request] = []

        def capture(req, timeout=None):
            captured_req.append(req)
            return _mock_urlopen_ok(204)

        with patch("urllib.request.urlopen", side_effect=capture):
            send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)

        ct = captured_req[0].get_header("Content-type")
        assert ct is not None
        assert "application/json" in ct

    def test_returns_true_on_204(self):
        with patch(
            "urllib.request.urlopen", side_effect=lambda _r, timeout=None: _mock_urlopen_ok(204)
        ):
            result = send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)
        assert result is True

    def test_returns_true_on_200(self):
        with patch(
            "urllib.request.urlopen", side_effect=lambda _r, timeout=None: _mock_urlopen_ok(200)
        ):
            result = send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)
        assert result is True

    def test_returns_false_on_403(self):
        def raise_403(req, timeout=None):
            raise urllib.error.HTTPError(
                _WEBHOOK, 403, "Forbidden", {}, BytesIO(b"error code: 1010")
            )

        with patch("urllib.request.urlopen", side_effect=raise_403):
            result = send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)
        assert result is False

    def test_returns_false_on_network_error(self):
        def raise_network(req, timeout=None):
            raise OSError("Network unreachable")

        with patch("urllib.request.urlopen", side_effect=raise_network):
            result = send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)
        assert result is False

    def test_body_is_valid_json(self):
        """Le corps envoyé à Discord doit être du JSON valide."""
        captured_data: list[bytes] = []

        def capture(req, timeout=None):
            captured_data.append(req.data)
            return _mock_urlopen_ok(204)

        with patch("urllib.request.urlopen", side_effect=capture):
            send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)

        assert len(captured_data) == 1
        parsed = json.loads(captured_data[0].decode("utf-8"))
        assert "embeds" in parsed

    def test_timeout_is_set(self):
        """Un timeout doit être passé à urlopen (éviter les blocages infinis)."""
        captured_timeouts: list[Any] = []

        def capture(req, timeout=None):
            captured_timeouts.append(timeout)
            return _mock_urlopen_ok(204)

        with patch("urllib.request.urlopen", side_effect=capture):
            send_discord_notification(self.SIMPLE_PAYLOAD, _WEBHOOK)

        assert len(captured_timeouts) == 1
        assert captured_timeouts[0] is not None
        assert captured_timeouts[0] > 0


# =============================================================================
# Tests — notify_operation_done (entrypoint public)
# =============================================================================


class TestNotifyOperationDone:
    """Tests du point d'entrée public — failsafe garanti."""

    def _call(self, players=None, success=True, disabled=False, webhook=_WEBHOOK):
        players = players or [_player()]
        t0 = _NOW - timedelta(minutes=1)
        t1 = _NOW
        mock_settings = {"discord_notifications_enabled": True}

        with (
            patch("src.utils.discord_notifier._load_app_settings", return_value=mock_settings),
            patch("os.environ", {**os.environ, "DISCORD_WEBHOOK_URL": webhook}),
            patch(
                "urllib.request.urlopen",
                side_effect=lambda _r, _timeout=None: _mock_urlopen_ok(204),
            ) as mock_http,
        ):
            notify_operation_done("sync_delta", t0, t1, players, success, disabled=disabled)
            return mock_http

    def test_disabled_flag_skips_http(self):
        mock_http = self._call(disabled=True)
        mock_http.assert_not_called()

    def test_sends_when_enabled(self):
        mock_http = self._call(disabled=False)
        mock_http.assert_called_once()

    def test_no_players_skips_http(self):
        t0, t1 = _NOW - timedelta(minutes=1), _NOW
        mock_settings = {"discord_notifications_enabled": True}
        with (
            patch("src.utils.discord_notifier._load_app_settings", return_value=mock_settings),
            patch("os.environ", {**os.environ, "DISCORD_WEBHOOK_URL": _WEBHOOK}),
            patch("urllib.request.urlopen") as mock_http,
        ):
            notify_operation_done("sync_delta", t0, t1, [], success=True)
            mock_http.assert_not_called()

    def test_no_webhook_skips_http(self):
        mock_http = self._call(webhook="")
        mock_http.assert_not_called()

    def test_notifications_disabled_in_settings_skips_http(self):
        t0, t1 = _NOW - timedelta(minutes=1), _NOW
        mock_settings = {"discord_notifications_enabled": False}
        with (
            patch("src.utils.discord_notifier._load_app_settings", return_value=mock_settings),
            patch("urllib.request.urlopen") as mock_http,
        ):
            notify_operation_done("sync_delta", t0, t1, [_player()], success=True)
            mock_http.assert_not_called()

    def test_failsafe_on_http_exception(self):
        """Une exception HTTP ne doit JAMAIS remonter."""
        t0, t1 = _NOW - timedelta(minutes=1), _NOW
        mock_settings = {"discord_notifications_enabled": True}

        def raise_always(req, timeout=None):
            raise RuntimeError("Crash simulé")

        with (
            patch("src.utils.discord_notifier._load_app_settings", return_value=mock_settings),
            patch("os.environ", {**os.environ, "DISCORD_WEBHOOK_URL": _WEBHOOK}),
            patch("urllib.request.urlopen", side_effect=raise_always),
        ):
            # Ne doit pas lever d'exception
            notify_operation_done("sync_delta", t0, t1, [_player()], success=True)

    def test_failsafe_on_internal_exception(self):
        """Une exception dans build_embed_payload ne doit JAMAIS remonter."""
        t0, t1 = _NOW - timedelta(minutes=1), _NOW
        mock_settings = {"discord_notifications_enabled": True}

        with (
            patch("src.utils.discord_notifier._load_app_settings", return_value=mock_settings),
            patch("os.environ", {**os.environ, "DISCORD_WEBHOOK_URL": _WEBHOOK}),
            patch(
                "src.utils.discord_notifier.build_embed_payload",
                side_effect=ValueError("bug interne"),
            ),
        ):
            notify_operation_done("sync_delta", t0, t1, [_player()], success=True)


# =============================================================================
# Tests — _load_app_settings
# =============================================================================


class TestLoadAppSettings:
    def test_returns_empty_dict_when_file_missing(self, tmp_path):
        with patch("src.utils.discord_notifier._APP_SETTINGS", tmp_path / "missing.json"):
            result = _load_app_settings()
        assert result == {}

    def test_returns_empty_dict_on_invalid_json(self, tmp_path):
        bad = tmp_path / "bad.json"
        bad.write_text("{ not valid json }", encoding="utf-8")
        with patch("src.utils.discord_notifier._APP_SETTINGS", bad):
            result = _load_app_settings()
        assert result == {}

    def test_reads_valid_settings(self, tmp_path):
        good = tmp_path / "settings.json"
        good.write_text(json.dumps({"discord_notifications_enabled": True}), encoding="utf-8")
        with patch("src.utils.discord_notifier._APP_SETTINGS", good):
            result = _load_app_settings()
        assert result["discord_notifications_enabled"] is True


# =============================================================================
# Tests — _get_webhook_url
# =============================================================================


class TestGetWebhookUrl:
    def test_returns_none_when_disabled(self, tmp_path):
        settings = tmp_path / "s.json"
        settings.write_text(json.dumps({"discord_notifications_enabled": False}), encoding="utf-8")
        with patch("src.utils.discord_notifier._APP_SETTINGS", settings):
            from src.utils.discord_notifier import _get_webhook_url

            assert _get_webhook_url() is None

    def test_returns_url_from_env_var(self, tmp_path):
        settings = tmp_path / "s.json"
        settings.write_text(json.dumps({"discord_notifications_enabled": True}), encoding="utf-8")
        with (
            patch("src.utils.discord_notifier._APP_SETTINGS", settings),
            patch.dict(os.environ, {"DISCORD_WEBHOOK_URL": _WEBHOOK}, clear=False),
            patch(
                "src.utils.discord_notifier._load_dotenv_if_present_safe",
                return_value=None,
                create=True,
            ),
        ):
            from src.utils.discord_notifier import _get_webhook_url

            result = _get_webhook_url()
        assert result == _WEBHOOK

    def test_env_var_takes_priority_over_settings(self, tmp_path):
        other_webhook = "https://discord.com/api/webhooks/999/other"
        settings = tmp_path / "s.json"
        settings.write_text(
            json.dumps(
                {"discord_notifications_enabled": True, "discord_webhook_url": other_webhook}
            ),
            encoding="utf-8",
        )
        with (
            patch("src.utils.discord_notifier._APP_SETTINGS", settings),
            patch.dict(os.environ, {"DISCORD_WEBHOOK_URL": _WEBHOOK}, clear=False),
        ):
            from src.utils.discord_notifier import _get_webhook_url

            # L'env var doit primer sur app_settings.json
            result = _get_webhook_url()
        assert result == _WEBHOOK

    def test_fallback_to_app_settings_when_no_env_var(self, tmp_path):
        settings = tmp_path / "s.json"
        settings.write_text(
            json.dumps({"discord_notifications_enabled": True, "discord_webhook_url": _WEBHOOK}),
            encoding="utf-8",
        )
        env_without_discord = {k: v for k, v in os.environ.items() if k != "DISCORD_WEBHOOK_URL"}
        with (
            patch("src.utils.discord_notifier._APP_SETTINGS", settings),
            patch("src.ui.profile_api_tokens._load_dotenv_if_present"),
            patch.dict(os.environ, env_without_discord, clear=True),
        ):
            from src.utils.discord_notifier import _get_webhook_url

            result = _get_webhook_url()
        assert result == _WEBHOOK

    def test_returns_none_for_invalid_url(self, tmp_path):
        settings = tmp_path / "s.json"
        settings.write_text(
            json.dumps(
                {
                    "discord_notifications_enabled": True,
                    "discord_webhook_url": "https://example.com/bad",
                }
            ),
            encoding="utf-8",
        )
        env_without_discord = {k: v for k, v in os.environ.items() if k != "DISCORD_WEBHOOK_URL"}
        with (
            patch("src.utils.discord_notifier._APP_SETTINGS", settings),
            patch("src.ui.profile_api_tokens._load_dotenv_if_present"),
            patch.dict(os.environ, env_without_discord, clear=True),
        ):
            from src.utils.discord_notifier import _get_webhook_url

            result = _get_webhook_url()
        assert result is None

    def test_returns_none_for_empty_url(self, tmp_path):
        settings = tmp_path / "s.json"
        settings.write_text(
            json.dumps({"discord_notifications_enabled": True, "discord_webhook_url": ""}),
            encoding="utf-8",
        )
        env_without_discord = {k: v for k, v in os.environ.items() if k != "DISCORD_WEBHOOK_URL"}
        with (
            patch("src.utils.discord_notifier._APP_SETTINGS", settings),
            patch("src.ui.profile_api_tokens._load_dotenv_if_present"),
            patch.dict(os.environ, env_without_discord, clear=True),
        ):
            from src.utils.discord_notifier import _get_webhook_url

            result = _get_webhook_url()
        assert result is None


# =============================================================================
# Tests — count_matches_missing_data
# =============================================================================


class TestCountMatchesMissingData:
    """Vérifie que le compteur de données incomplètes tient compte à la fois des
    flags booléens (medals_loaded / events_loaded) ET du bitmask backfill_completed.

    Logique attendue :
    - medals manquants  : medals_loaded=FALSE  ET (backfill_completed & 1) = 0
    - events manquants  : events_loaded=FALSE   ET (backfill_completed & 2) = 0
    Un match est complet dès que medals ET events sont couverts par l'une ou
    l'autre source.
    """

    XUID = "2533274823110022"
    MATCH_A = "match-alpha-0001"
    MATCH_B = "match-bravo-0002"
    MATCH_C = "match-charlie-003"

    def _setup_db(self, tmp_path: Path, rows: list[dict]) -> Path:
        """Crée un shared_matches.duckdb minimal avec les lignes fournies.

        Chaque dict dans rows contient :
            match_id, medals_loaded, events_loaded, backfill_completed
        """
        import duckdb

        db = tmp_path / "shared_matches.duckdb"
        conn = duckdb.connect(str(db))
        conn.execute(
            """
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                backfill_completed INTEGER DEFAULT 0,
                participants_loaded BOOLEAN DEFAULT FALSE,
                events_loaded BOOLEAN DEFAULT FALSE,
                medals_loaded BOOLEAN DEFAULT FALSE,
                player_count INTEGER DEFAULT 0
            )
            """
        )
        conn.execute(
            """
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR
            )
            """
        )
        for r in rows:
            conn.execute(
                "INSERT INTO match_registry VALUES (?, ?, FALSE, ?, ?, 2)",
                [
                    r["match_id"],
                    r.get("backfill_completed", 0),
                    r.get("events_loaded", False),
                    r.get("medals_loaded", False),
                ],
            )
            conn.execute(
                "INSERT INTO match_participants VALUES (?, ?)",
                [r["match_id"], self.XUID],
            )
        conn.close()
        return db

    def test_returns_zero_when_all_loaded_by_sync(self, tmp_path):
        """medals_loaded=TRUE et events_loaded=TRUE → sync a tout chargé → 0 incomplets."""
        db = self._setup_db(
            tmp_path,
            [{"match_id": self.MATCH_A, "medals_loaded": True, "events_loaded": True}],
        )
        with patch("src.utils.discord_notifier._SHARED_DB", db):
            assert count_matches_missing_data(self.XUID) == 0

    def test_returns_zero_when_backfilled_via_bitmask(self, tmp_path):
        """medals_loaded=FALSE mais backfill_completed & 3 = 3 (médailles + events) → 0."""
        db = self._setup_db(
            tmp_path,
            [
                {
                    "match_id": self.MATCH_A,
                    "medals_loaded": False,
                    "events_loaded": False,
                    "backfill_completed": 3,  # bit 0 (medals) + bit 1 (events)
                }
            ],
        )
        with patch("src.utils.discord_notifier._SHARED_DB", db):
            assert count_matches_missing_data(self.XUID) == 0

    def test_counts_match_with_no_medals_and_no_backfill(self, tmp_path):
        """medals_loaded=FALSE, backfill_completed & 1 = 0 → medals réellement manquantes → 1."""
        db = self._setup_db(
            tmp_path,
            [
                {
                    "match_id": self.MATCH_A,
                    "medals_loaded": False,
                    "events_loaded": True,  # events OK
                    "backfill_completed": 0,
                }
            ],
        )
        with patch("src.utils.discord_notifier._SHARED_DB", db):
            assert count_matches_missing_data(self.XUID) == 1

    def test_counts_match_with_no_events_and_no_backfill(self, tmp_path):
        """events_loaded=FALSE, backfill_completed & 2 = 0 → events réellement manquants → 1."""
        db = self._setup_db(
            tmp_path,
            [
                {
                    "match_id": self.MATCH_A,
                    "medals_loaded": True,  # medals OK
                    "events_loaded": False,
                    "backfill_completed": 0,
                }
            ],
        )
        with patch("src.utils.discord_notifier._SHARED_DB", db):
            assert count_matches_missing_data(self.XUID) == 1

    def test_medals_backfilled_but_events_still_missing(self, tmp_path):
        """backfill_completed = 1 (medals seulement) mais events absents → 1 incomplet."""
        db = self._setup_db(
            tmp_path,
            [
                {
                    "match_id": self.MATCH_A,
                    "medals_loaded": False,
                    "events_loaded": False,
                    "backfill_completed": 1,  # bit 0 seul, events (bit 1) manquants
                }
            ],
        )
        with patch("src.utils.discord_notifier._SHARED_DB", db):
            assert count_matches_missing_data(self.XUID) == 1

    def test_mixed_matches_counts_only_truly_missing(self, tmp_path):
        """3 matchs : 1 complet via sync, 1 via backfill, 1 vraiment manquant → 1."""
        db = self._setup_db(
            tmp_path,
            [
                # complet via sync
                {
                    "match_id": self.MATCH_A,
                    "medals_loaded": True,
                    "events_loaded": True,
                    "backfill_completed": 0,
                },
                # complet via backfill (bits 0+1)
                {
                    "match_id": self.MATCH_B,
                    "medals_loaded": False,
                    "events_loaded": False,
                    "backfill_completed": 3,
                },
                # réellement incomplet
                {
                    "match_id": self.MATCH_C,
                    "medals_loaded": False,
                    "events_loaded": False,
                    "backfill_completed": 0,
                },
            ],
        )
        with patch("src.utils.discord_notifier._SHARED_DB", db):
            assert count_matches_missing_data(self.XUID) == 1

    def test_returns_zero_for_empty_xuid(self, tmp_path):
        """XUID vide → 0 sans planter."""
        db = self._setup_db(tmp_path, [])
        with patch("src.utils.discord_notifier._SHARED_DB", db):
            assert count_matches_missing_data("") == 0

    def test_returns_zero_when_db_missing(self, tmp_path):
        """DB absente → 0 sans planter."""
        missing_db = tmp_path / "nope.duckdb"
        with patch("src.utils.discord_notifier._SHARED_DB", missing_db):
            assert count_matches_missing_data(self.XUID) == 0
