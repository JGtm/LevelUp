"""Tests ciblés pour l'enrichissement des défis live de la home V7."""

from __future__ import annotations

from src.ui.pages.home_mission_control_challenges import ActiveChallengeEntry


def test_resolve_localized_value_prefers_requested_language() -> None:
    """La traduction FR doit être préférée quand elle est disponible."""
    from src.ui.pages.home_mission_control_challenges import resolve_localized_value

    payload = {
        "translations": {
            "fr-FR": "La pratique fait la perfection",
            "en-US": "Practice Makes Perfection",
        },
        "value": "Practice Makes Perfection",
    }

    assert resolve_localized_value(payload, "fr") == "La pratique fait la perfection"


def test_resolve_localized_value_falls_back_to_value() -> None:
    """La valeur par défaut doit servir de fallback quand la langue manque."""
    from src.ui.pages.home_mission_control_challenges import resolve_localized_value

    payload = {"translations": {"de-DE": "Nur Deutsch"}, "value": "Fallback EN"}

    assert resolve_localized_value(payload, "en") == "Fallback EN"


def test_build_challenge_summary_from_decks_counts_active_and_completed() -> None:
    """Le résumé doit compter actifs + complétés et dédupliquer les paths."""
    from src.ui.pages.home_mission_control_challenges import build_challenge_summary_from_decks

    payload = {
        "AssignedDecks": [
            {
                "ActiveChallenges": [{"Path": "a.json"}, {"Path": "a.json"}],
                "CompletedChallenges": [{}, {}],
                "Expiration": {"ISO8601Date": "2026-04-12T18:00:00Z"},
            }
        ]
    }

    summary, challenges = build_challenge_summary_from_decks(payload)

    assert summary == {"total": 4, "completed": 2, "next_expiry": "12/04 18:00 UTC"}
    assert challenges == [ActiveChallengeEntry(path="a.json", progress=None)]


def test_build_challenge_summary_from_decks_keeps_progress() -> None:
    """La progression active du joueur doit être conservée pour le défi principal."""
    from src.ui.pages.home_mission_control_challenges import build_challenge_summary_from_decks

    payload = {
        "AssignedDecks": [
            {
                "ActiveChallenges": [{"Path": "a.json", "Progress": 3}],
                "CompletedChallenges": [],
                "Expiration": {"ISO8601Date": "2026-04-12T18:00:00Z"},
            }
        ]
    }

    _summary, challenges = build_challenge_summary_from_decks(payload)

    assert challenges == [ActiveChallengeEntry(path="a.json", progress=3)]


def test_extract_threshold_for_success_handles_int() -> None:
    """Le seuil CMS doit être extrait quand il est déjà scalaire."""
    from src.ui.pages.home_mission_control_challenges import extract_threshold_for_success

    assert extract_threshold_for_success({"ThresholdForSuccess": 5}) == 5


def test_build_challenge_badge_candidates_for_daily_challenge() -> None:
    """Un défi Daily normal doit produire le badge daily-normal en priorité."""
    from src.ui.pages.home_mission_control_challenges import build_challenge_badge_candidates

    stems = build_challenge_badge_candidates(
        "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/PlayNew/d0NPlaySP.json",
        "Daily",
        "normal",
    )

    assert stems[0] == "daily-normal"


def test_build_challenge_badge_candidates_for_weekly_action_challenge() -> None:
    """Un weekly Action heroic doit produire le stem action attendu."""
    from src.ui.pages.home_mission_control_challenges import build_challenge_badge_candidates

    stems = build_challenge_badge_candidates(
        "ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/w1.json",
        "Weekly",
        "Heroic",
    )

    assert "weekly-action-heroic" in stems


def test_build_challenge_badge_candidates_for_capstone_mythic() -> None:
    """Le défi ultime mythic doit retomber sur le badge capstone."""
    from src.ui.pages.home_mission_control_challenges import build_challenge_badge_candidates

    stems = build_challenge_badge_candidates(
        "ChallengeContent/ClientChallengeDefinitions/UltimateChallenges/u1.json",
        "Ultimate",
        "Mythic",
    )

    assert stems[0] == "capstone-mythic"
