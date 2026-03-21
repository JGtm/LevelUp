"""Tests pour les pages d'armes (match_view_weapon_kills, _timeseries_weapons, teammates_weapons).

Couvre :
- _resolve_weapon_name (pure function)
- _enrich_with_grenade_melee (match view)
- _load_grenade_melee_totals (timeseries)
- _append_grenade_melee (teammates)
- render_weapon_kills_section (flux UI, mock_st)
- render_weapon_kills_chart (flux UI, mock_st)
- Présence des clés i18n col_grenade_kills et col_melee
"""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock, patch

import duckdb
import polars as pl

# =============================================================================
# Helpers
# =============================================================================


def _make_shared_conn(
    *,
    xuid: str = "xuid_test",
    match_id: str = "match_001",
    grenade_kills: int = 3,
    melee_kills: int = 2,
) -> duckdb.DuckDBPyConnection:
    """Crée une connexion DuckDB in-memory avec un shared.match_participants minimal."""
    conn = duckdb.connect(":memory:")
    conn.execute("CREATE SCHEMA shared")
    conn.execute(
        """
        CREATE TABLE shared.match_participants (
            match_id VARCHAR,
            xuid     VARCHAR,
            grenade_kills SMALLINT,
            melee_kills   SMALLINT
        )
        """
    )
    conn.execute(
        "INSERT INTO shared.match_participants VALUES (?, ?, ?, ?)",
        [match_id, xuid, grenade_kills, melee_kills],
    )
    return conn


def _empty_weapons_df() -> pl.DataFrame:
    return pl.DataFrame(schema={"weapon_name": pl.Utf8, "kills": pl.Int32})


def _weapons_df_with_rows() -> pl.DataFrame:
    return pl.DataFrame(
        {"weapon_name": ["Bandit EVO", "BR75"], "kills": [10, 8]},
        schema={"weapon_name": pl.Utf8, "kills": pl.Int32},
    )


# =============================================================================
# Tests i18n
# =============================================================================


class TestI18nWeaponKeys:
    """Vérifie que les clés i18n pour les armes synthétiques existent."""

    def test_col_grenade_kills_key_exists(self) -> None:
        from src.ui.i18n import t

        result = t("col_grenade_kills")
        assert result not in ("col_grenade_kills", ""), f"Clé non résolue : {result!r}"

    def test_col_melee_key_exists(self) -> None:
        from src.ui.i18n import t

        result = t("col_melee")
        assert result not in ("col_melee", ""), f"Clé non résolue : {result!r}"

    def test_mv_weapon_kills_title_no_skull_emoji(self) -> None:
        from src.ui.i18n import t

        title = t("mv_weapon_kills_title")
        assert "💀" not in title, f"Emoji tête de mort inattendu dans le titre : {title!r}"
        assert "Outils de destruction" in title or "Tools of destruction" in title

    def test_ts_top_weapons_title_updated(self) -> None:
        from src.ui.i18n import t

        title = t("ts_top_weapons_title")
        assert "🔫" not in title, f"Emoji pistolet inattendu : {title!r}"

    def test_tm_weapon_kills_chart_updated(self) -> None:
        from src.ui.i18n import t

        title = t("tm_weapon_kills_chart")
        assert "Outils de destruction" in title or "Tools of destruction" in title


# =============================================================================
# Tests _resolve_weapon_name (teammates_weapons)
# =============================================================================


class TestResolveWeaponName:
    """_resolve_weapon_name est une fonction pure — pas de DB nécessaire."""

    def test_grenade_sentinel_returns_grenade_label(self) -> None:
        from src.ui.pages.teammates_weapons import _resolve_weapon_name

        result = _resolve_weapon_name(0, "fr")
        assert result == "Grenades"

    def test_grenade_sentinel_en(self) -> None:
        from src.ui.pages.teammates_weapons import _resolve_weapon_name

        result = _resolve_weapon_name(0, "en")
        assert result == "Grenades"

    def test_melee_sentinel_returns_melee_label_fr(self) -> None:
        from src.ui.pages.teammates_weapons import _resolve_weapon_name

        result = _resolve_weapon_name(1, "fr")
        assert result == "Corps à corps"

    def test_melee_sentinel_en(self) -> None:
        from src.ui.pages.teammates_weapons import _resolve_weapon_name

        result = _resolve_weapon_name(1, "en")
        assert result == "Melee"

    def test_regular_weapon_delegates_to_resolve_display(self) -> None:
        from src.ui.pages.teammates_weapons import _resolve_weapon_name

        # weapon_id inconnu → ne doit pas lever d'exception
        result = _resolve_weapon_name(99999, "fr")
        assert isinstance(result, str)


# =============================================================================
# Tests _append_grenade_melee (teammates_weapons)
# =============================================================================


class TestAppendGrenadeMelee:
    """Tests avec une vraie connexion DuckDB in-memory (shared.match_participants simulé)."""

    def _base_df(self) -> pl.DataFrame:
        return pl.DataFrame(
            {
                "weapon_id": pl.Series([12345678], dtype=pl.UInt64),
                "total_kills": pl.Series([5], dtype=pl.Int32),
            }
        )

    def test_appends_grenade_row_when_kills_gt_0(self) -> None:
        from src.ui.pages.teammates_weapons import _append_grenade_melee

        conn = _make_shared_conn(grenade_kills=3, melee_kills=0)
        df = self._base_df()
        result = _append_grenade_melee(conn, "xuid_test", ["match_001"], df)

        assert len(result) == 2
        weapon_ids = result["weapon_id"].to_list()
        assert 0 in weapon_ids  # grenade sentinel

    def test_appends_melee_row_when_kills_gt_0(self) -> None:
        from src.ui.pages.teammates_weapons import _append_grenade_melee

        conn = _make_shared_conn(grenade_kills=0, melee_kills=4)
        df = self._base_df()
        result = _append_grenade_melee(conn, "xuid_test", ["match_001"], df)

        assert len(result) == 2
        weapon_ids = result["weapon_id"].to_list()
        assert 1 in weapon_ids  # melee sentinel

    def test_appends_both_rows(self) -> None:
        from src.ui.pages.teammates_weapons import _append_grenade_melee

        conn = _make_shared_conn(grenade_kills=3, melee_kills=2)
        df = self._base_df()
        result = _append_grenade_melee(conn, "xuid_test", ["match_001"], df)

        assert len(result) == 3
        weapon_ids = set(result["weapon_id"].to_list())
        assert 0 in weapon_ids and 1 in weapon_ids

    def test_no_rows_appended_when_both_zero(self) -> None:
        from src.ui.pages.teammates_weapons import _append_grenade_melee

        conn = _make_shared_conn(grenade_kills=0, melee_kills=0)
        df = self._base_df()
        result = _append_grenade_melee(conn, "xuid_test", ["match_001"], df)

        assert len(result) == 1  # inchangé

    def test_correct_kill_count_in_appended_row(self) -> None:
        from src.ui.pages.teammates_weapons import _append_grenade_melee

        conn = _make_shared_conn(grenade_kills=7, melee_kills=0)
        df = self._base_df()
        result = _append_grenade_melee(conn, "xuid_test", ["match_001"], df)

        grenade_row = result.filter(pl.col("weapon_id") == 0)
        assert grenade_row["total_kills"].item() == 7

    def test_returns_original_df_on_db_error(self) -> None:
        from src.ui.pages.teammates_weapons import _append_grenade_melee

        bad_conn = MagicMock()
        bad_conn.execute.side_effect = RuntimeError("connexion fermée")
        df = self._base_df()
        result = _append_grenade_melee(bad_conn, "xuid_test", ["match_001"], df)

        assert result.equals(df)

    def test_without_match_ids(self) -> None:
        from src.ui.pages.teammates_weapons import _append_grenade_melee

        conn = _make_shared_conn(grenade_kills=2, melee_kills=1)
        df = self._base_df()
        result = _append_grenade_melee(conn, "xuid_test", [], df)

        # match_ids=[] → requête WHERE xuid = ? sans filtre match_id
        assert len(result) >= 1  # au moins la ligne originale


# =============================================================================
# Tests _enrich_with_grenade_melee (match_view_weapon_kills)
# =============================================================================


class TestEnrichWithGrenadeMelee:
    """Tests — le repo est passé directement, pas de création interne."""

    def _mock_repo(
        self,
        grenade_kills: int,
        melee_kills: int,
        api_total: int | None = None,
    ) -> Any:
        """Crée un mock repo. api_total par défaut = film_kills + grenade + melee (pas de limite)."""
        mock_repo = MagicMock()
        mock_repo.load_grenade_melee_kills.return_value = (grenade_kills, melee_kills)
        # _weapons_df_with_rows() a 18 kills ; par défaut on laisse tout passer
        default_total = 18 + grenade_kills + melee_kills
        mock_repo.load_total_kills_for_player.return_value = (
            api_total if api_total is not None else default_total
        )
        return mock_repo

    def test_adds_grenade_row_to_df(self) -> None:
        from src.ui.pages.match_view_weapon_kills import _enrich_with_grenade_melee

        result = _enrich_with_grenade_melee(
            _weapons_df_with_rows(),
            self._mock_repo(grenade_kills=5, melee_kills=0),
            "match_001",
            "xuid_test",
        )

        assert len(result) == 3
        assert any("grenade" in n.lower() for n in result["weapon_name"].to_list())

    def test_adds_melee_row_to_df(self) -> None:
        from src.ui.pages.match_view_weapon_kills import _enrich_with_grenade_melee

        result = _enrich_with_grenade_melee(
            _weapons_df_with_rows(),
            self._mock_repo(grenade_kills=0, melee_kills=3),
            "match_001",
            "xuid_test",
        )

        assert len(result) == 3
        assert any(
            "corps" in n.lower() or "melee" in n.lower() for n in result["weapon_name"].to_list()
        )

    def test_no_enrichment_when_both_zero(self) -> None:
        from src.ui.pages.match_view_weapon_kills import _enrich_with_grenade_melee

        result = _enrich_with_grenade_melee(
            _weapons_df_with_rows(),
            self._mock_repo(grenade_kills=0, melee_kills=0),
            "match_001",
            "xuid_test",
        )

        assert len(result) == 2  # inchangé

    def test_returns_original_df_on_error(self) -> None:
        from src.ui.pages.match_view_weapon_kills import _enrich_with_grenade_melee

        mock_repo = MagicMock()
        mock_repo.load_grenade_melee_kills.side_effect = RuntimeError("DB error")
        df = _weapons_df_with_rows()

        result = _enrich_with_grenade_melee(df, mock_repo, "match_001", "xuid_test")

        assert result.equals(df)

    def test_returns_original_df_when_load_returns_zeros(self) -> None:
        """Aucun résultat grenade/mêlée (ex. pas de ligne participant) → df inchangé."""
        from src.ui.pages.match_view_weapon_kills import _enrich_with_grenade_melee

        result = _enrich_with_grenade_melee(
            _weapons_df_with_rows(),
            self._mock_repo(grenade_kills=0, melee_kills=0),
            "match_001",
            "xuid_test",
        )

        assert len(result) == 2  # inchangé

    def test_no_double_count_when_film_covers_melee(self) -> None:
        """Si le film couvre déjà tous les kills (api_total = film_kills), melee=0 ajouté."""
        from src.ui.pages.match_view_weapon_kills import _enrich_with_grenade_melee

        # film = 18, api_total = 18 → remainder = 0 → melee_net = 0
        result = _enrich_with_grenade_melee(
            _weapons_df_with_rows(),
            self._mock_repo(grenade_kills=0, melee_kills=7, api_total=18),
            "match_001",
            "xuid_test",
        )

        assert len(result) == 2  # aucune ligne melee ajoutée (déjà dans le film)
        kills_total = result["kills"].sum()
        assert kills_total == 18

    def test_partial_melee_added_when_film_partially_covers(self) -> None:
        """Si film=18, api_total=21, melee=7 → remainder=3 → melee_net=3 seulement."""
        from src.ui.pages.match_view_weapon_kills import _enrich_with_grenade_melee

        result = _enrich_with_grenade_melee(
            _weapons_df_with_rows(),
            self._mock_repo(grenade_kills=0, melee_kills=7, api_total=21),
            "match_001",
            "xuid_test",
        )

        melee_row = [
            r
            for r in result.iter_rows(named=True)
            if "corps" in r["weapon_name"].lower() or "melee" in r["weapon_name"].lower()
        ]
        assert melee_row, "Une ligne mêlée doit être ajoutée"
        assert melee_row[0]["kills"] == 3  # min(7, 21-18)=3


# =============================================================================
# Tests _load_grenade_melee_totals (_timeseries_weapons)
# =============================================================================


class TestLoadGrenadeMeleeTotals:
    """Tests avec mock DuckDBRepository."""

    def _mock_repo_conn(self, grenade_kills: int, melee_kills: int) -> Any:
        mock_repo = MagicMock()
        mock_repo.load_grenade_melee_kills.return_value = (grenade_kills, melee_kills)
        return mock_repo

    def test_returns_correct_totals_with_match_ids(self) -> None:
        from src.ui.pages._timeseries_weapons import _load_grenade_melee_totals

        mock_repo = self._mock_repo_conn(grenade_kills=4, melee_kills=2)
        with patch("src.data.repositories.duckdb_repo.DuckDBRepository", return_value=mock_repo):
            g, m = _load_grenade_melee_totals("/fake", "xuid_test", ["match_001"])

        assert g == 4
        assert m == 2

    def test_returns_correct_totals_without_match_ids(self) -> None:
        from src.ui.pages._timeseries_weapons import _load_grenade_melee_totals

        mock_repo = self._mock_repo_conn(grenade_kills=6, melee_kills=1)
        with patch("src.data.repositories.duckdb_repo.DuckDBRepository", return_value=mock_repo):
            g, m = _load_grenade_melee_totals("/fake", "xuid_test", None)

        assert g == 6
        assert m == 1

    def test_returns_zeros_when_no_data(self) -> None:
        from src.ui.pages._timeseries_weapons import _load_grenade_melee_totals

        mock_repo = MagicMock()
        mock_repo.load_grenade_melee_kills.return_value = (0, 0)

        with patch("src.data.repositories.duckdb_repo.DuckDBRepository", return_value=mock_repo):
            g, m = _load_grenade_melee_totals("/fake", "xuid_test", ["no_such_match"])

        assert g == 0 and m == 0

    def test_returns_zeros_on_db_error(self) -> None:
        from src.ui.pages._timeseries_weapons import _load_grenade_melee_totals

        with patch(
            "src.data.repositories.duckdb_repo.DuckDBRepository",
            side_effect=RuntimeError("DB introuvable"),
        ):
            g, m = _load_grenade_melee_totals("/bad", "xuid_test", ["match_001"])

        assert g == 0 and m == 0


# =============================================================================
# Tests render_weapon_kills_section (flux UI)
# =============================================================================


_MOCK_REPO_CTX = "src.data.repositories.DuckDBRepository"


class TestRenderWeaponKillsSection:
    """Tests du flux de rendu avec mock_st.

    render_weapon_kills_section crée un DuckDBRepository unique qu'il passe
    aux fonctions privées → tous les tests mockent DuckDBRepository au niveau
    du module src.data.repositories.
    """

    def test_shows_no_data_caption_when_df_empty(self, mock_st) -> None:
        import src.ui.pages.match_view_weapon_kills as mod

        ms = mock_st(mod)

        with (
            patch(_MOCK_REPO_CTX),
            patch.object(mod, "_build_weapon_kills_df", return_value=_empty_weapons_df()),
        ):
            mod.render_weapon_kills_section(db_path="/fake", match_id="m1", xuid="u1", colors={})

        ms.calls["caption"].assert_called_once()

    def test_no_chart_when_film_empty_even_if_grenades_available(self, mock_st) -> None:
        """Régression : quand weapon_kills est vide, _enrich_with_grenade_melee n'est pas appelé
        (early return) — le message no_data est attendu, pas le pie chart."""
        import src.ui.pages.match_view_weapon_kills as mod

        ms = mock_st(mod)
        pie_called: list[bool] = []

        def capture_pie(_df: pl.DataFrame, _colors: dict) -> None:
            pie_called.append(True)

        with (
            patch(_MOCK_REPO_CTX),
            patch.object(mod, "_build_weapon_kills_df", return_value=_empty_weapons_df()),
            patch.object(mod, "_render_weapon_pie", side_effect=capture_pie),
        ):
            mod.render_weapon_kills_section(db_path="/fake", match_id="m1", xuid="u1", colors={})

        ms.calls["caption"].assert_called_once()
        assert not pie_called, "Le pie chart ne doit pas s'afficher quand weapon_kills est vide"

    def test_renders_pie_and_table_when_data_present(self, mock_st) -> None:
        import src.ui.pages.match_view_weapon_kills as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()  # noqa: needed for col_chart, col_table = st.columns([3, 2])

        df = _weapons_df_with_rows()
        with (
            patch(_MOCK_REPO_CTX),
            patch.object(mod, "_build_weapon_kills_df", return_value=df),
            patch.object(mod, "_enrich_with_grenade_melee", return_value=df),
            patch.object(mod, "_render_weapon_pie"),
            patch.object(mod, "_render_weapon_table"),
        ):
            mod.render_weapon_kills_section(db_path="/fake", match_id="m1", xuid="u1", colors={})

        ms.calls["subheader"].assert_called_once()

    def test_grenade_row_included_in_pie_chart(self, mock_st) -> None:
        """Vérifie que grenades apparaissent bien dans le df passé au pie chart."""
        import src.ui.pages.match_view_weapon_kills as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        captured: list[pl.DataFrame] = []

        def capture_pie(df: pl.DataFrame, colors: dict) -> None:
            captured.append(df)

        df_enriched = pl.DataFrame(
            {"weapon_name": ["Bandit EVO", "Grenades"], "kills": [10, 4]},
            schema={"weapon_name": pl.Utf8, "kills": pl.Int32},
        )

        with (
            patch(_MOCK_REPO_CTX),
            patch.object(mod, "_build_weapon_kills_df", return_value=_weapons_df_with_rows()),
            patch.object(mod, "_enrich_with_grenade_melee", return_value=df_enriched),
            patch.object(mod, "_render_weapon_pie", side_effect=capture_pie),
            patch.object(mod, "_render_weapon_table"),
        ):
            mod.render_weapon_kills_section(db_path="/fake", match_id="m1", xuid="u1", colors={})

        assert len(captured) == 1
        assert "Grenades" in captured[0]["weapon_name"].to_list()


# =============================================================================
# Tests render_weapon_kills_chart (timeseries — flux UI)
# =============================================================================


class TestRenderWeaponKillsChart:
    """Tests du flux de rendu timeseries avec mock_st."""

    def _match_df(self) -> pl.DataFrame:
        return pl.DataFrame({"match_id": ["m1", "m2"]})

    def test_no_render_when_weapons_empty(self, mock_st) -> None:
        import src.ui.pages._timeseries_weapons as mod

        ms = mock_st(mod)

        mock_repo = MagicMock()
        mock_repo.load_weapon_kills_aggregated.return_value = pl.DataFrame(
            schema={"weapon_id": pl.UInt64, "total_kills": pl.Int32}
        )

        with patch("src.data.repositories.duckdb_repo.DuckDBRepository", return_value=mock_repo):
            mod.render_weapon_kills_chart(self._match_df(), db_path="/fake", xuid="u1", lang="fr")

        ms.calls["subheader"].assert_not_called()

    def test_subheader_rendered_when_data_present(self, mock_st) -> None:
        import src.ui.pages._timeseries_weapons as mod

        ms = mock_st(mod)

        df_w = pl.DataFrame(
            {
                "weapon_id": pl.Series([12345], dtype=pl.UInt64),
                "total_kills": pl.Series([8], dtype=pl.Int32),
            }
        )
        mock_repo = MagicMock()
        mock_repo.load_weapon_kills_aggregated.return_value = df_w

        with (
            patch("src.data.repositories.duckdb_repo.DuckDBRepository", return_value=mock_repo),
            patch.object(mod, "_load_grenade_melee_totals", return_value=(0, 0)),
            patch.object(mod, "plot_top_weapons", return_value=MagicMock()),
        ):
            mod.render_weapon_kills_chart(self._match_df(), db_path="/fake", xuid="u1", lang="fr")

        ms.calls["subheader"].assert_called_once()
        call_args = ms.calls["subheader"].call_args[0][0]
        assert "Outils de destruction" in call_args or "Tools of destruction" in call_args
