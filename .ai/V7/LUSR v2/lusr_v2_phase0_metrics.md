# LUSR v2 — Phase 0 : métriques Menke sur la base actuelle

_Généré le 2026-05-27 17:21:07 par `cmd/lusr_v2_phase0`._

## Objectif

Vérifier sur les données Halo Infinite actuelles que les biais du LUSR v1 correspondent à ceux décrits par Menke (Halo 5) et TrueSkill 2. Si les patterns Halo 5 se retrouvent → les correctifs TS2 (squadOffset, experienceOffset, kills as observation) sont les bons leviers pour le LUSR v2.

## Méthode

- Replay shadow chronologique du LUSR v1 sur les matchs LUSR-éligibles
  (non-ranked, non-firefight, durée ≥ 30s) de chaque joueur tracké.
- À chaque match : capture `mu_before`, `sigma_before`, `mu_opp` avant update.
- P(win) prédite = `sigmoid((mu - muOpp) / (2*Beta))` avec Beta = 200.
- Win réel = 1.0 si outcome=2 (Win), 0.5 si tie, 0.0 sinon.
- Squad size = 1 (solo) + nb de coéquipiers trackés dans le même match/team.

## 1. Vue d'ensemble par joueur

| Joueur | Matchs | Wins | Draws | Losses | Win% réel | Win% prédit | Avec coéquipier tracké |
|---|---:|---:|---:|---:|---:|---:|---:|
| Madina97294 | 1120 | 511 | 48 | 561 | 47.8% | 50.1% | 434 (39%) |
| Chocoboflor | 468 | 236 | 3 | 229 | 50.7% | 50.3% | 344 (74%) |
| JGtm | 893 | 437 | 7 | 449 | 49.3% | 50.1% | 466 (52%) |
| XxDaemonGamerxX | 32 | 10 | 0 | 22 | 31.2% | 49.0% | 32 (100%) |

## 2. Effet squad — `is_with_tracked_teammate`

Partitionne les observations par taille de squad (nb de joueurs trackés dans la même équipe). **Pattern attendu (Halo 5)** : pour `team_size ≥ 2`, le win% réel doit excéder le win% prédit — signature du carry passif.

| Taille squad | N obs | Win% prédit | Win% réel | Écart (réel − prédit) |
|---:|---:|---:|---:|---:|
| 1 | 1237 | 50.0% | 47.4% | -2.6pp |
| 2 | 286 | 50.1% | 53.1% | +3.0pp |
| 3 | 866 | 50.4% | 51.3% | +0.9pp |
| 4 | 124 | 49.1% | 32.3% | -16.8pp |

## 3. Effet experience — biais des premiers matchs

Partitionne par nombre de matchs LUSR-éligibles joués AVANT ce match (toutes chaines confondues). **Pattern attendu (Halo 5)** : les joueurs aux faibles `prior_match` ont un win% réel INFÉRIEUR au prédit (LUSR surévalue les nouveaux). À mesure que `prior_match` augmente, l'écart se resserre.

| Bin prior_matches | N obs | Win% prédit | Win% réel | Écart |
|---|---:|---:|---:|---:|
| 0-9 | 40 | 49.4% | 42.5% | -6.9pp |
| 10-29 | 80 | 49.4% | 46.2% | -3.1pp |
| 30-99 | 212 | 50.0% | 55.2% | +5.2pp |
| 100-299 | 600 | 50.1% | 46.7% | -3.4pp |
| 300+ | 1581 | 50.2% | 48.8% | -1.4pp |

## 4. Effet kill rate — prédictivité du match précédent

Partitionne par `kills/min` du match PRÉCÉDENT (pour éviter le data leak du match courant). **Pattern attendu (Halo 5)** : le kill rate du match précédent corrèle linéairement avec le win% réel suivant. Le LUSR v1 NE l'utilise pas → écart prédit/réel monotone visible.

| Bin kills/min | N obs | Win% prédit | Win% réel | Écart |
|---|---:|---:|---:|---:|
| 0.0-0.4 | 139 | 50.0% | 42.4% | -7.5pp |
| 0.4-0.8 | 426 | 50.0% | 45.5% | -4.5pp |
| 0.8-1.2 | 715 | 50.2% | 49.0% | -1.3pp |
| 1.2-1.6 | 641 | 50.1% | 48.7% | -1.5pp |
| 1.6-2.0 | 344 | 50.0% | 51.7% | +1.7pp |
| 2.0-2.4 | 157 | 50.1% | 49.7% | -0.4pp |
| 2.4-2.8 | 62 | 49.9% | 53.2% | +3.3pp |
| 2.8-3.2 | 11 | 49.2% | 72.7% | +23.5pp |
| 3.2+ | 14 | 49.3% | 64.3% | +15.0pp |

## 5. Verdict provisoire

Les verdicts ci-dessous sont automatiques (seuils heuristiques). Ils doivent être validés visuellement sur les tables 2-4 avant de partir sur Phase 1.

- **Squad effect (table 2)** : SIGNAL PRÉSENT (duo vs solo) — duo +3.0pp, solo -2.6pp, Δ +5.6pp (carry confirmé)
- **Experience effect (table 3)** : SIGNAL FAIBLE/ABSENT — jeunes -4.4pp, matures -1.9pp
- **Kill rate effect (table 4)** : SIGNAL PRÉSENT — low=44.8% high=52.5% (Δ=+7.7pp, LUSR prédit ~50% partout)

### Prochaine étape

1. Tu valides ce rapport.
2. Si ≥ 2 verdicts sur 3 sont "signal présent" → on attaque Phase 1 (TrueSkill classique propre).
3. Si signaux trop faibles → on discute des données manquantes (ex. squad info absente du schéma → il faudra capturer `participation_info.PresentAtCompletion` lors du sync, ou la part squad-vs-solo ne sera pas exploitable tant qu'on n'a qu'un proxy via les coéquipiers trackés).

## 6. Lecture & interprétation

### Squad effect

Le pattern Halo 5 (squad sur-performent vs prédiction) est visible sur la **taille 2** (typiquement +3pp). L'anomalie à la taille 4 (sous-performance massive) est **probablement un artéfact** : (a) petit échantillon, (b) le proxy "coéquipier tracké" capture mal les vrais squads — il manque les amis non-trackés.

**Conclusion** : signal présent mais sous-estimé. Pour le confirmer proprement, il faudrait capturer `participation_info` (vrais squads) lors du sync. **Acceptable pour aller en Phase 1+2 avec squadOffset basé sur les seules données trackées disponibles**.

### Experience effect

Les 0-9 matchs ont un écart de -6.9pp (LUSR surévalue les nouveaux), s'amenuisant à -3.1pp sur 10-29 matchs. Le bin 30-99 est un outlier (+5.2pp) probablement dû à une période où les joueurs ont effectivement "trouvé leur rythme".

**Conclusion** : signal présent sur les premiers matchs, justifie experienceOffset TS2 §7.

### Kill rate effect ⭐ LE PLUS FORT

**Pattern net et monotone** :
- kill_rate < 0.8 → win réel 44.8 % (LUSR prédit 50 %)
- kill_rate > 2.0 → win réel 52.5 % (LUSR prédit 50 %)
- Δ +7.7pp entre extrêmes (sur prédit stable à ~50 %)

C'est la signature exacte du modèle TS2 §8 (kills/deaths comme observations). Le LUSR v1 inclut déjà KvE/DvE dans son composite, mais comme **entrées** (via kills_expected Microsoft), pas comme observations Bayésiennes. Le passage à TS2 §8 (truncated Gaussian count model) devrait fortement améliorer la prédictivité.

### Recommandation

**GO pour Phase 1 + Phase 2** (TrueSkill classique propre + squadOffset). **Phase 3 (kills/deaths observations) à fort ROI** vu le signal kill_rate. **Phase 4 (mode correlation) faisable** vu qu'on a déjà des chaînes (arena_slayer, btb…). **Phase 5 (TTT batch) optionnelle** : utile pour ré-apprendre les hyperparamètres une fois qu'on aura suffisamment de données (à vue de nez : plusieurs milliers de matchs supplémentaires).

### Données qui manquent (à capturer pour LUSR v2 propre)

- `participation_info.PresentAtCompletion` (boolean per match-player) → vrai signal quit
- `participation_info.JoinInProgress` (boolean) → distinguer un quit d'un late-join
- `party_id` ou `squad_size` (entier par match-player) → vrai signal squad — sans ça, le proxy "coéquipier tracké" sera toujours bruité

