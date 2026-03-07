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
    "force_skill": "skill",
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
    "force_teammates_sig": "teammates_sig",
    # ── Granulaires v5.2 ──
    "force_team_mmr": "team_mmr",
    "force_kills_expected": "kills_expected",
    "force_deaths_expected": "deaths_expected",
    "force_assists_expected": "assists_expected",
    "force_avg_life": "avg_life",
    "force_damage": "damage",
    "force_grenade_kills": "grenade_kills",
    "force_melee_kills": "melee_kills",
    "force_power_weapon_kills": "power_weapon_kills",
    "force_headshot_kills": "headshot_kills",
    "force_max_spree": "max_spree",
    "force_kda_recalc": "kda_recalc",
    "force_time_played": "time_played",
    "force_mmr": "mmr",
    "force_expected": "expected",
    "force_combat": "combat",
    "force_kills_detail": "kills_detail",
    "force_core_stats": "core_stats",
    # ── PVE v5.2 ──
    "force_pve_stats": "pve_stats",
    # ── LUSR / CSR / Skill Rank v5.3 ──
    "force_lusr": "lusr",
    "force_csr": "csr",
    "force_skill_rank": "skill_rank",
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
    "teammates_sig",
    # ── Granulaires v5.2 ──
    "team_mmr",
    "kills_expected",
    "deaths_expected",
    "assists_expected",
    "avg_life",
    "damage",
    "grenade_kills",
    "melee_kills",
    "power_weapon_kills",
    "headshot_kills",
    "max_spree",
    "kda_recalc",
    "time_played",
    "mmr",
    "expected",
    "combat",
    "kills_detail",
    "core_stats",
    # ── PVE v5.2 ──
    "pve_stats",
    # ── LUSR / CSR / Skill Rank v5.3 ──
    "lusr",
    "csr",
    "skill_rank",
    # Note : fetch_csr est un snapshot one-shot, non activé par --all-data
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
    # ── PVE stats (v5.2) ──
    "pve_stats": "pve_stats",
    # ── LUSR / CSR (v5.3) — bitmask backfill_completed sur match_registry ──
    "lusr": "lusr",
    "csr": "csr",
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
    teammates_sig: bool = False  # Backfiller teammates_signature + is_with_friends depuis shared

    # ── Types granulaires v5.2 (nouveaux champs bitmask) ─────────────────
    # Skill / MMR
    team_mmr: bool = False  # Backfill team_mmr si NULL
    # enemy_mmr conservé (legacy) — active les deux via --mmr
    kills_expected: bool = False  # Backfill kills_expected/stddev
    deaths_expected: bool = False  # Backfill deaths_expected/stddev
    assists_expected: bool = False  # Backfill assists_expected/stddev

    # Combat
    damage: bool = False  # Backfill damage_dealt/damage_taken
    avg_life: bool = False  # Backfill avg_life_seconds

    # Kills détaillés
    grenade_kills: bool = False
    melee_kills: bool = False
    power_weapon_kills: bool = False
    headshot_kills: bool = False
    max_spree: bool = False  # Backfill max_killing_spree

    # Divers
    kda_recalc: bool = False  # Recalculer kda si NULL
    time_played: bool = False  # Backfill time_played_seconds

    # Groupes (alias résolus dans resolve())
    mmr: bool = False  # = team_mmr + enemy_mmr
    expected: bool = False  # = kills_expected + deaths_expected + assists_expected
    combat: bool = False  # = accuracy + shots + damage
    kills_detail: bool = (
        False  # = grenade_kills + melee_kills + power_weapon_kills + headshot_kills
    )
    core_stats: bool = False  # = combat + avg_life + kills_detail + kda_recalc + time_played

    # ── Flags force ──────────────────────────────────────────────────────
    # Quand force_X = True, le backfill re-traite TOUS les matchs pour
    # le type X, même ceux déjà peuplés. Utile après correction de bugs.
    force_medals: bool = False
    force_skill: bool = False  # Ajouté v5.1 : force rescan MMR + expected/stddev
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
    force_teammates_sig: bool = False

    # ── Flags force granulaires v5.2 ─────────────────────────────────────
    force_team_mmr: bool = False
    force_kills_expected: bool = False
    force_deaths_expected: bool = False
    force_assists_expected: bool = False
    force_avg_life: bool = False
    force_damage: bool = False
    force_grenade_kills: bool = False
    force_melee_kills: bool = False
    force_power_weapon_kills: bool = False
    force_headshot_kills: bool = False
    force_max_spree: bool = False
    force_kda_recalc: bool = False
    force_time_played: bool = False
    force_mmr: bool = False
    force_expected: bool = False
    force_combat: bool = False
    force_kills_detail: bool = False
    force_core_stats: bool = False

    # ── PVE (Firefight) — v5.2 ───────────────────────────────────────────
    pve_stats: bool = False  # Stats PVE → shared_pve.duckdb

    # ── Flags force PVE ──────────────────────────────────────────────────
    force_pve_stats: bool = False  # Re-traiter même si MatchBits.PVE_STATS déjà posé

    # ── LUSR / CSR / Skill Rank — v5.3 ──────────────────────────────
    lusr: bool = False  # Calculer le LUSR pour les matchs non classés (local-only)
    force_lusr: bool = False  # Recalculer le LUSR depuis zéro
    csr: bool = False  # Backfill CSR depuis l'API pour les matchs classés
    force_csr: bool = False  # Forcer le re-fetch CSR pour tous les matchs classés
    fetch_csr: bool = False  # Snapshot CSR actuel via get_playlist_csr → skill_history
    skill_rank: bool = False  # Alias unifié : lusr=True + csr=True (v5.3)
    force_skill_rank: bool = False  # force_lusr=True + force_csr=True (v5.3)

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

    def resolve(self) -> None:  # noqa: C901
        """Applique les implications ``all_data`` → champs et ``force_X`` → X.

        Doit être appelé une seule fois après construction.

        Ordre critique des groupes v5.2 (extérieur → intérieur) :
        1. core_stats  → combat, kills_detail, avg_life, kda_recalc, time_played
        2. combat      → accuracy, shots, damage
        3. kills_detail → grenade_kills, melee_kills, power_weapon_kills, headshot_kills
        4. mmr         → team_mmr, enemy_mmr
        5. expected    → kills_expected, deaths_expected, assists_expected
        6. _FORCE_MAP  (toujours en dernier)
        """
        # all_data active tous les champs data
        if self.all_data:
            for field_name in _ALL_DATA_FIELDS:
                setattr(self, field_name, True)

        # ── 1. Groupes de haut niveau ──
        if self.core_stats:
            self.combat = True
            self.avg_life = True
            self.kills_detail = True
            self.kda_recalc = True
            self.time_played = True

        # ── 2. Groupes intermédiaires ──
        if self.combat:
            self.accuracy = True
            self.shots = True
            self.damage = True

        if self.kills_detail:
            self.grenade_kills = True
            self.melee_kills = True
            self.power_weapon_kills = True
            self.headshot_kills = True

        # ── 3. Groupes skill ──
        if self.mmr:
            self.team_mmr = True
            self.enemy_mmr = True

        if self.expected:
            self.kills_expected = True
            self.deaths_expected = True
            self.assists_expected = True

        # --skill active aussi les champs granulaires (rétrocompatibilité)
        if self.skill:
            self.team_mmr = True
            self.enemy_mmr = True
            self.kills_expected = True
            self.deaths_expected = True
            self.assists_expected = True

        # ── 3b. skill_rank = lusr + csr ──
        if self.skill_rank:
            self.lusr = True
            self.csr = True
        if self.force_skill_rank:
            self.force_lusr = True
            self.force_csr = True

        # ── 4. force_X implique X (toujours en dernier) ──
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
            "csr",  # re-fetch CSR nécessite l'API skill
            "fetch_csr",  # snapshot CSR via get_playlist_csr
            "skill_rank",  # skill_rank implique csr (côté API)
        }
        return any(getattr(self, f) for f in api_fields)

    @property
    def needs_local_only(self) -> bool:
        """True si des traitements locaux (sans API) sont demandés."""
        return any(
            getattr(self, f)
            for f in ("killer_victim", "end_time", "sessions", "citations", "lusr", "skill_rank")
        )

    @property
    def requested_types(self) -> list[str]:
        """Liste des types demandés pour le bitmask ``backfill_completed``."""
        return [
            type_key
            for field_name, type_key in _REQUESTED_TYPE_MAP.items()
            if getattr(self, field_name, False)
        ]
