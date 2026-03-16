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


from src.data.sync._match_processing_helpers import MatchProcessingHelpersMixin
from src.data.sync.migrations import BACKFILL_FLAGS
from src.data.sync.models import SyncOptions, SyncResult
from src.data.sync.transformers import (
    extract_aliases,
    extract_all_medals,
    extract_match_registry_data,
    extract_participants,
    transform_highlight_events,
    transform_match_stats,
)

logger = logging.getLogger(__name__)


class MatchProcessingMixin(MatchProcessingHelpersMixin):
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
                    self._accumulate_match_result(result, match_result)
                    existing_ids.add(match_id)
                    self._maybe_batch_commit(result.matches_inserted, options.batch_commit_size)

                if match_result.get("error"):
                    result.warnings.append(match_result["error"])

                remaining -= 1
                start += 1
                self._report_progress(result, options, remaining, progress_callback)

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
            if options.with_weapons:
                result["weapon_kills"] = await self._try_extract_weapon_kills(
                    client, match_id, shared_conn
                )
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
                n_events = self._backfill_events_block(shared_conn, match_id, highlight_events)
                result["events"] = n_events
                backfill_needed.append("events")

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
                highlight_events,
                alias_rows,
                options,
                result,
            )

            shared_conn = self._get_shared_connection()
            await self._try_insert_pve_stats(stats_json, match_id, shared_conn)
            if options.with_weapons:
                result["weapon_kills"] = await self._try_extract_weapon_kills(
                    client,
                    match_id,
                    shared_conn,
                    is_firefight=bool(getattr(registry_data, "is_firefight", False)),
                )

            ok = await self._save_player_data_new_match(match_id, stats_json, skill_json, result)
            if not ok:
                return result
            result["inserted"] = True

        except Exception as e:
            result["error"] = f"Erreur traitement new {match_id}: {e}"
            logger.warning(result["error"])
        return result

    async def _save_player_data_new_match(
        self: _SyncProtocol,
        match_id: str,
        stats_json: dict,
        skill_json: dict | None,
        result: dict[str, Any],
    ) -> bool:
        """Transforme et persiste les données joueur pour un nouveau match."""
        match_row = transform_match_stats(
            stats_json,
            self._xuid,
            skill_json=skill_json,
            metadata_resolver=self._metadata_resolver,
        )
        if match_row is None:
            result["error"] = f"Transformation match_stats échouée pour {match_id}"
            return False
        skill_row, personal_score_rows = self._extract_personal_data(
            stats_json, match_id, skill_json
        )
        await self._upsert_skill_new_match(match_id, skill_json, skill_row)
        await self._write_player_enrichments(match_id, match_row, personal_score_rows, skill_row)
        return True

    async def _insert_new_match_shared(  # noqa: PLR0913
        self: _SyncProtocol,
        match_id: str,
        registry_data: Any,
        participants: list,
        medals_all: list,
        event_rows: list,
        highlight_events: list,
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

            n_events = 0
            if event_rows:
                n_events = self._insert_shared_events(shared_conn, event_rows)
                result["events"] = n_events
                if n_events > 0:
                    self._insert_shared_killer_victim_pairs(
                        shared_conn,
                        match_id,
                        highlight_events,
                    )
                    shared_conn.execute(
                        "UPDATE match_registry "
                        "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                        "WHERE match_id = ?",
                        (BACKFILL_FLAGS["events"], match_id),
                    )

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
                (inserted > 0, n_events > 0, self._gamertag, _utc_now, match_id),
            )
            bf_mask = self._compute_backfill_mask(options)
            shared_conn.execute(
                "UPDATE match_registry "
                "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                "WHERE match_id = ?",
                (bf_mask, match_id),
            )
            self._update_match_participant_bits(shared_conn, match_id)
