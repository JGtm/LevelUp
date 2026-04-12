"""Notifications Discord pour les nouveaux médias indexés.

Envoie un embed Discord avec thumbnail (GIF ou image) quand de nouveaux
fichiers sont indexés dans media_files.

Anti-spam : la colonne discord_notified_at dans media_files garantit
qu'un média n'est notifié qu'une seule fois.
"""

from __future__ import annotations

import io
import json
import logging
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

# Limite upload Discord webhook (8 Mo en pratique)
_MAX_ATTACH_BYTES = 8 * 1024 * 1024


@dataclass
class _MediaRow:
    """Données d'un média à notifier."""

    file_path: str
    file_name: str
    kind: str  # "video" | "image"
    mtime: float
    thumbnail_path: str | None
    match_id: str | None
    map_name: str | None


def notify_new_media(db_path: Path, gamertag: str) -> None:
    """Envoie une notification Discord pour les nouveaux médias non encore notifiés.

    Entièrement failsafe — ne lève jamais d'exception.

    La fonction :
    1. Vérifie que discord_notify_new_media est activé.
    2. Requête media_files pour les lignes avec discord_notified_at IS NULL.
    3. Construit un embed et l'envoie au webhook (avec thumbnail si dispo).
    4. Marque les médias envoyés avec l'horodatage courant.
    """
    try:
        _do_notify(db_path, gamertag)
    except Exception as exc:
        logger.warning("[Discord:media] Erreur inattendue pour %s : %s", gamertag, exc)


def _do_notify(db_path: Path, gamertag: str) -> None:
    """Corps principal de notify_new_media."""
    webhook_url = _get_webhook_url_for_media()
    if not webhook_url:
        return

    rows = _query_unnotified(db_path)
    if not rows:
        return

    logger.info("[Discord:media] %d nouveau(x) média(s) à notifier pour %s", len(rows), gamertag)

    _send_notification(rows, gamertag, webhook_url)
    _mark_notified(db_path, [r.file_path for r in rows])


def _get_webhook_url_for_media() -> str | None:
    """Retourne l'URL webhook si discord_notifications_enabled ET discord_notify_new_media."""
    try:
        from src.utils.discord_notifier import _get_webhook_url, _load_app_settings

        cfg = _load_app_settings()
        if not cfg.get("discord_notify_new_media", True):
            return None
        return _get_webhook_url()
    except Exception:
        return None


def _query_unnotified(db_path: Path) -> list[_MediaRow]:
    """Charge les médias non encore notifiés depuis media_files."""
    import duckdb

    try:
        with duckdb.connect(str(db_path), read_only=True) as conn:
            rows = conn.execute(
                """
                SELECT
                    mf.file_path,
                    mf.file_name,
                    mf.kind,
                    mf.mtime,
                    mf.thumbnail_path,
                    mma.match_id,
                    NULL AS map_name
                FROM media_files mf
                LEFT JOIN media_match_associations mma
                    ON mma.file_path = mf.file_path
                WHERE mf.status = 'active'
                  AND (mf.discord_notified_at IS NULL)
                ORDER BY mf.mtime DESC
                LIMIT 10
                """
            ).fetchall()
    except Exception as exc:
        logger.debug("[Discord:media] Lecture media_files échouée : %s", exc)
        return []

    return [
        _MediaRow(
            file_path=r[0],
            file_name=r[1],
            kind=r[2],
            mtime=r[3],
            thumbnail_path=r[4],
            match_id=r[5],
            map_name=r[6],
        )
        for r in rows
    ]


def _mark_notified(db_path: Path, file_paths: list[str]) -> None:
    """Marque les médias comme notifiés dans la DB."""
    if not file_paths:
        return
    import duckdb

    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S")
    try:
        with duckdb.connect(str(db_path), read_only=False) as conn:
            placeholders = ", ".join(["?" for _ in file_paths])
            conn.execute(
                f"UPDATE media_files SET discord_notified_at = ? "
                f"WHERE file_path IN ({placeholders})",
                [now, *file_paths],
            )
            conn.commit()
    except Exception as exc:
        logger.warning("[Discord:media] Impossible de marquer les médias notifiés : %s", exc)


def _send_notification(rows: list[_MediaRow], gamertag: str, webhook_url: str) -> None:
    """Construit l'embed et envoie la notification Discord."""
    embed = _build_embed(rows, gamertag)
    payload: dict[str, Any] = {"embeds": [embed]}

    # Tenter d'envoyer avec thumbnail en pièce jointe
    thumb_path = _pick_thumbnail(rows)
    if thumb_path and thumb_path.exists() and thumb_path.stat().st_size <= _MAX_ATTACH_BYTES:
        payload["attachments"] = [{"id": 0, "filename": thumb_path.name}]
        embed["image"] = {"url": f"attachment://{thumb_path.name}"}
        _send_multipart(payload, webhook_url, thumb_path)
    else:
        _send_json(payload, webhook_url)


def _pick_thumbnail(rows: list[_MediaRow]) -> Path | None:
    """Retourne le chemin du meilleur thumbnail disponible parmi les médias."""
    # Préférer un GIF (vidéo) puis une image
    for kind in ("video", "image"):
        for row in rows:
            if row.kind == kind and row.thumbnail_path:
                p = Path(row.thumbnail_path)
                if p.exists():
                    return p
    return None


def _build_embed_title_desc(rows: list[_MediaRow], gamertag: str, lang: str) -> tuple[str, str]:
    """Construit le titre et la description de l'embed selon la langue."""
    n_videos = sum(1 for r in rows if r.kind == "video")
    n_images = sum(1 for r in rows if r.kind == "image")

    if lang == "fr":
        title = f"📸 Nouveaux médias — {gamertag}"
        parts = []
        if n_videos:
            parts.append(f"{n_videos} vidéo{'s' if n_videos > 1 else ''}")
        if n_images:
            parts.append(f"{n_images} capture{'s' if n_images > 1 else ''}")
        description = "Nouvellement indexés : " + " · ".join(parts)
    else:
        title = f"📸 New media — {gamertag}"
        parts = []
        if n_videos:
            parts.append(f"{n_videos} video{'s' if n_videos > 1 else ''}")
        if n_images:
            parts.append(f"{n_images} screenshot{'s' if n_images > 1 else ''}")
        description = "Newly indexed: " + " · ".join(parts)

    return title, description


def _build_embed_fields(rows: list[_MediaRow], lang: str) -> list[dict[str, Any]]:
    """Construit les champs de l'embed pour chaque média (max 6 + overflow)."""
    fields: list[dict[str, Any]] = []
    for row in rows[:6]:
        icon = "🎬" if row.kind == "video" else "🖼️"
        name = f"{icon} {row.file_name}"[:256]
        value_parts = []
        if row.match_id:
            value_parts.append(f"`{row.match_id[:8]}…`")
        if row.map_name:
            value_parts.append(row.map_name)
        value = " · ".join(value_parts) if value_parts else "—"
        fields.append({"name": name, "value": value[:1024], "inline": True})

    if len(rows) > 6:
        extra = len(rows) - 6
        suffix = "s" if extra > 1 else ""
        fields.append({"name": f"+ {extra} autre{suffix}…", "value": "—", "inline": False})
    return fields


def _build_embed(rows: list[_MediaRow], gamertag: str) -> dict[str, Any]:
    """Construit le payload Rich Embed Discord pour les nouveaux médias."""
    try:
        from src.utils._discord_embed import _get_discord_lang

        lang = _get_discord_lang()
    except Exception:
        lang = "fr"

    title, description = _build_embed_title_desc(rows, gamertag, lang)
    fields = _build_embed_fields(rows, lang)

    return {
        "title": title,
        "description": description,
        "color": 0x5865F2,  # Bleu Discord "Blurple"
        "fields": fields,
        "footer": {"text": "LevelUp · Halo Infinite"},
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


def _send_json(payload: dict[str, Any], webhook_url: str) -> None:
    """Envoie le payload via JSON simple."""
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        webhook_url,
        data=data,
        headers={
            "Content-Type": "application/json; charset=utf-8",
            "User-Agent": "LevelUp-HaloBot/1.0",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            if resp.status in (200, 204):
                logger.info("[Discord:media] Notification envoyée")
            else:
                logger.warning("[Discord:media] Réponse inattendue : %s", resp.status)
    except Exception as exc:
        logger.warning("[Discord:media] Envoi JSON échoué : %s", exc)


def _send_multipart(payload: dict[str, Any], webhook_url: str, thumb_path: Path) -> None:
    """Envoie le payload avec le thumbnail en pièce jointe (multipart/form-data)."""
    boundary = b"----LevelUpMediaBoundary"

    body = io.BytesIO()

    # Part 1 — payload_json
    payload_bytes = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    body.write(b"--" + boundary + b"\r\n")
    body.write(b'Content-Disposition: form-data; name="payload_json"\r\n')
    body.write(b"Content-Type: application/json; charset=utf-8\r\n\r\n")
    body.write(payload_bytes)
    body.write(b"\r\n")

    # Part 2 — fichier thumbnail
    ext = thumb_path.suffix.lower()
    content_type_map = {
        ".gif": b"image/gif",
        ".png": b"image/png",
        ".jpg": b"image/jpeg",
        ".jpeg": b"image/jpeg",
        ".webp": b"image/webp",
    }
    ct = content_type_map.get(ext, b"application/octet-stream")
    filename_bytes = thumb_path.name.encode("utf-8")

    try:
        file_data = thumb_path.read_bytes()
    except OSError as exc:
        logger.warning("[Discord:media] Lecture thumbnail échouée : %s", exc)
        _send_json({k: v for k, v in payload.items() if k != "attachments"}, webhook_url)
        return

    body.write(b"--" + boundary + b"\r\n")
    body.write(
        b'Content-Disposition: form-data; name="files[0]"; filename="' + filename_bytes + b'"\r\n'
    )
    body.write(b"Content-Type: " + ct + b"\r\n\r\n")
    body.write(file_data)
    body.write(b"\r\n")

    body.write(b"--" + boundary + b"--\r\n")

    req = urllib.request.Request(
        webhook_url,
        data=body.getvalue(),
        headers={
            "Content-Type": f"multipart/form-data; boundary={boundary.decode()}",
            "User-Agent": "LevelUp-HaloBot/1.0",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            if resp.status in (200, 204):
                logger.info("[Discord:media] Notification + thumbnail envoyés")
            else:
                logger.warning("[Discord:media] Réponse inattendue : %s", resp.status)
    except Exception as exc:
        logger.warning(
            "[Discord:media] Envoi multipart échoué : %s — tentative sans thumbnail", exc
        )
        payload_fallback = {k: v for k, v in payload.items() if k not in ("attachments",)}
        if "embeds" in payload_fallback:
            for emb in payload_fallback["embeds"]:
                emb.pop("image", None)
        _send_json(payload_fallback, webhook_url)
