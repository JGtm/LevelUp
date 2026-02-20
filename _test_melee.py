"""Test the actual batch_upsert_rows flow for melee_kills."""

import tempfile
from dataclasses import dataclass, fields
from pathlib import Path

import duckdb


@dataclass
class MockRow:
    match_id: str
    xuid: str
    team_id: int | None = None
    outcome: int | None = None
    gamertag: str | None = None
    rank: int | None = None
    score: int | None = None
    kills: int | None = None
    deaths: int | None = None
    assists: int | None = None
    shots_fired: int | None = None
    shots_hit: int | None = None
    damage_dealt: float | None = None
    damage_taken: float | None = None
    avg_life_seconds: float | None = None
    headshot_kills: int | None = None
    max_killing_spree: int | None = None
    kda: float | None = None
    accuracy: float | None = None
    time_played_seconds: int | None = None
    grenade_kills: int | None = None
    melee_kills: int | None = None
    power_weapon_kills: int | None = None
    personal_score: int | None = None
    team_mmr: float | None = None
    kills_expected: float | None = None
    kills_stddev: float | None = None
    deaths_expected: float | None = None
    deaths_stddev: float | None = None
    assists_expected: float | None = None
    assists_stddev: float | None = None


COLS = [
    "match_id",
    "xuid",
    "team_id",
    "outcome",
    "gamertag",
    "rank",
    "score",
    "kills",
    "deaths",
    "assists",
    "shots_fired",
    "shots_hit",
    "damage_dealt",
    "damage_taken",
    "avg_life_seconds",
    "headshot_kills",
    "max_killing_spree",
    "kda",
    "accuracy",
    "time_played_seconds",
    "grenade_kills",
    "melee_kills",
    "power_weapon_kills",
    "personal_score",
    "team_mmr",
    "kills_expected",
    "kills_stddev",
    "deaths_expected",
    "deaths_stddev",
    "assists_expected",
    "assists_stddev",
]

tmp = Path(tempfile.mkdtemp()) / "test.duckdb"
conn = duckdb.connect(str(tmp))
conn.execute("""
    CREATE TABLE match_participants (
        match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
        gamertag VARCHAR, team_id INTEGER, outcome INTEGER,
        rank SMALLINT, score INTEGER, kills SMALLINT, deaths SMALLINT,
        assists SMALLINT, shots_fired INTEGER, shots_hit INTEGER,
        damage_dealt FLOAT, damage_taken FLOAT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        avg_life_seconds FLOAT, headshot_kills SMALLINT,
        max_killing_spree SMALLINT, kda FLOAT, accuracy FLOAT,
        time_played_seconds INTEGER, grenade_kills SMALLINT,
        melee_kills SMALLINT, power_weapon_kills SMALLINT,
        personal_score INTEGER, team_mmr FLOAT,
        kills_expected FLOAT, kills_stddev FLOAT,
        deaths_expected FLOAT, deaths_stddev FLOAT,
        assists_expected FLOAT, assists_stddev FLOAT,
        PRIMARY KEY (match_id, xuid)
    )
""")

row = MockRow(
    match_id="m1",
    xuid="12345",
    team_id=0,
    outcome=2,
    gamertag="Test",
    rank=1,
    score=100,
    kills=10,
    deaths=5,
    assists=3,
    shots_fired=100,
    shots_hit=50,
    damage_dealt=1500.0,
    damage_taken=1000.0,
    avg_life_seconds=49.3,
    headshot_kills=4,
    max_killing_spree=3,
    kda=5.0,
    accuracy=50.0,
    time_played_seconds=600,
    grenade_kills=2,
    melee_kills=3,
    power_weapon_kills=1,
    personal_score=200,
)

row_dict = {f.name: getattr(row, f.name, None) for f in fields(row)}
values = tuple(row_dict.get(col) for col in COLS)
placeholders = ", ".join(["?"] * len(COLS))
col_list = ", ".join(COLS)
conn.execute(
    f"INSERT OR REPLACE INTO match_participants ({col_list}) VALUES ({placeholders})", values
)

r = conn.execute(
    "SELECT melee_kills, grenade_kills, power_weapon_kills, personal_score, headshot_kills FROM match_participants"
).fetchone()
print(f"melee={r[0]}, grenade={r[1]}, pw={r[2]}, ps={r[3]}, hs={r[4]}")
conn.close()
print("Done!")
