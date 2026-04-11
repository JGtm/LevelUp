"""Gestion des joueurs et de l'état de la configuration pour le lanceur LevelUp.

Contient : liste des joueurs (PlayerInfo), helpers DB joueur, initialisation
warehouse, chargement .env, vérification token, et la commande ``doctor``.
Extrait de launcher.py (F8 post-v7 housekeeping).
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass
from pathlib import Path

from src.utils.launcher_env import LANG, _import_duckdb
from src.utils.launcher_i18n import t as _t
from src.utils.paths import (
    METADATA_DB_FILENAME,
    PLAYER_DB_FILENAME,
    PLAYERS_DIR,
    WAREHOUSE_DIR,
    get_shared_matches_path,
)

logger = logging.getLogger(__name__)

# =============================================================================
# Modèle PlayerInfo
# =============================================================================


@dataclass
class PlayerInfo:
    """Informations sur un joueur (architecture v5)."""

    gamertag: str
    db_path: Path
    total_matches: int
    xuid: str | None = None


# =============================================================================
# Helpers DB joueur
# =============================================================================


def _get_player_db_path(gamertag: str) -> Path:
    """Retourne le chemin vers stats.duckdb d'un joueur."""
    return PLAYERS_DIR / gamertag / PLAYER_DB_FILENAME


def _player_db_exists(gamertag: str) -> bool:
    """Vérifie si la DB d'un joueur existe."""
    return _get_player_db_path(gamertag).exists()


def _count_matches_duckdb(db_path: Path) -> int:
    """Compte les matchs dans player_match_enrichment (stats.duckdb v5.1+)."""
    if not db_path.exists():
        return 0
    try:
        duckdb = _import_duckdb()
        with duckdb.connect(str(db_path), read_only=True) as con:
            row = con.execute("SELECT COUNT(*) FROM player_match_enrichment").fetchone()
            return row[0] if row else 0
    except Exception:
        return 0


def _display_path(p: Path) -> str:
    """Affiche un chemin relatif au repo."""
    from src.utils.paths import REPO_ROOT  # noqa: PLC0415

    try:
        return str(p.resolve().relative_to(REPO_ROOT))
    except Exception:
        return str(p)


def _metadata_db_exists() -> bool:
    """Vérifie si metadata.duckdb existe."""
    return (WAREHOUSE_DIR / METADATA_DB_FILENAME).exists()


def _list_players() -> list[PlayerInfo]:
    """Liste les joueurs depuis data/players/*/stats.duckdb."""
    players = []

    if not PLAYERS_DIR.exists():
        return players

    duckdb = _import_duckdb()

    for player_dir in sorted(PLAYERS_DIR.iterdir()):
        if not player_dir.is_dir():
            continue

        db_path = player_dir / PLAYER_DB_FILENAME
        if not db_path.exists():
            continue

        gamertag = player_dir.name
        total_matches = 0
        xuid = None
        db_readable = True

        try:
            with duckdb.connect(str(db_path), read_only=True) as con:
                # Architecture v5 : utilise player_match_enrichment si disponible
                try:
                    result = con.execute("SELECT COUNT(*) FROM player_match_enrichment").fetchone()
                    total_matches = result[0] if result else 0
                except Exception:
                    try:
                        result = con.execute("SELECT COUNT(*) FROM match_stats").fetchone()
                        total_matches = result[0] if result else 0
                    except Exception:
                        try:
                            result = con.execute(
                                "SELECT COUNT(*) FROM player_match_stats"
                            ).fetchone()
                            total_matches = result[0] if result else 0
                        except Exception:
                            pass
                # Récupérer le XUID depuis sync_meta si disponible
                try:
                    result = con.execute(
                        "SELECT value FROM sync_meta WHERE key = 'xuid'"
                    ).fetchone()
                    xuid = result[0] if result else None
                except Exception:
                    pass
        except Exception:
            db_readable = False

        players.append(
            PlayerInfo(
                gamertag=gamertag,
                db_path=db_path,
                total_matches=total_matches,
                xuid=xuid,
            )
        )
        if not db_readable:
            print(_t("db_unreadable", LANG, gamertag=gamertag))

    players.sort(key=lambda p: p.total_matches, reverse=True)
    return players


# =============================================================================
# Warehouse + env helpers
# =============================================================================


def _ensure_warehouse_dbs() -> None:
    """Crée data/warehouse/ et initialise shared_matches_v2.duckdb et metadata.duckdb (idempotent)."""
    WAREHOUSE_DIR.mkdir(parents=True, exist_ok=True)

    shared_path = get_shared_matches_path()
    if not shared_path.exists():
        try:
            from src.data.sync._engine_connections import (
                _bootstrap_shared_matches_db,  # noqa: PLC0415
            )

            _bootstrap_shared_matches_db(shared_path)
            print(_t("warehouse_shared_init", LANG), flush=True)
        except Exception as exc:
            print(_t("warehouse_shared_init_fail", LANG, err=exc), flush=True)

    meta_path = WAREHOUSE_DIR / "metadata.duckdb"
    if not meta_path.exists():
        try:
            import duckdb as _duckdb  # noqa: PLC0415

            with _duckdb.connect(str(meta_path)):
                pass
            print(_t("warehouse_meta_init", LANG), flush=True)
        except Exception as exc:
            print(_t("warehouse_meta_init_fail", LANG, err=exc), flush=True)


def _load_dotenv_for_launcher() -> None:
    """Charge .env.local puis .env dans os.environ (ne surcharge pas les vars existantes)."""
    from src.utils.paths import REPO_ROOT  # noqa: PLC0415

    for name in (".env.local", ".env"):
        env_file = REPO_ROOT / name
        if not env_file.exists():
            continue
        try:
            content = env_file.read_text(encoding="utf-8")
        except Exception:
            continue
        for raw_line in content.splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = value


def _env_check_for_player(gamertag: str) -> dict[str, object]:
    """Vérifie la présence d'un token valide pour un gamertag.

    Returns:
        Dict avec les clés ``has_auth`` (bool) et ``player_token_key`` (str).
    """
    _load_dotenv_for_launcher()
    gt_norm = gamertag.upper().replace(" ", "_").replace("-", "_")
    token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"

    has_msal_cache = False
    try:
        from src.auth._msal import (  # noqa: PLC0415
            acquire_token_silent,
            build_msal_app,
            load_msal_cache,
        )

        db_path = _get_player_db_path(gamertag)
        if db_path.exists():
            _cache = load_msal_cache(db_path)
            _app = build_msal_app(_cache)
            has_msal_cache = bool(acquire_token_silent(_app))
    except Exception:
        pass

    has_env_token = bool(os.environ.get(token_key) or os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN"))

    return {
        "client_id": True,
        "player_token": has_msal_cache or has_env_token,
        "player_token_key": token_key,
    }


# =============================================================================
# Commande: doctor
# =============================================================================


def _cmd_doctor(args) -> int:  # noqa: ANN001, ARG001, C901, PLR0912
    """Commande: vérifie la santé de l'environnement."""
    import platform as pf  # noqa: PLC0415
    from importlib import metadata as importlib_metadata  # noqa: PLC0415

    from src.utils.launcher_env import _preferred_python_executable  # noqa: PLC0415

    print("=" * 60)
    print("🩺 LEVELUP — DOCTOR")
    print("=" * 60)

    print(f"\n  OS:     {pf.system()} {pf.release()}")
    print(f"  Python: {__import__('sys').version.split()[0]}")
    print(f"  Exe:    {__import__('sys').executable}")
    import sys  # noqa: PLC0415

    print(f"  Venv:   {'oui' if sys.prefix != getattr(sys, 'base_prefix', sys.prefix) else 'non'}")

    errors: list[str] = []
    warnings: list[str] = []

    from src.utils.paths import REPO_ROOT  # noqa: PLC0415

    expected_venv = (REPO_ROOT / ".venv").resolve()
    if expected_venv.exists():
        _venv_py = _preferred_python_executable()
        if _venv_py is not None:
            exe_r = Path(sys.executable).resolve()
            if exe_r != _venv_py.resolve():
                errors.append(
                    _t("doctor_wrong_interpreter", LANG, exe=exe_r, expected=_venv_py.resolve())
                )
    else:
        errors.append(_t("doctor_no_venv", LANG))

    expected_packages = {
        "duckdb": "1.4.4",
        "polars": "1.38.1",
        "pyarrow": "23.0.0",
        "streamlit": None,
        "spnkr": None,
    }
    for pkg, expected_ver in expected_packages.items():
        try:
            actual = importlib_metadata.version(pkg)
            status = "✓"
            if expected_ver and actual != expected_ver:
                status = "⚠"
                warnings.append(
                    _t(
                        "doctor_pkg_version_mismatch",
                        LANG,
                        pkg=pkg,
                        actual=actual,
                        expected=expected_ver,
                    )
                )
            print(f"  {status} {pkg}=={actual}")
        except importlib_metadata.PackageNotFoundError:
            print(f"  ✗ {pkg} — MANQUANT")
            errors.append(_t("doctor_pkg_missing", LANG, pkg=pkg))

    print()
    players = _list_players()
    if players:
        total = sum(p.total_matches for p in players)
        print(_t("doctor_players_info", LANG, count=len(players), total=total))
    else:
        warnings.append(_t("doctor_no_players", LANG))

    meta_path = WAREHOUSE_DIR / METADATA_DB_FILENAME
    if meta_path.exists():
        print(f"  ✓ metadata.duckdb ({meta_path.stat().st_size / 1024 / 1024:.1f} MB)")
    else:
        warnings.append(_t("doctor_no_metadata", LANG))

    if warnings:
        print(_t("doctor_warnings", LANG))
        for w in warnings:
            print(f"  - {w}")

    if errors:
        print(_t("doctor_errors", LANG))
        for e in errors:
            print(f"  - {e}")
        print(_t("doctor_fix_hint", LANG))
        return 1

    print(_t("doctor_ok", LANG))
    return 0
