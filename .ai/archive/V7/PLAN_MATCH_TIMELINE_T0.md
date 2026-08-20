# Plan — Match Timeline T₀ et data quality temporelle

> Date : 2026-05-28
> Branche cible : `feat/match-timeline-t0` (depuis `feat/lusr-v2-phase0-metrics`)
> ADR à créer : `docs/adr/0024-match-timeline-t0.md`
> Documents liés :
> - `.ai/MATCH_DURATION_RESEARCH.md` — recherche historique sur T₀
> - `.ai/AUDIT_T0_IMPACT.md` — cartographie complète des sites impactés

---

## 1. Contexte

### Découverte (2026-05-27)

`match_participants.first_joined_time` pour joueurs `present_at_beginning = true` fournit T₀ = début réel du gameplay. `match_registry.start_time` marque le début du film (incluant ~28s de countdown pré-match).

Sur le match de référence Fortress `41b61fb9-3d71-40b7-bde7-45682fba6d57` :
- `start_time` = 21:58:18 (début film)
- `MIN(first_joined_time)` = 21:58:46 (début gameplay)
- T₀ = **28 secondes**
- `duration_seconds` = 447s → gameplay réel = 419s

### Statistiques sur 1724 matchs

- 100% ont `first_joined_time` populé (backfill antérieur via `cmd/backfill_participation_info/` ou similaire)
- 580 matchs (33.6%) donnent T₀ valide (0–120s, médiane 29s)
- 1144 matchs présentent une anomalie ≈ +1h ou +2h → problème de cast `start_time::TIMESTAMPTZ` dans une session UTC±X. Fix : utiliser le pattern canonique `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`
- 88 matchs avec T₀ ≈ 0–4s (Ranked Arena sans countdown ?) — à conserver tels quels

### Anomalies découvertes durant l'audit

1. **`real_start_time`** existe dans `match_registry` mais offset = 0ms pour tous les matchs (calculée via `start_time + (duration − playable_duration)`, formule cassée pour Ranked Arena où `playable_duration = duration`). À **repurposer** pour notre T₀.
2. **`playable_duration_seconds`** non fiable pour Ranked Arena. Notre `first_joined_time` est strictement meilleur.
3. **Aberration `AvgLifeSeconds`** : `internal/analysis/squad_breakdown.go:380-382` et `:542-544` écrasent la valeur API correcte avec `sumTimePlayed/nTime` (= temps moyen par match, pas moyenne de vie). Fix indépendant à inclure dans Phase 4.
4. **`time_played_seconds`** identique pour tous les joueurs d'un match (= `duration_seconds`), même pour quitters/latecomers. Doit être recalculé depuis `first_joined_time` / `last_leave_time`.

---

## 2. Décisions tranchées

| # | Décision | Justification |
|---|----------|----------------|
| D1 | Option hybride : colonne stockée + couche d'abstraction `domain.MatchTimeline` | Évite le DRY violation sur N requêtes SQL ET les JOINs répétés à chaque service |
| D2 | T₀ NULL → fallback comportement actuel (T₀ = 0) | Compatibilité totale, pas de panic sur dataset partiel |
| D3 | Backfill + recalcul au sync, avec filet multi-joueurs | Garantit la qualité, log les anomalies |
| D4 | Stratégie "strangler fig" : abstraction en Phase 1 sans changement de valeurs, puis bascule source en Phase 3 | Permet de valider chaque étape isolément |
| D5 | Branche `feat/match-timeline-t0` depuis `feat/lusr-v2-phase0-metrics` | Continuité travail courant |
| D6 | Phase 4 (`time_played_seconds`) en parallèle de Phase 2 | `AvgLifeSeconds` agrégé dépend de `time_played` correct (même si valeur API directe inchangée par T₀) |
| D7 | `MatchTimeline` Halo-only au départ, capability-ifiable plus tard | YAGNI ; structure prête à devenir canonique |
| D8 | Remplacement silencieux côté frontend ("7m27s" → "6m59s") | La durée gameplay est la "vraie" durée du point de vue joueur |
| D9 | Repurpose `real_start_time` plutôt qu'ajouter `t0_ms` | Évite une migration de colonne ; sémantique compatible |

---

## 3. Vue d'ensemble des phases

| Phase | Objectif | Effort | Changement comportement ? |
|-------|----------|--------|----------------------------|
| 0 | Audit d'impact | ✅ Fait | Non |
| 1 | Couche d'abstraction `MatchTimeline` (T₀=0 partout) | Moyen | **Non** — refacto pur |
| 2 | Migration `real_start_time` repurpose + backfill T₀ + sync engine | Moyen-Lourd | Indirect (DB seulement) |
| 3 | Activation T₀ (bascule source `BuildTimeline`) | Léger | **Oui** — timelines décalées |
| 4 | `time_played_seconds` recalculé + fix aberration `AvgLifeSeconds` | Moyen | **Oui** — KPIs corrigés |
| 5 | ADR, garde-rails linter, cleanup `playable_duration_seconds` | Léger | Non |

**Phases livrables indépendamment** : 1, 2, 4. Phases 3 et 5 dépendent de Phase 2.

---

## 4. Phase 1 — Couche d'abstraction (refacto pur)

### 4.1 Objectif

Introduire `domain.MatchTimeline` et l'utiliser partout où on lit `time_ms` ou `duration_seconds` aujourd'hui, **sans modifier ce qui sort**. T₀ = 0 partout au début. Tous les tests existants doivent rester verts. Aucun snapshot API ne doit bouger.

### 4.2 Architecture (couches Go)

| Couche | Ajout |
|--------|-------|
| `internal/domain/match_timeline.go` | Struct `MatchTimeline{DurationMs int64, T0Ms int64}`. Méthodes pures : `GameplayDurationMs() int64`, `CorrectEventTime(rawMs int64) int64`, `RawTimeFromCorrected(correctedMs int64) int64`, `IsValid() bool`. Constructeur `NewMatchTimeline(durationMs, t0Ms int64)` (panic si negative). |
| `internal/analysis/timeline/build.go` | Fonction pure `BuildFromRegistry(reg domain.MatchRegistryRow) MatchTimeline` (T₀ NULL → 0). |
| `internal/analysis/timeline/build_test.go` | Tests purs sur tous les cas (T₀=0, T₀>0, NULL, valeurs hors-bornes). |
| `internal/port/` | Si nécessaire : enrichir `MatchRepository.LoadRegistry()` pour exposer `T0Ms` (optionnel, mis à 0 en Phase 1). |
| `internal/service/` (callsites) | Remplacement progressif des accès directs par `MatchTimeline.CorrectEventTime(...)`. Voir §4.3 pour la liste. |

**Multi-titres** : `MatchTimeline` dans `internal/domain/` (concept partagé, pas spécifique). Construction Halo-spécifique reste dans `internal/games/halo_infinite/` quand on basculera la source (Phase 3).

### 4.3 Fichiers à refactoriser (issus de l'audit)

#### Backend Go — sites critiques

| Fichier | Action |
|---------|--------|
| `internal/analysis/match_impact.go` (105–200) | Injecter `MatchTimeline` dans les 6 badges narratifs. Remplacer chaque accès `ev.TimeMS` par `tl.CorrectEventTime(ev.TimeMS)`. |
| `internal/analysis/narrative/first_events.go` (9–94) | Pareil pour `FirstKillMS` / `FirstDeathMS`. |
| `internal/analysis/narrative/intensity.go` (37–98) | Pareil pour bucketing par `maxTime`. Remplacer `maxTime` par `tl.GameplayDurationMs()` quand approprié (note : ici `maxTime` était inféré depuis events ; en Phase 1, garder le calcul pour rester ISO mais l'envelopper). |
| `internal/analysis/narrative/cadence.go` | Pareil. |
| `internal/service/timeseries_service_events.go` | Pareil. |
| `internal/platform/duckdb/queries_match.go` (148, 153, 302, 317, 323) | `ORDER BY he.time_ms` reste tel quel (l'ordre relatif ne change pas avec un offset constant). Pas de modification. |

#### Backend Go — durée match dans payloads JSON

| Fichier | Action |
|---------|--------|
| `internal/domain/match_view.go:77` | Ajouter optionnellement `GameplayDurationSeconds *int64` à côté de `PlayableDurationSeconds`. En Phase 1, valeur = `PlayableDurationSeconds` (T₀=0). |
| `internal/service/match_view_builders_header.go` | Construire `MatchTimeline` au montage du header. |

### 4.4 Stratégie de tests snapshot — MINUTIEUSE

L'objectif : prouver que **rien ne change côté API en Phase 1**.

#### 4.4.1 Dataset de référence

Sélectionner 8 matchs couvrant la diversité :

| Match | Profil | Pourquoi |
|-------|--------|----------|
| `41b61fb9-3d71-40b7-bde7-45682fba6d57` | Fortress, 4v4 + quitter + latecomer + bot, T₀ valide 28s | Cas canonique |
| À sélectionner via SQL | Ranked Arena, `playable_duration = duration` | Cas T₀ probable 0 |
| À sélectionner | BTB 8v8, T₀ ≈ 60s | Cas T₀ long |
| À sélectionner | Firefight PvE | Couvre les services PvE |
| À sélectionner | Custom Slayer | Mode hors playlist |
| À sélectionner | Match avec start_time bug (T₀ apparent +7200s) | Validera Phase 2 |
| À sélectionner | Match sans `first_joined_time` populé | NULL T₀ (si trouvé) |
| À sélectionner | Match très court (1–2 minutes) | Edge case durée |

Un script `cmd/select_t0_fixtures/main.go` génère cette liste dans `apps/go-api/internal/testdata/t0_fixtures/match_ids.json`.

#### 4.4.2 Snapshots PRE-Phase 1 (baseline)

Sur la branche **avant** Phase 1, capturer pour chaque match :

| Endpoint | Champs à capturer |
|----------|-------------------|
| `GET /api/match/{id}/view` | Full JSON (header + scoreboard + teams) |
| `GET /api/match/{id}/narrative` | Badges + first_kill_ms + first_death_ms |
| `GET /api/match/{id}/timeseries` | Intensity rows + first_events distribution |
| `GET /api/match/{id}/scoreboard` | Per-player kills/deaths/time_played/avg_life |
| `GET /api/squad/{ids}/timeline` | KD cumulé timeline |
| `GET /api/player/{gt}/kpi-stats` | KPM, AvgLife, TotalPlay, AvgMatchSeconds |
| `GET /api/player/{gt}/squad-breakdown` | Solo/squad KPIs (incl. AvgLifeSeconds → contient le bug) |
| `GET /api/match/{id}/cumul-kd` | Série K-D pour `MatchKDCumulChart` |

Storage : `apps/go-api/internal/testdata/t0_fixtures/pre_phase1/{endpoint}/{match_id}.json`.

#### 4.4.3 Mécanisme

**Approche A — `golden_test.go` sur service layer (préféré)** :
- Test pilotable par un flag `-update` qui régénère les snapshots
- Tests Go classiques (rapides, déterministes)
- Mock du repo avec données figées de la DB (export SQL en seed)

**Approche B — HTTP end-to-end (complément)** :
- Démarre le serveur en mode test (DuckDB read-only sur snapshot DB)
- Tape les endpoints avec `httptest`
- Compare réponses brutes

**Décision** : commencer par A (golden_test.go) car plus rapide. Ajouter B en Phase 1.B si on veut une couverture renforcée.

#### 4.4.4 Snapshots POST-Phase 1

Après refacto Phase 1 (et avant Phase 2 / 3) :
- Rejouer les mêmes appels sur la même seed DB
- Diff strict avec `pre_phase1/`
- **Aucun diff toléré** sauf champ nouveau `GameplayDurationSeconds` (= valeur identique à `PlayableDurationSeconds` en Phase 1, car T₀=0)
- Si un diff inattendu apparaît : bug de refacto, à corriger

#### 4.4.5 Tests unitaires nouveaux

| Fichier | Cas |
|---------|-----|
| `internal/domain/match_timeline_test.go` | T₀=0, T₀>0, T₀ négatif (panic ou rejet), CorrectEventTime/RawTime aller-retour |
| `internal/analysis/timeline/build_test.go` | NULL T₀, bornes, registry mal formé |
| `internal/analysis/match_impact_timeline_test.go` | Badges narratifs avec T₀=0 vs T₀=30s sur fixtures synthétiques |

### 4.5 Logging

```go
slog.DebugContext(ctx, "build_match_timeline",
    "match_id", reg.MatchID,
    "duration_ms", tl.DurationMs,
    "t0_ms", tl.T0Ms,
)
```

### 4.6 Critères "done" Phase 1

- [ ] `MatchTimeline` implémenté + tests unitaires verts
- [ ] Tous les sites critiques de l'audit refactorisés
- [ ] Snapshots PRE-Phase 1 capturés (8 matchs × 8 endpoints = 64 fichiers JSON)
- [ ] Snapshots POST-Phase 1 capturés ET identiques à PRE (sauf `GameplayDurationSeconds` ajouté)
- [ ] `go test ./...` vert, `go vet` propre
- [ ] Entrée `thought_log.md` avec date 2026-MM-DD, statut Complété, décisions

---

## 5. Phase 2 — Persistance T₀ (repurpose `real_start_time` + backfill + sync)

### 5.1 Objectif

Calculer T₀ et le stocker dans `match_registry`. Ré-utiliser la colonne `real_start_time` existante (au lieu d'ajouter `t0_ms`) car elle est inutile dans son état actuel (offset 0ms pour tous les matchs).

### 5.2 Architecture (couches Go)

| Couche | Ajout / Modification |
|--------|----------------------|
| `internal/migration/steps_shared_repurpose_real_start_time.go` | Migration : commentaire de colonne mis à jour, ajout d'une colonne `t0_quality VARCHAR` (NULL/OK/SPREAD_HIGH/SUSPICIOUS) à côté pour traçabilité. |
| `internal/analysis/timeline/compute_t0.go` | Fonction pure `ComputeT0(participations []ParticipationTimestamps, startTimeUTC time.Time) (t0Ms int64, quality T0Quality, err error)`. |
| `internal/games/halo_infinite/timeline_mapper.go` | Mapper : extrait timestamps depuis API/DB → input pour `ComputeT0`. |
| `internal/sync/engine.go` (post-sync hook) | Appelle `ComputeT0` et persiste via `BatchBuilder` (cf. ADR 0019). |
| `apps/go-api/cmd/backfill_t0/main.go` | CLI one-shot pour les 1724 matchs existants. Logue distribution qualité. |

### 5.3 Algorithme T₀ avec filet multi-joueurs

```
inputs = first_joined_time[] des joueurs (present_at_beginning=true AND gamertag != NULL) // exclure bots
start_utc = COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')  // canonical timezone pattern

si len(inputs) == 0 :
    return NULL, NO_DATA
si len(inputs) == 1 :
    quality = SINGLE_SOURCE  (warning loggé)
    t0_candidate = inputs[0]
sinon :
    spread_ms = MAX(inputs) - MIN(inputs)
    si spread_ms > 2000 :
        quality = SPREAD_HIGH  (warning + log valeurs)
        t0_candidate = median(inputs)
    sinon :
        quality = OK
        t0_candidate = MIN(inputs)

t0_ms = (t0_candidate - start_utc).Milliseconds()

si t0_ms < 0 :
    return NULL, NEGATIVE  (rejet, fallback Phase 1)
si t0_ms > 120_000 :
    return NULL, SUSPICIOUS_HIGH  (rejet)

return t0_ms, quality
```

### 5.4 Backfill CLI

```
cmd/backfill_t0/main.go --dry-run  (default, log distribution)
cmd/backfill_t0/main.go --commit   (écrit en DB)
cmd/backfill_t0/main.go --since 2025-08-01  (filtrage temporel)
```

Logging structuré :
- `slog.InfoContext("t0_distribution", "ok", 1500, "spread_high", 50, "single_source", 30, ...)`
- `slog.WarnContext("t0_spread_high", "match_id", id, "spread_ms", 3400, "values", [...])`

### 5.5 Tests

| Fichier | Cas |
|---------|-----|
| `compute_t0_test.go` | OK / SPREAD_HIGH / NEGATIVE / SUSPICIOUS_HIGH / SINGLE_SOURCE / NO_DATA |
| Intégration backfill | DuckDB `:memory:` + 10 matchs synthétiques (incl. timezone bug) |
| Sync engine | Test que T₀ est bien persisté lors d'un sync de nouveau match |

### 5.6 Critères "done" Phase 2

- [ ] Migration appliquée (commentaire colonne + ajout `t0_quality`)
- [ ] CLI backfill exécuté en dry-run, distribution loggée
- [ ] CLI backfill exécuté en commit, > 90% matchs avec T₀ non-NULL valide
- [ ] Sync engine calcule T₀ à chaque nouveau match (test intégration)
- [ ] Pattern canonical timezone utilisé partout (`COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`)
- [ ] Entrée `thought_log.md`

---

## 6. Phase 3 — Activation T₀

### 6.1 Objectif

Faire en sorte que `BuildFromRegistry` charge la vraie valeur de `real_start_time` (interprétée comme T₀ offset) au lieu de retourner T₀=0.

### 6.2 Changements

| Couche | Modification |
|--------|--------------|
| `internal/port/match_repository.go` | `LoadRegistry` expose `T0Ms int64` (déjà préparé en Phase 1, valeur réelle ici) |
| `internal/platform/duckdb/match_view_repo.go` | Lit `real_start_time` repurposed (delta milliseconds depuis start_time_utc) |
| `internal/analysis/timeline/build.go` | `BuildFromRegistry` retourne `MatchTimeline{T0Ms: reg.T0Ms}` au lieu de `T0Ms: 0` |
| `internal/service/match_view_builders_header.go` | `GameplayDurationSeconds = (DurationMs - T0Ms) / 1000` |

### 6.3 Tests

- Tests Phase 1 (golden_test) rejoués → **diffs attendus** sur :
  - `narrative.first_kill_ms` décalés de −T₀
  - `timeseries.intensity` bucketing recalibré
  - `match_view.GameplayDurationSeconds` strictement inférieur à `PlayableDurationSeconds`
- Nouveau snapshot `post_phase3/` pour archive du comportement final
- Tests d'intégration sur match Fortress 41b61fb9 :
  - `T0Ms == 28000` (±1000ms)
  - `GameplayDurationSeconds == 419` (±1s)
  - Premier kill recalé de ~28s

### 6.4 Critères "done" Phase 3

- [ ] Match Fortress : T₀=28s, gameplay=419s
- [ ] Diff snapshot Phase 1 → Phase 3 cohérent avec valeurs T₀ par match
- [ ] Pages timeline frontend testées manuellement (audit visuel sur 3 pages : MatchView, SquadV2 timeline, Synthesis narratives)
- [ ] Entrée `thought_log.md`

---

## 7. Phase 4 — `time_played_seconds` individuel + fix aberration `AvgLifeSeconds`

### 7.1 Objectif

1. Recalculer `time_played_seconds` réel par joueur (gérant quitters, latecomers).
2. Supprimer l'aberration `kpis.AvgLifeSeconds = sumTimePlayed/nTime` dans `squad_breakdown.go`.

**Indépendant de Phases 1–3** — peut être livré en parallèle de Phase 2.

### 7.2 Algorithme `time_played_seconds`

```
gameplay_start = start_time_utc + (t0_ms / 1000)  (si T₀ disponible, sinon start_time_utc)
gameplay_end   = end_time_utc                      (= start_time_utc + duration_seconds)
joined         = MAX(first_joined_time, gameplay_start)
left           = COALESCE(last_leave_time, gameplay_end)
left_clamped   = MIN(left, gameplay_end)
time_played    = MAX(0, (left_clamped - joined).Seconds())
```

### 7.3 Garde-rail cross-check

Pour chaque match, vérifier :
- `SUM(time_played) / COUNT(joueurs full_match)` ≈ `gameplay_duration` (±5%)
- Si écart > 5% : log warning + match_id

### 7.4 Fix aberration `AvgLifeSeconds`

**`internal/analysis/squad_breakdown.go:380-388`** :

```go
// AVANT (faux)
if nTime > 0 {
    avgLife := math.Round(sumTimePlayed/float64(nTime)*10) / 10
    kpis.AvgLifeSeconds = &avgLife
    ...
}

// APRÈS (correct)
if nLife > 0 {  // nouveau compteur dédié, séparé de nTime
    avgLife := math.Round(sumLifeSeconds/float64(nLife)*10) / 10
    kpis.AvgLifeSeconds = &avgLife
}
if nTime > 0 {
    totalMinutes := sumTimePlayed / 60.0
    if totalMinutes > 0 {
        kpm := math.Round(sumKills/totalMinutes*100) / 100
        kpis.KillsPerMin = &kpm
    }
}
```

Idem ligne 542–544.

### 7.5 Architecture

| Couche | Ajout / Modification |
|--------|----------------------|
| `internal/analysis/timeline/compute_time_played.go` | Fonction pure `ComputeTimePlayed(participation, registry, t0_ms) (int64, Quality)` |
| `internal/games/halo_infinite/time_played_mapper.go` | Mapper participation → calcul |
| `internal/sync/engine.go` | Recalcul à chaque sync |
| `cmd/backfill_time_played/main.go` | CLI one-shot |
| `internal/analysis/squad_breakdown.go` | Fix aberration AvgLifeSeconds (2 sites) |
| `internal/analysis/squad_breakdown_test.go` | Test régression : `AvgLifeSeconds` doit refléter la moyenne API, pas `sumTimePlayed/n` |

### 7.6 Critères "done" Phase 4

- [ ] `time_played_seconds` recalculé pour les 1724 matchs
- [ ] Garde-rail cross-check passe sur > 95% des matchs
- [ ] Quitters / latecomers correctement reflétés (test : Vermitrax sur Fortress = 3m40s)
- [ ] Bug `AvgLifeSeconds` fixé (test régression vérifie que la valeur diffère de l'ancienne formule)
- [ ] Snapshots `squad_breakdown` mis à jour (diffs attendus AvgLife)
- [ ] Entrée `thought_log.md`

---

## 8. Phase 5 — ADR, garde-rails, cleanup

### 8.1 ADR

`docs/adr/0024-match-timeline-t0.md` :
- Raisons de l'approche hybride (colonne + abstraction)
- Pourquoi repurpose `real_start_time` plutôt que `t0_ms`
- Stratégie strangler fig en 3 étapes
- Lien vers `.ai/MATCH_DURATION_RESEARCH.md`

### 8.2 Garde-rails linter

`internal/testing/no_raw_time_ms_test.go` :
- Scan AST de tous les fichiers `internal/`
- Détecte les accès `event.TimeMs` ou `event.TimeMS` directs en dehors de :
  - `internal/domain/match_timeline.go`
  - `internal/analysis/timeline/`
  - `internal/platform/duckdb/` (SQL)
  - Tests
- Fail si pattern trouvé

Idem pour `duration_seconds` brut hors couche timeline.

### 8.3 Cleanup `playable_duration_seconds`

À évaluer : si elle n'est plus consommée qu'au mapper depuis l'API, on peut la garder comme metadata brute mais ne plus l'exposer aux services. Décision à prendre en fin de Phase 4.

### 8.4 Documentation

- Update `.ai/MATCH_DURATION_RESEARCH.md` : statut "T₀ résolu via `first_joined_time`" + lien vers ADR 0024
- Mémoire : ajouter reference `.ai/PLAN_MATCH_TIMELINE_T0.md`

### 8.5 Critères "done" Phase 5

- [ ] ADR 0024 mergée
- [ ] Linter `no_raw_time_ms` actif
- [ ] `.ai/MATCH_DURATION_RESEARCH.md` mis à jour
- [ ] Entrée `thought_log.md`

---

## 9. Risques et mitigations

| Risque | Probabilité | Impact | Mitigation |
|--------|-------------|--------|------------|
| Snapshots Phase 1 instables (timestamps, ordre de map) | Moyenne | Faux positifs sur diff | Normaliser/trier dans snapshot writer ; ignorer champs `_at` volatils |
| Bug timezone sur les 1144 matchs non corrigé | Élevée | T₀ rejeté pour 66% des matchs en Phase 3 | Phase 2 utilise pattern canonical `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')` |
| Régression visuelle silencieuse en Phase 3 | Moyenne | Graphe décalé sans détection | Snapshots Phase 1 + revue manuelle de 3 pages clés avant merge Phase 3 |
| 88 matchs Ranked Arena avec T₀=0 légitime | Faible | Heatmap intensité plate sur ces matchs | Comportement actuel préservé (fallback T₀=0) |
| Aberration `AvgLifeSeconds` masque un autre bug | Faible | Métriques squad faussées rétroactivement | Fix Phase 4 + test régression dédié sur valeurs avant/après |
| Conflit branche `feat/lusr-v2-phase0-metrics` | Faible | Rebase ennuyeux | Branche `feat/match-timeline-t0` parent. Si LUSR merge avant : rebase propre |

---

## 10. Ordre de livraison recommandé

```
Phase 1 (refacto pur, tests verts, snapshots stables)
  ├── PR #1 : MatchTimeline + tests unitaires + golden_test infra
  ├── PR #2 : Refacto callsites narrative (match_impact, first_events, intensity, cadence)
  ├── PR #3 : Refacto callsites service (timeseries, match_view_builders)
  └── PR #4 : Snapshots Phase 1 figés en testdata

Phase 2 (DB + sync)
  ├── PR #5 : Migration + ComputeT0 + tests
  └── PR #6 : CLI backfill_t0 + run en commit

Phase 4 (parallèle Phase 2)
  ├── PR #7 : ComputeTimePlayed + backfill + fix AvgLifeSeconds aberration
  └── PR #8 : Snapshots squad_breakdown mis à jour

Phase 3 (activation)
  └── PR #9 : Bascule source BuildFromRegistry + snapshots post-T₀

Phase 5 (cleanup)
  ├── PR #10 : ADR 0024 + linter no_raw_time_ms
  └── PR #11 : Cleanup playable_duration_seconds si justifié
```

---

## 11. Avant de démarrer Phase 1

- [ ] Créer branche `feat/match-timeline-t0` depuis `feat/lusr-v2-phase0-metrics`
- [ ] Confirmer que `start_time_utc` existe dans `match_registry` (sinon ajouter en Phase 2)
- [ ] Sélectionner les 8 matchs de référence pour les fixtures
- [ ] Créer `apps/go-api/internal/testdata/t0_fixtures/` (gitignored sauf manifest)
