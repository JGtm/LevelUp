# PLAN — Explorer briefing V2 : ajustements post-revue visuelle

Statut : PLANIFIE (aucune ligne de code écrite — plan rédigé en worktree lecture seule).
Date : 2026-07-16.
Auteur du plan : architecte Opus (worktree isolé).
Chantier d'origine (V1, livré au train 2026-07-15) : `feat/explorer-briefing-cards`,
contrat `.ai/V7/PLAN_EXPLORER_BRIEFING_CARDS_2026-07.md`.

Branche cible d'implémentation : **`feat/explorer-briefing-v2`** (depuis `main` à jour).
Ne PAS implémenter sur la branche worktree de rédaction de ce plan.

> Contrat d'exécution : ce plan s'exécute sous le skill **`plan-execution`** (ordre strict,
> une étape close avant la suivante, aucun report d'action exécutable maintenant, statut sur
> chaque item, zéro fix hors périmètre). En cas de divergence, le présent plan fait foi ; à
> défaut, le skill est le défaut. Avant de finaliser toute modification du plan : skill
> **`plan-review`**. Avant chaque commit : skill **`delivery-checklist`**. Code Go : skill
> **`arch-rules`** ; code React/TS : skill **`frontend-patterns`** ; toute couleur :
> **`color-tokens`** ; libellés inter-titres : `canonical-types` / capabilities (jamais
> `slug == …`).

> Contexte produit : les items 3 (« Pronostic ») et 4 (Δ LUSR cumulé) sont les deux
> « arbitrages D4 » explicitement laissés au mergeur dans le journal
> (`.ai/thought_log.md` [2026-07-15], §« Questions ouvertes mergeur »). Ce plan les
> instruit sur pièces et pré-tranche tout ce qui ne relève pas d'un choix de libellé.

---

## 1. Objectif et critères de succès (mesurables)

**Objectif.** Corriger sept écarts relevés par l'utilisateur en revue visuelle du bandeau de
briefing de l'Explorer (mode Matchs), sans toucher au socle fonctionnel V1 (KPIs, frise,
baseline, dimensions, tendance) au-delà des ajustements listés. Le briefing doit : ne plus
afficher de delta « ±0 » trompeur quand rien n'est filtré ; parler la terminologie FR du
projet ; nommer et expliquer correctement le module rétrospectif ; afficher le classement en
paliers connus du joueur plutôt qu'en points bruts ; dater complètement ; distinguer
solo/escouade quand c'est pertinent ; et présenter toutes ses cartes-sections avec une mise
en forme unifiée.

**Pourquoi.** Le bandeau V1 est fonctionnel mais plusieurs surfaces desservent la lecture :
deltas « = ±0 pts » systématiques par défaut (déroutant), anglicisme « playlist », un module
« Pronostic » mal nommé (c'est un bilan rétrospectif, pas une prédiction), un Δ classement en
points bruts illisible (−1380), des dates sans année, aucune distinction solo/escouade, et une
hétérogénéité visuelle des titres de cartes. Règles projet engagées : « afficher la métrique
connue du user » (mémoire `feedback_show_known_metric_not_raw`, `reference_lusr_target_levels`),
FR sans anglicismes (`feedback_fr_ui_no_anglicisms_solid_pills`), parité FR/EN par typage.

**Critères de succès (tous vérifiables) :**

1. Quand le scope affiché = tout l'historique (aucun filtre de recherche narrowing), AUCUN
   delta « vs habituel » (socle + lignes de dimension) n'est affiché — ni « ±0 pts », ni
   « ±0.00 », ni flèche « = ». Dès qu'un filtre narrowing est actif, les deltas réapparaissent.
2. Le libellé de la dimension playlist est « Par sélection » (FR) / « By playlist » (EN),
   cohérent avec `explorer.filters.playlist` (déjà « Sélection » / « Playlist »). Aucune
   occurrence résiduelle de « Par playlist ».
3. Le module ex-« Pronostic » porte un nom juste (validé §3, AWAIT-USER D-A) et un
   libellé/tooltip qui le décrit comme un BILAN RÉTROSPECTIF (attendu du système vs réel).
   Aucune occurrence résiduelle de « Pronostic »/« Prognosis ».
4. Le classement s'affiche en progression de paliers connus (ex. « Bronze I → Platine VI »)
   + une moyenne par match (ex. « −1,4 pt/match »), plus aucun nombre cumulé brut (« −1380 »).
   Dégradation propre si les paliers de début/fin ne sont pas résolvables.
5. La carte « Matchs » affiche la période AVEC l'année (ex. « 3 – 12 mars 2025 »), via
   l'utilitaire de date partagé.
6. Une carte contexte solo/escouade apparaît UNIQUEMENT quand elle est pertinente (les deux
   sous-groupes dépassent le seuil §3 D-B ET le scope n'est pas déjà filtré à un seul
   contexte), et est omise sinon (dégradation par omission).
7. Les cartes-sections du briefing (dimensions, module rétrospectif, carte solo/escouade)
   partagent la mise en forme d'en-tête du bloc « Tendance » (en-tête borduré `text-sm
   font-medium` de `ChartCard`).
8. Gates verts : `make check-types` = 0 ; `make test-web` (vitest) vert ; `cd apps/web &&
   npm run lint` = 0 ; et si le DTO backend change (items 4/6) : `cd apps/go-api && go test
   ./...` = 0, `make generate-types` régénère sans diff non commité, `make go-api-lint` = 0.
9. Vérification NAVIGATEUR (chrome-devtools, dev local `:8000`) sur des scopes réels des deux
   états (plein historique vs filtré), captures consignées au journal du plan.

---

## 2. Constat sur pièces — état actuel (fichier:ligne réels au 2026-07-16)

**Frontend — `apps/web/src/features/explorer/`**

- **Socle + orchestration.** `ExplorerBriefingStrip.tsx` : `formatPeriod` (`:41-55`) formate la
  période avec `Intl.DateTimeFormat(…, { day:'numeric', month:'short' })` → **PAS d'année**
  (`:47-50`). La carte « Matchs » rend `period` en `sub` (`:101-106`). Deltas socle « vs
  habituel » : Bilan `formatSignedPoints(baseline.delta_win_rate)` (`:131`), FDA
  `formatSignedFixed(baseline.delta_kda, 2)` (`:158`), Perf `formatSignedFixed(baseline.delta_perf,
  0)` (`:180`). Titres des micro-tuiles socle : `text-3xs uppercase tracking-wide
  text-muted-foreground` (`BriefingTile`, `:64-76`). Modules rendus via `ExplorerBriefingModules`
  (`:210`) sauf `low_sample`.
- **Modules conditionnels.** `ExplorerBriefingModules.tsx` :
  - `DIM_TITLE_KEY` (`:34-38`) mappe `playlist → 'explorer.briefing.dim_playlist'`.
  - `DimensionRow` (`:94-130`) : flèche `▲/▼/=` selon `signOf(dw)` (`:97`) + rendu
    `{arrow} {formatSignedPoints(dw)}` (`:113`) → affiche **« = ±0 pts »** quand `dw==0`.
  - `TrendCard` (`:134-155`) = `TimeseriesLineChart` avec `title` → en-tête `ChartCard`
    (**référence esthétique item 7**).
  - `RankedCard` (`:159-204`) : titre `t('explorer.briefing.ranked_title')` (`:166`) ; Δ
    cumulé `formatSignedFixed(ranked.delta_sum, 0)` (`:178`, **le « −1380 » brut**) ; bloc
    attendu vs réel (`:182-199`). Titres de carte en `text-3xs uppercase …` (`:81`, `:165`).
- **Helpers purs.** `ExplorerBriefing.logic.ts` : `formatSignedPoints` renvoie `'±0 pts'`
  quand nul (`:29`) ; `formatSignedFixed` renvoie `'±0.00'` quand nul (`:17`). Tests :
  `ExplorerBriefing.logic.test.ts`.
- **Parent.** `ExplorerPage.matchesMode.tsx` : `<ExplorerBriefingStrip briefing=… />` (`:372`).
  Le filtre contexte existe déjà (`squadScope: ''|'solo'|'squad'`, `:219-237`) avec compteurs
  `squadCountByValue` — donne le vocabulaire UI FR « Solo »/« Escouade »
  (`explorer.filters.context_solo/context_squad`).

**i18n — `apps/web/src/lib/i18n/manifests/explorer.toml`** (régénérer via
`node apps/web/scripts/build_i18n_manifests.mjs`)

- `explorer.filters.playlist` (`:151-153`) = FR **« Sélection »** / EN « Playlist » →
  **convention FR du projet pour playlist** (à répliquer pour la dimension).
- `explorer.briefing.dim_playlist` (`:899-901`) = FR « Par playlist » / EN « By playlist ».
- `explorer.briefing.ranked_title` (`:911-913`) = FR « Pronostic » / EN « Prognosis ».
- `explorer.briefing.ranked_delta` (`:915-917`) = FR « Δ classement cumulé ».
- `explorer.briefing.ranked_expected` / `ranked_actual` / `ranked_expected_vs_actual`
  (`:919-929`).
- `explorer.briefing.vs_baseline` (`:865-867`) = FR « vs habituel ».

**Backend — Go**

- Domain `internal/domain/explorer_briefing.go` : `ExplorerBriefingRanked` (`:120-133`) porte
  `RatingKind` (« csr »|« lusr »), `DeltaSum float64`, `ExpectedWinRate *float64`,
  `ActualWinRate`, `MatchesWithPrediction`. **Ni palier de début/fin, ni moyenne par match.**
  `ExplorerBriefingDimensionEntry.DeltaWinRate float64` (`:98`, toujours émis).
  `ExplorerBriefingBaseline` (`:76-84`) : `Matches`, `DeltaWinRate/KDA/Perf`.
  `ExplorerBriefingScope.Matches` (`:53`). **Aucun bloc contexte solo/escouade.**
- Service `internal/service/match_history_service_briefing.go` :
  - `buildBriefingRanked` (`:367-398`) consomme `SkillExpectedWinProb` (par row) +
    `scopedKPIs.RankDelta` (`:380-382`) → `DeltaSum = rd.Value` (`:394-396`). **N'utilise PAS
    `SkillTierLabel`** (pourtant présent sur les raw rows).
  - `buildDimension` (`:236-278`) : `DeltaWinRate` via `breakdown.CompareByKey` (`:266`) =
    WR(scope groupe) − WR(historique groupe) → **0 par construction quand scope = historique**.
  - `buildBriefingBaseline` (`:143-164`) : deltas socle = valeur(scope) − valeur(baseline) →
    **0 quand scope = baseline** (aucun filtre).
  - Seuils nommés : `MinDimensionGroupMatches = 10` (`:37`), `minTrendMatches = 20` (`:40`).
- Raw rows `internal/domain/match_history.go` : `MatchHistoryRawRow` porte `IsWithFriends bool`
  (`:35`, **signal solo/escouade**), `SkillTierLabel *string` (`:52`, ex. « Diamant IV », déjà
  résolu FR), `SkillRatingType *string` (`:51`, « LUSR »|« CSR »), `SkillExpectedWinProb`
  (`:55`). Peuplées en `match_history_repo.go` Q5 (`:222-226`).
- `internal/domain/squad_v2.go` `RankDelta` (`:167-171`) : `Kind`, `Value` (somme signée des
  per-match deltas du scope), `Count` (nb de matchs du Kind retenu). → **la moyenne par match
  = `Value / Count` est déjà calculable** (mais `Count` n'est pas exposé dans le DTO briefing).
- Grade ladder RÉUTILISABLE : `internal/analysis/skill_v2/tier.go` `InferTier` /
  `FormatTierLabel(mu, boundaries)` (`:110-149`) — mais opère sur **μ TrueSkill natif**, pas sur
  un delta. **Ne pas re-dériver μ→grade côté briefing** : les raw rows portent déjà le palier
  formaté FR par match (`SkillTierLabel`), source directe et déjà localisée.

**Utilitaires réutilisables**

- Date : `apps/web/src/lib/formatters/date.ts` `formatDate(value, locale, opts?, fallback?)`
  (`:31-40`) — défaut `dateStyle:'medium'` = **inclut l'année** (« 29 avr. 2026 »). À réutiliser
  pour item 5 (ne pas ré-instancier un `Intl.DateTimeFormat` local).
- En-tête de carte de référence (item 7) : `apps/web/src/components/charts/ChartCard.tsx`
  (`:125-131`) — `<div className="flex-none border-b border-border px-3 py-2 text-sm
  font-medium">{title}</div>` dans une carte `rounded-lg border border-border bg-card`.
- Primitive carte : `apps/web/src/components/cards/KpiCard.tsx` (`:33-47`) — chrome commun
  (bordure + `bg-card` + coins arrondis + accent 3px), **sans slot d'en-tête** (le titre est
  aujourd'hui rendu à la main par chaque carte).

**Conclusion du constat.** Items 1, 2, 3, 5, 7 sont **frontend-only** (i18n + composants +
un garde dérivé de données déjà servies). Items 4 et 6 exigent une **enrichissement du DTO
backend** (le briefing agrège sur TOUT le scope filtré, pas sur la page de table paginée : le
front ne peut pas reconstituer paliers début/fin ni split solo/escouade depuis les lignes
visibles). Le mapping μ→grade existe mais n'est pas la bonne source ; `SkillTierLabel` par
match l'est.

---

## 3. Décisions — PRÉ-TRANCHÉES et À CONFIRMER

### Pré-tranchées (fermes — ne pas re-débattre en exécution)

- **P-1 (item 1 — cause & remède).** Le « ±0 » est **justifié mathématiquement, pas un bug** :
  quand le scope = tout l'historique, la référence « habituel » EST le scope → tout delta = 0.
  Remède : **masquer** le delta « vs habituel » (socle + lignes de dimension : valeur ET
  flèche) quand `scope.matches === baseline.matches` (le scope est un sous-ensemble de la
  baseline ; cardinalités égales ⟺ ensembles identiques ⟺ deltas nuls). **Frontend-only**, via
  un booléen dérivé `isFullHistoryScope` calculé une fois et passé aux deux composants. Aucun
  changement backend. (Un filtre ne peut que rétrécir : sous-ensemble de même taille = ensemble
  total — propriété sûre.)
- **P-2 (item 2).** Dimension playlist : FR « Par sélection » / EN « By playlist », par
  cohérence avec `explorer.filters.playlist`. Modification i18n pure (`dim_playlist`).
- **P-3 (item 4 — source des paliers).** La progression de paliers se dérive du
  `SkillTierLabel` du **premier et du dernier match du scope (chronologiques) portant un
  palier** — déjà calculé et localisé FR côté repo. Si aucun match du scope ne porte de palier,
  la ligne progression est **omise** (pas de « — → — »). La moyenne par match = `RankDelta.Value
  / RankDelta.Count`. → DTO `ExplorerBriefingRanked` enrichi de `TierStartLabel *string`,
  `TierEndLabel *string`, `DeltaPerMatch *float64` ; le champ `DeltaSum` **cesse d'être affiché**
  (remplacé) — décision de le retirer ou le conserver : voir Phase 4, item 4g (défaut : le
  retirer, « 0 code mort » CLAUDE.md §7 — après bascule du front).
- **P-4 (item 4 — CSR & LUSR unifiés).** Le mécanisme paliers vaut pour les deux `rating_kind`
  (le `SkillTierLabel` couvre CSR « Diamant IV » ET LUSR). Pas de branchement `slug`/`kind`
  spécifique pour l'affichage ; brancher sur la présence des données.
- **P-5 (item 6 — source & emplacement).** Le split solo/escouade se calcule côté **backend**
  sur les raw rows du scope (`IsWithFriends`), nouveau bloc
  `ExplorerBriefing.ContextSplit *ExplorerBriefingContextSplit` (nil si non pertinent). Émis
  UNIQUEMENT si les deux sous-groupes (solo & escouade) atteignent le seuil D-B ET si le scope
  n'est pas déjà réduit à un seul contexte (les deux sous-groupes non vides). Front : une carte
  conditionnelle rendue seulement si le bloc est présent.
- **P-6 (item 7 — cible & périmètre).** Cible = l'en-tête bordurée de `ChartCard`
  (`text-sm font-medium` + `border-b`). Introduire un petit wrapper partagé
  `BriefingSectionCard` (carte `rounded-lg border border-border bg-card` + en-tête bordurée)
  et l'appliquer aux **cartes-sections** : dimensions, module rétrospectif, carte solo/escouade.
  Le bloc « Tendance » (déjà `ChartCard`) reste la référence, inchangé. Les **4 micro-tuiles
  socle** (KpiCard, type « KPI tile » du catalogue) NE sont PAS reformatées en cartes-sections
  (primitive distincte) : leur chrome (bordure/rayon/`bg-card`) est déjà cohérent ; toute
  volonté d'aligner aussi le socle = Découverte, pas traitée ici.
- **P-7 (multi-titre).** Le module rétrospectif et le split restent gatés par capability
  (`useCapability('ranked')` existant côté front ; `s.rankedCapable` côté service). Aucune
  comparaison `slug ==`. Le split solo/escouade ne dépend d'aucune capability rang (basé sur
  `IsWithFriends`, disponible tous titres) — vérifier qu'il dégrade proprement si un titre
  n'expose pas la notion (raw rows sans `IsWithFriends` fiable → sous-groupes vides → bloc omis).

### À CONFIRMER par l'utilisateur (bloquantes — arbitrage produit)

> Ces choix sont des libellés/seuils, pas des choix techniques. Le plan fournit un DÉFAUT
> recommandé pour que l'exécution ne stalle pas : au démarrage de la phase concernée, si
> l'utilisateur n'a pas tranché, appliquer le défaut et le signaler au point d'étape (report
> valide « blocage nécessitant l'utilisateur » — plan-execution).

- **D-A (item 3 — nom du module).** Le module n'est PAS un pronostic (prédiction future) : c'est
  un **bilan rétrospectif** comparant le résultat réel à ce que le système ATTENDAIT (proba de
  victoire pré-match vs issue réelle). Le pivot depuis le module « classé »/CSR d'origine (V1)
  a eu lieu car le delta CSR par match est nil en base. Propositions :
  - **« Attendu vs réel »** (aligné sur le sous-libellé existant `ranked_expected_vs_actual`) —
    **DÉFAUT RECOMMANDÉ** (concis, déjà présent dans l'UI, décrit exactement le contenu).
  - « Bilan vs attendu »
  - « Attentes vs résultats »
  - « Performance vs attendu »
  Tooltip cible (à poser sur le titre) : « Ce que le système attendait (probabilité de victoire
  avant match) comparé au résultat réel — bilan, pas prédiction. » Parité EN à fournir selon le
  choix (défaut : « Expected vs actual »).
- **D-B (item 6 — seuil d'affichage).** La carte solo/escouade n'apparaît que si CHAQUE
  sous-groupe atteint le seuil. Défaut recommandé : **≥ 10 matchs par sous-groupe** (aligné sur
  `MinDimensionGroupMatches = 10`, cohérence avec la fiabilité minimale d'un groupe de
  dimension). Alternative discutée : ≥ 20 (aligné `minTrendMatches`) si l'on veut une carte plus
  rare/robuste. Confirmer 10 ou 20.
- **D-C (item 4 — formulation).** Formulation de la progression et de la moyenne. Défaut :
  ligne 1 « `TierStartLabel` → `TierEndLabel` » (ex. « Bronze I → Platine VI » ; si début=fin,
  afficher le palier seul, ex. « Platine VI ») ; ligne 2 « ±X,X pt/match » (1 décimale, signe
  explicite, `pt/match`). Nouveau libellé de titre de section pour le classement (remplace
  `ranked_delta` « Δ classement cumulé ») : défaut « Classement ». Confirmer la formulation.

---

## 4. Périmètre

**Dans le périmètre :**
- Frontend `apps/web` : `ExplorerBriefingStrip.tsx`, `ExplorerBriefingModules.tsx`,
  `ExplorerBriefing.logic.ts` (+ tests), manifest `explorer.toml` (+ régénération), un wrapper
  `BriefingSectionCard`, réutilisation de `formatDate`.
- Backend `apps/go-api` (items 4 & 6 uniquement) : `internal/domain/explorer_briefing.go`
  (nouveaux champs `ExplorerBriefingRanked` + nouveau type `ExplorerBriefingContextSplit`),
  `internal/service/match_history_service_briefing.go` (`buildBriefingRanked`,
  nouveau `buildBriefingContextSplit`, câblage dans `buildExplorerBriefing`), tests service +
  handler, régénération OpenAPI (`make generate-types`).
- Vérification navigateur, journal du plan, thought_log.

**Hors périmètre (noter en Découvertes si rencontré, ne pas traiter) :**
- Reformatage des 4 micro-tuiles socle en cartes-sections (P-6).
- Toute refonte de la logique baseline/dimensions/tendance V1 au-delà des 7 items.
- Le module tendance (« Tendance ») : c'est la référence, inchangé.
- Recalcul μ→grade côté briefing (P-3 : on lit `SkillTierLabel`, on ne recalcule pas).
- Ajout d'un filtre/colonne au tableau des matchs, à la page Historique (autre handler, ne pose
  pas `IncludeExplorerBriefing`), ou à d'autres pages.
- Dette lint pré-existante (baseline gelée) ; tout Python (interdit).

---

## 5. Phases (ordre strict — une étape CLOSE avant la suivante)

> Clôture d'étape = gate passé (commandes exactes ci-dessous, sorties propres) + tous les items
> statués `[x]` fait / `[~]` couvert ailleurs (réf) / `[!]` non traité (justif écrite) + plan
> mis à jour + entrée `.ai/thought_log.md` + point d'étape utilisateur. Aucune case vide à la
> clôture. Zéro fix hors périmètre (→ §6 Découvertes).
>
> Notes d'exécution :
> - Vitest `apps/web` tourne HORS sandbox → invoquer avec `dangerouslyDisableSandbox=true`
>   (mémoire `reference_vitest_outside_sandbox`).
> - Après toute édition de `explorer.toml` : régénérer les manifests typés
>   (`node apps/web/scripts/build_i18n_manifests.mjs`) AVANT `make check-types`.
> - Ordre choisi : items rapides (terminologie/format) d'abord, puis le garde delta, puis
>   l'unification de forme, puis les deux items lourds backend, enfin la vérif navigateur —
>   ainsi l'unification de forme (Phase 3) et les nouvelles cartes (Phases 4-5) convergent sur
>   le wrapper partagé sans double travail.

### Phase 0 — Cadrage & branche (rapide)

- [ ] Créer `feat/explorer-briefing-v2` depuis `main` à jour (`git fetch` ; vérifier
      `git log --oneline -1 origin/main`).
- [ ] Relire §2 (constat) sur pièces : rouvrir chaque fichier:ligne cité et confirmer qu'il n'a
      pas bougé depuis le 2026-07-16 (le code a pu être modifié).
- [ ] Confirmer l'état des décisions AWAIT-USER (D-A, D-B, D-C) : noter au journal la valeur
      retenue (confirmée ou défaut).

Gate Phase 0 : `git branch --show-current` = `feat/explorer-briefing-v2` ; constat re-vérifié ;
décisions notées.

### Phase 1 — Terminologie, renommage & année (rapide, frontend-only) — items 2, 3, 5

- [ ] **1a (item 2).** `explorer.toml` `explorer.briefing.dim_playlist` → FR « Par sélection »
      (EN inchangé « By playlist »). Régénérer les manifests.
- [ ] **1b (item 3).** `explorer.toml` : renommer la valeur de `explorer.briefing.ranked_title`
      selon D-A (défaut FR « Attendu vs réel » / EN « Expected vs actual ») ; ajouter une clé
      tooltip `explorer.briefing.ranked_tooltip` (FR/EN, texte D-A). Le renommage du libellé de
      la ligne Δ (`ranked_delta`) est traité en Phase 4 (item 4). Poser le tooltip sur le titre
      de la carte (via le slot titre du wrapper Phase 3 s'il existe déjà, sinon note pour Phase 3).
- [ ] **1c (item 5).** `ExplorerBriefingStrip.tsx` `formatPeriod` (`:41-55`) : remplacer le
      `Intl.DateTimeFormat` local par `formatDate` (`lib/formatters/date.ts`) avec une année.
      Cible : « 3 – 12 mars 2025 » (année affichée une fois si début/fin même année ; sinon
      « 3 mars 2024 – 12 janv. 2025 »). Conserver le tiret d'intervalle et le cas start==end.
- [ ] **1d.** Garde-rail terminologie : ajouter/étendre un test (ou grep de clôture) vérifiant
      l'absence des littéraux « Par playlist » et « Pronostic »/« Prognosis » dans le manifest et
      les composants briefing.

Gate Phase 1 : `node apps/web/scripts/build_i18n_manifests.mjs` sans diff inattendu ;
`make check-types` = 0 ; `make test-web` (dangerouslyDisableSandbox) vert ;
`cd apps/web && npm run lint` = 0 ; grep de clôture : 0 occurrence « Par playlist » /
« Pronostic » / « Prognosis » sous `apps/web/src/features/explorer` et dans `explorer.toml`.

### Phase 2 — Delta « vs habituel » dégénéré (moyen, frontend-only) — item 1

- [ ] **2a.** Dériver `isFullHistoryScope = scope.matches != null && baseline?.matches ===
      scope.matches` dans `ExplorerBriefingStrip.tsx` (le composant a `scope` et `baseline`).
- [ ] **2b.** Socle : quand `isFullHistoryScope`, ne rendre AUCUN fragment « vs habituel »
      (Bilan `:124-135`, FDA `:152-163`, Perf `:174-185`) — retirer la valeur, la flèche et le
      «  vs habituel  ». Sinon, comportement V1 inchangé.
- [ ] **2c.** Dimensions : passer `isFullHistoryScope` (ou l'info baseline) à
      `ExplorerBriefingModules` → `DimensionRow` (`:94-130`). Quand vrai, masquer la colonne
      delta (flèche + `formatSignedPoints`) — garder libellé / n matchs / WR / note.
- [ ] **2d.** Vérifier que `formatSignedPoints`/`formatSignedFixed` restent inchangés (le « ±0 »
      reste correct pour le cas rare d'un delta réellement nul SOUS filtre actif — on ne masque
      QUE le cas plein-historique, pas tout zéro). Ajouter un test de rendu / logique couvrant
      les deux états (plein historique → pas de delta ; filtré → delta présent).

Gate Phase 2 : `make check-types` = 0 ; `make test-web` vert (nouveau test des deux états) ;
`npm run lint` = 0.

### Phase 3 — Unification de mise en forme des cartes-sections (moyen, frontend-only) — item 7

- [ ] **3a.** Créer `BriefingSectionCard` (dans `features/explorer/`) : carte `rounded-lg border
      border-border bg-card` + en-tête bordurée `flex-none border-b border-border px-3 py-2
      text-sm font-medium` (miroir de `ChartCard` `:125-131`), avec slot titre (acceptant un
      `ReactNode` pour un `InfoTooltip` — cf. `ChartCardProps.title`) et slot contenu. Tokens
      sémantiques uniquement (aucune couleur hex/Tailwind couleur — skill `color-tokens`).
- [ ] **3b.** Migrer `DimensionCard` (`:76-92`) et `RankedCard` (`:159-204`) vers
      `BriefingSectionCard` (remplacer les titres `text-3xs uppercase …` par l'en-tête bordurée).
      Poser le tooltip du module rétrospectif (1b) sur son titre ici.
- [ ] **3c.** Vérifier visuellement la cohérence avec « Tendance » (même graisse/bordure de
      titre). Ne PAS toucher les micro-tuiles socle (P-6).
- [ ] **3d.** Garde-rail anti-divergence (CLAUDE.md §6 « ≤ 2 copies ») : le pattern d'en-tête
      bordurée existe désormais en 2 endroits canoniques (`ChartCard` + `BriefingSectionCard`) ;
      documenter en commentaire que toute 3ᵉ carte-section du briefing DOIT passer par
      `BriefingSectionCard` (les cartes des Phases 4-5 s'y conforment).

Gate Phase 3 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 ; revue
visuelle (Phase 6) confirmera l'alignement.

### Phase 4 — Classement en grades (lourd, backend + frontend) — item 4

- [ ] **4a (domain).** `ExplorerBriefingRanked` (`explorer_briefing.go:120-133`) : ajouter
      `TierStartLabel *string`, `TierEndLabel *string`, `DeltaPerMatch *float64` (tags JSON
      `omitempty`, commentaires d'unité). Décider du sort de `DeltaSum` (item 4g).
- [ ] **4b (service).** `buildBriefingRanked` (`match_history_service_briefing.go:367-398`) :
      scanner les rows du scope triées par `StartTime` pour extraire le premier et le dernier
      `SkillTierLabel` non nil → `TierStartLabel`/`TierEndLabel` ; calculer `DeltaPerMatch =
      rd.Value / rd.Count` si `rd != nil && rd.Count > 0` (sinon nil). Journaliser en
      `slog.DebugContext` si un palier attendu est absent (dégradation best-effort documentée,
      jamais d'erreur avalée — CLAUDE.md §3). Ne PAS recalculer μ→grade.
- [ ] **4c (tests service).** `match_history_service_briefing_test.go` : cas paliers présents
      (start/end + moyenne) ; cas aucun palier (labels nil, moyenne éventuelle présente) ; cas
      `rd == nil` (moyenne nil) ; MAJ de l'assertion existante sur `DeltaSum` (`:150-162`) selon
      4g. Dataset hétérogène réaliste (mémoire `feedback_integration_tests_realistic_datasets`).
- [ ] **4d (OpenAPI).** `make generate-types` → `apps/web/src/lib/api/generated.ts` régénéré ;
      vérifier le test de drift OpenAPI (`internal/api/openapi_schema_drift_test.go`) vert.
- [ ] **4e (frontend).** `RankedCard` : remplacer le rendu `formatSignedFixed(ranked.delta_sum,
      0)` (`:178`) par : ligne « `tier_start_label` → `tier_end_label` » (si les deux présents ;
      si égaux, palier seul ; si absents, ligne omise) + ligne « `formatSignedFixed(delta_per_match,
      1)` pt/match » (si présent). Nouveau libellé de section D-C (défaut « Classement »).
      Nouvelles clés i18n `explorer.briefing.ranked_progress_*` / `ranked_per_match` (FR/EN).
- [ ] **4f (frontend).** Retirer tout affichage résiduel du nombre cumulé brut ; conserver le
      bloc « attendu vs réel » (indépendant de cet item).
- [ ] **4g (nettoyage).** Retirer `DeltaSum` du DTO et de ses lecteurs si plus aucun consommateur
      (défaut, « 0 code mort ») OU documenter par commentaire pourquoi il reste (source unique de
      `DeltaPerMatch` déjà dérivé côté service → normalement retirable). Mettre à jour les tests.

Gate Phase 4 : `cd apps/go-api && go test ./...` = 0 ; `make go-api-lint` = 0 ; `make
generate-types` sans diff non commité résiduel ; `make check-types` = 0 ; `make test-web` vert ;
`npm run lint` = 0 ; grep : 0 rendu de `delta_sum` brut dans les composants briefing.

### Phase 5 — Carte contexte solo/escouade conditionnelle (lourd, backend + frontend) — item 6

- [ ] **5a (domain).** Nouveau type `ExplorerBriefingContextSplit` (solo & escouade : `Matches`,
      `WinRate`, optionnellement `KDA`/`AvgPerf` par symétrie avec le socle) + champ
      `ContextSplit *ExplorerBriefingContextSplit` dans `ExplorerBriefing` (JSON `omitempty`,
      commentaire « nil si non pertinent »).
- [ ] **5b (service).** `buildBriefingContextSplit(scope []MatchHistoryRawRow)` : partition sur
      `IsWithFriends`, agréger via l'`aggregateRawStats` existant. Émettre nil si l'un des deux
      sous-groupes < seuil D-B, ou si l'un est vide (scope déjà mono-contexte). Constante nommée
      `minContextSplitMatches` (valeur D-B, pas de magic number — CLAUDE.md §Magic number).
      Câbler dans `buildExplorerBriefing` (`:62-75`) après les autres modules, sous garde
      `!LowSample`.
- [ ] **5c (tests service).** Cas pertinent (les deux ≥ seuil → bloc présent) ; cas mono-contexte
      (un sous-groupe vide → nil) ; cas sous le seuil (→ nil) ; cas low_sample (→ nil car modules
      omis).
- [ ] **5d (OpenAPI).** `make generate-types` → regénérer + drift test vert.
- [ ] **5e (frontend).** Nouvelle carte via `BriefingSectionCard` (Phase 3), rendue seulement si
      `briefing.context_split != null`, dans `ExplorerBriefingModules`. Libellés « Solo » /
      « Escouade » réutilisant les clés existantes (`explorer.filters.context_solo/squad` ou
      `explorer.matches.squad_solo/squad_party`) ; nouveau titre de section i18n
      `explorer.briefing.context_split_title` (FR/EN). WR coloré via `winRateColor` (tokens).
- [ ] **5f (tests frontend).** Rendu présent/absent selon la présence du bloc.

Gate Phase 5 : `cd apps/go-api && go test ./...` = 0 ; `make go-api-lint` = 0 ; `make
generate-types` propre ; `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0.

### Phase 6 — Vérification navigateur & clôture

- [ ] **6a.** Dev local (`make dev`, port `:8000`) ; ouvrir l'Explorer mode Matchs d'un joueur
      réel classé (LUSR/CSR).
- [ ] **6b.** État PLEIN HISTORIQUE (aucun filtre) : vérifier item 1 (aucun delta « vs habituel »
      nulle part, aucune « = ±0 pts »), item 2 (« Par sélection »), item 3 (nom + tooltip
      rétrospectif), item 4 (paliers + pt/match, pas de « −1380 »), item 5 (année),
      item 7 (en-têtes unifiés). Capturer.
- [ ] **6c.** État FILTRÉ (narrowing, ex. une carte / un mode) : les deltas « vs habituel »
      RÉAPPARAISSENT et sont sensés ; la carte solo/escouade apparaît/disparaît selon la
      pertinence (item 6). Capturer.
- [ ] **6d.** Vérifier la dégradation : titre H5 (`ranked` absent → module rétrospectif omis, pas
      de crash), scope mono-contexte (carte solo/escouade omise), scope sans palier (progression
      omise). Consigner captures + verdicts au journal du plan.
- [ ] **6e.** `delivery-checklist` complet ; entrée thought_log finale ; point d'étape
      utilisateur (revue visuelle de validation = merci de confirmer avant tout merge).

Gate Phase 6 : tous les critères §1 vérifiés en navigateur ; captures au journal ; gates §1.8
tous verts une dernière fois en une passe.

---

## 6. Découvertes (à remplir en exécution — ne pas traiter hors périmètre)

- (aucune pour l'instant)

Consigner ici toute anomalie/dette repérée hors des 7 items (ex. socle à aligner, incohérence
baseline, palier `SkillTierLabel` manquant sur des matchs attendus). Ne pas la corriger dans ce
chantier ; l'utilisateur arbitrera un chantier séparé.

---

## 7. Protocole de reprise de session

1. `git branch --show-current` doit être `feat/explorer-briefing-v2` (sinon la retrouver via
   `git log --oneline -10` / `git branch`). Ne jamais reprendre sur `main` ni sur une branche de
   train.
2. Lire ce fichier : la dernière phase dont le **Gate** est coché est close ; reprendre à la
   première non close. Les cases `[ ]` d'une phase non close = travail restant.
3. Lire l'entrée `.ai/thought_log.md` la plus récente de ce chantier (avancement + décisions
   AWAIT-USER retenues).
4. Re-vérifier sur pièces les fichier:ligne de la phase courante (le code a pu bouger) AVANT
   d'éditer ou de cocher (plan-execution : vérifier sur pièces avant de coder ET avant de cocher).
5. Ne jamais commencer une phase N+1 tant que le Gate de N n'est pas vert.

---

## 8. Effort estimé & dépendances

| Item | Sujet | Effort | Couche |
|---|---|---|---|
| 2 | « Par sélection » | Rapide | i18n |
| 3 | Renommage module rétrospectif + tooltip | Rapide | i18n / front (nom = D-A) |
| 5 | Année dans les dates | Rapide | front (réutilise `formatDate`) |
| 1 | Masquer delta « vs habituel » en plein historique | Moyen | front |
| 7 | En-têtes de cartes unifiés | Moyen | front (wrapper partagé) |
| 4 | Classement en grades + moyenne/match | Lourd | **backend DTO** + service + OpenAPI + front (D-C) |
| 6 | Carte solo/escouade conditionnelle | Lourd | **backend DTO** + service + OpenAPI + front (seuil = D-B) |

**Dépendances backend** : items 4 et 6 seulement (nouveaux champs `ExplorerBriefingRanked` +
nouveau `ExplorerBriefingContextSplit`, service, régénération OpenAPI + drift test). Items 1, 2,
3, 5, 7 sont frontend-only. **Dépendances inter-phases** : Phase 3 (wrapper `BriefingSectionCard`)
précède les cartes des Phases 4 et 5 (elles le réutilisent). **Dépendances utilisateur** : D-A
(nom, Phase 1/3), D-B (seuil, Phase 5), D-C (formulation, Phase 4) — chacune avec un défaut pour
ne pas bloquer. **Aucun déploiement prod** dans ce chantier (le merge `main` = deploy auto reste
la décision de l'utilisateur, après revue visuelle 6b-6d).
</content>
