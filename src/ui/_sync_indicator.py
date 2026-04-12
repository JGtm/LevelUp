"""Indicateur visuel de dernière synchronisation.

Affiche un badge coloré dans la sidebar Streamlit avec le temps écoulé
depuis la dernière sync.
"""

from __future__ import annotations

import html
import os
from datetime import datetime, timezone

import streamlit as st

from src.ui._sync_utils import _get_sync_metadata_smart


def render_sync_indicator(db_path: str, xuid: str | None = None, *, dot_only: bool = False) -> None:
    """Affiche l'indicateur de dernière synchronisation dans la sidebar.

    Couleurs:
    - 🟢 Vert: sync < 1h
    - 🟡 Jaune: sync < 24h
    - 🔴 Rouge: sync > 24h ou jamais

    Args:
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur (permet d'utiliser le repo caché).
        dot_only: Si True, affiche uniquement l'emoji indicateur sans texte.
    """
    if not db_path or not os.path.exists(db_path):
        return

    meta = _get_sync_metadata_smart(db_path, xuid=xuid)
    last_sync_raw = meta.get("last_sync_at")
    now = datetime.now(timezone.utc)
    sync_text = ""
    sync_status = "stale"

    last_sync: datetime | None = None
    if isinstance(last_sync_raw, datetime):
        last_sync = (
            last_sync_raw if last_sync_raw.tzinfo else last_sync_raw.replace(tzinfo=timezone.utc)
        )
    elif isinstance(last_sync_raw, str):
        try:
            last_sync = datetime.fromisoformat(last_sync_raw)
            if last_sync.tzinfo is None:
                last_sync = last_sync.replace(tzinfo=timezone.utc)
        except ValueError:
            last_sync = None

    if last_sync:
        sync_text, sync_status = _format_sync_time(now, last_sync, prefix="Sync")
    else:
        # Pas de métadonnées de sync, on utilise la date de modification du fichier
        try:
            mtime = os.path.getmtime(db_path)
            mtime_dt = datetime.fromtimestamp(mtime, tz=timezone.utc)
            sync_text, sync_status = _format_sync_time(now, mtime_dt, prefix="Modifié")
        except Exception:
            return

    if sync_text:
        if dot_only:
            safe_title = html.escape(sync_text, quote=True)
            st.markdown(
                (
                    "<div "
                    f"class='v7-l1-sync-dot v7-l1-sync-dot--{sync_status}' "
                    f"title='{safe_title}' aria-label='{safe_title}'></div>"
                ),
                unsafe_allow_html=True,
            )
        else:
            st.markdown(
                f"<div style='font-size: 0.85em; color: #888; margin: 4px 0 8px 0;'>{sync_text}</div>",
                unsafe_allow_html=True,
            )


def _format_sync_time(now: datetime, ref_time: datetime, *, prefix: str) -> tuple[str, str]:
    """Formate le temps écoulé avec indicateur coloré.

    Args:
        now: Datetime actuelle (UTC).
        ref_time: Datetime de référence (sync ou mtime).
        prefix: Préfixe du texte (ex: 'Sync', 'Modifié').

    Returns:
        Tuple (texte formaté, statut CSS).
    """
    delta = now - ref_time
    hours = delta.total_seconds() / 3600

    if hours < 1:
        minutes = int(delta.total_seconds() / 60)
        indicator = "🟢"
        status = "fresh"
        time_str = f"il y a {minutes} min" if minutes > 0 else "à l'instant"
    elif hours < 24:
        indicator = "🟡"
        status = "recent"
        h = int(hours)
        time_str = f"il y a {h}h"
    else:
        indicator = "🔴"
        status = "stale"
        days = int(hours / 24)
        time_str = f"il y a {days} jour{'s' if days > 1 else ''}"

    return f"{indicator} {prefix} {time_str}", status
