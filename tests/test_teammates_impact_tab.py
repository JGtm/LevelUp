"""Tests ciblés pour l'onglet Impact & Taquinerie (page coéquipiers)."""

from __future__ import annotations

from contextlib import contextmanager
from unittest.mock import MagicMock

import polars as pl

from src.ui.pages import teammates_impact as teammates


@contextmanager
def _fake_expander(*_args, **_kwargs):
    yield


class _FakeResult:
    def __init__(self, rows):
        self._rows = rows

    def fetchone(self):
        return self._rows[0] if self._rows else None

    def fetchall(self):
        return self._rows


class _FakeConn:
    def execute(self, query, _params=None):
        normalized_query = " ".join(str(query).split()).lower()

        if "information_schema.tables" in normalized_query:
            return _FakeResult([("highlight_events",)])
        if "from highlight_events" in normalized_query:
            return _FakeResult(
                [
                    ("m1", "100", "Alice", "Kill", 1000),
                    ("m2", "200", "Bob", "Kill", 1500),
                ]
            )
        if "from match_stats" in normalized_query:
            return _FakeResult([("m1", 2), ("m2", 3)])

        return _FakeResult([])


class _FakeRepo:
    def __init__(self, *_args, **_kwargs):
        self._conn = _FakeConn()

    def _get_connection(self):
        return self._conn


def _patch_streamlit(monkeypatch):
    info = MagicMock()
    warning = MagicMock()
    caption = MagicMock()
    subheader = MagicMock()
    plotly_chart = MagicMock()
    dataframe = MagicMock()
    success = MagicMock()
    error = MagicMock()
    radio = MagicMock(return_value="heatmap")

    metric_cols = [MagicMock(), MagicMock(), MagicMock(), MagicMock()]
    # 3 colonnes : légende, ranking, mvp/boulet
    summary_cols = [MagicMock(), MagicMock(), MagicMock()]
    columns = MagicMock(return_value=summary_cols)

    monkeypatch.setattr(teammates.st, "expander", _fake_expander)
    monkeypatch.setattr(teammates.st, "info", info)
    monkeypatch.setattr(teammates.st, "warning", warning)
    monkeypatch.setattr(teammates.st, "caption", caption)
    monkeypatch.setattr(teammates.st, "subheader", subheader)
    monkeypatch.setattr(teammates.st, "plotly_chart", plotly_chart)
    monkeypatch.setattr(teammates.st, "dataframe", dataframe)
    monkeypatch.setattr(teammates.st, "success", success)
    monkeypatch.setattr(teammates.st, "error", error)
    monkeypatch.setattr(teammates.st, "radio", radio)
    monkeypatch.setattr(teammates.st, "columns", columns)
    markdown = MagicMock()
    monkeypatch.setattr(teammates.st, "markdown", markdown)

    return {
        "info": info,
        "warning": warning,
        "caption": caption,
        "subheader": subheader,
        "plotly_chart": plotly_chart,
        "dataframe": dataframe,
        "success": success,
        "error": error,
        "radio": radio,
        "columns": columns,
        "metric_cols": metric_cols,
        "summary_cols": summary_cols,
        "markdown": markdown,
    }


def test_impact_tab_requires_at_least_two_friends(monkeypatch) -> None:
    """Affiche un message d'info si aucun coéquipier n'est sélectionné."""
    st_mocks = _patch_streamlit(monkeypatch)

    teammates.render_impact_taquinerie(
        db_path="dummy.duckdb",
        xuid="100",
        match_ids=["m1"],
        friend_xuids=[],
    )

    st_mocks["info"].assert_called_once()
    st_mocks["warning"].assert_not_called()


def test_impact_tab_warns_when_no_matches(monkeypatch) -> None:
    """Affiche un warning s'il n'y a aucun match à analyser."""
    st_mocks = _patch_streamlit(monkeypatch)

    teammates.render_impact_taquinerie(
        db_path="dummy.duckdb",
        xuid="100",
        match_ids=[],
        friend_xuids=["200", "300"],
    )

    st_mocks["warning"].assert_called_once()
    assert "Aucun match" in st_mocks["warning"].call_args[0][0]


def test_impact_tab_handles_missing_highlight_table(monkeypatch) -> None:
    """Affiche un message si highlight_events n'existe pas."""

    class _MinimalRepo:
        """Repo minimal : ne valide pas l'existence du fichier."""

        def __init__(self, *_args, **_kwargs):
            pass

        def _get_connection(self):
            return MagicMock()

    st_mocks = _patch_streamlit(monkeypatch)
    monkeypatch.setattr("src.ui.pages.teammates_impact.DuckDBRepository", _MinimalRepo)
    # Mocker ensure_shared_attached pour simuler que shared est attaché
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact.ensure_shared_attached",
        lambda _conn, _db_path: "shared",
    )
    # Mocker get_shared_matches_path_from_player
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact.get_shared_matches_path_from_player",
        lambda _p: MagicMock(exists=lambda: True),
    )
    # Mocker _load_highlight_events pour simuler l'absence de la table
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact._load_highlight_events",
        lambda _conn, _match_ids, _alias: None,
    )

    teammates.render_impact_taquinerie(
        db_path="dummy.duckdb",
        xuid="100",
        match_ids=["m1", "m2"],
        friend_xuids=["200", "300"],
    )

    st_mocks["info"].assert_called()
    # Le message d'absence est maintenant via t("tmi_no_events") — vérifier que st.info est appelé
    assert st_mocks["info"].call_count >= 1


def test_impact_tab_renders_heatmap_and_ranking(monkeypatch) -> None:
    """Parcours nominal : rendu heatmap + tableau + résumé MVP/Boulet."""
    st_mocks = _patch_streamlit(monkeypatch)

    class _MinimalRepo:
        """Repo minimal : ne valide pas l'existence du fichier."""

        def __init__(self, *_args, **_kwargs):
            pass

        def _get_connection(self):
            return MagicMock()

    monkeypatch.setattr("src.ui.pages.teammates_impact.DuckDBRepository", _MinimalRepo)
    # Mocker ensure_shared_attached pour simuler que shared est attaché
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact.ensure_shared_attached",
        lambda _conn, _db_path: "shared",
    )
    # Mocker get_shared_matches_path_from_player
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact.get_shared_matches_path_from_player",
        lambda _p: MagicMock(exists=lambda: True),
    )
    # Mocker _load_highlight_events pour retourner des événements
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact._load_highlight_events",
        lambda _conn, _match_ids, _alias: pl.DataFrame(
            {
                "match_id": ["m1", "m2"],
                "xuid": ["200", "300"],
                "gamertag": ["Alice", "Bob"],
                "event_type": ["Kill", "Kill"],
                "time_ms": [1000, 1500],
            }
        ),
    )
    # Mocker _load_match_outcomes pour retourner des outcomes
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact._load_match_outcomes",
        lambda _conn, _match_ids, _xuid, _alias: pl.DataFrame(
            {"match_id": ["m1", "m2"], "outcome": [2, 3]}
        ),
    )
    # Pas de participants → badge computation ignorée
    monkeypatch.setattr(
        "src.ui.pages.teammates_impact._load_match_participants",
        lambda *_: None,
    )

    scores = {"Alice": 3, "Bob": -1}

    # get_all_impact_events retourne un 6-tuple (last_group_kills + first_group_deaths ajoutés)
    monkeypatch.setattr(
        teammates,
        "get_all_impact_events",
        lambda *_args, **_kwargs: (
            {"m1": MagicMock(gamertag="Alice")},
            {},
            {},
            {},
            {},
            scores,
        ),
    )
    monkeypatch.setattr(
        teammates,
        "build_impact_matrix",
        lambda *_args, **_kwargs: pl.DataFrame(
            {
                "match_id": pl.Series([], dtype=pl.Utf8),
                "gamertag": pl.Series([], dtype=pl.Utf8),
                "events": pl.Series(
                    [],
                    dtype=pl.List(
                        pl.Struct([pl.Field("event", pl.Utf8), pl.Field("value", pl.Float64)])
                    ),
                ),
                "outcome": pl.Series([], dtype=pl.Int64),
            }
        ),
    )

    teammates.render_impact_taquinerie(
        db_path="dummy.duckdb",
        xuid="100",
        match_ids=["m1", "m2"],
        friend_xuids=["200", "300"],
    )

    # Le tableau de ranking est rendu via st.markdown (pas plotly_chart)
    assert st_mocks["markdown"].called
    assert not st_mocks["warning"].called
