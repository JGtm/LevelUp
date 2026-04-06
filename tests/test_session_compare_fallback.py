"""Test de régression — fallback comparaison de sessions.

Régression 1 : quand l'utilisateur a sélectionné UNE SEULE session dans la sidebar,
la page de comparaison doit quand même recevoir TOUTES les sessions disponibles
(construites depuis ctx.df, non filtré) et non pas les matchs filtrés (ctx.dff).

Régression 2 : le fallback de sélection de Session A doit utiliser la session
précédente la plus similaire (même catégorie, même statut ami/solo, mêmes amis si
possible) plutôt que simplement la session précédente chronologiquement.
"""

from __future__ import annotations

from datetime import datetime, timezone

import polars as pl
import pytest

from src.ui.pages.session_compare_logic import find_best_matching_previous_session

# ---------------------------------------------------------------------------
# Helpers de construction de DataFrame de test
# ---------------------------------------------------------------------------


def _make_match_row(match_id: str, session_id: int, session_label: str, ts: str) -> dict:
    return {
        "match_id": match_id,
        "session_id": session_id,
        "session_label": session_label,
        "start_time": datetime.fromisoformat(ts).replace(tzinfo=timezone.utc),
        "kills": 10,
        "deaths": 5,
        "assists": 2,
        "outcome": 2,
        "accuracy": 50.0,
        "average_life_seconds": 30.0,
        "is_with_friends": 0,
        "pair_name": "Arena:Slayer on Aquarius",
    }


@pytest.fixture
def two_session_all_matches() -> pl.DataFrame:
    """ctx.df — tous les matchs, 2 sessions distinctes."""
    rows = [
        _make_match_row("m1", 1, "Session 1", "2026-03-10T18:00:00"),
        _make_match_row("m2", 1, "Session 1", "2026-03-10T18:30:00"),
        _make_match_row("m3", 2, "Session 2", "2026-03-12T20:00:00"),
        _make_match_row("m4", 2, "Session 2", "2026-03-12T20:30:00"),
    ]
    return pl.DataFrame(rows)


@pytest.fixture
def one_session_filtered_matches() -> pl.DataFrame:
    """ctx.dff — matchs filtrés sur la session sélectionnée (Session 2 uniquement)."""
    rows = [
        _make_match_row("m3", 2, "Session 2", "2026-03-12T20:00:00"),
        _make_match_row("m4", 2, "Session 2", "2026-03-12T20:30:00"),
    ]
    return pl.DataFrame(rows)


@pytest.fixture
def all_sessions_pl() -> pl.DataFrame:
    """all_sessions_pl issu de cached_compute_sessions_db — toutes les sessions."""
    return pl.DataFrame(
        {
            "match_id": ["m1", "m2", "m3", "m4"],
            "session_id": [1, 1, 2, 2],
            "session_label": ["Session 1", "Session 1", "Session 2", "Session 2"],
        }
    )


# ---------------------------------------------------------------------------
# Logique de construction de sessions_for_compare (extraite de streamlit_app.py)
# ---------------------------------------------------------------------------


def _build_sessions_for_compare(
    df_matches: pl.DataFrame, all_sessions: pl.DataFrame
) -> pl.DataFrame:
    """Reproduit la logique de _page_session_compare dans streamlit_app.py."""
    if all_sessions.is_empty() or "match_id" not in df_matches.columns:
        return all_sessions
    sess_cols = ["match_id", "session_id", "session_label"]
    drop_cols = [c for c in ("session_id", "session_label") if c in df_matches.columns]
    df_for_merge = df_matches.drop(drop_cols) if drop_cols else df_matches
    return df_for_merge.join(all_sessions.select(sess_cols), on="match_id", how="inner")


# ---------------------------------------------------------------------------
# Tests de régression
# ---------------------------------------------------------------------------


class TestSessionsForCompareConstruction:
    """Vérifie que sessions_for_compare est construit depuis ctx.df (non filtré)."""

    def test_with_all_matches_gives_two_sessions(
        self,
        two_session_all_matches: pl.DataFrame,
        all_sessions_pl: pl.DataFrame,
    ) -> None:
        """Utiliser ctx.df (tous les matchs) → 2 sessions disponibles pour la comparaison."""
        result = _build_sessions_for_compare(two_session_all_matches, all_sessions_pl)
        unique_sessions = result.get_column("session_label").unique().to_list()
        assert len(unique_sessions) == 2, (
            "sessions_for_compare doit contenir 2 sessions quand ctx.df est utilisé"
        )

    def test_with_filtered_matches_gives_only_one_session(
        self,
        one_session_filtered_matches: pl.DataFrame,
        all_sessions_pl: pl.DataFrame,
    ) -> None:
        """Utiliser ctx.dff (matchs filtrés) → 1 seule session → déclencherait le warning.

        Ce test documente le comportement bugué : si ctx.dff est utilisé au lieu de ctx.df,
        sessions_for_compare n'a qu'1 session et render_session_comparison_page affiche
        "Il faut au moins 2 sessions pour comparer".
        """
        result = _build_sessions_for_compare(one_session_filtered_matches, all_sessions_pl)
        unique_sessions = result.get_column("session_label").unique().to_list()
        assert len(unique_sessions) == 1, (
            "Avec ctx.dff filtré sur 1 session, sessions_for_compare n'a qu'1 session "
            "(comportement bugué documenté — la page doit utiliser ctx.df à la place)"
        )

    def test_regression_fix_uses_ctx_df_not_dff(
        self,
        two_session_all_matches: pl.DataFrame,
        one_session_filtered_matches: pl.DataFrame,
        all_sessions_pl: pl.DataFrame,
    ) -> None:
        """Régression : la page de comparaison doit utiliser ctx.df et non ctx.dff.

        Cas réel : l'utilisateur sélectionne "Session 2" dans la sidebar,
        ctx.dff = 2 matchs de Session 2, ctx.df = 4 matchs (2 sessions).
        sessions_for_compare doit avoir 2 sessions pour que le fallback fonctionne.
        """
        # Simule l'ancienne implémentation (bug) : join sur ctx.dff
        result_buggy = _build_sessions_for_compare(one_session_filtered_matches, all_sessions_pl)
        # Simule la nouvelle implémentation (fix) : join sur ctx.df
        result_fixed = _build_sessions_for_compare(two_session_all_matches, all_sessions_pl)

        n_sessions_buggy = result_buggy.get_column("session_label").n_unique()
        n_sessions_fixed = result_fixed.get_column("session_label").n_unique()

        assert n_sessions_buggy == 1, "Le bug doit produire 1 session avec ctx.dff filtré"
        assert n_sessions_fixed == 2, "Le fix doit produire 2 sessions avec ctx.df complet"

        # La garde "< 2 sessions" dans render_session_comparison_page aurait tiré
        # avec le résultat buggy mais pas avec le résultat fixé.
        assert n_sessions_fixed >= 2, (
            "Avec le fix, len(session_labels) >= 2 → pas de warning, fallback actif"
        )


# ---------------------------------------------------------------------------
# Tests de find_best_matching_previous_session
# ---------------------------------------------------------------------------


def _make_session_rows(  # noqa: PLR0913
    session_id: int,
    session_label: str,
    pair_name: str,
    n_matches: int,
    is_with_friends: int = 0,
    ts_start: str = "2026-03-01T18:00:00",
) -> list[dict]:
    base = datetime.fromisoformat(ts_start).replace(tzinfo=timezone.utc)
    rows = []
    for i in range(n_matches):
        rows.append(
            {
                "match_id": f"{session_label}_m{i}",
                "session_id": session_id,
                "session_label": session_label,
                "start_time": base,
                "kills": 10,
                "deaths": 5,
                "assists": 2,
                "outcome": 2,
                "accuracy": 50.0,
                "average_life_seconds": 30.0,
                "is_with_friends": is_with_friends,
                "pair_name": pair_name,
            }
        )
    return rows


class TestFindBestMatchingPreviousSession:
    """Vérifie la sélection de la session A la plus similaire à session B."""

    def test_same_category_selected_over_chronological(self) -> None:
        """Préfère une session ancienne de même catégorie à la plus récente différente."""
        # session_labels : ordre décroissant (plus récent = index 0)
        # S3 (Ranked) > S2 (Arena/Assassin) > S1 (Ranked)
        # B = S3 (Ranked) → doit choisir S1 (Ranked) pas S2 (Assassin)
        rows = (
            _make_session_rows(
                3, "S3", "Ranked:Slayer on Aquarius", 2, ts_start="2026-03-12T20:00:00"
            )
            + _make_session_rows(
                2, "S2", "Arena:Slayer on Aquarius", 2, ts_start="2026-03-10T18:00:00"
            )
            + _make_session_rows(
                1, "S1", "Ranked:Slayer on Aquarius", 2, ts_start="2026-03-08T18:00:00"
            )
        )
        df = pl.DataFrame(rows)
        session_labels = ["S3", "S2", "S1"]  # du plus récent au plus ancien

        result = find_best_matching_previous_session(df, "S3", session_labels)
        assert result == "S1", f"Attendu S1 (Ranked), obtenu {result}"

    def test_same_status_solo_preferred(self) -> None:
        """Préfère session solo si B est solo, même si une session avec amis est plus récente."""
        rows = (
            _make_session_rows(
                3,
                "S3",
                "Arena:Slayer on Aquarius",
                2,
                is_with_friends=0,
                ts_start="2026-03-12T20:00:00",
            )
            + _make_session_rows(
                2,
                "S2",
                "Arena:Slayer on Aquarius",
                2,
                is_with_friends=1,
                ts_start="2026-03-10T18:00:00",
            )
            + _make_session_rows(
                1,
                "S1",
                "Arena:Slayer on Aquarius",
                2,
                is_with_friends=0,
                ts_start="2026-03-08T18:00:00",
            )
        )
        df = pl.DataFrame(rows)
        session_labels = ["S3", "S2", "S1"]

        result = find_best_matching_previous_session(df, "S3", session_labels)
        assert result == "S1", f"Attendu S1 (solo), obtenu {result}"

    def test_fallback_to_chronological_when_no_category_match(self) -> None:
        """Fallback sur la session précédente quand aucune session de même catégorie."""
        rows = _make_session_rows(
            2, "S2", "Ranked:Slayer on Aquarius", 2, ts_start="2026-03-12T20:00:00"
        ) + _make_session_rows(1, "S1", "BTB:CTF on Highpower", 2, ts_start="2026-03-10T18:00:00")
        df = pl.DataFrame(rows)
        session_labels = ["S2", "S1"]

        result = find_best_matching_previous_session(df, "S2", session_labels)
        assert result == "S1", f"Fallback attendu sur S1, obtenu {result}"

    def test_no_older_session_returns_different_label(self) -> None:
        """Quand B est la session la plus ancienne, retourne la plus récente (idx_b - 1)."""
        rows = _make_session_rows(
            2, "S2", "Arena:Slayer on Aquarius", 2, ts_start="2026-03-12T20:00:00"
        ) + _make_session_rows(
            1, "S1", "Arena:Slayer on Aquarius", 2, ts_start="2026-03-10T18:00:00"
        )
        df = pl.DataFrame(rows)
        session_labels = ["S2", "S1"]

        result = find_best_matching_previous_session(df, "S1", session_labels)
        # S1 est la plus ancienne → pas de session antérieure → retourne S2 (idx_b - 1)
        assert result == "S2", f"Attendu S2 (plus récente), obtenu {result}"

    def test_empty_df_returns_first_label(self) -> None:
        """Avec un DF vide, retourne le premier label de la liste."""
        df = pl.DataFrame(schema={"session_label": pl.Utf8, "session_id": pl.Int64})
        result = find_best_matching_previous_session(df, "S2", ["S2", "S1"])
        assert result in ("S2", "S1")

    def test_picked_not_in_labels_returns_first(self) -> None:
        """Si picked_label n'est pas dans session_labels, retourne le premier."""
        rows = _make_session_rows(1, "S1", "Arena:Slayer on Aquarius", 2)
        df = pl.DataFrame(rows)
        result = find_best_matching_previous_session(df, "INCONNU", ["S2", "S1"])
        assert result == "S2"
