"""Tests anti-régression : cliquer un lien gamertag ne switche pas le joueur actif.

Régression #24 (2026-03-14) — root cause complète supprimée le 2026-03-16 :

- Premier patch (#24) : supprimé la lecture de st.query_params["gamertag"] dans
  init_source_state() pour switch de joueur.
- Second patch (#24-bis) : ajout guard _is_nav_link avant _pick_best_duckdb_v4_player().
- Correctif définitif (#24-ter) : suppression de _pick_best_duckdb_v4_player()
  (heuristique "joueur avec le plus de matchs" → non-déterministe, trop coûteuse,
  source du bug récurrent). init_source_state() utilise désormais exclusivement
  default_db (premier joueur alphabétique, déterministe).

Couverture :
- db_path : nav link / sans nav link / env override / SPNKr / SPNKr vide / rérun
- xuid_input : legacy > inferred > guessed > identity
- waypoint_player : depuis identity
"""

from __future__ import annotations

import os
import tempfile
from unittest.mock import MagicMock, patch  # noqa: F401

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _fake_settings(prefer_spnkr: bool = False) -> MagicMock:
    s = MagicMock()
    s.prefer_spnkr_db_if_available = prefer_spnkr
    return s


def _run_init(  # noqa: PLR0913
    session: dict,
    query_params: dict | None = None,
    default_db: str = "",
    prefer_spnkr: bool = False,
    env: dict | None = None,
    identity: tuple[str, str, str] = ("", "", ""),
    guessed_xuid: str = "",
    inferred: str = "",
) -> None:
    """Exécute init_source_state avec des fakes injectés."""
    from src.app.data_loader import init_source_state

    with (
        patch("src.app.data_loader.st") as mock_st,
        patch("src.app.data_loader.default_identity_from_secrets", return_value=identity),
        patch("src.app.data_loader.guess_xuid_from_db_path", return_value=guessed_xuid),
        patch("src.app.data_loader.infer_spnkr_player_from_db_path", return_value=inferred),
        patch.dict("os.environ", env or {}, clear=False),
    ):
        mock_st.session_state = session
        mock_st.query_params = query_params or {}
        init_source_state(default_db, _fake_settings(prefer_spnkr))


# ===========================================================================
# db_path — sélection de la base de données
# ===========================================================================


class TestDbPathSelection:
    """init_source_state doit sélectionner db_path de manière déterministe."""

    def test_nav_gamertag_db_path_stays_default(self) -> None:
        """Régression #24 : ?gamertag= dans l'URL → db_path reste default_db (pas Madina)."""
        session: dict = {}
        _run_init(session, {"gamertag": "Madina97294"}, default_db="data/players/JGtm/stats.duckdb")
        assert session["db_path"] == "data/players/JGtm/stats.duckdb"

    def test_no_nav_gamertag_uses_default_db(self) -> None:
        """Sans ?gamertag= dans l'URL, db_path = default_db (déterministe)."""
        session: dict = {}
        _run_init(session, {}, default_db="data/players/JGtm/stats.duckdb")
        assert session["db_path"] == "data/players/JGtm/stats.duckdb"

    def test_empty_default_db_results_in_empty_path(self) -> None:
        """Sans joueur configuré, db_path est vide (pas de crash)."""
        session: dict = {}
        _run_init(session, {}, default_db="")
        assert session["db_path"] == ""

    def test_rerun_does_not_change_existing_db_path(self) -> None:
        """Si db_path est déjà en session_state (rerun), il ne doit pas être modifié."""
        existing = "data/players/JGtm/stats.duckdb"
        session = {"db_path": existing}
        _run_init(session, {"gamertag": "Madina97294"})
        assert session["db_path"] == existing

    def test_env_override_takes_priority(self) -> None:
        """LEVELUP_DB en env force le db_path, ignorant default_db et SPNKr."""
        session: dict = {}
        with patch("src.app.data_loader.pick_latest_spnkr_db_if_any") as mock_spnkr:
            _run_init(
                session,
                {},
                default_db="data/players/JGtm/stats.duckdb",
                prefer_spnkr=True,
                env={"LEVELUP_DB": "/forced/path.duckdb"},
            )
            mock_spnkr.assert_not_called()
            assert session["db_path"] == "/forced/path.duckdb"

    def test_spnkr_db_selected_when_prefer_enabled(self) -> None:
        """prefer_spnkr_db_if_available=True → SPNKr DB sélectionnée."""
        with tempfile.NamedTemporaryFile(suffix=".duckdb", delete=False) as f:
            spnkr_path = f.name
            f.write(b"x" * 100)
        try:
            session: dict = {}
            with patch("src.app.data_loader.pick_latest_spnkr_db_if_any", return_value=spnkr_path):
                _run_init(
                    session, {}, default_db="data/players/JGtm/stats.duckdb", prefer_spnkr=True
                )
            assert session["db_path"] == spnkr_path
        finally:
            os.unlink(spnkr_path)

    def test_spnkr_db_skipped_when_file_missing(self) -> None:
        """Si la SPNKr DB n'existe pas, fallback sur default_db."""
        session: dict = {}
        with patch(
            "src.app.data_loader.pick_latest_spnkr_db_if_any",
            return_value="/nonexistent/path.duckdb",
        ):
            _run_init(session, {}, default_db="data/players/JGtm/stats.duckdb", prefer_spnkr=True)
        assert session["db_path"] == "data/players/JGtm/stats.duckdb"

    def test_spnkr_db_skipped_when_file_empty(self) -> None:
        """Si la SPNKr DB est un fichier vide, fallback sur default_db."""
        with tempfile.NamedTemporaryFile(suffix=".duckdb", delete=False) as f:
            spnkr_path = f.name
            # fichier de 0 octet
        try:
            session: dict = {}
            with patch("src.app.data_loader.pick_latest_spnkr_db_if_any", return_value=spnkr_path):
                _run_init(
                    session, {}, default_db="data/players/JGtm/stats.duckdb", prefer_spnkr=True
                )
            assert session["db_path"] == "data/players/JGtm/stats.duckdb"
        finally:
            os.unlink(spnkr_path)

    def test_deep_link_match_restores_correct_player_db(self) -> None:
        """?gamertag=X&match_id=Y → restaure la DB du joueur X (lien depuis historique/carrière)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            from pathlib import Path

            players_dir = Path(tmpdir) / "data" / "players"
            # Joueur B : celui dont on veut voir le match (non-default)
            player_b_db = players_dir / "JGtm" / "stats.duckdb"
            player_b_db.parent.mkdir(parents=True)
            player_b_db.write_bytes(b"x" * 100)

            # default_db pointe vers le joueur A (premier alphabétique, pas JGtm)
            default_db = str(players_dir / "AaronPlayer" / "stats.duckdb")
            session: dict = {}
            _run_init(
                session,
                {"gamertag": "JGtm", "match_id": "abc-match-id-1234"},
                default_db=default_db,
            )
            assert session["db_path"] == str(player_b_db)

    def test_deep_link_match_falls_back_when_db_missing(self) -> None:
        """?gamertag=X&match_id=Y mais DB inexistante → fallback sur default_db."""
        session: dict = {}
        _run_init(
            session,
            {"gamertag": "UnknownPlayer", "match_id": "abc-match-id-1234"},
            default_db="data/players/JGtm/stats.duckdb",
        )
        assert session["db_path"] == "data/players/JGtm/stats.duckdb"

    def test_encounter_link_no_match_id_stays_default(self) -> None:
        """Régression #24 renforcé : ?gamertag=X SANS match_id → reste default_db.

        Les liens d'encounter Explorer (?gamertag= seul) ne doivent pas switcher
        même si la DB du gamertag existe réellement.
        """
        with tempfile.TemporaryDirectory() as tmpdir:
            from pathlib import Path

            players_dir = Path(tmpdir) / "data" / "players"
            # La DB de Madina EXISTS mais il n'y a PAS de match_id dans l'URL
            madina_db = players_dir / "Madina97294" / "stats.duckdb"
            madina_db.parent.mkdir(parents=True)
            madina_db.write_bytes(b"x" * 100)

            default_db = str(players_dir / "JGtm" / "stats.duckdb")
            session: dict = {}
            _run_init(
                session,
                {"gamertag": "Madina97294"},  # pas de match_id → encounter link
                default_db=default_db,
            )
            assert session["db_path"] == default_db

    def test_env_override_wins_over_deep_link_match(self) -> None:
        """LEVELUP_DB env prend priorité même sur un deep link match valide."""
        with tempfile.TemporaryDirectory() as tmpdir:
            from pathlib import Path

            players_dir = Path(tmpdir) / "data" / "players"
            player_db = players_dir / "JGtm" / "stats.duckdb"
            player_db.parent.mkdir(parents=True)
            player_db.write_bytes(b"x" * 100)

            default_db = str(players_dir / "AaronPlayer" / "stats.duckdb")
            session: dict = {}
            _run_init(
                session,
                {"gamertag": "JGtm", "match_id": "abc-match-id-1234"},
                default_db=default_db,
                env={"LEVELUP_DB": "/forced/env.duckdb"},
            )
            assert session["db_path"] == "/forced/env.duckdb"


# ===========================================================================
# xuid_input — priorité de remplissage
# ===========================================================================


class TestXuidInputPriority:
    """xuid_input suit la priorité : legacy > inferred > guessed > identity."""

    def test_legacy_xuid_wins(self) -> None:
        """Si 'xuid' est déjà dans session_state (legacy), c'est lui qui est utilisé."""
        session = {"xuid": "legacy_xuid"}
        _run_init(session, {}, default_db="", identity=("id_gt", "", ""), guessed_xuid="guessed")
        assert session["xuid_input"] == "legacy_xuid"

    def test_inferred_from_spnkr_path_used(self) -> None:
        """Si le nom de DB SPNKr permet de déduire un gamertag, il est utilisé."""
        session: dict = {}
        _run_init(
            session, {}, default_db="", inferred="InferredGT", identity=("IdentityGT", "", "")
        )
        assert session["xuid_input"] == "InferredGT"

    def test_guessed_xuid_from_db_path(self) -> None:
        """Si guess_xuid_from_db_path retourne quelque chose, c'est utilisé en fallback."""
        session: dict = {}
        _run_init(
            session, {}, default_db="", guessed_xuid="guessed_gt", identity=("IdentityGT", "", "")
        )
        assert session["xuid_input"] == "guessed_gt"

    def test_identity_gamertag_as_last_resort(self) -> None:
        """En dernier recours, on utilise le gamertag de l'identité."""
        session: dict = {}
        _run_init(session, {}, default_db="", identity=("IdentityGT", "", ""))
        assert session["xuid_input"] == "IdentityGT"

    def test_empty_when_nothing_available(self) -> None:
        """Sans aucune source d'identité, xuid_input est vide."""
        session: dict = {}
        _run_init(session, {}, default_db="", identity=("", "", ""))
        assert session["xuid_input"] == ""

    def test_xuid_input_not_overwritten_on_rerun(self) -> None:
        """Si xuid_input est déjà en session_state, il n'est pas écrasé."""
        session = {"xuid_input": "existing_user"}
        _run_init(session, {}, default_db="", identity=("other_gt", "", ""), guessed_xuid="other")
        assert session["xuid_input"] == "existing_user"


# ===========================================================================
# waypoint_player
# ===========================================================================


class TestWaypointPlayer:
    """waypoint_player vient de l'identité par défaut."""

    def test_waypoint_set_from_identity(self) -> None:
        session: dict = {}
        _run_init(session, {}, identity=("", "", "MyWaypoint"))
        assert session["waypoint_player"] == "MyWaypoint"

    def test_waypoint_empty_when_no_identity(self) -> None:
        session: dict = {}
        _run_init(session, {}, identity=("", "", ""))
        assert session["waypoint_player"] == ""

    def test_waypoint_not_overwritten_on_rerun(self) -> None:
        session = {"waypoint_player": "ExistingWP"}
        _run_init(session, {}, identity=("", "", "OtherWP"))
        assert session["waypoint_player"] == "ExistingWP"
