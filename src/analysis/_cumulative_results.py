"""Dataclasses de résultats pour l'analyse cumulative."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class CumulativeSeriesResult:
    """Résultat d'une série cumulative.

    Attributes:
        match_id: ID du match.
        start_time: Timestamp du match.
        value: Valeur pour ce match.
        cumulative: Valeur cumulative jusqu'à ce match.
    """

    match_id: str
    start_time: str
    value: float
    cumulative: float


@dataclass(frozen=True)
class CumulativeMetricsResult:
    """Métriques cumulées pour une session.

    Attributes:
        total_kills: Total des kills.
        total_deaths: Total des morts.
        total_assists: Total des assistances.
        cumulative_net_score: Net score cumulé final.
        cumulative_kd: K/D cumulé final.
        cumulative_kda: KDA cumulé final.
        matches_count: Nombre de matchs.
    """

    total_kills: int
    total_deaths: int
    total_assists: int
    cumulative_net_score: int
    cumulative_kd: float
    cumulative_kda: float
    matches_count: int

    @property
    def average_kills_per_match(self) -> float:
        """Kills moyens par match."""
        if self.matches_count == 0:
            return 0.0
        return self.total_kills / self.matches_count

    @property
    def average_deaths_per_match(self) -> float:
        """Morts moyennes par match."""
        if self.matches_count == 0:
            return 0.0
        return self.total_deaths / self.matches_count
