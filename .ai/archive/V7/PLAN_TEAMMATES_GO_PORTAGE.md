# Plan de portage de la page Coéquipiers — Python v7/cockpit -> Go + React

> Plan de portage par phases, basé sur l'audit `docs/AUDIT_TEAMMATES_V7_COCKPIT.md`.
> Branche source : `v7/cockpit` (Streamlit/Python, ~6 000 L sur 15 modules teammates_*).
> Branche cible : `feat/multi-title-adapters-and-mappings`.
> Branche de travail : **`feat/squad-page-portage`** (à créer depuis la branche cible avant Phase 0).
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

1. **LOWESS Smoothing** (§3.6 audit / Form Score) — pas en stdlib Go, ~2 j (impl from-scratch ou wrapper `gonum`).
2. **Impact 8 roles + N joueurs** (§3.7 audit) — l'algo Go actuel est limite a 4 roles et concu en mode bilateral 1v1 (`myXUID`/`friendXUID`). Doit etre generalise N joueurs et etendu (silent_hero, false_brother, top_killer, last_casualty, last_group_kill, first_group_death). **Le clutch actuel est aussi a corriger** (vraie fenetre temporelle au lieu d'une approximation par tiers de la liste). ~3 j.
3. **Kill timing endpoint** (§3.9 + §4.5 audit) — `LoadKillTimingForMatches` n'existe pas, bloque deux charts. ~0.5 j.

---

## 1. Etat actuel (verifie au code)

### 1.1 Cote Go — analyses & repository

| Fichier | Statut | Note |
|---------|--------|------|
| [internal/analysis/squad_score.go:21](../apps/go-api/internal/analysis/squad_score.go#L21) | OK | `ComputeSquadPerformanceScore` (base + 3 bonus + clamp + grade lettre). Quasi complet. |
| [internal/analysis/squad_score.go:110](../apps/go-api/internal/analysis/squad_score.go#L110) | OK | `resolveSquadGrade(score) string`. |
| [internal/analysis/performance_score.go](../apps/go-api/internal/analysis/performance_score.go) | OK | `ComputeSessionPerformanceScore` (test present). Ajustement MMR present (verifie : `EnemyMMR` consomme). |
| [internal/analysis/squad_impact.go:18](../apps/go-api/internal/analysis/squad_impact.go#L18) | INCOMPLET | `ComputeImpactSummary` ne couvre que **4 roles** (FirstBlood, Clutch, LastKill, FirstDeath) en mode 1v1. Vs 8 roles attendus, multi-joueurs. Clutch approxime via "dernier tiers de la liste". |
| [internal/analysis/highlight_event_parser.go](../apps/go-api/internal/analysis/highlight_event_parser.go) | OK | Parser highlight events utilisable pour impact etendu. |
| [internal/platform/duckdb/squad_repo.go](../apps/go-api/internal/platform/duckdb/squad_repo.go) | OK partiel | `LoadTopTeammates`, `LookupXUIDByGamertag`, `LoadSquadMatches` (Q30 — 18 colonnes riches), `LoadTeammateMatches` (Q31), `LoadImpactEvents` (Q32). |
| [internal/service/teammates_service.go](../apps/go-api/internal/service/teammates_service.go) | OK | `buildMatchSeries`, `computeKPIsFromSquadMatches`, `TeammatesPageResponse`. |
| [internal/service/squad_service.go](../apps/go-api/internal/service/squad_service.go) | OK | Existant. **N'utilise pas `TitleDataAdapter`** — lit `port.SquadRepository` directement. |

### 1.2 Cote Go — manquants

**Algos absents** (`internal/analysis/`) : `LOWESS`, `ComputeMatchIntensityProfiles`, `ComputeSquadCadenceProfiles`, `ComputeMapBreakdown`, `ComputeSquadRecords`, `ComputeParticipationProfile`, extension impact a 8 roles + N joueurs.

**Methodes repository absentes** (`internal/platform/duckdb/squad_repo.go` + interface dans `internal/port/repository.go`) : `LoadKillTimingForMatches`, `LoadWeaponKillsAggregated`, `LoadGrenadeMeleeKills`, `LoadMedalsForMatchesByXUID`, `LoadPersonalScoreAwards`, `LoadFullPerformanceHistory`.

**Handlers HTTP absents** (`internal/api/handlers/`) : aucun handler `squad_perf_score`, `squad_impact`, `squad_intensity`, `squad_cadence`, `squad_weapons`, `squad_medals`, `squad_form_score`, `squad_first_events`, `squad_map_breakdown`, `squad_kpi_stats`.

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

## 3. Conventions techniques transverses (a respecter dans toutes les phases)

### 3.1 Architecture Go — separation des couches

| Couche | Localisation | Regle |
|--------|--------------|-------|
| Algorithme pur | `internal/analysis/` | Stateless. Entree = struct/slice. Sortie = struct. **Aucun acces DB, aucun import de `service` ou `port`.** Tests unitaires obligatoires. |
| Type metier | `internal/domain/` | Squad reste **title-specific** (cf. decision §3.2). Nouveaux types (`MapBreakdownRow`, `KillTimingRow`, `SquadRecordSet`, `ImpactSummaryV2`, `IntensityProfile`, `CadenceProfile`, `WeaponStatsRow`, `SquadMedalRow`, `KPIStats`) vont dans `internal/domain/squad.go` (ou nouveau `internal/domain/teammates.go`). |
| Type canonique | `internal/games/canonical/` | **Pas concerne** ici (squad = title-specific, voir §3.2). Si on capability-gate plus tard, ajouter `CapSquadSession` dans `games/adapter.go`. |
| Orchestration | `internal/service/` | Compose repos + algos. Renvoie `*domain.XxxResponse`. **Aucun SQL inline.** Tests avec mock `port.SquadRepository`. |
| Interface repository | `internal/port/repository.go` | Toute nouvelle methode DuckDB doit **d'abord** etre declaree ici, puis implementee dans `platform/duckdb/squad_repo.go`, puis stubbed dans `noopSquadRepo`, puis mockee dans les tests service. |
| Implementation DuckDB | `internal/platform/duckdb/squad_repo.go` + `queries_squad.go` | Requetes SQL constantes nommees `Q34*`, `Q35*`, etc. Tests `:memory:` obligatoires pour chaque nouvelle methode. |
| Handler HTTP | `internal/api/handlers/` | **Aucune logique metier.** Decode params, appelle service, encode JSON. Tests `httptest`. |

### 3.2 Multi-titres — decision assumee pour Squad

**Decision** : la page Squad reste **title-specific Halo Infinite** dans cette iteration. Les services `teammates_service.go` et `squad_service.go` continuent d'utiliser `port.SquadRepository` directement, sans passer par `games.TitleDataAdapter`. Justification :
- Squad fait sens uniquement pour les titres avec notion d'equipe persistante + matchmaking (Halo Infinite). D'autres titres futurs (campagne solo, battle royale) n'auront pas de "coequipiers frequents".
- L'algo d'impact (8 roles) depend de `highlight_events` Halo-specifique. Pas de mapping generique pertinent.
- Generaliser via adapter ajouterait 5-7 j sans benefice immediat.

**Implications** :
- Pas d'ajout de capability `CapSquadSession` pour l'instant.
- Pas de modification de `config/titles/halo_infinite/mappings/{fields,assets,outcomes}.toml` **sauf** si on introduit de nouveaux `FieldKey` exposes au frontend (voir §3.5).
- Cote frontend, la page reste accessible uniquement quand `titleSlug == "halo_infinite"`. Capability gate cote React via `useFieldMappings()` (deja en place dans `SquadSynergiesPage.tsx`) — degradation gracieuse "feature unavailable" si mapping absent.

**Si l'utilisateur change d'avis** : extraire dans une iteration ulterieure (ajouter `CapSquadSession`, deplacer types vers `canonical/`, creer `adapter_squad.go` par titre).

### 3.3 Logging — conventions

- Erreurs non-triviales : `slog.ErrorContext(ctx, "squad: load impact events failed", "err", err, "match_count", len(matchIDs))`.
- Operations significatives : `slog.InfoContext(ctx, "squad: computing intensity profiles", "n_players", n, "n_matches", m, "duration_ms", elapsed)`.
- Cles structurees standards : `"err"`, `"match_id"`, `"player"`, `"xuid"`, `"titleSlug"`, `"duration_ms"`, `"n_matches"`, `"n_players"`.
- **Interdit** : `fmt.Println`, `log.Printf`, `log.Println` dans le nouveau code.
- Loggers per-package via `slog.Default()` ou logger injecte (suivre le pattern existant dans `squad_service.go`).

### 3.4 Tests — strategie par couche

| Couche | Outil | Couverture attendue |
|--------|-------|---------------------|
| `internal/analysis/` | `testing` standard | 100 % des branches algo. Tests deterministes (fixtures inline). Cas limites : empty slice, single match, donnees corrompues. |
| `internal/service/` | `testing` + mock `port.SquadRepository` | Wiring repo -> algo -> domain. Cas degradation : repo retourne erreur, repo retourne empty, methode non disponible (capability). |
| `internal/api/handlers/` | `httptest` | Decodage params (query, path), codes HTTP, format JSON, gestion d'erreur (400 / 404 / 500). |
| `internal/platform/duckdb/` | `:memory:` DuckDB avec schema seed | Une methode = un test. Verifier aussi la perf (LIMIT respecte, INDEX si besoin). |
| Frontend | Vitest | Empty states (no_selection / invalid_selection / no_chart_data), formatters i18n, hooks de query. |

**Tests de degradation obligatoires** :
- `personal_score_awards` table absente -> radar §4.2 retourne empty profile, pas d'erreur 500.
- `shared.match_kill_events` table absente -> §3.9 et §4.5 retournent empty, pas d'erreur 500.
- `shared.match_weapon_kills` empty pour un match -> §4.7 affiche tableau vide proprement.

### 3.5 Frontend — conventions

- Aucune couleur hex `#RRGGBB` ni classe Tailwind couleur dans `apps/web/src/features/squad/`. Utiliser `tokenCssVar(token)`, `resolveToken(token)`, `getSeriesColors(n, tokens[])`.
- i18n FR + EN obligatoire. Les nouvelles strings vont dans `apps/web/src/features/squad/i18n.ts` (`getSquadText(locale)`).
- Libelles metier (KPI, metriques radar, scores) **toujours** via `useFieldMappings()` + `FieldKey`. Si un nouveau `FieldKey` est introduit (`form_score_smoothed`, `intensity_bucket_kills`, `radar_objective_score`, etc.) :
  - Ajouter la constante dans `internal/games/canonical/fields.go`.
  - Ajouter la section dans `config/titles/halo_infinite/mappings/fields.toml` (label FR + EN).
  - Cote React, consommer via `useFieldLabel(FieldKey)`.
- Toutes les nouvelles queries client dans `apps/web/src/lib/query/keys.ts` (cles serialisables, format `['squad', 'perf-score', xuids, matchIds]`).
- Plotly figures via le wrapper `PlotlyChart` deja existant. Pas de `import Plot from 'react-plotly.js'` direct.
- **Interdiction** d'editer `routeTree.gen.ts` (genere par TanStack Router file-based).

### 3.6 Branche Git

- **Une seule branche** : `feat/squad-page-portage` depuis `feat/multi-title-adapters-and-mappings`.
- Phases = commits successifs, pas de sous-branches.
- Format commit : `feat(squad): phase X — courte description` avec corps detaillant ce qui est ajoute.

### 3.7 Dependances externes

- **`gonum/stat`** : a evaluer pour Phase 0 LOWESS. Decision a prendre lors de Phase 0 (impl from-scratch ~1.5 j vs wrapper gonum ~0.5 j + nouvelle dep `go.mod`). Si gonum retenu, justifier dans le commit message + `thought_log`.
- Aucune autre dependance externe prevue.

### 3.8 Stack chart frontend — Plotly (decision assumee)

**Decision** : on **reste sur Plotly** (`react-plotly.js` v2.6, `plotly.js` v3.5). Justification :
- Stack dominante dans tout le projet : `components/ui/plotly-chart.tsx` consomme par career, citations, session-compare, timeseries, squad existant.
- Charts existants `charts/{heatmap,timeline,hsPk}Chart.ts` deja en `PlotlyFigurePayload`.
- Bascule vers ECharts = migration de tout le projet, sans benefice technique pour squad.
- Plotly serialise tout en JSON, donc on peut **extraire les options exactes** des charts Python v7/cockpit (qui utilise aussi Plotly) -> portage quasi 1:1 facilite.

**Recharts** est present dans `match-view/MatchViewPage.tsx` (heritage isole, hors scope squad).

### 3.8 Done definition par phase

Une phase est consideree **DONE** quand :
- [ ] Tous les criteres de la checklist phase sont coches.
- [ ] `go test ./...` passe (depuis `apps/go-api/`).
- [ ] `go vet ./...` sans warning.
- [ ] `npm run typecheck` + `npm run lint` passent (depuis `apps/web/`).
- [ ] Aucun `fmt.Println` ni hex color introduit.
- [ ] Une entree `.ai/thought_log.md` est ajoutee (date, titre, decision, resultats, prochaine etape).
- [ ] Commit pousse sur `feat/squad-page-portage`.

---

## 4. Phases de portage

> Format de chaque phase : **Algos** (`internal/analysis/`) + **Domain** (`internal/domain/`) + **Port** (`internal/port/`) + **Repo** (`internal/platform/duckdb/`) + **Service** (`internal/service/`) + **Handler** (`internal/api/handlers/`) + **Frontend** (`apps/web/src/features/squad/`) + **Tests** + **i18n**.

### Phase 0bis — Specification visuelle (~3 j) — PREREQUIS PARALLELE

A faire **en parallele de Phase 0** (la spec visuelle ne bloque pas le dev backend Go). Doit etre **terminee avant Phase 1** sinon les charts seront approximatifs.

**Livrables** (3 documents) :

#### 0bis.1 [docs/SQUAD_VISUAL_SPEC.md](../docs/SQUAD_VISUAL_SPEC.md) — **1.5 j**
Pour chacun des 22 charts/sections + tableaux, une fiche structuree :
- Screenshot de reference depuis l'app Python `v7/cockpit` (lance Streamlit local, capture).
- Type Plotly exact + composition si type compose (lollipop = `bar` + `scatter`, bullet = `bar` empilees, etc.).
- Mapping data -> trace (`x`, `y`, `marker.color`, `text`, `customdata`).
- Layout precis (`xaxis.type`, `yaxis.range`, `barmode`, `hovermode`).
- Tooltip (`hovertemplate`) avec contenu exact.
- Legende (position, orient, visibility).
- Annotations (`shapes`, `annotations`).
- Formatters (durees, nombres, pourcentages, KDA `.3f`).
- Comportements interactifs (zoom, dataZoom, selectedpoints).
- Empty state, loading, error states.

**Methode** : pour les charts Python existants, extraire le JSON Plotly via `fig.to_json()` -> portable directement vers `PlotlyFigurePayload`. Pour les composants compose, decomposer en sous-traces.

#### 0bis.2 [docs/SQUAD_DESIGN_TOKENS.md](../docs/SQUAD_DESIGN_TOKENS.md) — **0.5 j**
- Palette joueurs Okabe-Ito (8 couleurs canoniques) + variantes claires/sombres pour deaths inversees (`_negative_color`).
- Mapping deterministe joueur (par xuid hash) -> indice palette.
- Couleurs outcome (W=`rgba(0,158,115,0.30)`, L=`rgba(213,94,0,0.30)`, T=`rgba(100,100,130,0.15)`) en tokens semantiques.
- Heatmap colorscales : perf_score (perso 0-100), win_rate (0-100), intensity (0-1 normalise).
- Records overlay : motif Plotly (`bar.marker.pattern.shape='/'`, `bar.marker.pattern.fgopacity=0.4`).
- Emojis impact + fallback texte.
- Mapping CSS classes Python (`os-perf-card`, `os-table`, `v7-context-toolbar-label`) vers composants React equivalents.

#### 0bis.3 [docs/SQUAD_CHART_HELPERS.md](../docs/SQUAD_CHART_HELPERS.md) — **1 j**
Specification des helpers TS a implementer dans `apps/web/src/features/squad/charts/_helpers.ts` (l'implementation effective est faite en debut de Phase 0) :
- `formatDuration(seconds, format='auto'|'mm:ss'|'h:mm:ss'|'d:h:mm')`.
- `formatNumber(n, options)`.
- `formatKDA(value)` (toujours `.3f`).
- `formatPercent(value, decimals)`.
- `attributePlayerColor(xuid, palette)` (deterministe, hash xuid).
- `applyHaloPlotStyle(option)` (equivalent `apply_halo_plot_style` de `theme.py`).
- `applyRecordsOverlay(option, records)` (equivalent `add_record_overlays`).
- `hideLegend(option)` (equivalent `_hide_legend`, masque legende et `update_traces.showlegend=false`).
- `outcomeMarkerSymbol(outcome) -> 'circle'|'cross'|'diamond'`.
- `outcomeBackgroundColor(outcome) -> tokenCssVar`.

**Done definition Phase 0bis** :
- [ ] `docs/SQUAD_VISUAL_SPEC.md` : 22 fiches remplies, 22 screenshots associes.
- [ ] `docs/SQUAD_DESIGN_TOKENS.md` : palette + mappings + colorscales documentes.
- [ ] `docs/SQUAD_CHART_HELPERS.md` : 10 signatures helpers + cas d'usage.
- [ ] Decision `gonum/stat` vs LOWESS from-scratch tranchee dans `thought_log`.

### Phase 0 — Briques transverses (~7.5 j) — PREREQUIS

A faire avant toute chart pour eviter de revenir sur les helpers N fois.

#### 0.1 Algos (`internal/analysis/`)
- [ ] `ComputeMapBreakdown(matches []domain.SquadMatchRow) []domain.MapBreakdownRow` — **0.5 j** — utilise par §3.1, §3.2, §3.3, §3.4. Test unitaire avec 3 cartes / outcomes mixtes.
- [ ] `ComputeSquadRecords(series, metrics, dominantPair) domain.SquadRecordSet` — **1 j** — utilise par §4.1, §4.3, §4.4. Test avec `pm_records` fallback.
- [ ] `LOWESS(points []Point, alpha float64) []Point` — **2 j** — bloqueur isole pour §3.6. **Decision a prendre** : impl from-scratch ou wrapper `gonum/stat`. Test : courbe de reference numpy/scipy comparee a tolerance 1e-3.
- [ ] `ComputeMatchIntensityProfiles(events []KillTimingRow, nBuckets int) []domain.IntensityProfile` — **0.5 j** — pour §4.5. Test : repartition deterministe sur 10 buckets.
- [ ] `ComputeSquadCadenceProfiles(events []KillTimingRow, phaseSeconds int) []domain.CadenceProfile` — **0.5 j** — pour §3.9.
- [ ] `ComputeParticipationProfile(awards []PersonalScoreAward, opts ProfileOptions) []domain.RadarProfile` + constante `RADAR_THRESHOLDS_PER_MODE` — **1 j** — pour §4.2. Test : normalisation par famille de mode (Slayer / CTF / Strongholds / Oddball / Custom).
- [ ] Etendre `ComputeImpactSummary` (renommer en `ComputeImpactSummaryV2`) a 8 roles + N joueurs — **2 j** — pour §3.7. **Inclut correction du clutch (vraie fenetre temporelle 30s finales).** Tests : un test par role + test multi-joueurs (3+).

#### 0.2 Domain (`internal/domain/`)
- [ ] Ajouter `MapBreakdownRow`, `KillTimingRow`, `IntensityProfile`, `CadenceProfile`, `RadarProfile`, `SquadRecordSet`, `ImpactSummaryV2`, `WeaponStatsRow`, `SquadMedalRow`, `KPIStats`, `PersonalScoreAward`, `FormScorePoint` dans `internal/domain/squad.go` (ou nouveau `teammates.go` si > 500 L).
- [ ] Documenter chaque struct (godoc).

#### 0.3 Port (`internal/port/repository.go`)
- [ ] Etendre l'interface `SquadRepository` avec :
  - `LoadKillTimingForMatches(ctx, matchIDs []string) ([]domain.KillTimingRow, error)`
  - `LoadWeaponKillsAggregated(ctx, xuid string, matchIDs []string) ([]domain.WeaponKillRow, error)`
  - `LoadGrenadeMeleeKills(ctx, xuid string, matchIDs []string) (domain.GrenadeMeleeAggregate, error)`
  - `LoadMedalsForMatchesByXUID(ctx, xuids []string, matchIDs []string) ([]domain.SquadMedalRow, error)`
  - `LoadPersonalScoreAwards(ctx, xuid string, matchIDs []string) ([]domain.PersonalScoreAward, error)`
  - `LoadFullPerformanceHistory(ctx, xuid string) ([]domain.FormScorePoint, error)`
- [ ] Stubber chaque methode dans `noopSquadRepo` (retourne empty + nil).
- [ ] Mettre a jour les mocks dans `service/squad_service_test.go` et `service/teammates_extra_test.go` pour implementer les nouvelles methodes.

#### 0.4 Repo (`internal/platform/duckdb/`)
- [ ] Implementer les 6 methodes ci-dessus dans `squad_repo.go`. Requetes nommees `Q34KillTiming`, `Q35WeaponKillsAggregated`, `Q36GrenadeMeleeKills`, `Q37SquadMedals`, `Q38PersonalScoreAwards`, `Q39FullPerformanceHistory` dans `queries_squad.go`.
- [ ] Tests `:memory:` pour chacune (1 fichier `squad_repo_phase0_test.go` avec 6 sous-tests). Verifier degradation gracieuse si table absente.

#### 0.5 Composants UI manquants ([apps/web/src/](../apps/web/src/))
- [ ] `SegmentedControl` — **0.3 j** — confirmer presence dans shadcn-ui local. Sinon, creer `apps/web/src/components/ui/segmented-control.tsx`.
- [ ] `RecordOverlay` — **0.5 j** — helper qui prend une `PlotlyFigurePayload` et ajoute des traces fantomes hachurees (`pattern_shape`). Localisation : `apps/web/src/lib/plotly/record-overlay.ts`.
- [ ] Panneau legende joueurs flottante — **0.5 j** — composant `apps/web/src/features/squad/components/SquadPlayerLegend.tsx`. Logique IntersectionObserver sur ancres `#llp-squad-start` / `#llp-medals-start`.
- [ ] `MedalsGallery` — **0.5 j** — composant `apps/web/src/features/squad/components/MedalsGallery.tsx`. Grid de cartes match.
- [ ] `WeaponsTable` — **0.5 j** — composant `apps/web/src/features/squad/components/WeaponsTable.tsx`. Colonnes dynamiques par joueur, slider min kills.

#### 0.6 i18n
- [ ] Ajouter cles transverses dans `apps/web/src/features/squad/i18n.ts` (FR + EN) : `squad.legend.show`, `squad.records.show`, `squad.weapons.minKills`, `squad.medals.viewMatch`, etc.

### Phase 1 — En-tetes structurants (~2.5 j) — PRIORITAIRE

#### §2.1 KPI personnels — **1.5 j**
- **Algos** : confirmer / completer `ComputeKPIStats(matches, scope) domain.KPIStats` dans `internal/analysis/kpi.go`. Inclure `_trend(current, reference, higherIsBetter, threshold=0.08)`.
- **Domain** : `KPIStats` (champs : `TotalMatches`, `TotalPlaySeconds`, `AvgMatchSeconds`, `KillsPerGame/Min`, `DeathsPerGame/Min`, `AssistsPerGame/Min`, `AvgAccuracy`, `AvgLifeSeconds`, `Wins`, `Losses`, `Ties`, `NoFinish`).
- **Service** : `KPIService.GetKPIStats(ctx, xuid, scope)`. Charge `dff` (current) + `df` (alltime) via repo, calcule trends.
- **Handler** : `GET /players/{slug}/kpi-stats?scope=current|alltime` -> 200 / 404 (player inconnu) / 400 (scope invalide).
- **Frontend** : composant `apps/web/src/features/squad/components/KpiStrip.tsx` (8 cartes + barre W/L/T/DNF). Query key `['players', slug, 'kpi-stats', scope]`.
- **Tests** : analysis (calculs deterministes), service (mock repo), handler (httptest), composant (vitest empty + happy + trend variations).
- **i18n** : `kpi_selected_matches`, `kpi_total_duration`, `kpi_kills_per_match`, `kpi_deaths_per_match`, `kpi_assists_per_match`, `kpi_avg_accuracy`, `kpi_avg_lifespan`, `kpi_wins/losses/ties/no_finish`.

#### §2.2 Score equipe + scores individuels — **1 j**
- **Algos** : confirmer `ComputeSessionPerformanceScoreV2(matches, includeMMRAdjustment bool)` dans `performance_score.go` (verifier presence ajustement MMR). `ComputeSquadPerformanceScore` deja porte.
- **Domain** : extension `SquadPerformanceScore` avec `Components{BaseAvg, TeamWinRate, MinKD, KillsStd}`.
- **Service** : `SquadService.GetPerfScore(ctx, xuids []string, matchIDs []string)`.
- **Handler** : `GET /squad/perf-score?xuids=...&matchIds=...`.
- **Frontend** : composant `SquadScoreHeader.tsx` (1 carte equipe avec grade lettre + N cartes individuelles compactes avec badge ▲/▼).
- **Tests** : algo (deja teste), service (mock repo, calcul collectif), handler (httptest), composant (badges, grades).
- **i18n** : `squad_score_header`, `squad_score_bonus`, `squad_score_base_only`, `squad_grade_*`.

### Phase 2 — Synergies "carte" (~4 j)

#### §3.1 Lollipop W/L par carte — **1 j**
- **Service** : `SquadService.GetMapBreakdown(ctx, matchIDs []string)`.
- **Handler** : `GET /squad/map-breakdown?matchIds=...`.
- **Frontend** : `charts/lollipopMapChart.ts` (20 dernieres cartes, ordre chronologique).
- **Tests** : service (mock), handler, chart (snapshot).

#### §3.2 Bullet winrate session vs historique — **1 j**
- **Service** : etendre `GetMapBreakdown` pour inclure `historical_breakdown` (ou nouveau `GetMapWinrateHistory`).
- **Handler** : option query `?include_history=true` ou endpoint dedie `/squad/map-winrate-history`.
- **Frontend** : `charts/bulletWinrateChart.ts`.

#### §3.3 Perf vs historique par carte — **1 j**
- **Service** : extension du meme endpoint avec delta `perf_session - perf_historique`.
- **Frontend** : `charts/perfDeltaChart.ts`.

#### §3.4 Heatmap escouade joueur x carte — **1 j**
- **Frontend** : refonte `charts/heatmapChart.ts` en 2D (player axis Y, map axis X, cellule = perf_score). Donnees deja chargees via `TeammatesPageResponse.MapBreakdown`.
- **Tests** : snapshot 2D.

### Phase 3 — Timeline + Form Score (~3 j)

#### §3.5 Timeline performance multi-joueurs — **1 j**
- **Frontend** : extension de `charts/timelineChart.ts` : trace par joueur (couleur Okabe-Ito), marker outcome (W/L/T = symbol + color).
- **Tests** : snapshot multi-joueurs.

#### §3.6 Form Score lisse (LOWESS) — **2 j**
- **Algos** : `LOWESS` deja en Phase 0.
- **Service** : `SquadService.GetFormScoreHistory(ctx, xuid string, matchIDs []string)`. Charge full history via `LoadFullPerformanceHistory`, lisse via LOWESS.
- **Handler** : `GET /players/{gt}/form-score?matchIds=...`.
- **Frontend** : `charts/formScoreChart.ts` (line chart lisse).
- **Tests** : algo (Phase 0), service (mock), handler.

### Phase 4 — Impact (~3 j)

#### §3.7 Impact 8 roles — **3 j**
- **Algos** : `ComputeImpactSummaryV2` deja en Phase 0.
- **Service** : `SquadService.GetImpact(ctx, xuids []string, matchIDs []string)`.
- **Handler** : `GET /squad/impact?xuids=...&matchIds=...`.
- **Frontend** : composants `ImpactHeatmap.tsx` + `ImpactRanking.tsx`. Heatmap roles x joueurs avec emojis (⚡🎯💀🐌🪦🛡️🗡️💥), fond outcome (W/L/T), tableau ranking 8 colonnes avec gradient Okabe-Ito + popover legende + toggle viz heatmap/scatter.
- **Tests** : service (mock 8 roles), handler, composants (empty / partial / full).
- **i18n** : `tm_impact_*` (8 cles role + legende complete).

### Phase 5 — Tableau historique + First Events (~2 j)

#### §3.8 Tableau historique escouade — **1 j**
- **Service** : utilise `LoadSquadMatches` existant + formatters (date locale, mode normalise via `mode_label.go`, playlist).
- **Handler** : reutilise endpoint existant ou ajoute `GET /squad/history?xuids=...&matchIds=...`.
- **Frontend** : composant `SquadHistoryTable.tsx` (carte, mode, playlist, date locale, resultat, lien Waypoint).
- **Tests** : composant (tri date desc, formatters).

#### §4.6 First Events — **1 j**
- **Service** : reutilise `LoadImpactEvents` filtre `event_type IN (FirstBlood, FirstDeath)`.
- **Handler** : `GET /squad/first-events?matchIds=...`.
- **Frontend** : `charts/firstEventsChart.ts`.

### Phase 6 — Cadence + Intensite (~3 j)

#### §3.9 Cadence trio — **1.5 j**
- **Algos** : `ComputeSquadCadenceProfiles` deja en Phase 0.
- **Repo** : `LoadKillTimingForMatches` deja en Phase 0.
- **Service** : `SquadService.GetCadence(ctx, xuids []string, matchIDs []string)`.
- **Handler** : `GET /squad/cadence?xuids=...&matchIds=...`.
- **Frontend** : `charts/cadenceChart.ts` + note post-graphe `tm_note_cadence`.
- **Tests** : algo, service, handler.

#### §4.5 Heatmap intensite — **1.5 j**
- **Algos** : `ComputeMatchIntensityProfiles` deja en Phase 0.
- **Service** : `SquadService.GetIntensity(ctx, xuids []string, matchIDs []string)`.
- **Handler** : `GET /squad/intensity?xuids=...&matchIds=...`.
- **Frontend** : composant `SquadIntensityHeatmap.tsx` avec `SegmentedControl` (Tous / joueur1 / joueur2).

### Phase 7 — Charts trio (~5 j)

#### §4.1 Stats par minute groupees — **1 j**
- **Frontend** : `charts/perMinuteChart.ts` (3 barres par joueur, axe zero blanc, deaths inversees).

#### §4.3 6 charts performance trio dedies — **3 j**
- **Frontend** : 6 fichiers `charts/trio{KillsDeaths,Assists,KDA,Accuracy,AvgLife,Performance}Chart.ts`. Records overlays via `RecordOverlay` (Phase 0).
- **Service** : exposer `SquadRecordSet` dans `TeammatesPageResponse` (toggle setting `show_records`).

#### §4.4 Killing Spree + HS/PK enrichis — **1 j**
- **Frontend** : `charts/killingSpreeChart.ts` + enrichissement `hsPkChart.ts` (records + smoothing 10).

### Phase 8 — Radar (~2 j)

#### §4.2 Radar complementarite — **2 j**
- **Algos** : `ComputeParticipationProfile` deja en Phase 0.
- **Repo** : `LoadPersonalScoreAwards` deja en Phase 0.
- **Service** : `SquadService.GetRadarProfiles(ctx, xuids []string, matchIDs []string)`.
- **Handler** : `GET /squad/radar?xuids=...&matchIds=...`.
- **Frontend** : refonte de `SquadContributionsPage.tsx` : 6 axes Combat / Survie / Soutien / Score / Objectifs / Impact, normalisation par famille de mode.
- **i18n + FieldKeys** : ajouter `radar.combat`, `radar.survie`, `radar.soutien`, `radar.score`, `radar.objectifs`, `radar.impact` dans `canonical/fields.go` + `fields.toml`.

### Phase 9 — Armes + Medailles (~4.5 j)

#### §4.7 Tableau armes — **2 j**
- **Repo** : `LoadWeaponKillsAggregated` + `LoadGrenadeMeleeKills` deja en Phase 0.
- **Service** : `SquadService.GetWeapons(ctx, xuid string, matchIDs []string)` avec reinjection grenade/melee capee par `remainder = api_total - film_kills`.
- **Handler** : `GET /squad/weapons?xuid=...&matchIds=...`.
- **Frontend** : composant `WeaponsTable` (Phase 0) wire dans `SquadContributionsPage`.

#### §4.8 Barplot armes top 12 grouped — **0.5 j**
- **Frontend** : `charts/weaponsBarChart.ts` (derive de §4.7).

#### §4.9 Galerie medailles — **2 j**
- **Repo** : `LoadMedalsForMatchesByXUID` deja en Phase 0.
- **Service** : `SquadService.GetMedals(ctx, xuids []string, matchIDs []string)` (top 20 matchs).
- **Handler** : `GET /squad/medals?xuids=...&matchIds=...`.
- **Frontend** : composant `MedalsGallery` (Phase 0) wire dans `SquadContributionsPage`.

---

## 5. Endpoints Go a exposer (recapitulatif)

| Endpoint | Phase | Handler | Service | Usage |
|----------|-------|---------|---------|-------|
| `GET /players/{slug}/kpi-stats?scope=current\|alltime` | 1 | `kpi_stats.go` | `KPIService.GetKPIStats` | §2.1 |
| `GET /squad/perf-score?xuids=...&matchIds=...` | 1 | `squad_perf_score.go` | `SquadService.GetPerfScore` | §2.2 |
| `GET /squad/map-breakdown?matchIds=...&include_history=...` | 2 | `squad_map_breakdown.go` | `SquadService.GetMapBreakdown` | §3.1, §3.2, §3.3, §3.4 |
| `GET /players/{gt}/form-score?matchIds=...` | 3 | `form_score.go` | `SquadService.GetFormScoreHistory` | §3.6 |
| `GET /squad/impact?xuids=...&matchIds=...` | 4 | `squad_impact.go` | `SquadService.GetImpact` | §3.7 (8 roles) |
| `GET /squad/history?xuids=...&matchIds=...` | 5 | `squad_history.go` | reutilise existant | §3.8 |
| `GET /squad/first-events?matchIds=...` | 5 | `squad_first_events.go` | `SquadService.GetFirstEvents` | §4.6 |
| `GET /squad/cadence?xuids=...&matchIds=...` | 6 | `squad_cadence.go` | `SquadService.GetCadence` | §3.9 |
| `GET /squad/intensity?xuids=...&matchIds=...` | 6 | `squad_intensity.go` | `SquadService.GetIntensity` | §4.5 |
| `GET /squad/radar?xuids=...&matchIds=...` | 8 | `squad_radar.go` | `SquadService.GetRadarProfiles` | §4.2 |
| `GET /squad/weapons?xuid=...&matchIds=...` | 9 | `squad_weapons.go` | `SquadService.GetWeapons` | §4.7, §4.8 |
| `GET /squad/medals?xuids=...&matchIds=...` | 9 | `squad_medals.go` | `SquadService.GetMedals` | §4.9 |

NB : certains de ces endpoints peuvent etre fusionnes dans `TeammatesPageResponse` existant pour reduire les round-trips (a decider au moment de l'implementation, conserver des handlers individuels pour la testabilite).

---

## 6. Estimation par phase

| Phase | Sections | Cout | Pre-requis |
|-------|----------|-----:|------------|
| Phase 0 — Briques transverses | helpers algos + domain + port + repo + UI | 7.5 j | A faire en premier |
| **Phase 0bis — Specification visuelle** | SQUAD_VISUAL_SPEC + DESIGN_TOKENS + CHART_HELPERS | **3 j** | **En parallele Phase 0, terminee avant Phase 1** |
| Phase 1 — En-tetes | §2.1, §2.2 | 2.5 j | Phase 0 (perf score v2) + **Phase 0bis** |
| Phase 2 — Synergies "carte" | §3.1, §3.2, §3.3, §3.4 | 4 j | `ComputeMapBreakdown` |
| Phase 3 — Timeline + Form Score | §3.5, §3.6 | 3 j | LOWESS |
| Phase 4 — Impact | §3.7 | 3 j | Extension a 8 roles |
| Phase 5 — Tableau + First Events | §3.8, §4.6 | 2 j | — |
| Phase 6 — Cadence + Intensite | §3.9, §4.5 | 3 j | `LoadKillTimingForMatches` |
| Phase 7 — Charts trio | §4.1, §4.3, §4.4 | 5 j | `ComputeSquadRecords` + `RecordOverlay` |
| Phase 8 — Radar | §4.2 | 2 j | `ComputeParticipationProfile` |
| Phase 9 — Armes + Medailles | §4.7, §4.8, §4.9 | 4.5 j | Repo armes + medailles |
| **Total** | 22 charts + briques + spec | **~39 j** | |

**MVP "good enough" ~25 j** : Phase 0 (sans LOWESS) + Phase 0bis (allegee : juste DESIGN_TOKENS + CHART_HELPERS, pas le SPEC complet) + Phases 1, 2, 4 (en restant a 4 roles), 5, 7 (sans records overlays), 9 (medailles seulement). Ramene la fidelite visuelle a ~80 %.

---

## 7. Suivi

| Phase | Statut | Branche | PR | Note |
|-------|--------|---------|----|----|
| 0 — Briques transverses | A faire | feat/squad-page-portage | — | LOWESS impl/wrapper a trancher |
| **0bis — Spec visuelle** | A faire | feat/squad-page-portage | — | Doit etre fini avant Phase 1 |
| 1 — En-tetes | A faire | feat/squad-page-portage | — | Depend Phase 0bis |
| 2 — Synergies carte | A faire | feat/squad-page-portage | — | |
| 3 — Timeline + Form Score | A faire | feat/squad-page-portage | — | Depend Phase 0 LOWESS |
| 4 — Impact | A faire | feat/squad-page-portage | — | Depend Phase 0 ImpactV2 |
| 5 — Tableau + First Events | A faire | feat/squad-page-portage | — | |
| 6 — Cadence + Intensite | A faire | feat/squad-page-portage | — | Depend Phase 0 KillTiming |
| 7 — Charts trio | A faire | feat/squad-page-portage | — | Depend Phase 0 Records + UI |
| 8 — Radar | A faire | feat/squad-page-portage | — | Depend Phase 0 ParticipationProfile |
| 9 — Armes + Medailles | A faire | feat/squad-page-portage | — | |

---

## 8. References

- Audit complet : [docs/AUDIT_TEAMMATES_V7_COCKPIT.md](../docs/AUDIT_TEAMMATES_V7_COCKPIT.md)
- Plan match view (modele de format) : [.ai/PLAN_MATCH_VIEW_GO_PORTAGE.md](PLAN_MATCH_VIEW_GO_PORTAGE.md)
- Skill `arch-rules` : separation des couches Go.
- Skill `frontend-patterns` : conventions React/TypeScript apps/web.
- Skill `delivery-checklist` : grille go/no-go avant chaque commit/PR.
- Source Python : branche `v7/cockpit`, modules `src/ui/pages/teammates*.py`, `src/data/services/teammates_service.py`, `src/analysis/{friends_impact,squad_records,match_intensity,_performance_form,_performance_squad,participation_radar}.py`, `src/visualization/{squad_*,teammates_*,match_intensity_heatmap,trio,_form_score}.py`.
- Cible Go : `apps/go-api/internal/{analysis,domain,port,service,api/handlers,platform/duckdb}/` + `apps/web/src/features/squad/`.
