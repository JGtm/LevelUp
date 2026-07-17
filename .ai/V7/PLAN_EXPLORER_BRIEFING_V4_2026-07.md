# PLAN — Explorer briefing V4 : corrections post-revue V3 + refonte tuiles + Classement par chaîne

Statut : PLANIFIE (aucune ligne de code écrite — plan rédigé par l'architecte Opus).
Date : 2026-07-17.
Chantiers précédents (mergés en local) : `.ai/V7/PLAN_EXPLORER_BRIEFING_V3_COMPACT_2026-07.md`
(compaction, committée mais avec écarts trouvés en revue visuelle) et
`.ai/V7/PLAN_EXPLORER_BRIEFING_V2_2026-07.md` (ajustements post-revue V1).

Branche cible d'implémentation : **`feat/explorer-briefing-compact`** (déjà la branche
courante ; NE PAS en changer ; NE committer que ce que ce plan autorise, par phase).

> Contrat d'exécution : ce plan s'exécute sous le skill **`plan-execution`** (ordre strict,
> une étape close avant la suivante, aucun report d'action exécutable maintenant, statut sur
> chaque item, zéro fix hors périmètre). En cas de divergence, le présent plan fait foi ; à
> défaut, le skill est le défaut. Avant de finaliser toute modification du plan : skill
> **`plan-review`**. Avant chaque commit : skill **`delivery-checklist`**. Code Go :
> **`arch-rules`** + **`canonical-types`** + **`db-schema`** + **`go-features`** ; code
> React/TS : **`frontend-patterns`** ; toute couleur : **`color-tokens`**. Rappels
> transverses : tokens sémantiques UNIQUEMENT (aucun hex/classe Tailwind couleur dans
> `features/`/`components/`) ; seuils fichier ≤ 500 L / fonction ≤ 80 L / ≤ 5 params /
> complexité ≤ 12 ; FR sans anglicismes (« série » pas « streak », « Taux de victoire » pas
> « WR ») ; parité i18n FR/EN par typage ; branchement par **capability** jamais par
> `slug == …` ; **pas de commandes `go` concurrentes** (corruption du cache Windows —
> séquentiel, tuer les `link.exe` orphelins) ; vitest hors sandbox
> (`dangerouslyDisableSandbox=true`) ; purger `apps/web/node_modules/.tmp` avant
> `make check-types`.

---

## 1. Objectif et critères de succès (mesurables)

**Objectif.** Corriger et faire évoluer le bandeau de briefing de l'Explorer (mode Matchs)
après la revue visuelle de la compaction V3, selon les 12 décisions utilisateur tranchées le
2026-07-17 (§3). En résumé : (a) **corriger le bug de fond du module Classement** — le
pt/match incohérent avec la progression de paliers — en recalculant **par chaîne de
playlist** (une ligne par chaîne, jamais de flèche inter-chaînes) ; (b) **supprimer
définitivement la sparkline** et purger le DTO `trend` devenu mort ; (c) **refondre le socle
de tuiles** (≤ 8) en reprenant le contenu de la tuile « Taux de victoire » de
`HomeHeroKPIGrid`, en colorant la Perf, en ajoutant « Durée totale » + « Pic FDA »
(+ éventuellement « Pic MMR »), et en renommant « Séries » → « Meilleure série » ; (d)
**déplacer le Classement** hors du socle vers une 4ᵉ colonne scindée avec « Par contexte » ;
(e) **corriger le clipping des tooltips** (portal) ; (f) **mettre en valeur MVP/LVP dans le
tableau** en réutilisant le style best/worst du scoreboard/matrice d'impact ; (g) **grilles
pleine largeur adaptatives**, padding de cellules réduit, reformulation `tip_dimensions` ;
(h) **nettoyer les dettes préexistantes** du périmètre en dernière phase.

**Critères de succès (tous vérifiables ; la vérification NAVIGATEUR est reprise par
l'utilisateur — cf. §« À vérifier visuellement ») :**

1. **Classement cohérent par chaîne.** Le module Classement n'affiche JAMAIS une progression
   de paliers dont le pt/match contredit le sens : chaque ligne correspond à UNE chaîne
   `(type, playlist_group)` ; les paliers début/fin ET le pt/match viennent des mêmes matchs
   de la MÊME chaîne. Si le scope couvre plusieurs chaînes LUSR, il y a une ligne par chaîne
   (jamais de flèche inter-chaînes). CSR reste une seule ligne (chaîne unique « ranked »,
   invariant P-3 préservé). Vérifié par tests unitaires analysis + service (§5) et revue
   visuelle.
2. **Sparkline supprimée.** Aucune `<Sparkline>` dans le bandeau Explorer ; DTO `trend`
   (`ExplorerBriefingTrend` + `...TrendPoint` + champ `Trend` + `buildBriefingTrend` +
   consts `minTrendMatches`/`minTrendSpanDays` + schémas OpenAPI + `types.ts`) entièrement
   purgé (grep de clôture = 0). La primitive `Sparkline` (`components/charts/`) et ses
   consommateurs admin restent INTACTS.
3. **Socle de tuiles refondu (≤ 8, pleine largeur, cascade).** Base présente hors low_sample
   (6) : Matchs, Taux de victoire (contenu hero repris de `HomeHeroKPIGrid` : `OutcomeBar` +
   V-D-N + tooltip), FDA, Perf **colorée** (`getPerfColor`), Durée totale, Pic FDA.
   Conditionnelles par ORDRE DE PRIORITÉ (cascade, cap à 8 → au plus 2 des 3) : (1) Meilleure
   série, (2) Pic rang, (3) Pic MMR. Jamais > 8, jamais de trou en fin de rangée (grilles
   adaptatives, §3 DEC-GRID). low_sample = 4 tuiles de base.
4. **« Séries » → « Meilleure série ».** `streaks_title` renommé FR « Meilleure série » /
   EN « Best streak » ; contenu inchangé (valeur bicolore best V / worst D).
5. **Classement en 4ᵉ colonne.** Le Classement a QUITTÉ le socle ; il est rendu comme bloc
   compact sous « Par contexte » dans la 4ᵉ colonne de la rangée « Par… », les hauteurs
   s'harmonisant avec les cartes de dimension.
6. **Tooltips non clippés.** Le panneau `InfoTooltip` est rendu via un **portal**
   (`createPortal` → `document.body`, position `fixed`) : lisible en entier au survol/focus
   d'une icône (i) dans une tuile (clipping `overflow-hidden` de `KpiCard` résolu), fermeture
   au scroll/resize/clic extérieur, `role="tooltip"` + aria conservés. Les ~10 consommateurs
   existants ne régressent pas (tests).
7. **MVP/LVP dans le tableau.** Le meilleur ET le pire par colonne clé (au moins Perf, FDA,
   Frags, Score) sont surlignés sur TOUT le scope chargé côté client, via le style best/worst
   RÉUTILISÉ de `MatchScoreboard.logic` (teinte 28 % oklab, tokens `outcome-win`/
   `outcome-loss`, gras 600/500, garde ≥ 2 valeurs), indépendant du tri. Aucune 3ᵉ copie du
   style créée.
8. **Padding réduit + reformulation.** En-tête `th` et cellules `td` du tableau passent de
   `px-2 py-1.5` à un padding plus compact (§3 DEC-PAD) ; `tip_dimensions` = texte reformulé
   (DEC-10, FR+EN).
9. **Gates verts** (par phase, §5) : `cd apps/go-api && go test ./...` = 0 (SÉQUENTIEL) ;
   `make go-api-lint` = 0 ; `make generate-types` idempotent (0 diff résiduel) +
   `TestOpenAPISchemaDrift` vert ; `make check-types` = 0 (cache `.tmp` purgé) ; `make
   test-web` (dangerouslyDisableSandbox) vert ; `cd apps/web && npm run lint` = 0 erreur ;
   greps de clôture anti-résidus verts. `-tags=integration` NON requis (§4, justifié).
10. **Dettes traitées.** `record_label` (i18n morte) purgée ; god-file `ExplorerMatchesTable`
    (800 L) réduit sous le seuil OU chantier séparé documenté ; `eslint-disable` inutile
    d'`ExplorerPage` retiré s'il existe ; déplacement des plans vers `.ai/V7/` committé
    (Phase 0). Les dettes trop larges sont listées « NON traitées (justification) ».
11. **Changelog** : entrée `[Unreleased]` v7.0 mise à jour dans `docs/CHANGELOG.md` ET
    `docs/FR/CHANGELOG.md` (parité EN/FR même commit).

---

## 2. Constat sur pièces — état actuel (fichier:ligne réels, vérifiés le 2026-07-17)

> Doctrine du projet : RE-VÉRIFIER chaque ancrage sur pièces AVANT de coder ET avant de
> cocher (le code a pu bouger). Numéros ci-dessous = état vérifié au 2026-07-17.

### 2.1 Worktree non committé — À PRÉSERVER (git status vérifié)

- `git branch --show-current` = `feat/explorer-briefing-compact`.
- **Déplacement des plans** : `.ai/PLAN_EXPLORER_BRIEFING_V2_2026-07.md` et
  `...V3_COMPACT...` sont supprimés de `.ai/` (statut ` D`) et re-créés sous `.ai/V7/`
  (statut `??`). → à committer proprement en Phase 0.
- **Édition manuelle utilisateur** : `apps/web/src/lib/i18n/manifests/explorer.toml` (` M`)
  + `apps/web/src/lib/i18n/generated/explorer.ts` (` M`). Le diff = `explorer.briefing.
  tip_dimensions` : l'utilisateur a RETIRÉ la parenthèse « (masqué quand tout l'historique
  est affiché) » / « (hidden when the full history is shown) » de FR ET EN. État courant du
  fichier (à NE PAS écraser sans lire — cf. DEC-10) :
  - FR = « Vos meilleurs et moins bons terrains sur les matchs affichés. Par ligne : nombre
    de matchs, taux de victoire, écart face à votre habitude et une note de performance,
    d'Excellent à Mauvais, fondée sur votre score de performance personnel. »

### 2.2 Backend — module Classement (bug de fond) et sa chaîne de données

- **DTO** `apps/go-api/internal/domain/explorer_briefing.go` :
  - `ExplorerBriefingRanked` (`:125-130`) = `Kinds []ExplorerBriefingRankedKind`.
  - `ExplorerBriefingRankedKind` (`:132-156`) : `Kind`, `Matches`, `TierStartLabel`,
    `TierEndLabel`, `TierStartIsPlacement`, `TierEndPlacementRemaining`, `DeltaPerMatch` —
    **une entrée par TYPE, PAS par chaîne** (pas de `PlaylistGroup`).
  - `ExplorerBriefingTrend` (`:105-110`) + `ExplorerBriefingTrendPoint` (`:112-118`) + champ
    `Trend` (`:39`) → **à purger (DEC-SPARK)**.
  - `ExplorerBriefingScope` (`:60-70`) : `Matches/Wins/Losses/Ties/DNF/WinRate/KDA/AvgPerf` —
    **aucun champ de durée ni de pic** (à étendre, DEC-TILES).
- **Service** `apps/go-api/internal/service/match_history_service_briefing_ranked.go` :
  - `buildBriefingRanked(ctx, scope []MatchHistoryRawRow, scopedKPIs *KPIStats)` (`:31-50`) :
    itère `scopedKPIs.RankDeltas` (buckets PAR TYPE), `DeltaPerMatch = rd.Value/rd.Count`
    (`:57-60`) = **moyenne d'une SOMME de deltas par-match tous playlist_group confondus au
    sein d'un type** → cause du bug.
  - `buildRankedKind` (`:55-86`) : filtre les rows du TYPE (`strings.EqualFold` casse,
    `:69`), tri chronologique (`:73`), `firstTieredRow`/`lastTieredRow` (`:90-106`) →
    `TierStartLabel`/`TierEndLabel` **du type, pas de la chaîne** ; placement via
    `applyTierStart`/`applyTierEnd` (`:114-138`, `PlacementDone`/`PlacementTotal`).
- **Orchestration** `apps/go-api/internal/service/match_history_service_briefing.go` :
  - `buildExplorerBriefing(ctx, filtered, allRaw, scopedKPIs)` (`:59-86`) : `if s.rankedCapable
    { b.Ranked = buildBriefingRanked(ctx, filtered, scopedKPIs) }` (`:76-78`).
  - `scopedKPIs` = `kpisFromScoped(scoped)` (`match_history_service.go:297,368-370`),
    calculé par `analysis.ComputeKPIStats` sur des `canonical.PlayerMatchRow`.
- **Accumulation** `apps/go-api/internal/analysis/kpi_stats.go` : `rankBuckets` keyé par
  `snap.RatingType` UNIQUEMENT (`:88-96`) ; `RankDelta` majoritaire (`:176-195`) + `RankDeltas`
  (tous les types, `:196-217`). **`RankDeltas` n'a qu'UN consommateur : `buildBriefingRanked`**
  (grep : + définition `squad_v2.go:146` + 2 tests `kpi_stats_test.go`).
- **Snapshot canonique** `apps/go-api/internal/games/canonical/match.go` : `SkillSnapshot`
  (`:276-292`) porte **DÉJÀ** `PlaylistGroup *string` (`:283`), `RatingValue *float64`
  (`:278`), `Delta *float64` (`:282`), `TierCodeFR`, `SubTier`, `MeasurementRemaining`.
- **Source du delta (au sync, PAR chaîne)** `apps/go-api/internal/sync/skill/
  skill_v2_canonical.go` : `d = rating − *prev` (`:163-166`) où `prev =
  loadPreviousLUSRRating(..., state.PlaylistGroup, ...)` (`:213-221`, `WHERE ... playlist_group
  = ?`). Le delta par-match est donc DÉJÀ per-chaîne, persisté en
  `match_skill_rank.rating_delta`.
- **CSR = chaîne unique** `apps/go-api/internal/sync/csr_writes.go:104` (« Le PlaylistGroup
  est figé à "ranked" ») + `:131` (`PlaylistGroup: PerfChainRanked`). → CSR ne se scinde PAS
  (une seule ligne).
- **Chaînes LUSR** (4) `apps/go-api/internal/games/halo_infinite/skillchain/classify.go:21-26`
  : `arena_slayer`, `arena_objectif`, `btb`, `chaos`.
- **Vue `_latest` = 1 ligne/match, priorité CSR > LUSR** `apps/go-api/internal/games/
  halo_infinite/migrations/steps_player_match_skill_rank.go:57-63,110` (+ Q5 comment
  `match_history_repo.go:207-210`). Chaque match ⟹ exactement une paire `(rating_type,
  playlist_group)`.

### 2.3 Backend — colonnes SQL disponibles (faisabilité vérifiée)

- **`match_skill_rank_latest` porte** : `rating_type, rating_value, tier, tier_fr, sub_tier,
  rating_delta, playlist_group, expected_win_prob, tier_label` — PROUVÉ par
  `apps/go-api/internal/platform/duckdb/player_matches_scan.go:129-141`
  (`playerMatchesSkillRankTpl` les SELECT tous). → **playlist_group, rating_value ET
  rating_delta sont disponibles**, aucune migration de schéma nécessaire.
- **Requête skill du briefing** `Q5PlayerSkillRankHistoryTpl`
  (`apps/go-api/internal/platform/duckdb/queries_career.go:99-108`) : SELECT SEULEMENT
  `match_id, tier, tier_fr, rating_type, tier_label, expected_win_prob` (6 colonnes) — NE
  lit PAS `playlist_group`/`rating_value`/`rating_delta`. Scan + assignation dans
  `mergeHistorySkillRanks` (`match_history_repo.go:180-229`, struct `skill` `:193-199`,
  scan `:204`, assignation `:222-226`).
- **Requête shared du briefing** `Q5SharedHistory` (`queries_career.go:61-96`) : `FROM
  v_match_full r JOIN match_participants p` ; lit `time_played_seconds` (`:87`) mais PAS
  `r.duration_seconds`. Or `v_match_full` PORTE `duration_seconds` (prouvé
  `player_matches_repo.go:274` : `COALESCE(r.duration_seconds, 0) AS duration_seconds`). →
  **la durée par match est exposable sur la raw row** en ajoutant `r.duration_seconds` au
  SELECT.
- **Raw row** `apps/go-api/internal/domain/match_history.go` : `MatchHistoryRawRow`
  (`:10-68`) porte `SkillRatingType` (`:51`), `SkillTier` (`:49`, nom EN « Diamond » —
  ordonnable via `analysis.CSRTierOrdinal`), `SkillTierLabel` (`:52`), `PlacementDone/
  PlacementTotal` (`:60-61`), `TeamMMR` (`:37`), `KDA` (`:42`), `TimePlayedSeconds` (`:46`) —
  mais **PAS** `PlaylistGroup`, ni `RatingValue`, ni `RatingDelta`, ni `SubTier`, ni
  `DurationSeconds`.

### 2.4 Frontend — bandeau (état V3 courant)

- **`apps/web/src/features/explorer/ExplorerBriefingStrip.tsx`** (200 L) :
  - Grille socle : `grid gap-2 grid-cols-2 sm:[grid-template-columns:repeat(auto-fit,
    minmax(150px,1fr))]` (`:79`).
  - Tuiles de base : Matchs (`:81-86`), Taux de victoire + **sparkline** (`:89-132`, sparkline
    `:98-108`, `trendValues` `:75`, `record_vdn` `:111`), FDA (`:135-159`), Perf (`:162-181`,
    **valeur NON colorée** `perf.toFixed(0)` `:166`).
  - `RankedTile` (`:184`, gaté `hasRanked` `useCapability('ranked')` `:53`) et `StreaksTile`
    (`:187`) rendues DANS le socle → **à déplacer/retravailler** (Classement en 4ᵉ colonne
    DEC-LAYOUT ; Séries renommée).
  - `import { Sparkline }` (`:13`) + `trendValues` (`:75`) → morts après DEC-SPARK.
- **`apps/web/src/features/explorer/ExplorerBriefingTiles.tsx`** (139 L) : `RankedTile`
  (`:89-109`, valeur = palier de fin du type majoritaire ; sous-texte 1-2 lignes PAR TYPE),
  `StreaksTile` (`:114-138`), helper local `rankedProgression` (`:29-40`). → `RankedTile`
  **retravaillée en bloc per-chaîne pour la 4ᵉ colonne** (DEC-RANK-FE) ; `StreaksTile`
  conservée (renommage seulement).
- **`apps/web/src/features/explorer/BriefingTile.tsx`** (43 L) : `KpiCard` + `px-3 py-2` ;
  label `text-3xs uppercase` + slot `info?` (`:33`) + valeur `text-xl font-bold` + slot
  `chart?` (`:38`) + sous-texte. **Slot `chart` deviendra inutilisé** après retrait de la
  sparkline (à retirer si plus aucun consommateur — DEC-SPARK / 0 code mort).
- **`apps/web/src/features/explorer/ExplorerBriefingModules.tsx`** (288 L) : grille « Par… »
  `grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4` (`:97`) rendant `DimensionCard` +
  `ContextSplitCard` (`:98-101`) ; `DominanceBand` (`:104-106,251-287`). `DOMINANCE_ITEMS`
  (`:56-70`). `InfoTooltip` déjà injecté (dimensions `:129`, contexte `:202`, bande `:265`).
- **`apps/web/src/features/explorer/BriefingSectionCard.tsx`** (57 L) : carte `rounded-lg
  border bg-card` + en-tête bordurée `flex-none border-b … px-3 py-2 text-sm font-medium`,
  slot `title: ReactNode`. Garde-rail anti-divergence `:15-21`. → **réutilisable pour le bloc
  Classement de la 4ᵉ colonne**.
- **`apps/web/src/components/ui/info-tooltip.tsx`** (57 L) : wrapper `relative inline-flex`
  (`:33`) ; panneau `absolute bottom-full … z-50 w-64 …` (`:47-53`) rendu INLINE → **clippé**
  par le `overflow-hidden` de `KpiCard`. Fermeture clic extérieur (`:23-30`), aria
  `common.tooltip.more_info_aria` (`:42`), `role="tooltip"` (`:48`). → **fix portal
  DEC-TOOLTIP**.
- **`apps/web/src/components/cards/KpiCard.tsx`** (47 L) : racine `overflow-hidden rounded-lg
  border bg-card` (`:37`) — c'est le clippeur. NE PAS le modifier (le fix vit dans
  `info-tooltip.tsx`).
- **`apps/web/src/components/ui/outcome-bar.tsx`** : primitive PARTAGÉE (consommée par
  `HomeHeroKPIGrid`, `HomeSessionCarousel`, `HomePage`, `ExplorerTargetSampleStats`,
  `ExplorerTargetCareerStats`, `SynthesisPage`). → **à réutiliser** pour la tuile WR refondue.

### 2.5 Frontend — références à RÉUTILISER (localisées sur pièces)

- **Tuile « Taux de victoire » hero** (contenu à reprendre, DEC-TILES) :
  `apps/web/src/features/home/HomeHeroKPIGrid.tsx:147-167` — valeur `${(wr*100).toFixed(0)}%`
  NEUTRE (`text-base font-bold text-foreground`, `KPI_VALUE_CLS:44`) ; V-D-N (victoires à
  gauche token `outcome-win`, `OutcomeBar` au centre, défaites à droite token `outcome-loss`,
  `:155-165`) ; tooltip des 4 issues (`:107-119`, pastilles rondes) ; accent 3px = sentiment
  (`winRateAccent` `:96-97`, bande neutre 0.45-0.55). Le sentiment est porté par l'ACCENT, pas
  par la valeur.
- **`getPerfColor`** (DEC-PERF) : `apps/web/src/lib/perf-color.ts:27-29` —
  `getPerfColor(score: number): string` retourne une **CSS var directe** (`tokenCssVar(
  perfScale(score))`, seuils 80/65/50/35). Usage : `style={{ color: getPerfColor(score) }}`
  (ex. `match-card.tsx:253`, `HomeSessionCarousel.tsx:206`).
- **Style best/worst MVP/LVP** (DEC-MVP) : source unique
  `apps/web/src/features/match-view/MatchScoreboard.logic.ts` — `getExtremes(rows, key)`
  (`:28-32`, garde ≥ 2 valeurs), `cellState(value, ex, inverted)` (`:36-43`), `cellStyle(state)`
  (`:56-64` : `backgroundColor: color-mix(in oklab, var(--ac-outcome-win|loss) 28%,
  transparent)`, `color: tokenVar`, `fontWeight: 600 best / 500 worst`). **Précédent de
  réutilisation transposable** : `apps/web/src/features/leaderboard/LeaderboardBlock.
  highlight.ts:10,38-92` (`HL_INVERTED` + `computeColumnExtremes` + `columnHighlightStyle =
  cellStyle(cellState(...))`). NB : `SquadImpactScoreboard.tsx` (matrice d'impact) DUPLIQUE le
  style inline (`extremeStyle`/`aggCellState`, `:117-133`) — 2ᵉ copie préexistante (dette,
  §6), donc la 3ᵉ surface (Explorer) DOIT importer le helper, pas recopier (CLAUDE.md §6).
- **Label de chaîne LUSR** (DEC-RANK-FE) : `apps/web/src/features/career/lusr-chains.ts` —
  `lusrChainLabel(group, locale)` (`:68-71`, résout `career.lusr.chain.{group}` du manifest
  career) + `LUSR_KNOWN_GROUPS_BY_TITLE` (`:16-22`) + `LUSR_GROUP_TOKENS` (`:32-38`). →
  **réutiliser `lusrChainLabel`** pour afficher le nom d'une chaîne (import cross-feature à
  arbitrer, DEC-RANK-FE).

### 2.6 Frontend — tableau, dettes, i18n

- **`apps/web/src/features/explorer/ExplorerMatchesTable.tsx`** (**800 L — god-file, CLAUDE.md
  §5**) : `HEADER_TH_CLASS` (`:123-124`, `px-2 py-1.5 …`), cellules `td` (`:757`,
  `px-2 py-1.5 …`). Colonne « Durée » (`:492-502`, `accessorFn r.duration_seconds`,
  `formatDurationMMSS`). Colonnes clés pour MVP/LVP : `kills` (`:411`), `deaths` (`:426`),
  `assists` (`:441`), `kda` (`:456`), `score_label` (`:481`), `perf_score` (`:504`),
  `delta_perf` (`:522`). `DOMINANCE_LABEL_KEYS`/`DOMINANCE_COLOR_TOKENS` (`:183-192`, 1ʳᵉ copie
  du mapping ; 2ᵉ = `ExplorerBriefingModules.DOMINANCE_ITEMS`). Ordinal de tier client
  `skillTierSortValue(r.skill_tier_label)` (`:564`) — utile pour un éventuel « Pic rang ».
- **`apps/web/src/features/explorer/ExplorerPage.tsx`** : `eslint-disable-next-line
  react-hooks/exhaustive-deps` en `:152` et `:161`. Le brief cite « eslint-disable inutile
  `:159` » (Découverte-8 V3) — **le n° de ligne a bougé ; RE-VÉRIFIER lequel (s'il en est) est
  réellement inutile** au lint baseline avant de retirer (Phase finale).
- **`apps/web/src/lib/i18n/manifests/explorer.toml`** : `record_label` (`:811-813`,
  « Bilan »/« Record ») — **AUCUN lecteur composant** (grep `apps/web/src` = seulement toml +
  generated) → MORTE, purgeable (DEC-DEBT). `record_vdn` (`:815-817`, utilisé Strip `:111`).
  `streaks_title` (`:884-886`, « Séries »/« Streaks ») → renommer (DEC-9). `tip_dimensions`
  (`:906-908`, édité par l'utilisateur — cf. 2.1) → reformuler (DEC-10). `streak_wins`/
  `streak_losses` (conservés).

**Conclusion du constat.** Le chantier est **backend d'abord** (Classement par chaîne +
purge trend + nouveaux agrégats socle), puis **frontend** (refonte tuiles + hero WR +
Durée/Pic + Perf couleur + renommage ; layout 4ᵉ colonne + grilles), puis **tooltips
portal**, **MVP/LVP** (réutilisation du style scoreboard), **padding + reformulation**, et
enfin **dettes**. Toutes les données du fix Classement existent déjà en base
(`match_skill_rank_latest`) et au niveau canonique (`SkillSnapshot`) — le fix est une
segmentation par chaîne + exposition de colonnes en LECTURE (aucune écriture, aucune migration
de schéma → `-tags=integration` non requis, §4).

---

## 3. Décisions — pré-tranchées (fermes, ne pas re-débattre en exécution)

### Décisions produit (utilisateur, 2026-07-17 — reprises telles quelles)

- **DEC-1 (Classement par chaîne).** Garder le format « {TYPE} … paliers début → fin »
  (flèche) MAIS recalculer par **chaîne de playlist** (`playlist_group`). Une ligne par chaîne ;
  si plusieurs chaînes dans le scope → plusieurs lignes ; JAMAIS de flèche inter-chaînes. CSR
  reste une ligne (chaîne unique « ranked »). Module CSR non cassé (P-3 : jamais mélanger deux
  systèmes). Algo pur en `internal/analysis/`, DTO segmenté par chaîne (détail technique
  DEC-RANK-BE / DEC-RANK-FE).
- **DEC-SPARK (sparkline supprimée).** La sparkline de tendance quitte définitivement la tuile
  Taux de victoire. Purge complète du DTO `trend` (mort après retrait — comme
  `outcome_sequence` en V3) : champ + types + construction service + consts + OpenAPI +
  `types.ts` + slot `chart` de `BriefingTile` s'il n'a plus de consommateur. Vérifier
  qu'aucun autre consommateur du DTO `trend` n'existe avant purge.
- **DEC-TILES (refonte du socle, ≤ 8 tuiles, règle de cascade).** Base TOUJOURS présente hors
  low_sample (6 tuiles) :
  1. **Matchs** (inchangé, + période).
  2. **Taux de victoire** — reprendre le CONTENU de la tuile WR de `HomeHeroKPIGrid`
     (`OutcomeBar` alimenté par `scope.wins/losses/ties/dnf` + V-D-N flanquant en tokens
     `outcome-win`/`outcome-loss` + tooltip des 4 issues) ; conserver le delta « vs habituel »
     en sous-texte (masqué en plein historique). L'`OutcomeBar` REMPLACE la sparkline (pas de
     donnée nouvelle : les comptes V/D/N/DNF sont déjà dans le scope).
  3. **FDA** (inchangé, coloré `kdaNetColor`).
  4. **Perf. moyenne** — valeur COLORÉE via `getPerfColor(score)` (DEC-PERF).
  5. **Durée totale** — somme des `duration_seconds` du scope (agrégat SERVEUR), format lisible
     « 42 h 10 » (DEC-DURATION).
  6. **Pic FDA** — meilleur `KDA` d'un seul match du scope (agrégat SERVEUR, DEC-PEAK).
  Puis CONDITIONNELLES, rendues par ORDRE DE PRIORITÉ DÉCROISSANTE (**règle de cascade** : on
  rend les présentes dans cet ordre et on s'arrête à 8 tuiles au total, soit AU PLUS 2 des 3) :
  - **(1) Meilleure série** — présente si au moins un segment > 0 (DEC-9, renommée).
  - **(2) Pic rang** — présente si au moins un pic de palier (LUSR et/ou CSR) sur le scope. UNE
    tuile, jusqu'à 2 lignes (une par système de rating), cf. DEC-PEAKRANK / DEC-PEAK.
  - **(3) Pic MMR** — présente si au moins un `team_mmr` sur le scope. Priorité la plus basse
    (métrique brute + soumise au masquage MMR).
  Le Classement N'EST PLUS une tuile (DEC-LAYOUT). Justification ≤ 8 : 6 base + au plus 2
  conditionnelles = 8 max ; jamais de trou (DEC-GRID). En low_sample : seules 1-4
  (Matchs/WR/FDA/Perf) ; Durée/Pic FDA/conditionnelles omises.
- **DEC-PERF (Perf colorée).** Valeur de la tuile Perf colorée `style={{ color:
  getPerfColor(score) }}` (`lib/perf-color.ts:27-29`, retourne déjà une CSS var — ne PAS
  l'envelopper dans `tokenCssVar`). Tokens sémantiques (perf-tier-*).
- **DEC-9 (« Séries » → « Meilleure série »).** Renommer `streaks_title` FR « Meilleure
  série » / EN « Best streak ». Contenu de la tuile inchangé.
- **DEC-TOOLTIP (portal).** Rendre le panneau `InfoTooltip` via `createPortal` vers
  `document.body`, position `fixed` calculée depuis `getBoundingClientRect()` du bouton, dans
  `info-tooltip.tsx` UNIQUEMENT (corrige tous les consommateurs, ne touche pas `KpiCard`).
  Fermeture au scroll/resize/clic extérieur/blur ; `role="tooltip"` + aria conservés ; pas de
  régression des ~10 consommateurs. Tests prévus.
- **DEC-GRID (pleine largeur adaptative).** Les rangées (socle 4-8 tuiles ; « Par… » 3-4
  cellules) occupent toute la largeur sans trou en fin de rangée. Conception (DEC-tech
  ci-dessous) : socle = `auto-fit`/`minmax` (déjà en place, étire les tuiles) ; « Par… » =
  `auto-fit`/`minmax` pour absorber la variabilité (0-3 dimensions + colonne « contexte +
  Classement »).
- **DEC-LAYOUT (Classement en 4ᵉ colonne scindée).** Le Classement QUITTE le socle et devient
  un bloc compact sous « Par contexte » : la 4ᵉ colonne de la rangée « Par… » empile « Par
  contexte » (haut) + « Classement » (bas), hauteurs harmonisées avec les cartes de dimension.
  Réutiliser `BriefingSectionCard` pour le bloc Classement.
- **DEC-MVP (MVP/LVP dans le tableau — À FAIRE).** Surligner, sur TOUT le scope chargé côté
  client (page_size 10000, tri/agrégat client déjà en place), le meilleur ET le pire par
  colonne clé (au minimum **Perf, FDA, Frags, Score**). RÉUTILISER le style best/worst de
  `MatchScoreboard.logic` (`cellState`/`cellStyle`), sur le patron éprouvé de
  `LeaderboardBlock.highlight.ts` (`HL_INVERTED` + `computeColumnExtremes` +
  `columnHighlightStyle`). Surlignage calculé sur le scope complet (pas la page visible),
  indépendant du tri ; égalités = plusieurs cellules surlignées (comportement natif du
  helper) ; valeurs nulles = neutre. Purement frontend.
- **DEC-10 (reformulation `tip_dimensions`).** Retenu : FR « Où vous performez le mieux et le
  moins bien selon la carte, le mode ou la sélection. » (EN en parité). Repartir de l'état
  courant du fichier (édition utilisateur, cf. 2.1) — cette reformulation la supersède
  INTENTIONNELLEMENT (décision utilisateur ; ne pas écraser sans lire = lecture faite §2.1).
- **DEC-PAD (padding réduit).** `th` (`HEADER_TH_CLASS:123-124`) et `td` (`:757`) passent de
  `px-2 py-1.5` à `px-2 py-1` (proposition ; l'exécutant peut affiner `px-1.5 py-1` en revue
  visuelle). Un seul point pour `th`, un pour `td`.

### Décisions techniques (architecte)

- **DEC-RANK-BE (Classement — architecture backend, algo pur).** Approche retenue (vérifiée
  faisable sur pièces §2.2/2.3) :
  1. **Exposer 3 colonnes sur la raw row** via `Q5PlayerSkillRankHistoryTpl` (skill,
     player-DB) : `playlist_group`, `rating_value`, `rating_delta` (déjà dans
     `match_skill_rank_latest`). Ajouter les 3 champs à `MatchHistoryRawRow`
     (`PlaylistGroup *string`, `RatingValue *float64`, `RatingDelta *float64`), étendre le
     `SELECT`, le scan et l'assignation de `mergeHistorySkillRanks`.
  2. **Nouvel algo pur** `internal/analysis/rank_progression.go` :
     `ComputeRankProgressionByChain(samples []RankChainSample) []RankChainProgression` où
     `RankChainSample` = { RatingType, PlaylistGroup, StartTime, TierLabel *string,
     RatingValue *float64, RatingDelta *float64, PlacementDone/PlacementTotal *int } et
     `RankChainProgression` = { RatingType, PlaylistGroup, Matches, TierStartLabel,
     TierEndLabel, TierStartIsPlacement, TierEndPlacementRemaining, DeltaPerMatch }. Algo :
     grouper par `(RatingType, PlaylistGroup)` ; par chaîne, trier chronologiquement, résoudre
     premier/dernier palier + flags placement (logique `firstTieredRow`/`applyTierStart`/
     `applyTierEnd` DÉPLACÉE ici, rendue pure), et **`DeltaPerMatch = (RatingValue du dernier
     match noté de la chaîne − RatingValue du premier) / count`** (variation nette du rating de
     LA CHAÎNE, ramenée au match). Ordre déterministe : type majoritaire d'abord (count total
     desc, tie CSR), puis chaînes du type par count desc (tie clé de chaîne). Title-agnostic,
     testable seul (`arch-rules`).
     - *Note pt/match (DÉFAUT ferme).* `(rating_value_fin − rating_value_début)/count` PAR
       CHAÎNE est **garanti co-signé avec la progression de paliers** de la même chaîne (les
       paliers sont monotones dans le rating) → supprime l'incohérence RAPPORTÉE (cross-chaînes)
       ET l'incohérence intra-chaîne résiduelle possible avec une somme de deltas sous filtrage
       non contigu. Il honore l'instruction « exposer rating_value ». Unité = points de rating
       natifs du type (mu pour LUSR — petites valeurs, le formatage existant `formatSignedFixed`
       les gère). Bords : chaîne à 1 seul match noté → `RatingDelta` de ce match s'il existe,
       sinon 0. *Alternative documentée* (si l'utilisateur préfère conserver la sémantique
       « moyenne de delta par match » exacte de V3) : `DeltaPerMatch = Σ(RatingDelta de la
       chaîne)/count` (utilise `rating_delta`, également per-chaîne, corrige aussi le bug
       cross-chaînes mais SANS la garantie de signe). Les deux nécessitent la segmentation par
       chaîne ; le DÉFAUT reste la variation nette.
  3. **DTO segmenté par chaîne** : ajouter `PlaylistGroup string` à
     `ExplorerBriefingRankedKind` ; chaque entrée = une chaîne `(type, playlist_group)`. Le
     champ `Kinds` reste (liste plate, une ligne par chaîne — le front rend une ligne par
     entrée).
  4. **Service** `buildBriefingRanked` : adapte `filtered` (raw rows) → `[]RankChainSample`,
     appelle l'algo, mappe → `[]ExplorerBriefingRankedKind`. **Ne consomme PLUS
     `scopedKPIs.RankDeltas`** → signature révisée (`scopedKPIs` retiré si plus utilisé par
     le module). `slog.DebugContext` si une chaîne n'a aucun palier (best-effort documenté).
  5. **Purge `RankDeltas`** (0 code mort, CLAUDE.md §7) : après bascule, `KPIStats.RankDeltas`
     (`squad_v2.go:146`) + le bloc `kpi_stats.go:196-217` + les 2 tests
     (`kpi_stats_test.go` `_RankDeltas_*`) perdent leur unique consommateur → supprimer.
     Conserver `RankDelta` singulier (consommé par squad). **RE-VÉRIFIER le grep avant
     purge** (un consommateur inattendu = ne pas purger, consigner §6).
- **DEC-DURATION (Durée totale — serveur).** Ajouter `r.duration_seconds` à `Q5SharedHistory`
  (dispo sur `v_match_full`) + `DurationSeconds *int` à `MatchHistoryRawRow` ; sommer dans
  `buildBriefingScope` → nouveau champ `TotalDurationSeconds *int` sur `ExplorerBriefingScope`
  (nil si aucune durée). Front : formateur « h min » — RÉUTILISER un formateur existant de
  `lib/formatters/duration.ts` s'il en existe un heures+minutes ; sinon en ajouter un
  (`formatDurationHM`, ex. 2530 s → « 42 min », 152 400 s → « 42 h 20 »). NE PAS réutiliser
  `formatDurationMMSS` (MM:SS, inadapté aux totaux longs).
- **DEC-PEAK (Pic FDA / Pic MMR / Pic rang — serveur).** Agrégats du scope sur les raw rows,
  posés sur `ExplorerBriefingScope` (tous `omitempty`) :
  - `PeakKDA *float64` (max `r.KDA`) — toujours calculé.
  - `PeakTeamMMR *float64` (max `r.TeamMMR`) — calculé si au moins une row porte `team_mmr`.
  - `PeakRanks []ExplorerBriefingPeakRank` (0, 1 ou 2 entrées ; type neuf
    `{ RatingType string; TierLabel string }`) — meilleur palier ATTEINT PAR SYSTÈME de rating
    (cf. DEC-PEAKRANK). Pour chaque `rating_type` présent : argmax des rows notées par
    `(analysis.CSRTierOrdinal(SkillTier), SubTier)` → `TierLabel` = `SkillTierLabel` de la row
    gagnante. Réutilise l'ordinal canonique `analysis.CSRTierOrdinal` (§2.3) — aucune nouvelle
    table de paliers. Ordre déterministe (ex. LUSR puis CSR). nil/omitempty si aucun palier.
- **DEC-PEAKRANK (Pic rang — RETENU, par système, cascade).** Décision utilisateur 2026-07-17 :
  **Pic rang RETENU**. Justification : le module Classement montre le palier de DÉPART →
  ARRIVÉE ; or on peut CULMINER plus haut que le palier final entre les deux → le « Pic rang » =
  le palier le plus élevé ATTEINT sur le scope, information NON redondante avec la progression
  début→fin (l'argument « redondant » de la 1ʳᵉ rédaction est infirmé). **Par système de rating,
  les deux si présents** : UNE tuile « Pic rang » affichant jusqu'à 2 lignes (ex. valeur « LUSR
  Diamant IV », sous-ligne « CSR Onyx »). Pas de comparaison inter-système — on liste le meilleur
  palier de CHAQUE système. Le pic d'un système = max de l'ordinal `(CSRTierOrdinal(tier),
  sub_tier)` parmi ses matchs du scope (le ladder Bronze→Onyx est nominalement le même quelle que
  soit la chaîne → un « meilleur badge atteint » a du sens sans comparer les chaînes).
  **Faisabilité VÉRIFIÉE sur pièces** (§2.3) : l'ordinal serveur EXISTE
  (`analysis.CSRTierOrdinal(tierEN string) int`, `home_canonical_skill.go:181`, testé, couvre
  Bronze→Onyx (+Champion), réutilisé par le pic de la home `home_repo_skill_peak.go:408`) ;
  `SkillTier` (nom EN) est déjà sur la raw row ; `sub_tier` est dans `match_skill_rank_latest` →
  à exposer sur la raw row (Phase 2). DTO : `PeakRanks` (DEC-PEAK). **Règle de cascade ≤ 8**
  (DEC-TILES) : les 3 conditionnelles sont rendues par priorité (1) Meilleure série, (2) Pic
  rang, (3) Pic MMR, cap à 8 → au plus 2 des 3 s'affichent. **Pic MMR NON supprimée** — juste la
  priorité la plus basse (métrique brute + masquée). Ainsi jamais > 8, jamais de trou en fin de
  rangée.
- **DEC-RANK-FE (Classement front — bloc 4ᵉ colonne + labels de chaîne).** Le Classement
  devient un `BriefingSectionCard` (titre « Classement » + `InfoTooltip` `tip_ranked`)
  contenant un `<ul>` d'une ligne par chaîne : « {TYPE maj}{ · label chaîne si le type a ≥ 2
  chaînes} · {paliers début → fin} · {±x pt/match coloré} ». Label de chaîne via
  **`lusrChainLabel(playlist_group, locale)`** (`features/career/lusr-chains.ts`). Rendu dans
  la 4ᵉ colonne (empilé sous `ContextSplitCard`) par `ExplorerBriefingModules`, gaté
  `useCapability('ranked')` (le hook migre du Strip vers Modules) + `briefing.ranked` présent.
  `RankedTile` (socle) SUPPRIMÉE. *Import cross-feature* `lusrChainLabel` (career→explorer) :
  DÉFAUT = importer tel quel (fonction pure de 3 lignes, clés du manifest career) ; si
  l'exécutant juge l'anti-pattern gênant, promouvoir `lusrChainLabel` vers un emplacement
  partagé `lib/i18n/` (miroir de SPARK-1 V3) — décision d'exécution à consigner §6.
- **DEC-MVP-FE (réutilisation du style + réduction du god-file).** Extraire la logique de
  surlignage MVP/LVP dans un sibling `ExplorerMatchesTable.highlight.ts` (calqué sur
  `LeaderboardBlock.highlight.ts`) qui **importe** `cellState`/`cellStyle` de
  `@/features/match-view/MatchScoreboard.logic` (import cross-feature déjà précédent via
  leaderboard) — jamais recopier le style. Colonnes surlignées + sens (`INVERTED` pour
  `deaths` : petit = meilleur). Cette extraction RÉDUIT le god-file (800 L) ; l'extraction
  complémentaire des définitions de colonnes est traitée en Phase finale (DEC-DEBT) si elle
  reste sûre et scopée.
- **DEC-TILE-TECH (structure des tuiles).** `BriefingTile` conserve `KpiCard` + label + `info`
  + valeur + sous-texte. Le slot `chart?` est RETIRÉ (plus de sparkline) si aucun consommateur
  ne subsiste. La tuile WR n'utilise plus `chart` mais compose `OutcomeBar` + V-D-N dans son
  `value`/sous-texte (ou un layout dédié) — l'exécutant choisit le placement le plus lisible
  en restant compact (revue visuelle).
- **DEC-i18n (parité).** Toute clé ajoutée/modifiée l'est en FR ET EN dans `explorer.toml`,
  suivie de `node apps/web/scripts/build_i18n_manifests.mjs` AVANT `make check-types`. Toute
  clé orphelinée par une phase est purgée DANS cette phase (grep de clôture = 0). Clés
  neuves/modifiées prévues :
  - `duration_total_label` FR « Durée totale » / EN « Total time » ; `tip_duration` (FR+EN).
  - `peak_fda_label` FR « Pic FDA » / EN « Peak FDA » ; `tip_peak_fda` (FR+EN).
  - `peak_rank_label` FR « Pic rang » / EN « Peak rank » ; `tip_peak_rank` (FR+EN — le tip
    précise « le meilleur palier atteint sur la période, PAS le rang final »).
  - `peak_mmr_label` FR « Pic MMR » / EN « Peak MMR » ; `tip_peak_mmr` (FR+EN) — priorité la
    plus basse de la cascade (DEC-PEAKRANK).
  - `streaks_title` → « Meilleure série » / « Best streak » (DEC-9).
  - `tip_win_rate` → RÉVISER (retirer la mention « mini-courbe » de la sparkline ; décrire le
    ruban V-D-N — DEC-SPARK/DEC-TILES).
  - `tip_dimensions` → DEC-10.
  - `record_label` → PURGER (morte, DEC-DEBT).
  - Labels de chaîne : réutiliser `career.lusr.chain.*` (pas de clé neuve).

---

## 4. Périmètre

**Dans le périmètre :**
- Backend `apps/go-api` :
  - Classement par chaîne (DEC-RANK-BE) : `queries_career.go` (Q5 skill : +3 colonnes),
    `match_history_repo.go` (scan/assignation), `domain/match_history.go` (+3 champs raw row),
    `analysis/rank_progression.go` (neuf, pur), `domain/explorer_briefing.go`
    (`ExplorerBriefingRankedKind` +`PlaylistGroup`), `service/match_history_service_briefing_
    ranked.go` (réécriture), purge `RankDeltas` (`analysis/kpi_stats.go`, `domain/squad_v2.go`,
    tests). Tests analysis + service.
  - Purge trend (DEC-SPARK) : `domain/explorer_briefing.go`, `service/match_history_service_
    briefing.go` (`buildBriefingTrend` + consts), tests, OpenAPI, `types.ts`.
  - Agrégats socle (DEC-DURATION/DEC-PEAK) : `queries_career.go` (Q5 shared : +duration ; Q5
    skill : +sub_tier), `match_history_repo.go` (scan), `domain/match_history.go`
    (+DurationSeconds, +SubTier), `domain/explorer_briefing.go` (Scope +TotalDurationSeconds/
    PeakKDA/PeakTeamMMR/PeakRanks + type `ExplorerBriefingPeakRank`), `service/match_history_
    service_briefing.go` (`buildBriefingScope`, réutilise `analysis.CSRTierOrdinal`), tests,
    OpenAPI, `types.ts`.
- Frontend `apps/web` : `ExplorerBriefingStrip.tsx`, `ExplorerBriefingTiles.tsx`,
  `BriefingTile.tsx`, `ExplorerBriefingModules.tsx`, nouveau `ExplorerMatchesTable.highlight.ts`,
  `ExplorerMatchesTable.tsx` (MVP/LVP + padding + éventuel split), `info-tooltip.tsx` (portal),
  réutilisation de `OutcomeBar`/`getPerfColor`/`lusrChainLabel`/`MatchScoreboard.logic`,
  manifest `explorer.toml` (+ régénération), tests (`*.test.tsx`).
- Vérification (reprise UTILISATEUR), journal du plan, `.ai/thought_log.md`, changelog
  (`docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`).
- Phase 0 : commit du déplacement des plans vers `.ai/V7/` + préservation de l'édition
  utilisateur `tip_dimensions`.

**Hors périmètre (noter en §6 Découvertes si rencontré, NE PAS traiter) :**
- Tout changement des CALCULS baseline/dimensions/context/streaks/dominance (inchangés).
- La primitive `Sparkline` (conservée) et ses consommateurs admin.
- La duplication inline du style best/worst dans `SquadImpactScoreboard.tsx` (dette
  préexistante — signalée §6, non corrigée ici sauf décision explicite).
- Le masquage MMR global (chantier séparé, cf. mémoire parité H5) — ce plan ne fait
  qu'omettre la tuile Pic MMR quand `team_mmr` est absent.
- Dette lint pré-existante (baseline gelée) ; tout Python (interdit) ; SQLite (interdit).
- Refonte du tableau au-delà de MVP/LVP + padding + split god-file scopé.

**`-tags=integration` NON requis (justification).** Le fix Classement et les agrégats socle
sont en LECTURE seule : ajout de colonnes déjà présentes dans `match_skill_rank_latest` et
`v_match_full` au SELECT de deux requêtes de lecture, nouveaux champs DTO, agrégation en
mémoire. **Aucune écriture, aucun writer/lease, aucune table créée, aucune migration de
schéma.** Les tests anti-ART (`-tags=integration`) couvrent les écritures per-match — sans
objet ici. La suite standard `cd apps/go-api && go test ./...` (inclut `platform/duckdb` en
`:memory:` + CGO) + `make go-api-lint` + `TestOpenAPISchemaDrift` suffit (règle CLAUDE.md :
`-tags=integration` OBLIGATOIRE seulement avant livraison sync/persist). Si l'exécution
constate qu'un test repo touché est en réalité gardé par le tag integration, le noter §6 et
l'exécuter — mais aucune écriture n'est planifiée.

---

## 5. Phases (ordre strict — une étape CLOSE avant la suivante)

> Clôture d'étape = gate passé (commandes exactes, sorties propres — jamais de test
> skippé/désactivé) + tous les items statués `[x]` fait / `[~]` couvert ailleurs (réf) /
> `[!]` non traité (justif écrite) + plan mis à jour (cases + journal) + entrée
> `.ai/thought_log.md` + point d'étape utilisateur. Aucune case vide à la clôture. Zéro fix
> hors périmètre (→ §6).
>
> Notes d'exécution : commandes `go` SÉQUENTIELLES (jamais concurrentes — cache Windows).
> Après toute édition de `explorer.toml` : `node apps/web/scripts/build_i18n_manifests.mjs`
> AVANT `make check-types`. vitest → `dangerouslyDisableSandbox=true`. Purger
> `apps/web/node_modules/.tmp` avant `make check-types`.

### Phase 0 — Cadrage, commit du move & re-vérification (rapide)

- [x] Confirmer `git branch --show-current` = `feat/explorer-briefing-compact`. Worktree :
      ne toucher que ce qui est autorisé ci-dessous. FAIT : branche OK ; `git status` =
      seuls `explorer.toml` + `generated/explorer.ts` modifiés (édition utilisateur intacte).
- [x] **Committer le déplacement des plans** vers `.ai/V7/` : `git add .ai/PLAN_EXPLORER_
      BRIEFING_V2_2026-07.md .ai/PLAN_EXPLORER_BRIEFING_V3_COMPACT_2026-07.md .ai/V7/
      PLAN_EXPLORER_BRIEFING_V2_2026-07.md .ai/V7/PLAN_EXPLORER_BRIEFING_V3_COMPACT_2026-07.md`
      (le `git mv` équivalent : suppression + ajout) + le présent plan V4, avec message
      `docs(explorer-briefing): déplace les plans V2/V3 sous .ai/V7 + ajoute plan V4`.
      **Ne PAS committer** l'édition `explorer.toml`/`generated/explorer.ts` (édition
      utilisateur en cours) : la LAISSER dans le worktree (elle sera intégrée au commit de la
      phase reformulation). Vérifier `git status` après : seuls `explorer.toml` +
      `generated/explorer.ts` restent modifiés. FAIT PAR LE SUPERVISEUR : commit
      `52cf17a4c` (`git log --oneline -1` le confirme) ; worktree vérifié = seuls les 2
      fichiers i18n modifiés.
- [x] Re-vérifier §2 sur pièces (rouvrir chaque fichier:ligne cité) ; consigner tout décalage
      en §6. FAIT : tous les ancrages §2.2/2.3 concordent (DTO explorer_briefing.go,
      service ranked + briefing, kpi_stats, raw row match_history.go, Q5 shared/skill,
      repo scan/merge, SkillSnapshot, CSRTierOrdinal:181, RankDeltas 1 seul consommateur
      prod). 2 décalages mineurs consignés → Découvertes-9 et 10.
- [x] Confirmer les points à trancher : DEC-PEAKRANK (Pic MMR conditionnel — défaut appliqué
      si l'utilisateur ne s'est pas prononcé) ; DEC-RANK-BE note pt/match (défaut FERME =
      variation nette `(rating_value_fin − rating_value_début)/count`, co-signée avec la
      progression — cf. corps DEC-RANK-BE ; `Σ(rating_delta)/count` = alternative écartée).
      Consigner au journal. FAIT : les deux défauts sont CONFIRMÉS par la mission superviseur
      (points tranchés co-signés) et appliqués tels quels. Journal mis à jour.

Gate Phase 0 : branche correcte ; commit du move fait (`git log --oneline -1` = le commit) ;
worktree ne contient plus que l'édition `tip_dimensions` ; constat re-vérifié. Pas de gate de
build (aucun code applicatif modifié).

### Phase 1 — Backend : Classement par chaîne (lourd) — DEC-1 / DEC-RANK-BE

- [x] **1a (raw row + repo).** Ajouter `PlaylistGroup *string`, `RatingValue *float64`,
      `RatingDelta *float64` à `MatchHistoryRawRow` (`domain/match_history.go`). Étendre
      `Q5PlayerSkillRankHistoryTpl` (`queries_career.go:99-108`) : SELECT `playlist_group,
      rating_value, rating_delta` (+ colonnes existantes). Étendre le struct `skill` + le
      `Scan` + l'assignation de `mergeHistorySkillRanks` (`match_history_repo.go:193-227`).
- [x] **1b (analysis pur).** Créer `internal/analysis/rank_progression.go` :
      `RankChainSample` (dont `RatingValue *float64` + `RatingDelta *float64`),
      `RankChainProgression`, `ComputeRankProgressionByChain(samples)` (groupement par
      `(RatingType, PlaylistGroup)` ; tri chronologique ; premier/dernier palier + flags
      placement ; `DeltaPerMatch = (RatingValue du dernier match noté − RatingValue du premier)
      / count` par chaîne — DÉFAUT DEC-RANK-BE, garanti co-signé ; bord 1 match → `RatingDelta`
      de ce match ou 0 ; ordre déterministe type-majoritaire puis chaîne par count). Déplacer/
      rendre pures les helpers `firstTieredRow`/`lastTieredRow`/`isPlacementRow`/
      `applyTierStart`/`applyTierEnd` depuis le service. Fichier ≤ 500 L, fonctions ≤ 80 L.
- [x] **1c (tests analysis).** `rank_progression_test.go` : mono-chaîne (progression + pt/match
      cohérent) ; multi-chaînes LUSR (une entrée par chaîne, jamais croisées) ; CSR chaîne
      unique « ranked » (une entrée) ; début en placement / fin en placement ; chaîne sans
      palier (progression omise, pt/match présent) ; ordre déterministe. Aucun test skippé.
- [x] **1d (DTO).** `ExplorerBriefingRankedKind` (`explorer_briefing.go`) : ajouter
      `PlaylistGroup string json:"playlist_group,omitempty"` + commentaire (une entrée = une
      chaîne `(type, playlist_group)`).
- [x] **1e (service).** Réécrire `buildBriefingRanked` (`match_history_service_briefing_
      ranked.go`) : adapter `filtered` → `[]RankChainSample`, appeler l'algo, mapper →
      `[]ExplorerBriefingRankedKind` (+`PlaylistGroup`). Retirer la dépendance à
      `scopedKPIs.RankDeltas` ; ajuster la signature (`buildExplorerBriefing` `:76-78` en
      conséquence). `slog.DebugContext` si une chaîne n'a aucun palier. Gate `rankedCapable`
      conservé.
- [x] **1f (purge `RankDeltas`).** RE-VÉRIFIER le grep `RankDeltas` (0 consommateur hors
      briefing/tests). Supprimer : bloc `kpi_stats.go:196-217`, champ `KPIStats.RankDeltas`
      (`squad_v2.go:146`), tests `kpi_stats_test.go` `_RankDeltas_*`. Conserver `RankDelta`
      singulier. (Si un lecteur inattendu → ne pas purger, consigner §6.)
- [x] **1g (tests service).** Mettre à jour `match_history_service_briefing_ranked_test.go` /
      `..._test.go` : entrées `RankDeltas` remplacées par des raw rows portant `PlaylistGroup`,
      `RatingValue` (+ `RatingDelta`, `SkillTierLabel`, `SkillRatingType`, `StartTime`) ;
      asserts multi-chaînes (LUSR 2 chaînes → 2 entrées ; CSR → 1) ; pt/match par chaîne
      co-signé avec la progression. Aucun test skippé.
- [x] **1h (OpenAPI + regen).** `api/openapi.yaml` : `ExplorerBriefingRankedKind` +
      `playlist_group`. `make generate-types` ; `types.ts`/`generated.ts` régénérés ;
      `TestOpenAPISchemaDrift` vert.

Gate Phase 1 : `cd apps/go-api && go test ./...` = 0 (SÉQUENTIEL ; tests 1c/1g inclus) ;
`make go-api-lint` = 0 ; `make generate-types` idempotent (re-run → 0 diff) ;
`TestOpenAPISchemaDrift` = 0 MISSING/DIVERGENT sur `ExplorerBriefingRanked*` ;
`make check-types` = 0 ; grep de clôture : 0 `RankDeltas` sous `apps/go-api` (hors historique).

> **GATE PHASE 1 — VERT (exécuté le 2026-07-17).** `CGO_ENABLED=1 go test ./...` = exit 0
> (111 packages ok, 0 FAIL). `go vet ./internal/domain/... ./internal/analysis/...` = exit 0.
> `npm run generate-types` idempotent (diff = 1 ligne `playlist_group?: string;`, stable au
> 2e run). `TestOpenAPISchemaDrift` = PASS, 0 MISSING ; `ExplorerBriefingRankedKind` +
> `KPIStats` ABSENTS de la liste des 26 divergents (baseline pré-existant, hors périmètre) —
> mes éditions les ont alignés (KPIStats était divergent avant la purge, désormais aligné).
> `npm run typecheck` (check-types, cache `.tmp` purgé) = 0 erreur. Grep `RankDeltas` sous
> `apps/go-api` = 0. Réalisé : 1a raw row (+3 champs) + Q5 skill (+3 col) + scan/merge ;
> 1b `analysis/rank_progression.go` (algo pur variation nette par chaîne, DEC-RANK-BE) ;
> 1c 10 tests algo pur ; 1d DTO `+PlaylistGroup` ; 1e service réécrit (samples → algo,
> signature `buildExplorerBriefing` sans `scopedKPIs`, caller MAJ) ; 1f purge `RankDeltas`
> (kpi_stats bloc + import `sort` + champ squad_v2 + 2 tests) + retrait const
> `minRankedKindMatches` (Découverte-11) ; 1g tests service réécrits per-chaîne (+ fix 3
> call-sites context/streaks) ; 1h openapi + regen. AUCUN commit (superviseur en fin de bloc).

### Phase 2 — Backend : purge trend + agrégats socle (moyen) — DEC-SPARK / DEC-DURATION / DEC-PEAK

- [x] **2a (purge trend — domain).** Supprimer `ExplorerBriefingTrend`,
      `ExplorerBriefingTrendPoint`, le champ `Trend` (`explorer_briefing.go:39,105-118`).
      Corriger les commentaires de package/`LowSample` mentionnant la tendance.
- [x] **2b (purge trend — service).** Supprimer `buildBriefingTrend`
      (`match_history_service_briefing.go:294-351`), l'appel (`:75`), les consts
      `minTrendMatches`/`minTrendSpanDays` (`:46-47`), les imports devenus morts (`temporal`
      si plus utilisé — VÉRIFIER). Supprimer/adapter les tests trend. Grep : 0
      `buildBriefingTrend`/`ExplorerBriefingTrend` restant.
- [x] **2c (colonnes raw row — repo).** (i) **Durée** : ajouter `r.duration_seconds` à
      `Q5SharedHistory` (`queries_career.go:61-96`, dispo sur `v_match_full`) + `DurationSeconds
      *int` à `MatchHistoryRawRow` + scan (requête shared) ; MAJ commentaire « 25 colonnes ».
      (ii) **Sous-palier (pour Pic rang)** : ajouter `sub_tier` à `Q5PlayerSkillRankHistoryTpl`
      (additif, à côté des colonnes Phase 1) + `SubTier *int` à `MatchHistoryRawRow` + scan/
      assignation (`mergeHistorySkillRanks`).
- [x] **2d (agrégats socle — DTO + service).** `ExplorerBriefingScope` : +`TotalDurationSeconds
      *int`, `PeakKDA *float64`, `PeakTeamMMR *float64`, `PeakRanks []ExplorerBriefingPeakRank`
      (type neuf `{ RatingType string; TierLabel string }`), tous `omitempty`.
      `buildBriefingScope` (`match_history_service_briefing.go:107-124` + `aggregateRawStats`) :
      somme des durées, max KDA, max TeamMMR ; **PeakRanks** = par `rating_type` présent, argmax
      des rows notées par `(analysis.CSRTierOrdinal(SkillTier), SubTier)` → `TierLabel` =
      `SkillTierLabel` gagnant (0/1/2 entrées, ordre déterministe LUSR puis CSR). Tests service :
      durée totale, pic FDA, pic MMR, **Pic rang** (LUSR seul ; LUSR+CSR = 2 entrées ; nil si
      aucun palier).
- [x] **2e (OpenAPI + regen).** `api/openapi.yaml` : retirer `ExplorerBriefingTrend`/
      `...TrendPoint`/`trend` ; ajouter `total_duration_seconds`/`peak_kda`/`peak_team_mmr`/
      `peak_ranks` (+ schéma `ExplorerBriefingPeakRank`) à `ExplorerBriefingScope`. `make
      generate-types` ; `TestOpenAPISchemaDrift` vert ; `types.ts`/`generated.ts` régénérés.

Gate Phase 2 : `cd apps/go-api && go test ./...` = 0 (SÉQUENTIEL) ; `make go-api-lint` = 0 ;
`make generate-types` idempotent ; `TestOpenAPISchemaDrift` vert ; `make check-types` = 0 ;
greps : 0 `Trend`/`buildBriefingTrend` (frise trend) sous `apps/go-api` ; champs Scope présents.

> **GATE PHASE 2 — VERT (exécuté le 2026-07-17).** `CGO_ENABLED=1 go test ./...` = exit 0
> (111 packages ok, 0 FAIL). `go vet ./internal/domain/... ./internal/analysis/...` = exit 0.
> `npm run generate-types` STRICTEMENT idempotent (md5 identique après re-run). `TestOpenAPI
> SchemaDrift` = PASS, **0 MISSING** ; émission divergents filtrée = **0 ExplorerBriefing***
> (Scope aligné via émission Huma byte-exact ; `ExplorerBriefingPeakRank` ajouté ; ExplorerBriefing
> aligné par retrait du ref `trend` ; 26 divergents = baseline pré-existant intact). `npm run
> typecheck` (check-types, `.tmp` purgé) = 0 erreur. Greps : 0 `buildBriefingTrend`/`Explorer
> BriefingTrend`/`minTrend*`/`trendRow` sous `apps/go-api`. Réalisé : 2a purge trend domain
> (champ + 2 types + docs « frise/tendance ») ; 2b purge trend service (fonction + trendRow +
> consts + import `temporal` + tests trend) ; 2c raw row +`DurationSeconds`/`SubTier`, Q5 shared
> +`duration_seconds` (comment 25→31), Q5 skill +`sub_tier`, scan/merge ; 2d Scope
> +`TotalDurationSeconds`/`PeakKDA`/`PeakTeamMMR`/`PeakRanks` + type `ExplorerBriefingPeakRank`,
> helpers `sumScopeDurations`/`maxScopeFloat`/`scopePeakRanks` (réutilise `analysis.CSRTierOrdinal`),
> 5 tests ; 2e openapi (retrait Trend + Scope Huma-exact + PeakRank) + regen. **Couplage
> Découverte-9 résolu** : purge frontend minimale (types.ts −2 exports Trend ; Strip.tsx −import
> Sparkline/−trendValues/−bloc chart) — 0 consommateur `briefing.trend` restant. AUCUN commit
> (superviseur en fin de bloc). Reste au bloc frontend (Phases 3-4) : tuiles socle, OutcomeBar,
> Perf couleur, Pic FDA/rang/MMR + cascade, RankedBlock 4ᵉ colonne, export type PeakRank.

### Phase 3 — Frontend : refonte du socle des tuiles (moyen) — DEC-TILES / DEC-PERF / DEC-9 / DEC-SPARK

- [ ] **3a (i18n).** Ajouter/modifier dans `explorer.toml` : `duration_total_label`,
      `tip_duration`, `peak_fda_label`, `tip_peak_fda`, `peak_mmr_label`, `tip_peak_mmr` (si
      tuile retenue) ; renommer `streaks_title` (DEC-9) ; réviser `tip_win_rate` (retirer la
      mention sparkline, décrire le ruban V-D-N). FR + EN. Régénérer les manifests.
- [ ] **3b (tuile WR hero).** Refondre la tuile Taux de victoire du Strip : composer
      `OutcomeBar` (`@/components/ui/outcome-bar`, alimenté par `scope.wins/losses/ties/dnf`)
      + V-D-N flanquant (tokens `outcome-win`/`outcome-loss`) + tooltip des 4 issues (calqué
      `HomeHeroKPIGrid:107-119`) ; conserver le delta « vs habituel » (masqué en plein
      historique). RETIRER la sparkline (`:98-108`), l'import `Sparkline` (`:13`),
      `trendValues` (`:75`), la lecture `briefing.trend`.
- [ ] **3c (Perf colorée).** Colorer la valeur de la tuile Perf : `style={{ color:
      getPerfColor(perf) }}` (import `@/lib/perf-color`). Ne pas envelopper dans `tokenCssVar`.
- [ ] **3d (Durée totale + Pic FDA + Pic rang + Pic MMR + cascade).** Tuiles de base : Durée
      totale (formateur « h min », DEC-DURATION) + Pic FDA (`scope.peak_kda`, coloré
      `kdaNetColor`). Conditionnelles rendues par PRIORITÉ (**cascade** DEC-TILES, cap à 8, au
      plus 2 des 3) : (1) Meilleure série, (2) **Pic rang** (tuile lisant `scope.peak_ranks` :
      valeur = 1ʳᵉ entrée « {RatingType maj} {TierLabel} », 2ᵉ système en sous-ligne ; omise si
      vide), (3) Pic MMR (`scope.peak_team_mmr != null`). Implémenter la règle de cascade
      (collecter les conditionnelles PRÉSENTES dans l'ordre de priorité, en prendre AU PLUS 2).
      Chaque tuile avec son `InfoTooltip`. Labels COURTS.
- [ ] **3e (Meilleure série).** Le renommage est porté par `streaks_title` (3a) ; vérifier
      que `StreaksTile` lit toujours cette clé (aucune autre modif).
- [ ] **3f (BriefingTile).** Si le slot `chart?` n'a plus AUCUN consommateur après 3b, le
      retirer de `BriefingTile` (0 code mort).
- [ ] **3g (grille socle).** Vérifier/ajuster la grille socle (DEC-GRID : `auto-fit`/`minmax`
      déjà en place étire les tuiles ; ajuster le `minmax` si 7-8 tuiles créent un trou —
      revue visuelle). Documenter le choix.
- [ ] **3h (tests).** Mettre à jour `ExplorerBriefingStrip.test.tsx` : WR (OutcomeBar + V-D-N,
      plus de sparkline) ; Perf colorée ; Durée/Pic FDA présentes ; **Pic rang** (1 ligne LUSR ;
      2 lignes LUSR+CSR ; omise si vide) ; **règle de cascade** (3 conditionnelles présentes →
      au plus 2 rendues, ordre Meilleure série > Pic rang > Pic MMR, jamais > 8 tuiles) ;
      renommage Meilleure série. Aucun test skippé.

Gate Phase 3 : `node …/build_i18n_manifests.mjs` (diff = clés attendues) ; `make check-types`
= 0 ; `make test-web` vert ; `cd apps/web && npm run lint` = 0 erreur ; greps : 0 `Sparkline`
et 0 `trend` sous `features/explorer` ; `getPerfColor` importé dans le Strip.

### Phase 4 — Frontend : Classement en 4ᵉ colonne + grille « Par… » (moyen) — DEC-LAYOUT / DEC-RANK-FE / DEC-GRID

- [ ] **4a (bloc Classement).** Créer un `RankedBlock` (dans `ExplorerBriefingModules.tsx` ou
      un sibling) = `BriefingSectionCard` (titre `ranked_title` + `InfoTooltip` `tip_ranked`)
      rendant un `<ul>` d'une ligne par entrée de `ranked.kinds` : « {kind maj}{ · label
      chaîne si le type a ≥ 2 chaînes} · {progression début → fin} · {±x pt/match coloré
      `deltaToken`} ». Label de chaîne via `lusrChainLabel(k.playlist_group, locale)`.
      Réutiliser `rankedProgression`/placement (déplacés depuis `ExplorerBriefingTiles`).
- [ ] **4b (4ᵉ colonne scindée).** Dans la grille « Par… », rendre une 4ᵉ cellule =
      `flex flex-col gap-2` empilant `ContextSplitCard` (si présent) + `RankedBlock` (si
      `useCapability('ranked')` + `briefing.ranked`). Migrer le hook `useCapability('ranked')`
      du Strip vers Modules. Retirer `RankedTile` du Strip (`:184`) et de
      `ExplorerBriefingTiles` (supprimer `RankedTile` + helpers devenus morts ; conserver
      `StreaksTile`).
- [ ] **4c (grille pleine largeur).** Ajuster la grille « Par… » (DEC-GRID) pour absorber la
      variabilité (0-3 dimensions + colonne « contexte+Classement ») sans trou en fin de
      rangée : `auto-fit`/`minmax` recommandé (cellules qui s'étirent). Documenter le choix ;
      harmoniser les hauteurs (`h-full`/`items-start` selon rendu).
- [ ] **4d (i18n).** Aucune clé neuve attendue (labels de chaîne = `career.lusr.chain.*`
      réutilisés). Si `RankedTile` supprimée orpheline une clé, la purger.
- [ ] **4e (tests).** Mettre à jour les tests : Classement absent du socle ; présent dans la
      4ᵉ colonne quand `ranked` + capability ; une ligne par chaîne (LUSR multi = plusieurs
      lignes ; CSR = 1) ; capability off → bloc absent. Aucun test skippé.

Gate Phase 4 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ;
greps : 0 `RankedTile` dans le socle (Strip) ; `RankedBlock` rendu dans la grille « Par… » ;
`lusrChainLabel` importé.

### Phase 5 — Frontend : tooltips via portal (moyen) — DEC-TOOLTIP

- [ ] **5a (portal).** Réécrire le rendu du panneau d'`info-tooltip.tsx` : `createPortal(
      panel, document.body)` avec `position: fixed`, coordonnées calculées depuis
      `buttonRef.getBoundingClientRect()` (centré au-dessus du bouton, repli en dessous si
      trop haut). Conserver `role="tooltip"`, `aria-label`, la flèche, les tokens. Fermeture :
      clic extérieur (existant) + `scroll`/`resize` (listeners ajoutés à l'ouverture, nettoyés
      à la fermeture) + `blur`/`mouseleave`. NE PAS modifier `KpiCard`.
- [ ] **5b (tests).** Ajouter/mettre à jour un test : le panneau est rendu hors du conteneur
      `overflow-hidden` (portal → `document.body`) ; ouverture au clic/focus affiche le
      contenu ; fermeture au clic extérieur ; `role="tooltip"` présent. Vérifier la
      non-régression d'un consommateur existant (ex. un test qui monte `InfoTooltip`). Aucun
      test skippé.

Gate Phase 5 : `make check-types` = 0 ; `make test-web` vert (tests 5b + non-régression des
~10 consommateurs) ; `npm run lint` = 0 erreur ; grep : `createPortal` présent dans
`info-tooltip.tsx` ; `git status` de `KpiCard.tsx` = vide (0 modif).

### Phase 6 — Frontend : MVP/LVP dans le tableau (moyen) — DEC-MVP / DEC-MVP-FE

- [ ] **6a (helper highlight).** Créer `ExplorerMatchesTable.highlight.ts` (sibling, calqué
      `LeaderboardBlock.highlight.ts`) : `import { cellState, cellStyle } from
      '@/features/match-view/MatchScoreboard.logic'` ; `EXPLORER_INVERTED` (au moins `deaths`
      inversé) ; `computeColumnExtremes(rows, keys)` (garde ≥ 2) ; `columnHighlightStyle(key,
      value, extremes)`. Colonnes surlignées : Perf, FDA, Frags (kills), Score (+ deaths). NE
      PAS recopier `cellStyle` (import obligatoire — CLAUDE.md §6).
- [ ] **6b (câblage table).** Calculer les extrêmes sur TOUT le scope chargé (les `data` de la
      table, pas la page visible) — mémoïsé (`useMemo`) et indépendant du tri. Appliquer
      `columnHighlightStyle` en `style` sur le `<td>`/le contenu des colonnes surlignées.
      Gérer les valeurs nulles (neutre, natif) et les égalités (multi-surlignage, natif).
- [ ] **6c (tests).** Test de rendu : meilleur/pire d'une colonne surlignés sur un dataset
      hétérogène (≥ 2 valeurs) ; colonne uniforme non surlignée ; null neutre ; surlignage
      indépendant du tri. Aucun test skippé.

Gate Phase 6 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ;
grep : 0 littéral `color-mix(in oklab` NOUVEAU sous `features/explorer` (style importé, pas
recopié) ; `MatchScoreboard.logic` importé par le sibling highlight.

### Phase 7 — Padding + reformulation (rapide) — DEC-PAD / DEC-10

- [ ] **7a (padding).** `ExplorerMatchesTable.tsx` : `HEADER_TH_CLASS` (`:123-124`) et la
      className `td` (`:757`) → `py-1.5` réduit à `py-1` (ou `px-1.5 py-1`, à valider en revue
      visuelle). Un seul point pour `th`, un pour `td`.
- [ ] **7b (reformulation).** `explorer.toml` `tip_dimensions` → texte DEC-10 (FR « Où vous
      performez le mieux et le moins bien selon la carte, le mode ou la sélection. » + EN en
      parité). Régénérer les manifests. **Ce commit intègre aussi l'édition utilisateur en
      cours** du worktree (le fichier était déjà modifié — le nouveau texte la remplace
      proprement).

Gate Phase 7 : `node …/build_i18n_manifests.mjs` (diff = `tip_dimensions`) ; `make check-types`
= 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ; revue visuelle du padding reprise par
l'utilisateur.

### Phase 8 — Dettes préexistantes + clôture (moyen) — DEC-DEBT

> Périmètre FERMÉ et SÛR. Pour chaque dette : soit corriger (scopé), soit lister « NON traitée
> (justification) ». Ne PAS transformer en refactor illimité.

- [ ] **8a (`record_label` morte).** RE-VÉRIFIER le grep (0 lecteur composant) puis purger
      `explorer.briefing.record_label` de `explorer.toml` + régénérer. (Si un lecteur apparaît
      → ne pas purger, consigner §6.)
- [ ] **8b (eslint-disable inutile).** RE-VÉRIFIER sur pièces dans `ExplorerPage.tsx` quel
      `eslint-disable` (le cas échéant) est signalé « unused » par le lint baseline ; le
      retirer s'il est réellement inutile. Sinon statuer `[!]` (aucun inutile trouvé).
- [ ] **8c (god-file `ExplorerMatchesTable` 800 L).** Extraire les définitions de colonnes
      (et/ou les mappings `DOMINANCE_*`, les helpers de format) vers un sibling
      (`ExplorerMatchesTable.columns.tsx`) pour repasser SOUS 500 L, SI l'extraction reste
      sûre et scopée (aucun changement de comportement). Sinon : statuer `[!]` et **signaler
      comme chantier à part** (refonte du tableau) — ne pas forcer un split risqué en fin de
      chantier.
- [ ] **8d (dette matrice d'impact — signalement).** Consigner §6 la duplication inline du
      style best/worst dans `SquadImpactScoreboard.tsx` (2ᵉ copie) : NON traitée ici (hors
      périmètre) ; candidate à un chantier de centralisation + garde-rail.
- [ ] **8e (changelog).** `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`, entrée `[Unreleased]`
      v7.0 : bullet React « Explorer — briefing V4 » (Classement par chaîne, tuiles refondues
      hero WR + Durée totale + Pic FDA (+ Pic MMR), Perf colorée, Meilleure série, tooltips
      non clippés, MVP/LVP dans le tableau, padding réduit) + bullet Go (retrait `trend` du
      DTO briefing, champs Scope durée/pics, Classement par chaîne). Parité EN/FR même commit.
- [ ] **8f (clôture).** Dérouler `delivery-checklist`. Passe finale des gates §1.9 verte en
      une fois. Entrée `.ai/thought_log.md` finale. Point d'étape utilisateur. NON committé la
      livraison finale sans autorisation (merge `main` = deploy prod auto → après revue
      visuelle utilisateur).

Gate Phase 8 : `cd apps/go-api && go test ./...` = 0 ; `make go-api-lint` = 0 ; `make
generate-types` idempotent ; `make check-types` = 0 ; `make test-web` vert ; `npm run lint`
= 0 erreur ; greps de clôture (record_label, Sparkline, trend, RankDeltas = 0) ; changelog EN
+ FR à jour ; chaque item de dette statué.

---

## 6. Découvertes (à remplir en exécution — ne pas traiter hors périmètre)

- **Découverte-0 (pré-notée) — le fix Classement est en LECTURE seule.** Toutes les colonnes
  nécessaires (`playlist_group`, `rating_value`, `rating_delta`) existent déjà dans
  `match_skill_rank_latest` ; `duration_seconds` dans `v_match_full`. Aucune migration, aucune
  écriture → `-tags=integration` non requis (§4). Le delta par-match est DÉJÀ per-chaîne au
  sync (`skill_v2_canonical.go:163`).
- **Découverte-1 (pré-notée) — CSR = chaîne unique « ranked ».** `csr_writes.go:104,131` fige
  `playlist_group = "ranked"` pour toutes les lignes CSR → CSR ne se scinde jamais (invariant
  P-3 « une entrée par type » automatiquement respecté). Seul LUSR (4 chaînes) se scinde.
- **Découverte-2 (pré-notée) — vue `_latest` = 1 ligne/match (CSR > LUSR).** Chaque match
  appartient à exactement une chaîne `(type, playlist_group)` → segmentation propre, pas de
  double-comptage. `snap.Delta` (KPIStats) et `rating_delta` (raw row) proviennent de la MÊME
  vue → valeurs identiques (bascule KPIStats → raw row sans changement de valeurs).
- **Découverte-3 (pré-notée) — `RankDeltas` purgeable.** Grep : unique consommateur =
  `buildBriefingRanked` (+ 2 tests). Après DEC-RANK-BE, purge (1f). RE-VÉRIFIER avant
  suppression.
- **Découverte-4 (pré-notée) — label de chaîne réutilisable.** `lusrChainLabel`
  (`features/career/lusr-chains.ts:68-71`, clés `career.lusr.chain.*`) résout déjà les 4
  chaînes LUSR FR/EN → réutiliser (import cross-feature ou promotion, DEC-RANK-FE). Pas de clé
  neuve.
- **Découverte-5 (pré-notée) — hero WR card = tuile de `HomeHeroKPIGrid`.** Il n'existe pas de
  « hero card » surdimensionnée ; la référence est la tuile WR de
  `HomeHeroKPIGrid.tsx:147-167` (OutcomeBar + V-D-N + tooltip + accent sentiment). Le contenu
  repris n'exige aucune donnée nouvelle (V/D/N/DNF déjà dans le scope).
- **Découverte-6 (pré-notée) — style MVP/LVP centralisé.** Source unique
  `MatchScoreboard.logic.ts` (`cellState`/`cellStyle`), déjà réutilisée par
  `LeaderboardBlock.highlight.ts`. La matrice d'impact (`SquadImpactScoreboard.tsx`) en est une
  2ᵉ copie inline (dette, 8d). Explorer = 3ᵉ surface → IMPORT obligatoire (pas de recopie).
- **Découverte-7 (pré-notée) — `eslint-disable` : n° de ligne dérivé.** Le brief cite
  `ExplorerPage.tsx:159` ; l'état courant montre des `exhaustive-deps` en `:152`/`:161`.
  RE-VÉRIFIER lequel (le cas échéant) est réellement « unused » avant retrait (8b).
- **Découverte-8 (pré-notée) — Pic rang faisable via l'ordinal canonique.**
  `analysis.CSRTierOrdinal(tierEN string) int` (`home_canonical_skill.go:181`, testé,
  Bronze→Onyx+Champion) est LA source d'ordre des paliers, déjà réutilisée par le pic de la home
  (`home_repo_skill_peak.go:388-418`, `pickBestCSRAlltime`). Le Pic rang du briefing la réutilise
  (+ `sub_tier` exposé sur la raw row en Phase 2) — pas de nouvelle table de paliers, pas de
  comparaison inter-système (max PAR `rating_type`). `SkillTier` (nom EN) est déjà sur la raw row ;
  `sub_tier` est dans `match_skill_rank_latest` (§2.3).

- **Découverte-9 (Phase 0, exécution) — couplage inter-phase Phase 2 ↔ Phase 3 sur `trend`
  et `make check-types`.** Le front `ExplorerBriefingStrip.tsx` LIT ENCORE `briefing.trend`
  (`:75` `trendValues`, `:99-101` bloc `Sparkline`). Purger le champ `trend` du DTO backend
  (Phase 2a/2e) régénère `generated.ts` sans `ExplorerBriefing.trend` → `briefing.trend`
  devient une erreur TS2339 → **`make check-types` (gate de Phase 2) échoue** tant que le read
  frontend n'est pas retiré (que le plan défère à Phase 3b). Résolution retenue en exécution :
  Phase 2 réalise le sous-ensemble MINIMAL et mécanique de 3b strictement forcé par la purge —
  retrait de `trendValues` (`:75`), du bloc `chart`/`Sparkline` (`:98-108`), de l'import
  `Sparkline` (`:13`) devenu mort, et des exports `ExplorerBriefingTrend`/`...TrendPoint` de
  `types.ts` (`:828-829`). AUCUN autre travail frontend (refonte tuiles, OutcomeBar, Perf
  couleur, nouvelles tuiles) n'est fait ici — il reste au bloc frontend Phases 3-4. Sans ce
  micro-retrait, le bloc backend n'est pas committable (build cassé).
- **Découverte-10 (Phase 0, exécution) — commentaire « 25 colonnes » déjà périmé.** Le
  commentaire de `Q5SharedHistory` (`queries_career.go:54-55`) et celui de `LoadAll`
  (`match_history_repo.go:28`) annoncent « 25 colonnes » mais le SELECT en compte réellement
  **30** (état pré-chantier). En ajoutant `duration_seconds` (Phase 2c) le compte passe à 31 :
  les deux commentaires seront corrigés au bon nombre dans la Phase 2 (pas un fix hors
  périmètre — c'est le commentaire de la requête modifiée).

- **Découverte-11 (Phase 1, exécution) — seuil `minRankedKindMatches` (type secondaire)
  retiré.** V3 n'émettait un type de rating SECONDAIRE que s'il atteignait 10 matchs
  (`minRankedKindMatches`, briefing.go). Le critère §1.1.1 (« une ligne par chaîne, jamais
  de flèche inter-chaînes ») et l'algo pur `ComputeRankProgressionByChain` (retourne UNE
  entrée par chaîne) rendent ce seuil de TYPE caduc : la segmentation est désormais par
  chaîne, toutes émises. Le const `minRankedKindMatches` (unique consommateur =
  l'ex-`buildBriefingRanked`) est donc supprimé (0 code mort, CLAUDE.md §7). Conséquence :
  une petite chaîne (ex. 3 matchs LUSR btb) apparaît désormais aux côtés de CSR — c'est le
  comportement voulu par DEC-1. Le test V3 `_RankedSecondaryTypeBelowThresholdOmitted` est
  remplacé par `_RankedSmallSecondaryChainStillEmitted`. NB : si la revue visuelle
  utilisateur juge les micro-chaînes bruyantes, un plancher par chaîne pourra être ajouté
  (décision produit, hors périmètre de ce chantier — noté pour l'étape « À vérifier
  visuellement » item 1).

Consigner ici tout décalage fichier:ligne vs §2, tout lecteur i18n inattendu, toute dette
repérée hors périmètre. Ne pas corriger dans ce chantier (hors Phase 8 scopée).

---

## 7. Protocole de reprise de session

1. `git branch --show-current` doit être `feat/explorer-briefing-compact` (sinon la retrouver
   via `git log --oneline -10`). Ne jamais reprendre sur `main` ni une branche de train.
2. Lire ce fichier : la dernière phase dont le **Gate** est coché est close ; reprendre à la
   première non close. Les cases `[ ]` d'une phase non close = travail restant.
3. Lire l'entrée `.ai/thought_log.md` la plus récente de ce chantier (avancement + décisions,
   dont DEC-PEAKRANK et la note pt/match DEC-RANK-BE retenues).
4. Re-vérifier sur pièces les fichier:ligne de la phase courante AVANT d'éditer ou de cocher
   (le code a pu bouger).
5. Ne jamais commencer une phase N+1 tant que le Gate de N n'est pas vert.

---

## 8. Effort estimé & dépendances

| Bloc | Phase | Effort | Couche |
|---|---|---|---|
| Commit move + cadrage | 0 | Rapide | git + plan |
| Classement par chaîne | 1 | **Lourd** | repo + **analysis pur** + domain + service + tests + OpenAPI |
| Purge trend + agrégats socle | 2 | Moyen | repo + domain + service + tests + OpenAPI |
| Refonte socle tuiles (hero WR, Perf, Durée, Pic FDA/rang/MMR + cascade, Meilleure série) | 3 | Moyen | front + i18n |
| Classement 4ᵉ colonne + grilles | 4 | Moyen | front (RankedBlock + layout) |
| Tooltips portal | 5 | Moyen | front (`info-tooltip.tsx`) + tests |
| MVP/LVP tableau (réutilisation style) | 6 | Moyen | front (sibling highlight) |
| Padding + reformulation | 7 | Rapide | front + i18n |
| Dettes + changelog + clôture | 8 | Moyen | front + i18n + docs |

**Dépendances inter-phases** : Phase 1 (DTO Classement par chaîne) précède Phase 4 (bloc
Classement front). Phase 2 (agrégats Scope) précède Phase 3 (tuiles Durée/Pics). Phases 5/6/7
sont indépendantes entre elles (portal / table highlight / padding+i18n) et suivent la
structure (3-4). Phase 8 en dernier. **Points à confirmer par l'utilisateur** : aucun blocage —
DEC-PEAKRANK est TRANCHÉE (Pic rang RETENU par système + règle de cascade ≤ 8 ; Pic MMR = priorité
la plus basse) et la note pt/match DEC-RANK-BE est TRANCHÉE (variation nette co-signée). **Aucun
déploiement prod** dans ce chantier (le merge `main` = deploy auto reste la décision de
l'utilisateur, après revue visuelle).

---

## À vérifier visuellement par l'utilisateur (repris par l'utilisateur, PAS une tâche agent)

Sur l'Explorer mode Matchs (dev local `:8000`/vite), profils réels halo_infinite (LUSR) + un
titre H5 (dégradation) + un état low_sample + spot-check EN :

1. **Classement** : une ligne par chaîne LUSR (labels de chaîne lisibles), pt/match cohérent
   avec la progression de chaque ligne ; CSR = une seule ligne ; jamais de flèche
   inter-chaînes. Bloc rendu en 4ᵉ colonne sous « Par contexte », hauteurs harmonieuses.
2. **Socle** : sparkline absente ; tuile WR (OutcomeBar + V-D-N + tooltip) ; Perf colorée ;
   Durée totale (« h min ») ; Pic FDA ; puis cascade des conditionnelles (Meilleure série >
   Pic rang > Pic MMR, AU PLUS 2 des 3) — Pic rang jusqu'à 2 lignes (LUSR/CSR) ; rangée pleine
   largeur à 4-8 tuiles sans trou (jamais > 8).
3. **Tooltips** : le panneau (i) s'affiche EN ENTIER (non clippé) sur une tuile ; fermeture au
   scroll/clic extérieur ; texte FR sans anglicismes ; EN en parité.
4. **Tableau** : MVP/LVP surlignés (meilleur/pire par colonne) sur tout le scope, indépendants
   du tri ; padding plus compact sans perte de lisibilité ; tooltip de légende `tip_dimensions`
   reformulé.
5. Console 0 erreur sur les 4 états. Puis décision de merge (`main` = deploy prod auto).
