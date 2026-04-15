# API_CONTRACTS.md — Contrats API détaillés

> Schémas, endpoints, stores front et query keys pour les slices 0 à 8 (toutes sections V7).
> Les slices 0–4 sont détaillés ; les slices 5–8 sont en état placeholder (endpoints identifiés, schémas à figer).
> Source : PLAN_MIGRATION_FASTAPI_REACT.md § Étape critique 3 + Backlog d’implémentation exécutable
> **Aligné sur les sections V7 réelles** — voir [FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) et [SLICES.md](migration/SLICES.md) pour la correspondance.

---

## Cadre de l'étape critique 3

> L'objectif n'est pas de lister des endpoints pour le principe. C'est de figer la frontière entre le backend Python et la future UI React afin d'éliminer les dépendances implicites à Streamlit avant de lancer l'implémentation.

### Principes de conception à figer avant implémentation

**1. Versioning et base path**
- Base path initial : `/api/v1`
- Aucune route front React ne doit parler directement à DuckDB ni connaître un chemin de DB
- Les routes React sont URL-first, les endpoints sont player-scoped quand ils dépendent du joueur courant

**2. Deux familles d'endpoints**
- Endpoints transverses : bootstrap, auth, setup, settings, jobs, résolution des filtres
- Endpoints page-oriented : payloads agrégés par écran pour minimiser l'orchestration front au début

**3. Règle d'agrégation**

On privilégie des endpoints de page relativement gras quand :
- la page existe déjà côté Python comme composition stabilisée
- la page réunit plusieurs sous-sections à valeur métier unique
- le risque de réinterpréter les calculs dans le front est élevé

On privilégie des sous-resources plus fines quand :
- une table doit être paginée/triable côté serveur
- une partie est relancée plus souvent qu'une autre
- une sous-section est lourde et peut être chargée à la demande

**4. Règle de source de vérité**
- Les calculs métier restent côté Python
- Le front consomme soit de la data brute pré-calculée, soit du Plotly JSON
- Aucune dérivation métier critique ne doit vivre seulement dans Zustand ou dans un composant React

**5. Contrat de contexte commun**

Tout endpoint de page MVP doit pouvoir reconstruire l'équivalent des composantes aujourd'hui dérivées dans `PageContext` :
- player context
- filter context effectif
- freshness / provenance des données
- data principale
- actions ou liens associés si nécessaire

### Ce que cette étape ne doit pas faire

- Ne pas transformer l'API en miroir technique des repositories DuckDB
- Ne pas pousser de logique métier critique dans le front sous prétexte de gagner du temps
- Ne pas figer trop tôt des micro-endpoints ultra-fins qui compliquent le MVP sans valeur produit immédiate
- Ne pas coupler les contrats FastAPI à des détails du rendu Streamlit ou à des noms de widgets historiques

### Définition of done pour l'étape 3

L'étape critique 3 est considérée comme couverte si :
- Chaque parcours MVP a au moins un endpoint cible, un schéma d'entrée et un schéma de sortie explicites
- Les conventions transverses de filtres, erreurs, pagination, figures et jobs sont unifiées
- L'auth et la session ont une frontière serveur claire, sans exposition des tokens au navigateur
- Les routes React prioritaires peuvent être branchées sans accéder à `PageContext`, `st.session_state` ou aux query params Streamlit historiques
- Les sections détaillées ci-dessous suffisent à lancer l'implémentation de Slice 0 à Slice 5 sans rediscuter la forme de l'API

---

## Conventions générales

- Base path : `/api/v1`
- Payloads JSON en snake_case
- Dates en ISO-8601 ; datetimes normalisés UTC ou explicitement étiquetés
- Identité joueur portée par `player_slug` dans les routes métier
- Tout composant React lit uniquement des schémas de données explicites ou des figures Plotly sérialisées
- Aucun endpoint n'expose de chemin de base, nom de table, détail de repository ou sémantique liée au worktree

### Transport auth/session et sécurité web

- Le navigateur ne reçoit jamais de token Halo exploitable ; il manipule uniquement une session web opaque.
- Le stockage de session doit être partagé entre workers/processus FastAPI.
- Le cookie de session est `httpOnly`, `Secure` en production et `SameSite=Lax` par défaut.
- Toute route mutante authentifiée par cookie doit être protégée contre CSRF via en-tête dédié ou mécanisme équivalent.
- Le CORS est limité aux origines frontend connues ; aucun wildcard en production.
- `player_slug` dans une route est un identifiant d'adressage. Le backend reste arbitre de l'accès réel à la ressource.

### Locale, timezone et préférences

- Une seule locale active pilote tout le shell et les appels API.
- Ordre de priorité recommandé : choix utilisateur explicite, préférence navigateur persistée, `bootstrap.settings_excerpt.lang`, `Accept-Language`, fallback `fr`.
- Le frontend ne déduit jamais seul une timezone métier ; il consomme la timezone utilisateur renvoyée par `bootstrap` ou `settings`.

### Transport des jobs longs

- Pour le MVP, le transport de progression repose sur polling HTTP via `AsyncJobStatus`.
- SSE ou WebSocket sont des optimisations futures possibles, mais ne conditionnent pas Slice 0/1.
- Toute route qui démarre un job doit retourner immédiatement un identifiant opaque de job ou un `AsyncJobStatus` initial.

### Contrat d'erreurs HTTP et comportement front

> **Chaque type d'erreur doit produire un comportement front déterministe, pas une heuristique.**

| Code HTTP | Catégorie | Comportement front attendu | Retry TanStack Query |
|-----------|-----------|---------------------------|---------------------|
| `400` | Validation / requête invalide | Message utilisateur + highlight du champ si `field_errors` | **Non** |
| `401` | Session expirée / absente | Redirection bootstrap ou login | **Non** — invalider `['bootstrap']` |
| `403` | Accès refusé | Message utilisateur, pas de retry | **Non** |
| `404` | Ressource introuvable | Empty state dédié, pas un spinner infini | **Non** |
| `409` | Conflit (job déjà en cours, lock DuckDB) | Message + bouton retry explicite | **Non** (retry manuel) |
| `422` | Validation Pydantic échouée | Même traitement que `400` via `field_errors` | **Non** |
| `429` | Rate limit (si implémenté) | Retry automatique avec backoff exponentiel | **Oui** — backoff |
| `500` | Erreur serveur inattendue | Message générique + 1 retry automatique | **Oui** — 1 retry, puis abandon |
| `502/503/504` | Infrastructure / timeout | Retry avec backoff, max 3 tentatives | **Oui** — backoff × 3 |

Configuration TanStack Query par défaut :
```typescript
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        if (error.status < 500) return false;  // pas de retry sur 4xx
        return failureCount < 2;                // 1 retry sur 5xx
      },
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10000),
      staleTime: 5 * 60 * 1000,  // 5 min par défaut
    },
  },
});
```

### Comportement en cas de payload composé avec échec partiel

Pour les endpoints page-oriented (`career`, `match-view`) qui assemblent plusieurs sous-sections :

- Si une sous-section échoue mais les autres sont OK, le backend **renvoie quand même un `200`** avec les sections réussies et un champ `partial_errors` dans la réponse.
- Le front affiche les sections disponibles et une alerte non-bloquante pour les sections en erreur.
- L'utilisateur peut relancer manuellement via un bouton de retry par section.

Schéma `partial_errors` (optionnel dans `PageEnvelope`) :
```
partial_errors: list[PartialError] | null

PartialError:
  section: str          # "combat_tab", "media_tab", etc.
  error: ApiError
```

### Loading states et empty states

Chaque page React doit gérer **3 états distincts explicitement**, jamais un seul spinner générique :

| État | Rendu front | Durée max attendue |
|------|-------------|-------------------|
| **Loading initial** | Skeleton screens mimant la structure de la page | < 2s en DEMO_MODE |
| **Loading refresh** | Contenu actuel visible + indicateur discret de rafraîchissement | < 5s |
| **Empty state** | Message contextuel + action suggérée (ex : "Lancez une sync pour voir vos matchs") | — |
| **Error state** | Message d'erreur lisible + code + bouton retry si `retryable: true` | — |

Règle : un spinner plein écran est interdit après le premier chargement d'une page. Le contenu déjà affiché reste visible pendant un refresh.

### Sémantique POST vs GET pour les lectures paginées

Les endpoints `match-history/query`, `explorer/matches-query` et `explorer/player-query` utilisent **POST** car le body `FilterContextInput` est trop complexe pour des query params.

Clarifications :
- Ces routes sont **idempotentes** (lecture pure, pas de mutation). Le POST est un choix de transport, pas de sémantique.
- TanStack Query les traite comme des queries (pas des mutations) : clé de cache basée sur un hash du body.
- Le cache HTTP navigateur ne fonctionnera pas sur ces routes — le cache est entièrement géré par TanStack Query côté front et par `staleTime` / `gcTime`.
- **Alternative future** : si le besoin de cache CDN apparaît, créer un GET avec `?filter_hash=xxx` côté serveur qui résout le hash en paramètres. Pas nécessaire pour le MVP.

### Allègement de PageEnvelope pour les requêtes paginées

`PageEnvelope[T]` complet (meta, player, filters, freshness, capabilities, data) est utilisé pour le **premier chargement** d'une page.

Pour les **requêtes de pagination, tri et refresh** au sein d'une page déjà chargée :
- L'endpoint renvoie uniquement `PaginatedResponse[T]` + un `freshness` léger.
- Les informations stables (`player`, `capabilities`, `filters`) proviennent du cache TanStack Query de `['bootstrap']` et `['filters-resolve']`.
- Cela évite de transférer 1KB+ de contexte redondant à chaque changement de page dans une table.

---

## Schémas transverses

### `ApiMeta`
```
request_id: str
generated_at: datetime ISO-8601
locale: "fr" | "en"
app_version: str
data_version: str | null   # pour invalider les caches front si les contrats changent
```

### `ApiError`
```
code: str
message: str
details: dict | list | null
retryable: bool
field_errors: list[FieldError] | null
```

### `FieldError`
```
field: str
message: str
code: str | null
```

Le front doit pouvoir distinguer sans heuristique :
- erreur fonctionnelle
- erreur de validation
- absence de données
- job en cours
- authentification requise

### `LabelValue`
```
label: str
value: str
disabled: bool | null
count: int | null
```

### `SortSpec`
```
field: str
direction: "asc" | "desc"
```

### `CapabilityMap`
```
can_read_local_data: bool
can_run_sync: bool
can_use_live_halo: bool
can_manage_settings: bool
can_reset_media_index: bool
can_view_media: bool
```

### `PlayerSummary`
```
player_slug: str
gamertag: str
xuid: str
waypoint_player: str
is_demo: bool
```

### `FilterContextInput`
```
filter_mode: "period" | "sessions"
period:
  start_date: date | null
  end_date: date | null
sessions:
  picked_session_label: str | null
  picked_solo_session_label: str | null
  picked_squad_session_label: str | null
  picked_sessions: list[str]
  gap_minutes: int          # invariant : 120 dans le code actuel
cascade:
  experience_types: list[str]
  playlists: list[str]
  modes: list[str]
  maps: list[str]
```

### `FilterContextResolved`
```
effective: FilterContextInput   # normalisé
available_options:
  experience_types: list[LabelValue]
  playlists: list[LabelValue]
  modes: list[LabelValue]
  maps: list[LabelValue]
session_options:
  all_labels: list[str]
  solo_labels: list[str]
  squad_labels: list[str]
counts:
  total_matches_before_filters: int
  total_matches_after_filters: int
```

### `PaginationRequest`
```
page: int >= 1
page_size: int
sort: list[SortSpec]
```

### `PaginatedResponse[T]`
```
items: list[T]
total: int
page: int
page_size: int
total_pages: int
```

### `PlotlyFigurePayload`
```
figure: dict                        # fig.to_plotly_json()
config_key: "clean" | "static"
revision_key: str | null
```

Une figure n'embarque jamais de logique applicative implicite nécessaire au reste de la page.

### `AsyncJobStatus`
```
job_id: str
job_type: "setup_smoke_test" | "sync" | "backfill" | "reindex_media" | "other"
status: "queued" | "running" | "succeeded" | "failed" | "cancelled"
progress_pct: int | null
current_step: str | null
started_at: datetime | null
finished_at: datetime | null
result: dict | null
error: ApiError | null
```

Règles de cycle de vie :
- `job_id` est opaque et scoped à la session ou au joueur qui l'a créé.
- Un job terminal (`succeeded`, `failed`, `cancelled`) reste consultable pendant une fenêtre de rétention explicite côté backend.
- Une route de suivi ne retourne `404` que pour un job inconnu ou expiré, jamais pour un job encore en cours.
- La fin d'un job doit pouvoir être reliée à une invalidation explicite des query keys concernées.

### `PageEnvelope[T]`
```
meta: ApiMeta
player: PlayerSummary
filters: FilterContextInput | null
freshness:
  source: "live" | "cached" | "mixed"
  sync_status: "fresh" | "stale" | "unknown"
  warnings: list[str]
capabilities: CapabilityMap
data: T
```

---

## Contrats Slice 0 — Bootstrap et filtres

### Endpoints

| Méthode | Path | Réponse | Source Python |
|---------|------|---------|---------------|
| GET | `/api/v1/bootstrap` | `BootstrapResponse` | — |
| GET | `/api/v1/players` | `PlayersListResponse` | — |
| POST | `/api/v1/session/context` | `SessionContextResponse` | — |
| POST | `/api/v1/players/{player_slug}/filters/resolve` | `FilterContextResolved` | `filters_render.py`, `filter_state.py` |

### `PlayersListResponse`
```
items: list[PlayerSummary]
default_player_slug: str | null
```

### `SessionContextRequest`
```
player_slug: str | null
locale: "fr" | "en" | null
hints_visible: bool | null
```

### `SessionContextResponse`
```
current_player: PlayerSummary | null
locale: "fr" | "en"
hints_visible: bool
capabilities: CapabilityMap
```

### `BootstrapResponse`
```
setup_required: bool
auth_state: "missing" | "partial" | "ready"
current_player: PlayerSummary | null
available_players: list[PlayerSummary]
locale: "fr" | "en"
hints_visible_default: bool
feature_flags:
  v7_enabled: bool
  media_enabled: bool
  demo_mode: bool
  discord_configured: bool
  tailscale_enabled: bool
capabilities: CapabilityMap
settings_excerpt:
  lang: str
  user_timezone: str
  show_records: bool
  normalize_mode_labels: bool
```

---

## Contrats Slice 1 — Setup / Auth / Settings

### Machine d'état canonique Setup/Auth

Le frontend ne doit pas inférer le setup à partir de bricolages d'UI. Il lit une machine d'état serveur observable.

États minimums à supporter :

- `choose_mode`
- `auth`
- `player`
- `smoke_test`
- `done`
- `failed` si une tentative courante casse et doit être reprise explicitement

Règles :

- Un refresh navigateur en plein Device Code Flow ou pendant un smoke test doit permettre de retrouver l'état courant.
- Une tentative expirée réapparaît comme expirée via `DeviceFlowStatusResponse`, jamais comme simple retour à l'état initial.
- Tant que `next_blocking_step != done`, le shell React bloque l'accès aux routes protégées.

### Endpoints

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| GET | `/api/v1/setup/status` | — | `SetupStatusResponse` | `setup_wizard_logic.py` |
| POST | `/api/v1/auth/device-flow/start` | `DeviceFlowStartRequest` | `DeviceFlowStartResponse` | `auth/provider.py`, `setup_wizard_xbox.py` |
| GET | `/api/v1/auth/device-flow/{attempt_id}` | — | `DeviceFlowStatusResponse` | `auth/provider.py` |
| POST | `/api/v1/setup/players` | `CreatePlayerProfileRequest` | `CreatePlayerProfileResponse` | `setup_wizard_logic.py`, `player_provisioning.py` |
| POST | `/api/v1/setup/smoke-test` | `SmokeTestStartRequest` | `AsyncJobStatus` | `setup_smoke_test_logic.py` |
| GET | `/api/v1/jobs/{job_id}` | — | `AsyncJobStatus` | infrastructure jobs |
| GET | `/api/v1/settings` | — | `SettingsResponse` | `AppSettings`, `load_settings` |
| PATCH | `/api/v1/settings` | `UpdateSettingsRequest` | `SettingsResponse` | `patch_settings`, `_write_settings` |
| POST | `/api/v1/settings/media/reset-index` | `MediaResetRequest` | `AsyncJobStatus` | `MediaIndexer.reset_media_tables` |

### Schémas

**`SetupStatusResponse`**
```
needs_setup: bool
auth:
  has_client_id: bool
  has_refresh_token: bool
  has_msal_cache: bool
  preferred_method: "refresh_token" | "device_code" | "unknown"
player:
  has_any_profile: bool
  default_player_slug: str | null
next_blocking_step: "choose_mode" | "auth" | "player" | "smoke_test" | "done"
```

**`DeviceFlowStartResponse`**
```
attempt_id: str
user_code: str
verification_uri: str
verification_uri_complete: str | null
expires_in_seconds: int
poll_interval_seconds: int
```

**`DeviceFlowStatusResponse`**
```
attempt_id: str
status: "pending" | "authorized" | "provisioned" | "failed" | "expired"
gamertag: str | null
xuid: str | null
error: ApiError | null
```

**`CreatePlayerProfileRequest`**
```
gamertag: str
xuid: str | null
profile_mode: "xbox" | "azure_manual"
```

**`CreatePlayerProfileResponse`**
```
player: PlayerSummary
db_created: bool
warnings: list[str]
```

**`SmokeTestStartRequest`**
```
player_slug: str
max_matches: int
run_backfill: bool
```

**`SettingsResponse` / `UpdateSettingsRequest`** (champs éditables)
```
lang
user_timezone
normalize_mode_labels
show_records
refresh_clears_caches
career_top_exclude_btb
media_captures_base_dir
media_tolerance_minutes
media_watcher_enabled
media_watcher_debounce_seconds
discord_notifications_enabled
discord_webhook_url_present: bool   # lecture seule côté frontend
discord_lang
discord_notify_sync
discord_notify_backfill
discord_notify_new_version
discord_notify_new_media
spnkr_refresh_with_backfill
spnkr_refresh_backfill_medals
spnkr_refresh_backfill_skill
spnkr_refresh_backfill_aliases
spnkr_refresh_backfill_personal_scores
spnkr_refresh_backfill_performance_scores
spnkr_refresh_backfill_lusr
spnkr_refresh_backfill_events
spnkr_refresh_backfill_weapons
```

**`MediaResetRequest`**
```
confirm_destructive: bool
reindex_after_reset: bool
```

### Stores / query keys
- `useSetupFlowStore` : selectedMode, currentAttemptId, currentJobId
- `useSettingsDraftStore` : dirtyFields, lastSavedAt, localUiPrefs (showHints, lastPlayerSlug)
- `['setup-status']`
- `['device-flow', attemptId]`
- `['settings']`
- `['job', jobId]`

### Critères de recette
- Un utilisateur non configuré voit uniquement le flow setup tant que `next_blocking_step != done`
- Le Device Code Flow produit le même résultat fonctionnel que le wizard Streamlit
- La création de profil crée le même `player_slug` et la même base `stats.duckdb` qu'aujourd'hui
- Le smoke test expose les mêmes phases, les mêmes warnings et le même résultat final
- Chaque changement de settings persiste réellement, sans perte au refresh du navigateur

---

## Contrats Slice 2 — Profil [V7 §7] : Carrière (Phase A)

### Endpoints

| Méthode | Path | Réponse | Source Python |
|---------|------|---------|---------------|
| GET | `/api/v1/players/{player_slug}/pages/career` | `CareerPageResponse` | `career.py`, `career_data.py`, `career_logic.py` |
| GET | `/api/v1/players/{player_slug}/pages/career/top-matches` | `CareerTopMatchesResponse` | `career_top_matches_render.py` |
| GET | `/api/v1/players/{player_slug}/pages/career/encounters` | `CareerEncountersResponse` | `career_encounters_render.py` |

### `CareerPageResponse`
```
summary:
  rank_number, rank_label, rank_name_raw, rank_tier
  current_xp, xp_for_next_rank, xp_total, progress_pct
  is_max_rank, recorded_at
hero_progress:
  xp_total_required, xp_remaining, percentage, current_rank
projections:
  xp_per_day_active, xp_per_day_fallback
  estimated_hero_date, estimated_rank_cap_date
charts:
  rank_progress_gauge: PlotlyFigurePayload | null
  hero_progress_gauge: PlotlyFigurePayload | null
  xp_history_figure: PlotlyFigurePayload | null
  lusr_rating_figure: PlotlyFigurePayload | null
xp_history: list[CareerHistoryPoint]
lusr:
  current_rating, current_tier_label, current_playlist_group, trend_label
  checkpoints: list[CareerLusrCheckpoint]
top_matches_preview: list[CareerTopMatch]
encounters_preview: list[CareerEncounter]
```

**`CareerTopMatch`**
```
match_id, start_time, map_ui, mode_ui, playlist_label
performance_score, badge_type, score_label, outcome_label
```

**`CareerEncounter`**
```
encounter_key, opponent_gamertag, count_matches, wins, losses, last_seen_at
```

**`CareerTopMatchesResponse`**
```
items: list[CareerTopMatch]
```

**`CareerEncountersResponse`**
```
items: list[CareerEncounter]
```

### Stores / query keys
- `useCareerPageStore` : expandedPanels, selectedTopMatchTab
- `['career', playerSlug]`
- `['career', playerSlug, 'top-matches']`
- `['career', playerSlug, 'encounters']`

### Critères de recette
- Même rang, même XP total, même progression Hero et même statut max rank que Streamlit
- Même historique XP et mêmes projections pour un joueur donné
- Même LUSR courant et même tendance
- Même liste des top matches et encounters à données équivalentes

---

## Contrats Slice 3 — Stats [V7 §2] : Historique des parties (Phase A)

### Endpoints

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| POST | `/api/v1/players/{player_slug}/pages/match-history/query` | `MatchHistoryQueryRequest` | `MatchHistoryPageResponse` | `match_history.py` |
| POST | `/api/v1/players/{player_slug}/pages/match-history/export` | `MatchHistoryExportRequest` | `FileTokenResponse` | `match_history.py` |

### `MatchHistoryQueryRequest`
```
filters: FilterContextInput
pagination: PaginationRequest
columns: list[str] | null
include_export_hint: bool
```

### `MatchHistoryPageResponse`
```
summary:
  total_matches_scoped, total_matches_unfiltered
  period_label, active_filter_mode
table: PaginatedResponse[MatchHistoryRow]
available_sort_fields: list[str]
export_hint:
  file_name, estimated_rows, token | null
```

**`MatchHistoryRow`**
```
match_id, start_time, start_time_label
outcome_code, outcome_label, score_label
map_ui, mode_ui, playlist_label
team_mmr, enemy_mmr, delta_mmr
win_rate_hist, win_rate_hist_total
performance_score_relative
average_life_mmss
match_url
```

**`MatchHistoryExportRequest`**
```
filters: FilterContextInput
sort: list[SortSpec] | null
columns: list[str] | null
format: "csv"
```

**`FileTokenResponse`**
```
file_token: str
file_name: str
content_type: str
download_path: str
expires_at: datetime
estimated_rows: int | null
```

### Stores / query keys
- `useGlobalFilterStore` : filterContext, resolvedOptions
- `useMatchHistoryTableStore` : page, pageSize, sorting, visibleColumns
- `['filters-resolve', playerSlug, filterContextHash]`
- `['match-history', playerSlug, filterContextHash, page, pageSize, sortHash]`

### Critères de recette
- Même cardinalité de lignes que la page Streamlit à scope égal
- Même ordre pour un tri équivalent
- Mêmes valeurs calculées sur chaque ligne critique : score, `win_rate_hist`, `performance_score_relative`, `delta_mmr`
- Export CSV représentant exactement le scope courant

---

## Contrats Slice 4 — Explorer [V7 §5] : Recherche + Filtres (Phase A)

### Endpoints

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| GET | `/api/v1/directory/gamertags/search` | `q, limit` | `GamertagSearchResponse` | `explorer_logic.py`, `explorer_data.py` |
| POST | `/api/v1/players/{player_slug}/pages/explorer/matches-query` | `ExplorerMatchesQueryRequest` | `ExplorerMatchesQueryResponse` | `explorer.py` |
| POST | `/api/v1/players/{player_slug}/pages/explorer/player-query` | `ExplorerPlayerQueryRequest` | `ExplorerPlayerQueryResponse` | `explorer_results.py` |

### Schémas

**`GamertagSearchResponse`**
```
query: str
items: list[GamertagSuggestion]
  - gamertag, xuid, score, exact_match: bool
```

**`ExplorerMatchesQueryRequest`**
```
filters: FilterContextInput
match_filters:
  selected_date: date | null
  squad_scope: "solo" | "squad" | "all"
  experience_type: str | null
  playlist: str | null
  mode: str | null
  map: str | null
  selected_match_id: str | null
pagination: PaginationRequest
```

**`ExplorerPlayerQueryRequest`**
```
target_gamertag: str
filters: FilterContextInput | null
```

**`ExplorerPlayerQueryResponse`**
```
target: { gamertag, xuid }
summary: { matches_together, wins_together, losses_together, last_seen_at }
allies_table: list[ExplorerEncounterRow]
enemies_table: list[ExplorerEncounterRow]
common_matches: list[ExplorerMatchRow]
```

**`ExplorerMatchesQueryResponse`**
```
summary:
  total_matches: int
  selected_match_id: str | null
table: PaginatedResponse[ExplorerMatchRow]
```

**`ExplorerEncounterRow`**
```
gamertag, xuid, count_matches, wins, losses, last_seen_at
same_team: bool | null
```

**`ExplorerMatchRow`**
```
match_id, start_time, start_time_label, map_ui, mode_ui
playlist_label, outcome_label, score_label, is_with_friends, experience_type_label
```

### Stores / query keys
- `useExplorerStore` : searchMode, playerSearchInput, selectedMatchId, localMatchFilters, pagination
- `['gamertag-search', q]`
- `['explorer', 'matches', playerSlug, filterContextHash, localMatchFilterHash, page, sortHash]`
- `['explorer', 'player', playerSlug, targetGamertag, filterContextHash]`

### Critères de recette
- Mêmes suggestions fuzzy pertinentes pour une même requête
- Même résolution gamertag → xuid qu'en Streamlit
- Mêmes résultats de filtres match par match
- Même comportement sur deep links `match_id` et `gamertag`
- Ouverture du même Match View à partir d'une ligne ou d'un deep link

---

## Contrats Slice 4 (suite) — Explorer [V7 §5] : Match View (Phase B) + Last Match (Phase C)

### Stratégie de chargement Match View

- La V1 du contrat reste **page-oriented** : un appel doit suffire à fermer l'écran complet sans recalcul métier côté front.
- Le backend retourne au minimum le `header`, `rank`, `summary_tab`, `team_tab`, `citations_tab` et les aperçus médias nécessaires au rendu complet.
- Si certaines sous-sections deviennent trop lourdes en pratique, seules `combat_tab` et `media_tab` peuvent être rendues lazy dans un second temps, par décision explicite et documentée.
- Même en lazy, le front ne recompose jamais lui-même les médailles, armes, rosters ou événements à partir de flux bruts non contractuels.

### Endpoints

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| GET | `/api/v1/players/{player_slug}/matches/{match_id}` | — | `MatchViewResponse` | `match_view.py` + `match_view_*` |
| POST | `/api/v1/players/{player_slug}/pages/last-match/resolve` | `LastMatchResolveRequest` | `LastMatchResolveResponse` | `last_match.py` |

### `MatchViewResponse`
```
header:
  match_id, start_time, start_time_label
  outcome_code, outcome_label, outcome_color
  score_label, dominance_flag, had_bot_teammate
  map_ui, map_id, mode_ui, playlist_label
  performance_display, performance_color
rank:
  rating_type: "CSR" | "LUSR" | "none"
  tier_label, numeric_value, delta_value, icon_url | null
summary_tab:
  kpis: dict
  personal_result: dict
  medals: list[MatchMedal]
  citations: list[MatchCitation]
combat_tab:
  weapon_kills: list[MatchWeaponKill]
  highlight_events: list[MatchHighlightEvent]
  charts: list[PlotlyFigurePayload]
team_tab:
  roster: list[MatchRosterRow]
  scoreboard: list[MatchScoreboardRow]
  nemesis: list[MatchNemesisRow]
  encounters: list[MatchEncounterRow]
media_tab:
  media_items: list[AssociatedMediaItem]
citations_tab:
  commendations: list[MatchCitation]
  medals: list[MatchMedal]
```

**`MatchRosterRow`**
```
xuid, gamertag, team_side, is_me, is_bot
kills, deaths, assists, kda, damage_dealt, damage_taken
```

**`MatchWeaponKill`**
```
weapon_id, weapon_label, effective_weapon_id, kill_count
```

**`MatchHighlightEvent`**
```
event_time_ms, event_type, actor_xuid, target_xuid, weapon_id | null
```

**`LastMatchResolveRequest`**
```
filters: FilterContextInput
```

**`LastMatchResolveResponse`**
```
current_match_id: str
total_matches_in_scope: int
current_index: int
previous_match_id: str | null
next_match_id: str | null
session_tracking_key: str
```

### Stores / query keys
- `useMatchViewStore` : activeTab, selectedScoreboardRow, mediaLightboxIndex
- `useLastMatchStore` : resolvedMatchId, currentIndex, total
- `['match-view', playerSlug, matchId]`
- `['last-match', playerSlug, filterContextHash]`

### Critères de recette
- Même score, même outcome, même dominance flag et même rank affichés pour un match donné
- Même scoreboard, même roster, même set de médailles et de citations
- Même section armes et même timeline d'événements à données équivalentes
- Last Match pointe vers le même match que Streamlit pour un scope donné
- prev/next navigue sur la même liste ordonnée que la page Streamlit

> **Note armes** : les kills par arme sont exposés via la vue SQL `v_weapon_kills`
> (avec `effective_weapon_id = COALESCE(reconciled_as, weapon_id)`).
> Les labels FR/EN proviennent de `weapon_labels` dans `metadata.duckdb`.
> Le front ne doit jamais résoudre de `weapon_id` lui-même.

---

## Contrats Slice 2 (suite) — Profil [V7 §7] : Citations (Phase B)

> **Statut** : placeholder — schémas à figer avant implémentation Phase B.

### Endpoints identifiés

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| POST | `/api/v1/players/{player_slug}/pages/citations` | `CitationsQueryRequest` | `CitationsPageResponse` | `citations.py` |

### Schémas (à détailler)

**`CitationsQueryRequest`**
```
filters: FilterContextInput
```

**`CitationsPageResponse`**
```
commendations: list[CommendationSummary]    # H5G commendations
medals_summary: list[MedalSummary]          # top médailles filtrées
deltas: { filtered_total, unfiltered_total, delta_count }
distribution_chart: PlotlyFigurePayload | null
```

### Stores / query keys
- `['citations', playerSlug, filterContextHash]`

---

## Contrats Slice 3 (suite) — Stats [V7 §2] : Séries temporelles (Phase B)

> **Statut** : placeholder — schémas à figer avant implémentation Phase B.

### Endpoints identifiés

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| POST | `/api/v1/players/{player_slug}/pages/timeseries` | `TimeseriesQueryRequest` | `TimeseriesPageResponse` | `timeseries.py`, `TimeseriesService` |

### Notes de conception
- La page a 5 onglets (KPIs, Cumul, Forme récente, Intensité, Distributions)
- Livrer du Plotly JSON plutôt que réécrire les graphes côté front
- Le downsampling (`downsample_for_plot`) reste côté Python

### Stores / query keys
- `useTimeseriesStore` : activeTab
- `['timeseries', playerSlug, filterContextHash]`

---

## Contrats Slice 3 (suite) — Stats [V7 §2] : Comparaison de sessions (Phase C)

> **Statut** : placeholder — schémas à figer avant implémentation Phase C.

### Endpoints identifiés

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| POST | `/api/v1/players/{player_slug}/pages/session-compare` | `SessionCompareRequest` | `SessionCompareResponse` | `session_compare.py`, `session_compare_logic.py` |

### Notes de conception
- Sélection A/B côté front, envoyée au backend
- Contexte historique calculé côté Python (`compute_historical_context`)
- 15 composants visuels à servir

### Stores / query keys
- `useSessionCompareStore` : sessionA, sessionB
- `['session-compare', playerSlug, filterContextHash, compareStateHash]`

---

## Contrats Slice 5 — Accueil [V7 §1] : Home Mission Control

> **Statut** : placeholder — schémas à figer après Slices 2+4 livrés.

### Endpoints identifiés

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| GET | `/api/v1/players/{player_slug}/pages/home` | — | `HomePageResponse` | `home_mission_control.py`, `home_mission_control_logic.py` |
| GET | `/api/v1/players/{player_slug}/battlepass` | — | `BattlePassResponse` | `home_mission_control_api.py` |
| GET | `/api/v1/players/{player_slug}/challenges` | — | `ChallengesResponse` | `home_mission_control_api.py` |

### Notes de conception
- La Home est une **route composée** (agrégateur) — dépend de Career, Match View, Media
- Battle Pass et Challenges sont des appels **live API Halo** avec cache process-level (4h / 1h)
- Ces endpoints live ont des loading states et staleTime/gcTime distincts des données locales

### KPI Bar (décision de design)

> La KPI Bar présente dans le header V7 (K/D, Win%, matches played, KDA) peut être alimentée par :
> 1. Un champ dans `FilterContextResolved` (calculé lors du resolve) — **recommandé MVP**
> 2. Un endpoint dédié `/api/v1/players/{player_slug}/kpi-bar`
> 3. Un champ dans `PageEnvelope` (`kpi_bar: KPIBarData`)
>
> **Décision provisoire** : option 1 (champ dans `FilterContextResolved`). À valider lors de Slice 0b.

### Stores / query keys
- `useHomeStore`
- `['home', playerSlug]`
- `['home', playerSlug, 'battlepass']` — staleTime : 4h
- `['home', playerSlug, 'challenges']` — staleTime : 1h

---

## Contrats Slice 6 — Escouade [V7 §3] : Synergies + Contributions

> **Statut** : placeholder — section la plus complexe, contrats à stabiliser en dernier.

### Endpoints identifiés

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| POST | `/api/v1/players/{player_slug}/pages/teammates` | `TeammatesQueryRequest` | `TeammatesPageResponse` | `TeammatesService`, sous-modules `teammates_*` |

### Notes de conception
- 13 sous-modules côté Python, sélection multi (max 3 coéquipiers)
- Données cross-player via `shared.match_participants` (pas les DBs individuelles)
- PersonalScores API pour le radar 6-axes (§3.7.2)
- Highlight events pour impact heatmap + ranking (8 emojis, scoring MVP/LVP)
- La clarification du modèle multi-joueur est un prérequis

### Stores / query keys
- `useTeammatesStore` : selectedMates, cachedOptions
- `['teammates', playerSlug, filterContextHash, teammatesSelectionHash]`

---

## Contrats Slice 7 — Synthèse [V7 §4] : Solo vs Escouade

> **Statut** : placeholder — contrats à figer après Accueil et Escouade.

### Endpoints identifiés

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| POST | `/api/v1/players/{player_slug}/pages/synthesis` | `SynthesisQueryRequest` | `SynthesisPageResponse` | `synthesis.py`, `KPIStats` |

### Notes de conception
- Sélecteur de période (all, 2y, 1y, 1m, 1w)
- Bipolaire Solo vs Escouade (6 métriques, Cyan/Vert)
- L'ancienne page Objective Analysis est **absorbée** (voir Slice 7 dans SLICES.md)
- La query key `['objective-analysis', ...]` est supprimée

### Stores / query keys
- `['synthesis', playerSlug, filterContextHash, period]`

---

## Contrats Slice 8 — Médias [V7 §6] : Galerie + Lightbox

> **Statut** : placeholder — contrats à figer après Explorer (Phase B).

### Endpoints identifiés

| Méthode | Path | Requête | Réponse | Source Python |
|---------|------|---------|---------|---------------|
| GET | `/api/v1/players/{player_slug}/pages/media` | querystring filtres | `MediaPageResponse` | `media_v2.py`, `MediaIndexer` |
| POST | `/api/v1/settings/media/reset-index` | `MediaResetRequest` | `AsyncJobStatus` | (déjà dans Slice 1) |

### Notes de conception
- 8 contrôles toolbar (tri, filtre, groupement, affichage)
- Likes persistés en **localStorage uniquement** (pas de sync serveur)
  - Clé : `levelup_liked_media` → `Set<media_id>`
  - Migration depuis `mv2_liked_media` (session_state) au premier chargement
- Lightbox avec navigation ◀ ▶ + métadonnées match enrichies
- Backend sert URLs/paths de thumbnails + enrichissements

### Stores / query keys
- `useMediaStore` : filters, sort, groupMode, likes
- `['media', playerSlug, mediaFilterHash]`

---

## Query keys TanStack normalisées (référence complète)

```
['bootstrap']
['players']
['filters-resolve', playerSlug, filterContextHash]
['settings']
['setup-status']
['job', jobId]
['device-flow', attemptId]
['career', playerSlug]
['career', playerSlug, 'top-matches']
['career', playerSlug, 'encounters']
['citations', playerSlug, filterContextHash]
['match-history', playerSlug, filterContextHash, page, pageSize, sortHash]
['gamertag-search', q]
['explorer', 'matches', playerSlug, filterContextHash, localMatchFilterHash, page, sortHash]
['explorer', 'player', playerSlug, targetGamertag, filterContextHash]
['match-view', playerSlug, matchId]
['last-match', playerSlug, filterContextHash]
['home', playerSlug]
['home', playerSlug, 'battlepass']
['home', playerSlug, 'challenges']
['timeseries', playerSlug, filterContextHash]
['session-compare', playerSlug, filterContextHash, compareStateHash]
['teammates', playerSlug, filterContextHash, teammatesSelectionHash]
['synthesis', playerSlug, filterContextHash, period]
['media', playerSlug, mediaFilterHash]
```
