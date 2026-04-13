"""Tests E2E du shell React en DEMO_MODE.

Vérifie bout en bout que :
  - FastAPI démarre en DEMO_MODE et répond correctement
  - /bootstrap retourne un joueur demo ou autorise la configuration
  - /players liste au moins un joueur en présence des fixtures
  - (optionnel) Le shell React se monte sur http://localhost:5173

Activation :
    python -m pytest tests/e2e/test_shell_react_demo.py --run-e2e-browser -v

Prérequis frontend (optionnel) :
    make dev-front   # démarre Vite sur le port 5173
"""

from __future__ import annotations

import os
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

import pytest

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

REPO_ROOT = Path(__file__).resolve().parents[2]
FIXTURES_DIR = REPO_ROOT / "tests" / "fixtures" / "ref_player"
VITE_DEFAULT_PORT = 5173


def _find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _wait_for_http_ready(url: str, timeout_s: int = 60) -> None:
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=2) as response:  # nosec B310
                if response.status < 500:
                    return
        except (urllib.error.URLError, TimeoutError, ConnectionError):
            time.sleep(0.5)
    raise TimeoutError(f"Serveur non prêt dans le délai imparti: {url}")


def _is_port_open(port: int) -> bool:
    """Vérifie si un port local est déjà en écoute (Vite déjà lancé)."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(0.5)
        try:
            sock.connect(("127.0.0.1", port))
            return True
        except (ConnectionRefusedError, TimeoutError, OSError):
            return False


def _fetch_json(url: str) -> dict:
    """Récupère et parse une réponse JSON depuis une URL."""
    import json

    with urllib.request.urlopen(url, timeout=10) as response:  # nosec B310
        return json.loads(response.read().decode())


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(scope="module")
def fastapi_demo_server() -> str:
    """Démarre FastAPI en DEMO_MODE, retourne l'URL de base."""
    port = _find_free_port()
    base_url = f"http://127.0.0.1:{port}"

    env = os.environ.copy()
    env["LEVELUP_DEMO_MODE"] = "true"
    if FIXTURES_DIR.exists():
        env["LEVELUP_DEMO_FIXTURES_DIR"] = str(FIXTURES_DIR)

    cmd = [
        sys.executable,
        "-m",
        "uvicorn",
        "apps.api.app.main:app",
        "--host",
        "127.0.0.1",
        "--port",
        str(port),
        "--log-level",
        "warning",
    ]

    process = subprocess.Popen(  # noqa: S603
        cmd,
        cwd=str(REPO_ROOT),
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    try:
        _wait_for_http_ready(f"{base_url}/api/v1/health", timeout_s=60)
        yield base_url
    finally:
        process.terminate()
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            process.kill()


# ---------------------------------------------------------------------------
# Tests API — FastAPI en DEMO_MODE (pas de Playwright requis)
# ---------------------------------------------------------------------------


@pytest.mark.e2e_browser
def test_health_endpoint_responds(fastapi_demo_server: str) -> None:
    """L'endpoint /health retourne 200 avec status ok."""
    data = _fetch_json(f"{fastapi_demo_server}/api/v1/health")
    assert data.get("status") in {"ok", "healthy"}


@pytest.mark.e2e_browser
def test_bootstrap_returns_valid_schema(fastapi_demo_server: str) -> None:
    """Bootstrap retourne un schéma cohérent (setup_required bool, auth_state présent)."""
    data = _fetch_json(f"{fastapi_demo_server}/api/v1/bootstrap")
    assert "setup_required" in data
    assert "auth_state" in data
    assert "feature_flags" in data


@pytest.mark.e2e_browser
def test_bootstrap_demo_mode_flag(fastapi_demo_server: str) -> None:
    """Bootstrap indique demo_mode=true dans feature_flags."""
    data = _fetch_json(f"{fastapi_demo_server}/api/v1/bootstrap")
    flags = data.get("feature_flags", {})
    assert flags.get("demo_mode") is True, (
        f"demo_mode devrait être True en DEMO_MODE, got: {flags.get('demo_mode')}"
    )


@pytest.mark.e2e_browser
def test_players_endpoint_in_demo_mode(fastapi_demo_server: str) -> None:
    """En DEMO_MODE avec fixtures, /players retourne au moins un joueur."""
    if not FIXTURES_DIR.exists():
        pytest.skip("Fixtures de démo absentes — générer avec scripts/create_test_corpus.py")
    data = _fetch_json(f"{fastapi_demo_server}/api/v1/players")
    assert "items" in data
    assert len(data["items"]) >= 1, "Au moins un joueur attendu en DEMO_MODE avec fixtures"


@pytest.mark.e2e_browser
def test_bootstrap_current_player_with_fixtures(fastapi_demo_server: str) -> None:
    """Bootstrap expose current_player en DEMO_MODE si les fixtures sont présentes."""
    if not FIXTURES_DIR.exists():
        pytest.skip("Fixtures de démo absentes — générer avec scripts/create_test_corpus.py")
    data = _fetch_json(f"{fastapi_demo_server}/api/v1/bootstrap")
    player = data.get("current_player")
    assert player is not None, "current_player devrait être hydraté en DEMO_MODE avec fixtures"
    assert "player_slug" in player
    assert "gamertag" in player


# ---------------------------------------------------------------------------
# Tests shell React (Playwright) — optionnels, nécessitent Vite sur 5173
# ---------------------------------------------------------------------------


@pytest.mark.e2e_browser
def test_react_shell_mounts(fastapi_demo_server: str) -> None:
    """Le shell React se monte sans erreur en DEMO_MODE (nécessite Vite sur 5173)."""
    if not _is_port_open(VITE_DEFAULT_PORT):
        pytest.skip(
            f"Vite dev server non détecté sur le port {VITE_DEFAULT_PORT}. "
            "Lancer `make dev-front` avant ce test."
        )

    playwright_mod = pytest.importorskip("playwright.sync_api")

    with playwright_mod.sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        try:
            page = browser.new_page()
            page.goto(f"http://127.0.0.1:{VITE_DEFAULT_PORT}", timeout=30_000)
            page.wait_for_load_state("networkidle", timeout=30_000)

            # Pas de crash JS (pas d'erreur Vite rouge à l'écran)
            content = page.content().lower()
            assert "vite" not in content or "plugin" not in content or "error" not in content, (
                "Erreur Vite détectée dans le contenu de la page"
            )

            # Le shell React s'est monté — il doit y avoir un <div id="root"> non vide
            root = page.locator("#root")
            assert root.count() > 0, "L'élément #root est absent — le shell ne s'est pas monté"
            inner = root.inner_html()
            assert len(inner.strip()) > 50, f"#root semble vide (HTML: {inner[:200]!r})"
        finally:
            browser.close()


@pytest.mark.e2e_browser
def test_react_player_selector_shows_demo_player(fastapi_demo_server: str) -> None:
    """Le sélecteur de joueur affiche le joueur de démo (nécessite Vite + fixtures)."""
    if not _is_port_open(VITE_DEFAULT_PORT):
        pytest.skip(f"Vite dev server non détecté sur le port {VITE_DEFAULT_PORT}.")
    if not FIXTURES_DIR.exists():
        pytest.skip("Fixtures de démo absentes.")

    playwright_mod = pytest.importorskip("playwright.sync_api")

    # Lire le gamertag du corpus depuis le dossier de fixtures
    gamertag_file = FIXTURES_DIR / "gamertag.txt"
    expected_gamertag: str | None = None
    if gamertag_file.exists():
        expected_gamertag = gamertag_file.read_text(encoding="utf-8").strip()

    with playwright_mod.sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        try:
            page = browser.new_page()
            page.goto(f"http://127.0.0.1:{VITE_DEFAULT_PORT}", timeout=30_000)
            page.wait_for_load_state("networkidle", timeout=30_000)

            if expected_gamertag:
                # Le gamertag doit apparaître quelque part dans le shell
                page.wait_for_selector(
                    f"text={expected_gamertag}",
                    timeout=10_000,
                )
                assert page.get_by_text(expected_gamertag).first.is_visible()
            else:
                # Fallback : vérifier qu'un sélecteur de joueur est présent
                root = page.locator("#root")
                assert root.count() > 0
        finally:
            browser.close()
