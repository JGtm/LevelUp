"""Graphique combiné Tirs à la tête + Frags parfaits pour la page Coéquipiers.

Barres empilées par joueur (base commune + excédent HS + excédent PK)
permettant de comparer les deux métriques en un seul graphe.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.config import PLOT_CONFIG
from src.ui.date_formats import FMT_SHORT_DATETIME_FR
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom


def plot_hs_pk_stacked(  # noqa: C901, PLR0912, PLR0915
    series: list[tuple[str, DataFrameLike]],
    *,
    colors: dict[str, str] | None = None,
    lang: str = "fr",
    records: dict[str, dict[str, float | None]] | None = None,
) -> go.Figure | None:
    """Graphique combiné Tirs à la tête + Frags parfaits par match (barres superposées).

    Pour chaque joueur, deux barres superposées au même offset :
    - Tirs à la tête (couleur pleine, arrière-plan)
    - Frags parfaits (même couleur, bordure blanche si > 0, premier plan)

    Args:
        series: Liste de tuples (nom_joueur, DataFrame). Chaque DataFrame doit
            contenir les colonnes ``match_id``, ``start_time``,
            ``headshot_kills`` et ``perfect_kills``.
        colors: Mapping nom_joueur → couleur hex.
        lang: Code langue (``"fr"`` ou ``"en"``).

    Returns:
        Figure Plotly ou ``None`` si les données sont insuffisantes.
    """
    if not series:
        return None

    normalized: list[tuple[str, pl.DataFrame]] = []
    for name, df_ in series:
        if df_ is None:
            continue
        df_pl = ensure_polars(df_)
        if df_pl.is_empty():
            continue
        needed = {"start_time", "headshot_kills", "perfect_kills"}
        if not needed.issubset(df_pl.columns):
            continue
        normalized.append((str(name), df_pl))

    if not normalized:
        return None

    has_match_id = all("match_id" in df_pl.columns for _, df_pl in normalized)

    # ── Construire l'axe X commun ─────────────────────────────────────────────
    all_match_data: list[pl.DataFrame] = []
    prepared: list[tuple[str, pl.DataFrame]] = []

    for name, df_pl in normalized:
        cols = ["start_time", "headshot_kills", "perfect_kills"]
        if has_match_id:
            cols.append("match_id")
        if "map_name" in df_pl.columns:
            cols.append("map_name")

        d = df_pl.select(cols)
        st_dtype = d.schema["start_time"]
        if st_dtype in (pl.String, pl.Utf8):
            d = d.with_columns(pl.col("start_time").str.to_datetime(strict=False))
        elif st_dtype == pl.Date:
            d = d.with_columns(pl.col("start_time").cast(pl.Datetime))

        d = d.drop_nulls("start_time").sort("start_time")
        if has_match_id:
            d = d.unique(subset=["match_id"], keep="first")
        else:
            d = d.unique(subset=["start_time"], keep="first")

        if d.is_empty():
            continue

        prepared.append((name, d))

        if has_match_id:
            map_cols = [pl.col("match_id").cast(pl.String).alias("match_id"), pl.col("start_time")]
            _disp_col_hs = "map_ui" if "map_ui" in d.columns else "map_name"
            if _disp_col_hs in d.columns:
                map_cols.append(pl.col(_disp_col_hs).fill_null("").alias("_map_display"))
            all_match_data.append(d.select(map_cols))
        else:
            ts_cols = [
                pl.col("start_time").dt.strftime("%Y-%m-%dT%H:%M:%S").alias("match_id"),
                pl.col("start_time"),
            ]
            _disp_col_hs2 = "map_ui" if "map_ui" in d.columns else "map_name"
            if _disp_col_hs2 in d.columns:
                ts_cols.append(pl.col(_disp_col_hs2).fill_null("").alias("_map_display"))
            all_match_data.append(d.select(ts_cols))

    if not prepared or not all_match_data:
        return None

    combined = pl.concat(all_match_data, how="diagonal")
    agg_exprs = [pl.col("start_time").min()]
    if "_map_display" in combined.columns:
        agg_exprs.append(pl.col("_map_display").first())
    match_times = combined.group_by("match_id").agg(agg_exprs).sort("start_time")

    match_ids_ordered = match_times.get_column("match_id").to_list()
    idx_map_df = pl.DataFrame(
        {"_match_key": match_ids_ordered, "_x": list(range(len(match_ids_ordered)))}
    )

    n_matches = len(match_ids_ordered)
    if "_map_display" in match_times.columns:
        labels = [
            f"#{i + 1}<br>{mn}" if mn else f"#{i + 1}"
            for i, mn in enumerate(match_times.get_column("_map_display").fill_null("").to_list())
        ]
    else:
        labels = match_times.get_column("start_time").dt.strftime(FMT_SHORT_DATETIME_FR).to_list()
    step = max(1, n_matches // 10) if n_matches else 1

    # ── Construire les traces ─────────────────────────────────────────────────
    fig = go.Figure()
    default_colors = ["#56B4E9", "#E69F00", "#009E73", "#CC79A7", "#D55E00"]
    _player_xs: dict[str, list[int]] = {}
    label_hs = "Tirs à la tête" if lang == "fr" else "Headshots"
    label_pk = "Frags parfaits" if lang == "fr" else "Perfect kills"
    first_player_trace: set[str] = set()

    for i, (name, d) in enumerate(prepared):
        base_color = (colors or {}).get(name) or default_colors[i % len(default_colors)]

        d = d.with_columns(
            pl.col("headshot_kills").cast(pl.Float64, strict=False).fill_null(0).alias("_hs"),
            pl.col("perfect_kills").cast(pl.Float64, strict=False).fill_null(0).alias("_pk"),
        )

        if has_match_id:
            d = d.with_columns(pl.col("match_id").cast(pl.String).alias("_match_key"))
        else:
            d = d.with_columns(
                pl.col("start_time").dt.strftime("%Y-%m-%dT%H:%M:%S").alias("_match_key")
            )

        d = d.join(idx_map_df, on="_match_key", how="inner").sort("_x")
        if d.is_empty():
            continue

        xs = d.get_column("_x").to_list()
        _player_xs[name] = xs
        hs_vals = d.get_column("_hs").to_list()
        pk_vals = d.get_column("_pk").to_list()

        show_in_legend = name not in first_player_trace
        if show_in_legend:
            first_player_trace.add(name)

        # Barre tirs à la tête — couleur pleine, en arrière-plan
        fig.add_trace(
            go.Bar(
                x=xs,
                y=hs_vals,
                name=name,
                marker_color=base_color,
                opacity=0.85,
                offsetgroup=name,
                legendgroup=name,
                showlegend=show_in_legend,
                hovertemplate=f"{name} {label_hs}: %{{y:.0f}}<extra></extra>",
            )
        )

        # Barre frags parfaits — superposée, bordure blanche épaisse pour la faire ressortir
        # (bordure masquée quand pk = 0 pour ne pas polluer visuellement)
        pk_border_width = [2.5 if v > 0 else 0 for v in pk_vals]
        fig.add_trace(
            go.Bar(
                x=xs,
                y=pk_vals,
                name=f"{name} — {label_pk}",
                marker_color=base_color,
                marker_line_color="white",
                marker_line_width=pk_border_width,
                opacity=1.0,
                offsetgroup=name,
                legendgroup=name,
                showlegend=False,
                hovertemplate=f"{name} {label_pk}: %{{y:.0f}}<extra></extra>",
            )
        )

    if not fig.data:
        return None

    if records and _player_xs:
        from src.visualization._squad_record_shapes import add_overlay_record_shapes
        _names_with_data = list(_player_xs.keys())
        for _pname, _xs in _player_xs.items():
            add_overlay_record_shapes(
                fig,
                xs=_xs,
                records={_pname: (records.get(_pname) or {}).get("hs_pk_total")},
                player_names=_names_with_data,
            )

    # ── Légende manuelle (annotations) ───────────────────────────────────────
    # Titre du graphe et mise en forme
    title = viz_t("hs_pk_combined_title", lang)

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 160},
        hovermode="closest",
        legend={**get_legend_horizontal_bottom(), "y": -0.55},
        barmode="overlay",
    )
    fig.update_yaxes(title_text=viz_t("hs_pk_y_axis", lang), rangemode="tozero")
    fig.update_xaxes(
        title_text=viz_t("axis_match_number", lang),
        tickmode="array",
        tickvals=list(range(len(labels)))[::step],
        ticktext=labels[::step],
        tickangle=-45,
    )

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)
