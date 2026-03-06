"""Algorithme TrueSkill 2 adapté pour le LUSR (LevelUp Skill Rank).

Ce module calcule un rating de compétence absolu par match en traitant
les matchs séquentiellement dans l'ordre chronologique. Chaque match met
à jour (mu, sigma) en fonction d'un score composite continu [0,1] et de
la force estimée de l'équipe adverse (depuis kills_expected de tous les
participants dans match_participants).

Modules internes :
- _trueskill_math : PlayerState, fonctions gaussiennes, update TrueSkill, decay
- _composite      : score composite continu [0,1]

Usage:
    from src.analysis.skill_rating import compute_skill_ratings_batch

    result_df = compute_skill_ratings_batch(df_player_matches, df_all_participants)
    # result_df colonnes : match_id, rating_value, rating_deviation, playlist_group
"""

from __future__ import annotations

import math
from typing import Any

import polars as pl

from src.analysis._composite import (
    _sigmoid_ratio,  # noqa: F401 — réexport
    compute_composite_score,
)
from src.analysis._trueskill_math import (
    PlayerState,
    apply_inactivity_decay,
    trueskill_update,
)
from src.analysis.skill_rating_config import (
    ACCURACY_HISTORY_SIZE,
    DEFAULT_OPPONENT_SIGMA,
    INDIVIDUAL_MU_ALPHA,
    INITIAL_MU,
    MIN_MATCHES_FOR_ACCURACY_DELTA,
    MIN_SIGMA,
    PLAYLIST_GROUPS,
    get_playlist_group,
)
from src.utils.safe_types import clamp as _clamp
from src.utils.safe_types import safe_float as _safe_float

# Réexports publics
__all__ = [
    "PlayerState",
    "apply_inactivity_decay",
    "compute_composite_score",
    "compute_enemy_strength",
    "compute_skill_ratings_batch",
    "estimate_individual_mu",
    "trueskill_update",
]


# =============================================================================
# Estimation μ individuel depuis kills_expected
# =============================================================================


def estimate_individual_mu(
    kills_expected: float,
    match_avg_ke: float,
    match_std_ke: float,
    base_mu: float,
) -> float:
    """Estime le μ TrueSkill d'un joueur depuis son kills_expected dans un match.

    Principe : kills_expected est une fonction monotone de μ dans un contexte donné.
    On l'utilise comme proxy normalisé pour estimer la force relative du joueur.

    Args:
        kills_expected: kills_expected du joueur dans ce match.
        match_avg_ke: Moyenne des kills_expected de tous les joueurs du match.
        match_std_ke: Écart-type des kills_expected du match.
        base_mu: Rating de référence (typiquement INITIAL_MU).

    Returns:
        μ estimé du joueur.
    """
    if match_std_ke < 1e-6:
        return base_mu
    z_score = (kills_expected - match_avg_ke) / match_std_ke
    return base_mu + INDIVIDUAL_MU_ALPHA * z_score


def compute_enemy_strength(
    enemy_rows: list[dict[str, Any]],
    match_avg_ke: float,
    match_std_ke: float,
    player_mu: float = INITIAL_MU,
) -> tuple[float, float]:
    """Calcule la force estimée de l'équipe adverse (mu_opp, sigma_opp).

    Si kills_expected est disponible pour les adversaires, on utilise
    estimate_individual_mu(). Sinon, on retourne les valeurs par défaut.

    Le paramètre player_mu ancre l'estimation de l'équipe adverse sur le rating
    courant du joueur suivi. Le matchmaking de Halo met tendanciellement des joueurs
    de niveau similaire → l'adversaire moyen est estimé à player_mu, et les
    déviations kills_expected indiquent les individus plus forts/faibles.

    Args:
        enemy_rows: Lignes match_participants des adversaires.
        match_avg_ke: Moyenne kills_expected du match entier.
        match_std_ke: Écart-type kills_expected du match entier.
        player_mu: Rating courant du joueur suivi (ancre l'estimation adversaire).

    Returns:
        Tuple (avg_mu_enemies, avg_sigma_enemies).
    """
    if not enemy_rows:
        return player_mu, DEFAULT_OPPONENT_SIGMA

    mu_estimates = []
    for row in enemy_rows:
        ke = _safe_float(row.get("kills_expected"))
        if ke is not None:
            mu_est = estimate_individual_mu(ke, match_avg_ke, match_std_ke, player_mu)
            mu_estimates.append(mu_est)

    if not mu_estimates:
        return player_mu, DEFAULT_OPPONENT_SIGMA

    avg_mu = sum(mu_estimates) / len(mu_estimates)
    # Variance des estimations → sigma de l'adversaire
    if len(mu_estimates) > 1:
        variance = sum((m - avg_mu) ** 2 for m in mu_estimates) / len(mu_estimates)
        sigma = _clamp(math.sqrt(variance) + MIN_SIGMA, MIN_SIGMA, DEFAULT_OPPONENT_SIGMA)
    else:
        sigma = DEFAULT_OPPONENT_SIGMA

    return avg_mu, sigma


# =============================================================================
# Agrégation des participants par équipe
# =============================================================================


def _compute_match_ke_stats(
    match_participants: list[dict[str, Any]],
) -> tuple[float, float]:
    """Calcule moyenne et écart-type de kills_expected pour un match."""
    ke_values = [
        ke
        for row in match_participants
        if (ke := _safe_float(row.get("kills_expected"))) is not None
    ]
    if not ke_values:
        return INITIAL_MU, 1.0
    n = len(ke_values)
    avg = sum(ke_values) / n
    if n < 2:
        return avg, 1.0
    variance = sum((k - avg) ** 2 for k in ke_values) / n
    std = math.sqrt(variance) if variance > 0 else 1.0
    return avg, std


def _split_participants(
    player_team_id: int | None,
    match_participants: list[dict[str, Any]],
) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    """Sépare les participants en coéquipiers et adversaires.

    Args:
        player_team_id: team_id du joueur suivi.
        match_participants: Tous les participants du match.

    Returns:
        Tuple (teammates, enemies).
        Note : le joueur principal n'est PAS filtré des coéquipiers
        (son xuid n'est pas connu ici).
    """
    if player_team_id is None:
        return [], match_participants

    teammates = [r for r in match_participants if r.get("team_id") == player_team_id]
    enemies = [r for r in match_participants if r.get("team_id") != player_team_id]
    return teammates, enemies


# =============================================================================
# Point d'entrée principal
# =============================================================================


def compute_skill_ratings_batch(  # noqa: C901, PLR0912, PLR0915
    df_matches: pl.DataFrame,
    df_all_participants: pl.DataFrame,
    *,
    existing_states: dict[str, PlayerState] | None = None,
    weights: dict[str, float] | None = None,
) -> pl.DataFrame:
    """Calcule skill_rating et skill_deviation pour chaque match du joueur.

    IMPORTANT : df_matches DOIT être trié par start_time ASC.
    Le traitement est séquentiel — chaque match dépend du précédent.

    Chaque playlist_group maintient son propre état TrueSkill (mu, sigma)
    indépendant. Un match dans "ranked" ne change pas le rating "arena".

    Args:
        df_matches: DataFrame Polars avec colonnes requises :
            match_id, start_time, outcome, kills, deaths,
            kills_expected, deaths_expected, enemy_mmr,
            damage_dealt, damage_taken, accuracy,
            team_id, playlist_name, pair_name, is_ranked
        df_all_participants: DataFrame de TOUS les participants des matchs :
            match_id, xuid, team_id, kills_expected, deaths_expected
            (pour estimation μ des adversaires)
        existing_states: États précédents du joueur par groupe pour le mode
            incrémental. Clés = playlist_group (ex: "ranked", "arena").
            Si None ou groupe absent, démarre avec les valeurs initiales.

    Returns:
        DataFrame Polars avec colonnes :
            match_id (str), rating_value (float), rating_deviation (float),
            playlist_group (str)
    """
    if df_matches.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "rating_value": pl.Float64,
                "rating_deviation": pl.Float64,
                "playlist_group": pl.Utf8,
            }
        )

    # Tri défensif : garantit l'ordre chronologique même si l'appelant ne l'a pas fait.
    if "start_time" in df_matches.columns:
        df_matches = df_matches.sort("start_time")

    # États par groupe — créés à la première occurrence via setdefault()
    states: dict[str, PlayerState] = existing_states or {}

    # Préparer l'index des participants par match_id
    participants_by_match: dict[str, list[dict[str, Any]]] = {}
    if not df_all_participants.is_empty():
        for row in df_all_participants.iter_rows(named=True):
            mid = row.get("match_id")
            if mid:
                participants_by_match.setdefault(mid, []).append(row)

    match_ids: list[str] = []
    ratings: list[float] = []
    deviations: list[float] = []
    groups: list[str] = []

    for row in df_matches.iter_rows(named=True):
        match_id = row.get("match_id", "")
        start_time = row.get("start_time")
        playlist_name = row.get("playlist_name")
        pair_name = row.get("pair_name")

        # ── Groupe playlist ──
        group = get_playlist_group(
            str(playlist_name) if playlist_name else None,
            str(pair_name) if pair_name else None,
        )
        weight_factor = PLAYLIST_GROUPS[group].weight_factor if group in PLAYLIST_GROUPS else 1.0

        # ── État per-groupe (créé si absent) ──
        state = states.setdefault(group, PlayerState())

        # ── Dérive d'inactivité (per-groupe) ──
        if state.last_match_time is not None and start_time is not None:
            try:
                delta = start_time - state.last_match_time
                days = delta.total_seconds() / 86400.0
                state.sigma = apply_inactivity_decay(state.sigma, days)
            except (AttributeError, TypeError):
                pass

        # ── Participants du match ──
        all_participants = participants_by_match.get(match_id, [])
        match_avg_ke, match_std_ke = _compute_match_ke_stats(all_participants)

        player_team_id = row.get("team_id")
        try:
            player_team_id = int(player_team_id) if player_team_id is not None else None
        except (ValueError, TypeError):
            player_team_id = None

        # Séparer coéquipiers et adversaires via team_id
        teammates, enemies = _split_participants(player_team_id, all_participants)

        # ── Force adversaire (ancrée sur state.mu — matchmaking similaire) ──
        # Ceci maintient t ≈ 0 et stabilise le système TrueSkill :
        # la zone draw (0.45-0.55) donne v_draw≈0, seules les vraies sur/sous-
        # performances (<0.45 ou >0.55) modifient le rating.
        mu_opp, sigma_opp = compute_enemy_strength(
            enemies, match_avg_ke, match_std_ke, player_mu=state.mu
        )

        # ── Précision moyenne historique (per-groupe) ──
        avg_acc = None
        if len(state.accuracy_history) >= MIN_MATCHES_FOR_ACCURACY_DELTA:
            avg_acc = sum(state.accuracy_history) / len(state.accuracy_history)

        # ── Efficacité dégâts moyenne historique (per-groupe) ──
        avg_damage_eff = None
        if len(state.damage_eff_history) >= MIN_MATCHES_FOR_ACCURACY_DELTA:
            avg_damage_eff = sum(state.damage_eff_history) / len(state.damage_eff_history)

        # ── kills_expected moyen des coéquipiers / adversaires ──
        teammate_kes = [
            ke for t in teammates if (ke := _safe_float(t.get("kills_expected"))) is not None
        ]
        teammate_avg_ke = sum(teammate_kes) / len(teammate_kes) if teammate_kes else None

        enemy_kes = [
            ke for e in enemies if (ke := _safe_float(e.get("kills_expected"))) is not None
        ]
        enemy_avg_ke = sum(enemy_kes) / len(enemy_kes) if enemy_kes else None

        # ── Guard : match sans résultat (outcome=None) ──
        # Un match sans outcome est un match fantôme (crash serveur, lobby avorté).
        # On maintient le state (decay inactivité + last_match_time) mais on ne
        # modifie pas mu/sigma pour ne pas polluer le rating avec un résultat invalide.
        raw_outcome = row.get("outcome")
        if raw_outcome is None:
            state.match_count += 1
            state.last_match_time = start_time
            match_ids.append(str(match_id))
            ratings.append(round(state.mu, 1))
            deviations.append(round(state.sigma, 1))
            groups.append(group)
            continue

        # ── Score composite ──
        composite = compute_composite_score(
            row,
            avg_acc,
            teammate_avg_ke,
            enemy_avg_ke,
            avg_damage_eff=avg_damage_eff,
            weights=weights,
        )

        # ── Mise à jour TrueSkill (per-groupe) ──
        new_mu, new_sigma = trueskill_update(
            state.mu,
            state.sigma,
            mu_opp,
            sigma_opp,
            composite,
            weight_factor=weight_factor,
        )
        state.mu = new_mu
        state.sigma = new_sigma
        state.match_count += 1
        state.last_match_time = start_time

        # ── Historique précision (per-groupe) ──
        acc = _safe_float(row.get("accuracy"))
        if acc is not None:
            state.accuracy_history.append(acc)
            if len(state.accuracy_history) > ACCURACY_HISTORY_SIZE:
                state.accuracy_history = state.accuracy_history[-ACCURACY_HISTORY_SIZE:]

        # ── Historique efficacité dégâts (per-groupe) ──
        dmg_dealt = _safe_float(row.get("damage_dealt"))
        dmg_taken = _safe_float(row.get("damage_taken"))
        if dmg_dealt is not None and dmg_taken is not None:
            _total = dmg_dealt + dmg_taken
            if _total > 0:
                _eff = _clamp(dmg_dealt / _total, 0.0, 1.0)
                state.damage_eff_history.append(_eff)
                if len(state.damage_eff_history) > ACCURACY_HISTORY_SIZE:
                    state.damage_eff_history = state.damage_eff_history[-ACCURACY_HISTORY_SIZE:]

        match_ids.append(str(match_id))
        ratings.append(round(state.mu, 1))
        deviations.append(round(state.sigma, 1))
        groups.append(group)

    return pl.DataFrame(
        {
            "match_id": match_ids,
            "rating_value": ratings,
            "rating_deviation": deviations,
            "playlist_group": groups,
        }
    )
