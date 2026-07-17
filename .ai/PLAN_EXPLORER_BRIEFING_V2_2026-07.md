# PLAN — Explorer briefing V2 : ajustements post-revue visuelle

Statut : PLANIFIE (aucune ligne de code écrite — plan rédigé en worktree lecture seule).
Date : 2026-07-16.
Auteur du plan : architecte Opus (worktree isolé).
Révision 1 : 2026-07-16 (revue sur pièces + arbitrages utilisateur) — classement PAR TYPE de
rating (CSR et LUSR séparés, jamais mélangés), tri des dimensions en plein historique (P-8),
helper partagé `formatDateRange`, traitement des paliers de placement (D-D), retrait du
sous-libellé redondant « attendu vs réel ».
Révision 2 : 2026-07-16 (décision produit utilisateur) — le bloc « attendu vs réel »
(`expected_win_prob`) est SUPPRIMÉ de l'UI : donnée jugée non fiable à ce jour. Le module
classé devient la carte « Classement » (progression par type uniquement). D-A tranché/obsolète.
Cartes de remplacement candidates notées en backlog §6, en attente d'arbitrage.
Révision 3 : 2026-07-16 (arbitrage utilisateur) — deux nouvelles cartes VALIDÉES : item 8
« Séries » et item 9 « Moments forts » (Phase 5b, P-9). « Records du scope » écarté. L'idée
MVP/LVP dans le TABLEAU (extrêmes sur tout le scope, pagination serveur) = chantier séparé,
architecture recommandée consignée en §6. Ajout (même jour) : mise à jour du changelog +
« What's new » v7.0 exigée en clôture (Phase 6, item 6e).
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
projet ; remplacer le module « Pronostic » par une carte « Classement » débarrassée du bloc
« attendu vs réel » (`expected_win_prob` non fiable — décision produit 2026-07-16) ; afficher
le classement en paliers connus du joueur plutôt qu'en points bruts ; dater complètement ;
distinguer
solo/escouade quand c'est pertinent ; et présenter toutes ses cartes-sections avec une mise
en forme unifiée. S'y ajoutent deux cartes validées en Révision 3, en remplacement du bloc
attendu/réel supprimé : « Séries » (meilleure/pire série du scope) et « Moments forts »
(compteurs de dominance).

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
3. Le module ex-« Pronostic » est devenu la carte « Classement » (D-A tranché 2026-07-16) :
   le bloc « attendu vs réel » n'est PLUS rendu et ses clés i18n sont supprimées. Aucune
   occurrence résiduelle de « Pronostic »/« Prognosis ».
4. Le classement s'affiche PAR TYPE DE RATING : une ligne par type (CSR, LUSR) suffisamment
   représenté dans le scope, chacune avec progression de paliers connus (ex. « Bronze I →
   Platine VI ») + moyenne par match (ex. « −1,4 pt/match ») — plus aucun nombre cumulé brut
   (« −1380 »), et jamais des paliers de deux systèmes mélangés sur une même ligne.
   Dégradation propre si les paliers de début/fin ne sont pas résolvables ; placement rendu
   selon D-D.
5. La carte « Matchs » affiche la période AVEC l'année (ex. « 3 – 12 mars 2025 »), via un
   nouvel utilitaire partagé `formatDateRange` (basé `Intl.formatRange`) dans
   `lib/formatters/date.ts`.
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
10. En plein historique, les entrées des cartes de dimension sont sélectionnées ET triées par
    taux de victoire (fini l'ordre pseudo-aléatoire hérité du tri par delta nul → clé) ; sous
    filtre actif, tri par delta V1 inchangé.
11. Le briefing ne consomme plus `expected_win_prob` : DTO ranked sans
    `expected_win_rate`/`actual_win_rate`/`matches_with_prediction`, service sans le calcul
    correspondant, lecteurs et clés i18n purgés (« 0 code mort »).
12. Carte « Séries » : meilleure série de victoires + pire série de défaites calculées sur
    TOUT le scope filtré (pas sur la frise cappée à 60), affichée hors low_sample ; segments
    à zéro omis ; carte omise si rien à afficher.
13. Carte « Moments forts » : compteurs `DominanceFlag` du scope (dominations, humiliations,
    remontadas, débandades, contre-remontadas), catégories à zéro omises ; carte omise si
    tous les compteurs sont à zéro.
14. Changelog à jour : les apports Explorer de ce chantier figurent dans l'entrée
    `[Unreleased]` (consolidée v7.0, « What's new ») de `docs/CHANGELOG.md` ET
    `docs/FR/CHANGELOG.md` (parité EN/FR dans le même commit).

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
  (`:31-41`) — formate UNE date isolée ; il ne peut PAS produire « 3 – 12 mars 2025 »
  (année/mois factorisés sur un intervalle). La factorisation est native via
  `Intl.DateTimeFormat.prototype.formatRange` → item 5 = nouveau helper canonique
  `formatDateRange(start, end, locale)` dans ce fichier (PAS un `Intl.DateTimeFormat` local
  dans le composant).
- En-tête de carte de référence (item 7) : `apps/web/src/components/charts/ChartCard.tsx`
  (`:125-131`) — `<div className="flex-none border-b border-border px-3 py-2 text-sm
  font-medium">{title}</div>` dans une carte `rounded-lg border border-border bg-card`.
- Primitive carte : `apps/web/src/components/cards/KpiCard.tsx` (`:33-47`) — chrome commun
  (bordure + `bg-card` + coins arrondis + accent 3px), **sans slot d'en-tête** (le titre est
  aujourd'hui rendu à la main par chaque carte).

**Compléments de revue (Révision 1, vérifiés sur pièces le 2026-07-16) :**

- `breakdown.CompareByKey` trie par `WinRateDelta` décroissant puis **clé** ascendante
  (`compare_by_key.go:61-66`). Scope = historique ⟹ tous les deltas = 0 ⟹ ordre par clé
  (GUID de map pour la dimension carte) : la sélection top/flop devient pseudo-aléatoire.
  Masquer la colonne delta (item 1) ne suffit donc pas — la SÉLECTION doit basculer sur le
  taux de victoire en plein historique (P-8).
- `analysis.ComputeKPIStats` accumule DÉJÀ les deltas de rang **par RatingType**
  (`kpi_stats.go:45-52`) et ne retient que le type majoritaire en sortie (`:186-194`) : le
  split CSR/LUSR (P-3 révisé) n'exige AUCUNE nouvelle donnée — exposer les buckets existants
  via un champ additif.
- `SkillTierLabel` peut valoir « Placement (N restants) » (`match_history.go:56-61`,
  `PlacementDone/PlacementTotal` déjà parsés par row). Le premier match d'un plein historique
  est presque toujours un placement → traitement dédié D-D.
- `MatchHistoryRawRow.SkillRatingType` vaut « LUSR » | « CSR » (`match_history.go:51`) ;
  confronter la casse aux constantes `canonical.RatingType*` avant tout filtre par type
  (vérifier sur pièces en Phase 4).
- Cartes Révision 3 : `DominanceFlag` est déjà porté par les raw rows
  (`match_history.go:64-67`, backfillé par `RunBackfillComebackBadges`) ; la frise
  `outcome_sequence` est CAPPÉE à 60 matchs (`maxOutcomeSequencePoints`, service `:43`) → les
  séries se calculent côté BACKEND sur tout le scope, jamais depuis `outcome_sequence`.
- Limitation ACTÉE (pas de fix dans ce chantier) : `SkillTierLabel` est stocké formaté FR en
  base — sous locale EN, la progression affichera des paliers FR, comme la colonne existante
  du tableau. Les rendus PLACEMENT passent par des clés i18n (D-D) et échappent à cette
  limite.

**Conclusion du constat.** Items 2, 3, 5, 7 sont **frontend-only** (i18n + composants + un
garde dérivé de données déjà servies). Item 1 = frontend + un re-tri service (P-8, aucun
changement de DTO). Items 4 et 6 exigent un **enrichissement du DTO backend** (le briefing
agrège sur TOUT le scope filtré, pas sur la page de table paginée : le front ne peut pas
reconstituer paliers début/fin ni split solo/escouade depuis les lignes visibles). Le mapping
μ→grade existe mais n'est pas la bonne source ; `SkillTierLabel` par match l'est.

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
- **P-3 (item 4 — classement PAR TYPE de rating ; arbitrage utilisateur 2026-07-16).** Le
  module classement émet une entrée PAR TYPE (CSR, LUSR) — jamais une progression mélangeant
  les paliers de deux systèmes (piège du scope mixte : départ LUSR, arrivée CSR). Source :
  les buckets par `RatingType` déjà accumulés dans `analysis.ComputeKPIStats`, exposés via un
  champ additif `domain.KPIStats.RankDeltas []RankDelta` (le `RankDelta` majoritaire reste,
  consommateurs existants intacts). Par type : paliers = `SkillTierLabel` du premier/dernier
  match chronologique du scope portant un palier ET dont `SkillRatingType` correspond au
  type ; moyenne = `Value / Count` du bucket. Seuil de signifiance : constante nommée
  `minRankedKindMatches = 10` (alignée `MinDimensionGroupMatches`) — le type MAJORITAIRE est
  toujours émis même sous le seuil (pas de régression vs V1), les autres types seulement s'ils
  l'atteignent. Si aucun match d'un type ne porte de palier, sa progression est omise (pas de
  « — → — »). `DeltaSum`/`RatingKind` du DTO : remplacés par `Kinds` — sort en Phase 4g
  (défaut : retirer, « 0 code mort » CLAUDE.md §7 — après bascule du front).
- **P-4 (item 4 — affichage unifié).** Même rendu pour tous les types ; le libellé du type est
  la métrique connue du joueur (« CSR » / « LUSR »). Pas de branchement `slug`/`kind`
  spécifique ; brancher sur la présence des données.
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
- **P-8 (item 1 — dimensions en plein historique).** Masquer la colonne delta ne corrige pas la
  SÉLECTION top/flop, qui dégénère quand tous les deltas sont nuls (tri par clé, cf. §2
  compléments). Quand scope = historique (détectable côté service : `len(scope) == len(all)`),
  `buildDimension` re-trie les groupes qualifiés par `Session.WinRate` décroissant (tie-break
  libellé) AVANT `selectTopFlop`. Sous filtre actif : tri par delta V1 inchangé.
  `breakdown.CompareByKey` (analysis partagé, autres consommateurs) n'est PAS modifié — le
  re-tri vit dans le service briefing.
- **P-9 (items 8/9 — règles de calcul ; cartes validées par l'utilisateur 2026-07-16).**
  Séries : tri chronologique (`StartTime`, rows sans date écartées comme pour la frise) ;
  série de victoires = victoires consécutives, rompue par TOUT autre outcome (défaite, nul,
  abandon) ; symétrique pour la pire série de défaites. Émises hors low_sample uniquement
  (comme les autres modules) ; nil si aucune row datée ; segment à zéro omis (scope 100 %
  victoires → pas de « Pire série : 0 »). Moments forts : compter `DominanceFlag` 1..5 sur le
  scope ; bloc nil si tous les compteurs sont à zéro (dégradation par omission). Avant de
  créer des clés i18n : grep des libellés existants des badges dominance (règle « vérifier
  l'existant » — les flags viennent des comeback badges).

### À CONFIRMER par l'utilisateur (bloquantes — arbitrage produit)

> Ces choix sont des libellés/seuils, pas des choix techniques. Le plan fournit un DÉFAUT
> recommandé pour que l'exécution ne stalle pas : au démarrage de la phase concernée, si
> l'utilisateur n'a pas tranché, appliquer le défaut et le signaler au point d'étape (report
> valide « blocage nécessitant l'utilisateur » — plan-execution).

- **D-A — TRANCHÉ 2026-07-16 (décision produit utilisateur, ne plus re-débattre).** Le bloc
  « attendu vs réel » (moyenne des `expected_win_prob` pré-match vs winrate réel) est SUPPRIMÉ
  de l'UI : `expected_win_prob` est jugé non fiable à ce jour. Le module devient la carte
  **« Classement »** (FR) / « Ranking » (EN) et ne porte QUE la progression par type de rating
  (P-3 / D-C / D-D). Pas de tooltip « rétrospectif » (plus rien à expliquer de ce genre).
  Conséquences instruites en Phases 1b et 4 : suppression des clés `ranked_expected*`, du
  calcul service et des champs DTO correspondants (« 0 code mort »). D'autres cartes candidates
  (séries, moments forts, records) sont notées en backlog §6, HORS périmètre tant que
  l'utilisateur n'a pas arbitré.
- **D-B (item 6 — seuil d'affichage).** La carte solo/escouade n'apparaît que si CHAQUE
  sous-groupe atteint le seuil. Défaut recommandé : **≥ 10 matchs par sous-groupe** (aligné sur
  `MinDimensionGroupMatches = 10`, cohérence avec la fiabilité minimale d'un groupe de
  dimension). Alternative discutée : ≥ 20 (aligné `minTrendMatches`) si l'on veut une carte plus
  rare/robuste. Confirmer 10 ou 20.
- **D-C (item 4 — formulation).** Formulation par type de rating. Défaut : une ligne par type,
  préfixée du type — « CSR · Bronze I → Platine VI · −1,4 pt/match » (si début=fin, palier
  seul ; si paliers absents, la moyenne seule ; placement → D-D). Moyenne :
  `formatSignedFixed(…, 1)` + suffixe `pt/match` — NOTE : le helper émet un point décimal
  (« −1.4 »), pas la virgule FR ; défaut = l'accepter (cohérent avec les autres deltas du
  bandeau), sinon localiser (décision d'exécution à consigner au journal). Nouveau libellé de
  titre de section (remplace `ranked_delta` « Δ classement cumulé ») : défaut « Classement ».
  Confirmer la formulation.
- **D-D (item 4 — paliers de placement ; arbitrage utilisateur 2026-07-16 : ni cacher, ni
  brut).** `SkillTierLabel` peut valoir « Placement (N restants) » et le palier de DÉPART d'un
  plein historique est presque toujours un placement (on commence non classé). Défaut retenu :
  - Début en placement → afficher « Placement » SANS compteur (« Placement → Platine VI » se
    lit comme un parcours ; honnête et concis).
  - Fin en placement (joueur encore en placement sur le scope) → afficher « Placement
    (N restants) » AVEC compteur (l'info utile est le reste à jouer).
  - Implémentation BILINGUE : ne pas parser le libellé FR côté front — le backend émet
    `TierStartIsPlacement bool` et `TierEndPlacementRemaining *int` (dérivés de
    `PlacementDone/PlacementTotal` de la row concernée) ; le front rend des clés i18n dédiées
    (`explorer.briefing.placement*`, FR/EN) ; le libellé brut n'est utilisé que hors placement.

---

## 4. Périmètre

**Dans le périmètre :**
- Frontend `apps/web` : `ExplorerBriefingStrip.tsx`, `ExplorerBriefingModules.tsx`,
  `ExplorerBriefing.logic.ts` (+ tests), manifest `explorer.toml` (+ régénération), un wrapper
  `BriefingSectionCard`, nouveau helper `formatDateRange` dans `lib/formatters/date.ts`
  (+ test).
- Backend `apps/go-api` (items 1, 4, 6, 8 & 9) : `internal/domain/explorer_briefing.go`
  (refonte `ExplorerBriefingRanked` par type + nouveaux types `ExplorerBriefingContextSplit`,
  `ExplorerBriefingStreaks`, `ExplorerBriefingDominance`),
  `internal/analysis/kpi_stats.go` (champ additif `KPIStats.RankDeltas []RankDelta` — exposer
  les buckets existants), `internal/service/match_history_service_briefing.go`
  (`buildBriefingRanked` par type, re-tri plein-historique de `buildDimension` (P-8), nouveaux
  `buildBriefingContextSplit` / `buildBriefingStreaks` / `buildBriefingDominance`, câblage dans
  `buildExplorerBriefing`), tests analysis + service + handler, régénération OpenAPI
  (`make generate-types`).
- Vérification navigateur, journal du plan, thought_log.
- Changelog : `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`, entrée `[Unreleased]` v7.0
  (item 6e — le « What's new » in-app est rendu depuis ce fichier).

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

- [x] Créer `feat/explorer-briefing-v2` depuis `main` à jour (`git fetch` ; vérifier
      `git log --oneline -1 origin/main`). — Branche déjà créée (item git fait par le
      superviseur) ; `git branch --show-current` = `feat/explorer-briefing-v2` ; worktree propre.
- [x] Relire §2 (constat) sur pièces : rouvrir chaque fichier:ligne cité et confirmer qu'il n'a
      pas bougé depuis le 2026-07-16 (le code a pu être modifié). — Re-vérifié 2026-07-16.
      Fichiers frontend EXACTS (formatPeriod `:41-55` sans année ; dim_playlist `:899-901`
      « Par playlist » ; ranked_title `:911-913` « Pronostic »/« Prognosis » ; ranked_delta
      `:915-917` ; ranked_expected* `:919-929` ; vs_baseline `:865-867` ; logic.ts
      `formatSignedFixed:17` / `formatSignedPoints:29` ; DimensionRow `:94-130` ; RankedCard
      `:159-204`). Backend : `ExplorerBriefingRanked` désormais à `:121-133` (le plan citait
      `:120-133` — décalage d'1 ligne, champs `RatingKind`/`DeltaSum`/`ExpectedWinRate`
      inchangés ; voir §6 Découverte-1). Les commits « sweep recovery player-DB » de la branche
      de base (021f24a7b, af6ccdd6f, 8750dfcc8) N'ONT touché AUCUN fichier explorer/briefing
      (service briefing + domain datent du Lot D 01f71104b, kpi_stats du chantier H5) → aucun
      impact sur l'explorer, confirmé.
- [x] Confirmer l'état des décisions AWAIT-USER restantes (D-B, D-C) : noter au journal la
      valeur retenue (confirmée ou défaut). D-A et D-D sont tranchées (2026-07-16). —
      **D-B = défaut 10** (≥ 10 matchs par sous-groupe, aligné `MinDimensionGroupMatches`).
      **D-C = défaut** (une ligne par type préfixée : « CSR · Bronze I → Platine VI · −1.4
      pt/match » ; point décimal accepté ; titre de section « Classement »). Décisions
      transmises par le superviseur au lancement du chantier ; s'appliquent par défaut.

Gate Phase 0 : `git branch --show-current` = `feat/explorer-briefing-v2` ; constat re-vérifié ;
décisions notées. — PASSÉ (2026-07-16).

### Phase 1 — Terminologie, renommage & année (rapide, frontend-only) — items 2, 3, 5

- [x] **1a (item 2).** `explorer.toml` `explorer.briefing.dim_playlist` → FR « Par sélection »
      (EN inchangé « By playlist »). Régénérer les manifests. — FAIT ; `explorer.ts` régénéré
      (diff = seule valeur FR changée).
- [x] **1b (item 3, D-A tranché).** `explorer.toml` : `explorer.briefing.ranked_title` → FR
      « Classement » / EN « Ranking ». PAS de clé tooltip. La suppression du bloc « attendu vs
      réel » (rendu + clés `ranked_expected*` + champs DTO) se fait en Phase 4 — le composant
      les consomme encore, les clés restent jusque-là pour ne pas casser le typecheck ;
      `ranked_delta` idem (Phase 4). — FAIT ; seul le libellé de titre changé, clés
      `ranked_expected*`/`ranked_delta` laissées intactes pour la Phase 4.
- [x] **1c (item 5).** Ajouter `formatDateRange(start, end, locale, fallback?)` dans
      `lib/formatters/date.ts` (mettre à jour l'en-tête de doc du fichier), basé sur
      `Intl.DateTimeFormat.prototype.formatRange` avec `{ day:'numeric', month:'short',
      year:'numeric' }` — factorisation année/mois native (« 3 – 12 mars 2025 » ;
      « 3 mars 2024 – 12 janv. 2025 » si années différentes ; date simple si end absent ou
      égal à start). Test unitaire du helper. Puis `ExplorerBriefingStrip.tsx` `formatPeriod`
      (`:41-55`) : remplacer le `Intl.DateTimeFormat` local par ce helper. — FAIT : helper
      ajouté + ré-exporté depuis `formatters/index.ts` ; en-tête de doc du fichier mise à
      jour ; 4 tests dans `formatters.test.ts` (même mois factorisé, années différentes, date
      simple, fallback) ; `formatPeriod` délègue désormais à `formatDateRange`.
- [x] **1d.** Garde-rail terminologie : ajouter/étendre un test (ou grep de clôture) vérifiant
      l'absence des littéraux « Par playlist » et « Pronostic »/« Prognosis » dans le manifest et
      les composants briefing. — FAIT : nouveau test node
      `features/explorer/explorerBriefingTerminology.guard.test.ts` (scanne `explorer.toml` +
      composants `*briefing*`, hors fichiers de test).

Gate Phase 1 : `node apps/web/scripts/build_i18n_manifests.mjs` sans diff inattendu ;
`make check-types` = 0 ; `make test-web` (dangerouslyDisableSandbox) vert ;
`cd apps/web && npm run lint` = 0 ; grep de clôture : 0 occurrence « Par playlist » /
« Pronostic » / « Prognosis » sous `apps/web/src/features/explorer` et dans `explorer.toml`.
— PASSÉ (2026-07-16) : manifests régénérés (seul `explorer.ts` modifié, 2 valeurs) ;
`make check-types` = 0 ; `make test-web` = 256 fichiers / 2172 passés / 14 skipped / 0 fail ;
`npm run lint` = 0 erreur (68 warnings baseline pré-existants, aucun sur les fichiers touchés) ;
greps de clôture = 0 occurrence (hors le garde-rail lui-même).

### Phase 2 — Delta « vs habituel » dégénéré (moyen, frontend + service) — item 1

- [x] **2a.** Dériver `isFullHistoryScope = scope.matches != null && baseline?.matches ===
      scope.matches` dans `ExplorerBriefingStrip.tsx` (le composant a `scope` et `baseline`). —
      FAIT : helper PUR `isFullHistoryScope(scopeMatches, baselineMatches)` extrait dans
      `ExplorerBriefing.logic.ts` (testable isolé) ; le Strip dérive `fullHistory` une fois et
      le réutilise pour le socle ET le passe aux modules.
- [x] **2b.** Socle : quand `isFullHistoryScope`, ne rendre AUCUN fragment « vs habituel »
      (Bilan `:124-135`, FDA `:152-163`, Perf `:174-185`) — retirer la valeur, la flèche et le
      «  vs habituel  ». Sinon, comportement V1 inchangé. — FAIT : les 3 fragments socle gatés
      `… && !fullHistory` (Bilan) / `… && !fullHistory ?` (FDA) / `… && !fullHistory && …`
      (Perf). Le socle (Matchs/WR/FDA/Perf + V-D-N) reste rendu.
- [x] **2c.** Dimensions : passer `isFullHistoryScope` (ou l'info baseline) à
      `ExplorerBriefingModules` → `DimensionRow` (`:94-130`). Quand vrai, masquer la colonne
      delta (flèche + `formatSignedPoints`) — garder libellé / n matchs / WR / note. — FAIT :
      prop `hideDelta` threadée `ExplorerBriefingModules` → `DimensionCard` → `DimensionRow` ;
      la colonne delta est rendue sous `{!hideDelta && (…)}`, le reste (label / n matchs / WR /
      note) inchangé.
- [x] **2d.** Vérifier que `formatSignedPoints`/`formatSignedFixed` restent inchangés (le « ±0 »
      reste correct pour le cas rare d'un delta réellement nul SOUS filtre actif — on ne masque
      QUE le cas plein-historique, pas tout zéro). Ajouter un test de rendu / logique couvrant
      les deux états (plein historique → pas de delta ; filtré → delta présent). — FAIT :
      `formatSignedPoints`/`formatSignedFixed` NON modifiés (masquage porté par le flag, pas par
      la valeur). Tests : `ExplorerBriefing.logic.test.ts` (nouveau describe `isFullHistoryScope`,
      4 cas) + `ExplorerBriefingStrip.test.tsx` (nouveau, 2 états : filtré → `+30 pts`/`+20 pts`/
      `vs_baseline` présents ; plein historique → absents, socle + dimensions conservés — deltas
      posés NON NULS pour prouver que le masquage dépend du flag).
- [x] **2e (P-8, service).** `buildBriefingDimensions`/`buildDimension`
      (`match_history_service_briefing.go:211-278`) : quand `len(scope) == len(all)` (plein
      historique), re-trier les groupes qualifiés par `Session.WinRate` décroissant (tie-break
      libellé) avant `selectTopFlop` ; sous filtre, comportement V1 inchangé. Ne PAS toucher
      `breakdown.CompareByKey`. Test service des deux états (plein historique → ordre par
      taux de victoire ; filtré → ordre par delta). — FAIT : booléen nommé `fullHistory :=
      len(scope) == len(all)` calculé dans `buildBriefingDimensions`, passé en param à
      `buildDimension` ; re-tri `sort.SliceStable` par `Session.WinRate` desc + tie-break
      `Label` AVANT `selectTopFlop`, uniquement si `fullHistory`. `breakdown.CompareByKey`
      inchangé. Tests : `TestBuildDimension_FullHistorySortsByWinRate` (MapIDs a1<m1<z1 en ordre
      INVERSE du WR → prouve le re-tri) + `TestBuildDimension_FilteredSortsByDelta` (scope⊊all,
      WR-desc ≠ delta-desc → ordre par delta conservé).

Gate Phase 2 : `make check-types` = 0 ; `make test-web` vert (nouveau test des deux états) ;
`npm run lint` = 0 ; `cd apps/go-api && go test ./...` = 0 (test 2e inclus) ;
`make go-api-lint` = 0. — PASSÉ (2026-07-16, depuis la racine du worktree) : `make check-types`
= 0 ; `make test-web` = 257 fichiers / 2178 passés / 14 skipped / 0 échec (dont
`ExplorerBriefingStrip.test.tsx` neuf) ; `npm run lint` = 0 erreur (68 warnings baseline
pré-existants, 0 sur les fichiers touchés) ; `go test ./...` = exit 0, 111 packages ok, 0 FAIL
(tests 2e inclus) ; `make go-api-lint` = exit 0 (+ `go vet ./internal/service/...` = 0).

### Phase 3 — Unification de mise en forme des cartes-sections (moyen, frontend-only) — item 7

- [x] **3a.** Créé `BriefingSectionCard` dans `apps/web/src/features/explorer/BriefingSectionCard.tsx` :
      carte `rounded-lg border border-border bg-card` + en-tête bordurée `flex-none border-b
      border-border px-3 py-2 text-sm font-medium` (byte-identique à `ChartCard:128`), slot
      titre `ReactNode` (compatible InfoTooltip futur) + slot contenu (`p-3`, comme le corps
      `ChartCard`). Tokens sémantiques uniquement (aucun hex/Tailwind couleur).
- [x] **3b.** Migré `DimensionCard` et `RankedCard` (`ExplorerBriefingModules.tsx`, lignes
      re-vérifiées après Phase 2) vers `BriefingSectionCard` : titres de carte `text-3xs
      uppercase …` supprimés (déplacés dans l'en-tête bordurée). Import `KpiCard` retiré (plus
      aucun usage — 0 code mort). AUCUN tooltip posé (D-A). Sous-labels internes du corps
      `RankedCard` (`ranked_delta`/`ranked_expected_vs_actual`) laissés en l'état — ils relèvent
      de la refonte du corps en Phase 4 (D-A), hors périmètre Phase 3.
- [x] **3c.** Cohérence avec « Tendance » vérifiée par lecture : l'en-tête de `BriefingSectionCard`
      réutilise la className EXACTE de l'en-tête `ChartCard` (`flex-none border-b border-border
      px-3 py-2 text-sm font-medium`) que rend `TrendCard` via `TimeseriesLineChart`. Même
      graisse (`font-medium`), même bordure. Micro-tuiles socle NON touchées (P-6). Alignement
      pixel confirmé en revue visuelle Phase 6.
- [x] **3d.** Garde-rail anti-divergence documenté en tête de `BriefingSectionCard.tsx` (bloc
      « GARDE-RAIL ANTI-DIVERGENCE », CLAUDE.md §6) : le pattern d'en-tête bordurée existe en 2
      endroits canoniques (`ChartCard` + `BriefingSectionCard`) ; toute 3ᵉ carte-section du
      briefing (Phases 4/5/5b) DOIT passer par `BriefingSectionCard`, jamais ré-inliner un
      `text-3xs uppercase …` ni recopier l'en-tête à la main.

Gate Phase 3 : `make check-types` = 0 (CLOS) ; `make test-web` vert 257 fichiers / 2178 passés /
14 skipped / 0 échec (CLOS) ; `npm run lint` = 0 erreur (68 warnings baseline, 0 sur fichiers
touchés) (CLOS) ; revue visuelle (Phase 6) confirmera l'alignement.

### Phase 4 — Classement en grades PAR TYPE (lourd, backend + frontend) — item 4

- [x] **4a (domain + analysis).** FAIT. `explorer_briefing.go` : type
      `ExplorerBriefingRankedKind` ajouté (Kind/Matches/TierStartLabel/TierEndLabel/
      TierStartIsPlacement/TierEndPlacementRemaining/DeltaPerMatch, JSON `omitempty` +
      commentaires d'unité) ; `ExplorerBriefingRanked` réduit à `Kinds []…` (RatingKind/DeltaSum/
      ExpectedWinRate/ActualWinRate/MatchesWithPrediction TOUS retirés — 4a+4g fusionnés,
      état final = `Kinds` seul). `domain.KPIStats.RankDeltas []RankDelta` additif
      (`squad_v2.go`), peuplé dans `analysis/kpi_stats.go` (ordre déterministe : Count desc,
      tie-break CSR ; majoritaire en tête = cohérent avec `RankDelta` singulier conservé).
      Tests analysis : `TestComputeKPIStats_RankDeltas_SplitByTypeDeterministic` +
      `_NilWhenNoRatedMatches`.
- [x] **4b (service).** FAIT. `buildBriefingRanked` réécrit + extrait dans nouveau fichier
      `match_history_service_briefing_ranked.go` (fichier principal repassait à 600 L > 500 —
      CLAUDE.md §5 ; extraction plutôt qu'accroître la dette). Émet une entrée par bucket de
      `KPIStats.RankDeltas` (majoritaire toujours ; secondaire si `Count >= minRankedKindMatches`
      = 10, constante nommée). Scan des rows de CE type triées `StartTime` ; casse confrontée
      sur pièces : raw rows « CSR »/« LUSR » (maj) vs `canonical.RatingType` « csr »/« lusr »
      (min) → `strings.EqualFold`. Premier/dernier `SkillTierLabel` non nil → labels ; flags
      placement D-D via `PlacementDone/PlacementTotal` (remaining = Total−Done, clampé ≥ 0).
      `DeltaPerMatch = Value/Count`. `slog.DebugContext` si aucun palier. Calcul attendu/réel
      supprimé. `analysis.ExpectedVsActual` : plus aucun consommateur → `expected_win.go` +
      `expected_win_test.go` SUPPRIMÉS. `MatchHistoryRawRow.SkillExpectedWinProb` + lecture repo
      Q5 : CONSERVÉS (lecteurs restants = session_page_service, match_history_service_enrich —
      grep sur pièces), notés ici.
- [x] **4c (tests service).** FAIT. Nouveaux tests : `_RankedMonoTypeProgression` (+ gating
      rankedCapable) ; `_RankedMixedTypesNeverMerged` (CSR+LUSR, paliers jamais croisés) ;
      `_RankedSecondaryTypeBelowThresholdOmitted` ; `_RankedNoTierLabels` (labels nil, moyenne
      présente) ; `_RankedStartInPlacement` ; `_RankedEndInPlacement` ;
      `_RankedNilWhenNoRankDeltas`. Anciens tests DeltaSum/ExpectedWinRate/predictions-only
      supprimés. Helper `briefingRankedRaw`. Tous les call-sites `buildExplorerBriefing`
      threadés `context.Background()`.
- [x] **4d (OpenAPI).** FAIT. `openapi.yaml` (manuel) : `ExplorerBriefingRanked` remplacé +
      `ExplorerBriefingRankedKind` ajouté (YAML émis exact via `OPENAPI_EMIT_*`). `make
      generate-types` régénéré ; `types.ts` : export `ExplorerBriefingRankedKind` ajouté ;
      `TestOpenAPISchemaDrift` vert (0 MISSING).
- [x] **4e (frontend).** FAIT. `RankedCard` réécrit : `<ul>` d'une ligne `RankedKindRow` par
      entrée de `kinds` — « {KIND maj} · {progression} · {moyenne} ». `rankedProgression`
      compose « début → fin » (égaux → palier seul ; absents → segment omis ; placement → clés
      i18n `placement`/`placement_remaining`, jamais parser le FR). Moyenne via
      `ranked_per_match` (`formatSignedFixed(delta_per_match, 1)` + « pt/match »). Titre section
      = `ranked_title` (« Classement »). Clés i18n neuves FR/EN : `ranked_per_match`,
      `placement`, `placement_remaining` (ICU plural). Pas de clé `ranked_progress_*` inventée
      (l'« → » est un littéral language-neutral — 0 clé morte).
- [x] **4f (frontend).** FAIT. Bloc attendu/réel + cumul brut entièrement retirés de
      `RankedCard`. Clés `ranked_delta`/`ranked_expected`/`ranked_actual`/
      `ranked_expected_vs_actual` supprimées de `explorer.toml` + manifests régénérés. Grep de
      clôture : 0 occurrence dans `features/explorer` ET `explorer.toml`.
- [x] **4g (nettoyage).** FAIT (fusionné dans 4a). `DeltaSum` ET `RatingKind` retirés du DTO
      (plus aucun consommateur après bascule front — grep). État final `ExplorerBriefingRanked`
      = `Kinds` seul. Tests MAJ.

Gate Phase 4 : PASSÉ (2026-07-16, racine du worktree). `go test ./...` = exit 0 (0 FAIL) ;
`make go-api-lint` (= `go vet ./internal/domain/... ./internal/analysis/...`) = 0 (+ `go vet
./internal/service/... ./internal/api/...` = 0 ; `golangci-lint --new-from-rev=origin/main`
service/analysis/domain = 0 issues) ; `make generate-types` idempotent (re-run → 0 diff
résiduel) ; `make check-types` = 0 ; `make test-web` = 257 fichiers / 2178 passés / 14 skipped /
0 échec ; `npm run lint` = 0 erreur (68 warnings baseline, 0 sur fichiers touchés) ; greps :
0 `delta_sum`/`ranked_expected`/`expected_win_rate` dans `features/explorer` + `explorer.toml`.

### Phase 5 — Carte contexte solo/escouade conditionnelle (lourd, backend + frontend) — item 6

- [x] **5a (domain).** FAIT. `explorer_briefing.go` : type `ExplorerBriefingContextSplit`
      (`Solo`/`Squad` de type `ExplorerBriefingContextGroup`) + `ExplorerBriefingContextGroup`
      (`Matches`/`WinRate`/`KDA`/`AvgPerf` — symétrique du socle `ExplorerBriefingScope`,
      unités ADR 0006 annotées) ; champ `ContextSplit *ExplorerBriefingContextSplit` ajouté à
      `ExplorerBriefing` (`json:"context_split,omitempty"`, commentaire « nil si non pertinent »).
- [x] **5b (service).** FAIT. `buildBriefingContextSplit(scope)` + `briefingContextGroup(rows)`
      extraits dans NOUVEAU fichier `match_history_service_briefing_context.go` (le fichier
      principal était à 481 L — extraction plutôt qu'accroître vers le seuil 500, CLAUDE.md §5).
      Partition sur `IsWithFriends`, agrégation via `aggregateRawStats` existant. Constante nommée
      `minContextSplitMatches = 10` (D-B, pas de magic number). Nil si l'un des deux sous-groupes
      `< minContextSplitMatches` (couvre aussi le scope mono-contexte : sous-groupe vide < seuil).
      Câblé dans `buildExplorerBriefing` après le module ranked, sous garde `!LowSample` (retour
      anticipé si LowSample) ; P-7 respecté : AUCUN gate capability.
- [x] **5c (tests service).** FAIT. Nouveau `match_history_service_briefing_context_test.go` :
      `_RelevantBothAboveThreshold` (les deux = 10 → bloc présent, WinRate solo 1.0 / squad 0.0,
      KDA ≈ 5.667, AvgPerf non nil) ; `_MonoContextNil` (tout solo → nil) ; `_BelowThresholdNil`
      (9 escouade < seuil → nil) ; `_ContextSplitOmittedWhenLowSample` (via `buildExplorerBriefing`,
      8 rows mixtes → LowSample true + ContextSplit nil). Helper `briefingCtxRaw` (IsWithFriends).
- [x] **5d (OpenAPI).** FAIT. `openapi.yaml` : `ExplorerBriefingContextSplit` +
      `ExplorerBriefingContextGroup` ajoutés (YAML émis exact via `OPENAPI_EMIT_OUT`), champ
      `context_split` ($ref) ajouté à `ExplorerBriefing`. `make generate-types` régénéré (15 L) ;
      `types.ts` : exports `ExplorerBriefingContextSplit`/`…ContextGroup` ajoutés ;
      `TestOpenAPISchemaDrift` vert (0 MISSING ; ExplorerBriefing réconcilié, plus divergent).
- [x] **5e (frontend).** FAIT. `ContextSplitCard`/`ContextSplitRow` ajoutés à
      `ExplorerBriefingModules`, rendus ssi `briefing.context_split != null` (early-return du
      module étendu). Libellés réutilisant `explorer.filters.context_solo`/`context_squad`
      (FR Solo/Escouade, EN Solo/Squad) ; nouveau titre `explorer.briefing.context_split_title`
      (FR « Solo vs Escouade » / EN « Solo vs Squad »), manifests régénérés (1 clé). Rendu par
      ligne : libellé · n matchs · WR (coloré `winRateColor`, tokens) · KDA. Aucun hex/Tailwind
      couleur ; aucun gate capability (P-7).
- [x] **5f (tests frontend).** FAIT. Nouveau describe dans `ExplorerBriefingStrip.test.tsx` :
      carte rendue quand `context_split` présent (titre + libellés solo/escouade) ; omise quand
      absent.

Gate Phase 5 : PASSÉ (2026-07-16, racine du worktree). `go test ./...` = exit 0 (0 FAIL) ;
`make go-api-lint` = 0 (+ `go vet ./internal/service/... ./internal/api/...` = 0 ;
`golangci-lint --new-from-rev=origin/main` service/domain = 0 issues) ; `make generate-types`
idempotent (re-run → diff stable 15 L, 0 résiduel) ; `make check-types` = 0 ; `make test-web`
= 257 fichiers / 2180 passés / 14 skipped / 0 échec (dont les 2 tests contexte neufs) ;
`npm run lint` = 0 erreur (68 warnings baseline pré-existants, 0 sur les fichiers touchés).

### Phase 5b — Cartes « Séries » et « Moments forts » (moyen, backend + frontend) — items 8, 9

- [x] **5b-a (domain).** FAIT. `explorer_briefing.go` : `ExplorerBriefingStreaks`
      (`BestWinStreak`/`WorstLossStreak int`, `json:",omitempty"` → segment à zéro omis) +
      `ExplorerBriefingDominance` (`Dominations`/`Humiliations`/`Remontadas`/`Debandades`/
      `ContreRemontadas int`, `omitempty`) ; champs `Streaks *ExplorerBriefingStreaks` /
      `Dominance *ExplorerBriefingDominance` ajoutés à `ExplorerBriefing` (`omitempty`,
      commentaire « nil si non pertinent »).
- [x] **5b-b (service).** FAIT. `buildBriefingStreaks` + `longestOutcomeRun` +
      `buildBriefingDominance` extraits dans NOUVEAU fichier
      `match_history_service_briefing_streaks.go` (le fichier principal était à 486 L —
      extraction plutôt qu'accroître vers 500, CLAUDE.md §5, cohérent avec _ranked/_context).
      Séries : rows non datées écartées, tri `StartTime` asc, série rompue par TOUT autre
      outcome (P-9), nil si aucune row datée. Dominance : `switch` sur les constantes nommées
      `analysis.DominanceFlag*` (pas de magic number), nil si tous compteurs à zéro. Câblé dans
      `buildExplorerBriefing` après le module contexte, sous garde `!LowSample` (retour anticipé
      LowSample). Tests (`match_history_service_briefing_streaks_test.go`) : `_HeadStreak`,
      `_TailStreak`, `_AllWinsNoLossStreak`, `_BrokenByAnyOutcome`, `_UndatedRowsDiscarded`,
      `_NilWhenNoDatedRows`, `_Counts`, `_NilWhenAllZero`, `_StreaksAndDominanceHeterogeneous`
      (dataset réaliste via `buildExplorerBriefing`), `_StreaksOmittedWhenLowSample`.
      RÉUTILISATION : pas de helper pur streaks existant réutilisable (les 3 usages
      `detectTilt`/`sliceBestWinStreakCanonical`/`currentStreak` sont couplés à leurs types)
      → logique locale (cf. Découverte-2).
- [x] **5b-c (OpenAPI).** FAIT. `openapi.yaml` : `ExplorerBriefingStreaks` +
      `ExplorerBriefingDominance` ajoutés (YAML émis exact via `OPENAPI_EMIT_OUT`), champs
      `streaks`/`dominance` ($ref) ajoutés à `ExplorerBriefing` (réconcilié — plus divergent).
      `make generate-types` régénéré + idempotent (hash stable) ; `types.ts` : re-exports
      `ExplorerBriefingStreaks`/`…Dominance` ajoutés ; `TestOpenAPISchemaDrift` vert (0 MISSING).
- [x] **5b-d (frontend).** FAIT. `StreaksCard` + `DominanceCard` ajoutés à
      `ExplorerBriefingModules`, rendus ssi leur bloc a du contenu (`showStreaks` = au moins un
      segment > 0 ; `showDominance` = au moins une catégorie > 0 — items 12/13 « carte omise si
      rien à afficher »). « Séries » : lignes « Meilleure série » / « Pire série » avec valeur
      `{n} V` / `{n} D` (segments à zéro omis), tokens `outcome-win`/`outcome-loss`. « Moments
      forts » : une pastille par catégorie non nulle, libellés RÉUTILISÉS de
      `narrative.dominance.*` (manifest match_view) + tokens `narrative-*` (mapping
      `DOMINANCE_ITEMS` calqué sur `ExplorerMatchesTable` — cf. Découverte-3), compteur `×N`
      language-neutral. Clés i18n neuves FR/EN : `streaks_title`, `streak_best`, `streak_worst`,
      `streak_wins` (`{n} V`/`{n} W`), `streak_losses` (`{n} D`/`{n} L`), `highlights_title` ;
      manifests régénérés. Tokens sémantiques uniquement (aucun hex/Tailwind couleur). Tests
      (`ExplorerBriefingStrip.test.tsx`) : séries présentes (2 segments), segment zéro omis,
      carte omise si 2 zéros ; dominance présente (compteurs `×3`/`×1`), carte omise si absente.

Gate Phase 5b : PASSÉ (2026-07-16, racine du worktree). `go test ./...` = exit 0 (0 FAIL) ;
`make go-api-lint` = 0 (+ `go vet ./internal/service/... ./internal/api/...` = 0 ;
`golangci-lint --new-from-rev=origin/main` service/domain = 0 issues) ; `make generate-types`
idempotent (md5 stable) ; `make check-types` = 0 ; `make test-web` = 257 fichiers / 2185 passés /
14 skipped / 0 échec (dont les 5 tests séries/dominance neufs) ; `npm run lint` = 0 erreur
(68 warnings baseline pré-existants, 0 sur les fichiers touchés).

### Phase 6 — Vérification navigateur & clôture

- [x] **6a.** FAIT (2026-07-16). Serveur buildé depuis le worktree (`cmd/server`, CGO) et lancé
      détaché avec WorkingDirectory = dépôt principal (données réelles) ; `:8000` healthz 200.
      Vite lancé depuis le worktree (5173/5174 occupés → **:5175**). Session admin JGtm réutilisée
      via cookie signé injecté (header CDP `extraHttpHeaders`, secret .env.local) — pas de re-login,
      pas de modif fichier. Explorer mode Matchs de JGtm ouvert sur **halo_infinite** (LUSR).
- [x] **6b.** FAIT — captures `02-hinfinite-fullhistory.png` (halo_infinite) + `01-explorer-initial.png`
      (halo_5). État PLEIN HISTORIQUE, halo_infinite JGtm (1015 matchs) :
      **item 1** — AUCUN delta « vs habituel » (socle Matchs/WR/FDA/Perf ni lignes de dimension),
      aucune « ±0 pts » ; dimensions ordonnées par taux de victoire décroissant (Par carte
      81/70/67/27/25/10 %, Par mode 54/53/50/47/47/46 %) — P-8 confirmé ✓.
      **item 2** — « Par sélection » (Partie rapide 981 matchs) ✓.
      **item 3** — carte « Classement » (plus aucun « Pronostic » ni bloc attendu/réel) ✓.
      **item 4** — une ligne « LUSR · Or II → Or VI · −1.4 pt/match » (paliers connus + pt/match,
      aucun cumul brut ; Or II ≠ placement → D-D ok ; mono-type = 1 seule ligne) ✓.
      **item 5** — « 22 nov. 2021 – 16 juil. 2026 » (année) ✓.
      **item 7** — en-têtes de cartes unifiés (Par carte/mode/sélection, Tendance, Classement,
      Solo vs Escouade, Séries, Moments forts) ✓.
      **item 8** — Séries « 11 V / 10 D » ; la frise montre un run rouge « x10 » corroborant la
      pire série ✓.
      **item 9** — Moments forts DOMINATION ×68 / HUMILIATION ×70 / REMONTADA ×4 / DÉBANDADE ×12,
      contre-remontada (=0) omise ✓.
      Équilibre socle sans sub (FDA/Perf) : valeurs top-alignées, tuiles de même hauteur — non
      choquant (pas de Découverte). Console navigateur : 0 erreur/warning.
- [x] **6c.** FAIT — capture `03-hinfinite-filtered-solo.png`. Filtre contexte Solo (491 matchs) :
      les deltas « vs habituel » RÉAPPARAISSENT (socle WR « −1 pts », FDA « +0.94 », Perf « ±0 » ;
      dimensions ▲/▼ « +8/+7/+6/−4/−5 pts ») ; « ±0 pts » présent sous filtre = comportement voulu
      (seul le plein historique masque, P-1/2d) ; dimensions triées par delta (WR non monotone
      58/56/64 % → tri delta, P-8 sous filtre inchangé) ; carte « Solo vs Escouade » DISPARAÎT
      (scope mono-contexte) ; Classement recalculé « LUSR · Or II → Or VI · −2.6 pt/match » ;
      Séries/Moments forts recalculés sur le scope filtré. Console : 0 erreur.
- [x] **6d.** FAIT — verdicts de dégradation :
      **titre H5** (halo_5, capture `01`) → carte « Classement » OMISE (JGtm H5 non ranked-capable
      sur ce scope), pas de crash, 0 erreur console ✓.
      **scope mono-contexte** (filtre Solo, 6c) → carte Solo/Escouade omise ✓.
      **scope mono-type de rating** (JGtm n'a que du LUSR) → une seule ligne Classement ✓.
      **scope sans palier** (progression omise) → `[~]` non reproductible en navigateur avec les
      données réelles (JGtm porte toujours un palier) — couvert par le test unitaire Phase 4c
      `_RankedNoTierLabels` (labels nil → moyenne seule).
      **scope sans aucun DominanceFlag** (carte Moments forts omise) → `[~]` idem non reproductible
      (les scopes réels ont toujours des flags) — couvert par le test `_NilWhenAllZero` ; l'omission
      par catégorie à zéro est visuellement confirmée (contre-remontada absente en 6b/6c).
      **Spot-check locale EN** (PATCH /settings lang=en, capture `04-en-locale-briefing.png`) :
      clés neuves en EN — « Ranking », « LUSR · Or II → Or VI · −2.6 pt/match » (« pt/match »
      traduit), « Streaks / Best streak 9 W / Worst streak 10 L », « Highlights »,
      dominance « DOMINATION/HUMILIATION/COMEBACK/COLLAPSE », « vs usual », « By map/mode/playlist »,
      date « Nov 22, 2021 – Jul 16, 2026 » (année). Item 2 : EN reste « By playlist » (seul le FR
      passe à « Par sélection ») — conforme. Paliers « Or II → Or VI » + modes/playlists FR restent
      en dur = limitation actée §2 (labels stockés FR en base). Console EN : 0 erreur.
      Session JGtm restaurée (halo_5, locale fr) en fin de vérif.
- [x] **6e (changelog / What's new v7.0).** FAIT. `docs/CHANGELOG.md` ET `docs/FR/CHANGELOG.md`,
      entrée `[Unreleased]` (v7.0) : bullet « Explorer — briefing V2 » dans « Added (React /
      TypeScript) » (classement par type CSR/LUSR en paliers + pt/match, cartes Séries/Moments
      forts, carte solo/escouade conditionnelle, deltas masqués en plein historique, en-têtes
      unifiés, dates avec année, « Par sélection », retrait du bloc attendu/réel mentionné en fin
      de bullet) + bullet « Explorer briefing DTO » dans « Added (Go API) » (ranked par type,
      `KPIStats.RankDeltas`, context split, streaks, dominance, retrait `expected_win_prob`).
      Parité EN/FR dans les 2 fichiers (hook docs-fr-sync). Format Keep a Changelog respecté.
- [x] **6f.** FAIT. `delivery-checklist` déroulé. Entrée thought_log finale ajoutée. Point d'étape
      utilisateur ci-dessous (revue visuelle de validation demandée avant tout merge).

Gate Phase 6 : PASSÉ (2026-07-16). Tous les critères §1 (1-14) vérifiés en navigateur (captures
`01`..`04` dans le dossier temp de session) ; console 0 erreur sur les 4 états (H5, plein
historique, filtré, EN) ; changelog EN+FR mis à jour (critère §1.14). Gates §1.8 tous verts en
une passe finale : `go test ./...` = exit 0 (0 FAIL) ; `go vet` domain/analysis/service/api = 0 ;
`make go-api-lint` = 0 ; `golangci-lint --new-from-rev=origin/main` (domain/analysis/service/api)
= 0 issues ; `make generate-types` idempotent (0 diff `generated.ts`) ; `make check-types` = 0
(cache `.tmp` purgé) ; `make test-web` = 257 fichiers / 2185 passés / 14 skipped / 0 échec ;
`npm run lint` = 0 erreur (68 warnings baseline pré-existants, 0 sur les fichiers du chantier).
NON committé (le superviseur commite ; merge `main` = deploy prod → après revue visuelle user).

**Statut final des critères §1 (vérifiés Phase 6, 2026-07-16)** — tous `[x]` sauf mention :
1. Deltas masqués en plein historique / réapparaissent sous filtre `[x]` (6b + 6c).
2. « Par sélection » FR, aucune « Par playlist » résiduelle (EN reste « By playlist ») `[x]` (6b/6d).
3. Carte « Classement », plus aucun « Pronostic » ni bloc attendu/réel `[x]` (6b).
4. Classement par type, paliers connus + pt/match, aucun cumul brut, placement D-D `[x]` (6b, LUSR).
5. Période avec année via `formatDateRange` `[x]` (6b « 22 nov. 2021 – 16 juil. 2026 »).
6. Carte solo/escouade conditionnelle (présente 6b, omise en mono-contexte 6c) `[x]`.
7. En-têtes de cartes-sections unifiés `[x]` (6b visuel).
8. Gates verts `[x]` (§1.8, passe finale Phase 6f).
9. Vérif navigateur des 2 états + captures `[x]` (6b/6c, captures `01`..`04`).
10. Dimensions triées par taux de victoire en plein historique, par delta sous filtre `[x]` (6b/6c, P-8).
11. `expected_win_prob` purgé (DTO + service + i18n) `[x]` (Phase 4 ; confirmé UI 6b : aucun bloc).
12. Carte « Séries » (meilleure/pire série sur tout le scope, segment nul omis) `[x]` (6b 11 V/10 D).
13. Carte « Moments forts » (compteurs dominance, catégories nulles omises) `[x]` (6b/6c).
14. Changelog `[Unreleased]` à jour EN+FR `[x]` (6e).
   Sous-cas de dégradation « scope sans palier » (item 4) et « scope tous-zéro dominance » (item 13)
   non reproductibles en navigateur avec les données réelles → `[~]` couverts par tests unitaires
   (`_RankedNoTierLabels`, `_NilWhenAllZero`) ; l'omission par catégorie/segment nul est confirmée
   visuellement (contre-remontada absente ; aucun « Pire série : 0 »).

---

## 6. Découvertes (à remplir en exécution — ne pas traiter hors périmètre)

- Découverte-1 (Phase 0, 2026-07-16, sans impact) : `type ExplorerBriefingRanked struct` est à
  `explorer_briefing.go:121` (le §2 citait `:120-133`) — décalage d'1 ligne, champs identiques
  (`RatingKind`/`DeltaSum`/`ExpectedWinRate`/`ActualWinRate`/`MatchesWithPrediction`). Aucune
  action : à traiter en Phase 4 sur pièces. Les commits « sweep recovery player-DB » de la base
  n'ont touché aucun fichier explorer/briefing (vérifié `git log -- <fichier>`).
- Arbitrage Révision 3 (2026-07-16) : « Séries » et « Moments forts » VALIDÉES → intégrées au
  périmètre (items 8/9, Phase 5b, P-9). « Records du scope » ÉCARTÉ (pas le bon endroit).
- Découverte-2 (Phase 5b, 2026-07-16, hors périmètre — NE PAS traiter) : le motif « plus longue
  série consécutive » (boucle max-run) existe en 3 endroits aux types/besoins distincts :
  `analysis/patterns/behavioral.go` `detectTilt` (renvoie run+start, couplé au tilt),
  `analysis/home_canonical_highlights_tiles.go` `sliceBestWinStreakCanonical` (canonical rows →
  HighlightSlide), et désormais `service/…_briefing_streaks.go` `longestOutcomeRun`
  (`MatchHistoryRawRow`). Aucun n'est un helper pur réutilisable en l'état (P-9 : « réutiliser
  si existe » → rien de réutilisable). Une centralisation `analysis.LongestRun`/`LongestOutcomeRun`
  + migration des 2 autres copies (CLAUDE.md §6, garde-rail) serait un refactor à part — noté ici,
  non traité (règle 7 : zéro fix hors périmètre).
- Découverte-3 (Phase 5b, 2026-07-16, hors périmètre) : le mapping DominanceFlag → clé i18n
  `narrative.dominance.*` + token `narrative-*` existe désormais en 2 endroits
  (`ExplorerMatchesTable` `DOMINANCE_LABEL_KEYS`/`DOMINANCE_COLOR_TOKENS` et
  `ExplorerBriefingModules` `DOMINANCE_ITEMS`). 2 copies = dans la limite CLAUDE.md §6 ; une 3ᵉ
  surface devra centraliser (helper partagé `features/explorer/dominance.ts` + garde-rail) ET
  migrer les deux copies dans le même commit (anti-pattern « factorisation abandonnée »).
- CHANTIER SÉPARÉ (validé dans son principe par l'utilisateur, à planifier À PART — ne pas
  traiter ici) : « lisibilité des extrêmes » du tableau des matchs de l'Explorer, en deux lots.
  **Lot 1 — tri par en-têtes de colonnes** (premier pas recommandé, discuté 2026-07-16) : le
  tri SERVEUR existe déjà via le `<select>` « Trier par »
  (`ExplorerPage.matchesMode.tsx:384-400`, clés `start_time` / `performance_score_relative` /
  `kda` / `kills` / `delta_mmr` / `outcome`) — le lot consiste à rendre les en-têtes
  cliquables (TanStack `manualSorting`, mapping colonne → `sortKey` existant, indicateur
  ▲/▼), sans nouveau backend ; étendre à d'autres colonnes = élargir la whitelist de clés de
  tri côté Go. **Lot 2 — surlignage MVP/LVP** (optionnel, à réévaluer après usage du lot 1) :
  extrêmes calculés côté BACKEND sur TOUT le scope filtré — jamais sur la page visible (un
  « max de la page » est trompeur et instable en paginant) — renvoyés comme `match_id`
  (+ colonne), surlignés par le front quand la row est visible. Décisions à instruire :
  colonnes, égalités, rendu (tokens), interaction avec le tri, raccourci « aller au meilleur
  match ».
  - **MISE À JOUR 2026-07-17 — Lot 1 LIVRÉ** (branche `feat/explorer-briefing-compact`,
    frontend-only, aucun changement Go). En-têtes de colonnes cliquables sur les 5 colonnes
    dont la clé de tri serveur est réellement honorée par `service.compareMatchHistoryRows`
    (`start_time`, `performance_score_relative`, `kda`, `kills`, `delta_mmr`) : TanStack en
    `manualSorting` (tri SERVEUR, jamais client — page cappée à 10000 lignes), clic = toggle
    asc/desc, `aria-sort` + indicateur ▲/▼, source de vérité UNIQUE = le même `sortKey` que
    le `<select>` (qui reste en place, avec ses options `asc` ajoutées pour rester
    synchronisé). Logique pure extraite/testée : `features/explorer/explorerMatchesSort.ts`
    (+ `.test.ts`). **Lot 2 (surlignage MVP/LVP) : toujours EN ATTENTE** (backend requis,
    non traité).
  - **DÉCOUVERTE 2026-07-17 (bug backend préexistant, hors périmètre — NON traité)** : le
    `<select>` « Trier par » expose une option « Résultat » de valeur `outcome:desc`, mais
    `service.compareMatchHistoryRows` ne connaît que le champ `outcome_code` — `outcome` tombe
    donc sur le `default` (tri par `start_time`). Le tri « Résultat » ne trie PAS par résultat
    (bug antérieur à ce lot). Pour cette raison, la colonne « Résultat » est **exclue** des
    en-têtes triables (on ne crée pas une affordance trompeuse). **Correctif = backlog** :
    soit le front envoie `outcome_code`, soit le back ajoute `case "outcome"` /
    l'entrée `outcome` à `availableSortFields` — au choix, à instruire à part. Toute extension
    de la whitelist de tri côté Go (autres colonnes triables) reste également du backlog.

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
| 3 | Carte « Classement » : renommage + suppression bloc attendu/réel | Rapide (1b) + porté par Phase 4 | i18n / front / DTO (D-A tranché) |
| 5 | Année dans les dates | Rapide | front (réutilise `formatDate`) |
| 1 | Masquer delta « vs habituel » + tri dimensions plein historique (P-8) | Moyen | front + service |
| 7 | En-têtes de cartes unifiés | Moyen | front (wrapper partagé) |
| 4 | Classement PAR TYPE (CSR/LUSR) en grades + moyenne/match | Lourd | **backend DTO + analysis** + service + OpenAPI + front (D-C/D-D) |
| 6 | Carte solo/escouade conditionnelle | Lourd | **backend DTO** + service + OpenAPI + front (seuil = D-B) |
| 8 | Carte « Séries » (meilleure/pire série du scope) | Moyen | **backend DTO** + service + OpenAPI + front (P-9) |
| 9 | Carte « Moments forts » (compteurs dominance) | Moyen | **backend DTO** + service + OpenAPI + front (P-9) |

**Dépendances backend** : items 4 et 6 (DTO `ExplorerBriefingRanked` par type + champ additif
`KPIStats.RankDeltas` + nouveau `ExplorerBriefingContextSplit`, service, régénération OpenAPI +
drift test) et item 1 (re-tri P-8, service seul, sans DTO). Items 2, 3, 5, 7 sont frontend-only.
**Dépendances inter-phases** : Phase 3 (wrapper `BriefingSectionCard`) précède les cartes des
Phases 4, 5 et 5b (elles le réutilisent). **Dépendances utilisateur** : D-B
(seuil, Phase 5), D-C (formulation, Phase 4) — chacune avec un défaut pour ne pas bloquer ;
D-A et D-D tranchées le 2026-07-16. **Aucun déploiement prod** dans ce chantier (le merge `main` = deploy auto reste
la décision de l'utilisateur, après revue visuelle 6b-6d).
</content>
