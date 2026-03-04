"""Tests unitaires pour scripts/monitor_uptime.py.

Couvre :
- Chargement/sauvegarde de l'état persistant
- Vérification HTTP (_check_site)
- Traductions (_t)
- Construction des payloads Discord
- Logique de debounce anti-flapping
- Transitions d'état complètes (main)
"""

from __future__ import annotations

import importlib
import sys
from datetime import datetime, timezone
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

_SCRIPTS_DIR = Path(__file__).resolve().parents[1] / "scripts"
if str(_SCRIPTS_DIR.parent) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS_DIR.parent))

# Import du module (sans déclencher les effets de bord car _setup_logging
# n'est appelé que dans main())
monitor_uptime = importlib.import_module("scripts.monitor_uptime")

_t = monitor_uptime._t
_check_site = monitor_uptime._check_site
_load_state = monitor_uptime._load_state
_save_state = monitor_uptime._save_state
_build_online_payload = monitor_uptime._build_online_payload
_build_offline_payload = monitor_uptime._build_offline_payload
_load_app_settings = monitor_uptime._load_app_settings
_DEFAULT_OFFLINE_THRESHOLD = monitor_uptime._DEFAULT_OFFLINE_THRESHOLD


# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture()
def state_file(tmp_path: Path) -> Path:
    """Fichier d'état temporaire."""
    return tmp_path / "uptime_state.json"


@pytest.fixture()
def _now() -> datetime:
    """Datetime UTC fixe pour les tests."""
    return datetime(2026, 3, 2, 12, 0, 0, tzinfo=timezone.utc)


# =============================================================================
# Tests : _t (traductions)
# =============================================================================


class TestTranslations:
    """Tests de la fonction de traduction _t."""

    def test_french_key(self) -> None:
        assert _t("online_title", "fr") == "✅ LevelUp — Site en ligne"

    def test_english_key(self) -> None:
        assert _t("offline_title", "en") == "❌ LevelUp — Dashboard offline"

    def test_unknown_lang_falls_back_to_fr(self) -> None:
        assert _t("footer", "de") == "LevelUp Uptime Monitor"

    def test_unknown_key_returns_placeholder(self) -> None:
        result = _t("nonexistent_key", "fr")
        assert result == "[nonexistent_key]"

    def test_unknown_key_unknown_lang(self) -> None:
        result = _t("nonexistent_key", "ja")
        assert result == "[nonexistent_key]"


# =============================================================================
# Tests : _load_state / _save_state
# =============================================================================


class TestStatePersistence:
    """Tests de la gestion de l'état persistant."""

    def test_load_missing_file(self, state_file: Path) -> None:
        """Un fichier inexistant retourne status=unknown."""
        result = _load_state(state_file)
        assert result == {"status": "unknown"}

    def test_load_corrupt_file(self, state_file: Path) -> None:
        """Un fichier JSON invalide retourne status=unknown."""
        state_file.write_text("not json!!!", encoding="utf-8")
        result = _load_state(state_file)
        assert result == {"status": "unknown"}

    def test_save_and_load_roundtrip(self, state_file: Path, _now: datetime) -> None:
        """Sauvegarde puis relecture retourne les mêmes données."""
        _save_state(
            "online",
            _now,
            url="https://test.ts.net",
            consecutive_failures=0,
            state_file=state_file,
        )
        loaded = _load_state(state_file)
        assert loaded["status"] == "online"
        assert loaded["url"] == "https://test.ts.net"
        assert loaded["consecutive_failures"] == 0

    def test_save_preserves_previous_url(self, state_file: Path, _now: datetime) -> None:
        """Si url est vide, l'URL du previous est conservée."""
        previous = {"status": "online", "url": "https://cached.ts.net"}
        _save_state(
            "offline",
            _now,
            url="",
            previous=previous,
            consecutive_failures=1,
            state_file=state_file,
        )
        loaded = _load_state(state_file)
        assert loaded["url"] == "https://cached.ts.net"

    def test_save_overwrites_url(self, state_file: Path, _now: datetime) -> None:
        """Si url est fournie, elle remplace la précédente."""
        previous = {"status": "online", "url": "https://old.ts.net"}
        _save_state(
            "online",
            _now,
            url="https://new.ts.net",
            previous=previous,
            state_file=state_file,
        )
        loaded = _load_state(state_file)
        assert loaded["url"] == "https://new.ts.net"

    def test_save_creates_parent_dirs(self, tmp_path: Path, _now: datetime) -> None:
        """_save_state crée les répertoires parents si nécessaire."""
        deep_path = tmp_path / "a" / "b" / "c" / "state.json"
        _save_state("online", _now, state_file=deep_path)
        assert deep_path.exists()

    def test_consecutive_failures_persisted(self, state_file: Path, _now: datetime) -> None:
        """Le compteur d'échecs consécutifs est bien persisté."""
        _save_state("online", _now, consecutive_failures=5, state_file=state_file)
        loaded = _load_state(state_file)
        assert loaded["consecutive_failures"] == 5


# =============================================================================
# Tests : _check_site
# =============================================================================


class TestCheckSite:
    """Tests de la vérification HTTP."""

    def test_site_up_200(self) -> None:
        """HTTP 200 → True."""
        mock_resp = MagicMock()
        mock_resp.status = 200
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)

        with patch("urllib.request.urlopen", return_value=mock_resp):
            assert _check_site("https://test.ts.net") is True

    def test_site_up_302(self) -> None:
        """HTTP 302 → True (redirect)."""
        mock_resp = MagicMock()
        mock_resp.status = 302
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)

        with patch("urllib.request.urlopen", return_value=mock_resp):
            assert _check_site("https://test.ts.net") is True

    def test_site_up_404(self) -> None:
        """HTTP 404 → True (site vivant, erreur applicative)."""
        import urllib.error

        error = urllib.error.HTTPError(
            url="https://test.ts.net",
            code=404,
            msg="Not Found",
            hdrs=None,
            fp=None,  # type: ignore[arg-type]
        )
        with patch("urllib.request.urlopen", side_effect=error):
            assert _check_site("https://test.ts.net") is True

    def test_site_down_500(self) -> None:
        """HTTP 500 → False."""
        import urllib.error

        error = urllib.error.HTTPError(
            url="https://test.ts.net",
            code=500,
            msg="ISE",
            hdrs=None,
            fp=None,  # type: ignore[arg-type]
        )
        with patch("urllib.request.urlopen", side_effect=error):
            assert _check_site("https://test.ts.net") is False

    def test_site_down_connection_error(self) -> None:
        """Erreur réseau → False."""
        with patch("urllib.request.urlopen", side_effect=ConnectionError("refused")):
            assert _check_site("https://test.ts.net") is False

    def test_site_down_timeout(self) -> None:
        """Timeout → False."""
        with patch("urllib.request.urlopen", side_effect=TimeoutError("timed out")):
            assert _check_site("https://test.ts.net") is False


# =============================================================================
# Tests : payloads Discord
# =============================================================================


class TestDiscordPayloads:
    """Tests de construction des payloads Discord."""

    def test_online_payload_structure(self, _now: datetime) -> None:
        """Le payload online contient les champs requis."""
        payload = _build_online_payload("https://test.ts.net", _now, "fr")
        assert "embeds" in payload
        embed = payload["embeds"][0]
        assert embed["color"] == 0x57F287
        assert "https://test.ts.net" in embed["description"]
        assert embed["footer"]["text"] == "LevelUp Uptime Monitor"

    def test_offline_payload_structure(self, _now: datetime) -> None:
        """Le payload offline contient les champs requis."""
        payload = _build_offline_payload(_now, "en")
        assert "embeds" in payload
        embed = payload["embeds"][0]
        assert embed["color"] == 0xED4245
        assert embed["title"] == "❌ LevelUp — Dashboard offline"

    def test_online_payload_lang_en(self, _now: datetime) -> None:
        """Le payload online en anglais utilise les bonnes traductions."""
        payload = _build_online_payload("https://test.ts.net", _now, "en")
        embed = payload["embeds"][0]
        assert "reachable" in embed["description"]

    def test_offline_payload_contains_timestamp(self, _now: datetime) -> None:
        """Le payload contient un timestamp ISO."""
        payload = _build_offline_payload(_now, "fr")
        assert payload["embeds"][0]["timestamp"] == _now.isoformat()


# =============================================================================
# Tests : _get_webhook_url
# =============================================================================


class TestGetWebhookUrl:
    """Tests de la résolution du webhook Discord."""

    def test_returns_none_when_notifications_disabled(self) -> None:
        """Retourne None si discord_notifications_enabled=False."""
        settings = {"discord_notifications_enabled": False}
        with patch.object(monitor_uptime, "_load_app_settings", return_value=settings):
            assert monitor_uptime._get_webhook_url() is None

    def test_returns_none_when_no_settings(self) -> None:
        """Retourne None si app_settings.json est vide."""
        with patch.object(monitor_uptime, "_load_app_settings", return_value={}):
            assert monitor_uptime._get_webhook_url() is None

    def test_returns_url_from_env(self) -> None:
        """Retourne l'URL depuis la variable d'environnement."""
        settings = {"discord_notifications_enabled": True}
        webhook = "https://discord.com/api/webhooks/123/abc"
        with (
            patch.object(monitor_uptime, "_load_app_settings", return_value=settings),
            patch.dict("os.environ", {"DISCORD_WEBHOOK_URL": webhook}),
        ):
            assert monitor_uptime._get_webhook_url() == webhook

    def test_returns_url_from_app_settings_fallback(self) -> None:
        """Retourne l'URL depuis app_settings.json si pas dans l'env."""
        webhook = "https://discord.com/api/webhooks/456/def"
        settings = {
            "discord_notifications_enabled": True,
            "discord_webhook_url": webhook,
        }
        with (
            patch.object(monitor_uptime, "_load_app_settings", return_value=settings),
            patch.dict("os.environ", {}, clear=False),
        ):
            # S'assurer que la var d'env n'est pas définie
            import os

            os.environ.pop("DISCORD_WEBHOOK_URL", None)
            assert monitor_uptime._get_webhook_url() == webhook

    def test_returns_none_for_invalid_url(self) -> None:
        """Retourne None pour une URL qui ne commence pas par le préfixe Discord."""
        settings = {"discord_notifications_enabled": True}
        with (
            patch.object(monitor_uptime, "_load_app_settings", return_value=settings),
            patch.dict("os.environ", {"DISCORD_WEBHOOK_URL": "https://evil.com/hook"}),
        ):
            assert monitor_uptime._get_webhook_url() is None


# =============================================================================
# Tests : logique de debounce (main)
# =============================================================================


class TestDebounceLogic:
    """Tests du mécanisme anti-flapping dans main()."""

    def _run_main(  # noqa: PLR0913
        self,
        *,
        is_up: bool,
        previous: dict,
        tailscale_url: str = "https://test.ts.net",
        webhook_url: str | None = "https://discord.com/api/webhooks/1/a",
        offline_threshold: int = 3,
        state_file: Path | None = None,
    ) -> tuple[dict | None, bool]:
        """Helper qui exécute la logique de main() et capture les résultats.

        Retourne (état_sauvegardé, notification_envoyée).
        """
        saved_state: dict | None = None
        notified = False

        def mock_save(  # noqa: PLR0913
            status, since, *, url="", previous=None, consecutive_failures=0, state_file=None
        ):
            nonlocal saved_state
            saved_state = {
                "status": status,
                "since": since.isoformat(),
                "url": url,
                "consecutive_failures": consecutive_failures,
            }

        def mock_notify_online(wh, su, since, lang):
            nonlocal notified
            notified = True

        def mock_notify_offline(wh, since, lang):
            nonlocal notified
            notified = True

        with (
            patch.object(monitor_uptime, "_setup_logging"),
            patch.object(monitor_uptime, "_load_secrets_once"),
            patch.object(monitor_uptime, "_load_state", return_value=previous),
            patch.object(monitor_uptime, "_resolve_tailscale_url", return_value=tailscale_url),
            patch.object(monitor_uptime, "_get_webhook_url", return_value=webhook_url),
            patch.object(monitor_uptime, "_check_site", return_value=is_up),
            patch.object(monitor_uptime, "_save_state", side_effect=mock_save),
            patch.object(monitor_uptime, "_notify_online", side_effect=mock_notify_online),
            patch.object(monitor_uptime, "_notify_offline", side_effect=mock_notify_offline),
            patch.object(monitor_uptime, "_get_lang", return_value="fr"),
            patch.dict("os.environ", {"UPTIME_OFFLINE_THRESHOLD": str(offline_threshold)}),
        ):
            monitor_uptime.main(offline_threshold=offline_threshold)

        return saved_state, notified

    def test_first_check_online(self) -> None:
        """Premier check réussi → transition unknown→online + notification."""
        previous = {"status": "unknown"}
        state, notified = self._run_main(is_up=True, previous=previous)
        assert state is not None
        assert state["status"] == "online"
        assert notified is True

    def test_stays_online_no_notification(self) -> None:
        """Site toujours online → pas de notification."""
        previous = {"status": "online", "url": "https://test.ts.net", "consecutive_failures": 0}
        state, notified = self._run_main(is_up=True, previous=previous)
        assert notified is False

    def test_first_failure_no_notification(self) -> None:
        """Premier échec (threshold=3) → compteur incrémenté, pas de notification."""
        previous = {"status": "online", "url": "https://test.ts.net", "consecutive_failures": 0}
        state, notified = self._run_main(is_up=False, previous=previous, offline_threshold=3)
        assert notified is False
        assert state is not None
        assert state["consecutive_failures"] == 1

    def test_second_failure_no_notification(self) -> None:
        """Deuxième échec (threshold=3) → compteur 2, pas de notification."""
        previous = {"status": "online", "url": "https://test.ts.net", "consecutive_failures": 1}
        state, notified = self._run_main(is_up=False, previous=previous, offline_threshold=3)
        assert notified is False
        assert state is not None
        assert state["consecutive_failures"] == 2

    def test_third_failure_triggers_notification(self) -> None:
        """Troisième échec (threshold=3) → notification OFFLINE envoyée."""
        previous = {"status": "online", "url": "https://test.ts.net", "consecutive_failures": 2}
        state, notified = self._run_main(is_up=False, previous=previous, offline_threshold=3)
        assert notified is True
        assert state is not None
        assert state["status"] == "offline"
        assert state["consecutive_failures"] == 3

    def test_threshold_1_immediate_offline(self) -> None:
        """Avec threshold=1, le premier échec déclenche la notification."""
        previous = {"status": "online", "url": "https://test.ts.net", "consecutive_failures": 0}
        state, notified = self._run_main(is_up=False, previous=previous, offline_threshold=1)
        assert notified is True
        assert state is not None
        assert state["status"] == "offline"

    def test_recovery_after_failures(self) -> None:
        """Site revient online après des échecs → notification ONLINE."""
        previous = {"status": "online", "url": "https://test.ts.net", "consecutive_failures": 2}
        state, notified = self._run_main(is_up=True, previous=previous)
        # Le site est toujours « online » (status précédent), pas de transition
        assert notified is False

    def test_recovery_from_offline(self) -> None:
        """Passage offline→online → notification ONLINE immédiate."""
        previous = {"status": "offline", "url": "https://test.ts.net", "consecutive_failures": 3}
        state, notified = self._run_main(is_up=True, previous=previous)
        assert notified is True
        assert state is not None
        assert state["status"] == "online"
        assert state["consecutive_failures"] == 0

    def test_stays_offline_no_notification(self) -> None:
        """Site toujours offline → pas de notification supplémentaire."""
        previous = {"status": "offline", "url": "https://test.ts.net", "consecutive_failures": 5}
        state, notified = self._run_main(is_up=False, previous=previous)
        assert notified is False

    def test_no_webhook_no_notification(self) -> None:
        """Pas de webhook → transition sans notification."""
        previous = {"status": "offline", "url": "https://test.ts.net", "consecutive_failures": 3}
        state, notified = self._run_main(is_up=True, previous=previous, webhook_url=None)
        assert notified is False
        assert state is not None
        assert state["status"] == "online"

    def test_url_change_updates_state(self) -> None:
        """Changement d'URL sans changement d'état → state mis à jour."""
        previous = {"status": "online", "url": "https://old.ts.net", "consecutive_failures": 0}
        state, notified = self._run_main(
            is_up=True, previous=previous, tailscale_url="https://new.ts.net"
        )
        assert notified is False
        assert state is not None
        assert state["url"] == "https://new.ts.net"


# =============================================================================
# Tests : _send_discord (retry)
# =============================================================================


class TestSendDiscordRetry:
    """Tests du mécanisme de retry Discord."""

    @patch("scripts.monitor_uptime.time.sleep")
    @patch("src.utils.discord_notifier.send_discord_notification")
    def test_success_first_try(self, mock_send: MagicMock, mock_sleep: MagicMock) -> None:
        """Succès au premier essai → pas de retry."""
        mock_send.return_value = True
        result = monitor_uptime._send_discord("https://hook", {"test": True})
        assert result is True
        assert mock_send.call_count == 1
        mock_sleep.assert_not_called()

    @patch("scripts.monitor_uptime.time.sleep")
    @patch("src.utils.discord_notifier.send_discord_notification")
    def test_success_on_retry(self, mock_send: MagicMock, mock_sleep: MagicMock) -> None:
        """Échec puis succès au retry → True."""
        mock_send.side_effect = [False, True]
        result = monitor_uptime._send_discord("https://hook", {"test": True})
        assert result is True
        assert mock_send.call_count == 2
        mock_sleep.assert_called_once()

    @patch("scripts.monitor_uptime.time.sleep")
    @patch("src.utils.discord_notifier.send_discord_notification")
    def test_all_retries_fail(self, mock_send: MagicMock, mock_sleep: MagicMock) -> None:
        """Tous les essais échouent → False."""
        mock_send.return_value = False
        result = monitor_uptime._send_discord("https://hook", {"test": True})
        assert result is False
        assert mock_send.call_count == 2  # _DISCORD_MAX_RETRIES = 2


# =============================================================================
# Tests : constantes et cohérence
# =============================================================================


class TestConstants:
    """Tests de cohérence des constantes."""

    def test_default_offline_threshold(self) -> None:
        assert _DEFAULT_OFFLINE_THRESHOLD >= 1

    def test_all_lang_keys_consistent(self) -> None:
        """Toutes les langues ont les mêmes clés."""
        fr_keys = set(monitor_uptime._STRINGS["fr"].keys())
        en_keys = set(monitor_uptime._STRINGS["en"].keys())
        assert fr_keys == en_keys, f"Clés divergentes : {fr_keys ^ en_keys}"

    def test_webhook_prefix_valid(self) -> None:
        assert monitor_uptime._WEBHOOK_PREFIX.startswith("https://")
