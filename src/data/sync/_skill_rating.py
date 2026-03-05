"""Mixin — calcul du skill rating (CSR classé + LUSR TrueSkill 2).

CSR : écrit directement depuis les données API pour les matchs classés.
LUSR : calculé séquentiellement via TrueSkill 2 pour les matchs non classés.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import TYPE_CHECKING, Any

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


class SkillRatingMixin:
    """Méthodes de calcul du skill rating (CSR et LUSR)."""

    def _upsert_csr_rating(
        self: _SyncProtocol,
        match_id: str,
        skill_row: Any,
    ) -> None:
        """Écrit le CSR dans ``match_skill_rank`` pour un match classé.

        Appelé lors du sync d'un match classé avec un ``SkillParticipantUpdate``
        contenant des données CSR (``post_match_csr`` non-null).

        Aucune action si ``post_match_csr`` est None ou si les modules LUSR
        ne sont pas disponibles.

        Args:
            match_id: ID du match.
            skill_row: ``SkillParticipantUpdate`` du joueur suivi.
        """
        post_csr = getattr(skill_row, "post_match_csr", None)
        if post_csr is None:
            return

        conn = self._get_connection()
        try:
            ensure_match_skill_rank_table(conn)

            pre_csr = getattr(skill_row, "pre_match_csr", None)
            csr_tier_name = getattr(skill_row, "csr_tier", None)
            csr_sub_tier = getattr(skill_row, "csr_sub_tier", None) or 0

            # Calcul delta intra-match (post - pre)
            delta: float | None = None
            if pre_csr is not None:
                delta = post_csr - pre_csr

            # Tier FR + label depuis SKILL_TIERS si disponibles
            tier_fr: str | None = None
            tier_label: str | None = None
            if _LUSR_AVAILABLE and csr_tier_name:
                tier_obj = next(
                    (t for t in SKILL_TIERS if t.name.lower() == csr_tier_name.lower()),
                    None,
                )
                if tier_obj:
                    tier_fr = tier_obj.name_fr
                    _roman = {1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}
                    tier_label = (
                        f"{tier_fr} {_roman[csr_sub_tier]}"
                        if csr_sub_tier and csr_sub_tier in _roman
                        else tier_fr
                    )

            now = datetime.now(timezone.utc)
            conn.execute(
                """
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
                """,
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

    def batch_compute_lusr(self: _SyncProtocol, *, force: bool = False) -> int:  # noqa: C901, PLR0912
        """Calcule le LUSR pour tous les matchs non classés sans rating LUSR.

        Traitement **séquentiel** (TrueSkill 2) : chaque match dépend du précédent.
        En mode incrémental, calcule tout mais n'écrit que les matchs sans entrée.
        En mode force, recalcule et écrase tout.

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
            shared_conn = self._get_shared_connection()
            if shared_conn is None or not self._xuid:
                logger.warning("shared_connection ou xuid manquant pour batch_compute_lusr")
                return 0

            conn = self._get_connection()
            ensure_match_skill_rank_table(conn)

            # 1. Charger tous les matchs non classés, non-Firefight du joueur (ordre ASC)
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
                return 0

            match_ids = df_matches["match_id"].to_list()

            # 2. Charger tous les participants de ces matchs (pour estimation μ)
            df_participants = shared_conn.execute(
                f"""
                SELECT match_id, xuid, team_id, kills_expected, deaths_expected
                FROM match_participants
                WHERE match_id IN ({",".join("?" * len(match_ids))})
                """,
                match_ids,
            ).pl()

            # 3. En mode incrémental, identifier les match_ids déjà dans match_skill_rank
            existing_lusr_ids: set[str] = set()
            if not force:
                try:
                    existing_df = conn.execute(
                        "SELECT match_id FROM match_skill_rank WHERE rating_type = 'LUSR'"
                    ).pl()
                    existing_lusr_ids = set(existing_df["match_id"].to_list())
                except Exception:
                    existing_lusr_ids = set()

            # 4. Calculer les ratings via TrueSkill 2 batch (séquentiel complet)
            ratings_df = compute_skill_ratings_batch(df_matches, df_participants)
            if ratings_df.is_empty():
                return 0

            # 5. Préparer les upserts
            now = datetime.now(timezone.utc)
            updates = 0

            # Map match_id → start_time (pour stocker dans match_skill_rank)
            start_time_map: dict[str, Any] = {}
            if "match_id" in df_matches.columns and "start_time" in df_matches.columns:
                for m_row in df_matches.select(["match_id", "start_time"]).iter_rows(named=True):
                    start_time_map[m_row["match_id"]] = m_row["start_time"]

            # Tracker le rating précédent par playlist_group (pour rating_delta)
            prev_rating: dict[str, float] = {}
            _LUSR_MAX_DELTA = 100.0  # Guard-rail : cap ±100 pts par match

            for row in ratings_df.iter_rows(named=True):
                mid = row["match_id"]
                rating_value: float = row["rating_value"]
                rating_dev: float | None = row.get("rating_deviation")
                pg: str = row.get("playlist_group") or "social"

                # Delta vs match précédent dans le même playlist_group
                delta: float | None = None
                if pg in prev_rating:
                    raw_delta = rating_value - prev_rating[pg]
                    # Guard-rail : limiter le delta à ±100 pts par match
                    if abs(raw_delta) > _LUSR_MAX_DELTA:
                        capped = _LUSR_MAX_DELTA if raw_delta > 0 else -_LUSR_MAX_DELTA
                        logger.warning(
                            "LUSR guard-rail: delta %+.1f capé à %+.0f pour %s (groupe %s)",
                            raw_delta,
                            capped,
                            mid,
                            pg,
                        )
                        delta = _LUSR_MAX_DELTA if raw_delta > 0 else -_LUSR_MAX_DELTA
                        rating_value = prev_rating[pg] + delta
                    else:
                        delta = raw_delta
                prev_rating[pg] = rating_value

                # Mode incrémental : sauter si déjà présent
                if not force and mid in existing_lusr_ids:
                    continue

                # Tier / label
                tier_obj, sub = get_tier_for_rating(rating_value)
                tier_name = tier_obj.name if tier_obj else None
                tier_fr = tier_obj.name_fr if tier_obj else None
                tier_label = format_tier_label(rating_value) if tier_obj else None

                match_start_time = start_time_map.get(mid)
                conn.execute(
                    """
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
                    """,
                    (
                        mid,
                        rating_value,
                        rating_dev,
                        tier_name,
                        tier_fr,
                        sub or 0,
                        tier_label,
                        delta,
                        pg,
                        match_start_time,
                        now,
                        now,
                    ),
                )
                updates += 1

            if updates:
                conn.commit()
                logger.info("LUSR batch : %s matchs mis à jour", updates)

            return updates

        except Exception as e:
            logger.warning("Erreur batch_compute_lusr : %s", e)
            return 0
