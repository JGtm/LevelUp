"""Page Paramètres (Settings) — V2.

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

from src.ui import AppSettings, directory_input, load_settings, save_settings
from src.ui.components.browser_storage import (
    hints_visible,
    persist_browser_prefs,
)
from src.ui.i18n import t
from src.ui.tz import CURATED_TZ_LIST


def _resolve_env_webhook() -> str:
    """Retourne l'URL webhook Discord si définie dans l'environnement."""
    import os

    return os.environ.get("DISCORD_WEBHOOK_URL", "").strip()


def _get_preserved_settings(settings: AppSettings) -> dict:
    """Champs non exposés dans l'UI — préservés tels quels depuis les settings actuels."""

    def _s(attr: str, default: str = "") -> str:
        return str(getattr(settings, attr, default) or default).strip()

    def _b(attr: str, default: bool = False) -> bool:
        return bool(getattr(settings, attr, default))

    def _i(attr: str, default: int = 0) -> int:
        return int(getattr(settings, attr, default) or 0)

    return {
        "aliases_path": _s("aliases_path"),
        "profiles_path": _s("profiles_path"),
        "profile_assets_download_enabled": _b("profile_assets_download_enabled"),
        "profile_assets_auto_refresh_hours": _i("profile_assets_auto_refresh_hours", 24),
        "profile_api_enabled": _b("profile_api_enabled"),
        "profile_api_auto_refresh_hours": _i("profile_api_auto_refresh_hours", 6),
        "profile_banner": _s("profile_banner"),
        "profile_emblem": _s("profile_emblem"),
        "profile_backdrop": _s("profile_backdrop"),
        "profile_nameplate": _s("profile_nameplate"),
        "profile_service_tag": _s("profile_service_tag"),
        "profile_id_badge_text_color": _s("profile_id_badge_text_color"),
        "profile_rank_label": _s("profile_rank_label"),
        "profile_rank_subtitle": _s("profile_rank_subtitle"),
        "cli_lang": _s("cli_lang", "fr"),
        "tailscale_funnel_enabled": _b("tailscale_funnel_enabled"),
        "doppler_enabled": _b("doppler_enabled"),
        "doppler_project": _s("doppler_project"),
        "doppler_config": _s("doppler_config"),
        # Auth
        "auth_method": _s("auth_method", "refresh_token"),
        # Médias — non exposés dans l'UI, préservés tels quels
        "media_indexing_interval_hours": _i("media_indexing_interval_hours", 4),
        "media_reindex_after_sync": _b("media_reindex_after_sync", True),
        # Sync — optimisé, non exposé, préservé tel quel
        "prefer_spnkr_db_if_available": _b("prefer_spnkr_db_if_available", True),
        "spnkr_refresh_on_start": _b("spnkr_refresh_on_start", True),
        "spnkr_refresh_on_manual_refresh": _b("spnkr_refresh_on_manual_refresh", True),
        "spnkr_refresh_match_type": _s("spnkr_refresh_match_type", "matchmaking"),
        "spnkr_refresh_max_matches": _i("spnkr_refresh_max_matches", 200),
        "spnkr_refresh_with_highlight_events": _b("spnkr_refresh_with_highlight_events", False),
        "enable_duckdb_analytics": _b("enable_duckdb_analytics", False),
    }


def _build_settings_from_ui(  # noqa: PLR0913
    *,
    settings: AppSettings,
    lang: str,
    user_timezone: str,
    backfill_enabled: bool,
    backfill_medals: bool,
    backfill_events: bool,
    backfill_skill: bool,
    backfill_personal_scores: bool,
    backfill_performance_scores: bool,
    backfill_aliases: bool,
    backfill_lusr: bool,
    backfill_weapons: bool,
    refresh_clears_caches: bool,
    normalize_mode_labels: bool,
    career_top_exclude_btb: bool,
    show_records: bool,
    media_captures_base_dir: str,
    media_tolerance_minutes: int,
    media_watcher_enabled: bool,
    media_watcher_debounce_seconds: int,
    discord_notifications_enabled: bool,
    discord_webhook_url: str,
    discord_lang: str,
) -> AppSettings:
    """Construit un AppSettings à partir des valeurs UI actuelles."""
    return AppSettings(
        media_enabled=True,
        media_screens_dir="",
        media_videos_dir="",
        media_captures_base_dir=media_captures_base_dir,
        media_tolerance_minutes=media_tolerance_minutes,
        media_watcher_enabled=media_watcher_enabled,
        media_watcher_debounce_seconds=media_watcher_debounce_seconds,
        refresh_clears_caches=refresh_clears_caches,
        repository_mode="duckdb",
        user_timezone=user_timezone,
        normalize_mode_labels=normalize_mode_labels,
        career_top_exclude_btb=career_top_exclude_btb,
        show_records=show_records,
        lang=lang,
        discord_notifications_enabled=discord_notifications_enabled,
        discord_webhook_url=discord_webhook_url,
        discord_lang=discord_lang,
        spnkr_refresh_with_backfill=backfill_enabled,
        spnkr_refresh_backfill_medals=backfill_medals,
        spnkr_refresh_backfill_events=backfill_events,
        spnkr_refresh_backfill_skill=backfill_skill,
        spnkr_refresh_backfill_personal_scores=backfill_personal_scores,
        spnkr_refresh_backfill_performance_scores=backfill_performance_scores,
        spnkr_refresh_backfill_aliases=backfill_aliases,
        spnkr_refresh_backfill_lusr=backfill_lusr,
        spnkr_refresh_backfill_weapons=backfill_weapons,
        **_get_preserved_settings(settings),
    )


def render_settings_page(settings: AppSettings) -> AppSettings:
    """Rend la page Paramètres et retourne les settings (potentiellement modifiés).

    Parameters
    ----------
    settings : AppSettings
        Paramètres actuels de l'application.

    Returns
    -------
    AppSettings
        Paramètres (modifiés ou non).
    """
    st.subheader(t("settings_title"))

    cols = st.columns(2)
    save_clicked = cols[0].button(t("btn_save"), width="stretch")
    reload_clicked = cols[1].button(t("btn_reload"), width="stretch")

    st.divider()

    lang, user_timezone = _render_language_section(settings)
    backfill_vals = _render_backfill_section(settings)
    refresh_clears_caches, normalize_mode_labels, career_top_exclude_btb, show_records = (
        _render_display_section(settings)
    )
    (
        media_captures_base_dir,
        media_tolerance_minutes,
        media_watcher_enabled,
        media_watcher_debounce_seconds,
    ) = _render_media_section(settings)
    discord_enabled, discord_url, discord_lang_val = _render_discord_section(settings)

    if save_clicked:
        new_settings = _build_settings_from_ui(
            settings=settings,
            lang=lang,
            user_timezone=str(user_timezone or "Europe/Paris"),
            refresh_clears_caches=refresh_clears_caches,
            normalize_mode_labels=normalize_mode_labels,
            career_top_exclude_btb=career_top_exclude_btb,
            show_records=show_records,
            media_captures_base_dir=str(media_captures_base_dir or ""),
            media_tolerance_minutes=int(media_tolerance_minutes),
            media_watcher_enabled=bool(media_watcher_enabled),
            media_watcher_debounce_seconds=int(media_watcher_debounce_seconds),
            discord_notifications_enabled=discord_enabled,
            discord_webhook_url=str(discord_url or ""),
            discord_lang=discord_lang_val,
            **backfill_vals,
        )
        ok, err = save_settings(new_settings)
        if ok:
            st.success(t("settings_save_ok"))
            st.session_state["app_settings"] = new_settings
            st.rerun()
        else:
            st.error(err)
        return new_settings

    if reload_clicked:
        reloaded = load_settings()
        st.session_state["app_settings"] = reloaded
        st.rerun()
        return reloaded

    return settings


def _render_language_section(settings: AppSettings) -> tuple[str, str]:
    """Rend la section Langue & Région."""
    st.subheader(t("set_language_section"))

    _lang_current = str(getattr(settings, "lang", "fr") or "fr")
    if _lang_current not in ("fr", "en"):
        _lang_current = "fr"
    lang = st.selectbox(
        t("set_lang_label"),
        options=["fr", "en"],
        index=["fr", "en"].index(_lang_current),
        format_func=lambda x: "Français" if x == "fr" else "English",
    )

    _tz_current = str(getattr(settings, "user_timezone", "Europe/Paris") or "Europe/Paris")
    if _tz_current not in CURATED_TZ_LIST:
        _tz_current = "Europe/Paris"
    user_timezone = st.selectbox(
        t("set_timezone_label"),
        options=CURATED_TZ_LIST,
        index=CURATED_TZ_LIST.index(_tz_current),
        help=t("set_timezone_help"),
    )

    st.divider()
    return str(lang), str(user_timezone)


def _render_backfill_section(settings: AppSettings) -> dict:
    """Rend la section Backfill (données manquantes)."""
    st.subheader(t("set_refresh_options"))
    st.caption(t("settings_backfill_caption"))

    backfill_enabled = st.toggle(
        t("settings_backfill_enable"),
        value=bool(getattr(settings, "spnkr_refresh_with_backfill", False)),
    )

    st.markdown(f"**{t('settings_backfill_data_label')}**")
    col1, col2, col3 = st.columns(3)
    with col1:
        backfill_medals = st.checkbox(
            t("set_backfill_medals"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_medals", False)),
            disabled=not backfill_enabled,
        )
        backfill_skill = st.checkbox(
            t("set_backfill_skill"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_skill", False)),
            disabled=not backfill_enabled,
        )
        backfill_aliases = st.checkbox(
            t("set_backfill_aliases"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_aliases", False)),
            disabled=not backfill_enabled,
        )
    with col2:
        backfill_personal_scores = st.checkbox(
            t("set_backfill_personal_scores"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_personal_scores", False)),
            disabled=not backfill_enabled,
        )
        backfill_performance_scores = st.checkbox(
            t("set_backfill_scores"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_performance_scores", True)),
            help=t("set_backfill_score_help"),
        )
        backfill_lusr = st.checkbox(
            t("set_backfill_lusr"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_lusr", True)),
            disabled=not backfill_enabled,
            help=t("set_backfill_lusr_help"),
        )
    with col3:
        backfill_events = st.checkbox(
            t("set_backfill_events"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_events", False)),
            disabled=not backfill_enabled,
            help=t("set_backfill_events_help"),
        )
        backfill_weapons = st.checkbox(
            t("set_backfill_weapons"),
            value=bool(getattr(settings, "spnkr_refresh_backfill_weapons", False)),
            disabled=not backfill_enabled,
            help=t("set_backfill_weapons_help"),
        )

    st.divider()
    return {
        "backfill_enabled": bool(backfill_enabled),
        "backfill_medals": bool(backfill_medals),
        "backfill_events": bool(backfill_events),
        "backfill_skill": bool(backfill_skill),
        "backfill_personal_scores": bool(backfill_personal_scores),
        "backfill_performance_scores": bool(backfill_performance_scores),
        "backfill_aliases": bool(backfill_aliases),
        "backfill_lusr": bool(backfill_lusr),
        "backfill_weapons": bool(backfill_weapons),
    }


def _auto_save_show_hints() -> None:
    """Persiste immédiatement show_hints dès le changement du toggle."""
    new_val = bool(st.session_state.get("setting_show_hints", True))
    st.session_state["_hints_visible"] = new_val
    persist_browser_prefs(show_hints="1" if new_val else "0")


def _auto_save_show_records() -> None:
    """Persiste immédiatement show_records dès le changement du toggle.

    Appelé via on_change : évite la perte de la valeur lors d'une navigation
    entre pages (st.navigation réinitialise les clés widget au retour).
    """
    current: AppSettings | None = st.session_state.get("app_settings")
    if current is None:
        return
    # Utilise la valeur courante des settings comme fallback (jamais True par défaut)
    new_val = bool(st.session_state.get("setting_show_records", current.show_records))
    if new_val == current.show_records:
        return  # Aucun changement réel, ne pas écrire inutilement
    updated = current.model_copy(update={"show_records": new_val})
    # Mettre à jour session_state AVANT la vérification du succès fichier.
    # Si l'écriture disque échoue (verrou Windows, permissions), le toggle
    # restera cohérent en session. Sans ça, un rechargement ou une reconnexion
    # WebSocket rechargerait la valeur périmée du disque.
    st.session_state["app_settings"] = updated
    ok, err = save_settings(updated)
    if not ok:
        st.toast(f"⚠️ Sauvegarde échouée : {err}", icon="⚠️")


def _render_display_section(settings: AppSettings) -> tuple[bool, bool, bool, bool]:
    """Rend la section Affichage."""
    st.subheader(t("set_display_section"))

    normalize_mode_labels = st.toggle(
        t("set_normalize_mode_labels"),
        value=bool(getattr(settings, "normalize_mode_labels", True)),
        help=t("set_normalize_mode_labels_help"),
        key="setting_normalize_mode_labels",
    )
    st.toggle(
        t("set_show_hints"),
        value=hints_visible(),
        help=t("set_show_hints_help"),
        key="setting_show_hints",
        on_change=_auto_save_show_hints,
    )
    show_records = st.toggle(
        t("set_show_records"),
        value=bool(getattr(settings, "show_records", True)),
        help=t("set_show_records_help"),
        key="setting_show_records",
        on_change=_auto_save_show_records,
    )
    career_top_exclude_btb = st.toggle(
        t("set_career_exclude_btb"),
        value=bool(getattr(settings, "career_top_exclude_btb", False)),
        help=t("set_career_exclude_btb_help"),
        key="setting_career_top_exclude_btb",
    )
    refresh_clears_caches = st.toggle(
        t("set_clear_cache_title"),
        value=bool(getattr(settings, "refresh_clears_caches", False)),
        help=t("set_clear_cache_help"),
        key="setting_refresh_clears_caches",
    )

    st.divider()
    return (
        bool(refresh_clears_caches),
        bool(normalize_mode_labels),
        bool(career_top_exclude_btb),
        bool(show_records),
    )


def _render_media_section(settings: AppSettings) -> tuple[str, int, bool, int]:
    """Rend la section Médias."""
    st.subheader(t("set_media_title"))

    media_captures_base_dir = directory_input(
        t("set_media_screenshots"),
        value=str(getattr(settings, "media_captures_base_dir", "") or ""),
        key="settings_media_captures_base_dir",
        help=t("set_media_root_help"),
        placeholder="Ex: D:/Captures",
    )
    media_tolerance_minutes = st.slider(
        t("set_media_tolerance"),
        min_value=0,
        max_value=30,
        value=int(settings.media_tolerance_minutes or 0),
        step=1,
    )
    media_watcher_enabled = st.toggle(
        "Watcher automatique (Linux/inotify)",
        value=bool(getattr(settings, "media_watcher_enabled", True)),
        key="setting_media_watcher_enabled",
        help="Sur Linux, surveille le dossier captures en temps réel (inotify). "
        "Désactiver pour revenir au scan périodique.",
    )
    media_watcher_debounce_seconds = st.slider(
        "Délai avant indexation (secondes)",
        min_value=1,
        max_value=60,
        value=int(getattr(settings, "media_watcher_debounce_seconds", 5) or 5),
        step=1,
        disabled=not media_watcher_enabled,
        help="Attend N secondes d'inactivité après le dernier fichier détecté avant d'indexer. "
        "Utile pour les copies de gros fichiers vidéo.",
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
    return (
        str(media_captures_base_dir or ""),
        int(media_tolerance_minutes),
        bool(media_watcher_enabled),
        int(media_watcher_debounce_seconds),
    )


def _render_discord_section(settings: AppSettings) -> tuple[bool, str, str]:
    """Rend la section Notifications Discord."""
    st.subheader(t("set_discord_section"))

    discord_enabled = st.toggle(
        t("set_discord_enable"),
        value=bool(getattr(settings, "discord_notifications_enabled", False)),
        key="setting_discord_enabled",
    )
    # Détection webhook via .env.local (prioritaire, non affiché pour sécurité)
    _env_webhook = _resolve_env_webhook()
    discord_url = st.text_input(
        t("set_discord_url"),
        value=str(getattr(settings, "discord_webhook_url", "") or ""),
        disabled=not discord_enabled,
        placeholder="https://discord.com/api/webhooks/...",
        key="setting_discord_webhook_url",
    )
    if _env_webhook and not discord_url:
        st.caption(t("set_discord_url_env_hint"))
    _discord_lang_current = str(getattr(settings, "discord_lang", "fr") or "fr")
    if _discord_lang_current not in ("fr", "en"):
        _discord_lang_current = "fr"
    discord_lang_val = st.selectbox(
        t("set_discord_lang_label"),
        options=["fr", "en"],
        index=["fr", "en"].index(_discord_lang_current),
        format_func=lambda x: "Français" if x == "fr" else "English",
        disabled=not discord_enabled,
    )

    return bool(discord_enabled), str(discord_url or ""), str(discord_lang_val)
