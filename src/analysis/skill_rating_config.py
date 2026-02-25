"""Configuration centralisée du LUSR (LevelUp Skill Rank).

Système de rating inspiré de TrueSkill 2, adapté pour Halo Infinite.
Les tiers réutilisent les images CSR existantes dans static/ranks/.

Les matchs classés utilisent le CSR de l'API ; les matchs non classés
utilisent le LUSR calculé localement. Un match ne peut avoir qu'un seul
type de rating dans la table match_skill_rank.
"""

from __future__ import annotations

from dataclasses import dataclass

# Groupes de playlists isolés dans un fichier dédié pour faciliter les
# modifications sans toucher à l'algorithme de rating.
from src.analysis.playlist_groups import (  # noqa: F401
    DEFAULT_PLAYLIST_GROUP,
    PLAYLIST_GROUPS,
    PlaylistGroupConfig,
    get_playlist_group,
)

# =============================================================================
# Version de l'algorithme
# =============================================================================

SKILL_RATING_VERSION = "v1-trueskill2-adapted"

# =============================================================================
# Paramètres TrueSkill 2
# =============================================================================

# Rating initial (mu) — centre de l'échelle
INITIAL_MU: float = 1500.0

# Déviation initiale (sigma) — incertitude de départ maximale
INITIAL_SIGMA: float = 350.0

# Déviation minimale (plancher — jamais en dessous même avec beaucoup de matchs)
MIN_SIGMA: float = 60.0

# Déviation maximale (plafond)
MAX_SIGMA: float = 350.0

# Bruit de performance beta (β) — sépare le vrai skill de la variance par match.
# Plus β est grand, plus un match individuel a d'impact sur le résultat.
BETA: float = 200.0

# Dérive dynamique tau (τ) — sigma augmente de τ entre chaque match.
# Représente l'évolution naturelle du skill dans le temps.
TAU: float = 25.0

# Probabilité implicite d'égalité (ε) — utilisée pour la zone de "draw margin".
DRAW_PROBABILITY: float = 0.06

# Minimum de matchs avant que le rating soit considéré fiable
MIN_MATCHES_FOR_RATING: int = 10

# Rating minimum global (jamais en dessous)
MIN_RATING: float = 200.0

# =============================================================================
# Elo-style continuous update
# =============================================================================

# Facteur K Elo — amplitude du changement mu par match.
# delta_mu = K_ELO × (composite - 0.5) × weight_factor
# ZÉRO exact quand composite = 0.5, élimine le biais de la zone draw TrueSkill.
# K=32 : gain max par match = 32 × 0.5 × wf → ≈ ±12.8 pts (arena, wf=0.8)
K_ELO: float = 32.0

# =============================================================================
# Inactivité
# =============================================================================

# Augmentation de sigma par jour d'inactivité.
# Réduit à 1.0 (vs 3.5 initial) pour éviter les pics de sigma après de longues
# pauses : dans un système one-sided (un seul joueur mis à jour), un sigma trop
# grand après un break cause des swings de ±200 pts en un seul match.
INACTIVITY_SIGMA_PER_DAY: float = 1.0

# Nombre max de jours d'inactivité pris en compte (au-delà = plafonné).
# Réduit à 14 (2 semaines) : max sigma additionnel = 1.0 × 13 = 13 pts.
MAX_INACTIVITY_DAYS: int = 14

# Seuil en jours pour considérer une inactivité (1 jour minimum)
INACTIVITY_THRESHOLD_DAYS: float = 1.0

# =============================================================================
# Score composite (outcome continu 0 → 1)
# =============================================================================

COMPOSITE_WEIGHTS: dict[str, float] = {
    # Calibration individuelle sur 1765 matchs (3 joueurs, niveaux Argent→Diamant).
    # Signal cible : individual_mmr = team_mmr × (kills_expected / ke_avg_match).
    # Pondération : nb_matchs × amélioration_MAE (Madina 36.7%, JGtm 40.0%, Chocobo 13.3%).
    "kills_vs_expected": 0.31,  # actual_kills / kills_expected (sigmoid)
    "deaths_vs_expected": 0.28,  # deaths_expected / actual_deaths (sigmoid, inversé)
    "win_factor": 0.05,  # victoire=1.0, égalité=0.5, défaite=0.0 — faible impact en social
    "damage_efficiency": 0.23,  # damage_dealt / (damage_dealt + damage_taken)
    "accuracy_delta": 0.13,  # précision vs moyenne historique du joueur
}

# Nombre de matchs récents conservés pour calculer la moyenne de précision
ACCURACY_HISTORY_SIZE: int = 50

# Minimum de matchs précédents pour activer l'accuracy_delta
MIN_MATCHES_FOR_ACCURACY_DELTA: int = 5

# =============================================================================
# Estimation μ individuel depuis kills_expected
# =============================================================================

# Coefficient α pour l'estimation μ d'un joueur :
# μ_estimé = base_mu + INDIVIDUAL_MU_ALPHA × (ke - avg_ke) / std_ke
INDIVIDUAL_MU_ALPHA: float = 150.0

# Sigma par défaut assigné aux adversaires estimés (incertitude standard)
DEFAULT_OPPONENT_SIGMA: float = 150.0

# =============================================================================
# Tiers (réutilisent static/ranks/120px-HINF-CSR_*.png)
# =============================================================================


@dataclass(frozen=True)
class SkillTier:
    """Définition d'un tier du LUSR."""

    name: str
    """Nom anglais (correspond au préfixe des images PNG)."""

    name_fr: str
    """Nom français pour l'UI."""

    min_rating: float
    """Rating minimum (inclus)."""

    max_rating: float
    """Rating maximum (exclus, sauf Onyx qui n'a pas de max réel)."""

    color: str
    """Couleur hex pour les graphiques."""

    sub_tiers: int
    """Nombre de sous-tiers (1-6 pour la plupart, 0/1 pour Onyx)."""


SKILL_TIERS: list[SkillTier] = [
    SkillTier("Bronze", "Bronze", 1000.0, 1200.0, "#CD7F32", 6),
    SkillTier("Silver", "Argent", 1200.0, 1400.0, "#C0C0C0", 6),
    SkillTier("Gold", "Or", 1400.0, 1600.0, "#FFD700", 6),
    SkillTier("Platinum", "Platine", 1600.0, 1800.0, "#00CED1", 6),
    SkillTier("Diamond", "Diamant", 1800.0, 2000.0, "#B9F2FF", 6),
    SkillTier("Onyx", "Onyx", 2000.0, 9999.0, "#1C1C1C", 1),
]

# Tier en dessous du Bronze (départ)
UNRANKED_COLOR: str = "#888888"
UNRANKED_LABEL: str = "Non classé"


def get_tier_for_rating(rating: float) -> tuple[SkillTier | None, int]:
    """Retourne le tier et le sous-tier pour un rating donné.

    Args:
        rating: Valeur LUSR du joueur.

    Returns:
        Tuple (SkillTier, sub_tier_1indexed).
        sub_tier = 0 pour Onyx (pas de sous-tiers).
        Retourne (None, 0) si en dessous de Bronze.
    """
    for tier in SKILL_TIERS:
        if tier.min_rating <= rating < tier.max_rating:
            if tier.sub_tiers <= 1:
                return tier, 0
            range_per_sub = (tier.max_rating - tier.min_rating) / tier.sub_tiers
            sub = int((rating - tier.min_rating) / range_per_sub) + 1
            sub = min(sub, tier.sub_tiers)
            return tier, sub
    return None, 0


def format_tier_label(rating: float) -> str:
    """Formate le label complet du tier en français.

    Examples:
        1450 → "Or II"
        2100 → "Onyx"
        900  → "Non classé"

    Args:
        rating: Valeur LUSR du joueur.

    Returns:
        Label formaté.
    """
    tier, sub = get_tier_for_rating(rating)
    if tier is None:
        return UNRANKED_LABEL
    if sub == 0:
        return tier.name_fr
    roman = {1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}
    return f"{tier.name_fr} {roman.get(sub, str(sub))}"


def get_tier_color(rating: float) -> str:
    """Retourne la couleur hex du tier pour un rating donné."""
    tier, _ = get_tier_for_rating(rating)
    return tier.color if tier else UNRANKED_COLOR


def get_rank_image_filename(tier_name: str, sub_tier: int) -> str:
    """Retourne le nom du fichier PNG pour un tier donné.

    Convention : 120px-HINF-CSR_{TierName}{SubTier}.png
    Exemples :
        "Gold", 3  → "120px-HINF-CSR_Gold3.png"
        "Onyx", 0  → "120px-HINF-CSR_Onyx.png"

    Args:
        tier_name: Nom anglais du tier (ex: "Gold", "Onyx").
        sub_tier: Sous-tier (0 pour Onyx).

    Returns:
        Nom de fichier PNG.
    """
    if sub_tier == 0:
        return f"120px-HINF-CSR_{tier_name}.png"
    return f"120px-HINF-CSR_{tier_name}{sub_tier}.png"


def get_rank_image_path(rating: float) -> str | None:
    """Retourne le chemin relatif de l'image PNG pour un rating.

    Args:
        rating: Valeur LUSR.

    Returns:
        Chemin relatif depuis la racine du projet (ex: "static/ranks/120px-HINF-CSR_Gold3.png"),
        ou None si non classé.
    """
    tier, sub = get_tier_for_rating(rating)
    if tier is None:
        return None
    filename = get_rank_image_filename(tier.name, sub)
    return f"static/ranks/{filename}"


def get_tier_size(rating: float) -> float:
    """Retourne la taille (en points) du sous-tier contenant ce rating.

    Utilisé pour calculer la progression dans la barre de rang.

    Args:
        rating: Valeur LUSR.

    Returns:
        Taille du sous-tier en points LUSR.
    """
    tier, _ = get_tier_for_rating(rating)
    if tier is None or tier.sub_tiers <= 1:
        return tier.max_rating - tier.min_rating if tier else 200.0
    return (tier.max_rating - tier.min_rating) / tier.sub_tiers


def get_sub_tier_start(rating: float) -> float:
    """Retourne la valeur de rating au début du sous-tier actuel."""
    tier, sub = get_tier_for_rating(rating)
    if tier is None:
        return 0.0
    if tier.sub_tiers <= 1:
        return tier.min_rating
    size = (tier.max_rating - tier.min_rating) / tier.sub_tiers
    return tier.min_rating + (sub - 1) * size


# =============================================================================
# Descriptions UI
# =============================================================================

LUSR_TITLE = "LUSR"
LUSR_SUBTITLE = "LevelUp Skill Rank"
LUSR_SHORT_DESC = "Rating de compétence absolu (TrueSkill 2 adapté)"

LUSR_FULL_DESC = f"""
Le **LUSR (LevelUp Skill Rank)** est un rating de compétence **absolu** qui suit
l'évolution de ton niveau au fil du temps, inspiré de **TrueSkill 2**
(le système utilisé par Halo Infinite en interne).

### Différence avec le Score de Performance
| | Score de Performance | LUSR |
|---|---|---|
| **Type** | Relatif (vs ton historique) | Absolu (échelle globale) |
| **Échelle** | 0-100 | ~{MIN_RATING:.0f}-2200+ |
| **Signification** | "Ce match vs ta moyenne" | "Ton niveau absolu" |
| **Adversaires** | Non pris en compte | Force de l'opposition intégrée |
| **Tendance** | Gravite vers 50 | Monte si tu progresses réellement |

### Pour les matchs classés
Les matchs **classés** affichent directement la valeur **CSR** (Competitive Skill Rating)
fournie par l'API Halo, au lieu du LUSR calculé. Un match ne peut avoir que l'un ou l'autre.

### Métriques du score composite
| Métrique | Poids | Description |
|----------|-------|-------------|
| Kills vs Expected | {COMPOSITE_WEIGHTS['kills_vs_expected']:.0%} | Frags réels / frags prédits |
| Deaths vs Expected | {COMPOSITE_WEIGHTS['deaths_vs_expected']:.0%} | Morts prédites / morts réelles |
| Facteur victoire | {COMPOSITE_WEIGHTS['win_factor']:.0%} | V=1, Égalité=0.5, D=0 |
| Efficacité dégâts | {COMPOSITE_WEIGHTS['damage_efficiency']:.0%} | Dégâts infligés / total |
| Delta précision | {COMPOSITE_WEIGHTS['accuracy_delta']:.0%} | Précision vs ta moyenne |

### Tiers
| Tier | Rating LUSR |
|------|-------------|
| Bronze I-VI | 1000-1199 |
| Argent I-VI | 1200-1399 |
| Or I-VI | 1400-1599 |
| Platine I-VI | 1600-1799 |
| Diamant I-VI | 1800-1999 |
| Onyx | 2000+ |

### Notes
- Minimum **{MIN_MATCHES_FOR_RATING} matchs** pour un rating significatif
- Les modes **Fun** ont 4× moins d'impact sur le rating que le **Classé**
- La déviation (incertitude) diminue avec les matchs, augmente avec l'inactivité
"""

LUSR_COMPACT_DESC = (
    f"**LUSR (0-2200+)** — Rating absolu inspiré TrueSkill 2. "
    f"Monte si tu t'améliores réellement. "
    f"Matchs classés = CSR API. Minimum {MIN_MATCHES_FOR_RATING} matchs requis."
)
