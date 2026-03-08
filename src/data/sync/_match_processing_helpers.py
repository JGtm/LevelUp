"""Helpers du mixin de traitement des matchs.

Fonctions utilitaires extraites de MatchProcessingMixin pour garder
la taille du module ≤ 500L (règle project).
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

import duckdb

from src.data.sync.migrations import BACKFILL_FLAGS
from src.data.sync.models import SyncOptions
from src.data.sync.transformers import (
    extract_personal_score_awards,
    extract_xuids_from_match,
    transform_all_skill_stats,
    transform_personal_score_awards,
    transform_skill_stats,
)

logger = logging.getLogger(__name__)


class MatchProcessingHelpersMixin:
    """Helpers de fetch/transform/write extraits de MatchProcessingMixin."""

    @staticmethod
    def _make_result(mode: str) -> dict[str, Any]:
        """Crée un dict résultat standard pour les fonctions de traitement."""
        return {
            "inserted": False,
            "mode": mode,
            "events": 0,
            "skill": 0,
            "aliases": 0,
            "api_calls_saved": 0,
            "error": None,
        }

    async def _fetch_match_data(
        self: _SyncProtocol,
        client: Any,
        match_id: str,
        options: SyncOptions,
    ) -> tuple[dict | None, dict | None, list]:
        """Télécharge stats, skill et events pour un match."""
        stats_json = await client.get_match_stats(match_id)
        if stats_json is None:
            return None, None, []

        if options.with_assets:
            from src.data.sync.api_client import enrich_match_info_with_assets

            await enrich_match_info_with_assets(client, stats_json)

        xuids = extract_xuids_from_match(stats_json)
        skill_json = None
        highlight_events: list = []

        if options.with_skill and xuids:
            skill_json = await client.get_skill_stats(match_id, xuids)
        if options.with_highlight_events:
            highlight_events = await client.get_highlight_events(match_id)

        return stats_json, skill_json, highlight_events

    def _upsert_skill_to_shared_participants(
        self: _SyncProtocol,
        shared_conn: duckdb.DuckDBPyConnection,
        match_id: str,
        skill_json: dict,
    ) -> None:
        """Écrit team_mmr/enemy_mmr + expected/stddev dans shared.match_participants."""
        all_skill_updates = transform_all_skill_stats(skill_json, match_id)
        for su in all_skill_updates:
            if su.team_mmr is not None or su.enemy_mmr is not None:
                shared_conn.execute(
                    "UPDATE match_participants SET "
                    "team_mmr = COALESCE(?, team_mmr), "
                    "enemy_mmr = COALESCE(?, enemy_mmr), "
                    "kills_expected = COALESCE(?, kills_expected), "
                    "kills_stddev = COALESCE(?, kills_stddev), "
                    "deaths_expected = COALESCE(?, deaths_expected), "
                    "deaths_stddev = COALESCE(?, deaths_stddev), "
                    "assists_expected = COALESCE(?, assists_expected), "
                    "assists_stddev = COALESCE(?, assists_stddev) "
                    "WHERE match_id = ? AND xuid = ?",
                    (
                        su.team_mmr,
                        su.enemy_mmr,
                        su.kills_expected,
                        su.kills_stddev,
                        su.deaths_expected,
                        su.deaths_stddev,
                        su.assists_expected,
                        su.assists_stddev,
                        match_id,
                        su.xuid,
                    ),
                )

    async def _upsert_single_player_skill_to_shared(
        self: _SyncProtocol,
        match_id: str,
        skill_row: Any,
    ) -> None:
        """Écrit team_mmr/enemy_mmr pour un seul joueur dans shared.match_participants."""
        if not skill_row or (skill_row.team_mmr is None and skill_row.enemy_mmr is None):
            return
        async with self._shared_db_lock:
            shared_conn = self._get_shared_connection()
            if shared_conn:
                shared_conn.execute(
                    "UPDATE match_participants SET "
                    "team_mmr = COALESCE(?, team_mmr), "
                    "enemy_mmr = COALESCE(?, enemy_mmr), "
                    "kills_expected = COALESCE(?, kills_expected), "
                    "kills_stddev = COALESCE(?, kills_stddev), "
                    "deaths_expected = COALESCE(?, deaths_expected), "
                    "deaths_stddev = COALESCE(?, deaths_stddev), "
                    "assists_expected = COALESCE(?, assists_expected), "
                    "assists_stddev = COALESCE(?, assists_stddev) "
                    "WHERE match_id = ? AND xuid = ?",
                    (
                        skill_row.team_mmr,
                        skill_row.enemy_mmr,
                        skill_row.kills_expected,
                        skill_row.kills_stddev,
                        skill_row.deaths_expected,
                        skill_row.deaths_stddev,
                        skill_row.assists_expected,
                        skill_row.assists_stddev,
                        match_id,
                        self._xuid,
                    ),
                )

    async def _write_player_enrichments(
        self: _SyncProtocol,
        match_id: str,
        match_row: Any,
        personal_score_rows: list,
        skill_row: Any,
    ) -> None:
        """Écrit les enrichissements personnels dans la player DB."""
        async with self._db_lock:
            if personal_score_rows:
                self._insert_personal_score_rows(personal_score_rows)
            self._insert_enrichment_row(match_id, match_row)
            self._update_performance_score(match_id, match_row)
            if skill_row is not None and match_row.is_ranked:
                self._upsert_csr_rating(match_id, skill_row)

    def _extract_personal_data(
        self: _SyncProtocol,
        stats_json: dict,
        match_id: str,
        skill_json: dict | None,
    ) -> tuple[Any, list]:
        """Extrait skill_row et personal_score_rows pour le joueur courant."""
        skill_row = None
        if skill_json:
            skill_row = transform_skill_stats(skill_json, match_id, self._xuid)
        personal_scores = extract_personal_score_awards(stats_json, self._xuid)
        personal_score_rows = []
        if personal_scores:
            personal_score_rows = transform_personal_score_awards(
                match_id,
                self._xuid,
                personal_scores,
            )
        return skill_row, personal_score_rows

    async def _try_insert_pve_stats(
        self: _SyncProtocol,
        stats_json: dict,
        match_id: str,
        shared_conn: duckdb.DuckDBPyConnection | None,
    ) -> None:
        """Extrait et insère les stats PVE dans shared_pve.duckdb si c'est un Firefight.

        Pose ``MatchBits.PVE_STATS`` dans ``match_registry.backfill_completed`` même si
        le match n'est pas un Firefight (guard pour éviter la re-détection infinie).

        Args:
            stats_json: JSON brut du match (get_match_stats).
            match_id: ID du match.
            shared_conn: Connexion vers shared_matches.duckdb (pour le bitmask guard).
        """
        from src.data.sync.batch_insert import batch_insert_pve_stats
        from src.data.sync.constants import MatchBits
        from src.data.sync.transformers import extract_pve_stats

        try:
            pve_rows = extract_pve_stats(stats_json)
            async with self._pve_db_lock:
                pve_conn = self._get_pve_connection()
                if pve_rows:
                    inserted = batch_insert_pve_stats(pve_conn, pve_rows)
                    logger.debug("PVE stats insérées pour %s: %s lignes", match_id, inserted)

            # Poser le guard dans match_registry.backfill_completed (même si 0 lignes)
            if shared_conn is not None:
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (MatchBits.PVE_STATS, match_id),
                )
        except Exception as e:
            logger.debug("PVE stats non insérées pour %s: %s", match_id, e)

    def _compute_backfill_mask(self, options: SyncOptions) -> int:
        """Calcule le bitmask backfill_completed pour un match.

        Args:
            options: Options de sync courantes.

        Returns:
            Bitmask entier.
        """
        bf_mask = 0
        bf_mask |= BACKFILL_FLAGS["medals"]
        bf_mask |= BACKFILL_FLAGS["personal_scores"]
        bf_mask |= BACKFILL_FLAGS["performance_scores"]
        bf_mask |= BACKFILL_FLAGS["accuracy"]
        bf_mask |= BACKFILL_FLAGS["shots"]
        if options.with_skill:
            bf_mask |= BACKFILL_FLAGS["skill"]
            bf_mask |= BACKFILL_FLAGS["enemy_mmr"]
        # Fix v5.4: le bit "events" n'est plus posé ici de façon inconditionnelle.
        # Il est posé uniquement quand des events sont réellement insérés
        # (cf. bloc events dans _sync_single_match_shared), pour éviter que
        # backfill_completed & 2 soit posé sur des matchs où l'API retourne [].
        if options.with_participants:
            bf_mask |= BACKFILL_FLAGS["participants"]
            bf_mask |= BACKFILL_FLAGS["participants_scores"]
            bf_mask |= BACKFILL_FLAGS["participants_kda"]
            bf_mask |= BACKFILL_FLAGS["participants_shots"]
            bf_mask |= BACKFILL_FLAGS["participants_damage"]
            bf_mask |= BACKFILL_FLAGS["participants_avg_life"]
        if options.with_aliases:
            bf_mask |= BACKFILL_FLAGS["aliases"]
        if options.with_assets:
            bf_mask |= BACKFILL_FLAGS["assets"]
        return bf_mask
