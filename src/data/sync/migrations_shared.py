"""Migrations de schéma DuckDB — DB partagée (shared_matches_v2.duckdb).

Ce module regroupe les fonctions de migration relatives à la base de données
partagée : match_participants, highlight_events, medals_earned, match_registry,
weapon_kills, vues de résolution (v_gamertag_lookup, v_match_full, etc.).

Extrait de migrations.py lors du housekeeping pré-v7 (H3).
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

from src.data.sync._migrations_utils import (
    _add_column_if_missing,
    get_table_columns,
    table_exists,
)

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)


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
# Correction bot XIDs (bug legacy migrate_sqlite) — shared_matches_v2.duckdb
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
# Scores PS (personal score sums) — shared_matches_v2.duckdb
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
# Schéma weapon_kills — shared_matches_v2.duckdb (v5.7, per-kill avec weapon_id UBIGINT)
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
    À appeler sur la connexion ``shared_matches_v2.duckdb``.
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
        logger.debug("Table weapon_kills initialisée (shared_matches_v2.duckdb)")
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
    """Crée la vue v_gamertag_lookup dans shared_matches_v2.duckdb.

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
    """Crée la vue v_match_full dans shared_matches_v2.duckdb.

    Résout les noms d'assets depuis metadata.duckdb (via meta_alias) si disponible.
    Priorité pour les noms : asset_translations (14 langues) > tables legacy (name_en/name_fr) > match_registry.
    Si meta_alias est None, les colonnes *_fr et mode_* sont NULL,
    et les colonnes EN tombent en fallback sur match_registry (comportement actuel).
    """
    if meta_alias:
        map_en = "COALESCE(at_map_en.name, mr.map_name)"
        pl_en = "COALESCE(at_pl_en.name, mr.playlist_name)"
        pp_en = "COALESCE(at_pair_en.name, mr.pair_name)"
        gv_en = "COALESCE(at_gv_en.name, mr.game_variant_name)"
        joins = f"""
        LEFT JOIN {meta_alias}.asset_translations at_map_en
            ON mr.map_id = at_map_en.asset_id AND at_map_en.asset_type = 'map' AND at_map_en.lang = 'en-US'
        LEFT JOIN {meta_alias}.asset_translations at_map_fr
            ON mr.map_id = at_map_fr.asset_id AND at_map_fr.asset_type = 'map' AND at_map_fr.lang = 'fr-FR'
        LEFT JOIN {meta_alias}.asset_translations at_pl_en
            ON mr.playlist_id = at_pl_en.asset_id AND at_pl_en.asset_type = 'playlist' AND at_pl_en.lang = 'en-US'
        LEFT JOIN {meta_alias}.asset_translations at_pl_fr
            ON mr.playlist_id = at_pl_fr.asset_id AND at_pl_fr.asset_type = 'playlist' AND at_pl_fr.lang = 'fr-FR'
        LEFT JOIN {meta_alias}.asset_translations at_pair_en
            ON mr.pair_id = at_pair_en.asset_id AND at_pair_en.asset_type = 'pair' AND at_pair_en.lang = 'en-US'
        LEFT JOIN {meta_alias}.asset_translations at_pair_fr
            ON mr.pair_id = at_pair_fr.asset_id AND at_pair_fr.asset_type = 'pair' AND at_pair_fr.lang = 'fr-FR'
        LEFT JOIN {meta_alias}.asset_translations at_gv_en
            ON mr.game_variant_id = at_gv_en.asset_id AND at_gv_en.asset_type = 'game_variant' AND at_gv_en.lang = 'en-US'
        LEFT JOIN {meta_alias}.asset_translations at_gv_fr
            ON mr.game_variant_id = at_gv_fr.asset_id AND at_gv_fr.asset_type = 'game_variant' AND at_gv_fr.lang = 'fr-FR'"""
        # Fallback par nom EN : certains assets ont plusieurs UUIDs (ex: deux versions de
        # "Quick Play"), dont certains sans traduction FR directe. Si la jointure par
        # playlist_id ne trouve pas de FR, on cherche par le nom EN dans asset_translations.
        fr_cols = f"""
        at_map_fr.name                               AS map_name_fr,
        COALESCE(
            at_pl_fr.name,
            (SELECT fb.name FROM {meta_alias}.asset_translations fb
             INNER JOIN {meta_alias}.asset_translations fb_en
                 ON fb.asset_id = fb_en.asset_id
                 AND fb_en.asset_type = 'playlist' AND fb_en.lang = 'en-US'
             WHERE fb.asset_type = 'playlist' AND fb.lang = 'fr-FR'
             AND fb_en.name = COALESCE(at_pl_en.name, mr.playlist_name)
             LIMIT 1)
        )                                            AS playlist_name_fr,
        at_pair_fr.name                              AS pair_name_fr,
        at_gv_fr.name                                AS game_variant_name_fr,
        NULL                                         AS mode_name,
        NULL                                         AS mode_name_fr,
        NULL                                         AS playlist_canonical_en,
        NULL                                         AS playlist_canonical_fr"""
    else:
        logger.warning("metadata.duckdb non attachée — v_match_full créée sans traductions FR")
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
    """Crée la vue v_killer_victim_full dans shared_matches_v2.duckdb.

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
    """Crée ou met à jour les 3 vues de résolution d'IDs dans shared_matches_v2.duckdb.

    Vues créées (idempotentes via CREATE OR REPLACE VIEW) :
    - v_gamertag_lookup : XUID → gamertag courant
    - v_match_full      : match_registry + noms résolus depuis metadata.duckdb
    - v_killer_victim_full : killer_victim_pairs + gamertags résolus

    Précondition : la connexion doit pointer sur shared_matches_v2.duckdb
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

    Retourne l'alias si réussi ET que la table asset_translations existe (v6), None sinon.
    """
    from src.utils.db import ensure_metadata_attached

    alias = None
    try:
        alias = ensure_metadata_attached(conn)
    except Exception as e:
        logger.debug("event=meta_attach_failed step=try_attach_meta_for_views error=%s", e)

    if alias is None:
        return None

    # Vérifier que asset_translations est présente (source i18n v6)
    try:
        db_name_rows = conn.execute(
            "SELECT database_name FROM duckdb_tables() WHERE table_name = 'asset_translations'"
        ).fetchall()
        if any(r[0] == alias for r in db_name_rows):
            return alias
    except Exception as e:
        logger.debug(
            "event=meta_asset_translations_check_failed step=try_attach_meta_for_views error=%s", e
        )
    return None


# ─────────────────────────────────────────────────────────────────────────────
# metadata.duckdb — Référentiel weapon_labels
# ─────────────────────────────────────────────────────────────────────────────


def ensure_match_registry_playable_duration(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute playable_duration_seconds et real_start_time à match_registry.

    - playable_duration_seconds : durée réelle du gameplay (sans countdown/lobby),
      extraite de MatchInfo.PlayableDuration (API SPNKr).
    - real_start_time : heure UTC du début effectif du gameplay, calculée comme
      start_time + (duration_seconds - playable_duration_seconds).

    Ces colonnes sont NULL pour les matchs syncés avant cette migration.
    Un backfill API (--playable-duration) est nécessaire pour les remplir
    rétroactivement.

    Idempotente via _add_column_if_missing().
    """
    if not table_exists(conn, "match_registry"):
        return
    _add_column_if_missing(conn, "match_registry", "playable_duration_seconds", "INTEGER")
    _add_column_if_missing(conn, "match_registry", "real_start_time", "TIMESTAMP")


def ensure_match_registry_film_start(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute film_match_start_ms à match_registry (shared).

    Colonne INTEGER nullable stockant le timestamp filmshell (en ms depuis le
    début de l'enregistrement) correspondant au premier mouvement réel des
    joueurs — c'est-à-dire la fin du countdown, le vrai t=0 du match.

    NULL pour les matchs dont les chunks filmshell n'ont pas encore été analysés.
    Le backfill est effectué via scripts/_exp_spawn_download.py.

    Idempotente via _add_column_if_missing().
    """
    if not table_exists(conn, "match_registry"):
        return
    _add_column_if_missing(conn, "match_registry", "film_match_start_ms", "INTEGER")


def fix_events_loaded_inconsistency(conn: duckdb.DuckDBPyConnection) -> None:
    """Corrige les matchs avec events_loaded=TRUE mais sans entrée dans highlight_events.

    Root cause : la migration add_highlight_events_autoincrement (2026-03-07) a recréé
    la table highlight_events, perdant tous les events antérieurs mais laissant
    events_loaded=TRUE dans match_registry.

    Remet events_loaded=FALSE pour ces matchs afin que :
    - les matchs récents soient retentés au prochain delta sync (via pending_events_ids)
    - le backfill --events puisse les cibler
    Idempotente.
    """
    if not table_exists(conn, "match_registry") or not table_exists(conn, "highlight_events"):
        return
    conn.execute(
        """UPDATE match_registry
           SET events_loaded = FALSE
           WHERE events_loaded = TRUE
             AND match_id NOT IN (SELECT DISTINCT match_id FROM highlight_events)"""
    )


# ─────────────────────────────────────────────────────────────────────────────
# Sessions — player (stats.duckdb)
# ─────────────────────────────────────────────────────────────────────────────
