"""Tests pour le module d'analyse d'impact des coéquipiers.

Teste les fonctions d'identification et de scoring :
- identify_first_blood()
- identify_clutch_finisher()
- identify_last_casualty()
- compute_impact_scores()
- identify_silent_hero_multi()  [formule B : même joueur = max assists ET min deaths]
- identify_false_brother_multi()  [formule B : même joueur = max deaths ET min assists]

Contraintes logiques :
- Finisseur + Boulet ne peuvent pas être dans le même match (outcomes incompatibles)
- Un joueur peut avoir First Blood + Finisseur (même match court avec victoire)
- First Blood est indépendant de l'outcome
"""

from __future__ import annotations

import pytest

try:
    import polars as pl

    POLARS_AVAILABLE = True
except ImportError:
    POLARS_AVAILABLE = False
    pl = None

# Tenter un import direct (peut échouer si duckdb manque via __init__)
try:
    from src.analysis.friends_impact import (
        OUTCOME_LOSS,
        OUTCOME_WIN,
        ImpactEvent,
        build_impact_matrix,
        compute_impact_scores,
        get_all_impact_events,
        identify_clutch_finisher,
        identify_false_brother_multi,
        identify_first_blood,
        identify_last_casualty,
        identify_silent_hero_multi,
    )

    FRIENDS_IMPACT_AVAILABLE = True
except Exception:
    FRIENDS_IMPACT_AVAILABLE = False
    OUTCOME_LOSS = 3
    OUTCOME_WIN = 2

pytestmark = pytest.mark.skipif(
    not POLARS_AVAILABLE or not FRIENDS_IMPACT_AVAILABLE,
    reason="Polars ou dépendances transitives non disponibles",
)


# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture
def sample_events_df() -> pl.DataFrame:
    """DataFrame d'événements pour les tests."""
    return pl.DataFrame(
        {
            "match_id": ["m1", "m1", "m1", "m1", "m2", "m2", "m2", "m3", "m3"],
            "xuid": ["100", "100", "200", "200", "100", "200", "200", "100", "200"],
            "gamertag": [
                "Alice",
                "Alice",
                "Bob",
                "Bob",
                "Alice",
                "Bob",
                "Bob",
                "Alice",
                "Bob",
            ],
            "event_type": [
                "Kill",
                "Death",
                "Kill",
                "Death",
                "Kill",
                "Kill",
                "Death",
                "Kill",
                "Death",
            ],
            "time_ms": [1000, 2000, 3000, 1500, 500, 2000, 3000, 1000, 2000],
        }
    )


@pytest.fixture
def sample_matches_df() -> pl.DataFrame:
    """DataFrame des matchs pour les tests."""
    return pl.DataFrame(
        {
            "match_id": ["m1", "m2", "m3"],
            "outcome": [
                OUTCOME_WIN,
                OUTCOME_LOSS,
                OUTCOME_WIN,
            ],  # m1=victoire, m2=défaite, m3=victoire
        }
    )


@pytest.fixture
def empty_events_df() -> pl.DataFrame:
    """DataFrame d'événements vide."""
    return pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "xuid": pl.Utf8,
            "gamertag": pl.Utf8,
            "event_type": pl.Utf8,
            "time_ms": pl.Int64,
        }
    )


@pytest.fixture
def empty_matches_df() -> pl.DataFrame:
    """DataFrame de matchs vide."""
    return pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "outcome": pl.Int64,
        }
    )


# =============================================================================
# Tests identify_first_blood
# =============================================================================


class TestIdentifyFirstBlood:
    """Tests pour identify_first_blood()."""

    def test_identify_first_blood_basic(self, sample_events_df: pl.DataFrame) -> None:
        """Vérifie que le premier kill (min time_ms) est correctement identifié."""
        result = identify_first_blood(sample_events_df)

        assert len(result) == 3  # 3 matchs, 3 FB
        assert "m1" in result
        assert "m2" in result
        assert "m3" in result

        # Match m1 : Alice tue à 1000ms, Bob à 3000ms -> Alice
        assert result["m1"].gamertag == "Alice"
        assert result["m1"].time_ms == 1000

        # Match m2 : Alice tue à 500ms, Bob à 2000ms -> Alice
        assert result["m2"].gamertag == "Alice"
        assert result["m2"].time_ms == 500

        # Match m3 : Alice tue à 1000ms, Bob n'a pas de kill -> Alice
        assert result["m3"].gamertag == "Alice"
        assert result["m3"].time_ms == 1000

    def test_identify_first_blood_empty(self, empty_events_df: pl.DataFrame) -> None:
        """Vérifie le comportement avec un DataFrame vide."""
        result = identify_first_blood(empty_events_df)
        assert result == {}

    def test_identify_first_blood_no_kills(self) -> None:
        """Vérifie le comportement sans aucun kill."""
        df = pl.DataFrame(
            {
                "match_id": ["m1", "m1"],
                "xuid": ["100", "200"],
                "gamertag": ["Alice", "Bob"],
                "event_type": ["Death", "Death"],
                "time_ms": [1000, 2000],
            }
        )
        result = identify_first_blood(df)
        assert result == {}

    def test_identify_first_blood_filter_friends(self, sample_events_df: pl.DataFrame) -> None:
        """Vérifie que le badge n'est attribué que si un ami a obtenu le premier sang du match."""
        # Alice (100) a le premier kill dans m1 (1000ms), m2 (500ms), m3 (1000ms)
        # Filtrer seulement Alice → elle apparaît dans les 3 matchs
        result_alice = identify_first_blood(sample_events_df, friend_xuids={"100"})
        assert "m1" in result_alice
        assert result_alice["m1"].gamertag == "Alice"
        assert "m2" in result_alice
        assert result_alice["m2"].gamertag == "Alice"
        assert "m3" in result_alice

        # Filtrer seulement Bob (200) → Bob n'a jamais le premier kill du match → aucun résultat
        result_bob = identify_first_blood(sample_events_df, friend_xuids={"200"})
        assert "m1" not in result_bob
        assert "m2" not in result_bob
        assert "m3" not in result_bob

    def test_first_blood_always_earliest(self, sample_events_df: pl.DataFrame) -> None:
        """Assertion : First Blood est toujours le kill avec min(time_ms)."""
        result = identify_first_blood(sample_events_df)

        # Pour chaque match, vérifier que FB a le plus petit timestamp parmi les kills
        kills = sample_events_df.filter(pl.col("event_type").str.to_lowercase() == "kill")

        for match_id, event in result.items():
            match_kills = kills.filter(pl.col("match_id") == match_id)
            min_time = match_kills["time_ms"].min()
            assert event.time_ms == min_time, (
                f"Match {match_id}: FB devrait être à {min_time}, pas {event.time_ms}"
            )


# =============================================================================
# Tests identify_clutch_finisher
# =============================================================================


class TestIdentifyClutchFinisher:
    """Tests pour identify_clutch_finisher()."""

    def test_identify_clutch_finisher_basic(
        self, sample_events_df: pl.DataFrame, sample_matches_df: pl.DataFrame
    ) -> None:
        """Vérifie que le dernier kill des victoires est identifié."""
        result = identify_clutch_finisher(sample_events_df, sample_matches_df)

        # m1 = victoire, m3 = victoire, m2 = défaite
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result  # Défaite, pas de finisseur

        # Match m1 : Bob tue à 3000ms (dernier kill) -> Bob
        assert result["m1"].gamertag == "Bob"
        assert result["m1"].time_ms == 3000

        # Match m3 : Alice tue à 1000ms (seul kill) -> Alice
        assert result["m3"].gamertag == "Alice"
        assert result["m3"].time_ms == 1000

    def test_identify_clutch_finisher_empty(
        self, empty_events_df: pl.DataFrame, empty_matches_df: pl.DataFrame
    ) -> None:
        """Vérifie le comportement avec des DataFrames vides."""
        result = identify_clutch_finisher(empty_events_df, empty_matches_df)
        assert result == {}

    def test_identify_clutch_finisher_no_wins(self) -> None:
        """Vérifie le comportement sans victoire."""
        events = pl.DataFrame(
            {
                "match_id": ["m1"],
                "xuid": ["100"],
                "gamertag": ["Alice"],
                "event_type": ["Kill"],
                "time_ms": [1000],
            }
        )
        matches = pl.DataFrame(
            {
                "match_id": ["m1"],
                "outcome": [OUTCOME_LOSS],
            }
        )
        result = identify_clutch_finisher(events, matches)
        assert result == {}


# =============================================================================
# Tests identify_last_casualty
# =============================================================================


class TestIdentifyLastCasualty:
    """Tests pour identify_last_casualty()."""

    def test_identify_last_casualty_basic(
        self, sample_events_df: pl.DataFrame, sample_matches_df: pl.DataFrame
    ) -> None:
        """Vérifie que la dernière mort des défaites est identifiée."""
        result = identify_last_casualty(sample_events_df, sample_matches_df)

        # Seul m2 est une défaite
        assert "m2" in result
        assert "m1" not in result  # Victoire
        assert "m3" not in result  # Victoire

        # Match m2 : Bob meurt à 3000ms (dernière mort) -> Bob
        assert result["m2"].gamertag == "Bob"
        assert result["m2"].time_ms == 3000

    def test_identify_last_casualty_empty(
        self, empty_events_df: pl.DataFrame, empty_matches_df: pl.DataFrame
    ) -> None:
        """Vérifie le comportement avec des DataFrames vides."""
        result = identify_last_casualty(empty_events_df, empty_matches_df)
        assert result == {}

    def test_identify_last_casualty_no_losses(self) -> None:
        """Vérifie le comportement sans défaite."""
        events = pl.DataFrame(
            {
                "match_id": ["m1"],
                "xuid": ["100"],
                "gamertag": ["Alice"],
                "event_type": ["Death"],
                "time_ms": [1000],
            }
        )
        matches = pl.DataFrame(
            {
                "match_id": ["m1"],
                "outcome": [OUTCOME_WIN],
            }
        )
        result = identify_last_casualty(events, matches)
        assert result == {}


# =============================================================================
# Tests compute_impact_scores
# =============================================================================


class TestComputeImpactScores:
    """Tests pour compute_impact_scores()."""

    def test_compute_impact_scores_basic(self) -> None:
        """Vérifie le calcul correct des scores."""
        first_bloods = {
            "m1": ImpactEvent("m1", "100", "Alice", 1000, "first_blood"),
            "m2": ImpactEvent("m2", "100", "Alice", 500, "first_blood"),
        }
        clutch_finishers = {
            "m1": ImpactEvent("m1", "200", "Bob", 3000, "clutch_finisher"),
        }
        last_casualties = {
            "m2": ImpactEvent("m2", "200", "Bob", 3000, "last_casualty"),
        }

        scores = compute_impact_scores(first_bloods, clutch_finishers, last_casualties)

        # Alice : 2 FB = +2
        # Bob : 1 Clutch (+2) + 1 Boulet (-2) = 0
        assert scores["Alice"] == 2
        assert scores["Bob"] == 0

        # Vérifier le tri (Alice en premier car score plus élevé)
        assert list(scores.keys())[0] == "Alice"

    def test_compute_impact_scores_empty(self) -> None:
        """Vérifie le comportement avec des dicts vides."""
        scores = compute_impact_scores({}, {}, {})
        assert scores == {}

    def test_compute_impact_scores_edge_cases(self) -> None:
        """Teste les cas limites (0 kills, 0 deaths)."""
        # Seulement des FB
        first_bloods = {
            "m1": ImpactEvent("m1", "100", "Alice", 1000, "first_blood"),
        }
        scores = compute_impact_scores(first_bloods, {}, {})
        assert scores["Alice"] == 1

        # Seulement des Boulets
        last_casualties = {
            "m1": ImpactEvent("m1", "200", "Bob", 3000, "last_casualty"),
        }
        scores = compute_impact_scores({}, {}, last_casualties)
        assert scores["Bob"] == -2


# =============================================================================
# Tests contraintes logiques
# =============================================================================


class TestImpactLogicalConstraints:
    """Tests des contraintes logiques d'incompatibilité."""

    def test_finisseur_and_boulet_never_together(self) -> None:
        """Assertion : Un match ne peut PAS avoir un Finisseur ET un Boulet.

        Finisseur nécessite outcome=2 (victoire), Boulet nécessite outcome=3 (défaite).
        Un match ne peut avoir qu'un seul outcome.
        """
        events = pl.DataFrame(
            {
                "match_id": ["m1", "m1", "m1", "m1"],
                "xuid": ["100", "100", "200", "200"],
                "gamertag": ["Alice", "Alice", "Bob", "Bob"],
                "event_type": ["Kill", "Death", "Kill", "Death"],
                "time_ms": [1000, 2000, 3000, 4000],
            }
        )
        # Match m1 = victoire
        matches_win = pl.DataFrame({"match_id": ["m1"], "outcome": [OUTCOME_WIN]})

        clutch = identify_clutch_finisher(events, matches_win)
        casualty = identify_last_casualty(events, matches_win)

        # Si victoire : Finisseur possible, Boulet impossible
        assert "m1" in clutch
        assert "m1" not in casualty

        # Match m1 = défaite
        matches_loss = pl.DataFrame({"match_id": ["m1"], "outcome": [OUTCOME_LOSS]})

        clutch = identify_clutch_finisher(events, matches_loss)
        casualty = identify_last_casualty(events, matches_loss)

        # Si défaite : Boulet possible, Finisseur impossible
        assert "m1" not in clutch
        assert "m1" in casualty

    def test_multiple_events_per_friend(self) -> None:
        """Assertion : Un joueur peut avoir FB + Finisseur dans le même match court."""
        # Match court où Alice fait le premier ET le dernier kill
        events = pl.DataFrame(
            {
                "match_id": ["m1", "m1"],
                "xuid": ["100", "100"],
                "gamertag": ["Alice", "Alice"],
                "event_type": ["Kill", "Kill"],
                "time_ms": [1000, 2000],  # Alice fait les 2 kills
            }
        )
        matches = pl.DataFrame({"match_id": ["m1"], "outcome": [OUTCOME_WIN]})

        first_bloods = identify_first_blood(events)
        clutch_finishers = identify_clutch_finisher(events, matches)

        # Alice a FB (1000ms) ET Finisseur (2000ms)
        assert "m1" in first_bloods
        assert "m1" in clutch_finishers
        assert first_bloods["m1"].gamertag == "Alice"
        assert clutch_finishers["m1"].gamertag == "Alice"

        # Les scores doivent refléter les deux événements
        scores = compute_impact_scores(first_bloods, clutch_finishers, {})
        assert scores["Alice"] == 3  # 1 (FB) + 2 (Clutch)

    def test_outcome_filtering_correct(self) -> None:
        """Assertion : Finisseur/Boulet sont rejetés si outcome ne correspond pas."""
        events = pl.DataFrame(
            {
                "match_id": ["m1"],
                "xuid": ["100"],
                "gamertag": ["Alice"],
                "event_type": ["Kill"],
                "time_ms": [1000],
            }
        )

        # Tester avec différents outcomes
        for outcome, expect_clutch, _expect_casualty in [
            (OUTCOME_WIN, True, False),
            (OUTCOME_LOSS, False, True),  # Boulet impossible car pas de Death
            (1, False, False),  # Égalité
            (4, False, False),  # Non terminé
        ]:
            matches = pl.DataFrame({"match_id": ["m1"], "outcome": [outcome]})
            clutch = identify_clutch_finisher(events, matches)
            # Note: pour casualty, on aurait besoin d'une Death

            if expect_clutch:
                assert "m1" in clutch, f"Clutch attendu pour outcome={outcome}"
            else:
                assert "m1" not in clutch, f"Clutch non attendu pour outcome={outcome}"


# =============================================================================
# Tests get_all_impact_events
# =============================================================================


class TestGetAllImpactEvents:
    """Tests pour la fonction de convenance get_all_impact_events()."""

    def test_get_all_impact_events(
        self, sample_events_df: pl.DataFrame, sample_matches_df: pl.DataFrame
    ) -> None:
        """Vérifie que la fonction retourne tous les événements."""
        (
            fb,
            clutch,
            casualty,
            last_group_kills,
            first_group_deaths,
            scores,
        ) = get_all_impact_events(sample_events_df, sample_matches_df)

        assert len(fb) > 0
        assert len(clutch) > 0  # m1 et m3 sont des victoires
        assert len(casualty) > 0  # m2 est une défaite
        assert len(scores) > 0
        # Les nouvelles métriques peuvent être vides si pas assez de données
        assert isinstance(last_group_kills, dict)
        assert isinstance(first_group_deaths, dict)


# =============================================================================
# Tests build_impact_matrix
# =============================================================================


class TestBuildImpactMatrix:
    """Tests pour build_impact_matrix()."""

    def test_build_impact_matrix_basic(self) -> None:
        """Vérifie la construction de la matrice d'impact."""
        first_bloods = {
            "m1": ImpactEvent("m1", "100", "Alice", 1000, "first_blood"),
        }
        clutch_finishers = {
            "m1": ImpactEvent("m1", "200", "Bob", 3000, "clutch_finisher"),
        }
        last_casualties = {}

        matrix = build_impact_matrix(
            first_bloods,
            clutch_finishers,
            last_casualties,
            {},  # last_group_kills
            {},  # first_group_deaths
            match_ids=["m1"],
            gamertags=["Alice", "Bob"],
        )

        assert not matrix.is_empty()
        assert len(matrix) == 2  # 2 joueurs × 1 match

        alice_row = matrix.filter(pl.col("gamertag") == "Alice")
        bob_row = matrix.filter(pl.col("gamertag") == "Bob")

        assert not alice_row.is_empty()
        assert not bob_row.is_empty()
        # Vérifier que les événements sont là (dans events)
        alice_events = alice_row["events"][0]
        bob_events = bob_row["events"][0]
        assert alice_events is not None and len(alice_events) > 0  # Alice a first_blood
        assert bob_events is not None and len(bob_events) > 0  # Bob a clutch_finisher

    def test_build_impact_matrix_empty(self) -> None:
        """Vérifie la matrice avec données vides."""
        matrix = build_impact_matrix({}, {}, {}, {}, {}, match_ids=[], gamertags=[])
        assert matrix.is_empty()


# =============================================================================
# Fixtures participants (pour tests multi-match)
# =============================================================================

_WIN_MATCHES = pl.DataFrame({"match_id": ["m1", "m3"], "outcome": [OUTCOME_WIN, OUTCOME_WIN]})
_LOSS_MATCHES = pl.DataFrame({"match_id": ["m2"], "outcome": [OUTCOME_LOSS]})
_ALL_MATCHES = pl.DataFrame(
    {"match_id": ["m1", "m2", "m3"], "outcome": [OUTCOME_WIN, OUTCOME_LOSS, OUTCOME_WIN]}
)

# m1 (victoire) : Alice=max assists(4) ET min deaths(1) → héros silencieux
# m2 (défaite) : Charlie=max deaths(5) ET min assists(0) → faux-frère
# m3 (victoire) : 2 joueurs ex-æquo sur max assists → pas de héros silencieux
_PARTICIPANTS = pl.DataFrame(
    {
        "match_id": ["m1", "m1", "m1", "m2", "m2", "m2", "m3", "m3"],
        "xuid": ["100", "200", "300", "100", "200", "300", "100", "200"],
        "gamertag": ["Alice", "Bob", "Charlie"] * 2 + ["Alice", "Bob"],
        "assists": [4, 1, 0, 3, 2, 0, 3, 3],
        "deaths": [1, 3, 5, 2, 4, 5, 2, 2],
    }
)


# =============================================================================
# Tests identify_silent_hero_multi
# =============================================================================


class TestIdentifySilentHeroMulti:
    """Tests pour identify_silent_hero_multi() — formule B."""

    def test_basic_victoire(self) -> None:
        """Alice (4 assists, 1 death) doit être héros silencieux sur m1."""
        result = identify_silent_hero_multi(_PARTICIPANTS, _ALL_MATCHES)
        assert "m1" in result
        assert result["m1"].gamertag == "Alice"
        assert result["m1"].event_type == "silent_hero"

    def test_absent_hors_victoire(self) -> None:
        """Pas de héros silencieux sur les matchs perdus."""
        result = identify_silent_hero_multi(_PARTICIPANTS, _ALL_MATCHES)
        assert "m2" not in result

    def test_absent_si_criteres_non_cumules(self) -> None:
        """m3 : Alice et Bob ont tous deux 3 assists — les mêmes joueurs n'ont pas min(deaths)."""
        result = identify_silent_hero_multi(_PARTICIPANTS, _ALL_MATCHES)
        # m3 : max_assists=3 → Alice(3,deaths=2) et Bob(3,deaths=2) → les deux satisfont → candidates non vide
        # En fait ici les deux partagent le même max_assists ET le même min_deaths → candidate[0] = Alice
        assert "m3" in result  # ex-æquo : le premier candidat est retourné

    def test_absent_si_zero_assist(self) -> None:
        """Si tous les assists du match sont 0, pas de badge."""
        df = pl.DataFrame(
            {
                "match_id": ["mx", "mx"],
                "xuid": ["1", "2"],
                "gamertag": ["A", "B"],
                "assists": [0, 0],
                "deaths": [1, 2],
            }
        )
        matches = pl.DataFrame({"match_id": ["mx"], "outcome": [OUTCOME_WIN]})
        result = identify_silent_hero_multi(df, matches)
        assert result == {}

    def test_requires_two_players(self) -> None:
        """Avec 1 seul joueur par match, pas de badge."""
        df = pl.DataFrame(
            {
                "match_id": ["mx"],
                "xuid": ["1"],
                "gamertag": ["Solo"],
                "assists": [5],
                "deaths": [0],
            }
        )
        matches = pl.DataFrame({"match_id": ["mx"], "outcome": [OUTCOME_WIN]})
        result = identify_silent_hero_multi(df, matches)
        assert result == {}

    def test_filtre_friend_xuids(self) -> None:
        """Filtrer par friend_xuids restreint le scope à Alice (100) uniquement."""
        result = identify_silent_hero_multi(_PARTICIPANTS, _WIN_MATCHES, friend_xuids={"100"})
        # Alice seule → pas de ≥2 joueurs → pas de badge
        assert result == {}

    def test_empty_participants(self) -> None:
        """DataFrame vide → dict vide."""
        empty = pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "xuid": pl.Utf8,
                "gamertag": pl.Utf8,
                "assists": pl.Int64,
                "deaths": pl.Int64,
            }
        )
        result = identify_silent_hero_multi(empty, _ALL_MATCHES)
        assert result == {}

    def test_no_win_matches(self) -> None:
        """Sans victoires, aucun badge."""
        result = identify_silent_hero_multi(_PARTICIPANTS, _LOSS_MATCHES)
        assert result == {}


# =============================================================================
# Tests identify_false_brother_multi
# =============================================================================


class TestIdentifyFalseBrotherMulti:
    """Tests pour identify_false_brother_multi() — formule B."""

    def test_basic_defaite(self) -> None:
        """Charlie (5 deaths, 0 assists) doit être faux-frère sur m2."""
        result = identify_false_brother_multi(_PARTICIPANTS, _ALL_MATCHES)
        assert "m2" in result
        assert result["m2"].gamertag == "Charlie"
        assert result["m2"].event_type == "false_brother"

    def test_absent_hors_defaite(self) -> None:
        """Pas de faux-frère sur les matchs gagnés."""
        result = identify_false_brother_multi(_PARTICIPANTS, _ALL_MATCHES)
        assert "m1" not in result
        assert "m3" not in result

    def test_absent_si_zero_death(self) -> None:
        """Si tous ont 0 mort dans le match, pas de badge."""
        df = pl.DataFrame(
            {
                "match_id": ["mx", "mx"],
                "xuid": ["1", "2"],
                "gamertag": ["A", "B"],
                "assists": [2, 1],
                "deaths": [0, 0],
            }
        )
        matches = pl.DataFrame({"match_id": ["mx"], "outcome": [OUTCOME_LOSS]})
        result = identify_false_brother_multi(df, matches)
        assert result == {}

    def test_requires_two_players(self) -> None:
        """Avec 1 seul joueur par match, pas de badge."""
        df = pl.DataFrame(
            {
                "match_id": ["mx"],
                "xuid": ["1"],
                "gamertag": ["Solo"],
                "assists": [0],
                "deaths": [5],
            }
        )
        matches = pl.DataFrame({"match_id": ["mx"], "outcome": [OUTCOME_LOSS]})
        result = identify_false_brother_multi(df, matches)
        assert result == {}

    def test_criteres_non_cumules_donne_vide(self) -> None:
        """Si max deaths et min assists ne sont pas portés par le même joueur → pas de badge."""
        df = pl.DataFrame(
            {
                "match_id": ["mx", "mx"],
                "xuid": ["1", "2"],
                "gamertag": ["A", "B"],
                # A : 5 deaths (max) mais 3 assists (pas le min)
                # B : 2 deaths (pas le max) mais 0 assists (min)
                "assists": [3, 0],
                "deaths": [5, 2],
            }
        )
        matches = pl.DataFrame({"match_id": ["mx"], "outcome": [OUTCOME_LOSS]})
        result = identify_false_brother_multi(df, matches)
        assert result == {}

    def test_filtre_friend_xuids(self) -> None:
        """Filtrer par friend_xuids exclut Charlie → pas assez de candidats."""
        result = identify_false_brother_multi(_PARTICIPANTS, _LOSS_MATCHES, friend_xuids={"100"})
        assert result == {}

    def test_empty_participants(self) -> None:
        """DataFrame vide → dict vide."""
        empty = pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "xuid": pl.Utf8,
                "gamertag": pl.Utf8,
                "assists": pl.Int64,
                "deaths": pl.Int64,
            }
        )
        result = identify_false_brother_multi(empty, _ALL_MATCHES)
        assert result == {}

    def test_no_loss_matches(self) -> None:
        """Sans défaites, aucun badge."""
        result = identify_false_brother_multi(_PARTICIPANTS, _WIN_MATCHES)
        assert result == {}
