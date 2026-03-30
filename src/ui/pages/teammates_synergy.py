"""Radars de complémentarité (synergy) pour la page Coéquipiers.

Extraits de teammates.py (Sprint 16 — refactoring Phase A).
"""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis.participation_radar import (
    RADAR_THRESHOLDS_PER_MODE,
    get_mode_family,
    is_objective_mode_from_pair_name,
)
from src.config import OKABE_ITO_PALETTE
from src.data.repositories import DuckDBRepository
from src.ui.chart_utils import safe_chart_render
from src.ui.components.radar_chart import create_participation_profile_radar
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG
from src.utils.db import duckdb_read_only
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.participation_radar import (
    RADAR_THRESHOLDS,
    compute_participation_profile,
    get_radar_axis_lines,
    get_radar_thresholds,
)


def _extract_player_xuid(df_player: pl.DataFrame) -> str | None:
    """Extrait le xuid d'un DataFrame joueur si disponible."""
    if "xuid" not in df_player.columns or df_player.is_empty():
        return None
    xuid_val = str(df_player["xuid"].item(0) or "").strip()
    return xuid_val or None


def _db_has_xuid(db_path: Path, xuid: str) -> bool:
    """Vérifie si une DB joueur correspond au xuid attendu via sync_meta."""
    try:
        with duckdb_read_only(db_path) as conn:
            row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid' LIMIT 1").fetchone()
        return bool(row and str(row[0]).strip() == xuid)
    except Exception:
        return False


def _resolve_player_db_path(
    base_dir: Path, player_name: str, player_xuid: str | None
) -> Path | None:
    """Résout le chemin DB joueur en priorisant le dossier nominal puis le xuid."""
    preferred = base_dir / player_name / "stats.duckdb"
    if preferred.exists() and not player_xuid:
        return preferred
    if preferred.exists() and player_xuid and _db_has_xuid(preferred, player_xuid):
        return preferred
    if not player_xuid:
        return preferred if preferred.exists() else None

    # Fallback robuste: retrouver le dossier joueur par xuid dans data/players/*/stats.duckdb
    for db_path in base_dir.glob("*/stats.duckdb"):
        if _db_has_xuid(db_path, player_xuid):
            logger.info(
                "teammates_synergy: DB résolue par xuid pour '%s' -> %s",
                player_name,
                db_path,
            )
            return db_path
    return preferred if preferred.exists() else None


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
    if df_player.is_empty():
        return None

    if not repo.has_personal_score_awards():
        return None

    ps = repo.load_personal_score_awards_as_polars(match_ids=shared_match_ids)
    if ps.is_empty():
        return None

    # n_matches identique pour tous les joueurs → même échelle sur le radar
    n_matches = max(1, len(shared_match_ids))

    match_row = {
        "deaths": int(df_player["deaths"].sum()) if "deaths" in df_player.columns else 0,
        "time_played_seconds": float(df_player["time_played_seconds"].sum())
        if "time_played_seconds" in df_player.columns
        else 600.0 * len(df_player),
        "avg_life_seconds": float(df_player["average_life_seconds"].mean())
        if "average_life_seconds" in df_player.columns
        else 0.0,
        "pair_name": df_player["pair_name"].item(0)
        if "pair_name" in df_player.columns and len(df_player) > 0
        else None,
    }

    # Les seuils sont calibrés par match unique → scaler par n_matches pour les axes absolus
    # Les axes Impact et Survie sont des rates (pts/min, morts/min) → pas de scaling
    # Pour "objectifs" : scaler uniquement par le nombre de matchs en mode objectif,
    # car kill_score/assist_score sont gagnés dans tous les modes mais objective_score
    # ne l'est que dans les matchs CTF/Strongholds/etc.
    scaled_th = dict(thresholds or RADAR_THRESHOLDS)
    per_mode = scaled_th.pop("per_mode", None) or RADAR_THRESHOLDS_PER_MODE

    pair_names = df_player["pair_name"].to_list() if "pair_name" in df_player.columns else []
    first_pair_name = pair_names[0] if pair_names else None
    is_obj_session = is_objective_mode_from_pair_name(first_pair_name)
    if is_obj_session:
        # objectifs_raw = objective_score → seuil = somme p80 des matchs objectif
        obj_threshold = sum(
            per_mode.get(get_mode_family(pn), RADAR_THRESHOLDS_PER_MODE.get(get_mode_family(pn), 800.0))
            for pn in pair_names
            if is_objective_mode_from_pair_name(pn)
        )
        obj_threshold = max(
            obj_threshold,
            per_mode.get(get_mode_family(first_pair_name or ""), 800.0),
        )
    else:
        # objectifs_raw = kill_score_total → seuil = p80_slayer × n_matches
        obj_threshold = per_mode.get("slayer", RADAR_THRESHOLDS_PER_MODE["slayer"]) * n_matches
    scaled_th["objectifs"] = max(1.0, obj_threshold)
    logger.debug(
        "radar profil '%s': famille=%s obj_threshold=%.0f (n_matches=%d pair0=%r)",
        name, get_mode_family(first_pair_name), obj_threshold, n_matches, first_pair_name,
    )
    for _key in ("combat", "support", "score"):
        scaled_th[_key] = scaled_th[_key] * n_matches

    return compute_participation_profile(
        ps,
        match_row=match_row,
        name=name,
        color=color,
        pair_name=match_row.get("pair_name"),
        thresholds=scaled_th,
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
    friend_df = ensure_polars(friend_sub)
    friend_xuid_resolved = friend_xuid or _extract_player_xuid(friend_df)
    friend_db_path = _resolve_player_db_path(base_dir, friend_name, friend_xuid_resolved)
    if friend_db_path is not None:
        try:
            friend_repo = DuckDBRepository(str(friend_db_path), "")
            profile = _compute_player_profile(
                friend_repo,
                friend_df,
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


_SquadPlayer = tuple[str, DataFrameLike, Path | None, str]


def _build_squad_player_list(  # noqa: PLR0913
    me_name: str,
    me_df: pl.DataFrame,
    f1_name: str,
    f1_df: pl.DataFrame,
    f2_name: str | None,
    f2_df: pl.DataFrame | None,
    f3_name: str | None,
    f3_df: pl.DataFrame | None,
    base_dir: Path,
    db_path: str,
    colors_by_name: dict[str, str],
) -> list[_SquadPlayer]:
    """Construit la liste (name, df, db_path, color) pour chaque membre de l'escouade."""
    players: list[_SquadPlayer] = [
        (me_name, me_df, Path(db_path), colors_by_name.get(me_name, OKABE_ITO_PALETTE[0])),
        (
            f1_name,
            f1_df,
            _resolve_player_db_path(base_dir, f1_name, _extract_player_xuid(f1_df)),
            colors_by_name.get(f1_name, OKABE_ITO_PALETTE[1]),
        ),
    ]
    if f2_name and f2_df is not None:
        players.append(
            (
                f2_name,
                f2_df,
                _resolve_player_db_path(base_dir, f2_name, _extract_player_xuid(f2_df)),
                colors_by_name.get(f2_name, OKABE_ITO_PALETTE[2]),
            )
        )
    if f3_name and f3_df is not None:
        players.append(
            (
                f3_name,
                f3_df,
                _resolve_player_db_path(
                    base_dir, f3_name, _extract_player_xuid(ensure_polars(f3_df))
                ),
                colors_by_name.get(f3_name, OKABE_ITO_PALETTE[3]),
            )
        )
    return players


def _compute_profiles_from_squad(
    players: list[_SquadPlayer],
    shared_match_ids: list[str],
    thresholds: dict | None,
) -> list[dict]:
    """Calcule les profils radar pour chaque membre de l'escouade."""
    profiles: list[dict] = []
    for name, df_player, player_db, color in players:
        if ensure_polars(df_player).is_empty():
            logger.warning("render_trio_synergy_radar: '%s' ignoré — DataFrame vide", name)
            continue
        if player_db is None or not Path(player_db).exists():
            logger.warning(
                "render_trio_synergy_radar: '%s' — stats.duckdb introuvable → %s",
                name,
                player_db,
            )
            continue
        try:
            repo = DuckDBRepository(str(player_db), "")
            profile = _compute_player_profile(
                repo, df_player, shared_match_ids, name, color, thresholds
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
    return profiles


def render_trio_synergy_radar(  # noqa: PLR0913
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike | None,
    me_name: str,
    f1_name: str,
    f2_name: str | None,
    colors_by_name: dict[str, str],
    *,
    db_path: str | None = None,
    f3_df: DataFrameLike | None = None,
    f3_name: str | None = None,
) -> None:
    """Radar complémentarité escouade (6 axes) : moi + 1, 2 ou 3 coéquipiers."""
    me_df = ensure_polars(me_df)
    f1_df = ensure_polars(f1_df)
    if me_df.is_empty():
        return

    if db_path is None:
        db_path = st.session_state.get("db_path", "")

    def _ids(df: pl.DataFrame) -> set[str]:
        if df.is_empty() or "match_id" not in df.columns:
            return set()
        return set(df["match_id"].cast(pl.Utf8).to_list())

    to_intersect: list[DataFrameLike] = [me_df, f1_df]
    if f2_df is not None:
        to_intersect.append(ensure_polars(f2_df))
    if f3_df is not None:
        to_intersect.append(ensure_polars(f3_df))
    shared = _ids(ensure_polars(to_intersect[0]))
    for _df in to_intersect[1:]:
        shared &= _ids(ensure_polars(_df))
    shared_match_ids = list(shared)
    if not shared_match_ids:
        return

    base_dir = Path(db_path).parent.parent
    thresholds = get_radar_thresholds(db_path) if db_path else None
    players = _build_squad_player_list(
        me_name,
        me_df,
        f1_name,
        f1_df,
        f2_name,
        ensure_polars(f2_df) if f2_df is not None else None,
        f3_name,
        ensure_polars(f3_df) if f3_df is not None else None,
        base_dir,
        db_path,
        colors_by_name,
    )
    profiles = _compute_profiles_from_squad(players, shared_match_ids, thresholds)
    _render_radar_display(profiles, title=t("tms_trio_title"), show_fill=False, static_plot=False)
