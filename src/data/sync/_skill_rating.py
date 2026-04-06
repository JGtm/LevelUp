"""Mixin — calcul du skill rating (CSR classé + LUSR TrueSkill 2).

CSR : écrit directement depuis les données API pour les matchs classés.
LUSR : calculé séquentiellement via TrueSkill 2 pour les matchs non classés.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import TYPE_CHECKING, Any

import polars as pl

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

from src.data.sync.migrations import ensure_match_skill_rank_table

logger = logging.getLogger(__name__)

# Import conditionnel pour le calcul du LUSR (LevelUp Skill Rank)
try:
    from src.analysis.skill_rating import compute_skill_ratings_batch
    from src.analysis.skill_rating_config import (
        SKILL_TIERS,
        format_tier_label,
        get_tier_for_rating,
    )

    _LUSR_AVAILABLE = True
except ImportError:
    compute_skill_ratings_batch = None  # type: ignore[assignment]
    SKILL_TIERS = []  # type: ignore[assignment]
    format_tier_label = None  # type: ignore[assignment]
    get_tier_for_rating = None  # type: ignore[assignment]
    _LUSR_AVAILABLE = False

_ROMAN = {1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}

_CSR_UPSERT_SQL = """
INSERT INTO match_skill_rank
    (match_id, rating_type, rating_value, tier, tier_fr,
     sub_tier, tier_label, rating_delta, playlist_group,
     created_at, updated_at)
VALUES (?, 'CSR', ?, ?, ?, ?, ?, ?, 'ranked', ?, ?)
ON CONFLICT (match_id) DO UPDATE SET
    rating_type   = 'CSR',
    rating_value  = EXCLUDED.rating_value,
    tier          = EXCLUDED.tier,
    tier_fr       = EXCLUDED.tier_fr,
    sub_tier      = EXCLUDED.sub_tier,
    tier_label    = EXCLUDED.tier_label,
    rating_delta  = EXCLUDED.rating_delta,
    playlist_group = 'ranked',
    updated_at    = EXCLUDED.updated_at
"""


def _build_csr_tier_label(
    csr_tier_name: str | None,
    csr_sub_tier: int,
) -> tuple[str | None, str | None]:
    """Résout le tier FR + label depuis SKILL_TIERS."""
    if not _LUSR_AVAILABLE or not csr_tier_name:
        return None, None
    tier_obj = next(
        (t for t in SKILL_TIERS if t.name.lower() == csr_tier_name.lower()),
        None,
    )
    if not tier_obj:
        return None, None
    tier_fr = tier_obj.name_fr
    tier_label = (
        f"{tier_fr} {_ROMAN[csr_sub_tier]}" if csr_sub_tier and csr_sub_tier in _ROMAN else tier_fr
    )
    return tier_fr, tier_label


_LUSR_UPSERT_SQL = """
INSERT INTO match_skill_rank
    (match_id, rating_type, rating_value, rating_deviation,
     tier, tier_fr, sub_tier, tier_label,
     rating_delta, playlist_group, start_time, created_at, updated_at)
VALUES (?, 'LUSR', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (match_id) DO UPDATE SET
    rating_type       = 'LUSR',
    rating_value      = EXCLUDED.rating_value,
    rating_deviation  = EXCLUDED.rating_deviation,
    tier              = EXCLUDED.tier,
    tier_fr           = EXCLUDED.tier_fr,
    sub_tier          = EXCLUDED.sub_tier,
    tier_label        = EXCLUDED.tier_label,
    rating_delta      = EXCLUDED.rating_delta,
    playlist_group    = EXCLUDED.playlist_group,
    start_time        = COALESCE(match_skill_rank.start_time, EXCLUDED.start_time),
    updated_at        = EXCLUDED.updated_at
"""

_LUSR_MAX_DELTA = 100.0  # Guard-rail : cap ±100 pts par match


class SkillRatingMixin:
    """Méthodes de calcul du skill rating (CSR et LUSR)."""

    def _upsert_csr_rating(
        self: _SyncProtocol,
        match_id: str,
        skill_row: Any,
    ) -> None:
        """Écrit le CSR dans ``match_skill_rank`` pour un match classé."""
        post_csr = getattr(skill_row, "post_match_csr", None)
        if post_csr is None:
            return

        conn = self._get_connection()
        try:
            ensure_match_skill_rank_table(conn)

            pre_csr = getattr(skill_row, "pre_match_csr", None)
            csr_tier_name = getattr(skill_row, "csr_tier", None)
            csr_sub_tier = getattr(skill_row, "csr_sub_tier", None) or 0

            delta: float | None = None
            if pre_csr is not None:
                delta = post_csr - pre_csr

            tier_fr, tier_label = _build_csr_tier_label(csr_tier_name, csr_sub_tier)

            now = datetime.now(timezone.utc)
            conn.execute(
                _CSR_UPSERT_SQL,
                (
                    match_id,
                    post_csr,
                    csr_tier_name,
                    tier_fr,
                    csr_sub_tier,
                    tier_label,
                    delta,
                    now,
                    now,
                ),
            )
            logger.debug("CSR écrit dans match_skill_rank pour %s : %s", match_id, post_csr)
        except Exception as e:
            logger.warning("Erreur écriture CSR pour %s: %s", match_id, e)

    def _load_lusr_match_data(
        self: _SyncProtocol,
    ) -> tuple[pl.DataFrame, pl.DataFrame] | None:
        """Charge les matchs non classés et leurs participants pour le calcul LUSR."""
        shared_conn = self._get_shared_connection()
        if shared_conn is None or not self._xuid:
            logger.warning("shared_connection ou xuid manquant pour batch_compute_lusr")
            return None

        df_matches = shared_conn.execute(
            """
            SELECT
                mr.match_id, mr.start_time, mr.playlist_name, mr.pair_name,
                mp.outcome, mp.kills, mp.deaths,
                mp.kills_expected, mp.deaths_expected,
                mp.damage_dealt, mp.damage_taken, mp.accuracy,
                mp.team_id
            FROM match_registry mr
            JOIN match_participants mp ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND COALESCE(mr.is_ranked, FALSE) = FALSE
              AND COALESCE(mr.is_firefight, FALSE) = FALSE
              AND mr.start_time IS NOT NULL
              AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
            ORDER BY mr.start_time ASC
            """,
            [self._xuid],
        ).pl()

        if df_matches.is_empty():
            logger.info("batch_compute_lusr : aucun match non classé trouvé")
            return None

        match_ids = df_matches["match_id"].to_list()
        df_participants = shared_conn.execute(
            f"""
            SELECT match_id, xuid, team_id, kills_expected, deaths_expected
            FROM match_participants
            WHERE match_id IN ({",".join("?" * len(match_ids))})
            """,
            match_ids,
        ).pl()

        return df_matches, df_participants

    def _upsert_lusr_ratings(  # noqa: PLR0913
        self: _SyncProtocol,
        conn: Any,
        ratings_df: pl.DataFrame,
        df_matches: pl.DataFrame,
        existing_csr_ids: set[str],
        existing_lusr_ids: set[str],
        *,
        force: bool,
        seed_ratings: dict[str, float] | None = None,
    ) -> int:
        """Insère/met à jour les ratings LUSR dans match_skill_rank.

        Args:
            seed_ratings: Ratings initiaux par playlist_group pour calculer le
                delta du premier nouveau match d'un batch incrémental.
                En mode force, doit être None.
        """
        now = datetime.now(timezone.utc)
        start_time_map: dict[str, Any] = {}
        if "match_id" in df_matches.columns and "start_time" in df_matches.columns:
            for m_row in df_matches.select(["match_id", "start_time"]).iter_rows(named=True):
                start_time_map[m_row["match_id"]] = m_row["start_time"]

        # Seed prev_rating depuis les derniers ratings connus pour que le delta
        # du premier nouveau match d'un batch soit cohérent avec la valeur stockée.
        prev_rating: dict[str, float] = dict(seed_ratings) if seed_ratings else {}
        rows_to_insert: list[tuple] = []

        for row in ratings_df.iter_rows(named=True):
            mid = row["match_id"]
            rating_value: float = row["rating_value"]
            rating_dev: float | None = row.get("rating_deviation")
            pg: str = row.get("playlist_group") or "social"

            delta: float | None = None
            if pg in prev_rating:
                raw_delta = rating_value - prev_rating[pg]
                if abs(raw_delta) > _LUSR_MAX_DELTA:
                    capped = _LUSR_MAX_DELTA if raw_delta > 0 else -_LUSR_MAX_DELTA
                    logger.warning(
                        "LUSR guard-rail: delta %+.1f capé à %+.0f pour %s (groupe %s)",
                        raw_delta,
                        capped,
                        mid,
                        pg,
                    )
                    delta = capped
                    rating_value = prev_rating[pg] + delta
                else:
                    delta = raw_delta
            prev_rating[pg] = rating_value

            if mid in existing_csr_ids:
                continue
            if not force and mid in existing_lusr_ids:
                continue

            tier_obj, sub = get_tier_for_rating(rating_value)
            rows_to_insert.append(
                (
                    mid,
                    rating_value,
                    rating_dev,
                    tier_obj.name if tier_obj else None,
                    tier_obj.name_fr if tier_obj else None,
                    sub or 0,
                    format_tier_label(rating_value) if tier_obj else None,
                    delta,
                    pg,
                    start_time_map.get(mid),
                    now,
                    now,
                )
            )

        updates = len(rows_to_insert)
        if rows_to_insert:
            conn.executemany(_LUSR_UPSERT_SQL, rows_to_insert)
            conn.commit()
            logger.info("LUSR batch : %s matchs mis à jour", updates)
        return updates

    def _load_existing_lusr_states(self: _SyncProtocol, conn: Any) -> dict:
        """Charge les derniers états LUSR (mu, sigma) par playlist_group depuis la DB."""
        existing_states: dict = {}
        try:
            from src.analysis._trueskill_math import PlayerState as _PS
            from src.analysis.skill_rating_config import INITIAL_SIGMA as _S0

            last_rows = conn.execute(
                """
                SELECT msr.playlist_group, msr.rating_value, msr.rating_deviation
                FROM match_skill_rank msr
                JOIN (
                    SELECT playlist_group, MAX(start_time) AS max_st
                    FROM match_skill_rank
                    WHERE rating_type = 'LUSR'
                    GROUP BY playlist_group
                ) last ON msr.playlist_group = last.playlist_group
                        AND msr.start_time = last.max_st
                WHERE msr.rating_type = 'LUSR'
                """
            ).fetchall()
            for pg, rv, rd in last_rows:
                existing_states[pg] = _PS(mu=rv, sigma=rd if rd is not None else _S0)
        except Exception as _e:
            logger.debug("Impossible de charger les états LUSR existants : %s", _e)
        return existing_states

    def batch_compute_lusr(self: _SyncProtocol, *, force: bool = False) -> int:  # noqa: C901
        """Calcule le LUSR pour tous les matchs non classés sans rating LUSR.

        Traitement **séquentiel** (TrueSkill 2) : chaque match dépend du précédent.
        En mode incrémental, seuls les **nouveaux matchs** (absents de match_skill_rank)
        sont passés à TrueSkill. existing_states sert de seed (état après le dernier
        match déjà calculé) — évite la cascade de dérive entre syncs séparés.
        En mode force, recalcule toute l'historique depuis INITIAL_MU.

        Seuls les matchs non classés, non-Firefight sont traités.
        Les matchs classés utilisent le CSR de l'API (géré par _upsert_csr_rating).

        Args:
            force: Si True, recalcule et réécrit tous les matchs LUSR.

        Returns:
            Nombre de matchs mis à jour.
        """
        logger.info(
            "batch_compute_lusr : démarrage (xuid=%s, force=%s)",
            getattr(self, "_xuid", None),
            force,
        )
        if not _LUSR_AVAILABLE:
            logger.debug("Modules LUSR non disponibles, skip batch_compute_lusr")
            return 0

        try:
            data = self._load_lusr_match_data()
            if data is None:
                return 0
            df_matches, df_participants = data

            conn = self._get_connection()
            ensure_match_skill_rank_table(conn)

            # Protéger les matchs CSR officiels
            existing_csr_ids: set[str] = set()
            try:
                csr_df = conn.execute(
                    "SELECT match_id FROM match_skill_rank WHERE rating_type = 'CSR'"
                ).pl()
                existing_csr_ids = set(csr_df["match_id"].to_list())
            except Exception:
                existing_csr_ids = set()

            # Mode incrémental : identifier les LUSR existants
            existing_lusr_ids: set[str] = set()
            if not force:
                try:
                    existing_df = conn.execute(
                        "SELECT match_id FROM match_skill_rank WHERE rating_type = 'LUSR'"
                    ).pl()
                    existing_lusr_ids = set(existing_df["match_id"].to_list())
                except Exception:
                    existing_lusr_ids = set()

            # Seed depuis les derniers ratings connus en DB pour le mode incrémental.
            # existing_states représente l'état APRÈS le dernier match déjà calculé.
            # En force, on repart depuis INITIAL_MU (pas de seed).
            existing_states: dict = {} if force else self._load_existing_lusr_states(conn)

            # Capturer seed_ratings AVANT d'appeler compute_skill_ratings_batch :
            # compute_skill_ratings_batch modifie existing_states en place →
            # lire state.mu après l'appel donnerait la valeur post-nouveau-match,
            # pas la valeur pré-nouveau-match qui sert de seed correct pour le delta.
            seed_ratings: dict[str, float] | None = None
            if existing_states and not force:
                seed_ratings = {pg: state.mu for pg, state in existing_states.items()}

            # En mode incrémental : ne passer à TrueSkill QUE les nouveaux matchs.
            # Recomputer toute l'historique avec un seed décalé causait une cascade
            # de dérive (rating_value s'effondrait de ~160 pts à chaque sync séparé).
            # existing_states fournit le point de départ correct pour les nouveaux matchs.
            if not force:
                already_computed = existing_lusr_ids | existing_csr_ids
                df_matches = df_matches.filter(~pl.col("match_id").is_in(already_computed))
                if df_matches.is_empty():
                    logger.info("batch_compute_lusr : aucun nouveau match à calculer")
                    return 0
                new_match_ids = set(df_matches["match_id"].to_list())
                df_participants = df_participants.filter(pl.col("match_id").is_in(new_match_ids))

            # Calculer les ratings via TrueSkill 2 batch
            ratings_df = compute_skill_ratings_batch(
                df_matches,
                df_participants,
                existing_states=existing_states if existing_states else None,
            )
            if ratings_df.is_empty():
                return 0

            return self._upsert_lusr_ratings(
                conn,
                ratings_df,
                df_matches,
                existing_csr_ids,
                existing_lusr_ids,
                force=force,
                seed_ratings=seed_ratings,
            )

        except Exception as e:
            logger.warning("Erreur batch_compute_lusr : %s", e)
            return 0
