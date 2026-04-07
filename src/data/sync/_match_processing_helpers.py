"""Helpers du mixin de traitement des matchs.

Fonctions utilitaires extraites de MatchProcessingMixin pour garder
la taille du module ≤ 500L (règle project).
"""

from __future__ import annotations

import asyncio
import functools
import logging
from collections.abc import Callable
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

import duckdb

from src.data.sync.migrations import BACKFILL_FLAGS
from src.data.sync.models import SyncOptions, SyncResult
from src.data.sync.transformers import (
    extract_personal_score_awards,
    extract_xuids_from_match,
    transform_all_skill_stats,
    transform_match_stats,
    transform_personal_score_awards,
    transform_skill_stats,
)

logger = logging.getLogger(__name__)


class MatchProcessingHelpersMixin:
    """Helpers de fetch/transform/write extraits de MatchProcessingMixin."""

    async def _transform_match_stats_async(
        self: _SyncProtocol,
        stats_json: dict,
        skill_json: dict | None,
    ) -> Any:
        """Axe 5 : exécute transform_match_stats dans un thread (CPU-bound)."""
        loop = asyncio.get_running_loop()
        fn = functools.partial(
            transform_match_stats,
            stats_json,
            self._xuid,
            skill_json=skill_json,
            metadata_resolver=self._metadata_resolver,
        )
        return await loop.run_in_executor(None, fn)

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
            "weapon_kills": 0,
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
    ) -> list:
        """Écrit team_mmr/enemy_mmr + expected/stddev dans shared.match_participants.

        Returns:
            La liste des SkillParticipantUpdate parsés (réutilisable sans re-parse).
        """
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

    async def _upsert_skill_new_match(
        self: _SyncProtocol,
        match_id: str,
        skill_json: dict | None,
        skill_row: Any,
    ) -> None:
        """Écrit les données skill dans shared.match_participants pour un nouveau match.

        Si skill_json est disponible, met à jour TOUS les participants (team_mmr,
        enemy_mmr, expected/stddev). Sinon, met à jour uniquement le joueur courant.
        Cela corrige le bug où seul le premier joueur à syncer obtenait son MMR.

        Collecte également les CSR des coéquipiers depuis la liste pré-parsée,
        évitant un double appel à transform_all_skill_stats.
        """
        if skill_json:
            async with self._shared_db_lock:
                shared_conn = self._get_shared_connection()
                if shared_conn:
                    skill_updates = self._upsert_skill_to_shared_participants(
                        shared_conn, match_id, skill_json
                    )
                    self._collect_csr_for_other_players(skill_updates)
        else:
            await self._upsert_single_player_skill_to_shared(match_id, skill_row)

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

    def _collect_psa_for_other_players(
        self: _SyncProtocol,
        stats_json: dict,
        match_id: str,
    ) -> None:
        """Extrait et met en attente les PSA des coéquipiers enregistrés.

        Le JSON stats_json contient les PersonalScores de TOUS les participants.
        Ici on les extrait pour chaque autre joueur enregistré dans db_profiles.json
        et on les stocke dans self._pending_other_psa pour distribution
        ultérieure par le fanout (_run_other_player_enrichment).

        Appelé pendant le traitement de chaque match (known et new), pendant
        que stats_json est disponible en mémoire.

        Args:
            stats_json: JSON brut du match (get_match_stats).
            match_id: ID du match en cours.
        """
        others = self._get_other_registered_players()
        for player in others:
            xuid = player.get("xuid", "")
            if not xuid:
                continue
            personal_scores = extract_personal_score_awards(stats_json, xuid)
            if not personal_scores:
                continue
            rows = transform_personal_score_awards(match_id, xuid, personal_scores)
            if rows:
                if xuid not in self._pending_other_psa:
                    self._pending_other_psa[xuid] = []
                self._pending_other_psa[xuid].extend(rows)
                logger.debug(
                    "psa_fanout: %d PSA en attente pour xuid=%s match=%s",
                    len(rows),
                    xuid,
                    match_id,
                )

    def _collect_csr_for_other_players(
        self: _SyncProtocol,
        all_skill_updates: list,
    ) -> None:
        """Collecte les données CSR des coéquipiers enregistrés depuis la liste pré-parsée.

        Reçoit la liste déjà parsée par _upsert_skill_to_shared_participants
        (évite un double appel à transform_all_skill_stats par match).
        Stocke les CSR dans self._pending_other_csr pour distribution par le fanout.

        Args:
            all_skill_updates: Liste de SkillParticipantUpdate déjà parsés.
        """
        csr_updates = [u for u in all_skill_updates if u.post_match_csr is not None]
        if not csr_updates:
            return

        my_xuid = self._xuid or ""
        others = {p["xuid"] for p in self._get_other_registered_players() if p.get("xuid")}

        for update in csr_updates:
            if update.xuid == my_xuid or update.xuid not in others:
                continue
            bucket = self._pending_other_csr.setdefault(update.xuid, [])
            bucket.append(update)
            logger.debug(
                "csr_fanout: CSR en attente pour xuid=%s match=%s (csr=%.0f)",
                update.xuid,
                update.match_id,
                update.post_match_csr,
            )

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
            shared_conn: Connexion vers shared_matches_v2.duckdb (pour le bitmask guard).
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
        # performance_scores supprimé : granularité joueur×match, incompatible avec
        # un bit par match dans match_registry.backfill_completed (cf. plan E.5).
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

    def _report_progress(
        self: _SyncProtocol,
        result: SyncResult,
        options: SyncOptions,
        remaining: int,
        progress_callback: Callable[[int, int], None] | None,
    ) -> None:
        """Appelle le callback de progression et logue si milestone atteint."""
        if progress_callback:
            progress_callback(options.max_matches - remaining, options.max_matches)
        if result.matches_inserted > 0 and result.matches_inserted % 10 == 0:
            logger.info("Importé %s matchs...", result.matches_inserted)

    def _accumulate_match_result(
        self: _SyncProtocol,
        result: SyncResult,
        match_result: dict[str, Any],
    ) -> None:
        """Accumule les compteurs d'un match traité dans result."""
        result.matches_inserted += 1
        result.highlight_events_inserted += match_result.get("events", 0)
        result.skill_records_inserted += match_result.get("skill", 0)
        result.aliases_updated += match_result.get("aliases", 0)
        result.weapon_kills_inserted += match_result.get("weapon_kills", 0)
        if mid := match_result.get("match_id"):
            result.inserted_match_ids.append(str(mid))

    def _maybe_batch_commit(self: _SyncProtocol, n_inserted: int, batch_size: int) -> None:
        """Commit le batch courant et ouvre une nouvelle transaction (player + shared).

        Phase H : repurposé comme gestionnaire transactionnel. Commit player DB
        puis shared DB, puis rouvre une transaction sur shared pour le prochain batch.
        """
        if batch_size > 0 and n_inserted % batch_size == 0:
            self._get_connection().commit()
            _stc = self._get_shared_connection()
            if _stc is not None:
                _stc.execute("COMMIT")
                _stc.execute("BEGIN TRANSACTION")
            logger.debug("event=batch_commit count=%d", n_inserted)
