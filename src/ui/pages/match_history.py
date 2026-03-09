"""Page Historique des parties.

Tableau complet de l'historique des matchs avec liens et MMR.

Sprint 4.2 : Optimisation N+1
- Les colonnes team_mmr et enemy_mmr sont déjà dans le DataFrame
- Plus besoin de requête individuelle par match (était: 500 requêtes)
- Gain de performance: ~90% (1 requête batch vs N requêtes)

Sprint 8bis : Vectorisation
- Remplacement de 7 map_elements() par des expressions Polars natives
- Gain de performance: ~50% sur le rendu de 250 matchs
"""

from __future__ import annotations

import html as html_lib

import polars as pl
import streamlit as st

from src.analysis.performance_score import compute_performance_series
from src.config import HALO_COLORS, OUTCOME_CODES
from src.ui.components.performance import get_score_class
from src.ui.date_formats import FMT_DATETIME_FR
from src.ui.i18n import get_outcome_map, t
from src.ui.pages.match_table_html import app_url
from src.visualization._compat import DataFrameLike, ensure_polars


def render_match_history_page(  # noqa: PLR0913
    dff: DataFrameLike,
    waypoint_player: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    df_full: DataFrameLike | None = None,
) -> None:
    """Affiche la page Historique des parties.

    Args:
        dff: DataFrame filtré des matchs.
        waypoint_player: Nom Waypoint du joueur.
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur.
        db_key: Clé de cache de la DB.
        df_full: DataFrame complet (non filtré) pour le calcul du score relatif.
    """
    dff = ensure_polars(dff)

    # Protection contre les DataFrames vides
    if dff.is_empty():
        st.warning(t("no_matches"))
        return

    st.subheader(t("mh_title"))

    dff_table = dff.clone()
    if "playlist_fr" not in dff_table.columns:
        # Vectorisation: utiliser replace_strict() au lieu de map_elements()
        from src.ui.translations import PLAYLIST_FR

        dff_table = dff_table.with_columns(
            pl.col("playlist_name")
            .replace_strict(PLAYLIST_FR, default=pl.col("playlist_name"))
            .alias("playlist_fr")
        )
    if "mode_ui" not in dff_table.columns:
        # Vectorisation: utiliser replace_strict() avec le dictionnaire PAIR_FR
        from src.ui.translations import PAIR_FR

        dff_table = dff_table.with_columns(
            pl.col("pair_name")
            .replace_strict(PAIR_FR, default=pl.col("pair_name"))
            .alias("mode_ui")
        )
    dff_table = dff_table.with_columns(
        (
            pl.lit("https://www.halowaypoint.com/halo-infinite/players/")
            + pl.lit(waypoint_player.strip())
            + pl.lit("/matches/")
            + pl.col("match_id").cast(pl.Utf8)
        ).alias("match_url")
    )

    # Vectorisation outcome_label: replace_strict() pour éviter dépréciation
    outcome_map = get_outcome_map()
    dff_table = dff_table.with_columns(
        pl.col("outcome").replace_strict(outcome_map, default=pl.lit("-")).alias("outcome_label")
    )

    # Vectorisation score: concat_str() au lieu de map_elements()
    dff_table = dff_table.with_columns(
        pl.concat_str(
            [
                pl.col("my_team_score")
                .cast(pl.Float64, strict=False)
                .round(0)
                .cast(pl.Int64, strict=False)
                .fill_null(pl.lit("-"))
                .cast(pl.Utf8),
                pl.lit(" - "),
                pl.col("enemy_team_score")
                .cast(pl.Float64, strict=False)
                .round(0)
                .cast(pl.Int64, strict=False)
                .fill_null(pl.lit("-"))
                .cast(pl.Utf8),
            ]
        ).alias("score")
    )

    # MMR équipe/adverse - Sprint 4.2 : Optimisation N+1
    # Les colonnes sont déjà dans le DataFrame (chargées par load_matches)
    # Plus de boucle N+1 (était: 1 requête par match = 500+ requêtes)
    if "team_mmr" not in dff_table.columns:
        dff_table = dff_table.with_columns(pl.lit(None).cast(pl.Float64).alias("team_mmr"))
    if "enemy_mmr" not in dff_table.columns:
        dff_table = dff_table.with_columns(pl.lit(None).cast(pl.Float64).alias("enemy_mmr"))

    # Calcul du delta MMR (vectorisé, pas de boucle)
    dff_table = dff_table.with_columns(
        (
            pl.col("team_mmr").cast(pl.Float64, strict=False)
            - pl.col("enemy_mmr").cast(pl.Float64, strict=False)
        ).alias("delta_mmr")
    )

    # Vectorisation start_time_fr: strftime() au lieu de map_elements()
    dff_table = dff_table.with_columns(
        pl.col("start_time").dt.strftime(FMT_DATETIME_FR).fill_null("-").alias("start_time_fr")
    )
    # Vectorisation average_life_mmss: calcul arithmétique au lieu de map_elements()
    dff_table = dff_table.with_columns(
        pl.concat_str(
            [
                (pl.col("average_life_seconds").cast(pl.Int64, strict=False) // 60)
                .fill_null(0)
                .cast(pl.Utf8),
                pl.lit(":"),
                (pl.col("average_life_seconds").cast(pl.Int64, strict=False) % 60)
                .fill_null(0)
                .cast(pl.Utf8)
                .str.zfill(2),
            ]
        ).alias("average_life_mmss")
    )

    # Calcul de la note de performance RELATIVE (basée sur l'historique complet)
    history_df = ensure_polars(df_full) if df_full is not None else dff_table
    perf_series = compute_performance_series(dff_table, history_df)
    # compute_performance_series retourne une Series Polars quand l'entrée est Polars
    if not isinstance(perf_series, pl.Series):
        perf_series = pl.Series("performance", perf_series.to_list())
    dff_table = dff_table.with_columns(perf_series.alias("performance"))
    # Vectorisation performance_display: round + cast au lieu de map_elements()
    dff_table = dff_table.with_columns(
        pl.col("performance")
        .round(0)
        .cast(pl.Int64, strict=False)
        .cast(pl.Utf8)
        .fill_null("-")
        .alias("performance_display")
    )

    # Table HTML
    _render_history_table(dff_table)

    # Export CSV
    _render_csv_download(dff_table)


def _render_history_table(dff_table: pl.DataFrame) -> None:  # noqa: C901, PLR0915
    """Affiche le tableau de l'historique via un tableau HTML avec couleurs."""
    view = dff_table.sort("start_time", descending=True).head(250)
    colors = HALO_COLORS.as_dict()

    def _fmt(v) -> str:
        if v is None:
            return "-"
        s = str(v)
        return s if s.strip() else "-"

    def _fmt_float(v, decimals: int = 2) -> str:
        if v is None:
            return "-"
        try:
            f = float(v)
            if f != f:
                return "-"
            return f"{f:.{decimals}f}"
        except Exception:
            return "-"

    def _fmt_mmr_int(v) -> str:
        if v is None:
            return "-"
        try:
            f = float(v)
            if f != f:
                return "-"
            return str(int(round(f)))
        except Exception:
            return "-"

    def _outcome_style(outcome, label: str) -> str:
        try:
            if int(outcome) == int(OUTCOME_CODES.WIN):
                return f"color:{colors['green']}; font-weight:800"
            if int(outcome) == int(OUTCOME_CODES.LOSS):
                return f"color:{colors['red']}; font-weight:800"
            if int(outcome) in (int(OUTCOME_CODES.TIE), int(OUTCOME_CODES.NO_FINISH)):
                return f"color:{colors['violet']}; font-weight:800"
        except Exception:
            pass
        v = str(label or "").strip().casefold()
        if v.startswith("victoire") or v == "win":
            return f"color:{colors['green']}; font-weight:800"
        if v.startswith("défaite") or v.startswith("defaite") or v == "loss":
            return f"color:{colors['red']}; font-weight:800"
        if v.startswith("égalité") or v.startswith("egalite") or v in ("tie", "draw"):
            return f"color:{colors['violet']}; font-weight:800"
        return "opacity:0.92"

    def _mmr_gap_style(v) -> str:
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

    cols = [
        (t("col_start_date"), "start_time_fr"),
        (t("col_map"), "map_name"),
        (t("col_playlist"), "playlist_fr"),
        (t("col_mode"), "mode_ui"),
        (t("col_result"), "outcome_label"),
        (t("col_score"), "score"),
        (t("mv_performance"), "performance"),
        (t("col_mmr_team"), "team_mmr"),
        (t("col_mmr_enemy"), "enemy_mmr"),
        (t("col_mmr_gap"), "delta_mmr"),
        (t("col_kda"), "kda"),
        (t("col_kills"), "kills"),
        (t("col_deaths"), "deaths"),
        (t("col_max_spree"), "max_killing_spree"),
        (t("col_headshots"), "headshot_kills"),
        (t("col_avg_life"), "average_life_mmss"),
        (t("col_assists"), "assists"),
        (t("col_accuracy"), "accuracy"),
        (t("col_ratio"), "ratio"),
    ]

    lbl_open = t("btn_open")
    head_cells = [
        f"<th>{html_lib.escape(lbl_open)}</th>",
        "<th>Waypoint</th>",
    ] + [f"<th>{html_lib.escape(h)}</th>" for h, _ in cols]
    head = "".join(head_cells)

    body_rows: list[str] = []
    for r in view.to_dicts():
        mid = str(r.get("match_id") or "").strip()
        app = app_url(
            "Explorer",
            match_id=mid,
            gamertag=str(st.session_state.get("waypoint_player") or "").strip(),
        )
        match_link = (
            f"<a href='{html_lib.escape(app)}' target='_self'>{html_lib.escape(lbl_open)}</a>"
            if mid
            else "-"
        )
        hw = str(r.get("match_url") or "").strip()
        hw_link = (
            f"<a href='{html_lib.escape(hw)}' target='_blank' rel='noopener'>{html_lib.escape(lbl_open)}</a>"
            if hw
            else "-"
        )

        tds = [f"<td>{match_link}</td>", f"<td>{hw_link}</td>"]
        outcome_code = r.get("outcome")

        for _h, key in cols:
            if key == "outcome_label":
                val = _fmt(r.get(key))
                style = _outcome_style(outcome_code, val)
                tds.append(f"<td style='{style}'>{html_lib.escape(val)}</td>")
            elif key == "performance":
                perf_val = r.get("performance")
                css_class = get_score_class(perf_val)
                display = _fmt_mmr_int(perf_val) if perf_val is not None else "-"
                tds.append(f"<td class='{css_class}'>{html_lib.escape(display)}</td>")
            elif key in ("team_mmr", "enemy_mmr"):
                tds.append(f"<td>{html_lib.escape(_fmt_mmr_int(r.get(key)))}</td>")
            elif key == "delta_mmr":
                val = r.get(key)
                style = _mmr_gap_style(val)
                try:
                    display = f"{int(round(float(val))):+d}"
                except Exception:
                    display = "-"
                tds.append(f"<td style='{style}'>{html_lib.escape(display)}</td>")
            elif key in ("kda", "accuracy", "ratio"):
                tds.append(f"<td>{html_lib.escape(_fmt_float(r.get(key)))}</td>")
            else:
                tds.append(f"<td>{html_lib.escape(_fmt(r.get(key)))}</td>")

        body_rows.append("<tr>" + "".join(tds) + "</tr>")

    st.markdown(
        "<div class='os-table-wrap'><table class='os-table'><thead><tr>"
        + head
        + "</tr></thead><tbody>"
        + "".join(body_rows)
        + "</tbody></table></div>",
        unsafe_allow_html=True,
    )


def _render_csv_download(dff_table: pl.DataFrame) -> None:
    """Affiche le bouton de téléchargement CSV."""
    show_cols = [
        "match_url",
        "start_time_fr",
        "map_name",
        "playlist_fr",
        "mode_ui",
        "outcome_label",
        "score",
        "team_mmr",
        "enemy_mmr",
        "delta_mmr",
        "kda",
        "kills",
        "deaths",
        "max_killing_spree",
        "headshot_kills",
        "average_life_mmss",
        "assists",
        "accuracy",
        "ratio",
    ]
    table = (
        dff_table.select(show_cols + ["start_time"])
        .sort("start_time", descending=True)
        .select(show_cols)
    )

    csv_table = table.rename({"start_time_fr": t("col_start_date")})
    csv = csv_table.write_csv().encode("utf-8")
    st.download_button(
        t("btn_download_csv"),
        data=csv,
        file_name="levelup_matches.csv",
        mime="text/csv",
    )
