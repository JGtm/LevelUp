"""Tests d'intégrité cross-DB et d'invariants métier (v6).

Vérifie les invariants architecturaux que DuckDB ne peut pas enforcer
via des contraintes (absence de FK cross-DB) mais qui sont des garanties
de la stack v6 :

Groupe 1 — Intégrité référentielle cross-DB
    Toute table satellite doit avoir ses match_id dans match_registry.

Groupe 2 — Cohérence flags ↔ données réelles
    participants_loaded, events_loaded, MatchBits.WEAPON_KILLS.

Groupe 3 — Cohérence PvE sémantique
    pve_match_stats uniquement pour des matchs is_firefight=TRUE.

Groupe 4 — Domaine de valeurs (enum guards)
    outcome, confidence, rating_type.

Groupe 5 — Invariants métier
    v_weapon_kills, weapon_id=0/confidence, performance_score, v_gamertag_lookup.
"""

from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path

import pytest

duckdb = pytest.importorskip("duckdb")

# ---------------------------------------------------------------------------
# Constantes métier (pas d'import src/ — évite les dépendances de prod)
# ---------------------------------------------------------------------------

_WEAPON_KILLS_BIT = 1 << 21  # MatchBits.WEAPON_KILLS
_NOW = datetime(2088, 3, 17, 12, 0, 0, tzinfo=timezone.utc)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def shared_db(tmp_path: Path) -> Path:
    """shared_matches_v2.duckdb minimale avec toutes les tables + vues v6."""
    db_path = tmp_path / "shared_matches_v2.duckdb"
    conn = duckdb.connect(str(db_path))
    try:
        conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP NOT NULL,
                is_firefight BOOLEAN DEFAULT FALSE,
                participants_loaded BOOLEAN DEFAULT FALSE,
                events_loaded BOOLEAN DEFAULT FALSE,
                medals_loaded BOOLEAN DEFAULT FALSE,
                backfill_completed INTEGER DEFAULT 0
            )
        """)
        conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                gamertag VARCHAR,
                outcome INTEGER,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        conn.execute("""
            CREATE TABLE medals_earned (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                medal_name_id BIGINT NOT NULL,
                count SMALLINT NOT NULL,
                PRIMARY KEY (match_id, xuid, medal_name_id)
            )
        """)
        conn.execute("""
            CREATE SEQUENCE highlight_events_id_seq;
            CREATE TABLE highlight_events (
                id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
                match_id VARCHAR NOT NULL,
                event_type VARCHAR NOT NULL
            )
        """)
        conn.execute("""
            CREATE TABLE killer_victim_pairs (
                match_id VARCHAR NOT NULL,
                killer_xuid VARCHAR NOT NULL,
                victim_xuid VARCHAR NOT NULL
            )
        """)
        conn.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                time_ms INTEGER NOT NULL,
                weapon_id UBIGINT,
                reconciled_as UBIGINT,
                confidence VARCHAR NOT NULL DEFAULT 'none'
            )
        """)
        conn.execute("""
            CREATE TABLE xuid_aliases (
                xuid VARCHAR PRIMARY KEY,
                gamertag VARCHAR NOT NULL
            )
        """)
        # Vues v6 garanties
        conn.execute("""
            CREATE VIEW v_gamertag_lookup AS
            SELECT
                COALESCE(xa.xuid, mp.xuid) AS xuid,
                COALESCE(xa.gamertag, mp.gamertag) AS gamertag
            FROM xuid_aliases xa
            FULL OUTER JOIN (
                SELECT DISTINCT xuid, gamertag FROM match_participants
            ) mp ON xa.xuid = mp.xuid
        """)
        conn.execute("""
            CREATE VIEW v_weapon_kills AS
            SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
            FROM weapon_kills
        """)

        # match1 : PvP — participants + events chargés + WEAPON_KILLS bitmask posé
        conn.execute(
            "INSERT INTO match_registry VALUES (?, ?, ?, ?, ?, ?, ?)",
            ["match1", _NOW, False, True, True, True, _WEAPON_KILLS_BIT],
        )
        # match2 : Firefight — participants chargés, pas d'events/weapon_kills
        conn.execute(
            "INSERT INTO match_registry VALUES (?, ?, ?, ?, ?, ?, ?)",
            ["match2", _NOW, True, True, False, False, 0],
        )

        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, ?, ?)",
            ["match1", "xuid_alice", "Alice", 2],  # WIN
        )
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, ?, ?)",
            ["match2", "xuid_bob", "Bob", 3],  # LOSS
        )

        conn.execute(
            "INSERT INTO medals_earned VALUES (?, ?, ?, ?)",
            ["match1", "xuid_alice", 100, 2],
        )
        conn.execute(
            "INSERT INTO medals_earned VALUES (?, ?, ?, ?)",
            ["match2", "xuid_bob", 200, 1],
        )

        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type) VALUES (?, ?)",
            ["match1", "kill"],
        )

        conn.execute(
            "INSERT INTO killer_victim_pairs VALUES (?, ?, ?)",
            ["match1", "xuid_alice", "xuid_bob"],
        )

        # weapon_kills : deux profils — weapon_id seul / reconciled_as seul
        conn.execute(
            "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?)",
            ["match1", "xuid_alice", 12000, 12345678, None, "fire_event"],
        )
        conn.execute(
            "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?)",
            ["match1", "xuid_alice", 13000, None, 99887766, "high"],
        )

        conn.execute("INSERT INTO xuid_aliases VALUES (?, ?)", ["xuid_alice", "Alice"])
        conn.execute("INSERT INTO xuid_aliases VALUES (?, ?)", ["xuid_charlie", "Charlie"])
    finally:
        conn.close()
    return db_path


@pytest.fixture
def player_db(tmp_path: Path) -> Path:
    """stats.duckdb joueur minimale (enrichissements + skill rank)."""
    db_path = tmp_path / "stats.duckdb"
    conn = duckdb.connect(str(db_path))
    try:
        conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                performance_score FLOAT,
                session_id VARCHAR,
                is_with_friends BOOLEAN
            )
        """)
        conn.execute("""
            CREATE TABLE match_skill_rank (
                match_id VARCHAR PRIMARY KEY,
                rating_type VARCHAR NOT NULL,
                rating_value FLOAT NOT NULL,
                tier VARCHAR,
                start_time TIMESTAMP
            )
        """)
        conn.execute(
            "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
            ["match1", 72.5, "s1", True],
        )
        conn.execute(
            "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
            ["match2", None, "s1", False],
        )
        conn.execute(
            "INSERT INTO match_skill_rank VALUES (?, ?, ?, ?, ?)",
            ["match1", "LUSR", 1540.0, "Gold", _NOW],
        )
    finally:
        conn.close()
    return db_path


@pytest.fixture
def pve_db(tmp_path: Path) -> Path:
    """shared_pve.duckdb minimale."""
    db_path = tmp_path / "shared_pve.duckdb"
    conn = duckdb.connect(str(db_path))
    try:
        conn.execute("""
            CREATE TABLE pve_match_stats (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                total_enemy_kills INTEGER,
                boss_kills INTEGER,
                pve_bits INTEGER DEFAULT 0,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        # match2 = Firefight (is_firefight=TRUE dans shared_db)
        conn.execute(
            "INSERT INTO pve_match_stats VALUES (?, ?, ?, ?, ?)",
            ["match2", "xuid_bob", 45, 3, 3],
        )
    finally:
        conn.close()
    return db_path


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _cross_db_conn(
    shared: Path,
    player: Path | None = None,
    pve: Path | None = None,
) -> duckdb.DuckDBPyConnection:
    """Connexion mémoire avec les DBs demandées attachées."""
    conn = duckdb.connect(":memory:")
    conn.execute(f"ATTACH '{shared.as_posix()}' AS shared (READ_ONLY)")
    if player is not None:
        conn.execute(f"ATTACH '{player.as_posix()}' AS player (READ_ONLY)")
    if pve is not None:
        conn.execute(f"ATTACH '{pve.as_posix()}' AS pve (READ_ONLY)")
    return conn


def _orphans(
    conn: duckdb.DuckDBPyConnection,
    satellite_table: str,
    registry_table: str = "shared.match_registry",
) -> list:
    """Retourne les match_id orphelins (présents dans satellite, absents du registre)."""
    return conn.execute(
        f"SELECT s.match_id FROM {satellite_table} s "
        f"LEFT JOIN {registry_table} r ON s.match_id = r.match_id "
        "WHERE r.match_id IS NULL"
    ).fetchall()


# ===========================================================================
# Groupe 1 — Intégrité référentielle cross-DB
# ===========================================================================


class TestReferentialIntegrity:
    """Tout match_id dans une table satellite doit exister dans match_registry."""

    def test_player_enrichment_no_orphan(self, shared_db: Path, player_db: Path) -> None:
        conn = _cross_db_conn(shared_db, player=player_db)
        try:
            orphans = _orphans(conn, "player.player_match_enrichment")
            assert not orphans, f"Enrichissements orphelins : {orphans}"
        finally:
            conn.close()

    def test_match_skill_rank_no_orphan(self, shared_db: Path, player_db: Path) -> None:
        conn = _cross_db_conn(shared_db, player=player_db)
        try:
            orphans = _orphans(conn, "player.match_skill_rank")
            assert not orphans, f"match_skill_rank orphelins : {orphans}"
        finally:
            conn.close()

    def test_medals_earned_no_orphan(self, shared_db: Path) -> None:
        conn = _cross_db_conn(shared_db)
        try:
            assert not _orphans(conn, "shared.medals_earned")
        finally:
            conn.close()

    def test_weapon_kills_no_orphan(self, shared_db: Path) -> None:
        conn = _cross_db_conn(shared_db)
        try:
            assert not _orphans(conn, "shared.weapon_kills")
        finally:
            conn.close()

    def test_killer_victim_pairs_no_orphan(self, shared_db: Path) -> None:
        conn = _cross_db_conn(shared_db)
        try:
            assert not _orphans(conn, "shared.killer_victim_pairs")
        finally:
            conn.close()

    def test_highlight_events_no_orphan(self, shared_db: Path) -> None:
        conn = _cross_db_conn(shared_db)
        try:
            assert not _orphans(conn, "shared.highlight_events")
        finally:
            conn.close()

    def test_pve_stats_no_orphan(self, shared_db: Path, pve_db: Path) -> None:
        conn = _cross_db_conn(shared_db, pve=pve_db)
        try:
            assert not _orphans(conn, "pve.pve_match_stats")
        finally:
            conn.close()

    # -- Vérification que la détection fonctionne sur des données corrompues --

    def test_orphan_detection_player_enrichment(self, shared_db: Path, tmp_path: Path) -> None:
        """Un enrichissement pointant vers un match inexistant est bien détecté."""
        bad_player = tmp_path / "bad_player.duckdb"
        c = duckdb.connect(str(bad_player))
        c.execute(
            "CREATE TABLE player_match_enrichment "
            "(match_id VARCHAR PRIMARY KEY, performance_score FLOAT)"
        )
        c.execute("INSERT INTO player_match_enrichment VALUES (?, ?)", ["ghost_match", 55.0])
        c.close()

        conn = _cross_db_conn(shared_db, player=bad_player)
        try:
            orphans = _orphans(conn, "player.player_match_enrichment")
            assert len(orphans) == 1
            assert orphans[0][0] == "ghost_match"
        finally:
            conn.close()

    def test_orphan_detection_weapon_kills(self, tmp_path: Path) -> None:
        """Un weapon_kill pointant vers un match inexistant est bien détecté."""
        db = tmp_path / "minimal_shared.duckdb"
        c = duckdb.connect(str(db))
        c.execute(
            "CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP NOT NULL)"
        )
        c.execute(
            "CREATE TABLE weapon_kills "
            "(match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL, time_ms INTEGER NOT NULL, "
            "weapon_id UBIGINT, reconciled_as UBIGINT, confidence VARCHAR NOT NULL DEFAULT 'none')"
        )
        c.execute("INSERT INTO match_registry VALUES (?, ?)", ["real_match", _NOW])
        c.execute(
            "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?)",
            ["real_match", "x1", 1000, 9998, None, "fire_event"],
        )
        c.execute(
            "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?)",
            ["ghost_match", "x1", 2000, 7777, None, "none"],
        )
        c.close()

        conn = _cross_db_conn(db)
        try:
            orphans = _orphans(conn, "shared.weapon_kills")
            assert len(orphans) == 1
            assert orphans[0][0] == "ghost_match"
        finally:
            conn.close()


# ===========================================================================
# Groupe 2 — Cohérence flags ↔ données réelles
# ===========================================================================


class TestFlagConsistency:
    """Un flag TRUE dans match_registry doit être corroboré par des lignes réelles."""

    def test_participants_loaded_implies_rows(self, shared_db: Path) -> None:
        """participants_loaded=TRUE → ≥ 1 ligne dans match_participants."""
        conn = _cross_db_conn(shared_db)
        try:
            violations = conn.execute("""
                SELECT mr.match_id
                FROM shared.match_registry mr
                WHERE mr.participants_loaded = TRUE
                  AND NOT EXISTS (
                      SELECT 1 FROM shared.match_participants mp
                      WHERE mp.match_id = mr.match_id
                  )
            """).fetchall()
            assert not violations, f"participants_loaded=TRUE sans participants : {violations}"
        finally:
            conn.close()

    def test_events_loaded_implies_rows(self, shared_db: Path) -> None:
        """events_loaded=TRUE → ≥ 1 ligne dans highlight_events."""
        conn = _cross_db_conn(shared_db)
        try:
            violations = conn.execute("""
                SELECT mr.match_id
                FROM shared.match_registry mr
                WHERE mr.events_loaded = TRUE
                  AND NOT EXISTS (
                      SELECT 1 FROM shared.highlight_events he
                      WHERE he.match_id = mr.match_id
                  )
            """).fetchall()
            assert not violations, f"events_loaded=TRUE sans events : {violations}"
        finally:
            conn.close()

    def test_weapon_kills_bitmask_implies_rows(self, shared_db: Path) -> None:
        """MatchBits.WEAPON_KILLS posé → ≥ 1 ligne dans weapon_kills."""
        conn = _cross_db_conn(shared_db)
        try:
            violations = conn.execute(f"""
                SELECT mr.match_id
                FROM shared.match_registry mr
                WHERE (COALESCE(mr.backfill_completed, 0) & {_WEAPON_KILLS_BIT}) != 0
                  AND NOT EXISTS (
                      SELECT 1 FROM shared.weapon_kills wk
                      WHERE wk.match_id = mr.match_id
                  )
            """).fetchall()
            assert not violations, f"WEAPON_KILLS bit posé mais aucun kill : {violations}"
        finally:
            conn.close()

    def test_flag_detection_participants_loaded_without_row(self, tmp_path: Path) -> None:
        """Détecte participants_loaded=TRUE sans ligne correspondante (flag mensonger)."""
        db = tmp_path / "lie.duckdb"
        c = duckdb.connect(str(db))
        c.execute(
            "CREATE TABLE match_registry "
            "(match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP NOT NULL, "
            "participants_loaded BOOLEAN DEFAULT FALSE)"
        )
        c.execute(
            "CREATE TABLE match_participants "
            "(match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL, PRIMARY KEY (match_id, xuid))"
        )
        c.execute("INSERT INTO match_registry VALUES (?, ?, ?)", ["liar_match", _NOW, True])
        # Aucune ligne dans match_participants
        c.close()

        conn = duckdb.connect(str(db), read_only=True)
        try:
            violations = conn.execute("""
                SELECT mr.match_id FROM match_registry mr
                WHERE mr.participants_loaded = TRUE
                  AND NOT EXISTS (
                      SELECT 1 FROM match_participants mp
                      WHERE mp.match_id = mr.match_id
                  )
            """).fetchall()
            assert len(violations) == 1
            assert violations[0][0] == "liar_match"
        finally:
            conn.close()


# ===========================================================================
# Groupe 3 — Cohérence PvE sémantique
# ===========================================================================


class TestPveSemantics:
    """Les stats PvE ne doivent exister que pour des matchs Firefight."""

    def test_pve_stats_only_for_firefight_matches(self, shared_db: Path, pve_db: Path) -> None:
        conn = _cross_db_conn(shared_db, pve=pve_db)
        try:
            violations = conn.execute("""
                SELECT p.match_id
                FROM pve.pve_match_stats p
                LEFT JOIN shared.match_registry mr ON p.match_id = mr.match_id
                WHERE mr.is_firefight IS NOT TRUE
            """).fetchall()
            assert not violations, f"Stats PvE sur match non-Firefight : {violations}"
        finally:
            conn.close()

    def test_pve_detection_on_pvp_match(self, shared_db: Path, tmp_path: Path) -> None:
        """Détecte des stats PvE insérées sur un match is_firefight=FALSE."""
        bad_pve = tmp_path / "bad_pve.duckdb"
        c = duckdb.connect(str(bad_pve))
        c.execute(
            "CREATE TABLE pve_match_stats "
            "(match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL, PRIMARY KEY (match_id, xuid))"
        )
        # match1 = is_firefight=FALSE dans shared_db
        c.execute("INSERT INTO pve_match_stats VALUES (?, ?)", ["match1", "xuid_alice"])
        c.close()

        conn = _cross_db_conn(shared_db, pve=bad_pve)
        try:
            violations = conn.execute("""
                SELECT p.match_id FROM pve.pve_match_stats p
                LEFT JOIN shared.match_registry mr ON p.match_id = mr.match_id
                WHERE mr.is_firefight IS NOT TRUE
            """).fetchall()
            assert len(violations) == 1
            assert violations[0][0] == "match1"
        finally:
            conn.close()


# ===========================================================================
# Groupe 4 — Domaine de valeurs (enum guards)
# ===========================================================================


class TestEnumDomains:
    """Colonnes à domaine fini : aucune valeur hors-spec."""

    def test_match_participants_outcome_valid(self, shared_db: Path) -> None:
        """outcome ∈ {1, 2, 3, 4} ou NULL (TIE/WIN/LOSS/DNF)."""
        conn = _cross_db_conn(shared_db)
        try:
            bad = conn.execute(
                "SELECT DISTINCT outcome FROM shared.match_participants "
                "WHERE outcome IS NOT NULL AND outcome NOT IN (1, 2, 3, 4)"
            ).fetchall()
            assert not bad, f"Valeurs outcome invalides : {[r[0] for r in bad]}"
        finally:
            conn.close()

    def test_weapon_kills_confidence_valid(self, shared_db: Path) -> None:
        """confidence ∈ {'none','low','medium','high','fire_event'}."""
        conn = _cross_db_conn(shared_db)
        try:
            bad = conn.execute(
                "SELECT DISTINCT confidence FROM shared.weapon_kills "
                "WHERE confidence NOT IN ('none', 'low', 'medium', 'high', 'fire_event')"
            ).fetchall()
            assert not bad, f"Valeurs confidence invalides : {[r[0] for r in bad]}"
        finally:
            conn.close()

    def test_match_skill_rank_rating_type_valid(self, player_db: Path) -> None:
        """rating_type ∈ {'LUSR', 'CSR'}."""
        conn = duckdb.connect(str(player_db), read_only=True)
        try:
            bad = conn.execute(
                "SELECT DISTINCT rating_type FROM match_skill_rank "
                "WHERE rating_type NOT IN ('LUSR', 'CSR')"
            ).fetchall()
            assert not bad, f"rating_type invalides : {[r[0] for r in bad]}"
        finally:
            conn.close()

    def test_enum_detection_bad_outcome(self, tmp_path: Path) -> None:
        """Détecte une valeur outcome=99 hors-spec."""
        db = tmp_path / "bad.duckdb"
        c = duckdb.connect(str(db))
        c.execute(
            "CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, outcome INTEGER)"
        )
        c.execute("INSERT INTO match_participants VALUES (?, ?, ?)", ["m1", "x1", 99])
        c.close()

        conn = duckdb.connect(str(db), read_only=True)
        try:
            bad = conn.execute(
                "SELECT outcome FROM match_participants WHERE outcome NOT IN (1, 2, 3, 4)"
            ).fetchall()
            assert len(bad) == 1 and bad[0][0] == 99
        finally:
            conn.close()


# ===========================================================================
# Groupe 5 — Invariants métier
# ===========================================================================


class TestBusinessInvariants:
    """Invariants de calcul et de vue qui peuvent être silencieusement violés."""

    def test_v_weapon_kills_effective_weapon_id_never_null(self, shared_db: Path) -> None:
        """COALESCE(reconciled_as, weapon_id) ne doit pas être NULL quand l'un des
        deux champs est renseigné — sinon la vue retourne un label 'Inconnu' silencieux."""
        conn = _cross_db_conn(shared_db)
        try:
            bad = conn.execute(
                "SELECT match_id, xuid, time_ms FROM shared.v_weapon_kills "
                "WHERE effective_weapon_id IS NULL "
                "  AND (weapon_id IS NOT NULL OR reconciled_as IS NOT NULL)"
            ).fetchall()
            assert (
                not bad
            ), f"effective_weapon_id NULL malgré weapon_id/reconciled_as renseigné : {bad}"
        finally:
            conn.close()

    def test_weapon_id_zero_not_high_confidence(self, shared_db: Path) -> None:
        """weapon_id=0 avec confidence='high' est une anomalie (cf. INV-113).
        weapon_id=0 = GRENADE_WEAPON_ID ; confiance 'high' n'est jamais légitime ici."""
        conn = _cross_db_conn(shared_db)
        try:
            anomalies = conn.execute(
                "SELECT match_id, xuid, time_ms FROM shared.weapon_kills "
                "WHERE weapon_id = 0 AND confidence = 'high'"
            ).fetchall()
            assert not anomalies, f"weapon_id=0 + confidence=high (anomalie INV-113) : {anomalies}"
        finally:
            conn.close()

    def test_performance_score_non_negative(self, player_db: Path) -> None:
        """performance_score doit être NULL ou ≥ 0 — un score négatif = overflow silencieux."""
        conn = duckdb.connect(str(player_db), read_only=True)
        try:
            bad = conn.execute(
                "SELECT match_id, performance_score FROM player_match_enrichment "
                "WHERE performance_score IS NOT NULL AND performance_score < 0"
            ).fetchall()
            assert not bad, f"performance_score négatif : {bad}"
        finally:
            conn.close()

    def test_v_gamertag_lookup_covers_all_known_xuids(self, shared_db: Path) -> None:
        """v_gamertag_lookup (FULL OUTER JOIN) doit couvrir TOUS les XUIDs de
        xuid_aliases ET match_participants — aucun XUID connu ne peut disparaître."""
        conn = _cross_db_conn(shared_db)
        try:
            all_known = {
                row[0]
                for row in conn.execute(
                    "SELECT xuid FROM shared.xuid_aliases "
                    "UNION "
                    "SELECT DISTINCT xuid FROM shared.match_participants"
                ).fetchall()
            }
            in_view = {
                row[0]
                for row in conn.execute(
                    "SELECT xuid FROM shared.v_gamertag_lookup WHERE xuid IS NOT NULL"
                ).fetchall()
            }
            missing = all_known - in_view
            assert not missing, f"XUIDs absents de v_gamertag_lookup : {missing}"
        finally:
            conn.close()

    def test_performance_score_detection_negative(self, tmp_path: Path) -> None:
        """Détecte un performance_score négatif inséré par erreur."""
        db = tmp_path / "bad_player.duckdb"
        c = duckdb.connect(str(db))
        c.execute(
            "CREATE TABLE player_match_enrichment "
            "(match_id VARCHAR PRIMARY KEY, performance_score FLOAT)"
        )
        c.execute("INSERT INTO player_match_enrichment VALUES (?, ?)", ["m1", -5.0])
        c.close()

        conn = duckdb.connect(str(db), read_only=True)
        try:
            bad = conn.execute(
                "SELECT match_id FROM player_match_enrichment "
                "WHERE performance_score IS NOT NULL AND performance_score < 0"
            ).fetchall()
            assert len(bad) == 1
            assert bad[0][0] == "m1"
        finally:
            conn.close()
