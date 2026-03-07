"""Mixin — traitement des matchs depuis l'API.

Itération sur l'historique API, dispatch vers _process_known_match ou
_process_new_match, backfill sélectif dans shared_matches.duckdb.
"""

from __future__ import annotations

import asyncio
import logging
from collections.abc import Callable
from dataclasses import replace as dc_replace
from datetime import datetime, timezone
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

import duckdb

from src.data.sync.migrations import BACKFILL_FLAGS
from src.data.sync.models import SyncOptions, SyncResult
from src.data.sync.transformers import (
    extract_aliases,
    extract_all_medals,
    extract_match_registry_data,
    extract_participants,
    extract_personal_score_awards,
    extract_xuids_from_match,
    transform_all_skill_stats,
    transform_highlight_events,
    transform_match_stats,
    transform_personal_score_awards,
    transform_skill_stats,
)

logger = logging.getLogger(__name__)


class MatchProcessingMixin:
    """Méthodes de traitement des matchs (fetch, transform, insert)."""

    async def _process_matches(  # noqa: C901, PLR0912, PLR0913
        self: _SyncProtocol,
        client: Any,
        options: SyncOptions,
        existing_ids: set[str],
        *,
        delta_mode: bool,
        progress_callback: Callable[[int, int], None] | None = None,
    ) -> SyncResult:
        """Traite les matchs depuis l'API."""
        result = SyncResult()
        result.started_at = datetime.now(timezone.utc)

        start = 0
        remaining = options.max_matches
        semaphore = asyncio.Semaphore(options.parallel_matches)

        while remaining > 0:
            # Récupérer un batch d'historique
            batch_size = min(25, remaining)

            history = await client.get_match_history(
                self._gamertag,
                match_type=options.match_type,
                start=start,
                count=batch_size,
            )

            if not history:
                break

            # Traiter les matchs
            for item in history:
                if remaining <= 0:
                    break

                match_id = item.match_id

                # Vérifier si le match existe déjà
                if match_id in existing_ids:
                    if delta_mode:
                        logger.info("[DELTA] Match %s déjà connu — arrêt", match_id)
                        return result
                    else:
                        result.matches_skipped += 1
                        remaining -= 1
                        start += 1
                        continue

                # Récupérer et traiter le match
                async with semaphore:
                    match_result = await self._process_single_match(
                        client,
                        match_id,
                        options,
                    )

                if match_result.get("inserted"):
                    result.matches_inserted += 1
                    result.highlight_events_inserted += match_result.get("events", 0)
                    result.skill_records_inserted += match_result.get("skill", 0)
                    result.aliases_updated += match_result.get("aliases", 0)
                    existing_ids.add(match_id)

                    # Sprint 6 : Commit intermédiaire tous les N matchs
                    if (
                        options.batch_commit_size > 0
                        and result.matches_inserted % options.batch_commit_size == 0
                    ):
                        conn = self._get_connection()
                        conn.commit()
                        logger.debug(
                            "Commit intermédiaire après %s matchs", result.matches_inserted
                        )

                if match_result.get("error"):
                    result.warnings.append(match_result["error"])

                remaining -= 1
                start += 1

                # Callback de progression
                if progress_callback:
                    progress_callback(
                        options.max_matches - remaining,
                        options.max_matches,
                    )

                # Log de progression
                if result.matches_inserted > 0 and result.matches_inserted % 10 == 0:
                    logger.info("Importé %s matchs...", result.matches_inserted)

            # Fin du batch
            if len(history) < batch_size:
                break

        return result

    async def _process_single_match(
        self: _SyncProtocol,
        client: Any,
        match_id: str,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un match unique : fetch, transform, insert.

        Si shared_matches est activé, délègue à _process_known_match()
        ou _process_new_match() selon que le match existe déjà dans le
        registre partagé.
        """
        # ── Mode shared v5 ─────────────────────────────────────────
        shared_conn = self._get_shared_connection()
        if shared_conn is not None:
            try:
                registry = shared_conn.execute(
                    """SELECT
                        backfill_completed,
                        participants_loaded,
                        events_loaded,
                        medals_loaded,
                        player_count
                    FROM match_registry
                    WHERE match_id = ?""",
                    (match_id,),
                ).fetchone()
            except Exception:
                registry = None

            if registry is not None:
                logger.info(
                    "Match %s déjà connu dans shared (player_count=%s)", match_id, registry[4]
                )
                return await self._process_known_match(
                    client,
                    match_id,
                    registry,
                    options,
                )
            else:
                logger.info("Nouveau match %s → sync complète vers shared", match_id)
                return await self._process_new_match(
                    client,
                    match_id,
                    options,
                )

        # ── Mode legacy v4 non supporté en v5.1 (8bis.B1) ─────────────────
        raise RuntimeError(
            f"Mode legacy v4 non supporté en v5.1 — shared_matches.duckdb requis. "
            f"Match {match_id} ne peut pas être traité sans shared DB. "
            f"Exécutez 'python scripts/migrate_to_v5.py' pour créer la DB partagée."
        )

    async def _process_known_match(
        self: _SyncProtocol,
        client: Any,
        match_id: str,
        registry: tuple,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un match déjà présent dans shared_matches (sync allégée)."""
        result = self._make_result("known_match")
        _bf_completed, participants_loaded, events_loaded, medals_loaded, _player_count = registry

        # Bloquer events si déjà chargés (économie d'API)
        fetch_options = options
        if events_loaded:
            result["api_calls_saved"] += 1
            fetch_options = dc_replace(options, with_highlight_events=False)

        try:
            stats_json, skill_json, highlight_events = await self._fetch_match_data(
                client,
                match_id,
                fetch_options,
            )
            if stats_json is None:
                result["error"] = f"Impossible de récupérer {match_id}"
                return result

            match_row = transform_match_stats(
                stats_json,
                self._xuid,
                skill_json=skill_json,
                metadata_resolver=self._metadata_resolver,
            )
            if match_row is None:
                result["error"] = f"Transformation échouée pour {match_id}"
                return result

            skill_row, personal_score_rows = self._extract_personal_data(
                stats_json,
                match_id,
                skill_json,
            )
            alias_rows = extract_aliases(stats_json) if options.with_aliases else []

            await self._write_player_enrichments(
                match_id,
                match_row,
                personal_score_rows,
                skill_row,
            )
            registry_flags = (participants_loaded, events_loaded, medals_loaded)
            await self._backfill_known_match_shared(
                match_id,
                stats_json,
                skill_json,
                highlight_events,
                alias_rows,
                registry_flags,
                options,
                result,
            )

            shared_conn = self._get_shared_connection()
            await self._try_insert_pve_stats(stats_json, match_id, shared_conn)
            result["inserted"] = True

        except Exception as e:
            result["error"] = f"Erreur traitement known {match_id}: {e}"
            logger.warning(result["error"])
        return result

    async def _backfill_known_match_shared(  # noqa: PLR0913
        self: _SyncProtocol,
        match_id: str,
        stats_json: dict,
        skill_json: dict | None,
        highlight_events: list,
        alias_rows: list,
        registry_flags: tuple[bool, bool, bool],
        options: SyncOptions,
        result: dict[str, Any],
    ) -> None:
        """Backfill sélectif des données manquantes dans shared pour un match connu."""
        participants_loaded, events_loaded, medals_loaded = registry_flags
        backfill_needed: list[str] = []
        async with self._shared_db_lock:
            shared_conn = self._get_shared_connection()
            if shared_conn is None:
                result["error"] = "shared_connection perdue"
                return

            if not participants_loaded:
                participants = extract_participants(stats_json)
                inserted = self._insert_shared_participants(shared_conn, participants)
                if inserted > 0:
                    shared_conn.execute(
                        "UPDATE match_registry SET participants_loaded = TRUE WHERE match_id = ?",
                        (match_id,),
                    )
                    backfill_needed.append("participants")
                elif participants:
                    logger.warning(
                        "_backfill_known_match_shared: 0/%d participants insérés pour "
                        "match=%s — participants_loaded NON marqué, backfill relancé au prochain sync",
                        len(participants),
                        match_id,
                    )

            if not events_loaded and highlight_events:
                event_rows = transform_highlight_events(highlight_events, match_id)
                self._insert_shared_events(shared_conn, event_rows)
                shared_conn.execute(
                    "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?",
                    (match_id,),
                )
                result["events"] = len(event_rows)
                backfill_needed.append("events")
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (BACKFILL_FLAGS["events"], match_id),
                )

            if not medals_loaded:
                medals_all = extract_all_medals(stats_json)
                self._insert_shared_medals(shared_conn, medals_all)
                shared_conn.execute(
                    "UPDATE match_registry SET medals_loaded = TRUE WHERE match_id = ?",
                    (match_id,),
                )
                backfill_needed.append("medals")

            if alias_rows:
                self._insert_shared_aliases(shared_conn, alias_rows)

            if skill_json:
                self._upsert_skill_to_shared_participants(shared_conn, match_id, skill_json)

            shared_conn.execute(
                "UPDATE match_registry "
                "SET player_count = player_count + 1, "
                "    last_updated_at = CURRENT_TIMESTAMP "
                "WHERE match_id = ?",
                (match_id,),
            )
            bf_mask = self._compute_backfill_mask(options)
            shared_conn.execute(
                "UPDATE match_registry "
                "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                "WHERE match_id = ?",
                (bf_mask, match_id),
            )
            self._update_match_participant_bits(shared_conn, match_id)

        if backfill_needed:
            logger.info("Backfill shared pour %s: %s", match_id, ", ".join(backfill_needed))

    async def _process_new_match(
        self: _SyncProtocol,
        client: Any,
        match_id: str,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un nouveau match (sync complète → shared + player DB)."""
        result = self._make_result("new_match")
        try:
            stats_json, skill_json, highlight_events = await self._fetch_match_data(
                client,
                match_id,
                options,
            )
            if stats_json is None:
                result["error"] = f"Impossible de récupérer {match_id}"
                return result

            registry_data = extract_match_registry_data(
                stats_json,
                metadata_resolver=self._metadata_resolver,
            )
            if registry_data is None:
                result["error"] = f"Extraction registry échouée pour {match_id}"
                return result

            participants = extract_participants(stats_json)
            medals_all = extract_all_medals(stats_json)
            alias_rows = extract_aliases(stats_json) if options.with_aliases else []
            event_rows = (
                transform_highlight_events(highlight_events, match_id) if highlight_events else []
            )

            await self._insert_new_match_shared(
                match_id,
                registry_data,
                participants,
                medals_all,
                event_rows,
                alias_rows,
                options,
                result,
            )

            shared_conn = self._get_shared_connection()
            await self._try_insert_pve_stats(stats_json, match_id, shared_conn)

            match_row = transform_match_stats(
                stats_json,
                self._xuid,
                skill_json=skill_json,
                metadata_resolver=self._metadata_resolver,
            )
            if match_row is None:
                result["error"] = f"Transformation match_stats échouée pour {match_id}"
                return result

            skill_row, personal_score_rows = self._extract_personal_data(
                stats_json,
                match_id,
                skill_json,
            )
            await self._upsert_single_player_skill_to_shared(match_id, skill_row)
            await self._write_player_enrichments(
                match_id,
                match_row,
                personal_score_rows,
                skill_row,
            )
            result["inserted"] = True

        except Exception as e:
            result["error"] = f"Erreur traitement new {match_id}: {e}"
            logger.warning(result["error"])
        return result

    async def _insert_new_match_shared(  # noqa: PLR0913
        self: _SyncProtocol,
        match_id: str,
        registry_data: Any,
        participants: list,
        medals_all: list,
        event_rows: list,
        alias_rows: list,
        options: SyncOptions,
        result: dict[str, Any],
    ) -> None:
        """Insère toutes les données communes d'un nouveau match dans shared."""
        async with self._shared_db_lock:
            shared_conn = self._get_shared_connection()
            if shared_conn is None:
                result["error"] = "shared_connection indisponible"
                return

            self._insert_shared_registry(shared_conn, registry_data)
            inserted = self._insert_shared_participants(shared_conn, participants)
            if inserted == 0 and participants:
                logger.warning(
                    "_insert_new_match_shared: 0/%d participants insérés pour match=%s "
                    "— participants_loaded reste FALSE",
                    len(participants),
                    match_id,
                )
            self._insert_shared_medals(shared_conn, medals_all)

            if event_rows:
                self._insert_shared_events(shared_conn, event_rows)
                result["events"] = len(event_rows)

            if alias_rows:
                self._insert_shared_aliases(shared_conn, alias_rows)
                result["aliases"] = len(alias_rows)

            _utc_now = datetime.now(timezone.utc).replace(tzinfo=None)
            shared_conn.execute(
                """UPDATE match_registry SET
                    participants_loaded = ?,
                    events_loaded = ?,
                    medals_loaded = TRUE,
                    first_sync_by = ?,
                    first_sync_at = ?,
                    player_count = 1
                WHERE match_id = ?""",
                (inserted > 0, len(event_rows) > 0, self._gamertag, _utc_now, match_id),
            )
            bf_mask = self._compute_backfill_mask(options)
            shared_conn.execute(
                "UPDATE match_registry "
                "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                "WHERE match_id = ?",
                (bf_mask, match_id),
            )
            self._update_match_participant_bits(shared_conn, match_id)

    # ── Helpers extraits de _process_known/new_match ──────────────────────

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
