"""Tests de non-régression pour le bug de cascade LUSR en mode incrémental.

Problème (avant fix) :
    En mode incrémental, batch_compute_lusr repassait TOUTE l'historique des
    matchs à compute_skill_ratings_batch avec un seed (last_stored_rating, sigma)
    injecté comme µ₀ au DÉBUT de l'historique. TrueSkill retraite 400+ matchs
    depuis ce µ₀ décalé → la valeur finale diverge de la valeur stockée, et le
    nouveau match hérite de cette valeur divergente. À chaque sync séparé, le
    rating absolu dérive de ~160 pts (cascade constatée sur Madina97294).

Fix :
    En mode incrémental, seuls les nouveaux matchs (absents de match_skill_rank)
    sont passés à compute_skill_ratings_batch. existing_states sert de seed
    représentant l'état APRÈS le dernier match déjà calculé.

Invariant vérifié :
    compute_incremental(match_A_state, [match_B]) == compute_full([A, B])[B]

    Autrement dit : calculer un match seul en partant du bon seed donne
    exactement le même résultat que le calculer dans un seul batch complet.
"""

from __future__ import annotations

import math
from datetime import datetime, timedelta, timezone

import duckdb
import polars as pl
import pytest

from src.analysis.skill_rating import PlayerState, compute_skill_ratings_batch
from src.analysis.skill_rating_config import INITIAL_MU
from src.data.sync._skill_rating import _LUSR_MAX_DELTA, SkillRatingMixin

# =============================================================================
# Helpers
# =============================================================================

_T0 = datetime(2025, 1, 1, 12, 0, 0, tzinfo=timezone.utc)


def _match(  # noqa: PLR0913
    *,
    match_id: str,
    hour: int = 0,
    outcome: int = 2,  # WIN
    kills: float = 12.0,
    kills_expected: float = 10.0,
    deaths: float = 5.0,
    deaths_expected: float = 5.0,
    accuracy: float = 35.0,
    playlist_name: str = "Quick Play",
    pair_name: str = "Arena:Slayer on Aquarius",
) -> dict:
    """Crée un dict de match simplifié pour compute_skill_ratings_batch."""
    return {
        "match_id": match_id,
        "start_time": _T0 + timedelta(hours=hour),
        "outcome": outcome,
        "kills": kills,
        "kills_expected": kills_expected,
        "deaths": deaths,
        "deaths_expected": deaths_expected,
        "damage_dealt": 5000.0,
        "damage_taken": 3000.0,
        "accuracy": accuracy,
        "team_id": 0,
        "playlist_name": playlist_name,
        "pair_name": pair_name,
        "is_ranked": False,
    }


def _df(*rows: dict) -> pl.DataFrame:
    return pl.DataFrame(list(rows))


def _empty_participants() -> pl.DataFrame:
    return pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "xuid": pl.Utf8,
            "team_id": pl.Int32,
            "kills_expected": pl.Float64,
            "deaths_expected": pl.Float64,
        }
    )


# =============================================================================
# Invariant de continuité — algorithme pur (sans DB)
# =============================================================================


class TestIncrementalContinuityInvariant:
    """Vérifie que le calcul incrémental (match par match) est équivalent
    au calcul en batch complet.

    C'est l'invariant fondamental que le fix restaure.
    """

    def test_single_match_incremental_equals_full_batch(self) -> None:
        """1 match seul → même résultat que dans un batch de 1."""
        row = _match(match_id="m1")
        result_full = compute_skill_ratings_batch(_df(row), _empty_participants())
        result_incr = compute_skill_ratings_batch(_df(row), _empty_participants())
        assert result_full["rating_value"][0] == pytest.approx(
            result_incr["rating_value"][0], abs=0.01
        )

    def test_two_matches_incremental_equals_full_batch(self) -> None:
        """Calculer m1 seul, seed depuis ce résultat, puis m2 seul
        doit donner le même rating pour m2 que calculer m1+m2 ensemble."""
        row1 = _match(match_id="m1", hour=0, outcome=2, kills=14, kills_expected=10)
        row2 = _match(match_id="m2", hour=1, outcome=3, kills=6, kills_expected=10)
        dp = _empty_participants()

        # ── Batch complet (référence) ──
        full = compute_skill_ratings_batch(_df(row1, row2), dp)
        rating_m2_full = full.filter(pl.col("match_id") == "m2")["rating_value"][0]
        rating_m1_full = full.filter(pl.col("match_id") == "m1")["rating_value"][0]

        # ── Incrémental : m1 uniquement ──
        res1 = compute_skill_ratings_batch(_df(row1), dp)
        rating_m1_incr = res1["rating_value"][0]
        sigma_m1_incr = res1["rating_deviation"][0]

        # Le rating de m1 doit être identique dans les deux cas
        assert rating_m1_incr == pytest.approx(rating_m1_full, abs=0.01)

        # ── Incrémental : m2 uniquement, seedé depuis m1 ──
        seed_state = PlayerState(mu=rating_m1_incr, sigma=sigma_m1_incr)
        res2 = compute_skill_ratings_batch(_df(row2), dp, existing_states={"arena": seed_state})
        rating_m2_incr = res2["rating_value"][0]

        # L'invariant : rating m2 est identique dans les deux modes
        assert rating_m2_incr == pytest.approx(rating_m2_full, abs=0.5), (
            f"Continuité rompue : batch={rating_m2_full:.2f}, "
            f"incrémental={rating_m2_incr:.2f} "
            f"(écart={abs(rating_m2_incr - rating_m2_full):.2f})"
        )

    def test_five_matches_all_incremental_equals_full_batch(self) -> None:
        """Cinq matchs calculés un par un donnent les mêmes ratings que en un seul batch."""
        rows = [_match(match_id=f"m{i}", hour=i, outcome=2 if i % 2 == 0 else 3) for i in range(5)]
        dp = _empty_participants()

        # ── Référence batch complet ──
        full = compute_skill_ratings_batch(_df(*rows), dp)
        full_ratings = {
            row["match_id"]: full.filter(pl.col("match_id") == row["match_id"])["rating_value"][0]
            for row in rows
        }

        # ── Incrémental match par match ──
        current_state: dict[str, PlayerState] = {}
        for row in rows:
            res = compute_skill_ratings_batch(
                _df(row),
                dp,
                existing_states=current_state if current_state else None,
            )
            r = res["rating_value"][0]
            sigma = res["rating_deviation"][0]
            group = res["playlist_group"][0]
            current_state[group] = PlayerState(mu=r, sigma=sigma)

            expected = full_ratings[row["match_id"]]
            assert r == pytest.approx(expected, abs=0.5), (
                f"Match {row['match_id']} : incrémental={r:.2f}, "
                f"batch={expected:.2f}, écart={abs(r - expected):.2f}"
            )

    def test_incremental_groups_are_independent(self) -> None:
        """Seed arena ne contamine pas le groupe ranked."""
        row_arena = _match(match_id="a1", hour=0, pair_name="Arena:Slayer on Aquarius")
        row_ranked = _match(
            match_id="r1",
            hour=1,
            pair_name="Ranked:Slayer on Aquarius",
            playlist_name="Ranked Arena",
        )
        dp = _empty_participants()

        # Batch complet
        full = compute_skill_ratings_batch(_df(row_arena, row_ranked), dp)
        rated_r = full.filter(pl.col("match_id") == "r1")["rating_value"][0]

        # Incrémental : arena d'abord, puis ranked
        res_a = compute_skill_ratings_batch(_df(row_arena), dp)
        state_a = PlayerState(mu=res_a["rating_value"][0], sigma=res_a["rating_deviation"][0])
        res_r = compute_skill_ratings_batch(_df(row_ranked), dp, existing_states={"arena": state_a})
        # ranked doit partir de INITIAL_MU (état absent → groupe créé à 0)
        # → ne pas hériter du seed arena
        incr_r = res_r["rating_value"][0]
        assert incr_r == pytest.approx(rated_r, abs=0.5)


# =============================================================================
# Détection de la cascade (scénario Madina)
# =============================================================================


class TestCascadeDriftDetection:
    """Vérifie qu'en mode incrémental correct, le rating n'accumule pas
    de dérive entre des syncs séparés."""

    def test_no_cascade_across_separate_syncs(self) -> None:
        """Simule 5 syncs d'un seul match chacun.

        Le rating final en mode correct (incrémental propre) doit être proche
        du rating calculé en un batch unique. Un écart > 100 pts indique la
        présence de la cascade des seeds décalés (bug).
        """
        # 5 matchs : victoires alternées avec défaites
        rows = [
            _match(
                match_id=f"m{i}",
                hour=i,
                outcome=2 if i % 2 == 0 else 3,
                kills=14.0 if i % 2 == 0 else 6.0,
                kills_expected=10.0,
            )
            for i in range(5)
        ]
        dp = _empty_participants()

        # ── Référence : batch unique ──
        full = compute_skill_ratings_batch(_df(*rows), dp)
        ref_final_rating = full["rating_value"][-1]

        # ── Mode correct (incrémental propre) ──
        state: dict[str, PlayerState] = {}
        for row in rows:
            res = compute_skill_ratings_batch(
                _df(row), dp, existing_states=state if state else None
            )
            group = res["playlist_group"][0]
            state[group] = PlayerState(mu=res["rating_value"][0], sigma=res["rating_deviation"][0])
        incr_final_rating = state.get("arena", PlayerState()).mu

        # L'écart entre mode correct et référence doit être négligeable
        drift = abs(incr_final_rating - ref_final_rating)
        assert drift < 1.0, (
            f"Cascade détectée : drift={drift:.2f} pts "
            f"(batch={ref_final_rating:.2f}, incrémental={incr_final_rating:.2f})"
        )

    def test_wrong_seed_causes_large_drift(self) -> None:
        """Démontre que la simulation du bug (seed décalé + recompute complet)
        produit bien une dérive. Ce test sert de preuve que le défaut existait.

        Scénario : 3 matchs déjà calculés (rating_stored = X), puis 1 nouveau
        match. Si on recompute les 4 matchs avec seed µ₀=X (mauvais comportement),
        la valeur finale diverge de la valeur correcte.
        """
        match_history = [
            _match(match_id=f"h{i}", hour=i, outcome=2, kills=14, kills_expected=10)
            for i in range(3)
        ]
        new_match = _match(match_id="new", hour=3, outcome=2, kills=14, kills_expected=10)
        dp = _empty_participants()

        # ── Calcul correct (batch complet) ──
        full = compute_skill_ratings_batch(_df(*match_history, new_match), dp)
        correct_rating = full.filter(pl.col("match_id") == "new")["rating_value"][0]

        # ── Calcul correct (incrémental) ──
        full_hist = compute_skill_ratings_batch(_df(*match_history), dp)
        last_rating = full_hist["rating_value"][-1]
        last_sigma = full_hist["rating_deviation"][-1]
        state_correct = {"arena": PlayerState(mu=last_rating, sigma=last_sigma)}
        incr_res = compute_skill_ratings_batch(_df(new_match), dp, existing_states=state_correct)
        incr_rating = incr_res["rating_value"][0]
        # Ceci doit être identique au batch complet
        assert incr_rating == pytest.approx(correct_rating, abs=0.5)

        # ── Simulation du bug : seed mu₀ = last_rating, mais recompute TOUT ──
        # (c'est ce que faisait l'ancien code : injecter le seed avant le premier
        # match de l'historique, puis retraiter tous les N matchs)
        buggy_state = {"arena": PlayerState(mu=last_rating, sigma=last_sigma)}
        buggy_res = compute_skill_ratings_batch(
            _df(*match_history, new_match), dp, existing_states=buggy_state
        )
        buggy_rating_for_new = buggy_res.filter(pl.col("match_id") == "new")["rating_value"][0]

        # Le bug produit un rating différent du résultat correct
        buggy_drift = abs(buggy_rating_for_new - correct_rating)
        # Avec un historique de victoires et un µ₀ < INITIAL_MU, TrueSkill
        # converge vers un plateau plus bas → dérive mesurable.
        # On vérifie simplement que le comportement buggy n'est PAS identique
        # au correct (ce serait une coïncidence mathématiquement impossible
        # sauf si last_rating == INITIAL_MU).
        if not math.isclose(last_rating, INITIAL_MU, abs_tol=1.0):
            assert buggy_drift > 0.1, (
                "Le bug de cascade n'a pas produit de dérive — vérifier le scénario"
            )


# =============================================================================
# Tests _upsert_lusr_ratings — seed_ratings
# =============================================================================


def _make_rating_row(
    match_id: str,
    rating_value: float,
    playlist_group: str = "arena",
) -> dict:
    return {
        "match_id": match_id,
        "rating_value": float(rating_value),
        "rating_deviation": 100.0,
        "playlist_group": playlist_group,
    }


def _make_ratings_df(*rows: dict) -> pl.DataFrame:
    return pl.DataFrame(list(rows))


def _make_matches_df(match_id: str, start_time: datetime) -> pl.DataFrame:
    return pl.DataFrame(
        {
            "match_id": [match_id],
            "start_time": [start_time],
        }
    )


def _build_mixin_instance(player_conn: duckdb.DuckDBPyConnection) -> SkillRatingMixin:
    """Crée un objet minimal conforme au protocole SkillRatingMixin pour les tests."""

    class _FakeMixin(SkillRatingMixin):
        def _get_connection(self):  # type: ignore[override]
            return player_conn

        def _get_shared_connection(self):  # type: ignore[override]
            return None

        _xuid = "test_xuid"
        _gamertag = "TestPlayer"
        _player_db_path = None  # type: ignore[assignment]

    return _FakeMixin()


@pytest.fixture()
def in_memory_db():
    """Base DuckDB in-memory avec match_skill_rank prête à l'emploi."""
    con = duckdb.connect(":memory:")
    from src.data.sync.migrations import ensure_match_skill_rank_table

    ensure_match_skill_rank_table(con)
    yield con
    con.close()


class TestUpsertLusrRatingsSeedRatings:
    """Tests unitaires de _upsert_lusr_ratings avec seed_ratings."""

    def test_delta_is_none_without_seed(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Sans seed_ratings, le premier match d'un batch a delta=None."""
        mixin = _build_mixin_instance(in_memory_db)
        ratings_df = _make_ratings_df(_make_rating_row("m1", 1550.0))
        df_matches = _make_matches_df("m1", _T0)

        mixin._upsert_lusr_ratings(
            in_memory_db, ratings_df, df_matches, set(), set(), force=False, seed_ratings=None
        )

        row = in_memory_db.execute(
            "SELECT rating_delta FROM match_skill_rank WHERE match_id = 'm1'"
        ).fetchone()
        assert row is not None
        assert row[0] is None  # pas de prev_rating → delta None

    def test_delta_correct_with_seed(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Avec seed_ratings, le delta du premier match equals rating - seed."""
        mixin = _build_mixin_instance(in_memory_db)
        seed_value = 1500.0
        new_value = 1508.3  # +8.3 par rapport au seed
        ratings_df = _make_ratings_df(_make_rating_row("m1", new_value))
        df_matches = _make_matches_df("m1", _T0)

        mixin._upsert_lusr_ratings(
            in_memory_db,
            ratings_df,
            df_matches,
            set(),
            set(),
            force=False,
            seed_ratings={"arena": seed_value},
        )

        row = in_memory_db.execute(
            "SELECT rating_delta, rating_value FROM match_skill_rank WHERE match_id = 'm1'"
        ).fetchone()
        assert row is not None
        assert row[0] == pytest.approx(new_value - seed_value, abs=0.01)
        assert row[1] == pytest.approx(new_value, abs=0.01)

    def test_seed_ratings_not_confused_across_groups(
        self, in_memory_db: duckdb.DuckDBPyConnection
    ) -> None:
        """Le seed d'un groupe ne pollue pas le delta d'un autre groupe."""
        mixin = _build_mixin_instance(in_memory_db)
        ratings_df = _make_ratings_df(
            _make_rating_row("m_arena", 1520.0, "arena"),
            _make_rating_row("m_btb", 1480.0, "btb"),
        )
        df_matches = pl.DataFrame(
            {
                "match_id": ["m_arena", "m_btb"],
                "start_time": [_T0, _T0 + timedelta(hours=1)],
            }
        )
        mixin._upsert_lusr_ratings(
            in_memory_db,
            ratings_df,
            df_matches,
            set(),
            set(),
            force=False,
            seed_ratings={"arena": 1510.0},  # seed pour arena seulement
        )

        arena_row = in_memory_db.execute(
            "SELECT rating_delta FROM match_skill_rank WHERE match_id = 'm_arena'"
        ).fetchone()
        btb_row = in_memory_db.execute(
            "SELECT rating_delta FROM match_skill_rank WHERE match_id = 'm_btb'"
        ).fetchone()

        # arena : delta = 1520 - 1510 = 10
        assert arena_row is not None
        assert arena_row[0] == pytest.approx(10.0, abs=0.01)
        # btb : pas de seed → delta None (premier match du groupe)
        assert btb_row is not None
        assert btb_row[0] is None

    def test_guard_rail_still_fires_with_seed(
        self, in_memory_db: duckdb.DuckDBPyConnection
    ) -> None:
        """Le cap ±100 pts s'applique même quand seed_ratings est fourni."""
        mixin = _build_mixin_instance(in_memory_db)
        seed_value = 1500.0
        extreme_value = 1800.0  # +300 par rapport au seed → doit être capé à +100
        ratings_df = _make_ratings_df(_make_rating_row("m1", extreme_value))
        df_matches = _make_matches_df("m1", _T0)

        mixin._upsert_lusr_ratings(
            in_memory_db,
            ratings_df,
            df_matches,
            set(),
            set(),
            force=False,
            seed_ratings={"arena": seed_value},
        )

        row = in_memory_db.execute(
            "SELECT rating_value, rating_delta FROM match_skill_rank WHERE match_id = 'm1'"
        ).fetchone()
        assert row is not None
        stored_rating, stored_delta = row
        assert stored_delta == pytest.approx(_LUSR_MAX_DELTA, abs=0.01)
        assert stored_rating == pytest.approx(seed_value + _LUSR_MAX_DELTA, abs=0.01)

    def test_existing_lusr_match_skipped_incremental(
        self, in_memory_db: duckdb.DuckDBPyConnection
    ) -> None:
        """En mode non-force, un match déjà en DB n'est pas écrasé."""
        mixin = _build_mixin_instance(in_memory_db)
        # Pré-insérer m1 avec un rating connu
        in_memory_db.execute(
            """INSERT INTO match_skill_rank
               (match_id, rating_type, rating_value, created_at, updated_at)
               VALUES ('m1', 'LUSR', 1550.0, NOW(), NOW())"""
        )

        ratings_df = _make_ratings_df(_make_rating_row("m1", 1600.0))  # valeur différente
        df_matches = _make_matches_df("m1", _T0)

        mixin._upsert_lusr_ratings(
            in_memory_db,
            ratings_df,
            df_matches,
            set(),
            {"m1"},  # déjà calculé
            force=False,
        )

        row = in_memory_db.execute(
            "SELECT rating_value FROM match_skill_rank WHERE match_id = 'm1'"
        ).fetchone()
        # La valeur 1550 doit rester inchangée (match skippé)
        # Note : le ON CONFLICT DO UPDATE dans le SQL va écraser — le skip est
        # dans _upsert_lusr_ratings via le continue. Vérifions qu'aucune ligne n'a été écrite.
        # En fait, le mixin fait `continue` avant d'ajouter à rows_to_insert.
        # La valeur reste 1550.
        assert row is not None
        assert row[0] == pytest.approx(1550.0, abs=0.01)

    def test_csr_match_skipped_in_upsert(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Un match protégé par existing_csr_ids n'est pas écrasé par LUSR."""
        mixin = _build_mixin_instance(in_memory_db)
        # Pré-insérer m_csr avec type CSR
        in_memory_db.execute(
            """INSERT INTO match_skill_rank
               (match_id, rating_type, rating_value, created_at, updated_at)
               VALUES ('m_csr', 'CSR', 1700.0, NOW(), NOW())"""
        )
        ratings_df = _make_ratings_df(_make_rating_row("m_csr", 1550.0))
        df_matches = _make_matches_df("m_csr", _T0)

        count = mixin._upsert_lusr_ratings(
            in_memory_db,
            ratings_df,
            df_matches,
            {"m_csr"},  # match protégé par CSR
            set(),
            force=False,
        )
        assert count == 0

        # Le rating CSR doit rester inchangé
        row = in_memory_db.execute(
            "SELECT rating_type, rating_value FROM match_skill_rank WHERE match_id = 'm_csr'"
        ).fetchone()
        assert row is not None
        assert row[0] == "CSR"
        assert row[1] == pytest.approx(1700.0, abs=0.01)


# =============================================================================
# Tests _load_existing_lusr_states
# =============================================================================


class TestLoadExistingLusrStates:
    """Tests pour _load_existing_lusr_states — chargement des seeds depuis la DB."""

    def test_empty_db_returns_empty_dict(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Sans données LUSR, retourne un dict vide."""
        mixin = _build_mixin_instance(in_memory_db)
        states = mixin._load_existing_lusr_states(in_memory_db)
        assert states == {}

    def test_loads_last_state_per_group(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Chaque groupe → le dernier état (mu, sigma) par start_time."""
        in_memory_db.execute(
            """INSERT INTO match_skill_rank
               (match_id, rating_type, rating_value, rating_deviation,
                playlist_group, start_time, created_at, updated_at)
               VALUES
               ('old', 'LUSR', 1450.0, 80.0, 'arena', '2025-01-01', NOW(), NOW()),
               ('new', 'LUSR', 1520.0, 75.0, 'arena', '2025-01-02', NOW(), NOW())"""
        )
        mixin = _build_mixin_instance(in_memory_db)
        states = mixin._load_existing_lusr_states(in_memory_db)

        assert "arena" in states
        assert states["arena"].mu == pytest.approx(1520.0, abs=0.01)
        assert states["arena"].sigma == pytest.approx(75.0, abs=0.01)

    def test_multiple_groups_isolated(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Plusieurs groupes → états indépendants."""
        in_memory_db.execute(
            """INSERT INTO match_skill_rank
               (match_id, rating_type, rating_value, rating_deviation,
                playlist_group, start_time, created_at, updated_at)
               VALUES
               ('a1', 'LUSR', 1600.0, 70.0, 'arena', '2025-01-01', NOW(), NOW()),
               ('b1', 'LUSR', 1400.0, 90.0, 'btb',   '2025-01-01', NOW(), NOW())"""
        )
        mixin = _build_mixin_instance(in_memory_db)
        states = mixin._load_existing_lusr_states(in_memory_db)

        assert states["arena"].mu == pytest.approx(1600.0, abs=0.01)
        assert states["btb"].mu == pytest.approx(1400.0, abs=0.01)

    def test_sigma_defaults_when_null(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Si rating_deviation est NULL, sigma prend INITIAL_SIGMA."""
        from src.analysis.skill_rating_config import INITIAL_SIGMA

        in_memory_db.execute(
            """INSERT INTO match_skill_rank
               (match_id, rating_type, rating_value, rating_deviation,
                playlist_group, start_time, created_at, updated_at)
               VALUES ('m1', 'LUSR', 1500.0, NULL, 'arena', '2025-01-01', NOW(), NOW())"""
        )
        mixin = _build_mixin_instance(in_memory_db)
        states = mixin._load_existing_lusr_states(in_memory_db)

        assert "arena" in states
        assert states["arena"].sigma == pytest.approx(INITIAL_SIGMA, abs=0.01)


# =============================================================================
# Tests batch_compute_lusr (end-to-end avec fake engine)
# =============================================================================


def _build_batch_mixin(
    player_conn: duckdb.DuckDBPyConnection,
    match_rows: list[dict],
    participant_rows: list[dict] | None = None,
) -> SkillRatingMixin:
    """Construit un FakeMixin avec _load_lusr_match_data mocké."""
    df_matches_fixed = (
        pl.DataFrame(match_rows)
        if match_rows
        else pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Datetime("us", "UTC"),
                "playlist_name": pl.Utf8,
                "pair_name": pl.Utf8,
                "outcome": pl.Int32,
                "kills": pl.Float64,
                "deaths": pl.Float64,
                "kills_expected": pl.Float64,
                "deaths_expected": pl.Float64,
                "damage_dealt": pl.Float64,
                "damage_taken": pl.Float64,
                "accuracy": pl.Float64,
                "team_id": pl.Int32,
            }
        )
    )
    dp_fixed = pl.DataFrame(participant_rows) if participant_rows else _empty_participants()

    class _FakeBatchMixin(SkillRatingMixin):
        def _get_connection(self):  # type: ignore[override]
            return player_conn

        def _get_shared_connection(self):  # type: ignore[override]
            return None

        def _load_lusr_match_data(self):  # type: ignore[override]
            if df_matches_fixed.is_empty():
                return None
            return df_matches_fixed, dp_fixed

        _xuid = "test_xuid"
        _gamertag = "TestPlayer"
        _player_db_path = None  # type: ignore[assignment]

    return _FakeBatchMixin()


class TestBatchComputeLusr:
    """Tests d'intégration pour batch_compute_lusr (avec fake engine)."""

    def test_force_computes_all_matches(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """En mode force=True, tous les matchs sont insérés depuis zéro."""
        rows = [_match(match_id=f"m{i}", hour=i) for i in range(3)]
        mixin = _build_batch_mixin(in_memory_db, rows)

        count = mixin.batch_compute_lusr(force=True)

        assert count == 3
        stored = in_memory_db.execute(
            "SELECT count(*) FROM match_skill_rank WHERE rating_type = 'LUSR'"
        ).fetchone()[0]
        assert stored == 3

    def test_incremental_skips_already_computed(
        self, in_memory_db: duckdb.DuckDBPyConnection
    ) -> None:
        """En mode force=False, les matchs déjà stockés ne sont pas recalculés."""
        rows = [_match(match_id=f"m{i}", hour=i) for i in range(3)]
        mixin = _build_batch_mixin(in_memory_db, rows)

        # Premier run — insère les 3 matchs
        count1 = mixin.batch_compute_lusr(force=False)
        assert count1 == 3

        # Deuxième run — aucun nouveau match
        count2 = mixin.batch_compute_lusr(force=False)
        assert count2 == 0

    def test_incremental_seeds_from_last_stored_rating(
        self, in_memory_db: duckdb.DuckDBPyConnection
    ) -> None:
        """En incrémental, le nouveau match est seedé depuis le dernier rating stocké.

        Le delta calculé doit correspondre à new_rating - last_stored_rating,
        pas new_rating - INITIAL_MU (comportement buggy).
        """
        rows_batch1 = [_match(match_id="m0", hour=0), _match(match_id="m1", hour=1)]
        mixin1 = _build_batch_mixin(in_memory_db, rows_batch1)
        mixin1.batch_compute_lusr(force=False)

        # Récupérer le rating de m1 (dernier match stocké)
        last_rating = in_memory_db.execute(
            "SELECT rating_value FROM match_skill_rank WHERE match_id = 'm1'"
        ).fetchone()[0]

        # Ajouter m2
        rows_batch2 = [
            _match(match_id="m0", hour=0),
            _match(match_id="m1", hour=1),
            _match(match_id="m2", hour=2, outcome=2, kills=14, kills_expected=10),
        ]
        mixin2 = _build_batch_mixin(in_memory_db, rows_batch2)
        mixin2.batch_compute_lusr(force=False)

        row_m2 = in_memory_db.execute(
            "SELECT rating_value, rating_delta FROM match_skill_rank WHERE match_id = 'm2'"
        ).fetchone()
        assert row_m2 is not None
        stored_rating, stored_delta = row_m2

        # Le delta doit être la différence entre le rating de m2 et celui de m1 (seed correct)
        assert stored_delta is not None
        assert stored_delta == pytest.approx(stored_rating - last_rating, abs=0.1)

    def test_no_matches_returns_zero(self, in_memory_db: duckdb.DuckDBPyConnection) -> None:
        """Sans matchs à calculer, retourne 0."""
        mixin = _build_batch_mixin(in_memory_db, [])
        count = mixin.batch_compute_lusr(force=False)
        assert count == 0

    def test_all_already_computed_returns_zero(
        self, in_memory_db: duckdb.DuckDBPyConnection
    ) -> None:
        """En incrémental, si tous les matchs sont déjà en DB, retourne 0."""
        rows = [_match(match_id="mx", hour=0)]
        mixin = _build_batch_mixin(in_memory_db, rows)
        mixin.batch_compute_lusr(force=False)  # premier run
        count = mixin.batch_compute_lusr(force=False)  # deuxième run
        assert count == 0
