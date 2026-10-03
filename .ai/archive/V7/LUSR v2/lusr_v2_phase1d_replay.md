# LUSR v2 — Phase 1d : replay sur joueurs trackés

Replay du shadow runner sur tout l'historique LUSR-éligible des joueurs ci-dessous, depuis l'état frais (priors par défaut TrueSkill : μ_0=25, σ_0=25/3, β=σ_0/2, τ=σ_0/100).

## Vue d'ensemble

| Joueur | XUID | Cible | Groupe le plus joué | μ (skill latent) | σ (incertitude) | Tier inféré | exp |
|---|---|---|---|---:|---:|---|---:|
| Madina97294 | 2533274858283686 | fin Platine / début Diamant (joueur fort) | btb | 23.98 | 2.68 | Gold | 491 |
| Chocoboflor | 2535469190789936 | milieu/bas Or (joueur moyen) | arena_slayer | 31.27 | 1.97 | Platinum | 229 |
| JGtm | 2533274823110022 | milieu/bas Or (joueur moyen) | arena_slayer | 28.25 | 1.81 | Platinum | 310 |
| XxDaemonGamerxX | 2533274833178266 | Bronze (joueur faible) | arena_slayer | 18.06 | 4.74 | Silver | 23 |

## Détail par joueur × groupe

### Madina97294 (cible : fin Platine / début Diamant (joueur fort))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 29.90 | 2.48 | Platinum | 150 |
| arena_slayer | 22.84 | 1.75 | Gold | 403 |
| btb | 23.98 | 2.68 | Gold | 491 |
| chaos | 30.13 | 5.48 | Platinum | 19 |

### Chocoboflor (cible : milieu/bas Or (joueur moyen))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 22.84 | 3.00 | Gold | 83 |
| arena_slayer | 31.27 | 1.97 | Platinum | 229 |
| btb | 25.78 | 8.27 | Gold | 1 |
| chaos | 29.83 | 5.71 | Platinum | 15 |

### JGtm (cible : milieu/bas Or (joueur moyen))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 34.65 | 2.62 | Diamond | 122 |
| arena_slayer | 28.25 | 1.81 | Platinum | 310 |
| btb | 28.12 | 7.88 | Platinum | 5 |
| chaos | 26.70 | 2.20 | Gold | 233 |

### XxDaemonGamerxX (cible : Bronze (joueur faible))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 29.34 | 7.07 | Platinum | 6 |
| arena_slayer | 18.06 | 4.74 | Silver | 23 |
| chaos | 21.99 | 7.99 | Silver | 1 |

## Lecture

- **μ ≈ 25** = niveau d'un joueur médian (prior de départ). 95% des nouveaux joueurs en 0..50.
- **σ** : incertitude. σ → 0 = skill très bien connu ; σ ≈ σ_0 (8.33) = peu joué.
- Mapping tier purement indicatif — à calibrer sur les résultats puis figer dans `lusr_hyperparams_v2` (`tier_boundary_*`).

## Décision

1. Validation qualitative : les μ doivent ordonner correctement les joueurs selon leurs niveaux connus.
2. Si OK → on peut envisager Phase 2 (squadOffset) + commencer à calibrer le mapping μ → grille [1000..2000] pour la bascule.
3. Si non OK → soit le modèle classique TrueSkill est insuffisant (besoin de Phase 3 kills/deaths), soit les priors doivent être ajustés (Sigma0 plus large pour bouger plus vite, par exemple).
