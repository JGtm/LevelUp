# Plan de portage de la page Coéquipiers — Python v7/cockpit -> Go + React

> Plan de portage par phases, basé sur l'audit `docs/AUDIT_TEAMMATES_V7_COCKPIT.md`.
> Branche source : `v7/cockpit` (Streamlit/Python, ~6 000 L sur 15 modules teammates_*).
> Branche cible : `feat/multi-title-adapters-and-mappings` (Go + React + Plotly).
> Date : 2026-04-26.

## 0. Synthèse executive

La page Coéquipiers actuelle en Go représente environ **20-25 % de la richesse fonctionnelle** de la version Python `v7/cockpit`. Sur les **22 charts/sections** identifiés dans l'audit :

| Catégorie | Nombre | Effort |
|-----------|--------|--------|
| Effort moyen (1 brique manquante) | 11 | ~10.5 j |
| Effort important (algo + endpoint + UI) | 9 | ~22 j |
| Briques transverses (helpers communs) | 14 items | ~7.5 j |
| **Total realiste** | 22 charts | **~36 j** |

**MVP "good enough"** : ~22 j pour ~80 % de la fidelite visuelle (sans LOWESS, en restant a 4 roles d'impact, sans records overlays haches).

**Aucun bloqueur de donnees** : toutes les tables DuckDB necessaires existent (`shared.highlight_events`, `shared.medals_earned`, `shared.match_weapon_kills`, `shared.match_kill_events`, `personal_score_awards` cote player DB). Le travail est entierement du dev.

**3 vrais bloqueurs techniques** :

1. **LOWESS Smoothing** (§3.6 Form Score) — pas en stdlib Go, ~2 j (impl from-scratch ou wrapper `gonum`).
2. **Impact 8 roles + N joueurs** (§3.7) — l'algo Go actuel est limite a 4 roles et concu en mode bilateral 1v1 (`myXUID`/`friendXUID`). Doit etre generalise N joueurs et etendu (silent_hero, false_brother, top_killer, last_casualty, last_group_kill, first_group_death). **Le clutch actuel est aussi a corriger** (vraie fenetre temporelle au lieu d'une approximation par tiers de la liste). ~3 j.
3. **Kill timing endpoint** (§3.9 + §4.5) — `LoadKillTimingForMatches` n'existe pas, bloque deux charts. ~0.5 j.

---

## 1. Etat actuel (verifie au code)

### 1.1 Cote Go — analyses & repository

| Fichier | Statut | Note |
|---------|--------|------|
| [internal/analysis/squad_score.go:21](../apps/go-api/internal/analysis/squad_score.go#L21) | OK | `ComputeSquadPerformanceScore` (base + 3 bonus + clamp + grade lettre). Quasi complet. |
| [internal/analysis/squad_score.go:110](../apps/go-api/internal/analysis/squad_score.go#L110) | OK | `resolveSquadGrade(score) string`. |
| [internal/analysis/performance_score.go](../apps/go-api/internal/analysis/performance_score.go) | OK | `ComputeSessionPerformanceScore` (test present). A confirmer si V2 + ajustement MMR present. |
| [internal/analysis/squad_impact.go:18](../apps/go-api/internal/analysis/squad_impact.go#L18) | INCOMPLET | `ComputeImpactSummary` ne couvre que **4 roles** (FirstBlood, Clutch, LastKill, FirstDeath) en mode 1v1. Vs 8 roles attendus, multi-joueurs. Clutch approxime via "dernier tiers de la liste". |
| [internal/analysis/highlight_event_parser.go](../apps/go-api/internal/analysis/highlight_event_parser.go) | OK | Parser highlight events utilisable pour impact etendu. |
| [internal/platform/duckdb/squad_repo.go](../apps/go-api/internal/platform/duckdb/squad_repo.go) | OK partiel | `LoadSquadMatches` (Q30 — 18 colonnes riches), `LoadTeammateMatches` (Q31), `LoadImpactEvents` (Q32), `LoadTopTeammates` (Q29). |
| [internal/service/teammates_service.go](../apps/go-api/internal/service/teammates_service.go) | OK | `buildMatchSeries`, `computeKPIsFromSquadMatches`, `TeammatesPageResponse`. |

### 1.2 Cote Go — manquants

**Algos absents** (`internal/analysis/`) : `LOWESS`, `ComputeMatchIntensityProfiles`, `ComputeSquadCadenceProfiles`, `ComputeMapBreakdown`, `ComputeSquadRecords`, `ComputeParticipationProfile`, extension impact a 8 roles + N joueurs.

**Methodes repository absentes** (`internal/platform/duckdb/squad_repo.go`) : `LoadKillTimingForMatches`, `LoadWeaponKillsAggregated`, `LoadGrenadeMeleeKills`, `LoadMedalsForMatchesByXUID`, `LoadPersonalScoreAwards`, `LoadFullPerformanceHistory`.

### 1.3 Cote frontend ([apps/web/src/features/squad/](../apps/web/src/features/squad/))

| Fichier | Lignes | Statut |
|---------|-------:|--------|
| `SquadLayout.tsx` | — | OK (selection coequipiers, scope). |
| `SquadContext.ts` | — | OK. |
| `SquadSynergiesPage.tsx` | 227 | 1 barplot generique + heatmap 1D + timeline 2-traces + HS/PK. |
| `SquadContributionsPage.tsx` | 143 | 1 radar 4-6 axes generiques. |
| `charts/heatmapChart.ts` | 72 | Heatmap 1D cartes (a passer en 2D joueur x carte). |
| `charts/timelineChart.ts` | 79 | Timeline perf + winrate (a etendre multi-joueurs + marker outcome). |
| `charts/hsPkChart.ts` | 79 | HS+PK stacked (a enrichir : records + smoothing). |
| `metrics.ts` | 101 | Wiring metriques. |
| `queries.ts` | 25 | Queries client. |

**Composants UI manquants** : `SegmentedControl` (a confirmer dans shadcn), `RecordOverlay` Plotly (motif `pattern_shape`), panneau legende joueurs flottante (`position:fixed` + `IntersectionObserver`), `MedalsGallery`, `WeaponsTable`.

---

## 2. Mapping Python -> Go (22 elements)

| # | Section | Faisabilite | Effort | Bloqueur principal |
|---|---------|------------|-------:|--------------------|
| 1 | §2.1 KPI personnels (8 cartes + tendance) | EFFORT MOYEN | 1.5 j | Endpoint `kpi-stats` + reference all-time + barre W/L/T/DNF |
| 2 | §2.2 Score equipe + grade + cartes ▲▼ | EFFORT MOYEN | 1 j | Endpoint `/squad/perf-score` + UI 4 cartes |
| 3 | §3.1 Lollipop W/L par carte | EFFORT MOYEN | 1 j | `ComputeMapBreakdown` + chart Plotly |
| 4 | §3.2 Bullet winrate session vs historique | EFFORT MOYEN | 1 j | Aggregation historique escouade par carte |
| 5 | §3.3 Perf vs historique par carte | EFFORT MOYEN | 1 j | Delta perf_score session vs historique |
| 6 | §3.4 Heatmap escouade joueur x carte | EFFORT MOYEN | 1 j | Refonte chart 1D -> 2D |
| 7 | §3.5 Timeline multi-joueurs + marker outcome | EFFORT MOYEN | 1 j | Extension chart actuel |
| 8 | §3.6 Form Score lisse (LOWESS) | EFFORT IMPORTANT | 2 j | LOWESS absent en Go |
| 9 | §3.7 Impact 8 roles (heatmap + ranking) | EFFORT IMPORTANT | 3 j | Algo Go limite a 4 roles, modele 1v1 |
| 10 | §3.8 Tableau historique escouade | EFFORT MOYEN | 1 j | Composant React + formatters |
| 11 | §3.9 Cadence trio (kills/phase 60 s) | EFFORT IMPORTANT | 2 j | `LoadKillTimingForMatches` + algo |
| 12 | §4.1 Stats par minute groupees | EFFORT MOYEN | 1 j | Normalisation par minute + chart |
| 13 | §4.2 Radar 6 axes normalises par mode | EFFORT IMPORTANT | 2 j | Refonte radar + endpoint personal_score_awards |
| 14 | §4.3 6 charts performance trio dedies | EFFORT IMPORTANT | 3 j | 5-6 charts au lieu d'un barplot generique + records overlay |
| 15 | §4.4 Killing Spree + HS/PK enrichis | EFFORT MOYEN | 1 j | KS absent ; HS/PK existant a etendre |
| 16 | §4.5 Heatmap intensite (match x 10 buckets) | EFFORT IMPORTANT | 2 j | Idem §3.9 + algo buckets + segmented_control |
| 17 | §4.6 First Events | EFFORT MOYEN | 1 j | Donnees deja chargeables via `LoadImpactEvents` |
| 18 | §4.7 Tableau armes (top N + grenade/melee) | EFFORT IMPORTANT | 2 j | Repo `LoadWeaponKillsAggregated` + cap remainder |
| 19 | §4.8 Barplot armes top 12 grouped | EFFORT MOYEN | 0.5 j | Derive direct de §4.7 |
| 20 | §4.9 Galerie medailles (top 20) | EFFORT IMPORTANT | 2 j | Repo `LoadMedalsForMatches(matchIDs)` + composant |

---

## 3. Phases de portage

### Phase 0 — Briques transverses (~7.5 j) — PREREQUIS

A faire avant toute chart pour eviter de revenir sur les helpers N fois.

#### 0.1 Algos manquants (`internal/analysis/`)
- [ ] `ComputeMapBreakdown(matches) []MapBreakdownRow` — **0.5 j** — utilise par §3.1, §3.2, §3.3, §3.4.
- [ ] `ComputeSquadRecords(series, metrics, dominantPair) SquadRecordSet` — **1 j** — utilise par §4.1, §4.3, §4.4.
- [ ] `LOWESS(points, alpha)` — **2 j** — bloqueur isole pour §3.6 (impl from-scratch ou wrapper `gonum`).
- [ ] `ComputeMatchIntensityProfiles(events, nBuckets=10)` — **0.5 j** — pour §4.5.
- [ ] `ComputeSquadCadenceProfiles(events, phaseSeconds=60)` — **0.5 j** — pour §3.9.
- [ ] `ComputeParticipationProfile(scores, options)` + `RADAR_THRESHOLDS_PER_MODE` — **1 j** — pour §4.2.
- [ ] Confirmer / completer `ComputeSessionPerformanceScoreV2(matches, includeMMRAdjustment)` — **0.5 j** — pour §2.2.
- [ ] Etendre `ComputeImpactSummary` a 8 roles + N joueurs — **2 j** — pour §3.7. **Inclut correction du clutch (vraie fenetre temporelle).**

#### 0.2 Methodes repository (`internal/platform/duckdb/squad_repo.go`)
- [ ] `LoadKillTimingForMatches(ctx, matchIDs) ([]KillTimingRow, error)` — **0.5 j** — bloque §3.9 + §4.5.
- [ ] `LoadWeaponKillsAggregated(ctx, xuid, matchIDs)` — **0.5 j** — pour §4.7.
- [ ] `LoadGrenadeMeleeKills(ctx, xuid, matchIDs)` — **0.2 j** — pour §4.7.
- [ ] `LoadMedalsForMatchesByXUID(ctx, xuids, matchIDs) ([]MedalRow, error)` — **0.3 j** — pour §4.9.
- [ ] `LoadPersonalScoreAwards(ctx, xuid, matchIDs)` — **0.3 j** — pour §4.2.
- [ ] `LoadFullPerformanceHistory(ctx, xuid)` — **0.2 j** — pour §3.6 (seuil `DETAIL_THRESHOLD`).

#### 0.3 Composants UI manquants ([apps/web/src/](../apps/web/src/))
- [ ] `SegmentedControl` — **0.3 j** (a confirmer dans shadcn — sinon coder).
- [ ] `RecordOverlay` Plotly (motif `pattern_shape` hachure) — **0.5 j** — pour records overlays.
- [ ] Panneau legende joueurs flottante (`position:fixed` + `IntersectionObserver`) — **0.5 j**.
- [ ] `MedalsGallery` (grille de cartes match avec icones medailles) — **0.5 j**.
- [ ] `WeaponsTable` generique (tri, slider min kills, colonnes par joueur) — **0.5 j**.

### Phase 1 — En-tetes structurants (~2.5 j) — PRIORITAIRE

Ces blocs sont visibles immediatement en haut de la page, donc fort impact UX.

- [ ] **§2.1 KPI personnels** — 8 cartes (matchs, duree totale, K/D/A par match, accuracy, vie moyenne, barre W/L/T/DNF) + fleches de tendance vs all-time (`_trend`, seuil 8 %). Endpoint `/players/{slug}/kpi-stats?scope=current|alltime`. Composant React `KpiStrip.tsx`. **1.5 j**
- [ ] **§2.2 Score d'equipe + scores individuels** — Carte equipe (score 0-100 + grade lettre + detail bonus) + N cartes individuelles compactes (score + label qualitatif + badge ▲/▼). Endpoint `/squad/perf-score?xuids=...&matchIds=...`. Composant `SquadScoreHeader.tsx`. **1 j**

### Phase 2 — Synergies "carte" (~4 j)

- [ ] **§3.1 Lollipop W/L par carte** — `buildLollipopMapChart.ts` (20 dernieres cartes, ordre chronologique). **1 j**
- [ ] **§3.2 Bullet winrate session vs historique** — 3 barres empilees par carte. **1 j**
- [ ] **§3.3 Perf vs historique par carte** — barres horizontales delta. **1 j**
- [ ] **§3.4 Heatmap escouade joueur x carte** — refonte de `heatmapChart.ts` en 2D. **1 j**

### Phase 3 — Timeline + Form Score (~3 j)

- [ ] **§3.5 Timeline performance multi-joueurs** — extension de `timelineChart.ts` : trace par joueur (couleur Okabe-Ito) + marker outcome (W/L/T). **1 j**
- [ ] **§3.6 Form Score lisse (LOWESS)** — endpoint `/players/{gt}/form-score?matchIds=...` + chart line lisse. **2 j** (depend du LOWESS phase 0).

### Phase 4 — Impact (~3 j)

- [ ] **§3.7 Impact 8 roles** — heatmap roles x joueurs avec emojis + tableau ranking 8 colonnes avec gradient Okabe-Ito + popover legende + toggle viz heatmap/scatter. Endpoint `/squad/impact?xuids=...&matchIds=...`. Composants `ImpactHeatmap.tsx` + `ImpactRanking.tsx`. **3 j** (depend de l'extension a 8 roles phase 0).

### Phase 5 — Tableau historique + First Events (~2 j)

- [ ] **§3.8 Tableau historique escouade** — `SquadHistoryTable.tsx` (carte, mode, playlist, date locale, resultat, lien Waypoint). **1 j**
- [ ] **§4.6 First Events** — endpoint `/squad/first-events?matchIds=...` + `buildFirstEventsChart.ts`. **1 j** (donnees deja chargeables via `LoadImpactEvents` filtre FirstBlood/FirstDeath).

### Phase 6 — Cadence + Intensite (~3 j)

- [ ] **§3.9 Cadence trio (kills/phase 60 s)** — endpoint `/squad/kill-timing` + `buildCadenceChart.ts` + note post-graphe. **1.5 j** (depend de `LoadKillTimingForMatches` phase 0).
- [ ] **§4.5 Heatmap intensite (match x 10 buckets)** — endpoint `/squad/intensity` + `segmented_control` Tous/joueur1/joueur2 + heatmap. **1.5 j**.

### Phase 7 — Charts trio (~5 j)

- [ ] **§4.1 Stats par minute groupees** — `buildPerMinuteChart.ts` (3 barres par joueur, axe zero blanc, deaths inversees). **1 j**
- [ ] **§4.3 6 charts performance trio dedies** — `buildTrioKillsDeaths.ts`, `buildTrioAssists.ts`, `buildTrioKDA.ts`, `buildTrioAccuracy.ts`, `buildTrioAvgLife.ts`, `buildTrioPerformance.ts`. Records overlays (depend phase 0). **3 j**
- [ ] **§4.4 Killing Spree (max) + HS/PK enrichis** — `buildKillingSpreeChart.ts` + enrichissement `hsPkChart.ts` (records + smoothing 10). **1 j**

### Phase 8 — Radar (~2 j)

- [ ] **§4.2 Radar complementarite (6 axes normalises par mode)** — refonte de `SquadContributionsPage`. Endpoint enrichi avec `personal_score_awards`. Normalisation par famille de mode (`is_objective_mode_from_pair_name`). **2 j** (depend de `ComputeParticipationProfile` phase 0).

### Phase 9 — Armes + Medailles (~4.5 j)

- [ ] **§4.7 Tableau armes (top N)** — `WeaponsTable.tsx` avec slider min kills + reinjection grenade/melee capee par `remainder = api_total - film_kills`. **2 j**
- [ ] **§4.8 Barplot armes top 12 grouped** — `buildWeaponsChart.ts` (derive de §4.7). **0.5 j**
- [ ] **§4.9 Galerie medailles (top 20 matchs)** — endpoint `/squad/medals` + `MedalsGallery.tsx`. **2 j**

---

## 4. Endpoints Go a exposer (recapitulatif)

| Endpoint | Phase | Usage |
|----------|-------|-------|
| `GET /players/{slug}/kpi-stats?scope=current\|alltime` | 1 | §2.1 KPI personnels + reference tendance |
| `GET /squad/perf-score?xuids=...&matchIds=...` | 1 | §2.2 scores individuels + collectif |
| `GET /squad/map-breakdown?matchIds=...` | 2 | §3.1, §3.2, §3.3, §3.4 |
| `GET /squad/map-winrate-history?matchIds=...` | 2 | §3.2 |
| `GET /players/{gt}/form-score?matchIds=...` | 3 | §3.6 |
| `GET /squad/impact?xuids=...&matchIds=...` | 4 | §3.7 (8 roles) |
| `GET /squad/first-events?matchIds=...` | 5 | §4.6 |
| `GET /squad/kill-timing?xuids=...&matchIds=...` | 6 | §3.9, §4.5 |
| `GET /squad/intensity?xuids=...&matchIds=...` | 6 | §4.5 |
| `GET /squad/weapons?xuid=...&matchIds=...` | 9 | §4.7, §4.8 |
| `GET /squad/medals?xuids=...&matchIds=...` | 9 | §4.9 |

NB : certains de ces endpoints peuvent etre fusionnes dans `TeammatesPageResponse` existant pour reduire les round-trips (a decider au moment de l'implementation).

---

## 5. Estimation par phase

| Phase | Sections | Cout | Pre-requis |
|-------|----------|-----:|------------|
| Phase 0 — Briques transverses | helpers algos + repo + UI | 7.5 j | A faire en premier |
| Phase 1 — En-tetes | §2.1, §2.2 | 2.5 j | Phase 0 (perf score v2) |
| Phase 2 — Synergies "carte" | §3.1, §3.2, §3.3, §3.4 | 4 j | `ComputeMapBreakdown` |
| Phase 3 — Timeline + Form Score | §3.5, §3.6 | 3 j | LOWESS |
| Phase 4 — Impact | §3.7 | 3 j | Extension a 8 roles |
| Phase 5 — Tableau + First Events | §3.8, §4.6 | 2 j | — |
| Phase 6 — Cadence + Intensite | §3.9, §4.5 | 3 j | `LoadKillTimingForMatches` |
| Phase 7 — Charts trio | §4.1, §4.3, §4.4 | 5 j | `ComputeSquadRecords` + `RecordOverlay` |
| Phase 8 — Radar | §4.2 | 2 j | `ComputeParticipationProfile` |
| Phase 9 — Armes + Medailles | §4.7, §4.8, §4.9 | 4.5 j | Repo armes + medailles |
| **Total** | 22 charts + briques | **~36 j** | |

**MVP "good enough" ~22 j** : Phase 0 (sans LOWESS) + Phases 1, 2, 4 (en restant a 4 roles), 5, 7 (sans records overlays), 9 (medailles seulement). Ramene la fidelite visuelle a ~80 %.

---

## 6. Conventions & contraintes

### 6.1 Architecture
- Respecter `arch-rules` : separation claire `internal/analysis/` (pure) vs `internal/service/` (orchestration) vs `internal/platform/duckdb/` (acces donnees).
- Pas de logique metier dans `internal/api/handlers/`.
- Tests Go obligatoires pour chaque algo de `internal/analysis/`.

### 6.2 Frontend
- Respecter `color-tokens` : aucune couleur hex dans `apps/web/src/features/squad/`. Utiliser `tokenCssVar`, `resolveToken`, `getSeriesColors`.
- i18n FR/EN pour toutes les nouvelles strings (utiliser `getSquadText` ou etendre).
- Field mappings via `useFieldMappings` pour les libelles metier (multi-titres).
- Tous les `PlotlyChart` passent par le wrapper centralise.

### 6.3 Tests
- Couvrir chaque endpoint avec un test handler.
- Snapshots Plotly figures pour eviter les regressions visuelles (si tooling dispo).
- Tests React Testing Library pour les empty states (no_selection / invalid_selection / no_chart_data).

### 6.4 Documentation
- Mettre a jour `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` au fil de l'eau si on devie.
- Entrees `.ai/thought_log.md` a chaque phase terminee.

---

## 7. Suivi

| Phase | Statut | Branche | PR | Note |
|-------|--------|---------|----|----|
| 0 — Briques transverses | A faire | — | — | |
| 1 — En-tetes | A faire | — | — | |
| 2 — Synergies carte | A faire | — | — | |
| 3 — Timeline + Form Score | A faire | — | — | |
| 4 — Impact | A faire | — | — | |
| 5 — Tableau + First Events | A faire | — | — | |
| 6 — Cadence + Intensite | A faire | — | — | |
| 7 — Charts trio | A faire | — | — | |
| 8 — Radar | A faire | — | — | |
| 9 — Armes + Medailles | A faire | — | — | |

---

## 8. References

- Audit complet : [docs/AUDIT_TEAMMATES_V7_COCKPIT.md](../docs/AUDIT_TEAMMATES_V7_COCKPIT.md)
- Plan match view (modele de format) : [.ai/PLAN_MATCH_VIEW_GO_PORTAGE.md](PLAN_MATCH_VIEW_GO_PORTAGE.md)
- Source Python : branche `v7/cockpit`, modules `src/ui/pages/teammates*.py`, `src/data/services/teammates_service.py`, `src/analysis/{friends_impact,squad_records,match_intensity,_performance_form,_performance_squad,participation_radar}.py`, `src/visualization/{squad_*,teammates_*,match_intensity_heatmap,trio,_form_score}.py`.
- Cible Go : `apps/go-api/internal/{analysis,service,platform/duckdb}/squad_*.go` + `apps/web/src/features/squad/`.
