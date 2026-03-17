"""Tests du scoreboard extensible inline."""

from __future__ import annotations

from unittest.mock import patch


def _sample_scoreboard_players() -> list[dict[str, object]]:
    return [
        {
            "xuid": "2533274790000001",
            "gamertag": "Alpha",
            "team_id": 0,
            "rank": 1,
            "score": 2200,
            "kills": 15,
            "deaths": 9,
            "assists": 7,
            "kda": 1.44,
            "top_weapon_id": None,
            "max_killing_spree": 5,
            "headshot_kills": 4,
            "perfect_kills": 1,
            "shots_fired": 240,
            "shots_hit": 118,
            "accuracy": 0.492,
            "melee_kills": 2,
            "power_weapon_kills": 1,
            "damage_dealt": 4250,
            "damage_taken": 3560,
            "avg_life_seconds": 52,
        },
        {
            "xuid": "2533274790000002",
            "gamertag": "Bravo",
            "team_id": 1,
            "rank": 2,
            "score": 1800,
            "kills": 11,
            "deaths": 13,
            "assists": 3,
            "kda": 0.85,
            "top_weapon_id": None,
            "max_killing_spree": 2,
            "headshot_kills": 1,
            "perfect_kills": 0,
            "shots_fired": 210,
            "shots_hit": 88,
            "accuracy": 0.419,
            "melee_kills": 1,
            "power_weapon_kills": 0,
            "damage_dealt": 3010,
            "damage_taken": 4110,
            "avg_life_seconds": 34,
        },
    ]


def test_render_match_scoreboard_emits_expandable_detail_rows(mock_st) -> None:
    from src.ui.pages import match_view_scoreboard as mod
    from src.ui.pages.match_view_scoreboard_detail import MedalDetailItem, ScoreboardPlayerExtraData

    ms = mock_st(mod)
    extra = ScoreboardPlayerExtraData(
        weapons=[("BR75", 8)],
        medals=[
            MedalDetailItem(
                name="Perfection",
                count=1,
                icon_url="/app/static/medals/icons/1512363953.png",
            )
        ],
        antagonist_items=[("Némésis", "Bravo (3)")],
        citations=[("Tireur d'elite", 2)],
        performance_score=87.5,
        had_bot_teammate=True,
        has_local_db=True,
        profile_link_html="<a class='os-sb-detail-link' href='/?page=Explorer&amp;gamertag=Alpha'>Explorer les matchs avec Alpha</a>",
    )

    with (
        patch.object(mod, "load_match_scoreboard", return_value=_sample_scoreboard_players()),
        patch.object(mod, "load_scoreboard_player_extra_data", return_value=extra),
    ):
        mod.render_match_scoreboard(
            match_id="match-123",
            db_path="dummy.duckdb",
            xuid="2533274790000001",
            db_key=None,
            load_match_gamertags_fn=lambda *_a, **_kw: {},
        )

    rendered_html = "\n".join(call.args[0] for call in ms.calls["markdown"].call_args_list)
    assert "os-sb-toggle" in rendered_html
    assert "os-sb-detail-row" in rendered_html
    assert "os-sb-detail-panel" in rendered_html
    assert "os-sb-team os-sb-team--mine" in rendered_html
    assert "Équipe Eagle" in rendered_html
    assert "BR75" in rendered_html
    assert "Perfection" in rendered_html
    assert "Antagoniste" in rendered_html
    assert "Bravo (3)" in rendered_html
    assert "/app/static/medals/icons/1512363953.png" in rendered_html
    assert "os-sb-detail-medal-icon" in rendered_html
    assert "Tireur d&#x27;elite" in rendered_html
    assert "Explorer les matchs avec Alpha" in rendered_html
    assert "Résumé" not in rendered_html
    ms.calls["caption"].assert_called()


def test_render_scoreboard_player_detail_html_handles_shared_only() -> None:
    from src.ui.pages.match_view_scoreboard_detail import (
        MedalDetailItem,
        ScoreboardPlayerExtraData,
        render_scoreboard_player_detail_html,
    )

    html_out = render_scoreboard_player_detail_html(
        extra=ScoreboardPlayerExtraData(
            weapons=[("Needler", 4)],
            medals=[
                MedalDetailItem(
                    name="Perfection",
                    count=1,
                    icon_url="/app/static/medals/icons/1512363953.png",
                )
            ],
        ),
    )

    assert "Needler" in html_out
    assert "/app/static/medals/icons/1512363953.png" in html_out
    assert "Données shared uniquement" in html_out
    assert "os-sb-detail-badge" in html_out
    assert "Explorer les matchs avec" not in html_out
    assert "Résumé" not in html_out


def test_render_scoreboard_player_detail_html_renders_antagonist_section() -> None:
    from src.ui.pages.match_view_scoreboard_detail import (
        ScoreboardPlayerExtraData,
        render_scoreboard_player_detail_html,
    )

    html_out = render_scoreboard_player_detail_html(
        extra=ScoreboardPlayerExtraData(
            antagonist_items=[
                ("Némésis", "Bravo (3)"),
                ("Souffre-douleur", "Charlie (2)"),
            ]
        ),
    )

    assert "Antagoniste" in html_out
    assert "Némésis" in html_out
    assert "Bravo (3)" in html_out
    assert "Souffre-douleur" in html_out
    assert "Charlie (2)" in html_out


def test_render_scoreboard_player_detail_html_colors_performance_score_value() -> None:
    from src.ui.components.performance import get_score_class
    from src.ui.pages.match_view_scoreboard_detail import (
        ScoreboardPlayerExtraData,
        render_scoreboard_player_detail_html,
    )

    html_out = render_scoreboard_player_detail_html(
        extra=ScoreboardPlayerExtraData(
            performance_score=87.5,
            has_local_db=True,
        ),
    )

    assert f"os-sb-detail-item-value {get_score_class(87.5)}" in html_out
    assert ">87.5<" in html_out


def test_scoreboard_styles_define_inline_expansion_rules() -> None:
    from pathlib import Path

    css_path = Path("static/styles.css")
    css_text = css_path.read_text(encoding="utf-8")

    assert ".os-sb-toggle" in css_text
    assert "tbody.os-sb-player:has(.os-sb-toggle:checked) .os-sb-detail-row" in css_text
    assert ".os-sb-detail-medal-icon" in css_text
    assert "font-size: 0.68em;" in css_text
