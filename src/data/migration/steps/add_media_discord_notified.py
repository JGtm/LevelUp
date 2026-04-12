"""Migration : colonne discord_notified_at sur media_files (player)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_media_discord_notified_column

register(
    Migration(
        name="add_media_discord_notified",
        target_db="player",
        description="Colonne discord_notified_at sur media_files pour anti-spam notifs Discord",
        apply_schema=ensure_media_discord_notified_column,
    )
)
