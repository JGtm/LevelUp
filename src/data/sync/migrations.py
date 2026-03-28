"""Migrations de schéma DuckDB centralisées.

Ce module regroupe toutes les fonctions de migration de colonnes
utilisées à la fois par engine.py (sync) et backfill_data.py.
Cela évite la duplication de code et garantit la cohérence du schéma.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

from src.data._score_sql import NORM_ENEMY_TEAM_SCORE_SQL, NORM_MY_TEAM_SCORE_SQL

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)


def get_table_columns(conn: duckdb.DuckDBPyConnection, table_name: str) -> set[str]:
    """Retourne l'ensemble des noms de colonnes d'une table.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.

    Returns:
        Ensemble des noms de colonnes (vide si la table n'existe pas).
    """
    try:
        cols = conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_schema = 'main' AND table_name = ?",
            [table_name],
        ).fetchall()
        return {r[0] for r in cols} if cols else set()
    except Exception as e:
        logger.debug("Impossible de lire les colonnes de %s: %s", table_name, e)
        return set()


def table_exists(conn: duckdb.DuckDBPyConnection, table_name: str) -> bool:
    """Vérifie si une table existe dans le schéma main.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.

    Returns:
        True si la table existe.
    """
    try:
        result = conn.execute(
            "SELECT COUNT(*) FROM information_schema.tables "
            "WHERE table_schema = 'main' AND table_name = ?",
            [table_name],
        ).fetchone()
        return bool(result and result[0] > 0)
    except Exception:
        return False


def column_exists(conn: duckdb.DuckDBPyConnection, table_name: str, column_name: str) -> bool:
    """Vérifie si une colonne existe dans une table.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.
        column_name: Nom de la colonne.

    Returns:
        True si la colonne existe.
    """
    try:
        result = conn.execute(
            "SELECT COUNT(*) FROM information_schema.columns "
            "WHERE table_schema = 'main' AND table_name = ? AND column_name = ?",
            [table_name, column_name],
        ).fetchone()
        return bool(result and result[0] > 0)
    except Exception:
        return False


def _add_column_if_missing(
    conn: duckdb.DuckDBPyConnection,
    table_name: str,
    column_name: str,
    column_type: str,
    existing_cols: set[str] | None = None,
) -> bool:
    """Ajoute une colonne à une table si elle n'existe pas.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.
        column_name: Nom de la colonne à ajouter.
        column_type: Type SQL de la colonne.
        existing_cols: Colonnes existantes (optionnel, évite un query).

    Returns:
        True si la colonne a été ajoutée, False sinon.
    """
    if existing_cols is not None:
        is_missing = column_name not in existing_cols
    else:
        is_missing = not column_exists(conn, table_name, column_name)

    if is_missing:
        try:
            conn.execute(f"ALTER TABLE {table_name} ADD COLUMN {column_name} {column_type}")
            logger.info("Ajout de la colonne %s à %s", column_name, table_name)
            return True
        except Exception as e:
            logger.warning("Impossible d'ajouter %s à %s: %s", column_name, table_name, e)
    return False


# ─────────────────────────────────────────────────────────────────────────────
# Migrations match_stats
# ─────────────────────────────────────────────────────────────────────────────


def ensure_match_stats_columns(conn: duckdb.DuckDBPyConnection) -> None:
    """S'assure que match_stats a toutes les colonnes optionnelles.

    Colonnes ajoutées si manquantes :
    - accuracy (FLOAT)
    - end_time (TIMESTAMP)
    - session_id (INTEGER)
    - session_label (VARCHAR)
    - rank (SMALLINT)
    - damage_dealt (FLOAT)
    - personal_score (INTEGER)
    - performance_score (FLOAT)
    """
    if not table_exists(conn, "match_stats"):
        return

    col_names = get_table_columns(conn, "match_stats")

    migrations: list[tuple[str, str]] = [
        ("accuracy", "FLOAT"),
        ("end_time", "TIMESTAMP"),
        ("session_id", "INTEGER"),
        ("session_label", "VARCHAR"),
        ("rank", "SMALLINT"),
        ("damage_dealt", "FLOAT"),
        ("personal_score", "INTEGER"),
        ("performance_score", "FLOAT"),
    ]

    for col_name, col_type in migrations:
        _add_column_if_missing(conn, "match_stats", col_name, col_type, col_names)


def ensure_performance_score_column(conn: duckdb.DuckDBPyConnection) -> None:
    """S'assure que la colonne performance_score existe dans match_stats."""
    _add_column_if_missing(conn, "match_stats", "performance_score", "FLOAT")


def ensure_end_time_column(conn: duckdb.DuckDBPyConnection) -> None:
    """S'assure que la colonne end_time existe dans match_stats."""
    _add_column_if_missing(conn, "match_stats", "end_time", "TIMESTAMP")


# ─────────────────────────────────────────────────────────────────────────────
# Migrations match_participants
# ─────────────────────────────────────────────────────────────────────────────


def ensure_match_participants_backfill_bits(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute ``backfill_bits`` à ``match_participants`` si absent.

    Cette colonne est le nouveau système granulaire de tracking par joueur.
    Elle remplace progressivement l'utilisation de ``match_registry.backfill_completed``
    pour les données per-participant.

    Valeur par défaut : NULL (les anciens enregistrements n'ont pas encore été migrés).
    Utiliser toujours ``COALESCE(backfill_bits, 0)`` dans les requêtes.
    """
    if not table_exists(conn, "match_participants"):
        return
    _add_column_if_missing(conn, "match_participants", "backfill_bits", "INTEGER DEFAULT 0")
    # Index pour détection rapide des participants avec données manquantes
    try:
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_mp_backfill ON match_participants(xuid, backfill_bits)"
        )
    except Exception as e:
        logger.debug("Index idx_mp_backfill ignoré: %s", e)


def ensure_match_participants_columns(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute rank, score, kills, deaths, assists, shots, damage à match_participants si absents.

    Colonnes ajoutées si manquantes :
    - rank (SMALLINT)
    - score (INTEGER)
    - kills (SMALLINT)
    - deaths (SMALLINT)
    - assists (SMALLINT)
    - shots_fired (INTEGER)
    - shots_hit (INTEGER)
    - damage_dealt (FLOAT)
    - damage_taken (FLOAT)
    - avg_life_seconds (FLOAT)
    - headshot_kills (SMALLINT)
    - max_killing_spree (SMALLINT)
    - kda (FLOAT)
    - accuracy (FLOAT)
    - time_played_seconds (INTEGER)
    - grenade_kills (SMALLINT)
    - melee_kills (SMALLINT)
    - power_weapon_kills (SMALLINT)
    - personal_score (INTEGER) — V5 finale
    - team_mmr (FLOAT) — V5 finale
    - kills_expected (FLOAT) — V5 finale
    - kills_stddev (FLOAT) — V5 finale
    - deaths_expected (FLOAT) — V5 finale
    - deaths_stddev (FLOAT) — V5 finale
    - assists_expected (FLOAT) — V5 finale
    - assists_stddev (FLOAT) — V5 finale
    """
    if not table_exists(conn, "match_participants"):
        return

    col_names = get_table_columns(conn, "match_participants")

    migrations: list[tuple[str, str]] = [
        ("rank", "SMALLINT"),
        ("score", "INTEGER"),
        ("kills", "SMALLINT"),
        ("deaths", "SMALLINT"),
        ("assists", "SMALLINT"),
        ("shots_fired", "INTEGER"),
        ("shots_hit", "INTEGER"),
        ("damage_dealt", "FLOAT"),
        ("damage_taken", "FLOAT"),
        ("avg_life_seconds", "FLOAT"),
        ("headshot_kills", "SMALLINT"),
        ("max_killing_spree", "SMALLINT"),
        ("kda", "FLOAT"),
        ("accuracy", "FLOAT"),
        ("time_played_seconds", "INTEGER"),
        ("grenade_kills", "SMALLINT"),
        ("melee_kills", "SMALLINT"),
        ("power_weapon_kills", "SMALLINT"),
        # V5 finale - Colonnes MMR/Skill
        ("personal_score", "INTEGER"),
        ("team_mmr", "FLOAT"),
        ("enemy_mmr", "FLOAT"),
        ("kills_expected", "FLOAT"),
        ("kills_stddev", "FLOAT"),
        ("deaths_expected", "FLOAT"),
        ("deaths_stddev", "FLOAT"),
        ("assists_expected", "FLOAT"),
        ("assists_stddev", "FLOAT"),
        # v5.2 — Bitmask granulaire par joueur
        ("backfill_bits", "INTEGER DEFAULT 0"),
    ]

    for col_name, col_type in migrations:
        _add_column_if_missing(conn, "match_participants", col_name, col_type, col_names)


# ─────────────────────────────────────────────────────────────────────────────
# Migration highlight_events : id DEFAULT nextval(séquence)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_highlight_events_autoincrement(conn: duckdb.DuckDBPyConnection) -> None:
    """Migre highlight_events pour que id utilise nextval() comme DEFAULT.

    Problème legacy : certaines DB ont été créées sans séquence, donc
    INSERT sans spécifier id échoue avec NOT NULL constraint.
    DuckDB ne supporte pas ALTER COLUMN SET DEFAULT, il faut recréer la table.

    Cette migration est idempotente : si le DEFAULT est déjà correct, rien n'est fait.
    """
    if not table_exists(conn, "highlight_events"):
        return

    # Vérifier si id a déjà le bon DEFAULT
    try:
        col_info = conn.execute(
            "SELECT column_default FROM information_schema.columns "
            "WHERE table_schema = 'main' AND table_name = 'highlight_events' "
            "AND column_name = 'id'"
        ).fetchone()
    except Exception:
        return

    has_nextval = col_info and col_info[0] and "nextval" in str(col_info[0]).lower()

    # Récupérer le max id actuel pour initialiser la séquence
    max_id_row = conn.execute("SELECT COALESCE(MAX(id), 0) FROM highlight_events").fetchone()
    max_id = max_id_row[0] if max_id_row else 0

    if has_nextval:
        # La colonne a déjà nextval, mais la séquence peut être désynchronisée.
        # Recréer la table complète pour réinitialiser la séquence proprement.
        _recreate_highlight_events_with_sequence(conn, max_id)
        return

    # Pas de DEFAULT → recreation complète
    logger.info(
        "Migration highlight_events: ajout séquence auto-increment (max_id=%s, %s rows)",
        max_id,
        max_id_row,
    )
    _recreate_highlight_events_with_sequence(conn, max_id)


def _recreate_highlight_events_with_sequence(conn: duckdb.DuckDBPyConnection, max_id: int) -> None:
    """Recrée highlight_events avec id DEFAULT nextval(séquence).

    Copie toutes les données existantes, recrée les index.
    Note: On sauvegarde les données AVANT de drop la séquence CASCADE
    car le CASCADE supprime aussi la table dépendante.
    """
    # 1) Sauvegarder les données existantes avant tout DROP
    conn.execute("DROP TABLE IF EXISTS highlight_events_backup")
    if table_exists(conn, "highlight_events"):
        conn.execute("CREATE TABLE highlight_events_backup AS SELECT * FROM highlight_events")
        conn.execute("DROP TABLE highlight_events CASCADE")

    # 2) Nettoyer séquence/table résiduelle
    conn.execute("DROP TABLE IF EXISTS highlight_events_new")
    conn.execute("DROP SEQUENCE IF EXISTS highlight_events_id_seq CASCADE")

    # 3) Créer la nouvelle séquence et table
    conn.execute(f"CREATE SEQUENCE highlight_events_id_seq START WITH {max_id + 1}")
    conn.execute("""
        CREATE TABLE highlight_events (
            id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
            match_id VARCHAR NOT NULL,
            event_type VARCHAR NOT NULL,
            time_ms INTEGER,
            xuid VARCHAR,
            type_hint INTEGER,
            raw_json VARCHAR
        )
    """)

    # 4) Restaurer les données
    if table_exists(conn, "highlight_events_backup"):
        conn.execute("""
            INSERT INTO highlight_events (id, match_id, event_type, time_ms, xuid, type_hint, raw_json)
            SELECT id, match_id, event_type, time_ms, xuid, type_hint, raw_json
            FROM highlight_events_backup
        """)
        conn.execute("DROP TABLE highlight_events_backup")

    conn.execute("CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id)")
    logger.info("✅ highlight_events migrée avec séquence auto-increment (start=%s)", max_id + 1)


# ─────────────────────────────────────────────────────────────────────────────
# Migration career_progression : id DEFAULT nextval(séquence)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_career_progression_autoincrement(conn: duckdb.DuckDBPyConnection) -> None:
    """Migre career_progression pour que id utilise nextval() comme DEFAULT.

    Problème legacy : certaines DB ont été créées sans séquence, donc
    INSERT sans spécifier id échoue avec NOT NULL constraint.
    DuckDB ne supporte pas ALTER COLUMN SET DEFAULT, il faut recréer la table.

    Cette migration est idempotente : si le DEFAULT est déjà correct, rien n'est fait.
    """
    if not table_exists(conn, "career_progression"):
        return

    # Vérifier si id a déjà le bon DEFAULT
    try:
        col_info = conn.execute(
            "SELECT column_default FROM information_schema.columns "
            "WHERE table_schema = 'main' AND table_name = 'career_progression' "
            "AND column_name = 'id'"
        ).fetchone()
    except Exception:
        return

    has_nextval = col_info and col_info[0] and "nextval" in str(col_info[0]).lower()

    if has_nextval:
        # La colonne a déjà nextval, pas de migration nécessaire
        return

    # Récupérer le max id actuel pour initialiser la séquence
    max_id_row = conn.execute("SELECT COALESCE(MAX(id), 0) FROM career_progression").fetchone()
    max_id = max_id_row[0] if max_id_row else 0

    # Pas de DEFAULT → recreation complète
    logger.info("Migration career_progression: ajout séquence auto-increment (max_id=%s)", max_id)
    _recreate_career_progression_with_sequence(conn, max_id)


def _recreate_career_progression_with_sequence(
    conn: duckdb.DuckDBPyConnection, max_id: int
) -> None:
    """Recrée career_progression avec id DEFAULT nextval(séquence).

    Copie toutes les données existantes, recrée les index.
    """
    # 1) Sauvegarder les données existantes avant tout DROP
    conn.execute("DROP TABLE IF EXISTS career_progression_backup")
    if table_exists(conn, "career_progression"):
        conn.execute("CREATE TABLE career_progression_backup AS SELECT * FROM career_progression")
        conn.execute("DROP TABLE career_progression CASCADE")

    # 2) Nettoyer séquence/table résiduelle
    conn.execute("DROP SEQUENCE IF EXISTS career_progression_id_seq CASCADE")

    # 3) Créer la nouvelle séquence et table
    conn.execute(f"CREATE SEQUENCE career_progression_id_seq START WITH {max_id + 1}")
    conn.execute("""
        CREATE TABLE career_progression (
            id INTEGER PRIMARY KEY DEFAULT nextval('career_progression_id_seq'),
            xuid VARCHAR NOT NULL,
            rank INTEGER NOT NULL,
            rank_name VARCHAR,
            rank_tier VARCHAR,
            current_xp INTEGER,
            xp_for_next_rank INTEGER,
            xp_total INTEGER,
            is_max_rank BOOLEAN DEFAULT FALSE,
            adornment_path VARCHAR,
            spartan_id VARCHAR,
            recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)

    # 4) Restaurer les données (colonnes explicites pour tolérer un backup sans spartan_id)
    if table_exists(conn, "career_progression_backup"):
        conn.execute("""
            INSERT INTO career_progression
                (id, xuid, rank, rank_name, rank_tier, current_xp,
                 xp_for_next_rank, xp_total, is_max_rank, adornment_path, recorded_at)
            SELECT id, xuid, rank, rank_name, rank_tier, current_xp,
                   xp_for_next_rank, xp_total, is_max_rank, adornment_path, recorded_at
            FROM career_progression_backup
        """)
        conn.execute("DROP TABLE career_progression_backup")

    conn.execute("CREATE INDEX IF NOT EXISTS idx_career_xuid ON career_progression(xuid)")
    conn.execute("CREATE INDEX IF NOT EXISTS idx_career_date ON career_progression(recorded_at)")
    logger.info("✅ career_progression migrée avec séquence auto-increment (start=%s)", max_id + 1)


# ─────────────────────────────────────────────────────────────────────────────
# Migration career_progression : colonne spartan_id
# ──────────────────────────────────────────────────────────────────────────────


def add_spartan_id_to_career_progression(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute la colonne spartan_id à career_progression si elle est absente.

    Migration non-destructive : utilise ``ALTER TABLE ... ADD COLUMN IF NOT EXISTS``
    (supporté nativement par DuckDB).
    Les lignes existantes auront ``spartan_id = NULL``.
    """
    if not table_exists(conn, "career_progression"):
        return
    try:
        conn.execute("ALTER TABLE career_progression ADD COLUMN IF NOT EXISTS spartan_id VARCHAR")
        logger.debug("✅ career_progression : colonne spartan_id présente")
    except Exception as e:
        # DuckDB peut lever une erreur si la syntaxe IF NOT EXISTS n'est pas reconnue
        # dans certaines versions anciennes — on ignore silencieusement.
        logger.debug("Colonne spartan_id déjà existante ou erreur non fatale : %s", e)


# ─────────────────────────────────────────────────────────────────────────────


# ─────────────────────────────────────────────────────────────────────────────
# Backfill bitmask : colonne match_stats.backfill_completed
# ─────────────────────────────────────────────────────────────────────────────

BACKFILL_FLAGS: dict[str, int] = {
    "medals": 1 << 0,  # 1
    "events": 1 << 1,  # 2
    "skill": 1 << 2,  # 4
    "personal_scores": 1 << 3,  # 8
    # performance_scores supprimé : granularité joueur×match, pas par match.
    # Jamais relu pour décider de recalculer (détection via IS NULL dans player_match_enrichment).
    "accuracy": 1 << 5,  # 32
    "shots": 1 << 6,  # 64
    "enemy_mmr": 1 << 7,  # 128
    "assets": 1 << 8,  # 256
    "participants": 1 << 9,  # 512
    "participants_scores": 1 << 10,  # 1024
    "participants_kda": 1 << 11,  # 2048
    "participants_shots": 1 << 12,  # 4096
    "participants_damage": 1 << 13,  # 8192
    "aliases": 1 << 14,  # 16384
    "participants_avg_life": 1 << 15,  # 32768 - Ajouté pour éviter détection infinie
    # ── Weapon kills (v5.5) — OBSOLÈTE ──
    # lusr/csr supprimés : collisionnaient avec MatchBits.EVENTS (1<<16) et
    # MatchBits.ASSETS (1<<17) de constants.py. Jamais écrits en production.
    # Supprimés pour éliminer le risque de corruption silencieuse (cf. plan E.5).
    # Ce bit (18 = 262144) n'est jamais posé en production.
    # Source de vérité : MatchBits.WEAPON_KILLS = 1 << 21 dans constants.py.
    # Conservé pour rétrocompatibilité des tests uniquement.
    "weapon_kills": 1 << 18,  # 262144 — OBSOLÈTE, voir MatchBits.WEAPON_KILLS (1<<21)
    # ── Bits 19-22 : définis dans src.data.sync.constants.MatchBits ──
    # KILLER_VICTIM_LOADED = 1 << 19  (524288)
    # PVE_STATS            = 1 << 20  (1048576)
    # WEAPON_KILLS         = 1 << 21  (2097152) ← source de vérité weapon_kills
    # WEAPON_KILLS_NO_FILM = 1 << 22  (4194304)
}


def compute_backfill_mask(*types: str) -> int:
    """Calcule le masque de bits pour les types demandés.

    >>> compute_backfill_mask("medals", "events")
    3
    """
    mask = 0
    for t in types:
        mask |= BACKFILL_FLAGS.get(t, 0)
    return mask


def ensure_backfill_completed_column(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute la colonne backfill_completed (bitmask) à match_stats si absente.

    Note: En V5.1, match_stats n'existe plus dans les player DBs. Cette fonction
    est conservée pour compatibilité avec les anciennes DBs en migration.
    """
    if not table_exists(conn, "match_stats"):
        return
    _add_column_if_missing(conn, "match_stats", "backfill_completed", "INTEGER DEFAULT 0")


# ─────────────────────────────────────────────────────────────────────────────
# Migration medals_earned (INT32 → BIGINT)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_medals_earned_bigint(conn: duckdb.DuckDBPyConnection) -> bool:
    """Migre medal_name_id de INTEGER vers BIGINT si nécessaire.

    Certaines medal_name_id dépassent INT32, il faut utiliser BIGINT.
    DuckDB ne supporte pas ALTER COLUMN TYPE, il faut recréer la table.

    Returns:
        True si la migration a été effectuée.
    """
    if not table_exists(conn, "medals_earned"):
        return False

    try:
        col_info = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name = 'medals_earned' AND column_name = 'medal_name_id'"
        ).fetchone()

        if col_info and col_info[0] in ("INTEGER", "INT32"):
            logger.info("Migration du schéma medals_earned: INTEGER -> BIGINT...")
            conn.execute("""
                CREATE TABLE IF NOT EXISTS medals_earned_new (
                    match_id VARCHAR,
                    medal_name_id BIGINT,
                    count SMALLINT,
                    PRIMARY KEY (match_id, medal_name_id)
                )
            """)
            conn.execute("""
                INSERT INTO medals_earned_new
                SELECT match_id, CAST(medal_name_id AS BIGINT), count
                FROM medals_earned
            """)
            conn.execute("DROP TABLE medals_earned")
            conn.execute("ALTER TABLE medals_earned_new RENAME TO medals_earned")
            logger.info("✅ Schéma medals_earned migré vers BIGINT")
            return True
    except Exception as e:
        logger.warning("Migration medals_earned échouée (continuation): %s", e)

    return False


# ─────────────────────────────────────────────────────────────────────────────
# v5.1 — Vue mv_player_matches (optimisation performance)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_mv_player_matches_view(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ou met à jour la vue mv_player_matches dans shared_matches.duckdb.

    Cette vue pré-calcule toutes les expressions COALESCE/CASE WHEN
    qui étaient construites dynamiquement par _get_match_source(),
    réduisant ~150 lignes de SQL dynamique à une simple référence.

    La vue est créée dans le catalog 'shared' si disponible.
    Idempotente : CREATE OR REPLACE VIEW.
    """
    # Vérifier que la table source existe dans le bon catalog
    # duckdb_tables() fonctionne avec tous les catalogs (y compris attached)
    # contrairement à {catalog}.information_schema.tables qui échoue en DuckDB 1.4.x
    #
    # Note : pour un fichier connecté directement, database_name = nom du fichier
    # (ex: "shared_matches"), PAS "main". On n'utilise un prefix que si le catalog
    # est exactement "shared" (ATTACH AS shared), sinon pas de prefix pour que la
    # vue reste portable.
    catalog = None
    try:
        rows = conn.execute(
            "SELECT database_name FROM duckdb_tables() WHERE table_name = 'match_registry'"
        ).fetchall()
        for row in rows:
            db_name = row[0]
            if db_name == "shared":
                catalog = "shared"
                break
            elif catalog is None:
                catalog = db_name
    except Exception:
        pass

    if catalog is None:
        raise RuntimeError(
            "match_registry introuvable — vue mv_player_matches non créée "
            "(la migration réessaiera au prochain démarrage après le sync)."
        )

    # Seul "shared" (ATTACH) nécessite un prefix ; pour une connexion directe
    # au fichier (database_name = nom du fichier), pas de prefix.
    prefix = "shared." if catalog == "shared" else ""

    # Utiliser v_match_full (v6) si disponible, sinon match_registry
    match_source = f"{prefix}match_registry"
    try:
        conn.execute(f"SELECT 1 FROM {prefix}v_match_full LIMIT 1")
        match_source = f"{prefix}v_match_full"
    except Exception:
        pass

    # Vérifier si la colonne enemy_mmr existe dans match_participants
    # (peut manquer dans les anciennes DBs ou les tests)
    # Note v5.1 : enemy_mmr est désormais correctement peuplé par le pipeline
    # skill (corrigé : était ignoré `_ = mmr_data` dans transform_all_skill_stats).
    # La détection dynamique reste pour compatibilité avec les anciennes DBs.
    has_enemy_mmr = False
    try:
        cols = conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_name = 'match_participants' AND column_name = 'enemy_mmr'"
        ).fetchall()
        has_enemy_mmr = len(cols) > 0
    except Exception:
        pass

    enemy_mmr_expr = "p.enemy_mmr" if has_enemy_mmr else "NULL AS enemy_mmr"

    # KDA : valeur officielle de l'API (colonne p.kda de match_participants).
    # Aucun recalcul local. Si la colonne est absente (schéma de test minimal), NULL.
    has_kda_col = False
    try:
        cols = conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_name = 'match_participants' AND column_name = 'kda'"
        ).fetchall()
        has_kda_col = len(cols) > 0
    except Exception:
        pass
    kda_expr = "p.kda" if has_kda_col else "NULL"

    conn.execute(f"""
        CREATE OR REPLACE VIEW {prefix}mv_player_matches AS
        SELECT
            r.match_id,
            r.start_time,
            r.map_id,
            r.map_name,
            r.playlist_id,
            r.playlist_name,
            r.pair_id,
            r.pair_name,
            r.game_variant_id,
            r.game_variant_name,
            p.xuid,
            p.outcome,
            p.team_id,

            -- KDA : valeur API officielle si disponible, recalcul local sinon
            {kda_expr} AS kda,

            -- Stats de base
            COALESCE(p.max_killing_spree, 0) AS max_killing_spree,
            COALESCE(p.headshot_kills, 0) AS headshot_kills,
            COALESCE(p.avg_life_seconds, 0) AS avg_life_seconds,
            COALESCE(r.duration_seconds, 0) AS time_played_seconds,
            COALESCE(p.kills, 0) AS kills,
            COALESCE(p.deaths, 0) AS deaths,
            COALESCE(p.assists, 0) AS assists,

            -- Accuracy
            CASE WHEN p.shots_fired > 0
            THEN CAST(p.shots_hit AS FLOAT) * 100.0 / CAST(p.shots_fired AS FLOAT)
            ELSE NULL
            END AS accuracy,

            -- Scores d'équipe.
            -- Pour les modes objectifs (CTF, Total Control, Stockpile, One Flag),
            -- si le score brut > 100, l'API a stocké une valeur corrompue → NULL.
            -- Le transformer lit désormais CaptureTheFlagStats/ZonesStats en priorité.
            {NORM_MY_TEAM_SCORE_SQL} AS my_team_score,
            {NORM_ENEMY_TEAM_SCORE_SQL} AS enemy_team_score,

            -- Somme des scores personnels par équipe
            CASE WHEN p.team_id = 0 THEN r.team_0_ps_score
                 ELSE r.team_1_ps_score END AS my_team_ps_score,
            CASE WHEN p.team_id = 0 THEN r.team_1_ps_score
                 ELSE r.team_0_ps_score END AS enemy_team_ps_score,

            -- MMR (enrichi depuis skill API)
            p.team_mmr,
            {enemy_mmr_expr},

            -- Rang dans le match
            p.rank,

            -- Score personnel
            p.score AS personal_score,

            -- Flags
            COALESCE(r.is_firefight, FALSE) AS is_firefight,
            COALESCE(r.is_ranked, FALSE) AS is_ranked

        FROM {match_source} r
        JOIN {prefix}match_participants p
            ON r.match_id = p.match_id
    """)

    logger.info("✅ Vue mv_player_matches créée/mise à jour")


# ─────────────────────────────────────────────────────────────────────────────
# v5.1 — Index de performance
# ─────────────────────────────────────────────────────────────────────────────


def ensure_performance_indexes(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée les index pour optimiser les requêtes v5 sur shared_matches.

    Index créés (match_participants) :
    - idx_mp_xuid_match : (xuid, match_id) — filtre joueur + jointure
    - idx_mp_match_xuid : (match_id, xuid) — jointures inversées
    - idx_mp_xuid_team : (xuid, team_id, match_id) — list_top_teammates

    Index créés (match_registry) :
    - idx_mr_start_time : start_time DESC — tri chronologique

    Index créés (highlight_events) :
    - idx_events_match_type : (match_id, event_type) — load_first_event_times

    Index créés (medals_earned) :
    - idx_medals_full : (match_id, xuid, medal_name_id) — count_medal_by_match

    Idempotente : CREATE INDEX IF NOT EXISTS.
    """
    catalog = None
    try:
        rows = conn.execute(
            "SELECT database_name FROM duckdb_tables() WHERE table_name = 'match_participants'"
        ).fetchall()
        for row in rows:
            db_name = row[0]
            if db_name == "shared":
                catalog = "shared"
                break
            elif catalog is None:
                catalog = db_name
    except Exception:
        pass

    if catalog is None:
        logger.debug("match_participants non trouvée, index non créés")
        return

    prefix = "shared." if catalog == "shared" else ""

    # --- match_participants ---
    _create_index_safe(
        conn,
        f"CREATE INDEX IF NOT EXISTS idx_mp_xuid_match "
        f"ON {prefix}match_participants(xuid, match_id)",
        "idx_mp_xuid_match",
    )
    _create_index_safe(
        conn,
        f"CREATE INDEX IF NOT EXISTS idx_mp_match_xuid "
        f"ON {prefix}match_participants(match_id, xuid)",
        "idx_mp_match_xuid",
    )
    _create_index_safe(
        conn,
        f"CREATE INDEX IF NOT EXISTS idx_mp_xuid_team "
        f"ON {prefix}match_participants(xuid, team_id, match_id)",
        "idx_mp_xuid_team",
    )

    # --- match_registry ---
    _create_index_safe(
        conn,
        f"CREATE INDEX IF NOT EXISTS idx_mr_start_time ON {prefix}match_registry(start_time DESC)",
        "idx_mr_start_time",
    )

    # --- highlight_events (composite pour load_first_event_times) ---
    _create_index_safe(
        conn,
        f"CREATE INDEX IF NOT EXISTS idx_events_match_type "
        f"ON {prefix}highlight_events(match_id, event_type)",
        "idx_events_match_type",
    )

    # --- medals_earned (covering index pour count_medal_by_match) ---
    _create_index_safe(
        conn,
        f"CREATE INDEX IF NOT EXISTS idx_medals_full "
        f"ON {prefix}medals_earned(match_id, xuid, medal_name_id)",
        "idx_medals_full",
    )

    logger.info("✅ Index de performance shared créés/vérifiés")


def ensure_player_performance_indexes(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée les index de performance sur les tables locales du joueur (stats.duckdb).

    Index créés (match_stats) :
    - idx_ms_start_time : start_time — tri chronologique (ORDER BY universel)
    - idx_ms_session_id : session_id — GROUP BY dans mv_session_stats
    - idx_ms_playlist_id : playlist_id — filtre sidebar
    - idx_ms_map_id : map_id — filtre sidebar
    - idx_ms_outcome : outcome — agrégations win/loss
    - idx_ms_is_firefight : is_firefight — filtre PvP/Firefight

    Index créés (personal_score_awards) :
    - idx_psa_match_xuid : (match_id, xuid) — filtre composite

    Idempotente : CREATE INDEX IF NOT EXISTS.
    """
    if not table_exists(conn, "match_stats"):
        logger.debug("match_stats non trouvée, index locaux non créés")
        return

    _create_index_safe(
        conn,
        "CREATE INDEX IF NOT EXISTS idx_ms_start_time ON match_stats(start_time)",
        "idx_ms_start_time",
    )
    _create_index_safe(
        conn,
        "CREATE INDEX IF NOT EXISTS idx_ms_session_id ON match_stats(session_id)",
        "idx_ms_session_id",
    )
    _create_index_safe(
        conn,
        "CREATE INDEX IF NOT EXISTS idx_ms_playlist_id ON match_stats(playlist_id)",
        "idx_ms_playlist_id",
    )
    _create_index_safe(
        conn,
        "CREATE INDEX IF NOT EXISTS idx_ms_map_id ON match_stats(map_id)",
        "idx_ms_map_id",
    )
    _create_index_safe(
        conn,
        "CREATE INDEX IF NOT EXISTS idx_ms_outcome ON match_stats(outcome)",
        "idx_ms_outcome",
    )
    _create_index_safe(
        conn,
        "CREATE INDEX IF NOT EXISTS idx_ms_is_firefight ON match_stats(is_firefight)",
        "idx_ms_is_firefight",
    )

    # Index composite sur personal_score_awards
    if table_exists(conn, "personal_score_awards"):
        _create_index_safe(
            conn,
            "CREATE INDEX IF NOT EXISTS idx_psa_match_xuid "
            "ON personal_score_awards(match_id, xuid)",
            "idx_psa_match_xuid",
        )

    logger.info("✅ Index de performance locaux créés/vérifiés")


def _create_index_safe(conn: duckdb.DuckDBPyConnection, sql: str, index_name: str) -> None:
    """Exécute une instruction CREATE INDEX de façon sûre (idempotente).

    Args:
        conn: Connexion DuckDB.
        sql: Instruction SQL CREATE INDEX IF NOT EXISTS.
        index_name: Nom de l'index (pour le log).
    """
    try:
        conn.execute(sql)
    except Exception as e:
        err = str(e).lower()
        if "already exists" in err or "read only" in err:
            logger.debug("Index %s ignoré: %s", index_name, e)
        elif "does not exist" in err or "table with name" in err:
            logger.debug("Index %s ignoré (table absente): %s", index_name, e)
        else:
            logger.warning("Index %s non créé: %s", index_name, e)


# ─────────────────────────────────────────────────────────────────────────────
# Schéma PvE — shared_pve.duckdb (v5.2)
# ─────────────────────────────────────────────────────────────────────────────

PVE_SCHEMA_DDL = """
CREATE TABLE IF NOT EXISTS pve_match_stats (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,

    -- Stats globales PvE (validées depuis interface PveStats API)
    total_enemy_kills  INTEGER,          -- API: Kills
    boss_kills         INTEGER,          -- API: BossKills

    -- Kills par type d'ennemi (API confirmée)
    grunt_kills    INTEGER DEFAULT 0,    -- API: GruntKills
    elite_kills    INTEGER DEFAULT 0,    -- API: EliteKills
    jackal_kills   INTEGER DEFAULT 0,    -- API: JackalKills
    brute_kills    INTEGER DEFAULT 0,    -- API: BruteKills
    hunter_kills   INTEGER DEFAULT 0,    -- API: HunterKills
    skimmer_kills  INTEGER DEFAULT 0,    -- API: SkimmerKills
    sentinel_kills INTEGER DEFAULT 0,    -- API: SentinelKills
    marine_kills   INTEGER DEFAULT 0,    -- API: MarineKills

    -- Bitmask granulaire (v5.2) — quels champs ont été récupérés
    -- Voir PveBits dans src/data/sync/constants.py
    pve_bits       INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (match_id, xuid)
);

CREATE INDEX IF NOT EXISTS idx_pve_xuid     ON pve_match_stats(xuid);
CREATE INDEX IF NOT EXISTS idx_pve_match_id ON pve_match_stats(match_id);
"""


def ensure_pve_schema(conn: duckdb.DuckDBPyConnection) -> None:
    """Initialise le schéma PvE dans shared_pve.duckdb (idempotent).

    Crée la table ``pve_match_stats`` et ses index si absents.
    Applique aussi les migrations de colonnes manquantes si la table existait
    avec une ancienne version du schéma.
    À appeler au démarrage de toute connexion vers shared_pve.duckdb.

    Args:
        conn: Connexion DuckDB vers shared_pve.duckdb.
    """
    try:
        # DDL complet (idempotent via IF NOT EXISTS)
        for stmt in PVE_SCHEMA_DDL.strip().split(";"):
            stmt = stmt.strip()
            if stmt:
                try:
                    conn.execute(stmt)
                except Exception as e:
                    err = str(e).lower()
                    if "already exists" not in err:
                        logger.warning("Erreur DDL PvE : %s", e)

        # Migration v5.2 : ajout des colonnes manquantes si ancienne version du schéma
        _pve_migrations = [
            ("sentinel_kills", "INTEGER DEFAULT 0"),
            ("marine_kills", "INTEGER DEFAULT 0"),
        ]
        existing_cols = {
            row[0]
            for row in conn.execute(
                "SELECT column_name FROM information_schema.columns"
                " WHERE table_name = 'pve_match_stats'"
            ).fetchall()
        }
        for col_name, col_def in _pve_migrations:
            if col_name not in existing_cols:
                try:
                    conn.execute(f"ALTER TABLE pve_match_stats ADD COLUMN {col_name} {col_def}")
                    logger.info("Migration PvE : colonne '%s' ajoutée à pve_match_stats", col_name)
                except Exception as e:
                    logger.warning("Migration PvE '%s': %s", col_name, e)

        logger.debug("Schéma PvE initialisé (shared_pve.duckdb)")
    except Exception as e:
        logger.error("Impossible d'initialiser le schéma PvE : %s", e)


# ─────────────────────────────────────────────────────────────────────────────
# Schéma LUSR / CSR — match_skill_rank (v5.2, stats.duckdb par joueur)
# ─────────────────────────────────────────────────────────────────────────────

_MATCH_SKILL_RANK_DDL = """
CREATE TABLE IF NOT EXISTS match_skill_rank (
    match_id         VARCHAR PRIMARY KEY,
    rating_type      VARCHAR  NOT NULL,          -- 'LUSR' ou 'CSR'
    rating_value     FLOAT    NOT NULL,           -- Valeur du rating
    rating_deviation FLOAT,                       -- Déviation σ (NULL pour CSR)
    tier             VARCHAR,                     -- 'Gold', 'Platinum', 'Onyx', etc.
    tier_fr          VARCHAR,                     -- 'Or', 'Platine', 'Onyx', etc.
    sub_tier         SMALLINT DEFAULT 0,          -- 1-6 pour non-Onyx, 0 pour Onyx/CSR
    tier_label       VARCHAR,                     -- 'Or III', 'Onyx', etc.
    rating_delta     FLOAT,                       -- Delta vs match précédent du même type
    playlist_group   VARCHAR,                     -- 'ranked','competitive','social','fun'
    start_time       TIMESTAMP,                   -- Copie de match_registry.start_time (tri chronologique)
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
"""

_MATCH_SKILL_RANK_INDEXES = [
    "CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)",
    "CREATE INDEX IF NOT EXISTS idx_msr_playlist    ON match_skill_rank(playlist_group)",
]


def ensure_skill_history_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table ``skill_history`` si elle n'existe pas (idempotente).

    Stocke les snapshots CSR du joueur récupérés via get_playlist_csr
    ainsi que l'historique all_time_max par playlist.

    À appeler dans la DB ``stats.duckdb`` du joueur concerné.

    Args:
        conn: Connexion DuckDB vers stats.duckdb du joueur.
    """
    try:
        conn.execute("""
            CREATE TABLE IF NOT EXISTS skill_history (
                playlist_id    VARCHAR,
                recorded_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                csr            INTEGER,
                tier           VARCHAR,
                division       INTEGER,
                matches_played INTEGER
            )
        """)
        logger.debug("Table skill_history initialisée (stats.duckdb)")
    except Exception as e:
        logger.error("Impossible d'initialiser skill_history : %s", e)


def ensure_match_skill_rank_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table ``match_skill_rank`` si elle n'existe pas (idempotente).

    Cette table stocke le rating LUSR calculé localement pour les matchs
    non classés, et le CSR fourni par l'API pour les matchs classés.
    Un match ne peut avoir qu'un seul rating (PRIMARY KEY sur match_id).

    À appeler dans la DB ``stats.duckdb`` du joueur concerné.

    Args:
        conn: Connexion DuckDB vers stats.duckdb du joueur.
    """
    try:
        conn.execute(_MATCH_SKILL_RANK_DDL)
        for idx_sql in _MATCH_SKILL_RANK_INDEXES:
            try:
                conn.execute(idx_sql)
            except Exception as e:
                err = str(e).lower()
                if "already exists" not in err:
                    logger.warning("Index match_skill_rank non créé : %s", e)
        logger.debug("Table match_skill_rank initialisée (stats.duckdb)")
    except Exception as e:
        logger.error("Impossible d'initialiser match_skill_rank : %s", e)


# ─────────────────────────────────────────────────────────────────────────────
# Migration match_registry : colonne sync_spnkr_version (v5.4)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_match_registry_spnkr_version(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute la colonne sync_spnkr_version à match_registry si absente.

    Stocke la version de SPNKr utilisée lors de la sync initiale du match.
    Permet de détecter les matchs syncés avec une version ayant un bug connu
    (ex : highlight_events film v41 cassé avant SPNKr 0.10.1).
    """
    if not table_exists(conn, "match_registry"):
        return
    _add_column_if_missing(conn, "match_registry", "sync_spnkr_version", "VARCHAR")


# ─────────────────────────────────────────────────────────────────────────────
# Migration player_match_enrichment : colonne had_bot_teammate (v5.5)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_bot_teammate_column(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute la colonne had_bot_teammate à player_match_enrichment si absente.

    Indique si un coéquipier était un bot IA (remplacement en cours de match)
    dans le même team_id que le joueur. Utilisé pour assouplir le score de
    performance (le joueur ne devrait pas être pénalisé pour un coéquipier bot).

    Bots identifiés par xuid LIKE 'bid(%' dans match_participants.
    """
    if not table_exists(conn, "player_match_enrichment"):
        return
    _add_column_if_missing(conn, "player_match_enrichment", "had_bot_teammate", "BOOLEAN")


# ─────────────────────────────────────────────────────────────────────────────
# Migration player_match_enrichment : colonne dominance_flag (v5.7)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_dominance_flag_column(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute la colonne dominance_flag à player_match_enrichment si absente.

    Qualifie le degré de domination d'un match :
    - NULL : données médailles absentes (medals_loaded=FALSE)
    - 0 : match normal (pas de Steaktacular)
    - 1 : domination totale (notre équipe a obtenu Steaktacular)
    - 2 : humiliation totale (l'équipe ennemie a obtenu Steaktacular)

    Voir ``src.analysis._medal_verdicts.DominanceFlag``.
    """
    if not table_exists(conn, "player_match_enrichment"):
        return
    _add_column_if_missing(conn, "player_match_enrichment", "dominance_flag", "TINYINT")


# ─────────────────────────────────────────────────────────────────────────────
# Correction bot XIDs (bug legacy migrate_sqlite) — shared_matches.duckdb
# ─────────────────────────────────────────────────────────────────────────────


def ensure_fix_bot_xuid_shared(conn: duckdb.DuckDBPyConnection) -> None:
    """Corrige les XIDs de bots sans parenthèse fermante dans match_participants.

    Bug d'origine : recover_from_sqlite.py utilisait
    ``player_id.replace('xuid(', '').replace(')', '')`` qui supprimait
    la ')' fermante de ``bid(0.0)`` → ``bid(0.0``.

    Cette migration corrige les 506 entrées affectées en ajoutant la ')'
    manquante, rendant les clés cohérentes avec BOT_MAP.
    Idempotente : ne touche que les lignes sans ')' finale.
    """
    if not table_exists(conn, "match_participants"):
        return
    try:
        result = conn.execute(
            "UPDATE match_participants SET xuid = xuid || ')' "
            "WHERE xuid LIKE 'bid(%' AND xuid NOT LIKE 'bid(%)'"
        )
        rows_affected = result.fetchone()
        logger.info(
            "✅ fix_bot_xuid_shared : %s xuid(s) de bots corrigés",
            rows_affected[0] if rows_affected else "?",
        )
    except Exception as e:
        logger.warning("ensure_fix_bot_xuid_shared : erreur non fatale : %s", e)


# ─────────────────────────────────────────────────────────────────────────────
# Scores PS (personal score sums) — shared_matches.duckdb
# ─────────────────────────────────────────────────────────────────────────────


def ensure_team_ps_scores(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute team_0_ps_score / team_1_ps_score à match_registry.

    Ces colonnes stockent la somme des scores personnels de chaque équipe
    calculée depuis match_participants, indépendamment du champ API
    team_0_score / team_1_score (CoreStats.Score) qui peut mélanger le score
    objectif et la somme des scores perso selon les versions de l'API Halo.

    Idempotente : utilise ADD COLUMN IF NOT EXISTS.
    """
    if not table_exists(conn, "match_registry"):
        return
    _add_column_if_missing(conn, "match_registry", "team_0_ps_score", "INTEGER")
    _add_column_if_missing(conn, "match_registry", "team_1_ps_score", "INTEGER")
    logger.info("✅ ensure_team_ps_scores : colonnes team_0/1_ps_score présentes")


# ─────────────────────────────────────────────────────────────────────────────
# Schéma weapon_kills — shared_matches.duckdb (v5.7, per-kill avec weapon_id UBIGINT)
# ─────────────────────────────────────────────────────────────────────────────

_WEAPON_KILLS_DDL = """\
CREATE TABLE IF NOT EXISTS weapon_kills (
    match_id       VARCHAR  NOT NULL,
    xuid           VARCHAR  NOT NULL,
    time_ms        INTEGER  NOT NULL,
    weapon_id      UBIGINT,
    delta_ms       INTEGER,
    confidence     VARCHAR  NOT NULL DEFAULT 'none',
    swap_detected  BOOLEAN  NOT NULL DEFAULT FALSE,
    delayed_damage BOOLEAN  NOT NULL DEFAULT FALSE
)
"""

_WEAPON_KILLS_LEGACY_COLUMNS = {"weapon_name", "kills"}


def _migrate_weapon_kills_schema(conn: duckdb.DuckDBPyConnection) -> None:
    """Migre weapon_kills vers le schéma v5.7 (weapon_id UBIGINT) si nécessaire."""
    try:
        rows = conn.execute(
            "SELECT column_name, data_type FROM information_schema.columns "
            "WHERE table_name = 'weapon_kills'"
        ).fetchall()
        cols = {r[0] for r in rows}
        col_types = {r[0]: r[1] for r in rows}
        # Legacy v5.6 (weapon_name VARCHAR) → convertir données vers weapon_id
        if cols & _WEAPON_KILLS_LEGACY_COLUMNS:
            _convert_weapon_name_to_id(conn)
        # Legacy BIGINT → UBIGINT (certains IDs dépassent INT64 max)
        elif "weapon_id" in col_types and col_types["weapon_id"].upper() == "BIGINT":
            _upgrade_weapon_id_bigint_to_ubigint(conn)
    except Exception:
        pass  # table absente → sera créée par _WEAPON_KILLS_DDL


def _convert_weapon_name_to_id(conn: duckdb.DuckDBPyConnection) -> None:
    """Convertit weapon_name VARCHAR → weapon_id UBIGINT en préservant les données.

    Gère deux formes d'ancien schéma :
    - Per-kill (time_ms, delta_ms, …) → conversion des données
    - Agrégé (kills INTEGER, sans time_ms) → DROP+CREATE (non convertible)
    """
    from src.analysis._weapon_data import (
        GRENADE_WEAPON_ID,
        MELEE_WEAPON_ID,
        WEAPON_NAME_TO_INT,
    )

    cols = {
        r[0]
        for r in conn.execute(
            "SELECT column_name FROM information_schema.columns WHERE table_name = 'weapon_kills'"
        ).fetchall()
    }

    # Ancien schéma agrégé (weapon_name + kills, pas de time_ms) → non convertible
    if "time_ms" not in cols:
        logger.info("Migration weapon_kills : ancien schéma agrégé → DROP+CREATE")
        conn.execute("DROP TABLE weapon_kills")
        return

    logger.info("Migration weapon_kills → schéma v5.7 weapon_id (conversion données)")
    names = conn.execute("SELECT DISTINCT weapon_name FROM weapon_kills").fetchall()

    mapping: list[tuple[str, int | None]] = []
    for (name,) in names:
        if name in WEAPON_NAME_TO_INT:
            mapping.append((name, WEAPON_NAME_TO_INT[name]))
        elif name == "MELEE":
            mapping.append((name, MELEE_WEAPON_ID))
        elif name == "GRENADE":
            mapping.append((name, GRENADE_WEAPON_ID))
        elif name.startswith("?"):
            try:
                mapping.append((name, int(name[1:], 16)))
            except ValueError:
                mapping.append((name, None))
        elif name.startswith("INCONNU (") and name.endswith(")"):
            try:
                mapping.append((name, int(name[9:-1], 16)))
            except ValueError:
                mapping.append((name, None))
        else:
            # UNKNOWN, NON TROUVE, Spike Grenade, etc. → NULL
            mapping.append((name, None))

    conn.execute("CREATE TEMP TABLE _wk_map (weapon_name VARCHAR, weapon_id UBIGINT)")
    conn.executemany("INSERT INTO _wk_map VALUES (?, ?)", mapping)

    conn.execute("""
        CREATE TABLE weapon_kills_new AS
        SELECT wk.match_id, wk.xuid, wk.time_ms, m.weapon_id,
               wk.delta_ms, wk.confidence, wk.swap_detected, wk.delayed_damage
        FROM weapon_kills wk
        LEFT JOIN _wk_map m ON wk.weapon_name = m.weapon_name
    """)
    conn.execute("DROP TABLE weapon_kills")
    conn.execute("ALTER TABLE weapon_kills_new RENAME TO weapon_kills")
    conn.execute("DROP TABLE IF EXISTS _wk_map")

    count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
    non_null = conn.execute(
        "SELECT COUNT(*) FROM weapon_kills WHERE weapon_id IS NOT NULL"
    ).fetchone()[0]
    logger.info(
        "✅ Migration weapon_kills terminée : %d lignes (%d avec weapon_id)",
        count,
        non_null,
    )


def _upgrade_weapon_id_bigint_to_ubigint(
    conn: duckdb.DuckDBPyConnection,
) -> None:
    """Convertit weapon_id BIGINT → UBIGINT en préservant les données."""
    logger.info("Migration weapon_kills : weapon_id BIGINT → UBIGINT")
    conn.execute("""
        CREATE TABLE weapon_kills_new AS
        SELECT match_id, xuid, time_ms,
               CAST(weapon_id AS UBIGINT) AS weapon_id,
               delta_ms, confidence, swap_detected, delayed_damage
        FROM weapon_kills
    """)
    conn.execute("DROP TABLE weapon_kills")
    conn.execute("ALTER TABLE weapon_kills_new RENAME TO weapon_kills")
    count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
    logger.info("✅ Migration BIGINT→UBIGINT terminée : %d lignes", count)


def ensure_weapon_kills_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table ``weapon_kills`` si elle n'existe pas (idempotente).

    Schéma per-kill v5.7 : une ligne par kill, avec weapon_id (UBIGINT),
    delta_ms, confidence, swap_detected, delayed_damage.
    À appeler sur la connexion ``shared_matches.duckdb``.
    """
    try:
        _migrate_weapon_kills_schema(conn)
        conn.execute(_WEAPON_KILLS_DDL)
        try:
            conn.execute(
                "CREATE INDEX IF NOT EXISTS idx_wk_match_xuid ON weapon_kills(match_id, xuid)"
            )
        except Exception as e:
            err = str(e).lower()
            if "already exists" not in err:
                logger.warning("Index weapon_kills non créé : %s", e)
        logger.debug("Table weapon_kills initialisée (shared_matches.duckdb)")
    except Exception as e:
        logger.error("Impossible d'initialiser weapon_kills : %s", e)


def ensure_weapon_kills_reconciled_as(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute les colonnes reconciled_as, attribution_path, player_index à weapon_kills.

    Parser v2 : reconciled_as stocke le sentinel API sans écraser weapon_id,
    attribution_path trace la source (fire_event/formula_a/none),
    player_index identifie le joueur dans le film.
    Idempotente.
    """
    _add_column_if_missing(conn, "weapon_kills", "reconciled_as", "UBIGINT")
    _add_column_if_missing(conn, "weapon_kills", "attribution_path", "VARCHAR DEFAULT 'none'")
    _add_column_if_missing(conn, "weapon_kills", "player_index", "INTEGER")

    # Vue v_weapon_kills : COALESCE(reconciled_as, weapon_id)
    conn.execute(
        "CREATE OR REPLACE VIEW v_weapon_kills AS "
        "SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id "
        "FROM weapon_kills"
    )


# ─────────────────────────────────────────────────────────────────────────────
# v6 — Vues de résolution d'IDs (abstraction couche SQL)
# ─────────────────────────────────────────────────────────────────────────────


def _detect_shared_prefix(conn: duckdb.DuckDBPyConnection, table: str) -> str | None:
    """Retourne le préfixe catalog ("shared." ou "") pour une table dans shared.

    Retourne None si la table n'est pas trouvée.
    """
    try:
        rows = conn.execute(
            "SELECT database_name FROM duckdb_tables() WHERE table_name = ?",
            [table],
        ).fetchall()
        for row in rows:
            db_name = row[0]
            if db_name == "shared":
                return "shared."
            if db_name:
                return ""
    except Exception:
        pass
    return None


def _create_v_gamertag_lookup(conn: duckdb.DuckDBPyConnection, prefix: str) -> None:
    """Crée la vue v_gamertag_lookup dans shared_matches.duckdb.

    Résolution XUID → gamertag courant.
    Priorité : xuid_aliases > match_participants (FULL OUTER JOIN).
    Filtre les lignes dont le gamertag est NULL.
    """
    conn.execute(f"""
        CREATE OR REPLACE VIEW {prefix}v_gamertag_lookup AS
        SELECT
            COALESCE(xa.xuid, mp.xuid) AS xuid,
            COALESCE(xa.gamertag, mp.gamertag) AS gamertag
        FROM {prefix}xuid_aliases xa
        FULL OUTER JOIN (
            SELECT xuid, MAX(gamertag) AS gamertag
            FROM {prefix}match_participants
            WHERE gamertag IS NOT NULL
            GROUP BY xuid
        ) mp ON xa.xuid = mp.xuid
        WHERE COALESCE(xa.gamertag, mp.gamertag) IS NOT NULL
    """)
    logger.info("✅ Vue v_gamertag_lookup créée/mise à jour")


def _create_v_match_full(
    conn: duckdb.DuckDBPyConnection,
    prefix: str,
    meta_alias: str | None,
) -> None:
    """Crée la vue v_match_full dans shared_matches.duckdb.

    Résout les noms d'assets depuis metadata.duckdb (via meta_alias) si disponible.
    Si meta_alias est None, les colonnes *_fr et mode_* sont NULL,
    et les colonnes EN tombent en fallback sur match_registry (comportement actuel).
    """
    if meta_alias:
        map_en = "COALESCE(m.name_en, mr.map_name)"
        pl_en = "COALESCE(p.name_en, mr.playlist_name)"
        pp_en = "COALESCE(pp.name_en, mr.pair_name)"
        gv_en = "COALESCE(gv.name_en, mr.game_variant_name)"
        joins = f"""
        LEFT JOIN {meta_alias}.maps m ON mr.map_id = m.asset_id
        LEFT JOIN {meta_alias}.playlists p ON mr.playlist_id = p.asset_id
        LEFT JOIN {meta_alias}.playlist_map_mode_pairs pp ON mr.pair_id = pp.asset_id
        LEFT JOIN {meta_alias}.game_variants gv ON mr.game_variant_id = gv.asset_id"""
        fr_cols = """
        m.name_fr                                    AS map_name_fr,
        p.name_fr                                    AS playlist_name_fr,
        pp.name_fr                                   AS pair_name_fr,
        gv.name_fr                                   AS game_variant_name_fr,
        gv.mode_name                                 AS mode_name,
        gv.mode_name_fr                              AS mode_name_fr,
        p.playlist_canonical_en                      AS playlist_canonical_en,
        p.playlist_canonical_fr                      AS playlist_canonical_fr"""
    else:
        map_en = "mr.map_name"
        pl_en = "mr.playlist_name"
        pp_en = "mr.pair_name"
        gv_en = "mr.game_variant_name"
        joins = ""
        fr_cols = """
        NULL AS map_name_fr,
        NULL AS playlist_name_fr,
        NULL AS pair_name_fr,
        NULL AS game_variant_name_fr,
        NULL AS mode_name,
        NULL AS mode_name_fr,
        NULL AS playlist_canonical_en,
        NULL AS playlist_canonical_fr"""

    conn.execute(f"""
        CREATE OR REPLACE VIEW {prefix}v_match_full AS
        SELECT
            mr.match_id,
            mr.start_time,
            mr.duration_seconds,
            mr.map_id,
            mr.playlist_id,
            mr.pair_id,
            mr.game_variant_id,
            mr.team_0_score,
            mr.team_1_score,
            mr.team_0_ps_score,
            mr.team_1_ps_score,
            mr.is_firefight,
            mr.is_ranked,
            mr.backfill_completed,
            mr.events_loaded,
            mr.medals_loaded,
            mr.participants_loaded,
            mr.sync_spnkr_version,
            {map_en}                                 AS map_name,
            {pl_en}                                  AS playlist_name,
            {pp_en}                                  AS pair_name,
            {gv_en}                                  AS game_variant_name,
            {fr_cols}
        FROM {prefix}match_registry mr{joins}
    """)
    logger.info("✅ Vue v_match_full créée/mise à jour (meta_alias=%s)", meta_alias)


def _create_v_killer_victim_full(conn: duckdb.DuckDBPyConnection, prefix: str) -> None:
    """Crée la vue v_killer_victim_full dans shared_matches.duckdb.

    Résout killer_gamertag / victim_gamertag via v_gamertag_lookup.
    Chaîne : vue (courant) > snapshot figé > xuid brut.
    """
    conn.execute(f"""
        CREATE OR REPLACE VIEW {prefix}v_killer_victim_full AS
        SELECT
            kv.match_id,
            kv.killer_xuid,
            COALESCE(vk.gamertag, kv.killer_gamertag, kv.killer_xuid) AS killer_gamertag,
            kv.victim_xuid,
            COALESCE(vv.gamertag, kv.victim_gamertag, kv.victim_xuid) AS victim_gamertag,
            kv.kill_count,
            kv.time_ms,
            kv.is_validated
        FROM {prefix}killer_victim_pairs kv
        LEFT JOIN {prefix}v_gamertag_lookup vk ON kv.killer_xuid = vk.xuid
        LEFT JOIN {prefix}v_gamertag_lookup vv ON kv.victim_xuid = vv.xuid
    """)
    logger.info("✅ Vue v_killer_victim_full créée/mise à jour")


def ensure_resolution_views(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ou met à jour les 3 vues de résolution d'IDs dans shared_matches.duckdb.

    Vues créées (idempotentes via CREATE OR REPLACE VIEW) :
    - v_gamertag_lookup : XUID → gamertag courant
    - v_match_full      : match_registry + noms résolus depuis metadata.duckdb
    - v_killer_victim_full : killer_victim_pairs + gamertags résolus

    Précondition : la connexion doit pointer sur shared_matches.duckdb
    (directement ou via ATTACH AS shared).
    """
    prefix = _detect_shared_prefix(conn, "match_registry")
    if prefix is None:
        logger.warning("ensure_resolution_views: match_registry introuvable, vues non créées")
        return

    _create_v_gamertag_lookup(conn, prefix)

    meta_alias = _try_attach_meta_for_views(conn)
    _create_v_match_full(conn, prefix, meta_alias)
    _create_v_killer_victim_full(conn, prefix)

    logger.info("✅ Vues de résolution v6 créées/mises à jour")


def _try_attach_meta_for_views(conn: duckdb.DuckDBPyConnection) -> str | None:
    """Tente d'attacher metadata.duckdb pour que v_match_full puisse résoudre les noms FR.

    Retourne l'alias si réussi ET que la table maps existe, None sinon.
    """
    from src.utils.db import ensure_metadata_attached

    alias = None
    try:
        alias = ensure_metadata_attached(conn)
    except Exception as e:
        logger.debug("event=meta_attach_failed step=try_attach_meta_for_views error=%s", e)

    if alias is None:
        return None

    # Vérifier que les tables i18n sont bien peuplées (Commit 0 requis)
    try:
        rows = conn.execute(
            "SELECT table_name FROM duckdb_tables() WHERE table_name = 'maps'"
        ).fetchall()
        for _row in rows:
            # Si la table est dans le bon catalog
            db_name_rows = conn.execute(
                "SELECT database_name FROM duckdb_tables() WHERE table_name = 'maps'"
            ).fetchall()
            if any(r[0] == alias for r in db_name_rows):
                return alias
    except Exception as e:
        logger.debug("event=meta_maps_check_failed step=try_attach_meta_for_views error=%s", e)
    return None


# ─────────────────────────────────────────────────────────────────────────────
# metadata.duckdb — Référentiel weapon_labels
# ─────────────────────────────────────────────────────────────────────────────

_WEAPON_LABELS_DDL = """
CREATE TABLE IF NOT EXISTS weapon_labels (
    weapon_id UBIGINT PRIMARY KEY,
    name_en   VARCHAR NOT NULL,
    name_fr   VARCHAR NOT NULL
)
"""


def ensure_weapon_labels(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée et peuple ``weapon_labels`` dans metadata.duckdb (idempotente).

    Stocke les labels EN/FR pour chaque weapon_id filmshell (UBIGINT) et les
    3 IDs sentinelles (0=Grenade, 1=Melee, 2=Vehicle).
    Les lignes existantes ne sont jamais écrasées (INSERT OR IGNORE).
    """
    from src.analysis._weapon_data import (
        WEAPON_FUSION_MAP,
        WEAPON_INT_TO_NAME,
        WEAPON_NAME_FR,
    )

    conn.execute(_WEAPON_LABELS_DDL)

    rows: list[tuple[int, str, str]] = [
        # Sentinelles
        (0, "Grenade", "Grenade"),
        (1, "Melee", "Corps à corps"),
        (2, "Vehicle", "Véhicule"),
    ]

    for wid, name_en in WEAPON_INT_TO_NAME.items():
        canonical = WEAPON_FUSION_MAP.get(name_en, name_en)
        name_fr = WEAPON_NAME_FR.get(canonical, canonical)
        rows.append((wid, name_en, name_fr))

    conn.executemany(
        "INSERT OR IGNORE INTO weapon_labels (weapon_id, name_en, name_fr) VALUES (?, ?, ?)",
        rows,
    )
    logger.debug("weapon_labels : %d lignes insérées/ignorées", len(rows))


# ─────────────────────────────────────────────────────────────────────────────
# metadata.duckdb — Référentiel medal_definitions
# ─────────────────────────────────────────────────────────────────────────────

_MEDAL_DEFINITIONS_DDL = """
CREATE TABLE IF NOT EXISTS medal_definitions (
    medal_name_id  BIGINT PRIMARY KEY,
    name_fr        VARCHAR NOT NULL,
    name_en        VARCHAR NOT NULL,
    description_fr VARCHAR DEFAULT '',
    description_en VARCHAR DEFAULT '',
    is_custom      BOOLEAN DEFAULT FALSE
)
"""

_CUSTOM_MEDAL_THRESHOLD = 9_000_000_000


def ensure_medal_definitions_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table ``medal_definitions`` dans metadata.duckdb (idempotente).

    Seul le schéma est créé ici. La population depuis les JSON est assurée
    par ``scripts/populate_medal_metadata.py``.
    """
    conn.execute(_MEDAL_DEFINITIONS_DDL)
