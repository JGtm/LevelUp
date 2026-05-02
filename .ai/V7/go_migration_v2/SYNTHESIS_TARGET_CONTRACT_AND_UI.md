# SYNTHESIS_TARGET_CONTRACT_AND_UI.md — Composition UI et contrat cible de Synthèse

## Périmètre

- Ce document détaille le **lot 3** défini dans `UX_CAREER_SYNTHESIS_BOUNDARY.md`.
- Il décrit la future page `Synthèse` côté React et son contrat cible côté Go.
- Il ne décrit pas la migration de navigation shell globale, sauf quand elle impacte directement la page.

## Référence amont

Le cadrage fonctionnel reste `UX_CAREER_SYNTHESIS_BOUNDARY.md`.

Ce document répond à une autre question :

> quelle doit être la vraie payload de Synthèse et comment la page doit-elle se composer pour absorber l'overview, les performances marquantes et les rivalités ?

## Snapshot courant

### Frontend React

- `apps/web/src/features/synthesis/SynthesisPage.tsx` affiche aujourd'hui :
  1. deux cartes de KPIs `Solo` / `Escouade` ;
  2. un graphique bipolaire ;
  3. une table de comparaison détaillée ;
  4. une heatmap jour × heure ;
  5. un tableau `Top semaines`.
- `apps/web/src/features/synthesis/queries.ts` appelle `POST /players/{slug}/pages/synthesis` avec `period` et `filters`.

### Backend Go

- `apps/go-api/internal/api/handlers/squad.go` porte encore `GetSynthesisPage` dans `SquadHandler`.
- Le body `SynthesisPageRequest` déclare `filters`, mais ces filtres ne sont pas réellement appliqués.
- `apps/go-api/internal/service/squad_service.go` retourne encore un `Period: "all"` fixe.
- `apps/go-api/internal/domain/squad.go` héberge encore les types Synthèse dans le même fichier que la page Escouade.

### Conséquence

La page se présente comme une synthèse filtrée, mais l'implémentation actuelle reste une vue solo/escouade enrichie, sans :

1. vue d'ensemble générale ;
2. performances marquantes ;
3. rivalités ;
4. breakdowns carte / mode ;
5. écho fiable du scope réellement appliqué.

## Décision d'architecture

`Synthèse` devient une page autonome, distincte d'`Escouade` au niveau :

1. du handler ;
2. du service ;
3. des types domaine ;
4. du contrat OpenAPI.

### Extraction recommandée

- `internal/api/handlers/synthesis.go`
- `internal/service/synthesis_service.go`
- `internal/domain/synthesis.go`
- `port.SynthesisService`

Cette extraction reflète enfin la frontière produit :

- `Escouade` = relations et synergies d'équipe
- `Synthèse` = récap analytique du scope courant

## Question produit à laquelle Synthèse doit répondre

> que racontent la période locale et les filtres actifs sur le jeu du joueur ?

Chaque bloc de la page doit rester cohérent avec cette question.

## Ordre de lecture cible

### Bloc 0 — Barre de scope

Visible juste sous le header.

Contenu attendu :

1. période locale active ;
2. résumé des filtres globaux résolus ;
3. nombre de matchs dans le scope ;
4. message explicite si certaines dimensions ont été ignorées faute de données.

Pourquoi : la crédibilité de la page dépend de l'explication du scope réellement appliqué.

### Bloc 1 — Vue d'ensemble

Premier vrai bloc analytique de la page.

Contenu attendu :

1. volumes cumulés ;
2. moyennes d'efficacité ;
3. records ou pics pertinents.

Découpage recommandé :

- volumes : matchs, victoires, défaites, kills, deaths, assists, temps joué ;
- efficacité : win rate, K/D, accuracy, kills/min, avg life, performance score ;
- pics : meilleur match kills, meilleur performance score, meilleure précision, meilleure série si fiable.

### Bloc 2 — Solo vs Escouade

Le bloc actuel est conservé, mais replacé comme **une** lecture du scope, pas comme la définition de toute la page.

Contenu attendu :

1. cartes de KPIs ;
2. comparaison bipolaire ;
3. table détaillée.

### Bloc 3 — Performances marquantes

Ce bloc absorbe l'ancien contenu Carrière de `top_matches_preview`.

Contenu attendu :

1. meilleurs matchs ;
2. pires matchs ;
3. possibilité d'ouvrir la vue détaillée du match.

### Bloc 4 — Rivalités

Ce bloc absorbe l'ancien contenu Carrière de `encounters_preview` et la logique antagonistes.

Contenu attendu :

1. top rencontrés ;
2. némésis ;
3. victimes ;
4. antagonistes si la distinction produit est conservée.

### Bloc 5 — Activité et temporalité

Contenu attendu :

1. heatmap jour × heure ;
2. top semaines ;
3. éventuellement une lecture sessions plus tard, mais sans empiéter sur la page dédiée.

### Bloc 6 — Répartition carte / mode

Contenu attendu :

1. breakdown cartes ;
2. breakdown modes ;
3. lecture croisée map × mode si la volumétrie le permet.

## Composition React recommandée

### Container

- `SynthesisPage.tsx` : orchestration du scope, queries et lazy panels.

### Sous-composants cibles

- `SynthesisScopeBar.tsx`
- `SynthesisOverviewSection.tsx`
- `SynthesisSoloSquadSection.tsx`
- `SynthesisHighlightsSection.tsx`
- `SynthesisRivalriesSection.tsx`
- `SynthesisActivitySection.tsx`
- `SynthesisBreakdownsSection.tsx`

### Règles de rendu

1. chaque section a son empty state explicite ;
2. la page peut afficher des sections partielles si certaines données manquent ;
3. l'absence d'un bloc ne doit jamais invalider la compréhension du scope global ;
4. tous les sous-composants doivent recevoir le même `scope` déjà résolu.

## Contrat de requête recommandé

### Requête principale

Le body doit rester en `POST`, car la page dépend d'un contexte de filtres structuré.

```go
type SynthesisPageRequest struct {
    Period        string             `json:"period"`
    Filters       FilterContextInput `json:"filters"`
    IncludePanels []string           `json:"include_panels,omitempty"`
}
```

### Règles

1. `period` est obligatoire côté logique, même si une valeur par défaut `all` reste admise ;
2. `filters` doit être réellement appliqué côté service ;
3. `IncludePanels` permet une extension future sans alourdir systématiquement la payload de base.

## Contrat de réponse recommandé

### Réponse principale

```go
type SynthesisPageResponse struct {
    Scope             SynthesisScope            `json:"scope"`
    Overview          SynthesisOverview         `json:"overview"`
    SoloSquad         *SynthesisSoloSquadBlock  `json:"solo_squad,omitempty"`
    HighlightsPreview *SynthesisHighlightsBlock `json:"highlights_preview,omitempty"`
    RivalriesPreview  *SynthesisRivalriesBlock  `json:"rivalries_preview,omitempty"`
    Activity          *SynthesisActivityBlock   `json:"activity,omitempty"`
    BreakdownsPreview *SynthesisBreakdownsBlock `json:"breakdowns_preview,omitempty"`
    LazyPanels        []string                  `json:"lazy_panels"`
}
```

### Scope

```go
type SynthesisScope struct {
    Period            string   `json:"period"`
    TotalMatches      int      `json:"total_matches"`
    AppliedFilters    []string `json:"applied_filters"`
    IgnoredFilters    []string `json:"ignored_filters,omitempty"`
    ScopeDescription  string   `json:"scope_description"`
}
```

Le `scope` est obligatoire. Il protège la page contre le flou actuel entre filtres demandés et filtres réellement honorés.

### Overview

```go
type SynthesisOverview struct {
    Totals      SynthesisTotals      `json:"totals"`
    Averages    SynthesisAverages    `json:"averages"`
    Peaks       SynthesisPeaks       `json:"peaks"`
}
```

### Solo vs Escouade

Réutiliser un maximum des structures actuelles :

1. `SynthesisKPIs`
2. `ComparisonMetricItem`
3. `TemporalHeatmapCell`
4. `TopWeekEntry`

Mais les regrouper dans un bloc nommé :

```go
type SynthesisSoloSquadBlock struct {
    SoloKPIs          SynthesisKPIs          `json:"solo_kpis"`
    SquadKPIs         SynthesisKPIs          `json:"squad_kpis"`
    ComparisonMetrics []ComparisonMetricItem `json:"comparison_metrics"`
}
```

### Highlights preview

```go
type SynthesisHighlightsBlock struct {
    BestMatches  []TopMatchDTO `json:"best_matches"`
    WorstMatches []TopMatchDTO `json:"worst_matches"`
    HasMore      bool          `json:"has_more"`
}
```

### Rivalries preview

```go
type SynthesisRivalriesBlock struct {
    MostEncountered []EncounterDTO `json:"most_encountered"`
    Nemeses         []EncounterDTO `json:"nemeses"`
    Victims         []EncounterDTO `json:"victims"`
    HasMore         bool           `json:"has_more"`
}
```

### Activity

```go
type SynthesisActivityBlock struct {
    HeatmapData []TemporalHeatmapCell `json:"heatmap_data"`
    TopWeeks    []TopWeekEntry        `json:"top_weeks"`
}
```

### Breakdowns preview

```go
type SynthesisBreakdownsBlock struct {
    ByMap   []BreakdownRow `json:"by_map"`
    ByMode  []BreakdownRow `json:"by_mode"`
    HasMore bool           `json:"has_more"`
}
```

## Lazy endpoints recommandés

Pour éviter une page monolithique, la payload principale doit rester un assemblage de previews.

### Endpoints recommandés

1. `POST /players/{slug}/pages/synthesis` → vue principale + previews
2. `POST /players/{slug}/pages/synthesis/highlights` → liste complète des meilleurs / pires matchs
3. `POST /players/{slug}/pages/synthesis/rivalries` → tables complètes rivalités / némésis / victimes
4. `POST /players/{slug}/pages/synthesis/breakdowns` → breakdowns détaillés carte / mode

### Règle de cohérence

Tous ces endpoints reçoivent **exactement** le même couple :

1. `period`
2. `filters`

et renvoient un `scope` identique dans la réponse.

## Migration des endpoints existants

### À déprécier

1. `GET /pages/career/top-matches`
2. `GET /pages/career/encounters`

### Destination cible

1. les top/pires matchs deviennent `synthesis/highlights`
2. les rencontres/rivalités deviennent `synthesis/rivalries`

### Règle de transition

Pendant la migration, les anciens endpoints peuvent survivre comme alias techniques, mais ils ne doivent plus être considérés comme la source métier canonique.

## Hooks React recommandés

### Hook principal

- `useSynthesisPage(playerSlug, request, scopeHash)`

### Hooks lazy

- `useSynthesisHighlights(playerSlug, request, scopeHash)`
- `useSynthesisRivalries(playerSlug, request, scopeHash)`
- `useSynthesisBreakdowns(playerSlug, request, scopeHash)`

### Query keys recommandées

- `synthesis(playerSlug, scopeHash)`
- `synthesisHighlights(playerSlug, scopeHash)`
- `synthesisRivalries(playerSlug, scopeHash)`
- `synthesisBreakdowns(playerSlug, scopeHash)`

Le `scopeHash` doit intégrer la période **et** le contexte de filtres, contrairement à la clé actuelle qui ne segmente que par `period`.

## Fixes préalables obligatoires

Avant d'enrichir réellement la page, trois corrections sont bloquantes :

1. le handler doit arrêter d'ignorer le body `filters` ;
2. le service doit arrêter de renvoyer `Period: "all"` en dur ;
3. la logique Synthèse doit sortir de `SquadHandler` / `SquadService`.

Sans ces corrections, toute extension de contrat resterait sémantiquement bancale.

## Critères d'acceptation

1. La page renvoie et affiche le scope réellement appliqué.
2. La page comprend une `Overview` en tête, avant le bloc solo / escouade.
3. Les meilleures / pires performances viennent de Synthèse et non plus de Carrière.
4. Les rivalités viennent de Synthèse et non plus de Carrière.
5. Les breakdowns carte / mode sont dans Synthèse.
6. Les lazy endpoints reprennent le même scope et le ré-affichent.
7. Le contrat OpenAPI reflète des types Synthèse distincts des types Escouade.

## Décision opérationnelle

La future Synthèse ne doit plus être pensée comme `une annexe d'Escouade avec heatmap`, mais comme une **page analytique principale du scope courant**.

Sa forme cible est donc :

- barre de scope
- overview
- solo vs escouade
- performances marquantes
- rivalités
- activité
- breakdowns

Et sa cible technique est :

- handler dédié
- service dédié
- types dédiés
- payload principale légère + panneaux lazy.
