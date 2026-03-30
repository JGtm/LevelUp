"""Tests unitaires pour _match_impact_events : _find_silent_hero_event et _find_false_brother_event.

Vérifie la logique de sélection des badges stats-only (formule B) :
- Héros silencieux (victoire) = même joueur avec max assists ET min deaths simultanément, ≥1 assist, équipe ≥2 joueurs
- Faux-frère (défaite) = même joueur avec max deaths ET min assists simultanément, ≥1 mort, équipe ≥2 joueurs
"""

from __future__ import annotations

import pytest

try:
    from src.visualization._match_impact_events import (
        _STATS_SENTINEL,
        _find_false_brother_event,
        _find_silent_hero_event,
        compute_single_match_impact,
    )

    AVAILABLE = True
except Exception:
    AVAILABLE = False

pytestmark = pytest.mark.skipif(not AVAILABLE, reason="Module _match_impact_events non disponible")

# =============================================================================
# Fixtures
# =============================================================================

P_ALICE = {"xuid": "100", "gamertag": "Alice", "team_id": 1, "kills": 5, "deaths": 1, "assists": 4}
P_BOB = {"xuid": "200", "gamertag": "Bob", "team_id": 1, "kills": 3, "deaths": 3, "assists": 1}
P_CHARLIE = {
    "xuid": "300",
    "gamertag": "Charlie",
    "team_id": 1,
    "kills": 2,
    "deaths": 5,
    "assists": 0,
}
TEAM = {"100", "200", "300"}


# =============================================================================
# Tests _find_silent_hero_event
# =============================================================================


def test_silent_hero_picks_best_ratio() -> None:
    """Bob (1 assist, 3 morts) doit être choisi : Alice (5 kills = top killer) est exclue."""
    result = _find_silent_hero_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="999")
    assert result is not None
    assert result.event_type == "silent_hero"
    assert result.xuid == "200"
    assert result.gamertag == "Bob"


def test_silent_hero_is_me_flag() -> None:
    """is_me=True quand me_xuid == hero.xuid (Bob, car Alice est exclue comme top killer)."""
    result = _find_silent_hero_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="200")
    assert result is not None
    assert result.is_me is True


def test_silent_hero_time_ms_is_sentinel() -> None:
    """time_ms doit valoir _STATS_SENTINEL (badge sans timestamp)."""
    result = _find_silent_hero_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="999")
    assert result is not None
    assert result.time_ms == _STATS_SENTINEL


def test_silent_hero_extra_label_fr() -> None:
    """extra_label FR doit contenir le nombre d'assists et de morts du héros (Bob : 1 assist, 3 morts)."""
    result = _find_silent_hero_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="999", lang="fr")
    assert result is not None
    assert "1" in result.extra_label  # 1 assist (Bob)
    assert "3" in result.extra_label  # 3 morts (Bob)


def test_silent_hero_extra_label_en() -> None:
    """extra_label EN doit contenir les stats en anglais."""
    result = _find_silent_hero_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="999", lang="en")
    assert result is not None
    assert "ast" in result.extra_label


def test_silent_hero_requires_at_least_one_assist() -> None:
    """Charlie (0 assists) ne peut pas être Héros silencieux même s'il est dans l'équipe."""
    only_charlie = [
        P_CHARLIE,
        {"xuid": "400", "gamertag": "Dave", "team_id": 1, "kills": 1, "deaths": 2, "assists": 0},
    ]
    result = _find_silent_hero_event(only_charlie, {"300", "400"}, me_xuid="999")
    assert result is None


def test_silent_hero_requires_min_two_players() -> None:
    """Retourne None si l'équipe a moins de 2 joueurs."""
    result = _find_silent_hero_event([P_ALICE], {"100"}, me_xuid="999")
    assert result is None


def test_silent_hero_filters_by_team_xuids() -> None:
    """Les joueurs hors team_xuids ne doivent pas être considérés."""
    enemy = {
        "xuid": "999",
        "gamertag": "Enemy",
        "team_id": 2,
        "kills": 0,
        "deaths": 0,
        "assists": 10,
    }
    result = _find_silent_hero_event([P_ALICE, P_BOB, P_CHARLIE, enemy], TEAM, me_xuid="000")
    assert result is not None
    assert result.xuid != "999"


# =============================================================================
# Tests _find_false_brother_event
# =============================================================================


def test_false_brother_picks_worst_ratio() -> None:
    """Charlie (5 deaths - 0 assists = +5) doit être choisi."""
    result = _find_false_brother_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="999")
    assert result is not None
    assert result.event_type == "false_brother"
    assert result.xuid == "300"
    assert result.gamertag == "Charlie"


def test_false_brother_is_me_flag() -> None:
    """is_me=True quand me_xuid == traitor.xuid."""
    result = _find_false_brother_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="300")
    assert result is not None
    assert result.is_me is True


def test_false_brother_time_ms_is_sentinel() -> None:
    """time_ms doit valoir _STATS_SENTINEL."""
    result = _find_false_brother_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="999")
    assert result is not None
    assert result.time_ms == _STATS_SENTINEL


def test_false_brother_extra_label_fr() -> None:
    """extra_label FR doit contenir morts et assists."""
    result = _find_false_brother_event([P_ALICE, P_BOB, P_CHARLIE], TEAM, me_xuid="999", lang="fr")
    assert result is not None
    assert "5" in result.extra_label  # 5 morts
    assert "0" in result.extra_label  # 0 assists


def test_false_brother_requires_at_least_one_death() -> None:
    """Joueur avec 0 mort ne peut pas être faux-frère."""
    no_death = [
        {"xuid": "1", "gamertag": "A", "team_id": 1, "kills": 5, "deaths": 0, "assists": 3},
        {"xuid": "2", "gamertag": "B", "team_id": 1, "kills": 5, "deaths": 0, "assists": 2},
    ]
    result = _find_false_brother_event(no_death, {"1", "2"}, me_xuid="999")
    assert result is None


def test_false_brother_requires_min_two_players() -> None:
    """Retourne None si l'équipe a moins de 2 joueurs."""
    result = _find_false_brother_event([P_CHARLIE], {"300"}, me_xuid="999")
    assert result is None


# =============================================================================
# Tests intégration : compute_single_match_impact
# =============================================================================


def _make_kill(xuid: str, gamertag: str, time_ms: int) -> dict:
    return {"event_type": "kill", "xuid": xuid, "gamertag": gamertag, "time_ms": time_ms}


def _make_death(xuid: str, gamertag: str, time_ms: int) -> dict:
    return {"event_type": "death", "xuid": xuid, "gamertag": gamertag, "time_ms": time_ms}


def test_compute_includes_silent_hero_on_win() -> None:
    """Héros silencieux doit apparaître dans les events en cas de victoire."""
    from src.data.domain.refdata import Outcome

    events = [_make_kill("100", "Alice", 5000), _make_death("200", "Bob", 10000)]
    result = compute_single_match_impact(
        events,
        me_xuid="100",
        outcome=Outcome.WIN,
        team_xuids={"100", "200", "300"},
        participants_stats=[P_ALICE, P_BOB, P_CHARLIE],
    )
    event_types = [e.event_type for e in result]
    assert "silent_hero" in event_types
    assert "false_brother" not in event_types


def test_compute_includes_false_brother_on_loss() -> None:
    """Faux-frère doit apparaître dans les events en cas de défaite."""
    from src.data.domain.refdata import Outcome

    events = [_make_kill("100", "Alice", 5000), _make_death("200", "Bob", 10000)]
    result = compute_single_match_impact(
        events,
        me_xuid="100",
        outcome=Outcome.LOSS,
        team_xuids={"100", "200", "300"},
        participants_stats=[P_ALICE, P_BOB, P_CHARLIE],
    )
    event_types = [e.event_type for e in result]
    assert "false_brother" in event_types
    assert "silent_hero" not in event_types


def test_compute_no_stats_badge_without_participants() -> None:
    """Sans participants_stats, aucun badge stats ne doit apparaître."""
    from src.data.domain.refdata import Outcome

    events = [_make_kill("100", "Alice", 5000)]
    result = compute_single_match_impact(
        events,
        me_xuid="100",
        outcome=Outcome.WIN,
        team_xuids={"100"},
        participants_stats=None,
    )
    event_types = [e.event_type for e in result]
    assert "silent_hero" not in event_types
    assert "false_brother" not in event_types
