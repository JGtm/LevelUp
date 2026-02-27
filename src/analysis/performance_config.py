"""Configuration centralisée du score de performance.

Ce module définit toutes les constantes et descriptions du score de performance
pour assurer la cohérence dans toute l'application.
"""

from __future__ import annotations

from dataclasses import dataclass

from src.ui.i18n import t

# =============================================================================
# Version du score
# =============================================================================

PERFORMANCE_SCORE_VERSION = "v5-relative"
# Version actuelle de l'algorithme de score.
#
# Historique:
# - v1: Score absolu basé sur K/D, victoires, précision
# - v2: Score modulaire avec composantes et poids dynamiques
# - v3-relative: Score relatif à l'historique personnel du joueur
# - v4-relative: v3 + PSPM, DPM damage, Rank Performance (8 métriques)
# - v5-relative: v4 + Kills vs Expected, Deaths vs Expected (10 métriques)


# =============================================================================
# Paramètres du calcul
# =============================================================================

MIN_MATCHES_FOR_RELATIVE = 10  # Nombre minimum de matchs pour activer le score relatif

# Poids des métriques pour le score relatif v5
RELATIVE_WEIGHTS = {
    "kpm": 0.17,  # Kills per minute
    "dpm_deaths": 0.13,  # Deaths per minute (inversé)
    "apm": 0.08,  # Assists per minute
    "kda": 0.13,  # FDA
    "accuracy": 0.06,  # Précision
    "pspm": 0.12,  # Personal Score Per Minute (v4)
    "dpm_damage": 0.09,  # Damage Per Minute (v4)
    "rank_perf": 0.04,  # Rank vs Expected (v4, optionnel)
    "kills_vs_expected": 0.10,  # Kills réels / Kills attendus (NOUVEAU v5)
    "deaths_vs_expected": 0.08,  # Deaths attendus / Deaths réels (NOUVEAU v5, inversé)
}

# Anciens poids v4 (conservés pour référence)
RELATIVE_WEIGHTS_V4 = {
    "kpm": 0.22,
    "dpm_deaths": 0.18,
    "apm": 0.10,
    "kda": 0.15,
    "accuracy": 0.08,
    "pspm": 0.12,
    "dpm_damage": 0.10,
    "rank_perf": 0.05,
}

# Anciens poids v3 (conservés pour référence)
RELATIVE_WEIGHTS_V3 = {
    "kpm": 0.30,
    "dpm": 0.25,
    "apm": 0.15,
    "kda": 0.20,
    "accuracy": 0.10,
}

# Seuils de couleur pour l'affichage
SCORE_THRESHOLDS = {
    "excellent": 75,  # Vert
    "good": 60,  # Cyan
    "average": 45,  # Ambre
    "below_average": 30,  # Orange
    # < 30 = Rouge
}

# Labels associés aux seuils
SCORE_LABELS = {
    "excellent": "Excellent",
    "good": "Bon",
    "average": "Moyen",
    "below_average": "Faible",
    "bad": "Difficile",
}


def get_score_labels(lang: str | None = None) -> dict[str, str]:
    """Retourne les labels traduits pour les seuils de performance."""
    return {
        "excellent": t("perf_label_excellent", lang=lang),
        "good": t("perf_label_good", lang=lang),
        "average": t("perf_label_average", lang=lang),
        "below_average": t("perf_label_below", lang=lang),
        "bad": t("perf_label_bad", lang=lang),
    }


# =============================================================================
# Description centralisée (pour l'UI)
# =============================================================================

PERFORMANCE_SCORE_TITLE = "Score de performance"

PERFORMANCE_SCORE_SHORT_DESC = "Relatif à ton historique"


def get_performance_title(lang: str | None = None) -> str:
    """Retourne le titre traduit du score de performance."""
    return t("perf_title", lang=lang)


def get_performance_short_desc(lang: str | None = None) -> str:
    """Retourne la description courte traduite."""
    return t("perf_short_desc", lang=lang)


PERFORMANCE_SCORE_FULL_DESC = f"""
Le **score de performance** (0-100) est un indicateur **relatif** qui compare
ta performance sur un match à ton **historique personnel**.

### Métriques utilisées
| Métrique | Poids | Description |
|----------|-------|-------------|
| KPM (Kills/min) | 17% | Frags par minute |
| DPM Deaths (Deaths/min) | 13% | Morts par minute (inversé) |
| FDA (KDA) | 13% | Ratio (Frags + Assists) / Morts |
| PSPM (Score/min) | 12% | Score personnel par minute |
| Kills vs Expected | 10% | Frags réels vs prédiction matchmaking |
| DPM Damage (Damage/min) | 9% | Dégâts infligés par minute |
| Deaths vs Expected | 8% | Morts réelles vs prédiction (inversé) |
| APM (Assists/min) | 8% | Assistances par minute |
| Précision | 6% | Pourcentage de tirs touchés |
| Rang vs Attendu | 4% | Performance du rang ajustée par le MMR |

### Interprétation
| Score | Signification |
|-------|---------------|
| **75-100** | Match exceptionnel pour toi |
| **60-75** | Au-dessus de ta moyenne |
| **45-60** | Performance typique |
| **30-45** | En-dessous de ta moyenne |
| **0-30** | Mauvaise partie pour toi |

### Calcul
1. Pour chaque métrique, on calcule le **percentile** de ta perf dans ce match
   par rapport à tout ton historique
2. Les percentiles sont combinés avec les poids ci-dessus
3. **50 = ta performance médiane**
4. Les métriques non disponibles sont ignorées (poids renormalisés)

### Notes
- Nécessite au moins **{MIN_MATCHES_FOR_RELATIVE} matchs** dans l'historique
- Le score est **stocké en DB** au moment de l'import
- Un joueur qui s'améliore verra ses nouveaux scores monter au-dessus de 50
"""


PERFORMANCE_SCORE_COMPACT_DESC = f"""
**Score relatif (0-100)** comparant ce match à ton historique.
- >=75: Exceptionnel | >=60: Bon | >=45: Normal | >=30: Sous ta moyenne | <30: Difficile
- 10 métriques: KPM ({RELATIVE_WEIGHTS['kpm']:.0%}), DPM inversé ({RELATIVE_WEIGHTS['dpm_deaths']:.0%}), FDA ({RELATIVE_WEIGHTS['kda']:.0%}), PSPM ({RELATIVE_WEIGHTS['pspm']:.0%}), Kills vs Expected ({RELATIVE_WEIGHTS['kills_vs_expected']:.0%}), DPM Damage ({RELATIVE_WEIGHTS['dpm_damage']:.0%}), Deaths vs Expected ({RELATIVE_WEIGHTS['deaths_vs_expected']:.0%}), APM ({RELATIVE_WEIGHTS['apm']:.0%}), Précision ({RELATIVE_WEIGHTS['accuracy']:.0%}), Rang ({RELATIVE_WEIGHTS['rank_perf']:.0%})
- Minimum {MIN_MATCHES_FOR_RELATIVE} matchs requis
"""


_PERFORMANCE_FULL_DESC_EN = f"""
The **performance score** (0-100) is a **relative** indicator that compares
your performance in a match to your **personal history**.

### Metrics used
| Metric | Weight | Description |
|--------|--------|-------------|
| KPM (Kills/min) | 17% | Kills per minute |
| DPM Deaths (Deaths/min) | 13% | Deaths per minute (inverted) |
| KDA | 13% | (Kills + Assists) / Deaths ratio |
| PSPM (Score/min) | 12% | Personal score per minute |
| Kills vs Expected | 10% | Actual kills vs matchmaking prediction |
| DPM Damage (Damage/min) | 9% | Damage dealt per minute |
| Deaths vs Expected | 8% | Actual deaths vs prediction (inverted) |
| APM (Assists/min) | 8% | Assists per minute |
| Accuracy | 6% | Shot accuracy percentage |
| Rank vs Expected | 4% | Rank performance adjusted by MMR |

### Interpretation
| Score | Meaning |
|-------|-------|
| **75-100** | Exceptional match for you |
| **60-75** | Above your average |
| **45-60** | Typical performance |
| **30-45** | Below your average |
| **0-30** | Tough match for you |

### Calculation
1. For each metric, the **percentile** of your performance in this match
   is calculated relative to your entire history
2. Percentiles are combined with the weights above
3. **50 = your median performance**
4. Unavailable metrics are ignored (weights renormalized)

### Notes
- Requires at least **{MIN_MATCHES_FOR_RELATIVE} matches** in history
- The score is **stored in DB** at import time
- A player who improves will see new scores rise above 50
"""


def get_performance_full_desc(lang: str | None = None) -> str:
    """Retourne la description complète traduite du score de performance."""
    resolved = lang
    if resolved is None:
        try:
            from src.ui.i18n import get_lang

            resolved = get_lang()
        except Exception:
            resolved = "fr"
    if resolved == "en":
        return _PERFORMANCE_FULL_DESC_EN
    return PERFORMANCE_SCORE_FULL_DESC


# =============================================================================
# Dataclass pour les résultats
# =============================================================================


@dataclass(frozen=True)
class PerformanceScoreResult:
    """Résultat d'un calcul de score de performance."""

    score: float | None
    """Score final 0-100 ou None si non calculable."""

    version: str = PERFORMANCE_SCORE_VERSION
    """Version de l'algorithme utilisé."""

    percentiles: dict[str, float] | None = None
    """Percentiles par métrique (optionnel, pour debug)."""

    match_count_ref: int | None = None
    """Nombre de matchs de référence utilisés."""

    @property
    def label(self) -> str:
        """Label textuel du score (traduit)."""
        return self.get_label()

    def get_label(self, lang: str | None = None) -> str:
        """Label textuel du score, traduit selon la langue."""
        if self.score is None:
            return "N/A"
        if self.score >= SCORE_THRESHOLDS["excellent"]:
            return t("perf_score_exceptional", lang=lang)
        if self.score >= SCORE_THRESHOLDS["good"]:
            return t("perf_score_good", lang=lang)
        if self.score >= SCORE_THRESHOLDS["average"]:
            return t("perf_score_normal", lang=lang)
        if self.score >= SCORE_THRESHOLDS["below_average"]:
            return t("perf_score_below", lang=lang)
        return t("perf_score_difficult", lang=lang)

    @property
    def color_class(self) -> str:
        """Classe CSS pour la couleur."""
        if self.score is None:
            return "text-muted"
        if self.score >= SCORE_THRESHOLDS["excellent"]:
            return "perf-excellent"
        if self.score >= SCORE_THRESHOLDS["good"]:
            return "perf-good"
        if self.score >= SCORE_THRESHOLDS["average"]:
            return "perf-average"
        if self.score >= SCORE_THRESHOLDS["below_average"]:
            return "perf-below"
        return "perf-bad"


def get_score_interpretation(score: float | None, lang: str | None = None) -> str:
    """Retourne l'interprétation textuelle d'un score (traduite)."""
    if score is None:
        return t("perf_insufficient", lang=lang)
    if score >= SCORE_THRESHOLDS["excellent"]:
        return t("perf_interp_excellent", lang=lang)
    if score >= SCORE_THRESHOLDS["good"]:
        return t("perf_interp_good", lang=lang)
    if score >= SCORE_THRESHOLDS["average"]:
        return t("perf_interp_average", lang=lang)
    if score >= SCORE_THRESHOLDS["below_average"]:
        return t("perf_interp_below", lang=lang)
    return t("perf_interp_bad", lang=lang)
