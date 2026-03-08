"""Page UI du smoke test post-installation.

Affiche un retour visuel en temps réel pendant la sync initiale,
puis vérifie l'intégrité de toutes les données et propose de continuer.

Utilisé comme dernière étape du wizard de configuration.
"""

from __future__ import annotations

import logging
import time
from typing import Any

import streamlit as st

from src.ui.i18n import t

logger = logging.getLogger(__name__)


def render_smoke_test(gamertag: str, db_path: str) -> None:
    """Rend la page complète du smoke test.

    Args:
        gamertag: Gamertag du joueur.
        db_path: Chemin vers stats.duckdb.
    """
    from src.ui.pages.setup_smoke_test_logic import SMOKE_TEST_MATCHES

    st.markdown(f"### 🧪 {t('smoke_title')}")
    st.caption(t("smoke_subtitle", gamertag=gamertag, count=SMOKE_TEST_MATCHES))

    if st.session_state.get("_smoke_test_done"):
        _render_results()
        return

    if st.button(
        t("smoke_start_btn"),
        key="smoke_start",
        type="primary",
        width="stretch",
    ):
        _execute_smoke_test(gamertag, db_path)
    else:
        st.info(t("smoke_info_before_start"))


def _execute_smoke_test(gamertag: str, db_path: str) -> None:
    """Exécute les 3 phases du smoke test avec retour visuel."""
    from src.ui.pages.setup_smoke_test_logic import (
        SMOKE_TEST_MATCHES,
        run_backfill_smoke_test,
        verify_data_integrity,
    )

    sync_success, sync_msg = _run_phase_sync(gamertag, db_path, SMOKE_TEST_MATCHES)
    if not sync_success:
        _save_results(sync_success=False, sync_message=sync_msg)
        _render_results()
        return

    bf_success, bf_msg, bf_stats = _run_phase_backfill(gamertag, run_backfill_smoke_test)

    result = _run_phase_verify(gamertag, db_path, verify_data_integrity)
    result.sync_success = sync_success
    result.sync_message = sync_msg
    result.backfill_done = bf_success
    result.backfill_message = bf_msg

    _save_results(
        sync_success=sync_success,
        sync_message=sync_msg,
        backfill_done=bf_success,
        backfill_message=bf_msg,
        backfill_stats=bf_stats,
        result=result,
    )
    _render_results()


def _run_phase_sync(
    gamertag: str,
    db_path: str,
    max_matches: int,
) -> tuple[bool, str]:
    """Phase 1 : sync avec progress bar."""
    from src.ui.pages.setup_smoke_test_logic import run_sync_smoke_test

    st.markdown(f"#### {t('smoke_phase1_title')}")
    progress = st.progress(0, text=t("smoke_phase1_connecting"))
    progress.progress(10, text=t("smoke_phase1_fetching", count=max_matches))

    start = time.time()
    sync_success, sync_msg = run_sync_smoke_test(gamertag, db_path, max_matches)
    elapsed = time.time() - start

    if sync_success:
        progress.progress(40, text=t("smoke_phase1_done", seconds=f"{elapsed:.1f}"))
        st.success(f"✅ {sync_msg}")
    else:
        progress.progress(40, text=t("smoke_phase1_failed"))
        st.error(f"❌ {sync_msg}")
    return sync_success, sync_msg


def _run_phase_backfill(gamertag: str, backfill_fn: Any) -> tuple[bool, str, dict]:
    """Phase 2 : backfill avec progress bar."""
    st.markdown(f"#### {t('smoke_phase2_title')}")
    progress = st.progress(40, text=t("smoke_phase2_running"))

    bf_success, bf_msg, bf_stats = backfill_fn(gamertag)

    if bf_success:
        progress.progress(70, text=t("smoke_phase2_done"))
        st.success(f"✅ {bf_msg}")
    else:
        progress.progress(70, text=t("smoke_phase2_partial"))
        st.warning(f"⚠️ {bf_msg}")
    return bf_success, bf_msg, bf_stats


def _run_phase_verify(gamertag: str, db_path: str, verify_fn: Any) -> Any:
    """Phase 3 : vérification d'intégrité avec progress bar."""
    st.markdown(f"#### {t('smoke_phase3_title')}")
    progress = st.progress(70, text=t("smoke_phase3_checking"))

    result = verify_fn(gamertag, db_path)
    progress.progress(100, text=t("smoke_phase3_done"))
    return result


def _save_results(  # noqa: PLR0913
    *,
    sync_success: bool,
    sync_message: str = "",
    backfill_done: bool = False,
    backfill_message: str = "",
    backfill_stats: dict | None = None,
    result: Any = None,
) -> None:
    """Sauvegarde les résultats du smoke test en session_state."""
    data: dict[str, Any] = {
        "sync_success": sync_success,
        "sync_message": sync_message,
        "backfill_done": backfill_done,
        "backfill_message": backfill_message,
        "backfill_stats": backfill_stats or {},
    }
    if result is not None:
        data["checks"] = [
            {
                "table": c.table,
                "db_label": c.db_label,
                "expected": c.expected,
                "row_count": c.row_count,
                "ok": c.ok,
                "detail": c.detail,
                "warn": c.warn,
            }
            for c in result.checks
        ]
        data["all_ok"] = result.all_ok
        data["passed"] = result.passed
        data["total"] = result.total
        data["warnings"] = result.warnings

    st.session_state["_smoke_test_done"] = True
    st.session_state["_smoke_test_result"] = data


def _render_results() -> None:
    """Affiche les résultats du smoke test."""
    data = st.session_state.get("_smoke_test_result", {})
    if not data:
        return

    st.divider()
    st.markdown(f"### 📋 {t('smoke_results_title')}")

    if not data.get("sync_success"):
        st.error(t("smoke_result_sync_failed", message=data.get("sync_message", "")))
        _render_retry_button()
        return

    all_ok = data.get("all_ok", False)
    passed = data.get("passed", 0)
    total = data.get("total", 0)
    warnings = data.get("warnings", 0)
    failed = total - passed

    if all_ok and warnings == 0:
        st.success(t("smoke_result_all_ok", passed=passed, total=total))
    elif all_ok:
        st.warning(
            t("smoke_result_ok_with_warnings", passed=passed, total=total, warnings=warnings)
        )
    else:
        st.error(t("smoke_result_issues", failed=failed, total=total))

    checks = data.get("checks", [])
    if checks:
        _render_checks_table(checks)

    _render_actions(all_ok, warnings)


def _render_checks_table(checks: list[dict]) -> None:
    """Affiche le tableau détaillé des vérifications."""
    st.markdown("---")
    for check in checks:
        icon = "✅" if check["ok"] and not check["warn"] else ("⚠️" if check["warn"] else "❌")
        label = f"{icon} **{check['table']}** `{check['db_label']}`"
        detail = check.get("detail", "")
        st.markdown(f"{label} — {detail}")


def _render_actions(all_ok: bool, warnings: int) -> None:
    """Affiche les boutons d'action selon le résultat."""
    st.markdown("---")
    if all_ok:
        st.markdown(f"#### {t('smoke_next_steps')}")
        st.info(t("smoke_next_steps_info"))
        col1, col2 = st.columns(2)
        with col1:
            if st.button(
                t("smoke_btn_full_sync"), key="smoke_continue", type="primary", width="stretch"
            ):
                st.session_state["_setup_smoke_completed"] = True
                st.session_state["_pending_page"] = "settings"
                st.rerun()
        with col2:
            if st.button(t("smoke_btn_dashboard"), key="smoke_dashboard", width="stretch"):
                st.session_state["_setup_smoke_completed"] = True
                st.rerun()
    else:
        _render_retry_button()


def _render_retry_button() -> None:
    """Affiche le bouton de relancement du test."""
    if st.button(t("smoke_btn_retry"), key="smoke_retry", width="stretch"):
        st.session_state.pop("_smoke_test_done", None)
        st.session_state.pop("_smoke_test_result", None)
        st.rerun()
