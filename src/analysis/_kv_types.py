"""Types et helpers pour l'analyse killer→victim.

Dataclasses, TypedDict et fonctions de coercition utilisées par le module
killer_victim et ses sous-modules.
"""

from __future__ import annotations

from collections import Counter
from dataclasses import dataclass
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from typing import TypedDict

    class MatchPlayerStats(TypedDict, total=False):
        xuid: str
        gamertag: str
        kills: int
        deaths: int
        assists: int
        team_id: int | None
        rank: int
        score: int | None


# =============================================================================
# Dataclasses
# =============================================================================


@dataclass(frozen=True)
class KVPair:
    killer_xuid: str
    killer_gamertag: str
    victim_xuid: str
    victim_gamertag: str
    time_ms: int


@dataclass(frozen=True)
class EstimatedCount:
    """Compteur avec séparation certain/estimé.

    Note: "estimé" signifie qu'il y avait ambiguïté (plusieurs candidats).
    """

    certain: int = 0
    estimated: int = 0

    @property
    def total(self) -> int:
        return int(self.certain) + int(self.estimated)

    @property
    def has_estimated(self) -> bool:
        return int(self.estimated) > 0


@dataclass(frozen=True)
class OpponentDuel:
    """Résumé d'un duel moi <-> adversaire."""

    xuid: str
    gamertag: str
    opponent_killed_me: EstimatedCount
    me_killed_opponent: EstimatedCount


@dataclass(frozen=True)
class ValidationResult:
    """Résultat de validation des paires killer→victim.

    Contient les écarts entre les totaux reconstitués et officiels.
    """

    xuid: str
    kills_reconstituted: int
    kills_official: int
    deaths_reconstituted: int
    deaths_official: int

    @property
    def kills_diff(self) -> int:
        """Écart entre kills reconstitués et officiels."""
        return self.kills_reconstituted - self.kills_official

    @property
    def deaths_diff(self) -> int:
        """Écart entre deaths reconstituées et officielles."""
        return self.deaths_reconstituted - self.deaths_official

    @property
    def is_consistent(self) -> bool:
        """Retourne True si les totaux sont cohérents."""
        return self.kills_diff == 0 and self.deaths_diff == 0


@dataclass(frozen=True)
class AntagonistsResult:
    """Résultat Némésis / Souffre-douleur pour un match."""

    nemesis: OpponentDuel | None
    bully: OpponentDuel | None
    my_deaths_total: int
    my_deaths_assigned_certain: int
    my_deaths_assigned_total: int
    my_kills_total: int
    my_kills_assigned_certain: int
    my_kills_assigned_total: int
    # Sprint 3.1: Ajout du flag de confiance
    is_validated: bool = False
    validation_notes: str = ""


# =============================================================================
# Helpers de coercition
# =============================================================================


def _coerce_int(v: Any) -> int | None:
    try:
        if v is None:
            return None
        if isinstance(v, bool):
            return None
        return int(v)
    except Exception:
        return None


def _coerce_str(v: Any) -> str | None:
    if v is None:
        return None
    s = str(v).strip()
    return s or None


# =============================================================================
# Fonctions de validation
# =============================================================================


def _get_xuid(s: Any) -> str:
    """Extrait le xuid d'un objet stats (dict ou mock)."""
    return s.xuid if hasattr(s, "xuid") else s["xuid"]


def _get_stat(s: Any, key: str) -> int:
    """Extrait une stat d'un objet stats (dict ou mock)."""
    return getattr(s, key, 0) if hasattr(s, key) else s.get(key, 0)


def validate_and_adjust_pairs(
    pairs: list[KVPair],
    official_stats: list[MatchPlayerStats],
) -> tuple[list[ValidationResult], bool]:
    """Valide la cohérence des paires killer→victim avec les stats officielles.

    Args:
        pairs: Liste des paires killer→victim reconstituées.
        official_stats: Stats officielles de chaque joueur.

    Returns:
        Tuple (liste de ValidationResult, is_globally_consistent)
    """
    if not official_stats:
        return [], True

    stats_by_xuid: dict[str, MatchPlayerStats] = {_get_xuid(s): s for s in official_stats}

    kills_reconstituted: dict[str, int] = Counter()
    deaths_reconstituted: dict[str, int] = Counter()

    for p in pairs:
        if p.killer_xuid:
            kills_reconstituted[p.killer_xuid] += 1
        if p.victim_xuid:
            deaths_reconstituted[p.victim_xuid] += 1

    all_xuids = (
        set(stats_by_xuid.keys())
        | set(kills_reconstituted.keys())
        | set(deaths_reconstituted.keys())
    )

    results: list[ValidationResult] = []
    for xuid in all_xuids:
        official = stats_by_xuid.get(xuid)
        results.append(
            ValidationResult(
                xuid=xuid,
                kills_reconstituted=kills_reconstituted.get(xuid, 0),
                kills_official=official.kills if official else 0,
                deaths_reconstituted=deaths_reconstituted.get(xuid, 0),
                deaths_official=official.deaths if official else 0,
            )
        )

    is_globally_consistent = all(r.is_consistent for r in results)
    return results, is_globally_consistent


def get_player_rank(xuid: str, official_stats: list[MatchPlayerStats]) -> int:
    """Retourne le rang d'un joueur dans le match (1 = meilleur).

    Args:
        xuid: XUID du joueur.
        official_stats: Stats officielles avec rangs.

    Returns:
        Rang du joueur (1 = meilleur), ou 999 si non trouvé.
    """
    for s in official_stats:
        if _get_xuid(s) == xuid:
            return s.rank if hasattr(s, "rank") else s["rank"]
    return 999


def _infer_event_type(event: dict[str, Any]) -> str | None:
    """Infère le type d'événement (kill/death) depuis un event brut."""
    et = _coerce_str(event.get("event_type"))
    if et:
        return et.lower()

    th = _coerce_int(event.get("type_hint"))
    if th == 50:
        return "kill"
    if th == 20:
        return "death"
    if th == 10:
        return "mode"
    return None
