"""Périmètre de données pour sync et backfill — ``SyncScope``.

Ce module centralise **tous** les flags de données dans une unique
dataclass partagée par ``sync.py``, ``backfill_data.py`` et leurs
sous-modules (orchestrator, detection, engine).

Ajouter un nouveau type de donnée se résume à :

1. Ajouter le champ booléen ici (+ ``force_*`` si pertinent).
2. Mettre à jour ``_ALL_DATA_FIELDS``, ``_FORCE_MAP``, ``_REQUESTED_TYPE_MAP``.
3. Ajouter l'argument CLI dans ``scripts/backfill/cli.py``.
4. Implémenter la logique métier dans l'orchestrateur / engine.

Le « plumbing » (CLI ↔ orchestrateur ↔ détection ↔ API) est géré
automatiquement grâce à cette dataclass.
"""

from __future__ import annotations

from dataclasses import dataclass, fields

# ─────────────────────────────────────────────────────────────────────────────
# Registre : chaque champ ``force_X`` implique le champ ``X``.
# ─────────────────────────────────────────────────────────────────────────────
_FORCE_MAP: dict[str, str] = {
    "force_medals": "medals",
    "force_accuracy": "accuracy",
    "force_shots": "shots",
    "force_performance_scores": "performance_scores",
    "force_participants_shots": "participants_shots",
    "force_participants_damage": "participants_damage",
    "force_participants_avg_life": "participants_avg_life",
    "force_enemy_mmr": "enemy_mmr",
    "force_aliases": "aliases",
    "force_assets": "assets",
    "force_participants": "participants",
    "force_end_time": "end_time",
    "force_sessions": "sessions",
    "force_citations": "citations",
    "force_participants_enrich": "participants_enrich",
}

# Champs activés par ``all_data``.  Listés explicitement pour ne pas
# activer accidentellement des options « admin-only » ajoutées plus tard.
_ALL_DATA_FIELDS: tuple[str, ...] = (
    "medals",
    "events",
    "skill",
    "personal_scores",
    "performance_scores",
    "aliases",
    "accuracy",
    "enemy_mmr",
    "assets",
    "participants",
    "shots",
    "participants_scores",
    "participants_kda",
    "participants_shots",
    "participants_damage",
    "participants_avg_life",
    "killer_victim",
    "end_time",
    "sessions",
    "citations",
    "participants_enrich",
)

# Mapping champ → clé pour ``requested_types`` (bitmask backfill_completed).
# Seuls les types présents ici participent au bitmask.
_REQUESTED_TYPE_MAP: dict[str, str] = {
    "medals": "medals",
    "events": "events",
    "skill": "skill",
    "personal_scores": "personal_scores",
    "performance_scores": "performance_scores",
    "aliases": "aliases",
    "accuracy": "accuracy",
    "shots": "shots",
    "enemy_mmr": "enemy_mmr",
    "assets": "assets",
    "participants": "participants",
    "participants_scores": "participants_scores",
    "participants_kda": "participants_kda",
    "participants_shots": "participants_shots",
    "participants_damage": "participants_damage",
    "participants_avg_life": "participants_avg_life",
}


@dataclass
class SyncScope:
    """Périmètre de données partagé entre sync et backfill.

    Décrit *quels types de données* doivent être traités, que ce soit
    lors d'une synchronisation (nouveaux matchs) ou d'un backfill
    (matchs existants avec données manquantes).

    Pour ajouter un nouveau type de données :
    1. Ajouter un champ booléen ci-dessous (+ ``force_*`` si pertinent).
    2. L'ajouter dans ``_ALL_DATA_FIELDS`` si ``--all-data`` doit l'activer.
    3. L'ajouter dans ``_FORCE_MAP`` si un ``--force-*`` existe.
    4. L'ajouter dans ``_REQUESTED_TYPE_MAP`` si un bitmask est nécessaire.
    """

    # ── Général ──────────────────────────────────────────────────────────
    dry_run: bool = False
    max_matches: int | None = None
    requests_per_second: int = 5
    detection_mode: str = "or"

    # ── Types de données ─────────────────────────────────────────────────
    medals: bool = False
    events: bool = False
    skill: bool = False
    personal_scores: bool = False
    performance_scores: bool = False
    aliases: bool = False
    accuracy: bool = False
    enemy_mmr: bool = False
    assets: bool = False
    participants: bool = False
    participants_scores: bool = False
    participants_kda: bool = False
    participants_shots: bool = False
    participants_damage: bool = False
    participants_avg_life: bool = False
    killer_victim: bool = False
    end_time: bool = False
    sessions: bool = False
    shots: bool = False
    citations: bool = False
    participants_enrich: bool = False

    # ── Flags force ──────────────────────────────────────────────────────
    force_medals: bool = False
    force_accuracy: bool = False
    force_shots: bool = False
    force_performance_scores: bool = False
    force_participants_shots: bool = False
    force_participants_damage: bool = False
    force_participants_avg_life: bool = False
    force_enemy_mmr: bool = False
    force_aliases: bool = False
    force_assets: bool = False
    force_participants: bool = False
    force_end_time: bool = False
    force_sessions: bool = False
    force_citations: bool = False
    force_participants_enrich: bool = False

    # ── Méta-flag ────────────────────────────────────────────────────────
    all_data: bool = False

    # ─────────────────────────────────────────────────────────────────────
    # Constructeurs alternatifs
    # ─────────────────────────────────────────────────────────────────────

    @classmethod
    def from_cli_args(cls, args: object) -> SyncScope:
        """Construit depuis un namespace argparse.

        Seuls les attributs existant à la fois dans *args* et dans la
        dataclass sont copiés.  ``resolve()`` est appelé automatiquement.
        """
        kwargs: dict = {}
        for f in fields(cls):
            if hasattr(args, f.name):
                kwargs[f.name] = getattr(args, f.name)
        scope = cls(**kwargs)
        scope.resolve()
        return scope

    @classmethod
    def make_all(cls, **overrides: object) -> SyncScope:
        """Crée un scope avec ``all_data=True`` pré-résolu."""
        scope = cls(all_data=True, **overrides)
        scope.resolve()
        return scope

    # ─────────────────────────────────────────────────────────────────────
    # Résolution automatique
    # ─────────────────────────────────────────────────────────────────────

    def resolve(self) -> None:
        """Applique les implications ``all_data`` → champs et ``force_X`` → X.

        Doit être appelé une seule fois après construction.
        """
        # all_data active tous les champs data
        if self.all_data:
            for field_name in _ALL_DATA_FIELDS:
                setattr(self, field_name, True)

        # force_X implique X
        for force_field, data_field in _FORCE_MAP.items():
            if getattr(self, force_field, False) and not getattr(self, data_field, False):
                setattr(self, data_field, True)

    # ─────────────────────────────────────────────────────────────────────
    # Propriétés utilitaires
    # ─────────────────────────────────────────────────────────────────────

    def has_any_option(self) -> bool:
        """True si au moins un type de données est activé."""
        return any(getattr(self, f) for f in _ALL_DATA_FIELDS)

    @property
    def needs_api(self) -> bool:
        """True si au moins un type nécessite un appel API."""
        api_fields = {
            "medals",
            "events",
            "skill",
            "personal_scores",
            "performance_scores",
            "aliases",
            "accuracy",
            "enemy_mmr",
            "assets",
            "participants",
            "shots",
            "participants_scores",
            "participants_kda",
            "participants_shots",
            "participants_damage",
            "participants_avg_life",
            "participants_enrich",
        }
        return any(getattr(self, f) for f in api_fields)

    @property
    def needs_local_only(self) -> bool:
        """True si des traitements locaux (sans API) sont demandés."""
        return any(getattr(self, f) for f in ("killer_victim", "end_time", "sessions", "citations"))

    @property
    def requested_types(self) -> list[str]:
        """Liste des types demandés pour le bitmask ``backfill_completed``."""
        return [
            type_key
            for field_name, type_key in _REQUESTED_TYPE_MAP.items()
            if getattr(self, field_name, False)
        ]
