"""Fixtures partagées pour les tests de parité API."""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parents[2]
FIXTURES_DIR = REPO_ROOT / "tests" / "fixtures" / "ref_player"
GOLDEN_DIR = REPO_ROOT / "tests" / "fixtures" / "golden_values"
SCOPES_DIR = REPO_ROOT / "tests" / "fixtures" / "scopes"


# ---------------------------------------------------------------------------
# Guard : skip si le corpus n'existe pas encore
# ---------------------------------------------------------------------------


def corpus_available() -> bool:
    """Retourne True ssi les 3 fichiers DuckDB de référence existent."""
    return (
        (FIXTURES_DIR / "stats.duckdb").exists()
        and (FIXTURES_DIR / "shared_matches_v2.duckdb").exists()
        and (FIXTURES_DIR / "metadata.duckdb").exists()
    )


requires_corpus = pytest.mark.skipif(
    not corpus_available(),
    reason=(
        "Corpus de référence absent — lance d'abord : "
        "python scripts/create_test_corpus.py --gamertag <Gamertag>"
    ),
)


# ---------------------------------------------------------------------------
# Helpers de chargement
# ---------------------------------------------------------------------------


@dataclass(frozen=True)
class CorpusInfo:
    """Métadonnées du corpus de test."""

    player_slug: str
    gamertag: str
    xuid: str
    stats_db: Path
    shared_db: Path
    metadata_db: Path


def _read_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


@pytest.fixture(scope="session")
def corpus_info() -> CorpusInfo:
    """Retourne les infos du corpus de référence (depuis xuid.txt + golden_values)."""
    if not corpus_available():
        pytest.skip("Corpus absent")

    xuid_file = FIXTURES_DIR / "xuid.txt"
    xuid = xuid_file.read_text(encoding="utf-8").strip() if xuid_file.exists() else ""

    try:
        career_gv = _read_json(GOLDEN_DIR / "career.json")
        gamertag = career_gv.get("gamertag", "demo-player")
    except FileNotFoundError:
        gamertag = "demo-player"

    player_slug = gamertag.lower().replace(" ", "-")

    return CorpusInfo(
        player_slug=player_slug,
        gamertag=gamertag,
        xuid=xuid,
        stats_db=FIXTURES_DIR / "stats.duckdb",
        shared_db=FIXTURES_DIR / "shared_matches_v2.duckdb",
        metadata_db=FIXTURES_DIR / "metadata.duckdb",
    )


@pytest.fixture(scope="session")
def golden_career() -> dict:
    """Golden values pour la page Carrière."""
    path = GOLDEN_DIR / "career.json"
    if not path.exists():
        pytest.skip("Golden values career.json absent")
    return _read_json(path)


@pytest.fixture(scope="session")
def golden_match_history() -> dict:
    """Golden values pour l'historique des matchs (scope complet)."""
    path = GOLDEN_DIR / "match_history_full.json"
    if not path.exists():
        pytest.skip("Golden values match_history_full.json absent")
    return _read_json(path)


@pytest.fixture(scope="session")
def scope_full_period() -> dict:
    """Scope FilterContextInput — période complète."""
    return _read_json(SCOPES_DIR / "full_period.json")


@pytest.fixture(scope="session")
def scope_last_month() -> dict:
    """Scope FilterContextInput — dernier mois."""
    return _read_json(SCOPES_DIR / "last_month.json")


@pytest.fixture(scope="session")
def player_context(corpus_info: CorpusInfo):
    """PlayerContext pointant sur le corpus de référence."""
    from apps.api.app.deps.players import PlayerContext

    return PlayerContext(
        player_slug=corpus_info.player_slug,
        gamertag=corpus_info.gamertag,
        xuid=corpus_info.xuid,
        waypoint_player=corpus_info.gamertag,
        db_path=str(corpus_info.stats_db),
        shared_db_path=str(corpus_info.shared_db),
        metadata_db_path=str(corpus_info.metadata_db),
        is_demo=True,
    )
