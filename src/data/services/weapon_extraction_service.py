"""Service d'extraction des armes depuis les films SPNKr.

Orchestre les ports (API), le domaine pur (weapon_parser) et le repo
(weapon_kills) sans dépendre directement des implémentations concrètes.

Architecture hexagonale : le service accepte un ``HaloAPIPort`` (Protocol)
et un ``duckdb.DuckDBPyConnection`` pour les écritures.

Algorithme v5.7 (FINDINGS §6a/b) :
- Section 2 fire events  → attribution POV (avec confidence zones)
- Formula A Section 1   → attribution T1 coéquipiers (snapshot)
- Tous joueurs traités  → player_index via méthode acurtis (inv #26)
"""

from __future__ import annotations

import asyncio
import logging
from pathlib import Path

import duckdb

from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    build_weapon_timeline,
    correlate_kills_to_weapons,
    detect_player_indices,
    find_chunk_at_time,
    scan_fire_events,
)
from src.data.repositories._weapon_kills_repo import WeaponKillsMixin
from src.data.sync.api_port import HaloAPIPort

logger = logging.getLogger(__name__)


_CHUNK_TIMEOUT_S = 30.0
_MAX_CONCURRENT_CHUNKS = 5


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
        self._chunk_sem = asyncio.Semaphore(_MAX_CONCURRENT_CHUNKS)

    async def process_match(
        self,
        match_id: str,
        gamertag: str,
        xuid: str,
        *,
        dry_run: bool = False,
    ) -> dict:
        """Traite un match : télécharge chunks, corrèle kills → armes, upsert.

        Utilise Section 2 (fire events) pour le POV et Formula A (Section 1)
        pour les coéquipiers T1. Tous joueurs dont le player_index est
        détectable via acurtis sont traités.

        Returns:
            Dict résumé {match_id, kills_total, kills_attributed,
            rows_inserted, players_processed, error?}.
        """
        summary: dict = {
            "match_id": match_id,
            "kills_total": 0,
            "kills_attributed": 0,
            "rows_inserted": 0,
            "players_processed": 0,
        }
        try:
            logger.debug("process_match %s (pov=%s)", match_id[:8], xuid[:8] if xuid else "?")
            all_participants = self._load_participants(match_id)
            if xuid and xuid not in all_participants:
                all_participants[xuid] = gamertag

            kill_times_all = self._load_all_kill_times(match_id, list(all_participants))
            if not kill_times_all:
                summary["error"] = "aucun kill trouvé"
                return summary

            chunks = await self._download_needed_chunks(
                match_id, [t for times in kill_times_all.values() for t in times]
            )
            if not chunks:
                summary["error"] = "aucun chunk téléchargé"
                return summary

            logger.debug("Match %s : %d chunks", match_id[:8], len(chunks))
            xuid_int_to_pi = self._resolve_player_indices(chunks, all_participants, xuid)
            timeline, timing = build_weapon_timeline(chunks)
            chunks_sorted = sorted(chunks.keys())

            kt, ka, rt = self._attribute_all_players(
                match_id,
                all_participants,
                xuid,
                chunks,
                chunks_sorted,
                timeline,
                timing,
                xuid_int_to_pi,
                dry_run,
            )
            if not dry_run:
                WeaponKillsMixin.mark_weapon_backfill_done(self._conn, match_id)

            summary["kills_total"] = kt
            summary["kills_attributed"] = ka
            summary["rows_inserted"] = rt

        except Exception as exc:
            logger.warning("Erreur match %s : %s", match_id[:8], exc)
            summary["error"] = str(exc)

        return summary

    def _resolve_player_indices(
        self,
        chunks: dict,
        all_participants: dict[str, str],
        pov_xuid: str,
    ) -> dict[int, int]:
        """Résout XUID → player_index via acurtis (inv #26), fallback pi=1 pour POV."""
        first_chunk_data = next(iter(v[0] for v in chunks.values()))
        xuid_int_map = {int(x): x for x in all_participants if x.isdigit()}
        pi_to_xuid_int = detect_player_indices(first_chunk_data, xuid_int_map)
        xuid_int_to_pi = {v: k for k, v in pi_to_xuid_int.items()}
        pov_int = int(pov_xuid) if pov_xuid and pov_xuid.isdigit() else None
        if pov_int and pov_int not in xuid_int_to_pi:
            xuid_int_to_pi[pov_int] = 1
            logger.debug("Fallback pi=1 pour POV %s", pov_xuid[:8] if pov_xuid else "?")
        return xuid_int_to_pi

    def _attribute_all_players(  # noqa: PLR0913
        self,
        match_id: str,
        all_participants: dict[str, str],
        pov_xuid: str,
        chunks: dict,
        chunks_sorted: list[int],
        timeline: dict,
        timing: list,
        xuid_int_to_pi: dict[int, int],
        dry_run: bool,
    ) -> tuple[int, int, int]:
        """Itère sur tous les participants, attribue les kills, écrit en DB.

        Returns:
            (kills_total, kills_attributed, rows_inserted)
        """
        kills_total = kills_attributed = rows_total = 0
        for xuid_str, gt in all_participants.items():
            xuid_i = int(xuid_str) if xuid_str.isdigit() else None
            if xuid_i is None:
                continue
            player_index = xuid_int_to_pi.get(xuid_i)
            if player_index is None:
                logger.debug("Match %s : pi inconnu pour %s", match_id[:8], gt)
                continue

            kills = WeaponKillsMixin.load_player_kills_for_match(self._conn, match_id, xuid_str)
            if not kills:
                continue

            kill_rows = self._attribute_kills(
                kills,
                chunks,
                chunks_sorted,
                timeline,
                timing,
                player_index,
                xuid_str,
                pov_xuid,
            )
            kills_total += len(kills)
            kills_attributed += sum(
                1 for r in kill_rows if r["weapon_name"] not in ("NON TROUVE", "UNKNOWN")
            )
            if not dry_run and kill_rows:
                rows = WeaponKillsMixin.insert_weapon_kill_rows(
                    self._conn, match_id, xuid_str, kill_rows
                )
                rows_total += rows
                logger.debug(
                    "Match %s %s (pi=%d) : %d lignes", match_id[:8], gt, player_index, rows
                )
        return kills_total, kills_attributed, rows_total

    # ── Attribution ───────────────────────────────────────────────────────

    def _attribute_kills(  # noqa: PLR0913
        self,
        kills: list[dict],
        chunks: dict,
        chunks_sorted: list[int],
        timeline: dict,
        timing: list,
        player_index: int,
        xuid_str: str,
        pov_xuid: str,
    ) -> list[dict]:
        """Attribution POV (fire events §6a) ou T1 (Formula A §6b)."""
        is_pov = xuid_str == pov_xuid
        if is_pov:
            fire_events = self._scan_player_chunks(chunks, player_index)
            return correlate_kills_to_weapons(kills, fire_events)
        return self._attribute_t1_kills(kills, chunks_sorted, timeline, timing, player_index)

    @staticmethod
    def _attribute_t1_kills(
        kills: list[dict],
        chunks_sorted: list[int],
        timeline: dict,
        timing: list,
        player_index: int,
    ) -> list[dict]:
        """Attribution T1 via Formula A snapshot (FINDINGS §6b).

        Pour chaque kill à T : chunk couvrant T → weapon_A[chunk][pi].
        Fallback chunk-1 si aucune MAJ dans ce chunk.
        """
        from src.analysis.weapon_parser import WEAPON_ID_MAP

        results = []
        for kill in kills:
            if kill.get("is_melee"):
                results.append(
                    {
                        **kill,
                        "weapon_name": "MELEE",
                        "delta_ms": None,
                        "confidence": "none",
                        "swap_detected": False,
                        "delayed_damage": False,
                    }
                )
                continue
            if kill.get("is_grenade"):
                results.append(
                    {
                        **kill,
                        "weapon_name": "GRENADE",
                        "delta_ms": None,
                        "confidence": "none",
                        "swap_detected": False,
                        "delayed_damage": False,
                    }
                )
                continue

            t_ms = kill["time_ms"]
            ck = find_chunk_at_time(chunks_sorted, timing, t_ms)
            wid = timeline.get(ck, {}).get(player_index) or timeline.get(ck - 1, {}).get(
                player_index
            )
            if wid is None:
                results.append(
                    {
                        **kill,
                        "weapon_name": "UNKNOWN",
                        "delta_ms": None,
                        "confidence": "none",
                        "swap_detected": False,
                        "delayed_damage": False,
                    }
                )
                continue

            wname = WEAPON_ID_MAP.get(wid, f"?{wid.hex()[:8]}")
            conf = "high" if wid in WEAPON_ID_MAP else "low"
            # Swap check : plusieurs wids pour ce pi dans ce chunk = MEDIUM
            chunk_state = timeline.get(ck, {})
            if len(chunk_state) > 1 and conf == "high":
                conf = "medium"
            results.append(
                {
                    **kill,
                    "weapon_name": wname,
                    "delta_ms": None,
                    "confidence": conf,
                    "swap_detected": False,
                    "delayed_damage": False,
                }
            )
        return results

    # ── Privées ──────────────────────────────────────────────────────────

    def _load_participants(self, match_id: str) -> dict[str, str]:
        try:
            rows = self._conn.execute(
                "SELECT mp.xuid, COALESCE(xa.gamertag, mp.xuid) "
                "FROM match_participants mp "
                "LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid "
                "WHERE mp.match_id = ? AND mp.xuid NOT LIKE 'bid(%'",
                (match_id,),
            ).fetchall()
            return {xuid: gt for xuid, gt in rows if xuid}
        except Exception as exc:
            logger.debug("_load_participants %s : %s", match_id[:8], exc)
            return {}

    def _load_all_kill_times(self, match_id: str, xuids: list[str]) -> dict[str, list[int]]:
        if not xuids:
            return {}
        try:
            placeholders = ", ".join("?" for _ in xuids)
            rows = self._conn.execute(
                f"SELECT xuid, time_ms FROM highlight_events "
                f"WHERE match_id = ? AND event_type = 'kill' AND xuid IN ({placeholders}) "
                f"ORDER BY time_ms",
                [match_id, *xuids],
            ).fetchall()
            result: dict[str, list[int]] = {}
            for xuid, t in rows:
                result.setdefault(xuid, []).append(t)
            return result
        except Exception as exc:
            logger.debug("_load_all_kill_times %s : %s", match_id[:8], exc)
            return {}

    async def _download_needed_chunks(  # noqa: C901, PLR0912
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
                if ch.chunk_type.value != 2:
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
            tasks = [
                self._download_chunk_with_sem(ch, blob_prefix, match_cache) for ch in to_download
            ]
            try:
                gathered = await asyncio.wait_for(
                    asyncio.gather(*tasks, return_exceptions=True),
                    timeout=_CHUNK_TIMEOUT_S * len(to_download),
                )
            except TimeoutError:
                logger.warning("Match %s : timeout chunks (%ds)", match_id[:8], _CHUNK_TIMEOUT_S)
                return result
            for item in gathered:
                if isinstance(item, Exception):
                    logger.debug("Chunk download exception : %s", item)
                    continue
                idx, data, start_ms, dur_ms = item
                if data is not None:
                    result[idx] = (data, start_ms, dur_ms)

        return result

    async def _download_chunk_with_sem(
        self, ch, blob_prefix: str, match_cache: Path
    ) -> tuple[int, bytes | None, int, int]:
        async with self._chunk_sem:
            return await self._download_chunk(ch, blob_prefix, match_cache)

    async def _download_chunk(
        self, ch, blob_prefix: str, match_cache: Path
    ) -> tuple[int, bytes | None, int, int]:
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
    def _scan_player_chunks(
        chunks: dict[int, tuple[bytes, int, int]],
        player_index: int,
    ) -> list[dict]:
        all_events: list[dict] = []
        for _idx, (chunk_data, start_ms, dur_ms) in sorted(chunks.items()):
            all_events.extend(scan_fire_events(chunk_data, player_index, start_ms, dur_ms))
        all_events.sort(key=lambda e: e["timestamp_ms"])
        return all_events
