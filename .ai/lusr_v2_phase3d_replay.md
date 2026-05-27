# LUSR v2 — Phase 1d : replay sur joueurs trackés

Replay du shadow runner sur tout l'historique LUSR-éligible des joueurs ci-dessous, depuis l'état frais (priors par défaut TrueSkill : μ_0=25, σ_0=25/3, β=σ_0/2, τ=σ_0/100).

## Vue d'ensemble

| Joueur | XUID | Cible | Groupe le plus joué | μ (skill latent) | σ (incertitude) | Tier inféré | exp |
|---|---|---|---|---:|---:|---|---:|
| Madina97294 | 2533274858283686 | fin Platine / début Diamant (joueur fort) | btb | 25.96 | 0.68 | Gold | 397 |
| Chocoboflor | 2535469190789936 | milieu/bas Or (joueur moyen) | arena_slayer | 23.87 | 0.67 | Gold | 192 |
| JGtm | 2533274823110022 | milieu/bas Or (joueur moyen) | arena_slayer | 23.46 | 0.67 | Gold | 260 |
| XxDaemonGamerxX | 2533274833178266 | Bronze (joueur faible) | arena_slayer | 20.29 | 1.16 | Silver | 22 |

## Détail par joueur × groupe

### Madina97294 (cible : fin Platine / début Diamant (joueur fort))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 25.40 | 0.69 | Gold | 111 |
| arena_slayer | 26.24 | 0.67 | Gold | 341 |
| btb | 25.96 | 0.68 | Gold | 397 |
| chaos | 25.82 | 1.38 | Gold | 15 |

### Chocoboflor (cible : milieu/bas Or (joueur moyen))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 22.77 | 0.78 | Gold | 61 |
| arena_slayer | 23.87 | 0.67 | Gold | 192 |
| btb | 24.58 | 4.57 | Gold | 1 |
| chaos | 23.47 | 1.53 | Gold | 12 |

### JGtm (cible : milieu/bas Or (joueur moyen))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 23.11 | 0.71 | Gold | 90 |
| arena_slayer | 23.46 | 0.67 | Gold | 260 |
| btb | 20.11 | 2.94 | Silver | 3 |
| chaos | 24.22 | 0.67 | Gold | 196 |

### XxDaemonGamerxX (cible : Bronze (joueur faible))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 20.28 | 2.56 | Silver | 4 |
| arena_slayer | 20.29 | 1.16 | Silver | 22 |
| chaos | 23.80 | 4.48 | Gold | 1 |

## Lecture

- **μ ≈ 25** = niveau d'un joueur médian (prior de départ). 95% des nouveaux joueurs en 0..50.
- **σ** : incertitude. σ → 0 = skill très bien connu ; σ ≈ σ_0 (8.33) = peu joué.
- Mapping tier purement indicatif — à calibrer sur les résultats puis figer dans `lusr_hyperparams_v2` (`tier_boundary_*`).

## Décision

1. Validation qualitative : les μ doivent ordonner correctement les joueurs selon leurs niveaux connus.
2. Si OK → on peut envisager Phase 2 (squadOffset) + commencer à calibrer le mapping μ → grille [1000..2000] pour la bascule.
3. Si non OK → soit le modèle classique TrueSkill est insuffisant (besoin de Phase 3 kills/deaths), soit les priors doivent être ajustés (Sigma0 plus large pour bouger plus vite, par exemple).
