"""Mixin — enrichissement fan-out des autres joueurs enregistrés post-sync.

Après la sync d'un joueur (A), met à jour les enrichissements locaux
(performance_score, sessions, citations) des autres joueurs enregistrés
dans db_profiles.json pour les matchs communs nouvellement insérés.

Contrainte d'ordre : appelé APRÈS _detach_shared_from_player_conn()
et _run_lusr_post_sync() pour que shared_matches_v2.duckdb soit libre.
"""

from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import TYPE_CHECKING, Any

import duckdb

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

logger = logging.getLogger(__name__)


class FanoutEnrichmentMixin:
    """Fan-out post-sync : enrichit player_match_enrichment des coéquipiers.

    Après la sync d'un joueur (A), met à jour les enrichissements locaux
    (performance_score, sessions, citations) des autres joueurs enregistrés
    dans db_profiles.json pour les matchs communs nouvellement insérés.
    """

    def _get_other_registered_players(self: _SyncProtocol) -> list[dict[str, Any]]:
        """Retourne les autres joueurs depuis db_profiles.json (hors joueur courant).

        Returns:
            Liste de dict {"gamertag": str, "xuid": str, "db_path": Path}.
        """
        project_root = self._player_db_path.parent.parent.parent
        profiles_path = project_root / "db_profiles.json"
        if not profiles_path.exists():
            logger.debug("fanout: db_profiles.json absent → skip")
            return []
        try:
            data = json.loads(profiles_path.read_text(encoding="utf-8"))
        except Exception as exc:
            logger.warning("fanout: lecture db_profiles.json échouée: %s", exc)
            return []

        current_xuid = str(self._xuid or "").strip()
        current_gt = str(self._gamertag or "").strip().lower()
        others: list[dict[str, Any]] = []

        for gamertag, profile in data.get("profiles", {}).items():
            xuid = str(profile.get("xuid") or "").strip()
            db_path_str = profile.get("db_path") or ""
            if xuid == current_xuid or gamertag.lower() == current_gt:
                continue
            if not db_path_str:
                continue
            others.append(
                {
                    "gamertag": gamertag,
                    "xuid": xuid,
                    "db_path": project_root / db_path_str,
                }
            )

        logger.debug("fanout: %d autre(s) joueur(s) trouvé(s)", len(others))
        return others

    def _find_common_match_ids(
        self: _SyncProtocol,
        xuid: str,
        shared_path: Path,
        new_match_ids: list[str],
    ) -> list[str]:
        """Identifie les matchs parmi new_match_ids où l'autre joueur était présent.

        Args:
            xuid: XUID du coéquipier à tester.
            shared_path: Chemin vers shared_matches_v2.duckdb.
            new_match_ids: IDs des matchs nouvellement insérés.

        Returns:
            Liste des match_ids communs.
        """
        try:
            placeholders = ", ".join(["?" for _ in new_match_ids])
            with duckdb.connect(str(shared_path), read_only=True) as conn:
                rows = conn.execute(
                    f"SELECT DISTINCT match_id FROM match_participants "
                    f"WHERE xuid = ? AND match_id IN ({placeholders})",
                    [xuid, *new_match_ids],
                ).fetchall()
            return [str(r[0]) for r in rows if r[0]]
        except Exception as exc:
            logger.warning("fanout: lecture shared pour xuid=%s échouée: %s", xuid, exc)
            return []

    def _enrich_single_player(  # noqa: PLR0913
        self: _SyncProtocol,
        gamertag: str,
        xuid: str,
        player_db_path: Path,
        new_match_ids: list[str],
    ) -> None:
        """Enrichit les données d'un coéquipier pour les matchs communs.

        Args:
            gamertag: Gamertag du coéquipier.
            xuid: XUID du coéquipier.
            player_db_path: Chemin vers stats.duckdb du coéquipier.
            new_match_ids: IDs des matchs nouvellement insérés par le joueur courant.
        """
        if not player_db_path.exists():
            logger.debug("fanout: DB absente pour %s → skip", gamertag)
            return
        if not xuid:
            logger.warning("fanout: XUID manquant pour %s → skip", gamertag)
            return

        from src.utils.paths import get_shared_matches_path_from_player

        shared_path = get_shared_matches_path_from_player(player_db_path)
        if not shared_path or not shared_path.exists():
            logger.debug("fanout: shared absent pour %s → skip", gamertag)
            return

        common_ids = self._find_common_match_ids(xuid, shared_path, new_match_ids)
        if not common_ids:
            logger.debug("fanout: aucun match commun pour %s", gamertag)
            return

        logger.info(
            "fanout [%s]: %d match(s) commun(s) → enrichissement", gamertag, len(common_ids)
        )
        self._run_other_player_enrichment(gamertag, xuid, player_db_path, shared_path)

    def _run_other_player_enrichment(
        self: _SyncProtocol,
        gamertag: str,
        xuid: str,
        player_db_path: Path,
        shared_path: Path,
    ) -> None:
        """Lance PSA + perf_scores + sessions + citations pour un autre joueur.

        Crée un DuckDBSyncEngine minimal (sans tokens API) pour réutiliser
        batch_compute_performance_scores, puis ferme la connexion shared avant
        d'appeler sessions et citations (même pattern que _run_post_sync_compute).
        """
        from src.data.sync.engine import DuckDBSyncEngine

        engine = DuckDBSyncEngine(
            player_db_path=player_db_path,
            xuid=xuid,
            gamertag=gamertag,
            shared_db_path=shared_path,
            shared_read_only=True,  # Fanout ne fait que lire shared (J.4)
        )
        try:
            # Écrire les PSA collectées pendant le traitement des matchs du joueur courant.
            # stats_json contient les PSA de TOUS les participants : on les extrait
            # côté expéditeur (self) et on les distribue ici vers chaque DB coéquipier.
            pending_psa = getattr(self, "_pending_other_psa", {}).get(xuid, [])
            if pending_psa:
                try:
                    engine._insert_personal_score_rows(pending_psa)
                    engine._get_connection().commit()
                    logger.info(
                        "fanout [%s]: %d PSA insérés (collectés depuis stats_json)",
                        gamertag,
                        len(pending_psa),
                    )
                except Exception as exc:
                    logger.warning("fanout [%s]: insertion PSA échouée: %s", gamertag, exc)

            perf_count = engine.batch_compute_performance_scores()
            logger.info("fanout [%s]: %d performance_score(s) calculé(s)", gamertag, perf_count)

            # Libérer shared avant sessions/citations (évite le conflit R/W + ATTACH R/O)
            if engine._shared_connection is not None:
                try:
                    engine._shared_connection.close()
                except Exception as e:
                    logger.debug("event=shared_conn_close_failed step=fanout error=%s", e)
                engine._shared_connection = None

            player_conn = engine._get_connection()
            self._run_other_sessions(gamertag, xuid, player_db_path, player_conn)
            self._run_other_citations(gamertag, xuid, player_db_path, player_conn)
            self._run_other_dominance(gamertag, xuid, shared_path, player_conn)
            lusr_count = engine.batch_compute_lusr(force=False)
            if lusr_count > 0:
                logger.info("fanout [%s]: %d LUSR calculé(s)", gamertag, lusr_count)
        finally:
            engine.close()

    def _run_other_sessions(
        self: _SyncProtocol,
        gamertag: str,
        xuid: str,
        db_path: Path,
        conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """Met à jour les sessions pour un autre joueur."""
        try:
            from src.data.sessions_backfill import backfill_sessions_for_player

            result = backfill_sessions_for_player(db_path=db_path, xuid=xuid, conn=conn)
            logger.info(
                "fanout [%s]: sessions — %d créée(s), %d mise(s) à jour",
                gamertag,
                result.get("sessions_created", 0),
                result.get("sessions_updated", 0),
            )
        except Exception as exc:
            logger.warning("fanout [%s]: sessions échoué (non bloquant): %s", gamertag, exc)

    def _run_other_citations(
        self: _SyncProtocol,
        gamertag: str,
        xuid: str,
        db_path: Path,
        conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """Met à jour les citations pour un autre joueur."""
        try:
            from src.data.citations_backfill import backfill_citations_for_player

            result = backfill_citations_for_player(db_path=db_path, xuid=xuid, conn=conn)
            logger.info(
                "fanout [%s]: %d citation(s) calculée(s)",
                gamertag,
                result.get("citations_computed", 0),
            )
        except Exception as exc:
            logger.warning("fanout [%s]: citations échoué (non bloquant): %s", gamertag, exc)

    def _run_other_dominance(
        self: _SyncProtocol,
        gamertag: str,
        xuid: str,
        shared_path: Path,
        conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """Calcule les dominance flags pour un autre joueur (fanout).

        Ouvre shared en R/O pour éviter tout conflit avec le dashboard ouvert.
        """
        try:
            from src.data.dominance_backfill import compute_dominance_for_player

            with duckdb.connect(str(shared_path), read_only=True) as shared_ro:
                result = compute_dominance_for_player(conn, shared_ro, xuid)
            logger.info(
                "fanout [%s]: dominance — %d traité(s)",
                gamertag,
                result.get("processed", 0),
            )
        except Exception as exc:
            logger.warning("fanout [%s]: dominance échoué (non bloquant): %s", gamertag, exc)

    def _enrich_other_registered_players(
        self: _SyncProtocol,
        new_match_ids: list[str],
    ) -> None:
        """Fan-out vers tous les autres joueurs enregistrés pour les nouveaux matchs.

        Chaque joueur est traité indépendamment — une erreur sur un joueur
        ne bloque pas les autres ni ne fait échouer le sync principal.

        Args:
            new_match_ids: IDs des matchs nouvellement insérés.
        """
        if not new_match_ids:
            return

        others = self._get_other_registered_players()
        if not others:
            logger.debug("fanout: aucun autre joueur enregistré → skip")
            return

        logger.info(
            "fanout: %d nouveaux match(s) → enrichissement de %d autre(s) joueur(s)",
            len(new_match_ids),
            len(others),
        )
        for player in others:
            try:
                self._enrich_single_player(
                    player["gamertag"],
                    player["xuid"],
                    player["db_path"],
                    new_match_ids,
                )
            except Exception as exc:
                logger.warning("fanout [%s]: erreur non bloquante: %s", player["gamertag"], exc)

    def _has_missing_enrichment(
        self: _SyncProtocol,
        xuid: str,
        player_db_path: Path,
    ) -> bool:
        """Retourne True si le joueur a des matchs sans performance_score dans PME."""
        from src.utils.paths import get_shared_matches_path_from_player

        shared_path = get_shared_matches_path_from_player(player_db_path)
        if not shared_path or not shared_path.exists():
            return False
        try:
            with duckdb.connect(str(player_db_path), read_only=True) as pconn:
                pconn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
                row = pconn.execute(
                    "SELECT COUNT(*) FROM shared.match_participants mp "
                    "WHERE mp.xuid = ? AND NOT EXISTS ("
                    "  SELECT 1 FROM player_match_enrichment pme "
                    "  WHERE pme.match_id = mp.match_id AND pme.performance_score IS NOT NULL"
                    ")",
                    [xuid],
                ).fetchone()
            return (row[0] if row else 0) > 0
        except Exception as exc:
            logger.debug("fanout repair: check PME échoué pour xuid=%s: %s", xuid, exc)
            return False

    def _run_lusr_for_other_players(self: _SyncProtocol) -> None:
        """Calcule le LUSR manquant pour tous les joueurs enregistrés (non bloquant).

        Appelé inconditionnellement après ``_run_lusr_post_sync`` (step 5) pour
        rattraper les matchs solo de coéquipiers enregistrés qui ne déclenchent
        jamais le fan-out (car le joueur principal y était absent).

        Crée un moteur léger (sans API) par joueur et appelle
        ``batch_compute_lusr(force=False)`` — mode incrémental, donc rapide
        quand il n'y a rien de nouveau.
        """
        from src.data.sync.engine import DuckDBSyncEngine
        from src.utils.paths import get_shared_matches_path_from_player

        others = self._get_other_registered_players()
        if not others:
            return

        for player in others:
            try:
                db_path: Path = player["db_path"]
                if not db_path.exists():
                    continue
                shared_path = get_shared_matches_path_from_player(db_path)
                if not shared_path or not shared_path.exists():
                    continue

                engine = DuckDBSyncEngine(
                    player_db_path=db_path,
                    xuid=player["xuid"],
                    gamertag=player["gamertag"],
                    shared_db_path=shared_path,
                    shared_read_only=True,
                )
                try:
                    lusr_count = engine.batch_compute_lusr(force=False)
                    if lusr_count > 0:
                        logger.info(
                            "lusr_catchup [%s]: %d match(s) calculé(s)",
                            player["gamertag"],
                            lusr_count,
                        )
                finally:
                    engine.close()
            except Exception as exc:
                logger.warning(
                    "lusr_catchup [%s]: erreur non bloquante: %s", player["gamertag"], exc
                )

    def fanout_repair_missing_scores(self: _SyncProtocol) -> None:
        """Répare les performance_scores manquants chez les coéquipiers enregistrés.

        Appelé lors du short-circuit delta (aucun nouveau match pour le joueur
        courant) pour ne pas laisser les coéquipiers avec des PME incomplets.
        """
        others = self._get_other_registered_players()
        if not others:
            return
        for player in others:
            try:
                if not player["db_path"].exists():
                    continue
                if not self._has_missing_enrichment(player["xuid"], player["db_path"]):
                    continue
                from src.utils.paths import get_shared_matches_path_from_player

                shared_path = get_shared_matches_path_from_player(player["db_path"])
                if not shared_path or not shared_path.exists():
                    continue
                logger.info(
                    "fanout repair [%s]: PME incomplet détecté → enrichissement",
                    player["gamertag"],
                )
                self._run_other_player_enrichment(
                    player["gamertag"], player["xuid"], player["db_path"], shared_path
                )
            except Exception as exc:
                logger.warning(
                    "fanout repair [%s]: erreur non bloquante: %s", player["gamertag"], exc
                )
