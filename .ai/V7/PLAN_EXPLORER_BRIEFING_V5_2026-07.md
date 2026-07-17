# PLAN — Explorer briefing V5 : triptyques FDA/Perf, cascade ≤ 8 révisée, Classement en rangée, décile MVP/LVP, alignement & finitions

Statut : PLANIFIE (aucune ligne de code écrite — plan rédigé par l'architecte Opus).
Date : 2026-07-17.
Chantier précédent : `.ai/V7/PLAN_EXPLORER_BRIEFING_V4_2026-07.md` (livré en 4 commits :
`b822cb651`, `13ef39182`, `a081a6e7e` + docs `52cf17a4c` ; serveur go-api rebuild, V4 actif).
Ce chantier = évolutions POST-revue visuelle V4, toutes CONFIRMÉES par l'utilisateur le
2026-07-17 (§3, DP-1..DP-10) — à VÉRIFIER sur pièces et structurer, jamais re-débattre.

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
> `make check-types`. **La vérification NAVIGATEUR est reprise par l'utilisateur** (§ « À
> vérifier visuellement ») — jamais planifiée comme tâche agent.

---

## 1. Objectif et critères de succès (mesurables)

**Objectif.** Faire évoluer le bandeau de briefing de l'Explorer (mode Matchs) selon les 10
décisions produit tranchées le 2026-07-17 (§3) : (DP-1) transformer les tuiles **FDA** et
**Perf** en triptyques **min · moyenne · max** (moyenne mise en avant, colorée), avec Min/Max
calculés côté SERVEUR ; **fusionner et supprimer** la tuile autonome « Pic FDA » ; (DP-2)
réviser la cascade des conditionnelles (cap 8, les **3** tiennent désormais — retrait de la
règle « au plus 2 des 3 ») ; (DP-3) mettre le **Classement** sur la MÊME rangée que les blocs
« Par… » (siblings d'une seule grille auto-fit, plus d'empilement) ; (DP-4) passer le
surlignage **MVP/LVP** du tableau d'un extrême unique à une **bande de décile** (p90/p10) ;
(DP-5) **aligner à droite** les en-têtes et cellules numériques ; (DP-6) porter un **accent**
sur TOUTES les KPI cards (jeton neutre sans sentiment) ; (DP-7) **retirer le compteur
redondant** « N matchs trouvés » et **déplacer l'export CSV sous le tableau** ; (DP-8)
**centrer** la valeur de première ligne des KPI cards ; (DP-9) renommer « Séries » →
« **Séries marquantes** » ; (DP-10) retirer le mot « matchs » des compteurs « XX matchs » des
lignes de dimension/contexte.

**Critères de succès (tous vérifiables ; la vérification NAVIGATEUR est reprise par
l'utilisateur — § « À vérifier visuellement ») :**

1. **Triptyques FDA & Perf (DP-1).** La tuile FDA affiche trois valeurs — `scope.min_kda`
   (petit), `scope.kda` (grand, coloré `kdaNetColor`), `scope.peak_kda` (petit) ; la tuile
   Perf — `scope.min_perf` (petit), `scope.avg_perf` (grand, coloré `getPerfColor`),
   `scope.max_perf` (petit). Min/Max viennent de NOUVEAUX champs SERVEUR (`MinKDA`,
   `MinPerf`, `MaxPerf` sur `ExplorerBriefingScope`, tous `*float64`/`omitempty`). La tuile
   autonome « Pic FDA » n'existe plus (fusionnée) ; `peak_kda` est réutilisé comme max FDA.
   Min/Max absents (nil) → le triptyque n'affiche que la moyenne (pas de « — » parasite).
   Vérifié par tests service (Min/Max/ordre) + tests Strip (rendu triptyque).
2. **Cascade ≤ 8 révisée (DP-2).** Base hors low_sample = **5** tuiles (Matchs, Taux de
   victoire, FDA triptyque, Perf triptyque, Durée totale). Conditionnelles = **3** (Séries
   marquantes, Pic rang, Pic MMR), rendues TOUTES si présentes (5 + 3 = 8). La règle V4
   « au plus 2 des 3 » (`slice(0, 2)`) est SUPPRIMÉE. Jamais > 8, jamais de trou en fin de
   rangée. low_sample = 4 (Matchs/WR/FDA/Perf).
3. **Classement sur la même rangée (DP-3).** Les cartes de dimension (« Par carte/mode/
   sélection »), « Par contexte » ET « Classement » sont des blocs SIBLINGS d'une seule
   grille responsive (`auto-fit`/`minmax`) — plus de 4ᵉ colonne empilée, plus de 2ᵉ rangée.
   Le bloc Classement liste ses chaînes VERTICALEMENT et s'allonge sans déborder si N chaînes.
   Vérifié par tests Modules (Classement = grid sibling, pas empilé sous « Par contexte »).
4. **MVP/LVP en décile (DP-4).** Le surlignage du tableau met en valeur le **top 10 %** et le
   **pire 10 %** de chaque colonne clé (seuils p90/p10 calculés client sur TOUT le scope) — la
   bande REMPLACE l'extrême unique (le meilleur/pire est naturellement dans sa bande). La
   teinte réutilise le style de `MatchScoreboard.logic`
   (`cellStyle`, tokens `outcome-win`/`outcome-loss`) — teinte DOUCE (intensité réduite) —
   jamais recopié (garde-rail `color-mix(in oklab` NOUVEAU sous `features/explorer` = 0).
   Indépendant du tri, égalités multi-surlignées, nulls neutres.
5. **Alignement par colonne (DP-5).** Colonnes NUMÉRIQUES alignées à DROITE (en-tête ET
   cellules) ; colonnes TEXTE à GAUCHE. `HEADER_TH_CLASS` ne porte plus `text-left` global ;
   l'alignement est PAR COLONNE. Jamais centré.
6. **Accent sur toutes les KPI cards (DP-6).** Chaque `BriefingTile` porte un `accent`
   (prop de `KpiCard`) : sentiment quand il existe (WR, FDA, Perf) ; jeton NEUTRE
   (`outcome-draw`, cf. `deltaToken(0)`) sinon (Matchs, Durée, Séries marquantes, Pic rang,
   Pic MMR). Aucune couleur inventée.
7. **Compteur retiré + CSV en bas (DP-7).** Le texte « N matchs trouvés »
   (`ExplorerPage.matchesMode.tsx`) au-dessus du tableau est SUPPRIMÉ. Le bouton « Exporter
   CSV » est déplacé SOUS `<ExplorerMatchesTable>`. La clé `count_label` reste vivante (encore
   consommée par le pied de pagination du tableau).
8. **Valeurs centrées (DP-8).** La valeur de première ligne des `BriefingTile` est centrée
   (`text-center`), cohérente avec les triptyques FDA/Perf et le ruban `OutcomeBar` de la
   tuile Taux de victoire (pas de désalignement).
9. **Renommage (DP-9).** `streaks_title` = FR « Séries marquantes » / EN « Notable streaks » ;
   contenu de la tuile inchangé. Garde-rail terminologie (`explorerBriefingTerminology.guard`)
   vert.
10. **Compteurs sans « matchs » (DP-10).** Les lignes de dimension (`DimensionRow`) et de
    contexte (`ContextSplitRow`) affichent juste le nombre « XX » (plus « XX matchs ») ; la
    clé `dim_matches` sert de libellé accessible (`title`/aria). FR sans anglicismes.
11. **Gates verts** (par phase, §5) : `cd apps/go-api && go test ./...` = 0 (SÉQUENTIEL) ;
    `make go-api-lint` = 0 ; `make generate-types` idempotent (0 diff résiduel) +
    `TestOpenAPISchemaDrift` vert ; `make check-types` = 0 (cache `.tmp` purgé) ;
    `make test-web` (dangerouslyDisableSandbox) vert ; `cd apps/web && npm run lint` = 0
    erreur ; greps de clôture verts. `-tags=integration` NON requis (§4, justifié).
12. **Changelog** : entrée `[Unreleased]` v7.0 mise à jour dans `docs/CHANGELOG.md` ET
    `docs/FR/CHANGELOG.md` (parité EN/FR même commit).

---

## 2. Constat sur pièces — état actuel post-V4 (fichier:ligne réels, vérifiés le 2026-07-17)

> Doctrine du projet : RE-VÉRIFIER chaque ancrage sur pièces AVANT de coder ET avant de
> cocher (le code a pu bouger). Numéros ci-dessous = état vérifié le 2026-07-17 (post-V4).

### 2.1 Backend — agrégats socle (DP-1)

- **Raw row porte DÉJÀ tout le nécessaire** `apps/go-api/internal/domain/match_history.go` :
  `MatchHistoryRawRow.KDA *float64` (`:42`, KDA natif par match) et `PerformanceScore
  *float64` (`:49`). **Aucune colonne SQL à ajouter** : `KDA`/`PerformanceScore` sont déjà
  scannés (source `p.kda`/`mp.kda` de `Q5SharedHistory`, `queries_career.go:84,251` ;
  `PerformanceScore` déjà agrégé en `scope.avg_perf`).
- **DTO** `apps/go-api/internal/domain/explorer_briefing.go` : `ExplorerBriefingScope`
  (`:58-81`) porte `KDA float64` (`:65`), `AvgPerf *float64` (`:67`), `PeakKDA *float64`
  (`:73`, = max KDA), `PeakTeamMMR`, `PeakRanks`, `TotalDurationSeconds`. **Manquent**
  `MinKDA`, `MinPerf`, `MaxPerf` (à ajouter, DP-1).
- **Service** `apps/go-api/internal/service/match_history_service_briefing.go` :
  - `buildBriefingScope` (`:100-119`) construit le Scope ; `PeakKDA = maxScopeFloat(scope, r
    => r.KDA)` (`:115`), `AvgPerf = a.perf` (`:113`), `KDA = a.kda` (`:112`).
  - `maxScopeFloat(scope, get)` (`:140-153`) = max générique d'un champ `*float64` (nil si
    absent partout). **Pas de `minScopeFloat`** (à créer, symétrique).
  - `aggregateRawStats` (`:228-259`) : `out.kda = analysis.AggregateKDA(k, a, d, matches)`
    (`:253`) ; `out.perf = moyenne des PerformanceScore` (`:248-257`).
- **Formule agrégat** `apps/go-api/internal/analysis/indicators.go` :
  `AggregateKDA(k, a, d, matches) = (k + a/3 − d) / matches` (`:38-43`) = **moyenne** du net
  par-match. Le KDA natif par match (`match_participants.kda`) est le même quantum
  `k + a/3 − d` → `min(r.KDA) ≤ scope.kda ≤ max(r.KDA)=peak_kda` (triptyque ordonné ;
  ordre à confirmer en test si arrondi DB — cf. Découverte-2). Pour Perf, `min ≤ moy ≤ max`
  est EXACT (moyenne arithmétique des mêmes `PerformanceScore`).
- **OpenAPI** `apps/go-api/api/openapi.yaml` : `ExplorerBriefingScope` (`:5030`), propriétés
  `avg_perf` (`:5033`), `peak_kda` (`:5048`), `total_duration_seconds` (`:5062`). Ajouter
  `min_kda`/`min_perf`/`max_perf` ici (émission Huma byte-exact → `TestOpenAPISchemaDrift`).

### 2.2 Frontend — socle des tuiles (DP-1 / DP-2 / DP-6 / DP-8 / DP-9)

- **`apps/web/src/features/explorer/ExplorerBriefingStrip.tsx`** (185 L) :
  - Grille socle `grid gap-2 grid-cols-2 sm:[grid-template-columns:repeat(auto-fit,
    minmax(150px,1fr))]` (`:99`).
  - **Tuile FDA** (`:112-136`) : `BriefingTile` `value = <span style={color kdaNetColor(kda)}>
    {kda.toFixed(2)}</span>`, `sub = delta_kda`, `accent = deltaToken(kda)`.
  - **Tuile Perf** (`:139-164`) : `value = <span style={color getPerfColor(perf)}>
    {perf.toFixed(0)}</span>`, `sub = delta_perf` — **AUCUN `accent`** (à ajouter, DP-6).
  - **Cascade** (`:80-95`) : `conditionalTiles` collectées par priorité (Séries `:86` >
    Pic rang `:89` > Pic MMR `:92`), puis `cappedConditionals = conditionalTiles.slice(0, 2)`
    (`:95`). **Retirer le `slice(0, 2)`** (DP-2 : rendre toutes les présentes).
  - **Pic FDA autonome** : `<PeakKdaTile value={scope.peak_kda} … />` rendu `:168`
    (`{scope && !lowSample && <PeakKdaTile … />}`) → **SUPPRIMER** (DP-1, fusionné dans FDA).
  - Base hors low_sample : Matchs (`:101`), WR (`:109`), FDA (`:112`), Perf (`:139`), Durée
    (`:167`), Pic FDA (`:168`). Après DP-1 : 5 (Pic FDA retiré).
- **`apps/web/src/features/explorer/ExplorerBriefingTiles.tsx`** (218 L) : composants
  `StreaksTile` (`:33`, lit `streaks_title` `:38` — renommage DP-9), `WinRateTile`
  (`:64-152`, hero OutcomeBar + V-D-N + tooltip, `accent` sentiment `:114`), `DurationTile`
  (`:156-164`, **pas d'accent**), `PeakKdaTile` (`:168-178`, **à supprimer** DP-1),
  `PeakRankTile` (`:184-206`, **pas d'accent**), `PeakMmrTile` (`:210-218`, **pas d'accent**).
- **`apps/web/src/features/explorer/BriefingTile.tsx`** (39 L) : `KpiCard accent={accent}`
  (`:26`) ; div valeur `<div className="mt-0.5 text-xl font-bold tabular-nums leading-tight
  text-foreground">` (`:32`) → **ajouter `text-center`** (DP-8). `TileProps.accent?:
  SemanticToken` (`:21`).
- **`apps/web/src/components/cards/KpiCard.tsx`** (47 L) : prop `accent?: SemanticToken`
  (`:24`, omis ⇒ pas d'accent), barre 3px top (`:41-43`). Utilisable pour DP-6, aucune modif
  de `KpiCard` requise.
- **Couleurs / échelles** : `kdaNetColor` (`@/lib/colors/outcomePalette`), `getPerfColor`
  (`apps/web/src/lib/perf-color.ts:27`, = `tokenCssVar(perfScale(score))`), `perfScale`
  (`@/lib/accessibility/scales`, retourne un `SemanticToken perf-tier-N` — utilisable comme
  accent Perf, DP-6).
- **Jeton neutre (DP-6)** `apps/web/src/lib/accessibility/semantic-tokens.ts` : PAS de token
  « neutral »/« muted » générique ; existe `divergent-neutral` (`:28`) et `outcome-draw`
  (`:15`). `deltaToken(0) = 'outcome-draw'` (`ExplorerBriefing.logic.ts:61-64`) et la tuile
  Matchs utilise DÉJÀ `accent="outcome-draw"` (Strip `:105`) → **`outcome-draw` = le neutre
  ÉTABLI du briefing** (retenu, DEC-NEUTRAL).
- **i18n** `apps/web/src/lib/i18n/manifests/explorer.toml` : `streaks_title` (`:887-889`,
  FR « Meilleure série » / EN « Best streak ») → renommer (DP-9). `peak_fda_label` +
  `tip_peak_fda` (ajoutés en V4) → **orphelinés** après suppression de `PeakKdaTile`
  (à purger DANS la phase, RE-VÉRIFIER le grep). `tip_fda`/`tip_perf` → réviser pour décrire
  le triptyque (min · moyenne · max).

### 2.3 Frontend — modules « Par… » (DP-3 / DP-10)

- **`apps/web/src/features/explorer/ExplorerBriefingModules.tsx`** (305 L) :
  - Grille « Par… » `grid grid-cols-1 gap-2 sm:[grid-template-columns:repeat(auto-fit,
    minmax(240px,1fr))]` (`:108`).
  - `DimensionCard` rendues en enfants DIRECTS de la grille (`:109-111`).
  - **4ᵉ colonne empilée (V4)** (`:112-119`) : `<div className="flex flex-col gap-2">`
    empilant `ContextSplitCard` (`:114`) + `RankedBlock` (`:116`) — piloté par
    `showSplitColumn = contextSplit != null || showRanked` (`:97`). → **DÉ-EMPILER** (DP-3 :
    `ContextSplitCard` et `RankedBlock` deviennent des enfants DIRECTS/siblings de la grille).
  - Garde early-return (`:101`) et condition de rendu de la grille (`:107`) référencent
    `showSplitColumn` → à adapter (`contextSplit != null || showRanked`).
  - `DimensionRow` (`:160-206`) : compteur `{t('explorer.briefing.dim_matches', { n:
    entry.matches })}` (`:177`) dans un `<span className="shrink-0 tabular-nums
    text-muted-foreground">` (`:176-178`). → DP-10.
  - `ContextSplitRow` (`:231-260`) : même compteur `dim_matches` (`:246`). → DP-10.
- **`apps/web/src/features/explorer/ExplorerRankedBlock.tsx`** (114 L) : `RankedBlock`
  (`:78-114`) = `BriefingSectionCard` (`:93`) + `<ul className="space-y-1">` (`:101`) d'une
  ligne PAR CHAÎNE (`RankedChainRow`). Déjà un bloc-carte autonome vertical → **directement
  utilisable comme grid sibling** (DP-3, aucun refactor du bloc).
- **`dim_matches`** `explorer.toml:861-863` : FR « {n, plural, one {# match} other
  {# matchs}} » / EN « {n} match(es) ». Deux consommateurs (Modules `:177` et `:246`). DP-10
  = rendu du nombre seul + `dim_matches` conservée en `title`/aria (non orpheline).

### 2.4 Frontend — tableau des matchs (DP-4 / DP-5 / DP-7)

- **`apps/web/src/features/explorer/ExplorerMatchesTable.tsx`** (819 L — god-file préexistant,
  hors périmètre de split, cf. V4 Découverte-18) :
  - `HEADER_TH_CLASS` (`:129-130`) = `'px-2 py-1 text-left whitespace-nowrap text-3xs
    font-medium text-muted-foreground border-r border-border last:border-r-0'` — **`text-left`
    global** (DP-5 : à retirer du socle partagé + alignement par colonne).
  - En-têtes rendus via `flexRender` dans `<th className={HEADER_TH_CLASS}>` (statique `:731`,
    triable `:741`). Cellules `<td className="px-2 py-1 whitespace-nowrap border-r
    border-border last:border-r-0">` (`:774`, pas de `text-align` → défaut gauche).
  - **Colonnes NUMÉRIQUES** (id) : `kills` (`:421`), `deaths` (`:436`), `assists` (`:451`),
    `kda` (`:466`), `score_label` (`:490`), `duration_seconds` (`:503`), `perf_score`
    (`:514`), `delta_perf` (`:532`), `team_mmr` (`:591`), `enemy_mmr` (`:602`), `delta_mmr`
    (`:613`). **Colonnes TEXTE** (gauche) : `open`, `start_time` (Date), map, playlist, mode,
    contexte, outcome, `dominance_flag` (`:396`), `rating_type` (`:556`, « CSR »/« LUSR »),
    `skill_tier_label` (`:574`, « Rang », libellé de palier). `columnIdOf(c)` (`:145`) résout
    l'id (accessorKey/id explicite).
  - **Surlignage MVP/LVP** câblé : import (`:55-60`) ; `highlightExtremes = useMemo(() =>
    computeColumnExtremes(rows), [rows])` (`:242`, sur TOUT le scope) ; application sur le
    `<td>` (`:767-775`) `columnHighlightStyle(colId, explorerHlExtract[colId](row.original),
    highlightExtremes)`.
- **`apps/web/src/features/explorer/ExplorerMatchesTable.highlight.ts`** (91 L) :
  `computeColumnExtremes(rows)` (`:70-78`, `{min,max}` par colonne, garde ≥ 2) ;
  `columnHighlightStyle(key, value, extremes)` (`:85-91`) = `cellStyle(cellState(value,
  extremes[key], EXPLORER_INVERTED[key]))`. `EXPLORER_INVERTED` (`:24-30`, `deaths:true`).
  `explorerHlExtract` (`:46-54`, valeur AFFICHÉE ; `perf_score` gardée sur `perf_tier`
  présent ; `score_label` → `ownTeamScore`). `isExplorerHighlightKey` (`:61-63`).
  **Seul consommateur = `ExplorerMatchesTable.tsx`** (grep confirmé) + test
  `ExplorerMatchesTable.highlight.test.ts`.
- **`apps/web/src/features/match-view/MatchScoreboard.logic.ts`** : `cellState(value, ex,
  inverted)` (`:36-43`, best/worst sur `value === ex.max`/`ex.min` — **exact, non adaptable à
  une bande**) ; `cellStyle(state)` (`:56-64`, tokens `outcome-win`/`outcome-loss`, teinte
  `color-mix(in oklab, … 28%, transparent)`, `fontWeight 600/500`) — **réutilisable** (source
  unique du littéral `color-mix`). Consommateurs : scoreboard + `LeaderboardBlock.highlight`
  (+ Explorer). `SquadImpactScoreboard.tsx` en DUPLIQUE le style (2ᵉ copie, dette V4
  Découverte-17, hors périmètre).
- **`apps/web/src/features/explorer/ExplorerPage.matchesMode.tsx`** :
  `ExplorerMatchesResultsBlock` (`:310-386`) : bandeau (`:352`), barre `<div className="flex
  items-center justify-between gap-2">` (`:356-372`) contenant le **compteur** `<p>
  {count_label}</p>` (`:357-361`) et le **bouton CSV** `<a … download>` (`:362-371`), puis
  `<ExplorerMatchesTable … sortable />` (`:377-383`). *(Les numéros ~376-380/~412-421 de la
  mission étaient approximatifs ; les vrais sont ci-dessus.)*
- **`count_label`** `explorer.toml:181-183` : AUSSI consommée par le pied de pagination du
  tableau (`ExplorerMatchesTable.tsx:789`). → retirer l'usage de la page (DP-7) N'ORPHELINE
  PAS la clé (aucune purge i18n).

### 2.5 Tests existants du périmètre (à mettre à jour, jamais skipper)

`apps/web/src/features/explorer/` : `ExplorerBriefingStrip.test.tsx` (socle : triptyques,
cascade, accents, centrage, renommage), `ExplorerMatchesTable.test.tsx` +
`ExplorerMatchesTable.highlight.test.ts` (décile, alignement), `ExplorerPage.test.tsx`
(compteur/CSV), `explorerBriefingTerminology.guard.test.ts` (DP-9/DP-10 : ne trippe pas —
n'interdit que « Par playlist »/« Pronostic »/« Prognosis », `:18`). Modules : chercher le
test couvrant `ExplorerBriefingModules` (Classement en grille). Backend :
`match_history_service_briefing*_test.go` (Min/Max scope).

**Conclusion du constat.** Le chantier est **backend léger d'abord** (3 champs Scope Min/Max
+ helper `minScopeFloat` — AUCUNE requête SQL, aucune écriture), puis **frontend socle**
(triptyques + cascade + accent + centrage + renommage), **modules & page** (Classement en
rangée + compteurs sans « matchs » + compteur/CSV), enfin **tableau** (décile + alignement).
Toutes les données Min/Max existent déjà sur la raw row → `-tags=integration` non requis (§4).

---

## 3. Décisions — pré-tranchées (fermes, ne pas re-débattre en exécution)

### Décisions produit (utilisateur, 2026-07-17 — reprises telles quelles)

- **DP-1 (FDA & Perf en min · moy · max).** Les tuiles FDA et Perf affichent trois valeurs :
  la plus basse, la moyenne (mise en avant, grande, colorée), la plus haute — affichage
  INVENTIF/compact (moy en gros au centre, min/max petits de part et d'autre). FDA :
  min = NOUVEAU `MinKDA`, moy = `scope.kda` (agrégat ADR 0006 existant), max = `peak_kda`
  (existant). Perf : min = NOUVEAU `MinPerf`, moy = `scope.avg_perf` (existant), max = NOUVEAU
  `MaxPerf`. Couleur de la moy : `kdaNetColor` (FDA) / `getPerfColor` (Perf). La tuile
  autonome « Pic FDA » est SUPPRIMÉE (fusionnée dans FDA). Min/Max calculés côté SERVEUR
  (cohérence avec `peak_kda`/`avg_perf`). Nouveaux champs `ExplorerBriefingScope` : `MinKDA
  *float64`, `MinPerf *float64`, `MaxPerf *float64` (nil si absents). `peak_kda` conservé.
- **DP-2 (jeu de tuiles ≤ 8, cascade révisée).** Base hors low_sample (5) : Matchs, Taux de
  victoire, FDA, Perf, Durée totale. Conditionnelles (3) : Séries marquantes, Pic rang, Pic
  MMR — **les 3 tiennent** (5 + 3 = 8 ; le retrait de Pic FDA libère le slot). SUPPRIMER la
  règle V4 « au plus 2 des 3 » (`slice(0, 2)`) ; cap à 8, les 3 s'affichent si présentes. Pic
  MMR redevient visible.
- **DP-3 (Classement sur la MÊME rangée de blocs).** Abandon du V4 (Classement empilé sous
  « Par contexte » en 4ᵉ colonne). Les cartes de dimension + « Par contexte » + « Classement »
  sont des blocs SIBLINGS d'une seule grille responsive (auto-fit), tout sur une rangée
  (pas de 2ᵉ rangée, pas d'empilement). Le bloc Classement liste ses chaînes VERTICALEMENT
  (s'allonge si multi-chaînes, sans déborder).
- **DP-4 (MVP/LVP en décile).** Remplacer le surlignage « meilleur/pire unique par colonne »
  par un surlignage du **top 10 %** et **pire 10 %** de chaque colonne clé (seuils p90/p10
  calculés client sur TOUT le scope chargé). Réutiliser la teinte de `MatchScoreboard.logic`
  (`cellStyle`), pilotée par des SEUILS de décile ; teinte DOUCE. Adapter
  `computeColumnExtremes` → seuils décile.
- **DP-5 (en-têtes numériques alignés à droite).** Colonnes NUMÉRIQUES à droite (en-tête ET
  cellules), colonnes TEXTE à gauche. `HEADER_TH_CLASS` (aujourd'hui `text-left` global) →
  alignement PAR COLONNE. Ne pas centrer.
- **DP-6 (accent sur TOUTES les KPI cards).** Chaque `BriefingTile` porte un accent ; jeton
  NEUTRE quand pas de sentiment. Réutiliser le prop `accent` de `KpiCard` + un token
  sémantique neutre existant (NE PAS inventer de couleur).
- **DP-7 (retrait du compteur redondant + CSV en bas).** Supprimer « N matchs trouvés »
  (`ExplorerPage.matchesMode.tsx`) — redondant avec la tuile Matchs. Déplacer « Exporter
  CSV » SOUS le tableau.
- **DP-8 (valeurs de KPI cards centrées).** Centrer la valeur de première ligne des KPI cards
  (`BriefingTile` : `text-center` sur l'élément valeur). Rester cohérent avec les triptyques
  FDA/Perf et le ruban `OutcomeBar` de la tuile Taux de victoire.
- **DP-9 (« Séries » → « Séries marquantes »).** Renommer `streaks_title` FR « Séries
  marquantes » / EN « Notable streaks ». Contenu inchangé.
- **DP-10 (retrait du mot « matchs »).** Les lignes de dimension et de contexte affichent
  juste le nombre « XX » (plus « XX matchs »). Garder `dim_matches` en `title`/aria pour la
  lisibilité. FR sans anglicismes.

### Décisions techniques (architecte)

- **DEC-MINMAX (agrégats Min/Max — serveur, DP-1).** Ajouter à `ExplorerBriefingScope`
  (`explorer_briefing.go`) : `MinKDA *float64`, `MinPerf *float64`, `MaxPerf *float64` (tous
  `json:"…,omitempty"`, commentaires reliant DP-1). Créer `minScopeFloat(scope, get func(
  *domain.MatchHistoryRawRow) *float64) *float64` (symétrique de `maxScopeFloat`
  `match_history_service_briefing.go:140-153`). Dans `buildBriefingScope` (`:100-119`) :
  `MinKDA = minScopeFloat(scope, r => r.KDA)`, `MinPerf = minScopeFloat(scope, r =>
  r.PerformanceScore)`, `MaxPerf = maxScopeFloat(scope, r => r.PerformanceScore)`. **Aucune
  requête SQL, aucun champ raw row neuf** (`KDA`/`PerformanceScore` déjà scannés). OpenAPI :
  `min_kda`/`min_perf`/`max_perf` sur `ExplorerBriefingScope` (`:5030`), émission Huma
  byte-exact → `TestOpenAPISchemaDrift`. Algo trivial → tests en couche service (pas
  d'`internal/analysis/` neuf ; `minScopeFloat` reste un helper de service, symétrie stricte
  de l'existant).
- **DEC-TRIPTYCH (triptyque front, DP-1).** Composant partagé
  `MinMaxTriptych({ min, mid, max, midColor, format })` (dans `ExplorerBriefingTiles.tsx`,
  exporté) : `min`/`max` petits (`text-2xs text-muted-foreground tabular-nums`), `mid` grand
  (hérite `text-xl` du conteneur `BriefingTile`, `style={{ color: midColor }}`), disposition
  `inline-flex items-baseline justify-center gap-1.5`. Chaque borne nulle est OMISE (pas de
  « — » parasite) ; `mid` nul → « — ». Deux consommateurs (FDA, Perf) → un composant unique
  (évite une 2ᵉ/3ᵉ copie, CLAUDE.md §6). FDA : `min=scope.min_kda`, `mid=scope.kda`,
  `max=scope.peak_kda`, `midColor=kdaNetColor(kda)`, `format=v=>v.toFixed(2)`. Perf :
  `min=scope.min_perf`, `mid=scope.avg_perf`, `max=scope.max_perf`,
  `midColor=getPerfColor(perf)`, `format=v=>v.toFixed(0)`. Les tuiles FDA/Perf restent inline
  dans le Strip (`:112-164`) OU sont extraites en `FdaTile`/`PerfTile` dans
  `ExplorerBriefingTiles` (choix d'exécution, consigné §6) — dans les deux cas le `value` du
  `BriefingTile` = `<MinMaxTriptych …/>`, le `sub` (delta) est conservé.
- **DEC-CASCADE (retrait du cap 2, DP-2).** Dans `ExplorerBriefingStrip.tsx`, supprimer
  `const cappedConditionals = conditionalTiles.slice(0, 2)` (`:95`) et rendre directement
  `{conditionalTiles}` (`:171`). Les 3 conditionnelles présentes s'affichent (max naturel 3 →
  5 base + 3 = 8). Retirer aussi `PeakKdaTile` de la base (`:168`) et son composant
  (`ExplorerBriefingTiles.tsx:168-178`) + import (Strip `:29`). Grille socle inchangée
  (`auto-fit`/`minmax(150px,1fr)`, absorbe 4-8 tuiles sans trou).
- **DEC-ACCENT (accent partout + jeton neutre, DP-6).** Jeton NEUTRE = **`outcome-draw`**
  (neutre établi du briefing : `deltaToken(0)`, déjà l'accent de la tuile Matchs). Accents par
  tuile : Matchs = `outcome-draw` (inchangé) ; WR = sentiment (inchangé) ; FDA =
  `deltaToken(scope.kda)` (inchangé) ; **Perf = `perfScale(scope.avg_perf)`** (sentiment,
  perf-tier — NOUVEAU) ; **Durée = `outcome-draw`** ; **Séries marquantes = `outcome-draw`** ;
  **Pic rang = `outcome-draw`** ; **Pic MMR = `outcome-draw`**. `perfScale` importé de
  `@/lib/accessibility/scales` (retourne un `SemanticToken`). *Alternative documentée* : si la
  revue visuelle juge l'ambre `outcome-draw` trop « égalité », basculer TOUS les neutres (y
  compris Matchs) vers `divergent-neutral` (un swap de token, source unique) — décision
  d'exécution consignable §6, mais DÉFAUT = `outcome-draw` (zéro churn, cohérent avec les
  deltas).
- **DEC-CENTER (centrage valeurs, DP-8).** `BriefingTile.tsx:32` : ajouter `text-center` à la
  div valeur. Vérifier sur pièces (revue visuelle utilisateur) que la tuile WR
  (`OutcomeBar` + V-D-N, layout `flex` interne `ExplorerBriefingTiles.tsx:115-136`) ne se
  désaligne pas — le pourcentage se centre, le ruban reste pleine largeur ; si un ajustement
  local est nécessaire (ex. `justify-center` sur la ligne du ruban), il reste DANS la tuile
  WR, pas dans `BriefingTile`. Le label (ligne muted uppercase) N'EST PAS centré (DP-8 =
  valeur de première ligne uniquement).
- **DEC-LAYOUT (Classement en rangée, DP-3).** Dans `ExplorerBriefingModules.tsx` : supprimer
  le conteneur `<div className="flex flex-col gap-2">` (`:112-119`) et rendre
  `ContextSplitCard` et `RankedBlock` comme enfants DIRECTS de la grille « Par… » (après les
  `DimensionCard`), chacun gaté par sa condition (`contextSplit != null` ; `showRanked`).
  Adapter la condition d'affichage de la grille (`:107`) et l'early-return (`:101`) :
  remplacer `showSplitColumn` par `(contextSplit != null || showRanked)`. Le hook
  `useCapability('ranked')` (`:89`) et le calcul `showRanked` (`:94`) restent. Harmonisation
  des hauteurs : `RankedBlock`/`ContextSplitCard` = grid cells (stretch `auto-fit` par défaut ;
  ajouter `h-full` si la revue visuelle révèle un déséquilibre — proposition, revue visuelle).
  Le `minmax(240px,1fr)` peut être abaissé (ex. `200px`) pour tenir plus de cellules sur une
  rangée en desktop (revue visuelle ; documenter le choix, jamais de trou grâce à `auto-fit`).
- **DEC-COUNT (compteur nombre seul, DP-10).** `DimensionRow` (`:176-178`) et
  `ContextSplitRow` (`:245-247`) : remplacer le texte `{t('explorer.briefing.dim_matches',
  {n})}` par le nombre `{entry.matches}` / `{group.matches}`, en portant
  `title={t('explorer.briefing.dim_matches', {n})}` sur le `<span>` (libellé accessible au
  survol). `dim_matches` reste consommée (non orpheline). Aucune clé neuve.
- **DEC-DECILE (surlignage par décile, DP-4).** Dans `ExplorerMatchesTable.highlight.ts` :
  1. Remplacer `computeColumnExtremes` → `computeColumnDeciles(rows): Record<key, { p10:
     number | null; p90: number | null }>` (percentile nearest-rank sur les valeurs non
     nulles triées ; `< MIN_DECILE_SAMPLE` valeurs → `{null,null}` = pas de highlight ;
     `MIN_DECILE_SAMPLE` = 10, nommé + commenté, ajustable en revue visuelle). Type `Deciles`
     local (ne PAS importer `Extremes`).
  2. Nouvelle fonction locale `decileCellState(value, d, inverted): CellState` : `d.p10/p90
     null` ou `p10 === p90` → `'neutral'` ; non inversé → `value >= p90 ? 'best' : value <=
     p10 ? 'worst' : 'neutral'` ; inversé (`deaths`) → symétrique (`<= p10 ? 'best'`, `>= p90
     ? 'worst'`). `CellState` type importé de `MatchScoreboard.logic`.
  3. `columnHighlightStyle(key, value, deciles)` = `cellStyle(decileCellState(value,
     deciles[key], EXPLORER_INVERTED[key]), DECILE_TINT_PCT)`. **`cellState` n'est PLUS
     importé** (l'exact `===` ne convient pas à une bande) ; **`cellStyle` RESTE importé** (la
     teinte, source unique du `color-mix` — CLAUDE.md §6).
  4. **Teinte douce** : paramétrer `cellStyle` (`MatchScoreboard.logic.ts:56-64`) avec un
     `intensityPct: number = 28` (défaut = comportement inchangé pour scoreboard/leaderboard) ;
     l'Explorer passe `DECILE_TINT_PCT` (≈ 16, nommé + commenté, ajustable revue visuelle).
     C'est la SEULE modification de `MatchScoreboard.logic.ts` (ajout d'un param défaulté, zéro
     changement de comportement des callers existants ; `MatchScoreboard.test.ts` reste vert).
     *Alternative* (si l'exécutant juge risqué de toucher le fichier partagé) : réutiliser
     `cellStyle` tel quel (28 %) — la teinte 28 % oklab est déjà translucide ; consigner §6.
     DÉFAUT = paramétrage (honore « teinte douce » explicite tout en gardant la source unique).
  5. Câblage `ExplorerMatchesTable.tsx` : `highlightExtremes` → `highlightDeciles = useMemo(
     () => computeColumnDeciles(rows), [rows])` (`:242`) ; l'application `:767-775` appelle
     `columnHighlightStyle(colId, explorerHlExtract[colId](row.original), highlightDeciles)`.
     `isExplorerHighlightKey`/`explorerHlExtract`/`EXPLORER_INVERTED` inchangés.
- **DEC-ALIGN (alignement par colonne, DP-5).** `ExplorerMatchesTable.tsx` :
  1. Retirer `text-left` de `HEADER_TH_CLASS` (`:129-130`) → base sans `text-align`.
  2. Définir `const RIGHT_ALIGNED_COLUMNS = new Set([...ids numériques...])` (kills, deaths,
     assists, kda, score_label, duration_seconds, perf_score, delta_perf, team_mmr,
     enemy_mmr, delta_mmr — RE-VÉRIFIER les ids sur pièces) ; helper `alignClass(colId)` →
     `'text-right'` (numérique) / `'text-left'` (texte).
  3. Appliquer `alignClass(h.column.id)` sur le `<th>` (statique `:731` + triable `:741`) et
     `alignClass(cell.column.id)` sur le `<td>` (`:772-774`). Pour un `<th>` triable, le
     `<button>` interne est `inline-flex` ; l'alignement à droite exige `justify-end` OU un
     `<th>` `text-right` avec le bouton en `inline-flex` poussé à droite (ex. `w-full
     justify-end`) — l'exécutant retient la variante qui aligne visuellement l'en-tête et la
     flèche de tri à droite (revue visuelle). Ne JAMAIS centrer.
- **DEC-CSV (compteur/CSV, DP-7).** `ExplorerPage.matchesMode.tsx` (`ExplorerMatchesResults
  Block`, `:349-385`) : supprimer le `<p>` compteur (`:357-361`) ; retirer la barre
  `<div className="flex … justify-between">` (`:356-372`) devenue mono-élément et rendre le
  bouton CSV (`:362-371`) APRÈS `<ExplorerMatchesTable>` (`:377-383`), dans un conteneur
  aligné à droite (ex. `<div className="flex justify-end">`). `count_label` NON purgée (encore
  utilisée `ExplorerMatchesTable.tsx:789`). Aucune clé i18n neuve.
- **DEC-i18n (parité).** Toute clé modifiée l'est en FR ET EN dans `explorer.toml`, suivie de
  `node apps/web/scripts/build_i18n_manifests.mjs` AVANT `make check-types`. Clés touchées :
  - `streaks_title` → FR « Séries marquantes » / EN « Notable streaks » (DP-9).
  - `tip_fda` / `tip_perf` → réviser pour décrire les trois valeurs (le plus bas, la moyenne,
    le plus haut), FR + EN.
  - `peak_fda_label` + `tip_peak_fda` → **PURGER** (orphelinées par la suppression de
    `PeakKdaTile` ; RE-VÉRIFIER le grep = 0 lecteur composant avant purge).
  - `dim_matches` → conservée (rendu nombre seul + `title`, DP-10 ; pas de changement de
    valeur de clé).
  - `count_label` → conservée (usage pagination).

---

## 4. Périmètre

**Dans le périmètre :**
- Backend `apps/go-api` : agrégats Min/Max (DEC-MINMAX) — `domain/explorer_briefing.go`
  (`ExplorerBriefingScope` + 3 champs), `service/match_history_service_briefing.go`
  (`minScopeFloat` + `buildBriefingScope`), tests service, `api/openapi.yaml`
  (`min_kda`/`min_perf`/`max_perf`) + `make generate-types`.
- Frontend `apps/web` : `ExplorerBriefingStrip.tsx`, `ExplorerBriefingTiles.tsx`
  (`MinMaxTriptych`, accents, suppression `PeakKdaTile`), `BriefingTile.tsx` (centrage),
  `ExplorerBriefingModules.tsx` (Classement en rangée + compteurs), `ExplorerMatchesTable.
  highlight.ts` (décile + `cellStyle` paramétré), `ExplorerMatchesTable.tsx` (câblage décile +
  alignement), `MatchScoreboard.logic.ts` (param `intensityPct` défaulté), `ExplorerPage.
  matchesMode.tsx` (compteur/CSV), manifest `explorer.toml` (+ régénération), tests
  (`*.test.ts(x)`).
- Vérification (reprise UTILISATEUR), journal du plan, `.ai/thought_log.md`, changelog
  (`docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`).

**Hors périmètre (noter en §6 Découvertes si rencontré, NE PAS traiter) :**
- Tout changement des CALCULS baseline/dimensions/context/streaks/dominance/Classement
  par chaîne (inchangés depuis V4).
- La 2ᵉ copie inline du style best/worst dans `SquadImpactScoreboard.tsx` (dette V4
  Découverte-17 — signalée, non corrigée ici).
- Le split du god-file `ExplorerMatchesTable.tsx` (819 L — chantier séparé documenté V4
  Découverte-18 ; ce plan ne l'aggrave pas significativement : câblage décile ≈ neutre en
  lignes, alignement quelques lignes).
- Le masquage MMR global (chantier séparé) — ce plan n'omet la tuile Pic MMR que si
  `team_mmr` est absent.
- Dette lint pré-existante (baseline gelée) ; tout Python (interdit) ; SQLite (interdit).

**`-tags=integration` NON requis (justification).** DEC-MINMAX est en LECTURE/agrégation
mémoire pure : `MinKDA`/`MinPerf`/`MaxPerf` sont des min/max sur `r.KDA`/`r.PerformanceScore`
DÉJÀ scannés (aucune colonne SQL neuve, aucune requête modifiée). **Aucune écriture, aucun
writer/lease, aucune table créée, aucune migration.** Les tests anti-ART
(`-tags=integration`) couvrent les écritures per-match — sans objet ici. La suite standard
`cd apps/go-api && go test ./...` (inclut `platform/duckdb` en `:memory:` + CGO) +
`make go-api-lint` + `TestOpenAPISchemaDrift` suffit (règle CLAUDE.md : `-tags=integration`
OBLIGATOIRE seulement avant livraison sync/persist). Si l'exécution constate qu'un test repo
touché est gardé par le tag integration, le noter §6 et l'exécuter — mais aucune écriture
n'est planifiée.

---

## 5. Phases (ordre strict — une étape CLOSE avant la suivante)

> Clôture d'étape = gate passé (commandes exactes, sorties propres — jamais de test
> skippé/désactivé) + tous les items statués `[x]` fait / `[~]` couvert ailleurs (réf) /
> `[!]` non traité (justif écrite) + plan mis à jour (cases + journal) + entrée
> `.ai/thought_log.md` + point d'étape utilisateur. Aucune case vide à la clôture. Zéro fix
> hors périmètre (→ §6).
>
> Notes d'exécution : commandes `go` SÉQUENTIELLES (jamais concurrentes — cache Windows ;
> tuer les `link.exe` orphelins). Après toute édition de `explorer.toml` : `node
> apps/web/scripts/build_i18n_manifests.mjs` AVANT `make check-types`. vitest →
> `dangerouslyDisableSandbox=true`. Purger `apps/web/node_modules/.tmp` avant
> `make check-types`. Commits par phase (skill `plan-execution`) ; le superviseur exécute la
> clôture git (push `main` = deploy auto → seulement après revue visuelle utilisateur).

### Phase 0 — Cadrage & re-vérification sur pièces (rapide)  ✅ CLOSE (2026-07-17, bloc A)

- [x] Confirmer `git branch --show-current` = `feat/explorer-briefing-compact` (sinon la
      retrouver ; NE PAS reprendre sur `main` ni une branche de train). Ne toucher que ce que
      les phases autorisent. → **FAIT** : branche `feat/explorer-briefing-compact` ; `git status`
      propre hormis le plan V5 untracked. CGO OK (CGO_ENABLED=1, gcc 15.2.0 msys64 ucrt64).
- [x] Re-vérifier §2 sur pièces (rouvrir chaque fichier:ligne cité — le code a pu bouger
      depuis le 2026-07-17) ; consigner tout décalage en §6. → **FAIT** : §2.1 (backend), §2.2
      (socle), §2.3 (modules), §2.4 (page/table) re-vérifiés ; ancrages conformes. Décalages
      mineurs → Découverte-8/9/10 (§6).
- [x] Confirmer les défauts techniques tranchés : DEC-NEUTRAL (`outcome-draw`), DEC-DECILE
      teinte (paramétrage `intensityPct`), `MIN_DECILE_SAMPLE`=10, `DECILE_TINT_PCT`≈16.
      Consigner au journal. → **FAIT** : DEC-NEUTRAL=`outcome-draw` retenu (bloc A) ; params
      décile confirmés pour bloc B (Phase 4, hors bloc A courant).

Gate Phase 0 : branche correcte ; constat re-vérifié ; défauts confirmés. Pas de gate de
build (aucun code applicatif modifié). Pas de commit (ou commit doc du plan si le superviseur
le demande). → **PASSÉ**.

### Phase 1 — Backend : agrégats Min/Max (rapide-moyen) — DP-1 / DEC-MINMAX  ✅ CLOSE (2026-07-17)

- [x] **1a (DTO).** `ExplorerBriefingScope` (`explorer_briefing.go:58-81`) : ajouter `MinKDA
      *float64 json:"min_kda,omitempty"`, `MinPerf *float64 json:"min_perf,omitempty"`,
      `MaxPerf *float64 json:"max_perf,omitempty"` + commentaires (min FDA/perf du scope ;
      moy = `KDA`/`AvgPerf` ; max FDA = `PeakKDA`). → **FAIT** : 3 champs ajoutés (MinKDA après
      KDA, MinPerf/MaxPerf après AvgPerf) ; commentaires reliant chaque borne au triptyque
      (DP-1) ; PeakKDA re-commenté « borne HAUTE du triptyque FDA ».
- [x] **1b (service).** Créer `minScopeFloat(scope, get)` (symétrique de `maxScopeFloat`
      `:140-153`). Dans `buildBriefingScope` (`:105-118`) : `MinKDA = minScopeFloat(scope, r
      => r.KDA)`, `MinPerf = minScopeFloat(scope, r => r.PerformanceScore)`, `MaxPerf =
      maxScopeFloat(scope, r => r.PerformanceScore)`. Fonctions ≤ 80 L, fichier ≤ 500 L
      (vérifier ; extraire si dépassement). → **FAIT** : `minScopeFloat` ajouté (miroir strict,
      `<` au lieu de `>`) ; câblé dans `buildBriefingScope` ; commentaire `maxScopeFloat` étendu
      (sert aussi MaxPerf). Fichier service = 489 L (≤ 500) ; `buildBriefingScope` 21 L (≤ 80).
- [x] **1c (tests service).** Étendre `match_history_service_briefing*_test.go` : dataset
      hétérogène → `MinKDA` = min de `r.KDA`, `MaxPerf`/`MinPerf` = max/min de
      `PerformanceScore` ; ordre `min_kda ≤ kda ≤ peak_kda` et `min_perf ≤ avg_perf ≤
      max_perf` ; nil si le champ est absent partout (KDA/perf tous nil). Aucun test skippé.
      → **FAIT** : `TestBuildBriefingScope_MinMaxTriptych` (r.KDA = net natif → ordre garanti ;
      min/max/moy exacts) + `TestBuildBriefingScope_MinMaxNilWhenAbsent` (rows nues → tout nil).
      Verts (`go test ./internal/service/`).
- [x] **1d (OpenAPI + regen).** `api/openapi.yaml` : `min_kda`/`min_perf`/`max_perf` (number,
      nullable) sur `ExplorerBriefingScope` (`:5030`). `make generate-types` ; `types.ts`/
      `generated.ts` régénérés (`ExplorerBriefingScope` gagne `min_kda?`/`min_perf?`/
      `max_perf?`) ; `TestOpenAPISchemaDrift` = 0 MISSING/DIVERGENT sur `ExplorerBriefingScope`.
      → **FAIT** : émission Huma byte-exact via `OPENAPI_EMIT_DIVERGENT_OUT`+`PREFIX` (les 3
      champs `format: double, type: number`, SANS `nullable` — comme `avg_perf`), collée dans
      openapi.yaml (ordre alpha entre `matches` et `peak_kda`). `make generate-types` →
      generated.ts gagne `max_perf?/min_kda?/min_perf?` ; idempotent (re-run 0 diff).
      `TestOpenAPISchemaDrift` : MISSING=0, `ExplorerBriefingScope` HORS liste divergents.
      Note : `types.ts:824` = alias `components['schemas'][…]` → fields dans generated.ts (grep OK).

Gate Phase 1 : `cd apps/go-api && go test ./...` = 0 (SÉQUENTIEL ; test 1c inclus) ;
`make go-api-lint` = 0 ; `make generate-types` idempotent (re-run → 0 diff) ;
`TestOpenAPISchemaDrift` vert ; `make check-types` = 0 (cache `.tmp` purgé) ; grep :
`min_kda`/`min_perf`/`max_perf` présents dans `types.ts`.
→ **PASSÉ (2026-07-17)** : `go test ./...` EXIT=0 (111 ok / 0 fail) ; `make go-api-lint` EXIT=0
(go vet) ; `make generate-types` idempotent (+6 L stables) ; `TestOpenAPISchemaDrift` PASS
(MISSING=0, Scope non divergent) ; `make check-types` EXIT=0 (`.tmp` purgé) ; grep 3 champs OK
dans generated.ts (aliasé par `types.ts`).

### Phase 2 — Frontend : socle des tuiles (moyen) — DP-1 / DP-2 / DP-6 / DP-8 / DP-9  ✅ CLOSE (2026-07-17)

- [x] **2a (i18n).** `explorer.toml` : renommer `streaks_title` (DP-9 : FR « Séries
      marquantes » / EN « Notable streaks ») ; réviser `tip_fda`/`tip_perf` (décrire les trois
      valeurs, FR + EN). Régénérer les manifests (`build_i18n_manifests.mjs`). → **FAIT** :
      `streaks_title` = « Séries marquantes »/« Notable streaks » ; `tip_fda`/`tip_perf` décrivent
      « le plus bas · la moyenne (au centre, en couleur) · le plus haut » (FR+EN) ; commentaires de
      section MAJ. Manifests régénérés (explorer : 224 clés).
- [x] **2b (triptyque).** Créer `MinMaxTriptych` (`ExplorerBriefingTiles.tsx`, exporté,
      DEC-TRIPTYCH). Câbler dans les tuiles FDA et Perf : `value = <MinMaxTriptych …/>`.
      Conserver les `sub`. Bornes nulles omises ; moy nulle → « — ». → **FAIT** : `MinMaxTriptych`
      exporté (min/max `text-2xs font-normal muted`, mid hérite `text-xl`, coloré via `midColor` ;
      `inline-flex items-baseline justify-center`). FDA : min_kda/kda/peak_kda + `kdaNetColor` ;
      Perf : min_perf/avg_perf/max_perf + `getPerfColor`. `sub` (deltas) conservés.
- [x] **2c (retrait Pic FDA).** Supprimer `PeakKdaTile`, son rendu (`Strip`) et son import.
      RE-VÉRIFIER le grep `peak_fda_label`/`tip_peak_fda` → **purger** du toml + régénérer.
      `scope.peak_kda` reste consommé (max du triptyque FDA). → **FAIT** : composant + rendu +
      import supprimés ; grep confirmé (seuls lecteurs = PeakKdaTile + tests) ; `peak_fda_label` +
      `tip_peak_fda` purgés du toml + regen (absents de generated). `import kdaNetColor` retiré de
      Tiles.tsx (orphelin). Découverte-4 confirmée.
- [x] **2d (cascade ≤ 8).** `Strip` : supprimer `slice(0, 2)` → rendre `{conditionalTiles}`.
      5 base + 3 conditionnelles = 8 max, pas de trou. → **FAIT** : `cappedConditionals` supprimé,
      rendu direct de `{conditionalTiles}` ; commentaire MAJ (les 3 tiennent, Pic MMR revisible).
      Test « socle = 8 cellules » vert.
- [x] **2e (accents).** Perf = `perfScale(scope.avg_perf)` (fallback `outcome-draw` si nil) ;
      Durée / Séries marquantes / Pic rang / Pic MMR = `outcome-draw`. → **FAIT** : `perfScale`
      importé dans le Strip (accent Perf) ; `accent="outcome-draw"` sur DurationTile/StreaksTile/
      PeakRankTile/PeakMmrTile. Matchs/WR/FDA inchangés. Test « 8 barres 3px » vert.
- [x] **2f (centrage).** `BriefingTile` div valeur : `text-center`. WR : ajuster LOCALEMENT si
      désalignement. → **FAIT** : `text-center` ajouté (`mt-0.5 text-center text-xl …`). WR : le
      pourcentage (inline) se centre, le ruban (flex) reste pleine largeur → aucun ajustement local
      requis (revue visuelle utilisateur). Label non centré. Test « 5 divs text-center+text-xl » vert.
- [x] **2g (renommage vérifié).** `StreaksTile` lit toujours `streaks_title`. → **FAIT** :
      inchangé (`t('explorer.briefing.streaks_title')`) ; seul le libellé i18n a changé (2a).
- [x] **2h (tests).** `ExplorerBriefingStrip.test.tsx` : triptyques FDA/Perf, Pic FDA absente,
      cascade 3/3, accents, centrage, Séries marquantes. Aucun test skippé. → **FAIT** : nouveau
      describe « triptyques, accents & centrage » (5 tests) ; cascade réécrite (DP-2 : 3 rendues +
      cap 8) ; décomptes tooltips 12→11 / 6→5 ; tests Durée/Pic-FDA réécrits ; test Perf-couleur
      pré-existant corrigé (span coloré vs racine du triptyque). 31/31 verts.

Gate Phase 2 : `node …/build_i18n_manifests.mjs` (diff = clés attendues) ; `make check-types`
= 0 ; `make test-web` (dangerouslyDisableSandbox) vert ; `cd apps/web && npm run lint` = 0
erreur ; greps : 0 `PeakKdaTile`, 0 `peak_fda_label`/`tip_peak_fda` (toml + generated +
composants), 0 `slice(0, 2)` sur les conditionnelles du Strip, `MinMaxTriptych` importé par
FDA + Perf, `perfScale` importé dans le socle.
→ **PASSÉ (2026-07-17)** : manifests régénérés (224 clés) ; `make check-types` EXIT=0 ;
suite explorer (19 fichiers / 133 tests) verte (full `make test-web` en gate fin de bloc) ;
`npm run lint` EXIT=0 (0 erreur / 68 warnings baseline, 0 sur fichiers touchés) ; greps de
clôture OK (les 3 `peak_fda_label` restants = assertions `not.toContain` dans les tests).

### Phase 3 — Frontend : modules & page (moyen) — DP-3 / DP-10 / DP-7  ✅ CLOSE (2026-07-17)

- [x] **3a (Classement en rangée).** `ExplorerBriefingModules.tsx` : supprimer le conteneur
      `<div className="flex flex-col gap-2">` (`:112-119`) ; rendre `ContextSplitCard` (gaté
      `contextSplit != null`) et `RankedBlock` (gaté `showRanked`) comme enfants DIRECTS de la
      grille « Par… », après les `DimensionCard`. Adapter la condition de rendu de la grille
      (`:107`) et l'early-return (`:101`) : `showSplitColumn` → `(contextSplit != null ||
      showRanked)`. Harmoniser les hauteurs (`h-full` si nécessaire). Documenter tout ajustement
      du `minmax` (DEC-LAYOUT).
      → **FAIT** : wrapper `<div className="flex flex-col gap-2">` supprimé ; `ContextSplitCard`
      (gaté `contextSplit != null`) et `RankedBlock` (gaté `showRanked`) sont enfants DIRECTS de
      la grille (siblings des `DimensionCard`). `showSplitColumn` renommé `hasContextOrRanked`
      (nom reflétant la présence, plus l'empilement) ; early-return + condition grille adaptés.
      `h-full` ajouté à `ContextSplitCard` + `RankedBlock` (harmonisation hauteur cellule, comme
      DimensionCard). `minmax(240px,1fr)` INCHANGÉ (auto-fit absorbe déjà 3-5 cellules sans trou ;
      abaissement non nécessaire — laissé à la revue visuelle). Headers Modules + RankedBlock MAJ.
- [x] **3b (compteurs sans « matchs »).** DP-10/DEC-COUNT : `DimensionRow` et `ContextSplitRow`
      → afficher `{entry.matches}` / `{group.matches}` seul, avec `title=dim_matches`. → **FAIT** :
      les deux lignes rendent le nombre seul ; `title={t('…dim_matches',{n})}` sur le `<span>` ;
      `dim_matches` conservée (non orpheline). Grep : `dim_matches` uniquement en `title=`.
- [x] **3c (compteur/CSV).** DP-7/DEC-CSV : supprimer le `<p>` compteur ; déplacer le bouton CSV
      SOUS `<ExplorerMatchesTable>` dans un `flex justify-end` ; retirer la barre `justify-between`.
      `count_label` NON purgée. → **FAIT** : `<p>` compteur + barre `justify-between` supprimés ;
      export CSV rendu APRÈS le tableau (`div flex justify-end`). Commentaire reformulé (retrait du
      littéral `count_label` → grep page propre). Clé i18n `count_label` conservée (pagination).
- [x] **3d (tests).** Modules (Classement = grid sibling) ; `ExplorerPage.test.tsx` (compteur
      absent au-dessus / CSV sous le tableau) ; compteurs = nombre seul. Garde-rail terminologie
      vert. Aucun test skippé. → **FAIT** : `ExplorerBriefingStrip.test.tsx` — assertions « enfant
      DIRECT de la grille » pour « Par contexte » ET Classement + nouveau test « cellules SÉPARÉES
      (siblings) » + describe DP-10 (nombre + title). `ExplorerPage.test.tsx` — `ExplorerMatchesResultsBlock`
      exporté (seam) + test DP-7 (count_label absent, CSV dernier enfant/sous le tableau, table
      RÉELLE avec items vides → pas de mock global qui casserait le smoke). Garde-rail terminologie
      vert. 136 tests explorer verts.

Gate Phase 3 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ;
`build_i18n_manifests.mjs` (aucune clé neuve attendue) ; greps : 0 `flex flex-col gap-2`
empilant `RankedBlock` (Modules) ; `RankedBlock` = enfant direct de la grille ; 0
`dim_matches` rendu en texte visible (rendu = nombre + `title`) ; compteur `count_label`
absent de `ExplorerPage.matchesMode.tsx` ; CSV APRÈS `<ExplorerMatchesTable>`.
→ **PASSÉ (2026-07-17)** : `make check-types` EXIT=0 ; suite explorer 136 tests verte (full
`make test-web` en gate fin de bloc) ; `npm run lint` 0 erreur / 68 warnings (baseline
INCHANGÉE ; le warning `react-refresh` sur `normalizeExplorerTableRows` pré-existe en HEAD) ;
`build_i18n_manifests.mjs` aucune clé neuve ; greps de clôture tous verts.

### Phase 4 — Frontend : tableau (moyen) — DP-4 / DP-5 / DEC-DECILE / DEC-ALIGN

- [ ] **4a (décile — helper).** `ExplorerMatchesTable.highlight.ts` (DEC-DECILE) : remplacer
      `computeColumnExtremes` → `computeColumnDeciles(rows)` (p10/p90 nearest-rank,
      `MIN_DECILE_SAMPLE`=10) ; ajouter `decileCellState(value, d, inverted)` (bande ;
      `p10===p90` → neutre) ; `columnHighlightStyle` compose `cellStyle(decileCellState(…),
      DECILE_TINT_PCT)`. Ne PLUS importer `cellState` ; GARDER l'import `cellStyle`. Type
      `Deciles` local.
- [ ] **4b (teinte douce — `cellStyle`).** `MatchScoreboard.logic.ts:56-64` : ajouter
      `intensityPct: number = 28` à `cellStyle` (défaut = inchangé) ; utiliser `${intensityPct}
      %` dans le `color-mix`. `MatchScoreboard.test.ts` reste vert (défaut). L'Explorer passe
      `DECILE_TINT_PCT`≈16. (Alternative 28 % sans toucher au partagé → consigner §6 si
      retenue.)
- [ ] **4c (décile — câblage).** `ExplorerMatchesTable.tsx` : `highlightExtremes` →
      `highlightDeciles = useMemo(() => computeColumnDeciles(rows), [rows])` (`:242`) ;
      application `:767-775` avec `highlightDeciles`. Imports (`:55-60`) mis à jour.
- [ ] **4d (alignement par colonne).** DEC-ALIGN : retirer `text-left` de `HEADER_TH_CLASS`
      (`:129-130`) ; définir `RIGHT_ALIGNED_COLUMNS` (ids numériques RE-VÉRIFIÉS) +
      `alignClass(colId)` ; appliquer sur `<th>` statique (`:731`) + triable (`:741`,
      `justify-end` sur le bouton) et `<td>` (`:772-774`). Colonnes texte à gauche. Jamais
      centré.
- [ ] **4e (tests).** `ExplorerMatchesTable.highlight.test.ts` : réécrire pour les déciles —
      top 10 % surligné best, pire 10 % worst (dataset ≥ 10 valeurs hétérogènes) ; `deaths`
      inversé ; `< MIN_DECILE_SAMPLE` → aucun highlight ; `p10===p90` neutre ; nulls neutres.
      `ExplorerMatchesTable.test.tsx` : surlignage indépendant du tri ; en-têtes/cellules
      numériques `text-right`, texte `text-left`. `MatchScoreboard.test.ts` (param `cellStyle`
      défaulté) vert. Aucun test skippé.

Gate Phase 4 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ;
greps : 0 littéral `color-mix(in oklab` NOUVEAU sous `features/explorer` (teinte importée) ;
`cellStyle` importé par le sibling highlight ; `cellState` NON importé par le sibling ;
`computeColumnDeciles` remplace `computeColumnExtremes` ; `text-left` absent de
`HEADER_TH_CLASS`.

### Phase 5 — Clôture (rapide) — changelog + delivery-checklist

- [ ] **5a (changelog).** `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`, entrée `[Unreleased]`
      v7.0 : bullet React « Explorer — briefing V5 » (triptyques FDA/Perf min·moy·max, cascade
      3 conditionnelles, Classement en rangée, MVP/LVP par décile, alignement colonnes
      numériques, accents généralisés, valeurs centrées, compteur/CSV, « Séries marquantes ») +
      bullet Go (`min_kda`/`min_perf`/`max_perf` sur `ExplorerBriefingScope`). Parité EN/FR
      même commit.
- [ ] **5b (clôture).** Dérouler `delivery-checklist`. Passe finale des gates §1.11 verte en
      une fois. Entrée `.ai/thought_log.md` finale. Point d'étape utilisateur. NON committer la
      livraison finale sans autorisation (merge `main` = deploy prod auto → après revue
      visuelle utilisateur).

Gate Phase 5 : `cd apps/go-api && go test ./...` = 0 ; `make go-api-lint` = 0 ; `make
generate-types` idempotent ; `make check-types` = 0 ; `make test-web` vert ; `npm run lint`
= 0 erreur ; greps de clôture (PeakKdaTile, peak_fda_label, slice(0,2), computeColumnExtremes,
text-left dans HEADER_TH_CLASS = 0) ; changelog EN + FR à jour ; chaque item statué.

---

## 6. Découvertes (à remplir en exécution — ne pas traiter hors périmètre)

- **Découverte-1 (pré-notée) — DP-1 sans SQL.** `MinKDA`/`MinPerf`/`MaxPerf` s'obtiennent par
  min/max sur `r.KDA`/`r.PerformanceScore` DÉJÀ scannés (`match_history.go:42,49`). Aucune
  requête (`queries_career.go`) ni scan (`match_history_repo.go`) modifié → `-tags=integration`
  non requis (§4).
- **Découverte-2 (pré-notée) — ordre du triptyque FDA.** `scope.kda = AggregateKDA(Σk,Σa,Σd,N)
  = moyenne du net par-match` (`indicators.go:42`) et `min_kda`/`peak_kda` = extrêmes de
  `r.KDA` natif (`match_participants.kda`). `min ≤ moy ≤ max` tient si `r.KDA` natif =
  `k + a/3 − d` (définition Halo) ; un arrondi DB du `kda` stocké pourrait le mettre hors
  bornes de façon cosmétique. Le test 1c ASSERTE l'ordre sur dataset réaliste ; si un cas
  d'arrondi le viole, c'est acceptable (valeurs distinctes-mais-reliées, choix utilisateur) —
  consigner ici, ne pas « corriger » l'agrégat. Perf : `min ≤ moy ≤ max` EXACT (moyenne
  arithmétique des mêmes valeurs).
- **Découverte-3 (pré-notée) — `count_label` non orpheline.** DP-7 retire l'usage de
  `ExplorerPage.matchesMode.tsx` mais `count_label` reste consommée par le pied de pagination
  (`ExplorerMatchesTable.tsx:789`) → PAS de purge i18n. (Si un jour la pagination cesse de
  l'utiliser, réévaluer.)
- **Découverte-4 (pré-notée) — `peak_fda_label`/`tip_peak_fda` orphelinées.** La suppression de
  `PeakKdaTile` (2c) retire leur unique lecteur → purge en Phase 2 (RE-VÉRIFIER le grep avant).
- **Découverte-5 (pré-notée) — teinte partagée `cellStyle`.** DP-4 « teinte douce » exige de
  réduire l'intensité sur une BANDE (plus dense qu'un extrême unique). Paramétrer `cellStyle`
  (`MatchScoreboard.logic.ts`) avec `intensityPct` défaulté (28) préserve scoreboard/leaderboard
  ET garde la source unique du `color-mix` (CLAUDE.md §6). `cellState` (exact `===`) n'est pas
  réutilisable pour une bande → `decileCellState` local (le style seul est mutualisé).
- **Découverte-6 (pré-notée) — jeton neutre.** Pas de token « neutral »/« muted » générique
  (`semantic-tokens.ts`) ; `outcome-draw` (= `deltaToken(0)`, déjà l'accent Matchs) est le
  neutre établi → retenu (DEC-NEUTRAL). `divergent-neutral` existe (alternative si revue
  visuelle).
- **Découverte-7 (pré-notée) — numéros DP-7 corrigés.** La mission citait ~376-380 (compteur)
  et ~412-421 (CSV) ; l'état réel = `ExplorerPage.matchesMode.tsx:357-361` (compteur) et
  `:362-371` (CSV), barre `:356-372`. Corrigé dans §2.4 (constat sur pièces).

- **Découverte-8 (Phase 0) — offsets mineurs vérifiés.** Import `PeakKdaTile` dans le Strip
  est ligne **30** (bloc d'import 28-35), non `:29` comme indiqué DEC-CASCADE/§2.2. Trivial.
  Tous les autres ancrages §2.1-2.4 conformes au 2026-07-17.
- **Découverte-9 (Phase 0) — tests Modules dans le Strip.** Aucun fichier
  `ExplorerBriefingModules.test.tsx` : la couverture des modules (Classement en grille,
  « Par contexte ») vit dans `ExplorerBriefingStrip.test.tsx` (le Strip rend les Modules).
  Les tests décile/alignement sont dans `ExplorerMatchesTable*.test.ts(x)`. Phase 3d = mise à
  jour de `ExplorerBriefingStrip.test.tsx` (pas de nouveau fichier Modules).
- **Découverte-10 (Phase 0) — tests V4 à réécrire (Phase 2h).** `ExplorerBriefingStrip.test.tsx`
  encode le comportement V4 : cascade « au plus 2 » (`:378-404`, assert `not.toContain
  peak_mmr_label`) et tuile Pic FDA autonome (`:263-343`, décomptes tooltips 12/6 + assert
  `peak_fda_label`/`4.20`). DP-1 (retrait Pic FDA) et DP-2 (retrait cap 2) invalident ces
  assertions → réécriture en Phase 2h (nouveaux décomptes 11/5, triptyques, 3 conditionnelles).
- **Découverte-11 (Phase 0) — `make go-api-lint` = go vet ciblé.** La cible Makefile
  `go-api-lint` lance `go vet ./internal/domain/... ./internal/analysis/...` (pas golangci-lint,
  pas le package `service`). Le compile-check de `service` est couvert par `go test ./...`
  (vet inclus par défaut). Gate honoré via la cible make + la suite complète.
- **Découverte-12 (Phase 3, pré-notée) — commentaire stale `BriefingSectionCard.tsx:9-13`.**
  Dit « Classement et Séries sont des tuiles du socle » — inexact depuis V4 (le Classement est
  un `BriefingSectionCard` via `RankedBlock`). DP-3 le confirme (grid sibling). Fichier NON
  édité en Phase 3 (aucun changement de code requis) ; commentaire corrigé opportunément si le
  fichier est touché, sinon laissé (dette doc V4 pré-existante, hors périmètre strict).

Consigner ici tout décalage fichier:ligne vs §2, tout lecteur i18n inattendu, toute dette
repérée hors périmètre. Ne pas corriger dans ce chantier (hors items scopés).

---

## 7. Protocole de reprise de session

1. `git branch --show-current` doit être `feat/explorer-briefing-compact` (sinon la retrouver
   via `git log --oneline -10`). Ne jamais reprendre sur `main` ni une branche de train.
2. Lire ce fichier : la dernière phase dont le **Gate** est coché est close ; reprendre à la
   première non close. Les cases `[ ]` d'une phase non close = travail restant.
3. Lire l'entrée `.ai/thought_log.md` la plus récente de ce chantier (avancement + décisions,
   dont DEC-NEUTRAL, DEC-DECILE teinte/param, `MIN_DECILE_SAMPLE`, `DECILE_TINT_PCT`).
4. Re-vérifier sur pièces les fichier:ligne de la phase courante AVANT d'éditer ou de cocher
   (le code a pu bouger).
5. Ne jamais commencer une phase N+1 tant que le Gate de N n'est pas vert.

---

## 8. Effort estimé & dépendances

| Bloc | Phase | Effort | Couche |
|---|---|---|---|
| Cadrage + re-vérif | 0 | Rapide | git + plan |
| Agrégats Min/Max | 1 | Rapide-Moyen | domain + service + tests + OpenAPI |
| Socle tuiles (triptyques, cascade, accents, centrage, renommage) | 2 | Moyen | front + i18n |
| Modules & page (Classement en rangée, compteurs, CSV) | 3 | Moyen | front + i18n |
| Tableau (décile + alignement) | 4 | Moyen | front (highlight + logic partagé) + tests |
| Changelog + clôture | 5 | Rapide | docs + gates |

**Dépendances inter-phases** : Phase 2 (triptyques front) DÉPEND de Phase 1 (champs Scope
Min/Max + `types.ts` régénéré). Phases 3 et 4 sont frontend-only, indépendantes entre elles
et de la Phase 1 (aucun champ backend requis), et suivent la structure du socle (Phase 2 pour
un worktree cohérent, mais techniquement parallélisables). Phase 5 en dernier. **Points à
confirmer par l'utilisateur** : aucun blocage — les 10 DP sont TRANCHÉES ; les micro-défauts
techniques (neutre `outcome-draw`, teinte douce paramétrée ≈16 %, `MIN_DECILE_SAMPLE`=10) sont
appliqués tels quels et ajustables en revue visuelle. **Aucun déploiement prod** dans ce
chantier (le merge `main` = deploy auto reste la décision de l'utilisateur, après revue
visuelle).

---

## À vérifier visuellement par l'utilisateur (repris par l'utilisateur, PAS une tâche agent)

Sur l'Explorer mode Matchs (dev local `:8000`/vite), profils réels halo_infinite (LUSR) + un
titre H5 (dégradation) + un état low_sample + spot-check EN :

1. **Triptyques FDA & Perf** : trois valeurs par tuile (le plus bas, la moyenne mise en avant
   et colorée, le plus haut) ; lisibles/compacts ; « Pic FDA » autonome absente ; moyenne
   colorée (`kdaNetColor`/`getPerfColor`) ; bornes absentes → moyenne seule sans « — »
   parasite.
2. **Socle & cascade** : 5 tuiles de base + jusqu'à 3 conditionnelles (Séries marquantes, Pic
   rang, Pic MMR) — les 3 s'affichent quand présentes (jamais > 8, jamais de trou) ; Pic MMR
   de nouveau visible ; chaque tuile a un accent (neutre `outcome-draw` sans sentiment) ;
   valeurs de première ligne centrées (y compris ruban WR aligné).
3. **Classement en rangée** : cartes « Par… » + « Par contexte » + « Classement » sur une
   seule rangée (siblings, pas d'empilement, pas de 2ᵉ rangée en desktop) ; le bloc Classement
   s'allonge verticalement si multi-chaînes sans déborder ; compteurs de dimension/contexte =
   nombre seul (survol = « X matchs »).
4. **Tableau** : surlignage MVP/LVP en bande de décile (top 10 %/pire 10 %) sur tout le scope,
   teinte douce, indépendant du tri ; en-têtes ET cellules numériques alignés à DROITE, texte
   à gauche ; compteur « N matchs trouvés » retiré au-dessus ; bouton « Exporter CSV » sous le
   tableau.
5. **Terminologie** : FR sans anglicismes (« Séries marquantes ») ; EN en parité. Console 0
   erreur sur les 4 états. Puis décision de merge (`main` = deploy prod auto).
