# Audit d'impact T₀ — Chronologie des matchs Halo Infinite

> Statut : **Phase 0 du plan refonte timeline** terminée
> Date : 2026-05-28
> Plan associé : refonte `domain.MatchTimeline` (cf. ADR 0024 à venir)

---

## Contexte

`match_participants.first_joined_time` (pour joueurs `present_at_beginning=true`) fournit T₀ = début réel du gameplay. `match_registry.start_time` marque le début du film (incluant ~28s de countdown pré-match). Les chronologies sont décalées de 0–60s.

Solution : `domain.MatchTimeline` qui encapsule T₀ et expose `CorrectEventTime(ms)` / `GameplayDuration()`.

---

## 1. Consommateurs `highlight_events.time_ms` (Go)

### 1.1 Lecture et calculs temporels — TRÈS CRITIQUE

| Fichier | Ligne(s) | Usage | Sévérité |
|---------|----------|-------|----------|
| `internal/analysis/narrative/first_events.go` | 9–21, 94 | Agrégation `FirstKillMS`/`FirstDeathMS` | **TRÈS CRITIQUE** |
| `internal/analysis/narrative/intensity.go` | 37–98 | Bucketing par `maxTime`, heatmap intensité | **TRÈS CRITIQUE** |
| `internal/analysis/narrative/cadence.go` | – | Profils cadence par phase 30/60s | **CRITIQUE** |
| `internal/service/timeseries_service_events.go` | – | "Premier événement" (distribution) | **CRITIQUE** |
| `internal/analysis/match_impact.go` | 29–40, 105–200 | 6 badges narratifs (cf. §4) | **TRÈS CRITIQUE** |
| `internal/platform/duckdb/queries_match.go` | 148, 153, 302, 317, 323 | `ORDER BY he.time_ms` | **IMPORTANT** |

### 1.2 Tests

| Fichier | Adaptation requise |
|---------|---------------------|
| `internal/analysis/narrative/first_events_test.go` | Fixtures TimeMS — recalibrer |
| `internal/analysis/match_impact_test.go` (42–53) | Fixtures badges — recalibrer |
| `internal/sync/highlight_events_test.go` | Tests insertion — vérifier |
| `internal/sync/golden_test.go` | Régression temps réel |

---

## 2. Consommateurs `match_registry.duration_seconds` / `playable_duration_seconds`

### 2.1 Affichage (impact UI seulement)

| Fichier | Usage |
|---------|-------|
| `internal/domain/match_view.go:77` | `PlayableDurationSeconds *int64` dans Header JSON |
| `internal/api/handlers/match_history.go` | Colonne durée tableau historique |
| `internal/openspartan/mapper/rows.go:31-32` | Mappage depuis payload Halo |

### 2.2 Calcul — CRITIQUE

| Fichier | Ligne(s) | Note |
|---------|----------|------|
| `internal/sync/transforms.go` | 105–120 | Calcule `real_start_time = start_time + (duration - playable_duration)`. **Devient redondant avec T₀ réel** |
| `internal/analysis/narrative/intensity.go` | 71 | Infère durée match depuis `maxTime` events (cross-check possible avec T₀) |

### 2.3 Migrations

- `internal/migration/steps_shared.go:254-258` : colonne `playable_duration_seconds` + `real_start_time`
- `internal/sync/schema.go` : définition schema DuckDB

**Note** : `real_start_time` existe déjà mais offset=0ms pour tous les matchs (cf. MATCH_DURATION_RESEARCH.md). À déprécier ou repurposer.

---

## 3. Consommateurs `match_participants.time_played_seconds` (faux aujourd'hui)

### 3.1 Calculs critiques

| Fichier | Ligne(s) | Usage |
|---------|----------|-------|
| `internal/analysis/kpi_stats.go` | 56–57 | `stats.TotalPlaySeconds += *r.Self.TimePlayed` (sert aux ratios per-minute) |
| `internal/analysis/stats_canonical.go` | 52 | Conversion canonical → legacy `StatsMatchRow.TimePlayedSeconds` |
| `internal/service/stats_service.go` | – | Calcul KPM per-player |

### 3.2 Affichage

- `internal/domain/match_view.go` : tableau récapitulatif durée
- `internal/api/gen/types.gen.go` : sérialisation JSON

---

## 4. Badges narratifs (`internal/analysis/match_impact.go`)

| Badge | Lignes | Logique temporelle | Sévérité |
|-------|--------|---------------------|----------|
| `first_blood` | 105–112 | Premier kill global | **TRÈS CRITIQUE** |
| `first_group_death` | 115–127 | Première mort en équipe | **TRÈS CRITIQUE** |
| `clutch_finisher` | 129–137 | Dernier kill (gagnants) | **CRITIQUE** |
| `last_casualty` | 139–147 | Dernière mort (perdants) | **CRITIQUE** |
| `last_group_kill` | 149–162 | 1er kill équipe le plus lent | **CRITIQUE** |
| `top_gun` | 191–200 | 1er à atteindre 10+ kills | **CRITIQUE** |

Tous dépendent de `TimeMS` brut → tous décalés sans T₀.

---

## 5. `avg_life_seconds` (60 fichiers consultés)

### Formule : `avg_life = total_alive_time / (D+1)` où D = morts

Faux aujourd'hui car `total_alive_time` = durée totale match (pas gameplay) → biaisé d'autant que T₀.

### Consommateurs critiques

| Fichier | Ligne(s) | Usage |
|---------|----------|-------|
| `internal/analysis/kpi_stats.go` | 68–71 | Accumulation `totalLifeSeconds` |
| `internal/analysis/squad_breakdown.go` | – | Comparaison solo vs squad |
| `internal/analysis/home_canonical_recent.go` | – | Carte récente (home page) |
| `internal/api/gen/types.gen.go` | – | `AverageLife`, `AverageLifeMMSS` |

### Tests

- `internal/analysis/kpi_stats_test.go` (39–46, 120)
- `internal/analysis/synthesis_canonical_test.go`

---

## 6. Frontend — affichage timeline et durée

### 6.1 Charts timeline

| Fichier | Note |
|---------|------|
| `src/features/match-view/MatchKDCumulChart.tsx` | Graphe cumul kills/deaths |
| `src/features/squad/charts/timelineChart.ts` | Timeline ECharts |
| `src/features/squad/charts/squadSessionTimelineChart.ts` | Timeline session |
| `src/features/session-detail/SessionKDATimeline.tsx` | KDA timeline session |
| `src/features/ascension/RecordsTimeline.tsx` | Records/jalons |

### 6.2 Affichage durée match (mm:ss)

| Fichier | Note |
|---------|------|
| `src/features/match-view/MatchHeader.tsx` | Durée en-tête |
| `src/features/match-view/MatchHeader.utils.ts` | `formatDuration()` (agnostique) |
| `src/features/squad/v2/components/HistoryTable.tsx` | Colonne durée tableau |
| `src/features/match-view/MatchStatCards.tsx` | Carte statistiques |

### 6.3 Historique matches

| Fichier |
|---------|
| `src/features/squad/SquadMatchHistoryTable.tsx` |
| `src/features/squad/v2/components/HistoryTable.tsx` |
| `src/features/session-compare/SessionCompareMatchHistory.tsx` |
| `src/features/explorer/ExplorerMatchesTable.tsx` |
| `src/features/career/CareerTopMatchesTable.tsx` |

### 6.4 Badges narratifs (rendu)

| Fichier |
|---------|
| `src/components/feedback/NarrativeBadge.tsx` |
| `src/features/match-view/MatchNarrativeSection.tsx` |
| `src/features/match-view/MatchImpactBadgesBar.tsx` |
| `src/features/synthesis/SynthesisHighlightsSection.tsx` |

### 6.5 Distributions temporelles

- `src/features/timeseries/TimeseriesFirstEventDistribution.tsx`
- `src/features/match-view/_chartSeries.ts`

---

## 7. Synthèse par type d'impact

### TRÈS CRITIQUE (logique métier, recalcul obligatoire)
1. Badges narratifs `match_impact.go` (6 badges, lignes 105–200)
2. `FirstEventsPerMatch` agrégation
3. `IntensityProfile` bucketing/heatmap
4. `KPIStats` : TotalPlaySeconds, AvgLifeSeconds
5. `TimeseriesService` (intensity rows, first events distribution)

### IMPORTANT (contrats données)
1. `match_registry.playable_duration_seconds` colonne référence
2. `match_participants.time_played_seconds` à recalculer
3. API contracts JSON (`PlayableDurationSeconds`, `AverageLife`)

### MODÉRÉ (affichage)
1. Frontend timeline charts ECharts (passifs)
2. `formatDuration()` agnostique
3. Tableaux historique
4. `NarrativeBadge` rendu

---

## 8. Fichiers clés à lire en priorité pour le plan

- `apps/go-api/internal/analysis/match_impact.go`
- `apps/go-api/internal/analysis/narrative/first_events.go`
- `apps/go-api/internal/analysis/narrative/intensity.go`
- `apps/go-api/internal/analysis/kpi_stats.go`
- `apps/go-api/internal/sync/transforms.go`
- `apps/go-api/internal/service/match_view_builders_header.go`
- `apps/go-api/internal/platform/duckdb/queries_match.go`

---

## Stratégie de refactorisation issue de l'audit

**Phase 1** — Brancher `MatchTimeline` (T₀=0 partout) sur tous les sites listés `TRÈS CRITIQUE` + `CRITIQUE` ci-dessus, refacto sans changement de comportement. Tests existants doivent rester verts.

**Phase 2** — Migration + backfill `t0_ms` dans `match_registry`. Filet de sécurité multi-joueurs (spread max-min < 2s).

**Phase 3** — Bascule de la source : `analysis.BuildTimeline` charge `t0_ms` réel. Comportement change ici.

**Phase 4** — `time_played_seconds` individuel recalculé (indépendant, peut être avant/après/parallèle).

**Phase 5** — ADR 0024 + garde-rails linter `no_raw_time_ms`.
