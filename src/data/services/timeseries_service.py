"""Service Séries Temporelles — agrégats pour la page timeseries.

Encapsule tous les calculs lourds : performance cumulative, distributions,
corrélations, et accès DB (first kill/death, perfect kills).

Contrat : les pages UI appellent ces fonctions, jamais de calcul inline.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass

import polars as pl

from src.config import CORE_STAT_COLUMNS
from src.data.domain.refdata import Outcome

logger = logging.getLogger(__name__)

# ─── Dataclasses retour ────────────────────────────────────────────────


@dataclass(frozen=True)
class PerformanceData:
    """Données de performance enrichies (score calculé)."""

    dff: pl.DataFrame
    """DataFrame Polars filtré avec colonne performance_score ajoutée."""


@dataclass(frozen=True)
class CumulativeMetrics:
    """Métriques cumulatives (net score, K/D, rolling K/D)."""

    cumul_net: pl.DataFrame
    """Net score cumulé."""
    cumul_kd: pl.DataFrame
    """K/D cumulé."""
    rolling_kd: pl.DataFrame
    """K/D glissant."""
    time_played_seconds: list[int | float] | None
    """Durées de jeu par match (pour marqueurs)."""
    has_enough_for_trend: bool
    """True si >= 4 matchs pour la tendance session."""
    pl_df: pl.DataFrame
    """DataFrame Polars trié pour graphes de tendance."""


@dataclass(frozen=True)
class ScorePerMinuteData:
    """Distribution du score personnel par minute."""

    values: pl.Series
    """Série Polars des valeurs score/min."""
    has_data: bool
    """True si assez de données (> 5)."""


@dataclass(frozen=True)
class RollingWinRateData:
    """Distribution du taux de victoire glissant."""

    values: pl.Series
    """Série Polars des taux de victoire glissants."""
    has_data: bool
    """True si assez de données (> 5)."""
    missing_column: bool
    """True si colonne outcome manquante."""
    not_enough_matches: bool
    """True si < 10 matchs."""


@dataclass(frozen=True)
class FirstEventData:
    """Timestamps du premier frag / première mort par match."""

    first_kills: dict[str, int | None]
    first_deaths: dict[str, int | None]
    available: bool
    """True si au moins un événement trouvé."""


@dataclass(frozen=True)
class PerfectKillsData:
    """Nombre de frags parfaits par match."""

    counts: dict[str, int] | None
    """Mapping match_id → nombre de perfect kills, ou None."""


# ─── Service ───────────────────────────────────────────────────────────


class TimeseriesService:
    """Service d'agrégation pour la page Séries Temporelles.

    Encapsule les calculs lourds et accès DB. Les pages UI n'ont plus
    qu'à récupérer des dataclasses typées et les afficher.
    """

    @staticmethod
    def enrich_performance_score(
        dff: pl.DataFrame,
        df_full: pl.DataFrame | None = None,
    ) -> pl.DataFrame:
        """Ajoute la colonne performance_score au DataFrame si absente.

        Args:
            dff: DataFrame Polars filtré des matchs.
            df_full: DataFrame complet pour le calcul relatif.

        Returns:
            DataFrame Polars enrichi avec performance_score.
        """
        from src.analysis.performance_score import compute_performance_series

        history_df = df_full if df_full is not None else dff
        if "performance_score" not in dff.columns:
            dff = dff.clone()
            dff = dff.with_columns(
                pl.Series("performance_score", compute_performance_series(dff, history_df))
            )
        return dff

    @staticmethod
    def compute_cumulative_metrics(dff: pl.DataFrame) -> CumulativeMetrics | None:
        """Calcule les métriques cumulatives (net score, K/D, rolling K/D).

        Args:
            dff: DataFrame Polars filtré trié par start_time, avec kills/deaths.

        Returns:
            CumulativeMetrics ou None si colonnes manquantes.
        """
        from src.analysis.cumulative import (
            compute_cumulative_kd_series_polars,
            compute_cumulative_net_score_series_polars,
            compute_rolling_kd_polars,
        )

        _required = CORE_STAT_COLUMNS
        if not all(c in dff.columns for c in _required):
            return None

        _cumul_cols = list(_required)
        _has_tps = "time_played_seconds" in dff.columns
        if _has_tps:
            _cumul_cols.append("time_played_seconds")

        pl_df = dff.select(_cumul_cols).sort("start_time")

        if pl_df.is_empty():
            return None

        _tps_list: list[int | float] | None = None
        if _has_tps:
            _tps_list = pl_df["time_played_seconds"].fill_null(0).to_list()

        cumul_net = compute_cumulative_net_score_series_polars(pl_df)
        cumul_kd = compute_cumulative_kd_series_polars(pl_df)
        rolling_kd = compute_rolling_kd_polars(pl_df, window_size=5)

        return CumulativeMetrics(
            cumul_net=cumul_net,
            cumul_kd=cumul_kd,
            rolling_kd=rolling_kd,
            time_played_seconds=_tps_list,
            has_enough_for_trend=len(pl_df) >= 4,
            pl_df=pl_df,
        )

    @staticmethod
    def compute_score_per_minute(dff: pl.DataFrame) -> ScorePerMinuteData:
        """Calcule la distribution du score personnel par minute.

        Args:
            dff: DataFrame Polars filtré avec personal_score et time_played_seconds.

        Returns:
            ScorePerMinuteData avec les valeurs.
        """
        if "personal_score" not in dff.columns or "time_played_seconds" not in dff.columns:
            return ScorePerMinuteData(values=pl.Series(dtype=pl.Float64), has_data=False)

        _ps = dff.select(["personal_score", "time_played_seconds"]).drop_nulls()
        _ps = _ps.filter(pl.col("time_played_seconds") > 0)

        if len(_ps) <= 5:
            return ScorePerMinuteData(values=pl.Series(dtype=pl.Float64), has_data=False)

        values = _ps["personal_score"] / (_ps["time_played_seconds"] / 60)
        return ScorePerMinuteData(values=values, has_data=True)

    @staticmethod
    def compute_rolling_win_rate(dff: pl.DataFrame) -> RollingWinRateData:
        """Calcule la distribution du taux de victoire glissant (fenêtre 5).

        Args:
            dff: DataFrame Polars filtré avec outcome et start_time.

        Returns:
            RollingWinRateData avec les valeurs.
        """
        if "outcome" not in dff.columns:
            return RollingWinRateData(
                values=pl.Series(dtype=pl.Float64),
                has_data=False,
                missing_column=True,
                not_enough_matches=False,
            )

        if len(dff) < 5:
            return RollingWinRateData(
                values=pl.Series(dtype=pl.Float64),
                has_data=False,
                missing_column=False,
                not_enough_matches=True,
            )

        _wr_df = dff.sort("start_time") if "start_time" in dff.columns else dff
        _wins = (_wr_df["outcome"] == Outcome.WIN).cast(pl.Float64)
        win_rate_rolling = _wins.rolling_mean(window_size=5, min_samples=5) * 100
        win_rate_clean = win_rate_rolling.drop_nulls()

        return RollingWinRateData(
            values=win_rate_clean,
            has_data=len(win_rate_clean) > 5,
            missing_column=False,
            not_enough_matches=False,
        )

    @staticmethod
    def load_first_event_times(
        db_path: str | None,
        xuid: str | None,
        match_ids: list[str],
    ) -> FirstEventData:
        """Charge les timestamps du premier frag / première mort depuis la DB.

        Args:
            db_path: Chemin vers la DB DuckDB.
            xuid: XUID du joueur.
            match_ids: Liste des match_id à interroger.

        Returns:
            FirstEventData avec les dictionnaires d'événements.
        """
        first_kills: dict[str, int | None] = {}
        first_deaths: dict[str, int | None] = {}

        if not db_path or not xuid or not match_ids:
            return FirstEventData(first_kills={}, first_deaths={}, available=False)

        try:
            from src.data.repositories.duckdb_repo import DuckDBRepository

            if db_path.endswith(".duckdb"):
                repo = DuckDBRepository(db_path, str(xuid).strip())
                first_kills, first_deaths = repo.get_first_kill_death_times(match_ids)
        except Exception:
            pass

        available = bool(first_kills or first_deaths)
        return FirstEventData(
            first_kills=first_kills,
            first_deaths=first_deaths,
            available=available,
        )

    @staticmethod
    def load_perfect_kills(
        db_path: str | None,
        xuid: str | None,
        match_ids: list[str],
    ) -> PerfectKillsData:
        """Charge le nombre de frags parfaits par match depuis la DB.

        Args:
            db_path: Chemin vers la DB DuckDB.
            xuid: XUID du joueur.
            match_ids: Liste des match_id.

        Returns:
            PerfectKillsData avec le mapping.
        """
        if not db_path or not xuid or not match_ids:
            return PerfectKillsData(counts=None)

        try:
            from src.data.repositories.duckdb_repo import DuckDBRepository

            if db_path.endswith(".duckdb"):
                repo = DuckDBRepository(db_path, str(xuid).strip())
                counts = repo.count_perfect_kills_by_match(match_ids)
                return PerfectKillsData(counts=counts)
        except Exception:
            pass
        return PerfectKillsData(counts=None)

    # =========================================================================
    # Progression avancée (Phase refonte Progression)
    # =========================================================================

    @staticmethod
    def compute_ewma_kd(
        dff: pl.DataFrame,
        alpha: float = 0.2,
    ) -> pl.DataFrame:
        """Calcule le K/D lissé EWMA. Délègue à cumulative_progression.

        Args:
            dff: DataFrame des matchs.
            alpha: Facteur de lissage (0 = très lisse, 0.5 = réactif).

        Returns:
            DataFrame avec kd, ewma_kd, outcome (si disponible).
        """
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        return compute_ewma_kd_polars(dff, alpha=alpha)

    @staticmethod
    def compute_cumulative_kd_with_ci(
        dff: pl.DataFrame,
        z: float = 1.645,
    ) -> pl.DataFrame:
        """Calcule le K/D cumulé + IC 90 %. Délègue à cumulative_progression.

        Args:
            dff: DataFrame des matchs.
            z: Z-score (1.645 = 90 %, 1.96 = 95 %).

        Returns:
            DataFrame avec cumulative_kd, ci_lower, ci_upper.
        """
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        return compute_cumulative_kd_with_ci(dff, z=z)

    @staticmethod
    def compute_linear_regression_kd(
        ewma_df: pl.DataFrame,
        kd_col: str = "ewma_kd",
    ) -> dict:
        """Calcule régression linéaire sur K/D + win rate. Délègue à cumulative_progression.

        Args:
            ewma_df: DataFrame avec ewma_kd (et outcome si disponible).
            kd_col: Colonne K/D à régresser.

        Returns:
            Dict avec slope, r_squared, trend, is_significant, win_rate_slope…
        """
        from src.analysis.cumulative_progression import compute_linear_regression_kd

        return compute_linear_regression_kd(ewma_df, kd_col=kd_col)

    @staticmethod
    def compute_rolling_net_score_per_hour(
        dff: pl.DataFrame,
        window_size: int | None = None,
    ) -> pl.DataFrame | None:
        """Calcule le net score/h rolling. Retourne None si time_played_seconds absent.

        Args:
            dff: DataFrame des matchs.
            window_size: Taille de fenêtre (None = adaptatif).

        Returns:
            DataFrame ou None si données insuffisantes.
        """
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        result = compute_rolling_net_score_per_hour(dff, window_size=window_size)
        return result if not result.is_empty() else None
