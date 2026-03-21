"""Tests pour src/analysis/filters.py — mark_firefight et build_xuid_option_map."""

import polars as pl

from src.analysis.filters import build_xuid_option_map, mark_firefight


class TestMarkFirefight:
    """Tests pour mark_firefight."""

    def test_firefight_playlist(self) -> None:
        df = pl.DataFrame(
            {
                "match_id": ["m1", "m2"],
                "playlist_name": ["Firefight", "Ranked Arena"],
                "pair_name": ["x", "y"],
                "game_variant_name": ["a", "b"],
            }
        )
        result = mark_firefight(df)
        assert "is_firefight" in result.columns
        values = result["is_firefight"].to_list()
        assert values[0] is True
        assert values[1] is False

    def test_firefight_pair_name(self) -> None:
        df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "playlist_name": ["Quick Play"],
                "pair_name": ["Firefight:King of the Hill"],
                "game_variant_name": ["standard"],
            }
        )
        result = mark_firefight(df)
        assert result["is_firefight"].to_list() == [True]

    def test_no_firefight(self) -> None:
        df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "playlist_name": ["Ranked Arena"],
                "pair_name": ["Arena:Slayer"],
                "game_variant_name": ["standard"],
            }
        )
        result = mark_firefight(df)
        assert result["is_firefight"].to_list() == [False]

    def test_missing_columns(self) -> None:
        df = pl.DataFrame({"match_id": ["m1"]})
        result = mark_firefight(df)
        assert "is_firefight" in result.columns
        assert result["is_firefight"].to_list() == [False]

    def test_case_insensitive(self) -> None:
        df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "playlist_name": ["FIREFIGHT KOTH"],
            }
        )
        result = mark_firefight(df)
        assert result["is_firefight"].to_list() == [True]


class TestBuildXuidOptionMap:
    """Tests pour build_xuid_option_map."""

    def test_basic(self) -> None:
        result = build_xuid_option_map(["xuid123", "xuid456"])
        assert len(result) == 2
        assert all(isinstance(v, str) for v in result.values())

    def test_with_display_fn(self) -> None:
        def display(xuid: str) -> str:
            return f"Player_{xuid[-3:]}"

        result = build_xuid_option_map(["xuid123", "xuid456"], display_name_fn=display)
        # Labels include "display — xuid" format
        keys = list(result.keys())
        assert any("Player_123" in k for k in keys)
        assert any("Player_456" in k for k in keys)

    def test_empty(self) -> None:
        result = build_xuid_option_map([])
        assert result == {}
