"""Page Paramètres (Settings)."""

from __future__ import annotations

from collections.abc import Callable
from pathlib import Path

import streamlit as st

from src.ui.i18n import t

from src.config import get_default_db_path
from src.ui import (
    AppSettings,
    directory_input,
    load_settings,
    save_settings,
)
from src.ui.sections import render_source_section


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

    with st.expander("Source", expanded=False):
        st.caption(
            "Gestion de la base de données et des caches. "
            "Cette section permet de vider les caches et rafraîchir la liste des joueurs disponibles. "
            "La sélection du joueur actif se fait via le sélecteur dans la sidebar principale."
        )
        default_db = get_default_db_path()
        render_source_section(
            default_db,
            get_local_dbs=get_local_dbs_fn,
            on_clear_caches=on_clear_caches_fn,
        )

    with st.expander("Synchronisation", expanded=False):
        st.caption(
            "Configuration de la synchronisation via scripts/sync.py. "
            "Le bouton 🔄 Actualiser dans la sidebar lance une sync complète "
            "(matchs, highlights, médailles, aliases, skill)."
        )

        st.info(
            "ℹ️ **Architecture v5** : La synchronisation récupère automatiquement TOUTES les données :\n"
            "- Nouveaux matchs (matchmaking uniquement)\n"
            "- Highlight events (clips)\n"
            "- Médailles\n"
            "- Stats skill/MMR\n"
            "- Personal score awards\n"
            "- Aliases XUID\n\n"
            "Ces options ne sont plus configurables - tout est récupéré à chaque sync."
        )

        max_matches = st.number_input(
            "Max matchs par sync",
            min_value=10,
            max_value=5000,
            value=int(getattr(settings, "spnkr_refresh_max_matches", 500) or 500),
            step=10,
            help="Nombre maximum de matchs récents à vérifier lors d'une sync",
        )

        rps = st.number_input(
            "Requêtes API / seconde",
            min_value=1,
            max_value=50,
            value=int(getattr(settings, "spnkr_refresh_rps", 3) or 3),
            step=1,
            help="Attention : des valeurs trop élevées peuvent entraîner un bannissement API. Recommandé : 3-10.",
        )

    with st.expander("Options du bouton Actualiser", expanded=False):
        st.caption(
            "Configurez ce que fait le bouton 🔄 Synchroniser dans la sidebar. "
            "Le backfill remplit les données manquantes pour les matchs existants."
        )

        # Option pour activer le backfill complet
        backfill_enabled = st.toggle(
            "Activer le backfill après synchronisation",
            value=bool(getattr(settings, "spnkr_refresh_with_backfill", False)),
            help="Remplit automatiquement les données manquantes après chaque sync",
        )

        st.markdown("**Données à backfill :**")
        backfill_all = st.checkbox(
            "Toutes les données",
            value=False,
            help="Cochez pour backfill toutes les données, ou choisissez individuellement ci-dessous",
            disabled=not backfill_enabled,
        )

        if backfill_all and backfill_enabled:
            backfill_medals = True
            backfill_events = True
            backfill_skill = True
            backfill_personal_scores = True
            backfill_performance_scores = True
            backfill_aliases = True
        else:
            col1, col2 = st.columns(2)
            with col1:
                backfill_medals = st.checkbox(
                    "Médailles",
                    value=bool(getattr(settings, "spnkr_refresh_backfill_medals", False)),
                    disabled=not backfill_enabled,
                )
                backfill_events = st.checkbox(
                    "Highlight events",
                    value=bool(getattr(settings, "spnkr_refresh_backfill_events", False)),
                    disabled=not backfill_enabled,
                )
                backfill_skill = st.checkbox(
                    "Stats skill/MMR",
                    value=bool(getattr(settings, "spnkr_refresh_backfill_skill", False)),
                    disabled=not backfill_enabled,
                )
            with col2:
                backfill_personal_scores = st.checkbox(
                    "Personal score awards",
                    value=bool(getattr(settings, "spnkr_refresh_backfill_personal_scores", False)),
                    disabled=not backfill_enabled,
                )
                backfill_performance_scores = st.checkbox(
                    "Scores de performance",
                    value=bool(
                        getattr(settings, "spnkr_refresh_backfill_performance_scores", True)
                    ),
                    help="Calcule les scores de performance manquants (peut être activé même sans backfill général)",
                )
                backfill_aliases = st.checkbox(
                    "Aliases XUID",
                    value=bool(getattr(settings, "spnkr_refresh_backfill_aliases", False)),
                    disabled=not backfill_enabled,
                )

    with st.expander("Médias", expanded=True):
        st.info(
            "ℹ️ **Architecture v5** : La section Médias est toujours active. "
            "Configurez le dossier de base et la tolérance temporelle."
        )
        media_captures_base_dir = directory_input(
            "Dossier de base des captures",
            value=str(getattr(settings, "media_captures_base_dir", "") or ""),
            key="settings_media_captures_base_dir",
            help=(
                "Racine des captures. Un sous-dossier par joueur, nommé comme le gamertag "
                "(ex: D:/Captures/PlayerA/, D:/Captures/PlayerB/). Images et vidéos dans le même dossier."
            ),
            placeholder="Ex: D:/Captures",
        )
        media_tolerance_minutes = st.slider(
            "Tolérance (minutes) autour du match",
            min_value=0,
            max_value=30,
            value=int(settings.media_tolerance_minutes or 0),
            step=1,
        )
        # Bouton reset index médias
        if st.button("Réinitialiser l'index médias", key="settings_reset_media_index"):
            from src.data.media_indexer import MediaIndexer

            db_path = st.session_state.get("db_path") or get_default_db_path()
            if db_path:
                try:
                    idx = MediaIndexer(Path(db_path))
                    idx.reset_media_tables()
                    st.success("Index médias réinitialisé (joueur courant).")
                except Exception as e:
                    st.error(t("error_loading", error=e))

    with st.expander("Expérience", expanded=True):
        refresh_clears_caches = st.toggle(
            "Le bouton Actualiser vide aussi les caches",
            value=bool(getattr(settings, "refresh_clears_caches", False)),
            help="Utile si la DB change en dehors de l'app (NAS / import externe).",
        )

    # Architecture v5 : DuckDB avec shared_matches (valeurs fixes, plus d'UI dédiée)
    repository_mode_val = "duckdb"
    enable_duckdb_val = True

    # Section "Fichiers (avancé)" masquée - valeurs conservées depuis settings
    aliases_path = str(getattr(settings, "aliases_path", "") or "").strip()
    profiles_path = str(getattr(settings, "profiles_path", "") or "").strip()

    # Profil joueur (bannière / rang)
    # Par défaut, on masque ces réglages et on garde les valeurs actuelles.
    profile_assets_download_enabled = bool(
        getattr(settings, "profile_assets_download_enabled", False)
    )
    profile_assets_auto_refresh_hours = int(
        getattr(settings, "profile_assets_auto_refresh_hours", 24) or 0
    )
    profile_api_enabled = bool(getattr(settings, "profile_api_enabled", False))
    profile_api_auto_refresh_hours = int(
        getattr(settings, "profile_api_auto_refresh_hours", 6) or 0
    )
    profile_banner = str(getattr(settings, "profile_banner", "") or "").strip()
    profile_emblem = str(getattr(settings, "profile_emblem", "") or "").strip()
    profile_backdrop = str(getattr(settings, "profile_backdrop", "") or "").strip()
    profile_nameplate = str(getattr(settings, "profile_nameplate", "") or "").strip()
    profile_service_tag = str(getattr(settings, "profile_service_tag", "") or "").strip()
    profile_id_badge_text_color = str(
        getattr(settings, "profile_id_badge_text_color", "") or ""
    ).strip()
    profile_rank_label = str(getattr(settings, "profile_rank_label", "") or "").strip()
    profile_rank_subtitle = str(getattr(settings, "profile_rank_subtitle", "") or "").strip()

    # Section "Profil joueur (avancé)" masquée - valeurs conservées depuis settings

    cols = st.columns(2)
    if cols[0].button("Enregistrer", width="stretch"):
        new_settings = AppSettings(
            media_enabled=True,  # Toujours activé en v5
            media_screens_dir="",  # Legacy - non utilisé en v5
            media_videos_dir="",  # Legacy - non utilisé en v5
            media_captures_base_dir=str(media_captures_base_dir or "").strip(),
            media_tolerance_minutes=int(media_tolerance_minutes),
            refresh_clears_caches=bool(refresh_clears_caches),
            # SPNKr - valeurs fixes pour v5 (tout est récupéré automatiquement)
            prefer_spnkr_db_if_available=True,
            spnkr_refresh_on_start=False,  # Désactivé par défaut (sync manuelle via bouton)
            spnkr_refresh_on_manual_refresh=True,
            spnkr_refresh_match_type="matchmaking",  # Toujours matchmaking
            spnkr_refresh_max_matches=int(max_matches),
            spnkr_refresh_rps=int(rps),
            spnkr_refresh_with_highlight_events=True,  # Toujours activé
            spnkr_refresh_with_backfill=bool(backfill_enabled),
            spnkr_refresh_backfill_medals=bool(backfill_medals),
            spnkr_refresh_backfill_events=bool(backfill_events),
            spnkr_refresh_backfill_skill=bool(backfill_skill),
            spnkr_refresh_backfill_personal_scores=bool(backfill_personal_scores),
            spnkr_refresh_backfill_performance_scores=bool(backfill_performance_scores),
            spnkr_refresh_backfill_aliases=bool(backfill_aliases),
            aliases_path=str(aliases_path or "").strip(),
            profiles_path=str(profiles_path or "").strip(),
            profile_assets_download_enabled=bool(profile_assets_download_enabled),
            profile_assets_auto_refresh_hours=int(profile_assets_auto_refresh_hours),
            profile_api_enabled=bool(profile_api_enabled),
            profile_api_auto_refresh_hours=int(profile_api_auto_refresh_hours),
            profile_banner=str(profile_banner or "").strip(),
            profile_emblem=str(profile_emblem or "").strip(),
            profile_backdrop=str(profile_backdrop or "").strip(),
            profile_nameplate=str(profile_nameplate or "").strip(),
            profile_service_tag=str(profile_service_tag or "").strip(),
            profile_id_badge_text_color=str(profile_id_badge_text_color or "").strip(),
            profile_rank_label=str(profile_rank_label or "").strip(),
            profile_rank_subtitle=str(profile_rank_subtitle or "").strip(),
            repository_mode=str(repository_mode_val),
            enable_duckdb_analytics=bool(enable_duckdb_val),
        )
        ok, err = save_settings(new_settings)
        if ok:
            st.success("Paramètres enregistrés.")
            st.session_state["app_settings"] = new_settings
            st.rerun()
        else:
            st.error(err)
        return new_settings

    if cols[1].button("Recharger depuis fichier", width="stretch"):
        reloaded = load_settings()
        st.session_state["app_settings"] = reloaded
        st.rerun()
        return reloaded

    return settings
