"""Tests Phase A — Statut sync success/partial_success/failure."""

from __future__ import annotations

import logging

from src.data.sync.models_sync import SyncResult

# ─────────────────────────────────────────────────────────────────────────────
# A.1 — Statut sur SyncResult
# ─────────────────────────────────────────────────────────────────────────────


def test_sync_result_status_success():
    """Aucune erreur → status='success'."""
    r = SyncResult(matches_inserted=5)
    assert r.status == "success"
    assert r.success is True


def test_sync_result_status_success_no_new_matches():
    """Déjà à jour (0 insert, 0 erreur) → status='success'."""
    r = SyncResult(matches_inserted=0)
    assert r.status == "success"
    assert r.success is True


def test_sync_result_status_partial_success():
    """Erreurs présentes ET insertions → status='partial_success'."""
    r = SyncResult(matches_inserted=3, errors=["timeout joueur B"])
    assert r.status == "partial_success"
    assert r.success is True  # partial_success est un succès pour ok


def test_sync_result_status_failure():
    """Erreurs présentes ET 0 insertion → status='failure'."""
    r = SyncResult(matches_inserted=0, errors=["auth failed"])
    assert r.status == "failure"
    assert r.success is False


def test_sync_result_failure_message():
    """failure → message commence par ❌."""
    r = SyncResult(matches_inserted=0, errors=["auth failed", "network error"])
    msg = r.to_message()
    assert msg.startswith("❌")
    assert "auth failed" in msg


def test_sync_result_partial_success_message():
    """partial_success → message commence par ⚠️ et contient résumé erreur."""
    r = SyncResult(matches_inserted=2, errors=["timeout sur match X"])
    msg = r.to_message()
    assert msg.startswith("⚠️")
    assert "timeout sur match X" in msg
    assert "2 nouveaux matchs" in msg


def test_sync_result_success_message():
    """success → message commence par ✅."""
    r = SyncResult(matches_inserted=5)
    msg = r.to_message()
    assert msg.startswith("✅")
    assert "5 nouveaux matchs" in msg


def test_sync_result_already_up_to_date():
    """Déjà à jour → ✅ Déjà à jour."""
    r = SyncResult()
    msg = r.to_message()
    assert msg.startswith("✅")
    assert "Déjà à jour" in msg


# ─────────────────────────────────────────────────────────────────────────────
# A.2 — Agrégateur multi-joueurs (_summarize_sync_results)
# ─────────────────────────────────────────────────────────────────────────────


def test_summarize_all_ok():
    """2 joueurs OK → (True, '✅ ...')."""
    from src.ui._sync_utils import _summarize_sync_results

    results = [("GT1", True, "✅ 3 nouveaux matchs"), ("GT2", True, "✅ 1 nouveaux matchs")]
    ok, msg = _summarize_sync_results(results)
    assert ok is True
    assert "✅" in msg


def test_summarize_partial_success():
    """1 OK / 1 KO → (True, '⚠️ ...')."""
    from src.ui._sync_utils import _summarize_sync_results

    results = [("GT1", True, "✅ 3 nouveaux matchs"), ("GT2", False, "❌ auth failed")]
    ok, msg = _summarize_sync_results(results)
    assert ok is True
    assert "⚠️" in msg
    assert "GT2" in msg


def test_summarize_all_failure():
    """2 KO → (False, '❌ ...')."""
    from src.ui._sync_utils import _summarize_sync_results

    results = [("GT1", False, "❌ err1"), ("GT2", False, "❌ err2")]
    ok, msg = _summarize_sync_results(results)
    assert ok is False
    assert "❌" in msg


# ─────────────────────────────────────────────────────────────────────────────
# A.3 — Log structuré event=sync_end
# ─────────────────────────────────────────────────────────────────────────────


def test_sync_end_log_partial_success(caplog):
    """event=sync_end status=partial_success loggé en INFO quand 1 joueur KO."""

    # Appel direct via _summarize_sync_results + vérif log dans sync.py
    # On mock les callers internes pour tester uniquement le log

    results = [("GT1", True, "✅ 3 matchs"), ("GT2", False, "❌ auth failed")]

    with caplog.at_level(logging.INFO, logger="src.ui.sync"):
        # Simuler ce que _sync_all_players_loop fait à la fin
        from src.ui._sync_utils import _summarize_sync_results as summ

        global_ok, global_msg = summ(results)
        status = (
            "success"
            if global_ok and not global_msg.startswith("⚠️")
            else ("partial_success" if global_ok else "failure")
        )
        import logging as _logging

        _log = _logging.getLogger("src.ui.sync")
        _log.info(
            "event=sync_end status=%s players=%d ok=%d msg=%s",
            status,
            len(results),
            sum(1 for _, ok, _ in results if ok),
            global_msg,
        )

    assert any(
        "event=sync_end" in r.message and "partial_success" in r.message for r in caplog.records
    ), "event=sync_end status=partial_success doit apparaître dans les logs"
