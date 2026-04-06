"""Logique métier du smoke test post-installation.

Vérifie l'intégrité complète des données après une sync initiale :
- Tables shared_matches_v2.duckdb (match_registry, match_participants, etc.)
- Tables stats.duckdb (player_match_enrichment, match_citations, etc.)
- Cohérence croisée entre les deux bases
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from src.utils.db import duckdb_read_only
from src.utils.paths import get_shared_matches_path

logger = logging.getLogger(__name__)

# Nombre de matchs pour le smoke test
SMOKE_TEST_MATCHES = 20


@dataclass
class TableCheck:
    """Résultat de vérification d'une table."""

    table: str
    db_label: str
    expected: bool  # True si la table DOIT avoir des données
    row_count: int = 0
    ok: bool = False
    detail: str = ""
    warn: bool = False  # Avertissement (pas bloquant)


@dataclass
class SmokeTestResult:
    """Résultat global du smoke test."""

    checks: list[TableCheck] = field(default_factory=list)
    sync_success: bool = False
    sync_message: str = ""
    matches_synced: int = 0
    backfill_done: bool = False
    backfill_message: str = ""

    @property
    def all_ok(self) -> bool:
        """Retourne True si tous les checks obligatoires passent."""
        return self.sync_success and all(c.ok for c in self.checks if c.expected)

    @property
    def passed(self) -> int:
        return sum(1 for c in self.checks if c.ok)

    @property
    def total(self) -> int:
        return len(self.checks)

    @property
    def warnings(self) -> int:
        return sum(1 for c in self.checks if c.warn)


def run_sync_smoke_test(
    gamertag: str,
    db_path: str | Path,
    max_matches: int = SMOKE_TEST_MATCHES,
) -> tuple[bool, str]:
    """Lance la sync initiale pour le smoke test."""
    from src.ui._sync_duckdb_ops import sync_player_duckdb
    from src.ui.cache_loaders import _resolve_player_xuid

    xuid = _resolve_player_xuid(str(db_path))
    return sync_player_duckdb(
        gamertag=gamertag,
        xuid=xuid,
        max_matches=max_matches,
        delta=False,
    )


def run_backfill_smoke_test(gamertag: str) -> tuple[bool, str, dict[str, int]]:
    """Lance le backfill complet pour le smoke test."""
    import asyncio

    from src.data.sync.scope import SyncScope

    scope = SyncScope(
        sessions=True,
        citations=True,
        performance_scores=True,
        killer_victim=True,
        lusr=True,
        csr=True,
        skill_rank=True,
        end_time=True,
    )
    scope.resolve()

    try:
        from scripts.backfill.orchestrator import backfill_player_data

        stats = asyncio.run(backfill_player_data(gamertag, scope=scope))
        total = sum(stats.values())
        return True, f"{total} enrichissements appliqués", stats
    except Exception as e:
        logger.exception("Erreur backfill smoke test")
        return False, f"Erreur backfill: {e}", {}


def verify_data_integrity(
    gamertag: str,
    db_path: str | Path,
) -> SmokeTestResult:
    """Vérifie l'intégrité de toutes les données après sync+backfill."""
    result = SmokeTestResult()
    db_path = Path(db_path)
    shared_path = get_shared_matches_path()

    if shared_path.exists():
        _check_shared_tables(shared_path, result)
    else:
        result.checks.append(
            TableCheck(
                table=shared_path.name,
                db_label="shared",
                expected=True,
                detail="Fichier introuvable",
            )
        )

    if db_path.exists():
        _check_player_tables(db_path, result)
    else:
        result.checks.append(
            TableCheck(
                table="stats.duckdb",
                db_label="player",
                expected=True,
                detail="Fichier introuvable",
            )
        )

    if shared_path.exists() and db_path.exists():
        _check_cross_consistency(shared_path, db_path, result)

    return result


# ── Helpers ─────────────────────────────────────────────────────────────


def _count_rows(conn: Any, table: str) -> int:
    """Compte les lignes d'une table."""
    row = conn.execute(f'SELECT COUNT(*) FROM "{table}"').fetchone()  # noqa: S608
    return row[0] if row else 0


def _get_tables(conn: Any) -> set[str]:
    """Liste les tables d'une connexion DuckDB."""
    return {
        r[0]
        for r in conn.execute(
            "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
        ).fetchall()
    }


@dataclass
class _SimpleCheckParams:
    """Paramètres pour _simple_check."""

    table: str
    db_label: str
    expected: bool
    detail_name: str


def _simple_check(conn: Any, tables: set[str], params: _SimpleCheckParams) -> TableCheck:
    """Check simple : table existe et a des lignes."""
    check = TableCheck(
        table=params.table,
        db_label=params.db_label,
        expected=params.expected,
    )
    if params.table not in tables:
        check.ok = not params.expected
        check.detail = "Table absente" if params.expected else "Table absente (optionnel)"
        check.warn = not params.expected
        return check

    check.row_count = _count_rows(conn, params.table)
    check.ok = check.row_count > 0 if params.expected else True
    if check.row_count == 0 and not params.expected:
        check.detail = f"Aucun(e) {params.detail_name} (optionnel)"
        check.warn = True
    else:
        check.detail = f"{check.row_count} {params.detail_name}"
    return check


# ── Vérifications shared ────────────────────────────────────────────────


def _check_shared_tables(shared_path: Path, result: SmokeTestResult) -> None:
    """Vérifie les tables de shared_matches_v2.duckdb."""
    with duckdb_read_only(shared_path) as conn:
        tables = _get_tables(conn)
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "match_registry",
                    "shared",
                    True,
                    "matchs enregistrés",
                ),
            )
        )
        result.checks.append(_check_match_participants(conn, tables))
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "medals_earned",
                    "shared",
                    True,
                    "médailles",
                ),
            )
        )
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "highlight_events",
                    "shared",
                    True,
                    "événements filmés",
                ),
            )
        )
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "killer_victim_pairs",
                    "shared",
                    True,
                    "paires killer/victim",
                ),
            )
        )
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "xuid_aliases",
                    "shared",
                    True,
                    "alias XUID→gamertag",
                ),
            )
        )


def _check_match_participants(conn: Any, tables: set[str]) -> TableCheck:
    """Vérifie match_participants avec validation colonnes critiques."""
    check = TableCheck(table="match_participants", db_label="shared", expected=True)
    if "match_participants" not in tables:
        check.detail = "Table absente"
        return check

    check.row_count = _count_rows(conn, "match_participants")
    check.ok = check.row_count > 0
    check.detail = f"{check.row_count} lignes de stats"

    if check.ok:
        nulls = conn.execute(
            "SELECT COUNT(*) FROM match_participants WHERE kills IS NULL OR deaths IS NULL"
        ).fetchone()
        null_count = nulls[0] if nulls else 0
        if null_count > 0:
            check.detail += f" ({null_count} avec kills/deaths NULL)"
            check.warn = True
    return check


# ── Vérifications player ────────────────────────────────────────────────


def _check_player_tables(db_path: Path, result: SmokeTestResult) -> None:
    """Vérifie les tables de stats.duckdb (par joueur)."""
    with duckdb_read_only(db_path) as conn:
        tables = _get_tables(conn)
        result.checks.append(_check_enrichment(conn, tables))
        result.checks.append(_check_performance_scores())
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "match_citations",
                    "player",
                    True,
                    "citations",
                ),
            )
        )
        result.checks.append(_check_skill_rank(conn, tables))
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "sessions",
                    "player",
                    True,
                    "sessions",
                ),
            )
        )
        result.checks.append(
            _simple_check(
                conn,
                tables,
                _SimpleCheckParams(
                    "sync_meta",
                    "player",
                    True,
                    "entrées de métadonnées",
                ),
            )
        )


def _check_enrichment(conn: Any, tables: set[str]) -> TableCheck:
    """Vérifie player_match_enrichment avec session_id."""
    check = TableCheck(table="player_match_enrichment", db_label="player", expected=True)
    if "player_match_enrichment" not in tables:
        check.detail = "Table absente"
        return check

    check.row_count = _count_rows(conn, "player_match_enrichment")
    check.ok = check.row_count > 0
    check.detail = f"{check.row_count} matchs enrichis"

    if check.ok:
        sessions = conn.execute(
            "SELECT COUNT(*) FROM player_match_enrichment WHERE session_id IS NOT NULL"
        ).fetchone()
        session_count = sessions[0] if sessions else 0
        if session_count == 0:
            check.detail += " (session_id tous NULL)"
            check.warn = True
        else:
            check.detail += f" ({session_count} avec session)"
    return check


def _check_performance_scores() -> TableCheck:
    """Vérifie les performance_score dans shared.match_participants."""
    check = TableCheck(
        table="performance_score",
        db_label="player (via shared)",
        expected=True,
    )
    shared_path = get_shared_matches_path()
    if not shared_path.exists():
        check.detail = f"{shared_path.name} introuvable"
        check.ok = True
        check.warn = True
        return check

    try:
        with duckdb_read_only(shared_path) as sconn:
            row = sconn.execute(
                "SELECT "
                "  COUNT(*) FILTER (WHERE performance_score IS NOT NULL), "
                "  COUNT(*) "
                "FROM match_participants"
            ).fetchone()
            scored, total_count = (row[0], row[1]) if row else (0, 0)
            check.row_count = scored
            check.ok = True
            if scored == 0 and total_count > 0:
                check.ok = False
                check.detail = f"0/{total_count} — performance_score non calculé"
            elif total_count > 0:
                check.detail = f"{scored}/{total_count} avec score"
            else:
                check.detail = "Aucun participant"
                check.warn = True
    except Exception as e:
        check.detail = f"Erreur: {e}"
        check.warn = True
        check.ok = True
    return check


def _check_skill_rank(conn: Any, tables: set[str]) -> TableCheck:
    """Vérifie match_skill_rank (LUSR/CSR)."""
    check = TableCheck(table="match_skill_rank", db_label="player", expected=True)
    if "match_skill_rank" not in tables:
        check.ok = False
        check.detail = "Table absente — backfill LUSR/CSR non exécuté"
        return check

    check.row_count = _count_rows(conn, "match_skill_rank")
    check.ok = check.row_count > 0
    if check.row_count == 0:
        check.detail = "Aucun rating — backfill LUSR/CSR non exécuté"
    else:
        row = conn.execute(
            "SELECT "
            "  COUNT(*) FILTER (WHERE lusr IS NOT NULL), "
            "  COUNT(*) FILTER (WHERE csr IS NOT NULL) "
            "FROM match_skill_rank"
        ).fetchone()
        lusr_n, csr_n = (row[0], row[1]) if row else (0, 0)
        check.detail = f"{check.row_count} ratings (LUSR: {lusr_n}, CSR: {csr_n})"
    return check


# ── Cohérence croisée ───────────────────────────────────────────────────


def _check_cross_consistency(
    shared_path: Path,
    db_path: Path,
    result: SmokeTestResult,
) -> None:
    """Vérifie la cohérence entre shared et player DB."""
    check = TableCheck(table="cohérence shared↔player", db_label="croisé", expected=True)
    try:
        with duckdb_read_only(shared_path) as sconn:
            r = sconn.execute("SELECT COUNT(DISTINCT match_id) FROM match_participants").fetchone()
            shared_count = r[0] if r else 0

        with duckdb_read_only(db_path) as pconn:
            tables = _get_tables(pconn)
            player_count = (
                _count_rows(pconn, "player_match_enrichment")
                if "player_match_enrichment" in tables
                else 0
            )

        check.ok, check.detail, check.warn = _format_consistency(shared_count, player_count)
    except Exception as e:
        check.detail = f"Erreur de vérification: {e}"
        check.ok = False

    result.checks.append(check)


def _format_consistency(shared_count: int, player_count: int) -> tuple[bool, str, bool]:
    """Formate le résultat de cohérence croisée."""
    if shared_count == 0 and player_count == 0:
        return False, "Aucune donnée des deux côtés", False
    if player_count > 0 and shared_count > 0:
        return (
            True,
            f"shared: {shared_count} matchs uniques, player: {player_count} enrichissements",
            False,
        )
    if player_count == 0 and shared_count > 0:
        return (
            True,
            f"shared: {shared_count} matchs, player: 0 enrichissements (backfill en attente)",
            True,
        )
    return False, "Incohérence détectée", False
