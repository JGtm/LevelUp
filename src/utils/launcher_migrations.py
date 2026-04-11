"""Migrations de schéma et healthcheck pour le lanceur LevelUp.

Contient : ``_run_migrations`` et ``_run_db_healthcheck``, exécutés avant le
lancement de Streamlit pour mettre à jour les schémas DB de manière non-bloquante.
Extrait de launcher.py (F8 post-v7 housekeeping).
"""

from __future__ import annotations

import logging

from src.utils.launcher_i18n import t as _t
from src.utils.launcher_players import _ensure_warehouse_dbs, _list_players
from src.utils.paths import (
    PLAYER_DB_FILENAME,
    PLAYERS_DIR,
    get_metadata_db_path,
    get_pve_db_path,
    get_shared_matches_path,
)

logger = logging.getLogger(__name__)

# _LANG importé via launcher_env pour éviter une double détection
from src.utils.launcher_env import LANG as _LANG  # noqa: E402


def _run_migrations() -> None:  # noqa: C901, PLR0912
    """Applique les migrations de schéma pendantes sur toutes les DB.

    Exécuté avant le lancement de Streamlit. Non-bloquant en cas d'erreur.
    """
    # Streamlit réinitialise son logger lors de son premier import — on le pré-importe
    # et on re-silence avant les imports qui appliquent @st.cache_data.
    try:
        import logging as _lg  # noqa: PLC0415

        import streamlit  # noqa: F401, PLC0415

        _lg.getLogger("streamlit.runtime.caching.cache_data_api").setLevel(_lg.ERROR)
        _lg.getLogger("streamlit.runtime.caching").setLevel(_lg.ERROR)
    except Exception:
        pass

    from src.data.migration.runner import apply_pending_migrations  # noqa: PLC0415

    players = _list_players()
    shared_path = get_shared_matches_path()
    pve_path = get_pve_db_path()

    # Initialiser shared_matches_v2.duckdb si absent (premier lancement avec joueurs)
    if players and not shared_path.exists():
        _ensure_warehouse_dbs()

    if not players and not shared_path.exists():
        return

    print(_t("migrations_checking", _LANG), flush=True)

    total_schemas = 0
    total_backfills = 0
    errors: list[str] = []

    # Migrations shared (une seule fois)
    try:
        meta_path = get_metadata_db_path()
        report = apply_pending_migrations(
            shared_db_path=shared_path if shared_path.exists() else None,
            pve_db_path=pve_path if pve_path.exists() else None,
            metadata_db_path=meta_path if meta_path.exists() else None,
        )
        total_schemas += report.schemas_applied
        total_backfills += report.backfills_applied
        if report.errors:
            errors.extend(report.errors)
    except Exception as e:
        errors.append(f"shared: {e}")

    # Migrations player (pour chaque joueur)
    for player in players:
        db_path = PLAYERS_DIR / player.gamertag / PLAYER_DB_FILENAME
        if not db_path.exists():
            continue
        try:
            report = apply_pending_migrations(player_db_path=db_path)
            total_schemas += report.schemas_applied
            total_backfills += report.backfills_applied
            if report.errors:
                errors.extend(report.errors)
        except Exception as e:
            errors.append(f"{player.gamertag}: {e}")

    if total_schemas == 0 and total_backfills == 0:
        print(_t("migrations_up_to_date", _LANG), flush=True)
    else:
        if total_schemas:
            print(_t("migrations_schemas_applied", _LANG, n=total_schemas), flush=True)
        if total_backfills:
            print(_t("migrations_backfills_applied", _LANG, n=total_backfills), flush=True)

    if errors:
        print(_t("migrations_non_blocking_errors", _LANG, n=len(errors)), flush=True)
        for err in errors[:5]:
            print(f"     - {err}", flush=True)


def _run_db_healthcheck() -> None:
    """Vérifie l'état des DB et vues après les migrations.

    Affiche un résumé rapide. Non-bloquant en cas d'erreur.
    """
    try:
        from src.utils.healthcheck_db import run_healthcheck  # noqa: PLC0415
    except ImportError:
        return

    try:
        results = run_healthcheck(deep=False, auto_repair=True)
    except Exception as e:
        logger.debug("DB healthcheck échoué: %s", e)
        return

    errors = [r for r in results if r.status == "error"]
    warnings = [r for r in results if r.status == "warning"]

    if not errors and not warnings:
        print(_t("healthcheck_ok", _LANG), flush=True)
        return

    for r in warnings:
        for c in r.issues:
            print(f"  ⚠️  {r.db_name}: {c.name} — {c.message}", flush=True)

    for r in errors:
        for c in r.issues:
            print(f"  ❌ {r.db_name}: {c.name} — {c.message}", flush=True)

    if errors:
        print(
            "  💡 python scripts/healthcheck_db.py --verbose  pour un diagnostic complet",
            flush=True,
        )
