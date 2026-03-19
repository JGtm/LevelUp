"""Graphiques pour la comparaison escouade (3 ou 4 joueurs)."""

import plotly.graph_objects as go
import polars as pl

from src.config import OKABE_ITO_PALETTE, PLOT_CONFIG
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom

# Couleurs "négatives" explicites, une par slot Okabe-Ito.
# Toutes dans la famille rouge-orange, variées en luminance/teinte pour
# rester distinguables y compris pour les daltoniens (variation de luminance).
# Slot 0 (Sky Blue)       → rouge cramoisi clair
# Slot 1 (Orange)         → rouge brique saturé
# Slot 2 (Bluish Green)   → rouge sombre (basse luminance)
# Slot 3 (Reddish Purple) → vermillon chaud
# Slot 4 (Vermilion)      → rouge profond / bordeaux
_OKABE_NEGATIVE_COLORS: dict[str, str] = {
    "#56B4E9": "#c0392b",  # cramoisi clair
    "#E69F00": "#e55b2c",  # brique orangé
    "#009E73": "#8b1a1a",  # rouge sombre
    "#CC79A7": "#d44000",  # vermillon chaud
    "#D55E00": "#7b1829",  # bordeaux profond
}
_NEGATIVE_FALLBACK = "#b41414"


def _negative_color(hex_color: str) -> str:
    """Retourne la teinte négative (rouge) associée à une couleur Okabe-Ito.

    Chaque slot de la palette Okabe-Ito a un rouge distinct, distinguable
    même en deutéranopie (variation de luminance, pas seulement de teinte).
    """
    return _OKABE_NEGATIVE_COLORS.get(hex_color, _NEGATIVE_FALLBACK)


def plot_trio_metric(  # noqa: PLR0913, C901 — graphe multi-joueurs
    d_self: DataFrameLike,
    d_f1: DataFrameLike,
    d_f2: DataFrameLike,
    *,
    metric: str,
    names: tuple[str, ...],
    title: str,
    y_title: str,
    y_suffix: str = "",
    y_format: str = "",
    smooth_window: int = 7,
    lang: str = "fr",
    d_f3: DataFrameLike | None = None,
    colors_by_name: dict[str, str] | None = None,
    is_inverse: bool = False,
    match_labels: list[str] | None = None,
) -> go.Figure:
    """Graphique comparant une métrique entre 3 ou 4 joueurs.

    Les DataFrames doivent être pré-alignés sur les mêmes matchs.

    Args:
        d_self: DataFrame du joueur principal.
        d_f1: DataFrame du premier coéquipier.
        d_f2: DataFrame du deuxième coéquipier.
        metric: Nom de la colonne à comparer.
        names: Tuple des noms (3 ou 4 éléments).
        title: Titre du graphique.
        y_title: Titre de l'axe Y.
        y_suffix: Suffixe pour les valeurs Y (ex: "%").
        y_format: Format pour le hover (ex: ".2f").
        d_f3: DataFrame optionnel du 3ème coéquipier (4ème joueur total).
        colors_by_name: Mapping nom → couleur hex. Si None, utilise OKABE_ITO_PALETTE.

    Returns:
        Figure Plotly.
    """
    all_input_dfs = [d_self, d_f1, d_f2]
    if d_f3 is not None:
        all_input_dfs.append(d_f3)

    # Palette couleurs : colors_by_name en priorité, sinon Okabe-Ito
    if colors_by_name is not None:
        color_list = [
            colors_by_name.get(n, OKABE_ITO_PALETTE[i % len(OKABE_ITO_PALETTE)])
            for i, n in enumerate(names)
        ]
    else:
        color_list = [
            OKABE_ITO_PALETTE[i % len(OKABE_ITO_PALETTE)] for i in range(len(all_input_dfs))
        ]

    # Normaliser les entrées en Polars
    pl_dfs = [ensure_polars(df) for df in all_input_dfs]

    def _prep(df: pl.DataFrame, alias: str) -> pl.DataFrame:
        """Prépare un DataFrame : sélection, cast datetime, tri."""
        if df is None or df.is_empty():
            return pl.DataFrame(schema={"start_time": pl.Datetime, alias: pl.Float64})
        out = df.select(["start_time", metric])
        # Cast start_time en Datetime si nécessaire
        if not out.schema["start_time"].is_temporal():
            out = out.with_columns(pl.col("start_time").str.to_datetime(strict=False))
        out = out.drop_nulls(subset=["start_time"]).sort("start_time").rename({metric: alias})
        return out

    col_names = [f"v{i}" for i in range(len(pl_dfs))]
    prepped = [_prep(df, col) for df, col in zip(pl_dfs, col_names, strict=False)]

    # Aligne sur l'intersection des timestamps
    aligned = prepped[0]
    for p_df in prepped[1:]:
        aligned = aligned.join(p_df, on="start_time", how="inner")

    fig = go.Figure()
    if aligned.is_empty():
        fig.update_layout(title=title)
        return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)

    def _roll(s: pl.Series) -> list:
        """Moyenne glissante, retourne une liste pour Plotly."""
        w = int(smooth_window) if smooth_window else 0
        if w <= 1:
            return s.to_list()
        return s.rolling_mean(window_size=w, min_samples=1).to_list()

    # Formatage des dates pour ticktext
    # Construction post-alignement pour garantir longueur == len(aligned) et numérotation correcte
    _ref_pl = pl_dfs[0]
    if "map_name" in _ref_pl.columns and not _ref_pl.is_empty():
        _ref_clean = _ref_pl.drop_nulls(subset=["start_time"])
        _ts_to_map: dict = dict(
            zip(
                _ref_clean["start_time"].to_list(),
                _ref_clean["map_name"].fill_null("?").to_list(),
                strict=False,
            )
        )
        ticktext = [
            f"#{i + 1}<br>{_ts_to_map.get(ts, '?')}"
            for i, ts in enumerate(aligned["start_time"].to_list())
        ]
    elif match_labels and len(match_labels) == len(aligned):
        ticktext = match_labels
    else:
        ticktext = aligned["start_time"].dt.strftime("%d/%m").fill_null("").to_list()
    xs = list(range(len(aligned)))

    series_lists = [aligned[col].to_list() for col in col_names]
    series_cols = [aligned[col] for col in col_names]

    # Moyenne horizontale de toutes les séries
    avg_all = aligned.select(pl.mean_horizontal(*col_names)).to_series()

    for _idx, (s_list, s_col, name, color) in enumerate(
        zip(series_lists, series_cols, names, color_list, strict=False)
    ):
        hover_format = f"%{{y{':' + y_format if y_format else ''}}}{y_suffix}<extra></extra>"
        neg_color = _negative_color(color)
        bar_colors = [color if (v is not None and v >= 0) else neg_color for v in s_list]
        bar_kwargs: dict = {
            "x": xs,
            "y": s_list,
            "name": f"{name} (match)",
            "marker_color": bar_colors,
            "opacity": 0.75,
            "customdata": ticktext,
            "hovertemplate": hover_format,
        }
        if is_inverse:
            bar_kwargs["marker"] = {
                "color": bar_colors,
                "pattern": {
                    "shape": "/",
                    "fgcolor": "rgba(255, 80, 80, 0.5)",
                    "solidity": 0.15,
                },
            }
        fig.add_trace(go.Bar(**bar_kwargs))

        fig.add_trace(
            go.Scatter(
                x=xs,
                y=_roll(s_col),
                mode="lines",
                name=f"{name} {viz_t('suffix_smoothed', lang)}",
                line={"width": 3, "color": color},
                customdata=ticktext,
                hovertemplate=hover_format,
            )
        )

    # Moyenne lissée de tous les joueurs (ligne neutre pointillée)
    avg_color = "rgba(255, 255, 255, 0.55)"
    hover_format_avg = f"%{{y{':' + y_format if y_format else ''}}}{y_suffix}<extra></extra>"
    fig.add_trace(
        go.Scatter(
            x=xs,
            y=_roll(avg_all),
            mode="lines",
            name=viz_t("trace_avg_3_smoothed", lang),
            line={"width": 3, "color": avg_color, "dash": "dot"},
            customdata=ticktext,
            hovertemplate=hover_format_avg,
        )
    )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 40},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
        barmode="group",
    )
    fig.update_xaxes(tickmode="array", tickvals=xs, ticktext=ticktext, title_text="")
    fig.update_yaxes(
        title_text=y_title, zeroline=True, zerolinecolor="rgba(255,255,255,0.3)", zerolinewidth=1
    )

    if y_suffix:
        fig.update_yaxes(ticksuffix=y_suffix)

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)
