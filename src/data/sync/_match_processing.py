"""Mixin — traitement des matchs depuis l'API.

Itération sur l'historique API, dispatch vers _process_known_match ou
_process_new_match, backfill sélectif dans shared_matches.duckdb.
"""

from __future__ import annotations

import asyncio
import logging
from collections.abc import Callable
from datetime import datetime, timezone
from typing import Any

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

    async def _process_matches(
        self,
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
                self._gamertag,  # type: ignore[attr-defined]
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
                        logger.info(f"[DELTA] Match {match_id} déjà connu — arrêt")
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
                        conn = self._get_connection()  # type: ignore[attr-defined]
                        conn.commit()
                        logger.debug(f"Commit intermédiaire après {result.matches_inserted} matchs")

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
                    logger.info(f"Importé {result.matches_inserted} matchs...")

            # Fin du batch
            if len(history) < batch_size:
                break

        return result

    async def _process_single_match(
        self,
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
        shared_conn = self._get_shared_connection()  # type: ignore[attr-defined]
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
                    f"Match {match_id} déjà connu dans shared " f"(player_count={registry[4]})"
                )
                return await self._process_known_match(
                    client,
                    match_id,
                    registry,
                    options,
                )
            else:
                logger.info(f"Nouveau match {match_id} → sync complète vers shared")
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
        self,
        client: Any,
        match_id: str,
        registry: tuple,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un match déjà présent dans shared_matches (sync allégée).

        Seules les données personnelles (match_stats, enrichment) sont
        insérées dans la DB joueur. Les données communes manquantes sont
        backfillées dans shared si nécessaire.

        Args:
            client: Client API SPNKr.
            match_id: ID du match.
            registry: Tuple (backfill_completed, participants_loaded,
                      events_loaded, medals_loaded, player_count).
            options: Options de sync.

        Returns:
            Dict résultat avec mode='known_match'.
        """
        result: dict[str, Any] = {
            "inserted": False,
            "mode": "known_match",
            "events": 0,
            "skill": 0,
            "aliases": 0,
            "api_calls_saved": 0,
            "error": None,
        }

        _bf_completed, participants_loaded, events_loaded, medals_loaded, _player_count = registry

        try:
            # 1. Télécharger les stats (obligatoire pour extraire les données perso)
            stats_json = await client.get_match_stats(match_id)
            if stats_json is None:
                result["error"] = f"Impossible de récupérer {match_id}"
                return result

            if options.with_assets:
                from src.data.sync.api_client import enrich_match_info_with_assets

                await enrich_match_info_with_assets(client, stats_json)

            # 2. Transformer en match_row pour la player DB (mode legacy)
            xuids = extract_xuids_from_match(stats_json)
            skill_json = None
            highlight_events: list = []

            # Skill toujours utile pour le joueur (MMR dans match_stats)
            if options.with_skill and xuids:
                skill_json = await client.get_skill_stats(match_id, xuids)

            # Events : seulement si absent du shared
            if options.with_highlight_events and not events_loaded:
                highlight_events = await client.get_highlight_events(match_id)
            elif events_loaded:
                result["api_calls_saved"] += 1

            match_row = transform_match_stats(
                stats_json,
                self._xuid,  # type: ignore[attr-defined]
                skill_json=skill_json,
                metadata_resolver=self._metadata_resolver,  # type: ignore[attr-defined]
            )
            if match_row is None:
                result["error"] = f"Transformation échouée pour {match_id}"
                return result

            _skill_row = None
            if skill_json:
                _skill_row = transform_skill_stats(skill_json, match_id, self._xuid)  # type: ignore[attr-defined]

            # Extraire les médailles perso (player DB)
            from src.data.sync.transformers import extract_medals

            _medal_rows = extract_medals(stats_json, self._xuid)  # type: ignore[attr-defined]

            # PersonalScores
            personal_scores = extract_personal_score_awards(stats_json, self._xuid)  # type: ignore[attr-defined]
            personal_score_rows = []
            if personal_scores:
                personal_score_rows = transform_personal_score_awards(
                    match_id,
                    self._xuid,  # type: ignore[attr-defined]
                    personal_scores,
                )

            alias_rows = []
            if options.with_aliases:
                alias_rows = extract_aliases(stats_json)

            _participant_rows = []
            if options.with_participants:
                _participant_rows = extract_participants(stats_json)

            # 3. V5 finale : Insérer minimalement dans la player DB (seulement enrichissements)
            async with self._db_lock:  # type: ignore[attr-defined]
                # ✅ CONSERVÉ : personal_score_awards (enrichissement personnel)
                if personal_score_rows:
                    self._insert_personal_score_rows(personal_score_rows)  # type: ignore[attr-defined]

                # ✅ NOUVEAU : player_match_enrichment (performance_score, sessions, etc.)
                self._insert_enrichment_row(match_id, match_row)  # type: ignore[attr-defined]
                self._compute_and_update_performance_score(match_id, match_row)

                # ✅ CSR (v5.2) : écrire le CSR dans match_skill_rank si match classé
                if _skill_row is not None and match_row.is_ranked:
                    self._upsert_csr_rating(match_id, _skill_row)

            # 4. Backfill sélectif dans shared si des données manquent
            backfill_needed: list[str] = []
            async with self._shared_db_lock:  # type: ignore[attr-defined]
                shared_conn = self._get_shared_connection()  # type: ignore[attr-defined]
                if shared_conn is None:
                    result["error"] = "shared_connection perdue"
                    return result

                if not participants_loaded:
                    participants = extract_participants(stats_json)
                    self._insert_shared_participants(shared_conn, participants)
                    shared_conn.execute(
                        "UPDATE match_registry SET participants_loaded = TRUE WHERE match_id = ?",
                        (match_id,),
                    )
                    backfill_needed.append("participants")

                if not events_loaded and highlight_events:
                    event_rows_shared = transform_highlight_events(highlight_events, match_id)
                    self._insert_shared_events(shared_conn, event_rows_shared)
                    shared_conn.execute(
                        "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?",
                        (match_id,),
                    )
                    result["events"] = len(event_rows_shared)
                    backfill_needed.append("events")
                    # Fix v5.4: poser le bit ici (insertion réelle) plutôt que dans
                    # _compute_backfill_mask (qui le posait même sans insertion effective)
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

                # Aliases vers shared
                if alias_rows:
                    self._insert_shared_aliases(shared_conn, alias_rows)

                # ✅ Écrire team_mmr/enemy_mmr + expected/stddev dans match_participants (V5.1)
                if skill_json:
                    all_skill_updates = transform_all_skill_stats(skill_json, match_id)
                    for skill_update in all_skill_updates:
                        if skill_update.team_mmr is not None or skill_update.enemy_mmr is not None:
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
                                    skill_update.team_mmr,
                                    skill_update.enemy_mmr,
                                    skill_update.kills_expected,
                                    skill_update.kills_stddev,
                                    skill_update.deaths_expected,
                                    skill_update.deaths_stddev,
                                    skill_update.assists_expected,
                                    skill_update.assists_stddev,
                                    match_id,
                                    skill_update.xuid,
                                ),
                            )

                # Incrémenter player_count
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET player_count = player_count + 1, "
                    "    last_updated_at = CURRENT_TIMESTAMP "
                    "WHERE match_id = ?",
                    (match_id,),
                )

                # Marquer le bitmask backfill_completed avec les types syncés
                bf_mask = self._compute_backfill_mask(options)
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (bf_mask, match_id),
                )

                # Mettre à jour backfill_bits par participant (v5.2)
                self._update_match_participant_bits(shared_conn, match_id)

            # Stats PVE → shared_pve.duckdb (Firefight uniquement) — v5.2
            await self._try_insert_pve_stats(stats_json, match_id, shared_conn)

            if backfill_needed:
                logger.info(f"Backfill shared pour {match_id}: {', '.join(backfill_needed)}")

            result["inserted"] = True

        except Exception as e:
            result["error"] = f"Erreur traitement known {match_id}: {e}"
            logger.warning(result["error"])

        return result

    async def _process_new_match(
        self,
        client: Any,
        match_id: str,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un nouveau match (sync complète → shared + player DB).

        Toutes les données communes sont insérées dans shared_matches,
        les données personnelles dans la player DB.

        Args:
            client: Client API SPNKr.
            match_id: ID du match.
            options: Options de sync.

        Returns:
            Dict résultat avec mode='new_match'.
        """
        result: dict[str, Any] = {
            "inserted": False,
            "mode": "new_match",
            "events": 0,
            "skill": 0,
            "aliases": 0,
            "error": None,
        }

        try:
            # 1. Télécharger les stats
            stats_json = await client.get_match_stats(match_id)
            if stats_json is None:
                result["error"] = f"Impossible de récupérer {match_id}"
                return result

            if options.with_assets:
                from src.data.sync.api_client import enrich_match_info_with_assets

                await enrich_match_info_with_assets(client, stats_json)

            # 2. Télécharger events et skill
            xuids = extract_xuids_from_match(stats_json)
            skill_json = None
            highlight_events: list = []

            if options.with_skill and xuids:
                skill_json = await client.get_skill_stats(match_id, xuids)

            if options.with_highlight_events:
                highlight_events = await client.get_highlight_events(match_id)

            # 3. Extraire les données communes pour shared
            registry_data = extract_match_registry_data(
                stats_json,
                metadata_resolver=self._metadata_resolver,  # type: ignore[attr-defined]
            )
            if registry_data is None:
                result["error"] = f"Extraction registry échouée pour {match_id}"
                return result

            participants = extract_participants(stats_json)
            medals_all = extract_all_medals(stats_json)
            alias_rows = extract_aliases(stats_json) if options.with_aliases else []

            event_rows_shared = []
            if highlight_events:
                event_rows_shared = transform_highlight_events(highlight_events, match_id)

            # 4. Insérer dans shared_matches
            async with self._shared_db_lock:  # type: ignore[attr-defined]
                shared_conn = self._get_shared_connection()  # type: ignore[attr-defined]
                if shared_conn is None:
                    result["error"] = "shared_connection indisponible"
                    return result

                self._insert_shared_registry(shared_conn, registry_data)
                self._insert_shared_participants(shared_conn, participants)
                self._insert_shared_medals(shared_conn, medals_all)

                if event_rows_shared:
                    self._insert_shared_events(shared_conn, event_rows_shared)
                    result["events"] = len(event_rows_shared)

                if alias_rows:
                    self._insert_shared_aliases(shared_conn, alias_rows)
                    result["aliases"] = len(alias_rows)

                # Mettre à jour les flags du registre
                _utc_now = datetime.now(timezone.utc).replace(tzinfo=None)
                shared_conn.execute(
                    """UPDATE match_registry SET
                        participants_loaded = TRUE,
                        events_loaded = ?,
                        medals_loaded = TRUE,
                        first_sync_by = ?,
                        first_sync_at = ?,
                        player_count = 1
                    WHERE match_id = ?""",
                    (
                        len(event_rows_shared) > 0,
                        self._gamertag,  # type: ignore[attr-defined]
                        _utc_now,
                        match_id,
                    ),
                )

                # Marquer le bitmask backfill_completed avec les types syncés
                bf_mask = self._compute_backfill_mask(options)
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (bf_mask, match_id),
                )

                # Mettre à jour backfill_bits par participant (v5.2)
                self._update_match_participant_bits(shared_conn, match_id)

            # 4b. Stats PVE → shared_pve.duckdb (Firefight uniquement) — v5.2
            await self._try_insert_pve_stats(stats_json, match_id, shared_conn)

            # 5. Insérer les données personnelles dans la player DB
            match_row = transform_match_stats(
                stats_json,
                self._xuid,  # type: ignore[attr-defined]
                skill_json=skill_json,
                metadata_resolver=self._metadata_resolver,  # type: ignore[attr-defined]
            )
            if match_row is None:
                result["error"] = f"Transformation match_stats échouée pour {match_id}"
                return result

            _skill_row = None
            if skill_json:
                _skill_row = transform_skill_stats(skill_json, match_id, self._xuid)  # type: ignore[attr-defined]

            # ✅ Écrire team_mmr/enemy_mmr + expected/stddev dans shared.match_participants (V5.1)
            if _skill_row and (_skill_row.team_mmr is not None or _skill_row.enemy_mmr is not None):
                async with self._shared_db_lock:  # type: ignore[attr-defined]
                    shared_conn = self._get_shared_connection()  # type: ignore[attr-defined]
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
                                _skill_row.team_mmr,
                                _skill_row.enemy_mmr,
                                _skill_row.kills_expected,
                                _skill_row.kills_stddev,
                                _skill_row.deaths_expected,
                                _skill_row.deaths_stddev,
                                _skill_row.assists_expected,
                                _skill_row.assists_stddev,
                                match_id,
                                self._xuid,  # type: ignore[attr-defined]
                            ),
                        )

            from src.data.sync.transformers import extract_medals

            _medal_rows_personal = extract_medals(stats_json, self._xuid)  # type: ignore[attr-defined]

            personal_scores = extract_personal_score_awards(stats_json, self._xuid)  # type: ignore[attr-defined]
            personal_score_rows = []
            if personal_scores:
                personal_score_rows = transform_personal_score_awards(
                    match_id,
                    self._xuid,  # type: ignore[attr-defined]
                    personal_scores,
                )

            _participant_rows_player = []
            if options.with_participants:
                _participant_rows_player = participants  # Réutiliser l'extraction

            # V5 finale : écriture minimale dans player DB (seulement enrichissements personnels)
            async with self._db_lock:  # type: ignore[attr-defined]
                # ✅ CONSERVÉ : personal_score_awards (enrichissement personnel)
                if personal_score_rows:
                    self._insert_personal_score_rows(personal_score_rows)  # type: ignore[attr-defined]

                # ✅ NOUVEAU : player_match_enrichment (performance_score, sessions, etc.)
                self._insert_enrichment_row(match_id, match_row)  # type: ignore[attr-defined]
                self._compute_and_update_performance_score(match_id, match_row)

                # ✅ CSR (v5.2) : écrire le CSR dans match_skill_rank si match classé
                if _skill_row is not None and match_row.is_ranked:
                    self._upsert_csr_rating(match_id, _skill_row)

            result["inserted"] = True

        except Exception as e:
            result["error"] = f"Erreur traitement new {match_id}: {e}"
            logger.warning(result["error"])

        return result

    async def _try_insert_pve_stats(
        self,
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
            async with self._pve_db_lock:  # type: ignore[attr-defined]
                pve_conn = self._get_pve_connection()  # type: ignore[attr-defined]
                if pve_rows:
                    inserted = batch_insert_pve_stats(pve_conn, pve_rows)
                    logger.debug(f"PVE stats insérées pour {match_id}: {inserted} lignes")

            # Poser le guard dans match_registry.backfill_completed (même si 0 lignes)
            if shared_conn is not None:
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (MatchBits.PVE_STATS, match_id),
                )
        except Exception as e:
            logger.debug(f"PVE stats non insérées pour {match_id}: {e}")

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
