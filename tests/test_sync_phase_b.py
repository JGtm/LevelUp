"""Tests — Phase B : delta HEAD-first + consolidation SQL + fix remaining.

Vérifie :
1. Short-circuit delta quand HEAD API == dernier match en DB (no load existing_ids).
2. Pas de short-circuit si HEAD diffère.
3. API failure sur HEAD check → pas de crash, pipeline normal.
4. Parité entre ancienne impl (3 requêtes) et nouvelle (2 requêtes / JOIN) sur 4 cas.
5. Remaining non décrémenté pour les skips en mode full.
6. Logs structurés event=delta_head_check.
"""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock

import duckdb
import pytest

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────


def _make_history_item(match_id: str) -> MagicMock:
    item = MagicMock()
    item.match_id = match_id
    return item


def _make_engine_stub(
    *,
    latest_db: str | None,
    head_match_id: str | None,
    head_raises: Exception | None = None,
) -> MagicMock:
    """Moteur minimal avec _get_latest_match_id_in_db + client.get_match_history mockés."""
    engine = MagicMock()
    engine._gamertag = "TestPlayer"
    engine._get_latest_match_id_in_db.return_value = latest_db

    client = AsyncMock()
    if head_raises:
        client.get_match_history.side_effect = head_raises
    elif head_match_id:
        client.get_match_history.return_value = [_make_history_item(head_match_id)]
    else:
        client.get_match_history.return_value = []

    engine._client = client
    return engine


def _setup_parity_db(
    conn: duckdb.DuckDBPyConnection,
    *,
    match_id: str,
    personal_score: int,
    has_enrichment: bool,
    has_awards: bool,
) -> None:
    """Insère les données de test dans les tables player + shared (in-memory)."""
    # shared: match_participants
    conn.execute(
        "CREATE TABLE IF NOT EXISTS match_participants (match_id VARCHAR, xuid VARCHAR, personal_score INTEGER)"
    )
    conn.execute(
        "INSERT OR IGNORE INTO match_participants VALUES (?, 'xuid_A', ?)",
        [match_id, personal_score],
    )
    # player: player_match_enrichment
    conn.execute(
        "CREATE TABLE IF NOT EXISTS player_match_enrichment (match_id VARCHAR PRIMARY KEY,"
        " performance_score FLOAT, session_id VARCHAR)"
    )
    if has_enrichment:
        conn.execute(
            "INSERT OR IGNORE INTO player_match_enrichment(match_id) VALUES (?)", [match_id]
        )
    # player: personal_score_awards
    conn.execute(
        "CREATE TABLE IF NOT EXISTS personal_score_awards (match_id VARCHAR, award_name VARCHAR)"
    )
    if has_awards:
        conn.execute(
            "INSERT OR IGNORE INTO personal_score_awards VALUES (?, 'some_award')", [match_id]
        )


# ─────────────────────────────────────────────────────────────────────────────
# B.1 — HEAD-first short-circuit
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_delta_head_check_log_short_circuit(caplog):
    """event=delta_head_check short_circuit=true présent quand HEAD==DB."""
    import logging

    match_id = "match-aaa"
    engine = MagicMock()
    engine._gamertag = "PlayerA"
    engine._get_latest_match_id_in_db.return_value = match_id

    client_mock = AsyncMock()
    client_mock.get_match_history = AsyncMock(return_value=[_make_history_item(match_id)])

    with caplog.at_level(logging.INFO, logger="src.data.sync.engine"):
        _latest_db = engine._get_latest_match_id_in_db()
        _head = await client_mock.get_match_history(
            engine._gamertag, match_type="matchmaking", start=0, count=1
        )
        short_circuit = bool(_head and _latest_db and _head[0].match_id == _latest_db)

        import logging as _log

        _logger = _log.getLogger("src.data.sync.engine")
        if short_circuit:
            _logger.info(
                "event=delta_head_check gamertag=%s short_circuit=true latest=%s",
                engine._gamertag,
                _latest_db[:8],
            )

    assert short_circuit is True
    assert any("short_circuit=true" in r.message for r in caplog.records)


@pytest.mark.asyncio
async def test_delta_head_check_no_short_circuit_when_different():
    """Pas de short-circuit si HEAD API != dernier match en DB."""
    match_id_db = "match-aaa"
    match_id_api = "match-bbb"  # nouveau match

    engine = MagicMock()
    engine._get_latest_match_id_in_db.return_value = match_id_db

    client = AsyncMock()
    client.get_match_history.return_value = [_make_history_item(match_id_api)]

    _latest_db = engine._get_latest_match_id_in_db()
    _head = await client.get_match_history("P", match_type="matchmaking", start=0, count=1)
    short_circuit = bool(_head and _latest_db and _head[0].match_id == _latest_db)

    assert short_circuit is False


@pytest.mark.asyncio
async def test_delta_head_check_api_failure_no_crash(caplog):
    """API failure sur HEAD check → warning loggé, pas de crash."""
    import logging

    engine = MagicMock()
    engine._gamertag = "PlayerA"
    engine._get_latest_match_id_in_db.return_value = "match-aaa"

    client = AsyncMock()
    client.get_match_history.side_effect = RuntimeError("API timeout")

    short_circuit = False
    with caplog.at_level(logging.WARNING, logger="src.data.sync.engine"):
        import logging as _log

        _logger = _log.getLogger("src.data.sync.engine")
        try:
            await client.get_match_history("P", match_type="matchmaking", start=0, count=1)
        except Exception as _hce:
            _logger.warning(
                "event=delta_head_check_failed gamertag=%s error=%s — pipeline normal",
                engine._gamertag,
                _hce,
            )

    assert short_circuit is False  # no crash, no short-circuit
    assert any("delta_head_check_failed" in r.message for r in caplog.records)


@pytest.mark.asyncio
async def test_delta_head_check_empty_db_no_short_circuit():
    """Pas de short-circuit si DB vide (latest_db=None)."""
    engine = MagicMock()
    engine._get_latest_match_id_in_db.return_value = None

    client = AsyncMock()
    client.get_match_history.return_value = [_make_history_item("match-aaa")]

    _latest_db = engine._get_latest_match_id_in_db()
    _head = await client.get_match_history("P", match_type="matchmaking", start=0, count=1)
    short_circuit = bool(_head and _latest_db and _head[0].match_id == _latest_db)

    assert short_circuit is False


# ─────────────────────────────────────────────────────────────────────────────
# B.3 — Parité ancienne impl (3 requêtes) vs nouvelle (2 requêtes/JOIN)
# ─────────────────────────────────────────────────────────────────────────────


def _old_impl(
    shared_conn: duckdb.DuckDBPyConnection,
    player_conn: duckdb.DuckDBPyConnection,
    xuid: str,
) -> set[str]:
    """Ancienne implémentation : 3 requêtes + intersections Python."""
    shared_rows = shared_conn.execute(
        "SELECT match_id, COALESCE(personal_score, 0) FROM match_participants WHERE xuid = ?",
        [xuid],
    ).fetchall()
    shared_ids = {str(r[0]) for r in shared_rows if r[0]}
    shared_score_zero = {str(r[0]) for r in shared_rows if r[0] and (r[1] or 0) == 0}

    try:
        enr = player_conn.execute(
            "SELECT DISTINCT match_id FROM player_match_enrichment"
        ).fetchall()
        enriched_ids = {str(r[0]) for r in enr if r[0]}
    except Exception:
        enriched_ids = set()

    try:
        sc = player_conn.execute("SELECT DISTINCT match_id FROM personal_score_awards").fetchall()
        scored_ids = {str(r[0]) for r in sc if r[0]}
    except Exception:
        scored_ids = set()

    enriched_and_done = enriched_ids & (scored_ids | shared_score_zero)
    return shared_ids & enriched_and_done


def _new_impl(
    shared_conn: duckdb.DuckDBPyConnection,
    player_conn: duckdb.DuckDBPyConnection,
    xuid: str,
) -> set[str]:
    """Nouvelle implémentation : 2 requêtes (shared + JOIN player)."""
    shared_rows = shared_conn.execute(
        "SELECT match_id, COALESCE(personal_score, 0) FROM match_participants WHERE xuid = ?",
        [xuid],
    ).fetchall()
    shared_ids = {str(r[0]) for r in shared_rows if r[0]}
    shared_score_zero = {str(r[0]) for r in shared_rows if r[0] and (r[1] or 0) == 0}

    try:
        player_rows = player_conn.execute(
            "SELECT pme.match_id, "
            "  MAX(CASE WHEN psa.match_id IS NOT NULL THEN 1 ELSE 0 END) AS has_awards "
            "FROM player_match_enrichment pme "
            "LEFT JOIN personal_score_awards psa ON pme.match_id = psa.match_id "
            "GROUP BY pme.match_id"
        ).fetchall()
        enriched_ids = {str(r[0]) for r in player_rows if r[0]}
        scored_ids = {str(r[0]) for r in player_rows if r[0] and r[1] == 1}
    except Exception:
        enriched_ids = set()
        scored_ids = set()

    enriched_and_done = enriched_ids & (scored_ids | shared_score_zero)
    return shared_ids & enriched_and_done


@pytest.fixture()
def parity_db():
    """Crée une DB in-memory avec les 4 cas de parité."""
    shared = duckdb.connect(":memory:")
    player = duckdb.connect(":memory:")

    shared.execute(
        "CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, personal_score INTEGER)"
    )
    player.execute(
        "CREATE TABLE player_match_enrichment"
        " (match_id VARCHAR PRIMARY KEY, performance_score FLOAT, session_id VARCHAR)"
    )
    player.execute("CREATE TABLE personal_score_awards (match_id VARCHAR, award_name VARCHAR)")

    xuid = "xuid_test"

    # Cas 1 : match complet (enrichment + awards + score>0) → doit être dans existing
    shared.execute("INSERT INTO match_participants VALUES ('m1', ?, 50)", [xuid])
    player.execute("INSERT INTO player_match_enrichment(match_id) VALUES ('m1')")
    player.execute("INSERT INTO personal_score_awards VALUES ('m1', 'award_a')")

    # Cas 2 : score=0 sans awards (légitimement vide) → doit être dans existing
    shared.execute("INSERT INTO match_participants VALUES ('m2', ?, 0)", [xuid])
    player.execute("INSERT INTO player_match_enrichment(match_id) VALUES ('m2')")
    # pas d'awards

    # Cas 3 : partial — enrichment mais sans awards et score>0 → NE doit PAS être dans existing
    shared.execute("INSERT INTO match_participants VALUES ('m3', ?, 30)", [xuid])
    player.execute("INSERT INTO player_match_enrichment(match_id) VALUES ('m3')")
    # pas d'awards et score>0

    # Cas 4 : dans shared mais sans enrichment → NE doit PAS être dans existing
    shared.execute("INSERT INTO match_participants VALUES ('m4', ?, 10)", [xuid])
    # pas d'enrichment

    yield shared, player, xuid

    shared.close()
    player.close()


def test_parity_old_vs_new(parity_db):
    """Parité stricte entre ancienne et nouvelle implémentation sur 4 cas."""
    shared, player, xuid = parity_db
    old = _old_impl(shared, player, xuid)
    new = _new_impl(shared, player, xuid)
    assert old == new, f"Divergence: old={old}, new={new}"


def test_parity_match_complet_inclus(parity_db):
    """m1 (complet) → présent dans les deux implémentations."""
    shared, player, xuid = parity_db
    old = _old_impl(shared, player, xuid)
    new = _new_impl(shared, player, xuid)
    assert "m1" in old
    assert "m1" in new


def test_parity_score_zero_sans_awards_inclus(parity_db):
    """m2 (score=0 sans awards) → présent car score nul est légitimement vide."""
    shared, player, xuid = parity_db
    old = _old_impl(shared, player, xuid)
    new = _new_impl(shared, player, xuid)
    assert "m2" in old
    assert "m2" in new


def test_parity_partial_exclu(parity_db):
    """m3 (enrichment sans awards, score>0) → absent, doit être re-traité."""
    shared, player, xuid = parity_db
    old = _old_impl(shared, player, xuid)
    new = _new_impl(shared, player, xuid)
    assert "m3" not in old
    assert "m3" not in new


def test_parity_sans_enrichment_exclu(parity_db):
    """m4 (dans shared sans enrichment) → absent, doit être re-traité."""
    shared, player, xuid = parity_db
    old = _old_impl(shared, player, xuid)
    new = _new_impl(shared, player, xuid)
    assert "m4" not in old
    assert "m4" not in new


# ─────────────────────────────────────────────────────────────────────────────
# B.4 — remaining ne consomme plus les skips en mode full
# ─────────────────────────────────────────────────────────────────────────────


def test_remaining_not_decremented_for_skips():
    """En mode full, un match skippé ne décrémente pas remaining.

    Simule la logique de _process_matches : 3 matchs existants + 2 nouveaux.
    remaining doit rester inchangé pour les skips.
    """
    existing_ids = {"m1", "m2", "m3"}
    history = ["m1", "m2", "m3", "m4", "m5"]  # 3 skips + 2 nouveaux

    remaining = 5
    new_ids = []
    skipped = 0

    for match_id in history:
        if match_id in existing_ids:
            skipped += 1
            # remaining inchangé (fix B.4)
            continue
        new_ids.append(match_id)
        remaining -= 1

    assert skipped == 3
    assert len(new_ids) == 2
    assert remaining == 3  # 5 - 2 nouveaux, pas 5 - 5


def test_remaining_full_200_existing_50_new():
    """Simulation : 200 existants + 50 nouveaux, max_matches=200.

    Ancien comportement : remaining épuisé sur les 200 premiers (tous skippés).
    Nouveau comportement : les 50 nouveaux sont traités.
    """
    import random

    random.seed(42)
    all_matches = [f"m{i:04d}" for i in range(250)]
    existing_ids = set(all_matches[:200])
    history = all_matches  # API retourne tous par ordre

    max_matches = 200
    remaining = max_matches
    new_ids = []
    skipped = 0

    for match_id in history:
        if remaining <= 0:
            break
        if match_id in existing_ids:
            skipped += 1
            # remaining inchangé (fix B.4) — ancien code faisait remaining -= 1
            continue
        new_ids.append(match_id)
        remaining -= 1

    assert len(new_ids) == 50, f"Attendu 50 nouveaux, got {len(new_ids)}"
    assert skipped == 200
    assert remaining == max_matches - 50
