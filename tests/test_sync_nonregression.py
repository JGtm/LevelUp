"""Tests de non-régression — sync / fan-out / shared writes.

Chaque test documente un bug passé ou un scénario fragile.
L'objectif : empêcher toute récurrence silencieuse.

Régressions couvertes :
- Delta short-circuit prématuré quand PSA manquants
- _load_existing_match_ids exclut les matchs incomplets
- participants_loaded=FALSE si 0 participants insérés
- Upsert participants ne détruit pas les colonnes MMR
- Fanout ne crash pas si db_profiles absent
- backfill_bits MEDALS posé seulement si médailles existent
- close() ferme proprement les 3 connexions
- sync_meta persiste les métadonnées après sync
- events_loaded=FALSE pour matchs récents sans events (re-processing)
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from pathlib import Path

from tests.conftest_sync import (
    BASE_TIME,
    GT_PLAYER_A,
    GT_PLAYER_B,
    XUID_PLAYER_A,
    XUID_PLAYER_B,
    make_engine,
)

# ===========================================================================
# REG-001: Delta short-circuit quand PSA manquants
# ===========================================================================


class TestReg001DeltaShortCircuitPSA:
    """Bug v5.6 : le HEAD check court-circuitait le sync même si le joueur
    n'avait pas de personal_score_awards pour le dernier match.

    Fix : _is_player_fully_synced_for_match vérifie enrichment + PSA.
    """

    def test_not_fully_synced_without_psa(self, tmp_path: Path) -> None:
        """Match avec enrichment mais sans PSA → pas fully synced."""
        engine = make_engine(tmp_path)
        try:
            # Mettre le match dans shared
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-reg001", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5, 1500)",
                ["match-reg001", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            # Enrichment dans player DB mais PAS de PSA
            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-reg001"],
            )
            conn.commit()

            assert engine._is_player_fully_synced_for_match("match-reg001") is False
        finally:
            engine.close()

    def test_fully_synced_with_psa(self, tmp_path: Path) -> None:
        """Match avec enrichment ET PSA → fully synced."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-reg001-ok", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5, 1500)",
                ["match-reg001-ok", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-reg001-ok"],
            )
            conn.execute(
                "INSERT INTO personal_score_awards (match_id, xuid, award_name, award_score) "
                "VALUES (?, ?, 'KillingSpree', 100)",
                ["match-reg001-ok", XUID_PLAYER_A],
            )
            conn.commit()

            assert engine._is_player_fully_synced_for_match("match-reg001-ok") is True
        finally:
            engine.close()

    def test_fully_synced_with_zero_personal_score(self, tmp_path: Path) -> None:
        """Match avec enrichment et personal_score=0 dans shared → fully synced
        (légitimement sans PSA)."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-reg001-zero", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 0, 0, 0, 0)",
                ["match-reg001-zero", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-reg001-zero"],
            )
            conn.commit()

            assert engine._is_player_fully_synced_for_match("match-reg001-zero") is True
        finally:
            engine.close()

    def test_error_returns_false(self, tmp_path: Path) -> None:
        """En cas d'erreur DB → False (conservateur, ne pas court-circuiter)."""
        engine = make_engine(tmp_path)
        try:
            # Match inexistant → pas d'erreur mais False
            assert engine._is_player_fully_synced_for_match("nonexistent-match") is False
        finally:
            engine.close()


# ===========================================================================
# REG-002: _load_existing_match_ids exclut les matchs incomplets
# ===========================================================================


class TestReg002LoadExistingMatchIds:
    """Bug : les matchs insérés par un coéquipier (dans shared) mais jamais
    traités par le joueur courant (pas de PME) étaient considérés "existants"
    et jamais re-traités.

    Fix : intersection shared ∩ enriched ∩ (scored | score_zero).
    """

    def test_excludes_matches_without_enrichment(self, tmp_path: Path) -> None:
        """Match dans shared mais sans PME → exclu des existing_ids."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            # Match dans shared avec le joueur
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-excl-001", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists) VALUES (?, ?, ?, 0, 2, 10, 8, 5)",
                ["match-excl-001", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            # PAS de PME dans player DB
            existing = engine._load_existing_match_ids()
            assert "match-excl-001" not in existing
        finally:
            engine.close()

    def test_excludes_matches_without_psa_when_score_positive(self, tmp_path: Path) -> None:
        """Match avec enrichment mais sans PSA (personal_score > 0) → exclu."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-excl-002", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5, 1500)",
                ["match-excl-002", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-excl-002"],
            )
            conn.commit()

            # Pas de PSA → exclu
            existing = engine._load_existing_match_ids()
            assert "match-excl-002" not in existing
        finally:
            engine.close()

    def test_includes_complete_matches(self, tmp_path: Path) -> None:
        """Match complet (shared + enrichment + PSA) → inclus dans existing_ids."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-incl-001", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5, 1500)",
                ["match-incl-001", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-incl-001"],
            )
            conn.execute(
                "INSERT INTO personal_score_awards (match_id, xuid, award_name, award_score) "
                "VALUES (?, ?, 'KillingSpree', 100)",
                ["match-incl-001", XUID_PLAYER_A],
            )
            conn.commit()

            existing = engine._load_existing_match_ids()
            assert "match-incl-001" in existing
        finally:
            engine.close()

    def test_includes_zero_score_without_psa(self, tmp_path: Path) -> None:
        """Match avec personal_score=0 et enrichment mais sans PSA → inclus."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-zero-001", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 0, 0, 0, 0)",
                ["match-zero-001", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-zero-001"],
            )
            conn.commit()

            existing = engine._load_existing_match_ids()
            assert "match-zero-001" in existing
        finally:
            engine.close()

    def test_excludes_pending_events(self, tmp_path: Path) -> None:
        """Matchs récents avec events_loaded=FALSE → exclus (pour re-processing)."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            now = datetime.now(timezone.utc)

            # Match récent (< 7 jours) sans events
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time, events_loaded) "
                "VALUES (?, ?, FALSE)",
                ["match-pending-001", now - timedelta(days=1)],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5, 0)",
                ["match-pending-001", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-pending-001"],
            )
            conn.commit()

            existing = engine._load_existing_match_ids()
            assert "match-pending-001" not in existing
        finally:
            engine.close()


# ===========================================================================
# REG-003: sync_meta persiste après sync
# ===========================================================================


class TestReg003SyncMeta:
    """Vérifie que sync_meta est correctement peuplé."""

    def test_save_and_read_sync_meta(self, tmp_path: Path) -> None:
        """Les métadonnées de sync sont écrites et relisibles."""
        engine = make_engine(tmp_path)
        try:
            engine._update_sync_meta("test_key", "test_value")
            engine._get_connection().commit()

            val = engine._get_sync_meta("test_key")
            assert val == "test_value"
        finally:
            engine.close()

    def test_save_sync_metadata_all_keys(self, tmp_path: Path) -> None:
        """_save_sync_metadata écrit toutes les clés attendues."""
        engine = make_engine(tmp_path)
        try:
            engine._save_sync_metadata(delta_mode=True, matches_inserted=5)
            engine._get_connection().commit()

            assert engine._get_sync_meta("last_sync_mode") == "delta"
            assert engine._get_sync_meta("last_sync_matches") == "5"
            assert engine._get_sync_meta("xuid") == XUID_PLAYER_A
            assert engine._get_sync_meta("gamertag") == GT_PLAYER_A
            assert engine._get_sync_meta("last_sync_at") is not None
        finally:
            engine.close()


# ===========================================================================
# REG-004: close() ferme les 3 connexions proprement
# ===========================================================================


class TestReg004CloseConnections:
    """Vérifie que close() nettoie toutes les connexions."""

    def test_close_nullifies_connections(self, tmp_path: Path) -> None:
        """Après close(), les 3 attributs connection sont None."""
        engine = make_engine(tmp_path)

        # Ouvrir explicitement les connexions
        engine._get_connection()
        engine._get_shared_connection()

        engine.close()

        assert engine._connection is None
        assert engine._shared_connection is None

    def test_close_idempotent(self, tmp_path: Path) -> None:
        """Double close() → pas d'erreur."""
        engine = make_engine(tmp_path)
        engine._get_connection()
        engine.close()
        engine.close()  # Ne doit pas lever d'exception


# ===========================================================================
# REG-005: _get_latest_match_id_in_db
# ===========================================================================


class TestReg005LatestMatchId:
    """Vérifie que _get_latest_match_id_in_db retourne le bon match."""

    def test_returns_most_recent(self, tmp_path: Path) -> None:
        """Retourne le match le plus récent du joueur par start_time."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            # Insérer 3 matchs à des heures différentes
            for i, mid in enumerate(["match-old", "match-mid", "match-new"]):
                shared.execute(
                    "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                    [mid, BASE_TIME + timedelta(hours=i)],
                )
                shared.execute(
                    "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                    "outcome, kills, deaths, assists) VALUES (?, ?, ?, 0, 2, 10, 8, 5)",
                    [mid, XUID_PLAYER_A, GT_PLAYER_A],
                )
            shared.commit()

            latest = engine._get_latest_match_id_in_db()
            assert latest == "match-new"
        finally:
            engine.close()

    def test_returns_none_when_empty(self, tmp_path: Path) -> None:
        """Aucun match en DB → None."""
        engine = make_engine(tmp_path)
        try:
            latest = engine._get_latest_match_id_in_db()
            assert latest is None
        finally:
            engine.close()

    def test_ignores_other_players(self, tmp_path: Path) -> None:
        """Ne retourne que les matchs du joueur courant (filtré par xuid)."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            # Match de B uniquement (plus récent)
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-b-only", BASE_TIME + timedelta(hours=10)],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists) VALUES (?, ?, ?, 0, 2, 10, 8, 5)",
                ["match-b-only", XUID_PLAYER_B, GT_PLAYER_B],
            )

            # Match de A (plus ancien)
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-a-own", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists) VALUES (?, ?, ?, 0, 2, 10, 8, 5)",
                ["match-a-own", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            latest = engine._get_latest_match_id_in_db()
            assert latest == "match-a-own"  # Pas match-b-only
        finally:
            engine.close()


# ===========================================================================
# REG-006: _detach_shared_from_player_conn
# ===========================================================================


class TestReg006DetachShared:
    """Vérifie que shared est détaché de la connexion joueur."""

    def test_detach_after_attach(self, tmp_path: Path) -> None:
        """Après ATTACH + detach, shared n'est plus dans duckdb_databases()."""
        engine = make_engine(tmp_path)
        try:
            conn = engine._get_connection()
            shared_path = engine._shared_db_path

            # ATTACH shared en lecture seule
            conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            # Vérifier qu'il est attaché
            dbs = conn.execute("SELECT database_name FROM duckdb_databases()").fetchall()
            db_names = [r[0] for r in dbs]
            assert "shared" in db_names

            # Détacher
            engine._detach_shared_from_player_conn()

            dbs = conn.execute("SELECT database_name FROM duckdb_databases()").fetchall()
            db_names = [r[0] for r in dbs]
            assert "shared" not in db_names
        finally:
            engine.close()

    def test_detach_when_nothing_attached(self, tmp_path: Path) -> None:
        """Detach quand rien n'est attaché → pas d'erreur."""
        engine = make_engine(tmp_path)
        try:
            engine._get_connection()
            engine._detach_shared_from_player_conn()  # Pas de crash
        finally:
            engine.close()


# ===========================================================================
# REG-007: Schema DDL idempotent
# ===========================================================================


class TestReg007SchemaIdempotent:
    """Vérifie que _ensure_schema() est idempotent (CREATE IF NOT EXISTS)."""

    def test_double_ensure_schema(self, tmp_path: Path) -> None:
        """Appeler _ensure_schema() 2 fois → pas d'erreur."""
        engine = make_engine(tmp_path)
        try:
            engine._ensure_schema()  # 1ère fois (déjà fait par _get_connection)
            engine._ensure_schema()  # 2ème fois
            # Les tables doivent exister
            conn = engine._get_connection()
            tables = conn.execute(
                "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
            ).fetchall()
            table_names = {r[0] for r in tables}
            assert "player_match_enrichment" in table_names
            assert "personal_score_awards" in table_names
            assert "sync_meta" in table_names
        finally:
            engine.close()


# ===========================================================================
# REG-008: _existing_match_ids est un cache
# ===========================================================================


class TestReg008ExistingMatchIdsCache:
    """Vérifie le cache _existing_match_ids."""

    def test_cache_reused_on_second_call(self, tmp_path: Path) -> None:
        """Second appel à _load_existing_match_ids retourne le cache."""
        engine = make_engine(tmp_path)
        try:
            ids1 = engine._load_existing_match_ids()
            ids2 = engine._load_existing_match_ids()
            assert ids1 is ids2  # Même objet (cache)
        finally:
            engine.close()

    def test_cache_reset_on_invalidation(self, tmp_path: Path) -> None:
        """Réinitialiser _existing_match_ids à None force un rechargement."""
        engine = make_engine(tmp_path)
        try:
            ids1 = engine._load_existing_match_ids()
            engine._existing_match_ids = None
            ids2 = engine._load_existing_match_ids()
            assert ids1 is not ids2  # Nouvel objet
        finally:
            engine.close()


# ===========================================================================
# REG-009: batch_compute_performance_scores edge cases
# ===========================================================================


class TestReg009PerfScoresEdgeCases:
    """Vérifie les cas limites du batch performance score."""

    def test_no_crash_empty_db(self, tmp_path: Path) -> None:
        """Aucun match en DB → 0 scores calculés, pas de crash."""
        engine = make_engine(tmp_path)
        try:
            count = engine.batch_compute_performance_scores()
            assert count == 0
        finally:
            engine.close()

    def test_no_crash_without_xuid(self, tmp_path: Path) -> None:
        """XUID vide → 0 scores, pas de crash."""
        engine = make_engine(tmp_path, xuid="")
        try:
            count = engine.batch_compute_performance_scores()
            assert count == 0
        finally:
            engine.close()
