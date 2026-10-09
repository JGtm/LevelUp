# Audit de Parité : Go API vs FastAPI — Comparaison Exhaustive

> Date : 2026-04-16 — Lecture complète de chaque fichier handler Go + chaque schema Pydantic Python
>
> Remplace l'audit narratif du 2026-04-15 par une comparaison champ-par-champ.

---

## 1. Table de Correspondance des Routes

| # | FastAPI Route (sous `/api/v1`) | Méthode | Go Route (sous `/api/v1` sauf health) | Méthode | Status | Écarts |
|---|---|---|---|---|---|---|
| 1 | `/health` | GET | `/health` (hors `/api/v1`) | GET | ⚠️ | Préfixe `/api/v1` manquant côté Go — casse les healthchecks Docker |
| 2 | `/bootstrap` | GET | `/bootstrap` | GET | ✅ | |
| 3 | `/players` | GET | `/players` | GET | ✅ | |
| 4 | `/session/context` | POST | `/session/context` | POST | ✅ | |
| 5 | `/auth/device-flow/start` | POST | `/auth/device-flow/start` | POST | ✅ | |
| 6 | `/auth/device-flow/{attempt_id}` | GET | `/auth/device-flow/{attempt_id}` | GET | ✅ | |
| 7 | `/settings` | GET | `/settings` | GET | ✅ | |
| 8 | `/settings` | PATCH | `/settings` | PATCH | ✅ | |
| 9 | `/settings/media/reset-index` | POST | `/settings/media/reset-index` | POST | ✅ | Go: stub (TODO Sprint 19) |
| 10 | `/setup/players` | POST | `/setup/players` | POST | ✅ | |
| 11 | — | — | `/setup/smoke-test` | POST | 🟦 GO-ONLY | Absent en FastAPI |
| 12 | `/jobs/{job_id}` | GET | `/jobs/{job_id}` | GET | ✅ | |
| 13 | `/sync/initial` | POST | `/sync/initial` | POST | ✅ | |
| 14 | `/players/{s}/filters/resolve` | POST | `/players/{s}/filters/resolve` | POST | ✅ | |
| 15 | `/players/{s}/pages/match-history/query` | POST | `/players/{s}/pages/match-history/query` | POST | ⚠️ | Champ `columns` absent Go — voir §3.8 |
| 16 | `/players/{s}/pages/match-history/export` | POST | — | — | ❌ ABSENT | Export CSV non implémenté |
| 17 | `/players/{s}/pages/career` | GET | `/players/{s}/pages/career` | GET | ✅ | |
| 18 | `/players/{s}/pages/career/top-matches` | GET | `/players/{s}/pages/career/top-matches` | GET | ✅ | |
| 19 | `/players/{s}/pages/career/encounters` | GET | `/players/{s}/pages/career/encounters` | GET | ✅ | |
| 20 | `/players/{s}/pages/citations` | **POST** | `/players/{s}/pages/citations` | **GET** | ⚠️ MÉTHODE | Python=POST+body filtres, Go=GET sans body |
| 21 | — | — | `/players/{s}/pages/commendations` | GET | 🟦 GO-ONLY | Python fusionne dans citations |
| 22 | `/players/{s}/matches/{match_id}` | GET | `/players/{s}/matches/{match_id}` | GET | ⚠️ | Réponse: 13+ colonnes scoreboard manquantes Go |
| 23 | `/players/{s}/pages/explorer/matches-query` | POST | — | — | ❌ ABSENT | Explorer filtré non implémenté |
| 24 | `/players/{s}/pages/explorer/player-query` | POST | `/players/{s}/pages/explorer/player-query` | POST | ⚠️ DIFFS | Body+réponse divergent — voir §3.9 |
| 25 | `/players/{s}/pages/last-match/resolve` | POST | — | — | ❌ ABSENT | |
| 26 | — | — | `/players/{s}/pages/sessions` | GET | 🟦 GO-ONLY | FastAPI n'a pas d'endpoint sessions dédié |
| 27 | `/players/{s}/pages/timeseries` | POST | — | — | ❌ ABSENT | Remplacé par stats/query (incompatible) |
| 28 | — | — | `/players/{s}/pages/stats/query` | POST | 🟦 GO-ONLY | Contrat totalement différent de timeseries |
| 29 | `/players/{s}/pages/session-compare` | POST | — | — | ❌ ABSENT | |
| 30 | `/players/{s}/pages/home` | GET | `/players/{s}/pages/home` | GET | ✅ | |
| 31 | `/players/{s}/battlepass` | GET | `/players/{s}/battlepass` | GET | ✅ | |
| 32 | `/players/{s}/challenges` | GET | `/players/{s}/challenges` | GET | ✅ | |
| 33 | `/players/{s}/pages/teammates` | **POST** | `/players/{s}/pages/squad` | **GET** | ⚠️ ROUTE+MÉTHODE | Route renommée + POST→GET + contrat diff |
| 34 | `/players/{s}/pages/synthesis` | **POST** | `/players/{s}/pages/synthesis` | **GET** | ⚠️ MÉTHODE | POST→GET, body filtres perdu |
| 35 | `/players/{s}/pages/media` | **POST** | `/players/{s}/pages/media` | **GET** | ⚠️ MÉTHODE+CONTRAT | POST body filtres/tri → GET ?page=N |
| 36 | `/directory/gamertags/search` | GET | `/directory/gamertags/search` | GET | ⚠️ | Go ne set pas `query` dans la réponse |

---

## 2. Résumé Comptable

| Métrique | Valeur |
|---|---|
| Endpoints FastAPI total | **28** |
| Endpoints Go total | **27** |
| Routes identiques (route + méthode + contrat conforme) | **16** |
| Routes avec méthode HTTP différente | **4** (citations, teammates/squad, synthesis, media) |
| Endpoints absents en Go | **5** (export, matches-query, last-match, session-compare, timeseries) |
| Endpoints Go-only | **4** (smoke-test, commendations, sessions, stats/query) |
| Réponses appauvries en Go | **5** (citations, synthesis, media, explorer/player-query, match scoreboard) |
| Réponses enrichies en Go | **1** (squad vs teammates) |
| Handlers avec `http.Error()` plain text | **3** (home, sessions, stats) |
| Handlers avec violation hexagonale | **15/22** |

---

## 3. Différences Champ-par-Champ

### 3.1 `pages/timeseries` (FastAPI) vs `pages/stats/query` (Go) — INCOMPATIBLE

**FastAPI `TimeseriesPageResponse`** :
```
total_matches: int
summary_tab:
  kpi_cards: [{key, label, value, delta, color}]
  win_rate_chart: PlotlyFigurePayload | null
  score_chart: PlotlyFigurePayload | null
  kda_dist_chart: PlotlyFigurePayload | null
cumul_tab:
  cumul_net_chart: PlotlyFigurePayload | null
  cumul_kd_chart: PlotlyFigurePayload | null
  rolling_kd_chart: PlotlyFigurePayload | null
form_tab:
  ewma_kd_chart: PlotlyFigurePayload | null
  regression_chart: PlotlyFigurePayload | null
  net_score_per_hour_chart: PlotlyFigurePayload | null
  regression_stats: {kd_slope, winrate_slope, r_squared, has_enough_for_trend, trend}
intensity_tab:
  intensity_heatmap: PlotlyFigurePayload | null
  score_per_minute_chart: PlotlyFigurePayload | null
distributions_tab:
  kda_distribution: PlotlyFigurePayload | null
  first_kill_dist: PlotlyFigurePayload | null
  correlations: [PlotlyFigurePayload]
```
→ Chaque chart = `PlotlyFigurePayload` (data+layout JSON Plotly, rendu server-side).

**Go `StatsPageResponse`** :
```json
{
  "win_loss": {"points": [WinLossPoint], "win_rate": float, "total_matches": int,
               "rolling_win_rate": [float], "cumulative_kd": [CumulativePoint],
               "cumulative_net": [CumulativePoint]},
  "accuracy": {"points": [AccuracyPoint], "mean": float, "has_data": bool,
               "score_per_min": [float]},
  "objective": {"points": [ObjectivePoint], "total_score": int, "avg_assists": float,
                "has_data": bool},
  "form":     {"points": [PerformancePoint], "mean": *float, "has_enough_data": bool},
  "lusr":     {"points": [LUSRPoint], "current_rating": *float, "has_data": bool},
  "bucket_info": {"type": string, "label": string},
  "total_matches": int
}
```
→ Points de données bruts — **AUCUN Plotly** — le frontend construit les charts.

**Verdict** : ❌ **Contrats radicalement différents** — pas d'adaptation possible sans réécriture.

### 3.2 `pages/teammates` (FastAPI) vs `pages/squad` (Go) — INCOMPATIBLE

**FastAPI `TeammatesPageResponse`** :
```
options: [{gamertag, xuid, encounter_count, last_seen_at}]
teammates: [{gamertag, xuid, encounter_count, last_seen_at,
             with_kpis: {match_count, wins, kd_ratio, win_rate, accuracy,
                         kills_per_game, assists_per_game},
             without_kpis: idem | null}]
solo_reference: TeammateKPIs | null
total_matches: int
```

**Go `SquadPageResponse`** :
```json
{
  "top_teammates": [{xuid, gamertag, games_together, wins_together, win_rate,
                     avg_kda, avg_kills}],
  "selected_teammate": {gamertag, xuid, games_together,
                        squad_score: {score, grade, components},
                        radar_me: {name, color, values},
                        radar_teammate: {name, color, values},
                        impact: {first_bloods, clutch_kills, last_kills, first_deaths,
                                 available},
                        records: {}, timeseries: []},
  "solo_stats":  {match_count, win_rate, avg_kda, avg_kills},
  "squad_stats": {match_count, win_rate, avg_kda, avg_kills}
}
```

| Aspect | FastAPI | Go |
|---|---|---|
| Route | `pages/teammates` | `pages/squad` |
| Méthode | POST body `{selected_gamertags[], filters}` | GET query `?teammate=xuid` |
| KPIs comparés | `with_kpis` vs `without_kpis` par coéquipier | solo_stats vs squad_stats global |
| `kd_ratio, accuracy, kills_per_game, assists_per_game` | ✅ | ❌ remplacé par `avg_kda, avg_kills` |
| Squad score / radar / impact / records / timeseries | ❌ | ✅ (Go plus riche) |
| Filtres cascade support | ✅ via FilterContextInput body | ❌ |

### 3.3 `pages/citations` — MÉTHODE + CONTRAT DIFF

**FastAPI `CitationsPageResponse`** (POST avec `FilterContextInput` body) :
```
commendations: [{key, label, category, current_value, color, icon_path,
                 tier_label, mastery_pct}]
medals_summary: [{medal_name_id, name, count_filtered, count_total, description}]
deltas: {filtered_total, unfiltered_total, delta_count}
distribution_chart: PlotlyFigurePayload | null
```

**Go `CitationsPageResponse`** (GET, pas de body) :
```json
{"citations": [{name_norm, name_display, category, total, image_path, description}],
 "categories": ["..."], "total_count": int}
```

**Go `CommendationsPageResponse`** (endpoint séparé GET) :
```json
{"categories": [{category, items: [{medal_id, medal_name, count, category,
                                     image_path}], total}],
 "total_count": int}
```

| Champ | FastAPI | Go |
|---|---|---|
| Fusion commendations+médailles | 1 endpoint | 2 endpoints séparés |
| `count_filtered` vs `count_total` | ✅ (filtres appliqués) | ❌ 1 seul `total` |
| `deltas` | ✅ | ❌ ABSENT |
| `distribution_chart` | PlotlyFigurePayload | ❌ ABSENT |
| `mastery_pct`, `tier_label` | ✅ | ❌ ABSENT |
| Filtres cascade | ✅ POST body | ❌ Aucun |

### 3.4 `pages/synthesis` — MÉTHODE + CONTRAT DIFF

**FastAPI `SynthesisPageResponse`** (POST avec body `{period, filters}`) :
```
period: str, total_matches: int
solo_kpis: {match_count, wins, kd_ratio, win_rate, accuracy,
            kills_per_min, avg_life_seconds, performance_score}
squad_kpis: idem
comparison_metrics: [{label, solo_value, squad_value, solo_text, squad_text}]
heatmap_data: [{dow: int, hour: int, count: int}]
top_weeks: [{week_label, match_count, win_rate, kd_ratio}]
```

**Go `SynthesisPageResponse`** (GET, pas de body) :
```json
{"heatmap_data": [{row_key, col_key, value, count}],
 "top_weeks": [{week_label, win_rate, avg_kills, match_count}],
 "total_matches": int, "overall_win_rate": float}
```

| Champ | FastAPI | Go |
|---|---|---|
| `solo_kpis` + `squad_kpis` (8 champs chacun) | ✅ | ❌ ABSENT |
| `comparison_metrics[]` | ✅ | ❌ ABSENT |
| `period` filtrable | ✅ | ❌ ABSENT |
| Heatmap format | `{dow, hour, count}` (temporel) | `{row_key, col_key, value, count}` (map×mode) |
| `top_weeks.kd_ratio` | ✅ | ❌ `avg_kills` à la place |

### 3.5 `pages/media` — MÉTHODE + CONTRAT DIFF

**FastAPI `MediaPageResponse`** (POST avec body `MediaQueryRequest`) :
```
items: PaginatedResponse[MediaItemRow]
  → items[]: {basename, file_path, kind, thumbnail_path, match_id,
              capture_end_utc, match_start_time, section, owner_gamertag, map_name}
  → pagination: {total, page, page_size, has_next, has_prev}
total_mine: int, total_teammates: int, total_unassigned: int
```
Body entrant : `{sort, kind_filter, section_filter, map_filter, mode_filter, group_by, pagination}`

**Go `MediaPageResponse`** (GET `?page=N`) :
```json
{"items": [{file_name, file_path, kind, thumbnail_path, capture_end_utc,
            match_id, match_start_time}],
 "total_count": int, "page": int, "page_size": int, "has_more": bool}
```

| Champ | FastAPI | Go |
|---|---|---|
| Naming item | `basename` | `file_name` |
| `section` (mine/teammates/unassigned) | ✅ | ❌ ABSENT |
| `owner_gamertag`, `map_name` | ✅ | ❌ ABSENT |
| `total_mine/teammates/unassigned` | ✅ (3 compteurs) | ❌ 1 seul `total_count` |
| Sort, kind_filter, group_by | ✅ POST body | ❌ ABSENT |
| Pagination wrapper | `{total, page, page_size, has_next, has_prev}` | `{total_count, page, page_size, has_more}` (flat) |

### 3.6 `pages/home` — CONFORME (avec bug d'erreurs)

Champs JSON identiques entre FastAPI et Go : `hero`, `highlights`, `recent_matches`, `recent_media`, `solo_session`, `squad_session`.

**Bug** : `home.go` utilise `http.Error()` (plain text) au lieu de `writeError()` (JSON structuré) sur 3 paths d'erreur.

### 3.7 `matches/{match_id}` — DIFFS DE COLONNES

**Champs manquants en Go** :

| Champ FastAPI | Présent Go |
|---|---|
| `rank.icon_url` | ❌ |
| `combat_tab.weapon_kills[].effective_weapon_id` | ❌ |
| `combat_tab.highlight_events.event_time_ms` | ❌ (Go: `tick_count`) |
| `combat_tab.highlight_events.target_xuid` | ❌ |
| `combat_tab.highlight_events.weapon_id` | ❌ |
| `team_tab.scoreboard[].is_bot` | ❌ |
| `team_tab.scoreboard[].betrayals` | ❌ |
| `team_tab.scoreboard[].suicides` | ❌ |
| `team_tab.scoreboard[].shots_accuracy` | ❌ |
| `team_tab.scoreboard[].damage_efficiency` | ❌ |
| `team_tab.scoreboard[].average_life` | ❌ |
| `team_tab.scoreboard[].objectives_stolen` | ❌ |
| `team_tab.scoreboard[].headshot_kills` | ❌ |
| `team_tab.scoreboard[].max_killing_spree` | ❌ |
| `team_tab.scoreboard[].perfect_kills` | ❌ |
| `team_tab.scoreboard[].power_weapon_kills` | ❌ |
| `team_tab.scoreboard[].melee_kills` | ❌ |

**13+ colonnes** du scoreboard FastAPI absentes du Go.

### 3.8 `pages/match-history/query`

**Request body** :
| Champ | FastAPI | Go |
|---|---|---|
| `filters` | ✅ FilterContextInput | ✅ |
| `pagination` | `{page, page_size}` | `{page, page_size}` ✅ |
| `columns` | ✅ `list[str] | None` | ❌ ABSENT |
| `include_export_hint` | ✅ `bool` | ✅ |
| Sort | via `PaginationRequest` inherited | `sort_field + sort_dir` (flat) |

**Response** :
| Champ | FastAPI | Go |
|---|---|---|
| `table.items[].outcome_code` | `int | None` | `int` (non-nullable) |
| `table.items[].map_ui` | `str` (non-nullable) | `*string` (nullable) |
| `table.items[].mode_ui` | `str` (non-nullable) | `*string` (nullable) |
| `table.pagination.freshness` | ❌ | ✅ (Go-only) |
| `export_hint` | ✅ | ❌ (export absent) |

### 3.9 `pages/explorer/player-query` — DIFFS MAJEURS

**Request** :
| Champ | FastAPI | Go |
|---|---|---|
| Champ gamertag | `target_gamertag` | `other_gamertag` (**nom différent**) |
| `filters` | ✅ `FilterContextInput | None` | ❌ ABSENT |
| `limit` | ❌ | ✅ `int` (Go-only) |

**Response** :
| FastAPI `ExplorerPlayerQueryResponse` | Go `ExplorerPlayerQueryResponse` |
|---|---|
| `target: {gamertag, xuid}` | `other_gamertag` + `other_xuid` (flat) |
| `summary: {matches_together, wins_together, losses_together, last_seen_at}` | ❌ ABSENT |
| `allies_table: [ExplorerEncounterRow]` | ❌ ABSENT |
| `enemies_table: [ExplorerEncounterRow]` | ❌ ABSENT |
| `common_matches[]: {match_id, start_time, start_time_label, map_ui, mode_ui, playlist_label, outcome_label, score_label, is_with_friends, experience_type_label}` | `common_matches[]: {match_id, start_time, map_ui, mode_ui, were_teammates, player_outcome}` |

→ Réponse Go très simplifiée — pas de summary, pas de allies/enemies, pas de labels formatés.

### 3.10 `directory/gamertags/search`

Struct identique : `{query, items: [{gamertag, xuid, score, exact_match}]}`

**Bug Go** : le handler ne set pas le champ `query` dans la réponse → toujours `""`.

### 3.11 `filters/resolve` — CONFORME

`FilterContextResolved` a la même structure : `effective`, `available_options`, `session_options`, `counts`. ✅

---

## 4. Analyse des Handlers Go — Violations d'Architecture

| Handler | Importe `platform/duckdb` | Appelle `config.ResolvePlayer` | Construit repos inline | `http.Error()` (plain text) |
|---|:---:|:---:|:---:|:---:|
| `auth.go` | ❌ | ❌ | ❌ | ❌ |
| `bootstrap.go` | ❌ | ❌ | ❌ (service injecté) | ❌ |
| `career.go` | ✅ | ✅ | ✅ | ❌ |
| `citations.go` | ✅ | ✅ | ✅ | ❌ |
| `explorer.go` | ✅ | ✅ | ✅ | ❌ |
| `filters.go` | ✅ | ✅ | ✅ | ❌ |
| `gamertag.go` | ✅ (`OpenReadOnly` + `NewGamertagRepo`) | ❌ (`config.SharedDBPath`) | ✅ | ❌ |
| `health.go` | ❌ | ❌ | ❌ (port injecté) | ❌ |
| **`home.go`** | ✅ | ✅ | ✅ | **✅ 3× `http.Error()`** |
| `jobs.go` | ❌ | ❌ | ❌ | ❌ |
| `match_history.go` | ✅ | ✅ | ✅ | ❌ |
| `match_view.go` | ✅ | ✅ | ✅ | ❌ |
| `media.go` | ✅ | ✅ | ✅ | ❌ |
| **`sessions.go`** | ✅ | ✅ | ✅ | **✅ 2× `http.Error()`** |
| `settings.go` | ❌ | ❌ | ❌ | ❌ |
| `setup.go` | ❌ | ❌ | ❌ | ❌ |
| `squad.go` | ✅ | ✅ | ✅ | ❌ |
| **`stats.go`** | ✅ | ✅ | ✅ | **✅ 2× `http.Error()`** |
| `sync_handler.go` | ❌ | ❌ | ❌ | ❌ |
| `session_context.go` | ❌ | ❌ | ❌ | ❌ |
| `helpers.go` | — | — | — | — |

### Résumé violations

- **15/22 handlers** importent `internal/platform/duckdb` directement
- **14/22 handlers** appellent `config.ResolvePlayer` directement  
- **14/22 handlers** construisent repos + services inline
- **3 handlers** utilisent `http.Error()` (plain text) : `home.go`, `sessions.go`, `stats.go`

### Handlers correctement injectés (7/22)
`auth.go`, `bootstrap.go`, `health.go`, `jobs.go`, `settings.go`, `setup.go`, `sync_handler.go`, `session_context.go`

---

## 5. Priorités de Correction

### P0 — Ruptures de contrat transport
1. `home.go`, `sessions.go`, `stats.go` → remplacer `http.Error()` par `writeError()`
2. `gamertag.go` → set `query` dans `GamertagSearchResponse`

### P0 — Méthodes HTTP incompatibles avec le frontend
3. `citations` → accepter POST avec FilterContextInput body
4. `synthesis` → accepter POST avec SynthesisQueryRequest body
5. `media` → accepter POST avec MediaQueryRequest body
6. `teammates/squad` → réaligner route `pages/teammates` + POST body

### P1 — Endpoints absents bloquant la bascule
7. `POST .../pages/session-compare`
8. `POST .../pages/match-history/export`
9. `POST .../pages/explorer/matches-query`
10. `POST .../pages/last-match/resolve`
11. `POST .../pages/timeseries` (ou réaligner le frontend)

### P1 — Réponses appauvries
12. `media` → ajouter section, owner_gamertag, map_name, totaux par section, filtrage
13. `citations` → ajouter deltas, count_filtered/count_total, mastery_pct, tier_label
14. `synthesis` → ajouter solo_kpis, squad_kpis, comparison_metrics
15. `explorer/player-query` → renommer `other_gamertag`→`target_gamertag`, ajouter summary, allies/enemies
16. `match_view` scoreboard → ajouter 13 colonnes manquantes

### P2 — Architecture
17. Extraire `ResolvePlayer` + repo/service construction dans un middleware/factory
18. `health` → ajouter sous `/api/v1/health` en plus de `/health`

---

## Méthodologie

Tous les fichiers suivants lus intégralement (pas d'extraits) :

**Go** : `internal/api/server.go`, `internal/api/handlers/*.go` (22 fichiers), `internal/domain/{squad,stats,sessions,home,media,citations,match_view,filters,match_history,explorer,career}.go`

**Python** : `apps/api/app/main.py`, `apps/api/app/routers/*.py` (16 fichiers), `apps/api/app/schemas/*.py` (17 fichiers)

Audit statique — aucune exécution runtime.
# Audit de parité Python -> Go et qualité d'architecture

Date : 2026-04-16

## Portée de l'audit

Cet audit consolide deux angles d'analyse complémentaires :

- la parité produit entre le legacy Python/Streamlit, le frontend React actuel, le backend FastAPI transitoire et le backend Go cible ;
- la qualité réelle du code Go, au-delà du simple fait que des packages et des services existent.

Le point de vigilance principal reste le même : un backend Go peut être techniquement déjà bien avancé tout en restant non substituable s'il ne sert pas le contrat réellement consommé par le frontend React.

## Résumé exécutif

1. Le portage Go est substantiel sur le noyau technique.
   Le moteur de sync, les migrations DuckDB, le Device Code Flow, l'échange Halo, une partie importante de l'analyse métier et les opérations CLI existent réellement côté Go. Le projet n'est pas un simple squelette.

2. Ce portage ne suffit pas encore à remplacer la chaîne Python en production.
   Docker, compose, Makefile de développement et génération de types frontend restent arrimés à FastAPI. La cible d'exécution effective n'est donc pas Go aujourd'hui.

3. Le vrai blocage est le contrat produit.
   Le frontend React appelle plusieurs routes absentes ou incompatibles côté Go. Il existe même déjà un drift entre le frontend et les backends inspectés, par exemple sur GET /setup/status.

4. L'onboarding Go reste incohérent de bout en bout.
   L'authentification Microsoft/Halo est partiellement portée, mais bootstrap, provisioning de l'identité Halo et création de profil ne sont pas alignés.

5. L'architecture Go est bonne au niveau des packages, mais encore incomplète dans les dépendances réelles.
   Les handlers composent encore directement repos et services, importent l'infrastructure, et dupliquent le pattern resolvePlayer + NewRepo + NewService.

6. La deuxième passe fait remonter plusieurs défauts internes concrets à ajouter au diagnostic.
   Fuite potentielle de connexions dans le pool DuckDB, concaténation SQL dans le backfill, erreurs silencieusement ignorées dans MatchView, fichiers trop gros, logique métier dans setup.go et divergence de contrat d'erreur HTTP.

## Ce que la seconde passe confirme côté Go

Cette partie corrige deux lectures trop pessimistes rencontrées pendant la revue : certains sous-systèmes existent bel et bien, mais leur intégration produit reste incomplète.

| Domaine | Etat après relecture | Nuance importante |
|---------|----------------------|-------------------|
| Sync/backfill | largement porté | le coeur existe, mais cela ne garantit pas la parité du produit web |
| Migrations DuckDB | réellement portées | framework Go crédible et déjà alimenté |
| Device Code Flow + échange Halo | présents | la faiblesse porte sur la persistance et le provisioning, pas sur l'absence totale d'auth |
| MSAL silent helper | présent | AcquireTokenSilent et sérialisation mémoire existent, mais le cache n'est pas persisté dans sync_meta |
| Weapon parser | présent | les briques d'analyse weapon_parser / correlation / reconciliation existent, mais cela ne prouve pas que toute la chaîne legacy est branchée sur les bons flux |
| Discord / CLI / ops | largement portés | bonne couverture technique, mais hors du chemin critique de bascule React -> Go |
| Analyses UI avancées | partielles | le manque est tolérable tant que le frontend React ne les consomme pas |

Conclusion intermédiaire : le constat principal n'est pas "Go est vide", mais "Go est riche techniquement sans être encore prêt comme backend de référence du produit actuel".

## Constats critiques de parité produit

### 1. La migration n'est pas effectivement basculée sur Go

Constat : la documentation de pilotage présente un état plus avancé que la chaîne runtime réelle.

Preuves :

- Dockerfile démarre encore un runtime Python avec uvicorn sur apps/api ;
- docker-compose.yml reste câblé sur un healthcheck FastAPI ;
- Makefile garde les cibles api, dev et generate-types côté Python ;
- apps/web/package.json génère toujours les types depuis l'OpenAPI FastAPI locale.

Impact :

- le backend Go n'est pas la cible réellement déployée ;
- la dette de double maintenance Python + Go reste entière ;
- parler de migration terminée est prématuré.

### 2. Le frontend React n'est pas servi par un contrat Go compatible

Constat : plusieurs routes utilisées par apps/web n'existent pas côté Go, ou existent avec une autre méthode, un autre path ou un autre DTO.

#### Matrice des écarts bloquants

| Surface | Contrat consommé par apps/web | Go actuel | Effet |
|---------|-------------------------------|-----------|-------|
| Setup status | GET /setup/status | absent | le frontend et generated.ts référencent une route sans implémentation trouvée ni en FastAPI ni en Go |
| Health infra | GET /api/v1/health | GET /health | healthchecks à reprendre si on bascule le runtime |
| Explorer matchs | POST /players/{slug}/pages/explorer/matches-query | absent | vue Explorer incomplète |
| Last match resolve | POST /players/{slug}/pages/last-match/resolve | absent | page Dernier match non fonctionnelle |
| Session compare | POST /players/{slug}/pages/session-compare | absent | route sessions du frontend non alimentée |
| Match history export | POST /players/{slug}/pages/match-history/export | absent | export CSV perdu |
| Timeseries | POST /players/{slug}/pages/timeseries | POST /players/{slug}/pages/stats/query | route et DTO différents |
| Squad | POST /players/{slug}/pages/teammates | GET /players/{slug}/pages/squad | route, méthode et shape différents |
| Synthesis | POST /players/{slug}/pages/synthesis | GET /players/{slug}/pages/synthesis | méthode et payload différents |
| Media | POST /players/{slug}/pages/media | GET /players/{slug}/pages/media | méthode et payload différents |
| Citations | POST /players/{slug}/pages/citations | GET /players/{slug}/pages/citations | filtres POST perdus |
| Explorer player query | body = target_gamertag | body Go = other_gamertag | incompatibilité JSON malgré une route existante |

Le point le plus important est celui-ci : même quand une feature existe côté Go, elle n'est pas forcément branchée sur la bonne abstraction produit.

### 3. Certaines pages Go sont portées contre un contrat différent de celui du frontend

Exemples les plus nets :

- Stats : le frontend attend une page Timeseries avec summary_tab, cumulative_tab, form_tab, intensity_tab et distributions_tab ; le Go expose une page stats/query orientée win_loss, accuracy, objective, form et lusr.
- Squad : le frontend attend teammates, options, solo_reference et total_matches ; le Go renvoie une réponse centrée sur top_teammates, selected_teammate, solo_stats et squad_stats.
- Synthesis : le frontend attend solo_kpis, squad_kpis, comparison_metrics et heatmap_data ; le Go ne renvoie qu'une synthèse plus courte centrée sur heatmap, top_weeks, total_matches et overall_win_rate.
- Media : le frontend attend une pagination et des totaux mine / teammates / unassigned ; le Go reste plus proche d'une galerie legacy.

Conclusion : le portage Go n'est pas seulement incomplet, il est parfois aligné sur une autre cible fonctionnelle que celle d'apps/web.

### 4. L'onboarding Go n'est pas bouclé de bout en bout

Constat : le Device Code Flow et la chaîne d'échange Halo existent, mais le wizard complet n'est pas cohérent.

Chaîne observée :

- pollDeviceFlow passe la tentative à authorized et stocke les tokens Halo, mais ne renseigne ni Gamertag ni XUID ;
- GetDeviceFlowStatus ne copie linked_halo_identity en session que si snapshot.Gamertag est présent ;
- AttemptStore prévoit un état provisioned, mais aucune preuve trouvée d'un passage effectif à cet état avec Gamertag/XUID ;
- BootstrapService.Build renvoie encore AuthState = missing en dur ;
- resolveSetupState ne regarde pas l'état réel du sync initial ;
- SetupHandler.CreatePlayer exige une identité Halo liée pour le mode xbox.

Conséquence pratique :

- le login peut sembler réussi techniquement ;
- le frontend ne reçoit pas un état de bootstrap fidèle ;
- la création de profil dépend d'une identité liée qui n'est pas correctement provisionnée.

Ce n'est pas un simple TODO. C'est un défaut de cohérence applicative dans le parcours d'entrée.

## Qualité d'architecture et de code Go

### 5. L'hexagone existe dans les packages, mais pas encore dans le wiring réel

Le socle structurel est bon : domain, port, service, platform, api et cmd existent et sont lisibles.

En revanche, plusieurs handlers violent encore les règles formelles du repo :

- ils appellent config.ResolvePlayer ;
- ils importent directement internal/platform/duckdb ;
- ils instancient eux-mêmes duckdb.New...Repo(...) puis service.New...Service(...).

Le pattern est répétitif dans :

- career.go ;
- citations.go ;
- explorer.go ;
- home.go ;
- match_history.go ;
- match_view.go ;
- media.go ;
- sessions.go ;
- squad.go ;
- stats.go ;
- filters.go.

Impact :

- couplage direct transport -> infrastructure ;
- duplication du wiring ;
- testabilité plus faible ;
- difficulté à déplacer le composition root au bon niveau.

### 6. Il y a trop de sources de vérité pour le contrat API

Le contrat est aujourd'hui éclaté entre :

- FastAPI/Pydantic ;
- generated.ts côté web ;
- types TypeScript encore maintenus à la main ;
- structs Go ;
- routes réellement exposées par internal/api/server.go.

Symptômes concrets :

- apps/web/src/lib/api/types.ts documente encore des types manuels ;
- package.json continue à générer des types depuis FastAPI ;
- generated.ts et les MSW handlers web exposent /setup/status ;
- aucune implémentation de /setup/status n'a été trouvée ni dans apps/api ni dans apps/go-api.

Ce point est central : il existe déjà du drift avant même de parler de la seule parité Go.

### 7. La robustesse transport est encore inégale

Le client React attend des erreurs JSON structurées. Le backend Go dispose bien d'un helper writeError, mais plusieurs handlers utilisent encore http.Error :

- home.go ;
- sessions.go ;
- stats.go.

Autres constats transport :

- StatsHandler accepte un JSON malformé et retombe silencieusement sur des defaults win_loss / period ;
- plusieurs réponses JSON ignorent l'erreur d'Encode via nolint:errcheck ou affectation muette.

Impact :

- erreurs moins exploitables côté frontend ;
- comportements silencieux qui masquent les défauts de contrat ;
- résilience client/serveur dégradée.

### 8. Trois défauts internes concrets méritent d'être remontés au niveau critique

#### 8.1 Pool DuckDB : fermeture incomplète des doublons et du shutdown

Dans pool.go :

- si deux goroutines ouvrent le même PlayerDB, le doublon fermé ne ferme que pdb.Player ;
- Shared et Metadata du doublon ne sont pas fermées ;
- CloseAll() ne ferme aussi que Player.

Risque : fuite de connexions ou d'handles, surtout sur une charge réelle multi-requêtes.

#### 8.2 Backfill : concaténation SQL dans playerDoneGuard

Dans backfill.go, la clause NOT IN est construite en concaténant les match_id entre quotes.

Le risque pratique est limité si les match_id restent maîtrisés, mais le pattern reste incorrect et évitable.

#### 8.3 MatchView : sept erreurs de repository sont ignorées

Dans match_view_service.go, les appels à :

- GetPlayerMatchStats ;
- GetMatchEnrichment ;
- GetMatchScoreboard ;
- GetMatchMedals ;
- GetMatchEvents ;
- GetMatchWeaponKills ;
- GetMatchKVPairs

sont tous évalués avec _, sans log ni warning.

Conséquence : un match partiellement cassé peut être servi au client sans diagnostic exploitable.

### 9. Quelques dettes de maintenabilité sont réelles et méritent d'être priorisées

#### 9.1 Fichiers trop gros

Mesure vérifiée lors de cette seconde passe :

| Fichier | Lignes |
|---------|--------|
| internal/analysis/squad.go | 812 |
| internal/sync/skill_rating.go | 731 |
| internal/platform/duckdb/queries.go | 714 |
| internal/sync/transforms.go | 570 |
| cmd/levelup/main.go | 532 |

Ces dépassements ne sont pas cosmétiques. Ils concentrent des responsabilités qui devraient être séparées.

#### 9.2 Mélange des couches dans setup.go et skill_rating.go

- setup.go garde createPlayerInProfiles, qui lit, fusionne et réécrit db_profiles.json directement depuis le handler ;
- skill_rating.go mélange encore de l'algorithme et des requêtes SQL.

Cela va dans le sens inverse des règles du repo sur la séparation transport / service / repository / analysis.

#### 9.3 Duplication évitable dans feature_flags.go

BackendFor() et applyFlagsMap() ré-encodent la même table de correspondance surface -> backend dans deux switchs séparés.

Ce n'est pas critique à court terme, mais c'est un signal clair qu'une partie du câblage est encore manuelle.

## Ce que je ne considère pas comme une régression problématique

Je ne considère pas comme des défauts bloquants :

- le remplacement de Streamlit par React/TanStack Router ;
- la redistribution des anciennes pages vers des routes plus explicites ;
- l'existence temporaire de scripts Python ou de surfaces techniques encore non utilisées par apps/web ;
- l'absence de certains algorithmes purement UI tant que le frontend actuel ne les consomme pas.

Le problème commence quand le statut projet laisse entendre que l'application Go remplace déjà le backend réel.

## Points positifs à conserver dans le diagnostic

L'audit reste volontairement sévère, mais il ne doit pas être injuste.

Points positifs confirmés :

- le backend Go a une vraie structure modulaire ;
- les packages domain, port, service, platform et api couvrent déjà une part significative du métier ;
- le socle sync/migrations/ops n'est pas anecdotique ;
- l'effort de portage est crédible et sérieux ;
- la parité de surface produit côté navigation React est globalement bien avancée.

Autrement dit : la base est sérieuse, mais la dernière ligne droite n'est pas une formalité. Elle demande un réalignement de contrat, un nettoyage de composition et une fermeture de plusieurs faux-semblants runtime.

## Verdict

### Parité fonctionnelle

Verdict : parité de surface globalement bonne, mais backend Go non substituable au backend courant.

Le produit visé couvre déjà l'essentiel du legacy Python. En revanche, le backend Go ne sert pas encore le contrat réellement attendu par le frontend React.

### Qualité d'architecture

Verdict : socle prometteur, qualité d'implémentation encore inégale.

Le design cible est bon. Les principaux écarts portent sur :

- le couplage handler -> config/platform ;
- le wiring répété ;
- les contrats API multi-sources ;
- l'onboarding non bouclé ;
- plusieurs défauts concrets de robustesse interne.

## Recommandations prioritaires

1. Fixer une source de vérité unique pour le contrat produit.
   Figer l'OpenAPI cible, puis réaligner apps/web, FastAPI transitoire et Go dessus.

2. Corriger l'onboarding Go avant toute bascule.
   AuthState, provisioning Gamertag/XUID, setup state et create player doivent raconter la même histoire applicative.

3. Fermer les écarts bloquants de routes, méthodes et DTO.
   Priorité : setup/status, last-match/resolve, session-compare, explorer/matches-query, export history, timeseries, teammates, synthesis, media, citations.

4. Déplacer la composition hors des handlers.
   Les handlers doivent recevoir des services, factories ou ports injectés, pas construire eux-mêmes leur infrastructure.

5. Corriger les défauts internes les plus risqués.
   Pool DuckDB, concaténation SQL du backfill, erreurs silencieuses de MatchView, fallback silencieux de StatsHandler.

6. Réduire la dette de maintenabilité avant d'ajouter plus de surfaces.
   Découper les gros fichiers, sortir la logique métier de setup.go et séparer SQL et algorithmes dans skill_rating.go.

7. Réviser la documentation de pilotage.
   Tant que Docker, compose, Makefile et la génération de types restent branchés sur Python, le statut global ne doit pas être formulé comme une migration totalement terminée.

## Méthodologie

Fichiers principalement inspectés pour cette version consolidée :

- legacy Python : streamlit_app.py, src/app/page_router.py, src/ui/pages/v7_sections.py ;
- runtime transitoire : Dockerfile, docker-compose.yml, Makefile, apps/api/app/main.py ;
- contrats frontend : apps/web/src/lib/api/client.ts, generated.ts, types.ts, features/*/queries.ts, test/handlers.ts ;
- backend Go : cmd/server/main.go, internal/api/server.go, internal/api/handlers/*.go, internal/service/bootstrap_service.go, internal/service/match_view_service.go, internal/platform/auth/msal_client.go, internal/platform/duckdb/pool.go, internal/sync/backfill.go, internal/analysis/weapon_parser.go, internal/config/feature_flags.go ;
- gouvernance de migration : MATRIX.md, SPRINT_ROADMAP.md, GO_ARCHITECTURE_RULES.md.

Note : audit statique uniquement. Aucune suite de tests, aucun build end-to-end et aucune validation navigateur n'ont été exécutés pendant cette passe consolidée.
