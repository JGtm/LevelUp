"""Algorithme TrueSkill 2 adapté pour le LUSR (LevelUp Skill Rank).

Ce module calcule un rating de compétence absolu par match en traitant
les matchs séquentiellement dans l'ordre chronologique. Chaque match met
à jour (mu, sigma) en fonction d'un score composite continu [0,1] et de
la force estimée de l'équipe adverse (depuis kills_expected de tous les
participants dans match_participants).

Usage:
    from src.analysis.skill_rating import compute_skill_ratings_batch

    result_df = compute_skill_ratings_batch(df_player_matches, df_all_participants)
    # result_df colonnes : match_id, rating_value, rating_deviation, playlist_group
"""

from __future__ import annotations

import math
from dataclasses import dataclass, field
from typing import Any

import polars as pl

from src.analysis.skill_rating_config import (
    ACCURACY_HISTORY_SIZE,
    BETA,
    COMPOSITE_WEIGHTS,
    DEFAULT_OPPONENT_SIGMA,
    DRAW_PROBABILITY,
    INACTIVITY_SIGMA_PER_DAY,
    INACTIVITY_THRESHOLD_DAYS,
    INDIVIDUAL_MU_ALPHA,
    INITIAL_MU,
    INITIAL_SIGMA,
    K_ELO,
    MAX_INACTIVITY_DAYS,
    MAX_SIGMA,
    MIN_MATCHES_FOR_ACCURACY_DELTA,
    MIN_RATING,
    MIN_SIGMA,
    PLAYLIST_GROUPS,
    TAU,
    get_playlist_group,
)

# =============================================================================
# État du joueur entre deux matchs
# =============================================================================


@dataclass
class PlayerState:
    """État TrueSkill 2 du joueur, mis à jour après chaque match."""

    mu: float = INITIAL_MU
    """Rating courant (centre de la distribution gaussienne)."""

    sigma: float = INITIAL_SIGMA
    """Déviation courante (incertitude sur le rating)."""

    match_count: int = 0
    """Nombre de matchs traités depuis l'initialisation."""

    last_match_time: Any = None
    """Timestamp du dernier match traité (datetime ou None)."""

    accuracy_history: list[float] = field(default_factory=list)
    """Historique des précisions des derniers matchs (pour accuracy_delta)."""

    damage_eff_history: list[float] = field(default_factory=list)
    """Historique damage_efficiency des derniers matchs (pour damage_eff_delta)."""

    @classmethod
    def from_csr(cls, csr: float) -> PlayerState:
        """Initialise un PlayerState à partir d'un CSR connu (seed LUSR).

        Mapping bijection tier-à-tier entre l'échelle CSR Halo et l'échelle LUSR :
            mu = 1000 + csr × (2/3)

        Exemples :
            CSR   0  (Bronze I)   → mu 1000  (LUSR Bronze I)
            CSR 300  (Silver I)   → mu 1200  (LUSR Silver I)
            CSR 600  (Gold I)     → mu 1400  (LUSR Gold I)
            CSR 711  (Gold III)   → mu 1474  (LUSR Gold III)
            CSR 900  (Plat I)     → mu 1600  (LUSR Platine I)
            CSR 1200 (Diamond I)  → mu 1800  (LUSR Diamond I)
            CSR 1400 (Diamond V)  → mu 1933  (LUSR Diamond V)
            CSR 1500 (Onyx)       → mu 2000  (LUSR Onyx)

        Le sigma est réduit à 60 % de l'initial pour refléter la confiance
        apportée par le CSR comme ancre (incertitude réduite mais non minimale).
        """
        mu = 1000.0 + max(0.0, float(csr)) * (2.0 / 3.0)
        # Sigma = MIN_SIGMA (état stable) : le CSR est un ancrage très fort.
        # On démarre directement au steady-state (~65 après TAU) pour éviter
        # la phase de haute volatilité qui cause des swings aléatoires de ±100 pts
        # dans les premiers matchs du groupe.
        sigma = MIN_SIGMA  # 60 → devient ~65 après TAU
        return cls(mu=mu, sigma=sigma)


# =============================================================================
# Fonctions utilitaires
# =============================================================================


from src.utils.safe_types import clamp as _clamp  # noqa: E402
from src.utils.safe_types import safe_float as _safe_float  # noqa: E402


def _sigmoid_ratio(numerator: float, denominator: float) -> float:
    """Mappe ratio positif → [0,1] via sigmoid symétrique.

    ratio = 1.0 → 0.5 (performance attendue)
    ratio = 2.0 → ~0.67 (sur-performance)
    ratio = 0.5 → ~0.33 (sous-performance)
    """
    if denominator <= 0:
        return 0.5
    ratio = numerator / denominator
    # Utilise x/(1+x) plutôt qu'un sigmoid standard pour un mapping naturel
    return _clamp(ratio / (1.0 + ratio), 0.0, 1.0)


# =============================================================================
# Fonctions TrueSkill 2 — truncated Gaussian
# =============================================================================


def _draw_margin(beta: float) -> float:
    """Calcule la marge d'égalité ε à partir de la probabilité de draw."""
    # ε = Φ^{-1}((p_draw + 1) / 2) × c, approché par la formule standard
    # Pour DRAW_PROBABILITY ≈ 0.06 : ε ≈ 0.074 × c
    if DRAW_PROBABILITY <= 0:
        return 0.0
    p = (DRAW_PROBABILITY + 1.0) / 2.0
    # Approximation de Φ^{-1}(p) pour p proche de 0.5
    # Utilise la méthode de Beasley-Springer-Moro (approximation rapide)
    t = math.sqrt(-2.0 * math.log(1.0 - p)) if p < 1.0 else 8.0
    return t * beta


def _v_win(t: float, eps: float) -> float:
    """Facteur v pour le cas victoire (score > 0.5)."""
    x = t - eps
    denom = _standard_normal_cdf(x)
    if denom < 1e-10:
        return -x  # asymptote
    return _standard_normal_pdf(x) / denom


def _w_win(t: float, eps: float) -> float:
    """Facteur w pour le cas victoire."""
    x = t - eps
    v = _v_win(t, eps)
    return v * (v + x)


def _v_draw(t: float, eps: float) -> float:  # pragma: no cover — non utilisé (v5.3)
    """Facteur v pour le cas égalité — conservé comme référence TrueSkill 2.

    .. deprecated::
        Non appelé dans le calcul courant (Halo Infinite n'a pas de matchs
        nuls vrais en compétitif). Archivé pour référence algorithmique.
    """
    abs_t = abs(t)
    num = _standard_normal_pdf(eps - abs_t) - _standard_normal_pdf(-eps - abs_t)
    denom = _standard_normal_cdf(eps - abs_t) - _standard_normal_cdf(-eps - abs_t)
    if denom < 1e-10:
        return 0.0
    if t < 0:
        return -num / denom
    return num / denom


def _w_draw(t: float, eps: float) -> float:  # pragma: no cover — non utilisé (v5.3)
    """Facteur w pour le cas égalité — conservé comme référence TrueSkill 2.

    .. deprecated::
        Non appelé dans le calcul courant. Archivé pour référence algorithmique.
    """
    abs_t = abs(t)
    eps_m = eps - abs_t
    eps_p = -eps - abs_t
    num = eps_m * _standard_normal_pdf(eps_m) - eps_p * _standard_normal_pdf(eps_p)
    denom = _standard_normal_cdf(eps_m) - _standard_normal_cdf(eps_p)
    if denom < 1e-10:
        return 1.0
    return (
        num / denom
        + ((eps_m * _standard_normal_pdf(eps_m) - eps_p * _standard_normal_pdf(eps_p)) / denom) ** 2
    )


def _standard_normal_pdf(x: float) -> float:
    return math.exp(-0.5 * x * x) / math.sqrt(2.0 * math.pi)


def _standard_normal_cdf(x: float) -> float:
    return 0.5 * (1.0 + math.erf(x / math.sqrt(2.0)))


# =============================================================================
# Score composite (0 → 1)
# =============================================================================


def compute_composite_score(  # noqa: C901, PLR0912, PLR0913
    row: dict[str, Any],
    avg_accuracy: float | None,
    teammate_avg_ke: float | None,
    enemy_avg_ke: float | None,
    avg_damage_eff: float | None = None,
    *,
    weights: dict[str, float] | None = None,
) -> float:
    """Calcule le score composite continu [0, 1] d'un match.

    Ce score est l'équivalent du "résultat" dans TrueSkill 2 — il combine
    plusieurs métriques en un indicateur unique de performance.

    La valeur 0.5 correspond à une performance parfaitement attendue.
    > 0.5 : sur-performance, < 0.5 : sous-performance.

    Graceful degradation : si des métriques sont manquantes, les poids sont
    renormalisés automatiquement sur les métriques disponibles.

    Args:
        row: Ligne du match avec kills, deaths, kills_expected, deaths_expected,
             outcome, damage_dealt, damage_taken, accuracy.
        avg_accuracy: Moyenne historique de la précision du joueur.
        teammate_avg_ke: kills_expected moyen des coéquipiers (ajustement carry).
        enemy_avg_ke: kills_expected moyen des adversaires (contexte opposition).
        avg_damage_eff: Moyenne historique damage_efficiency du joueur.
            Si fourni, le composant est calculé en DELTA (ce match vs sa moyenne)
            pour éviter le biais systématique des joueurs forts qui ont toujours
            une efficacité > 0.5. Si None, utilise la valeur brute.

    Returns:
        Score composite entre 0.0 et 1.0.
    """
    components: dict[str, float] = {}
    weights_used: dict[str, float] = {}
    # Poids effectifs : permet à la calibration de passer ses propres poids
    # sans muter le global COMPOSITE_WEIGHTS (thread-safe).
    _w = weights if weights is not None else COMPOSITE_WEIGHTS

    # ── 1. Kills vs Expected ──
    kills = _safe_float(row.get("kills"))
    kills_expected = _safe_float(row.get("kills_expected"))
    if kills is not None and kills_expected is not None and kills_expected > 0:
        score = _sigmoid_ratio(kills, kills_expected)
        # Ajustement carry : si mes coéquipiers ont beaucoup plus de kills_expected,
        # une bonne perf peut être une chance — réduire légèrement l'impact
        if teammate_avg_ke is not None and kills_expected > 0 and teammate_avg_ke > 0:
            carry_ratio = kills_expected / teammate_avg_ke
            # carry_ratio < 1 → j'étais le plus faible, mes kills comptent plus
            # carry_ratio > 1 → j'étais le plus fort, normal de faire plus de kills
            carry_adj = _clamp(carry_ratio, 0.5, 2.0)
            score = _clamp(score * (1.0 / carry_adj) + 0.5 * (1.0 - 1.0 / carry_adj), 0.0, 1.0)
        components["kills_vs_expected"] = score
        weights_used["kills_vs_expected"] = _w["kills_vs_expected"]

    # ── 2. Deaths vs Expected (inversé) ──
    deaths = _safe_float(row.get("deaths"))
    deaths_expected = _safe_float(row.get("deaths_expected"))
    if deaths is not None and deaths_expected is not None and deaths_expected > 0:
        deaths_safe = max(1.0, deaths)
        score = _sigmoid_ratio(deaths_expected, deaths_safe)
        components["deaths_vs_expected"] = score
        weights_used["deaths_vs_expected"] = _w["deaths_vs_expected"]

    # ── 3. Win factor ──
    outcome = row.get("outcome")
    if outcome is not None:
        try:
            outcome_int = int(outcome)
        except (ValueError, TypeError):
            outcome_int = -1
        win_map = {2: 1.0, 1: 0.5, 3: 0.0, 4: 0.15}
        win_score = win_map.get(outcome_int)
        if win_score is not None:
            components["win_factor"] = win_score
            weights_used["win_factor"] = _w["win_factor"]

    # ── 4. Damage efficiency ──
    # Si avg_damage_eff fourni : delta vs historique personnel (élimine le biais
    # systématique des bons joueurs qui ont toujours efficacité > 0.5).
    # Sinon : valeur brute (pour la phase de bootstrap avant que l'historique existe).
    damage_dealt = _safe_float(row.get("damage_dealt"))
    damage_taken = _safe_float(row.get("damage_taken"))
    if damage_dealt is not None and damage_taken is not None:
        total = damage_dealt + damage_taken
        if total > 0:
            raw_eff = _clamp(damage_dealt / total, 0.0, 1.0)
            if avg_damage_eff is not None and avg_damage_eff > 0:
                # Delta : ce match vs ma moyenne → neutre à 0.5 en moyenne
                score_eff = _sigmoid_ratio(raw_eff, avg_damage_eff)
            else:
                score_eff = raw_eff
            components["damage_efficiency"] = score_eff
            weights_used["damage_efficiency"] = _w["damage_efficiency"]

    # ── 5. Accuracy delta ──
    accuracy = _safe_float(row.get("accuracy"))
    if accuracy is not None and avg_accuracy is not None and avg_accuracy > 0:
        score = _sigmoid_ratio(accuracy, avg_accuracy)
        components["accuracy_delta"] = score
        weights_used["accuracy_delta"] = _w["accuracy_delta"]

    if not components:
        return 0.5  # Aucune donnée → résultat neutre

    total_weight = sum(weights_used.values())
    if total_weight < 1e-12:
        return 0.5

    composite = sum(components[k] * weights_used[k] for k in components) / total_weight
    return _clamp(composite, 0.0, 1.0)


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
# Mise à jour TrueSkill 2
# =============================================================================


def trueskill_update(  # noqa: PLR0913
    mu: float,
    sigma: float,
    mu_opp: float,
    sigma_opp: float,
    actual_score: float,
    weight_factor: float = 1.0,
) -> tuple[float, float]:
    """Met à jour (mu, sigma) après un match.

    **Mu** : formule Elo-style continue (K_ELO × (score - 0.5) × wf).
        → ZÉRO exact quand composite = 0.5, indépendant de mu_opp.
        → Élimine le biais asymétrique de la zone draw TrueSkill classique
          (v_draw avec t > 0 donnait des mises à jour positives même sur des
          scores neutres quand le joueur dépassait son niveau de référence).

    **Sigma** : réduction TrueSkill à t = 0 (symétrique, matchmaking ~équivalent).
        → mu_opp et sigma_opp influencent c² → ampleur de la réduction.
        → TAU ajouté après (dérive dynamique par match).

    Args:
        mu: Rating courant du joueur.
        sigma: Déviation courante du joueur.
        mu_opp: Rating estimé de l'équipe adverse (utilisé pour c², i.e. sigma).
        sigma_opp: Déviation estimée de l'équipe adverse.
        actual_score: Score composite du match [0, 1].
        weight_factor: Facteur playlist (1.0=ranked, 0.15=fun).

    Returns:
        Tuple (new_mu, new_sigma).
    """
    # ── Mu : Elo-style continu ──
    # delta = K_ELO × (composite - 0.5) × wf
    # Propriétés :
    #   composite = 0.5  → delta = 0 (performance attendue, rating stable)
    #   composite = 1.0  → delta = +K_ELO × 0.5 × wf (sur-performance maximale)
    #   composite = 0.0  → delta = -K_ELO × 0.5 × wf (sous-performance maximale)
    delta_mu = K_ELO * (actual_score - 0.5) * weight_factor
    new_mu = max(MIN_RATING, mu + delta_mu)

    # ── Sigma : TrueSkill à t = 0 ──
    # c² = 2β² + σ² + σ_opp²  (c reflète l'incertitude totale du match)
    # w = facteur de réduction d'incertitude au point t=0 (symétrique)
    c2 = 2.0 * BETA**2 + sigma**2 + sigma_opp**2
    c = math.sqrt(c2)
    eps = _draw_margin(BETA)

    sigma2 = sigma**2
    w = _w_win(0.0, eps / c)  # t=0 toujours → pas de biais directionnel
    delta_sigma2 = sigma2 * (sigma2 / c2) * w * weight_factor

    new_sigma2 = max(MIN_SIGMA**2, sigma2 - delta_sigma2)
    new_sigma = math.sqrt(new_sigma2)

    # Dérive TAU (sigma croît légèrement à chaque match — évolution naturelle)
    new_sigma = min(math.sqrt(new_sigma**2 + TAU**2), MAX_SIGMA)

    return new_mu, new_sigma


def apply_inactivity_decay(sigma: float, days_inactive: float) -> float:
    """Augmente sigma proportionnellement à la durée d'inactivité.

    Args:
        sigma: Déviation courante.
        days_inactive: Jours depuis le dernier match.

    Returns:
        Nouvelle sigma (augmentée).
    """
    capped = min(days_inactive, MAX_INACTIVITY_DAYS)
    if capped <= INACTIVITY_THRESHOLD_DAYS:
        return sigma
    added = INACTIVITY_SIGMA_PER_DAY * (capped - INACTIVITY_THRESHOLD_DAYS)
    return _clamp(sigma + added, MIN_SIGMA, MAX_SIGMA)


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
