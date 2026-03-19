"""Mixin — enrichissement fan-out des autres joueurs enregistrés post-sync.

Après la sync d'un joueur (A), met à jour les enrichissements locaux
(performance_score, sessions, citations) des autres joueurs enregistrés
dans db_profiles.json pour les matchs communs nouvellement insérés.

Contrainte d'ordre : appelé APRÈS _detach_shared_from_player_conn()
et _run_lusr_post_sync() pour que shared_matches_v2.duckdb soit libre.
"""

from __future__ import annotations

import contextlib
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
        """Lance perf_scores + sessions + citations pour un autre joueur.

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
        )
        try:
            perf_count = engine.batch_compute_performance_scores()
            logger.info("fanout [%s]: %d performance_score(s) calculé(s)", gamertag, perf_count)

            # Libérer shared avant sessions/citations (évite le conflit R/W + ATTACH R/O)
            if engine._shared_connection is not None:
                with contextlib.suppress(Exception):
                    engine._shared_connection.close()
                engine._shared_connection = None

            player_conn = engine._get_connection()
            self._run_other_sessions(gamertag, xuid, player_db_path, player_conn)
            self._run_other_citations(gamertag, xuid, player_db_path, player_conn)
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
                logger.warning(
                    "fanout [%s]: erreur non bloquante: %s", player["gamertag"], exc
                )
