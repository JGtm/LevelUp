# OPENAPI_MVP_P0_P1.md — Contrats OpenAPI MVP des parcours P0/P1

> Dernier livrable de cadrage avant implémentation.
> Ce document fige les premiers contrats HTTP que le backend Go devra servir en priorité, sans attendre de documenter l'intégralité du produit.

## Rôle du document

Ce document borne le périmètre contractuel minimal du backend Go avant la première ligne de code produit.

Il sert à :

1. éviter que les handlers Go improvisent les shapes de réponse ;
2. préserver les parcours P0/P1 déjà matérialisés dans l'API FastAPI actuelle ;
3. limiter le freeze OpenAPI aux surfaces réellement prioritaires.

## Périmètre couvert

### P0

1. bootstrap shell ;
2. players list ;
3. session context ;
4. résolution de filtres.

### P1

1. career ;
2. match history ;
3. explorer ;
4. match view.

## Sources de vérité existantes

Les routes et schémas actuels à préserver viennent principalement de :

1. `apps/api/app/routers/bootstrap.py`
2. `apps/api/app/routers/filters.py`
3. `apps/api/app/routers/career.py`
4. `apps/api/app/routers/match_history.py`
5. `apps/api/app/routers/explorer.py`
6. `apps/api/app/schemas/bootstrap.py`
7. `apps/api/app/schemas/filters.py`
8. `apps/api/app/schemas/career.py`
9. `apps/api/app/schemas/match_history.py`
10. `apps/api/app/schemas/explorer.py`
11. `apps/api/app/schemas/match_view.py`
12. `apps/api/app/schemas/common.py`

## Invariants globaux à figer

1. Préfixe d'API : `/api/v1`.
2. Les méthodes HTTP actuelles des routes P0/P1 restent inchangées pour le MVP Go.
3. Le contrat d'erreur reste `ApiErrorSchema` tel que documenté par `apps/api/app/schemas/common.py`.
4. `X-Request-ID` reste présent dans les réponses.
5. Les timestamps sont sérialisés en ISO-8601.
6. Une liste connue mais vide reste une liste vide.
7. Une valeur optionnelle absente reste nullable/omise ; elle n'est pas remplacée par un faux défaut métier.
8. Le MVP Go ne doit pas imposer une enveloppe générique unique si le contrat FastAPI actuel expose déjà des réponses top-level spécifiques.

## Contrat d'erreur commun

Toute erreur bloquante des routes P0/P1 doit rester compatible avec :

```json
{
  "code": "player_not_found",
  "message": "Joueur introuvable.",
  "retryable": false,
  "details": null,
  "field_errors": null
}
```

La taxonomie à utiliser pour les erreurs Halo est détaillée dans [HALO_PROVIDER_ERROR_TAXONOMY.md](HALO_PROVIDER_ERROR_TAXONOMY.md).

## Routes MVP P0

### 1. `GET /api/v1/bootstrap`

**Contrat de sortie** : `BootstrapResponse`

Blocs top-level à préserver :

1. `setup_required`
2. `auth_state`
3. `setup_state`
4. `current_player`
5. `available_players`
6. `locale`
7. `hints_visible_default`
8. `feature_flags`
9. `capabilities`
10. `settings_excerpt`
11. `linked_halo_identity`
12. `active_sync_job_id`

Invariants :

1. `auth_state` reste l'un de `missing`, `partial`, `ready`.
2. `setup_state` reste piloté par le produit, pas par le provider brut.
3. `current_player` peut être `null`.
4. `available_players` ne doit jamais devenir `null` si la liste est connue.
5. Les capabilities shell existantes restent distinctes du bloc Halo/capabilities documenté dans [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md).

### 2. `GET /api/v1/players`

**Contrat de sortie** : `PlayersListResponse`

Top-level à préserver :

1. `items`
2. `default_player_slug`

Invariant :

`items` est une liste de `PlayerSummary` avec `player_slug`, `gamertag`, `xuid`, `waypoint_player`, `is_demo`.

### 3. `POST /api/v1/session/context`

**Contrat d'entrée** : `SessionContextRequest`

1. `player_slug`
2. `locale`
3. `hints_visible`

**Contrat de sortie** : `SessionContextResponse`

1. `current_player`
2. `locale`
3. `hints_visible`
4. `capabilities`

Invariant :

la route modifie le contexte de session sans exposer la session entière.

### 4. `POST /api/v1/players/{player_slug}/filters/resolve`

**Contrat d'entrée** : `FilterContextInput`

Blocs à préserver :

1. `filter_mode`
2. `period`
3. `sessions`
4. `cascade`

**Contrat de sortie** : `FilterContextResolved`

1. `effective`
2. `available_options`
3. `session_options`
4. `counts`

Invariant :

`effective` représente la forme normalisée du contexte, pas un echo brut de la requête.

## Routes MVP P1

### 5. `GET /api/v1/players/{player_slug}/pages/career`

**Contrat de sortie** : `CareerPageResponse`

Blocs à préserver :

1. `summary`
2. `hero_progress`
3. `projections`
4. `charts`
5. `xp_history`
6. `lusr`
7. `top_matches_preview`
8. `encounters_preview`

Invariants :

1. `charts` peut contenir des valeurs `null` ciblées.
2. `summary` et `lusr` restent optionnels si la donnée manque.
3. Le contrat métier reste page-oriented, pas une simple projection brute du canonique Halo.

### 6. `GET /api/v1/players/{player_slug}/pages/career/top-matches`

**Contrat de sortie** : `CareerTopMatchesResponse`

Top-level à préserver :

1. `items`

### 7. `GET /api/v1/players/{player_slug}/pages/career/encounters`

**Contrat de sortie** : `CareerEncountersResponse`

Top-level à préserver :

1. `items`

### 8. `POST /api/v1/players/{player_slug}/pages/match-history/query`

**Contrat d'entrée** : `MatchHistoryQueryRequest`

1. `filters`
2. `pagination`
3. `columns`
4. `include_export_hint`

**Contrat de sortie** : `MatchHistoryPageResponse`

1. `summary`
2. `table`
3. `available_sort_fields`
4. `export_hint`

Invariants :

1. `table` reste une `PaginatedResponse` avec `items` et `pagination`.
2. `export_hint` reste optionnel.
3. `match_url` reste porté par chaque ligne, pas reconstruit côté frontend.

### 9. `POST /api/v1/players/{player_slug}/pages/match-history/export`

**Contrat d'entrée** : `MatchHistoryExportRequest`

1. `filters`
2. `sort`
3. `columns`
4. `format`

**Contrat de sortie** : `FileTokenResponse`

1. `file_token`
2. `file_name`
3. `content_type`
4. `download_path`
5. `expires_at`
6. `estimated_rows`

Invariant :

le MVP renvoie un jeton/tunnel d'export, pas un flux fichier direct sur cette route.

### 10. `GET /api/v1/directory/gamertags/search`

**Paramètres** :

1. `q`
2. `limit`

**Contrat de sortie** : `GamertagSearchResponse`

1. `query`
2. `items`

Invariant :

une recherche sans résultat renvoie `200` avec `items = []`.

### 11. `POST /api/v1/players/{player_slug}/pages/explorer/matches-query`

**Contrat d'entrée** : `ExplorerMatchesQueryRequest`

1. `filters`
2. `match_filters`
3. `pagination`

**Contrat de sortie** : `ExplorerMatchesQueryResponse`

1. `summary`
2. `table`

### 12. `POST /api/v1/players/{player_slug}/pages/explorer/player-query`

**Contrat d'entrée** : `ExplorerPlayerQueryRequest`

1. `target_gamertag`
2. `filters`

**Contrat de sortie** : `ExplorerPlayerQueryResponse`

1. `target`
2. `summary`
3. `allies_table`
4. `enemies_table`
5. `common_matches`

### 13. `GET /api/v1/players/{player_slug}/matches/{match_id}`

**Contrat de sortie** : `MatchViewResponse`

Blocs top-level à préserver :

1. `header`
2. `rank`
3. `summary_tab`
4. `combat_tab`
5. `team_tab`
6. `media_tab`
7. `citations_tab`

Invariants :

1. La vue match reste un contrat par onglets/blocs métier, pas un dump unique du `MatchDetail` canonique.
2. Un match introuvable renvoie `404` avec `code = match_not_found` ou équivalent stable.
3. Les blocs partiels restent autorisés si le contrat principal du match est servable ; utiliser warnings/limitations plutôt qu'un échec global quand possible.

## Statuts HTTP MVP à conserver

| Cas | Statut attendu |
|-----|----------------|
| succès lecture | 200 |
| création profil/setup hors périmètre MVP P0/P1 | 201 |
| job async hors périmètre MVP P0/P1 | 202 |
| validation / requête invalide | 400 ou 422 selon le contrat |
| auth requise | 401 |
| accès interdit | 403 |
| ressource introuvable | 404 |
| conflit produit | 409 |
| rate limit provider | 429 |
| erreur interne | 500 |
| réponse provider invalide | 502 |
| provider indisponible | 503 |
| timeout provider | 504 |

## Ce que ce document ne freeze pas

1. setup/auth device flow ;
2. sync et jobs persistants ;
3. home/battlepass/challenges ;
4. media page complète ;
5. PvE, citations, timeseries, session compare, settings.

Ces surfaces existent dans le produit, mais elles ne sont pas nécessaires pour arrêter la phase documentaire préalable au code Go.

## Décision de gel

Le MVP contractuel préalable au code est considéré suffisant dès lors que :

1. les routes P0/P1 ci-dessus sont figées ;
2. la taxonomie d'erreurs est figée ;
3. toute évolution ultérieure vient d'un besoin d'implémentation ou d'un écart de parité réel.

## Documents liés

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) pour la source canonique Halo.
2. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md) pour la projection canonique -> read models produit.
3. [HALO_PROVIDER_ERROR_TAXONOMY.md](HALO_PROVIDER_ERROR_TAXONOMY.md) pour les erreurs et limitations.
4. [PORTING_REFERENCE.md](PORTING_REFERENCE.md) pour le cadrage global des surfaces prioritaires.
5. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) pour le rattachement au programme.
