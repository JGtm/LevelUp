"""Service Victoires/Défaites — agrégats pour la page win_loss.

Encapsule les calculs lourds : bucketing temporel, breakdown par carte,
ratio global, et tableau de période.

Contrat : les pages UI appellent ces fonctions, jamais de calcul inline.
Pure Polars — aucun import pandas.
"""

from __future__ import annotations

from dataclasses import dataclass

import polars as pl

from src.data.domain.refdata import Outcome

# ─── Dataclasses retour ────────────────────────────────────────────────


@dataclass(frozen=True)
class PeriodTable:
    """Tableau de victoires/défaites par période."""

    table: pl.DataFrame
    """DataFrame Polars avec colonnes Victoires, Défaites, Égalités, etc."""
    bucket_label: str
    """Label du type de bucket (heure, jour, semaine, mois)."""
    is_empty: bool
    """True si aucune donnée."""


@dataclass(frozen=True)
class MapBreakdownResult:
    """Résultat de l'analyse par carte."""

    breakdown: pl.DataFrame
    """DataFrame Polars avec stats par carte (win_rate, loss_rate, ratio, etc.)."""
    is_empty: bool
    """True si pas assez de matchs."""


@dataclass(frozen=True)
class FriendMatchIds:
    """Résultat de recherche de matchs avec un ami."""

    match_ids: set[str]
    """Set des match_id partagés."""
    scope_df: pl.DataFrame
    """DataFrame Polars filtré sur ces matchs."""


# ─── Service ───────────────────────────────────────────────────────────


class WinLossService:
    """Service d'agrégation pour la page Victoires/Défaites.

    Encapsule les calculs de bucketing, de breakdown par carte,
    et d'accès DB (matchs partagés avec amis).
    """

    @staticmethod
    def compute_period_table(  # noqa: PLR0912
        dff: pl.DataFrame,
        bucket_label: str,
        is_session_scope: bool = False,
        lang: str = "fr",
    ) -> PeriodTable:
        """Construit le tableau de victoires/défaites par période.

        Args:
            dff: DataFrame Polars filtré des matchs.
            bucket_label: Label du bucket temporel (pour l'en-tête).
            is_session_scope: True si mode session actif.
            lang: Langue cible ("fr" ou "en").

        Returns:
            PeriodTable avec le tableau formaté.
        """
        from src.config import SESSION_CONFIG

        # Noms de colonnes selon la langue
        _col_wins = "Victoires" if lang == "fr" else "Wins"
        _col_losses = "Défaites" if lang == "fr" else "Losses"
        _col_draws = "Égalités" if lang == "fr" else "Draws"
        _col_unfinished = "Non terminés" if lang == "fr" else "Unfinished"
        _col_total = "Total"
        _col_winrate = "Taux de victoires" if lang == "fr" else "Win rate"

        if dff.is_empty() or "outcome" not in dff.columns:
            return PeriodTable(table=pl.DataFrame(), bucket_label=bucket_label, is_empty=True)

        d = dff.drop_nulls(subset=["outcome"])
        if d.is_empty():
            return PeriodTable(table=pl.DataFrame(), bucket_label=bucket_label, is_empty=True)

        # Calcul du bucket selon le scope et la plage temporelle
        d = d.sort("start_time")
        n_rows = len(d)

        if is_session_scope:
            if n_rows <= 20:
                d = d.with_row_index("bucket", offset=1)
            else:
                d = d.with_columns(pl.col("start_time").dt.truncate("1h").alias("bucket"))
        else:
            # Calcul de la plage temporelle
            start_col = d.get_column("start_time").drop_nulls()
            if start_col.is_empty():
                return PeriodTable(table=pl.DataFrame(), bucket_label=bucket_label, is_empty=True)

            tmin = start_col.min()
            tmax = start_col.max()
            if tmin is None or tmax is None:
                days = 999.0
            else:
                dt_range = tmax - tmin
                days = dt_range.total_seconds() / 86400.0 if dt_range else 0.0

            cfg = SESSION_CONFIG
            if days < cfg.bucket_threshold_hourly:
                d = d.with_row_index("bucket", offset=1)
            elif days <= cfg.bucket_threshold_daily:
                d = d.with_columns(pl.col("start_time").dt.truncate("1h").alias("bucket"))
            elif days <= cfg.bucket_threshold_weekly:
                d = d.with_columns(pl.col("start_time").dt.date().cast(pl.Utf8).alias("bucket"))
            elif days <= cfg.bucket_threshold_monthly:
                # Semaine ISO (début lundi)
                d = d.with_columns(
                    pl.col("start_time").dt.truncate("1w").dt.date().cast(pl.Utf8).alias("bucket")
                )
            else:
                # Par mois
                d = d.with_columns(pl.col("start_time").dt.strftime("%Y-%m").alias("bucket"))

        # Pivot: compter les outcomes par bucket
        pivot = (
            d.group_by("bucket")
            .agg(
                [
                    (pl.col("outcome") == Outcome.WIN).sum().alias(_col_wins),
                    (pl.col("outcome") == Outcome.LOSS).sum().alias(_col_losses),
                    (pl.col("outcome") == Outcome.TIE).sum().alias(_col_draws),
                    (pl.col("outcome") == Outcome.DID_NOT_FINISH).sum().alias(_col_unfinished),
                ]
            )
            .sort("bucket")
        )

        # Calcul du total et du taux de victoires
        out_tbl = pivot.with_columns(
            [
                (
                    pl.col(_col_wins)
                    + pl.col(_col_losses)
                    + pl.col(_col_draws)
                    + pl.col(_col_unfinished)
                ).alias(_col_total),
            ]
        ).with_columns(
            [
                pl.when(pl.col(_col_total) > 0)
                .then(100.0 * pl.col(_col_wins) / pl.col(_col_total))
                .otherwise(0.0)
                .alias(_col_winrate)
            ]
        )

        # Renommer la colonne bucket
        out_tbl = out_tbl.rename({"bucket": bucket_label.capitalize()})

        return PeriodTable(table=out_tbl, bucket_label=bucket_label, is_empty=False)

    @staticmethod
    def compute_map_breakdown(
        base_scope: pl.DataFrame,
        min_matches: int = 1,
        df_history: pl.DataFrame | None = None,
    ) -> MapBreakdownResult:
        """Calcule le breakdown statistique par carte.

        Args:
            base_scope: DataFrame Polars à analyser (session ou filtre courant).
            min_matches: Nombre minimum de matchs par carte.
            df_history: Historique complet pour le calcul du score relatif.
                Si None, utilise base_scope comme référence (moins précis pour les
                petites sessions).

        Returns:
            MapBreakdownResult avec le breakdown filtré.
        """
        from src.analysis import compute_map_breakdown

        breakdown = compute_map_breakdown(base_scope, df_history=df_history)
        breakdown = breakdown.filter(pl.col("matches") >= int(min_matches))

        return MapBreakdownResult(
            breakdown=breakdown if not breakdown.is_empty() else pl.DataFrame(),
            is_empty=breakdown.is_empty(),
        )

    @staticmethod
    def get_friend_scope_df(  # noqa: PLR0913
        scope: str,
        dff: pl.DataFrame,
        base: pl.DataFrame,
        db_path: str,
        xuid: str,
        db_key: tuple[int, int] | None,
    ) -> pl.DataFrame:
        """Retourne le DataFrame Polars filtré selon le scope (moi, ami, tous).

        Args:
            scope: Label du scope sélectionné.
            dff: DataFrame filtré courant.
            base: DataFrame de base complet.
            db_path: Chemin DB.
            xuid: XUID du joueur.
            db_key: Clé de cache DB.

        Returns:
            DataFrame Polars filtré sur le scope demandé.
        """
        from src.ui.cache import cached_same_team_match_ids_with_friend

        if scope == "Moi (toutes les parties)":
            return base
        elif scope == "Avec SpartanA":
            match_ids = set(
                cached_same_team_match_ids_with_friend(
                    db_path,
                    xuid.strip(),
                    "2533274858283686",
                    db_key=db_key,
                )
            )
            return base.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(match_ids)))
        elif scope == "Avec SpartanB":
            match_ids = set(
                cached_same_team_match_ids_with_friend(
                    db_path,
                    xuid.strip(),
                    "2535469190789936",
                    db_key=db_key,
                )
            )
            return base.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(match_ids)))
        else:
            return dff
