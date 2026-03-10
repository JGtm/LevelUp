"""Service d'extraction des armes depuis les films SPNKr.

Orchestre les ports (API), le domaine pur (weapon_parser) et le repo
(weapon_kills) sans dépendre directement des implémentations concrètes.

Architecture hexagonale : le service accepte un ``HaloAPIPort`` (Protocol)
et un ``duckdb.DuckDBPyConnection`` pour les écritures.
"""

from __future__ import annotations

import asyncio
import logging
from pathlib import Path

import duckdb

from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    POV_PLAYER_INDEX,
    correlate_kills_to_weapons,
    count_kills_by_api_weapon,
    scan_fire_events,
)
from src.data.repositories._weapon_kills_repo import WeaponKillsMixin
from src.data.sync.api_port import HaloAPIPort

logger = logging.getLogger(__name__)


class WeaponExtractionService:
    """Orchestre l'extraction d'armes pour un match donné."""

    def __init__(
        self,
        api: HaloAPIPort,
        conn: duckdb.DuckDBPyConnection,
        cache_dir: Path,
    ) -> None:
        self._api = api
        self._conn = conn
        self._cache_dir = cache_dir

    async def process_match(
        self,
        match_id: str,
        gamertag: str,
        xuid: str,
        *,
        dry_run: bool = False,
    ) -> dict:
        """Traite un match : charge kills, télécharge chunks, corrèle, upsert.

        Retourne un dict résumé {match_id, kills_total, kills_attributed,
        rows_inserted, weapon_counts, error?}.
        """
        summary: dict = {
            "match_id": match_id,
            "kills_total": 0,
            "kills_attributed": 0,
            "rows_inserted": 0,
        }
        try:
            logger.debug("process_match %s pour %s (xuid=%s)", match_id[:8], gamertag, xuid[:8])

            kills = WeaponKillsMixin.load_player_kills_for_match(self._conn, match_id, gamertag)
            if not kills:
                summary["error"] = "aucun kill POV"
                logger.debug("Match %s : aucun kill POV trouvé", match_id[:8])
                return summary

            summary["kills_total"] = len(kills)
            kill_times = [k["time_ms"] for k in kills]
            logger.debug("Match %s : %d kills POV chargés", match_id[:8], len(kills))

            chunks = await self._download_needed_chunks(match_id, kill_times)
            if not chunks:
                summary["error"] = "aucun chunk téléchargé"
                logger.debug("Match %s : aucun chunk REPLICATION_DATA", match_id[:8])
                return summary

            logger.debug("Match %s : %d chunks téléchargés", match_id[:8], len(chunks))

            all_fire_events = self._scan_all_chunks(chunks)
            logger.debug("Match %s : %d fire events détectés", match_id[:8], len(all_fire_events))

            correlated = correlate_kills_to_weapons(kills, all_fire_events)
            counts = count_kills_by_api_weapon(correlated)

            attributed = sum(
                1
                for r in correlated
                if r.get("matched_fire_event") or r.get("is_melee") or r.get("is_grenade")
            )
            summary["kills_attributed"] = attributed
            summary["weapon_counts"] = dict(counts)

            if not dry_run:
                rows = WeaponKillsMixin.upsert_weapon_kills(self._conn, match_id, xuid, counts)
                WeaponKillsMixin.mark_weapon_backfill_done(self._conn, match_id)
                summary["rows_inserted"] = rows
                logger.debug("Match %s : %d lignes upsertées", match_id[:8], rows)
            else:
                logger.debug("Match %s : dry-run, %d armes trouvées", match_id[:8], len(counts))
        except Exception as exc:
            logger.warning("Erreur match %s : %s", match_id[:8], exc)
            summary["error"] = str(exc)

        return summary

    # ── Privées ──────────────────────────────────────────────────────────

    async def _download_needed_chunks(
        self,
        match_id: str,
        kill_times_ms: list[int],
    ) -> dict[int, tuple[bytes, int, int]]:
        """Télécharge les chunks REPLICATION_DATA couvrant les kill_times."""
        match_cache = self._cache_dir / match_id[:8]
        match_cache.mkdir(parents=True, exist_ok=True)

        film = await self._api.get_film_by_match_id(match_id)
        if film is None:
            return {}

        blob_prefix = film.blob_storage_path_prefix
        chunks_meta = film.custom_data.chunks

        needed: set[int] = set()
        for kill_t in kill_times_ms:
            window_start = kill_t - KILL_WINDOW_MS
            for ch in chunks_meta:
                if ch.chunk_type.value != 2:  # REPLICATION_DATA
                    continue
                ch_start = ch.chunk_start_time_offset_milliseconds
                ch_end = ch_start + ch.duration_milliseconds
                if ch_end >= window_start and ch_start <= kill_t:
                    needed.add(ch.index)

        result: dict[int, tuple[bytes, int, int]] = {}
        to_download = []

        for ch in sorted(chunks_meta, key=lambda c: c.index):
            if ch.index not in needed:
                continue
            cache_path = match_cache / f"chunk_{ch.index:02d}.bin"
            if cache_path.exists():
                data = cache_path.read_bytes()
                result[ch.index] = (
                    data,
                    ch.chunk_start_time_offset_milliseconds,
                    ch.duration_milliseconds,
                )
            else:
                to_download.append(ch)

        if to_download:
            tasks = [self._download_chunk(ch, blob_prefix, match_cache) for ch in to_download]
            for idx, data, start_ms, dur_ms in await asyncio.gather(*tasks):
                if data is not None:
                    result[idx] = (data, start_ms, dur_ms)

        return result

    async def _download_chunk(
        self, ch, blob_prefix: str, match_cache: Path
    ) -> tuple[int, bytes | None, int, int]:
        """Télécharge un chunk via l'API et le cache localement."""
        url = blob_prefix + ch.file_relative_path.lstrip("/")
        data = await self._api.download_film_chunk(url)
        if data is not None:
            cache_path = match_cache / f"chunk_{ch.index:02d}.bin"
            cache_path.write_bytes(data)
        return (
            ch.index,
            data,
            ch.chunk_start_time_offset_milliseconds,
            ch.duration_milliseconds,
        )

    @staticmethod
    def _scan_all_chunks(
        chunks: dict[int, tuple[bytes, int, int]],
    ) -> list[dict]:
        """Scanne les fire events POV dans tous les chunks."""
        all_events: list[dict] = []
        for _idx, (chunk_data, start_ms, dur_ms) in sorted(chunks.items()):
            all_events.extend(scan_fire_events(chunk_data, POV_PLAYER_INDEX, start_ms, dur_ms))
        all_events.sort(key=lambda e: e["timestamp_ms"])
        return all_events
