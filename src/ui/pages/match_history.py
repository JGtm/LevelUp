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

import polars as pl
import streamlit as st

from src.analysis.performance_score import compute_performance_series
from src.visualization._compat import DataFrameLike, ensure_polars


def _app_url(page: str, **params: str) -> str:
    """Génère une URL interne vers une page de l'app."""
    import urllib.parse

    base = "/"
    qp = {"page": page, **params}
    return base + "?" + urllib.parse.urlencode(qp)


def render_match_history_page(
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
        st.warning("Aucun match à afficher. Vérifiez vos filtres ou synchronisez les données.")
        return

    st.subheader("Historique des parties")

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
    outcome_map = {2: "Victoire", 3: "Défaite", 1: "Égalité", 4: "Non terminé"}
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
        pl.col("start_time").dt.strftime("%d/%m/%Y %H:%M").fill_null("-").alias("start_time_fr")
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


def _render_history_table(dff_table: pl.DataFrame) -> None:
    """Affiche le tableau de l'historique via `st.dataframe` modernisé."""
    view = dff_table.sort("start_time", descending=True).head(250)

    view = view.with_columns(
        (pl.lit("/?page=Match&match_id=") + pl.col("match_id").cast(pl.Utf8).fill_null(pl.lit("")))
        .alias("match_link")
        .str.replace(r"\s+", "", literal=False)
    )

    display = view.select(
        [
            pl.col("match_link").alias("Match"),
            pl.col("match_url").alias("HaloWaypoint"),
            pl.col("start_time_fr").alias("Date de début"),
            pl.col("map_name").fill_null("-").alias("Carte"),
            pl.col("playlist_fr").fill_null("-").alias("Playlist"),
            pl.col("mode_ui").fill_null("-").alias("Mode"),
            pl.col("outcome_label").fill_null("-").alias("Résultat"),
            pl.col("score").fill_null("-").alias("Score"),
            pl.col("performance_display").fill_null("-").alias("Performance"),
            pl.col("team_mmr")
            .cast(pl.Float64, strict=False)
            .round(0)
            .cast(pl.Int64, strict=False)
            .cast(pl.Utf8)
            .fill_null("-")
            .alias("MMR équipe"),
            pl.col("enemy_mmr")
            .cast(pl.Float64, strict=False)
            .round(0)
            .cast(pl.Int64, strict=False)
            .cast(pl.Utf8)
            .fill_null("-")
            .alias("MMR adverse"),
            pl.col("delta_mmr")
            .cast(pl.Float64, strict=False)
            .round(0)
            .cast(pl.Int64, strict=False)
            .cast(pl.Utf8)
            .fill_null("-")
            .alias("Écart MMR"),
            pl.col("kda").cast(pl.Utf8).fill_null("-").alias("FDA"),
            pl.col("kills").cast(pl.Utf8).fill_null("-").alias("Frags"),
            pl.col("deaths").cast(pl.Utf8).fill_null("-").alias("Morts"),
            pl.col("max_killing_spree").cast(pl.Utf8).fill_null("-").alias("Tuerie (max)"),
            pl.col("headshot_kills").cast(pl.Utf8).fill_null("-").alias("Têtes"),
            pl.col("average_life_mmss").fill_null("-").alias("Durée vie"),
            pl.col("assists").cast(pl.Utf8).fill_null("-").alias("Assistances"),
            pl.col("accuracy").cast(pl.Utf8).fill_null("-").alias("Précision"),
            pl.col("ratio").cast(pl.Utf8).fill_null("-").alias("Ratio"),
        ]
    )

    st.dataframe(
        display,
        width="stretch",
        hide_index=True,
        column_config={
            "Match": st.column_config.LinkColumn("Match", display_text="Ouvrir"),
            "HaloWaypoint": st.column_config.LinkColumn("HaloWaypoint", display_text="Ouvrir"),
        },
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

    csv_table = table.rename({"start_time_fr": "Date de début"})
    csv = csv_table.write_csv().encode("utf-8")
    st.download_button(
        "Télécharger CSV",
        data=csv,
        file_name="openspartan_matches.csv",
        mime="text/csv",
    )
