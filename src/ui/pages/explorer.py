"""Page Explorer — recherche et navigation unifiée dans les matchs.

Remplace l'ancienne page "Match" avec :
- Filtres en cascade (date, escouade, type, playlist, mode, carte)
- Recherche floue par gamertag avec badges d'encounter
- Résultats en tableau HTML + vue match détaillée
- Deep linking : ``?page=Explorer&gamertag=XXX`` ou ``&match_id=XXX``
"""

from __future__ import annotations

import logging
from collections.abc import Callable
from datetime import date
from typing import Any

import polars as pl
import streamlit as st

from src.app._page_context import MatchViewParams
from src.ui.i18n import t
from src.ui.pages.explorer_data import (
    get_all_gamertags,
    load_is_with_friends,
    resolve_gamertag_to_xuid,
)
from src.ui.pages.explorer_logic import (
    classify_experience_type,
    filter_by_date,
    filter_by_squad,
    find_closest_date,
    get_distinct_dates,
)
from src.ui.pages.explorer_results import (
    render_filter_results,
    render_player_results,
    show_single_match,
)
from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Point d'entrée public
# ---------------------------------------------------------------------------


def render_explorer_page(
    df: pl.DataFrame,
    dff: pl.DataFrame,
    params: MatchViewParams,
) -> None:
    """Rend la page Explorer complète."""
    df, dff = ensure_polars(df), ensure_polars(dff)
    _inject_explorer_css()
    pending_gt, pending_mid = _consume_deep_links()

    filtered, selected_mid = _render_match_filters(
        dff,
        params["db_path"],
        params["format_datetime_fn"],
        params["normalize_mode_label_fn"],
    )

    st.divider()
    player_gt, player_xuid = _render_player_search(params["db_path"], pending_gt)

    search_clicked = st.button(t("btn_search"), width="stretch", type="primary")

    if pending_mid and isinstance(pending_mid, str) and pending_mid.strip():
        show_single_match(df, pending_mid.strip(), params)
        return

    # Auto-déclencher la recherche si un gamertag est passé en deep link
    if not search_clicked and not pending_gt:
        st.info(t("exp_select_match_hint"))
        return

    _dispatch_results(
        player_gt,
        player_xuid,
        filtered,
        selected_mid,
        df,
        params,
    )


def _consume_deep_links() -> tuple[str | None, str]:
    """Extrait les paramètres deep link depuis session_state."""
    pending_gt = st.session_state.pop("_pending_gamertag", None)
    pending_mid = st.session_state.get("match_id_input", "")
    if pending_gt:
        logger.info("Explorer deep link gamertag=%s", pending_gt)
    if pending_mid:
        logger.info("Explorer deep link match_id=%s", pending_mid)
    return pending_gt, pending_mid


def _dispatch_results(  # noqa: PLR0913
    player_gt: str,
    player_xuid: str | None,
    filtered: pl.DataFrame,
    selected_mid: str | None,
    df: pl.DataFrame,
    params: MatchViewParams,
) -> None:
    """Affiche les résultats selon le contexte de recherche."""
    logger.debug(
        "Explorer recherche: player_gt=%r, player_xuid=%r, filtered=%d matchs",
        player_gt or None,
        player_xuid or None,
        len(filtered),
    )
    if player_gt and player_xuid:
        render_player_results(
            target_xuid=player_xuid,
            target_gt=player_gt,
            df=df,
            params=params,
        )
    elif player_gt and not player_xuid:
        logger.warning("Explorer: joueur introuvable %r", player_gt)
        st.warning(t("exp_player_not_found", gamertag=player_gt))
    else:
        render_filter_results(
            filtered,
            selected_mid,
            df,
            params=params,
        )


# ---------------------------------------------------------------------------
# CSS — bords droits pour la zone de recherche
# ---------------------------------------------------------------------------


def _inject_explorer_css() -> None:
    """Injecte le CSS pour bords droits sur les inputs Explorer."""
    st.markdown(
        "<style>"
        "[data-testid='stAppViewContainer'] .stSelectbox > div > div,"
        "[data-testid='stAppViewContainer'] .stDateInput > div > div,"
        "[data-testid='stAppViewContainer'] .stTextInput > div > div"
        "{ border-radius: 0 !important; }"
        "</style>",
        unsafe_allow_html=True,
    )


# ---------------------------------------------------------------------------
# Section 1 — Filtres par match
# ---------------------------------------------------------------------------


def _render_match_filters(
    dff: pl.DataFrame,
    db_path: str,
    format_datetime_fn: Callable[..., Any],
    normalize_mode_label_fn: Callable[[str | None], str | None],
) -> tuple[pl.DataFrame, str | None]:
    """Rend les filtres en cascade et retourne (df_filtrée, match_id)."""
    if dff.is_empty():
        st.info(t("no_data_filter"))
        return dff, None

    dates = get_distinct_dates(dff)
    last_date = dates[-1] if dates else date.today()

    # Ligne 1 : Date + Escouade
    col_date, col_squad = st.columns([2, 1])
    with col_date:
        selected_date = st.date_input(
            t("exp_filter_date"),
            value=last_date,
            min_value=dates[0] if dates else None,
            max_value=dates[-1] if dates else None,
            format="DD/MM/YYYY",
            key="_exp_date",
        )
    with col_squad:
        squad_opts = [
            t("exp_squad_all"),
            t("exp_squad_solo"),
            t("exp_squad_squad"),
        ]
        squad_pick = st.selectbox(
            t("exp_filter_squad"),
            squad_opts,
            key="_exp_squad",
        )

    # Filtrage par date
    day_df = filter_by_date(dff, selected_date)
    if day_df.is_empty() and dates:
        closest = find_closest_date(dff, selected_date)
        if closest:
            logger.debug("Aucun match le %s, fallback vers %s", selected_date, closest)
            st.info(
                t("exp_no_match_date", date=closest.strftime("%d/%m/%Y")),
            )
        day_df = dff  # Fallback : garder tout

    # Filtrage escouade / solo
    squad_mode = _resolve_squad_mode(squad_pick)
    if squad_mode != "all" and "match_id" in day_df.columns:
        ids = day_df["match_id"].cast(pl.Utf8).to_list()
        friends_map = load_is_with_friends(db_path, ids)
        day_df = filter_by_squad(day_df, friends_map, squad_mode)

    # Ligne 2 : Type + Playlist + Mode + Carte
    filtered = _render_cascade_filters(day_df)

    # Ligne 3 : Sélection de match
    selected_mid = _render_match_selector(
        filtered,
        format_datetime_fn,
        normalize_mode_label_fn,
    )

    return filtered, selected_mid


def _resolve_squad_mode(squad_pick: str) -> str:
    """Convertit le label traduit en code interne ``all|solo|squad``."""
    if squad_pick == t("exp_squad_solo"):
        return "solo"
    if squad_pick == t("exp_squad_squad"):
        return "squad"
    return "all"


def _render_cascade_filters(day_df: pl.DataFrame) -> pl.DataFrame:
    """Rend les 4 filtres cascade (type, playlist, mode, carte)."""
    all_lbl = t("exp_filter_all")
    col_type, col_pl, col_mode, col_map = st.columns(4)

    playlists = _unique_sorted(day_df, "playlist_name")
    types = sorted({classify_experience_type(p) for p in playlists})
    type_labels = _experience_type_labels()

    with col_type:
        type_pick = st.selectbox(
            t("exp_filter_type"),
            [all_lbl] + [type_labels.get(tp, tp) for tp in types],
            key="_exp_type",
        )

    if type_pick != all_lbl and "playlist_name" in day_df.columns:
        code = next(
            (k for k, v in type_labels.items() if v == type_pick),
            None,
        )
        if code:
            keep = [p for p in playlists if classify_experience_type(p) == code]
            day_df = day_df.filter(pl.col("playlist_name").is_in(keep))
            playlists = keep

    with col_pl:
        pl_pick = st.selectbox(
            t("exp_filter_playlist"),
            [all_lbl] + playlists,
            key="_exp_pl",
        )
    if pl_pick != all_lbl and "playlist_name" in day_df.columns:
        day_df = day_df.filter(pl.col("playlist_name") == pl_pick)

    modes = _unique_sorted(day_df, "pair_name")
    with col_mode:
        mode_pick = st.selectbox(
            t("exp_filter_mode"),
            [all_lbl] + modes,
            key="_exp_mode",
        )
    if mode_pick != all_lbl and "pair_name" in day_df.columns:
        day_df = day_df.filter(pl.col("pair_name") == mode_pick)

    maps = _unique_sorted(day_df, "map_name")
    with col_map:
        map_pick = st.selectbox(
            t("exp_filter_map"),
            [all_lbl] + maps,
            key="_exp_map",
        )
    if map_pick != all_lbl and "map_name" in day_df.columns:
        day_df = day_df.filter(pl.col("map_name") == map_pick)

    return day_df


def _render_match_selector(
    filtered: pl.DataFrame,
    format_datetime_fn: Callable[..., Any],
    normalize_mode_label_fn: Callable[[str | None], str | None],
) -> str | None:
    """Rend le selectbox des matchs et retourne le match_id choisi."""
    if filtered.is_empty():
        return None

    sorted_df = filtered.sort("start_time", descending=True)
    options: list[tuple[str, str]] = []
    for row in sorted_df.iter_rows(named=True):
        mid = str(row.get("match_id") or "")
        dt = row.get("start_time")
        time_str = format_datetime_fn(dt) if dt else "?"
        mode_label = normalize_mode_label_fn(row.get("pair_name")) or row.get("pair_name", "?")
        label = f"{time_str} — {mode_label} — {row.get('map_name', '?')}"
        options.append((label, mid))

    all_lbl = t("exp_filter_all")
    labels = [all_lbl] + [lbl for lbl, _ in options]
    pick = st.selectbox(
        t("exp_match_select"),
        labels,
        key="_exp_match_pick",
    )

    if pick and pick != all_lbl:
        idx = labels.index(pick) - 1
        if 0 <= idx < len(options):
            return options[idx][1]
    return None


# ---------------------------------------------------------------------------
# Section 2 — Recherche par joueur
# ---------------------------------------------------------------------------


def _render_player_search(
    db_path: str,
    pending_gt: str | None,
) -> tuple[str, str | None]:
    """Rend la section recherche joueur. Retourne (gamertag, xuid|None)."""
    st.subheader(t("exp_player_search"))
    all_gts = _cached_all_gamertags(db_path)

    # Pré-sélection si un gamertag est passé en paramètre (ex: navigation)
    default_index: int | None = None
    if pending_gt:
        pending_lower = pending_gt.strip().casefold()
        for i, gt in enumerate(all_gts):
            if gt.casefold() == pending_lower:
                default_index = i
                break

    picked = st.selectbox(
        t("exp_player_hint"),
        options=all_gts,
        index=default_index,
        key="_exp_player_input",
        label_visibility="collapsed",
        placeholder=t("exp_player_hint"),
    )

    gt_value = (picked or "").strip()
    resolved_xuid = resolve_gamertag_to_xuid(db_path, gt_value) if gt_value else None

    return gt_value, resolved_xuid


@st.cache_data(ttl=300, show_spinner=False)
def _cached_all_gamertags(db_path: str) -> list[str]:
    """Cache la liste des gamertags connus (5 min TTL)."""
    return get_all_gamertags(db_path)


# ---------------------------------------------------------------------------
# Utilitaires
# ---------------------------------------------------------------------------


def _unique_sorted(df: pl.DataFrame, col: str) -> list[str]:
    """Retourne les valeurs uniques triées d'une colonne."""
    if col not in df.columns or df.is_empty():
        return []
    return df[col].drop_nulls().unique().sort().to_list()


def _experience_type_labels() -> dict[str, str]:
    """Mapping code → label traduit pour les types d'expérience."""
    return {
        "unranked": t("exp_pvp_unranked"),
        "ranked": t("exp_pvp_ranked"),
        "pve": t("exp_pve"),
    }
