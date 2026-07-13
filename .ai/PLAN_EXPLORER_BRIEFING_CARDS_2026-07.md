# PLAN — Cards de synthèse au-dessus du tableau Explorer (mode Matchs) (2026-07)

> Rédigé le 2026-07-11 après cartographie sur pièces (front + backend, 2 agents Explore +
> vérification manuelle des ancrages). Destiné à être exécuté par un agent (Opus).
> **Exécution sous contrat du skill `plan-execution`** (ordre strict, périmètre fermé,
> statuts obligatoires, zéro fix hors périmètre). Revue faite avec le skill `plan-review`.
> Doctrine RE-VÉRIFIER : toutes les références fichier:ligne de ce plan ont été vérifiées
> le 2026-07-11 mais DOIVENT être re-vérifiées sur pièces au moment de coder.

## Objectif et critère de succès

Le mode Matchs de la page Explorer affiche aujourd'hui un tableau brut précédé d'un seul
compteur (« N matchs trouvés »). Objectif : un **bandeau de briefing** au-dessus du
tableau qui aide à LIRE le résultat de la recherche — KPI agrégés du sous-ensemble
filtré, comparaison à la baseline personnelle, frise des résultats, et modules
conditionnels (notes par carte/mode/playlist, tendance, classé) qui s'activent selon les
filtres, les capabilities du titre et la taille d'échantillon.

**Critère de succès global** : sur le profil de test (JGtm), pour les 6 scénarios de
recherche du gate D2, le bandeau affiche des valeurs exactes (recoupées à la main sur 1
scénario), aucun libellé anglais sous locale FR, aucun GUID/clé brute, aucun module
affiché sous son seuil d'échantillon, et le module classé est absent sous Halo 5.
`make check-types`, `make test-web`, `cd apps/go-api && go test ./...`,
`make go-api-lint` verts.

**Ce que ce bandeau N'EST PAS** : une deuxième page Synthèse. C'est une lecture compacte
du résultat de recherche. Tout ce qui ressemble à un dashboard complet est hors périmètre.

**Branche cible** : `feat/explorer-briefing-cards`, créée depuis `main`.
Rappel : push sur `main` = déploiement prod automatique — merge uniquement avec accord
utilisateur explicite.

**Effort estimé** : Lot A moyen (~1 j), Lot B rapide (~0,5 j), Lot C moyen (~0,5-1 j),
Lot D rapide. Aucun blocker externe identifié. Chantiers parallèles en attente
(`PLAN_H5_MATCHVIEW_RESIDUS`, `PLAN_ASCENSION_UX`) : fichiers disjoints a priori, mais si
l'un d'eux est en cours sur `match_history_service*.go`, se coordonner avant de démarrer.

## Décisions produit (TRANCHÉES — modifier ici avant exécution si désaccord)

- **DEC-1 Agrégats 100 % côté serveur.** Aucun agrégat calculé côté client sur
  `table.items`. Raisons : (a) le KDA agrégat ADR 0006 n'est PAS le quotient — tout
  recalcul client divergera ; (b) le client ne reçoit qu'au plus `page_size` lignes
  (10 000 aujourd'hui, `ExplorerPage.tsx:~199`) — au-delà, un agrégat client est
  silencieusement faux ; (c) une seule source de vérité testable.
- **DEC-2 La « note » par carte/mode/playlist = perf score moyen du groupe, PAS une
  nouvelle stat.** Moyenne des `performance_score` per-match du groupe (0-100), servie
  avec le palier via `analysis.PerfTier` (`internal/analysis/indicators.go:113`) — les
  MÊMES paliers 1..5 que le filtre « Palier de performance » du tableau (libellés front
  existants réutilisés). Rationale : le perf score est déjà un percentile pondéré vs
  l'historique personnel (`performance_score.go`), donc une « note personnelle » par
  construction, comparable entre cartes et normalisée par minute. Créer une stat
  composite nouvelle = redondance + inexplicabilité + maintenance (violerait la règle
  n°14 et l'esprit ADR 0006). La note est toujours accompagnée du nombre de matchs du
  groupe et du delta winrate vs baseline (c'est le delta qui porte la narration).
- **DEC-3 Baseline = historique complet du joueur** (toutes les canonical rows du titre,
  post-exclusions manuelles, AVANT filtres de recherche). Les deltas (winrate, KDA,
  perf) sont calculés côté serveur et servis prêts à afficher. Zéro I/O supplémentaire :
  `loadBriefingKPIs` charge DÉJÀ l'historique canonical complet
  (`match_history_service.go:303-305`) — on le réutilise.
- **DEC-4 Seuils nommés (constantes Go, pas de magic numbers)** :
  - `MinBriefingModulesMatches = 10` : en dessous, le bandeau sert le socle (KPIs +
    frise) avec `low_sample: true` ; les modules baseline/dimensions/tendance/classé
    sont omis (nil). Aligné sur `analysis.MinMatchesForRelative`.
  - `MinDimensionGroupMatches = 10` : un groupe (carte/mode/playlist) sous ce seuil
    n'a pas de note (il peut apparaître dans top/flop uniquement s'il est qualifié).
  - Tendance : servie si `len(filtered) >= 20` ET étendue temporelle ≥ 14 jours.
  - Frise : `MaxOutcomeSequencePoints = 60` derniers matchs du sous-ensemble, ordre
    chronologique ascendant, servie par le SERVEUR (pas dérivée des items client :
    l'ordre de tri utilisateur et le cap `page_size` rendraient la dérivation fausse).
- **DEC-5 Mini-graphes v1 = frise `OutcomeSequenceTape` + sparkline de tendance.**
  Réutiliser les wrappers ECharts existants (`components/charts/`) — pas de nouveau
  wrapper sauf impossibilité démontrée (si nouveau : README charts + tokens sémantiques
  obligatoires). Le pattern visuel de référence est la carte de rivalité
  (`features/palmares/RelationsRivalryCards.tsx` : frise 64px + mini-chart 120px +
  KPI texte). Pas d'autre graphe en v1 (anti-dashboard).
- **DEC-6 Contrat API : champ `briefing` optionnel + opt-in requête.**
  `ExplorerMatchesQueryRequest.include_briefing: bool` → propagé en
  `MatchHistoryQueryRequest.IncludeExplorerBriefing`. `false` (défaut) = réponse
  strictement identique à aujourd'hui (la page Historique ne paie rien). Ce n'est PAS
  un feature flag OFF (règle n°11) : l'Explorer l'envoie à `true` dès le lot B, dans le
  même chantier.
- **DEC-7 Multi-titre par capabilities, jamais par slug.** Module classé (delta CSR,
  attendu vs réel) : gate backend via la capability `match.skill.snapshot` (même
  mécanisme que `titleSupportsLiveCSR`, `internal/api/wire/registry_pages_explorer.go:98-101`),
  gate front via `useCapability('ranked')` (miroir du filtre skill tiers,
  `ExplorerPage.matchesMode.tsx:~279`). Axe OC/DR : déjà neutralisé par
  `ComputeKPIStats` quand `damage_taken` absent (Halo 5) — ne rien réinventer, vérifier
  que le front tolère les champs nil. Titre sans capability = module absent de la
  réponse (pas de N/A affiché, pas de panic, pas d'`ErrCapabilityNotSupported` à
  propager : dégradation = omission).
- **DEC-8 Modules « dimensions » servis uniquement si la dimension est libre** : le
  serveur n'émet une dimension (carte, mode, playlist) que si le sous-ensemble filtré
  contient ≥ 2 valeurs distinctes pour elle. Une recherche déjà réduite à une seule
  carte n'affiche pas de « note par carte » (elle dégénérerait en la note globale).
  Par dimension émise : top 3 + flop 3 des groupes qualifiés (tri par delta winrate).

## Cartographie vérifiée (ancrages, 2026-07-11)

**Backend** :
- Handler : `apps/go-api/internal/api/handlers/explorer.go:106-177`
  (`handleQueryMatches`). Délègue à `MatchHistoryService.GetPage` puis projette. La
  projection actuelle **jette** `mhResp.BriefingKPIs` (lignes 157-175) alors qu'il est
  déjà calculé sur le sous-ensemble filtré exact.
- Service : `apps/go-api/internal/service/match_history_service.go:182-293` (`GetPage`) —
  `filtered` finalisé ligne 236 (`totalScoped := len(filtered)` l.238) ;
  `loadBriefingKPIs(ctx, filtered)` l.265 → `analysis.ComputeKPIStats` l.323, via
  `playerMatchesRepo.LoadPlayerMatches` (port, canonical rows, historique complet).
- Filtres purs : `match_history_service_filters.go` (Go pur sur rows en mémoire).
- DTO existant : `domain.KPIStats` (`internal/domain/squad_v2.go:122-158`) — matchs,
  K/D/A per game+minute, accuracy, avg life, `RankDelta` (somme des deltas CSR/LUSR du
  scope — le cœur du module classé existe déjà), `PerformanceScore` (moyenne 0-100),
  OC/DR (nil-safe), Outcomes W/L/T/DNF.
- Algos réutilisables : `analysis.ComputeKPIStats` (kpi_stats.go) ;
  `breakdown.ByMap/ByMode/ByPlaylist` + `breakdown.CompareToHistorical`
  (`analysis/breakdown/compare.go:25`, `MapDelta` avec `WinRateDelta` +
  `AvgPerformanceScoreDelta`) ; `analysis.PerfTier` (indicators.go:113) ; binning
  serveur `analysis/temporal` (`ResolveAdaptive`, `BucketByGranularity`, ADR 0010) ;
  KDA/WinRate agrégés canoniques ADR 0006 dans `analysis/indicators.go` (vérifier les
  noms exacts sur pièces — skill `go-features`).
- La page Synthèse (`internal/service/synthesis_service.go:121-237`) convertit déjà des
  canonical rows vers `breakdown.Row` : RÉUTILISER ce convertisseur (le localiser via
  les call-sites de `breakdown.ByMap` ; s'il est privé au service synthesis, l'extraire
  vers un helper partagé — règle n°6, pas de 3e copie).

**Frontend** :
- Point d'insertion : `apps/web/src/features/explorer/ExplorerPage.matchesMode.tsx`,
  fonction `ExplorerMatchesResultsBlock` (~l.368-410) — actuellement compteur + tri +
  export CSV.
- Hook : `features/explorer/queries.ts:14-48` (`useExplorerMatches`, POST
  `/players/{slug}/pages/explorer/matches-query`, `page_size: 10000`).
- Types front : `apps/web/src/lib/api/types.ts:658-755` (`ExplorerMatchRow`,
  `ExplorerMatchesQuerySummary`, `ExplorerMatchesQueryRequest`).
- Briques : `components/cards/KpiCard.tsx` (primitive accent 3px),
  `components/charts/OutcomeSequenceTape` (frise, cf. usage RivalryCard),
  `features/_shared/SessionBriefing/KpiGrid.tsx` (`KpiCell` : label + valeur + sub +
  tendances), `features/explorer/ExplorerEncounterBriefing.tsx` (patron local de rangée
  KPI du mode Joueur), `lib/colors/outcomePalette.ts` (`winRateColor`).
- i18n : manifest `apps/web/src/lib/i18n/manifests/explorer.toml` (fr + en, ICU),
  régénéré par `node apps/web/scripts/build_i18n_manifests.mjs`. JAMAIS d'édition de
  `lib/i18n/generated/explorer.ts` à la main.
- Capabilities front : `useCapability('ranked')`, `useProvidesTeamMmr()` (gates
  existantes dans `ExplorerPage.matchesMode.tsx` et `ExplorerMatchesTable.tsx`).

## LOT A — Backend : bloc `briefing` dans la réponse matches-query

Gate d'entrée : brancher sur `feat/explorer-briefing-cards`, relire ce plan, re-vérifier
les ancrages ci-dessus sur pièces.

- [x] **A1 — DTOs domaine** (couvert par le WIP hérité `c7ccda510`, vérifié sur pièces +
  compile) : nouveau fichier `internal/domain/explorer_briefing.go`
  (domaine pur, pas de logique) :
  - `ExplorerBriefing { KPIs *KPIStats; LowSample bool; PeriodStart/PeriodEnd *time.Time;
    OutcomeSequence []ExplorerBriefingOutcome; Baseline *ExplorerBriefingBaseline;
    Dimensions []ExplorerBriefingDimension; Trend *ExplorerBriefingTrend;
    Ranked *ExplorerBriefingRanked }` — tous les blocs optionnels `omitempty`.
  - `ExplorerBriefingOutcome { MatchID string; OutcomeCode int; StartTime time.Time }`.
  - `ExplorerBriefingBaseline { Matches int; WinRate float64; KDA float64;
    AvgPerf *float64; DeltaWinRate float64; DeltaKDA float64; DeltaPerf *float64 }`
    (unités ADR 0006 : WinRate en 0..1, le front formate).
  - `ExplorerBriefingDimension { Dimension string /* "map"|"mode"|"playlist" */;
    Entries []ExplorerBriefingDimensionEntry }` ;
    `ExplorerBriefingDimensionEntry { Label string; Matches int; WinRate float64;
    DeltaWinRate float64; AvgPerf *float64; NoteTier *int /* 1..5, nil si groupe
    < MinDimensionGroupMatches */ }`.
  - `ExplorerBriefingTrend { Granularity string; Points []ExplorerBriefingTrendPoint }` ;
    point = `{ BucketStart time.Time; Matches int; WinRate float64; AvgPerf *float64 }`.
  - `ExplorerBriefingRanked { RatingKind string; DeltaSum float64;
    ExpectedWinRate *float64; ActualWinRate float64; MatchesWithPrediction int }`.
  - Champ `Briefing *ExplorerBriefing` ajouté à `MatchHistoryPageResponse` ET à
    `ExplorerMatchesQueryResponse` ; champ `IncludeBriefing bool` sur
    `ExplorerMatchesQueryRequest` (json `include_briefing`) et
    `IncludeExplorerBriefing bool` sur `MatchHistoryQueryRequest`.
- [x] **A2 — Refactor du chargement canonical (pré-requis, aucun comportement modifié)** :
  `loadBriefingKPIs` scindé en `loadCanonicalForScope` (charge + filtre par match_id) +
  `kpisFromScoped`. ÉCART assumé : ne retourne QUE `scoped` (pas `all` canonical) — la
  baseline/dimensions/tendance sont bâties sur les raw rows (cf. journal), donc `all`
  canonical serait mort. Best-effort conservé (WarnContext + nil).
  dans `match_history_service.go`, scinder `loadBriefingKPIs` en (a)
  `loadCanonicalForScope(ctx, filtered) (scoped, all []canonical.PlayerMatchRow)`
  (charge une seule fois, filtre par match_id) et (b) le calcul KPI existant. `GetPage`
  appelle (a) une fois ; `BriefingKPIs` reste servi à l'identique (la page Historique ne
  change pas d'un octet de réponse). Best-effort conservé : échec de chargement → log
  `slog.WarnContext` existant + briefing nil, jamais d'erreur 500.
- [x] **A3 — Constructeur du briefing (socle)** : `match_history_service_briefing.go`
  (~430 L), méthode `s.buildExplorerBriefing(filtered, allRaw, scopedKPIs)`. Socle
  (KPIs socle canonical + période + frise 60 chrono asc + LowSample) livré. Constantes
  seuils nommées/commentées. ÉCART de signature assumé (raw rows + scopedKPIs au lieu de
  scoped/all canonical — cf. journal).
  `buildExplorerBriefing(ctx, filtered []domain.MatchHistoryRawRow, scoped, all
  []canonical.PlayerMatchRow, rankedCapable bool) *domain.ExplorerBriefing`, appelé
  depuis `GetPage` UNIQUEMENT si `req.IncludeExplorerBriefing`. Socle :
  - `KPIs` : réutilise le calcul A2 (aucun recalcul).
  - `PeriodStart/End` : min/max start_time du sous-ensemble.
  - `OutcomeSequence` : 60 derniers matchs (DEC-4), tri chrono asc, depuis `filtered`.
  - `LowSample` : `len(filtered) < MinBriefingModulesMatches` → seuls KPIs + frise +
    période sont émis, tous les autres blocs nil.
  - Constantes de seuils déclarées ici, nommées, commentées (DEC-4).
- [x] **A4 — Module baseline** : deltas signés scope − baseline via `aggregateRawStats`
  (wins/total, KDA agrégat `analysis.AggregateKDA` NOUVEAU helper canonique, perf moyenne).
  ÉCART : baseline sur `allRaw` (historique complet post-exclusions = DEC-3) au lieu de
  `all` canonical → mêmes chiffres, évite l'I/O canonical inutile. NB : `Baseline` calculé sur `all` (canonical complet,
  DEC-3) avec les helpers agrégés canoniques existants de `analysis/indicators.go`
  (WinRate, KDA agrégat ADR 0006 — vérifier les noms exacts sur pièces, ne PAS
  recalculer ad hoc) ; deltas = valeur(scoped) − valeur(all). Perf moyenne : moyenne
  des per-match scores disponibles (même convention que `KPIStats.PerformanceScore`).
- [x] **A5 — Module dimensions + notes** : dimension libre (≥2 valeurs distinctes),
  top 3 + flop 3 des groupes ≥ MinDimensionGroupMatches triés par delta winrate,
  NoteTier via `analysis.PerfTier` (nil si perf absente). Comparateur générique
  `breakdown.CompareByKey` AJOUTÉ (+ tests) car `CompareToHistorical` est map-only.
  ÉCART MAJEUR assumé : conversion depuis les **raw rows** (`rawRowsToBreakdownRows`,
  libellés FR MapNameFR/PairNameFR/PlaylistName) et NON les canonical rows — les
  canonical de `LoadPlayerMatches` ne sont pas enrichies FR → auraient affiché des
  libellés EN sous locale FR (violation critère de succès). Le convertisseur canonical
  `rowsToBreakdownInputs` (squad) reste map-only et intouché (pas de 3e copie créée).
  Détail plan initial : conversion des canonical rows vers
  `breakdown.Row` via le convertisseur EXISTANT de la page Synthèse (le localiser ;
  s'il est privé, l'extraire vers un helper partagé sans dupliquer — règle n°6). Pour
  chaque dimension libre (DEC-8) : agrégats scoped via
  `breakdown.ByMap/ByMode/ByPlaylist`, deltas vs `all` via
  `breakdown.CompareToHistorical` (cartes) ou le même alignement par clé pour
  mode/playlist (si `CompareToHistorical` est map-only, ajouter dans
  `internal/analysis/breakdown/` un équivalent générique par clé — algo pur + tests
  unitaires purs, PAS dans le service). `NoteTier` via `analysis.PerfTier` sur la
  moyenne de perf du groupe, nil sous `MinDimensionGroupMatches` (DEC-2/DEC-4).
- [x] **A6 — Module tendance** : gate `len>=20 && span>=14j` (DEC-4), granularité via
  `temporal.ResolveAdaptive` (période dérivée de l'étendue : ≤31j→1d, ≤366j→1w, sinon 1m),
  `temporal.BucketByGranularity` sur un wrapper `trendRow` (HasStartTime). Par bucket :
  matchs, winrate, perf moyenne. Aucun lissage serveur. Détail : binning via
  `analysis/temporal` (`ResolveAdaptive` pour la granularité, `BucketByGranularity`
  pour les buckets — signatures à re-vérifier sur pièces) sur `scoped` ; par bucket :
  matchs, winrate, perf moyenne. Aucun lissage côté serveur en v1 (le front trace la
  série brute par bucket).
- [x] **A7 — Module classé** : émis si `rankedCapable` ET scope majoritairement CSR
  (`RankDelta.Kind=="csr"` → absent en scope non-classé ou LUSR, cf. scénarios D2).
  `DeltaSum`/`RatingKind` réutilisent `KPIStats.RankDelta` ; attendu vs réel via NOUVEAU
  helper pur `analysis.ExpectedVsActual` (+ test) sur les `SkillExpectedWinProb` des raw
  rows CSR. Détail : émis si `rankedCapable` (capability
  `match.skill.snapshot` injectée au service via le wiring, pattern
  `titleSupportsLiveCSR` — DEC-7) ET si le scope contient des matchs avec rating CSR.
  `DeltaSum`/`RatingKind` : réutiliser `KPIStats.RankDelta` (déjà calculé) ;
  `ExpectedWinRate` : moyenne des `expected_win_prob` disponibles sur le scope (source :
  même champ que celui servi par ligne dans `ExplorerMatchesRow`) ;
  `ActualWinRate` : winrate réel du scope. Nouveau helper pur dans
  `internal/analysis/` si le calcul attendu-vs-réel n'existe pas (grep d'abord — skill
  `go-features`) + test unitaire.
- [x] **A8 — Handler + wiring** : `handleQueryMatches` propage `include_briefing` →
  `IncludeExplorerBriefing` et recopie `mhResp.Briefing`. Capability câblée via
  `WithRankedCapable(r.titleSupportsLiveCSR(pdb))` dans `MatchHistoryCtx`
  (registry_pages_home.go) — partagé Explorer+Historique, inoffensif pour l'Historique
  (flag absent → briefing non construit, non-régression testée A9). Détail : recopie
  `mhResp.Briefing` dans la réponse Explorer (mapping pur, zéro logique) ; le flag
  `include_briefing` de la requête Explorer est propagé (A1). Wiring de la capability
  vers le service dans `registry_pages_*` (suivre le pattern existant). Vérifier que la
  page Historique (handler match-history) ne sert PAS le briefing étendu (flag absent).
- [x] **A9 — Tests backend** : analysis (`ExpectedVsActual`, `AggregateKDA`,
  `CompareByKey` — nominal/vide/données manquantes) ; service
  (`match_history_service_briefing_test.go` : low-sample, dimension mono-valeur non émise,
  groupe <10 exclu, ranked false→nil + LUSR→nil, span<14j→nil, deltas baseline signés,
  frise cappée+triée) ; handler (`explorer_briefing_test.go` : propagation flag +
  briefing recopié / absent). NB : cas testés directement sur la méthode + fixtures raw
  (pas de repo mock nécessaire). Détail plan :
  - analysis : tests unitaires purs des nouveaux helpers (comparaison par clé A5,
    attendu-vs-réel A7) — cas nominal, vide, données manquantes.
  - service : tests de `buildExplorerBriefing` sur fixtures (repo mocké via `port`) —
    (1) n < 10 → socle seul + LowSample ; (2) dimension à 1 seule valeur → non émise ;
    (3) groupe < 10 matchs → NoteTier nil ; (4) rankedCapable=false → Ranked nil ;
    (5) étendue < 14 j → Trend nil ; (6) deltas baseline signés corrects sur fixture
    connue ; (7) frise cappée à 60 et triée chrono asc.
  - handler : httptest — `include_briefing:true` → briefing présent ;
    absent/false → réponse identique à l'existant (non-régression Historique incluse).
- [x] **A10 — Contrat OpenAPI** (fait en Lot A car GATÉ par
  `openapi_schema_drift_test.go`) : les 8 schémas `ExplorerBriefing*` auto-dérivés par
  Huma étaient MISSING → ajoutés au `api/openapi.yaml` manuel (émis via l'outil intégré
  `OPENAPI_EMIT_OUT`), + propriété `briefing` sur `ExplorerMatchesQueryResponse` et
  `MatchHistoryPageResponse` ; `generated.ts` régénéré (`make generate-types`). Types
  miroir manuels `types.ts` → traités en B1 (front). Détail plan initial :
  vérifier le workflow (grep `openapi`
  dans le Makefile et `docs/COMMANDS.md`) : si `openapi.yaml` est dumpé depuis Huma,
  le régénérer puis `make generate-types` ; dans tous les cas, MAJ manuelle des types
  miroir dans `apps/web/src/lib/api/types.ts` (pattern des types Explorer existants,
  l.658-755). Cet item peut être livré en tête du Lot B si le workflow l'exige —
  le noter `[~]` ici avec référence dans ce cas.

**Gate Lot A** :
```
cd apps/go-api && go test ./internal/analysis/... ./internal/service/... ./internal/api/...
cd apps/go-api && go test ./...
make go-api-lint
```
Pas de `-tags=integration` requis : aucune écriture persist/sync touchée (lecture pure).
Aucun item sans statut. Commit(s) du lot avec accord utilisateur.

## LOT B — Front : socle du bandeau

- [x] **B1 — Types + requête** : alias `ExplorerBriefing*` (depuis `components['schemas']`,
  generated.ts régénéré en A10) + `include_briefing?: boolean` dans `types.ts` ;
  `ExplorerPage.tsx` envoie `include_briefing: true` (mode Matchs uniquement, pas ally/enemy).
  Query key inchangée (flag constant → confirmé sur pièces dans queries.ts). Détail plan :
  types `ExplorerBriefing*` dans `types.ts` (si non faits
  en A10) ; `ExplorerPage.tsx` envoie `include_briefing: true` dans la requête (~l.205-227).
  Vérifier que la clé de query (`filterHash` / queryKey dans `queries.ts` +
  `lib/query/keys.ts`) reste correcte — le flag étant constant, pas d'entrée de clé
  nécessaire, le confirmer sur pièces.
- [x] **B2 — Composant `ExplorerBriefingStrip.tsx`** : rangée socle 4 tuiles (Matchs+période,
  Taux de victoire+V-D-N+delta, FDA agrégat+delta, Perf. moyenne+delta) via `KpiCard` (chrome
  partagé) + frise `OutcomeSequenceTape` (height 64). Deltas colorés par signe (tokens),
  `winRateColor`/`kdaNetColor`. low_sample → socle + mention. briefing absent → rien. Logique
  pure extraite dans `ExplorerBriefing.logic.ts`. Détail plan : nouveau fichier sous
  `features/explorer/`, ≤ 500 L ; extraire des sous-fichiers si dépassement) :
  - Rangée socle de 4 cards `KpiCard` (pattern `ExplorerEncounterBriefing`) :
    Matchs (n + période), Bilan (V-D-N + taux de victoire), FDA (agrégat), Perf moyenne.
  - Deltas vs baseline en sous-texte de card (pattern trends de `KpiCell`) quand
    `baseline` présent ; `winRateColor` pour le taux de victoire.
  - Frise `OutcomeSequenceTape` (height 64, labels i18n) sous la rangée quand
    `outcome_sequence` non vide.
  - `low_sample: true` → socle rendu + mention « Échantillon faible (N matchs) »
    (i18n), aucun module.
  - Briefing absent (réponse sans le champ) → le composant ne rend rien (aucune
    régression d'affichage).
- [x] **B3 — Insertion** : `<ExplorerBriefingStrip>` rendu en tête de
  `ExplorerMatchesResultsBlock`, au-dessus du compteur/tri/export (fichier matchesMode non
  gonflé, composant dans son propre fichier). Détail plan : rendu dans `ExplorerMatchesResultsBlock`
  (`ExplorerPage.matchesMode.tsx`) au-dessus du bandeau compteur/tri/export, sans
  gonfler ce fichier au-delà du seuil (le composant vit dans son propre fichier).
- [x] **B4 — i18n + couleurs** : section `[explorer.briefing.*]` (fr+en, « Bilan », « Taux
  de victoire », « Échantillon faible », séries) régénérée. Zéro hex/classe couleur (grep OK),
  tokens sémantiques uniquement. Détail plan : toutes les strings dans `explorer.toml` (fr + en, FR
  sans anglicismes : « Bilan », « Taux de victoire », « Échantillon faible », « Série » —
  jamais « streak »/« winrate ») puis régénération du manifest
  (`node apps/web/scripts/build_i18n_manifests.mjs`). Aucune couleur hex ni classe
  Tailwind couleur : tokens sémantiques uniquement (skill `color-tokens`).
- [x] **B5 — Tests front** : `ExplorerBriefing.logic.test.ts` (10 cas : KDA agrégat,
  winrate, formatage deltas signés, mapping outcome, palier). `make check-types` OK ;
  vitest complet 2161 passés / 0 échec ; eslint 0 erreur (2 warnings pré-existants hors
  périmètre). Détail plan : vitest sur la logique extraite (formatage deltas, choix
  d'affichage low-sample — extraire en helper pur testable si nécessaire) ;
  `make check-types` ; `make test-web`.

**Gate Lot B** :
```
make check-types
make test-web
grep -rEn '#[0-9a-fA-F]{3,6}' apps/web/src/features/explorer/ExplorerBriefing* && echo FAIL || echo OK
```
Vérification visuelle rapide (make dev, une recherche large) : rangée + frise rendues.
Aucun item sans statut. Commit(s) avec accord utilisateur.

## LOT C — Front : modules conditionnels + mini-graphes

- [x] **C1 — Module dimensions (notes)** : `ExplorerBriefingModules.tsx` → `DimensionCard`
  (1 carte/dimension) + `DimensionRow` : libellé, n, taux de victoire (coloré), delta signé
  ▲/▼ (tokens), note = badge palier 1..5 avec les MÊMES libellés que le filtre
  (`explorer.filters.perf_tier_*`, token `perf-tier-N`). note_tier nil → n + delta sans note
  (tiret). Détail plan : sous la rangée socle, une carte par dimension
  servie (carte/mode/playlist) : top 3 / flop 3 avec libellé, n, taux de victoire,
  delta signé vs baseline (▲/▼, tokens), et la note (palier 1..5) rendue avec les MÊMES
  libellés que le filtre « Palier de performance » (réutiliser la source de labels des
  options de filtre — pas de nouveau mapping). Note absente (`note_tier` nil) →
  afficher n + delta sans note.
- [x] **C2 — Module tendance** : `TrendCard` via wrapper existant `TimeseriesLineChart`
  (height 120, xAxisType time), série taux de victoire par bucket (couleur via
  `colorToken: 'outcome-win'`, résolu en hex par le wrapper — PAS `tokenCssVar` qui donnerait
  un `var()` non résolu en canvas). Perf omise en v1 (DEC-5 : seconde série optionnelle,
  écartée pour lisibilité à 120px). Aucun nouveau wrapper. Détail plan : sparkline compacte (~120 px de haut, pattern
  mini-chart RivalryCard) à partir de `trend.points` via le wrapper timeseries
  existant (`components/charts/` — vérifier le catalogue README avant d'envisager un
  nouveau wrapper, DEC-5) : taux de victoire par bucket (axe principal) ; perf moyenne
  en seconde série si le wrapper le permet sans surcharge visuelle, sinon omise.
- [x] **C3 — Module classé** : `RankedCard` gated `useCapability('ranked')` ET
  `briefing.ranked` présent : delta CSR cumulé (signe coloré tokens) + ligne « Attendu vs
  réel » (`expected_win_rate` / `actual_win_rate` en %). Sous H5 : capability absente +
  backend n'émet rien → module absent (pas de card N/A). Détail plan : gated `useCapability('ranked')` (DEC-7) ET
  `briefing.ranked` présent : card delta CSR cumulé (signe coloré tokens outcome) +
  ligne « Attendu vs réel » (`expected_win_rate` vs `actual_win_rate`, formatés %).
  Sous Halo 5 : module absent (le backend n'émet rien, le front n'affiche rien — pas
  de card N/A).
- [x] **C4 — États dégradés** : `ExplorerBriefingModules` retourne null si aucun module ;
  chaque module gaté sur présence de son bloc (`dimensions.length`, `trend != null`,
  `ranked != null` + capability). Deltas nil → formatteurs renvoient '' ; perf nil → note
  absente. Aucun NaN/undefined rendu. Détail plan : chaque module s'omet proprement quand son bloc est nil
  (aucun placeholder vide) ; vérifier qu'aucun module ne rend de NaN/undefined avec
  des blocs partiels (perf nil, expected nil).
- [x] **C5 — i18n des modules** : clés `[explorer.briefing.dim_*|trend_*|ranked_*]` (fr+en)
  + réutilisation des `explorer.filters.perf_tier_*` pour les notes ; manifest régénéré.
  Gate : `check-types` OK, eslint 0 erreur, hex 0, vitest complet 2161 verts.

**Gate Lot C** :
```
make check-types
make test-web
node apps/web/scripts/build_i18n_manifests.mjs && git diff --exit-code apps/web/src/lib/i18n/generated/ && echo MANIFESTS-OK
```
Aucun item sans statut. Commit(s) avec accord utilisateur.

## LOT D — Livraison

- [ ] **D1 — Gates complets** (delivery-checklist) :
  ```
  cd apps/go-api && go test ./...
  make go-api-lint
  make check-types
  make test-web
  ```
- [ ] **D2 — Vérification visuelle en conditions réelles** (make dev, profil JGtm,
  locale FR puis EN) — 6 scénarios, chacun avec capture du comportement attendu :
  1. Recherche sans aucun filtre (gros n) : socle + frise + dimensions + tendance.
  2. Filtre sur UNE carte : module « par carte » ABSENT, « par mode/playlist » présents.
  3. Contexte classé (PVP classé) : module classé présent, deltas CSR cohérents.
  4. Période courte / peu de matchs (n < 10) : socle + mention échantillon faible,
     AUCUN module.
  5. Titre Halo 5 : bandeau rendu, module classé ABSENT, aucun N/A ni valeur aberrante
     (OC/DR nil tolérés).
  6. Recherche à 0 résultat : aucun bandeau, état vide existant inchangé.
  Recouper à la main les valeurs du scénario 1 (winrate + n) avec le tableau.
- [ ] **D3 — Non-régression Historique** : la page Historique de matchs (consommateur
  de `GetPage`) rend à l'identique (réponse sans briefing étendu) — vérification
  visuelle + test handler A9 déjà en place.
- [ ] **D4 — Docs** : si `.ai/CHARTS_AND_TABLES.md` recense les surfaces par page, y
  ajouter le bandeau ; si un nouveau wrapper chart a été créé (normalement non, DEC-5),
  MAJ `apps/web/src/components/charts/README.md`.
- [ ] **D5 — Thought log** : entrée de clôture (date, décisions, résultats des gates,
  écarts éventuels vs plan) — OBLIGATOIRE avant de rendre la main.

## Hors périmètre (NE PAS TRAITER — consigner en Découvertes si tentation)

- Mode Joueur de l'Explorer (player-query) et ses cards existantes.
- Page Synthèse (aucune modification, aucune factorisation opportuniste de ses calculs
  au-delà du convertisseur breakdown.Row explicitement prévu en A5).
- Narration textuelle générée (phrases coach/moteur narratif) — v2 éventuelle.
- Export CSV du briefing ; notes dans les cellules du tableau ; tri par note.
- Toute retouche du tableau `ExplorerMatchesTable` ou des filtres existants.

## Découvertes en cours d'exécution

> Consigner ici (ou en thought log) toute anomalie/opportunité rencontrée hors
> périmètre. NE PAS la traiter dans ce chantier.

- **[Lot A] KDA agrégat inliné** : `internal/analysis/explorer_target_stats.go:69` réinline
  la formule ADR 0006 `((k+a/3)−d)/n`. Helper canonique `analysis.AggregateKDA` créé et
  utilisé par le briefing ; l'inline legacy N'A PAS été migré (hors périmètre). Dette : à
  la prochaine occurrence, migrer + garde-rail (règle n°6).
- **[Lot A] Convertisseurs canonical→breakdown.Row multiples** : `rowsToBreakdownInputs`
  (squad, map-only) + `rawRowsToBreakdownRows` (briefing, raw→FR). Non factorisés (sources
  et champs différents) — 2 copies, sous le seuil règle n°6.
- **[Lot A] Décision raw-rows vs canonical pour dimensions/baseline** : les canonical de
  `LoadPlayerMatches` ne sont pas enrichies FR (contrairement à Synthèse qui appelle
  `EnrichCanonicalAssetTranslations`). Pour éviter des libellés EN sous FR, dimensions/
  tendance/baseline sont bâties sur les `MatchHistoryRawRow` (déjà FR + post-exclusions).
  Le socle KPIs + le delta rating restent canonical.

## Protocole de reprise de session

1. `git branch --show-current` → `feat/explorer-briefing-cards` ; `git log --oneline -10`.
2. Relire ce plan : le premier item non coché du premier lot non clos est le point de
   reprise. Un lot est clos quand tous ses items ont un statut ET son gate est passé.
3. Relire l'entrée thought_log la plus récente du chantier.
4. Re-vérifier sur pièces les ancrages du lot en cours (doctrine RE-VÉRIFIER).

## Statuts d'items (rappel du contrat plan-execution)

`[x]` fait et vérifié · `[~]` couvert ailleurs (référence obligatoire) · `[!]` non
traité (justification écrite obligatoire). Aucune case vide à la clôture. Ordre strict :
un item/lot ne démarre pas tant que le précédent n'est pas clos. Les seuls reports
valides sont ceux listés par le skill `plan-execution`.
