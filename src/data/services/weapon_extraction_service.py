"""Service d'extraction des armes depuis les films SPNKr.

Orchestre les ports (API), le domaine pur (weapon_parser) et le repo (weapon_kills).
v3 — player_index = b5>>4 (inv134) : scan universel, claim-and-remove par pi.
"""

from __future__ import annotations

import asyncio
import contextlib
import logging
from dataclasses import dataclass, field
from pathlib import Path

import duckdb

from src.analysis._global_correlation import correlate_kills_global
from src.analysis._kill_attribution import KillAttribution
from src.analysis._parser_logging import MatchLogCollector
from src.analysis.packet_index import (
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)
from src.analysis.reconciliation import reconcile_api_aggregates
from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    build_weapon_timelines,
    detect_player_indices,
    scan_fire_events_all,
)
from src.data.repositories._weapon_kills_repo import WeaponKillsMixin
from src.data.services._film_manifest_cache import (
    compute_needed_chunks,
    load_manifest_cache,
    write_manifest_cache,
)
from src.ports.api import HaloAPIPort

logger = logging.getLogger(__name__)


_CHUNK_TIMEOUT_S = 30.0
_MAX_CONCURRENT_CHUNKS = 50


@dataclass
class MatchProcessingResult:
    """Résultat du traitement d'un match."""

    match_id: str
    kills_total: int = 0
    kills_attributed: int = 0
    rows_inserted: int = 0
    players_processed: int = 0
    log_summary: dict = field(default_factory=dict)
    error: str | None = None

    @classmethod
    def empty(cls, match_id: str) -> MatchProcessingResult:
        """Résultat vide (aucun kill trouvé)."""
        return cls(match_id=match_id)


@dataclass
class ScanResult:
    """Résultat de la phase Scan multi-chunks."""

    fire_events_global: list[dict]
    timeline: dict[int, dict[int, bytes]]
    timeline_ns: dict[int, dict[int, bytes]]
    swap_pis: dict[int, set[int]]
    timing: list[tuple[int, int]]
    chunks_sorted: list[int]


class WeaponExtractionService:
    """Orchestre l'extraction d'armes v2 pour un match donné."""

    def __init__(  # noqa: PLR0913
        self,
        api: HaloAPIPort,
        conn: duckdb.DuckDBPyConnection,
        cache_dir: Path,
        *,
        manifest_dir: Path | None = None,
        write_lock: asyncio.Lock | None = None,
    ) -> None:
        self._api = api
        self._conn = conn
        self._cache_dir = cache_dir
        self._manifest_dir = manifest_dir or cache_dir.parent / "film_manifests"
        self._chunk_sem = asyncio.Semaphore(_MAX_CONCURRENT_CHUNKS)
        self._write_lock: asyncio.Lock | contextlib.AbstractAsyncContextManager = (
            write_lock if write_lock is not None else contextlib.nullcontext()
        )

    # ── Point d'entrée principal ─────────────────────────────────────────

    async def process_match(  # noqa: PLR0913
        self,
        match_id: str,
        gamertag: str,
        xuid: str,
        *,
        dry_run: bool = False,
        enable_reconciliation: bool = True,
        log_collector: MatchLogCollector | None = None,
    ) -> MatchProcessingResult:
        """Traite un match : télécharge chunks, corrèle kills → armes, upsert."""
        log = log_collector or MatchLogCollector(match_id)
        result = MatchProcessingResult(match_id=match_id)
        try:
            result = await self._process_match_inner(
                match_id,
                gamertag,
                xuid,
                dry_run,
                enable_reconciliation,
                log,
            )
        except Exception as exc:
            logger.warning("Erreur match %s : %s", match_id[:8], exc)
            result.error = str(exc)
        log.flush()
        result.log_summary = log.summary()
        return result

    async def _process_match_inner(  # noqa: PLR0913
        self,
        match_id: str,
        gamertag: str,
        xuid: str,
        dry_run: bool,
        enable_reconciliation: bool,
        log: MatchLogCollector,
    ) -> MatchProcessingResult:
        """Pipeline interne : load → download → scan → correlate → write."""
        participants = self._load_participants(match_id)
        if xuid and xuid not in participants:
            participants[xuid] = gamertag

        all_kills = WeaponKillsMixin.load_all_kills_for_match(self._conn, match_id)
        log.record_step("load_data", kills_total=sum(len(v) for v in all_kills.values()))

        if not any(all_kills.values()):
            await self._mark_done(match_id, no_film=False)
            return MatchProcessingResult(match_id=match_id)

        all_kill_times = [t for kills in all_kills.values() for k in kills for t in [k["time_ms"]]]
        chunks = await self._download_needed_chunks(match_id, all_kill_times)
        log.record_step("download", chunks_downloaded=len(chunks))

        if not chunks:
            await self._mark_done(match_id, no_film=True)
            return MatchProcessingResult(match_id=match_id, error="aucun chunk (404/expiré)")

        # Scan Phase : fire events + timelines + résolution player_index (passe unique CPU)
        scan, pi_map = await asyncio.to_thread(self._run_scan_phase, chunks, participants, log)

        # Le joueur POV a pi=0 dans le système fire_seq — l'ajouter si absent
        if xuid and xuid.isdigit():
            pi_map.setdefault(int(xuid), 0)

        # Correlation Phase : claim-and-remove par joueur
        all_attributions = self._correlate_all_players(
            match_id,
            all_kills,
            pi_map,
            scan,
            log,
        )

        # Réconciliation API optionnelle
        if enable_reconciliation:
            api_counts = self._load_api_weapon_counts(match_id, list(all_kills.keys()))
            for xuid_str, counts in api_counts.items():
                player_attrs = [a for a in all_attributions if a.xuid == xuid_str]
                if player_attrs and counts:
                    reconciled = reconcile_api_aggregates(
                        player_attrs,
                        counts,
                        log_collector=log,
                    )
                    # Remplacer dans la liste globale
                    attr_set = {id(a) for a in player_attrs}
                    all_attributions = [
                        a for a in all_attributions if id(a) not in attr_set
                    ] + reconciled

        # Écriture DB
        rows = 0
        if not dry_run and all_attributions:
            async with self._write_lock:
                rows = WeaponKillsMixin.insert_weapon_kill_rows_v2(
                    self._conn,
                    match_id,
                    all_attributions,
                )
                WeaponKillsMixin.mark_weapon_backfill_done(self._conn, match_id)
            log.record_step("write_db", rows_inserted=rows)

        return MatchProcessingResult(
            match_id=match_id,
            kills_total=len(all_attributions),
            kills_attributed=sum(1 for a in all_attributions if a.weapon_id is not None),
            rows_inserted=rows,
            players_processed=len(all_kills),
        )

    # ── Scan Phase ───────────────────────────────────────────────────────

    def _run_scan_phase(
        self,
        chunks: dict[int, tuple[bytes, int, int]],
        all_participants: dict[str, str],
        log: MatchLogCollector,
    ) -> tuple[ScanResult, dict[int, int]]:
        """Scanne fire events + timelines et résout player_index en une seule passe CPU.

        Évite le double index_chunk sur le chunk 0 : les packets indexés lors
        du scan sont réutilisés directement pour la résolution XUID → pi.

        Returns:
            (ScanResult, pi_map) où pi_map = {xuid_int: player_index}
        """
        all_raw_events: list[dict] = []
        timing: list[tuple[int, int]] = []
        chunks_sorted: list[int] = []
        first_data: bytes = b""
        first_packets: list = []

        for idx in sorted(chunks):
            data, start_ms, dur_ms = chunks[idx]
            try:
                packets = index_chunk(data)
            except Exception as exc:
                log.warn("index_chunk_failed", chunk=idx, error=str(exc))
                packets = []
            if not chunks_sorted:
                # Conserver le premier chunk indexé pour la résolution XUID → pi
                first_data = data
                first_packets = packets
            chunks_sorted.append(idx)
            timing.append((start_ms, start_ms + dur_ms))

            events = scan_fire_events_all(data, start_ms, dur_ms, packets=packets)
            all_raw_events.extend(events)
            log.record_step("scan_fire", chunk=idx, events=len(events))

        all_raw_events.sort(key=lambda e: e["timestamp_ms"])

        # Timelines Section 1 (raw + NS) — passe unique
        timeline, timeline_ns, swap_pis, _ = build_weapon_timelines(chunks)

        scan = ScanResult(
            fire_events_global=all_raw_events,
            timeline=timeline,
            timeline_ns=timeline_ns,
            swap_pis=swap_pis,
            timing=timing,
            chunks_sorted=chunks_sorted,
        )

        # Résolution player_index — réutilise first_packets déjà indexés
        pi_map = self._resolve_from_chunk(first_data, first_packets, all_participants)
        log.record_step("resolve_pi", resolved=len(pi_map), total=len(all_participants))

        return scan, pi_map

    # ── Correlation Phase ────────────────────────────────────────────────

    def _correlate_all_players(  # noqa: PLR0913
        self,
        match_id: str,
        all_kills: dict[str, list[dict]],
        pi_map: dict[int, int],
        scan: ScanResult,
        log: MatchLogCollector,
    ) -> list[KillAttribution]:
        """Corrèle tous les kills via claim-and-remove par joueur.

        Chaque kill ne consomme que les fire events dont player_index == pi du tueur
        (player_index = b5>>4, extrait directement lors du scan — inv134).
        """
        xuid_to_pi = {str(xuid_int): pi for xuid_int, pi in pi_map.items()}

        for xuid_str, kills in all_kills.items():
            if kills and xuid_str not in xuid_to_pi:
                log.warn("unresolved_player", xuid=xuid_str, kills=len(kills))

        kills_flat = [k for kills in all_kills.values() for k in kills]
        return correlate_kills_global(
            kills=kills_flat,
            fire_events_all=scan.fire_events_global,
            xuid_to_pi=xuid_to_pi,
            timeline=scan.timeline,
            swap_pis=scan.swap_pis,
            timing=scan.timing,
            chunks_sorted=scan.chunks_sorted,
            match_id=match_id,
            timeline_ns=scan.timeline_ns,
            log_collector=log,
        )

    # ── Résolution player_index ──────────────────────────────────────────

    def _resolve_from_chunk(
        self,
        first_chunk_data: bytes,
        first_packets: list,
        all_participants: dict[str, str],
    ) -> dict[int, int]:
        """Résout XUID → player_index depuis les packets déjà indexés du premier chunk.

        Méthode 1 : PLAYER_METADATA packet (précis, 16 joueurs).
        Fallback   : acurtis detect_player_indices (heuristique).
        """
        if not first_chunk_data:
            return {}

        xuid_int_map = {int(x): x for x in all_participants if x.isdigit()}

        try:
            metadata = extract_metadata_payload(first_chunk_data, first_packets)
        except Exception as exc:
            logger.debug("resolve_pi: metadata failed: %s", exc)
            metadata = None

        if metadata is not None:
            pi_to_xuid_int = detect_pi_from_metadata(metadata, xuid_int_map)
            if pi_to_xuid_int:
                # Résoudre les joueurs absents du METADATA (joueur POV → pi=0)
                # Le joueur POV n'a que des occurrences pi=0 dans METADATA donc
                # detect_pi_from_metadata ne le retourne pas. On le résout via
                # acurtis qui trouve son XUID avec pi=0 dans les données brutes.
                resolved = set(pi_to_xuid_int.values())
                missing = {x: g for x, g in xuid_int_map.items() if x not in resolved}
                if missing:
                    pov_result = detect_player_indices(first_chunk_data, missing)
                    pi_to_xuid_int.update(pov_result)
                    logger.debug(
                        "resolve_pi: POV résolu via acurtis (%d/%d)",
                        len(pov_result),
                        len(missing),
                    )
                logger.debug("resolve_pi: metadata OK (%d joueurs)", len(pi_to_xuid_int))
                return {v: k for k, v in pi_to_xuid_int.items()}

        # Fallback : méthode acurtis (metadata absente ou vide)
        logger.debug("resolve_pi: fallback acurtis (metadata=%s)", metadata is not None)
        pi_to_xuid_int = detect_player_indices(first_chunk_data, xuid_int_map)
        return {v: k for k, v in pi_to_xuid_int.items()}

    # ── Chargement données ───────────────────────────────────────────────

    def _load_participants(self, match_id: str) -> dict[str, str]:
        """Charge les participants du match (xuid → gamertag)."""
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

    def _load_api_weapon_counts(
        self,
        match_id: str,
        xuids: list[str],
    ) -> dict[str, dict[int, int]]:
        """Charge les agrégats weapon kills API pour la réconciliation."""
        result: dict[str, dict[int, int]] = {}
        if not xuids:
            return result
        try:
            placeholders = ", ".join("?" for _ in xuids)
            rows = self._conn.execute(
                f"SELECT xuid, COALESCE(grenade_kills, 0), COALESCE(melee_kills, 0), "
                f"COALESCE(kills, 0) "
                f"FROM match_participants WHERE match_id = ? AND xuid IN ({placeholders})",
                [match_id, *xuids],
            ).fetchall()
            from src.analysis._weapon_data import GRENADE_WEAPON_ID, MELEE_WEAPON_ID

            for xuid, grenade_k, melee_k, _total_k in rows:
                counts: dict[int, int] = {}
                if grenade_k > 0:
                    counts[GRENADE_WEAPON_ID] = grenade_k
                if melee_k > 0:
                    counts[MELEE_WEAPON_ID] = melee_k
                if counts:
                    result[xuid] = counts
        except Exception as exc:
            logger.debug("_load_api_weapon_counts %s : %s", match_id[:8], exc)
        return result

    async def _mark_done(self, match_id: str, *, no_film: bool) -> None:
        """Pose le bit approprié selon que le film était disponible ou non."""
        async with self._write_lock:
            if no_film:
                WeaponKillsMixin.mark_weapon_no_film(self._conn, match_id)
            else:
                WeaponKillsMixin.mark_weapon_backfill_done(self._conn, match_id)

    # ── Téléchargement chunks ────────────────────────────────────────────

    async def _download_needed_chunks(  # noqa: PLR0912
        self,
        match_id: str,
        kill_times_ms: list[int],
    ) -> dict[int, tuple[bytes, int, int]]:
        """Télécharge les chunks REPLICATION_DATA couvrant les kill_times.

        Le manifest (blob_prefix + chunk metadata) est mis en cache dans
        manifest.json pour éviter un appel API par match sur les re-runs.
        """
        match_cache = self._cache_dir / match_id[:8]
        match_cache.mkdir(parents=True, exist_ok=True)

        cached = load_manifest_cache(self._manifest_dir, match_id)
        if cached is not None:
            blob_prefix, chunks_meta = cached
        else:
            film = await self._api.get_film_by_match_id(match_id)
            if film is None:
                return {}
            blob_prefix = film.blob_storage_path_prefix
            chunks_meta = film.custom_data.chunks
            write_manifest_cache(self._manifest_dir, match_id, blob_prefix, chunks_meta)

        needed = compute_needed_chunks(kill_times_ms, chunks_meta, KILL_WINDOW_MS)
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
            try:
                gathered = await asyncio.wait_for(
                    asyncio.gather(*tasks, return_exceptions=True),
                    timeout=_CHUNK_TIMEOUT_S * len(to_download),
                )
            except TimeoutError:
                logger.warning(
                    "Match %s : timeout chunks (%ds)",
                    match_id[:8],
                    _CHUNK_TIMEOUT_S,
                )
                return result
            for item in gathered:
                if isinstance(item, Exception):
                    logger.debug("Chunk download exception : %s", item)
                    continue
                idx, data, start_ms, dur_ms = item
                if data is not None:
                    result[idx] = (data, start_ms, dur_ms)

        return result

    async def _download_chunk(
        self,
        ch,
        blob_prefix: str,
        match_cache: Path,
    ) -> tuple[int, bytes | None, int, int]:
        async with self._chunk_sem:
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
