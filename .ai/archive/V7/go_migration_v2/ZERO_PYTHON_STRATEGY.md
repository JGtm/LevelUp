# Stratégie Zéro Python — LevelUp Go Migration

> [!IMPORTANT]
> **Objectif dur** : le produit final ne contient **aucun runtime Python**.
>
> Ce document complète [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md),
> [MATRIX.md](MATRIX.md) et [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).
> Il ne les remplace pas — il durcit la cible finale.

## Lecture obligatoire

1. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) — trajectoire, phases, gates, risques, décisions.
2. [MATRIX.md](MATRIX.md) — couverture package/script/commande.
3. [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) — compat auth/jobs/packaging.

---

## Position : zéro Python, pas "moins de Python"

Le plan maître propose un remplacement progressif. Ce document précise la cible terminale :

| Critère | Valeur |
|---------|--------|
| Runtime Python en production | **0** |
| `.py` dans le chemin critique | **0** |
| Dépendance pip/venv à l'installation | **0** |
| Bridge Python SPNKr | **Supprimé** — client Go direct dès S11 |
| `src/ai/` (RAG, MCP) | **Hors scope** — outillage dev, pas produit |

**Ce que "zéro Python" signifie concrètement** :
1. L'utilisateur final ne doit **jamais** installer Python, pip ou un venv.
2. Le déploiement (Docker, binaire, script) ne contient aucun `.py` exécuté.
3. Les tests de parité peuvent rester en Python pendant la transition — ils ne tournent pas en prod.
4. `src/ai/` (ChromaDB, MCP server) est un outil développeur séparé — pas un composant produit.

---

## Inventaire complet : chaque module Python et son destin

### Couche API (`apps/api/`) — ~12 000 LOC

| Module Python | Destin | Remplacement Go | Sprint |
|---------------|--------|-----------------|:------:|
| `routers/*.py` (16 fichiers, 28+ endpoints) | **Remplacé** | `internal/api/handlers/` | S06 |
| `services/*.py` (18 fichiers) | **Remplacé** | `internal/{domain}/service.go` | S06–S13 |
| `deps/auth.py` (SessionData, SessionStore) | **Remplacé** | `internal/auth/session.go` | S14 |
| `core/errors.py` (ApiError) | **Remplacé** | `internal/platform/errors/` | S04 |
| `core/config.py` (Settings) | **Remplacé** | `internal/platform/config/` | S04 |
| `middleware/` | **Remplacé** | Middleware Chi | S04 |

### Couche données (`src/data/`) — ~12 000 LOC

| Module Python | Destin | Remplacement Go | Sprint |
|---------------|--------|-----------------|:------:|
| `repositories/duckdb_repo.py` (~500L) | **Remplacé** | `internal/platform/duckdb/` | S05 |
| `repositories/_match_queries*.py` | **Remplacé** | SQL + `internal/queries/` | S05 |
| `repositories/_medals_repo.py` | **Remplacé** | SQL queries Go | S05 |
| `repositories/_career_repo.py` | **Remplacé** | SQL queries Go | S05 |
| `repositories/_killer_victim_repo.py` | **Remplacé** | SQL queries Go | S05 |
| `repositories/_weapon_kills_repo.py` | **Remplacé** | SQL queries Go | S05 |
| `repositories/_media_repo.py` | **Remplacé** | SQL queries Go | S13 |
| `services/*.py` | **Remplacé** | `internal/{domain}/` | S06–S12 |
| `sync/engine.py` + 12 mixins (~13 000 LOC avec transformers) | **Remplacé** | `internal/sync/` | S18–S20 |
| `sync/api_client.py` (SPNKr wrapper) | **Remplacé** | `pkg/haloapi/` (client Go direct) | S11 puis S18 |
| `sync/scope.py` (SyncScope, 96 champs) | **Remplacé** | `internal/sync/scope.go` | S20 |
| `sync/models*.py` | **Remplacé** | Structs Go | S18 |
| `sync/migrations*.py` (35 steps) | **Remplacé** | `internal/platform/migrations/` | S21 |
| `sync/transformers/` (~2 400 LOC) | **Remplacé** | `internal/sync/transform/` | S18 |
| `sync/_batch_audit.py`, `_batch_columns.py` | **Remplacé** | `internal/sync/batch/` | S18 |
| `sync/_career_rank_api.py`, `_tokens.py`, `_asset_langs.py` | **Remplacé** | `internal/sync/` | S18 |
| `migration/` (registry + steps) | **Remplacé** | `internal/platform/migrations/` | S21 |
| `media_indexer.py` | **Remplacé** | `internal/media/indexer.go` | S24 |

### Couche analyse (`src/analysis/`) — ~14 000 LOC (62 fichiers)

| Module Python | Destin | Remplacement Go | Sprint |
|---------------|--------|-----------------|:------:|
| `performance_score.py` + `_performance_relative.py` | **Remplacé** | `internal/analysis/performance.go` | S10 |
| `skill_rating.py` + `_trueskill_math.py` | **Remplacé** | `internal/analysis/skill.go` | S10 |
| `sessions.py` | **Remplacé** | `internal/analysis/sessions.go` | S09 |
| `killer_victim.py` | **Remplacé** | `internal/analysis/killervictim.go` | S08 |
| `weapon_parser.py` (parsing binaire film) | **Remplacé** | `internal/analysis/weaponparser.go` | S22 |
| `objective_participation.py` | **Remplacé** | `internal/analysis/objectives.go` | S12 |
| `map_analysis.py` | **Remplacé** | SQL DuckDB uniquement | S10 |
| `match_cadence.py` | **Remplacé** | SQL DuckDB ou Go trivial | S10 |
| `win_streaks.py` | **Remplacé** | SQL DuckDB ou Go trivial | S10 |
| `citations/` (engine + custom rules) | **Remplacé** | `internal/analysis/citations/` | S13 |
| `spawn_detection.py` | **Remplacé** | `internal/analysis/spawns.go` | S12 |
| `_composite.py` | **Remplacé** | `internal/analysis/composite.go` | S12 |
| `_calibration_loaders.py` | **Remplacé** | `internal/analysis/calibration.go` | S10 |

### Auth (`src/auth/`) — ~900 LOC (5 fichiers)

| Module Python | Destin | Remplacement Go | Sprint |
|---------------|--------|-----------------|:------:|
| `provider.py` (tokens, cache 4h TTL) | **Remplacé** | `internal/auth/provider.go` | S15 |
| `_msal.py` (MSAL wrapper) | **Remplacé** | MSAL Go SDK | S15 |
| `_halo_exchange.py` (access→spartan) | **Remplacé** | `internal/halo/exchange.go` | S15 |
| `_constants.py` | **Remplacé** | `internal/auth/constants.go` | S15 |

### Utils (`src/utils/`) — ~800 LOC utiles

| Module Python | Destin | Remplacement Go | Sprint |
|---------------|--------|-----------------|:------:|
| `db.py` (context managers DuckDB) | **Remplacé** | `internal/platform/duckdb/pool.go` | S05 |
| `discord_notifier.py` + `_discord_*.py` | **Remplacé** | `internal/platform/discord/` | S25 |
| `formatting.py` | **Remplacé** | `internal/platform/format/` | S05 |
| `profiles.py` | **Remplacé** | `internal/platform/config/profiles.go` | S04 |
| `sync_lock.py` | **Remplacé** | `internal/sync/lock.go` | S18 |
| `env.py`, `paths.py`, `secrets.py` | **Remplacé** | `internal/platform/config/` | S04 |
| `demo.py` | **Remplacé** | `internal/platform/config/demo.go` | S04 |
| `tailscale.py` | **Supprimé** | Hors scope Go | — |
| `log_config.py` | **Remplacé** | `log/slog` stdlib | S04 |

### UI Streamlit (`src/ui/`, `streamlit_app.py`) — ~6 000 LOC

| Module Python | Destin |
|---------------|--------|
| Tout `src/ui/` | **Supprimé** (remplacé par React, déjà fait) |
| `streamlit_app.py` | **Supprimé** |
| `streamlit_app_v7.py` | **Supprimé** |

### Scripts (`scripts/`) — ~3 000 LOC

| Script Python | Destin | Remplacement Go | Sprint |
|---------------|--------|-----------------|:------:|
| `sync.py` | **Remplacé** | `cmd/levelup/ sync` | S18 |
| `backfill_data.py` | **Remplacé** | `cmd/levelup/ backfill` | S20 |
| `backup_player.py` | **Remplacé** | `cmd/levelup/ backup` | S24 |
| `restore_player.py` | **Remplacé** | `cmd/levelup/ restore` | S24 |
| `check_env.py` | **Remplacé** | `cmd/levelup/ check-env` | S04 |
| `diagnose_player_db.py` | **Remplacé** | `cmd/levelup/ diagnose` | S24 |
| `index_media.py` | **Remplacé** | `cmd/levelup/ index-media` | S24 |
| `archive_season.py` | **Remplacé** | `cmd/levelup/ archive` | S24 |
| `populate_*.py` | **Remplacé** | `cmd/levelup/ seed` | S05 |
| `healthcheck_db.py` | **Remplacé** | `cmd/levelup/ healthcheck` | S24 |
| `post_sync_compute.py` | **Absorbé** | Intégré dans sync | S19 |

### Ports (`src/ports/`) — ~200 LOC

| Module Python | Destin | Remplacement Go |
|---------------|--------|-----------------|
| `api.py` (HaloAPIPort) | **Remplacé** | `internal/halo/port.go` (interface) |
| `repository.py` (DataRepository) | **Remplacé** | `internal/platform/duckdb/repository.go` (interface) |

### SPNKr (`spnkr_pr/` + dépendance `spnkr`) — ~800 LOC local

| Module | Destin | Détail |
|--------|--------|--------|
| Lib `spnkr` (pip) | **Éliminée** | Remplacée par client HTTP Go direct |
| `spnkr_pr/` (patches locaux) | **Éliminés** | Portés dans le client Go |

### AI / Outillage dev (`src/ai/`) — ~1 200 LOC

| Module | Destin |
|--------|--------|
| `rag.py`, `_rag_*.py`, `mcp_server.py` | **Hors scope** — Reste Python si souhaité, pas un composant produit |

### Launcher et packaging

| Fichier | Destin | Remplacement Go |
|---------|--------|-----------------|
| `launcher.py` | **Remplacé** | `cmd/levelup/main.go` |
| `LevelUp.bat` / `LevelUp.sh` | **Simplifiés** | Lancent le binaire Go directement |
| `Dockerfile` | **Réécrit** | Multi-stage Go build, pas de Python |
| `docker-compose.yml` | **Simplifié** | Un seul service Go |
| `pyproject.toml` | **Supprimé** | `go.mod` |
| `Makefile` | **Réécrit** | Targets Go : build, test, lint |

---

## SPNKr : la seule exception Python — et comment l'éliminer

### Pourquoi SPNKr reste temporairement

SPNKr est une lib Python tierce (`pip install spnkr`) qui fait des appels HTTP vers les endpoints 343 Industries. Elle gère :
- L'authentification XBL/XSTS/Spartan/Clearance
- Le rate limiting (60 req/min)
- Le parsing des réponses JSON
- La gestion des films (chunks binaires)

### Ce que SPNKr fait réellement (et qui doit être reproduit en Go)

| Endpoint 343i | Méthode SPNKr | Usage LevelUp | Go equivalent |
|---------------|---------------|---------------|---------------|
| Profile | `get_user_by_gamertag()` | Résolution xuid | `GET /users/...` |
| Match History | `get_match_history()` | Sync delta/full | `GET /stats/matches/...` |
| Match Stats | `get_match_stats()` | Détails match | `GET /stats/matches/{id}` |
| Skill | `get_match_skill()` | CSR / MMR | `GET /skill/...` |
| Discovery | `get_map/playlist/variant()` | Assets metadata | `GET /discovery/...` |
| Film | `get_film_by_match_id()` | Weapon parsing | `GET /ugc/films/...` |
| Film chunk | `download_film_chunk()` | Parsing binaire | `GET` (direct binary) |
| Career Rank | `get_career_rank_progression()` | Progression | `GET /career/...` |
| Match Count | `get_match_count()` | Stats par mode | `GET /stats/summary/...` |
| Player Customization | `get_player_customization()` | Armures | `GET /appearance/...` |
| Users by ID | `get_users_by_id()` | Bulk xuid→gamertag | `GET /users/...` |
| Battle Pass | (SPNKr hors scope) | Battle Pass live | `GET /economy/...` |
| Challenges | (SPNKr hors scope) | Défis actifs | `GET /economy/...` |

### Ce que ces endpoints sont en réalité

Des appels HTTP REST classiques avec :
- **Headers** : `x-343-authorization-spartan: {spartan_token}`, `343-clearance: {clearance_token}`
- **Rate limit** : 60 req/min max (réponse 429 si dépassé)
- **Auth chain** : `access_token (MSAL) → XBL token → XSTS token → spartan_token + clearance_token`
- **Base URLs** : `https://halostats.svc.halowaypoint.com`, `https://economy.svc.halowaypoint.com`, `https://discovery-infiniteugc.svc.halowaypoint.com`, `https://gamecms-hacs.svc.halowaypoint.com`

**SPNKr n'a rien de magique** — c'est un client HTTP Python avec retry et parsing JSON. Tout est reproductible en ~500-800 lignes de Go.

### Décision d'architecture : API Halo en 2 niveaux

> **Décision** : le remplacement de SPNKr est structuré en deux niveaux :
> un **socle provider public** (`pkg/haloapi/`) et des **providers de titre**
> (`pkg/haloapi/titles/haloinfinite/`, puis d'autres titres si nécessaire).
> LevelUp consomme ensuite ces providers via un adaptateur interne orienté produit.

#### Pourquoi un package public

1. **Évolutivité produit** : LevelUp garde une API interne stable même si Halo Infinite est remplacé par un autre titre.
2. **Réutilisabilité** : le socle provider peut être utilisé par d'autres projets Go Halo sans embarquer LevelUp.
3. **Séparation des responsabilités** : le socle gère le HTTP, l'auth et les endpoints ; le provider de titre gère le mapping ; LevelUp gère DuckDB et la logique métier.
4. **Publication future** : le socle provider peut devenir un module public sans dépendance LevelUp.

#### Architecture du package

```
pkg/
   haloapi/
      client.go           # Client principal, rate limiter, retry
      client_options.go   # Options fonctionnelles (WithHTTPClient, WithRateLimiter, etc.)
      endpoints.go        # Registre centralisé d'endpoints (URLs, paths, versions)
      auth.go             # Échange tokens : MSAL → XBL → XSTS → Spartan → Clearance
      auth_models.go      # TokenSet, DeviceCodePending, etc.
      provider.go         # Interface TitleProvider + capabilities
      canonical.go        # Modèles canoniques (match, identity, career, assets)
      errors.go           # Types d'erreur (RateLimited, Unauthorized, etc.)
      titles/
         haloinfinite/
            provider.go     # Mapping Halo Infinite -> canonical
            endpoints.go    # Endpoints et versions Halo Infinite
            stats.go        # Match history, match details, skill
            discovery.go    # Maps, playlists, game variants
            economy.go      # Battle pass, challenges, customization
            ugc.go          # Films, chunks
            refdata.go      # Constantes et enums propres au titre
```

#### Principes d'architecture du package

1. **Registre d'endpoints centralisé** : toutes les URLs et paths 343i dans une struct `ServiceEndpoints` unique, modifiable pour tests (mock server) et overridable partiellement. Jamais de constantes magiques dispersées.
2. **Options fonctionnelles** : construction du client via `NewClient(opts ...Option)` — extensible sans casser l'API.
3. **Provider de titre explicite** : chaque titre mappe vers un modèle canonique unique avant exposition au produit.
4. **Pas de DuckDB** — aucune persistance, aucun import de base de données.
5. **Pas de logique LevelUp** — pas de performance score, pas de sessions, pas de citations.
6. **Pas de cache applicatif** — le cache de tokens est géré par le consommateur (LevelUp), pas par le client.
7. **Pas d'opinion sur le framework web** — utilisable avec Chi, Echo, Gin, ou sans framework.

> Note : les exemples de code Go détaillés (structs, constructeurs, méthodes) seront produits au moment de l'implémentation (S11 et S15), pas dans ce document de cadrage.

#### Relation avec LevelUp (`internal/`)

```
pkg/haloapi/               ← Socle provider Halo public
internal/halo/             ← Adaptateur LevelUp orienté produit
   adapter.go               ← Implémente HaloAPIPort via pkg/haloapi
  token_cache.go           ← Gère le cache MSAL → DuckDB sync_meta
  rate_policy.go           ← Politiques retry spécifiques LevelUp
   titles/
      haloinfinite.go        ← Sélection du provider Halo Infinite
```

L'adaptateur fait le pont : il gère le cache de tokens (DuckDB), sélectionne le provider de titre et délègue les appels réseau au socle public.

### Préparation documentaire avant toute implémentation multi-titre

Avant d'écrire le provider Go réel, cadrer et versionner :

1. le modèle canonique Halo consommé par LevelUp ;
2. la matrice de capabilities par titre et par surface produit ;
3. le registre des zones spécifiques au jeu à isoler ;
4. la politique de dégradation quand une surface n'est pas disponible sur un titre donné.

### Stratégie de remplacement — Client Go direct (pas de bridge Python)

> **Décision** : pas de bridge SPNKr Python transitoire. On implémente directement
> le client Go `pkg/haloapi/` dès le Sprint 11 (socle provider Halo).
> SPNKr n'a rien de magique — c'est un client HTTP avec retry et parsing JSON,
> reproductible en ~800-1 200 lignes de Go.

#### Phase A — Client Go `pkg/haloapi/` + provider `titles/haloinfinite/` (S11 puis S15)

Implémentation directe du package Go public.

**Modules à implémenter** :
1. `pkg/haloapi/client.go` — Client HTTP avec rate limiter + retry exponentiel + options fonctionnelles
2. `pkg/haloapi/endpoints.go` — Registre d'endpoints centralisé et modifiable
3. `pkg/haloapi/auth.go` — Échange `access_token → XBL → XSTS → spartan + clearance`
4. `pkg/haloapi/provider.go` — Interface provider + capability map
5. `pkg/haloapi/canonical.go` — Modèles canoniques consommés par LevelUp
6. `pkg/haloapi/titles/haloinfinite/stats.go` — Match history, match stats, match skill, career rank
7. `pkg/haloapi/titles/haloinfinite/discovery.go` — Maps, playlists, game variants
8. `pkg/haloapi/titles/haloinfinite/economy.go` — Battle Pass, challenges, customization
9. `pkg/haloapi/titles/haloinfinite/ugc.go` — Films, chunks binaires
10. `pkg/haloapi/titles/haloinfinite/refdata.go` — Constantes et spécificités du titre
11. `internal/halo/adapter.go` — Adaptateur LevelUp (cache tokens DuckDB, politiques métier)

**LOC estimées** : 800–1 200 lignes Go pour le package public + ~200 lignes pour l'adaptateur LevelUp.

#### Phase B — Validation complète (Gate Phase 4)

| # | Critère de validation | Vérification |
|---|---------------------|--------------|
| 1 | Tous les endpoints 343i utilisés ont un équivalent Go testé | Tests sur fixtures + 1 appel live |
| 2 | L'échange de tokens `access_token → spartan + clearance` fonctionne en Go natif | Test E2E device flow complet |
| 3 | Le rate limiter Go respecte les 60 req/min | Test de charge |
| 4 | Le retry exponentiel est implémenté et testé | Test avec mock 429/500 |
| 5 | 3 cycles de sync complets passent avec le client Go natif | Sync delta réelle sur 2+ joueurs |
| 6 | Le parsing des films fonctionne en Go (ou est reporté) | Tests weapon parser ou décision D6 |
| 7 | Battle Pass + Challenges fonctionnent via le client Go | Test live |

**Nettoyage Post-validation** :
1. Supprimer `spnkr` de tout fichier de dépendances
2. Supprimer `spnkr_pr/` (patches locaux)
3. Vérifier que le Dockerfile et les scripts de déploiement ne référencent plus Python
4. Vérifier qu'aucun `import spnkr` ne subsiste

#### Phase C — Publication en module Go indépendant (post-Gate Phase 5, optionnel)

Une fois le client stabilisé par l'usage LevelUp :

1. Extraire `pkg/haloapi/` vers un repo `github.com/{user}/haloapi-go`
2. Publier avec `go.mod` propre, README, exemples et tests
3. LevelUp devient un consommateur du module via `go get github.com/{user}/haloapi-go`
4. Le repo LevelUp conserve uniquement `internal/halo/adapter.go`

Cette extraction n'est **pas un prérequis** de la migration — c'est une opportunité post-migration.

---

## Échange de tokens Halo : le vrai défi (pas SPNKr)

La partie complexe n'est pas d'appeler les endpoints 343i — c'est d'obtenir les tokens.

### Chaîne d'authentification complète

```
1. MSAL Device Code Flow
   └─→ access_token (Microsoft OAuth2)

2. Xbox Live (XBL) Authentication
   POST https://user.auth.xboxlive.com/user/authenticate
   Body: { "RelyingParty": "http://auth.xboxlive.com", "TokenType": "JWT", "Properties": { "AuthMethod": "RPS", "SiteName": "user.auth.xboxlive.com", "RpsTicket": "d={access_token}" } }
   └─→ xbl_token + user_hash

3. XSTS Token
   POST https://xsts.auth.xboxlive.com/xsts/authorize
   Body: { "RelyingParty": "https://prod.xsts.halowaypoint.com", "TokenType": "JWT", "Properties": { "SandboxId": "RETAIL", "UserTokens": ["{xbl_token}"] } }
   └─→ xsts_token

4. Spartan Token
   POST https://settings.svc.halowaypoint.com/spartan-token
   Body: { "Audience": "urn:343:web-companion", "MinVersion": "4", "Proof": [{ "Token": "{xsts_token}", "TokenType": "Xbox_XSTSv3" }] }
   └─→ spartan_token (header x-343-authorization-spartan)

5. Clearance Token
   GET https://settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/players/xuid({xuid})/active
   Header: x-343-authorization-spartan: {spartan_token}
   └─→ clearance_token (header 343-clearance)
```

### Implémentation Go

L'implémentation Go reproduira cette chaîne à 5 étapes dans `pkg/haloapi/auth.go` : struct `TokenSet` (SpartanToken, ClearanceToken, ExpiresAt, XUID), méthode `ExchangeAccessToken(ctx, accessToken) → (*TokenSet, error)` qui enchaîne XBL → XSTS → Spartan → Clearance. Chaque méthode interne utilise `c.endpoints` pour résoudre les URLs — jamais de hardcode.

> Note : le code Go détaillé sera produit au Sprint 3.2, pas dans ce document de cadrage.

---

## Dépendances Python → Go : table de correspondance complète

| Dep Python | Usage | Dep Go | Notes |
|------------|-------|--------|-------|
| `fastapi` | API HTTP | `chi` | Choix : Chi (proche stdlib) |
| `uvicorn` | Serveur ASGI | `net/http` stdlib | Zéro dépendance |
| `pydantic` v2 | Validation | `go-playground/validator` + structs | Ou validation manuelle |
| `duckdb` | OLAP | `github.com/duckdb/duckdb-go` | CGo — POC Sprint 0 |
| `polars` | DataFrames | SQL DuckDB natif + structs Go | Pas d'équivalent nécessaire |
| `msal` | Auth MS | `github.com/AzureAD/microsoft-authentication-library-for-go` | SDK officiel |
| `spnkr` | API Halo | Client HTTP Go direct (`net/http`) | Voir section SPNKr |
| `aiohttp` | HTTP async | `net/http` + goroutines | Natif |
| `pyarrow` | Parquet | `github.com/apache/arrow-go` | Pour archives |
| `itsdangerous` | Cookie de session signé | `github.com/gorilla/securecookie` ou HMAC maison | Conforme à D4, pas de JWT |
| `structlog` | Logging | `log/slog` stdlib | Go 1.21+ |
| `watchdog` | FS watch | `github.com/fsnotify/fsnotify` | Pour media indexer |
| `bitstring` | Parsing binaire | `encoding/binary` stdlib | Weapon parser |
| `filelock` | Verrou fichier | `internal/sync/lock.go` (flock/LockFileEx) | Portage du write lease |
| `python-multipart` | Formdata | `net/http` stdlib | Natif |
| `plotly` | Charts | `domain/chart/` + `adapter/plotly/` | Backend seulement pour les surfaces serveur ; les figures déjà assemblées dans React restent frontend |
| `streamlit` | UI web | N/A | Supprimé (React) |
| `chromadb` | RAG vectoriel | N/A | Outillage dev, hors scope |
| `pandas` | DataFrames | N/A | Déjà interdit, Polars → SQL |
| `numpy` | Calcul num. | `math` stdlib | Trivial |
| `pytest` | Tests | `testing` stdlib + `testify` | |
| `httpx` | Test client | `net/http/httptest` stdlib | |
| `black`/`ruff` | Formatage | `gofmt`/`golangci-lint` | Natif |
| `mypy` | Typage | Compilateur Go | Natif |

**Total dépendances Go runtime estimées** : ~7-8 (`duckdb-go`, chi, msal-go, securecookie ou HMAC maison, validator, arrow-go, fsnotify, rate).

Contre ~15 dépendances Python runtime actuelles.

---

## Packaging final : zéro Python

### Distribution cible

```
levelup-v8.0.0-windows-amd64/
├── levelup.exe              # Binaire unique Go (API + sync + tools)
├── web/                     # Frontend React (go:embed ou dossier séparé)
│   ├── index.html
│   ├── assets/
│   └── ...
├── data/                    # Données utilisateur (DuckDB, config)
│   ├── warehouse/
│   ├── players/
│   └── cache/
├── db_profiles.json
├── app_settings.json
└── README.md
```

**Pas de** : `python`, `pip`, `venv`, `.py`, `requirements.txt`, `pyproject.toml`.

### Comparaison avant/après

| Aspect | Avant (Python) | Après (Go) |
|--------|----------------|------------|
| Installation | Python 3.12 + pip + venv + 15 deps | Télécharger + dézipper |
| Taille | ~500 MB (venv + deps) | ~100-200 MB (binaire CGo+DuckDB + web) |
| Démarrage | ~2-5 secondes | ~50 ms |
| Build multi-OS | Docker ou VM obligatoire | Build matrix par OS cible (CGO), pas de promesse de cross-compile triviale |
| Dockerfile | Multi-stage + pip install | Multi-stage Go build |
| CI | Python matrix + pip cache | Go build matrix |

### Dockerfile zéro Python

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o levelup ./cmd/levelup/

# Web build stage
FROM node:22-alpine AS web-builder
WORKDIR /app
COPY apps/web/ .
RUN npm ci && npm run build

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ffmpeg  # Pour media indexing (ffprobe)
COPY --from=builder /app/levelup /usr/local/bin/
COPY --from=web-builder /app/dist /opt/levelup/web
ENTRYPOINT ["levelup", "api"]
```

**Seule dépendance externe runtime** : `ffmpeg`/`ffprobe` (pour l'indexation media). Pas Python.

---

## Calendrier ajusté pour la cible zéro Python

Le plan maître propose 5 phases. L'objectif zéro Python signifie : **aucun Python dans le chemin produit dès Gate 4**.

| Phase | Durée | Python restant | Objectif zéro Python |
|-------|:-----:|:--------------:|----------------------|
| Sprint 0 | 2 jours | Tout | POC DuckDB + MSAL Go |
| Phase 0 | 2-3 sem | Tout | Cadrage, golden values |
| Phase 1 | 4-6 sem | Tout (Go read-only) | Socle Go, 0 Python dans le chemin read-only |
| Phase 2 | 4-6 sem | Tout (Go read-only) | Client HTTP Go 343i dès S11 |
| Phase 3 | 3-4 sem | Tout (Go read-only) | Auth Go native, échange tokens Go |
| Phase 4 | 6-8 sem | Tout (Go read-only) | Sync Go + client Go Halo complet |
| Phase 5 | 2-4 sem | **Zéro** | Validation, nettoyage, suppression `.py` |

### Gate "Zéro Python" (entre Phase 4 et Phase 5)

Ce gate n'existe pas dans le plan maître. Il est ajouté ici :

- [ ] **Aucun** fichier `.py` dans le chemin d'exécution du produit
- [ ] **Aucun** processus Python lancé par le binaire Go
- [ ] **Aucun** `pip install` dans le Dockerfile
- [ ] Le client Go Halo fonctionne pour tous les endpoints 343i nécessaires
- [ ] Les tests de parité client Go vs Python sont verts sur fixtures
- [ ] 3 cycles de sync complets sans Python
- [ ] `src/ai/` explicitement exclu du build et du packaging produit

---

## Risques spécifiques à la cible zéro Python

### Risque ZP-1 — Weapon parser binaire

Le module `weapon_parser.py` (~400 LOC) parse des chunks binaires de films Halo. C'est le module le plus risqué à porter.

**Mitigation** :
- Le portage est planifié en Sprint 22 (tard dans le programme).
- `encoding/binary` de Go est plus naturel que `bitstring` Python pour le parsing binaire.
- Golden values sur 50+ matchs avec kills armes variées.
- Fallback : si le portage échoue, le weapon parsing peut devenir optionnel (les données armes sont un enrichissement, pas un invariant critique).

### Risque ZP-2 — Regression subtile dans l'échange de tokens

La chaîne `MSAL → XBL → XSTS → Spartan → Clearance` a 5 étapes HTTP. Une erreur subtile (header manquant, encoding, timing) peut casser l'auth sans message clair.

**Mitigation** :
- Tests E2E de la chaîne complète dans Sprint 0 (MSAL) et Sprint 15 (full chain).
- Comparer les headers HTTP Python vs Go request par request.
- Circuit breaker : après 3 échecs auth consécutifs, remonter `reauth_required` sans boucle.

### Risque ZP-3 — Formules mathématiques non identiques

Performance Score et LUSR utilisent des flottants. Les différences d'arrondi entre Python et Go (`float64`) peuvent donner des résultats légèrement différents.

**Mitigation** :
- Tolérance documentée : ε < 0.01 pour les scores, ε < 0.1 pour mu/sigma.
- Golden values chiffrées sur 500+ matchs.
- Go utilise `math.Round()` aux mêmes endroits que Python.

### Risque ZP-4 — Pression sur le calendrier solo

Porter **tout** en Go (y compris les algorithmes complexes et le client Halo) demande plus de temps que l'approche hybride. Estimation révisée : **+2-3 mois** par rapport à un hybride qui garderait Python pour sync/analytics.

**Mitigation** :
- Les phases read-only (1-2) sont identiques au plan hybride.
- Le client Go Halo est implémenté dès S11, ce qui donne du temps pour le stabiliser avant Phase 4.
- Le surcoût est concentré sur Phases 3-4 (auth Go + client Go 343i + sync Go).

---

## Règle de non-régression

À chaque sprint, avant de merger :

1. **Aucun** fichier `.py` n'a été ajouté au chemin produit.
2. **Aucune** dépendance Python n'a été ajoutée.
3. Le client Go Halo n'a pas été remplacé par un bridge ou subprocess Python (sauf D6 weapon parser si portage échoue).
4. Le nombre de fichiers Python dans le repo n'a pas augmenté.
5. Si un module Python a été remplacé par Go, le Python est supprimé dans le même lot.

---

## Résumé décisionnel

| Question | Décision |
|----------|----------|
| Python en prod ? | **Non — zéro** |
| SPNKr en prod ? | **Non** — client Go direct dès S11, pas de bridge |
| SPNKr sera remplacé par quoi ? | Socle Go public `pkg/haloapi/` + provider `titles/haloinfinite/` |
| Client API Halo : interne ou public ? | **Public** (`pkg/`) — extractible en module Go indépendant |
| Endpoints 343i : hardcodés ? | **Non** — registre centralisé `ServiceEndpoints` modifiable |
| Polars remplacé par quoi ? | SQL DuckDB natif + structs Go |
| MSAL Python → Go ? | Oui — SDK MSAL Go officiel |
| Weapon parser Python → Go ? | Oui — Sprint 4.5, fallback : fonctionnalité optionnelle |
| `src/ai/` ? | Hors scope (outillage dev, pas produit) |
| Estimation surcoût vs hybride ? | +2-3 mois |
| Date d'extinction Python ? | Gate entre Phase 4 et Phase 5 |
| Publication `haloinfinite-go` ? | Post-Gate 4, optionnel mais encouragé |
