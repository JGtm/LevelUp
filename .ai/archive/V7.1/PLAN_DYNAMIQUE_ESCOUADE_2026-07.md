# PLAN — Onglet Dynamique, Solde de vies, Écart d'engagement cumulé, essais visuels

> Date : 2026-07-23 · Branche cible : `feat/squad-dynamique`
> Contrat d'exécution : skill `plan-execution` (ordre strict, gates, statuts
> `[x]` fait / `[~]` couvert ailleurs (référence) / `[!]` non traité (justification écrite),
> aucune case vide à la clôture, zéro fix hors périmètre).
> Reprise de session : lire ce fichier (statuts) + `git log --oneline -10` sur la branche.

## Objectif et critère de succès

Quatre évolutions des concepts engagement / rendement / résistance :

1. Nouvel onglet Escouade **« Dynamique »** regroupant intensité, engagement,
   rendement, résistance (+ les nouveaux graphes ci-dessous).
2. **Séparation Rendement / Résistance** en deux graphes multi-joueurs (tous les
   joueurs affichés en même temps, plus de switch 1-joueur).
3. Nouvelle métrique **« Balance des dégâts »** (carry/liability) :
   `(dégâts infligés − dégâts subis) / PV-pour-tuer du titre`, cumulée — sur
   l'onglet Dynamique (1 courbe/joueur) et la page Session (aire cumulée + KPI).
4. **Écart d'engagement cumulé** (résidu × durée, en événements) — Timeseries,
   Dynamique, Session.
5. Essai visuel : **aire entre la courbe Rendement et la ligne « 1 vie »** sur
   Timeseries (jetable si non concluant).

Succès = les 3 onglets Escouade fonctionnels sur Halo Infinite ET Halo 5 (avec
dégradations propres), nouveaux graphes visibles et validés visuellement,
`make check-types` + vitest + suite Go vertes, i18n FR/EN complet.

## Décisions produit (TRANCHÉES — ne pas rouvrir en cours d'exécution)

| Sujet | Décision |
|---|---|
| Nom onglet | FR « Dynamique » / EN « Dynamics » |
| Titre carry | FR « Balance des dégâts » (cumul : « Balance des dégâts cumulée ») / EN « Damage balance » — axe/valeurs exprimés en vies (dégâts nets ÷ PV-pour-tuer) |
| Formule carry | `(damage_dealt − damage_taken) / hpToKill` par match ; `null` si l'un des deux absent |
| Carry sur H5 | MASQUÉ (pas de `damage_taken`) — aucun fallback FDA déguisé |
| Titre engagement cumulé | FR « Écart d'engagement cumulé » / EN « Cumulative engagement gap » |
| Unité engagement cumulé | événements en excès/déficit = Σ(résidu évén./min × durée_min) — la durée vient du backend (P4), pas de cumul du résidu brut |
| Rendement/Résistance | 2 ChartCards multi-joueurs (couleur = joueur, repère « 1 vie » sur les deux) ; le mode toggle 1-joueur est SUPPRIMÉ (code mort) |
| Aire Timeseries | Rendement uniquement, `areaStyle.origin: oneLife` (ECharts 5.6 OK), opacité ~0.10 ; Résistance reste en pointillé sans aire |
| Factorisation cumul | 3e copie du cumul signé → extraire un helper générique de `cumulativeFdaGap` (carry-forward D5 conservé) + garde-rail |
| Session — placement | Les 2 nouveaux charts s'insèrent dans `SessionChartStack`, avec échelle partagée A/B (`_compareScale`) comme `SessionFdaGapCumulative` |

## Pré-requis / blocker

- [ ] **B0** — Un WIP est en cours sur `feat/career-medals-and-fixes` (season pass
      badge, fichiers modifiés non commités). Règle : ne pas changer de branche
      pendant un travail en cours. Ce chantier démarre APRÈS livraison/commit du
      WIP, par `git checkout main && git pull` puis
      `git checkout -b feat/squad-dynamique`. L'utilisateur tranche le moment.

## Phase 1 — Onglet « Dynamique » (réceptacle)

- [x] 1.1 Route file-based
      `apps/web/src/routes/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique.tsx`
      (calquée sur `contributions.tsx`). `routeTree.gen.ts` régénéré via l'API
      programmatique `@tanstack/router-generator` (voir Découvertes), diff purement
      additif (26 lignes).
- [x] 1.2 `features/squad/SquadDynamiquePage.tsx` créée : `SquadIntensityHeatmapChart`,
      `SquadEfficiencyChart`, `SquadEngagementSection` DÉPLACÉS depuis
      `SquadContributionsPage.tsx` (blocs JSX + dérivations
      `engagementTeammates`/`engagementMatchIds`/`intensityProfileLocalized` +
      imports retirés de Contributions). Même mécanisme de données : `useSquadContext`
      (SquadContext fourni par SquadLayout) — aucune query key propre.
- [x] 1.3 Navigation 3e onglet ajoutée dans `SquadLayout.tsx` (`dynamiqueRoute` +
      `isDynamique` + `<Link>`) ; libellés `nav.dynamique` FR « Dynamique » /
      EN « Dynamics » ajoutés à `SquadText` + FR_TEXT + EN_TEXT (parité typée).
- [x] 1.4 Tests : `SquadContributionsPage.test.tsx` adapté (mock + test engagement
      retirés, sections déplacées) ; `SquadDynamiquePage.test.tsx` créé (mount +
      sections présentes + match_ids engagement ASC cap 15 déplacé ici).

**Gate P1** : `make check-types` · `npx vitest run src/features/squad` (hors
sandbox, cf. mémoire vitest) · vérif visuelle des 3 onglets (dev :8000).

> Journal P1 [2026-07-23] — Complété. `npx tsc --noEmit` exit 0 (aucune erreur).
> `npx vitest run src/features/squad` : 34 fichiers / 259 tests passés (dont le
> nouveau smoke Dynamique et le Contributions adapté). Vérif visuelle :3 onglets
> NON exécutée (dev :8000 hors périmètre agent — à faire par le superviseur).

## Phase 2 — Séparation Rendement / Résistance (multi-joueurs)

- [x] 2.1 `features/squad/charts/squadEfficiencyChart.ts` : `buildSquadRendementMultiOption`
      généralisé et renommé `buildSquadEfficiencyMultiOption`, paramétré par
      `metric: 'damagePerKill' | 'damagePerDeath'` (helper interne `metricValue`),
      repère « 1 vie » conservé sur les deux (série fantôme hors légende).
- [x] 2.2 `SquadEfficiencyChart.tsx` : rend DEUX ChartCards (`rendementCardTitle`,
      `resistanceCardTitle`), 1 courbe/joueur colorée par joueur, légende ECharts
      native togglable. H5 (`hasResistance === false`) → seule la carte Rendement
      est rendue. Props `title`/`monoTitle` remplacées par `infoTooltip?: ReactNode`
      (rendu à côté du titre Rendement) ; `SquadDynamiquePage` adaptée.
- [x] 2.3 Supprimés : `buildSquadEfficiencyTrackOption` (+ `EfficiencyTrackOpts`),
      les boutons segmentés 1-joueur, la légende footer SVG, l'état `selectedPlayer`.
      Grep callers : `buildSquadEfficiencyTrackOption` n'avait qu'UN caller
      (SquadEfficiencyChart). Gradients `offensiveDamageGradient`/`defensiveDamageGradient`
      CONSERVÉS (autres callers : `features/timeseries/TimeseriesSquadAdapted.tsx`
      + leurs tests). Aucun test associé au track builder n'existait (rien à supprimer).
- [x] 2.4 i18n `features/squad/i18n.ts` : ajout `efficiencySeries.rendementCardTitle`
      / `resistanceCardTitle` (FR « Rendement — dégâts par frag » / « Résistance —
      dégâts par mort » ; EN « Offensive efficiency — damage per kill » /
      « Defensive resistance — damage per death »), parité typée. Clés devenues
      mortes retirées : `title`, `rendementTitle`, `rendementLabel`, `resistanceLabel`.
- [x] 2.5 Tests : `charts/squadEfficiencyChart.test.ts` créé (6 cas : vide, 2 métriques,
      repère/légende, bornes d'axe incluant le repère, joueur sans donnée). Le test
      Dynamique (P1) mocke `SquadEfficiencyChart` → non impacté par la refonte interne.

**Gate P2** : `make check-types` · `npx vitest run src/features/squad` · vérif
visuelle Halo Infinite ET Halo 5 (dégradation mono-carte).

> Journal P2 [2026-07-23] — Complété. `npx tsc --noEmit` exit 0. `npx vitest run
> src/features/squad` : 35 fichiers / 265 tests passés (+1 fichier / +6 tests vs P1 :
> le nouveau builder test). Vérif visuelle Infinite/H5 (:8000) NON exécutée
> (hors périmètre agent — superviseur).

## Phase 3 — « Balance des dégâts » (frontend uniquement)

- [x] 3.1 `lib/charts/cumulativeSeries.ts` créé : `cumulativeSigned` (cumul signé +
      report carry-forward D5), `meanOfValid` (moyenne sur points valides),
      `finiteOrNull` (garde D5 partagée). `cumulativeFdaGap.ts` DÉLÈGUE (helper
      local `gapOf` → `cumulativeSigned`/`meanOfValid`, plus aucune impl de
      `carryForward`). `cumulativeFdaGap.guard.test.ts` : ALLOWED = {cumulativeSeries.ts,
      cumulativeFdaGap.ts} (délégant documenté), regex `\bcarryForward\b` inchangée.
- [x] 3.2 `lib/charts/netLives.ts` : `netLives(damageDealt, damageTaken, hpToKill)`
      → `number | null` (null si terme manquant/non-fini OU barème invalide ;
      division par le barème du titre). Tests `netLives.test.ts`.
- [x] 3.3 `features/squad/charts/squadNetLivesChart.ts` (`cumulativeNetLivesSeries` +
      `buildNetLivesCumulativeOption`, calqué `squadFdaGapChart.ts`) + composant
      `SquadNetLivesChart.tsx` (1 courbe cumulée/joueur, couleur par joueur, markLine 0,
      InfoTooltip formule). Monté dans `SquadDynamiquePage`. Masquage via capability
      `damage_taken` (`useProvidesDamageTaken`) — voir Découvertes (gate title-agnostic
      choisi plutôt que sonder les rows).
- [x] 3.4 `features/session-detail/SessionNetLivesCumulative.tsx` (calqué
      `SessionFdaGapCumulative.tsx`) — aire signée `divergentZeroGradient` ancrée à 0 +
      markLine 0 + pastille KPI « balance moyenne par match » (`meanOfValid`). yDomain
      via `_compareScale.netLives` (nouveau champ ; `computeCompareScale` prend `hp`,
      passé par `SessionDetailPage`). Branché dans `SessionChartStack` (dense + principal,
      après `fdaGap`). Masquage capability `damage_taken`.
- [x] 3.5 i18n : `features/squad/i18n.ts` (`netLives.title`/`netLives.tooltip` FR/EN,
      parité typée) + `session.toml` (`chart_net_lives_title`, `net_lives_series`,
      `net_lives_match`, `net_lives_average_caption`, `net_lives_average_unit`,
      `net_lives_tooltip`) régénéré via `node scripts/build_i18n_manifests.mjs`
      (seul `generated/session.ts` modifié). Barème injecté via `substituteHpToken`
      ({{HP}}). Titres FR « Balance des dégâts cumulée » / EN « Cumulative damage balance ».
      Valeurs/axe en vies (signées, 0-centrées), positif = porte l'équipe.
- [x] 3.6 Tests : `netLives.test.ts`, `cumulativeSeries.test.ts`,
      `charts/squadNetLivesChart.test.ts`, `SquadNetLivesChart.test.tsx` (gate
      capability), `SessionNetLivesCumulative.test.tsx` (pur + gate + KPI).

**Gate P3** : `make check-types` ·
`npx vitest run src/lib/charts src/features/session-detail src/features/squad` ·
vérif visuelle Session + Dynamique · vérif H5 = masqué proprement.

> Journal P3 [2026-07-23] — Complété. `npx tsc --noEmit` exit 0. `npx vitest run
> src/lib/charts src/features/session-detail src/features/squad` : 58 fichiers /
> 401 tests passés (dont guard cumulativeFdaGap toujours vert après ajout de
> cumulativeSeries.ts à l'allowlist). Vérif visuelle Session/Dynamique + H5 masqué
> (:8000) NON exécutée (hors périmètre agent — superviseur).

## Phase 4 — Écart d'engagement cumulé (Go + web)

Backend (durées absentes des contrats engagement — seul travail Go du chantier) :

- [x] 4.1 `internal/domain/engagement_score.go` : `DurationSeconds int64`
      (`duration_seconds`) ajouté à `EngagementMatchSummary` ; `DurationsSeconds
      []int64` (`durations_seconds`) ajouté à `SquadEngagementSession`.
- [x] 4.2 Durée remplie : `computeMatchSummary` (helper `durationSecondsFromContext`
      = (EndTimeMS−StartTimeMS)/1000) ; binning `engagementBucketAcc.durationSeconds`
      SOMMÉ par bucket (session/week/month), reporté dans `finalizeEngagementBuckets`.
      Tests : `TestAggregateEngagementBySession_DurationSummedPerBin` +
      `TestRollupEngagementByPeriod_DurationSummedPerBin`.
- [x] 4.3 `engagement_squad_service.go` : `DurationsSeconds` initialisé dans
      `GetSquadSession`, rempli dans `appendMatchToSession` (aligné sur Labels),
      durée posée dans `computeMatchBundle`. Tests `engagement_squad_service_test.go`
      (alignement Labels + cas vide), mock `memoCountingEngagementRepo` réutilisé.
- [x] 4.4 openapi.yaml (bloc manuel `components.schemas`) mis à jour pour les 2
      schémas + `npm run generate-types` → `generated.ts` (diff +3 lignes additif,
      idempotent au 2e run). Drift test `TestOpenAPISchemaDrift` vert (schémas
      réconciliés Huma↔manuel). Interfaces miroir manuelles `types.ts` mises à jour.

Frontend :

- [x] 4.5 `lib/charts/engagementGap.ts` : `engagementGapEvents(residualPerMinute,
      durationSeconds)` = résidu × (durée/60), `null` si terme absent/non-fini
      (report D5) ; cumul délégué à `cumulativeSigned` (helper P3). Tests `engagementGap.test.ts`.
- [x] 4.6 `features/timeseries/TimeseriesEngagementGapTrend.tsx` (calqué
      `TimeseriesFdaGapTrend`) : résidu solo = `pace_joueur − pace_attendu`, aire
      cumulée `divergentZeroGradient` + markLine 0. Réutilise `useEngagementTimeseries`
      (dédup cache). Monté sous le `FeatureGate engagement` de `TimeseriesPage.progression`.
- [x] 4.7 `features/squad/charts/squadEngagementGapChart.ts` +
      `SquadEngagementGapChart.tsx` : 1 courbe cumulée/joueur, résidu =
      `pace_observed − team_expected` × durée/60, couleur par joueur, markLine 0.
      Réutilise `useSquadEngagementSession` (dédup). Monté sous le `FeatureGate
      engagement` de `SquadDynamiquePage`.
- [x] 4.8 `features/session-detail/SessionEngagementCumulative.tsx` : cumul de
      `engagement_score × durée/60`, zip `match_series` (index) ↔ rows (start_time) ;
      aire divergente + markLine 0 ; `_compareScale.engagementGap` (nouveau domaine A/B) ;
      branché dans `SessionChartStack` sous le `FeatureGate engagement`, après les barres.
- [x] 4.9 i18n : `engagement.toml` (`engagement.cumulative_gap.*` : title FR « Écart
      d'engagement cumulé »/EN « Cumulative engagement gap », axis « événements
      (excès/déficit) »), `session.toml` (`chart_engagement_cumulative_title` +
      series/match/axis), `features/squad/i18n.ts` (`engagementGap.title`/`tooltip`
      FR/EN). Manifests régénérés.
- [x] 4.10 Tests : Go (4.2 binning ×2, 4.3 squad ×2) ; web : `engagementGap.test.ts`,
      `squadEngagementGapChart.test.ts` (builder), `SessionEngagementCumulative.test.tsx`
      (pur : compute + build). Le chart Timeseries réutilise les helpers testés
      (`cumulativeSigned`/`engagementGapEvents`).

**Gate P4** : `cd apps/go-api && go test ./internal/service/... ./internal/api/...`
· `make generate-types` sans diff résiduel · `make check-types` · vitest ciblés ·
vérif visuelle Timeseries + Dynamique + Session.

> Journal P4 [2026-07-23] — Complété. `go test ./internal/service/... ./internal/api/...
> ./internal/domain/...` : tout vert (drift OpenAPI inclus). `npm run generate-types`
> idempotent (diff +3 lignes). `npx tsc --noEmit` exit 0. `npx vitest run src/lib/charts
> src/features/engagement src/features/timeseries src/features/session-detail
> src/features/squad` : 67 fichiers / 462 tests passés. Vérif visuelle (:8000) NON
> exécutée (hors périmètre agent — superviseur).

## Phase 5 — Essai : aire Rendement → ligne « 1 vie » (Timeseries)

- [~] 5.1 `features/timeseries/TimeseriesSquadAdapted.tsx` (section Rendement &
      Résistance) : `areaStyle` ajouté sur la SEULE série Rendement (dégâts/frag),
      `origin: hp` (barème « 1 vie » du titre, origin numérique ECharts >= 5.3.2),
      `opacity: 0.1`, couleur = même `offensiveDamageGradient(dmgKill, hp)` que le
      trait (aucune couleur hex nouvelle). Résistance (pointillé) inchangée, sans aire.
      Modification localisée d'UNE propriété (trivialement revertable). Code posé,
      gates verts. Statut `[~]` : la **validation visuelle utilisateur RESTE OUVERTE**
      (thème clair + sombre) — décision go/no-go en attente. Non concluant → revert de
      la propriété `areaStyle`, item repassé `[!]` avec justification.

**Gate P5** : vérif visuelle par l'utilisateur (thème clair + sombre). Non
concluant → revert du commit P5, item passé `[!]` avec justification.

> Journal P5 [2026-07-23] — Code posé. `npx tsc --noEmit` exit 0 ; `npx vitest run
> src/features/timeseries` : 5 fichiers / 43 tests passés. Diff = 1 propriété
> `areaStyle` ajoutée (+ commentaire) sur la série Rendement. Validation visuelle
> utilisateur NON réalisée (hors périmètre agent) — go/no-go en attente.

## Hors périmètre (ne pas traiter, consigner ici)

- Variante « part des dégâts − part des morts » intra-escouade (angle
  complémentaire acté mais NON retenu pour ce lot).
- Toute retouche des graphes FDA gap existants au-delà de la délégation 3.1.

## Découvertes en cours d'exécution

(noter ici, ne pas traiter)

- [P1, 2026-07-23] `tsr` CLI absent du worktree (`node_modules/.bin/tsr`
  introuvable ; pas dans les scripts `package.json`). Le routeTree est
  normalement régénéré par le plugin vite `TanStackRouterVite` (dev/build).
  Contournement retenu pour le gate tsc hors dev-server : script node éphémère
  utilisant l'API programmatique `@tanstack/router-generator` (`Generator` +
  `getConfig`, mêmes options que le plugin : target react, autoCodeSplitting,
  routeFileIgnorePattern `\.test\.tsx?$`). Script supprimé après usage ; diff
  routeTree purement additif. À reproduire pour les prochaines routes du chantier.
- [P3, 2026-07-23] Masquage « Balance des dégâts » : décision de gater par la
  capability `damage_taken` (`useProvidesDamageTaken`, source unique déjà en place
  dans `lib/damage/effectiveHp.ts`) plutôt que sonder la présence de `damage_taken`
  dans les rows. Motif : title-agnostic (CLAUDE.md — brancher sur capability, jamais
  slug==), aligné avec le masquage des autres surfaces combat (résistance), et H5
  ne déclare pas la capability → masquage propre. Le report D5 par-point
  (`cumulativeSigned`) couvre les matchs Infinite isolés sans `damage_taken`.
- [P4, 2026-07-23] Régénération OpenAPI : le repo n'a PAS de dump auto de
  `openapi.yaml` — le bloc `components.schemas` est MANUEL, réconcilié aux schémas
  auto-dérivés Huma par `TestOpenAPISchemaDrift` (gate MISSING=0, DIVERGENT logué).
  Les 2 champs ont donc été ajoutés à la main dans `openapi.yaml` (+ interfaces
  miroir manuelles `types.ts`), puis `npm run generate-types`. Mode emit dispo si
  besoin : `OPENAPI_EMIT_DIVERGENT_OUT=... CGO_ENABLED=1 go test ./internal/api/ -run TestOpenAPISchemaDrift`.
- [P4, 2026-07-23] Résidu d'engagement par surface : Timeseries/Escouade exposent
  les paces (résidu = observé − attendu) ; Session expose `engagement_score` de
  `match_series` qui EST DÉJÀ le résidu évén./min (cf. doc SessionEngagementChart).
  Helper `engagementGapEvents` générique (résidu, durée) ; chaque surface calcule
  son résidu. Charts Timeseries/Squad réutilisent la query engagement existante
  (dédup cache TanStack Query, zéro fetch supplémentaire).
- [P3, 2026-07-23] `computeCompareScale` prend désormais un 5e paramètre `hp` (le
  domaine partagé de la balance cumulée est en vies = dégâts nets ÷ barème). Seul
  caller = `SessionDetailPage` (hp via `useEffectiveHpToKill`). Aucun autre appelant.
- [P2, 2026-07-23] `efficiencySeries.description` (i18n squad) est mort AVANT ce
  chantier (aucun caller — grep vide) ; conservé tel quel, hors périmètre P2
  (nettoyage non traité). Son texte mentionnait « trait plein / pointillé » qui ne
  correspond plus aux 2 cartes séparées — à réévaluer si un jour rebranché.

## Livraison

- Chaque phase = 1+ commits sur `feat/squad-dynamique` (demander avant commit).
- Avant la clôture : gates complets (`make check-types`, `make test-web`,
  `go test ./...` apps/go-api, lint si Go touché) + entrée `thought_log.md`.
- Merge dans `main` = déploiement prod automatique : prévenir l'utilisateur.

## Estimation

| Phase | Effort |
|---|---|
| P1 onglet | moyen |
| P2 séparation | rapide |
| P3 solde de vies | moyen |
| P4 engagement cumulé | lourd (seule phase Go) |
| P5 aire | rapide |
