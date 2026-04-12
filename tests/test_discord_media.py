"""Tests unitaires pour src/utils/_discord_media.py.

Couvre :
- _MediaRow dataclass
- _build_embed_title_desc — FR/EN, comptage video/image
- _build_embed_fields — champs, overflow > 6
- _build_embed — structure complète du payload
- _pick_thumbnail — priorité video > image, fichier absent
- _get_webhook_url_for_media — setting désactivé, notif désactivée
- _query_unnotified — médias retournés, table absente (failsafe)
- _mark_notified — mise à jour DB, liste vide (no-op)
- _send_notification — branche JSON vs multipart
- _send_json — statuts 200/204/400, exception réseau
- _send_multipart — payload multipart, fallback si lecture fichier échoue
- notify_new_media — failsafe global, flux complet
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import duckdb
import pytest

from src.utils._discord_media import (
    _MAX_ATTACH_BYTES,
    _build_embed,
    _build_embed_fields,
    _build_embed_title_desc,
    _get_webhook_url_for_media,
    _mark_notified,
    _MediaRow,
    _pick_thumbnail,
    _query_unnotified,
    _send_json,
    _send_notification,
    notify_new_media,
)

_WEBHOOK = "https://discord.com/api/webhooks/123/token"


# =============================================================================
# Fixtures
# =============================================================================


def _row(  # noqa: PLR0913
    file_path: str = "/data/clip.mp4",
    file_name: str = "clip.mp4",
    kind: str = "video",
    thumbnail_path: str | None = None,
    match_id: str | None = "match-abc123",
    map_name: str | None = "Bazaar",
) -> _MediaRow:
    return _MediaRow(
        file_path=file_path,
        file_name=file_name,
        kind=kind,
        mtime=1700000000.0,
        thumbnail_path=thumbnail_path,
        match_id=match_id,
        map_name=map_name,
    )


def _make_db(tmp_path: Path) -> Path:
    """Crée une stats.duckdb minimale avec media_files + discord_notified_at."""
    db = tmp_path / "stats.duckdb"
    with duckdb.connect(str(db)) as conn:
        conn.execute(
            """
            CREATE TABLE media_files (
                file_path    VARCHAR PRIMARY KEY,
                file_name    VARCHAR,
                kind         VARCHAR,
                mtime        DOUBLE,
                status       VARCHAR,
                thumbnail_path VARCHAR,
                discord_notified_at TIMESTAMP
            )
            """
        )
        conn.execute(
            """
            CREATE TABLE media_match_associations (
                file_path VARCHAR,
                match_id  VARCHAR
            )
            """
        )
        conn.execute(
            """
            INSERT INTO media_files VALUES
            ('/data/clip.mp4',  'clip.mp4',  'video', 1700000000, 'active', NULL, NULL),
            ('/data/shot.png',  'shot.png',  'image', 1700000001, 'active', NULL, NULL),
            ('/data/old.png',   'old.png',   'image', 1699000000, 'active', NULL, '2026-01-01 00:00:00')
            """
        )
        conn.commit()
    return db


# =============================================================================
# _build_embed_title_desc
# =============================================================================


class TestBuildEmbedTitleDesc:
    def test_fr_videos_only(self) -> None:
        rows = [_row(kind="video"), _row(kind="video")]
        title, desc = _build_embed_title_desc(rows, "SpartanX", "fr")
        assert "SpartanX" in title
        assert "2 vidéos" in desc

    def test_fr_images_only(self) -> None:
        rows = [_row(kind="image")]
        _, desc = _build_embed_title_desc(rows, "SpartanX", "fr")
        assert "1 capture" in desc
        assert "vidéo" not in desc

    def test_fr_mixed(self) -> None:
        rows = [_row(kind="video"), _row(kind="image"), _row(kind="image")]
        _, desc = _build_embed_title_desc(rows, "SpartanX", "fr")
        assert "1 vidéo" in desc
        assert "2 captures" in desc

    def test_en_mixed(self) -> None:
        rows = [_row(kind="video"), _row(kind="image")]
        title, desc = _build_embed_title_desc(rows, "SpartanX", "en")
        assert "New media" in title
        assert "1 video" in desc
        assert "1 screenshot" in desc

    def test_en_plural_videos(self) -> None:
        rows = [_row(kind="video"), _row(kind="video")]
        _, desc = _build_embed_title_desc(rows, "GT", "en")
        assert "2 videos" in desc


# =============================================================================
# _build_embed_fields
# =============================================================================


class TestBuildEmbedFields:
    def test_basic_field(self) -> None:
        fields = _build_embed_fields([_row()], "fr")
        assert len(fields) == 1
        assert "clip.mp4" in fields[0]["name"]
        # match-abc123[:8] == "match-ab"
        assert "match-ab" in fields[0]["value"]
        assert "Bazaar" in fields[0]["value"]

    def test_no_match_id(self) -> None:
        fields = _build_embed_fields([_row(match_id=None, map_name=None)], "fr")
        assert fields[0]["value"] == "—"

    def test_overflow_shows_extra(self) -> None:
        rows = [_row(file_name=f"f{i}.mp4") for i in range(8)]
        fields = _build_embed_fields(rows, "fr")
        # 6 lignes + 1 overflow
        assert len(fields) == 7
        assert "2 autres" in fields[-1]["name"]

    def test_exactly_6_no_overflow(self) -> None:
        rows = [_row(file_name=f"f{i}.mp4") for i in range(6)]
        fields = _build_embed_fields(rows, "fr")
        assert len(fields) == 6

    def test_video_icon(self) -> None:
        fields = _build_embed_fields([_row(kind="video")], "fr")
        assert "🎬" in fields[0]["name"]

    def test_image_icon(self) -> None:
        fields = _build_embed_fields([_row(kind="image")], "fr")
        assert "🖼️" in fields[0]["name"]


# =============================================================================
# _build_embed
# =============================================================================


class TestBuildEmbed:
    def test_structure(self) -> None:
        # _get_discord_lang est importé lazy depuis _discord_embed
        with patch("src.utils._discord_embed._get_discord_lang", return_value="fr"):
            embed = _build_embed([_row()], "SpartanX")
        assert "title" in embed
        assert "description" in embed
        assert "color" in embed
        assert "fields" in embed
        assert "footer" in embed
        assert "timestamp" in embed
        assert embed["color"] == 0x5865F2

    def test_lang_fallback_on_error(self) -> None:
        with patch("src.utils._discord_embed._get_discord_lang", side_effect=RuntimeError("boom")):
            embed = _build_embed([_row()], "GT")
        assert "title" in embed  # ne lève pas


# =============================================================================
# _pick_thumbnail
# =============================================================================


class TestPickThumbnail:
    def test_prefers_video_over_image(self, tmp_path: Path) -> None:
        gif = tmp_path / "clip.gif"
        gif.write_bytes(b"GIF89a")
        img = tmp_path / "shot_thumb.png"
        img.write_bytes(b"PNG")
        rows = [
            _row(kind="image", thumbnail_path=str(img)),
            _row(kind="video", thumbnail_path=str(gif)),
        ]
        result = _pick_thumbnail(rows)
        assert result == gif

    def test_falls_back_to_image_if_no_video_thumb(self, tmp_path: Path) -> None:
        img = tmp_path / "shot.png"
        img.write_bytes(b"PNG")
        rows = [
            _row(kind="video", thumbnail_path=None),
            _row(kind="image", thumbnail_path=str(img)),
        ]
        assert _pick_thumbnail(rows) == img

    def test_absent_file_skipped(self, tmp_path: Path) -> None:
        rows = [_row(kind="video", thumbnail_path=str(tmp_path / "missing.gif"))]
        assert _pick_thumbnail(rows) is None

    def test_no_thumbnails_returns_none(self) -> None:
        rows = [_row(thumbnail_path=None), _row(kind="image", thumbnail_path=None)]
        assert _pick_thumbnail(rows) is None


# =============================================================================
# _get_webhook_url_for_media
# =============================================================================


class TestGetWebhookUrlForMedia:
    # _load_app_settings et _get_webhook_url sont importés depuis discord_notifier
    def test_disabled_global(self) -> None:
        with (
            patch(
                "src.utils.discord_notifier._load_app_settings",
                return_value={
                    "discord_notifications_enabled": False,
                    "discord_notify_new_media": True,
                },
            ),
            patch("src.utils.discord_notifier._get_webhook_url", return_value=None),
        ):
            assert _get_webhook_url_for_media() is None

    def test_disabled_media_setting(self) -> None:
        with (
            patch(
                "src.utils.discord_notifier._load_app_settings",
                return_value={
                    "discord_notifications_enabled": True,
                    "discord_notify_new_media": False,
                },
            ),
            patch("src.utils.discord_notifier._get_webhook_url", return_value=_WEBHOOK),
        ):
            assert _get_webhook_url_for_media() is None

    def test_enabled_returns_webhook(self) -> None:
        with (
            patch(
                "src.utils.discord_notifier._load_app_settings",
                return_value={
                    "discord_notifications_enabled": True,
                    "discord_notify_new_media": True,
                },
            ),
            patch("src.utils.discord_notifier._get_webhook_url", return_value=_WEBHOOK),
        ):
            assert _get_webhook_url_for_media() == _WEBHOOK

    def test_exception_returns_none(self) -> None:
        with patch(
            "src.utils.discord_notifier._load_app_settings", side_effect=RuntimeError("boom")
        ):
            assert _get_webhook_url_for_media() is None


# =============================================================================
# _query_unnotified
# =============================================================================


class TestQueryUnnotified:
    def test_returns_unnotified_only(self, tmp_path: Path) -> None:
        db = _make_db(tmp_path)
        rows = _query_unnotified(db)
        # clip.mp4 + shot.png ont discord_notified_at=NULL → retournés
        # old.png déjà notifié → exclu
        assert len(rows) == 2
        names = {r.file_name for r in rows}
        assert "clip.mp4" in names
        assert "shot.png" in names
        assert "old.png" not in names

    def test_empty_table(self, tmp_path: Path) -> None:
        db = tmp_path / "empty.duckdb"
        with duckdb.connect(str(db)) as conn:
            conn.execute(
                """
                CREATE TABLE media_files (
                    file_path VARCHAR, file_name VARCHAR, kind VARCHAR,
                    mtime DOUBLE, status VARCHAR, thumbnail_path VARCHAR,
                    discord_notified_at TIMESTAMP
                )
                """
            )
            conn.execute(
                "CREATE TABLE media_match_associations (file_path VARCHAR, match_id VARCHAR)"
            )
        assert _query_unnotified(db) == []

    def test_missing_table_returns_empty(self, tmp_path: Path) -> None:
        db = tmp_path / "notables.duckdb"
        with duckdb.connect(str(db)):
            pass  # DB vide, aucune table
        result = _query_unnotified(db)
        assert result == []

    def test_row_fields(self, tmp_path: Path) -> None:
        db = _make_db(tmp_path)
        rows = _query_unnotified(db)
        video = next(r for r in rows if r.kind == "video")
        assert video.file_path == "/data/clip.mp4"
        assert video.file_name == "clip.mp4"


# =============================================================================
# _mark_notified
# =============================================================================


class TestMarkNotified:
    def test_marks_rows(self, tmp_path: Path) -> None:
        db = _make_db(tmp_path)
        _mark_notified(db, ["/data/clip.mp4"])
        with duckdb.connect(str(db), read_only=True) as conn:
            row = conn.execute(
                "SELECT discord_notified_at FROM media_files WHERE file_path = '/data/clip.mp4'"
            ).fetchone()
        assert row is not None
        assert row[0] is not None

    def test_noop_on_empty_list(self, tmp_path: Path) -> None:
        db = _make_db(tmp_path)
        # Ne doit pas lever et ne rien modifier
        _mark_notified(db, [])
        with duckdb.connect(str(db), read_only=True) as conn:
            count = conn.execute(
                "SELECT COUNT(*) FROM media_files WHERE discord_notified_at IS NOT NULL"
            ).fetchone()[0]
        assert count == 1  # seulement old.png déjà notifié

    def test_db_error_does_not_raise(self, tmp_path: Path) -> None:
        # DB qui n'existe pas → doit logger et retourner sans exception
        _mark_notified(tmp_path / "ghost.duckdb", ["/data/x.mp4"])


# =============================================================================
# _send_json
# =============================================================================


class TestSendJson:
    def _mock_response(self, status: int) -> MagicMock:
        resp = MagicMock()
        resp.status = status
        resp.__enter__ = lambda s: s
        resp.__exit__ = MagicMock(return_value=False)
        return resp

    def test_200_ok(self) -> None:
        with patch("urllib.request.urlopen", return_value=self._mock_response(200)):
            _send_json({"embeds": []}, _WEBHOOK)  # ne lève pas

    def test_204_ok(self) -> None:
        with patch("urllib.request.urlopen", return_value=self._mock_response(204)):
            _send_json({"embeds": []}, _WEBHOOK)

    def test_400_warns(self, caplog: pytest.LogCaptureFixture) -> None:
        import logging

        with (
            patch("urllib.request.urlopen", return_value=self._mock_response(400)),
            caplog.at_level(logging.WARNING, logger="src.utils._discord_media"),
        ):
            _send_json({"embeds": []}, _WEBHOOK)
        assert any("400" in r.message for r in caplog.records)

    def test_network_error_does_not_raise(self) -> None:
        import urllib.error

        with patch("urllib.request.urlopen", side_effect=urllib.error.URLError("timeout")):
            _send_json({"embeds": []}, _WEBHOOK)  # failsafe


# =============================================================================
# _send_notification
# =============================================================================


class TestSendNotification:
    def test_uses_json_when_no_thumbnail(self) -> None:
        rows = [_row(thumbnail_path=None)]
        with (
            patch("src.utils._discord_media._send_json") as mock_json,
            patch("src.utils._discord_media._send_multipart") as mock_multi,
            patch("src.utils._discord_embed._get_discord_lang", return_value="fr"),
        ):
            _send_notification(rows, "GT", _WEBHOOK)
        mock_json.assert_called_once()
        mock_multi.assert_not_called()

    def test_uses_multipart_when_thumbnail_small(self, tmp_path: Path) -> None:
        thumb = tmp_path / "clip.gif"
        thumb.write_bytes(b"GIF89a" + b"\x00" * 100)
        rows = [_row(thumbnail_path=str(thumb))]
        with (
            patch("src.utils._discord_media._send_json") as mock_json,
            patch("src.utils._discord_media._send_multipart") as mock_multi,
            patch("src.utils._discord_embed._get_discord_lang", return_value="fr"),
        ):
            _send_notification(rows, "GT", _WEBHOOK)
        mock_multi.assert_called_once()
        mock_json.assert_not_called()

    def test_uses_json_when_thumbnail_too_large(self, tmp_path: Path) -> None:
        thumb = tmp_path / "huge.gif"
        thumb.write_bytes(b"\x00" * (_MAX_ATTACH_BYTES + 1))
        rows = [_row(thumbnail_path=str(thumb))]
        with (
            patch("src.utils._discord_media._send_json") as mock_json,
            patch("src.utils._discord_media._send_multipart") as mock_multi,
            patch("src.utils._discord_embed._get_discord_lang", return_value="fr"),
        ):
            _send_notification(rows, "GT", _WEBHOOK)
        mock_json.assert_called_once()
        mock_multi.assert_not_called()


# =============================================================================
# notify_new_media (flux complet + failsafe)
# =============================================================================


class TestNotifyNewMedia:
    def test_full_flow(self, tmp_path: Path) -> None:
        db = _make_db(tmp_path)
        with (
            patch("src.utils._discord_media._get_webhook_url_for_media", return_value=_WEBHOOK),
            patch("src.utils._discord_media._send_json") as mock_send,
            patch("src.utils._discord_embed._get_discord_lang", return_value="fr"),
        ):
            notify_new_media(db, "SpartanX")

        mock_send.assert_called_once()
        payload = mock_send.call_args[0][0]
        assert "embeds" in payload
        assert payload["embeds"][0]["color"] == 0x5865F2

        # Les médias doivent être marqués notifiés
        with duckdb.connect(str(db), read_only=True) as conn:
            count = conn.execute(
                "SELECT COUNT(*) FROM media_files "
                "WHERE discord_notified_at IS NOT NULL AND file_name != 'old.png'"
            ).fetchone()[0]
        assert count == 2

    def test_skips_when_no_webhook(self, tmp_path: Path) -> None:
        db = _make_db(tmp_path)
        with (
            patch("src.utils._discord_media._get_webhook_url_for_media", return_value=None),
            patch("src.utils._discord_media._send_json") as mock_send,
        ):
            notify_new_media(db, "GT")
        mock_send.assert_not_called()

    def test_skips_when_no_unnotified(self, tmp_path: Path) -> None:
        db = _make_db(tmp_path)
        # Marquer tous les médias comme déjà notifiés
        with duckdb.connect(str(db)) as conn:
            conn.execute("UPDATE media_files SET discord_notified_at = '2026-01-01 00:00:00'")
        with (
            patch("src.utils._discord_media._get_webhook_url_for_media", return_value=_WEBHOOK),
            patch("src.utils._discord_media._send_json") as mock_send,
        ):
            notify_new_media(db, "GT")
        mock_send.assert_not_called()

    def test_failsafe_on_exception(self, tmp_path: Path) -> None:
        # Une exception dans _do_notify ne doit jamais remonter
        with patch("src.utils._discord_media._do_notify", side_effect=RuntimeError("crash")):
            notify_new_media(tmp_path / "ghost.duckdb", "GT")  # ne lève pas

    def test_idempotent_second_call(self, tmp_path: Path) -> None:
        """Un second appel après le premier ne doit rien envoyer."""
        db = _make_db(tmp_path)
        with (
            patch("src.utils._discord_media._get_webhook_url_for_media", return_value=_WEBHOOK),
            patch("src.utils._discord_media._send_json"),
            patch("src.utils._discord_embed._get_discord_lang", return_value="fr"),
        ):
            notify_new_media(db, "GT")

        with (
            patch("src.utils._discord_media._get_webhook_url_for_media", return_value=_WEBHOOK),
            patch("src.utils._discord_media._send_json") as mock_second,
            patch("src.utils._discord_embed._get_discord_lang", return_value="fr"),
        ):
            notify_new_media(db, "GT")

        mock_second.assert_not_called()
