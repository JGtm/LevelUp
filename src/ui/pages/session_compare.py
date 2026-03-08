"""Page de comparaison de sessions."""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.config import CORE_STAT_COLUMNS
from src.ui.chart_utils import safe_chart_render
from src.ui.components.performance import (
    compute_session_performance_score_v2_ui,
    render_metric_comparison_row,
    render_performance_score_card,
)
from src.ui.i18n import t
from src.ui.pages._session_compare_viz import (
    render_kd_progression,
    render_outcomes_distribution,
    render_session_temporal_header,
)
from src.ui.pages.session_compare_charts import (
    SESSION_COLORS,  # noqa: F401 — re-exported
    render_comparison_bar_chart,
    render_comparison_radar_chart,
    render_participation_trend_section,
    render_session_history_table,
)
from src.ui.pages.session_compare_logic import (
    compute_historical_context,
    compute_similar_sessions_average,  # noqa: F401 — re-exported
    format_seconds_to_mmss,
    get_session_friends_signature,  # noqa: F401 — re-exported
    infer_session_dominant_category,
    is_session_with_friends,  # noqa: F401 — re-exported
    outcome_class,  # noqa: F401 — re-exported
)
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, fragment_if_available
from src.visualization._compat import (
    DataFrameLike,
    ensure_polars,
)
from src.visualization.performance import plot_cumulative_comparison


def _get_category_fr() -> dict[str, str]:
    """Retourne les labels de catégories traduits."""
    return {
        "Assassin": "Assassin",
        "Fiesta": "Fiesta",
        "BTB": t("cat_btb"),
        "Ranked": t("cat_ranked"),
        "Firefight": t("cat_firefight"),
        "Other": t("cat_other"),
    }


def _get_friends_names(df_session: DataFrameLike) -> set[str]:  # noqa: C901, PLR0912
    """Récupère les noms/nicknames des amis présents dans une session.

    Utilise les aliases chargés en session_state si disponibles,
    sinon retourne les XUIDs tronqués.

    Args:
        df_session: DataFrame des matchs avec colonne friends_xuids.

    Returns:
        Set des noms d'amis (gamertags ou nicknames).
    """
    df_session = ensure_polars(df_session)
    if df_session.is_empty() or "friends_xuids" not in df_session.columns:
        return set()

    # Collecter tous les XUIDs des amis
    friends_xuids: set[str] = set()
    for friends_str in df_session.get_column("friends_xuids").drop_nulls().to_list():
        if friends_str:
            friends_xuids.update(friends_str.split(","))
    friends_xuids.discard("")

    if not friends_xuids:
        return set()

    # Charger les noms des amis depuis la DB si possible
    friends_mapping: dict[str, str] = {}

    # 1. Essayer depuis session_state (aliases chargés)
    aliases = st.session_state.get("xuid_aliases", {})

    # 2. Essayer de charger depuis la table Friends
    db_path = st.session_state.get("db_path")
    xuid = st.session_state.get("player_xuid")
    if db_path and xuid:
        try:
            # Utiliser le repo caché au lieu d'ouvrir une nouvelle connexion
            # (8bis.A3 : gain ~200-400ms par visite de page)
            if db_path.endswith(".duckdb"):
                from src.ui.cache_loaders import get_cached_repository_st

                repo = get_cached_repository_st(db_path, xuid)
                conn = repo._get_connection()
                # Vérifier si la table existe
                from src.utils.db import has_table

                if has_table(conn, "friends"):
                    result = conn.execute(
                        "SELECT friend_xuid, friend_gamertag, nickname FROM friends WHERE owner_xuid = ?",
                        (xuid,),
                    ).fetchall()
                    for row in result:
                        fxuid, gamertag, nickname = row
                        friends_mapping[fxuid] = nickname or gamertag or fxuid
        except Exception:
            pass

    # Construire les noms
    names: set[str] = set()
    for xuid in friends_xuids:
        if xuid in friends_mapping:
            names.add(friends_mapping[xuid])
        elif xuid in aliases:
            names.add(aliases[xuid])
        else:
            # Garder le XUID tronqué comme fallback
            names.add(xuid[-6:] if len(xuid) > 6 else xuid)

    return names


def _select_sessions(session_labels: list[str]) -> tuple[str, str]:
    """Affiche les sélecteurs de sessions A et B et retourne les labels choisis."""
    col_sel_a, col_sel_b = st.columns(2)
    with col_sel_a:
        # Session A = avant-dernière par défaut
        default_a = session_labels[1] if len(session_labels) > 1 else session_labels[0]
        session_a_label = st.selectbox(
            t("sc_session_a_ref"),
            options=session_labels,
            index=session_labels.index(default_a) if default_a in session_labels else 1,
            key="compare_session_a",
        )
    with col_sel_b:
        # Session B = dernière par défaut
        session_b_label = st.selectbox(
            t("sc_session_b_cmp"),
            options=session_labels,
            index=0,
            key="compare_session_b",
        )
    return session_a_label, session_b_label


def _build_session_labels(
    all_sessions_df: DataFrameLike,
    df_session_b: DataFrameLike,
    hist_avg: dict,
    compare_mode: str,
    session_b_category: str,
) -> tuple[str, str]:
    """Construit les labels de type de session et de comparaison."""
    has_with_friends_col = "is_with_friends" in all_sessions_df.columns
    session_b_with_friends = (
        is_session_with_friends(df_session_b) if has_with_friends_col else False
    )
    session_b_friends = (
        get_session_friends_signature(df_session_b) if has_with_friends_col else set()
    )
    cat_label = _get_category_fr().get(session_b_category, session_b_category)

    if not has_with_friends_col:
        session_type_label = t("sc_friends_unavailable")
        compare_label = t("sc_compare_all", n=hist_avg.get("session_count", 0), cat=cat_label)
    elif session_b_with_friends and len(session_b_friends) > 0:
        # Récupérer les gamertags des amis (depuis la table Friends si possible)
        friends_names = _get_friends_names(df_session_b)
        if friends_names:
            friends_display = ", ".join(sorted(friends_names))
            session_type_label = t("sc_with_friends", names=friends_display)
        else:
            session_type_label = t("sc_with_n_friends", n=len(session_b_friends))

        if compare_mode == "same_friends":
            compare_label = t(
                "sc_compare_same_friends", n=hist_avg.get("session_count", 0), cat=cat_label
            )
        else:
            compare_label = t(
                "sc_compare_with_friends", n=hist_avg.get("session_count", 0), cat=cat_label
            )
    else:
        session_type_label = t("sc_solo")
        compare_label = t("sc_compare_solo", n=hist_avg.get("session_count", 0), cat=cat_label)

    return session_type_label, compare_label


def _render_score_cards(perf_a: dict, perf_b: dict) -> None:
    """Affiche les grandes cartes de score côte à côte."""
    score_a = perf_a.get("score")
    score_b = perf_b.get("score")
    is_b_better = None
    if score_a is not None and score_b is not None:
        if score_b > score_a:
            is_b_better = True
        elif score_b < score_a:
            is_b_better = False

    st.markdown(t("sc_performance_score"))
    col_score_a, col_score_b = st.columns(2)
    with col_score_a:
        render_performance_score_card(
            "Session A",
            perf_a,
            is_better=not is_b_better if is_b_better is not None else None,
        )
    with col_score_b:
        render_performance_score_card("Session B", perf_b, is_better=is_b_better)

    st.markdown("---")


def _render_detailed_metrics(perf_a: dict, perf_b: dict) -> None:
    """Affiche la section des métriques détaillées."""
    st.markdown(t("sc_detailed_metrics"))

    # En-têtes
    col_h1, col_h2, col_h3 = st.columns([2, 1, 1])
    with col_h1:
        st.markdown(t("sc_metric_label"))
    with col_h2:
        st.markdown(f"**{t('sc_session_a')}**")
    with col_h3:
        st.markdown(f"**{t('sc_session_b')}**")

    st.markdown("---")

    render_metric_comparison_row(t("sc_match_count"), perf_a["matches"], perf_b["matches"], "{}")
    render_metric_comparison_row(t("sc_kda_label"), perf_a["kda"], perf_b["kda"], "{:.2f}")
    render_metric_comparison_row(
        t("sc_win_rate"), perf_a["win_rate"], perf_b["win_rate"], "{:.1f}%"
    )
    render_metric_comparison_row(
        t("sc_avg_life"),
        perf_a["avg_life_seconds"],
        perf_b["avg_life_seconds"],
        format_seconds_to_mmss,
    )

    st.markdown("---")

    render_metric_comparison_row(t("sc_total_kills"), perf_a["kills"], perf_b["kills"], "{}")
    render_metric_comparison_row(
        t("sc_total_deaths"), perf_a["deaths"], perf_b["deaths"], "{}", higher_is_better=False
    )
    render_metric_comparison_row(t("sc_total_assists"), perf_a["assists"], perf_b["assists"], "{}")


def _render_mmr_comparison(perf_a: dict, perf_b: dict) -> None:
    """Affiche la section de comparaison MMR."""
    st.markdown("---")
    st.markdown(t("sc_mmr_comparison"))

    col_mmr1, col_mmr2, col_mmr3 = st.columns([2, 1, 1])
    with col_mmr1:
        st.markdown(t("sc_mmr_metric_label"))
    with col_mmr2:
        st.markdown(f"**{t('sc_session_a')}**")
    with col_mmr3:
        st.markdown(f"**{t('sc_session_b')}**")

    render_metric_comparison_row(
        t("sc_mmr_team_avg"), perf_a["team_mmr_avg"], perf_b["team_mmr_avg"], "{:.1f}"
    )
    render_metric_comparison_row(
        t("sc_mmr_enemy_avg"),
        perf_a["enemy_mmr_avg"],
        perf_b["enemy_mmr_avg"],
        "{:.1f}",
        higher_is_better=False,
    )
    render_metric_comparison_row(
        t("sc_mmr_gap_avg"), perf_a["delta_mmr_avg"], perf_b["delta_mmr_avg"], "{:+.1f}"
    )


@fragment_if_available
def _render_cumulative_section(
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
    session_a_label: str,
    session_b_label: str,
) -> None:
    """Affiche la section du net score cumulé par session."""
    df_session_a = ensure_polars(df_session_a)
    df_session_b = ensure_polars(df_session_b)
    _req = CORE_STAT_COLUMNS
    if not all(c in df_session_a.columns for c in _req) or not all(
        c in df_session_b.columns for c in _req
    ):
        return

    try:
        pl_a = df_session_a.sort("start_time").select(_req)
        pl_b = df_session_b.sort("start_time").select(_req)
        if not pl_a.is_empty() and not pl_b.is_empty():
            st.markdown(t("sc_net_score_cumul"))
            st.caption(t("sc_net_score_desc"))
            with safe_chart_render():
                fig_cumul = plot_cumulative_comparison(
                    pl_a,
                    pl_b,
                    label_a=session_a_label,
                    label_b=session_b_label,
                    title="",
                )
                if fig_cumul is not None:
                    st.plotly_chart(fig_cumul, width="stretch", config=PLOTLY_CLEAN_CONFIG)
                else:
                    st.info(t("insufficient_data_chart"))
    except Exception:
        pass


@fragment_if_available
def render_session_comparison_page(
    all_sessions_df: DataFrameLike,
    df_full: DataFrameLike | None = None,
    *,
    db_path: str | None = None,
    xuid: str | None = None,
) -> None:
    """Rend la page de comparaison de sessions (orchestrateur)."""
    all_sessions_df = ensure_polars(all_sessions_df)
    if df_full is not None:
        df_full = ensure_polars(df_full)
    # Récupérer db_path et xuid depuis session_state si non fournis
    if db_path is None:
        db_path = st.session_state.get("db_path", "")
    if xuid is None:
        xuid = st.session_state.get("xuid", "")
    st.caption(t("sc_loading_caption"))

    if all_sessions_df.is_empty():
        st.info(t("sc_no_sessions"))
        return

    # Liste des sessions triées (plus récente en premier)
    session_info = (
        all_sessions_df.group_by(["session_id", "session_label"])
        .agg(pl.col("start_time").max().alias("last_match_time"), pl.len().alias("count"))
        .sort("last_match_time", descending=True)
    )
    session_labels = session_info.get_column("session_label").to_list()

    if len(session_labels) < 2:
        st.warning(t("sc_need_two_sessions"))
        return

    # Sélecteurs de sessions
    session_a_label, session_b_label = _select_sessions(session_labels)

    # Filtrer les DataFrames
    df_session_a = all_sessions_df.filter(pl.col("session_label") == session_a_label)
    df_session_b = all_sessions_df.filter(pl.col("session_label") == session_b_label)

    # Calculer les scores de performance (v2)
    perf_a = compute_session_performance_score_v2_ui(df_session_a)
    perf_b = compute_session_performance_score_v2_ui(df_session_b)

    # Contexte historique
    session_a_id = (
        df_session_a[0, "session_id"]
        if not df_session_a.is_empty() and "session_id" in df_session_a.columns
        else None
    )
    session_b_id = (
        df_session_b[0, "session_id"]
        if not df_session_b.is_empty() and "session_id" in df_session_b.columns
        else None
    )
    exclude_ids = [sid for sid in [session_a_id, session_b_id] if sid is not None]
    session_b_category = infer_session_dominant_category(df_session_b)

    hist_avg, compare_mode = compute_historical_context(
        all_sessions_df,
        df_session_b,
        exclude_ids,
        session_b_category,
    )
    session_type_label, compare_label = _build_session_labels(
        all_sessions_df,
        df_session_b,
        hist_avg,
        compare_mode,
        session_b_category,
    )

    # ══════════════════════════════════════════════════════════════════════════
    # Sections de la page
    # ══════════════════════════════════════════════════════════════════════════
    render_session_temporal_header(df_session_a, df_session_b)
    _render_score_cards(perf_a, perf_b)

    # Répartition des résultats (W/L/T)
    render_outcomes_distribution(df_session_a, df_session_b)

    _render_detailed_metrics(perf_a, perf_b)
    _render_mmr_comparison(perf_a, perf_b)

    # Graphiques comparatifs
    st.markdown("---")
    if hist_avg and hist_avg.get("session_count", 0) >= 1:
        st.markdown(
            f"{t('sc_comparative_charts')}\n"
            f"*Session {session_type_label} — Moyenne historique : {compare_label}*"
        )
    else:
        st.markdown(f"{t('sc_comparative_charts')}\n*Session {session_type_label}*")

    col_radar, col_bars = st.columns(2)
    with col_radar:
        st.markdown(t("sc_radar_view"))
        render_comparison_radar_chart(perf_a, perf_b, hist_avg=hist_avg)
    with col_bars:
        st.markdown(t("sc_metric_comparison"))
        render_comparison_bar_chart(perf_a, perf_b, hist_avg=hist_avg)

    # Net score cumulé + évolution F/M
    _render_cumulative_section(df_session_a, df_session_b, session_a_label, session_b_label)
    render_kd_progression(df_session_a, df_session_b, session_a_label, session_b_label)

    # Tendance de participation (PersonalScores) - Sprint 8.2
    render_participation_trend_section(
        df_session_a=df_session_a,
        df_session_b=df_session_b,
        db_path=db_path,
        xuid=xuid,
    )

    # Tableau historique des parties
    st.markdown("---")
    st.markdown(t("sc_match_history"))
    tab_hist_a, tab_hist_b = st.tabs([t("sc_session_a"), t("sc_session_b")])
    with tab_hist_a:
        render_session_history_table(df_session_a, "Session A", df_full=df_full)
    with tab_hist_b:
        render_session_history_table(df_session_b, "Session B", df_full=df_full)
