"""Formatage et checks profonds du healthcheck DB.

Sépare les préoccupations d'affichage et d'intégrité de la logique de vérification structurelle.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import duckdb

    from src.utils.healthcheck_db import HealthCheckResult


def _deep_check_shared(conn: duckdb.DuckDBPyConnection, result: HealthCheckResult) -> None:
    """Checks profonds sur shared_matches : intégrité référentielle, doublons."""
    from src.utils.healthcheck_db import CheckDetail as CD

    # Participants orphelins
    try:
        row = conn.execute(
            "SELECT COUNT(*) FROM match_participants mp "
            "WHERE NOT EXISTS (SELECT 1 FROM match_registry mr WHERE mr.match_id = mp.match_id)"
        ).fetchone()
        count = row[0] if row else 0
        if count > 0:
            result.add(CD("integrity", "orphan_participants", "warning", f"{count} orphelin(s)"))
        else:
            result.add(CD("integrity", "orphan_participants", "ok"))
    except Exception as e:
        result.add(CD("integrity", "orphan_participants", "error", str(e)))

    # Médailles orphelines
    try:
        row = conn.execute(
            "SELECT COUNT(*) FROM medals_earned me "
            "WHERE NOT EXISTS (SELECT 1 FROM match_registry mr WHERE mr.match_id = me.match_id)"
        ).fetchone()
        count = row[0] if row else 0
        if count > 0:
            result.add(CD("integrity", "orphan_medals", "warning", f"{count} orphelin(s)"))
        else:
            result.add(CD("integrity", "orphan_medals", "ok"))
    except Exception as e:
        result.add(CD("integrity", "orphan_medals", "error", str(e)))

    # Doublons match_registry
    try:
        row = conn.execute(
            "SELECT COUNT(*) FROM (SELECT match_id FROM match_registry "
            "GROUP BY match_id HAVING COUNT(*) > 1)"
        ).fetchone()
        count = row[0] if row else 0
        if count > 0:
            result.add(CD("integrity", "duplicate_matches", "warning", f"{count} doublon(s)"))
        else:
            result.add(CD("integrity", "duplicate_matches", "ok"))
    except Exception as e:
        result.add(CD("integrity", "duplicate_matches", "error", str(e)))

    # Doublons match_participants
    try:
        row = conn.execute(
            "SELECT COUNT(*) FROM (SELECT match_id, xuid FROM match_participants "
            "GROUP BY match_id, xuid HAVING COUNT(*) > 1)"
        ).fetchone()
        count = row[0] if row else 0
        if count > 0:
            result.add(CD("integrity", "duplicate_participants", "warning", f"{count} doublon(s)"))
        else:
            result.add(CD("integrity", "duplicate_participants", "ok"))
    except Exception as e:
        result.add(CD("integrity", "duplicate_participants", "error", str(e)))


def format_results(results: list[HealthCheckResult], *, verbose: bool = False) -> str:  # noqa: C901, PLR0912
    """Formate les résultats du healthcheck en texte lisible.

    Args:
        results: Résultats de run_healthcheck().
        verbose: Affiche aussi les checks OK.

    Returns:
        Texte formaté multi-lignes.
    """
    lines: list[str] = []
    lines.append("LevelUp — DB Healthcheck")
    lines.append("═" * 50)

    total_warnings = 0
    total_errors = 0

    for r in results:
        size_str = _format_size(r.size_bytes) if r.size_bytes else ""
        status_icon = {"ok": "✅", "warning": "⚠️", "error": "❌"}.get(r.status, "?")
        header = f"{status_icon} {r.db_name}"
        if size_str:
            header += f" ({size_str})"
        lines.append(f"\n{header}")

        if not r.checks:
            continue

        for check in r.checks:
            if check.status == "ok" and not verbose:
                continue

            icon = {
                "ok": "  ✓",
                "missing": "  ✗",
                "broken": "  ✗",
                "repaired": "  ↻",
                "warning": "  ⚠",
                "error": "  ✗",
            }.get(check.status, "  ?")

            msg = f"{icon} [{check.category}] {check.name}"
            if check.message:
                msg += f" — {check.message}"
            lines.append(msg)

            if check.status in ("error", "broken"):
                total_errors += 1
            elif check.status in ("missing", "warning"):
                total_warnings += 1

        if not verbose and not r.issues:
            ok_count = len(r.checks)
            lines.append(f"  {ok_count} check(s) OK")

    lines.append("\n" + "═" * 50)
    if total_errors == 0 and total_warnings == 0:
        lines.append("✅ Toutes les vérifications passent")
    else:
        parts = []
        if total_warnings:
            parts.append(f"{total_warnings} warning(s)")
        if total_errors:
            parts.append(f"{total_errors} erreur(s)")
        lines.append(f"Résultat: {', '.join(parts)}")

    return "\n".join(lines)


def _format_size(size_bytes: int) -> str:
    """Formate une taille en octets en format lisible."""
    if size_bytes < 1024:
        return f"{size_bytes} B"
    if size_bytes < 1024 * 1024:
        return f"{size_bytes / 1024:.0f} KB"
    if size_bytes < 1024 * 1024 * 1024:
        return f"{size_bytes / (1024 * 1024):.1f} MB"
    return f"{size_bytes / (1024 * 1024 * 1024):.1f} GB"
