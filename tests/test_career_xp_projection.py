"""Tests des fonctions d'estimation et projection XP de la page Carrière.

Couvre :
- ``_compute_estimated_xp_curve`` : estimation de l'XP pré-sync
- ``_compute_active_xp_per_day``  : rythme XP/jour actif (hors inactivité)
- ``_compute_hero_projections``   : courbes de projection vers le rang Héros
- ``_load_other_players_histories`` : chargement historiques multi-joueurs
- ``_create_xp_history_chart``    : traces autres joueurs sur le graphe XP
"""

from __future__ import annotations

import unittest.mock as mock
from datetime import datetime, timedelta

import pytest

from src.ui.components.career_progress_circle import XP_HERO_TOTAL
from src.ui.pages.career_charts import (
    _OTHER_PLAYERS_COLORS,
    _create_xp_history_chart,
)
from src.ui.pages.career_data import (
    _load_other_players_histories,
)
from src.ui.pages.career_logic import (
    CAREER_XP_LAUNCH_DATE,
    DAILY_CHALLENGE_XP,
    INACTIVITY_GAP_DAYS,
    WEEKLY_CHALLENGE_XP,
    XP_BOOST_MULTIPLIER,
    _compute_active_xp_per_day,
    _compute_estimated_xp_curve,
    _compute_hero_projections,
)

# ── Helpers ──────────────────────────────────────────────────────────────────


def _make_history(
    snapshots: list[tuple[datetime, int, int]],  # (date, rank, xp_total)
) -> list[dict]:
    """Construit une liste de snapshots career_progression factices."""
    return [
        {
            "rank": rank,
            "rank_name": "Private",
            "rank_tier": "Gold",
            "current_xp": 100,
            "xp_for_next_rank": 1000,
            "xp_total": xp,
            "is_max_rank": False,
            "recorded_at": dt,
        }
        for dt, rank, xp in snapshots
    ]


# ── Tests _compute_active_xp_per_day ─────────────────────────────────────────


class TestComputeActiveXpPerDay:
    def test_histoire_vide(self):
        assert _compute_active_xp_per_day([]) == 0.0

    def test_un_seul_snapshot(self):
        history = _make_history([(datetime(2024, 1, 1), 10, 5000)])
        assert _compute_active_xp_per_day(history) == 0.0

    def test_rythme_constant(self):
        """7000 XP gagnés en 7 jours continus → 1000 XP/jour."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 0),
                (datetime(2024, 1, 8), 20, 7000),
            ]
        )
        result = _compute_active_xp_per_day(history)
        assert result == pytest.approx(1000.0, rel=1e-3)

    def test_rythme_plusieurs_snapshots(self):
        """3 snapshots réguliers espacés de 7j : 10000 XP / 14 jours actifs = ~714 XP/j."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 0),
                (datetime(2024, 1, 8), 20, 5000),
                (datetime(2024, 1, 15), 30, 10000),
            ]
        )
        result = _compute_active_xp_per_day(history)
        assert result == pytest.approx(10000 / 14.0, rel=1e-3)

    def test_gap_inactivite_superieur_au_seuil(self):
        """Un gap de 30j est remplacé par INACTIVITY_GAP_DAYS/2 = 7j dans le calcul."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 0),
                (datetime(2024, 1, 31), 20, 7000),  # gap 30j > 14j
            ]
        )
        result = _compute_active_xp_per_day(history)
        # 7000 XP / (INACTIVITY_GAP_DAYS/2) jours = 7000 / 7 = 1000
        expected = 7000.0 / (INACTIVITY_GAP_DAYS / 2)
        assert result == pytest.approx(expected, rel=1e-3)

    def test_historique_xp_negatif_ou_nul(self):
        """Si l'XP n'a pas augmenté (ex: rechargement du même snapshot), retourner 0."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 5000),
                (datetime(2024, 1, 8), 10, 5000),
            ]
        )
        assert _compute_active_xp_per_day(history) == 0.0


# ── Tests _compute_estimated_xp_curve ────────────────────────────────────────


class TestComputeEstimatedXpCurve:
    def _run(self, history, pre_sync_dates):
        """Appelle directement _compute_estimated_xp_curve (plus de mock DB)."""
        return _compute_estimated_xp_curve(history, pre_sync_dates)

    def test_pas_de_matchs_pre_sync(self):
        history = _make_history(
            [
                (datetime(2024, 2, 1), 10, 5000),
                (datetime(2024, 3, 1), 20, 10000),
            ]
        )
        result = self._run(history, [])
        assert result == []

    def test_history_vide(self):
        """Historique vide → pas d'estimation possible."""
        result = self._run([], [datetime(2024, 1, 15), datetime(2024, 1, 20)])
        assert result == []

    def test_courbe_relie_premier_snapshot(self):
        """Le dernier point de la courbe estimée doit être le 1er snapshot réel."""
        d0 = datetime(2024, 2, 1)
        d1 = datetime(2024, 3, 1)
        history = _make_history([(d0, 10, 5000), (d1, 20, 15000)])
        pre_sync_dates = [datetime(2024, 1, 10), datetime(2024, 1, 20)]
        result = self._run(history, pre_sync_dates)
        assert len(result) >= 1
        # Le dernier point doit raccorder au 1er snapshot (date d0, xp 5000)
        last_date, last_xp = result[-1]
        assert last_date == d0
        assert last_xp == 5000

    def test_xp_ne_descend_pas_en_dessous_de_zero(self):
        """L'XP estimé ne doit jamais être négatif."""
        d0 = datetime(2024, 2, 1)
        d1 = datetime(2024, 3, 1)
        # XP très faible au 1er snapshot : la remontée va saturer à 0
        history = _make_history([(d0, 2, 100), (d1, 10, 10100)])
        # Beaucoup de matchs pré-sync pour vite tomber à 0
        pre_sync_dates = [datetime(2024, 1, 1) + timedelta(days=i) for i in range(20)]
        result = self._run(history, pre_sync_dates)
        xp_values = [pt[1] for pt in result]
        assert all(v >= 0 for v in xp_values)

    def test_dates_chronologiques(self):
        """La courbe retournée doit être en ordre chronologique."""
        d0 = datetime(2024, 2, 1)
        d1 = datetime(2024, 3, 1)
        history = _make_history([(d0, 10, 5000), (d1, 20, 15000)])
        pre_sync_dates = [
            datetime(2024, 1, 10),
            datetime(2024, 1, 15),
            datetime(2024, 1, 25),
        ]
        result = self._run(history, pre_sync_dates)
        dates = [pt[0] for pt in result]
        assert dates == sorted(dates)

    def test_matchs_avant_lancement_exclus(self):
        """Les matchs antérieurs au 20/06/2023 ne doivent pas figurer dans la courbe."""
        d0 = datetime(2024, 2, 1)
        history = _make_history([(d0, 10, 3000)])
        avant_lancement = datetime(2022, 11, 15)  # Halo Infinite sorti avant les rangs
        apres_lancement = datetime(2023, 9, 1)  # après le 20/06/2023
        pre_sync_dates = [avant_lancement, apres_lancement]

        result = self._run(history, pre_sync_dates)

        dates = [pt[0] for pt in result]
        assert avant_lancement not in dates
        for d, _ in result[:-1]:  # exclure le point de raccord final
            assert d.date() >= CAREER_XP_LAUNCH_DATE.date()

    def test_tous_matchs_avant_lancement_retourne_vide(self):
        """Si tous les matchs pré-sync sont antérieurs au 20/06/2023, retourner []."""
        d0 = datetime(2024, 2, 1)
        history = _make_history([(d0, 10, 5000)])
        pre_sync_dates = [
            datetime(2022, 1, 1),
            datetime(2022, 11, 20),
            datetime(2023, 3, 15),
        ]
        result = self._run(history, pre_sync_dates)
        assert result == []


# ── Tests _compute_hero_projections ──────────────────────────────────────────


class TestComputeHeroProjections:
    BASE_DATE = datetime(2024, 6, 1)
    # XP quelconque sous le seuil Hero
    BASE_XP = 5_000_000
    # Rythme suffisant pour atteindre Hero en < 10 ans
    # 4 319 350 XP restants / 2000 XP/j = ~2160 jours ≈ 5.9 ans
    FAST_XP_PER_DAY = 2000.0

    def test_rythme_nul_retourne_vide(self):
        normal, optimistic = _compute_hero_projections(self.BASE_XP, self.BASE_DATE, 0.0)
        assert normal == []
        assert optimistic == []

    def test_deja_hero_retourne_vide(self):
        normal, optimistic = _compute_hero_projections(XP_HERO_TOTAL, self.BASE_DATE, 1000.0)
        assert normal == []
        assert optimistic == []

    def test_point_depart_est_xp_actuel(self):
        """Les deux courbes commencent à l'XP actuel du joueur."""
        normal, _ = _compute_hero_projections(self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY)
        assert normal[0][1] == self.BASE_XP
        assert normal[0][0] == self.BASE_DATE

    def test_projection_atteint_hero(self):
        """Les deux courbes doivent se terminer à XP_HERO_TOTAL (rythme suffisant)."""
        normal, optimistic = _compute_hero_projections(
            self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY
        )
        assert normal[-1][1] == XP_HERO_TOTAL
        assert optimistic[-1][1] == XP_HERO_TOTAL

    def test_optimiste_plus_rapide_que_normale(self):
        """La projection optimiste doit atteindre Hero avant la normale."""
        normal, optimistic = _compute_hero_projections(
            self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY
        )
        assert optimistic[-1][0] < normal[-1][0]

    def test_courbes_chronologiques(self):
        """Les points de chaque courbe sont ordonnés dans le temps."""
        # Utiliser un rythme rapide pour rester dans le cap 10 ans
        normal, optimistic = _compute_hero_projections(
            self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY
        )
        normal_dates = [pt[0] for pt in normal]
        opt_dates = [pt[0] for pt in optimistic]
        assert normal_dates == sorted(normal_dates)
        assert opt_dates == sorted(opt_dates)

    def test_courbes_chronologiques_avec_cap(self):
        """Même avec le cap 10 ans (Hero non atteint), l'ordre est préservé."""
        # 1 XP/jour → 9M+ jours, cap forcé
        normal, optimistic = _compute_hero_projections(100, self.BASE_DATE, 1.0)
        if normal:
            normal_dates = [pt[0] for pt in normal]
            assert normal_dates == sorted(normal_dates)
        if optimistic:
            opt_dates = [pt[0] for pt in optimistic]
            assert opt_dates == sorted(opt_dates)

    def test_optimiste_integre_challenges_et_boost(self):
        """Vérifie que le rythme optimiste = (xp_day + hebdo/7 + quotidien) × boost."""
        xp_per_day = self.FAST_XP_PER_DAY
        challenge_per_day = WEEKLY_CHALLENGE_XP / 7.0 + DAILY_CHALLENGE_XP
        expected_opt_rate = (xp_per_day + challenge_per_day) * XP_BOOST_MULTIPLIER

        xp_remaining = XP_HERO_TOTAL - self.BASE_XP
        expected_days = xp_remaining / expected_opt_rate

        normal, optimistic = _compute_hero_projections(self.BASE_XP, self.BASE_DATE, xp_per_day)
        actual_days = (optimistic[-1][0] - self.BASE_DATE).total_seconds() / 86400.0
        # ±7 jours de tolérance (granularité hebdomadaire)
        assert abs(actual_days - expected_days) <= 7

    def test_cap_10_ans(self):
        """Un rythme très faible ne doit pas générer une courbe > 10 ans."""
        normal, optimistic = _compute_hero_projections(
            100,
            self.BASE_DATE,
            1.0,  # 1 XP/jour → 9M+ jours sans le cap
        )
        # Avec le cap à 365*10 jours, la courbe créée doit rester bornée
        if normal:
            max_days = (normal[-1][0] - self.BASE_DATE).total_seconds() / 86400.0
            assert max_days <= 365 * 10 + 8  # +8j : arrondi hebdomadaire


# ── Constantes ───────────────────────────────────────────────────────────────


class TestConstants:
    def test_weekly_challenge_xp(self):
        """950 XP = 4×50 + 3×100 + 3×150 (Normal + Heroic + Legendary post-CU32)."""
        assert WEEKLY_CHALLENGE_XP == 4 * 50 + 3 * 100 + 3 * 150

    def test_daily_challenge_xp(self):
        """500 XP par défi quotidien (source : Halopedia)."""
        assert DAILY_CHALLENGE_XP == 500

    def test_xp_boost_multiplier(self):
        assert XP_BOOST_MULTIPLIER == 2.0

    def test_inactivity_gap_days(self):
        assert INACTIVITY_GAP_DAYS == 14


# ── Tests _OTHER_PLAYERS_COLORS ───────────────────────────────────────────────


class TestOtherPlayersColors:
    def test_six_couleurs(self):
        """La palette doit contenir au moins 6 couleurs distinctes."""
        assert len(_OTHER_PLAYERS_COLORS) >= 6

    def test_couleurs_uniques(self):
        """Pas de doublon dans la palette."""
        assert len(_OTHER_PLAYERS_COLORS) == len(set(_OTHER_PLAYERS_COLORS))

    def test_pas_de_conflit_avec_traces_fixes(self):
        """Les couleurs ne doivent pas chevaucher les traces prédéfinies."""
        reserved = {"#CE93D8", "#FFA726", "#66BB6A"}  # estimé, proj, optimiste
        for color in _OTHER_PLAYERS_COLORS:
            assert color.upper() not in {c.upper() for c in reserved}


# ── Tests _load_other_players_histories ───────────────────────────────────────


XUID_CURRENT = "xuid_current"
XUID_OTHER_A = "xuid_player_a"
XUID_OTHER_B = "xuid_player_b"

_PROFILES_MOCK = {
    "PlayerCurrent": {"xuid": XUID_CURRENT, "db_path": "/db/current.duckdb"},
    "PlayerA": {"xuid": XUID_OTHER_A, "db_path": "/db/player_a.duckdb"},
    "PlayerB": {"xuid": XUID_OTHER_B, "db_path": "/db/player_b.duckdb"},
}

_HIST_A = [
    {
        "rank": 10,
        "rank_name": "X",
        "rank_tier": "G",
        "current_xp": 0,
        "xp_for_next_rank": 1000,
        "xp_total": 1000,
        "is_max_rank": False,
        "recorded_at": datetime(2024, 1, 1),
    },
    {
        "rank": 20,
        "rank_name": "X",
        "rank_tier": "G",
        "current_xp": 0,
        "xp_for_next_rank": 1000,
        "xp_total": 5000,
        "is_max_rank": False,
        "recorded_at": datetime(2024, 2, 1),
    },
]
_HIST_B = [
    {
        "rank": 5,
        "rank_name": "X",
        "rank_tier": "G",
        "current_xp": 0,
        "xp_for_next_rank": 1000,
        "xp_total": 500,
        "is_max_rank": False,
        "recorded_at": datetime(2024, 1, 15),
    },
    {
        "rank": 15,
        "rank_name": "X",
        "rank_tier": "G",
        "current_xp": 0,
        "xp_for_next_rank": 1000,
        "xp_total": 2000,
        "is_max_rank": False,
        "recorded_at": datetime(2024, 3, 1),
    },
]


class TestLoadOtherPlayersHistories:
    def _run(self, histories_by_xuid: dict, existing_paths: set | None = None):
        """Helper : mocke load_profiles, os.path.exists et _load_career_history."""
        if existing_paths is None:
            existing_paths = {"/db/player_a.duckdb", "/db/player_b.duckdb"}

        def fake_history(db_path, xuid, limit=50):
            return histories_by_xuid.get(xuid, [])

        with (
            mock.patch("src.utils.profiles.load_profiles", return_value=_PROFILES_MOCK),
            mock.patch("src.ui.pages.career_data._load_career_history", side_effect=fake_history),
            mock.patch("os.path.exists", side_effect=lambda p: p in existing_paths),
        ):
            return _load_other_players_histories(XUID_CURRENT)

    def test_exclut_joueur_actuel(self):
        """Le joueur dont le XUID correspond à current_xuid est ignoré."""
        results = self._run({XUID_OTHER_A: _HIST_A, XUID_OTHER_B: _HIST_B})
        gamertags = [r["gamertag"] for r in results]
        assert "PlayerCurrent" not in gamertags

    def test_retourne_autres_joueurs(self):
        """Retourne les profils avec au moins 2 snapshots."""
        results = self._run({XUID_OTHER_A: _HIST_A, XUID_OTHER_B: _HIST_B})
        assert len(results) == 2
        gamertags = {r["gamertag"] for r in results}
        assert gamertags == {"PlayerA", "PlayerB"}

    def test_ignore_db_inexistante(self):
        """Un profil dont le fichier DB n'existe pas est ignoré."""
        results = self._run(
            {XUID_OTHER_A: _HIST_A, XUID_OTHER_B: _HIST_B},
            existing_paths={"/db/player_a.duckdb"},  # player_b absent
        )
        gamertags = [r["gamertag"] for r in results]
        assert "PlayerA" in gamertags
        assert "PlayerB" not in gamertags

    def test_ignore_historique_insuffisant(self):
        """Un profil avec moins de 2 snapshots est ignoré."""
        hist_court = [_HIST_A[0]]  # 1 seul snapshot
        results = self._run({XUID_OTHER_A: hist_court, XUID_OTHER_B: _HIST_B})
        gamertags = [r["gamertag"] for r in results]
        assert "PlayerA" not in gamertags
        assert "PlayerB" in gamertags

    def test_retourne_liste_vide_si_exception(self):
        """Une exception dans load_profiles → liste vide, pas de crash."""
        with mock.patch("src.utils.profiles.load_profiles", side_effect=RuntimeError("boom")):
            result = _load_other_players_histories(XUID_CURRENT)
        assert result == []

    def test_structure_retournee(self):
        """Chaque élément contient 'gamertag' et 'history'."""
        results = self._run({XUID_OTHER_A: _HIST_A})
        assert len(results) == 1
        assert "gamertag" in results[0]
        assert "history" in results[0]
        assert results[0]["gamertag"] == "PlayerA"
        assert results[0]["history"] == _HIST_A


# ── Tests _create_xp_history_chart (traces autres joueurs) ────────────────────


class TestCreateXpHistoryChartOtherPlayers:
    BASE_HISTORY = [
        {
            "rank": 10,
            "rank_name": "Private",
            "rank_tier": "Bronze",
            "current_xp": 100,
            "xp_for_next_rank": 1000,
            "xp_total": 10000,
            "is_max_rank": False,
            "recorded_at": datetime(2024, 1, 1),
        },
        {
            "rank": 20,
            "rank_name": "Corporal",
            "rank_tier": "Silver",
            "current_xp": 200,
            "xp_for_next_rank": 2000,
            "xp_total": 50000,
            "is_max_rank": False,
            "recorded_at": datetime(2024, 6, 1),
        },
    ]

    def test_sans_autres_joueurs_pas_de_traces_supplementaires(self):
        """Sans other_players, seule la trace du joueur principal existe."""
        fig = _create_xp_history_chart(self.BASE_HISTORY)
        assert fig is not None
        assert len(fig.data) == 1

    def test_avec_un_autre_joueur(self):
        """Avec 1 autre joueur, 2 traces au total."""
        fig = _create_xp_history_chart(
            self.BASE_HISTORY,
            other_players=[{"gamertag": "PlayerA", "history": _HIST_A}],
        )
        assert fig is not None
        assert len(fig.data) == 2

    def test_avec_deux_autres_joueurs(self):
        """Avec 2 autres joueurs, 3 traces au total."""
        fig = _create_xp_history_chart(
            self.BASE_HISTORY,
            other_players=[
                {"gamertag": "PlayerA", "history": _HIST_A},
                {"gamertag": "PlayerB", "history": _HIST_B},
            ],
        )
        assert fig is not None
        assert len(fig.data) == 3

    def test_traces_autres_joueurs_masquees_par_defaut(self):
        """Les traces des autres joueurs doivent être visible='legendonly'."""
        fig = _create_xp_history_chart(
            self.BASE_HISTORY,
            other_players=[{"gamertag": "PlayerA", "history": _HIST_A}],
        )
        # La première trace est le joueur principal (visible), la seconde masquée
        assert fig.data[0].visible is None or fig.data[0].visible is True
        assert fig.data[1].visible == "legendonly"

    def test_nom_trace_contient_gamertag(self):
        """Le nom de la trace doit contenir le gamertag du joueur."""
        fig = _create_xp_history_chart(
            self.BASE_HISTORY,
            other_players=[{"gamertag": "PlayerA", "history": _HIST_A}],
        )
        assert "PlayerA" in fig.data[1].name

    def test_couleurs_cycliques_palette(self):
        """7 joueurs → les couleurs bouclent sur la palette."""
        players = [{"gamertag": f"P{i}", "history": _HIST_A} for i in range(7)]
        fig = _create_xp_history_chart(self.BASE_HISTORY, other_players=players)
        assert fig is not None
        # La 7ème trace reprend la couleur de la 1ère
        color_1 = fig.data[1].line.color
        color_7 = fig.data[7].line.color
        assert color_1 == color_7

    def test_liste_vide_equivalent_a_none(self):
        """other_players=[] produit le même résultat que other_players=None."""
        fig_none = _create_xp_history_chart(self.BASE_HISTORY, other_players=None)
        fig_empty = _create_xp_history_chart(self.BASE_HISTORY, other_players=[])
        assert len(fig_none.data) == len(fig_empty.data)
