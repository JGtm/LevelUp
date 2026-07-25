# Plan — Revue analytique Timeseries & Escouade (corrections + outillage de revue)

**Date** : 2026-07-12 — révisé le 2026-07-23 (revue UX) — **RÉÉCRIT le 2026-07-25**
(item V721-10 : le plan avait été doublé par du code livré les 23 et 24/07 ; ~40 % de son
contenu était périmé, dont un item qui aurait rouvert un bug corrigé).
**Branche Git** : `feat/v7.2.1-notion-batch`
**Statut** : **EXÉCUTÉ le 2026-07-25** — reste la tournée visuelle utilisateur (Z1).
**Effort révisé** : ~1,5 j réalisé (estimation initiale 2,5 à 3 j — l'écart vient des lots
devenus sans objet : B2, B4, B7 et une partie de B3).
**Contrat d'exécution** : skill `plan-execution` (ordre strict, gates, statuts
`[x]`/`[~]`/`[!]`, zéro fix hors périmètre — découvertes consignées en fin de fichier).

---

## Contexte

Revue critique du 2026-07-12 : trois passes (Timeseries, Escouade, gisement backend) ont
identifié des défauts de rendu, des redondances possibles et des angles non exploités.
Cadrage utilisateur inchangé : **rien n'est supprimé dans ce chantier** — les candidats
sont SURLIGNÉS à l'écran (lot A), l'utilisateur tranche après vérification visuelle.

### Réécriture du 2026-07-25 — ce qui a changé et pourquoi

Le plan a été écrit le 23/07 puis **jamais exécuté**. Entre-temps, trois livraisons l'ont
partiellement rendu caduc. Vérifications sur pièces (2026-07-25) :

| Item | Ce que disait le plan | État réel constaté |
|---|---|---|
| **B2** rendement combat | « monter le composant orphelin `TimeseriesCombatYield` (`useCombatYieldHistory`) » | **Prémisse FAUSSE** : ni le composant ni le hook n'existent (grep vide sur `apps/web/src`). Ce qui existe : `components/ui/combat-yield-display.tsx`, déjà monté sur Accueil, Synthèse, cartes de match et briefing de session. `OffensiveConversion`/`DefensiveResistance` (`legacymatch/types.go`) ne sont PAS exposés dans `TimeseriesMatchRow` : l'item coûtait un contrat + un composant complet, pas un branchement. |
| **B3** radar | « recalibrer l'axe Score (seuil `350×n` écrasant) » | **Déjà fait** (commit `642ef31f8`) : `teammates_squad_charts_synergy.go:26` définit `ScorePerMinuteP80 = 195.0` et `:34` normalise l'axe Score par un seuil CONSTANT. Le `350×n` cité est aujourd'hui l'axe **Objective**, pas Score. Seul B3(3) (valeur brute au survol) restait valide. |
| **B4** heatmaps d'intensité | « renormaliser les 2 heatmaps par le max global » | **SANS OBJET** : les deux heatmaps ont été REMPLACÉES par un profil médian + enveloppe P25–P75 (commits `2f5eb07de`, `3bac71260`). Il n'y a plus de heatmap à renormaliser. |
| **B5** axes WR/MMR | « corriger Timeseries, vérifier si Escouade partage le défaut » | Défaut réel sur **Timeseries UNIQUEMENT** (`TimeseriesSquadAdapted.tsx` : un axe Y2 partagé par un taux en % et un MMR en milliers). `squadSessionTimelineChart.ts:64-84` a déjà deux axes. Donc **1 badge, pas 2**. |
| **F1a** dominance | « zéro nouveau calcul, champ déjà chargé par `StatsMatchRow` » | **Sous-estimé** : `StatsMatchRow` ne portait PAS le drapeau. Il a fallu l'ajouter (+ le converter canonical), étendre **2 DTO** (`TimeseriesMatchRow`, `SquadMatchHistoryRow`), plus `SquadMatchRow` et son hydratation repo. Aucun SQL nouveau en revanche (la colonne est déjà lue par `LoadPlayerMatchEnrichments`). |

### Décisions utilisateur du 2026-07-25 (fermes — ne pas rouvrir)

- **B7 / DEC-6 — référence « vs historique » de l'Escouade** : **« on ne touche pas »**.
  La baseline reste la **composition exacte** livrée le 24/07, verrouillée par le garde-rail
  `internal/service/teammates/no_raw_squad_intersection_test.go`. **DEC-6 est SUPERSÉDÉE.**
- **B2 — rendement combat sur Timeseries** : **abandon** (« option c »). C'est de la redite
  (le rendement est déjà visible à 4 endroits) et le coût a doublé depuis l'écriture du plan.
  `OffensiveConversion`/`DefensiveResistance` ne sont PAS exposés.

### Rappel des élagages du 2026-07-23 (inchangés)

Supprimés : F2 fatigue intra-session (biais systématique), F3 premier sang (causalité
inversée), F4 part de contribution, F5 solo vs escouade (mauvais emplacement → Découvertes),
F6 breakdown par mode, F7a/F7b (3e lecture du skill ; mu/sigma non relié au vécu),
E1/E2/E3 (form score opaque ; stack V2 intact), D1 heatmap jour×heure.
F1 comeback agrégé remplacé par des drapeaux de dominance PAR MATCH sur la
`OutcomeSequenceTape` (fiable match par match, cf. `internal/analysis/comeback.go`).

## Objectif & critère de succès

Corriger les défauts de lecture, brancher les gains rapides — et que CHAQUE graphe touché,
ajouté ou suspecté porte un badge visible à l'écran (« À vérifier » / « Nouveau » /
« Suppression ? ») avec une note, pour que l'utilisateur balaye les deux pages et statue
sans rien rater. Succès = gates verts + tournée visuelle où chaque badge est statué
(l'entrée est retirée du manifeste une fois l'item validé).

## Hors périmètre (explicite)

- Toute suppression (chart, champ backend, stack V2) — surlignage seulement.
- Bascule des pages Escouade vers `/pages/squad/v2` (DEC-7, chantier dédié ultérieur).
- Toute feature fondée sur `expected_win_prob`.
- Refonte de la définition de l'escouade (**décision utilisateur : on ne touche pas**).
- Le rendement de combat sur Timeseries (**décision utilisateur : abandon**).
- Tous les items supprimés le 2026-07-23.

---

## Décisions (DEC) — état après réécriture

| # | Question | Décision |
|---|---|---|
| DEC-1 | Lots retenus | **A + B (partiel) + C + D + F** — B2/B4/B7 tombent (sans objet ou décision utilisateur) |
| DEC-2 | B2 : Combat Yield côte à côte ? | **CADUQUE** — B2 abandonné (utilisateur, « option c ») |
| DEC-3 | B3 : recalibrage des seuils | **DÉJÀ APPLIQUÉ** hors de ce plan (commit `642ef31f8`) |
| DEC-4 | B4 : normalisation d'intensité | **CADUQUE** — les heatmaps n'existent plus (profil médian + enveloppe) |
| DEC-5 | Data morte backend | **Rien n'est branché** ; tout part en liste cleanup (consigné en Découvertes) |
| DEC-6 | B7 référence « vs historique » | **SUPERSÉDÉE 2026-07-25** — l'utilisateur ne veut pas y toucher ; baseline = composition exacte |
| DEC-7 | Sort du stack `squad_service_v2` | **HORS PÉRIMÈTRE** : chantier dédié ultérieur (inchangé) |
| DEC-8 | Outillage badges conservé après le chantier ? | **Garder** : réutilisable, **inerte quand le manifeste est vide** |
| DEC-9 | Items optionnels E2/E3/F6/F7b | **Sans objet** : supprimés le 2026-07-23 |

---

## Lot A — Outillage de badges de revue (prérequis) — FAIT

Mécanisme : un manifeste central + un badge accolé aux titres de charts. Cycle de vie d'une
entrée : ajoutée par ce chantier → l'utilisateur vérifie à l'écran → l'entrée est RETIRÉE du
manifeste (commit de clôture). **Manifeste vide = aucun badge, aucun nœud DOM** (DEC-8).

- [x] A1 `apps/web/src/lib/review/chart-review.ts` : `ChartReviewStatus`
      (`verify` | `new` | `removal`), `ChartReview { status, note: Record<Locale,string> }`,
      `CHART_REVIEW: Record<string, ChartReview>` (clé = identifiant stable du graphe),
      helper `chartReview(key)` → `undefined` hors tournée.
- [x] A2 `apps/web/src/components/charts/ReviewBadge.tsx` : badge compact (libellé +
      note au survol via `title`/`aria-label`). Tokens : `verify` → `warning`,
      `new` → `info`, `removal` → `destructive` (aucun hex, aucune classe Tailwind de
      couleur). Libellés FR **et** EN dans `apps/web/src/lib/review/i18n.ts`
      (`Record<Locale, Record<ChartReviewStatus, …>>`).
- [x] A3 `ChartCard` : nouvelle prop optionnelle `reviewKey?: string` → badge rendu dans la
      barre de titre. `ChartFromOption` la transmet. Les surfaces non-ChartCard (les deux
      `OutcomeSequenceTape`) posent `<ReviewBadge reviewKey=… />` directement à côté de leur
      intitulé.
- [x] A4 Tests vitest : `components/charts/ReviewBadge.test.tsx` (inerte sans clé / clé
      inconnue, libellé FR, libellé EN, statut `removal`) + `lib/review/chart-review.test.ts`
      (helper + intégrité du manifeste : statut connu, notes FR **et** EN non vides).

**Gate A** : `make check-types` + `make test-web` (joué par le pilote).

## Lot B — Corrections de rendu/donnée

- [x] **B1 — Cesser d'estimer la durée de vie.** La valeur réelle existe et est peuplée
      (`StatsMatchRow.AvgLifeSeconds`, alimentée par `analysis/stats_canonical.go` depuis
      `player_matches_repo.go` / `p.avg_life_seconds`) ; les deux consommateurs utilisaient
      encore le proxy `temps_joué / (morts + 1)`.
      - Backend : `avg_life_seconds` exposé dans `TimeseriesMatchRow` ; helper
        `service.matchAvgLifeSeconds` (valeur réelle → repli proxy → rien) ;
        `buildLifeBuckets` rebasé dessus, avec `slog.DebugContext` comptant les replis
        (`matches_fallback` / `matches_used` / `matches_total`) — jamais de dégradation muette.
      - Front : helper `lib/charts/avgLife.ts` (même ordre de préférence) consommé par
        `TimeseriesAvgLifeTrend`.
      - Badges `verify` sur la courbe ET l'histogramme.
- [!] **B2 — Rendement combat.** NON TRAITÉ, deux raisons cumulées : (1) prémisse du plan
      invalide (aucun composant orphelin `TimeseriesCombatYield` ni hook
      `useCombatYieldHistory` n'existe ; ce qui existe est `ui/combat-yield-display.tsx`,
      déjà monté sur 4 surfaces) ; (2) **décision utilisateur du 2026-07-25 : abandon**
      (« option c ») — redite payée au prix fort depuis que le coût a doublé. Aucun champ
      `OffensiveConversion`/`DefensiveResistance` n'est exposé.
- **B3 — Radar de synergie**
  - [!] B3(1) diagnostic « axe Score à zéro » : sans objet, la cause est identifiée et
        corrigée depuis le 2026-07-24.
  - [~] B3(2) recalibrage : COUVERT AILLEURS — commit `642ef31f8`,
        `teammates_squad_charts_synergy.go:26` (`ScorePerMinuteP80 = 195.0`) et `:34`
        (seuil CONSTANT, comme Survie/Impact).
  - [x] B3(3) tooltip : la valeur BRUTE (déjà portée par `SquadSynergyRadarAxis.Raw`)
        s'affiche à côté de la normalisée, avec une précision adaptée à l'ordre de grandeur
        de l'axe (Combat ~ centaines, Survie ~ 1,6, Score ~ 195/min). Libellé
        « brut » / « raw » via `features/squad/i18n.ts`. Badge `verify`.
- [!] **B4 — Heatmaps d'intensité.** SANS OBJET : les deux heatmaps ont été remplacées par
      un profil médian + enveloppe P25–P75 (`2f5eb07de`, `3bac71260`). Il n'y a plus de
      normalisation per-match à corriger. Les deux profils reçoivent en revanche un badge
      `verify` (ils n'ont jamais été validés à l'écran par l'utilisateur).
- [x] **B5 — Axes taux de victoire / MMR.** `TimeseriesSquadAdapted.TimeseriesSessionPerformance` :
      Y1 = performance [0,100], **Y2 = taux de victoire borné [0,100] avec suffixe `%`**,
      **Y3 = MMR** (axe propre, `offset: 48`, ajouté seulement si le titre fournit le MMR —
      Halo 5 n'a donc aucun axe fantôme). Marge droite élargie en conséquence. Badge `verify`.
  - [~] Chart Escouade : COUVERT AILLEURS — `squadSessionTimelineChart.ts:64-84` sépare déjà
        les axes. **Aucune modification, 1 seul badge** (le plan en prévoyait 2 à tort).
- [~] **B6 — Tri des armes** : déjà traité dans un chantier dédié (constat utilisateur
      2026-07-23). Simple coup d'œil pendant la tournée Z1.
- [!] **B7 — Référence « vs historique » (Escouade).** NON TRAITÉ — **décision utilisateur du
      2026-07-25 : « on ne touche pas »**. La baseline reste la composition exacte livrée le
      24/07 et verrouillée par `no_raw_squad_intersection_test.go`. DEC-6 est supersédée ;
      aucun code n'a été écrit dans son sens.

**Gate B** : `make go-api-test` + `go test ./...` ; **`make openapi-gen` puis
`make generate-types`** (le contrat change : `avg_life_seconds`, `dominance_flag`) ; puis
`make check-types` + `make test-web`. Lectures seules côté DB → pas de tests d'intégration
persist requis.

## Lot C — Redondances : surlignage SEULEMENT

Aucun changement de code hors manifeste. Notes rédigées en question ouverte.

- [x] C1 Badges `verify` sur « Distribution FDA » (`timeseries.fda_distribution`) ET
      « FDA (valeur) » (`timeseries.fda_value_trend`).
- [x] C2 Badges `verify` sur « Progression CSR/LUSR » (`timeseries.skill_progression`) ET
      « Classement + Performance » (`timeseries.skill_rank_perf`).

## Lot D — Data morte backend : consigner, rien brancher, rien supprimer

DEC-5 : **rien n'est branché**, tout part en liste cleanup.

| Champ | Contenu | Décision |
|---|---|---|
| `intensity_tab.heatmap_data` | heatmap jour×heure | Cleanup ultérieur (ex-D1 abandonné) |
| `intensity_tab.score_per_min_data` | score/min par période | Cleanup ultérieur |
| `cumul_tab` | K/D cumulé + rolling 5 | Cleanup ultérieur (recouvre les trends existants) |
| `outcomes_over_time` | V/D par période | Cleanup ultérieur (recouvre Session Perf) |
| `distributions_tab.rolling_wr_buckets` | distribution WR glissant (fenêtre 14) | Cleanup ultérieur |
| `distributions_tab.score_per_min_buckets` | distribution score/min | Cleanup ultérieur |
| `correlation_points` kills↔kd_ratio | scatter jamais affiché | Cleanup ultérieur |

- [x] D2 Tableau consigné dans la section Découvertes ci-dessous.
      La ligne `thought_log` correspondante est à ajouter par le pilote (fichier réservé,
      cf. Z2).

## Lot E — SUPPRIMÉ (révision 2026-07-23)

Réhabilitation du stack V2 abandonnée (form score LOWESS = métrique synthétique opaque
recouvrant les trends existants). Le stack `squad_service_v2` reste INTACT ; son sort =
DEC-7, chantier dédié.

## Lot F — Drapeaux de dominance sur la bande de résultats

Pas de compteur agrégé (les flags ne sont jamais produits sans timeline de score,
cf. `internal/analysis/comeback.go` — un agrégat de période serait silencieusement faussé
par la couverture partielle). Le drapeau s'affiche là où il est fiable : SUR le match.
L'absence de marqueur = match ordinaire OU timeline absente : aucun mensonge.

- [x] F1a **Backend** (coût réel supérieur à l'estimation du plan, cf. tableau de
      réécriture) : `DominanceFlag` ajouté à `legacymatch.StatsMatchRow` et peuplé par
      `analysis.StatsMatchRowFromCanonical` depuis `r.Enrichment.DominanceFlag` ;
      `TimeseriesMatchRow.DominanceFlag` (`json:"dominance_flag,omitempty"`) ;
      `domain.SquadMatchRow.DominanceFlag` hydraté dans `duckdb.SquadRepo.LoadSquadMatches`
      depuis `LoadPlayerMatchEnrichments` (**aucune requête nouvelle** — la colonne était
      déjà lue) ; `domain.SquadMatchHistoryRow.DominanceFlag` renseigné par
      `buildSquadMatchHistory`.
- [x] F1b **Modèle front** : `OutcomePoint.dominance?: DominanceValue` (1..5) dans
      `outcomeSequence.ts` + helper `asDominance(flag)` qui ne retient que 1..5 (0, null et
      codes inconnus d'un futur titre → `undefined`, pas de marqueur inventé). `toRuns`
      propage le point tel quel. Champ ABSENT chez les 4 autres consommateurs de la bande →
      rendu strictement identique.
- [x] F1c **Rendu** : losange (7 px de diagonale) centré sur la cellule du match, À
      L'INTÉRIEUR de la bande (ne touche pas aux brackets de séries), rempli du token
      `narrative-*` du drapeau. Liseré 1 px couleur de fond de carte (`--popover`) —
      **et non `CHART_BG` comme le prévoyait le plan : `CHART_BG` vaut `'transparent'` et
      ne séparerait rien**. Seuil de densité : marqueur omis sous 6 px par match (le
      tooltip porte alors l'information).
- [x] F1d **Tooltip** : la ligne du match est suffixée du libellé du drapeau
      (« · Aquarius (Slayer) — Remontada »). Libellés fournis par les pages via
      `dominanceLabels(locale)`, qui lit les clés `narrative.dominance.*` du manifeste
      `match_view` (FR + EN déjà présents, chargé par les deux pages). Prop optionnelle :
      sans elle, la ligne reste inchangée.
- [x] F1e Aucune légende ajoutée, aucune couleur nouvelle, aucun hex. Les tokens et les clés
      i18n sont désormais centralisés dans `apps/web/src/lib/narrative/dominance.ts`, et
      `ExplorerMatchesTable` a été migré dessus (une seule table pour les 2 surfaces —
      évite la 3e copie interdite par la règle « ≤ 2 copies »).
- [x] F1f **Tests** : `outcomeSequence` (propagation dans `toRuns`, `asDominance`),
      rendu (`renderItem` : losange présent avec drapeau + largeur suffisante, absent sous
      le seuil de densité, absent chez un consommateur sans le champ), tooltip (suffixe
      présent avec libellé, ligne inchangée sans libellé).
- [x] F1g Badges `new` sur les deux bandes (`timeseries.outcome_tape`, `squad.outcome_tape`).

**Multi-titre** : `dominance_flag` est `omitempty` côté Go → absent du JSON pour Halo 5
(pas de timeline de score) → `asDominance` renvoie `undefined` → aucun marqueur, aucun
suffixe, aucune erreur. Dégradation gracieuse par construction, sans test de slug.

**Gate F** : `make openapi-gen` + `make generate-types` (contrat modifié) ; `make go-api-test`
+ `go test ./...` ; `make check-types` + `make test-web`.

## Clôture du chantier

- [ ] Z1 **Tournée visuelle AVEC l'utilisateur** : les deux pages, badge par badge ; chaque
      item statué → entrée retirée de `lib/review/chart-review.ts` dans un commit de clôture.
      Les « Suppression ? » validés partent dans la liste cleanup. **Seul report légitime :
      cette étape requiert l'utilisateur.**
      Badges posés (12) :

      | Clé | Statut | Page |
      |---|---|---|
      | `timeseries.avg_life_trend` | verify | Timeseries / Résumé |
      | `timeseries.life_histogram` | verify | Timeseries / Distributions |
      | `timeseries.session_performance` | verify | Timeseries / Résumé |
      | `timeseries.fda_distribution` | verify | Timeseries / Résumé |
      | `timeseries.fda_value_trend` | verify | Timeseries / Résumé |
      | `timeseries.skill_progression` | verify | Timeseries / Progression |
      | `timeseries.skill_rank_perf` | verify | Timeseries / Progression |
      | `timeseries.intensity_profile` | verify | Timeseries / Progression |
      | `timeseries.outcome_tape` | new | Timeseries / Résumé |
      | `squad.synergy_radar` | verify | Escouade / Contributions |
      | `squad.intensity_profile` | verify | Escouade / Dynamique |
      | `squad.outcome_tape` | new | Escouade / Synergies |

- [!] Z2 Entrée `thought_log` + skill `delivery-checklist` + demande de validation avant
      commit : **NON TRAITÉ ICI** — `.ai/thought_log.md` est réservé à un autre agent sur ce
      chantier multi-agents. Le pilote consigne l'entrée et joue les gates en une passe
      sérialisée. Rappel : push `main` = déploiement prod, prévenir l'utilisateur.

**Gate final** (joué par le pilote, dans cet ordre) :
```bash
make openapi-gen          # contrat : avg_life_seconds + dominance_flag (x2 DTO)
make generate-types
make go-api-lint
make go-api-test
cd apps/go-api && go test ./...
make check-types
make test-web
```

## Protocole de reprise de session

Lire ce fichier (statuts) + la dernière entrée `thought_log` mentionnant « revue
analytique ». Le manifeste `lib/review/chart-review.ts` reflète l'état de la tournée
visuelle : entrées restantes = graphes encore à statuer. Manifeste vide = tournée close.

## Découvertes hors périmètre (consignées, non traitées)

- **Liste cleanup ultérieur (DEC-5, D2)** — champs calculés et servis par
  `timeseries_service` que le front ignore ; à compléter par la tournée Z1 :
  - `intensity_tab.heatmap_data` (heatmap jour×heure — ex-D1 abandonné : fun-fact bruité,
    petits effectifs par case)
  - `intensity_tab.score_per_min_data`
  - `cumul_tab` (K/D cumulé + rolling 5) — recouvre les trends existants
  - `outcomes_over_time` — recouvre Session Perf
  - `distributions_tab.rolling_wr_buckets` (fenêtre 14)
  - `distributions_tab.score_per_min_buckets`
  - `correlation_points` kills↔kd_ratio (scatter jamais affiché)
- **Durée de vie : troisième consommateur non migré.**
  `service/timeseries_service_tabs.go` → `buildCorrelationPoints` calcule encore
  `lifespan = time_played / (deaths + 1)` pour les scatters « Durée de vie vs frags » et
  « Durée de vie vs morts » de l'onglet Distributions. Le helper `matchAvgLifeSeconds`
  existe et s'y appliquerait tel quel (3 lignes), mais l'item B1 ne nommait que deux
  consommateurs : **non traité pour ne pas modifier un graphe hors périmètre**. À reprendre
  dans le chantier cleanup — sinon l'histogramme et le scatter racontent deux histoires
  différentes de la même métrique.
- **3e copie de la table de dominance, non migrée** :
  `apps/web/src/features/match-view/MatchHeader.card.tsx` porte encore son propre
  `DOMINANCE_TOKENS`. `ExplorerMatchesTable` a été migré sur la table canonique
  (`lib/narrative/dominance.ts`) ; match-view ne l'a pas été car ce répertoire était
  **réservé à un autre agent** pendant ce chantier. On est donc à 2 copies (limite de la
  règle, pas au-delà) : à migrer dès que le répertoire se libère, sinon la 3e divergence
  reviendra.
- **Doc obsolète** : `analysis/stats_canonical.go`, en-tête de
  `StatsMatchRowFromCanonical`, affirme encore que `OffensiveConversion`/
  `DefensiveResistance` « restent nil » alors qu'ils sont calculés plus bas
  (`ComputeCombatYield`). Commentaire faux, pas de bug — hors périmètre.
- **Candidat futur chantier Synthèse** (consigné 2026-07-23) : **Solo vs escouade** (ex-F5)
  — baseline KDA/WR/perf du main avec vs sans escouade (flag `is_with_friends`), barres
  groupées avec effectifs n + phrase narrative. Emplacement pressenti : page Synthèse (ni
  Timeseries, dont l'objet est le temps, ni Escouade, cadrée sur les matchs joués ensemble).
- **`/pages/squad/v2`** : stack dormant, jamais passé en prod, non fiable par défaut. Sort =
  DEC-7, chantier dédié.
