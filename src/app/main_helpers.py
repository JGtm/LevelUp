"""Helpers pour le main() de streamlit_app.py.

Fonctions d'initialisation, de validation et de rendu du profil.
"""

from __future__ import annotations

import contextlib
import logging
import os
import time as _time
from typing import TYPE_CHECKING

logger = logging.getLogger(__name__)

import polars as pl
import streamlit as st

from src.analysis import mark_firefight
from src.app.data_loader import (
    default_identity_from_secrets,
    propagate_identity_env,
)
from src.ui import (
    display_name_from_xuid,
    ensure_spnkr_tokens,
    get_hero_html,
    get_profile_appearance,
)
from src.ui.cache import db_cache_key, load_df_optimized
from src.ui.i18n import t
from src.ui.player_assets import download_image_to_cache, ensure_local_image_path
from src.ui.sync import (
    is_spnkr_db_path,
    pick_latest_spnkr_db_if_any,
)

if TYPE_CHECKING:
    from src.ui import AppSettings


def propagate_identity_to_env() -> None:
    """Propage les defaults depuis secrets vers l'environnement."""
    try:
        xuid_or_gt, xuid_fallback, wp = default_identity_from_secrets()
        propagate_identity_env(xuid_or_gt, xuid_fallback, wp)
    except Exception:
        pass


def validate_and_fix_db_path(db_path: str, default_db: str) -> str:
    """Valide le chemin DB et applique un fallback si nécessaire.

    Returns:
        Chemin valide ou chaîne vide.
    """
    if db_path and not os.path.exists(db_path):
        return ""

    if db_path and os.path.exists(db_path):
        try:
            if os.path.getsize(db_path) <= 0:
                st.warning(t("hlp_db_empty_warning"))
                fallback = ""
                if is_spnkr_db_path(db_path):
                    fallback = pick_latest_spnkr_db_if_any()
                    if fallback and os.path.exists(fallback) and os.path.getsize(fallback) <= 0:
                        fallback = ""
                if not fallback:
                    fallback = str(default_db or "").strip()
                    if not (fallback and os.path.exists(fallback)):
                        fallback = ""
                if fallback and fallback != db_path:
                    st.info(t("hlp_db_fallback_info", path=fallback))
                    st.session_state["db_path"] = fallback
                    return fallback
                return ""
        except Exception:
            pass

    return db_path


def _load_spartan_id_from_db(db_path: str, xuid: str) -> str | None:
    """Charge le dernier spartan_id depuis career_progression via le repository.

    Retourne None si la colonne n'existe pas, la table est vide, ou toute erreur.
    """
    try:
        from src.ui._cache_core import get_cached_repository_st

        career = get_cached_repository_st(db_path, xuid).load_career_data()
        if career:
            val = career.get("spartan_id")
            return str(val).strip() or None if val is not None else None
    except Exception:
        logger.debug("_load_spartan_id_from_db: échec pour xuid=%s", xuid, exc_info=True)
    return None


def load_profile_api(
    xuid: str, settings: AppSettings, *, db_path: str | None = None
) -> tuple[object | None, str | None]:
    """Charge le profil depuis le cache ou l'API SPNKr.

    Tente toujours le cache disque d'abord (même si l'API est désactivée).
    N'appelle l'API que si le cache est invalide/absent ET l'API est activée.

    Returns:
        (api_appearance, error_message)
    """
    api_enabled = settings.profile_api_enabled
    api_refresh_h = settings.profile_api_auto_refresh_hours
    api_app = None
    api_err = None

    xu = str(xuid or "").strip()
    if not xu:
        return None, None

    # Résoudre le gamertag pour les tokens per-player (endpoints economy player-gated)
    gamertag: str | None = None
    if db_path and os.path.exists(db_path):
        with contextlib.suppress(Exception):
            gamertag = display_name_from_xuid(xu, db_path=db_path) or None

    # Toujours essayer le cache d'abord, même si l'API est désactivée
    if not api_enabled:
        # API désactivée : charger depuis le cache sans limite de fraîcheur
        with contextlib.suppress(Exception):
            api_app, api_err = get_profile_appearance(
                xuid=xu,
                enabled=False,
                refresh_hours=999_999,  # Cache "infini" — ne jamais expirer
            )
        return api_app, api_err

    # API activée : cache + appel API si nécessaire
    try:
        api_app, api_err = get_profile_appearance(
            xuid=xu,
            enabled=True,
            refresh_hours=api_refresh_h,
            gamertag=gamertag,
        )
    except Exception as e:
        api_app, api_err = None, str(e)
    if api_err:
        st.caption(t("app_profile_api_err", err=api_err))

    return api_app, api_err


def _needs_halo_auth(url: str) -> bool:
    """Vérifie si l'URL nécessite une authentification Halo."""
    u = str(url or "").strip().lower()
    if not u:
        return False
    return (
        ("/hi/images/file/" in u)
        or ("/hi/waypoint/file/images/" in u)
        or u.startswith("inventory/")
        or u.startswith("/inventory/")
        or ("gamecms-hacs.svc.halowaypoint.com/hi/images/file/" in u)
    )


def render_profile_hero(  # noqa: C901
    xuid: str,
    settings: AppSettings,
    api_app: object | None,
    db_path: str | None = None,
) -> None:
    """Rend le hero/header du profil joueur."""
    me_name = (
        display_name_from_xuid(xuid.strip(), db_path=db_path)
        if str(xuid or "").strip()
        else t("app_player_default")
    )

    dl_enabled = settings.profile_assets_download_enabled or settings.profile_api_enabled
    refresh_h = settings.profile_assets_auto_refresh_hours

    # Valeurs manuelles (prioritaires) / sinon auto depuis API
    banner_value = settings.profile_banner.strip()
    emblem_value = settings.profile_emblem.strip() or (
        getattr(api_app, "emblem_image_url", None) if api_app else ""
    )
    backdrop_value = settings.profile_backdrop.strip() or (
        getattr(api_app, "backdrop_image_url", None) if api_app else ""
    )
    nameplate_value = settings.profile_nameplate.strip() or (
        getattr(api_app, "nameplate_image_url", None) if api_app else ""
    )
    service_tag_value = settings.profile_service_tag.strip() or (
        getattr(api_app, "service_tag", None) if api_app else ""
    )
    rank_label_value = settings.profile_rank_label.strip() or (
        getattr(api_app, "rank_label", None) if api_app else ""
    )
    rank_subtitle_value = settings.profile_rank_subtitle.strip() or (
        getattr(api_app, "rank_subtitle", None) if api_app else ""
    )
    rank_icon_value = (getattr(api_app, "rank_image_url", None) if api_app else "") or ""
    adornment_value = (getattr(api_app, "adornment_image_url", None) if api_app else "") or ""

    # Fallback : adornment_path + rank_label issus de la dernière sync career rank (DB)
    # Utile quand l'API profile est désactivée ou le cache froid.
    _needs_db_career = (
        (not adornment_value or not rank_label_value) and db_path and str(xuid or "").strip()
    )
    if _needs_db_career:
        with contextlib.suppress(Exception):
            from src.ui._cache_core import get_cached_repository_st
            from src.ui.career_ranks import format_career_rank_label_fr
            from src.ui.career_ranks import get_rank_info as _get_meta_rank_info

            career = get_cached_repository_st(db_path, str(xuid).strip()).load_career_data()
            if career:
                if not adornment_value and career.get("adornment_path"):
                    adornment_value = str(career["adornment_path"]).strip()
                if not rank_label_value and career.get("rank"):
                    _meta = _get_meta_rank_info(int(career["rank"]))
                    if _meta:
                        rank_label_value = _meta.full_label_fr or ""
                    if not rank_label_value:
                        rank_label_value = format_career_rank_label_fr(
                            tier=str(career.get("rank_tier") or ""),
                            title=str(career.get("rank_name") or ""),
                            grade=None,
                        )

    # Tokens Halo si nécessaire (gamertag transmis pour les refresh tokens per-player)
    _gamertag_for_tokens = str(me_name or "").strip() or None
    if (
        dl_enabled
        and (not str(os.environ.get("SPNKR_CLEARANCE_TOKEN") or "").strip())
        and (
            _needs_halo_auth(backdrop_value)
            or _needs_halo_auth(rank_icon_value)
            or _needs_halo_auth(nameplate_value)
            or _needs_halo_auth(adornment_value)
        )
    ):
        ensure_spnkr_tokens(timeout_seconds=12, db_path=db_path, gamertag=_gamertag_for_tokens)

    # Résolution des chemins locaux
    banner_path = ensure_local_image_path(
        banner_value, prefix="banner", download_enabled=dl_enabled, auto_refresh_hours=refresh_h
    )
    emblem_path = ensure_local_image_path(
        emblem_value, prefix="emblem", download_enabled=dl_enabled, auto_refresh_hours=refresh_h
    )
    backdrop_path = ensure_local_image_path(
        backdrop_value, prefix="backdrop", download_enabled=dl_enabled, auto_refresh_hours=refresh_h
    )
    nameplate_path = ensure_local_image_path(
        nameplate_value,
        prefix="nameplate",
        download_enabled=dl_enabled,
        auto_refresh_hours=refresh_h,
    )
    rank_icon_path = ensure_local_image_path(
        rank_icon_value, prefix="rank", download_enabled=dl_enabled, auto_refresh_hours=refresh_h
    )
    adornment_path = ensure_local_image_path(
        adornment_value,
        prefix="adornment",
        download_enabled=dl_enabled,
        auto_refresh_hours=refresh_h,
    )

    # Diagnostics non bloquants
    def _warn_asset(prefix: str, url: str, path: str | None) -> None:
        if not dl_enabled:
            return
        u = str(url or "").strip()
        if not u or (not u.startswith("http://") and not u.startswith("https://")):
            return
        if path:
            return
        key = f"_warned_asset_{prefix}_{hash(u)}"
        if st.session_state.get(key):
            return
        st.session_state[key] = True
        ok, err, _out = download_image_to_cache(u, prefix=prefix, timeout_seconds=12)
        if not ok:
            st.caption(t("app_asset_error", prefix=prefix, err=err))

    _warn_asset("rank", rank_icon_value, rank_icon_path)

    # Spartan ID depuis la DB (sync player-gated, source de vérité)
    spartan_id_value: str | None = None
    if db_path and str(xuid or "").strip():
        with contextlib.suppress(Exception):
            spartan_id_value = _load_spartan_id_from_db(str(db_path), str(xuid).strip())

    st.markdown(
        get_hero_html(
            player_name=me_name,
            service_tag=str(service_tag_value or "").strip() or None,
            rank_label=str(rank_label_value or "").strip() or None,
            rank_subtitle=str(rank_subtitle_value or "").strip() or None,
            rank_icon_path=rank_icon_path,
            adornment_path=adornment_path,
            banner_path=banner_path,
            backdrop_path=backdrop_path,
            nameplate_path=nameplate_path,
            id_badge_text_color=settings.profile_id_badge_text_color.strip() or None,
            emblem_path=emblem_path,
            spartan_id=spartan_id_value,
        ),
        unsafe_allow_html=True,
    )


def _fix_untranslated_mode_ui(df: pl.DataFrame, lang: str) -> pl.DataFrame:
    """Traduit ``mode_ui`` via ``translate_pair_name`` pour les lignes sans ``pair_name_fr``.

    ``add_i18n_display_columns`` est un module pur (0 accès DB). Quand ``pair_name_fr``
    est NULL, il produit un ``mode_ui`` brut (ex. ``"Arena:CTF"``). Ce second pass
    utilise ``translate_pair_name`` (avec lookup DB metadata) pour ces cas.
    """
    needed_cols = {"mode_ui", "pair_name_fr", "pair_name"}
    if not needed_cols.issubset(df.columns):
        return df

    untranslated = df.filter(pl.col("pair_name_fr").is_null()).get_column("pair_name").unique()
    if untranslated.is_empty():
        return df

    from src.ui import translate_pair_name
    from src.utils.polars_compat import build_mapping

    tr_map = build_mapping(untranslated, lambda x: translate_pair_name(x, lang=lang))
    if not tr_map:
        return df

    return df.with_columns(
        pl.when(pl.col("pair_name_fr").is_null())
        .then(
            pl.col("pair_name")
            .cast(pl.Utf8)
            .replace_strict(tr_map, default=pl.col("mode_ui"), return_dtype=pl.Utf8)
        )
        .otherwise(pl.col("mode_ui"))
        .alias("mode_ui")
    )


def load_match_dataframe(
    db_path: str, xuid: str, cache_buster: int = 0
) -> tuple[pl.DataFrame, tuple[int, ...] | None]:
    """Charge le DataFrame des matchs.

    Args:
        db_path: Chemin vers la DB.
        xuid: XUID du joueur.
        cache_buster: Token pour forcer l'invalidation du cache après sync.

    Returns:
        (df Polars, db_key)
    """
    import polars as pl

    from src.ui.perf import perf_section

    df = pl.DataFrame()
    db_key = db_cache_key(db_path) if db_path else None

    from src.ui._cache_core import SharedDBUnavailableError
    from src.utils.log_config import log_duration

    _t0 = _time.perf_counter()
    if db_path and os.path.exists(db_path) and str(xuid or "").strip():
        try:
            with (
                perf_section("db/load_df_optimized"),
                log_duration("db/load_df_optimized", logger, threshold_ms=500),
            ):
                df = load_df_optimized(
                    db_path, xuid.strip(), db_key=db_key, cache_buster=cache_buster
                )
        except SharedDBUnavailableError as exc:
            # Erreur transitoire (DB verrouillée par MediaIndexer ou xuid non résolu).
            # On tente UN rerun automatique ; si ça échoue encore, on affiche l'erreur.
            retry_key = "_shared_db_retry"
            if not st.session_state.get(retry_key):
                st.session_state[retry_key] = True
                logger.info("SharedDBUnavailableError transitoire, rerun auto: %s", exc)
                st.rerun()
            else:
                st.session_state.pop(retry_key, None)
                st.error(
                    "⚠️ Base de données temporairement inaccessible "
                    "(verrouillage concurrent). Rechargez la page."
                )
            return pl.DataFrame(), db_key
        # Chargement réussi : effacer le flag de retry
        st.session_state.pop("_shared_db_retry", None)
        if df.is_empty():
            st.warning(t("app_no_match"))
    else:
        st.info(t("hlp_configure_db"))

    if not df.is_empty():
        with perf_section("analysis/mark_firefight"):
            df = mark_firefight(df)
        from src.app.i18n_columns import add_i18n_display_columns
        from src.ui.i18n import get_lang

        lang = get_lang()
        df = add_i18n_display_columns(df, lang=lang)
        df = _fix_untranslated_mode_ui(df, lang)

    _ms = (_time.perf_counter() - _t0) * 1000
    logger.info(
        "DataFrame chargé: %d matchs en %.0f ms (xuid=%s...)",
        len(df),
        _ms,
        str(xuid or "")[:8],
    )
    return df, db_key
