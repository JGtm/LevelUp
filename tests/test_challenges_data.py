"""Tests ciblés pour la persistance DuckDB des défis Halo."""

from __future__ import annotations

from pathlib import Path

import duckdb


def _metadata_path(tmp_path: Path) -> Path:
    return tmp_path / "data" / "warehouse" / "metadata.duckdb"


def _player_db_path(tmp_path: Path) -> Path:
    return tmp_path / "data" / "players" / "TestPlayer" / "stats.duckdb"


def _definition_payload() -> dict:
    return {
        "Category": "Daily",
        "Difficulty": "Normal",
        "ThresholdForSuccess": 1,
        "Reward": {"SoftExperience": 200},
        "SecondaryReward": {"SoftExperience": 50},
        "Title": {
            "value": "Practice Makes Perfection",
            "translations": {
                "en-US": "Practice Makes Perfection",
                "fr-FR": "La pratique fait la perfection",
                "de-DE": "Ubung macht den Meister",
            },
        },
        "Description": {
            "value": "Play a match.",
            "translations": {
                "en-US": "Play a match.",
                "fr-FR": "Disputez une partie.",
                "de-DE": "SchlieBe ein Match ab.",
            },
        },
    }


def test_normalize_challenge_lang_handles_short_and_compact_codes() -> None:
    """Les codes courts et compacts doivent être normalisés en BCP-47."""
    from src.data.challenges import normalize_challenge_lang

    assert normalize_challenge_lang("fr") == "fr-FR"
    assert normalize_challenge_lang("en") == "en-US"
    assert normalize_challenge_lang("frFR") == "fr-FR"


def test_persist_challenge_catalog_stores_all_available_translations(
    tmp_path: Path,
    monkeypatch,
) -> None:
    """Le catalogue doit persister toutes les langues présentes dans le CMS."""
    from src.data.challenges import persist_challenge_catalog

    metadata_path = _metadata_path(tmp_path)
    monkeypatch.setattr("src.data.challenges.get_metadata_db_path", lambda: metadata_path)

    hashes = persist_challenge_catalog(
        {
            "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/PlayNew/d0NPlaySP.json": _definition_payload()
        }
    )

    assert hashes
    with duckdb.connect(str(metadata_path)) as conn:
        langs = conn.execute("SELECT lang FROM challenge_translations ORDER BY lang").fetchall()
        row = conn.execute(
            "SELECT category, difficulty, threshold_for_success, reward_xp, secondary_reward_xp, is_current "
            "FROM challenge_definitions"
        ).fetchone()

    assert langs == [("de-DE",), ("en-US",), ("fr-FR",)]
    assert row == ("Daily", "Normal", 1, 200, 50, True)


def test_load_challenge_metadata_map_falls_back_to_en_us_when_fr_missing(
    tmp_path: Path,
    monkeypatch,
) -> None:
    """La lecture locale doit retomber sur l'anglais si le FR manque."""
    from src.data.challenges import load_challenge_metadata_map, persist_challenge_catalog

    metadata_path = _metadata_path(tmp_path)
    monkeypatch.setattr("src.data.challenges.get_metadata_db_path", lambda: metadata_path)
    payload = _definition_payload()
    payload["Title"] = {
        "value": "Practice Makes Perfection",
        "translations": {"en-US": "Practice Makes Perfection"},
    }
    payload["Description"] = {"value": "Play a match.", "translations": {"en-US": "Play a match."}}

    persist_challenge_catalog(
        {
            "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/PlayNew/d0NPlaySP.json": payload
        }
    )

    metadata = load_challenge_metadata_map(
        ["ChallengeContent/ClientChallengeDefinitions/DailyChallenges/PlayNew/d0NPlaySP.json"],
        lang="fr",
    )

    entry = metadata[
        "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/PlayNew/d0NPlaySP.json"
    ]
    assert entry.title == "Practice Makes Perfection"
    assert entry.description == "Play a match."


def test_persist_challenge_snapshots_deduplicates_same_state(
    tmp_path: Path,
    monkeypatch,
) -> None:
    """Un même état de défi ne doit pas être inséré deux fois de suite."""
    from src.data.challenges import (
        persist_challenge_catalog,
        persist_challenge_snapshots,
    )

    metadata_path = _metadata_path(tmp_path)
    player_db_path = _player_db_path(tmp_path)
    monkeypatch.setattr("src.data.challenges.get_metadata_db_path", lambda: metadata_path)

    definitions = {
        "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/PlayNew/d0NPlaySP.json": _definition_payload()
    }
    hashes = persist_challenge_catalog(definitions)
    decks = {
        "AssignedDecks": [
            {
                "Expiration": {"ISO8601Date": "2026-04-12T18:00:00Z"},
                "ActiveChallenges": [
                    {
                        "Id": "daily-1",
                        "Path": "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/PlayNew/d0NPlaySP.json",
                        "Progress": 0,
                        "CanReroll": True,
                    }
                ],
                "CompletedChallenges": [],
                "UpcomingChallenges": [],
            }
        ]
    }

    inserted_first = persist_challenge_snapshots(
        player_db_path,
        "2535469190789936",
        decks,
        definitions=definitions,
        definition_hashes=hashes,
    )
    inserted_second = persist_challenge_snapshots(
        player_db_path,
        "2535469190789936",
        decks,
        definitions=definitions,
        definition_hashes=hashes,
    )
    decks["AssignedDecks"][0]["ActiveChallenges"][0]["Progress"] = 1
    inserted_third = persist_challenge_snapshots(
        player_db_path,
        "2535469190789936",
        decks,
        definitions=definitions,
        definition_hashes=hashes,
    )

    with duckdb.connect(str(player_db_path)) as conn:
        count = conn.execute("SELECT COUNT(*) FROM challenge_snapshots").fetchone()[0]
        latest = conn.execute(
            "SELECT progress_current, progress_target, xp_reward, status "
            "FROM challenge_snapshots ORDER BY snapshot_at DESC LIMIT 1"
        ).fetchone()

    assert inserted_first == 1
    assert inserted_second == 0
    assert inserted_third == 1
    assert count == 2
    assert latest == (1, 1, 200, "active")
