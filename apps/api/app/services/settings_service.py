"""Service des paramètres utilisateur — Slice 1.

Fournit une interface sans Streamlit pour lire et persister AppSettings.
N'expose jamais discord_webhook_url en clair côté API.

Usage :
    from apps.api.app.services.settings_service import load_api_settings, update_api_settings
"""

from __future__ import annotations

import logging

from apps.api.app.schemas.settings import SettingsResponse, UpdateSettingsRequest

logger = logging.getLogger(__name__)


def load_api_settings() -> SettingsResponse:
    """Charge les paramètres depuis app_settings.json et les retourne comme SettingsResponse.

    Utilise le même ``load_settings()`` que l'UI Streamlit — source de vérité unique.
    ``discord_webhook_url`` est masqué ; seule sa présence est exposée.
    """
    from apps.api.app._pure_bridge import load_settings

    s = load_settings()
    return _to_response(s)


def update_api_settings(req: UpdateSettingsRequest) -> SettingsResponse:
    """Applique une mise à jour partielle et persiste sur disque.

    Seuls les champs non-None dans ``req`` sont mis à jour.

    Raises:
        ApiError: 500 si la persistance échoue.
    """
    from apps.api.app._pure_bridge import load_settings, save_settings
    from apps.api.app.core.errors import ApiError

    current = load_settings()

    # Construire le dict de mise à jour (uniquement les champs fournis)
    updates: dict = {}
    for field_name, value in req.model_dump(exclude_unset=True).items():
        if value is not None:
            updates[field_name] = value

    if not updates:
        return _to_response(current)

    updated = current.model_copy(update=updates)
    ok, err = save_settings(updated)
    if not ok:
        raise ApiError.internal(f"Impossible de persister les settings : {err}")

    return _to_response(updated)


def reset_media_index(reindex_after: bool = False) -> None:
    """Réinitialise l'index des médias.

    Délègue à l'opération SQL de nettoyage des tables media_*.
    Si ``reindex_after=True``, déclenche une réindexation immédiate.

    Note : cette fonction est appelée depuis le router qui crée un job
    asynchrone — elle ne doit pas bloquer.

    Raises:
        Exception: propagée vers le job qui la gère.
    """
    from src.utils.paths import PLAYERS_DIR

    if not PLAYERS_DIR.exists():
        return

    for player_dir in PLAYERS_DIR.iterdir():
        db = player_dir / "stats.duckdb"
        if not db.exists():
            continue
        try:
            from src.utils.db import duckdb_read_write

            with duckdb_read_write(db) as conn:
                # Vérifier si les tables existent
                for tbl in ("media_files", "media_match_associations"):
                    t = conn.execute(
                        "SELECT 1 FROM information_schema.tables WHERE table_name=? LIMIT 1",
                        (tbl,),
                    ).fetchone()
                    if t:
                        conn.execute(f"DELETE FROM {tbl}")
                        logger.info("media_reset: table %s vidée dans %s", tbl, db.name)
        except Exception as exc:
            logger.warning("media_reset: erreur sur %s — %s", db, exc)

    if reindex_after:
        _trigger_media_reindex()


def _trigger_media_reindex() -> None:
    """Déclenche une réindexation des médias après reset."""
    try:
        from src.data.media.indexer import MediaIndexer

        MediaIndexer().run_full_index()
    except Exception as exc:
        logger.warning("_trigger_media_reindex: erreur — %s", exc)


def _to_response(s: object) -> SettingsResponse:
    """Convertit un AppSettings en SettingsResponse (masque le webhook URL)."""
    webhook_present = bool(getattr(s, "discord_webhook_url", ""))

    return SettingsResponse(
        lang=getattr(s, "lang", "fr"),
        discord_lang=getattr(s, "discord_lang", "fr"),
        user_timezone=getattr(s, "user_timezone", "Europe/Paris"),
        normalize_mode_labels=getattr(s, "normalize_mode_labels", True),
        show_records=getattr(s, "show_records", False),
        refresh_clears_caches=getattr(s, "refresh_clears_caches", False),
        career_top_exclude_btb=getattr(s, "career_top_exclude_btb", False),
        media_captures_base_dir=getattr(s, "media_captures_base_dir", ""),
        media_tolerance_minutes=getattr(s, "media_tolerance_minutes", 3),
        media_watcher_enabled=getattr(s, "media_watcher_enabled", True),
        media_watcher_debounce_seconds=getattr(s, "media_watcher_debounce_seconds", 5),
        discord_notifications_enabled=getattr(s, "discord_notifications_enabled", False),
        discord_webhook_url_present=webhook_present,
        discord_notify_sync=getattr(s, "discord_notify_sync", True),
        discord_notify_backfill=getattr(s, "discord_notify_backfill", True),
        discord_notify_new_version=getattr(s, "discord_notify_new_version", True),
        discord_notify_new_media=getattr(s, "discord_notify_new_media", True),
        spnkr_refresh_with_backfill=getattr(s, "spnkr_refresh_with_backfill", False),
        spnkr_refresh_backfill_medals=getattr(s, "spnkr_refresh_backfill_medals", False),
        spnkr_refresh_backfill_skill=getattr(s, "spnkr_refresh_backfill_skill", False),
        spnkr_refresh_backfill_aliases=getattr(s, "spnkr_refresh_backfill_aliases", False),
        spnkr_refresh_backfill_personal_scores=getattr(
            s, "spnkr_refresh_backfill_personal_scores", False
        ),
        spnkr_refresh_backfill_performance_scores=getattr(
            s, "spnkr_refresh_backfill_performance_scores", True
        ),
        spnkr_refresh_backfill_lusr=getattr(s, "spnkr_refresh_backfill_lusr", True),
        spnkr_refresh_backfill_events=getattr(s, "spnkr_refresh_backfill_events", False),
        spnkr_refresh_backfill_weapons=getattr(s, "spnkr_refresh_backfill_weapons", False),
    )
