"""Orchestration du backfill pour un ou plusieurs joueurs.

Ce module contient la logique principale de backfill :
- backfill_player_data  : traitement d'un joueur
- backfill_all_players  : itération sur tous les joueurs DuckDB

Note architecture (v5.2+) :
    Les fonctions publiques acceptent un paramètre ``scope: SyncScope``
    qui remplace les 30+ kwargs individuels.  Les kwargs sont conservés
    temporairement pour rétro-compatibilité (marqués ``LEGACY``).
    Nouveau code : toujours passer ``scope=SyncScope(...)``.
"""

from __future__ import annotations

import contextlib
import logging
from pathlib import Path
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync.scope import SyncScope

from scripts.backfill.core import (
    insert_alias_rows,
    insert_event_rows,
    insert_medal_rows,
    insert_participant_rows,
    insert_personal_score_rows,
)
from scripts.backfill.detection import compute_backfill_mask, find_matches_missing_data
from scripts.backfill.strategies import (
    backfill_end_time,
    backfill_killer_victim_pairs,
    compute_performance_score_for_match,
)
from src.data.sync.scope import SyncScope

logger = logging.getLogger(__name__)

# Répertoire racine du projet
_PROJECT_ROOT = Path(__file__).resolve().parents[2]


def _get_shared_connection(db_path: Path) -> Any | None:
    """Ouvre une connexion vers shared_matches.duckdb (v5).

    Le chemin est dérivé depuis la racine du projet.
    Retourne None si la base n'existe pas.
    """
    import duckdb

    shared_path = _PROJECT_ROOT / "data" / "warehouse" / "shared_matches.duckdb"
    if not shared_path.exists():
        logger.warning(f"shared_matches.duckdb introuvable: {shared_path}")
        return None
    try:
        return duckdb.connect(str(shared_path), read_only=False)
    except Exception as e:
        logger.warning(f"Impossible d'ouvrir shared_matches.duckdb: {e}")
        return None


def _empty_result() -> dict[str, int]:
    """Retourne un dict de résultat vide."""
    return {
        "matches_checked": 0,
        "matches_missing_data": 0,
        "medals_inserted": 0,
        "events_inserted": 0,
        "skill_inserted": 0,
        "personal_scores_inserted": 0,
        "performance_scores_inserted": 0,
        "aliases_inserted": 0,
        "accuracy_updated": 0,
        "shots_updated": 0,
        "enemy_mmr_updated": 0,
        "assets_updated": 0,
        "participants_inserted": 0,
        "participants_scores_updated": 0,
        "participants_kda_updated": 0,
        "participants_shots_updated": 0,
        "participants_damage_updated": 0,
        "participants_avg_life_updated": 0,
        "killer_victim_pairs_inserted": 0,
        "end_time_updated": 0,
        "sessions_updated": 0,
        "citations_computed": 0,
        "participants_enriched": 0,
        "pve_stats_inserted": 0,
        "lusr_computed": 0,
        "csr_fetched": 0,
        "csr_backfilled": 0,
    }


async def backfill_player_data(
    gamertag: str,
    *,
    scope: SyncScope | None = None,
) -> dict[str, int]:
    """Remplit les données manquantes pour un joueur.

    Args:
        gamertag: Gamertag du joueur.
        scope: Périmètre de données (SyncScope). Obligatoire en pratique ;
               si ``None``, un scope vide est créé (aucune action).

    Returns:
        Dict avec les statistiques.
    """
    # ── Construire / résoudre le scope ──
    if scope is None:
        scope = SyncScope()
    scope.resolve()

    # Extraire les valeurs résolues en variables locales (used below)
    dry_run = scope.dry_run
    max_matches = scope.max_matches
    requests_per_second = scope.requests_per_second
    killer_victim = scope.killer_victim
    end_time = scope.end_time
    force_end_time = scope.force_end_time
    sessions = scope.sessions
    citations = scope.citations
    participants_enrich = scope.participants_enrich
    force_sessions = scope.force_sessions
    force_citations = scope.force_citations
    force_participants_enrich = scope.force_participants_enrich
    teammates_sig = getattr(scope, "teammates_sig", False)
    force_teammates_sig = getattr(scope, "force_teammates_sig", False)
    lusr = scope.lusr
    force_lusr = scope.force_lusr
    csr = scope.csr
    fetch_csr = scope.fetch_csr
    skill_rank = scope.skill_rank
    force_skill_rank = scope.force_skill_rank

    # Vérifier qu'au moins une option est activée
    if not scope.has_any_option():
        logger.warning(
            "Aucune option de backfill activée. Utilisez --all ou spécifiez des options."
        )
        return _empty_result()

    # Imports lourds
    from src.ui.sync import get_player_duckdb_path, is_duckdb_player
    from src.utils import resolve_xuid_from_db

    # Vérifier que c'est un joueur DuckDB
    if not is_duckdb_player(gamertag):
        logger.error(f"{gamertag} n'a pas de DB DuckDB. Ce script ne fonctionne que pour DuckDB.")
        return _empty_result()

    db_path = get_player_duckdb_path(gamertag)
    if not db_path or not db_path.exists():
        logger.error(f"DB DuckDB introuvable pour {gamertag}")
        return _empty_result()

    # Résoudre le XUID
    xuid = resolve_xuid_from_db(str(db_path), gamertag)
    if not xuid:
        xuid = _resolve_xuid_fallback(db_path, gamertag)
        if not xuid:
            return _empty_result()
    else:
        logger.info(f"✅ XUID résolu depuis xuid_aliases: {xuid}")

    import duckdb

    conn = duckdb.connect(str(db_path), read_only=False)

    # V5 FINAL : Toujours ouvrir shared_conn (source principale depuis migration V5)
    shared_conn_for_detection = _get_shared_connection(db_path)
    if shared_conn_for_detection is None:
        logger.error(
            f"❌ {gamertag}: shared_matches.duckdb introuvable — "
            "impossible de backfill en V5. Vérifiez l'installation."
        )
        conn.close()
        return _empty_result()

    try:
        # V5 : Vérifier que le joueur a des données dans shared DB
        mp_count = shared_conn_for_detection.execute(
            "SELECT COUNT(*) FROM match_participants WHERE xuid = ("
            "  SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?) LIMIT 1"
            ")",
            [gamertag],
        ).fetchone()
        # Fallback: aussi vérifier par xuid direct
        has_data_in_shared = bool(mp_count and mp_count[0] > 0)
        if not has_data_in_shared:
            mp_count2 = shared_conn_for_detection.execute(
                "SELECT COUNT(*) FROM match_participants WHERE xuid = ?",
                [xuid],
            ).fetchone()
            has_data_in_shared = bool(mp_count2 and mp_count2[0] > 0)

        if not has_data_in_shared:
            logger.warning(
                f"⏭️ {gamertag}: Aucune donnée dans shared DB. "
                f"Lancer d'abord: python scripts/sync.py --player {gamertag} --delta"
            )
            conn.close()
            shared_conn_for_detection.close()
            return _empty_result()

        # Migrations de schéma (shared DB)
        _apply_schema_migrations(
            conn,
            shared_conn_for_detection,
        )

        # Détection des matchs manquants
        match_ids = find_matches_missing_data(
            conn,
            xuid,
            shared_conn=shared_conn_for_detection,
            scope=scope,
        )

        logger.info(f"Matchs trouvés avec données manquantes: {len(match_ids)}")

        if dry_run:
            logger.info("Mode dry-run: aucun traitement effectué")
            result = _empty_result()
            result["matches_checked"] = len(match_ids)
            result["matches_missing_data"] = len(match_ids)
            return result

        # Cas : pas de matchs API mais backfill local à faire
        needs_local_only = (
            killer_victim
            or end_time
            or sessions
            or citations
            or lusr
            or skill_rank
            or teammates_sig
        )
        needs_api = bool(match_ids) or fetch_csr or csr or skill_rank

        # Participants-enrich : mode autonome sur shared (pas besoin de match_ids détectés)
        if participants_enrich:
            from scripts.backfill.strategies import backfill_participants_enrich

            logger.info("Backfill colonnes étendues + MMR (participants-enrich)...")
            n = await backfill_participants_enrich(
                shared_conn_for_detection,
                xuid=xuid,
                max_matches=max_matches,
                force=force_participants_enrich,
                requests_per_second=requests_per_second,
            )
            result_pe = _empty_result()
            result_pe["participants_enriched"] = n

            # Si pas d'autres flags, retourner directement
            if not needs_api and not needs_local_only:
                return result_pe

        if not needs_api and not needs_local_only and not participants_enrich:
            logger.info("Tous les matchs ont déjà toutes les données demandées")
            return _empty_result()

        if not needs_api and needs_local_only:
            # Fermer shared_conn_for_detection avant _backfill_local_only
            # car sessions/citations ont besoin d'attacher le fichier à conn
            if shared_conn_for_detection is not None:
                try:
                    shared_conn_for_detection.commit()
                    shared_conn_for_detection.close()
                    shared_conn_for_detection = None
                except Exception:
                    pass

            result = _backfill_local_only(
                conn,
                db_path,
                xuid,
                killer_victim=killer_victim,
                end_time=end_time,
                sessions=sessions,
                citations=citations,
                lusr=lusr or skill_rank,
                force_end_time=force_end_time,
                force_sessions=force_sessions,
                force_citations=force_citations,
                force_lusr=force_lusr or force_skill_rank,
                teammates_sig=teammates_sig,
                force_teammates_sig=force_teammates_sig,
                dry_run=dry_run,
            )
            if participants_enrich:
                result["participants_enriched"] = result_pe.get("participants_enriched", 0)
            return result

        # Traitement API
        return await _backfill_with_api(
            conn,
            db_path,
            xuid,
            match_ids,
            scope=scope,
            gamertag=gamertag,
            existing_shared_conn=shared_conn_for_detection,
        )

    finally:
        with contextlib.suppress(Exception):
            conn.commit()
        conn.close()
        if shared_conn_for_detection is not None:
            with contextlib.suppress(Exception):
                shared_conn_for_detection.close()


async def backfill_all_players(
    *,
    scope: SyncScope | None = None,
) -> dict[str, Any]:
    """Backfill pour tous les joueurs DuckDB.

    Args:
        scope: Périmètre de données (SyncScope). Obligatoire en pratique ;
               si ``None``, un scope vide est créé (aucune action).
    """
    # ── Construire le scope si non fourni ──
    if scope is None:
        scope = SyncScope()

    from src.ui.multiplayer import list_duckdb_v4_players

    players = list_duckdb_v4_players()

    if not players:
        logger.warning("Aucun joueur DuckDB trouvé")
        return {"players_processed": 0, "total_results": {}}

    logger.info(f"Trouvé {len(players)} joueur(s) DuckDB")

    total_results = _empty_result()
    per_player_results: dict[str, Any] = {}

    for i, player_info in enumerate(players, 1):
        logger.info(f"\n{'=' * 60}")
        logger.info(f"[{i}/{len(players)}] Traitement de {player_info.gamertag}")
        logger.info(f"{'=' * 60}")

        result = await backfill_player_data(
            player_info.gamertag,
            scope=scope,
        )

        for key in total_results:
            total_results[key] += result.get(key, 0)
        per_player_results[player_info.gamertag] = result

    return {
        "players_processed": len(players),
        "total_results": total_results,
        "per_player": per_player_results,
    }


# ─────────────────────────────────────────────────────────────────────────────
# Helpers internes
# ─────────────────────────────────────────────────────────────────────────────


def _resolve_xuid_fallback(db_path: Path, gamertag: str) -> str | None:
    """Tente de résoudre le XUID via db_profiles.json puis highlight_events."""
    import duckdb

    logger.warning(f"XUID introuvable dans xuid_aliases pour {gamertag}")

    # 1. Chercher dans db_profiles.json (source la plus fiable)
    try:
        from src.data.repositories.factory import load_db_profiles

        profiles_data = load_db_profiles()
        profiles = profiles_data.get("profiles", {})
        for key, info in profiles.items():
            if (
                key.lower() == gamertag.lower()
                or str(info.get("waypoint_player", "")).lower() == gamertag.lower()
            ):
                xuid = str(info.get("xuid", "")).strip()
                if xuid and xuid.isdigit():
                    logger.info(f"✅ XUID trouvé depuis db_profiles.json: {xuid}")
                    return xuid
    except Exception as e:
        logger.debug(f"Lecture db_profiles.json échouée: {e}")

    # 2. Fallback : chercher dans highlight_events (DB existantes)
    logger.info("Tentative d'extraction depuis les matchs existants...")
    conn = duckdb.connect(str(db_path), read_only=True)
    try:
        # Vérifier que la table existe avant de la requêter
        table_check = conn.execute(
            "SELECT COUNT(*) FROM information_schema.tables "
            "WHERE table_schema = 'main' AND table_name = 'highlight_events'"
        ).fetchone()
        if not table_check or table_check[0] == 0:
            logger.error(
                f"❌ Impossible de résoudre le XUID pour {gamertag} "
                "(DB vide, pas encore synchronisée)"
            )
            logger.error(f"💡 Solution: python scripts/sync.py --player {gamertag} --delta")
            return None

        result = conn.execute(
            """
            SELECT DISTINCT xuid
            FROM highlight_events
            WHERE LOWER(gamertag) = LOWER(?)
              AND xuid IS NOT NULL AND xuid != ''
            LIMIT 1
            """,
            [gamertag],
        ).fetchone()

        if result and result[0]:
            xuid = str(result[0])
            logger.info(f"✅ XUID trouvé depuis highlight_events: {xuid}")
            return xuid

        logger.error(f"❌ Impossible de résoudre le XUID pour {gamertag}")
        logger.error(f"💡 Solution: python scripts/sync.py --player {gamertag} --delta")
        return None
    finally:
        conn.close()


def _apply_schema_migrations(
    conn: Any,
    shared_conn: Any,
) -> None:
    """Applique les migrations de schéma nécessaires avant le backfill."""
    from src.data.sync.migrations import (
        ensure_backfill_completed_column,
        ensure_highlight_events_autoincrement,
        ensure_match_participants_backfill_bits,
        ensure_match_participants_columns,
        ensure_medals_earned_bigint,
    )

    # Migration medals_earned INT32 → BIGINT (shared)
    with contextlib.suppress(Exception):
        ensure_medals_earned_bigint(shared_conn)

    # Migration highlight_events : séquence auto-increment pour id (shared)
    with contextlib.suppress(Exception):
        ensure_highlight_events_autoincrement(shared_conn)

    # Colonnes match_participants (shared)
    with contextlib.suppress(Exception):
        ensure_match_participants_columns(shared_conn)

    # Colonne backfill_bits (v5.2 — bitmask granulaire par joueur)
    with contextlib.suppress(Exception):
        ensure_match_participants_backfill_bits(shared_conn)

    # Colonne backfill_completed dans match_registry (shared)
    with contextlib.suppress(Exception):
        ensure_backfill_completed_column(shared_conn)


def _backfill_local_only(
    conn: Any,
    db_path: Path,
    xuid: str,
    *,
    killer_victim: bool,
    end_time: bool,
    sessions: bool,
    citations: bool = False,
    lusr: bool = False,
    force_end_time: bool = False,
    force_sessions: bool = False,
    force_citations: bool = False,
    force_lusr: bool = False,
    teammates_sig: bool = False,
    force_teammates_sig: bool = False,
    dry_run: bool = False,
) -> dict[str, int]:
    """Backfill local uniquement (pas d'API nécessaire)."""
    logger.info("Pas de données de match à backfill via API...")
    result = _empty_result()

    if killer_victim:
        logger.info("Backfill des paires killer/victim depuis highlight_events...")
        # Ouvrir une connexion vers shared_matches.duckdb (v5)
        shared_conn = _get_shared_connection(db_path)
        n = backfill_killer_victim_pairs(conn, xuid, shared_conn=shared_conn)
        if shared_conn is not None:
            shared_conn.close()
        result["killer_victim_pairs_inserted"] = n
        if n > 0:
            logger.info(f"✅ {n} paires killer/victim insérées")
        else:
            logger.info("Aucune nouvelle paire killer/victim à insérer")

    if end_time:
        logger.info("Backfill de l'heure de fin des matchs (end_time)...")
        shared_conn = _get_shared_connection(db_path)
        n = backfill_end_time(conn, force=force_end_time, shared_conn=shared_conn)
        if shared_conn is not None:
            shared_conn.close()
        result["end_time_updated"] = n
        if n > 0:
            logger.info(f"✅ {n} match(s) avec end_time mis à jour")
        else:
            logger.info("Aucun match à mettre à jour pour end_time")

    if sessions:
        n = _backfill_sessions(conn, db_path, xuid, force=force_sessions, dry_run=dry_run)
        result["sessions_updated"] = n

    if citations:
        from scripts.backfill.strategies import backfill_citations

        logger.info("Backfill des citations...")
        n = backfill_citations(conn, db_path, xuid, force=force_citations)
        result["citations_computed"] = n

    if lusr:
        from scripts.backfill.strategies import compute_lusr_for_player

        logger.info("Calcul du LUSR (LevelUp Skill Rank) pour les matchs non classés...")
        n = compute_lusr_for_player(conn, db_path, xuid, force=force_lusr)
        result["lusr_computed"] = n
        if n > 0:
            logger.info(f"✅ LUSR calculé pour {n} match(s)")
        else:
            logger.info("Aucun nouveau match à calculer pour le LUSR")

    if teammates_sig:
        from src.data.sessions_backfill import backfill_teammates_signatures

        logger.info("Backfill teammates_signature + is_with_friends depuis shared...")
        r = backfill_teammates_signatures(db_path, xuid, force=force_teammates_sig)
        n = r.get("updated", 0)
        result["teammates_sig_updated"] = n
        for e in r.get("errors", []):
            logger.warning(f"  Erreur teammates_sig: {e}")
        logger.info(
            f"✅ {n} signature(s) de coéquipiers mises à jour"
            if n
            else "Aucune signature à mettre à jour"
        )

    return result


def _backfill_sessions(
    conn: Any,
    db_path: Path,
    xuid: str,
    *,
    force: bool,
    dry_run: bool,
) -> int:
    """Backfill des sessions (session_id, session_label)."""
    from src.config import SESSION_CONFIG
    from src.data.sessions_backfill import backfill_sessions_for_player

    logger.info("Backfill des sessions (session_id, session_label)...")
    r = backfill_sessions_for_player(
        db_path,
        xuid,
        conn=conn,
        gap_minutes=SESSION_CONFIG.advanced_gap_minutes,
        include_recent=True,
        force=force,
        dry_run=dry_run,
    )
    n = r.get("updated", 0)
    if r.get("errors"):
        for e in r["errors"]:
            logger.warning(f"  Erreur session: {e}")
    if n > 0:
        logger.info(f"✅ {n} match(s) avec sessions mis à jour")
    else:
        logger.info("Aucun match à mettre à jour pour sessions")
    return n


async def _backfill_with_api(
    conn: Any,
    db_path: Path,
    xuid: str,
    match_ids: list[str],
    *,
    scope: SyncScope | None = None,
    # ── LEGACY kwargs (v5.1) ──────────────────────────────────────────────
    # TODO(cleanup): supprimer ces kwargs quand tous les appelants
    #   utilisent SyncScope.
    # ─────────────────────────────────────────────────────────────────────
    requests_per_second: int = 5,
    medals: bool = False,
    events: bool = False,
    skill: bool = False,
    personal_scores: bool = False,
    performance_scores: bool = False,
    aliases: bool = False,
    accuracy: bool = False,
    enemy_mmr: bool = False,
    assets: bool = False,
    participants: bool = False,
    participants_scores: bool = False,
    participants_kda: bool = False,
    participants_shots: bool = False,
    participants_damage: bool = False,
    participants_avg_life: bool = False,
    killer_victim: bool = False,
    end_time: bool = False,
    sessions: bool = False,
    force_medals: bool = False,
    force_accuracy: bool = False,
    force_shots: bool = False,
    force_performance_scores: bool = False,
    force_enemy_mmr: bool = False,
    force_end_time: bool = False,
    force_sessions: bool = False,
    force_participants_shots: bool = False,
    force_participants_damage: bool = False,
    force_participants_avg_life: bool = False,
    shots: bool = False,
    citations: bool = False,
    force_citations: bool = False,
    gamertag: str = "",
    dry_run: bool = False,
    existing_shared_conn: Any | None = None,
) -> dict[str, int]:
    """Traitement des matchs via l'API SPNKr.

    .. deprecated:: v5.2
        Passer ``scope=SyncScope(...)`` au lieu des kwargs individuels.
    """
    # ── Construire / résoudre le scope ──
    if scope is None:
        scope = SyncScope(
            dry_run=dry_run,
            requests_per_second=requests_per_second,
            medals=medals,
            events=events,
            skill=skill,
            personal_scores=personal_scores,
            performance_scores=performance_scores,
            aliases=aliases,
            accuracy=accuracy,
            enemy_mmr=enemy_mmr,
            assets=assets,
            participants=participants,
            participants_scores=participants_scores,
            participants_kda=participants_kda,
            participants_shots=participants_shots,
            participants_damage=participants_damage,
            participants_avg_life=participants_avg_life,
            killer_victim=killer_victim,
            end_time=end_time,
            sessions=sessions,
            shots=shots,
            citations=citations,
            force_medals=force_medals,
            force_accuracy=force_accuracy,
            force_shots=force_shots,
            force_performance_scores=force_performance_scores,
            force_participants_shots=force_participants_shots,
            force_participants_damage=force_participants_damage,
            force_participants_avg_life=force_participants_avg_life,
            force_enemy_mmr=force_enemy_mmr,
            force_end_time=force_end_time,
            force_sessions=force_sessions,
            force_citations=force_citations,
        )
    scope.resolve()

    # Extraire les valeurs résolues en variables locales
    requests_per_second = scope.requests_per_second
    dry_run = scope.dry_run
    medals = scope.medals
    events = scope.events
    skill = scope.skill
    personal_scores = scope.personal_scores
    performance_scores = scope.performance_scores
    aliases = scope.aliases
    accuracy = scope.accuracy
    enemy_mmr = scope.enemy_mmr
    assets = scope.assets
    participants = scope.participants
    participants_scores = scope.participants_scores
    participants_kda = scope.participants_kda
    participants_shots = scope.participants_shots
    participants_damage = scope.participants_damage
    participants_avg_life = scope.participants_avg_life
    killer_victim = scope.killer_victim
    end_time = scope.end_time
    sessions = scope.sessions
    shots = scope.shots
    citations = scope.citations
    force_medals = scope.force_medals
    force_accuracy = scope.force_accuracy
    force_shots = scope.force_shots
    # force_skill est utilisé via scope dans find_matches_missing_data()
    force_performance_scores = scope.force_performance_scores
    force_enemy_mmr = scope.force_enemy_mmr
    force_end_time = scope.force_end_time
    force_sessions = scope.force_sessions
    force_citations = scope.force_citations
    pve_stats = getattr(scope, "pve_stats", False)
    force_pve_stats = getattr(scope, "force_pve_stats", False)
    from src.data.sync.api_client import get_tokens_from_env
    from src.data.sync.api_factory import create_api_client
    from src.data.sync.migrations import ensure_match_participants_columns
    from src.data.sync.transformers import (
        extract_aliases,
        extract_medals,
        extract_participants,
        extract_personal_score_awards,
        extract_xuids_from_match,
        transform_highlight_events,
        transform_match_stats,
        transform_personal_score_awards,
        transform_skill_stats,
    )

    tokens = await get_tokens_from_env()
    if not tokens:
        logger.error("Tokens SPNKr non disponibles")
        return _empty_result()

    # Réutiliser la connexion shared existante ou en ouvrir une nouvelle
    shared_conn = existing_shared_conn or _get_shared_connection(db_path)
    owns_shared_conn = existing_shared_conn is None  # True si on a ouvert nous-même
    if shared_conn is not None:
        from src.data.sync.migrations import ensure_match_participants_columns as _ensure_mp

        try:
            _ensure_mp(shared_conn)
            shared_conn.commit()
        except Exception:
            pass

    # v5 : Déterminer si on travaille uniquement sur shared DB
    _participants_only = (
        participants_scores
        or participants_kda
        or participants_shots
        or participants_damage
        or participants_avg_life
        or killer_victim
    ) and not any(
        [
            medals,
            events,
            skill,
            personal_scores,
            performance_scores,
            aliases,
            accuracy,
            enemy_mmr,
            assets,
            participants,
            end_time,
            sessions,
            citations,
            shots,
        ]
    )

    # Compteurs
    totals = _empty_result()
    totals["matches_checked"] = len(match_ids)
    totals["matches_missing_data"] = len(match_ids)

    async with create_api_client(
        tokens=tokens,
        requests_per_second=requests_per_second,
    ) as client:
        for i, match_id in enumerate(match_ids, 1):
            try:
                logger.info(f"[{i}/{len(match_ids)}] Traitement {match_id}...")

                stats_json = await client.get_match_stats(match_id)
                if not stats_json:
                    logger.warning(f"Impossible de récupérer {match_id}")
                    continue

                inserted = {}

                # ── Participants scores/kda/shots/damage/avg_life (UPDATE) ──
                if (
                    participants_scores
                    or participants_kda
                    or participants_shots
                    or participants_damage
                    or participants_avg_life
                ):
                    # V5 FINAL: toujours shared_conn (plus de fallback local)
                    mp_conn = shared_conn
                    ensure_match_participants_columns(mp_conn)
                    ps, pk, psh, pd, pal = _update_participants_details(
                        mp_conn,
                        stats_json,
                        participants_scores=participants_scores,
                        participants_kda=participants_kda,
                        participants_shots=participants_shots,
                        participants_damage=participants_damage,
                        participants_avg_life=participants_avg_life,
                    )
                    inserted["participants_scores"] = ps
                    inserted["participants_kda"] = pk
                    inserted["participants_shots"] = psh
                    inserted["participants_damage"] = pd
                    inserted["participants_avg_life"] = pal
                    totals["participants_scores_updated"] += ps
                    totals["participants_kda_updated"] += pk
                    totals["participants_shots_updated"] += psh
                    totals["participants_damage_updated"] += pd
                    totals["participants_avg_life_updated"] += pal

                # ── Assets (V5: shared.match_registry) ──
                if assets:
                    await _backfill_assets(client, shared_conn, stats_json, xuid, match_id)
                    totals["assets_updated"] += 1

                xuids = extract_xuids_from_match(stats_json)

                skill_json = None
                highlight_events: list = []

                if (skill or enemy_mmr) and xuids:
                    skill_json = await client.get_skill_stats(match_id, xuids)
                if events:
                    highlight_events = await client.get_highlight_events(match_id)

                # ── Accuracy / Shots (V5: shared.match_participants) ──
                if accuracy or shots:
                    match_row = transform_match_stats(stats_json, xuid)
                    if match_row and shared_conn is not None:
                        a, s = _update_accuracy_shots(
                            shared_conn,
                            match_row,
                            match_id,
                            xuid,
                            accuracy=accuracy,
                            shots=shots,
                            force_accuracy=force_accuracy,
                            force_shots=force_shots,
                        )
                        totals["accuracy_updated"] += a
                        totals["shots_updated"] += s

                # ── Médailles (V5: shared.medals_earned) ──
                if medals:
                    medal_rows = extract_medals(stats_json, xuid)
                    if medal_rows:
                        n = insert_medal_rows(shared_conn, medal_rows, xuid)
                        totals["medals_inserted"] += n
                    # Marquer medals_loaded=TRUE MÊME si 0 médailles insérées :
                    # certains modes (Assassin, events custom) ne donnent pas de médailles.
                    # Ne pas le faire ici causerait une boucle infinie de re-tentatives.
                    # Cohérent avec le comportement de engine.py lors du sync initial.
                    if shared_conn is not None:
                        shared_conn.execute(
                            "UPDATE match_registry SET medals_loaded = TRUE WHERE match_id = ?",
                            (match_id,),
                        )

                # ── Events (V5: shared.highlight_events) ──
                if events:
                    if highlight_events:
                        event_rows = transform_highlight_events(highlight_events, match_id)
                        if event_rows:
                            n = insert_event_rows(shared_conn, event_rows)
                            totals["events_inserted"] += n
                    # Fix v5.4: marquer events_loaded=TRUE quoi qu'il arrive (même si
                    # l'API retourne [] pour les modes sans events type BTB), pour éviter
                    # la re-détection infinie. events_loaded est la source de vérité.
                    shared_conn.execute(
                        "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?",
                        (match_id,),
                    )

                # ── Skill (V5: UPDATE shared.match_participants pour TOUS les joueurs) ──
                if skill and skill_json:
                    skill_row = transform_skill_stats(skill_json, match_id, xuid)
                    if skill_row:
                        n = _upsert_skill_to_participants(shared_conn, skill_row, xuid)
                        totals["skill_inserted"] += n

                # ── Enemy MMR (V5: shared.match_participants) ──
                if enemy_mmr and skill_json:
                    _update_enemy_mmr(shared_conn, skill_json, match_id, xuid, force_enemy_mmr)
                    totals["enemy_mmr_updated"] += 1

                # ── Personal scores (reste dans player DB) ──
                if personal_scores:
                    ps_data = extract_personal_score_awards(stats_json, xuid)
                    if ps_data:
                        ps_rows = transform_personal_score_awards(match_id, xuid, ps_data)
                        if ps_rows:
                            n = insert_personal_score_rows(conn, ps_rows)
                            totals["personal_scores_inserted"] += n

                # ── Aliases (V5: shared.xuid_aliases) ──
                if aliases:
                    alias_rows = extract_aliases(stats_json)
                    if alias_rows:
                        n = insert_alias_rows(shared_conn, alias_rows)
                        totals["aliases_inserted"] += n

                # ── Participants (V5: shared.match_participants) ──
                if participants:
                    participant_rows = extract_participants(stats_json)
                    if participant_rows:
                        n = insert_participant_rows(shared_conn, participant_rows)
                        totals["participants_inserted"] += n

                # ── Performance scores (V5: player_match_enrichment) ──
                if performance_scores and compute_performance_score_for_match(
                    conn,
                    match_id,
                    shared_conn=shared_conn,
                    xuid=xuid,
                    force=force_performance_scores,
                ):
                    totals["performance_scores_inserted"] += 1

                # ── Mise à jour backfill_bits (v5.2 — granulaire par joueur) ──
                _update_participant_backfill_bits(
                    shared_conn,
                    match_id,
                    xuid,
                    scope,
                    skill_inserted=bool(skill and skill_json),
                    medals_inserted=bool(medals and totals.get("medals_inserted", 0) > 0),
                )

                # ── PVE stats → shared_pve.duckdb (v5.2) ──
                if pve_stats:
                    n = await _backfill_pve_for_match(
                        stats_json,
                        match_id,
                        shared_conn,
                        db_path,
                        force=force_pve_stats,
                    )
                    totals["pve_stats_inserted"] += n

                # Marquer le bitmask backfill_completed pour tous les types demandés
                requested_types = scope.requested_types

                # Marquer backfill_completed
                # V5 FINAL: TOUJOURS dans shared.match_registry, jamais dans match_stats
                # Fix v5.4: exclure "events" du mask global — events_loaded est la source
                # de vérité et est géré directement ci-dessus. Le bit events dans
                # backfill_completed était la cause des faux positifs silencieux.
                mask = compute_backfill_mask(*(t for t in requested_types if t != "events"))
                if mask > 0:
                    try:
                        shared_conn.execute(
                            "UPDATE match_registry "
                            "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                            "WHERE match_id = ?",
                            [mask, match_id],
                        )
                    except Exception as e:
                        logger.debug(
                            f"Impossible de marquer backfill_completed dans match_registry: {e}"
                        )

                # Commit après chaque match
                conn.commit()
                if shared_conn is not None:
                    shared_conn.commit()

                logger.info(f"  ✅ Match {match_id[:20]}... traité")

            except Exception as e:
                logger.error(f"Erreur traitement {match_id}: {e}")
                import traceback

                traceback.print_exc()
                continue

    # ── Backfill local post-API ──
    if killer_victim:
        logger.info("Backfill des paires killer/victim depuis highlight_events...")
        kv_shared = shared_conn or _get_shared_connection(db_path)
        n = backfill_killer_victim_pairs(conn, xuid, shared_conn=kv_shared)
        if kv_shared is not shared_conn and kv_shared is not None:
            kv_shared.close()
        totals["killer_victim_pairs_inserted"] = n
        if n > 0:
            logger.info(f"✅ {n} paires killer/victim insérées")

    if end_time:
        logger.info("Backfill de l'heure de fin des matchs (end_time)...")
        n = backfill_end_time(conn, force=force_end_time, shared_conn=shared_conn)
        totals["end_time_updated"] = n
        if n > 0:
            logger.info(f"✅ {n} match(s) avec end_time mis à jour")

    # Fermer shared_conn AVANT sessions/citations pour libérer le verrou fichier
    # car ces fonctions ont besoin d'attacher shared à la connexion player
    # On ferme TOUJOURS (même si on ne l'a pas ouvert nous-même) car sessions/citations
    # vont attacher le fichier à conn et ont besoin du verrou exclusif
    if shared_conn is not None and (sessions or citations):
        try:
            shared_conn.commit()
            shared_conn.close()
            shared_conn = None
            owns_shared_conn = False  # Marquer comme fermé
        except Exception:
            pass

    if sessions:
        n = _backfill_sessions(conn, db_path, xuid, force=force_sessions, dry_run=dry_run)
        totals["sessions_updated"] = n

    if citations:
        from scripts.backfill.strategies import backfill_citations

        logger.info("Backfill des citations...")
        n = backfill_citations(conn, db_path, xuid, force=force_citations)
        totals["citations_computed"] = n

    # ── LUSR (calcul local, possible après API) ───────────────────────────────
    lusr_flag = getattr(scope, "lusr", False) or getattr(scope, "skill_rank", False)
    force_lusr_flag = getattr(scope, "force_lusr", False) or getattr(
        scope, "force_skill_rank", False
    )
    if lusr_flag:
        from scripts.backfill.strategies import compute_lusr_for_player

        logger.info("Calcul du LUSR (LevelUp Skill Rank) pour les matchs non classés...")
        # Ouvrir shared_conn si fermé (citations l'a peut-être fermé)
        _lusr_shared = _get_shared_connection(db_path) if shared_conn is None else shared_conn
        n = compute_lusr_for_player(
            conn, db_path, xuid, force=force_lusr_flag, shared_conn=_lusr_shared
        )
        if _lusr_shared is not shared_conn:
            with contextlib.suppress(Exception):
                _lusr_shared.close()
        totals["lusr_computed"] = n
        if n > 0:
            logger.info(f"✅ LUSR calculé pour {n} match(s)")

    # ── Snapshot CSR (fetch_csr) ─────────────────────────────────────────────
    fetch_csr_flag = getattr(scope, "fetch_csr", False)
    if fetch_csr_flag:
        from scripts.backfill.strategies import fetch_current_csr_for_player

        logger.info(f"Récupération snapshot CSR pour {gamertag} ({xuid})...")
        async with create_api_client(
            tokens=tokens,
            requests_per_second=requests_per_second,
        ) as csr_client:
            csr_results = await fetch_current_csr_for_player(conn, xuid, csr_client)
        totals["csr_fetched"] = len(csr_results)
        for pl_name, info in csr_results.items():
            logger.info(f"  {pl_name}: {info['tier']} {info['sub_tier']} (CSR {info['csr']})")

    # ── Backfill CSR par match (csr) ─────────────────────────────────────────
    csr_flag = getattr(scope, "csr", False)
    force_csr_flag = getattr(scope, "force_csr", False)
    if csr_flag:
        from scripts.backfill.strategies import backfill_csr_for_player

        logger.info(f"Backfill CSR par match pour {gamertag} ({xuid})...")
        async with create_api_client(
            tokens=tokens,
            requests_per_second=requests_per_second,
        ) as csr_api:
            n_csr = await backfill_csr_for_player(
                conn,
                db_path,
                xuid,
                csr_api,
                force=force_csr_flag,
                shared_conn=shared_conn if shared_conn is not None else None,
            )
        totals["csr_backfilled"] = n_csr
        if n_csr > 0:
            logger.info(f"✅ CSR backfill : {n_csr} match(s) écrits dans match_skill_rank")

    # Fermer shared_conn si on l'a ouvert et pas encore fermé
    if shared_conn is not None and owns_shared_conn:
        try:
            shared_conn.commit()
            shared_conn.close()
        except Exception:
            pass

    logger.info(f"Backfill terminé pour {gamertag}")
    return totals


# ─────────────────────────────────────────────────────────────────────────────
# Helpers de traitement par match
# ─────────────────────────────────────────────────────────────────────────────


def _update_participant_backfill_bits(
    shared_conn: Any,
    match_id: str,
    xuid: str,
    scope: SyncScope,
    *,
    skill_inserted: bool = False,
    medals_inserted: bool = False,
) -> None:
    """Met à jour ``backfill_bits`` dans ``match_participants`` après un backfill.

    Lit l'état réel de TOUTES les colonnes du participant dans la DB et pose
    les bits correspondants (OR incrémental). Approche inconditionnelle : pas
    besoin de lister les champs scope — on se base sur ce qui est NOT NULL.

    Args:
        shared_conn: Connexion à shared_matches.duckdb.
        match_id: ID du match.
        xuid: XUID du joueur.
        scope: Périmètre de données (conservé pour compatibilité de signature).
        skill_inserted: Inutilisé (conservé pour compatibilité).
        medals_inserted: Inutilisé (conservé pour compatibilité).
    """
    if shared_conn is None:
        return

    from src.data.sync.constants import ParticipantBits

    bits = 0

    # ── Scan inconditionnel : toutes les colonnes en une seule requête ──
    # On ne conditionne PAS sur scope : on lit ce qui est réellement NOT NULL.
    # Cela couvre correctement tous les champs granulaires (team_mmr, grenade_kills…)
    # quel que soit le sous-ensemble demandé.
    try:
        row = shared_conn.execute(
            "SELECT team_mmr, enemy_mmr, kills_expected, deaths_expected, assists_expected, "
            "accuracy, shots_fired, damage_dealt, avg_life_seconds, "
            "grenade_kills, melee_kills, power_weapon_kills, personal_score, "
            "headshot_kills, max_killing_spree, kda, time_played_seconds "
            "FROM match_participants WHERE match_id = ? AND xuid = ?",
            (match_id, xuid),
        ).fetchone()
        if row:
            if row[0] is not None:
                bits |= ParticipantBits.TEAM_MMR
            if row[1] is not None:
                bits |= ParticipantBits.ENEMY_MMR
            if row[2] is not None:
                bits |= ParticipantBits.KILLS_EXP
            if row[3] is not None:
                bits |= ParticipantBits.DEATHS_EXP
            if row[4] is not None:
                bits |= ParticipantBits.ASSISTS_EXP
            if row[5] is not None:
                bits |= ParticipantBits.ACCURACY
            if row[6] is not None:
                bits |= ParticipantBits.SHOTS
            if row[7] is not None:
                bits |= ParticipantBits.DAMAGE
            if row[8] is not None:
                bits |= ParticipantBits.AVG_LIFE
            if row[9] is not None:
                bits |= ParticipantBits.GRENADE_KILLS
            if row[10] is not None:
                bits |= ParticipantBits.MELEE_KILLS
            if row[11] is not None:
                bits |= ParticipantBits.POWER_WEAPON
            if row[12] is not None:
                bits |= ParticipantBits.PERSONAL_SCORE
            if row[13] is not None:
                bits |= ParticipantBits.HEADSHOT_KILLS
            if row[14] is not None:
                bits |= ParticipantBits.MAX_SPREE
            if row[15] is not None:
                bits |= ParticipantBits.KDA
            if row[16] is not None:
                bits |= ParticipantBits.TIME_PLAYED
    except Exception as e:
        logger.debug(f"Lecture participant bits pour {xuid}/{match_id}: {e}")

    # ── Médailles (table séparée) ──
    try:
        count = shared_conn.execute(
            "SELECT COUNT(*) FROM medals_earned WHERE match_id = ? AND xuid = ?",
            (match_id, xuid),
        ).fetchone()
        if count and count[0] > 0:
            bits |= ParticipantBits.MEDALS
    except Exception as e:
        logger.debug(f"Lecture medals bits pour {xuid}/{match_id}: {e}")

    if bits == 0:
        return

    try:
        shared_conn.execute(
            "UPDATE match_participants "
            "SET backfill_bits = COALESCE(backfill_bits, 0) | ? "
            "WHERE match_id = ? AND xuid = ?",
            (bits, match_id, xuid),
        )
    except Exception as e:
        logger.debug(f"Mise à jour backfill_bits pour {xuid}/{match_id}: {e}")


async def _backfill_pve_for_match(
    stats_json: dict,
    match_id: str,
    shared_conn: Any,
    db_path: Path,
    *,
    force: bool = False,
) -> int:
    """Backfill les stats PVE pour un match Firefight → shared_pve.duckdb.

    Pose ``MatchBits.PVE_STATS`` dans ``match_registry.backfill_completed``
    même si le match n'est pas un Firefight (guard anti-boucle infinie).

    Args:
        stats_json: JSON brut du match.
        match_id: ID du match.
        shared_conn: Connexion vers shared_matches.duckdb (pour le guard).
        db_path: Chemin de la DB joueur (pour dériver shared_pve.duckdb).
        force: Si True, insert même si PVE_STATS déjà posé.

    Returns:
        Nombre de lignes insérées (0 si pas un Firefight).
    """
    import duckdb

    from src.data.sync.batch_insert import batch_insert_pve_stats
    from src.data.sync.constants import MatchBits
    from src.data.sync.migrations import ensure_pve_schema
    from src.data.sync.transformers import extract_pve_stats

    try:
        pve_rows = extract_pve_stats(stats_json)

        from src.utils.paths import get_pve_db_path_from_player

        pve_path = get_pve_db_path_from_player(db_path)
        pve_path.parent.mkdir(parents=True, exist_ok=True)
        pve_conn = duckdb.connect(str(pve_path), read_only=False)
        try:
            ensure_pve_schema(pve_conn)
            inserted = batch_insert_pve_stats(pve_conn, pve_rows) if pve_rows else 0
            pve_conn.commit()
        finally:
            pve_conn.close()

        # Poser le guard dans match_registry (même si 0 lignes insérées)
        if shared_conn is not None:
            try:
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (MatchBits.PVE_STATS, match_id),
                )
            except Exception as e:
                logger.debug(f"Guard PVE_STATS pour {match_id}: {e}")

        return inserted
    except Exception as e:
        logger.warning(f"Backfill PVE échoué pour {match_id}: {e}")
        return 0


def _update_participants_details(
    conn: Any,
    stats_json: Any,
    *,
    participants_scores: bool,
    participants_kda: bool,
    participants_shots: bool,
    participants_damage: bool,
    participants_avg_life: bool = False,
) -> tuple[int, int, int, int, int]:
    """Met à jour les détails des participants (scores, kda, shots, damage, avg_life).

    Returns:
        Tuple (scores, kda, shots, damage, avg_life) nombre de mises à jour.
    """
    from src.data.sync.transformers import extract_participants

    participant_rows = extract_participants(stats_json)
    ps = pk = psh = pd = pal = 0

    for row in participant_rows:
        try:
            if participants_scores:
                conn.execute(
                    "UPDATE match_participants SET rank = ?, score = ? "
                    "WHERE match_id = ? AND xuid = ?",
                    (row.rank, row.score, row.match_id, row.xuid),
                )
            if participants_kda:
                conn.execute(
                    "UPDATE match_participants SET kills = ?, deaths = ?, assists = ? "
                    "WHERE match_id = ? AND xuid = ?",
                    (row.kills, row.deaths, row.assists, row.match_id, row.xuid),
                )
            if participants_shots and (row.shots_fired is not None or row.shots_hit is not None):
                conn.execute(
                    "UPDATE match_participants SET shots_fired = ?, shots_hit = ? "
                    "WHERE match_id = ? AND xuid = ?",
                    (row.shots_fired, row.shots_hit, row.match_id, row.xuid),
                )
                psh += 1
            if participants_damage and (
                row.damage_dealt is not None or row.damage_taken is not None
            ):
                conn.execute(
                    "UPDATE match_participants SET damage_dealt = ?, damage_taken = ? "
                    "WHERE match_id = ? AND xuid = ?",
                    (row.damage_dealt, row.damage_taken, row.match_id, row.xuid),
                )
                pd += 1
            if (
                participants_avg_life
                and hasattr(row, "avg_life_seconds")
                and row.avg_life_seconds is not None
            ):
                conn.execute(
                    "UPDATE match_participants SET avg_life_seconds = ? "
                    "WHERE match_id = ? AND xuid = ?",
                    (row.avg_life_seconds, row.match_id, row.xuid),
                )
                pal += 1
        except Exception as e:
            logger.debug(f"Update participant {row.xuid}: {e}")

    if participant_rows:
        if participants_scores:
            ps = len(participant_rows)
        if participants_kda:
            pk = len(participant_rows)

    return ps, pk, psh, pd, pal


def _update_accuracy_shots(
    shared_conn: Any,
    match_row: Any,
    match_id: str,
    xuid: str,
    *,
    accuracy: bool,
    shots: bool,
    force_accuracy: bool,
    force_shots: bool,
) -> tuple[int, int]:
    """Met à jour accuracy et/ou shots pour un match dans shared.match_participants (V5).

    Args:
        shared_conn: Connexion à shared_matches.duckdb
        match_row: Données du match (contient accuracy, shots_fired, shots_hit)
        match_id: ID du match
        xuid: XUID du joueur
        accuracy: Si True, mettre à jour accuracy
        shots: Si True, mettre à jour shots_fired/shots_hit
        force_accuracy: Force la mise à jour même si la valeur existe
        force_shots: Force la mise à jour même si les valeurs existent

    Returns:
        Tuple (accuracy_updated, shots_updated).
    """
    a_updated = 0
    s_updated = 0

    if accuracy and match_row.accuracy is not None:
        if force_accuracy:
            shared_conn.execute(
                "UPDATE match_participants SET accuracy = ? WHERE match_id = ? AND xuid = ?",
                (match_row.accuracy, match_id, xuid),
            )
            a_updated = 1
        else:
            existing = shared_conn.execute(
                "SELECT accuracy FROM match_participants WHERE match_id = ? AND xuid = ?",
                (match_id, xuid),
            ).fetchone()
            if existing and existing[0] is None:
                shared_conn.execute(
                    "UPDATE match_participants SET accuracy = ? WHERE match_id = ? AND xuid = ?",
                    (match_row.accuracy, match_id, xuid),
                )
                a_updated = 1

    if shots and (match_row.shots_fired is not None or match_row.shots_hit is not None):
        if force_shots:
            shared_conn.execute(
                "UPDATE match_participants SET shots_fired = ?, shots_hit = ? WHERE match_id = ? AND xuid = ?",
                (match_row.shots_fired, match_row.shots_hit, match_id, xuid),
            )
            s_updated = 1
        else:
            existing = shared_conn.execute(
                "SELECT shots_fired, shots_hit FROM match_participants WHERE match_id = ? AND xuid = ?",
                (match_id, xuid),
            ).fetchone()
            if existing and (existing[0] is None or existing[1] is None):
                shared_conn.execute(
                    "UPDATE match_participants SET shots_fired = ?, shots_hit = ? WHERE match_id = ? AND xuid = ?",
                    (match_row.shots_fired, match_row.shots_hit, match_id, xuid),
                )
                s_updated = 1

    return a_updated, s_updated


async def _backfill_assets(
    client: Any,
    shared_conn: Any,
    stats_json: Any,
    xuid: str,
    match_id: str,
) -> None:
    """Met à jour les noms d'assets (playlist, map, pair, game_variant) dans shared.match_registry (V5)."""
    from src.data.sync.api_client import enrich_match_info_with_assets
    from src.data.sync.transformers import create_metadata_resolver, transform_match_stats

    await enrich_match_info_with_assets(client, stats_json)

    metadata_resolver = create_metadata_resolver(None)
    match_row = transform_match_stats(stats_json, xuid, metadata_resolver=metadata_resolver)
    if match_row and (
        match_row.playlist_name
        or match_row.map_name
        or match_row.pair_name
        or match_row.game_variant_name
    ):
        shared_conn.execute(
            """UPDATE match_registry SET
                playlist_name = COALESCE(?, playlist_name),
                map_name = COALESCE(?, map_name),
                pair_name = COALESCE(?, pair_name),
                game_variant_name = COALESCE(?, game_variant_name)
                WHERE match_id = ?""",
            (
                match_row.playlist_name,
                match_row.map_name,
                match_row.pair_name,
                match_row.game_variant_name,
                match_id,
            ),
        )


def _update_enemy_mmr(
    shared_conn: Any,
    skill_json: Any,
    match_id: str,
    xuid: str,
    force: bool,
) -> None:
    """Met à jour enemy_mmr pour un match dans shared.match_participants (V5).

    Pipeline v5.1 : enemy_mmr est désormais correctement extrait par le
    transformer (corrigé : était ignoré `_ = mmr_data`).
    Cette fonction écrit aussi les 6 colonnes expected/stddev via
    _upsert_skill_to_participants().
    """
    from src.data.sync.transformers import transform_skill_stats

    skill_row = transform_skill_stats(skill_json, match_id, xuid)
    if not skill_row or skill_row.enemy_mmr is None:
        return

    if force:
        _upsert_skill_to_participants(shared_conn, skill_row, xuid)
    else:
        existing = shared_conn.execute(
            "SELECT team_mmr FROM match_participants WHERE match_id = ? AND xuid = ?",
            (match_id, xuid),
        ).fetchone()
        if existing is None:
            _upsert_skill_to_participants(shared_conn, skill_row, xuid)
        elif existing[0] is None:
            # Mettre à jour les colonnes MMR dans match_participants
            shared_conn.execute(
                "UPDATE match_participants SET team_mmr = ?, enemy_mmr = ?, kills_expected = ?, "
                "kills_stddev = ?, deaths_expected = ?, deaths_stddev = ?, "
                "assists_expected = ?, assists_stddev = ? "
                "WHERE match_id = ? AND xuid = ?",
                (
                    skill_row.team_mmr,
                    getattr(skill_row, "enemy_mmr", None),
                    skill_row.kills_expected,
                    skill_row.kills_stddev,
                    skill_row.deaths_expected,
                    skill_row.deaths_stddev,
                    skill_row.assists_expected,
                    skill_row.assists_stddev,
                    match_id,
                    xuid,
                ),
            )


def _upsert_skill_to_participants(conn: Any, skill_row: Any, xuid: str) -> int:
    """Upsert les données skill/MMR dans match_participants (V5).

    Remplace insert_skill_row (qui écrivait dans player_match_stats) :
    en V5, les colonnes MMR sont dans shared.match_participants.

    Pipeline v5.1 complet :
        team_mmr, enemy_mmr, kills_expected, kills_stddev,
        deaths_expected, deaths_stddev, assists_expected, assists_stddev.

    ⚠️ assists_expected/assists_stddev : toujours NULL (limitation API Halo,
    StatPerformances ne fournit que Kills et Deaths).

    Returns:
        1 si mis à jour, 0 sinon.
    """
    if not skill_row:
        return 0
    try:
        conn.execute(
            "UPDATE match_participants SET "
            "team_mmr = ?, enemy_mmr = ?, "
            "kills_expected = ?, kills_stddev = ?, "
            "deaths_expected = ?, deaths_stddev = ?, "
            "assists_expected = ?, assists_stddev = ? "
            "WHERE match_id = ? AND xuid = ?",
            (
                skill_row.team_mmr,
                getattr(skill_row, "enemy_mmr", None),
                skill_row.kills_expected,
                skill_row.kills_stddev,
                skill_row.deaths_expected,
                skill_row.deaths_stddev,
                skill_row.assists_expected,
                skill_row.assists_stddev,
                skill_row.match_id,
                xuid,
            ),
        )
        return 1
    except Exception as e:
        logger.debug(f"Upsert skill → match_participants: {e}")
        return 0
