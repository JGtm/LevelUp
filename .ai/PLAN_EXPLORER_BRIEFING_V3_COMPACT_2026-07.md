# PLAN — Explorer briefing V3 : compaction du bandeau (Variante B)

Statut : PLANIFIE (aucune ligne de code écrite — plan rédigé par l'architecte Opus).
Date : 2026-07-17.
Chantier précédent (V2, livré/mergé en local le 2026-07-16) :
`.ai/PLAN_EXPLORER_BRIEFING_V2_2026-07.md` (contrat V2). V1 :
`.ai/V7/PLAN_EXPLORER_BRIEFING_CARDS_2026-07.md`.

Branche cible d'implémentation : **`feat/explorer-briefing-compact`** (déjà la branche
courante ; NE PAS en changer, NE PAS committer sans autorisation du tour).

> Contrat d'exécution : ce plan s'exécute sous le skill **`plan-execution`** (ordre strict,
> une étape close avant la suivante, aucun report d'action exécutable maintenant, statut sur
> chaque item, zéro fix hors périmètre). En cas de divergence, le présent plan fait foi ; à
> défaut, le skill est le défaut. Avant de finaliser toute modification du plan : skill
> **`plan-review`**. Avant chaque commit : skill **`delivery-checklist`**. Code React/TS :
> skill **`frontend-patterns`** ; toute couleur : **`color-tokens`** ; code Go :
> **`arch-rules`**. Rappels transverses : tokens sémantiques UNIQUEMENT (aucun hex/classe
> Tailwind couleur dans `features/`/`components/`) ; seuils fichier ≤ 500 L / fonction ≤ 80 L
> / ≤ 5 params / complexité ≤ 12 ; FR sans anglicismes (« série » pas « streak », « Taux de
> victoire » pas « WR ») ; parité i18n FR/EN par typage ; **pas de commandes `go`
> concurrentes** (corruption du cache Windows — séquentiel, tuer les `link.exe` orphelins) ;
> vitest hors sandbox (`dangerouslyDisableSandbox=true`).

---

## 1. Objectif et critères de succès (mesurables)

**Objectif.** Compacter le bandeau de briefing de l'Explorer (mode Matchs) qui, en V2,
relègue le tableau de résultats trop bas : après la rangée « Par… », s'empilent 4 cartes
pleine largeur quasi vides (Tendance ~170 px, Classement ~70 px, Solo vs Escouade ~90 px,
Séries ~90 px) + Moments forts (~70 px) + la frise (64 px). Cible : bandeau **~300-330 px**
au lieu de ~850, en réorganisant l'information selon la **Variante B** validée par
l'utilisateur (2026-07-17) — SANS toucher aux calculs backend des modules
ranked/streaks/context/dominance/trend, et en respectant la règle « 0 code mort » (purge
complète de la frise `outcome_sequence`).

**Pourquoi.** Le contenu des 5 cartes pleine largeur est maigre (1-3 lignes chacune) mais
occupe toute la largeur et pousse le tableau sous la ligne de flottaison. La Variante B
range chaque information dans le gabarit qui lui convient : les indicateurs synthétiques
(Classement, Séries) deviennent des tuiles du socle ; la tendance devient une micro-courbe
dans la tuile Taux de victoire ; le contexte solo/escouade rejoint la grille « Par… » ; les
moments forts deviennent une bande de pastilles nue ; la frise disparaît.

**Critères de succès (tous vérifiables) :**

1. **Hauteur du bandeau ~300-330 px** en plein historique (profil réel, halo_infinite),
   mesurée via chrome-devtools (`getBoundingClientRect` du conteneur racine du Strip), en
   nette baisse vs V2 (~850 px). Le tableau des résultats remonte au-dessus de la ligne de
   flottaison sur un écran standard.
2. **Frise supprimée** : aucun `OutcomeSequenceTape` dans le DOM du briefing Explorer. Le
   composant `OutcomeSequenceTape` lui-même RESTE dans le dépôt (consommé par
   `RelationsRivalryCards`). Purge backend complète de `outcome_sequence` (DTO + type
   `ExplorerBriefingOutcome` + `buildOutcomeSequence` + const `maxOutcomeSequencePoints` +
   OpenAPI + `types.ts`) et frontend (`outcomeCodeToValue`, clés i18n `series_*`).
3. **Classement = tuile KPI** sur la rangée socle (via `BriefingTile`) : valeur = palier de
   FIN (ex. « Or VI » ; fin en placement → clé `placement_remaining`), sous-texte = « {TYPE}
   · depuis {palier début} · {−1.4} pt/match ». Gate `useCapability('ranked')` conservé.
   Aucune carte pleine largeur `RankedCard`.
4. **Séries = tuile KPI** : valeur bicolore « {best} V / {worst} D » (tokens
   `outcome-win`/`outcome-loss`), segment à zéro omis, tuile omise si les deux à zéro. Aucune
   carte pleine largeur `StreaksCard`.
5. **Tendance = micro-sparkline nue** intégrée à la tuile Taux de victoire (socle), SANS axes
   ni chrome `ChartCard`, hauteur ~24-32 px. Aucune carte pleine largeur `TrendCard`. Le DTO
   `trend` backend RESTE.
6. **Solo vs Escouade = 4e carte de la rangée « Par… »**, retitrée FR « Par contexte » / EN
   « By context » ; même gabarit de lignes que les dimensions. La grille passe à 4
   emplacements responsives. Carte omise si `context_split` absent (dégradation par omission).
7. **Moments forts = bande de pastilles NUE** (sans `BriefingSectionCard`, sans en-tête de
   carte) sous la rangée « Par… », préfixée d'un libellé discret muted (`highlights_title`).
   Réutilise `DOMINANCE_ITEMS`. Aucune carte pleine largeur `DominanceCard`.
8. **Rangée socle responsive 4 à 6 tuiles** (Matchs, Taux de victoire+sparkline, FDA, Perf,
   [Classement], [Séries]) — tuiles lisibles même à 4.
9. **low_sample inchangé** : socle 4 tuiles + mention ; Classement/Séries/sparkline s'omettent
   naturellement (le backend n'émet PAS ranked/streaks/context/dominance/trend sous LowSample
   — vérifié §2). Aucun module « Par… » ni bande.
10. **0 code mort** : greps de clôture verts (§5 gates) ; l'unique consommateur restant de
    `OutcomeSequenceTape` est `RelationsRivalryCards` ; les clés i18n orphelinées
    (`series_*`, `trend_title`, `streak_best`, `streak_worst` si confirmées sans lecteur) sont
    purgées ; le commentaire garde-rail de `BriefingSectionCard` est mis à jour (plus de
    « doc inversée »).
11. **Gates verts** : `make check-types` = 0 (cache `node_modules\.tmp` purgé en clôture) ;
    `make test-web` (vitest, `dangerouslyDisableSandbox=true`) vert ; `cd apps/web && npm run
    lint` = 0 erreur ; `cd apps/go-api && go test ./...` = 0 ; `make go-api-lint` = 0 ;
    `make generate-types` idempotent (0 diff résiduel après re-run) ; `TestOpenAPISchemaDrift`
    vert. `-tags=integration` NON requis (voir §4).
12. **Vérification NAVIGATEUR** (chrome-devtools, dev local `:8000`/vite) sur profils réels,
    FR + spot-check EN, quatre états : plein historique, scope filtré, titre H5 (dégradation),
    low_sample ; console 0 erreur ; captures consignées au journal.
13. **Changelog** : entrée `[Unreleased]` v7.0 mise à jour dans `docs/CHANGELOG.md` ET
    `docs/FR/CHANGELOG.md` (parité EN/FR dans le même commit).
14. **Tooltips de légende (amendement utilisateur 2026-07-17)** : une icône (i) accessible
    (composant canonique `InfoTooltip`, `components/ui/info-tooltip.tsx` — vérifié §2) avec
    texte explicatif i18n FR+EN sur : les 4 cartes-sections « Par… » (colonnes : n matchs,
    taux de victoire, delta « vs habituel », note = palier de performance 1..5 basé sur le
    score de performance personnel) ; les tuiles socle non évidentes (Taux de victoire
    [V-D-N], FDA [terme en entier + « l'agrégat n'est pas un simple quotient », formulation
    grand public — PAS la formule ADR brute], Perf. moyenne [score 0-100 relatif à
    l'historique personnel], Classement [signification du pt/match et de la progression de
    paliers], Séries) ; la bande « Moments forts » (signification des catégories de
    dominance). Les labels de tuiles restent COURTS (arbitrage superviseur : compacité
    d'abord — les termes en entier vivent dans les tooltips, pas dans les labels).
15. **FDA coloré partout** : toute valeur FDA affichée dans le bandeau est colorée via
    `kdaNetColor` (tokens) — notamment les lignes de la future carte « Par contexte »
    (`ContextSplitRow`, aujourd'hui muted non coloré — vérifié §2). Vérification finale :
    aucune surface du bandeau n'affiche un FDA non coloré.

---

## 2. Constat sur pièces — état actuel (fichier:ligne réels, vérifiés le 2026-07-17)

> Doctrine du projet : RE-VÉRIFIER chaque ancrage sur pièces AVANT de coder ET avant de
> cocher (le code a pu bouger depuis la rédaction). Les numéros ci-dessous sont l'état au
> 2026-07-17.

**Frontend — `apps/web/src/features/explorer/`**

- `ExplorerBriefingStrip.tsx` (214 L) :
  - Imports à retirer/ajuster : `OutcomeSequenceTape, type OutcomePoint` (`:13`),
    `outcomeCodeToValue` (`:24`).
  - `BriefingTile` (composant local, `:60-72`) : `KpiCard` + label (`text-3xs uppercase …`)
    + value (`text-xl font-bold`) + sub optionnel (`text-2xs text-muted-foreground`) + accent.
    **Pas de slot graphique** aujourd'hui.
  - `matchesCount = scope?.matches ?? briefing.outcome_sequence?.length ?? 0` (`:81`) —
    fallback frise à retirer.
  - `tapePoints` (`:83-86`) puis rendu frise `<OutcomeSequenceTape … labels={series_win/…}>`
    (`:190-202`).
  - Grille socle : `grid grid-cols-2 gap-2 sm:grid-cols-4` (`:99`), 4 tuiles fixes
    (Matchs `:101-106`, Taux de victoire `:109-140`, FDA `:143-166`, Perf `:169-187`).
  - `fullHistory = isFullHistoryScope(...)` (`:95`) passé à `ExplorerBriefingModules … hideDelta`
    (`:210`). Branche `low_sample` : socle + `<p>` mention, sinon Modules (`:205-211`).
- `ExplorerBriefingModules.tsx` (425 L) :
  - `useCapability('ranked')` (`:99`), `showRanked = hasRanked && briefing.ranked != null`
    (`:101`).
  - Grille dimensions : `grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3` (`:123`),
    `DimensionCard` via `BriefingSectionCard` (`:142-161`), `DimensionRow` avec colonne delta
    gatée `!hideDelta` (`:163-209`).
  - `TrendCard` (`:213-234`) = `TimeseriesLineChart` `title=trend_title`, `height={120}` —
    **carte pleine largeur à supprimer**, remplacée par la sparkline dans la tuile Taux de
    victoire.
  - `RankedCard` (`:238-250`) + `rankedProgression` (`:255-266`) + `RankedKindRow`
    (`:268-300`) — **à convertir en tuile socle** ; `rankedProgression`/`RankedKindRow` à
    déplacer vers le fichier tuiles.
  - `ContextSplitCard` (`:307-316`) + `ContextSplitRow` (`:318-346`), titre `context_split_title`
    (`:309`) — **à déplacer dans la grille « Par… »** (4e carte, retitrée). **FDA de
    `ContextSplitRow` rendu NON coloré** (`:341-343` : `text-muted-foreground` +
    `group.kda.toFixed(2)`) alors que la tuile socle FDA est colorée (`kdaNetColor`,
    Strip `:147`) → amendement 2026-07-17 : colorer via `kdaNetColor` (le fichier n'importe
    aujourd'hui que `winRateColor`, `:20` — ajouter l'import).
  - `StreaksCard` (`:353-384`) titre `streaks_title`, lignes `streak_best`/`streak_worst` +
    valeurs `streak_wins`/`streak_losses` — **à convertir en tuile socle** (valeur compacte).
  - `DominanceCard` (`:390-424`) via `BriefingSectionCard`, `DOMINANCE_ITEMS` (`:70-84`),
    titre `highlights_title` (`:400`) — **à convertir en bande nue**.
  - Garde early-return (`:110-118`) référence dimensions/trend/ranked/context/streaks/dominance
    — **à réduire** (Modules ne gère plus que dimensions + context + dominance).
- `ExplorerBriefing.logic.ts` : `outcomeCodeToValue` (`:56-67`) + import `OutcomeValue`
  (`:8`) — **deviennent morts après retrait de la frise** (seul lecteur = Strip). Helpers
  `isFullHistoryScope`/`signOf`/`formatSignedFixed`/`formatSignedPoints` CONSERVÉS.
- `ExplorerBriefing.logic.test.ts` : `describe('outcomeCodeToValue')` (`:56-62`) + import
  (`:7`) — à retirer avec le helper.
- `ExplorerBriefingStrip.test.tsx` : `outcome_sequence: []` dans `makeBriefing` (`:46`) ;
  tests contexte (`:83-105`), Séries (`:107-131`, assertions `streaks_title`/`streak_best`/
  `streak_worst`), Moments forts (`:133-153`, `highlights_title`/`×N`) — **à réécrire** pour
  les nouveaux gabarits (tuiles / bande).
- `BriefingSectionCard.tsx` (55 L) : chrome + en-tête bordurée (calqué `ChartCard`).
  Commentaire garde-rail (`:11-19`) qui liste « classement par type de rating, contexte
  solo/escouade, séries, moments forts » comme devant passer par ce wrapper — **deviendra
  faux** (seuls dimensions + contexte y restent) → mettre à jour (anti-pattern « doc
  inversée », CLAUDE.md §Diagnostic-9). Composant CONSERVÉ (dimensions + contexte).

**Primitive sparkline réutilisable — DÉCOUVERTE (hors `components/charts/`)**

- Le brief demandait de chercher une primitive sparkline nue dans
  `apps/web/src/components/charts/` : **il n'y en a PAS** (le README y catalogue 11 wrappers
  ECharts ; aucun n'est une micro-courbe nue). MAIS une primitive pure et testée existe
  ailleurs : `apps/web/src/features/admin/sync/Sparkline.tsx` (`:9-40`, SVG inline
  `<polyline>` + dot, sans axes ni tooltip, couleur par token, props `values/token/width/
  height/ariaLabel`) + géométrie pure `sparklineGeometry.ts` (`sparklinePoints`/`lastPoint`,
  testée `sparklineGeometry.test.ts`). Consommateurs actuels : `PostSyncMatrix.tsx:11`,
  `SyncCycleHistory.tsx:10`. → décision SPARK-1 (§3).
- `apps/web/src/features/palmares/PalmaresRelationsPage.tsx` porte un `OutcomeSparkline`
  distinct (bande d'issues W/L catégorielle, PAS une courbe de valeur) — pattern différent,
  ne pas confondre.

**Composant tooltip canonique — TROUVÉ (amendement 2026-07-17, vérifié sur pièces)**

- `apps/web/src/components/ui/info-tooltip.tsx` : `InfoTooltip({ content: ReactNode,
  iconClass? })` (`:17-57`) — icône (i) en `<button type="button">` FOCUSABLE, ouverture au
  hover/focus/clic, fermeture blur/clic extérieur, `aria-label` résolu via la clé commune
  `common.tooltip.more_info_aria` (manifest `common`), panneau `role="tooltip"` (`:48`),
  styles par tokens (`border-input`, `text-muted-foreground`, `bg-background`). Déjà consommé
  par ~10 surfaces (HomePage, CareerRankingBlock, SquadEfficiencyChart, MatchSummaryCharts,
  TimeseriesPage.progression, SessionChartStack, …) → **RÉUTILISER tel quel, ne rien créer**.
- Points d'accroche existants : `BriefingSectionCard.title` accepte déjà un `ReactNode`
  « pour pouvoir injecter un InfoTooltip à côté du libellé » (`BriefingSectionCard.tsx:24-28`,
  prévu au plan V2 Phase 3a) ; `ChartCard.title` idem (`ChartCard.tsx:44-45`). `BriefingTile`
  (label `string`, Strip `:60-72`) n'a PAS de slot — à prévoir lors de l'extraction DEC-1.

**i18n — `apps/web/src/lib/i18n/manifests/explorer.toml`** (régénérer via
`node apps/web/scripts/build_i18n_manifests.mjs`)

- `matches_label` (`:841`) « Matchs »/« Matches » ; `record_label` (`:845`) « Bilan »/
  « Record » ; `win_rate_label` (`:853`) « Taux de victoire »/« Win rate » (label de la tuile
  qui accueillera la sparkline) ; `vs_baseline` (`:865`).
- `series_win/loss/tie/dnf` (`:873-887`) : **labels de la frise. Seul lecteur = Strip**
  (vérifié : grep `series_win|…` ne renvoie que `explorer.toml`, `generated/explorer.ts`,
  `ExplorerBriefingStrip.tsx`) → **purgeables** après retrait de la frise.
- `trend_title` (`:907`) : seul lecteur = `TrendCard` → **purgeable** après conversion en
  sparkline nue (à re-vérifier sur pièces).
- `ranked_title` (`:911`) « Classement »/« Ranking » (label tuile) ; `ranked_per_match`
  (`:915`) « {delta} pt/match » (réutilisé) ; `placement` (`:919`) / `placement_remaining`
  (`:923`) (réutilisés, D-D V2) — CONSERVÉS.
- `context_split_title` (`:927`) « Solo vs Escouade »/« Solo vs Squad » → **à modifier** en
  FR « Par contexte » / EN « By context ».
- `streaks_title` (`:933`) « Séries »/« Streaks » (label tuile, CONSERVÉ) ; `streak_best`
  (`:937`) / `streak_worst` (`:941`) « Meilleure/Pire série » → **purgeables** (plus de lignes
  dans la tuile compacte, à re-vérifier) ; `streak_wins` (`:946`) « {n} V » / `streak_losses`
  (`:950`) « {n} D » (réutilisés dans la valeur de la tuile, CONSERVÉS).
- `highlights_title` (`:954`) « Moments forts »/« Highlights » (label de la bande nue,
  CONSERVÉ).
- **Clé à AJOUTER** : `ranked_since` — FR « depuis {tier} » / EN « since {tier} » (sous-texte
  de la tuile Classement).
- **Clés tooltip à AJOUTER (amendement, liste FERMÉE — Phase 5b)** :
  `explorer.briefing.tip_dimensions` (colonnes des 3 cartes dimensions),
  `explorer.briefing.tip_context` (colonnes de « Par contexte », FDA au lieu de la note),
  `explorer.briefing.tip_win_rate`, `explorer.briefing.tip_fda`,
  `explorer.briefing.tip_perf`, `explorer.briefing.tip_ranked`,
  `explorer.briefing.tip_streaks`, `explorer.briefing.tip_highlights` — FR + EN, FR sans
  anglicismes (formulations par défaut en DP-9/DEC-7).

**Backend — Go**

- `internal/domain/explorer_briefing.go` : champ `OutcomeSequence []ExplorerBriefingOutcome`
  (`:34`) + struct `ExplorerBriefingOutcome` (`:75-81`) → **à supprimer**. Types `Ranked`
  (`:136-167`), `ContextSplit` (`:174-191`), `Streaks` (`:198-203`), `Dominance` (`:209-215`),
  `Trend` (`:117-129`) → **CONSERVÉS** (aucun changement de calcul).
- `internal/service/match_history_service_briefing.go` : const `maxOutcomeSequencePoints = 60`
  (`:49`) ; `b.OutcomeSequence = buildOutcomeSequence(filtered)` (`:71`) ; fonction
  `buildOutcomeSequence` (`:110-133`) → **à supprimer**. **Confirmation low_sample** :
  `buildExplorerBriefing` fait `if b.LowSample { return b }` (`:72-75`) AVANT de peupler
  Baseline/Dimensions/Trend/Ranked/ContextSplit/Streaks/Dominance → tous nil sous low_sample
  (critère §1.9 garanti côté backend, aucune modif nécessaire).
- `internal/service/match_history_service_briefing_streaks.go:5` : commentaire qui référence
  « frise outcome_sequence, cappée à maxOutcomeSequencePoints » → à corriger (la const
  disparaît).
- `internal/service/match_history_service_briefing_test.go` : contrôle `len(b.OutcomeSequence)`
  (`:61-62`) dans un test de base + `TestBuildExplorerBriefing_OutcomeSequenceCappedAndSorted`
  (`:531-544`) → **à supprimer/ajuster**.
- `api/openapi.yaml` : propriété `outcome_sequence` de `ExplorerBriefing` (`:5017-5021`) +
  schéma `ExplorerBriefingOutcome` (`:5144-5159`) → **à supprimer** (émission exacte via
  `OPENAPI_EMIT_OUT` puis alignement ; drift test `TestOpenAPISchemaDrift` valide, cf.
  `internal/api/openapi_schema_drift_test.go:113-132`).
- `apps/web/src/lib/api/types.ts` : `ExplorerBriefingOutcome` (`:828`) + champ
  `outcome_sequence` de `ExplorerBriefing` → régénérés par `make generate-types`.

**Changelog** — `docs/CHANGELOG.md` (`[Unreleased] - 2026-06-15`, `:9`) et `docs/FR/CHANGELOG.md`
(`[Non publié]`, `:9`) : sections « Added (React / TypeScript) » (`:13`, bullet « Explorer —
briefing V2 » `:17`) et « Added (Go API) » (`:41`). V3 ajoute un bullet React (compaction) et
complète le bullet Go (retrait `outcome_sequence`).

**Conclusion du constat.** Le chantier est **frontend d'abord** (réorganisation pure de la
présentation : les 5 DTO modules restent inchangés), suivi d'une **purge backend ciblée** du
seul `outcome_sequence` (le DTO le plus mort après retrait de la frise), puis vérif
navigateur. Aucune modification des calculs ranked/streaks/context/dominance/trend. Une
primitive sparkline réutilisable existe (admin/sync) → promotion + réutilisation plutôt qu'une
3e copie manuelle (§3 SPARK-1).

---

## 3. Décisions — pré-tranchées (fermes, ne pas re-débattre en exécution)

### Décisions produit (utilisateur, 2026-07-17 — reportées telles quelles)

- **DP-1 (frise).** `OutcomeSequenceTape` retiré du bandeau. Le composant RESTE (utilisé par
  `RelationsRivalryCards`). Purge backend complète de `outcome_sequence` (règle « 0 code
  mort »).
- **DP-2 (Classement → tuile).** Tuile socle via `BriefingTile`. Valeur = palier de FIN
  (placement en fin → `placement_remaining`). Sous-texte = « {TYPE} · depuis {palier début} ·
  {±x} pt/match » (réutiliser `ranked_since` [neuve], `ranked_per_match`, `placement`).
  Dégradations : paliers absents → valeur = pt/match seul ; multi-type (rare) → type
  MAJORITAIRE (première entrée de `kinds`, déjà ordonnée Count desc) en valeur/sous-texte,
  type secondaire sur une 2e ligne de sous-texte compacte — JAMAIS mélanger les paliers de
  deux systèmes (invariant P-3 du plan V2). Le préfixe type (« LUSR ») reste visible en tête
  du sous-texte. Gate `useCapability('ranked')` conservé. `RankedCard` pleine largeur
  SUPPRIMÉE.
- **DP-3 (Séries → tuile).** Tuile socle. Valeur = « {best} V / {worst} D » bicolore (tokens
  `outcome-win`/`outcome-loss`), segment à zéro omis ; tuile omise si les deux à zéro.
  `StreaksCard` pleine largeur SUPPRIMÉE.
- **DP-4 (Solo vs Escouade → 4e carte « Par… »).** Retitrée FR « Par contexte » / EN « By
  context » (modifier `context_split_title`), même gabarit de lignes que `DimensionRow`
  (libellé, n matchs, taux de victoire coloré, FDA). La grille des dimensions passe à 4
  emplacements. `ContextSplitCard` déménage dans cette grille (garde `BriefingSectionCard`).
- **DP-5 (Moments forts → bande nue).** Sans `BriefingSectionCard`, sans en-tête de carte,
  sous la rangée « Par… », préfixée d'un libellé discret muted (`highlights_title`). Réutilise
  `DOMINANCE_ITEMS`. `DominanceCard` SUPPRIMÉE (remplacée par la bande).
- **DP-6 (Tendance → micro-sparkline dans la tuile Taux de victoire).** Courbe du taux de
  victoire par bucket depuis `briefing.trend.points`, SANS axes ni chrome `ChartCard`, hauteur
  ~24-32 px. `TrendCard` pleine largeur SUPPRIMÉE. Le DTO `trend` backend RESTE. Couleur via
  token sémantique (`outcome-win`).
- **DP-7 (low_sample inchangé).** Socle 4 tuiles + mention ; Classement/Séries/sparkline
  s'omettent naturellement (backend nil sous LowSample — vérifié §2).
- **DP-8 (rangée socle responsive).** 4 à 6 tuiles selon les blocs présents, tuiles lisibles à
  4 (grille définie en DEC-3).
- **DP-9 (tooltips de légende — amendement 2026-07-17).** Le bandeau n'a aujourd'hui aucune
  légende et assume que l'utilisateur comprend tout. Ajouter un tooltip accessible via une
  icône (i) sur :
  - les 4 cartes-sections « Par… » (carte/mode/sélection/contexte) : expliquer les colonnes
    (n matchs, taux de victoire, delta « vs habituel », la note = palier de performance 1..5
    basé sur le score de performance personnel) ;
  - les tuiles socle non évidentes : FDA (terme en entier + préciser que l'agrégat n'est pas
    le quotient — formulation grand public, pas la formule ADR brute), Taux de victoire
    (V-D-N), Perf. moyenne (score 0-100 relatif à l'historique personnel), Classement
    (signification du pt/match et de la progression de paliers), Séries ;
  - la bande « Moments forts » (signification des catégories de dominance).
  Composant : réutiliser le canonique `InfoTooltip` (trouvé §2). Tous les textes en i18n
  FR + EN (clés dédiées `tip_*`, FR sans anglicismes).
- **DP-10 (FDA coloré partout — amendement 2026-07-17).** La tuile socle FDA est déjà colorée
  (`kdaNetColor`) ; les lignes de `ContextSplitRow` (futur « Par contexte ») rendent le FDA
  en muted non coloré → le colorer via `kdaNetColor` (tokens). Vérifier qu'aucune autre
  surface du bandeau n'affiche un FDA non coloré.
- **DP-11 (acronymes — arbitrage superviseur 2026-07-17).** Les tuiles gardent les libellés
  COURTS (l'objectif du chantier est la compacité) ; les termes en entier + explications
  vivent dans les tooltips (DP-9). Ne pas allonger les labels de tuiles.

### Décisions techniques (architecte)

- **SPARK-1 (source de la sparkline — DÉFAUT FERME : promotion + réutilisation).** Réutiliser
  la primitive existante `Sparkline` + `sparklineGeometry` (pure, testée) plutôt que d'écrire
  une 3e sparkline à la main (CLAUDE.md §14 « vérifier l'existant avant d'implémenter » + §6
  « ≤ 2 copies »). Comme elle vit dans `features/admin/sync/` (un import cross-feature depuis
  `features/explorer/` est un anti-pattern), la **promouvoir** vers un emplacement partagé
  `apps/web/src/components/charts/` : `git mv` de `Sparkline.tsx` + `sparklineGeometry.ts` +
  `sparklineGeometry.test.ts`, mise à jour des 2 imports admin (`PostSyncMatrix.tsx:11`,
  `SyncCycleHistory.tsx:10`) et ajout d'une sous-section « Primitives SVG pures (non-ECharts) »
  au `components/charts/README.md`. Usage explorer : géométrie FIXE (`width≈120`, `height=28`)
  dans la tuile — pas de responsive à ajouter à la primitive (une largeur fixe modeste suffit
  dans une tuile ≥ ~140 px). *Repli documenté (n'activer QUE si l'utilisateur refuse de
  toucher `features/admin/`) : SVG inline auto-suffisant dans le composant tuile — à signaler
  au point d'étape comme décision utilisateur ; le DÉFAUT reste la promotion.*
- **DEC-1 (extraction `BriefingTile`).** `BriefingTile` (aujourd'hui local au Strip) est
  extrait dans `apps/web/src/features/explorer/BriefingTile.tsx` (+ un slot optionnel
  `chart?: ReactNode` rendu sous la valeur, au-dessus du sous-texte). Nécessaire pour éviter un
  cycle d'import : les tuiles Classement/Séries (nouveau fichier) importent `BriefingTile` que
  le Strip importe aussi. Le Strip continue de composer les 4 tuiles de base avec ce composant.
- **DEC-2 (tuiles Classement/Séries).** Nouveau fichier
  `apps/web/src/features/explorer/ExplorerBriefingTiles.tsx` exportant `RankedTile` et
  `StreaksTile` (composant `BriefingTile`). `rankedProgression` (helper de composition
  « début → fin », D-C/D-D) est déplacé de `ExplorerBriefingModules.tsx` vers ce fichier.
  `useCapability('ranked')` est appelé dans le Strip (hook, AVANT le `return null` précoce si
  `!briefing`) et gate `RankedTile`.
- **DEC-3 (grilles responsives).**
  - Socle (4-6 tuiles) : `grid gap-2 grid-cols-2
    sm:[grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]` (auto-fit → tuiles lisibles
    à 4, remplissage propre à 5-6 ; l'exécutant peut ajuster le `minmax` à la revue visuelle).
  - « Par… » (3 ou 4 cartes) : `grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4`. Les
    cartes de dimension + la carte contexte (si présente) sont rendues dans cette même grille.
- **DEC-4 (répartition Strip ↔ Modules après compaction).**
  - `ExplorerBriefingStrip.tsx` : rend la rangée socle (4 tuiles de base — la tuile Taux de
    victoire porte la sparkline via `chart` — + `RankedTile`/`StreaksTile` conditionnelles),
    puis (low_sample ? mention : `ExplorerBriefingModules`).
  - `ExplorerBriefingModules.tsx` : rend UNIQUEMENT la grille « Par… » (dimensions + contexte)
    puis la bande « Moments forts ». Supprime `TrendCard`, `RankedCard`, `StreaksCard` et leurs
    imports (`TimeseriesLineChart`, types `ExplorerBriefingRanked`/`…RankedKind`/`…Trend`/
    `…Streaks` migrent vers le Strip/Tiles selon usage). `useCapability`/`showRanked` retirés
    de Modules. Garde early-return réduite à `dimensions.length===0 && contextSplit==null &&
    !showDominance`.
- **DEC-5 (parité i18n).** Toute clé ajoutée/modifiée l'est en FR ET EN dans `explorer.toml`,
  suivie de `node apps/web/scripts/build_i18n_manifests.mjs` AVANT `make check-types`. Toute
  clé orphelinée par une phase est purgée DANS cette phase (grep de clôture = 0), CLAUDE.md §7.
- **DEC-6 (backend inchangé hors purge).** Seul `outcome_sequence` est purgé côté Go. Aucune
  autre modification de `match_history_service_briefing*.go`, `explorer_briefing.go`,
  `kpi_stats.go` — les calculs ranked/streaks/context/dominance/trend restent identiques.
- **DEC-7 (implémentation des tooltips — DP-9).** Réutiliser `InfoTooltip`
  (`components/ui/info-tooltip.tsx`) SANS le modifier (accessible : bouton focusable,
  aria-label commun, `role="tooltip"`). Points d'injection :
  - Cartes « Par… » : via le slot `title: ReactNode` de `BriefingSectionCard` (prévu à cet
    effet, V2 Phase 3a) — `title={<span className="inline-flex items-center gap-1.5">{t(titleKey)}
    <InfoTooltip content={t(tipKey)} /></span>}`. `tip_dimensions` partagé par les 3 cartes
    dimensions (colonnes identiques) ; `tip_context` dédié à « Par contexte » (FDA au lieu de
    la note).
  - Tuiles socle : ajouter à `BriefingTile` (extrait en DEC-1) une prop optionnelle
    `info?: ReactNode` rendue à côté du label (rangée `inline-flex items-center gap-1`) ;
    passer `<InfoTooltip content={t('explorer.briefing.tip_…')} iconClass="w-3.5 h-3.5" />`
    depuis le Strip / les tuiles. Tuiles couvertes : Taux de victoire, FDA, Perf, Classement,
    Séries — PAS la tuile Matchs (évidente, liste fermée DP-9).
  - Bande « Moments forts » : `InfoTooltip` accolé au libellé muted (`highlights_title`).
  Textes par DÉFAUT (l'exécutant peut affiner la formulation, pas le contenu — FR sans
  anglicismes, EN en parité ; le tooltip FDA reste grand public, sans formule ADR brute) :
  - `tip_dimensions` FR : « Vos meilleurs et moins bons terrains sur le scope affiché. Par
    ligne : nombre de matchs, taux de victoire, écart face à votre habitude (masqué quand
    tout l'historique est affiché), et une note de performance de Excellent à Mauvais fondée
    sur votre score de performance personnel. »
  - `tip_context` FR : « Vos résultats selon que vous jouiez seul ou en escouade : nombre de
    matchs, taux de victoire et FDA de chaque contexte. »
  - `tip_win_rate` FR : « Part de matchs gagnés sur le scope affiché. Dessous : victoires,
    défaites, nuls — et la mini-courbe montre l'évolution du taux de victoire sur la
    période. »
  - `tip_fda` FR : « Frags, Décès, Assistances : un indicateur d'impact par match qui
    valorise les frags et les assistances et pénalise les décès — ce n'est pas une simple
    division des frags par les décès. »
  - `tip_perf` FR : « Score de performance de 0 à 100, calculé par rapport à votre propre
    historique : 50 = votre niveau habituel. »
  - `tip_ranked` FR : « Votre palier de classement en fin de scope, le palier de départ, et
    la moyenne de points de classement gagnés ou perdus par match. »
  - `tip_streaks` FR : « Vos séries extrêmes sur le scope : la plus longue suite de
    victoires et la plus longue suite de défaites. »
  - `tip_highlights` FR : « Vos matchs marquants : dominations (large victoire), humiliations
    (large défaite), remontadas (victoire après avoir été mené), débandades (défaite après
    avoir mené), contre-remontadas (remontée adverse stoppée). »

---

## 4. Périmètre

**Dans le périmètre :**
- Frontend `apps/web` : `ExplorerBriefingStrip.tsx`, `ExplorerBriefingModules.tsx`,
  `ExplorerBriefing.logic.ts` (+ test), `ExplorerBriefingStrip.test.tsx`,
  `BriefingSectionCard.tsx` (commentaire), nouveaux fichiers `BriefingTile.tsx` +
  `ExplorerBriefingTiles.tsx`, promotion de `Sparkline`/`sparklineGeometry` vers
  `components/charts/` (+ README + 2 imports admin), manifest `explorer.toml` (+ régénération).
  Tooltips (amendement) : RÉUTILISATION de `components/ui/info-tooltip.tsx` SANS le modifier ;
  clés `tip_*` dans `explorer.toml` ; coloration FDA de `ContextSplitRow` (`kdaNetColor`).
- Backend `apps/go-api` : purge `outcome_sequence` uniquement — `internal/domain/
  explorer_briefing.go`, `internal/service/match_history_service_briefing.go` (+ commentaire
  streaks) + tests, `api/openapi.yaml` (+ `make generate-types`, drift test), `types.ts`
  (régénéré).
- Vérification navigateur, journal du plan, `.ai/thought_log.md`.
- Changelog : `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`, entrée `[Unreleased]` v7.0.

**Hors périmètre (noter en §6 Découvertes si rencontré, NE PAS traiter) :**
- **Tri par en-têtes de colonnes + surlignage MVP/LVP du tableau** — chantier séparé (§6 du
  plan V2, « lisibilité des extrêmes », 2 lots). Aucune retouche du tableau des matchs.
- Tout changement des CALCULS backend ranked/streaks/context/dominance/trend (DEC-6).
- Toute refonte du socle V1/V2 au-delà de la compaction (deltas masqués, tri dimensions,
  helpers de format restent tels quels).
- La primitive `OutcomeSequenceTape` elle-même (conservée) et son consommateur
  `RelationsRivalryCards` (inchangé).
- Le `OutcomeSparkline` de `PalmaresRelationsPage` (pattern distinct, non concerné).
- Dette lint pré-existante (baseline gelée — 68 warnings) ; tout Python (interdit).
- `record_label` (« Bilan », clé peut-être inutilisée) : hors périmètre — non orphelinée par
  ce chantier ; simple observation en §6 si confirmée.

**`-tags=integration` NON requis (justification).** Ce chantier ne touche AUCUN chemin de
persist/sync/DuckDB : côté Go, seule une purge de champ DTO + fonction d'agrégation lecture
(`buildExplorerBriefing` agrège des raw rows en mémoire, aucun writer, aucune table, aucune
lease). Les tests anti-ART (`-tags=integration`) couvrent les écritures — sans objet ici. La
suite standard `go test ./...` + `make go-api-lint` + drift test suffit (règle CLAUDE.md :
`-tags=integration` OBLIGATOIRE seulement avant livraison sync/persist).

---

## 5. Phases (ordre strict — une étape CLOSE avant la suivante)

> Clôture d'étape = gate passé (commandes exactes, sorties propres — jamais de test
> skippé/désactivé) + tous les items statués `[x]` fait / `[~]` couvert ailleurs (réf) /
> `[!]` non traité (justif écrite) + plan mis à jour (cases + journal) + entrée
> `.ai/thought_log.md` + point d'étape utilisateur. Aucune case vide à la clôture. Zéro fix
> hors périmètre (→ §6). Ordre choisi : promotion sparkline (enabler isolé) → migration des
> blocs vers le socle (sparkline + tuiles, thème « pleine largeur → socle ») → réorg de la
> rangée « Par… » + bande → retrait frise (front) → purge backend → tooltips de légende
> (Phase 5b, sur la structure FINALE du bandeau) → vérif navigateur. Chaque phase frontend
> laisse le bandeau VISUELLEMENT COHÉRENT (pas de double-rendu d'un même bloc).
>
> Notes d'exécution : vitest hors sandbox → `dangerouslyDisableSandbox=true`. Après toute
> édition de `explorer.toml` : `node apps/web/scripts/build_i18n_manifests.mjs` AVANT
> `make check-types`. Commandes `go` SÉQUENTIELLES (jamais concurrentes — cache Windows).

### Phase 0 — Cadrage & re-vérification du constat (rapide)

- [x] Confirmer `git branch --show-current` = `feat/explorer-briefing-compact` (sinon la
      retrouver ; ne jamais reprendre sur `main` ni une branche de train). Worktree propre.
      → FAIT (2026-07-17) : branche = `feat/explorer-briefing-compact` ; `git status --porcelain`
      = seul `.ai/PLAN_EXPLORER_BRIEFING_V3_COMPACT_2026-07.md` untracked (worktree propre).
- [x] Re-vérifier §2 sur pièces : rouvrir chaque fichier:ligne cité et confirmer qu'il n'a pas
      bougé (le V2 a été mergé le 2026-07-16 ; le code peut différer). Consigner tout décalage
      en §6 Découvertes.
      → FAIT : Strip/Modules/logic/BriefingSectionCard/tests/Sparkline/geometry/KpiCard/README/
      explorer.toml rouverts. Tous les ancrages §2 correspondent (lignes exactes). Un seul
      décalage : `PostSyncMatrix.tsx` est dans `features/admin/convergence/` (pas `admin/sync/`),
      import `'../sync/Sparkline'` (Découverte-5).
- [x] Confirmer SPARK-1 (promotion) : au démarrage, si l'utilisateur n'a pas objecté, appliquer
      le défaut (promotion) ; sinon noter le repli SVG inline (report valide « décision
      utilisateur »).
      → FAIT : aucune objection utilisateur (confirmé par le brief superviseur) → DÉFAUT
      appliqué (promotion vers `components/charts/`).

Gate Phase 0 : branche correcte ; constat re-vérifié (décalages notés) ; SPARK-1 tranché. Pas
de gate de build (aucun code modifié). → **GATE PASSÉ (2026-07-17).**

### Phase 1 — Promotion de la primitive Sparkline (frontend-only, isolé) — SPARK-1

- [x] **1a.** `git mv apps/web/src/features/admin/sync/Sparkline.tsx`,
      `sparklineGeometry.ts`, `sparklineGeometry.test.ts` vers
      `apps/web/src/components/charts/`. Ajuster l'import interne de `Sparkline.tsx`
      (`./sparklineGeometry` reste relatif si co-localisé) et l'import de token
      (`@/lib/accessibility/semantic-tokens` inchangé, alias absolu).
      → FAIT : 3 `git mv` exécutés. Imports internes de `Sparkline.tsx` inchangés
      (`./sparklineGeometry` toujours co-localisé ; token alias absolu) — aucune retouche requise.
- [x] **1b.** Mettre à jour les 2 consommateurs admin : `PostSyncMatrix.tsx:11` et
      `SyncCycleHistory.tsx:10` → importer `Sparkline` depuis `@/components/charts/Sparkline`.
      → FAIT : `features/admin/convergence/PostSyncMatrix.tsx:11` (`'../sync/Sparkline'` →
      `'@/components/charts/Sparkline'`, cf. Découverte-5 pour le chemin réel) et
      `features/admin/sync/SyncCycleHistory.tsx:10` (`'./Sparkline'` → `'@/components/charts/Sparkline'`).
- [x] **1c.** Ajouter au `apps/web/src/components/charts/README.md` une sous-section
      « Primitives SVG pures (non-ECharts) » documentant `Sparkline` (micro-courbe nue, sans
      axes, couleur par token) — la primitive n'est PAS un wrapper ECharts, le préciser pour
      ne pas fausser le cadrage du catalogue.
      → FAIT : section « Primitives SVG pures (non-ECharts) » ajoutée avant « Color tokens »
      (props, géométrie testée, consommateurs, distinction vs `OutcomeSparkline` palmarès).
- [x] **1d.** Vérifier qu'aucun autre fichier ne référence l'ancien chemin
      `features/admin/sync/Sparkline` ou `.../sparklineGeometry` (grep de clôture = 0 hors les
      3 fichiers déplacés).
      → FAIT : grep = seuls les 2 imports `./sparklineGeometry` co-localisés dans
      `components/charts/` (Sparkline.tsx, sparklineGeometry.test.ts) subsistent (attendus).
      0 référence à l'ancien chemin admin.

Gate Phase 1 : `make check-types` = 0 ; `make test-web` (dangerouslyDisableSandbox) vert (dont
`sparklineGeometry.test.ts` à son nouveau chemin) ; `cd apps/web && npm run lint` = 0 erreur ;
grep de clôture : 0 référence à `features/admin/sync/Sparkline`/`sparklineGeometry`.
→ **GATE PASSÉ (2026-07-17)** : check-types EXIT=0 ; test-web = 260 fichiers / 2260 tests OK,
14 skipped (baseline pré-existante), EXIT=0 ; ciblé `sparklineGeometry.test.ts` nouveau chemin
= 7/7 OK ; lint EXIT=0, 0 erreur (68 warnings baseline gelée) ; grep clôture = 0.

### Phase 2 — Blocs « pleine largeur → socle » : sparkline + tuiles Classement & Séries (moyen, frontend-only) — DP-2, DP-3, DP-6, DEC-1, DEC-2, DEC-4

> Cette phase migre EN UNE FOIS les trois blocs qui quittent la pleine largeur pour le socle
> (Tendance→sparkline, Classement→tuile, Séries→tuile) ET retire leurs cartes de `Modules`,
> pour ne jamais laisser un bloc rendu deux fois.

- [x] **2a (extraction `BriefingTile`).** Créer `apps/web/src/features/explorer/BriefingTile.tsx`
      avec `BriefingTile` (+ `TileProps`) déplacé depuis le Strip, en AJOUTANT un slot optionnel
      `chart?: ReactNode` rendu entre `value` et `sub`. Le Strip importe désormais `BriefingTile`
      de ce fichier (supprimer la définition locale). Tokens sémantiques uniquement.
      → FAIT : `BriefingTile.tsx` créé (`TileProps` exporte `chart?: ReactNode` rendu entre valeur
      et sous-texte) ; Strip importe `./BriefingTile`, définition locale + `KpiCard` import retirés.
- [x] **2b (i18n).** Ajouter `explorer.briefing.ranked_since` (FR « depuis {tier} » / EN
      « since {tier} ») à `explorer.toml`. Régénérer les manifests. (Purge des clés orphelines
      `trend_title`/`streak_best`/`streak_worst` en 2f, après retrait des lecteurs.)
      → FAIT : `ranked_since` ajouté après `placement_remaining` ; manifests régénérés (diff
      généré = +ranked_since uniquement à ce stade).
- [x] **2c (tuiles).** Créer `apps/web/src/features/explorer/ExplorerBriefingTiles.tsx` :
      - `RankedTile({ ranked, t })` : composant `BriefingTile`, label = `ranked_title`. Valeur =
        palier de FIN du type MAJORITAIRE (`kinds[0]`) : `tier_end_placement_remaining != null`
        → `t('placement_remaining', { n })` ; sinon `tier_end_label` ; sinon (aucun palier) →
        pt/match seul (`ranked_per_match`) ; sinon « — ». Sous-texte = ligne 1
        « {kind.kind maj} · {ranked_since(début)} · {ranked_per_match} » où début =
        `tier_start_is_placement ? placement : tier_start_label` (segment « depuis » omis si
        début non résolvable) ; ligne 2 (SI un 2e type existe dans `kinds`) = compact
        « {kind2 maj} · {rankedProgression(kind2)} · {ranked_per_match(kind2)} ». JAMAIS croiser
        les paliers de deux types. Couleur du pt/match via token (`deltaToken`).
      - `StreaksTile({ streaks, t })` : `BriefingTile`, label = `streaks_title`. Valeur =
        ReactNode bicolore « {best} V / {worst} D » (`streak_wins`/`streak_losses`, tokens
        `outcome-win`/`outcome-loss`), segment à zéro omis (rendu conditionnel).
      - Déplacer `rankedProgression` depuis `ExplorerBriefingModules.tsx` vers ce fichier.
      → FAIT : `ExplorerBriefingTiles.tsx` créé avec `RankedTile`/`StreaksTile` (via `BriefingTile`),
      `rankedProgression` déplacé (LOCAL non exporté — sinon `react-refresh/only-export-components` ;
      un fichier de composants n'exporte que des composants), + helpers locaux `perMatchLabel`/
      `rankedValue`/`sinceLabel`/`rankedSubLine`. Couleur pt/match via `deltaToken` CENTRALISÉ
      (Découverte-6). Multi-type = 2 lignes, paliers jamais croisés.
- [x] **2d (Strip).** Dans `ExplorerBriefingStrip.tsx` : appeler `useCapability('ranked')`
      AVANT le `return null` précoce (règles des hooks). Rendre, dans la grille socle (DEC-3) :
      la tuile Taux de victoire avec `chart={<Sparkline values={(trend.points ?? []).map(p =>
      Math.round(p.win_rate*100))} token="outcome-win" width={120} height={28} ariaLabel={…} />}`
      uniquement si `briefing.trend != null` ; puis `hasRanked && briefing.ranked != null &&
      <RankedTile …>` et une garde `showStreaks` (au moins un segment > 0) `&& <StreaksTile …>`.
      Appliquer la grille socle responsive DEC-3.
      → FAIT : `useCapability('ranked')` appelé AVANT `if (!briefing) return null` ; sparkline
      (`width=120 height=28 token=outcome-win`) dans la tuile Taux de victoire si `trend != null`,
      `ariaLabel` = `win_rate_label` (trend_title purgé, cf. Découverte-7) ; `RankedTile`/
      `StreaksTile` conditionnels ajoutés ; grille DEC-3 `grid-cols-2
      sm:[grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]`. deltaToken importé du helper
      canonique. Frise conservée (retrait = Phase 4).
- [x] **2e (Modules).** Supprimer de `ExplorerBriefingModules.tsx` : `TrendCard`, `RankedCard`,
      `RankedKindRow`, `StreaksCard`, leurs rendus (`:129`, `:130`, `:132`) et imports devenus
      morts (`TimeseriesLineChart`, `ChartSeries`/`ChartPoint2D`, `useCapability`, types
      `ExplorerBriefingRanked`/`…RankedKind`/`…Trend`/`…Streaks`). Retirer `showRanked`,
      `showStreaks`, `streaks` de Modules. Réduire la garde early-return à `dimensions.length===0
      && contextSplit==null && !showDominance` (la bande Moments forts et la carte contexte
      restent gérées par Modules jusqu'à la Phase 3). Mettre à jour le commentaire d'en-tête du
      fichier (retirer « Tendance : sparkline » et « Classement »).
      → FAIT : Modules réécrit — ne rend plus que dimensions + `ContextSplitCard` + `DominanceCard` ;
      TrendCard/RankedCard/RankedKindRow/StreaksCard + `rankedProgression` + imports morts supprimés ;
      `hasRanked`/`showRanked`/`showStreaks`/`streaks` retirés ; garde réduite ; `deltaToken`
      importé du helper canonique (plus de def locale) ; en-tête MAJ. `formatSignedFixed` retiré
      de l'import (plus d'usage).
- [x] **2f (purge i18n orphelines).** Re-vérifier sur pièces qu'aucun composant ne lit plus
      `trend_title`, `streak_best`, `streak_worst` (grep) ; les supprimer de `explorer.toml` +
      régénérer les manifests. (Si un lecteur subsiste — improbable —, ne pas purger et le noter
      en §6.)
      → FAIT : grep = 0 lecteur de production (seuls toml/generated + vieux tests réécrits) ; 3 clés
      purgées de `explorer.toml` (+ commentaire de section « cartes » → « tuile/bande » corrigé pour
      éviter la doc inversée) ; manifests régénérés (diff = +ranked_since, -trend_title,
      -streak_best, -streak_worst).
- [x] **2g (tests).** Réécrire dans `ExplorerBriefingStrip.test.tsx` le describe « Séries »
      pour la TUILE (valeur `streak_wins`/`streak_losses`, segment nul omis, tuile omise si deux
      zéros) et AJOUTER un describe « Classement » (tuile présente si `ranked` + capability ;
      valeur = palier de fin ; sous-texte type + `ranked_since` + `ranked_per_match` ; multi-type
      = 2 lignes ; capability off → tuile absente). Vérifier que la capability `ranked` est
      activable dans `renderWithProviders` (sinon fournir le provider adéquat — consulter
      `@/lib/capabilities/capabilities`). Aucun test skippé.
      → FAIT : describe « Séries » réécrit (tuile) + describe « Classement » ajouté (single/multi/
      capability-off via `useAppShellStore.setState` + `afterEach` reset ; fail-open = capability
      active par défaut → cas « on » couvert sans provider spécial). Aucun test skippé.

Gate Phase 2 : `node …/build_i18n_manifests.mjs` (diff = seules clés attendues) ;
`make check-types` = 0 ; `make test-web` vert (describes Classement/Séries réécrits) ;
`npm run lint` = 0 erreur ; greps de clôture : 0 `TrendCard`/`RankedCard`/`StreaksCard`/
`TimeseriesLineChart` dans `ExplorerBriefingModules.tsx` ; 0 `trend_title`/`streak_best`/
`streak_worst` sous `apps/web/src` (hors historique).
→ **GATE PASSÉ (2026-07-17)** : regen i18n EXIT=0 (diff conforme) ; check-types EXIT=0 ;
test-web complet = 261 fichiers / 2264 tests OK, 14 skipped (baseline), EXIT=0 (dont garde-rail
deltaToken + describes réécrits) ; lint EXIT=0, 0 erreur, 68 warnings (baseline restaurée après
correction du `react-refresh` sur `rankedProgression`) ; greps de clôture = 0.

### Phase 3 — Rangée « Par… » : carte contexte + bande Moments forts + FDA coloré (moyen, frontend-only) — DP-4, DP-5, DP-10, DEC-3, DEC-4

- [x] **3a (i18n).** Modifier `explorer.briefing.context_split_title` → FR « Par contexte » /
      EN « By context ». Régénérer les manifests.
      → FAIT : toml modifié, manifests régénérés (diff généré = context_split_title uniquement).
- [x] **3b (grille « Par… »).** Dans `ExplorerBriefingModules.tsx`, rendre `ContextSplitCard`
      (retitrée via `context_split_title`) comme 4e cellule de la MÊME grille que les dimensions
      (DEC-3 : `grid-cols-1 sm:grid-cols-2 xl:grid-cols-4`), conditionnée à `contextSplit != null`.
      `ContextSplitCard`/`ContextSplitRow` conservent `BriefingSectionCard` et le gabarit de
      lignes (libellé, n matchs, WR coloré, FDA) déjà aligné sur `DimensionRow`. Retirer l'ancien
      rendu séparé de la carte contexte (`:131`).
      → FAIT : grille unique `grid-cols-1 sm:grid-cols-2 xl:grid-cols-4` rendant dimensions +
      `ContextSplitCard` (garde `contextSplit != null`) ; ancien rendu séparé retiré. Test dédié
      vérifie que la carte contexte est dans la grille `xl:grid-cols-4` avec `dim_map`.
- [x] **3c (bande Moments forts).** Remplacer `DominanceCard` par une bande NUE (nouveau
      composant local `DominanceBand` ou refonte de `DominanceCard`) : plus de
      `BriefingSectionCard`, plus d'en-tête de carte. Structure = un libellé discret muted
      (`highlights_title`, ex. `text-2xs uppercase tracking-wide text-muted-foreground`) suivi de
      la même rangée `flex flex-wrap gap-1.5` de pastilles (réutiliser `DOMINANCE_ITEMS` +
      `DOMINANCE` styling par token, catégories à zéro omises). Rendue sous la grille « Par… ».
      → FAIT : `DominanceCard` → `DominanceBand` (div `flex flex-wrap items-center gap-1.5`,
      libellé muted `text-2xs uppercase tracking-wide text-muted-foreground` + pastilles inchangées).
      Plus de `BriefingSectionCard`. Test vérifie `label.closest('.border-b') === null`.
- [x] **3d (garde-rail `BriefingSectionCard`).** Mettre à jour le commentaire garde-rail de
      `BriefingSectionCard.tsx:11-19` : seules les cartes-sections RESTANTES (dimensions +
      contexte) doivent passer par ce wrapper ; retirer la mention de « classement / séries /
      moments forts » (désormais tuiles/bande) — corriger la « doc inversée ».
      → FAIT : en-tête + garde-rail réécrits (périmètre V3 : seules dimensions + « Par contexte » ;
      Classement/Séries = tuiles, Tendance = sparkline, Moments forts = bande). Réf « module
      Tendance » de ChartCard retirée.
- [x] **3e (FDA coloré — DP-10, amendement).** Dans `ExplorerBriefingModules.tsx`, colorer le
      FDA de `ContextSplitRow` (`:341-343`, aujourd'hui `text-muted-foreground`) via
      `style={{ color: kdaNetColor(group.kda) }}` (ajouter l'import `kdaNetColor` depuis
      `@/lib/colors/outcomePalette`, aux côtés de `winRateColor` `:20`) — même convention que
      la tuile socle FDA (Strip `:147`). Balayer ensuite le bandeau entier (Strip + Modules +
      Tiles) : AUCUNE autre valeur FDA non colorée (vérification par lecture, consigner le
      résultat au journal).
      → FAIT : import `kdaNetColor` ajouté, FDA de `ContextSplitRow` coloré. BALAYAGE : grep
      `toFixed` sur Strip/Modules/Tiles = exactement 2 surfaces FDA — Strip:145 (span
      `kdaNetColor(kda)`) et Modules:222 (span `kdaNetColor(group.kda)`), TOUTES colorées. Tiles
      n'affiche aucun FDA. Aucun FDA non coloré dans le bandeau.
- [x] **3f (tests).** Mettre à jour dans `ExplorerBriefingStrip.test.tsx` : le describe contexte
      (titre `context_split_title` toujours vérifié, valeur désormais « Par contexte » — le stub
      `t` renvoie la clé, assertions inchangées) ; le describe Moments forts (bande : libellé
      `highlights_title` + `×N` toujours présents ; carte-header ABSENT — vérifier qu'aucun
      chrome de carte n'entoure la bande si un test le contrôle). Ajouter au besoin un test que
      la carte contexte apparaît dans la grille « Par… ».
      → FAIT : describe contexte renommé « Par contexte » + assertion grille `xl:grid-cols-4`
      contient contexte + `dim_map` ; describe Moments forts renommé « bande » + assertion
      `label.closest('.border-b') === null` (bande nue, pas d'en-tête de carte). Assertions clés
      inchangées (stub `t` renvoie la clé).

Gate Phase 3 : `node …/build_i18n_manifests.mjs` (diff = `context_split_title`) ;
`make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ; grep de clôture :
0 `DominanceCard`/`BriefingSectionCard` autour des moments forts (bande nue) ; `ContextSplitCard`
rendu dans la grille dimensions ; `kdaNetColor` présent dans `ExplorerBriefingModules.tsx`
(FDA contexte coloré, DP-10).
→ **GATE PASSÉ (2026-07-17)** : regen i18n EXIT=0 (diff = context_split_title) ; check-types
EXIT=0 ; test-web complet = 261 fichiers / 2264 tests OK, 14 skipped, EXIT=0 ; lint EXIT=0,
0 erreur, 68 warnings (baseline) ; greps clôture : DominanceCard=0, kdaNetColor présent,
contexte dans la grille, bande sans en-tête.

### Phase 4 — Retrait de la frise (frontend) + purge des lecteurs (rapide, frontend-only) — DP-1 (front)

- [x] **4a (Strip).** Retirer de `ExplorerBriefingStrip.tsx` : imports `OutcomeSequenceTape`/
      `OutcomePoint` (`:13`) et `outcomeCodeToValue` (`:24`) ; le calcul `tapePoints` (`:83-86`) ;
      le rendu frise (`:190-202`). Simplifier `matchesCount` → `scope?.matches ?? 0` (retirer le
      fallback `outcome_sequence?.length`).
      → FAIT : import frise retiré, `outcomeCodeToValue` retiré de l'import logic, `tapePoints` +
      rendu frise supprimés, `matchesCount = scope?.matches ?? 0`, en-tête MAJ (« puis frise »
      retiré).
- [x] **4b (logic).** Supprimer `outcomeCodeToValue` (`ExplorerBriefing.logic.ts:56-67`) + import
      `OutcomeValue` (`:8`) — devenus morts (aucun autre lecteur, vérifié §2). Supprimer le
      describe correspondant dans `ExplorerBriefing.logic.test.ts:56-62` + son import.
      → FAIT : `outcomeCodeToValue` + import `OutcomeValue` supprimés (grep = 0 lecteur restant) ;
      describe + import retirés du test ; en-tête logic MAJ (« mapping de la frise » retiré).
- [x] **4c (i18n).** Supprimer `series_win`/`series_loss`/`series_tie`/`series_dnf`
      (`explorer.toml:873-887`) — seul lecteur = la frise du Strip (vérifié §2). Régénérer les
      manifests.
      → FAIT : 4 clés purgées, manifests régénérés (diff = -series_win/loss/tie/dnf ; explorer
      221 clés).
- [x] **4d (tests Strip).** Retirer `outcome_sequence: []` du `makeBriefing`
      (`ExplorerBriefingStrip.test.tsx:46`). Vérifier que les describes existants passent sans la
      frise.
      → FAIT : `outcome_sequence: []` retiré (`outcome_sequence?` optionnel dans le type généré,
      vérifié) ; tous les describes passent.

Gate Phase 4 : `node …/build_i18n_manifests.mjs` (diff = retrait `series_*`) ;
`make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ; greps de clôture
sous `apps/web/src/features/explorer` : 0 `OutcomeSequenceTape`, 0 `outcomeCodeToValue`,
0 `series_win|series_loss|series_tie|series_dnf` ; grep global : `OutcomeSequenceTape` ne
subsiste QUE dans `components/charts/OutcomeSequenceTape.tsx` + `RelationsRivalryCards.tsx`
(+ showcases/README).
→ **GATE PASSÉ (2026-07-17), avec réserve documentée sur la formulation du grep global
(Découverte-9)** : regen i18n EXIT=0 (diff = -series_*) ; check-types EXIT=0 ; test-web complet
= 261 fichiers / 2263 tests OK, 14 skipped, EXIT=0 ; lint EXIT=0, 0 erreur, 68 warnings
(baseline) ; greps `features/explorer` = 0 (frise absente du briefing — l'exigence substantielle
§1.2/§1.10). RÉSERVE : le grep global attendu par le plan (« QUE charts + RelationsRivalryCards »)
est FACTUELLEMENT INEXACT — `OutcomeSequenceTape` est un chart partagé app-wide utilisé aussi par
`HomePage`, `TimeseriesPage.summary`, `SquadSynergiesPage`, `ChartsShowcasePage` (consommateurs
pré-existants, hors périmètre, NON touchés). La substance (frise retirée du briefing Explorer,
composant préservé, RelationsRivalryCards intact) est vérifiée ; la formulation du plan est
corrigée en Découverte-9. Aucun contournement de gate.

### Phase 5 — Purge backend de `outcome_sequence` (moyen, backend + regen) — DP-1 (back)

- [ ] **5a (domain).** `internal/domain/explorer_briefing.go` : supprimer le champ
      `OutcomeSequence` (`:34`, + ses 2 lignes de commentaire `:32-33`) et la struct
      `ExplorerBriefingOutcome` (`:75-81`).
- [ ] **5b (service).** `match_history_service_briefing.go` : supprimer la const
      `maxOutcomeSequencePoints` (`:49`), la ligne `b.OutcomeSequence = buildOutcomeSequence(...)`
      (`:71`), la fonction `buildOutcomeSequence` (`:110-133`). Corriger le commentaire
      `match_history_service_briefing_streaks.go:5` qui référence la frise/const disparue
      (reformuler sans mentionner `maxOutcomeSequencePoints`). Vérifier qu'aucun autre fichier Go
      ne référence `maxOutcomeSequencePoints`/`buildOutcomeSequence`/`OutcomeSequence` (grep).
- [ ] **5c (tests service).** `match_history_service_briefing_test.go` : retirer le contrôle
      `len(b.OutcomeSequence)` (`:61-62`) et supprimer
      `TestBuildExplorerBriefing_OutcomeSequenceCappedAndSorted` (`:531-544`). Vérifier qu'aucun
      autre test ne lit `OutcomeSequence`.
- [ ] **5d (OpenAPI + regen).** `api/openapi.yaml` : supprimer la propriété `outcome_sequence`
      de `ExplorerBriefing` (`:5017-5021`) et le schéma `ExplorerBriefingOutcome` (`:5144-5159`).
      Régénérer via le mécanisme d'émission si nécessaire
      (`OPENAPI_EMIT_OUT=… go test ./internal/api/ -run TestOpenAPISchemaDrift`) pour rester
      byte-exact avec Huma, puis `make generate-types`. `types.ts`/`generated.ts` :
      `ExplorerBriefingOutcome` et le champ `outcome_sequence` disparaissent (régénérés, ne PAS
      éditer à la main).

Gate Phase 5 : `cd apps/go-api && go test ./...` = 0 (dont `TestOpenAPISchemaDrift` = 0 MISSING/
DIVERGENT sur `ExplorerBriefing`) ; `make go-api-lint` = 0 ; `make generate-types` idempotent
(re-run → 0 diff) ; `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ;
greps de clôture : 0 `outcome_sequence`/`OutcomeSequence`/`ExplorerBriefingOutcome`/
`buildOutcomeSequence`/`maxOutcomeSequencePoints` sous `apps/go-api` (hors historique) ET
0 `outcome_sequence`/`ExplorerBriefingOutcome` sous `apps/web/src` (hors `.ai`/docs).

### Phase 5b — Tooltips de légende (moyen, frontend-only) — DP-9, DP-11, DEC-7 (amendement 2026-07-17)

> Placée APRÈS les phases de structure : les tooltips se posent sur le bandeau FINAL (tuiles,
> cartes « Par… », bande), pas sur des composants qui vont encore bouger.

- [ ] **5b-a (i18n).** Ajouter à `explorer.toml` les 8 clés `tip_*` (liste fermée §2), FR + EN,
      à partir des textes par défaut DEC-7 (FR sans anglicismes ; EN en parité ; tooltip FDA
      grand public, sans formule ADR brute). AVANT de finaliser `tip_highlights` : re-vérifier
      sur pièces la sémantique exacte des 5 catégories (`analysis.DominanceFlag*` +
      libellés/docs `narrative.dominance.*` du manifest match_view) — ajuster la formulation si
      elle contredit la définition réelle, consigner au journal. Régénérer les manifests.
- [ ] **5b-b (tuiles socle).** Ajouter la prop optionnelle `info?: ReactNode` à `BriefingTile`
      (rangée label : `inline-flex items-center gap-1`, l'icône ne casse pas l'uppercase du
      label — le panneau du tooltip force déjà `normal-case`). Passer un
      `<InfoTooltip content={t('explorer.briefing.tip_…')} iconClass="w-3.5 h-3.5" />` sur :
      Taux de victoire (`tip_win_rate`), FDA (`tip_fda`), Perf (`tip_perf`) dans le Strip ;
      Classement (`tip_ranked`) et Séries (`tip_streaks`) dans `ExplorerBriefingTiles.tsx`.
      PAS de tooltip sur la tuile Matchs (liste fermée DP-9). Labels de tuiles INCHANGÉS
      (DP-11 : courts).
- [ ] **5b-c (cartes « Par… »).** Dans `ExplorerBriefingModules.tsx`, injecter l'`InfoTooltip`
      dans le slot `title` de `BriefingSectionCard` (DEC-7) : `tip_dimensions` sur les 3 cartes
      dimensions (via `DimensionCard`), `tip_context` sur la carte « Par contexte ».
- [ ] **5b-d (bande).** Accoler un `<InfoTooltip content={t('explorer.briefing.tip_highlights')}
      iconClass="w-3.5 h-3.5" />` au libellé muted `highlights_title` de la bande Moments forts.
- [ ] **5b-e (tests).** Étendre `ExplorerBriefingStrip.test.tsx` : (1) présence des boutons (i)
      — compter les `getAllByRole('button', { name: /more_info|complément/i })` selon la valeur
      réelle de `common.tooltip.more_info_aria` (la vérifier sur pièces) : attendu = 5 tuiles +
      4 cartes + 1 bande sur un briefing complet, et MOINS quand des blocs sont omis (ex. pas de
      `ranked` → pas de tooltip Classement) ; (2) un test d'interaction : clic sur l'icône de la
      carte dimension → le contenu `explorer.briefing.tip_dimensions` apparaît (le stub `t`
      renvoie la clé). Aucun test skippé.

Gate Phase 5b : `node …/build_i18n_manifests.mjs` (diff = 8 clés `tip_*`) ; `make check-types`
= 0 ; `make test-web` vert (tests 5b-e inclus) ; `npm run lint` = 0 erreur ; grep de clôture :
les 8 clés `tip_*` présentes dans `explorer.toml` (FR ET EN — parité garantie par le typage du
manifest) ; `InfoTooltip` importé depuis `@/components/ui/info-tooltip` (aucune primitive
tooltip nouvelle créée) ; 0 modification de `components/ui/info-tooltip.tsx`.

### Phase 6 — Vérification navigateur & clôture

- [ ] **6a.** Lancer l'environnement : serveur go-api (`:8000` healthz 200 ; CGO, données
      réelles du dépôt principal) + vite. Réutiliser la session admin réelle (cf. protocole
      Phase 6a du plan V2 : cookie signé injecté via CDP `extraHttpHeaders`, sans re-login ni
      modif fichier). Ouvrir l'Explorer mode Matchs d'un profil réel sur halo_infinite (LUSR).
- [ ] **6b (plein historique, FR).** Capturer. Vérifier : critère §1.1 **hauteur du bandeau
      ~300-330 px** (`getBoundingClientRect`) ; frise absente (§1.2) ; tuile Classement (§1.3) ;
      tuile Séries (§1.4) ; sparkline dans la tuile Taux de victoire (§1.5) ; carte « Par
      contexte » en 4e cellule (§1.6) ; bande Moments forts nue (§1.7) ; socle 4-6 tuiles
      lisibles (§1.8) ; **tooltips** (§1.14) : survol/focus de l'icône (i) sur une tuile
      (ex. FDA), une carte « Par… » et la bande → panneau lisible, texte FR sans anglicismes,
      fermeture au blur/clic extérieur (capturer un tooltip ouvert) ; **FDA coloré** (§1.15)
      dans les lignes « Par contexte ». Console 0 erreur.
- [ ] **6c (scope filtré, FR).** Appliquer un filtre narrowing (ex. contexte Solo). Vérifier :
      deltas « vs habituel » réapparaissent (comportement V2 préservé) ; carte « Par contexte »
      disparaît si scope mono-contexte ; tuiles Classement/Séries recalculées ; bande recalculée.
      Console 0 erreur.
- [ ] **6d (dégradations + EN).** Titre H5 (profil non ranked-capable) → tuile Classement OMISE
      (et son tooltip avec), pas de crash. low_sample (scope réduit sous le seuil) → socle 4
      tuiles + mention, aucun module/bande/sparkline. Spot-check locale EN (PATCH settings
      lang=en) : « Ranking », « since {tier} », « Streaks » valeur « {n} W / {n} L »,
      « Highlights », « By context », dates avec année, ET un tooltip ouvert en EN (contenu
      `tip_*` anglais, pas la clé brute). Console 0 erreur. Restaurer la session
      (halo_infinite, fr) en fin.
- [ ] **6e (changelog).** `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`, entrée `[Unreleased]`
      (v7.0) : ajouter un bullet « Explorer — briefing V3 » dans « Added (React / TypeScript) »
      (bandeau compacté : Classement & Séries en tuiles du socle, tendance en micro-sparkline,
      Solo/Escouade fusionné dans la grille « Par contexte », Moments forts en bande, frise
      retirée ; tooltips de légende (i) sur les tuiles, cartes « Par… » et la bande ; FDA
      coloré partout) ; compléter le bullet Go de « Added (Go API) » avec le retrait de
      `outcome_sequence`/`ExplorerBriefingOutcome`. Parité EN/FR dans le même commit (hook
      docs-fr-sync). Format Keep a Changelog respecté.
- [ ] **6f (clôture).** Dérouler `delivery-checklist`. Entrée `.ai/thought_log.md` finale
      (date, titre, statut, décision technique principale = SPARK-1/compaction, résultats
      observés = hauteur mesurée + gates, prochaine étape = revue visuelle utilisateur avant
      merge). Point d'étape utilisateur. NON committé sans autorisation (merge `main` = deploy
      prod auto → après revue visuelle).

Gate Phase 6 : tous les critères §1 (1-15) vérifiés en navigateur (captures au journal, dont
un tooltip ouvert FR et un EN) ; console 0 erreur sur les 4 états ; changelog EN+FR à jour ;
passe finale des gates §1.11 verts en une fois (`go test ./...`, `make go-api-lint`,
`make generate-types` idempotent, `make check-types` cache `.tmp` purgé, `make test-web`,
`npm run lint`).

---

## 6. Découvertes (à remplir en exécution — ne pas traiter hors périmètre)

- **Découverte-0 (pré-notée, 2026-07-17) — primitive sparkline mal localisée.** Le brief
  supposait la recherche d'une sparkline nue dans `components/charts/` (aucune). Une primitive
  pure/testée existe en `features/admin/sync/` (`Sparkline.tsx` + `sparklineGeometry.ts`). D'où
  la décision SPARK-1 (promotion vers `components/charts/` + réutilisation), qui dévie du choix
  binaire (a) SVG inline / (b) nouveau wrapper du brief : c'est une 3e voie (réutiliser
  l'existant) conforme à CLAUDE.md §14/§6. Repli SVG inline documenté si l'utilisateur refuse de
  toucher `features/admin/`.
- **Découverte-1 (pré-notée) — clés i18n orphelinées.** `series_*` (frise), `trend_title`
  (tendance), `streak_best`/`streak_worst` (lignes de la carte Séries) perdent leur unique
  lecteur avec la compaction → purgées dans la phase qui les orpheline (2f/4c). Re-vérifier
  chaque grep AVANT purge (un lecteur inattendu = ne pas purger, consigner ici).
- **Découverte-2 (pré-notée, HORS périmètre) — `record_label`.** La clé
  `explorer.briefing.record_label` (« Bilan »/« Record ») semble sans lecteur (la tuile utilise
  `win_rate_label`). NON orphelinée par ce chantier → ne pas la traiter ; si confirmée morte,
  la signaler comme dette pré-existante.
- **Découverte-3 (pré-notée) — `DOMINANCE_ITEMS` = 2e copie du mapping DominanceFlag→i18n/token**
  (la 1re est `ExplorerMatchesTable`, cf. Découverte-3 du plan V2). Reste 2 copies (dans la
  limite CLAUDE.md §6). La bande Moments forts NE crée PAS de 3e copie (réutilise
  `DOMINANCE_ITEMS`). Si une 3e surface apparaît un jour → centraliser + garde-rail.
- **Découverte-4 (pré-notée, amendement 2026-07-17) — composant tooltip canonique TROUVÉ.**
  `components/ui/info-tooltip.tsx` (`InfoTooltip`) : accessible (bouton focusable, aria-label
  via `common.tooltip.more_info_aria`, `role="tooltip"`), ~10 consommateurs existants, et le
  slot `title: ReactNode` de `BriefingSectionCard` avait été prévu « compatible InfoTooltip »
  dès le plan V2 Phase 3a → RÉUTILISATION pure (DEC-7), aucune primitive à créer, aucune
  modification du composant.

- **Découverte-5 (constatée Phase 0, 2026-07-17) — chemin réel de `PostSyncMatrix`.** Le §2
  cite « `PostSyncMatrix.tsx:11` » sous `features/admin/sync/` (SPARK-1 / item 1b). En réalité
  le fichier est `apps/web/src/features/admin/convergence/PostSyncMatrix.tsx` (ligne 11 :
  `import { Sparkline } from '../sync/Sparkline'`). Les 2 consommateurs réels de `Sparkline`
  sont donc : `features/admin/convergence/PostSyncMatrix.tsx:11` et
  `features/admin/sync/SyncCycleHistory.tsx:10`. Aucune conséquence sur la promotion (item 1b
  ajusté à ces 2 chemins). Aucun autre lecteur (grep exhaustif : le `OutcomeSparkline` de
  `PalmaresRelationsPage` est un composant distinct, hors périmètre).

- **Découverte-6 (Phase 2, 2026-07-17) — centralisation de `deltaToken` (déclenchée par la
  tuile Classement).** Le helper `deltaToken(v) → SemanticToken` (signe d'un delta → token
  `outcome-win`/`outcome-loss`/`outcome-draw`) existait en 2 copies locales identiques (Strip
  `:37-40`, Modules `:60-63`). `RankedTile` (DP-2) en a besoin (couleur du pt/match) → 3e usage
  ⇒ CLAUDE.md §6 impose « centraliser + garde-rail ». Fait DANS la phase (périmètre direct de
  l'ajout, pas un fix opportuniste) : `deltaToken` déplacé dans `ExplorerBriefing.logic.ts`
  (export, import type-only `SemanticToken` → helper reste pur), importé par Strip/Modules/Tiles
  (0 copie inline restante), + garde-rail `explorerDeltaToken.guard.test.ts` (interdit de
  ré-inliner le ternaire dans un composant du briefing). NB : `MatchStatCards.tsx` et
  `SquadVerdict.tsx` contiennent un `deltaToken` HOMONYME mais DIFFÉRENT (variable locale sur
  `skillDeltaScale`/tokens `divergent-*`) — hors périmètre, non touché, non compté.
- **Découverte-7 (Phase 2, 2026-07-17) — `ariaLabel` de la micro-sparkline.** Le plan (2d) note
  `ariaLabel={…}` sans clé. `trend_title` étant purgé (2f), la sparkline ne peut pas le réutiliser.
  Choix : `ariaLabel = t('explorer.briefing.win_rate_label')` (« Taux de victoire » / « Win rate »)
  — la sparkline vit dans la tuile Taux de victoire et en montre l'évolution ; aucune clé neuve
  hors `ranked_since` (conforme au plan). Décision fixée, pas de sur-caveat.
- **Découverte-8 (Phase 2, 2026-07-17, HORS périmètre) — warning `Unused eslint-disable`.**
  `ExplorerPage.tsx:159` porte un `eslint-disable react-hooks/set-state-in-effect` désormais
  inutile (fait partie des 68 warnings baseline gelée). Fichier NON touché par ce chantier →
  simple observation, non corrigé.
- **Découverte-9 (Phase 4, 2026-07-17) — le gate « grep global `OutcomeSequenceTape` » du plan
  est FACTUELLEMENT INEXACT.** Le Gate Phase 4 et le critère §1.10 affirment que
  `OutcomeSequenceTape` ne subsiste QUE dans `components/charts/OutcomeSequenceTape.tsx` +
  `RelationsRivalryCards.tsx` (+ showcases/README), et que « l'unique consommateur restant est
  `RelationsRivalryCards` ». C'est faux : `OutcomeSequenceTape` est un **wrapper chart partagé
  app-wide** (cf. son README : « HomePage, MatchHistoryPage, SquadV2Page ») consommé aussi par
  `features/home/HomePage.tsx:439`, `features/timeseries/TimeseriesPage.summary.tsx:84`,
  `features/squad/SquadSynergiesPage.tsx:112`, `features/lab/ChartsShowcasePage.tsx:330` — tous
  PRÉEXISTANTS et HORS PÉRIMÈTRE (non touchés). L'exigence SUBSTANTIELLE (frise retirée du
  bandeau Explorer + composant préservé + `RelationsRivalryCards` intact) EST satisfaite : grep
  `features/explorer` = 0 `OutcomeSequenceTape`. Le plan a sous-compté les consommateurs ; aucun
  code à corriger, aucune régression. La formulation « unique consommateur » du plan doit être
  amendée si le plan est réutilisé.

Consigner ici tout décalage fichier:ligne vs §2, tout lecteur i18n inattendu, toute dette
repérée hors des blocs Variante B. Ne pas corriger dans ce chantier.

---

## 7. Protocole de reprise de session

1. `git branch --show-current` doit être `feat/explorer-briefing-compact` (sinon la retrouver
   via `git log --oneline -10`). Ne jamais reprendre sur `main` ni une branche de train.
2. Lire ce fichier : la dernière phase dont le **Gate** est coché est close ; reprendre à la
   première non close. Les cases `[ ]` d'une phase non close = travail restant.
3. Lire l'entrée `.ai/thought_log.md` la plus récente de ce chantier (avancement + décisions
   retenues, dont SPARK-1 et le statut low_sample vérifié).
4. Re-vérifier sur pièces les fichier:ligne de la phase courante AVANT d'éditer ou de cocher
   (plan-execution : vérifier sur pièces avant de coder ET avant de cocher — le V2 a bougé le
   code, les numéros de §2 peuvent avoir dérivé).
5. Ne jamais commencer une phase N+1 tant que le Gate de N n'est pas vert.

---

## 8. Effort estimé & dépendances

| Bloc Variante B | Phase | Effort | Couche |
|---|---|---|---|
| Promotion Sparkline | 1 | Rapide | front (déplacement + README + 2 imports) |
| Tendance → sparkline tuile | 2 | Moyen | front (BriefingTile.chart + Strip) |
| Classement → tuile | 2 | Moyen | front (RankedTile + `ranked_since`) |
| Séries → tuile | 2 | Moyen | front (StreaksTile) |
| Solo/Escouade → « Par contexte » | 3 | Rapide | front (grille + i18n) |
| Moments forts → bande nue | 3 | Rapide | front (DominanceBand) |
| Frise retirée (front) | 4 | Rapide | front + i18n (`series_*`) |
| Frise purge (back) | 5 | Moyen | **domain + service + OpenAPI + regen** |
| Tooltips de légende (amendement) | 5b | Moyen | front + i18n (8 clés `tip_*`, réutilise `InfoTooltip`) |
| FDA coloré « Par contexte » (amendement) | 3 (item 3e) | Rapide | front (`kdaNetColor`) |
| Vérif navigateur + changelog | 6 | Moyen | navigateur + docs |

**Dépendances inter-phases** : Phase 1 (Sparkline promue) précède la Phase 2 (la tuile Taux de
victoire la consomme). Phases 2 et 3 sont frontend-only et indépendantes du backend (les DTO
sont inchangés). Phase 5 (purge backend) est indépendante des Phases 2-4 côté logique mais
placée après pour que le front ait cessé de lire `outcome_sequence` avant le retrait du DTO
(évite un intermédiaire où le front lit un champ absent). Phase 5b (tooltips) vient APRÈS
toutes les phases de structure : elle décore le bandeau final (tuiles + cartes + bande) et
dépend de `BriefingTile` (DEC-1) et de la bande (Phase 3). **Aucune dépendance utilisateur
bloquante** : SPARK-1 a un défaut ferme (promotion), toutes les décisions produit sont
tranchées (DP-1..11). **Aucun déploiement prod** dans ce chantier (le merge `main` = deploy
auto reste la décision de l'utilisateur, après revue visuelle 6b-6d).
