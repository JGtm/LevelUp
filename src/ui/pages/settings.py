"""Page Paramètres (Settings) — V3.

Sections fixes (sans expanders) :
1. Langue & Région
2. Backfill
3. Affichage
4. Médias
5. Notifications Discord
"""

from __future__ import annotations

from pathlib import Path

import streamlit as st

from src.ui import AppSettings, directory_input, load_settings
from src.ui.components.browser_storage import (
    hints_visible,
    persist_browser_prefs,
)
from src.ui.i18n import t
from src.ui.settings import _write_settings, patch_settings
from src.ui.tz import CURATED_TZ_LIST


def _resolve_env_webhook() -> str:
    """Retourne l'URL webhook Discord si définie dans l'environnement."""
    import os

    return os.environ.get("DISCORD_WEBHOOK_URL", "").strip()


# ---------------------------------------------------------------------------
# Handlers génériques on_change (V3)
# ---------------------------------------------------------------------------


def _on_change_setting(field: str, widget_key: str) -> None:
    """Handler générique on_change pour les widgets settings purement disque/session."""
    new_val = st.session_state.get(widget_key)
    _, ok, err = patch_settings(field, new_val)
    if not ok:
        st.toast(f"⚠️ Sauvegarde échouée : {err}", icon="⚠️")


def _on_change_show_hints() -> None:
    """Handler dédié show_hints : session_state + browser storage."""
    new_val = bool(st.session_state.get("setting_show_hints", True))
    st.session_state["_hints_visible"] = new_val
    persist_browser_prefs(show_hints="1" if new_val else "0")


def _check_settings_consistency(settings: AppSettings) -> list[str]:
    """Retourne des avertissements de cohérence (non-bloquants)."""
    warnings: list[str] = []
    if (
        settings.discord_notifications_enabled
        and not settings.discord_webhook_url
        and not _resolve_env_webhook()
    ):
        warnings.append(t("warn_discord_no_webhook"))
    if settings.media_enabled and not settings.media_captures_base_dir:
        warnings.append(t("warn_media_no_dir"))
    return warnings


def render_settings_page(settings: AppSettings) -> AppSettings:
    """Rend la page Paramètres et retourne les settings (potentiellement modifiés)."""
    # Toujours partir de session_state — le paramètre peut être une version périmée
    settings = st.session_state.get("app_settings") or settings
    if not isinstance(settings, AppSettings):
        settings = load_settings()

    st.subheader(t("settings_title"))

    cols = st.columns(2)
    save_clicked = cols[0].button(t("btn_save"), width="stretch")
    reload_clicked = cols[1].button(t("btn_reload"), width="stretch")

    st.divider()

    _render_language_section(settings)
    _render_display_section(settings)
    _render_media_section(settings)
    _render_discord_section(settings)
    _render_backfill_section(settings)

    # Avertissements de cohérence (non-bloquants)
    current = st.session_state.get("app_settings") or settings
    for warning in _check_settings_consistency(current):
        st.warning(warning, icon="⚠️")

    if save_clicked:
        # Les on_change ont déjà persisté chaque champ individuellement.
        # Le bouton est une confirmation visuelle + double-write de sécurité.
        current = st.session_state.get("app_settings") or settings
        if isinstance(current, AppSettings):
            ok, err = _write_settings(current)
            if ok:
                st.success(t("settings_save_ok"))
                st.rerun()
            else:
                st.error(err)
        return current

    if reload_clicked:
        reloaded = load_settings()
        st.session_state["app_settings"] = reloaded
        st.rerun()
        return reloaded

    return settings


def _render_language_section(settings: AppSettings) -> None:
    """Rend la section Langue & Région."""
    st.subheader(t("set_language_section"))

    _lang_current = str(getattr(settings, "lang", "fr") or "fr")
    if _lang_current not in ("fr", "en"):
        _lang_current = "fr"
    st.selectbox(
        t("set_lang_label"),
        options=["fr", "en"],
        index=["fr", "en"].index(_lang_current),
        format_func=lambda x: "Français" if x == "fr" else "English",
        key="setting_lang",
        on_change=_on_change_setting,
        args=("lang", "setting_lang"),
    )

    _tz_current = str(getattr(settings, "user_timezone", "Europe/Paris") or "Europe/Paris")
    if _tz_current not in CURATED_TZ_LIST:
        _tz_current = "Europe/Paris"
    st.selectbox(
        t("set_timezone_label"),
        options=CURATED_TZ_LIST,
        index=CURATED_TZ_LIST.index(_tz_current),
        help=t("set_timezone_help"),
        key="setting_user_timezone",
        on_change=_on_change_setting,
        args=("user_timezone", "setting_user_timezone"),
    )

    st.divider()


def _render_backfill_checkboxes(settings: AppSettings, bf_on: bool) -> None:
    """Rend les cases à cocher des types de données backfill en 3 colonnes."""
    st.markdown(f"**{t('settings_backfill_data_label')}**")
    col1, col2, col3 = st.columns(3)
    with col1:
        st.checkbox(
            t("set_backfill_medals"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_medals", False)),
            disabled=not bf_on,
            key="setting_spnkr_refresh_backfill_medals",
            on_change=_on_change_setting,
            args=("spnkr_refresh_backfill_medals", "setting_spnkr_refresh_backfill_medals"),
        )
        st.checkbox(
            t("set_backfill_skill"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_skill", False)),
            disabled=not bf_on,
            key="setting_spnkr_refresh_backfill_skill",
            on_change=_on_change_setting,
            args=("spnkr_refresh_backfill_skill", "setting_spnkr_refresh_backfill_skill"),
        )
        st.checkbox(
            t("set_backfill_aliases"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_aliases", False)),
            disabled=not bf_on,
            key="setting_spnkr_refresh_backfill_aliases",
            on_change=_on_change_setting,
            args=("spnkr_refresh_backfill_aliases", "setting_spnkr_refresh_backfill_aliases"),
        )
    with col2:
        st.checkbox(
            t("set_backfill_personal_scores"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_personal_scores", False)),
            disabled=not bf_on,
            key="setting_spnkr_refresh_backfill_personal_scores",
            on_change=_on_change_setting,
            args=(
                "spnkr_refresh_backfill_personal_scores",
                "setting_spnkr_refresh_backfill_personal_scores",
            ),
        )
        st.checkbox(
            t("set_backfill_scores"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_performance_scores", True)),
            help=t("set_backfill_score_help"),
            key="setting_spnkr_refresh_backfill_performance_scores",
            on_change=_on_change_setting,
            args=(
                "spnkr_refresh_backfill_performance_scores",
                "setting_spnkr_refresh_backfill_performance_scores",
            ),
        )
        st.checkbox(
            t("set_backfill_lusr"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_lusr", True)),
            disabled=not bf_on,
            help=t("set_backfill_lusr_help"),
            key="setting_spnkr_refresh_backfill_lusr",
            on_change=_on_change_setting,
            args=("spnkr_refresh_backfill_lusr", "setting_spnkr_refresh_backfill_lusr"),
        )
    with col3:
        st.checkbox(
            t("set_backfill_events"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_events", False)),
            disabled=not bf_on,
            help=t("set_backfill_events_help"),
            key="setting_spnkr_refresh_backfill_events",
            on_change=_on_change_setting,
            args=("spnkr_refresh_backfill_events", "setting_spnkr_refresh_backfill_events"),
        )
        st.checkbox(
            t("set_backfill_weapons"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_weapons", False)),
            disabled=not bf_on,
            help=t("set_backfill_weapons_help"),
            key="setting_spnkr_refresh_backfill_weapons",
            on_change=_on_change_setting,
            args=("spnkr_refresh_backfill_weapons", "setting_spnkr_refresh_backfill_weapons"),
        )


def _render_backfill_section(settings: AppSettings) -> None:
    """Rend la section Backfill (données manquantes), collapsée par défaut."""
    st.subheader(t("set_refresh_options"))
    st.caption(t("settings_backfill_warning"))
    with st.expander(t("settings_backfill_expand_label"), expanded=False):
        st.toggle(
            t("settings_backfill_enable"),
            value=bool(getattr(settings, "spnkr_refresh_with_backfill", False)),
            key="setting_spnkr_refresh_with_backfill",
            on_change=_on_change_setting,
            args=("spnkr_refresh_with_backfill", "setting_spnkr_refresh_with_backfill"),
        )
        _bf_on = bool(
            st.session_state.get(
                "setting_spnkr_refresh_with_backfill",
                getattr(settings, "spnkr_refresh_with_backfill", False),
            )
        )
        _render_backfill_checkboxes(settings, _bf_on)


def _render_display_section(settings: AppSettings) -> None:
    """Rend la section Affichage."""
    st.subheader(t("set_display_section"))

    st.toggle(
        t("set_normalize_mode_labels"),
        value=bool(getattr(settings, "normalize_mode_labels", True)),
        help=t("set_normalize_mode_labels_help"),
        key="setting_normalize_mode_labels",
        on_change=_on_change_setting,
        args=("normalize_mode_labels", "setting_normalize_mode_labels"),
    )
    st.toggle(
        t("set_show_hints"),
        value=hints_visible(),
        help=t("set_show_hints_help"),
        key="setting_show_hints",
        on_change=_on_change_show_hints,
    )
    st.toggle(
        t("set_show_records"),
        value=bool(getattr(settings, "show_records", False)),
        help=t("set_show_records_help"),
        key="setting_show_records",
        on_change=_on_change_setting,
        args=("show_records", "setting_show_records"),
    )
    st.toggle(
        t("set_career_exclude_btb"),
        value=bool(getattr(settings, "career_top_exclude_btb", False)),
        help=t("set_career_exclude_btb_help"),
        key="setting_career_top_exclude_btb",
        on_change=_on_change_setting,
        args=("career_top_exclude_btb", "setting_career_top_exclude_btb"),
    )
    st.toggle(
        t("set_clear_cache_title"),
        value=bool(getattr(settings, "refresh_clears_caches", False)),
        help=t("set_clear_cache_help"),
        key="setting_refresh_clears_caches",
        on_change=_on_change_setting,
        args=("refresh_clears_caches", "setting_refresh_clears_caches"),
    )

    st.divider()


def _render_media_section(settings: AppSettings) -> None:
    """Rend la section Médias."""
    st.subheader(t("set_media_title"))

    directory_input(
        t("set_media_screenshots"),
        value=str(getattr(settings, "media_captures_base_dir", "") or ""),
        key="settings_media_captures_base_dir",
        help=t("set_media_root_help"),
        placeholder="Ex: D:/Captures",
        on_change=_on_change_setting,
        args=("media_captures_base_dir", "settings_media_captures_base_dir__text"),
    )
    st.slider(
        t("set_media_tolerance"),
        min_value=0,
        max_value=30,
        value=int(settings.media_tolerance_minutes or 0),
        step=1,
        key="setting_media_tolerance_minutes",
        on_change=_on_change_setting,
        args=("media_tolerance_minutes", "setting_media_tolerance_minutes"),
    )
    st.toggle(
        "Watcher automatique (Linux/inotify)",
        value=bool(getattr(settings, "media_watcher_enabled", True)),
        key="setting_media_watcher_enabled",
        help="Sur Linux, surveille le dossier captures en temps réel (inotify). "
        "Désactiver pour revenir au scan périodique.",
        on_change=_on_change_setting,
        args=("media_watcher_enabled", "setting_media_watcher_enabled"),
    )
    _watcher_on = bool(
        st.session_state.get(
            "setting_media_watcher_enabled", getattr(settings, "media_watcher_enabled", True)
        )
    )
    st.slider(
        "Délai avant indexation (secondes)",
        min_value=1,
        max_value=60,
        value=int(getattr(settings, "media_watcher_debounce_seconds", 5) or 5),
        step=1,
        disabled=not _watcher_on,
        help="Attend N secondes d'inactivité après le dernier fichier détecté avant d'indexer. "
        "Utile pour les copies de gros fichiers vidéo.",
        key="setting_media_watcher_debounce_seconds",
        on_change=_on_change_setting,
        args=("media_watcher_debounce_seconds", "setting_media_watcher_debounce_seconds"),
    )
    if st.button(t("ml_rescan"), key="settings_reset_media_index"):
        from src.config import get_default_db_path
        from src.data.media_indexer import MediaIndexer

        db_path = st.session_state.get("db_path") or get_default_db_path()
        if db_path:
            try:
                idx = MediaIndexer(Path(db_path))
                idx.reset_media_tables()
                st.success(t("settings_index_reset"))
            except Exception as e:
                st.error(t("error_loading", error=e))

    st.divider()


def _render_discord_section(settings: AppSettings) -> None:
    """Rend la section Notifications Discord."""
    st.subheader(t("set_discord_section"))

    st.toggle(
        t("set_discord_enable"),
        value=bool(getattr(settings, "discord_notifications_enabled", False)),
        key="setting_discord_enabled",
        on_change=_on_change_setting,
        args=("discord_notifications_enabled", "setting_discord_enabled"),
    )
    _discord_on = bool(
        st.session_state.get(
            "setting_discord_enabled", getattr(settings, "discord_notifications_enabled", False)
        )
    )
    # Détection webhook via .env.local (prioritaire, non affiché pour sécurité)
    _env_webhook = _resolve_env_webhook()
    st.text_input(
        t("set_discord_url"),
        value=str(getattr(settings, "discord_webhook_url", "") or ""),
        disabled=not _discord_on,
        placeholder="https://discord.com/api/webhooks/...",
        key="setting_discord_webhook_url",
        on_change=_on_change_setting,
        args=("discord_webhook_url", "setting_discord_webhook_url"),
    )
    if _env_webhook and not getattr(settings, "discord_webhook_url", ""):
        st.caption(t("set_discord_url_env_hint"))
    _discord_lang_current = str(getattr(settings, "discord_lang", "fr") or "fr")
    if _discord_lang_current not in ("fr", "en"):
        _discord_lang_current = "fr"
    st.selectbox(
        t("set_discord_lang_label"),
        options=["fr", "en"],
        index=["fr", "en"].index(_discord_lang_current),
        format_func=lambda x: "Français" if x == "fr" else "English",
        disabled=not _discord_on,
        key="setting_discord_lang",
        on_change=_on_change_setting,
        args=("discord_lang", "setting_discord_lang"),
    )

    st.markdown(f"**{t('set_discord_notify_types_label')}**")
    st.checkbox(
        t("set_discord_notify_sync"),
        value=bool(getattr(settings, "discord_notify_sync", True)),
        disabled=not _discord_on,
        key="setting_discord_notify_sync",
        on_change=_on_change_setting,
        args=("discord_notify_sync", "setting_discord_notify_sync"),
    )
    st.checkbox(
        t("set_discord_notify_backfill"),
        value=bool(getattr(settings, "discord_notify_backfill", True)),
        disabled=not _discord_on,
        key="setting_discord_notify_backfill",
        on_change=_on_change_setting,
        args=("discord_notify_backfill", "setting_discord_notify_backfill"),
    )
    st.checkbox(
        t("set_discord_notify_new_version"),
        value=bool(getattr(settings, "discord_notify_new_version", True)),
        disabled=not _discord_on,
        key="setting_discord_notify_new_version",
        help=t("set_discord_notify_new_version_help"),
        on_change=_on_change_setting,
        args=("discord_notify_new_version", "setting_discord_notify_new_version"),
    )

    st.divider()
