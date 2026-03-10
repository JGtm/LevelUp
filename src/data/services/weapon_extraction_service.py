"""Service d'extraction des armes depuis les films SPNKr.

Orchestre les ports (API), le domaine pur (weapon_parser) et le repo
(weapon_kills) sans dépendre directement des implémentations concrètes.

Architecture hexagonale : le service accepte un ``HaloAPIPort`` (Protocol)
et un ``duckdb.DuckDBPyConnection`` pour les écritures.

Algorithme v5.6 (FINDINGS §6a/b) :
- Section 2 fire events  → attribution POV (avec confidence zones)
- Formula A Section 1   → attribution T1 coéquipiers (snapshot)
- Tous joueurs traités  → player_index via méthode acurtis (inv #26)
"""

from __future__ import annotations

import asyncio
import contextlib
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


def _attribution_row(kill: dict, weapon_name: str, confidence: str = "none") -> dict:
    """Construit une ligne d'attribution (weapon_name, confidence) depuis un kill brut."""
    return {
        **kill,
        "weapon_name": weapon_name,
        "delta_ms": None,
        "confidence": confidence,
        "swap_detected": False,
        "delayed_damage": False,
    }


class WeaponExtractionService:
    """Orchestre l'extraction d'armes pour un match donné."""

    def __init__(
        self,
        api: HaloAPIPort,
        conn: duckdb.DuckDBPyConnection,
        cache_dir: Path,
        *,
        write_lock: asyncio.Lock | None = None,
    ) -> None:
        self._api = api
        self._conn = conn
        self._cache_dir = cache_dir
        self._chunk_sem = asyncio.Semaphore(_MAX_CONCURRENT_CHUNKS)
        # Sérialiseur d'écritures DuckDB (requis si plusieurs matchs en parallèle)
        self._write_lock: asyncio.Lock | contextlib.AbstractAsyncContextManager = (
            write_lock if write_lock is not None else contextlib.nullcontext()
        )

    async def _mark_done(self, match_id: str, *, no_film: bool) -> None:
        """Pose le bit approprié selon que le film était disponible ou non."""
        async with self._write_lock:
            if no_film:
                WeaponKillsMixin.mark_weapon_no_film(self._conn, match_id)
            else:
                WeaponKillsMixin.mark_weapon_backfill_done(self._conn, match_id)

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
        détectable via acurtis sont traités. Retourne un dict résumé
        {match_id, kills_total, kills_attributed, rows_inserted, error?}.
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
                await self._mark_done(match_id, no_film=False)
                summary["error"] = "aucun kill trouvé"
                return summary

            chunks = await self._download_needed_chunks(
                match_id, [t for times in kill_times_all.values() for t in times]
            )
            if not chunks:
                await self._mark_done(match_id, no_film=True)
                summary["error"] = "aucun chunk téléchargé (404 ou expiré)"
                return summary

            logger.debug("Match %s : %d chunks", match_id[:8], len(chunks))
            (
                xuid_int_to_pi,
                timeline,
                swap_pis,
                timing,
                all_kills_by_xuid,
            ) = await self._prepare_match_data(match_id, chunks, all_participants)
            chunks_sorted = sorted(chunks.keys())
            # Le joueur cible (xuid) est celui pour qui on fait la Section 2 (POV pi=1).
            # L'espace pi Section 2 est indépendant de l'acurtis pi — ne pas confondre.
            kt, ka, rt = await self._attribute_all_players(
                match_id,
                all_participants,
                xuid,
                chunks,
                chunks_sorted,
                timeline,
                swap_pis,
                timing,
                xuid_int_to_pi,
                all_kills_by_xuid,
                dry_run,
            )
            if not dry_run:
                async with self._write_lock:
                    WeaponKillsMixin.mark_weapon_backfill_done(self._conn, match_id)

            summary["kills_total"] = kt
            summary["kills_attributed"] = ka
            summary["rows_inserted"] = rt

        except Exception as exc:
            logger.warning("Erreur match %s : %s", match_id[:8], exc)
            summary["error"] = str(exc)

        return summary

    async def _prepare_match_data(
        self,
        match_id: str,
        chunks: dict,
        all_participants: dict[str, str],
    ) -> tuple[dict, dict, dict, list, dict]:
        """Résout player_indices, construit la timeline et charge les kills batch.

        CPU-bound (bitstring) offloadé sur thread pool pour libérer l'event loop.
        Batch SQL : 2 requêtes pour tous les joueurs (Phase 2 + Phase 4).
        """
        xuid_int_to_pi = await asyncio.to_thread(
            self._resolve_player_indices, chunks, all_participants
        )
        timeline, swap_pis, timing = await asyncio.to_thread(build_weapon_timeline, chunks)
        all_kills_by_xuid = WeaponKillsMixin.load_all_kills_for_match(self._conn, match_id)
        logger.debug(
            "Match %s : %d pi résolus, %d chunks timeline, %d joueurs avec kills",
            match_id[:8],
            len(xuid_int_to_pi),
            len(timeline),
            len(all_kills_by_xuid),
        )
        return xuid_int_to_pi, timeline, swap_pis, timing, all_kills_by_xuid

    def _resolve_player_indices(
        self,
        chunks: dict,
        all_participants: dict[str, str],
    ) -> dict[int, int]:
        """Résout XUID → player_index via acurtis (inv #26)."""
        first_chunk_data = next(iter(v[0] for v in chunks.values()))
        xuid_int_map = {int(x): x for x in all_participants if x.isdigit()}
        pi_to_xuid_int = detect_player_indices(first_chunk_data, xuid_int_map)
        return {v: k for k, v in pi_to_xuid_int.items()}

    async def _attribute_all_players(  # noqa: PLR0913
        self,
        match_id: str,
        all_participants: dict[str, str],
        pov_xuid: str,
        chunks: dict,
        chunks_sorted: list[int],
        timeline: dict,
        swap_pis: dict[int, set[int]],
        timing: list,
        xuid_int_to_pi: dict[int, int],
        all_kills_by_xuid: dict[str, list[dict]],
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
            is_target = xuid_str == pov_xuid
            if player_index is None and not is_target:
                logger.debug("Match %s : pi inconnu pour %s", match_id[:8], gt)
                continue
            if player_index is None:
                player_index = 0  # dummy : non utilisé pour Section 2 (pi=1 invariant)

            kills = all_kills_by_xuid.get(xuid_str, [])
            if not kills:
                continue

            kill_rows = await self._attribute_kills(
                kills,
                chunks,
                chunks_sorted,
                timeline,
                swap_pis,
                timing,
                player_index,
                xuid_str,
                pov_xuid,
            )
            if xuid_str == pov_xuid:
                kill_rows = self._reconcile_api_aggregates(
                    kill_rows, self._conn, match_id, xuid_str
                )
            kills_total += len(kills)
            kills_attributed += sum(
                1 for r in kill_rows if r["weapon_name"] not in ("NON TROUVE", "UNKNOWN")
            )
            if not dry_run and kill_rows:
                async with self._write_lock:
                    rows = WeaponKillsMixin.insert_weapon_kill_rows(
                        self._conn, match_id, xuid_str, kill_rows
                    )
                rows_total += rows
                logger.debug(
                    "Match %s %s (pi=%d) : %d lignes", match_id[:8], gt, player_index, rows
                )
        return kills_total, kills_attributed, rows_total

    # ── Attribution ───────────────────────────────────────────────────────

    async def _attribute_kills(  # noqa: PLR0913
        self,
        kills: list[dict],
        chunks: dict,
        chunks_sorted: list[int],
        timeline: dict,
        swap_pis: dict[int, set[int]],
        timing: list,
        player_index: int,
        xuid_str: str,
        pov_xuid: str,
    ) -> list[dict]:
        """Attribution POV (fire events §6a) ou T1 (Formula A §6b)."""
        is_pov = xuid_str == pov_xuid
        if is_pov:
            # Section 2 : le POV est TOUJOURS à pi=1 (inv §6a / inv #6/#23/#27/#41)
            # L'espace pi Section 2 est indépendant du pi acurtis — ne pas utiliser player_index
            # CPU-bound bitstring : offload sur thread pour libérer l'event loop (Phase 4)
            fire_events = await asyncio.to_thread(self._scan_player_chunks, chunks, 1)
            return correlate_kills_to_weapons(kills, fire_events)
        return self._attribute_t1_kills(
            kills, chunks_sorted, timeline, swap_pis, timing, player_index
        )

    @staticmethod
    def _attribute_t1_kills(  # noqa: PLR0913
        kills: list[dict],
        chunks_sorted: list[int],
        timeline: dict,
        swap_pis: dict[int, set[int]],
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
                results.append(_attribution_row(kill, "MELEE"))
                continue
            if kill.get("is_grenade"):
                results.append(_attribution_row(kill, "GRENADE"))
                continue

            t_ms = kill["time_ms"]
            ck = find_chunk_at_time(chunks_sorted, timing, t_ms)
            wid = timeline.get(ck, {}).get(player_index) or timeline.get(ck - 1, {}).get(
                player_index
            )
            if wid is None:
                results.append(_attribution_row(kill, "UNKNOWN"))
                continue

            wname = WEAPON_ID_MAP.get(wid, f"?{wid.hex()}")
            conf = "high" if wid in WEAPON_ID_MAP else "low"
            # MEDIUM si ce pi a eu plusieurs armes distinctes dans ce chunk
            # (swap intra-chunk ~19s — FINDINGS §6b Step 3)
            if player_index in swap_pis.get(ck, set()) and conf == "high":
                conf = "medium"
            results.append(_attribution_row(kill, wname, conf))
        n_unknown_wid = sum(1 for r in results if r["weapon_name"].startswith("?"))
        n_unknown = sum(1 for r in results if r["weapon_name"] == "UNKNOWN")
        if n_unknown_wid or n_unknown:
            logger.debug(
                "_attribute_t1 pi=%d : %d kills, %d ?hex, %d UNKNOWN",
                player_index,
                len(results),
                n_unknown_wid,
                n_unknown,
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
    def _reconcile_api_aggregates(
        kill_rows: list[dict],
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
        xuid_str: str,
    ) -> list[dict]:
        """Réconcilie la confidence POV avec les agrégats API (FINDINGS §6a Step 4).

        - Step 4a : si HIGH weapon kills > api → demote les moins certains (grand delta)
        - Step 4c : si HIGH weapon kills < api → promouvoir des MEDIUM en HIGH
        """
        try:
            row = conn.execute(
                "SELECT COALESCE(grenade_kills, 0), COALESCE(melee_kills, 0) "
                "FROM match_participants WHERE match_id = ? AND xuid = ?",
                (match_id, xuid_str),
            ).fetchone()
        except Exception:
            return kill_rows
        if row is None:
            return kill_rows
        api_grenade = int(row[0])
        api_melee = int(row[1])
        api_weapon_kills = max(len(kill_rows) - api_grenade - api_melee, 0)
        _excluded = ("MELEE", "GRENADE", "NON TROUVE", "UNKNOWN")
        weapon_high = [
            r
            for r in kill_rows
            if r.get("confidence") == "high" and r.get("weapon_name") not in _excluded
        ]
        # Step 4a — surdétection : demote les HIGH les moins certains
        if len(weapon_high) > api_weapon_kills:
            excess = len(weapon_high) - api_weapon_kills
            logger.debug(
                "_reconcile %s %s : step4a demote %d→%d (−%d)",
                match_id[:8],
                xuid_str[:8],
                len(weapon_high),
                api_weapon_kills,
                excess,
            )
            for r in sorted(weapon_high, key=lambda x: x.get("delta_ms") or 0, reverse=True)[
                :excess
            ]:
                r["confidence"] = "medium"
            return kill_rows
        # Step 4c — sous-détection : promouvoir des MEDIUM en HIGH jusqu'à combler le déficit
        high_after = sum(
            1
            for r in kill_rows
            if r.get("confidence") == "high" and r.get("weapon_name") not in _excluded
        )
        if high_after < api_weapon_kills:
            deficit = api_weapon_kills - high_after
            logger.debug(
                "_reconcile %s %s : step4c promote %d→%d (+%d)",
                match_id[:8],
                xuid_str[:8],
                high_after,
                api_weapon_kills,
                deficit,
            )
            medium_kills = sorted(
                [
                    r
                    for r in kill_rows
                    if r.get("confidence") == "medium" and r.get("weapon_name") not in _excluded
                ],
                key=lambda x: x.get("delta_ms") or 0,  # plus certains en premier
            )
            for r in medium_kills[:deficit]:
                r["confidence"] = "high"
        return kill_rows

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
