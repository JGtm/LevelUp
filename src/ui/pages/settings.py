"""Page Paramètres (Settings)."""

from __future__ import annotations

from collections.abc import Callable
from pathlib import Path

import streamlit as st

from src.config import get_default_db_path
from src.ui import (
    AppSettings,
    directory_input,
    load_settings,
    save_settings,
)
from src.ui.i18n import t
from src.ui.sections import render_source_section
from src.ui.tz import CURATED_TZ_LIST  # liste curated des timezones disponibles

_ALL_BACKFILL_FLAGS: dict = {
    "backfill_enabled": True,
    "backfill_medals": True,
    "backfill_events": True,
    "backfill_skill": True,
    "backfill_personal_scores": True,
    "backfill_performance_scores": True,
    "backfill_aliases": True,
    "backfill_lusr": True,
    "backfill_weapons": False,
}


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
        "lang": _s("lang", "fr"),
        "discord_lang": _s("discord_lang", "fr"),
        "cli_lang": _s("cli_lang", "fr"),
        "discord_notifications_enabled": _b("discord_notifications_enabled"),
        "discord_webhook_url": _s("discord_webhook_url"),
        "tailscale_funnel_enabled": _b("tailscale_funnel_enabled"),
        "doppler_enabled": _b("doppler_enabled"),
        "doppler_project": _s("doppler_project"),
        "doppler_config": _s("doppler_config"),
    }


def _build_settings_from_ui(  # noqa: PLR0913
    *,
    settings: AppSettings,
    media_captures_base_dir: str,
    media_tolerance_minutes: int,
    refresh_clears_caches: bool,
    max_matches: int,
    rps: int,
    backfill_enabled: bool,
    backfill_medals: bool,
    backfill_events: bool,
    backfill_skill: bool,
    backfill_personal_scores: bool,
    backfill_performance_scores: bool,
    backfill_aliases: bool,
    backfill_lusr: bool,
    backfill_weapons: bool,
    user_timezone: str = "Europe/Paris",
) -> AppSettings:
    """Construit un AppSettings à partir des valeurs UI actuelles."""
    return AppSettings(
        media_enabled=True,
        media_screens_dir="",
        media_videos_dir="",
        media_captures_base_dir=str(media_captures_base_dir or "").strip(),
        media_tolerance_minutes=int(media_tolerance_minutes),
        refresh_clears_caches=bool(refresh_clears_caches),
        prefer_spnkr_db_if_available=True,
        spnkr_refresh_on_start=False,
        spnkr_refresh_on_manual_refresh=True,
        spnkr_refresh_match_type="matchmaking",
        spnkr_refresh_max_matches=int(max_matches),
        spnkr_refresh_rps=int(rps),
        spnkr_refresh_with_highlight_events=True,
        spnkr_refresh_with_backfill=bool(backfill_enabled),
        spnkr_refresh_backfill_medals=bool(backfill_medals),
        spnkr_refresh_backfill_events=bool(backfill_events),
        spnkr_refresh_backfill_skill=bool(backfill_skill),
        spnkr_refresh_backfill_personal_scores=bool(backfill_personal_scores),
        spnkr_refresh_backfill_performance_scores=bool(backfill_performance_scores),
        spnkr_refresh_backfill_aliases=bool(backfill_aliases),
        spnkr_refresh_backfill_lusr=bool(backfill_lusr),
        spnkr_refresh_backfill_weapons=bool(backfill_weapons),
        repository_mode="duckdb",
        enable_duckdb_analytics=True,
        user_timezone=str(user_timezone or "Europe/Paris").strip(),
        **_get_preserved_settings(settings),
    )


def render_settings_page(
    settings: AppSettings,
    *,
    get_local_dbs_fn: Callable[[], list[str]],
    on_clear_caches_fn: Callable[[], None],
) -> AppSettings:
    """Rend l'onglet Paramètres et retourne les settings (potentiellement modifiés).

    Parameters
    ----------
    settings : AppSettings
        Paramètres actuels de l'application.
    get_local_dbs_fn : Callable[[], list[str]]
        Fonction pour lister les bases de données locales.
    on_clear_caches_fn : Callable[[], None]
        Fonction pour vider les caches de l'application.

    Returns
    -------
    AppSettings
        Paramètres (modifiés ou non).
    """
    st.subheader(t("settings_title"))

    with st.expander(t("xbox_connect_section_title"), expanded=True):
        from src.ui.xbox_oauth_ui import render_xbox_login_section

        render_xbox_login_section()

    with st.expander("Source", expanded=False):
        st.caption(t("set_db_mgmt"))
        default_db = get_default_db_path()
        render_source_section(
            default_db,
            get_local_dbs=get_local_dbs_fn,
            on_clear_caches=on_clear_caches_fn,
        )

    max_matches, rps = _render_sync_section(settings)
    backfill_vals = _render_backfill_section(settings)
    media_captures_base_dir, media_tolerance_minutes = _render_media_section(settings)
    refresh_clears_caches, user_timezone = _render_experience_section(settings)

    cols = st.columns(2)
    if cols[0].button(t("btn_save"), width="stretch"):
        new_settings = _build_settings_from_ui(
            settings=settings,
            media_captures_base_dir=str(media_captures_base_dir or ""),
            media_tolerance_minutes=int(media_tolerance_minutes),
            refresh_clears_caches=bool(refresh_clears_caches),
            max_matches=int(max_matches),
            rps=int(rps),
            **backfill_vals,
            user_timezone=str(user_timezone or "Europe/Paris"),
        )
        ok, err = save_settings(new_settings)
        if ok:
            st.success(t("settings_save_ok"))
            st.session_state["app_settings"] = new_settings
            st.rerun()
        else:
            st.error(err)
        return new_settings

    if cols[1].button(t("btn_reload"), width="stretch"):
        reloaded = load_settings()
        st.session_state["app_settings"] = reloaded
        st.rerun()
        return reloaded

    return settings


def _render_sync_section(settings: AppSettings) -> tuple[int, int]:
    """Rend le bloc 'Synchronisation' et retourne (max_matches, rps)."""
    with st.expander(t("set_sync_title"), expanded=False):
        st.caption(t("settings_sync_help"))
        st.info(t("set_arch_v5_info"))
        max_matches = st.number_input(
            t("set_sync_max_matches"),
            min_value=10,
            max_value=5000,
            value=int(getattr(settings, "spnkr_refresh_max_matches", 500) or 500),
            step=10,
        )
        rps = st.number_input(
            t("set_sync_rate"),
            min_value=1,
            max_value=50,
            value=int(getattr(settings, "spnkr_refresh_rps", 3) or 3),
            step=1,
        )
    return int(max_matches), int(rps)


def _render_backfill_section(settings: AppSettings) -> dict:  # noqa: PLR0912
    """Rend le bloc 'Options de rafraîchissement' et retourne les flags backfill."""
    with st.expander(t("set_refresh_options"), expanded=False):
        st.caption(t("settings_backfill_caption"))
        backfill_enabled = st.toggle(
            t("settings_backfill_enable"),
            value=bool(getattr(settings, "spnkr_refresh_with_backfill", False)),
        )
        st.markdown(f"**{t('settings_backfill_data_label')}**")
        backfill_all = st.checkbox(
            t("backfill_all_data"),
            value=False,
            help=t("settings_backfill_all_help"),
            disabled=not backfill_enabled,
        )
        if backfill_all and backfill_enabled:
            return _ALL_BACKFILL_FLAGS
        col1, col2 = st.columns(2)
        with col1:
            backfill_medals = st.checkbox(
                t("set_backfill_medals"),
                value=bool(getattr(settings, "spnkr_refresh_backfill_medals", False)),
                disabled=not backfill_enabled,
            )
            backfill_events = st.checkbox(
                t("set_backfill_events"),
                value=bool(getattr(settings, "spnkr_refresh_backfill_events", False)),
                disabled=not backfill_enabled,
            )
            backfill_skill = st.checkbox(
                t("set_backfill_skill"),
                value=bool(getattr(settings, "spnkr_refresh_backfill_skill", False)),
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
            backfill_aliases = st.checkbox(
                t("set_backfill_aliases"),
                value=bool(getattr(settings, "spnkr_refresh_backfill_aliases", False)),
                disabled=not backfill_enabled,
            )
            backfill_lusr = st.checkbox(
                t("set_backfill_lusr"),
                value=bool(getattr(settings, "spnkr_refresh_backfill_lusr", True)),
                disabled=not backfill_enabled,
                help=t("set_backfill_lusr_help"),
            )
            backfill_weapons = st.checkbox(
                t("set_backfill_weapons"),
                value=bool(getattr(settings, "spnkr_refresh_backfill_weapons", False)),
                disabled=not backfill_enabled,
                help=t("set_backfill_weapons_help"),
            )
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


def _render_media_section(settings: AppSettings) -> tuple[str, int]:
    """Rend le bloc 'Médias' et retourne (media_captures_base_dir, media_tolerance_minutes)."""
    with st.expander(t("set_media_title"), expanded=True):
        st.info(t("set_media_arch_info"))
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
        if st.button(t("ml_rescan"), key="settings_reset_media_index"):
            from src.data.media_indexer import MediaIndexer

            db_path = st.session_state.get("db_path") or get_default_db_path()
            if db_path:
                try:
                    idx = MediaIndexer(Path(db_path))
                    idx.reset_media_tables()
                    st.success(t("settings_index_reset"))
                except Exception as e:
                    st.error(t("error_loading", error=e))
    return str(media_captures_base_dir or ""), int(media_tolerance_minutes)


def _render_experience_section(settings: AppSettings) -> tuple[bool, str]:
    """Rend le bloc 'Expérience' et retourne (refresh_clears_caches, user_timezone)."""
    with st.expander(t("set_experience_title"), expanded=True):
        refresh_clears_caches = st.toggle(
            t("set_clear_cache_title"),
            value=bool(getattr(settings, "refresh_clears_caches", False)),
            help="Utile si la DB change en dehors de l'app (NAS / import externe).",
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
    return bool(refresh_clears_caches), str(user_timezone or "Europe/Paris")
