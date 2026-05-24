# LUSR — Chute en rang d'un joueur carry agressif

**Date création** : 2026-05-24
**Statut** : Problème ouvert — aucune solution satisfaisante trouvée à ce jour
**Branche d'investigation** : `fix/duckdb-art-corruption-rebuild` (2026-05-22)
**Fichiers clés** :
- `apps/go-api/internal/sync/skill_rating.go` — logique TrueSkill 2 + composite
- `apps/go-api/internal/sync/skill_config.go` — poids `CompositeWeights`, constantes
- `apps/go-api/internal/sync/skill_rating_loaders.go` — loaders SQL + upsert
- `apps/go-api/cmd/diag_lusr_player/` — outil replay TrueSkill (avec/sans variantes)

---

## 1. Contexte

Madina97294 est perçu par le squad comme le meilleur joueur (niveau estimé Diamant).
Pourtant, son LUSR stagnait en **Argent III-IV** (1286 MU arena_slayer) tandis que
JGtm et Chocoboflor atteignaient **Or II** (~1444-1451).

Cibles de référence (voir `memory/reference_lusr_target_levels.md`) :
- Madina97294 → fin Platine / début Diamant (~1750-1900)
- Chocoboflor + JGtm → milieu/bas Or (~1400-1500)

---

## 2. Architecture actuelle de la formule LUSR

### Score composite [0, 1]

8 composantes, renormalisées par `totalWeight` (somme ≈ 1.02) :

| Composante | Clé | Poids | Source |
|---|---|---|---|
| kills_vs_expected | `MetricKeyKillsVsExpected` | **0.27** | API Halo MMR |
| deaths_vs_expected | `MetricKeyDeathsVsExpected` | **0.24** | API Halo MMR |
| offensive_conversion | `MetricKeyOffensiveConv` | 0.16 | `225*(kills+assists/3)/damage_dealt` |
| damage_efficiency | `MetricKeyDamageEfficiency` | 0.10 | `damage_dealt/(damage_dealt+damage_taken)` |
| accuracy_delta | `MetricKeyAccuracyDelta` | 0.10 | vs moyenne historique personnelle |
| defensive_resistance | `MetricKeyDefensiveResist` | 0.06 | `damage_taken/(225*deaths)` |
| win_factor | `MetricKeyWinFactor` | **0.05** | WIN=1.0, TIE=0.5, LOSS=0.0, DNF=0.15 |
| medal_exploit | `MetricKeyMedalExploit` | 0.04 | score brut médailles exploit |

Le composite alimente `trueskillUpdate` comme `actualScore` (vs `expectedScore` basé sur `muOpp`).

### Carry adjustment (actuel, asymétrique)

Dans `computeCompositeScoreWithBreakdown`, pour la composante `kills_vs_expected` :

```go
if score > 0.5 {
    carryRatio := row.KillsExpected / *enemyAvgKE
    carryAdj  := clampF(carryRatio, 1.0, 2.0)
    score = clampF(0.5 + (score-0.5)/carryAdj, 0.0, 1.0)
}
// score <= 0.5 : pénalité pleine, non modifiée
```

Logique : si le joueur surperforme son KE face à des adversaires **faibles** (son KE > enemyAvgKE),
le bonus est compressé. Si les adversaires sont plus forts, carryAdj = 1.0 (floor) — pas d'amplification.

### TrueSkill update

```go
expectedScore = 1 / (1 + exp(-(mu - muOpp) / (2 * Beta)))
deltaMU       = KElo * (actualScore - expectedScore) * weightFactor  // weightFactor=1.0 fixe
newMU = max(MinRating, mu + deltaMU)
```

`muOpp` est estimé depuis les `kills_expected` des adversaires via `estimateIndividualMU`.
`KElo = 32`, `Beta = 200`, `InitialMU = 1500`.

---

## 3. Diagnostic — causes racines identifiées

### Cause 1 : kills_vs_expected indexé sur le MMR personnel (principal)

L'API Halo fournit `StatPerformances.Kills.Expected` basé sur le **MMR du joueur**,
pas sur le lobby. Un joueur avec un MMR élevé reçoit des KE de 13-21.

Conséquence : même quand Madina **domine un lobby** à 17-19 kills (top de la partie),
`kills/KE ≈ 1.0` → `sigmoidRatio(kills, KE) → 0.5` → composite ≈ 0.5 → mu stagne.

JGtm avec KE 7-12 sur les mêmes matchs sur-performe plus facilement → mu monte.

> Le LUSR mesure la **consistance vs son propre MMR**, pas la dominance en lobby.
> Un joueur sur-coté par Halo dans un squad de niveau inférieur est mécaniquement plafonné.

Les deux composantes concernées (`kills_vs_expected` 0.27 + `deaths_vs_expected` 0.24)
représentent **51% du composite total**.

### Cause 2 : win_factor à 0.05 (secondaire)

Un carry qui joue dans une équipe moyenne peut gagner individuellement mais perdre collectivement.
`win_factor = 0.05` signifie que perdre 1.0 → 0.0 sur cette composante coûte au maximum
`0.05 / 1.02 * (1.0 - 0.5) ≈ 0.025` de composite — impact très limité sur le mu final.
Ce n'est donc **pas** la cause principale, mais ça s'accumule sur des centaines de défaites.

### Cause 3 : fenêtre temporelle (confirmée empiriquement)

Test fenêtré arena_slayer (toujours depuis InitialMU=1500) :
- 20 derniers matchs : Madina **1483** > JGtm 1474 — Madina est devant
- 100 derniers matchs : Madina 1432 ≈ JGtm 1449 — quasi-égaux
- Full history : Madina 1286, JGtm 1444 — **gap 158 pts**

L'écart full-history reflète des performances plus faibles de Madina en **début de carrière**
(plusieurs centaines de matchs anciens). La formule en elle-même n'est pas fautive sur la fenêtre récente.

---

## 4. Variantes testées et résultats

Simulation via `cmd/diag_lusr_player/` + flag `--compare-formulas`, full history, arena_slayer :

| Variante | Madina | JGtm | Chocoboflor | Verdict |
|---|---|---|---|---|
| **Baseline actuel** | 1286 (Arg III) | 1444 (Or II) | 1451 (Or II) | — |
| **Piste-C** : DvE 0.24 → 0.12 | +58 = **1344** | +23 | **−58** | Réduit l'écart mais dévisse Choco |
| **Piste-A** : DE × kills/KE | +23 = 1309 | +48 | −39 | Empire le gap Madina/JGtm |
| **Piste-A+C** combiné | +69 = **1355** | +48 | **−78 !** | Meilleure option formule, Choco dévisse |
| **Piste-B** : KDA fusionné | — | **1656 !** | **1691 !** | Inflation massive, éliminé |

Aucune variante de formule ne corrige le problème sans créer d'effet de bord sur les autres joueurs.

### Pourquoi carry adjustment (piste originale) ne fonctionnait pas

Version initiale utilisait `teammateAvgKE` → pénalisait le joueur pour la faiblesse de ses alliés.
Empiriquement : la condition `teammateAvgKE > 0` n'était presque jamais remplie pour Madina
(équipe adverse de ses amis → `teammates_kills_expected = 0` dans les matchs communs).

Corrigé vers `enemyAvgKE` (2026-05-22) — amélioration marginale, ne résout pas le fond.

---

## 5. Fixes livrés (partiels, dans le code actuel)

1. **Carry adjustment asymétrique** (`skill_rating.go:178-190`) — basé sur `enemyAvgKE`, floor 1.0, asymétrique (pénalité pleine conservée, bonus compressé uniquement). Implémenté, testé (4 tests `TestCarryAdj_*`).

2. **`muOpp` branché** dans `trueskillUpdate` — était présent en signature mais ignoré avant 2026-05-22. Maintenant : `expectedScore = 1/(1+exp(-(mu-muOpp)/(2*Beta)))`. Battre des adversaires plus forts donne plus de gain.

Ces deux fixes améliorent la dynamique générale mais ne résolvent pas la stagnation Madina.

---

## 6. Pistes non explorées

### Piste 1 : LUSR_recent (recommandée)
Variante fenêtrée sur les **50 derniers matchs par chaîne**, calculée en dry-run / lecture seule,
affichée en priorité dans l'UI à côté du LUSR full-history.

- Sur 100 matchs récents, Madina est quasi-égal à JGtm → la piste est valide empiriquement.
- Implémentation : réutiliser `batchComputeLUSRPreview` avec un slice `matches[max(0, len-50):]`.
- Pas de modification du schéma — valeur non persistée ou stockée dans une colonne dédiée.

### Piste 2 : `kills_vs_lobby_avg`
Remplacer `kills_vs_expected` (basé MMR Halo) par `kills / lobby_avg_kills`
où `lobby_avg_kills` = moyenne des kills de tous les participants du match.

- Élimine la dépendance au MMR Halo.
- Mesure la dominance en lobby plutôt que la consistance MMR.
- Risque : un match avec des bots (kills_expected=0 pour certains) fausse la moyenne.

### Piste 3 : réduction du poids KE/DE
`kills_vs_expected` 0.27 → 0.10, `deaths_vs_expected` 0.24 → 0.10,
redistribuer sur `offensive_conversion` et `damage_efficiency`.

- Piste-C avait réduit seulement DvE → Madina +58, Choco -58.
- Réduire les deux ensemble en renormalisant pourrait être plus équilibré.
- À simuler via `--compare-formulas` avant tout commit.

### Piste 4 : composante `lobby_rank`
Rang du joueur dans la liste kills du match (1er/8 = 1.0, 8e/8 = 0.0).
Récompense la dominance absolue en lobby, indépendamment du MMR.

### Piste 5 : reset / decay accéléré des vieux matchs
Fenêtre glissante implicite : multiplier le `weightFactor` de `trueskillUpdate`
par un facteur de décroissance temporelle (matchs vieux de 6+ mois pèsent moins).
Nécessiterait un recalcul séquentiel complet à chaque changement — coûteux.

---

## 7. État actuel du LUSR Madina97294 (post-backfill 2026-05-24)

Après restauration backup + syncs frais (post-bug ART, 2026-05-24) :
- arena_slayer : ~1286 (Argent III) — estimation, à vérifier via dry-run
- BTB : ~1078 (Bronze II) — statistiquement solide (526+ matchs, sigma ~115)
- Cible attendue : fin Platine / début Diamant (~1750-1900)

**Écart estimé** : ~500-600 pts en arena, ~700 pts en BTB.
Aucun fix formule connu ne peut combler cet écart sans régresser les autres joueurs.
La seule piste structurelle propre reste le **LUSR_recent fenêtré**.

---

## 8. Prochaine étape suggérée

Implémenter `GET /api/v1/players/:player/lusr/recent` qui retourne le LUSR calculé
sur les N derniers matchs (N configurable, défaut 50) par chaîne, **sans écriture en DB**.

Basé sur `batchComputeLUSRPreview` avec filtre temporel :
```go
// Dans batchComputeLUSRPreview (ou variante dédiée) :
if windowSize > 0 && len(matches) > windowSize {
    matches = matches[len(matches)-windowSize:]
}
```

Côté UI : afficher `LUSR récent (50 matchs)` en priorité, `LUSR historique` en secondaire/tooltip.
