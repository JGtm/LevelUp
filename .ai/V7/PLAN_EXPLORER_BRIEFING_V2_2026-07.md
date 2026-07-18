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

- [ ] Créer `feat/explorer-briefing-v2` depuis `main` à jour (`git fetch` ; vérifier
      `git log --oneline -1 origin/main`).
- [ ] Relire §2 (constat) sur pièces : rouvrir chaque fichier:ligne cité et confirmer qu'il n'a
      pas bougé depuis le 2026-07-16 (le code a pu être modifié).
- [ ] Confirmer l'état des décisions AWAIT-USER restantes (D-B, D-C) : noter au journal la
      valeur retenue (confirmée ou défaut). D-A et D-D sont tranchées (2026-07-16).

Gate Phase 0 : `git branch --show-current` = `feat/explorer-briefing-v2` ; constat re-vérifié ;
décisions notées.

### Phase 1 — Terminologie, renommage & année (rapide, frontend-only) — items 2, 3, 5

- [ ] **1a (item 2).** `explorer.toml` `explorer.briefing.dim_playlist` → FR « Par sélection »
      (EN inchangé « By playlist »). Régénérer les manifests.
- [ ] **1b (item 3, D-A tranché).** `explorer.toml` : `explorer.briefing.ranked_title` → FR
      « Classement » / EN « Ranking ». PAS de clé tooltip. La suppression du bloc « attendu vs
      réel » (rendu + clés `ranked_expected*` + champs DTO) se fait en Phase 4 — le composant
      les consomme encore, les clés restent jusque-là pour ne pas casser le typecheck ;
      `ranked_delta` idem (Phase 4).
- [ ] **1c (item 5).** Ajouter `formatDateRange(start, end, locale, fallback?)` dans
      `lib/formatters/date.ts` (mettre à jour l'en-tête de doc du fichier), basé sur
      `Intl.DateTimeFormat.prototype.formatRange` avec `{ day:'numeric', month:'short',
      year:'numeric' }` — factorisation année/mois native (« 3 – 12 mars 2025 » ;
      « 3 mars 2024 – 12 janv. 2025 » si années différentes ; date simple si end absent ou
      égal à start). Test unitaire du helper. Puis `ExplorerBriefingStrip.tsx` `formatPeriod`
      (`:41-55`) : remplacer le `Intl.DateTimeFormat` local par ce helper.
- [ ] **1d.** Garde-rail terminologie : ajouter/étendre un test (ou grep de clôture) vérifiant
      l'absence des littéraux « Par playlist » et « Pronostic »/« Prognosis » dans le manifest et
      les composants briefing.

Gate Phase 1 : `node apps/web/scripts/build_i18n_manifests.mjs` sans diff inattendu ;
`make check-types` = 0 ; `make test-web` (dangerouslyDisableSandbox) vert ;
`cd apps/web && npm run lint` = 0 ; grep de clôture : 0 occurrence « Par playlist » /
« Pronostic » / « Prognosis » sous `apps/web/src/features/explorer` et dans `explorer.toml`.

### Phase 2 — Delta « vs habituel » dégénéré (moyen, frontend + service) — item 1

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
- [ ] **2e (P-8, service).** `buildBriefingDimensions`/`buildDimension`
      (`match_history_service_briefing.go:211-278`) : quand `len(scope) == len(all)` (plein
      historique), re-trier les groupes qualifiés par `Session.WinRate` décroissant (tie-break
      libellé) avant `selectTopFlop` ; sous filtre, comportement V1 inchangé. Ne PAS toucher
      `breakdown.CompareByKey`. Test service des deux états (plein historique → ordre par
      taux de victoire ; filtré → ordre par delta).

Gate Phase 2 : `make check-types` = 0 ; `make test-web` vert (nouveau test des deux états) ;
`npm run lint` = 0 ; `cd apps/go-api && go test ./...` = 0 (test 2e inclus) ;
`make go-api-lint` = 0.

### Phase 3 — Unification de mise en forme des cartes-sections (moyen, frontend-only) — item 7

- [ ] **3a.** Créer `BriefingSectionCard` (dans `features/explorer/`) : carte `rounded-lg border
      border-border bg-card` + en-tête bordurée `flex-none border-b border-border px-3 py-2
      text-sm font-medium` (miroir de `ChartCard` `:125-131`), avec slot titre (acceptant un
      `ReactNode` pour un `InfoTooltip` — cf. `ChartCardProps.title`) et slot contenu. Tokens
      sémantiques uniquement (aucune couleur hex/Tailwind couleur — skill `color-tokens`).
- [ ] **3b.** Migrer `DimensionCard` (`:76-92`) et `RankedCard` (`:159-204`) vers
      `BriefingSectionCard` (remplacer les titres `text-3xs uppercase …` par l'en-tête
      bordurée). Le slot titre accepte un `ReactNode` (InfoTooltip possible plus tard), mais
      aucun tooltip n'est posé dans ce chantier (D-A).
- [ ] **3c.** Vérifier visuellement la cohérence avec « Tendance » (même graisse/bordure de
      titre). Ne PAS toucher les micro-tuiles socle (P-6).
- [ ] **3d.** Garde-rail anti-divergence (CLAUDE.md §6 « ≤ 2 copies ») : le pattern d'en-tête
      bordurée existe désormais en 2 endroits canoniques (`ChartCard` + `BriefingSectionCard`) ;
      documenter en commentaire que toute 3ᵉ carte-section du briefing DOIT passer par
      `BriefingSectionCard` (les cartes des Phases 4, 5 et 5b s'y conforment).

Gate Phase 3 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 ; revue
visuelle (Phase 6) confirmera l'alignement.

### Phase 4 — Classement en grades PAR TYPE (lourd, backend + frontend) — item 4

- [ ] **4a (domain + analysis).** `explorer_briefing.go` : nouveau type
      `ExplorerBriefingRankedKind` (`Kind string`, `Matches int`, `TierStartLabel *string`,
      `TierEndLabel *string`, `TierStartIsPlacement bool`, `TierEndPlacementRemaining *int`,
      `DeltaPerMatch *float64` — tags JSON `omitempty`, commentaires d'unité) ; champ
      `Kinds []ExplorerBriefingRankedKind` sur `ExplorerBriefingRanked` (`:120-133`).
      `analysis/kpi_stats.go` : exposer les buckets par type via champ additif
      `KPIStats.RankDeltas []RankDelta` (ordre déterministe ; le `RankDelta` majoritaire
      reste, consommateurs existants intacts) + test analysis. RETRAIT (D-A) des champs
      `ExpectedWinRate`, `ActualWinRate`, `MatchesWithPrediction` du DTO. Sort de
      `DeltaSum`/`RatingKind` : item 4g.
- [ ] **4b (service).** `buildBriefingRanked` (`match_history_service_briefing.go:367-398`) :
      pour chaque entrée de `KPIStats.RankDeltas` retenue (type majoritaire toujours ; autres
      types si `Count >= minRankedKindMatches`, constante nommée §3 P-3), scanner les rows du
      scope triées par `StartTime`, restreintes à `SkillRatingType` == type (confronter la
      casse aux constantes `canonical.RatingType*` sur pièces), pour extraire le premier et le
      dernier `SkillTierLabel` non nil → `TierStartLabel`/`TierEndLabel` + flags placement D-D
      (via `PlacementDone/PlacementTotal` de la row concernée) ; `DeltaPerMatch = Value /
      Count` si `Count > 0`. Journaliser en `slog.DebugContext` si un palier attendu est
      absent (dégradation best-effort documentée, jamais d'erreur avalée — CLAUDE.md §3).
      Ne PAS recalculer μ→grade. RETRAIT (D-A) du calcul attendu/réel : boucle
      `SkillExpectedWinProb` + appel `analysis.ExpectedVsActual` supprimés ; le module est
      émis ssi au moins une entrée de type existe (sinon nil). Si `analysis.ExpectedVsActual`
      n'a plus de consommateur : le supprimer avec ses tests (« 0 code mort ») ; même
      vérification (grep) pour `MatchHistoryRawRow.SkillExpectedWinProb` et sa lecture repo Q5
      (`match_history_repo.go:222-226`) — purger si plus aucun lecteur, sinon les laisser.
- [ ] **4c (tests service).** `match_history_service_briefing_test.go` : scope mono-type (une
      entrée) ; scope MIXTE CSR+LUSR (deux entrées, paliers jamais mélangés entre types) ; type
      secondaire sous le seuil (omis) ; aucun palier (labels nil, moyenne présente) ; début en
      placement (`TierStartIsPlacement`) ; fin en placement (`TierEndPlacementRemaining`) ;
      cas `rd == nil` (module nil — plus d'émission via prédictions seules) ; MAJ/suppression
      des assertions existantes sur `DeltaSum` (`:150-162`) et sur attendu/réel selon 4g et
      D-A. Dataset hétérogène réaliste (mémoire
      `feedback_integration_tests_realistic_datasets`).
- [ ] **4d (OpenAPI).** `make generate-types` → `apps/web/src/lib/api/generated.ts` régénéré ;
      vérifier le test de drift OpenAPI (`internal/api/openapi_schema_drift_test.go`) vert.
- [ ] **4e (frontend).** `RankedCard` : remplacer le rendu `formatSignedFixed(ranked.delta_sum,
      0)` (`:178`) par une ligne PAR entrée de `kinds`, selon D-C : « {KIND} · {progression} ·
      {moyenne} » — progression « `tier_start_label` → `tier_end_label` » (égaux → palier
      seul ; absents → segment omis ; placement → clés i18n D-D) ; moyenne
      « `formatSignedFixed(delta_per_match, 1)` pt/match » (si présente). Libellé de section
      D-C (défaut « Classement »). Nouvelles clés i18n `explorer.briefing.ranked_progress_*` /
      `ranked_per_match` / `placement*` (FR/EN).
- [ ] **4f (frontend).** Retirer tout affichage résiduel du nombre cumulé brut ET tout le bloc
      « attendu vs réel » (D-A) : rendu `expected_win_rate`/`actual_win_rate`/
      `matches_with_prediction` supprimé de `RankedCard` (`:182-199`), clés i18n
      `ranked_expected` / `ranked_actual` / `ranked_expected_vs_actual` supprimées du manifest
      (+ régénération).
- [ ] **4g (nettoyage).** Retirer `DeltaSum` ET `RatingKind` du DTO et de leurs lecteurs si
      plus aucun consommateur (défaut, « 0 code mort » — remplacés par `Kinds`) OU documenter
      par commentaire pourquoi ils restent. Mettre à jour les tests.

Gate Phase 4 : `cd apps/go-api && go test ./...` = 0 ; `make go-api-lint` = 0 ; `make
generate-types` sans diff non commité résiduel ; `make check-types` = 0 ; `make test-web` vert ;
`npm run lint` = 0 ; grep : 0 rendu de `delta_sum` brut, 0 occurrence de `ranked_expected` /
`expected_win_rate` dans les composants briefing ET dans `explorer.toml`.

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

### Phase 5b — Cartes « Séries » et « Moments forts » (moyen, backend + frontend) — items 8, 9

- [ ] **5b-a (domain).** `ExplorerBriefingStreaks` (`BestWinStreak int`, `WorstLossStreak int`)
      et `ExplorerBriefingDominance` (compteurs `Dominations`, `Humiliations`, `Remontadas`,
      `Debandades`, `ContreRemontadas` int) ; champs `Streaks *…` / `Dominance *…` dans
      `ExplorerBriefing` (JSON `omitempty`, commentaire « nil si non pertinent »).
- [ ] **5b-b (service).** `buildBriefingStreaks(scope)` et `buildBriefingDominance(scope)`
      selon P-9 ; câblage dans `buildExplorerBriefing` sous garde `!LowSample`. Tests : série
      en tête et en queue de scope ; scope 100 % victoires (pire série absente) ; rows non
      datées écartées ; dominance tous-zéro → nil ; dataset hétérogène réaliste (mémoire
      `feedback_integration_tests_realistic_datasets`).
- [ ] **5b-c (OpenAPI).** `make generate-types` régénéré + drift test vert.
- [ ] **5b-d (frontend).** Deux cartes via `BriefingSectionCard` dans
      `ExplorerBriefingModules`, rendues ssi leur bloc est présent. « Séries » : « Meilleure
      série : 7 V · Pire série : 4 D » (segments à zéro omis ; clés i18n FR/EN neuves,
      réutiliser `explorer.briefing.series_win/loss` si adaptées). « Moments forts » :
      compteurs par catégorie, zéros omis, libellés dominance existants si trouvés (P-9),
      couleurs via tokens sémantiques uniquement. Tests de rendu présent/absent.

Gate Phase 5b : mêmes gates que Phase 5 (`go test ./...` = 0 ; `make go-api-lint` = 0 ;
`make generate-types` propre ; `make check-types` = 0 ; `make test-web` vert ;
`npm run lint` = 0).

### Phase 6 — Vérification navigateur & clôture

- [ ] **6a.** Dev local (`make dev`, port `:8000`) ; ouvrir l'Explorer mode Matchs d'un joueur
      réel classé (LUSR/CSR).
- [ ] **6b.** État PLEIN HISTORIQUE (aucun filtre) : vérifier item 1 (aucun delta « vs habituel »
      nulle part, aucune « = ±0 pts », entrées des dimensions ordonnées par taux de victoire —
      P-8), item 2 (« Par sélection »), item 3 (carte « Classement », plus aucun « Pronostic »
      ni bloc attendu/réel), item 4 (une ligne par type CSR/LUSR, paliers + pt/match, pas de
      « −1380 », placement rendu selon D-D), item 5 (année), item 7 (en-têtes unifiés),
      items 8/9 (cartes « Séries » et « Moments forts » présentes et cohérentes avec le
      tableau — recompter une série sur la frise pour vérifier).
      Vérifier aussi l'équilibre visuel des tuiles socle privées de leur sub (FDA/Perf) — si
      choquant, consigner en Découvertes (pas de fix hors périmètre). Capturer.
- [ ] **6c.** État FILTRÉ (narrowing, ex. une carte / un mode) : les deltas « vs habituel »
      RÉAPPARAISSENT et sont sensés ; la carte solo/escouade apparaît/disparaît selon la
      pertinence (item 6). Capturer.
- [ ] **6d.** Vérifier la dégradation : titre H5 (`ranked` absent → module rétrospectif omis, pas
      de crash), scope mono-contexte (carte solo/escouade omise), scope sans palier (progression
      omise), scope mono-type de rating (une seule ligne classement), scope sans aucun
      DominanceFlag (carte « Moments forts » omise). Spot-check locale EN
      (clés i18n neuves en EN ; paliers FR = limitation actée §2 compléments, ne pas la
      « corriger » ici). Consigner captures + verdicts au journal du plan.
- [ ] **6e (changelog / What's new v7.0).** Mettre à jour `docs/CHANGELOG.md` ET
      `docs/FR/CHANGELOG.md` (parité EN/FR dans le même commit — politique docs CLAUDE.md
      §15) : dans l'entrée `[Unreleased]` (consolidée v7.0, rendue par la page Changelog
      in-app = « What's new »), section « Added (React / TypeScript) », un bullet
      « Explorer — briefing V2 » couvrant : classement par type CSR/LUSR en paliers +
      pt/match, cartes « Séries » et « Moments forts », carte solo/escouade conditionnelle,
      deltas « vs habituel » masqués en plein historique, en-têtes de cartes unifiés, dates
      avec année, « Par sélection ». Mentionner le retrait du bloc attendu/réel (bullet
      « Removed » ou dans l'entrée). Ajouter un bullet « Added (Go API) » si le DTO briefing
      enrichi le justifie. Respecter le format Keep a Changelog (la page in-app le parse).
- [ ] **6f.** `delivery-checklist` complet ; entrée thought_log finale ; point d'étape
      utilisateur (revue visuelle de validation = merci de confirmer avant tout merge).

Gate Phase 6 : tous les critères §1 vérifiés en navigateur ; captures au journal ; changelog
EN+FR mis à jour (critère §1.14) ; gates §1.8 tous verts une dernière fois en une passe.

---

## 6. Découvertes (à remplir en exécution — ne pas traiter hors périmètre)

- (aucune découverte d'exécution pour l'instant)
- Arbitrage Révision 3 (2026-07-16) : « Séries » et « Moments forts » VALIDÉES → intégrées au
  périmètre (items 8/9, Phase 5b, P-9). « Records du scope » ÉCARTÉ (pas le bon endroit).
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
