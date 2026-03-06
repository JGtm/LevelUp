"""Stratégies de backfill spécifiques.

Chaque stratégie est une fonction autonome qui opère sur une connexion DuckDB.
Aucun commit : le commit est géré par l'orchestrateur.
"""

from __future__ import annotations

import contextlib
import logging
from typing import Any

logger = logging.getLogger(__name__)


# ─────────────────────────────────────────────────────────────────────────────
# Killer / Victim pairs
# ─────────────────────────────────────────────────────────────────────────────


def backfill_killer_victim_pairs(
    conn: Any,
    me_xuid: str,
    *,
    force: bool = False,
    shared_conn: Any | None = None,
) -> int:
    """Extrait les paires killer/victim depuis highlight_events vers shared.

    En v5, cette table est mutualisée dans shared_matches.duckdb car les
    paires sont identiques quel que soit le joueur POV.

    Mode incrémental par défaut : ne traite que les matchs qui n'ont pas
    encore de paires dans killer_victim_pairs.
    Mode force : DROP + recréation complète de la table.

    Args:
        conn: Connexion DuckDB joueur (utilisée comme fallback pour highlight_events).
        me_xuid: XUID du joueur principal (pour référence).
        force: Si True, reconstruit toute la table.
        shared_conn: Connexion vers shared_matches.duckdb (v5).
            Si fourni, lit/écrit depuis shared. Sinon, fallback local.

    Returns:
        Nombre de paires insérées.
    """
    from src.analysis.killer_victim import KVPair, compute_killer_victim_pairs

    # Déterminer la connexion cible (shared ou locale)
    target_conn = shared_conn if shared_conn is not None else conn
    events_source = "highlight_events"  # même nom dans les deux DBs

    # highlight_events utilise les colonnes xuid/gamertag
    events_xuid_expr = "xuid"
    events_gt_expr = "gamertag"

    if force:
        target_conn.execute("DROP TABLE IF EXISTS killer_victim_pairs")
        logger.info("Table killer_victim_pairs supprimée (mode --force)")

    # Créer la table si elle n'existe pas
    target_conn.execute("""
        CREATE TABLE IF NOT EXISTS killer_victim_pairs (
            match_id VARCHAR NOT NULL,
            killer_xuid VARCHAR NOT NULL,
            killer_gamertag VARCHAR,
            victim_xuid VARCHAR NOT NULL,
            victim_gamertag VARCHAR,
            kill_count INTEGER DEFAULT 1,
            time_ms INTEGER,
            is_validated BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    with contextlib.suppress(Exception):
        target_conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_kv_match ON killer_victim_pairs(match_id)"
        )
        target_conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_kv_killer ON killer_victim_pairs(killer_xuid)"
        )
        target_conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_kv_victim ON killer_victim_pairs(victim_xuid)"
        )

    # Charger les matchs avec highlight events kill/death
    # qui n'ont PAS encore de paires (mode incrémental)
    # highlight_events et killer_victim_pairs sont dans la même connexion (target_conn)
    read_conn = target_conn
    try:
        matches = read_conn.execute(f"""
            SELECT DISTINCT he.match_id
            FROM {events_source} he
            WHERE LOWER(he.event_type) IN ('kill', 'death')
              AND NOT EXISTS (
                  SELECT 1 FROM killer_victim_pairs kvp
                  WHERE kvp.match_id = he.match_id
              )
        """).fetchall()
    except Exception as e:
        logger.warning(f"Erreur lecture highlight_events: {e}")
        return 0

    if not matches:
        logger.info(
            "Aucun nouveau match à traiter pour killer/victim (incrémental, tous déjà traités)"
        )
        return 0

    logger.info(f"Trouvé {len(matches)} matchs à traiter pour paires killer/victim")
    total_pairs = 0
    skipped_no_pairs = 0
    logged_debug = False

    for (match_id,) in matches:
        try:
            events = read_conn.execute(
                f"""
                SELECT event_type, time_ms, {events_xuid_expr}, {events_gt_expr}
                FROM {events_source}
                WHERE match_id = ?
                  AND LOWER(event_type) IN ('kill', 'death')
                ORDER BY time_ms
                """,
                [match_id],
            ).fetchall()
        except Exception as e:
            logger.warning(f"Impossible de charger les events pour match {match_id}: {e}")
            continue

        if not events:
            continue

        event_dicts = [
            {
                "event_type": row[0],
                "time_ms": row[1],
                "xuid": row[2],
                "gamertag": row[3],
            }
            for row in events
        ]

        kills_count = sum(
            1 for e in event_dicts if str(e.get("event_type") or "").lower() == "kill"
        )
        deaths_count = sum(
            1 for e in event_dicts if str(e.get("event_type") or "").lower() == "death"
        )

        is_first_match = not logged_debug
        if is_first_match:
            logged_debug = True
            sample_types = {str(e.get("event_type")) for e in event_dicts[:10]}
            logger.info(f"  [DEBUG] Sample event_types: {sample_types}")
            logger.info(
                f"  [DEBUG] Match {match_id[:20]}...: {len(events)} events, "
                f"{kills_count} kills, {deaths_count} deaths"
            )

        if kills_count == 0 or deaths_count == 0:
            skipped_no_pairs += 1
            continue

        pairs: list[KVPair] = compute_killer_victim_pairs(event_dicts, tolerance_ms=5)

        if is_first_match and pairs:
            logger.info(f"  [DEBUG] Paires calculées: {len(pairs)}")
            logger.info(
                f"  [DEBUG] Première paire: killer={pairs[0].killer_xuid}, "
                f"victim={pairs[0].victim_xuid}, time={pairs[0].time_ms}"
            )

        if not pairs:
            skipped_no_pairs += 1
            continue

        insert_errors = 0
        for p in pairs:
            try:
                target_conn.execute(
                    """
                    INSERT INTO killer_victim_pairs
                    (match_id, killer_xuid, killer_gamertag, victim_xuid,
                     victim_gamertag, kill_count, time_ms)
                    VALUES (?, ?, ?, ?, ?, 1, ?)
                    """,
                    [
                        match_id,
                        p.killer_xuid,
                        p.killer_gamertag,
                        p.victim_xuid,
                        p.victim_gamertag,
                        p.time_ms,
                    ],
                )
                total_pairs += 1
            except Exception as e:
                insert_errors += 1
                if insert_errors == 1:
                    logger.warning(f"  Erreur INSERT: {e}")

    if skipped_no_pairs > 0:
        logger.info(
            f"  Matchs skippés (pas de kills/deaths ou algorithme vide): {skipped_no_pairs}"
        )

    return total_pairs


# ─────────────────────────────────────────────────────────────────────────────
# CSR — snapshot via get_playlist_csr (v5.3)
# ─────────────────────────────────────────────────────────────────────────────

#: Playlists classées Halo Infinite (Ranked Arena + Ranked Slayer)
_RANKED_PLAYLISTS: dict[str, str] = {
    "Ranked Arena": "edfef3ac-9cbe-4fa2-b949-8f29deafd483",
    "Ranked Slayer": "dcb2e24e-05fb-4390-8076-32a0cdb4326e",
}


async def fetch_current_csr_for_player(
    conn: Any,
    xuid: str,
    api_client: Any,
) -> dict[str, Any]:
    """Récupère le CSR actuel du joueur via l'API et le stocke dans skill_history.

    Appelle ``get_playlist_csr`` pour Ranked Arena et Ranked Slayer.
    Stocke le ``all_time_max`` de chaque playlist comme snapshot dans
    ``skill_history`` (recorded_at = now()).

    Args:
        conn: Connexion DuckDB vers stats.duckdb du joueur.
        xuid: XUID du joueur.
        api_client: Instance ``SPNKrAPIClient`` déjà authentifiée.

    Returns:
        Dict ``{playlist_name: {csr, tier, sub_tier, value}}`` pour
        les playlists où le joueur a un historique (all_time_max > 0).
    """
    from datetime import datetime, timezone

    from src.data.sync.migrations import ensure_skill_history_table

    results: dict[str, Any] = {}

    # S'assurer que skill_history existe (DDL centralisé dans migrations.py)
    ensure_skill_history_table(conn)

    now = datetime.now(timezone.utc)

    for playlist_name, playlist_id in _RANKED_PLAYLISTS.items():
        try:
            resp = await api_client.client.skill.get_playlist_csr(playlist_id, [xuid])
            data = await resp.parse()

            for entry in data.value:
                if not hasattr(entry, "result") or entry.result is None:
                    continue
                atm = entry.result.all_time_max
                if atm is None or getattr(atm, "value", -1) <= 0:
                    continue

                csr_value = atm.value
                tier_name = atm.tier.value if hasattr(atm.tier, "value") else str(atm.tier)
                sub_tier_raw = atm.sub_tier
                # SubTier enum: I=0, II=1, … VI=5  → division affichée : 1-6
                division = (sub_tier_raw.value + 1) if hasattr(sub_tier_raw, "value") else 1

                conn.execute(
                    """
                    INSERT INTO skill_history
                        (playlist_id, recorded_at, csr, tier, division, matches_played)
                    VALUES (?, ?, ?, ?, ?, 0)
                    """,
                    [playlist_id, now, csr_value, tier_name, division],
                )

                results[playlist_name] = {
                    "csr": csr_value,
                    "tier": tier_name,
                    "sub_tier": division,
                    "playlist_id": playlist_id,
                }
                logger.info(
                    f"CSR snapshot {playlist_name}: {tier_name} {division} "
                    f"({csr_value}) pour xuid={xuid}"
                )
        except Exception as e:
            logger.warning(f"fetch_current_csr_for_player {playlist_name}: {e}")

    if results:
        conn.commit()

    return results


def get_best_csr_for_player(conn: Any) -> float | None:
    """Retourne le meilleur CSR historique du joueur depuis skill_history.

    Cherche le MAX(csr) sur toutes les playlists et tous les snapshots.
    Retourne None si aucun CSR n'est disponible.
    """
    try:
        row = conn.execute("SELECT MAX(csr) FROM skill_history WHERE csr > 0").fetchone()
        return float(row[0]) if row and row[0] is not None else None
    except Exception:
        return None


# ─────────────────────────────────────────────────────────────────────────────
# LUSR — LevelUp Skill Rank (v5.2)
# ─────────────────────────────────────────────────────────────────────────────


def compute_lusr_for_player(
    conn: Any,
    db_path: Any,
    xuid: str,
    *,
    force: bool = False,
    shared_conn: Any | None = None,
) -> int:
    """Calcule et stocke le LUSR pour tous les matchs non classés du joueur.

    Traitement **séquentiel** (TrueSkill 2) : les matchs sont traités dans
    l'ordre chronologique et chaque résultat dépend du précédent.
    En mode incrémental, calcule tout mais n'écrit que les matchs sans LUSR.
    En mode force, recalcule et réécrit tous les matchs LUSR.

    Seuls les matchs non classés et non-Firefight sont traités.
    Les matchs classés utilisent le CSR fourni par l'API.

    Args:
        conn: Connexion DuckDB vers stats.duckdb du joueur.
        db_path: Chemin vers la DB joueur (utilisé pour dériver shared si shared_conn=None).
        xuid: XUID du joueur.
        force: Si True, recalcule et réécrit tous les matchs LUSR.
        shared_conn: Connexion vers shared_matches.duckdb (ouverte si None).

    Returns:
        Nombre de matchs mis à jour dans match_skill_rank.
    """
    try:
        from src.analysis.skill_rating import compute_skill_ratings_batch
        from src.analysis.skill_rating_config import format_tier_label, get_tier_for_rating
        from src.data.sync.migrations import ensure_match_skill_rank_table
    except ImportError as e:
        logger.warning(f"Modules LUSR non disponibles (skip): {e}")
        return 0

    # Ouvrir shared_conn si non fournie
    _owned_shared = False
    if shared_conn is None:
        try:
            from pathlib import Path

            import duckdb

            shared_path = (
                Path(__file__).resolve().parents[2] / "data" / "warehouse" / "shared_matches.duckdb"
            )
            if not shared_path.exists():
                logger.warning(f"shared_matches.duckdb introuvable: {shared_path}")
                return 0
            shared_conn = duckdb.connect(str(shared_path), read_only=False)
            _owned_shared = True
        except Exception as e:
            logger.warning(f"Impossible d'ouvrir shared_matches.duckdb: {e}")
            return 0

    try:
        ensure_match_skill_rank_table(conn)

        # 1. Charger tous les matchs non classés, non-Firefight du joueur (ordre ASC)
        try:
            import polars as pl  # noqa: F401
        except ImportError:
            logger.warning("Polars non disponible, skip LUSR")
            return 0

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
            [xuid],
        ).pl()

        if df_matches.is_empty():
            logger.info("compute_lusr_for_player : aucun match non classé trouvé")
            return 0

        match_ids_list = df_matches["match_id"].to_list()

        # 2. Charger tous les participants de ces matchs (pour estimation μ opposants)
        placeholders = ",".join("?" * len(match_ids_list))
        df_participants = shared_conn.execute(
            f"""
            SELECT match_id, xuid, team_id, kills_expected, deaths_expected
            FROM match_participants
            WHERE match_id IN ({placeholders})
            """,
            match_ids_list,
        ).pl()

        # 3. Identifier les matchs déjà traités (mode incrémental)
        existing_lusr_ids: set[str] = set()
        if not force:
            try:
                existing_df = conn.execute(
                    "SELECT match_id FROM match_skill_rank WHERE rating_type = 'LUSR'"
                ).pl()
                existing_lusr_ids = set(existing_df["match_id"].to_list())
            except Exception:
                existing_lusr_ids = set()

        # 4. Seed depuis CSR si disponible (toujours, pas seulement au premier run)
        # Bug fix : en mode incrémental, le seed était ignoré car
        # existing_lusr_ids n'était pas vide, ce qui faisait démarrer la
        # recomputation à μ=1500 au lieu du CSR réel du joueur.
        existing_states_seed: dict | None = None
        best_csr = get_best_csr_for_player(conn)
        if best_csr is not None:
            from src.analysis.skill_rating import PlayerState

            seeded = PlayerState.from_csr(best_csr)
            existing_states_seed = {
                group: PlayerState(mu=seeded.mu, sigma=seeded.sigma)
                for group in ("ranked", "arena", "btb", "tactical", "social", "fun")
            }
            logger.info(
                f"LUSR seed depuis CSR={best_csr:.0f} → mu={seeded.mu:.1f} "
                f"sigma={seeded.sigma:.1f}"
            )

        # 5. Calcul TrueSkill 2 batch (séquentiel complet sur tout l'historique)
        ratings_df = compute_skill_ratings_batch(
            df_matches, df_participants, existing_states=existing_states_seed
        )
        if ratings_df.is_empty():
            return 0

        # Map match_id → start_time depuis df_matches
        start_time_map: dict[str, object] = {}
        if "match_id" in df_matches.columns and "start_time" in df_matches.columns:
            for m_row in df_matches.select(["match_id", "start_time"]).iter_rows(named=True):
                start_time_map[m_row["match_id"]] = m_row["start_time"]

        # 5. UPSERT dans match_skill_rank
        from datetime import datetime, timezone

        now = datetime.now(timezone.utc)
        prev_rating: dict[str, float] = {}
        _LUSR_MAX_DELTA = 100.0  # Guard-rail : cap ±100 pts par match
        updates = 0

        for row in ratings_df.iter_rows(named=True):
            mid = row["match_id"]
            rating_value: float = row["rating_value"]
            rating_dev: float | None = row.get("rating_deviation")
            pg: str = row.get("playlist_group") or "social"

            # Delta vs match précédent dans le même groupe
            delta: float | None = None
            if pg in prev_rating:
                raw_delta = rating_value - prev_rating[pg]
                # Guard-rail : limiter le delta à ±100 pts par match
                if abs(raw_delta) > _LUSR_MAX_DELTA:
                    logger.warning(
                        f"LUSR guard-rail: delta {raw_delta:+.1f} capé à "
                        f"{_LUSR_MAX_DELTA if raw_delta > 0 else -_LUSR_MAX_DELTA:+.0f} "
                        f"pour {mid} (groupe {pg})"
                    )
                    delta = _LUSR_MAX_DELTA if raw_delta > 0 else -_LUSR_MAX_DELTA
                    rating_value = prev_rating[pg] + delta
                else:
                    delta = raw_delta
            prev_rating[pg] = rating_value

            # Mode incrémental : sauter si déjà présent
            # NOTE : prev_rating est mis à jour AVANT le skip pour que le prochain
            # match nouveau hérite du bon rating précédent (sinon delta=None)
            if not force and mid in existing_lusr_ids:
                continue

            # Tier info
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
                     rating_delta, playlist_group,
                     start_time, created_at, updated_at)
                VALUES (?, 'LUSR', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ON CONFLICT (match_id) DO UPDATE SET
                    rating_type      = 'LUSR',
                    rating_value     = EXCLUDED.rating_value,
                    rating_deviation = EXCLUDED.rating_deviation,
                    tier             = EXCLUDED.tier,
                    tier_fr          = EXCLUDED.tier_fr,
                    sub_tier         = EXCLUDED.sub_tier,
                    tier_label       = EXCLUDED.tier_label,
                    rating_delta     = EXCLUDED.rating_delta,
                    playlist_group   = EXCLUDED.playlist_group,
                    start_time       = COALESCE(match_skill_rank.start_time, EXCLUDED.start_time),
                    updated_at       = EXCLUDED.updated_at
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
            logger.info(f"LUSR : {updates} matchs mis à jour dans match_skill_rank")

        return updates

    except Exception as e:
        logger.warning(f"Erreur compute_lusr_for_player : {e}")
        return 0
    finally:
        if _owned_shared and shared_conn is not None:
            with contextlib.suppress(Exception):
                shared_conn.close()


# ─────────────────────────────────────────────────────────────────────────────
# CSR par match — backfill depuis l'API skill (v5.3)
# ─────────────────────────────────────────────────────────────────────────────


async def backfill_csr_for_player(
    conn: Any,
    db_path: Any,
    xuid: str,
    api_client: Any,
    *,
    force: bool = False,
    shared_conn: Any | None = None,
) -> int:
    """Backfill le CSR (Competitive Skill Rating) match par match depuis l'API skill.

    Pour chaque match classé du joueur, appelle ``get_skill_stats`` et extrait
    ``PostMatchCsr`` depuis ``RankRecap``. Écrit dans ``match_skill_rank`` avec
    ``rating_type = 'CSR'``.

    Un match ne peut avoir qu'un seul rating (PK ``match_id`` exclusive). Si le match
    a déjà un LUSR, il sera remplacé par le CSR (plus fiable car fourni par Halo).

    Args:
        conn: Connexion DuckDB vers stats.duckdb du joueur.
        db_path: Chemin vers la DB joueur (utilisé pour dériver shared si shared_conn=None).
        xuid: XUID du joueur.
        api_client: Instance ``SPNKrAPIClient`` déjà authentifiée.
        force: Si True, re-fetche le CSR même si déjà présent.
        shared_conn: Connexion vers shared_matches.duckdb (ouverte si None).

    Returns:
        Nombre de matchs écrits dans match_skill_rank.
    """
    from datetime import datetime, timezone
    from pathlib import Path

    import duckdb

    from src.analysis.skill_rating_config import (
        format_tier_label,
        get_playlist_group,
        get_tier_for_rating,
    )
    from src.data.sync.migrations import ensure_match_skill_rank_table
    from src.data.sync.transformers import transform_all_skill_stats

    # Ouvrir shared_conn si non fournie
    _owned_shared = False
    if shared_conn is None:
        try:
            shared_path = (
                Path(__file__).resolve().parents[2] / "data" / "warehouse" / "shared_matches.duckdb"
            )
            if not shared_path.exists():
                logger.warning(f"shared_matches.duckdb introuvable: {shared_path}")
                return 0
            shared_conn = duckdb.connect(str(shared_path), read_only=True)
            _owned_shared = True
        except Exception as e:
            logger.warning(f"Impossible d'ouvrir shared_matches.duckdb: {e}")
            return 0

    try:
        ensure_match_skill_rank_table(conn)

        # 1. Charger les matchs classés du joueur
        df_ranked = shared_conn.execute(
            """
            SELECT mr.match_id, mr.start_time, mr.playlist_name, mr.pair_name
            FROM match_registry mr
            JOIN match_participants mp ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND COALESCE(mr.is_ranked, FALSE) = TRUE
              AND mr.start_time IS NOT NULL
            ORDER BY mr.start_time ASC
            """,
            [xuid],
        ).fetchall()

        if not df_ranked:
            logger.info("backfill_csr_for_player : aucun match classé trouvé")
            return 0

        # 2. Identifier les matchs déjà traités (mode incrémental)
        existing_csr_ids: set[str] = set()
        if not force:
            try:
                rows = conn.execute(
                    "SELECT match_id FROM match_skill_rank WHERE rating_type = 'CSR'"
                ).fetchall()
                existing_csr_ids = {r[0] for r in rows}
            except Exception:
                existing_csr_ids = set()

        xuid_int: list[int] = []
        with contextlib.suppress(ValueError, TypeError):
            xuid_int = [int(xuid)]
        if not xuid_int:
            logger.warning(f"XUID invalide pour CSR backfill: {xuid!r}")
            return 0

        now = datetime.now(timezone.utc)
        updates = 0
        prev_csr: float | None = None

        for match_id, start_time, playlist_name, pair_name in df_ranked:
            if not force and match_id in existing_csr_ids:
                continue

            # 3. Appel API skill pour ce match
            try:
                skill_json = await api_client.get_skill_stats(match_id, xuid_int)
            except Exception as e:
                logger.debug(f"CSR API error match {match_id}: {e}")
                continue

            if not skill_json:
                logger.debug(f"Pas de skill_json pour match {match_id}")
                continue

            # 4. Extraire le CSR du joueur depuis transform_all_skill_stats
            skill_updates = transform_all_skill_stats(skill_json, match_id)
            player_skill = next((u for u in skill_updates if u.xuid == xuid), None)

            if player_skill is None or player_skill.post_match_csr is None:
                logger.debug(f"Pas de CSR dans skill_json pour match {match_id} xuid={xuid}")
                continue

            csr_value: float = player_skill.post_match_csr
            csr_tier_raw: str | None = player_skill.csr_tier
            csr_sub_raw: int | None = player_skill.csr_sub_tier

            # 5. Tier label depuis config
            tier_obj, sub = get_tier_for_rating(csr_value)
            tier_name = csr_tier_raw or (tier_obj.name if tier_obj else None)
            tier_fr = tier_obj.name_fr if tier_obj else None
            tier_label = format_tier_label(csr_value) if tier_obj else None
            sub_tier = csr_sub_raw if csr_sub_raw is not None else (sub or 0)

            # Delta CSR vs match précédent
            delta: float | None = None
            if prev_csr is not None:
                delta = csr_value - prev_csr
            prev_csr = csr_value

            pg = get_playlist_group(
                str(playlist_name) if playlist_name else None,
                str(pair_name) if pair_name else None,
            )

            # 6. UPSERT dans match_skill_rank (rating_type = 'CSR')
            conn.execute(
                """
                INSERT INTO match_skill_rank
                    (match_id, rating_type, rating_value, rating_deviation,
                     tier, tier_fr, sub_tier, tier_label,
                     rating_delta, playlist_group,
                     start_time, created_at, updated_at)
                VALUES (?, 'CSR', ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ON CONFLICT (match_id) DO UPDATE SET
                    rating_type    = 'CSR',
                    rating_value   = EXCLUDED.rating_value,
                    tier           = EXCLUDED.tier,
                    tier_fr        = EXCLUDED.tier_fr,
                    sub_tier       = EXCLUDED.sub_tier,
                    tier_label     = EXCLUDED.tier_label,
                    rating_delta   = EXCLUDED.rating_delta,
                    playlist_group = EXCLUDED.playlist_group,
                    start_time     = COALESCE(match_skill_rank.start_time, EXCLUDED.start_time),
                    updated_at     = EXCLUDED.updated_at
                """,
                (
                    match_id,
                    csr_value,
                    tier_name,
                    tier_fr,
                    sub_tier,
                    tier_label,
                    delta,
                    pg,
                    start_time,
                    now,
                    now,
                ),
            )
            updates += 1
            delta_str = f"{delta:.0f}" if delta is not None else "None"
            logger.debug(
                f"CSR match {match_id}: {tier_name} {sub_tier} ({csr_value:.0f}) delta={delta_str}"
            )

        if updates:
            conn.commit()
            logger.info(f"CSR : {updates} matchs écrits dans match_skill_rank")

        return updates

    except Exception as e:
        logger.warning(f"Erreur backfill_csr_for_player : {e}")
        return 0
    finally:
        if _owned_shared and shared_conn is not None:
            with contextlib.suppress(Exception):
                shared_conn.close()


# ─────────────────────────────────────────────────────────────────────────────
# End time
# ─────────────────────────────────────────────────────────────────────────────


def backfill_end_time(conn: Any, force: bool = False, *, shared_conn: Any) -> int:
    """Met à jour end_time (start_time + duration_seconds) dans shared.match_registry.

    Args:
        conn: Connexion DuckDB joueur (non utilisée, conservée pour compatibilité de signature).
        force: Si True, recalcule pour tous les matchs.
        shared_conn: Connexion shared_matches.duckdb (obligatoire).

    Returns:
        Nombre de lignes mises à jour.
    """
    try:
        where_clause = (
            "WHERE start_time IS NOT NULL AND duration_seconds IS NOT NULL"
            if force
            else "WHERE end_time IS NULL "
            "AND start_time IS NOT NULL "
            "AND duration_seconds IS NOT NULL"
        )
        cursor = shared_conn.execute(
            f"""
            UPDATE match_registry
            SET end_time = start_time + (duration_seconds * INTERVAL '1 SECOND')
            {where_clause}
            RETURNING match_id
            """
        )
        updated = cursor.fetchall()
        return len(updated)
    except Exception as e:
        logger.warning(f"Erreur backfill end_time: {e}")
        return 0


# ─────────────────────────────────────────────────────────────────────────────
# Performance score
# ─────────────────────────────────────────────────────────────────────────────

# Import conditionnel pour le calcul des scores de performance
try:
    import polars as pl

    from src.analysis.performance_config import MIN_MATCHES_FOR_RELATIVE
    from src.analysis.performance_score import compute_relative_performance_score

    PERFORMANCE_SCORE_AVAILABLE = True
except ImportError:
    PERFORMANCE_SCORE_AVAILABLE = False
    pl = None
    compute_relative_performance_score = None
    MIN_MATCHES_FOR_RELATIVE = 10


def compute_performance_score_for_match(
    conn: Any, match_id: str, *, shared_conn: Any, xuid: str, force: bool = False
) -> bool:
    """Calcule et met à jour le score de performance pour un match.

    Lit depuis shared.match_participants + match_registry,
    écrit dans player_match_enrichment (player DB).

    Architecture v5.1 :
        Le calcul utilise team_mmr et enemy_mmr directement depuis
        mp.enemy_mmr (corrigé v5.1 : remplace l'ancienne sous-requête
        corrélée qui calculait la moyenne de l'équipe adverse).
        Le rank_perf (composante de rang) utilise le delta MMR pour
        ajuster le score selon la difficulté de l'adversaire.

    Args:
        conn: Connexion DuckDB (player DB pour player_match_enrichment).
        match_id: ID du match.
        shared_conn: Connexion vers shared_matches.duckdb (obligatoire).
        xuid: XUID du joueur (obligatoire).
        force: Si True, recalcule même si le score existe déjà.

    Returns:
        True si le score a été calculé, False sinon.
    """
    if not PERFORMANCE_SCORE_AVAILABLE:
        return False

    try:
        # S'assurer que player_match_enrichment existe
        conn.execute("""
            CREATE TABLE IF NOT EXISTS player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                performance_score FLOAT,
                session_id VARCHAR,
                session_label VARCHAR,
                is_with_friends BOOLEAN,
                teammates_signature VARCHAR,
                known_teammates_count SMALLINT,
                friends_xuids VARCHAR,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)

        # Vérifier si le score existe déjà dans player_match_enrichment (sauf si force)
        if not force:
            existing = conn.execute(
                "SELECT performance_score FROM player_match_enrichment WHERE match_id = ?",
                (match_id,),
            ).fetchone()
            if existing and existing[0] is not None:
                return False

        # Lire depuis shared.match_participants + match_registry
        match_data = shared_conn.execute(
            """
            SELECT mp.match_id, mr.start_time, mp.kills, mp.deaths, mp.assists,
                   mp.kda, mp.accuracy, mp.time_played_seconds, mp.avg_life_seconds,
                   mp.personal_score, mp.damage_dealt, mp.rank, mp.team_mmr,
                   mp.enemy_mmr, mp.kills_expected, mp.deaths_expected
            FROM match_participants mp
            JOIN match_registry mr ON mr.match_id = mp.match_id
            WHERE mp.match_id = ? AND mp.xuid = ?
            """,
            (match_id, xuid),
        ).fetchone()

        if not match_data:
            return False

        match_start_time = match_data[1]
        if match_start_time is None:
            return False

        # Historique depuis shared
        try:
            history_df = shared_conn.execute(
                """
                SELECT
                    mp.match_id, mr.start_time, mp.kills, mp.deaths, mp.assists,
                    mp.kda, mp.accuracy, mp.time_played_seconds, mp.avg_life_seconds,
                    mp.personal_score, mp.damage_dealt, mp.rank, mp.team_mmr,
                    mp.enemy_mmr, mp.kills_expected, mp.deaths_expected
                FROM match_participants mp
                JOIN match_registry mr ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND mp.match_id != ?
                  AND mr.start_time IS NOT NULL
                  AND mr.start_time < ?
                ORDER BY mr.start_time ASC
                """,
                (xuid, match_id, match_start_time),
            ).pl()
        except Exception as e:
            logger.warning(f"Erreur chargement historique pour {match_id}: {e}")
            return False

        if history_df.is_empty() or len(history_df) < MIN_MATCHES_FOR_RELATIVE:
            return False

        # Dict pour le calcul du score
        match_dict = {
            "kills": match_data[2] or 0,
            "deaths": match_data[3] or 0,
            "assists": match_data[4] or 0,
            "kda": match_data[5],
            "accuracy": match_data[6],
            "time_played_seconds": match_data[7] or 600.0,
            "personal_score": match_data[9],
            "damage_dealt": match_data[10],
            "rank": match_data[11],
            "team_mmr": match_data[12],
            "enemy_mmr": match_data[13],
            "kills_expected": match_data[14],
            "deaths_expected": match_data[15],
        }

        score = compute_relative_performance_score(match_dict, history_df)

        if score is not None:
            # Écrire dans player_match_enrichment (player DB)
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id, performance_score) "
                "VALUES (?, ?) "
                "ON CONFLICT (match_id) DO UPDATE SET "
                "performance_score = EXCLUDED.performance_score, "
                "updated_at = now()",
                (match_id, score),
            )
            return True

        return False

    except Exception as e:
        logger.warning(f"Erreur calcul score performance pour {match_id}: {e}")
        return False


# ─────────────────────────────────────────────────────────────────────────────
# Participants Enrich (V5)
# ─────────────────────────────────────────────────────────────────────────────


async def backfill_participants_enrich(
    shared_conn: Any,
    *,
    xuid: str | None = None,
    max_matches: int | None = None,
    force: bool = False,
    requests_per_second: int = 5,
) -> int:
    """Backfill les colonnes étendues + MMR dans shared.match_participants.

    Colonnes stats : headshot_kills, max_killing_spree, kda, accuracy,
    time_played_seconds, grenade_kills, melee_kills, power_weapon_kills,
    personal_score, shots_fired, shots_hit, damage_dealt, damage_taken,
    avg_life_seconds, kills, deaths, assists.
    Colonnes MMR : team_mmr, kills_expected, kills_stddev, deaths_expected,
    deaths_stddev, assists_expected, assists_stddev.

    Args:
        shared_conn: Connexion en écriture vers shared_matches.duckdb.
        xuid: Si fourni, ne traiter que les matchs de ce joueur.
        max_matches: Nombre max de matchs à traiter.
        force: Si True, recalcule même si les colonnes existent déjà.
        requests_per_second: Rate limiting API.

    Returns:
        Nombre de matchs enrichis.
    """
    from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env
    from src.data.sync.transformers import (
        extract_participants,
        extract_xuids_from_match,
        transform_skill_stats,
    )

    # Détection des matchs à enrichir
    if force:
        query = (
            "SELECT DISTINCT mp.match_id "
            "FROM match_participants mp "
            "JOIN match_registry mr ON mp.match_id = mr.match_id "
        )
        params: list = []
        if xuid:
            query += "WHERE mp.xuid = ? "
            params.append(xuid)
        query += "ORDER BY mr.start_time DESC"
    else:
        # Détecter les matchs avec colonnes essentielles NULL
        # Note: team_mmr peut être NULL légitimement (API skill ne retourne pas toujours de données)
        # Guard: ne pas retraiter les matchs déjà marqués dans backfill_completed
        from src.data.sync.migrations import BACKFILL_FLAGS

        participants_bit = BACKFILL_FLAGS.get("participants", 0)
        avg_life_bit = BACKFILL_FLAGS.get("participants_avg_life", 0)
        guard_mask = participants_bit | avg_life_bit

        query = (
            "SELECT DISTINCT mp.match_id "
            "FROM match_participants mp "
            "JOIN match_registry mr ON mp.match_id = mr.match_id "
            f"WHERE (mp.headshot_kills IS NULL OR mp.kda IS NULL OR mp.avg_life_seconds IS NULL) "
            f"AND (COALESCE(mr.backfill_completed, 0) & {guard_mask} = 0) "
        )
        params = []
        if xuid:
            query += "AND mp.xuid = ? "
            params.append(xuid)
        query += "ORDER BY mr.start_time DESC"

    if max_matches:
        query += f" LIMIT {int(max_matches)}"

    match_ids = [r[0] for r in shared_conn.execute(query, params).fetchall()]

    if not match_ids:
        logger.info("Aucun match à enrichir (participants-enrich)")
        return 0

    logger.info(f"Participants-enrich : {len(match_ids)} match(s) à traiter")

    tokens = await get_tokens_from_env()
    if not tokens:
        logger.error("Tokens SPNKr non disponibles")
        return 0

    count = 0
    async with SPNKrAPIClient(
        tokens=tokens,
        requests_per_second=requests_per_second,
    ) as client:
        for i, match_id in enumerate(match_ids, 1):
            try:
                logger.info(f"  [{i}/{len(match_ids)}] Enrichissement {match_id[:20]}...")

                # 1. Récupérer les stats du match (pour extract_participants)
                stats_json = await client.get_match_stats(match_id)
                if not stats_json:
                    logger.warning(f"  Impossible de récupérer {match_id}")
                    continue

                # 2. Extraire les participants avec toutes les colonnes étendues
                participant_rows = extract_participants(stats_json)
                if not participant_rows:
                    continue

                # 3. UPDATE match_participants pour chaque participant
                for row in participant_rows:
                    shared_conn.execute(
                        "UPDATE match_participants SET "
                        "kills = COALESCE(?, kills), "
                        "deaths = COALESCE(?, deaths), "
                        "assists = COALESCE(?, assists), "
                        "shots_fired = COALESCE(?, shots_fired), "
                        "shots_hit = COALESCE(?, shots_hit), "
                        "damage_dealt = COALESCE(?, damage_dealt), "
                        "damage_taken = COALESCE(?, damage_taken), "
                        "avg_life_seconds = COALESCE(?, avg_life_seconds), "
                        "headshot_kills = COALESCE(?, headshot_kills), "
                        "max_killing_spree = COALESCE(?, max_killing_spree), "
                        "kda = COALESCE(?, kda), "
                        "accuracy = COALESCE(?, accuracy), "
                        "time_played_seconds = COALESCE(?, time_played_seconds), "
                        "grenade_kills = COALESCE(?, grenade_kills), "
                        "melee_kills = COALESCE(?, melee_kills), "
                        "power_weapon_kills = COALESCE(?, power_weapon_kills), "
                        "personal_score = COALESCE(?, personal_score) "
                        "WHERE match_id = ? AND xuid = ?",
                        (
                            row.kills,
                            row.deaths,
                            row.assists,
                            row.shots_fired,
                            row.shots_hit,
                            row.damage_dealt,
                            row.damage_taken,
                            row.avg_life_seconds,
                            row.headshot_kills,
                            row.max_killing_spree,
                            row.kda,
                            row.accuracy,
                            row.time_played_seconds,
                            row.grenade_kills,
                            row.melee_kills,
                            row.power_weapon_kills,
                            row.personal_score,
                            row.match_id,
                            row.xuid,
                        ),
                    )

                # 4. Récupérer et appliquer les données skill/MMR
                xuids = extract_xuids_from_match(stats_json)
                if xuids:
                    skill_json = await client.get_skill_stats(match_id, xuids)
                    if skill_json:
                        value = skill_json.get("Value")
                        if isinstance(value, list):
                            for player in value:
                                if not isinstance(player, dict):
                                    continue
                                player_id = player.get("Id", "")
                                if not isinstance(player_id, str):
                                    continue
                                # Extraire le xuid du player_id
                                import re

                                xuid_match = re.search(r"xuid\((\d+)\)", player_id)
                                if not xuid_match:
                                    continue
                                p_xuid = xuid_match.group(1)
                                p_skill = transform_skill_stats(skill_json, match_id, p_xuid)
                                if p_skill and p_skill.team_mmr is not None:
                                    shared_conn.execute(
                                        "UPDATE match_participants SET "
                                        "team_mmr = COALESCE(?, team_mmr), "
                                        "kills_expected = COALESCE(?, kills_expected), "
                                        "kills_stddev = COALESCE(?, kills_stddev), "
                                        "deaths_expected = COALESCE(?, deaths_expected), "
                                        "deaths_stddev = COALESCE(?, deaths_stddev), "
                                        "assists_expected = COALESCE(?, assists_expected), "
                                        "assists_stddev = COALESCE(?, assists_stddev) "
                                        "WHERE match_id = ? AND xuid = ?",
                                        (
                                            p_skill.team_mmr,
                                            p_skill.kills_expected,
                                            p_skill.kills_stddev,
                                            p_skill.deaths_expected,
                                            p_skill.deaths_stddev,
                                            p_skill.assists_expected,
                                            p_skill.assists_stddev,
                                            match_id,
                                            p_xuid,
                                        ),
                                    )

                shared_conn.commit()
                count += 1

                # Marquer le match comme traité dans backfill_completed
                # Bits: participants (512), participants_avg_life (32768)
                try:
                    from src.data.sync.migrations import BACKFILL_FLAGS

                    flags_to_mark = (
                        BACKFILL_FLAGS.get("participants", 0)
                        | BACKFILL_FLAGS.get("participants_scores", 0)
                        | BACKFILL_FLAGS.get("participants_kda", 0)
                        | BACKFILL_FLAGS.get("participants_shots", 0)
                        | BACKFILL_FLAGS.get("participants_damage", 0)
                        | BACKFILL_FLAGS.get("participants_avg_life", 0)
                    )

                    shared_conn.execute(
                        "UPDATE match_registry SET "
                        "backfill_completed = COALESCE(backfill_completed, 0) | ? "
                        "WHERE match_id = ?",
                        (flags_to_mark, match_id),
                    )
                    shared_conn.commit()
                except Exception as e:
                    logger.warning(f"  Impossible de marquer backfill_completed: {e}")

                logger.info(f"  ✅ Match {match_id[:20]}... enrichi")

            except Exception as e:
                logger.error(f"  Erreur enrichissement {match_id}: {e}")
                import traceback

                traceback.print_exc()
                continue

    logger.info(f"✅ Participants-enrich terminé : {count}/{len(match_ids)} matchs enrichis")
    return count


# ─────────────────────────────────────────────────────────────────────────────
# Team scores (match_registry)
# ─────────────────────────────────────────────────────────────────────────────


async def backfill_team_scores(
    shared_conn: Any,
    *,
    max_matches: int | None = None,
    force: bool = False,
    requests_per_second: int = 5,
) -> int:
    """Peuple team_0_score / team_1_score dans shared.match_registry via l'API.

    Ces colonnes sont NULL pour les matchs insérés avant le correctif
    d'extraction (TotalPoints / Stats.CoreStats.Score).

    Args:
        shared_conn: Connexion en écriture vers shared_matches.duckdb.
        max_matches: Nombre max de matchs à traiter.
        force: Si True, recalcule même si les scores sont déjà présents.
        requests_per_second: Rate limiting API.

    Returns:
        Nombre de matchs mis à jour.
    """
    from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env
    from src.data.sync.transformers import _extract_team_scores_by_id

    where_clause = "" if force else "WHERE team_0_score IS NULL OR team_1_score IS NULL"
    query = f"SELECT match_id FROM match_registry {where_clause} ORDER BY start_time DESC"
    if max_matches:
        query += f" LIMIT {int(max_matches)}"

    match_ids = [r[0] for r in shared_conn.execute(query).fetchall()]

    if not match_ids:
        logger.info("Aucun match avec team scores manquants")
        return 0

    logger.info(f"Team scores : {len(match_ids)} match(s) à traiter")

    tokens = await get_tokens_from_env()
    if not tokens:
        logger.error("Tokens SPNKr non disponibles")
        return 0

    count = 0
    async with SPNKrAPIClient(tokens=tokens, requests_per_second=requests_per_second) as client:
        for i, match_id in enumerate(match_ids, 1):
            try:
                logger.info(f"  [{i}/{len(match_ids)}] {match_id[:20]}...")
                stats_json = await client.get_match_stats(match_id)
                if not stats_json:
                    logger.warning(f"  Impossible de récupérer {match_id}")
                    continue

                t0, t1 = _extract_team_scores_by_id(stats_json)
                if t0 is None and t1 is None:
                    logger.debug(f"  Scores toujours NULL pour {match_id} (mode sans équipes ?)")
                    continue

                shared_conn.execute(
                    "UPDATE match_registry SET team_0_score = ?, team_1_score = ? WHERE match_id = ?",
                    (t0, t1, match_id),
                )
                shared_conn.commit()
                count += 1
                logger.info(f"  ✅ team_0={t0}, team_1={t1}")

            except Exception as e:
                logger.error(f"  Erreur {match_id}: {e}")
                continue

    logger.info(f"✅ Team scores terminé : {count}/{len(match_ids)} matchs mis à jour")
    return count


# ─────────────────────────────────────────────────────────────────────────────
# Citations
# ─────────────────────────────────────────────────────────────────────────────


def backfill_citations(
    conn: Any,
    db_path: str | Any,
    xuid: str,
    *,
    force: bool = False,
) -> int:
    """Délègue à la source canonique ``src.data.citations_backfill``.

    Args:
        conn: Connexion DuckDB (réutilisée si fournie).
        db_path: Chemin vers la DB joueur.
        xuid: XUID du joueur.
        force: Si True, recalcule pour tous les matchs.

    Returns:
        Nombre de matchs traités avec citations.
    """
    from src.data.citations_backfill import backfill_citations_for_player

    if force:
        # Supprimer les citations existantes pour forcer le recalcul complet
        conn.execute("DELETE FROM match_citations")

    result = backfill_citations_for_player(db_path, xuid, conn=conn)
    return result.get("citations_computed", 0)


# ─────────────────────────────────────────────────────────────────────────────
# Mode category (match_registry — local, sans API)
# ─────────────────────────────────────────────────────────────────────────────


def backfill_mode_category(shared_conn: Any, *, force: bool = False) -> int:
    """Recalcule ``mode_category`` dans ``match_registry`` depuis ``pair_name``.

    Opération purement locale : aucun appel API. Utilise
    ``infer_custom_category_from_pair_name`` sur les ``pair_name`` déjà
    stockés en base.

    Args:
        shared_conn: Connexion en écriture vers shared_matches.duckdb.
        force: Si True, recalcule pour TOUS les matchs (même ceux déjà renseignés).

    Returns:
        Nombre de matchs mis à jour.
    """
    from src.analysis.mode_categories import infer_custom_category_from_pair_name

    where = "" if force else "WHERE mode_category IS NULL"
    rows = shared_conn.execute(
        f"SELECT match_id, pair_name FROM match_registry {where} ORDER BY start_time DESC"
    ).fetchall()

    if not rows:
        logger.info("Aucun match avec mode_category manquant")
        return 0

    logger.info(f"mode_category : {len(rows)} match(s) à recalculer (force={force})")
    count = 0
    for match_id, pair_name in rows:
        category = infer_custom_category_from_pair_name(pair_name)
        if not category:
            continue
        shared_conn.execute(
            "UPDATE match_registry SET mode_category = ?, updated_at = CURRENT_TIMESTAMP WHERE match_id = ?",
            (category, match_id),
        )
        count += 1

    shared_conn.commit()
    logger.info(f"✅ mode_category : {count}/{len(rows)} matchs mis à jour")
    return count


# ─────────────────────────────────────────────────────────────────────────────
# Nettoyage structurel des DBs joueurs (vues cassées + tables legacy)
# ─────────────────────────────────────────────────────────────────────────────


def cleanup_player_dbs_legacy(players_dir: str | Any = "data/players") -> dict[str, int]:
    """Supprime les vues cassées et tables legacy dans chaque stats.duckdb joueur.

    Problèmes corrigés :
    - 4 vues (``v_highlight_events``, ``v_match_participants``, ``v_match_stats``,
      ``v_medals_earned``) référencent des tables supprimées lors de la migration v5.1.
    - Table ``match_participants`` legacy encore présente dans certains joueurs
      (données centralisées dans shared_matches.duckdb depuis v5.1).

    Args:
        players_dir: Dossier racine des joueurs (``data/players/``).

    Returns:
        Dict ``{gamertag: nb_ops}`` avec le nombre d'objets supprimés par joueur.
    """
    import os
    from pathlib import Path

    import duckdb

    players_dir = Path(players_dir)
    broken_views = [
        "v_highlight_events",
        "v_match_participants",
        "v_match_stats",
        "v_medals_earned",
    ]
    # Tables legacy présentes dans les stats.duckdb mais centralisées dans shared (v5.1)
    # Source : CLAUDE.md § "8 tables supprimées lors du cleanup v5.1"
    legacy_tables = [
        "match_participants",
        "match_stats",
        "highlight_events",
        "medals_earned",
        "killer_victim_pairs",
        "player_match_stats",
        "xuid_aliases",
        "teammates_aggregate",
    ]

    results: dict[str, int] = {}

    for gt in sorted(os.listdir(players_dir)):
        db_path = players_dir / gt / "stats.duckdb"
        if not db_path.exists():
            continue
        ops = 0
        try:
            con = duckdb.connect(str(db_path), read_only=False)
            existing_tables = {
                r[0]
                for r in con.execute(
                    "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
                ).fetchall()
            }
            existing_views = {
                r[0]
                for r in con.execute(
                    "SELECT table_name FROM information_schema.views WHERE table_schema='main'"
                ).fetchall()
            }

            for view in broken_views:
                if view in existing_views:
                    con.execute(f"DROP VIEW IF EXISTS {view}")
                    logger.info(f"  [{gt}] DROP VIEW {view}")
                    ops += 1

            for table in legacy_tables:
                if table in existing_tables:
                    row_count = con.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                    con.execute(f"DROP TABLE IF EXISTS {table}")
                    logger.info(
                        f"  [{gt}] DROP TABLE {table} ({row_count} lignes legacy supprimées)"
                    )
                    ops += 1

            con.commit()
            con.close()
            results[gt] = ops
            if ops:
                logger.info(f"  ✅ {gt}: {ops} objet(s) nettoyé(s)")
            else:
                logger.debug(f"  {gt}: rien à nettoyer")
        except Exception as e:
            logger.error(f"  Erreur cleanup {gt}: {e}")
            results[gt] = -1

    return results
