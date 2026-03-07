"""Migration : highlight_events id → nextval(séquence) (shared)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_highlight_events_autoincrement

register(
    Migration(
        name="add_highlight_events_autoincrement",
        target_db="shared",
        description="highlight_events : séquence auto-increment sur id",
        apply_schema=ensure_highlight_events_autoincrement,
    )
)
