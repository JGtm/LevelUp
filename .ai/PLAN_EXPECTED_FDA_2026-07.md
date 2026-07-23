# PLAN — Différentiel FDA réel vs attendu (Timeseries, Sessions, Escouade/Synergies)

> Créé le 2026-07-23. Branche : `feat/expected-fda-differential` (worktree
> `.claude/worktrees/expected-fda`, base `main` 1b5944ed6). Exécution sous contrat
> `plan-execution` (skill). Effort estimé : moyen (~1,5-2 j).
> Supervision : agent principal ; implémentation : agents Opus par lot.

## Objectif et critère de succès

Exposer l'écart entre le FDA réel et le FDA attendu sur 3 surfaces :
1. Page **Timeseries** (onglet Résumé) : différentiel lissé par match, aire signée.
2. Page **Sessions** (détail) : différentiel **cumulé** sur la session, aire signée.
3. Page **Escouade / onglet Synergies** : différentiel **cumulé** par joueur (1 courbe
   par membre).

Succès = les 3 charts affichés avec données réelles sur un profil Halo Infinite,
masqués proprement sur Halo 5 (capability), gates complètes vertes (lot D).

## Données validées en amont (2026-07-23, parquets staging du jour)

- `match_participants.kills_expected/deaths_expected` (Infinite) : couverture ~98 %
  (514/524 Chocoboflor, 1040/1060 JGtm, 1181/1212 Madina, 32/32 Daemon).
- **6 lignes avec valeurs infinies** (3 JGtm, 3 Madina) → garde `isfinite` OBLIGATOIRE.
- Bruit mesuré : écart moyen kills réel−attendu < 1 ; dispersion par match ±3-5 ;
  p10..p90 ≈ −4..+7 → le cumulé/lissé est la bonne forme, pas le brut.
- Modèle assists personnel (`player_assists_model`, per-mode) : Madina 14 modes,
  JGtm 7, Chocoboflor 4, **Daemon 0** (32 matchs < seuil) → fallback populationnel
  (`assists_model_coefs`, metadata) requis, déjà codé pour is_me.
- **Halo 5 : 0 `kills_expected` en shared** (0/5590 sur les 4 joueurs), 0 modèle
  assists → feature ABSENTE sur H5 (capability), le modèle local `localExpectedKD`
  (affichage vue match) n'est PAS persisté et reste hors périmètre.

## Décisions produit (TRANCHÉES — ne pas rouvrir en cours d'exécution)

- **D1** Timeseries = chart « Écart au FDA attendu » dans l'onglet Résumé, sous
  `TimeseriesKdaValueTrend` : différentiel (FDA réel − FDA attendu) par match, MÊME
  lissage que `TimeseriesKdaValueTrend` (reprendre sa fenêtre/mécanique à l'identique),
  aire ancrée à 0 avec dégradé divergent à bascule sur zéro (pattern
  `SessionNetScoreArea`, PAS de visualMap). Tooltip : réel, attendu, écart.
- **D2** Sessions = chart « Écart cumulé au FDA attendu » dans `SessionChartStack`,
  à côté de `SessionNetScoreArea` : somme cumulée du différentiel sur les matchs de
  la session, même pattern visuel (aire signée, markLine 0).
- **D3** Synergies = chart « Écart cumulé au FDA attendu » : 1 courbe par joueur
  (main + coéquipiers confirmés), cumul par `match_order`, couleurs
  `getSquadPlayerColors` existantes, pastille KPI par joueur = écart moyen par match
  (ex. « +0,7/match »). Pas d'aire (multi-séries), lignes + markLine 0.
- **D4** Multi-titre : nouvelle capability produit `expected_stats`
  (`internal/domain/title/registry.go`, pattern `CapTeamMMR`) — Infinite = oui,
  H5 = non ; miroir front `TITLE_CAPABILITIES` + `useCapability('expected_stats')`
  masque les 3 charts (pattern `team_mmr` dans `TimeseriesPage.summary.tsx:63` et
  `SessionChartStack.tsx:64`). Jamais de `slug ==`. Côté Go, pas
  d'`ErrCapabilityNotSupported` nécessaire : champs additifs optionnels, nil si
  absents (dégradation par construction).
- **D5** Matchs sans attendu (NULL, non-fini, ~2 %) : différentiel nul → trou dans
  le chart (pas de zéro) ; le CUMUL saute ces matchs (reporte la dernière valeur).
- **D6** FDA réel per-match = valeur native (`StatsMatchRow.KDA`, ADR 0006 — jamais
  recalculé). FDA attendu = `kills_expected + assists_expected/3 − deaths_expected`
  via helper canonique UNIQUE `internal/analysis` (voir A1).
- **D7** Libellés (FR sans anglicismes, EN en parité) : FR « Écart au FDA attendu » /
  « Écart cumulé au FDA attendu » ; EN « KDA gap to expected » / « Cumulative KDA gap
  to expected ». Le nom de la métrique réutilise `fieldMappings.fields['kda'].label`
  quand il apparaît seul.
- **D8** Assists attendus par membre d'escouade : modèle personnel du membre
  (sa player DB) → fallback populationnel → nil (même chaîne que
  `computeExpectedAssists`). Échec de lecture d'un modèle = log `slog.WarnContext`
  puis fallback, jamais silencieux.

## Lots (ordre strict, contrat plan-execution)

### Lot A — Backend (Go) : helper, capability, extensions de structs, projections

- [x] **A1** Helper canonique `internal/analysis/expected_fda.go` :
  `ExpectedFDA(killsExp, deathsExp, assistsExp *float64) *float64` — nil si
  kills/deaths manquants ou non finis (garde `IsBadFloat` = NaN/±Inf) ; assists nil
  = terme 0 (attendu partiel K/D, documenté) ; + `FDADiff(actualKDA, expectedFDA)`.
  Tests purs (nil, ±Inf, NaN, nominal, écart négatif) verts.
- [x] **A2** Garde-rail `internal/archlint/no_inline_expected_fda_test.go` :
  flague toute ligne Go non-test HORS `internal/analysis/` combinant un champ
  `(Kills|Deaths|Assists)Expected` ET une division `/3`. Allowlist vide (datée
  2026-07-23) — la formule vit uniquement dans `analysis` (exclu du walk).
  `TestNoInlineExpectedFDA` + test de discrimination de la regex verts.
- [x] **A3** Capability `CapExpectedStats = "expected_stats"` (`registry.go`, pattern
  `CapTeamMMR`), ajoutée aux capabilities halo_infinite + `knownCapabilities`
  (`config_loader.go`) ; miroir front `capabilities.ts` (`TITLE_CAPABILITIES`).
  Test registre Go + test vitest miroir (gate présent/absent) verts.
- [x] **A4** `TimeseriesMatchRow` +4 champs (omitempty) ; projection dans
  `buildMatchRows` via `analysis.ExpectedFDA`. AssistsExpected = batch
  `computeExpectedAssistsBatch` (per-mode, une résolution/mode) câblé dans
  `TimeseriesService.GetPage` via `WithExpectedAssists` (DI `registry_pages.go`,
  gatée capability). Test service (projection + null-safety) vert.
  **Écart plan** : la page charge le CANONICAL (`StatsMatchRowsFromCanonical`), pas
  Q23 → source de l'attendu = `MatchParticipant.KillsExpected/DeathsExpected`
  (nouveaux champs canonical + query/scan `PlayerMatchesRepo`), voir Découvertes.
- [x] **A5** `SessionDetailMatchRow` +4 champs ; projection dans
  `buildSessionDetailRows` (param `assistsExpected`) via A1 + batch A4, câblé dans
  `SessionPageService.GetPage`. Rows compare/best-worst : assists nil (chart cumulé =
  session principale, D2). Test service (projection + null-safety) vert.
- [x] **A6** `SquadPerformanceSeriesPoint` +4 champs ; `buildSquadPerformanceSeries`
  lit `r.Self.KillsExpected/DeathsExpected` (champs ajoutés à `MatchParticipant`),
  assists attendus PAR MEMBRE via 2 méthodes port ajoutées à `SquadV2Loader`
  (`LoadPlayerAssistsModel` + `LoadPopulationalAssistsCoef`, impl `SquadV2LoaderAdapter`)
  + résolveur cache par membre+mode (chaîne D8, échecs `slog.WarnContext`). Gaté
  capability. Test service teammates (mock) vert.
- [x] **A7** `openapi.yaml` : 3 schémas étendus (4 champs chacun, style manuel
  omitempty = sans nullable) ; `npm ci` (worktree) puis `generate-types` → 12
  occurrences dans `generated.ts` ; drift test `TestOpenAPISchemaDrift` vert.
- [x] **Gate A** : `go test ./internal/analysis/... ./internal/service/...
  ./internal/archlint/... ./internal/domain/...` → tous `ok` (exit 0) ;
  `go vet ./...` → exit 0 ; `generated.ts` régénéré (exit 0). CGO msys64.
  (Bonus hors périmètre Gate A vérifiés : `platform/duckdb` tests + drift OpenAPI verts.)

### Lot B — Front : Timeseries + Sessions

- [x] **B1** Timeseries (D1) : nouveau composant `TimeseriesFdaGapTrend`
  (`features/timeseries/TimeseriesFdaGapTrend.tsx`) — aire divergente du
  différentiel BRUT `kda − kda_expected` (trous D5 visibles, `connectNulls:false`)
  + ligne lissée rolling-5 (mécanique répliquée de `TimeseriesKdaValueTrend`).
  Monté dans `TimeseriesPage.summary.tsx` sous la tendance FDA valeur. Gate =
  self-gate `useCapability('expected_stats')` DANS le composant (retour null) —
  décision d'implémentation (latitude du plan) pour tester « non rendu » en
  isolation et garder DRY la double-disposition de SessionChartStack. Tooltip
  réel/attendu/écart, « — » si attendu absent (D5).
- [x] **B2** Sessions (D2) : nouveau composant `SessionFdaGapCumulative`
  (`features/session-detail/SessionFdaGapCumulative.tsx`) — cumul du différentiel
  (`computeCumulativeFdaGap`, tri chronologique, report D5 : match sans attendu
  sauté, la courbe reporte la dernière valeur), aire signée divergente + markLine 0.
  Monté via `SessionChartStack.tsx` juste après `SessionNetScoreArea` (dispositions
  dense ET normale), self-gate capability.
- [x] **B3** i18n : 4 clés `timeseries.summary.fda_gap_*` + 5 clés
  `session.detail.fda_gap_*`/`chart_fda_gap_title` (FR+EN, sans anglicismes) dans
  les manifests TOML, régénérés (`build_i18n_manifests.mjs`). Zéro hex/classe
  Tailwind : dégradé via helper `divergentZeroGradient` (tokens `divergent-*`).
- [x] **B4** Tests : `TimeseriesFdaGapTrend.test.tsx` (builder nominal + trous D5 +
  gate présent/absent), `SessionFdaGapCumulative.test.tsx` (cumul + report D5 + tri
  + builder + gate), `divergentZeroGradient.test.ts` (math bascule) +
  `divergentZeroGradient.guard.test.ts` (garde-rail règle 6). echarts-for-react
  mocké (pattern `ChartFromOption.test`).
- [x] **Gate B** : `npx tsc -b --force` → exit 0 ; `npm run test -- run` (8 fichiers,
  54 tests, incl. suites touchées SessionNetScoreArea/SessionDetailPage/capabilities)
  → exit 0. Vitest hors sandbox (limitation connue).

### Lot C — Front : Escouade / Synergies

- [x] **C1** Builder `buildFdaGapCumulativeOption` (cumul par `match_order`, D5,
  1 série line par joueur, `colorByPlayer`, markLine 0, pas d'aire). **Écart plan
  (latitude accordée)** : `squadPerformanceLineCharts.ts` faisait déjà 511 L (> 500,
  seuil dépassé AVANT mon ajout) → builder extrait dans un fichier frère
  `features/squad/charts/squadFdaGapChart.ts` (helpers partagés `orderedPlayers`/
  `maxLength`/`xAxisLabels` + `CommonOpts` exportés du fichier voisin, DRY, pas de
  duplication). Consomme `kda_expected` (projeté backend, D6), pas de recalcul. Pas
  d'extension `extractValue`/`PerformanceMetricKey` (cumul ≠ valeur per-match simple)
  → helpers dédiés `cumulativeFdaGapSeries` + `meanFdaGapPerMatch` (KPI D3).
- [x] **C2** Composant self-gated `SquadFdaGapCumulativeCard.tsx`
  (`useCapability('expected_stats')` → null, pattern Lot B) monté sur
  `SquadSynergiesPage.tsx` APRÈS `SquadFragSection`, AVANT la section Impact
  (regroupe les visuels de performance par joueur). Couleurs `getSquadPlayerColors`
  (`playerColors` L65) + ordre `playerOrder` (L66-68) déjà calculés par la page.
  Pastilles KPI en FOOTER de la ChartCard (slot `children`) : 1 pilule/joueur (point
  coloré token-dérivé + gamertag + écart moyen signé « +0,7/match » via
  `Intl.NumberFormat(intlLocale, signDisplay:'always')` + suffixe i18n `units.perGame`
  ; « — » si aucun match avec attendu).
- [x] **C3** i18n squad FR+EN (`features/squad/i18n.ts`) : section `fdaGap`
  `{ title, averageCaption }` (parité `Record<SquadLocale, T>`). FR « Écart cumulé au
  FDA attendu » / « Écart moyen par match » (D7, sans anglicismes) ; EN « Cumulative
  KDA gap to expected » / « Average gap per match ». Suffixe pastille réutilise
  `units.perGame` existant (pas de nouvelle clé).
- [x] **C4** Tests vitest : `squadFdaGapChart.test.ts` (13 tests — cumul, report D5,
  non-fini, `match_order` désordonné, trou d'intersection, moyenne, builder
  multi-joueurs/couleurs/markLine/hidden) + `SquadFdaGapCumulativeCard.test.tsx`
  (3 tests — capability présente/absente, pastilles signées + suffixe i18n + « — »,
  pattern `SessionFdaGapCumulative.test.tsx`). `SquadSynergiesPage.test.tsx` existant
  reste vert (intégration page, pageData null → carte gate fail-open sans throw).
- [x] **Gate C** : depuis `apps/web` (worktree) — `npx tsc -b --force` (après purge
  `node_modules/.tmp`) → exit 0 ; `npm run test -- run squadFdaGapChart
  SquadFdaGapCumulativeCard squadPerformanceLineCharts SquadSynergiesPage` →
  4 fichiers / 28 tests passés, exit 0 (vitest hors sandbox, limitation connue).

### Lot D — Gates finales, docs, clôture

- [ ] **D1** Suite complète : `go test ./...` + `go vet ./...` (apps/go-api, CGO) ;
  purge `node_modules/.tmp` puis `npm run typecheck`, `npm run lint`, `npm run test`
  (apps/web). Baseline lint non aggravée (`make go-api-lint` ratchet).
- [ ] **D2** Skill `delivery-checklist` passé intégralement (complétude, logging,
  multi-titre, couleurs, i18n).
- [ ] **D3** Plan statué (toutes cases), section Découvertes remplie, entrée
  `.ai/thought_log.md`.
- [ ] **D4** Point final utilisateur : résumé, risque de merge D8/plan revue
  analytique, proposition de vérification visuelle (serveur dev) avant merge.

## Risques et coordination

- **Chevauchement** avec `.ai/PLAN_REVUE_ANALYTIQUE_TIMESERIES_SQUAD_2026-07.md`
  (non commencé) : collisions attendues sur `TimeseriesMatchRow`/`buildMatchRows`,
  `TimeseriesPage.summary.tsx`, `SquadSynergiesPage.tsx`, `openapi.yaml`/`generated.ts`.
  Décision : livrer CE chantier d'abord (petit, additif) ; l'autre rebasera.
- Worktree sans `data/` : gates = tests uniquement ; vérification visuelle après
  merge dans le checkout principal (proposée au user, pas exécutée d'office).
- `npm ci` requis dans le worktree avant generate-types/typecheck (one-shot, long).

## Découvertes (hors périmètre — ne pas traiter)

- 6 lignes `kills_expected = +Inf` en base Infinite (3 JGtm, 3 Madina) : bug amont
  probable du calcul local ou de l'API — à investiguer un jour (nettoyage données).
- `capabilities.toml` fin : clé `analytics.timeseries` marquée `not_exposed` sur les
  deux titres alors que la page est servie (chemin legacy) — incohérence doc à
  revoir dans un chantier capabilities.
- **Réf A6 imprécise dans le plan** : `r.Self.KillsExpected/DeathsExpected
  (canonical/match.go:286-287)` pointait en fait `SkillSnapshot.KillsExpected/
  DeathsExpected`. Vérif sur pièces (2026-07-23) : ces champs `SkillSnapshot`
  ne sont JAMAIS peuplés par le loader `PlayerMatchesRepo` (`projectSkillSnapshot`
  ne lit pas les colonnes), et `MatchParticipant` (= `r.Self`) n'avait pas ces
  champs. Décision : champs ajoutés à `MatchParticipant` + peuplés depuis
  `match_participants.kills_expected/deaths_expected` (query/scan `PlayerMatchesRepo`
  étendus, +2 colonnes). `StatsMatchRowsFromCanonical` lit désormais `r.Self`
  (plus `SkillSnapshot`). **Conséquence hors périmètre à noter** :
  `SkillSnapshot.KillsExpected/DeathsExpected` (canonical/match.go) ne sont plus lus
  nulle part en prod (uniquement en tests) → code potentiellement mort à retirer dans
  un futur nettoyage canonical (NON traité ici).
- `StatsMatchRow` (path canonical Timeseries/Sessions) n'exposait pas `shots_hit` :
  champ `ShotsHit` ajouté (mappé depuis `Self.ShotsHit`) pour le fallback populationnel
  des assists attendus — sinon la chaîne D8 dégradait toujours en modèle personnel/nil.
- `openapi.yaml` manuel diverge de Huma sur `nullable` des pointeurs `omitempty`
  (préexistant, non gaté par `TestOpenAPISchemaDrift` qui ne gate que le MISSING).
  Nouveaux champs alignés sur la convention MANUELLE (sans `nullable`, comme `kda`).
- **Lot B — 2 corrections IN-PÉRIMÈTRE (conséquences directes, pas opportunistes)** :
  1. **Bloqueur tsc** : ajouter `expected_stats` à `TITLE_CAPABILITIES` (Lot A) a
     rendu `FeatureUnavailable.tsx` (`Record<TitleCapability, {fr,en}>`) incomplet →
     `tsc` échouait (TS2741). Entrée `expected_stats` ajoutée (FR/EN). Gate A était
     Go-only, d'où le non-détecté en A3.
  2. **Centralisation règle 6** : le dégradé divergent-ancré-à-0 existait DÉJÀ en 2
     copies (`SessionNetScoreArea`, `features/palmares/CumulativeFragGapChart`). Les 2
     charts B en faisaient 4 → extraction du helper canonique
     `lib/charts/divergentZeroGradient.ts` + migration des 3 copies existantes +
     garde-rail grep `divergentZeroGradient.guard.test.ts` (identifiant `zeroRatio`
     verrouillé hors du helper). CumulativeFragGapChart (palmares) migré par
     nécessité (anti-pattern 8 : pas de factorisation abandonnée) ; sortie visuelle
     inchangée (même structure de dégradé).

## Journal d'exécution

- 2026-07-23 : plan créé, revu (grille plan-review), données validées sur parquets
  du jour. Aucun lot exécuté à ce stade.
- 2026-07-23 : **Lot A exécuté et clôturé** (agent Opus, worktree `expected-fda`,
  aucun commit — revue superviseur). A1→A7 tous `[x]` (vérifiés). Décisions
  d'implémentation notables :
  - Source unifiée de l'attendu K/D : les 3 surfaces (Timeseries/Sessions via
    `StatsMatchRowsFromCanonical`, Escouade via `SquadV2Loader.LoadFor`) chargent le
    canonical `PlayerMatchRow` par `PlayerMatchesRepo`. Champs `KillsExpected/
    DeathsExpected` ajoutés à `canonical.MatchParticipant`, peuplés par la query/scan
    shared étendus (+`p.kills_expected`, `p.deaths_expected`). `StatsMatchRow` +
    `ShotsHit` (fallback populationnel assists).
  - Assists attendus : arithmétique OLS/populationnelle centralisée
    (`analysis.ApplyPersonalAssistsModel` / `ApplyPopulationalAssists`, source unique
    partagée service ↔ teammates) ; `computeExpectedAssists` (Match View) refactoré
    dessus ; batch per-mode côté service ; par-membre côté teammates (2 méthodes port).
  - DI : `WithExpectedAssists` sur Timeseries + Session services, gaté
    `CapExpectedStats` (jamais slug==) dans `registry_pages.go` → zéro accès DB / bruit
    de logs sur Halo 5.
  - Gate A verte : `go test` (analysis/service/archlint/domain) exit 0 ; `go vet ./...`
    exit 0 ; `generate-types` OK (12 nouveaux champs dans `generated.ts`). Bonus verts :
    `platform/duckdb` (query/scan), `TestOpenAPISchemaDrift`, vitest miroir capabilities.
  - Prochaine étape : Lot B (front Timeseries + Sessions).
- 2026-07-23 : **Lot B exécuté et clôturé** (agent Opus, worktree `expected-fda`,
  aucun commit — revue superviseur). B1→B4 + Gate B tous `[x]`. Décisions notables :
  - **Gate = self-gate** `useCapability('expected_stats')` dans chaque composant
    (retour null) plutôt que booléen parent : teste « non rendu » en isolation et
    garde DRY la double-disposition (dense/normale) de SessionChartStack. Honore D4.
  - **B1 forme** : aire divergente du différentiel BRUT par match (trous D5 visibles,
    `connectNulls:false`) + ligne lissée rolling-5 (réplique la mécanique de
    `TimeseriesKdaValueTrend`, 2 séries comme le modèle). Tooltip réel/attendu/écart
    (« — » si attendu absent).
  - **B2 forme** : `computeCumulativeFdaGap` pur (tri chronologique + report D5) →
    aire signée divergente ancrée à 0 (pas de coloration outcome, sémantique = signe).
  - **Centralisation (règle 6)** : helper `lib/charts/divergentZeroGradient.ts` +
    migration des 3 copies existantes (SessionNetScoreArea, CumulativeFragGapChart
    palmares, + les 2 nouveaux charts) + garde-rail grep. Bloqueur tsc corrigé
    (`FeatureUnavailable.tsx` : entrée `expected_stats`). Cf. Découvertes.
  - Gate B verte : `tsc -b --force` exit 0 ; `vitest run` 8 fichiers / 54 tests exit 0
    (incl. suites touchées). Aucun `go` touché (Lot B = front pur).
  - Prochaine étape : Lot C (Escouade / Synergies), puis Lot D (gates finales, revue).
- 2026-07-23 : **Lot C exécuté et clôturé** (agent Opus, worktree `expected-fda`,
  aucun commit — revue superviseur). C1→C4 + Gate C tous `[x]`. Décisions notables :
  - **Extraction fichier frère** : `squadPerformanceLineCharts.ts` était déjà à 511 L
    (> seuil 500 AVANT l'ajout) → builder `buildFdaGapCumulativeOption` placé dans
    `features/squad/charts/squadFdaGapChart.ts` (ne pas accroître la dette god-file,
    latitude accordée par le pilote). Helpers `orderedPlayers`/`maxLength`/`xAxisLabels`
    + type `CommonOpts` EXPORTÉS du fichier voisin et réutilisés (DRY, zéro duplication).
  - **Forme C1** : cumul par `match_order` croissant du différentiel `kda − kda_expected`
    (attendu projeté backend, D6, jamais recalculé). D5 : match sans attendu → cumul
    reporte la dernière valeur (le point figure quand même) ; trous d'intersection = null
    (pontés `connectNulls`). 1 line/joueur, couleur `colorByPlayer`, markLine 0 sur le
    1er joueur, PAS d'aire (multi-séries). Robuste au désordre (`sort` par match_order)
    et au non-fini (`Number.isFinite`).
  - **Montage C2** : composant self-gated `SquadFdaGapCumulativeCard.tsx`
    (`useCapability('expected_stats')` → null, honore D4, pattern Lot B) monté sur
    `SquadSynergiesPage.tsx` après `SquadFragSection`, avant Impact. Pastilles KPI en
    footer ChartCard : écart moyen/match par joueur (matchs AVEC attendu uniquement, D3),
    signé, couleur token-dérivée du joueur, suffixe i18n `units.perGame`, « — » si vide.
  - **i18n C3** : section `fdaGap` FR+EN (parité), D7 « Écart cumulé au FDA attendu ».
  - Gate C verte : `tsc -b --force` exit 0 ; `vitest run` (squadFdaGapChart +
    SquadFdaGapCumulativeCard + squadPerformanceLineCharts + SquadSynergiesPage) →
    4 fichiers / 28 tests exit 0. Aucun `go` touché (Lot C = front pur).
  - Prochaine étape : Lot D (gates finales complètes, delivery-checklist, revue merge).
