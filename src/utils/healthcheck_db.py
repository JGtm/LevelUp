"""Healthcheck des bases DuckDB et vues SQL v6.

Vérifie l'état de santé des DB après déploiement ou au startup :
- Existence des fichiers DB
- Tables attendues présentes
- Vues v6 présentes et interrogeables
- Migrations appliquées vs registre
- metadata.duckdb attachable

Deux modes :
- **quick** (par défaut) : checks structurels rapides (~100ms)
- **deep** : + intégrité référentielle, doublons (~2-5s)

Usage programmatique::

    from src.utils.healthcheck_db import run_healthcheck
    results = run_healthcheck()
    for r in results:
        if r.status != "ok":
            print(r)

Usage CLI : ``python scripts/healthcheck_db.py``
"""

from __future__ import annotations

import contextlib
import logging
from dataclasses import dataclass, field
from pathlib import Path
from typing import Literal

from src.utils._healthcheck_schema import (
    METADATA_EXPECTED_TABLES,
    PLAYER_CRITICAL_COLUMNS,
    PLAYER_EXPECTED_TABLES,
    PVE_EXPECTED_TABLES,
    SHARED_CRITICAL_COLUMNS,
    SHARED_EXPECTED_TABLES,
    SHARED_EXPECTED_VIEWS,
)
from src.utils.db import duckdb_read_only, duckdb_read_write
from src.utils.paths import (
    PLAYERS_DIR,
    get_metadata_db_path,
    get_pve_db_path,
    get_shared_matches_path,
)

logger = logging.getLogger(__name__)

# ─────────────────────────────────────────────────────────────────────────────
# Modèle de résultat
# ─────────────────────────────────────────────────────────────────────────────

Status = Literal["ok", "missing", "broken", "repaired", "warning", "error"]


@dataclass
class CheckDetail:
    """Résultat d'un check individuel."""

    category: str  # "db", "table", "view", "column", "migration", "attach", "integrity"
    name: str
    status: Status
    message: str | None = None


@dataclass
class HealthCheckResult:
    """Résultat du healthcheck pour une DB."""

    db_name: str
    db_path: Path | None = None
    status: Status = "ok"
    size_bytes: int = 0
    checks: list[CheckDetail] = field(default_factory=list)

    def add(self, check: CheckDetail) -> None:
        """Ajoute un check et met à jour le status global."""
        self.checks.append(check)
        if check.status in ("error", "broken") and self.status != "error":
            self.status = "error"
        elif (
            check.status == "repaired"
            and self.status == "ok"
            or check.status in ("missing", "warning")
            and self.status == "ok"
        ):
            self.status = "warning"

    def recompute_status(self) -> None:
        """Recalcule le statut global à partir des checks individuels."""
        self.status = "ok"
        for check in self.checks:
            if check.status in ("error", "broken"):
                self.status = "error"
                return
            if check.status in ("missing", "warning", "repaired"):
                self.status = "warning"

    @property
    def issues(self) -> list[CheckDetail]:
        """Retourne les checks non-OK."""
        return [c for c in self.checks if c.status != "ok"]


# ─────────────────────────────────────────────────────────────────────────────
# Helpers internes
# ─────────────────────────────────────────────────────────────────────────────


def _get_tables(conn) -> set[str]:
    """Retourne les noms de tables dans la DB connectée."""
    rows = conn.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
    ).fetchall()
    return {r[0] for r in rows}


def _get_views(conn) -> set[str]:
    """Retourne les noms de vues dans la DB connectée."""
    rows = conn.execute(
        "SELECT view_name FROM duckdb_views() WHERE schema_name = 'main'"
    ).fetchall()
    return {r[0] for r in rows}


def _get_columns(conn, table: str) -> set[str]:
    """Retourne les noms de colonnes d'une table."""
    rows = conn.execute(
        "SELECT column_name FROM information_schema.columns "
        "WHERE table_name = ? AND table_schema = 'main'",
        [table],
    ).fetchall()
    return {r[0] for r in rows}


def _get_applied_migrations(conn) -> set[str]:
    """Retourne les noms des migrations appliquées."""
    try:
        rows = conn.execute("SELECT name FROM schema_migrations").fetchall()
        return {r[0] for r in rows}
    except Exception:
        return set()


def _get_known_migrations(target_db: str) -> set[str]:
    """Retourne les noms des migrations connues pour un type de DB."""
    # Charger les steps pour peupler le registre
    from src.data.migration import steps  # noqa: F401
    from src.data.migration.registry import MIGRATIONS

    return {m.name for m in MIGRATIONS if m.target_db == target_db}


def _check_view_queryable(conn, view_name: str) -> str | None:
    """Tente un SELECT sur la vue. Retourne le message d'erreur ou None si OK."""
    try:
        conn.execute(f"SELECT 1 FROM {view_name} LIMIT 0")  # noqa: S608
        return None
    except Exception as e:
        return str(e)


# ─────────────────────────────────────────────────────────────────────────────
# Checks par DB
# ─────────────────────────────────────────────────────────────────────────────


def _check_shared(deep: bool = False) -> HealthCheckResult:  # noqa: C901, PLR0912
    """Vérifie shared_matches.duckdb."""
    path = get_shared_matches_path()
    result = HealthCheckResult(db_name="shared_matches", db_path=path)

    if not path.exists():
        result.add(CheckDetail("db", str(path.name), "missing", "Fichier introuvable"))
        return result

    result.size_bytes = path.stat().st_size

    with duckdb_read_only(path) as conn:
        # Attacher metadata pour que les vues qui la référencent soient testables
        meta_path = get_metadata_db_path()
        if meta_path.exists():
            with contextlib.suppress(Exception):
                conn.execute(f"ATTACH '{meta_path}' AS meta (READ_ONLY)")  # noqa: S608

        # Tables
        actual_tables = _get_tables(conn)
        for t in sorted(SHARED_EXPECTED_TABLES):
            if t in actual_tables:
                result.add(CheckDetail("table", t, "ok"))
            else:
                result.add(CheckDetail("table", t, "missing", f"Table {t} absente"))

        # Vues v6
        actual_views = _get_views(conn)
        for v in sorted(SHARED_EXPECTED_VIEWS):
            if v not in actual_views:
                result.add(CheckDetail("view", v, "missing", f"Vue {v} absente"))
            else:
                err = _check_view_queryable(conn, v)
                if err:
                    result.add(CheckDetail("view", v, "broken", f"Vue cassée: {err}"))
                else:
                    result.add(CheckDetail("view", v, "ok"))

        # Colonnes critiques
        for table, expected_cols in SHARED_CRITICAL_COLUMNS.items():
            if table not in actual_tables:
                continue
            actual_cols = _get_columns(conn, table)
            for col in sorted(expected_cols):
                if col not in actual_cols:
                    result.add(
                        CheckDetail("column", f"{table}.{col}", "missing", "Colonne absente")
                    )

        # Migrations
        applied = _get_applied_migrations(conn)
        known = _get_known_migrations("shared")
        pending = known - applied
        if pending:
            result.add(
                CheckDetail(
                    "migration",
                    "pending",
                    "warning",
                    f"{len(pending)} migration(s) en attente: {', '.join(sorted(pending))}",
                )
            )
        else:
            result.add(CheckDetail("migration", "all", "ok", f"{len(applied)} appliquée(s)"))

        # Deep checks
        if deep:
            from src.utils._healthcheck_format import _deep_check_shared

            _deep_check_shared(conn, result)

    return result


def _check_metadata() -> HealthCheckResult:
    """Vérifie metadata.duckdb."""
    path = get_metadata_db_path()
    result = HealthCheckResult(db_name="metadata", db_path=path)

    if not path.exists():
        result.add(
            CheckDetail("db", str(path.name), "warning", "Fichier introuvable (i18n dégradé)")
        )
        return result

    result.size_bytes = path.stat().st_size

    with duckdb_read_only(path) as conn:
        actual_tables = _get_tables(conn)
        for t in sorted(METADATA_EXPECTED_TABLES):
            if t in actual_tables:
                result.add(CheckDetail("table", t, "ok"))
            else:
                result.add(CheckDetail("table", t, "missing", f"Table {t} absente"))

        # Migrations metadata
        applied = _get_applied_migrations(conn)
        known = _get_known_migrations("metadata")
        pending = known - applied
        if pending:
            result.add(
                CheckDetail(
                    "migration",
                    "pending",
                    "warning",
                    f"{len(pending)} migration(s) en attente: {', '.join(sorted(pending))}",
                )
            )
        else:
            result.add(CheckDetail("migration", "all", "ok", f"{len(applied)} appliquée(s)"))

    return result


def _check_metadata_attachable() -> CheckDetail:
    """Vérifie que metadata.duckdb peut être ATTACHé depuis shared."""
    shared_path = get_shared_matches_path()
    meta_path = get_metadata_db_path()

    if not shared_path.exists() or not meta_path.exists():
        return CheckDetail("attach", "metadata→shared", "warning", "DB manquante")

    try:
        with duckdb_read_only(shared_path) as conn:
            conn.execute(f"ATTACH '{meta_path}' AS _hc_meta (READ_ONLY)")
            # Vérifier que asset_translations est accessible
            conn.execute("SELECT 1 FROM _hc_meta.asset_translations LIMIT 0")
            conn.execute("DETACH _hc_meta")
            return CheckDetail("attach", "metadata→shared", "ok")
    except Exception as e:
        return CheckDetail("attach", "metadata→shared", "error", str(e))


def _check_pve() -> HealthCheckResult:
    """Vérifie shared_pve.duckdb."""
    path = get_pve_db_path()
    result = HealthCheckResult(db_name="shared_pve", db_path=path)

    if not path.exists():
        result.add(CheckDetail("db", str(path.name), "ok", "Fichier absent (optionnel)"))
        return result

    result.size_bytes = path.stat().st_size

    with duckdb_read_only(path) as conn:
        actual_tables = _get_tables(conn)
        for t in sorted(PVE_EXPECTED_TABLES):
            if t in actual_tables:
                result.add(CheckDetail("table", t, "ok"))
            else:
                result.add(CheckDetail("table", t, "missing", f"Table {t} absente"))

    return result


def _check_player(gamertag: str) -> HealthCheckResult:
    """Vérifie stats.duckdb d'un joueur."""
    path = PLAYERS_DIR / gamertag / "stats.duckdb"
    result = HealthCheckResult(db_name=f"player:{gamertag}", db_path=path)

    if not path.exists():
        result.add(CheckDetail("db", str(path.name), "missing", "Fichier introuvable"))
        return result

    result.size_bytes = path.stat().st_size

    with duckdb_read_only(path) as conn:
        actual_tables = _get_tables(conn)
        for t in sorted(PLAYER_EXPECTED_TABLES):
            if t in actual_tables:
                result.add(CheckDetail("table", t, "ok"))
            else:
                # Certaines tables sont créées à la demande (media, etc.)
                severity: Status = (
                    "warning" if t in {"media_files", "media_match_associations"} else "missing"
                )
                result.add(CheckDetail("table", t, severity, f"Table {t} absente"))

        # Colonnes critiques
        for table, expected_cols in PLAYER_CRITICAL_COLUMNS.items():
            if table not in actual_tables:
                continue
            actual_cols = _get_columns(conn, table)
            for col in sorted(expected_cols):
                if col not in actual_cols:
                    result.add(
                        CheckDetail("column", f"{table}.{col}", "missing", "Colonne absente")
                    )

        # Migrations player
        applied = _get_applied_migrations(conn)
        known = _get_known_migrations("player")
        pending = known - applied
        if pending:
            result.add(
                CheckDetail(
                    "migration",
                    "pending",
                    "warning",
                    f"{len(pending)} migration(s) en attente: {', '.join(sorted(pending))}",
                )
            )
        else:
            result.add(CheckDetail("migration", "all", "ok", f"{len(applied)} appliquée(s)"))

    return result


def _list_player_gamertags() -> list[str]:
    """Liste les gamertags ayant un dossier dans data/players/."""
    if not PLAYERS_DIR.exists():
        return []
    return sorted(
        d.name for d in PLAYERS_DIR.iterdir() if d.is_dir() and (d / "stats.duckdb").exists()
    )


# ─────────────────────────────────────────────────────────────────────────────
# Auto-repair des vues
# ─────────────────────────────────────────────────────────────────────────────


def _try_repair_views(result: HealthCheckResult) -> list[CheckDetail]:
    """Tente de recréer les vues v6 manquantes ou cassées.

    Retourne la liste des checks mis à jour (repaired ou toujours broken).
    """
    path = get_shared_matches_path()
    if not path.exists():
        return []

    missing_or_broken = [
        c for c in result.checks if c.category == "view" and c.status in ("missing", "broken")
    ]
    if not missing_or_broken:
        return []

    repaired: list[CheckDetail] = []
    try:
        with duckdb_read_write(path) as conn:
            from src.data.sync.migrations import ensure_resolution_views

            ensure_resolution_views(conn)

            # Re-vérifier chaque vue
            actual_views = _get_views(conn)
            for check in missing_or_broken:
                if check.name in actual_views:
                    err = _check_view_queryable(conn, check.name)
                    if err is None:
                        check.status = "repaired"
                        check.message = "Vue recréée avec succès"
                    else:
                        check.status = "broken"
                        check.message = f"Vue recréée mais cassée: {err}"
                repaired.append(check)
    except Exception as e:
        logger.warning("Auto-repair des vues échoué: %s", e)

    return repaired


# ─────────────────────────────────────────────────────────────────────────────
# Point d'entrée principal
# ─────────────────────────────────────────────────────────────────────────────


def run_healthcheck(
    *,
    deep: bool = False,
    auto_repair: bool = True,
    player: str | None = None,
) -> list[HealthCheckResult]:
    """Lance le healthcheck complet sur toutes les DB.

    Args:
        deep: Active les checks profonds (intégrité référentielle, doublons).
        auto_repair: Tente de recréer les vues v6 manquantes.
        player: Si spécifié, ne vérifie que ce joueur (en plus des DB globales).

    Returns:
        Liste de résultats par DB.
    """
    results: list[HealthCheckResult] = []

    # shared_matches
    shared_result = _check_shared(deep=deep)
    if auto_repair:
        _try_repair_views(shared_result)
        shared_result.recompute_status()
    results.append(shared_result)

    # metadata
    results.append(_check_metadata())

    # metadata attachable
    attach_check = _check_metadata_attachable()
    results[1].add(attach_check)

    # shared_pve
    results.append(_check_pve())

    # Joueurs
    if player:
        results.append(_check_player(player))
    else:
        for gt in _list_player_gamertags():
            results.append(_check_player(gt))

    return results


# Re-export depuis le sous-module de formatage
from src.utils._healthcheck_format import (
    _format_size,  # noqa: E402, F401
    format_results,  # noqa: E402, F401
)
