"""Radars de complémentarité (synergy) pour la page Coéquipiers.

Extraits de teammates.py (Sprint 16 — refactoring Phase A).
"""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.config import OKABE_ITO_PALETTE
from src.data.repositories import DuckDBRepository
from src.ui.chart_utils import safe_chart_render
from src.ui.components.radar_chart import create_participation_profile_radar
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.participation_radar import (
    compute_participation_profile,
    get_radar_axis_lines,
    get_radar_thresholds,
)


def _compute_player_profile(  # noqa: PLR0913
    repo: DuckDBRepository,
    df_player: DataFrameLike,
    shared_match_ids: list[str],
    name: str,
    color: str,
    thresholds: dict | None,
) -> dict | None:
    """Calcule le profil de participation d'un joueur pour le radar.

    Args:
        repo: Repository DuckDB ouvert.
        df_player: DataFrame des matchs du joueur.
        shared_match_ids: IDs des matchs partagés.
        name: Nom du joueur.
        color: Couleur assignée.
        thresholds: Seuils radar (ou None).

    Returns:
        Profil dict ou None si données indisponibles.
    """
    df_player = ensure_polars(df_player)
    if not repo.has_personal_score_awards():
        return None

    ps = repo.load_personal_score_awards_as_polars(match_ids=shared_match_ids)
    if ps.is_empty():
        return None

    match_row = {
        "deaths": int(df_player["deaths"].sum()) if "deaths" in df_player.columns else 0,
        "time_played_seconds": float(df_player["time_played_seconds"].sum())
        if "time_played_seconds" in df_player.columns
        else 600.0 * len(df_player),
        "pair_name": df_player["pair_name"].item(0)
        if "pair_name" in df_player.columns and len(df_player) > 0
        else None,
    }
    return compute_participation_profile(
        ps,
        match_row=match_row,
        name=name,
        color=color,
        pair_name=match_row.get("pair_name"),
        thresholds=thresholds,
    )


def _render_radar_display(
    profiles: list[dict],
    title: str = "🤝 Complémentarité",
    *,
    show_fill: bool = True,
    static_plot: bool = True,
) -> None:
    """Affiche le radar et la légende des axes.

    Args:
        profiles: Liste de profils de participation.
        title: Titre de la section.
        show_fill: Si True, remplit les zones ; False = lignes uniquement.
        static_plot: Si True, désactive l'interactivité (pas de légende cliquable).
    """
    if not profiles:
        st.subheader(title)
        st.info(t("insufficient_data_chart"))
        return

    st.subheader(title)
    col_radar, col_legend = st.columns([2, 1])
    with col_radar, safe_chart_render():
        fig = create_participation_profile_radar(
            profiles,
            title=t("tms_participation_title"),
            height=380,
            show_fill=show_fill,
        )
        if fig is not None:
            config = PLOTLY_STATIC_CONFIG if static_plot else PLOTLY_CLEAN_CONFIG
            st.plotly_chart(fig, width="stretch", config=config)
        else:
            st.info(t("insufficient_data_chart"))
    with col_legend:
        st.markdown(t("tms_axes"))
        for line in get_radar_axis_lines():
            st.markdown(line)


def render_synergy_radar(  # noqa: PLR0913
    sub: DataFrameLike,
    friend_sub: DataFrameLike,
    me_name: str,
    friend_name: str,
    colors_by_name: dict[str, str],
    *,
    db_path: str | None = None,
    xuid: str | None = None,
    friend_xuid: str | None = None,
) -> None:
    """Affiche le radar de complémentarité (6 axes) entre moi et un coéquipier.

    Objectifs, Combat, Support, Score, Impact, Survie.
    Utilise PersonalScores depuis ma DB et la DB du coéquipier.
    """
    sub = ensure_polars(sub)
    friend_sub = ensure_polars(friend_sub)
    if sub.is_empty() or friend_sub.is_empty():
        return

    if db_path is None:
        db_path = st.session_state.get("db_path", "")
    if xuid is None:
        xuid = st.session_state.get("xuid", "")

    shared_match_ids = list(
        set(sub["match_id"].cast(pl.Utf8).to_list())
        & set(friend_sub["match_id"].cast(pl.Utf8).to_list())
    )
    if not shared_match_ids:
        return

    thresholds = get_radar_thresholds(db_path) if db_path else None
    profiles: list[dict] = []

    # Mon profil
    try:
        repo = DuckDBRepository(db_path, xuid or "")
        profile = _compute_player_profile(
            repo,
            sub,
            shared_match_ids,
            me_name,
            colors_by_name.get(me_name, "#636EFA"),
            thresholds,
        )
        if profile:
            profiles.append(profile)
    except Exception:
        pass

    # Profil du coéquipier (depuis sa DB)
    base_dir = Path(db_path).parent.parent
    friend_db_path = base_dir / friend_name / "stats.duckdb"
    if friend_db_path.exists():
        try:
            friend_repo = DuckDBRepository(str(friend_db_path), "")
            profile = _compute_player_profile(
                friend_repo,
                friend_sub,
                shared_match_ids,
                friend_name,
                colors_by_name.get(friend_name, "#EF553B"),
                thresholds,
            )
            if profile:
                profiles.append(profile)
        except Exception:
            pass

    _render_radar_display(profiles)


def render_trio_synergy_radar(  # noqa: PLR0913, C901
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike,
    me_name: str,
    f1_name: str,
    f2_name: str,
    colors_by_name: dict[str, str],
    *,
    db_path: str | None = None,
    f3_df: DataFrameLike | None = None,
    f3_name: str | None = None,
) -> None:
    """Radar complémentarité escouade (6 axes) : moi + 2 ou 3 coéquipiers."""
    me_df = ensure_polars(me_df)
    f1_df = ensure_polars(f1_df)
    f2_df = ensure_polars(f2_df)
    if me_df.is_empty():
        return

    if db_path is None:
        db_path = st.session_state.get("db_path", "")

    def _match_ids_set(df: pl.DataFrame) -> set[str]:
        if df.is_empty() or "match_id" not in df.columns:
            return set()
        return set(df["match_id"].cast(pl.Utf8).to_list())

    to_intersect = [me_df, f1_df, f2_df] + ([f3_df] if f3_df is not None else [])
    shared_match_ids_set = _match_ids_set(ensure_polars(to_intersect[0]))
    for _df in to_intersect[1:]:
        shared_match_ids_set &= _match_ids_set(ensure_polars(_df))
    shared_match_ids = list(shared_match_ids_set)
    if not shared_match_ids:
        return

    thresholds = get_radar_thresholds(db_path) if db_path else None
    base_dir = Path(db_path).parent.parent
    profiles: list[dict] = []

    players = [
        (me_name, me_df, db_path, colors_by_name.get(me_name, OKABE_ITO_PALETTE[0])),
        (
            f1_name,
            f1_df,
            str(base_dir / f1_name / "stats.duckdb"),
            colors_by_name.get(f1_name, OKABE_ITO_PALETTE[1]),
        ),
        (
            f2_name,
            f2_df,
            str(base_dir / f2_name / "stats.duckdb"),
            colors_by_name.get(f2_name, OKABE_ITO_PALETTE[2]),
        ),
    ]
    if f3_name and f3_df is not None:
        players.append(
            (
                f3_name,
                f3_df,
                str(base_dir / f3_name / "stats.duckdb"),
                colors_by_name.get(f3_name, OKABE_ITO_PALETTE[3]),
            )
        )

    for name, df_player, player_db, color in players:
        if ensure_polars(df_player).is_empty():
            logger.debug("render_trio_synergy_radar: %s ignoré (DataFrame vide)", name)
            continue
        if not Path(player_db).exists():
            logger.warning(
                "render_trio_synergy_radar: DB introuvable pour '%s' → %s",
                name,
                player_db,
            )
            continue
        try:
            repo = DuckDBRepository(player_db, "")
            profile = _compute_player_profile(
                repo,
                df_player,
                shared_match_ids,
                name,
                color,
                thresholds,
            )
            if profile:
                profiles.append(profile)
            else:
                logger.debug(
                    "render_trio_synergy_radar: profil None pour '%s' (personal_score_awards vide?)",
                    name,
                )
        except Exception:
            logger.exception("render_trio_synergy_radar: erreur profil '%s'", name)

    # Trio : lignes uniquement (show_fill=False) + légende cliquable (static_plot=False)
    _render_radar_display(
        profiles,
        title=t("tms_trio_title"),
        show_fill=False,
        static_plot=False,
    )
