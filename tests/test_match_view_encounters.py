"""Tests unitaires pour la logique de la section Historique des rencontres.

Couvre : ordinal_fr, compute_encounter_badges, filter_encounter_xuids, build_friends_set.
Aucun import Streamlit — logique pure uniquement.
"""

from __future__ import annotations

import json
from datetime import datetime, timezone

from src.ui.pages.match_view_encounters_logic import (
    EncounterStats,
    _relative_date_fr,
    build_friends_set,
    compute_encounter_badges,
    filter_encounter_xuids,
    ordinal_fr,
)

# ---------------------------------------------------------------------------
# ordinal_fr
# ---------------------------------------------------------------------------


class TestOrdinalFr:
    def test_one(self):
        assert ordinal_fr(1) == "1ère"

    def test_two(self):
        assert ordinal_fr(2) == "2e"

    def test_three(self):
        assert ordinal_fr(3) == "3e"

    def test_twelve(self):
        assert ordinal_fr(12) == "12e"

    def test_hundred(self):
        assert ordinal_fr(100) == "100e"

    def test_zero_or_negative(self):
        assert ordinal_fr(0) == "0"
        assert ordinal_fr(-1) == "-1"


# ---------------------------------------------------------------------------
# compute_encounter_badges
# ---------------------------------------------------------------------------


def _make_stats(**kwargs) -> EncounterStats:
    defaults = {
        "xuid": "xuid_test",
        "gamertag": "TestPlayer",
        "total_encounters": 10,
        "ally_count": 5,
        "enemy_count": 5,
        "winrate_as_ally": None,
        "winrate_vs_enemy": None,
        "kills_dealt": 0,
        "deaths_suffered": 0,
        "last_seen": None,
        "current_side": "ennemi",
    }
    defaults.update(kwargs)
    return EncounterStats(**defaults)


class TestComputeEncounterBadges:
    def test_no_badges_by_default(self):
        stats = _make_stats()
        assert compute_encounter_badges(stats) == []

    def test_nemesis_badge(self):
        stats = _make_stats(kills_dealt=1, deaths_suffered=5)
        badges = compute_encounter_badges(stats)
        keys = [b.label_key for b in badges]
        assert "badge_tough_nut" in keys

    def test_nemesis_requires_min_deaths(self):
        """Sous 3 morts, pas de badge Dur à cuire même si ratio > 2."""
        stats = _make_stats(kills_dealt=1, deaths_suffered=2)
        badges = compute_encounter_badges(stats)
        assert not any(b.label_key == "badge_tough_nut" for b in badges)

    def test_nemesis_zero_kills_dealt(self):
        """0 kills / 3+ morts → Dur à cuire quand même."""
        stats = _make_stats(kills_dealt=0, deaths_suffered=3)
        badges = compute_encounter_badges(stats)
        assert any(b.label_key == "badge_tough_nut" for b in badges)

    def test_ally_plus_badge(self):
        stats = _make_stats(winrate_as_ally=0.70, ally_count=3)
        badges = compute_encounter_badges(stats)
        assert any(b.label_key == "badge_ally_plus" for b in badges)

    def test_ally_plus_requires_min_ally_count(self):
        stats = _make_stats(winrate_as_ally=0.80, ally_count=1)
        badges = compute_encounter_badges(stats)
        assert not any(b.label_key == "badge_ally_plus" for b in badges)

    def test_ally_plus_requires_winrate_threshold(self):
        stats = _make_stats(winrate_as_ally=0.60, ally_count=5)
        badges = compute_encounter_badges(stats)
        assert not any(b.label_key == "badge_ally_plus" for b in badges)

    def test_coriace_badge(self):
        stats = _make_stats(winrate_vs_enemy=0.25, enemy_count=4)
        badges = compute_encounter_badges(stats)
        assert any(b.label_key == "badge_coriace" for b in badges)

    def test_coriace_requires_min_enemy_count(self):
        stats = _make_stats(winrate_vs_enemy=0.20, enemy_count=2)
        badges = compute_encounter_badges(stats)
        assert not any(b.label_key == "badge_coriace" for b in badges)

    def test_multiple_badges(self):
        """Un joueur peut cumuler Dur à cuire + Coriace."""
        stats = _make_stats(
            kills_dealt=1,
            deaths_suffered=5,
            winrate_vs_enemy=0.20,
            enemy_count=5,
        )
        badges = compute_encounter_badges(stats)
        keys = [b.label_key for b in badges]
        assert "badge_tough_nut" in keys
        assert "badge_coriace" in keys

    def test_badge_css_class(self):
        stats = _make_stats(kills_dealt=0, deaths_suffered=4)
        badges = compute_encounter_badges(stats)
        dur = next(b for b in badges if b.label_key == "badge_tough_nut")
        assert dur.css_class == "os-sb-td--worst"

    def test_ally_plus_css_class(self):
        stats = _make_stats(winrate_as_ally=0.70, ally_count=3)
        badges = compute_encounter_badges(stats)
        ally = next(b for b in badges if b.label_key == "badge_ally_plus")
        assert ally.css_class == "os-sb-td--best"


# ---------------------------------------------------------------------------
# filter_encounter_xuids
# ---------------------------------------------------------------------------


class TestFilterEncounterXuids:
    def _players(self):
        return [
            {"xuid": "xuid_me", "gamertag": "Me"},
            {"xuid": "xuid_friend1", "gamertag": "FriendA"},
            {"xuid": "xuid_enemy1", "gamertag": "Enemy1"},
            {"xuid": "xuid_enemy2", "gamertag": "Enemy2"},
        ]

    def test_excludes_self(self):
        result = filter_encounter_xuids(self._players(), "xuid_me", set())
        assert "xuid_me" not in result

    def test_excludes_friend_by_xuid(self):
        result = filter_encounter_xuids(self._players(), "xuid_me", {"xuid_friend1"})
        assert "xuid_friend1" not in result

    def test_excludes_friend_by_gamertag_casefold(self):
        result = filter_encounter_xuids(self._players(), "xuid_me", {"frienda"})
        assert "xuid_friend1" not in result

    def test_keeps_non_friends(self):
        result = filter_encounter_xuids(self._players(), "xuid_me", {"xuid_friend1"})
        assert "xuid_enemy1" in result
        assert "xuid_enemy2" in result

    def test_empty_players(self):
        result = filter_encounter_xuids([], "xuid_me", set())
        assert result == []

    def test_no_duplicates(self):
        players = [
            {"xuid": "xuid_a", "gamertag": "A"},
            {"xuid": "xuid_a", "gamertag": "A"},  # doublon
        ]
        result = filter_encounter_xuids(players, "xuid_me", set())
        assert result.count("xuid_a") == 1


# ---------------------------------------------------------------------------
# build_friends_set
# ---------------------------------------------------------------------------


class TestBuildFriendsSet:
    def test_csv_source_used_when_populated(self):
        result = build_friends_set("xuid_me", "xuid_friend1,xuid_friend2")
        assert result == {"xuid_friend1", "xuid_friend2"}

    def test_empty_csv_falls_back_to_json(self, tmp_path, monkeypatch):
        """Fallback vers friends_defaults.json quand CSV est vide."""
        friends_data = {"xuid_me": ["FriendFromJson"]}
        json_path = tmp_path / "friends_defaults.json"
        json_path.write_text(json.dumps(friends_data), encoding="utf-8")

        import src.ui.pages.match_view_encounters_logic as mod
        monkeypatch.setattr(mod, "_FRIENDS_JSON", json_path)

        result = build_friends_set("xuid_me", "")
        assert "FriendFromJson" in result

    def test_tilde_prefix_excluded(self, tmp_path, monkeypatch):
        """Les amis préfixés ~ sont exclus."""
        friends_data = {"xuid_me": ["~InactiveGT", "ActiveGT"]}
        json_path = tmp_path / "friends_defaults.json"
        json_path.write_text(json.dumps(friends_data), encoding="utf-8")

        import src.ui.pages.match_view_encounters_logic as mod
        monkeypatch.setattr(mod, "_FRIENDS_JSON", json_path)

        result = build_friends_set("xuid_me", None)
        assert "~InactiveGT" not in result
        assert "ActiveGT" in result

    def test_missing_json_returns_empty_on_fallback(self, tmp_path, monkeypatch):
        import src.ui.pages.match_view_encounters_logic as mod
        monkeypatch.setattr(mod, "_FRIENDS_JSON", tmp_path / "nonexistent.json")

        result = build_friends_set("xuid_me", None)
        assert result == set()


# ---------------------------------------------------------------------------
# _relative_date_fr
# ---------------------------------------------------------------------------


class TestRelativeDateFr:
    def _dt(self, days_ago: int) -> datetime:
        from datetime import timedelta
        return datetime.now(tz=timezone.utc) - timedelta(days=days_ago)

    def test_today(self):
        assert _relative_date_fr(self._dt(0)) == "aujourd'hui"

    def test_yesterday(self):
        assert _relative_date_fr(self._dt(1)) == "hier"

    def test_days(self):
        assert "j" in _relative_date_fr(self._dt(4))

    def test_weeks(self):
        assert "sem." in _relative_date_fr(self._dt(10))

    def test_months(self):
        assert "mois" in _relative_date_fr(self._dt(45))

    def test_years(self):
        result = _relative_date_fr(self._dt(400))
        assert "an" in result
