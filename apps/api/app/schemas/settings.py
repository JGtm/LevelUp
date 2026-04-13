"""Schémas Pydantic pour les endpoints Settings (Slice 1).

Expose uniquement les champs éditables depuis l'UI React —
les champs internes ou sensibles (discord_webhook_url, etc.) sont masqués.

``discord_webhook_url_present`` est en lecture seule (bool) :
le front sait si un webhook est configuré sans jamais voir l'URL.
"""

from __future__ import annotations

from pydantic import BaseModel, Field


class SettingsResponse(BaseModel):
    """Lecture des paramètres utilisateur persistés (GET /settings)."""

    # Internationalisation
    lang: str
    discord_lang: str

    # Affichage
    user_timezone: str
    normalize_mode_labels: bool
    show_records: bool
    refresh_clears_caches: bool

    # Carrière
    career_top_exclude_btb: bool

    # Médias
    media_captures_base_dir: str
    media_tolerance_minutes: int
    media_watcher_enabled: bool
    media_watcher_debounce_seconds: int

    # Discord — webhook_url masqué, on expose uniquement sa présence
    discord_notifications_enabled: bool
    discord_webhook_url_present: bool
    discord_notify_sync: bool
    discord_notify_backfill: bool
    discord_notify_new_version: bool
    discord_notify_new_media: bool

    # Backfill SPNKr
    spnkr_refresh_with_backfill: bool
    spnkr_refresh_backfill_medals: bool
    spnkr_refresh_backfill_skill: bool
    spnkr_refresh_backfill_aliases: bool
    spnkr_refresh_backfill_personal_scores: bool
    spnkr_refresh_backfill_performance_scores: bool
    spnkr_refresh_backfill_lusr: bool
    spnkr_refresh_backfill_events: bool
    spnkr_refresh_backfill_weapons: bool


class UpdateSettingsRequest(BaseModel):
    """Mise à jour partielle des paramètres (PATCH /settings).

    Tous les champs sont optionnels — seuls les champs fournis sont mis à jour.
    ``discord_webhook_url`` peut être mis à jour mais ne sera jamais renvoyé.
    """

    # Internationalisation
    lang: str | None = None
    discord_lang: str | None = None

    # Affichage
    user_timezone: str | None = None
    normalize_mode_labels: bool | None = None
    show_records: bool | None = None
    refresh_clears_caches: bool | None = None

    # Carrière
    career_top_exclude_btb: bool | None = None

    # Médias
    media_captures_base_dir: str | None = None
    media_tolerance_minutes: int | None = Field(default=None, ge=0)
    media_watcher_enabled: bool | None = None
    media_watcher_debounce_seconds: int | None = Field(default=None, ge=1, le=60)

    # Discord
    discord_notifications_enabled: bool | None = None
    discord_webhook_url: str | None = None  # Mise à jour possible, jamais retourné
    discord_notify_sync: bool | None = None
    discord_notify_backfill: bool | None = None
    discord_notify_new_version: bool | None = None
    discord_notify_new_media: bool | None = None

    # Backfill SPNKr
    spnkr_refresh_with_backfill: bool | None = None
    spnkr_refresh_backfill_medals: bool | None = None
    spnkr_refresh_backfill_skill: bool | None = None
    spnkr_refresh_backfill_aliases: bool | None = None
    spnkr_refresh_backfill_personal_scores: bool | None = None
    spnkr_refresh_backfill_performance_scores: bool | None = None
    spnkr_refresh_backfill_lusr: bool | None = None
    spnkr_refresh_backfill_events: bool | None = None
    spnkr_refresh_backfill_weapons: bool | None = None


class MediaResetRequest(BaseModel):
    """Corps de POST /settings/media/reset-index."""

    confirm_destructive: bool
    reindex_after_reset: bool = False
