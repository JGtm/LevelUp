"""Fixtures réutilisables pour les tests sync/fan-out/shared writes.

Fournit des DBs DuckDB in-memory ou tmpdir avec schéma V6 complet,
des données de test réalistes, et un mock API SPNKr minimal.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any
from unittest.mock import AsyncMock

import duckdb

# ---------------------------------------------------------------------------
# Constantes de test
# ---------------------------------------------------------------------------
XUID_PLAYER_A = "2535400000001"
XUID_PLAYER_B = "2535400000002"
XUID_PLAYER_C = "2535400000003"
XUID_OPPONENT = "2535400000099"

GT_PLAYER_A = "PlayerA"
GT_PLAYER_B = "PlayerB"
GT_PLAYER_C = "PlayerC"
GT_OPPONENT = "OpponentX"

BASE_TIME = datetime(2024, 6, 1, 12, 0, 0, tzinfo=timezone.utc)

# ---------------------------------------------------------------------------
# Schéma DDL partagé (shared_matches)
# ---------------------------------------------------------------------------
SHARED_SCHEMA = """
CREATE TABLE IF NOT EXISTS match_registry (
    match_id VARCHAR PRIMARY KEY,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    playlist_id VARCHAR, playlist_name VARCHAR,
    map_id VARCHAR, map_name VARCHAR,
    pair_id VARCHAR, pair_name VARCHAR,
    game_variant_id VARCHAR, game_variant_name VARCHAR,
    mode_category VARCHAR,
    is_ranked BOOLEAN DEFAULT FALSE,
    is_firefight BOOLEAN DEFAULT FALSE,
    duration_seconds INTEGER,
    playable_duration_seconds INTEGER,
    real_start_time TIMESTAMP,
    team_0_score SMALLINT, team_1_score SMALLINT,
    team_0_ps_score INTEGER, team_1_ps_score INTEGER,
    backfill_completed INTEGER DEFAULT 0,
    participants_loaded BOOLEAN DEFAULT FALSE,
    events_loaded BOOLEAN DEFAULT FALSE,
    medals_loaded BOOLEAN DEFAULT FALSE,
    first_sync_by VARCHAR, first_sync_at TIMESTAMP,
    last_updated_at TIMESTAMP, player_count SMALLINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS match_participants (
    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
    gamertag VARCHAR, team_id INTEGER, outcome INTEGER,
    rank SMALLINT, score INTEGER,
    kills SMALLINT, deaths SMALLINT, assists SMALLINT,
    shots_fired INTEGER, shots_hit INTEGER,
    damage_dealt FLOAT, damage_taken FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    kda FLOAT, accuracy FLOAT, time_played_seconds INTEGER,
    avg_life_seconds FLOAT, personal_score INTEGER,
    team_mmr FLOAT, enemy_mmr FLOAT,
    kills_expected FLOAT, deaths_expected FLOAT,
    kills_stddev FLOAT, deaths_stddev FLOAT,
    assists_expected FLOAT, assists_stddev FLOAT,
    grenade_kills SMALLINT DEFAULT 0,
    melee_kills SMALLINT DEFAULT 0,
    power_weapon_kills SMALLINT DEFAULT 0,
    headshot_kills SMALLINT DEFAULT 0,
    max_killing_spree SMALLINT DEFAULT 0,
    backfill_bits INTEGER DEFAULT 0,
    PRIMARY KEY (match_id, xuid)
);
CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
CREATE TABLE IF NOT EXISTS highlight_events (
    id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
    match_id VARCHAR NOT NULL, event_type VARCHAR NOT NULL,
    time_ms INTEGER,
    xuid VARCHAR,
    type_hint INTEGER, raw_json VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS killer_victim_pairs (
    match_id VARCHAR NOT NULL,
    killer_xuid VARCHAR NOT NULL, killer_gamertag VARCHAR,
    victim_xuid VARCHAR NOT NULL, victim_gamertag VARCHAR,
    kill_count INTEGER DEFAULT 1, time_ms INTEGER,
    is_validated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS medals_earned (
    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
    medal_name_id BIGINT NOT NULL, count SMALLINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, xuid, medal_name_id)
);
CREATE TABLE IF NOT EXISTS xuid_aliases (
    xuid VARCHAR PRIMARY KEY, gamertag VARCHAR NOT NULL,
    last_seen TIMESTAMP, source VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS weapon_kills (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    weapon_id UBIGINT NOT NULL,
    kills SMALLINT DEFAULT 0,
    headshot_kills SMALLINT DEFAULT 0,
    damage_dealt FLOAT DEFAULT 0,
    confidence VARCHAR DEFAULT 'high',
    delta_ms INTEGER,
    path VARCHAR,
    reconciled_as UBIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, xuid, weapon_id)
);
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    description VARCHAR NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""

# ---------------------------------------------------------------------------
# Schéma DDL joueur (stats.duckdb)
# ---------------------------------------------------------------------------
PLAYER_SCHEMA = """
CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq;
CREATE TABLE IF NOT EXISTS personal_score_awards (
    id INTEGER PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    award_name VARCHAR NOT NULL,
    award_category VARCHAR,
    award_count INTEGER DEFAULT 1,
    award_score INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_psa_match ON personal_score_awards(match_id);

CREATE TABLE IF NOT EXISTS player_match_enrichment (
    match_id VARCHAR PRIMARY KEY,
    performance_score FLOAT,
    session_id VARCHAR,
    session_label VARCHAR,
    is_with_friends BOOLEAN,
    teammates_signature VARCHAR,
    known_teammates_count SMALLINT,
    friends_xuids VARCHAR,
    had_bot_teammate BOOLEAN,
    dominance_flag SMALLINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS match_citations (
    match_id VARCHAR NOT NULL,
    citation_key VARCHAR NOT NULL,
    citation_label VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, citation_key)
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id VARCHAR PRIMARY KEY,
    session_label VARCHAR,
    first_match_time TIMESTAMP,
    last_match_time TIMESTAMP,
    match_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS match_skill_rank (
    match_id VARCHAR PRIMARY KEY,
    playlist_id VARCHAR,
    csr_value INTEGER,
    csr_tier VARCHAR,
    csr_sub_tier INTEGER,
    csr_initial_value INTEGER,
    csr_post_value INTEGER,
    lusr_mu FLOAT,
    lusr_sigma FLOAT,
    lusr_rating FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE IF NOT EXISTS career_progression_id_seq;
CREATE TABLE IF NOT EXISTS career_progression (
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
);
"""


# ---------------------------------------------------------------------------
# Data builders
# ---------------------------------------------------------------------------
@dataclass
class MatchData:
    """Données pour un match de test."""

    match_id: str
    start_time: datetime
    participants: list[dict[str, Any]] = field(default_factory=list)
    events: list[dict[str, Any]] = field(default_factory=list)
    medals: list[dict[str, Any]] = field(default_factory=list)
    duration_seconds: int = 600
    team_0_score: int = 50
    team_1_score: int = 45
    is_ranked: bool = False
    is_firefight: bool = False


def make_participant(  # noqa: PLR0913
    match_id: str,
    xuid: str,
    gamertag: str,
    *,
    team_id: int = 0,
    outcome: int = 2,
    kills: int = 10,
    deaths: int = 8,
    assists: int = 5,
    personal_score: int = 1500,
) -> dict[str, Any]:
    """Crée un dict participant réaliste."""
    return {
        "match_id": match_id,
        "xuid": xuid,
        "gamertag": gamertag,
        "team_id": team_id,
        "outcome": outcome,
        "kills": kills,
        "deaths": deaths,
        "assists": assists,
        "kda": round((kills + assists / 3) / max(deaths, 1), 2),
        "accuracy": 0.45,
        "time_played_seconds": 600,
        "avg_life_seconds": 45.0,
        "personal_score": personal_score,
        "damage_dealt": 3000.0,
        "damage_taken": 2800.0,
        "rank": 1,
        "score": personal_score,
        "shots_fired": 200,
        "shots_hit": 90,
        "team_mmr": 1500.0,
        "enemy_mmr": 1480.0,
        "kills_expected": 9.5,
        "deaths_expected": 8.5,
        "kills_stddev": 2.0,
        "deaths_stddev": 2.0,
        "assists_expected": 5.0,
        "assists_stddev": 1.5,
        "grenade_kills": 0,
        "melee_kills": 0,
        "power_weapon_kills": 0,
        "headshot_kills": 3,
        "max_killing_spree": 5,
    }


def make_match_data(
    match_id: str,
    participants_xuids: list[tuple[str, str]],
    *,
    start_time: datetime | None = None,
    kills_a: int = 10,
) -> MatchData:
    """Crée un MatchData complet.

    Args:
        match_id: Identifiant du match.
        participants_xuids: Liste de (xuid, gamertag) pour chaque participant.
        start_time: Heure de début (défaut BASE_TIME).
        kills_a: Kills du premier participant (pour varier les stats).
    """
    t = start_time or BASE_TIME
    parts = []
    for i, (xuid, gt) in enumerate(participants_xuids):
        k = kills_a if i == 0 else 8
        parts.append(make_participant(match_id, xuid, gt, kills=k, team_id=i % 2))
    return MatchData(match_id=match_id, start_time=t, participants=parts)


def generate_n_matches(
    n: int,
    players: list[tuple[str, str]],
    *,
    base_time: datetime = BASE_TIME,
) -> list[MatchData]:
    """Génère n matchs avec les joueurs donnés."""
    return [
        make_match_data(
            f"match-{i:04d}",
            players,
            start_time=base_time + timedelta(hours=i),
            kills_a=10 + (i % 5),
        )
        for i in range(n)
    ]


# ---------------------------------------------------------------------------
# DB builders
# ---------------------------------------------------------------------------
def create_shared_db(path: Path, matches: list[MatchData] | None = None) -> Path:
    """Crée une shared_matches DB avec schéma complet, vues v6, et données optionnelles."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    try:
        for stmt in SHARED_SCHEMA.split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)
        # Vues v6 garanties (comme en prod)
        for stmt in V6_SHARED_VIEWS.split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)
        if matches:
            _insert_matches_to_shared(conn, matches)
        conn.commit()
    finally:
        conn.close()
    return path


def create_player_db(path: Path) -> Path:
    """Crée une player DB avec schéma complet."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    try:
        for stmt in PLAYER_SCHEMA.split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)
        conn.commit()
    finally:
        conn.close()
    return path


def _insert_matches_to_shared(conn: duckdb.DuckDBPyConnection, matches: list[MatchData]) -> None:
    """Insère les matchs dans les tables shared."""
    for m in matches:
        conn.execute(
            "INSERT OR IGNORE INTO match_registry "
            "(match_id, start_time, duration_seconds, team_0_score, team_1_score, "
            "is_ranked, is_firefight, participants_loaded, events_loaded, medals_loaded) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, TRUE, FALSE, FALSE)",
            (
                m.match_id,
                m.start_time,
                m.duration_seconds,
                m.team_0_score,
                m.team_1_score,
                m.is_ranked,
                m.is_firefight,
            ),
        )
        for p in m.participants:
            cols = [
                "match_id",
                "xuid",
                "gamertag",
                "team_id",
                "outcome",
                "kills",
                "deaths",
                "assists",
                "kda",
                "accuracy",
                "time_played_seconds",
                "avg_life_seconds",
                "personal_score",
                "damage_dealt",
                "damage_taken",
                "rank",
                "score",
                "shots_fired",
                "shots_hit",
                "team_mmr",
                "enemy_mmr",
                "kills_expected",
                "deaths_expected",
                "kills_stddev",
                "deaths_stddev",
                "assists_expected",
                "assists_stddev",
            ]
            placeholders = ", ".join(["?" for _ in cols])
            vals = [p.get(c) for c in cols]
            conn.execute(
                f"INSERT OR IGNORE INTO match_participants ({', '.join(cols)}) "
                f"VALUES ({placeholders})",
                vals,
            )


def insert_pme_rows(
    conn: duckdb.DuckDBPyConnection,
    match_ids: list[str],
    *,
    performance_score: float | None = None,
    session_id: str | None = None,
) -> None:
    """Insère des lignes player_match_enrichment."""
    for mid in match_ids:
        conn.execute(
            "INSERT OR IGNORE INTO player_match_enrichment "
            "(match_id, performance_score, session_id) VALUES (?, ?, ?)",
            (mid, performance_score, session_id),
        )


# ---------------------------------------------------------------------------
# db_profiles.json builder
# ---------------------------------------------------------------------------
def create_db_profiles(
    project_root: Path,
    players: list[dict[str, Any]],
) -> Path:
    """Crée db_profiles.json avec les joueurs donnés.

    Args:
        project_root: Racine du projet (parent de data/).
        players: Liste de {"gamertag": str, "xuid": str, "db_path": str}.
    """
    profiles = {}
    for p in players:
        profiles[p["gamertag"]] = {
            "xuid": p["xuid"],
            "db_path": p["db_path"],
        }
    content = {"profiles": profiles}
    path = project_root / "db_profiles.json"
    path.write_text(json.dumps(content, indent=2), encoding="utf-8")
    return path


# ---------------------------------------------------------------------------
# Mock API helpers
# ---------------------------------------------------------------------------
def create_mock_api_client(
    match_history: list[Any] | None = None,
    match_stats: dict[str, Any] | None = None,
) -> AsyncMock:
    """Crée un mock SPNKrAPIClient minimal.

    Args:
        match_history: Résultat de get_match_history().
        match_stats: Dict match_id → stats JSON.
    """
    client = AsyncMock()
    client.get_match_history = AsyncMock(return_value=match_history or [])

    async def _get_stats(match_id: str) -> dict[str, Any] | None:
        if match_stats:
            return match_stats.get(match_id)
        return None

    client.get_match_stats = AsyncMock(side_effect=_get_stats)
    client.__aenter__ = AsyncMock(return_value=client)
    client.__aexit__ = AsyncMock(return_value=None)
    return client


# ---------------------------------------------------------------------------
# Assertions helpers
# ---------------------------------------------------------------------------
def count_rows(conn: duckdb.DuckDBPyConnection, table: str) -> int:
    """Compte le nombre de lignes dans une table."""
    try:
        row = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()
        return row[0] if row else 0
    except duckdb.CatalogException:
        return 0


def get_match_ids(conn: duckdb.DuckDBPyConnection, table: str) -> set[str]:
    """Récupère les match_ids distincts d'une table."""
    rows = conn.execute(f"SELECT DISTINCT match_id FROM {table}").fetchall()
    return {str(r[0]) for r in rows}


def get_pme_scores(
    conn: duckdb.DuckDBPyConnection,
) -> dict[str, float | None]:
    """Récupère match_id → performance_score depuis PME."""
    rows = conn.execute(
        "SELECT match_id, performance_score FROM player_match_enrichment"
    ).fetchall()
    return {str(r[0]): r[1] for r in rows}


# ---------------------------------------------------------------------------
# Vues SQL v6 (ajoutées aux shared DBs de test)
# ---------------------------------------------------------------------------
V6_SHARED_VIEWS = """
CREATE OR REPLACE VIEW v_gamertag_lookup AS
SELECT
    COALESCE(xa.xuid, mp.xuid) AS xuid,
    COALESCE(xa.gamertag, mp.gamertag) AS gamertag
FROM xuid_aliases xa
FULL OUTER JOIN (
    SELECT xuid, MAX(gamertag) AS gamertag
    FROM match_participants
    WHERE gamertag IS NOT NULL
    GROUP BY xuid
) mp ON xa.xuid = mp.xuid
WHERE COALESCE(xa.gamertag, mp.gamertag) IS NOT NULL;

CREATE OR REPLACE VIEW v_weapon_kills AS
SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
FROM weapon_kills;

CREATE OR REPLACE VIEW v_killer_victim_full AS
SELECT
    kvp.*,
    ka.gamertag AS killer_gt,
    va.gamertag AS victim_gt
FROM killer_victim_pairs kvp
LEFT JOIN xuid_aliases ka ON kvp.killer_xuid = ka.xuid
LEFT JOIN xuid_aliases va ON kvp.victim_xuid = va.xuid;
"""

# ---------------------------------------------------------------------------
# Metadata DB builder (centralisé — ne plus copier dans chaque fichier test)
# ---------------------------------------------------------------------------
METADATA_SCHEMA = """
CREATE TABLE IF NOT EXISTS asset_translations (
    asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR,
    name VARCHAR, description VARCHAR,
    PRIMARY KEY (asset_id, asset_type, lang)
);
CREATE TABLE IF NOT EXISTS weapon_labels (
    weapon_id UBIGINT PRIMARY KEY,
    label_en VARCHAR, label_fr VARCHAR
);
CREATE TABLE IF NOT EXISTS career_ranks (
    rank_id INTEGER PRIMARY KEY,
    rank_name VARCHAR, rank_tier VARCHAR,
    xp_required INTEGER DEFAULT 0
);
CREATE TABLE IF NOT EXISTS citation_mappings (
    medal_name_id BIGINT PRIMARY KEY,
    citation_key VARCHAR NOT NULL
);
"""


def create_metadata_db(path: Path) -> Path:
    """Crée une metadata.duckdb avec schéma complet pour les tests."""
    path.parent.mkdir(parents=True, exist_ok=True)
    with duckdb.connect(str(path)) as conn:
        for stmt in METADATA_SCHEMA.split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)
    return path


def make_engine(
    tmp_path: Path,
    gamertag: str = GT_PLAYER_A,
    xuid: str = XUID_PLAYER_A,
    *,
    shared_read_only: bool = False,
    tokens: Any = None,
):
    """Crée un DuckDBSyncEngine avec DBs réelles dans tmp_path.

    Centralise la construction — ne plus copier _make_engine dans chaque test.
    """
    from unittest.mock import MagicMock

    from src.data.sync.engine import DuckDBSyncEngine

    data_dir = tmp_path / "data"
    player_dir = data_dir / "players" / gamertag
    warehouse = data_dir / "warehouse"

    player_db = create_player_db(player_dir / "stats.duckdb")
    shared_db = create_shared_db(warehouse / "shared_matches_v2.duckdb")
    metadata_db = create_metadata_db(warehouse / "metadata.duckdb")

    return DuckDBSyncEngine(
        player_db_path=player_db,
        xuid=xuid,
        gamertag=gamertag,
        shared_db_path=shared_db,
        metadata_db_path=metadata_db,
        shared_read_only=shared_read_only,
        tokens=tokens or MagicMock(),
    )
