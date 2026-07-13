# Plan — Histogramme momentum sur la carte Dominance (Match View)

**Date** : 2026-07-12
**Branche Git** : `feat/matchview-momentum` (à créer depuis `main` — jamais de travail sur `main`)
**Statut** : Rédigé — DEC-1..7 tranchées ci-dessous, exécution après validation utilisateur
**Effort estimé** : ~0,5 j (frontend uniquement, zéro changement backend)
**Contrat d'exécution** : skill `plan-execution` (ordre strict, gates, statuts `[x]`/`[~]`/`[!]`,
zéro fix hors périmètre — les découvertes vont dans la section dédiée en fin de fichier).

---

## Contexte & objectif

La carte « Dominance » (`MatchTugOfWarChart.tsx`, spec match_view.10, onglet Détails §1
« Déroulé du match ») rend aujourd'hui la domination par tranche de 30 s en **barres
empilées normalisées 0–100 %**. Ce rendu masque l'amplitude (une tranche 2-1 et une
tranche 8-4 se ressemblent) et duplique l'info de cumul avec « Frags cumulés »
(match_view.09) juste au-dessus.

**Objectif** : remplacer ce rendu par un **histogramme momentum divergent** (référence
visuelle : Squeeze Momentum TradingView) — une barre signée par tranche :

- valeur = `delta = kills alliés − kills ennemis` dans la tranche ;
- barre au-dessus de zéro colorée **`team-ally`**, en dessous **`team-enemy`**
  (tokens configurables par l'utilisateur : Réglages → Accessibilité, appliqués par
  `theme-provider.tsx` sur `--ac-team-ally` / `--ac-team-enemy`) ;
- **intensité** : opacité pleine quand le momentum se renforce, atténuée quand il
  s'essouffle (détail DEC-4) ;
- kill feed (lanes + scatter + vagues collectives) **conservé** sur le même axe X.

**Critère de succès** : sur un match riche en events (Infinite OU H5), la carte montre
d'un coup d'œil qui domine, avec quelle amplitude, et les bascules de momentum ; le
changement de couleur d'équipe dans les réglages se répercute immédiatement ; un match
sans données de combat (voie live-only, tout titre) affiche l'EmptyState comme
aujourd'hui ; `make check-types` et `make test-web` verts.

**Halo 5 — couvert nominalement, zéro travail spécifique** (vérifié sur pièces
2026-07-12, correction d'une hypothèse initiale fausse) : le kill-feed H5 est natif et
persisté en local (`killer_victim_pairs` + `weapon_kills` + `kill_positions`) ; la voie
repo-first du Match View synthétise les events kill/death depuis `killer_victim_pairs`
quand `highlight_events` ne porte que des médailles
(`applyKVSynthesisIfNeeded`, `internal/service/match_view_builders_combat.go`) →
`combat_tab.highlight_events` et `combat_tab.tug_of_war` sont peuplés pour H5 comme
pour Infinite. Le seul cas EmptyState est un match servi live-only (absent du substrat
local → `CombatTab` vide + `combat_narrative_unavailable`,
`match_view_canonical.go`), indépendant du titre. La section n'est pas gatée par
capability côté front (disponibilité pilotée par la donnée via le guard
`hasKillEvents`) — cohérent, aucun changement de capability requis.

## Ce qui ne change PAS

- **Backend** : aucun changement. `combat_tab.tug_of_war` (bornes de bins) et
  `combat_tab.highlight_events` suffisent. On continue de **recomputer les counts par
  équipe côté front** depuis les events (les `team_kills`/`enemy_kills` du backend sont
  un delta net, un seul non-nul par bin — commentaire existant lignes 125-127).
- Props du composant (`bins`, `events`, `scoreboard`, `meXUID`, `t`), emplacement dans
  `MatchViewPage.tsx` (grille « Dominance | Cadence »), hauteur 360.
- Guard EmptyState `hasKillEvents` (matchs sans données de combat : voie live-only au
  combat tab vide, tout titre — H5 nominal rend le chart via le kill-feed synthétisé).
- Kill feed : lanes, scatter par kill, détection et rendu des vagues (`detectTeamWaves`).
- Nom du composant/fichier `MatchTugOfWarChart.tsx` (aligné sur le type de contrat
  `MatchTugOfWarBin` ; « tug of war » décrit toujours la sémantique).

## Hors périmètre (explicite)

- Pas de wrapper générique `DivergingBarChart` dans `components/charts/` : un seul
  usage → la factorisation attendrait une 2e occurrence (règle des copies inversée).
  Si une version Escouade/Solo est décidée plus tard, on extraira à ce moment-là.
- Pas de version momentum inter-matchs (Escouade/Solo) — écartée avec l'utilisateur
  le 2026-07-12.
- Pas de refonte de `MatchKDCumulChart`, `MatchCadenceChart`, `MatchFragDiffChart`.
- Pas de nouveau token sémantique (les variantes d'intensité sont des alpha-mix
  structurels sur les tokens existants, précédent documenté dans le fichier même).

---

## Décisions tranchées (DEC)

| # | Décision | Justification |
|---|---|---|
| DEC-1 | Titre de carte inchangé : « Dominance » (FR et EN, clé `combatTugOfWarTitle`) | Concept identique, seul le rendu change ; « Momentum » serait un anglicisme (règle UI FR) |
| DEC-2 | Kill feed conservé intégralement (lanes, scatter, vagues, layout 2 grilles) | Info unique ancrée sur le même axe X, pas redondante |
| DEC-3 | Suppression des markPoints de cumul (labels encadrés leader `teamCum`/`enemyCum`) ; les cumuls restent dans le tooltip | Redondants avec « Frags cumulés » (match_view.09) affiché juste au-dessus dans la même section |
| DEC-4 | Intensité : pour `delta > 0`, opacité 0.9 si `delta[i] > delta[i-1]` sinon 0.45 ; pour `delta < 0`, opacité 0.9 si `delta[i] < delta[i-1]` sinon 0.45 ; premier bin non nul : 0.9 | Adaptation directe du schéma 4 teintes TradingView (vif = momentum qui se renforce) sans nouveau token : alpha-mix sur `team-ally`/`team-enemy` résolus |
| DEC-5 | `delta = 0` → pas de barre (pas de tick artificiel) ; lisibilité assurée par tooltip `trigger: 'axis'` et par le kill feed qui montre l'activité | Honnêteté visuelle : momentum nul = rien ne penche |
| DEC-6 | Échelle Y symétrique dynamique : `yMax = max(1, max(abs(delta)))`, barres dans `[-yMax, +yMax]` ; positions des lanes/labels recalculées en fonction de `yMax` (plus de constantes 0–100 hardcodées) | Le mock 0–100 disparaît avec la normalisation |
| DEC-7 | `markLine` horizontale à `y = 0` (dashed, `tc.splitLine`) remplace la ligne 50 % | L'axe de symétrie devient zéro |

## Blockers / risques

- **B1 (faible)** — Légende ECharts : une seule série bar avec `itemStyle` par point ne
  donne pas deux entrées de légende. Parade retenue : **deux séries bar** superposées
  (même `stack`), « Mon équipe » = deltas positifs (sinon `null`), « Adversaires » =
  deltas négatifs (sinon `null`) — la légende existante (`combatTeamLabel` /
  `combatEnemyLabel`) reste fonctionnelle sans nouvelle string i18n.
- **B2 (cosmétique)** — Repositionnement des lanes kill feed avec l'échelle dynamique :
  à caler visuellement (facteurs `yMax × k`), vérification écran en Phase 4.

---

## Phase 1 — Centraliser l'alpha-mix hex (pré-requis, règle « ≤ 2 copies »)

Deux copies identiques de `hexToRgba(hex, alpha)` (contexte canvas/ECharts) existent
déjà : `MatchTugOfWarChart.tsx:88` et `MatchImpactBadgesBar.tsx:54`. Ce plan en
ajouterait un 3e usage intensif → centralisation + garde-rail obligatoires AVANT le
rendu. NB : la variante `hexToRgba(cssVar, alpha)` de
`components/ui/match-card-presentation.ts:16` est un autre pattern (CSS `color-mix`
sur var, contexte DOM) — elle reste en place, hors périmètre.

- [x] 1.1 Ajouter `hexToRgba(hex: string, alpha: number): string` dans
      `apps/web/src/components/charts/_utils.ts`, avec le commentaire justificatif
      existant (alpha-mix structurel sur hex résolu via token, pas un choix sémantique)
      et une note distinguant la variante `color-mix` CSS. FAIT (`_utils.ts`, note
      color-mix + renvoi au garde-rail).
- [x] 1.2 Migrer `MatchTugOfWarChart.tsx` et `MatchImpactBadgesBar.tsx` vers l'import ;
      supprimer les deux copies locales. FAIT (import depuis `@/components/charts/_utils`,
      les 2 copies locales supprimées).
- [x] 1.3 Garde-rail : `apps/web/src/components/charts/hex-alpha.guard.test.ts` sur le
      modèle de `lib/query/keys.guard.test.ts` — scan de `apps/web/src/features/**` :
      interdire toute déclaration locale `function hexToRgba(` / `const hexToRgba`. FAIT
      (test node-env, 1 test vert).

**Gate Phase 1** : `make check-types` + `make test-web` verts (le nouveau guard passe).
PASSÉ 2026-07-13 : typecheck OK ; vitest 253 fichiers / 2144 tests verts (14 skipped) ;
eslint 0 sur les 4 fichiers touchés.

## Phase 2 — Logique pure momentum + tests unitaires

- [x] 2.1 Nouveau fichier `apps/web/src/features/match-view/_momentum.ts` :
      `computeMomentumBins(bins, events, xuidMeta)` → tableau par bin
      `{ delta, teamKills, enemyKills, cumTeam, cumEnemy, trend: 'up' | 'down' }`.
      Le calcul d'affectation kill→bin/équipe est **déplacé** (pas dupliqué) depuis
      `MatchTugOfWarChart.tsx` (boucle events + `fracInBin`, lignes ~128-168 actuelles) ;
      la liste `KillEvent[]` (pour scatter/vagues) est retournée aussi pour que le
      composant n'itère les events qu'une fois. `trend` suit DEC-4. FAIT (module pur,
      retour `{ momentum, kills }` ; `trend` via `computeTrend(delta, prevDelta)`,
      prevDelta=0 avant le 1er bin → 1er bin non nul = 'up' ; delta 0 → 'down' neutre).
      NB : le **déplacement effectif** de la boucle hors de `MatchTugOfWarChart.tsx`
      (suppression de la copie dans le composant) est réalisé en Phase 3 lors de la
      réécriture de `buildOption` — Phase 2 introduit la source pure + ses tests sans
      encore débrancher l'ancien rendu (option « pas encore touché » du gate Phase 2).
      Les types `MomentumBin/MomentumKill/MomentumData/MomentumTrend` restent **internes**
      en Phase 2 (garde-rail pre-push `knip-ratchet` : aucun consommateur externe encore →
      un export inutilisé = régression code mort) ; ils seront exportés en Phase 3 quand
      `MatchTugOfWarChart` les importe. Seul `computeMomentumBins` est exporté (utilisé
      par le test).
- [x] 2.2 Tests `_momentum.test.ts` (vitest, colocalisé) : (a)–(g) tous écrits et verts
      (7 tests). FAIT.

**Gate Phase 2** : `make test-web` vert (a–g passent) ; `MatchTugOfWarChart.tsx` compile
encore (l'ancien rendu consomme provisoirement le nouveau helper ou n'est pas encore
touché — au choix de l'exécutant, mais Phase 2 close = tests verts sur les deux états).
PASSÉ 2026-07-13 : `_momentum.test.ts` 7/7 verts ; typecheck OK ; vitest global 254
fichiers / 2151 tests verts ; eslint 0 sur les 2 nouveaux fichiers ; ancien
`MatchTugOfWarChart.tsx` intact et compile (débranché en Phase 3).

## Phase 3 — Rendu histogramme divergent

Réécriture de `buildOption` dans `MatchTugOfWarChart.tsx` :

- [ ] 3.1 Supprimer : normalisation `teamPct`/`enemyPct`, markPoints de cumul
      (`cumulMarkPoints`, DEC-3), markLine 50 %, constantes de layout 0–100
      (`teamCumLabelY = 112`, etc.). Zéro code mort résiduel.
- [ ] 3.2 Deux séries bar signées (B1) : positifs `team-ally`, négatifs `team-enemy`,
      `itemStyle` par point avec opacité DEC-4 via `hexToRgba(resolveToken(...), α)` ;
      `barCategoryGap` serré pour l'effet histogramme (cf. screenshot de référence).
- [ ] 3.3 `markLine` à `y = 0` (DEC-7) ; échelle symétrique dynamique (DEC-6) ;
      lanes/scatter/vagues repositionnés en fonction de `yMax` (grille double conservée).
- [ ] 3.4 Tooltip `trigger: 'axis'` sur les barres : tranche horaire, delta signé,
      détail `X kills / Y kills`, cumuls `cumTeam` / `cumEnemy` (remplace les labels
      supprimés). Tooltips scatter/vagues inchangés (`trigger: 'item'` par série).
- [ ] 3.5 Mettre à jour l'en-tête doc du fichier (match_view.10 : décrire le rendu
      momentum, plus le stacked 0–100 %).
- [ ] 3.6 Seuils : fichier ≤ 500 L et `buildOption` ≤ 80 L après réécriture — 
      l'extraction `_momentum.ts` y contribue ; si `buildOption` dépasse, extraire des
      sous-fonctions de construction de séries dans le même fichier ou `_momentum.ts`.

**Gate Phase 3** : `make check-types` + `make test-web` verts ;
`grep -rn "hexToRgba\|#[0-9a-fA-F]\{6\}" apps/web/src/features/match-view/` ne montre
aucune déclaration locale ni hex en dur (hors commentaires).

## Phase 4 — i18n, vérification visuelle, clôture

- [ ] 4.1 i18n : `combatTugOfWarTitle` inchangé (DEC-1). Si un libellé de tooltip
      s'ajoute (ex. « Delta »), le déclarer FR **et** EN dans
      `features/match-view/i18n.ts` (parité garantie par le typage `Record<Locale, T>`).
- [ ] 4.2 Vérification visuelle (serveur dev + MCP browser) :
      (a) match Halo Infinite riche en events → barres signées lisibles, intensités
          visibles, kill feed aligné, tooltip complet ;
      (b) match Halo 5 (substrat local) → histogramme rendu à l'identique depuis le
          kill-feed synthétisé (kills + morts depuis `killer_victim_pairs`),
          affectation d'équipe correcte via le scoreboard ;
      (c) réglage couleur équipe (Réglages → Accessibilité) modifié → l'histogramme
          reflète immédiatement le choix (tokens `team-ally`/`team-enemy`) ;
      (d) match sans données de combat (servi live-only) → EmptyState inchangé ;
      (e) thème clair ET sombre.
- [ ] 4.3 Skill `delivery-checklist` ; entrée `thought_log.md` (statut Complété) ;
      demander validation utilisateur avant commit (jamais de commit non demandé).

**Gate final** : tous les items `[x]`/`[~]`/`[!]` statués, commandes de gate rejouées :

```bash
make check-types
make test-web
```

---

## Protocole de reprise de session

Lire ce fichier (statuts des items) + la dernière entrée `thought_log.md` mentionnant
« momentum ». Reprendre à la première case non statuée de la première phase non close.
Une phase est close quand tous ses items sont statués ET son gate est passé.

## Découvertes hors périmètre (à consigner, ne pas traiter)

- (vide)
