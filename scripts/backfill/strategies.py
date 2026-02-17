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
    conn: Any, match_id: str, *, shared_conn: Any, xuid: str
) -> bool:
    """Calcule et met à jour le score de performance pour un match.

    Lit depuis shared.match_participants + match_registry,
    écrit dans player_match_enrichment (player DB).

    Args:
        conn: Connexion DuckDB (player DB pour player_match_enrichment).
        match_id: ID du match.
        shared_conn: Connexion vers shared_matches.duckdb (obligatoire).
        xuid: XUID du joueur (obligatoire).

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

        # Vérifier si le score existe déjà dans player_match_enrichment
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
                   mp.personal_score, mp.damage_dealt, mp.rank, mp.team_mmr
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
                    mp.personal_score, mp.damage_dealt, mp.rank, mp.team_mmr
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
            "enemy_mmr": match_data[13] if len(match_data) > 13 else None,
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
        # On ne considère que headshot_kills et kda car team_mmr peut être NULL
        # légitimement (API skill ne retourne pas toujours de données)
        query = (
            "SELECT DISTINCT mp.match_id "
            "FROM match_participants mp "
            "JOIN match_registry mr ON mp.match_id = mr.match_id "
            "WHERE (mp.headshot_kills IS NULL OR mp.kda IS NULL) "
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
                logger.info(f"  ✅ Match {match_id[:20]}... enrichi")

            except Exception as e:
                logger.error(f"  Erreur enrichissement {match_id}: {e}")
                import traceback

                traceback.print_exc()
                continue

    logger.info(f"✅ Participants-enrich terminé : {count}/{len(match_ids)} matchs enrichis")
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
    """Calcule et insère les citations pour les matchs existants.

    Mode incrémental par défaut : ne traite que les matchs sans citations.
    Mode force : recalcule pour tous les matchs.

    Args:
        conn: Connexion DuckDB (non utilisée directement, on ouvre via engine).
        db_path: Chemin vers la DB joueur.
        xuid: XUID du joueur.
        force: Si True, recalcule pour tous les matchs.

    Returns:
        Nombre de matchs traités.
    """
    from pathlib import Path

    from src.analysis.citations.engine import CitationEngine

    db_path = Path(db_path) if not isinstance(db_path, Path) else db_path
    engine = CitationEngine(str(db_path), xuid, conn=conn)

    # Vérifier que les mappings existent
    mappings = engine.load_mappings()
    if not mappings:
        logger.warning("Aucun mapping de citations trouvé")
        return 0

    # Récupérer les match_ids à traiter via shared.match_participants
    shared_path = db_path.parent.parent.parent / "warehouse" / "shared_matches.duckdb"
    if not shared_path.exists():
        logger.warning("shared_matches.duckdb introuvable pour les citations")
        return 0

    # Vérifier si shared est déjà attaché (sous n'importe quel alias)
    # D'abord chercher via le nom de fichier ou via match_participants
    shared_alias = None
    try:
        dbs = conn.execute("SELECT database_name, path FROM duckdb_databases()").fetchall()
        for db_name, db_path_val in dbs:
            # Chercher par nom de fichier dans le path
            if db_path_val and "shared_matches.duckdb" in str(db_path_val).lower():
                shared_alias = db_name
                break
            # Ou par nom de base contenant "shared"
            if db_name and "shared" in db_name.lower():
                # Vérifier que cette DB a bien match_participants
                try:
                    conn.execute(f"SELECT 1 FROM {db_name}.match_participants LIMIT 1")
                    shared_alias = db_name
                    break
                except Exception:
                    continue
    except Exception:
        pass

    # Si pas trouvé, chercher n'importe quelle DB avec match_participants
    if not shared_alias:
        try:
            dbs = conn.execute("SELECT database_name FROM duckdb_databases()").fetchall()
            for (db_name,) in dbs:
                if db_name not in ("memory", "temp", "main", "stats", "system"):
                    try:
                        conn.execute(f"SELECT 1 FROM {db_name}.match_participants LIMIT 1")
                        shared_alias = db_name
                        break
                    except Exception:
                        continue
        except Exception:
            pass

    # En dernier recours, essayer d'attacher
    if not shared_alias:
        try:
            conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
            shared_alias = "shared"
        except Exception as e:
            logger.debug(f"Attach shared pour citations: {e}")

    if not shared_alias:
        logger.warning("Impossible de trouver la base shared pour les citations")
        return 0

    # S'assurer que match_citations existe (table locale dans player DB)
    conn.execute("""
        CREATE TABLE IF NOT EXISTS match_citations (
            match_id VARCHAR NOT NULL,
            citation_name_norm VARCHAR NOT NULL,
            value INTEGER DEFAULT 0,
            PRIMARY KEY (match_id, citation_name_norm)
        )
    """)

    if force:
        match_ids = [
            r[0]
            for r in conn.execute(
                f"SELECT DISTINCT mp.match_id "
                f"FROM {shared_alias}.match_participants mp "
                f"JOIN {shared_alias}.match_registry mr ON mp.match_id = mr.match_id "
                f"WHERE mp.xuid = ? "
                f"ORDER BY mr.start_time",
                [xuid],
            ).fetchall()
        ]
    else:
        match_ids = [
            r[0]
            for r in conn.execute(
                f"SELECT DISTINCT mp.match_id "
                f"FROM {shared_alias}.match_participants mp "
                f"JOIN {shared_alias}.match_registry mr ON mp.match_id = mr.match_id "
                f"WHERE mp.xuid = ? "
                f"  AND NOT EXISTS ("
                f"    SELECT 1 FROM match_citations mc "
                f"    WHERE mc.match_id = mp.match_id"
                f"  ) "
                f"ORDER BY mr.start_time",
                [xuid],
            ).fetchall()
        ]

    if not match_ids:
        logger.info("Aucun match à traiter pour les citations")
        return 0

    logger.info(f"Traitement de {len(match_ids)} match(s) pour les citations...")

    count = 0
    for i, match_id in enumerate(match_ids, 1):
        try:
            n = engine.compute_and_store_for_match(match_id, conn=conn)
            if n > 0:
                count += 1
            if i % 100 == 0:
                logger.info(f"  [{i}/{len(match_ids)}] {count} matchs avec citations")
        except Exception as e:
            logger.warning(f"Erreur citation pour {match_id}: {e}")
            continue

    logger.info(f"✅ {count} match(s) traités pour les citations")
    return count
