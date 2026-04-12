"""Migrations de schéma DuckDB — DB joueur (stats.duckdb).

Ce module regroupe les fonctions de migration relatives à la base de données
par joueur : match_stats, career_progression, vues matérialisées, indexes,
skill_history, match_skill_rank, player_match_enrichment.

Extrait de migrations.py lors du housekeeping pré-v7 (H3).
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

from src.data._score_sql import NORM_ENEMY_TEAM_SCORE_SQL, NORM_MY_TEAM_SCORE_SQL
from src.data.sync._migrations_utils import (
    _add_column_if_missing,
    _create_index_safe,
    get_table_columns,
    table_exists,
)

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)


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


def ensure_mv_player_matches_view(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ou met à jour la vue mv_player_matches dans shared_matches_v2.duckdb.

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
    has_v_match_full = False
    try:
        conn.execute(f"SELECT 1 FROM {prefix}v_match_full LIMIT 1")
        match_source = f"{prefix}v_match_full"
        has_v_match_full = True
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

    # Colonnes de traduction FR (disponibles seulement via v_match_full — v6)
    if has_v_match_full:
        fr_cols_expr = "r.map_name_fr,\n            r.playlist_name_fr,\n            r.pair_name_fr,\n            r.game_variant_name_fr,"
    else:
        logger.warning("v_match_full absente — mv_player_matches créée sans colonnes FR")
        fr_cols_expr = "NULL AS map_name_fr,\n            NULL AS playlist_name_fr,\n            NULL AS pair_name_fr,\n            NULL AS game_variant_name_fr,"

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
            -- Traductions FR (depuis v_match_full via asset_translations)
            {fr_cols_expr}
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

    # Colonnes ajoutées après la création initiale de la table — migration idempotente.
    # CREATE TABLE IF NOT EXISTS ne suffit pas si la table existe déjà sans ces colonnes.
    _MISSING_COLS = [
        ("start_time", "TIMESTAMP"),
        ("rating_deviation", "FLOAT"),
        ("playlist_group", "VARCHAR"),
    ]
    for col_name, col_type in _MISSING_COLS:
        try:
            _add_column_if_missing(conn, "match_skill_rank", col_name, col_type)
        except Exception as e:
            logger.warning("ensure_match_skill_rank_table: migration colonne %s: %s", col_name, e)


# ─────────────────────────────────────────────────────────────────────────────
# Migration match_registry : colonne sync_spnkr_version (v5.4)
# ─────────────────────────────────────────────────────────────────────────────


def ensure_pme_session_index(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée l'index sur player_match_enrichment(session_id) s'il n'existe pas.

    Utilisé dans les GROUP BY et filter de mv_session_stats.
    Idempotente : CREATE INDEX IF NOT EXISTS.
    """
    if not table_exists(conn, "player_match_enrichment"):
        logger.debug("player_match_enrichment absente, index session ignoré")
        return
    _create_index_safe(
        conn,
        "CREATE INDEX IF NOT EXISTS idx_pme_session ON player_match_enrichment(session_id)",
        "idx_pme_session",
    )


def ensure_media_discord_notified_column(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute la colonne discord_notified_at à media_files si elle est absente.

    Permet de tracer quels médias ont déjà fait l'objet d'une notification
    Discord, pour éviter le spam lors des re-scans.
    """
    if not table_exists(conn, "media_files"):
        return
    _add_column_if_missing(conn, "media_files", "discord_notified_at", "TIMESTAMP")


_CHALLENGE_SNAPSHOTS_DDL = """
CREATE TABLE IF NOT EXISTS challenge_snapshots (
    snapshot_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    xuid              VARCHAR NOT NULL,
    challenge_path    VARCHAR NOT NULL,
    challenge_id      VARCHAR,
    content_hash      VARCHAR,
    status            VARCHAR NOT NULL,
    progress_current  INTEGER,
    progress_target   INTEGER,
    xp_reward         INTEGER DEFAULT 0,
    can_reroll        BOOLEAN,
    expires_at        TIMESTAMP,
    deck_index        INTEGER,
    state_hash        VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_xuid_time
    ON challenge_snapshots(xuid, snapshot_at DESC);
CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_path_time
    ON challenge_snapshots(challenge_path, snapshot_at DESC);
"""


def ensure_challenge_snapshots_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``challenge_snapshots`` dans stats.duckdb (idempotente).

    Historise les états des défis d'un joueur en mode append-only dédupliqué.
    """
    conn.execute(_CHALLENGE_SNAPSHOTS_DDL)


def ensure_mv_session_stats_varchar(conn: duckdb.DuckDBPyConnection) -> None:
    """Migre mv_session_stats.session_id de INTEGER vers VARCHAR si nécessaire.

    La table était créée avec session_id INTEGER PRIMARY KEY mais reçoit du
    VARCHAR depuis player_match_enrichment. Cette migration recrée la table
    avec le bon type (la table est de toute façon reconstruite à chaque sync).
    Idempotente : vérifie le type avant d'agir.
    """
    if not table_exists(conn, "mv_session_stats"):
        return
    row = conn.execute(
        "SELECT data_type FROM information_schema.columns "
        "WHERE table_name = 'mv_session_stats' AND column_name = 'session_id'"
    ).fetchone()
    if not row or row[0].upper() in ("VARCHAR", "TEXT"):
        return  # déjà correct

    logger.info("mv_session_stats.session_id est %s → migration vers VARCHAR", row[0])
    conn.execute("DROP TABLE mv_session_stats")
    conn.execute(
        """CREATE TABLE mv_session_stats (
            session_id VARCHAR PRIMARY KEY, match_count INTEGER,
            start_time TIMESTAMP, end_time TIMESTAMP,
            total_kills INTEGER, total_deaths INTEGER, total_assists INTEGER,
            kd_ratio DOUBLE, win_rate DOUBLE,
            avg_accuracy DOUBLE, avg_life_seconds DOUBLE, updated_at TIMESTAMP)"""
    )
