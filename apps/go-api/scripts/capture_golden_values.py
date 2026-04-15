#!/usr/bin/env python3
"""Script de capture des golden values depuis l'API Python LevelUp.

Usage :
    # Démarrer l'API Python en premier :
    cd apps/api && uvicorn app.main:app --port 8000

    # Puis lancer la capture (dans un autre terminal) :
    python apps/go-api/tests/fixtures/golden_values/capture.py

    # Capture d'un endpoint spécifique uniquement :
    python apps/go-api/tests/fixtures/golden_values/capture.py --only health bootstrap

    # Avec un joueur spécifique :
    python apps/go-api/tests/fixtures/golden_values/capture.py --player Chocoboflor

Variables d'environnement :
    LEVELUP_API_URL  : URL de l'API Python (défaut : http://localhost:8000)
    LEVELUP_PLAYER   : Slug du joueur à utiliser (défaut : Chocoboflor)

Description :
    Capture les réponses réelles de l'API Python et les sauvegarde comme golden values.
    Ces fixtures servent d'oracle pour vérifier la parité des endpoints Go.

    Le fichier _meta.source est mis à "captured_live" pour distinguer les fixtures
    schema-conformant des fixtures capturées depuis l'API réelle.
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

try:
    import httpx
except ImportError:
    print("Installer httpx : pip install httpx")
    sys.exit(1)

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

import os

API_URL = os.environ.get("LEVELUP_API_URL", "http://localhost:8000/api/v1")
PLAYER = os.environ.get("LEVELUP_PLAYER", "Chocoboflor")
OUT_DIR = Path(__file__).parent

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def meta(source: str = "captured_live", tolerances: dict | None = None) -> dict:
    """Génère le bloc _meta à injecter dans chaque fixture."""
    return {
        "_meta": {
            "golden_value_version": "live",
            "captured_at": datetime.now(timezone.utc).date().isoformat(),
            "source": source,
            "tolerances": tolerances or {},
        }
    }


def save(filename: str, data: dict[str, Any]) -> None:
    path = OUT_DIR / filename
    with path.open("w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False, default=str)
    print(f"  ✓ {filename}")


def capture_get(client: httpx.Client, path: str) -> dict[str, Any]:
    r = client.get(path)
    r.raise_for_status()
    return r.json()


def capture_post(client: httpx.Client, path: str, body: dict) -> dict[str, Any]:
    r = client.post(path, json=body)
    r.raise_for_status()
    return r.json()


# ---------------------------------------------------------------------------
# Captures par endpoint
# ---------------------------------------------------------------------------


def capture_health(client: httpx.Client) -> None:
    # /health est hors du préfixe /api/v1
    r = client.get(client.base_url.copy_with(path="/health"))
    r.raise_for_status()
    data = r.json()
    save("health_ok.json", {**data, **meta()})


def capture_bootstrap(client: httpx.Client) -> None:
    data = capture_get(client, "/bootstrap")
    save("bootstrap_no_auth.json", {**data, **meta()})


def capture_players(client: httpx.Client) -> None:
    data = capture_get(client, "/players")
    save("players_list.json", {**data, **meta()})


def capture_filters_resolve(client: httpx.Client, player: str) -> None:
    # Cas 1 : tout coché (pas de filtre)
    body: dict[str, Any] = {
        "filter_mode": "period",
        "period": {"start_date": None, "end_date": None},
        "cascade": {},
    }
    data = capture_post(client, f"/players/{player}/filters/resolve", body)
    save(
        "filters_resolve_all.json",
        {**data, **meta(tolerances={"counts": "± selon la DB du moment"})},
    )

    # Cas 2 : filtre qui mène à 0 matchs (playlist inexistante)
    body2: dict[str, Any] = {
        "filter_mode": "period",
        "cascade": {"playlists": ["__nonexistent_playlist__"]},
    }
    data2 = capture_post(client, f"/players/{player}/filters/resolve", body2)
    save(
        "filters_resolve_zero_matches.json",
        {
            **data2,
            **meta(tolerances={"counts.total_matches_after_filters": "0 attendu"}),
        },
    )


def capture_career(client: httpx.Client, player: str) -> None:
    data = capture_get(client, f"/players/{player}/pages/career")
    save(
        f"career_page_{player.lower()}.json",
        {**data, **meta(tolerances={"charts": "null acceptable Sprint 0/1"})},
    )

    data_top = capture_get(client, f"/players/{player}/pages/career/top-matches")
    save(f"career_top_matches_{player.lower()}.json", {**data_top, **meta()})

    data_enc = capture_get(client, f"/players/{player}/pages/career/encounters")
    save(f"career_encounters_{player.lower()}.json", {**data_enc, **meta()})


def capture_match_history(client: httpx.Client, player: str) -> None:
    body: dict[str, Any] = {
        "filters": {"filter_mode": "period"},
        "pagination": {"page": 1, "page_size": 50},
        "include_export_hint": True,
    }
    data = capture_post(client, f"/players/{player}/pages/match-history/query", body)
    save(
        "match_history_page1_nofilter.json",
        {**data, **meta(tolerances={"table.pagination.total": "± stable"})},
    )


def capture_gamertag_search(client: httpx.Client) -> None:
    # Cas 1 : résultat attendu
    data = capture_get(client, "/directory/gamertags/search?q=cho&limit=8")
    save("gamertag_search_cho.json", {**data, **meta()})

    # Cas 2 : aucun résultat
    data2 = capture_get(client, "/directory/gamertags/search?q=zzznotexistent&limit=8")
    save("gamertag_search_empty.json", {**data2, **meta()})


def capture_match_view(client: httpx.Client, player: str) -> None:
    # Récupère le premier match de l'historique pour avoir un match_id réel
    body: dict[str, Any] = {
        "filters": {"filter_mode": "period"},
        "pagination": {"page": 1, "page_size": 1},
    }
    history = capture_post(client, f"/players/{player}/pages/match-history/query", body)
    items = history.get("table", {}).get("items", [])
    if not items:
        print("  ⚠ Aucun match disponible, skip match_view")
        return
    match_id = items[0]["match_id"]
    data = capture_get(client, f"/players/{player}/matches/{match_id}")
    save(
        "match_view_slayer.json",
        {**data, **meta(tolerances={"combat_tab.charts": "[] acceptable Sprint 0/1"})},
    )


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

ALL_CAPTURES = [
    "health",
    "bootstrap",
    "players",
    "filters",
    "career",
    "match_history",
    "gamertag_search",
    "match_view",
]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--only", nargs="*", choices=ALL_CAPTURES)
    parser.add_argument("--player", default=PLAYER)
    args = parser.parse_args()

    targets = set(args.only) if args.only else set(ALL_CAPTURES)
    player = args.player

    print(f"Capture golden values depuis {API_URL} pour joueur '{player}'")

    with httpx.Client(base_url=API_URL, timeout=30.0) as client:
        if "health" in targets:
            print("→ health")
            capture_health(client)
        if "bootstrap" in targets:
            print("→ bootstrap")
            capture_bootstrap(client)
        if "players" in targets:
            print("→ players")
            capture_players(client)
        if "filters" in targets:
            print(f"→ filters ({player})")
            capture_filters_resolve(client, player)
        if "career" in targets:
            print(f"→ career ({player})")
            capture_career(client, player)
        if "match_history" in targets:
            print(f"→ match_history ({player})")
            capture_match_history(client, player)
        if "gamertag_search" in targets:
            print("→ gamertag_search")
            capture_gamertag_search(client)
        if "match_view" in targets:
            print(f"→ match_view ({player})")
            capture_match_view(client, player)

    print("\nCapture terminée.")


if __name__ == "__main__":
    main()
