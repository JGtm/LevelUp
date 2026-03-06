"""Tests pour les optimisations perf du switch Période→Sessions.

Couvre :
- _classify_sessions_solo_squad : classification solo/escouade
- Pré-chargement sessions dans render_filters_sidebar (prefetched_*)
- Suppression du st.rerun() dans le reset escouade→solo
"""

from __future__ import annotations

from datetime import datetime
from unittest.mock import patch

import polars as pl

# ──────────────────────────────────────────────────────────────────────────────
# _classify_sessions_solo_squad — logique pure (pas de Streamlit)
# ──────────────────────────────────────────────────────────────────────────────


def _make_sessions_df(
    rows: list[dict],
) -> pl.DataFrame:
    """Construit un DataFrame sessions de test."""
    return pl.DataFrame(rows).with_columns(
        pl.col("start_time").cast(pl.Datetime("us")),
    )


FRIEND_XUID = "2533274791234567"
OTHER_XUID = "2533274799999999"


class TestClassifySessionsSoloSquad:
    """Tests pour _classify_sessions_solo_squad."""

    def _classify(
        self, base_s: pl.DataFrame, friends: frozenset[str]
    ) -> tuple[list[str], list[str]]:
        with patch("src.app._filters_session.st"):
            from src.app._filters_session import _classify_sessions_solo_squad

            return _classify_sessions_solo_squad(base_s, friends)

    def test_empty_dataframe(self) -> None:
        df = pl.DataFrame(
            schema={
                "session_id": pl.Utf8,
                "session_label": pl.Utf8,
                "start_time": pl.Datetime,
                "teammates_signature": pl.Utf8,
            }
        )
        solo, squad = self._classify(df, frozenset({FRIEND_XUID}))
        assert solo == []
        assert squad == []

    def test_no_friends_all_solo(self) -> None:
        df = _make_sessions_df(
            [
                {
                    "session_id": "1",
                    "session_label": "S1",
                    "start_time": datetime(2025, 10, 1, 20, 0),
                    "teammates_signature": OTHER_XUID,
                },
            ]
        )
        solo, squad = self._classify(df, frozenset())
        assert solo == ["S1"]
        assert squad == []

    def test_friend_present_marks_squad(self) -> None:
        df = _make_sessions_df(
            [
                {
                    "session_id": "1",
                    "session_label": "S1",
                    "start_time": datetime(2025, 10, 1, 20, 0),
                    "teammates_signature": f"{FRIEND_XUID},{OTHER_XUID}",
                },
                {
                    "session_id": "2",
                    "session_label": "S2",
                    "start_time": datetime(2025, 10, 2, 20, 0),
                    "teammates_signature": OTHER_XUID,
                },
            ]
        )
        solo, squad = self._classify(df, frozenset({FRIEND_XUID}))
        assert "S2" in solo
        assert "S1" in squad
        assert len(solo) == 1
        assert len(squad) == 1

    def test_sorted_by_most_recent_first(self) -> None:
        df = _make_sessions_df(
            [
                {
                    "session_id": "old",
                    "session_label": "Old",
                    "start_time": datetime(2025, 1, 1, 10, 0),
                    "teammates_signature": "",
                },
                {
                    "session_id": "new",
                    "session_label": "New",
                    "start_time": datetime(2025, 12, 1, 10, 0),
                    "teammates_signature": "",
                },
            ]
        )
        solo, squad = self._classify(df, frozenset({FRIEND_XUID}))
        assert solo == ["New", "Old"]
        assert squad == []

    def test_no_teammates_signature_column(self) -> None:
        """Sans colonne teammates_signature, tout est solo."""
        df = pl.DataFrame(
            {
                "session_id": ["1"],
                "session_label": ["S1"],
                "start_time": [datetime(2025, 10, 1, 20, 0)],
            }
        )
        solo, squad = self._classify(df, frozenset({FRIEND_XUID}))
        assert solo == ["S1"]
        assert squad == []


# ──────────────────────────────────────────────────────────────────────────────
# Pré-chargement sessions (warm cache) — tests structurels
# ──────────────────────────────────────────────────────────────────────────────


class TestPrefetchSessionsInRenderFiltersSidebar:
    """Vérifie que render_filters_sidebar pré-charge les sessions."""

    def test_prefetch_calls_before_radio(self) -> None:
        """Le pré-chargement (get_friends + compute_sessions) est appelé
        AVANT le radio filter_mode dans le source.
        """
        from pathlib import Path

        source = (Path("src/app/filters_render.py")).read_text(encoding="utf-8")

        # Les appels prefetch doivent apparaître AVANT le radio filter_mode
        prefetch_pos = source.find("_prefetch_friends")
        radio_pos = source.find('key="filter_mode"')
        assert prefetch_pos > 0, "Pré-chargement _prefetch_friends non trouvé"
        assert radio_pos > 0, "Radio filter_mode non trouvé"
        assert prefetch_pos < radio_pos, "Le pré-chargement doit être AVANT le radio filter_mode"

    def test_prefetch_passed_to_render_session_filter(self) -> None:
        """Les données pré-chargées sont passées via kwargs."""
        import inspect

        from src.app.filters_render import render_filters_sidebar

        source = inspect.getsource(render_filters_sidebar)

        assert "prefetched_friends=_prefetch_friends" in source
        assert "prefetched_sessions=_prefetch_sessions" in source


class TestRenderSessionFilterPrefetchSignature:
    """Vérifie la signature et le comportement du prefetch dans _render_session_filter."""

    def test_accepts_prefetched_kwargs(self) -> None:
        """_render_session_filter accepte prefetched_friends et prefetched_sessions."""
        import inspect

        from src.app._filters_session import _render_session_filter

        sig = inspect.signature(_render_session_filter)
        params = sig.parameters

        assert "prefetched_friends" in params
        assert "prefetched_sessions" in params
        # Doivent être keyword-only (après *)
        assert params["prefetched_friends"].kind == inspect.Parameter.KEYWORD_ONLY
        assert params["prefetched_sessions"].kind == inspect.Parameter.KEYWORD_ONLY
        # Doivent avoir None comme default
        assert params["prefetched_friends"].default is None
        assert params["prefetched_sessions"].default is None


# ──────────────────────────────────────────────────────────────────────────────
# Suppression st.rerun() dans le reset escouade→solo
# ──────────────────────────────────────────────────────────────────────────────


class TestNoRerunOnSquadReset:
    """Vérifie que le reset escouade→solo n'appelle pas st.rerun()."""

    def test_no_rerun_in_squad_to_solo_reset(self) -> None:
        """Le bloc de détection changement escouade ne doit PAS contenir st.rerun().

        L'ancien code faisait :
            if _post_squad != _pre_squad:
                st.session_state[...] = ...
                st.rerun()  # ← supprimé

        Le changement de selectbox déclenche un rerun naturel Streamlit.
        """
        import inspect

        from src.app._filters_session import _render_session_filter

        source = inspect.getsource(_render_session_filter)

        # Localiser le bloc de détection changement escouade
        marker = "Détection changement escouade"
        marker_pos = source.find(marker)
        assert marker_pos > 0, f"Marqueur '{marker}' non trouvé dans le source"

        # Prendre le bloc entre ce marqueur et la section suivante
        next_section = source.find("Sélection active consolidée", marker_pos)
        assert next_section > marker_pos
        squad_reset_block = source[marker_pos:next_section]

        # Vérifier qu'il n'y a pas d'appel réel st.rerun()
        # (exclure les commentaires qui mentionnent st.rerun())
        import re

        # Cherche st.rerun() en début de ligne (indenté), pas dans les commentaires
        real_rerun_calls = re.findall(r"^\s+st\.rerun\(\)", squad_reset_block, re.MULTILINE)
        assert (
            len(real_rerun_calls) == 0
        ), "st.rerun() ne doit plus être appelé dans le bloc de reset escouade→solo"

    def test_rerun_still_used_for_buttons(self) -> None:
        """Les boutons "Dernière session" / "Session précédente" utilisent
        toujours st.rerun() — c'est attendu car les boutons n'ont pas
        de rerun naturel comme les selectboxes.
        """
        import inspect

        from src.app._filters_session import _render_session_filter

        source = inspect.getsource(_render_session_filter)

        # Les boutons doivent toujours avoir st.rerun()
        assert (
            source.count("st.rerun()") >= 4
        ), "Les 4 boutons (solo last/prev, squad last/prev) doivent garder st.rerun()"


# ──────────────────────────────────────────────────────────────────────────────
# Logs de diagnostic
# ──────────────────────────────────────────────────────────────────────────────


class TestPrefetchLogs:
    """Vérifie que les logs de diagnostic sont présents."""

    def test_prefetch_log_in_render_filters_sidebar(self) -> None:
        import inspect

        from src.app.filters_render import render_filters_sidebar

        source = inspect.getsource(render_filters_sidebar)
        assert "Sessions pré-chargées" in source

    def test_cache_usage_log_in_render_session_filter(self) -> None:
        import inspect

        from src.app._filters_session import _render_session_filter

        source = inspect.getsource(_render_session_filter)
        assert "utilisation du cache pré-chargé" in source

    def test_squad_reset_log_in_render_session_filter(self) -> None:
        import inspect

        from src.app._filters_session import _render_session_filter

        source = inspect.getsource(_render_session_filter)
        assert "Reset solo" in source
