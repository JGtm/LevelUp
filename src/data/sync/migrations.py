"""Migrations de schéma DuckDB centralisées.

Ce module regroupe toutes les fonctions de migration de colonnes
utilisées à la fois par engine.py (sync) et backfill_data.py.
Cela évite la duplication de code et garantit la cohérence du schéma.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

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
        logger.debug(f"Impossible de lire les colonnes de {table_name}: {e}")
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
            logger.info(f"Ajout de la colonne {column_name} à {table_name}")
            return True
        except Exception as e:
            logger.warning(f"Impossible d'ajouter {column_name} à {table_name}: {e}")
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
            "CREATE INDEX IF NOT EXISTS idx_mp_backfill "
            "ON match_participants(xuid, backfill_bits)"
        )
    except Exception as e:
        logger.debug(f"Index idx_mp_backfill ignoré: {e}")


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
        f"Migration highlight_events: ajout séquence auto-increment "
        f"(max_id={max_id}, {max_id_row} rows)"
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
            gamertag VARCHAR,
            type_hint INTEGER,
            raw_json VARCHAR
        )
    """)

    # 4) Restaurer les données
    if table_exists(conn, "highlight_events_backup"):
        conn.execute("INSERT INTO highlight_events SELECT * FROM highlight_events_backup")
        conn.execute("DROP TABLE highlight_events_backup")

    conn.execute("CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id)")
    logger.info("✅ highlight_events migrée avec séquence auto-increment " f"(start={max_id + 1})")


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
    logger.info(
        f"Migration career_progression: ajout séquence auto-increment " f"(max_id={max_id})"
    )
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
            recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)

    # 4) Restaurer les données
    if table_exists(conn, "career_progression_backup"):
        conn.execute("INSERT INTO career_progression SELECT * FROM career_progression_backup")
        conn.execute("DROP TABLE career_progression_backup")

    conn.execute("CREATE INDEX IF NOT EXISTS idx_career_xuid ON career_progression(xuid)")
    conn.execute("CREATE INDEX IF NOT EXISTS idx_career_date ON career_progression(recorded_at)")
    logger.info(f"✅ career_progression migrée avec séquence auto-increment (start={max_id + 1})")


# ─────────────────────────────────────────────────────────────────────────────
# Migration medals_earned (INT32 → BIGINT)
# ─────────────────────────────────────────────────────────────────────────────


# ─────────────────────────────────────────────────────────────────────────────
# Backfill bitmask : colonne match_stats.backfill_completed
# ─────────────────────────────────────────────────────────────────────────────

BACKFILL_FLAGS: dict[str, int] = {
    "medals": 1 << 0,  # 1
    "events": 1 << 1,  # 2
    "skill": 1 << 2,  # 4
    "personal_scores": 1 << 3,  # 8
    "performance_scores": 1 << 4,  # 16
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
        logger.warning(f"Migration medals_earned échouée (continuation): {e}")

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
            "SELECT database_name FROM duckdb_tables() " "WHERE table_name = 'match_registry'"
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
        logger.debug("match_registry non trouvée, vue mv_player_matches non créée")
        return

    # Seul "shared" (ATTACH) nécessite un prefix ; pour une connexion directe
    # au fichier (database_name = nom du fichier), pas de prefix.
    prefix = "shared." if catalog == "shared" else ""

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

            -- KDA pré-calculé
            CASE WHEN p.deaths > 0
            THEN (CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0)
                 / CAST(p.deaths AS FLOAT)
            ELSE CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0
            END AS kda,

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

            -- Scores d'équipe (depuis match_registry uniquement)
            CASE WHEN p.team_id = 0 THEN r.team_0_score
                 ELSE r.team_1_score END AS my_team_score,
            CASE WHEN p.team_id = 0 THEN r.team_1_score
                 ELSE r.team_0_score END AS enemy_team_score,

            -- MMR (enrichi depuis skill API)
            p.team_mmr,
            {enemy_mmr_expr},

            -- Score personnel
            p.score AS personal_score,

            -- Flags
            COALESCE(r.is_firefight, FALSE) AS is_firefight,
            COALESCE(r.is_ranked, FALSE) AS is_ranked

        FROM {prefix}match_registry r
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
            "SELECT database_name FROM duckdb_tables() " "WHERE table_name = 'match_participants'"
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
        f"CREATE INDEX IF NOT EXISTS idx_mr_start_time "
        f"ON {prefix}match_registry(start_time DESC)",
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
            logger.debug(f"Index {index_name} ignoré: {e}")
        else:
            logger.warning(f"Index {index_name} non créé: {e}")


# ─────────────────────────────────────────────────────────────────────────────
# Schéma PvE — shared_pve.duckdb (v5.2)
# ─────────────────────────────────────────────────────────────────────────────

PVE_SCHEMA_DDL = """
CREATE TABLE IF NOT EXISTS pve_match_stats (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,

    -- Stats globales PvE (à valider avec JSON réel en Phase 1)
    waves_completed    INTEGER,
    max_wave_reached   INTEGER,
    boss_kills         INTEGER,
    mythic_boss_kills  INTEGER,
    total_enemy_kills  INTEGER,

    -- Kills par type d'ennemi — Banished
    grunt_kills    INTEGER DEFAULT 0,
    elite_kills    INTEGER DEFAULT 0,
    jackal_kills   INTEGER DEFAULT 0,
    brute_kills    INTEGER DEFAULT 0,
    hunter_kills   INTEGER DEFAULT 0,
    skimmer_kills  INTEGER DEFAULT 0,

    -- Kills par type d'ennemi — Forerunners (si disponible dans l'API)
    crawler_kills  INTEGER DEFAULT 0,
    soldier_kills  INTEGER DEFAULT 0,
    knight_kills   INTEGER DEFAULT 0,
    warden_kills   INTEGER DEFAULT 0,

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
                        logger.warning(f"Erreur DDL PvE : {e}")
        logger.debug("Schéma PvE initialisé (shared_pve.duckdb)")
    except Exception as e:
        logger.error(f"Impossible d'initialiser le schéma PvE : {e}")
