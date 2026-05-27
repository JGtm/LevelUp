# LUSR v2 — Phase 1d : replay sur joueurs trackés

Replay du shadow runner sur tout l'historique LUSR-éligible des joueurs ci-dessous, depuis l'état frais (priors par défaut TrueSkill : μ_0=25, σ_0=25/3, β=σ_0/2, τ=σ_0/100).

## Vue d'ensemble

| Joueur | XUID | Cible | Groupe le plus joué | μ (skill latent) | σ (incertitude) | Tier inféré | exp |
|---|---|---|---|---:|---:|---|---:|
| Madina97294 | 2533274858283686 | fin Platine / début Diamant (joueur fort) | btb | 26.17 | 0.68 | Gold | 315 |
| Chocoboflor | 2535469190789936 | milieu/bas Or (joueur moyen) | arena_slayer | 23.81 | 0.67 | Gold | 171 |
| JGtm | 2533274823110022 | milieu/bas Or (joueur moyen) | arena_slayer | 23.52 | 0.67 | Gold | 236 |
| XxDaemonGamerxX | 2533274833178266 | Bronze (joueur faible) | arena_slayer | 20.38 | 1.23 | Silver | 19 |

## Détail par joueur × groupe

### Madina97294 (cible : fin Platine / début Diamant (joueur fort))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 25.75 | 0.71 | Gold | 92 |
| arena_slayer | 26.12 | 0.67 | Gold | 298 |
| btb | 26.17 | 0.68 | Gold | 315 |
| chaos | 25.91 | 1.47 | Gold | 13 |

### Chocoboflor (cible : milieu/bas Or (joueur moyen))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 23.44 | 0.81 | Gold | 52 |
| arena_slayer | 23.81 | 0.67 | Gold | 171 |
| btb | 24.58 | 4.57 | Gold | 1 |
| chaos | 23.24 | 1.59 | Gold | 11 |

### JGtm (cible : milieu/bas Or (joueur moyen))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 23.74 | 0.73 | Gold | 80 |
| arena_slayer | 23.52 | 0.67 | Gold | 236 |
| btb | 21.33 | 3.48 | Silver | 2 |
| chaos | 24.52 | 0.67 | Gold | 167 |

### XxDaemonGamerxX (cible : Bronze (joueur faible))

| Groupe | μ | σ | Tier inféré | exp |
|---|---:|---:|---|---:|
| arena_objectif | 21.88 | 3.44 | Silver | 2 |
| arena_slayer | 20.38 | 1.23 | Silver | 19 |
| chaos | 23.76 | 4.49 | Gold | 1 |

## Lecture

- **μ ≈ 25** = niveau d'un joueur médian (prior de départ). 95% des nouveaux joueurs en 0..50.
- **σ** : incertitude. σ → 0 = skill très bien connu ; σ ≈ σ_0 (8.33) = peu joué.
- Mapping tier purement indicatif — à calibrer sur les résultats puis figer dans `lusr_hyperparams_v2` (`tier_boundary_*`).

## Décision

1. Validation qualitative : les μ doivent ordonner correctement les joueurs selon leurs niveaux connus.
2. Si OK → on peut envisager Phase 2 (squadOffset) + commencer à calibrer le mapping μ → grille [1000..2000] pour la bascule.
3. Si non OK → soit le modèle classique TrueSkill est insuffisant (besoin de Phase 3 kills/deaths), soit les priors doivent être ajustés (Sigma0 plus large pour bouger plus vite, par exemple).
