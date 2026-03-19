"""Options et résultats de synchronisation extraits de models.py."""

from __future__ import annotations

import logging
from dataclasses import dataclass, field, replace
from datetime import datetime
from typing import Any

_logger = logging.getLogger(__name__)


@dataclass
class SyncOptions:
    """Options de synchronisation.

    Attributes:
        match_type: Type de matchs (all, matchmaking, custom, local).
        max_matches: Nombre maximum de matchs à récupérer.
        with_highlight_events: Récupérer les highlight events (kills/deaths).
        with_skill: Récupérer les données MMR/skill.
        with_aliases: Mettre à jour les aliases XUID → Gamertag.
        with_participants: Extraire les participants de chaque match (roster complet).
        with_assets: Récupérer les assets (maps, playlists).
        requests_per_second: Rate limiting API (requêtes/seconde).
        parallel_matches: Nombre de matchs traités en parallèle.
        defer_performance_score: Différer le calcul du score de performance en batch post-sync.
        batch_commit_size: Taille du batch de commits (-1 = auto, 0 = commit final uniquement, >0 = fixe).
    """

    match_type: str = "matchmaking"
    max_matches: int = 200
    with_highlight_events: bool = True
    with_skill: bool = True
    with_aliases: bool = True
    with_participants: bool = True  # Sprint Gamertag Roster Fix
    with_assets: bool = True
    with_career_rank: bool = True  # Sync progression de rang carrière
    requests_per_second: int = 15  # Sprint 6: augmenté de 5→10, benchmark: 15 optimal
    parallel_matches: int = 10  # Sprint 6: augmenté de 3→5, benchmark: 10 optimal
    defer_performance_score: bool = True  # Sprint 6: calcul batch post-sync
    batch_commit_size: int = -1  # -1 = auto (adaptatif), 0 = commit final, >0 = fixe
    with_weapons: bool = True  # v5.5 : extraire kills par arme depuis films SPNKr

    @staticmethod
    def compute_optimal_batch_size(max_matches: int) -> int:
        """Calcule le batch_commit_size optimal selon le volume attendu.

        Plus le volume est grand, plus on espace les commits pour réduire
        le nombre d'opérations fsync sur la DB joueur.
        """
        if max_matches <= 25:
            return 0  # commit final uniquement
        if max_matches <= 100:
            return 25
        if max_matches <= 500:
            return 50
        return 100

    def with_resolved_batch_size(self) -> SyncOptions:
        """Retourne une copie avec batch_commit_size résolu si auto (-1)."""
        if self.batch_commit_size != -1:
            return self
        resolved = self.compute_optimal_batch_size(self.max_matches)
        _logger.debug("batch_commit_size auto → %d (max_matches=%d)", resolved, self.max_matches)
        return replace(self, batch_commit_size=resolved)


@dataclass
class SyncResult:
    """Résultat d'une synchronisation.

    Contient les compteurs et erreurs pour le rapport final.
    """

    matches_inserted: int = 0
    matches_updated: int = 0
    matches_skipped: int = 0
    highlight_events_inserted: int = 0
    skill_records_inserted: int = 0
    aliases_updated: int = 0
    assets_imported: int = 0
    weapon_kills_inserted: int = 0
    inserted_match_ids: list[str] = field(default_factory=list)
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    duration_seconds: float = 0.0
    started_at: datetime | None = None
    finished_at: datetime | None = None

    @property
    def success(self) -> bool:
        """True si la sync a réussi (même partiellement)."""
        # Succès si au moins un match inséré ou aucune erreur fatale
        return self.matches_inserted > 0 or len(self.errors) == 0

    @property
    def total_matches_processed(self) -> int:
        """Nombre total de matchs traités (insérés + mis à jour + skippés)."""
        return self.matches_inserted + self.matches_updated + self.matches_skipped

    def to_message(self) -> str:
        """Message de résumé pour l'UI."""
        if not self.success:
            error_preview = ", ".join(self.errors[:2])
            return f"❌ Sync échouée: {error_preview}"

        parts = []
        if self.matches_inserted > 0:
            parts.append(f"{self.matches_inserted} nouveaux matchs")
        if self.matches_updated > 0:
            parts.append(f"{self.matches_updated} mis à jour")
        if self.highlight_events_inserted > 0:
            parts.append(f"{self.highlight_events_inserted} events")
        if self.aliases_updated > 0:
            parts.append(f"{self.aliases_updated} aliases")
        if self.weapon_kills_inserted > 0:
            parts.append(f"{self.weapon_kills_inserted} kills/arme")

        if not parts:
            parts.append("Déjà à jour")

        duration_str = ""
        if self.duration_seconds > 0:
            duration_str = f" ({self.duration_seconds:.1f}s)"

        return f"✅ {', '.join(parts)}{duration_str}"

    def to_dict(self) -> dict[str, Any]:
        """Convertit en dict pour sérialisation JSON."""
        return {
            "success": self.success,
            "matches_inserted": self.matches_inserted,
            "matches_updated": self.matches_updated,
            "matches_skipped": self.matches_skipped,
            "highlight_events_inserted": self.highlight_events_inserted,
            "skill_records_inserted": self.skill_records_inserted,
            "aliases_updated": self.aliases_updated,
            "assets_imported": self.assets_imported,
            "weapon_kills_inserted": self.weapon_kills_inserted,
            "errors": self.errors,
            "warnings": self.warnings,
            "duration_seconds": self.duration_seconds,
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "finished_at": self.finished_at.isoformat() if self.finished_at else None,
        }
