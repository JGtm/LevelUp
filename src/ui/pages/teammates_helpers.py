"""Utilitaires pour la page Coéquipiers.

Fonctions d'aide pour l'affichage des cartes joueurs et tableaux.
"""

from __future__ import annotations

import html as html_lib
import urllib.parse

import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.ui import (
    display_name_from_xuid,
    get_hero_html,
    get_profile_appearance,
    translate_playlist_name,
)
from src.ui.cache import cached_load_player_match_result
from src.ui.date_formats import FMT_DATETIME_FR
from src.ui.i18n import t
from src.ui.pages.win_loss_table_style import map_name_cell_html
from src.ui.player_assets import ensure_local_image_path
from src.ui.vectorize_helpers import build_mapping
from src.visualization._compat import DataFrameLike, ensure_polars


def _format_datetime_fr_hm(dt: object) -> str:
    """Formate une date FR avec heures/minutes."""
    if dt is None:
        return "-"
    try:
        return dt.strftime(FMT_DATETIME_FR)
    except Exception:
        return str(dt)


def _normalize_mode_label(pair_name: str | None) -> str | None:
    """Normalise un pair_name en label UI."""
    from src.ui.translations import translate_pair_name

    return translate_pair_name(pair_name) if pair_name else None


def _app_url(page: str, **params: str) -> str:
    """Génère une URL interne vers une page de l'app."""
    base = "/"
    qp = {"page": page, **params}
    return base + "?" + urllib.parse.urlencode(qp)


def _clear_min_matches_maps_friends_auto() -> None:
    """Callback pour désactiver le mode auto du slider coéquipiers."""
    st.session_state["_min_matches_maps_friends_auto"] = False


@st.cache_data(show_spinner=False, ttl=300)
def _get_teammate_card_data(
    t_xuid: str,
    api_refresh_h: int,
    db_path: str = "",
) -> dict:
    """Récupère les données de carte d'un coéquipier (caché 5min).

    Cette fonction est cachée car get_profile_appearance peut être lente
    si les données ne sont pas en cache disque.
    """
    t_name = display_name_from_xuid(t_xuid, db_path=db_path or None)
    t_app, _ = get_profile_appearance(
        xuid=t_xuid,
        enabled=True,
        refresh_hours=api_refresh_h,
    )
    return {
        "name": t_name,
        "emblem_url": getattr(t_app, "emblem_image_url", None) if t_app else None,
        "backdrop_url": getattr(t_app, "backdrop_image_url", None) if t_app else None,
        "nameplate_url": getattr(t_app, "nameplate_image_url", None) if t_app else None,
        "service_tag": getattr(t_app, "service_tag", None) if t_app else None,
        "rank_label": getattr(t_app, "rank_label", None) if t_app else None,
        "rank_subtitle": getattr(t_app, "rank_subtitle", None) if t_app else None,
        "rank_icon_url": getattr(t_app, "rank_image_url", None) if t_app else None,
        "adornment_url": getattr(t_app, "adornment_image_url", None) if t_app else None,
    }


def render_teammate_cards(picked_xuids: list[str], settings: object, db_path: str = "") -> None:
    """Affiche les cartes Spartan des coéquipiers sélectionnés.

    Args:
        picked_xuids: Liste des XUIDs des coéquipiers.
        settings: Paramètres de l'application.
        db_path: Chemin vers la DB joueur (pour résolution gamertag).

    Note: Optimisé avec cache TTL pour éviter les appels API répétés.
    """
    if not picked_xuids:
        return

    teammate_cards_html = []
    dl_enabled = bool(getattr(settings, "profile_assets_download_enabled", False)) or bool(
        getattr(settings, "profile_api_enabled", False)
    )
    refresh_h = int(getattr(settings, "profile_assets_auto_refresh_hours", 0) or 0)
    api_refresh_h = int(getattr(settings, "profile_api_auto_refresh_hours", 0) or 0)

    for t_xuid in picked_xuids:
        # Utiliser la fonction cachée pour les données de profil
        card_data = _get_teammate_card_data(t_xuid, api_refresh_h, db_path=db_path)

        t_emblem_path = ensure_local_image_path(
            card_data["emblem_url"],
            prefix="emblem",
            download_enabled=dl_enabled,
            auto_refresh_hours=refresh_h,
        )
        t_backdrop_path = ensure_local_image_path(
            card_data["backdrop_url"],
            prefix="backdrop",
            download_enabled=dl_enabled,
            auto_refresh_hours=refresh_h,
        )
        t_nameplate_path = ensure_local_image_path(
            card_data["nameplate_url"],
            prefix="nameplate",
            download_enabled=dl_enabled,
            auto_refresh_hours=refresh_h,
        )
        t_rank_icon_path = ensure_local_image_path(
            card_data["rank_icon_url"],
            prefix="rank",
            download_enabled=dl_enabled,
            auto_refresh_hours=refresh_h,
        )
        t_adornment_path = ensure_local_image_path(
            card_data["adornment_url"],
            prefix="adornment",
            download_enabled=dl_enabled,
            auto_refresh_hours=refresh_h,
        )

        card_html = get_hero_html(
            player_name=card_data["name"],
            service_tag=card_data["service_tag"],
            rank_label=str(card_data["rank_label"] or "").strip() or None,
            rank_subtitle=str(card_data["rank_subtitle"] or "").strip() or None,
            rank_icon_path=t_rank_icon_path,
            adornment_path=t_adornment_path,
            emblem_path=t_emblem_path,
            backdrop_path=t_backdrop_path,
            nameplate_path=t_nameplate_path,
            grid_mode=True,
        )
        teammate_cards_html.append(card_html)

    if teammate_cards_html:
        cols = st.columns(2)
        for i, card_html in enumerate(teammate_cards_html):
            with cols[i % 2]:
                st.markdown(card_html, unsafe_allow_html=True)


def _prepare_friends_table_data(  # noqa: PLR0912, PLR0913
    friends_table: pl.DataFrame,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    waypoint_player: str,
    full_df: pl.DataFrame | None = None,
) -> pl.DataFrame:
    """Prépare le DataFrame pour le tableau d'historique coéquipiers.

    Ajoute les colonnes traduites, scores, MMR, delta et URLs.
    """
    friends_table = friends_table.with_columns(
        pl.col("start_time").dt.strftime(FMT_DATETIME_FR).fill_null("-").alias("start_time_fr")
    )
    if "playlist_fr" not in friends_table.columns:
        _playlist_map = build_mapping(friends_table["playlist_name"], translate_playlist_name)
        friends_table = friends_table.with_columns(
            pl.col("playlist_name")
            .cast(pl.Utf8)
            .replace_strict(
                _playlist_map, default=pl.col("playlist_name").cast(pl.Utf8), return_dtype=pl.Utf8
            )
            .alias("playlist_fr")
        )
    if "mode_ui" in friends_table.columns:
        friends_table = friends_table.with_columns(
            pl.when(
                pl.col("mode_ui").is_not_null()
                & (pl.col("mode_ui").cast(pl.Utf8).str.strip_chars() != "")
            )
            .then(pl.col("mode_ui"))
            .otherwise(pl.lit(None))
            .alias("mode")
        )
    else:
        friends_table = friends_table.with_columns(pl.lit(None).cast(pl.Utf8).alias("mode"))
    if friends_table["mode"].is_null().any() and "pair_name" in friends_table.columns:
        _mode_map = build_mapping(friends_table["pair_name"], _normalize_mode_label)
        friends_table = friends_table.with_columns(
            pl.when(pl.col("mode").is_null())
            .then(
                pl.col("pair_name")
                .cast(pl.Utf8)
                .replace_strict(_mode_map, default=None, return_dtype=pl.Utf8)
            )
            .otherwise(pl.col("mode"))
            .alias("mode")
        )
    friends_table = friends_table.with_columns(pl.col("mode").fill_null("-"))
    friends_table = friends_table.with_columns(
        pl.col("outcome")
        .replace_strict(
            {
                2: t("tmh_outcome_win"),
                3: t("tmh_outcome_loss"),
                1: t("tmh_outcome_draw"),
                4: t("tmh_outcome_unfinished"),
            },
            default="-",
            return_dtype=pl.Utf8,
        )
        .alias("outcome_label")
    )
    friends_table = _add_score_columns(friends_table)
    friends_table = _add_win_rate_column(friends_table, full_df)
    friends_table = _add_mmr_columns(friends_table, db_path, xuid, db_key)
    wp = str(waypoint_player or "").strip()
    if wp:
        friends_table = friends_table.with_columns(
            (
                pl.lit("https://www.halowaypoint.com/halo-infinite/players/" + wp + "/matches/")
                + pl.col("match_id").cast(pl.Utf8)
            ).alias("match_url")
        )
    else:
        friends_table = friends_table.with_columns(pl.lit("").alias("match_url"))
    return friends_table


def _add_win_rate_column(df: pl.DataFrame, full_df: pl.DataFrame | None) -> pl.DataFrame:
    """Ajoute win_rate_hist : % victoires sur cette carte sur TOUT l'historique escouade."""
    if "outcome" not in df.columns or "map_name" not in df.columns:
        return df.with_columns(pl.lit(None).cast(pl.Float64).alias("win_rate_hist"))
    base = full_df if full_df is not None else df
    map_wr = (
        base.group_by("map_name")
        .agg(
            pl.col("outcome").eq(2).cast(pl.Float64).sum().alias("_wins"),
            pl.len().alias("_total"),
        )
        .with_columns((pl.col("_wins") / pl.col("_total") * 100).round(1).alias("win_rate_hist"))
        .select(["map_name", "win_rate_hist"])
    )
    return df.join(map_wr, on="map_name", how="left")


def _add_score_columns(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute la colonne score formatée 'X - Y'."""
    score_cols = ["my_team_score", "enemy_team_score"]
    if not all(c in df.columns for c in score_cols):
        return df.with_columns(pl.lit("-").alias("score"))

    def _safe_score(c: str) -> pl.Expr:
        return (
            pl.when(pl.col(c).is_null() | pl.col(c).is_nan())
            .then(pl.lit("-"))
            .otherwise(pl.col(c).round(0).cast(pl.Int64).cast(pl.Utf8))
        )

    return df.with_columns(
        (_safe_score("my_team_score") + pl.lit(" - ") + _safe_score("enemy_team_score")).alias(
            "score"
        )
    )


def _add_mmr_columns(
    df: pl.DataFrame, db_path: str, xuid: str, db_key: tuple[int, int] | None
) -> pl.DataFrame:
    """Ajoute les colonnes team_mmr, enemy_mmr et delta_mmr."""
    if "team_mmr" not in df.columns or df["team_mmr"].is_null().all():
        match_ids_list = df["match_id"].cast(pl.Utf8).to_list()
        team_mmrs: list[object] = []
        enemy_mmrs: list[object] = []
        for mid in match_ids_list:
            pm = cached_load_player_match_result(db_path, str(mid), xuid.strip(), db_key=db_key)
            if not isinstance(pm, dict):
                team_mmrs.append(None)
                enemy_mmrs.append(None)
            else:
                team_mmrs.append(pm.get("team_mmr"))
                enemy_mmrs.append(pm.get("enemy_mmr"))
        df = df.with_columns([pl.Series("team_mmr", team_mmrs), pl.Series("enemy_mmr", enemy_mmrs)])
    if "team_mmr" not in df.columns:
        df = df.with_columns(pl.lit(None).alias("team_mmr"))
    if "enemy_mmr" not in df.columns:
        df = df.with_columns(pl.lit(None).alias("enemy_mmr"))
    return df.with_columns(
        pl.when(pl.col("team_mmr").is_not_null() & pl.col("enemy_mmr").is_not_null())
        .then(pl.col("team_mmr").cast(pl.Float64) - pl.col("enemy_mmr").cast(pl.Float64))
        .otherwise(pl.lit(None))
        .alias("delta_mmr")
    )


def _fmt_value(v: object) -> str:
    """Formate une valeur pour affichage HTML (tiret si vide/null/NaN)."""
    if v is None:
        return "-"
    try:
        if v != v:
            return "-"
    except Exception:
        pass
    s = str(v)
    return s if s.strip() else "-"


def _fmt_mmr_int(v: object) -> str:
    """Formate une valeur MMR en entier."""
    if v is None:
        return "-"
    try:
        if v != v:
            return "-"
    except Exception:
        pass
    try:
        return str(int(round(float(v))))
    except Exception:
        return _fmt_value(v)


def _win_rate_td(raw: object, colors: dict[str, str]) -> str:
    """Génère une cellule <td> colorée pour le taux de victoire cumulatif."""
    try:
        f = float(raw)  # type: ignore[arg-type]
        if f >= 55.0:
            style = f"color:{colors['green']}; font-weight:700"
        elif f <= 45.0:
            style = f"color:{colors['red']}; font-weight:700"
        else:
            style = f"color:{colors['cyan']}; font-weight:700"
        return f"<td style='{style}'>{html_lib.escape(f'{int(round(f))}%')}</td>"
    except Exception:
        return "<td>-</td>"


def _build_html_rows(  # noqa: C901, PLR0912
    view: pl.DataFrame,
    cols: list[tuple[str, str]],
    colors: dict[str, str],
) -> list[str]:
    """Construit les lignes HTML du tableau d'historique coéquipiers."""

    def _outcome_style(label: str) -> str:
        v = str(label or "").strip().casefold()
        if v.startswith("victoire") or v == "win":
            return f"color:{colors['green']}; font-weight:800"
        if v.startswith("défaite") or v.startswith("defaite") or v == "loss":
            return f"color:{colors['red']}; font-weight:800"
        if v.startswith("égalité") or v.startswith("egalite") or v == "draw":
            return f"color:{colors['violet']}; font-weight:800"
        if v.startswith("non") or v.startswith("did"):
            return f"color:{colors['violet']}; font-weight:800"
        return "opacity:0.92"

    def _mmr_gap_style(v: object) -> str:
        try:
            f = float(v)
            if f != f:
                return ""
            if f > 0:
                return f"color:{colors['green']}; font-weight:600"
            if f < 0:
                return f"color:{colors['red']}; font-weight:600"
        except Exception:
            pass
        return ""

    body_rows: list[str] = []
    for r in view.to_dicts():
        mid = str(r.get("match_id") or "").strip()
        app = _app_url("Match", match_id=mid)
        match_link = (
            f"<a href='{html_lib.escape(app)}' target='_self'>{html_lib.escape(t('tmh_link_open'))}</a>"
            if mid
            else "-"
        )
        hw = str(r.get("match_url") or "").strip()
        hw_link = (
            f"<a href='{html_lib.escape(hw)}' target='_blank' rel='noopener'>{html_lib.escape(t('tmh_link_open'))}</a>"
            if hw
            else "-"
        )

        tds: list[str] = []
        for _h, key in cols:
            if key == "_app":
                tds.append(f"<td>{match_link}</td>")
            elif key == "match_url":
                tds.append(f"<td>{hw_link}</td>")
            elif key == "outcome_label":
                val = _fmt_value(r.get(key))
                tds.append(f"<td style='{_outcome_style(val)}'>{html_lib.escape(val)}</td>")
            elif key == "win_rate_hist":
                tds.append(_win_rate_td(r.get(key), colors))
            elif key in ("team_mmr", "enemy_mmr"):
                val = _fmt_mmr_int(r.get(key))
                tds.append(f"<td>{html_lib.escape(val)}</td>")
            elif key == "delta_mmr":
                raw = r.get(key)
                style = _mmr_gap_style(raw)
                try:
                    display = f"{int(round(float(raw))):+d}"
                except Exception:
                    display = "-"
                tds.append(f"<td style='{style}'>{html_lib.escape(display)}</td>")
            elif key == "map_name":
                tds.append(map_name_cell_html(r.get(key)))
            else:
                val = _fmt_value(r.get(key))
                tds.append(f"<td>{html_lib.escape(val)}</td>")
        body_rows.append("<tr>" + "".join(tds) + "</tr>")
    return body_rows


def render_friends_history_table(  # noqa: PLR0913
    sub_all: DataFrameLike,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    waypoint_player: str,
    full_df: DataFrameLike | None = None,
) -> None:
    """Affiche le tableau d'historique des matchs avec coéquipiers."""
    friends_table = ensure_polars(sub_all)
    _full = ensure_polars(full_df) if full_df is not None else None
    friends_table = _prepare_friends_table_data(
        friends_table, db_path, xuid, db_key, waypoint_player, full_df=_full
    )
    view = friends_table.sort("start_time", descending=True).head(250)

    cols = [
        (t("tmh_col_match"), "_app"),
        (t("tmh_waypoint"), "match_url"),
        (t("tmh_col_date"), "start_time_fr"),
        (t("tmh_col_map"), "map_name"),
        (t("tmh_col_playlist"), "playlist_fr"),
        (t("tmh_col_mode"), "mode"),
        (t("tmh_col_result"), "outcome_label"),
        (t("col_win_rate_hist"), "win_rate_hist"),
        (t("tmh_col_score"), "score"),
        (t("tmh_col_team_mmr"), "team_mmr"),
        (t("tmh_col_enemy_mmr"), "enemy_mmr"),
        (t("tmh_col_delta_mmr"), "delta_mmr"),
    ]

    head = "".join(f"<th>{html_lib.escape(h)}</th>" for h, _ in cols)
    body_rows = _build_html_rows(view, cols, HALO_COLORS.as_dict())

    st.markdown(
        "<div class='os-table-wrap os-table-wrap--map-hover'><table class='os-table'><thead><tr>"
        + head
        + "</tr></thead><tbody>"
        + "".join(body_rows)
        + "</tbody></table></div>",
        unsafe_allow_html=True,
    )
