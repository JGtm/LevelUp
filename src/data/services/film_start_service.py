"""Service de détection du début de match via chunks filmshell.

Orchestre le chargement/téléchargement des chunks REPLICATION_DATA
(chunk_01 + éventuellement chunk_02) et délègue le calcul à
``src.analysis.spawn_detection``.

Ce service est appelé automatiquement après l'extraction weapon_kills
pour tout nouveau match ayant un film disponible.
"""

from __future__ import annotations

import logging
from pathlib import Path

import duckdb

from src.analysis.spawn_detection import estimate_film_match_start_ms
from src.data.services._film_manifest_cache import load_manifest_cache
from src.ports.api import HaloAPIPort

logger = logging.getLogger(__name__)

#: Chunks à analyser pour la détection de spawn (0-20s et 20-40s).
#: chunk_01 suffit pour la plupart des matchs ; chunk_02 couvre les
#: matchs à chargement lent ou parties démarrant après 20s.
_SPAWN_CHUNK_INDICES: tuple[int, ...] = (1, 2)

#: Nombre minimum de joueurs non-AFK requis pour valider l'estimation.
_MIN_PLAYERS: int = 3

#: Type de chunk REPLICATION_DATA dans le manifest
_REPLICATION_DATA_TYPE: int = 2


class FilmStartService:
    """Détecte et persiste film_match_start_ms dans match_registry."""

    def __init__(
        self,
        api: HaloAPIPort,
        conn: duckdb.DuckDBPyConnection,
        cache_dir: Path,
        manifest_dir: Path,
    ) -> None:
        self._api = api
        self._conn = conn
        self._cache_dir = cache_dir
        self._manifest_dir = manifest_dir

    async def compute_and_write(self, match_id: str) -> int | None:
        """Détecte et écrit film_match_start_ms pour ce match.

        Si la valeur est déjà présente dans match_registry → skip.
        Utilise le manifest déjà mis en cache par WeaponExtractionService.
        Télécharge chunk_01 (et chunk_02 si nécessaire) depuis le cache
        disque ou l'API.

        Returns:
            Valeur écrite en ms, ou None si non détectée.
        """
        if self._already_done(match_id):
            logger.debug("film_start %s : déjà calculé, skip", match_id[:8])
            return None

        chunks = await self._load_spawn_chunks(match_id)
        if not chunks:
            logger.debug("film_start %s : aucun chunk disponible", match_id[:8])
            return None

        estimate_ms = estimate_film_match_start_ms(
            chunks,
            min_players=_MIN_PLAYERS,
            api_first_event_ms=self._get_first_event_ms(match_id),
        )
        if estimate_ms is None:
            logger.debug("film_start %s : aucun mouvement détecté", match_id[:8])
            return None

        self._write_result(match_id, estimate_ms)
        logger.debug("film_start %s : %dms écrit", match_id[:8], estimate_ms)
        return estimate_ms

    # ── Helpers ──────────────────────────────────────────────────────────

    def _already_done(self, match_id: str) -> bool:
        """Vérifie si film_match_start_ms est déjà renseigné pour ce match."""
        try:
            row = self._conn.execute(
                "SELECT film_match_start_ms FROM match_registry WHERE match_id = ?",
                (match_id,),
            ).fetchone()
            return row is not None and row[0] is not None
        except Exception as exc:
            logger.debug("film_start _already_done %s : %s", match_id[:8], exc)
            return False

    async def _load_spawn_chunks(
        self,
        match_id: str,
    ) -> dict[int, tuple[bytes, int, int]]:
        """Charge les chunks de spawn depuis le cache disque ou l'API.

        N'appelle l'API que si le manifest est déjà en cache (= le film
        existe). Si le manifest n'est pas encore en cache, retourne {}.

        Returns:
            {chunk_index: (data, start_ms, dur_ms)}
        """
        cached = load_manifest_cache(self._manifest_dir, match_id)
        if cached is None:
            return {}

        blob_prefix, chunks_meta = cached
        match_cache = self._cache_dir / match_id[:8]

        result: dict[int, tuple[bytes, int, int]] = {}
        to_download = []

        for ch in sorted(chunks_meta, key=lambda c: c.index):
            if ch.chunk_type.value != _REPLICATION_DATA_TYPE:
                continue
            if ch.index not in _SPAWN_CHUNK_INDICES:
                continue

            start_ms: int = ch.chunk_start_time_offset_milliseconds
            dur_ms: int = ch.duration_milliseconds

            cache_path = match_cache / f"chunk_{ch.index:02d}.bin"
            if cache_path.exists():
                result[ch.index] = (cache_path.read_bytes(), start_ms, dur_ms)
            else:
                to_download.append((ch, start_ms, dur_ms))

        for ch, start_ms, dur_ms in to_download:
            url = blob_prefix + ch.file_relative_path.lstrip("/")
            try:
                data = await self._api.download_film_chunk(url)
            except Exception as exc:
                logger.debug("film_start %s chunk_%02d download: %s", match_id[:8], ch.index, exc)
                data = None
            if data:
                match_cache.mkdir(parents=True, exist_ok=True)
                (match_cache / f"chunk_{ch.index:02d}.bin").write_bytes(data)
                result[ch.index] = (data, start_ms, dur_ms)

        return result

    def _write_result(self, match_id: str, film_match_start_ms: int) -> None:
        """Écrit film_match_start_ms dans match_registry (idempotente)."""
        self._conn.execute(
            "UPDATE match_registry SET film_match_start_ms = ? WHERE match_id = ?",
            [film_match_start_ms, match_id],
        )

    def _get_first_event_ms(self, match_id: str) -> float | None:
        """Retourne le timestamp du premier kill/death pour ce match (ms).

        Contrainte dure : aucun kill/death ne peut précéder le début du match.
        Utilisé pour détecter et corriger les estimations trop précoces.
        """
        try:
            row = self._conn.execute(
                """
                SELECT MIN(time_ms)
                FROM highlight_events
                WHERE match_id = ?
                  AND event_type IN ('kill', 'death')
                """,
                (match_id,),
            ).fetchone()
            return float(row[0]) if row and row[0] is not None else None
        except Exception as exc:
            logger.debug("film_start _get_first_event_ms %s : %s", match_id[:8], exc)
            return None

